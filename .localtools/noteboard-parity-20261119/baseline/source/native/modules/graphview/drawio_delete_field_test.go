package graphview_test

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/modules/graphview"
	"levnote.local/native/ui/editor"
)

func TestDrawioFieldDeletePrecedesCanvasDeletionAndPreservesTwoHistoryTransactions(t *testing.T) {
	original := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\r\n" + `<mxfile host="keep"><diagram id="field" name="Field Delete"><mxGraphModel custom="keep"><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="a" value="标签🙂" custom="keep-cell" vertex="1" parent="1"><mxGeometry x="0" y="0" width="80" height="80" as="geometry"/></mxCell><!-- 保留🙂 --></root></mxGraphModel></diagram></mxfile>`
	edited := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\r\n" + `<mxfile host="keep"><diagram id="field" name="Field Delete"><mxGraphModel custom="keep"><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="a" value="" custom="keep-cell" vertex="1" parent="1"><mxGeometry x="0" y="0" width="80" height="80" as="geometry"/></mxCell><!-- 保留🙂 --></root></mxGraphModel></diagram></mxfile>`
	deleted := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\r\n" + `<mxfile host="keep"><diagram id="field" name="Field Delete"><mxGraphModel custom="keep"><root><mxCell id="0"/><mxCell id="1" parent="0"/><!-- 保留🙂 --></root></mxGraphModel></diagram></mxfile>`
	buffer := text.New(original)
	if err := buffer.SetSelection(7, 23); err != nil {
		t.Fatal(err)
	}
	initialSelection := text.Range{Anchor: 7, Head: 23}
	changed := 0
	var reported []error
	view := graphview.DrawioAdapter().Create(editor.Host{
		Buffer:  buffer,
		Changed: func() { changed++ },
		Report:  func(err error) { reported = append(reported, err) },
	})
	t.Cleanup(view.Release)
	tester := ui.NewTester(view.Draw, 960, 640)
	canvas, ok := tester.Find("draw.io 画布：Field Delete")
	if !ok || canvas.W <= 0 || canvas.H <= 0 {
		t.Fatalf("public draw.io canvas not rendered: %v", tester.Texts())
	}
	// One visible square auto-fits at the public canvas center. Both target
	// selection and property focus are established by genuine pointer input.
	tester.ClickAt(canvas.X+canvas.W/2, canvas.Y+canvas.H/2)
	if err := tester.Click("节点文本"); err != nil {
		t.Fatal(err)
	}
	if active, noInputMethod := tester.TextInputActive(); !active || noInputMethod {
		t.Fatal("genuine property click did not focus the public native text input")
	}
	tester.Command("selectAll")
	if buffer.String() != original || buffer.Selection() != initialSelection || changed != 0 ||
		buffer.History().UndoDepth != 0 || buffer.History().RedoDepth != 0 || buffer.InGroup() || len(reported) != 0 {
		t.Fatalf("property focus/selectAll changed authoritative source, caret or history: source=%q selection=%+v history=%+v changed=%d reports=%v", buffer.String(), buffer.Selection(), buffer.History(), changed, reported)
	}
	tester.Key(0, ui.KeyDelete)
	fieldSelection := text.Range{Anchor: 367, Head: 367}
	if buffer.String() != edited || buffer.Selection() != fieldSelection || changed != 1 ||
		buffer.History().UndoDepth != 1 || buffer.History().RedoDepth != 0 || buffer.InGroup() || len(reported) != 0 {
		t.Fatalf("focused property Delete did not clear only the label in one source transaction: source=%q selection=%+v history=%+v changed=%d reports=%v", buffer.String(), buffer.Selection(), buffer.History(), changed, reported)
	}
	canvas, ok = tester.Find("draw.io 画布：Field Delete")
	if !ok || canvas.W <= 0 || canvas.H <= 0 {
		t.Fatalf("canvas vanished after ordinary field commit: %v", tester.Texts())
	}
	tester.ClickAt(canvas.X+canvas.W/2, canvas.Y+canvas.H/2)
	if buffer.String() != edited || buffer.Selection() != fieldSelection || changed != 1 ||
		buffer.History().UndoDepth != 1 || buffer.History().RedoDepth != 0 || buffer.InGroup() {
		t.Fatal("genuine return to canvas altered the field transaction")
	}
	tester.Key(0, ui.KeyDelete)
	deleteSelection := text.Range{Anchor: 231, Head: 231}
	if buffer.String() != deleted || buffer.Selection() != deleteSelection || changed != 2 ||
		buffer.History().UndoDepth != 2 || buffer.History().RedoDepth != 0 || buffer.InGroup() || len(reported) != 0 {
		t.Fatalf("canvas Delete did not create a separate second structural transaction: source=%q selection=%+v history=%+v changed=%d reports=%v", buffer.String(), buffer.Selection(), buffer.History(), changed, reported)
	}
	tester.Key(ui.Cmd, ui.KeyZ)
	if buffer.String() != edited || buffer.Selection() != fieldSelection || changed != 3 ||
		buffer.History().UndoDepth != 1 || buffer.History().RedoDepth != 1 || buffer.InGroup() || len(reported) != 0 {
		t.Fatalf("first native Undo did not restore the field-only literal and numerical UTF-16 caret: source=%q selection=%+v history=%+v changed=%d reports=%v", buffer.String(), buffer.Selection(), buffer.History(), changed, reported)
	}
	tester.Key(ui.Cmd, ui.KeyZ)
	if buffer.String() != original || buffer.Selection() != initialSelection || changed != 4 ||
		buffer.History().UndoDepth != 0 || buffer.History().RedoDepth != 2 || buffer.InGroup() || len(reported) != 0 {
		t.Fatalf("second native Undo did not restore the original literal and independent source caret: source=%q selection=%+v history=%+v changed=%d reports=%v", buffer.String(), buffer.Selection(), buffer.History(), changed, reported)
	}
}
