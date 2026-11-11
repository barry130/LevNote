package plugins

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const schema3Literal = `{"schema":3,"id":"task-notes","name":"Task Notes","extensions":["TASKMD",".taskmd"],"nativeRoute":"builtin.markdown"}`

func TestSchema3RoutesAndOmittedDefaults(t *testing.T) {
	for _, route := range []NativeRoute{NativeRouteMarkdown, NativeRouteMindmap, NativeRouteDrawio} {
		t.Run(string(route), func(t *testing.T) {
			input := strings.Replace(schema3Literal, "builtin.markdown", string(route), 1)
			syntax := mustDecode(t, input)
			if syntax.Schema != 3 || syntax.ID != "task-notes" || syntax.Name != "Task Notes" || syntax.Version != "1.0.0" || syntax.APIVersion != 1 || syntax.NativeRoute != route || syntax.Outline != nil || !reflect.DeepEqual(syntax.Extensions, []string{"TASKMD", ".taskmd"}) {
				t.Fatalf("decoded route/defaults = %#v", syntax)
			}
		})
	}
	full := `{"schema":3,"id":"native-syntax","name":"原生😀","version":"2","apiVersion":1,"extensions":[".x"],"nativeRoute":"builtin.mindmap","lineComment":"#","blockStart":"/*","blockEnd":"*/","keywords":["true","false"]}`
	syntax := mustDecode(t, full)
	if syntax.Version != "2" || syntax.Name != "原生😀" || syntax.LineComment != "#" || syntax.BlockStart != "/*" || syntax.BlockEnd != "*/" || !reflect.DeepEqual(syntax.Keywords, []string{"true", "false"}) {
		t.Fatalf("source syntax was lost: %#v", syntax)
	}
}

func TestSchema3StrictAdmission(t *testing.T) {
	// Expected invalid fragments are authored independently of the decoder.
	cases := map[string]string{
		"unknown-route":       `"nativeRoute":"builtin.unknown"`,
		"wrongcase-drawio":    `"nativeRoute":"BUILTIN.DRAWIO"`,
		"trailing-drawio":     `"nativeRoute":"builtin.drawio "`,
		"leading-drawio":      `"nativeRoute":" builtin.drawio"`,
		"empty-route":         `"nativeRoute":""`,
		"path-route":          `"nativeRoute":"C:/engine.exe"`,
		"array-route":         `"nativeRoute":["builtin.markdown"]`,
		"version-empty":       `"version":""`,
		"version-whitespace":  `"version":" "`,
		"version-number":      `"version":1`,
		"api-zero":            `"apiVersion":0`,
		"api-unsupported":     `"apiVersion":2`,
		"api-fraction":        `"apiVersion":1.1`,
		"api-string":          `"apiVersion":"1"`,
		"id-invalid":          `"id":"Bad ID"`,
		"id-double-dash":      `"id":"bad--id"`,
		"name-empty":          `"name":""`,
		"extensions-empty":    `"extensions":[]`,
		"extension-empty":     `"extensions":[""]`,
		"extension-path":      `"extensions":["../x"]`,
		"extension-null":      `"extensions":[null]`,
		"extension-number":    `"extensions":[1]`,
		"extensions-object":   `"extensions":{}`,
		"extensions-string":   `"extensions":".x"`,
		"keyword-null":        `"keywords":[null]`,
		"keyword-number":      `"keywords":[3]`,
		"keyword-empty":       `"keywords":[""]`,
		"keywords-string":     `"keywords":"x"`,
		"comment-number":      `"lineComment":1`,
		"block-unpaired":      `"blockStart":"/*"`,
		"outline-empty":       `"outline":[]`,
		"outline-rule":        `"outline":[{"prefix":"#","level":1}]`,
		"command":             `"command":"worker.exe"`,
		"script":              `"script":"file.js"`,
		"path":                `"path":"C:/engine.exe"`,
		"url":                 `"url":"https://example.invalid"`,
		"dll":                 `"dll":"engine.dll"`,
		"executable":          `"executable":"engine.exe"`,
		"future":              `"future":true`,
		"wrongcase-route":     `"NativeRoute":"builtin.markdown"`,
		"wrongcase-extension": `"Extensions":[".x"]`,
	}
	baseFields := []string{`"schema":3`, `"id":"task-notes"`, `"name":"Task Notes"`, `"extensions":["TASKMD",".taskmd"]`, `"nativeRoute":"builtin.markdown"`}
	for name, fragment := range cases {
		t.Run(name, func(t *testing.T) {
			key := strings.SplitN(fragment, ":", 2)[0]
			input := strings.TrimSuffix(schema3Literal, "}") + "," + fragment + "}"
			for _, original := range baseFields {
				if strings.HasPrefix(original, key+":") {
					input = strings.Replace(schema3Literal, original, fragment, 1)
					break
				}
			}
			if _, err := decodeManifest([]byte(input)); err == nil {
				t.Fatalf("invalid schema3 accepted: %s", input)
			}
		})
	}
	for _, field := range []string{"schema", "id", "name", "version", "apiVersion", "extensions", "nativeRoute", "lineComment", "blockStart", "blockEnd", "keywords"} {
		t.Run("null-"+field, func(t *testing.T) {
			parts := make([]string, 0, len(baseFields)+1)
			for _, value := range baseFields {
				if !strings.HasPrefix(value, `"`+field+`":`) {
					parts = append(parts, value)
				}
			}
			parts = append(parts, `"`+field+`":null`)
			if _, err := decodeManifest([]byte("{" + strings.Join(parts, ",") + "}")); err == nil {
				t.Fatal("present null accepted")
			}
		})
	}
	for _, field := range baseFields {
		t.Run("missing-"+strings.SplitN(field, ":", 2)[0], func(t *testing.T) {
			var parts []string
			for _, value := range baseFields {
				if field != value {
					parts = append(parts, value)
				}
			}
			if _, err := decodeManifest([]byte("{" + strings.Join(parts, ",") + "}")); err == nil {
				t.Fatal("missing required field accepted")
			}
		})
	}
}

