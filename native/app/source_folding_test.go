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
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

const foldGoFixture = "package p\n" +
	"// 🙂 source prefix makes byte and UTF-16 offsets differ\n" +
	"func f() {\n" +
	"    _ = \"{\"\n" +
	"    // }\n" +
	"    if true {\n" +
	"        println(\"🙂\")\n" +
	"    }\n" +
	"}\n"

const foldJSONFixture = "{\n" +
	"  \"items\": [\n" +
	"    1,\n" +
	"    {\n" +
	"      \"text\": \"}\"\n" +
	"    }\n" +
	"  ]\n" +
	"}\n"

func openSourceFoldFixture(t *testing.T, suffix, source string) (*App, *Tab, *ui.Tester, string) {
	t.Helper()
	root := t.TempDir()
	a := newTestApp(t, filepath.Join(root, "config"))
	path := filepath.Join(root, "sample"+suffix)
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	a.Open(path)
	tab := a.Current()
	if tab == nil {
		t.Fatalf("document did not open: %s", a.Message)
	}
	tester := ui.NewTester(a.View, 900, 620)
	tester.Frame()
	return a, tab, tester, path
}

func foldPaletteCommandEnabled(t *testing.T, a *App, tester *ui.Tester, title string) bool {
	t.Helper()
	paletteOpen(t, a, tester)
	tester.Type(title)
	tester.Frame()
	info, ok := tester.AccessByRole(ui.RoleButton, "执行命令：编辑："+title)
	if !ok {
		t.Fatalf("fold command missing from palette: %q; texts=%v", title, tester.Texts())
	}
	return !info.Disabled
}

type foldSplitTestView struct{}

func (foldSplitTestView) Draw(*ui.Context)              {}
func (foldSplitTestView) Release()                      {}
func (foldSplitTestView) Warnings() []string            { return nil }
func (foldSplitTestView) Export(string) ([]byte, error) { return nil, nil }

func assertFoldDocumentUnchanged(t *testing.T, tab *Tab, snapshot text.Snapshot, version uint64, history text.HistoryStatus, selection text.Range, changes int) {
	t.Helper()
	buffer := tab.Doc.Buffer
	if !buffer.EqualSnapshot(snapshot) || buffer.Version() != version || buffer.History() != history ||
		buffer.Selection() != selection || changes != 0 {
		t.Fatalf("fold action changed document: textEqual=%v version=%d/%d history=%+v/%+v selection=%+v/%+v changes=%d",
			buffer.EqualSnapshot(snapshot), buffer.Version(), version, buffer.History(), history,
			buffer.Selection(), selection, changes)
	}
}

func TestBuiltinSourceFoldProviderUsesGoAndStrictJSONStructure(t *testing.T) {
	for _, tc := range []struct {
		name, suffix, source string
		openToken            string
		wantFolds            int
	}{
		{"go", ".go", foldGoFixture, "func f() {", 2},
		{"strict-json", ".json", foldJSONFixture, "{", 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, tab, tester, _ := openSourceFoldFixture(t, tc.suffix, tc.source)
			if tab.MatchedPlugin.ID != "code" || !tab.MatchedPlugin.Builtin || tab.Adapter != nil || tab.Mode != "source" {
				t.Fatalf("fixture did not use builtin source profile: id=%q builtin=%v adapter=%v mode=%q",
					tab.MatchedPlugin.ID, tab.MatchedPlugin.Builtin, tab.Adapter, tab.Mode)
			}
			candidates := tab.Editor.FoldCandidates()
			if len(candidates) != tc.wantFolds || !tester.HasText("折叠当前块") {
				t.Fatalf("fold provider not installed or delimiter count wrong: folds=%d texts=%v", len(candidates), tester.Texts())
			}
			byteStart := strings.Index(tc.source, tc.openToken)
			if byteStart < 0 {
				t.Fatal("fixture opening delimiter missing")
			}
			openByte := byteStart + len(tc.openToken) - 1
			openLine := strings.Count(tc.source[:openByte], "\n")
			wantStart := text.UTF16Len(tc.source[:openByte])
			found := false
			for _, candidate := range candidates {
				if candidate.Range.OpenLine == openLine {
					found = true
					if candidate.Range.OpenStart != wantStart {
						t.Fatalf("delimiter UTF-16 offset=%d want %d", candidate.Range.OpenStart, wantStart)
					}
				}
			}
			if !found {
				t.Fatalf("no fold candidate for expected opener on line %d: %+v", openLine, candidates)
			}
			if tab.Doc.Buffer.LenBytes() > editor.MaxFoldBytes {
				t.Fatal("fixture unexpectedly exceeds provider budget")
			}
			if a.Current() != tab {
				t.Fatal("app lost actual active source owner")
			}
		})
	}
}

