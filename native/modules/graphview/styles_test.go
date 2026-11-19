package graphview

import (
	"encoding/json"
	"errors"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/mindmap"
	"levnote.local/native/ui/editor"
)

func stylesFixture(style string) string {
	source := markerFixture([]string{"⭐", "custom"}, true)
	if style != "" {
		source = strings.Replace(source, `"unknownNode":`, `"style":`+style+`,"unknownNode":`, 1)
	}
	return source
}

func openStyles(t *testing.T, tester *ui.Tester) {
	t.Helper()
	revealFieldControl(t, tester, "展开颜色面板", 760, 480)
	chooseField(t, tester, "展开颜色面板")
}

func clickStyle(t *testing.T, tester *ui.Tester, label string) {
	t.Helper()
	revealFieldControl(t, tester, label, 760, 480)
	chooseField(t, tester, label)
}

// Ordinary native input writes through OnChange immediately. A source group
// blocks that write and retains the real editor draft until the color action.
func stageStyleField(t *testing.T, tester *ui.Tester, v *view, value string, width, height float32) {
	t.Helper()
	revealFieldControl(t, tester, "节点文本", width, height)
	chooseField(t, tester, "节点文本")
	source, history := v.host.Buffer.String(), v.host.Buffer.History()
	if err := v.host.Buffer.BeginGroup(); err != nil {
		t.Fatal(err)
	}
	tester.Command("selectAll")
	tester.Type(value)
	if v.host.Buffer.EndGroup() {
		t.Fatal("setup group unexpectedly wrote source")
	}
	tester.Frame()
	if v.host.Buffer.String() != source || v.host.Buffer.History() != history || !v.pendingDraft() || !errors.Is(v.draftError, text.ErrGroup) {
		t.Fatal("setup did not retain pending field", v.draftError, v.pendingDraft())
	}
}

func clickStyleNode(t *testing.T, tester *ui.Tester, v *view, id string) {
	t.Helper()
	for _, node := range v.layout.Nodes {
		if node.ID == id {
			x := v.canvasRect.X + float32(node.X+node.Width/2)*v.zoom + v.panX
			y := v.canvasRect.Y + float32(node.Y+node.Height/2)*v.zoom + v.panY
			tester.ClickAt(x, y)
			if v.selectedID != id {
				t.Fatal("canvas did not select node", id)
			}
			return
		}
	}
	t.Fatal("missing scene node", id)
}

func expectStyle(t *testing.T, buffer *text.Buffer, id, key, want string, present bool) {
	t.Helper()
	node, ok := parseMind(t, buffer).FindNode(id)
	if !ok {
		t.Fatal("missing node", id)
	}
	var got string
	exists := false
	if node.Style != nil {
		got, exists = objectString(*node.Style, key)
	}
	if exists != present || present && got != want {
		t.Fatalf("%s.%s = %q present=%v, want %q present=%v", id, key, got, exists, want, present)
	}
}

func TestStylesNativePointerKeyboardUndoAndMetadata(t *testing.T) {
	for _, mode := range []string{"pointer", "enter", "space"} {
		t.Run(mode, func(t *testing.T) {
			source := stylesFixture(`{"future":{"n":1.20},"border":1e3}`)
			buffer := text.New(source)
			_ = buffer.SetSelection(11, 3)
			selection := buffer.Selection()
			changed := 0
			v := newView(kindMindmap, editor.Host{Buffer: buffer, Changed: func() { changed++ }})
			tester := ui.NewTester(v.Draw, 760, 480)
			clickStyleNode(t, tester, v, "root")
			openStyles(t, tester)
			if mode == "pointer" {
				clickStyle(t, tester, "文字色：蓝")
			} else {
				for i := 0; i < 40 && !tester.Focused("文字色：蓝"); i++ {
					tester.Key(0, ui.KeyTab)
				}
				if !tester.Focused("文字色：蓝") {
					t.Fatal("Tab did not reach color")
				}
				key := ui.KeyEnter
				if mode == "space" {
					key = ui.KeySpace
				}
				tester.KeyDown(key)
				tester.KeyRepeat(0, key)
				tester.KeyUp(key)
			}
			expectStyle(t, buffer, "root", "color", "#1e40af", true)
			if changed != 1 || buffer.History().UndoDepth != 1 || buffer.Selection() != selection {
				t.Fatal("color was not one selection-preserving transaction", changed, buffer.History(), buffer.Selection(), v.styleError)
			}
			committed := buffer.String()
			var before, after map[string]any
			_ = json.Unmarshal([]byte(source), &before)
			_ = json.Unmarshal([]byte(committed), &after)
			delete(after["root"].(map[string]any)["style"].(map[string]any), "color")
			if !reflect.DeepEqual(before, after) {
				t.Fatal("unrelated content changed")
			}
			for _, literal := range []string{"1.00", "1.20", "2.00", "0.30", "1e3"} {
				if !strings.Contains(committed, literal) {
					t.Fatal("raw number changed", literal)
				}
			}
			clickStyleNode(t, tester, v, "root")
			tester.Key(ui.Cmd, ui.KeyZ)
			if buffer.String() != source || buffer.Selection() != selection {
				t.Fatal("native undo did not restore exact bytes/selection")
			}
			tester.Key(ui.Cmd|ui.Shift, ui.KeyZ)
			if buffer.String() != committed || buffer.Selection() != selection {
				t.Fatal("native redo did not restore exact bytes/selection")
			}
			clickStyle(t, tester, "文字色：蓝")
			if buffer.String() != committed || buffer.History().UndoDepth != 1 || changed != 3 {
				t.Fatal("same color created history", changed, buffer.History())
			}
		})
	}
}

