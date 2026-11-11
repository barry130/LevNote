package graphview

import (
	"fmt"

	"levnote.local/native/formats/drawio"
	"levnote.local/native/formats/mindmap"
)

var errMindmapPreviewTooLarge = fmt.Errorf("思维导图超过 %d MiB 预览上限，完整原文仍保留；请切到源码视图：%w", mindmap.DefaultMaxBytes>>20, mindmap.ErrLimitExceeded)
var errDrawioPreviewTooLarge = fmt.Errorf("draw.io 超过 %d MiB 预览上限，完整原文仍保留；请切到源码视图：%w", drawio.DefaultMaxBytes>>20, drawio.ErrLimitExceeded)

// previewSizeError is shared by parse entry points, including pending field
// Flush, which must parse the host document to safely locate its target node.
func (v *view) previewSizeError() error {
	if v.host.Buffer == nil {
		return nil
	}
	if v.kind == kindMindmap && v.host.Buffer.LenBytes() > mindmap.DefaultMaxBytes {
		return errMindmapPreviewTooLarge
	}
	if v.kind == kindDrawio && v.host.Buffer.LenBytes() > drawio.DefaultMaxBytes {
		return errDrawioPreviewTooLarge
	}
	return nil
}

func (v *view) clearParsedCache() {
	if v.mind != nil || v.diagram != nil {
		v.bumpExportEpoch()
	}
	v.mind, v.diagram = nil, nil
	v.layout = mindmap.LayoutResult{}
	v.clearRenderCache()
	v.pendingOutlineCenter = ""
	if v.outline != nil {
		v.outline.Release()
		v.outline = nil
	}
}
