package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/code"
	"levnote.local/native/ui/editor"
)

// These timers never run themselves. Worker notifications enqueue under mu;
// tests explicitly advance the clock and drain one UI turn at a time.
type autoProbeTimer struct {
	delay   time.Duration
	fn      func()
	stopped bool
	onStop  func()
}

func (t *autoProbeTimer) Stop() bool {
	active := !t.stopped
	t.stopped = true
	if active && t.onStop != nil {
		t.onStop()
	}
	return active
}

type autoProbeClock struct {
	mu                 sync.Mutex
	now                time.Time
	recovery, language []*autoProbeTimer
	queue              []func()
}

func newAutoProbeClock(a *App) *autoProbeClock {
	h := &autoProbeClock{now: time.Date(2026, 11, 21, 10, 0, 0, 0, time.UTC)}
	post := func(fn func()) { h.mu.Lock(); h.queue = append(h.queue, fn); h.mu.Unlock() }
	makeTimer := func(slot *[]*autoProbeTimer) func(time.Duration, func()) autosaveTimer {
		return func(d time.Duration, fn func()) autosaveTimer {
			t := &autoProbeTimer{delay: d, fn: fn}
			h.mu.Lock()
			*slot = append(*slot, t)
			h.mu.Unlock()
			return t
		}
	}
	a.autosaveHooks = &autosaveHooks{now: func() time.Time { return h.now }, afterFunc: makeTimer(&h.recovery), dispatch: post}
	a.sourceAutoHooks = &sourceAutoHooks{queued: true, now: func() time.Time { return h.now }, afterFunc: makeTimer(&h.language), dispatch: post}
	a.sourceLanguageHooks = &sourceLanguageHooks{dispatch: post}
	return h
}
func (h *autoProbeClock) takeQueue() []func() {
	h.mu.Lock()
	q := h.queue
	h.queue = nil
	h.mu.Unlock()
	return q
}
func (h *autoProbeClock) drain() {
	for _, fn := range h.takeQueue() {
		fn()
	}
}
func (h *autoProbeClock) latest(slots []*autoProbeTimer) *autoProbeTimer {
	for i := len(slots) - 1; i >= 0; i-- {
		if !slots[i].stopped {
			return slots[i]
		}
	}
	return nil
}
func (h *autoProbeClock) fireLanguage(t *testing.T) {
	t.Helper()
	timer := h.latest(h.language)
	if timer == nil {
		t.Fatal("no automatic timer")
	}
	timer.stopped = true
	timer.fn()
}
func autoProbeFixture(t *testing.T, suffix, source string) (*App, *Tab, *autoProbeClock) {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "probe"+suffix)
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	if tab == nil || tab.Doc == nil || tab.Editor == nil {
		t.Fatalf("open failed: %s", a.Message)
	}
	return a, tab, newAutoProbeClock(a)
}
func autoProbeReplace(t *testing.T, tab *Tab, source string) {
	t.Helper()
	tab.Editor.ReplaceText(&ui.TextInputRange{Start: 0, End: tab.Doc.Buffer.LenUTF16()}, source)
	if tab.Doc.Buffer.String() != source {
		t.Fatal("Editor replacement was not accepted")
	}
}
func autoProbeFinish(t *testing.T, a *App, h *autoProbeClock) {
	t.Helper()
	task := a.sourceLanguage.task
	if task == nil {
		t.Fatal("no real physical analysis task")
	}
	languageWait(t, task.done)
	h.drain()
	if a.sourceLanguage.task != nil {
		t.Fatal("finished physical slot was not retired")
	}
}
func autoProbeStart(t *testing.T, a *App, h *autoProbeClock) {
	t.Helper()
	d := a.sourceLanguage.auto.desired
	if d == nil {
		t.Fatal("no scalar automatic descriptor")
	}
	h.now = d.due
	h.fireLanguage(t)
	if a.sourceLanguage.task != nil {
		t.Fatal("timer started analysis without queued UI drain")
	}
	h.drain()
	if a.sourceLanguage.task == nil || a.sourceLanguage.intent.operation != code.ServiceSymbols {
		t.Fatal("queued due did not start the real Symbols service")
	}
}

func TestLanguageAutoConfirmedQuietTimeNoFrame(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		t.Run(map[bool]string{false: "symbols", true: "diagnostics"}[invalid], func(t *testing.T) {
			a, tab, h := autoProbeFixture(t, ".go", "package p\nfunc Before() {}\n")
			started := make(chan uint64, 1)
			a.sourceLanguageHooks.beforeAnalyze = func(_ context.Context, id uint64) { started <- id }
			source := "package p\nfunc After() {}\n"
			if invalid {
				source = "package p\nfunc After( {\n"
			}
			before := h.now
			autoProbeReplace(t, tab, source)
			if len(h.language) != 1 || len(h.recovery) != 1 || h.language[0].delay != 500*time.Millisecond || h.recovery[0].delay != 900*time.Millisecond {
				t.Fatal("independent deadlines not armed exactly once")
			}
			if !a.lastEdit.Equal(before) {
				t.Fatal("confirmed recovery timestamp")
			}
			h.now = before.Add(499 * time.Millisecond)
			h.fireLanguage(t)
			h.drain()
			if len(started) != 0 || a.sourceLanguage.task != nil {
				t.Fatal("analyzer started early")
			}
			if timer := h.latest(h.language); timer == nil || timer.delay != time.Millisecond {
				t.Fatal("early delivery lost the original due")
			}
			h.now = before.Add(500 * time.Millisecond)
			h.fireLanguage(t)
			h.drain()
			id := languageReceive(t, started)
			if a.sourceLanguage.task == nil || a.sourceLanguage.task.id != id {
				t.Fatal("physical service identity")
			}
			autoProbeFinish(t, a, h)
			r := a.sourceLanguage.result
			if invalid {
				if r.Status != code.ServiceInvalid || len(r.Diagnostics) == 0 {
					t.Fatalf("real diagnostic result missing: %+v", r)
				}
			} else {
				found := false
				for _, symbol := range r.Symbols {
					if symbol.Name == "After" {
						found = true
					}
				}
				if r.Status != code.ServiceComplete || !found {
					t.Fatalf("real symbol result missing: %+v", r)
				}
			}
			if len(h.recovery) != 1 || !a.lastEdit.Equal(before) {
				t.Fatal("language result changed recovery clock")
			}
		})
	}
}

