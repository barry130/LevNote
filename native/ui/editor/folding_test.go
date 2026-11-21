package editor

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

func braceFolds(source string) []FoldRange {
	var stack []struct{ line, start, end int }
	var folds []FoldRange
	line, offset := 0, 0
	for _, part := range strings.SplitAfter(source, "\n") {
		for local, r := range part {
			if r == '{' {
				start := offset + text.UTF16Len(part[:local])
				stack = append(stack, struct{ line, start, end int }{line: line, start: start, end: start + 1})
			} else if r == '}' && len(stack) > 0 {
				open := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				close := offset + text.UTF16Len(part[:local])
				if line-open.line >= 2 {
					folds = append(folds, FoldRange{OpenLine: open.line, CloseLine: line, OpenStart: open.start, OpenEnd: open.end, CloseStart: close, CloseEnd: close + 1})
				}
			}
		}
		offset += text.UTF16Len(part)
		if strings.HasSuffix(part, "\n") {
			line++
		}
	}
	return folds
}

func installBraceProvider(e *Editor) error {
	return e.SetFoldProvider(func(snapshot FoldSnapshot) (FoldResult, error) {
		return FoldResult{Complete: true, Folds: braceFolds(snapshot.Source)}, nil
	}, 1, MaxFoldBytes)
}

func TestFoldIsViewOnlyNestedStateAndStaleTargets(t *testing.T) {
	value := "{\nouter\n{\ninner\n}\n}\ntail"
	b := text.New(value)
	e := New(b)
	_ = ui.NewTester(e.View, 480, 240)
	if err := installBraceProvider(e); err != nil {
		t.Fatal(err)
	}
	initial := e.FoldCandidates()
	if len(initial) != 2 {
		t.Fatalf("nested folds=%d: %+v", len(initial), initial)
	}
	version, history, selection := b.Version(), b.History(), b.Selection()
	changes := 0
	e.OnChange = func() { changes++ }
	e.ReadOnly = true
	if err := e.ToggleFold(initial[1].Target); err != nil {
		t.Fatal(err)
	}
	if err := e.SetPosition(0, 0); err != nil {
		t.Fatal(err)
	}
	if err := e.ToggleFoldAt(0); err != nil {
		t.Fatal(err)
	}
	if !e.FoldCandidates()[0].Collapsed || !e.FoldCandidates()[1].Collapsed {
		t.Fatal("outer fold lost the nested collapsed state")
	}
	if err := e.UnfoldAtPosition(0); err != nil {
		t.Fatal(err)
	}
	if !e.FoldCandidates()[0].Collapsed && !e.FoldCandidates()[1].Collapsed {
		t.Fatal("opening parent did not preserve child")
	}
	if b.Version() != version || b.History() != history || b.Selection() != selection || changes != 0 || b.String() != value {
		t.Fatal("folding changed document state")
	}
	old := e.FoldCandidates()[0].Target
	if err := b.ReplaceUTF16(b.LenUTF16(), b.LenUTF16(), "!"); err != nil {
		t.Fatal(err)
	}
	if err := b.Undo(); !err {
		t.Fatal("undo failed")
	}
	if err := e.ToggleFold(old); !errors.Is(err, ErrFoldStale) {
		t.Fatalf("old target accepted after same-byte undo: %v", err)
	}
	if e.folds.ready || len(e.FoldCandidates()) != 0 {
		t.Fatal("buffer version change did not expose full text")
	}
}

func TestFoldRefusesCaretSelectionMarkedAndInvalidProvider(t *testing.T) {
	value := "{\ninside\n}\noutside"
	for _, mode := range []string{"caret", "selection", "reversed-selection", "marked"} {
		t.Run(mode, func(t *testing.T) {
			b := text.New(value)
			e := New(b)
			tester := ui.NewTester(e.View, 480, 240)
			if err := installBraceProvider(e); err != nil {
				t.Fatal(err)
			}
			span, _ := b.LineRange(1)
			_ = b.SetSelection(0, 0)
			if mode == "caret" {
				_ = b.SetSelection(span.Start()+1, span.Start()+1)
			}
			if mode == "selection" {
				_ = b.SetSelection(span.Start(), span.End())
			}
			if mode == "reversed-selection" {
				_ = b.SetSelection(span.End(), span.Start())
			}
			if mode == "marked" {
				r := ui.TextInputRange{Start: span.Start(), End: span.End()}
				e.SetMarkedText(&r, "inside", ui.TextInputRange{Start: 6, End: 6})
				if err := e.RefreshFolds(); err != nil {
					t.Fatal(err)
				}
			}
			snapshot := b.Snapshot()
			generation, selections := b.SelectionGeneration(), b.SelectionSet()
			candidates := e.FoldCandidates()
			marked, composing := e.MarkedRange()
			changes := 0
			e.OnChange = func() { changes++ }
			version, history, selection := b.Version(), b.History(), b.Selection()
			if err := e.ToggleFoldAt(0); err == nil {
				t.Fatal("fold intersecting hidden state accepted")
			}
			if e.FoldCandidates()[0].Collapsed || b.Version() != version || b.History() != history || b.Selection() != selection {
				t.Fatal("rejected fold changed source/selection/history")
			}
			gotMarked, stillComposing := e.MarkedRange()
			if !b.EqualSnapshot(snapshot) || b.String() != snapshot.String() || b.SelectionGeneration() != generation ||
				!reflect.DeepEqual(b.SelectionSet(), selections) || !reflect.DeepEqual(e.FoldCandidates(), candidates) ||
				gotMarked != marked || stillComposing != composing || changes != 0 {
				t.Fatal("fold refusal changed exact source, directed selections, candidates or composition")
			}
			if mode == "marked" {
				if _, ok := e.MarkedRange(); !ok {
					t.Fatal("fold refusal ended composition")
				}
				e.cancelComposition()
			}
			tester.Frame()
		})
	}
}

