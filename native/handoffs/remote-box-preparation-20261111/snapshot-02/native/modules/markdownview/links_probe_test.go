package markdownview

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// This isolates the public native seam before authored-link activation is
// introduced. No platform internals, OS URL opener or vendor patch is used.
func TestNativeInlineModifierHitProbe(t *testing.T) {
	for _, observed := range []bool{false, true} {
		t.Run(map[bool]string{false: "context-read-only", true: "observed-event-modifiers"}[observed], func(t *testing.T) {
			var held ui.Modifiers
			activations, doubles := 0, 0
			start, end := 0, 0
			tester := ui.NewTester(func(c *ui.Context) {
				held = c.Modifiers()
				row := ui.Column(c).Width(265).Height(200)
				if observed {
					row.ObserveInput(func(event ui.InputEvent) { held = event.Mods })
				}
				row.HandleInput(func(event ui.InputEvent) bool {
					if event.Kind == ui.InputPointerDown && event.Button == 0 && event.Clicks >= 2 {
						doubles++
						return true
					}
					return false
				})
				row.Children(func() {
					paragraph := ui.RichText(c).FontSize(16).Selectable()
					paragraph.Children(func() {
						ui.Text(c, "prefix ")
						link := ui.Text(c, "linked words").TextColor(c.Theme().Accent).Underline().Focusable().Role(ui.RoleLink).Label("probe-link")
						link.HitTest(func(x, y float32, r ui.Rect) bool {
							if observed {
								return held&ui.Cmd != 0
							}
							return c.Modifiers()&ui.Cmd != 0
						})
						// Owned handler isolates activation from the parent edit
						// handler; plain pointer does not hit this child.
						link.HandleInput(func(event ui.InputEvent) bool { return false })
						if link.Clicked() {
							activations++
						}
						ui.Text(c, " suffix")
					})
					start, end = paragraph.TextSelection()
				})
			}, 265, 200)
			bounds, ok := tester.Find("probe-link")
			if !ok {
				t.Fatal("probe inline missing")
			}
			x, y := bounds.X+bounds.W/2, bounds.Y+bounds.H/2
			tester.ClickAtWith(ui.Cmd, x, y)
			if !observed {
				if activations != 0 {
					t.Fatal("unexpected usable build Context outside frame")
				}
				t.Log("Context.Modifiers() inside native hit callback cannot see dispatch modifiers")
				return
			}
			if activations != 1 || doubles != 0 {
				t.Fatal("observed native modifiers failed Ctrl click", activations, doubles)
			}
			tester.Advance(time.Second)
			tester.Press(bounds.X+4, y)
			tester.Move(bounds.X+bounds.W-4, y)
			tester.Release(bounds.X+bounds.W-4, y)
			if start == end || activations != 1 {
				t.Fatal("plain drag starting inside link lost paragraph selection", start, end, activations)
			}
			tester.Command("copy")
			if tester.Clipboard() == "" {
				t.Fatal("selected link text was not copyable")
			}
			tester.Advance(time.Second)
			tester.ClickAt(x, y)
			tester.ClickAt(x, y)
			if doubles != 1 || activations != 1 {
				t.Fatal("plain doubleclick did not preserve parent editing", doubles, activations)
			}
			tester.Key(0, ui.KeyTab)
			info, _ := tester.AccessByRole(ui.RoleLink, "probe-link")
			if !info.Focused {
				t.Fatal("Tab did not focus native link", info)
			}
			tester.Key(0, ui.KeyEnter)
			tester.Key(0, ui.KeySpace)
			if activations != 2 {
				t.Fatal("Enter/Space native role behavior", activations)
			}
			t.Logf("public observed-modifier seam passed Ctrl/plain drag+copy/doubleclick/Tab+Enter; selection=%d..%d", start, end)
		})
	}
}

func TestNativeInlineWrappedHitAndCancellationProbe(t *testing.T) {
	var held ui.Modifiers
	activations := 0
	var duringPress func()
	tester := ui.NewTester(func(c *ui.Context) {
		held = c.Modifiers()
		ui.Column(c).Width(265).Height(240).
			ObserveInput(func(event ui.InputEvent) { held = event.Mods }).Children(func() {
			ui.RichText(c).FontSize(16).Selectable().Children(func() {
				ui.Text(c, "prefix ")
				link := ui.Text(c, "wrapped link words 中文🙂 across several lines with more words to reach the final line").
					Underline().Focusable().Role(ui.RoleLink).Label("wrapped-probe")
				link.HitTest(func(x, y float32, r ui.Rect) bool { return held&ui.Cmd != 0 })
				link.HandleInput(func(event ui.InputEvent) bool {
					if event.Kind == ui.InputPointerDown && event.Button == 0 && duringPress != nil {
						duringPress()
					}
					return false
				})
				if link.Clicked() {
					activations++
				}
				ui.Text(c, " suffix")
			})
		})
	}, 265, 240)
	first, ok := tester.Find("wrapped-probe")
	all, accessOK := tester.AccessByRole(ui.RoleLink, "wrapped-probe")
	if !ok || !accessOK || all.Bounds.H < first.H*2 {
		t.Fatal("fixture did not wrap", first, all)
	}
	tester.ClickAtWith(ui.Cmd, first.X+4, first.Y+first.H/2)
	tester.ClickAtWith(ui.Cmd, all.Bounds.X+4, all.Bounds.Y+all.Bounds.H-first.H/2)
	if activations != 2 {
		t.Fatal("first/last wrapped fragment was not native clickable", activations, first, all.Bounds)
	}
	// The enclosing accessibility rectangle contains non-link space on its
	// first line. Native glyph fragments must not make that empty area active.
	tester.ClickAtWith(ui.Cmd, all.Bounds.X+4, first.Y+first.H/2)
	if activations != 2 {
		t.Fatal("wrapped bounding-box gap activated link")
	}
	// Public nested events model a system cancellation/modifier release during
	// the press. The toolkit queues release until the down handler completes.
	duringPress = tester.CancelPress
	tester.ClickAtWith(ui.Cmd, first.X+4, first.Y+first.H/2)
	if activations != 2 {
		t.Fatal("canceled native press activated link", activations)
	}
	duringPress = func() { tester.Release(first.X+4, first.Y+first.H/2) }
	tester.ClickAtWith(ui.Cmd, first.X+4, first.Y+first.H/2)
	if activations != 2 {
		t.Fatal("modifier released before pointer up activated link", activations)
	}
	duringPress = nil
	tester.Press(first.X+4, first.Y+first.H/2)
	tester.HoldModifiers(ui.Cmd)
	// Tester.Release sends Mods=0. This sequence verifies a real release
	// after modifier gain; it does not model an OS release with Cmd held.
	tester.Release(first.X+4, first.Y+first.H/2)
	if activations != 2 {
		t.Fatal("plain press gained modifier and activated link")
	}
	t.Logf("wrapped first+last fragments activate, bounding gap/cancel/modifier-loss/plain-down+gain+release(Mods=0) do not; bounds=%+v", all.Bounds)
}
