package drawio

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"
)

const richDrawio = "\r\n" + `<mxfile host="local" modified="2026-11-08T01:02:03.000Z">
  <diagram id="page-one" name="One">
    <mxGraphModel dx="800" dy="600" grid="1" gridSize="10" page="1">
      <root>
        <mxCell id="0"/>
        <mxCell id="1" parent="0"/>
        <mxCell id="a" value="A" style="rounded=1;fillColor=#ffe599;strokeColor=#8c6d1f;html=0;" vertex="1" parent="1">
          <mxGeometry x="10" y="20" width="120" height="60" as="geometry" />
        </mxCell>
        <mxCell id="b" value="B &amp; C" style="ellipse;whiteSpace=wrap;html=0;" vertex="1" parent="1">
          <mxGeometry x="220.5" y="30" width="80.25" height="50" as="geometry"/>
        </mxCell>
        <mxCell id="diamond" value="D" style="rhombus;html=0;" vertex="1" parent="1">
          <mxGeometry x="400" y="40" width="70" height="70" as="geometry"/>
        </mxCell>
        <mxCell id="html" value="&lt;b&gt;bold&lt;/b&gt;" style="html=1;rounded=0;" vertex="1" parent="1">
          <mxGeometry x="10" y="140" width="100" height="40" as="geometry"/>
        </mxCell>
        <mxCell id="image" value="picture" style="shape=image;image=data:image/png;base64,AAAA;" vertex="1" parent="1">
          <mxGeometry x="150" y="140" width="100" height="40" as="geometry"/>
        </mxCell>
        <mxCell id="stencil" value="stencil" style="shape=mxgraph.flowchart.process;" vertex="1" parent="1">
          <mxGeometry x="290" y="140" width="100" height="40" as="geometry"/>
        </mxCell>
        <mxCell id="edge" value="connects" style="endArrow=classic;html=0;strokeColor=#333333;" edge="1" parent="1" source="a" target="b">
          <mxGeometry relative="1" as="geometry"/>
        </mxCell>
        <future extension="123.00"><nested huge="123456789012345678901234567890"/></future>
      </root>
    </mxGraphModel>
  </diagram>
</mxfile>
`

