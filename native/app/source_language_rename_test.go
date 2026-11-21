package app

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/egoist/mygo/ui"
	"golang.org/x/text/encoding/simplifiedchinese"
	"levnote.local/native/core/document"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/code"
)

func renameUnits(s string) int { return len(utf16.Encode([]rune(s))) }

func renamePosition(t *testing.T, tab *Tab, source string, at int) {
	t.Helper()
	unit := renameUnits(source[:at])
	if err := tab.Doc.Buffer.SetSelection(unit, unit); err != nil {
		t.Fatal(err)
	}
}

func renameName(t *testing.T, a *App, name string) {
	t.Helper()
	d := a.sourceLanguage.rename
	if d == nil || d.field == nil {
		t.Fatal("missing native name field")
	}
	d.field.ReplaceText(&ui.TextInputRange{Start: 0, End: d.buffer.LenUTF16()}, name)
	if d.buffer.String() != name {
		t.Fatalf("native name replacement refused: %q -> %q", d.buffer.String(), name)
	}
}

func renameFinish(t *testing.T, a *App, q *languageQueue) {
	t.Helper()
	task := a.sourceLanguage.task
	if task == nil {
		t.Fatal("missing actual Rename Analyzer task")
	}
	languageWait(t, task.done)
	languageReceive(t, q.calls)()
}

func renamePreview(t *testing.T, a *App, q *languageQueue, name string) {
	t.Helper()
	renameName(t, a, name)
	if err := a.requestSourceRename(); err != nil {
		t.Fatal(err)
	}
	renameFinish(t, a, q)
}

func renameReady(t *testing.T, a *App, q *languageQueue, name string) {
	t.Helper()
	if err := a.openSourceRename(); err != nil {
		t.Fatal(err)
	}
	renamePreview(t, a, q, name)
	if a.sourceLanguage.result.Status != code.ServiceComplete || a.sourceLanguage.result.NoOp || len(a.sourceLanguage.rename.patch) == 0 {
		t.Fatalf("real local Rename failed: status=%s reason=%s failure=%v message=%s", a.sourceLanguage.status, a.sourceLanguage.reason, a.sourceLanguage.failure, a.sourceLanguage.message)
	}
}

func renameConfirm(t *testing.T, a *App) {
	t.Helper()
	a.queueSourceLanguageAction("rename-confirm", "")
	if a.sourceLanguage.action == nil {
		t.Fatal("missing sealed queued rename confirmation")
	}
	a.processSourceLanguageActions()
}

func TestRenameLocalBindingPreservesShadowCommentStringAndSelector(t *testing.T) {
	const source = "package p\r\n// 🙂 x stays\r\nfunc f(x int, obj struct{x int}) int {\n  _ = \"x\"; _ = obj.x\n  { x := 9; _ = x }\r\n  return x\n}\n"
	const want = "package p\r\n// 🙂 x stays\r\nfunc f(结果 int, obj struct{x int}) int {\n  _ = \"x\"; _ = obj.x\n  { x := 9; _ = x }\r\n  return 结果\n}\n"
	a, tab, q, path := languageFixture(t, ".go", source)
	at := strings.Index(source, "x int")
	set := text.SelectionSet{Ranges: []text.Range{{Anchor: renameUnits(source[:at]) + 1, Head: renameUnits(source[:at])}, {Anchor: 0, Head: 0}, {Anchor: renameUnits(source), Head: renameUnits(source)}}, Primary: 0}
	if err := tab.Doc.Buffer.SetSelectionSet(set); err != nil {
		t.Fatal(err)
	}
	before := languageState(tab)
	renameReady(t, a, q, "结果")
	languageUnchanged(t, tab, before)
	rows := a.sourceLanguageRows()
	if len(rows) != 2 || len(a.sourceLanguage.result.Edits) != 2 {
		t.Fatal("rename fell back to spelling replacement", rows)
	}
	if rows[0].position.line != 3 || rows[0].position.column != 8 || rows[1].position.line != 6 || rows[1].position.column != 10 {
		t.Fatal("physical line / UTF-16 columns", rows)
	}
	if !strings.Contains(rows[0].label, "x → 结果") {
		t.Fatal("preview omits actual spellings", rows)
	}
	var changes int
	original := tab.Editor.OnChange
	tab.Editor.OnChange = func() {
		changes++
		if original != nil {
			original()
		}
	}
	renameConfirm(t, a)
	if tab.Doc.Buffer.String() != want || changes != 1 || !tab.Doc.Dirty() || string(mustReadFile(t, path)) != source {
		t.Fatal("whole native commit/dirty/no-auto-save", changes, tab.Doc.Buffer.String())
	}
	after := tab.Doc.Buffer.SelectionSet()
	if len(after.Ranges) != 3 || after.Primary != before.selections.Primary || after.Ranges[after.Primary].Anchor <= after.Ranges[after.Primary].Head {
		t.Fatal("directed source selections lost", after)
	}
	if !tab.Doc.Buffer.Undo() || tab.Doc.Buffer.String() != source || !reflect.DeepEqual(tab.Doc.Buffer.SelectionSet(), before.selections) || tab.Doc.Dirty() {
		t.Fatal("one Undo did not restore exact source/selection/clean state")
	}
	if !tab.Doc.Buffer.Redo() || tab.Doc.Buffer.String() != want || !reflect.DeepEqual(tab.Doc.Buffer.SelectionSet(), after) {
		t.Fatal("one Redo did not restore whole rename")
	}
}

