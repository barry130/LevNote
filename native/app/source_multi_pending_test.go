package app

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/session"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

func task40SourceFixture(t *testing.T, source string, limit int) (*App, *Tab, *ui.Tester) {
	t.Helper()
	a := newTestApp(t, filepath.Join(t.TempDir(), "config"))
	a.NewDocument("text")
	tab := a.Current()
	if tab == nil || tab.Editor == nil || tab.Doc == nil {
		t.Fatal("task40 source fixture has no text tab")
	}
	buffer := text.NewWithLimit(source, limit)
	tab.Doc.Buffer = buffer
	tab.Editor.Buffer = buffer
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	if err := tester.Click(tab.Editor.Label); err != nil {
		t.Fatal(err)
	}
	return a, tab, tester
}

func task40SelectionsEqual(got, want text.SelectionSet) bool {
	return got.Primary == want.Primary && slices.Equal(got.Ranges, want.Ranges)
}

func task40SetSelections(t *testing.T, tab *Tab, ranges []text.Range, primary int) text.SelectionSet {
	t.Helper()
	want := text.SelectionSet{Ranges: slices.Clone(ranges), Primary: primary}
	if err := tab.Doc.Buffer.SetSelectionSet(want); err != nil {
		t.Fatal(err)
	}
	if got := tab.Doc.Buffer.SelectionSet(); !task40SelectionsEqual(got, want) {
		t.Fatalf("selection set = %+v, want %+v", got, want)
	}
	return want
}

func task40StartPending(t *testing.T, tab *Tab, tester *ui.Tester, value string, caret int) text.SelectionSet {
	t.Helper()
	start := task40SetSelections(t, tab, []text.Range{{Anchor: 1, Head: 1}, {Anchor: 4, Head: 4}}, 0)
	tester.Compose(value, caret)
	if !tab.Editor.MultiCompositionPending() || !tab.Editor.CompositionActive() {
		t.Fatalf("multi composition was not retained: pending=%v active=%v", tab.Editor.MultiCompositionPending(), tab.Editor.CompositionActive())
	}
	return start
}

func TestSourceMultiPendingNavigationBlocksRealShortcuts(t *testing.T) {
	a, tab, tester := task40SourceFixture(t, "ab cd", 9)
	start := task40StartPending(t, tab, tester, "拼🙂", 2)
	before := tab.Doc.Buffer.Snapshot()
	beforeSelection := tab.Doc.Buffer.Selection()
	beforeHistory := tab.Doc.Buffer.History()
	state := currentTask40CaptureState(tab)
	if _, marked := tab.Editor.MarkedRange(); marked || !task40SelectionsEqual(tab.Doc.Buffer.SelectionSet(), start) || !tab.Doc.Buffer.InGroup() {
		t.Fatalf("refused first preedit lost pending owner: marked=%v selection=%+v group=%v", marked, tab.Doc.Buffer.SelectionSet(), tab.Doc.Buffer.InGroup())
	}
	tester.Key(ui.Cmd, ui.KeyF)
	if a.ShowFind || !tester.Focused(tab.Editor.Label) || !strings.Contains(a.Message, "输入法正在组合") {
		t.Fatalf("Cmd+F crossed refused multi pending: show=%v focus=%v message=%q", a.ShowFind, tester.Focused(tab.Editor.Label), a.Message)
	}
	tester.Key(ui.Cmd, ui.KeyG)
	if a.ShowGoToLine || !tester.Focused(tab.Editor.Label) || !strings.Contains(a.Message, "输入法正在组合") {
		t.Fatalf("Cmd+G crossed refused multi pending: show=%v focus=%v message=%q", a.ShowGoToLine, tester.Focused(tab.Editor.Label), a.Message)
	}
	if err := tab.Editor.Flush(); !errors.Is(err, text.ErrCompositionPending) {
		t.Fatalf("refused owner was settled by navigation: %v", err)
	}
	if !tab.Doc.Buffer.EqualSnapshot(before) || !task40SelectionsEqual(tab.Doc.Buffer.SelectionSet(), start) || tab.Doc.Buffer.Selection() != beforeSelection || tab.Doc.Buffer.History() != beforeHistory || !currentTask40CaptureState(tab).equal(state) {
		t.Fatal("navigation changed refused source state")
	}

	tester.Key(0, ui.KeyEscape)
	if tab.Editor.MultiCompositionPending() || tab.Doc.Buffer.InGroup() || !tester.Focused(tab.Editor.Label) {
		t.Fatal("Escape did not cancel the refused multi owner in the source editor")
	}
	tester.Key(ui.Cmd, ui.KeyF)
	tester.Frame()
	if !a.ShowFind || !tester.Focused("查找内容") {
		t.Fatalf("Cmd+F did not open Find after explicit Escape: show=%v texts=%q", a.ShowFind, tester.Texts())
	}
	if err := tester.Click(tab.Editor.Label); err != nil {
		t.Fatal(err)
	}
	task40StartPending(t, tab, tester, "拼🙂", 2)
	tester.Key(ui.Cmd, ui.KeyF)
	if !a.ShowFind || !tester.Focused(tab.Editor.Label) || !strings.Contains(a.Message, "输入法正在组合") {
		t.Fatalf("Cmd+F stole source focus while Find was open: show=%v sourceFocus=%v message=%q", a.ShowFind, tester.Focused(tab.Editor.Label), a.Message)
	}
	tester.Key(0, ui.KeyEscape)
	if tab.Editor.MultiCompositionPending() || !tester.Focused(tab.Editor.Label) || !a.ShowFind {
		t.Fatalf("source Escape was not delivered before Find close: pending=%v focus=%v find=%v", tab.Editor.MultiCompositionPending(), tester.Focused(tab.Editor.Label), a.ShowFind)
	}
	tester.Key(ui.Cmd, ui.KeyF)
	tester.Frame()
	if !a.ShowFind || !tester.Focused("查找内容") {
		t.Fatal("Find could not refocus after source Escape")
	}
}

