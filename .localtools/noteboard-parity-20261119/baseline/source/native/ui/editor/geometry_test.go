package editor

import (
	"errors"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

func assertCaretRoundTrip(t *testing.T, e *Editor, tester *ui.Tester, offset int) ui.Rect {
	t.Helper()
	e.affinity = ui.TextDownstream
	e.selectTo(offset, false)
	tester.Frame()
	r, actual, ok := e.BoundsForRange(ui.TextInputRange{Start: offset, End: offset})
	if !ok || actual.Start != offset || actual.End != offset {
		t.Fatalf("caret %d unavailable: rect=%+v actual=%+v visible=%v", offset, r, actual, ok)
	}
	index, ok := e.IndexForPoint(ui.Point{X: r.X, Y: r.Y + r.H/2})
	if !ok || index != offset {
		t.Fatalf("caret/pointer roundtrip %d => %d, rect=%+v", offset, index, r)
	}
	box, _ := tester.Find("文档编辑区")
	ime, active := tester.TextCaret()
	if !active || math.Abs(float64(ime.Y-box.Y-r.Y)) > 1 || math.Abs(float64(ime.X-box.X-r.X)) > 1 {
		t.Fatalf("native IME differs from drawn caret: ime=%+v local=%+v box=%+v active=%v", ime, r, box, active)
	}
	return r
}

func TestCaretPointerIMEAgreeAcrossWrapLinesResizeAndFonts(t *testing.T) {
	value := "alpha 中文🙂 e\u0301 " + strings.Repeat("word ", 18) + "\r\n\r\nsecond 中文🙂 line\nlast"
	e := New(text.New(value))
	e.SoftWrap = true
	tester := ui.NewTester(e.View, 290, 180)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	for _, width := range []int{290, 410, 210} {
		tester.SetSize(width, 180)
		tester.Frame()
		for _, size := range []float32{16, 21} {
			e.SetFontSize(size)
			tester.Frame()
			for _, line := range []int{0, 1, 2, 3} {
				span, _ := e.Buffer.LineRange(line)
				assertCaretRoundTrip(t, e, tester, span.Start())
				assertCaretRoundTrip(t, e, tester, span.End())
			}
			for _, offset := range []int{6, 8, 10, 13, 24, 42} {
				assertCaretRoundTrip(t, e, tester, offset)
			}
		}
	}
}

func TestWrapHomeEndAndVerticalPreserveVisualColumn(t *testing.T) {
	e := New(text.New("abcdefghijklmno\nx\nabcdefghijklmno\n" + strings.Repeat("abcd ", 30)))
	e.SoftWrap = true
	tester := ui.NewTester(e.View, 260, 150)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	_ = e.SetPosition(0, 8)
	tester.Frame()
	tester.Key(0, ui.KeyDown)
	if line, col, _ := e.GetPosition(); line != 1 || col != 1 {
		t.Fatalf("short line %d:%d", line, col)
	}
	tester.Key(0, ui.KeyDown)
	if line, col, _ := e.GetPosition(); line != 2 || col != 8 {
		t.Fatalf("desired column was lost: %d:%d", line, col)
	}
	_ = e.SetPosition(3, 0)
	tester.Frame()
	l := e.line(3)
	if l.rows < 4 {
		t.Fatal("fixture must wrap over several rows")
	}
	_ = e.SetPosition(3, l.lines[2].Start+2)
	tester.Frame()
	tester.Key(0, ui.KeyHome)
	_, column, visual := e.GetPosition()
	if column != l.lines[2].Start || visual != e.visualBase(3)+2 {
		t.Fatalf("wrapped Home moved to %d / row %d", column, visual)
	}
	tester.Key(0, ui.KeyEnd)
	_, column, visual = e.GetPosition()
	if column != l.lines[2].End || visual != e.visualBase(3)+2 {
		t.Fatalf("wrapped End lost upstream affinity: %d / row %d", column, visual)
	}
	tester.Key(0, ui.KeyDown)
	if _, _, next := e.GetPosition(); next != visual+1 {
		t.Fatalf("Down skipped wrapped row: %d => %d", visual, next)
	}
	_ = e.SetPosition(3, 2)
	tester.Frame()
	_, _, before := e.GetPosition()
	tester.Key(0, ui.KeyPageDown)
	_, _, after := e.GetPosition()
	want := max(1, int(e.height/e.rowHeight())-1)
	if after-before != want {
		t.Fatalf("PageDown moved %d rows; want %d", after-before, want)
	}
	tester.Key(0, ui.KeyPageUp)
	if _, _, after := e.GetPosition(); after != before {
		t.Fatalf("PageUp did not restore row %d: %d", before, after)
	}
}

func TestLongLineTailAndSegmentBoundaryEditDeleteUndo(t *testing.T) {
	prefix := strings.Repeat("a", segmentUnits-1)
	value := prefix + "🙂e\u0301" + strings.Repeat("b", maxShapedLine+50) + "👨‍👩‍👧‍👦"
	e := New(text.New(value))
	var warnings int
	e.OnError = func(error) { warnings++ }
	tester := ui.NewTester(e.View, 480, 240)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	if !tester.HasText("UTF-16 位置") || !tester.HasText("下一段") {
		t.Fatal("long line segment disclosure/controls missing")
	}
	_ = e.SetPosition(0, e.Buffer.LenUTF16())
	tester.Frame()
	assertCaretRoundTrip(t, e, tester, e.Buffer.LenUTF16())
	tester.Type("中")
	if !strings.HasSuffix(e.Buffer.String(), "👨‍👩‍👧‍👦中") {
		t.Fatal("tail was not editable")
	}
	tester.Key(0, ui.KeyBackspace)
	tester.Key(0, ui.KeyBackspace)
	if strings.HasSuffix(e.Buffer.String(), "👨‍👩‍👧‍👦") || strings.Contains(e.Buffer.String(), "\u200d") {
		t.Fatal("family grapheme was only partially deleted")
	}
	tester.Key(ui.Cmd, ui.KeyZ)
	tester.Key(ui.Cmd, ui.KeyZ)
	tester.Key(ui.Cmd, ui.KeyZ)
	if e.Buffer.String() != value {
		t.Fatal("tail undo did not preserve original")
	}
	_ = e.SetPosition(0, segmentUnits+1)
	tester.Frame()
	tester.Key(0, ui.KeyBackspace)
	if e.Buffer.String() != prefix+"e\u0301"+strings.Repeat("b", maxShapedLine+50)+"👨‍👩‍👧‍👦" {
		t.Fatal("surrogate at segment edge was corrupted")
	}
	tester.Key(ui.Cmd, ui.KeyZ)
	_ = e.SetPosition(0, segmentUnits+3)
	tester.Frame()
	tester.Key(0, ui.KeyBackspace)
	if strings.Contains(e.Buffer.String(), "e\u0301") {
		t.Fatal("combining grapheme at segment edge was split")
	}
	tester.Key(ui.Cmd, ui.KeyZ)
	if e.Buffer.String() != value || warnings != 1 {
		t.Fatalf("original/warning mismatch, warnings=%d", warnings)
	}
	for _, l := range e.cache {
		if text.UTF16Len(l.value) > segmentUnits+2 {
			t.Fatal("long line shaped past bounded segment")
		}
	}
}

func TestSegmentControlsSurrogateBoundaryAndWrappedNavigation(t *testing.T) {
	value := strings.Repeat("x", segmentUnits-1) + "🙂" + strings.Repeat("y ", maxShapedLine)
	e := New(text.New(value))
	e.SoftWrap = true
	tester := ui.NewTester(e.View, 600, 230)
	_ = tester.Click("文档编辑区")
	_ = e.SetPosition(0, 0)
	tester.Frame()
	if err := tester.Click("下一段"); err != nil {
		t.Fatal(err)
	}
	if !tester.Focused("文档编辑区") {
		t.Fatal("segment button did not return editing focus")
	}
	line, column, _ := e.GetPosition()
	segment := e.lineAt(line, column)
	if segment.windowStart == 0 || column != segmentUnits+1 {
		t.Fatalf("next segment at %d, window %d", column, segment.windowStart)
	}
	tester.Type("Z")
	if got, _ := e.Buffer.SliceUTF16(segmentUnits-1, segmentUnits+2); got != "🙂Z" {
		t.Fatalf("next segment edit %q", got)
	}
	tester.Key(ui.Cmd, ui.KeyZ)
	if err := tester.Click("上一段"); err != nil {
		t.Fatal(err)
	}
	if _, column, _ := e.GetPosition(); column != 0 {
		t.Fatalf("previous segment at %d", column)
	}
	_ = e.SetPosition(0, segmentUnits-2)
	tester.Frame()
	tester.Key(0, ui.KeyRight)
	tester.Key(0, ui.KeyRight)
	if _, column, _ := e.GetPosition(); column != segmentUnits+1 {
		t.Fatalf("right split surrogate: %d", column)
	}
	tester.Key(0, ui.KeyLeft)
	if _, column, _ := e.GetPosition(); column != segmentUnits-1 {
		t.Fatalf("left split surrogate: %d", column)
	}
	_ = e.SetPosition(0, segmentUnits-2)
	tester.Frame()
	tester.Key(0, ui.KeyDown)
	if _, column, _ := e.GetPosition(); column < segmentUnits+1 {
		t.Fatalf("Down did not cross segment edge: %d", column)
	}
}

func TestRowIndexAndCaretRemainCoherentAfterPrefixEditUndo(t *testing.T) {
	e := New(text.New(strings.Repeat("first long paragraph words words words ", 7) + "\n\nthird line\nfourth"))
	e.SoftWrap = true
	tester := ui.NewTester(e.View, 250, 170)
	_ = tester.Click("文档编辑区")
	for line := 0; line < e.Buffer.LineCount(); line++ {
		e.line(line)
	}
	_ = e.SetPosition(3, 4)
	tester.Frame()
	for _, operation := range []func(){func() { _ = e.Buffer.ReplaceUTF16(0, 0, "new\n") }, func() { e.Buffer.Undo() }, func() { e.Buffer.Redo() }} {
		operation()
		e.invalidateCacheIfChanged()
		base := 0
		for line := 0; line < e.Buffer.LineCount(); line++ {
			l := e.line(line)
			if got := e.visualBase(line); got != base {
				t.Fatalf("line %d visual base %d want %d", line, got, base)
			}
			for row := 0; row < l.rows; row++ {
				gotLine, gotRow := e.lineAndRowForVisual(base + row)
				if gotLine != line || gotRow != row {
					t.Fatalf("row index %d => %d:%d want %d:%d", base+row, gotLine, gotRow, line, row)
				}
			}
			base += l.rows
		}
		span, _ := e.Buffer.LineRange(e.Buffer.LineCount() - 1)
		assertCaretRoundTrip(t, e, tester, span.End())
	}
}

func TestIndentMappingDirectionEndpointAndGroupOwnership(t *testing.T) {
	for _, selection := range []text.Range{{Anchor: 0, Head: 6}, {Anchor: 6, Head: 0}} {
		e := New(text.New("a\nbbb\nc"))
		_ = e.Buffer.SetSelection(selection.Anchor, selection.Head)
		e.Command("indent")
		if e.Buffer.String() != "    a\n    bbb\nc" {
			t.Fatalf("line at selection end was indented: %q", e.Buffer.String())
		}
		want := text.Range{Anchor: 4, Head: 14}
		if selection.Anchor > selection.Head {
			want = text.Range{Anchor: 14, Head: 4}
		}
		if e.Buffer.Selection() != want || e.Buffer.History().UndoDepth != 1 {
			t.Fatal("mapped selection/group", e.Buffer.Selection(), e.Buffer.History())
		}
		e.Command("outdent")
		if e.Buffer.Selection() != selection || e.Buffer.String() != "a\nbbb\nc" {
			t.Fatal("outdent mapping", e.Buffer.Selection(), e.Buffer.String())
		}
		e.Buffer.Undo()
		e.Buffer.Undo()
		if e.Buffer.Selection() != selection || e.Buffer.String() != "a\nbbb\nc" {
			t.Fatal("two commands did not undo in two steps")
		}
	}
	for _, value := range []string{" ", "  ", "   ", "    ", "\tx", "  x", "x"} {
		e := New(text.New(value))
		_ = e.Buffer.SetSelection(0, 0)
		var reported error
		e.OnError = func(err error) { reported = err }
		e.HandleInput(ui.InputEvent{Kind: ui.InputKeyDown, Mods: ui.Shift, Key: ui.KeyTab})
		if e.Buffer.String() != strings.TrimLeft(value, " \t") || reported != nil || e.Buffer.InGroup() {
			t.Fatalf("caret outdent %q => %q, err=%v", value, e.Buffer.String(), reported)
		}
	}
	e := New(text.New("  own"))
	_ = e.Buffer.BeginGroup()
	e.Command("outdent")
	if !e.Buffer.InGroup() || e.Buffer.String() != "  own" {
		t.Fatal("indent operation stole another transaction")
	}
	e.Buffer.CancelGroup()
}

func TestEnterPreservesLastLineCRLFAndOnlyPrecedingIndent(t *testing.T) {
	for _, tc := range []struct {
		value string
		at    int
		want  string
	}{
		{"one\r\n    last", 13, "one\r\n    last\r\n    "},
		{"one\r\n    last", 7, "one\r\n  \r\n    last"},
		{"\t  text", 1, "\t\n\t  text"},
		{"one\r\n", 5, "one\r\n\r\n"},
	} {
		e := New(text.New(tc.value))
		_ = e.Buffer.SetSelection(tc.at, tc.at)
		e.insertNewlineWithIndent()
		if e.Buffer.String() != tc.want {
			t.Fatalf("Enter %q at %d => %q, want %q", tc.value, tc.at, e.Buffer.String(), tc.want)
		}
		if !e.Buffer.Undo() || e.Buffer.String() != tc.value {
			t.Fatal("newline was not one undo")
		}
	}
}

func TestNativeClipboardFailureAndDelayedCutNeverLoseText(t *testing.T) {
	e := New(text.New("retain"))
	tester := ui.NewTester(e.View, 400, 240)
	_ = tester.Click("文档编辑区")
	tester.Key(ui.Cmd, ui.KeyA)
	tester.FailClipboard(errors.New("clipboard busy"))
	var reported error
	e.OnError = func(err error) { reported = err }
	tester.Command("cut")
	if e.Buffer.String() != "retain" || reported == nil {
		t.Fatal("native failed cut lost text or failure")
	}
	tester.FailClipboard(nil)
	tester.Command("cut")
	if e.Buffer.String() != "" || tester.Clipboard() != "retain" {
		t.Fatal("confirmed native cut did not complete")
	}
	e.Buffer.Undo()
	_ = e.Buffer.SetSelection(0, 6)
	var complete func(error)
	e.CopyThen = func(_ string, done func(error)) { complete = done }
	e.Command("cut")
	e.ReplaceText(&ui.TextInputRange{Start: 6, End: 6}, "!")
	complete(nil)
	if e.Buffer.String() != "retain!" {
		t.Fatal("delayed cut removed newer text")
	}
}

func TestReleaseCompositionAndStyleDoNotLoseHistoryOrStealFocus(t *testing.T) {
	e := New(text.New("a"))
	show := true
	tester := ui.NewTester(func(c *ui.Context) {
		if show {
			e.View(c)
		}
		ui.Button(c, "Other")
	}, 400, 240)
	_ = tester.Click("文档编辑区")
	_ = e.SetPosition(0, 1)
	tester.Compose("中文", 2)
	e.SetFontSize(20)
	e.SetSoftWrap(true)
	tester.Frame()
	if !tester.Focused("文档编辑区") {
		t.Fatal("style change discarded current focus")
	}
	e.ReleaseView()
	show = false
	tester.Frame()
	if e.Buffer.InGroup() || len(e.cache) != 0 || len(e.rowTree) != 0 || e.focused {
		t.Fatal("inactive view retained composition or geometry")
	}
	if !e.Buffer.Undo() || e.Buffer.String() != "a" {
		t.Fatal("release lost composition undo")
	}
	_ = tester.Click("Other")
	e.SetFontSize(16)
	tester.Frame()
	if !tester.Focused("Other") {
		t.Fatal("inactive style change stole focus")
	}
}

func TestLayoutIsBoundedAndIdleFramesDoNotReshape(t *testing.T) {
	e := New(text.New(strings.Repeat("one two three 中文🙂 words words words\n", 100_000)))
	e.SoftWrap = true
	shapes, largest := 0, 0
	e.Highlight = func(s string) []ui.Span {
		shapes++
		largest = max(largest, text.UTF16Len(s))
		return []ui.Span{{Text: s}}
	}
	tester := ui.NewTester(e.View, 280, 180)
	_ = tester.Click("文档编辑区")
	if shapes > 12 || len(e.cache) > 12 {
		t.Fatalf("initial frame shaped %d paragraphs/cache %d", shapes, len(e.cache))
	}
	before := shapes
	for i := 0; i < 10; i++ {
		tester.Advance(time.Second)
		tester.Frame()
	}
	if shapes != before || tester.FramePending() {
		t.Fatalf("idle work: shapes %d => %d, pending %v", before, shapes, tester.FramePending())
	}
	_ = e.SetPosition(99_990, 5)
	tester.Frame()
	if shapes-before > 16 || len(e.rowTree) > 512 {
		t.Fatalf("far jump shaped the document: new shapes=%d sparse rows=%d", shapes-before, len(e.rowTree))
	}
	for i := 0; i < lineCacheLimit+20; i++ {
		e.line(i * 200)
	}
	if len(e.cache) > lineCacheLimit || largest > maxShapedLine+2 {
		t.Fatal("layout cache escaped its budget")
	}
	e.releaseGeometry()
	retained := e.line(0).layout
	_ = e.SetPosition(99_999, 2)
	e.ReplaceText(nil, "x")
	if got := e.line(0).layout; got != retained {
		t.Fatal("tail edit invalidated unaffected prefix")
	}
}

// Optional output used by the acceptance run; ordinary tests leave no files.
func TestRenderAcceptance(t *testing.T) {
	dir := os.Getenv("LEVNOTE_EDITOR_SCREENSHOT_DIR")
	if dir == "" {
		t.Skip("set LEVNOTE_EDITOR_SCREENSHOT_DIR to save acceptance PNGs")
	}
	e := New(text.New("原生编辑器 · UTF-16 / Unicode\r\n\r\n" + strings.Repeat("Wrapped text 中文🙂 e\u0301 — ", 16) + "\r\n\r\n    保留 CRLF 与缩进\r\n最后一行"))
	e.SoftWrap = true
	tester := ui.NewTester(e.View, 700, 420)
	_ = tester.Click("文档编辑区")
	_ = e.SetPosition(2, 37)
	tester.Compose("正在输入", 4)
	tester.Frame()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, "editor-wrap-ime.png"))
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, tester.Image()); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}
