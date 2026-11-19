package markdownview

import (
	"errors"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/markdown"
	"levnote.local/native/ui/editor"
)

func tableCellAccess(t *testing.T, tester *ui.Tester, role ui.Role, name string) ui.AccessInfo {
	t.Helper()
	info, ok := tester.AccessByRole(role, name)
	if !ok {
		t.Fatalf("cell %q role %v missing: texts=%q", name, role, tester.Texts())
	}
	if info.Disabled || !info.Focusable || info.Offscreen ||
		info.Bounds.X < 0 || info.Bounds.Y < 0 || info.Bounds.W <= 0 || info.Bounds.H <= 0 {
		t.Fatalf("cell %q state/bounds = %#v", name, info)
	}
	if info.Description != tableCellActionHint || len(info.Name) > tableTooltipBytes || !utf8.ValidString(info.Name) {
		t.Fatalf("cell %q accessibility = %#v", name, info)
	}
	return info
}

func doubleClickAccess(t *testing.T, tester *ui.Tester, info ui.AccessInfo) {
	t.Helper()
	x, y := info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2
	tester.ClickAt(x, y)
	tester.Frame()
	tester.ClickAt(x, y)
	tester.Frame()
}

func focusAccessByTab(t *testing.T, tester *ui.Tester, role ui.Role, name string) ui.AccessInfo {
	t.Helper()
	for attempt := 0; attempt < 64; attempt++ {
		info, ok := tester.AccessByRole(role, name)
		if ok && info.Focused {
			return info
		}
		tester.Key(0, ui.KeyTab)
		tester.Frame()
	}
	info, _ := tester.AccessByRole(role, name)
	t.Fatalf("cell %q never received Tab focus: %#v", name, info)
	return ui.AccessInfo{}
}

func openCellByPointer(t *testing.T, tester *ui.Tester, role ui.Role, name string) ui.AccessInfo {
	t.Helper()
	info := tableCellAccess(t, tester, role, name)
	doubleClickAccess(t, tester, info)
	return info
}

func replaceDraftText(t *testing.T, tester *ui.Tester, value string) {
	t.Helper()
	tester.Command("selectAll")
	tester.Compose(value, utf8.RuneCountInString(value))
	tester.Frame()
}

func clickDraftControl(t *testing.T, tester *ui.Tester, label string) {
	t.Helper()
	info, ok := tester.AccessByRole(ui.RoleButton, label)
	if !ok || info.Disabled || info.Offscreen {
		t.Fatalf("draft control %q unavailable: %#v texts=%q", label, info, tester.Texts())
	}
	tester.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	tester.Frame()
}

