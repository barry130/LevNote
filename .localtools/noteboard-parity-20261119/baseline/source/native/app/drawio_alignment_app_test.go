package app

import (
	"bytes"
	"fmt"
	"image"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/drawio"
)

const drawioAlignmentCanvas = "draw.io 画布：对齐页"

var drawioAlignmentOriginalGeometry = [3]string{
	`<mxGeometry x="-12.50" y="8.250" width="8.1e1" height="42.00" as="geometry"/>`,
	`<mxGeometry height='25.0' x='143.250' width='62.00' as='geometry' y='99.50'/>`,
	`<mxGeometry y="190.250" width="99.0" x="+48.75" height="3.1e1" as="geometry"/>`,
}

// Every expected tag and coordinate below is hand-calculated. No planner or
// serializer participates in constructing the disk-byte oracle.
var drawioAlignmentCases = []struct {
	label    string
	geometry [3]string
	xy       [3][2]float64
}{
	{"左对齐", [3]string{
		`<mxGeometry x="-12.50" y="8.250" width="8.1e1" height="42.00" as="geometry"/>`,
		`<mxGeometry height='25.0' x='-12.5' width='62.00' as='geometry' y='99.50'/>`,
		`<mxGeometry y="190.250" width="99.0" x="-12.5" height="3.1e1" as="geometry"/>`,
	}, [3][2]float64{{-12.5, 8.25}, {-12.5, 99.5}, {-12.5, 190.25}}},
	{"水平居中", [3]string{
		`<mxGeometry x="55.875" y="8.250" width="8.1e1" height="42.00" as="geometry"/>`,
		`<mxGeometry height='25.0' x='65.375' width='62.00' as='geometry' y='99.50'/>`,
		`<mxGeometry y="190.250" width="99.0" x="46.875" height="3.1e1" as="geometry"/>`,
	}, [3][2]float64{{55.875, 8.25}, {65.375, 99.5}, {46.875, 190.25}}},
	{"右对齐", [3]string{
		`<mxGeometry x="124.25" y="8.250" width="8.1e1" height="42.00" as="geometry"/>`,
		`<mxGeometry height='25.0' x='143.250' width='62.00' as='geometry' y='99.50'/>`,
		`<mxGeometry y="190.250" width="99.0" x="106.25" height="3.1e1" as="geometry"/>`,
	}, [3][2]float64{{124.25, 8.25}, {143.25, 99.5}, {106.25, 190.25}}},
	{"顶对齐", [3]string{
		`<mxGeometry x="-12.50" y="8.250" width="8.1e1" height="42.00" as="geometry"/>`,
		`<mxGeometry height='25.0' x='143.250' width='62.00' as='geometry' y='8.25'/>`,
		`<mxGeometry y="8.25" width="99.0" x="+48.75" height="3.1e1" as="geometry"/>`,
	}, [3][2]float64{{-12.5, 8.25}, {143.25, 8.25}, {48.75, 8.25}}},
	{"垂直居中", [3]string{
		`<mxGeometry x="-12.50" y="93.75" width="8.1e1" height="42.00" as="geometry"/>`,
		`<mxGeometry height='25.0' x='143.250' width='62.00' as='geometry' y='102.25'/>`,
		`<mxGeometry y="99.25" width="99.0" x="+48.75" height="3.1e1" as="geometry"/>`,
	}, [3][2]float64{{-12.5, 93.75}, {143.25, 102.25}, {48.75, 99.25}}},
	{"底对齐", [3]string{
		`<mxGeometry x="-12.50" y="179.25" width="8.1e1" height="42.00" as="geometry"/>`,
		`<mxGeometry height='25.0' x='143.250' width='62.00' as='geometry' y='196.25'/>`,
		`<mxGeometry y="190.250" width="99.0" x="+48.75" height="3.1e1" as="geometry"/>`,
	}, [3][2]float64{{-12.5, 179.25}, {143.25, 196.25}, {48.75, 190.25}}},
}