func TestStylesNativeDefaultsCustomValuesAndReadOnly(t *testing.T) {
	source := stylesFixture(`{"color":"rebeccapurple","background-color":"var(custom)","border":7,"future":1.20}`)
	buffer := text.New(source)
	v := newView(kindMindmap, editor.Host{Buffer: buffer})
	tester := ui.NewTester(v.Draw, 760, 480)
	if _, ok := tester.AccessByRole(ui.RoleButton, "文字色：蓝"); ok {
		t.Fatal("closed palette built buttons")
	}
	openStyles(t, tester)
	for _, label := range []string{"当前文字色：rebeccapurple", "当前底色：var(custom)（原值保留，显示默认）", "当前边框色：7（原值保留，显示默认）"} {
		if _, ok := tester.Find(label); !ok {
			t.Fatal("missing actual value", label)
		}
	}
	if !tester.HasText("文字色：rebeccapurple") || !tester.HasText("底色：var(custom)（原值保留，显示默认）") || buffer.String() != source || buffer.CanUndo() {
		t.Fatal("opening changed or hid custom value")
	}
	clickStyle(t, tester, "文字色恢复默认")
	expectStyle(t, buffer, "root", "color", "", false)
	node := parseMind(t, buffer).Root
	for _, key := range []string{"background-color", "border", "future"} {
		if _, ok := node.Style.Get(key); !ok {
			t.Fatal("reset erased another property", key)
		}
	}
	if buffer.History().UndoDepth != 1 {
		t.Fatal("reset not one undo")
	}
	v.host.ReadOnly = true
	tester.Frame()
	before, history := buffer.String(), buffer.History()
	clickStyle(t, tester, "底色：浅蓝")
	a, ok := tester.AccessByRole(ui.RoleButton, "底色：浅蓝")
	if !ok || !a.Disabled || buffer.String() != before || buffer.History() != history {
		t.Fatal("readonly color edited")
	}
	g := v.captureStyle(v.stylePresentation(), 1, 1)
	if err := v.activateStyle(g); !errors.Is(err, errStyleReadOnly) {
		t.Fatal("direct readonly path", err)
	}
	v.host.ReadOnly = false
	tester.Frame()
	clickStyle(t, tester, "文字色恢复默认")
	if buffer.String() != before || buffer.History() != history {
		t.Fatal("already-default reset created edit")
	}
}

