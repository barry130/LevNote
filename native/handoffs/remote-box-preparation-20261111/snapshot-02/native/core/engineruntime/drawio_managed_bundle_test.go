//go:build windows

package engineruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"levnote.local/native/core/enginecatalog"
	"levnote.local/native/core/engineipc"
)

const (
	drawioManagedSmokeEnable    = "LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE"
	drawioManagedSmokeCandidate = "LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_CANDIDATE"
	drawioManagedSmokeEvidence  = "LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_EVIDENCE"
	drawioManagedSmokeConfig    = "LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_CONFIG"
)

type drawioSmokeSelections struct {
	Markdown       string
	MindmapSVG     string
	MindmapPreview string
}

type drawioSmokeExport struct {
	NewManagedCalls     int
	ExportCalls         int
	AdmissionCalls      int
	VerifyCalls         int
	VerifyIdentityError string
	StartCalls          int
	WaitCalls           int
	Returned            bool
	Error               string
	PageIndex           int
	PageID              string
}

type drawioSmokeProcess struct {
	StartObserved           bool
	StartUTC                string
	PID                     int
	DuplicateHandleObserved bool
	DuplicateHandle         uint64
	DuplicateError          string
	ImagePathObserved       bool
	ImagePath               string
	ExpectedImagePath       string
	ImagePathError          string
	CreationObserved        bool
	CreationFiletime100ns   uint64
	CreationUTC             string
	TimesError              string
	WaitObserved            bool
	WaitUTC                 string
	ExitCode                *uint32
	ExitCodeError           string
	PrimaryCloseAttempted   bool
	PrimaryClosed           bool
	PrimaryCloseError       string
	FallbackCloseAttempted  bool
	FallbackClosed          bool
	FallbackCloseError      string
	handle                  syscall.Handle
}

type drawioSmokeRunner struct {
	CloseCalled         bool
	CloseError          string
	FallbackCloseCalled bool
	FallbackCloseError  string
}

type drawioSmokeRecord struct {
	Schema                     int
	Status                     string
	StartedUTC                 string
	FinishedUTC                string
	CandidateRelease           string
	EvidencePath               string
	EvidenceRoot               string
	ConfigRoot                 string
	BundleCopy                 string
	CandidateVersion           string
	CandidateExeSHA256Before   string
	CandidateExeSHA256After    string
	BundledFourthSelection     string
	ImportFourthSelection      string
	ActivatedFourthSelection   string
	LegacyStateBytes           int
	LegacyStateSHA256          string
	LegacyStateUnchangedOnLoad bool
	InitialSelections          drawioSmokeSelections
	AfterImportSelections      drawioSmokeSelections
	AfterActivateSelections    drawioSmokeSelections
	ImportDigest               string
	SourceBytes                int
	SourceSHA256Before         string
	SourceSHA256After          string
	PayloadBytes               int
	PayloadSHA256              string
	Warnings                   []engineipc.Warning
	Export                     drawioSmokeExport
	Process                    drawioSmokeProcess
	Runner                     drawioSmokeRunner
	Failures                   []string
}

