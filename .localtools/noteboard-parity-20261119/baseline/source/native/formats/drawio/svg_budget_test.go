package drawio

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

type svgPreChangeOracle struct {
	Source      string     `json:"source"`
	Options     SVGOptions `json:"options"`
	EdgePoints  []Point    `json:"edgePoints"`
	SVG         string     `json:"svg"`
	Issues      []Issue    `json:"issues"`
	SVGByteSize int        `json:"svgByteSize"`
}

func loadSVGPreChangeOracle(t *testing.T) svgPreChangeOracle {
	t.Helper()
	data, err := os.ReadFile("testdata/svg-prechange-oracle.json")
	if err != nil {
		t.Fatal(err)
	}
	var oracle svgPreChangeOracle
	if err := json.Unmarshal(data, &oracle); err != nil {
		t.Fatal(err)
	}
	if oracle.Source == "" || oracle.SVG == "" || oracle.SVGByteSize != len(oracle.SVG) {
		t.Fatalf("invalid pre-change oracle: source=%d svg=%d recorded=%d", len(oracle.Source), len(oracle.SVG), oracle.SVGByteSize)
	}
	return oracle
}

func svgOracleDocument(t *testing.T, oracle svgPreChangeOracle) (*Document, []Issue) {
	t.Helper()
	document, issues, err := Parse([]byte(oracle.Source), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(document.Pages) == 0 || len(document.Pages[0].Model.Edges) == 0 {
		t.Fatal("pre-change oracle lost its edge fixture")
	}
	document.Pages[0].Model.Edges[0].Points = append([]Point(nil), oracle.EdgePoints...)
	return document, issues
}

func cloneSVGTestPage(page Page) Page {
	clone := page
	clone.Model.Nodes = append([]Node(nil), page.Model.Nodes...)
	for index := range clone.Model.Nodes {
		clone.Model.Nodes[index].Unsupported = append([]string(nil), page.Model.Nodes[index].Unsupported...)
	}
	clone.Model.Edges = append([]Edge(nil), page.Model.Edges...)
	for index := range clone.Model.Edges {
		edge := &clone.Model.Edges[index]
		source := page.Model.Edges[index]
		edge.Points = append([]Point(nil), source.Points...)
		edge.Unsupported = append([]string(nil), source.Unsupported...)
		if source.SourcePoint != nil {
			point := *source.SourcePoint
			edge.SourcePoint = &point
		}
		if source.TargetPoint != nil {
			point := *source.TargetPoint
			edge.TargetPoint = &point
		}
	}
	return clone
}

func assertSVGSourceUnchanged(t *testing.T, document *Document, source, original []byte) {
	t.Helper()
	if !bytes.Equal(document.Source, source) || !bytes.Equal(document.original, original) {
		t.Fatal("SVG export changed exported source fields")
	}
}

func TestSVGSuccessMatchesPreChangeOracle(t *testing.T) {
	oracle := loadSVGPreChangeOracle(t)
	document, parseIssues := svgOracleDocument(t, oracle)
	// The immutable oracle preceded warning suppression. Only its two canonical
	// container warnings are intentionally absent now; the wrapper stays visible.
	wantParseIssues := []Issue{
		{Severity: SeverityWarning, Code: IssueHTMLLabel, Message: "HTML label is preserved verbatim and rendered as escaped plain text", Page: 0, CellID: "a"},
		{Severity: SeverityWarning, Code: IssueHTMLLabel, Message: "HTML label is preserved verbatim and rendered as escaped plain text", Page: 0, CellID: "html"},
		{Severity: SeverityWarning, Code: IssueImage, Message: "image or remote image style is preserved; native SVG export does not fetch it", Page: 0, CellID: "image"},
		{Severity: SeverityWarning, Code: IssueStencil, Message: "custom stencil is preserved and rendered with a basic-shape fallback", Page: 0, CellID: "stencil"},
		{Severity: SeverityWarning, Code: IssueUnsupportedElement, Message: "cell wrapper <future> is preserved but not rendered", Page: 0},
	}
	if !reflect.DeepEqual(parseIssues, wantParseIssues) {
		t.Fatalf("source issue subtraction changed:\ngot  %#v\nwant %#v", parseIssues, wantParseIssues)
	}
	beforePage := cloneSVGTestPage(document.Pages[0])
	beforeSource := append([]byte(nil), document.Source...)
	beforeOriginal := append([]byte(nil), document.original...)
	before, err := document.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	svg, issues, err := document.ExportSVG(0, oracle.Options)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(svg, []byte(oracle.SVG)) {
		t.Fatalf("successful SVG bytes changed: got=%d want=%d", len(svg), len(oracle.SVG))
	}
	if !reflect.DeepEqual(issues, oracle.Issues) {
		t.Fatalf("ordered issues changed:\ngot  %#v\nwant %#v", issues, oracle.Issues)
	}
	after, err := document.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, before) {
		t.Fatal("successful SVG export changed source")
	}
	assertSVGSourceUnchanged(t, document, beforeSource, beforeOriginal)
	if !reflect.DeepEqual(document.Pages[0], beforePage) {
		t.Fatal("successful SVG export changed page/model state")
	}
	decoder := xml.NewDecoder(bytes.NewReader(svg))
	for {
		if _, err := decoder.Token(); err == io.EOF {
			break
		} else if err != nil {
			t.Fatalf("exported SVG is not well-formed through EOF: %v", err)
		}
	}
}

