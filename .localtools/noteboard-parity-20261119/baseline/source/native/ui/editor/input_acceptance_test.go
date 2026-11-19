package editor_test

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

func TestEditorInputAcceptanceSameLiteralTypeNotifiesWithoutMutation(t *testing.T) {
	const literal = "同字🙂\r\n原样"
	b := text.New(literal)
	e := editor.New(b)
	calls := 0
	var reported error
	e.OnError = func(err error) { reported = errors.Join(reported, err) }
	e.OnChange = func() {
		calls++
		if b.String() != literal || b.Version() != 0 || b.Selection() != (text.Range{Anchor: 8, Head: 8}) || b.History() != (text.HistoryStatus{}) || b.InGroup() {
			t.Fatalf("accepted callback preceded final public state: bytes=%q version=%d selection=%+v history=%+v group=%v", b.String(), b.Version(), b.Selection(), b.History(), b.InGroup())
		}
		if _, active := e.MarkedRange(); active {
			t.Fatal("ordinary same-literal Type opened a composition")
		}
	}
	tester := ui.NewTester(e.View, 480, 240)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	tester.Command("selectAll")
	tester.Key(0, ui.KeyRight)
	selection, version, history, snapshot := b.Selection(), b.Version(), b.History(), b.Snapshot()
	if selection != (text.Range{Anchor: 8, Head: 8}) || calls != 0 || reported != nil {
		t.Fatal("public selection fixture", selection, calls, reported)
	}
	tester.Command("selectAll")
	if b.Selection() != (text.Range{Anchor: 0, Head: 8}) {
		t.Fatal("real selectAll did not select the initiating literal", b.Selection())
	}
	tester.Type(literal)
	if calls != 1 || reported != nil || b.String() != literal || b.Version() != version || b.History() != history || b.Selection() != selection || !b.EqualSnapshot(snapshot) || b.InGroup() {
		t.Fatalf("accepted ordinary same-literal Type: calls=%d error=%v bytes=%q version=%d selection=%+v history=%+v group=%v", calls, reported, b.String(), b.Version(), b.Selection(), b.History(), b.InGroup())
	}
	if _, active := e.MarkedRange(); active {
		t.Fatal("ordinary input retained a composition")
	}
	if err := e.Flush(); err != nil || calls != 1 || b.Version() != version || b.History() != history || b.Selection() != selection {
		t.Fatal("settled Flush mutated or replayed accepted input", err, calls)
	}
	t.Log("accepted noncomposing same-literal Type notified once after finalization, with identical bytes/version/history/end selection")
}

func TestEditorInputAcceptanceInvalidUTF8DoesNotNotify(t *testing.T) {
	const literal = "拒收🙂\r\n原样"
	b := text.New(literal)
	e := editor.New(b)
	calls := 0
	var reported []error
	e.OnChange = func() { calls++ }
	e.OnError = func(err error) { reported = append(reported, err) }
	tester := ui.NewTester(e.View, 480, 240)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	tester.Command("selectAll")
	if err := b.SetSelection(8, 0); err != nil {
		t.Fatal(err)
	}
	selection, version, history, snapshot := b.Selection(), b.Version(), b.History(), b.Snapshot()
	tester.Type("\xff")
	if len(reported) != 1 || reported[0] != text.ErrUTF8 || calls != 0 || b.String() != literal || b.Version() != version || b.Selection() != selection || b.History() != history || !b.EqualSnapshot(snapshot) || b.InGroup() {
		t.Fatalf("invalid UTF-8 input notified or mutated state: calls=%d errors=%v bytes=%q selection=%+v", calls, reported, b.String(), b.Selection())
	}
	if _, active := e.MarkedRange(); active {
		t.Fatal("invalid ordinary input opened a composition")
	}
	if err := e.Flush(); err != nil || calls != 0 || len(reported) != 1 || b.Selection() != selection {
		t.Fatal("ordinary refusal poisoned Flush or replayed callbacks", err, calls, reported)
	}
	tester.Type(literal)
	if calls != 1 || len(reported) != 1 || b.String() != literal || b.Version() != version || b.History() != history || b.Selection() != (text.Range{Anchor: 8, Head: 8}) || b.InGroup() {
		t.Fatal("accepted same-literal retry did not retain original error and input semantics", calls, reported, b.Selection())
	}
	t.Log("invalid UTF-8 Type reported the original error once without OnChange; accepted retry notified once")
}

