package editor

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

type languageReentryFixture struct {
	e      *Editor
	b      *text.Buffer
	tester *ui.Tester
	target SourceTarget
	span   text.Range
	line   int
}

func newLanguageReentryFixture(t *testing.T, long bool) languageReentryFixture {
	t.Helper()
	body := "🙂 e\u0301 target 中文 " + strings.Repeat("word ", 35)
	if long {
		body += strings.Repeat("long 中文🙂 ", 650)
	}
	prefix := "\ufeff" + strings.Repeat("prefix ", 36) + "\r\n" + strings.Repeat("padding\n", 24) + "{\r\nouter\n{\n"
	source := prefix + body + "\r\n}\n}\r\ntail rest\n{\nsibling\n}\n"
	b := text.New(source)
	if err := b.ReplaceUTF16(b.LenUTF16(), b.LenUTF16(), "!"); err != nil || !b.Undo() {
		t.Fatal("redo fixture", err)
	}
	e := New(b)
	e.SoftWrap = true
	tester := ui.NewTester(func(c *ui.Context) {
		ui.Button(c, "outside")
		e.View(c)
	}, 320, 190)
	if err := installBraceProvider(e); err != nil {
		t.Fatal(err)
	}
	start := text.UTF16Len(prefix)
	line, _, err := b.LineColumn(start)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.ToggleFoldAt(line - 1); err != nil {
		t.Fatal(err)
	}
	if err := e.ToggleFoldAt(line - 3); err != nil {
		t.Fatal(err)
	}
	if err := tester.Click("outside"); err != nil {
		t.Fatal(err)
	}
	tail := start + text.UTF16Len(body) + text.UTF16Len("\r\n}\n}\r\n")
	setSelections(t, b, []text.Range{{Anchor: 5, Head: 1}, {Anchor: tail + 4, Head: tail + 1}}, 1)
	e.hasDesiredX, e.desiredX = true, 17
	e.affinity = ui.TextUpstream
	if _, cached := e.cache[line]; cached || e.longLineWarned {
		t.Fatal("target must have a cold layout and no prior long-line disclosure")
	}
	if len(e.rowCounts) == 0 || len(e.rowTree) == 0 || !tester.Focused("outside") {
		t.Fatal("fixture needs retained wrapped rows and actual outside focus")
	}
	return languageReentryFixture{e: e, b: b, tester: tester, target: captureLanguageTarget(t, e), span: text.Range{Anchor: start, Head: start + text.UTF16Len(body)}, line: line}
}

