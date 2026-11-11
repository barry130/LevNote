package graphview

import (
	"errors"
	"fmt"

	"levnote.local/native/formats/drawio"
	"levnote.local/native/ui/editor"
)

const (
	maxViewExportPageIndex   = 249_999
	maxViewExportPageIDBytes = 512
)

var (
	ErrViewExportKind      = errors.New("view export requests are only available for draw.io")
	ErrViewExportNoBuffer  = errors.New("draw.io export has no host buffer")
	ErrViewExportGroup     = errors.New("draw.io export cannot prepare while the host buffer is in an edit group")
	ErrViewExportPending   = errors.New("draw.io export cannot prepare while a field draft is pending")
	ErrViewExportPageIndex = errors.New("draw.io export page index is outside 0..249999")
	ErrViewExportPageID    = errors.New("draw.io export page ID exceeds 512 UTF-8 bytes")
	ErrViewExportChanged   = errors.New("draw.io view changed while preparing the export request")
	ErrViewExportDocument  = errors.New("draw.io export document is not readable")
)

var (
	_ editor.ExportRequestPreparer      = (*view)(nil)
	_ editor.ExportRequestTokenProvider = (*view)(nil)
)

// ExportRequestToken reports the current invalidation epoch without
// synchronizing the cache or touching the document. Callers compare this value
// before publish; preparation returns the same value it accepted.
func (v *view) ExportRequestToken() editor.ExportToken {
	return editor.ExportToken(v.exportEpoch)
}

// PrepareExportRequest refreshes a stale parsed cache and copies the selected
// page identity. It never flushes a field draft, edits source, or chooses a
// fallback page. The host must have completed its own Flush before calling it.
func (v *view) PrepareExportRequest() (editor.ViewExportRequest, error) {
	if v.kind != kindDrawio {
		return editor.ViewExportRequest{}, ErrViewExportKind
	}
	buffer := v.host.Buffer
	if buffer == nil {
		return editor.ViewExportRequest{}, ErrViewExportNoBuffer
	}
	if buffer.InGroup() {
		return editor.ViewExportRequest{}, ErrViewExportGroup
	}
	if err := v.previewSizeError(); err != nil {
		return editor.ViewExportRequest{}, err
	}
	if v.pendingGraphDraft() {
		return editor.ViewExportRequest{}, ErrViewExportPending
	}

	version, released, pageEpoch := buffer.Version(), v.releaseEpoch, v.pageEpoch
	pageIndex := v.pageIndex
	pageID, pageKnown := "", false
	if page := v.currentPage(); page != nil {
		pageID, pageKnown = page.ID, true
	}

	v.ensureSynced(nil)

	if v.host.Buffer != buffer || buffer.Version() != version || v.releaseEpoch != released ||
		v.pageEpoch != pageEpoch || v.pageIndex != pageIndex {
		return editor.ViewExportRequest{}, ErrViewExportChanged
	}
	if buffer.InGroup() {
		return editor.ViewExportRequest{}, ErrViewExportGroup
	}
	if v.pendingGraphDraft() {
		return editor.ViewExportRequest{}, ErrViewExportPending
	}
	if v.syncError != nil {
		return editor.ViewExportRequest{}, v.syncError
	}
	if v.diagram == nil {
		return editor.ViewExportRequest{}, ErrViewExportDocument
	}
	if pageKnown {
		page := v.currentPage()
		if page == nil || page.ID != pageID {
			return editor.ViewExportRequest{}, ErrViewExportChanged
		}
	}
	if pageIndex < 0 || pageIndex > maxViewExportPageIndex {
		return editor.ViewExportRequest{}, fmt.Errorf("%w: %d", ErrViewExportPageIndex, pageIndex)
	}
	if pageIndex >= len(v.diagram.Pages) {
		return editor.ViewExportRequest{}, fmt.Errorf("%w: %d", drawio.ErrPageNotFound, pageIndex)
	}
	page := v.diagram.Pages[pageIndex]
	if page.Index < 0 || page.Index > maxViewExportPageIndex {
		return editor.ViewExportRequest{}, fmt.Errorf("%w: %d", ErrViewExportPageIndex, page.Index)
	}
	if len(page.ID) > maxViewExportPageIDBytes {
		return editor.ViewExportRequest{}, fmt.Errorf("%w: %d", ErrViewExportPageID, len(page.ID))
	}
	return editor.ViewExportRequest{
		Page:  editor.ExportPage{Index: page.Index, ID: page.ID},
		Token: v.ExportRequestToken(),
	}, nil
}

func (v *view) bumpExportEpoch() {
	v.exportEpoch++
}

// setPageIndex records every actual selected-page transition, including
// A -> B -> A. Assigning the same index is not a transition.
func (v *view) setPageIndex(index int) {
	if v.pageIndex == index {
		return
	}
	v.pageIndex = index
	v.pageEpoch++
	v.bumpExportEpoch()
}
