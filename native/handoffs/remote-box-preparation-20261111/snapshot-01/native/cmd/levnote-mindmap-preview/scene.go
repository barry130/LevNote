package main

import (
	"encoding/json"

	"levnote.local/native/core/enginescene"
	"levnote.local/native/formats/mindmap"
)

// previewScene copies only neutral presentation values. It retains no model
// pointers, source metadata, image references, or host theme/selection state.
func previewScene(document *mindmap.Document, layout mindmap.LayoutResult) enginescene.Scene {
	scene := enginescene.Scene{
		Schema: enginescene.SchemaVersion, Layout: layout.Kind,
		Bounds: enginescene.Rect{
			X: layout.Bounds.X, Y: layout.Bounds.Y,
			Width: layout.Bounds.Width, Height: layout.Bounds.Height,
		},
		Nodes: make([]enginescene.Node, len(layout.Nodes)),
		Edges: make([]enginescene.Edge, len(layout.Edges)),
	}
	rootID := document.RootID()
	for i, node := range layout.Nodes {
		scene.Nodes[i] = enginescene.Node{
			ID: node.ID, Label: mindmap.DisplayLabel(node.Node, node.Text),
			X: node.X, Y: node.Y, Width: node.Width, Height: node.Height,
			Shape: node.Shape, Root: node.ID == rootID, Collapsed: node.Collapsed,
			Fill:   nodePaint(node.Node, "background-color"),
			Stroke: nodePaint(node.Node, "border"), Text: nodePaint(node.Node, "color"),
		}
	}
	for i, edge := range layout.Edges {
		scene.Edges[i] = enginescene.Edge{
			FromID: edge.From, ToID: edge.To,
			From:   enginescene.Point{X: edge.FromPoint.X, Y: edge.FromPoint.Y},
			To:     enginescene.Point{X: edge.ToPoint.X, Y: edge.ToPoint.Y},
			Stroke: normalizedPaint(edge.Color),
		}
	}
	return scene
}

func nodePaint(node *mindmap.Node, key string) string {
	if node == nil || node.Style == nil {
		return ""
	}
	raw, ok := node.Style.Get(key)
	if !ok {
		return ""
	}
	var value string
	if json.Unmarshal(raw, &value) != nil {
		return ""
	}
	return normalizedPaint(value)
}

// LiteralColor is the shared admission rule. This helper changes only its
// successful spelling to the neutral DTO's lowercase eight-digit RGBA form.
func normalizedPaint(value string) string {
	literal, ok := mindmap.LiteralColor(value)
	if !ok {
		return ""
	}
	switch len(literal) {
	case 4, 5:
		out := [9]byte{'#', 0, 0, 0, 0, 0, 0, 'f', 'f'}
		for i := 1; i < len(literal); i++ {
			out[i*2-1], out[i*2] = literal[i], literal[i]
		}
		return string(out[:])
	case 7:
		return literal + "ff"
	case 9:
		return literal
	default:
		return ""
	}
}
