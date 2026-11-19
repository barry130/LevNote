package graphview

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/enginescene"
)

func TestNativeSceneStylePaintSelectionAndBaseline(t *testing.T) {
	for _, dark := range []bool{false, true} {
		for _, styled := range []bool{false, true} {
			t.Run(fmt.Sprintf("dark=%v-styled=%v", dark, styled), func(t *testing.T) {
				v, requests, _ := nativeSceneFixture(t)
				defer v.Release()
				before, version, history, selection := v.host.Buffer.String(), v.host.Buffer.Version(), v.host.Buffer.History(), v.host.Buffer.Selection()
				_ = v.Warnings()
				result := literalNativeScene()
				if styled {
					// Authored neutral DTO: the worker contract independently proves
					// this fallback text is emitted for the same opaque pale fill.
					result.Scene.Nodes[0].Fill = "#fef3c7ff"
					result.Scene.Nodes[0].Stroke = "#166534ff"
					result.Scene.Nodes[0].Text = "#17211eff"
					result.Scene.Nodes[0].Label = "Readable text"
				}
				(*requests)[0].complete(result, nil)
				tester := ui.NewTester(v.Draw, 760, 480)
				tester.SetDark(dark)
				tester.Frame()
				node := result.Scene.Nodes[0]
				point := v.point(enginescene.Point{X: node.X, Y: node.Y}, v.canvasRect)
				height := float32(node.Height) * v.zoom
				greenBefore, ringBefore := sceneStyleBorderPixels(tester, point, height)
				if ringBefore != 0 || (styled && greenBefore < 4) {
					t.Fatal("unselected node border", greenBefore, ringBefore)
				}
				tester.ClickAt(point[0]+float32(node.Width)*v.zoom/2, point[1]+height/2)
				tester.Frame()
				if v.selectedID != node.ID {
					t.Fatal("native pointer failed to select node")
				}
				greenAfter, ringAfter := sceneStyleBorderPixels(tester, point, height)
				if styled && (greenAfter < 4 || ringAfter < 4) {
					t.Fatal("selected document border and outer ring must coexist", greenAfter, ringAfter)
				}
				if !styled && ringAfter != 0 {
					t.Fatal("unstyled selection changed its old single-border baseline", ringAfter)
				}
				theme := ui.LightTheme()
				if dark {
					theme = ui.DarkTheme()
				}
				fill, stroke, ink := sceneNodeColors(node, node.ID, theme)
				if styled {
					if fill != ui.Hex("#fef3c7ff") || stroke != ui.Hex("#166534ff") || ink != ui.Hex("#17211eff") {
						t.Fatal("neutral scene literals changed with theme or selection")
					}
					pixels, inkPixels := tester.Image(), 0
					for y := int(point[1]) + 8; y < int(point[1]+height)-8; y++ {
						for x := int(point[0]) + 12; x < int(point[0]+float32(node.Width)*v.zoom)-12; x++ {
							p := pixels.RGBAAt(x, y)
							if p.R < 70 && p.G < 70 && p.B < 70 {
								inkPixels++
							}
						}
					}
					if inkPixels < 4 {
						t.Fatal("pale readonly fill lost its dark fallback text", inkPixels)
					}
					if out := os.Getenv("LEVNOTE_GRAPH_STYLE_SCREENSHOT_DIR"); out != "" {
						if err := os.MkdirAll(out, 0755); err != nil {
							t.Fatal(err)
						}
						f, err := os.Create(filepath.Join(out, fmt.Sprintf("node-colors-dark-%v-760-readonly.png", dark)))
						if err != nil {
							t.Fatal(err)
						}
						encodeErr, closeErr := png.Encode(f, tester.Image()), f.Close()
						if encodeErr != nil || closeErr != nil {
							t.Fatal(encodeErr, closeErr)
						}
					}
				} else if fill != theme.Accent.Alpha(0.16) || stroke != theme.Accent || ink != theme.Text {
					t.Fatal("unstyled root theme baseline changed")
				}
				plain := enginescene.Node{ID: "plain"}
				fill, stroke, ink = sceneNodeColors(plain, "", theme)
				if fill != theme.Surface || stroke != theme.Border || ink != theme.Text {
					t.Fatal("unstyled non-root baseline changed")
				}
				_, stroke, _ = sceneNodeColors(plain, plain.ID, theme)
				if stroke != theme.Accent {
					t.Fatal("unstyled selection baseline changed")
				}
				tester.Advance(time.Second)
				if len(*requests) != 1 || v.host.Buffer.String() != before || v.host.Buffer.Version() != version || v.host.Buffer.History() != history || v.host.Buffer.Selection() != selection {
					t.Fatal("readonly colors/selection rewrote source or requested again")
				}
			})
		}
	}
}

func sceneStyleBorderPixels(tester *ui.Tester, point [2]float32, height float32) (green, ring int) {
	pixels := tester.Image()
	for y := int(point[1] + height*0.35); y < int(point[1]+height*0.65); y++ {
		for x := int(point[0]) - 6; x <= int(point[0])+3; x++ {
			p := pixels.RGBAAt(x, y)
			if int(p.G) > int(p.R)+20 && int(p.G) > int(p.B)+20 {
				green++
			}
			if float32(x) < point[0]-2 && int(p.B) > int(p.R)+20 && int(p.B) > int(p.G)+20 {
				ring++
			}
		}
	}
	return green, ring
}
