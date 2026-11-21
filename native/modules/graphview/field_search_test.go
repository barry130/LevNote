package graphview

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/search"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/drawio"
	"levnote.local/native/ui/editor"
)

const fieldSearchMindFixture = `{"format":"levnote.mindmap","version":1,"layout":"logicalStructure","future":{"retain":[1,"\u672a\u6765"]},"root":{"id":"root","topic":"🙂 cat","note":"cat🙂","hyperLink":"never-open.txt","futureNode":{"keep":true},"children":[{"id":"a","topic":"CAT category","note":"cat"},{"id":"b","topic":"cat","note":"other"}]}}`
const fieldSearchDrawFixture = `<mxfile future="keep"><diagram id="one" name="First"><mxGraphModel custom="keep"><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="a" value="cat &amp; dog🙂" vertex="1" parent="1" future="exact"><mxGeometry x="10" y="10" width="80" height="40" as="geometry"/></mxCell><mxCell id="b" value="&lt;b&gt;cat&lt;/b&gt;" style="html=1;" vertex="1" parent="1"><mxGeometry x="200" y="10" width="80" height="40" as="geometry"/></mxCell><mxCell id="e" value="cat edge🙂" edge="1" parent="1" source="a" target="b"><mxGeometry relative="1" as="geometry"/></mxCell><object id="opaque" label="cat"/></root></mxGraphModel></diagram><diagram id="two" name="Second"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="z" value="cat other page" vertex="1" parent="1"><mxGeometry x="10" y="20" width="80" height="40" as="geometry"/></mxCell></root></mxGraphModel></diagram></mxfile>`

