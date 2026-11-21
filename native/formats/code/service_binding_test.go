package code

import "testing"

func serviceMarked(t *testing.T, source, name, marker string) int {
	t.Helper()
	return serviceFind(t, source, name+"/*"+marker+"*/")
}

func TestServiceGoControlScopes(t *testing.T) {
	const src = `package demo
func F(items []int, x/*param*/ int, a any, ch <-chan int) (result/*result*/ int) {
  const amount/*const*/ = 4
  type Node/*type*/ struct { next *Node/*recursive*/ }
  var _ Node/*var-type*/
  if x/*if*/, y := x/*if-rhs*/, 1; x/*if-cond*/ > y {
    _ = x/*if-body*/
  } else { _ = x/*else*/ }
  for i, x/*range*/ := range items { result/*result-use*/ += i + x/*range-body*/ }
  for x/*for*/ := x/*for-init*/; x/*for-test*/ > 0; x/*for-post*/-- { _ = x/*for-body*/ }
  switch x/*switch*/ := x/*switch-rhs*/; x/*switch-test*/ {
  case 1:
    x/*case*/ := x/*case-rhs*/
    _ = x/*case-body*/
  default:
    _ = x/*other-case*/
  }
  select {
  case x/*comm*/ := <-ch:
    _ = x/*comm-body*/
  default:
    _ = x/*select-default*/
  }
  switch x/*guard*/ := a.(type) {
  case int: _ = x/*type-case*/
  default: _ = x/*type-default*/
  }
  x/*label*/: for { break x/*break*/ }
  result += x/*after*/ + amount/*amount-use*/
  return
}`
	request := serviceRequest(src, StructureGo, ServiceDefinition)
	for _, item := range []struct{ name, use, decl, kind string }{
		{"result", "result-use", "result", "result"},
		{"amount", "amount-use", "const", "const"},
		{"Node", "recursive", "type", "type"},
		{"Node", "var-type", "type", "type"},
		{"x", "if-rhs", "param", "parameter"},
		{"x", "if-cond", "if", "var"},
		{"x", "if-body", "if", "var"},
		{"x", "else", "if", "var"},
		{"x", "range-body", "range", "var"},
		{"x", "for-init", "param", "parameter"},
		{"x", "for-test", "for", "var"},
		{"x", "for-post", "for", "var"},
		{"x", "for-body", "for", "var"},
		{"x", "switch-rhs", "param", "parameter"},
		{"x", "switch-test", "switch", "var"},
		{"x", "case-rhs", "switch", "var"},
		{"x", "case-body", "case", "var"},
		{"x", "other-case", "switch", "var"},
		{"x", "comm-body", "comm", "var"},
		{"x", "select-default", "param", "parameter"},
		{"x", "after", "param", "parameter"},
	} {
		t.Run(item.use, func(t *testing.T) {
			request.ByteOffset = serviceMarked(t, src, item.name, item.use)
			got := serviceTestRun(request)
			serviceWantStatus(t, got, ServiceComplete)
			start := serviceMarked(t, src, item.name, item.decl)
			if got.Definition == nil || got.Definition.Kind != item.kind || got.Definition.Span != serviceSpanAt(src, start, start+len(item.name)) {
				t.Fatalf("definition = %+v", got.Definition)
			}
		})
	}
	for _, marker := range []string{"guard", "type-case", "type-default", "label", "break"} {
		request.ByteOffset = serviceMarked(t, src, "x", marker)
		got := serviceTestRun(request)
		serviceWantStatus(t, got, ServiceUnsupported)
		serviceWantNoOutput(t, got)
	}
}

