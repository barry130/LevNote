package graphview_test

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/modules/graphview"
	"levnote.local/native/ui/editor"
)

func TestDrawioCanvasDeleteUndoRedoPreservesSource(t *testing.T) {
	original := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\r\n" + `<mxfile host="keep"><diagram id="p" name="Delete"><mxGraphModel custom="keep"><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="a" value="remove" custom="remove-meta" vertex="1" parent="1"><mxGeometry x="0" y="0" width="80" height="80" as="geometry"/></mxCell><!-- keep --></root></mxGraphModel></diagram></mxfile>`
	want := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\r\n" + `<mxfile host="keep"><diagram id="p" name="Delete"><mxGraphModel custom="keep"><root><mxCell id="0"/><mxCell id="1" parent="0"/><!-- keep --></root></mxGraphModel></diagram></mxfile>`
	buffer := text.New(original)
	changed := 0
	var reported []error
	view := graphview.DrawioAdapter().Create(editor.Host{
		Buffer:  buffer,
		Changed: func() { changed++ },
		Report:  func(err error) { reported = append(reported, err) },
	})
	t.Cleanup(view.Release)
	tester := ui.NewTester(view.Draw, 960, 640)
	canvas, ok := tester.Find("draw.io 画布：Delete")
	if !ok || canvas.W <= 0 || canvas.H <= 0 {
		t.Fatalf("public draw.io canvas not rendered: %v", tester.Texts())
	}
	// A single 80x80 square auto-fits at the visible canvas center. No private
	// selection, geometry, pan/zoom or handler injection establishes the target.
	tester.ClickAt(canvas.X+canvas.W/2, canvas.Y+canvas.H/2)
	if buffer.String() != original || changed != 0 || buffer.CanUndo() || buffer.InGroup() {
		t.Fatal("drawing or the genuine canvas click changed source/history")
	}
	selection := buffer.Selection()
	tester.Key(0, ui.KeyDelete)
	if buffer.String() != want || changed != 1 || !buffer.CanUndo() || buffer.InGroup() {
		t.Fatalf("canvas Delete did not commit one exact source transaction: changed=%d source=%q reports=%v", changed, buffer.String(), reported)
	}
	if len(reported) != 0 {
		t.Fatalf("ordinary supported deletion reported failure: %v", reported)
	}
	tester.Key(ui.Cmd, ui.KeyZ)
	if buffer.String() != original || buffer.Selection() != selection || changed != 2 || buffer.CanUndo() || !buffer.CanRedo() || buffer.InGroup() {
		t.Fatalf("single native Undo did not restore exact bytes/selection/history: changed=%d source=%q", changed, buffer.String())
	}
	tester.Key(ui.Cmd|ui.Shift, ui.KeyZ)
	if buffer.String() != want || changed != 3 || !buffer.CanUndo() || buffer.CanRedo() || buffer.InGroup() {
		t.Fatalf("native Redo was not stable: changed=%d source=%q", changed, buffer.String())
	}
}

