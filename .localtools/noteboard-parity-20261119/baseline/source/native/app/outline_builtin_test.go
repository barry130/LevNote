package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

const builtinCollapsedMindmap = `{
  "format": "levnote.mindmap",
  "version": 1,
  "layout": "logicalStructure",
  "futureTop": {"keep": true},
  "root": {
    "id": "root",
    "topic": "根主题",
    "collapsed": true,
    "futureRoot": {"keep": "root"},
    "children": [
      {
        "id": "branch",
        "topic": "折叠分支",
        "collapsed": true,
        "futureBranch": {"keep": "branch"},
        "children": [
          {"id": "hidden", "topic": "隐藏目标", "children": []}
        ]
      }
    ]
  }
}`

type countingOutlineSource struct {
	editor.OutlineSource
	snapshots int
}

func (s *countingOutlineSource) OutlineSnapshot() (editor.OutlineSnapshot, error) {
	s.snapshots++
	return s.OutlineSource.OutlineSnapshot()
}

func outlineItemIndex(a *App, id string) int {
	for i, item := range a.OutlineItems {
		if item.ID == id {
			return i
		}
	}
	return -1
}

func beginMarkdownTextComposition(t *testing.T, tab *Tab, tester *ui.Tester, value string) {
	t.Helper()
	rect, ok := tester.Find(value)
	if !ok {
		t.Fatalf("Markdown draft target %q did not render: %#v", value, tester.Texts())
	}
	tester.ClickAt(rect.X+rect.W/2, rect.Y+rect.H/2)
	tester.Frame()
	tester.ClickAt(rect.X+rect.W/2, rect.Y+rect.H/2)
	tester.Frame()
	tester.Command("selectAll")
	tester.Compose("中文🙂", 3)
	if strings.Contains(tab.Doc.Buffer.String(), "中文🙂") {
		t.Fatal("preview draft reached the host buffer before outline activation")
	}
}

func TestBuiltinMarkdownOutlinePreservesDuplicateIdentityAcrossComposedInsertion(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "duplicate.md")
	if err := os.WriteFile(path, []byte("# 重复\r\n\r\n第一段\r\n\r\n# 重复\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	if tab == nil || tab.Adapter == nil || tab.Adapter.Outline == nil {
		t.Fatalf("built-in Markdown outline factory missing: %#v", tab)
	}
	if !a.setMode(tab, "source") {
		t.Fatalf("Markdown source mode failed: %s", a.Message)
	}
	base, _ := a.adapterByID("markdown")
	createCalls := 0
	counted := base
	counted.Create = func(host editor.Host) editor.View {
		createCalls++
		return base.Create(host)
	}
	tab.Adapter = &counted
	a.ShowOutline = true
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	tester.Frame()
	if len(a.OutlineItems) != 2 ||
		a.OutlineItems[0].AnchorID != "重复" ||
		a.OutlineItems[1].AnchorID != "重复-2" {
		t.Fatalf("duplicate anchors = %#v", a.OutlineItems)
	}
	a.OutlineSelected = 1
	tester.Frame()
	oldID := a.OutlineItems[1].ID
	oldOffset := a.OutlineItems[1].Offset
	insertion := "# 重复\r\n\r\n"
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	tab.Editor.Reveal(0, 0)
	tester.Compose(insertion, len([]rune(insertion)))
	if _, composing := tab.Editor.MarkedRange(); !composing {
		t.Fatal("duplicate-shift insertion did not remain a pending native composition")
	}

	if !a.activateSelectedOutlineItem() {
		t.Fatalf("duplicate target activation failed: %s", a.Message)
	}
	wantOffset := oldOffset + text.UTF16Len(insertion)
	if selection := tab.Doc.Buffer.Selection(); selection.Start() != wantOffset || selection.End() != wantOffset {
		t.Fatalf("old duplicate identity mapped to offset %#v, want %d; old=%q", selection, wantOffset, oldID)
	}
	if len(a.OutlineItems) != 3 || a.OutlineItems[1].AnchorID != "重复-2" || a.OutlineItems[2].AnchorID != "重复-3" {
		t.Fatalf("inserted duplicate did not preserve / disambiguate anchors: %#v", a.OutlineItems)
	}
	selected, ok := a.outlineRowAt(a.OutlineSelected)
	if !ok || selected.Offset != wantOffset || selected.ID == oldID {
		t.Fatalf("selection did not adopt the mapped current duplicate: selected=%d item=%#v old=%q", a.OutlineSelected, selected, oldID)
	}
	if createCalls != 0 {
		t.Fatalf("source-only duplicate navigation constructed a Markdown preview %d times", createCalls)
	}
}