func fieldFind(t *testing.T, v *view, opts search.Options) editor.FieldSearchResult {
	t.Helper()
	result, err := v.FindFields(opts, editor.DefaultFieldSearchLimits())
	if err != nil {
		t.Fatal(err)
	}
	return result
}
func fieldUnchanged(t *testing.T, b *text.Buffer, source string, version uint64, selection text.Range, history text.HistoryStatus) {
	t.Helper()
	if b.String() != source || b.Version() != version || b.Selection() != selection || b.History() != history || b.InGroup() {
		t.Fatal("refusal/inspection changed authority", b.Version(), version, b.Selection(), selection, b.History(), history)
	}
}
func TestFieldSearchPureOrderUTF16AndSharedOptions(t *testing.T) {
	for _, tc := range []struct {
		name  string
		opts  search.Options
		count int
	}{
		{"literal", search.Options{Query: "cat"}, 6},
		{"case", search.Options{Query: "cat", CaseSensitive: true}, 5},
		{"whole", search.Options{Query: "cat", WholeWord: true}, 5},
		{"regex", search.Options{Query: "(?P<word>cat)", Regex: true, WholeWord: true}, 5},
		{"syntax", search.Options{Query: "topic"}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := text.New(fieldSearchMindFixture)
			_ = b.SetSelection(b.LenUTF16(), 0)
			sel, hist, ver := b.Selection(), b.History(), b.Version()
			changed := 0
			v := newView(kindMindmap, editor.Host{Buffer: b, Changed: func() { changed++ }})
			v.selectedID = "b"
			v.panX = 11
			v.panY = 22
			v.zoom = 1.25
			r := fieldFind(t, v, tc.opts)
			if r.Count != tc.count || r.Truncated || changed != 0 || v.mind != nil || v.draft != nil || v.selectedID != "b" || v.panX != 11 || v.panY != 22 || v.zoom != 1.25 {
				t.Fatal("read-only scan mutated view or wrong count", r, v.selectedID)
			}
			fieldUnchanged(t, b, fieldSearchMindFixture, ver, sel, hist)
			if r.Count > 0 {
				hits := v.search.results.hits
				fields := v.search.results.fields
				if fields[hits[0].field].field != fieldTopic || fields[hits[1].field].field != fieldNote || fields[hits[0].field].id != "root" || hits[0].match.Start != 3 || hits[0].match.End != 6 || hits[1].match.Start != 0 {
					t.Fatal("field-independent UTF16/order", fields, hits)
				}
			}
		})
	}
}
func TestFieldSearchReplaceAllFieldsAtomicUndoMetadata(t *testing.T) {
	b := text.New(fieldSearchMindFixture)
	_ = b.SetSelection(b.LenUTF16(), 0)
	selection := b.Selection()
	changed, opened := 0, 0
	v := newView(kindMindmap, editor.Host{Buffer: b, Changed: func() { changed++ }, OpenReference: func(string) { opened++ }})
	r := fieldFind(t, v, search.Options{Query: "(?P<x>cat)", Regex: true, WholeWord: true})
	result, err := v.ReplaceFieldMatch(r.Token, 0, "${x}!", true)
	if err != nil || result.Count != 5 || !result.Changed || !result.Undoable || changed != 1 || opened != 0 || b.History().UndoDepth != 1 {
		t.Fatal(result, err, changed, b.History())
	}
	doc := parseMind(t, b)
	if doc.Root.Topic != "🙂 cat!" || *doc.Root.Note != "cat!🙂" || doc.Root.Children[0].Topic != "CAT! category" || *doc.Root.Children[0].Note != "cat!" || doc.Root.Children[1].Topic != "cat!" || *doc.Root.Children[1].Note != "other" || *doc.Root.HyperLink != "never-open.txt" {
		t.Fatal("node+field groups collided", b.String())
	}
	for _, part := range []string{`"future"`, `"retain"`, `"futureNode"`, `"keep"`} {
		if !strings.Contains(b.String(), part) {
			t.Fatal("unknown metadata lost", part)
		}
	}
	after := b.String()
	afterSelection := b.Selection()
	if !b.Undo() || b.String() != fieldSearchMindFixture || b.Selection() != selection || !b.Redo() || b.String() != after || b.Selection() != afterSelection {
		t.Fatal("single transaction did not restore exact bytes/reversed selection")
	}
	if _, err := v.ReplaceFieldMatch(r.Token, 0, "bad", false); !errors.Is(err, editor.ErrFieldSearchStale) {
		t.Fatal("written token revived", err)
	}
}
func TestFieldSearchNoteClearExactAndNoopKeepsRedo(t *testing.T) {
	b := text.New(fieldSearchMindFixture)
	v := newView(kindMindmap, editor.Host{Buffer: b})
	r := fieldFind(t, v, search.Options{Query: "cat🙂", CaseSensitive: true})
	if r.Count != 1 {
		t.Fatal(r)
	}
	if result, err := v.ReplaceFieldMatch(r.Token, 0, "", false); err != nil || !result.Changed {
		t.Fatal(result, err)
	}
	if doc := parseMind(t, b); doc.Root.Note == nil || *doc.Root.Note != "" || doc.Root.Topic != "🙂 cat" {
		t.Fatal("empty note must remain present without trim", b.String())
	}
	if !b.Undo() {
		t.Fatal("undo")
	}
	r = fieldFind(t, v, search.Options{Query: "cat🙂", CaseSensitive: true})
	ver, sel, hist := b.Version(), b.Selection(), b.History()
	if result, err := v.ReplaceFieldMatch(r.Token, 0, "cat🙂", false); err != nil || result.Changed || result.Count != 1 {
		t.Fatal(result, err)
	}
	fieldUnchanged(t, b, fieldSearchMindFixture, ver, sel, hist)
	if !b.Redo() {
		t.Fatal("noop cleared redo")
	}
}
func TestFieldSearchBudgetPartialAndAtomicRefusal(t *testing.T) {
	for _, limit := range []int{-1, 0} {
		l := editor.DefaultFieldSearchLimits()
		l.Matches = limit
		b := text.New(fieldSearchMindFixture)
		v := newView(kindMindmap, editor.Host{Buffer: b})
		if _, err := v.FindFields(search.Options{Query: "cat"}, l); !errors.Is(err, editor.ErrFieldSearchBudget) {
			t.Fatal("invalid budget", limit, err)
		}
	}
	for _, delta := range []int{-1, 0, 1} {
		b := text.New(fieldSearchMindFixture)
		v := newView(kindMindmap, editor.Host{Buffer: b})
		l := editor.DefaultFieldSearchLimits()
		l.SourceBytes = len(fieldSearchMindFixture) + delta
		_, err := v.FindFields(search.Options{Query: "cat"}, l)
		if (delta < 0) != errors.Is(err, editor.ErrFieldSearchBudget) {
			t.Fatal("source budget boundary", delta, err)
		}
	}
	for _, lower := range []func(*editor.FieldSearchLimits){
		func(l *editor.FieldSearchLimits) { l.Matches = 2 }, func(l *editor.FieldSearchLimits) { l.Fields = 2 },
		func(l *editor.FieldSearchLimits) { l.ScanBytes = 6 }, func(l *editor.FieldSearchLimits) { l.FieldBytes = 6 },
	} {
		b := text.New(fieldSearchMindFixture)
		v := newView(kindMindmap, editor.Host{Buffer: b})
		l := editor.DefaultFieldSearchLimits()
		lower(&l)
		r, err := v.FindFields(search.Options{Query: "cat"}, l)
		if err != nil || !r.Truncated {
			t.Fatal("missing explicit truncation", r, err)
		}
		ver, sel, hist := b.Version(), b.Selection(), b.History()
		if _, err = v.ReplaceFieldMatch(r.Token, 0, "x", true); !errors.Is(err, editor.ErrFieldSearchPartial) {
			t.Fatal("partial replace-all", err)
		}
		fieldUnchanged(t, b, fieldSearchMindFixture, ver, sel, hist)
	}
	b := text.NewWithLimit(fieldSearchMindFixture, len(fieldSearchMindFixture))
	_ = b.ReplaceUTF16(0, b.LenUTF16(), "")
	b.Undo()
	_ = b.SetSelection(b.LenUTF16(), 0)
	v := newView(kindMindmap, editor.Host{Buffer: b})
	r := fieldFind(t, v, search.Options{Query: "cat"})
	ver, sel, hist := b.Version(), b.Selection(), b.History()
	if _, err := v.ReplaceFieldMatch(r.Token, 0, strings.Repeat("x", 1000), true); !errors.Is(err, text.ErrSizeLimit) {
		t.Fatal("growth preflight", err)
	}
	fieldUnchanged(t, b, fieldSearchMindFixture, ver, sel, hist)
	l := editor.DefaultFieldSearchLimits()
	l.PreviewBytes = 1
	r, err := v.FindFields(search.Options{Query: "🙂"}, l)
	if err != nil || r.Truncated || !r.PreviewTruncated {
		t.Fatal("preview budget must not make full scan partial", r, err)
	}
	if _, err = v.ReplaceFieldMatch(r.Token, 0, "\xff", true); !errors.Is(err, text.ErrUTF8) {
		t.Fatal(err)
	}
	fieldUnchanged(t, b, fieldSearchMindFixture, ver, sel, hist)
}
func TestFieldSearchStaleOwnerQueryPageReleaseAndOtherProvider(t *testing.T) {
	for _, mode := range []string{"source", "buffer-return", "release", "query-return", "page-return", "readonly-return", "other-provider"} {
		t.Run(mode, func(t *testing.T) {
			kind, source := kindMindmap, fieldSearchMindFixture
			if mode == "page-return" {
				kind, source = kindDrawio, fieldSearchDrawFixture
			}
			b := text.New(source)
			v := newView(kind, editor.Host{Buffer: b})
			r := fieldFind(t, v, search.Options{Query: "cat"})
			switch mode {
			case "source":
				_ = b.ReplaceUTF16(0, 0, " ")
			case "buffer-return":
				v.host.Buffer = text.New(source)
				v.ensureSynced(nil)
				v.host.Buffer = b
				v.ensureSynced(nil)
			case "release":
				v.Release()
				v.ensureSynced(nil)
			case "query-return":
				fieldFind(t, v, search.Options{Query: "other"})
				fieldFind(t, v, search.Options{Query: "cat"})
			case "page-return":
				v.setPageIndex(1)
				v.setPageIndex(0)
			case "readonly-return":
				v.host.ReadOnly = true
				v.ensureSynced(nil)
				v.host.ReadOnly = false
				v.ensureSynced(nil)
			case "other-provider":
				other := newView(kind, editor.Host{Buffer: b})
				r = fieldFind(t, other, search.Options{Query: "cat"})
			}
			before, ver, sel, hist := b.String(), b.Version(), b.Selection(), b.History()
			if err := v.RevealFieldMatch(r.Token, 0); !errors.Is(err, editor.ErrFieldSearchStale) {
				t.Fatal(mode, err)
			}
			if _, err := v.ReplaceFieldMatch(r.Token, 0, "bad", true); !errors.Is(err, editor.ErrFieldSearchStale) {
				t.Fatal(mode, err)
			}
			fieldUnchanged(t, b, before, ver, sel, hist)
		})
	}
}
func TestFieldSearchHiddenReadonlyAndNoteReveal(t *testing.T) {
	source := strings.Replace(fieldSearchMindFixture, `"topic":"🙂 cat"`, `"topic":"🙂 cat","collapsed":true`, 1)
	for _, readOnly := range []bool{true, false} {
		b := text.New(source)
		_ = b.SetSelection(b.LenUTF16(), 0)
		sel := b.Selection()
		changed := 0
		v := newView(kindMindmap, editor.Host{Buffer: b, ReadOnly: readOnly, Changed: func() { changed++ }})
		v.selectedID = "root"
		v.panX = 19
		v.panY = 31
		v.zoom = 1.2
		r := fieldFind(t, v, search.Options{Query: "other"})
		err := v.RevealFieldMatch(r.Token, 0)
		if readOnly {
			if !errors.Is(err, editor.ErrOutlineReadOnlyHidden) || v.selectedID != "root" || v.panX != 19 || v.panY != 31 || v.zoom != 1.2 || b.String() != source || b.Selection() != sel || b.CanUndo() || changed != 0 || v.draft != nil {
				t.Fatal("readonly hidden refusal mutated state", err)
			}
		} else {
			if err != nil || changed != 1 || v.selectedID != "b" || v.field != fieldNote || v.draft == nil || v.draft.input.Buffer.String() != "other" || v.draft.input.Buffer.Selection() != (text.Range{Anchor: 0, Head: 5}) {
				t.Fatal("note not visibly revealed", err, v.selectedID, v.field, v.draft)
			}
			if !b.Undo() || b.String() != source || b.Selection() != sel {
				t.Fatal("ancestor expansion not one exact undo")
			}
		}
	}
}
func TestFieldSearchDrawioDecodedEdgesCurrentPageAndRefusal(t *testing.T) {
	b := text.New(fieldSearchDrawFixture)
	changed := 0
	v := newView(kindDrawio, editor.Host{Buffer: b, Changed: func() { changed++ }})
	r := fieldFind(t, v, search.Options{Query: "cat"})
	if r.Count != 2 || r.Truncated || !strings.Contains(r.Scope, "排除") {
		t.Fatal(r)
	}
	if err := v.RevealFieldMatch(r.Token, 1); err != nil || v.selectedID != "e" || v.search.edgeInput == nil || v.search.edgeInput.Buffer.String() != "cat edge🙂" || v.search.edgeInput.Buffer.Selection() != (text.Range{Anchor: 0, Head: 3}) {
		t.Fatal("edge field navigation", err)
	}
	result, err := v.ReplaceFieldMatch(r.Token, 0, "dog & 🙂", true)
	if err != nil || result.Count != 2 || !result.Changed || changed != 1 {
		t.Fatal(result, err)
	}
	after := b.String()
	for _, exact := range []string{`future="keep"`, `future="exact"`, `value="&lt;b&gt;cat&lt;/b&gt;"`, `<object id="opaque" label="cat"/>`, `<diagram id="two" name="Second">`} {
		if !strings.Contains(after, exact) {
			t.Fatal("opaque/other fields changed", exact, after)
		}
	}
	other := fieldSearchDrawFixture[strings.Index(fieldSearchDrawFixture, `<diagram id="two"`):]
	if !strings.HasSuffix(after, other) {
		t.Fatal("other page bytes changed")
	}
	doc, _, err := drawio.Parse([]byte(after), drawio.ParseOptions{})
	if err != nil || doc.Pages[0].Model.Nodes[0].PlainText != "dog & 🙂 & dog🙂" || doc.Pages[0].Model.Edges[0].PlainText != "dog & 🙂 edge🙂" {
		t.Fatal(doc, err)
	}
	if !b.Undo() || b.String() != fieldSearchDrawFixture || !b.Redo() || b.String() != after {
		t.Fatal("drawio multi-field undo/redo")
	}
	for _, source := range []string{fieldSearchDrawFixture, compressedPage(t)} {
		b := text.New(source)
		v := newView(kindDrawio, editor.Host{Buffer: b})
		query := "cat"
		if source != fieldSearchDrawFixture {
			query = "compressed"
		}
		r := fieldFind(t, v, search.Options{Query: query})
		ver, sel, hist := b.Version(), b.Selection(), b.History()
		_, err := v.ReplaceFieldMatch(r.Token, 0, "\x01", true)
		if err == nil {
			t.Fatal("unrepresentable/readonly label accepted")
		}
		fieldUnchanged(t, b, source, ver, sel, hist)
	}
}
func TestFieldSearchHistoryBoundaryAndCallbackOwner(t *testing.T) {
	seed := func(budget int) (*text.Buffer, *view, editor.FieldReplaceResult) {
		b := text.New(fieldSearchMindFixture)
		if budget >= 0 {
			b.SetHistoryLimits(256, budget)
		}
		v := newView(kindMindmap, editor.Host{Buffer: b})
		r := fieldFind(t, v, search.Options{Query: "cat🙂"})
		result, err := v.ReplaceFieldMatch(r.Token, 0, "X", false)
		if err != nil || !result.Changed {
			t.Fatal(result, err)
		}
		return b, v, result
	}
	baseline, _, _ := seed(-1)
	cost := baseline.History().RetainedBytes
	for _, budget := range []int{0, cost - 1, cost, cost + 1} {
		b, v, result := seed(budget)
		want := budget >= cost
		if result.Undoable != want || b.CanUndo() != want || (!want && !strings.Contains(v.search.notice, "此次替换不可撤销")) {
			t.Fatal("history boundary", budget, cost, result, b.History(), v.search.notice)
		}
		if want && (!b.Undo() || b.String() != fieldSearchMindFixture || !b.Redo()) {
			t.Fatal("boundary undo")
		}
	}
	b := text.New(fieldSearchMindFixture)
	var v *view
	replacementOwner := text.New(fieldSearchMindFixture)
	changes := 0
	v = newView(kindMindmap, editor.Host{Buffer: b, Changed: func() { changes++; v.host.Buffer = replacementOwner; v.Release() }})
	r := fieldFind(t, v, search.Options{Query: "cat🙂"})
	result, err := v.ReplaceFieldMatch(r.Token, 0, "written", false)
	if !errors.Is(err, editor.ErrFieldSearchStale) || !result.Changed || result.Count != 1 || changes != 1 || *parseMind(t, b).Root.Note != "written" || replacementOwner.String() != fieldSearchMindFixture {
		t.Fatal("callback write receipt", result, err, changes)
	}
}

