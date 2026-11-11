package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/text"
)

const (
	markdownRowInsert = "在下方插入行"
	markdownRowDelete = "删除此数据行"
)

func requireMarkdownRowButton(t *testing.T, tester *ui.Tester, label string, disabled bool) ui.AccessInfo {
	t.Helper()
	info, ok := tester.AccessByRole(ui.RoleButton, label)
	if !ok || info.Disabled != disabled || (!disabled && !info.Focusable) || (disabled && info.Focusable) || info.Offscreen || info.Bounds.W <= 0 || info.Bounds.H <= 0 {
		t.Fatalf("row control %q unavailable: %+v texts=%q", label, info, tester.Texts())
	}
	return info
}

func activateMarkdownRow(t *testing.T, tester *ui.Tester, label string, keyboard bool) {
	t.Helper()
	if keyboard {
		// Begin at the cell editor's actual focus. F6 is the documented local
		// escape; no private Markdown view or pointer focus shortcut is used.
		tester.Key(0, ui.KeyF6)
		if !requireMarkdownRowButton(t, tester, markdownRowInsert, false).Focused {
			t.Fatal("F6 did not reach the first row button from the cell draft")
		}
		key := ui.KeyEnter
		if label == markdownRowDelete {
			tester.Key(0, ui.KeyTab)
			key = ui.KeySpace
		}
		if !requireMarkdownRowButton(t, tester, label, false).Focused {
			t.Fatal("keyboard traversal did not reach the requested row control")
		}
		tester.Key(0, key)
	} else {
		button := requireMarkdownRowButton(t, tester, label, false)
		tester.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
	}
	tester.Frame()
	tester.Frame()
}

