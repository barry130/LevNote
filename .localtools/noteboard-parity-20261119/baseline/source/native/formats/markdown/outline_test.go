package markdown

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestHeadingAnchorsShareParserRecognitionAndHTML(t *testing.T) {
	source := "# A\n# A\n# a-2\n## 你好 世界🙂\n### E\u0301\n# !!!\n# !!!\n# A\n\n```md\n# fenced\n```\n\nnot # a heading\n"
	doc := Parse(source)
	headings, partial := Headings(doc, 100)
	want := []string{"a", "a-2", "a-2-2", "你好-世界", "e\u0301", "section", "section-2", "a-3"}
	if partial || len(headings) != len(want) {
		t.Fatalf("headings=%+v partial=%v", headings, partial)
	}
	html, _, err := ExportHTMLChecked(doc)
	if err != nil {
		t.Fatal(err)
	}
	for i, heading := range headings {
		if heading.AnchorID != want[i] || source[heading.Range.Start] != '#' {
			t.Fatalf("heading%d=%+v", i, heading)
		}
		if !strings.Contains(html, fmt.Sprintf("<h%d id=\"%s\">", heading.Level, want[i])) {
			t.Fatalf("outline/HTML anchor mismatch: %s", html)
		}
	}
}

func TestHeadingAnchorBoundsAndCountDisclosure(t *testing.T) {
	doc := Parse("# " + strings.Repeat("中", 300) + "\n# second\n")
	headings, partial := Headings(doc, 1)
	if !partial || len(headings) != 1 || len(headings[0].AnchorID) > maxAnchorBaseBytes || !utf8.ValidString(headings[0].AnchorID) {
		t.Fatalf("unbounded or undisclosed anchors: %+v partial%v", headings, partial)
	}
	var anchors HeadingAnchors
	for i := 0; i < 8192; i++ {
		got, _ := anchors.Next("same")
		want := "same"
		if i > 0 {
			want = fmt.Sprintf("same-%d", i+1)
		}
		if got != want {
			t.Fatalf("duplicate%d=%s", i, got)
		}
	}
	if anchors.next["same"] != 8193 {
		t.Fatal("duplicate cursor did not advance")
	}
}

func BenchmarkHeadingAnchorsRepeated(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var anchors HeadingAnchors
		for n := 0; n < 8192; n++ {
			anchors.Next("中文 repeated heading")
		}
	}
}