func TestSourceFoldFailureDiagnosticsAndUnsupportedProfilesStayVisible(t *testing.T) {
	t.Run("invalid-go-clears-all-folds-and-shows-status", func(t *testing.T) {
		source := "package p\nfunc f() {\n if true {\n  println(1)\n}\n"
		a, tab, tester, _ := openSourceFoldFixture(t, ".go", source)
		if len(tab.Editor.FoldCandidates()) != 0 || !strings.Contains(strings.Join(a.Issues, "\n"), "诊断") ||
			!tester.HasText("折叠分析") || !tester.HasText("暂时显示全文") {
			t.Fatalf("invalid Go structure was not rejected visibly: candidates=%v message=%q texts=%v",
				tab.Editor.FoldCandidates(), a.Message, tester.Texts())
		}
	})
	t.Run("strict-json-comments-are-diagnostics", func(t *testing.T) {
		source := "{\n // JSONC comment\n \"x\": [\n  1\n ]\n}\n"
		a, tab, tester, _ := openSourceFoldFixture(t, ".json", source)
		if len(tab.Editor.FoldCandidates()) != 0 || !strings.Contains(strings.Join(a.Issues, "\n"), "诊断") ||
			!tester.HasText("折叠分析") || !tester.HasText("暂时显示全文") {
			t.Fatalf("strict JSON comment was not rejected visibly: candidates=%v message=%q texts=%v",
				tab.Editor.FoldCandidates(), a.Message, tester.Texts())
		}
	})
	for _, suffix := range []string{".jsonc", ".ts"} {
		t.Run(suffix+"-unsupported", func(t *testing.T) {
			a, tab, tester, _ := openSourceFoldFixture(t, suffix, foldGoFixture)
			if tab.MatchedPlugin.ID != "code" || len(tab.Editor.FoldCandidates()) != 0 ||
				tester.HasText("折叠当前块") || sourceFoldCommandAvailable(a) {
				t.Fatalf("unsupported suffix received structural semantics: matched=%q folds=%v texts=%v",
					tab.MatchedPlugin.ID, tab.Editor.FoldCandidates(), tester.Texts())
			}
		})
	}
}