func TestStylesNativeFlushCompositionAndConflict(t *testing.T) {
	t.Run("field-separate-undo", func(t *testing.T) {
		source := stylesFixture("")
		buffer := text.New(source)
		changed := 0
		v := newView(kindMindmap, editor.Host{Buffer: buffer, Changed: func() { changed++ }})
		tester := ui.NewTester(v.Draw, 760, 480)
		stageStyleField(t, tester, v, "已确认文字🙂", 760, 480)
		openStyles(t, tester)
		clickStyle(t, tester, "文字色：绿")
		if parseMind(t, buffer).Root.Topic != "已确认文字🙂" || changed != 2 || buffer.History().UndoDepth != 2 {
			t.Fatal("field and style did not commit separately", v.styleError, buffer.History(), changed)
		}
		expectStyle(t, buffer, "root", "color", "#166534", true)
		clickStyleNode(t, tester, v, "root")
		tester.Key(ui.Cmd, ui.KeyZ)
		if parseMind(t, buffer).Root.Topic != "已确认文字🙂" {
			t.Fatal("color undo reverted field")
		}
		expectStyle(t, buffer, "root", "color", "", false)
		tester.Key(ui.Cmd, ui.KeyZ)
		if buffer.String() != source {
			t.Fatal("second undo did not restore field")
		}
	})
	t.Run("composition", func(t *testing.T) {
		buffer := text.New(stylesFixture(""))
		v := newView(kindMindmap, editor.Host{Buffer: buffer})
		tester := ui.NewTester(v.Draw, 760, 480)
		openStyles(t, tester)
		revealFieldControl(t, tester, "节点文本", 760, 480)
		composeField(t, tester, "节点文本", "组合🙂")
		before := buffer.String()
		clickStyle(t, tester, "文字色：蓝")
		if buffer.String() != before || buffer.CanUndo() {
			t.Fatal("composition changed source through color", v.styleError)
		}
		if !v.pendingDraft() {
			t.Fatal("composition draft lost")
		}
	})
	t.Run("conflict", func(t *testing.T) {
		buffer := text.New(stylesFixture(""))
		v := newView(kindMindmap, editor.Host{Buffer: buffer})
		tester := ui.NewTester(v.Draw, 760, 480)
		stageStyleField(t, tester, v, "待恢复文字", 760, 480)
		candidate := parseMind(t, buffer)
		if err := candidate.RenameNode("root", "外部版本"); err != nil {
			t.Fatal(err)
		}
		raw, err := candidate.Serialize(mindmap.SerializeOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if err = buffer.ReplaceUTF16(0, buffer.LenUTF16(), string(raw)); err != nil {
			t.Fatal(err)
		}
		tester.Frame()
		before, history := buffer.String(), buffer.History()
		openStyles(t, tester)
		clickStyle(t, tester, "文字色：蓝")
		if !errors.Is(v.styleError, ErrDraftConflict) || buffer.String() != before || buffer.History() != history || !v.pendingDraft() {
			t.Fatal("conflicting field overwritten", v.styleError)
		}
	})
}

func TestStylesNativeHeldGestureIdentityCancelAndScroll(t *testing.T) {
	for _, mode := range []string{"node", "node-roundtrip", "buffer", "version", "selection", "release", "cancel", "outside", "scroll"} {
		t.Run(mode, func(t *testing.T) {
			buffer := text.New(stylesFixture(""))
			v := newView(kindMindmap, editor.Host{Buffer: buffer})
			tester := ui.NewTester(v.Draw, 760, 480)
			openStyles(t, tester)
			revealFieldControl(t, tester, "文字色：蓝", 760, 480)
			r, _ := tester.Find("文字色：蓝")
			x, y := r.X+r.W/2, r.Y+r.H/2
			tester.Advance(time.Second)
			tester.Press(x, y)
			if !v.stylePress.active {
				t.Fatal("pointer did not capture action")
			}
			switch mode {
			case "node":
				v.selectNode("left")
			case "node-roundtrip":
				v.selectNode("left")
				v.selectNode("root")
			case "buffer":
				v.host.Buffer = text.New(buffer.String())
			case "version":
				_ = buffer.ReplaceUTF16(buffer.LenUTF16(), buffer.LenUTF16(), " ")
			case "selection":
				_ = buffer.SetSelection(1, 1)
			case "release":
				v.Release()
			case "cancel":
				tester.CancelPress()
			case "scroll":
				tester.Wheel(750, 445, 0, 90)
				tester.Advance(time.Second)
			}
			original, current := buffer.String(), v.host.Buffer.String()
			history, activeHistory := buffer.History(), v.host.Buffer.History()
			if mode == "outside" {
				tester.Release(10, 10)
			} else {
				tester.Release(x, y)
			}
			if buffer.String() != original || v.host.Buffer.String() != current || buffer.History() != history || v.host.Buffer.History() != activeHistory {
				t.Fatal("stale gesture edited source", mode, v.styleError)
			}
			expectStyle(t, buffer, "root", "color", "", false)
			if v.stylePress.active {
				t.Fatal("released gesture retained")
			}
		})
	}
}

func TestStylesRejectChangedReentryDuringFlush(t *testing.T) {
	for _, mode := range []string{"buffer", "release", "readonly", "selection", "source", "node"} {
		t.Run(mode, func(t *testing.T) {
			buffer := text.New(stylesFixture(""))
			v := newView(kindMindmap, editor.Host{Buffer: buffer})
			tester := ui.NewTester(v.Draw, 760, 480)
			stageStyleField(t, tester, v, "待提交字段", 760, 480)
			calls, reports := 0, 0
			v.host.Report = func(error) { reports++ }
			v.host.Changed = func() {
				calls++
				if calls != 1 {
					t.Fatal("style unexpectedly called Changed after invalidated flush")
				}
				switch mode {
				case "buffer":
					v.host.Buffer = text.New(stylesFixture(""))
				case "release":
					v.Release()
				case "readonly":
					v.host.ReadOnly = true
				case "selection":
					_ = buffer.SetSelection(1, 1)
				case "source":
					_ = buffer.ReplaceUTF16(buffer.LenUTF16(), buffer.LenUTF16(), " ")
				case "node":
					v.selectedID = "left"
				}
			}
			openStyles(t, tester)
			clickStyle(t, tester, "文字色：蓝")
			if calls != 1 || reports == 0 {
				t.Fatal("invalidated field receipt accepted", calls, v.styleError)
			}
			expectStyle(t, buffer, "root", "color", "", false)
			expectStyle(t, v.host.Buffer, "root", "color", "", false)
		})
	}
}

// Tester.Find returns clipped rectangles: a positive height alone can expose
// only a sliver. Capture targets must show their complete 28px button.
func revealStyleCapture(t *testing.T, tester *ui.Tester, label string) {
	t.Helper()
	tester.Wheel(754, 445, 0, -3000)
	tester.Advance(time.Second)
	for i := 0; i < 40; i++ {
		r, ok := tester.Find(label)
		if ok && r.W > 0 && r.H >= 27.9 && r.X >= 0 && r.Y >= 0 && r.X+r.W <= 760.1 && r.Y+r.H <= 480.1 {
			return
		}
		tester.Wheel(754, 445, 0, 30)
		tester.Advance(time.Second)
	}
	t.Fatalf("%s not fully visible for capture", label)
}

func TestStylesNativeCanvasSVGAndOptionalCapture(t *testing.T) {
	for _, dark := range []bool{false, true} {
		t.Run(fmt.Sprintf("dark=%v-width=760", dark), func(t *testing.T) {
			buffer := text.New(stylesFixture(""))
			v := newView(kindMindmap, editor.Host{Buffer: buffer})
			tester := ui.NewTester(v.Draw, 760, 480)
			tester.SetDark(dark)
			tester.Frame()
			openStyles(t, tester)
			for _, label := range []string{"文字色：蓝", "底色：浅黄", "边框色：绿"} {
				clickStyle(t, tester, label)
			}
			// Inspect the real painter output inside the selected node: its green
			// document border must remain visible beneath the separate focus ring.
			image := tester.Image()
			greenPixels := 0
			for _, item := range v.layout.Nodes {
				if item.ID != "root" {
					continue
				}
				x := int(v.canvasRect.X + float32(item.X)*v.zoom + v.panX)
				y := int(v.canvasRect.Y + float32(item.Y)*v.zoom + v.panY)
				for row := y; row < y+int(float32(item.Height)*v.zoom); row++ {
					for col := x; col < x+int(float32(item.Width)*v.zoom); col++ {
						pixel := image.RGBAAt(col, row)
						if int(pixel.G) > int(pixel.R)+20 && int(pixel.G) > int(pixel.B)+20 {
							greenPixels++
						}
					}
				}
			}
			if greenPixels < 4 {
				t.Fatal("selected actual border was replaced by focus accent", greenPixels)
			}
			node := parseMind(t, buffer).Root
			fill, stroke, ink := mindmapNodeColors(node, ui.Color{}, ui.Color{}, ui.Color{})
			if fill != ui.Hex("#fef3c7") || stroke != ui.Hex("#166534") || ink != ui.Hex("#1e40af") {
				t.Fatal("canvas colors differ from palette")
			}
			svg, err := v.Export("svg")
			if err != nil {
				t.Fatal(err)
			}
			for _, literal := range []string{`fill="#fef3c7"`, `stroke="#166534"`, `fill="#1e40af"`} {
				if !strings.Contains(string(svg), literal) {
					t.Fatal("SVG color absent", literal)
				}
			}
			original, history := buffer.String(), buffer.History()
			for _, label := range []string{"文字色：蓝", "底色：浅黄", "边框色：绿", "文字色恢复默认", "底色恢复默认", "边框色恢复默认"} {
				revealFieldControl(t, tester, label, 760, 480)
				a, ok := tester.AccessByRole(ui.RoleButton, label)
				if !ok || a.Disabled {
					t.Fatal("color control inaccessible", label, a)
				}
			}
			for _, stage := range []struct{ name, label string }{{"text", "文字色恢复默认"}, {"fill-border", "边框色恢复默认"}} {
				revealStyleCapture(t, tester, stage.label)
				tester.Advance(time.Second)
				if out := os.Getenv("LEVNOTE_GRAPH_STYLE_SCREENSHOT_DIR"); out != "" {
					if err := os.MkdirAll(out, 0755); err != nil {
						t.Fatal(err)
					}
					f, err := os.Create(filepath.Join(out, fmt.Sprintf("node-colors-dark-%v-760-%s.png", dark, stage.name)))
					if err != nil {
						t.Fatal(err)
					}
					encodeErr, closeErr := png.Encode(f, tester.Image()), f.Close()
					if encodeErr != nil || closeErr != nil {
						t.Fatal(encodeErr, closeErr)
					}
				}
			}
			tester.Frame()
			tester.Frame()
			tester.Advance(time.Second)
			if tester.FramePending() || buffer.String() != original || buffer.History() != history {
				t.Fatal("idle style panel changed source or kept invalidating")
			}
		})
	}
}