func TestServiceGoGenericAndCompositeRoles(t *testing.T) {
	const src = `package p
import ext "example.invalid/not-loaded"
type Pair[A/*type-generic*/ any, B any] struct { First A/*field-type*/; Second B }
func (r/*recv*/ Pair[A/*method-generic*/, B]) Get(input A/*method-arg*/) A/*method-return*/ { _ = r/*recv-use*/; return input }
func Later/*later*/() {}
func Use[T/*func-generic*/ interface{ ~int }](x T/*func-arg*/) {
  const index/*index*/ = 0
  _ = [1]int{index/*array-key*/: 5}
  _ = map[int]T{index/*map-key*/: x}
  _ = struct{index int}{index/*struct-key*/: 5}
  _ = ext/*qualifier*/.Any/*selector*/
  Later/*call*/()
}`
	request := serviceRequest(src, StructureGo, ServiceDefinition)
	for _, item := range []struct{ name, marker, decl string }{
		{"A", "field-type", "type-generic"}, {"A", "method-arg", "method-generic"},
		{"A", "method-return", "method-generic"}, {"r", "recv-use", "recv"},
		{"T", "func-arg", "func-generic"}, {"index", "array-key", "index"},
		{"index", "map-key", "index"}, {"Later", "call", "later"},
	} {
		request.ByteOffset = serviceMarked(t, src, item.name, item.marker)
		got := serviceTestRun(request)
		serviceWantStatus(t, got, ServiceComplete)
		start := serviceMarked(t, src, item.name, item.decl)
		if got.Definition == nil || got.Definition.Span != serviceSpanAt(src, start, start+len(item.name)) {
			t.Fatalf("%s definition = %+v", item.marker, got.Definition)
		}
	}
	for _, item := range []struct{ name, marker string }{{"index", "struct-key"}, {"ext", "qualifier"}, {"Any", "selector"}} {
		request.ByteOffset = serviceMarked(t, src, item.name, item.marker)
		serviceWantStatus(t, serviceTestRun(request), ServiceUnsupported)
	}
}

func TestServiceGoInheritedConstantBinding(t *testing.T) {
	const source = `package p
const n/*package*/ = 7
func f() {
  const (
    n/*local*/ = n/*shared*/ + 1
    second
  )
  _, _ = n/*use*/, second
}
func g() {
  const (
    x/*candidate*/ = n/*capture*/
    second
  )
  _, _ = x, second
}`
	request := serviceRequest(source, StructureGo, ServiceDefinition)
	request.ByteOffset = serviceMarked(t, source, "n", "shared")
	got := serviceTestRun(request)
	serviceWantStatus(t, got, ServiceAmbiguous)
	serviceWantNoOutput(t, got)
	request.Operation = ServiceReferences
	for _, marker := range []string{"package", "local"} {
		request.ByteOffset = serviceMarked(t, source, "n", marker)
		got = serviceTestRun(request)
		serviceWantStatus(t, got, ServiceAmbiguous)
		serviceWantNoOutput(t, got)
	}
	request.Operation = ServiceRename
	request.ByteOffset = serviceMarked(t, source, "n", "local")
	request.NewName = "localName"
	got = serviceTestRun(request)
	serviceWantStatus(t, got, ServiceAmbiguous)
	serviceWantNoOutput(t, got)
	request.ByteOffset = serviceMarked(t, source, "x", "candidate")
	request.NewName = "n"
	got = serviceTestRun(request)
	serviceWantStatus(t, got, ServiceAmbiguous)
	serviceWantNoOutput(t, got)
	if got.Reason != "rename-captures-use" {
		t.Fatalf("inherited capture: %+v", got)
	}
}

func TestServiceGoAmbiguityDoesNotInventReferences(t *testing.T) {
	const source = `package p
func F(key/*decl*/ int) {
  type Table map[int]int
  _ = Table{key/*possible*/: 1}
  var d, d int
  _ = d/*duplicate*/
  _ = missing/*unresolved*/
}`
	request := serviceRequest(source, StructureGo, ServiceDefinition)
	for _, item := range []struct {
		name, marker string
		status       ServiceStatus
	}{{"key", "possible", ServiceAmbiguous}, {"d", "duplicate", ServiceAmbiguous}, {"missing", "unresolved", ServiceUnsupported}} {
		request.ByteOffset = serviceMarked(t, source, item.name, item.marker)
		got := serviceTestRun(request)
		serviceWantStatus(t, got, item.status)
		serviceWantNoOutput(t, got)
	}
	request.Operation, request.ByteOffset = ServiceReferences, serviceMarked(t, source, "key", "decl")
	got := serviceTestRun(request)
	serviceWantStatus(t, got, ServiceAmbiguous)
	serviceWantNoOutput(t, got)
}

