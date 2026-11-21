package graphview

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/search"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

func fieldUIClick(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	if err := tt.Click(label); err != nil {
		t.Fatal(err)
	}
}
func fieldUISet(t *testing.T, tt *ui.Tester, label, value string) {
	t.Helper()
	fieldUIClick(t, tt, label)
	tt.Command("selectAll")
	tt.Type(value)
}
func fieldUIOpen(t *testing.T, tt *ui.Tester) {
	t.Helper()
	fieldUIClick(t, tt, "打开或收起字段查找替换")
	if !tt.Focused("字段查找文字") {
		t.Fatal("toolbar did not focus accessible query")
	}
}
func fieldUIFocusTab(t *testing.T, tt *ui.Tester, label string) {
	t.Helper()
	for i := 0; i < 50 && !tt.Focused(label); i++ {
		tt.Key(0, ui.KeyTab)
	}
	if !tt.Focused(label) {
		t.Fatal("Tab cannot reach", label)
	}
}
func fieldUIBounds(t *testing.T, tt *ui.Tester, label string) ui.Rect {
	t.Helper()
	r, ok := tt.Find(label)
	if !ok || r.W <= 0 || r.H <= 0 {
		t.Fatal("missing accessible control", label, r)
	}
	return r
}

func TestFieldSearchUIRealToolbarKeyboardFindNoteReplaceAll(t *testing.T) {
	b := text.New(fieldSearchMindFixture)
	changed, opened := 0, 0
	v := newView(kindMindmap, editor.Host{Buffer: b, Changed: func() { changed++ }, OpenReference: func(string) { opened++ }})
	tt := ui.NewTester(v.Draw, 960, 720)
	fieldUIOpen(t, tt)
	tt.Type("cat")
	tt.Key(0, ui.KeyEnter)
	if v.search.results == nil || v.search.results.result.Count != 6 || b.String() != fieldSearchMindFixture || changed != 0 {
		t.Fatal("query Enter did not run pure field find", v.search.err)
	}
	fieldUIFocusTab(t, tt, "下一个字段匹配")
	tt.Key(0, ui.KeyEnter)
	if v.field != fieldTopic || v.selectedID != "root" || v.draft.input.Buffer.Selection() != (text.Range{Anchor: 3, Head: 6}) {
		t.Fatal("keyboard topic UTF16 navigation")
	}
	fieldUIClick(t, tt, "下一个字段匹配")
	if v.field != fieldNote || v.draft.input.Buffer.String() != "cat🙂" || v.draft.input.Buffer.Selection() != (text.Range{Anchor: 0, Head: 3}) || !tt.Focused("节点备注（纯文本）") {
		t.Fatal("real note editor was not shown/selected")
	}
	fieldUISet(t, tt, "字段替换文字", "X")
	fieldUIFocusTab(t, tt, "替换全部字段匹配")
	tt.Key(0, ui.KeyEnter)
	if v.search.err != nil || changed != 1 || opened != 0 || b.String() == fieldSearchMindFixture || b.History().UndoDepth != 1 {
		t.Fatal("real keyboard replace-all", v.search.err, changed, b.History())
	}
	if !tt.HasText("已替换6处；一次撤销可恢复。") {
		t.Fatal("replacement result not visible", tt.Texts())
	}
	after := b.String()
	if !b.Undo() || b.String() != fieldSearchMindFixture || !b.Redo() || b.String() != after {
		t.Fatal("real UI replacement history")
	}
}
func TestFieldSearchUIQueryAndReplacementIMEActionsNeverConfirm(t *testing.T) {
	for _, label := range []string{"字段查找文字", "字段替换文字"} {
		t.Run(label, func(t *testing.T) {
			b := text.New(fieldSearchMindFixture)
			changed := 0
			v := newView(kindMindmap, editor.Host{Buffer: b, Changed: func() { changed++ }})
			tt := ui.NewTester(v.Draw, 960, 720)
			fieldUIOpen(t, tt)
			tt.Type("cat")
			fieldUIClick(t, tt, "查找图字段")
			fieldUIClick(t, tt, label)
			tt.Command("selectAll")
			tt.Compose("猫🙂", 2)
			input := v.search.query
			if label == "字段替换文字" {
				input = v.search.replacement
			}
			preedit, selection, ver := input.Buffer.String(), input.Buffer.Selection(), input.Buffer.Version()
			for _, action := range []string{"查找图字段", "下一个字段匹配", "替换全部字段匹配"} {
				fieldUIClick(t, tt, action)
				if !errors.Is(v.search.err, editor.ErrFieldSearchDraft) || input.Buffer.String() != preedit || input.Buffer.Selection() != selection || input.Buffer.Version() != ver || !tt.Focused(label) || b.String() != fieldSearchMindFixture || changed != 0 {
					t.Fatal("pointer action committed/moved composing input", action, v.search.err)
				}
				if _, marked := input.MarkedRange(); !marked {
					t.Fatal("pointer action unmarked query")
				}
			}
			tt.Key(0, ui.KeyEnter)
			if b.String() != fieldSearchMindFixture || changed != 0 || input.Buffer.String() != preedit {
				t.Fatal("query Enter acted during IME")
			}
			tt.Type("cat")
			fieldUIClick(t, tt, "查找图字段")
			if v.search.err != nil || v.search.results == nil || v.search.results.result.Count != 6 {
				t.Fatal("native confirmation did not permit retry", v.search.err)
			}
		})
	}
}
func TestFieldSearchUIFieldIMEAndFailedDraftAreNotFlushed(t *testing.T) {
	for _, failed := range []bool{false, true} {
		b := text.New(fieldSearchMindFixture)
		changed := 0
		v := newView(kindMindmap, editor.Host{Buffer: b, Changed: func() { changed++ }})
		tt := ui.NewTester(v.Draw, 960, 720)
		fieldUIOpen(t, tt)
		tt.Type("cat")
		fieldUIClick(t, tt, "查找图字段")
		if failed {
			v.draft.input.Buffer = text.NewWithLimit(v.draft.base, len(v.draft.base))
		}
		fieldUIClick(t, tt, "节点文本")
		tt.Command("selectAll")
		tt.Compose("非常长的尚未完成草稿🙂", 12)
		if failed && v.draftError == nil {
			t.Fatal("failed native compose fixture did not fail")
		}
		draft := v.draft
		input := draft.input
		preedit, selection, ver := input.Buffer.String(), input.Buffer.Selection(), input.Buffer.Version()
		for _, action := range []string{"字段查找文字", "字段替换文字", "查找图字段", "下一个字段匹配", "替换全部字段匹配"} {
			fieldUIClick(t, tt, action)
			if !errors.Is(v.search.err, editor.ErrFieldSearchDraft) || v.draft != draft || input.Buffer.String() != preedit || input.Buffer.Selection() != selection || input.Buffer.Version() != ver || b.String() != fieldSearchMindFixture || changed != 0 || !tt.Focused("节点文本") {
				t.Fatal("field draft silently flushed/discarded", failed, action, v.search.err)
			}
		}
	}
}
func TestFieldSearchUIPressOwnerChangeFrameReleaseRejects(t *testing.T) {
	for _, mode := range []string{"buffer", "buffer-return", "release", "page-return", "query-return", "readonly-return"} {
		t.Run(mode, func(t *testing.T) {
			kind, source := kindMindmap, fieldSearchMindFixture
			if mode == "page-return" {
				kind, source = kindDrawio, fieldSearchDrawFixture
			}
			b := text.New(source)
			changed := 0
			v := newView(kind, editor.Host{Buffer: b, Changed: func() { changed++ }})
			tt := ui.NewTester(v.Draw, 1100, 760)
			fieldUIOpen(t, tt)
			tt.Type("cat")
			fieldUIClick(t, tt, "查找图字段")
			fieldUISet(t, tt, "字段替换文字", "bad")
			r := fieldUIBounds(t, tt, "替换全部字段匹配")
			x, y := r.X+r.W/2, r.Y+r.H/2
			tt.Press(x, y)
			switch mode {
			case "buffer":
				v.host.Buffer = text.New(source)
				tt.Frame()
			case "buffer-return":
				v.host.Buffer = text.New(source)
				tt.Frame()
				v.host.Buffer = b
				tt.Frame()
			case "release":
				v.Release()
				tt.Frame()
			case "page-return":
				v.setPageIndex(1)
				v.lastSynced = ^uint64(0)
				tt.Frame()
				v.setPageIndex(0)
				v.lastSynced = ^uint64(0)
				tt.Frame()
			case "query-return":
				v.invalidateFieldSearch()
				tt.Frame()
				fieldFind(t, v, search.Options{Query: "cat"})
				tt.Frame()
			case "readonly-return":
				v.host.ReadOnly = true
				tt.Frame()
				v.host.ReadOnly = false
				tt.Frame()
			}
			tt.Release(x, y)
			if b.String() != source || v.host.Buffer.String() != source || changed != 0 {
				t.Fatal("stale native release changed current authority", mode, v.search.err)
			}
			if !errors.Is(v.search.err, editor.ErrFieldSearchStale) {
				t.Fatal("stale native release lacked explicit refusal", mode, v.search.err)
			}
		})
	}
}

