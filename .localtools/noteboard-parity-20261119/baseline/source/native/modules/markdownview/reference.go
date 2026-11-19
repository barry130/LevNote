package markdownview

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/markdown"
)

const maxReferenceTargetBytes = 4096
const maxReferenceTitleBytes = 256
const maxReferenceMetadataBytes = 64
const maxReferenceTooltipBytes = 512

const referenceTooLong = "引用不可用：目标超过4096字节；完整原文仍保留，请切到源码查看。"
const referenceNoHost = "引用不可用：当前宿主未提供引用打开能力。"

var errReferenceStale = errors.New("引用原位置已改变或无法确认，请重新查看后操作；未打开引用")

// A reference gesture owns numeric source identity plus one complete bounded
// destination. It never keeps a Block, Document, raw header or List snapshot.
type referenceTarget struct {
	buffer            *text.Buffer
	version, rowID    uint64
	enclosing, header text.Range
	src               string
}

func boundedReferenceCopy(value string, limit int) string {
	if len(value) <= limit {
		return strings.Clone(value)
	}
	end := max(0, limit-len("…"))
	for end > 0 && !utf8.RuneStart(value[end]) {
		end--
	}
	return value[:end] + "…"
}

func referenceTooltip(prefix, value string) string {
	if len(prefix)+len(value) <= maxReferenceTooltipBytes {
		return strings.Clone(prefix + value)
	}
	end := max(0, maxReferenceTooltipBytes-len(prefix)-len("…"))
	for end > 0 && !utf8.RuneStart(value[end]) {
		end--
	}
	return strings.Clone(prefix + value[:end] + "…")
}

func referenceHeaderRange(raw string, block markdown.Block, row previewRow) (text.Range, bool) {
	header := block.DocBlock.HeaderRange
	if block.Kind != markdown.BlockDocBlock ||
		header.Start < block.Range.Start || header.End < header.Start ||
		header.End > block.Range.End || block.Range.Start < 0 ||
		block.Range.End > len(raw) {
		return text.Range{}, false
	}
	start := row.span.Start() + text.UTF16Len(raw[block.Range.Start:header.Start])
	end := start + text.UTF16Len(raw[header.Start:header.End])
	return text.Range{Anchor: start, Head: end}, true
}

func (v *View) docBlock(c *ui.Context, preview previewSnapshot, row previewRow, b markdown.Block) {
	ref := b.DocBlock.Attrs
	ui.Column(c).Padding(14).Radius(5).Border(1, c.Theme().Border).Gap(8).Children(func() {
		label := ref.Title
		if label == "" {
			label = ref.Src
		}
		if label == "" {
			label = "内联文档"
		}
		ui.Text(c, boundedReferenceCopy(label, maxReferenceTitleBytes)).Bold()
		kind := boundedReferenceCopy(ref.Kind, maxReferenceMetadataBytes)
		mode := boundedReferenceCopy(ref.Mode, maxReferenceMetadataBytes)
		ui.Text(c, kind+" · "+mode).FontSize(12).TextColor(c.Theme().TextMuted)

		switch {
		case b.DocBlock.ReferenceProblem != "":
			problem := "引用头不可用：" + boundedReferenceCopy(b.DocBlock.ReferenceProblem, maxReferenceTooltipBytes)
			ui.Text(c, problem).FontSize(11).TextColor(c.Theme().Warning).Tooltip(problem)
		case ref.Mode == "reference" && ref.Src != "":
			v.referenceButton(c, preview, row, b)
		}
		if ref.Mode == "inline" {
			ui.Text(c, b.DocBlock.Payload).Font("monospace").FontSize(13).Selectable()
		}
	})
}

func (v *View) referenceButton(c *ui.Context, preview previewSnapshot, row previewRow, b markdown.Block) {
	oversize := len(b.DocBlock.Attrs.Src) > maxReferenceTargetBytes
	target, ok := referenceTargetOf(preview, row, b)
	ownsGesture := v.referenceActivation.buffer == target.buffer && v.referenceActivation.rowID == target.rowID
	if ownsGesture {
		v.referenceGestureBuiltRowID = target.rowID
	}
	reason := ""
	switch {
	case oversize:
		reason = referenceTooLong
	case !ok:
		reason = "引用不可用：无法确认引用头范围。"
	case v.host.OpenReference == nil:
		reason = referenceNoHost
	}
	tip := reason
	if tip == "" {
		tip = referenceTooltip("打开引用：", target.src)
	}
	if ownsGesture && reason != "" {
		// An unavailable row cannot consume the release. The visible reason
		// replaces the obsolete button, so drop the held source identity now.
		v.referenceActivation = referenceTarget{}
		v.referencePointerUpBuffer, v.referencePointerUpRowID = nil, 0
	}
	button := ui.Button(c, "打开引用").Label("打开引用").Tooltip(tip).Disabled(reason != "")
	if reason == "" {
		button.HandleInput(func(event ui.InputEvent) bool {
			if event.Kind == ui.InputPointerDown && event.Button == 0 && event.Mods == 0 ||
				event.Kind == ui.InputKeyDown && event.Mods == 0 && (event.Key == ui.KeyEnter || event.Key == ui.KeySpace) {
				v.referenceActivation = target
				v.referencePointerUpBuffer, v.referencePointerUpRowID = nil, 0
			}
			if event.Kind == ui.InputScroll && v.referenceActivation.buffer != nil {
				v.referenceActivation = referenceTarget{}
				v.referencePointerUpBuffer, v.referencePointerUpRowID = nil, 0
			}
			if event.Kind == ui.InputPointerUp {
				captured := v.referenceActivation
				if captured.buffer == nil {
					return false
				}
				if _, _, ok := v.referenceMapped(captured); !ok {
					v.referenceActivation = referenceTarget{}
					v.referencePointerUpBuffer, v.referencePointerUpRowID = nil, 0
					v.report(errReferenceStale)
				} else if event.Canceled {
					// A canceled press commits nothing, including when the
					// pressed row was rebuilt after an unrelated edit.
					v.referenceActivation = referenceTarget{}
					v.referencePointerUpBuffer, v.referencePointerUpRowID = nil, 0
				} else {
					// Let MyGo decide whether this was a click. The next pass
					// either consumes a valid click or drops a release that
					// ended outside the button.
					v.referencePointerUpBuffer, v.referencePointerUpRowID = captured.buffer, captured.rowID
				}
			}
			return false
		})
	}
	clicked := button.Clicked()
	pendingOwnsGesture := v.referencePointerUpBuffer == target.buffer && v.referencePointerUpRowID == target.rowID
	if clicked {
		captured := v.referenceActivation
		v.referenceActivation = referenceTarget{}
		v.referencePointerUpBuffer, v.referencePointerUpRowID = nil, 0
		if captured.buffer == nil || captured.buffer != target.buffer || captured.rowID != target.rowID {
			v.report(errReferenceStale)
			return
		}
		err := v.activateReference(captured)
		if err != nil {
			v.report(err)
		}
	} else if pendingOwnsGesture {
		// A valid pointer-up that produced no click, as a release outside the
		// control, is an ordinary canceled gesture. An earlier reference row
		// must not consume a later row's pending release.
		v.referenceActivation = referenceTarget{}
		v.referencePointerUpBuffer, v.referencePointerUpRowID = nil, 0
	}
	if reason != "" {
		ui.Text(c, reason).FontSize(11).TextColor(c.Theme().Warning).Tooltip(reason)
	}
}

