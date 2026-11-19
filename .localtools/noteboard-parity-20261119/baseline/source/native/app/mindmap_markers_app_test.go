package app

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/formats/mindmap"
)

const mindmapMarkersFixture = `{
  "format": "levnote.mindmap",
  "version": 1,
  "futureTop": 123456789012345678901234567890,
  "layout": "logicalStructure",
  "root": {
    "id": "root",
    "topic": "中心",
    "icons": ["1️⃣", "自定义", "⭐"],
    "tags": [{"text": "标签", "style": {"color": "red", "future": 1.20}, "className": "dense", "riskLevel": 2.00}],
    "image": {"url": "images/local.png", "width": 24, "height": 16, "future": 0.30},
    "unknownNode": {"kept": 1.00},
    "children": []
  }
}`

func openMarkerPanel(t *testing.T, tester *ui.Tester, width, height float32) {
	t.Helper()
	if _, ok := tester.AccessByRole(ui.RoleButton, "展开标记面板"); !ok {
		t.Fatalf("collapsed marker toggle is missing: %#v", tester.Texts())
	}
	clickMarkerControl(t, tester, width, height, "展开标记面板")
	for _, label := range []string{"优先级 1", "优先级 2", "红色标记", "星标标记", "清空全部标记"} {
		if _, ok := tester.Find(label); !ok {
			t.Fatalf("marker control %q is missing after opening: %#v", label, tester.Texts())
		}
	}
}

func revealMarkerControl(t *testing.T, tester *ui.Tester, width, height float32, label string, minHeight float32) ui.Rect {
	t.Helper()
	statusTop := height - 29
	tester.Wheel(width-6, height-35, 0, -3000)
	tester.Advance(time.Second)
	for attempt := 0; attempt < 30; attempt++ {
		rect, ok := tester.Find(label)
		if attempt > 0 || ok {
			t.Logf("marker control %q rect=%#v found=%v", label, rect, ok)
		}
		if ok && rect.W > 0 && rect.H >= minHeight-0.1 && rect.X >= 0 && rect.Y >= 0 &&
			rect.X+rect.W <= width+0.1 && rect.Y+rect.H <= statusTop+0.1 {
			return rect
		}
		tester.Wheel(width-6, height-35, 0, 30)
		tester.Advance(time.Second)
	}
	t.Fatalf("marker control %q is not reachable: %#v", label, tester.Texts())
	return ui.Rect{}
}

func clickMarkerControl(t *testing.T, tester *ui.Tester, width, height float32, label string) {
	t.Helper()
	rect := revealMarkerControl(t, tester, width, height, label, 30)
	tester.ClickAt(rect.X+rect.W/2, rect.Y+rect.H/2)
	tester.Frame()
}

func revealMarkerRecoveryButton(t *testing.T, tester *ui.Tester, width, height float32, label string) ui.Rect {
	t.Helper()
	statusTop := height - 29
	tester.Wheel(width-6, height-35, 0, -3000)
	tester.Advance(time.Second)
	for attempt := 0; attempt < 30; attempt++ {
		access, ok := tester.AccessByRole(ui.RoleButton, label)
		if attempt > 0 || ok {
			t.Logf("marker recovery button %q rect=%#v offscreen=%v found=%v", label, access.Bounds, access.Offscreen, ok)
		}
		if ok && !access.Offscreen && access.Bounds.W > 0 && access.Bounds.H >= 28-0.1 &&
			access.Bounds.X >= 0 && access.Bounds.Y >= 0 &&
			access.Bounds.X+access.Bounds.W <= width+0.1 && access.Bounds.Y+access.Bounds.H <= statusTop+0.1 {
			return access.Bounds
		}
		tester.Wheel(width-6, height-35, 0, 30)
		tester.Advance(time.Second)
	}
	access, _ := tester.AccessByRole(ui.RoleButton, label)
	t.Fatalf("marker recovery button %q is not reachable: role=%d name=%q bounds=%#v offscreen=%v texts=%#v",
		label, access.Role, access.Name, access.Bounds, access.Offscreen, tester.Texts())
	return ui.Rect{}
}

func clickMarkerRecoveryButton(t *testing.T, tester *ui.Tester, width, height float32, label string) {
	t.Helper()
	rect := revealMarkerRecoveryButton(t, tester, width, height, label)
	tester.ClickAt(rect.X+rect.W/2, rect.Y+rect.H/2)
	tester.Frame()
}