func TestSchema3DrawioRouteInstalledLifecycle(t *testing.T) {
	root := t.TempDir()
	installRoot := filepath.Join(root, "installed")
	manager, err := NewManager(filepath.Join(root, "settings"), installRoot)
	if err != nil {
		t.Fatal(err)
	}
	input := `{"schema":3,"id":"diagram-notes","name":"图纸笔记","extensions":["DIAGRAMNOTE",".diagramnote2"],"nativeRoute":"builtin.drawio"}`
	syntax, err := decodeManifest([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if syntax.NativeRoute != NativeRouteDrawio || syntax.Outline != nil ||
		!reflect.DeepEqual(syntax.Extensions, []string{"DIAGRAMNOTE", ".diagramnote2"}) {
		t.Fatalf("draw.io route decode = %#v", syntax)
	}
	source := filepath.Join(root, "diagram-notes.json")
	if err := os.WriteFile(source, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := manager.InstallFile(source); err != nil {
		t.Fatal(err)
	}
	match, err := manager.Match("sample.DiagramNote")
	if err != nil || match.ID != "diagram-notes" {
		t.Fatalf("normalized draw.io route match = %#v %v", match, err)
	}
	manifest, _ := manager.Manifest("diagram-notes")
	if !manifest.Enabled || !manifest.Active || manifest.Builtin ||
		!reflect.DeepEqual(manifest.Extensions, []string{".diagramnote", ".diagramnote2"}) {
		t.Fatalf("draw.io route manifest = %#v", manifest)
	}
	if err := manager.SetEnabled("drawio", false); err != nil {
		t.Fatal(err)
	}
	if match, err := manager.Match("sample.diagramnote2"); err != nil || match.ID != "diagram-notes" {
		t.Fatalf("builtin draw.io state changed installed route = %#v %v", match, err)
	}
	if err := manager.SetEnabled("diagram-notes", false); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Match("sample.diagramnote"); err == nil {
		t.Fatal("disabled installed draw.io route still matched")
	}
}

func TestSchema3DuplicatesCannotDisguiseSchema(t *testing.T) {
	for name, input := range map[string]string{
		"schema-3-to-2":  `{"schema":3,"schema":2,"id":"x","name":"X","extensions":[".x"]}`,
		"schema-2-to-3":  `{"schema":2,"schema":3,"id":"x","name":"X","extensions":[".x"],"nativeRoute":"builtin.markdown"}`,
		"escaped-schema": `{"schema":3,"\u0073chema":2,"id":"x","name":"X","extensions":[".x"]}`,
		"wrongcase":      `{"Schema":3,"schema":2,"id":"x","name":"X","extensions":[".x"]}`,
		"legacy-route":   strings.Replace(schema3Literal, `"schema":3`, `"schema":2`, 1),
		"second-object":  schema3Literal + `{}`,
		"array":          `[` + schema3Literal + `]`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := decodeManifest([]byte(input)); err == nil {
				t.Fatalf("ambiguous manifest accepted: %s", input)
			}
		})
	}
	for _, duplicate := range []string{`"id":"task-notes"`, `"\u0069d":"task-notes"`, `"schema":3`, `"nativeRoute":"builtin.markdown"`, `"version":"1","version":"2"`, `"keywords":[],"keywords":[]`} {
		input := strings.TrimSuffix(schema3Literal, "}") + "," + duplicate + "}"
		if _, err := decodeManifest([]byte(input)); err == nil || !strings.Contains(err.Error(), "duplicate") {
			t.Fatalf("duplicate %s = %v", duplicate, err)
		}
	}
}

func TestSchema3UnicodeEscapeAndUTF8Admission(t *testing.T) {
	for _, spelling := range []string{`"\ud800"`, `"\udc00"`, `"\ud800a"`, `"\ud800\u0041"`, `"\\\"\ud800"`} {
		for _, field := range []string{`"name":` + spelling, `"keywords":[` + spelling + `]`, `"extensions":[` + spelling + `]`} {
			input := strings.TrimSuffix(schema3Literal, "}") + "," + field + "}"
			// Remove existing fields so this oracle must reach surrogate validation.
			if strings.HasPrefix(field, `"name":`) {
				input = strings.Replace(schema3Literal, `"name":"Task Notes"`, field, 1)
			} else if strings.HasPrefix(field, `"extensions":`) {
				input = strings.Replace(schema3Literal, `"extensions":["TASKMD",".taskmd"]`, field, 1)
			}
			if _, err := decodeManifest([]byte(input)); err == nil || !strings.Contains(err.Error(), "surrogate") {
				t.Fatalf("malformed escape accepted/misclassified: %s: %v", input, err)
			}
		}
	}
	for spelling, want := range map[string]string{
		`"\\ud800"`:      `\ud800`,
		`"\\\"x"`:        `\"x`,
		`"\ud83d\ude00"`: "😀",
		`"\u4e2d"`:       "中",
		`"�"`:            "�",
	} {
		got := mustDecode(t, strings.Replace(schema3Literal, `"name":"Task Notes"`, `"name":`+spelling, 1))
		if got.Name != want {
			t.Fatalf("%s decoded %q; want %q", spelling, got.Name, want)
		}
	}
	input := []byte(schema3Literal)
	input[strings.Index(schema3Literal, "Task Notes")] = 0xff
	if _, err := decodeManifest(input); err == nil || !strings.Contains(err.Error(), "UTF-8") {
		t.Fatalf("invalid UTF-8 = %v", err)
	}
}

func TestSchema3BoundsAndLegacyCompatibility(t *testing.T) {
	atCap := []byte(schema3Literal + strings.Repeat(" ", maxManifestBytes-len(schema3Literal)))
	if len(atCap) != 65536 {
		t.Fatal("fixture cap")
	}
	if _, err := decodeManifest(atCap); err != nil {
		t.Fatal("exact cap", err)
	}
	if _, err := decodeManifest(append(atCap, ' ')); err == nil || !strings.Contains(err.Error(), "larger") {
		t.Fatalf("direct over-limit decode = %v", err)
	}
	path := filepath.Join(t.TempDir(), "limit.json")
	if err := os.WriteFile(path, atCap, 0o600); err != nil {
		t.Fatal(err)
	}
	if raw, _, err := readManifestFile(path); err != nil || !bytes.Equal(raw, atCap) {
		t.Fatal("exact file cap", err)
	}
	if err := os.WriteFile(path, append(atCap, ' '), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readManifestFile(path); err == nil {
		t.Fatal("oversize file accepted")
	}
	for _, fragment := range []string{
		`"name":"` + strings.Repeat("x", 129) + `"`,
		`"version":"` + strings.Repeat("x", 65) + `"`,
		`"id":"` + strings.Repeat("x", 65) + `"`,
		`"extensions":[".` + strings.Repeat("x", 32) + `"]`,
		`"keywords":["` + strings.Repeat("x", 65) + `"]`,
		`"keywords":[` + strings.Repeat(`"x",`, 512) + `"x"]`,
		`"lineComment":"123456789"`,
	} {
		key := strings.SplitN(fragment, ":", 2)[0]
		input := strings.TrimSuffix(schema3Literal, "}") + "," + fragment + "}"
		for _, old := range []string{`"name":"Task Notes"`, `"id":"task-notes"`, `"extensions":["TASKMD",".taskmd"]`} {
			if strings.HasPrefix(old, key+":") {
				input = strings.Replace(schema3Literal, old, fragment, 1)
			}
		}
		if _, err := decodeManifest([]byte(input)); err == nil {
			t.Fatalf("over-limit field accepted: %s", key)
		}
	}
	for _, schema := range []int{1, 2} {
		// Do not silently tighten old duplicate/default semantics.
		input := fmt.Sprintf(`{"schema":%d,"id":"legacy","name":"first","name":"last","extensions":["TXT"],"version":null,"apiVersion":0}`, schema)
		got := mustDecode(t, input)
		if got.Name != "last" || got.Version != "1.0.0" || got.APIVersion != 1 || got.NativeRoute != "" || got.Extensions[0] != "TXT" {
			t.Fatal("legacy compatibility", got)
		}
	}
}

func TestSchema3InstalledLifecycleNormalizedMetadataAndCopies(t *testing.T) {
	root := t.TempDir()
	installRoot := filepath.Join(root, "installed")
	manager, err := NewManager(filepath.Join(root, "settings"), installRoot)
	if err != nil {
		t.Fatal(err)
	}
	input := strings.Replace(schema3Literal, `["TASKMD",".taskmd"]`, `["TASKMD",".taskmd","MD","txt"]`, 1)
	source := filepath.Join(root, "a-route.json")
	if err := os.WriteFile(source, []byte(input), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := manager.InstallFile(source); err != nil {
		t.Fatal(err)
	}
	installed, err := os.ReadFile(filepath.Join(installRoot, "a-route.json"))
	if err != nil || string(installed) != input {
		t.Fatal("install changed manifest bytes", err)
	}
	assertMatch := func(path, id string) {
		t.Helper()
		match, err := manager.Match(path)
		if err != nil || match.ID != id {
			t.Fatalf("%s = %q %v; want %s", path, match.ID, err, id)
		}
	}
	assertMatch("a.TASKMD", "task-notes")
	assertMatch("a.md", "markdown")
	assertMatch("a.txt", "task-notes")
	manifest, _ := manager.Manifest("task-notes")
	wantExtensions := []string{".taskmd", ".md", ".txt"}
	if !reflect.DeepEqual(manifest.Extensions, wantExtensions) || !manifest.Enabled || !manifest.Active || !manifest.Declarative || manifest.Builtin {
		t.Fatal("normalized manifest metadata", manifest)
	}
	manifest.Extensions[0] = ".bad"
	for _, entry := range manager.Manifests() {
		if entry.ID == "task-notes" {
			if !reflect.DeepEqual(entry.Extensions, wantExtensions) {
				t.Fatal("Manifests returns raw or aliased suffixes", entry)
			}
			entry.Extensions[0] = ".also-bad"
		}
	}
	syntax, ok := manager.Syntax("task-notes")
	if !ok || syntax.NativeRoute != NativeRouteMarkdown || !reflect.DeepEqual(syntax.Extensions, []string{"TASKMD", ".taskmd", "MD", "txt"}) {
		t.Fatal("syntax must retain raw spelling", syntax)
	}
	syntax.Extensions[0], syntax.NativeRoute = "bad", NativeRouteMindmap
	again, _ := manager.Syntax("task-notes")
	if again.Extensions[0] != "TASKMD" || again.NativeRoute != NativeRouteMarkdown {
		t.Fatal("Syntax aliases manager state", again)
	}
	assertMatch("a.taskmd", "task-notes")
	if err := manager.SetEnabled("markdown", false); err != nil {
		t.Fatal(err)
	}
	assertMatch("a.md", "task-notes")
	if err := manager.SetEnabled("task-notes", false); err != nil {
		t.Fatal(err)
	}
	assertMatch("a.txt", "code")
	if _, err := manager.Match("a.taskmd"); err == nil || manager.Enabled("task-notes") {
		t.Fatal("disabled route still matches")
	}
	disabled, _ := manager.Syntax("task-notes")
	if disabled.NativeRoute != NativeRouteMarkdown {
		t.Fatal("disabled route syntax lost")
	}
	if err := manager.SetEnabled("task-notes", true); err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(input, "builtin.markdown", "builtin.mindmap", 1)
	if err := os.WriteFile(filepath.Join(installRoot, "a-route.json"), []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := manager.Refresh(); err != nil {
		t.Fatal(err)
	}
	refreshed, _ := manager.Syntax("task-notes")
	if refreshed.NativeRoute != NativeRouteMindmap {
		t.Fatal("same-ID refresh retained old route", refreshed)
	}
	assertMatch("a.taskmd", "task-notes")
	manager, err = NewManager(filepath.Join(root, "settings"), installRoot)
	if err != nil || manager.Enabled("markdown") || !manager.Enabled("task-notes") {
		t.Fatal("settings reload", err)
	}
	assertMatch("a.taskmd", "task-notes")
}

func TestSchema3InstallFailuresPreserveSelection(t *testing.T) {
	root := t.TempDir()
	manager, err := NewManager(filepath.Join(root, "settings"), filepath.Join(root, "installed"))
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "a-route.json")
	if err := os.WriteFile(source, []byte(schema3Literal), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := manager.InstallFile(source); err != nil {
		t.Fatal(err)
	}
	before := manager.Manifests()
	for _, sameID := range []bool{true, false} {
		name, input := "other.json", schema3Literal
		if !sameID {
			name = "a-route.json"
			input = strings.Replace(input, "task-notes", "second-route", 1)
		}
		dir := t.TempDir()
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := manager.InstallFile(path); err == nil || !reflect.DeepEqual(manager.Manifests(), before) {
			t.Fatal("exclusive install changed registry", sameID, err)
		}
	}
	// A directory at the settings-file path forces a real persistence failure.
	if err := os.MkdirAll(SettingsPath(filepath.Join(root, "settings")), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := manager.SetEnabled("task-notes", false); err == nil || !reflect.DeepEqual(manager.Manifests(), before) {
		t.Fatal("persistence refusal changed live selection", err)
	}
	// Fill the existing bounded loader; the new file is published but then
	// excluded by the real 256-manifest limit, exercising install rollback.
	for i := 1; i < maxInstalledPlugins; i++ {
		input := fmt.Sprintf(`{"schema":1,"id":"p-%03d","name":"Old","extensions":[".old%03d"]}`, i, i)
		if err := os.WriteFile(filepath.Join(root, "installed", fmt.Sprintf("b%03d.json", i)), []byte(input), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	manager.reload()
	before = manager.Manifests()
	newSource := filepath.Join(root, "z-overflow.json")
	if err := os.WriteFile(newSource, []byte(strings.Replace(schema3Literal, "task-notes", "overflow", 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := manager.InstallFile(newSource); err == nil || !strings.Contains(err.Error(), "failed to load") {
		t.Fatalf("post-publish refusal = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "installed", "z-overflow.json")); !os.IsNotExist(err) {
		t.Fatal("published failure was not removed", err)
	}
	if !reflect.DeepEqual(manager.Manifests(), before) || manager.Enabled("overflow") {
		t.Fatal("rollback changed registry selection")
	}
	if match, err := manager.Match("a.taskmd"); err != nil || match.ID != "task-notes" {
		t.Fatal("rollback lost original route", match, err)
	}
}

func TestSchema3EnableFailureSignalsOnlyReloadedState(t *testing.T) {
	for _, problem := range []string{"removed", "corrupt", "settings"} {
		t.Run(problem, func(t *testing.T) {
			root := t.TempDir()
			settings, installed := filepath.Join(root, "settings"), filepath.Join(root, "installed")
			manager, err := NewManager(settings, installed)
			if err != nil {
				t.Fatal(err)
			}
			source := filepath.Join(root, "route.json")
			if err := os.WriteFile(source, []byte(schema3Literal), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := manager.InstallFile(source); err != nil {
				t.Fatal(err)
			}
			switch problem {
			case "removed":
				if err := os.Remove(filepath.Join(installed, "route.json")); err != nil {
					t.Fatal(err)
				}
			case "corrupt":
				if err := os.WriteFile(filepath.Join(installed, "route.json"), []byte("{"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "settings":
				if err := os.MkdirAll(SettingsPath(settings), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			err = manager.SetEnabled("task-notes", true)
			if err == nil {
				t.Fatal("expected actual failure")
			}
			if errors.Is(fmt.Errorf("host wrapper: %w", err), ErrEnabledNotLoaded) != (problem != "settings") {
				t.Fatal("reload signal is not exact", err)
			}
			if manager.Enabled("task-notes") != (problem == "settings") {
				t.Fatal("failure state disagrees with mutation signal")
			}
		})
	}
}
