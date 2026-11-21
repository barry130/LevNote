package code

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestStructureBudgetsExactLimitAndOneOver(t *testing.T) {
	const maxInt = int(^uint(0) >> 1)
	hard := StructureLimits{2 << 20, 65_536, 262_144, 256, 4_096, 256}
	if got := DefaultStructureLimits(); got != hard {
		t.Fatalf("hard ceilings = %#v, want %#v", got, hard)
	}
	oversized := StructureLimits{maxInt, maxInt, maxInt, maxInt, maxInt, maxInt}
	type fixture struct {
		fits, exceeds string
		folds, diags  int
	}
	cases := []struct {
		resource StructureResource
		profile  StructureProfile
		small    int
		field    func(*StructureLimits) *int
		fixture  func(int) fixture
	}{
		{StructureBytes, StructureJSON, 7, func(l *StructureLimits) *int { return &l.Bytes }, func(n int) fixture {
			// A single line, exactly n bytes but n-2 UTF-16 code units.
			fits := `"🙂` + strings.Repeat(" ", n-6) + `"`
			return fixture{fits: fits, exceeds: fits + " "}
		}},
		{StructureLines, StructureJSON, 3, func(l *StructureLimits) *int { return &l.Lines }, func(n int) fixture {
			fits := "0" + strings.Repeat("\r\n", n-1)
			return fixture{fits: fits, exceeds: fits + "\n"}
		}},
		{StructureTokens, StructureJSON, 6, func(l *StructureLimits) *int { return &l.Tokens }, func(n int) fixture {
			prefix := "[" + strings.Repeat("0,", (n-4)/2)
			return fixture{fits: prefix + "[]]", exceeds: prefix + "[0]]"}
		}},
		{StructureDepth, StructureJSON, 2, func(l *StructureLimits) *int { return &l.Depth }, func(n int) fixture {
			fits := strings.Repeat("[", n) + "\n0\n" + strings.Repeat("]", n)
			return fixture{fits: fits, exceeds: "[" + fits + "]", folds: n}
		}},
		{StructureFolds, StructureJSON, 2, func(l *StructureLimits) *int { return &l.Folds }, func(n int) fixture {
			prefix := "[\n" + strings.Repeat("[\n0\n],\n", n-2)
			return fixture{fits: prefix + "[\n0\n]\n]", exceeds: prefix + "[\n0\n],\n[\n0\n]\n]", folds: n}
		}},
		{StructureDiagnostics, StructureGo, 2, func(l *StructureLimits) *int { return &l.Diagnostics }, func(n int) fixture {
			// One token can emit all n lexical diagnostics before the closing
			// brace; the valid delimiter pair cannot make the result foldable.
			prefix := "package p\nfunc f() {\n _ = \"" + strings.Repeat(`\q`, n)
			return fixture{fits: prefix + "\"\n}", exceeds: prefix + "\\q\"\n}", diags: n}
		}},
	}
	for _, tt := range cases {
		t.Run(string(tt.resource), func(t *testing.T) {
			t.Run("small", func(t *testing.T) {
				limits := hard
				*tt.field(&limits) = tt.small
				f := tt.fixture(tt.small)
				assertStructureFits(t, f.fits, tt.profile, limits, f.folds, f.diags)
				assertStructureBudget(t, f.exceeds, tt.profile, limits, tt.resource, tt.small)
			})
			t.Run("hard-and-clamp", func(t *testing.T) {
				n := *tt.field(&hard)
				f := tt.fixture(n)
				assertStructureFits(t, f.fits, tt.profile, hard, f.folds, f.diags)
				assertStructureBudget(t, f.exceeds, tt.profile, hard, tt.resource, n)
				assertStructureFits(t, f.fits, tt.profile, oversized, f.folds, f.diags)
				assertStructureBudget(t, f.exceeds, tt.profile, oversized, tt.resource, n)
			})
		})
	}
}

