package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

func TestDrawioAlignmentAppNoopKeepsRedoAndSelectionHistory(t *testing.T) {
	a, tab, tester, path := openDrawioAlignmentApp(t, drawioAlignmentAppOptions{})
	original, buffer := tab.Doc.Buffer.String(), tab.Doc.Buffer
	drawioAlignmentSelectThree(t, tester)
	drawioAlignmentClick(t, tester, "左对齐")
	left := drawioAlignmentSource(drawioAlignmentCases[0].geometry)
	drawioAlignmentClick(t, tester, "顶对齐")
	both := drawioAlignmentSource([3]string{
		`<mxGeometry x="-12.50" y="8.250" width="8.1e1" height="42.00" as="geometry"/>`,
		`<mxGeometry height='25.0' x='-12.5' width='62.00' as='geometry' y='8.25'/>`,
		`<mxGeometry y="8.25" width="99.0" x="-12.5" height="3.1e1" as="geometry"/>`,
	})
	if buffer.String() != both || buffer.History().UndoDepth != 2 {
		t.Fatal("two commands did not make two whole-selection transactions")
	}
	tester.Command("undo")
	tester.Frame()
	if buffer.String() != left || !buffer.CanRedo() {
		t.Fatal("undo did not retain top-alignment redo")
	}
	version, selection, history := buffer.Version(), buffer.Selection(), buffer.History()
	// Ctrl toggles must not enter the document history or clear redo.
	drawioAlignmentSelectNode(t, tester, "b", ui.Cmd)
	if !tester.HasText("已选择 2 个图形") {
		t.Fatal("Ctrl did not remove the primary selection", tester.Texts())
	}
	drawioAlignmentSelectNode(t, tester, "b", ui.Cmd)
	if !tester.HasText("已选择 3 个图形") {
		t.Fatal("Ctrl did not restore the third selection", tester.Texts())
	}
	drawioAlignmentClick(t, tester, "左对齐")
	if buffer.String() != left || buffer.Version() != version || buffer.Selection() != selection || buffer.History() != history || !buffer.CanRedo() || !tab.Doc.Dirty() {
		t.Fatal("no-op alignment or selection toggles consumed redo/history")
	}
	// Multi-select Delete/Backspace deliberately consume one transaction;
	// exact deletion and undo-scope coverage lives in drawio_multi_delete_test.go.
	if field, ok := tester.AccessByRole(ui.RoleTextField, "节点文本"); ok && !field.Disabled && !field.ReadOnly {
		t.Fatal("multiselection exposed writable primary properties")
	}
	tester.Command("redo")
	tester.Frame()
	if buffer.String() != both || buffer.History().UndoDepth != 2 {
		t.Fatal("retained redo did not restore the whole top alignment")
	}
	if !bytes.Equal(mustReadFile(t, path), drawioAlignmentDisk(original)) {
		t.Fatal("selection/noop/history actions wrote disk")
	}
	if !a.Save(tab, false) || !bytes.Equal(mustReadFile(t, path), drawioAlignmentDisk(both)) {
		t.Fatal("redo-save failed", a.Message)
	}
}

func TestDrawioAlignmentAppCapacityFailureKeepsRedoAndDisk(t *testing.T) {
	original := drawioAlignmentSource(drawioAlignmentOriginalGeometry)
	a, tab, tester, path := openDrawioAlignmentApp(t, drawioAlignmentAppOptions{limit: len(original)})
	buffer := tab.Doc.Buffer
	// Seed an independently meaningful redo without growing the limited buffer.
	end := buffer.LenUTF16()
	tab.Editor.ReplaceText(&ui.TextInputRange{Start: end - 2, End: end}, "")
	tab.Editor.Command("undo")
	tester.Frame()
	if buffer.String() != original || !buffer.CanRedo() || tab.Doc.Dirty() {
		t.Fatal("capacity fixture did not retain the restored disk baseline and redo")
	}
	drawioAlignmentSelectThree(t, tester)
	version, selection, history, snapshot := buffer.Version(), buffer.Selection(), buffer.History(), buffer.Snapshot()
	// Bottom alignment grows two y literals by one byte each; all targets
	// must remain unchanged when the public document budget admits neither.
	drawioAlignmentClick(t, tester, "底对齐")
	if buffer.Version() != version || buffer.Selection() != selection || buffer.History() != history || !buffer.EqualSnapshot(snapshot) || tab.Doc.Dirty() || !buffer.CanRedo() ||
		!bytes.Equal(mustReadFile(t, path), drawioAlignmentDisk(original)) || !tester.HasText("已选择 3 个图形") {
		t.Fatal("capacity failure changed source, selection, history, disk, or selected vertices", a.Message)
	}
	if !strings.Contains(a.Message, "内容上限") && !tester.HasText("内容上限") {
		t.Fatal("capacity failure was not visible", a.Message, tester.Texts())
	}
	if !a.Save(tab, false) || !bytes.Equal(mustReadFile(t, path), drawioAlignmentDisk(original)) {
		t.Fatal("unchanged document stopped saving after capacity failure", a.Message)
	}
	tester.Command("redo")
	tester.Frame()
	if buffer.String() != strings.TrimSuffix(original, "\r\n") {
		t.Fatal("capacity failure consumed or replaced the preexisting redo")
	}
	tester.Command("undo")
	tester.Frame()
	if buffer.String() != original || tab.Doc.Dirty() {
		t.Fatal("independent redo/undo did not restore the exact file baseline")
	}
}