func requireMarkerIcons(t *testing.T, data []byte, want []string) {
	t.Helper()
	if bytes.HasPrefix(data, []byte("\xef\xbb\xbf")) {
		data = data[3:]
	}
	parsed := parseMindmapBytes(t, data)
	got, present, err := parsed.Icons("root")
	if err != nil || !present || !slices.Equal(got, want) {
		t.Fatalf("icons = %#v present=%v err=%v, want %#v", got, present, err, want)
	}
}

func requireMarkerMetadata(t *testing.T, data []byte) {
	t.Helper()
	if bytes.HasPrefix(data, []byte("\xef\xbb\xbf")) {
		data = data[3:]
	}
	requireUnknownMindmapMetadata(t, data)
	parsed := parseMindmapBytes(t, data)
	if parsed.Root == nil || parsed.Root.Tags == nil || len(*parsed.Root.Tags) != 1 {
		t.Fatalf("rich tags were not preserved: %#v", parsed.Root.Tags)
	}
	tag := (*parsed.Root.Tags)[0]
	if tag.Text != "标签" || tag.ClassName != "dense" || !tag.ClassNameSet ||
		!tag.StyleSet || tag.Style.Len() != 2 {
		t.Fatalf("rich tag fields changed: %#v", tag)
	}
	if got, ok := tag.Style.Get("color"); !ok || string(got) != `"red"` {
		t.Fatalf("tag style color changed: %s", got)
	}
	if got, ok := tag.Style.Get("future"); !ok || string(got) != "1.20" {
		t.Fatalf("tag style number spelling changed: %s", got)
	}
	if got, ok := tag.Extra.Get("riskLevel"); !ok || string(got) != "2.00" {
		t.Fatalf("tag unknown member changed: %s", got)
	}
	if parsed.Root.Image == nil || parsed.Root.Image.URL != "images/local.png" ||
		parsed.Root.Image.Width != "24" || parsed.Root.Image.Height != "16" ||
		parsed.Root.Image.Extra.Len() != 1 {
		t.Fatalf("rich image metadata changed: %#v", parsed.Root.Image)
	}
	if got, ok := parsed.Root.Image.Extra.Get("future"); !ok || string(got) != "0.30" {
		t.Fatalf("image unknown member changed: %s", got)
	}
}