func TestTableCellNativePointerKeyboardStackedAndSecondTable(t *testing.T) {
	t.Run("pointer-nonzero-source-offset", func(t *testing.T) {
		source := "# 导语\r\n\r\n" + tableViewSource
		buffer := text.New(source)
		v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
		tester := ui.NewTester(v.Draw, 760, 480)
		tester.Frame()
		openCellByPointer(t, tester, ui.RoleCell, "数据，第 2 行，第 1 列：甲")
		if v.draft == nil || v.cellEdit == nil || v.draft.Buffer.String() != "甲" {
			t.Fatalf("pointer did not open exact cell: %#v", v.draft)
		}
		wantStart := text.UTF16Len(source[:strings.Index(source, "| 甲 |")+2])
		if v.editSpan.Start() != wantStart || v.editSpan.End() != wantStart+1 || buffer.String() != source {
			t.Fatalf("cell span=%+v want=%d hostChanged=%v", v.editSpan, wantStart, buffer.String() != source)
		}
		if !tester.HasText(tableCellStagedNotice) || !tester.HasText("放弃草稿") {
			t.Fatalf("staged controls/notice missing: %q", tester.Texts())
		}
		notice, ok := tester.Access(tableCellStagedNotice)
		if !ok || notice.Description != tableCellSessionHint {
			t.Fatalf("cell staging recovery-scope tooltip = %#v found=%v", notice, ok)
		}
		if _, ok := tester.AccessByRole(ui.RoleColumnHeader, "表头，第 1 行，第 1 列：名称"); !ok {
			t.Fatal("cell editing suppressed the immutable table preview context")
		}
	})

	t.Run("header-enter-space-and-body-enter", func(t *testing.T) {
		for _, target := range []struct {
			role ui.Role
			name string
			want string
		}{
			{ui.RoleColumnHeader, "表头，第 1 行，第 2 列：备注", "备注"},
			{ui.RoleCell, "数据，第 2 行，第 1 列：甲", "甲"},
		} {
			t.Run(target.want, func(t *testing.T) {
				buffer := text.New(tableViewSource)
				v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
				tester := ui.NewTester(v.Draw, 760, 480)
				tester.Frame()
				focusAccessByTab(t, tester, target.role, target.name)
				tester.Key(0, ui.KeySpace)
				tester.Frame()
				if v.draft != nil {
					t.Fatal("Space opened the cell editor instead of scrolling")
				}
				tester.Key(0, ui.KeyEnter)
				tester.Frame()
				if v.draft == nil || v.cellEdit == nil || v.draft.Buffer.String() != target.want {
					t.Fatalf("Enter did not open %q: %#v", target.name, v.draft)
				}
			})
		}
	})

	t.Run("stacked-empty", func(t *testing.T) {
		buffer := text.New(tableViewSource)
		v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
		tester := ui.NewTester(v.Draw, 265, 480)
		tester.Frame()
		tester.Frame()
		if tableRowUsableForGrid(v, 2) {
			t.Fatal("narrow fixture unexpectedly used the grid")
		}
		openCellByPointer(t, tester, ui.RoleCell, "数据，第 3 行，第 2 列："+tableEmptyCell)
		if v.draft == nil || v.cellEdit == nil || v.draft.Buffer.String() != "" ||
			v.editSpan.Start() != v.editSpan.End() {
			t.Fatalf("empty stacked cell did not open its zero-width field: %#v span=%+v", v.draft, v.editSpan)
		}
	})

	t.Run("second-duplicate-table", func(t *testing.T) {
		source := "前言\r\n\r\n" + tableViewSource + "\r\n\r\n中段\r\n\r\n" + tableViewSource + "\r\n\r\n结尾\r\n"
		buffer := text.New(source)
		v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
		tester := ui.NewTester(v.Draw, 265, 1600)
		tester.Frame()
		tester.Frame()
		firstLead, ok := tester.Find("前言")
		if !ok {
			t.Fatal("first duplicate-table lead missing")
		}
		secondLead, ok := tester.Find("中段")
		if !ok {
			t.Fatal("second duplicate-table lead missing")
		}
		firstCell := tableCellAccess(t, tester, ui.RoleCell, "数据，第 2 行，第 1 列：甲")
		x := firstCell.Bounds.X + firstCell.Bounds.W/2
		y := firstCell.Bounds.Y + firstCell.Bounds.H/2 + secondLead.Y - firstLead.Y
		if x < 0 || x >= 265 || y < 0 || y >= 1600 {
			t.Fatalf("second duplicate-table cell pointer %.1f:%.1f is offscreen", x, y)
		}
		tester.ClickAt(x, y)
		tester.Frame()
		tester.ClickAt(x, y)
		tester.Frame()
		if v.draft == nil || v.draft.Buffer.String() != "甲" {
			t.Fatalf("second duplicate table did not open its body cell: %#v", v.draft)
		}
		second := strings.LastIndex(source, tableViewSource)
		wantStart := text.UTF16Len(source[:second+strings.Index(tableViewSource, "| 甲 |")+2])
		if v.editSpan.Start() != wantStart || v.editSpan.End() != wantStart+1 {
			t.Fatalf("second table span=%+v want=%d", v.editSpan, wantStart)
		}
	})
}

func TestTableCellStagedFlushNoopAndDiscard(t *testing.T) {
	source := "# 导语\r\n\r\n" + tableViewSource
	buffer := text.New(source)
	changes := 0
	v := Adapter().Create(editor.Host{
		Buffer: buffer, Changed: func() { changes++ },
	}).(*View)
	tester := ui.NewTester(v.Draw, 760, 480)
	tester.Frame()
	openCellByPointer(t, tester, ui.RoleCell, "数据，第 2 行，第 1 列：甲")
	version, hostSelection := buffer.Version(), buffer.Selection()
	replaceDraftText(t, tester, "乙🙂")
	if buffer.String() != source || buffer.Version() != version || buffer.Selection() != hostSelection ||
		changes != 0 || buffer.CanUndo() || !v.pendingDraft() {
		t.Fatalf("local cell input reached host: version=%d history=%#v changes=%d", buffer.Version(), buffer.History(), changes)
	}
	clickDraftControl(t, tester, "完成编辑")
	want := strings.Replace(source, "| 甲 |", "| 乙🙂 |", 1)
	if buffer.String() != want || changes != 1 || buffer.History().UndoDepth != 1 ||
		buffer.History().RedoDepth != 0 || v.draft != nil || v.cellEdit != nil {
		t.Fatalf("cell commit = source %q changes=%d history=%#v draft=%#v", buffer.String(), changes, buffer.History(), v.draft)
	}
	if !buffer.Undo() || buffer.String() != source {
		t.Fatal("one undo did not restore the exact original cell")
	}

	t.Run("noop", func(t *testing.T) {
		buffer := text.New(tableViewSource)
		changes := 0
		v := Adapter().Create(editor.Host{Buffer: buffer, Changed: func() { changes++ }}).(*View)
		tester := ui.NewTester(v.Draw, 760, 480)
		tester.Frame()
		openCellByPointer(t, tester, ui.RoleCell, "数据，第 2 行，第 1 列：甲")
		version := buffer.Version()
		clickDraftControl(t, tester, "完成编辑")
		if v.draft != nil || buffer.String() != tableViewSource || buffer.Version() != version ||
			buffer.CanUndo() || changes != 0 {
			t.Fatal("no-op cell Flush wrote source, history, or Changed")
		}
	})

	t.Run("discard", func(t *testing.T) {
		buffer := text.New(tableViewSource)
		v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
		tester := ui.NewTester(v.Draw, 760, 480)
		tester.Frame()
		openCellByPointer(t, tester, ui.RoleCell, "数据，第 2 行，第 1 列：甲")
		replaceDraftText(t, tester, "不要写回")
		version, selection, history := buffer.Version(), buffer.Selection(), buffer.History()
		clickDraftControl(t, tester, "放弃草稿")
		if v.draft != nil || v.cellEdit != nil || buffer.String() != tableViewSource ||
			buffer.Version() != version || buffer.Selection() != selection || buffer.History() != history {
			t.Fatal("explicit discard changed host state")
		}
	})
}