func drawioAlignmentSource(geometry [3]string) string {
	return strings.Join([]string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`<?levnote keep="processing-instruction"?>`,
		`<!-- 原样🙂 CRLF 与未知数据 -->`,
		`<mxfile host='external' vendor="keep &amp; exact" xmlns:v="urn:alignment-fixture">`,
		`  <diagram id="p1" name="对齐页"><mxGraphModel grid="0"><root>`,
		`    <mxCell id="0"/><mxCell id="1" parent="0"/>`,
		`    <mxCell id="a" value="甲🙂" style="shape=rectangle;fillColor=#b7416e;strokeColor=#211a32;" vertex="1" parent="1" vendor="保留甲">` + geometry[0] + `</mxCell>`,
		`    <mxCell parent='1' id='b' value='乙 &amp; B' vertex='1' style='shape=rectangle;fillColor=#2c8e84;' vendor='保留乙'>` + geometry[1] + `</mxCell>`,
		`    <mxCell id="c" value="丙" vertex="1" parent="1" style="shape=rectangle;fillColor=#5773c9;" vendor="保留丙">` + geometry[2] + `</mxCell>`,
		`    <mxCell id="untouched" value="未选" vertex="1" parent="1" style="shape=rectangle;fillColor=#a48339;"><mxGeometry x="280.00" y="260" width="40" height="30" as="geometry"/></mxCell>`,
		`    <mxCell id="edge" edge="1" source="a" target="c" parent="1" vendor="route"><mxGeometry relative="1" as="geometry"><Array as="points"><mxPoint x="11.00" y="112.0"/><mxPoint x="47" y="175"/></Array></mxGeometry></mxCell>`,
		`    <v:metadata preserve="001.00"><v:item><![CDATA[unknown <xml> & data]]></v:item></v:metadata>`,
		`  </root></mxGraphModel></diagram>`,
		// Fixed raw-DEFLATE/URI-encoded second page, generated independently.
		`  <diagram id="p2" name="压缩页">jVBBDsIwDHuNj0glBUGvjHUnHlGxiE7a6NQVGL9HLB0TByRusR0ncaCLbqyi6/0p1NxCl9BFDCFJ1Y0Fty1INTX0EUQKRCD7Q11Pqupd5Gv6x3AOXR95GLheDd71LP67a28sDSg3OFiYI8odzBbGwCoY+y72lNs5Jh5/XjBReX3FoeMUnyA1G3bieAokLfDR1MkLZSSC8txcfJ65yZwbBF8+c5ewIJvzznD566R9vf0F</diagram>`,
		`</mxfile>`,
		"",
	}, "\r\n")
}

func drawioAlignmentDisk(source string) []byte {
	return append([]byte{0xef, 0xbb, 0xbf}, []byte(source)...)
}

type drawioAlignmentAppOptions struct {
	source                             string
	customRoute, split, dark, readonly bool
	limit                              int
}

func openDrawioAlignmentApp(t *testing.T, options drawioAlignmentAppOptions) (*App, *Tab, *ui.Tester, string) {
	t.Helper()
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	if err := os.Mkdir(workspace, 0755); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Root, a.Dark = workspace, options.dark
	suffix := ".drawio"
	if options.customRoute {
		installAppPluginManifest(t, a, "diagram-notes.json", drawioRouteManifest)
		suffix = ".diagramnote"
	}
	source := options.source
	if source == "" {
		source = drawioAlignmentSource(drawioAlignmentOriginalGeometry)
	}
	path := filepath.Join(workspace, "alignment"+suffix)
	if err := os.WriteFile(path, drawioAlignmentDisk(source), 0600); err != nil {
		t.Fatal(err)
	}
	if options.readonly {
		if err := os.Chmod(path, 0400); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(path, 0600) })
	}
	a.refreshTreeNow()
	a.Open(path)
	tab := a.Current()
	if tab == nil || tab.Adapter == nil || tab.Adapter.ID != "drawio" || tab.Mode != "preview" ||
		tab.Doc.Buffer.String() != source || !tab.Doc.BOM || tab.Doc.Encoding != document.UTF8 || tab.Doc.Dirty() {
		t.Fatal("draw.io fixture did not open through the real registered App", a.Message)
	}
	if options.readonly && (!tab.Doc.ReadOnly || !tab.Editor.ReadOnly) {
		t.Fatal("readonly file was not exposed as readonly by the App")
	}
	if options.customRoute {
		requireDrawioRoute(t, tab)
	}
	if options.limit > 0 {
		// Install the public limited buffer before App creates its real Host.
		if tab.View != nil {
			t.Fatal("limited fixture must be bound before the first real view")
		}
		tab.Doc.Buffer = text.NewWithLimit(source, options.limit)
		tab.Editor.Buffer = tab.Doc.Buffer
		if tab.Doc.Dirty() {
			t.Fatal("limited buffer lost its disk baseline")
		}
	}
	if options.split && !a.setMode(tab, "split") {
		t.Fatal(a.Message)
	}
	tester := ui.NewTester(a.View, 1180, 780)
	tester.SetDark(options.dark)
	tester.Frame()
	tester.Frame()
	if _, ok := tester.Find(drawioAlignmentCanvas); !ok {
		t.Fatal("real draw.io canvas missing", tester.Texts())
	}
	return a, tab, tester, path
}

