package app

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/search"
	"levnote.local/native/core/text"
)

func newSearchTextApp(t *testing.T, source string) (*App, *Tab) {
	t.Helper()
	a := newTestApp(t, t.TempDir())
	a.NewDocument("text")
	tab := a.Current()
	if err := tab.Doc.Buffer.ReplaceUTF16(0, tab.Doc.Buffer.LenUTF16(), source); err != nil {
		t.Fatal(err)
	}
	return a, tab
}

func setSearchSelection(t *testing.T, tab *Tab, anchor, head int) {
	t.Helper()
	if err := tab.Doc.Buffer.SetSelection(anchor, head); err != nil {
		t.Fatal(err)
	}
}

func assertSearchSelection(t *testing.T, tab *Tab, start, end int) {
	t.Helper()
	selection := tab.Doc.Buffer.Selection()
	if selection.Start() != start || selection.End() != end {
		t.Fatalf("search selection = %#v, want ordered bounds [%d,%d)", selection, start, end)
	}
}

func TestFindAdjacentMatchesProgress(t *testing.T) {
	a, tab := newSearchTextApp(t, "aaaa")
	a.Query = "aa"
	a.CaseSensitive = true

	setSearchSelection(t, tab, 0, 0)
	a.find(true)
	assertSearchSelection(t, tab, 0, 2)
	a.find(true)
	assertSearchSelection(t, tab, 2, 4)
	a.find(false)
	assertSearchSelection(t, tab, 0, 2)
}

func TestFindSingleResultWrapsToItself(t *testing.T) {
	a, tab := newSearchTextApp(t, "needle")
	a.Query = "needle"
	setSearchSelection(t, tab, 0, 0)
	a.find(true)
	assertSearchSelection(t, tab, 0, 6)
	a.find(true)
	assertSearchSelection(t, tab, 0, 6)
	a.find(false)
	assertSearchSelection(t, tab, 0, 6)
	if a.Message != "匹配 1 / 1" {
		t.Fatalf("single-result message = %q", a.Message)
	}
}

func TestFindNearestMatchUsesOrderedSelectionBounds(t *testing.T) {
	const source = "aa xx aa yy aa"
	cases := []struct {
		name         string
		anchor, head int
		next         bool
		start, end   int
	}{
		{name: "between previous", anchor: 3, head: 5, next: false, start: 0, end: 2},
		{name: "between next", anchor: 3, head: 5, next: true, start: 6, end: 8},
		{name: "caret at match start includes match forward", anchor: 6, head: 6, next: true, start: 6, end: 8},
		{name: "caret at match start uses previous when going backward", anchor: 6, head: 6, next: false, start: 0, end: 2},
		{name: "caret at match end includes match backward", anchor: 2, head: 2, next: false, start: 0, end: 2},
		{name: "caret at match end advances forward", anchor: 2, head: 2, next: true, start: 6, end: 8},
		{name: "reversed selection ignores head for forward", anchor: 8, head: 0, next: true, start: 12, end: 14},
		{name: "reversed selection ignores head for previous", anchor: 8, head: 0, next: false, start: 12, end: 14},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, tab := newSearchTextApp(t, source)
			a.Query = "aa"
			a.CaseSensitive = true
			setSearchSelection(t, tab, tc.anchor, tc.head)
			a.find(tc.next)
			assertSearchSelection(t, tab, tc.start, tc.end)
		})
	}
}

func TestFindExactSelectionAdvancesInEitherDirectionAndWraps(t *testing.T) {
	const source = "aa xx aa yy aa"
	cases := []struct {
		name         string
		anchor, head int
		next         bool
		start, end   int
	}{
		{name: "normal next", anchor: 6, head: 8, next: true, start: 12, end: 14},
		{name: "reversed next", anchor: 8, head: 6, next: true, start: 12, end: 14},
		{name: "normal previous", anchor: 6, head: 8, next: false, start: 0, end: 2},
		{name: "reversed previous", anchor: 8, head: 6, next: false, start: 0, end: 2},
		{name: "next wraps", anchor: 12, head: 14, next: true, start: 0, end: 2},
		{name: "previous wraps", anchor: 0, head: 2, next: false, start: 12, end: 14},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, tab := newSearchTextApp(t, source)
			a.Query = "aa"
			a.CaseSensitive = true
			setSearchSelection(t, tab, tc.anchor, tc.head)
			a.find(tc.next)
			assertSearchSelection(t, tab, tc.start, tc.end)
		})
	}
}

