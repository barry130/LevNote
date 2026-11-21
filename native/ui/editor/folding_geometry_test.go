package editor

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

func TestFoldRealGutterClickProjectionHitBoundsAndTabAction(t *testing.T) {
	value := "{\nfirst hidden\n}\nafter wrap 中文🙂 " + strings.Repeat("word ", 3)
	b := text.New(value)
	e := New(b)
	e.SoftWrap = true
	tester := ui.NewTester(e.View, 280, 300)
	if err := installBraceProvider(e); err != nil {
		t.Fatal(err)
	}
	if err := e.SetPosition(0, 0); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	box, ok := tester.Find("文档编辑区")
	if !ok {
		t.Fatal("editor box not found")
	}
	start, _ := b.LineRange(3)
	before, _, visible := e.BoundsForRange(ui.TextInputRange{Start: start.Start(), End: start.Start()})
	if !visible {
		t.Fatalf("tail line initially unavailable rect=%+v actual=%+v scroll=%v height=%v rows=%v", before, func() ui.TextInputRange {
			_, a, _ := e.BoundsForRange(ui.TextInputRange{Start: start.Start(), End: start.Start()})
			return a
		}(), e.ScrollY, e.height, e.totalVisualRows())
	}
	// Gutter arrow is a real pointer-operated surface control.
	tester.ClickAt(box.X+8, box.Y+12+e.rowHeight()/2)
	tester.Frame()
	if !e.FoldCandidates()[0].Collapsed {
		t.Fatal("gutter click did not fold")
	}
	after, _, visible := e.BoundsForRange(ui.TextInputRange{Start: start.Start(), End: start.Start()})
	if !visible || !(after.Y < before.Y) {
		t.Fatalf("tail y did not project: before=%+v after=%+v visible=%v", before, after, visible)
	}
	hidden, _ := b.OffsetOfLine(1)
	if _, _, ok := e.BoundsForRange(ui.TextInputRange{Start: hidden, End: hidden}); ok {
		t.Fatal("hidden caret received IME bounds")
	}
	index, ok := e.IndexForPoint(ui.Point{X: after.X, Y: after.Y + after.H/2})
	if !ok || index != start.Start() {
		t.Fatalf("visible source hit=%d want=%d ok=%v", index, start.Start(), ok)
	}
	if err := e.SetPosition(0, 0); err != nil {
		t.Fatal(err)
	}
	// Focus a real operation button, then reach the next one by keyboard Tab
	// and activate it with Enter (not by clicking the second button).
	if err := tester.Click("折叠当前块"); err != nil {
		t.Fatal(err)
	}
	tester.Key(0, ui.KeyTab)
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	if e.FoldCandidates()[0].Collapsed {
		t.Fatal("Tab-reachable action did not unfold current block")
	}
	if err := tester.Click("展开全部"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if !tester.HasText("重新分析折叠") {
		t.Fatal("fold controls/status absent")
	}
}

func TestFoldNavigationAfterResizeAndFontChange(t *testing.T) {
	value := "{\r\nhide🙂\n}\nafter\n" + strings.Repeat("wrap words ", 45) + "\n" + strings.Repeat("x", maxShapedLine+200) + "\nend"
	b := text.New(value)
	e := New(b)
	e.SoftWrap = true
	if err := installBraceProvider(e); err != nil {
		t.Fatal(err)
	}
	tester := ui.NewTester(e.View, 490, 280)
	if err := tester.Click(e.Label); err != nil {
		t.Fatal(err)
	}
	if err := e.SetPosition(0, 0); err != nil {
		t.Fatal(err)
	}
	if err := e.ToggleFoldAt(0); err != nil {
		t.Fatal(err)
	}
	tester.SetSize(300, 260)
	e.SetFontSize(20)
	tester.Frame()
	tester.Key(0, ui.KeyDown)
	if line, _, visual := e.GetPosition(); line != 2 || visual != 1 {
		t.Fatalf("Down did not jump fold: %d/%d", line, visual)
	}
	closeStart, _ := b.OffsetOfLine(2)
	assertCaretRoundTrip(t, e, tester, closeStart)
	tester.Key(0, ui.KeyEnd)
	if line, col, _ := e.GetPosition(); line != 2 || col != 1 {
		t.Fatalf("End = %d:%d", line, col)
	}
	tester.Key(0, ui.KeyHome)
	if line, col, _ := e.GetPosition(); line != 2 || col != 0 {
		t.Fatalf("Home = %d:%d", line, col)
	}
	tester.Key(0, ui.KeyPageDown)
	if _, _, v := e.GetPosition(); v <= 1 {
		t.Fatal("PageDown did not scroll visible rows")
	}
	tester.Key(0, ui.KeyPageUp)
	if line, _, v := e.GetPosition(); line != 2 || v != 1 {
		t.Fatalf("PageUp returned %d/%d", line, v)
	}
	tester.Key(0, ui.KeyLeft)
	if line, _, _ := e.GetPosition(); line != 1 || e.FoldCandidates()[0].Collapsed {
		t.Fatal("Left did not reveal source neighbor")
	}
	if err := e.SetPosition(0, 0); err != nil {
		t.Fatal(err)
	}
	if err := e.ToggleFoldAt(0); err != nil {
		t.Fatal(err)
	}
	r, _ := b.LineRange(5)
	assertCaretRoundTrip(t, e, tester, r.End()-12)
	if e.ScrollY <= 0 || !e.FoldCandidates()[0].Collapsed {
		t.Fatal("scroll to segmented line disturbed the fold")
	}
	hidden, _ := b.LineRange(1)
	e.Reveal(hidden.End(), hidden.Start())
	if b.Selection() != (text.Range{Anchor: hidden.End(), Head: hidden.Start()}) || e.FoldCandidates()[0].Collapsed {
		t.Fatal("Reveal lost direction or left input hidden")
	}
}

func TestFoldPointerAndKeyboardRejectSameTextUndo(t *testing.T) {
	value := "{\ninside\n}\nafter"
	b := text.New(value)
	e := New(b)
	e.LineNumbers = false
	if err := installBraceProvider(e); err != nil {
		t.Fatal(err)
	}
	tester := ui.NewTester(e.View, 480, 240)
	box, ok := tester.Find(e.Label)
	if !ok {
		t.Fatal("missing editor")
	}
	x, y := box.X+8, box.Y+12+e.rowHeight()/2
	refreshAfterUndo := func() {
		t.Helper()
		end := b.LenUTF16()
		if err := b.ReplaceUTF16(end, end, "!"); err != nil {
			t.Fatal(err)
		}
		if !b.Undo() || b.String() != value || len(e.FoldCandidates()) != 0 {
			t.Fatal("same text undo did not invalidate folds")
		}
		if err := e.RefreshFolds(); err != nil {
			t.Fatal(err)
		}
		tester.Frame()
	}
	for round := 0; round < 20; round++ {
		tester.Press(x, y)
		refreshAfterUndo()
		tester.Release(x, y)
		if candidates := e.FoldCandidates(); len(candidates) != 1 || candidates[0].Collapsed {
			t.Fatalf("old arrow press was accepted in round %d", round)
		}
		tester.ClickAt(x, y)
		if !e.FoldCandidates()[0].Collapsed {
			t.Fatalf("new arrow click failed in round %d", round)
		}
		if err := e.UnfoldAll(); err != nil {
			t.Fatal(err)
		}
		tester.Frame()
	}
	if err := tester.Click("折叠当前块"); err != nil {
		t.Fatal(err)
	}
	if err := e.UnfoldAll(); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if !tester.Focused("折叠当前块") {
		t.Fatal("fold button lost focus")
	}
	tester.KeyDown(ui.KeyEnter)
	refreshAfterUndo()
	tester.KeyUp(ui.KeyEnter)
	if e.FoldCandidates()[0].Collapsed {
		t.Fatal("old keyboard press was accepted")
	}
	tester.Key(0, ui.KeyEnter)
	if !e.FoldCandidates()[0].Collapsed {
		t.Fatal("fresh keyboard press failed")
	}
}

func TestFoldTenThousandHiddenLinesDoNotShapeAndRowCountsAreNotCapped(t *testing.T) {
	var builder strings.Builder
	builder.WriteString("{\n")
	for i := 0; i < 10000; i++ {
		builder.WriteString(strings.Repeat("m", 120) + "\n")
	}
	builder.WriteString("}\nafter")
	value := builder.String()
	b := text.New(value)
	e := New(b)
	e.SoftWrap = true
	tester := ui.NewTester(e.View, 420, 240)
	if err := e.SetFoldProvider(func(snapshot FoldSnapshot) (FoldResult, error) {
		closeStart := text.UTF16Len(snapshot.Source) - len("\nafter") - 1
		return FoldResult{Complete: true, Folds: []FoldRange{{OpenLine: 0, CloseLine: 10001, OpenStart: 0, OpenEnd: 1, CloseStart: closeStart, CloseEnd: closeStart + 1}}}, nil
	}, 1, MaxFoldBytes); err != nil {
		t.Fatal(err)
	}
	// Visit 400 rows to demonstrate that shape counts are not assumed to fit
	// within lineCacheLimit=256.
	for line := 0; line < 400; line++ {
		e.line(line)
	}
	if len(e.rowCounts) < 350 {
		t.Fatalf("visited wrap rows were not retained beyond the shape cache: %d", len(e.rowCounts))
	}
	if err := e.SetPosition(0, 0); err != nil {
		t.Fatal(err)
	}
	if err := e.ToggleFoldAt(0); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	for line := range e.cache {
		if e.hiddenLine(line) {
			t.Fatalf("hidden line %d was shaped", line)
		}
	}
	if len(e.cache) > lineCacheLimit {
		t.Fatalf("shape cache exceeded bound: %d", len(e.cache))
	}
	if e.totalVisualRows() > 10 {
		t.Fatalf("folded projection still exposes hidden rows: %d", e.totalVisualRows())
	}
	hidden, _ := b.OffsetOfLine(5000)
	if _, _, ok := e.BoundsForRange(ui.TextInputRange{Start: hidden, End: hidden}); ok {
		t.Fatal("hidden tail returned geometry")
	}
}