func TestSourceMultiPendingGoToLineShortcutAndConfirmedRetry(t *testing.T) {
	a, tab, tester := task40SourceFixture(t, "ab cd", 9)
	task40StartPending(t, tab, tester, "拼🙂", 2)
	tester.Key(ui.Cmd, ui.KeyG)
	if a.ShowGoToLine || !tester.Focused(tab.Editor.Label) || !strings.Contains(a.Message, "输入法正在组合") {
		t.Fatalf("Cmd+G stole source focus: show=%v focus=%v message=%q", a.ShowGoToLine, tester.Focused(tab.Editor.Label), a.Message)
	}
	tester.Key(0, ui.KeyEscape)
	if tab.Editor.MultiCompositionPending() || !tester.Focused(tab.Editor.Label) {
		t.Fatal("Escape did not clear GoToLine-blocking pending state")
	}
	// A fresh accepted multi preedit remains pending until native final text;
	// navigation must reject it, and final confirmation must restore the route.
	buffer := tab.Doc.Buffer
	// Replacing two ASCII characters permits two three-byte final characters
	// within the same nine-byte ceiling; inserting twice would exceed it.
	task40SetSelections(t, tab, []text.Range{{Anchor: 2, Head: 1}, {Anchor: 4, Head: 5}}, 0)
	tester.Compose("中", 1)
	if !tab.Editor.MultiCompositionPending() {
		t.Fatal("legal multi preedit did not remain pending for final text")
	}
	state := currentTask40CaptureState(tab)
	tester.Key(ui.Cmd, ui.KeyG)
	if a.ShowGoToLine || !tester.Focused(tab.Editor.Label) || !currentTask40CaptureState(tab).equal(state) {
		t.Fatal("GoToLine crossed or changed a live multi preedit")
	}
	tester.Key(ui.Cmd, ui.KeyF)
	if a.ShowFind || !tester.Focused(tab.Editor.Label) || !currentTask40CaptureState(tab).equal(state) {
		t.Fatal("Find crossed or changed a live multi preedit")
	}
	tester.Type("尾")
	if tab.Editor.MultiCompositionPending() || buffer.String() != "a尾 c尾" {
		t.Fatalf("native final confirmation did not restore source editing: pending=%v source=%q", tab.Editor.MultiCompositionPending(), buffer.String())
	}
	tester.Key(ui.Cmd, ui.KeyG)
	tester.Frame()
	if !a.ShowGoToLine || !tester.Focused("转到行号") {
		t.Fatal("GoToLine did not open after multi confirmation")
	}
	tester.Key(0, ui.KeyEscape)
}

