package dependencypatches

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBothPatchFilesAndAllVendoredChangesMatchPinnedManifest(t *testing.T) {
	manifestBytes, err := os.ReadFile("mygo-v0.8.3-patch-files.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Schema int    `json:"schema"`
		Module string `json:"module"`
		Files  []struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"files"`
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Schema != 1 || manifest.Module != mygoVersion {
		t.Fatal("wrong patch manifest pin")
	}
	wrapper, err := os.ReadFile(filepath.Join("..", "scripts", "go.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(manifestBytes)
	if !strings.Contains(string(wrapper), "$patchManifestSHA256 = '"+hex.EncodeToString(sum[:])+"'") {
		t.Fatal("wrapper does not pin exact patch manifest")
	}
	// Actual diff headers independently require every modified source/test file
	// in both patches to be inventoried exactly once.
	want := map[string]bool{}
	for _, name := range []string{"mygo-v0.8.3-no-startup-cleanup.patch", "mygo-v0.8.3-bitmap-dispose.patch"} {
		patch, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		want["dependency-patches/"+name] = true
		count := 0
		for _, line := range strings.Split(string(patch), "\n") {
			if path, ok := strings.CutPrefix(line, "+++ b/"); ok {
				if !strings.HasPrefix(path, "vendor/github.com/egoist/mygo/") || want[path] {
					t.Fatalf("unexpected/duplicate patch path %s", path)
				}
				want[path] = true
				count++
			}
		}
		if count == 0 {
			t.Fatalf("empty patch: %s", name)
		}
	}
	for _, file := range manifest.Files {
		if !want[file.Path] || filepath.IsAbs(file.Path) || strings.Contains(file.Path, "..") {
			t.Fatalf("unexpected/duplicate manifest path %s", file.Path)
		}
		delete(want, file.Path)
		data, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(file.Path)))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != file.SHA256 {
			t.Fatalf("incomplete MyGo patch: changed %s", file.Path)
		}
	}
	if len(want) != 0 {
		t.Fatalf("patch files absent from manifest: %v", want)
	}
}

const (
	mygoVersion     = "github.com/egoist/mygo v0.8.3"
	unsafeStartCall = "go cleanUpdateLeftovers()"
	patchMarker     = "// LEVNOTE PATCH: this app does not enable MyGo updates"
)

func TestVendoredMyGoStartupCleanupPatchIsPresent(t *testing.T) {
	t.Helper()
	source, err := os.ReadFile(filepath.Join("..", "vendor", "github.com", "egoist", "mygo", "app.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	if strings.Contains(text, unsafeStartCall) {
		t.Fatalf("vendored %s still starts %s; rerun dependency-patches/refresh.ps1", mygoVersion, unsafeStartCall)
	}
	if !strings.Contains(text, patchMarker) {
		t.Fatalf("vendored %s is missing the LevNote startup-safety marker; rerun dependency-patches/refresh.ps1", mygoVersion)
	}
}

func TestVendorManifestPinsMyGoAndLicensesRemain(t *testing.T) {
	t.Helper()
	modules, err := os.ReadFile(filepath.Join("..", "vendor", "modules.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(modules), "# "+mygoVersion) {
		t.Fatalf("vendor/modules.txt does not pin %s", mygoVersion)
	}
	for _, relative := range []string{
		filepath.Join("github.com", "egoist", "mygo", "LICENSE"),
		filepath.Join("github.com", "egoist", "mygo", "README.md"),
	} {
		if _, err := os.Stat(filepath.Join("..", "vendor", relative)); err != nil {
			t.Fatalf("vendored MyGo notice %s is missing: %v", relative, err)
		}
	}
}