func TestEditorInputAcceptanceSizeFailureDoesNotNotify(t *testing.T) {
	b := text.NewWithLimit("a", 4)
	e := editor.New(b)
	calls := 0
	var reported []error
	e.OnChange = func() { calls++ }
	e.OnError = func(err error) { reported = append(reported, err) }
	tester := ui.NewTester(e.View, 480, 240)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	tester.Command("selectAll")
	tester.Key(0, ui.KeyRight)
	selection, version, history, snapshot := b.Selection(), b.Version(), b.History(), b.Snapshot()
	tester.Type("🙂")
	var limitError *text.SizeLimitError
	if len(reported) != 1 || !errors.Is(reported[0], text.ErrSizeLimit) || !errors.As(reported[0], &limitError) || limitError.Limit != 4 || calls != 0 || b.String() != "a" || b.Version() != version || b.Selection() != selection || b.History() != history || !b.EqualSnapshot(snapshot) || b.InGroup() {
		t.Fatalf("oversized ordinary Type notified or mutated: calls=%d errors=%v bytes=%q selection=%+v", calls, reported, b.String(), b.Selection())
	}
	if _, active := e.MarkedRange(); active {
		t.Fatal("ordinary size refusal opened a composition")
	}
	if err := e.Flush(); err != nil || calls != 0 || len(reported) != 1 {
		t.Fatal("ordinary size refusal blocked settled Flush", err, calls, reported)
	}
	tester.Type("你")
	if calls != 1 || len(reported) != 1 || b.String() != "a你" || b.Version() != version+1 || b.Selection() != (text.Range{Anchor: 2, Head: 2}) || b.History() != (text.HistoryStatus{UndoDepth: 1, RetainedBytes: 3}) || b.InGroup() {
		t.Fatal("accepted at-limit retry changed notification/history semantics", calls, reported, b.String(), b.History())
	}
	t.Log("oversized Type reported the four-byte content limit without OnChange; accepted at-limit retry notified once")
}

func TestEditorInputAcceptanceReadOnlyDoesNotNotify(t *testing.T) {
	const literal = "只读🙂\r\n原样"
	b := text.New(literal)
	e := editor.New(b)
	e.ReadOnly = true
	calls := 0
	var reported []error
	e.OnChange = func() { calls++ }
	e.OnError = func(err error) { reported = append(reported, err) }
	tester := ui.NewTester(e.View, 480, 240)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	tester.Command("selectAll")
	selection, version, history, snapshot := b.Selection(), b.Version(), b.History(), b.Snapshot()
	if selection != (text.Range{Anchor: 0, Head: 8}) {
		t.Fatal("readonly source remained navigable", selection)
	}
	tester.Type(literal)
	tester.Type("改动")
	tester.Type("")
	tester.Compose("组合", 2)
	if calls != 0 || len(reported) != 0 || b.String() != literal || b.Version() != version || b.Selection() != selection || b.History() != history || !b.EqualSnapshot(snapshot) || b.InGroup() {
		t.Fatalf("readonly ordinary/equal/empty/composed input mutated or notified: calls=%d errors=%v bytes=%q selection=%+v", calls, reported, b.String(), b.Selection())
	}
	if _, active := e.MarkedRange(); active {
		t.Fatal("readonly input opened a composition")
	}
	if err := e.Flush(); err != nil || calls != 0 || len(reported) != 0 || b.Selection() != selection {
		t.Fatal("readonly Flush mutated or notified", err, calls, reported)
	}
	t.Log("readonly equal/changed/empty/composed input preserved selected source and emitted no callbacks")
}