func TestServiceGoUnknownImportScopesAreConservative(t *testing.T) {
	const defaultImport = `package p
import "example.invalid/yaml.v3"
var yaml/*pkg*/ = 1
func use() {
  _ = yaml/*root*/
  _ = yaml/*selector*/.Node
}`
	request := serviceRequest(defaultImport, StructureGo, ServiceDefinition)
	for _, operation := range []ServiceOperation{ServiceDefinition, ServiceInfo} {
		request.Operation = operation
		for _, marker := range []string{"root", "selector"} {
			request.ByteOffset = serviceMarked(t, defaultImport, "yaml", marker)
			got := serviceTestRun(request)
			serviceWantStatus(t, got, ServiceUnsupported)
			serviceWantNoOutput(t, got)
		}
	}
	request.Operation = ServiceReferences
	request.ByteOffset = serviceMarked(t, defaultImport, "yaml", "pkg")
	got := serviceTestRun(request)
	serviceWantStatus(t, got, ServiceAmbiguous)
	serviceWantNoOutput(t, got)

	const dotImport = `package p
import . "example.invalid/external"
var External/*pkg*/ = 1
func use() { _ = External/*use*/ }`
	request = serviceRequest(dotImport, StructureGo, ServiceDefinition)
	request.ByteOffset = serviceMarked(t, dotImport, "External", "use")
	got = serviceTestRun(request)
	serviceWantStatus(t, got, ServiceUnsupported)
	serviceWantNoOutput(t, got)
	request.Operation = ServiceReferences
	request.ByteOffset = serviceMarked(t, dotImport, "External", "pkg")
	got = serviceTestRun(request)
	serviceWantStatus(t, got, ServiceAmbiguous)
	serviceWantNoOutput(t, got)
}

func TestServiceGoKnownImportNamesAndLocalShadowing(t *testing.T) {
	const explicitAlias = `package p
import y "example.invalid/yaml.v3"
var yaml/*pkg*/ = 1
func use() { _ = yaml/*use*/ }`
	request := serviceRequest(explicitAlias, StructureGo, ServiceDefinition)
	request.ByteOffset = serviceMarked(t, explicitAlias, "yaml", "use")
	got := serviceTestRun(request)
	serviceWantStatus(t, got, ServiceComplete)
	if got.Definition == nil || got.Definition.Span != serviceSpanAt(explicitAlias,
		serviceMarked(t, explicitAlias, "yaml", "pkg"), serviceMarked(t, explicitAlias, "yaml", "pkg")+len("yaml")) {
		t.Fatalf("explicit alias incorrectly blocked package declaration: %+v", got.Definition)
	}

	const blankImport = `package p
import _ "example.invalid/yaml.v3"
var yaml/*pkg*/ = 1
func use() { _ = yaml/*use*/ }`
	request = serviceRequest(blankImport, StructureGo, ServiceDefinition)
	request.ByteOffset = serviceMarked(t, blankImport, "yaml", "use")
	got = serviceTestRun(request)
	serviceWantStatus(t, got, ServiceComplete)

	const localShadow = `package p
import "example.invalid/yaml.v3"
var yaml = 1
func use() { yaml/*local*/ := 2; _ = yaml/*local-use*/ }`
	request = serviceRequest(localShadow, StructureGo, ServiceRename)
	request.ByteOffset = serviceMarked(t, localShadow, "yaml", "local-use")
	request.NewName = "localName"
	got = serviceTestRun(request)
	serviceWantStatus(t, got, ServiceComplete)
	if len(got.Edits) != 2 {
		t.Fatalf("local shadow rename = %+v", got)
	}
	request.Operation = ServiceReferences
	got = serviceTestRun(request)
	serviceWantStatus(t, got, ServiceComplete)
	if len(got.References) != 1 {
		t.Fatalf("local shadow references = %+v", got)
	}
}