func TestRename1024FullPreviewAnd256SelectionsVersus1025AtomicLimit(t *testing.T) {
	for _, edits := range []int{1024, 1025} {
		t.Run(string(rune('A'+edits-1024)), func(t *testing.T) {
			source := "package p\nfunc f(){ x := 0\n" + strings.Repeat("_ = x\n", edits-1) + "}\n"
			a, tab, q, path := languageFixture(t, ".go", source)
			declaration := strings.Index(source, "x :=")
			set := text.SelectionSet{Ranges: []text.Range{{Anchor: renameUnits(source[:declaration]) + 1, Head: renameUnits(source[:declaration])}}, Primary: 0}
			cursor := declaration + 1
			for i := 1; i < 256; i++ {
				relative := strings.Index(source[cursor:], "_ = x")
				if relative < 0 {
					t.Fatal("fixture")
				}
				at := cursor + relative + len("_ = x")
				set.Ranges = append(set.Ranges, text.Range{Anchor: renameUnits(source[:at]), Head: renameUnits(source[:at])})
				cursor = at
			}
			if err := tab.Doc.Buffer.SetSelectionSet(set); err != nil {
				t.Fatal(err)
			}
			before := languageState(tab)
			if err := a.openSourceRename(); err != nil {
				t.Fatal(err)
			}
			renamePreview(t, a, q, "longName")
			languageUnchanged(t, tab, before)
			if edits == 1025 {
				if a.sourceLanguage.result.Status != code.ServiceLimit || a.sourceLanguage.result.Resource != "rename-edits" || a.sourceLanguage.result.Limit != 1024 || len(a.sourceLanguage.result.Edits) != 0 || len(a.sourceLanguage.rename.patch) != 0 || len(a.sourceLanguageRows()) != 0 {
					t.Fatal("1025 retained partial result", a.sourceLanguage.result)
				}
				a.queueSourceLanguageAction("rename-confirm", "")
				a.processSourceLanguageActions()
				languageUnchanged(t, tab, before)
				return
			}
			if len(a.sourceLanguage.result.Edits) != 1024 || len(a.sourceLanguage.rename.patch) != 1024 || len(a.sourceLanguageRows()) != 1024 {
				t.Fatal("1024 capped to display or selection count")
			}
			want := strings.ReplaceAll(source, "x", "longName") // fixture has no other x
			renameConfirm(t, a)
			if tab.Doc.Buffer.String() != want || string(mustReadFile(t, path)) != source || tab.Doc.Buffer.History().UndoDepth != before.history.UndoDepth+1 {
				t.Fatal("1024 not one whole unsaved history patch")
			}
			after := tab.Doc.Buffer.SelectionSet()
			if len(after.Ranges) != 256 || after.Primary != 0 {
				t.Fatal("rename manufactured carets", after)
			}
			for i, r := range after.Ranges {
				old := before.selections.Ranges[i]
				wantAnchor := old.Anchor + 7*(i+1)
				wantHead := old.Head + 7*(i+1)
				if i == 0 {
					wantHead = old.Head
				}
				if r.Anchor != wantAnchor || r.Head != wantHead {
					t.Fatalf("independent selection map %d: %+v want %d,%d", i, r, wantAnchor, wantHead)
				}
			}
			if !tab.Doc.Buffer.Undo() || tab.Doc.Buffer.String() != source || !reflect.DeepEqual(tab.Doc.Buffer.SelectionSet(), before.selections) || !tab.Doc.Buffer.Redo() || tab.Doc.Buffer.String() != want || !reflect.DeepEqual(tab.Doc.Buffer.SelectionSet(), after) {
				t.Fatal("1024/256 whole Undo/Redo")
			}
		})
	}
}

