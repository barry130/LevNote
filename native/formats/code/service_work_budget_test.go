package code

import (
	"context"
	"go/ast"
	"go/token"
	"reflect"
	"strings"
	"testing"
)

func serviceWorkRun(ctx context.Context, req ServiceRequest, limit int) (ServiceResult, *goServiceWork) {
	run := &serviceRun{ctx: ctx, request: req, limits: req.Limits}
	work := &goServiceWork{limit: limit}
	return run.goResultWithWork(work), work
}

func serviceWantWorkLimit(t *testing.T, got ServiceResult, work *goServiceWork, limit int) {
	t.Helper()
	if got.Status != ServiceLimit || got.Resource != serviceGoWorkResource || got.Limit != limit {
		t.Fatalf("work limit %d after %d visits: %+v", limit, work.used, got)
	}
	if work.used != limit {
		t.Fatalf("performed %d work visits, want exact ceiling %d", work.used, limit)
	}
	serviceWantNoOutput(t, got)
}

func TestServiceGoWorkExactAndOneLess(t *testing.T) {
	const source = "\ufeffpackage p\r\n// 🙂 docs\r\nfunc f(变量 int) int { _ = 变量; return 变量 }"
	for _, operation := range []ServiceOperation{ServiceSymbols, ServiceCompletion, ServiceDefinition, ServiceReferences, ServiceInfo, ServiceRename} {
		t.Run(string(operation), func(t *testing.T) {
			req := serviceRequest(source, StructureGo, operation)
			req.ByteOffset, req.NewName = strings.Index(source, "变量 int"), "新名"
			want := serviceTestRun(req)
			serviceWantStatus(t, want, ServiceComplete)
			got, full := serviceWorkRun(context.Background(), req, serviceGoWorkLimit)
			if !reflect.DeepEqual(got, want) || full.used <= 1 {
				t.Fatalf("private/public result mismatch or missing accounting: used=%d got=%+v want=%+v", full.used, got, want)
			}
			got, exact := serviceWorkRun(context.Background(), req, full.used)
			if !reflect.DeepEqual(got, want) || exact.used != full.used {
				t.Fatalf("exact budget: used=%d got=%+v", exact.used, got)
			}
			// Sweep every work boundary in this tiny request, including partial
			// binding, accumulated references/completions, and rename planning.
			for limit := 0; limit < full.used; limit++ {
				got, work := serviceWorkRun(context.Background(), req, limit)
				serviceWantWorkLimit(t, got, work, limit)
			}
			if again := serviceTestRun(req); !reflect.DeepEqual(again, want) {
				t.Fatalf("a failed request changed the next independent request: %+v", again)
			}
			t.Logf("exact work visits=%d", full.used)
		})
	}
}

// This is the audit's multiplied lookup shape, scaled to small deterministic
// work. The outer parameter has no uses; every leaf x binds the inner shadow.
func serviceWorkInheritedFixture(height, occurrences int) string {
	var source strings.Builder
	source.WriteString("package p\nimport u \"unsafe\"\nfunc f(x/*outer*/ int) {\n{\nx := 1\n_ = x\n")
	for i := 0; i < height; i++ {
		source.WriteString("const (\n_ = u.Sizeof(func() {\n{\n")
	}
	source.WriteString("_ = []int{")
	source.WriteString(strings.Repeat("x,", occurrences))
	source.WriteString("}\n")
	for i := 0; i < height; i++ {
		source.WriteString("}\n})\n_\n)\n")
	}
	source.WriteString("}\n}\n")
	return source.String()
}

func TestServiceGoWorkInheritedShadowCannotEvadeBudget(t *testing.T) {
	source := serviceWorkInheritedFixture(4, 16)
	checked, err := AnalyzeStructure(source, StructureGo, DefaultStructureLimits())
	if err != nil || !checked.Complete {
		t.Fatalf("bounded source was not admitted: %+v %v", checked, err)
	}
	for _, operation := range []ServiceOperation{ServiceReferences, ServiceRename} {
		t.Run(string(operation), func(t *testing.T) {
			req := serviceRequest(source, StructureGo, operation)
			req.ByteOffset, req.NewName = strings.Index(source, "x/*outer*/"), "z"
			got, complete := serviceWorkRun(context.Background(), req, serviceGoWorkLimit)
			serviceWantStatus(t, got, ServiceComplete)
			if len(got.References) != 0 || (operation == ServiceRename && len(got.Edits) != 1) {
				t.Fatalf("shadowing changed: %+v", got)
			}
			// Finish the complete bind and start resolving occurrences before
			// exhaustion, so the limit is not merely an AST/output-count check.
			got, limited := serviceWorkRun(context.Background(), req, complete.used-10)
			serviceWantWorkLimit(t, got, limited, complete.used-10)
			t.Logf("source bytes=%d, bounded complete visits=%d; zero refs/one rename edit", len(source), complete.used)
		})
	}
}

