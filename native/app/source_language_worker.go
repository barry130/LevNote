package app

import (
	"context"
	"fmt"
	"sync"

	"levnote.local/native/formats/code"
)

// These instance-local seams control scheduling and defensive failures only.
// Successful semantics always come from the real AnalyzeService.
type sourceLanguageHooks struct {
	dispatch      func(func())
	beforeAnalyze func(context.Context, uint64)
	afterAnalyze  func(uint64)
	fail          error
	panicValue    string
	malformed     bool
}

type sourceLanguageOutcome struct {
	result code.ServiceResult
	err    error
}

// A task has no live document, editor, buffer, plugin manager or filesystem.
// The UI alone owns its seal. The notifier queues an ID-only UI closure.
type sourceLanguageTask struct {
	mu               sync.Mutex
	id               uint64
	ctx              context.Context
	cancel           context.CancelFunc
	request          code.ServiceRequest
	outcome          sourceLanguageOutcome
	ready, abandoned bool
	done             chan struct{}
}

func newSourceLanguageTask(id uint64, request code.ServiceRequest) *sourceLanguageTask {
	ctx, cancel := context.WithCancel(context.Background())
	return &sourceLanguageTask{id: id, ctx: ctx, cancel: cancel, request: request, done: make(chan struct{})}
}

func (t *sourceLanguageTask) abandon() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.abandoned = true
	if t.cancel != nil {
		t.cancel()
	}
	t.cancel = nil
	t.request = code.ServiceRequest{}
	t.outcome = sourceLanguageOutcome{}
}

func (t *sourceLanguageTask) take() (sourceLanguageOutcome, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.ready {
		return sourceLanguageOutcome{}, false
	}
	out := t.outcome
	t.outcome = sourceLanguageOutcome{}
	t.cancel = nil
	return out, true
}

func (t *sourceLanguageTask) run(notify func(uint64), hooks sourceLanguageHooks) {
	// No UI acknowledgement is part of done. Even a canceled synchronous
	// parser keeps this physical slot until its invocation actually returns.
	defer close(t.done)
	t.mu.Lock()
	request := t.request
	t.request = code.ServiceRequest{}
	abandoned := t.abandoned
	t.mu.Unlock()
	var out sourceLanguageOutcome
	if !abandoned {
		out = analyzeSourceLanguage(t.ctx, t.id, request, hooks)
	}
	request = code.ServiceRequest{}
	t.mu.Lock()
	if !t.abandoned {
		t.outcome = out
	}
	out = sourceLanguageOutcome{}
	t.ready = true
	if t.cancel != nil {
		t.cancel()
	}
	t.cancel = nil
	t.mu.Unlock()
	notify(t.id)
}

func analyzeSourceLanguage(ctx context.Context, id uint64, request code.ServiceRequest, hooks sourceLanguageHooks) (out sourceLanguageOutcome) {
	defer func() {
		if value := recover(); value != nil {
			out = sourceLanguageOutcome{err: fmt.Errorf("language analyzer panic: %v", value)}
		}
		if hooks.afterAnalyze != nil {
			hooks.afterAnalyze(id)
		}
	}()
	if hooks.beforeAnalyze != nil {
		hooks.beforeAnalyze(ctx, id)
	}
	if hooks.panicValue != "" {
		panic(hooks.panicValue)
	}
	if hooks.fail != nil {
		return sourceLanguageOutcome{err: hooks.fail}
	}
	out.result = code.AnalyzeService(ctx, request)
	if hooks.malformed {
		// Defensive negative control, never a fabricated successful payload.
		out.result.Status = "broken-host-payload"
	}
	return out
}
