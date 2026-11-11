package app

import (
	"context"
	"errors"
	"sync"

	"levnote.local/native/core/engineipc"
	"levnote.local/native/core/engineruntime"
)

// engineTask contains captured immutable input and a protected completion slot;
// the execution loop never reads a tab, buffer or view. Completion must not call
// a UI dispatcher. notify is the production window's nonblocking Invalidate.
type engineTask struct {
	ctx      context.Context
	cancel   context.CancelFunc
	id       engineipc.EngineID
	source   engineruntime.Source
	export   bool
	complete func(engineruntime.Result, error)
	notify   func()
	// Protected by the lane lock, not the view's lifecycle lock.
	runCancel context.CancelFunc
	preempted bool
}

// One loop owns Runner.Export. An export reserves the lane through its UI
// publication phase; there is only one latest pending preview, including a
// preempted request that should resume after publication.
type engineLane struct {
	runner                  exportRunner
	mu                      sync.Mutex
	wake                    chan struct{}
	done                    chan struct{}
	active, pending, export *engineTask
	reserved, stopped       bool
	cleanupErr              error
	closeOnce               sync.Once
	closed                  chan struct{}
}

func newEngineLane(runner exportRunner) *engineLane {
	l := &engineLane{runner: runner, wake: make(chan struct{}, 1), done: make(chan struct{}), closed: make(chan struct{})}
	go l.run()
	return l
}

func (l *engineLane) signal() {
	select {
	case l.wake <- struct{}{}:
	default:
	}
}

func finishEngineTask(task *engineTask, result engineruntime.Result, err error) {
	task.source = engineruntime.Source{}
	task.cancel()
	complete, notify := task.complete, task.notify
	task.complete, task.notify = nil, nil
	complete(result, err)
	if notify != nil {
		notify()
	}
}

func (l *engineLane) unavailableLocked() error {
	if l.cleanupErr != nil {
		return l.cleanupErr
	}
	if l.stopped {
		return errors.New("引擎任务队列已关闭")
	}
	return nil
}

func (l *engineLane) submitPreview(task *engineTask) error {
	l.mu.Lock()
	if err := l.unavailableLocked(); err != nil {
		l.mu.Unlock()
		return err
	}
	old := l.pending
	l.pending = task
	if old != nil {
		old.cancel()
		if old == l.active {
			old = nil
		}
	}
	if l.active != nil && !l.active.export {
		l.active.cancel()
		if l.active.runCancel != nil {
			l.active.runCancel()
		}
	}
	l.mu.Unlock()
	if old != nil {
		finishEngineTask(old, engineruntime.Result{}, context.Canceled)
	}
	l.signal()
	return nil
}

func (l *engineLane) reserveExport(task *engineTask) error {
	l.mu.Lock()
	if err := l.unavailableLocked(); err != nil {
		l.mu.Unlock()
		return err
	}
	if l.reserved {
		l.mu.Unlock()
		return errors.New("已有导出正在完成或清理")
	}
	l.reserved, l.export = true, task
	if active := l.active; active != nil && !active.export {
		active.preempted = true
		if l.pending == nil && active.ctx.Err() == nil {
			l.pending = active
		}
		if active.runCancel != nil {
			active.runCancel()
		}
	}
	l.mu.Unlock()
	l.signal()
	return nil
}

func (l *engineLane) releaseExport() {
	l.mu.Lock()
	l.reserved = false
	l.mu.Unlock()
	l.signal()
}

func (l *engineLane) cancelTask(task *engineTask) {
	if task == nil {
		return
	}
	l.mu.Lock()
	task.cancel()
	if task.runCancel != nil {
		task.runCancel()
	}
	dropped := l.pending == task && l.active != task
	if l.pending == task {
		l.pending = nil
	}
	l.mu.Unlock()
	if dropped {
		finishEngineTask(task, engineruntime.Result{}, context.Canceled)
	}
	l.signal()
}

func cleanupFailure(err error) bool {
	var failure *engineruntime.Failure
	return errors.As(err, &failure) && failure.Kind == engineruntime.FailureCleanup
}

func (l *engineLane) run() {
	defer close(l.done)
	for {
		<-l.wake
		for {
			l.mu.Lock()
			if l.stopped {
				l.mu.Unlock()
				return
			}
			task := l.export
			if task != nil {
				l.export = nil
			} else if !l.reserved {
				task, l.pending = l.pending, nil
			}
			if task == nil {
				l.mu.Unlock()
				break
			}
			l.active = task
			task.preempted = false
			ctx, cancel := context.WithCancel(task.ctx)
			task.runCancel = cancel
			poison := l.cleanupErr
			l.mu.Unlock()
			var result engineruntime.Result
			err := ctx.Err()
			if poison != nil {
				err = poison
			}
			if err == nil {
				result, err = l.runner.Export(ctx, task.id, task.source)
			}
			cancel()
			l.mu.Lock()
			if cleanupFailure(err) && l.cleanupErr == nil {
				l.cleanupErr = err
			}
			resume := task.preempted && l.pending == task && !l.stopped && l.cleanupErr == nil && task.ctx.Err() == nil
			if !resume && l.pending == task {
				l.pending = nil
			}
			task.runCancel = nil
			l.active = nil
			l.mu.Unlock()
			if !resume {
				finishEngineTask(task, result, err)
			}
		}
	}
}

func (l *engineLane) failure() error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.cleanupErr
}

func (l *engineLane) close() error {
	if l == nil {
		return nil
	}
	l.closeOnce.Do(func() {
		l.mu.Lock()
		l.stopped = true
		pending, export := l.pending, l.export
		l.pending, l.export = nil, nil
		if active := l.active; active != nil {
			active.cancel()
			if active.runCancel != nil {
				active.runCancel()
			}
			if pending == active {
				pending = nil
			}
		}
		l.mu.Unlock()
		if pending != nil {
			finishEngineTask(pending, engineruntime.Result{}, context.Canceled)
		}
		if export != nil {
			finishEngineTask(export, engineruntime.Result{}, context.Canceled)
		}
		err := l.runner.Close()
		l.signal()
		<-l.done
		l.mu.Lock()
		if l.cleanupErr == nil {
			l.cleanupErr = err
		}
		l.mu.Unlock()
		close(l.closed)
	})
	<-l.closed
	return l.failure()
}