func TestFieldSearchUIPointerCancelOutsideScrollAndFreshClick(t *testing.T) {
	for _, mode := range []string{"cancel", "outside", "scroll"} {
		t.Run(mode, func(t *testing.T) {
			b := text.New(fieldSearchMindFixture)
			v := newView(kindMindmap, editor.Host{Buffer: b})
			tt := ui.NewTester(v.Draw, 960, 720)
			fieldUIOpen(t, tt)
			tt.Type("cat")
			fieldUIClick(t, tt, "查找图字段")
			fieldUISet(t, tt, "字段替换文字", "X")
			r := fieldUIBounds(t, tt, "替换全部字段匹配")
			x, y := r.X+r.W/2, r.Y+r.H/2
			tt.Press(x, y)
			switch mode {
			case "cancel":
				tt.CancelPress()
			case "outside":
				tt.Release(0, 710)
			case "scroll":
				tt.Wheel(x, y, 0, 20)
				tt.Release(x, y)
			}
			if b.String() != fieldSearchMindFixture || b.CanUndo() {
				t.Fatal("canceled native gesture mutated")
			}
			fieldUIClick(t, tt, "替换全部字段匹配")
			if b.String() == fieldSearchMindFixture || v.search.err != nil {
				t.Fatal("fresh click was not actionable", v.search.err)
			}
		})
	}
}
func TestFieldSearchUIRevealChangedReentryPreservesCurrentPublication(t *testing.T) {
	for _, label := range []string{"上一个字段匹配", "下一个字段匹配"} {
		for _, mode := range []string{"same-query-new-token", "new-query", "query-version", "query-buffer", "query-editor", "replacement-version", "replacement-buffer", "replacement-editor", "options", "toolbar-close", "toolbar-session", "buffer-owner", "release-owner", "readonly-owner", "publication-only", "draft-only"} {
			t.Run(label+"/"+mode, func(t *testing.T) {
				source := strings.Replace(fieldSearchMindFixture, `"topic":"🙂 cat"`, `"topic":"🙂 cat","collapsed":true`, 1)
				b := text.New(source)
				_ = b.SetSelection(b.LenUTF16(), 0)
				beforeSelection, version := b.SelectionSet(), b.Version()
				newErr := errors.New("current reentrant reveal error")
				var v *view
				var current *graphFieldResults
				var draft *fieldDraft
				var edge *editor.Editor
				var index int
				changed := 0
				v = newView(kindMindmap, editor.Host{Buffer: b, Changed: func() {
					changed++
					v.ensureSynced(nil)
					s := &v.search
					switch mode {
					case "same-query-new-token", "new-query":
						query := "other"
						if mode == "new-query" {
							query = "cat"
						}
						r := fieldFind(t, v, search.Options{Query: query})
						if err := v.RevealFieldMatch(r.Token, 0); err != nil {
							t.Fatal("reentrant reveal", err)
						}
					case "query-version":
						_ = s.query.Buffer.ReplaceUTF16(0, 0, "new")
					case "query-buffer":
						s.query.Buffer = text.New("other")
					case "query-editor":
						s.query = editor.New(s.query.Buffer)
					case "replacement-version":
						_ = s.replacement.Buffer.ReplaceUTF16(0, 0, "new")
					case "replacement-buffer":
						s.replacement.Buffer = text.New("new")
					case "replacement-editor":
						s.replacement = editor.New(s.replacement.Buffer)
					case "options":
						s.opts.Regex = !s.opts.Regex
					case "toolbar-close", "toolbar-session":
						v.runFieldSearchAction(v.captureFieldSearchAction(fieldSearchToggle))
						if mode == "toolbar-session" {
							v.runFieldSearchAction(v.captureFieldSearchAction(fieldSearchToggle))
						}
					case "buffer-owner":
						v.host.Buffer = text.New(b.String())
						v.ensureSynced(nil)
					case "release-owner":
						v.releaseEpoch++
						v.syncFieldSearchOwner()
					case "readonly-owner":
						v.host.ReadOnly = true
						v.syncFieldSearchOwner()
					}
					if mode == "draft-only" {
						v.clearDraft()
						v.field = fieldNote
						v.refreshLabelDraft()
					} else {
						s.err, s.notice, s.persistentNotice = newErr, "current reentrant notice", true
						s.index = 7
						s.edgeInput = editor.New(text.New("current preview"))
					}
					current, draft, edge, index = s.results, v.draft, s.edgeInput, s.index
				}})
				tt := ui.NewTester(v.Draw, 1100, 760)
				fieldUIOpen(t, tt)
				tt.Type("other")
				fieldUIClick(t, tt, "查找图字段")
				old := v.search.results.result.Token
				fieldUIClick(t, tt, label)
				if changed != 1 || b.String() == source || b.Version() != version+1 || b.History().UndoDepth != 1 {
					t.Fatal("expansion commit was lost/duplicated", changed, b.History())
				}
				if mode != "draft-only" && (v.search.err != newErr || v.search.notice != "current reentrant notice" || !v.search.persistentNotice) {
					t.Fatal("old reveal overwrote current error/notice", v.search.err, v.search.notice)
				}
				if v.search.results != current || v.search.index != index || v.search.edgeInput != edge || v.draft != draft {
					t.Fatal("old reveal overwrote current results/index/edge/draft", v.search.index, index)
				}
				if strings.Contains(mode, "query-new-token") || mode == "new-query" {
					if current == nil || current.result.Token == old {
						t.Fatal("reentrant Find did not retire the same-query token")
					}
				}
				after, afterSelection := b.String(), b.SelectionSet()
				if !b.Undo() || b.String() != source || !reflect.DeepEqual(b.SelectionSet(), beforeSelection) || b.CanUndo() || !b.Redo() || b.String() != after || !reflect.DeepEqual(b.SelectionSet(), afterSelection) || b.CanRedo() || changed != 1 {
					t.Fatal("committed expansion exact single Undo/Redo", b.History())
				}
			})
		}
	}
}