// Both boundary snapshots are raw. In particular no Frame, Draw, Bounds,
// fold-syncing getter or observing inputStamp may repair state around the call.
func TestLanguageReentryRejectsCallbackChangesBeforeDraw(t *testing.T) {
	for _, trigger := range []string{"highlight", "long-error"} {
		for _, mutation := range []string{
			"buffer-primed", "buffer-ABA", "readonly", "readonly-ABA", "release",
			"edit-undo", "secondary", "primary", "direction", "selection-ABA",
			"group", "composition", "provider", "fold-ABA", "style", "style-ABA", "scroll", "nested-reveal",
			"wrap-ABA", "geometry", "row-count-ABA", "row-tree-ABA", "focus-request", "highlight-change", "error-change",
		} {
			t.Run(trigger+"/"+mutation, func(t *testing.T) {
				f := newLanguageReentryFixture(t, trigger == "long-error")
				e, b := f.e, f.b
				other := text.New("replacement cache must remain attached to this buffer")
				entry := captureSourceEditorRejectState(e)
				entryBuffer := captureSourceRejectState(b)
				var after []any
				var afterBuffers [2]sourceRejectState
				armed, calls, changes := true, 0, 0
				e.OnChange = func() { changes++ }
				hook := func() {
					if !armed {
						return
					}
					armed = false
					calls++
					if !reflect.DeepEqual(captureSourceEditorRejectState(e), entry) || !reflect.DeepEqual(captureSourceRejectState(b), entryBuffer) {
						t.Error("host callback observed provisional navigation before commit")
					}
					set := b.SelectionSet()
					switch mutation {
					case "buffer-primed":
						e.SetBuffer(other)
						e.line(0)
						e.ScrollX, e.ScrollY = 23, 7
					case "buffer-ABA":
						e.SetBuffer(other)
						e.SetBuffer(b)
					case "readonly":
						e.SetReadOnly(true)
					case "readonly-ABA":
						e.SetReadOnly(true)
						e.SetReadOnly(false)
					case "release":
						e.ReleaseView()
					case "edit-undo":
						if err := b.ReplaceUTF16(b.LenUTF16(), b.LenUTF16(), "!"); err != nil || !b.Undo() {
							t.Fatal("callback edit/undo", err)
						}
					case "secondary", "primary", "direction", "selection-ABA":
						changed := cloneSourceSelectionSet(set)
						switch mutation {
						case "secondary", "selection-ABA":
							changed.Ranges[0] = text.Range{Anchor: 6, Head: 2}
						case "primary":
							changed.Primary = (set.Primary + 1) % len(set.Ranges)
						case "direction":
							changed.Ranges[0].Anchor, changed.Ranges[0].Head = changed.Ranges[0].Head, changed.Ranges[0].Anchor
						}
						if err := b.SetSelectionSet(changed); err != nil {
							t.Fatal(err)
						}
						if mutation == "selection-ABA" {
							if err := b.SetSelectionSet(set); err != nil {
								t.Fatal(err)
							}
						}
					case "group":
						if err := b.BeginGroup(); err != nil {
							t.Fatal(err)
						}
					case "composition":
						e.nativeTextClient().SetMarkedText(nil, "拼", ui.TextInputRange{Start: 1, End: 1})
					case "provider":
						if err := installBraceProvider(e); err != nil {
							t.Fatal(err)
						}
					case "fold-ABA":
						for range 2 {
							if err := e.ToggleFoldAt(b.LineCount() - 4); err != nil {
								t.Fatal(err)
							}
						}
					case "style":
						e.SetFontSize(20)
					case "style-ABA":
						e.SetFontSize(20)
						e.SetFontSize(16)
					case "scroll":
						e.ScrollX += 13
						e.ScrollY += 29
					case "wrap-ABA":
						e.SetSoftWrap(false)
						e.SetSoftWrap(true)
					case "geometry":
						e.width += 37
						e.height += 11
					case "row-count-ABA":
						old := max(1, e.rowCounts[0])
						e.setRows(0, old+1)
						e.setRows(0, old)
					case "row-tree-ABA":
						e.addRows(0, 1)
						e.addRows(0, -1)
					case "focus-request":
						e.Focus()
					case "highlight-change":
						e.Highlight = nil
					case "error-change":
						e.OnError = nil
					case "nested-reveal":
						if err := e.RevealSourceTarget(captureLanguageTarget(t, e), 0, 0); err != nil {
							t.Fatal("nested reveal", err)
						}
					}
					after = captureSourceEditorRejectState(e)
					afterBuffers = [2]sourceRejectState{captureSourceRejectState(b), captureSourceRejectState(other)}
				}
				e.Highlight = func(value string) []ui.Span {
					if trigger == "highlight" {
						hook()
					}
					return []ui.Span{{Text: value}}
				}
				e.OnError = func(err error) {
					if !errors.Is(err, errLineTooLong) {
						t.Errorf("unexpected error callback: %v", err)
					}
					if trigger == "long-error" {
						hook()
					}
				}
				err := e.RevealSourceTarget(f.target, f.span.Head, f.span.Anchor)
				if !errors.Is(err, errSourceTargetStale) {
					t.Errorf("callback-invalidated reveal = %v, want stale", err)
				}
				if calls != 1 || after == nil {
					t.Fatalf("cold %s callback was not exercised exactly once: %d", trigger, calls)
				}
				if !reflect.DeepEqual(captureSourceEditorRejectState(e), after) {
					t.Error("old reveal overwrote callback-owned raw Editor state before redraw")
				}
				requireSourceRejectState(t, b, afterBuffers[0])
				requireSourceRejectState(t, other, afterBuffers[1])
				if mutation != "composition" && changes != 0 {
					t.Fatal("navigation emitted OnChange")
				}
				if !f.tester.Focused("outside") {
					t.Fatal("old reveal changed committed toolkit focus before redraw")
				}
				// Composition/group cleanup belongs to this explicit owner after
				// the raw oracle, never to the rejected reveal.
				if mutation == "group" {
					if !b.InGroup() || !b.CancelGroup() {
						t.Fatal("foreign group ownership was lost")
					}
				}
				if mutation == "composition" {
					if !e.compositionMulti || !b.OwnsComposition(e.compositionToken) {
						t.Fatal("new native pending owner was lost")
					}
					e.cancelComposition()
				}
				if mutation != "nested-reveal" && mutation != "focus-request" {
					f.tester.Frame()
					if !f.tester.Focused("outside") {
						t.Fatal("old reveal left a focus request after callback invalidation")
					}
				}
			})
		}
	}
}

