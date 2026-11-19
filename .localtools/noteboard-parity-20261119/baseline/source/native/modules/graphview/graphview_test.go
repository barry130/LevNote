package graphview

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/drawio"
	"levnote.local/native/formats/mindmap"
	"levnote.local/native/ui/editor"
)

const testMindmap = `{
  "format": "levnote.mindmap",
  "version": 1,
  "layout": "logicalStructure",
  "root": {
    "id": "root",
    "topic": "中心",
    "children": [
      {"id": "left", "topic": "左分支", "children": []},
      {"id": "right", "topic": "右分支", "children": []}
    ]
  }
}
`

const testDrawio = `<mxfile host="levnote"><diagram id="p1" name="流程"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="a" value="开始" style="rounded=1;fillColor=#dae8fc;strokeColor=#6c8ebf;" vertex="1" parent="1"><mxGeometry x="20" y="20" width="120" height="60" as="geometry"/></mxCell><mxCell id="b" value="结束" style="ellipse;fillColor=#d5e8d4;strokeColor=#82b366;" vertex="1" parent="1"><mxGeometry x="260" y="30" width="120" height="60" as="geometry"/></mxCell></root></mxGraphModel></diagram></mxfile>`

func TestAdaptersTemplatesAndPaths(t *testing.T) {
	mindAdapter := MindmapAdapter()
	drawAdapter := DrawioAdapter()
	if !mindAdapter.Matches("A.MINDMAP") || mindAdapter.Matches("a.drawio") {
		t.Fatal("mindmap adapter path matching")
	}
	if !drawAdapter.Matches("A.DRAWIO") || drawAdapter.Matches("a.mindmap") {
		t.Fatal("drawio adapter path matching")
	}
	if _, _, err := mindmap.Parse([]byte(mindAdapter.Template()), mindmap.ParseOptions{}); err != nil {
		t.Fatalf("mindmap template: %v", err)
	}
	if _, issues, err := drawio.Parse([]byte(drawAdapter.Template()), drawio.ParseOptions{}); err != nil {
		t.Fatalf("drawio template: %v", err)
	} else if len(issues) != 0 {
		t.Fatalf("drawio template has structural warnings: %#v", issues)
	}
}