func TestFieldSearchMultipleMatchesPerFieldEmojiReverseAndSingle(t *testing.T) {
	source := strings.Replace(fieldSearchMindFixture, `"topic":"🙂 cat","note":"cat🙂"`, `"topic":"🙂 cat cat","note":"cat🙂cat"`, 1)
	b := text.New(source)
	v := newView(kindMindmap, editor.Host{Buffer: b})
	r := fieldFind(t, v, search.Options{Query: `(cat)`, Regex: true, CaseSensitive: true})
	result, err := v.ReplaceFieldMatch(r.Token, 2, "${1}🙂", false)
	if err != nil || result.Count != 1 || !result.Changed {
		t.Fatal(result, err)
	}
	doc := parseMind(t, b)
	if doc.Root.Topic != "🙂 cat cat" || *doc.Root.Note != "cat🙂🙂cat" {
		t.Fatal("single note match changed topic/other range", b.String())
	}
	if !b.Undo() || b.String() != source {
		t.Fatal("single undo")
	}
	r = fieldFind(t, v, search.Options{Query: `(cat)`, Regex: true, CaseSensitive: true})
	result, err = v.ReplaceFieldMatch(r.Token, 0, "${1}🙂", true)
	if err != nil || !result.Changed {
		t.Fatal(result, err)
	}
	doc = parseMind(t, b)
	if doc.Root.Topic != "🙂 cat🙂 cat🙂" || *doc.Root.Note != "cat🙂🙂cat🙂" {
		t.Fatal("reverse field application/UTF16 failed", b.String())
	}
	if !b.Undo() || b.String() != source || !b.Redo() {
		t.Fatal("multi-field reverse undo/redo")
	}
}

