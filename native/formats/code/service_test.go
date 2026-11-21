package code

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"unicode/utf16"
)

func serviceRequest(src string, profile StructureProfile, op ServiceOperation) ServiceRequest {
	return ServiceRequest{Source: src, Profile: profile, Operation: op, Limits: DefaultServiceLimits()}
}

func serviceTestRun(req ServiceRequest) ServiceResult {
	return AnalyzeService(context.Background(), req)
}

func serviceFind(t *testing.T, src, marker string) int {
	t.Helper()
	offset := strings.Index(src, marker)
	if offset < 0 {
		t.Fatalf("missing fixture marker %q", marker)
	}
	return offset
}

func serviceSpanAt(src string, start, end int) StructureSpan {
	return StructureSpan{
		Bytes: StructureByteRange{start, end},
		UTF16: StructureUTF16Range{len(utf16.Encode([]rune(src[:start]))), len(utf16.Encode([]rune(src[:end])))},
		Line:  strings.Count(src[:start], "\n"),
	}
}

func serviceWantStatus(t *testing.T, got ServiceResult, want ServiceStatus) {
	t.Helper()
	if got.Status != want {
		t.Fatalf("status %q (%s / %s %d), want %q: %+v", got.Status, got.Reason, got.Resource, got.Limit, want, got)
	}
}

func serviceWantNoOutput(t *testing.T, got ServiceResult) {
	t.Helper()
	if got.Diagnostics != nil || got.Symbols != nil || got.Completion != nil || got.Definition != nil ||
		got.References != nil || got.Info != nil || got.Edits != nil || got.NoOp {
		t.Fatalf("incomplete result has payload: %+v", got)
	}
}

func TestServiceGoBindingsAndOriginalCoordinates(t *testing.T) {
	src := "\ufeffpackage p\r\n//line elsewhere.go:50:7\r\n" +
		"// 🙂 docs\r\nfunc F(变量 int) int {\r\n" +
		"  var next = 变量 // 变量 in comment\r\n" +
		"  _ = \"变量🙂\"; _ = `变量`\n" +
		"  变量, also := next, 变量\n" +
		"  { 变量 := 变量 + 1; _ = func() int { return 变量 } }\n" +
		"  var s struct{ 变量 int }; _ = s.变量\n" +
		"  return 变量 + also\r\n}"
	param := serviceFind(t, src, "变量 int)")
	inner := serviceFind(t, src, "变量 :=")
	closureUse := serviceFind(t, src, "变量 } }")
	request := serviceRequest(src, StructureGo, ServiceDefinition)
	for _, item := range []struct{ at, decl int }{
		{closureUse, inner},
		{serviceFind(t, src, "变量 + 1"), param},
		{serviceFind(t, src, "变量, also"), param},
		{serviceFind(t, src, "变量 + also") + len("变"), param},
	} {
		request.ByteOffset = item.at
		got := serviceTestRun(request)
		serviceWantStatus(t, got, ServiceComplete)
		if got.Definition == nil || got.Definition.Name != "变量" || got.Definition.Span != serviceSpanAt(src, item.decl, item.decl+len("变量")) {
			t.Fatalf("definition at %d: %+v", item.at, got.Definition)
		}
	}
	request.ByteOffset = serviceFind(t, src, "s.变量") + len("s.")
	serviceWantStatus(t, serviceTestRun(request), ServiceUnsupported)
	request.ByteOffset = serviceFind(t, src, "变量 in comment")
	serviceWantStatus(t, serviceTestRun(request), ServiceUnsupported)
	request.ByteOffset = param + 1
	serviceWantStatus(t, serviceTestRun(request), ServiceInvalid)

	request.Operation, request.ByteOffset = ServiceReferences, param
	want := make([]StructureSpan, 0, 5)
	for _, marker := range []string{"变量 //", "变量, also", "变量\n", "变量 + 1", "变量 + also"} {
		i := serviceFind(t, src, marker)
		want = append(want, serviceSpanAt(src, i, i+len("变量")))
	}
	got := serviceTestRun(request)
	serviceWantStatus(t, got, ServiceComplete)
	if !reflect.DeepEqual(got.References, want) {
		t.Fatalf("references = %+v, want %+v", got.References, want)
	}
	got.References[0] = StructureSpan{}
	if again := serviceTestRun(request); !reflect.DeepEqual(again.References, want) {
		t.Fatalf("repeated references changed: %+v", again)
	}

	request.Operation = ServiceSymbols
	got = serviceTestRun(request)
	serviceWantStatus(t, got, ServiceComplete)
	wantNames := []string{"F", "变量", "next", "also", "变量", "s"}
	wantKinds := []string{"func", "parameter", "var", "var", "var", "var"}
	if len(got.Symbols) != len(wantNames) {
		t.Fatalf("symbols = %+v", got.Symbols)
	}
	for i, symbol := range got.Symbols {
		if symbol.Name != wantNames[i] || symbol.Kind != wantKinds[i] || src[symbol.Span.Bytes.Start:symbol.Span.Bytes.End] != symbol.Name {
			t.Fatalf("symbol %d = %+v", i, symbol)
		}
	}
	got.Symbols[1].Name = "injected"
	if again := serviceTestRun(request); again.Symbols[1].Name != "变量" {
		t.Fatalf("repeated symbols changed: %+v", again)
	}
}