func TestSourceFoldCommandsUseCurrentCaretAndRemainViewOnly(t *testing.T) {
	a, tab, tester, _ := openSourceFoldFixture(t, ".go", foldGoFixture)
	if len(tab.Editor.FoldCandidates()) != 2 {
		t.Fatal("expected the Go function and conditional candidates")
	}
	// The palette captures only a command ID. Move the live caret after the
	// row was rendered; execution must resolve this position now, not retain it.
	paletteOpen(t, a, tester)
	tester.Type("折叠当前位置")
	inner := strings.Index(foldGoFixture, "if true {")
	if inner < 0 {
		t.Fatal("nested Go block missing")
	}
	inner += len("if true ") // place the live caret on the nested opener
	if err := tab.Doc.Buffer.SetSelection(text.UTF16Len(foldGoFixture[:inner]), text.UTF16Len(foldGoFixture[:inner])); err != nil {
		t.Fatal(err)
	}
	before := tab.Doc.Buffer.Snapshot()
	version, history := tab.Doc.Buffer.Version(), tab.Doc.Buffer.History()
	selection := tab.Doc.Buffer.Selection()
	changes := 0
	oldOnChange := tab.Editor.OnChange
	tab.Editor.OnChange = func() {
		changes++
		if oldOnChange != nil {
			oldOnChange()
		}
	}
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	candidates := tab.Editor.FoldCandidates()
	innerCollapsed := false
	outerCollapsed := false
	innerFound, outerFound := false, false
	for _, candidate := range candidates {
		if candidate.Range.OpenLine == 5 {
			innerFound = true
			innerCollapsed = candidate.Collapsed
		}
		if candidate.Range.OpenLine == 2 {
			outerFound = true
			outerCollapsed = candidate.Collapsed
		}
	}
	if !innerFound || !outerFound {
		t.Fatalf("expected both inner and outer fixture candidates: %+v", candidates)
	}
	if !innerCollapsed || outerCollapsed {
		t.Fatalf("palette used stale rather than current caret: %+v", candidates)
	}
	assertFoldDocumentUnchanged(t, tab, before, version, history, selection, changes)

	paletteRun(t, a, tester, "展开当前位置")
	for _, candidate := range tab.Editor.FoldCandidates() {
		if candidate.Collapsed {
			t.Fatal("expand-current command left the caret block collapsed")
		}
	}
	if err := tester.Click("折叠当前块"); err != nil {
		t.Fatal(err)
	}
	if !tab.Editor.FoldCandidates()[1].Collapsed {
		t.Fatal("native editor toolbar did not fold the actual current block")
	}
	paletteRun(t, a, tester, "展开全部")
	for _, candidate := range tab.Editor.FoldCandidates() {
		if candidate.Collapsed {
			t.Fatal("expand-all command left a fold collapsed")
		}
	}
	paletteRun(t, a, tester, "重新分析折叠")
	if len(tab.Editor.FoldCandidates()) != 2 {
		t.Fatal("reanalysis did not republish current Go folds")
	}
	assertFoldDocumentUnchanged(t, tab, before, version, history, selection, changes)
}