func TestSVGOutputLimitContract(t *testing.T) {
	oracle := loadSVGPreChangeOracle(t)
	want := []byte(oracle.SVG)

	document, _ := svgOracleDocument(t, oracle)
	options := oracle.Options
	options.MaxBytes = len(want)
	exact, issues, err := document.ExportSVG(0, options)
	if err != nil || !bytes.Equal(exact, want) || !reflect.DeepEqual(issues, oracle.Issues) {
		t.Fatalf("exact cap: err=%v size=%d issues=%d", err, len(exact), len(issues))
	}

	document, _ = svgOracleDocument(t, oracle)
	options.MaxBytes = len(want) - 1
	output, issues, err := document.ExportSVG(0, options)
	if !errors.Is(err, ErrSVGOutputLimit) || output != nil || issues != nil {
		t.Fatalf("cap-minus-one: err=%v output=%d issues=%d", err, len(output), len(issues))
	}

	document, _ = svgOracleDocument(t, oracle)
	options.MaxBytes = -1
	output, issues, err = document.ExportSVG(0, options)
	if !errors.Is(err, ErrSVGOutputLimit) || output != nil || issues != nil {
		t.Fatalf("negative cap: err=%v output=%d issues=%d", err, len(output), len(issues))
	}
}

func TestSVGOutputLimitNormalization(t *testing.T) {
	for _, test := range []struct {
		in, want int
		err      error
	}{
		{in: 0, want: MaxSVGBytes},
		{in: 1, want: 1},
		{in: MaxSVGBytes, want: MaxSVGBytes},
		{in: MaxSVGBytes + 1, want: MaxSVGBytes},
		{in: -1, err: ErrSVGOutputLimit},
	} {
		got, err := resolveSVGMaxBytes(test.in)
		if got != test.want || !errors.Is(err, test.err) {
			t.Fatalf("resolveSVGMaxBytes(%d) = %d, %v; want %d, %v", test.in, got, err, test.want, test.err)
		}
	}
}

func TestSVGEscapingChunksPreserveEncodingXMLSemantics(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{
			name:  "valid-boundaries",
			value: "ab<&🙂\"'\n\t结尾&<>",
		},
		{
			name:  "invalid-and-controls",
			value: string([]byte{'a', 0xff, 0x80, '<', 0x00, '\x01', '\r', '\n', '\t', '"', '\'', 0xc3}),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			want := []byte(escapeAttr(test.value))
			for _, chunkSize := range []int{4, 5, 7, 8, 9, 13} {
				writer := svgWriter{limit: 1 << 20, data: make([]byte, 1<<20)}
				writeEscapedXMLChunks(&writer, test.value, chunkSize)
				if writer.err != nil {
					t.Fatalf("chunk %d: %v", chunkSize, writer.err)
				}
				if got := writer.data[:writer.written]; !bytes.Equal(got, want) {
					t.Fatalf("chunk %d changed encoding/xml bytes:\ngot  %q\nwant %q", chunkSize, got, want)
				}
			}
		})
	}
}

