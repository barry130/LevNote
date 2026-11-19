// Package imageview presents one host-decoded PNG or JPEG in a native,
// read-only viewport. It never opens the document or decodes source bytes.
package imageview

import (
	"path/filepath"
	"strings"

	"levnote.local/native/ui/editor"
)

// Adapter returns the neutral image-reader adapter. Registration belongs to the
// application composition root after its binary document route is wired.
func Adapter() editor.Adapter {
	return editor.Adapter{
		ID:   "image",
		Name: "原生图片阅读器",
		Matches: func(path string) bool {
			switch strings.ToLower(filepath.Ext(path)) {
			case ".png", ".jpg", ".jpeg":
				return true
			default:
				return false
			}
		},
		Create: func(host editor.Host) editor.View {
			return newView(host)
		},
	}
}
