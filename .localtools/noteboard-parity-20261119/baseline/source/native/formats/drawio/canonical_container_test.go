package drawio

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func canonicalContainerIssue(id string) Issue {
	return Issue{
		Severity: SeverityWarning,
		Code:     IssueUnsupportedElement,
		Message:  "mxCell is neither a vertex nor an edge and is not rendered",
		Page:     0,
		CellID:   id,
	}
}

func rootWrapperIssue(name string) Issue {
	return Issue{
		Severity: SeverityWarning,
		Code:     IssueUnsupportedElement,
		Message:  "cell wrapper <" + name + "> is preserved but not rendered",
		Page:     0,
	}
}

func rootTextIssue() Issue {
	return Issue{
		Severity: SeverityWarning,
		Code:     IssueUnsupportedElement,
		Message:  "text inside <root> is preserved",
		Page:     0,
	}
}

func cellTextIssue(id string) Issue {
	return Issue{
		Severity: SeverityWarning,
		Code:     IssueUnsupportedElement,
		Message:  "text inside mxCell is preserved",
		Page:     0,
		CellID:   id,
	}
}

func assertIssues(t *testing.T, got, want []Issue) {
	t.Helper()
	if len(got) != len(want) || (len(got) != 0 && !reflect.DeepEqual(got, want)) {
		t.Fatalf("issues:\ngot  %#v\nwant %#v", got, want)
	}
}

func TestCanonicalContainerPairRecognition(t *testing.T) {
	tests := []struct {
		name   string
		root   string
		source string
		want   []Issue
	}{
		{
			name: "self-closing",
			root: `<mxCell id="0"/><mxCell id="1" parent="0"/>`,
		},
		{
			name: "attribute order quotes whitespace comments and explicit close",
			root: "<mxCell\n id='0'> \n<!-- first -->\n</mxCell>\n" +
				"<!-- between -->\n" +
				"<mxCell parent='0' id='1'>\t<!-- second --></mxCell>",
		},
		{
			name: "preceding unrelated sibling remains visible",
			root: `<future/><mxCell id="0"/><mxCell id="1" parent="0"/>`,
			want: []Issue{rootWrapperIssue("future")},
		},
		{
			name: "valid cells follow pair",
			root: `<mxCell id="0"/><mxCell id="1" parent="0"/>` +
				`<mxCell id="a" value="A" vertex="1" parent="1"><mxGeometry x="1" y="2" width="3" height="4" as="geometry"/></mxCell>` +
				`<mxCell id="b" value="B" vertex="1" parent="1"><mxGeometry x="5" y="6" width="7" height="8" as="geometry"/></mxCell>` +
				`<mxCell id="edge" edge="1" parent="1" source="a" target="b"><mxGeometry relative="1" as="geometry"/></mxCell>`,
		},
		{
			name: "third nonvertex cell keeps warning",
			root: `<mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="2"/>`,
			want: []Issue{canonicalContainerIssue("2")},
		},
		{
			name: "unrelated sibling after pair remains visible",
			root: `<mxCell id="0"/><mxCell id="1" parent="0"/><future/>`,
			want: []Issue{rootWrapperIssue("future")},
		},
		{
			name: "sibling between pair preserves ordered warnings",
			root: `<mxCell id="0"/><future/><mxCell id="1" parent="0"/>`,
			want: []Issue{
				canonicalContainerIssue("0"),
				rootWrapperIssue("future"),
				canonicalContainerIssue("1"),
			},
		},
		{
			name: "missing second leaves first warning",
			root: `<mxCell id="0"/>`,
			want: []Issue{canonicalContainerIssue("0")},
		},
		{
			name: "wrong second identity leaves both warnings",
			root: `<mxCell id="0"/><mxCell id="1" parent="wrong"/>`,
			want: []Issue{canonicalContainerIssue("0"), canonicalContainerIssue("1")},
		},
		{
			name: "first parent leaves both warnings",
			root: `<mxCell id="0" parent="0"/><mxCell id="1" parent="0"/>`,
			want: []Issue{canonicalContainerIssue("0"), canonicalContainerIssue("1")},
		},
		{
			name: "extra empty attribute leaves both warnings",
			root: `<mxCell id="0"/><mxCell id="1" extra=""/>`,
			want: []Issue{canonicalContainerIssue("0"), canonicalContainerIssue("1")},
		},
		{
			name: "reversed pair keeps warnings",
			root: `<mxCell id="1" parent="0"/><mxCell id="0"/>`,
			want: []Issue{canonicalContainerIssue("1"), canonicalContainerIssue("0")},
		},
		{
			name: "processing instruction inside first disqualifies",
			root: `<mxCell id="0"><?inside first?></mxCell><mxCell id="1" parent="0"/>`,
			want: []Issue{canonicalContainerIssue("0"), canonicalContainerIssue("1")},
		},
		{
			name: "processing instruction between disqualifies",
			root: `<mxCell id="0"/><?between?><mxCell id="1" parent="0"/>`,
			want: []Issue{canonicalContainerIssue("0"), canonicalContainerIssue("1")},
		},
		{
			name: "processing instruction inside second disqualifies",
			root: `<mxCell id="0"/><mxCell id="1" parent="0"><?inside second?></mxCell>`,
			want: []Issue{canonicalContainerIssue("0"), canonicalContainerIssue("1")},
		},
		{
			name: "non-whitespace text inside disqualifies",
			root: `<mxCell id="0">bad</mxCell><mxCell id="1" parent="0"/>`,
			want: []Issue{
				canonicalContainerIssue("0"),
				cellTextIssue("0"),
				canonicalContainerIssue("1"),
			},
		},
		{
			name: "child inside disqualifies",
			root: `<mxCell id="0"><mxGeometry/></mxCell><mxCell id="1" parent="0"/>`,
			want: []Issue{canonicalContainerIssue("0"), canonicalContainerIssue("1")},
		},
		{
			name: "inherited element namespace disqualifies",
			source: `<mxGraphModel><root xmlns="urn:test">` +
				`<mxCell id="0"/><mxCell id="1" parent="0"/></root></mxGraphModel>`,
			want: []Issue{canonicalContainerIssue("0"), canonicalContainerIssue("1")},
		},
		{
			name: "non-whitespace text inside second disqualifies",
			root: `<mxCell id="0"/><mxCell id="1" parent="0">bad</mxCell>`,
			want: []Issue{
				canonicalContainerIssue("0"),
				canonicalContainerIssue("1"),
				cellTextIssue("1"),
			},
		},
		{
			name: "NBSP inside first disqualifies without changing issue emission",
			root: `<mxCell id="0">` + "\u00a0" + `</mxCell><mxCell id="1" parent="0"/>`,
			want: []Issue{canonicalContainerIssue("0"), canonicalContainerIssue("1")},
		},
		{
			name: "em space inside second disqualifies without changing issue emission",
			root: `<mxCell id="0"/><mxCell id="1" parent="0">` + "\u2003" + `</mxCell>`,
			want: []Issue{canonicalContainerIssue("0"), canonicalContainerIssue("1")},
		},
		{
			name: "NBSP between cells disqualifies without changing issue emission",
			root: `<mxCell id="0"/>` + "\u00a0" + `<mxCell id="1" parent="0"/>`,
			want: []Issue{canonicalContainerIssue("0"), canonicalContainerIssue("1")},
		},
		{
			name: "em space between cells disqualifies without changing issue emission",
			root: `<mxCell id="0"/>` + "\u2003" + `<mxCell id="1" parent="0"/>`,
			want: []Issue{canonicalContainerIssue("0"), canonicalContainerIssue("1")},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := test.source
			if source == "" {
				source = `<mxGraphModel><root>` + test.root + `</root></mxGraphModel>`
			}
			document, issues, err := Parse([]byte(source), ParseOptions{})
			if err != nil {
				t.Fatal(err)
			}
			assertIssues(t, issues, test.want)
			serialized, err := document.Serialize()
			if err != nil || string(serialized) != source {
				t.Fatalf("source round trip: err=%v bytes=%d source=%d", err, len(serialized), len(source))
			}
		})
	}
}