func TestUncompressedParseModelAndExactRoundTrip(t *testing.T) {
	document, issues, err := Parse([]byte(richDrawio), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(document.Pages) != 1 {
		t.Fatalf("pages = %d", len(document.Pages))
	}
	page := document.Pages[0]
	if page.ID != "page-one" || page.Name != "One" || page.ReadOnly {
		t.Fatalf("page = %#v", page)
	}
	if len(page.Model.Nodes) != 6 || len(page.Model.Edges) != 1 {
		t.Fatalf("nodes=%d edges=%d", len(page.Model.Nodes), len(page.Model.Edges))
	}
	shapes := map[string]string{}
	for _, node := range page.Model.Nodes {
		shapes[node.ID] = node.Shape
	}
	if shapes["a"] != ShapeRounded || shapes["b"] != ShapeEllipse || shapes["diamond"] != ShapeDiamond {
		t.Fatalf("shapes = %#v", shapes)
	}
	if page.Model.Bounds.Width <= 0 || page.Model.Bounds.Height <= 0 {
		t.Fatalf("bounds = %#v", page.Model.Bounds)
	}
	if page.Model.Nodes[1].Text != "B & C" {
		t.Fatalf("XML entity was not decoded: %q", page.Model.Nodes[1].Text)
	}
	codes := issueCodes(issues)
	for _, code := range []IssueCode{IssueHTMLLabel, IssueImage, IssueStencil, IssueUnsupportedElement} {
		if !codes[code] {
			t.Fatalf("missing issue %s in %#v", code, issues)
		}
	}
	serialized, err := document.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(serialized, []byte(richDrawio)) {
		t.Fatal("untouched draw.io source changed bytes")
	}
}

func TestTextAndGeometryMutationsPreserveUnknownXML(t *testing.T) {
	document, _, err := Parse([]byte(richDrawio), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := document.UpdateText(0, "a", `new & <label>`); err != nil {
		t.Fatal(err)
	}
	if err := document.UpdateGeometry(0, "a", 11.25, 22.5, 130, 70); err != nil {
		t.Fatal(err)
	}
	serialized, err := document.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	text := string(serialized)
	for _, exact := range []string{
		`value="new &amp; &lt;label&gt;"`,
		`x="11.25"`,
		`y="22.5"`,
		`width="130"`,
		`height="70"`,
		`<future extension="123.00"><nested huge="123456789012345678901234567890"/></future>`,
		`  <diagram id="page-one" name="One">`,
	} {
		if !strings.Contains(text, exact) {
			t.Fatalf("edited source lost %q:\n%s", exact, text)
		}
	}
	reparsed, _, err := Parse(serialized, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	node := findNode(reparsed.Pages[0].Model.Nodes, "a")
	if node == nil || node.Text != `new & <label>` || node.Width != 130 {
		t.Fatalf("reparsed node = %#v", node)
	}
}

func TestAddShapeAndConnector(t *testing.T) {
	document, _, err := Parse([]byte(richDrawio), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	id, err := document.AddShape(0, ShapeSpec{
		Text: "新形状", Shape: ShapeDiamond, X: 20, Y: 300, Width: 140, Height: 70,
	})
	if err != nil {
		t.Fatal(err)
	}
	connector, err := document.AddConnector(0, ConnectorSpec{
		Text: "→", SourceID: id, TargetID: "b",
	})
	if err != nil {
		t.Fatal(err)
	}
	serialized, err := document.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	reparsed, _, err := Parse(serialized, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if findNode(reparsed.Pages[0].Model.Nodes, id) == nil {
		t.Fatalf("added shape %q missing", id)
	}
	found := false
	for _, edge := range reparsed.Pages[0].Model.Edges {
		if edge.ID == connector {
			found = true
		}
	}
	if !found {
		t.Fatalf("added connector %q missing", connector)
	}
	if !strings.Contains(string(serialized), `<future extension="123.00">`) {
		t.Fatal("insertion changed unknown XML")
	}
	if _, err := document.AddShape(0, ShapeSpec{ID: "b", Shape: ShapeRectangle, Width: 1, Height: 1}); !errors.Is(err, ErrDuplicateID) {
		t.Fatalf("expected duplicate ID, got %v", err)
	}
}

func TestCompressedPageRendersReadOnlyAndRoundTrips(t *testing.T) {
	model := `<mxGraphModel dx="800" dy="600"><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="z" value="压缩" style="rounded=1;" vertex="1" parent="1"><mxGeometry x="5" y="6" width="100" height="40" as="geometry"/></mxCell></root></mxGraphModel>`
	payload := compressRawDeflate(model)
	source := `<mxfile host="local"><diagram id="raw" name="Raw"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="plain" value="plain" vertex="1" parent="1"><mxGeometry x="1" y="2" width="3" height="4" as="geometry"/></mxCell></root></mxGraphModel></diagram><diagram id="zip" name="Zip">` + payload + `</diagram></mxfile>`
	document, _, err := Parse([]byte(source), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(document.Pages) != 2 {
		t.Fatalf("pages = %d", len(document.Pages))
	}
	compressed := document.Pages[1]
	if !compressed.Compressed || !compressed.ReadOnly || compressed.ReadOnlyReason == "" {
		t.Fatalf("compressed page = %#v", compressed)
	}
	if findNode(compressed.Model.Nodes, "z") == nil {
		t.Fatal("compressed model was not rendered")
	}
	if err := document.UpdateText(1, "z", "no"); !errors.Is(err, ErrReadOnly) {
		t.Fatalf("expected read-only error, got %v", err)
	}
	if err := document.UpdateText(0, "plain", "typed"); err != nil {
		t.Fatal(err)
	}
	serialized, err := document.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(serialized), payload) {
		t.Fatal("unrelated edit rewrote the compressed page payload")
	}
	reparsed, _, err := Parse(serialized, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if findNode(reparsed.Pages[1].Model.Nodes, "z") == nil {
		t.Fatal("compressed page did not survive an unrelated edit")
	}
}

func TestSVGIsSelfContainedAndEscaped(t *testing.T) {
	document, _, err := Parse([]byte(richDrawio), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if err := document.UpdateText(0, "a", `2 < 3 & "safe"`); err != nil {
		t.Fatal(err)
	}
	svg, warnings, err := document.ExportSVG(0, SVGOptions{})
	if err != nil {
		t.Fatal(err)
	}
	text := string(svg)
	networkScan := strings.ReplaceAll(text, `xmlns="http://www.w3.org/2000/svg"`, "")
	if strings.Contains(text, "<script") || strings.Contains(text, "<image") ||
		strings.Contains(text, "href=") || strings.Contains(networkScan, "http://") ||
		strings.Contains(networkScan, "https://") {
		t.Fatalf("SVG contains active or network content:\n%s", text)
	}
	for _, exact := range []string{"2 &lt; 3 &amp; &#34;safe&#34;", "<ellipse", "<polygon", "<rect", "marker-end"} {
		if !strings.Contains(text, exact) {
			t.Fatalf("SVG missing %q:\n%s", exact, text)
		}
	}
	var parsed any
	decoder := xml.NewDecoder(bytes.NewReader(svg))
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("SVG is not well-formed through EOF: %v", err)
		}
		if start, ok := token.(xml.StartElement); ok && parsed == nil {
			parsed = start.Name.Local
		}
	}
	if parsed != "svg" {
		t.Fatalf("SVG is not well-formed XML: %q", parsed)
	}
	codes := issueCodes(warnings)
	for _, code := range []IssueCode{IssueHTMLLabel, IssueImage, IssueStencil} {
		if !codes[code] {
			t.Fatalf("SVG warning missing %s: %#v", code, warnings)
		}
	}
}

func TestRawGraphModelRoot(t *testing.T) {
	source := `<mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="a" value="A" vertex="1" parent="1"><mxGeometry x="1" y="2" width="3" height="4" as="geometry"/></mxCell></root></mxGraphModel>`
	document, _, err := Parse([]byte(source), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if document.IsMxFile || len(document.Pages) != 1 || len(document.Pages[0].Model.Nodes) != 1 {
		t.Fatalf("document = %#v", document)
	}
	if err := document.UpdateText(0, "a", "A2"); err != nil {
		t.Fatal(err)
	}
	serialized, _ := document.Serialize()
	if !strings.Contains(string(serialized), `value="A2"`) {
		t.Fatal("raw mxGraphModel was not editable")
	}
}

func TestEmptyRootStaysReadableAndExact(t *testing.T) {
	for _, source := range []string{
		`<mxGraphModel><root/></mxGraphModel>`,
		`<mxfile><diagram id="empty"><mxGraphModel><root/></mxGraphModel></diagram></mxfile>`,
		`<mxGraphModel/>`,
	} {
		document, _, err := Parse([]byte(source), ParseOptions{})
		if err != nil {
			t.Fatalf("parse %q: %v", source, err)
		}
		if !document.Pages[0].ReadOnly || document.Pages[0].ReadOnlyReason == "" {
			t.Fatalf("empty root was not read-only: %#v", document.Pages[0])
		}
		serialized, err := document.Serialize()
		if err != nil || string(serialized) != source {
			t.Fatalf("empty root roundtrip: err=%v got=%q", err, serialized)
		}
		if _, err := document.AddShape(0, ShapeSpec{Shape: ShapeRectangle, Width: 10, Height: 10}); err == nil {
			t.Fatal("empty root accepted an unsafe insertion")
		}
	}
}

func TestHostileInputsAndLimits(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		options ParseOptions
		target  error
	}{
		{
			name:   "doctype",
			source: `<!DOCTYPE x [<!ENTITY boom "boom">]><mxGraphModel><root/></mxGraphModel>`,
			target: ErrInvalidXML,
		},
		{
			name:   "unknown entity",
			source: `<mxGraphModel><root><mxCell id="a" value="&boom;" vertex="1"/></root></mxGraphModel>`,
			target: ErrInvalidXML,
		},
		{
			name:   "duplicate cell id",
			source: `<mxGraphModel><root><mxCell id="same" vertex="1"/><mxCell id="same" vertex="1"/></root></mxGraphModel>`,
			target: ErrDuplicateID,
		},
		{
			name:    "byte limit",
			source:  `<mxGraphModel><root/></mxGraphModel>`,
			options: ParseOptions{MaxBytes: 4},
			target:  ErrLimitExceeded,
		},
		{
			name:    "element limit",
			source:  `<mxGraphModel><root><mxCell id="a" vertex="1"/></root></mxGraphModel>`,
			options: ParseOptions{MaxElements: 1},
			target:  ErrLimitExceeded,
		},
		{
			name:    "depth limit",
			source:  `<mxGraphModel><root><x><y><z/></y></x></root></mxGraphModel>`,
			options: ParseOptions{MaxXMLDepth: 2},
			target:  ErrLimitExceeded,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, err := Parse([]byte(test.source), test.options); !errors.Is(err, test.target) {
				t.Fatalf("got %v, want %v", err, test.target)
			}
		})
	}

	bombXML := strings.Repeat(" ", 4096) + `<mxGraphModel><root/></mxGraphModel>`
	payload := compressRawDeflate(bombXML)
	bomb := `<mxfile><diagram id="bomb">` + payload + `</diagram></mxfile>`
	if _, _, err := Parse([]byte(bomb), ParseOptions{MaxDecompressedBytes: 128}); !errors.Is(err, ErrLimitExceeded) {
		t.Fatalf("decompression bomb got %v", err)
	}
}

func findNode(nodes []Node, id string) *Node {
	for index := range nodes {
		if nodes[index].ID == id {
			return &nodes[index]
		}
	}
	return nil
}

func issueCodes(issues []Issue) map[IssueCode]bool {
	codes := make(map[IssueCode]bool)
	for _, issue := range issues {
		codes[issue.Code] = true
	}
	return codes
}

func compressRawDeflate(value string) string {
	var compressed bytes.Buffer
	writer, err := flate.NewWriter(&compressed, flate.DefaultCompression)
	if err != nil {
		panic(err)
	}
	if _, err := writer.Write([]byte(percentEncode(value))); err != nil {
		panic(err)
	}
	if err := writer.Close(); err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(compressed.Bytes())
}

func percentEncode(value string) string {
	var output strings.Builder
	for _, value := range []byte(value) {
		if value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' ||
			value >= '0' && value <= '9' || strings.ContainsRune("-_.!~*'()", rune(value)) {
			output.WriteByte(value)
			continue
		}
		output.WriteByte('%')
		const hex = "0123456789ABCDEF"
		output.WriteByte(hex[value>>4])
		output.WriteByte(hex[value&15])
	}
	return output.String()
}

func attrFloat(node *Node, value float64) string {
	return strconv.FormatFloat(value, 'g', -1, 64)
}