func TestFieldSearchUIRevealChangedInputOnlyRetiresCompletion(t *testing.T) {
	for _, label := range []string{"上一个字段匹配", "下一个字段匹配"} {
		for _, mode := range []string{"session", "query-version", "query-editor", "replacement-buffer", "draft-selection"} {
			t.Run(label+"/"+mode, func(t *testing.T) {
				source := strings.Replace(fieldSearchMindFixture, `"topic":"🙂 cat"`, `"topic":"🙂 cat","collapsed":true`, 1)
				b := text.New(source)
				changed := 0
				var v *view
				v = newView(kindMindmap, editor.Host{Buffer: b, Changed: func() {
					changed++
					s := &v.search
					// No Find, error, notice or draft replacement masks the individual
					// input/session identity change under test.
					switch mode {
					case "session":
						v.runFieldSearchAction(v.captureFieldSearchAction(fieldSearchToggle))
						v.runFieldSearchAction(v.captureFieldSearchAction(fieldSearchToggle))
					case "query-version":
						_ = s.query.Buffer.ReplaceUTF16(0, 0, "new")
					case "query-editor":
						s.query = editor.New(s.query.Buffer)
					case "replacement-buffer":
						s.replacement.Buffer = text.New("new")
					case "draft-selection":
						if err := v.draft.input.Buffer.SetSelection(3, 3); err != nil {
							t.Fatal("valid post-emoji draft selection", err)
						}
					}
				}})
				tt := ui.NewTester(v.Draw, 1100, 760)
				fieldUIOpen(t, tt)
				tt.Type("other")
				fieldUIClick(t, tt, "查找图字段")
				draft := v.draft
				fieldUIClick(t, tt, label)
				if changed != 1 || b.String() == source || b.History().UndoDepth != 1 || v.search.err != nil || v.search.results != nil || v.search.index != -1 || v.draft != draft {
					t.Fatal("retired input-only completion published", v.search.err, v.search.index, mode)
				}
				if mode == "draft-selection" && v.draft.input.Buffer.Selection() != (text.Range{Anchor: 3, Head: 3}) {
					t.Fatal("reentrant selection on same draft was overwritten")
				}
				if !b.Undo() || b.String() != source || !b.Redo() {
					t.Fatal("input-only reentry lost committed history")
				}
			})
		}
	}
}

