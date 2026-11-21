package app

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/code"
)

func TestLanguageHostActualBuiltinAndCustomRoute(t *testing.T) {
	for _, tc := range []struct {
		name, suffix, source string
		profile              code.StructureProfile
	}{
		{"go", ".go", languageGoSource, code.StructureGo},
		{"strict-json", ".json", "{\r\n  \"🙂\": [{\"\\uDEAD\":1,\"\\uDEAD\":2}],\n  \"alpha\": true\r\n}", code.StructureJSON},
		{"jsonc", ".jsonc", "{\"x\":1}", ""},
		{"typescript", ".ts", "const x = 1", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, tab, q, _ := languageFixture(t, tc.suffix, tc.source)
			if tab.MatchedPlugin.ID != "code" || tab.sourceLanguageProfile != tc.profile || a.sourceLanguage.task != nil || q.outstanding.Load() != 0 {
				t.Fatal("actual built-in configuration capability/idle state mismatch")
			}
			before := languageState(tab)
			if tc.profile == "" {
				if err := a.requestSourceLanguage(languageIntent(code.ServiceSymbols)); err == nil || a.sourceLanguage.status != string(code.ServiceUnsupported) || a.sourceLanguage.task != nil {
					t.Fatal("unsupported suffix inherited Go/JSON semantics")
				}
			} else {
				languageComplete(t, a, q, languageIntent(code.ServiceSymbols))
				if a.sourceLanguage.status != string(code.ServiceComplete) || len(a.sourceLanguage.result.Symbols) == 0 {
					t.Fatalf("real symbols missing: %+v", a.sourceLanguage)
				}
				for _, symbol := range a.sourceLanguage.result.Symbols {
					span := symbol.Span
					if span.Bytes.Start < 0 || span.Bytes.End < span.Bytes.Start || span.Bytes.End > len(tc.source) || tc.source[span.Bytes.Start:span.Bytes.End] != symbol.Name ||
						span.UTF16.Start != len(utf16.Encode([]rune(tc.source[:span.Bytes.Start]))) || span.UTF16.End != len(utf16.Encode([]rune(tc.source[:span.Bytes.End]))) ||
						span.Line != strings.Count(tc.source[:span.Bytes.Start], "\n") {
						t.Fatalf("real original byte/UTF-16/physical-line span changed: %+v", symbol)
					}
				}
				if tc.profile == code.StructureJSON {
					duplicate := 0
					for _, symbol := range a.sourceLanguage.result.Symbols {
						if symbol.Name == `"\uDEAD"` {
							duplicate++
						}
					}
					if duplicate != 2 {
						t.Fatalf("strict JSON lost original duplicate keys: %+v", a.sourceLanguage.result.Symbols)
					}
				}
			}
			languageUnchanged(t, tab, before)
		})
	}
	for _, schema := range []int{1, 2, 3} {
		t.Run(fmt.Sprintf("installed-schema-%d", schema), func(t *testing.T) {
			a, tab, q, _ := languageFixture(t, ".go", languageGoSource)
			languageComplete(t, a, q, languageIntent(code.ServiceSymbols))
			old := a.sourceLanguage.owner
			extra := ""
			if schema == 2 {
				extra = `,"outline":[{"prefix":"func ","suffix":"","level":1}]`
			}
			if schema == 3 {
				extra = `,"nativeRoute":"builtin.markdown"`
			}
			manifest := fmt.Sprintf(`{"schema":%d,"id":"language-occupied","name":"Local route","extensions":[".go"]%s}`, schema, extra)
			installAppPluginManifest(t, a, "language-occupied.json", manifest)
			a.reconfigureTabs()
			if tab.MatchedPlugin.ID != "language-occupied" || tab.sourceLanguageProfile != "" || a.sourceLanguage.owner != nil || a.sourceLanguageOwnerValid(old, false) == nil {
				t.Fatal("installed route retained builtin language ownership")
			}
			if schema == 3 && (tab.Adapter == nil || tab.Adapter.ID != "markdown") {
				t.Fatal("schema3 fixture did not install actual native route")
			}
			before := languageState(tab)
			if err := a.requestSourceLanguage(languageIntent(code.ServiceDiagnostics)); err == nil || a.sourceLanguage.status != string(code.ServiceUnsupported) || a.sourceLanguage.task != nil {
				t.Fatal("installed syntax/native route inherited Go semantics")
			}
			languageUnchanged(t, tab, before)
			a.setPluginEnabled("language-occupied", false)
			if tab.MatchedPlugin.ID != "code" || tab.sourceLanguageProfile != code.StructureGo {
				t.Fatalf("builtin route did not return: %q / %q", tab.MatchedPlugin.ID, tab.sourceLanguageProfile)
			}
			languageComplete(t, a, q, languageIntent(code.ServiceSymbols))
			if a.sourceLanguage.status != string(code.ServiceComplete) {
				t.Fatal("real builtin analyzer did not return")
			}
			a.setPluginEnabled("code", false)
			if err := a.requestSourceLanguage(languageIntent(code.ServiceDiagnostics)); err == nil || a.sourceLanguage.status != string(code.ServiceUnsupported) {
				t.Fatal("disabled builtin still analyzed")
			}
			a.setPluginEnabled("code", true)
			languageComplete(t, a, q, languageIntent(code.ServiceDiagnostics))
			if a.sourceLanguage.status != string(code.ServiceComplete) || old.request == a.sourceLanguage.owner.request {
				t.Fatal("re-enable did not establish a fresh language owner")
			}
		})
	}
}

