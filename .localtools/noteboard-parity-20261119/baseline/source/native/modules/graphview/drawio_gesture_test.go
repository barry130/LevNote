package graphview

import (
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

func multiWorldWindow(v *view, x, y float64) (float32, float32) {
	return v.canvasRect.X + float32(x)*v.zoom + v.panX, v.canvasRect.Y + float32(y)*v.zoom + v.panY
}

func marqueeMulti(t *testing.T, tester *ui.Tester, v *view, x1, y1, x2, y2 float64, mods ui.Modifiers) {
	t.Helper()
	sx, sy := multiWorldWindow(v, x1, y1)
	ex, ey := multiWorldWindow(v, x2, y2)
	tester.Advance(time.Second)
	// Tester 的 Press/Move/Release 不携带修饰键（仅 ClickAtWith 支持），
	// 这里直接以构造的输入事件驱动同一套手势状态机。
	if !v.drawioPointerDown(ui.InputEvent{Kind: ui.InputPointerDown, X: sx, Y: sy, Mods: mods}) {
		t.Fatal("marquee down rejected")
	}
	if !v.moveDrawioPointer(ui.InputEvent{Kind: ui.InputPointerMove, X: ex, Y: ey}) {
		t.Fatal("marquee move rejected")
	}
	if !v.endDrawioPointer(ui.InputEvent{Kind: ui.InputPointerUp, X: ex, Y: ey}) {
		t.Fatal("marquee up rejected")
	}
}

func TestDrawioMarqueeContainmentUnionAndDirections(t *testing.T) {
	for _, tc := range []struct {
		name  string
		box   [4]float64
		union bool
		want  []string
	}{
		{"forward", [4]float64{10, 20, 290, 170}, false, []string{"a", "b"}},
		{"reverse", [4]float64{290, 170, 10, 20}, false, []string{"a", "b"}},
		{"partial", [4]float64{10, 20, 200, 170}, false, []string{"a"}},
		{"union", [4]float64{10, 20, 290, 170}, true, []string{"c", "a", "b"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, tester, buffer, changed := newMultiDrawio(t, multiDrawio, 1180, 780)
			clickMultiNode(t, tester, v, "c", 0)
			history, caret, version := buffer.History(), buffer.Selection(), buffer.Version()
			mods := ui.Shift
			if tc.union {
				mods |= ui.Cmd
			}
			marqueeMulti(t, tester, v, tc.box[0], tc.box[1], tc.box[2], tc.box[3], mods)
			if !slices.Equal(v.drawSelection.ids, tc.want) {
				t.Fatalf("marquee=%v want %v", v.drawSelection.ids, tc.want)
			}
			if buffer.String() != multiDrawio || buffer.History() != history || buffer.Selection() != caret || buffer.Version() != version || *changed != 0 {
				t.Fatal("marquee wrote source")
			}
		})
	}
}

func TestDrawioMarqueePreviewThresholdAndCancellation(t *testing.T) {
	for _, mode := range []string{"cancel", "escape", "tiny", "resize", "pan", "zoom", "source", "buffer", "release", "page"} {
		t.Run(mode, func(t *testing.T) {
			v, tester, buffer, changed := newMultiDrawio(t, multiDrawio, 1180, 780)
			clickMultiNode(t, tester, v, "c", 0)
			sx, sy := multiWorldWindow(v, 10, 20)
			ex, ey := multiWorldWindow(v, 290, 170)
			tester.Advance(time.Second)
			// Tester 的 Press/Move/Release 不携带修饰键（仅 ClickAtWith 支持），
			// 这里直接以构造的输入事件驱动同一套手势状态机。
			if !v.drawioPointerDown(ui.InputEvent{Kind: ui.InputPointerDown, X: sx, Y: sy, Mods: ui.Shift}) {
				t.Fatal("marquee down rejected")
			}
			if mode == "tiny" {
				ex, ey = sx+2, sy+2
			}
			if !v.moveDrawioPointer(ui.InputEvent{Kind: ui.InputPointerMove, X: ex, Y: ey}) {
				t.Fatal("marquee move rejected")
			}
			if !slices.Equal(v.drawSelection.ids, []string{"c"}) || buffer.String() != multiDrawio {
				t.Fatal("marquee move published selection/source")
			}
			if mode != "tiny" && !slices.Equal(v.drawGesture.candidates, []string{"a", "b"}) {
				t.Fatal("preview candidates", v.drawGesture.candidates)
			}
			switch mode {
			case "escape":
				tester.Key(0, ui.KeyEscape)
			case "resize":
				tester.SetSize(1100, 720)
			case "pan":
				tester.Scroll(sx, sy, 12, 18)
			case "zoom":
				tester.Pinch(sx, sy, 1.1)
			case "source":
				if err := buffer.ReplaceUTF16(buffer.LenUTF16(), buffer.LenUTF16(), " "); err != nil {
					t.Fatal(err)
				}
				tester.Frame()
			case "buffer":
				v.host.Buffer = text.New(multiDrawio)
				tester.Frame()
			case "release":
				v.Release()
			case "page":
				v.setPageIndex(1)
				v.setPageIndex(0)
				tester.Frame()
			}
			original, history, version := buffer.String(), buffer.History(), buffer.Version()
			v.endDrawioPointer(ui.InputEvent{Kind: ui.InputPointerUp, X: ex, Y: ey, Canceled: mode == "cancel"})
			if buffer.String() != original || buffer.History() != history || buffer.Version() != version || *changed != 0 {
				t.Fatal("canceled marquee wrote", mode)
			}
			if mode != "buffer" && mode != "release" && mode != "page" && !slices.Equal(v.drawSelection.ids, []string{"c"}) {
				t.Fatal("canceled marquee replaced starting selection", v.drawSelection.ids)
			}
		})
	}
}

func TestDrawioGroupDragCommonSnapAndEdgePreview(t *testing.T) {
	v, tester, buffer, changed := newMultiDrawio(t, multiDrawio, 1180, 780)
	selectMultiNodes(t, tester, v, "a", "b", "c")
	originals := slices.Clone(v.currentPage().Model.Nodes)
	edge := v.currentPage().Model.Edges[0]
	// diagramEdgePoints 复用共享 edgeScratch，快照必须先克隆再跨第二次调用比较。
	oldPoints := slices.Clone(func() [][2]float32 { p, _ := v.diagramEdgePoints(edge, v.canvasRect); return p }())
	a := findDrawioNode(originals, "a")
	sx, sy := nodeCenterOnWindow(v, *a)
	tester.Advance(time.Second)
	tester.Press(sx, sy)
	ex, ey := sx+14*v.zoom, sy+9*v.zoom
	tester.Move(ex, ey)
	if buffer.String() != multiDrawio || *changed != 0 {
		t.Fatal("preview committed")
	}
	if !slices.Equal(v.drawSelection.ids, []string{"a", "b", "c"}) || v.selectedID != "a" {
		t.Fatal("drag discarded selection")
	}
	for _, node := range originals {
		x, y := v.displayNodePosition(node)
		if math.Abs(x-(node.X+15.5)) > 1e-8 || math.Abs(y-(node.Y+9.75)) > 1e-8 {
			t.Fatal("inconsistent snapped preview", node.ID, x, y)
		}
		box := v.diagramNodeBox(node, v.canvasRect)
		hit, ok := v.drawioNodeAt(box.X-v.canvasRect.X+box.W/2, box.Y-v.canvasRect.Y+box.H/2)
		if !ok || hit != node.ID {
			t.Fatal("hit test differs from preview", node.ID, hit)
		}
	}
	points, ok := v.diagramEdgePoints(edge, v.canvasRect)
	if !ok || len(points) != len(oldPoints) || points[1] != oldPoints[1] || points[0] == oldPoints[0] || points[len(points)-1] == oldPoints[len(oldPoints)-1] {
		t.Fatal("edge preview moved bends or retained stale endpoints", oldPoints, points)
	}
	tester.Release(ex, ey)
	if *changed != 1 {
		t.Fatal("group drag should report once", *changed)
	}
	parsed := parseMultiDrawio(t, buffer)
	for _, before := range originals {
		after := findDrawioNode(parsed.Pages[0].Model.Nodes, before.ID)
		if after.X != before.X+15.5 || after.Y != before.Y+9.75 || after.Width != before.Width || after.Height != before.Height {
			t.Fatal("group offsets/dimensions", before.ID, after)
		}
	}
	if !strings.Contains(buffer.String(), `<mxPoint x="135.5" y="80.25"/>`) {
		t.Fatal("manual waypoint changed")
	}
	moved := buffer.String()
	tester.Key(ui.Cmd, ui.KeyZ)
	if buffer.String() != multiDrawio {
		t.Fatal("group undo not atomic")
	}
	tester.Key(ui.Cmd, ui.KeyY)
	if buffer.String() != moved {
		t.Fatal("group redo not exact")
	}
}

func TestDrawioGroupDragCanceledOrStaleNeverWrites(t *testing.T) {
	for _, mode := range []string{"cancel", "escape", "zero", "readonly", "group", "resize", "zoom", "pan", "source", "caret", "buffer", "release", "page"} {
		t.Run(mode, func(t *testing.T) {
			v, tester, buffer, changed := newMultiDrawio(t, multiDrawio, 1180, 780)
			selectMultiNodes(t, tester, v, "a", "b", "c")
			node := findDrawioNode(v.currentPage().Model.Nodes, "a")
			sx, sy := nodeCenterOnWindow(v, *node)
			tester.Advance(time.Second)
			tester.Press(sx, sy)
			ex, ey := sx+18*v.zoom, sy+10*v.zoom
			tester.Move(ex, ey)
			switch mode {
			case "cancel":
				tester.CancelPress()
			case "escape":
				tester.Key(0, ui.KeyEscape)
			case "zero":
				ex, ey = sx, sy
				tester.Move(ex, ey)
			case "readonly":
				v.host.ReadOnly = true
			case "group":
				if err := buffer.BeginGroup(); err != nil {
					t.Fatal(err)
				}
				defer buffer.CancelGroup()
			case "resize":
				tester.SetSize(1120, 760)
			case "zoom":
				tester.Pinch(sx, sy, 1.2)
			case "pan":
				tester.Scroll(sx, sy, 14, 10)
			case "source":
				if err := buffer.ReplaceUTF16(buffer.LenUTF16(), buffer.LenUTF16(), " "); err != nil {
					t.Fatal(err)
				}
				tester.Frame()
			case "caret":
				if err := buffer.SetSelection(3, 8); err != nil {
					t.Fatal(err)
				}
			case "buffer":
				v.host.Buffer = text.New(multiDrawio)
				tester.Frame()
			case "release":
				v.Release()
			case "page":
				v.setPageIndex(1)
				v.setPageIndex(0)
				tester.Frame()
			}
			source, history, version := buffer.String(), buffer.History(), buffer.Version()
			tester.Release(ex, ey)
			if buffer.String() != source || buffer.History() != history || buffer.Version() != version || *changed != 0 {
				t.Fatal("canceled group drag wrote", mode)
			}
		})
	}
}

func TestDrawioPendingLabelAndDragSeparateHistory(t *testing.T) {
	v, tester, buffer, changed := newMultiDrawio(t, multiDrawio, 1180, 780)
	stageStyleField(t, tester, v, "先写回文字🙂", 1180, 780)
	if *changed != 0 {
		t.Fatal("pending setup committed")
	}
	a := findDrawioNode(v.currentPage().Model.Nodes, "a")
	sx, sy := nodeCenterOnWindow(v, *a)
	tester.Advance(time.Second)
	tester.Press(sx, sy)
	tester.Move(sx+20*v.zoom, sy+8*v.zoom)
	if buffer.String() != multiDrawio || *changed != 0 {
		t.Fatal("drag flushed pending label before commit")
	}
	tester.Release(sx+20*v.zoom, sy+8*v.zoom)
	if *changed != 2 || v.pendingDraft() {
		t.Fatal("label+geometry should be two transactions", *changed, v.draftError)
	}
	after := buffer.String()
	tester.Key(ui.Cmd, ui.KeyZ)
	labelOnly := parseMultiDrawio(t, buffer)
	a = findDrawioNode(labelOnly.Pages[0].Model.Nodes, "a")
	if a.PlainText != "先写回文字🙂" || a.X != 20.5 || a.Y != 30.25 {
		t.Fatal("first undo should remove geometry only", a)
	}
	tester.Key(ui.Cmd, ui.KeyZ)
	if buffer.String() != multiDrawio {
		t.Fatal("second undo should remove label")
	}
	tester.Key(ui.Cmd, ui.KeyY)
	tester.Key(ui.Cmd, ui.KeyY)
	if buffer.String() != after {
		t.Fatal("two redo lost geometry/label")
	}
}

func TestDrawioPendingLabelNoMoveAndCompositionRetained(t *testing.T) {
	for _, mode := range []string{"zero", "cancel", "composition"} {
		t.Run(mode, func(t *testing.T) {
			v, tester, buffer, changed := newMultiDrawio(t, multiDrawio, 1180, 780)
			if mode == "composition" {
				if err := tester.Click("节点文本"); err != nil {
					t.Fatal(err)
				}
				tester.Command("selectAll")
				tester.Compose("正在组合", 4)
			} else {
				stageStyleField(t, tester, v, "保留草稿", 1180, 780)
			}
			source, history := buffer.String(), buffer.History()
			a := findDrawioNode(v.currentPage().Model.Nodes, "a")
			sx, sy := nodeCenterOnWindow(v, *a)
			tester.Advance(time.Second)
			tester.Press(sx, sy)
			if mode == "cancel" {
				tester.Move(sx+20, sy+20)
				tester.CancelPress()
			}
			tester.Release(sx, sy)
			if buffer.String() != source || buffer.History() != history || *changed != 0 || !v.pendingDraft() {
				t.Fatal("non-action flushed draft", mode, v.draftError)
			}
			if mode == "composition" && !v.CompositionActive() {
				t.Fatal("canvas press unmarked composition")
			}
		})
	}
}

func TestDrawioSelectionPrunesMissingAndResetsBufferPage(t *testing.T) {
	v, tester, buffer, _ := newMultiDrawio(t, multiDrawio, 1180, 780)
	selectMultiNodes(t, tester, v, "a", "b", "c")
	source := strings.Replace(multiDrawio, `id="b"`, `id="replacement"`, 1)
	if err := buffer.ReplaceUTF16(0, buffer.LenUTF16(), source); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if !slices.Equal(v.drawSelection.ids, []string{"a", "c"}) {
		t.Fatal("missing node not pruned", v.drawSelection.ids)
	}
	v.host.Buffer = text.New(multiDrawio)
	tester.Frame()
	if len(v.drawSelection.ids) != 0 || v.selectedID != "" {
		t.Fatal("buffer swap retained selection")
	}
	clickMultiNode(t, tester, v, "b", 0)
	v.setPageIndex(1)
	v.setPageIndex(0)
	v.lastSynced = ^uint64(0)
	tester.Frame()
	if len(v.drawSelection.ids) != 0 || v.selectedID != "" {
		t.Fatal("page round-trip retained selection")
	}
}
