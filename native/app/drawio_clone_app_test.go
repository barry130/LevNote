package app

import (
	"bytes"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/formats/drawio"
)

func TestDrawioCloneAppExactDiskUndoRedoAndReopen(t *testing.T) {
	a, tab, tester, path := openDrawioAlignmentApp(t, drawioAlignmentAppOptions{})
	before, buffer, baselineDisk := tab.Doc.Buffer.String(), tab.Doc.Buffer, mustReadFile(t, path)
	if err := buffer.SetSelection(18, 31); err != nil { t.Fatal(err) }
	caret := buffer.Selection()
	page := before[strings.Index(before, "<diagram id=\"p2\""):strings.Index(before, "</mxfile>")]
	drawioAlignmentSelectNode(t, tester, "a", 0)
	drawioAlignmentSelectNode(t, tester, "c", ui.Cmd)
	if !tester.HasText("已选择 2 个图形") { t.Fatal("clone fixture did not select two vertices", tester.Texts()) }
	history := buffer.History()
	usedBefore := usedDiagramIDs(t, before)
	tester.Key(ui.Cmd, ui.KeyD)
	tester.Frame(); tester.Frame()
	if buffer.History().UndoDepth != history.UndoDepth+1 || !tab.Doc.Dirty() { t.Fatal("clone was not one dirty document transaction") }
	after := buffer.String()
	if !tester.HasText("已克隆 2 个图形及内部连线到当前页；可撤销") { t.Fatal("clone result message missing", tester.Texts()) }
	doc, _, err := drawio.Parse([]byte(after), drawio.ParseOptions{})
	if err != nil || len(doc.Pages) != 2 || len(doc.Pages[0].Model.Nodes) != 6 || len(doc.Pages[0].Model.Edges) != 2 { t.Fatal("clone produced wrong graph", err) }
	clonedNodes := map[string]drawio.Node{}
	for _, node := range doc.Pages[0].Model.Nodes { if !usedBefore[node.ID] { clonedNodes[node.ID] = node } }
	if len(clonedNodes) != 2 { t.Fatal("expected exactly two cloned vertices", clonedNodes) }
	var cloneA, cloneC drawio.Node
	foundA, foundC := false, false
	for _, node := range clonedNodes {
		switch {
		case node.X == 11.5 && node.Y == 32.25 && node.Width == 81 && node.Height == 42 && node.Parent == "1" && !foundA:
			cloneA, foundA = node, true
		case node.X == 72.75 && node.Y == 214.25 && node.Width == 99 && node.Height == 31 && node.Parent == "1" && !foundC:
			cloneC, foundC = node, true
		default:
			t.Fatal("cloned vertex has wrong geometry or parent", node)
		}
	}
	if !foundA || !foundC { t.Fatal("cloned vertices missing shifted copies of a and c", clonedNodes) }
	edgeByID := map[string]drawio.Edge{}
	for _, edge := range doc.Pages[0].Model.Edges { edgeByID[edge.ID] = edge }
	if len(edgeByID) != 2 || edgeByID["edge"].SourceID != "a" || edgeByID["edge"].TargetID != "c" { t.Fatal("original outer edge was rewired", edgeByID) }
	var clonedEdge *drawio.Edge
	for _, edge := range doc.Pages[0].Model.Edges { if !usedBefore[edge.ID] { copied := edge; clonedEdge = &copied } }
	if clonedEdge == nil || clonedEdge.SourceID != cloneA.ID || clonedEdge.TargetID != cloneC.ID || len(clonedEdge.Points) != 2 { t.Fatal("internal edge was not copied exactly once", clonedEdge) }
	if buffer.Selection() != caret || !strings.Contains(after, page) || !bytes.Equal(mustReadFile(t, path), baselineDisk) { t.Fatal("clone changed caret/page/disk before explicit save") }
	for _, key := range []ui.Key{ui.KeyZ, ui.KeyY} {
		tester.Key(ui.Cmd, key)
		tester.Frame(); tester.Frame()
	}
	if buffer.String() != after || buffer.Selection() != caret { t.Fatal("repeat undo/redo lost clone bytes or source caret") }
	tester.Key(ui.Cmd, ui.KeyZ); tester.Frame(); tester.Frame()
	if buffer.String() != before || buffer.Selection() != caret { t.Fatal("Ctrl+Z did not restore pre-clone bytes and caret") }
	if !a.Save(tab, false) || tab.Doc.Dirty() { t.Fatal("saving after clone undo failed", a.Message) }
	disk := mustReadFile(t, path)
	if !bytes.Equal(disk, drawioAlignmentDisk(before)) || !bytes.HasPrefix(disk, []byte{0xef,0xbb,0xbf}) || !bytes.Contains(disk, []byte{13,10}) { t.Fatal("saved XML lost BOM/CRLF or did not return exact cloned-undo bytes") }
	if !a.CloseTab(0) { t.Fatal("saved-state blocked tab close", a.Message) }
	a.Open(path); tester.Frame(); tester.Frame()
	reopened := a.Current(); if reopened == nil || reopened.Doc.Buffer.String() != before || reopened.Doc.Dirty() || !reopened.Doc.BOM { t.Fatal("reopen did not return exact disk source") }
	if _, ok := tester.Find(drawioAlignmentCanvas); !ok { t.Fatal("reopened cloned-save graph did not render", tester.Texts()) }
}

func usedDiagramIDs(t *testing.T, source string) map[string]bool {
	t.Helper()
	doc, _, err := drawio.Parse([]byte(source), drawio.ParseOptions{})
	if err != nil { t.Fatal(err) }
	used := map[string]bool{}
	for _, node := range doc.Pages[0].Model.Nodes { used[node.ID] = true }
	for _, edge := range doc.Pages[0].Model.Edges { used[edge.ID] = true }
	return used
}