func TestFieldSearchAllInvalidLimitsAndExactFieldScanPreviewBudgets(t *testing.T) {
	for _, set := range []func(*editor.FieldSearchLimits, int){
		func(l *editor.FieldSearchLimits, n int) { l.SourceBytes = n }, func(l *editor.FieldSearchLimits, n int) { l.ScanBytes = n },
		func(l *editor.FieldSearchLimits, n int) { l.FieldBytes = n }, func(l *editor.FieldSearchLimits, n int) { l.Fields = n },
		func(l *editor.FieldSearchLimits, n int) { l.Matches = n }, func(l *editor.FieldSearchLimits, n int) { l.PreviewBytes = n },
		func(l *editor.FieldSearchLimits, n int) { l.QueryBytes = n }, func(l *editor.FieldSearchLimits, n int) { l.ReplacementBytes = n },
		func(l *editor.FieldSearchLimits, n int) { l.OutputBytes = n },
	} {
		for _, n := range []int{-1, 0} {
			b := text.New(fieldSearchMindFixture)
			v := newView(kindMindmap, editor.Host{Buffer: b})
			limits := editor.DefaultFieldSearchLimits()
			set(&limits, n)
			if _, err := v.FindFields(search.Options{Query: "cat"}, limits); !errors.Is(err, editor.ErrFieldSearchBudget) || v.mind != nil || b.CanUndo() {
				t.Fatal("invalid limits expanded source", n, err)
			}
		}
	}
	source := `{"format":"levnote.mindmap","version":1,"root":{"id":"r","topic":"🙂cat"}}`
	for _, delta := range []int{-1, 0, 1} {
		b := text.New(source)
		v := newView(kindMindmap, editor.Host{Buffer: b})
		limits := editor.DefaultFieldSearchLimits()
		limits.FieldBytes = len("🙂cat") + delta
		limits.ScanBytes = limits.FieldBytes
		limits.Fields = 1
		limits.Matches = 1
		limits.PreviewBytes = 3
		r, err := v.FindFields(search.Options{Query: "cat"}, limits)
		if err != nil || r.Truncated != (delta < 0) || r.Count != map[bool]int{true: 0, false: 1}[delta < 0] || r.PreviewTruncated {
			t.Fatal("exact field/scan/match/preview boundary", delta, r, err)
		}
	}
}

