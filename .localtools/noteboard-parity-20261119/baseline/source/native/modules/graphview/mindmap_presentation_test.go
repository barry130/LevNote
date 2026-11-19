package graphview

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/mindmap"
	"levnote.local/native/ui/editor"
)

// The expected bytes/metadata came from the retained pre-extraction graphview
// run. This test reads them; it has no mode for rewriting the oracle.
func TestMindmapPresentationRetainedBaselineAndIdle(t *testing.T) {
	for _, name := range []string{"right-rich", "left-plain", "negative-unsafe", "blank-long"} {
		t.Run(name, func(t *testing.T) {
			read := func(ext string) []byte {
				t.Helper()
				data, err := os.ReadFile(filepath.Join("..", "..", "formats", "mindmap", "testdata", "presentation", name+ext))
				if err != nil {
					t.Fatal(err)
				}
				return data
			}
			source, expected := read(".mindmap"), read(".svg")
			var meta struct {
				Bounds   mindmap.Rect
				Warnings []string
			}
			if err := json.Unmarshal(read(".json"), &meta); err != nil {
				t.Fatal(err)
			}
			buffer := text.New(string(source))
			v := newView(kindMindmap, editor.Host{Buffer: buffer})
			v.ensureSynced(nil)
			if v.syncFailed {
				t.Fatal("retained source did not parse", v.Warnings())
			}
			if !reflect.DeepEqual(v.layout, mindmap.Layout(v.mind, mindmap.NativeLayoutOptions())) {
				t.Fatal("canvas layout differs from shared native options")
			}
			if name == "negative-unsafe" {
				v.layout.Bounds = meta.Bounds
			}
			if v.layout.Bounds != meta.Bounds || !reflect.DeepEqual(v.Warnings(), meta.Warnings) {
				t.Fatalf("retained bounds/warning order changed: %+v %q", v.layout.Bounds, v.Warnings())
			}
			version, history, selection := buffer.Version(), buffer.History(), buffer.Selection()
			svg, err := v.Export("svg")
			if err != nil || !bytes.Equal(svg, expected) {
				t.Fatal("retained SVG bytes changed", err)
			}
			for _, scene := range v.layout.Nodes {
				if boundedMindmapLabel(scene.Node, scene.Text) != mindmap.DisplayLabel(scene.Node, scene.Text) {
					t.Fatal("canvas and exported label decision differ")
				}
			}
			tester := ui.NewTester(v.Draw, 900, 620)
			tester.Frame()
			doc, draft := v.mind, v.draft
			firstScene := &v.layout.Nodes[0]
			tester.Frame()
			if v.mind != doc || v.draft != draft || &v.layout.Nodes[0] != firstScene || v.frameLabels != 0 || tester.FramePending() {
				t.Fatal("idle frames reparse/layout/shape or invalidate")
			}
			serialized, err := v.mind.Serialize(mindmap.SerializeOptions{})
			if err != nil || !bytes.Equal(serialized, source) || buffer.String() != string(source) ||
				buffer.Version() != version || buffer.History() != history || buffer.Selection() != selection {
				t.Fatal("shared presentation changed source/history/selection", err)
			}
		})
	}
}