func TestMindmapMarkersRealAdapterSaveReopenUndoAndReadonly(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "markers.mindmap")
	if err := os.WriteFile(path, []byte(mindmapMarkersFixture), 0600); err != nil {
		t.Fatal(err)
	}

	a := newTestApp(t, filepath.Join(root, "config-a"))
	a.Open(path)
	tab := a.Current()
	if tab == nil || tab.Adapter == nil || tab.Adapter.ID != "mindmap" || tab.Mode != "preview" {
		t.Fatalf("real mindmap adapter did not open: %#v", tab)
	}
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	openMarkerPanel(t, tester, 1180, 780)

	before := tab.Doc.Buffer.String()
	beforeVersion := tab.Doc.Buffer.Version()
	clickMarkerControl(t, tester, 1180, 780, "优先级 1")
	requireMarkerIcons(t, []byte(tab.Doc.Buffer.String()), []string{"自定义", "⭐"})
	if tab.Doc.Buffer.String() == before || tab.Doc.Buffer.Version() == beforeVersion || !tab.Doc.Dirty() {
		t.Fatal("clicking an existing priority glyph did not remove it as a real edit")
	}

	clickMarkerControl(t, tester, 1180, 780, "优先级 2")
	requireMarkerIcons(t, []byte(tab.Doc.Buffer.String()), []string{"自定义", "⭐", "2️⃣"})
	if !tab.Doc.Dirty() {
		t.Fatal("real marker click did not mark the document dirty")
	}

	clickMarkerControl(t, tester, 1180, 780, "红色标记")
	preStarSource := []byte(tab.Doc.Buffer.String())
	clickMarkerControl(t, tester, 1180, 780, "星标标记")
	requireMarkerIcons(t, []byte(tab.Doc.Buffer.String()), []string{"自定义", "2️⃣", "🟥"})
	if depth := tab.Doc.Buffer.History().UndoDepth; depth != 4 {
		t.Fatalf("four real marker clicks created %d undo steps, want 4", depth)
	}

	// The focused marker panel may retain keyboard focus; app editor commands
	// cover the shared undo route without claiming a native canvas gesture.
	if !tab.Editor.Command("undo") {
		t.Fatal("app editor undo command failed")
	}
	tester.Frame()
	if got := []byte(tab.Doc.Buffer.String()); !bytes.Equal(got, preStarSource) {
		t.Fatalf("undo did not restore the exact pre-star source:\n got=%s\nwant=%s", got, preStarSource)
	}
	requireMarkerIcons(t, preStarSource, []string{"自定义", "⭐", "2️⃣", "🟥"})
	if !tab.Editor.Command("redo") {
		t.Fatal("app editor redo command failed")
	}
	tester.Frame()
	requireMarkerIcons(t, []byte(tab.Doc.Buffer.String()), []string{"自定义", "2️⃣", "🟥"})
	requireMarkerMetadata(t, []byte(tab.Doc.Buffer.String()))

	clickMarkerControl(t, tester, 1180, 780, "清空全部标记")
	requireMarkerIcons(t, []byte(tab.Doc.Buffer.String()), []string{})
	requireMarkerMetadata(t, []byte(tab.Doc.Buffer.String()))

	if !a.Save(tab, false) {
		t.Fatalf("marker save failed: %s", a.Message)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	requireMarkerIcons(t, saved, []string{})
	requireMarkerMetadata(t, saved)
	editedSVG := requireMarkerExport(t, tab)
	editedTspan := "中心 [标签]"
	if !strings.Contains(string(editedSVG), "<tspan x=") ||
		!strings.Contains(string(editedSVG), ">"+editedTspan+"</tspan>") {
		t.Fatalf("edited SVG marker/topic/tag text is wrong, want %q: %s", editedTspan, editedSVG)
	}

	reopened := newTestApp(t, filepath.Join(root, "config-b"))
	reopened.Open(path)
	reopenedTab := reopened.Current()
	reopenedTester := ui.NewTester(reopened.View, 1180, 780)
	reopenedTester.Frame()
	openMarkerPanel(t, reopenedTester, 1180, 780)
	clickMarkerControl(t, reopenedTester, 1180, 780, "优先级 1")
	requireMarkerIcons(t, []byte(reopenedTab.Doc.Buffer.String()), []string{"1️⃣"})
	requireMarkerMetadata(t, []byte(reopenedTab.Doc.Buffer.String()))
	reopenedSVG := requireMarkerExport(t, reopenedTab)
	reopenedTspan := "1️⃣ 中心 [标签]"
	if !strings.Contains(string(reopenedSVG), ">"+reopenedTspan+"</tspan>") ||
		strings.Contains(string(reopenedSVG), "⭐") {
		t.Fatalf("reopened SVG marker text is wrong: %s", reopenedSVG)
	}

	readonlyDoc, err := document.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	readonlyDoc.ReadOnly = true
	readonlyApp := newTestApp(t, filepath.Join(root, "config-readonly"))
	readonlyApp.add(readonlyDoc, "markers.mindmap", "")
	readonlyTab := readonlyApp.Current()
	if readonlyTab == nil || !readonlyTab.Doc.ReadOnly || !readonlyTab.Editor.ReadOnly {
		t.Fatalf("readonly marker document was not configured: %#v", readonlyTab)
	}
	runner := newControlledExporter()
	readonlyApp.exporter = runner
	readonlyTester := ui.NewTester(readonlyApp.View, 1180, 780)
	call, job := awaitExportCall(t, runner), readonlyApp.previewJob
	call.response <- previewSuccess()
	awaitPreviewDone(t, job)
	readonlyTester.Frame()
	readonlyBefore := readonlyTab.Doc.Buffer.String()
	readonlyVersion := readonlyTab.Doc.Buffer.Version()
	readonlyHistory, readonlySelection := readonlyTab.Doc.Buffer.History(), readonlyTab.Doc.Buffer.Selection()
	if _, ok := readonlyTester.AccessByRole(ui.RoleButton, "优先级 2"); ok {
		t.Fatal("scene-only readonly view exposed marker editing controls")
	}
	if !readonlyTester.HasText("此视图只读；节点属性、标签与图片引用请在源码中查看。") {
		t.Fatal("readonly scene guidance is not visible")
	}
	if err := readonlyTester.Click("只读思维导图画布"); err != nil {
		t.Fatal(err)
	}
	readonlyTester.Type("不能更改标记🙂")
	if readonlyTab.Doc.Buffer.String() != readonlyBefore || readonlyTab.Doc.Buffer.Version() != readonlyVersion || readonlyTab.Doc.Buffer.History() != readonlyHistory || readonlyTab.Doc.Buffer.Selection() != readonlySelection || readonlyTab.Doc.Dirty() {
		t.Fatal("readonly marker state changed the immutable host source")
	}
}