func TestTableCellEmptyCommitAndSourceIdentityRefusals(t *testing.T) {
	t.Run("empty-stacked-cell-insert", func(t *testing.T) {
		buffer := text.New(tableViewSource)
		changes := 0
		v := Adapter().Create(editor.Host{
			Buffer: buffer, Changed: func() { changes++ },
		}).(*View)
		tester := ui.NewTester(v.Draw, 265, 480)
		tester.Frame()
		tester.Frame()
		openCellByPointer(t, tester, ui.RoleCell, "数据，第 3 行，第 2 列："+tableEmptyCell)
		if v.editSpan.Start() != v.editSpan.End() || v.draft.Buffer.String() != "" {
			t.Fatal("empty cell did not open at its parser right-edge anchor")
		}
		replaceDraftText(t, tester, "新增🙂")
		if buffer.String() != tableViewSource || buffer.CanUndo() || changes != 0 {
			t.Fatal("staged empty-cell insertion reached the host before commit")
		}
		clickDraftControl(t, tester, "完成编辑")
		// The empty field's parser anchor is after its existing two spaces, so
		// insertion preserves those bytes and lands immediately before the
		// closing delimiter rather than inventing trailing padding.
		want := strings.Replace(tableViewSource, "|  |\r\n", "|  新增🙂|\r\n", 1)
		if buffer.String() != want || changes != 1 || buffer.History().UndoDepth != 1 ||
			buffer.History().RedoDepth != 0 || v.draft != nil || v.cellEdit != nil {
			t.Fatalf("empty-cell commit source=%q changes=%d history=%#v draft=%#v",
				buffer.String(), changes, buffer.History(), v.draft)
		}
		if !buffer.Undo() || buffer.String() != tableViewSource {
			t.Fatal("empty-cell commit was not exactly one undo entry")
		}
	})

	t.Run("table-32k-bound", func(t *testing.T) {
		source := tableCell32KiBBoundarySource(t)
		buffer := text.New(source)
		v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
		tester := ui.NewTester(v.Draw, 760, 480)
		tester.Frame()
		openCellByPointer(t, tester, ui.RoleColumnHeader, "表头，第 1 行，第 1 列：a")
		draft := v.draft
		value := strings.Repeat("b", tableCellMaxDraftBytes)
		if err := draft.Buffer.ReplaceUTF16(0, draft.Buffer.LenUTF16(), value); err != nil {
			t.Fatal(err)
		}
		version, history := buffer.Version(), buffer.History()
		err := v.Flush()
		if !errors.Is(err, errTableCellTableSize) || v.draft != draft ||
			draft.Buffer.String() != value || buffer.Version() != version ||
			buffer.History() != history || buffer.String() != source {
			t.Fatalf("32 KiB refusal err=%v retained=%v local=%d host=%d history=%#v",
				err, v.draft == draft, draft.Buffer.LenBytes(), buffer.LenBytes(), buffer.History())
		}
	})

	t.Run("document-parse-bound", func(t *testing.T) {
		table := "|a|\n|---|\n"
		overhead := len(table) + len("\n\n```\n") + len("\n```\n")
		source := table + "\n\n```\n" +
			strings.Repeat("x", markdown.MaxParseBytes-overhead) + "\n```\n"
		if len(source) != markdown.MaxParseBytes {
			t.Fatalf("document-bound fixture bytes=%d want=%d", len(source), markdown.MaxParseBytes)
		}
		doc := markdown.Parse(source)
		if doc.Partial {
			t.Fatalf("document-bound fixture is unexpectedly Partial: %#v", doc.Warnings)
		}
		buffer := text.New(source)
		v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
		tester := ui.NewTester(v.Draw, 760, 480)
		tester.Frame()
		openCellByPointer(t, tester, ui.RoleColumnHeader, "表头，第 1 行，第 1 列：a")
		draft := v.draft
		if err := draft.Buffer.ReplaceUTF16(0, draft.Buffer.LenUTF16(), "abcd"); err != nil {
			t.Fatal(err)
		}
		version, history := buffer.Version(), buffer.History()
		err := v.Flush()
		if !errors.Is(err, errTableCellDocumentSize) || v.draft != draft ||
			draft.Buffer.String() != "abcd" || buffer.Version() != version ||
			buffer.History() != history || buffer.LenBytes() != markdown.MaxParseBytes {
			t.Fatalf("document-bound refusal err=%v draft=%v hostBytes=%d history=%#v",
				err, v.draft != nil, buffer.LenBytes(), buffer.History())
		}
	})
}