func TestLanguageAutoRecoveryPlainUnsupportedAndInactive(t *testing.T) {
	for _, inactive := range []bool{false, true} {
		t.Run(map[bool]string{false: "unsupported", true: "inactive"}[inactive], func(t *testing.T) {
			a, tab, h := autoProbeFixture(t, ".txt", "plain draft")
			if inactive {
				a.NewDocument("text")
				if a.Current() == tab {
					t.Fatal("inactive fixture")
				}
			}
			// A rich-view attachment must not gate legacy source recovery.
			tab.View = &failingView{}
			h.now = h.now.Add(time.Second)
			autoProbeReplace(t, tab, "plain committed edit")
			if len(h.language) != 0 || a.sourceLanguage.auto.desired != nil || !a.lastEdit.Equal(h.now) || !a.persistPending {
				t.Fatal("recovery was coupled to active language eligibility")
			}
			last := a.lastEdit
			h.now = h.now.Add(20 * time.Millisecond)
			tab.Editor.SetMarkedText(&ui.TextInputRange{Start: 0, End: 1}, "预", ui.TextInputRange{Start: 1, End: 1})
			if !a.lastEdit.After(last) || !tab.Doc.Buffer.InGroup() {
				t.Fatal("legacy provisional recovery was lost")
			}
			h.now = a.lastEdit.Add(autosaveDelay)
			timer := h.latest(h.recovery)
			if timer == nil {
				t.Fatal("no recovery timer for noneligible document")
			}
			timer.fn()
			h.drain()
			snapshot := readRecovery(t, a)
			found := false
			for _, record := range snapshot.Tabs {
				if record.Draft != nil && *record.Draft == tab.Doc.Buffer.String() {
					found = true
				}
			}
			if !found || !tab.Doc.Buffer.InGroup() {
				t.Fatal("recovery did not persist provisional bytes without settling input")
			}
		})
	}
}

func TestLanguageAutoRecoveryDeadlineIndependent(t *testing.T) {
	a, tab, h := autoProbeFixture(t, ".go", "package p\nfunc Same() {}\n")
	last, pending := a.lastEdit, a.persistPending
	autoProbeReplace(t, tab, tab.Doc.Buffer.String())
	tab.Editor.ReplaceText(&ui.TextInputRange{Start: 0, End: 0}, "")
	if len(h.recovery) != 0 || len(h.language) != 0 || a.persistPending != pending || !a.lastEdit.Equal(last) {
		t.Fatal("equal/empty generic callbacks invented an edit")
	}
	autoProbeReplace(t, tab, "package p\nfunc First() {}\n")
	due, recoveryDue := a.sourceLanguage.auto.desired.due, a.autosave.due
	h.now = h.now.Add(100 * time.Millisecond)
	tab.Editor.SetMarkedText(&ui.TextInputRange{Start: 0, End: 1}, "p", ui.TextInputRange{Start: 1, End: 1})
	last, recoveryDue = a.lastEdit, a.autosave.due
	if !recoveryDue.Equal(last.Add(900 * time.Millisecond)) {
		t.Fatal("preedit did not retain legacy recovery semantics")
	}
	if err := tab.Editor.Flush(); err != nil {
		t.Fatal(err)
	}
	if !a.lastEdit.Equal(last) || !a.autosave.due.Equal(recoveryDue) || a.sourceLanguage.auto.desired == nil || !a.sourceLanguage.auto.desired.due.Equal(due) {
		t.Fatal("unchanged terminal shifted an independent clock")
	}
	autoProbeReplace(t, tab, tab.Doc.Buffer.String())
	if !a.lastEdit.Equal(last) || !a.autosave.due.Equal(recoveryDue) {
		t.Fatal("same bytes reset recovery")
	}
}

func TestLanguageAutoTerminalFlushAndDueNetZeroReacquiresTarget(t *testing.T) {
	for _, finish := range []string{"flush", "unmark", "cancel"} {
		t.Run(finish, func(t *testing.T) {
			a, tab, h := autoProbeFixture(t, ".go", "package p\nfunc Before() {}\n")
			autoProbeReplace(t, tab, "package p\nfunc After() {}\n")
			d := a.sourceLanguage.auto.desired
			old, err := tab.Editor.CaptureSourceSnapshot(code.DefaultServiceLimits().Structure.Bytes)
			if err != nil {
				t.Fatal(err)
			}
			h.now = h.now.Add(100 * time.Millisecond)
			tab.Editor.SetMarkedText(&ui.TextInputRange{Start: 0, End: 1}, "x", ui.TextInputRange{Start: 1, End: 1})
			if finish != "cancel" {
				tab.Editor.SetMarkedText(&ui.TextInputRange{Start: 0, End: 1}, "p", ui.TextInputRange{Start: 1, End: 1})
			}
			h.now = d.due
			// Deliver a Stop-racing old timer while input is still blocked.
			h.language[0].fn()
			h.drain()
			if a.sourceLanguage.task != nil || a.sourceLanguage.auto.desired != d || !a.sourceLanguage.auto.blocked {
				t.Fatal("blocked due captured or replaced its descriptor")
			}
			last, recoveryCount, timers := a.lastEdit, len(h.recovery), len(h.language)
			switch finish {
			case "flush":
				if err := tab.Editor.Flush(); err != nil {
					t.Fatal(err)
				}
			case "unmark":
				tab.Editor.UnmarkText()
			case "cancel":
				if !tab.Editor.HandleInput(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEscape}) {
					t.Fatal("own cancel refused")
				}
			}
			if a.sourceLanguage.task != nil || a.sourceLanguage.auto.desired != d || !a.sourceLanguage.auto.queued || !d.due.Equal(h.now) {
				t.Fatal("terminal did not queue original due descriptor")
			}
			if !a.lastEdit.Equal(last) || len(h.language) != timers {
				t.Fatal("readiness reset recovery or installed a polling timer")
			}
			if finish != "cancel" && len(h.recovery) != recoveryCount {
				t.Fatal("net-zero terminal rearmed recovery")
			}
			if tab.Editor.CheckSourceTarget(old.Target, false) == nil {
				t.Fatal("old coordinate target survived rawVersion changes")
			}
			h.drain()
			if a.sourceLanguage.task == nil || a.sourceLanguage.owner.version == old.Version || a.sourceLanguage.intent.automatic != d {
				t.Fatal("resumed analysis did not recapture current target")
			}
			autoProbeFinish(t, a, h)
			if a.sourceLanguage.result.Status != code.ServiceComplete {
				t.Fatal("real resumed Symbols failed")
			}
		})
	}
}

