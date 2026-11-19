package markdownview

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/formats/markdown"
)

func (v *View) block(c *ui.Context, b markdown.Block, preview previewSnapshot, row previewRow) {
	switch b.Kind {
	case markdown.BlockHeading:
		sizes := []float32{30, 25, 21, 19, 17, 16}
		v.richText(c, preview, row, b.Text, b.TextRange.Start, b.Inline).FontSize(sizes[max(0, min(5, b.Level-1))]).Bold().Selectable()
	case markdown.BlockCode:
		ui.Column(c).Padding(14).Gap(8).Radius(5).Background(c.Theme().Surface).Children(func() {
			if b.Language != "" {
				ui.Text(c, b.Language).FontSize(11).TextColor(c.Theme().TextMuted)
			}
			ui.Text(c, b.Text).Font("monospace").FontSize(14).Selectable()
		})
	case markdown.BlockList:
		// Lists are flattened by the virtual preview; never build all items
		// inside one otherwise-visible block.
	case markdown.BlockQuote:
		v.richText(c, preview, row, b.Text, b.TextRange.Start, b.Inline).Padding(12).Background(c.Theme().Surface).TextColor(c.Theme().TextMuted).FontSize(16).Selectable()
	case markdown.BlockDivider:
		ui.Divider(c)
	case markdown.BlockDocBlock:
		v.docBlock(c, preview, row, b)
	case markdown.BlockTable:
		v.tableRow(c, b, preview, row)
	default:
		v.richText(c, preview, row, b.Text, b.TextRange.Start, b.Inline).FontSize(16).LineHeight(1.7).Selectable()
	}
}

func (v *View) listItem(c *ui.Context, b markdown.Block, i int, target taskTarget, preview previewSnapshot, row previewRow) {
	item := b.Items[i]
	prefix := "•"
	if b.Ordered {
		prefix = fmt.Sprintf("%d.", b.Start+i)
	}
	ui.Row(c).AlignItems(ui.Start).Gap(10).Children(func() {
		if item.Task == nil || b.Ordered {
			ui.Text(c, prefix).Width(24)
		}
		if item.Task != nil {
			v.taskCheckbox(c, item, target)
		}
		if len(item.Text) > maxPreviewBlockBytes {
			ui.Text(c, "较大列表项：双击编辑，或切到源码查看完整内容").Grow(1).TextColor(c.Theme().Warning)
		} else if item.Task != nil && item.Text == "" {
			ui.Text(c, "（空任务）").Grow(1).FontSize(16).TextColor(c.Theme().TextMuted)
		} else {
			v.richText(c, preview, row, item.Text, item.TextRange.Start, item.Inline).Grow(1).FontSize(16).Selectable()
		}
	})
}
func spans(value string, base int, inline []markdown.InlineSpan, theme *ui.Theme) []ui.Span {
	result := []ui.Span{}
	cursor := 0
	for _, in := range inline {
		left, right := in.Range.Start-base, in.Range.End-base
		if left < cursor || left < 0 || right > len(value) || right < left {
			continue
		}
		if left > cursor {
			result = append(result, ui.Span{Text: value[cursor:left]})
		}
		span := ui.Span{Text: in.Text}
		switch in.Kind {
		case markdown.InlineBold:
			span.Weight = 700
		case markdown.InlineItalic:
			span.Italic = true
		case markdown.InlineCode:
			span.Font = "monospace"
			span.Background = theme.Surface
			span.PaddingX = 3
			span.Radius = 3
		case markdown.InlineLink:
			span.Color = linkForeground(theme)
			span.Underline = true
		case markdown.InlineImage:
			// Dense/fallback rows must retain the complete reference syntax.
			span.Text = value[left:right]
		}
		result = append(result, span)
		cursor = right
	}
	if cursor < len(value) {
		result = append(result, ui.Span{Text: value[cursor:]})
	}
	return result
}
