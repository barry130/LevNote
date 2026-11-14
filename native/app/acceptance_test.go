package app

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/search"
	"levnote.local/native/ui/editor"
)

const acceptanceMarkdown = `# 原生写作

轻笺把 Markdown、代码、思维导图和图纸放在同一个本地工作区。

## 今天的记录

- 自研文本缓冲与撤销历史
- 原生画布编辑
- 声明式格式插件

> 文件留在本机，降级和失败始终明确显示。

` + "```go" + `
func localOnly() bool {
	return true
}
` + "```" + `

### 下一步

继续完善恢复、搜索和真实桌面输入验收。
`

const acceptanceMindmap = `{
  "format": "levnote.mindmap",
  "version": 1,
  "layout": "logicalStructure",
  "root": {
    "id": "root",
    "topic": "轻笺",
    "children": [
      {"id": "write", "topic": "写作", "children": []},
      {"id": "code", "topic": "代码", "children": []},
      {"id": "graph", "topic": "图谱", "children": []}
    ]
  }
}
`

const acceptanceDrawio = `<mxfile host="levnote"><diagram id="p1" name="本地流程"><mxGraphModel><root><mxCell id="open" value="打开文件" style="rounded=1;fillColor=#dae8fc;strokeColor=#6c8ebf;" vertex="1"><mxGeometry x="20" y="20" width="140" height="64" as="geometry"/></mxCell><mxCell id="edit" value="原生编辑" style="rounded=1;fillColor=#fff2cc;strokeColor=#d6b656;" vertex="1"><mxGeometry x="230" y="20" width="140" height="64" as="geometry"/></mxCell><mxCell id="save" value="原子保存" style="ellipse;fillColor=#d5e8d4;strokeColor=#82b366;" vertex="1"><mxGeometry x="440" y="20" width="140" height="64" as="geometry"/></mxCell><mxCell id="e1" style="edgeStyle=orthogonalEdgeStyle;endArrow=block;" edge="1" source="open" target="edit"><mxGeometry relative="1" as="geometry"/></mxCell><mxCell id="e2" style="edgeStyle=orthogonalEdgeStyle;endArrow=block;" edge="1" source="edit" target="save"><mxGeometry relative="1" as="geometry"/></mxCell></root></mxGraphModel></diagram></mxfile>`

const acceptanceCollapsedMindmap = `{
  "format": "levnote.mindmap",
  "version": 1,
  "layout": "logicalStructure",
  "root": {
    "id": "root",
    "topic": "产品规划",
    "collapsed": true,
    "children": [
      {
        "id": "research",
        "topic": "用户研究",
        "collapsed": true,
        "children": [
          {"id": "interview", "topic": "访谈记录", "children": []},
          {"id": "insight", "topic": "关键发现", "children": []}
        ]
      },
      {"id": "delivery", "topic": "交付节奏", "children": []}
    ]
  }
}`

type acceptanceFixture struct {
	root, workspace, config                                 string
	source, markdown, longOutline, partialOutline           string
	mindmap, collapsedMindmap, drawio, pluginPath, readonly string
	sourceOutline, farSourceOutline, encoding               string
}

