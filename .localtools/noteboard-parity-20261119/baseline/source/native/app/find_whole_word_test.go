package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestFindWholeWordCheckboxNavigationAndWrap(t *testing.T) {
	a, tab := newSearchTextApp(t, "cat category _cat cat_ cat")
	a.Query = "cat"
	a.ShowFind = true
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()

	if err := tester.Click("全词"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if !a.WholeWord {
		t.Fatal("whole-word checkbox did not update runtime state")
	}

	setSearchSelection(t, tab, 0, 0)
	if err := tester.Click("下一个"); err != nil {
		t.Fatal(err)
	}
	assertSearchSelection(t, tab, 0, 3)
	if err := tester.Click("下一个"); err != nil {
		t.Fatal(err)
	}
	assertSearchSelection(t, tab, 23, 26)
	if err := tester.Click("下一个"); err != nil {
		t.Fatal(err)
	}
	assertSearchSelection(t, tab, 0, 3)
	if err := tester.Click("上一个"); err != nil {
		t.Fatal(err)
	}
	assertSearchSelection(t, tab, 23, 26)
}

func TestFindWholeWordUnicodeRegexAndCase(t *testing.T) {
	cases := []struct {
		name                 string
		source, query        string
		regex, caseSensitive bool
		wantStart, wantEnd   []int
	}{
		{
			name: "identifiers", source: "cat category _cat cat_ cat", query: "cat",
			wantStart: []int{0, 23}, wantEnd: []int{3, 26},
		},
		{
			name: "cjk", source: "猫 猫咪", query: "猫",
			wantStart: []int{0}, wantEnd: []int{1},
		},
		{
			name: "combining-mark", source: "e e\u0301", query: "e",
			wantStart: []int{0}, wantEnd: []int{1},
		},
		{
			name: "regex-case-insensitive", source: "CAT cat category cat_ 猫 猫咪", query: "cat|猫",
			regex: true, wantStart: []int{0, 4, 22}, wantEnd: []int{3, 7, 23},
		},
		{
			name: "regex-case-sensitive", source: "CAT cat category cat_ 猫 猫咪", query: "cat|猫",
			regex: true, caseSensitive: true, wantStart: []int{4, 22}, wantEnd: []int{7, 23},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, tab := newSearchTextApp(t, tc.source)
			a.Query, a.Regex, a.CaseSensitive, a.WholeWord = tc.query, tc.regex, tc.caseSensitive, true
			a.find(true)
			matches := a.findNavigation.matches
			if len(matches) != len(tc.wantStart) {
				t.Fatalf("whole-word matches = %#v, want %d", matches, len(tc.wantStart))
			}
			for i := range matches {
				if matches[i].Start != tc.wantStart[i] || matches[i].End != tc.wantEnd[i] {
					t.Fatalf("match %d = %#v, want [%d,%d)", i, matches[i], tc.wantStart[i], tc.wantEnd[i])
				}
			}
			if len(matches) > 0 {
				assertSearchSelection(t, tab, matches[0].Start, matches[0].End)
			}
		})
	}
}