func TestFieldSearchReplacementWorkBudgetRefusalIsAtomic(t *testing.T) {
	topic := strings.Repeat("x", 16)
	source := strings.Replace(fieldSearchMindFixture, `"topic":"🙂 cat"`, `"topic":"`+topic+`"`, 1)
	query := strings.Repeat(`(?P<late>y)|`, 15) + `(?P<late>x)`
	b := text.New(source)
	v := newView(kindMindmap, editor.Host{Buffer: b})
	r := fieldFind(t, v, search.Options{Query: query, Regex: true, CaseSensitive: true})
	before, version, selection, history := b.String(), b.Version(), b.Selection(), b.History()
	work := 1024
	result, err := v.replaceFieldMatchWithWork(r.Token, 0, strings.Repeat("$missing", 64), true, &work)
	if result.Changed || !errors.Is(err, editor.ErrFieldSearchBudget) {
		t.Fatal("replacement work budget was not a stable visible refusal", result, err)
	}
	fieldUnchanged(t, b, before, version, selection, history)
}

func fieldSearchChangedReentry(t *testing.T, toolbar, all, persistent bool) {
	t.Helper()
	b := text.New(fieldSearchDrawFixture)
	_ = b.SetSelection(b.LenUTF16(), 0)
	selection, version := b.SelectionSet(), b.Version()
	newNotice, newErr := "new query notice", errors.New("new query error")
	newOpts := search.Options{Query: "edge", Regex: true, CaseSensitive: true, WholeWord: true}
	var v *view
	var newResult editor.FieldSearchResult
	var newEdge *editor.Editor
	var newOwner fieldSearchOwner
	var newEpoch uint64
	changed := 0
	v = newView(kindDrawio, editor.Host{Buffer: b, Changed: func() {
		changed++
		var err error
		newResult, err = v.FindFields(newOpts, editor.DefaultFieldSearchLimits())
		if err != nil || newResult.Count != 1 || newResult.Truncated {
			t.Fatal("reentrant complete find", newResult, err)
		}
		if err = v.RevealFieldMatch(newResult.Token, 0); err != nil {
			t.Fatal("reentrant reveal", err)
		}
		v.search.notice, v.search.err, v.search.persistentNotice = newNotice, newErr, persistent
		newEdge, newOwner, newEpoch = v.search.edgeInput, v.fieldSearchOwner(), v.search.epoch
	}})
	var old editor.FieldSearchResult
	var tt *ui.Tester
	if toolbar {
		tt = ui.NewTester(v.Draw, 1100, 760)
		fieldUIOpen(t, tt)
		tt.Type("cat")
		tt.Key(0, ui.KeyEnter)
		fieldUISet(t, tt, "字段替换文字", "dog & 🙂")
		old = v.search.results.result
	} else {
		old = fieldFind(t, v, search.Options{Query: "cat"})
	}
	if err := v.RevealFieldMatch(old.Token, 1); err != nil {
		t.Fatal("old edge reveal", err)
	}
	oldEdge := v.search.edgeInput
	if toolbar {
		// Publish the API-selected edge into the displayed button target before
		// sending native pointer events; the previous frame displayed index -1.
		tt.Frame()
	}
	count := 1
	if all {
		count = 2
	}
	if toolbar {
		label := "替换当前字段匹配"
		if all {
			label = "替换全部字段匹配"
		}
		fieldUIClick(t, tt, label)
	} else {
		result, err := v.ReplaceFieldMatch(old.Token, 1, "dog & 🙂", all)
		if !errors.Is(err, editor.ErrFieldSearchStale) || !result.Changed || result.Count != count || !result.Undoable {
			t.Fatal("committed stale receipt must retain true result", result, err)
		}
	}
	want := strings.Replace(fieldSearchDrawFixture, `value="cat edge🙂"`, `value="dog &amp; 🙂 edge🙂"`, 1)
	if all {
		want = strings.Replace(want, `value="cat &amp; dog🙂"`, `value="dog &amp; 🙂 &amp; dog🙂"`, 1)
	}
	if b.String() != want || b.Version() != version+1 || changed != 1 || b.History().UndoDepth != 1 {
		t.Fatal("complete source/count/transaction differs", b.String(), b.Version(), changed, b.History(), count)
	}
	if newEdge == nil || newEdge == oldEdge || v.search.edgeInput != newEdge || newEdge.Buffer.String() != "dog & 🙂 edge🙂" || !newEdge.ReadOnly {
		t.Fatal("new edge publication was cleared/replaced", v.search.edgeInput)
	}
	start := text.UTF16Len("dog & 🙂 ")
	if newEdge.Buffer.Selection() != (text.Range{Anchor: start, Head: start + len("edge")}) {
		t.Fatal("new field UTF16 reveal lost", newEdge.Buffer.Selection())
	}
	if v.search.notice != newNotice || v.search.err != newErr || v.search.persistentNotice != persistent || v.fieldSearchOwner() != newOwner || v.search.epoch != newEpoch || v.search.index != 0 {
		t.Fatal("old publication overwrote new state", v.search.notice, v.search.err, v.search.persistentNotice)
	}
	if v.search.results == nil || v.search.results.result != newResult || newResult.Token == old.Token || v.search.results.opts != newOpts || !strings.Contains(newResult.Scope, "draw.io") {
		t.Fatal("new query/token/scope was consumed", v.search.results)
	}
	if err := v.RevealFieldMatch(old.Token, 1); !errors.Is(err, editor.ErrFieldSearchStale) || v.search.results.result.Token != newResult.Token || v.search.edgeInput != newEdge || v.search.err != newErr {
		t.Fatal("old token revived/removed the new publication", err)
	}
	if !b.Undo() || b.String() != fieldSearchDrawFixture || b.SelectionSet().Primary != selection.Primary || fmt.Sprint(b.SelectionSet().Ranges) != fmt.Sprint(selection.Ranges) {
		t.Fatal("committed stale replacement lost exact Undo", b.String(), b.SelectionSet(), selection)
	}
	if !b.Redo() || b.String() != want {
		t.Fatal("committed stale replacement lost exact Redo", b.String())
	}
}
func TestFieldSearchChangedReentryPreservesNewPublicationAndUndo(t *testing.T) {
	for _, all := range []bool{false, true} {
		for _, persistent := range []bool{false, true} {
			t.Run(fmt.Sprintf("all=%t/persistent=%t", all, persistent), func(t *testing.T) {
				fieldSearchChangedReentry(t, false, all, persistent)
			})
		}
	}
}
func TestFieldSearchUIChangedReentryPreservesNewPublication(t *testing.T) {
	for _, all := range []bool{false, true} {
		for _, persistent := range []bool{false, true} {
			t.Run(fmt.Sprintf("all=%t/persistent=%t", all, persistent), func(t *testing.T) {
				fieldSearchChangedReentry(t, true, all, persistent)
			})
		}
	}
}

