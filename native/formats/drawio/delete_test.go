package drawio_test

import (
	"bytes"
	"testing"

	"levnote.local/native/formats/drawio"
)

func TestDeleteCellPreservesUnrelatedSource(t *testing.T) {
	original := []byte(`<?xml version="1.0"?>
<mxGraphModel custom="keep-model"><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="a" value="delete" vertex="1" parent="1"><mxGeometry x="20" y="20" width="80" height="40" as="geometry"/></mxCell><!-- keep this comment --><mxCell id="b" custom="keep-cell" value="keep" vertex="1" parent="1"><mxGeometry x="160" y="20" width="80" height="40" as="geometry"/></mxCell></root></mxGraphModel>`)
	want := []byte(`<?xml version="1.0"?>
<mxGraphModel custom="keep-model"><root><mxCell id="0"/><mxCell id="1" parent="0"/><!-- keep this comment --><mxCell id="b" custom="keep-cell" value="keep" vertex="1" parent="1"><mxGeometry x="160" y="20" width="80" height="40" as="geometry"/></mxCell></root></mxGraphModel>`)

	document, _, err := drawio.Parse(original, drawio.ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	incidentEdges, err := document.DeleteCell(0, "a")
	if err != nil {
		t.Fatal(err)
	}
	if incidentEdges != 0 {
		t.Fatalf("removed incident edges = %d, want 0", incidentEdges)
	}
	serialized, err := document.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(serialized, want) {
		t.Fatalf("delete changed bytes outside the selected cell:\n%s", serialized)
	}
	if !document.Dirty() || !bytes.Equal(document.OriginalBytes(), original) {
		t.Fatal("delete did not retain the exact pristine source separately")
	}
	if len(document.Pages) != 1 || len(document.Pages[0].Model.Nodes) != 1 || document.Pages[0].Model.Nodes[0].ID != "b" {
		t.Fatal("serialized deletion was not published as the current model")
	}
}

func TestDeleteCellRemovesIncidentEdgesOnce(t *testing.T) {
	original := []byte(`<mxfile custom="keep"><diagram id="first" name="First"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="a" value="delete" vertex="1" parent="1"><mxGeometry width="80" height="80" as="geometry"/></mxCell><!-- keep gap --><mxCell id="b" value="keep" vertex="1" parent="1"><mxGeometry x="160" width="80" height="80" as="geometry"/></mxCell><mxCell id="out" edge="1" parent="1" source="a" target="b"><mxGeometry relative="1" as="geometry"/></mxCell><mxCell id="in" edge="1" parent="1" source="b" target="a"><mxGeometry relative="1" as="geometry"/></mxCell><mxCell id="loop" edge="1" parent="1" source="a" target="a"><mxGeometry relative="1" as="geometry"/></mxCell><mxCell id="keep-edge" custom="keep-edge" edge="1" parent="1" source="b" target="b"><mxGeometry relative="1" as="geometry"/></mxCell></root></mxGraphModel></diagram><diagram id="other" name="Other"><mxGraphModel custom="other-page"><root><mxCell id="0"/><mxCell id="1" parent="0"/><object id="opaque" parent="a"/></root></mxGraphModel></diagram></mxfile>`)
	want := []byte(`<mxfile custom="keep"><diagram id="first" name="First"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><!-- keep gap --><mxCell id="b" value="keep" vertex="1" parent="1"><mxGeometry x="160" width="80" height="80" as="geometry"/></mxCell><mxCell id="keep-edge" custom="keep-edge" edge="1" parent="1" source="b" target="b"><mxGeometry relative="1" as="geometry"/></mxCell></root></mxGraphModel></diagram><diagram id="other" name="Other"><mxGraphModel custom="other-page"><root><mxCell id="0"/><mxCell id="1" parent="0"/><object id="opaque" parent="a"/></root></mxGraphModel></diagram></mxfile>`)

	document, _, err := drawio.Parse(original, drawio.ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	incidentEdges, err := document.DeleteCell(0, "a")
	if err != nil {
		t.Fatalf("known incident edges were not removed atomically: %v", err)
	}
	if incidentEdges != 3 {
		t.Fatalf("removed incident edges = %d, want 3 with the self-loop counted once", incidentEdges)
	}
	serialized, err := document.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(serialized, want) || !bytes.Equal(document.OriginalBytes(), original) {
		t.Fatalf("incident-edge deletion changed unrelated cells, comments or the other page:\n%s", serialized)
	}
	if len(document.Pages) != 2 || len(document.Pages[0].Model.Nodes) != 1 || len(document.Pages[0].Model.Edges) != 1 || document.Pages[0].Model.Edges[0].ID != "keep-edge" {
		t.Fatal("incident-edge deletion did not publish exactly the expected cells")
	}
}
