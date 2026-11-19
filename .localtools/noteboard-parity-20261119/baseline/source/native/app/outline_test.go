package app

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

type outlineSourceStub struct {
	buffer *text.Buffer

	snapshotFn      func() (editor.OutlineSnapshot, error)
	resolveItemFn   func(string) (editor.OutlineItem, error)
	resolveAnchorFn func(string) (editor.OutlineItem, error)

	snapshotCalls int
	releases      int
	itemInputs    []string
	anchorInputs  []string
}

func (s *outlineSourceStub) OutlineSnapshot() (editor.OutlineSnapshot, error) {
	s.snapshotCalls++
	if s.snapshotFn == nil {
		return editor.OutlineSnapshot{Version: s.buffer.Version()}, nil
	}
	return s.snapshotFn()
}

func (s *outlineSourceStub) ResolveOutlineItem(id string) (editor.OutlineItem, error) {
	s.itemInputs = append(s.itemInputs, id)
	if s.resolveItemFn == nil {
		return editor.OutlineItem{}, editor.ErrOutlineStale
	}
	return s.resolveItemFn(id)
}

func (s *outlineSourceStub) ResolveOutlineAnchor(anchor string) (editor.OutlineItem, error) {
	s.anchorInputs = append(s.anchorInputs, anchor)
	if s.resolveAnchorFn == nil {
		return editor.OutlineItem{}, editor.ErrOutlineUnsupported
	}
	return s.resolveAnchorFn(anchor)
}

func (s *outlineSourceStub) Release() {
	s.releases++
	s.buffer = nil
}

type outlineNavigatorView struct {
	failingView
	navigations []string
	navigateErr error
	onFlush     func()
}

func (v *outlineNavigatorView) Flush() error {
	v.flushes++
	if v.onFlush != nil {
		v.onFlush()
	}
	return v.flushErr
}

func (v *outlineNavigatorView) NavigateOutlineItem(id string) error {
	v.navigations = append(v.navigations, id)
	return v.navigateErr
}

func installOutlineStub(a *App, tab *Tab, source *outlineSourceStub) (*int, *outlineNavigatorView) {
	createCalls := 0
	created := &outlineNavigatorView{}
	tab.Adapter = &editor.Adapter{
		ID:      "outline-test",
		Name:    "大纲测试",
		Matches: func(string) bool { return true },
		Outline: func(host editor.Host) editor.OutlineSource {
			source.buffer = host.Buffer
			return source
		},
		Create: func(host editor.Host) editor.View {
			createCalls++
			created.failingView = failingView{}
			return created
		},
	}
	return &createCalls, created
}

func installSourceOutlineFactory(tab *Tab, source *outlineSourceStub, calls *int) {
	tab.Adapter = nil
	tab.OutlineFactory = func(host editor.Host) editor.OutlineSource {
		*calls++
		source.buffer = host.Buffer
		return source
	}
}

func newOutlineTab(a *App, value, name string) *Tab {
	a.add(document.New(value), name, "")
	return a.Current()
}