func TestFieldSearchReplacementExactCommonLedgerOracle(t *testing.T) {
	const source = "{\n  \"format\": \"levnote.mindmap\",\n  \"version\": 1,\n  \"layout\": \"logicalStructure\",\n  \"root\": {\n    \"id\": \"r\",\n    \"topic\": \"xy\",\n    \"children\": []\n  }\n}\n"
	want := strings.Replace(source, `"xy"`, `"x!y!"`, 1)
	// Independent arithmetic: core's two-hit oracle costs 140. Graph adds
	// scan/group=3N+4, field lookup=N+3, format-bound scan=N+1,
	// two upper-bound passes=2U+1, serialized copy=M, commit=1+9(N+M),
	// selection preflight=1+8M per range (4 boundary scans, 2 probe
	// normalization scans, 2 authoritative normalization scans). N=152,
	// M=154; 58 original JSON bytes are outside strings,
	// U=6N+1024*58+6*4=60328. Total=125707 (old 124783 + 6*154).
	const total = 125707
	if len(source) != 152 || len(want) != 154 {
		t.Fatal("oracle fixture changed")
	}
	for _, budget := range []int{0, 54, 55, 55 + 460, 55 + 460 + 155, 1024, total - 1, total, total + 1} {
		t.Run(fmt.Sprint(budget), func(t *testing.T) {
			b := text.New(source)
			changed := 0
			v := newView(kindMindmap, editor.Host{Buffer: b, Changed: func() { changed++ }})
			r := fieldFind(t, v, search.Options{Query: `(?P<n>x)|(?P<n>y)`, Regex: true, CaseSensitive: true})
			before, version, selection, history := b.String(), b.Version(), b.Selection(), b.History()
			work := budget
			result, err := v.replaceFieldMatchWithWork(r.Token, 0, "${n}!", true, &work)
			if budget < total {
				if !errors.Is(err, editor.ErrFieldSearchBudget) || !errors.Is(err, search.ErrReplacementWork) || result.Changed || changed != 0 || work != 0 {
					t.Fatal("shared phase budget did not atomically refuse", result, err, changed, work)
				}
				fieldUnchanged(t, b, before, version, selection, history)
			} else {
				if err != nil || !result.Changed || !result.Undoable || result.Count != 2 || changed != 1 || b.String() != want || work != budget-total {
					t.Fatal("exact combined parse/bind/hit/stitch/format/commit oracle", result, err, changed, work, b.String())
				}
				if !b.Undo() || b.String() != source {
					t.Fatal("boundary transaction Undo")
				}
			}
		})
	}
}

