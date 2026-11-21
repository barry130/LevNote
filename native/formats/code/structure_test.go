package code

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestStructureGoCoordinatesAndLexicalStates(t *testing.T) {
	source := "package p\r\n" +
		"type T struct {\r\n" +
		"\tName string\n" +
		"}\r\n" +
		"func f() /*甲🙂e\u0301👩\u200d💻*/ { // }\r\n" +
		"\tif true {\n" +
		"\t\t_ = \"{[\\\"}]\"\n" +
		"\t\t_ = '}'\r\n" +
		"\t\t_ = `}[\n" +
		"({])` // [}\r\n" +
		"\t\t/* }[\n" +
		")({ */\r\n" +
		"\t}\n" +
		"}"
	want := []StructureFold{
		{structureTestDelimiter(25, 25, 1), structureTestDelimiter(41, 41, 3), StructureLineRange{2, 3}},
		{structureTestDelimiter(96, 85, 5), structureTestDelimiter(164, 153, 12), StructureLineRange{6, 12}},
		{structureTestDelimiter(79, 68, 4), structureTestDelimiter(166, 155, 13), StructureLineRange{5, 13}},
	}
	checkStructureFolds(t, source, StructureGo, want)
}

func TestStructureJSONCoordinatesAndEscapes(t *testing.T) {
	source := "{\"甲🙂e\u0301👩\u200d💻\": [\r\n" +
		"\t{\"s\": \"}\\\"[\\\\\\u007b\",\n" +
		"\t \"ok\": null,\r\n" +
		"\t \"n\": -0.42e+9},\r\n" +
		"\ttrue\r\n" +
		"]}"
	want := []StructureFold{
		{structureTestDelimiter(30, 19, 1), structureTestDelimiter(82, 71, 3), StructureLineRange{2, 3}},
		{structureTestDelimiter(26, 15, 0), structureTestDelimiter(93, 82, 5), StructureLineRange{1, 5}},
		{structureTestDelimiter(0, 0, 0), structureTestDelimiter(94, 83, 5), StructureLineRange{1, 5}},
	}
	checkStructureFolds(t, source, StructureJSON, want)
}

func TestStructureGoParenBracketAndPhysicalLines(t *testing.T) {
	t.Run("paren-and-bracket", func(t *testing.T) {
		source := "package p\nvar _ = f(\n a[\n 0:\n 1],\n)\n"
		want := []StructureFold{
			{structureTestDelimiter(23, 23, 2), structureTestDelimiter(31, 31, 4), StructureLineRange{3, 4}},
			{structureTestDelimiter(19, 19, 1), structureTestDelimiter(34, 34, 5), StructureLineRange{2, 5}},
		}
		checkStructureFolds(t, source, StructureGo, want)
	})
	t.Run("bare-cr-stays-in-line-comment", func(t *testing.T) {
		const source = "package p\nfunc f() {\n // } \r ]\n}\n"
		checkStructureFolds(t, source, StructureGo, []StructureFold{
			{structureTestDelimiter(19, 19, 1), structureTestDelimiter(31, 31, 3), StructureLineRange{2, 3}},
		})
	})
	t.Run("line-directive-keeps-physical-coordinates", func(t *testing.T) {
		const source = "package p\n//line nonexistent.go:900:12\nfunc f() {\n f()\n}"
		checkStructureFolds(t, source, StructureGo, []StructureFold{
			{structureTestDelimiter(48, 48, 2), structureTestDelimiter(55, 55, 4), StructureLineRange{3, 4}},
		})
	})
}

func TestStructureNoFoldWithoutFullMiddleLine(t *testing.T) {
	cases := []struct {
		profile StructureProfile
		source  string
	}{
		{StructureGo, "package p\nfunc f() {}"},
		{StructureGo, "package p\nfunc f() {\r\n}"},
		{StructureGo, "package p\n// { \n// }\n"},
		{StructureJSON, `[{"x":"{}"},[]]`},
		{StructureJSON, "[\r \r\n]"},
		{StructureJSON, "{\r\r}"},
		{StructureJSON, `"a\n{\n}"`},
	}
	for _, tt := range cases {
		checkStructureFolds(t, tt.source, tt.profile, nil)
	}
}

func TestStructureUnsupportedAndInvalidUTF8(t *testing.T) {
	const invalid = "[\r\n\"🙂\"\n]" + "\xff"
	for _, profile := range []StructureProfile{"", "jsonc", "JSON", "typescript", "python", "rust", "xml"} {
		limits := DefaultStructureLimits()
		limits.Bytes = 1 // unsupported never scans the source
		result, err := AnalyzeStructure(invalid, profile, limits)
		if err != nil || !reflect.DeepEqual(result, StructureResult{Unsupported: true}) {
			t.Errorf("profile %q: result = %#v, err = %v", profile, result, err)
		}
	}
	for _, profile := range []StructureProfile{StructureGo, StructureJSON} {
		result, err := AnalyzeStructure(invalid, profile, DefaultStructureLimits())
		want := StructureSpan{StructureByteRange{11, 11}, StructureUTF16Range{9, 9}, 2}
		if err != nil || result.Complete || result.Unsupported || len(result.Folds) != 0 ||
			len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "invalid-utf8" || result.Diagnostics[0].Span != want {
			t.Errorf("profile %q: result = %#v, err = %v; want invalid byte at %#v", profile, result, err, want)
		}
	}
	checkStructureFolds(t, `"�"`, StructureJSON, nil) // a real U+FFFD isn't invalid UTF-8
}