func TestEditorInputAcceptanceChangedTypeNotifiesOnce(t *testing.T) {
	b := text.New("a")
	e := editor.New(b)
	calls := 0
	var reported []error
	e.OnError = func(err error) { reported = append(reported, err) }
	e.OnChange = func() {
		calls++
		if b.String() != "你🙂\r\n" || b.Version() != 1 || b.Selection() != (text.Range{Anchor: 5, Head: 5}) || b.History() != (text.HistoryStatus{UndoDepth: 1, RetainedBytes: 10}) || b.InGroup() {
			t.Fatalf("changed callback missed finalized public state: bytes=%q version=%d selection=%+v history=%+v group=%v", b.String(), b.Version(), b.Selection(), b.History(), b.InGroup())
		}
		if _, active := e.MarkedRange(); active {
			t.Fatal("ordinary changed Type opened a composition")
		}
	}
	tester := ui.NewTester(e.View, 480, 240)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	tester.Command("selectAll")
	beforeSelection := b.Selection()
	tester.Type("你🙂\r\n")
	if calls != 1 || len(reported) != 0 || b.String() != "你🙂\r\n" || b.Version() != 1 || b.Selection() != (text.Range{Anchor: 5, Head: 5}) || b.History() != (text.HistoryStatus{UndoDepth: 1, RetainedBytes: 10}) {
		t.Fatal("changed Type duplicated or missed notification", calls, reported, b.String(), b.History())
	}
	if err := e.Flush(); err != nil || calls != 1 {
		t.Fatal("changed input Flush duplicated notification", err, calls)
	}
	if !b.Undo() || b.String() != "a" || b.Selection() != beforeSelection || !b.Redo() || b.String() != "你🙂\r\n" || calls != 1 || len(reported) != 0 {
		t.Fatal("single accepted replacement history lost exact source or notified without an input event", calls, b.String(), b.Selection())
	}
	t.Log("changed Type notified exactly once after finalized UTF-8 bytes, UTF-16 selection, version, and one undo transaction")
}

func TestEditorInputAcceptanceEmptyTypeNotifiesWithoutMutation(t *testing.T) {
	b := text.New("")
	e := editor.New(b)
	calls := 0
	var reported []error
	e.OnError = func(err error) { reported = append(reported, err) }
	e.OnChange = func() {
		calls++
		if b.String() != "" || b.Version() != 0 || b.Selection() != (text.Range{}) || b.History() != (text.HistoryStatus{}) || b.InGroup() {
			t.Fatalf("empty input callback missed finalized empty state: bytes=%q version=%d selection=%+v history=%+v group=%v", b.String(), b.Version(), b.Selection(), b.History(), b.InGroup())
		}
		if _, active := e.MarkedRange(); active {
			t.Fatal("ordinary empty Type opened a composition")
		}
	}
	tester := ui.NewTester(e.View, 480, 240)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	snapshot := b.Snapshot()
	tester.Type("")
	if calls != 1 || len(reported) != 0 || b.String() != "" || b.Version() != 0 || b.Selection() != (text.Range{}) || b.History() != (text.HistoryStatus{}) || !b.EqualSnapshot(snapshot) || b.InGroup() {
		t.Fatal("accepted empty Type did not notify once without mutation", calls, reported, b.Selection(), b.History())
	}
	if err := e.Flush(); err != nil || calls != 1 || len(reported) != 0 {
		t.Fatal("empty input Flush replayed callbacks", err, calls, reported)
	}
	t.Log("accepted empty ordinary Type notified once while empty bytes/version/history/selection remained exact")
}