func TestFindWholeWordReplacementAndGroupedUndo(t *testing.T) {
	t.Run("current eligible", func(t *testing.T) {
		a, tab := newSearchTextApp(t, "cat category cat")
		a.Query, a.Replacement, a.WholeWord = "cat", "dog", true
		setSearchSelection(t, tab, 0, 3)
		a.replace(false)
		if got := tab.Doc.Buffer.String(); got != "dog category cat" {
			t.Fatalf("Replace Current with whole word = %q", got)
		}
	})

	t.Run("current without whole word replaces inside-word selection", func(t *testing.T) {
		a, tab := newSearchTextApp(t, "category")
		a.Query, a.Replacement, a.WholeWord = "cat", "dog", false
		setSearchSelection(t, tab, 0, 3)
		selected, err := tab.Doc.Buffer.SliceUTF16(0, 3)
		if err != nil || selected != a.Query {
			t.Fatalf("inside-word selection = %q, %v; want query %q at [0,3)", selected, err, a.Query)
		}
		a.replace(false)
		if got := tab.Doc.Buffer.String(); got != "dogegory" {
			t.Fatalf("Replace Current without whole word = %q, want replaced inside-word selection", got)
		}
	})

	t.Run("current with whole word rejects inside-word selection", func(t *testing.T) {
		a, tab := newSearchTextApp(t, "category")
		a.Query, a.Replacement, a.WholeWord = "cat", "dog", true
		setSearchSelection(t, tab, 0, 3)
		selected, err := tab.Doc.Buffer.SliceUTF16(0, 3)
		if err != nil || selected != a.Query {
			t.Fatalf("inside-word selection = %q, %v; want query %q at [0,3)", selected, err, a.Query)
		}
		before := tab.Doc.Buffer.String()
		version := tab.Doc.Buffer.Version()
		selection := tab.Doc.Buffer.Selection()
		a.replace(false)
		if tab.Doc.Buffer.String() != before || tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.Selection() != selection {
			t.Fatalf("Replace Current accepted inside-word selection: before=%q after=%q version=%d/%d selection=%v/%v",
				before, tab.Doc.Buffer.String(), version, tab.Doc.Buffer.Version(), selection, tab.Doc.Buffer.Selection())
		}
	})

	t.Run("all undo group", func(t *testing.T) {
		const source = "cat category _cat cat_ cat"
		a, tab := newSearchTextApp(t, source)
		a.Query, a.Replacement, a.WholeWord = "cat", "dog", true
		a.replace(true)
		if got := tab.Doc.Buffer.String(); got != "dog category _cat cat_ dog" {
			t.Fatalf("Replace All with whole word = %q", got)
		}
		if !tab.Doc.Buffer.Undo() {
			t.Fatal("Replace All whole-word change was not undoable")
		}
		if got := tab.Doc.Buffer.String(); got != source {
			t.Fatalf("one Undo did not restore whole Replace All group: %q", got)
		}
	})
}

func TestFindWholeWordLifecycleGuardsPreserveBuffer(t *testing.T) {
	t.Run("readonly", func(t *testing.T) {
		a, tab := newSearchTextApp(t, "cat category")
		tab.Doc.ReadOnly = true
		tab.Editor.ReadOnly = true
		a.Query, a.Replacement, a.WholeWord = "cat", "dog", true
		before := tab.Doc.Buffer.String()
		version := tab.Doc.Buffer.Version()
		selection := tab.Doc.Buffer.Selection()
		a.replace(true)
		if tab.Doc.Buffer.String() != before || tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.Selection() != selection {
			t.Fatalf("readonly whole-word replace changed source: before=%q after=%q version=%d/%d selection=%v/%v",
				before, tab.Doc.Buffer.String(), version, tab.Doc.Buffer.Version(), selection, tab.Doc.Buffer.Selection())
		}
	})

	t.Run("failed flush", func(t *testing.T) {
		a, tab := newSearchTextApp(t, "cat category")
		tab.View = &failingView{flushErr: errors.New("draft conflict")}
		tab.Mode = "preview"
		a.Query, a.Replacement, a.WholeWord = "cat", "dog", true
		setSearchSelection(t, tab, 0, 3)
		before := tab.Doc.Buffer.String()
		version := tab.Doc.Buffer.Version()
		selection := tab.Doc.Buffer.Selection()
		a.replace(true)
		if tab.Doc.Buffer.String() != before || tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.Selection() != selection {
			t.Fatalf("failed Flush whole-word replace changed source: before=%q after=%q version=%d/%d selection=%v/%v",
				before, tab.Doc.Buffer.String(), version, tab.Doc.Buffer.Version(), selection, tab.Doc.Buffer.Selection())
		}
	})
}

func TestFindWholeWordReachesWorkspaceSearch(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "boundaries.txt")
	if err := os.WriteFile(path, []byte("cat category _cat cat_"), 0600); err != nil {
		t.Fatal(err)
	}

	a := newTestApp(t, filepath.Join(root, "config"))
	queue := make(chan func(), 16)
	a.uiDispatch = func(fn func()) { queue <- fn }
	a.Root = root
	a.Query = "cat"
	a.WholeWord = true
	a.searchWorkspace()

	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	for a.Searching {
		select {
		case fn := <-queue:
			fn()
		case <-timeout.C:
			t.Fatal("workspace whole-word search did not finish")
		}
	}
	if len(a.SearchResults) != 1 {
		t.Fatalf("workspace whole-word results = %#v, want only the standalone cat", a.SearchResults)
	}
	for _, result := range a.SearchResults {
		if filepath.Clean(result.Path) != filepath.Clean(path) {
			t.Fatalf("workspace result path = %q, want %q", result.Path, path)
		}
	}
}
