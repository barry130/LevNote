package app

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"unicode/utf16"

	"golang.org/x/text/encoding/simplifiedchinese"
	"levnote.local/native/core/document"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/code"
	"levnote.local/native/ui/editor"
)

func completionUnits(value string) int { return len(utf16.Encode([]rune(value))) }

func completionSelect(t *testing.T, tab *Tab, source string, left, right int, backward bool) {
	t.Helper()
	a, z := completionUnits(source[:left]), completionUnits(source[:right])
	if backward {
		a, z = z, a
	}
	if err := tab.Doc.Buffer.SetSelection(a, z); err != nil {
		t.Fatal(err)
	}
}

func completionFinish(t *testing.T, a *App, queue *languageQueue) {
	t.Helper()
	task := a.sourceLanguage.task
	if task == nil {
		t.Fatal("missing real physical Analyzer task")
	}
	languageWait(t, task.done)
	languageReceive(t, queue.calls)()
}

func completionReady(t *testing.T, a *App, queue *languageQueue) {
	t.Helper()
	if err := a.requestSourceCompletion(); err != nil {
		t.Fatal(err)
	}
	if err := a.openSourceLanguagePresentation(); err != nil {
		t.Fatal(err)
	}
	completionFinish(t, a, queue)
	if a.sourceLanguage.status != string(code.ServiceComplete) || a.sourceLanguage.result.Status != code.ServiceComplete || a.sourceLanguage.intent.completion == nil {
		t.Fatalf("real completion did not publish: status=%s reason=%s message=%s failure=%v", a.sourceLanguage.status, a.sourceLanguage.reason, a.Message, a.sourceLanguage.failure)
	}
}

func completionCandidate(t *testing.T, a *App, value string) string {
	t.Helper()
	rows := a.sourceLanguageRows()
	for i, item := range a.sourceLanguage.result.Completion {
		if item.Text == value {
			if i >= len(rows) || rows[i].kind != "completion" {
				t.Fatal("real candidate not in current bounded display")
			}
			return rows[i].id
		}
	}
	t.Fatalf("real vocabulary lacks %q: %+v", value, a.sourceLanguage.result.Completion)
	return ""
}

func completionConfirm(t *testing.T, a *App, value string) {
	t.Helper()
	id := completionCandidate(t, a, value)
	a.queueSourceLanguageAction("complete", id)
	if a.sourceLanguage.action == nil {
		t.Fatal("no queued completion action")
	}
	a.processSourceLanguageActions()
}

func TestCompletionRealGoVocabularyUnicodeRangesAndSingleUndo(t *testing.T) {
	for _, tc := range []struct {
		name, source, prefix, candidate string
		selected, backward              bool
	}{
		{"word", "package p; func f(val int) { _ = va }", "va", "val", false, false},
		{"unicode", "package p\r\n// 🙂 unchanged\r\nfunc f(变量 int) { _ = 变 }\n", "变", "变量", false, false},
		{"eof-comment", "package p\r\nvar variable = 1\n// va", "va", "variable", false, false},
		{"whole-selected-backward", "package p; var value int; func f(val int) { _ = val }", "val", "value", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, tab, queue, path := languageFixture(t, ".go", tc.source)
			at := strings.LastIndex(tc.source, tc.prefix)
			left := at + len(tc.prefix)
			if tc.selected {
				left = at
			}
			completionSelect(t, tab, tc.source, left, at+len(tc.prefix), tc.backward)
			before := languageState(tab)
			completionReady(t, a, queue)
			languageUnchanged(t, tab, before)
			if tc.name == "word" {
				id := completionCandidate(t, a, "var")
				found := false
				for _, r := range a.sourceLanguageRows() {
					if r.id == id && strings.Contains(r.label, "Go 关键字") {
						found = true
					}
				}
				if !found {
					t.Fatal("keyword kind not truthful")
				}
			}
			calls := 0
			original := tab.Editor.OnChange
			tab.Editor.OnChange = func() {
				calls++
				if original != nil {
					original()
				}
			}
			completionConfirm(t, a, tc.candidate)
			want := tc.source[:at] + tc.candidate + tc.source[at+len(tc.prefix):]
			caret := completionUnits(tc.source[:at] + tc.candidate)
			if tab.Doc.Buffer.String() != want || calls != 1 || tab.Doc.Buffer.History().UndoDepth != before.history.UndoDepth+1 || tab.Doc.Buffer.Selection() != (text.Range{Anchor: caret, Head: caret}) {
				t.Fatal("completion was not one exact patch/caret/notification", tab.Doc.Buffer.String(), calls, tab.Doc.Buffer.History())
			}
			if !tab.Doc.Buffer.Undo() || tab.Doc.Buffer.String() != tc.source || !reflect.DeepEqual(tab.Doc.Buffer.SelectionSet(), before.selections) {
				t.Fatal("whole completion Undo")
			}
			if !tab.Doc.Buffer.Redo() || tab.Doc.Buffer.String() != want || tab.Doc.Buffer.Selection() != (text.Range{Anchor: caret, Head: caret}) {
				t.Fatal("whole completion Redo")
			}
			if got := mustReadFile(t, path); string(got) != tc.source {
				t.Fatal("completion implicitly saved")
			}
			if queue.maximum.Load() != 1 || queue.outstanding.Load() != 0 {
				t.Fatal("second physical task/inline publication")
			}
		})
	}
}

