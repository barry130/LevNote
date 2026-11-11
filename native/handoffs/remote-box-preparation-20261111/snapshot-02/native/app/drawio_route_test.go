package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/engineipc"
	"levnote.local/native/formats/drawio"
	"levnote.local/native/ui/editor"
)

const drawioRouteManifest = `{"schema":3,"id":"diagram-notes","name":"图纸笔记","extensions":[".diagramnote",".diagramnote2"],"nativeRoute":"builtin.drawio"}`

func drawioRouteApp(t *testing.T) (*App, string, string) {
	t.Helper()
	root := t.TempDir()
	a := newTestApp(t, filepath.Join(root, "config"))
	manifestPath := installAppPluginManifest(t, a, "diagram-notes.json", drawioRouteManifest)
	return a, root, manifestPath
}

func drawioMetadataBody() string {
	return strings.Join([]string{
		`<mxfile host="levnote" exportMode="preserve" vendor="keep">`,
		`  <diagram id="p1" name="第一页面"><mxGraphModel><root>`,
		`    <mxCell id="0"/><mxCell id="1" parent="0"/>`,
		`    <mxCell id="same" value="PAGE_ONE" vertex="1" parent="1" customMeta="keep"><mxGeometry x="20" y="20" width="120" height="60" as="geometry"/></mxCell>`,
		`  </root></mxGraphModel></diagram>`,
		`</mxfile>`,
	}, "\r\n")
}

func requireDrawioRoute(t *testing.T, tab *Tab) {
	t.Helper()
	if tab == nil || tab.Adapter == nil || tab.Adapter.ID != "drawio" ||
		tab.MatchedPlugin.ID != "diagram-notes" || !tab.MatchedPlugin.Declarative ||
		tab.OutlineFactory != nil || tab.Adapter.Outline != nil || tab.Adapter.SceneEngine != nil ||
		tab.Editor.Highlight == nil || tab.Adapter.ExportEngine == nil ||
		*tab.Adapter.ExportEngine != (editor.ExportEngine{
			ID: "levnote.drawio-svg", Format: "svg", RequiresView: true,
		}) {
		t.Fatalf("custom draw.io route identities/capabilities: %+v", tab)
	}
	if !strings.Contains(tab.SourceCapabilityDetail, "随包原生格式能力") ||
		strings.Contains(tab.SourceCapabilityDetail, "大纲") ||
		!strings.HasPrefix(tab.SourceCapabilityBadge, "使用 draw.io") {
		t.Fatalf("custom draw.io capability is inaccurate: %q / %q",
			tab.SourceCapabilityBadge, tab.SourceCapabilityDetail)
	}
}

func assertDrawioTemplate(t *testing.T, source string) {
	t.Helper()
	document, issues, err := drawio.Parse([]byte(source), drawio.ParseOptions{})
	if err != nil || document == nil || len(document.Pages) != 1 ||
		document.Pages[0].ID != "page-1" || document.Pages[0].Name != "页面 1" {
		t.Fatalf("new draw.io route template = document:%#v issues:%#v err:%v",
			document, issues, err)
	}
	for _, issue := range issues {
		if issue.Code != drawio.IssueUnsupportedElement {
			t.Fatalf("new draw.io route template has unrecognized issue: %#v", issue)
		}
	}
}

