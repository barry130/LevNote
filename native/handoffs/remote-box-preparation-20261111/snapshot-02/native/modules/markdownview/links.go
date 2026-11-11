package markdownview

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/markdown"
)

const maxLinkTargetBytes = 4096
const maxInlineLinkControls = 16
const maxInteractiveInlineSpans = 64
const denseLinkGuidance = "此段链接或图片引用较多，已显示完整文字；请切到源码查看目标"
const styledLinkGuidance = "此段行内格式较多，已显示完整文字；请切到源码查看链接或图片引用目标"
const linkTooLong = "链接目标超过4096字节；请在源码中查看，未打开链接"
const linkNoHost = "当前宿主未提供链接打开能力"
const imageReferenceNoHost = "当前宿主未提供图片引用打开能力"

var errLinkStale = errors.New("链接原位置已改变或无法确认，请重新查看后操作；未打开链接")

// Accent is a control fill, not a guaranteed readable text foreground.
// These foregrounds are checked against both native background and quote
// surface in the light/dark palette (normal 16px text, minimum 4.5:1).
func linkForeground(theme *ui.Theme) ui.Color {
	if theme.Dark {
		return ui.Hex("#7dd3c4")
	}
	return ui.Hex("#24675d")
}

// A retained gesture owns only numeric source identity and a bounded cloned
// full destination. In particular no closure keeps a tiny slice of doc.Raw.
type linkTarget struct {
	buffer            *text.Buffer
	version, rowID    uint64
	released          uint64
	enclosing, syntax text.Range
	kind              markdown.BlockKind
	inlineKind        markdown.InlineKind
	item              bool
	href              string
}

func containsLink(inline []markdown.InlineSpan) bool {
	return linkCount(inline) > 0
}

func linkCount(inline []markdown.InlineSpan) int {
	count := 0
	for _, span := range inline {
		if isInlineReference(span.Kind) {
			count++
		}
	}
	return count
}

func isInlineReference(kind markdown.InlineKind) bool {
	return kind == markdown.InlineLink || kind == markdown.InlineImage
}

func linkFallbackReason(inline []markdown.InlineSpan) string {
	count := linkCount(inline)
	if count > maxInlineLinkControls {
		return denseLinkGuidance
	}
	if count > 0 && len(inline) > maxInteractiveInlineSpans {
		return styledLinkGuidance
	}
	return ""
}

func boundedLinkText(value string, limit int) string {
	if len(value) <= limit {
		return strings.Clone(value)
	}
	end := limit
	for !utf8.RuneStart(value[end]) {
		end--
	}
	return value[:end] + "…"
}

func linkName(label string) string {
	if label == "" {
		label = "（空链接）"
	}
	return "链接：" + boundedLinkText(label, 128)
}

func imageReferenceName(label string) string {
	if label == "" {
		label = "（无替代文字）"
	}
	return "图片引用：" + boundedLinkText(label, 128)
}