// Locate only fixture-colored pixels inside the actual canvas. The test does
// not duplicate fitCanvas/pan/zoom math or access private graphview selection.
func drawioAlignmentNodePoint(t *testing.T, tester *ui.Tester, id string) (float32, float32) {
	t.Helper()
	colors := map[string][3]uint8{"a": {0xb7, 0x41, 0x6e}, "b": {0x2c, 0x8e, 0x84}, "c": {0x57, 0x73, 0xc9}}
	want, ok := colors[id]
	if !ok {
		t.Fatalf("unknown fixture node %q", id)
	}
	canvas, ok := tester.Find(drawioAlignmentCanvas)
	if !ok {
		t.Fatal("fixture canvas missing")
	}
	pixels := tester.Image()
	clip := image.Rect(int(math.Ceil(float64(canvas.X))), int(math.Ceil(float64(canvas.Y))),
		int(math.Floor(float64(canvas.X+canvas.W))), int(math.Floor(float64(canvas.Y+canvas.H)))).Intersect(pixels.Bounds())
	minX, minY, maxX, maxY, count := clip.Max.X, clip.Max.Y, clip.Min.X, clip.Min.Y, 0
	for y := clip.Min.Y; y < clip.Max.Y; y++ {
		for x := clip.Min.X; x < clip.Max.X; x++ {
			pixel := pixels.RGBAAt(x, y)
			if pixel.R == want[0] && pixel.G == want[1] && pixel.B == want[2] && pixel.A == 255 {
				minX, minY, maxX, maxY = min(minX, x), min(minY, y), max(maxX, x), max(maxY, y)
				count++
			}
		}
	}
	if count < 16 || maxX-minX < 4 || maxY-minY < 4 || count*4 < (maxX-minX+1)*(maxY-minY+1) {
		t.Fatalf("node %s has no distinct rendered fill in canvas: count=%d box=(%d,%d)-(%d,%d)", id, count, minX, minY, maxX, maxY)
	}
	return float32(minX+maxX+1) / 2, float32(minY+maxY+1) / 2
}

func drawioAlignmentSelectNode(t *testing.T, tester *ui.Tester, id string, mods ui.Modifiers) {
	t.Helper()
	tester.Advance(time.Second)
	x, y := drawioAlignmentNodePoint(t, tester, id)
	tester.ClickAtWith(mods, x, y)
	tester.Frame()
	tester.Frame()
}

func drawioAlignmentSelectThree(t *testing.T, tester *ui.Tester) {
	t.Helper()
	drawioAlignmentSelectNode(t, tester, "a", 0)
	// An ordinary click on an already selected member preserves its group.
	// Clear through the real canvas keyboard route before constructing it.
	tester.Key(0, ui.KeyEscape)
	tester.Frame()
	drawioAlignmentSelectNode(t, tester, "a", 0)
	drawioAlignmentSelectNode(t, tester, "c", ui.Cmd)
	drawioAlignmentSelectNode(t, tester, "b", ui.Cmd)
	if !tester.HasText("已选择 3 个图形") {
		t.Fatal("Ctrl pointer selection did not reach three vertices", tester.Texts())
	}
}

func drawioAlignmentButton(t *testing.T, tester *ui.Tester, label string, disabled bool) ui.AccessInfo {
	t.Helper()
	button, ok := tester.AccessByRole(ui.RoleButton, label)
	if !ok || button.Disabled != disabled || button.Bounds.W < 38 || button.Bounds.H < 24 ||
		button.Bounds.X < 0 || button.Bounds.Y < 0 || button.Bounds.X+button.Bounds.W > 1180 || button.Bounds.Y+button.Bounds.H > 751 {
		t.Fatalf("alignment %q missing, clipped, or incorrectly enabled: %+v", label, button)
	}
	return button
}

func drawioAlignmentClick(t *testing.T, tester *ui.Tester, label string) {
	t.Helper()
	drawioAlignmentButton(t, tester, label, false)
	tester.Advance(time.Second)
	if err := tester.Click(label); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	tester.Frame()
}