func TestLanguageAutoMarklessRefusalResumeAndManualCancel(t *testing.T) {
	a, tab, h := autoProbeFixture(t, ".go", "package p\n")
	b := text.NewWithLimit("package p\n", len("package p\n"))
	tab.Doc.Buffer = b
	tab.Editor.SetBuffer(b)
	a.bindSourceInput(tab)
	autoProbeReplace(t, tab, "package q\n")
	d := a.sourceLanguage.auto.desired
	if d == nil {
		t.Fatal("confirmed descriptor fixture")
	}
	tab.Editor.SetMarkedText(&ui.TextInputRange{Start: 0, End: 0}, "🙂", ui.TextInputRange{})
	if !errors.Is(tab.Editor.InputActionError(), text.ErrSizeLimit) || b.InGroup() {
		t.Fatal("not first-refused markless input")
	}
	h.now = d.due
	h.language[0].fn()
	h.drain()
	last := a.lastEdit
	if !tab.Editor.HandleInput(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEscape}) || !a.lastEdit.Equal(last) || a.sourceLanguage.auto.desired != d || !a.sourceLanguage.auto.queued {
		t.Fatal("markless own cancel did not resume without a recovery edit")
	}
	a.cancelSourceLanguage()
	h.drain()
	if a.sourceLanguage.task != nil || a.sourceLanguage.auto.desired != nil || !a.sourceLanguage.auto.manual {
		t.Fatal("queued readiness escaped manual cancel")
	}
	tab.Editor.UnmarkText()
	if err := tab.Editor.Flush(); err != nil {
		t.Fatal(err)
	}
	autoProbeReplace(t, tab, b.String())
	if !a.sourceLanguage.auto.manual || a.sourceLanguage.auto.desired != nil || !a.lastEdit.Equal(last) {
		t.Fatal("readiness/same bytes cleared manual latch")
	}
	autoProbeReplace(t, tab, "package r\n")
	if a.sourceLanguage.auto.manual || a.sourceLanguage.auto.desired == nil || !a.sourceLanguage.auto.desired.due.Equal(h.now.Add(500*time.Millisecond)) {
		t.Fatal("fresh edit failed to rearm canceled generation")
	}
}

func TestLanguageAutoBaselineInvalidatesBeforeDuplicateDelivery(t *testing.T) {
	a, tab, h := autoProbeFixture(t, ".go", "package p\n")
	autoProbeReplace(t, tab, "package q\n")
	b, binding := tab.Doc.Buffer, tab.inputBinding
	delivery, ok := tab.Editor.LastInputDelivery()
	if !ok {
		t.Fatal("real Editor did not publish its delivery")
	}
	stamp, last := b.ChangeStamp(), a.lastEdit
	if err := b.BeginGroup(); err != nil {
		t.Fatal(err)
	}
	if err := b.ReplaceUTF16(0, 1, "x"); err != nil {
		t.Fatal(err)
	}
	b.ResetHistory()
	if b.ChangeStamp().BaselineID == stamp.BaselineID || b.ChangeStamp().ConfirmedChangeID != stamp.ConfirmedChangeID {
		t.Fatal("baseline-only invalidation fixture")
	}
	if !a.consumeSourceInput(binding, delivery) || a.sourceLanguage.auto.desired != nil || !a.lastEdit.Equal(last) || len(h.language) != 1 {
		t.Fatal("duplicate record overrode invalidated baseline or invented edit")
	}
	autoProbeReplace(t, tab, "package r\n")
	if a.sourceLanguage.auto.desired == nil {
		t.Fatal("real commit on primed new baseline did not arm")
	}
}

func TestLanguageAutoUnprovedRetirementNeverResumes(t *testing.T) {
	a, tab, h := autoProbeFixture(t, ".go", "package p\n")
	autoProbeReplace(t, tab, "package q\n")
	tab.Editor.SetMarkedText(&ui.TextInputRange{Start: 0, End: 1}, "x", ui.TextInputRange{Start: 1, End: 1})
	tab.Doc.Buffer.ResetHistory()
	last, before := a.lastEdit, len(h.language)
	a.Message = ""
	reports := 0
	var reported error
	originalOnError := tab.Editor.OnError
	tab.Editor.OnError = func(err error) {
		reports++
		reported = err
		if originalOnError != nil {
			originalOnError(err)
		}
	}
	tab.Editor.HandleInput(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEscape})
	if a.sourceLanguage.auto.desired != nil || !a.lastEdit.Equal(last) || len(h.language) != before || reports != 1 || !errors.Is(reported, text.ErrComposition) || a.Message != reported.Error() {
		t.Fatal("unproved retirement resumed the old due, missed its actual error, or reported a stale message")
	}
	for _, timer := range h.language {
		timer.fn()
	}
	h.drain()
	if a.sourceLanguage.task != nil || tab.Doc.Buffer.String() != "xackage q\n" {
		t.Fatal("reset-retained provisional bytes were analyzed as proven readiness")
	}
}

func TestLanguageAutoDuePendingKeepsOriginalOwner(t *testing.T) {
	for _, mutation := range []string{"buffer-ABA", "readonly-ABA", "binding-ABA", "view-release", "document", "activation", "route", "restore"} {
		t.Run(mutation, func(t *testing.T) {
			a, tab, h := autoProbeFixture(t, ".go", "package p\n")
			autoProbeReplace(t, tab, "package q\n")
			oldBinding, old := tab.inputBinding, h.language[0]
			last := a.lastEdit
			switch mutation {
			case "buffer-ABA":
				b := tab.Doc.Buffer
				tab.Editor.SetBuffer(text.New(b.String()))
				tab.Editor.SetBuffer(b)
			case "readonly-ABA":
				tab.Editor.SetReadOnly(true)
				tab.Editor.SetReadOnly(false)
			case "binding-ABA":
				a.revokeSourceInput(tab)
				a.bindSourceInput(tab)
			case "view-release":
				tab.Editor.ReleaseView()
			case "document":
				d := document.New(tab.Doc.Buffer.String())
				d.Buffer = tab.Doc.Buffer
				tab.Doc = d
			case "activation":
				a.retireSourceLanguage(nil)
			case "route":
				a.configureTab(tab)
			case "restore":
				tab.Doc.Buffer.Restore(text.New(tab.Doc.Buffer.String()).Snapshot())
			}
			exit := languageState(tab)
			message := a.Message
			h.now = h.now.Add(time.Second)
			old.fn()
			h.drain()
			// A retained old wrapper is never rebound onto a successor owner.
			if d, ok := oldBinding.input.LastInputDelivery(); ok {
				a.consumeSourceInput(oldBinding, d)
			}
			if a.sourceLanguage.task != nil || a.sourceLanguage.auto.desired != nil || !a.lastEdit.Equal(last) || a.Message != message || !reflect.DeepEqual(languageState(tab), exit) {
				t.Fatal("old due crossed owner/baseline authority or changed callback-exit state")
			}
		})
	}
}