func TestBuiltinMarkdownPreviewFlushesDifferentDraftThenNavigatesWithoutModeChange(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "preview.md")
	if err := os.WriteFile(path, []byte("# 目标章\r\n\r\n草稿段落\r\n\r\n## 后章\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	if tab == nil || tab.Adapter == nil || tab.Mode != "preview" {
		t.Fatalf("Markdown preview did not open normally: %#v", tab)
	}
	a.ShowOutline = true
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	tester.Frame()
	a.OutlineSelected = 0
	tester.Frame()
	beginMarkdownTextComposition(t, tab, tester, "草稿段落")

	if !a.activateSelectedOutlineItem() {
		t.Fatalf("real preview outline activation failed: %s", a.Message)
	}
	if tab.Mode != "preview" || tab.View == nil {
		t.Fatalf("preview outline navigation changed or dropped the mounted view: mode=%q view=%#v", tab.Mode, tab.View)
	}
	if !strings.Contains(tab.Doc.Buffer.String(), "中文🙂") ||
		!strings.Contains(tab.Doc.Buffer.String(), "# 目标章") {
		t.Fatalf("preview lifecycle lost the draft or target: %q", tab.Doc.Buffer.String())
	}
	selected, ok := a.outlineRowAt(a.OutlineSelected)
	if !ok || selected.Text != "目标章" || a.Message != "已定位：目标章" {
		t.Fatalf("preview selection/status = selected=%#v message=%q", selected, a.Message)
	}
}

func TestBuiltinMarkdownSplitNavigatesExistingViewAndRevealsUTF16Offset(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "split.md")
	value := "# 🙂首章\r\n\r\n正文\r\n\r\n## 后章\r\n"
	if err := os.WriteFile(path, []byte(value), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	a.ShowOutline = true
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	tester.Frame()
	if !a.setMode(tab, "split") {
		t.Fatalf("Markdown split mode failed: %s", a.Message)
	}
	tester.Frame()
	tester.Frame()
	if len(a.OutlineItems) < 2 {
		t.Fatalf("UTF-16 outline is incomplete: %#v", a.OutlineItems)
	}
	a.OutlineSelected = 1
	view := tab.View
	targetOffset := a.OutlineItems[1].Offset
	beforeCaret := tab.Doc.Buffer.Selection()

	if !a.activateSelectedOutlineItem() {
		t.Fatalf("real split outline activation failed: %s", a.Message)
	}
	if tab.Mode != "split" || tab.View != view {
		t.Fatalf("split navigation changed or replaced its view: mode=%q view=%#v old=%#v", tab.Mode, tab.View, view)
	}
	if selection := tab.Doc.Buffer.Selection(); selection.Start() != targetOffset || selection.End() != targetOffset {
		t.Fatalf("split reveal = %#v, want UTF-16 offset %d", selection, targetOffset)
	}
	if beforeCaret == tab.Doc.Buffer.Selection() {
		t.Log("split source caret already matched the resolved heading")
	}
}