func TestEditorInputAcceptanceForeignCompositionDoesNotNotify(t *testing.T) {
	const literal = "所有🙂\r\n原样"
	b := text.New(literal)
	e := editor.New(b)
	calls := 0
	var reported []error
	e.OnChange = func() { calls++ }
	e.OnError = func(err error) { reported = append(reported, err) }
	tester := ui.NewTester(e.View, 480, 240)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	tester.Command("selectAll")
	if err := b.SetSelection(8, 0); err != nil {
		t.Fatal(err)
	}
	owner, err := b.BeginCompositionGroup()
	if err != nil {
		t.Fatal(err)
	}
	selection, version, history, snapshot := b.Selection(), b.Version(), b.History(), b.Snapshot()
	tester.Type(literal)
	tester.Compose("新", 1)
	if len(reported) != 2 || reported[0] != text.ErrGroup || reported[1] != text.ErrGroup || calls != 0 || b.String() != literal || b.Version() != version || b.Selection() != selection || b.History() != history || !b.EqualSnapshot(snapshot) || !b.InGroup() || !b.OwnsComposition(owner) {
		t.Fatalf("foreign composition input notified or stole state: calls=%d errors=%v bytes=%q selection=%+v owner=%v", calls, reported, b.String(), b.Selection(), b.OwnsComposition(owner))
	}
	if _, active := e.MarkedRange(); active {
		t.Fatal("Editor claimed the foreign composition")
	}
	if err := e.Flush(); err != nil || calls != 0 || len(reported) != 2 || !b.InGroup() || !b.OwnsComposition(owner) {
		t.Fatal("Flush closed or notified a foreign-owned group", err, calls, reported)
	}
	if err := b.CancelCompositionGroup(owner); err != nil {
		t.Fatal(err)
	}
	tester.Type(literal)
	if calls != 1 || len(reported) != 2 || b.String() != literal || b.Version() != version || b.History() != history || b.Selection() != (text.Range{Anchor: 8, Head: 8}) || b.InGroup() || b.OwnsComposition(owner) {
		t.Fatal("same-literal input did not resume after explicit owner cancellation", calls, reported, b.Selection())
	}
	t.Log("foreign composition rejected equal Type and new Compose without OnChange or owner loss; explicit owner cancellation enabled one accepted notification")
}

func TestEditorInputAcceptanceLostCompositionDoesNotNotify(t *testing.T) {
	const literal = "失效🙂\r\n原样"
	b := text.New(literal)
	e := editor.New(b)
	calls := 0
	var reported []error
	e.OnChange = func() { calls++ }
	e.OnError = func(err error) { reported = append(reported, err) }
	tester := ui.NewTester(e.View, 480, 240)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	tester.Command("selectAll")
	tester.Compose(literal, 7)
	if mark, active := e.MarkedRange(); !active || mark != (ui.TextInputRange{Start: 0, End: 8}) || !b.InGroup() || calls != 1 || len(reported) != 0 || b.String() != literal || b.Version() != 0 || b.History() != (text.HistoryStatus{}) {
		t.Fatal("real same-literal composition was not accepted", mark, active, calls, reported)
	}
	b.ResetHistory()
	if err := b.BeginGroup(); err != nil {
		t.Fatal(err)
	}
	selection, version, history, snapshot := b.Selection(), b.Version(), b.History(), b.Snapshot()
	tester.Type(literal)
	if len(reported) != 1 || reported[0] != text.ErrComposition || calls != 1 || b.String() != literal || b.Version() != version || b.Selection() != selection || b.History() != history || !b.EqualSnapshot(snapshot) || !b.InGroup() {
		t.Fatalf("lost composition commit notified or stole replacement group: calls=%d errors=%v bytes=%q selection=%+v group=%v", calls, reported, b.String(), b.Selection(), b.InGroup())
	}
	if _, active := e.MarkedRange(); active {
		t.Fatal("lost token was reported as an owned composition")
	}
	if err := e.Flush(); err != text.ErrComposition || calls != 1 || len(reported) != 1 || !b.InGroup() {
		t.Fatal("lost composition Flush notified or closed a replacement group", err, calls, reported)
	}
	tester.Key(0, ui.KeyEscape)
	if len(reported) != 2 || reported[1] != text.ErrComposition || calls != 1 || e.Flush() != nil || !b.InGroup() || b.String() != literal || b.Version() != version || b.Selection() != selection || b.History() != history {
		t.Fatal("explicit stale-token cancel notified or stole the foreign ordinary group", calls, reported, b.Selection())
	}
	if !b.CancelGroup() {
		t.Fatal("foreign ordinary owner lost its cancellation capability")
	}
	tester.Command("selectAll")
	tester.Type(literal)
	if calls != 2 || len(reported) != 2 || b.String() != literal || b.Version() != version || b.History() != history || b.Selection() != (text.Range{Anchor: 8, Head: 8}) || b.InGroup() {
		t.Fatal("ordinary same-literal input did not resume after explicit stale-token and owner cancellation", calls, reported, b.Selection())
	}
	t.Log("failed same-literal commit on a lost composition token emitted no acceptance notification and preserved the new owner's group")
}