func TestLanguageAutoExplicitPresentationDropsWithoutReplay(t *testing.T) {
	for _, order := range []string{"timer-before-open", "open-before-timer", "in-flight"} {
		t.Run(order, func(t *testing.T) {
			a, tab, h := autoProbeFixture(t, ".go", "package p\nfunc Before() {}\n")
			autoProbeReplace(t, tab, "package p\nfunc First() {}\n")
			old := h.language[0]
			var automatic *sourceLanguageTask
			var release func()
			if order == "timer-before-open" {
				h.now = a.sourceLanguage.auto.desired.due
				old.fn()
			}
			if order == "in-flight" {
				started, finish := languageHold(t, a)
				release = finish
				autoProbeStart(t, a, h)
				automatic = a.sourceLanguage.task
				languageReceive(t, started)
			}
			a.runSourceLanguageCommand("symbols")
			if !a.sourceLanguage.visible || a.sourceLanguage.auto.desired != nil || !a.sourceLanguage.auto.explicit {
				t.Fatal("explicit presentation did not drop auto immediately")
			}
			if order == "open-before-timer" {
				old.fn()
			}
			h.drain()
			autoProbeReplace(t, tab, "package p\nfunc DuringExplicit() {}\n")
			if a.sourceLanguage.auto.desired != nil {
				t.Fatal("edit under explicit presentation created hidden auto intent")
			}
			if automatic != nil {
				release()
				languageWait(t, automatic.done)
				h.drain()
			}
			if task := a.sourceLanguage.task; task != nil {
				languageWait(t, task.done)
				h.drain()
			}
			a.closeSourceLanguagePresentation()
			count := len(h.language)
			tab.Editor.UnmarkText()
			autoProbeReplace(t, tab, tab.Doc.Buffer.String())
			h.drain()
			if a.sourceLanguage.auto.desired != nil || a.sourceLanguage.task != nil || len(h.language) != count {
				t.Fatal("explicit close replayed suppressed work")
			}
			autoProbeReplace(t, tab, "package p\nfunc AfterClose() {}\n")
			if a.sourceLanguage.auto.desired == nil {
				t.Fatal("fresh post-close edit did not arm")
			}
		})
	}
}

func TestLanguageAutoSinglePhysicalSlotAndLatestDescriptor(t *testing.T) {
	a, tab, h := autoProbeFixture(t, ".go", "package p\nfunc Before() {}\n")
	started, release := languageHold(t, a)
	var active, maximum, calls atomic.Int32
	hold := a.sourceLanguageHooks.beforeAnalyze
	a.sourceLanguageHooks.beforeAnalyze = func(ctx context.Context, id uint64) {
		n := active.Add(1)
		languageMax(&maximum, n)
		calls.Add(1)
		hold(ctx, id)
	}
	a.sourceLanguageHooks.afterAnalyze = func(uint64) { active.Add(-1) }
	autoProbeReplace(t, tab, "package p\nfunc First() {}\n")
	autoProbeStart(t, a, h)
	first := a.sourceLanguage.task
	languageReceive(t, started)
	for _, name := range []string{"Second", "Third", "Latest"} {
		h.now = h.now.Add(time.Second)
		autoProbeReplace(t, tab, "package p\nfunc "+name+"() {}\n")
		h.now = a.sourceLanguage.auto.desired.due
		h.fireLanguage(t)
		h.drain()
		if a.sourceLanguage.task != first || a.sourceLanguage.pending != nil || a.sourceLanguage.owner != nil {
			t.Fatal("occupied analyzer retained a second target/source")
		}
	}
	latest := a.sourceLanguage.auto.desired
	if latest == nil || calls.Load() != 1 || maximum.Load() != 1 {
		t.Fatal("bursts crossed the sole physical slot")
	}
	// Accepted preedit and proven net-zero may only keep the scalar descriptor.
	tab.Editor.SetMarkedText(&ui.TextInputRange{Start: 0, End: 1}, "x", ui.TextInputRange{Start: 1, End: 1})
	tab.Editor.SetMarkedText(&ui.TextInputRange{Start: 0, End: 1}, "p", ui.TextInputRange{Start: 1, End: 1})
	if err := tab.Editor.Flush(); err != nil {
		t.Fatal(err)
	}
	h.drain()
	if a.sourceLanguage.task != first || a.sourceLanguage.auto.desired != latest {
		t.Fatal("readiness captured while occupied")
	}
	release()
	languageWait(t, first.done)
	h.drain()
	second := a.sourceLanguage.task
	if second == nil || second == first || a.sourceLanguage.intent.automatic != latest {
		t.Fatal("physical retirement did not admit latest eligible descriptor")
	}
	languageReceive(t, started)
	a.completeSourceLanguage(first.id)
	if a.sourceLanguage.task != second {
		t.Fatal("duplicate old notice released successor slot")
	}
	autoProbeFinish(t, a, h)
	if calls.Load() != 2 || maximum.Load() != 1 || active.Load() != 0 {
		t.Fatal("actual analyzer invocation bound violated")
	}
	found := false
	for _, symbol := range a.sourceLanguage.result.Symbols {
		if symbol.Name == "Latest" {
			found = true
		}
	}
	if !found {
		t.Fatal("latest real Symbols result missing")
	}
}

func TestLanguageAutoNoInlineFallbackOrPolling(t *testing.T) {
	for _, mode := range []string{"headless-inline", "untrusted-hook", "lying-hook"} {
		t.Run(mode, func(t *testing.T) {
			a, tab, h := autoProbeFixture(t, ".go", "package p\n")
			a.uiDispatch = func(fn func()) { fn() }
			switch mode {
			case "headless-inline":
				a.sourceAutoHooks = nil
			case "untrusted-hook":
				a.sourceAutoHooks.queued = false
			case "lying-hook":
				a.sourceAutoHooks.dispatch = func(fn func()) { fn() }
			}
			autoProbeReplace(t, tab, "package q\n")
			if mode != "lying-hook" && len(h.language) != 0 {
				t.Fatal("inline-only host armed a real timer")
			}
			for _, timer := range h.language {
				h.now = h.now.Add(time.Second)
				timer.fn()
			}
			h.drain()
			if a.sourceLanguage.task != nil {
				t.Fatal("inline seam entered UI analysis")
			}
		})
	}
}

func TestLanguageAutoBothTimerGenerationsCommitBeforeReentry(t *testing.T) {
	for _, seam := range []string{"recovery-after", "language-after", "recovery-stop", "language-stop"} {
		t.Run(seam, func(t *testing.T) {
			a, tab, h := autoProbeFixture(t, ".go", "package p\n")
			if seam == "recovery-stop" || seam == "language-stop" {
				autoProbeReplace(t, tab, "package q\n")
			}
			entered := false
			var exit languageDocumentState
			var desired *sourceAutoDescriptor
			var last time.Time
			reenter := func() {
				if entered {
					return
				}
				entered = true
				if a.autosave.request == 0 || a.sourceLanguage.auto.timer.request == 0 {
					t.Fatal("one timer generation was uncommitted at the first seam")
				}
				h.now = h.now.Add(17 * time.Millisecond)
				autoProbeReplace(t, tab, "package successor\n")
				a.Message = "successor callback state"
				exit, desired, last = languageState(tab), a.sourceLanguage.auto.desired, a.lastEdit
			}
			switch seam {
			case "recovery-after":
				next := a.autosaveHooks.afterFunc
				a.autosaveHooks.afterFunc = func(d time.Duration, fn func()) autosaveTimer { reenter(); return next(d, fn) }
			case "language-after":
				next := a.sourceAutoHooks.afterFunc
				a.sourceAutoHooks.afterFunc = func(d time.Duration, fn func()) autosaveTimer { reenter(); return next(d, fn) }
			case "recovery-stop":
				h.latest(h.recovery).onStop = reenter
			case "language-stop":
				h.latest(h.language).onStop = reenter
			}
			h.now = h.now.Add(time.Millisecond)
			tab.Editor.ReplaceText(&ui.TextInputRange{Start: 0, End: tab.Doc.Buffer.LenUTF16()}, "package outer\n")
			if !entered || !reflect.DeepEqual(languageState(tab), exit) || a.sourceLanguage.auto.desired != desired || !a.lastEdit.Equal(last) || a.Message != "successor callback state" {
				t.Fatal("old scheduling return overwrote callback-exit successor")
			}
			if desired == nil || !desired.due.Equal(last.Add(sourceAutoDelay)) || a.sourceLanguage.auto.timer.timer != h.latest(h.language) {
				t.Fatal("old returned timer adopted over successor")
			}
		})
	}
}

