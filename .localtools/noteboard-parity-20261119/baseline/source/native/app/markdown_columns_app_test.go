package app

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
)

func TestMarkdownColumnsRealAdapterBOMCRLFSaveReopen(t *testing.T) {
	a, tab, tester, path := openMarkdownAlignmentApp(t, markdownAlignmentBody, 1180, 780, false, false)
	buffer := tab.Doc.Buffer
	openAlignmentAppCell(t, tester, ui.RoleCell, markdownCellBodyName, markdownCellBodyEditor)
	// Normal mode transition flushes this existing draft; column actions do not
	// introduce a separate path or persist the original file.
	replaceMarkdownCellDraft(t, tester, markdownCellBodyEditor, "改<&🙂")
	if !a.setMode(tab, "source") {
		t.Fatal("normal draft mode flush refused", a.Message)
	}
	written := strings.Replace(markdownAlignmentBody, "| 甲 |", "| 改<&🙂 |", 1)
	if buffer.String() != written || buffer.History().UndoDepth != 1 || !tab.Doc.Dirty() {
		t.Fatal("mode switch lost normal cell flush")
	}
	if !a.setMode(tab, "preview") {
		t.Fatal("preview return refused")
	}
	tester.Frame()
	tester.Frame()
	openAlignmentAppCell(t, tester, ui.RoleColumnHeader, markdownCellHeaderName, markdownCellHeaderEditor)
	activateAlignmentApp(t, tester, "在此列后插入列", true)
	inserted := strings.Replace(written,
		"| 名称 | 说明 |\r\n| \t:----- \t| ---: |\r\n| 改<&🙂 | **字面**<&🙂 |\r\n| 乙\\|丙 | [本地](a.md) |",
		"| 名称 || 说明 |\r\n| \t:----- \t|---| ---: |\r\n| 改<&🙂 || **字面**<&🙂 |\r\n| 乙\\|丙 || [本地](a.md) |", 1)
	if buffer.String() != inserted || buffer.History().UndoDepth != 2 || !tab.Doc.Dirty() ||
		!bytes.Equal(mustReadFile(t, path), alignmentDisk(markdownAlignmentBody)) {
		t.Fatal("insert did not preserve source/disk transaction", buffer.String())
	}
	if cell := requireMarkdownCellAccess(t, tester, ui.RoleColumnHeader, "表头，第 1 行，第 3 列：说明"); cell.Disabled {
		t.Fatal("inserted column descriptors not visible")
	}
	openAlignmentAppCell(t, tester, ui.RoleCell, "数据，第 2 行，第 2 列：（空单元格）", "编辑单元格 · 数据 · 第 2 行 · 第 2 列")
	activateAlignmentApp(t, tester, "删除此列", false)
	if buffer.String() != written || buffer.History().UndoDepth != 3 || !tab.Doc.Dirty() ||
		!bytes.Equal(mustReadFile(t, path), alignmentDisk(markdownAlignmentBody)) {
		t.Fatal("delete touched another column or original disk")
	}
	tab.Editor.Command("undo")
	if buffer.String() != inserted {
		t.Fatal("column deletion was not one undo")
	}
	tab.Editor.Command("redo")
	if buffer.String() != written {
		t.Fatal("column deletion redo changed neighboring content")
	}
	// Save the structural result after proving deletion undo/redo, so reopen
	// must preserve the inserted column rather than only the earlier cell edit.
	tab.Editor.Command("undo")
	if buffer.String() != inserted || !bytes.Equal(mustReadFile(t, path), alignmentDisk(markdownAlignmentBody)) {
		t.Fatal("restoring the inserted candidate touched original disk")
	}
	if !a.Save(tab, false) || !bytes.Equal(mustReadFile(t, path), alignmentDisk(inserted)) ||
		tab.Doc.Dirty() || tab.Doc.Encoding != document.UTF8 || !tab.Doc.BOM {
		t.Fatal("Save changed BOM/CRLF or exact neighboring bytes", a.Message)
	}
	reopened := newTestApp(t, filepath.Join(t.TempDir(), "config"))
	reopened.Open(path)
	rtab := reopened.Current()
	rtester := ui.NewTester(reopened.View, 1180, 780)
	rtester.Frame()
	if rtab == nil || rtab.Adapter == nil || rtab.Adapter.ID != "markdown" || rtab.Doc.Buffer.String() != inserted ||
		rtab.Doc.Encoding != document.UTF8 || !rtab.Doc.BOM || rtab.Doc.Dirty() {
		t.Fatal("registered adapter reopen lost exact bytes/metadata")
	}
	requireMarkdownCellAccess(t, rtester, ui.RoleCell, "数据，第 2 行，第 1 列：改<&🙂")
	requireMarkdownCellAccess(t, rtester, ui.RoleColumnHeader, "表头，第 1 行，第 3 列：说明")
	requireMarkdownCellAccess(t, rtester, ui.RoleColumnHeader, "表头，第 1 行，第 2 列：（空单元格）")
	requireMarkdownCellAccess(t, rtester, ui.RoleCell, "数据，第 2 行，第 2 列：（空单元格）")
}