func TestSVGEscapingProductionChunkBoundary(t *testing.T) {
	value := strings.Repeat("a", svgEscapeChunkSize-1) + "é<&" + string([]byte{0xff}) + "中"
	if got := svgEscapeChunkLength(value, svgEscapeChunkSize); got != svgEscapeChunkSize-1 {
		t.Fatalf("fixture does not straddle production boundary: first chunk=%d", got)
	}
	writer := svgWriter{limit: 1 << 20, data: make([]byte, 1<<20)}
	writeEscapedXMLChunks(&writer, value, svgEscapeChunkSize)
	if writer.err != nil {
		t.Fatal(writer.err)
	}
	want := []byte(escapeAttr(value))
	if got := writer.data[:writer.written]; !bytes.Equal(got, want) {
		t.Fatalf("production boundary changed encoding/xml bytes:\ngot  %q\nwant %q", got, want)
	}
}

func TestSVGWarningFreeSuccessPreservesNonNilIssues(t *testing.T) {
	source := `<mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="a" value="plain" vertex="1" parent="1"><mxGeometry x="1" y="2" width="100" height="40" as="geometry"/></mxCell></root></mxGraphModel>`
	document, _, err := Parse([]byte(source), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	svg, issues, err := document.ExportSVG(0, SVGOptions{})
	if err != nil || len(svg) == 0 || issues == nil || len(issues) != 0 {
		t.Fatalf("warning-free success: err=%v svg=%d issues-nil=%t issues=%d", err, len(svg), issues == nil, len(issues))
	}
}

func TestSVGNewlineExpansionHonorsCapWithoutPartialOutput(t *testing.T) {
	source := `<mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="a" value="seed" vertex="1" parent="1"><mxGeometry x="1" y="2" width="100" height="40" as="geometry"/></mxCell></root></mxGraphModel>`
	document, _, err := Parse([]byte(source), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	beforePage := cloneSVGTestPage(document.Pages[0])
	document.Pages[0].Model.Nodes[0].PlainText = strings.Repeat("x\n", 2000)
	beforeSource := append([]byte(nil), document.Source...)
	beforeOriginal := append([]byte(nil), document.original...)
	before, err := document.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	output, issues, err := document.ExportSVG(0, SVGOptions{MaxBytes: 1024})
	if !errors.Is(err, ErrSVGOutputLimit) || output != nil || issues != nil {
		t.Fatalf("newline amplification: err=%v output=%d issues=%d", err, len(output), len(issues))
	}
	after, err := document.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, before) {
		t.Fatal("failed SVG export changed source")
	}
	assertSVGSourceUnchanged(t, document, beforeSource, beforeOriginal)
	// The deliberate PlainText amplification is the expected page mutation.
	beforePage.Model.Nodes[0].PlainText = document.Pages[0].Model.Nodes[0].PlainText
	if !reflect.DeepEqual(document.Pages[0], beforePage) {
		t.Fatal("failed SVG export changed page/model state")
	}
}

func TestSVGSelectedPageAndPageLookupPrecedence(t *testing.T) {
	source := `<mxfile><diagram id="one" name="One"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="a" value="PAGE_ONE" vertex="1" parent="1"><mxGeometry x="1" y="2" width="100" height="40" as="geometry"/></mxCell></root></mxGraphModel></diagram><diagram id="two" name="Two"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="b" value="PAGE_TWO" vertex="1" parent="1"><mxGeometry x="3" y="4" width="90" height="30" as="geometry"/></mxCell></root></mxGraphModel></diagram></mxfile>`
	document, _, err := Parse([]byte(source), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	pagesBefore := make([]Page, len(document.Pages))
	for index := range document.Pages {
		pagesBefore[index] = cloneSVGTestPage(document.Pages[index])
	}
	beforeSource := append([]byte(nil), document.Source...)
	beforeOriginal := append([]byte(nil), document.original...)
	before, err := document.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	svg, issues, err := document.ExportSVG(1, SVGOptions{MaxBytes: 64 << 10})
	if err != nil || len(issues) != 0 {
		t.Fatalf("selected page: err=%v issues=%#v", err, issues)
	}
	if !bytes.Contains(svg, []byte("PAGE_TWO")) || bytes.Contains(svg, []byte("PAGE_ONE")) {
		t.Fatalf("exported the wrong page: %s", svg)
	}
	after, err := document.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, before) {
		t.Fatal("selected-page export changed source")
	}
	assertSVGSourceUnchanged(t, document, beforeSource, beforeOriginal)
	if !reflect.DeepEqual(document.Pages, pagesBefore) {
		t.Fatal("selected-page export changed page/model state")
	}
	if _, _, err := document.ExportSVG(-1, SVGOptions{MaxBytes: -1}); !errors.Is(err, ErrPageNotFound) {
		t.Fatalf("page lookup precedence changed: %v", err)
	}
}