type autoProbeView struct {
	failingView
	onFlush   func()
	onRelease func()
}

func (v *autoProbeView) Flush() error {
	v.flushes++
	if v.onFlush != nil {
		v.onFlush()
	}
	return v.flushErr
}
func (v *autoProbeView) Release() {
	v.releases++
	if v.onRelease != nil {
		v.onRelease()
	}
}

func TestLanguageAutoFlushViewReplacementStopsOldTail(t *testing.T) {
	for _, mutation := range []string{"view", "nil-view", "buffer-ABA", "readonly-ABA", "binding-ABA", "editor", "view-flush"} {
		t.Run(mutation, func(t *testing.T) {
			a, tab, _ := autoProbeFixture(t, ".go", "package p\n")
			old, replacement := &autoProbeView{}, &autoProbeView{}
			tab.View = old
			if mutation == "nil-view" {
				tab.View = nil
			}
			a.bindSourceInput(tab)
			tab.Editor.SetMarkedText(&ui.TextInputRange{Start: 0, End: 1}, "x", ui.TextInputRange{Start: 1, End: 1})
			var exit languageDocumentState
			var successor *sourceAutoDescriptor
			entered := false
			replace := func() {
				if entered {
					return
				}
				entered = true
				switch mutation {
				case "buffer-ABA":
					b := tab.Doc.Buffer
					tab.Editor.SetBuffer(text.New(b.String()))
					tab.Editor.SetBuffer(b)
				case "readonly-ABA":
					tab.Editor.SetReadOnly(true)
					tab.Editor.SetReadOnly(false)
				case "binding-ABA":
					a.revokeSourceInput(tab)
				case "editor":
					tab.Editor = editor.New(tab.Doc.Buffer)
				default:
					tab.View = replacement
				}
				a.bindSourceInput(tab)
				autoProbeReplace(t, tab, "package successor\n")
				a.Message = "replacement state"
				exit, successor = languageState(tab), a.sourceLanguage.auto.desired
			}
			if mutation == "view-flush" {
				old.onFlush = replace
			} else {
				next := a.sourceAutoHooks.afterFunc
				a.sourceAutoHooks.afterFunc = func(d time.Duration, fn func()) autosaveTimer { replace(); return next(d, fn) }
			}
			if a.flushTab(tab, "test") || !entered {
				t.Fatal("stale Flush reported success or missed terminal reentry")
			}
			wantFlushes := 0
			if mutation == "view-flush" {
				wantFlushes = 1
			}
			if old.flushes != wantFlushes || replacement.flushes != 0 || old.releases != 0 || replacement.releases != 0 {
				t.Fatal("stale continuation flushed/released a substituted View")
			}
			if !reflect.DeepEqual(languageState(tab), exit) || a.sourceLanguage.auto.desired != successor || a.Message != "replacement state" {
				t.Fatal("old Flush changed callback-exit Host or Editor state")
			}
		})
	}
}

func TestLanguageAutoErrorSettlementAndNestedB3(t *testing.T) {
	a, tab, h := autoProbeFixture(t, ".go", "package p\n")
	tab.Editor.SetMarkedText(&ui.TextInputRange{Start: 0, End: 1}, "x", ui.TextInputRange{Start: 0, End: 1})
	marked, composing := tab.Editor.MarkedRange()
	selection := tab.Editor.Selection()
	if !tab.Editor.CompositionActive() || !composing || marked != (ui.TextInputRange{Start: 0, End: 1}) || selection.Range != (ui.TextInputRange{Start: 0, End: 1}) {
		t.Fatal("copy fixture did not retain a valid composition and non-empty selection")
	}
	h.now = h.now.Add(time.Second)
	failure := errors.New("actual clipboard error")
	oldLease, oldEditor, oldBuffer := tab.inputBinding.lease, tab.Editor, tab.Doc.Buffer
	copyCalls, errorReports := 0, 0
	var copiedText string
	originalOnError := tab.Editor.OnError
	if originalOnError == nil {
		t.Fatal("original error callback absent")
	}
	tab.Editor.Copy = func(value string) error { copyCalls++; copiedText = value; return failure }
	tab.Editor.OnError = func(err error) { errorReports++; originalOnError(err) }
	if !tab.Editor.Command("copy") {
		t.Fatal("copy action fixture")
	}
	if copyCalls != 1 || copiedText != "x" || errorReports != 1 {
		t.Fatal("copy error fixture did not reach the real clipboard error callback")
	}
	if a.Message != failure.Error() || a.sourceLanguage.auto.desired == nil || !a.lastEdit.Equal(h.now) {
		t.Fatal("error boundary lost a successful settlement or did not report last")
	}
	before := len(h.language)
	last := a.lastEdit
	delivery, ok := tab.Editor.LastInputDelivery()
	if !ok {
		t.Fatal("error settlement record absent")
	}
	if delivery.Binding != oldLease || delivery.Editor != oldEditor || delivery.Buffer != oldBuffer ||
		delivery.Stamp != oldBuffer.ChangeStamp() || delivery.Reason != editor.InputSettled ||
		delivery.Disposition != editor.InputRealCommit || !delivery.Failed {
		t.Fatal("last delivery is not the real error settlement")
	}
	a.consumeSourceInput(tab.inputBinding, delivery)
	if len(h.language) != before || !a.lastEdit.Equal(last) {
		t.Fatal("error settlement consumed twice")
	}
	oldDesired := a.sourceLanguage.auto.desired
	if oldDesired == nil {
		t.Fatal("old source auto descriptor absent")
	}
	oldTimer := &a.sourceLanguage.auto.timer
	oldTimer.mu.Lock()
	oldTimerDue := oldTimer.due
	oldTimerArmed, oldTimerHandle := oldTimer.armed, oldTimer.timer
	oldTimer.mu.Unlock()
	if !oldTimerArmed || oldTimerHandle == nil || !oldDesired.due.Equal(oldTimerDue) {
		t.Fatal("old source auto timer was not installed")
	}
	// A real B3 edit reentering the timer seam must own the final descriptor.
	entered := false
	exitCaptured := false
	var exit languageDocumentState
	var desired *sourceAutoDescriptor
	next := a.sourceAutoHooks.afterFunc
	a.sourceAutoHooks.afterFunc = func(d time.Duration, fn func()) autosaveTimer {
		if !entered {
			entered = true
			snapshot, err := tab.Editor.CaptureSourceSnapshot(code.DefaultServiceLimits().Structure.Bytes)
			if err != nil {
				t.Fatal(err)
			}
			if changed, err := tab.Editor.ApplySourceEdits(snapshot.Target, []text.PatchEdit{{Start: 0, End: 1, OldText: "p", NewText: "x"}}, editor.SingleInsertionCaret); err != nil || !changed {
				t.Fatalf("nested B3: %v/%v", changed, err)
			}
			exit, desired = languageState(tab), a.sourceLanguage.auto.desired
			if desired == nil || desired.stamp != tab.Doc.Buffer.ChangeStamp() {
				t.Fatal("nested B3 did not commit the live source auto descriptor")
			}
			exitCaptured = true
		}
		return next(d, fn)
	}
	h.now = h.now.Add(time.Millisecond)
	if !h.now.Before(oldTimerDue) {
		t.Fatal("clock advance crossed the old source auto due")
	}
	tab.Editor.ReplaceText(&ui.TextInputRange{Start: 0, End: tab.Doc.Buffer.LenUTF16()}, "package q\n")
	if !entered || !exitCaptured {
		t.Fatal("outer OnChange did not capture the nested B3 exit")
	}
	if !reflect.DeepEqual(languageState(tab), exit) || a.sourceLanguage.auto.desired != desired {
		t.Fatal("outer OnChange left a tail after nested B3")
	}
}