func TestDrawioManagedRealBundleSmoke(t *testing.T) {
	if os.Getenv(drawioManagedSmokeEnable) != "1" {
		t.Skip("real managed draw.io smoke requires explicit LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE=1")
	}

	record := drawioSmokeRecord{
		Schema:     1,
		Status:     "FAIL",
		StartedUTC: time.Now().UTC().Format(time.RFC3339Nano),
	}
	fail := func(format string, args ...any) {
		message := fmt.Sprintf(format, args...)
		record.Failures = append(record.Failures, message)
		t.Fatal(message)
	}

	candidate := drawioSmokeAbsoluteEnv(drawioManagedSmokeCandidate)
	if candidate == "" {
		fail("%s must name the absolute candidate release directory", drawioManagedSmokeCandidate)
	}
	evidencePath := drawioSmokeAbsoluteEnv(drawioManagedSmokeEvidence)
	if evidencePath == "" {
		fail("%s must name the absent absolute JSON evidence path", drawioManagedSmokeEvidence)
	}
	configRoot := drawioSmokeAbsoluteEnv(drawioManagedSmokeConfig)
	if configRoot == "" {
		fail("%s must name the absent absolute run-owned config root", drawioManagedSmokeConfig)
	}
	evidenceRoot := filepath.Dir(evidencePath)
	record.CandidateRelease = candidate
	record.EvidencePath = evidencePath
	record.EvidenceRoot = evidenceRoot
	record.ConfigRoot = configRoot

	t.Cleanup(func() {
		record.FinishedUTC = time.Now().UTC().Format(time.RFC3339Nano)
		record.Status = "PASS"
		if t.Failed() || len(record.Failures) != 0 {
			record.Status = "FAIL"
		}
		encoded, err := json.MarshalIndent(record, "", "  ")
		if err != nil {
			t.Errorf("encode draw.io managed smoke record: %v", err)
			return
		}
		file, err := os.OpenFile(evidencePath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			t.Errorf("create draw.io managed smoke evidence: %v", err)
			return
		}
		if _, err = file.Write(append(encoded, '\n')); err == nil {
			err = file.Sync()
		}
		closeErr := file.Close()
		if err = errors.Join(err, closeErr); err != nil {
			t.Errorf("write draw.io managed smoke evidence: %v", err)
		}
		t.Log("drawio-managed-real-smoke-record-json\n" + string(encoded))
	})

	exe, err := os.Executable()
	if err != nil {
		fail("locate test executable: %v", err)
	}
	bundleCopy := filepath.Dir(exe)
	record.BundleCopy = bundleCopy
	if !drawioSmokePathWithin(bundleCopy, evidenceRoot) || drawioSmokeSamePath(bundleCopy, evidenceRoot) {
		fail("test executable must run from a bundle COPY below evidence root: bundle=%s evidence=%s",
			bundleCopy, evidenceRoot)
	}
	if drawioSmokePathWithin(evidenceRoot, candidate) || drawioSmokePathWithin(candidate, evidenceRoot) {
		fail("candidate release and evidence root must be separate directory trees")
	}
	if drawioSmokePathWithin(evidencePath, bundleCopy) {
		fail("JSON evidence must be outside the evidence-owned bundle COPY")
	}
	if !drawioSmokePathWithin(configRoot, evidenceRoot) || drawioSmokeSamePath(configRoot, evidenceRoot) {
		fail("config root must be a fresh run-owned directory below evidence root")
	}
	if drawioSmokePathWithin(configRoot, bundleCopy) || drawioSmokePathWithin(bundleCopy, configRoot) {
		fail("config root and bundle COPY must be separate directory trees")
	}
	if _, err = os.Lstat(configRoot); !errors.Is(err, os.ErrNotExist) {
		fail("config root must not exist before the smoke: %v", err)
	}
	if _, err = os.Lstat(evidencePath); !errors.Is(err, os.ErrNotExist) {
		fail("evidence path must not exist before the smoke: %v", err)
	}
	if err = os.MkdirAll(configRoot, 0700); err != nil {
		fail("create run-owned config root: %v", err)
	}

	descriptor, ok := enginecatalog.Lookup(enginecatalog.DrawioSVG)
	if !ok {
		fail("draw.io engine descriptor is unavailable")
	}
	candidateExePath := filepath.Join(candidate, filepath.FromSlash(descriptor.RelativePath), descriptor.Filename)
	candidateExeBefore, err := os.ReadFile(candidateExePath)
	if err != nil {
		fail("read candidate draw.io executable: %v", err)
	}
	record.CandidateExeSHA256Before = managedSmokeHash(candidateExeBefore)
	candidateMetadata, err := managedSmokeSourceMetadata(candidate, enginecatalog.DrawioSVG)
	if err != nil {
		fail("verify candidate release metadata: %v", err)
	}
	if candidateMetadata.ExeSHA256 != record.CandidateExeSHA256Before {
		fail("candidate executable hash differs between literal read and manifest verification")
	}
	record.CandidateVersion = candidateMetadata.Version
	bundledMetadata, err := managedSmokeSourceMetadata(bundleCopy, enginecatalog.DrawioSVG)
	if err != nil {
		fail("verify evidence-owned bundled draw.io metadata: %v", err)
	}
	if bundledMetadata != candidateMetadata {
		fail("evidence bundle COPY differs from candidate draw.io metadata")
	}

	engineRoot := filepath.Join(configRoot, "engine-packages")
	if err = os.MkdirAll(filepath.Join(engineRoot, string(enginecatalog.DrawioSVG)), 0700); err != nil {
		fail("create run-owned engine state root: %v", err)
	}
	legacyState := []byte(`{"schema":1,"selections":[` +
		`{"engine":"levnote.markdown-export","digest":"","version":""},` +
		`{"engine":"levnote.mindmap-svg","digest":"","version":""},` +
		`{"engine":"levnote.mindmap-preview","digest":"","version":""}]}`)
	statePath := filepath.Join(engineRoot, packageStateName)
	if err = os.WriteFile(statePath, legacyState, 0600); err != nil {
		fail("write legacy schema1 state fixture: %v", err)
	}
	record.LegacyStateBytes = len(legacyState)
	record.LegacyStateSHA256 = managedSmokeHash(legacyState)

	manager, err := NewPackageManager(bundleCopy, configRoot)
	if err != nil {
		fail("initialize package manager: %v", err)
	}
	if manager.LoadError() != nil {
		fail("load legacy schema1 state: %v", manager.LoadError())
	}
	stateAfterLoad, err := os.ReadFile(statePath)
	if err != nil {
		fail("read legacy state after load: %v", err)
	}
	record.LegacyStateUnchangedOnLoad = bytes.Equal(stateAfterLoad, legacyState)
	if !record.LegacyStateUnchangedOnLoad {
		fail("legacy schema1 state was rewritten during load")
	}
	record.InitialSelections = drawioSmokeLegacySelections(manager)
	if record.InitialSelections != (drawioSmokeSelections{"bundled", "bundled", "bundled"}) {
		fail("legacy old3 selections are not bundled: %+v", record.InitialSelections)
	}
	record.BundledFourthSelection = managedSmokeSelection(manager, enginecatalog.DrawioSVG)
	if record.BundledFourthSelection != "bundled" {
		fail("fourth engine is not initially bundled: %s", record.BundledFourthSelection)
	}

	receipt, err := manager.ImportFromReleaseDir(context.Background(), candidate, enginecatalog.DrawioSVG)
	if err != nil {
		fail("import candidate draw.io package: %v", err)
	}
	record.ImportDigest = receipt.Digest
	record.AfterImportSelections = drawioSmokeLegacySelections(manager)
	record.ImportFourthSelection = managedSmokeSelection(manager, enginecatalog.DrawioSVG)
	if receipt.Engine != enginecatalog.DrawioSVG || receipt.Version != candidateMetadata.Version ||
		receipt.Digest == "" || record.AfterImportSelections != record.InitialSelections ||
		record.ImportFourthSelection != "bundled" {
		fail("import identity or no-implicit-activation contract failed: receipt=%+v old3=%+v fourth=%s",
			receipt, record.AfterImportSelections, record.ImportFourthSelection)
	}
	if err = manager.Activate(context.Background(), enginecatalog.DrawioSVG, receipt.Digest); err != nil {
		fail("activate exact imported draw.io package: %v", err)
	}
	record.ActivatedFourthSelection = managedSmokeSelection(manager, enginecatalog.DrawioSVG)
	record.AfterActivateSelections = drawioSmokeLegacySelections(manager)
	if record.ActivatedFourthSelection != "installed:"+receipt.Digest ||
		record.AfterActivateSelections != record.InitialSelections {
		fail("activation selected the wrong package or disturbed old3 selections: fourth=%s old3=%+v",
			record.ActivatedFourthSelection, record.AfterActivateSelections)
	}

	// Frozen fixture and oracle from the accepted draw.io worker second-page test.
	source := []byte(`<mxfile><diagram id="first" name="One"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="one" value="one" vertex="1" parent="1"><mxGeometry x="0" y="0" width="40" height="20" as="geometry"/></mxCell></root></mxGraphModel></diagram><diagram id="second" name="Two"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="two" value="two" vertex="1" parent="1"><mxGeometry x="10" y="20" width="80" height="40" as="geometry"/></mxCell></root></mxGraphModel></diagram></mxfile>`)
	sourceBefore := append([]byte(nil), source...)
	record.SourceBytes = len(source)
	record.SourceSHA256Before = managedSmokeHash(source)
	record.Export.PageIndex = 1
	record.Export.PageID = "second"

	runner, err := NewManaged(manager)
	if err != nil {
		fail("construct managed runner: %v", err)
	}
	record.Export.NewManagedCalls++
	process := &record.Process
	process.ExpectedImagePath = filepath.Join(
		manager.installedPath(enginecatalog.DrawioSVG, receipt.Digest),
		descriptor.Filename,
	)
	runner.hooks = &testHooks{
		afterAdmission: func() { record.Export.AdmissionCalls++ },
		afterVerify: func(bundle *verifiedBundle) {
			record.Export.VerifyCalls++
			if bundle == nil || bundle.info.ID != enginecatalog.DrawioSVG ||
				bundle.info.Source != PackageInstalled || bundle.info.PackageDigest != receipt.Digest {
				record.Export.VerifyIdentityError = "verified bundle identity differs from imported package"
			}
		},
		afterStart: func(started *os.Process) {
			record.Export.StartCalls++
			process.observeStart(started)
		},
		afterWait: func() {
			record.Export.WaitCalls++
			process.observeWait()
		},
	}
	runnerClosed := false
	t.Cleanup(func() {
		process.closeFallback()
		if runnerClosed {
			return
		}
		record.Runner.FallbackCloseCalled = true
		if closeErr := runner.Close(); closeErr != nil {
			record.Runner.FallbackCloseError = closeErr.Error()
			record.Failures = append(record.Failures, "fallback Runner.Close: "+closeErr.Error())
			t.Errorf("fallback Runner.Close: %v", closeErr)
		}
	})

	page := &engineipc.RequestPage{Index: 1, ID: "second"}
	record.Export.ExportCalls++
	result, exportErr := runner.Export(context.Background(), engineipc.DrawioSVG, Source{
		Size: int64(len(source)), Data: bytes.NewReader(source), Page: page,
	})
	record.Export.Returned = true
	if exportErr != nil {
		record.Export.Error = exportErr.Error()
	}
	if err = runner.Close(); err != nil {
		record.Runner.CloseError = err.Error()
	}
	record.Runner.CloseCalled = true
	runnerClosed = true
	process.closePrimary()

	check := func(ok bool, format string, args ...any) {
		if ok {
			return
		}
		message := fmt.Sprintf(format, args...)
		record.Failures = append(record.Failures, message)
		t.Error(message)
	}
	check(exportErr == nil, "managed draw.io export failed: %v", exportErr)
	check(result.Engine == (EngineInfo{
		ID: enginecatalog.DrawioSVG, Version: candidateMetadata.Version,
		SHA256: candidateMetadata.ExeSHA256, Source: PackageInstalled, PackageDigest: receipt.Digest,
	}), "engine metadata differs from imported candidate: %+v", result.Engine)
	check(record.Export.NewManagedCalls == 1 && record.Export.ExportCalls == 1 &&
		record.Export.AdmissionCalls == 1 && record.Export.VerifyCalls == 1 &&
		record.Export.StartCalls == 1 && record.Export.WaitCalls == 1,
		"managed operation count differs: %+v", record.Export)
	check(record.Export.VerifyIdentityError == "", "%s", record.Export.VerifyIdentityError)
	check(process.StartObserved && process.WaitObserved,
		"process Start/Wait observation is incomplete: %+v", process)
	check(process.DuplicateHandleObserved && process.DuplicateError == "",
		"precise process handle observation failed: %s", process.DuplicateError)
	check(process.ImagePathObserved && strings.EqualFold(
		filepath.Clean(process.ImagePath), filepath.Clean(process.ExpectedImagePath),
	), "process image=%q want=%q error=%s", process.ImagePath, process.ExpectedImagePath, process.ImagePathError)
	check(process.CreationObserved && process.CreationFiletime100ns != 0,
		"process creation identity unavailable: error=%s filetime=%d",
		process.TimesError, process.CreationFiletime100ns)
	check(process.ExitCode != nil && *process.ExitCode == 0,
		"process exit code unobserved or nonzero: code=%v error=%s",
		process.ExitCode, process.ExitCodeError)
	check(process.PrimaryClosed && process.PrimaryCloseError == "",
		"precise process handle did not close cleanly: %s", process.PrimaryCloseError)
	check(!record.Runner.FallbackCloseCalled && record.Runner.CloseError == "",
		"Runner.Close failed or needed fallback: %+v", record.Runner)

	record.SourceSHA256After = managedSmokeHash(source)
	check(bytes.Equal(source, sourceBefore), "source bytes changed after managed export")
	record.PayloadBytes = len(result.Payload)
	record.PayloadSHA256 = managedSmokeHash(result.Payload)
	const expectedSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="-14 -4 128 88" width="128" height="88"><defs><marker id="levnote-arrow" markerWidth="10" markerHeight="7" refX="9" refY="3.5" orient="auto" markerUnits="strokeWidth"><path d="M0,0 L10,3.5 L0,7 z" fill="context-stroke"/></marker></defs><rect x="-14" y="-4" width="128" height="88" fill="#ffffff"/><g font-family="Arial, sans-serif" font-size="14"><rect x="10" y="20" width="80" height="40" fill="#dae8fc" stroke="#6c8ebf" stroke-width="1.5"/><text x="50" y="44.9" text-anchor="middle" fill="#111111"><tspan x="50" dy="0">two</tspan></text></g></svg>` + "\n"
	check(bytes.Equal(result.Payload, []byte(expectedSVG)),
		"managed SVG differs from independent second-page oracle")
	wantWarnings := []engineipc.Warning{}
	check(drawioSmokeWarningsEqual(result.Warnings, wantWarnings),
		"managed warning metadata/order differs: %+v", result.Warnings)
	record.Warnings = append([]engineipc.Warning(nil), result.Warnings...)

	candidateExeAfter, err := os.ReadFile(candidateExePath)
	check(err == nil, "read candidate executable after smoke: %v", err)
	record.CandidateExeSHA256After = managedSmokeHash(candidateExeAfter)
	check(record.CandidateExeSHA256After == record.CandidateExeSHA256Before,
		"candidate executable changed during smoke")
}

func drawioSmokeAbsoluteEnv(name string) string {
	value := os.Getenv(name)
	if value == "" || !filepath.IsAbs(value) {
		return ""
	}
	return filepath.Clean(value)
}

func drawioSmokeLegacySelections(manager *PackageManager) drawioSmokeSelections {
	return drawioSmokeSelections{
		Markdown:       managedSmokeSelection(manager, enginecatalog.MarkdownHTML),
		MindmapSVG:     managedSmokeSelection(manager, enginecatalog.MindmapSVG),
		MindmapPreview: managedSmokeSelection(manager, enginecatalog.MindmapPreview),
	}
}

func drawioSmokePathWithin(path, root string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." &&
		!strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func drawioSmokeSamePath(left, right string) bool {
	return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
}

func drawioSmokeWarningsEqual(left, right []engineipc.Warning) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func (p *drawioSmokeProcess) observeStart(process *os.Process) {
	started := time.Now()
	p.StartObserved = true
	p.StartUTC, p.PID = started.UTC().Format(time.RFC3339Nano), process.Pid
	current, err := syscall.GetCurrentProcess()
	if err != nil {
		p.DuplicateError = "GetCurrentProcess: " + err.Error()
		return
	}
	var duplicate syscall.Handle
	var duplicateErr error
	err = process.WithHandle(func(handle uintptr) {
		duplicateErr = syscall.DuplicateHandle(
			current, syscall.Handle(handle), current, &duplicate, 0, false, syscall.DUPLICATE_SAME_ACCESS,
		)
	})
	if err = errors.Join(err, duplicateErr); err != nil {
		p.DuplicateError = "DuplicateHandle: " + err.Error()
		return
	}
	p.handle = duplicate
	p.DuplicateHandleObserved = true
	p.DuplicateHandle = uint64(duplicate)

	var image [32768]uint16
	length := uint32(len(image))
	ok, _, queryErr := resourceImageName.Call(
		uintptr(p.handle), 0, uintptr(unsafe.Pointer(&image[0])), uintptr(unsafe.Pointer(&length)),
	)
	if ok == 0 {
		p.ImagePathError = resourceCallError("QueryFullProcessImageNameW", queryErr)
	} else {
		p.ImagePathObserved = true
		p.ImagePath = syscall.UTF16ToString(image[:length])
	}

	var creation, exit, kernel, user syscall.Filetime
	if err = syscall.GetProcessTimes(p.handle, &creation, &exit, &kernel, &user); err != nil {
		p.TimesError = "GetProcessTimes: " + err.Error()
		return
	}
	p.CreationObserved = true
	p.CreationFiletime100ns = uint64(creation.HighDateTime)<<32 | uint64(creation.LowDateTime)
	p.CreationUTC = resourceFiletimeUTC(creation)
}

func (p *drawioSmokeProcess) observeWait() {
	p.WaitObserved = true
	p.WaitUTC = time.Now().UTC().Format(time.RFC3339Nano)
	if p.handle == 0 {
		p.ExitCodeError = "process handle was not observed"
		return
	}
	var code uint32
	if err := syscall.GetExitCodeProcess(p.handle, &code); err != nil {
		p.ExitCodeError = "GetExitCodeProcess: " + err.Error()
		return
	}
	p.ExitCode = &code
}

func (p *drawioSmokeProcess) closePrimary() {
	p.PrimaryCloseAttempted = true
	p.PrimaryCloseError, p.PrimaryClosed = p.close()
}

func (p *drawioSmokeProcess) closeFallback() {
	if p.handle == 0 {
		return
	}
	p.FallbackCloseAttempted = true
	p.FallbackCloseError, p.FallbackClosed = p.close()
}

func (p *drawioSmokeProcess) close() (string, bool) {
	if p.handle == 0 {
		return "precise process handle was not observed", false
	}
	handle := p.handle
	p.handle = 0
	if err := syscall.CloseHandle(handle); err != nil {
		return "CloseHandle: " + err.Error(), false
	}
	return "", true
}