func TestCanonicalContainerNamespaceAndRootSiblingRules(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   []Issue
	}{
		{
			name: "namespaced id",
			source: `<mxGraphModel><root xmlns:x="urn:test">` +
				`<mxCell x:id="0"/><mxCell id="1" parent="0"/></root></mxGraphModel>`,
			want: []Issue{canonicalContainerIssue("0"), canonicalContainerIssue("1")},
		},
		{
			name:   "preceding text is visible but does not replace first mxCell rule",
			source: `<mxGraphModel><root>text<mxCell id="0"/><mxCell id="1" parent="0"/></root></mxGraphModel>`,
			want:   []Issue{rootTextIssue()},
		},
		{
			name:   "non-whitespace text between cells disqualifies",
			source: `<mxGraphModel><root><mxCell id="0"/>text<mxCell id="1" parent="0"/></root></mxGraphModel>`,
			want: []Issue{
				canonicalContainerIssue("0"),
				rootTextIssue(),
				canonicalContainerIssue("1"),
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, issues, err := Parse([]byte(test.source), ParseOptions{})
			if err != nil {
				t.Fatal(err)
			}
			assertIssues(t, issues, test.want)
		})
	}
}

func TestCanonicalContainerDuplicateAndBudgetPaths(t *testing.T) {
	duplicate := []byte(`<mxGraphModel><root><mxCell id="0"/><mxCell id="0"/></root></mxGraphModel>`)
	_, issues, err := Parse(duplicate, ParseOptions{})
	if !errors.Is(err, ErrDuplicateID) {
		t.Fatalf("duplicate pair error = %v", err)
	}
	if len(issues) != 1 || !reflect.DeepEqual(issues[0], canonicalContainerIssue("0")) {
		t.Fatalf("duplicate checks must precede suppression: %#v", issues)
	}

	exact := []byte(`<mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/></root></mxGraphModel>`)
	if _, issues, err := Parse(exact, ParseOptions{MaxElements: 3}); err != nil || len(issues) != 0 {
		t.Fatalf("exact element budget: err=%v issues=%#v", err, issues)
	}
	if _, _, err := Parse(exact, ParseOptions{MaxElements: 2}); !errors.Is(err, ErrLimitExceeded) {
		t.Fatalf("over element budget error = %v", err)
	}

	whitespace := []byte(`<mxGraphModel><root><mxCell id="0">        </mxCell><mxCell id="1" parent="0"/></root></mxGraphModel>`)
	if _, _, err := Parse(whitespace, ParseOptions{MaxTextBytes: 7}); !errors.Is(err, ErrLimitExceeded) {
		t.Fatalf("whitespace text budget error = %v", err)
	}

	invalidSecond := []byte(`<mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"></notMxCell></root></mxGraphModel>`)
	_, issues, err = Parse(invalidSecond, ParseOptions{})
	if !errors.Is(err, ErrInvalidXML) {
		t.Fatalf("invalid second error = %v", err)
	}
	foundFirst := false
	for _, issue := range issues {
		if reflect.DeepEqual(issue, canonicalContainerIssue("0")) {
			foundFirst = true
		}
	}
	if !foundFirst {
		t.Fatalf("failed validation suppressed first warning: %#v", issues)
	}
}