func TestSourceOutlineFactoryDoesNotRequireRichAdapterAndInvalidatesByGeneration(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	tab := newOutlineTab(a, "[一]\n# 二\n", "source.ini")
	tab.Mode = "source"
	first := &outlineSourceStub{}
	first.snapshotFn = func() (editor.OutlineSnapshot, error) {
		return editor.OutlineSnapshot{Version: first.buffer.Version(), Items: []editor.OutlineItem{{
			ID: "first", Text: "第一版", Level: 1, Offset: 0,
		}}}, nil
	}
	firstCalls := 0
	installSourceOutlineFactory(tab, first, &firstCalls)
	a.ShowOutline = true
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	tester.Frame()
	if tab.Adapter != nil || tab.View != nil || firstCalls != 1 || len(a.OutlineItems) != 1 {
		t.Fatalf("source-only factory created a rich view or did not load: adapter=%#v view=%#v calls=%d items=%#v",
			tab.Adapter, tab.View, firstCalls, a.OutlineItems)
	}
	if a.OutlineFactoryGeneration != tab.OutlineFactoryGeneration {
		t.Fatalf("cached factory generation = %d, tab generation = %d",
			a.OutlineFactoryGeneration, tab.OutlineFactoryGeneration)
	}

	second := &outlineSourceStub{}
	second.snapshotFn = func() (editor.OutlineSnapshot, error) {
		return editor.OutlineSnapshot{Version: second.buffer.Version(), Items: []editor.OutlineItem{{
			ID: "second", Text: "第二版", Level: 1, Offset: 0,
		}}}, nil
	}
	secondCalls := 0
	installSourceOutlineFactory(tab, second, &secondCalls)
	tab.OutlineFactoryGeneration++
	if err := a.ensureOutlineSnapshot(tab, "", false); err != nil {
		t.Fatal(err)
	}
	if first.releases != 1 || secondCalls != 1 || len(a.OutlineItems) != 1 ||
		a.OutlineItems[0].ID != "second" ||
		a.OutlineFactoryGeneration != tab.OutlineFactoryGeneration {
		t.Fatalf("changed factory generation reused stale source: releases=%d secondCalls=%d items=%#v appGen=%d tabGen=%d",
			first.releases, secondCalls, a.OutlineItems, a.OutlineFactoryGeneration, tab.OutlineFactoryGeneration)
	}
}

func TestOutlineSourceModeResolvesOldIdentityAfterFlushWithoutCreatingView(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	tab := newOutlineTab(a, "# 标题\n\n段落", "大纲.md")
	tab.Mode = "source"
	source := &outlineSourceStub{}
	source.snapshotFn = func() (editor.OutlineSnapshot, error) {
		return editor.OutlineSnapshot{
			Version: source.buffer.Version(),
			Items: []editor.OutlineItem{{
				ID:   fmt.Sprintf("heading-v%d", source.buffer.Version()),
				Text: "目标大纲", Level: 1, Offset: 0, AnchorID: "标题",
			}},
		}, nil
	}
	source.resolveItemFn = func(oldID string) (editor.OutlineItem, error) {
		if oldID == "" {
			t.Fatal("resolver received an empty identity")
		}
		return editor.OutlineItem{
			ID:   fmt.Sprintf("heading-v%d", source.buffer.Version()),
			Text: "目标大纲", Level: 1, Offset: 0, AnchorID: "标题",
		}, nil
	}
	createCalls, _ := installOutlineStub(a, tab, source)
	a.ShowOutline = true
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	clickTesterText(t, tester, "目标大纲")
	tester.Frame()
	oldID := a.OutlineItems[0].ID
	tab.Editor.ReplaceText(&ui.TextInputRange{
		Start: tab.Doc.Buffer.LenUTF16(), End: tab.Doc.Buffer.LenUTF16(),
	}, "\n插入")

	if !a.activateSelectedOutlineItem() {
		t.Fatalf("source outline activation failed: %s", a.Message)
	}
	if *createCalls != 0 || tab.View != nil {
		t.Fatalf("source navigation constructed a preview: create=%d view=%#v", *createCalls, tab.View)
	}
	if len(source.itemInputs) == 0 || source.itemInputs[0] != oldID {
		t.Fatalf("old identity was not resolved through the source: inputs=%#v old=%q", source.itemInputs, oldID)
	}
	if !strings.Contains(tab.Doc.Buffer.String(), "插入") {
		t.Fatal("outline lifecycle did not flush the source buffer")
	}
	if selection := tab.Doc.Buffer.Selection(); selection.Start() != 0 || selection.End() != 0 {
		t.Fatalf("heading offset reveal = %#v, want UTF-16 offset 0", selection)
	}
	item, ok := a.outlineRowAt(a.OutlineSelected)
	if !ok || item.ID == oldID || item.ID != fmt.Sprintf("heading-v%d", source.buffer.Version()) {
		t.Fatalf("selection did not adopt the resolver's current identity: selected=%d item=%#v", a.OutlineSelected, item)
	}
	if !a.persistPending {
		t.Fatal("outline navigation did not mark the session position pending")
	}
}