func TestFieldSearchUIRevealCurrentErrorAndLegalTokenControls(t *testing.T) {
	for _, label := range []string{"上一个字段匹配", "下一个字段匹配"} {
		for _, mode := range []string{"hidden", "readonly-hidden", "visible", "stale-entry"} {
			t.Run(label+"/"+mode, func(t *testing.T) {
				source := fieldSearchMindFixture
				if mode != "visible" {
					source = strings.Replace(source, `"topic":"🙂 cat"`, `"topic":"🙂 cat","collapsed":true`, 1)
				}
				b := text.New(source)
				changed := 0
				v := newView(kindMindmap, editor.Host{Buffer: b, ReadOnly: mode == "readonly-hidden", Changed: func() { changed++ }})
				tt := ui.NewTester(v.Draw, 1100, 760)
				fieldUIOpen(t, tt)
				tt.Type("other")
				fieldUIClick(t, tt, "查找图字段")
				old := v.search.results.result.Token
				before, selection, generation, history := b.Version(), b.SelectionSet(), b.SelectionGeneration(), b.History()
				if mode == "stale-entry" {
					r := fieldUIBounds(t, tt, label)
					x, y := r.X+r.W/2, r.Y+r.H/2
					tt.Press(x, y)
					fresh := fieldFind(t, v, search.Options{Query: "other"})
					tt.Frame()
					tt.Release(x, y)
					if fresh.Token == old || v.search.results.result.Token != fresh.Token || !errors.Is(v.search.err, editor.ErrFieldSearchStale) {
						t.Fatal("stale entry callback must visibly refuse", v.search.err)
					}
				} else {
					fieldUIClick(t, tt, label)
					if mode == "readonly-hidden" {
						if !errors.Is(v.search.err, editor.ErrOutlineReadOnlyHidden) || !tt.HasText(v.search.err.Error()) {
							t.Fatal("current readonly error was hidden", v.search.err)
						}
					} else if v.search.err != nil || v.selectedID != "b" || v.field != fieldNote || v.draft.input.Buffer.Selection() != (text.Range{Anchor: 0, Head: 5}) {
						t.Fatal("legal current navigation rejected", v.search.err, v.selectedID)
					}
				}
				if mode == "hidden" {
					if changed != 1 || b.Version() != before+1 || v.search.results == nil || v.search.results.result.Token == old || v.search.index != 0 || !b.Undo() || b.String() != source || !b.Redo() {
						t.Fatal("legal hidden Reveal must publish a new token with one transaction", v.search.err)
					}
				} else if changed != 0 || b.String() != source || b.Version() != before || !reflect.DeepEqual(b.SelectionSet(), selection) || b.SelectionGeneration() != generation || b.History() != history {
					t.Fatal("nonmutation navigation/refusal changed authority")
				}
			})
		}
	}
}

