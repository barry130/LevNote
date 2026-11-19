package drawio_test

import (
	"bytes"
	"reflect"
	"testing"

	"levnote.local/native/formats/drawio"
)

func TestParseGeometryPointClosuresPreserveModelAndBytes(t *testing.T) {
	cases := []struct {
		name           string
		xml            string
		source, target *drawio.Point
		points         []drawio.Point
	}{
		{name: "source-self-closing", xml: `<mxPoint x='12.5' y="-7.25" as="sourcePoint"/>`, source: &drawio.Point{X: 12.5, Y: -7.25}},
		{name: "source-expanded", xml: `<mxPoint x='12.5' y="-7.25" as="sourcePoint"></mxPoint>`, source: &drawio.Point{X: 12.5, Y: -7.25}},
		{name: "target-self-closing", xml: `<mxPoint x="120" y='90' as="targetPoint" />`, target: &drawio.Point{X: 120, Y: 90}},
		{name: "target-expanded", xml: `<mxPoint x="120" y='90' as="targetPoint"> <!-- keep --> </mxPoint>`, target: &drawio.Point{X: 120, Y: 90}},
		{name: "offset-self-closing", xml: `<mxPoint x="-3" y="4.5" as="offset" future="0007"/>`, points: []drawio.Point{{X: -3, Y: 4.5}}},
		{name: "offset-expanded", xml: `<mxPoint x="-3" y="4.5" as="offset" future="0007"></mxPoint>`, points: []drawio.Point{{X: -3, Y: 4.5}}},
		{
			name:   "adjacent-self-closing",
			xml:    `<mxPoint x="12.5" y="-7.25" as="sourcePoint"/><mxPoint x="120" y="90" as="targetPoint"/><mxPoint x="-3" y="4.5" as="offset"/>`,
			source: &drawio.Point{X: 12.5, Y: -7.25}, target: &drawio.Point{X: 120, Y: 90}, points: []drawio.Point{{X: -3, Y: 4.5}},
		},
		{
			name:   "adjacent-expanded",
			xml:    `<mxPoint x="12.5" y="-7.25" as="sourcePoint"></mxPoint><mxPoint x="120" y="90" as="targetPoint"></mxPoint><mxPoint x="-3" y="4.5" as="offset"></mxPoint>`,
			source: &drawio.Point{X: 12.5, Y: -7.25}, target: &drawio.Point{X: 120, Y: 90}, points: []drawio.Point{{X: -3, Y: 4.5}},
		},
		{
			name:   "adjacent-mixed",
			xml:    `<mxPoint x="12.5" y="-7.25" as="sourcePoint"/><mxPoint x="120" y="90" as="targetPoint"></mxPoint><mxPoint x="-3" y="4.5" as="offset"/>`,
			source: &drawio.Point{X: 12.5, Y: -7.25}, target: &drawio.Point{X: 120, Y: 90}, points: []drawio.Point{{X: -3, Y: 4.5}},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte("<?xml version='1.0'?>\r\n" +
				`<mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="e" edge="1" parent="1"><mxGeometry relative="1" as="geometry">` +
				tc.xml + `</mxGeometry></mxCell><mxCell id="after" vertex="1" parent="1"><mxGeometry x="1" y="2" width="3" height="4" as="geometry"/></mxCell></root></mxGraphModel>` + "\r\n")
			document, issues, err := drawio.Parse(source, drawio.ParseOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if len(issues) != 0 || len(document.Pages) != 1 {
				t.Fatalf("issues=%#v pages=%d", issues, len(document.Pages))
			}
			model := document.Pages[0].Model
			if len(model.Edges) != 1 || len(model.Nodes) != 1 {
				t.Fatalf("edges=%#v nodes=%#v", model.Edges, model.Nodes)
			}
			edge := model.Edges[0]
			// Offset retains the reader's existing generic Points representation.
			// This regression concerns element closure, not point-role semantics.
			if edge.ID != "e" || !reflect.DeepEqual(edge.SourcePoint, tc.source) || !reflect.DeepEqual(edge.TargetPoint, tc.target) || !reflect.DeepEqual(edge.Points, tc.points) {
				t.Fatalf("edge=%#v, want source=%#v target=%#v points=%#v", edge, tc.source, tc.target, tc.points)
			}
			node := model.Nodes[0]
			if node.ID != "after" || node.X != 1 || node.Y != 2 || node.Width != 3 || node.Height != 4 {
				t.Fatalf("following cell lost parser synchronization: %#v", node)
			}
			serialized, err := document.Serialize()
			if err != nil {
				t.Fatal(err)
			}
			if document.Dirty() || !bytes.Equal(serialized, source) || !bytes.Equal(document.OriginalBytes(), source) || !bytes.Equal(document.Source, source) {
				t.Fatalf("point closure changed bytes: dirty=%v\ngot %s\nwant %s", document.Dirty(), serialized, source)
			}
		})
	}
}