func TestBuiltinMarkdownAnchorUsesSlugResolver(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "anchor.md")
	if err := os.WriteFile(path, []byte("# 🙂标题\r\n\r\n正文\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	tester.Frame()

	a.editorHost(tab).OpenReference("#%E6%A0%87%E9%A2%98")
	if !strings.Contains(a.Message, "已定位：") ||
		strings.Contains(strings.Join(a.Issues, "\n"), "锚点") ||
		tab.Mode != "preview" {
		t.Fatalf("built-in Markdown anchor resolution failed: mode=%q message=%q issues=%#v",
			tab.Mode, a.Message, a.Issues)
	}
}

func TestBuiltinMindmapOutlineExpandsCollapsedAncestorsWithOneUndo(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "collapsed.mindmap")
	if err := os.WriteFile(path, []byte(builtinCollapsedMindmap), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	if tab == nil || tab.Adapter == nil || tab.Adapter.ID != "mindmap" {
		t.Fatalf("built-in mindmap adapter did not open: %#v", tab)
	}
	a.ShowOutline = true
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	tester.Frame()
	if len(a.OutlineItems) != 3 {
		t.Fatalf("collapsed descendants were omitted from the outline: %#v", a.OutlineItems)
	}
	hiddenIndex := outlineItemIndex(a, "hidden")
	if hiddenIndex < 0 || a.OutlineItems[hiddenIndex].Level != 3 || a.OutlineItems[hiddenIndex].Offset != -1 {
		t.Fatalf("hidden descendant descriptor = %d %#v", hiddenIndex, a.OutlineItems)
	}
	original := tab.Doc.Buffer.String()
	oldVersion := tab.Doc.Buffer.Version()

	if !a.activateOutlineTarget(tab, "hidden", false, "大纲定位") {
		t.Fatalf("real mindmap collapsed-target navigation failed: %s", a.Message)
	}
	if tab.Doc.Buffer.Version() == oldVersion {
		t.Fatal("collapsed ancestor expansion produced no document transaction")
	}
	expanded := tab.Doc.Buffer.String()
	for _, retained := range []string{"futureTop", "futureRoot", "futureBranch"} {
		if !strings.Contains(expanded, retained) {
			t.Fatalf("unknown metadata %q was lost after expansion: %s", retained, expanded)
		}
	}
	if got := strings.Count(expanded, `"collapsed": false`); got != 2 {
		t.Fatalf("collapsed ancestor edits = %d, want root and branch in one transaction: %s", got, expanded)
	}
	if selected, ok := a.outlineRowAt(a.OutlineSelected); !ok || selected.ID != "hidden" {
		t.Fatalf("expanded target was not selected: selected=%d item=%#v", a.OutlineSelected, selected)
	}

	if !tab.Editor.Command("undo") {
		t.Fatal("expanding a mindmap outline target was not undoable")
	}
	if tab.Doc.Buffer.String() != original {
		t.Fatalf("one undo did not restore both folds and exact source:\n got %s\nwant %s",
			tab.Doc.Buffer.String(), original)
	}
}

func TestBuiltinMindmapReadonlyHiddenTargetRefusesWithoutMutation(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "readonly.mindmap")
	if err := os.WriteFile(path, []byte(builtinCollapsedMindmap), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	tab.Doc.ReadOnly = true
	tab.Editor.ReadOnly = true
	a.ShowOutline = true
	runner := newControlledExporter()
	a.exporter = runner
	tester := ui.NewTester(a.View, 760, 480)
	call, job := awaitExportCall(t, runner), a.previewJob
	outcome := previewSuccess()
	outcome.result.Payload = []byte(strings.NewReplacer(`"folded"`, `"branch"`, "预览🙂", "根主题", "折叠", "折叠分支").Replace(previewLiteral))
	call.response <- outcome
	awaitPreviewDone(t, job)
	tester.Frame()
	tester.Frame()
	if tab.View == nil {
		t.Fatal("real mindmap view was not mounted")
	}
	if err := tab.Doc.Buffer.SetSelection(3, 3); err != nil {
		t.Fatal(err)
	}
	original := tab.Doc.Buffer.String()
	version := tab.Doc.Buffer.Version()
	selection := tab.Doc.Buffer.Selection()

	if a.activateOutlineTarget(tab, "hidden", false, "大纲定位") {
		t.Fatal("readonly hidden mindmap target unexpectedly navigated")
	}
	if !strings.Contains(a.Message, editor.ErrOutlineReadOnlyHidden.Error()) {
		t.Fatalf("readonly-hidden limitation was not explicit: %q", a.Message)
	}
	if tab.Doc.Buffer.String() != original || tab.Doc.Buffer.Version() != version ||
		tab.Doc.Buffer.Selection() != selection {
		t.Fatalf("readonly-hidden failure mutated the source: version=%d/%d selection=%#v/%#v",
			tab.Doc.Buffer.Version(), version, tab.Doc.Buffer.Selection(), selection)
	}
}

func TestBuiltinMindmapSourceOnlyUsesFactoryWithoutCreatingCanvas(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "source-only.mindmap")
	if err := os.WriteFile(path, []byte(builtinCollapsedMindmap), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	if !a.setMode(tab, "source") {
		t.Fatalf("mindmap source mode failed: %s", a.Message)
	}
	base := *tab.Adapter
	createCalls := 0
	counted := base
	counted.Create = func(host editor.Host) editor.View {
		createCalls++
		return base.Create(host)
	}
	tab.Adapter = &counted
	a.ShowOutline = true
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	tester.Frame()
	if createCalls != 0 || tab.View != nil {
		t.Fatalf("source outline collection mounted a graph canvas: calls=%d view=%#v", createCalls, tab.View)
	}
	original := tab.Doc.Buffer.String()
	version := tab.Doc.Buffer.Version()

	if a.activateOutlineTarget(tab, "hidden", false, "大纲定位") {
		t.Fatal("source-only graph item without a source coordinate unexpectedly navigated")
	}
	if createCalls != 0 || tab.View != nil || tab.Doc.Buffer.String() != original ||
		tab.Doc.Buffer.Version() != version ||
		!strings.Contains(a.Message, "请切到阅读/分屏定位") {
		t.Fatalf("source-only graph navigation mutated or mounted state: calls=%d view=%#v version=%d/%d message=%q",
			createCalls, tab.View, tab.Doc.Buffer.Version(), version, a.Message)
	}
}

func TestBuiltinMindmapSplitHiddenNavigationPreservesSourceSelectionAcrossUndoRedo(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "split.mindmap")
	if err := os.WriteFile(path, []byte(builtinCollapsedMindmap), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	if !a.setMode(tab, "split") {
		t.Fatalf("mindmap split mode failed: %s", a.Message)
	}
	a.ShowOutline = true
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	tester.Frame()
	if tab.View == nil || len(a.OutlineItems) != 3 {
		t.Fatalf("real split graph did not expose its outline: view=%#v items=%#v", tab.View, a.OutlineItems)
	}
	if err := tab.Doc.Buffer.SetSelection(12, 3); err != nil {
		t.Fatal(err)
	}
	original := tab.Doc.Buffer.String()
	view, oldVersion, selection := tab.View, tab.Doc.Buffer.Version(), tab.Doc.Buffer.Selection()
	if selection.Start() == 0 || selection.Start() == selection.End() {
		t.Fatalf("test requires a nonzero source range: %#v", selection)
	}

	if !a.activateOutlineTarget(tab, "hidden", false, "大纲定位") {
		t.Fatalf("real split hidden graph navigation failed: %s", a.Message)
	}
	expandedVersion := tab.Doc.Buffer.Version()
	expanded := tab.Doc.Buffer.String()
	if tab.Mode != "split" || tab.View != view || expandedVersion == oldVersion {
		t.Fatalf("split navigation changed its pane or omitted the expansion transaction: mode=%q view=%#v/%#v version=%d/%d",
			tab.Mode, tab.View, view, tab.Doc.Buffer.Version(), oldVersion)
	}
	if tab.Doc.Buffer.Selection() != selection {
		t.Fatalf("split graph expansion replaced the source selection: %#v/%#v",
			tab.Doc.Buffer.Selection(), selection)
	}
	if !strings.Contains(a.Message, "已在图形中定位") || !strings.Contains(a.Message, "没有源码定位坐标") {
		t.Fatalf("split graph limitation was not disclosed: %q", a.Message)
	}

	if !tab.Editor.Command("undo") {
		t.Fatal("split hidden-node expansion was not undoable")
	}
	if tab.Doc.Buffer.String() != original || tab.Doc.Buffer.Selection() != selection {
		t.Fatalf("undo did not restore the exact folded source and selection: selection=%#v/%#v",
			tab.Doc.Buffer.Selection(), selection)
	}
	if !tab.Editor.Command("redo") {
		t.Fatal("split hidden-node expansion was not redoable")
	}
	if tab.Doc.Buffer.String() != expanded || tab.Doc.Buffer.Selection() != selection {
		t.Fatalf("redo did not restore the expansion while preserving selection: selection=%#v/%#v",
			tab.Doc.Buffer.Selection(), selection)
	}
}

func TestBuiltinOutlineReleasedOnPluginModeTabAndReload(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "note.md")
	if err := os.WriteFile(path, []byte("# 标题\n\n正文\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	a.ShowOutline = true
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	if a.OutlineSource == nil {
		t.Fatal("Markdown outline source was not created")
	}

	if !a.setMode(tab, "source") {
		t.Fatalf("mode change failed: %s", a.Message)
	}
	if a.OutlineSource != nil {
		t.Fatal("mode change retained the previous outline source")
	}
	tester.Frame()
	if a.OutlineSource == nil {
		t.Fatal("source-mode outline was not recreated from the factory")
	}

	a.NewDocument("text")
	if a.OutlineSource != nil {
		t.Fatal("tab change retained the inactive document's outline source")
	}
	if !a.Select(0) {
		t.Fatal("could not return to the Markdown tab")
	}
	tester.Frame()
	if a.OutlineSource == nil {
		t.Fatal("returning to the Markdown tab did not recreate its source")
	}

	a.setPluginEnabled("markdown", false)
	if tab.Adapter != nil || a.OutlineSource != nil {
		t.Fatalf("plugin reconfigure retained a stale outline: adapter=%#v source=%#v", tab.Adapter, a.OutlineSource)
	}
	a.setPluginEnabled("markdown", true)
	tester.Frame()
	if a.OutlineSource == nil {
		t.Fatal("reenabled Markdown outline source was not recreated")
	}
	a.Reload()
	if a.OutlineSource != nil {
		t.Fatal("reload retained the pre-reload outline controller")
	}
}

func TestBuiltinMarkdownLargeOutlineCachesSnapshotAndVirtualizesRows(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "large.md")
	var markdown strings.Builder
	markdown.WriteString("# 起始\n\n")
	for i := 0; i < editor.MaxOutlineItems+20; i++ {
		fmt.Fprintf(&markdown, "## 条目 %04d\n\n正文。\n\n", i)
	}
	if err := os.WriteFile(path, []byte(markdown.String()), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	if !a.setMode(tab, "source") {
		t.Fatalf("large Markdown source mode failed: %s", a.Message)
	}
	base := *tab.Adapter
	originalOutline := base.Outline
	var counted *countingOutlineSource
	base.Outline = func(host editor.Host) editor.OutlineSource {
		counted = &countingOutlineSource{OutlineSource: originalOutline(host)}
		return counted
	}
	tab.Adapter = &base
	a.ShowOutline = true
	tester := ui.NewTester(a.View, 760, 480)
	builtPerFrame := []int{}
	a.outlineRowBuilt = func(int) {
		if len(builtPerFrame) > 0 {
			builtPerFrame[len(builtPerFrame)-1]++
		}
	}
	for i := 0; i < 4; i++ {
		builtPerFrame = append(builtPerFrame, 0)
		tester.Frame()
	}
	if counted == nil || counted.snapshots != 1 {
		snapshots := 0
		if counted != nil {
			snapshots = counted.snapshots
		}
		t.Fatalf("repeated frames rebuilt the large outline snapshot: calls=%d", snapshots)
	}
	if len(a.OutlineItems) != editor.MaxOutlineItems || !a.OutlineTruncated {
		t.Fatalf("large outline bounds = items=%d truncated=%v",
			len(a.OutlineItems), a.OutlineTruncated)
	}
	for frame, built := range builtPerFrame {
		if built <= 0 || built >= 100 {
			t.Fatalf("frame %d built %d of %d outline rows", frame, built, len(a.OutlineItems))
		}
	}
}