func TestMarkdownRowsRealAdapterBytesUndoModesSaveReopenAndHTML(t *testing.T) {
	a, tab, tester, path := openRealMarkdownTables(t, 1180, 780)
	buffer := tab.Doc.Buffer
	if tab.Doc.Encoding != document.UTF8 || !tab.Doc.BOM {
		t.Fatal("fixture must retain decoded UTF-8 BOM metadata")
	}
	header := requireMarkdownCellAccess(t, tester, ui.RoleColumnHeader, markdownCellHeaderName)
	openMarkdownCellPointer(t, tester, header, markdownCellHeaderEditor)
	requireMarkdownRowButton(t, tester, markdownRowDelete, true)
	if !tester.HasText("表头不能删除；请选中数据行。") {
		t.Fatal("header deletion refusal has no visible explanation")
	}
	originalSelection := text.Range{Anchor: buffer.LenUTF16(), Head: 2}
	if err := buffer.SetSelection(originalSelection.Anchor, originalSelection.Head); err != nil {
		t.Fatal(err)
	}
	activateMarkdownRow(t, tester, markdownRowInsert, true)
	headerInserted := strings.Replace(markdownTablesBody, "| :--- | ---: |\r\n", "| :--- | ---: |\r\n| | |\r\n", 1)
	if buffer.String() != headerInserted || buffer.History().UndoDepth != 1 ||
		buffer.Selection() != (text.Range{Anchor: originalSelection.Anchor + 7, Head: 2}) {
		t.Fatalf("header insertion not an exact single source edit: %q history=%+v selection=%+v", buffer.String(), buffer.History(), buffer.Selection())
	}
	tab.Editor.Command("undo")
	if buffer.String() != markdownTablesBody || buffer.Selection() != originalSelection {
		t.Fatal("no-op cell plus row undo lost original bytes/selection")
	}
	tab.Editor.Command("redo")
	if buffer.String() != headerInserted {
		t.Fatal("header row redo failed")
	}
	tab.Editor.Command("undo")
	tester.Frame()

	body := requireMarkdownCellAccess(t, tester, ui.RoleCell, markdownCellBodyName)
	openMarkdownCellPointer(t, tester, body, markdownCellBodyEditor)
	replaceMarkdownCellDraft(t, tester, markdownCellBodyEditor, "改<&🙂")
	if buffer.String() != markdownTablesBody || buffer.History().UndoDepth != 0 {
		t.Fatal("cell composition must remain staged before the row action")
	}
	activateMarkdownRow(t, tester, markdownRowInsert, false)
	cellOnly := strings.Replace(markdownTablesBody, "| 甲 | **字面** |", "| 改<&🙂 | **字面** |", 1)
	inserted := strings.Replace(cellOnly, "| 改<&🙂 | **字面** |\r\n", "| 改<&🙂 | **字面** |\r\n| | |\r\n", 1)
	if buffer.String() != inserted || buffer.History().UndoDepth != 2 || !tab.Doc.Dirty() {
		t.Fatalf("pending composition and row action must be two edits: %q history=%+v message=%q", buffer.String(), buffer.History(), a.Message)
	}
	tab.Editor.Command("undo")
	if buffer.String() != cellOnly {
		t.Fatal("first undo did not isolate the row patch")
	}
	tab.Editor.Command("undo")
	if buffer.String() != markdownTablesBody {
		t.Fatal("second undo did not restore the original cell")
	}
	tab.Editor.Command("redo")
	if buffer.String() != cellOnly {
		t.Fatal("first redo must write only the cell")
	}
	tab.Editor.Command("redo")
	if buffer.String() != inserted {
		t.Fatal("second redo must insert only the row")
	}
	for _, mode := range []string{"source", "split", "preview"} {
		history := buffer.History()
		if !a.setMode(tab, mode) {
			t.Fatalf("mode %s refused after completed row action: %s", mode, a.Message)
		}
		tester.Frame()
		tester.Frame()
		if tab.Mode != mode || tab.Doc.Buffer != buffer || buffer.String() != inserted || buffer.History() != history {
			t.Fatal("mode switch changed row source/history/Buffer", mode)
		}
	}

	empty := requireMarkdownCellAccess(t, tester, ui.RoleCell, "数据，第 3 行，第 1 列：（空单元格）")
	openMarkdownCellPointer(t, tester, empty, "编辑单元格 · 数据 · 第 3 行 · 第 1 列")
	activateMarkdownRow(t, tester, markdownRowDelete, true)
	if buffer.String() != cellOnly || buffer.History().UndoDepth != 3 {
		t.Fatalf("native Space deletion did not remove exactly the selected body row: %q", buffer.String())
	}
	tab.Editor.Command("undo")
	if buffer.String() != inserted {
		t.Fatal("body deletion undo failed")
	}
	tab.Editor.Command("redo")
	if buffer.String() != cellOnly {
		t.Fatal("body deletion redo failed")
	}
	if !bytes.Equal(mustReadFile(t, path), markdownTablesFixture()) {
		t.Fatal("row actions or mode switches wrote to disk before Save")
	}
	if !a.Save(tab, false) {
		t.Fatal("row-edited document Save failed", a.Message)
	}
	wantDisk := append([]byte{0xef, 0xbb, 0xbf}, []byte(cellOnly)...)
	if !bytes.Equal(mustReadFile(t, path), wantDisk) || tab.Doc.Encoding != document.UTF8 || !tab.Doc.BOM || tab.Doc.Dirty() {
		t.Fatal("row Save changed exact BOM/CRLF source or document metadata")
	}

	reopened := newTestApp(t, filepath.Join(t.TempDir(), "config"))
	reopened.Open(path)
	rtab := reopened.Current()
	rtester := ui.NewTester(reopened.View, 1180, 780)
	rtester.Frame()
	if rtab == nil || rtab.Adapter == nil || rtab.Adapter.ID != "markdown" || rtab.Doc.Buffer.String() != cellOnly ||
		rtab.Doc.Encoding != document.UTF8 || !rtab.Doc.BOM || rtab.Doc.Dirty() {
		t.Fatal("saved row result failed registered-adapter reopen")
	}
	requireMarkdownCellAccess(t, rtester, ui.RoleCell, "数据，第 2 行，第 1 列：改<&🙂")
	version, selection, history := rtab.Doc.Buffer.Version(), rtab.Doc.Buffer.Selection(), rtab.Doc.Buffer.History()
	// This is the registered View.Export reference path, not a claim that the
	// asynchronously dispatched repository worker has run in this test.
	html, err := rtab.View.Export("html")
	wantTable := "<table>\n<thead>\n<tr><th style=\"text-align:left\">名称</th><th style=\"text-align:right\">说明</th></tr>\n</thead>\n<tbody>\n" +
		"<tr><td style=\"text-align:left\">改&lt;&amp;🙂</td><td style=\"text-align:right\">**字面**</td></tr>\n" +
		"<tr><td style=\"text-align:left\">🙂</td><td style=\"text-align:right\">[链接](https://example.com)</td></tr>\n</tbody>\n</table>\n"
	if err != nil || !bytes.Contains(html, []byte(wantTable)) || bytes.Count(html, []byte("<tr>")) != 3 ||
		!bytes.Contains(html, []byte("<p>表格前的正文</p>")) || !bytes.Contains(html, []byte("<p>表格后的正文</p>")) ||
		bytes.Contains(html, []byte("href=\"https://example.com")) || bytes.Contains(html, []byte("<strong>字面")) {
		t.Fatalf("row HTML literal/escaped/order oracle failed: %v\n%s", err, html)
	}
	if rtab.Doc.Buffer.Version() != version || rtab.Doc.Buffer.Selection() != selection || rtab.Doc.Buffer.History() != history ||
		rtab.Doc.Buffer.String() != cellOnly || !bytes.Equal(mustReadFile(t, path), wantDisk) || !rtab.Doc.BOM || rtab.Doc.Encoding != document.UTF8 {
		t.Fatal("reference export/reopen changed source, history, encoding or disk")
	}
}

