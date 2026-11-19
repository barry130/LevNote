package graphview

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/drawio"
	"levnote.local/native/ui/editor"
)

const multiDrawio = `<mxfile custom="keep-🙂"><diagram id="p1" name="布局"><mxGraphModel dx="1024"><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="a" value="开始" vertex="1" parent="1" style="rounded=1;fillColor=#dae8fc;"><mxGeometry x="20.5" y="30.25" width="80.0" height="40.00" as="geometry"/></mxCell><mxCell id="b" value="处理" vertex="1" parent="1" style="fillColor=#d5e8d4;"><mxGeometry x="160.25" y="100.5" width="120.00" height="60" as="geometry"/></mxCell><mxCell id="c" value="确认" vertex="1" parent="1" style="ellipse;fillColor=#fff2cc;"><mxGeometry x="310.75" y="200.75" width="60" height="100.0" as="geometry"/></mxCell><mxCell id="edge" edge="1" parent="1" source="a" target="b"><mxGeometry relative="1" as="geometry"><Array as="points"><mxPoint x="135.5" y="80.25"/></Array></mxGeometry></mxCell><!--unknown spelling--><future z='007'/></root></mxGraphModel></diagram></mxfile>`

func newMultiDrawio(t *testing.T, raw string, width, height int) (*view, *ui.Tester, *text.Buffer, *int) {
	t.Helper()
	buffer := text.New(raw)
	changed := new(int)
	v := DrawioAdapter().Create(editor.Host{Buffer: buffer, Changed: func() { *changed++ }}).(*view)
	tester := ui.NewTester(v.Draw, width, height)
	if v.syncFailed || v.currentPage() == nil {
		t.Fatal("fixture parse", v.Warnings())
	}
	return v, tester, buffer, changed
}

func clickMultiNode(t *testing.T, tester *ui.Tester, v *view, id string, mods ui.Modifiers) {
	t.Helper()
	node := findDrawioNode(v.currentPage().Model.Nodes, id)
	if node == nil {
		t.Fatal("missing vertex", id)
	}
	x, y := nodeCenterOnWindow(v, *node)
	tester.Advance(time.Second)
	tester.ClickAtWith(mods, x, y)
}

func selectMultiNodes(t *testing.T, tester *ui.Tester, v *view, ids ...string) {
	t.Helper()
	for i, id := range ids {
		mods := ui.Modifiers(0)
		if i > 0 {
			mods = ui.Cmd
		}
		clickMultiNode(t, tester, v, id, mods)
	}
	if !slices.Equal(v.drawSelection.ids, ids) {
		t.Fatalf("selection=%v want %v", v.drawSelection.ids, ids)
	}
}

