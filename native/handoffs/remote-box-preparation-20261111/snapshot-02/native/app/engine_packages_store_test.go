//go:build windows

package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"levnote.local/native/core/enginecatalog"
	"levnote.local/native/core/engineipc"
	"levnote.local/native/core/engineruntime"
)

// These are the exact fifteen authored/attribution paths in the fixed
// eighteen-file store contract. Keeping the oracle test-owned means a future
// production omission cannot silently weaken the app fixture.
var enginePackageFixtureNotices = [...]string{
	"LICENSE",
	"README.md",
	"THIRD_PARTY_NOTICES.md",
	"attribution/SOURCE_THIRD_PARTY_NOTICES.txt",
	"attribution/mygo-README.md",
	"attribution/purego-README.md",
	"attribution/mygo-v0.8.3-no-startup-cleanup.patch",
	"attribution/dependency-patches-README.md",
	"licenses/Go-BSD-3-Clause.txt",
	"licenses/go-text-typesetting.txt",
	"licenses/mygo-MIT.txt",
	"licenses/purego-Apache-2.0.txt",
	"licenses/x-image-BSD-3-Clause.txt",
	"licenses/x-net-BSD-3-Clause.txt",
	"licenses/x-text-BSD-3-Clause.txt",
}

type enginePackageFixtureFile struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

func enginePackageFixtureHash(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func enginePackageFixtureWrite(t *testing.T, root, relative string, data []byte, entries *[]enginePackageFixtureFile) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("create fixture directory for %s: %v", relative, err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write fixture file %s: %v", relative, err)
	}
	*entries = append(*entries, enginePackageFixtureFile{
		Path: relative, Size: int64(len(data)), SHA256: enginePackageFixtureHash(data),
	})
}

