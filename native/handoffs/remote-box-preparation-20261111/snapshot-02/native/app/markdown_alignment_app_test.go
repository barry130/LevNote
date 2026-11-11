package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/text"
)

const markdownAlignmentBody = "# 对齐🙂\r\n\r\n前面的正文\r\n\r\n" +
	"| 名称 | 说明 |\r\n| \t:----- \t| ---: |\r\n" +
	"| 甲 | **字面**<&🙂 |\r\n| 乙\\|丙 | [本地](a.md) |\r\n\r\n" +
	"中间正文\r\n\r\n| 另一 | 表 |\r\n| --- | :----: |\r\n| A | B |\r\n\r\n尾部正文\r\n"

const markdownAlignmentNarrowBody = "| 名称 | 说明 |\r\n| \t:----- \t| ---: |\r\n" +
	"| 甲 | 字面<&🙂 |\r\n| 乙\\|丙 | 原文 |\r\n"

func alignmentDisk(source string) []byte {
	return append([]byte{0xef, 0xbb, 0xbf}, []byte(source)...)
}

func openMarkdownAlignmentApp(t *testing.T, source string, width, height int, split, dark bool) (*App, *Tab, *ui.Tester, string) {
	t.Helper()
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	if err := os.Mkdir(workspace, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(workspace, "alignment.md")
	if err := os.WriteFile(path, alignmentDisk(source), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Root, a.Dark = workspace, dark
	a.refreshTreeNow()
	a.Open(path)
	tab := a.Current()
	if tab == nil || tab.Adapter == nil || tab.Adapter.ID != "markdown" || tab.Doc.Encoding != document.UTF8 || !tab.Doc.BOM || tab.Doc.Dirty() {
		t.Fatal("alignment fixture did not open through registered UTF8-BOM adapter")
	}
	if split && !a.setMode(tab, "split") {
		t.Fatal("registered split view unavailable")
	}
	tester := ui.NewTester(a.View, width, height)
	tester.SetDark(dark)
	tester.Frame()
	tester.Frame()
	return a, tab, tester, path
}

func openAlignmentAppCell(t *testing.T, tester *ui.Tester, role ui.Role, name, editorLabel string) {
	t.Helper()
	// A separate physical double-click must not continue a previous opening
	// through controls that the new panel places at the old cell coordinates.
	tester.Advance(time.Second)
	openMarkdownCellPointer(t, tester, requireMarkdownCellAccess(t, tester, role, name), editorLabel)
}

func activateAlignmentApp(t *testing.T, tester *ui.Tester, name string, keyboard bool) {
	t.Helper()
	if keyboard {
		tester.Key(0, ui.KeyF6)
		if !requireMarkdownRowButton(t, tester, markdownRowInsert, false).Focused {
			t.Fatal("F6 did not enter native table actions from confirmed cell")
		}
		for step := 0; step < 9 && !requireMarkdownRowButton(t, tester, name, false).Focused; step++ {
			tester.Key(0, ui.KeyTab)
		}
		if !requireMarkdownRowButton(t, tester, name, false).Focused {
			t.Fatal("Tab did not reach native alignment")
		}
		tester.Key(0, ui.KeyEnter)
	} else {
		button := requireMarkdownRowButton(t, tester, name, false)
		tester.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
	}
	tester.Frame()
	tester.Frame()
}

func requireAlignmentPanelClosed(t *testing.T, tester *ui.Tester, label string) {
	t.Helper()
	if _, ok := tester.AccessByRole(ui.RoleTextField, label); ok {
		t.Fatal("completed alignment/noop recreated the cell panel")
	}
	if _, ok := tester.AccessByRole(ui.RoleButton, "第1列默认对齐"); ok {
		t.Fatal("completed alignment/noop retained stale actions")
	}
}

func TestMarkdownAlignmentRealAdapterExactNoopSaveReopenModesAndHTML(t *testing.T) {
	a, tab, tester, path := openMarkdownAlignmentApp(t, markdownAlignmentBody, 1180, 780, false, false)
	buffer := tab.Doc.Buffer
	openAlignmentAppCell(t, tester, ui.RoleColumnHeader, markdownCellHeaderName, markdownCellHeaderEditor)
	version, selection, history := buffer.Version(), buffer.Selection(), buffer.History()
	activateAlignmentApp(t, tester, "第1列左对齐", true)
	requireAlignmentPanelClosed(t, tester, markdownCellHeaderEditor)
	if buffer.String() != markdownAlignmentBody || buffer.Version() != version || buffer.Selection() != selection || buffer.History() != history ||
		tab.Doc.Dirty() || !bytes.Equal(mustReadFile(t, path), alignmentDisk(markdownAlignmentBody)) {
		t.Fatal("clean semantic no-op changed source/disk/history/dirty/selection")
	}
	if !a.Save(tab, false) || !bytes.Equal(mustReadFile(t, path), alignmentDisk(markdownAlignmentBody)) {
		t.Fatal("no-op save normalized BOM/CRLF/marker padding")
	}
	openAlignmentAppCell(t, tester, ui.RoleColumnHeader, markdownCellHeaderName, markdownCellHeaderEditor)
	selection = text.Range{Anchor: buffer.LenUTF16(), Head: text.UTF16Len("# 对齐🙂")}
	if err := buffer.SetSelection(selection.Anchor, selection.Head); err != nil {
		t.Fatal(err)
	}
	activateAlignmentApp(t, tester, "第1列居中对齐", true)
	centered := strings.Replace(markdownAlignmentBody, "\t:----- \t", "\t:-----: \t", 1)
	if buffer.String() != centered || buffer.History().UndoDepth != 1 || buffer.Selection() != (text.Range{Anchor: selection.Anchor + 1, Head: selection.Head}) || !tab.Doc.Dirty() {
		t.Fatal("selected header column patch was not exact", buffer.String(), buffer.History())
	}
	tab.Editor.Command("undo")
	if buffer.String() != markdownAlignmentBody || buffer.Selection() != selection || tab.Doc.Dirty() {
		t.Fatal("header alignment undo failed clean baseline")
	}
	tab.Editor.Command("redo")
	if buffer.String() != centered {
		t.Fatal("header alignment redo failed")
	}
	for _, mode := range []string{"source", "split", "preview"} {
		history := buffer.History()
		if !a.setMode(tab, mode) {
			t.Fatal("completed alignment blocked mode", mode, a.Message)
		}
		tester.Frame()
		tester.Frame()
		if tab.Doc.Buffer != buffer || buffer.String() != centered || buffer.History() != history {
			t.Fatal("mode switch changed aligned source/history", mode)
		}
	}
	const secondBody = "数据，第 2 行，第 2 列：**字面**<&🙂"
	const secondEditor = "编辑单元格 · 数据 · 第 2 行 · 第 2 列"
	openAlignmentAppCell(t, tester, ui.RoleCell, secondBody, secondEditor)
	activateAlignmentApp(t, tester, "第2列默认对齐", false)
	changed := strings.Replace(centered, "| ---: |\r\n", "| --- |\r\n", 1)
	if buffer.String() != changed || buffer.History().UndoDepth != 2 {
		t.Fatal("body column default did not remove just selected colon")
	}
	if !bytes.Equal(mustReadFile(t, path), alignmentDisk(markdownAlignmentBody)) || !a.Save(tab, false) ||
		!bytes.Equal(mustReadFile(t, path), alignmentDisk(changed)) || tab.Doc.Encoding != document.UTF8 || !tab.Doc.BOM || tab.Doc.Dirty() {
		t.Fatal("alignment Save did not preserve exact BOM/CRLF bytes")
	}
	reopened := newTestApp(t, filepath.Join(t.TempDir(), "config"))
	reopened.Open(path)
	rtab := reopened.Current()
	rtester := ui.NewTester(reopened.View, 1180, 780)
	rtester.Frame()
	if rtab == nil || rtab.Adapter == nil || rtab.Adapter.ID != "markdown" || rtab.Doc.Buffer.String() != changed || !rtab.Doc.BOM || rtab.Doc.Encoding != document.UTF8 || rtab.Doc.Dirty() {
		t.Fatal("registered adapter reopen changed alignment document")
	}
	version, selection, history = rtab.Doc.Buffer.Version(), rtab.Doc.Buffer.Selection(), rtab.Doc.Buffer.History()
	// Registered reference View.Export, not the asynchronous worker route.
	html, err := rtab.View.Export("html")
	wantFirst := "<table>\n<thead>\n<tr><th style=\"text-align:center\">名称</th><th>说明</th></tr>\n</thead>\n<tbody>\n" +
		"<tr><td style=\"text-align:center\">甲</td><td>**字面**&lt;&amp;🙂</td></tr>\n" +
		"<tr><td style=\"text-align:center\">乙\\|丙</td><td>[本地](a.md)</td></tr>\n</tbody>\n</table>\n"
	wantSecond := "<tr><th>另一</th><th style=\"text-align:center\">表</th></tr>"
	if err != nil || !bytes.Contains(html, []byte(wantFirst)) || !bytes.Contains(html, []byte(wantSecond)) ||
		bytes.Count(html, []byte("<table>")) != 2 || bytes.Contains(html, []byte("text-align:right")) ||
		bytes.Contains(html, []byte("href=\"a.md\"")) || bytes.Contains(html, []byte("<strong>字面")) {
		t.Fatalf("independent column/literal HTML oracle failed: %v\n%s", err, html)
	}
	if rtab.Doc.Buffer.Version() != version || rtab.Doc.Buffer.Selection() != selection || rtab.Doc.Buffer.History() != history ||
		rtab.Doc.Buffer.String() != changed || !bytes.Equal(mustReadFile(t, path), alignmentDisk(changed)) || !rtab.Doc.BOM || rtab.Doc.Encoding != document.UTF8 {
		t.Fatal("reference export changed source/history/disk/BOM")
	}
}

func TestMarkdownAlignmentRealAdapterPendingCellAndNoopUndo(t *testing.T) {
	for _, noop := range []bool{false, true} {
		t.Run(map[bool]string{false: "separate-two-edits", true: "cell-only-noop"}[noop], func(t *testing.T) {
			a, tab, tester, path := openMarkdownAlignmentApp(t, markdownAlignmentBody, 1180, 780, false, false)
			openAlignmentAppCell(t, tester, ui.RoleCell, markdownCellBodyName, markdownCellBodyEditor)
			replaceMarkdownCellDraft(t, tester, markdownCellBodyEditor, "改<&🙂")
			if tab.Doc.Dirty() || tab.Doc.Buffer.String() != markdownAlignmentBody {
				t.Fatal("composition was implicitly written before action")
			}
			action := "第1列右对齐"
			if noop {
				action = "第1列左对齐"
			}
			activateAlignmentApp(t, tester, action, false)
			requireAlignmentPanelClosed(t, tester, markdownCellBodyEditor)
			cellOnly := strings.Replace(markdownAlignmentBody, "| 甲 |", "| 改<&🙂 |", 1)
			want, depth := cellOnly, 1
			if !noop {
				want, depth = strings.Replace(cellOnly, "\t:----- \t", "\t-----: \t", 1), 2
			}
			if tab.Doc.Buffer.String() != want || tab.Doc.Buffer.History().UndoDepth != depth || !tab.Doc.Dirty() {
				t.Fatalf("cell+alignment undo contract failed: %q history=%+v message=%s", tab.Doc.Buffer.String(), tab.Doc.Buffer.History(), a.Message)
			}
			tab.Editor.Command("undo")
			if !noop {
				if tab.Doc.Buffer.String() != cellOnly {
					t.Fatal("first undo did not isolate alignment")
				}
				tab.Editor.Command("undo")
			}
			if tab.Doc.Buffer.String() != markdownAlignmentBody || tab.Doc.Buffer.History().UndoDepth != 0 || tab.Doc.Dirty() {
				t.Fatal("cell undo did not restore exact clean baseline")
			}
			tab.Editor.Command("redo")
			if tab.Doc.Buffer.String() != cellOnly {
				t.Fatal("first redo changed more than cell")
			}
			if !noop {
				tab.Editor.Command("redo")
			}
			if tab.Doc.Buffer.String() != want || !bytes.Equal(mustReadFile(t, path), alignmentDisk(markdownAlignmentBody)) {
				t.Fatal("redo or staged action mutated disk unexpectedly")
			}
		})
	}
}

func TestMarkdownAlignmentRealAdapterRefusalRecoveryReadonly(t *testing.T) {
	t.Run("split-conflict-recovery", func(t *testing.T) {
		a, tab, tester, path := openMarkdownAlignmentApp(t, markdownAlignmentBody, 1180, 780, true, true)
		openAlignmentAppCell(t, tester, ui.RoleCell, markdownCellBodyName, markdownCellBodyEditor)
		replaceMarkdownCellDraft(t, tester, markdownCellBodyEditor, "草稿🙂")
		start := text.UTF16Len(markdownAlignmentBody[:strings.Index(markdownAlignmentBody, "**字面**")])
		tab.Editor.ReplaceText(&ui.TextInputRange{Start: start, End: start + text.UTF16Len("**字面**")}, "外部")
		want := strings.Replace(markdownAlignmentBody, "**字面**", "外部", 1)
		tester.Frame()
		version, history := tab.Doc.Buffer.Version(), tab.Doc.Buffer.History()
		activateAlignmentApp(t, tester, "第1列居中对齐", false)
		if tab.Doc.Buffer.String() != want || tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.History() != history ||
			!strings.Contains(a.Message, "对齐未修改") || tab.Mode != "split" || a.Save(tab, false) {
			t.Fatal("split conflict escaped source/history/save barriers", a.Message)
		}
		requireMarkdownCellEditor(t, tester, markdownCellBodyEditor)
		button := requireMarkdownRowButton(t, tester, markdownCellRecoverLabel, false)
		tester.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
		tester.Frame()
		recovered := a.Current()
		if len(a.Tabs) != 2 || recovered == tab || !recovered.Doc.IsNew || recovered.Doc.Path != "" || recovered.Doc.Buffer.String() != "草稿🙂" ||
			tab.Doc.Buffer.String() != want || tab.Doc.Buffer.History() != history || !bytes.Equal(mustReadFile(t, path), alignmentDisk(markdownAlignmentBody)) {
			t.Fatal("app recovery failed exact draft/source/disk ownership")
		}
	})
	t.Run("cell-written-alignment-refused", func(t *testing.T) {
		source := "| 名称 | 说明 |\r\n|" + strings.Repeat("-", 4096) + "|---|\r\n| 甲 | 字面 |\r\n"
		a, tab, tester, path := openMarkdownAlignmentApp(t, source, 1180, 780, false, false)
		openAlignmentAppCell(t, tester, ui.RoleCell, markdownCellBodyName, markdownCellBodyEditor)
		replaceMarkdownCellDraft(t, tester, markdownCellBodyEditor, "已写🙂")
		activateAlignmentApp(t, tester, "第1列居中对齐", false)
		want := strings.Replace(source, "| 甲 |", "| 已写🙂 |", 1)
		if tab.Doc.Buffer.String() != want || tab.Doc.Buffer.History().UndoDepth != 1 || !strings.Contains(a.Message, "单元格已写回，对齐未修改") || !strings.Contains(a.Message, "上限") {
			t.Fatal("post-cell alignment refusal is untruthful", a.Message)
		}
		requireAlignmentPanelClosed(t, tester, markdownCellBodyEditor)
		tab.Editor.Command("undo")
		if tab.Doc.Buffer.String() != source || tab.Doc.Dirty() || !bytes.Equal(mustReadFile(t, path), alignmentDisk(source)) {
			t.Fatal("post-cell refusal lost independent undo or original file")
		}
	})
	t.Run("readonly", func(t *testing.T) {
		a := newTestApp(t, t.TempDir())
		doc := document.New(markdownAlignmentNarrowBody)
		doc.ReadOnly = true
		tab := a.add(doc, "readonly-alignment.md", "")
		tester := ui.NewTester(a.View, 1180, 780)
		tester.Frame()
		version, history := tab.Doc.Buffer.Version(), tab.Doc.Buffer.History()
		openMarkdownCellPointerWithoutEditor(t, tester)
		if tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.History() != history || tab.Doc.Buffer.String() != markdownAlignmentNarrowBody || !strings.Contains(a.Message, "只读") {
			t.Fatal("readonly cell/alignment activation changed source")
		}
		if _, ok := tester.AccessByRole(ui.RoleButton, "第1列左对齐"); ok {
			t.Fatal("readonly cell activation exposed editing actions")
		}
	})
}

func TestMarkdownAlignmentAppNarrowCaptures(t *testing.T) {
	out := os.Getenv("LEVNOTE_MARKDOWN_ALIGNMENT_SCREENSHOT_DIR")
	if out == "" {
		t.Skip("set LEVNOTE_MARKDOWN_ALIGNMENT_SCREENSHOT_DIR for integrated alignment captures")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	for _, dark := range []bool{false, true} {
		name := "light-column"
		if dark {
			name = "dark-recovery"
		}
		t.Run(name, func(t *testing.T) {
			a, tab, tester, path := openMarkdownAlignmentApp(t, markdownAlignmentNarrowBody, 760, 480, true, dark)
			openAlignmentAppCell(t, tester, ui.RoleCell, markdownCellBodyName, markdownCellBodyEditor)
			value := "确认🙂"
			if dark {
				value = "坏|值"
			}
			replaceMarkdownCellDraft(t, tester, markdownCellBodyEditor, value)
			tester.Key(0, ui.KeyF6)
			if field := requireMarkdownCellEditor(t, tester, markdownCellBodyEditor); !field.Focused || !tester.HasText("请先确认输入法组合") {
				t.Fatal("integrated F6 failed to retain composing draft")
			}
			if dark {
				button := requireMarkdownRowButton(t, tester, "第1列居中对齐", false)
				x, y := button.Bounds.X+4, button.Bounds.Y+4
				tester.Press(x, y)
				if after := requireMarkdownRowButton(t, tester, "第1列居中对齐", false); after.Bounds != button.Bounds {
					t.Fatal("composition confirmation moved held alignment control", button.Bounds, after.Bounds)
				}
				tester.Release(x, y)
				tester.Frame()
				if !strings.Contains(a.Message, "对齐未修改") {
					t.Fatal("refused capture lacks alignment distinction", a.Message)
				}
			} else {
				tester.Type(value) // native commit replaces the active marked text
				tester.Key(0, ui.KeyF6)
				if !requireMarkdownRowButton(t, tester, markdownRowInsert, false).Focused {
					t.Fatal("confirmed F6 route is unavailable")
				}
				for step := 0; step < 7; step++ {
					tester.Key(0, ui.KeyTab)
				}
				if !requireMarkdownRowButton(t, tester, "第1列居中对齐", false).Focused {
					t.Fatal("Tab did not reach actual alignment action")
				}
				// Keyboard focus shows MyGo's tooltip immediately. Close it
				// through the normal overlay Escape route, keeping button focus
				// for the later Enter and leaving all editing controls visible.
				if _, ok := tester.AccessByRole(ui.RoleTooltip, ""); !ok {
					t.Fatal("keyboard tooltip precondition is missing")
				}
				tester.Move(20, 240)
				tester.Key(0, ui.KeyEscape)
				if _, ok := tester.AccessByRole(ui.RoleTooltip, ""); ok || !requireMarkdownRowButton(t, tester, "第1列居中对齐", false).Focused {
					t.Fatal("normal tooltip dismissal changed alignment focus or left overlay")
				}
			}
			tester.Frame()
			imagePath := filepath.Join(out, "markdown-alignment-"+name+"-760x480.png")
			captureAcceptanceTester(t, imagePath, tester) // preserve failing geometry too
			// Resolve the real status text and preview column from this frame.
			// app.View centers this text in a 29-DIP status row. Use the real
			// text center and declared row height, not text Y minus padding.
			status, statusOK := tester.Find(a.Message)
			preview, previewOK := tester.Find("双击内容编辑 · Ctrl+Enter 完成")
			field := requireMarkdownCellEditor(t, tester, markdownCellBodyEditor)
			if a.Message == "" || !statusOK || status.Y < 430 || status.H <= 0 || status.Y+status.H > 480 ||
				!previewOK || preview.W < 240 || preview.W > 280 || preview.X+preview.W > 760 {
				t.Fatalf("actual status/preview geometry unavailable: status=%+v preview=%+v", status, preview)
			}
			statusTop := status.Y + status.H/2 - 29.0/2
			if r := field.Bounds; r.X < preview.X || r.X+r.W > preview.X+preview.W || r.Y < preview.Y+preview.H ||
				r.W < 190 || r.H < 48 || r.Y+r.H > statusTop {
				t.Fatalf("cell input is clipped or outside actual split preview: field=%+v preview=%+v status=%+v", r, preview, status)
			}
			t.Logf("actual split preview=%+v input=%+v statusText=%+v statusTop=%g", preview, field.Bounds, status, statusTop)
			labels := []string{markdownCellFinishLabel, markdownCellDiscardLabel, markdownRowInsert, markdownRowDelete, "第1列默认对齐", "第1列左对齐", "第1列居中对齐", "第1列右对齐"}
			if dark {
				labels = append(labels, markdownCellRecoverLabel)
			}
			for _, label := range labels {
				button := requireMarkdownRowButton(t, tester, label, false)
				r := button.Bounds
				t.Logf("actual integrated control %q bounds=%+v", label, r)
				if r.X < preview.X || r.Y < field.Bounds.Y+field.Bounds.H || r.X+r.W > preview.X+preview.W || r.Y+r.H > statusTop {
					t.Fatalf("alignment control overlaps real status/chrome: %q %+v image=%s", label, r, imagePath)
				}
			}
			if !tester.HasText("第1列 · 当前：左") || tab.Doc.Buffer.String() != markdownAlignmentNarrowBody || tab.Doc.Dirty() ||
				!bytes.Equal(mustReadFile(t, path), alignmentDisk(markdownAlignmentNarrowBody)) {
				t.Fatal("capture lacks current column or mutated source/disk")
			}
			if dark {
				button := requireMarkdownRowButton(t, tester, markdownCellRecoverLabel, false)
				tester.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
				tester.Frame()
				if len(a.Tabs) != 2 || a.Current() == tab || !a.Current().Doc.IsNew || a.Current().Doc.Path != "" || a.Current().Doc.Buffer.String() != value {
					t.Fatal("visible recovery did not create exact unsaved draft")
				}
			} else {
				tester.Key(0, ui.KeyEnter)
				tester.Frame()
				want := strings.Replace(markdownAlignmentNarrowBody, "| 甲 |", "| "+value+" |", 1)
				want = strings.Replace(want, "\t:----- \t", "\t:-----: \t", 1)
				if tab.Doc.Buffer.String() != want || tab.Doc.Buffer.History().UndoDepth != 2 {
					t.Fatal("confirmed text was duplicated/lost or keyboard alignment failed", tab.Doc.Buffer.String())
				}
				requireAlignmentPanelClosed(t, tester, markdownCellBodyEditor)
			}
		})
	}
}