func TestSourceFoldCommandsAreAvailableForReadonlyAndSplitAndRejectIME(t *testing.T) {
	root := t.TempDir()
	a := newTestApp(t, filepath.Join(root, "config"))
	path := filepath.Join(root, "readonly.json")
	if err := os.WriteFile(path, []byte(foldJSONFixture), 0400); err != nil {
		t.Fatal(err)
	}
	a.Open(path)
	tab := a.Current()
	if tab == nil || !tab.Doc.ReadOnly || !tab.Editor.ReadOnly {
		t.Fatalf("readonly fixture was not recognized by the real app route: %#v", tab)
	}
	tab.Mode = "split"
	tab.Adapter = &editor.Adapter{
		ID: "fold-test-split", Name: "折叠分屏测试",
		Create: func(editor.Host) editor.View { return foldSplitTestView{} },
	}
	tester := ui.NewTester(a.View, 900, 620)
	tester.Frame()
	if !tester.HasText("折叠当前块") || !sourceFoldCommandAvailable(a) {
		t.Fatal("read-only split source with a supported provider must keep view folding available")
	}
	caret := strings.Index(foldJSONFixture, "1,")
	if err := tab.Doc.Buffer.SetSelection(text.UTF16Len(foldJSONFixture[:caret]), text.UTF16Len(foldJSONFixture[:caret])); err != nil {
		t.Fatal(err)
	}
	// This original body caret lies inside the lines that the array would hide.
	// Readonly and split keep folding available, but cannot hide live selection.
	buffer := tab.Doc.Buffer
	before := buffer.Snapshot()
	version, history, selection := buffer.Version(), buffer.History(), buffer.Selection()
	generation, selections := buffer.SelectionGeneration(), buffer.SelectionSet()
	ownerDoc, ownerEditor := tab.Doc, tab.Editor
	beforeCandidates := tab.Editor.FoldCandidates()
	changes := 0
	oldOnChange := tab.Editor.OnChange
	tab.Editor.OnChange = func() {
		changes++
		if oldOnChange != nil {
			oldOnChange()
		}
	}
	paletteRun(t, a, tester, "折叠当前位置")
	if !strings.Contains(a.Message, "请先将光标、选区或输入法组合移出折叠区域") ||
		!tester.HasText("请先将光标、选区或输入法组合移出折叠区域") {
		t.Fatalf("hidden body caret was not rejected visibly: message=%q texts=%v", a.Message, tester.Texts())
	}
	assertFoldDocumentUnchanged(t, tab, before, version, history, selection, changes)
	if buffer.String() != foldJSONFixture || buffer.SelectionGeneration() != generation ||
		!reflect.DeepEqual(buffer.SelectionSet(), selections) ||
		!reflect.DeepEqual(tab.Editor.FoldCandidates(), beforeCandidates) ||
		a.Current() != tab || tab.Doc != ownerDoc || tab.Editor != ownerEditor ||
		tab.Editor.Buffer != buffer || tab.Doc.Buffer != buffer {
		t.Fatal("hidden-caret refusal changed exact source, directed selections, candidates or owner")
	}
	arrayFound := false
	for _, candidate := range tab.Editor.FoldCandidates() {
		arrayFound = arrayFound || candidate.Range.OpenLine == 1
		if candidate.Collapsed {
			t.Fatal("hidden body caret refusal left a JSON fold collapsed")
		}
	}
	if len(tab.Editor.FoldCandidates()) != 3 || !arrayFound {
		t.Fatal("hidden body caret refusal lost the original array candidate")
	}

	// Composition is provisional document state. Neither opening the palette
	// nor running a fold action may Flush/Unmark or commit it implicitly.
	tab.Doc.ReadOnly, tab.Editor.ReadOnly = false, false
	tab.Mode = "source"
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	open := tab.Editor.FoldCandidates()[1].Range.OpenStart
	if err := tab.Doc.Buffer.SetSelection(open, open); err != nil {
		t.Fatal(err)
	}
	tab.Editor.SetMarkedText(nil, "候选", ui.TextInputRange{Start: 2, End: 2})
	marked, composing := tab.Editor.MarkedRange()
	if !composing {
		t.Fatal("composition fixture did not begin")
	}
	composedSnapshot := tab.Doc.Buffer.Snapshot()
	composedVersion, composedHistory, composedSelection := tab.Doc.Buffer.Version(), tab.Doc.Buffer.History(), tab.Doc.Buffer.Selection()
	if a.openCommandPalette() || a.commandPalette.open {
		t.Fatal("command palette opened during source composition")
	}
	a.runSourceFoldCommand(foldCurrentPosition)
	gotMarked, stillComposing := tab.Editor.MarkedRange()
	if !stillComposing || gotMarked != marked || !tab.Doc.Buffer.EqualSnapshot(composedSnapshot) ||
		tab.Doc.Buffer.Version() != composedVersion || tab.Doc.Buffer.History() != composedHistory ||
		tab.Doc.Buffer.Selection() != composedSelection {
		t.Fatal("fold command flushed, unmarked, or changed active composition")
	}
	tab.Editor.HandleInput(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEscape})
	if _, stillComposing := tab.Editor.MarkedRange(); stillComposing {
		t.Fatal("Escape did not cancel the IME test fixture")
	}
}

