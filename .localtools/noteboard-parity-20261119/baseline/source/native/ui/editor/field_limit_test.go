package editor

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

func TestSourceLongUnicodeResourceProbe(t *testing.T) {
	value := strings.Repeat("中🙂", 10000)
	for _, width := range []int{800, 265} {
		for _, wrap := range []bool{false, true} {
			t.Run(fmt.Sprintf("width%d-wrap%v", width, wrap), func(t *testing.T) {
				e := New(text.New(value))
				e.SoftWrap = wrap
				runtime.GC()
				var before, after runtime.MemStats
				runtime.ReadMemStats(&before)
				start := time.Now()
				tester := ui.NewTester(e.View, width, 480)
				elapsed := time.Since(start)
				runtime.ReadMemStats(&after)
				t.Logf("source cold %s %d B %d allocs limit=%d pending=%v", elapsed, after.TotalAlloc-before.TotalAlloc, after.Mallocs-before.Mallocs, e.shapingLimit(), tester.FramePending())
				if e.Buffer.String() != value {
					t.Fatal("source input changed")
				}
			})
		}
	}
}

func TestSmallFieldShapeLimitInvalidatesWithoutLosingFocusOrTail(t *testing.T) {
	value := strings.Repeat("a", 511) + "🙂" + strings.Repeat("b", 2000)
	e := New(text.New(value))
	e.LineNumbers, e.MinHeight = false, 48
	tester := ui.NewTester(e.View, 220, 180)
	if err := tester.Click(e.Label); err != nil {
		t.Fatal(err)
	}
	e.ShapeLimit = 512
	tester.Frame()
	if len(e.lineAt(0, 0).value) > 516 || !e.focused {
		t.Fatal("lower field limit failed to invalidate existing geometry/focus")
	}
	if err := tester.Click("下一段"); err != nil {
		t.Fatal(err)
	}
	if _, column, _ := e.GetPosition(); column < 513 {
		t.Fatal("segment boundary split/lost surrogate", column)
	}
	_ = e.SetPosition(0, e.Buffer.LenUTF16())
	tester.Frame()
	tester.Type("尾")
	if e.Buffer.String() != value+"尾" {
		t.Fatal("small field could not edit tail")
	}
	tester.Key(ui.Cmd, ui.KeyZ)
	if e.Buffer.String() != value {
		t.Fatal("small field undo lost original")
	}
}