func TestStructureGoRejectsLexicalStructuralAndGrammarErrors(t *testing.T) {
	const prefix = "package p\nfunc f() {\n f()\n}\n"
	cases := []struct{ name, source, code string }{
		{"empty", "", "go-syntax"},
		{"unclosed-block", prefix + "func g() {\n", "go-delimiter"},
		{"mismatched-delimiter", prefix + "func g() {\n ]\n}", "go-delimiter"},
		{"stray-close", prefix + "}", "go-delimiter"},
		{"interpreted-eof", prefix + `var s = "abc`, "go-lexical"},
		{"interpreted-newline", prefix + "var s = \"abc\n\"", "go-lexical"},
		{"invalid-escape", prefix + `var s = "\q"`, "go-lexical"},
		{"invalid-rune", prefix + `var s = 'ab'`, "go-lexical"},
		{"invalid-number", prefix + `var i = 0b2`, "go-lexical"},
		{"raw-eof", prefix + "var s = `abc", "go-lexical"},
		{"comment-eof", prefix + "/* } ]\n", "go-lexical"},
		{"trailing-garbage", prefix + "garbage", "go-syntax"},
		{"missing-operand", prefix + "func g() {\n _ =\n}", "go-syntax"},
		{"missing-comma", prefix + "var xs = []int{\n 1\n}", "go-syntax"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			checkStructureRejected(t, tt.source, StructureGo, tt.code)
		})
	}
	t.Run("physical-diagnostic-line-overrides-directive", func(t *testing.T) {
		source := prefix + "//line other.go:500:20\r\nvar = 1"
		got := checkStructureRejected(t, source, StructureGo, "go-syntax")
		offset := strings.IndexByte(source, '=')
		if want := structureTestDelimiter(offset, offset, 5); got.Span != want {
			t.Fatalf("diagnostic span = %#v, want %#v", got.Span, want)
		}
	})
}

func TestStructureStrictJSONRejectsInvalidGrammar(t *testing.T) {
	cases := []struct{ name, source string }{
		{"empty", ""}, {"whitespace-only", " \t\r\n"},
		{"line-comment", "[\n// comment\n0\n]"}, {"block-comment", `[/* comment */0]`},
		{"array-trailing-comma", "[\n[\n0\n],\n]"}, {"object-trailing-comma", `{"x":1,}`},
		{"tail-garbage-after-completed-fold", "[\n0\n]\nxy"}, {"second-value", `true false`},
		{"array-missing-comma", `[0 1]`}, {"object-missing-comma", `{"x":1 "y":2}`},
		{"object-missing-colon", `{"x" 0}`}, {"object-missing-value", `{"x":}`},
		{"unquoted-key", `{x:1}`}, {"single-quote", `{'x':1}`},
		{"wrong-closer", "{\n\"x\": [\n1\n}\n]"}, {"unclosed-container", "[\n0\n"},
		{"unclosed-string", `"abc`}, {"escape-eof", `"\`}, {"escape", `"\q"`},
		{"unicode-nonhex", `"\u00x0"`}, {"unicode-short", `"\u01"`}, {"unicode-eof", `"\u01`},
		{"raw-newline", "\"abc\ndef\""}, {"raw-control", "\"\x00\""},
		{"form-feed", "[\f0]"}, {"nonbreaking-space", "[\u00a00]"}, {"bom", "\ufeff{}"},
		{"positive-sign", `+1`}, {"leading-zero", `01`}, {"negative-leading-zero", `-00`},
		{"bare-minus", `-`}, {"leading-dot", `.1`}, {"fraction", `1.`},
		{"exponent", `1e`}, {"exponent-sign", `1E+`}, {"hex", `0x2`}, {"separator", `1_2`},
		{"nan", `NaN`}, {"infinity", `Infinity`}, {"truncated-literal", `tru`},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			checkStructureRejected(t, tt.source, StructureJSON, "json-syntax")
		})
	}
	t.Run("unicode-diagnostic-coordinates", func(t *testing.T) {
		source := "[\r\n\"🙂\\q\"\n]"
		diagnostic := checkStructureRejected(t, source, StructureJSON, "json-syntax")
		if want := structureTestDelimiter(9, 7, 1); diagnostic.Span != want {
			t.Fatalf("diagnostic span = %#v, want %#v", diagnostic.Span, want)
		}
	})
	t.Run("eof-point-after-final-lf", func(t *testing.T) {
		source := "[\r\n\"🙂\"\n"
		diagnostic := checkStructureRejected(t, source, StructureJSON, "json-syntax")
		want := StructureSpan{StructureByteRange{10, 10}, StructureUTF16Range{8, 8}, 2}
		if diagnostic.Span != want {
			t.Fatalf("diagnostic span = %#v, want %#v", diagnostic.Span, want)
		}
	})
}