func tableCell32KiBBoundarySource(t *testing.T) string {
	t.Helper()
	var source strings.Builder
	source.WriteString("|a|\n|---|\n")
	for index := 0; index < 7; index++ {
		source.WriteByte('|')
		source.WriteString(strings.Repeat("x", 4096))
		source.WriteString("|\n")
	}
	source.WriteByte('|')
	source.WriteString(strings.Repeat("y", 4045))
	source.WriteString("|\n")
	value := source.String()
	remaining := maxPreviewBlockBytes - len(value)
	if remaining <= 0 || remaining >= tableCellMaxDraftBytes {
		t.Fatalf("32 KiB boundary fixture bytes=%d leaves unusable headroom=%d", len(value), remaining)
	}
	doc := markdown.Parse(value)
	if doc.Partial || len(doc.Blocks) != 1 || doc.Blocks[0].Kind != markdown.BlockTable ||
		doc.Blocks[0].Table.Problem != "" || len(doc.Blocks[0].Raw) != len(value) {
		t.Fatalf("32 KiB fixture is not one complete table: %#v", doc.Blocks)
	}
	return value
}

func TestTableCellFailedFlushRetainsAndCorrects(t *testing.T) {
	buffer := text.New(tableViewSource)
	var reports []string
	v := Adapter().Create(editor.Host{
		Buffer: buffer,
		Report: func(err error) { reports = append(reports, err.Error()) },
	}).(*View)
	tester := ui.NewTester(v.Draw, 760, 480)
	tester.Frame()
	openCellByPointer(t, tester, ui.RoleCell, "数据，第 2 行，第 1 列：甲")
	replaceDraftText(t, tester, "坏|值")
	draft := v.draft
	localSelection, version, hostSelection, history := draft.Buffer.Selection(), buffer.Version(), buffer.Selection(), buffer.History()
	clickDraftControl(t, tester, "完成编辑")
	if v.draft != draft || draft.Buffer.Selection() != localSelection || buffer.String() != tableViewSource ||
		buffer.Version() != version || buffer.Selection() != hostSelection || buffer.History() != history {
		t.Fatal("failed cell Flush lost draft/selection or changed host")
	}
	if !errors.Is(v.draftError, errTableCellPipe) || len(reports) == 0 ||
		!tester.HasText(tableCellStagedNotice) || !tester.HasText("放弃草稿") ||
		!tester.HasText("打开未同步副本") {
		t.Fatalf("failed Flush state=%v reports=%#v texts=%q", v.draftError, reports, tester.Texts())
	}
	v.draft.Focus()
	tester.Frame()
	replaceDraftText(t, tester, "修正🙂")
	clickDraftControl(t, tester, "完成编辑")
	want := strings.Replace(tableViewSource, "| 甲 |", "| 修正🙂 |", 1)
	if v.draft != nil || buffer.String() != want {
		t.Fatalf("corrected local draft did not commit: source=%q draft=%q error=%v", buffer.String(), draft.Buffer.String(), v.draftError)
	}
}

