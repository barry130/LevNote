package app

import (
	"bytes"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/formats/drawio"
)

func TestDrawioMultiDeleteAppExactDiskUndoAndReopen(t *testing.T) {
	a, tab, tester, path := openDrawioAlignmentApp(t, drawioAlignmentAppOptions{})
	before := tab.Doc.Buffer.String()
	baselineDisk := mustReadFile(t, path)
	if err := tab.Doc.Buffer.SetSelection(18, 31); err != nil {
		t.Fatal(err)
	}
	caret := tab.Doc.Buffer.Selection()
	drawioAlignmentSelectNode(t, tester, "a", 0)
	drawioAlignmentSelectNode(t, tester, "c", ui.Cmd)
	if !tester.HasText("已选择 2 个图形") {
		t.Fatal("real App did not expose two selected vertices", tester.Texts())
	}
	history := tab.Doc.Buffer.History()
	tester.Key(0, ui.KeyDelete)
	tester.Frame()
	if !tab.Doc.Dirty() || tab.Doc.Buffer.History().UndoDepth != history.UndoDepth+1 {
		t.Fatal("multi delete not exposed as one document transaction")
	}
	after := tab.Doc.Buffer.String()
	doc, _, err := drawio.Parse([]byte(after), drawio.ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	page := doc.Pages[0]
	if len(page.Model.Nodes) != 2 || page.Model.Nodes[0].ID != "b" || page.Model.Nodes[1].ID != "untouched" || len(page.Model.Edges) != 0 {
		t.Fatalf("wrong XML closure after deleting primary and outgoing target: %#v %#v", page.Model.Nodes, page.Model.Edges)
	}
	if tab.Doc.Buffer.Selection() != caret {
		t.Fatal("batch delete relocated the source caret/selection")
	}
	unrelated := []string{
		drawioAlignmentOriginalGeometry[1],
		`id="untouched" value="未选" vertex="1" parent="1"`,
		`<?levnote keep="processing-instruction"?>`,
		`<v:item><![CDATA[unknown <xml> & data]]></v:item>`,
		before[strings.Index(before, `<diagram id="p2"`):strings.Index(before, `</mxfile>`)],
	}
	for _, literal := range unrelated {
		if !strings.Contains(after, literal) {
			t.Fatalf("unselected bytes lost from buffer: %s", literal)
		}
	}
	if !bytes.Equal(mustReadFile(t, path), baselineDisk) {
		t.Fatal("canvas deletion wrote disk before explicit save")
	}
	tester.Key(ui.Cmd, ui.KeyZ)
	if tab.Doc.Buffer.String() != before || tab.Doc.Buffer.Selection() != caret || tab.Doc.Buffer.History().UndoDepth != history.UndoDepth {
		t.Fatal("one undo did not restore deleted atoms and source selection")
	}
	drawioAlignmentSelectNode(t, tester, "a", 0)
	drawioAlignmentSelectNode(t, tester, "c", ui.Cmd)
	tester.Key(0, ui.KeyBackspace)
	if tab.Doc.Buffer.String() != after {
		t.Fatal("retained multi-selection could not be deleted again after undo")
	}
	if !a.Save(tab, false) || tab.Doc.Dirty() {
		t.Fatal("real App save failed", a.Message)
	}
	disk := mustReadFile(t, path)
	if !bytes.Equal(disk, drawioAlignmentDisk(after)) ||
		!bytes.HasPrefix(disk, []byte{0xef, 0xbb, 0xbf}) ||
		!bytes.Contains(disk, []byte{13, 10}) {
		t.Fatal("save lost BOM, CRLF or exact XML splices")
	}
	if !a.CloseTab(0) {
		t.Fatal("saved batch-delete tab did not close", a.Message)
	}
	a.Open(path)
	tester.Frame()
	tester.Frame()
	reopened := a.Current()
	if reopened == nil || reopened.Adapter == nil || reopened.Adapter.ID != "drawio" ||
		reopened.Doc.Buffer.String() != after || reopened.Doc.Dirty() || !reopened.Doc.BOM {
		t.Fatal("reopened disk did not match the exact saved deletion", a.Message)
	}
	if _, ok := tester.Find(drawioAlignmentCanvas); !ok {
		t.Fatal("saved graph did not render after reopen")
	}
	drawioAlignmentNodePoint(t, tester, "b")
}
