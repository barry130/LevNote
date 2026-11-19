package app

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestModernTopSearchPreservesDocumentAndComposition(t *testing.T) {
	for _, tc := range []struct {
		name  string
		width int
		dark  bool
	}{
		{"dark_narrow", 760, true},
		{"light_narrow", 760, false},
		{"dark_wide", 1180, true},
		{"light_wide", 1180, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, first, second, tester := openDocumentsFixture(t)
			a.Dark = tc.dark
			tester.SetSize(tc.width, 480)
			tester.Frame()
			r, ok := tester.Find("搜索已打开的文档")
			if !ok || r.W < 140 || r.H < 30 || r.X < 0 || r.Y < 0 || r.X+r.W > float32(tc.width) || r.Y+r.H > 56 {
				t.Fatalf("top search is clipped: %+v", r)
			}
			before, history := first.Doc.Buffer.Snapshot(), first.Doc.Buffer.History()
			pickerClick(t, tester, "搜索已打开的文档")
			if !a.openDocuments.open || !tester.Focused(openDocumentsQueryLabel) {
				t.Fatal("top search did not focus the actual document picker")
			}
			tester.Key(0, ui.KeyEscape)
			if a.openDocuments.open || !tester.Focused("文档编辑区") || a.Current() != first {
				t.Fatal("top search did not restore its editor origin")
			}
			pickerClick(t, tester, "搜索已打开的文档")
			tester.Type("BETA")
			tester.Key(0, ui.KeyEnter)
			tester.Frame()
			if a.openDocuments.open || a.Current() != second || !tester.Focused("文档编辑区") || !first.Doc.Buffer.EqualSnapshot(before) || first.Doc.Buffer.History() != history {
				t.Fatal("top search failed navigation or changed the origin document")
			}
			tester.Compose("未提交", 3)
			snapshot, selection, h := second.Doc.Buffer.Snapshot(), second.Doc.Buffer.Selection(), second.Doc.Buffer.History()
			mark, marked := second.Editor.MarkedRange()
			pickerClick(t, tester, "搜索已打开的文档")
			next, active := second.Editor.MarkedRange()
			if a.openDocuments.open || a.Current() != second || !tester.Focused("文档编辑区") || !second.Doc.Buffer.EqualSnapshot(snapshot) || second.Doc.Buffer.Selection() != selection || second.Doc.Buffer.History() != h || mark != next || marked != active {
				t.Fatal("top search stole or committed composition")
			}
		})
	}
}