func TestMarkdownRowsRealAdapterStaleRecoveryAndPostCellRefusal(t *testing.T) {
	t.Run("stale-table-recover", func(t *testing.T) {
		a, tab, tester, path := openRealMarkdownTables(t, 1180, 780)
		body := requireMarkdownCellAccess(t, tester, ui.RoleCell, markdownCellBodyName)
		openMarkdownCellPointer(t, tester, body, markdownCellBodyEditor)
		replaceMarkdownCellDraft(t, tester, markdownCellBodyEditor, "草稿🙂")
		start := text.UTF16Len(markdownTablesBody[:strings.Index(markdownTablesBody, "**字面**")])
		if err := tab.Doc.Buffer.ReplaceUTF16(start, start+text.UTF16Len("**字面**"), "外部"); err != nil {
			t.Fatal(err)
		}
		// The replacement covers exactly six UTF-16 units, irrespective of
		// UTF-8 byte length. Keep the expected source independent of the planner.
		want := strings.Replace(markdownTablesBody, "**字面**", "外部", 1)
		if tab.Doc.Buffer.String() != want {
			t.Fatalf("stale-source fixture range was wrong: %q", tab.Doc.Buffer.String())
		}
		tester.Frame()
		version, history, view := tab.Doc.Buffer.Version(), tab.Doc.Buffer.History(), tab.View
		activateMarkdownRow(t, tester, markdownRowDelete, false)
		if tab.Doc.Buffer.String() != want || tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.History() != history ||
			tab.View != view || tab.Mode != "preview" || !strings.Contains(a.Message, "行未修改") {
			t.Fatalf("stale row action changed source or hid refusal: %q", a.Message)
		}
		requireMarkdownCellEditor(t, tester, markdownCellBodyEditor)
		if a.Save(tab, false) || a.setMode(tab, "split") {
			t.Fatal("conflicting row draft escaped the existing Save/mode barriers")
		}
		if !bytes.Equal(mustReadFile(t, path), markdownTablesFixture()) {
			t.Fatal("stale row/save refusal changed original disk bytes")
		}
		button := requireMarkdownRowButton(t, tester, markdownCellRecoverLabel, false)
		tester.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
		tester.Frame()
		recovered := a.Current()
		if len(a.Tabs) != 2 || recovered == tab || !recovered.Doc.IsNew || recovered.Doc.Path != "" ||
			recovered.Doc.Buffer.String() != "草稿🙂" || tab.Doc.Buffer.String() != want || tab.Doc.Buffer.History() != history {
			t.Fatal("row refusal recovery failed to preserve literal draft and original source")
		}
	})
	t.Run("cell-written-row-limit", func(t *testing.T) {
		source := "| 名称 | 说明 |\r\n| :--- | ---: |\r\n| 甲 | **字面** |\r\n" + strings.Repeat("| x | y |\r\n", 255)
		a := newTestApp(t, filepath.Join(t.TempDir(), "config"))
		tab := a.add(document.New(source), "row-limit.md", "")
		tester := ui.NewTester(a.View, 1180, 780)
		tester.Frame()
		body := requireMarkdownCellAccess(t, tester, ui.RoleCell, markdownCellBodyName)
		openMarkdownCellPointer(t, tester, body, markdownCellBodyEditor)
		replaceMarkdownCellDraft(t, tester, markdownCellBodyEditor, "已写🙂")
		activateMarkdownRow(t, tester, markdownRowInsert, false)
		want := strings.Replace(source, "| 甲 |", "| 已写🙂 |", 1)
		if tab.Doc.Buffer.String() != want || tab.Doc.Buffer.History().UndoDepth != 1 ||
			!strings.Contains(a.Message, "单元格已写回，行未修改") || !strings.Contains(a.Message, "上限") {
			t.Fatalf("post-cell row refusal lost its committed-cell distinction: %q", a.Message)
		}
		if _, ok := tester.AccessByRole(ui.RoleTextField, markdownCellBodyEditor); ok {
			t.Fatal("successful cell write still appears staged")
		}
		tab.Editor.Command("undo")
		if tab.Doc.Buffer.String() != source {
			t.Fatal("successful own cell write was not independently undoable")
		}
	})
}