func TestOutlineActivationFlushesPendingNativeComposition(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	tab := newOutlineTab(a, "# 标题\n\n段落", "ime-outline.md")
	tab.Mode = "source"
	source := &outlineSourceStub{}
	source.snapshotFn = func() (editor.OutlineSnapshot, error) {
		return editor.OutlineSnapshot{Version: source.buffer.Version(), Items: []editor.OutlineItem{{
			ID: "target", Text: "目标大纲", Level: 1, Offset: 0,
		}}}, nil
	}
	source.resolveItemFn = func(string) (editor.OutlineItem, error) {
		return editor.OutlineItem{ID: "target", Text: "目标大纲", Level: 1, Offset: 0}, nil
	}
	installOutlineStub(a, tab, source)
	a.ShowOutline = true
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	clickTesterText(t, tester, "目标大纲")
	tester.Frame()
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	end := tab.Doc.Buffer.LenUTF16()
	tab.Editor.Reveal(end, end)
	tester.Frame()
	tester.Compose("中文🙂", 3)
	if _, composing := tab.Editor.MarkedRange(); !composing {
		t.Fatal("native composition was not pending before outline activation")
	}
	if !a.activateSelectedOutlineItem() {
		t.Fatalf("outline activation failed with pending composition: %s", a.Message)
	}
	if !strings.Contains(tab.Doc.Buffer.String(), "中文🙂") {
		t.Fatalf("outline activation lost pending composition: %q", tab.Doc.Buffer.String())
	}
}

func TestOutlineFlushFailureRetainsStateAndNegativeOffsetIsActionable(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	tab := newOutlineTab(a, "source", "graph-source.mindmap")
	tab.Mode = "source"
	source := &outlineSourceStub{}
	source.snapshotFn = func() (editor.OutlineSnapshot, error) {
		return editor.OutlineSnapshot{Version: source.buffer.Version(), Items: []editor.OutlineItem{{
			ID: "graph-node", Text: "图形节点", Level: 1, Offset: -1,
		}}}, nil
	}
	source.resolveItemFn = func(string) (editor.OutlineItem, error) {
		return editor.OutlineItem{ID: "graph-node", Text: "图形节点", Level: 1, Offset: -1}, nil
	}
	createCalls, _ := installOutlineStub(a, tab, source)
	view := &failingView{flushErr: errors.New("draft conflict")}
	tab.View = view
	a.ShowOutline = true
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	clickTesterText(t, tester, "图形节点")
	tester.Frame()
	selected := a.OutlineSelected
	items := append([]editor.OutlineItem(nil), a.OutlineItems...)

	if a.activateSelectedOutlineItem() {
		t.Fatal("outline activation succeeded after a failed view flush")
	}
	if tab.View != view || tab.Mode != "source" || a.OutlineSelected != selected ||
		fmt.Sprintf("%#v", a.OutlineItems) != fmt.Sprintf("%#v", items) || len(source.itemInputs) != 0 {
		t.Fatalf("failed flush mutated lifecycle state: view=%v mode=%q selected=%d items=%#v resolves=%#v",
			tab.View, tab.Mode, a.OutlineSelected, a.OutlineItems, source.itemInputs)
	}

	view.flushErr = nil
	beforeVersion := tab.Doc.Buffer.Version()
	beforeSelection := tab.Doc.Buffer.Selection()
	if a.activateSelectedOutlineItem() {
		t.Fatal("source-only graph item without a source offset unexpectedly succeeded")
	}
	if *createCalls != 0 || tab.Doc.Buffer.Version() != beforeVersion ||
		tab.Doc.Buffer.Selection() != beforeSelection ||
		!strings.Contains(a.Message, "请切到阅读/分屏定位") {
		t.Fatalf("negative source offset mutated app: create=%d version=%d/%d selection=%#v/%#v message=%q",
			*createCalls, beforeVersion, tab.Doc.Buffer.Version(), beforeSelection, tab.Doc.Buffer.Selection(), a.Message)
	}
}

