package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestBuiltinOrderAndFallback(t *testing.T) {
	registry := NewRegistry()
	tests := []struct {
		path string
		id   string
	}{
		{"README.MARKDOWN", "markdown"},
		{"notes/x.mdown", "markdown"},
		{"diagram/arch.mindmap", "mindmap"},
		{"diagram/arch.drawio", "drawio"},
		{"pictures/photo.png", "image"},
		{"pictures/photo.JPG", "image"},
		{"main.go", "code"},
	}
	for _, test := range tests {
		match, err := registry.Match(test.path)
		if err != nil {
			t.Fatalf("Match(%q): %v", test.path, err)
		}
		if match.ID != test.id {
			t.Errorf("Match(%q) = %q; want %q", test.path, match.ID, test.id)
		}
	}
}

func TestManagerLoadsAndPersistsInstalledPlugin(t *testing.T) {
	root := t.TempDir()
	manifest := `{"schema":1,"id":"toml","name":"TOML","version":"1.0.0","apiVersion":1,"extensions":[".toml"],"lineComment":"#","keywords":["true"]}`
	if err := os.WriteFile(filepath.Join(root, "toml.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	manager, err := NewManager(root, root)
	if err != nil {
		t.Fatal(err)
	}
	match, err := manager.Match("settings.toml")
	if err != nil || match.ID != "toml" {
		t.Fatalf("toml match = %#v, %v", match, err)
	}
	if err := manager.SetEnabled("toml", false); err != nil {
		t.Fatal(err)
	}
	var listed Manifest
	for _, candidate := range manager.Manifests() {
		if candidate.ID == "toml" {
			listed = candidate
			break
		}
	}
	if listed.ID != "toml" || listed.Enabled || listed.Active || listed.SourcePath == "" {
		t.Fatalf("disabled plugin must remain listed as inactive: %#v", listed)
	}
	syntax, ok := manager.Syntax("toml")
	if !ok || syntax.LineComment != "#" || len(syntax.Keywords) != 1 || syntax.Keywords[0] != "true" {
		t.Fatalf("disabled plugin lost declarative syntax: %#v, %v", syntax, ok)
	}
	if match, err := manager.Match("settings.toml"); err != nil || match.ID != "code" {
		t.Fatalf("disabled plugin did not fall back to code: %#v, %v", match, err)
	}
	if err := manager.SetEnabled("missing-plugin", true); err == nil {
		t.Fatal("unknown plugin ID must not be persisted")
	}
	if err := manager.SetEnabled("markdown", false); err != nil {
		t.Fatal(err)
	}
	if manager.Enabled("markdown") {
		t.Fatal("disabled builtin must not be enabled")
	}
	if match, err := manager.Match("README.md"); err != nil || match.ID != "code" {
		t.Fatalf("disabled Markdown did not fall back to code: %#v, %v", match, err)
	}
	for _, manifest := range manager.Manifests() {
		if manifest.ID == "markdown" && (manifest.Enabled || !manifest.Builtin) {
			t.Fatalf("disabled builtin disappeared or stayed enabled: %#v", manifest)
		}
	}
	if err := manager.SetEnabled("markdown", true); err != nil {
		t.Fatal(err)
	}
	if !manager.Enabled("markdown") {
		t.Fatal("reenabled builtin must match again")
	}
	if err := manager.SetEnabled("toml", true); err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewManager(root, root)
	if err != nil {
		t.Fatal(err)
	}
	if match, err := reloaded.Match("settings.toml"); err != nil || match.ID != "toml" {
		t.Fatalf("enabled plugin did not survive reload: %#v, %v", match, err)
	}
	if err := os.Remove(filepath.Join(root, "plugins.json")); err != nil {
		t.Fatal(err)
	}
	if len(reloaded.LoadErrors()) != 0 {
		t.Fatal("reloaded manager retained stale errors")
	}
}

func TestImageBuiltinCapabilitiesAndManagerPersistence(t *testing.T) {
	settingsRoot := t.TempDir()
	installRoot := t.TempDir()
	manager, err := NewManager(settingsRoot, installRoot)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := manager.Manifest("image")
	if err != nil {
		t.Fatal(err)
	}
	want := Manifest{
		ID: "image", Name: "Native Image Reader",
		Extensions: []string{".png", ".jpg", ".jpeg"},
		Version:    "1.0.0", APIVersion: APIVersion,
		Caps:  Capabilities{Read: true, Edit: false, Source: true, Export: false},
		Order: 4, Enabled: true, Active: true, Builtin: true,
	}
	if !reflect.DeepEqual(manifest, want) {
		t.Fatalf("image manifest = %#v; want %#v", manifest, want)
	}
	for _, path := range []string{"a.png", "a.PNG", "b.jpg", "b.Jpg", "c.jpeg"} {
		match, err := manager.Match(path)
		if err != nil || match.ID != "image" {
			t.Fatalf("Match(%q) = %#v, %v; want image", path, match, err)
		}
	}
	if err := manager.SetEnabled("image", false); err != nil {
		t.Fatal(err)
	}
	if manager.Enabled("image") {
		t.Fatal("disabled image builtin remained active")
	}
	if _, err := manager.Match("a.png"); err == nil {
		t.Fatal("disabled image builtin still claimed .png")
	}
	if err := manager.SetEnabled("image", true); err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewManager(settingsRoot, installRoot)
	if err != nil {
		t.Fatal(err)
	}
	if !reloaded.Enabled("image") {
		t.Fatal("image enablement did not survive manager reload")
	}
	if match, err := reloaded.Match("a.jpeg"); err != nil || match.ID != "image" {
		t.Fatalf("reload image match = %#v, %v", match, err)
	}
}

func TestBundledDeclarativePluginExampleIsAccepted(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "plugins", "toml.json"))
	if err != nil {
		t.Fatal(err)
	}
	syntax, err := decodeManifest(data)
	if err != nil {
		t.Fatalf("bundled example is invalid: %v", err)
	}
	if syntax.ID != "toml" || len(syntax.Extensions) != 1 || syntax.Extensions[0] != ".toml" {
		t.Fatalf("bundled example decoded as %#v", syntax)
	}
}

