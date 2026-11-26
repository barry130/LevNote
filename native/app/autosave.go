package app

import (
	"sync"
	"time"
)

const autosaveDelay = 900 * time.Millisecond

type autosaveTimer interface {
	Stop() bool
}

// Only the timer slot is shared with Shutdown. No external seam runs under mu.
type autosaveState struct {
	mu      sync.Mutex
	timer   autosaveTimer
	request uint64
	due     time.Time
	armed   bool
}

type autosaveHooks struct {
	now       func() time.Time
	afterFunc func(time.Duration, func()) autosaveTimer
	dispatch  func(func())
}

func (a *App) autosaveNow() time.Time {
	if a.autosaveHooks != nil && a.autosaveHooks.now != nil {
		return a.autosaveHooks.now()
	}
	return time.Now()
}

func (a *App) noteEdit() {
	a.lastEdit = a.autosaveNow()
	a.markPersistPending()
}

func (a *App) markPersistPending() {
	if a.shuttingDown.Load() {
		return
	}
	a.persistPending = true
	a.scheduleAutosave()
}

func (a *App) scheduleAutosave() {
	// Modal completion already resumes recovery here. Also resume a preserved
	// automatic descriptor at its original due; this never invents an edit.
	recovery := a.prepareAutosave()
	desired := a.sourceLanguage.auto.desired
	automatic := a.prepareSourceAuto()
	if recovery != nil {
		recovery()
	}
	if a.sourceLanguage.auto.desired == desired && automatic != nil {
		automatic()
	}
}

// Reserve the generation before either Host timer seam can reenter. Input
// delivery prepares both clocks before running either returned effect.
func (a *App) prepareAutosave() func() {
	if a.shuttingDown.Load() || !a.persistPending {
		return nil
	}
	if a.restoring || a.modalDepth > 0 {
		return a.prepareCancelAutosave()
	}
	var dispatch func(func())
	afterFunc := func(delay time.Duration, fn func()) autosaveTimer { return time.AfterFunc(delay, fn) }
	if hooks := a.autosaveHooks; hooks != nil {
		if hooks.afterFunc == nil || hooks.dispatch == nil {
			return nil
		}
		afterFunc, dispatch = hooks.afterFunc, hooks.dispatch
	} else if window := a.Window; window != nil {
		dispatch = window.Update
	} else {
		return nil
	}
	due := a.lastEdit.Add(autosaveDelay)
	delay := max(time.Duration(0), due.Sub(a.autosaveNow()))
	s := &a.autosave
	s.mu.Lock()
	if a.shuttingDown.Load() || s.armed && s.timer != nil && s.due.Equal(due) {
		s.mu.Unlock()
		return nil
	}
	old := s.timer
	s.timer = nil
	s.request++
	request := s.request
	s.due, s.armed = due, true
	s.mu.Unlock()
	return func() {
		if old != nil {
			old.Stop()
		}
		s.mu.Lock()
		current := s.request == request && s.armed && !a.shuttingDown.Load()
		s.mu.Unlock()
		if !current {
			return
		}
		timer := afterFunc(delay, func() { dispatch(func() { a.deliverAutosave(request) }) })
		s.mu.Lock()
		current = s.request == request && s.armed && !a.shuttingDown.Load()
		if current {
			s.timer = timer
		}
		s.mu.Unlock()
		if !current && timer != nil {
			timer.Stop()
		}
	}
}

func (a *App) deliverAutosave(request uint64) {
	if a.shuttingDown.Load() {
		return
	}
	s := &a.autosave
	s.mu.Lock()
	if s.request != request || !s.armed {
		s.mu.Unlock()
		return
	}
	s.timer, s.armed, s.due = nil, false, time.Time{}
	s.mu.Unlock()
	if a.shuttingDown.Load() || !a.persistPending || a.restoring || a.modalDepth > 0 {
		return
	}
	if a.autosaveNow().Before(a.lastEdit.Add(autosaveDelay)) {
		a.scheduleAutosave()
		return
	}
	_ = a.persistAutomatic()
}

func (a *App) prepareCancelAutosave() func() {
	s := &a.autosave
	s.mu.Lock()
	s.request++
	old := s.timer
	s.timer, s.armed, s.due = nil, false, time.Time{}
	s.mu.Unlock()
	return func() {
		if old != nil {
			old.Stop()
		}
	}
}

func (a *App) cancelAutosave() { a.prepareCancelAutosave()() }
