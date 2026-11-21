package code

import (
	"context"
	"math"
	"reflect"
	"strings"
	"testing"
)

func TestServiceRenameBoundOccurrencesOnly(t *testing.T) {
	const source = "\ufeffpackage p\r\n" +
		"func F(旧/*decl*/ int) int {\r\n" +
		"  旧/*assign*/, extra := 旧/*rhs*/+1, 8\r\n" +
		"  f := func() int { return 旧/*closure*/ }\n" +
		"  { 旧 := 旧/*shadow-init*/; _ = 旧 }\r\n" +
		"  var obj struct { 旧 int }; _ = obj.旧\n" +
		"  _ = \"旧🙂\"; _ = `旧` // 旧\r\n" +
		"  return f() + extra + 旧/*return*/\n}"
	const want = "\ufeffpackage p\r\n" +
		"func F(新名/*decl*/ int) int {\r\n" +
		"  新名/*assign*/, extra := 新名/*rhs*/+1, 8\r\n" +
		"  f := func() int { return 新名/*closure*/ }\n" +
		"  { 旧 := 新名/*shadow-init*/; _ = 旧 }\r\n" +
		"  var obj struct { 旧 int }; _ = obj.旧\n" +
		"  _ = \"旧🙂\"; _ = `旧` // 旧\r\n" +
		"  return f() + extra + 新名/*return*/\n}"
	request := serviceRequest(source, StructureGo, ServiceRename)
	request.ByteOffset = serviceMarked(t, source, "旧", "closure")
	request.NewName = "新名"
	got := serviceTestRun(request)
	serviceWantStatus(t, got, ServiceComplete)
	markers := []string{"decl", "assign", "rhs", "closure", "shadow-init", "return"}
	if len(got.Edits) != len(markers) || got.NoOp {
		t.Fatalf("rename = %+v", got)
	}
	for i, marker := range markers {
		start := serviceMarked(t, source, "旧", marker)
		if edit := got.Edits[i]; edit.Span != serviceSpanAt(source, start, start+len("旧")) || edit.OldText != "旧" || edit.NewText != "新名" {
			t.Fatalf("edit %d = %+v", i, edit)
		}
	}
	// Apply to the raw fixture from the end and compare to an independent
	// expected document, including every unedited CRLF, LF, and BOM byte.
	applied := source
	end := len(source)
	for i := len(got.Edits) - 1; i >= 0; i-- {
		edit := got.Edits[i]
		span := edit.Span.Bytes
		if span.End > end || applied[span.Start:span.End] != edit.OldText {
			t.Fatalf("overlapping or stale edit: %+v", edit)
		}
		applied = applied[:span.Start] + edit.NewText + applied[span.End:]
		end = span.Start
	}
	if applied != want || request.Source != source {
		t.Fatalf("applied = %q, want %q", applied, want)
	}
	preserved := append([]ServiceEdit(nil), got.Edits...)
	got.Edits[0].NewText = "injected"
	if again := serviceTestRun(request); !reflect.DeepEqual(again.Edits, preserved) {
		t.Fatalf("second plan = %+v", again)
	}
}

func TestServiceRenameRejectsConflictsAndCapture(t *testing.T) {
	for _, test := range []struct {
		name, statements, old, marker, replacement string
		status                                     ServiceStatus
		reason                                     string
	}{
		{"same-scope later", `x/*decl*/ := 0; y := 1; _, _ = x, y`, "x", "decl", "y", ServiceAmbiguous, "rename-declaration-conflict"},
		{"inner capture", `x/*decl*/ := 0; { y := 1; _ = func(){ _, _ = x, y } }`, "x", "decl", "y", ServiceAmbiguous, "rename-captured-use"},
		{"captured outer", `y := 1; { x/*decl*/ := 0; _, _ = x, y }`, "x", "decl", "y", ServiceAmbiguous, "rename-captures-use"},
		{"captured unresolved", `x/*decl*/ := 0; _, _ = x, notHere`, "x", "decl", "notHere", ServiceAmbiguous, "rename-captures-use"},
		{"captured builtin", `x/*decl*/ := 0; _, _ = x, len("abc")`, "x", "decl", "len", ServiceAmbiguous, "rename-captures-use"},
		{"possible composite key", `x/*decl*/ := 0; type M map[int]int; _ = M{x: 1}`, "x", "decl", "y", ServiceAmbiguous, "possible-reference"},
		{"embedded field name", `type T/*decl*/ int; _ = struct{T}{}`, "T", "decl", "U", ServiceUnsupported, "rename-embedded-field"},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := "package p; func f() {" + test.statements + "}"
			request := serviceRequest(source, StructureGo, ServiceRename)
			request.ByteOffset = serviceMarked(t, source, test.old, test.marker)
			request.NewName = test.replacement
			got := serviceTestRun(request)
			serviceWantStatus(t, got, test.status)
			serviceWantNoOutput(t, got)
			if got.Reason != test.reason {
				t.Fatalf("reason = %q, want %q", got.Reason, test.reason)
			}
		})
	}
}

