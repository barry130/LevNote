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
	"levnote.local/native/ui/editor"
)

const tableViewSource = "| 名称 | 备注 |\r\n| :--- | ---: |\r\n| 甲 | **字面** |\r\n| 🙂 |  |\r\n"

func tableAccess(t *testing.T, tester *ui.Tester) ui.AccessInfo {
	t.Helper()
	info, ok := tester.AccessByRole(ui.RoleButton, tableEditButtonLabel)
	if !ok {
		t.Fatalf("table source button missing: %#v", tester.Texts())
	}
	if info.Disabled || !info.Focusable || info.Bounds.W <= 0 || info.Bounds.H <= 0 || info.Offscreen {
		t.Fatalf("table source button state/geometry = %#v", info)
	}
	if len(info.Description) > tableTooltipBytes || !utf8.ValidString(info.Description) {
		t.Fatalf("table source button description exceeds bound: %d %q", len(info.Description), info.Description)
	}
	return info
}

func focusTableEditButton(t *testing.T, tester *ui.Tester, info ui.AccessInfo) {
	t.Helper()
	tester.Press(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	tester.CancelPress()
	tester.Frame()
	if tester.Focused(tableEditButtonLabel) {
		return
	}
	for attempt := 0; attempt < 120; attempt++ {
		tester.Key(0, ui.KeyTab)
		tester.Frame()
		if tester.Focused(tableEditButtonLabel) {
			return
		}
	}
	t.Fatalf("table source button could not receive keyboard focus: %#v", tester.Texts())
}

func replaceTableBufferText(t *testing.T, buffer *text.Buffer, old, replacement string) {
	t.Helper()
	source := buffer.String()
	index := strings.Index(source, old)
	if index < 0 {
		t.Fatalf("fixture text %q is absent", old)
	}
	start := text.UTF16Len(source[:index])
	end := start + text.UTF16Len(old)
	if err := buffer.ReplaceUTF16(start, end, replacement); err != nil {
		t.Fatal(err)
	}
}

func TestTableNativePointerEnterSpaceCancelAndFlush(t *testing.T) {
	for _, gesture := range []string{"pointer", "enter", "space"} {
		t.Run(gesture, func(t *testing.T) {
			original := "# 导语\r\n\r\n" + tableViewSource
			buffer := text.New(original)
			changes := 0
			v := Adapter().Create(editor.Host{
				Buffer: buffer,
				Changed: func() {
					changes++
				},
			}).(*View)
			tester := ui.NewTester(v.Draw, 760, 480)
			tester.Frame()

			beginDraft(t, v, tester, "导语")
			tester.Command("selectAll")
			tester.Compose("已同步🙂", 3)
			tester.Frame()
			button := tableAccess(t, tester)
			draft := v.draft

			tester.Press(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
			tester.CancelPress()
			tester.Frame()
			if v.draft == nil || v.draft != draft || v.editPanelHeight != 0 ||
				buffer.String() != original {
				t.Fatalf("ordinary canceled press changed draft/panel/host: sameDraft=%v panel=%v host=%q wantHost=%q gesture=%v pointer=%v/%d",
					v.draft == draft, v.editPanelHeight, buffer.String(), original,
					v.tableEditActivation.buffer != nil, v.tablePointerUpBuffer != nil, v.tablePointerUpRowID)
			}
			if v.tableEditActivation.buffer != nil || v.tablePointerUpBuffer != nil {
				t.Fatal("ordinary canceled press retained table source identity")
			}

			switch gesture {
			case "pointer":
				tester.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
			case "enter", "space":
				focusTableEditButton(t, tester, button)
				if gesture == "enter" {
					tester.Key(0, ui.KeyEnter)
				} else {
					tester.Key(0, ui.KeySpace)
				}
			}
			tester.Frame()
			if v.draft == nil || v.draft.Buffer.String() != tableViewSource {
				t.Fatalf("%s table editor source = %#v", gesture, func() string {
					if v.draft == nil {
						return "<nil>"
					}
					return v.draft.Buffer.String()
				}())
			}
			if v.editPanelHeight != 240 {
				t.Fatalf("%s table editor height = %v", gesture, v.editPanelHeight)
			}
			want := "# 已同步🙂\r\n\r\n" + tableViewSource
			if buffer.String() != want {
				t.Fatalf("%s did not Flush pending Compose: %q", gesture, buffer.String())
			}
			if changes != 1 || buffer.History().UndoDepth != 1 || buffer.History().RedoDepth != 0 {
				t.Fatalf("%s Flush history = %#v changes=%d", gesture, buffer.History(), changes)
			}
			if v.tableEditActivation.buffer != nil || v.tablePointerUpBuffer != nil {
				t.Fatal("table gesture survived activation")
			}
		})
	}
}

func TestTableHeldSourceIdentityRebaseAndRefusal(t *testing.T) {
	for _, interveningFrame := range []bool{false, true} {
		t.Run("insertion-before/frame="+map[bool]string{false: "false", true: "true"}[interveningFrame], func(t *testing.T) {
			buffer := text.New("# 导语\n\n" + tableViewSource)
			v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
			tester := ui.NewTester(v.Draw, 760, 480)
			tester.Frame()
			button := tableAccess(t, tester)
			x, y := button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2
			tester.Press(x, y)
			if err := buffer.ReplaceUTF16(0, 0, "前置段落\r\n\r\n"); err != nil {
				t.Fatal(err)
			}
			if interveningFrame {
				tester.Frame()
			}
			tester.Release(x, y)
			tester.Frame()
			if v.draft == nil || v.draft.Buffer.String() != tableViewSource {
				t.Fatalf("unrelated insertion did not rebase the exact table: draft=%#v", v.draft)
			}
		})
	}

	for _, interveningFrame := range []bool{false, true} {
		t.Run("same-table-edit/frame="+map[bool]string{false: "false", true: "true"}[interveningFrame], func(t *testing.T) {
			buffer := text.New("# 导语\n\n" + tableViewSource)
			var reports []string
			v := Adapter().Create(editor.Host{
				Buffer: buffer,
				Report: func(err error) { reports = append(reports, err.Error()) },
			}).(*View)
			tester := ui.NewTester(v.Draw, 760, 480)
			tester.Frame()
			button := tableAccess(t, tester)
			x, y := button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2
			tester.Press(x, y)
			replaceTableBufferText(t, buffer, "**字面**", "被修改")
			if interveningFrame {
				tester.Frame()
			}
			tester.Release(x, y)
			tester.Frame()
			if v.draft != nil {
				t.Fatal("same-table source change opened a stale editor")
			}
			if len(reports) != 1 || !strings.Contains(reports[0], "表格") {
				t.Fatalf("stale table gesture reports = %#v", reports)
			}
			if v.tableEditActivation.buffer != nil || v.tablePointerUpBuffer != nil {
				t.Fatal("stale table gesture retained source identity")
			}
		})
	}
}

func TestTableSecondVisiblePointerTarget(t *testing.T) {
	body := "前言\r\n\r\n" + tableViewSource + "\r\n\r\n中段\r\n\r\n" + tableViewSource + "\r\n\r\n结尾\r\n"
	buffer := text.New(body)
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	tester := ui.NewTester(v.Draw, 1180, 780)
	tester.Frame()
	tester.Frame()
	if tester.FramePending() || !v.tableViewportKnown {
		t.Fatal("second-table target was measured before responsive layout settled")
	}

	first := tableAccess(t, tester)
	secondByte := strings.LastIndex(body, tableViewSource)
	secondStart := text.UTF16Len(body[:secondByte])
	firstLead, ok := tester.Find("前言")
	if !ok {
		t.Fatal("first table lead missing")
	}
	secondLead, ok := tester.Find("中段")
	if !ok {
		t.Fatal("second table lead missing")
	}
	secondHeaderIndex := -1
	for index, row := range v.rows {
		if row.tableRow == 0 && row.span.Start() == secondStart {
			secondHeaderIndex = index
			break
		}
	}
	if secondHeaderIndex < 0 {
		t.Fatal("second table header row missing")
	}
	x := first.Bounds.X + first.Bounds.W/2
	y := first.Bounds.Y + first.Bounds.H/2 + secondLead.Y - firstLead.Y
	if x < 0 || x >= 1180 || y <= secondLead.Y || y >= 780 {
		t.Fatalf("second-table pointer %.1f:%.1f is outside the settled second table column", x, y)
	}
	tester.Press(x, y)
	if v.tableEditActivation.buffer != buffer || v.tableEditActivation.rowID != v.rows[secondHeaderIndex].id {
		t.Fatalf("second pointer captured row %d, want visible second table row %d", v.tableEditActivation.rowID, v.rows[secondHeaderIndex].id)
	}
	tester.Release(x, y)
	tester.Frame()

	wantStart := text.UTF16Len(body[:secondByte])
	wantEnd := wantStart + text.UTF16Len(tableViewSource)
	if v.draft == nil || v.draft.Buffer.String() != tableViewSource ||
		v.editSpan.Start() != wantStart || v.editSpan.End() != wantEnd {
		t.Fatalf("second same-shape table opened span %+v, want %d:%d", v.editSpan, wantStart, wantEnd)
	}
}

func TestTableDoubleClickUsesCellSourceIdentity(t *testing.T) {
	buffer := text.New("# 导语\n\n" + tableViewSource)
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	tester := ui.NewTester(v.Draw, 760, 480)
	tester.Frame()
	cell, ok := tester.AccessByRole(ui.RoleCell, "数据，第 2 行，第 1 列：甲")
	if !ok {
		t.Fatal("table body cell missing")
	}
	tester.ClickAt(cell.Bounds.X+cell.Bounds.W/2, cell.Bounds.Y+cell.Bounds.H/2)
	tester.Frame()
	tester.ClickAt(cell.Bounds.X+cell.Bounds.W/2, cell.Bounds.Y+cell.Bounds.H/2)
	tester.Frame()
	if v.draft == nil || v.draft.Buffer.String() != "甲" || v.cellEdit == nil {
		t.Fatalf("double-click did not open the literal cell source: %#v", v.draft)
	}
	want := text.UTF16Len("# 导语\n\n| 名称 | 备注 |\r\n| :--- | ---: |\r\n| ")
	if v.editSpan.Start() != want || v.editSpan.End() != want+text.UTF16Len("甲") {
		t.Fatalf("cell edit span = %+v, want %d:%d", v.editSpan, want, want+text.UTF16Len("甲"))
	}
}

func TestTableOverPreviewLimitDoubleClickOpensExplicitSourceFallback(t *testing.T) {
	var source strings.Builder
	source.WriteString("| 行号 | 内容 |\n| --- | --- |\n")
	for row := 0; len(source.String()) <= maxPreviewBlockBytes; row++ {
		fmt.Fprintf(&source, "| %d | %s |\n", row, strings.Repeat("界", 80))
	}
	table := source.String()
	if len(table) <= maxPreviewBlockBytes {
		t.Fatalf("fixture is %d bytes, want over %d", len(table), maxPreviewBlockBytes)
	}
	buffer := text.New(table)
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	tester := ui.NewTester(v.Draw, 760, 480)
	tester.Frame()

	if _, ok := tester.AccessByRole(ui.RoleButton, tableEditButtonLabel); ok {
		t.Fatal("over-limit table claimed the bounded whole-table action")
	}
	guidance := fmt.Sprintf("较大内容块（%d 字节）：双击分段编辑，或切到源码查看完整内容", len(table))
	placeholder, ok := tester.Find(guidance)
	if !ok || placeholder.X < 0 || placeholder.Y < 0 || placeholder.X+placeholder.W > 760 || placeholder.Y+placeholder.H > 480 {
		t.Fatalf("over-limit table source fallback missing: %#v texts=%q", placeholder, tester.Texts())
	}
	x, y := placeholder.X+placeholder.W/2, placeholder.Y+placeholder.H/2
	tester.ClickAt(x, y)
	tester.Frame()
	tester.ClickAt(x, y)
	tester.Frame()
	if v.draft == nil || v.draft.Buffer.String() != table {
		t.Fatalf("over-limit double-click did not execute the advertised source path: %#v", v.draft)
	}
	if v.editSpan.Start() != 0 || v.editSpan.End() != text.UTF16Len(table) {
		t.Fatalf("over-limit source span=%+v want=0:%d", v.editSpan, text.UTF16Len(table))
	}
	if buffer.String() != table || buffer.CanUndo() || buffer.CanRedo() {
		t.Fatal("opening over-limit source fallback changed bytes or history")
	}
}

func TestTableFallbackDoubleClickUsesSnapshotAcrossInsertion(t *testing.T) {
	raw := "| a | b |\n| - |\n| more | bad |\n"
	source := "# 导语\n\n" + raw + "\n尾部\n"
	buffer := text.New(source)
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	tester := ui.NewTester(v.Draw, 760, 480)
	tester.Frame()
	fallback, ok := tester.Find(raw)
	if !ok {
		t.Fatalf("known table fallback missing: %#v", tester.Texts())
	}
	x, y := fallback.X+fallback.W/2, fallback.Y+fallback.H/2
	tester.ClickAt(x, y)

	prefix := "前置段落\r\n\r\n"
	if err := buffer.ReplaceUTF16(0, 0, prefix); err != nil {
		t.Fatal(err)
	}
	tester.ClickAt(x, y)
	tester.Frame()
	if v.draft == nil || v.draft.Buffer.String() != raw {
		t.Fatalf("fallback double-click did not open complete source: %#v", v.draft)
	}
	wantStart := text.UTF16Len(prefix + source[:strings.Index(source, raw)])
	if v.editSpan.Start() != wantStart || v.editSpan.End() != wantStart+text.UTF16Len(raw) {
		t.Fatalf("fallback double-click span = %+v, want %d", v.editSpan, wantStart)
	}
}

func TestTableResponsiveSettlesAndKeepsRowIdentity(t *testing.T) {
	buffer := text.New(tableViewSource)
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	tester := ui.NewTester(v.Draw, 760, 480)
	tester.Frame()
	if !tableRowUsableForGrid(v, 2) || tester.HasText("表头 · 第 1 行") {
		t.Fatalf("wide table did not settle as a grid: viewport=%v known=%v", v.tableViewport, v.tableViewportKnown)
	}
	ids := make([]uint64, len(v.rows))
	for index, row := range v.rows {
		ids[index] = row.id
	}

	tester.SetSize(265, 480)
	tester.Frame()
	if tableRowUsableForGrid(v, 2) || !tester.HasText("表头 · 第 1 行") || !tester.HasText("数据 · 第 2 行") {
		t.Fatalf("narrow table did not settle as labelled records: viewport=%v known=%v", v.tableViewport, v.tableViewportKnown)
	}
	if v.frameRows > 20 || tester.FramePending() {
		t.Fatal("narrow resize did not converge without idle frames", v.frameRows)
	}
	for index, row := range v.rows {
		if row.id != ids[index] {
			t.Fatal("responsive breakpoint changed source row identity", index, row.id, ids[index])
		}
	}
	if buffer.CanUndo() || buffer.CanRedo() {
		t.Fatal("layout resize changed undo history")
	}

	tester.SetSize(760, 480)
	tester.Frame()
	if !tableRowUsableForGrid(v, 2) || tester.HasText("表头 · 第 1 行") {
		t.Fatal("wide resize did not return to the equal grid")
	}
	for index, row := range v.rows {
		if row.id != ids[index] {
			t.Fatal("wide resize changed source row identity", index, row.id, ids[index])
		}
	}
	if tester.FramePending() {
		t.Fatal("table layout retained a perpetual invalidate")
	}
}

func TestTableEightColumnHeaderActionAndBoundedLiteralCells(t *testing.T) {
	long := strings.Repeat("界", 200)
	table := "| 一 | 二 | 三 | 四 | 五 | 六 | 七 | 八 |\r\n" +
		"| --- | --- | --- | --- | --- | --- | --- | --- |\r\n" +
		"| " + long + " |  | A | **B** | <i>C</i> | [D](x) | E🙂 | F |\r\n"
	buffer := text.New(table)
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	tester := ui.NewTester(v.Draw, 265, 480)
	tester.Frame()

	button := tableAccess(t, tester)
	header, ok := tester.Find("表头 · 第 1 行")
	if !ok {
		t.Fatal("stacked header label missing")
	}
	if button.Bounds.X < 0 || button.Bounds.X+button.Bounds.W > 265 ||
		button.Bounds.Y < 0 || button.Bounds.Y+button.Bounds.H > 480 || button.Bounds.Y+button.Bounds.H > header.Y {
		t.Fatalf("source action is clipped or pushed below its first record: button=%#v header=%#v", button.Bounds, header)
	}

	v.list.ScrollTo(1, ui.Start)
	tester.Frame()
	cellName := "数据，第 2 行，第 1 列：" + long
	cellName = boundedTableText(cellName, tableTooltipBytes)
	cell, ok := tester.AccessByRole(ui.RoleCell, cellName)
	if !ok || cell.Offscreen || cell.Bounds.X < 0 || cell.Bounds.X+cell.Bounds.W > 265 ||
		cell.Bounds.Y < 0 || cell.Bounds.Y+cell.Bounds.H > 480 {
		t.Fatalf("visible long literal cell missing or clipped: %#v found=%v", cell, ok)
	}
	if cell.Description != tableCellActionHint || !cell.Focusable || !utf8.ValidString(cell.Name) {
		t.Fatalf("long cell action/accessibility = %#v", cell)
	}
	empty, ok := tester.AccessByRole(ui.RoleCell, "数据，第 2 行，第 2 列："+tableEmptyCell)
	if !ok || empty.Offscreen || empty.Bounds.X < 0 || empty.Bounds.X+empty.Bounds.W > 265 ||
		empty.Bounds.Y < 0 || empty.Bounds.Y+empty.Bounds.H > 480 || empty.Description != tableCellActionHint {
		t.Fatalf("empty cell marker/tooltip = %#v found=%v", empty, ok)
	}
	if !tester.HasText("表格") || !tester.HasText("原文") || buffer.String() != table {
		t.Fatal("stacked literal preview changed or omitted source guidance")
	}
}

func TestTableReadonlyFailedFlushRecoveryAndRelease(t *testing.T) {
	t.Run("readonly", func(t *testing.T) {
		source := "# 导语\n\n" + tableViewSource
		buffer := text.New(source)
		v := Adapter().Create(editor.Host{Buffer: buffer, ReadOnly: true}).(*View)
		tester := ui.NewTester(v.Draw, 760, 480)
		tester.Frame()
		if _, ok := tester.AccessByRole(ui.RoleButton, tableEditButtonLabel); ok ||
			!tester.HasText(tableReadonlyLabel) || !tester.HasText("完整源码") {
			t.Fatalf("readonly controls/guidance = %#v", tester.Texts())
		}
		cell, ok := tester.Find("甲")
		if !ok {
			t.Fatal("readonly table cell missing")
		}
		tester.ClickAt(cell.X+cell.W/2, cell.Y+cell.H/2)
		tester.Frame()
		tester.ClickAt(cell.X+cell.W/2, cell.Y+cell.H/2)
		tester.Frame()
		if v.draft != nil || buffer.String() != source || buffer.CanUndo() {
			t.Fatal("readonly table double-click created an editor or history")
		}
	})

	t.Run("readonly-after-pointer-down-retains-draft", func(t *testing.T) {
		source := "# 导语\n\n" + tableViewSource
		buffer := text.New(source)
		var reports []string
		v := Adapter().Create(editor.Host{
			Buffer: buffer,
			Report: func(err error) { reports = append(reports, err.Error()) },
		}).(*View)
		tester := ui.NewTester(v.Draw, 760, 480)
		beginDraft(t, v, tester, "导语")
		tester.Command("selectAll")
		tester.Compose("保留草稿🙂", 3)
		tester.Frame()
		draft := v.draft
		button := tableAccess(t, tester)
		x, y := button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2
		tester.Press(x, y)
		v.host.ReadOnly = true
		tester.Release(x, y)
		tester.Frame()
		if v.draft != draft || buffer.String() != source || buffer.CanUndo() {
			t.Fatal("readonly flip changed the pending draft or source")
		}
		if len(reports) == 0 || !strings.Contains(strings.Join(reports, "\n"), "只读") {
			t.Fatalf("readonly flip refusal = %#v", reports)
		}
		if v.tableEditActivation.buffer != nil || v.tablePointerUpBuffer != nil {
			t.Fatal("readonly flip retained the table gesture")
		}
	})

	t.Run("failed-flush-recovery", func(t *testing.T) {
		source := "# 导语\n\n" + tableViewSource
		buffer := text.New(source)
		var reports []string
		v := Adapter().Create(editor.Host{
			Buffer: buffer,
			Report: func(err error) { reports = append(reports, err.Error()) },
		}).(*View)
		tester := ui.NewTester(v.Draw, 760, 480)
		beginDraft(t, v, tester, "导语")
		tester.Command("selectAll")
		tester.Compose("保留草稿🙂", 5)
		replaceTableBufferText(t, buffer, "导语", "外部")
		tester.Frame()
		draft := v.draft
		version, selection, history := buffer.Version(), buffer.Selection(), buffer.History()
		button := tableAccess(t, tester)
		tester.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
		tester.Frame()
		if v.draft != draft || buffer.Version() != version || buffer.Selection() != selection || buffer.History() != history {
			t.Fatal("failed table Flush changed source, draft, selection or history")
		}
		if !errors.Is(v.draftError, errDraftConflict) ||
			!strings.Contains(strings.Join(reports, "\n"), "草稿仍保留") {
			t.Fatalf("failed table Flush refusal = %#v error=%v", reports, v.draftError)
		}
		if !tester.HasText("完成编辑") || !tester.HasText("打开未同步副本") {
			t.Fatalf("failed table Flush lost recovery controls: %#v", tester.Texts())
		}
	})

	t.Run("whole-table-conflict-controls-fit", func(t *testing.T) {
		source := "# 导语\n\n" + tableViewSource
		buffer := text.New(source)
		v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
		tester := ui.NewTester(v.Draw, 760, 480)
		button := tableAccess(t, tester)
		tester.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
		tester.Frame()
		if v.draft == nil || v.editPanelHeight != 240 {
			t.Fatal("whole-table editor did not open with its fixed 240-DIP panel")
		}
		tester.Command("selectAll")
		tester.Compose("表格草稿🙂", 3)
		tester.Frame()
		replaceTableBufferText(t, buffer, "名称", "外部")
		tester.Frame()
		if !errors.Is(v.draftError, errDraftConflict) || !tester.HasText("草稿尚未写回") {
			t.Fatalf("table conflict status = %v texts=%#v", v.draftError, tester.Texts())
		}
		for _, label := range []string{"完成编辑", "打开未同步副本"} {
			info, ok := tester.AccessByRole(ui.RoleButton, label)
			if !ok || info.Offscreen || info.Bounds.X < 0 || info.Bounds.X+info.Bounds.W > 760 ||
				info.Bounds.Y < 0 || info.Bounds.Y+info.Bounds.H > 480 {
				t.Fatalf("%s is not reachable at 760x480: %#v", label, info)
			}
		}
	})

	t.Run("release-cache", func(t *testing.T) {
		buffer := text.New(tableViewSource)
		v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
		tester := ui.NewTester(v.Draw, 265, 480)
		tester.Frame()
		if !v.hasTables || !v.tableViewportKnown {
			t.Fatal("table cache was not built")
		}
		v.Release()
		if v.hasTables || v.tableViewportKnown || len(v.rows) != 0 || v.doc.Raw() != "" ||
			v.tableEditActivation.buffer != nil || v.tablePointerUpBuffer != nil || buffer.CanUndo() {
			t.Fatal("Release retained table preview, gesture or layout cache")
		}
	})
}

func TestTableFarRowsVirtualizeWithoutIdleWork(t *testing.T) {
	var source strings.Builder
	source.WriteString("| A | B |\n| --- | --- |\n")
	for index := 0; index < 256; index++ {
		source.WriteString("| row | value |\n")
	}
	buffer := text.New(source.String())
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	tester := ui.NewTester(v.Draw, 265, 480)
	v.list.ScrollTo(240, ui.Start)
	tester.Frame()
	if v.frameRows > 60 || v.frameRows == 0 || tester.FramePending() {
		t.Fatalf("far table virtualization built rows=%d pending=%v", v.frameRows, tester.FramePending())
	}
	for attempt := 0; attempt < 3; attempt++ {
		tester.Frame()
	}
	if tester.FramePending() || buffer.CanUndo() || buffer.CanRedo() {
		t.Fatal("idle table preview frames or navigation changed history")
	}
}

func TestTableTooltipBoundIsRuneSafe(t *testing.T) {
	value := strings.Repeat("界", 300)
	got := boundedTableTooltip("第 1 列：", value, tableTooltipBytes)
	if len(got) > tableTooltipBytes || !strings.HasSuffix(got, "…") || !utf8.ValidString(got) {
		t.Fatalf("tooltip bound/runes = %d %q", len(got), got)
	}
}

func TestTableScreenshotsLightDark(t *testing.T) {
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

		v := Adapter().Create(editor.Host{Buffer: text.New(tableViewSource)}).(*View)
		tester := ui.NewTesterWith(v.Draw, 265, 480, ui.TesterOptions{Dark: dark})
		tester.Frame()
		tester.Frame()
		if tester.FramePending() || !v.tableViewportKnown || tableRowUsableForGrid(v, 2) {
			t.Fatalf("%s narrow image did not settle as a stacked table: known=%v viewport=%v pending=%v",
				theme, v.tableViewportKnown, v.tableViewport, tester.FramePending())
		}
		capture("table-stacked-"+theme+"-265x480.png", tester)

		v = Adapter().Create(editor.Host{Buffer: text.New(tableViewSource)}).(*View)
		tester = ui.NewTesterWith(v.Draw, 760, 480, ui.TesterOptions{Dark: dark})
		tester.Frame()
		tester.Frame()
		if tester.FramePending() || !tableRowUsableForGrid(v, 2) {
			t.Fatalf("%s wide image did not settle as a grid: viewport=%v pending=%v",
				theme, v.tableViewport, tester.FramePending())
		}
		capture("table-grid-"+theme+"-760x480.png", tester)

		buffer := text.New("# 导语\n\n" + tableViewSource)
		v = Adapter().Create(editor.Host{Buffer: buffer}).(*View)
		tester = ui.NewTesterWith(v.Draw, 760, 480, ui.TesterOptions{Dark: dark})
		tester.Frame()
		button := tableAccess(t, tester)
		tester.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
		tester.Frame()
		if v.draft == nil || v.editPanelHeight != 240 {
			t.Fatalf("%s conflict image did not open the fixed 240-DIP table editor", theme)
		}
		tester.Command("selectAll")
		tester.Compose("表格草稿🙂", 3)
		tester.Frame()
		replaceTableBufferText(t, buffer, "名称", "外部")
		tester.Frame()
		tester.Frame()
		if !errors.Is(v.draftError, errDraftConflict) {
			t.Fatalf("%s conflict image state = %v", theme, v.draftError)
		}
		for _, label := range []string{"完成编辑", "打开未同步副本"} {
			info, ok := tester.AccessByRole(ui.RoleButton, label)
			if !ok || info.Offscreen || info.Bounds.X < 0 || info.Bounds.X+info.Bounds.W > 760 ||
				info.Bounds.Y < 0 || info.Bounds.Y+info.Bounds.H > 480 {
				t.Fatalf("%s conflict image hid %s: %#v", theme, label, info)
			}
		}
		capture("table-conflict-"+theme+"-760x480.png", tester)
	}
}
