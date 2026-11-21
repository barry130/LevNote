package editor

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

func TestFoldLineCommandRevealsAffectedHiddenLineBeforeMove(t *testing.T) {
	value := "first\n{\nhidden\n}\nlast"
	b := text.New(value)
	e := New(b)
	tester := ui.NewTester(e.View, 480, 220)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	if err := installBraceProvider(e); err != nil {
		t.Fatal(err)
	}
	if err := e.SetPosition(1, 0); err != nil {
		t.Fatal(err)
	}
	if err := e.ToggleFoldAt(1); err != nil {
		t.Fatal(err)
	}
	closeStart, _ := b.OffsetOfLine(3)
	if err := b.SetSelection(closeStart, closeStart); err != nil {
		t.Fatal(err)
	}
	tester.Key(ui.Alt, ui.KeyUp)
	if e.folds.ready {
		t.Fatal("edit should invalidate candidates and leave full text")
	}
	if b.String() == value {
		t.Fatal("line command did not move source")
	}
	if !e.Command("undo") || b.String() != value {
		t.Fatal("line operation did not undo exactly")
	}
}

func TestFoldCopyPreservesHiddenBytesAndDelayedCutRejectsNewProvider(t *testing.T) {
	value := "{\nsecret🙂\n}\nafter\r\nlast"
	b := text.New(value)
	e := New(b)
	_ = ui.NewTester(e.View, 480, 220)
	if err := installBraceProvider(e); err != nil {
		t.Fatal(err)
	}
	if err := e.SetPosition(0, 0); err != nil {
		t.Fatal(err)
	}
	if err := e.ToggleFoldAt(0); err != nil {
		t.Fatal(err)
	}
	if err := b.SetSelection(0, b.LenUTF16()); err != nil {
		t.Fatal(err)
	}
	var copied string
	e.Copy = func(s string) error { copied = s; return nil }
	if !e.Command("copy") || copied != value {
		t.Fatalf("copy omitted hidden source: %q", copied)
	}
	if err := e.UnfoldAll(); err != nil {
		t.Fatal(err)
	}
	if err := b.SetSelection(0, b.LenUTF16()); err != nil {
		t.Fatal(err)
	}
	e.Copy = nil
	var done func(error)
	e.CopyThen = func(_ string, finish func(error)) { done = finish }
	if !e.Command("cut") || done == nil {
		t.Fatal("async cut not issued")
	}
	if err := e.SetFoldProvider(func(FoldSnapshot) (FoldResult, error) { return FoldResult{Complete: true}, nil }, 2, MaxFoldBytes); err != nil {
		t.Fatal(err)
	}
	done(nil)
	if b.String() != value {
		t.Fatal("stale cut deleted source")
	}
}

func TestFoldRevealBeforeReplaceAndReadonlyOrBudgetRejection(t *testing.T) {
	value := "{\nhidden\n}\nend"
	b := text.NewWithLimit(value, len(value))
	e := New(b)
	_ = ui.NewTester(e.View, 420, 200)
	if err := installBraceProvider(e); err != nil {
		t.Fatal(err)
	}
	if err := e.SetPosition(0, 0); err != nil {
		t.Fatal(err)
	}
	if err := e.ToggleFoldAt(0); err != nil {
		t.Fatal(err)
	}
	hidden, _ := b.OffsetOfLine(1)
	if err := b.SetSelection(hidden, hidden+1); err != nil {
		t.Fatal(err)
	}
	version, history := b.Version(), b.History()
	var refused error
	e.OnError = func(err error) { refused = err }
	e.ReadOnly = true
	e.ReplaceText(nil, "x")
	if !e.FoldCandidates()[0].Collapsed || b.Version() != version || b.History() != history {
		t.Fatal("read-only refusal altered fold or source")
	}
	e.ReadOnly = false
	e.ReplaceText(nil, "longer")
	if !e.FoldCandidates()[0].Collapsed || b.Version() != version || b.History() != history || b.Selection() != (text.Range{Anchor: hidden, Head: hidden + 1}) {
		t.Fatal("size rejection changed fold or source state")
	}
	e.ReplaceText(nil, "X")
	if e.folds.ready || !strings.Contains(b.String(), "X") {
		t.Fatalf("accepted edit did not reveal and replace: ready=%v text=%q selection=%+v composition=%v err=%v refused=%v", e.folds.ready, b.String(), b.Selection(), e.composing, e.compositionIssue(), refused)
	}
	if !e.Command("undo") || b.String() != value {
		t.Fatal("edit undo did not restore source")
	}
}

func TestFoldLineCopyBudgetRefusalAndFreshCut(t *testing.T) {
	value := "{\r\ninside🙂\n}\nafter"
	b := text.NewWithLimit(value, len(value))
	e := New(b)
	if err := installBraceProvider(e); err != nil {
		t.Fatal(err)
	}
	if err := e.ToggleFoldAt(0); err != nil {
		t.Fatal(err)
	}
	closeStart, _ := b.OffsetOfLine(2)
	if err := b.SetSelection(closeStart, closeStart); err != nil {
		t.Fatal(err)
	}
	beforeVersion, beforeHistory := b.Version(), b.History()
	var reported error
	e.OnError = func(err error) { reported = err }
	if !e.Command("copyLinesUp") || reported == nil || !e.FoldCandidates()[0].Collapsed || b.Version() != beforeVersion || b.History() != beforeHistory {
		t.Fatal("rejected line copy changed fold or history")
	}
	if err := b.SetSelection(b.LenUTF16(), 0); err != nil {
		t.Fatal(err)
	}
	var complete func(error)
	e.CopyThen = func(valueOnClipboard string, done func(error)) {
		if valueOnClipboard != value {
			t.Fatalf("cut clipboard=%q", valueOnClipboard)
		}
		complete = done
	}
	if !e.Command("cut") || complete == nil || b.String() != value || !e.FoldCandidates()[0].Collapsed {
		t.Fatal("cut changed source before clipboard confirmation")
	}
	complete(nil)
	if b.String() != "" || len(e.FoldCandidates()) != 0 || !e.Command("undo") || b.String() != value || b.Selection() != (text.Range{Anchor: text.UTF16Len(value), Head: 0}) {
		t.Fatal("fresh cut/undo lost source or selection direction")
	}
}

func TestFoldStaleTargetAfterBufferReplacement(t *testing.T) {
	e := New(text.New("{\na\n}"))
	_ = ui.NewTester(e.View, 420, 180)
	if err := installBraceProvider(e); err != nil {
		t.Fatal(err)
	}
	target := e.FoldCandidates()[0].Target
	other := New(e.Buffer)
	if err := installBraceProvider(other); err != nil {
		t.Fatal(err)
	}
	if err := other.ToggleFold(target); err == nil {
		t.Fatal("target for a different editor sharing this buffer was accepted")
	}
	e.Buffer = text.New("{\na\n}")
	if err := e.ToggleFold(target); err == nil {
		t.Fatal("target for previous buffer accepted")
	}
}