func TestDrawioRouteNewOpenModesAndExactSaveAs(t *testing.T) {
	a, root, _ := drawioRouteApp(t)
	if !a.NewFromPlugin("diagram-notes") {
		t.Fatal(a.Message)
	}
	created := a.Current()
	requireDrawioRoute(t, created)
	if created.Name != "未命名.diagramnote" || created.Mode != "preview" {
		t.Fatalf("custom draw.io new document = %q / %q", created.Name, created.Mode)
	}
	assertDrawioTemplate(t, created.Doc.Buffer.String())

	content := drawioMetadataBody()
	path, disk := routingFile(t, root, "source.diagramnote", content)
	a.Open(path)
	tab := a.Current()
	requireDrawioRoute(t, tab)
	if tab.Doc.Buffer.String() != content || !tab.Doc.BOM || tab.Doc.Dirty() {
		t.Fatal("custom draw.io open did not preserve source/BOM/clean state")
	}
	tester := ui.NewTester(a.View, 1180, 780)
	for _, mode := range []string{"preview", "source", "split"} {
		if !a.setMode(tab, mode) {
			t.Fatal(a.Message)
		}
		tester.Frame()
		requireDrawioRoute(t, tab)
	}
	if !a.Save(tab, false) {
		t.Fatal(a.Message)
	}
	if saved := mustReadFile(t, path); !bytes.Equal(saved, disk) {
		t.Fatal("unchanged custom draw.io save changed BOM/CRLF/XML bytes")
	}

	copyPath := filepath.Join(root, "copy.diagramnote2")
	a.saveDialog = func(*Tab) (string, error) { return copyPath, nil }
	if !a.Save(tab, true) {
		t.Fatal(a.Message)
	}
	requireDrawioRoute(t, tab)
	if tab.Doc.Path != copyPath || tab.Name != "copy.diagramnote2" ||
		tab.MatchedPlugin.ID != "diagram-notes" || tab.Doc.Buffer.String() != content {
		t.Fatal("SaveAs did not rematch the mapped custom suffix")
	}
	if saved := mustReadFile(t, copyPath); !bytes.Equal(saved, disk) {
		t.Fatal("SaveAs changed custom draw.io disk bytes")
	}
	if original := mustReadFile(t, path); !bytes.Equal(original, disk) {
		t.Fatal("SaveAs changed the original custom draw.io file")
	}

	reopened := newTestApp(t, a.configDir)
	reopened.Open(copyPath)
	requireDrawioRoute(t, reopened.Current())
	if reopened.Current().Doc.Buffer.String() != content || !reopened.Current().Doc.BOM {
		t.Fatal("reopened custom draw.io suffix differs")
	}
}

func TestDrawioRouteReadonlySaveAsCreatesWritableExactCopy(t *testing.T) {
	a, root, _ := drawioRouteApp(t)
	content := drawioMetadataBody()
	path, disk := routingFile(t, root, "readonly.diagramnote", content)
	if err := os.Chmod(path, 0o400); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o600) })
	a.Open(path)
	tab := a.Current()
	requireDrawioRoute(t, tab)
	if !tab.Doc.ReadOnly || !tab.Editor.ReadOnly {
		t.Skip("filesystem did not expose readonly fixture")
	}
	copyPath := filepath.Join(root, "writable.diagramnote2")
	a.saveDialog = func(*Tab) (string, error) { return copyPath, nil }
	if !a.Save(tab, true) || tab.Doc.ReadOnly || tab.Editor.ReadOnly {
		t.Fatal("custom draw.io readonly SaveAs failed", a.Message)
	}
	requireDrawioRoute(t, tab)
	if original := mustReadFile(t, path); !bytes.Equal(original, disk) {
		t.Fatal("readonly custom draw.io original changed")
	}
	if copy := mustReadFile(t, copyPath); !bytes.Equal(copy, disk) {
		t.Fatal("readonly custom draw.io copy changed exact bytes")
	}
}