func TestRenameServiceNameSyntaxConflictPackageAndSelectorOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name, source, target, newName string
		status                        code.ServiceStatus
		reason                        string
	}{
		{"empty", "package p; func f(x int){ _ = x }", "x int", "", code.ServiceInvalid, "invalid-new-name"},
		{"keyword", "package p; func f(x int){ _ = x }", "x int", "for", code.ServiceInvalid, "invalid-new-name"},
		{"underscore", "package p; func f(x int){ _ = x }", "x int", "_", code.ServiceInvalid, "invalid-new-name"},
		{"space", "package p; func f(x int){ _ = x }", "x int", " x ", code.ServiceInvalid, "invalid-new-name"},
		{"newline", "package p; func f(x int){ _ = x }", "x int", "new\nname", code.ServiceInvalid, "invalid-new-name"},
		{"conflict", "package p; func f(x,y int){ _ = x; _ = y }", "x,y", "y", code.ServiceAmbiguous, "rename-declaration-conflict"},
		{"capture", "package p; func f(x int){ { y:=1; _=y; _=x } }", "x int", "y", code.ServiceAmbiguous, "rename-captured-use"},
		{"captures-use", "package p; var y int; func f(x int){ _=y; _=x }", "x int", "y", code.ServiceAmbiguous, "rename-captures-use"},
		{"syntax", "package p; func f(x int){ _ = x ", "x int", "next", code.ServiceInvalid, "source"},
		{"package", "package p; var x int; func f(){ _ = x }", "x int", "next", code.ServiceUnsupported, "rename-local-only"},
		// Import/package names are not resolved outside the supplied file.
		// The binder classifies the explicit alias as unresolved/external before
		// rename's narrower package-scope local-only check can run.
		{"import", "package p; import x \"fmt\"; func f(){ x.Println() }", "x \"fmt\"", "next", code.ServiceUnsupported, "unresolved-or-external"},
		{"selector", "package p; func f(obj struct{x int}){ _ = obj.x }", "x }", "next", code.ServiceUnsupported, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, tab, q, _ := languageFixture(t, ".go", tc.source)
			at := strings.Index(tc.source, tc.target)
			if at < 0 {
				t.Fatal("fixture target")
			}
			renamePosition(t, tab, tc.source, at)
			before := languageState(tab)
			if err := a.openSourceRename(); err != nil {
				t.Fatal(err)
			}
			renamePreview(t, a, q, tc.newName)
			result := a.sourceLanguage.result
			if result.Status != tc.status || tc.reason != "" && result.Reason != tc.reason || len(result.Edits) != 0 || len(a.sourceLanguage.rename.patch) != 0 {
				t.Fatalf("actual service outcome: %+v, host %s/%s %v", result, a.sourceLanguage.status, a.sourceLanguage.reason, a.sourceLanguage.failure)
			}
			a.queueSourceLanguageAction("rename-confirm", "")
			a.processSourceLanguageActions()
			languageUnchanged(t, tab, before)
		})
	}
}

func TestRenameUnicodeExactNameLimitNoTrimAndSourceBudget(t *testing.T) {
	const source = "package p; func f(x int){ _=x }"
	for _, tc := range []struct {
		name         string
		bytes, units int
	}{
		{"变量", len("变量"), 2},
		{strings.Repeat("a", 128), 128, 128},
		// U+10400 is four UTF-8 bytes and two UTF-16 units per rune.
		{strings.Repeat("𐐀", 64), 256, 128},
		// U+4E2D is three UTF-8 bytes and one UTF-16 unit per rune.
		{strings.Repeat("中", 128), 384, 128},
	} {
		name := tc.name
		if len(name) != tc.bytes || renameUnits(name) != tc.units {
			t.Fatalf("fixture encoding: %d UTF-8 bytes / %d UTF-16 units", len(name), renameUnits(name))
		}
		a, tab, q, _ := languageFixture(t, ".go", source)
		renamePosition(t, tab, source, strings.Index(source, "x int"))
		renameReady(t, a, q, name)
		if a.sourceLanguage.intent.newName != name || a.sourceLanguage.rename.request.name != name {
			t.Fatal("name rewritten")
		}
		renameConfirm(t, a)
		want := strings.ReplaceAll(source, "x", name)
		if tab.Doc.Buffer.String() != want {
			t.Fatal("Unicode byte/unit boundary name changed", tab.Doc.Buffer.String())
		}
	}
	a, tab, q, _ := languageFixture(t, ".go", source)
	renamePosition(t, tab, source, strings.Index(source, "x int"))
	if err := a.openSourceRename(); err != nil {
		t.Fatal(err)
	}
	renameName(t, a, strings.Repeat("a", 129))
	before := languageState(tab)
	if err := a.requestSourceRename(); err == nil || a.sourceLanguage.task != nil || q.outstanding.Load() != 0 {
		t.Fatal("129-unit name analyzed/truncated")
	}
	languageUnchanged(t, tab, before)
	d := a.sourceLanguage.rename
	value := d.buffer.String()
	d.field.ReplaceText(&ui.TextInputRange{Start: 0, End: d.buffer.LenUTF16()}, strings.Repeat("a", 513))
	if d.buffer.String() != value || a.sourceLanguage.task != nil {
		t.Fatal("513-byte programmatic field replacement was partial or analyzed")
	}
	oversize := "package p\n//" + strings.Repeat("a", code.DefaultServiceLimits().Structure.Bytes) + "\n"
	a, tab, q, _ = languageFixture(t, ".go", oversize)
	before = languageState(tab)
	if err := a.openSourceRename(); err == nil || a.sourceLanguage.rename != nil || a.sourceLanguage.task != nil || q.outstanding.Load() != 0 {
		t.Fatal("over-budget source opened/analyzed")
	}
	languageUnchanged(t, tab, before)
}