func parseMultiDrawio(t *testing.T, b *text.Buffer) *drawio.Document {
	t.Helper()
	d, _, err := drawio.Parse([]byte(b.String()), drawio.ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestDrawioAlignmentSixActionsPreserveSourceSelectionAndHistory(t *testing.T) {
	tests := []struct {
		label string
		x, y  [3]float64
	}{
		{"左对齐", [3]float64{20.5, 20.5, 20.5}, [3]float64{30.25, 100.5, 200.75}},
		{"水平居中", [3]float64{155.625, 135.625, 165.625}, [3]float64{30.25, 100.5, 200.75}},
		{"右对齐", [3]float64{290.75, 250.75, 310.75}, [3]float64{30.25, 100.5, 200.75}},
		{"顶对齐", [3]float64{20.5, 160.25, 310.75}, [3]float64{30.25, 30.25, 30.25}},
		{"垂直居中", [3]float64{20.5, 160.25, 310.75}, [3]float64{145.5, 135.5, 115.5}},
		{"底对齐", [3]float64{20.5, 160.25, 310.75}, [3]float64{260.75, 240.75, 200.75}},
	}
	for _, tc := range tests {
		for _, reverse := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/reverse=%v", tc.label, reverse), func(t *testing.T) {
				v, tester, buffer, changed := newMultiDrawio(t, multiDrawio, 1180, 780)
				selectMultiNodes(t, tester, v, "a", "b", "c")
				selection := text.Range{Anchor: 8, Head: 52}
				if reverse {
					selection.Anchor, selection.Head = selection.Head, selection.Anchor
				}
				if err := buffer.SetSelection(selection.Anchor, selection.Head); err != nil {
					t.Fatal(err)
				}
				before := buffer.String()
				version := buffer.Version()
				if err := tester.Click(tc.label); err != nil {
					t.Fatal(err)
				}
				if buffer.Version() != version+1 || *changed != 1 || buffer.Selection() != selection {
					t.Fatal("transaction/caret", buffer.Version(), *changed, buffer.Selection())
				}
				parsed := parseMultiDrawio(t, buffer)
				for i, id := range []string{"a", "b", "c"} {
					node := findDrawioNode(parsed.Pages[0].Model.Nodes, id)
					if node.X != tc.x[i] || node.Y != tc.y[i] {
						t.Fatalf("%s position=%g,%g want %g,%g", id, node.X, node.Y, tc.x[i], tc.y[i])
					}
				}
				for _, literal := range []string{`width="80.0" height="40.00"`, `width="120.00" height="60"`, `width="60" height="100.0"`, `<future z='007'/>`, `<mxPoint x="135.5" y="80.25"/>`} {
					if !strings.Contains(buffer.String(), literal) {
						t.Fatal("untouched XML lost", literal)
					}
				}
				after := buffer.String()
				tester.Key(ui.Cmd, ui.KeyZ)
				if buffer.String() != before || buffer.Selection() != selection || *changed != 2 {
					t.Fatal("undo did not restore exact source/caret", *changed, buffer.Selection())
				}
				tester.Key(ui.Cmd, ui.KeyY)
				if buffer.String() != after || buffer.Selection() != selection || *changed != 3 {
					t.Fatal("redo did not restore exact source/caret")
				}
				if !slices.Equal(v.drawSelection.ids, []string{"a", "b", "c"}) {
					t.Fatal("selection lost across geometry history", v.drawSelection.ids)
				}
			})
		}
	}
}

func TestDrawioCtrlSelectionAndSingleActionGuards(t *testing.T) {
	v, tester, buffer, changed := newMultiDrawio(t, multiDrawio, 1180, 780)
	selectMultiNodes(t, tester, v, "a", "b", "c")
	clickMultiNode(t, tester, v, "b", ui.Cmd)
	if !slices.Equal(v.drawSelection.ids, []string{"a", "c"}) || v.selectedID != "c" {
		t.Fatal("toggle off changed primary/order", v.drawSelection)
	}
	clickMultiNode(t, tester, v, "c", ui.Cmd)
	if !slices.Equal(v.drawSelection.ids, []string{"a"}) || v.selectedID != "a" {
		t.Fatal("primary fallback", v.drawSelection)
	}
	clickMultiNode(t, tester, v, "b", ui.Cmd)
	if _, ok := tester.Find("节点文本"); ok {
		t.Fatal("multi selection exposes primary label input")
	}
	a, ok := tester.AccessByRole(ui.RoleButton, "连接两个节点")
	if !ok || !a.Disabled {
		t.Fatal("multi connector enabled", a)
	}
	version := buffer.Version()
	tester.Key(0, ui.KeyDelete)
	if len(parseMultiDrawio(t, buffer).Pages[0].Model.Nodes) != 1 || parseMultiDrawio(t, buffer).Pages[0].Model.Nodes[0].ID != "c" || buffer.Version() != version+1 || *changed != 1 {
		t.Fatal("multi delete failed to remove the selected set as one transaction")
	}
	tester.Key(ui.Cmd, ui.KeyZ)
	selectMultiNodes(t, tester, v, "a", "b")
	tester.Command("delete")
	page := parseMultiDrawio(t, buffer).Pages[0]
	if len(page.Model.Nodes) != 1 || page.Model.Nodes[0].ID != "c" || *changed != 3 {
		t.Fatalf("delete command did not follow the same multi-selection route: nodes=%#v changed=%d", page.Model.Nodes, *changed)
	}
	tester.Key(ui.Cmd, ui.KeyZ)
	selectMultiNodes(t, tester, v, "a", "b")
	before := buffer.String()
	changedBefore := *changed
	clickMultiNode(t, tester, v, "a", 0)
	if !slices.Equal(v.drawSelection.ids, []string{"a", "b"}) || v.selectedID != "a" {
		t.Fatal("plain selected click discarded group")
	}
	tester.Key(0, ui.KeyEscape)
	if len(v.drawSelection.ids) != 0 || v.selectedID != "" {
		t.Fatal("Escape did not clear selection")
	}
	clickMultiNode(t, tester, v, "a", 0)
	clickMultiNode(t, tester, v, "a", ui.Cmd)
	if len(v.drawSelection.ids) != 0 || v.selectedID != "" {
		t.Fatal("toggle final member did not clear")
	}
	if buffer.String() != before || *changed != changedBefore {
		t.Fatal("selection dirtied source")
	}
}