// enginePackageFixture creates a complete, inert release directory matching
// the fixed eighteen-file package store contract. The executable is text and
// is never executed by these tests.
func enginePackageFixture(t *testing.T, engine enginecatalog.EngineID, version string) string {
	t.Helper()
	descriptor, ok := enginecatalog.Lookup(engine)
	if !ok {
		t.Fatalf("fixture engine %q is outside the fixed catalog", engine)
	}
	if version == "" || len(version) > engineipc.EngineVersionMaxBytes {
		t.Fatalf("fixture version %q is outside the engine version bounds", version)
	}

	root := t.TempDir()
	entries := make([]enginePackageFixtureFile, 0, 17)
	for _, name := range enginePackageFixtureNotices {
		enginePackageFixtureWrite(t, root, name, []byte("authored attribution: "+name+"\r\n"), &entries)
	}

	executable := []byte("inert executable text; never executed: " + string(engine) + ":" + version + "\r\n")
	enginePackageFixtureWrite(t, root, descriptor.RelativePath+"/"+descriptor.Filename, executable, &entries)

	manifest, err := json.Marshal(struct {
		Schema        int                    `json:"schema"`
		Engine        enginecatalog.EngineID `json:"engine"`
		EngineVersion string                 `json:"engineVersion"`
		Protocol      int                    `json:"protocol"`
		Format        string                 `json:"format"`
		GOOS          string                 `json:"goos"`
		GOARCH        string                 `json:"goarch"`
		Filename      string                 `json:"filename"`
		Size          int64                  `json:"size"`
		SHA256        string                 `json:"sha256"`
	}{
		Schema: 1, Engine: engine, EngineVersion: version, Protocol: descriptor.Protocol,
		Format: descriptor.Format, GOOS: descriptor.GOOS, GOARCH: descriptor.GOARCH,
		Filename: descriptor.Filename, Size: int64(len(executable)), SHA256: enginePackageFixtureHash(executable),
	})
	if err != nil {
		t.Fatalf("encode fixture manifest: %v", err)
	}
	enginePackageFixtureWrite(t, root, descriptor.RelativePath+"/manifest.json", manifest, &entries)

	inventory, err := json.Marshal(struct {
		Schema        int                        `json:"schema"`
		Version       string                     `json:"version"`
		GOOS          string                     `json:"goos"`
		GOARCH        string                     `json:"goarch"`
		GOAMD64       string                     `json:"goamd64"`
		Excludes      []string                   `json:"excludes"`
		Scope         string                     `json:"scope"`
		Configuration string                     `json:"configuration"`
		Files         []enginePackageFixtureFile `json:"files"`
	}{
		Schema: 1, Version: version, GOOS: "windows", GOARCH: "amd64", GOAMD64: "v1",
		Excludes: []string{"release-manifest.json"}, Scope: "native app package fixture",
		Configuration: "windows amd64 release fixture", Files: entries,
	})
	if err != nil {
		t.Fatalf("encode fixture inventory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "release-manifest.json"), inventory, 0o600); err != nil {
		t.Fatalf("write fixture release inventory: %v", err)
	}
	return root
}

const enginePackageTestWait = 10 * time.Second

func runEnginePackageJob(t *testing.T, a *App, action packageAction, engine enginecatalog.EngineID, value string) *packageJob {
	t.Helper()
	a.startPackageWork(action, engine, value)
	job := a.packageJob
	if job == nil {
		t.Fatalf("package action %q did not start: message=%q", action, a.Message)
	}
	timer := time.NewTimer(enginePackageTestWait)
	defer timer.Stop()
	select {
	case <-job.done:
	case <-timer.C:
		t.Fatalf("package action %q did not finish within %s", action, enginePackageTestWait)
	}
	a.deliverPackageWork()
	if a.packageJob != nil {
		t.Fatalf("package action %q completed but delivery did not release the result slot", action)
	}
	return job
}

func enginePackageStatus(t *testing.T, a *App, engine enginecatalog.EngineID) engineruntime.PackageStatus {
	t.Helper()
	for _, status := range a.packageStatuses {
		if status.Engine == engine {
			return status
		}
	}
	t.Fatalf("package status missing for %s: %#v", engine, a.packageStatuses)
	return engineruntime.PackageStatus{}
}

func assertEnginePackageBundled(t *testing.T, a *App, engine enginecatalog.EngineID) {
	t.Helper()
	status := enginePackageStatus(t, a, engine)
	if status.Source != engineruntime.PackageBundled || status.Digest != "" || status.Version != "" {
		t.Fatalf("%s is not bundled: %#v", engine, status)
	}
}

func assertEnginePackageInstalled(t *testing.T, a *App, engine enginecatalog.EngineID, digest, version string) {
	t.Helper()
	status := enginePackageStatus(t, a, engine)
	if status.Source != engineruntime.PackageInstalled || status.Digest != digest || status.Version != version {
		t.Fatalf("%s installed status = %#v; want digest=%s version=%s", engine, status, digest, version)
	}
}

func enginePackageCachedReceipt(t *testing.T, a *App, engine enginecatalog.EngineID, version string) engineruntime.PackageReceipt {
	t.Helper()
	list := a.packageLists[engine]
	for _, receipt := range list.receipts {
		if receipt.Version == version {
			return receipt
		}
	}
	t.Fatalf("cached receipt for %s version %q is missing: %#v", engine, version, list)
	return engineruntime.PackageReceipt{}
}

func TestEnginePackageStoreImportActivateResetAndRestart(t *testing.T) {
	type fixtureCase struct {
		engine  enginecatalog.EngineID
		version string
		release string
	}
	fixtures := []fixtureCase{
		{enginecatalog.MarkdownHTML, "app.import.1", enginePackageFixture(t, enginecatalog.MarkdownHTML, "app.import.1")},
		{enginecatalog.MindmapSVG, "app.import.2", enginePackageFixture(t, enginecatalog.MindmapSVG, "app.import.2")},
		{enginecatalog.MindmapPreview, "app.import.3", enginePackageFixture(t, enginecatalog.MindmapPreview, "app.import.3")},
	}

	config := t.TempDir()
	a := newTestApp(t, config)
	receipts := make(map[enginecatalog.EngineID]engineruntime.PackageReceipt, len(fixtures))
	for _, item := range fixtures {
		runEnginePackageJob(t, a, packageImportAction, item.engine, item.release)
		receipt := enginePackageCachedReceipt(t, a, item.engine, item.version)
		if receipt.Engine != item.engine || len(receipt.Digest) != 64 || len(receipt.Files) != 18 {
			t.Fatalf("cached import receipt = %#v", receipt)
		}
		if !strings.Contains(a.packageOutcome, "尚未启用") {
			t.Fatalf("import outcome claimed activation: %q", a.packageOutcome)
		}
		receipts[item.engine] = receipt
	}
	if _, err := os.Stat(filepath.Join(config, "engine-packages", "selections.json")); !os.IsNotExist(err) {
		t.Fatalf("import wrote a selection state: %v", err)
	}
	for _, item := range fixtures {
		assertEnginePackageBundled(t, a, item.engine)
	}

	runEnginePackageJob(t, a, packageActivateAction, enginecatalog.MarkdownHTML, receipts[enginecatalog.MarkdownHTML].Digest)
	assertEnginePackageInstalled(t, a, enginecatalog.MarkdownHTML, receipts[enginecatalog.MarkdownHTML].Digest, receipts[enginecatalog.MarkdownHTML].Version)
	assertEnginePackageBundled(t, a, enginecatalog.MindmapSVG)
	assertEnginePackageBundled(t, a, enginecatalog.MindmapPreview)

	runEnginePackageJob(t, a, packageActivateAction, enginecatalog.MindmapPreview, receipts[enginecatalog.MindmapPreview].Digest)
	assertEnginePackageInstalled(t, a, enginecatalog.MarkdownHTML, receipts[enginecatalog.MarkdownHTML].Digest, receipts[enginecatalog.MarkdownHTML].Version)
	assertEnginePackageBundled(t, a, enginecatalog.MindmapSVG)
	assertEnginePackageInstalled(t, a, enginecatalog.MindmapPreview, receipts[enginecatalog.MindmapPreview].Digest, receipts[enginecatalog.MindmapPreview].Version)

	runEnginePackageJob(t, a, packageResetAction, enginecatalog.MarkdownHTML, "")
	assertEnginePackageBundled(t, a, enginecatalog.MarkdownHTML)
	assertEnginePackageBundled(t, a, enginecatalog.MindmapSVG)
	assertEnginePackageInstalled(t, a, enginecatalog.MindmapPreview, receipts[enginecatalog.MindmapPreview].Digest, receipts[enginecatalog.MindmapPreview].Version)

	restored := newTestApp(t, config)
	if restored.packageManager == nil || restored.packageManager.LoadError() != nil {
		t.Fatalf("restored package manager is not usable: manager=%v error=%v", restored.packageManager, restored.packageManager.LoadError())
	}
	assertEnginePackageBundled(t, restored, enginecatalog.MarkdownHTML)
	assertEnginePackageBundled(t, restored, enginecatalog.MindmapSVG)
	assertEnginePackageInstalled(t, restored, enginecatalog.MindmapPreview, receipts[enginecatalog.MindmapPreview].Digest, receipts[enginecatalog.MindmapPreview].Version)

	runEnginePackageJob(t, restored, packageListAction, enginecatalog.MindmapPreview, "")
	list := restored.packageLists[enginecatalog.MindmapPreview]
	if !list.checked || list.err != nil || len(list.receipts) != 1 || list.receipts[0].Digest != receipts[enginecatalog.MindmapPreview].Digest {
		t.Fatalf("restored preview package list = %#v", list)
	}
}

func TestEnginePackageStoreListReturnsHealthyPackagesAndCorruption(t *testing.T) {
	good := enginePackageFixture(t, enginecatalog.MarkdownHTML, "app.list.good")
	bad := enginePackageFixture(t, enginecatalog.MarkdownHTML, "app.list.bad")
	config := t.TempDir()
	a := newTestApp(t, config)

	runEnginePackageJob(t, a, packageImportAction, enginecatalog.MarkdownHTML, good)
	goodReceipt := enginePackageCachedReceipt(t, a, enginecatalog.MarkdownHTML, "app.list.good")
	runEnginePackageJob(t, a, packageImportAction, enginecatalog.MarkdownHTML, bad)
	badReceipt := enginePackageCachedReceipt(t, a, enginecatalog.MarkdownHTML, "app.list.bad")

	corruptPath := filepath.Join(config, "engine-packages", string(enginecatalog.MarkdownHTML), badReceipt.Digest, "LICENSE")
	if err := os.WriteFile(corruptPath, []byte("tampered fixture notice\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	runEnginePackageJob(t, a, packageListAction, enginecatalog.MarkdownHTML, "")
	list := a.packageLists[enginecatalog.MarkdownHTML]
	if !list.checked || list.err == nil {
		t.Fatalf("corrupt installed package was hidden: %#v", list)
	}
	if len(list.receipts) != 1 || list.receipts[0].Digest != goodReceipt.Digest {
		t.Fatalf("healthy package receipt was not retained: %#v", list.receipts)
	}
	if !strings.Contains(a.packageOutcome, "未完整完成") || !strings.Contains(a.packageOutcome, badReceipt.Digest) {
		t.Fatalf("list outcome does not expose corruption: %q", a.packageOutcome)
	}
}

func TestEnginePackageStoreCorruptStateStaysBlockedUntilRepairRetainsPackage(t *testing.T) {
	release := enginePackageFixture(t, enginecatalog.MarkdownHTML, "app.repair.keeps")
	config := t.TempDir()
	a := newTestApp(t, config)
	runEnginePackageJob(t, a, packageImportAction, enginecatalog.MarkdownHTML, release)
	receipt := enginePackageCachedReceipt(t, a, enginecatalog.MarkdownHTML, "app.repair.keeps")

	statePath := filepath.Join(config, "engine-packages", "selections.json")
	corrupt := []byte(`{"schema":1,"selections":[`)
	if err := os.WriteFile(statePath, corrupt, 0o600); err != nil {
		t.Fatal(err)
	}
	blocked := newTestApp(t, config)
	if blocked.packageManager == nil || blocked.packageManager.LoadError() == nil {
		t.Fatalf("corrupt selection state was not reported: manager=%v", blocked.packageManager)
	}

	job := runEnginePackageJob(t, blocked, packageActivateAction, enginecatalog.MarkdownHTML, receipt.Digest)
	job.mu.Lock()
	jobErr := job.err
	job.mu.Unlock()
	if jobErr == nil {
		t.Fatal("activation under corrupt selection state unexpectedly succeeded")
	}
	if blocked.packageManager.LoadError() == nil || blocked.enginePoison != nil {
		t.Fatalf("failed activation changed block/poison truthfully: load=%v poison=%v", blocked.packageManager.LoadError(), blocked.enginePoison)
	}
	assertEnginePackageBundled(t, blocked, enginecatalog.MarkdownHTML)
	if !strings.Contains(blocked.packageOutcome, "未完整完成") {
		t.Fatalf("failed activation outcome is not truthful: %q", blocked.packageOutcome)
	}
	afterFailure, err := os.ReadFile(statePath)
	if err != nil || !bytes.Equal(afterFailure, corrupt) {
		t.Fatalf("failed activation rewrote corrupt state: err=%v data=%q", err, afterFailure)
	}

	runEnginePackageJob(t, blocked, packageRepairAction, enginecatalog.MarkdownHTML, "")
	if blocked.packageManager.LoadError() != nil {
		t.Fatalf("explicit ALL3 repair left state blocked: %v", blocked.packageManager.LoadError())
	}
	for _, engine := range []enginecatalog.EngineID{enginecatalog.MarkdownHTML, enginecatalog.MindmapSVG, enginecatalog.MindmapPreview} {
		assertEnginePackageBundled(t, blocked, engine)
	}

	runEnginePackageJob(t, blocked, packageListAction, enginecatalog.MarkdownHTML, "")
	list := blocked.packageLists[enginecatalog.MarkdownHTML]
	if !list.checked || list.err != nil || len(list.receipts) != 1 || list.receipts[0].Digest != receipt.Digest {
		t.Fatalf("ALL3 repair did not retain the imported package: %#v", list)
	}
}
