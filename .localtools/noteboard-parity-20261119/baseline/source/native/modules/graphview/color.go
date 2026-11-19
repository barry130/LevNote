package graphview

import (
	"github.com/egoist/mygo/ui"
	"levnote.local/native/formats/mindmap"
)

// Keep the existing draw.io/native color subset while sharing pure admission.
func literalGraphColor(value string) (string, bool) {
	return mindmap.LiteralColor(value)
}

func graphColor(value string) (ui.Color, bool) {
	literal, ok := literalGraphColor(value)
	if !ok {
		return ui.Color{}, false
	}
	return ui.Hex(literal), true
}
