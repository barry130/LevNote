package editor

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

// wordSelectionPoint uses only the editor's public native geometry queries.
// The offset is a literal test premise, not an expected word boundary.
func wordSelectionPoint(t *testing.T, e *Editor, tester *ui.Tester, offset int) ui.Point {
	t.Helper()
	r, actual, visible := e.BoundsForRange(ui.TextInputRange{Start: offset, End: offset})
	if !visible || actual != (ui.TextInputRange{Start: offset, End: offset}) {
		t.Fatalf("literal caret %d unavailable: rect=%+v actual=%+v visible=%v", offset, r, actual, visible)
	}
	local := ui.Point{X: r.X, Y: r.Y + r.H/2}
	if index, ok := e.IndexForPoint(local); !ok || index != offset {
		t.Fatalf("literal caret %d maps to %d, ok=%v", offset, index, ok)
	}
	box, ok := tester.Find("文档编辑区")
	if !ok {
		t.Fatal("built-in editor missing")
	}
	return ui.Point{X: box.X + local.X, Y: box.Y + local.Y}
}

func TestDoubleClickWordKeepsCRLFBoundaries(t *testing.T) {
	e := New(text.New("one\r\ntwo\r\nthree"))
	tester := ui.NewTester(e.View, 600, 300)
	point := wordSelectionPoint(t, e, tester, 6)
	// The backend counts these two ordinary pointer clicks itself.
	tester.ClickAt(point.X, point.Y)
	tester.ClickAt(point.X, point.Y)
	if got := e.Selection(); got.Range != (ui.TextInputRange{Start: 5, End: 8}) || got.Reversed {
		t.Fatalf("double-click selected %+v; want literal two at [5,8)", got)
	}
	if got, actual := e.TextForRange(e.Selection().Range); got != "two" || actual != (ui.TextInputRange{Start: 5, End: 8}) {
		t.Fatalf("double-click text=%q actual=%+v; want literal two at [5,8)", got, actual)
	}
}

func TestDoubleClickWordReplacementUndoRedoKeepsCRLF(t *testing.T) {
	e := New(text.New("one\r\ntwo\r\nthree"))
	tester := ui.NewTester(e.View, 600, 300)
	point := wordSelectionPoint(t, e, tester, 6)
	tester.ClickAt(point.X, point.Y)
	tester.ClickAt(point.X, point.Y)
	tester.Type("X")
	if got := e.Buffer.String(); got != "one\r\nX\r\nthree" {
		t.Fatalf("replacement=%q; want literal CRLF on both sides", got)
	}
	if got := e.Selection(); got.Range != (ui.TextInputRange{Start: 6, End: 6}) || got.Reversed {
		t.Fatalf("replacement caret=%+v; want literal [6,6)", got)
	}
	tester.Key(ui.Cmd, ui.KeyZ)
	if got := e.Buffer.String(); got != "one\r\ntwo\r\nthree" {
		t.Fatalf("undo=%q; want original literal", got)
	}
	if got := e.Selection(); got.Range != (ui.TextInputRange{Start: 5, End: 8}) || got.Reversed {
		t.Fatalf("undo selection=%+v; want literal [5,8)", got)
	}
	tester.Key(ui.Cmd, ui.KeyY)
	if got := e.Buffer.String(); got != "one\r\nX\r\nthree" {
		t.Fatalf("redo=%q; want literal CRLF on both sides", got)
	}
	if got := e.Selection(); got.Range != (ui.TextInputRange{Start: 6, End: 6}) || got.Reversed {
		t.Fatalf("redo caret=%+v; want literal [6,6)", got)
	}
}

func TestDoubleClickWordKeepsLFBoundaries(t *testing.T) {
	e := New(text.New("one\ntwo\nthree"))
	tester := ui.NewTester(e.View, 600, 300)
	point := wordSelectionPoint(t, e, tester, 5)
	tester.ClickAt(point.X, point.Y)
	tester.ClickAt(point.X, point.Y)
	if got := e.Selection(); got.Range != (ui.TextInputRange{Start: 4, End: 7}) || got.Reversed {
		t.Fatalf("LF word selection=%+v; want literal [4,7)", got)
	}
	tester.Type("X")
	if got := e.Buffer.String(); got != "one\nX\nthree" {
		t.Fatalf("LF replacement=%q; want literal surrounding lines", got)
	}
}

