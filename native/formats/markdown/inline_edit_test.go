package markdown

import (
	"strings"
	"testing"
)

func TestInlineEditWrapToggleAndByteFidelity(t *testing.T) {
	for _, tc := range []struct {
		name   string
		kind   InlineKind
		marker string
	}{
		{"bold", InlineBold, "**"}, {"italic", InlineItalic, "*"}, {"code", InlineCode, "`"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prefix, selected, suffix := "前🙂 ", "中文🙂", " 后\r\n<!-- 未知 x=1 -->\r\n"
			source := prefix + selected + suffix
			plan, err := PlanInlineEdit(source, rangeAt(len(prefix), len(prefix)+len(selected)), tc.kind)
			if err != nil {
				t.Fatal(err)
			}
			got := source[:plan.Range.Start] + plan.Text + source[plan.Range.End:]
			want := prefix + tc.marker + selected + tc.marker + suffix
			if got != want || got[plan.Selection.Start:plan.Selection.End] != selected {
				t.Fatalf("wrap = %q, plan %#v", got, plan)
			}
			spans, _ := inlineEditSpans(got)
			if len(spans) != 1 || spans[0].Kind != tc.kind || spans[0].Text != selected {
				t.Fatal("production parser disagrees", spans)
			}
			undo, err := PlanInlineEdit(got, plan.Selection, tc.kind)
			if err != nil {
				t.Fatal(err)
			}
			if got[:undo.Range.Start]+undo.Text+got[undo.Range.End:] != source {
				t.Fatal("second toggle changed untouched bytes")
			}
			whole, err := PlanInlineEdit(got, rangeAt(len(prefix), len(prefix)+len(tc.marker)*2+len(selected)), tc.kind)
			if err != nil || whole != undo {
				t.Fatalf("whole mark toggle = %#v, %v", whole, err)
			}
		})
	}
}

func TestInlineEditEmptyCaretAndExistingAlternateMarks(t *testing.T) {
	for _, tc := range []struct {
		kind   InlineKind
		marker string
	}{
		{InlineBold, "**"}, {InlineItalic, "*"}, {InlineCode, "`"},
	} {
		prefix, suffix := "前🙂 ", " 后"
		plan, err := PlanInlineEdit(prefix+suffix, rangeAt(len(prefix), len(prefix)), tc.kind)
		if err != nil {
			t.Fatal(err)
		}
		got := prefix + plan.Text + suffix
		p := len(prefix) + len(tc.marker)
		if plan.Selection != rangeAt(p, p) || plan.Text != tc.marker+tc.marker {
			t.Fatal("empty insertion lost caret", plan)
		}
		cancel, err := PlanInlineEdit(got, plan.Selection, tc.kind)
		if err != nil || got[:cancel.Range.Start]+cancel.Text+got[cancel.Range.End:] != prefix+suffix {
			t.Fatal("empty toggle", cancel, err)
		}
	}
	for _, tc := range []struct {
		source string
		kind   InlineKind
		start  int
		want   string
		caret  int
	}{
		{"__中文__", InlineBold, 5, "中文", 3},
		{"_中文_", InlineItalic, 4, "中文", 3},
		{"``a`b``", InlineCode, 3, "a`b", 1},
	} {
		plan, err := PlanInlineEdit(tc.source, rangeAt(tc.start, tc.start), tc.kind)
		if err != nil || plan.Text != tc.want || plan.Selection != rangeAt(tc.caret, tc.caret) {
			t.Fatalf("existing caret %q: %#v %v", tc.source, plan, err)
		}
	}
}

func TestInlineEditCodeDelimiterAndExplicitBoundaries(t *testing.T) {
	for _, source := range []string{"a`b", "a``b", "a b", " a", "a "} {
		plan, err := PlanInlineEdit(source, rangeAt(0, len(source)), InlineCode)
		if err != nil {
			t.Fatalf("plain code %q: %v", source, err)
		}
		spans, _ := inlineEditSpans(plan.Text)
		if len(spans) != 1 || spans[0].Text != source || spans[0].Kind != InlineCode {
			t.Fatal("code changed displayed text", source, spans)
		}
	}
	for _, tc := range []struct {
		source    string
		selection SourceRange
		kind      InlineKind
	}{
		{"a\r\nb", rangeAt(0, 4), InlineBold},
		{"a\nb", rangeAt(0, 3), InlineItalic},
		{"中文", rangeAt(1, 3), InlineBold},
		{"abc", rangeAt(-1, 0), InlineBold},
		{"abc", rangeAt(2, 1), InlineBold},
		{"abc", rangeAt(0, 4), InlineBold},
		{"**abc**", rangeAt(2, 5), InlineItalic},
		{"**abc**", rangeAt(2, 4), InlineBold},
		{"**abc**", rangeAt(1, 1), InlineBold},
		{"a *b* c", rangeAt(0, 7), InlineBold},
		{"[text](a.md)", rangeAt(1, 5), InlineBold},
		{"`abc`", rangeAt(2, 3), InlineItalic},
		{"\\abc", rangeAt(1, 4), InlineBold},
		{"*abc", rangeAt(1, 4), InlineBold},
		{"`abc", rangeAt(0, 4), InlineCode},
		{"abc`", rangeAt(0, 4), InlineCode},
		{" a ", rangeAt(0, 3), InlineCode},
		{"` a `", rangeAt(0, 5), InlineCode},
		{"**a*b***", rangeAt(2, 5), InlineBold},
	} {
		if _, err := PlanInlineEdit(tc.source, tc.selection, tc.kind); err == nil || err.Error() == "" {
			t.Fatalf("ambiguous edit accepted: %#v", tc)
		}
	}
	large := strings.Repeat("x", maxInlineScanSuffix+1)
	if _, err := PlanInlineEdit(large, rangeAt(0, 1), InlineBold); err == nil {
		t.Fatal("format budget missing")
	}
}

func TestInlineEditEmptyInsertionDoesNotConsumeExistingMarks(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		kind   InlineKind
	}{
		{"italic-before-bold", "**a**", InlineItalic},
		{"italic-before-separated-bold", "x **a**", InlineItalic},
		{"code-before-code", "`a`", InlineCode},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := PlanInlineEdit(tc.source, SourceRange{Start: 0, End: 0}, tc.kind)
			if err == nil {
				t.Fatalf("empty insertion would consume an existing mark: source=%q plan=%#v", tc.source, plan)
			}
		})
	}
}

func TestInlineEditEmptyInsertionKeepsCompatibleMarks(t *testing.T) {
	for _, tc := range []struct {
		name, source, want string
		kind               InlineKind
		caret              int
	}{
		{"bold-before-bold", "**a**", "******a**", InlineBold, 2},
		{"italic-before-underscore-bold", "__a__", "**__a__", InlineItalic, 1},
		{"code-before-bold", "**a**", "``**a**", InlineCode, 1},
		{"code-before-separated-code", "x `a`", "``x `a`", InlineCode, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plan, err := PlanInlineEdit(tc.source, SourceRange{Start: 0, End: 0}, tc.kind)
			if err != nil {
				t.Fatal("compatible existing mark should be retained", err)
			}
			got := tc.source[:plan.Range.Start] + plan.Text + tc.source[plan.Range.End:]
			if got != tc.want || plan.Selection != (SourceRange{Start: tc.caret, End: tc.caret}) {
				t.Fatalf("empty insertion=%q, selection=%#v; want=%q caret=%d", got, plan.Selection, tc.want, tc.caret)
			}
		})
	}
}
