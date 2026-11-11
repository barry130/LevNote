package mindmap

import "testing"

func TestCollapsedRootVisibleSetAcrossLayouts(t *testing.T) {
	for _, layout := range []string{LayoutBoth, LayoutLeft, LayoutRight, LayoutOrg} {
		t.Run(layout, func(t *testing.T) {
			collapsed := true
			doc := &Document{Root: NewNode("root", "中心"), Layout: layout, Version: Version}
			doc.Root.Children = []*Node{NewNode("child", "分支")}
			doc.Root.Children[0].Children = []*Node{NewNode("leaf", "叶子")}
			doc.Root.Collapsed = &collapsed
			folded := Layout(doc, LayoutOptions{})
			if len(folded.Nodes) != 1 || folded.Nodes[0].ID != "root" || len(folded.Edges) != 0 {
				t.Fatalf("folded root leaked descendants: %+v", folded.Nodes)
			}
			if err := doc.SetCollapsed("root", false); err != nil {
				t.Fatal(err)
			}
			visible := Layout(doc, LayoutOptions{})
			if len(visible.Nodes) != 3 || len(visible.Edges) != 2 {
				t.Fatalf("expanded visible set: %+v", visible.Nodes)
			}
		})
	}
}