func TestLanguageAutoOrdinarySaveAndRefusedClosePreserveDue(t *testing.T) {
	a, tab, h := autoProbeFixture(t, ".go", "package p\n")
	autoProbeReplace(t, tab, "package q\n")
	d, binding := a.sourceLanguage.auto.desired, tab.inputBinding
	if !a.Save(tab, false) || a.sourceLanguage.auto.desired != d || tab.inputBinding != binding {
		t.Fatal("ordinary unchanged Save retired the read owner")
	}
	autoProbeReplace(t, tab, "package r\n")
	d = a.sourceLanguage.auto.desired
	a.closeDialog = func(*Tab) (closeDecision, error) {
		h.now = d.due
		h.fireLanguage(t)
		h.drain()
		if a.sourceLanguage.auto.desired != d || a.sourceLanguage.task != nil {
			t.Fatal("modal canceled close invalidated due intent or started analysis")
		}
		return closeCancel, nil
	}
	if a.CloseTab(a.Active) || a.sourceLanguage.auto.desired != d || !a.sourceLanguage.auto.queued {
		t.Fatal("refused close lost original due")
	}
	h.drain()
	autoProbeFinish(t, a, h)
}

func TestLanguageAutoErrorReentryAndBindingStopPreserveSuccessor(t *testing.T) {
	for _, boundary := range []string{"error", "binding-stop"} {
		t.Run(boundary, func(t *testing.T) {
			a, tab, h := autoProbeFixture(t, ".go", "package p\n")
			entered, reports := false, 0
			exitCaptured := false
			var exit languageDocumentState
			var desired *sourceAutoDescriptor
			reenter := func() {
				if entered {
					return
				}
				entered = true
				a.revokeSourceInput(tab)
				a.bindSourceInput(tab)
				autoProbeReplace(t, tab, "package successor\n")
				// Distinct callback authority must survive the old bind return.
				tab.Editor.OnError = func(error) { reports++; a.Message = "successor error callback" }
				a.Message = "successor state"
				exit, desired = languageState(tab), a.sourceLanguage.auto.desired
			}
			if boundary == "binding-stop" {
				autoProbeReplace(t, tab, "package q\n")
				h.latest(h.language).onStop = reenter
				tab.View = &failingView{}
				a.bindSourceInput(tab)
			} else {
				tab.Editor.SetMarkedText(&ui.TextInputRange{Start: 0, End: 1}, "x", ui.TextInputRange{Start: 0, End: 1})
				marked, composing := tab.Editor.MarkedRange()
				selection := tab.Editor.Selection()
				if !tab.Editor.CompositionActive() || !composing || marked != (ui.TextInputRange{Start: 0, End: 1}) || selection.Range != (ui.TextInputRange{Start: 0, End: 1}) {
					t.Fatal("copy fixture did not retain a valid composition and non-empty selection")
				}
				next := a.sourceAutoHooks.afterFunc
				a.sourceAutoHooks.afterFunc = func(d time.Duration, fn func()) autosaveTimer {
					reenter()
					exitCaptured = true
					return next(d, fn)
				}
				failure := errors.New("old clipboard failure")
				copyCalls, oldErrorReports := 0, 0
				var copiedText string
				var reportedError error
				originalOnError := tab.Editor.OnError
				if originalOnError == nil {
					t.Fatal("original error callback absent")
				}
				tab.Editor.Copy = func(value string) error {
					copyCalls++
					copiedText = value
					return failure
				}
				tab.Editor.OnError = func(err error) { oldErrorReports++; reportedError = err; originalOnError(err) }
				tab.Editor.Command("copy")
				if copyCalls != 1 || copiedText != "x" || oldErrorReports != 1 || reportedError != failure {
					t.Fatal("old boundary did not receive the real clipboard error")
				}
			}
			if boundary == "error" && (!entered || !exitCaptured) {
				t.Fatal("old error boundary did not capture reentry and callback-exit state")
			}
			if !entered || a.Message != "successor state" || !reflect.DeepEqual(languageState(tab), exit) || a.sourceLanguage.auto.desired != desired {
				t.Fatal("old boundary changed successor callback-exit state")
			}
			tab.Editor.OnError(errors.New("probe successor callback"))
			if reports != 1 || a.Message != "successor error callback" {
				t.Fatal("old binding return overwrote successor error callback")
			}
		})
	}
}

func TestLanguageAutoDirectPresentationOpenCloseFloor(t *testing.T) {
	for _, inFlight := range []bool{false, true} {
		t.Run(map[bool]string{false: "pending", true: "in-flight"}[inFlight], func(t *testing.T) {
			a, tab, h := autoProbeFixture(t, ".go", "package p\n")
			autoProbeReplace(t, tab, "package q\n")
			var old *sourceLanguageTask
			var release func()
			if inFlight {
				started, finish := languageHold(t, a)
				release = finish
				autoProbeStart(t, a, h)
				old = a.sourceLanguage.task
				languageReceive(t, started)
			}
			last := a.lastEdit
			if err := a.openSourceLanguagePresentation(); err != nil {
				t.Fatal(err)
			}
			if !a.sourceLanguage.visible || !a.sourceLanguage.auto.explicit || a.sourceLanguage.auto.desired != nil || len(a.sourceLanguage.result.Symbols) != 0 || len(a.sourceLanguage.result.Diagnostics) != 0 || !a.lastEdit.Equal(last) {
				t.Fatal("direct open did not synchronously drop auto without changing recovery")
			}
			// Defer the actual Host OnChange invocation across close. The real
			// Editor still publishes its origin-bound delivery at the old OC site.
			onChange := tab.Editor.OnChange
			tab.Editor.OnChange = func() {}
			autoProbeReplace(t, tab, "package r\n")
			tab.Editor.OnChange = onChange
			if !a.closeSourceLanguagePresentation() {
				t.Fatal("direct close lost its own continuation")
			}
			if a.sourceLanguage.auto.explicit || a.sourceLanguage.auto.closeBinding != tab.inputBinding || a.sourceLanguage.auto.closeFloor != tab.Doc.Buffer.ChangeStamp() {
				t.Fatal("close did not record its independent confirmed floor")
			}
			onChange()
			if a.sourceLanguage.auto.desired != nil {
				t.Fatal("an edit at the close floor replayed as automatic work")
			}
			if old != nil {
				release()
				languageWait(t, old.done)
				h.drain()
			}
			if a.sourceLanguage.task != nil || len(a.sourceLanguage.result.Symbols) != 0 || len(a.sourceLanguage.result.Diagnostics) != 0 {
				t.Fatal("abandoned auto outcome became a hidden result")
			}
			autoProbeReplace(t, tab, "package s\n")
			if a.sourceLanguage.auto.desired == nil {
				t.Fatal("fresh edit after direct close did not arm")
			}
		})
	}
}

