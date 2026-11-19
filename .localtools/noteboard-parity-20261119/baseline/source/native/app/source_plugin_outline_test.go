package app

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

const appSchema2OutlineManifest = `{
  "schema": 2,
  "id": "ini-outline",
  "name": "INI source outline",
  "version": "1.0.0",
  "apiVersion": 1,
  "extensions": [".ini"],
  "lineComment": "#",
  "outline": [
    {"prefix": "[", "suffix": "]", "level": 1},
    {"prefix": "# ", "suffix": "", "level": 2}
  ]
}`

func installAppPluginManifest(t *testing.T, a *App, name, manifest string) string {
	t.Helper()
	source := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(source, []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	if err := a.pluginManager.InstallFile(source); err != nil {
		t.Fatalf("install local plugin %s: %v", name, err)
	}
	return filepath.Join(a.configDir, "plugins", name)
}

func writeSchema2Source(t *testing.T, dir string) (string, []byte, string) {
	t.Helper()
	content := "  [重复]\r\n# 🙂 标题\r\n正文\r\n[重复]\r\n# 末尾"
	value := append([]byte{0xef, 0xbb, 0xbf}, []byte(content)...)
	path := filepath.Join(dir, "settings.ini")
	if err := os.WriteFile(path, value, 0600); err != nil {
		t.Fatal(err)
	}
	return path, value, content
}

func openInstalledOutline(t *testing.T) (*App, *Tab, string, []byte, string) {
	t.Helper()
	root := t.TempDir()
	a := newTestApp(t, filepath.Join(root, "config"))
	installAppPluginManifest(t, a, "ini-outline.json", appSchema2OutlineManifest)
	path, original, content := writeSchema2Source(t, root)
	a.Open(path)
	tab := a.Current()
	if tab == nil {
		t.Fatal("installed source document did not open")
	}
	return a, tab, path, original, content
}

func loadInstalledOutline(t *testing.T, a *App, width, height int) *ui.Tester {
	t.Helper()
	a.ShowOutline = true
	tester := ui.NewTester(a.View, width, height)
	tester.Frame()
	tester.Frame()
	if a.OutlineError != "" || len(a.OutlineItems) != 4 {
		t.Fatalf("installed outline did not load: error=%q items=%#v", a.OutlineError, a.OutlineItems)
	}
	return tester
}

func sourceOffset(t *testing.T, content, fragment string) int {
	t.Helper()
	index := strings.Index(content, fragment)
	if index < 0 {
		t.Fatalf("fragment %q is missing from fixture", fragment)
	}
	return text.UTF16Len(content[:index])
}

func TestInstalledSchema2OutlineUsesRealInstallAndFlushesPendingCompose(t *testing.T) {
	a, tab, _, _, content := openInstalledOutline(t)
	if tab.Adapter != nil || tab.OutlineFactory == nil || tab.Mode != "source" ||
		tab.MatchedPlugin.ID != "ini-outline" || tab.Editor.Highlight == nil {
		t.Fatalf("installed schema2 routing = %#v", tab)
	}
	if tab.SourceCapabilityBadge != "源码大纲 · 高亮" ||
		tab.SourceCapabilityDetail != "schema2 · 源码大纲（2 条逐行字面规则）+ 基础字符串/数字高亮与已配置行内规则；不识别多行注释/字符串上下文" {
		t.Fatalf("configured source capability descriptor = badge %q detail %q",
			tab.SourceCapabilityBadge, tab.SourceCapabilityDetail)
	}
	if !strings.Contains(tab.SyntaxNotice, "多行注释、字符串") {
		t.Fatalf("line-local limitation is not visible: %q", tab.SyntaxNotice)
	}
	tester := loadInstalledOutline(t, a, 1180, 780)
	wantTexts := []string{"重复", "🙂 标题", "重复", "末尾"}
	for i, want := range wantTexts {
		if a.OutlineItems[i].Text != want {
			t.Fatalf("outline item %d = %#v, want text %q; all=%#v", i, a.OutlineItems[i], want, a.OutlineItems)
		}
	}
	target := len(a.OutlineItems) - 1
	a.selectOutlineItem(a.OutlineItems[target].ID, true)
	oldID := a.OutlineItems[target].ID
	oldOffset := a.OutlineItems[target].Offset
	tester.Frame()
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	tab.Editor.Reveal(0, 0)
	insertion := "无害🙂\r\n"
	tester.Compose(insertion, len([]rune(insertion)))
	if _, composing := tab.Editor.MarkedRange(); !composing {
		t.Fatal("source composition was not pending before installed-outline activation")
	}
	// The captured row may disappear from the regenerated list while the
	// composition changes the source version. Activation still submits the ID
	// captured before editing; the provider owns journal-based revalidation.
	if !a.activateOutlineTarget(tab, oldID, false, "大纲定位") {
		t.Fatalf("installed outline activation failed: %s", a.Message)
	}
	if _, composing := tab.Editor.MarkedRange(); composing {
		t.Fatal("installed outline activation did not flush the native composition")
	}
	if !strings.HasPrefix(tab.Doc.Buffer.String(), insertion) {
		t.Fatalf("flushed composition is missing from source: %q", tab.Doc.Buffer.String())
	}
	wantOffset := oldOffset + text.UTF16Len(insertion)
	if selection := tab.Doc.Buffer.Selection(); selection.Start() != wantOffset || selection.End() != wantOffset {
		t.Fatalf("resolved target selection = %#v, want old offset %d shifted by %d to %d; old ID %q",
			selection, oldOffset, text.UTF16Len(insertion), wantOffset, oldID)
	}
	selected, ok := a.outlineRowAt(a.OutlineSelected)
	if !ok || selected.Text != "末尾" || selected.ID == oldID {
		t.Fatalf("navigation did not adopt current target identity: selected=%#v old=%q", selected, oldID)
	}
	if tab.Adapter != nil || tab.View != nil {
		t.Fatalf("source navigation mounted a rich view: adapter=%#v view=%#v", tab.Adapter, tab.View)
	}
	if a.OutlineItems[0].Offset != sourceOffset(t, content, "  [重复]")+text.UTF16Len(insertion) {
		t.Fatalf("first outline offset was not refreshed after composition: %#v", a.OutlineItems[0])
	}
}

func TestInstalledSchema2UnmodifiedSavePreservesBOMCRLFAndReadonlyNavigation(t *testing.T) {
	a, tab, path, original, content := openInstalledOutline(t)
	tester := loadInstalledOutline(t, a, 1180, 780)
	a.OutlineSelected = 3
	if !a.activateSelectedOutlineItem() {
		t.Fatalf("installed readonly outline activation failed: %s", a.Message)
	}
	if selection := tab.Doc.Buffer.Selection(); selection.Start() != sourceOffset(t, content, "# 末尾") {
		t.Fatalf("far source reveal = %#v", selection)
	}
	if !a.Save(tab, false) {
		t.Fatalf("unmodified installed source save failed: %s", a.Message)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(saved, original) {
		t.Fatalf("unmodified save changed BOM/CRLF bytes:\n got %x\nwant %x", saved, original)
	}

	tab.Doc.ReadOnly = true
	tab.Editor.ReadOnly = true
	version := tab.Doc.Buffer.Version()
	textBefore := tab.Doc.Buffer.String()
	tester.Frame()
	a.OutlineSelected = 1
	if !a.activateSelectedOutlineItem() {
		t.Fatalf("readonly installed outline navigation failed: %s", a.Message)
	}
	if tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.String() != textBefore {
		t.Fatalf("readonly navigation mutated source: version=%d/%d text=%q/%q",
			tab.Doc.Buffer.Version(), version, tab.Doc.Buffer.String(), textBefore)
	}
}

func TestInstalledSchema2EditedSavePreservesBOMAndCRLF(t *testing.T) {
	a, tab, path, original, content := openInstalledOutline(t)
	if !bytes.HasPrefix(original, []byte{0xef, 0xbb, 0xbf}) {
		t.Fatal("fixture lost its UTF-8 BOM before the save test")
	}
	start := sourceOffset(t, content, "正文")
	tab.Editor.ReplaceText(&ui.TextInputRange{
		Start: start, End: start + text.UTF16Len("正文"),
	}, "正文已编辑🙂")
	if !tab.Doc.Dirty() {
		t.Fatal("source edit was not marked dirty")
	}
	if !a.Save(tab, false) {
		t.Fatalf("edited installed source save failed: %s", a.Message)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := append([]byte{0xef, 0xbb, 0xbf}, []byte(strings.Replace(content, "正文", "正文已编辑🙂", 1))...)
	if !bytes.Equal(saved, want) {
		t.Fatalf("edited save changed BOM/CRLF or intended text:\n got %x\nwant %x", saved, want)
	}
}

func TestInstalledSchema2DisableEnableAndEmptySchemaDoNotLeaveStaleCapability(t *testing.T) {
	a, tab, _, _, _ := openInstalledOutline(t)
	loadInstalledOutline(t, a, 760, 480)
	firstGeneration := tab.OutlineFactoryGeneration
	oldID := a.OutlineItems[3].ID

	a.setPluginEnabled("ini-outline", false)
	if tab.Adapter != nil || tab.OutlineFactory != nil || a.OutlineSource != nil ||
		tab.MatchedPlugin.ID != "code" || tab.Mode != "source" {
		t.Fatalf("disable retained installed outline capability: plugin=%q adapter=%#v factory=%v source=%#v",
			tab.MatchedPlugin.ID, tab.Adapter, tab.OutlineFactory != nil, a.OutlineSource)
	}
	selection := tab.Doc.Buffer.Selection()
	if a.activateOutlineTarget(tab, oldID, false, "大纲定位") {
		t.Fatal("disabled installed plugin still resolved its old outline ID")
	}
	if tab.Doc.Buffer.Selection() != selection {
		t.Fatal("disabled plugin stale navigation moved the source caret")
	}

	a.setPluginEnabled("ini-outline", true)
	if tab.Adapter != nil || tab.OutlineFactory == nil ||
		tab.OutlineFactoryGeneration <= firstGeneration || tab.MatchedPlugin.ID != "ini-outline" {
		t.Fatalf("re-enable did not recreate a new installed factory: adapter=%#v factory=%v generation=%d/%d plugin=%q",
			tab.Adapter, tab.OutlineFactory != nil, tab.OutlineFactoryGeneration, firstGeneration, tab.MatchedPlugin.ID)
	}
	if tab.SourceCapabilityBadge != "源码大纲 · 高亮" ||
		tab.SourceCapabilityDetail != "schema2 · 源码大纲（2 条逐行字面规则）+ 基础字符串/数字高亮与已配置行内规则；不识别多行注释/字符串上下文" {
		t.Fatalf("re-enable retained a stale capability descriptor: %q / %q",
			tab.SourceCapabilityBadge, tab.SourceCapabilityDetail)
	}
	loadInstalledOutline(t, a, 760, 480)

	emptyManifest := `{
  "schema": 2,
  "id": "empty-outline",
  "name": "Empty outline metadata",
  "version": "1.0.0",
  "apiVersion": 1,
  "extensions": [".cfgx"],
  "outline": []
}`
	installAppPluginManifest(t, a, "empty-outline.json", emptyManifest)
	emptyPath := filepath.Join(t.TempDir(), "empty.cfgx")
	if err := os.WriteFile(emptyPath, []byte("[not-an-outline]\nkey = \"quoted\" 42\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a.Open(emptyPath)
	empty := a.Current()
	if empty == nil || empty.MatchedPlugin.ID != "empty-outline" || empty.Adapter != nil ||
		empty.OutlineFactory != nil || empty.Editor.Highlight == nil {
		t.Fatalf("schema2 empty outline gained a capability: %#v", empty)
	}
	spans := empty.Editor.Highlight(`key = "quoted" 42`)
	highlighted := ""
	for _, span := range spans {
		highlighted += span.Text
	}
	if highlighted != `key = "quoted" 42` || len(spans) < 3 {
		t.Fatalf("schema2 baseline highlight = %#v", spans)
	}
	if empty.SourceCapabilityBadge != "仅行内高亮" ||
		empty.SourceCapabilityDetail != "schema2 · 未声明源码大纲；提供基础字符串/数字高亮（未配置额外高亮规则）" {
		t.Fatalf("empty schema2 capability descriptor = %q / %q",
			empty.SourceCapabilityBadge, empty.SourceCapabilityDetail)
	}
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	tester.Frame()
	if !tester.HasText(editor.ErrOutlineUnsupported.Error()) ||
		!tester.HasText("仅行内高亮") {
		t.Fatalf("empty schema2 capability is misleading or hidden: %#v", tester.Texts())
	}
}

func TestInstalledSourceOnlyFlushFailureRetainsOutlineState(t *testing.T) {
	a, tab, _, _, _ := openInstalledOutline(t)
	loadInstalledOutline(t, a, 760, 480)
	a.OutlineSelected = 0
	view := &failingView{flushErr: errors.New("unfinished source draft")}
	tab.View = view
	source := a.OutlineSource
	generation := a.outlineGeneration
	factoryGeneration := tab.OutlineFactoryGeneration
	selected := a.OutlineSelected
	items := append([]editor.OutlineItem(nil), a.OutlineItems...)
	selection := tab.Doc.Buffer.Selection()
	version := tab.Doc.Buffer.Version()

	if a.activateSelectedOutlineItem() {
		t.Fatal("source-only navigation succeeded after its Flush failed")
	}
	if view.flushes != 1 || tab.View != view || tab.Mode != "source" ||
		tab.Adapter != nil || tab.OutlineFactory == nil ||
		a.OutlineSource != source || a.outlineGeneration != generation ||
		tab.OutlineFactoryGeneration != factoryGeneration ||
		a.OutlineSelected != selected || fmt.Sprintf("%#v", a.OutlineItems) != fmt.Sprintf("%#v", items) ||
		tab.Doc.Buffer.Selection() != selection || tab.Doc.Buffer.Version() != version {
		t.Fatalf("failed source Flush mutated lifecycle: flush=%d view=%#v mode=%q adapter=%#v factory=%v source=%#v gen=%d/%d selected=%d items=%#v selection=%#v/%#v version=%d/%d message=%q",
			view.flushes, tab.View, tab.Mode, tab.Adapter, tab.OutlineFactory != nil,
			a.OutlineSource, a.outlineGeneration, generation, a.OutlineSelected, a.OutlineItems,
			tab.Doc.Buffer.Selection(), selection, tab.Doc.Buffer.Version(), version, a.Message)
	}
	if !strings.Contains(a.Message, "unfinished source draft") {
		t.Fatalf("Flush failure was hidden: %q", a.Message)
	}
}

func TestInstalledSourceOnlyFactoryChangeDuringFlushRejectsOldController(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	tab := newOutlineTab(a, "[目标]\n", "factory.ini")
	tab.Mode = "source"
	tab.Adapter = nil
	tab.OutlineFactoryGeneration = 41

	first := &outlineSourceStub{}
	first.snapshotFn = func() (editor.OutlineSnapshot, error) {
		return editor.OutlineSnapshot{Version: first.buffer.Version(), Items: []editor.OutlineItem{{
			ID: "target", Text: "目标", Level: 1, Offset: 0,
		}}}, nil
	}
	firstCalls := 0
	installSourceOutlineFactory(tab, first, &firstCalls)
	a.ShowOutline = true
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	tester.Frame()
	if firstCalls != 1 || len(a.OutlineItems) != 1 {
		t.Fatalf("source-only factory did not initialize: calls=%d items=%#v", firstCalls, a.OutlineItems)
	}

	secondCalls := 0
	second := &outlineSourceStub{}
	view := &outlineNavigatorView{}
	view.onFlush = func() {
		tab.OutlineFactory = func(editor.Host) editor.OutlineSource {
			secondCalls++
			return second
		}
		tab.OutlineFactoryGeneration++
	}
	tab.View = view
	selection := tab.Doc.Buffer.Selection()
	oldSource := a.OutlineSource
	oldControllerGeneration := a.outlineGeneration

	if a.activateOutlineTarget(tab, "target", false, "大纲定位") {
		t.Fatal("navigation survived a source-only factory change during Flush")
	}
	if len(first.itemInputs) != 0 || secondCalls != 0 ||
		tab.View != view || a.OutlineSource != oldSource ||
		a.outlineGeneration != oldControllerGeneration ||
		tab.Doc.Buffer.Selection() != selection ||
		!strings.Contains(a.Message, editor.ErrOutlineStale.Error()) {
		t.Fatalf("factory change reused a stale controller: firstResolve=%#v secondCalls=%d view=%#v source=%#v gen=%d/%d selection=%#v/%#v message=%q",
			first.itemInputs, secondCalls, tab.View, a.OutlineSource,
			a.outlineGeneration, oldControllerGeneration,
			tab.Doc.Buffer.Selection(), selection, a.Message)
	}
}

func TestInstalledSourceOutlinePartialWithNoItemsShowsByteAndLineGuidance(t *testing.T) {
	root := t.TempDir()
	a := newTestApp(t, filepath.Join(root, "config"))
	installAppPluginManifest(t, a, "ini-outline.json", appSchema2OutlineManifest)
	path := filepath.Join(root, "oversize.ini")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", (2<<20)+32)), 0600); err != nil {
		t.Fatal(err)
	}
	a.Open(path)
	a.ShowOutline = true
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	tester.Frame()
	if a.OutlineError != "" || !a.OutlineTruncated || len(a.OutlineItems) != 0 {
		t.Fatalf("byte-limited source outline = error=%q truncated=%v items=%#v",
			a.OutlineError, a.OutlineTruncated, a.OutlineItems)
	}
	if !tester.HasText(outlineTruncationNotice(a.Current())) ||
		tester.HasText("此文档没有可显示的大纲项。") {
		t.Fatalf("zero-item partial source outline guidance is missing or misleading: %#v", tester.Texts())
	}
	notice := outlineTruncationNotice(a.Current())
	for _, want := range []string{"内容或标题文字已省略", "2 MiB", "65,536 行", "2048 项", "512 字节", "完整内容请看源码"} {
		if !strings.Contains(notice, want) {
			t.Fatalf("partial source notice %q does not contain %q", notice, want)
		}
	}
	if strings.Contains(notice, "达到字节、行或条目上限") {
		t.Fatalf("partial source notice misdiagnoses label-only clipping as an omitted-entry limit: %q", notice)
	}
}

func TestInstalledSourceOutlineNarrowHeaderKeepsTabEditorAndStatusUsable(t *testing.T) {
	a, _, _, _, _ := openInstalledOutline(t)
	a.ShowOutline = true
	a.Dark = true
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	tester.Frame()
	a.selectOutlineItem(a.OutlineItems[0].ID, true)
	if !a.activateSelectedOutlineItem() {
		t.Fatalf("narrow installed outline activation failed: %s", a.Message)
	}
	tester.Frame()
	tester.Advance(time.Second)
	tester.Frame()

	rects := make(map[string]ui.Rect)
	for _, label := range []string{
		"settings.ini", "关闭 settings.ini",
		"INI source outline · ini-outline", "源码大纲 · 高亮",
		"文档编辑区", "已定位：重复",
	} {
		rect, ok := tester.Find(label)
		if !ok {
			t.Fatalf("%q is missing at 760x480: %#v", label, tester.Texts())
		}
		if rect.W <= 0 || rect.H <= 0 || rect.X < 0 || rect.Y < 0 ||
			rect.X+rect.W > 760 || rect.Y+rect.H > 480 {
			t.Fatalf("%q is clipped at 760x480: %#v", label, rect)
		}
		rects[label] = rect
	}
	tab := rects["settings.ini"]
	close := rects["关闭 settings.ini"]
	pluginLabel := rects["INI source outline · ini-outline"]
	badge := rects["源码大纲 · 高亮"]
	editor := rects["文档编辑区"]
	status := rects["已定位：重复"]
	if tab.Y < close.Y || tab.Y+tab.H > close.Y+close.H ||
		pluginLabel.Y < close.Y+close.H || badge.Y < close.Y+close.H ||
		pluginLabel.W < 100 || badge.W > 150 ||
		editor.Y < badge.Y+badge.H || editor.H < 100 ||
		editor.Y+editor.H > status.Y || status.Y+status.H > 480 {
		t.Fatalf("narrow source layout overlaps: tab=%#v close=%#v plugin=%#v badge=%#v editor=%#v status=%#v",
			tab, close, pluginLabel, badge, editor, status)
	}
}

func TestInstalledSourceOnlyCannotEnterRichModesOrExport(t *testing.T) {
	a, tab, _, _, _ := openInstalledOutline(t)
	loadInstalledOutline(t, a, 1180, 780)
	if a.setMode(tab, "preview") {
		t.Fatal("installed source-only format entered preview mode")
	}
	if a.setMode(tab, "split") {
		t.Fatal("installed source-only format entered split mode")
	}
	if tab.Mode != "source" || tab.Adapter != nil || tab.View != nil {
		t.Fatalf("rich mode rejection mutated source routing: mode=%q adapter=%#v view=%#v",
			tab.Mode, tab.Adapter, tab.View)
	}
	a.exportCurrent()
	if tab.View != nil || !strings.Contains(a.Message, "只提供源码编辑") {
		t.Fatalf("source-only export routed ambiguously: message=%q view=%#v", a.Message, tab.View)
	}
}

func TestInstalledSchema1AndEnabledBuiltinDoNotUseOutlineFactory(t *testing.T) {
	root := t.TempDir()
	a := newTestApp(t, filepath.Join(root, "config"))
	installAppPluginManifest(t, a, "legacy.json", `{
  "schema": 1,
  "id": "legacy-source",
  "name": "Legacy source",
  "version": "1.0.0",
  "apiVersion": 1,
  "extensions": [".legacyx"],
  "lineComment": "#"
}`)
	installAppPluginManifest(t, a, "markdown-shadow.json", `{
  "schema": 2,
  "id": "markdown-shadow",
  "name": "Markdown shadow",
  "version": "1.0.0",
  "apiVersion": 1,
  "extensions": [".md"],
  "outline": [{"prefix": "# ", "suffix": "", "level": 1}]
}`)
	legacyPath := filepath.Join(root, "legacy.legacyx")
	if err := os.WriteFile(legacyPath, []byte("# not outlined\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a.Open(legacyPath)
	legacy := a.Current()
	if legacy == nil || legacy.Adapter != nil || legacy.OutlineFactory != nil ||
		legacy.MatchedPlugin.ID != "legacy-source" {
		t.Fatalf("schema1 acquired an outline capability: %#v", legacy)
	}

	markdownPath := filepath.Join(root, "builtin.md")
	if err := os.WriteFile(markdownPath, []byte("# Builtin wins\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a.Open(markdownPath)
	markdown := a.Current()
	if markdown == nil || markdown.Adapter == nil || markdown.Adapter.ID != "markdown" ||
		markdown.OutlineFactory != nil || markdown.MatchedPlugin.ID != "markdown" {
		t.Fatalf("installed shadow replaced a specific enabled builtin: %#v", markdown)
	}
}

func TestInstalledSchema2ChangedRulesRejectCapturedID(t *testing.T) {
	a, tab, _, _, _ := openInstalledOutline(t)
	loadInstalledOutline(t, a, 1180, 780)
	oldID := a.OutlineItems[3].ID
	selection := tab.Doc.Buffer.Selection()
	changed := strings.Replace(appSchema2OutlineManifest, `"prefix": "[", "suffix": "]"`, `"prefix": "[[", "suffix": "]]"`, 1)
	installed := filepath.Join(a.configDir, "plugins", "ini-outline.json")
	if err := os.WriteFile(installed, []byte(changed), 0600); err != nil {
		t.Fatal(err)
	}
	a.refreshPlugins()
	if tab.OutlineFactory == nil || tab.MatchedPlugin.ID != "ini-outline" {
		t.Fatalf("changed-rule refresh lost installed routing: plugin=%q factory=%v message=%q",
			tab.MatchedPlugin.ID, tab.OutlineFactory != nil, a.Message)
	}
	if a.activateOutlineTarget(tab, oldID, false, "大纲定位") {
		t.Fatal("captured outline ID survived installed rules change")
	}
	if !strings.Contains(a.Message, editor.ErrOutlineStale.Error()) {
		t.Fatalf("changed rule failure is not explicit: %q", a.Message)
	}
	if tab.Doc.Buffer.Selection() != selection {
		t.Fatalf("stale-ID failure moved source selection: %#v/%#v", tab.Doc.Buffer.Selection(), selection)
	}
}

func TestInstalledOutlineFarRowActivatesByKeyboard(t *testing.T) {
	root := t.TempDir()
	a := newTestApp(t, filepath.Join(root, "config"))
	installAppPluginManifest(t, a, "ini-outline.json", appSchema2OutlineManifest)
	var content strings.Builder
	for i := 0; i < 2000; i++ {
		fmt.Fprintf(&content, "[条目 %04d]\r\n", i)
	}
	path := filepath.Join(root, "far.ini")
	if err := os.WriteFile(path, []byte(content.String()), 0600); err != nil {
		t.Fatal(err)
	}
	a.Open(path)
	tab := a.Current()
	a.ShowOutline = true
	tester := ui.NewTester(a.View, 760, 480)
	var built []int
	a.outlineRowBuilt = func(i int) { built = append(built, i) }
	tester.Frame()
	tester.Frame()
	if len(a.OutlineItems) != 2000 || a.OutlineTruncated {
		t.Fatalf("far fixture bounds = items=%d truncated=%v", len(a.OutlineItems), a.OutlineTruncated)
	}
	first, last := a.OutlineList.Visible()
	if first != 0 || last < 0 || last-first+1 >= 100 || len(built) == 0 || len(built) >= 200 {
		t.Fatalf("installed outline virtualization = visible %d..%d callbacks=%d", first, last, len(built))
	}
	clickTesterText(t, tester, "条目 0000")
	tester.Frame()
	built = nil
	tester.Key(0, ui.KeyEnd)
	tester.Frame()
	tester.Frame()
	if a.OutlineSelected != 1999 || len(built) >= 200 {
		t.Fatalf("End did not reach far installed row: selected=%d callbacks=%d", a.OutlineSelected, len(built))
	}
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	want := sourceOffset(t, content.String(), "[条目 1999]")
	if selection := tab.Doc.Buffer.Selection(); selection.Start() != want || selection.End() != want {
		t.Fatalf("far installed activation = %#v, want %d", selection, want)
	}
}