func TestCompletionGoEmptyPrefixAndFieldBoundaryAdmission(t *testing.T) {
	source := "package p; func f() {}"
	a, tab, q, _ := languageFixture(t, ".go", source)
	at := strings.Index(source, "{") + 1
	completionSelect(t, tab, source, at, at, false)
	completionReady(t, a, q)
	if a.sourceLanguage.intent.prefix != "" || a.sourceLanguage.intent.completion.oldText != "" || completionCandidate(t, a, "var") == "" {
		t.Fatal("empty prefix request not real vocabulary")
	}
	a.cancelSourceLanguage()
	for _, size := range []int{sourceLanguageFieldBytes, sourceLanguageFieldBytes + 1} {
		id := strings.Repeat("a", size)
		source = "package p; var " + id + " int"
		a, tab, q, _ = languageFixture(t, ".go", source)
		at = strings.Index(source, id) + len(id)
		completionSelect(t, tab, source, at, at, false)
		before := languageState(tab)
		err := a.requestSourceCompletion()
		if size == sourceLanguageFieldBytes {
			if err != nil {
				t.Fatal("exact bounded prefix refused", err)
			}
			completionFinish(t, a, q)
			if a.sourceLanguage.result.Status != code.ServiceComplete || a.sourceLanguage.intent.completion.oldText != id {
				t.Fatal("exact prefix truncated or syntax not validated")
			}
		} else if err == nil || a.sourceLanguage.task != nil || !strings.Contains(err.Error(), "16 KiB") {
			t.Fatal("+1 prefix suffix silently admitted", err)
		}
		languageUnchanged(t, tab, before)
	}
}

func TestCompletionGoSelectedIdentifierMustBeWholeAndDirected(t *testing.T) {
	source := "package p; var variable int"
	for _, span := range [][2]int{{strings.Index(source, "variable") + 1, strings.Index(source, "variable") + 3}, {0, 9}} {
		a, tab, q, _ := languageFixture(t, ".go", source)
		completionSelect(t, tab, source, span[0], span[1], true)
		before := languageState(tab)
		if err := a.requestSourceCompletion(); err == nil || a.sourceLanguage.task != nil || q.outstanding.Load() != 0 {
			t.Fatal("fragment selection was expanded/truncated")
		}
		languageUnchanged(t, tab, before)
	}
}