func TestTableCellReadonlyAndHeldIdentity(t *testing.T) {
	t.Run("readonly", func(t *testing.T) {
		buffer := text.New(tableViewSource)
		var reports []string
		v := Adapter().Create(editor.Host{
			Buffer: buffer, ReadOnly: true,
			Report: func(err error) { reports = append(reports, err.Error()) },
		}).(*View)
		tester := ui.NewTester(v.Draw, 760, 480)
		tester.Frame()
		openCellByPointer(t, tester, ui.RoleCell, "数据，第 2 行，第 1 列：甲")
		if v.draft != nil || v.cellEdit != nil || buffer.String() != tableViewSource ||
			buffer.CanUndo() || len(reports) == 0 || !strings.Contains(strings.Join(reports, "\n"), "只读") {
			t.Fatalf("readonly cell activation draft=%#v reports=%#v", v.draft, reports)
		}
	})

	for _, interveningFrame := range []bool{false, true} {
		t.Run(fmt.Sprintf("rebase/frame=%v", interveningFrame), func(t *testing.T) {
			source := "# 导语\n\n" + tableViewSource
			buffer := text.New(source)
			v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
			tester := ui.NewTester(v.Draw, 760, 480)
			tester.Frame()
			info := tableCellAccess(t, tester, ui.RoleCell, "数据，第 2 行，第 1 列：甲")
			x, y := info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2
			tester.ClickAt(x, y)
			tester.Frame()
			tester.Press(x, y)
			prefix := "前置\r\n\r\n"
			if err := buffer.ReplaceUTF16(0, 0, prefix); err != nil {
				t.Fatal(err)
			}
			if interveningFrame {
				tester.Frame()
			}
			tester.Release(x, y)
			tester.Frame()
			if v.draft == nil || v.draft.Buffer.String() != "甲" {
				t.Fatalf("insertion before table did not rebase held cell: %#v", v.draft)
			}
			want := text.UTF16Len(prefix + source[:strings.Index(source, "| 甲 |")+2])
			if v.editSpan.Start() != want || v.editSpan.End() != want+1 {
				t.Fatalf("rebased cell span=%+v want=%d", v.editSpan, want)
			}
		})
	}

	for _, interveningFrame := range []bool{false, true} {
		t.Run(fmt.Sprintf("same-table/frame=%v", interveningFrame), func(t *testing.T) {
			buffer := text.New(tableViewSource)
			var reports []string
			v := Adapter().Create(editor.Host{
				Buffer: buffer,
				Report: func(err error) { reports = append(reports, err.Error()) },
			}).(*View)
			tester := ui.NewTester(v.Draw, 760, 480)
			tester.Frame()
			info := tableCellAccess(t, tester, ui.RoleCell, "数据，第 2 行，第 1 列：甲")
			x, y := info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2
			tester.ClickAt(x, y)
			tester.Frame()
			tester.Press(x, y)
			replaceTableBufferText(t, buffer, "**字面**", "外部")
			if interveningFrame {
				tester.Frame()
			}
			tester.Release(x, y)
			tester.Frame()
			if v.draft != nil || len(reports) == 0 || !strings.Contains(strings.Join(reports, "\n"), "单元格") {
				t.Fatalf("same-table held edit draft=%#v reports=%#v", v.draft, reports)
			}
		})
	}
}

func TestTableCellPointerCancelAndReleaseClearGesture(t *testing.T) {
	for _, end := range []string{"cancel", "release-view"} {
		t.Run(end, func(t *testing.T) {
			buffer := text.New(tableViewSource)
			var reports []string
			v := Adapter().Create(editor.Host{
				Buffer: buffer,
				Report: func(err error) { reports = append(reports, err.Error()) },
			}).(*View)
			tester := ui.NewTester(v.Draw, 760, 480)
			tester.Frame()
			cell := tableCellAccess(t, tester, ui.RoleCell, "数据，第 2 行，第 1 列：甲")
			x, y := cell.Bounds.X+cell.Bounds.W/2, cell.Bounds.Y+cell.Bounds.H/2
			tester.ClickAt(x, y)
			tester.Frame()
			tester.Press(x, y)
			if v.tableCellActivation.buffer != buffer {
				t.Fatal("second pointer down did not capture the cell")
			}
			if end == "cancel" {
				tester.CancelPress()
				tester.Frame()
				if v.draft != nil || v.cellEdit != nil || v.tableCellActivation.buffer != nil ||
					len(reports) != 0 || buffer.String() != tableViewSource || buffer.CanUndo() {
					t.Fatalf("canceled cell gesture changed state: reports=%#v", reports)
				}
			} else {
				v.Release()
				if v.draft != nil || v.cellEdit != nil || v.tableCellActivation.buffer != nil ||
					buffer.String() != tableViewSource || buffer.CanUndo() {
					t.Fatal("Release retained a cell gesture or changed the host")
				}
			}
		})
	}
}