func requireMarkerExport(t *testing.T, tab *Tab) []byte {
	t.Helper()
	if tab == nil || tab.View == nil {
		t.Fatalf("marker export tab has no native view: %#v", tab)
	}
	data, err := tab.View.Export("svg")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "<svg ") || !strings.Contains(string(data), "</svg>") {
		t.Fatalf("marker export is not an SVG document: %s", data)
	}
	return data
}

func persistMarkerEvidence(t *testing.T, name string) string {
	t.Helper()
	dir := os.Getenv("LEVNOTE_MINDMAP_MARKERS_APP_SCREENSHOT_DIR")
	if dir == "" {
		dir = t.TempDir()
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dir, name)
}

func TestMindmapMarkersPendingFlushModeSplitAndSourcePreview(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "markers.mindmap")
	if err := os.WriteFile(path, []byte(mindmapMarkersFixture), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	openMarkerPanel(t, tester, 1180, 780)

	const exactNote = "标记前写回🙂"
	beginGraphFieldComposition(t, tester, "字段：纯文本备注", "节点备注（纯文本）", exactNote)
	clickMarkerControl(t, tester, 1180, 780, "优先级 2")
	parsed := parseMindmapBytes(t, []byte(tab.Doc.Buffer.String()))
	note, present, err := parsed.Field("root", mindmap.FieldNote)
	if err != nil || !present || note != exactNote {
		t.Fatalf("marker click did not flush pending note: note=%q present=%v err=%v", note, present, err)
	}
	requireMarkerIcons(t, []byte(tab.Doc.Buffer.String()), []string{"自定义", "⭐", "2️⃣"})
	requireMarkerMetadata(t, []byte(tab.Doc.Buffer.String()))

	if !a.setMode(tab, "split") {
		t.Fatalf("split-mode barrier failed: %s", a.Message)
	}
	a.Dark = true
	splitTester := ui.NewTester(a.View, 760, 480)
	splitTester.Frame()
	openMarkerPanel(t, splitTester, 760, 480)
	beforeToggle := tab.Doc.Buffer.String()
	clearRect := revealMarkerControl(t, splitTester, 760, 480, "清空全部标记", 30)
	if clearRect.X < 0 || clearRect.Y < 0 || clearRect.X+clearRect.W > 760 || clearRect.Y+clearRect.H > 451 {
		t.Fatalf("split clear control is outside the 760x480 window: %#v", clearRect)
	}
	splitTester.ClickAt(clearRect.X+clearRect.W/2, clearRect.Y+clearRect.H/2)
	splitTester.Frame()
	requireMarkerIcons(t, []byte(tab.Doc.Buffer.String()), []string{})
	requireMarkerMetadata(t, []byte(tab.Doc.Buffer.String()))
	if tab.Doc.Buffer.String() == beforeToggle || !tab.Doc.Dirty() {
		t.Fatal("split native marker action did not mutate the host")
	}
	clearedSVG := requireMarkerExport(t, tab)
	clearedTspan := "中心 [标签]"
	if !strings.Contains(string(clearedSVG), ">"+clearedTspan+"</tspan>") ||
		strings.Contains(string(clearedSVG), "⭐") || strings.Contains(string(clearedSVG), "自定义") {
		t.Fatalf("cleared SVG marker text is wrong, want %q: %s", clearedTspan, clearedSVG)
	}
	splitEvidence := persistMarkerEvidence(t, "mindmap-markers-app-split-760x480.png")
	captureAcceptanceTester(t, splitEvidence, splitTester)
	if stat, err := os.Stat(splitEvidence); err != nil || stat.Size() == 0 {
		t.Fatalf("split evidence image was not written: %#v err=%v", stat, err)
	}

	if !a.setMode(tab, "source") {
		t.Fatalf("source-mode barrier failed: %s", a.Message)
	}
	sourceBefore := tab.Doc.Buffer.String()
	if !a.setMode(tab, "preview") {
		t.Fatalf("preview-mode barrier failed: %s", a.Message)
	}
	if tab.Mode != "preview" || tab.Doc.Buffer.String() != sourceBefore {
		t.Fatalf("mode round trip changed mode/source: mode=%q", tab.Mode)
	}
}