func TestCanonicalContainerWarningRemovalClearsBackingTail(t *testing.T) {
	source := []byte(`<mxGraphModel><root><before/><mxCell id="0"/><mxCell id="1" parent="0"/><after/></root></mxGraphModel>`)
	_, issues, err := Parse(source, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	want := []Issue{rootWrapperIssue("before"), rootWrapperIssue("after")}
	assertIssues(t, issues, want)
	backing := issues[:cap(issues)]
	for index := len(issues); index < len(backing); index++ {
		if backing[index] != (Issue{}) {
			t.Fatalf("removed issue backing slot %d retains %#v", index, backing[index])
		}
	}
}

func TestCanonicalContainerPreservesEditOffsetsAndSource(t *testing.T) {
	source := []byte(`<mxfile host="local"><diagram id="p"><mxGraphModel><root>` +
		`<mxCell id='0'><!--first--></mxCell>` +
		`<mxCell parent='0' id='1'>` + "\n\t" + `<!--second--></mxCell>` +
		`<mxCell id="a" value="A" vertex="1" parent="1"><mxGeometry x="1" y="2" width="3" height="4" as="geometry"/></mxCell>` +
		`<future extension="keep"/></root></mxGraphModel></diagram></mxfile>`)
	before := append([]byte(nil), source...)
	document, issues, err := Parse(source, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 1 || !reflect.DeepEqual(issues[0], rootWrapperIssue("future")) {
		t.Fatalf("initial issues = %#v", issues)
	}
	if document.Dirty() {
		t.Fatal("fresh document is dirty")
	}
	node := findNode(document.Pages[0].Model.Nodes, "a")
	if node == nil || node.startTagStart < 0 || node.geometryStart < node.startTagStart {
		t.Fatalf("node offsets = %#v", node)
	}
	if err := document.UpdateText(0, "a", `changed & <safe>`); err != nil {
		t.Fatal(err)
	}
	if err := document.UpdateGeometry(0, "a", 11.25, 22.5, 13, 14); err != nil {
		t.Fatal(err)
	}
	edited, err := document.Serialize()
	if err != nil {
		t.Fatal(err)
	}
	if !document.Dirty() || !bytes.Equal(document.OriginalBytes(), before) {
		t.Fatal("edit lost original source or dirty state")
	}
	for _, exact := range []string{
		`<mxCell id='0'><!--first--></mxCell>`,
		"<mxCell parent='0' id='1'>\n\t<!--second--></mxCell>",
		`<future extension="keep"/>`,
		`value="changed &amp; &lt;safe&gt;"`,
		`x="11.25" y="22.5" width="13" height="14"`,
	} {
		if !strings.Contains(string(edited), exact) {
			t.Fatalf("edited source lost %q:\n%s", exact, edited)
		}
	}
	reparsed, _, err := Parse(edited, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	node = findNode(reparsed.Pages[0].Model.Nodes, "a")
	if node == nil || node.Text != `changed & <safe>` || node.X != 11.25 || node.Width != 13 {
		t.Fatalf("reparsed node = %#v", node)
	}
}