func TestDrawioAlignmentNoOpPreservesRedo(t *testing.T) {
	v, tester, buffer, changed := newMultiDrawio(t, multiDrawio, 1180, 780)
	selectMultiNodes(t, tester, v, "a", "b", "c")
	if err := tester.Click("左对齐"); err != nil {
		t.Fatal(err)
	}
	aligned := buffer.String()
	if err := buffer.ReplaceUTF16(buffer.LenUTF16(), buffer.LenUTF16(), " "); err != nil {
		t.Fatal(err)
	}
	if !buffer.Undo() {
		t.Fatal("setup undo")
	}
	tester.Frame()
	history := buffer.History()
	version := buffer.Version()
	caret := buffer.Selection()
	if err := tester.Click("左对齐"); err != nil {
		t.Fatal(err)
	}
	if buffer.String() != aligned || buffer.Version() != version || buffer.History() != history || buffer.Selection() != caret || !buffer.CanRedo() || *changed != 1 {
		t.Fatal("no-op discarded history or wrote")
	}
}

func TestDrawioAlignmentHeldPressInvalidation(t *testing.T) {
	for _, mode := range []string{"source", "caret", "selection", "buffer", "release", "page-round-trip", "field"} {
		t.Run(mode, func(t *testing.T) {
			v, tester, buffer, changed := newMultiDrawio(t, multiDrawio, 1180, 780)
			selectMultiNodes(t, tester, v, "a", "b", "c")
			rect, ok := tester.Find("水平居中")
			if !ok {
				t.Fatal("alignment missing")
			}
			tester.Press(rect.X+rect.W/2, rect.Y+rect.H/2)
			switch mode {
			case "source":
				if err := buffer.ReplaceUTF16(buffer.LenUTF16(), buffer.LenUTF16(), " "); err != nil {
					t.Fatal(err)
				}
			case "caret":
				if err := buffer.SetSelection(2, 4); err != nil {
					t.Fatal(err)
				}
			case "selection":
				v.selectNode("a")
				v.selectNode("c")
			case "buffer":
				v.host.Buffer = text.New(multiDrawio)
			case "release":
				v.Release()
			case "page-round-trip":
				v.setPageIndex(1)
				v.setPageIndex(0)
			case "field":
				v.field = fieldNote
			}
			source, history, version := buffer.String(), buffer.History(), buffer.Version()
			tester.Frame()
			tester.Release(rect.X+rect.W/2, rect.Y+rect.H/2)
			if buffer.String() != source || buffer.History() != history || buffer.Version() != version || *changed != 0 {
				t.Fatal("stale alignment wrote", mode)
			}
		})
	}
}

func TestDrawioAlignmentIneligibleAndReadonlyAreAtomic(t *testing.T) {
	for _, mode := range []string{"readonly", "relative", "zero-width", "group", "capacity"} {
		t.Run(mode, func(t *testing.T) {
			raw := multiDrawio
			if mode == "relative" {
				raw = strings.Replace(raw, `x="160.25"`, `relative="1" x="160.25"`, 1)
			}
			if mode == "zero-width" {
				raw = strings.Replace(raw, `width="120.00"`, `width="0"`, 1)
			}
			v, tester, buffer, changed := newMultiDrawio(t, raw, 1180, 780)
			if mode == "capacity" {
				v.host.Buffer = text.NewWithLimit(raw, len(raw))
				buffer = v.host.Buffer
				tester.Frame()
			}
			selectMultiNodes(t, tester, v, "a", "b", "c")
			if mode == "readonly" {
				v.host.ReadOnly = true
			}
			if mode == "group" {
				if err := buffer.BeginGroup(); err != nil {
					t.Fatal(err)
				}
				defer buffer.CancelGroup()
			}
			tester.Frame()
			source, history := buffer.String(), buffer.History()
			// A disabled button rejects Click; capacity fails after full planning.
			_ = tester.Click("水平居中")
			if buffer.String() != source || buffer.History() != history || *changed != 0 {
				t.Fatal("ineligible alignment partially wrote", mode)
			}
			if mode != "capacity" {
				state, ok := tester.AccessByRole(ui.RoleButton, "水平居中")
				if !ok || !state.Disabled {
					t.Fatal("reason not reflected on button", mode, state)
				}
			}
		})
	}
}