func TestDrawioCanvasEdgeDeletePreservesVerticesAndHistory(t *testing.T) {
	original := `<mxfile host="keep"><diagram id="p" name="Edge"><mxGraphModel custom="keep"><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="a" value="A" vertex="1" parent="1"><mxGeometry x="0" y="0" width="80" height="80" as="geometry"/></mxCell><!-- keep gap --><mxCell id="b" value="B" custom="keep" vertex="1" parent="1"><mxGeometry x="240" y="0" width="80" height="80" as="geometry"/></mxCell><mxCell id="edge" edge="1" source="a" target="b" parent="1"><mxGeometry relative="1" as="geometry"/></mxCell></root></mxGraphModel></diagram></mxfile>`
	want := `<mxfile host="keep"><diagram id="p" name="Edge"><mxGraphModel custom="keep"><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="a" value="A" vertex="1" parent="1"><mxGeometry x="0" y="0" width="80" height="80" as="geometry"/></mxCell><!-- keep gap --><mxCell id="b" value="B" custom="keep" vertex="1" parent="1"><mxGeometry x="240" y="0" width="80" height="80" as="geometry"/></mxCell></root></mxGraphModel></diagram></mxfile>`
	buffer := text.New(original)
	changed := 0
	var reported []error
	view := graphview.DrawioAdapter().Create(editor.Host{
		Buffer:  buffer,
		Changed: func() { changed++ },
		Report:  func(err error) { reported = append(reported, err) },
	})
	t.Cleanup(view.Release)
	tester := ui.NewTester(view.Draw, 960, 640)
	canvas, ok := tester.Find("draw.io 画布：Edge")
	if !ok || canvas.W <= 0 || canvas.H <= 0 {
		t.Fatalf("public edge canvas not rendered: %v", tester.Texts())
	}
	// The symmetric 320x80 diagram's center is the visible segment between
	// two vertices, not either vertex. Only a genuine public pointer targets it.
	tester.ClickAt(canvas.X+canvas.W/2, canvas.Y+canvas.H/2)
	if buffer.String() != original || changed != 0 || buffer.CanUndo() || buffer.InGroup() {
		t.Fatal("edge targeting or its pointer release changed source/history")
	}
	selection := buffer.Selection()
	tester.Key(0, ui.KeyDelete)
	if buffer.String() != want || changed != 1 || !buffer.CanUndo() || buffer.InGroup() || len(reported) != 0 {
		t.Fatalf("public edge Delete did not preserve both vertices exactly: changed=%d source=%q reports=%v", changed, buffer.String(), reported)
	}
	tester.Key(ui.Cmd, ui.KeyZ)
	if buffer.String() != original || buffer.Selection() != selection || changed != 2 || buffer.CanUndo() || !buffer.CanRedo() || buffer.InGroup() {
		t.Fatalf("one native Undo did not restore the edge and caret exactly: changed=%d source=%q", changed, buffer.String())
	}
	tester.Key(ui.Cmd|ui.Shift, ui.KeyZ)
	if buffer.String() != want || changed != 3 || !buffer.CanUndo() || buffer.CanRedo() || buffer.InGroup() {
		t.Fatalf("native edge Redo was not stable: changed=%d source=%q", changed, buffer.String())
	}
}