func TestFoldProviderBudgetFailureAndDefaultOff(t *testing.T) {
	b := text.New("{\nx\n}")
	e := New(b)
	calls, reports := 0, 0
	e.OnError = func(error) { reports++ }
	_ = ui.NewTester(e.View, 420, 180)
	if err := e.ToggleFoldAt(0); err == nil {
		t.Fatal("disabled editor accepted folds")
	}
	if calls != 0 {
		t.Fatal("disabled editor called provider")
	}
	provider := func(snapshot FoldSnapshot) (FoldResult, error) {
		calls++
		return FoldResult{Complete: true, Folds: braceFolds(snapshot.Source)}, nil
	}
	if err := e.SetFoldProvider(provider, 1, 0); err == nil {
		t.Fatal("nonpositive budget accepted")
	}
	if calls != 0 || len(e.FoldCandidates()) != 0 || !strings.Contains(e.folds.status, "预算") {
		t.Fatal("budget failure was not explicit and full-visible")
	}
	if err := e.RefreshFolds(); err == nil {
		t.Fatal("zero budget refresh accepted")
	}
	if reports != 1 {
		t.Fatalf("budget signature reported %d times", reports)
	}
	if err := e.SetFoldProvider(func(FoldSnapshot) (FoldResult, error) { return FoldResult{Complete: true, Unsupported: true}, nil }, 2, MaxFoldBytes); err == nil {
		t.Fatal("unsupported result accepted")
	}
	if len(e.FoldCandidates()) != 0 || !strings.Contains(e.folds.status, "不支持") {
		t.Fatal("unsupported status missing")
	}
}

func TestFoldProviderRejectsWholeInvalidResultAndDoesNotAnalyzeInFrames(t *testing.T) {
	value := "🙂{\ninside\n}\nafter"
	b := text.New(value)
	e := New(b)
	tester := ui.NewTester(e.View, 480, 240)
	valid := braceFolds(value)[0]
	bad := valid
	bad.OpenStart = 1 // splits the original emoji surrogate pair
	calls := 0
	result := FoldResult{Complete: true, Folds: []FoldRange{valid}}
	if err := e.SetFoldProvider(func(snapshot FoldSnapshot) (FoldResult, error) {
		calls++
		if snapshot.Source != value || snapshot.Version != b.Version() || snapshot.LineCount != 4 {
			t.Fatalf("bad snapshot %v", snapshot)
		}
		return result, nil
	}, 1, MaxFoldBytes); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		tester.Frame()
	}
	if calls != 1 {
		t.Fatalf("paint invoked provider %d times", calls)
	}
	if err := e.ToggleFoldAt(0); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []FoldResult{
		{Complete: true, Folds: []FoldRange{valid, bad}},
		{Complete: true, Folds: []FoldRange{valid, valid}},
		{Complete: true, Folds: make([]FoldRange, MaxFoldCandidates+1)},
		{Complete: true, Diagnostics: 1, Folds: []FoldRange{valid}},
		{Folds: []FoldRange{valid}},
	} {
		result = invalid
		if err := e.RefreshFolds(); err == nil || len(e.FoldCandidates()) != 0 || e.totalVisualRows() != 4 {
			t.Fatalf("published partial result, ready=%v rows=%d err=%v", e.folds.ready, e.totalVisualRows(), err)
		}
	}
	if err := b.ReplaceUTF16(b.LenUTF16(), b.LenUTF16(), "!"); err != nil {
		t.Fatal(err)
	}
	before := calls
	tester.Frame()
	if calls != before || !tester.HasText("暂时显示全文") {
		t.Fatal("edit should display whole source and wait for explicit refresh")
	}
}