func TestFindZeroWidthAndUTF16OffsetsProgress(t *testing.T) {
	a, tab := newSearchTextApp(t, "🙂\n猫\n🙂")
	a.Query = "(?m)^"
	a.Regex = true
	a.CaseSensitive = true

	setSearchSelection(t, tab, 0, 0)
	a.find(true)
	assertSearchSelection(t, tab, 0, 0)
	a.find(true)
	assertSearchSelection(t, tab, 3, 3)
	a.find(false)
	assertSearchSelection(t, tab, 0, 0)

	setSearchSelection(t, tab, 7, 7)
	a.find(false)
	assertSearchSelection(t, tab, 5, 5)
	a.find(false)
	assertSearchSelection(t, tab, 3, 3)

	a.Query = "猫"
	a.Regex = false
	a.find(true)
	assertSearchSelection(t, tab, 3, 4)

	a.Query = "🙂"
	a.find(true)
	assertSearchSelection(t, tab, 5, 7)
}

func TestFindCacheReusesColdOffsetsAndTracksSourceVersion(t *testing.T) {
	oldFind := searchNavigationFind
	t.Cleanup(func() { searchNavigationFind = oldFind })
	cold := 0
	searchNavigationFind = func(snapshot text.Snapshot, options search.Options) ([]search.Match, error) {
		cold++
		return search.Find(snapshot, options)
	}

	a, tab := newSearchTextApp(t, "needle needless needle")
	a.Query = "needle"
	a.CaseSensitive = true
	setSearchSelection(t, tab, 0, 0)
	a.find(true)
	a.find(true)
	if cold != 1 {
		t.Fatalf("unchanged source cold scans = %d, want 1", cold)
	}

	a.CaseSensitive = false
	a.find(true)
	a.Regex = true
	a.find(true)
	if want := []findMatchOffset{{Start: 0, End: 6}, {Start: 7, End: 13}, {Start: 16, End: 22}}; !reflect.DeepEqual(a.findNavigation.matches, want) {
		t.Fatalf("pre-whole-word cached offsets = %#v, want %#v", a.findNavigation.matches, want)
	}
	a.WholeWord = true
	a.find(true)
	if cold != 4 {
		t.Fatalf("query/options cold scans = %d, want 4", cold)
	}
	if want := []findMatchOffset{{Start: 0, End: 6}, {Start: 16, End: 22}}; !reflect.DeepEqual(a.findNavigation.matches, want) {
		t.Fatalf("filtered whole-word cached offsets = %#v, want %#v", a.findNavigation.matches, want)
	}
	a.find(true)
	if cold != 4 {
		t.Fatalf("filtered whole-word cache was not reused: cold scans = %d, want 4", cold)
	}

	if err := tab.Doc.Buffer.ReplaceUTF16(0, 0, "needle "); err != nil {
		t.Fatal(err)
	}
	a.find(true)
	if cold != 5 {
		t.Fatalf("edit cold scans = %d, want 5", cold)
	}
	if !tab.Doc.Buffer.Undo() {
		t.Fatal("edit could not be undone")
	}
	a.find(true)
	if cold != 6 {
		t.Fatalf("undo cold scans = %d, want 6", cold)
	}

	snapshot := tab.Doc.Buffer.Snapshot()
	if err := tab.Doc.Buffer.ReplaceUTF16(0, 0, "changed "); err != nil {
		t.Fatal(err)
	}
	a.find(true)
	tab.Doc.Buffer.Restore(snapshot)
	a.find(true)
	if cold != 8 {
		t.Fatalf("restore cold scans = %d, want 8", cold)
	}

	a.NewDocument("text")
	if err := a.Current().Doc.Buffer.ReplaceUTF16(0, 0, "needle"); err != nil {
		t.Fatal(err)
	}
	a.find(true)
	if cold != 9 {
		t.Fatalf("new-buffer cold scans = %d, want 9", cold)
	}
	a.Query = ""
	a.find(true)
	if cold != 9 || a.findNavigation.tab != nil || a.FindIndex != -1 {
		t.Fatalf("empty query retained cache: scans=%d cache=%#v FindIndex=%d", cold, a.findNavigation, a.FindIndex)
	}
}