const fieldUIFoldedNeedle = `{"format":"levnote.mindmap","version":1,"layout":"logicalStructure","root":{"id":"r","topic":"root","collapsed":true,"children":[{"id":"a","topic":"needle"}]}}`

const fieldUIUnfoldedNeedle = `{
  "format": "levnote.mindmap",
  "version": 1,
  "layout": "logicalStructure",
  "root": {
    "id": "r",
    "topic": "root",
    "collapsed": false,
    "children": [
      {
        "id": "a",
        "topic": "needle",
        "children": []
      }
    ]
  }
}
`

// Tester renders immediately after dispatch. Observe the action's raw output
// at the first subsequent frame entry, before Draw/ensureSynced can repair it.
func fieldUIRevealTester(v *view, changed *int, observe func()) *ui.Tester {
	return ui.NewTester(func(c *ui.Context) {
		if observe != nil && *changed > 0 {
			observe()
			observe = nil
		}
		v.Draw(c)
	}, 1100, 760)
}

func fieldUIAssertUnfoldHistory(t *testing.T, b *text.Buffer, before uint64, wantSelection text.Range, changed int) {
	t.Helper()
	if changed != 1 || b.Version() != before+1 || b.String() != fieldUIUnfoldedNeedle || b.Selection() != wantSelection || b.InGroup() || b.History().UndoDepth != 1 || b.CanRedo() {
		t.Fatalf("expected one committed unfold (Changed=%d, version=%d, history=%+v): %q", changed, b.Version(), b.History(), b.String())
	}
	if !b.Undo() || b.String() != fieldUIFoldedNeedle || b.Selection() != wantSelection || b.CanUndo() ||
		!b.Redo() || b.String() != fieldUIUnfoldedNeedle || b.Selection() != wantSelection || b.History().UndoDepth != 1 || b.CanRedo() {
		t.Fatal("unfold did not replay as one exact Undo/Redo", b.History())
	}
}