func TestDrawioRouteSecondPageFlushesOneTransactionBeforePrepare(t *testing.T) {
	a, root, _ := drawioRouteApp(t)
	path, disk := routingFile(t, root, "second.diagramnote", drawioWorkerBody)
	a.Open(path)
	tab := a.Current()
	requireDrawioRoute(t, tab)
	f, queue := newControlledExporter(), make(chan func(), 16)
	a.exporter, a.uiDispatch = f, func(fn func()) { queue <- fn }
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	clickDrawioPageButton(t, tester, "下一页")

	baseline, versionBefore := tab.Doc.Buffer.String(), tab.Doc.Buffer.Version()
	historyBefore := tab.Doc.Buffer.History()
	const pending = "第二页路由草稿🙂"
	if err := tester.Click("节点文本"); err != nil {
		t.Fatal(err)
	}
	tester.Command("selectAll")
	tester.Compose(pending, len([]rune(pending)))
	if tab.Doc.Buffer.String() != baseline || tab.Doc.Buffer.Version() != versionBefore ||
		tab.Doc.Buffer.History() != historyBefore {
		t.Fatal("pending custom draw.io field reached the buffer before Flush")
	}

	out := filepath.Join(root, "second-page.svg")
	dialogs := 0
	var dialogSource string
	a.exportDialog = func(mygo.SaveDialogOptions) (string, error) {
		dialogs++
		dialogSource = tab.Doc.Buffer.String()
		return out, nil
	}
	a.exportCurrent()
	job, call := a.exportJob, awaitExportCall(t, f)
	sourceAfterFlush := tab.Doc.Buffer.String()
	if job == nil || call.id != engineipc.DrawioSVG || call.source.Page == nil ||
		*call.source.Page != (engineipc.RequestPage{Index: 1, ID: "p2"}) {
		t.Fatalf("custom draw.io request = job:%v id:%s page:%+v",
			job != nil, call.id, call.source.Page)
	}
	if !strings.Contains(sourceAfterFlush, pending) ||
		tab.Doc.Buffer.Version() != versionBefore+1 ||
		tab.Doc.Buffer.History().UndoDepth != historyBefore.UndoDepth+1 ||
		tab.Doc.Buffer.History().RedoDepth != 0 {
		t.Fatalf("custom draw.io Flush transaction = version:%d->%d history:%+v",
			versionBefore, tab.Doc.Buffer.Version(), tab.Doc.Buffer.History())
	}
	var snapshot bytes.Buffer
	n, err := call.source.Data.WriteTo(&snapshot)
	if err != nil || n != int64(len(sourceAfterFlush)) ||
		call.source.Size != int64(len(sourceAfterFlush)) ||
		snapshot.String() != sourceAfterFlush || dialogs != 0 {
		t.Fatalf("custom draw.io post-Flush snapshot = n:%d size:%d err:%v dialogs:%d",
			n, call.source.Size, err, dialogs)
	}

	call.response <- drawioWorkerOutcome()
	awaitExportDone(t, job)
	drainOneExportCallback(t, queue)
	if dialogs != 1 || dialogSource != sourceAfterFlush ||
		!bytes.Equal(mustReadFile(t, out), []byte(drawioWorkerSVG)) ||
		!bytes.Equal(mustReadFile(t, path), disk) {
		t.Fatal("custom draw.io publication or original disk preservation mismatch")
	}
	if !tab.Doc.Buffer.Undo() || tab.Doc.Buffer.String() != baseline {
		t.Fatal("single custom draw.io field transaction did not undo to baseline")
	}

	calls := len(f.calls)
	if !a.setMode(tab, "source") {
		t.Fatal(a.Message)
	}
	a.exportCurrent()
	if len(f.calls) != calls || !strings.Contains(a.Message, "阅读或分屏") {
		t.Fatalf("source draw.io export was not visibly refused: %q", a.Message)
	}
	if !a.setMode(tab, "preview") {
		t.Fatal(a.Message)
	}
	tester.Frame()
	tab.View = nil
	a.exportCurrent()
	if len(f.calls) != calls || !strings.Contains(a.Message, "视图尚未准备好") {
		t.Fatalf("nil-view draw.io export was not visibly refused: %q", a.Message)
	}
}

func TestDrawioRouteLifecycleFallbackAndInstallDialog(t *testing.T) {
	a, root, _ := drawioRouteApp(t)
	var installMessage string
	a.openDialog = func(options mygo.OpenDialogOptions) ([]string, error) {
		installMessage = options.Message
		return nil, nil
	}
	a.installPlugin()
	if !strings.Contains(installMessage, "Markdown / 思维导图 / draw.io") {
		t.Fatalf("install dialog omitted draw.io: %q", installMessage)
	}

	path, _ := routingFile(t, root, "lifecycle.diagramnote", drawioMetadataBody())
	a.Open(path)
	tab := a.Current()
	requireDrawioRoute(t, tab)
	a.setPluginEnabled("drawio", false)
	requireDrawioRoute(t, tab)
	if tab.Mode != "preview" {
		t.Fatal("builtin draw.io disable changed installed route mode")
	}

	a.setPluginEnabled("diagram-notes", false)
	if tab.Adapter != nil || tab.Mode != "source" ||
		!strings.Contains(tab.FormatWarning, "仍可编辑和保存") {
		t.Fatalf("installed draw.io route did not fall back to editable source: %+v", tab)
	}
	a.setPluginEnabled("diagram-notes", true)
	requireDrawioRoute(t, tab)
	if !a.setMode(tab, "preview") {
		t.Fatal(a.Message)
	}

	adapters := a.adapters
	a.adapters = nil
	a.reconfigureTabs()
	if tab.Adapter != nil || tab.Mode != "source" ||
		!strings.Contains(tab.FormatWarning, "配置指定的原生格式不可用") {
		t.Fatalf("missing compiled draw.io adapter did not report fallback: %+v", tab)
	}
	a.adapters = adapters
}