func TestTableCellHeldBufferGroupExpiryPartialAndReleaseRecovery(t *testing.T) {
	pressBodyCell := func(t *testing.T) (*text.Buffer, *View, *ui.Tester) {
		t.Helper()
		buffer := text.New(tableViewSource)
		var reports []string
		v := Adapter().Create(editor.Host{
			Buffer: buffer,
			Report: func(err error) { reports = append(reports, err.Error()) },
		}).(*View)
		tester := ui.NewTester(v.Draw, 760, 480)
		tester.Frame()
		cell := tableCellAccess(t, tester, ui.RoleCell, "数据，第 2 行，第 1 列：甲")
		x, y := cell.Bounds.X+cell.Bounds.W/2, cell.Bounds.Y+cell.Bounds.H/2
		tester.ClickAt(x, y)
		tester.Frame()
		tester.Press(x, y)
		if v.tableCellActivation.buffer != buffer {
			t.Fatal("actual second pointer-down did not capture the cell")
		}
		return buffer, v, tester
	}

	t.Run("replaced-buffer", func(t *testing.T) {
		oldBuffer, v, tester := pressBodyCell(t)
		var reports []string
		v.host.Report = func(err error) { reports = append(reports, err.Error()) }
		replacement := text.New(tableViewSource)
		v.host.Buffer = replacement
		tester.Release(400, 200)
		tester.Frame()
		if v.draft != nil || v.tableCellActivation.buffer != nil || replacement.String() != tableViewSource ||
			replacement.CanUndo() || oldBuffer.CanUndo() || len(reports) == 0 ||
			!strings.Contains(strings.Join(reports, "\n"), "单元格") {
			t.Fatalf("replaced buffer refusal draft=%#v old=%v replacement=%v reports=%#v",
				v.draft, oldBuffer.CanUndo(), replacement.CanUndo(), reports)
		}
	})

	t.Run("group", func(t *testing.T) {
		buffer, v, tester := pressBodyCell(t)
		var reports []string
		v.host.Report = func(err error) { reports = append(reports, err.Error()) }
		if err := buffer.BeginGroup(); err != nil {
			t.Fatal(err)
		}
		if err := buffer.ReplaceUTF16(0, 0, "组内前缀\r\n"); err != nil {
			t.Fatal(err)
		}
		tester.Release(400, 200)
		if v.draft != nil || v.tableCellActivation.buffer != nil || len(reports) == 0 ||
			!strings.Contains(strings.Join(reports, "\n"), "单元格") {
			t.Fatalf("open group refusal draft=%#v reports=%#v", v.draft, reports)
		}
		if !buffer.CancelGroup() || buffer.String() != tableViewSource {
			t.Fatal("group fixture was not restored exactly")
		}
	})

	t.Run("expired-journal", func(t *testing.T) {
		buffer, v, tester := pressBodyCell(t)
		var reports []string
		v.host.Report = func(err error) { reports = append(reports, err.Error()) }
		for index := 0; index < 65; index++ {
			if err := buffer.ReplaceUTF16(0, 0, "x"); err != nil {
				t.Fatal(err)
			}
		}
		tester.Release(400, 200)
		if v.draft != nil || v.tableCellActivation.buffer != nil || len(reports) == 0 ||
			!strings.Contains(strings.Join(reports, "\n"), "单元格") {
			t.Fatalf("expired journal refusal draft=%#v reports=%#v", v.draft, reports)
		}
	})

	t.Run("partial-current-document", func(t *testing.T) {
		buffer, v, tester := pressBodyCell(t)
		var reports []string
		v.host.Report = func(err error) { reports = append(reports, err.Error()) }
		if err := buffer.ReplaceUTF16(buffer.LenUTF16(), buffer.LenUTF16(), strings.Repeat("\n", 65537)); err != nil {
			t.Fatal(err)
		}
		tester.Release(400, 200)
		if v.draft != nil || v.tableCellActivation.buffer != nil || len(reports) == 0 ||
			!strings.Contains(strings.Join(reports, "\n"), "不完整") {
			t.Fatalf("partial document refusal draft=%#v reports=%#v", v.draft, reports)
		}
	})

	for _, mode := range []string{"invalid", "stale"} {
		t.Run("failed-release-recovery/"+mode, func(t *testing.T) {
			buffer := text.New(tableViewSource)
			var recoveredTitle, recoveredContent string
			var reports []string
			v := Adapter().Create(editor.Host{
				Buffer: buffer,
				Report: func(err error) { reports = append(reports, err.Error()) },
				RecoverDraft: func(title, content string) error {
					recoveredTitle, recoveredContent = title, content
					return nil
				},
			}).(*View)
			tester := ui.NewTester(v.Draw, 760, 480)
			tester.Frame()
			openCellByPointer(t, tester, ui.RoleCell, "数据，第 2 行，第 1 列：甲")
			localValue := "坏|值"
			if mode == "stale" {
				localValue = "本地🙂"
			}
			replaceDraftText(t, tester, localValue)
			if mode == "stale" {
				replaceTableBufferText(t, buffer, "**字面**", "外部")
			}
			draft := v.draft
			if err := draft.Flush(); err != nil {
				t.Fatal("failed to finish local input before Release fixture:", err)
			}
			tester.Frame()
			localSelection, localHistory := draft.Buffer.Selection(), draft.Buffer.History()
			hostText, hostVersion, hostSelection, hostHistory :=
				buffer.String(), buffer.Version(), buffer.Selection(), buffer.History()
			v.Release()
			if len(v.rows) != 0 || v.doc.Raw() != "" {
				t.Fatalf("%s failed Release retained parsed preview cache", mode)
			}
			tester.Frame()
			if v.draft != draft || draft.Buffer.String() != localValue ||
				draft.Buffer.Selection() != localSelection || draft.Buffer.History() != localHistory ||
				buffer.String() != hostText || buffer.Version() != hostVersion ||
				buffer.Selection() != hostSelection || buffer.History() != hostHistory {
				t.Fatalf("%s failed Release lost draft/local/host state: local=%q selection=%v/%v history=%#v/%#v",
					mode, draft.Buffer.String(), draft.Buffer.Selection(), localSelection,
					draft.Buffer.History(), localHistory)
			}
			if v.draftError == nil ||
				!tester.HasText("完成编辑") || !tester.HasText("放弃草稿") ||
				!tester.HasText("打开未同步副本") {
				t.Fatalf("%s failed Release controls error=%v texts=%q", mode, v.draftError, tester.Texts())
			}
			warnings := strings.Join(v.Warnings(), "\n")
			if !strings.Contains(warnings, "单元格草稿尚未写回") ||
				strings.Contains(warnings, "块草稿尚未写回") {
				t.Fatalf("%s warning prefix is not cell-specific: %q", mode, warnings)
			}
			clickDraftControl(t, tester, "打开未同步副本")
			if recoveredTitle != "Markdown-块草稿.txt" || recoveredContent != localValue ||
				v.draft != nil || buffer.String() != hostText || buffer.Version() != hostVersion ||
				buffer.History() != hostHistory || len(reports) == 0 {
				t.Fatalf("%s recovery title=%q content=%q draft=%#v hostChanged=%v reports=%#v",
					mode, recoveredTitle, recoveredContent, v.draft, buffer.String() != hostText, reports)
			}
		})
	}
}