func TestFieldSearchUIRevealRefindBudgetFailure(t *testing.T) {
	for _, label := range []string{"上一个字段匹配", "下一个字段匹配"} {
		for _, mode := range []string{"current", "changed-refind"} {
			t.Run(label+"/"+mode, func(t *testing.T) {
				b := text.New(fieldUIFoldedNeedle)
				if err := b.SetSelection(b.LenUTF16(), 0); err != nil {
					t.Fatal(err)
				}
				before, selection := b.Version(), b.Selection()
				opts := search.Options{Query: "needle"}
				var v *view
				var current *graphFieldResults
				var currentDraft *fieldDraft
				changed := 0
				v = newView(kindMindmap, editor.Host{Buffer: b, Changed: func() {
					changed++
					if mode == "changed-refind" {
						fresh := fieldFind(t, v, opts)
						if fresh.Count != 1 {
							t.Fatal("reentrant request must still find the same needle", fresh)
						}
						if err := v.RevealFieldMatch(fresh.Token, 0); err != nil {
							t.Fatal("reentrant reveal", err)
						}
						current, currentDraft = v.search.results, v.draft
					}
				}})
				var old editor.FieldSearchResult
				observed := false
				tt := fieldUIRevealTester(v, &changed, func() {
					observed = true
					if v.selectedID != "a" || v.field != fieldTopic || v.draft == nil || v.draft.input.Buffer.String() != "needle" || v.draft.input.Buffer.Selection() != (text.Range{Anchor: 0, Head: 6}) {
						t.Error("unfolded match was not located with its field selection")
					}
					if mode == "changed-refind" {
						if v.search.err != nil || current == nil || v.search.results != current || current.result.Token == old.Token || v.search.index != 0 || v.draft != currentDraft {
							t.Error("superseded low-budget reveal published into the newer Find", v.search.err, v.search.index)
						}
						return
					}
					if !errors.Is(v.search.err, editor.ErrFieldSearchBudget) {
						t.Errorf("missing current automatic refind budget failure before redraw: %v", v.search.err)
					}
					if v.search.results != nil || v.search.index != -1 {
						t.Errorf("failed automatic refind left a results/index mismatch: %p / %d", v.search.results, v.search.index)
					}
				})
				fieldUIOpen(t, tt)
				tt.Type(opts.Query)
				// The provider API permits a custom SourceBytes ceiling; the
				// toolbar's own Find ordinarily uses the full default limits.
				limits := editor.DefaultFieldSearchLimits()
				limits.SourceBytes = len(fieldUIFoldedNeedle)
				old, err := v.FindFields(opts, limits)
				if err != nil || old.Count != 1 || old.Truncated || b.Version() != before || changed != 0 || len(fieldUIUnfoldedNeedle) <= limits.SourceBytes {
					t.Fatal("valid compact source must fit the custom search budget and yield one hit", old, err)
				}
				tt.Frame()
				fieldUIClick(t, tt, label)
				if !observed {
					t.Fatal("did not observe native toolbar completion")
				}
				if visible := tt.HasText(editor.ErrFieldSearchBudget.Error()); visible != (mode == "current") {
					t.Errorf("toolbar budget warning visibility = %v for %s", visible, mode)
				}
				if err := v.RevealFieldMatch(old.Token, 0); !errors.Is(err, editor.ErrFieldSearchStale) {
					t.Error("unfold revived the consumed token", err)
				}
				fieldUIAssertUnfoldHistory(t, b, before, selection, changed)
			})
		}
	}
}

