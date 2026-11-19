package app

import (
	"sync"
	"time"
)

const autosaveDelay = 900 * time.Millisecond

type autosaveTimer interface {
	Stop() bool
}

// Only the timer slot is shared with Shutdown. Document and UI state remains
// owned by the main thread; the timer goroutine only posts a captured callback.
type autosaveState struct {
	mu      sync.Mutex
	timer   autosaveTimer
	request uint64
	due     time.Time
}

// Tests opt into a fake timer and a queued dispatcher together. uiDispatch is
// deliberately not used: existing headless callers can execute it inline.
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
	if a.shuttingDown.Load() || !a.persistPending {
		return
	}
	if a.restoring || a.modalDepth > 0 {
		a.cancelAutosave()
		return
	}
	var dispatch func(func())
	afterFunc := func(delay time.Duration, fn func()) autosaveTimer {
		return time.AfterFunc(delay, fn)
	}
	if hooks := a.autosaveHooks; hooks != nil {
		if hooks.afterFunc == nil || hooks.dispatch == nil {
			return
		}
		afterFunc, dispatch = hooks.afterFunc, hooks.dispatch
	} else if window := a.Window; window != nil {
		dispatch = window.Update
	} else {
		// No real timer may call App.update's synchronous headless fallback.
		return
	}
	due := a.lastEdit.Add(autosaveDelay)
	delay := max(time.Duration(0), due.Sub(a.autosaveNow()))
	s := &a.autosave
	s.mu.Lock()
	defer s.mu.Unlock()
	if a.shuttingDown.Load() || s.timer != nil && s.due.Equal(due) {
		return
	}
	if s.timer != nil {
		s.timer.Stop()
	}
	s.request++
	request := s.request
	s.due = due
	s.timer = afterFunc(delay, func() {
		// Do not inspect App here, even to choose its dispatcher. Window.Update
		// queues this callback on the main thread independently of painting.
		dispatch(func() { a.deliverAutosave(request) })
	})
}

func (a *App) deliverAutosave(request uint64) {
	if a.shuttingDown.Load() {
		return
	}
	s := &a.autosave
	s.mu.Lock()
	if s.request != request || s.timer == nil {
		s.mu.Unlock()
		return
	}
	s.timer = nil
	s.due = time.Time{}
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

func (a *App) cancelAutosave() {
	s := &a.autosave
	s.mu.Lock()
	defer s.mu.Unlock()
	s.request++
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	s.due = time.Time{}
}
