package code

import (
	"context"
	"math"
	"strings"
	"testing"
)

func TestServiceBudgetsExactAndPlusOne(t *testing.T) {
	for _, test := range []struct {
		resource StructureResource
		limit    int
		profile  StructureProfile
		op       ServiceOperation
		pass     string
		fail     string
		set      func(*ServiceLimits, int)
		at       string
		valid    ServiceStatus
	}{
		{StructureBytes, 3, StructureJSON, ServiceDiagnostics, `"a"`, `"ab"`, func(l *ServiceLimits, n int) { l.Structure.Bytes = n }, "", ServiceComplete},
		{StructureLines, 2, StructureJSON, ServiceDiagnostics, "0\n", "0\n\n", func(l *ServiceLimits, n int) { l.Structure.Lines = n }, "", ServiceComplete},
		{StructureTokens, 3, StructureJSON, ServiceDiagnostics, "[0]", "[[]]", func(l *ServiceLimits, n int) { l.Structure.Tokens = n }, "", ServiceComplete},
		{StructureDepth, 2, StructureJSON, ServiceDiagnostics, "[[]]", "[[[]]]", func(l *ServiceLimits, n int) { l.Structure.Depth = n }, "", ServiceComplete},
		{StructureFolds, 1, StructureJSON, ServiceDiagnostics, "[\n0\n]", "[\n[\n0\n]\n]", func(l *ServiceLimits, n int) { l.Structure.Folds = n }, "", ServiceComplete},
		{StructureDiagnostics, 1, StructureGo, ServiceDiagnostics, "package p\nvar a = @", "package p\nvar a = @ @", func(l *ServiceLimits, n int) { l.Structure.Diagnostics = n }, "", ServiceInvalid},
		{serviceSymbolsResource, 1, StructureJSON, ServiceSymbols, `{"a":0}`, `{"a":0,"a":1}`, func(l *ServiceLimits, n int) { l.Symbols = n }, "", ServiceComplete},
		{serviceCompletionResource, 1, StructureJSON, ServiceCompletion, `{"a":0,"a":1}`, `{"a":0,"b":1}`, func(l *ServiceLimits, n int) { l.Completion = n }, "", ServiceComplete},
		{serviceReferencesResource, 1, StructureGo, ServiceReferences, "package p; func f(x int) { _ = x }", "package p; func f(x int) { _ = x; _ = x }", func(l *ServiceLimits, n int) { l.References = n }, "x int", ServiceComplete},
		{serviceTextResource, 11, StructureJSON, ServiceSymbols, `{"ab":0}`, `{"ab":0,"":1}`, func(l *ServiceLimits, n int) { l.TextBytes = n }, "", ServiceComplete},
	} {
		t.Run(string(test.resource), func(t *testing.T) {
			request := serviceRequest(test.pass, test.profile, test.op)
			if test.at != "" {
				request.ByteOffset = serviceFind(t, test.pass, test.at)
			}
			test.set(&request.Limits, test.limit)
			got := serviceTestRun(request)
			serviceWantStatus(t, got, test.valid)
			if test.resource == StructureDiagnostics && len(got.Diagnostics) != 1 {
				t.Fatalf("diagnostics = %+v", got.Diagnostics)
			}
			request.Source = test.fail
			if test.resource == serviceTextResource {
				// Same 11-byte output, one byte above a 10-byte cap.
				request.Source = test.pass
				test.set(&request.Limits, test.limit-1)
			}
			got = serviceTestRun(request)
			serviceWantStatus(t, got, ServiceLimit)
			if got.Resource != test.resource || (got.Limit != test.limit && test.resource != serviceTextResource) {
				t.Fatalf("reported wrong limit: %+v", got)
			}
			serviceWantNoOutput(t, got)
			test.set(&request.Limits, 0)
			got = serviceTestRun(request)
			serviceWantStatus(t, got, ServiceInvalid)
			serviceWantNoOutput(t, got)
		})
	}
}

func TestServiceHardLimits(t *testing.T) {
	request := serviceRequest(strings.Repeat(" ", (2<<20)-1)+"0", StructureJSON, ServiceDiagnostics)
	request.Limits.Structure.Bytes = math.MaxInt
	serviceWantStatus(t, serviceTestRun(request), ServiceComplete)
	request.Source += " "
	got := serviceTestRun(request)
	if got.Status != ServiceLimit || got.Resource != StructureBytes || got.Limit != 2<<20 {
		t.Fatalf("byte clamp: %+v", got)
	}
	serviceWantNoOutput(t, got)

	request = serviceRequest(`{"`+strings.Repeat("x", serviceFieldBytes-2)+`":0}`, StructureJSON, ServiceCompletion)
	serviceWantStatus(t, serviceTestRun(request), ServiceComplete)
	request.Source = `{"` + strings.Repeat("x", serviceFieldBytes-1) + `":0}`
	got = serviceTestRun(request)
	if got.Status != ServiceLimit || got.Resource != serviceFieldResource || got.Limit != serviceFieldBytes {
		t.Fatalf("field limit: %+v", got)
	}
	serviceWantNoOutput(t, got)

	request = serviceRequest(`{"`+strings.Repeat("x", serviceFieldBytes-5)+`":0}`, StructureJSON, ServiceSymbols)
	serviceWantStatus(t, serviceTestRun(request), ServiceComplete)
	request.Source = `{"` + strings.Repeat("x", serviceFieldBytes-4) + `":0}`
	got = serviceTestRun(request)
	if got.Status != ServiceLimit || got.Resource != serviceFieldResource {
		t.Fatalf("path field limit: %+v", got)
	}
	serviceWantNoOutput(t, got)

	request = serviceRequest("package p; var x = "+strings.Repeat("1+", serviceASTDepth-4)+"1", StructureGo, ServiceSymbols)
	serviceWantStatus(t, serviceTestRun(request), ServiceComplete)
	request.Source += "+1"
	got = serviceTestRun(request)
	if got.Status != ServiceLimit || got.Resource != serviceASTResource || got.Limit != serviceASTDepth {
		t.Fatalf("AST limit: %+v", got)
	}
	serviceWantNoOutput(t, got)

	request = serviceRequest("0", StructureJSON, ServiceDiagnostics)
	request.Limits.RenameEdits = -1
	got = serviceTestRun(request)
	serviceWantStatus(t, got, ServiceInvalid)
	serviceWantNoOutput(t, got)
}

type serviceCancelAfter struct {
	context.Context
	cancel context.CancelFunc
	checks int
	at     int
}

func (c *serviceCancelAfter) Err() error {
	c.checks++
	if c.checks == c.at {
		c.cancel()
	}
	return c.Context.Err()
}

func TestServiceCancellationNeverReturnsPartialPayload(t *testing.T) {
	request := serviceRequest(`{"a":1,"b":2,"c":3,"d":4}`, StructureJSON, ServiceSymbols)
	for _, at := range []int{1, 2, 15} {
		ctx, cancel := context.WithCancel(context.Background())
		gate := &serviceCancelAfter{Context: ctx, cancel: cancel, at: at}
		got := AnalyzeService(gate, request)
		cancel()
		serviceWantStatus(t, got, ServiceCanceled)
		serviceWantNoOutput(t, got)
	}
	if got := serviceTestRun(request); got.Status != ServiceComplete || len(got.Symbols) != 4 {
		t.Fatalf("valid request after cancel: %+v", got)
	}
}
