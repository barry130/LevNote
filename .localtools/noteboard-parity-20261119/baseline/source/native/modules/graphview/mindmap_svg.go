package graphview

import "levnote.local/native/formats/mindmap"

func exportMindmapSVG(document *mindmap.Document, layout mindmap.LayoutResult) ([]byte, error) {
	return mindmap.ExportSVG(document, layout, 0)
}