func TestMindmapMarkersConflictBlocksActionAndRecoversPendingDraft(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "markers.mindmap")
	if err := os.WriteFile(path, []byte(mindmapMarkersFixture), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	openMarkerPanel(t, tester, 1180, 780)

	const exactNote = "冲突后仍可恢复🙂"
	beginGraphFieldComposition(t, tester, "字段：纯文本备注", "节点备注（纯文本）", exactNote)
	external := strings.Replace(tab.Doc.Buffer.String(), `"topic": "中心"`, `"topic": "中心", "note": "外部改动"`, 1)
	if external == tab.Doc.Buffer.String() {
		t.Fatal("fixture topic was not replaced for the conflict test")
	}
	if err := tab.Doc.Buffer.ReplaceUTF16(0, tab.Doc.Buffer.LenUTF16(), external); err != nil {
		t.Fatal(err)
	}
	authority := tab.Doc.Buffer.String()
	authorityVersion := tab.Doc.Buffer.Version()
	authorityHistory := tab.Doc.Buffer.History().UndoDepth
	authorityDirty := tab.Doc.Dirty()
	tester.Frame()

	clickMarkerControl(t, tester, 1180, 780, "优先级 2")
	if tab.Doc.Buffer.String() != authority || tab.Doc.Buffer.Version() != authorityVersion ||
		tab.Doc.Buffer.History().UndoDepth != authorityHistory || tab.Doc.Dirty() != authorityDirty {
		t.Fatalf("blocked marker action changed host authority: version=%d history=%d dirty=%v",
			tab.Doc.Buffer.Version(), tab.Doc.Buffer.History().UndoDepth, tab.Doc.Dirty())
	}
	requireMarkerIcons(t, []byte(tab.Doc.Buffer.String()), []string{"1️⃣", "自定义", "⭐"})
	if !strings.Contains(strings.Join(tester.Texts(), "\n"), "节点草稿尚未写回") {
		t.Fatalf("conflict recovery guidance is not visible: %#v", tester.Texts())
	}
	problem, ok := tester.Access("标记操作问题")
	if !ok || !strings.Contains(problem.Description, "未完成“优先级 2”") ||
		!strings.Contains(problem.Description, "节点的源字段已改变") {
		t.Fatalf("marker conflict failure is not visible: found=%v info=%#v texts=%#v", ok, problem, tester.Texts())
	}
	for _, label := range []string{"重试写回", "恢复为新文档"} {
		rect := revealMarkerRecoveryButton(t, tester, 1180, 780, label)
		if rect.W <= 0 || rect.H <= 0 {
			t.Fatalf("recovery control %q is not reachable: %#v", label, rect)
		}
	}
	a.Dark = true
	conflictTester := ui.NewTester(a.View, 760, 480)
	conflictTester.Frame()
	for _, label := range []string{"重试写回", "恢复为新文档"} {
		revealMarkerRecoveryButton(t, conflictTester, 760, 480, label)
	}
	conflictEvidence := persistMarkerEvidence(t, "mindmap-markers-app-conflict-760x480-dark.png")
	captureAcceptanceTester(t, conflictEvidence, conflictTester)
	if stat, err := os.Stat(conflictEvidence); err != nil || stat.Size() == 0 {
		t.Fatalf("conflict evidence image was not written: %#v err=%v", stat, err)
	}
	clickMarkerRecoveryButton(t, conflictTester, 760, 480, "恢复为新文档")

	if len(a.Tabs) != 2 || a.Active != 1 || a.Current() == tab {
		t.Fatalf("draft recovery did not open exactly one new active tab: active=%d tabs=%d", a.Active, len(a.Tabs))
	}
	recovered := a.Current()
	wantRecovery := "markers.mindmap\n节点 ID: root\n页面 ID: \n字段: 节点备注（纯文本）\n原字段存在: false\n\n" + exactNote
	if recovered == nil || !recovered.Doc.IsNew || recovered.Doc.Buffer.String() != wantRecovery ||
		!strings.Contains(recovered.Warning, "未保存副本") {
		t.Fatalf("recovered draft = %#v want=%q", recovered, wantRecovery)
	}
	if tab.Doc.Buffer.String() != authority || tab.Doc.Buffer.Version() != authorityVersion {
		t.Fatal("draft recovery changed the authoritative marker host")
	}
	requireMarkerIcons(t, []byte(tab.Doc.Buffer.String()), []string{"1️⃣", "自定义", "⭐"})
}

