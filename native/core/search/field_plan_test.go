package search

import (
	"errors"
	"math"
	"regexp"
	"strings"
	"testing"

	"levnote.local/native/core/text"
)

func TestFieldPlanBoundedMatchesShareSemantics(t *testing.T) {
	for _, opts := range []Options{
		{Query: "cat"}, {Query: "cat", WholeWord: true}, {Query: "cat", CaseSensitive: true},
		{Query: `(?P<name>cat)|🙂`, Regex: true}, {Query: `(?m)^|$`, Regex: true},
	} {
		source := "🙂 CAT cat category _cat cat_ 猫\né"
		want, err := FindString(source, opts)
		if err != nil {
			t.Fatal(err)
		}
		got, err := FindStringBounded(source, opts, math.MaxInt)
		if err != nil || len(got) != len(want) {
			t.Fatal(opts, len(got), len(want), err)
		}
		for i, hit := range got {
			if hit.Start != want[i].Start || hit.End != want[i].End || hit.Value != want[i].Value {
				t.Fatal("UTF16/field semantics differ", got, want)
			}
		}
	}
	for _, limit := range []int{-1, 0} {
		if got, err := FindStringBounded("x", Options{Query: "x"}, limit); got != nil || !errors.Is(err, ErrTooManyMatches) {
			t.Fatal("nonpositive budget", got, err)
		}
	}
	for _, count := range []int{2, 3, 4} {
		hits, err := FindStringBounded(strings.Repeat("x", count), Options{Query: "x"}, 3)
		if len(hits) != min(count, 3) || errors.Is(err, ErrTooManyMatches) != (count == 4) {
			t.Fatal("match boundary", count, len(hits), err)
		}
	}
	hits, err := FindStringBounded("xxxx cat", Options{Query: "x|cat", Regex: true, WholeWord: true}, 2)
	if len(hits) != 0 || !errors.Is(err, ErrTooManyMatches) {
		t.Fatal("filtered candidates must disclose partial scan", hits, err)
	}
	if _, err := FindStringBounded("x", Options{Query: "(", Regex: true}, 1); err == nil {
		t.Fatal("invalid regex")
	}
	if _, err := FindStringBounded("\xff", Options{Query: "x"}, 1); !errors.Is(err, text.ErrUTF8) {
		t.Fatal("invalid source", err)
	}
}

func TestFieldPlanExpansionRegexpOracle(t *testing.T) {
	for _, tc := range []struct{ source, query string }{
		{"🙂 a=12 b=34", `(?P<key>\w)=(\d+)`},
		{"ab b", `(?P<dup>a)?(?P<dup>b)`},
		{"a b", `(?P<01>a)|(?P<9999999999>b)`},
		{"🙂\na", `(?m)^|$`},
		{"ab", `(a)(b)?(c)?`},
	} {
		opts := Options{Query: tc.query, Regex: true, CaseSensitive: true}
		re := regexp.MustCompile(tc.query)
		indices := re.FindAllStringSubmatchIndex(tc.source, -1)
		hits, err := FindStringBounded(tc.source, opts, 50)
		if err != nil || len(hits) != len(indices) {
			t.Fatal(hits, indices, err)
		}
		for _, template := range []string{
			"", "literal🙂", "$0", "$1", "${1}", "${2}:$1", "$$$1", "$key/$dup", "$missing",
			"$01/${01}", "$9999999999", "${9999999999}", "$3", "$$", "$", "${}", "${1", "$-a",
			"$1text", "${key}🙂$key", "$é", "$１２", "${1}}$0", "$$$", "$0000000000001",
		} {
			for i, hit := range hits {
				want := string(re.ExpandString(nil, template, tc.source, indices[i]))
				got, err := ExpandReplacement(tc.source, hit, opts, template, len(want))
				if err != nil || got != want {
					t.Fatalf("%q %q hit=%d got=%q want=%q err=%v", tc.query, template, i, got, want, err)
				}
				if len(want) > 0 {
					if _, err := ExpandReplacement(tc.source, hit, opts, template, len(want)-1); !errors.Is(err, ErrReplacementLimit) {
						t.Fatal("budget-1 did not refuse", template, err)
					}
				}
				if got, err := ExpandReplacement(tc.source, hit, opts, template, len(want)+1); err != nil || got != want {
					t.Fatal("budget+1", got, want, err)
				}
			}
		}
	}
	// An exhaustive short grammar oracle also covers adjacent/malformed refs.
	opts := Options{Query: `(?P<a>x)(y)?`, Regex: true}
	hits, _ := FindStringBounded("x", opts, 1)
	re := regexp.MustCompile("(?i)" + opts.Query)
	alphabet := []byte{'$', '{', '}', '0', '1', 'a', '_'}
	var visit func(string, int)
	visit = func(template string, depth int) {
		want := string(re.ExpandString(nil, template, "x", hits[0].indices))
		got, err := ExpandReplacement("x", hits[0], opts, template, len(want))
		if err != nil || got != want {
			t.Fatalf("oracle %q got=%q want=%q err=%v", template, got, want, err)
		}
		if depth != 0 {
			for _, ch := range alphabet {
				visit(template+string(ch), depth-1)
			}
		}
	}
	visit("", 4)
}

