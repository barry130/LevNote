package app

import (
	"context"
	"errors"
	"io"
	"sync"

	"levnote.local/native/core/document"
	"levnote.local/native/modules"
	"levnote.local/native/ui/editor"
)

type binaryRequest struct {
	op       string
	path     string
	kind     document.BinaryKind
	token    document.BinaryToken
	snapshot *document.BinarySnapshot
	decode   func(context.Context, io.ReaderAt, int64) (editor.ImageResult, error)
}

type binaryOutput struct {
	file       *document.BinaryFile // newly created, transferred once to the UI
	snapshot   *document.BinarySnapshot
	image      editor.ImageResult
	inspection *document.Inspection
	copy       document.BinarySaveResult
	receipt    document.BinaryCopyReceipt
	err        error
}

// No tab, view or live owner crosses into the execution loop. Only this result
// slot is shared; Invalidate carries neither a callback nor image payload.
type binaryTask struct {
	ctx              context.Context
	cancel           context.CancelFunc
	request          binaryRequest
	notify           func()
	done             chan struct{}
	mu               sync.Mutex
	ready, abandoned bool
	output           binaryOutput
	once             sync.Once
}

func newBinaryTask(request binaryRequest, notify func()) *binaryTask {
	ctx, cancel := context.WithCancel(context.Background())
	return &binaryTask{ctx: ctx, cancel: cancel, request: request, notify: notify, done: make(chan struct{})}
}

func (t *binaryTask) finish(output binaryOutput) {
	t.once.Do(func() {
		t.request = binaryRequest{}
		t.cancel()
		t.mu.Lock()
		if !t.abandoned {
			t.output, t.ready = output, true
		}
		t.mu.Unlock()
		close(t.done)
		if t.notify != nil {
			t.notify()
		}
		t.notify = nil
	})
}

func (t *binaryTask) take() (binaryOutput, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.ready {
		return binaryOutput{}, false
	}
	out := t.output
	t.output, t.ready = binaryOutput{}, false
	return out, true
}

func (t *binaryTask) abandon() {
	t.cancel()
	t.mu.Lock()
	t.abandoned, t.ready, t.output = true, false, binaryOutput{}
	t.mu.Unlock()
}

// A single loop owns full reads, codec work and copies. An explicit operation
// reserves it through UI outcome handling; navigation keeps one newest request.
type binaryLane struct {
	mu                        sync.Mutex
	active, pending, explicit *binaryTask
	awaiting                  *binaryTask
	activeExplicit            bool
	reserved, stopped         bool
	wake, done                chan struct{}
	execute                   func(context.Context, binaryRequest) binaryOutput
}

func newBinaryLane(execute func(context.Context, binaryRequest) binaryOutput) *binaryLane {
	if execute == nil {
		execute = executeBinary
	}
	l := &binaryLane{wake: make(chan struct{}, 1), done: make(chan struct{}), execute: execute}
	go l.run()
	return l
}

func (l *binaryLane) signal() {
	select {
	case l.wake <- struct{}{}:
	default:
	}
}

func (l *binaryLane) submit(t *binaryTask, explicit bool) error {
	l.mu.Lock()
	if l.stopped || explicit && l.reserved {
		l.mu.Unlock()
		return errors.New("图片任务正在完成或已关闭，请稍候")
	}
	var dropped *binaryTask
	if explicit {
		l.explicit, l.reserved = t, true
	} else {
		dropped, l.pending = l.pending, t
	}
	if !explicit && l.active != nil && !l.activeExplicit {
		l.active.cancel()
	}
	l.mu.Unlock()
	if dropped != nil {
		dropped.finish(binaryOutput{err: context.Canceled})
	}
	l.signal()
	return nil
}

func (l *binaryLane) release(t *binaryTask, explicit bool) {
	l.mu.Lock()
	if l.awaiting == t {
		l.awaiting = nil
	}
	if explicit {
		l.reserved = false
	}
	l.mu.Unlock()
	l.signal()
}

func (l *binaryLane) cancelTask(t *binaryTask, abandon bool) {
	if t == nil {
		return
	}
	if abandon {
		t.abandon()
	} else {
		t.cancel()
	}
	l.mu.Lock()
	if l.awaiting == t {
		l.awaiting = nil
	}
	dropped := l.pending == t
	if dropped {
		l.pending = nil
	}
	l.mu.Unlock()
	if dropped {
		t.finish(binaryOutput{err: context.Canceled})
	}
	l.signal()
}

func (l *binaryLane) run() {
	defer close(l.done)
	for range l.wake {
		for {
			l.mu.Lock()
			if l.stopped {
				l.mu.Unlock()
				return
			}
			if l.awaiting != nil {
				l.mu.Unlock()
				break
			}
			t := l.explicit
			explicit := t != nil
			if explicit {
				l.explicit = nil
			} else if !l.reserved {
				t, l.pending = l.pending, nil
			}
			if t == nil {
				l.mu.Unlock()
				break
			}
			l.active, l.activeExplicit = t, explicit
			l.mu.Unlock()
			out := binaryOutput{err: t.ctx.Err()}
			if out.err == nil {
				out = l.execute(t.ctx, t.request)
			}
			l.mu.Lock()
			l.active, l.activeExplicit = nil, false
			t.mu.Lock()
			if !t.abandoned {
				l.awaiting = t
			}
			t.mu.Unlock()
			l.mu.Unlock()
			t.finish(out)
		}
	}
}

func (l *binaryLane) close() {
	if l == nil {
		return
	}
	l.mu.Lock()
	l.stopped = true
	pending, explicit := l.pending, l.explicit
	l.pending, l.explicit = nil, nil
	l.awaiting = nil
	if l.active != nil {
		l.active.cancel()
	}
	l.mu.Unlock()
	if pending != nil {
		pending.finish(binaryOutput{err: context.Canceled})
	}
	if explicit != nil {
		explicit.finish(binaryOutput{err: context.Canceled})
	}
	l.signal()
	<-l.done
}

func executeBinary(ctx context.Context, request binaryRequest) binaryOutput {
	out := binaryOutput{snapshot: request.snapshot}
	switch request.op {
	case "open", "restore", "reload":
		out.file, out.snapshot, out.err = document.OpenBinary(ctx, request.path, request.kind)
		if errors.Is(out.err, document.ErrTooLarge) && request.op != "reload" {
			inspection, err := document.Inspect(request.path, 64<<10)
			out.inspection, out.err = &inspection, err
		}
		return out
	case "preview", "bytes", "copy":
		if out.snapshot == nil {
			out.snapshot, out.err = document.ReadBinarySnapshot(ctx, request.token)
			if out.err != nil {
				return out
			}
		}
	default:
		out.err = errors.New("未知图片任务")
		return out
	}
	if request.op == "copy" {
		out.copy, out.receipt, out.err = document.PublishBinaryCopy(ctx, request.path, out.snapshot)
	} else if request.op == "preview" {
		decode := request.decode
		if decode == nil {
			decode = modules.DecodeImage
		}
		out.image, out.err = decode(ctx, out.snapshot.ReaderAt(), out.snapshot.Size())
		if out.err == nil && (request.kind == document.BinaryKindPNG && out.image.Format != "PNG" ||
			request.kind == document.BinaryKindJPEG && out.image.Format != "JPEG") {
			out.image = editor.ImageResult{}
			out.err = errors.New("图片签名与所选文件类型不一致；请查看只读字节")
		}
	}
	return out
}