func TestLanguageAutoPresentationRefusalAndManualFloorDoNotSnapshot(t *testing.T) {
	a, tab, h := autoProbeFixture(t, ".go", "package p\n")
	autoProbeReplace(t, tab, "package q\n")
	d := a.sourceLanguage.auto.desired
	tab.Editor.SetMarkedText(&ui.TextInputRange{Start: 0, End: 1}, "x", ui.TextInputRange{Start: 1, End: 1})
	before, last, count := languageState(tab), a.lastEdit, len(h.language)
	if err := a.openSourceLanguagePresentation(); err == nil {
		t.Fatal("direct open crossed preedit")
	}
	if a.sourceLanguage.visible || a.sourceLanguage.auto.desired != d || a.sourceLanguage.task != nil || a.sourceLanguage.owner != nil || len(h.language) != count || !a.lastEdit.Equal(last) || !reflect.DeepEqual(languageState(tab), before) {
		t.Fatal("refused open captured/settled/rearmed provisional input")
	}
	if !a.closeSourceLanguagePresentation() || !a.lastEdit.Equal(last) || !reflect.DeepEqual(languageState(tab), before) {
		t.Fatal("close changed blocked source or recovery")
	}
	tab.Editor.SetMarkedText(&ui.TextInputRange{Start: 0, End: 1}, "p", ui.TextInputRange{Start: 1, End: 1})
	if err := tab.Editor.Flush(); err != nil {
		t.Fatal(err)
	}
	if a.sourceLanguage.auto.desired != nil {
		t.Fatal("net-zero after close replayed old intent")
	}
	// A real delivery already at the manual cancellation floor cannot clear
	// cancellation even if its Host callback is deliberately delivered later.
	onChange := tab.Editor.OnChange
	tab.Editor.OnChange = func() {}
	autoProbeReplace(t, tab, "package r\n")
	tab.Editor.OnChange = onChange
	a.cancelSourceLanguage()
	onChange()
	if !a.sourceLanguage.auto.manual || a.sourceLanguage.auto.desired != nil {
		t.Fatal("late floor-equal delivery cleared manual cancellation")
	}
	autoProbeReplace(t, tab, "package s\n")
	if a.sourceLanguage.auto.manual || a.sourceLanguage.auto.desired == nil {
		t.Fatal("fresh post-floor commit failed to arm")
	}
}

func TestLanguageAutoExplicitConstructorsPreserveStopSuccessor(t *testing.T) {
	for _, command := range []string{"read", "completion", "rename"} {
		t.Run(command, func(t *testing.T) {
			a, tab, h := autoProbeFixture(t, ".go", "package p\nvar before = 1\n")
			autoProbeReplace(t, tab, "package p\nvar renameMe = 1\n")
			at := ui.UTF16Len("package p\nvar rename")
			if err := tab.Doc.Buffer.SetSelection(at, at); err != nil {
				t.Fatal(err)
			}
			entered := false
			var dialog *sourceRenameDialog
			var owner *sourceLanguageOwner
			var task *sourceLanguageTask
			var request, presentation uint64
			var source languageDocumentState
			h.latest(h.language).onStop = func() {
				if entered {
					return
				}
				entered = true
				if command == "rename" {
					if err := a.openSourceRename(); err != nil {
						t.Fatal(err)
					}
					dialog = a.sourceLanguage.rename
					if dialog == nil {
						t.Fatal("successor rename dialog absent")
					}
					dialog.field.ReplaceText(&ui.TextInputRange{Start: 0, End: dialog.buffer.LenUTF16()}, "Successor")
				} else {
					a.runSourceLanguageCommand("symbols")
				}
				a.Message = "constructor successor"
				owner, task = a.sourceLanguage.owner, a.sourceLanguage.task
				request, presentation = a.sourceLanguage.request, a.sourceLanguage.presentation
				source = languageState(tab)
			}
			switch command {
			case "read":
				a.runSourceLanguageCommand("symbols")
			case "completion":
				a.runSourceCompletionCommand()
			case "rename":
				a.runSourceRenameCommand()
			}
			if !entered || a.Message != "constructor successor" || a.sourceLanguage.owner != owner || a.sourceLanguage.task != task || a.sourceLanguage.rename != dialog || a.sourceLanguage.request != request || a.sourceLanguage.presentation != presentation || !reflect.DeepEqual(languageState(tab), source) {
				t.Fatal("old explicit constructor overwrote callback-installed successor")
			}
			if command == "rename" && dialog.buffer.String() != "Successor" {
				t.Fatal("old rename seed overwrote successor field")
			}
			if task != nil {
				autoProbeFinish(t, a, h)
			}
		})
	}
}

func TestLanguageAutoRoutingAndRestoreStopPreserveSuccessor(t *testing.T) {
	for _, action := range []string{"configure", "reconfigure-command", "restore-auto", "restore-recovery"} {
		t.Run(action, func(t *testing.T) {
			suffix := ".go"
			if action == "restore-recovery" {
				suffix = ".txt"
			}
			a, tab, h := autoProbeFixture(t, suffix, "package p\n")
			autoProbeReplace(t, tab, "package q\n")
			entered := false
			var source languageDocumentState
			var desired *sourceAutoDescriptor
			var last time.Time
			var binding *sourceInputBinding
			var route uint64
			replace := func() {
				if entered {
					return
				}
				entered = true
				doc := document.New("package replacement\n")
				doc.Path = tab.Doc.Path
				a.revokeSourceInput(tab)
				tab.Doc, tab.Editor = doc, editor.New(doc.Buffer)
				a.bindSourceInput(tab)
				h.now = h.now.Add(time.Millisecond)
				autoProbeReplace(t, tab, "package successor\n")
				tab.SourceCapabilityBadge = "successor route"
				a.Message = "lifecycle successor"
				source, desired, last, binding, route = languageState(tab), a.sourceLanguage.auto.desired, a.lastEdit, tab.inputBinding, tab.sourceLanguageRoute
			}
			if action == "restore-recovery" {
				h.latest(h.recovery).onStop = replace
			} else {
				h.latest(h.language).onStop = replace
			}
			switch action {
			case "configure":
				if a.configureTab(tab) {
					t.Fatal("stale configuration succeeded")
				}
			case "reconfigure-command":
				a.finishSetPluginEnabled("fixture", true, nil)
			default:
				a.Restore()
			}
			if !entered || a.restoring || a.Message != "lifecycle successor" || !reflect.DeepEqual(languageState(tab), source) || a.sourceLanguage.auto.desired != desired || !a.lastEdit.Equal(last) || tab.inputBinding != binding || tab.sourceLanguageRoute != route || tab.SourceCapabilityBadge != "successor route" {
				t.Fatal("old route/Restore continuation overwrote callback-exit successor")
			}
		})
	}
}

