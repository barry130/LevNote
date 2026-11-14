package drawio_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"levnote.local/native/formats/drawio"
)

func TestDeleteCellRefusesSurvivingDuplicatePortReference(t *testing.T) {
	original := []byte(`<mxfile host="keep-host"><diagram id="ports" name="Port references"><mxGraphModel custom="keep-model"><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="a" value="delete" vertex="1" parent="1"><mxGeometry width="80" height="80" as="geometry"/></mxCell><!-- keep gap --><mxCell id="b" custom="keep-cell" value="keep" vertex="1" parent="1"><mxGeometry x="160" width="80" height="80" as="geometry"/></mxCell><mxCell id="survivor" custom="keep-edge" edge="1" parent="1" source="b" target="b" style="sourcePort=b;targetPort=b;targetPort=a;endArrow=classic;custom=keep;"><mxGeometry relative="1" as="geometry"/></mxCell></root></mxGraphModel></diagram></mxfile>`)
	document, issues, err := drawio.Parse(original, drawio.ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if document.Dirty() || len(document.Pages) != 1 || document.Pages[0].ReadOnly ||
		len(document.Pages[0].Model.Nodes) != 2 || len(document.Pages[0].Model.Edges) != 1 {
		t.Fatal("fixture is not an initially editable page with two vertices and one surviving edge")
	}
	survivor := document.Pages[0].Model.Edges[0]
	if survivor.ID != "survivor" || survivor.Parent != "1" || survivor.SourceID != "b" ||
		survivor.TargetID != "b" || len(survivor.Unsupported) != 0 {
		t.Fatalf("fixture does not contain an understood non-incident b-to-b surviving edge: %+v", survivor)
	}
	for _, issue := range issues {
		if issue.Code == drawio.IssueUnsupportedElement {
			t.Fatalf("fixture was refused before its duplicate port reference could be checked: %+v", issue)
		}
	}
	// Snapshot the public model by value, not by retaining mutable page slices.
	beforePages, err := json.Marshal(document.Pages)
	if err != nil {
		t.Fatal(err)
	}
	// The edge's source and target remain b. Its second targetPort still names
	// a, so deleting a would orphan a surviving raw reference regardless of
	// which duplicate style key a renderer chooses to display.
	incidentEdges, deletionErr := document.DeleteCell(0, "a")
	serialized, err := document.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	if !errors.Is(deletionErr, drawio.ErrUnsupported) {
		t.Fatalf("surviving duplicate port reference did not refuse deletion: incidentEdges=%d err=%v dirty=%v source=%q", incidentEdges, deletionErr, document.Dirty(), serialized)
	}
	if incidentEdges != 0 || document.Dirty() || !bytes.Equal(serialized, original) ||
		!bytes.Equal(document.OriginalBytes(), original) || !bytes.Equal(document.Source, original) {
		t.Fatalf("refused deletion published source, dirty state or removal count: incidentEdges=%d dirty=%v source=%q", incidentEdges, document.Dirty(), serialized)
	}
	afterPages, err := json.Marshal(document.Pages)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(afterPages, beforePages) {
		t.Fatalf("refused deletion changed the public page model: before=%s after=%s", beforePages, afterPages)
	}
}