func TestFieldSearchReplacementMultiRangeCommonLedgerOracle(t *testing.T) {
	const source = "{\n  \"format\": \"levnote.mindmap\",\n  \"version\": 1,\n  \"layout\": \"logicalStructure\",\n  \"root\": {\n    \"id\": \"r\",\n    \"topic\": \"xy\",\n    \"children\": []\n  }\n}\n"
	want := strings.Replace(source, `"xy"`, `"x!y!"`, 1)
	// Same complete replacement and candidate for every range count. The
	// independently counted single-range cost is 125707; each additional
	// range contributes exactly 1+8*154=1233. These are mathematical constants,
	// not thresholds discovered by calling the implementation.
	for n, total := range []int{125707, 126940, 128173, 129406} {
		for _, budget := range []int{total - 1, total, total + 1} {
			t.Run(fmt.Sprintf("ranges=%d/budget=%d", n+1, budget), func(t *testing.T) {
				b := text.New(source)
				ranges := make([]text.Range, n+1)
				for i := range ranges {
					ranges[i] = text.Range{Anchor: i * 2, Head: i * 2}
				}
				if err := b.SetSelectionSet(text.SelectionSet{Ranges: ranges, Primary: n}); err != nil {
					t.Fatal(err)
				}
				version, generation, selection, history := b.Version(), b.SelectionGeneration(), b.SelectionSet(), b.History()
				changed := 0
				v := newView(kindMindmap, editor.Host{Buffer: b, Changed: func() { changed++ }})
				r := fieldFind(t, v, search.Options{Query: `(?P<n>x)|(?P<n>y)`, Regex: true, CaseSensitive: true})
				work := budget
				result, err := v.replaceFieldMatchWithWork(r.Token, 0, "${n}!", true, &work)
				if budget < total {
					if !errors.Is(err, editor.ErrFieldSearchBudget) || !errors.Is(err, search.ErrReplacementWork) || result.Changed || changed != 0 || b.String() != source || b.Version() != version || b.SelectionGeneration() != generation || !reflect.DeepEqual(b.SelectionSet(), selection) || b.History() != history || b.InGroup() || work != 0 {
						t.Fatal("multi-range complete-ledger refusal was not atomic", result, err, work)
					}
					return
				}
				if err != nil || !result.Changed || result.Count != 2 || !result.Undoable || changed != 1 || b.String() != want || b.Version() != version+1 || !reflect.DeepEqual(b.SelectionSet(), selection) || b.History().UndoDepth != 1 || work != budget-total {
					t.Fatal("multi-range full replacement exact ledger", result, err, work)
				}
				if !b.Undo() || b.String() != source || !reflect.DeepEqual(b.SelectionSet(), selection) || b.CanUndo() || !b.Redo() || b.String() != want || !reflect.DeepEqual(b.SelectionSet(), selection) || b.CanRedo() || changed != 1 {
					t.Fatal("multi-range full replacement single exact Undo/Redo")
				}
			})
		}
	}
}

func TestFieldSearchCommitMultiRangeCoordinateLedgerAndHistory(t *testing.T) {
	const source = `{"format":"levnote.mindmap","version":1,"layout":"logicalStructure","root":{"id":"r","topic":"x🙂y","children":[]}}`
	const want = `{"format":"levnote.mindmap","version":1,"layout":"logicalStructure","root":{"id":"r","topic":"🙂","children":[]}}`
	// Independent tiny commit oracle: N=117, M=115 bytes, before/after
	// UTF-16 lengths=115/113, topic starts at 94. Fixed commit=1+9*(117+115)
	// =2089. Each original range costs 1+8*115=921, even if endpoints later
	// clamp or merge. Totals for 1..4 ranges are 3010,3931,4852,5773.
	// Original offset 95 is a valid boundary before 🙂; after removing x it
	// lies inside that surrogate pair and clamps to 94. Tail 115/114→113/113.
	if len(source) != 117 || len(want) != 115 || text.UTF16Len(source) != 115 || text.UTF16Len(want) != 113 {
		t.Fatal("independent coordinate fixture changed")
	}
	original := []text.Range{{Anchor: 97, Head: 95}, {Anchor: 0, Head: 2}, {Anchor: 115, Head: 114}, {Anchor: 12, Head: 14}}
	expected := []text.SelectionSet{
		{Ranges: []text.Range{{Anchor: 97, Head: 94}}, Primary: 0},
		{Ranges: []text.Range{{Anchor: 0, Head: 2}, {Anchor: 97, Head: 94}}, Primary: 1},
		{Ranges: []text.Range{{Anchor: 0, Head: 2}, {Anchor: 97, Head: 94}, {Anchor: 113, Head: 113}}, Primary: 1},
		{Ranges: []text.Range{{Anchor: 0, Head: 2}, {Anchor: 12, Head: 14}, {Anchor: 97, Head: 94}, {Anchor: 113, Head: 113}}, Primary: 2},
	}
	for n, total := range []int{3010, 3931, 4852, 5773} {
		for _, budget := range []int{total - 1, total, total + 1} {
			t.Run(fmt.Sprintf("ranges=%d/budget=%d", n+1, budget), func(t *testing.T) {
				b := text.New(source)
				if err := b.ReplaceUTF16(115, 115, " "); err != nil || !b.Undo() || b.String() != source || !b.CanRedo() {
					t.Fatal("redo seed", err)
				}
				if err := b.SetSelectionSet(text.SelectionSet{Ranges: original[:n+1], Primary: 0}); err != nil {
					t.Fatal("original directed selections", err)
				}
				version, generation, selection, history := b.Version(), b.SelectionGeneration(), b.SelectionSet(), b.History()
				changed := 0
				v := newView(kindMindmap, editor.Host{Buffer: b, Changed: func() { changed++ }})
				fieldFind(t, v, search.Options{Query: "x"})
				work := budget
				result, err := v.commitFieldSearchCandidateWithWork(v.search.results, want, "替换", &work)
				if budget < total {
					if !errors.Is(err, editor.ErrFieldSearchBudget) || !errors.Is(err, search.ErrReplacementWork) || result.Changed || changed != 0 || b.String() != source || b.Version() != version || b.SelectionGeneration() != generation || !reflect.DeepEqual(b.SelectionSet(), selection) || b.History() != history || b.InGroup() || work != 0 {
						t.Fatal("coordinate refusal changed bytes/full selections/version/generation/history/redo/Changed", result, err, work)
					}
					if !b.Redo() || b.String() != source+" " || changed != 0 {
						t.Fatal("refusal lost the actual pre-existing redo")
					}
					return
				}
				if err != nil || !result.Changed || !result.Undoable || !result.SourceSelectionClamped || b.String() != want || b.Version() != version+1 || changed != 1 || !reflect.DeepEqual(b.SelectionSet(), expected[n]) || b.History().UndoDepth != 1 || b.CanRedo() || b.InGroup() || work != budget-total {
					t.Fatal("coordinate ceiling/direction/primary/tail/surrogate", result, err, b.SelectionSet(), expected[n], work)
				}
				if !b.Undo() || b.String() != source || !reflect.DeepEqual(b.SelectionSet(), selection) || b.CanUndo() || !b.Redo() || b.String() != want || !reflect.DeepEqual(b.SelectionSet(), expected[n]) || b.CanRedo() || changed != 1 {
					t.Fatal("coordinate commit single exact Undo/Redo", b.History())
				}
			})
		}
	}
}