func TestFoldProjectionMatchesBruteForceWithCachedWrapRows(t *testing.T) {
	value := "{\n0123456789 0123456789 0123456789\n{\n0123456789 0123456789\n}\n0123456789 0123456789\n}\nafter"
	b := text.New(value)
	e := New(b)
	e.SoftWrap = true
	tester := ui.NewTester(e.View, 190, 260)
	if err := installBraceProvider(e); err != nil {
		t.Fatal(err)
	}
	for line := 0; line < b.LineCount(); line++ {
		e.line(line)
	}
	rows := make([]int, b.LineCount())
	for line := range rows {
		rows[line] = max(1, e.rowCounts[line])
	}
	if err := b.SetSelection(0, 0); err != nil {
		t.Fatal(err)
	}
	if err := e.ToggleFoldAt(2); err != nil {
		t.Fatal(err)
	}
	if err := e.ToggleFoldAt(0); err != nil {
		t.Fatal(err)
	}
	hidden := make([]bool, b.LineCount())
	for _, interval := range e.folds.intervals {
		for line := interval.start; line < interval.end; line++ {
			hidden[line] = true
		}
	}
	want := 0
	for line := 0; line < b.LineCount(); line++ {
		if hidden[line] {
			continue
		}
		if got := e.visualBase(line); got != want {
			t.Fatalf("P(%d)=%d want %d", line, got, want)
		}
		want += rows[line]
	}
	if e.totalVisualRows() != want {
		t.Fatalf("total=%d want %d", e.totalVisualRows(), want)
	}
	if err := e.UnfoldAll(); err != nil {
		t.Fatal(err)
	}
	for line := 0; line < b.LineCount(); line++ {
		if got := e.visualBase(line); got != sumRows(rows, line) {
			t.Fatalf("unfold P(%d)=%d", line, got)
		}
	}
	tester.Frame()
}
func sumRows(rows []int, end int) int {
	n := 0
	for i := 0; i < end; i++ {
		n += rows[i]
	}
	return n
}

func TestFoldDiagnosticsKeepAnalysisSourceAndFullVisibility(t *testing.T) {
	const source = "{\r\ninside🙂\n}\r\nafter"
	buffer := text.New(source)
	e := New(buffer)
	tester := ui.NewTester(e.View, 600, 260)
	result := FoldResult{Complete: true, Folds: braceFolds(source)}
	var reported []error
	e.OnError = func(err error) { reported = append(reported, err) }
	if err := e.SetFoldProvider(func(FoldSnapshot) (FoldResult, error) { return result, nil }, 1, MaxFoldBytes); err != nil {
		t.Fatal(err)
	}
	if err := buffer.SetSelection(buffer.LenUTF16(), buffer.LenUTF16()-2); err != nil {
		t.Fatal(err)
	}
	if err := e.ToggleFoldAt(0); err != nil {
		t.Fatal(err)
	}
	if !e.FoldCandidates()[0].Collapsed {
		t.Fatal("diagnostic fixture did not start with a collapsed block")
	}
	snapshot := buffer.Snapshot()
	version, history := buffer.Version(), buffer.History()
	generation, selections := buffer.SelectionGeneration(), buffer.SelectionSet()
	changes := 0
	e.OnChange = func() { changes++ }
	result.Diagnostics = 2
	for attempt := 0; attempt < 2; attempt++ {
		err := e.RefreshFolds()
		if err == nil || !strings.Contains(err.Error(), "折叠分析") ||
			!strings.Contains(err.Error(), "文档存在结构诊断；暂时显示全文") {
			t.Fatalf("diagnostic source or original explanation missing: %v", err)
		}
		tester.Frame()
		if !tester.HasText("折叠分析") || !tester.HasText("文档存在结构诊断；暂时显示全文") ||
			len(e.FoldCandidates()) != 0 || e.totalVisualRows() != buffer.LineCount() {
			t.Fatalf("diagnostics did not leave persistent explanation and full source: %v", tester.Texts())
		}
		if buffer.String() != source || !buffer.EqualSnapshot(snapshot) || buffer.Version() != version ||
			buffer.History() != history || buffer.SelectionGeneration() != generation ||
			!reflect.DeepEqual(buffer.SelectionSet(), selections) || changes != 0 {
			t.Fatal("diagnostic publication changed source, directed selections or history")
		}
	}
	if len(reported) != 1 || !strings.Contains(reported[0].Error(), "文档存在结构诊断；暂时显示全文") {
		t.Fatalf("diagnostic was repeated or its original explanation was lost: %v", reported)
	}
}