func TestSourceMultiPendingPersistAndRestorePreservesPreviousBytes(t *testing.T) {
	a, tab, tester := task40SourceFixture(t, "ab cd", 9)
	baselineSelection := task40SetSelections(t, tab, []text.Range{{Anchor: 2, Head: 1}, {Anchor: 4, Head: 4}}, 0)
	primary := baselineSelection.Ranges[baselineSelection.Primary]
	a.Persist()
	path := filepath.Join(a.configDir, "session.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	source := tab.Doc.Buffer.String()
	// Make a wrong publication observable even when the first preedit changes
	// no text or selection. A successful retry must save this preference later.
	a.Dark = !a.Dark
	tester.Compose("拼🙂", 2)
	if _, marked := tab.Editor.MarkedRange(); marked {
		t.Fatal("refused first preedit unexpectedly has a live mark")
	}
	if !tab.Editor.MultiCompositionPending() || !task40SelectionsEqual(tab.Doc.Buffer.SelectionSet(), baselineSelection) {
		t.Fatalf("first refusal did not preserve pending source state: selection=%+v", tab.Doc.Buffer.SelectionSet())
	}
	state := currentTask40CaptureState(tab)
	// Count recovery failures separately from the editor's size-refusal issue.
	issuesBefore := len(a.Issues)
	a.Persist()
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || len(a.Issues) != issuesBefore+1 || !strings.Contains(a.lastPersistError, "多选输入") {
		t.Fatalf("refused pending persistence changed bytes or hid issue: equal=%v issues=%q error=%q", bytes.Equal(before, after), a.Issues, a.lastPersistError)
	}
	if got := currentTask40CaptureState(tab); !got.equal(state) || !tester.Focused(tab.Editor.Label) {
		t.Fatalf("refused persistence changed source ownership state: before=%+v after=%+v", state, got)
	}
	reopenedBeforeCancel := newTestApp(t, a.configDir)
	reopenedBeforeCancel.Restore()
	if len(reopenedBeforeCancel.Tabs) != 1 {
		t.Fatalf("Restore during pending has %d tabs", len(reopenedBeforeCancel.Tabs))
	}
	if got := reopenedBeforeCancel.Current().Doc.Buffer; got.String() != source || got.Selection() != primary {
		t.Fatalf("Restore during pending lost prior source/selection: source=%q selection=%+v", got.String(), got.Selection())
	}
	a.Message = "另一项操作的告知"
	a.Persist()
	if got := len(a.Issues); got != issuesBefore+1 || a.Message != a.lastPersistError || !bytes.Equal(mustReadFile(t, path), before) || !currentTask40CaptureState(tab).equal(state) {
		t.Fatalf("unchanged pending persistence duplicated issue or hid its retry: issues=%d message=%q error=%q", got, a.Message, a.lastPersistError)
	}
	message := a.Message
	if err := a.reportPersistFailure(nil); err != nil || a.Message != message || len(a.Issues) != issuesBefore+1 || a.lastPersistError != message {
		t.Fatal("nil persistence error changed the current disclosure")
	}
	tester.Key(0, ui.KeyEscape)
	if tab.Editor.MultiCompositionPending() || tab.Doc.Buffer.InGroup() || !task40SelectionsEqual(tab.Doc.Buffer.SelectionSet(), baselineSelection) {
		t.Fatal("Escape did not restore the original directed selections before retry")
	}
	a.Persist()
	snapshot := readRecovery(t, a)
	if len(snapshot.Tabs) != 1 || snapshot.Tabs[0].Draft == nil || *snapshot.Tabs[0].Draft != source || snapshot.Tabs[0].Selection != primary || a.lastPersistError != "" {
		t.Fatalf("successful retry did not preserve source/primary selection: %#v error=%q", snapshot, a.lastPersistError)
	}
	reopened := newTestApp(t, a.configDir)
	reopened.Restore()
	if len(reopened.Tabs) != 1 {
		t.Fatalf("Restore after retry has %d tabs", len(reopened.Tabs))
	}
	if got := reopened.Current().Doc.Buffer; got.String() != source || got.Selection() != primary || reopened.Dark != a.Dark {
		t.Fatalf("Restore lost retry source/selection/preference: source=%q selection=%+v dark=%v", got.String(), got.Selection(), reopened.Dark)
	}
}

func TestRecoveryAutosaveMultiPendingUsesQueuedDeadlineWithoutRetry(t *testing.T) {
	a, tab, tester := task40SourceFixture(t, "ab cd", 9)
	task40SetSelections(t, tab, []text.Range{{Anchor: 1, Head: 1}, {Anchor: 4, Head: 4}}, 0)
	a.Persist()
	path := filepath.Join(a.configDir, "session.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	h := installAutosaveClock(a)
	a.Dark = !a.Dark // Distinguish a forbidden publication with unchanged text.
	tester.Compose("拼🙂", 2)
	state := currentTask40CaptureState(tab)
	issuesBefore := len(a.Issues)
	a.noteEdit()
	if len(h.timers) != 1 {
		t.Fatalf("autosave did not arm one quiet timer: %d", len(h.timers))
	}
	h.now = h.now.Add(autosaveDelay - time.Millisecond)
	h.latest(t).fn()
	if len(h.queue) != 1 {
		t.Fatal("timer did not post through queued dispatcher")
	}
	if got := mustReadFile(t, path); !bytes.Equal(got, before) {
		t.Fatal("899ms timer published before queued UI dispatch")
	}
	h.drain()
	if got := mustReadFile(t, path); !bytes.Equal(got, before) {
		t.Fatal("899ms UI delivery published before the deadline")
	}
	if len(h.timers) != 2 || h.latest(t).delay != time.Millisecond {
		t.Fatalf("899ms dispatch did not rearm the final millisecond: timers=%d delay=%s", len(h.timers), h.latest(t).delay)
	}
	h.now = h.now.Add(time.Millisecond)
	rejectTimerCount := len(h.timers)
	rejectTimer := h.latest(t)
	rejectTimer.fn()
	if len(h.queue) != 1 || !bytes.Equal(mustReadFile(t, path), before) {
		t.Fatal("900ms timer captured before queued UI dispatch")
	}
	h.drain()
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || a.persistPending || len(h.timers) != rejectTimerCount || len(a.Issues) != issuesBefore+1 || !tab.Editor.MultiCompositionPending() {
		t.Fatalf("automatic pending capture published/retried/cleared state: equal=%v pending=%v timers=%d issues=%q multi=%v", bytes.Equal(before, after), a.persistPending, len(h.timers), a.Issues, tab.Editor.MultiCompositionPending())
	}
	if got := currentTask40CaptureState(tab); !got.equal(state) || !tester.Focused(tab.Editor.Label) {
		t.Fatalf("automatic capture changed source state: before=%+v after=%+v", state, got)
	}
	if err := tab.Editor.Flush(); !errors.Is(err, text.ErrCompositionPending) {
		t.Fatalf("automatic capture settled pending token: %v", err)
	}
	rejectTimer.fn()
	h.drain()
	if len(h.timers) != rejectTimerCount || len(a.Issues) != issuesBefore+1 || !bytes.Equal(mustReadFile(t, path), before) {
		t.Fatal("duplicate queued deadline retried unchanged pending input")
	}
	// A second explicit attempt reports the same issue once and leaves bytes.
	a.Persist()
	afterRetry, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, afterRetry) || len(a.Issues) != issuesBefore+1 {
		t.Fatalf("duplicate explicit rejection changed state: err=%v equal=%v issues=%q", err, bytes.Equal(before, afterRetry), a.Issues)
	}
	tester.Key(0, ui.KeyEscape)
	if tab.Editor.MultiCompositionPending() || tab.Doc.Buffer.InGroup() {
		t.Fatal("Escape retained the pending owner after autosave refusal")
	}
	a.Persist()
	snapshot := readRecovery(t, a)
	if len(snapshot.Tabs) != 1 || snapshot.Tabs[0].Draft == nil || *snapshot.Tabs[0].Draft != "ab cd" || snapshot.Tabs[0].Selection != (text.Range{Anchor: 1, Head: 1}) || a.lastPersistError != "" {
		t.Fatalf("post-Escape recovery did not contain original source and primary: %#v error=%q", snapshot, a.lastPersistError)
	}
}

func TestSessionDraftCaptureReentrantFirstRefusalIsCheckedAfterCallback(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.NewDocument("text")
	tab := a.Current()
	if tab == nil {
		t.Fatal("missing text tab")
	}
	buffer := text.NewWithLimit("ab cd", 9)
	tab.Doc.Buffer, tab.Editor.Buffer = buffer, buffer
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	if err := tester.Click(tab.Editor.Label); err != nil {
		t.Fatal(err)
	}
	selection := task40SetSelections(t, tab, []text.Range{{Anchor: 1, Head: 1}, {Anchor: 4, Head: 4}}, 0)
	a.Persist()
	path := filepath.Join(a.configDir, "session.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	state := currentTask40CaptureState(tab)
	// The callback itself legitimately creates the pending owner. Capture must
	// leave that resulting group intact, along with all original buffer state.
	state.group, state.compositionActive, state.multiPending = true, true, true
	owner := &sessionDraftFixture{draft: editor.SessionDraft{Title: "cell", Content: "confirmed"}, dirty: true}
	owner.capture = func() {
		if !tab.Editor.HandleInput(ui.InputEvent{Kind: ui.InputCompose, Text: "拼🙂"}) {
			t.Fatal("reentrant composition callback was not consumed")
		}
	}
	tab.View = owner
	if err := a.persistTabs(a.Tabs, a.Active, "捕获归属测试"); err == nil {
		t.Fatal("reentrant first multi refusal was published")
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || owner.captures != 1 || !tab.Editor.MultiCompositionPending() || !task40SelectionsEqual(buffer.SelectionSet(), selection) {
		t.Fatalf("callback pending was not rejected after capture: equal=%v captures=%d pending=%v selection=%+v", bytes.Equal(before, after), owner.captures, tab.Editor.MultiCompositionPending(), buffer.SelectionSet())
	}
	if got := currentTask40CaptureState(tab); !got.equal(state) {
		t.Fatalf("reentrant capture changed buffer/selection/history state: before=%+v after=%+v", state, got)
	}
	if len(a.Issues) == 0 || !strings.Contains(a.Issues[len(a.Issues)-1], "多选输入") {
		t.Fatalf("reentrant refusal was not visible: %q", a.Issues)
	}
}

func TestSourceMultiPendingLivePersistAndConfirmedRestore(t *testing.T) {
	a, tab, tester := task40SourceFixture(t, "ab cd", 9)
	original := task40SetSelections(t, tab, []text.Range{{Anchor: 2, Head: 1}, {Anchor: 5, Head: 4}}, 1)
	a.Persist()
	path := filepath.Join(a.configDir, "session.json")
	before := mustReadFile(t, path)
	tester.Compose("中", 1)
	state := currentTask40CaptureState(tab)
	if !state.markActive || !state.multiPending || state.source == "ab cd" {
		t.Fatalf("live primary-only preedit fixture failed: %+v", state)
	}
	issues := len(a.Issues)
	a.Persist()
	if !bytes.Equal(mustReadFile(t, path), before) || len(a.Issues) != issues+1 || !strings.Contains(a.Message, "多选输入") || !currentTask40CaptureState(tab).equal(state) || !tester.Focused(tab.Editor.Label) {
		t.Fatal("explicit capture published or changed live multi preedit")
	}
	reopened := newTestApp(t, a.configDir)
	reopened.Restore()
	if len(reopened.Tabs) != 1 || reopened.Current().Doc.Buffer.String() != "ab cd" || reopened.Current().Doc.Buffer.Selection() != original.Ranges[original.Primary] {
		t.Fatal("rejected live preedit did not retain the previous recoverable text and directed primary")
	}
	tester.Type("尾")
	confirmed := text.SelectionSet{Ranges: []text.Range{{Anchor: 2, Head: 2}, {Anchor: 5, Head: 5}}, Primary: 1}
	if tab.Doc.Buffer.String() != "a尾 c尾" || !task40SelectionsEqual(tab.Doc.Buffer.SelectionSet(), confirmed) || tab.Editor.CompositionActive() || tab.Doc.Buffer.InGroup() {
		t.Fatal("capture lost the complete native confirmation authority")
	}
	a.Persist()
	snapshot := readRecovery(t, a)
	if len(snapshot.Tabs) != 1 || snapshot.Tabs[0].Draft == nil || *snapshot.Tabs[0].Draft != "a尾 c尾" || snapshot.Tabs[0].Selection != confirmed.Ranges[confirmed.Primary] || a.lastPersistError != "" {
		t.Fatalf("confirmed multi text/primary was not persisted: %#v", snapshot)
	}
	reopened = newTestApp(t, a.configDir)
	reopened.Restore()
	if len(reopened.Tabs) != 1 || reopened.Current().Doc.Buffer.String() != "a尾 c尾" || reopened.Current().Doc.Buffer.Selection() != confirmed.Ranges[confirmed.Primary] {
		t.Fatal("confirmed complete text and primary did not survive real Restore")
	}
}

func TestRecoveryAutosaveMultiPendingLivePreeditEscRestoresAfterQuietPeriod(t *testing.T) {
	a, tab, tester := task40SourceFixture(t, "ab cd", 9)
	original := task40SetSelections(t, tab, []text.Range{{Anchor: 2, Head: 1}, {Anchor: 5, Head: 4}}, 0)
	a.Persist()
	path := filepath.Join(a.configDir, "session.json")
	before := mustReadFile(t, path)
	h := installAutosaveClock(a)
	tester.Compose("中", 1)
	state := currentTask40CaptureState(tab)
	if !state.markActive || !state.multiPending || len(h.timers) != 1 {
		t.Fatalf("live preedit did not arm its actual edit deadline: %+v timers=%d", state, len(h.timers))
	}
	issues := len(a.Issues)
	h.now = h.now.Add(autosaveDelay - time.Millisecond)
	h.latest(t).fn()
	if len(h.queue) != 1 || !bytes.Equal(mustReadFile(t, path), before) {
		t.Fatal("live preedit timer captured before UI drain")
	}
	h.drain()
	if !bytes.Equal(mustReadFile(t, path), before) || len(h.timers) != 2 || h.latest(t).delay != time.Millisecond || !currentTask40CaptureState(tab).equal(state) {
		t.Fatal("live preedit captured at 899ms or changed source state")
	}
	h.now = h.now.Add(time.Millisecond)
	rejected := h.latest(t)
	rejected.fn()
	if len(h.queue) != 1 || !bytes.Equal(mustReadFile(t, path), before) {
		t.Fatal("live preedit captured at 900ms before UI drain")
	}
	h.drain()
	if !bytes.Equal(mustReadFile(t, path), before) || a.persistPending || len(a.Issues) != issues+1 || len(h.timers) != 2 || !currentTask40CaptureState(tab).equal(state) || !tester.Focused(tab.Editor.Label) {
		t.Fatal("900ms live-preedit rejection changed state, published, retried or hid its issue")
	}
	rejected.fn()
	h.drain()
	if len(h.timers) != 2 || len(a.Issues) != issues+1 || !bytes.Equal(mustReadFile(t, path), before) {
		t.Fatal("live preedit rejection scheduled an unchanged retry")
	}
	tester.Key(0, ui.KeyEscape)
	if tab.Editor.CompositionActive() || tab.Doc.Buffer.String() != "ab cd" || !task40SelectionsEqual(tab.Doc.Buffer.SelectionSet(), original) || !a.persistPending || len(h.timers) != 3 {
		t.Fatal("Escape did not restore the complete source and arm a fresh edit deadline")
	}
	h.now = h.now.Add(autosaveDelay - time.Millisecond)
	h.latest(t).fn()
	h.drain()
	if !bytes.Equal(mustReadFile(t, path), before) || h.latest(t).delay != time.Millisecond {
		t.Fatal("post-Escape capture ignored its new 899ms quiet boundary")
	}
	h.now = h.now.Add(time.Millisecond)
	h.latest(t).fn()
	if len(h.queue) != 1 || !a.persistPending {
		t.Fatal("post-Escape capture bypassed the UI queue")
	}
	h.drain()
	snapshot := readRecovery(t, a)
	if len(snapshot.Tabs) != 1 || snapshot.Tabs[0].Draft == nil || *snapshot.Tabs[0].Draft != "ab cd" || snapshot.Tabs[0].Selection != original.Ranges[original.Primary] || a.persistPending || a.lastPersistError != "" {
		t.Fatalf("post-Escape quiet capture lost restored source/primary: %#v", snapshot)
	}
	reopened := newTestApp(t, a.configDir)
	reopened.Restore()
	if len(reopened.Tabs) != 1 || reopened.Current().Doc.Buffer.String() != "ab cd" || reopened.Current().Doc.Buffer.Selection() != original.Ranges[original.Primary] {
		t.Fatal("post-Escape automatic recovery did not survive real Restore")
	}
}

func TestSourceMultiPendingCaptureChecksAllTabsBeforeAndAfterCallbacks(t *testing.T) {
	for _, phase := range []string{"preflight", "callback"} {
		for _, position := range []string{"earlier", "later"} {
			t.Run(phase+"-"+position, func(t *testing.T) {
				a, first, tester := task40SourceFixture(t, "ab cd", 9)
				middle := a.appendTab(document.New("confirmed host"), "middle.txt", "", false, false)
				last := a.appendTab(document.New("ab cd"), "last.txt", "", false, false)
				last.Doc.Buffer = text.NewWithLimit("ab cd", 9)
				last.Editor.Buffer = last.Doc.Buffer
				for _, tab := range []*Tab{first, last} {
					task40SetSelections(t, tab, []text.Range{{Anchor: 2, Head: 1}, {Anchor: 5, Head: 4}}, 0)
				}
				a.Persist()
				path := filepath.Join(a.configDir, "session.json")
				before := mustReadFile(t, path)
				tabs := []*Tab{first, middle, last}
				owners := make([]*sessionDraftFixture, len(tabs))
				for i, tab := range tabs {
					owners[i] = &sessionDraftFixture{draft: editor.SessionDraft{Content: "confirmed cell"}, dirty: true}
					tab.View = owners[i]
				}
				target := first
				if position == "later" {
					target = last
				}
				states := make([]task40CaptureState, len(tabs))
				for i, tab := range tabs {
					states[i] = currentTask40CaptureState(tab)
				}
				startPending := func() {
					if !target.Editor.HandleInput(ui.InputEvent{Kind: ui.InputCompose, Text: "拼🙂"}) {
						t.Fatal("callback did not consume its native composition")
					}
				}
				if phase == "preflight" {
					startPending()
				} else {
					owners[1].capture = startPending
				}
				for i, tab := range tabs {
					if tab == target {
						states[i].group, states[i].compositionActive, states[i].multiPending = true, true, true
					}
				}
				builder, err := a.captureSessionTabs(a.Tabs, a.Active, nil, "跨标签恢复捕获")
				if builder != nil || !errors.Is(err, text.ErrCompositionPending) || !strings.Contains(a.Message, "多选输入") || !bytes.Equal(mustReadFile(t, path), before) {
					t.Fatalf("%s did not reject pending with nil builder and intact recovery: builder=%v err=%v", phase, builder, err)
				}
				for i, tab := range tabs {
					wantCaptures := 0
					if phase == "callback" {
						wantCaptures = 1
					}
					if owners[i].captures != wantCaptures || owners[i].flushes != 0 || owners[i].releases != 0 || !currentTask40CaptureState(tab).equal(states[i]) {
						t.Fatalf("capture crossed preflight or mutated tab %d: captures=%d want=%d before=%+v after=%+v", i, owners[i].captures, wantCaptures, states[i], currentTask40CaptureState(tab))
					}
				}
				if !tester.Focused(first.Editor.Label) || a.persistPending {
					t.Fatal("refused capture changed focus or retained an automatic request")
				}
				if err := target.Editor.Flush(); !errors.Is(err, text.ErrCompositionPending) {
					t.Fatalf("capture settled the callback's owner: %v", err)
				}
				if !target.Editor.HandleInput(ui.InputEvent{Kind: ui.InputText, Text: "尾"}) || target.Doc.Buffer.String() != "a尾 c尾" || target.Editor.CompositionActive() {
					t.Fatal("capture lost the original callback's final broadcast authority")
				}
			})
		}
	}
}

func TestSourceMultiPendingCaptureSkipsBinaryInspectorAndDiscardedDrafts(t *testing.T) {
	for _, kind := range []string{"binary", "inspector", "discard-untitled", "discard-path"} {
		t.Run(kind, func(t *testing.T) {
			a, ignored, _ := task40SourceFixture(t, "ab cd", 9)
			task40SetSelections(t, ignored, []text.Range{{Anchor: 2, Head: 1}, {Anchor: 5, Head: 4}}, 0)
			ignored.Editor.HandleInput(ui.InputEvent{Kind: ui.InputCompose, Text: "拼🙂"})
			if !ignored.Editor.MultiCompositionPending() {
				t.Fatal("ignored source lacks the adversarial pending owner")
			}
			var discarded []bool
			wantTabs := 2 // ordinary source plus its confirmed independent draft
			switch kind {
			case "binary":
				ignored.BinaryPath, ignored.BinaryKind = filepath.Join(t.TempDir(), "image.png"), document.BinaryKindPNG
				ignored.MatchedPlugin.ID = "image"
				wantTabs++
			case "inspector":
				ignored.ByteInspector = true
			case "discard-untitled", "discard-path":
				discarded = []bool{true}
				if kind == "discard-path" {
					ignored.Doc.Path = filepath.Join(t.TempDir(), "original.txt")
					wantTabs++
				}
			}
			ignoredOwner := &sessionDraftFixture{draft: editor.SessionDraft{Content: "must not capture"}, dirty: true}
			ignored.View = ignoredOwner
			kept := a.appendTab(document.New("kept confirmed source"), "kept.txt", "", false, false)
			keptOwner := &sessionDraftFixture{draft: editor.SessionDraft{Content: "kept confirmed cell"}, dirty: true}
			kept.View = keptOwner
			state := currentTask40CaptureState(ignored)
			if err := a.persistTabsWithDiscard(a.Tabs, 1, discarded, "跳过不捕获的草稿"); err != nil {
				t.Fatal(err)
			}
			if ignoredOwner.captures != 0 || keptOwner.captures != 1 || !currentTask40CaptureState(ignored).equal(state) || ignoredOwner.flushes != 0 || keptOwner.flushes != 0 {
				t.Fatal("capture queried an excluded draft callback or changed its pending owner")
			}
			snapshot := readRecovery(t, a)
			if len(snapshot.Tabs) != wantTabs || snapshot.Tabs[wantTabs-2].Draft == nil || *snapshot.Tabs[wantTabs-2].Draft != "kept confirmed source" || snapshot.Tabs[wantTabs-1].Draft == nil || *snapshot.Tabs[wantTabs-1].Draft != "kept confirmed cell" {
				t.Fatalf("excluded pending blocked or displaced legitimate capture: %#v", snapshot)
			}
			if kind == "binary" {
				record, err := session.RestoreBinary(snapshot.Tabs[0])
				if err != nil || record.Path != ignored.BinaryPath || record.Kind != document.BinaryKindPNG {
					t.Fatalf("excluded binary did not preserve its metadata-only record: %#v err=%v", record, err)
				}
			}
			if kind == "discard-path" && (snapshot.Tabs[0].Path != ignored.Doc.Path || snapshot.Tabs[0].Draft != nil) {
				t.Fatalf("discarded source did not preserve its metadata-only record: %#v", snapshot)
			}
		})
	}
}

type task40CaptureState struct {
	source                          string
	selection                       text.SelectionSet
	version, selectionGeneration    uint64
	history                         text.HistoryStatus
	dirty, group                    bool
	marked                          ui.TextInputRange
	markActive                      bool
	compositionActive, multiPending bool
}

func (s task40CaptureState) equal(other task40CaptureState) bool {
	return s.source == other.source && task40SelectionsEqual(s.selection, other.selection) &&
		s.version == other.version && s.selectionGeneration == other.selectionGeneration &&
		s.history == other.history && s.dirty == other.dirty && s.group == other.group &&
		s.marked == other.marked && s.markActive == other.markActive &&
		s.compositionActive == other.compositionActive && s.multiPending == other.multiPending
}

func currentTask40CaptureState(tab *Tab) task40CaptureState {
	marked, active := tab.Editor.MarkedRange()
	return task40CaptureState{
		source: tab.Doc.Buffer.String(), selection: tab.Doc.Buffer.SelectionSet(),
		version: tab.Doc.Buffer.Version(), selectionGeneration: tab.Doc.Buffer.SelectionGeneration(),
		history: tab.Doc.Buffer.History(), dirty: tab.Doc.Dirty(), group: tab.Doc.Buffer.InGroup(),
		marked: marked, markActive: active,
		compositionActive: tab.Editor.CompositionActive(), multiPending: tab.Editor.MultiCompositionPending(),
	}
}

func TestSourceMultiPendingGetterIsNilSafeAndPure(t *testing.T) {
	var nilEditor *editor.Editor
	if nilEditor.CompositionActive() || nilEditor.MultiCompositionPending() {
		t.Fatal("nil editor getter was not false")
	}
	empty := &editor.Editor{}
	if empty.CompositionActive() || empty.MultiCompositionPending() {
		t.Fatal("unattached editor getter was not false")
	}
	_, tab, _ := task40SourceFixture(t, "a b", 9)
	before := currentTask40CaptureState(tab)
	if tab.Editor.CompositionActive() || tab.Editor.MultiCompositionPending() || !currentTask40CaptureState(tab).equal(before) {
		t.Fatal("fresh editor getter changed state")
	}
	// Legacy single-selection composition remains active without becoming
	// pending multi-selection input, and a query must not settle its group.
	tab.Editor.SetMarkedText(nil, "中", ui.TextInputRange{Start: 1, End: 1})
	before = currentTask40CaptureState(tab)
	if !tab.Editor.CompositionActive() || tab.Editor.MultiCompositionPending() || !currentTask40CaptureState(tab).equal(before) {
		t.Fatal("legacy single-selection getter changed state or classification")
	}
	if err := tab.Editor.Flush(); err != nil || tab.Editor.CompositionActive() || tab.Doc.Buffer.InGroup() {
		t.Fatalf("legacy single-selection Flush changed contract: %v", err)
	}

	for _, tc := range []struct {
		name, preedit string
		caret         int
		marked        bool
	}{
		{name: "first-refused-no-mark", preedit: "拼🙂", caret: 2},
		{name: "accepted-live-preedit", preedit: "中", caret: 1, marked: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, tab, tester := task40SourceFixture(t, "ab cd", 9)
			original := task40SetSelections(t, tab, []text.Range{{Anchor: 2, Head: 1}, {Anchor: 4, Head: 5}}, 0)
			tester.Compose(tc.preedit, tc.caret)
			state := currentTask40CaptureState(tab)
			if state.markActive != tc.marked || !state.group || !state.multiPending || !state.compositionActive {
				t.Fatalf("incorrect preedit fixture: %+v", state)
			}
			if !tc.marked && (state.source != "ab cd" || !task40SelectionsEqual(state.selection, original)) {
				t.Fatalf("refused first preedit changed original source/selections: %+v", state)
			}
			for range 8 {
				if !tab.Editor.CompositionActive() || !tab.Editor.MultiCompositionPending() {
					t.Fatal("pending getter changed after a repeated read")
				}
			}
			if got := currentTask40CaptureState(tab); !got.equal(state) {
				t.Fatalf("pending getter mutated source state: before=%+v after=%+v", state, got)
			}
			// This legal final value also proves queries retained the original
			// token and complete directed selection set for atomic broadcast.
			tester.Type("尾")
			want := text.SelectionSet{Ranges: []text.Range{{Anchor: 2, Head: 2}, {Anchor: 5, Head: 5}}, Primary: 0}
			if tab.Doc.Buffer.String() != "a尾 c尾" || !task40SelectionsEqual(tab.Doc.Buffer.SelectionSet(), want) || tab.Editor.CompositionActive() || tab.Editor.MultiCompositionPending() || tab.Doc.Buffer.InGroup() {
				t.Fatalf("getter lost final broadcast authority: source=%q selections=%+v", tab.Doc.Buffer.String(), tab.Doc.Buffer.SelectionSet())
			}
		})
	}
}