func TestMindmapToolbarInspectorUndoAndExactRoundTrip(t *testing.T) {
	buffer := text.New(testMindmap)
	changed := 0
	host := editor.Host{
		Buffer:  buffer,
		Changed: func() { changed++ },
	}
	view := MindmapAdapter().Create(host).(*view)
	tester := ui.NewTester(view.Draw, 960, 640)

	if !tester.HasText("节点属性") || view.currentLabel() != "中心" {
		t.Fatalf("canvas/inspector did not render: %#v", tester.Texts())
	}
	if serialized, err := view.Export("svg"); err != nil || !bytes.Contains(serialized, []byte("<svg")) {
		t.Fatalf("SVG export: %v %q", err, serialized)
	}
	if _, err := view.Export("png"); !errors.Is(err, ErrPNGUnsupported) {
		t.Fatalf("PNG error = %v", err)
	}
	if buffer.String() != testMindmap {
		t.Fatal("drawing changed the untouched buffer")
	}

	if err := tester.Click("添加子节点"); err != nil {
		t.Fatal(err)
	}
	parsed, _, err := mindmap.Parse([]byte(buffer.String()), mindmap.ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Root.Children) != 3 || view.selectedID == "" || changed == 0 {
		t.Fatalf("add child failed: children=%d selected=%q changed=%d", len(parsed.Root.Children), view.selectedID, changed)
	}

	if err := tester.Click("节点文本"); err != nil {
		t.Fatal(err)
	}
	tester.Command("selectAll")
	tester.Type("新节点名称")
	parsed, _, err = mindmap.Parse([]byte(buffer.String()), mindmap.ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	selected, ok := parsed.FindNode(view.selectedID)
	if !ok || selected.Topic != "新节点名称" {
		t.Fatalf("rename did not reach buffer: %#v", selected)
	}

	if err := tester.Click("预览画布"); err == nil {
		// There is no second canvas; this branch only documents that exported
		// labels are not used as canvas handles.
		t.Fatal("unexpected canvas label")
	}
	canvasX, canvasY, err := canvasCenter(view)
	if err != nil {
		t.Fatal(err)
	}
	tester.ClickAt(canvasX, canvasY)
	oldZoom := view.zoom
	tester.Pinch(canvasX, canvasY, 1.2)
	if view.zoom <= oldZoom {
		t.Fatalf("pinch did not zoom: %g -> %g", oldZoom, view.zoom)
	}

	beforeDelete := buffer.String()
	view.selectNode("left")
	tester.Key(0, ui.KeyDelete)
	if buffer.String() == beforeDelete {
		t.Fatal("Delete key did not edit the host buffer")
	}
	tester.Key(ui.Cmd, ui.KeyZ)
	if buffer.String() != beforeDelete {
		t.Fatal("undo did not restore the exact previous graph buffer")
	}
}

func TestMindmapMoveErrorsNeverMutateBuffer(t *testing.T) {
	buffer := text.New(testMindmap)
	report := 0
	view := MindmapAdapter().Create(editor.Host{
		Buffer: buffer,
		Report: func(error) { report++ },
	}).(*view)
	_ = ui.NewTester(view.Draw, 900, 600)
	before := buffer.String()
	view.moveMindmapNode("root", "left")
	if buffer.String() != before || report == 0 {
		t.Fatal("root move error changed buffer or was unreported")
	}
	view.moveMindmapNode("left", "right")
	if buffer.String() == before {
		t.Fatal("valid toolbar reparent did not edit")
	}
}

func TestDrawioDragShapeConnectAndUndo(t *testing.T) {
	buffer := text.New(testDrawio)
	var errorsSeen []error
	view := DrawioAdapter().Create(editor.Host{
		Buffer: buffer,
		Report: func(err error) { errorsSeen = append(errorsSeen, err) },
	}).(*view)
	tester := ui.NewTester(view.Draw, 1080, 680)
	if view.diagram == nil {
		t.Fatalf("draw.io parse failed: warnings=%v errors=%v", view.warnings, errorsSeen)
	}
	if view.canvasRect.W <= 0 || len(view.currentPage().Model.Nodes) != 2 {
		t.Fatalf("draw.io canvas did not render: warnings=%v nodes=%d", view.warnings, len(view.currentPage().Model.Nodes))
	}
	nodeA := findDrawioNode(view.currentPage().Model.Nodes, "a")
	if nodeA == nil {
		t.Fatal("node a missing")
	}
	x, y := nodeCenterOnWindow(view, *nodeA)
	startX := nodeA.X
	tester.Press(x, y)
	tester.Move(x+48, y+24)
	tester.Release(x+48, y+24)
	if len(errorsSeen) > 0 {
		t.Fatalf("drag reported errors: %v", errorsSeen)
	}
	parsed, _, err := drawio.Parse([]byte(buffer.String()), drawio.ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	moved := findDrawioNode(parsed.Pages[0].Model.Nodes, "a")
	if moved == nil || moved.X == startX {
		t.Fatalf("drag did not reach buffer: before=%g after=%#v", startX, moved)
	}
	if err := tester.Click("添加椭圆"); err != nil {
		t.Fatal(err)
	}
	parsed, _, err = drawio.Parse([]byte(buffer.String()), drawio.ParseOptions{})
	if err != nil || len(parsed.Pages[0].Model.Nodes) != 3 {
		t.Fatalf("add ellipse: err=%v nodes=%d", err, len(parsed.Pages[0].Model.Nodes))
	}

	// Two-step connector mode uses the selected node as the source, then the
	// next canvas selection as the target.
	view.selectNode(moved.ID)
	if err := tester.Click("连接两个节点"); err != nil {
		t.Fatal(err)
	}
	nodeB := findDrawioNode(view.currentPage().Model.Nodes, "b")
	x2, y2 := nodeCenterOnWindow(view, *nodeB)
	tester.ClickAt(x2, y2)
	parsed, _, err = drawio.Parse([]byte(buffer.String()), drawio.ParseOptions{})
	if err != nil || len(parsed.Pages[0].Model.Edges) != 1 {
		t.Fatalf("connector: err=%v edges=%d", err, len(parsed.Pages[0].Model.Edges))
	}

	tester.ClickAt(x2+10, y2+10)
	tester.Key(ui.Cmd, ui.KeyZ)
	parsed, _, err = drawio.Parse([]byte(buffer.String()), drawio.ParseOptions{})
	if err != nil || len(parsed.Pages[0].Model.Edges) != 0 {
		t.Fatalf("draw.io undo: err=%v edges=%d", err, len(parsed.Pages[0].Model.Edges))
	}
	svg, _, err := view.diagram.ExportSVG(0, drawio.SVGOptions{})
	if err != nil || !bytes.Contains(svg, []byte("<ellipse")) {
		t.Fatalf("draw.io SVG: err=%v svg=%q", err, svg)
	}
}

func TestInvalidBufferShowsFallbackWithoutEditing(t *testing.T) {
	buffer := text.New(testMindmap)
	var reported []error
	view := MindmapAdapter().Create(editor.Host{
		Buffer: buffer,
		Report: func(err error) { reported = append(reported, err) },
	}).(*view)
	_ = ui.NewTester(view.Draw, 800, 500)
	_ = buffer.BeginGroup()
	_ = buffer.ReplaceUTF16(0, buffer.LenUTF16(), `{broken`)
	buffer.EndGroup()
	before := buffer.String()
	tester := ui.NewTester(view.Draw, 800, 500)
	if len(reported) == 0 || buffer.String() != before || !tester.HasText("无法解析") {
		t.Fatalf("fallback changed buffer or stayed silent: report=%v text=%v", reported, tester.Texts())
	}
}

func TestDrawioSVGExportUsesSelectedPageWithDefaultBudget(t *testing.T) {
	source := `<mxfile>` + pageXML("p1", "PAGE_ONE") + pageXML("p2", "PAGE_TWO") + `</mxfile>`
	buffer := text.New(source)
	v := newView(kindDrawio, editor.Host{Buffer: buffer})
	v.ensureSynced(nil)
	if !v.changePage(1) || v.pageIndex != 1 {
		t.Fatal("could not select the second page")
	}
	before := buffer.String()
	svg, err := v.Export("svg")
	if err != nil {
		t.Fatal(err)
	}
	want, _, err := v.diagram.ExportSVG(1, drawio.SVGOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(svg, want) || !bytes.Contains(svg, []byte("PAGE_TWO")) || bytes.Contains(svg, []byte("PAGE_ONE")) {
		t.Fatal("view export did not use the selected page with default options")
	}
	if v.pageIndex != 1 || buffer.String() != before {
		t.Fatal("view export changed selected page or source")
	}
}

func nodeCenterOnWindow(view *view, node drawio.Node) (float32, float32) {
	localX, localY := view.worldToScreen(
		float32(node.X+node.Width/2),
		float32(node.Y+node.Height/2),
	)
	return view.canvasRect.X + localX, view.canvasRect.Y + localY
}

func canvasCenter(view *view) (float32, float32, error) {
	if view.canvasRect.W <= 1 || view.canvasRect.H <= 1 {
		return 0, 0, errors.New("canvas has no visible area")
	}
	return view.canvasRect.X + view.canvasRect.W/2, view.canvasRect.Y + view.canvasRect.H/2, nil
}

func findDrawioNode(nodes []drawio.Node, id string) *drawio.Node {
	for index := range nodes {
		if nodes[index].ID == id {
			return &nodes[index]
		}
	}
	return nil
}

func TestSourceViewContractExportsOnlySupportedFormats(t *testing.T) {
	for _, adapter := range []editor.Adapter{MindmapAdapter(), DrawioAdapter()} {
		if len(adapter.Exports) != 1 || adapter.Exports[0] != "svg" {
			t.Fatalf("%s exports = %#v", adapter.ID, adapter.Exports)
		}
		if strings.TrimSpace(adapter.Template()) == "" {
			t.Fatalf("%s has no source template", adapter.ID)
		}
	}
}