func TestDrawioRouteRealConflictRetainsDraftAndBlocksDisable(t *testing.T) {
	a, root, _ := drawioRouteApp(t)
	path, _ := routingFile(t, root, "conflict.diagramnote", drawioWorkerBody)
	a.Open(path)
	tab := a.Current()
	requireDrawioRoute(t, tab)
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	if err := tester.Click("节点文本"); err != nil {
		t.Fatal(err)
	}
	tester.Command("selectAll")
	const pending = "冲突后仍保留🙂"
	tester.Compose(pending, len([]rune(pending)))
	baseline, view := tab.Doc.Buffer.String(), tab.View
	external := strings.Replace(baseline, `value="PAGE_ONE"`, `value="PAGE_ONE_EXTERNAL"`, 1)
	if external == baseline {
		t.Fatal("conflict fixture value missing")
	}
	if err := tab.Doc.Buffer.ReplaceUTF16(0, tab.Doc.Buffer.LenUTF16(), external); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	a.setPluginEnabled("diagram-notes", false)
	if !a.pluginManager.Enabled("diagram-notes") || tab.Adapter == nil ||
		tab.Adapter.ID != "drawio" || tab.View != view {
		t.Fatalf("conflicted custom draw.io draft did not block disable: %q", a.Message)
	}

	if err := tab.Doc.Buffer.ReplaceUTF16(0, tab.Doc.Buffer.LenUTF16(), baseline); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	a.setPluginEnabled("diagram-notes", false)
	if a.pluginManager.Enabled("diagram-notes") || tab.Adapter != nil ||
		!strings.Contains(tab.Doc.Buffer.String(), pending) {
		t.Fatalf("recovered custom draw.io draft was not committed before disable: %q", a.Message)
	}
}

func TestDrawioRouteSameIDRefreshRejectsLateExport(t *testing.T) {
	a, root, manifestPath := drawioRouteApp(t)
	path, _ := routingFile(t, root, "late.diagramnote", drawioWorkerBody)
	a.Open(path)
	tab := a.Current()
	requireDrawioRoute(t, tab)
	f, queue := newControlledExporter(), make(chan func(), 16)
	f.ignoreCancellation = true
	a.exporter, a.uiDispatch = f, func(fn func()) { queue <- fn }
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	out := filepath.Join(root, "must-not-publish.svg")
	dialogs := 0
	a.exportDialog = func(mygo.SaveDialogOptions) (string, error) {
		dialogs++
		return out, nil
	}
	a.exportCurrent()
	job, call := a.exportJob, awaitExportCall(t, f)
	if job == nil || call.id != engineipc.DrawioSVG {
		t.Fatal("custom draw.io export did not start")
	}
	oldAdapter := tab.Adapter
	updated := strings.Replace(drawioRouteManifest, `"name":"图纸笔记"`, `"name":"图纸笔记 已刷新"`, 1)
	if err := os.WriteFile(manifestPath, []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}
	a.refreshPlugins()
	if call.ctx.Err() == nil || tab.Adapter == oldAdapter {
		t.Fatal("same-ID refresh did not cancel and replace the old draw.io route")
	}
	requireDrawioRoute(t, tab)
	if tab.MatchedPlugin.Name != "图纸笔记 已刷新" {
		t.Fatalf("same-ID refresh did not expose the updated manifest name: %q", tab.MatchedPlugin.Name)
	}
	call.response <- drawioWorkerOutcome()
	awaitExportDone(t, job)
	drainOneExportCallback(t, queue)
	responseDrained := len(call.response) == 0
	job.mu.Lock()
	ready := job.ready
	payloadCleared := job.result.Payload == nil && len(job.result.Warnings) == 0 && job.err == nil
	job.mu.Unlock()
	job.lane.mu.Lock()
	laneReleased := !job.lane.reserved
	job.lane.mu.Unlock()
	if dialogs != 0 || a.exportJob != nil || !ready || !responseDrained || !payloadCleared || !laneReleased {
		t.Fatalf("late custom draw.io export passed a publication gate: dialogs=%d job=%v ready=%v responseDrained=%v cleared=%v laneReleased=%v",
			dialogs, a.exportJob, ready, responseDrained, payloadCleared, laneReleased)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Fatalf("late custom draw.io export published a file: %v", err)
	}
}