func TestLanguageReentryLongWindowAndRichMetrics(t *testing.T) {
	for _, wrap := range []bool{false, true} {
		for _, rich := range []bool{false, true} {
			t.Run(map[bool]string{false: "horizontal", true: "wrap"}[wrap]+"/"+map[bool]string{false: "plain", true: "rich"}[rich], func(t *testing.T) {
				body := strings.Repeat("a", segmentUnits-1) + "🙂e\u0301" + strings.Repeat("b", maxShapedLine+17) + "🙂tail"
				b := text.New("{\n" + body + "\r\n}\n")
				e := New(b)
				e.SoftWrap = wrap
				e.affinity = ui.TextUpstream
				if err := installBraceProvider(e); err != nil {
					t.Fatal(err)
				}
				if err := e.ToggleFoldAt(0); err != nil {
					t.Fatal(err)
				}
				_ = ui.NewTester(e.View, 320, 160)
				target := captureLanguageTarget(t, e)
				entry, before := captureSourceEditorRejectState(e), captureSourceRejectState(b)
				value := "🙂e\u0301" + strings.Repeat("b", segmentUnits-3)
				calls, warnings := 0, 0
				e.OnError = func(err error) {
					warnings++
					if !errors.Is(err, errLineTooLong) || !reflect.DeepEqual(entry, captureSourceEditorRejectState(e)) {
						t.Fatal("long-window warning observed provisional state", err)
					}
				}
				e.Highlight = func(got string) []ui.Span {
					calls++
					if got != value || !reflect.DeepEqual(entry, captureSourceEditorRejectState(e)) || !reflect.DeepEqual(before, captureSourceRejectState(b)) {
						t.Fatal("long-window callback changed source/entry or split a surrogate")
					}
					if rich {
						return []ui.Span{{Text: got, Size: 21, Weight: 700}}
					}
					return nil
				}
				const bodyStart, column = 2, 4095
				if err := e.RevealSourceTarget(target, bodyStart, bodyStart+column); err != nil {
					t.Fatal(err)
				}
				l, ok := e.cache[1]
				if !ok || l.value != value || l.windowStart != segmentUnits-1 || l.start != bodyStart+segmentUnits-1 || calls != 1 || warnings != 1 {
					t.Fatal("incorrect bounded long-line window before Draw")
				}
				font := ui.Font{Family: "monospace", Size: 16, LineHeight: 1.65}
				width := float32(0)
				if wrap {
					width = e.width - e.gutter() - 16
				}
				want := ui.ShapeText(value, font, width)
				if rich {
					want = ui.ShapeRichText([]ui.Span{{Text: value, Size: 21, Weight: 700}}, font, width)
				}
				caret := ui.TextCaretPosition{Index: column - (segmentUnits - 1), Affinity: ui.TextUpstream}
				if !reflect.DeepEqual(l.lines, want.Lines()) || l.layout.CaretAt(caret) != want.CaretAt(caret) {
					t.Fatal("prepared rich/plain metrics differ from toolkit shaping")
				}
				if wrap && (l.rows < 2 || e.ScrollX != 0) || !wrap && (l.rows != 1 || e.ScrollX <= 0) {
					t.Fatal("long-window wrap/horizontal scroll behavior changed")
				}
			})
		}
	}
}

func TestLanguageReentryNoOpNestedNavigationAndEmptyRich(t *testing.T) {
	for _, nested := range []bool{false, true} {
		t.Run(map[bool]string{false: "empty-rich", true: "nested-no-op"}[nested], func(t *testing.T) {
			f := newLanguageReentryFixture(t, false)
			e, b := f.e, f.b
			if err := b.SetSelection(0, 0); err != nil {
				t.Fatal(err)
			}
			entry := captureSourceEditorRejectState(e)
			var after []any
			calls := 0
			e.Highlight = func(value string) []ui.Span {
				calls++
				if !reflect.DeepEqual(entry, captureSourceEditorRejectState(e)) {
					t.Fatal("callback entry changed")
				}
				if nested {
					generation := b.SelectionGeneration()
					if err := e.RevealSourceTarget(f.target, 0, 0); err != nil || b.SelectionGeneration() != generation {
						t.Fatal("nested navigation should keep this already-single selection", err)
					}
					after = captureSourceEditorRejectState(e)
				}
				return nil
			}
			err := e.RevealSourceTarget(f.target, f.span.Head, f.span.Anchor)
			if nested {
				if !errors.Is(err, errSourceTargetStale) || !reflect.DeepEqual(after, captureSourceEditorRejectState(e)) {
					t.Fatal("outer call overwrote nested navigation without a selection change", err)
				}
			} else if err != nil {
				t.Fatal("nil highlighter result must use plain text", err)
			}
			if calls != 1 {
				t.Fatal("unexpected callback replay", calls)
			}
		})
	}

	b := text.New("{\n\n}\n")
	e := New(b)
	if err := installBraceProvider(e); err != nil {
		t.Fatal(err)
	}
	if err := e.ToggleFoldAt(0); err != nil {
		t.Fatal(err)
	}
	_ = ui.NewTester(e.View, 320, 160)
	target := captureLanguageTarget(t, e)
	calls := 0
	e.Highlight = func(value string) []ui.Span {
		calls++
		if value != "" {
			t.Fatal("expected empty physical line")
		}
		return nil
	}
	if err := e.RevealSourceTarget(target, 2, 2); err != nil || calls != 1 || e.cache[1].layout == nil {
		t.Fatal("empty rich line failed", err)
	}
}