func TestReplacementTemplateSharedWorkAccounting(t *testing.T) {
	opts := Options{Query: `(?P<name>a)(?P<name>b)?`, Regex: true, CaseSensitive: true}
	source := "ab"
	hits, err := FindStringBounded(source, opts, 1)
	if err != nil || len(hits) != 1 {
		t.Fatal(hits, err)
	}
	re := regexp.MustCompile(opts.Query)
	want := re.ExpandString(nil, `${name}-$1-$$-$missing`, source, hits[0].indices)
	work := 100000
	beforeCompile := work
	template, err := CompileReplacementTemplate(opts, `${name}-$1-$$-$missing`, &work)
	if err != nil || work >= beforeCompile {
		t.Fatal("template compilation did not charge work", err, work, beforeCompile)
	}
	beforeBind := work
	bound, err := template.Bind(source, &work)
	if err != nil || work >= beforeBind {
		t.Fatal("template binding did not charge work", err, work, beforeBind)
	}
	beforeExpand := work
	expansion, err := bound.Expand(hits[0], len(want), &work)
	if err != nil || expansion.Text != string(want) || expansion.StartByte != 0 || expansion.EndByte != len(source) || work >= beforeExpand {
		t.Fatal("template expansion disagrees with regexp oracle", expansion, string(want), err, work, beforeExpand)
	}
	remaining := 1
	expansion, err = bound.Expand(hits[0], len(want), &remaining)
	if expansion.Text != "" || !errors.Is(err, ErrReplacementWork) || remaining < 0 {
		t.Fatal("exhausted shared work returned partial output", expansion, err, remaining)
	}
}

func TestReplacementTemplateChargesZeroOutputNameWork(t *testing.T) {
	query := strings.Repeat(`(?P<late>y)|`, 15) + `(?P<late>x)`
	opts := Options{Query: query, Regex: true, CaseSensitive: true}
	source := strings.Repeat("x", 16)
	hits, err := FindStringBounded(source, opts, 16)
	if err != nil || len(hits) != 16 {
		t.Fatal(len(hits), err)
	}
	replacement := strings.Repeat("$missing", 64)
	work := 100000
	template, err := CompileReplacementTemplate(opts, replacement, &work)
	if err != nil {
		t.Fatal(err)
	}
	bound, err := template.Bind(source, &work)
	if err != nil {
		t.Fatal(err)
	}
	remaining := 8
	expansion, err := bound.Expand(hits[0], 0, &remaining)
	if expansion.Text != "" || !errors.Is(err, ErrReplacementWork) || remaining < 0 {
		t.Fatal("zero-output expansion bypassed work budget", expansion, err, remaining)
	}
}

// The oracle uses only two bytes/two hits, and counts the documented primitive
// charges independently: compile=1+2*17+5 + 2 outer bytes + 1 brace + 2
// name-rune probes + 1 numeric probe + 1 cached-name lookup byte + (2+3+3)
// candidate comparisons = 55. Bind=1+4*2=9. Per hit validation=1+17+6+1,
// ref=1, duplicate probes=2/1, part passes=2+2, output=2 => 34/33.
// Descending output stitches cost 1+3 and 1+4. Total = 140, not output bytes.
func TestReplacementTemplateExactSharedLedgerOracle(t *testing.T) {
	opts := Options{Query: `(?P<n>x)|(?P<n>y)`, Regex: true, CaseSensitive: true}
	hits, err := FindStringBounded("xy", opts, 2)
	if err != nil || len(hits) != 2 {
		t.Fatal(hits, err)
	}
	const total = 55 + 9 + 34 + 4 + 33 + 5
	for budget := 0; budget <= total; budget++ {
		work := budget
		compileBefore := work
		template, err := CompileReplacementTemplate(opts, "${n}!", &work)
		var bound *BoundReplacement
		next := "xy"
		if err == nil {
			if compileBefore-work != 55 {
				t.Fatal("compile/name-parse/candidate oracle", compileBefore-work)
			}
			before := work
			bound, err = template.Bind(next, &work)
			if err == nil && before-work != 9 {
				t.Fatal("bind oracle", before-work)
			}
		}
		for i := 1; i >= 0 && err == nil; i-- {
			before := work
			var expansion FieldExpansion
			expansion, err = bound.Expand(hits[i], 4, &work)
			if err != nil {
				if expansion != (FieldExpansion{}) {
					t.Fatal("partial expansion", expansion)
				}
				break
			}
			if want := 33 + i; before-work != want {
				t.Fatal("hit/candidate/output oracle", i, before-work, want)
			}
			before = work
			next, err = StitchReplacement(next, expansion, 4, &work)
			if err != nil && next != "" {
				t.Fatal("partial stitch", next)
			}
			if err == nil && before-work != 5-i {
				t.Fatal("stitch oracle", i, before-work)
			}
		}
		if budget == total {
			if err != nil || next != "x!y!" || work != 0 {
				t.Fatal("exact shared ceiling must succeed", next, work, err)
			}
		} else if !errors.Is(err, ErrReplacementWork) || work != 0 {
			t.Fatal("every smaller common ledger must exhaust", budget, work, err)
		}
	}
}