func TestFieldSearchChangedReceiptInputAndSameQueryReentry(t *testing.T) {
	for _, mode := range []string{"same-query-new-token", "query-buffer", "replacement-buffer", "query-session", "replacement-session", "options", "selection", "toolbar-open"} {
		t.Run(mode, func(t *testing.T) {
			b := text.New(fieldSearchDrawFixture)
			var v *view
			newErr := errors.New("new publication error")
			var newToken editor.FieldSearchToken
			v = newView(kindDrawio, editor.Host{Buffer: b, Changed: func() {
				s := &v.search
				switch mode {
				case "same-query-new-token":
					r := fieldFind(t, v, search.Options{Query: "cat"})
					newToken = r.Token
					if err := v.RevealFieldMatch(r.Token, 1); err != nil {
						t.Fatal(err)
					}
				case "query-buffer":
					_ = s.query.Buffer.ReplaceUTF16(0, 0, "new")
				case "replacement-buffer":
					_ = s.replacement.Buffer.ReplaceUTF16(0, 0, "new")
				case "query-session":
					s.query = editor.New(s.query.Buffer)
				case "replacement-session":
					s.replacement = editor.New(s.replacement.Buffer)
				case "options":
					s.opts.Regex = !s.opts.Regex
				case "selection":
					_ = b.SetSelection(1, 1)
				case "toolbar-open":
					s.open = !s.open
				}
				s.notice, s.err, s.persistentNotice = "new publication", newErr, true
			}})
			v.initFieldSearchInputs()
			r := fieldFind(t, v, search.Options{Query: "cat"})
			result, err := v.ReplaceFieldMatch(r.Token, 0, "cat!", true)
			if !errors.Is(err, editor.ErrFieldSearchStale) || !result.Changed || result.Count != 2 || !result.Undoable || v.search.notice != "new publication" || v.search.err != newErr || !v.search.persistentNotice {
				t.Fatal("changed receipt did not suppress old publication", result, err, v.search.notice, v.search.err)
			}
			if mode == "same-query-new-token" && (newToken == r.Token || v.search.results == nil || v.search.results.result.Token != newToken || v.search.edgeInput == nil) {
				t.Fatal("identical options revived old token/lost new preview")
			}
			if !b.Undo() || b.String() != fieldSearchDrawFixture {
				t.Fatal("receipt rejection rolled back/lost committed history")
			}
		})
	}
}

func TestFieldSearchReadonlyVisibleAndOutputWorkBudgetRefusal(t *testing.T) {
	b := text.New(fieldSearchMindFixture)
	v := newView(kindMindmap, editor.Host{Buffer: b, ReadOnly: true})
	r := fieldFind(t, v, search.Options{Query: "cat🙂"})
	if err := v.RevealFieldMatch(r.Token, 0); err != nil || v.field != fieldNote || v.draft == nil || !v.draft.input.ReadOnly {
		t.Fatal("readonly visible note", err)
	}
	before, ver, sel, hist := b.String(), b.Version(), b.Selection(), b.History()
	if _, err := v.ReplaceFieldMatch(r.Token, 0, "bad", false); err == nil {
		t.Fatal("readonly replaced")
	}
	fieldUnchanged(t, b, before, ver, sel, hist)
	b = text.New(fieldSearchMindFixture)
	v = newView(kindMindmap, editor.Host{Buffer: b})
	limits := editor.DefaultFieldSearchLimits()
	limits.OutputBytes = len(fieldSearchMindFixture)
	r, err := v.FindFields(search.Options{Query: "cat"}, limits)
	if err != nil {
		t.Fatal(err)
	}
	before, ver, sel, hist = b.String(), b.Version(), b.Selection(), b.History()
	if _, err = v.ReplaceFieldMatch(r.Token, 0, "growing replacement", true); !errors.Is(err, editor.ErrFieldSearchBudget) {
		t.Fatal("output limit did not atomically refuse", err)
	}
	fieldUnchanged(t, b, before, ver, sel, hist)
}