func TestDrawioConnectorButtonRequiresVertexSource(t *testing.T) {
	original := `<mxfile host="keep"><diagram id="p" name="Connector source"><mxGraphModel custom="keep"><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="a" value="A" vertex="1" parent="1"><mxGeometry x="0" y="0" width="80" height="80" as="geometry"/></mxCell><!-- keep gap --><mxCell id="b" value="B" custom="keep" vertex="1" parent="1"><mxGeometry x="240" y="0" width="80" height="80" as="geometry"/></mxCell><mxCell id="edge" edge="1" source="a" target="b" parent="1"><mxGeometry relative="1" as="geometry"/></mxCell></root></mxGraphModel></diagram></mxfile>`
	buffer := text.New(original)
	if err := buffer.SetSelection(7, 23); err != nil {
		t.Fatal(err)
	}
	version, selection, history := buffer.Version(), buffer.Selection(), buffer.History()
	changed := 0
	var reported []error
	view := graphview.DrawioAdapter().Create(editor.Host{
		Buffer:  buffer,
		Changed: func() { changed++ },
		Report:  func(err error) { reported = append(reported, err) },
	})
	t.Cleanup(view.Release)
	tester := ui.NewTester(view.Draw, 960, 640)
	unchanged := func(step string) {
		t.Helper()
		if buffer.String() != original || buffer.Version() != version || buffer.Selection() != selection ||
			buffer.History() != history || buffer.InGroup() || changed != 0 || len(reported) != 0 {
			t.Fatalf("%s changed source, caret, history or reported failure: changed=%d source=%q reports=%v", step, changed, buffer.String(), reported)
		}
	}
	canvas, ok := tester.Find("draw.io 画布：Connector source")
	if !ok || canvas.W <= 0 || canvas.H <= 0 {
		t.Fatalf("public connector canvas not rendered: %v", tester.Texts())
	}
	tester.ClickAt(canvas.X+canvas.W/2, canvas.Y+canvas.H/2)
	if _, visible := tester.Find("edge"); !visible {
		t.Fatalf("genuine segment click did not show the selected edge ID: %v", tester.Texts())
	}
	unchanged("edge selection")
	buttonRect, ok := tester.Find("连接两个节点")
	if !ok || buttonRect.W <= 0 || buttonRect.H <= 0 {
		t.Fatalf("public connector button not rendered: %v", tester.Texts())
	}
	// A real pointer is delivered even when the button is correctly disabled.
	tester.ClickAt(buttonRect.X+buttonRect.W/2, buttonRect.Y+buttonRect.H/2)
	unchanged("edge source button")
	button, visible := tester.Access("连接两个节点")
	if !visible || !button.Disabled || tester.HasText("选终点") || tester.HasText("连接模式：点击目标节点") {
		t.Fatalf("edge was offered as a connector source: visible=%v disabled=%v texts=%v", visible, button.Disabled, tester.Texts())
	}
	canvas, ok = tester.Find("draw.io 画布：Connector source")
	if !ok || canvas.W <= 0 || canvas.H <= 0 {
		t.Fatal("connector canvas disappeared after refusing an edge source")
	}
	// In this fitted symmetric diagram the left eighth lies inside vertex a.
	tester.ClickAt(canvas.X+canvas.W/8, canvas.Y+canvas.H/2)
	if _, visible := tester.Find("a"); !visible {
		t.Fatalf("genuine vertex click did not show vertex a: %v", tester.Texts())
	}
	unchanged("vertex selection")
	button, visible = tester.Access("连接两个节点")
	if !visible || button.Disabled {
		t.Fatalf("supported vertex source was disabled: visible=%v disabled=%v", visible, button.Disabled)
	}
	if err := tester.Click("连接两个节点"); err != nil {
		t.Fatal(err)
	}
	if !tester.HasText("选终点") || !tester.HasText("连接模式：点击目标节点") {
		t.Fatalf("supported vertex source did not enter connector mode: %v", tester.Texts())
	}
	unchanged("supported connector source button")
}