func TestDrawioAlignmentChangedReentryKeepsReplacement(t *testing.T) {
	for _, mode := range []string{"source", "caret", "release", "buffer", "readonly"} {
		t.Run(mode, func(t *testing.T) {
			v, tester, buffer, changed := newMultiDrawio(t, multiDrawio, 1180, 780)
			selectMultiNodes(t, tester, v, "a", "b", "c")
			replacement := text.New(multiDrawio)
			v.host.Changed = func() {
				*changed++
				switch mode {
				case "source":
					if err := buffer.ReplaceUTF16(buffer.LenUTF16(), buffer.LenUTF16(), "<!--callback-->"); err != nil {
						t.Fatal(err)
					}
				case "caret":
					if err := buffer.SetSelection(1, 5); err != nil {
						t.Fatal(err)
					}
				case "release":
					v.Release()
				case "buffer":
					v.host.Buffer = replacement
				case "readonly":
					v.host.ReadOnly = true
				}
			}
			if err := tester.Click("左对齐"); err != nil {
				t.Fatal(err)
			}
			if *changed != 1 {
				t.Fatal("reentry duplicated commit", *changed)
			}
			if mode == "source" && !strings.HasSuffix(buffer.String(), "<!--callback-->") {
				t.Fatal("callback source overwritten")
			}
			if mode == "caret" && buffer.Selection() != (text.Range{Anchor: 1, Head: 5}) {
				t.Fatal("callback caret overwritten")
			}
			if mode == "buffer" && (v.host.Buffer != replacement || replacement.String() != multiDrawio || replacement.CanUndo()) {
				t.Fatal("replacement owner overwritten")
			}
		})
	}
}

func TestDrawioMultiselectNativeCaptures(t *testing.T) {
	for _, size := range [][2]int{{760, 480}, {1180, 780}} {
		for _, dark := range []bool{false, true} {
			t.Run(fmt.Sprintf("%dx%d-dark-%v", size[0], size[1], dark), func(t *testing.T) {
				v, tester, buffer, changed := newMultiDrawio(t, multiDrawio, size[0], size[1])
				tester.SetDark(dark)
				selectMultiNodes(t, tester, v, "a", "b", "c")
				var previous ui.Rect
				for i, spec := range drawioAlignmentButtons {
					rect, ok := tester.Find(spec.label)
					if !ok || rect.W < 45 || rect.H < 29.9 {
						t.Fatal("clipped alignment", spec.label, rect)
					}
					state, ok := tester.AccessByRole(ui.RoleButton, spec.label)
					if !ok || state.Disabled {
						t.Fatal("inaccessible alignment", state)
					}
					if i == 1 || i == 2 || i == 4 || i == 5 {
						if rect.Y != previous.Y {
							t.Fatal("row wrapping", rect, previous)
						}
					}
					if i == 3 && rect.Y <= previous.Y {
						t.Fatal("expected two alignment rows")
					}
					previous = rect
				}
				if !tester.HasText("已选择 3 个图形") {
					t.Fatal("selection count absent")
				}
				if buffer.String() != multiDrawio || *changed != 0 {
					t.Fatal("selection capture wrote source")
				}
				tester.Advance(time.Second)
				if out := os.Getenv("LEVNOTE_DRAWIO_MULTI_SCREENSHOT_DIR"); out != "" {
					if err := os.MkdirAll(out, 0755); err != nil {
						t.Fatal(err)
					}
					f, err := os.Create(filepath.Join(out, fmt.Sprintf("drawio-multiselect-%dx%d-dark-%v.png", size[0], size[1], dark)))
					if err != nil {
						t.Fatal(err)
					}
					encodeErr, closeErr := png.Encode(f, tester.Image()), f.Close()
					if encodeErr != nil || closeErr != nil {
						t.Fatal(encodeErr, closeErr)
					}
				}
			})
		}
	}
}