func TestFindCacheRefusesPartialLimitResultAndKeepsOffsetsOnly(t *testing.T) {
	offsetType := reflect.TypeOf(findMatchOffset{})
	if offsetType.NumField() != 2 || offsetType.Field(0).Name != "Start" || offsetType.Field(0).Type.Kind() != reflect.Int ||
		offsetType.Field(1).Name != "End" || offsetType.Field(1).Type.Kind() != reflect.Int {
		t.Fatalf("retained match entry widened beyond Start/End ints: %v", offsetType)
	}
	cacheType := reflect.TypeOf(findNavigationCache{})
	matches, ok := cacheType.FieldByName("matches")
	if !ok || matches.Type != reflect.TypeOf([]findMatchOffset(nil)) {
		t.Fatalf("cache matches field = %v, want []findMatchOffset", matches.Type)
	}
	for _, forbidden := range []reflect.Type{
		reflect.TypeOf(search.Match{}),
		reflect.TypeOf(text.Snapshot{}),
		reflect.TypeOf([]search.Match(nil)),
	} {
		for i := 0; i < cacheType.NumField(); i++ {
			if cacheType.Field(i).Type == forbidden {
				t.Fatalf("cache retains forbidden %v in %s", forbidden, cacheType.Field(i).Name)
			}
		}
	}

	oldFind := searchNavigationFind
	t.Cleanup(func() { searchNavigationFind = oldFind })
	cold := 0
	searchNavigationFind = func(text.Snapshot, search.Options) ([]search.Match, error) {
		cold++
		return []search.Match{{Start: 0, End: 1, Value: "too many"}}, search.ErrTooManyMatches
	}
	a, tab := newSearchTextApp(t, "source")
	a.Query = "source"
	setSearchSelection(t, tab, 1, 1)
	before := tab.Doc.Buffer.Selection()
	a.find(true)
	a.find(true)
	assertSearchSelection(t, tab, 1, 1)
	if cold != 1 || a.FindIndex != -1 || tab.Doc.Buffer.Selection() != before ||
		!errors.Is(a.findNavigation.err, search.ErrTooManyMatches) || len(a.findNavigation.matches) != 0 {
		t.Fatalf("partial limit result: cold=%d FindIndex=%d cache=%#v message=%q", cold, a.FindIndex, a.findNavigation, a.Message)
	}
}

func TestFindCacheNoResultAndSearchErrorClearCursor(t *testing.T) {
	oldFind := searchNavigationFind
	t.Cleanup(func() { searchNavigationFind = oldFind })
	cold := 0
	searchNavigationFind = func(snapshot text.Snapshot, options search.Options) ([]search.Match, error) {
		cold++
		return search.Find(snapshot, options)
	}
	a, tab := newSearchTextApp(t, "alpha needle")
	a.Query = "needle"
	setSearchSelection(t, tab, 0, 0)
	a.find(true)
	if a.FindIndex != 0 || a.findNavigation.lastIndex != 0 {
		t.Fatalf("initial cursor = FindIndex %d cache %d", a.FindIndex, a.findNavigation.lastIndex)
	}

	a.Query = "absent"
	a.find(true)
	a.find(true)
	assertSearchSelection(t, tab, 6, 12)
	if cold != 2 || a.FindIndex != -1 || a.findNavigation.lastIndex != -1 || a.Message != "没有匹配结果" {
		t.Fatalf("no result retained cursor: cold=%d FindIndex=%d cache=%d message=%q", cold, a.FindIndex, a.findNavigation.lastIndex, a.Message)
	}

	a.Query = "["
	a.Regex = true
	a.find(true)
	a.find(true)
	assertSearchSelection(t, tab, 6, 12)
	if cold != 3 || a.FindIndex != -1 || a.findNavigation.lastIndex != -1 || a.findNavigation.err == nil {
		t.Fatalf("error retained cursor: cold=%d FindIndex=%d cache=%#v", cold, a.FindIndex, a.findNavigation)
	}
}