func writeAcceptanceFixture(t *testing.T, root string) acceptanceFixture {
	t.Helper()
	fixture := acceptanceFixture{
		root:             root,
		workspace:        filepath.Join(root, "workspace"),
		config:           filepath.Join(root, "config"),
		source:           filepath.Join(root, "workspace", "source.go"),
		markdown:         filepath.Join(root, "workspace", "notes.md"),
		longOutline:      filepath.Join(root, "workspace", "outline-long.md"),
		partialOutline:   filepath.Join(root, "workspace", "outline-partial.md"),
		mindmap:          filepath.Join(root, "workspace", "ideas.mindmap"),
		collapsedMindmap: filepath.Join(root, "workspace", "planning.mindmap"),
		drawio:           filepath.Join(root, "workspace", "flow.drawio"),
		pluginPath:       filepath.Join(root, "workspace", "settings.toml"),
		readonly:         filepath.Join(root, "workspace", "readonly.txt"),
		sourceOutline:    filepath.Join(root, "workspace", "settings.ini"),
		farSourceOutline: filepath.Join(root, "workspace", "source-outline-far.ini"),
		encoding:         filepath.Join(root, "workspace", "legacy.txt"),
	}
	if err := os.MkdirAll(filepath.Join(fixture.config, "plugins"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(fixture.workspace, 0755); err != nil {
		t.Fatal(err)
	}
	writeAcceptanceFile(t, filepath.Join(fixture.config, "plugins", "toml.json"), `{"schema":1,"id":"toml","name":"TOML","version":"1.0.0","apiVersion":1,"extensions":[".toml"],"lineComment":"#","blockStart":"/*","blockEnd":"*/","keywords":["true","false","table"]}`)
	writeAcceptanceFile(t, filepath.Join(fixture.config, "plugins", "ini-outline.json"), `{
  "schema": 2,
  "id": "ini-outline",
  "name": "INI 源码大纲",
  "version": "1.0.0",
  "apiVersion": 1,
  "extensions": [".ini"],
  "lineComment": "#",
  "outline": [
    {"prefix": "[", "suffix": "]", "level": 1},
    {"prefix": "# ", "suffix": "", "level": 2}
  ]
}`)
	writeAcceptanceFile(t, fixture.source, "package main\n\nfunc main() {\n\t// 原生源码视图\n\tprintln(\"local only\", true, 42)\n}\n")
	writeAcceptanceFile(t, fixture.markdown, acceptanceMarkdown)
	var longOutline strings.Builder
	for i := 1; i <= 90; i++ {
		fmt.Fprintf(&longOutline, "## 第 %02d 章 · 本地知识整理\n\n这一节用于验证长文档大纲的虚拟列表与远端键盘定位。\n\n", i)
	}
	writeAcceptanceFile(t, fixture.longOutline, longOutline.String())
	var partialOutline strings.Builder
	partialOutline.WriteString("# 大纲数量超过上限\n\n")
	for i := 1; i <= editor.MaxOutlineItems+2; i++ {
		fmt.Fprintf(&partialOutline, "## 部分条目 %04d\n\n正文。\n\n", i)
	}
	writeAcceptanceFile(t, fixture.partialOutline, partialOutline.String())
	writeAcceptanceFile(t, fixture.mindmap, acceptanceMindmap)
	writeAcceptanceFile(t, fixture.collapsedMindmap, acceptanceCollapsedMindmap)
	writeAcceptanceFile(t, fixture.drawio, acceptanceDrawio)
	writeAcceptanceFile(t, fixture.pluginPath, "enabled = true\n# local syntax metadata\n")
	sourceOutline := append([]byte{0xef, 0xbb, 0xbf}, []byte("  [重复]\r\n# 🙂 标题\r\n正文\r\n[重复]\r\n# 末尾\r\n")...)
	if err := os.WriteFile(fixture.sourceOutline, sourceOutline, 0600); err != nil {
		t.Fatal(err)
	}
	var farSourceOutline strings.Builder
	farSourceOutline.WriteString("[重复]\r\n# 🙂 重复标题\r\n")
	for i := 1; i <= 90; i++ {
		fmt.Fprintf(&farSourceOutline, "[第 %02d 章 · 本地配置]\r\n# 章节说明 %02d 🙂\r\n", i, i)
	}
	farSourceOutline.WriteString("[重复]\r\n# 末尾目标 🙂\r\n")
	if err := os.WriteFile(fixture.farSourceOutline, []byte(farSourceOutline.String()), 0600); err != nil {
		t.Fatal(err)
	}
	writeAcceptanceFile(t, fixture.readonly, "只读文本仍可预览和另存为。\n文件内容不会被覆盖。\n")
	if err := os.WriteFile(fixture.encoding, []byte{0xc4, 0xe3, 0xba, 0xc3}, 0600); err != nil {
		t.Fatal(err)
	}
	return fixture
}

func TestAcceptanceRuntimeFixture(t *testing.T) {
	root := os.Getenv("LEVNOTE_ACCEPTANCE_RUNTIME_DIR")
	if root == "" {
		t.Skip("set LEVNOTE_ACCEPTANCE_RUNTIME_DIR to materialize the runtime fixture")
	}
	fixture := writeAcceptanceFixture(t, root)
	for _, path := range []string{fixture.source, fixture.markdown, fixture.mindmap, fixture.drawio} {
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			t.Fatalf("runtime fixture document %q is unavailable: %v", path, err)
		}
	}
	t.Logf("runtime fixture workspace: %s", fixture.workspace)
}

func TestAcceptanceScreenshots(t *testing.T) {
	out := os.Getenv("LEVNOTE_ACCEPTANCE_SCREENSHOT_DIR")
	if out == "" {
		t.Skip("set LEVNOTE_ACCEPTANCE_SCREENSHOT_DIR for visual acceptance")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	fixture := writeAcceptanceFixture(t, t.TempDir())
	workspace, config := fixture.workspace, fixture.config
	sourcePath, markdownPath, mindmapPath := fixture.source, fixture.markdown, fixture.mindmap
	longOutlinePath, partialOutlinePath, collapsedMindmapPath := fixture.longOutline, fixture.partialOutline, fixture.collapsedMindmap
	drawioPath, pluginPath, readonlyPath := fixture.drawio, fixture.pluginPath, fixture.readonly
	encodingPath := fixture.encoding

	newApp := func() *App {
		a := newTestApp(t, config)
		a.Root = workspace
		a.refreshTreeNow()
		return a
	}

	sourceLight := newApp()
	sourceLight.Open(sourcePath)
	captureAcceptanceApp(t, filepath.Join(out, "source-light.png"), sourceLight, 1180, 780)
	sourceDark := newApp()
	sourceDark.Open(sourcePath)
	sourceDark.Dark = true
	captureAcceptanceApp(t, filepath.Join(out, "source-dark.png"), sourceDark, 1180, 780)

	markdownPreview := newApp()
	markdownPreview.Open(markdownPath)
	markdownPreview.Current().Mode = "preview"
	captureAcceptanceApp(t, filepath.Join(out, "markdown-preview.png"), markdownPreview, 1180, 780)
	markdownSplit := newApp()
	markdownSplit.Open(markdownPath)
	if !markdownSplit.setMode(markdownSplit.Current(), "split") {
		t.Fatal("could not enter Markdown split mode")
	}
	captureAcceptanceApp(t, filepath.Join(out, "markdown-split.png"), markdownSplit, 1180, 780)
	markdownSplitDraft := newApp()
	markdownSplitDraft.Open(markdownPath)
	if !markdownSplitDraft.setMode(markdownSplitDraft.Current(), "split") {
		t.Fatal("could not enter Markdown split mode for active-draft evidence")
	}
	splitTester := ui.NewTester(markdownSplitDraft.View, 1180, 780)
	splitTester.Frame()
	beginRealMarkdownComposition(t, markdownSplitDraft.Current(), splitTester)
	captureAcceptanceTester(t, filepath.Join(out, "markdown-split-active-draft.png"), splitTester)
	markdownConflictNarrow := newApp()
	markdownConflictNarrow.Open(markdownPath)
	if !markdownConflictNarrow.setMode(markdownConflictNarrow.Current(), "split") {
		t.Fatal("could not enter narrow Markdown split mode for conflict evidence")
	}
	conflictTester := ui.NewTester(markdownConflictNarrow.View, 760, 480)
	conflictTester.Frame()
	conflictTab := markdownConflictNarrow.Current()
	beginRealMarkdownComposition(t, conflictTab, conflictTester)
	if err := conflictTab.Doc.Buffer.ReplaceUTF16(2, 4, "外部"); err != nil {
		t.Fatal(err)
	}
	if markdownConflictNarrow.flushTab(conflictTab, "验收冲突") {
		t.Fatal("overlapping Markdown draft unexpectedly flushed")
	}
	conflictTester.Frame()
	captureAcceptanceTester(t, filepath.Join(out, "markdown-conflict-760x480.png"), conflictTester)
	markdownDark := newApp()
	markdownDark.Open(markdownPath)
	markdownDark.Current().Mode = "preview"
	markdownDark.Dark = true
	captureAcceptanceApp(t, filepath.Join(out, "markdown-dark.png"), markdownDark, 1180, 780)

	captureMarkdownOutline := func(name string, mode string, dark bool, width, height int) {
		a := newApp()
		a.Open(markdownPath)
		a.ShowOutline = true
		if mode != "preview" && !a.setMode(a.Current(), mode) {
			t.Fatalf("could not enter Markdown %s mode for %s", mode, name)
		}
		a.Dark = dark
		tester := ui.NewTester(a.View, width, height)
		tester.Frame()
		if len(a.OutlineItems) == 0 {
			t.Fatalf("Markdown %s screenshot has no outline items", name)
		}
		a.selectOutlineItem(a.OutlineItems[min(1, len(a.OutlineItems)-1)].ID, true)
		if !a.activateSelectedOutlineItem() {
			t.Fatalf("Markdown %s screenshot outline activation failed: %s", name, a.Message)
		}
		a.OutlineList.ScrollIntoView(max(0, a.OutlineSelected))
		tester.Frame()
		tester.Advance(time.Second)
		tester.Frame()
		captureAcceptanceTester(t, filepath.Join(out, name), tester)
	}
	captureMarkdownOutline("outline-markdown-preview-1180x780.png", "preview", false, 1180, 780)
	captureMarkdownOutline("outline-markdown-source-1180x780.png", "source", false, 1180, 780)
	captureMarkdownOutline("outline-markdown-split-1180x780.png", "split", false, 1180, 780)
	captureMarkdownOutline("outline-markdown-preview-dark-1180x780.png", "preview", true, 1180, 780)
	captureMarkdownOutline("outline-markdown-split-dark-760x480.png", "split", true, 760, 480)

	outlineFar := newApp()
	outlineFar.Open(longOutlinePath)
	outlineFar.ShowOutline = true
	farTester := ui.NewTester(outlineFar.View, 760, 480)
	farTester.Frame()
	if len(outlineFar.OutlineItems) < 2 {
		t.Fatalf("long Markdown outline is incomplete: %#v", outlineFar.OutlineItems)
	}
	outlineFar.selectOutlineItem(outlineFar.OutlineItems[len(outlineFar.OutlineItems)-1].ID, false)
	if !outlineFar.activateSelectedOutlineItem() {
		t.Fatalf("far Markdown outline activation failed: %s", outlineFar.Message)
	}
	outlineFar.OutlineList.ScrollIntoView(outlineFar.OutlineSelected)
	farTester.Frame()
	farTester.Advance(time.Second)
	farTester.Frame()
	captureAcceptanceTester(t, filepath.Join(out, "outline-far-heading-760x480.png"), farTester)

	outlinePartial := newApp()
	outlinePartial.Open(partialOutlinePath)
	outlinePartial.ShowOutline = true
	partialTester := ui.NewTester(outlinePartial.View, 760, 480)
	partialTester.Frame()
	partialTester.Frame()
	if !outlinePartial.OutlineTruncated || len(outlinePartial.OutlineItems) != editor.MaxOutlineItems {
		t.Fatalf("partial Markdown outline was not bounded: items=%d truncated=%v",
			len(outlinePartial.OutlineItems), outlinePartial.OutlineTruncated)
	}
	outlinePartial.selectOutlineItem(outlinePartial.OutlineItems[len(outlinePartial.OutlineItems)-1].ID, false)
	if !outlinePartial.activateSelectedOutlineItem() {
		t.Fatalf("partial Markdown outline activation failed: %s", outlinePartial.Message)
	}
	outlinePartial.OutlineList.ScrollIntoView(outlinePartial.OutlineSelected)
	partialTester.Frame()
	partialTester.Advance(time.Second)
	partialTester.Frame()
	captureAcceptanceTester(t, filepath.Join(out, "outline-partial-760x480.png"), partialTester)

	mindmapOutline := newApp()
	mindmapOutline.Open(collapsedMindmapPath)
	mindmapOutline.ShowOutline = true
	mindmapTester := ui.NewTester(mindmapOutline.View, 1180, 780)
	mindmapTester.Frame()
	mindmapTester.Frame()
	if !mindmapOutline.activateOutlineTarget(mindmapOutline.Current(), "interview", false, "大纲定位") {
		t.Fatalf("collapsed mindmap outline navigation failed: %s", mindmapOutline.Message)
	}
	mindmapTester.Frame()
	mindmapTester.Frame()
	captureAcceptanceTester(t, filepath.Join(out, "outline-mindmap-hidden-1180x780.png"), mindmapTester)

	mindmapOutlineDark := newApp()
	mindmapOutlineDark.Open(collapsedMindmapPath)
	mindmapOutlineDark.ShowOutline = true
	mindmapOutlineDark.Dark = true
	mindmapDarkTester := ui.NewTester(mindmapOutlineDark.View, 1180, 780)
	mindmapDarkTester.Frame()
	mindmapDarkTester.Frame()
	if !mindmapOutlineDark.activateOutlineTarget(mindmapOutlineDark.Current(), "interview", false, "大纲定位") {
		t.Fatalf("collapsed dark mindmap outline navigation failed: %s", mindmapOutlineDark.Message)
	}
	mindmapDarkTester.Frame()
	mindmapDarkTester.Frame()
	captureAcceptanceTester(t, filepath.Join(out, "outline-mindmap-hidden-dark-1180x780.png"), mindmapDarkTester)

	captureReadonlyMindmapOutline(t, out, fixture)

	drawioOutline := newApp()
	drawioOutline.Open(drawioPath)
	drawioOutline.ShowOutline = true
	drawioOutlineTester := ui.NewTester(drawioOutline.View, 760, 480)
	drawioOutlineTester.Frame()
	drawioOutlineTester.Frame()
	captureAcceptanceTester(t, filepath.Join(out, "outline-drawio-unsupported-760x480.png"), drawioOutlineTester)

	mindmap := newApp()
	mindmap.Open(mindmapPath)
	captureAcceptanceApp(t, filepath.Join(out, "mindmap.png"), mindmap, 1180, 780)
	mindmapDark := newApp()
	mindmapDark.Open(mindmapPath)
	mindmapDark.Dark = true
	captureAcceptanceApp(t, filepath.Join(out, "mindmap-dark.png"), mindmapDark, 1180, 780)
	mindmapNarrow := newApp()
	mindmapNarrow.Open(mindmapPath)
	captureAcceptanceApp(t, filepath.Join(out, "mindmap-760x480.png"), mindmapNarrow, 760, 480)
	drawio := newApp()
	drawio.Open(drawioPath)
	captureAcceptanceApp(t, filepath.Join(out, "drawio.png"), drawio, 1180, 780)
	drawioDark := newApp()
	drawioDark.Open(drawioPath)
	drawioDark.Dark = true
	captureAcceptanceApp(t, filepath.Join(out, "drawio-dark.png"), drawioDark, 1180, 780)
	drawioNarrow := newApp()
	drawioNarrow.Open(drawioPath)
	captureAcceptanceApp(t, filepath.Join(out, "drawio-760x480.png"), drawioNarrow, 760, 480)

	pluginManager := newApp()
	pluginManager.Open(pluginPath)
	pluginManager.ShowPlugins = true
	captureAcceptanceApp(t, filepath.Join(out, "plugin-manager.png"), pluginManager, 1180, 780)
	pluginManagerNarrow := newApp()
	pluginManagerNarrow.Open(pluginPath)
	pluginManagerNarrow.ShowPlugins = true
	captureAcceptanceApp(t, filepath.Join(out, "plugin-manager-760x480.png"), pluginManagerNarrow, 760, 480)
	pluginFallback := newApp()
	pluginFallback.Open(pluginPath)
	pluginFallback.setPluginEnabled("toml", false)
	captureAcceptanceApp(t, filepath.Join(out, "plugin-fallback.png"), pluginFallback, 1180, 780)

	captureInstalledSourceOutlineAcceptance(t, out, fixture)

	readonly := newApp()
	readonly.Open(readonlyPath)
	readonly.Current().Doc.ReadOnly = true
	readonly.Current().Editor.ReadOnly = true
	captureAcceptanceApp(t, filepath.Join(out, "readonly-text.png"), readonly, 1180, 780)
	encoding := newApp()
	encoding.Open(encodingPath)
	captureAcceptanceApp(t, filepath.Join(out, "encoding-fallback.png"), encoding, 1180, 780)
	encodingNarrow := newApp()
	encodingNarrow.Open(encodingPath)
	captureAcceptanceApp(t, filepath.Join(out, "encoding-fallback-760x480.png"), encodingNarrow, 760, 480)

	settingsNarrow := newApp()
	settingsNarrow.Open(markdownPath)
	settingsNarrow.ShowSettings = true
	captureAcceptanceApp(t, filepath.Join(out, "settings-760x480.png"), settingsNarrow, 760, 480)

	largeTreeSearch := newApp()
	populateAcceptanceLists(largeTreeSearch, workspace)
	captureAcceptanceLargeLists(t, filepath.Join(out, "large-tree-search-1180x780.png"), largeTreeSearch, 1180, 780)
	largeTreeSearchNarrow := newApp()
	populateAcceptanceLists(largeTreeSearchNarrow, workspace)
	captureAcceptanceLargeLists(t, filepath.Join(out, "large-tree-search-760x480.png"), largeTreeSearchNarrow, 760, 480)

	narrow := newApp()
	narrow.Open(markdownPath)
	if !narrow.setMode(narrow.Current(), "split") {
		t.Fatal("could not enter narrow Markdown split mode")
	}
	captureAcceptanceApp(t, filepath.Join(out, "minimum-760x480.png"), narrow, 760, 480)
}

func TestAcceptanceReadonlyMindmapScreenshot(t *testing.T) {
	out := os.Getenv("LEVNOTE_ACCEPTANCE_SCREENSHOT_DIR")
	if out == "" {
		t.Skip("set LEVNOTE_ACCEPTANCE_SCREENSHOT_DIR for visual acceptance")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	captureReadonlyMindmapOutline(t, out, writeAcceptanceFixture(t, t.TempDir()))
}

func TestAcceptanceInstalledSourceOutlineScreenshots(t *testing.T) {
	out := os.Getenv("LEVNOTE_SOURCE_PLUGIN_SCREENSHOT_DIR")
	if out == "" {
		t.Skip("set LEVNOTE_SOURCE_PLUGIN_SCREENSHOT_DIR for installed source-outline visual acceptance")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	captureInstalledSourceOutlineAcceptance(t, out, writeAcceptanceFixture(t, t.TempDir()))
}

func captureInstalledSourceOutlineAcceptance(t *testing.T, out string, fixture acceptanceFixture) {
	t.Helper()
	newApp := func() *App {
		a := newTestApp(t, fixture.config)
		a.Root = fixture.workspace
		a.refreshTreeNow()
		return a
	}
	captureOutline := func(name string, dark bool, width, height int, path string, selectLast bool) {
		a := newApp()
		a.Open(path)
		a.ShowOutline = true
		a.Dark = dark
		tester := ui.NewTester(a.View, width, height)
		tester.Frame()
		tester.Frame()
		if a.OutlineError != "" || len(a.OutlineItems) == 0 ||
			a.Current().Adapter != nil || a.Current().OutlineFactory == nil {
			t.Fatalf("%s installed outline did not load: error=%q items=%d tab=%#v",
				name, a.OutlineError, len(a.OutlineItems), a.Current())
		}
		index := 0
		if selectLast {
			index = len(a.OutlineItems) - 1
		}
		a.selectOutlineItem(a.OutlineItems[index].ID, true)
		if !a.activateSelectedOutlineItem() {
			t.Fatalf("%s installed outline activation failed: %s", name, a.Message)
		}
		tester.Frame()
		tester.Advance(time.Second)
		tester.Frame()
		captureAcceptanceTester(t, filepath.Join(out, name), tester)
	}
	captureOutline(
		"outline-installed-ini-narrow-dark-760x480.png",
		true, 760, 480, fixture.sourceOutline, false,
	)
	captureOutline(
		"outline-installed-ini-far-unicode-crlf-760x480.png",
		false, 760, 480, fixture.farSourceOutline, true,
	)

	capability := newApp()
	capability.Open(fixture.sourceOutline)
	capability.ShowPlugins = true
	capabilityTester := ui.NewTester(capability.View, 1180, 780)
	capabilityTester.Frame()
	capabilityTester.Frame()
	if !capabilityTester.HasText("schema2 · 源码大纲（2 条逐行字面规则）+ 基础字符串/数字高亮与已配置行内规则；不识别多行注释/字符串上下文") {
		t.Fatalf("schema2 plugin capability is not visible: %#v", capabilityTester.Texts())
	}
	captureAcceptanceTester(t, filepath.Join(out, "plugin-manager-schema2-outline.png"), capabilityTester)
}

func captureReadonlyMindmapOutline(t *testing.T, out string, fixture acceptanceFixture) {
	t.Helper()
	a := newTestApp(t, fixture.config)
	a.Root = fixture.workspace
	a.refreshTreeNow()
	a.Open(fixture.collapsedMindmap)
	a.Current().Doc.ReadOnly = true
	a.Current().Editor.ReadOnly = true
	a.ShowOutline = true
	runner := newControlledExporter()
	a.exporter = runner
	tester := ui.NewTester(a.View, 760, 480)
	call, job := awaitExportCall(t, runner), a.previewJob
	outcome := previewSuccess()
	// Independent neutral scene for this headless capture. Hidden interview
	// is deliberately absent; real bundled producer acceptance is separate.
	outcome.result.Payload = []byte(`{"schema":1,"layout":"logicalStructureRight","nodeCount":3,"edgeCount":2,"bounds":{"x":0,"y":0,"width":312,"height":160},"nodes":[{"id":"root","label":"产品规划","x":28,"y":60,"width":92,"height":40,"shape":"rounded","fill":"","stroke":"","text":"","root":true,"collapsed":false},{"id":"research","label":"用户研究","x":192,"y":28,"width":92,"height":40,"shape":"rounded","fill":"","stroke":"","text":"","root":false,"collapsed":true},{"id":"delivery","label":"交付节奏","x":192,"y":92,"width":92,"height":40,"shape":"rounded","fill":"","stroke":"","text":"","root":false,"collapsed":false}],"edges":[{"from":"root","to":"research","fromPoint":{"x":120,"y":80},"toPoint":{"x":192,"y":48},"stroke":""},{"from":"root","to":"delivery","fromPoint":{"x":120,"y":80},"toPoint":{"x":192,"y":112},"stroke":""}]}`)
	call.response <- outcome
	awaitPreviewDone(t, job)
	tester.Frame()
	tester.Frame()
	before, version, selection := a.Current().Doc.Buffer.String(), a.Current().Doc.Buffer.Version(), a.Current().Doc.Buffer.Selection()
	if a.activateOutlineTarget(a.Current(), "interview", false, "大纲定位") {
		t.Fatal("readonly hidden mindmap outline unexpectedly navigated")
	}
	if !strings.Contains(a.Message, editor.ErrOutlineReadOnlyHidden.Error()) || a.Current().Doc.Buffer.String() != before || a.Current().Doc.Buffer.Version() != version || a.Current().Doc.Buffer.Selection() != selection {
		t.Fatal("readonly scene hidden-target refusal was silent or changed source")
	}
	tester.Frame()
	tester.Advance(time.Second)
	tester.Frame()
	captureAcceptanceTester(t, filepath.Join(out, "outline-mindmap-readonly-760x480.png"), tester)
}

func writeAcceptanceFile(t *testing.T, path, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(value), 0600); err != nil {
		t.Fatal(err)
	}
}

func captureAcceptanceApp(t *testing.T, path string, a *App, width, height int) {
	t.Helper()
	tester := ui.NewTester(a.View, width, height)
	tester.Frame()
	tester.Advance(time.Second)
	tester.Frame()
	captureAcceptanceTester(t, path, tester)
}

func captureAcceptanceTester(t *testing.T, path string, tester *ui.Tester) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, tester.Image()); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func captureAcceptanceLargeLists(t *testing.T, path string, a *App, width, height int) {
	t.Helper()
	tester := ui.NewTester(a.View, width, height)
	tester.Frame()
	a.TreeSelected = len(a.Entries) - 1
	a.SearchSelected = len(a.SearchResults) - 5
	a.TreeList.ScrollTo(a.TreeSelected, ui.Center)
	a.SearchList.ScrollTo(a.SearchSelected, ui.Center)
	tester.Frame()
	tester.Frame()
	captureAcceptanceTester(t, path, tester)
}

func populateAcceptanceLists(a *App, workspace string) {
	a.Entries = make([]Entry, treeEntryLimit)
	for i := range a.Entries {
		name := fmt.Sprintf("archive-%02d-%04d.txt", i/80, i)
		a.Entries[i] = Entry{
			Path: filepath.Join(workspace, "archive", name),
			Name: name, Depth: i % 4,
		}
	}
	a.TreeTruncated = true
	a.TreeSelected = treeEntryLimit - 1
	a.TreeList.ScrollTo(a.TreeSelected, ui.Center)

	a.searchRoot = workspace
	a.SearchResults = make([]search.FileMatch, 2000)
	for i := range a.SearchResults {
		name := fmt.Sprintf("archive-%04d.txt", i)
		a.SearchResults[i] = search.FileMatch{
			Path: filepath.Join(workspace, "archive", name), Line: i + 1,
			Column: 8, Start: i, End: i + 4,
			Preview: fmt.Sprintf("很靠后的本地搜索结果 %04d", i),
		}
	}
	a.SearchSummary = "2000 个匹配 · 2000 个文件 · 跳过 0 项"
	a.ShowWorkspaceSearch = true
	a.SearchSelected = len(a.SearchResults) - 5
	a.SearchList.ScrollTo(a.SearchSelected, ui.Center)
}