func TestLanguageHostUnavailableAndUnconfiguredRouteRejectsCompletion(t *testing.T) {
	for _, name := range []string{"manager-nil", "manager-disabled-without-reconfigure", "inspector-early-return", "binary-early-return"} {
		t.Run(name, func(t *testing.T) {
			a, tab, q, _ := languageFixture(t, ".go", languageGoSource)
			started, release := languageHold(t, a)
			task := languageRequest(t, a, languageIntent(code.ServiceSymbols))
			languageReceive(t, started)
			manager := a.pluginManager
			switch name {
			case "manager-nil":
				a.pluginManager = nil
			case "manager-disabled-without-reconfigure":
				if err := manager.SetEnabled("code", false); err != nil {
					t.Fatal(err)
				}
			case "inspector-early-return":
				tab.ByteInspector = true
				a.configureTab(tab)
			case "binary-early-return":
				tab.BinaryPath = filepath.Join(t.TempDir(), "large.png")
				tab.BinarySize = 20 << 20
				a.configureTab(tab)
			}
			before := languageState(tab)
			release()
			languageWait(t, task.done)
			languageReceive(t, q.calls)()
			if a.sourceLanguage.status == string(code.ServiceComplete) || a.sourceLanguage.owner != nil || len(a.sourceLanguage.result.Symbols) != 0 {
				t.Fatal("unavailable/replaced actual route published old result")
			}
			languageUnchanged(t, tab, before)
			a.pluginManager = manager
		})
	}
}

func TestLanguageHostSaveAsAndPublicationMetadataAdoption(t *testing.T) {
	for _, name := range []string{"go-to-json", "go-to-text", "extensionless-to-go", "same-extension-adopted-failure", "same-path-adopted-failure"} {
		t.Run(name, func(t *testing.T) {
			suffix := ".go"
			if name == "extensionless-to-go" {
				suffix = ""
			}
			a, tab, q, path := languageFixture(t, suffix, languageGoSource)
			if err := tab.Doc.Buffer.ReplaceUTF16(0, 0, " "); err != nil {
				t.Fatal(err)
			}
			var task *sourceLanguageTask
			var release func()
			var owner *sourceLanguageOwner
			if suffix != "" {
				started, finish := languageHold(t, a)
				release = finish
				task = languageRequest(t, a, languageIntent(code.ServiceSymbols))
				languageReceive(t, started)
				owner = a.sourceLanguage.owner
			}
			target := filepath.Join(filepath.Dir(path), "saved.go")
			if name == "go-to-json" {
				target = filepath.Join(filepath.Dir(path), "saved.json")
			}
			if name == "go-to-text" {
				target = filepath.Join(filepath.Dir(path), "saved.txt")
			}
			if name == "same-path-adopted-failure" {
				target = path
			}
			a.saveDialog = func(*Tab) (string, error) { return target, nil }
			before := languageState(tab)
			adoptedFailure := strings.Contains(name, "adopted-failure")
			if adoptedFailure {
				a.saveDocument = func(doc *document.Document, destination string) error {
					if err := os.WriteFile(destination, []byte("publication not confirmed"), 0600); err != nil {
						t.Fatal(err)
					}
					doc.Path, doc.IsNew, doc.ReadOnly = destination, false, false
					return &document.PublicationError{Path: destination, MetadataAdopted: true, Cause: document.ErrConflict}
				}
			}
			if ok := a.Save(tab, true); ok == adoptedFailure {
				t.Fatalf("SaveAs result=%v adoptedFailure=%v: %s", ok, adoptedFailure, a.Message)
			}
			if owner != nil && a.sourceLanguageOwnerValid(owner, false) == nil {
				t.Fatal("SaveAs metadata adoption preserved old owner")
			}
			if a.sourceLanguage.owner != nil || a.sourceLanguage.pending != nil || len(a.sourceLanguage.result.Symbols) != 0 {
				t.Fatal("SaveAs retained language owner or payload")
			}
			if tab.Doc.Path != target || tab.Editor.Buffer != before.buffer || tab.Doc.Buffer != before.buffer || tab.Doc.Encoding != before.encoding || tab.Doc.BOM != before.bom ||
				tab.Doc.Buffer.String() != before.source || tab.Doc.Buffer.Version() != before.version || tab.Doc.Buffer.History() != before.history || !reflect.DeepEqual(tab.Doc.Buffer.SelectionSet(), before.selections) {
				t.Fatal("SaveAs language invalidation changed document content/history/encoding")
			}
			if adoptedFailure && (!tab.Doc.Dirty() || !tab.Conflict || strings.Contains(a.Message, "已保存 ") || !strings.Contains(a.Message, "元数据已采用目标")) {
				t.Fatal("language host made adopted publication failure clean/successful")
			}
			if !adoptedFailure && tab.Doc.Dirty() {
				t.Fatal("successful Document SaveAs remained dirty")
			}
			if task != nil {
				release()
				languageWait(t, task.done)
				languageReceive(t, q.calls)()
			}
			if a.sourceLanguage.status == string(code.ServiceComplete) {
				t.Fatal("late pre-save result published")
			}
			wantProfile := code.StructureGo
			if name == "go-to-json" {
				wantProfile = code.StructureJSON
			}
			if name == "go-to-text" {
				wantProfile = ""
			}
			if tab.sourceLanguageProfile != wantProfile {
				t.Fatalf("adopted route profile=%q want %q", tab.sourceLanguageProfile, wantProfile)
			}
			if name == "extensionless-to-go" {
				languageComplete(t, a, q, languageIntent(code.ServiceSymbols))
				if a.sourceLanguage.status != string(code.ServiceComplete) {
					t.Fatal("new .go metadata did not enable the real analyzer")
				}
			}
		})
	}
}