// Only rows containing a link need native inline child widgets. Other rows
// retain the previous Span path. Source-coordinate conversion advances once
// through each visible row, never rescanning the document prefix per link.
func (v *View) richText(c *ui.Context, p previewSnapshot, row previewRow, value string, base int, inline []markdown.InlineSpan) ui.Element {
	count := linkCount(inline)
	if count == 0 {
		return ui.RichText(c, spans(value, base, inline, c.Theme())...)
	}
	if linkFallbackReason(inline) != "" {
		readable := spans(value, base, inline, c.Theme())
		for i := range readable {
			readable[i].Underline = false
		}
		return ui.RichText(c, readable...)
	}
	return ui.RichText(c).Children(func() {
		cursor := 0
		block := p.doc.Blocks[row.block]
		byteCursor := block.Range.Start
		if row.item >= 0 {
			byteCursor = block.Items[row.item].Range.Start
		}
		unitCursor := row.span.Start()
		for _, in := range inline {
			left, right := in.Range.Start-base, in.Range.End-base
			if left < cursor || left < 0 || right > len(value) || right < left {
				continue
			}
			if left > cursor {
				ui.Text(c, value[cursor:left])
			}
			if isInlineReference(in.Kind) {
				unitCursor += text.UTF16Len(p.doc.Raw()[byteCursor:in.Range.Start])
				start := unitCursor
				unitCursor += text.UTF16Len(p.doc.Raw()[in.Range.Start:in.Range.End])
				byteCursor = in.Range.End
				target := linkTarget{buffer: p.buffer, version: p.version, rowID: row.id, enclosing: row.span,
					syntax: text.Range{Anchor: start, Head: unitCursor}, kind: row.kind, inlineKind: in.Kind,
					item: row.item >= 0, released: v.linkReleased}
				reason := ""
				if len(in.Href) > maxLinkTargetBytes {
					reason = linkTooLong
				} else if v.host.OpenReference == nil {
					reason = linkNoHost
					if in.Kind == markdown.InlineImage {
						reason = imageReferenceNoHost
					}
				} else {
					target.href = strings.Clone(in.Href)
				}
				label := in.Text
				if in.Kind == markdown.InlineImage {
					label = value[left:right] // explicit reference, never inline pixels or alt-only text
				}
				v.inlineLink(c.Key(in.Range.Start-base), label, in.Text, target, reason)
			} else {
				// Reuse the existing non-link styles without introducing a
				// second interpretation of nested/unsupported Markdown syntax.
				ui.RichText(c, spans(value[left:right], in.Range.Start, []markdown.InlineSpan{in}, c.Theme())...)
			}
			cursor = right
		}
		if cursor < len(value) {
			ui.Text(c, value[cursor:])
		}
	})
}

func (v *View) inlineLink(c *ui.Context, label, alt string, target linkTarget, unavailable string) {
	image := target.inlineKind == markdown.InlineImage
	name := linkName(alt)
	if image {
		name = imageReferenceName(alt)
	}
	if label == "" {
		label = "（空链接）"
	}
	tip := unavailable
	if tip == "" {
		tip = "Ctrl/Cmd+点击或 Tab 后 Enter 打开：" + boundedLinkText(target.href, 512)
		if image {
			tip = "图片引用（不内嵌显示）；Ctrl/Cmd+点击或 Tab 后 Enter 打开：" + boundedLinkText(target.href, 512)
		}
	}
	if v.linkActivation.buffer == target.buffer && v.linkActivation.rowID == target.rowID &&
		v.linkActivation.syntax == target.syntax && v.linkActivation.inlineKind == target.inlineKind {
		v.linkGestureBuilt = true
		if unavailable != "" {
			v.linkActivation = linkTarget{}
		}
	}
	link := ui.Text(c, label).TextColor(linkForeground(c.Theme())).Underline().Role(ui.RoleLink).
		Label(name).Tooltip(tip).Disabled(unavailable != "")
	if unavailable == "" {
		link.Focusable()
	}
	link.HitTest(func(x, y float32, bounds ui.Rect) bool { return unavailable == "" && v.linkModifiers&ui.Cmd != 0 })
	link.HandleInput(func(event ui.InputEvent) bool {
		if image && event.Kind == ui.InputKeyDown && event.Key == ui.KeyEnter && event.Mods == 0 && event.Repeat {
			return true // a held Enter never opens the same image repeatedly
		}
		if image && event.Kind == ui.InputPointerDown && event.Button == 0 && event.Mods&ui.Cmd != 0 &&
			(event.Clicks > 1 || v.linkPointerContinuation) {
			v.linkActivation = linkTarget{}
			return true
		}
		if event.Kind == ui.InputPointerDown && event.Button == 0 && event.Mods&ui.Cmd != 0 ||
			event.Kind == ui.InputKeyDown && event.Key == ui.KeyEnter && event.Mods == 0 {
			v.linkActivation = target
			v.linkPointerUp = false
		}
		if event.Kind == ui.InputScroll || event.Kind == ui.InputZoom {
			v.linkActivation = linkTarget{}
		}
		if event.Kind == ui.InputPointerUp {
			v.linkPointerUp = true
			if event.Canceled || event.Mods&ui.Cmd == 0 {
				v.linkActivation = linkTarget{}
			}
		}
		// Plain pointer misses the inline child; Cmd pointer is handled here
		// so a rapid second click cannot begin the enclosing block's draft.
		return false
	})
	if link.Clicked() {
		captured := v.linkActivation
		v.linkActivation = linkTarget{}
		if captured.buffer == nil {
			return // canceled, scrolled, or a repeated image gesture
		}
		if captured.buffer != target.buffer || captured.rowID != target.rowID ||
			captured.inlineKind != target.inlineKind || captured.released != target.released ||
			captured.syntax.Start()-captured.enclosing.Start() != target.syntax.Start()-target.enclosing.Start() {
			v.report(errLinkStale)
		} else {
			v.report(v.activateLink(captured))
		}
	}
	if unavailable != "" {
		message := "〔链接不可用〕"
		if image {
			message = "〔图片引用不可用〕"
		}
		ui.Text(c, message).TextColor(c.Theme().Warning).Tooltip(tip)
	}
}