func serviceWorkScopeChain(work *goServiceWork, depth int) *goServiceScope {
	root := newGoServiceScope(nil, true)
	root.work = work
	for i := 1; i < depth; i++ {
		root = newGoServiceScope(root, true)
	}
	return root
}

func TestServiceGoWorkCountsActualScopeAndProbeVisits(t *testing.T) {
	t.Run("scope-depth", func(t *testing.T) {
		run := &serviceRun{ctx: context.Background()}
		work := &goServiceWork{run: run, limit: 3}
		scope := serviceWorkScopeChain(work, 5)
		decl, opaque := scope.lookup("missing", token.Pos(20), nil, nil)
		if decl != nil || opaque {
			t.Fatalf("unexpected lookup: %+v %v", decl, opaque)
		}
		serviceWantWorkLimit(t, run.failure, work, 3)
	})
	for _, kind := range []string{"matching", "renamed", "target"} {
		t.Run(kind, func(t *testing.T) {
			run := &serviceRun{ctx: context.Background()}
			work := &goServiceWork{run: run, limit: 100}
			scope := serviceWorkScopeChain(work, 1)
			target := &goServiceDecl{id: &ast.Ident{Name: "x", NamePos: 1}, scope: scope}
			scope.decls = map[string]*goServiceDecl{"x": target}
			item := goServiceOccurrence{id: &ast.Ident{Name: "x", NamePos: 10}, scope: scope}
			for i := 0; i < 7; i++ {
				item.inherited = &goServiceConstProbe{parent: item.inherited, scope: scope, pos: 20}
			}
			binding := &goServiceBinding{run: run, work: work}
			switch kind {
			case "matching":
				if match, uncertain := binding.matchingUse(item, target); !match || uncertain {
					t.Fatalf("matching use changed: %v %v", match, uncertain)
				}
			case "renamed":
				if any, all := renamedBinding(item, target, "newName"); !any || !all {
					t.Fatalf("overlay changed: %v %v", any, all)
				}
			case "target":
				binding.file = token.NewFileSet().AddFile("", 1, 30)
				binding.uses = []goServiceOccurrence{item}
				run.request.ByteOffset = 9
				if decl, failure := binding.target(); decl != target || failure.Status != "" {
					t.Fatalf("target changed: %+v %+v", decl, failure)
				}
			}
			want := 1 + 7*2 // each probe and its one visited scope
			if kind == "target" {
				want++ // occurrence admission in at()
			}
			if work.used != want {
				t.Fatalf("used=%d want=%d: probe/scope work not charged", work.used, want)
			}
		})
	}
}

type serviceWorkCancelContext struct {
	context.Context
	work *goServiceWork
	at   int
}

func (c *serviceWorkCancelContext) Err() error {
	if c.work.used >= c.at {
		return context.Canceled
	}
	return nil
}

func TestServiceGoWorkCancellationInsideLookupAndProbes(t *testing.T) {
	for _, path := range []string{"scope", "probe", "opaque-ancestry", "rename-probe", "target-probe", "return-site"} {
		t.Run(path, func(t *testing.T) {
			run := &serviceRun{}
			work := &goServiceWork{run: run, limit: 100}
			run.ctx = &serviceWorkCancelContext{Context: context.Background(), work: work, at: 3}
			scope := serviceWorkScopeChain(work, 12)
			target := &goServiceDecl{id: &ast.Ident{Name: "x", NamePos: 1}, scope: scope, kind: "var"}
			scope.decls = map[string]*goServiceDecl{"x": target}
			item := goServiceOccurrence{id: &ast.Ident{Name: "x", NamePos: 10}, scope: scope}
			for i := 0; i < 12; i++ {
				item.inherited = &goServiceConstProbe{parent: item.inherited, scope: scope, pos: 20}
			}
			binding := &goServiceBinding{run: run, work: work}
			switch path {
			case "scope":
				scope.lookup("missing", 20, nil, nil)
			case "probe":
				binding.matchingUse(item, target)
			case "opaque-ancestry":
				scope.opaque = true
				target.scope = serviceWorkScopeChain(work, 1)
				binding.matchingUse(item, target)
			case "rename-probe":
				renamedBinding(item, target, "newName")
			case "target-probe":
				binding.file = token.NewFileSet().AddFile("", 1, 30)
				binding.uses = []goServiceOccurrence{item}
				run.request.ByteOffset = 9
				binding.target()
			case "return-site":
				target.scope = scope.parent.parent.parent.parent
				run.request.NewName = "newName"
				binding.returns = []goServiceReturnSite{{scope: scope, results: scope, pos: 20}}
				binding.rename(target)
			}
			serviceWantStatus(t, run.failure, ServiceCanceled)
			serviceWantNoOutput(t, run.failure)
			if work.used != 3 {
				t.Fatalf("cancellation was not checked inside work: used=%d", work.used)
			}
		})
	}
}