func TestLanguageHostReadPreservesEncodingBOMHistoryAndReadonly(t *testing.T) {
	for _, name := range []string{"utf8-bom", "utf16le-bom", "readonly"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "original.go")
			raw := append([]byte{0xef, 0xbb, 0xbf}, []byte(languageGoSource)...)
			if name == "utf16le-bom" {
				raw = []byte{0xff, 0xfe}
				for _, value := range utf16.Encode([]rune(languageGoSource)) {
					raw = binary.LittleEndian.AppendUint16(raw, value)
				}
			}
			mode := os.FileMode(0600)
			if name == "readonly" {
				mode = 0400
			}
			if err := os.WriteFile(path, raw, mode); err != nil {
				t.Fatal(err)
			}
			a := newTestApp(t, filepath.Join(root, "config"))
			a.Open(path)
			tab := a.Current()
			if tab == nil || tab.Doc.Buffer.String() != languageGoSource || !tab.Doc.BOM || (name == "readonly" && (!tab.Doc.ReadOnly || !tab.Editor.ReadOnly)) {
				t.Fatal("real encoded/readonly App fixture mismatch")
			}
			q := newLanguageQueue()
			a.sourceLanguageHooks = &sourceLanguageHooks{dispatch: q.dispatch}
			// Preserve an actual redo entry and a full directed selection set.
			if name != "readonly" {
				if err := tab.Doc.Buffer.ReplaceUTF16(0, 0, " "); err != nil {
					t.Fatal(err)
				}
				tab.Doc.Buffer.Undo()
			}
			if err := tab.Doc.Buffer.SetSelectionSet(text.SelectionSet{Ranges: []text.Range{{Anchor: 5, Head: 1}, {Anchor: 8, Head: 8}}, Primary: 1}); err != nil {
				t.Fatal(err)
			}
			before := languageState(tab)
			changes := 0
			tab.Editor.OnChange = func() { changes++ }
			languageComplete(t, a, q, languageIntent(code.ServiceSymbols))
			if a.sourceLanguage.status != string(code.ServiceComplete) || changes != 0 {
				t.Fatal("real readonly/encoded analysis failed or emitted an edit")
			}
			languageUnchanged(t, tab, before)
			if !bytes.Equal(mustReadFile(t, path), raw) {
				t.Fatal("analysis touched original disk bytes")
			}
			if name != "readonly" {
				if !a.Save(tab, false) || !bytes.Equal(mustReadFile(t, path), raw) {
					t.Fatal("explicit ordinary Save changed BOM/encoding/EOL after analysis")
				}
			}
			if name == "readonly" {
				tester := ui.NewTester(a.View, 900, 620)
				tester.Frame()
				languageUnchanged(t, tab, before)
			}
		})
	}
}