func TestOutlineActivationRejectsLifecycleChangesDuringFlush(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*App, *Tab)
	}{
		{name: "controller released", mutate: func(a *App, _ *Tab) { a.releaseOutline() }},
		{name: "adapter replaced", mutate: func(_ *App, tab *Tab) { tab.Adapter = nil }},
		{name: "source factory generation replaced", mutate: func(_ *App, tab *Tab) {
			tab.OutlineFactoryGeneration++
		}},
		{name: "path replaced", mutate: func(_ *App, tab *Tab) { tab.Doc.Path = "replacement.md" }},
		{name: "mode replaced", mutate: func(_ *App, tab *Tab) { tab.Mode = "source" }},
		{name: "tab identity replaced", mutate: func(_ *App, tab *Tab) { tab.ID++ }},
		{name: "buffer replaced", mutate: func(_ *App, tab *Tab) { tab.Doc = document.New("replacement") }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			a := newTestApp(t, t.TempDir())
			tab := newOutlineTab(a, "# 标题", "stable.md")
			tab.Mode = "preview"
			source := &outlineSourceStub{}
			source.snapshotFn = func() (editor.OutlineSnapshot, error) {
				return editor.OutlineSnapshot{Version: source.buffer.Version(), Items: []editor.OutlineItem{{
					ID: "heading", Text: "目标", Level: 1, Offset: 0,
				}}}, nil
			}
			source.resolveItemFn = func(string) (editor.OutlineItem, error) {
				return editor.OutlineItem{ID: "heading", Text: "目标", Level: 1, Offset: 0}, nil
			}
			installOutlineStub(a, tab, source)
			view := &outlineNavigatorView{}
			tab.View = view
			a.ShowOutline = true
			tester := ui.NewTester(a.View, 1180, 780)
			tester.Frame()
			if len(a.OutlineItems) != 1 {
				t.Fatalf("outline fixture did not initialize: %#v", a.OutlineItems)
			}
			view.onFlush = func() { test.mutate(a, tab) }

			if a.activateOutlineTarget(tab, "heading", false, "大纲定位") {
				t.Fatal("navigation succeeded after lifecycle changed during Flush")
			}
			if len(source.itemInputs) != 0 || len(view.navigations) != 0 {
				t.Fatalf("stale controller was reused: resolves=%#v navigations=%#v",
					source.itemInputs, view.navigations)
			}
			if !strings.Contains(a.Message, editor.ErrOutlineStale.Error()) {
				t.Fatalf("lifecycle change was not reported as stale: %q", a.Message)
			}
		})
	}
}

func TestOutlineSplitGraphNavigatesWithoutInventingSourcePosition(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	tab := newOutlineTab(a, "graph", "graph.mindmap")
	tab.Mode = "split"
	source := &outlineSourceStub{}
	source.resolveItemFn = func(string) (editor.OutlineItem, error) {
		return editor.OutlineItem{ID: "node-7", Text: "节点七", Level: 2, Offset: -1}, nil
	}
	createCalls, _ := installOutlineStub(a, tab, source)
	view := &outlineNavigatorView{}
	tab.View = view
	if err := tab.Doc.Buffer.SetSelection(1, 1); err != nil {
		t.Fatal(err)
	}
	if !a.activateOutlineTarget(tab, "node-7", false, "大纲定位") {
		t.Fatalf("split graph navigation failed: %s", a.Message)
	}
	if *createCalls != 0 || len(view.navigations) != 1 || view.navigations[0] != "node-7" {
		t.Fatalf("existing graph pane was not used: create=%d navigations=%#v", *createCalls, view.navigations)
	}
	if selection := tab.Doc.Buffer.Selection(); selection.Start() != 1 || selection.End() != 1 {
		t.Fatalf("negative graph offset moved the source caret: %#v", selection)
	}
	if !strings.Contains(a.Message, "已在图形中定位") || !strings.Contains(a.Message, "没有源码定位坐标") {
		t.Fatalf("split graph limitation was not disclosed: %q", a.Message)
	}
}

