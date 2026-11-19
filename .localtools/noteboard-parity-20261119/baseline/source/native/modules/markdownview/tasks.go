package markdownview

import (
	"errors"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/markdown"
)

var errTaskStale = errors.New("任务原位置已改变或无法确认，请重新查看后操作；未切换任务状态")
var errTaskReadOnly = errors.New("文档当前只读，无法切换任务状态")

// A command owns numeric identity only, never a label or a parser/raw string.
type taskTarget struct {
	buffer       *text.Buffer
	version      uint64
	rowID        uint64
	item, marker text.Range
}

func (p previewSnapshot) taskTarget(row previewRow, item markdown.ListItem) taskTarget {
	if item.Task == nil {
		return taskTarget{}
	}
	// The row already owns the full UTF-16 item range. Only the short list
	// prefix is scanned here, not the complete document before every row.
	start := row.span.Start() + text.UTF16Len(p.doc.Raw()[item.Range.Start:item.Task.Range.Start])
	return taskTarget{p.buffer, p.version, row.id, row.span, text.Range{Anchor: start, Head: start + 3}}
}

func taskName(item markdown.ListItem) string {
	label := item.Text
	if label == "" {
		label = "（空任务）"
	} else if len(label) > 128 {
		end := 128
		for !utf8.RuneStart(label[end]) {
			end--
		}
		label = label[:end] + "…"
	}
	if item.Task.Checked {
		return "已完成任务：" + label
	}
	return "未完成任务：" + label
}

func (v *View) taskCheckbox(c *ui.Context, item markdown.ListItem, target taskTarget) {
	checked := item.Task.Checked
	name := taskName(item)
	checkbox := ui.Checkbox(c.Key("task"), &checked, "").Label(name).Tooltip(name).Disabled(v.host.ReadOnly)
	checkbox.HandleInput(func(event ui.InputEvent) bool {
		if event.Kind == ui.InputPointerDown && event.Button == 0 ||
			event.Kind == ui.InputKeyDown && event.Key == ui.KeySpace && event.Mods == 0 {
			if v.host.ReadOnly {
				v.taskActivation = taskTarget{}
				v.report(errTaskReadOnly)
				return true
			}
			v.taskActivation = target
		}
		if event.Kind == ui.InputPointerUp && event.Canceled {
			v.taskActivation = taskTarget{}
		}
		// Native input still toggles/focuses the checkbox. Owning this handler
		// keeps rapid second presses from reaching the parent draft handler.
		return false
	})
	if checkbox.Changed() {
		if v.taskActivation.rowID == target.rowID && v.taskActivation.buffer != nil {
			target = v.taskActivation
		}
		v.taskActivation = taskTarget{}
		v.report(v.toggleTask(target))
	}
}

func (v *View) toggleTask(target taskTarget) error {
	if err := v.Flush(); err != nil {
		return err
	}
	buffer := v.host.Buffer
	if buffer == nil || buffer != target.buffer {
		return errTaskStale
	}
	if v.host.ReadOnly {
		return errTaskReadOnly
	}
	if buffer.InGroup() {
		return text.ErrGroup
	}
	if buffer.LenBytes() > markdown.MaxParseBytes {
		return errPreviewTooLarge
	}
	itemRange, itemOK := buffer.MapRangeSince(target.version, target.item)
	markerRange, markerOK := buffer.MapRangeSince(target.version, target.marker)
	if !itemOK || !markerOK {
		return errTaskStale
	}
	doc := v.doc
	if v.docBuffer != buffer || v.version != buffer.Version() {
		// Temporary validation never replaces arrays being read by this List
		// build. In particular an inserted fence can change untouched syntax.
		doc = markdown.Parse(buffer.String())
	}
	if doc.Partial {
		return markdown.ErrPartialDocument
	}
	checked, recognized := findTask(doc, itemRange, markerRange)
	if !recognized {
		return errTaskStale
	}
	value := "x"
	if checked {
		value = " "
	}
	selection := buffer.Selection()
	if err := buffer.BeginGroup(); err != nil {
		return err
	}
	if err := buffer.ReplaceUTF16(markerRange.Start()+1, markerRange.Start()+2, value); err != nil {
		buffer.CancelGroup()
		return err
	}
	if err := buffer.SetSelection(selection.Anchor, selection.Head); err != nil {
		buffer.CancelGroup()
		return err
	}
	buffer.EndGroup()
	v.taskRow = taskTarget{buffer, buffer.Version(), target.rowID, itemRange, markerRange}
	v.loaded = false // next pass reparses; the current pass stays immutable
	if v.host.Changed != nil {
		v.host.Changed()
	}
	return nil
}

// Ordered ranges advance one UTF-16 cursor; even a late task never causes a
// full-prefix scan per list item. Only the exact item AND marker may match.
func findTask(doc markdown.Document, itemRange, markerRange text.Range) (bool, bool) {
	byteCursor, unitCursor := 0, 0
	for _, block := range doc.Blocks {
		if block.Kind != markdown.BlockList {
			continue
		}
		for _, item := range block.Items {
			unitCursor += text.UTF16Len(doc.Raw()[byteCursor:item.Range.Start])
			start := unitCursor
			unitCursor += text.UTF16Len(doc.Raw()[item.Range.Start:item.Range.End])
			byteCursor = item.Range.End
			if start > itemRange.Start() {
				return false, false
			}
			if item.Task == nil || start != itemRange.Start() || unitCursor != itemRange.End() {
				continue
			}
			markerStart := start + text.UTF16Len(doc.Raw()[item.Range.Start:item.Task.Range.Start])
			if markerStart == markerRange.Start() && markerStart+3 == markerRange.End() {
				return item.Task.Checked, true
			}
		}
	}
	return false, false
}
