package drawio_test

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"levnote.local/native/formats/drawio"
)

// The agreed public seam is CloneCells plus Serialize/OriginalBytes/Dirty and
// the public parsed model/issues: no internal offsets or scanners are tested.
func cloneCorrectionDocument(t *testing.T, source string) *drawio.Document {
	t.Helper()
	document, _, err := drawio.Parse([]byte(source), drawio.ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return document
}

func assertCloneCorrectionRefusal(t *testing.T, source string) {
	t.Helper()
	document := cloneCorrectionDocument(t, source)
	beforeSource := append([]byte(nil), document.Source...)
	beforeOriginal := document.OriginalBytes()
	beforeModel := append([]drawio.Page(nil), document.Pages...)
	beforeIssues := append([]drawio.Issue(nil), document.Issues...)
	ids, err := document.CloneCells(0, []string{"a", "b"}, 1, 2)
	if err == nil || ids != nil {
		t.Fatalf("CloneCells() = (%v, %v), want fatal refusal", ids, err)
	}
	serialized, err := document.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(serialized, beforeSource) ||
		!bytes.Equal(document.Source, beforeSource) ||
		!bytes.Equal(document.OriginalBytes(), beforeOriginal) ||
		!reflect.DeepEqual(document.Pages, beforeModel) ||
		!reflect.DeepEqual(document.Issues, beforeIssues) || document.Dirty() {
		t.Fatalf("fatal clone altered source/original/model/issues/dirty: dirty=%v\nsource=%s", document.Dirty(), serialized)
	}
}

func TestCloneCellsCorrection20261117RejectsCopiedPorts(t *testing.T) {
	for _, tc := range []struct{ name, cell string }{
		{name: "source-attribute-double-quote", cell: `<mxCell id="internal" edge="1" parent="1" source="a" target="b" sourcePort="unselected"/>`},
		{name: "target-attribute-single-quote", cell: `<mxCell id='internal' edge='1' parent='1' source='a' target='b' targetPort='x'/>`},
		{name: "empty-attribute", cell: `<mxCell id="internal" edge="1" parent="1" source="a" target="b" sourcePort=""/>`},
		{name: "duplicate-style-keys", cell: `<mxCell id="internal" edge="1" parent="1" source="a" target="b" style="sourcePort=not-selected;targetPort=other;sourcePort=;"></mxCell>`},
		{name: "style-without-value", cell: `<mxCell id="internal" edge="1" parent="1" source="a" target="b" style="targetPort;"/>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertCloneCorrectionRefusal(t, "<mxfile><diagram><mxGraphModel><root>"+
				`<mxCell id="0"/><mxCell id="1" parent="0"/>`+
				`<mxCell id="a" parent="1" vertex="1"><mxGeometry x="1" y="2" width="30" height="40" as="geometry"/></mxCell>`+
				`<mxCell id="b" parent="1" vertex="1"><mxGeometry x="50" y="60" width="70" height="80" as="geometry"/></mxCell>`+
				tc.cell+`</root></mxGraphModel></diagram></mxfile>`)
		})
	}
}

func cloneCorrectionEdgeXML(body string) string {
	return "<mxfile><diagram><mxGraphModel><root>" +
		`<mxCell id="0"/><mxCell id="1" parent="0"/>` +
		`<mxCell id="a" parent="1" vertex="1"><mxGeometry x="1" y="2" width="30" height="40" as="geometry"/></mxCell>` +
		`<mxCell id="b" parent="1" vertex="1"><mxGeometry x="50" y="60" width="70" height="80" as="geometry"/></mxCell>` +
		`<mxCell id="internal" edge="1" parent="1" source="a" target="b">` + body + `</mxCell>` +
		`</root></mxGraphModel></diagram></mxfile>`
}

func TestCloneCellsCorrection20261117RejectsInvalidInternalEdgeSubtrees(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{name: "missing-geometry", body: ""},
		{name: "duplicate-geometry", body: `<mxGeometry as="geometry"/><mxGeometry as="geometry"/>`},
		{name: "unknown-cell-child", body: `<mxGeometry as="geometry"/><future port="a"/>`},
		{name: "cell-text", body: "label" + `<mxGeometry as="geometry"/>`},
		{name: "geometry-text", body: `<mxGeometry as="geometry">label</mxGeometry>`},
		{name: "unknown-geometry-child", body: `<mxGeometry as="geometry"><future/></mxGeometry>`},
		{name: "unknown-array-role", body: `<mxGeometry as="geometry"><Array as="future"/></mxGeometry>`},
		{name: "unknown-point-child", body: `<mxGeometry as="geometry"><Array as="points"><mxPoint x="1" y="2"><future/></mxPoint></Array></mxGeometry>`},
		{name: "waypoint-text", body: `<mxGeometry as="geometry"><Array as="points">label<mxPoint x="1" y="2"/></Array></mxGeometry>`},
		{name: "invalid-waypoint", body: `<mxGeometry as="geometry"><Array as="points"><mxPoint x="NaN" y="2"/></Array></mxGeometry>`},
		{name: "unknown-geometry-role", body: `<mxGeometry as="future"/>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assertCloneCorrectionRefusal(t, cloneCorrectionEdgeXML(tc.body))
		})
	}
}