func TestOutlinePreviewNavigatesOnlyVisibleViewAndReplacesStaleMessage(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	tab := newOutlineTab(a, "preview text", "preview.md")
	tab.Mode = "preview"
	source := &outlineSourceStub{}
	source.resolveItemFn = func(string) (editor.OutlineItem, error) {
		return editor.OutlineItem{ID: "heading", Text: "目标", Level: 1, Offset: 0}, nil
	}
	createCalls, _ := installOutlineStub(a, tab, source)
	view := &outlineNavigatorView{}
	tab.View = view
	if err := tab.Doc.Buffer.SetSelection(3, 5); err != nil {
		t.Fatal(err)
	}
	before := tab.Doc.Buffer.Selection()
	a.Message = "上一条无关消息"

	if !a.activateOutlineTarget(tab, "heading", false, "大纲定位") {
		t.Fatalf("preview outline navigation failed: %s", a.Message)
	}
	if *createCalls != 0 || len(view.navigations) != 1 || view.navigations[0] != "heading" {
		t.Fatalf("visible preview was not the navigation target: create=%d navigations=%#v", *createCalls, view.navigations)
	}
	if tab.Mode != "preview" || tab.Doc.Buffer.Selection() != before {
		t.Fatalf("preview navigation changed mode or hidden source selection: mode=%q selection=%#v before=%#v",
			tab.Mode, tab.Doc.Buffer.Selection(), before)
	}
	if a.Message != "已定位：目标" {
		t.Fatalf("successful jump retained a stale status message: %q", a.Message)
	}
}

func TestOutlineEmptyButTruncatedSnapshotKeepsPartialNotice(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	tab := newOutlineTab(a, "# hidden", "partial.md")
	tab.Mode = "source"
	source := &outlineSourceStub{}
	source.snapshotFn = func() (editor.OutlineSnapshot, error) {
		return editor.OutlineSnapshot{Version: source.buffer.Version(), Truncated: true}, nil
	}
	installOutlineStub(a, tab, source)
	a.ShowOutline = true
	tester := ui.NewTester(a.View, 760, 480)
	partial := fmt.Sprintf("大纲内容不完整；最多显示 %d 项，较长文字已截短，完整内容仍在文档中。", editor.MaxOutlineItems)
	if !tester.HasText(partial) || tester.HasText("此文档没有可显示的大纲项。") {
		t.Fatalf("empty partial state was mislabeled: %#v", tester.Texts())
	}
}

