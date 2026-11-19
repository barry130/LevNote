package graphview_test

import (
	"crypto/sha256"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/modules/graphview"
	"levnote.local/native/ui/editor"
)

func TestDrawioRedoCallbackReleaseStopsObsoleteOwner(t *testing.T) {
	for _, nativeCommand := range []bool{false, true} {
		name := "CtrlY"
		if nativeCommand {
			name = "nativeRedo"
		}
		t.Run(name, func(t *testing.T) {
			original := `<mxfile host="old-owner"><diagram id="old" name="Redo owner"><mxGraphModel custom="keep"><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="a" value="remove" custom="remove-meta" vertex="1" parent="1"><mxGeometry x="0" y="0" width="80" height="80" as="geometry"/></mxCell><!-- keep old metadata --></root></mxGraphModel></diagram></mxfile>`
			deleted := `<mxfile host="old-owner"><diagram id="old" name="Redo owner"><mxGraphModel custom="keep"><root><mxCell id="0"/><mxCell id="1" parent="0"/><!-- keep old metadata --></root></mxGraphModel></diagram></mxfile>`
			retired := `<mxfile><diagram id="retired" name="Retired owner">`
			replacement := `<mxfile host="new-owner"><diagram id="new" name="Replacement owner"><mxGraphModel custom="new"><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="fresh" value="new owner" vertex="1" parent="1"><mxGeometry x="0" y="0" width="80" height="80" as="geometry"/></mxCell><!-- keep new metadata --></root></mxGraphModel></diagram></mxfile>`
			oldBuffer := text.New(original)
			if err := oldBuffer.SetSelection(7, 23); err != nil {
				t.Fatal(err)
			}
			oldVersion, oldSelection, oldHistory := oldBuffer.Version(), oldBuffer.Selection(), oldBuffer.History()
			newBuffer := text.New(replacement)
			if err := newBuffer.SetSelection(5, 19); err != nil {
				t.Fatal(err)
			}
			newVersion, newSelection, newHistory := newBuffer.Version(), newBuffer.Selection(), newBuffer.History()
			oldChanged, newChanged := 0, 0
			switched, callbackCaptured := false, false
			var beforeSwitchReports, obsoleteReports, newReports []error
			var oldView, activeView editor.View
			var replacementFocus ui.Handle
			var callbackVersion uint64
			var callbackSelection text.Range
			var callbackHistory text.HistoryStatus
			oldView = graphview.DrawioAdapter().Create(editor.Host{
				Buffer: oldBuffer,
				Changed: func() {
					oldChanged++
					if oldChanged != 3 {
						return
					}
					if oldBuffer.String() != deleted || oldBuffer.Selection() != oldSelection || !oldBuffer.CanUndo() ||
						oldBuffer.CanRedo() || oldBuffer.InGroup() {
						t.Fatalf("Redo callback did not receive the deleted old document: source=%q selection=%v history=%+v", oldBuffer.String(), oldBuffer.Selection(), oldBuffer.History())
					}
					// The real host flushes/releases its public View and mounts another
					// View. No nested input event or external Host reassignment is used.
					if err := editor.FlushView(oldView); err != nil {
						t.Fatal(err)
					}
					oldView.Release()
					activeView = graphview.DrawioAdapter().Create(editor.Host{
						Buffer:  newBuffer,
						Changed: func() { newChanged++ },
						Report:  func(err error) { newReports = append(newReports, err) },
					})
					switched = true
					replacementFocus.Focus()
					// The host can still edit its retired Buffer. The intentionally
					// broken XML detects obsolete synchronization via the old Report
					// callback, without drawing or querying the released View again.
					if err := oldBuffer.ReplaceUTF16(0, text.UTF16Len(deleted), retired); err != nil {
						t.Fatal(err)
					}
					if err := oldBuffer.SetSelection(3, 13); err != nil {
						t.Fatal(err)
					}
					callbackVersion, callbackSelection, callbackHistory = oldBuffer.Version(), oldBuffer.Selection(), oldBuffer.History()
					callbackCaptured = true
				},
				Report: func(err error) {
					if switched {
						obsoleteReports = append(obsoleteReports, err)
						return
					}
					beforeSwitchReports = append(beforeSwitchReports, err)
				},
			})
			activeView = oldView
			t.Cleanup(func() { activeView.Release() })
			tester := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().Children(func() {
					if switched {
						ui.Button(c, "替换视图焦点").Bind(&replacementFocus)
					}
					activeView.Draw(c)
				})
			}, 960, 640)
			canvas, ok := tester.Find("draw.io 画布：Redo owner")
			if !ok || canvas.W <= 0 || canvas.H <= 0 {
				t.Fatalf("old public draw.io canvas not rendered: %v", tester.Texts())
			}
			tester.ClickAt(canvas.X+canvas.W/2, canvas.Y+canvas.H/2)
			if _, visible := tester.Find("a"); !visible {
				t.Fatalf("genuine canvas click did not show the selected vertex: %v", tester.Texts())
			}
			if oldBuffer.String() != original || oldBuffer.Version() != oldVersion || oldBuffer.Selection() != oldSelection ||
				oldBuffer.History() != oldHistory || oldBuffer.InGroup() || oldChanged != 0 || len(beforeSwitchReports) != 0 {
				t.Fatal("old rendering or vertex targeting changed source, caret, history or reported failure")
			}
			tester.Key(0, ui.KeyDelete)
			if oldBuffer.String() != deleted || oldChanged != 1 || !oldBuffer.CanUndo() || oldBuffer.CanRedo() ||
				oldBuffer.InGroup() || len(beforeSwitchReports) != 0 {
				t.Fatalf("supported Delete did not establish one Undo transaction: changed=%d source=%q reports=%v", oldChanged, oldBuffer.String(), beforeSwitchReports)
			}
			// Public Undo restores both source and focus; no click or Tab between
			// Undo and Redo can conceal a failed continuous keyboard route.
			tester.Key(ui.Cmd, ui.KeyZ)
			if oldBuffer.String() != original || oldBuffer.Selection() != oldSelection || oldChanged != 2 ||
				oldBuffer.CanUndo() || !oldBuffer.CanRedo() || oldBuffer.InGroup() || len(beforeSwitchReports) != 0 {
				t.Fatalf("public Undo did not restore one redoable transaction: changed=%d source=%q history=%+v reports=%v", oldChanged, oldBuffer.String(), oldBuffer.History(), beforeSwitchReports)
			}
			if nativeCommand {
				tester.Command("redo")
			} else {
				tester.Key(ui.Cmd, ui.KeyY)
			}
			if !switched || !callbackCaptured || oldChanged != 3 {
				t.Fatalf("public Redo did not reach legal replacement: switched=%v captured=%v changed=%d", switched, callbackCaptured, oldChanged)
			}
			t.Logf("receipt oldChanged=%d newChanged=%d obsoleteReports=%d oldVersion=%d oldSelection=%v oldHistory=%+v newVersion=%d newSelection=%v newHistory=%+v oldSHA256=%x newSHA256=%x",
				oldChanged, newChanged, len(obsoleteReports), oldBuffer.Version(), oldBuffer.Selection(), oldBuffer.History(), newBuffer.Version(), newBuffer.Selection(), newBuffer.History(),
				sha256.Sum256([]byte(oldBuffer.String())), sha256.Sum256([]byte(newBuffer.String())))
			if oldBuffer.String() != retired || oldBuffer.Version() != callbackVersion || oldBuffer.Selection() != callbackSelection ||
				callbackSelection != (text.Range{Anchor: 3, Head: 13}) || oldBuffer.History() != callbackHistory || oldBuffer.InGroup() {
				t.Fatalf("obsolete Redo changed retired receipt: source=%q selection=%v version=%d history=%+v", oldBuffer.String(), oldBuffer.Selection(), oldBuffer.Version(), oldBuffer.History())
			}
			if newBuffer.String() != replacement || newBuffer.Version() != newVersion || newBuffer.Selection() != newSelection ||
				newBuffer.History() != newHistory || newBuffer.InGroup() || newChanged != 0 || len(newReports) != 0 {
				t.Fatalf("old Redo affected replacement owner: changed=%d source=%q reports=%v", newChanged, newBuffer.String(), newReports)
			}
			if _, visible := tester.Find("draw.io 画布：Replacement owner"); !visible || tester.HasText("draw.io 画布：Redo owner") {
				t.Fatalf("wrapper did not draw only replacement: %v", tester.Texts())
			}
			focus, ok := tester.AccessByRole(ui.RoleButton, "替换视图焦点")
			if !ok || !focus.Focused || tester.FramePending() {
				t.Fatalf("obsolete Redo stole replacement focus or left a queued frame: focus=%+v pending=%v", focus, tester.FramePending())
			}
			if len(obsoleteReports) != 0 {
				t.Fatalf("obsolete draw.io owner synchronized/reported retired XML after Redo released it: %v", obsoleteReports)
			}
		})
	}
}