func TestReplacementTemplateFiniteDefaultsAndCachedDuplicateNames(t *testing.T) {
	// Verify finite nil defaults without constructing an oversized source.
	w := replacementWork(nil)
	if *w != DefaultReplacementWork || *w <= 0 || *w >= math.MaxInt || w == replacementWork(nil) {
		t.Fatal("nil ledger is not a fresh finite default", *w)
	}
	if err := spendReplacementWork(w, *w+1); !errors.Is(err, ErrReplacementWork) || *w != 0 {
		t.Fatal("default budget cannot reject", *w, err)
	}
	opts := Options{Query: `(?P<n>x)|(?P<n>y)`, Regex: true, CaseSensitive: true}
	hits, _ := FindStringBounded("y", opts, 1)
	template, err := CompileReplacementTemplate(opts, "${n}${n}", nil)
	if err != nil || len(template.references) != 1 {
		t.Fatal("duplicate reference compiled more than once", template, err)
	}
	bound, err := template.Bind("y", nil)
	if err != nil {
		t.Fatal(err)
	}
	expansion, err := bound.Expand(hits[0], 2, nil)
	if err != nil || expansion.Text != "yy" {
		t.Fatal("first participating duplicate name", expansion, err)
	}
	if stitched, err := StitchReplacement("y", expansion, 2, nil); err != nil || stitched != "yy" {
		t.Fatal("default stitch", stitched, err)
	}
}

func TestFieldPlanExpansionRejectsInvalidIdentityAndBudget(t *testing.T) {
	opts := Options{Query: "🙂"}
	hits, _ := FindStringBounded("x🙂", opts, 1)
	hit := hits[0]
	for _, tc := range []struct {
		name, source, replacement string
		opts                      Options
		hit                       Match
		budget                    int
		want                      error
	}{
		{"stale-source", "y🙂", "x", opts, hit, 1, ErrStale},
		{"stale-options", "x🙂", "x", Options{Query: "🙂", CaseSensitive: true}, hit, 1, ErrStale},
		{"zero-match", "x🙂", "x", opts, Match{}, 1, ErrStale},
		{"negative", "x🙂", "", opts, hit, -1, ErrReplacementLimit},
		{"invalid-output", "x🙂", "\xff", opts, hit, 1, text.ErrUTF8},
		{"literal-dollar", "x🙂", "$1", opts, hit, 1, ErrReplacementLimit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := ExpandReplacement(tc.source, tc.hit, tc.opts, tc.replacement, tc.budget); got != "" || !errors.Is(err, tc.want) {
				t.Fatal(got, err)
			}
		})
	}
	for _, mutate := range []func(*Match){
		func(m *Match) { m.Start++ }, func(m *Match) { m.End-- }, func(m *Match) { m.Value = "bad" },
		func(m *Match) { m.indices = nil }, func(m *Match) { m.indices = []int{2, 5} },
		func(m *Match) { m.indices = []int{-1, -1} }, func(m *Match) { m.indices = []int{0, 99} },
	} {
		copy := hit
		mutate(&copy)
		if _, err := ExpandReplacement("x🙂", copy, opts, "x", 1); !errors.Is(err, ErrStale) {
			t.Fatal("tampered match accepted", copy, err)
		}
	}
	if got, err := ExpandReplacement("x🙂", hit, opts, "$1", 2); err != nil || got != "$1" {
		t.Fatal("literal must keep capture syntax", got, err)
	}
	large := strings.Repeat("a", 64<<10)
	largeOpts := Options{Query: "(a+)", Regex: true}
	largeHits, _ := FindStringBounded(large, largeOpts, 1)
	if got, err := ExpandReplacement(large, largeHits[0], largeOpts, strings.Repeat("$1", 4096), 64<<10); got != "" || !errors.Is(err, ErrReplacementLimit) {
		t.Fatal("large expansion allocated/accepted", len(got), err)
	}
}
