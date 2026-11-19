//go:build windows

package engineruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"levnote.local/native/core/engineipc"
)

const resourceSampleInterval = 2 * time.Millisecond
const resourceMaxAttempts = 20000

var resourceRequestUsed atomic.Bool

var resourceMemoryInfo = kernel32.NewProc("K32GetProcessMemoryInfo")
var resourceImageName = kernel32.NewProc("QueryFullProcessImageNameW")
var resourceWindowsVersion = syscall.NewLazyDLL("ntdll.dll").NewProc("RtlGetVersion")

type resourceArtifact struct {
	Path, SHA256 string
	Bytes        int64
}

type resourceFixture struct {
	name, source, sourceSHA, htmlSHA, warningSHA string
	htmlBytes, warningCount                      int
}

func workerResourceFixture(name string) (resourceFixture, error) {
	switch name {
	case "EscapedFence1MiB":
		return resourceFixture{name, "```\n" + strings.Repeat("&", 1<<20) + "\n```\n",
			"0371420aad5aa78a8f68488a92db63f7ce3ad24cd1c713f507879bbf5ba9dbec",
			"3042906ac2b1f579e799b92f5dc9b8a180a28e114177b60852a1cc1dd35fc0d9",
			"4f53cda18c2baa0c0354bb5f9a3ecbe5ed12ab4d8e11ba873c2f11161202b945", 5242960, 0}, nil
	case "Mixed32":
		const mixed = "# Repeated 字 heading\n\n" +
			"Plain <&> \"quote\" 'apostrophe', **bold & 字**, *italic*, `code<`, [local](./file.md?a=1&b=2) and [refused](https://example.com).\n\n" +
			"3. [x] done\n4. [ ] later\n\n" +
			"| A & | B |\n| :--- | ---: |\n| <tag> | **literal** |\n\n" +
			"> quote <&>\n\n```go\nfmt.Println(\"<&>\")\n```\n\n<script>raw</script>\n\n---\n\n"
		return resourceFixture{name, strings.Repeat(mixed, 32),
			"40ba0b8857ca84e06a5968b7129ffa9eec2875bd1f57c831a4d6680f5c47e221",
			"30969c1a616107e23cafaf3ba572993be2ea3dad3d5c9f41d7d9350398b9d0a4",
			"60c451822ff9073813ed711901c448f5111d1966416583672ea95dd719ee8017", 24203, 32}, nil
	default:
		return resourceFixture{}, errors.New("unknown fixed resource fixture")
	}
}

func resourceAcceptedHashes(version string) (worker, manifest string, err error) {
	switch version {
	case "native-worker.20261109.1":
		return "db8f02e6e187833fb52b4fc02b076b4ac96e357fb3e554533a698d8f050f6ad0", "64472251bd737d3bb82afb05c35f648882d2e9d8d4a57bcfd1cde8cf457cc453", nil
	case "native-worker.20261109.2":
		return "94db0fc39a538d78af06a1a40384d059bfc86a8790a57655424377f7754f16ac", "790dccc0a968d59a820392db6c2da71392b8768ee1a05c05e7b2221b2a132f73", nil
	default:
		return "", "", errors.New("resource probe only accepts the two reviewed releases")
	}
}

