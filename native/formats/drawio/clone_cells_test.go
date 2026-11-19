package drawio_test

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"levnote.local/native/formats/drawio"
)

const cloneCellsXML = `<mxfile custom="keep"><diagram id="page" name="克隆"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="a" parent="1" title="第一" vertex="1"><mxGeometry x="20.50" y="30" width="80.00" height="40" as="geometry"/></mxCell><!--原边注--><mxCell id="b" parent="1" vertex="1"><mxGeometry x='180' y='100.5' width='100' height='60' as='geometry'/></mxCell><mxCell id="c" parent="1" vertex="1"><mxGeometry x="320" y="0" width="40" height="40" as="geometry"/></mxCell><mxCell id="internal" edge="1" parent="1" source="a" target="b"><mxGeometry relative="1" as="geometry"/></mxCell><mxCell id="outer" edge="1" parent="1" source="a" target="c"><mxGeometry relative="1" as="geometry"/></mxCell><object id="levnote-1" vendor="preserved"/><custom data="unchanged"/></root></mxGraphModel></diagram></mxfile>`

func TestCloneCellsPreservesOriginalSpansAndRewiresInternalEdge(t *testing.T) {
	doc, _, err := drawio.Parse([]byte(cloneCellsXML), drawio.ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	ids, err := doc.CloneCells(0, []string{"a", "b"}, 10.25, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 2 || ids[0] != "levnote-2" || ids[1] != "levnote-3" || !doc.Dirty() {
		t.Fatalf("ID allocation/selection return failed: %#v dirty=%t", ids, doc.Dirty())
	}
	after, _ := doc.Serialize()
	for _, original := range []string{
		`<mxCell id="a" parent="1" title="第一" vertex="1"><mxGeometry x="20.50" y="30" width="80.00" height="40" as="geometry"/></mxCell>`,
		"<!--原边注-->",
		`<mxCell id="outer" edge="1" parent="1" source="a" target="c">`,
		`<object id="levnote-1" vendor="preserved"/><custom data="unchanged"/>`,
	} {
		if !strings.Contains(string(after), original) {
			t.Fatalf("original/unknown XML changed: %q", original)
		}
	}
	for _, clone := range []string{
		`<mxCell id="levnote-2" parent="1" title="第一" vertex="1"><mxGeometry x="30.75" y="50" width="80.00" height="40" as="geometry"/></mxCell>`,
		`<mxCell id="levnote-3" parent="1" vertex="1"><mxGeometry x='190.25' y='120.5' width='100' height='60' as='geometry'/></mxCell>`,
		`<mxCell id="levnote-4" edge="1" parent="1" source="levnote-2" target="levnote-3">`,
	} {
		if !strings.Contains(string(after), clone) {
			t.Fatalf("cloned bytes changed beyond id/geometry: %q", clone)
		}
	}
	parsed, _, err := drawio.Parse(after, drawio.ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.Pages[0].Model.Nodes) != 5 || len(parsed.Pages[0].Model.Edges) != 3 {
		t.Fatalf("wrong cloned graph sizes: %d nodes/%d edges", len(parsed.Pages[0].Model.Nodes), len(parsed.Pages[0].Model.Edges))
	}
	if strings.Count(string(after), `id="outer"`) != 1 || strings.Count(string(after), `id="internal"`) != 1 {
		t.Fatal("outer edge duplicated or original internal edge lost")
	}
}

func TestCloneCellsEmitsVerticesInSelectionOrderThenInternalEdges(t *testing.T) {
	doc, _, err := drawio.Parse([]byte(cloneCellsXML), drawio.ParseOptions{})
	if err != nil { t.Fatal(err) }
	ids, err := doc.CloneCells(0, []string{"b", "a"}, 10.25, 20)
	if err != nil { t.Fatal(err) }
	if len(ids) != 2 || ids[0] != "levnote-2" || ids[1] != "levnote-3" { t.Fatalf("selection-return order: %v", ids) }
	after, _ := doc.Serialize()
	wantB := `<mxCell id="levnote-2" parent="1" vertex="1"><mxGeometry x='190.25' y='120.5' width='100' height='60' as='geometry'/></mxCell>`
	wantA := `<mxCell id="levnote-3" parent="1" title="第一" vertex="1"><mxGeometry x="30.75" y="50" width="80.00" height="40" as="geometry"/></mxCell>`
	wantE := `<mxCell id="levnote-4" edge="1" parent="1" source="levnote-3" target="levnote-2">`
	posB, posA, posE := strings.Index(string(after), wantB), strings.Index(string(after), wantA), strings.Index(string(after), wantE)
	if posB < 0 || posA <= posB || posE <= posA { t.Fatalf("clone order must be requested vertices, then internal edges: %d/%d/%d", posB, posA, posE) }
}

func TestCloneCellsFailureLeavesSourceOriginalAndClean(t *testing.T) {
	port := strings.Replace(cloneCellsXML, `id="outer" edge="1"`, `id="outer" style="targetPort=a;targetPort=b;" edge="1"`, 1)
	cases := []struct {
		name, source string
		ids []string
		dx, dy float64
	}{
		{"empty", cloneCellsXML, nil, 1, 1},
		{"duplicate", cloneCellsXML, []string{"a", "a"}, 1, 1},
		{"unknown", cloneCellsXML, []string{"a", "missing"}, 1, 1},
		{"edge", cloneCellsXML, []string{"a", "internal"}, 1, 1},
		{"page-missing", cloneCellsXML, []string{"a", "b"}, 1, 1},
		{"port-style", port, []string{"a", "b"}, 1, 1},
		{"nan", cloneCellsXML, []string{"a", "b"}, math.NaN(), 1},
		{"overflow", cloneCellsXML, []string{"a", "b"}, math.Inf(1), 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc, _, err := drawio.Parse([]byte(tc.source), drawio.ParseOptions{})
			if err != nil {
				t.Fatal(err)
			}
			before, _ := doc.Serialize()
			page := 0
			if tc.name == "page-missing" {
				page = 9
			}
			ids, err := doc.CloneCells(page, tc.ids, tc.dx, tc.dy)
			if err == nil || ids != nil {
				t.Fatalf("invalid clone accepted: %#v %v", ids, err)
			}
			after, _ := doc.Serialize()
			if !bytes.Equal(after, before) || doc.Dirty() {
				t.Fatal("failed clone staged partial bytes or dirty state")
			}
		})
	}
}
