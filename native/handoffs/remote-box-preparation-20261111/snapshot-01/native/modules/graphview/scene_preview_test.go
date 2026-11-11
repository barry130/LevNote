package graphview

import (
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/enginescene"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

type sceneRequestRecord struct {
	request  editor.SceneRequest
	complete func(editor.SceneResult, error)
	canceled bool
}

func literalNativeScene() editor.SceneResult {
	return editor.SceneResult{Scene: enginescene.Scene{Schema: 1, Layout: "logicalStructureRight",
		Bounds: enginescene.Rect{Width: 312, Height: 96},
		Nodes: []enginescene.Node{
			{ID: "root", Label: "中🙂 Root", X: 28, Y: 28, Width: 92, Height: 40, Shape: "rounded", Root: true},
			{ID: "fold", Label: "折叠", X: 192, Y: 28, Width: 92, Height: 40, Shape: "rounded", Collapsed: true, Fill: "#ff0000ff"},
		},
		Edges: []enginescene.Edge{{FromID: "root", ToID: "fold", From: enginescene.Point{X: 120, Y: 48}, To: enginescene.Point{X: 192, Y: 48}, Stroke: "#123456ff"}},
	}}
}

func nativeSceneFixture(t *testing.T) (*sceneView, *[]*sceneRequestRecord, *int) {
	t.Helper()
	requests := []*sceneRequestRecord{}
	sourceActions := 0
	capability := *MindmapAdapter().SceneEngine
	host := editor.Host{Buffer: text.New("invalid JSON must never be parsed here"), ReadOnly: true, SceneEngine: &capability}
	host.BeginScene = func(owner editor.View, request editor.SceneRequest, complete func(editor.SceneResult, error)) (func(), error) {
		if _, ok := owner.(*sceneView); !ok || request.Capability != capability {
			t.Fatal("wrong generic owner/capability")
		}
		record := &sceneRequestRecord{request: request, complete: complete}
		requests = append(requests, record)
		return func() { record.canceled = true }, nil
	}
	host.ShowSource = func(editor.View) bool { sourceActions++; return true }
	view, ok := MindmapAdapter().Create(host).(*sceneView)
	if !ok {
		t.Fatal("configured readonly adapter did not choose scene view")
	}
	return view, &requests, &sourceActions
}

func TestNativeSceneLifecycleStaleAndNoParserFallback(t *testing.T) {
	v, requests, _ := nativeSceneFixture(t)
	before, version, history, selection := v.host.Buffer.String(), v.host.Buffer.Version(), v.host.Buffer.History(), v.host.Buffer.Selection()
	if warnings := v.Warnings(); len(warnings) != 1 || !strings.Contains(warnings[0], "加载") || len(*requests) != 1 {
		t.Fatal("initial request", warnings)
	}
	tester := ui.NewTester(v.Draw, 760, 480)
	tester.Frame()
	tester.Frame()
	if len(*requests) != 1 {
		t.Fatal("idle requested again")
	}
	(*requests)[0].complete(literalNativeScene(), nil)
	tester.Frame()
	if v.frameNodes != 2 || v.frameEdges != 1 || len(v.Warnings()) != 0 {
		t.Fatal("literal scene not painted")
	}
	if v.host.Buffer.String() != before || v.host.Buffer.Version() != version || v.host.Buffer.History() != history || v.host.Buffer.Selection() != selection {
		t.Fatal("readonly scene changed source")
	}
	if err := v.host.Buffer.ReplaceUTF16(0, v.host.Buffer.LenUTF16(), "new invalid source"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if len(*requests) != 2 || len(v.scene.Nodes) != 2 || !v.stale || !strings.Contains(v.Warnings()[0], "旧内容") {
		t.Fatal("missing bounded stale state")
	}
	(*requests)[0].complete(editor.SceneResult{}, errors.New("stale callback"))
	if v.err != nil || !v.loading {
		t.Fatal("stale completion replaced active generation")
	}
	v.Release()
	if !(*requests)[1].canceled || len(v.scene.Nodes) != 0 || v.index != nil || v.labels.layouts != nil {
		t.Fatal("release retained heavy state")
	}
	(*requests)[1].complete(literalNativeScene(), nil)
	if len(v.scene.Nodes) != 0 {
		t.Fatal("late completion revived released view")
	}
	tester.Frame()
	if len(*requests) != 3 || (*requests)[2].request.Epoch <= (*requests)[1].request.Epoch {
		t.Fatal("same view did not reactivate")
	}
	(*requests)[2].complete(literalNativeScene(), nil)
	tester.Frame()
	if tester.FramePending() {
		t.Fatal("idle scene causes perpetual frames")
	}
}

func TestNativeSceneFailureNoRetryAndMissingProvider(t *testing.T) {
	v, requests, _ := nativeSceneFixture(t)
	tester := ui.NewTester(v.Draw, 760, 480)
	(*requests)[0].complete(editor.SceneResult{}, errors.New("missing worker"))
	for i := 0; i < 4; i++ {
		tester.Frame()
		_ = v.Warnings()
	}
	if len(*requests) != 1 || !strings.Contains(v.Warnings()[0], "missing worker") {
		t.Fatal("failure invisible or retried")
	}
	capability := *MindmapAdapter().SceneEngine
	missing := MindmapAdapter().Create(editor.Host{ReadOnly: true, Buffer: text.New(testMindmap), SceneEngine: &capability})
	if _, ok := missing.(*sceneView); !ok || !strings.Contains(missing.Warnings()[0], "缺少场景引擎") {
		t.Fatal("missing provider fell back")
	}
	if _, err := missing.Export("svg"); err == nil {
		t.Fatal("scene export parsed source")
	}
	if _, ok := MindmapAdapter().Create(editor.Host{Buffer: text.New(testMindmap), SceneEngine: &capability}).(*view); !ok {
		t.Fatal("editable path changed")
	}
	if _, ok := DrawioAdapter().Create(editor.Host{ReadOnly: true, Buffer: text.New(testDrawio), SceneEngine: &capability}).(*view); !ok {
		t.Fatal("drawio path changed")
	}
}

func TestNativeSceneControlsOutlineThemeAndFirstPaintCenter(t *testing.T) {
	v, requests, sourceActions := nativeSceneFixture(t)
	_ = v.Warnings()
	(*requests)[0].complete(literalNativeScene(), nil)
	if err := v.NavigateOutlineItem("fold"); err != nil || v.pendingCenter != "fold" {
		t.Fatal("lazy outline navigation", err)
	}
	tester := ui.NewTester(v.Draw, 760, 480)
	tester.Frame()
	if v.pendingCenter != "" || math.Abs(float64(v.panX+(192+46)*v.zoom-v.canvasRect.W/2)) > 0.01 || math.Abs(float64(v.panY+48*v.zoom-v.canvasRect.H/2)) > 0.01 {
		t.Fatal("first positive canvas failed to center")
	}
	beforeSelection, beforeHistory := v.host.Buffer.Selection(), v.host.Buffer.History()
	if err := v.NavigateOutlineItem("hidden"); !errors.Is(err, editor.ErrOutlineReadOnlyHidden) || v.selectedID != "fold" {
		t.Fatal("hidden readonly target", err)
	}
	zoom := v.zoom
	if err := tester.Click("放大画布"); err != nil || v.zoom <= zoom {
		t.Fatal("zoom control", err)
	}
	if err := tester.Click("适应画布"); err != nil {
		t.Fatal(err)
	}
	box := v.canvasRect
	tester.Scroll(box.X+box.W/2, box.Y+box.H/2, 17, 23)
	x, y := v.panX, v.panY
	tester.ClickAt(box.X+5, box.Y+5)
	tester.Key(0, ui.KeyRight)
	if v.panX == x && v.panY == y {
		t.Fatal("real canvas key did not pan")
	}
	if err := v.NavigateOutlineItem("root"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	// Tab traversal from the actual canvas reaches the ordinary source button.
	for i := 0; i < 8 && *sourceActions == 0; i++ {
		tester.Key(0, ui.KeyTab)
		if source, ok := tester.AccessByRole(ui.RoleButton, "查看只读文档源码"); ok && source.Focused {
			tester.Key(0, ui.KeyEnter)
		}
	}
	if *sourceActions != 1 {
		t.Fatal("source action unreachable by keyboard")
	}
	light := ui.LightTheme()
	dark := ui.DarkTheme()
	node := literalNativeScene().Scene.Nodes[0]
	fill, stroke, _ := sceneNodeColors(node, "", light)
	if fill != light.Accent.Alpha(0.16) || stroke != light.Accent {
		t.Fatal("root roles")
	}
	node.Fill, node.Stroke, node.Text = "#01020304", "#112233ff", "#445566ff"
	fill, stroke, color := sceneNodeColors(node, "root", dark)
	if fill != ui.Hex("#01020304") || stroke != dark.Accent || color != ui.Hex("#445566ff") {
		t.Fatal("literal/selection precedence")
	}
	tester.SetDark(true)
	tester.Frame()
	tester.Advance(time.Second)
	if len(*requests) != 1 || v.host.Buffer.Selection() != beforeSelection || v.host.Buffer.History() != beforeHistory {
		t.Fatal("theme/navigation changed source or requested")
	}
}

func TestNativeSceneTinyDimensionsFiniteTransformsAndLabelBudget(t *testing.T) {
	v, requests, _ := nativeSceneFixture(t)
	tester := ui.NewTester(v.Draw, 760, 480)
	result := literalNativeScene()
	result.Scene.Bounds = enginescene.Rect{X: 1e9, Y: -1e9, Width: math.SmallestNonzeroFloat64, Height: math.SmallestNonzeroFloat64}
	for i := range result.Scene.Nodes {
		result.Scene.Nodes[i].X, result.Scene.Nodes[i].Y = 1e9, -1e9
		result.Scene.Nodes[i].Width, result.Scene.Nodes[i].Height = math.SmallestNonzeroFloat64, math.SmallestNonzeroFloat64
	}
	(*requests)[0].complete(result, nil)
	tester.Frame()
	v.fitScene()
	v.zoomAt(10, 10, 1.2)
	if !finiteSceneInput(v.zoom, v.panX, v.panY) || v.zoom <= 0 {
		t.Fatal("subnormal dimension produced nonfinite transform")
	}
	oldX, oldY := v.panX, v.panY
	v.sceneInput(ui.InputEvent{Kind: ui.InputScroll, DX: float32(math.Inf(1))})
	if v.panX != oldX || v.panY != oldY {
		t.Fatal("invalid input changed pan")
	}
	v.scene = literalNativeScene().Scene
	v.labels = labelCache{}
	v.selectedID = ""
	v.zoom, v.panX, v.panY = 0.2, 20, 20
	tester.Frame()
	if v.frameHiddenLabels != 2 || !tester.HasText("概览中部分标签已隐藏；放大或选中节点查看") {
		t.Fatal("hidden labels not disclosed")
	}
	if tester.FramePending() {
		t.Fatal("finite scene does not settle")
	}
}