func TestFindAbandonsResultsWhenActiveTabChangesDuringScan(t *testing.T) {
	a, first := newSearchTextApp(t, "needle")
	a.NewDocument("text")
	second := a.Current()
	if err := second.Doc.Buffer.ReplaceUTF16(0, 0, "needle"); err != nil {
		t.Fatal(err)
	}
	if !a.Select(0) {
		t.Fatal("could not return to first tab")
	}

	oldFind := searchNavigationFind
	t.Cleanup(func() { searchNavigationFind = oldFind })
	searchNavigationFind = func(snapshot text.Snapshot, options search.Options) ([]search.Match, error) {
		if !a.Select(1) {
			t.Fatal("could not switch tabs from search seam")
		}
		return search.Find(snapshot, options)
	}
	a.Query = "needle"
	setSearchSelection(t, first, 0, 0)
	secondSelection := second.Doc.Buffer.Selection()
	a.find(true)
	if a.Current() != second || a.findNavigation.tab != nil || a.FindIndex != -1 {
		t.Fatalf("scan result acted after tab change: current=%p cache=%p FindIndex=%d", a.Current(), a.findNavigation.tab, a.FindIndex)
	}
	if got := second.Doc.Buffer.Selection(); got != secondSelection {
		t.Fatalf("scan result changed switched tab selection: got %#v, want %#v", got, secondSelection)
	}
}

func TestFindWarmCacheStillFlushesAndFailedFlushPreservesState(t *testing.T) {
	a, tab := newSearchTextApp(t, "needle needle")
	a.Query = "needle"
	setSearchSelection(t, tab, 0, 0)
	a.find(true)
	if a.findNavigation.tab != tab {
		t.Fatal("search cache was not established")
	}
	beforeSelection := tab.Doc.Buffer.Selection()
	beforeVersion := tab.Doc.Buffer.Version()
	beforeHistory := tab.Doc.Buffer.History()
	view := &failingView{flushErr: errors.New("draft conflict")}
	tab.View = view
	tab.Mode = "preview"
	a.find(true)
	if view.flushes != 1 || tab.View != view || tab.Mode != "preview" {
		t.Fatalf("warm find bypassed or released failed flush: flushes=%d view=%p mode=%q", view.flushes, tab.View, tab.Mode)
	}
	if tab.Doc.Buffer.Selection() != beforeSelection || tab.Doc.Buffer.Version() != beforeVersion || tab.Doc.Buffer.History() != beforeHistory {
		t.Fatalf("failed warm flush changed buffer state: selection=%#v version=%d history=%#v", tab.Doc.Buffer.Selection(), tab.Doc.Buffer.Version(), tab.Doc.Buffer.History())
	}
	if a.findNavigation.tab != tab {
		t.Fatal("failed flush discarded the still-valid warm cache")
	}
}

type searchMutatingView struct {
	buffer  *text.Buffer
	source  string
	flushes int
}

func (*searchMutatingView) Draw(*ui.Context)              {}
func (*searchMutatingView) Release()                      {}
func (*searchMutatingView) Warnings() []string            { return nil }
func (*searchMutatingView) Export(string) ([]byte, error) { return nil, nil }
func (v *searchMutatingView) Flush() error {
	v.flushes++
	return v.buffer.ReplaceUTF16(0, v.buffer.LenUTF16(), v.source)
}

func TestFindUsesVersionAfterSourceChangingRichFlush(t *testing.T) {
	a, tab := newSearchTextApp(t, "stale")
	view := &searchMutatingView{buffer: tab.Doc.Buffer, source: "a b"}
	tab.View = view
	tab.Mode = "preview"
	a.Query = "b"
	a.find(true)
	assertSearchSelection(t, tab, 2, 3)
	version := tab.Doc.Buffer.Version()
	if a.findNavigation.version != version {
		t.Fatalf("cache version = %d, want post-flush %d", a.findNavigation.version, version)
	}

	view.source = "a c b"
	a.find(true)
	assertSearchSelection(t, tab, 4, 5)
	if view.flushes != 2 || a.findNavigation.version != tab.Doc.Buffer.Version() {
		t.Fatalf("second flush/cache version: flushes=%d cache=%d live=%d", view.flushes, a.findNavigation.version, tab.Doc.Buffer.Version())
	}
}

type searchIdentityChangingView struct {
	tab     *Tab
	nextDoc *document.Document
	flushes int
}

func (*searchIdentityChangingView) Draw(*ui.Context)              {}
func (*searchIdentityChangingView) Release()                      {}
func (*searchIdentityChangingView) Warnings() []string            { return nil }
func (*searchIdentityChangingView) Export(string) ([]byte, error) { return nil, nil }
func (v *searchIdentityChangingView) Flush() error {
	v.flushes++
	v.tab.Doc = v.nextDoc
	v.tab.Editor.Buffer = v.nextDoc.Buffer
	return nil
}