func TestServiceRenameNakedReturnsAndDisjointScopes(t *testing.T) {
	const source = `package p; func f() (r/*result*/ int) { { inner/*decl*/ := 1; _ = inner; return } }`
	request := serviceRequest(source, StructureGo, ServiceRename)
	for _, test := range []struct{ name, marker, rename, reason string }{
		{"r", "result", "inner", "rename-captured-result"},
		{"inner", "decl", "r", "rename-captures-result"},
	} {
		request.ByteOffset = serviceMarked(t, source, test.name, test.marker)
		request.NewName = test.rename
		got := serviceTestRun(request)
		serviceWantStatus(t, got, ServiceAmbiguous)
		serviceWantNoOutput(t, got)
		if got.Reason != test.reason {
			t.Fatalf("reason = %q, want %q", got.Reason, test.reason)
		}
	}
	request.Source = `package p; var y = 1; func f() { x/*decl*/ := y; _ = x; { y := 2; _ = y }; _ = func(){ y := 3; _ = y } }`
	request.ByteOffset = serviceMarked(t, request.Source, "x", "decl")
	request.NewName = "y"
	got := serviceTestRun(request)
	serviceWantStatus(t, got, ServiceComplete)
	if len(got.Edits) != 2 {
		t.Fatalf("harmless shadowing should change declaration and its one use: %+v", got)
	}
}

func TestServiceRenameNameAndTargetAdmission(t *testing.T) {
	const source = `package p; var pkg/*pkg*/ int; func f(x/*param*/ int) { _ = x }`
	request := serviceRequest(source, StructureGo, ServiceRename)
	request.ByteOffset = serviceMarked(t, source, "x", "param")
	for _, name := range []string{"", "_", "return", "1abc", "ab cd", "á", "🙂", "\xff"} {
		request.NewName = name
		got := serviceTestRun(request)
		serviceWantStatus(t, got, ServiceInvalid)
		serviceWantNoOutput(t, got)
	}
	request.NewName = "x"
	got := serviceTestRun(request)
	if got.Status != ServiceComplete || !got.NoOp || got.Edits != nil {
		t.Fatalf("same-name local rename = %+v", got)
	}
	request.ByteOffset, request.NewName = serviceMarked(t, source, "pkg", "pkg"), "updated"
	got = serviceTestRun(request)
	serviceWantStatus(t, got, ServiceUnsupported)
	serviceWantNoOutput(t, got)
}

func TestServiceRenameChecksOldFieldBudgetBeforePlanning(t *testing.T) {
	old := strings.Repeat("x", serviceFieldBytes+1)
	source := "package p; func f(" + old + " int) { _ = " + old + " }"
	request := serviceRequest(source, StructureGo, ServiceRename)
	request.ByteOffset = strings.Index(source, old)
	request.NewName = "shortName"
	got := serviceTestRun(request)
	if got.Status != ServiceLimit || got.Resource != serviceFieldResource || got.Limit != serviceFieldBytes {
		t.Fatalf("oversized old identifier = %+v", got)
	}
	serviceWantNoOutput(t, got)

	request.NewName = old
	got = serviceTestRun(request)
	if got.Status != ServiceLimit || got.Resource != serviceFieldResource || got.Limit != serviceFieldBytes {
		t.Fatalf("oversized same-name no-op = %+v", got)
	}
	serviceWantNoOutput(t, got)
}

func TestServiceRenameExactBudgetAndCancellation(t *testing.T) {
	request := serviceRequest("package p; func f(x int) {"+strings.Repeat("_ = x;", 1023)+"}", StructureGo, ServiceRename)
	request.ByteOffset, request.NewName = serviceFind(t, request.Source, "x int"), "y"
	request.Limits.RenameEdits = math.MaxInt
	got := serviceTestRun(request)
	serviceWantStatus(t, got, ServiceComplete)
	if len(got.Edits) != 1024 {
		t.Fatalf("exact rename cap: %d", len(got.Edits))
	}
	request.Source = strings.TrimSuffix(request.Source, "}") + "_ = x;}"
	got = serviceTestRun(request)
	serviceWantStatus(t, got, ServiceLimit)
	if got.Resource != serviceRenameEditsResource || got.Limit != 1024 {
		t.Fatalf("rename cap: %+v", got)
	}
	serviceWantNoOutput(t, got)

	request.Source = "package p; func f(x int) { _ = x }"
	request.Limits.TextBytes = 4 // two edits, each retaining both one-byte strings
	got = serviceTestRun(request)
	if got.Status != ServiceComplete || len(got.Edits) != 2 {
		t.Fatalf("at exact text cap: %+v", got)
	}
	request.Limits.TextBytes--
	got = serviceTestRun(request)
	if got.Status != ServiceLimit || got.Resource != serviceTextResource {
		t.Fatalf("rename text budget: %+v", got)
	}
	serviceWantNoOutput(t, got)

	request.Limits.TextBytes = DefaultServiceLimits().TextBytes
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	gate := &serviceCancelAfter{Context: ctx, cancel: cancel, at: math.MaxInt}
	if complete := AnalyzeService(gate, request); complete.Status != ServiceComplete || len(complete.Edits) != 2 {
		t.Fatalf("exact request before cancellation: %+v", complete)
	}
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	late := &serviceCancelAfter{Context: ctx, cancel: stop, at: gate.checks - 1}
	got = AnalyzeService(late, request)
	serviceWantStatus(t, got, ServiceCanceled)
	serviceWantNoOutput(t, got)
}