func resourceHash(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func resourceFile(path string) (resourceArtifact, error) {
	result := resourceArtifact{Path: path}
	f, err := os.Open(path)
	if err != nil {
		return result, err
	}
	hash := sha256.New()
	result.Bytes, err = io.Copy(hash, f)
	result.SHA256 = hex.EncodeToString(hash.Sum(nil))
	return result, errors.Join(err, f.Close())
}

// The layout is PROCESS_MEMORY_COUNTERS_EX. All sizes are bytes. PagefileUsage
// and PrivateUsage are process commit counters, not physical pagefile I/O.
type resourceMemoryCounters struct {
	CB, PageFaultCount                                 uint32
	PeakWorkingSetSize, WorkingSetSize                 uintptr
	QuotaPeakPagedPoolUsage, QuotaPagedPoolUsage       uintptr
	QuotaPeakNonPagedPoolUsage, QuotaNonPagedPoolUsage uintptr
	PagefileUsage, PeakPagefileUsage, PrivateUsage     uintptr
}

type resourceMemoryRead struct {
	AtUTC                    string
	OffsetNS, QueryElapsedNS int64
	Phase, Error             string
	Counters                 *resourceMemoryCounters
}

type resourceProcessTimes struct {
	CreationFiletime100ns, ExitFiletime100ns uint64
	Kernel100ns, User100ns                   uint64
	CreationUTC, ExitUTC                     string
	LifetimeNS                               int64
}

type resourceTimesRead struct {
	AtUTC                    string
	OffsetNS, QueryElapsedNS int64
	Error                    string
	Times                    *resourceProcessTimes
}

type resourceCounterSummary struct {
	Attempts, SuccessfulReads, FailedReads                     int
	FirstAttemptUTC, LastAttemptUTC                            string
	ObservationWindowNS                                        int64
	SampledMaxWorkingSetBytes, SampledMaxPrivateUsageBytes     uint64
	ObservedOSPeakWorkingSetBytes, ObservedOSPeakPagefileBytes uint64
	FinalMemoryReadOK, LifetimePeakCountersComplete            bool
	FinalCPUReadOK                                             bool
	PeakCoverage                                               string
}

type resourceProbe struct {
	PID                           int
	ImagePath, IdentityError      string
	AfterStartUTC, AfterWaitUTC   string
	BeforeInputHookElapsedNS      int64
	Duplicated, AfterWaitObserved bool
	SamplerStarted, SamplerJoined bool
	AttemptLimitReached           bool
	Memory                        []resourceMemoryRead
	InitialTimes, FinalTimes      *resourceTimesRead
	WaitState, ExitCode           *uint32
	WaitError, ExitCodeError      string
	HandleCloseError              string
	HandleClosed                  bool
	Summary                       resourceCounterSummary
	Errors                        []string
	anchor                        time.Time
	handle                        syscall.Handle
	stop, done                    chan struct{}
}

func resourceUTC(at time.Time) string { return at.UTC().Format(time.RFC3339Nano) }

func resourceCallError(api string, err error) string {
	if err == nil || errors.Is(err, syscall.Errno(0)) {
		return api + " returned failure without a Windows error code"
	}
	return api + ": " + err.Error()
}

func (p *resourceProbe) sample(phase string) {
	at := time.Now()
	read := resourceMemoryRead{AtUTC: resourceUTC(at), OffsetNS: at.Sub(p.anchor).Nanoseconds(), Phase: phase}
	counters := resourceMemoryCounters{}
	counters.CB = uint32(unsafe.Sizeof(counters))
	ok, _, err := resourceMemoryInfo.Call(uintptr(p.handle), uintptr(unsafe.Pointer(&counters)), unsafe.Sizeof(counters))
	read.QueryElapsedNS = time.Since(at).Nanoseconds()
	if ok == 0 {
		read.Error = resourceCallError("K32GetProcessMemoryInfo", err)
	} else {
		read.Counters = &counters
	}
	p.Memory = append(p.Memory, read)
}

func resourceFiletime(value syscall.Filetime) uint64 {
	return uint64(value.HighDateTime)<<32 | uint64(value.LowDateTime)
}

func resourceFiletimeUTC(value syscall.Filetime) string {
	if resourceFiletime(value) == 0 {
		return ""
	}
	return resourceUTC(time.Unix(0, value.Nanoseconds()))
}

func (p *resourceProbe) times() *resourceTimesRead {
	at := time.Now()
	read := &resourceTimesRead{AtUTC: resourceUTC(at), OffsetNS: at.Sub(p.anchor).Nanoseconds()}
	var creation, exit, kernel, user syscall.Filetime
	err := syscall.GetProcessTimes(p.handle, &creation, &exit, &kernel, &user)
	read.QueryElapsedNS = time.Since(at).Nanoseconds()
	if err != nil {
		read.Error = "GetProcessTimes: " + err.Error()
		return read
	}
	values := &resourceProcessTimes{
		CreationFiletime100ns: resourceFiletime(creation), ExitFiletime100ns: resourceFiletime(exit),
		Kernel100ns: resourceFiletime(kernel), User100ns: resourceFiletime(user),
		CreationUTC: resourceFiletimeUTC(creation), ExitUTC: resourceFiletimeUTC(exit),
	}
	if values.ExitFiletime100ns >= values.CreationFiletime100ns {
		values.LifetimeNS = int64(values.ExitFiletime100ns-values.CreationFiletime100ns) * 100
	}
	read.Times = values
	return read
}

func (p *resourceProbe) afterStart(process *os.Process) {
	start := time.Now()
	p.AfterStartUTC, p.PID = resourceUTC(start), process.Pid
	defer func() { p.BeforeInputHookElapsedNS = time.Since(start).Nanoseconds() }()
	current, err := syscall.GetCurrentProcess() // Pseudo handle; never close it.
	if err != nil {
		p.Errors = append(p.Errors, "GetCurrentProcess: "+err.Error())
		return
	}
	var duplicateErr error
	err = process.WithHandle(func(handle uintptr) {
		duplicateErr = syscall.DuplicateHandle(current, syscall.Handle(handle), current, &p.handle, 0, false, syscall.DUPLICATE_SAME_ACCESS)
	})
	if err = errors.Join(err, duplicateErr); err != nil {
		p.Errors = append(p.Errors, "duplicate exact process handle: "+err.Error())
		return
	}
	p.Duplicated = true
	var image [32768]uint16
	length := uint32(len(image))
	ok, _, queryErr := resourceImageName.Call(uintptr(p.handle), 0, uintptr(unsafe.Pointer(&image[0])), uintptr(unsafe.Pointer(&length)))
	if ok == 0 {
		p.IdentityError = resourceCallError("QueryFullProcessImageNameW", queryErr)
	} else {
		p.ImagePath = syscall.UTF16ToString(image[:length])
	}
	p.InitialTimes = p.times()
	p.sample("before-input")
	p.stop, p.done = make(chan struct{}), make(chan struct{})
	p.SamplerStarted = true
	go func() {
		defer close(p.done)
		ticker := time.NewTicker(resourceSampleInterval)
		defer ticker.Stop()
		for {
			select {
			case <-p.stop:
				return
			case <-ticker.C:
				// Reserve one final read after Wait. Never wait for a sample quota.
				if len(p.Memory) >= resourceMaxAttempts-1 {
					p.AttemptLimitReached = true
					return
				}
				p.sample("interval")
			}
		}
	}()
}

func (p *resourceProbe) joinSampler() {
	if p.stop == nil || p.SamplerJoined {
		return
	}
	close(p.stop)
	<-p.done
	p.SamplerJoined = true
}

func (p *resourceProbe) afterWait() {
	p.AfterWaitObserved, p.AfterWaitUTC = true, resourceUTC(time.Now())
	p.joinSampler()
	if p.handle != 0 {
		p.sample("after-wait")
		p.FinalTimes = p.times()
	}
}

func (p *resourceProbe) finish() {
	p.joinSampler()
	if p.handle != 0 {
		state, err := syscall.WaitForSingleObject(p.handle, 0)
		if err != nil {
			p.WaitError = err.Error()
		} else {
			p.WaitState = &state
		}
		var code uint32
		if err := syscall.GetExitCodeProcess(p.handle, &code); err != nil {
			p.ExitCodeError = err.Error()
		} else {
			p.ExitCode = &code
		}
		if err := syscall.CloseHandle(p.handle); err != nil {
			p.HandleCloseError = err.Error()
		} else {
			p.HandleClosed = true
		}
		p.handle = 0
	}
	p.summarize()
}

func (p *resourceProbe) summarize() {
	summary := resourceCounterSummary{Attempts: len(p.Memory), PeakCoverage: "incomplete: final peak counters unavailable"}
	var final *resourceMemoryCounters
	for _, read := range p.Memory {
		if read.Counters == nil {
			summary.FailedReads++
			continue
		}
		summary.SuccessfulReads++
		c := read.Counters
		summary.SampledMaxWorkingSetBytes = max(summary.SampledMaxWorkingSetBytes, uint64(c.WorkingSetSize))
		summary.SampledMaxPrivateUsageBytes = max(summary.SampledMaxPrivateUsageBytes, uint64(c.PrivateUsage))
		summary.ObservedOSPeakWorkingSetBytes = max(summary.ObservedOSPeakWorkingSetBytes, uint64(c.PeakWorkingSetSize))
		summary.ObservedOSPeakPagefileBytes = max(summary.ObservedOSPeakPagefileBytes, uint64(c.PeakPagefileUsage))
		if read.Phase == "after-wait" {
			final = c
		}
	}
	if len(p.Memory) != 0 {
		first, last := p.Memory[0], p.Memory[len(p.Memory)-1]
		summary.FirstAttemptUTC, summary.LastAttemptUTC = first.AtUTC, last.AtUTC
		summary.ObservationWindowNS = last.OffsetNS + last.QueryElapsedNS - first.OffsetNS
	}
	summary.FinalMemoryReadOK = final != nil
	if final != nil {
		if final.PeakWorkingSetSize != 0 && final.PeakPagefileUsage != 0 && uint64(final.PeakWorkingSetSize) >= summary.ObservedOSPeakWorkingSetBytes && uint64(final.PeakPagefileUsage) >= summary.ObservedOSPeakPagefileBytes {
			summary.LifetimePeakCountersComplete = true
			summary.PeakCoverage = "final after-Wait OS peak counters retained through exit"
		} else {
			summary.PeakCoverage = "incomplete: after-Wait peak counters zero or below an earlier observation"
		}
	}
	summary.FinalCPUReadOK = p.FinalTimes != nil && p.FinalTimes.Times != nil && p.FinalTimes.Times.ExitFiletime100ns != 0
	p.Summary = summary
}

type resourcePlatform struct {
	GOOS, GOARCH, GoVersion, WindowsQueryError string
	LogicalCPUs, GOMAXPROCS                    int
	WindowsMajor, WindowsMinor, WindowsBuild   uint32
}

func resourcePlatformInfo() resourcePlatform {
	p := resourcePlatform{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, GoVersion: runtime.Version(), LogicalCPUs: runtime.NumCPU(), GOMAXPROCS: runtime.GOMAXPROCS(0)}
	var version struct {
		Size, Major, Minor, Build, Platform uint32
		ServicePack                         [128]uint16
	}
	version.Size = uint32(unsafe.Sizeof(version))
	status, _, _ := resourceWindowsVersion.Call(uintptr(unsafe.Pointer(&version)))
	if status != 0 {
		p.WindowsQueryError = fmt.Sprintf("RtlGetVersion NTSTATUS=0x%x", status)
	} else {
		p.WindowsMajor, p.WindowsMinor, p.WindowsBuild = version.Major, version.Minor, version.Build
	}
	return p
}

type resourceRecord struct {
	Schema                                  int
	Status, CreatedAtUTC, Fixture, Version  string
	Ordinal, MatrixPosition                 int
	Platform                                resourcePlatform
	Tester, WorkerBefore, ManifestBefore    resourceArtifact
	WorkerAfter, ManifestAfter              resourceArtifact
	SourceBytes, HTMLBytes, WarningCount    int
	SourceSHA256, HTMLSHA256, WarningSHA256 string
	Warnings                                []engineipc.Warning
	Engine                                  EngineInfo
	ExportStartUTC, ExportEndUTC            string
	ExportElapsedNS, SampleIntervalNS       int64
	MaxAttempts                             int
	ExportError, CloseError                 string
	ExportReturned, RunnerCloseReturned     bool
	Probe                                   resourceProbe
	Failures                                []string
}

// This is an opt-in measurement, not a default performance threshold test. It
// uses only the real fixed bundle and never substitutes the helper worker.
func TestBundledWorkerResources(t *testing.T) {
	if os.Getenv("LEVNOTE_WORKER_RESOURCES") != "1" {
		t.Skip("real worker resource probe requires explicit LEVNOTE_WORKER_RESOURCES=1")
	}
	if !resourceRequestUsed.CompareAndSwap(false, true) {
		t.Fatal("resource probe permits one request per fresh tester process")
	}
	output := os.Getenv("LEVNOTE_WORKER_RESOURCES_OUTPUT")
	if !filepath.IsAbs(output) {
		t.Fatal("resource output must be a new absolute JSON path")
	}
	f, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	record := resourceRecord{Schema: 1, Status: "FAIL", CreatedAtUTC: resourceUTC(time.Now()), Platform: resourcePlatformInfo(), SampleIntervalNS: resourceSampleInterval.Nanoseconds(), MaxAttempts: resourceMaxAttempts}
	defer func() {
		record.Probe.finish()
		if t.Failed() && len(record.Failures) == 0 {
			record.Failures = append(record.Failures, "probe aborted before completion; see test stderr/stdout")
		}
		encoder := json.NewEncoder(f)
		encoder.SetIndent("", "  ")
		writeErr := encoder.Encode(&record)
		if err := errors.Join(writeErr, f.Close()); err != nil {
			t.Errorf("write resource evidence: %v", err)
		}
	}()
	check := func(ok bool, description string) {
		if !ok {
			record.Failures = append(record.Failures, description)
			t.Error(description)
		}
	}
	record.Version = os.Getenv("LEVNOTE_WORKER_RESOURCES_VERSION")
	workerSHA, manifestSHA, err := resourceAcceptedHashes(record.Version)
	if err != nil {
		t.Fatal(err)
	}
	record.Ordinal, err = strconv.Atoi(os.Getenv("LEVNOTE_WORKER_RESOURCES_ORDINAL"))
	if err != nil || record.Ordinal < 1 || record.Ordinal > 3 {
		t.Fatal("resource sample ordinal must be 1, 2 or 3")
	}
	fixture, err := workerResourceFixture(os.Getenv("LEVNOTE_WORKER_RESOURCES_FIXTURE"))
	if err != nil {
		t.Fatal(err)
	}
	record.Fixture = fixture.name
	record.MatrixPosition = 2*record.Ordinal - 1
	if strings.HasSuffix(record.Version, ".2") {
		record.MatrixPosition++
	}
	if fixture.name == "Mixed32" {
		record.MatrixPosition += 6
	}
	record.SourceBytes, record.SourceSHA256 = len(fixture.source), resourceHash([]byte(fixture.source))
	check(record.SourceSHA256 == fixture.sourceSHA, "fixture source identity differs from accepted baseline")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	record.Tester, err = resourceFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	workerPath := filepath.Join(filepath.Dir(exe), "engines", "markdown-export", "levnote-markdown-export.exe")
	manifestPath := filepath.Join(filepath.Dir(workerPath), "manifest.json")
	record.WorkerBefore, err = resourceFile(workerPath)
	if err != nil {
		t.Fatal(err)
	}
	record.ManifestBefore, err = resourceFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	check(record.WorkerBefore.SHA256 == workerSHA, "worker copy is not the accepted release")
	check(record.ManifestBefore.SHA256 == manifestSHA, "manifest copy is not the accepted release")
	if t.Failed() {
		return // Never launch an unverified fixture/copy.
	}
	runner, err := NewBundled()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if !record.RunnerCloseReturned {
			if err := runner.Close(); err != nil {
				record.CloseError = err.Error()
				t.Errorf("deferred Runner.Close: %v", err)
			}
			record.RunnerCloseReturned = true
		}
	}()
	runner.hooks = &testHooks{afterStart: record.Probe.afterStart, afterWait: record.Probe.afterWait}
	start := time.Now()
	record.Probe.anchor = start
	record.ExportStartUTC = resourceUTC(start)
	result, exportErr := runner.Export(context.Background(), engineipc.MarkdownHTML, Source{Size: int64(len(fixture.source)), Data: strings.NewReader(fixture.source)})
	record.ExportElapsedNS, record.ExportEndUTC, record.ExportReturned = time.Since(start).Nanoseconds(), resourceUTC(time.Now()), true
	if exportErr != nil {
		record.ExportError = exportErr.Error()
	}
	closeErr := runner.Close()
	record.RunnerCloseReturned = true
	if closeErr != nil {
		record.CloseError = closeErr.Error()
	}
	record.Probe.finish()
	check(exportErr == nil, "real Export failed: "+record.ExportError)
	check(closeErr == nil, "Runner.Close failed: "+record.CloseError)
	p := &record.Probe
	check(len(p.Errors) == 0 && p.Duplicated, fmt.Sprintf("precise process handle acquisition failed: %v", p.Errors))
	check(p.IdentityError == "" && strings.EqualFold(filepath.Clean(p.ImagePath), filepath.Clean(workerPath)), "held process image identity differs from verified bundle: "+p.IdentityError)
	check(p.AfterWaitObserved && p.SamplerJoined && !p.AttemptLimitReached, "sampler or runtime Wait did not join within the fixed bounds")
	check(p.WaitError == "" && p.WaitState != nil && *p.WaitState == syscall.WAIT_OBJECT_0, "held process did not signal after joined runtime return: "+p.WaitError)
	check(p.ExitCodeError == "" && p.ExitCode != nil && *p.ExitCode == 0, "held process did not exit zero: "+p.ExitCodeError)
	check(p.HandleClosed && p.HandleCloseError == "", "duplicate process handle close failed: "+p.HandleCloseError)
	record.Engine, record.Warnings = result.Engine, result.Warnings
	check(result.Engine.ID == engineipc.MarkdownHTML && result.Engine.Version == record.Version && result.Engine.SHA256 == workerSHA, "returned engine identity differs from accepted bundle")
	record.HTMLBytes, record.HTMLSHA256, record.WarningCount = len(result.Payload), resourceHash(result.Payload), len(result.Warnings)
	// Match the baseline markdown.Warning JSON exactly without importing its
	// parser/exporter or using either current renderer to construct the oracle.
	type baselineWarning struct {
		Code, Message string
		Range         struct{ Start, End int }
	}
	warnings := make([]baselineWarning, len(result.Warnings))
	for i, warning := range result.Warnings {
		warnings[i].Code, warnings[i].Message = warning.Code, warning.Message
		warnings[i].Range.Start, warnings[i].Range.End = int(warning.Start), int(warning.End)
	}
	warningJSON, err := json.Marshal(warnings)
	if err != nil {
		t.Fatal(err)
	}
	record.WarningSHA256 = resourceHash(warningJSON)
	check(record.HTMLBytes == fixture.htmlBytes && record.HTMLSHA256 == fixture.htmlSHA, "full HTML identity differs from accepted baseline")
	check(record.WarningCount == fixture.warningCount && record.WarningSHA256 == fixture.warningSHA, "ordered warning identity differs from accepted baseline")
	record.WorkerAfter, err = resourceFile(workerPath)
	check(err == nil && record.WorkerAfter == record.WorkerBefore, fmt.Sprintf("worker copy changed: %v", err))
	record.ManifestAfter, err = resourceFile(manifestPath)
	check(err == nil && record.ManifestAfter == record.ManifestBefore, fmt.Sprintf("manifest copy changed: %v", err))
	if !t.Failed() {
		record.Status = "PASS"
	}
	t.Logf("resource fixture=%s version=%s ordinal=%d matrix=%d status=%s attempts=%d successful=%d peakComplete=%t cpuComplete=%t report=%s", record.Fixture, record.Version, record.Ordinal, record.MatrixPosition, record.Status, p.Summary.Attempts, p.Summary.SuccessfulReads, p.Summary.LifetimePeakCountersComplete, p.Summary.FinalCPUReadOK, output)
}