func drawioAlignmentRequireGeometry(t *testing.T, source string, xy [3][2]float64) {
	t.Helper()
	doc, _, err := drawio.Parse([]byte(source), drawio.ParseOptions{})
	if err != nil || doc == nil || len(doc.Pages) != 2 || len(doc.Pages[0].Model.Nodes) != 4 {
		t.Fatalf("disk XML did not retain four vertices and two pages: %v", err)
	}
	for index, id := range []string{"a", "b", "c", "untouched"} {
		wantX, wantY, width, height := 280.0, 260.0, 40.0, 30.0
		if index < 3 {
			wantX, wantY = xy[index][0], xy[index][1]
			width, height = [3]float64{81, 62, 99}[index], [3]float64{42, 25, 31}[index]
		}
		found := false
		for _, node := range doc.Pages[0].Model.Nodes {
			if node.ID == id {
				found = true
				if node.X != wantX || node.Y != wantY || node.Width != width || node.Height != height || node.Parent != "1" {
					t.Fatalf("%s geometry = (%g,%g,%g,%g), want (%g,%g,%g,%g)", id, node.X, node.Y, node.Width, node.Height, wantX, wantY, width, height)
				}
			}
		}
		if !found {
			t.Fatalf("missing vertex %s", id)
		}
	}
	edges := doc.Pages[0].Model.Edges
	if len(edges) != 1 || edges[0].ID != "edge" || edges[0].SourceID != "a" || edges[0].TargetID != "c" ||
		len(edges[0].Points) != 2 || edges[0].Points[0] != (drawio.Point{X: 11, Y: 112}) || edges[0].Points[1] != (drawio.Point{X: 47, Y: 175}) {
		t.Fatal("alignment changed connection or manual waypoints", edges)
	}
	compressed := doc.Pages[1]
	if !compressed.Compressed || !compressed.ReadOnly || len(compressed.Model.Nodes) != 1 || compressed.Model.Nodes[0].ID != "compressed-shape" || compressed.Model.Nodes[0].PlainText != "保留🙂" {
		t.Fatal("compressed page fixture did not survive", compressed)
	}
}

func TestDrawioAlignmentAppSixCommandsSaveReopen(t *testing.T) {
	for _, split := range []bool{false, true} {
		for _, tc := range drawioAlignmentCases {
			t.Run(fmt.Sprintf("split-%v/%s", split, tc.label), func(t *testing.T) {
				a, tab, tester, path := openDrawioAlignmentApp(t, drawioAlignmentAppOptions{split: split, dark: split})
				before, buffer := tab.Doc.Buffer.String(), tab.Doc.Buffer
				// A directed UTF-16 selection around an astral scalar must survive
				// geometry edits in both preview and split source presentation.
				start := text.UTF16Len(before[:strings.Index(before, "🙂")])
				if err := buffer.SetSelection(start+2, start); err != nil {
					t.Fatal(err)
				}
				version, selection, history := buffer.Version(), buffer.Selection(), buffer.History()
				drawioAlignmentSelectThree(t, tester)
				if buffer.String() != before || buffer.Version() != version || buffer.Selection() != selection || buffer.History() != history || tab.Doc.Dirty() {
					t.Fatal("selection changed the document or source selection")
				}
				for _, action := range drawioAlignmentCases {
					drawioAlignmentButton(t, tester, action.label, false)
				}
				drawioAlignmentClick(t, tester, tc.label)
				want := drawioAlignmentSource(tc.geometry)
				if buffer.String() != want || buffer.Version() != version+1 || buffer.Selection() != selection || buffer.History().UndoDepth != history.UndoDepth+1 || !tab.Doc.Dirty() {
					t.Fatalf("%s did not commit one exact geometry transaction: version=%d history=%+v source=%s", tc.label, buffer.Version(), buffer.History(), buffer.String())
				}
				drawioAlignmentRequireGeometry(t, buffer.String(), tc.xy)
				if !bytes.Equal(mustReadFile(t, path), drawioAlignmentDisk(before)) {
					t.Fatal("alignment wrote disk before Save")
				}
				tester.Command("undo")
				tester.Frame()
				if buffer.String() != before || buffer.Selection() != selection || buffer.History().UndoDepth != history.UndoDepth || tab.Doc.Dirty() {
					t.Fatal("one undo did not restore the entire selection")
				}
				tester.Command("redo")
				tester.Frame()
				if buffer.String() != want || buffer.Selection() != selection {
					t.Fatal("one redo did not restore the entire alignment")
				}
				if !a.Save(tab, false) || tab.Doc.Dirty() || !bytes.Equal(mustReadFile(t, path), drawioAlignmentDisk(want)) {
					t.Fatal("Save changed exact BOM/CRLF/XML", a.Message)
				}
				if !a.CloseTab(0) {
					t.Fatal("clean aligned tab did not close", a.Message)
				}
				a.Open(path)
				tester.Frame()
				tester.Frame()
				reopened := a.Current()
				if reopened == nil || reopened == tab || reopened.Adapter == nil || reopened.Adapter.ID != "drawio" || reopened.Doc.Buffer.String() != want || !reopened.Doc.BOM || reopened.Doc.Encoding != document.UTF8 || reopened.Doc.Dirty() {
					t.Fatal("closed disk document did not reopen losslessly", a.Message)
				}
				if _, ok := tester.Find(drawioAlignmentCanvas); !ok {
					t.Fatal("reopened App did not render the graph")
				}
				drawioAlignmentRequireGeometry(t, reopened.Doc.Buffer.String(), tc.xy)
			})
		}
	}
}