func TestStructureNonpositiveLimitsAlwaysRejected(t *testing.T) {
	fields := []func(*StructureLimits) *int{
		func(l *StructureLimits) *int { return &l.Bytes },
		func(l *StructureLimits) *int { return &l.Lines },
		func(l *StructureLimits) *int { return &l.Tokens },
		func(l *StructureLimits) *int { return &l.Depth },
		func(l *StructureLimits) *int { return &l.Folds },
		func(l *StructureLimits) *int { return &l.Diagnostics },
	}
	for i, field := range fields {
		for _, n := range []int{0, -1, -int(^uint(0)>>1) - 1} {
			limits := DefaultStructureLimits()
			*field(&limits) = n
			for _, profile := range []StructureProfile{StructureJSON, "jsonc"} {
				got, err := AnalyzeStructure(`0`, profile, limits)
				if !errors.Is(err, ErrInvalidStructureLimits) || !reflect.DeepEqual(got, StructureResult{}) {
					t.Fatalf("limit %d=%d on %s: result = %#v, err = %v", i, n, profile, got, err)
				}
			}
		}
	}
}

func TestStructureBudgetAdmissionAndNoPartialResults(t *testing.T) {
	t.Run("byte-precheck-precedes-source-inspection", func(t *testing.T) {
		limits := DefaultStructureLimits()
		limits.Bytes = 3
		assertStructureBudget(t, "\xff\xff\xff\xff", StructureJSON, limits, StructureBytes, 3)
	})
	t.Run("go-comments-and-semicolons-are-events", func(t *testing.T) {
		const source = "package p\r\n// marker\nfunc f() {\n f()\n}"
		limits := DefaultStructureLimits()
		limits.Tokens = 15
		assertStructureFits(t, source, StructureGo, limits, 1, 0)
		limits.Tokens-- // final implicit semicolon, after the fold was found
		assertStructureBudget(t, source, StructureGo, limits, StructureTokens, 14)
	})
	t.Run("go-matches-depth-before-growing-the-stack", func(t *testing.T) {
		const source = "package p\nvar x = (((0)))"
		limits := DefaultStructureLimits()
		limits.Depth = 3
		assertStructureFits(t, source, StructureGo, limits, 0, 0)
		limits.Depth--
		assertStructureBudget(t, source, StructureGo, limits, StructureDepth, 2)
	})
	t.Run("accepted-diagnostics-do-not-mask-later-exhaustion", func(t *testing.T) {
		limits := DefaultStructureLimits()
		limits.Tokens = 2
		assertStructureBudget(t, "@ @ @", StructureGo, limits, StructureTokens, 2)
	})
	t.Run("go-grammar-validator-limit-is-not-a-syntax-diagnostic", func(t *testing.T) {
		const prefix = "package p\nfunc f() {\n f()\n}\nvar x = "
		source := prefix + strings.Repeat("!", 100_005) + "true"
		result, err := AnalyzeStructure(source, StructureGo, DefaultStructureLimits())
		if !errors.Is(err, ErrStructureValidationLimit) || !reflect.DeepEqual(result, StructureResult{}) {
			t.Fatalf("Go validation limit: result = %#v, err = %v", result, err)
		}
	})
}

func assertStructureFits(t *testing.T, source string, profile StructureProfile, limits StructureLimits, folds, diags int) {
	t.Helper()
	result, err := AnalyzeStructure(source, profile, limits)
	if err != nil || result.Complete != (diags == 0) || result.Unsupported || len(result.Folds) != folds || len(result.Diagnostics) != diags {
		t.Fatalf("%q at limit: bytes=%d, complete=%t, unsupported=%t, folds=%d, diagnostics=%d, err=%v; want %d folds, %d diagnostics",
			profile, len(source), result.Complete, result.Unsupported, len(result.Folds), len(result.Diagnostics), err, folds, diags)
	}
	for _, d := range result.Diagnostics {
		if d.Code != "go-lexical" {
			t.Fatalf("unexpected diagnostic: %#v", d)
		}
	}
}

func assertStructureBudget(t *testing.T, source string, profile StructureProfile, limits StructureLimits, resource StructureResource, limit int) {
	t.Helper()
	result, err := AnalyzeStructure(source, profile, limits)
	var budget *StructureBudgetError
	if !errors.As(err, &budget) || budget.Resource != resource || budget.Limit != limit || !reflect.DeepEqual(result, StructureResult{}) {
		t.Fatalf("%q over limit: bytes=%d, complete=%t, unsupported=%t, folds=%d, diagnostics=%d, err=%#v; want %s limit %d and no partial result",
			profile, len(source), result.Complete, result.Unsupported, len(result.Folds), len(result.Diagnostics), err, resource, limit)
	}
}