func TestLanguageAutoRealMarkdownHostRebindAndInactiveRecovery(t *testing.T) {
	a, tab, tester, path := openRealMarkdownTasks(t, []byte("- [ ] pending\n"), 1180, 780)
	h := newAutoProbeClock(a)
	before := tab.Doc.Buffer.ChangeStamp()
	// The real Markdown factory retained its original Host. Legitimate B7
	// terminal registration replacement must not disconnect that Host.Changed.
	a.revokeSourceInput(tab)
	a.bindSourceInput(tab)
	if err := tester.Click(taskAccessName(false, "pending")); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if tab.Doc.Buffer.ChangeStamp() == before || !a.lastEdit.Equal(h.now) || len(h.recovery) != 1 || len(h.language) != 0 || !a.persistPending {
		t.Fatal("real Markdown commit lost recovery after legitimate rebind")
	}
	stamp, last, count := tab.Doc.Buffer.ChangeStamp(), a.lastEdit, len(h.recovery)
	host := a.editorHost(tab)
	host.Changed() // duplicate complete Host boundary, not a new edit
	h.now = h.now.Add(time.Second)
	if !a.Save(tab, false) {
		t.Fatal(a.Message)
	}
	if tab.Doc.Buffer.ChangeStamp() != stamp || !a.lastEdit.Equal(last) || len(h.recovery) != count {
		t.Fatal("duplicate Host callback/Save created another confirmed edit or quiet-time reset")
	}
	bytes, err := os.ReadFile(path)
	if err != nil || string(bytes) != "- [x] pending\n" {
		t.Fatalf("actual Markdown save bytes: %q / %v", bytes, err)
	}
	if !a.releasePresentation(tab) || tab.formatHostActive {
		t.Fatal("format release fixture")
	}
	h.now = h.now.Add(time.Second)
	tester.Frame() // legitimate retained-view reactivation, not an edit poll
	if err := tester.Click(taskAccessName(true, "pending")); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if !a.lastEdit.Equal(h.now) || len(h.recovery) != count+1 || tab.Doc.Buffer.String() != "- [ ] pending\n" {
		t.Fatal("retained real Markdown Host did not reactivate recovery")
	}
	// Append another tab without releasing the still-live original format view.
	other := a.appendTab(document.New("other"), "other.txt", "", false, false)
	if other == nil {
		t.Fatal("inactive live-tab fixture")
	}
	a.Active = len(a.Tabs) - 1
	h.now = h.now.Add(time.Second)
	if err := host.Buffer.ReplaceUTF16(host.Buffer.LenUTF16(), host.Buffer.LenUTF16(), "\n"); err != nil {
		t.Fatal(err)
	}
	host.Changed()
	if !a.lastEdit.Equal(h.now) || !a.persistPending || len(h.language) != 0 || a.Current() == tab {
		t.Fatal("live inactive format Host lost recovery or acquired language work")
	}
}

func TestLanguageAutoOldFormatHostsCannotConsumeSuccessorCommit(t *testing.T) {
	for _, mutation := range []string{"document", "route", "released"} {
		t.Run(mutation, func(t *testing.T) {
			a, tab, _, _ := openRealMarkdownTasks(t, []byte("- [ ] pending\n"), 1180, 780)
			h := newAutoProbeClock(a)
			old := a.editorHost(tab)
			switch mutation {
			case "document":
				doc := document.New("- [ ] successor\n")
				doc.Path = tab.Doc.Path
				tab.Doc, tab.Editor = doc, editor.New(doc.Buffer)
				a.bindSourceInput(tab)
			case "route":
				if !a.configureTab(tab) {
					t.Fatal("route fixture")
				}
				a.bindSourceInput(tab)
			case "released":
				if !a.releasePresentation(tab) {
					t.Fatal("release fixture")
				}
			}
			last, count, binding := a.lastEdit, len(h.recovery), tab.inputBinding
			if binding == nil {
				t.Fatal("successor binding absent")
			}
			observed := binding.stamp
			// This is a real committed successor transaction. Its own boundary
			// has not been delivered; an old Host may not adopt that observation.
			b := tab.Doc.Buffer
			if err := b.ReplaceUTF16(b.LenUTF16(), b.LenUTF16(), "\n"); err != nil {
				t.Fatal(err)
			}
			if b.ChangeStamp() == observed {
				t.Fatal("confirmed successor fixture")
			}
			old.Changed()
			if !a.lastEdit.Equal(last) || len(h.recovery) != count || binding.stamp != observed || a.sourceLanguage.auto.desired != nil {
				t.Fatal("old Doc/route/released format Host was redirected to successor observer")
			}
		})
	}
}

func TestLanguageAutoSaveAsMetadataRouteAndShutdownRetire(t *testing.T) {
	for _, transition := range []string{"same-path-save-as", "metadata", "route", "shutdown"} {
		t.Run(transition, func(t *testing.T) {
			a, tab, h := autoProbeFixture(t, ".go", "package p\n")
			autoProbeReplace(t, tab, "package q\n")
			old := h.language[0]
			switch transition {
			case "same-path-save-as":
				a.saveDialog = func(*Tab) (string, error) { return tab.Doc.Path, nil }
				if !a.Save(tab, true) {
					t.Fatal(a.Message)
				}
			case "metadata":
				tab.Doc.BOM = !tab.Doc.BOM
				a.reconcileSavedMetadata(tab, tab.Doc.Path, tab.Mode, false)
			case "route":
				a.configureTab(tab)
			case "shutdown":
				defer a.shuttingDown.Store(false)
				a.shuttingDown.Store(true)
				a.stopSourceLanguage()
			}
			message, last := a.Message, a.lastEdit
			h.now = h.now.Add(time.Second)
			old.fn()
			h.drain()
			if a.sourceLanguage.auto.desired != nil || a.sourceLanguage.task != nil || a.Message != message || !a.lastEdit.Equal(last) {
				t.Fatal("lifecycle transition did not retire old work")
			}
		})
	}
}