func TestHostileManifestRejected(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "bad.json"), []byte(`{"schema":1,"id":"bad id","name":"x","apiVersion":1,"extensions":[".x"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	result := LoadInstallDir(root)
	if len(result.Manifests) != 0 || len(result.Errors) != 1 {
		t.Fatalf("result = %#v", result)
	}
}

func TestManifestWithoutExtensionsIsRejected(t *testing.T) {
	data := []byte(`{"schema":1,"id":"empty","name":"Empty","version":"1","apiVersion":1,"extensions":[]}`)
	if _, err := decodeManifest(data); err == nil {
		t.Fatal("plugin with no extensions was accepted")
	}
}

func TestInstalledPrioritySitsBeforeCodeButAfterSpecificBuiltins(t *testing.T) {
	root := t.TempDir()
	goPlugin := `{"schema":1,"id":"go-syntax","name":"Go syntax","version":"1.0.0","apiVersion":1,"extensions":[".go"],"lineComment":"//","keywords":["func"]}`
	markdownPlugin := `{"schema":1,"id":"notes","name":"Notes","version":"1.0.0","apiVersion":1,"extensions":[".md"],"lineComment":">","keywords":["note"]}`
	if err := os.WriteFile(filepath.Join(root, "a-go.json"), []byte(goPlugin), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "b-notes.json"), []byte(markdownPlugin), 0o600); err != nil {
		t.Fatal(err)
	}
	manager, err := NewManager(root, root)
	if err != nil {
		t.Fatal(err)
	}
	if match, err := manager.Match("main.go"); err != nil || match.ID != "go-syntax" {
		t.Fatalf("installed syntax plugin did not precede Code fallback: %#v, %v", match, err)
	}
	if match, err := manager.Match("README.md"); err != nil || match.ID != "markdown" {
		t.Fatalf("installed fallback overrode Markdown: %#v, %v", match, err)
	}
}

func TestInstalledPriorityDoesNotFallBehindCodeAfterManyPlugins(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 96; i++ {
		manifest := fmt.Sprintf(
			`{"schema":1,"id":"syntax-%03d","name":"Syntax","version":"1","apiVersion":1,"extensions":[".s%03d"]}`,
			i, i,
		)
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("a%03d.json", i)), []byte(manifest), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	manifest := `{"schema":1,"id":"go-syntax","name":"Go syntax","version":"1","apiVersion":1,"extensions":[".go"],"lineComment":"//"}`
	if err := os.WriteFile(filepath.Join(root, "z-go.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	manager, err := NewManager(filepath.Join(root, "settings"), root)
	if err != nil {
		t.Fatal(err)
	}
	if match, err := manager.Match("main.go"); err != nil || match.ID != "go-syntax" {
		t.Fatalf("late installed syntax was shadowed by Code: %#v, %v", match, err)
	}
}

func TestManifestLoadErrorsRemainVisible(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "broken.json"), []byte(`{"schema":1`), 0o600); err != nil {
		t.Fatal(err)
	}
	manager, err := NewManager(root, root)
	if err != nil {
		t.Fatal(err)
	}
	errors := manager.LoadErrors()
	if len(errors) != 1 || errors[0].Path == "" || errors[0].Message == "" {
		t.Fatalf("load errors = %#v", errors)
	}
}

func TestBrokenSettingsDoNotHideInstalledPlugin(t *testing.T) {
	root := t.TempDir()
	manifest := `{"schema":1,"id":"toml","name":"TOML","version":"1.0.0","apiVersion":1,"extensions":[".toml"],"lineComment":"#"}`
	if err := os.WriteFile(filepath.Join(root, "toml.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(SettingsPath(root), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	manager, err := NewManager(root, root)
	if err != nil {
		t.Fatal(err)
	}
	if match, err := manager.Match("settings.toml"); err != nil || match.ID != "toml" {
		t.Fatalf("installed plugin was hidden by settings failure: %#v, %v", match, err)
	}
	if len(manager.LoadErrors()) == 0 {
		t.Fatal("settings failure must be reported")
	}
	if err := os.WriteFile(SettingsPath(root), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := manager.Refresh(); err != nil {
		t.Fatal(err)
	}
	if len(manager.LoadErrors()) != 0 {
		t.Fatalf("successful rescan retained stale errors: %#v", manager.LoadErrors())
	}
}

func TestOversizeManifestRejectedBeforeDecode(t *testing.T) {
	root := t.TempDir()
	data := make([]byte, maxManifestBytes+1)
	if err := os.WriteFile(filepath.Join(root, "large.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	result := LoadInstallDir(root)
	if len(result.Manifests) != 0 || len(result.Errors) != 1 {
		t.Fatalf("oversize result = %#v", result)
	}
}

func TestManifestAndSyntaxReturnCopies(t *testing.T) {
	root := t.TempDir()
	manifest := `{"schema":1,"id":"toml","name":"TOML","version":"1.0.0","apiVersion":1,"extensions":[".toml"],"keywords":["true"]}`
	if err := os.WriteFile(filepath.Join(root, "toml.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	manager, err := NewManager(t.TempDir(), root)
	if err != nil {
		t.Fatal(err)
	}
	manifests := manager.Manifests()
	for i := range manifests {
		if manifests[i].ID == "toml" {
			manifests[i].Extensions[0] = ".mutated"
		}
	}
	syntax, ok := manager.Syntax("toml")
	if !ok {
		t.Fatal("syntax missing")
	}
	syntax.Extensions[0] = ".mutated"
	syntax.Keywords[0] = "mutated"
	if match, err := manager.Match("settings.toml"); err != nil || match.ID != "toml" {
		t.Fatalf("manifest slice mutation changed matching: %#v, %v", match, err)
	}
	again, _ := manager.Syntax("toml")
	if again.Extensions[0] != ".toml" || again.Keywords[0] != "true" {
		t.Fatalf("syntax slice mutation changed definitions: %#v", again)
	}
	match, err := manager.Match("settings.toml")
	if err != nil {
		t.Fatal(err)
	}
	match.Manifest.Extensions[0] = ".mutated"
	if again, err := manager.Match("settings.toml"); err != nil || again.Manifest.Extensions[0] != ".toml" {
		t.Fatalf("match slice mutation changed matching: %#v, %v", again, err)
	}
}

func TestEqualPriorityPreservesRegistrationOrder(t *testing.T) {
	registry := NewRegistry()
	base := Manifest{
		Name: "same-priority", Extensions: []string{".same"}, Version: "1.0.0",
		APIVersion: APIVersion, Order: 7, Enabled: true, Active: true,
	}
	first := base
	first.ID = "z-first"
	second := base
	second.ID = "a-second"
	if err := registry.Register(first, true); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(second, true); err != nil {
		t.Fatal(err)
	}
	match, err := registry.Match("value.same")
	if err != nil || match.ID != first.ID {
		t.Fatalf("equal priority changed registration order: %#v, %v", match, err)
	}
}

func TestRegistryDoesNotRetainCallerExtensionSlice(t *testing.T) {
	registry := NewRegistry()
	extensions := []string{".alias"}
	manifest := Manifest{
		ID: "alias", Name: "Alias", Extensions: extensions, Version: "1",
		APIVersion: APIVersion, Order: 7, Enabled: true, Active: true,
	}
	if err := registry.Register(manifest, true); err != nil {
		t.Fatal(err)
	}
	extensions[0] = ".mutated"
	if match, err := registry.Match("value.alias"); err != nil || match.ID != "alias" {
		t.Fatalf("caller mutation changed registry matching: %#v, %v", match, err)
	}
	manifest.Extensions[0] = ".mutated"
	match, err := registry.Match("value.alias")
	if err != nil {
		t.Fatal(err)
	}
	match.Manifest.Extensions[0] = ".mutated"
	listed := registry.Manifests()
	listed[0].Extensions[0] = ".mutated"
	if again, err := registry.Match("value.alias"); err != nil || again.Manifest.Extensions[0] != ".alias" {
		t.Fatalf("returned/source slice mutation changed registry: %#v, %v", again, err)
	}
}

func TestDuplicateAndReservedIDsRemainVisible(t *testing.T) {
	root := t.TempDir()
	first := `{"schema":1,"id":"duplicate","name":"First","version":"1","apiVersion":1,"extensions":[".dup"]}`
	second := `{"schema":1,"id":"duplicate","name":"Second","version":"1","apiVersion":1,"extensions":[".other"]}`
	reserved := `{"schema":1,"id":"markdown","name":"Fake","version":"1","apiVersion":1,"extensions":[".fake"]}`
	if err := os.WriteFile(filepath.Join(root, "a.json"), []byte(first), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "b.json"), []byte(second), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "c.json"), []byte(reserved), 0o600); err != nil {
		t.Fatal(err)
	}
	manager, err := NewManager(t.TempDir(), root)
	if err != nil {
		t.Fatal(err)
	}
	if match, err := manager.Match("value.dup"); err != nil || match.ID != "duplicate" {
		t.Fatalf("first duplicate did not remain authoritative: %#v, %v", match, err)
	}
	if _, err := manager.Match("value.other"); err == nil {
		t.Fatal("colliding duplicate ID activated its syntax")
	}
	if match, err := manager.Match("README.md"); err != nil || match.ID != "markdown" {
		t.Fatalf("reserved builtin ID was replaced: %#v, %v", match, err)
	}
	if len(manager.LoadErrors()) < 2 {
		t.Fatalf("duplicate and reserved collisions were hidden: %#v", manager.LoadErrors())
	}
}

func TestInstallFileIsExclusiveAndRejectsExistingIDs(t *testing.T) {
	root := t.TempDir()
	installRoot := filepath.Join(root, "installed")
	settingsRoot := filepath.Join(root, "settings")
	manager, err := NewManager(settingsRoot, installRoot)
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "toml.json")
	manifest := `{"schema":1,"id":"toml","name":"TOML","version":"1","apiVersion":1,"extensions":[".toml"],"keywords":["true"]}`
	if err := os.WriteFile(source, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := manager.InstallFile(source); err != nil {
		t.Fatal(err)
	}
	if match, err := manager.Match("settings.toml"); err != nil || match.ID != "toml" {
		t.Fatalf("installed file did not load: %#v, %v", match, err)
	}
	if err := manager.InstallFile(source); err == nil {
		t.Fatal("installing the same file again must fail")
	}

	other := filepath.Join(root, "duplicate-id.json")
	if err := os.WriteFile(other, []byte(strings.ReplaceAll(manifest, `"toml.json"`, `"other.json"`)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := manager.InstallFile(other); err == nil {
		t.Fatal("installing a duplicate plugin ID must fail")
	}

	collisionDir := filepath.Join(root, "collision")
	if err := os.MkdirAll(collisionDir, 0o700); err != nil {
		t.Fatal(err)
	}
	collisionSource := filepath.Join(collisionDir, "toml.json")
	collisionManifest := `{"schema":1,"id":"other","name":"Other","version":"1","apiVersion":1,"extensions":[".other"]}`
	if err := os.WriteFile(collisionSource, []byte(collisionManifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := manager.InstallFile(collisionSource); err == nil {
		t.Fatal("existing destination file must not be overwritten")
	}
}

func TestInstallFileSucceedsWithPreexistingLoadErrors(t *testing.T) {
	root := t.TempDir()
	installRoot := filepath.Join(root, "installed")
	if err := os.MkdirAll(installRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installRoot, "broken.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	manager, err := NewManager(filepath.Join(root, "settings"), installRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(manager.LoadErrors()) != 1 {
		t.Fatalf("initial errors = %#v", manager.LoadErrors())
	}
	source := filepath.Join(root, "valid.json")
	manifest := `{"schema":1,"id":"valid","name":"Valid","version":"1","apiVersion":1,"extensions":[".valid"]}`
	if err := os.WriteFile(source, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := manager.InstallFile(source); err != nil {
		t.Fatalf("valid install was rejected because an unrelated load error remained: %v", err)
	}
	if match, err := manager.Match("example.valid"); err != nil || match.ID != "valid" {
		t.Fatalf("installed plugin is not active: %#v, %v", match, err)
	}
	if len(manager.LoadErrors()) != 1 {
		t.Fatalf("unrelated error changed during install: %#v", manager.LoadErrors())
	}
}

func TestInstallFileRollsBackWhenManifestLimitRejectsNewPlugin(t *testing.T) {
	root := t.TempDir()
	installRoot := filepath.Join(root, "installed")
	if err := os.MkdirAll(installRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < maxInstalledPlugins; i++ {
		id := fmt.Sprintf("plugin-%03d", i)
		manifest := fmt.Sprintf(
			`{"schema":1,"id":%q,"name":"Plugin","version":"1","apiVersion":1,"extensions":[".p%03d"]}`,
			id, i,
		)
		if err := os.WriteFile(filepath.Join(installRoot, fmt.Sprintf("a%03d.json", i)), []byte(manifest), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	manager, err := NewManager(filepath.Join(root, "settings"), installRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(manager.LoadErrors()) != 0 {
		t.Fatalf("initial plugin set should be valid: %#v", manager.LoadErrors())
	}

	source := filepath.Join(root, "overflow.json")
	manifest := `{"schema":1,"id":"overflow","name":"Overflow","version":"1","apiVersion":1,"extensions":[".overflow"]}`
	if err := os.WriteFile(source, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := manager.InstallFile(source); err == nil {
		t.Fatal("install succeeded after exceeding the manifest limit")
	}
	destination := filepath.Join(installRoot, "overflow.json")
	if _, err := os.Lstat(destination); !os.IsNotExist(err) {
		t.Fatalf("failed install was not rolled back: %v", err)
	}
	if manager.Enabled("overflow") {
		t.Fatal("failed install remained active after rollback")
	}
}

func TestFailedSettingsPersistenceDoesNotMutateLiveState(t *testing.T) {
	root := t.TempDir()
	installRoot := filepath.Join(root, "installed")
	if err := os.MkdirAll(installRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := `{"schema":1,"id":"toml","name":"TOML","version":"1","apiVersion":1,"extensions":[".toml"]}`
	if err := os.WriteFile(filepath.Join(installRoot, "toml.json"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	settingsRoot := filepath.Join(root, "settings-file")
	if err := os.WriteFile(settingsRoot, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	manager, err := NewManager(settingsRoot, installRoot)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.SetEnabled("toml", false); err == nil {
		t.Fatal("expected settings persistence failure")
	}
	if !manager.Enabled("toml") {
		t.Fatal("failed persistence changed live plugin state")
	}
	if match, err := manager.Match("settings.toml"); err != nil || match.ID != "toml" {
		t.Fatalf("failed persistence changed matching: %#v, %v", match, err)
	}
}

func TestSettingsAreBoundedAndIDsValidated(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(SettingsPath(root), make([]byte, maxSettingsBytes+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSettings(root); err == nil {
		t.Fatal("oversize settings must be rejected")
	}
	if err := os.WriteFile(SettingsPath(root), []byte(`{"disabled":["Bad ID"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSettings(root); err == nil {
		t.Fatal("invalid plugin ID in settings must be rejected")
	}
}

func TestInstallDirectoryRejectsSymlinksAndNonRegularManifests(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "directory.json"), 0o700); err != nil {
		t.Fatal(err)
	}
	result := LoadInstallDir(root)
	if len(result.Errors) != 1 {
		t.Fatalf("manifest directory was not reported: %#v", result.Errors)
	}
	target := filepath.Join(t.TempDir(), "target.json")
	manifest := `{"schema":1,"id":"target","name":"Target","version":"1","apiVersion":1,"extensions":[".target"]}`
	if err := os.WriteFile(target, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "linked.json")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	result = LoadInstallDir(root)
	found := false
	for _, issue := range result.Errors {
		if issue.Path == link && strings.Contains(issue.Message, "symbolic links") {
			found = true
		}
	}
	if !found {
		t.Fatalf("manifest symlink was not reported: %#v", result.Errors)
	}
}

func TestInstallDirectoryLimitUsesExtraEntryProbe(t *testing.T) {
	exact := t.TempDir()
	for i := 0; i < maxInstallDirEntries; i++ {
		if err := os.WriteFile(filepath.Join(exact, fmt.Sprintf("%04d.txt", i)), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	result := LoadInstallDir(exact)
	if len(result.Errors) != 0 {
		t.Fatalf("exactly %d entries was incorrectly truncated: %#v", maxInstallDirEntries, result.Errors)
	}

	overflow := t.TempDir()
	for i := 0; i <= maxInstallDirEntries; i++ {
		if err := os.WriteFile(filepath.Join(overflow, fmt.Sprintf("%04d.txt", i)), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	result = LoadInstallDir(overflow)
	if len(result.Errors) != 1 || !strings.Contains(result.Errors[0].Message, "truncated") {
		t.Fatalf("entry overflow was not reported as truncation: %#v", result.Errors)
	}
}

func TestPluginRootsRejectSymlinks(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "target")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "linked")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	manager, err := NewManager(link, target)
	if err != nil {
		t.Fatal(err)
	}
	if len(manager.LoadErrors()) == 0 {
		t.Fatal("settings root symlink was not reported")
	}
	if _, err := NewManager(target, link); err == nil {
		t.Fatal("install root symlink was accepted")
	}
	if err := SaveSettings(link, Settings{Disabled: map[string]bool{}}); err == nil {
		t.Fatal("settings save followed a symlinked root")
	}
}