func TestServiceGoInformationAndVocabulary(t *testing.T) {
	const src = "package p\r\n// Exact 🙂\r\n/* spaced\r\n   doc */\r\nfunc Solve[T any](\r\n\tval T,\r\n) T {\r\n\treturn val // Solo\r\n}\n"
	request := serviceRequest(src, StructureGo, ServiceInfo)
	request.ByteOffset = serviceFind(t, src, "Solve[")
	got := serviceTestRun(request)
	serviceWantStatus(t, got, ServiceComplete)
	if got.Info == nil || got.Info.Symbol.Name != "Solve" || got.Info.Signature != "func Solve[T any](\r\n\tval T,\r\n) T" ||
		got.Info.Documentation != "// Exact 🙂\r\n/* spaced\r\n   doc */" {
		t.Fatalf("information = %+v", got.Info)
	}
	got.Info.Signature = "injected"
	if again := serviceTestRun(request); again.Info.Signature == got.Info.Signature {
		t.Fatalf("repeated info changed: %+v", again)
	}
	request.Operation, request.Prefix = ServiceCompletion, "va"
	got = serviceTestRun(request)
	want := []ServiceCompletionItem{{"val", "file-word"}, {"var", "keyword"}}
	if got.Status != ServiceComplete || !reflect.DeepEqual(got.Completion, want) {
		t.Fatalf("completion = %+v, want %+v", got, want)
	}
	got.Completion[0].Text = "injected"
	if again := serviceTestRun(request); !reflect.DeepEqual(again.Completion, want) {
		t.Fatalf("repeated completion changed: %+v", again)
	}
	request.Prefix = "Solo"
	if got := serviceTestRun(request); got.Status != ServiceComplete || len(got.Completion) != 0 {
		t.Fatalf("comment leaked to vocabulary: %+v", got)
	}
}

