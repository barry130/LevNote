package editor

import (
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

func BenchmarkLargeEditorCachedCaret(b *testing.B) {
	e := New(text.New(strings.Repeat("A line of 中文🙂 text to edit.\n", 100_000)))
	e.SoftWrap = true
	e.width, e.height = 640, 480
	_ = e.SetPosition(99_990, 10)
	offset := e.Buffer.Selection().Head
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.BoundsForRange(ui.TextInputRange{Start: offset, End: offset})
	}
}

func BenchmarkLongLineTailInput(b *testing.B) {
	e := New(text.New(strings.Repeat("a", 2<<20)))
	e.width, e.height = 640, 480
	_ = e.SetPosition(0, e.Buffer.LenUTF16())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e.ReplaceText(nil, "x")
		e.Command("undo")
	}
}

func BenchmarkLargeEditorIdleFrame(b *testing.B) {
	e := New(text.New(strings.Repeat("A line of 中文🙂 text.\n", 100_000)))
	e.SoftWrap = true
	tester := ui.NewTester(e.View, 640, 480)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tester.Frame()
	}
}