func TestStructureJSONGrammarAgainstSyntaxOracle(t *testing.T) {
	const punctuation = "{}[],:\"\\/-+0129.eEtfnuABC \r\n\t\x00\f"
	for _, seed := range []string{
		`{"a":[true,false,null,-0.12E+3,{},[],{"b":"\u00ea\t\"\\\/"}]}`,
		`{"a":1,"a":2}`, `"\uDEAD\uD800"`, `1e999999999999999999999`, `"👩‍💻"`,
	} {
		checkStructureFolds(t, seed, StructureJSON, nil)
		for i := 0; i < len(seed); i++ {
			for j := 0; j < len(punctuation); j++ {
				mutated := seed[:i] + punctuation[j:j+1] + seed[i+1:]
				result, err := AnalyzeStructure(mutated, StructureJSON, DefaultStructureLimits())
				want := utf8.ValidString(mutated) && json.Valid([]byte(mutated))
				if err != nil || result.Complete != want || (!want && (len(result.Folds) != 0 || len(result.Diagnostics) != 1)) {
					t.Fatalf("mutation %q: result = %#v, err = %v, want complete = %t", mutated, result, err, want)
				}
			}
		}
	}
}

func TestStructureDeterministicIndependentResults(t *testing.T) {
	const source = "[\n [\n  0\n ]\n]"
	limits := DefaultStructureLimits()
	a, errA := AnalyzeStructure(source, StructureJSON, limits)
	b, errB := AnalyzeStructure(source, StructureJSON, limits)
	if errA != nil || errB != nil || !a.Complete || len(a.Folds) != 2 || !reflect.DeepEqual(a, b) {
		t.Fatalf("analysis changed: %#v (%v), %#v (%v)", a, errA, b, errB)
	}
	a.Folds[0].Open.Bytes.Start = -1
	a.Folds[1].HiddenLines = StructureLineRange{}
	c, err := AnalyzeStructure(source, StructureJSON, limits)
	if err != nil || !reflect.DeepEqual(b, c) {
		t.Fatalf("fold slice mutation leaked: %#v, %#v (%v)", b, c, err)
	}
	bad := source[:len(source)-1] + "}"
	first, errA := AnalyzeStructure(bad, StructureJSON, limits)
	second, errB := AnalyzeStructure(bad, StructureJSON, limits)
	if errA != nil || errB != nil || first.Complete || len(first.Folds) != 0 || len(first.Diagnostics) != 1 || !reflect.DeepEqual(first, second) {
		t.Fatalf("invalid candidate survived delimiter edit: %#v (%v), %#v (%v)", first, errA, second, errB)
	}
	first.Diagnostics[0] = StructureDiagnostic{}
	third, err := AnalyzeStructure(bad, StructureJSON, limits)
	if err != nil || !reflect.DeepEqual(second, third) || second.Diagnostics[0].Code == "" {
		t.Fatalf("diagnostic slice mutation leaked: %#v, %#v (%v)", second, third, err)
	}
}

func structureTestDelimiter(offset, units, line int) StructureSpan {
	return StructureSpan{StructureByteRange{offset, offset + 1}, StructureUTF16Range{units, units + 1}, line}
}

func checkStructureFolds(t *testing.T, source string, profile StructureProfile, want []StructureFold) {
	t.Helper()
	original := strings.Clone(source)
	got, err := AnalyzeStructure(source, profile, DefaultStructureLimits())
	if err != nil || !got.Complete || got.Unsupported || len(got.Diagnostics) != 0 || !reflect.DeepEqual(got.Folds, want) {
		t.Fatalf("%q: result = %#v, err = %v; want complete with folds %#v", profile, got, err, want)
	}
	if source != original {
		t.Fatal("analysis changed the source")
	}
}

func checkStructureRejected(t *testing.T, source string, profile StructureProfile, code string) StructureDiagnostic {
	t.Helper()
	result, err := AnalyzeStructure(source, profile, DefaultStructureLimits())
	if err != nil || result.Complete || result.Unsupported || len(result.Folds) != 0 || len(result.Diagnostics) == 0 {
		t.Fatalf("%q: result = %#v, err = %v; want only diagnostics", profile, result, err)
	}
	diagnostic := result.Diagnostics[0]
	if diagnostic.Code != code || diagnostic.Message == "" {
		t.Fatalf("diagnostic = %#v, want code %q", diagnostic, code)
	}
	return diagnostic
}
