package drawio_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"levnote.local/native/formats/drawio"
)

const multiDeleteXML = `<mxfile host="keep"><diagram id="p1"><mxGraphModel custom="keep-model"><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="a" value="A" vertex="1" parent="1" style="rounded=1;"><mxGeometry x="20" y="20" width="80" height="40" as="geometry"/></mxCell><!--keep--><mxCell id="b" value="B" vertex="1" parent="1"><mxGeometry x="150" y="30" width="80" height="40" as="geometry"/></mxCell><mxCell id="c" custom="keep" value="C" vertex="1" parent="1"><mxGeometry x="320" y="20" width="80.00" height="40" as="geometry"/></mxCell><mxCell id="inner" edge="1" parent="1" source="a" target="b"><mxGeometry relative="1" as="geometry"/></mxCell><mxCell id="outer" edge="1" parent="1" source="b" target="c"><mxGeometry relative="1" as="geometry"/></mxCell><mxCell id="c-loop" edge="1" parent="1" source="c" target="c"><mxGeometry relative="1" as="geometry"/></mxCell><unknown note='exact'/></root></mxGraphModel></diagram><diagram id="p2"><mxGraphModel><root><object id="opaque" parent="a"/></root></mxGraphModel></diagram></mxfile>`

func TestDeleteCellsRemovesSelectionAndIncidentEdgesAtomically(t *testing.T) {
	original := []byte(multiDeleteXML)
	document, _, err := drawio.Parse(original, drawio.ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	incident, err := document.DeleteCells(0, []string{"b", "a"})
	if err != nil {
		t.Fatal(err)
	}
	if incident != 2 {
		t.Fatalf("incident edges = %d, want unique 2", incident)
	}
	got, err := document.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	for _, absent := range []string{`id="a"`, `id="b"`, `id="inner"`, `id="outer"`} {
		if bytes.Contains(got, []byte(absent)) {
			t.Fatalf("selected cell remains: %s", absent)
		}
	}
	for _, literal := range []string{"<!--keep-->", `custom="keep"`, `width="80.00"`, `id="c-loop"`, `<unknown note='exact'/>`, `<object id="opaque" parent="a"/>`} {
		if !bytes.Contains(got, []byte(literal)) {
			t.Fatalf("unrelated byte content lost: %s", literal)
		}
	}
	if !document.Dirty() || !bytes.Equal(document.OriginalBytes(), original) {
		t.Fatal("pristine source changed")
	}
	if len(document.Pages) != 2 || len(document.Pages[0].Model.Nodes) != 1 || document.Pages[0].Model.Nodes[0].ID != "c" || len(document.Pages[0].Model.Edges) != 1 {
		t.Fatalf("published model unexpectedly changed: %+v", document.Pages[0].Model)
	}
	if !strings.HasPrefix(string(got), strings.Replace(multiDeleteXML, `<mxCell id="a" value="A" vertex="1" parent="1" style="rounded=1;"><mxGeometry x="20" y="20" width="80" height="40" as="geometry"/></mxCell>`, "", 1)[:49]) {
		t.Fatal("unexpected document prefix rewrite")
	}
}

func TestDeleteCellsFailureLeavesWholeDocumentUntouched(t *testing.T) {
	for _, selected := range [][]string{{}, {"", "a"}, {"a", "a"}, {"0", "a"}, {"unknown", "a"}, {"a", "b", "outer"}} {
		document, _, err := drawio.Parse([]byte(multiDeleteXML), drawio.ParseOptions{})
		if err != nil {
			t.Fatal(err)
		}
		_, err = document.DeleteCells(0, selected)
		if err == nil {
			t.Fatalf("wanted refusal for %q", selected)
		}
		got, serializationErr := document.Serialize()
		if serializationErr != nil {
			t.Fatal(serializationErr)
		}
		if document.Dirty() || !bytes.Equal(got, []byte(multiDeleteXML)) || !bytes.Equal(document.OriginalBytes(), []byte(multiDeleteXML)) {
			t.Fatalf("failed deletion mutated document for %q: %v", selected, err)
		}
	}
}

func TestDeleteCellsRefusesSurvivingPortReference(t *testing.T) {
	original := `<mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="a" vertex="1" parent="1"><mxGeometry width="80" height="80" as="geometry"/></mxCell><mxCell id="b" vertex="1" parent="1"><mxGeometry width="80" height="80" as="geometry"/></mxCell><mxCell id="survivor" edge="1" parent="1" source="b" target="b" style="sourcePort=b;targetPort=b;targetPort=a;"><mxGeometry relative="1" as="geometry"/></mxCell></root></mxGraphModel>`
	document, _, err := drawio.Parse([]byte(original), drawio.ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := document.DeleteCells(0, []string{"a"}); !errors.Is(err, drawio.ErrUnsupported) {
		t.Fatalf("wanted unsupported refusal, got %v", err)
	}
	got, err := document.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	if document.Dirty() || string(got) != original {
		t.Fatal("failed raw-reference guard mutated bytes")
	}
}