func TestTableCellDraftByteCap(t *testing.T) {
	for _, size := range []int{tableCellMaxDraftBytes, tableCellMaxDraftBytes + 1} {
		t.Run(fmt.Sprintf("bytes-%d", size), func(t *testing.T) {
			source := "|a|\n|---|\n"
			buffer := text.New(source)
			v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
			tester := ui.NewTester(v.Draw, 760, 480)
			tester.Frame()
			openCellByPointer(t, tester, ui.RoleColumnHeader, "表头，第 1 行，第 1 列：a")
			value := strings.Repeat("x", size)
			draft := v.draft
			if err := draft.Buffer.ReplaceUTF16(0, draft.Buffer.LenUTF16(), value); err != nil {
				t.Fatal(err)
			}
			if !v.pendingDraft() {
				t.Fatal("direct local draft mutation was not pending")
			}
			err := v.Flush()
			if size == tableCellMaxDraftBytes {
				if err != nil || v.draft != nil || buffer.String() != "|"+value+"|\n|---|\n" {
					t.Fatalf("4096-byte cell commit err=%v draft=%#v sourceLen=%d", err, v.draft, buffer.LenBytes())
				}
			} else {
				if !errors.Is(err, errTableCellTooLarge) || v.draft != draft ||
					draft.Buffer.LenBytes() != size || buffer.String() != source ||
					buffer.CanUndo() {
					t.Fatalf("4097-byte cell refusal err=%v draft=%#v source=%q", err, v.draft, buffer.String())
				}
			}
		})
	}
}

func tableCellTargetForValidation(t *testing.T, source string, row, column int) tableCellTarget {
	t.Helper()
	buffer := text.New(source)
	doc := markdown.Parse(source)
	if doc.Partial || len(doc.Blocks) != 1 || doc.Blocks[0].Kind != markdown.BlockTable ||
		doc.Blocks[0].Table.Problem != "" {
		t.Fatalf("validation fixture is not one complete table: %#v", doc.Blocks)
	}
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	v.setPreview(doc)
	preview := previewSnapshot{doc: doc, rows: v.rows, buffer: buffer, version: buffer.Version()}
	for _, previewRow := range v.rows {
		if previewRow.tableRow != row {
			continue
		}
		target, ok := tableCellTargetOf(preview, previewRow, doc.Blocks[0], column)
		if !ok {
			t.Fatalf("cell %d:%d target unavailable", row, column)
		}
		target, ok = captureTableCellTarget(preview, target)
		if !ok {
			t.Fatalf("cell %d:%d source identity unavailable", row, column)
		}
		return target
	}
	t.Fatalf("table row %d unavailable", row)
	return tableCellTarget{}
}

func tableCellCandidate(target tableCellTarget, value string) string {
	fieldStart := target.cellRange.Start - target.tableRange.Start
	fieldEnd := target.cellRange.End - target.tableRange.Start
	return target.tableSource[:fieldStart] + value + target.tableSource[fieldEnd:]
}