func TestDrawioAlignmentAppReadonlyCustomRouteSaveAs(t *testing.T) {
	a, tab, tester, path := openDrawioAlignmentApp(t, drawioAlignmentAppOptions{readonly: true, customRoute: true})
	original, buffer := tab.Doc.Buffer.String(), tab.Doc.Buffer
	version, selection, history := buffer.Version(), buffer.Selection(), buffer.History()
	drawioAlignmentSelectThree(t, tester)
	for _, action := range drawioAlignmentCases {
		button := drawioAlignmentButton(t, tester, action.label, true)
		tester.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
		tester.Frame()
	}
	if buffer.String() != original || buffer.Version() != version || buffer.Selection() != selection || buffer.History() != history || tab.Doc.Dirty() ||
		!bytes.Equal(mustReadFile(t, path), drawioAlignmentDisk(original)) {
		t.Fatal("readonly multiselection mutated the original document")
	}
	copyPath := filepath.Join(filepath.Dir(path), "writable.diagramnote2")
	a.saveDialog = func(*Tab) (string, error) { return copyPath, nil }
	if !a.Save(tab, true) || tab.Doc.ReadOnly || tab.Editor.ReadOnly || tab.Doc.Path != copyPath {
		t.Fatal("readonly SaveAs did not yield a writable real document", a.Message)
	}
	tester.Frame()
	tester.Frame()
	requireDrawioRoute(t, tab)
	if !bytes.Equal(mustReadFile(t, path), drawioAlignmentDisk(original)) || !bytes.Equal(mustReadFile(t, copyPath), drawioAlignmentDisk(original)) {
		t.Fatal("readonly SaveAs changed XML/BOM before editing")
	}
	drawioAlignmentSelectThree(t, tester)
	drawioAlignmentClick(t, tester, "底对齐")
	want := drawioAlignmentSource(drawioAlignmentCases[5].geometry)
	if tab.Doc.Buffer.String() != want || !a.Save(tab, false) || !bytes.Equal(mustReadFile(t, copyPath), drawioAlignmentDisk(want)) || !bytes.Equal(mustReadFile(t, path), drawioAlignmentDisk(original)) {
		t.Fatal("writable SaveAs copy did not isolate alignment from readonly original", a.Message)
	}
	if !a.CloseTab(0) {
		t.Fatal(a.Message)
	}
	a.Open(copyPath)
	tester.Frame()
	reopened := a.Current()
	requireDrawioRoute(t, reopened)
	if reopened == tab || reopened.Doc.ReadOnly || !reopened.Doc.BOM || reopened.Doc.Dirty() || reopened.Doc.Buffer.String() != want {
		t.Fatal("custom-route aligned copy failed real close/reopen")
	}
	drawioAlignmentRequireGeometry(t, reopened.Doc.Buffer.String(), drawioAlignmentCases[5].xy)
}

func TestDrawioAlignmentAppLabelThenGeometryHaveSeparateUndo(t *testing.T) {
	a, tab, tester, path := openDrawioAlignmentApp(t, drawioAlignmentAppOptions{})
	original, buffer := tab.Doc.Buffer.String(), tab.Doc.Buffer
	drawioAlignmentSelectNode(t, tester, "a", 0)
	if err := tester.Click("节点文本"); err != nil {
		t.Fatal(err)
	}
	tester.Command("selectAll")
	tester.Type("独立标签🙂")
	tester.Frame()
	labelOnly := strings.Replace(original, `value="甲🙂"`, `value="独立标签🙂"`, 1)
	if buffer.String() != labelOnly || buffer.History().UndoDepth != 1 {
		t.Fatal("real label field did not commit its independent history")
	}
	drawioAlignmentSelectThree(t, tester)
	drawioAlignmentClick(t, tester, "水平居中")
	want := strings.Replace(drawioAlignmentSource(drawioAlignmentCases[1].geometry), `value="甲🙂"`, `value="独立标签🙂"`, 1)
	if buffer.String() != want || buffer.History().UndoDepth != 2 {
		t.Fatal("geometry merged with or discarded label transaction")
	}
	for _, expected := range []string{labelOnly, original} {
		tester.Command("undo")
		tester.Frame()
		if buffer.String() != expected {
			t.Fatal("undo did not respect label then geometry ordering", buffer.String())
		}
	}
	for _, expected := range []string{labelOnly, want} {
		tester.Command("redo")
		tester.Frame()
		if buffer.String() != expected {
			t.Fatal("redo did not respect label then geometry ordering", buffer.String())
		}
	}
	if !a.Save(tab, false) || !bytes.Equal(mustReadFile(t, path), drawioAlignmentDisk(want)) {
		t.Fatal("label+geometry redo did not save exact XML", a.Message)
	}
}