func TestSourceFoldOpeningBracketCommandsPreserveReadonlySourceAndSplit(t *testing.T) {
	for _, mode := range []string{"source", "split"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			a := newTestApp(t, filepath.Join(root, "config"))
			path := filepath.Join(root, "readonly.json")
			if err := os.WriteFile(path, []byte(foldJSONFixture), 0400); err != nil {
				t.Fatal(err)
			}
			a.Open(path)
			tab := a.Current()
			if tab == nil || !tab.Doc.ReadOnly || !tab.Editor.ReadOnly || tab.Mode != "source" ||
				tab.Editor.Buffer != tab.Doc.Buffer || tab.MatchedPlugin.ID != "code" || !tab.MatchedPlugin.Builtin {
				t.Fatalf("readonly source fixture did not retain its real owner: %#v", tab)
			}
			tester := ui.NewTester(a.View, 900, 620)
			preview := &foldSplitTestView{}
			var previewHost editor.Host
			creates := 0
			if mode == "split" {
				tab.Adapter = &editor.Adapter{
					ID: "fold-test-readonly-preview", Name: "只读折叠分屏测试",
					Create: func(host editor.Host) editor.View {
						creates++
						previewHost = host
						return preview
					},
				}
				tester.Frame()
				if err := tester.Click("分屏"); err != nil {
					t.Fatal(err)
				}
			}
			tester.Frame()
			if tab.Mode != mode || !sourceFoldCommandAvailable(a) || !tester.HasText("折叠当前块") {
				t.Fatalf("readonly %s view did not expose folding: mode=%q texts=%v", mode, tab.Mode, tester.Texts())
			}
			if mode == "split" && (creates != 1 || tab.View != preview || !previewHost.ReadOnly ||
				previewHost.Buffer != tab.Doc.Buffer || previewHost.Path != path) {
				t.Fatalf("split preview lost readonly/source ownership: creates=%d host=%+v view=%T", creates, previewHost, tab.View)
			}

			// Seed both undo and redo entries, returning to the exact file bytes
			// before exercising the readonly view commands.
			buffer := tab.Doc.Buffer
			body := text.UTF16Len(foldJSONFixture[:strings.Index(foldJSONFixture, "1,")])
			if err := buffer.ReplaceUTF16(body, body+1, "2"); err != nil {
				t.Fatal(err)
			}
			if err := buffer.ReplaceUTF16(body, body+1, "1"); err != nil {
				t.Fatal(err)
			}
			end := buffer.LenUTF16()
			if err := buffer.ReplaceUTF16(end, end, " "); err != nil {
				t.Fatal(err)
			}
			if !buffer.Undo() || buffer.String() != foldJSONFixture || buffer.History().UndoDepth != 2 || buffer.History().RedoDepth != 1 {
				t.Fatal("fixture did not seed nonempty undo and redo histories with the original bytes")
			}
			paletteRun(t, a, tester, "重新分析折叠")
			opener := text.UTF16Len(foldJSONFixture[:strings.Index(foldJSONFixture, "[")])
			if err := buffer.SetSelection(opener, opener); err != nil {
				t.Fatal(err)
			}
			tester.Frame()
			before := buffer.Snapshot()
			version, history, selection := buffer.Version(), buffer.History(), buffer.Selection()
			generation, selections := buffer.SelectionGeneration(), buffer.SelectionSet()
			ownerDoc, ownerEditor, ownerView := tab.Doc, tab.Editor, tab.View
			changes := 0
			oldOnChange := tab.Editor.OnChange
			tab.Editor.OnChange = func() {
				changes++
				if oldOnChange != nil {
					oldOnChange()
				}
			}
			assertState := func(collapsed bool) {
				t.Helper()
				assertFoldDocumentUnchanged(t, tab, before, version, history, selection, changes)
				if buffer.String() != foldJSONFixture || buffer.SelectionGeneration() != generation ||
					!reflect.DeepEqual(buffer.SelectionSet(), selections) || a.Current() != tab ||
					tab.Doc != ownerDoc || tab.Editor != ownerEditor || tab.Doc.Buffer != buffer ||
					tab.Editor.Buffer != buffer || tab.View != ownerView || !tab.Doc.ReadOnly || !tab.Editor.ReadOnly {
					t.Fatal("fold view command changed exact source, directed selections or readonly owner")
				}
				if mode == "split" && (creates != 1 || tab.View != preview || !previewHost.ReadOnly || previewHost.Buffer != buffer) {
					t.Fatal("fold view command replaced or made the split preview writable")
				}
				candidates := tab.Editor.FoldCandidates()
				arrayFound := false
				for _, candidate := range candidates {
					if candidate.Range.OpenStart == opener {
						arrayFound = true
						if candidate.Collapsed != collapsed {
							t.Fatalf("array collapsed=%v want %v: %+v", candidate.Collapsed, collapsed, candidates)
						}
					} else if candidate.Collapsed {
						t.Fatalf("opener command collapsed another JSON owner: %+v", candidates)
					}
				}
				if len(candidates) != 3 || !arrayFound {
					t.Fatalf("expected all JSON folds including the actual array opener: %+v", candidates)
				}
				disk, err := os.ReadFile(path)
				if err != nil || string(disk) != foldJSONFixture {
					t.Fatalf("readonly view command changed original disk bytes: %q err=%v", disk, err)
				}
			}
			assertState(false)
			paletteRun(t, a, tester, "折叠当前位置")
			assertState(true)
			paletteRun(t, a, tester, "展开当前位置")
			assertState(false)
			if err := tester.Click("折叠当前块"); err != nil {
				t.Fatal(err)
			}
			tester.Frame()
			assertState(true)
			if err := tester.Click("展开当前块"); err != nil {
				t.Fatal(err)
			}
			tester.Frame()
			assertState(false)

			// Replay the original entries to prove history contents survived too.
			if !buffer.Redo() || buffer.String() != foldJSONFixture+" " || !buffer.Undo() || buffer.String() != foldJSONFixture {
				t.Fatal("fold commands changed the original redo entry")
			}
			if !buffer.Undo() || buffer.String() != strings.Replace(foldJSONFixture, "1,", "2,", 1) ||
				!buffer.Redo() || buffer.String() != foldJSONFixture || buffer.History() != history {
				t.Fatal("fold commands changed the original undo entry or history order")
			}
		})
	}
}