func TestOutlineVirtualListReachesFarRowAndActivatesByKeyboard(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	tab := newOutlineTab(a, strings.Repeat("x", 3000), "far.md")
	tab.Mode = "source"
	source := &outlineSourceStub{}
	source.snapshotFn = func() (editor.OutlineSnapshot, error) {
		items := make([]editor.OutlineItem, 2000)
		for i := range items {
			items[i] = editor.OutlineItem{
				ID: fmt.Sprintf("item-%04d", i), Text: fmt.Sprintf("item-%04d", i),
				Level: i%6 + 1, Offset: i,
			}
		}
		return editor.OutlineSnapshot{Version: source.buffer.Version(), Items: items}, nil
	}
	source.resolveItemFn = func(id string) (editor.OutlineItem, error) {
		var i int
		if _, err := fmt.Sscanf(id, "item-%04d", &i); err != nil {
			return editor.OutlineItem{}, err
		}
		return editor.OutlineItem{ID: id, Text: id, Level: i%6 + 1, Offset: i}, nil
	}
	installOutlineStub(a, tab, source)
	var built []int
	a.outlineRowBuilt = func(i int) { built = append(built, i) }
	a.ShowOutline = true
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	tester.Frame()
	first, last := a.OutlineList.Visible()
	if first != 0 || last < 0 || last-first+1 >= 100 || len(built) == 0 || len(built) >= 200 {
		t.Fatalf("outline virtualization = visible %d..%d callbacks=%d", first, last, len(built))
	}
	clickTesterText(t, tester, "item-0000")
	tester.Frame()
	built = nil
	tester.Key(0, ui.KeyEnd)
	tester.Frame()
	tester.Frame()
	if a.OutlineSelected != 1999 || len(built) >= 200 {
		t.Fatalf("End did not reach the far outline row: selected=%d callbacks=%d", a.OutlineSelected, len(built))
	}
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	if len(source.itemInputs) == 0 || source.itemInputs[len(source.itemInputs)-1] != "item-1999" {
		t.Fatalf("far row Enter did not resolve its identity: %#v", source.itemInputs)
	}
	if selection := tab.Doc.Buffer.Selection(); selection.Start() != 1999 || selection.End() != 1999 {
		t.Fatalf("far outline reveal = %#v", selection)
	}
}

func TestOutlineSnapshotFailureDoesNotRetryEveryFrameAndReleaseInvalidates(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	tab := newOutlineTab(a, "# heading", "bounded.md")
	tab.Mode = "source"
	source := &outlineSourceStub{}
	source.snapshotFn = func() (editor.OutlineSnapshot, error) {
		return editor.OutlineSnapshot{}, errors.New("bounded parse failure")
	}
	installOutlineStub(a, tab, source)
	a.ShowOutline = true
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	tester.Frame()
	tester.Frame()
	if source.snapshotCalls != 1 || !tester.HasText("读取大纲失败：bounded parse failure") {
		t.Fatalf("snapshot failure retried or hidden: calls=%d texts=%#v", source.snapshotCalls, tester.Texts())
	}
	if err := tester.Click("重试大纲"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if source.snapshotCalls != 2 {
		t.Fatalf("explicit retry did not request a fresh snapshot: %d", source.snapshotCalls)
	}

	source.snapshotFn = func() (editor.OutlineSnapshot, error) {
		items := make([]editor.OutlineItem, editor.MaxOutlineItems+17)
		for i := range items {
			items[i] = editor.OutlineItem{ID: fmt.Sprintf("id-%d", i), Text: fmt.Sprintf("row-%d", i), Level: 1, Offset: 0}
		}
		return editor.OutlineSnapshot{Version: source.buffer.Version(), Items: items}, nil
	}
	if source.releases != 1 {
		t.Fatalf("source release count = %d", source.releases)
	}
	if err := a.ensureOutlineSnapshot(tab, "", true); err != nil {
		t.Fatal(err)
	}
	if len(a.OutlineItems) != editor.MaxOutlineItems || !a.OutlineTruncated {
		t.Fatalf("bounded outline = %d truncated=%v", len(a.OutlineItems), a.OutlineTruncated)
	}
	tab.View = &failingView{}
	if !a.setMode(tab, "preview") {
		t.Fatal("mode invalidation failed")
	}
	if a.OutlineSource != nil || source.releases != 2 {
		t.Fatalf("mode change retained outline source: source=%#v releases=%d", a.OutlineSource, source.releases)
	}
}

func TestOutlineUnsupportedStateIsExplicitForDrawio(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.NewDocument("drawio")
	a.ShowOutline = true
	tester := ui.NewTester(a.View, 760, 480)
	if !tester.HasText(editor.ErrOutlineUnsupported.Error()) {
		t.Fatalf("unsupported draw.io outline state is hidden: %#v", tester.Texts())
	}
}