func TestLanguageReentryReadOnlyObserversSeeEntryAndCommitPreparedGeometry(t *testing.T) {
	for _, long := range []bool{false, true} {
		for _, rich := range []bool{false, true} {
			t.Run(map[bool]string{false: "short", true: "long"}[long]+"/"+map[bool]string{false: "plain-fallback", true: "rich"}[rich], func(t *testing.T) {
				f := newLanguageReentryFixture(t, long)
				e, b := f.e, f.b
				e.SetReadOnly(true)
				f.target = captureLanguageTarget(t, e)
				// Navigation tokens ignore ordinary selection movement since capture;
				// its separate transaction must still protect the newer entry set.
				setSelections(t, b, []text.Range{{Anchor: 7, Head: 2}, {Anchor: b.LenUTF16(), Head: b.LenUTF16()}}, 0)
				entry, before := captureSourceEditorRejectState(e), captureSourceRejectState(b)
				highlights, warnings, changes := 0, 0, 0
				observe := func() {
					if !reflect.DeepEqual(entry, captureSourceEditorRejectState(e)) || !reflect.DeepEqual(before, captureSourceRejectState(b)) {
						t.Error("read-only host callback saw provisional navigation")
					}
					if snapshot, err := e.CaptureSourceSnapshot(maxLanguageSourceBytes); err != nil || !equalSourceSelectionSet(snapshot.Selections, b.SelectionSet()) {
						t.Error("read-only source callback could not inspect entry state", err)
					}
					if err := e.CheckSourceTarget(f.target, false); err != nil {
						t.Error("read-only callback rejected entry target", err)
					}
				}
				e.Highlight = func(value string) []ui.Span {
					highlights++
					observe()
					if !rich {
						return []ui.Span{{Text: "host cannot replace source text"}}
					}
					return []ui.Span{{Text: value, Color: ui.RGB(120, 40, 80)}}
				}
				e.OnError = func(err error) {
					if !errors.Is(err, errLineTooLong) {
						t.Error(err)
					}
					warnings++
					observe()
				}
				e.OnChange = func() { changes++ }
				if err := e.RevealSourceTarget(f.target, f.span.Head, f.span.Anchor); err != nil {
					t.Fatal(err)
				}
				// These checks deliberately precede any toolkit draw or observing
				// geometry helper. A later frame cannot fix a broken commit.
				l, ok := e.cache[f.line]
				if !ok || l.layout == nil || l.start != f.span.Anchor || l.rows < 2 || e.rowCounts[f.line] != l.rows || e.ScrollY <= 0 || e.hasDesiredX {
					t.Fatalf("prepared layout/rows/scroll were not committed: cached=%v layout=%+v scroll=%v", ok, l, e.ScrollY)
				}
				if b.Selection() != (text.Range{Anchor: f.span.Head, Head: f.span.Anchor}) || len(b.SelectionSet().Ranges) != 1 || b.SelectionGeneration() != before.edit.selectionGen+1 {
					t.Fatal("directed single navigation was not published exactly once")
				}
				for _, r := range e.folds.ranges {
					if r.OpenLine < f.line && r.CloseLine > f.line && e.folds.collapsed[r] {
						t.Fatal("target ancestor remained collapsed before Draw")
					}
				}
				if b.Version() != before.edit.version || b.String() != before.edit.source || b.History() != before.edit.history || !b.CanRedo() || changes != 0 || highlights != 1 || warnings != map[bool]int{false: 0, true: 1}[long] {
					t.Fatalf("successful navigation changed content/history or callbacks: highlight=%d warning=%d change=%d", highlights, warnings, changes)
				}
				// Later paint may inspect state but must reuse the committed target
				// geometry. Disable the entry-only observer for unrelated visible lines.
				e.Highlight = nil
				e.OnError = func(error) { warnings++ }
				f.tester.Frame()
				if !f.tester.Focused(e.Label) {
					t.Fatal("successful prepared navigation did not focus the actual control")
				}
				if long && warnings != 1 {
					t.Fatal("committed long-line disclosure was repeated on Draw")
				}
			})
		}
	}
}