func TestInstalledCustomGoRouteIsNotForcedGoAndReconfigurationInvalidatesTargets(t *testing.T) {
	root := t.TempDir()
	a := newTestApp(t, filepath.Join(root, "config"))
	installAppPluginManifest(t, a, "custom-go.json", `{"schema":1,"id":"custom-go","name":"Custom Go-like source","extensions":[".go"]}`)
	path := filepath.Join(root, "sample.go")
	if err := os.WriteFile(path, []byte(foldGoFixture), 0600); err != nil {
		t.Fatal(err)
	}
	a.Open(path)
	tab := a.Current()
	if tab == nil || tab.MatchedPlugin.ID != "custom-go" || tab.MatchedPlugin.Builtin ||
		len(tab.Editor.FoldCandidates()) != 0 {
		t.Fatalf("installed custom .go route was forced into builtin Go: tab=%#v folds=%v", tab, tab.Editor.FoldCandidates())
	}
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	if tester.HasText("折叠当前块") || sourceFoldCommandAvailable(a) {
		t.Fatal("custom Go route exposed builtin folding controls")
	}
	if err := a.pluginManager.SetEnabled("custom-go", false); err != nil {
		t.Fatal(err)
	}
	a.reconfigureTabs()
	tester.Frame()
	if tab.MatchedPlugin.ID != "code" || len(tab.Editor.FoldCandidates()) != 2 || !tester.HasText("折叠当前块") {
		t.Fatalf("disabled custom owner did not select the actual builtin code profile: id=%q folds=%d", tab.MatchedPlugin.ID, len(tab.Editor.FoldCandidates()))
	}

	old := tab.Editor.FoldCandidates()[0].Target
	oldPath, oldName := tab.Doc.Path, tab.Name
	tab.Doc.Path = strings.TrimSuffix(oldPath, ".go") + ".jsonc"
	tab.Name = filepath.Base(tab.Doc.Path)
	a.configureTab(tab)
	tester.Frame()
	if len(tab.Editor.FoldCandidates()) != 0 || tester.HasText("折叠当前块") {
		t.Fatal("renaming to unsupported .jsonc retained the previous Go provider")
	}
	tab.Doc.Path, tab.Name = oldPath, oldName
	a.configureTab(tab)
	tester.Frame()
	if len(tab.Editor.FoldCandidates()) != 2 {
		t.Fatal("A→B→A did not install a fresh builtin Go provider")
	}
	if err := tab.Editor.ToggleFold(old); !errors.Is(err, editor.ErrFoldStale) {
		t.Fatalf("old target survived A→B→A provider reconfiguration: %v", err)
	}
	for _, candidate := range tab.Editor.FoldCandidates() {
		if candidate.Collapsed {
			t.Fatal("old collapsed state was restored after reconfiguration")
		}
	}
}