func TestServiceJSONKeysAndCompletion(t *testing.T) {
	const src = "{\r\n  \"🙂\": [{\"\\uDEAD\": 1, \"\\uDEAD\": 2}], \"text\": \"no-key\", \"\\uD83D\\uDE42\": 0\n}"
	request := serviceRequest(src, StructureJSON, ServiceSymbols)
	got := serviceTestRun(request)
	serviceWantStatus(t, got, ServiceComplete)
	wantKeys := []string{`"🙂"`, `"\uDEAD"`, `"\uDEAD"`, `"text"`, `"\uD83D\uDE42"`}
	wantPaths := []string{`$["🙂"]`, `$["🙂"][0]["\uDEAD"]`, `$["🙂"][0]["\uDEAD"]`, `$["text"]`, `$["\uD83D\uDE42"]`}
	if len(got.Symbols) != len(wantKeys) {
		t.Fatalf("JSON symbols = %+v", got.Symbols)
	}
	last := 0
	for i, symbol := range got.Symbols {
		start := last + serviceFind(t, src[last:], wantKeys[i])
		last = start + len(wantKeys[i])
		if symbol.Name != wantKeys[i] || symbol.Path != wantPaths[i] || symbol.Kind != "json-key" || symbol.Span != serviceSpanAt(src, start, last) {
			t.Fatalf("symbol %d = %+v, key %q path %q", i, symbol, wantKeys[i], wantPaths[i])
		}
	}
	request.Operation, request.Prefix = ServiceCompletion, `"\uD`
	got = serviceTestRun(request)
	want := []ServiceCompletionItem{{`"\uD83D\uDE42"`, "json-key"}, {`"\uDEAD"`, "json-key"}}
	if got.Status != ServiceComplete || !reflect.DeepEqual(got.Completion, want) {
		t.Fatalf("JSON completion = %+v, want %+v", got, want)
	}
}

func TestServiceDiagnosticAndUnsupportedDistinction(t *testing.T) {
	for _, test := range []struct {
		source  string
		profile StructureProfile
		code    string
		point   string
	}{
		{"{\r\n\"🙂\": ]}", StructureJSON, "json-syntax", "]"},
		{"package p\n//line other:100\n//🙂\nvar x = @", StructureGo, "go-lexical", "@"},
		{"package p\nfunc f(,){ }", StructureGo, "go-syntax", ","},
		{"[\"🙂\",\xff]", StructureJSON, "invalid-utf8", "\xff"},
	} {
		request := serviceRequest(test.source, test.profile, ServiceDiagnostics)
		got := serviceTestRun(request)
		serviceWantStatus(t, got, ServiceInvalid)
		start := serviceFind(t, test.source, test.point)
		end := start + 1
		if test.code == "invalid-utf8" {
			end = start
		}
		if len(got.Diagnostics) != 1 || got.Diagnostics[0].Code != test.code || got.Diagnostics[0].Span != serviceSpanAt(test.source, start, end) {
			t.Fatalf("%q: %+v", test.source, got.Diagnostics)
		}
		request.Operation = ServiceSymbols
		if invalid := serviceTestRun(request); invalid.Status != ServiceInvalid || invalid.Symbols != nil || !reflect.DeepEqual(invalid.Diagnostics, got.Diagnostics) {
			t.Fatalf("invalid symbols result = %+v", invalid)
		}
	}
	for _, bad := range []string{"// comment\n{}", "{\"x\": 1,}", "{} true", "\ufeff{}", "\"\\u0G00\""} {
		serviceWantStatus(t, serviceTestRun(serviceRequest(bad, StructureJSON, ServiceDiagnostics)), ServiceInvalid)
	}
	for _, good := range []string{"true", "null", "-12.03e+2", " [true, null, \"\"] ", `"\uD800"`, `{"x":1,"x":2}`} {
		serviceWantStatus(t, serviceTestRun(serviceRequest(good, StructureJSON, ServiceDiagnostics)), ServiceComplete)
	}
	badProfile := serviceTestRun(serviceRequest("// comment\n{}", "jsonc", ServiceDiagnostics))
	serviceWantStatus(t, badProfile, ServiceUnsupported)
	serviceWantNoOutput(t, badProfile)
	unsupported := serviceTestRun(serviceRequest("{}", StructureJSON, ServiceDefinition))
	serviceWantStatus(t, unsupported, ServiceUnsupported)
	serviceWantNoOutput(t, unsupported)
}