func TestDrawioUndoCallbackReleaseStopsObsoleteOwner(t *testing.T) {
	original := `<mxfile host="old-owner"><diagram id="old" name="Undo owner"><mxGraphModel custom="keep"><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="a" value="remove" custom="remove-meta" vertex="1" parent="1"><mxGeometry x="0" y="0" width="80" height="80" as="geometry"/></mxCell><!-- keep old metadata --></root></mxGraphModel></diagram></mxfile>`
	deleted := `<mxfile host="old-owner"><diagram id="old" name="Undo owner"><mxGraphModel custom="keep"><root><mxCell id="0"/><mxCell id="1" parent="0"/><!-- keep old metadata --></root></mxGraphModel></diagram></mxfile>`
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
	var callbackVersion uint64
	var callbackSelection text.Range
	var callbackHistory text.HistoryStatus
	oldView = graphview.DrawioAdapter().Create(editor.Host{
		Buffer: oldBuffer,
		Changed: func() {
			oldChanged++
			if oldChanged != 2 {
				return
			}
			if oldBuffer.String() != original || oldBuffer.Selection() != oldSelection || oldBuffer.CanUndo() ||
				!oldBuffer.CanRedo() || oldBuffer.InGroup() {
				t.Fatalf("Undo callback did not receive the restored old document: source=%q selection=%v history=%+v", oldBuffer.String(), oldBuffer.Selection(), oldBuffer.History())
			}
			// The host flushes and releases its public View before replacing the
			// actually mounted owner. No external Host variable is reassigned.
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
			// This lawful host edit belongs to the retired Buffer, not to the
			// released view. Its deliberately incomplete XML must not be parsed
			// or reported by the obsolete Undo continuation after this callback.
			if err := oldBuffer.ReplaceUTF16(0, text.UTF16Len(original), retired); err != nil {
				t.Fatal(err)
			}
			if err := oldBuffer.SetSelection(3, 13); err != nil {
				t.Fatal(err)
			}
			callbackVersion, callbackSelection, callbackHistory = oldBuffer.Version(), oldBuffer.Selection(), oldBuffer.History()
			callbackCaptured = true
		},
		Report: func(err error) {
			// Release itself may defensively flush. Only reports after Release
			// returned and the actual mounted owner switched are obsolete.
			if switched {
				obsoleteReports = append(obsoleteReports, err)
				return
			}
			beforeSwitchReports = append(beforeSwitchReports, err)
		},
	})
	activeView = oldView
	t.Cleanup(func() { activeView.Release() })
	tester := ui.NewTester(func(c *ui.Context) { activeView.Draw(c) }, 960, 640)
	canvas, ok := tester.Find("draw.io 画布：Undo owner")
	if !ok || canvas.W <= 0 || canvas.H <= 0 {
		t.Fatalf("old public draw.io canvas not rendered: %v", tester.Texts())
	}
	tester.ClickAt(canvas.X+canvas.W/2, canvas.Y+canvas.H/2)
	if _, visible := tester.Find("a"); !visible {
		t.Fatalf("genuine canvas click did not show the selected vertex: %v", tester.Texts())
	}
	if oldBuffer.String() != original || oldBuffer.Version() != oldVersion || oldBuffer.Selection() != oldSelection ||
		oldBuffer.History() != oldHistory || oldBuffer.InGroup() || oldChanged != 0 || len(beforeSwitchReports) != 0 {
		t.Fatal("old view rendering or genuine vertex targeting changed source, caret, history or reported failure")
	}
	tester.Key(0, ui.KeyDelete)
	if oldBuffer.String() != deleted || oldChanged != 1 || !oldBuffer.CanUndo() || oldBuffer.CanRedo() ||
		oldBuffer.InGroup() || len(beforeSwitchReports) != 0 {
		t.Fatalf("supported Delete did not establish one real Undo transaction: changed=%d source=%q reports=%v", oldChanged, oldBuffer.String(), beforeSwitchReports)
	}
	// A single ordinary native Undo, outside any callback, reaches Changed #2.
	// There is no nested Tester.Key and the wrapper never draws a released view.
	tester.Key(ui.Cmd, ui.KeyZ)
	if !switched || !callbackCaptured || oldChanged != 2 {
		t.Fatalf("native Undo did not reach the legal host replacement: switched=%v captured=%v changed=%d", switched, callbackCaptured, oldChanged)
	}
	if oldBuffer.String() != retired || oldBuffer.Version() != callbackVersion || oldBuffer.Selection() != callbackSelection ||
		callbackSelection != (text.Range{Anchor: 3, Head: 13}) || oldBuffer.History() != callbackHistory || oldBuffer.InGroup() {
		t.Fatalf("obsolete Undo continuation changed the host's retired document receipt: source=%q selection=%v version=%d history=%+v", oldBuffer.String(), oldBuffer.Selection(), oldBuffer.Version(), oldBuffer.History())
	}
	if newBuffer.String() != replacement || newBuffer.Version() != newVersion || newBuffer.Selection() != newSelection ||
		newBuffer.History() != newHistory || newBuffer.InGroup() || newChanged != 0 || len(newReports) != 0 {
		t.Fatalf("old Undo affected the actual replacement owner: changed=%d source=%q reports=%v", newChanged, newBuffer.String(), newReports)
	}
	if _, visible := tester.Find("draw.io 画布：Replacement owner"); !visible || tester.HasText("draw.io 画布：Undo owner") {
		t.Fatalf("wrapper did not draw only the actual replacement view: %v", tester.Texts())
	}
	if len(obsoleteReports) != 0 {
		t.Fatalf("obsolete draw.io owner reported after Undo callback released and replaced it: %v", obsoleteReports)
	}
}