func TestCloneCellsCorrection20261117PreservesValidEdgeGeometryAndWaypoints(t *testing.T) {
	const geometry = `<mxGeometry relative="1" as="geometry" vendor="retained"><!-- keep --><mxPoint x='12.5' y="13.5" as="sourcePoint" vendor='first'/><mxPoint x="14" y='15' as="targetPoint"/><Array as='points'><mxPoint x="20.5" y="21.5" vendor="keep"/><mxPoint as='offset' x="1" y='2'/></Array><Array as='points'/><mxPoint as="offset" x="2" y='3'/></mxGeometry>`
	document := cloneCorrectionDocument(t, cloneCorrectionEdgeXML(geometry))
	ids, err := document.CloneCells(0, []string{"a", "b"}, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ids, []string{"levnote-1", "levnote-2"}) {
		t.Fatalf("new IDs = %#v", ids)
	}
	after, err := document.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	want := `<mxCell id="levnote-3" edge="1" parent="1" source="levnote-1" target="levnote-2">` + geometry + `</mxCell>`
	if !strings.Contains(string(after), want) {
		t.Fatalf("legal edge bytes/values changed; want retained span %q\nafter=%s", want, after)
	}
}

func TestCloneCellsCorrection20261117AvoidsRawIDsAcrossOnlySelectedDiagram(t *testing.T) {
	const source = "<mxfile><diagram id=\"selected\">" +
		"<meta id=\"levnote-1\"/><mxGraphModel><root>" +
		"<mxCell id=\"0\"/><mxCell id=\"1\" parent=\"0\"/>" +
		"<mxCell id=\"a\" parent=\"1\" vertex=\"1\"><mxGeometry x=\"1\" y=\"2\" width=\"30\" height=\"40\" as=\"geometry\"/></mxCell>" +
		"<mxCell id=\"b\" parent=\"1\" vertex=\"1\"><mxGeometry x=\"50\" y=\"60\" width=\"70\" height=\"80\" as=\"geometry\"/></mxCell>" +
		"<mxCell id=\"internal\" edge=\"1\" parent=\"1\" source=\"a\" target=\"b\"><mxGeometry as=\"geometry\"><Array as=\"points\"/></mxGeometry></mxCell>" +
		"<object id='levnote-2'/><wrapper><item id=\"levnote-3\"/></wrapper>" +
		"</root></mxGraphModel><vendor id='levnote-5'/></diagram>" +
		"<diagram><meta id='levnote-4'/><meta id=\"levnote-7\"/></diagram></mxfile>"
	document := cloneCorrectionDocument(t, source)
	ids, err := document.CloneCells(0, []string{"a", "b"}, 3, 4)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ids, []string{"levnote-4", "levnote-6"}) {
		t.Fatalf("selected diagram raw IDs were not fully reserved: %v", ids)
	}
	after, err := document.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	for _, retained := range []string{"<vendor id='levnote-5'/>", "id=\"levnote-7\" edge=\"1\" parent=\"1\" source=\"levnote-4\" target=\"levnote-6\""} {
		if !strings.Contains(string(after), retained) {
			t.Errorf("missing retained/allocated raw span: %q", retained)
		}
	}
}
func TestCloneCellsCorrection20261117StandaloneModel(t *testing.T) {
	const source = `<mxGraphModel vendor="keep"><root><mxCell id="0"/><mxCell id="1" parent="0"/><!--before--><mxCell id="a" vertex="1" parent="1"><mxGeometry x="20" y="30" width="80" height="40" as="geometry"/></mxCell></root></mxGraphModel>`
	document := cloneCorrectionDocument(t, source)
	ids, err := document.CloneCells(0, []string{"a"}, 10, 5)
	if err != nil { t.Fatal(err) }
	after, err := document.Serialize()
	if err != nil { t.Fatal(err) }
	want := "<!--before-->" + `<mxCell id="a" vertex="1" parent="1"><mxGeometry x="20" y="30" width="80" height="40" as="geometry"/></mxCell>` + `<mxCell id="levnote-1" vertex="1" parent="1"><mxGeometry x="30" y="35" width="80" height="40" as="geometry"/></mxCell></root></mxGraphModel>`
	if !bytes.Contains(after, []byte(want)) || len(ids) != 1 || ids[0] != "levnote-1" || !document.Dirty() { t.Fatalf("standalone clone lost exact insertion anchor: %v\n%s", ids, after) }
}

func TestCloneCellsCorrection20261117RejectsEmptyRoot(t *testing.T) {
	document := cloneCorrectionDocument(t, `<mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="a" parent="1" vertex="1"><mxGeometry x="0" y="0" width="1" height="1" as="geometry"/></mxCell></root><root><mxCell id="b"/></root></mxGraphModel>`)
	ids, err := document.CloneCells(0, []string{"a", "b"}, 1, 2)
	if err == nil { t.Fatalf("ambiguous root accepted: %v", ids) }
	assertCloneCorrectionRefusal(t, `<mxfile><diagram><mxGraphModel><root></root></mxGraphModel></diagram></mxfile>`)
}
