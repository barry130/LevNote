//go:build windows

package engineruntime

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"levnote.local/native/core/enginecatalog"
	"levnote.local/native/core/engineipc"
)

type managedSmokeRecord struct {
	Schema                       int
	Status                       string
	Scope                        string
	CreatedUTC                   string
	OldRelease                   string
	NewBundled                   string
	OldVersion                   string
	OldExeSHA256                 string
	NewVersion                   string
	NewExeSHA256                 string
	PackageDigest                string
	ImportSelection              string
	ActivatedSelection           string
	ResetSelection               string
	SourceBytes                  int
	SourceSHA256                 string
	InstalledHTMLBytes           int
	InstalledHTMLSHA256          string
	BundledHTMLBytes             int
	BundledHTMLSHA256            string
	InstalledExportOK            bool
	BundledExportOK              bool
	InstalledCloseCalled         bool
	InstalledCloseError          string
	InstalledFallbackCloseCalled bool
	InstalledFallbackCloseError  string
	BundledCloseCalled           bool
	BundledCloseError            string
	BundledFallbackCloseCalled   bool
	BundledFallbackCloseError    string
	Failures                     []string
}

type managedSmokeMetadata struct {
	Version   string
	ExeSHA256 string
}

func TestManagedRealOldReleaseThenBundledResetSmoke(t *testing.T) {
	if os.Getenv("LEVNOTE_MANAGED_REAL_SMOKE") != "1" {
		t.Skip("real managed old-release smoke requires LEVNOTE_MANAGED_REAL_SMOKE=1 and an explicit source release")
	}
	sourceRelease := os.Getenv("LEVNOTE_MANAGED_REAL_SMOKE_RELEASE")
	if sourceRelease == "" {
		t.Fatal("LEVNOTE_MANAGED_REAL_SMOKE_RELEASE must name the explicit source-release directory")
	}
	if !filepath.IsAbs(sourceRelease) {
		t.Fatal("source release path must be absolute")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	hostDir := filepath.Dir(exe)
	stateDir := t.TempDir()
	record := &managedSmokeRecord{
		Schema:     1,
		Status:     "FAIL",
		Scope:      "import old release, activate one Markdown export, explicit Reset, then one bundled Markdown export",
		CreatedUTC: time.Now().UTC().Format(time.RFC3339Nano),
		OldRelease: sourceRelease,
		NewBundled: hostDir,
	}
	fail := func(format string, args ...any) {
		message := fmt.Sprintf(format, args...)
		record.Failures = append(record.Failures, message)
		t.Error(message)
	}
	t.Cleanup(func() {
		record.Status = "PASS"
		if t.Failed() || len(record.Failures) != 0 {
			record.Status = "FAIL"
		}
		encoded, encodeErr := json.MarshalIndent(record, "", "  ")
		if encodeErr != nil {
			t.Errorf("encode managed smoke record failed: %v", encodeErr)
			return
		}
		t.Log("managed-real-smoke-record-json\n" + string(encoded))
	})
	manager, err := NewPackageManager(hostDir, stateDir)
	if err != nil {
		t.Fatal(err)
	}
	if strings.EqualFold(filepath.Clean(sourceRelease), filepath.Clean(hostDir)) {
		t.Fatal("source release and fresh host directory must differ")
	}
	oldMeta, err := managedSmokeSourceMetadata(sourceRelease, enginecatalog.MarkdownHTML)
	if err != nil {
		t.Fatal(err)
	}
	newMeta, err := managedSmokeSourceMetadata(hostDir, enginecatalog.MarkdownHTML)
	if err != nil {
		t.Fatal(err)
	}
	record.OldVersion, record.OldExeSHA256 = oldMeta.Version, oldMeta.ExeSHA256
	record.NewVersion, record.NewExeSHA256 = newMeta.Version, newMeta.ExeSHA256
	if oldMeta.Version == newMeta.Version {
		t.Fatalf("old release and new bundled version must differ: %s", oldMeta.Version)
	}
	receipt, err := manager.ImportFromReleaseDir(context.Background(), sourceRelease, enginecatalog.MarkdownHTML)
	if err != nil {
		t.Fatal(err)
	}
	record.PackageDigest = receipt.Digest
	record.ImportSelection = managedSmokeSelection(manager, enginecatalog.MarkdownHTML)
	if record.ImportSelection != "bundled" {
		fail("Import changed selection to %s", record.ImportSelection)
	}
	if err := manager.Activate(context.Background(), enginecatalog.MarkdownHTML, receipt.Digest); err != nil {
		t.Fatal(err)
	}
	record.ActivatedSelection = managedSmokeSelection(manager, enginecatalog.MarkdownHTML)
	if record.ActivatedSelection != "installed:"+receipt.Digest {
		fail("Activate did not select exact package: %s", record.ActivatedSelection)
	}
	const markdownSource = "# 字🙂 CRLF <&>\r\n"
	sourceBytes := []byte(markdownSource)
	record.SourceBytes, record.SourceSHA256 = len(sourceBytes), managedSmokeHash(sourceBytes)
	expectedHTML := "<!doctype html>\n<meta charset=\"utf-8\">\n<body>\n<h1 id=\"字-crlf\">字🙂 CRLF &lt;&amp;&gt;</h1>\n</body>\n"
	runner, err := NewManaged(manager)
	if err != nil {
		t.Fatal(err)
	}
	runnerClosed := false
	t.Cleanup(func() {
		if runnerClosed {
			return
		}
		record.InstalledFallbackCloseCalled = true
		if closeErr := runner.Close(); closeErr != nil {
			record.InstalledFallbackCloseError = closeErr.Error()
			record.Failures = append(record.Failures, "cleanup fallback for installed runner failed: "+closeErr.Error())
			t.Errorf("cleanup fallback for installed runner failed: %v", closeErr)
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	installedResult, installedErr := runner.Export(ctx, engineipc.MarkdownHTML, Source{
		Size: int64(len(sourceBytes)), Data: bytes.NewReader(sourceBytes),
	})
	record.InstalledExportOK = installedErr == nil
	if installedErr != nil {
		fail("installed export failed: %v", installedErr)
	} else {
		record.InstalledHTMLBytes, record.InstalledHTMLSHA256 = len(installedResult.Payload), managedSmokeHash(installedResult.Payload)
		if installedResult.Engine.Source != PackageInstalled || installedResult.Engine.PackageDigest != receipt.Digest ||
			installedResult.Engine.ID != enginecatalog.MarkdownHTML || installedResult.Engine.Version != oldMeta.Version ||
			installedResult.Engine.SHA256 != oldMeta.ExeSHA256 {
			fail("installed engine metadata differs from release metadata: %+v", installedResult.Engine)
		}
		if receipt.Engine != enginecatalog.MarkdownHTML || receipt.Version != oldMeta.Version {
			fail("receipt identity/version differs from source release: %+v", receipt)
		}
		if !bytes.Equal(installedResult.Payload, []byte(expectedHTML)) {
			fail("installed HTML differs from independent oracle: %q", string(installedResult.Payload))
		}
		if !bytes.Equal(sourceBytes, []byte(markdownSource)) {
			fail("source bytes changed after installed export")
		}
	}
	if err := runner.Close(); err != nil {
		fail("installed Runner.Close failed: %v", err)
		record.InstalledCloseError = err.Error()
	}
	record.InstalledCloseCalled = true
	runnerClosed = true
	if err := manager.Reset(context.Background(), enginecatalog.MarkdownHTML); err != nil {
		t.Fatal(err)
	}
	record.ResetSelection = managedSmokeSelection(manager, enginecatalog.MarkdownHTML)
	if record.ResetSelection != "bundled" {
		fail("Reset did not restore bundled selection: %s", record.ResetSelection)
	}
	freshRunner, err := NewManaged(manager)
	if err != nil {
		t.Fatal(err)
	}
	freshRunnerClosed := false
	t.Cleanup(func() {
		if freshRunnerClosed {
			return
		}
		record.BundledFallbackCloseCalled = true
		if closeErr := freshRunner.Close(); closeErr != nil {
			record.BundledFallbackCloseError = closeErr.Error()
			record.Failures = append(record.Failures, "cleanup fallback for bundled runner failed: "+closeErr.Error())
			t.Errorf("cleanup fallback for bundled runner failed: %v", closeErr)
		}
	})
	ctx2, cancel2 := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel2()
	bundledResult, bundledErr := freshRunner.Export(ctx2, engineipc.MarkdownHTML, Source{
		Size: int64(len(sourceBytes)), Data: bytes.NewReader(sourceBytes),
	})
	record.BundledExportOK = bundledErr == nil
	if bundledErr != nil {
		fail("bundled export failed: %v", bundledErr)
	} else {
		record.BundledHTMLBytes, record.BundledHTMLSHA256 = len(bundledResult.Payload), managedSmokeHash(bundledResult.Payload)
		if bundledResult.Engine.Source != PackageBundled || bundledResult.Engine.PackageDigest != "" ||
			bundledResult.Engine.ID != enginecatalog.MarkdownHTML || bundledResult.Engine.Version != newMeta.Version ||
			bundledResult.Engine.SHA256 != newMeta.ExeSHA256 {
			fail("bundled engine metadata differs from release metadata: %+v", bundledResult.Engine)
		}
		if !bytes.Equal(bundledResult.Payload, []byte(expectedHTML)) {
			fail("bundled HTML differs from independent oracle: %q", string(bundledResult.Payload))
		}
		if !bytes.Equal(sourceBytes, []byte(markdownSource)) {
			fail("source bytes changed after bundled export")
		}
		if !bytes.Equal(installedResult.Payload, bundledResult.Payload) {
			fail("installed and bundled output parity differs")
		}
	}
	if err := freshRunner.Close(); err != nil {
		fail("bundled Runner.Close failed: %v", err)
		record.BundledCloseError = err.Error()
	}
	record.BundledCloseCalled = true
	freshRunnerClosed = true
}

func managedSmokeSelection(m *PackageManager, engine enginecatalog.EngineID) string {
	for _, status := range m.Snapshot() {
		if status.Engine != engine {
			continue
		}
		if status.Source == PackageBundled {
			return "bundled"
		}
		return string(status.Source) + ":" + status.Digest
	}
	return "missing"
}

func managedSmokeHash(data []byte) string {
	value := sha256.Sum256(data)
	return hex.EncodeToString(value[:])
}

func managedSmokeSourceMetadata(root string, engine enginecatalog.EngineID) (metadata managedSmokeMetadata, returned error) {
	descriptor, ok := enginecatalog.Lookup(engine)
	if !ok {
		return metadata, errors.New("unknown engine")
	}
	ctx := context.Background()
	var guard packageGuard
	defer func() {
		if closeErr := guard.close(); closeErr != nil {
			returned = errors.Join(returned, closeErr)
		}
	}()
	inventoryFile, err := guard.open(ctx, filepath.Join(root, "release-manifest.json"))
	if err != nil {
		return metadata, err
	}
	inventoryBytes, err := packageRead(ctx, inventoryFile, releaseInventoryLimit)
	if err != nil {
		return metadata, err
	}
	inventory, err := parseReleaseInventory(inventoryBytes)
	if err != nil {
		return metadata, err
	}
	exePath := descriptor.RelativePath + "/" + descriptor.Filename
	entry, listed := inventory.Files[exePath]
	manifestPath := descriptor.RelativePath + "/manifest.json"
	manifestEntry, manifestListed := inventory.Files[manifestPath]
	if !listed || !manifestListed {
		return metadata, errors.New("release inventory is missing fixed Markdown executable or manifest")
	}
	hostEntry, hostListed := inventory.Files["levnote-native.exe"]
	if !hostListed {
		return metadata, errors.New("release inventory is missing ordinary host executable")
	}
	hostFile, err := guard.open(ctx, filepath.Join(root, "levnote-native.exe"))
	if err != nil {
		return metadata, err
	}
	hostInfo, err := hostFile.Stat()
	if err != nil || hostInfo.Size() != hostEntry.Size {
		return metadata, errors.New("ordinary host executable is missing or its size differs from inventory")
	}
	manifestFile, err := guard.open(ctx, filepath.Join(root, filepath.FromSlash(manifestPath)))
	if err != nil {
		return metadata, err
	}
	manifestBytes, err := packageRead(ctx, manifestFile, manifestLimit)
	if err != nil {
		return metadata, err
	}
	manifest, err := parseManifest(manifestBytes, engine)
	if err != nil {
		return metadata, err
	}
	if inventory.Version != manifest.EngineVersion ||
		manifest.Filename != descriptor.Filename || manifest.Size != entry.Size || int64(len(manifestBytes)) != manifestEntry.Size ||
		manifest.SHA256 != entry.SHA256 || manifest.Engine != descriptor.Engine || manifest.Format != descriptor.Format ||
		manifest.GOOS != descriptor.GOOS || manifest.GOARCH != descriptor.GOARCH {
		return metadata, errors.New("inventory and fixed Markdown manifest are inconsistent")
	}
	metadata.Version, metadata.ExeSHA256 = manifest.EngineVersion, strings.ToLower(manifest.SHA256)
	return metadata, nil
}
