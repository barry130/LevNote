package code

import (
	"strings"
	"testing"
)

func TestHighlightSpansAndLanguages(t *testing.T) {
	tests := []struct {
		source   string
		language Language
		token    Token
		kind     string
		text     string
	}{
		{"func main() {}", LanguageGo, TokenKeyword, "word", "func"},
		{"\"hello\"", LanguageJSON, TokenString, "string", "\"hello\""},
		{"# note", LanguagePython, TokenComment, "comment", "# note"},
		{"<tag>", LanguageXML, TokenTag, "tag", "<tag>"},
		{"let x = 1;", LanguageTypeScript, TokenKeyword, "word", "let"},
	}
	for _, test := range tests {
		result := Highlight(test.source, test.language)
		if len(result.Spans) == 0 {
			t.Fatalf("language %s returned no spans", test.language)
		}
		found := false
		for _, span := range result.Spans {
			if span.Token == test.token && span.Kind == test.kind && test.source[span.Start:span.End] == test.text {
				found = true
			}
		}
		if !found {
			t.Errorf("language %s did not emit %q: %#v", test.language, test.text, result.Spans)
		}
	}
}

func TestLargeFileGuard(t *testing.T) {
	source := make([]byte, maxHighlightBytes+1)
	for i := range source {
		source[i] = 'x'
	}
	result := Highlight(string(source), LanguageGo)
	if len(result.Spans) != 0 || len(result.Warnings) != 1 {
		t.Fatalf("guard result = %#v", result)
	}
}

func TestUnsupportedLanguageDoesNotGuess(t *testing.T) {
	result := Highlight("foo", Language("brainfuck"))
	if len(result.Spans) != 0 || result.Warnings[0].Code != "language" {
		t.Fatalf("result = %#v", result)
	}
}

func TestLanguageForPath(t *testing.T) {
	tests := map[string]Language{
		"main.GO":      LanguageGo,
		"app.tsx":      LanguageTypeScript,
		"data.mindmap": LanguageJSON,
		"icon.svg":     LanguageXML,
		"notes.txt":    "",
	}
	for path, want := range tests {
		if got := LanguageForPath(path); got != want {
			t.Errorf("LanguageForPath(%q) = %q; want %q", path, got, want)
		}
	}
}

func TestDeclarativeSyntax(t *testing.T) {
	source := `# comment true
name = "string true"
enabled = true
/* block false */`
	result := HighlightDeclarative(source, DeclarativeSyntax{
		LineComment: "#", BlockStart: "/*", BlockEnd: "*/", Keywords: []string{"true", "false"},
	})
	if len(result.Warnings) != 0 {
		t.Fatalf("warnings = %#v", result.Warnings)
	}
	joined := ""
	count := map[Token]int{}
	for _, span := range result.Spans {
		if span.Start < 0 || span.End < span.Start || span.End > len(source) {
			t.Fatalf("span out of bounds: %#v", span)
		}
		joined += source[span.Start:span.End]
		count[span.Token]++
	}
	if joined != source {
		t.Fatal("declarative highlighting changed source text")
	}
	if count[TokenComment] != 2 || count[TokenString] != 1 || count[TokenKeyword] != 1 {
		t.Fatalf("token counts = %#v", count)
	}
}

func TestDeclarativeSyntaxForwardProgressOnNewlineMarker(t *testing.T) {
	source := "alpha\nbeta"
	result := HighlightDeclarative(source, DeclarativeSyntax{LineComment: "\n"})
	if len(result.Warnings) != 0 {
		t.Fatalf("warnings = %#v", result.Warnings)
	}
	var reconstructed strings.Builder
	for _, span := range result.Spans {
		if span.End <= span.Start || span.End > len(source) {
			t.Fatalf("invalid non-progressing span: %#v", span)
		}
		reconstructed.WriteString(source[span.Start:span.End])
	}
	if reconstructed.String() != source {
		t.Fatalf("spans reconstructed %q; want %q", reconstructed.String(), source)
	}
}

func TestUnicodeIdentifiersAndMalformedByteForwardProgress(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		language Language
		word     string
	}{
		{"go-unicode", "变量名 := 函数🙂", LanguageGo, "变量名"},
		{"typescript-unicode", "let 变量🙂 = 1", LanguageTypeScript, "变量"},
		{"malformed-leading-byte", string([]byte{0xc3, 'z', ' ', 0xf0, 0x9f, 0x99, 0x82}), LanguageJavaScript, "z"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := Highlight(test.source, test.language)
			if len(result.Warnings) != 0 {
				t.Fatalf("warnings = %#v", result.Warnings)
			}
			var reconstructed strings.Builder
			foundWord := false
			for _, span := range result.Spans {
				if span.Start < 0 || span.End <= span.Start || span.End > len(test.source) {
					t.Fatalf("invalid span: %#v", span)
				}
				reconstructed.WriteString(test.source[span.Start:span.End])
				if span.Token == TokenName && test.source[span.Start:span.End] == test.word {
					foundWord = true
				}
			}
			if reconstructed.String() != test.source {
				t.Fatalf("spans reconstructed %q; want %q", reconstructed.String(), test.source)
			}
			if !foundWord {
				t.Fatalf("Unicode word %q missing from %#v", test.word, result.Spans)
			}
		})
	}
}
