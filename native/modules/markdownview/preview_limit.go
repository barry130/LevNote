package markdownview

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/formats/markdown"
)

var oversizeSourceGuidance = fmt.Sprintf("Markdown 超过 %d MiB 预览上限。完整原文仍保留，请切到源码视图；HTML 导出已暂停。", markdown.MaxParseBytes>>20)
var errPreviewTooLarge = fmt.Errorf("%s：%w", oversizeSourceGuidance, markdown.ErrPartialDocument)

// checkPreviewSize refuses before whole-source or prefix materialization. A
// refusal owns no raw source and is cached by buffer identity and version.
// Independent field editors and their range-only Flush remain available.
func (v *View) checkPreviewSize() error {
	buffer := v.host.Buffer
	if buffer != nil && buffer.LenBytes() > markdown.MaxParseBytes {
		if !v.loaded || v.docBuffer != buffer || v.version != buffer.Version() || v.previewError == nil {
			v.dropPreviewCache()
			v.doc = markdown.Document{Partial: true}
			v.docBuffer, v.version, v.loaded = buffer, buffer.Version(), true
			v.previewError = errPreviewTooLarge
			v.report(errPreviewTooLarge)
		}
		return errPreviewTooLarge
	}
	if v.previewError != nil {
		v.dropPreviewCache()
		v.docBuffer, v.loaded, v.previewError = nil, false, nil
	}
	return nil
}

func (v *View) dropPreviewCache() {
	v.doc = markdown.Document{}
	v.rows = nil
	v.taskActivation, v.taskRow = taskTarget{}, taskTarget{}
	v.linkActivation, v.hasLinks = linkTarget{}, false
	v.referenceActivation = referenceTarget{}
	v.referencePointerUpBuffer, v.referencePointerUpRowID = nil, 0
	v.tableEditActivation = tableEditTarget{}
	v.tablePointerUpBuffer, v.tablePointerUpRowID = nil, 0
	v.tableCellActivation, v.tableCellGestureBuilt = tableCellTarget{}, false
	v.tableViewport, v.tableViewportKnown = 0, false
	v.tableWindowWidth, v.tableWindowHeight, v.tableWindowKnown = 0, 0, false
	v.hasTables = false
	v.list = &ui.ListState{}
	v.outlineCurrent, v.outlineCaret = "", 0
	if v.outline != nil {
		v.outline.Release()
		v.outline = nil
	}
}
