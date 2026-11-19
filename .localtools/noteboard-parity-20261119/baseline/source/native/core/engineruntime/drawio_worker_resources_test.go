//go:build windows

package engineruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"levnote.local/native/core/enginecatalog"
	"levnote.local/native/core/engineipc"
)

const drawioResourceHarnessEnable = "LEVNOTE_DRAWIO_WORKER_RESOURCES_HARNESS"
const drawioResourceHarnessOutput = "LEVNOTE_DRAWIO_WORKER_RESOURCES_HARNESS_OUTPUT"
const drawioResourceMeasurementEnable = "LEVNOTE_DRAWIO_WORKER_RESOURCES_MEASUREMENT"

var drawioResourceMeasurementUsed atomic.Bool

const drawioResourceHarnessExpectedSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="-14 -4 128 88" width="128" height="88"><defs><marker id="levnote-arrow" markerWidth="10" markerHeight="7" refX="9" refY="3.5" orient="auto" markerUnits="strokeWidth"><path d="M0,0 L10,3.5 L0,7 z" fill="context-stroke"/></marker></defs><rect x="-14" y="-4" width="128" height="88" fill="#ffffff"/><g font-family="Arial, sans-serif" font-size="14"><rect x="10" y="20" width="80" height="40" fill="#dae8fc" stroke="#6c8ebf" stroke-width="1.5"/><text x="50" y="44.9" text-anchor="middle" fill="#111111"><tspan x="50" dy="0">two</tspan></text></g></svg>` + "\n"

type drawioResourceHarnessPackage struct {
	Version               string           `json:"version"`
	Root                  string           `json:"root"`
	Executable            resourceArtifact `json:"executable"`
	ReleaseManifest       resourceArtifact `json:"releaseManifest"`
	EngineManifest        resourceArtifact `json:"engineManifest"`
	ExecutableBefore      resourceArtifact `json:"executableBefore"`
	ReleaseBefore         resourceArtifact `json:"releaseManifestBefore"`
	EngineManifestAfter   resourceArtifact `json:"engineManifestAfter"`
	ExecutableAfter       resourceArtifact `json:"executableAfter"`
	ReleaseAfter          resourceArtifact `json:"releaseManifestAfter"`
	EngineManifestSHA     string           `json:"engineManifestSha256"`
	ExpectedWorkerSHA     string           `json:"expectedWorkerSha256"`
	ExpectedReleaseSHA    string           `json:"expectedReleaseSha256"`
	WorkerIdentityError   string           `json:"workerIdentityError"`
	ReleaseIdentityError  string           `json:"releaseIdentityError"`
	ManifestIdentityError string           `json:"manifestIdentityError"`
}

type drawioResourceHarnessFixture struct {
	Name             string                         `json:"name"`
	Path             string                         `json:"path"`
	ExpectedSVGPath  string                         `json:"expectedSvgPath"`
	SourceBytes      int                            `json:"sourceBytes"`
	SourceSHA256     string                         `json:"sourceSha256"`
	SVGBytes         int                            `json:"svgBytes"`
	SVGSHA256        string                         `json:"svgSha256"`
	Pages            int                            `json:"pages"`
	Nodes            int                            `json:"nodes"`
	Edges            int                            `json:"edges"`
	SelectedIndex    int                            `json:"selectedIndex"`
	SelectedID       string                         `json:"selectedId"`
	ExpectedWarnings map[string][]engineipc.Warning `json:"expectedWarnings"`
}

type drawioResourceHarnessRecord struct {
	Schema     int                            `json:"schema"`
	Status     string                         `json:"status"`
	CreatedUTC string                         `json:"createdUtc"`
	Platform   resourcePlatform               `json:"platform"`
	Tester     resourceArtifact               `json:"tester"`
	Packages   []drawioResourceHarnessPackage `json:"packages"`
	Fixtures   []drawioResourceHarnessFixture `json:"fixtures"`
	Output     string                         `json:"output"`
	Failures   []string                       `json:"failures"`
}

type drawioResourceMeasurementRecord struct {
	Schema               int                 `json:"schema"`
	Status               string              `json:"status"`
	CreatedAtUTC         string              `json:"createdAtUtc"`
	Fixture              string              `json:"fixture"`
	Version              string              `json:"version"`
	Ordinal              int                 `json:"ordinal"`
	MatrixPosition       int                 `json:"matrixPosition"`
	Platform             resourcePlatform    `json:"platform"`
	Tester               resourceArtifact    `json:"tester"`
	PackageRoot          string              `json:"packageRoot"`
	ConfigRoot           string              `json:"configRoot"`
	SourceBytes          int                 `json:"sourceBytes"`
	SourceSHA256         string              `json:"sourceSha256"`
	SourceAfterSHA256    string              `json:"sourceAfterSha256"`
	ExpectedSourceSHA256 string              `json:"expectedSourceSha256"`
	ExpectedSVGBytes     int                 `json:"expectedSvgBytes"`
	ExpectedSVGSHA256    string              `json:"expectedSvgSha256"`
	ActualSVGBytes       int                 `json:"actualSvgBytes"`
	ActualSVGSHA256      string              `json:"actualSvgSha256"`
	SelectedPageIndex    int                 `json:"selectedPageIndex"`
	SelectedPageID       string              `json:"selectedPageId"`
	ExpectedWarnings     []engineipc.Warning `json:"expectedWarnings"`
	ActualWarnings       []engineipc.Warning `json:"actualWarnings"`
	ImportReceiptDigest  string              `json:"importReceiptDigest"`
	AdmissionCalls       int                 `json:"admissionCalls"`
	VerifyCalls          int                 `json:"verifyCalls"`
	StartCalls           int                 `json:"startCalls"`
	WaitCalls            int                 `json:"waitCalls"`
	Engine               EngineInfo          `json:"engine"`
	ExportStartUTC       string              `json:"exportStartUtc"`
	ExportEndUTC         string              `json:"exportEndUtc"`
	ExportElapsedNS      int64               `json:"exportElapsedNs"`
	SampleIntervalNS     int64               `json:"sampleIntervalNs"`
	MaxAttempts          int                 `json:"maxAttempts"`
	ExportError          string              `json:"exportError"`
	CloseError           string              `json:"closeError"`
	ExportReturned       bool                `json:"exportReturned"`
	RunnerCloseReturned  bool                `json:"runnerCloseReturned"`
	Probe                resourceProbe       `json:"probe"`
	Failures             []string            `json:"failures"`
	Limitations          []string            `json:"limitations"`
}

func TestDrawioWorkerResourcesHarnessPreflight(t *testing.T) {
	if os.Getenv(drawioResourceHarnessEnable) != "1" {
		t.Skip("draw.io worker resource harness requires explicit LEVNOTE_DRAWIO_WORKER_RESOURCES_HARNESS=1")
	}

	output := drawioSmokeAbsoluteEnv(drawioResourceHarnessOutput)
	if output == "" {
		t.Fatalf("%s must name an absent absolute evidence directory", drawioResourceHarnessOutput)
	}
	if _, err := os.Lstat(output); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("evidence directory must not exist before preflight: %v", err)
	}
	if err := os.MkdirAll(output, 0700); err != nil {
		t.Fatal(err)
	}

	record := drawioResourceHarnessRecord{
		Schema:     1,
		Status:     "FAIL",
		CreatedUTC: resourceUTC(time.Now()),
		Platform:   resourcePlatformInfo(),
		Output:     output,
	}
	if !drawioResourceHarnessMeasurementRecordShape() {
		t.Fatal("measurement record JSON shape validation failed")
	}
	fail := func(format string, args ...any) {
		message := fmt.Sprintf(format, args...)
		record.Failures = append(record.Failures, message)
		t.Error(message)
	}
	expectedMatrix := []drawioResourceHarnessMatrixSample{
		{Fixture: "NormalMultiPage", Version: "native-worker.20261110.20", Ordinal: 1},
		{Fixture: "NormalMultiPage", Version: "native-worker.20261110.21", Ordinal: 1},
		{Fixture: "NormalMultiPage", Version: "native-worker.20261110.21", Ordinal: 2},
		{Fixture: "NormalMultiPage", Version: "native-worker.20261110.20", Ordinal: 2},
		{Fixture: "NormalMultiPage", Version: "native-worker.20261110.20", Ordinal: 3},
		{Fixture: "NormalMultiPage", Version: "native-worker.20261110.21", Ordinal: 3},
		{Fixture: "MetadataHeavy", Version: "native-worker.20261110.21", Ordinal: 1},
		{Fixture: "MetadataHeavy", Version: "native-worker.20261110.20", Ordinal: 1},
		{Fixture: "MetadataHeavy", Version: "native-worker.20261110.20", Ordinal: 2},
		{Fixture: "MetadataHeavy", Version: "native-worker.20261110.21", Ordinal: 2},
		{Fixture: "MetadataHeavy", Version: "native-worker.20261110.21", Ordinal: 3},
		{Fixture: "MetadataHeavy", Version: "native-worker.20261110.20", Ordinal: 3},
	}
	if len(expectedMatrix) != 12 {
		fail("fixed measurement matrix must have exactly 12 entries: %d", len(expectedMatrix))
	}
	for index, sample := range expectedMatrix {
		if position := drawioResourceHarnessMatrixPosition(sample.Fixture, sample.Version, sample.Ordinal); position != index+1 {
			fail("fixed measurement matrix mapping differs at %d: fixture=%s version=%s ordinal=%d position=%d",
				index+1, sample.Fixture, sample.Version, sample.Ordinal, position)
		}
	}
	t.Cleanup(func() {
		if t.Failed() || len(record.Failures) != 0 {
			record.Status = "FAIL"
		} else {
			record.Status = "PASS"
		}
		encoded, err := json.MarshalIndent(record, "", "  ")
		if err != nil {
			t.Errorf("encode preflight record: %v", err)
			return
		}
		path := filepath.Join(output, "preflight.json")
		if err := os.WriteFile(path, append(encoded, '\n'), 0600); err != nil {
			t.Errorf("write preflight record: %v", err)
		}
		t.Log("drawio-worker-resources-preflight-json\n" + string(encoded))
	})

	repositoryRoot, err := drawioResourceHarnessRepositoryRoot()
	if err != nil {
		t.Fatal(err)
	}
	tester, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if record.Tester, err = resourceFile(tester); err != nil {
		t.Fatal(err)
	}

	expectedPackages := []struct {
		version, directory, workerSHA, releaseSHA string
	}{
		{
			version: "native-worker.20261110.20", directory: "levnote-native-worker-20261110.20",
			workerSHA:  "70c15942339ecf074f81192c01ce652b19fdbc7dbdfaba139fed22e66c46fe9a",
			releaseSHA: "53c0b71c317734001596988f6ab2bf44201879cc322ba9db3b9ab494e553dbfa",
		},
		{
			version: "native-worker.20261110.21", directory: "levnote-native-worker-20261110.21",
			workerSHA:  "16e51c34688d39625626eab8a4d68aeb507ecc1cb9ffba7ea7478f9b8c0ff5d4",
			releaseSHA: "c2556b0e372466b5f67febf7c9fbacba0d44050b00f9892c64837ed16ce9a824",
		},
	}
	for _, expected := range expectedPackages {
		root := filepath.Join(repositoryRoot, "native", "bin", expected.directory)
		metadata, metadataErr := managedSmokeSourceMetadata(root, enginecatalog.DrawioSVG)
		descriptor, _ := enginecatalog.Lookup(enginecatalog.DrawioSVG)
		executablePath := filepath.Join(root, filepath.FromSlash(descriptor.RelativePath), descriptor.Filename)
		engineManifestPath := filepath.Join(root, filepath.FromSlash(descriptor.RelativePath), "manifest.json")
		releaseManifestPath := filepath.Join(root, "release-manifest.json")
		packageRecord := drawioResourceHarnessPackage{
			Version: expected.version, Root: root,
			ExecutableBefore: mustResourceArtifact(t, executablePath),
			ReleaseBefore:    mustResourceArtifact(t, releaseManifestPath),
		}
		packageRecord.EngineManifest = mustResourceArtifact(t, engineManifestPath)
		if metadataErr != nil {
			packageRecord.ManifestIdentityError = metadataErr.Error()
			fail("verify %s package metadata: %v", expected.version, metadataErr)
		} else if metadata.Version != expected.version || metadata.ExeSHA256 != expected.workerSHA {
			packageRecord.ManifestIdentityError = "engine manifest identity differs from accepted package"
			fail("engine manifest identity differs for %s: version=%s sha256=%s", expected.version, metadata.Version, metadata.ExeSHA256)
		}
		if packageRecord.ExecutableBefore.SHA256 != expected.workerSHA {
			packageRecord.WorkerIdentityError = "executable hash differs from accepted package"
			fail("executable identity differs for %s: %s", expected.version, packageRecord.ExecutableBefore.SHA256)
		}
		if packageRecord.ReleaseBefore.SHA256 != expected.releaseSHA {
			packageRecord.ReleaseIdentityError = "release manifest hash differs from accepted package"
			fail("release manifest identity differs for %s: %s", expected.version, packageRecord.ReleaseBefore.SHA256)
		}
		for _, digest := range []string{packageRecord.ExecutableBefore.SHA256, packageRecord.ReleaseBefore.SHA256, packageRecord.EngineManifest.SHA256} {
			if len(digest) != 64 {
				fail("package identity for %s is not an exact 64-hex SHA256: %q", expected.version, digest)
			}
		}
		packageRecord.Executable = packageRecord.ExecutableBefore
		packageRecord.ReleaseManifest = packageRecord.ReleaseBefore
		packageRecord.ExpectedWorkerSHA = expected.workerSHA
		packageRecord.ExpectedReleaseSHA = expected.releaseSHA
		packageRecord.EngineManifestSHA = packageRecord.EngineManifest.SHA256

		packageRecord.ExecutableAfter = mustResourceArtifact(t, executablePath)
		packageRecord.ReleaseAfter = mustResourceArtifact(t, releaseManifestPath)
		packageRecord.EngineManifestAfter = mustResourceArtifact(t, engineManifestPath)
		if packageRecord.ExecutableAfter != packageRecord.ExecutableBefore ||
			packageRecord.ReleaseAfter != packageRecord.ReleaseBefore ||
			packageRecord.EngineManifestAfter != packageRecord.EngineManifest {
			fail("candidate source package changed during no-process preflight: %s", expected.version)
		}
		record.Packages = append(record.Packages, packageRecord)
	}

	fixtureRoot := filepath.Join(output, "fixtures")
	if err := os.MkdirAll(fixtureRoot, 0700); err != nil {
		t.Fatal(err)
	}
	for _, fixture := range []struct {
		name          string
		selectedIndex int
		selectedID    string
	}{
		{name: "NormalMultiPage", selectedIndex: 2, selectedID: "page-3"},
		{name: "MetadataHeavy", selectedIndex: 3, selectedID: "page-4"},
	} {
		source := drawioResourceHarnessFixtureSource(fixture.name)
		pages, nodes, edges, countErr := drawioResourceHarnessCountXML(source)
		if countErr != nil {
			t.Fatal(countErr)
		}
		sourcePath := filepath.Join(fixtureRoot, fixture.name+".drawio")
		expectedSVGPath := filepath.Join(fixtureRoot, fixture.name+".expected.svg")
		if err := os.WriteFile(sourcePath, source, 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(expectedSVGPath, []byte(drawioResourceHarnessExpectedSVG), 0600); err != nil {
			t.Fatal(err)
		}
		sourceArtifact, sourceArtifactErr := resourceFile(sourcePath)
		svgArtifact, svgArtifactErr := resourceFile(expectedSVGPath)
		if sourceArtifactErr != nil || svgArtifactErr != nil {
			t.Fatalf("freeze fixture identities: source=%v svg=%v", sourceArtifactErr, svgArtifactErr)
		}
		if sourceArtifact.SHA256 != resourceHash(source) || svgArtifact.SHA256 != resourceHash([]byte(drawioResourceHarnessExpectedSVG)) {
			t.Fatalf("fixture identity changed while freezing %s", fixture.name)
		}
		warnings := map[string][]engineipc.Warning{
			"native-worker.20261110.20": drawioResourceHarnessCanonicalPairWarnings(len(source), fixture.selectedIndex),
			"native-worker.20261110.21": {},
		}
		if len(source) >= 1<<20 {
			fail("%s source is not under 1MiB: %d bytes", fixture.name, len(source))
		}
		if pages == 0 || nodes == 0 || edges == 0 {
			fail("%s XML counts are malformed: pages=%d nodes=%d edges=%d", fixture.name, pages, nodes, edges)
		}
		expectedCounts := drawioResourceHarnessExpectedCounts(fixture.name)
		if pages != expectedCounts.pages || nodes != expectedCounts.nodes || edges != expectedCounts.edges {
			fail("%s XML counts differ: pages=%d/%d nodes=%d/%d edges=%d/%d",
				fixture.name, pages, expectedCounts.pages, nodes, expectedCounts.nodes, edges, expectedCounts.edges)
		}
		if !drawioResourceHarnessHasPageIdentity(source, fixture.selectedIndex, fixture.selectedID) {
			fail("%s selected page identity is not exactly index=%d id=%s", fixture.name, fixture.selectedIndex, fixture.selectedID)
		}
		if len(sourceArtifact.SHA256) != 64 || len(svgArtifact.SHA256) != 64 {
			fail("%s fixture hash is not exact 64 hex: source=%d svg=%d",
				fixture.name, len(sourceArtifact.SHA256), len(svgArtifact.SHA256))
		}
		record.Fixtures = append(record.Fixtures, drawioResourceHarnessFixture{
			Name: fixture.name, Path: sourcePath, ExpectedSVGPath: expectedSVGPath,
			SourceBytes: len(source), SourceSHA256: sourceArtifact.SHA256,
			SVGBytes: len(drawioResourceHarnessExpectedSVG), SVGSHA256: svgArtifact.SHA256,
			Pages: pages, Nodes: nodes, Edges: edges,
			SelectedIndex: fixture.selectedIndex, SelectedID: fixture.selectedID,
			ExpectedWarnings: warnings,
		})
	}
}

func TestDrawioWorkerResourceMeasurement(t *testing.T) {
	if os.Getenv(drawioResourceMeasurementEnable) != "1" {
		t.Skip("draw.io worker resource measurement requires explicit LEVNOTE_DRAWIO_WORKER_RESOURCES_MEASUREMENT=1")
	}
	if !drawioResourceMeasurementUsed.CompareAndSwap(false, true) {
		t.Fatal("draw.io measurement permits one request per fresh tester process")
	}
	fixtureName := os.Getenv("LEVNOTE_DRAWIO_WORKER_RESOURCES_FIXTURE")
	version := os.Getenv("LEVNOTE_DRAWIO_WORKER_RESOURCES_VERSION")
	ordinalText := os.Getenv("LEVNOTE_DRAWIO_WORKER_RESOURCES_ORDINAL")
	expectedSourceSHA := os.Getenv("LEVNOTE_DRAWIO_WORKER_RESOURCES_SOURCE_SHA256")
	expectedTesterSHA := os.Getenv("LEVNOTE_DRAWIO_WORKER_RESOURCES_TESTER_SHA256")
	output := drawioSmokeAbsoluteEnv(drawioResourceHarnessOutput)
	if fixtureName == "" || version == "" || ordinalText == "" || expectedSourceSHA == "" ||
		expectedTesterSHA == "" || output == "" {
		t.Fatal("measurement requires explicit fixture, version, ordinal, pinned source/tester digests, and absent absolute output directory")
	}
	ordinal, err := strconv.Atoi(ordinalText)
	if err != nil || ordinal < 1 || ordinal > 3 {
		t.Fatalf("ordinal must be 1..3: %q", ordinalText)
	}
	if !drawioResourceHarnessKnownFixture(fixtureName) {
		t.Fatalf("unknown fixed draw.io resource fixture: %q", fixtureName)
	}
	if drawioResourceHarnessExpectedPackage(version).directory == "" {
		t.Fatalf("unknown fixed draw.io worker version: %q", version)
	}
	if _, err = os.Lstat(output); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("measurement output directory must not exist: %v", err)
	}
	if !drawioResourceHarnessMeasurementRecordShape() {
		t.Fatal("measurement record JSON shape validation failed")
	}
	if err = os.MkdirAll(output, 0700); err != nil {
		t.Fatal(err)
	}
	record := drawioResourceMeasurementRecord{
		Schema: 1, Status: "FAIL", CreatedAtUTC: resourceUTC(time.Now()), Fixture: fixtureName,
		Version: version, Ordinal: ordinal, Platform: resourcePlatformInfo(),
		MatrixPosition:   drawioResourceHarnessMatrixPosition(fixtureName, version, ordinal),
		SampleIntervalNS: resourceSampleInterval.Nanoseconds(), MaxAttempts: resourceMaxAttempts,
		Limitations: []string{},
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
			t.Errorf("encode measurement record: %v", encodeErr)
			return
		}
		path := filepath.Join(output, "measurement.json")
		if writeErr := os.WriteFile(path, append(encoded, '\n'), 0600); writeErr != nil {
			t.Errorf("write measurement record: %v", writeErr)
		}
		t.Log("drawio-worker-resource-measurement-json\n" + string(encoded))
	})

	source := drawioResourceHarnessFixtureSource(fixtureName)
	sourceHash := resourceHash(source)
	if sourceHash != expectedSourceSHA {
		t.Fatalf("fixture source digest differs from preflight pin: got=%s want=%s", sourceHash, expectedSourceSHA)
	}
	record.SourceBytes, record.SourceSHA256 = len(source), sourceHash
	record.ExpectedSourceSHA256 = expectedSourceSHA
	record.ExpectedSVGBytes = len(drawioResourceHarnessExpectedSVG)
	record.ExpectedSVGSHA256 = resourceHash([]byte(drawioResourceHarnessExpectedSVG))
	switch fixtureName {
	case "NormalMultiPage":
		record.SelectedPageIndex, record.SelectedPageID = 2, "page-3"
	case "MetadataHeavy":
		record.SelectedPageIndex, record.SelectedPageID = 3, "page-4"
	default:
		t.Fatalf("unknown fixed draw.io resource fixture: %q", fixtureName)
	}
	record.ExpectedWarnings = drawioResourceHarnessExpectedWarningsFor(version, len(source), record.SelectedPageIndex)
	if record.ExpectedWarnings == nil {
		t.Fatalf("unknown fixed draw.io worker version: %q", version)
	}
	tester, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if record.Tester, err = resourceFile(tester); err != nil {
		t.Fatal(err)
	}
	if record.Tester.SHA256 != expectedTesterSHA {
		t.Fatalf("tester digest differs from preflight pin: got=%s want=%s", record.Tester.SHA256, expectedTesterSHA)
	}
	repositoryRoot, err := drawioResourceHarnessRepositoryRoot()
	if err != nil {
		t.Fatal(err)
	}
	expected := drawioResourceHarnessExpectedPackage(version)
	if expected.directory == "" {
		t.Fatalf("unknown fixed draw.io worker version: %q", version)
	}
	record.PackageRoot = filepath.Join(repositoryRoot, "native", "bin", expected.directory)
	metadata, metadataErr := managedSmokeSourceMetadata(record.PackageRoot, enginecatalog.DrawioSVG)
	if metadataErr != nil || metadata.Version != version || metadata.ExeSHA256 != expected.workerSHA {
		t.Fatalf("verify fixed package identity: metadata=%+v err=%v", metadata, metadataErr)
	}
	releaseArtifact, releaseErr := resourceFile(filepath.Join(record.PackageRoot, "release-manifest.json"))
	if releaseErr != nil || releaseArtifact.SHA256 != expected.releaseSHA {
		t.Fatalf("verify fixed release manifest: artifact=%+v err=%v", releaseArtifact, releaseErr)
	}

	record.ConfigRoot = filepath.Join(output, "config")
	if err = os.MkdirAll(record.ConfigRoot, 0700); err != nil {
		t.Fatal(err)
	}
	manager, err := NewPackageManager(record.PackageRoot, record.ConfigRoot)
	if err != nil || manager.LoadError() != nil {
		t.Fatalf("initialize package manager: manager=%v err=%v", manager, err)
	}
	receipt, err := manager.ImportFromReleaseDir(context.Background(), record.PackageRoot, enginecatalog.DrawioSVG)
	if err != nil {
		t.Fatalf("import fixed package: %v", err)
	}
	if receipt.Engine != enginecatalog.DrawioSVG || receipt.Version != version || receipt.Digest == "" {
		t.Fatalf("import identity differs: %+v", receipt)
	}
	record.ImportReceiptDigest = receipt.Digest
	if err = manager.Activate(context.Background(), enginecatalog.DrawioSVG, receipt.Digest); err != nil {
		t.Fatalf("activate fixed package: %v", err)
	}
	runner, err := NewManaged(manager)
	if err != nil {
		t.Fatal(err)
	}
	admissionCalls, verifyCalls, startCalls, waitCalls := 0, 0, 0, 0
	runnerClosed := false
	finishProbe := func() { record.Probe.finish() }
	t.Cleanup(func() {
		defer finishProbe()
		if runnerClosed {
			return
		}
		if closeErr := runner.Close(); closeErr != nil {
			record.Failures = append(record.Failures, "fallback Runner.Close: "+closeErr.Error())
			t.Errorf("fallback Runner.Close: %v", closeErr)
		}
	})
	record.Probe.anchor = time.Now()
	runner.hooks = &testHooks{
		afterAdmission: func() { admissionCalls++ },
		afterVerify:    func(*verifiedBundle) { verifyCalls++ },
		afterStart:     func(started *os.Process) { startCalls++; record.Probe.afterStart(started) },
		afterWait:      func() { waitCalls++; record.Probe.afterWait() },
	}
	start := time.Now()
	record.ExportStartUTC = resourceUTC(start)
	result, exportErr := runner.Export(context.Background(), engineipc.DrawioSVG, Source{
		Size: int64(len(source)), Data: bytes.NewReader(source),
		Page: &engineipc.RequestPage{Index: record.SelectedPageIndex, ID: record.SelectedPageID},
	})
	record.ExportElapsedNS, record.ExportEndUTC, record.ExportReturned = time.Since(start).Nanoseconds(), resourceUTC(time.Now()), true
	if exportErr != nil {
		record.ExportError = exportErr.Error()
	}
	closeErr := runner.Close()
	record.RunnerCloseReturned = true
	runnerClosed = true
	if closeErr != nil {
		record.CloseError = closeErr.Error()
	}
	finishProbe()
	record.Engine = result.Engine
	record.AdmissionCalls, record.VerifyCalls, record.StartCalls, record.WaitCalls =
		admissionCalls, verifyCalls, startCalls, waitCalls
	record.ActualSVGBytes, record.ActualSVGSHA256 = len(result.Payload), resourceHash(result.Payload)
	record.ActualWarnings = make([]engineipc.Warning, 0, len(result.Warnings))
	record.ActualWarnings = append(record.ActualWarnings, result.Warnings...)
	record.SourceAfterSHA256 = resourceHash(source)

	check := func(ok bool, format string, args ...any) {
		if ok {
			return
		}
		fail(format, args...)
	}
	check(exportErr == nil, "managed draw.io export failed: %v", exportErr)
	check(closeErr == nil, "Runner.Close failed: %v", closeErr)
	check(bytes.Equal(result.Payload, []byte(drawioResourceHarnessExpectedSVG)), "SVG differs from fixed oracle")
	check(drawioSmokeWarningsEqual(result.Warnings, record.ExpectedWarnings), "warnings differ from fixed oracle: %+v", result.Warnings)
	check(result.Engine.ID == enginecatalog.DrawioSVG && result.Engine.Version == version &&
		result.Engine.SHA256 == expected.workerSHA && result.Engine.Source == PackageInstalled &&
		result.Engine.PackageDigest == receipt.Digest, "engine identity differs: %+v", result.Engine)
	probe := &record.Probe
	check(len(probe.Errors) == 0 && probe.Duplicated, "precise process handle failed: %v", probe.Errors)
	descriptor, _ := enginecatalog.Lookup(enginecatalog.DrawioSVG)
	expectedImage := filepath.Join(
		manager.installedPath(enginecatalog.DrawioSVG, receipt.Digest), descriptor.Filename,
	)
	check(probe.IdentityError == "" && strings.EqualFold(filepath.Clean(probe.ImagePath), filepath.Clean(expectedImage)),
		"process image differs: image=%q expected=%q error=%s", probe.ImagePath, expectedImage, probe.IdentityError)
	check(probe.AfterWaitObserved && probe.SamplerJoined && !probe.AttemptLimitReached, "probe did not join cleanly")
	check(probe.WaitError == "" && probe.WaitState != nil && *probe.WaitState == syscall.WAIT_OBJECT_0,
		"process did not signal after joined runtime: %s", probe.WaitError)
	check(probe.ExitCodeError == "" && probe.ExitCode != nil && *probe.ExitCode == 0,
		"process exit was not zero: code=%v error=%s", probe.ExitCode, probe.ExitCodeError)
	check(probe.HandleClosed && probe.HandleCloseError == "", "duplicate handle close failed: %s", probe.HandleCloseError)
	check(record.SourceAfterSHA256 == sourceHash, "source bytes changed during export")
	check(admissionCalls == 1 && verifyCalls == 1 && startCalls == 1 && waitCalls == 1,
		"managed operation count differs: admission=%d verify=%d start=%d wait=%d",
		admissionCalls, verifyCalls, startCalls, waitCalls)
	check(probe.InitialTimes != nil && probe.InitialTimes.Error == "" &&
		probe.InitialTimes.Times != nil && probe.InitialTimes.Times.CreationFiletime100ns != 0,
		"initial creation identity unavailable")
	var initialTimes, finalTimes *resourceProcessTimes
	if probe.InitialTimes != nil {
		initialTimes = probe.InitialTimes.Times
	}
	if probe.FinalTimes != nil {
		finalTimes = probe.FinalTimes.Times
	}
	check(probe.FinalTimes != nil && probe.FinalTimes.Error == "" &&
		finalTimes != nil && finalTimes.ExitFiletime100ns != 0,
		"final creation/exit identity unavailable")
	if initialTimes != nil && finalTimes != nil {
		check(finalTimes.CreationFiletime100ns == initialTimes.CreationFiletime100ns &&
			finalTimes.ExitFiletime100ns >= finalTimes.CreationFiletime100ns,
			"creation/exit ordering is invalid: creation=%d exit=%d",
			finalTimes.CreationFiletime100ns, finalTimes.ExitFiletime100ns)
	}
	check(probe.Summary.FailedReads == 0, "memory probe reads failed: %d", probe.Summary.FailedReads)
	check(probe.Summary.FinalMemoryReadOK, "final memory read unavailable")
	check(probe.Summary.FinalCPUReadOK, "final CPU read unavailable")
	if !probe.Summary.LifetimePeakCountersComplete {
		record.Limitations = append(record.Limitations,
			"limited observation: final lifetime peak counters incomplete: "+probe.Summary.PeakCoverage)
	}
}

func drawioResourceHarnessMeasurementRecordShape() bool {
	record := drawioResourceMeasurementRecord{
		Status: "FAIL", CreatedAtUTC: "time", Fixture: "fixture", Version: "version",
		Ordinal: 1, MatrixPosition: 1, PackageRoot: "package", ConfigRoot: "config",
		SourceBytes: 1, SourceSHA256: "source", SourceAfterSHA256: "source-after",
		ExpectedSVGBytes: 1, ExpectedSVGSHA256: "svg", ActualSVGBytes: 1, ActualSVGSHA256: "svg",
		SelectedPageIndex: 1, SelectedPageID: "page", ExpectedWarnings: []engineipc.Warning{},
		ActualWarnings: []engineipc.Warning{}, ExportStartUTC: "start", ExportEndUTC: "end",
		ImportReceiptDigest: "receipt",
		AdmissionCalls:      1, VerifyCalls: 1, StartCalls: 1, WaitCalls: 1,
		ExportElapsedNS: 1, SampleIntervalNS: 1, MaxAttempts: 1, ExportError: "error",
		ExpectedSourceSHA256: "expected-source",
		CloseError:           "close", ExportReturned: true, RunnerCloseReturned: true,
		Limitations: []string{"limitation"},
	}
	data, err := json.Marshal(record)
	if err != nil {
		return false
	}
	var values map[string]any
	if err := json.Unmarshal(data, &values); err != nil {
		return false
	}
	for _, key := range []string{
		"status", "createdAtUtc", "fixture", "version", "ordinal", "matrixPosition",
		"packageRoot", "configRoot", "sourceBytes", "sourceSha256", "sourceAfterSha256",
		"expectedSvgBytes", "expectedSvgSha256", "actualSvgBytes", "actualSvgSha256",
		"selectedPageIndex", "selectedPageId", "expectedWarnings", "actualWarnings", "importReceiptDigest",
		"expectedSourceSha256", "admissionCalls", "verifyCalls", "startCalls", "waitCalls",
		"limitations",
		"exportStartUtc", "exportEndUtc", "exportElapsedNs", "sampleIntervalNs",
		"maxAttempts", "exportError", "closeError", "exportReturned",
		"runnerCloseReturned",
	} {
		if _, ok := values[key]; !ok {
			return false
		}
	}
	return true
}

func drawioResourceHarnessRepositoryRoot() (string, error) {
	current, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		goScript := filepath.Join(current, "native", "scripts", "go.ps1")
		packageRoot := filepath.Join(current, "native", "bin", "levnote-native-worker-20261110.20")
		if info, err := os.Lstat(goScript); err == nil && !info.IsDir() {
			if info, err := os.Lstat(packageRoot); err == nil && info.IsDir() {
				return current, nil
			}
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", errors.New("could not locate the LevNote repository root")
		}
		current = parent
	}
}

func drawioResourceHarnessFixtureSource(name string) []byte {
	var source strings.Builder
	source.WriteString("<mxfile host=\"levnote\" modified=\"2026-11-10T00:00:00.000Z\">\n")
	source.WriteString("  <!-- Harness fixture: bytes are frozen by preflight, not by this comment. -->\n")
	switch name {
	case "NormalMultiPage":
		for page := 0; page < 8; page++ {
			drawioResourceHarnessWriteNormalPage(&source, page, page == 2)
		}
	case "MetadataHeavy":
		for page := 0; page < 6; page++ {
			drawioResourceHarnessWriteMetadataPage(&source, page, page == 3)
		}
	default:
		panic("unknown draw.io resource harness fixture")
	}
	source.WriteString("</mxfile>\n")
	return []byte(source.String())
}

func drawioResourceHarnessWriteNormalPage(source *strings.Builder, page int, selected bool) {
	fmt.Fprintf(source, "  <diagram id=\"page-%d\" name=\"Page %d\">\n", page+1, page+1)
	source.WriteString("    <mxGraphModel dx=\"800\" dy=\"600\" page=\"1\">\n      <root>\n")
	source.WriteString("        <mxCell id=\"0\"/>\n        <mxCell id=\"1\" parent=\"0\"/>\n")
	if selected {
		source.WriteString("        <mxCell id=\"selected\" value=\"two\" vertex=\"1\" parent=\"1\"><mxGeometry x=\"10\" y=\"20\" width=\"80\" height=\"40\" as=\"geometry\"/></mxCell>\n")
		source.WriteString("      </root>\n    </mxGraphModel>\n  </diagram>\n")
		return
	}
	for node := 0; node < 21; node++ {
		fmt.Fprintf(source, "        <mxCell id=\"p%d-n%d\" value=\"Node %d\" vertex=\"1\" parent=\"1\"><mxGeometry x=\"%d\" y=\"%d\" width=\"120\" height=\"60\" as=\"geometry\"/></mxCell>\n",
			page, node, node, 20+(node%4)*180, 100+(node/4)*100)
	}
	for edge := 0; edge < 6; edge++ {
		fmt.Fprintf(source, "        <mxCell id=\"p%d-e%d\" value=\"Edge %d\" edge=\"1\" parent=\"1\" source=\"p%d-n%d\" target=\"p%d-n%d\"><mxGeometry relative=\"1\" as=\"geometry\"/></mxCell>\n",
			page, edge, edge, page, edge, page, edge+1)
	}
	if !selected {
		source.WriteString("        <unknown inert=\"keep-me\"/>\n")
	}
	source.WriteString("      </root>\n    </mxGraphModel>\n  </diagram>\n")
}

func drawioResourceHarnessWriteMetadataPage(source *strings.Builder, page int, selected bool) {
	fmt.Fprintf(source, "  <diagram id=\"page-%d\" name=\"Page %d\">\n", page+1, page+1)
	source.WriteString("    <mxGraphModel dx=\"800\" dy=\"600\" page=\"1\">\n      <root>\n")
	source.WriteString("        <mxCell id=\"0\"/>\n        <mxCell id=\"1\" parent=\"0\"/>\n")
	nodeCount, edgeCount := 8, 12
	if selected {
		source.WriteString("        " + drawioResourceHarnessMetadataCell(fmt.Sprintf("selected"), "two", "vertex", 1, 10, 20, 80, 40, "", "") + "\n")
		source.WriteString("      </root>\n    </mxGraphModel>\n  </diagram>\n")
		return
	}
	for node := 0; node < nodeCount; node++ {
		cellID := fmt.Sprintf("p%d-n%d", page, node)
		source.WriteString("        " + drawioResourceHarnessMetadataCell(cellID, fmt.Sprintf("Node %d", node), "vertex", 1,
			20+(node%4)*180, 100+(node/4)*100, 120, 60, "", "") + "\n")
	}
	for edge := 0; edge < edgeCount; edge++ {
		cellID := fmt.Sprintf("p%d-e%d", page, edge)
		source.WriteString("        " + drawioResourceHarnessMetadataCell(cellID, fmt.Sprintf("Edge %d", edge), "edge", 1,
			0, 0, 0, 0, fmt.Sprintf("p%d-n%d", page, edge%8), fmt.Sprintf("p%d-n%d", page, (edge+1)%8)) + "\n")
	}
	if !selected {
		source.WriteString("        " + drawioResourceHarnessMetadataElement("unknown") + "\n")
	}
	source.WriteString("      </root>\n    </mxGraphModel>\n  </diagram>\n")
}

func drawioResourceHarnessMetadataCell(id, value, kind string, enabled, x, y, width, height int, sourceID, targetID string) string {
	var cell strings.Builder
	enabledText := "0"
	if enabled != 0 {
		enabledText = "1"
	}
	fmt.Fprintf(&cell, "<mxCell id=%q value=%q %s=%q parent=\"1\"", id, value, kind, enabledText)
	if sourceID != "" {
		fmt.Fprintf(&cell, " source=%q target=%q", sourceID, targetID)
	}
	for index := 0; index < 32; index++ {
		fmt.Fprintf(&cell, " custom%02d=\"metadata-%02d\"", index, index)
	}
	if kind == "vertex" {
		fmt.Fprintf(&cell, "><mxGeometry x=\"%d\" y=\"%d\" width=\"%d\" height=\"%d\" as=\"geometry\"/></mxCell>", x, y, width, height)
	} else {
		cell.WriteString("><mxGeometry relative=\"1\" as=\"geometry\"/></mxCell>")
	}
	return cell.String()
}

func drawioResourceHarnessMetadataElement(name string) string {
	var element strings.Builder
	fmt.Fprintf(&element, "<%s", name)
	for index := 0; index < 32; index++ {
		fmt.Fprintf(&element, " metadata%02d=\"inert-%02d\"", index, index)
	}
	element.WriteString("/>")
	return element.String()
}

func drawioResourceHarnessCanonicalPairWarnings(sourceBytes, page int) []engineipc.Warning {
	message := func(cell string) string {
		return fmt.Sprintf("page %d cell %s: mxCell is neither a vertex nor an edge and is not rendered", page, cell)
	}
	return []engineipc.Warning{
		{Code: "UNSUPPORTED_ELEMENT", Message: message("0"), Start: 0, End: uint32(sourceBytes)},
		{Code: "UNSUPPORTED_ELEMENT", Message: message("1"), Start: 0, End: uint32(sourceBytes)},
	}
}

func drawioResourceHarnessCountXML(source []byte) (pages, nodes, edges int, returned error) {
	decoder := xml.NewDecoder(bytes.NewReader(source))
	for {
		token, tokenErr := decoder.Token()
		if errors.Is(tokenErr, io.EOF) {
			return pages, nodes, edges, nil
		}
		if tokenErr != nil {
			return 0, 0, 0, tokenErr
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		switch start.Name.Local {
		case "diagram":
			pages++
		case "mxCell":
			if attrValue(start.Attr, "vertex") == "1" {
				nodes++
			}
			if attrValue(start.Attr, "edge") == "1" {
				edges++
			}
		}
	}
}

type drawioResourceHarnessCounts struct {
	pages, nodes, edges int
}

func drawioResourceHarnessExpectedCounts(name string) drawioResourceHarnessCounts {
	switch name {
	case "NormalMultiPage":
		return drawioResourceHarnessCounts{pages: 8, nodes: 148, edges: 42}
	case "MetadataHeavy":
		return drawioResourceHarnessCounts{pages: 6, nodes: 41, edges: 60}
	default:
		return drawioResourceHarnessCounts{}
	}
}

func drawioResourceHarnessMatrixPosition(fixture, version string, ordinal int) int {
	order := []drawioResourceHarnessMatrixSample{
		{Fixture: "NormalMultiPage", Version: "native-worker.20261110.20", Ordinal: 1},
		{Fixture: "NormalMultiPage", Version: "native-worker.20261110.21", Ordinal: 1},
		{Fixture: "NormalMultiPage", Version: "native-worker.20261110.21", Ordinal: 2},
		{Fixture: "NormalMultiPage", Version: "native-worker.20261110.20", Ordinal: 2},
		{Fixture: "NormalMultiPage", Version: "native-worker.20261110.20", Ordinal: 3},
		{Fixture: "NormalMultiPage", Version: "native-worker.20261110.21", Ordinal: 3},
		{Fixture: "MetadataHeavy", Version: "native-worker.20261110.21", Ordinal: 1},
		{Fixture: "MetadataHeavy", Version: "native-worker.20261110.20", Ordinal: 1},
		{Fixture: "MetadataHeavy", Version: "native-worker.20261110.20", Ordinal: 2},
		{Fixture: "MetadataHeavy", Version: "native-worker.20261110.21", Ordinal: 2},
		{Fixture: "MetadataHeavy", Version: "native-worker.20261110.21", Ordinal: 3},
		{Fixture: "MetadataHeavy", Version: "native-worker.20261110.20", Ordinal: 3},
	}
	for index, sample := range order {
		if sample.Fixture == fixture && sample.Version == version && sample.Ordinal == ordinal {
			return index + 1
		}
	}
	return 0
}

func drawioResourceHarnessKnownFixture(name string) bool {
	switch name {
	case "NormalMultiPage", "MetadataHeavy":
		return true
	default:
		return false
	}
}

func drawioResourceHarnessHasPageIdentity(source []byte, index int, id string) bool {
	decoder := xml.NewDecoder(bytes.NewReader(source))
	pageIndex := -1
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			return false
		}
		if err != nil {
			return false
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		if start.Name.Local != "diagram" {
			continue
		}
		pageIndex++
		if pageIndex == index {
			return attrValue(start.Attr, "id") == id
		}
	}
}

type drawioResourceHarnessPackageIdentity struct {
	directory, workerSHA, releaseSHA string
}

type drawioResourceHarnessMatrixSample struct {
	Fixture, Version string
	Ordinal          int
}

func drawioResourceHarnessExpectedPackage(version string) drawioResourceHarnessPackageIdentity {
	switch version {
	case "native-worker.20261110.20":
		return drawioResourceHarnessPackageIdentity{
			directory:  "levnote-native-worker-20261110.20",
			workerSHA:  "70c15942339ecf074f81192c01ce652b19fdbc7dbdfaba139fed22e66c46fe9a",
			releaseSHA: "53c0b71c317734001596988f6ab2bf44201879cc322ba9db3b9ab494e553dbfa",
		}
	case "native-worker.20261110.21":
		return drawioResourceHarnessPackageIdentity{
			directory:  "levnote-native-worker-20261110.21",
			workerSHA:  "16e51c34688d39625626eab8a4d68aeb507ecc1cb9ffba7ea7478f9b8c0ff5d4",
			releaseSHA: "c2556b0e372466b5f67febf7c9fbacba0d44050b00f9892c64837ed16ce9a824",
		}
	default:
		return drawioResourceHarnessPackageIdentity{}
	}
}

func drawioResourceHarnessExpectedWarningsFor(version string, sourceBytes, page int) []engineipc.Warning {
	switch version {
	case "native-worker.20261110.20":
		return drawioResourceHarnessCanonicalPairWarnings(sourceBytes, page)
	case "native-worker.20261110.21":
		return []engineipc.Warning{}
	default:
		return nil
	}
}

func attrValue(attrs []xml.Attr, name string) string {
	for _, attr := range attrs {
		if attr.Name.Local == name {
			return attr.Value
		}
	}
	return ""
}

func mustResourceArtifact(t *testing.T, path string) resourceArtifact {
	t.Helper()
	artifact, err := resourceFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return artifact
}