func TestCompletionJSONRealRawKeysEscapesLoneSurrogateAndHugeNumber(t *testing.T) {
	const source = `{"a":1,"\u0061":2,"a":3,"q\"uote":4,"\uD800":5,"n":1e99999,"dst":"old"}`
	for _, candidate := range []string{`"a"`, `"\u0061"`, `"q\"uote"`, `"\uD800"`} {
		t.Run(candidate, func(t *testing.T) {
			a, tab, q, _ := languageFixture(t, ".json", source)
			at := strings.Index(source, `"old"`)
			completionSelect(t, tab, source, at, at+len(`"old"`), true)
			before := languageState(tab)
			completionReady(t, a, q)
			if a.sourceLanguage.intent.prefix != "" || !a.sourceLanguage.intent.completion.proven {
				t.Fatal("raw token did not pass structural decoder proof")
			}
			counts := map[string]int{}
			for _, item := range a.sourceLanguage.result.Completion {
				counts[item.Text]++
				if item.Kind != "json-key" {
					t.Fatal("invented JSON kind")
				}
			}
			if counts[`"a"`] != 1 || counts[`"\u0061"`] != 1 || counts[`"\uD800"`] != 1 {
				t.Fatal("raw key spelling was decoded/deduped by value", counts)
			}
			if candidate == `"\uD800"` && (len(candidate) != 8 || len(candidate[1:len(candidate)-1]) != 6 || strings.Count(candidate, `\`) != 1) {
				t.Fatal("lone-surrogate fixture spelling")
			}
			completionConfirm(t, a, candidate)
			want := source[:at] + candidate + source[at+len(`"old"`):]
			if tab.Doc.Buffer.String() != want || !json.Valid([]byte(want)) {
				t.Fatal("raw key insertion added/removed quotes, colon or escapes", tab.Doc.Buffer.String())
			}
			if !tab.Doc.Buffer.Undo() || tab.Doc.Buffer.String() != source || !reflect.DeepEqual(tab.Doc.Buffer.SelectionSet(), before.selections) || !tab.Doc.Buffer.Redo() || tab.Doc.Buffer.String() != want {
				t.Fatal("JSON full Undo/Redo")
			}
		})
	}
}

func TestCompletionJSONBothTokenEndpointsRejectStandaloneFragments(t *testing.T) {
	for _, tc := range []struct {
		source, fragment string
		standalone       bool
	}{
		{`{"a" : "b"}`, `" : "`, true},
		{`{"k":"a\"b"}`, `"b"`, true},
		{`{"a":"old"}`, `old`, false},
		{`{"a":"old"}`, ``, false},
	} {
		a, tab, q, _ := languageFixture(t, ".json", tc.source)
		at := strings.Index(tc.source, tc.fragment)
		if tc.fragment == "" {
			at = 0
		}
		if at < 0 || (tc.standalone && !json.Valid([]byte(tc.fragment))) {
			t.Fatal("counterexample raw bytes are not the declared standalone token")
		}
		if strings.Contains(tc.source, `a\"b`) && strings.Count(tc.source, `\`) != 1 {
			t.Fatal("escaped quote fixture must have one backslash")
		}
		completionSelect(t, tab, tc.source, at, at+len(tc.fragment), false)
		selected, err := tab.Doc.Buffer.SliceUTF16(completionUnits(tc.source[:at]), completionUnits(tc.source[:at+len(tc.fragment)]))
		if err != nil || selected != tc.fragment {
			t.Fatal("counterexample endpoints do not select exact raw fragment")
		}
		before := languageState(tab)
		if err := a.requestSourceCompletion(); err != nil {
			t.Fatal("non-token proof must still run actual source syntax", err)
		}
		completionFinish(t, a, q)
		if a.sourceLanguage.status != sourceLanguageHostError || a.sourceLanguage.failure == nil || !strings.Contains(a.sourceLanguage.failure.Error(), sourceCompletionJSONInstruction) || len(a.sourceLanguage.result.Completion) != 0 || a.sourceLanguage.intent.completion != nil {
			t.Fatal("standalone fragment accepted as source token", a.sourceLanguage)
		}
		languageUnchanged(t, tab, before)
	}
}

func TestCompletionInvalidGoJSONAlwaysShowRealSyntaxDiagnostics(t *testing.T) {
	for _, tc := range []struct{ suffix, source string }{{".go", "package p; func broken("}, {".json", `{"k":"bad\q"}`}} {
		a, tab, q, _ := languageFixture(t, tc.suffix, tc.source)
		at := len(tc.source)
		completionSelect(t, tab, tc.source, at, at, false)
		before := languageState(tab)
		if err := a.requestSourceCompletion(); err != nil {
			t.Fatal(err)
		}
		completionFinish(t, a, q)
		if a.sourceLanguage.result.Status != code.ServiceInvalid || len(a.sourceLanguage.result.Diagnostics) == 0 || a.sourceLanguage.intent.completion != nil || len(a.sourceLanguage.result.Completion) != 0 {
			t.Fatal("invalid source was promoted to empty successful vocabulary", a.sourceLanguage.result)
		}
		languageUnchanged(t, tab, before)
	}
}

func TestCompletionEqualCandidateRequiresAdmissionAndPreservesRedo(t *testing.T) {
	const source = "package p; func f(val int) { _ = val }"
	a, tab, q, _ := languageFixture(t, ".go", source)
	b := tab.Doc.Buffer
	if err := b.ReplaceUTF16(0, 0, " "); err != nil || !b.Undo() {
		t.Fatal("redo fixture", err)
	}
	at := strings.LastIndex(source, "val")
	completionSelect(t, tab, source, at, at+3, true)
	before := languageState(tab)
	completionReady(t, a, q)
	calls := 0
	tab.Editor.OnChange = func() { calls++ }
	completionConfirm(t, a, "val")
	languageUnchanged(t, tab, before)
	if calls != 0 || !b.CanRedo() || a.sourceLanguage.visible || a.sourceLanguage.intent.completion != nil {
		t.Fatal("equal completion changed selection/history or retained consumed payload")
	}
	if !b.Redo() || b.String() != " "+source {
		t.Fatal("equal candidate erased redo")
	}
}

func TestCompletionSelectionTargetAndReadonlyChangesRefuseConfirmation(t *testing.T) {
	const source = "package p; func f(val int) { _ = va }"
	for _, kind := range []string{"selection", "selection-ABA", "secondary-ABA", "direction-ABA", "undo-same-bytes", "buffer", "editor", "readonly-doc", "readonly-editor-ABA", "route", "mode", "session", "tab"} {
		t.Run(kind, func(t *testing.T) {
			a, tab, q, _ := languageFixture(t, ".go", source)
			at := strings.LastIndex(source, "va") + 2
			completionSelect(t, tab, source, at, at, false)
			completionReady(t, a, q)
			id := completionCandidate(t, a, "val")
			old := tab.Doc.Buffer
			selected := old.Selection()
			switch kind {
			case "selection":
				_ = old.SetSelection(0, 0)
			case "selection-ABA":
				_ = old.SetSelection(0, 0)
				_ = old.SetSelection(selected.Anchor, selected.Head)
			case "secondary-ABA":
				_ = old.SetSelectionSet(text.SelectionSet{Ranges: []text.Range{{Anchor: 0, Head: 0}, selected}, Primary: 1})
				_ = old.SetSelection(selected.Anchor, selected.Head)
			case "direction-ABA":
				_ = old.SetSelection(at-2, at)
				_ = old.SetSelection(at, at-2)
				_ = old.SetSelection(selected.Anchor, selected.Head)
			case "undo-same-bytes":
				_ = old.ReplaceUTF16(0, 0, " ")
				old.Undo()
				_ = old.SetSelection(selected.Anchor, selected.Head)
			case "buffer":
				next := text.New(source)
				tab.Doc.Buffer = next
				tab.Editor.SetBuffer(next)
			case "editor":
				tab.Editor = editor.New(old)
			case "readonly-doc":
				tab.Doc.ReadOnly = true
			case "readonly-editor-ABA":
				tab.Editor.SetReadOnly(true)
				tab.Editor.SetReadOnly(false)
			case "route":
				tab.sourceLanguageRoute++
			case "mode":
				tab.Mode = "preview"
			case "session":
				a.sourceLanguage.session++
			case "tab":
				a.NewDocument("text")
			}
			state := languageState(tab)
			a.queueSourceLanguageAction("complete", id)
			a.processSourceLanguageActions()
			languageUnchanged(t, tab, state)
			if old.String() != source {
				t.Fatal("stale completion edited original Buffer")
			}
		})
	}
}

func TestCompletionPendingUsesOriginalSelectionAndOnePhysicalSlot(t *testing.T) {
	const source = "package p; func f(val int) { _ = va }"
	for _, changed := range []bool{false, true} {
		a, tab, q, _ := languageFixture(t, ".go", source)
		at := strings.LastIndex(source, "va") + 2
		completionSelect(t, tab, source, at, at, false)
		started, release := languageHold(t, a)
		var active, maximum atomic.Int32
		beforeAnalyze := a.sourceLanguageHooks.beforeAnalyze
		a.sourceLanguageHooks.beforeAnalyze = func(ctx context.Context, id uint64) { languageMax(&maximum, active.Add(1)); beforeAnalyze(ctx, id) }
		a.sourceLanguageHooks.afterAnalyze = func(uint64) { active.Add(-1) }
		first := languageRequest(t, a, languageIntent(code.ServiceDiagnostics))
		languageReceive(t, started)
		if changed {
			_ = tab.Doc.Buffer.SetSelection(0, 0)
			_ = tab.Doc.Buffer.SetSelection(completionUnits(source[:at]), completionUnits(source[:at]))
		}
		err := a.requestSourceCompletion()
		if changed {
			if err == nil || a.sourceLanguage.pending != nil || a.sourceLanguage.task != first {
				t.Fatal("busy completion rebound target after selection ABA")
			}
		} else if err != nil || a.sourceLanguage.task != first || a.sourceLanguage.pending == nil || a.sourceLanguage.pending.intent.completion == nil {
			t.Fatal("completion did not coalesce bounded original descriptor", err)
		}
		release()
		languageWait(t, first.done)
		languageReceive(t, q.calls)()
		if !changed {
			second := a.sourceLanguage.task
			if second == nil || second == first {
				t.Fatal("pending original completion did not launch after physical retirement")
			}
			languageWait(t, second.done)
			languageReceive(t, q.calls)()
			if a.sourceLanguage.result.Status != code.ServiceComplete || a.sourceLanguage.intent.completion == nil {
				t.Fatal("queued completion lost real result/intent")
			}
		} else if a.sourceLanguage.task != nil || a.sourceLanguage.intent.completion != nil {
			t.Fatal("refused busy result revived")
		}
		if q.maximum.Load() != 1 || q.outstanding.Load() != 0 || maximum.Load() != 1 || active.Load() != 0 {
			t.Fatal("physical analyzer/queued publication overlap")
		}
	}
}

func TestCompletionSelectionChangesBeforeQueuedPublicationCannotRevive(t *testing.T) {
	const source = "package p; func f(val int) { _ = va }"
	for _, kind := range []string{"selection", "secondary-ABA", "readonly", "route", "cancel"} {
		a, tab, q, _ := languageFixture(t, ".go", source)
		at := strings.LastIndex(source, "va") + 2
		completionSelect(t, tab, source, at, at, false)
		if err := a.requestSourceCompletion(); err != nil {
			t.Fatal(err)
		}
		task := a.sourceLanguage.task
		languageWait(t, task.done)
		callback := languageReceive(t, q.calls)
		b := tab.Doc.Buffer
		original := b.Selection()
		switch kind {
		case "selection":
			_ = b.SetSelection(0, 0)
		case "secondary-ABA":
			_ = b.SetSelectionSet(text.SelectionSet{Ranges: []text.Range{{Anchor: 0, Head: 0}, original}, Primary: 1})
			_ = b.SetSelection(original.Anchor, original.Head)
		case "readonly":
			tab.Doc.ReadOnly = true
		case "route":
			tab.sourceLanguageRoute++
		case "cancel":
			a.cancelSourceLanguage()
		}
		before := languageState(tab)
		callback()
		callback()
		if a.sourceLanguage.intent.completion != nil || len(a.sourceLanguage.result.Completion) != 0 || a.sourceLanguage.owner != nil {
			t.Fatal("stale/canceled result remained applicable", kind)
		}
		languageUnchanged(t, tab, before)
	}
}

func TestCompletionPublicationRetainsActualDispatcherAndSourceMaterialization(t *testing.T) {
	const source = "package p; func f(val int) { _ = va }"
	a, tab, q, _ := languageFixture(t, ".go", source)
	at := strings.LastIndex(source, "va") + 2
	completionSelect(t, tab, source, at, at, false)
	start := make(chan struct{}, 1)
	release := make(chan struct{})
	var once sync.Once
	finish := func() { once.Do(func() { close(release) }) }
	t.Cleanup(finish)
	a.sourceLanguageHooks.beforeAnalyze = func(context.Context, uint64) { start <- struct{}{}; <-release }
	if err := a.requestSourceCompletion(); err != nil {
		t.Fatal(err)
	}
	task := a.sourceLanguage.task
	languageReceive(t, start)
	// A newer hook dispatcher cannot intercept the old admitted UI queue.
	a.sourceLanguageHooks.dispatch = func(func()) { t.Error("worker rebound to newer dispatcher") }
	finish()
	languageWait(t, task.done)
	if a.sourceLanguage.status != sourceLanguageRunning || q.outstanding.Load() != 1 {
		t.Fatal("worker mutated Host before queued UI delivery")
	}
	languageReceive(t, q.calls)()
	if a.sourceLanguage.result.Status != code.ServiceComplete || a.sourceLanguage.intent.completion.oldText != "va" {
		t.Fatal("single-source completion not published")
	}
}

func TestCompletionSaveAsRetiresOriginalEditingTargetBeforePublicationOrConfirm(t *testing.T) {
	const source = "package p; func f(val int) { _ = va }"
	for _, published := range []bool{false, true} {
		a, tab, q, _ := languageFixture(t, ".go", source)
		at := strings.LastIndex(source, "va") + 2
		completionSelect(t, tab, source, at, at, false)
		if err := a.requestSourceCompletion(); err != nil {
			t.Fatal(err)
		}
		if err := a.openSourceLanguagePresentation(); err != nil {
			t.Fatal(err)
		}
		task := a.sourceLanguage.task
		languageWait(t, task.done)
		callback := languageReceive(t, q.calls)
		var old sourceLanguageAction
		if published {
			callback()
			id := completionCandidate(t, a, "val")
			old = sourceLanguageAction{kind: "complete", rowID: id, request: a.sourceLanguage.request, presentation: a.sourceLanguage.presentation}
		}
		destination := filepath.Join(t.TempDir(), "renamed.go")
		a.saveDialog = func(*Tab) (string, error) { return destination, nil }
		if !a.Save(tab, true) || tab.Doc.Path != destination || string(mustReadFile(t, destination)) != source {
			t.Fatal("real SaveAs fixture failed", a.Message)
		}
		before := languageState(tab)
		if !published {
			callback()
		} else {
			a.sourceLanguage.action = &old
			a.processSourceLanguageActions()
		}
		languageUnchanged(t, tab, before)
		if a.sourceLanguage.intent.completion != nil || a.sourceLanguage.owner != nil || len(a.sourceLanguage.result.Completion) != 0 {
			t.Fatal("SaveAs left old completion applicable")
		}
	}
}

func TestCompletionLaterExplicitSavePreservesBOMEncodingAndFailure(t *testing.T) {
	for _, encoding := range []document.Encoding{document.UTF8, document.GBK} {
		source := "package p\r\nfunc f(变量 int) { _ = 变 }\n"
		root := t.TempDir()
		path := filepath.Join(root, "completion.go")
		raw := append([]byte{0xef, 0xbb, 0xbf}, []byte(source)...)
		if encoding == document.GBK {
			var err error
			raw, err = simplifiedchinese.GBK.NewEncoder().Bytes([]byte(source))
			if err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
		a := newTestApp(t, filepath.Join(root, "config"))
		a.Open(path)
		if encoding == document.GBK {
			if !a.reopenWithEncoding(a.Current(), document.GBK) {
				t.Fatal(a.Message)
			}
		}
		tab := a.Current()
		q := newLanguageQueue()
		a.sourceLanguageHooks = &sourceLanguageHooks{dispatch: q.dispatch}
		at := strings.LastIndex(source, "变") + len("变")
		completionSelect(t, tab, source, at, at, false)
		completionReady(t, a, q)
		completionConfirm(t, a, "变量")
		want := strings.Replace(source, "_ = 变 }", "_ = 变量 }", 1)
		if tab.Doc.Buffer.String() != want || tab.Doc.Encoding != encoding || tab.Doc.BOM != (encoding == document.UTF8) || !tab.Doc.Dirty() || !bytes.Equal(mustReadFile(t, path), raw) {
			t.Fatal("completion altered encoding/BOM or saved early")
		}
		if !a.Save(tab, false) {
			t.Fatal("later explicit save", a.Message)
		}
		expected := append([]byte{0xef, 0xbb, 0xbf}, []byte(want)...)
		if encoding == document.GBK {
			var err error
			expected, err = simplifiedchinese.GBK.NewEncoder().Bytes([]byte(want))
			if err != nil {
				t.Fatal(err)
			}
		}
		if !bytes.Equal(mustReadFile(t, path), expected) || tab.Doc.Dirty() {
			t.Fatal("explicit save rewrote source/EOL/encoding")
		}
		if encoding == document.GBK {
			unrepresentable := "package p; var 𐐀x int; func f(){ _ = 𐐀 }"
			if err := tab.Doc.Buffer.ReplaceUTF16(0, tab.Doc.Buffer.LenUTF16(), unrepresentable); err != nil {
				t.Fatal(err)
			}
			at = strings.LastIndex(unrepresentable, "𐐀") + len("𐐀")
			completionSelect(t, tab, unrepresentable, at, at, false)
			completionReady(t, a, q)
			completionConfirm(t, a, "𐐀x")
			if a.Save(tab, false) || !strings.Contains(a.Message, document.ErrUnrepresentableEncoding.Error()) || !bytes.Equal(mustReadFile(t, path), expected) {
				t.Fatal("unrepresentable completion silently transcoded/saved", a.Message)
			}
		}
	}
}