func TestServiceGoWorkCancellationWinsAtLimitAndAfterPartialWork(t *testing.T) {
	const source = "package p; func f(x int) { _ = x; _ = x; _ = x }"
	for _, operation := range []ServiceOperation{ServiceSymbols, ServiceCompletion, ServiceDefinition, ServiceReferences, ServiceInfo, ServiceRename} {
		t.Run(string(operation), func(t *testing.T) {
			req := serviceRequest(source, StructureGo, operation)
			req.ByteOffset, req.NewName = strings.Index(source, "x int"), "z"
			_, measured := serviceWorkRun(context.Background(), req, serviceGoWorkLimit)
			for at := 0; at <= measured.used; at++ {
				run := &serviceRun{request: req, limits: req.Limits}
				work := &goServiceWork{limit: at}
				run.ctx = &serviceWorkCancelContext{Context: context.Background(), work: work, at: at}
				got := run.goResultWithWork(work)
				serviceWantStatus(t, got, ServiceCanceled)
				serviceWantNoOutput(t, got)
				if work.used != at {
					t.Fatalf("cancel/limit at %d used=%d", at, work.used)
				}
			}
		})
	}
}

func TestServiceGoWorkCancellationOverridesExistingLimit(t *testing.T) {
	run := &serviceRun{ctx: context.Background()}
	work := &goServiceWork{run: run, limit: 3}
	scope := serviceWorkScopeChain(work, 5)
	scope.lookup("missing", 20, nil, nil)
	serviceWantWorkLimit(t, run.failure, work, 3)
	run.ctx = &serviceWorkCancelContext{Context: context.Background(), work: work, at: 3}
	if work.step() || work.used != 3 {
		t.Fatalf("work resumed after exhaustion/cancellation: %d", work.used)
	}
	serviceWantStatus(t, run.failure, ServiceCanceled)
	serviceWantNoOutput(t, run.failure)
}

func TestServiceGoWorkSameErrorPriority(t *testing.T) {
	// Conflicting rename declarations remain rejected before any use/probe
	// resolution; the work budget does not turn an admitted conflict into edits.
	const source = "package p; func f(x int) { y := 1; _, _ = x, y }"
	req := serviceRequest(source, StructureGo, ServiceRename)
	req.ByteOffset, req.NewName = strings.Index(source, "x int"), "y"
	want := serviceTestRun(req)
	if want.Status != ServiceAmbiguous || want.Reason != "rename-declaration-conflict" {
		t.Fatalf("conflict priority changed: %+v", want)
	}
	got, measured := serviceWorkRun(context.Background(), req, serviceGoWorkLimit)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("private/public priority: %+v %+v", got, want)
	}
	got, _ = serviceWorkRun(context.Background(), req, measured.used)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("same-boundary priority: %+v %+v", got, want)
	}
	t.Logf("conflict determined at %d charged visits", measured.used)
}

func TestServiceGoWorkCountGrowthIsMeasuredNotTimed(t *testing.T) {
	var counts [2]int
	for i, height := range []int{1, 4} {
		source := serviceWorkInheritedFixture(height, 12)
		req := serviceRequest(source, StructureGo, ServiceReferences)
		req.ByteOffset = strings.Index(source, "x/*outer*/")
		got, measured := serviceWorkRun(context.Background(), req, serviceGoWorkLimit)
		serviceWantStatus(t, got, ServiceComplete)
		counts[i] = measured.used
	}
	if counts[1] <= counts[0]*2 {
		t.Fatalf("deep const lookup work not reflected in counter: %v", counts)
	}
	t.Logf("bounded actual visits at H1/H4: %v", counts)
}