func TestMindmapMarkersBOMAndUnicodeReopen(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "markers-bom.mindmap")
	source := "\ufeff" + strings.Replace(mindmapMarkersFixture, `"自定义"`, `"中文🙂"`, 1)
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	openMarkerPanel(t, tester, 1180, 780)
	clickMarkerControl(t, tester, 1180, 780, "优先级 3")
	requireMarkerIcons(t, []byte(tab.Doc.Buffer.String()), []string{"中文🙂", "⭐", "3️⃣"})
	requireMarkerMetadata(t, []byte(tab.Doc.Buffer.String()))
	if !a.Save(tab, false) {
		t.Fatalf("BOM marker save failed: %s", a.Message)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved) == 0 {
		t.Fatal("BOM marker save wrote an empty file")
	}
	if !strings.HasPrefix(string(saved), "\ufeff") {
		t.Fatal("marker save removed the UTF-8 BOM")
	}
	requireMarkerIcons(t, saved, []string{"中文🙂", "⭐", "3️⃣"})
	requireMarkerMetadata(t, saved)

	reopened := newTestApp(t, filepath.Join(root, "config-reopen"))
	reopened.Open(path)
	reopenedTab := reopened.Current()
	if reopenedTab == nil || reopenedTab.Adapter == nil || reopenedTab.Adapter.ID != "mindmap" ||
		reopenedTab.Mode != "preview" || reopenedTab.Doc.Encoding != document.UTF8 ||
		!reopenedTab.Doc.BOM || reopenedTab.Doc.ReadOnly || reopenedTab.Editor.ReadOnly {
		t.Fatalf("saved BOM marker document did not reopen writable in preview: %#v", reopenedTab)
	}
	if len(reopenedTab.Doc.Buffer.String()) == 0 {
		t.Fatal("reopened BOM marker document is empty")
	}
	reopenedTester := ui.NewTester(reopened.View, 1180, 780)
	reopenedTester.Frame()
	openMarkerPanel(t, reopenedTester, 1180, 780)
	requireMarkerIcons(t, []byte(reopenedTab.Doc.Buffer.String()), []string{"中文🙂", "⭐", "3️⃣"})
	requireMarkerMetadata(t, []byte(reopenedTab.Doc.Buffer.String()))
	clickMarkerControl(t, reopenedTester, 1180, 780, "优先级 4")
	requireMarkerIcons(t, []byte(reopenedTab.Doc.Buffer.String()), []string{"中文🙂", "⭐", "4️⃣"})
	requireMarkerMetadata(t, []byte(reopenedTab.Doc.Buffer.String()))
}

func TestMindmapMarkersAppNarrowScreenshots(t *testing.T) {
	outputDir := os.Getenv("LEVNOTE_MINDMAP_MARKERS_APP_SCREENSHOT_DIR")
	if outputDir == "" {
		t.Skip("set LEVNOTE_MINDMAP_MARKERS_APP_SCREENSHOT_DIR")
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatal(err)
	}

	capture := func(t *testing.T, name string, dark bool) {
		t.Helper()
		root := t.TempDir()
		path := filepath.Join(root, "markers.mindmap")
		source := strings.Replace(mindmapMarkersFixture, `    "unknownNode": {"kept": 1.00},`, "", 1)
		if err := os.WriteFile(path, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
		a := newTestApp(t, filepath.Join(root, "config"))
		a.Dark = dark
		a.Open(path)
		tab := a.Current()
		if tab == nil || tab.Mode != "preview" {
			t.Fatalf("screenshot document did not open in preview: %#v", tab)
		}
		tester := ui.NewTester(a.View, 760, 480)
		tester.SetDark(dark)
		tester.Frame()
		beginGraphFieldComposition(t, tester, "字段：纯文本备注", "节点备注（纯文本）", "保留待写回🙂")
		openMarkerPanel(t, tester, 760, 480)
		clickMarkerControl(t, tester, 760, 480, "优先级 2")
		clickMarkerControl(t, tester, 760, 480, "星标标记")
		tester.Scroll(750, 300, 0, 30)
		tester.Frame()
		rect := revealMarkerControl(t, tester, 760, 480, "清空全部标记", 30)
		if rect.X < 0 || rect.Y < 0 || rect.X+rect.W > 760 || rect.Y+rect.H > 451 {
			t.Fatalf("clear action is not reachable in the 760x480 inspector: %#v", rect)
		}
		captureAcceptanceTester(t, filepath.Join(outputDir, name), tester)
	}

	t.Run("light", func(t *testing.T) { capture(t, "mindmap-markers-app-light-760x480.png", false) })
	t.Run("dark", func(t *testing.T) { capture(t, "mindmap-markers-app-dark-760x480.png", true) })
}