func referenceTargetOf(preview previewSnapshot, row previewRow, b markdown.Block) (referenceTarget, bool) {
	header, ok := referenceHeaderRange(preview.doc.Raw(), b, row)
	if !ok || row.item >= 0 ||
		b.DocBlock.ReferenceProblem != "" || b.DocBlock.Attrs.Mode != "reference" ||
		b.DocBlock.Attrs.Src == "" || len(b.DocBlock.Attrs.Src) > maxReferenceTargetBytes {
		return referenceTarget{}, false
	}
	return referenceTarget{
		buffer: preview.buffer, version: preview.version, rowID: row.id,
		enclosing: row.span, header: header, src: strings.Clone(b.DocBlock.Attrs.Src),
	}, true
}

func (v *View) activateReference(target referenceTarget) error {
	if target.buffer == nil {
		return errReferenceStale
	}
	if len(target.src) > maxReferenceTargetBytes {
		return errors.New(referenceTooLong)
	}
	if err := v.Flush(); err != nil {
		return err
	}
	buffer := v.host.Buffer
	if buffer == nil || buffer != target.buffer {
		return errReferenceStale
	}
	if buffer.InGroup() {
		return fmt.Errorf("引用不可用：文档正在另一事务中编辑；%w", text.ErrGroup)
	}
	if buffer.LenBytes() > markdown.MaxParseBytes {
		return fmt.Errorf("引用不可用：%w", errPreviewTooLarge)
	}
	enclosing, header, ok := v.referenceMapped(target)
	if !ok {
		return errReferenceStale
	}
	if v.host.OpenReference == nil {
		return errors.New(referenceNoHost)
	}
	doc := v.doc
	if v.docBuffer != buffer || v.version != buffer.Version() {
		doc = markdown.Parse(buffer.String())
	}
	if doc.Partial || !findReference(doc, enclosing, header, target.src) {
		return errReferenceStale
	}
	// OpenReference may synchronously switch tabs and Release this view.
	// Nothing owned by v is read after this call.
	v.host.OpenReference(target.src)
	return nil
}

func (v *View) referenceMapped(target referenceTarget) (text.Range, text.Range, bool) {
	buffer := v.host.Buffer
	if buffer == nil || buffer != target.buffer || buffer.InGroup() ||
		buffer.LenBytes() > markdown.MaxParseBytes {
		return text.Range{}, text.Range{}, false
	}
	enclosing, enclosingOK := buffer.MapRangeSince(target.version, target.enclosing)
	header, headerOK := buffer.MapRangeSince(target.version, target.header)
	if !enclosingOK || !headerOK {
		return text.Range{}, text.Range{}, false
	}
	return enclosing, header, true
}

func findReference(doc markdown.Document, enclosing, header text.Range, src string) bool {
	raw := doc.Raw()
	byteCursor, unitCursor := 0, 0
	for _, block := range doc.Blocks {
		unitCursor += text.UTF16Len(raw[byteCursor:block.Range.Start])
		start := unitCursor
		unitCursor += text.UTF16Len(raw[block.Range.Start:block.Range.End])
		byteCursor = block.Range.End
		if block.Kind != markdown.BlockDocBlock || block.DocBlock.ReferenceProblem != "" ||
			block.DocBlock.Attrs.Mode != "reference" || block.DocBlock.Attrs.Src != src {
			continue
		}
		if start != enclosing.Start() || unitCursor != enclosing.End() {
			continue
		}
		headerRange := block.DocBlock.HeaderRange
		if headerRange.Start < block.Range.Start || headerRange.End < headerRange.Start ||
			headerRange.End > block.Range.End {
			continue
		}
		currentHeaderStart := start + text.UTF16Len(raw[block.Range.Start:headerRange.Start])
		currentHeaderEnd := currentHeaderStart + text.UTF16Len(raw[headerRange.Start:headerRange.End])
		if currentHeaderStart == header.Start() && currentHeaderEnd == header.End() {
			return true
		}
	}
	return false
}