func TestFindAcceptsNoResultAfterFlushChangesBufferIdentity(t *testing.T) {
	a, tab := newSearchTextApp(t, "needle")
	oldBuffer := tab.Doc.Buffer
	nextDoc := document.New("replacement")
	view := &searchIdentityChangingView{tab: tab, nextDoc: nextDoc}
	tab.View = view
	tab.Mode = "preview"
	a.Query = "needle"
	setSearchSelection(t, tab, 1, 1)

	a.find(true)
	if view.flushes != 1 || a.findNavigation.tab != nil || a.FindIndex != -1 {
		t.Fatalf("identity-changing flush acted on stale result: flushes=%d cache=%#v FindIndex=%d", view.flushes, a.findNavigation, a.FindIndex)
	}
	if oldBuffer.Selection() != (text.Range{Anchor: 1, Head: 1}) || tab.Doc.Buffer != nextDoc.Buffer {
		t.Fatalf("identity-changing flush state: old=%#v current=%p", oldBuffer.Selection(), tab.Doc.Buffer)
	}
}

func TestFindCacheLifecycleClearsOnCommitNotFailedTransitions(t *testing.T) {
	a, first := newSearchTextApp(t, "needle")
	a.Query = "needle"
	setSearchSelection(t, first, 0, 0)
	a.find(true)
	a.ShowFind = true
	a.closeFind()
	if a.ShowFind || a.findNavigation.tab != nil || a.FindIndex != -1 {
		t.Fatal("closing Find did not release cache")
	}

	a.ShowFind = true
	a.find(true)
	a.NewDocument("text")
	if second := a.Current(); second == first || a.findNavigation.tab != nil {
		t.Fatalf("new current buffer did not clear cache: current=%p cache=%p", second, a.findNavigation.tab)
	}
	second := a.Current()
	if err := second.Doc.Buffer.ReplaceUTF16(0, 0, "needle"); err != nil {
		t.Fatal(err)
	}
	a.find(true)
	if a.findNavigation.tab != second {
		t.Fatal("cache did not warm on the second buffer")
	}
	if !a.Select(0) || a.findNavigation.tab != nil {
		t.Fatal("committed tab switch did not clear cache")
	}

	a.find(true)
	first.View = &failingView{flushErr: errors.New("draft conflict")}
	first.Mode = "preview"
	if a.Select(1) {
		t.Fatal("failed tab switch unexpectedly succeeded")
	}
	if a.Current() != first || a.findNavigation.tab != first || first.View == nil {
		t.Fatal("failed tab switch did not preserve active document and cache")
	}
	first.View = nil
	first.Mode = "source"

	a.closeDialog = func(*Tab) (closeDecision, error) { return closeCancel, nil }
	if a.CloseTab(0) {
		t.Fatal("cancelled close unexpectedly removed the tab")
	}
	if a.Current() != first || a.findNavigation.tab != first {
		t.Fatal("cancelled close did not preserve active document and cache")
	}
	a.closeDialog = func(*Tab) (closeDecision, error) { return closeDiscard, nil }
	if !a.CloseTab(0) || a.findNavigation.tab != nil {
		t.Fatal("committed remove did not remove the tab and clear cache")
	}
}

func TestFindCacheExpiresOnNextFrameForQueryAndVersion(t *testing.T) {
	a, tab := newSearchTextApp(t, "needle")
	a.Query = "needle"
	setSearchSelection(t, tab, 0, 0)
	a.find(true)
	if a.findNavigation.tab != tab {
		t.Fatal("cache did not warm before query-frame test")
	}
	a.Query = "other"
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	if a.findNavigation.tab != nil || a.FindIndex != -1 {
		t.Fatal("query change survived the next frame")
	}

	a.Query = "needle"
	a.find(true)
	if a.findNavigation.tab != tab {
		t.Fatal("cache did not re-warm before version-frame test")
	}
	if err := tab.Doc.Buffer.ReplaceUTF16(0, 0, "x"); err != nil {
		t.Fatal(err)
	}
	if a.findNavigation.tab != tab {
		t.Fatal("version change was expected to remain lazy until the next frame")
	}
	tester.Frame()
	if a.findNavigation.tab != nil || a.FindIndex != -1 {
		t.Fatal("buffer version change survived the next frame")
	}
}

