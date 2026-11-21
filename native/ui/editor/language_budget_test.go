package editor

import (
	"errors"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

// These fixtures visit real wrapped lines through the ordinary layout path.
// Both visit counts fill the same bounded shape cache. Only retained row-index
// history differs; the full seed is under 64 KiB, not a large-document stress.
func languageBudgetFixture(t *testing.T, visited int, transition, trigger string) (*Editor, SourceTarget, int, *int) {
	t.Helper()
	const lines = 960
	body := strings.Repeat("word ", 8) + "\n"
	b := text.NewWithLimit(strings.Repeat(body, lines), 1024)
	e := New(b)
	e.SoftWrap = true
	_ = ui.NewTester(e.View, 220, 150)
	for line := lines - visited; line < lines; line++ {
		e.line(line)
	}
	if len(e.rowCounts) < visited || len(e.cache) != lineCacheLimit {
		t.Fatal("fixture did not retain visited wrapped geometry", len(e.rowCounts), len(e.cache))
	}
	current := "x"
	if trigger == "long-error" {
		current = strings.Repeat("a", maxShapedLine+1)
	}
	switch transition {
	case "swap":
		e.SetBuffer(text.New(current))
	case "shrink":
		if err := b.ReplaceUTF16(0, b.LenUTF16(), current); err != nil {
			t.Fatal(err)
		}
	case "compatible":
	default:
		t.Fatal("unknown transition")
	}
	end := 1
	if transition == "compatible" && trigger == "warm" {
		end, _ = b.OffsetOfLine(lines - 1)
	}
	if err := e.Buffer.SetSelection(end, end); err != nil {
		t.Fatal(err)
	}
	target := captureLanguageTarget(t, e)
	// Inspect raw invocation state at each callback without copying or scanning
	// retained maps inside the allocation measurement. The comprehensive raw
	// entry/exit equality oracles remain in language_reentry_test.go.
	cached, version, indexed := e.cachedBuffer, e.version, e.indexedLines
	rows, tree, cache := len(e.rowCounts), len(e.rowTree), len(e.cache)
	lastRows, scroll, generation := e.rowCounts[lines-1], e.ScrollY, e.geometryGeneration
	selection, selectionGen := e.Buffer.Selection(), e.Buffer.SelectionGeneration()
	calls := new(int)
	observe := func() {
		*calls++
		if e.cachedBuffer != cached || e.version != version || e.indexedLines != indexed || len(e.rowCounts) != rows || len(e.rowTree) != tree || len(e.cache) != cache || e.rowCounts[lines-1] != lastRows || e.ScrollY != scroll || e.geometryGeneration != generation || e.Buffer.Selection() != selection || e.Buffer.SelectionGeneration() != selectionGen {
			t.Error("callback observed provisional cleanup or geometry")
		}
	}
	e.Highlight = func(value string) []ui.Span {
		observe()
		return []ui.Span{{Text: value}}
	}
	e.OnError = func(err error) {
		if !errors.Is(err, errLineTooLong) {
			t.Error(err)
		}
		observe()
	}
	return e, target, end, calls
}

// This oracle measures allocated bytes, never elapsed time. Map scans without
// allocation are excluded separately by the row-mutation epoch audit.
func TestLanguageBudgetRetainedRows(t *testing.T) {
	for _, transition := range []string{"swap", "shrink", "compatible"} {
		triggers := []string{"highlight", "long-error"}
		if transition == "compatible" {
			triggers = []string{"highlight", "warm"}
		}
		for _, trigger := range triggers {
			t.Run(transition+"/"+trigger, func(t *testing.T) {
				var allocated [2]uint64
				for i, visited := range []int{320, 960} {
					e, target, end, calls := languageBudgetFixture(t, visited, transition, trigger)
					runtime.GC()
					var before, after runtime.MemStats
					runtime.ReadMemStats(&before)
					err := e.RevealSourceTarget(target, end, end)
					runtime.ReadMemStats(&after)
					allocated[i] = after.TotalAlloc - before.TotalAlloc
					if err != nil {
						t.Fatal(err)
					}
					wantCalls := 1
					if trigger == "long-error" {
						wantCalls = 2
					} else if trigger == "warm" {
						wantCalls = 0
					}
					if *calls != wantCalls || e.Buffer.Selection() != (text.Range{Anchor: end, Head: end}) {
						t.Fatal("navigation skipped callbacks or selection", *calls, wantCalls)
					}
				}
				t.Logf("retained rows 320 -> 960: allocated %d -> %d bytes", allocated[0], allocated[1])
				// Fixed cache/shape/callback costs are identical. A 16 KiB
				// allowance covers allocator size classes; cloning/scanning the
				// extra row-map domain must not enter this operation's budget.
				if allocated[1] > allocated[0]+16<<10 {
					t.Fatalf("navigation allocation grew with retained row history: %d -> %d bytes", allocated[0], allocated[1])
				}
			})
		}
	}
}

func TestLanguageBudgetPrefixWrapAndFolds(t *testing.T) {
	for _, shrink := range []bool{false, true} {
		for _, warm := range []bool{false, true} {
			name := map[bool]string{false: "compatible", true: "shrink"}[shrink] + "/" + map[bool]string{false: "cold", true: "warm"}[warm]
			t.Run(name, func(t *testing.T) {
				body := strings.Repeat("word ", 12)
				prefix := body + "\r\n{\r\n" + body + "\r\n}\r\n{\r\n" + body + "\r\n}\r\n" + body + "\r\n"
				b := text.New(prefix + strings.Repeat(body+"\n", 320))
				e := New(b)
				e.SoftWrap = true
				_ = ui.NewTester(e.View, 240, 160)
				if err := installBraceProvider(e); err != nil {
					t.Fatal(err)
				}
				// Installing a provider changes the gutter. Retain rows only
				// after that ordinary geometry reset has completed.
				for line := 0; line < b.LineCount(); line++ {
					e.line(line)
				}
				for _, line := range []int{1, 4} {
					if err := e.ToggleFoldAt(line); err != nil {
						t.Fatal(err)
					}
				}
				// The target is outside the collapsed blocks, so a warm cache can
				// survive while hidden wrapped rows still require subtraction.
				targetStart, _ := b.OffsetOfLine(7)
				if warm {
					e.line(7)
				}
				wantRows := make([]int, 8)
				for line := range wantRows {
					wantRows[line] = max(1, e.rowCounts[line])
				}
				if shrink {
					if err := b.ReplaceUTF16(text.UTF16Len(prefix), b.LenUTF16(), "end"); err != nil {
						t.Fatal(err)
					}
				}
				target := captureLanguageTarget(t, e)
				setSelections(t, b, []text.Range{{Anchor: 5, Head: 1}, {Anchor: targetStart + 4, Head: targetStart + 2}}, 1)
				if !warm {
					font := ui.Font{Family: "monospace", Size: e.FontSize, LineHeight: 1.65}
					wantRows[7] = len(ui.ShapeRichText([]ui.Span{{Text: body, Size: 21}}, font, e.lineWidth()).Lines())
				}
				entry, before := captureSourceEditorRejectState(e), captureSourceRejectState(b)
				calls := 0
				e.Highlight = func(value string) []ui.Span {
					calls++
					if !reflect.DeepEqual(entry, captureSourceEditorRejectState(e)) || !reflect.DeepEqual(before, captureSourceRejectState(b)) {
						t.Error("prefix preparation changed raw callback entry")
					}
					return []ui.Span{{Text: value, Size: 21}}
				}
				if err := e.RevealSourceTarget(target, targetStart+5, targetStart+2); err != nil {
					t.Fatal(err)
				}
				if calls != map[bool]int{false: 1, true: 0}[warm] {
					t.Fatal("warm/cold target cache changed", calls)
				}
				for line, count := range wantRows {
					if max(1, e.rowCounts[line]) != count {
						t.Fatal("valid prefix geometry was discarded", line, count, e.rowCounts[line])
					}
				}
				// Sum actual current line rows directly, independently of Fenwick
				// and detached overlay math, including hidden wrapped row removal.
				wrap, visible := 0, 0
				for line := 0; line <= b.LineCount(); line++ {
					if rowPrefix(e.rowTree, line) != wrap || e.visualBase(line) != visible {
						t.Fatal("prefix/Fenwick/fold geometry differs", line, rowPrefix(e.rowTree, line), wrap, e.visualBase(line), visible)
					}
					count := max(1, e.rowCounts[line])
					wrap += count - 1
					hidden := false
					for _, r := range e.folds.ranges {
						hidden = hidden || e.folds.collapsed[r] && r.OpenLine < line && line < r.CloseLine
					}
					if !hidden {
						visible += count
					}
				}
				if shrink && (len(e.folds.collapsed) != 0 || len(e.rowCounts) > 9 || len(e.rowTree) > 9) {
					t.Fatal("obsolete suffix or version-stale fold projection survived")
				}
				if !shrink && len(e.folds.collapsed) != 2 {
					t.Fatal("unrelated collapsed blocks changed")
				}
				if b.Version() != before.edit.version || b.History() != before.edit.history || b.String() != before.edit.source || b.Selection() != (text.Range{Anchor: targetStart + 5, Head: targetStart + 2}) {
					t.Fatal("prefix navigation changed source/history/directed selection")
				}
			})
		}
	}
}

func TestLanguageBudgetStaleRetainedBase(t *testing.T) {
	for _, transition := range []string{"swap", "shrink", "compatible"} {
		for _, trigger := range []string{"highlight", "long-error"} {
			if transition == "compatible" && trigger == "long-error" {
				continue // The compatible fixture has no long physical line.
			}
			t.Run(transition+"/"+trigger, func(t *testing.T) {
				e, target, end, _ := languageBudgetFixture(t, 960, transition, trigger)
				entry, before := captureSourceEditorRejectState(e), captureSourceRejectState(e.Buffer)
				var exit []any
				mutated := false
				mutate := func() {
					if mutated {
						t.Fatal("continued callbacks after stale boundary")
					}
					mutated = true
					if !reflect.DeepEqual(entry, captureSourceEditorRejectState(e)) || !reflect.DeepEqual(before, captureSourceRejectState(e.Buffer)) {
						t.Fatal("callback saw stale-base repair")
					}
					// Advance and restore the actual row tree. Only an epoch can
					// detect this ABA; the maps' final values compare equal.
					e.addRows(0, 1)
					e.addRows(0, -1)
					exit = captureSourceEditorRejectState(e)
				}
				e.Highlight = func(value string) []ui.Span {
					if trigger == "highlight" {
						mutate()
					}
					return []ui.Span{{Text: value}}
				}
				e.OnError = func(err error) {
					if !errors.Is(err, errLineTooLong) {
						t.Error(err)
					}
					mutate()
				}
				if err := e.RevealSourceTarget(target, end, end); !errors.Is(err, errSourceTargetStale) || !mutated {
					t.Fatal("retained-base callback ABA was not rejected", err)
				}
				if !reflect.DeepEqual(exit, captureSourceEditorRejectState(e)) {
					t.Fatal("stale plan overwrote callback-owned raw exit")
				}
				requireSourceRejectState(t, e.Buffer, before)
			})
		}
	}
}