func (v *View) observeInlineReferenceInput(event ui.InputEvent) {
	if event.Kind == ui.InputPointerDown {
		v.linkPointerContinuation = event.Clicks > 1
	} else if event.Kind == ui.InputPointerUp || event.Kind == ui.InputKeyDown {
		v.linkPointerContinuation = false
	}
	if event.Kind == ui.InputPointerDown || event.Kind == ui.InputScroll || event.Kind == ui.InputZoom ||
		event.Kind == ui.InputPointerUp && event.Canceled ||
		event.Kind == ui.InputKeyDown && (event.Key != ui.KeyEnter || event.Mods != 0) {
		v.linkActivation = linkTarget{}
	}
}

func (v *View) activateLink(target linkTarget) error {
	if target.buffer == nil || target.released != v.linkReleased || !isInlineReference(target.inlineKind) {
		return errLinkStale
	}
	if err := v.Flush(); err != nil {
		return err
	}
	buffer := v.host.Buffer
	if buffer == nil || buffer != target.buffer || target.released != v.linkReleased {
		return errLinkStale
	}
	if len(target.href) > maxLinkTargetBytes {
		return errors.New(linkTooLong)
	}
	if buffer.InGroup() {
		return text.ErrGroup
	}
	if buffer.LenBytes() > markdown.MaxParseBytes {
		return errPreviewTooLarge
	}
	enclosing, outerOK := buffer.MapRangeSince(target.version, target.enclosing)
	syntax, syntaxOK := buffer.MapRangeSince(target.version, target.syntax)
	if !outerOK || !syntaxOK {
		return errLinkStale
	}
	doc := v.doc
	if v.docBuffer != buffer || v.version != buffer.Version() {
		// Validation is independent of the immutable current List snapshot.
		doc = markdown.Parse(buffer.String())
	}
	if doc.Partial {
		return markdown.ErrPartialDocument
	}
	if !findLink(doc, enclosing, syntax, target) {
		return errLinkStale
	}
	if v.host.OpenReference == nil {
		return errors.New(linkNoHost)
	}
	v.host.OpenReference(target.href)
	return nil
}

func findLink(doc markdown.Document, enclosing, syntax text.Range, target linkTarget) bool {
	byteCursor, unitCursor := 0, 0
	check := func(r markdown.SourceRange, inline []markdown.InlineSpan, kind markdown.BlockKind, item bool) bool {
		unitCursor += text.UTF16Len(doc.Raw()[byteCursor:r.Start])
		start := unitCursor
		unitCursor += text.UTF16Len(doc.Raw()[r.Start:r.End])
		byteCursor = r.End
		if start != enclosing.Start() || unitCursor != enclosing.End() || kind != target.kind || item != target.item {
			return false
		}
		at, units := r.Start, start
		for _, in := range inline {
			if !isInlineReference(in.Kind) {
				continue
			}
			units += text.UTF16Len(doc.Raw()[at:in.Range.Start])
			linkStart := units
			units += text.UTF16Len(doc.Raw()[in.Range.Start:in.Range.End])
			at = in.Range.End
			if linkStart == syntax.Start() && units == syntax.End() && in.Kind == target.inlineKind && in.Href == target.href {
				return true
			}
		}
		return false
	}
	for _, block := range doc.Blocks {
		if block.Kind == markdown.BlockList {
			for _, item := range block.Items {
				if check(item.Range, item.Inline, block.Kind, true) {
					return true
				}
			}
		} else if check(block.Range, block.Inline, block.Kind, false) {
			return true
		}
		if unitCursor > enclosing.End() {
			return false
		}
	}
	return false
}