func TestDrawioAlignmentAppFailedDraftBlocksCtrlSelectionAndLifecycle(t *testing.T) {
	for _, failure := range []string{"source-conflict", "capacity"} {
		t.Run(failure, func(t *testing.T) {
			original := drawioAlignmentSource(drawioAlignmentOriginalGeometry)
			options := drawioAlignmentAppOptions{split: true}
			if failure == "capacity" {
				options.limit = len(original)
			}
			a, tab, tester, path := openDrawioAlignmentApp(t, options)
			drawioAlignmentSelectNode(t, tester, "a", 0)
			if err := tester.Click("节点文本"); err != nil {
				t.Fatal(err)
			}
			tester.Command("selectAll")
			const pending = "保留的节点草稿🙂"
			tester.Compose(pending, len([]rune(pending)))
			if tab.Doc.Buffer.String() != original {
				t.Fatal("preedit escaped to the main document")
			}
			want := original
			if failure == "source-conflict" {
				offset := strings.Index(original, "甲🙂")
				start := text.UTF16Len(original[:offset])
				tab.Editor.ReplaceText(&ui.TextInputRange{Start: start, End: start + text.UTF16Len("甲🙂")}, "外部🙂")
				want = strings.Replace(original, `value="甲🙂"`, `value="外部🙂"`, 1)
				tester.Frame()
			}
			buffer, view := tab.Doc.Buffer, tab.View
			version, selection, history := buffer.Version(), buffer.Selection(), buffer.History()
			drawioAlignmentSelectNode(t, tester, "b", ui.Cmd)
			if tester.HasText("已选择 2 个图形") || tester.HasText("已选择 3 个图形") {
				t.Fatal("Ctrl selection replaced an unresolved label owner")
			}
			if a.Save(tab, false) || a.setMode(tab, "source") || a.CloseTab(0) {
				t.Fatal("failed label flush crossed save/mode/close barrier", a.Message)
			}
			tester.Frame()
			if a.Current() != tab || tab.View != view || tab.Mode != "split" || buffer.String() != want || buffer.Version() != version || buffer.Selection() != selection || buffer.History() != history ||
				!bytes.Equal(mustReadFile(t, path), drawioAlignmentDisk(original)) || !tester.HasText("节点草稿尚未写回") {
				t.Fatal("failed Ctrl/flush discarded draft owner, source, disk, or failure details", a.Message)
			}
			if err := tester.Click("恢复为新文档"); err != nil {
				t.Fatal("real failed-draft recovery missing", err, tester.Texts())
			}
			tester.Frame()
			recovered := a.Current()
			// 恢复内容与既有 RecoverDraft 信封契约一致（mindmap 侧同款断言），草稿原文作为信封正文。
			wantRecovered := filepath.Base(path) + "\n节点 ID: a\n页面 ID: p1\n字段: 节点文本\n原字段存在: true\n\n" + pending
			if recovered == tab || recovered == nil || !recovered.Doc.IsNew || recovered.Doc.Path != "" || recovered.Doc.Buffer.String() != wantRecovered ||
				tab.Doc.Buffer.String() != want || !bytes.Equal(mustReadFile(t, path), drawioAlignmentDisk(original)) {
				got := "<nil>"
				if recovered != nil && recovered.Doc != nil {
					got = recovered.Doc.Buffer.String()
				}
				t.Fatalf("real recovery lost the exact pending draft or altered original; got=%q want=%q", got, wantRecovered)
			}
			recoveryPath := filepath.Join(filepath.Dir(path), "recovered-label.txt")
			a.saveDialog = func(*Tab) (string, error) { return recoveryPath, nil }
			if !a.Save(recovered, true) {
				t.Fatal(a.Message)
			}
			saved, err := os.ReadFile(recoveryPath)
			if err != nil || string(saved) != wantRecovered {
				t.Fatal("recovered label did not save as a real independent file", err)
			}
		})
	}
}