func TestConfigureTabClearsFoldingForInspectorAndBinaryOwners(t *testing.T) {
	a, tab, tester, _ := openSourceFoldFixture(t, ".go", foldGoFixture)
	if len(tab.Editor.FoldCandidates()) == 0 {
		t.Fatal("fixture did not start with configured Go folds")
	}
	tab.ByteInspector = true
	a.configureTab(tab)
	tester.Frame()
	if len(tab.Editor.FoldCandidates()) != 0 || tester.HasText("折叠当前块") {
		t.Fatal("byte-inspector reconfiguration retained source folding")
	}
	tab.ByteInspector = false
	a.configureTab(tab)
	if len(tab.Editor.FoldCandidates()) == 0 {
		t.Fatal("leaving byte inspector did not restore the current builtin profile")
	}
	tab.BinaryPath = filepath.Join(t.TempDir(), "sample.go")
	a.configureTab(tab)
	tester.Frame()
	if len(tab.Editor.FoldCandidates()) != 0 || tester.HasText("折叠当前块") {
		t.Fatal("binary reconfiguration retained text folding")
	}
}

func TestSourceFoldOversizeStopsAtEditorByteBudget(t *testing.T) {
	const size = editor.MaxFoldBytes + 1
	source := strings.Repeat("x", size)
	a := newTestApp(t, t.TempDir())
	tab := a.appendTab(document.New(source), "large.go", "", false, false)
	a.Active = len(a.Tabs) - 1
	if tab.Doc.Buffer.LenBytes() != size || len(tab.Editor.FoldCandidates()) != 0 ||
		!strings.Contains(a.Message, "字节预算") {
		t.Fatalf("oversize source was not rejected at the editor budget: bytes=%d message=%q", tab.Doc.Buffer.LenBytes(), a.Message)
	}
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	if !tester.HasText("折叠分析超过字节预算") || !tester.HasText("已显示全文") {
		t.Fatalf("oversize status is not persistent/user-visible: %v", tester.Texts())
	}
}

func TestFoldPaletteRejectsOldOwnerAfterTabSwitch(t *testing.T) {
	root := t.TempDir()
	a := newTestApp(t, filepath.Join(root, "config"))
	for _, name := range []string{"first.go", "second.go"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(foldGoFixture), 0600); err != nil {
			t.Fatal(err)
		}
	}
	a.Open(filepath.Join(root, "first.go"))
	first := a.Current()
	a.Open(filepath.Join(root, "second.go"))
	second := a.Current()
	tester := ui.NewTester(a.View, 900, 620)
	if !a.Select(0) {
		t.Fatal("could not activate first tab")
	}
	paletteOpen(t, a, tester)
	tester.Type("折叠当前位置")
	a.commandPalette.pendingID = "fold.current"
	a.Active = 1 // invalidate the palette's captured owner before dispatch
	tester.Frame()
	for _, candidate := range second.Editor.FoldCandidates() {
		if candidate.Collapsed {
			t.Fatal("stale palette action folded the replacement tab")
		}
	}
	for _, candidate := range first.Editor.FoldCandidates() {
		if candidate.Collapsed {
			t.Fatal("stale palette action folded the former owner")
		}
	}
	if a.Current() != second || a.commandPalette.open {
		t.Fatal("stale command did not expire with its original palette owner")
	}
}

func TestSourceFoldPaletteAvailabilityTracksProfileAndReadonly(t *testing.T) {
	for _, tc := range []struct {
		suffix string
		want   bool
	}{
		{".go", true}, {".json", true}, {".jsonc", false}, {".ts", false}, {".txt", false},
	} {
		t.Run(tc.suffix, func(t *testing.T) {
			a, tab, tester, _ := openSourceFoldFixture(t, tc.suffix, foldJSONFixture)
			if tc.want {
				tab.Doc.ReadOnly, tab.Editor.ReadOnly = true, true
			}
			enabled := foldPaletteCommandEnabled(t, a, tester, "折叠当前位置")
			if enabled != tc.want {
				t.Fatalf("palette enabled=%v want %v for suffix %s; %v", enabled, tc.want, tc.suffix, tester.Texts())
			}
		})
	}
}