func TestFieldSearchUIRevealChangedInputOwnership(t *testing.T) {
	for _, label := range []string{"上一个字段匹配", "下一个字段匹配"} {
		for _, owner := range []string{"query", "replacement"} {
			for _, mode := range []string{"group", "mark"} {
				t.Run(label+"/"+owner+"/"+mode, func(t *testing.T) {
					b := text.New(fieldUIFoldedNeedle)
					if err := b.SetSelection(b.LenUTF16(), 0); err != nil {
						t.Fatal(err)
					}
					before, selection := b.Version(), b.Selection()
					var input *editor.Editor
					changed := 0
					v := newView(kindMindmap, editor.Host{Buffer: b, Changed: func() {
						changed++
						switch {
						case mode == "group":
							if err := input.Buffer.BeginGroup(); err != nil {
								t.Fatal("begin input-only group", err)
							}
						case owner == "replacement":
							input.SetMarkedText(nil, "", ui.TextInputRange{})
						default:
							// Mark the exact existing query and leave the caret at
							// its old endpoint; query OnChange still runs normally.
							end := input.Buffer.LenUTF16()
							input.SetMarkedText(&ui.TextInputRange{End: end}, "needle", ui.TextInputRange{Start: end, End: end})
						}
					}})
					var draft *fieldDraft
					var draftSelection, inputSelection text.Range
					var inputBuffer *text.Buffer
					var inputVersion uint64
					var inputText string
					observed := false
					tt := fieldUIRevealTester(v, &changed, func() {
						observed = true
						if v.search.err != nil || v.search.notice != "" || v.search.results != nil || v.search.index != -1 || v.search.edgeInput != nil {
							t.Errorf("old reveal published into the new input owner before redraw: err=%v, index=%d", v.search.err, v.search.index)
						}
						if v.draft != draft || v.selectedID != "r" || draft.input.Buffer.Selection() != draftSelection {
							t.Error("superseded reveal changed the original graph draft or selection")
						}
						current := v.search.query
						if owner == "replacement" {
							current = v.search.replacement
						}
						mark, marked := input.MarkedRange()
						if current != input || input.Buffer != inputBuffer || inputBuffer.Version() != inputVersion || inputBuffer.Selection() != inputSelection || inputBuffer.String() != inputText || !inputBuffer.InGroup() || marked != (mode == "mark") || (marked && mark != (ui.TextInputRange{End: len(inputText)})) {
							t.Error("input-only ownership was not retained with the same editor, bytes, version, and selection", mark, marked)
						}
					})
					fieldUIOpen(t, tt)
					tt.Type("needle")
					fieldUIClick(t, tt, "查找图字段")
					if v.search.results == nil || v.search.results.result.Count != 1 {
						t.Fatal("expected one hidden match", v.search.err)
					}
					input = v.search.query
					if owner == "replacement" {
						input = v.search.replacement
					}
					inputBuffer = input.Buffer
					inputVersion, inputSelection, inputText = inputBuffer.Version(), inputBuffer.Selection(), inputBuffer.String()
					if inputBuffer.InGroup() || inputSelection != (text.Range{Anchor: len(inputText), Head: len(inputText)}) || owner == "replacement" && inputText != "" {
						t.Fatal("input-only test must begin without a group at the existing text endpoint", inputSelection, inputText)
					}
					draft = v.draft
					draftSelection = draft.input.Buffer.Selection()
					fieldUIClick(t, tt, label)
					if !observed {
						t.Fatal("did not observe native toolbar completion")
					}
					fieldUIAssertUnfoldHistory(t, b, before, selection, changed)
				})
			}
		}
	}
}