func TestFindCacheDropsOnReloadAndPluginRematch(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "note.txt")
	if err := os.WriteFile(path, []byte("needle"), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	a.Query = "needle"
	setSearchSelection(t, tab, 0, 0)
	a.find(true)
	if a.findNavigation.tab != tab {
		t.Fatal("cache did not warm before reload")
	}
	a.Reload()
	if a.findNavigation.tab != nil || a.FindIndex != -1 {
		t.Fatal("reload retained search cache")
	}

	a.find(true)
	if a.findNavigation.tab != tab {
		t.Fatal("cache did not re-warm before rematch")
	}
	a.reconfigureTabs()
	if a.findNavigation.tab != nil || a.FindIndex != -1 {
		t.Fatal("plugin rematch retained search cache")
	}
}

func TestShutdownClearsFindCache(t *testing.T) {
	a, tab := newSearchTextAppShutdown(t)
	a.Query = "needle"
	setSearchSelection(t, tab, 0, 0)
	a.find(true)
	if a.findNavigation.tab != tab {
		t.Fatal("cache did not warm before shutdown")
	}
	a.Shutdown()
	if a.findNavigation.tab != nil || a.findNavigation.matches != nil || a.FindIndex != -1 {
		t.Fatalf("shutdown retained search cache: %#v", a.findNavigation)
	}
}

func newSearchTextAppShutdown(t *testing.T) (*App, *Tab) {
	t.Helper()
	a := New(t.TempDir())
	a.NewDocument("text")
	tab := a.Current()
	if err := tab.Doc.Buffer.ReplaceUTF16(0, 0, "needle"); err != nil {
		t.Fatal(err)
	}
	return a, tab
}

func TestReplaceCurrentInvalidatesNavigationCacheAndFindsFreshText(t *testing.T) {
	a, tab := newSearchTextApp(t, "a a")
	a.Query = "a"
	a.CaseSensitive = true
	setSearchSelection(t, tab, 0, 0)
	a.find(true)
	oldCacheMatches := a.findNavigation.matches

	a.Replacement = "b"
	a.replace(false)
	if tab.Doc.Buffer.String() != "b a" {
		t.Fatalf("replace result = %q", tab.Doc.Buffer.String())
	}
	assertSearchSelection(t, tab, 2, 3)
	if a.findNavigation.version != tab.Doc.Buffer.Version() || len(a.findNavigation.matches) != 1 ||
		&a.findNavigation.matches[0] == &oldCacheMatches[0] {
		t.Fatalf("navigation results were not recomputed after replacement: cache=%#v old=%#v", a.findNavigation, oldCacheMatches)
	}
}

func TestFindReadonlySourceReveal(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	doc := document.New("a needle")
	doc.ReadOnly = true
	tab := a.add(doc, "readonly.txt", "")
	a.Query = "needle"
	setSearchSelection(t, tab, 0, 0)
	a.find(true)
	assertSearchSelection(t, tab, 2, 8)
	if tab.Mode != "source" || !a.persistPending {
		t.Fatalf("readonly source reveal mode=%q persist=%v", tab.Mode, a.persistPending)
	}
}

var benchmarkFindSink []findMatchOffset

func BenchmarkSearchNavigationCacheWarmCold(b *testing.B) {
	b.Run("warm", func(b *testing.B) {
		a := New(b.TempDir())
		b.Cleanup(a.Shutdown)
		a.NewDocument("text")
		tab := a.Current()
		source := strings.Repeat("needle filler ", 4096)
		if err := tab.Doc.Buffer.ReplaceUTF16(0, 0, source); err != nil {
			b.Fatal(err)
		}
		options := search.Options{Query: "needle"}
		if _, err := a.searchMatchesCurrent(tab, options); err != nil {
			b.Fatal(err)
		}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cache, err := a.searchMatchesCurrent(tab, options)
			if err != nil {
				b.Fatal(err)
			}
			benchmarkFindSink = cache.matches
		}
	})
	b.Run("cold", func(b *testing.B) {
		a := New(b.TempDir())
		b.Cleanup(a.Shutdown)
		a.NewDocument("text")
		tab := a.Current()
		source := strings.Repeat("needle filler ", 4096)
		if err := tab.Doc.Buffer.ReplaceUTF16(0, 0, source); err != nil {
			b.Fatal(err)
		}
		options := search.Options{Query: "needle"}
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			a.invalidateFindCache()
			cache, err := a.searchMatchesCurrent(tab, options)
			if err != nil {
				b.Fatal(err)
			}
			benchmarkFindSink = cache.matches
		}
	})
}