func TestDoubleClickWordKeepsSpaces(t *testing.T) {
	e := New(text.New("one two three"))
	tester := ui.NewTester(e.View, 600, 300)
	point := wordSelectionPoint(t, e, tester, 5)
	tester.ClickAt(point.X, point.Y)
	tester.ClickAt(point.X, point.Y)
	if got := e.Selection(); got.Range != (ui.TextInputRange{Start: 4, End: 7}) || got.Reversed {
		t.Fatalf("space word selection=%+v; want literal [4,7)", got)
	}
	tester.Type("X")
	if got := e.Buffer.String(); got != "one X three" {
		t.Fatalf("space replacement=%q; want literal surrounding spaces", got)
	}
}

func TestDoubleClickWordKeepsPunctuation(t *testing.T) {
	e := New(text.New("one,(two);three"))
	tester := ui.NewTester(e.View, 600, 300)
	point := wordSelectionPoint(t, e, tester, 6)
	tester.ClickAt(point.X, point.Y)
	tester.ClickAt(point.X, point.Y)
	if got := e.Selection(); got.Range != (ui.TextInputRange{Start: 5, End: 8}) || got.Reversed {
		t.Fatalf("punctuation word selection=%+v; want literal [5,8)", got)
	}
	tester.Type("X")
	if got := e.Buffer.String(); got != "one,(X);three" {
		t.Fatalf("punctuation replacement=%q; want literal surrounding delimiters", got)
	}
}

func TestDoubleClickWordKeepsEmojiAndCombiningGraphemes(t *testing.T) {
	e := New(text.New("one a🙂e\u0301z three"))
	tester := ui.NewTester(e.View, 600, 300)
	point := wordSelectionPoint(t, e, tester, 7)
	tester.ClickAt(point.X, point.Y)
	tester.ClickAt(point.X, point.Y)
	if got := e.Selection(); got.Range != (ui.TextInputRange{Start: 4, End: 10}) || got.Reversed {
		t.Fatalf("Unicode word selection=%+v; want literal UTF-16 [4,10)", got)
	}
	if got, actual := e.TextForRange(e.Selection().Range); got != "a🙂e\u0301z" || actual != (ui.TextInputRange{Start: 4, End: 10}) {
		t.Fatalf("Unicode word text=%q actual=%+v; want literal complete graphemes", got, actual)
	}
	tester.Type("X")
	if got := e.Buffer.String(); got != "one X three" {
		t.Fatalf("Unicode replacement=%q; want literal surrounding spaces", got)
	}
}

func TestDoubleClickDelimiterAndWordEdgeRetainSelection(t *testing.T) {
	for _, tc := range []struct {
		name   string
		value  string
		offset int
		want   ui.TextInputRange
	}{
		{"between-spaces", "one  two", 4, ui.TextInputRange{Start: 3, End: 5}},
		{"word-edge", "one two three", 4, ui.TextInputRange{Start: 3, End: 8}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := New(text.New(tc.value))
			tester := ui.NewTester(e.View, 600, 300)
			point := wordSelectionPoint(t, e, tester, tc.offset)
			tester.ClickAt(point.X, point.Y)
			tester.ClickAt(point.X, point.Y)
			if got := e.Selection(); got.Range != tc.want || got.Reversed {
				t.Fatalf("compatible selection=%+v; want literal %+v", got, tc.want)
			}
			if got := e.Buffer.String(); got != tc.value {
				t.Fatalf("selection changed text=%q; want %q", got, tc.value)
			}
		})
	}
}

func TestThirdClickRetainsExistingWordSelection(t *testing.T) {
	e := New(text.New("one\r\ntwo\r\nthree"))
	tester := ui.NewTester(e.View, 600, 300)
	point := wordSelectionPoint(t, e, tester, 6)
	tester.ClickAt(point.X, point.Y)
	tester.ClickAt(point.X, point.Y)
	tester.ClickAt(point.X, point.Y)
	if got := e.Selection(); got.Range != (ui.TextInputRange{Start: 3, End: 10}) || got.Reversed {
		t.Fatalf("third-click selection=%+v; want existing literal [3,10)", got)
	}
	if got := e.Buffer.String(); got != "one\r\ntwo\r\nthree" {
		t.Fatalf("third-click changed text=%q", got)
	}
}