func TestTableCellCandidateOracleExactBytes(t *testing.T) {
	t.Run("no-outer-leading-padding-growth", func(t *testing.T) {
		source := "a | b\n--- | ---\n| x | y |\n"
		target := tableCellTargetForValidation(t, source, 0, 0)
		wantCandidate := " x | b\n--- | ---\n| x | y |\n"
		if got := tableCellCandidate(target, " x"); got != wantCandidate {
			t.Fatalf("candidate = %q, want %q", got, wantCandidate)
		}
		if err := validateTableCellCandidate(target, " x", len(" x")-len(target.cellSource)); err != nil {
			t.Fatalf("padding-only target reclassification was refused: %v", err)
		}
	})

	t.Run("leading-pipe-to-empty", func(t *testing.T) {
		source := "|a\n|---\n"
		target := tableCellTargetForValidation(t, source, 0, 0)
		if err := validateTableCellCandidate(target, "", len("")-len(target.cellSource)); err != nil {
			t.Fatalf("valid |a to | empty cell was refused: %v", err)
		}
	})

	t.Run("trailing-backslash-cannot-absorb-outer-pipe", func(t *testing.T) {
		source := "|a|\n|---|\n"
		target := tableCellTargetForValidation(t, source, 0, 0)
		if err := validateTableCellCandidate(target, `b\`, 1); !errors.Is(err, errTableCellShape) {
			t.Fatalf("outer-pipe absorption error = %v", err)
		}
	})

	t.Run("escaped-literal-pipe", func(t *testing.T) {
		source := "|a|\n|---|\n"
		target := tableCellTargetForValidation(t, source, 0, 0)
		if err := validateTableCellCandidate(target, `b\|`, 2); err != nil {
			t.Fatalf("odd-backslash literal pipe was refused: %v", err)
		}
		if tableCellStructuralAt(`b\|`, 2) {
			t.Fatal("odd-backslash pipe was classified as structural")
		}
		if !tableCellStructuralAt(`b\\|`, 3) {
			t.Fatal("even-backslash pipe was not classified as structural")
		}
	})

	t.Run("crlf-and-eof-preserved", func(t *testing.T) {
		source := "|a|b|\r\n|---|---|\r\n|c|d|"
		target := tableCellTargetForValidation(t, source, 1, 0)
		if err := validateTableCellCandidate(target, "更长🙂", len("更长🙂")-len(target.cellSource)); err != nil {
			t.Fatalf("CRLF no-final-newline candidate was refused: %v", err)
		}
		wantSuffix := "|更长🙂|d|"
		if got := tableCellCandidate(target, "更长🙂"); !strings.HasSuffix(got, wantSuffix) ||
			strings.Count(got, "\r\n") != 2 || strings.HasSuffix(got, "\n") {
			t.Fatalf("CRLF/EOF candidate = %q", got)
		}
	})

	t.Run("non-target-ranges-shift-only", func(t *testing.T) {
		source := "| a | b | cc |\n| --- | :---: | ---: |\n| d | middle | ff |\n"
		target := tableCellTargetForValidation(t, source, 1, 1)
		if err := validateTableCellCandidate(target, "longer", len("longer")-len(target.cellSource)); err != nil {
			t.Fatalf("longer non-target delta case was refused: %v", err)
		}
		if err := validateTableCellCandidate(target, "m", len("m")-len(target.cellSource)); err != nil {
			t.Fatalf("shorter non-target delta case was refused: %v", err)
		}
	})
}

func TestTableCellScreenshotsLightDark(t *testing.T) {
	dir := os.Getenv("LEVNOTE_TABLE_SCREENSHOT_DIR")
	if dir == "" {
		t.Skip("set LEVNOTE_TABLE_SCREENSHOT_DIR")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	capture := func(name string, tester *ui.Tester) {
		t.Helper()
		file, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		err = png.Encode(file, tester.Image())
		closeErr := file.Close()
		if err != nil || closeErr != nil {
			t.Fatal(err, closeErr)
		}
	}
	for _, dark := range []bool{false, true} {
		theme := "light"
		if dark {
			theme = "dark"
		}
		source := "# 导语\n\n" + tableViewSource
		buffer := text.New(source)
		v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
		tester := ui.NewTesterWith(v.Draw, 760, 480, ui.TesterOptions{Dark: dark})
		tester.Frame()
		openCellByPointer(t, tester, ui.RoleCell, "数据，第 2 行，第 1 列：甲")
		replaceDraftText(t, tester, "草稿🙂")
		tester.Frame()
		if !tester.HasText(tableCellStagedNotice) {
			t.Fatalf("%s staged image omitted its notice: %q", theme, tester.Texts())
		}
		for _, label := range []string{"完成编辑", "放弃草稿"} {
			info, ok := tester.AccessByRole(ui.RoleButton, label)
			if !ok || info.Offscreen || info.Bounds.X < 0 || info.Bounds.X+info.Bounds.W > 760 ||
				info.Bounds.Y < 0 || info.Bounds.Y+info.Bounds.H > 480 {
				t.Fatalf("%s staged image hid %s: %#v", theme, label, info)
			}
		}
		capture("table-cell-staged-"+theme+"-760x480.png", tester)

		replaceDraftText(t, tester, "坏|值")
		clickDraftControl(t, tester, "完成编辑")
		if !errors.Is(v.draftError, errTableCellPipe) || !tester.HasText(tableCellStagedNotice) ||
			buffer.String() != source {
			t.Fatalf("%s invalid image state=%v texts=%q", theme, v.draftError, tester.Texts())
		}
		for _, label := range []string{"完成编辑", "放弃草稿", "打开未同步副本"} {
			info, ok := tester.AccessByRole(ui.RoleButton, label)
			if !ok || info.Offscreen || info.Bounds.X < 0 || info.Bounds.X+info.Bounds.W > 760 ||
				info.Bounds.Y < 0 || info.Bounds.Y+info.Bounds.H > 480 {
				t.Fatalf("%s invalid image hid %s: %#v", theme, label, info)
			}
		}
		capture("table-cell-invalid-"+theme+"-760x480.png", tester)
	}
}