func TestFieldSearchUINarrowLabelsScopeReadonlyAndHistoryNotice(t *testing.T) {
	b := text.New(fieldSearchMindFixture)
	b.SetHistoryLimits(0, 0)
	v := newView(kindMindmap, editor.Host{Buffer: b})
	tt := ui.NewTester(v.Draw, 760, 480)
	fieldUIOpen(t, tt)
	for _, label := range []string{"字段查找文字", "字段替换文字", "查找图字段", "上一个字段匹配", "下一个字段匹配", "替换当前字段匹配", "替换全部字段匹配", "字段查找范围说明", "字段查找整词匹配"} {
		r := fieldUIBounds(t, tt, label)
		if r.X < 0 || r.Y < 0 || r.X+r.W > 760.1 || r.Y+r.H > 480.1 {
			t.Fatal("control inaccessible in minimum window", label, r)
		}
	}
	tt.Type("cat")
	fieldUIClick(t, tt, "查找图字段")
	fieldUISet(t, tt, "字段替换文字", "X")
	fieldUIClick(t, tt, "替换全部字段匹配")
	tt.Frame()
	if !strings.Contains(v.search.notice, "此次替换不可撤销") || b.CanUndo() {
		t.Fatal("zero history success misreported", v.search.notice)
	}
	fieldUISet(t, tt, "字段查找文字", "other")
	fieldUIClick(t, tt, "查找图字段")
	tt.Frame()
	if !tt.HasText("此次替换不可撤销：当前历史预算未保留本次事务。") {
		t.Fatal("history disclosure did not persist", v.search.notice, v.search.err, tt.Texts())
	}
	fieldUISet(t, tt, "字段替换文字", "other")
	fieldUIClick(t, tt, "替换全部字段匹配")
	if !tt.HasText("此次替换不可撤销：当前历史预算未保留本次事务。") {
		t.Fatal("no-op replacement hid the prior unretained transaction", v.search.notice)
	}
}