func TestMarkdownRowsAppNarrowScreenshots(t *testing.T) {
	out := os.Getenv("LEVNOTE_MARKDOWN_ROWS_SCREENSHOT_DIR")
	if out == "" {
		t.Skip("set LEVNOTE_MARKDOWN_ROWS_SCREENSHOT_DIR for integrated row screenshots")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	for _, dark := range []bool{false, true} {
		name := "staged-light"
		if dark {
			name = "refused-dark"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			workspace := filepath.Join(root, "workspace")
			if err := os.Mkdir(workspace, 0755); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(workspace, "rows.md")
			original := append([]byte{0xef, 0xbb, 0xbf}, []byte(markdownTablesScreenshotBody)...)
			if err := os.WriteFile(path, original, 0600); err != nil {
				t.Fatal(err)
			}
			a := newTestApp(t, filepath.Join(root, "config"))
			a.Root, a.Dark = workspace, dark
			a.refreshTreeNow()
			a.Open(path)
			tab := a.Current()
			if tab == nil || tab.Adapter == nil || tab.Adapter.ID != "markdown" || !a.setMode(tab, "split") {
				t.Fatal("screenshot requires the registered split Markdown view")
			}
			tester := ui.NewTester(a.View, 760, 480)
			tester.SetDark(dark)
			tester.Frame()
			body := requireMarkdownCellAccess(t, tester, ui.RoleCell, markdownCellBodyName)
			openMarkdownCellPointer(t, tester, body, markdownCellBodyEditor)
			value := "阶段🙂"
			if dark {
				value = "坏|值"
			}
			replaceMarkdownCellDraft(t, tester, markdownCellBodyEditor, value)
			if dark {
				activateMarkdownRow(t, tester, markdownRowDelete, false)
				if !strings.Contains(a.Message, "行未修改") {
					t.Fatal("refused screenshot lacks visible row refusal", a.Message)
				}
			} else {
				tester.Type(value) // confirm native composition, retain staged cell
				tester.Key(0, ui.KeyF6)
				if !requireMarkdownRowButton(t, tester, markdownRowInsert, false).Focused {
					t.Fatal("integrated screenshot keyboard route is unusable")
				}
			}
			tester.Frame()
			imagePath := filepath.Join(out, "markdown-rows-"+name+"-760x480.png")
			// Preserve the actual failing layout too, before geometry assertions.
			captureAcceptanceTester(t, imagePath, tester)
			labels := []string{markdownRowInsert, markdownRowDelete, markdownCellFinishLabel, markdownCellDiscardLabel}
			if dark {
				labels = append(labels, markdownCellRecoverLabel)
			}
			for _, label := range labels {
				button := requireMarkdownRowButton(t, tester, label, false)
				r := button.Bounds
				if r.X < 0 || r.Y < 0 || r.X+r.W > 760 || r.Y+r.H > 480-29 {
					t.Fatalf("integrated row control %q overlaps chrome/status: %+v capture=%s", label, r, imagePath)
				}
			}
			if tab.Doc.Buffer.String() != markdownTablesScreenshotBody || tab.Doc.Dirty() || !bytes.Equal(mustReadFile(t, path), original) {
				t.Fatal("staged/refused screenshot changed the host document or original bytes")
			}
			if dark {
				button := requireMarkdownRowButton(t, tester, markdownCellRecoverLabel, false)
				tester.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
				tester.Frame()
				if len(a.Tabs) != 2 || a.Current() == tab || a.Current().Doc.Buffer.String() != value {
					t.Fatal("visible recovery control could not recover the exact row draft")
				}
			} else {
				button := requireMarkdownRowButton(t, tester, markdownCellFinishLabel, false)
				tester.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
				tester.Frame()
				want := strings.Replace(markdownTablesScreenshotBody, "| 甲 |", "| "+value+" |", 1)
				if tab.Doc.Buffer.String() != want || tab.Doc.Buffer.History().UndoDepth != 1 {
					t.Fatal("composition confirmation duplicated or lost staged text before the screenshot", tab.Doc.Buffer.String())
				}
			}
		})
	}
}