func TestRenameSameNameNoOpPreservesRedoAndNoApplyAction(t *testing.T) {
	const source = "package p; func f(x int){ _=x }"
	a, tab, q, _ := languageFixture(t, ".go", source)
	if err := tab.Doc.Buffer.ReplaceUTF16(0, 0, " "); err != nil {
		t.Fatal(err)
	}
	if !tab.Doc.Buffer.Undo() {
		t.Fatal("redo fixture")
	}
	renamePosition(t, tab, source, strings.Index(source, "x int"))
	before := languageState(tab)
	if err := a.openSourceRename(); err != nil {
		t.Fatal(err)
	}
	renamePreview(t, a, q, "x")
	if a.sourceLanguage.result.Status != code.ServiceComplete || !a.sourceLanguage.result.NoOp || a.sourceLanguage.message != "名称未改变" || len(a.sourceLanguage.rename.patch) != 0 || len(a.sourceLanguageRows()) != 0 {
		t.Fatal("genuine same-name NoOp lost", a.sourceLanguage.result)
	}
	a.queueSourceLanguageAction("rename-confirm", "")
	a.processSourceLanguageActions()
	languageUnchanged(t, tab, before)
	if !tab.Doc.Buffer.Redo() || tab.Doc.Buffer.String() != " "+source {
		t.Fatal("same-name preview consumed redo")
	}
}

func TestRenameLaterExplicitSaveBOMUTF16LEGBKAndUnrepresentableFailure(t *testing.T) {
	const source = "package p\r\nfunc f(变量 int){ _=变量 }\n"
	const want = "package p\r\nfunc f(结果 int){ _=结果 }\n"
	encode := func(t *testing.T, s string, encoding document.Encoding) []byte {
		t.Helper()
		switch encoding {
		case document.UTF8:
			return append([]byte{0xef, 0xbb, 0xbf}, []byte(s)...)
		case document.GBK:
			b, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(s))
			if err != nil {
				t.Fatal(err)
			}
			return b
		default:
			b := []byte{0xff, 0xfe}
			for _, u := range utf16.Encode([]rune(s)) {
				b = append(b, byte(u), byte(u>>8))
			}
			return b
		}
	}
	for _, encoding := range []document.Encoding{document.UTF8, document.UTF16LE, document.GBK} {
		root := t.TempDir()
		path := filepath.Join(root, "rename.go")
		raw := encode(t, source, encoding)
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
		a := newTestApp(t, filepath.Join(root, "config"))
		a.Open(path)
		if encoding == document.GBK && !a.reopenWithEncoding(a.Current(), document.GBK) {
			t.Fatal(a.Message)
		}
		tab := a.Current()
		q := newLanguageQueue()
		a.sourceLanguageHooks = &sourceLanguageHooks{dispatch: q.dispatch}
		if tab.Doc.Buffer.String() != source || tab.Doc.Encoding != encoding {
			t.Fatal("real encoding open", tab.Doc.Encoding)
		}
		renamePosition(t, tab, source, strings.Index(source, "变量"))
		renameReady(t, a, q, "结果")
		renameConfirm(t, a)
		if tab.Doc.Buffer.String() != want || tab.Doc.BOM != (encoding != document.GBK) || tab.Doc.Encoding != encoding || !tab.Doc.Dirty() || !bytes.Equal(mustReadFile(t, path), raw) {
			t.Fatal("rename changed metadata or saved early")
		}
		if !a.Save(tab, false) || !bytes.Equal(mustReadFile(t, path), encode(t, want, encoding)) || tab.Doc.Dirty() {
			t.Fatal("later explicit save fidelity", a.Message)
		}
		if encoding == document.GBK {
			renamePosition(t, tab, want, strings.Index(want, "结果"))
			renameReady(t, a, q, "𐐀name")
			renameConfirm(t, a)
			beforeSave := mustReadFile(t, path)
			if a.Save(tab, false) || !tab.Doc.Dirty() || !bytes.Equal(mustReadFile(t, path), beforeSave) || a.Message == "" {
				t.Fatal("unrepresentable GBK silently saved/lost bytes", a.Message)
			}
		}
	}
}
