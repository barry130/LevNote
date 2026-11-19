package drawio

import (
	"strings"
	"testing"
)

func TestMxFileIssuesUseDocumentScopeWithoutRelabelingPageIssues(t *testing.T) {
	source := []byte(`<mxfile>outside<documentOnly/>` +
		`<diagram id="first"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><pageZero/></root></mxGraphModel></diagram>` +
		`<diagram id="second"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><pageOne/></root></mxGraphModel></diagram>` +
		`</mxfile>`)
	_, issues, err := Parse(source, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]int{}
	for _, issue := range issues {
		found[issue.Message] = issue.Page
	}
	for _, test := range []struct {
		fragment string
		page     int
	}{
		{"non-whitespace text inside mxfile", DocumentPage},
		{"unknown mxfile child <documentOnly>", DocumentPage},
		{"cell wrapper <pageZero>", 0},
		{"cell wrapper <pageOne>", 1},
	} {
		page, ok := -999, false
		for message, issuePage := range found {
			if strings.Contains(message, test.fragment) {
				page, ok = issuePage, true
				break
			}
		}
		if !ok || page != test.page {
			t.Fatalf("issue %q page=%d present=%v, want %d; issues=%+v", test.fragment, page, ok, test.page, issues)
		}
	}
}
