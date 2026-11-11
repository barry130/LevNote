//go:build windows

package engineruntime

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"levnote.local/native/core/engineipc"
	"levnote.local/native/core/text"
)

const helperVersion = "runtime-test-v1"

func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "--stdio-v1" {
		mode := "valid"
		if len(os.Args) > 2 {
			mode = os.Args[2]
		}
		os.Exit(runHelper(mode))
	}
	if len(os.Args) > 1 && os.Args[1] == "--owned-descendant" {
		helperBlock()
		os.Exit(0)
	}
	if len(os.Args) > 3 && os.Args[1] == "--runtime-parent" {
		os.Exit(runParentHelper(os.Args[2], os.Args[3]))
	}
	os.Exit(m.Run())
}

func runParentHelper(phase, dir string) int {
	r := newRunner(dir)
	r.hooks = &testHooks{workerArgs: []string{"valid"}}
	var child *os.Process
	gate := func() {
		_, _ = fmt.Fprintln(os.Stdout, child.Pid)
		var p [1]byte
		_, _ = os.Stdin.Read(p[:])
	}
	r.hooks.afterStart = func(p *os.Process) {
		child = p
		if phase == "before-attach" {
			gate()
		}
	}
	if phase == "after-attach" {
		r.hooks.afterAttach = func(*processJob) { gate() }
	}
	_, err := r.Export(context.Background(), engineipc.MarkdownHTML, sourceOf("x"))
	_ = r.Close()
	if err != nil {
		return 95
	}
	return 0
}

func helperBlock() {
	reader, writer, err := os.Pipe()
	if err != nil {
		os.Exit(90)
	}
	defer writer.Close()
	var p [1]byte
	_, _ = reader.Read(p[:]) // Real blocked I/O, released only by owned termination.
}

// Independent response encoder: tests don't ask the decoder's own encoder to
// construct its oracle. The child is this repository-built test executable.
func runHelper(mode string) int {
	if mode == "no-input" {
		helperBlock()
		return 0
	}
	mindmap := strings.HasPrefix(mode, "mindmap-")
	if mindmap {
		mode = strings.TrimPrefix(mode, "mindmap-")
	}
	drawio := strings.HasPrefix(mode, "drawio-")
	if drawio {
		mode = strings.TrimPrefix(mode, "drawio-")
	}
	sourceLimit := int64(8 << 20)
	if mindmap {
		sourceLimit = 16 << 20
	} else if drawio {
		sourceLimit = 32 << 20
	}
	request, err := io.ReadAll(io.LimitReader(os.Stdin, sourceLimit+(4<<10)+25))
	if err != nil {
		return 91
	}
	if len(request) < 24 {
		return 92
	}
	metaSize := binary.LittleEndian.Uint32(request[12:16])
	if int(metaSize)+24 > len(request) {
		return 93
	}
	source := request[24+int(metaSize):]
	response := map[string]any{"engine": "levnote.markdown-export", "engineVersion": helperVersion, "format": "html", "status": "ok", "warnings": []any{}}
	payload := []byte("<p>" + html.EscapeString(string(source)) + "</p>")
	if mindmap {
		var metadata map[string]string
		if json.Unmarshal(request[24:24+int(metaSize)], &metadata) != nil || metadata["engine"] != "levnote.mindmap-svg" || metadata["format"] != "svg" || binary.LittleEndian.Uint64(request[16:24]) != uint64(len(source)) {
			return 96
		}
		response["engine"], response["format"] = "levnote.mindmap-svg", "svg"
		payload = []byte(`<svg xmlns="http://www.w3.org/2000/svg"><text>` + html.EscapeString(string(source)) + `</text></svg>`)
	}
	if drawio {
		var metadata struct {
			Engine string `json:"engine"`
			Format string `json:"format"`
			Page   *struct {
				Index int    `json:"index"`
				ID    string `json:"id"`
			} `json:"page"`
		}
		if json.Unmarshal(request[24:24+int(metaSize)], &metadata) != nil || metadata.Engine != "levnote.drawio-svg" ||
			metadata.Format != "svg" || metadata.Page == nil || binary.LittleEndian.Uint64(request[16:24]) != uint64(len(source)) {
			return 97
		}
		response["engine"], response["format"] = "levnote.drawio-svg", "svg"
		payload = []byte(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" data-page="%d" data-id="%s"><text>%s</text></svg>`,
			metadata.Page.Index, html.EscapeString(metadata.Page.ID), html.EscapeString(string(source))))
	}
	if mode == "wrong-engine" {
		response["engine"] = "levnote.markdown-export"
	}
	if mode == "wrong-format" {
		response["format"] = "html"
	}
	if mode == "warning" {
		response["warnings"] = []any{map[string]any{"code": "test-warning", "message": "test warning", "start": 0, "end": len(source)}}
	}
	if mode == "engine-error" {
		response["status"] = "error"
		response["error"] = map[string]any{"code": "partial-document", "message": "bounded test refusal"}
		payload = nil
	}
	if mode == "wrong-version" {
		response["engineVersion"] = "other"
	}
	if mode == "stderr" {
		_, _ = os.Stderr.Write([]byte("not for user display"))
	}
	if mode == "stderr-flood" {
		_, _ = os.Stderr.Write(bytes.Repeat([]byte{'x'}, maxStderrBytes+1))
		helperBlock()
		return 0
	}
	if mode == "descendant" || mode == "inherited-pipes" || mode == "descendant-cancel" {
		exe, _ := os.Executable()
		child := exec.Command(exe, "--owned-descendant")
		configureCommand(child)
		if mode == "inherited-pipes" || mode == "descendant-cancel" {
			child.Stdout = os.Stdout
			child.Stderr = os.Stderr
		}
		if child.Start() != nil {
			return 94
		}
		response["warnings"] = []any{map[string]any{"code": "owned-child", "message": fmt.Sprint(child.Process.Pid), "start": 0, "end": 0}}
		_ = child.Process.Release()
	}
	metadata, _ := json.Marshal(response)
	header := make([]byte, 24)
	copy(header, "LVEXRES\x00")
	binary.LittleEndian.PutUint16(header[8:10], 1)
	binary.LittleEndian.PutUint16(header[10:12], 24)
	binary.LittleEndian.PutUint32(header[12:16], uint32(len(metadata)))
	binary.LittleEndian.PutUint64(header[16:24], uint64(len(payload)))
	if mode == "stdout-flood" {
		binary.LittleEndian.PutUint64(header[16:24], (64<<20)+1)
		_, _ = os.Stdout.Write(header)
		helperBlock()
		return 0
	}
	if mode == "malformed" {
		header[0] = 'X'
	}
	frame := append(append(header, metadata...), payload...)
	if mode == "truncated" {
		frame = frame[:len(frame)-1]
	}
	if mode == "extra" {
		frame = append(frame, 'x')
	}
	if mode == "fragmented" {
		for _, c := range frame {
			_, _ = os.Stdout.Write([]byte{c})
		}
	} else {
		_, _ = os.Stdout.Write(frame)
	}
	if mode == "nonzero" {
		return 7
	}
	if mode == "descendant-cancel" {
		helperBlock()
	}
	return 0
}

func makeBundle(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	return dir, makeBundleFor(t, dir, engineipc.MarkdownHTML)
}

func makeBundleFor(t *testing.T, dir string, id engineipc.EngineID) string {
	t.Helper()
	// Literal fixture identities independently check the production catalog.
	engineDir, filename, format := "markdown-export", "levnote-markdown-export.exe", "html"
	if id == engineipc.MindmapSVG {
		engineDir, filename, format = "mindmap-svg", "levnote-mindmap-svg.exe", "svg"
	} else if id == engineipc.DrawioSVG {
		engineDir, filename, format = "drawio-svg", "levnote-drawio-svg.exe", "svg"
	} else if id != engineipc.MarkdownHTML {
		t.Fatal("unknown test fixture identity")
	}
	engineDir = filepath.Join(dir, "engines", engineDir)
	if err := os.MkdirAll(engineDir, 0700); err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.Open(exe)
	if err != nil {
		t.Fatal(err)
	}
	defer source.Close()
	dst := filepath.Join(engineDir, filename)
	file, err := os.Create(dst)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.New()
	n, err := io.Copy(io.MultiWriter(file, hash), source)
	closeErr := file.Close()
	if err != nil || closeErr != nil {
		t.Fatalf("copy %v / %v", err, closeErr)
	}
	m := bundleManifest{1, id, helperVersion, int(engineipc.ProtocolVersion), format, "windows", "amd64", filename, n, hex.EncodeToString(hash.Sum(nil))}
	data, _ := json.Marshal(m)
	if err := os.WriteFile(filepath.Join(engineDir, "manifest.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("helper=%s bytes=%d sha256=%s", dst, n, m.SHA256)
	return dst
}

func makeTestRunner(t *testing.T, mode string) (*Runner, string) {
	t.Helper()
	dir, exe := makeBundle(t)
	r := newRunner(dir)
	r.timeout = 5 * time.Second
	r.waitDelay = 100 * time.Millisecond
	r.hooks = &testHooks{workerArgs: []string{mode}, afterStart: func(p *os.Process) { t.Logf("owned pid=%d", p.Pid) }}
	t.Cleanup(func() {
		if err := r.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	return r, exe
}

func sourceOf(s string) Source { return Source{Size: int64(len(s)), Data: strings.NewReader(s)} }
func requireFailure(t *testing.T, result Result, err error, kind FailureKind, verified bool) {
	t.Helper()
	var failure *Failure
	if !errors.As(err, &failure) || failure.Kind != kind {
		t.Fatalf("got error %v; want %s", err, kind)
	}
	if result.Payload != nil {
		t.Fatalf("failed run returned payload: %q", result.Payload)
	}
	if (result.Engine.ID != "") != verified {
		t.Fatalf("verified=%v info=%+v", verified, result.Engine)
	}
}

func TestRuntimeSnapshotAndFrameOutcomes(t *testing.T) {
	for _, mode := range []string{"valid", "fragmented", "warning", "engine-error", "wrong-version", "nonzero", "stderr", "stderr-flood", "stdout-flood", "malformed", "truncated", "extra", "descendant", "inherited-pipes"} {
		t.Run(mode, func(t *testing.T) {
			r, _ := makeTestRunner(t, mode)
			var held *heldDescendant
			if mode == "descendant" || mode == "inherited-pipes" {
				held = observeOwnedDescendant(t, r)
			}
			buf := text.New("字形\ufeff<&\r\n✅")
			snapshot := buf.Snapshot()
			size := buf.LenBytes()
			if err := buf.ReplaceUTF16(0, 0, "later edit"); err != nil {
				t.Fatal(err)
			}
			result, err := r.Export(context.Background(), engineipc.MarkdownHTML, Source{Size: int64(size), Data: snapshot})
			if held != nil {
				held.requireExited(t)
			}
			want := map[string]FailureKind{"engine-error": FailureEngine, "wrong-version": FailureProtocol, "nonzero": FailureWorkerExit, "stderr": FailureUnexpectedStderr, "stderr-flood": FailureStderrLimit, "stdout-flood": FailureStdoutLimit, "malformed": FailureProtocol, "truncated": FailureProtocol, "extra": FailureProtocol, "inherited-pipes": FailureWorkerExit}
			if kind, ok := want[mode]; ok {
				requireFailure(t, result, err, kind, true)
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if string(result.Payload) != "<p>字形\ufeff&lt;&amp;\r\n✅</p>" {
				t.Fatalf("snapshot changed: %q", result.Payload)
			}
			if result.Engine.Version != helperVersion || len(result.Engine.SHA256) != 64 {
				t.Fatalf("metadata %+v", result.Engine)
			}
			if mode == "warning" && (len(result.Warnings) != 1 || result.Warnings[0].End != uint32(size)) {
				t.Fatalf("warnings %+v", result.Warnings)
			}
		})
	}
}

// The oracle holds a verified member process object while stdout arrives,
// before the runtime can terminate its job. Post-return PID lookup is not used.
type heldDescendant struct {
	mu     sync.Mutex
	job    *processJob
	data   []byte
	handle syscall.Handle
	pid    uint32
	err    error
	done   bool
	ready  chan struct{}
}

func observeOwnedDescendant(t *testing.T, r *Runner) *heldDescendant {
	t.Helper()
	p := &heldDescendant{ready: make(chan struct{})}
	r.hooks.afterAttach = func(job *processJob) { p.mu.Lock(); p.job = job; p.mu.Unlock() }
	r.hooks.observeStdout = p.observe
	t.Cleanup(func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		if p.handle != 0 {
			state, _ := syscall.WaitForSingleObject(p.handle, 0)
			if state != syscall.WAIT_OBJECT_0 {
				// This exact handle was checked against our job before retention.
				killErr := syscall.TerminateProcess(p.handle, 99)
				state, waitErr := syscall.WaitForSingleObject(p.handle, 5000)
				t.Errorf("fallback cleanup of held owned descendant pid=%d: terminate=%v wait=%d/%v", p.pid, killErr, state, waitErr)
			}
			if err := syscall.CloseHandle(p.handle); err != nil {
				t.Errorf("held oracle handle close: %v", err)
			}
		}
	})
	return p
}

func (p *heldDescendant) observe(data []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.done {
		return
	}
	if len(data) > 16384-len(p.data) {
		p.err = errors.New("helper metadata exceeds oracle bound")
		p.done = true
		close(p.ready)
		return
	}
	p.data = append(p.data, data...)
	if len(p.data) < 24 {
		return
	}
	metadataSize := binary.LittleEndian.Uint32(p.data[12:16])
	if metadataSize > 16360 {
		p.err = errors.New("invalid helper metadata length")
		p.done = true
		close(p.ready)
		return
	}
	if len(p.data) < 24+int(metadataSize) {
		return
	}
	p.done = true
	defer close(p.ready)
	var metadata struct {
		Warnings []struct{ Code, Message string }
	}
	if err := json.Unmarshal(p.data[24:24+int(metadataSize)], &metadata); err != nil {
		p.err = err
		return
	}
	if len(metadata.Warnings) != 1 || metadata.Warnings[0].Code != "owned-child" {
		p.err = errors.New("missing helper descendant identity")
		return
	}
	if _, err := fmt.Sscan(metadata.Warnings[0].Message, &p.pid); err != nil || p.pid == 0 {
		p.err = errors.New("invalid helper descendant PID")
		return
	}
	h, err := syscall.OpenProcess(syscall.SYNCHRONIZE|syscall.PROCESS_TERMINATE|0x1000, false, p.pid)
	if err != nil {
		p.err = err
		return
	}
	if p.job == nil {
		_ = syscall.CloseHandle(h)
		p.err = errors.New("job not attached before helper output")
		return
	}
	p.job.mu.Lock()
	var member uint32
	ok, _, memberErr := isProcessInJob.Call(uintptr(h), uintptr(p.job.handle), uintptr(unsafe.Pointer(&member)))
	p.job.mu.Unlock()
	if ok == 0 || member == 0 {
		_ = syscall.CloseHandle(h)
		p.err = errors.Join(errors.New("helper PID is not in exact owned job"), memberErr)
		return
	}
	p.handle = h
}

func (p *heldDescendant) requireExited(t *testing.T) {
	t.Helper()
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.err != nil || p.handle == 0 {
		t.Fatalf("could not retain exact descendant handle: %v", p.err)
	}
	state, err := syscall.WaitForSingleObject(p.handle, 0)
	if err != nil || state != syscall.WAIT_OBJECT_0 {
		t.Fatalf("held descendant pid=%d not terminated when Export returned: state=%d err=%v", p.pid, state, err)
	}
	t.Logf("held owned descendant pid=%d signaled before Export return", p.pid)
}

func TestRuntimeCancelWithInheritedDescendantPipesWaitsForHeldProcess(t *testing.T) {
	r, _ := makeTestRunner(t, "descendant-cancel")
	held := observeOwnedDescendant(t, r)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type outcome struct {
		result Result
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		result, err := r.Export(ctx, engineipc.MarkdownHTML, sourceOf("x"))
		done <- outcome{result, err}
	}()
	select {
	case <-held.ready:
	case <-time.After(5 * time.Second):
		t.Fatal("helper did not expose held descendant")
	}
	cancel()
	got := <-done
	requireFailure(t, got.result, got.err, FailureCanceled, true)
	held.requireExited(t)
}

func TestRuntimeEmptySourceAndReuse(t *testing.T) {
	r, _ := makeTestRunner(t, "valid")
	for range 2 {
		result, err := r.Export(context.Background(), engineipc.MarkdownHTML, sourceOf(""))
		if err != nil || string(result.Payload) != "<p></p>" {
			t.Fatalf("empty/reuse: %q %v", result.Payload, err)
		}
	}
}

func TestRuntimeCleanupFailureIsStoredForAllCloseCallers(t *testing.T) {
	dir, _ := makeBundle(t)
	r := newRunner(dir)
	r.hooks = &testHooks{workerArgs: []string{"valid"}}
	r.hooks.afterVerify = func(bundle *verifiedBundle) {
		// A real already-closed handle makes the eventual cleanup report failure;
		// no OS resource is leaked by this deliberate test-only close.
		if err := bundle.files[len(bundle.files)-1].Close(); err != nil {
			t.Fatal(err)
		}
	}
	result, err := r.Export(context.Background(), engineipc.MarkdownHTML, sourceOf("x"))
	requireFailure(t, result, err, FailureCleanup, true)
	first := r.Close()
	second := r.Close()
	if first == nil || second == nil || first != second {
		t.Fatalf("Close outcomes differ: %v / %v", first, second)
	}
	result, err = r.Export(context.Background(), engineipc.MarkdownHTML, sourceOf("x"))
	requireFailure(t, result, err, FailureClosed, false)
}

func requirePIDGone(t *testing.T, pid int) {
	t.Helper()
	h, err := syscall.OpenProcess(syscall.SYNCHRONIZE|syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))
	if err == syscall.Errno(87) { // ERROR_INVALID_PARAMETER: PID no longer exists.
		return
	}
	if err != nil {
		t.Fatalf("check owned pid%d: %v", pid, err)
	}
	defer syscall.CloseHandle(h)
	wait, err := syscall.WaitForSingleObject(h, 0)
	if err != nil || wait != syscall.WAIT_OBJECT_0 {
		t.Fatalf("owned pid%d still alive (wait=%d,err=%v)", pid, wait, err)
	}
}

func TestRuntimeRejectsBeforeLaunchAndRetainsVerification(t *testing.T) {
	r, _ := NewBundled()
	if r == nil {
		t.Fatal("NewBundled requires no worker")
	}
	_ = r.Close()
	dir, exe := makeBundle(t)
	for _, test := range []struct {
		name   string
		mutate func()
		kind   FailureKind
	}{
		{"missing", func() { _ = os.Remove(filepath.Join(dir, "engines", "markdown-export", "manifest.json")) }, FailureUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			test.mutate()
			r := newRunner(dir)
			defer r.Close()
			result, err := r.Export(context.Background(), engineipc.MarkdownHTML, sourceOf("x"))
			requireFailure(t, result, err, test.kind, false)
		})
	}
	_ = exe
	r, _ = makeTestRunner(t, "valid")
	var starts atomic.Int32
	r.hooks.afterStart = func(*os.Process) { starts.Add(1) }
	for _, src := range []Source{{Size: -1, Data: strings.NewReader("")}, {Size: engineipc.SourceMaxBytes + 1, Data: strings.NewReader("")}} {
		result, err := r.Export(context.Background(), engineipc.MarkdownHTML, src)
		requireFailure(t, result, err, FailureSourceLimit, false)
	}
	result, err := r.Export(context.Background(), "arbitrary.exe", sourceOf("x"))
	requireFailure(t, result, err, FailureUnavailable, false)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err = r.Export(ctx, engineipc.MarkdownHTML, sourceOf("x"))
	requireFailure(t, result, err, FailureCanceled, false)
	if starts.Load() != 0 {
		t.Fatal("preflight started child")
	}
}

type fakeSource struct {
	value    string
	reported int64
	err      error
}

func (s fakeSource) WriteTo(w io.Writer) (int64, error) {
	_, err := io.WriteString(w, s.value)
	if err != nil {
		return 0, err
	}
	return s.reported, s.err
}
func TestRuntimeExactWriter(t *testing.T) {
	for _, test := range []struct {
		name string
		size int64
		data io.WriterTo
	}{
		{"short", 2, strings.NewReader("x")}, {"long", 1, strings.NewReader("xx")},
		{"false-count", 1, fakeSource{"x", 0, nil}}, {"write-error", 1, fakeSource{"x", 1, errors.New("local writer error")}},
	} {
		t.Run(test.name, func(t *testing.T) {
			r, _ := makeTestRunner(t, "valid")
			result, err := r.Export(context.Background(), engineipc.MarkdownHTML, Source{Size: test.size, Data: test.data})
			requireFailure(t, result, err, FailureSourceSize, true)
		})
	}
}

func TestRuntimeCloseJoinsAllCallersAndBlocksAdmission(t *testing.T) {
	r, _ := makeTestRunner(t, "no-input")
	started := make(chan int, 1)
	joined := make(chan struct{})
	releaseJoin := make(chan struct{})
	r.hooks.afterStart = func(p *os.Process) { started <- p.Pid }
	r.hooks.afterWait = func() { close(joined); <-releaseJoin }
	exportDone := make(chan error, 1)
	go func() {
		_, err := r.Export(context.Background(), engineipc.MarkdownHTML, sourceOf(strings.Repeat("x", 1<<20)))
		exportDone <- err
	}()
	pid := <-started
	result, err := r.Export(context.Background(), engineipc.MarkdownHTML, sourceOf("x"))
	requireFailure(t, result, err, FailureBusy, false)
	closed := make(chan error, 2)
	go func() { closed <- r.Close() }()
	<-joined
	go func() { closed <- r.Close() }()
	select {
	case err := <-closed:
		t.Fatalf("Close returned before cleanup: %v", err)
	default:
	}
	result, err = r.Export(context.Background(), engineipc.MarkdownHTML, sourceOf("x"))
	requireFailure(t, result, err, FailureClosed, false)
	close(releaseJoin)
	for range 2 {
		if err := <-closed; err != nil {
			t.Fatal(err)
		}
	}
	if err := <-exportDone; !errors.Is(err, context.Canceled) {
		t.Fatalf("Export: %v", err)
	}
	requirePIDGone(t, pid)
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestRuntimeTimeoutAndAttachFailureWaitExactlyOnce(t *testing.T) {
	for _, attachFailure := range []bool{false, true} {
		t.Run(fmt.Sprint(attachFailure), func(t *testing.T) {
			r, _ := makeTestRunner(t, "no-input")
			r.timeout = 2 * time.Second
			var pid, waits int
			r.hooks.afterStart = func(p *os.Process) { pid = p.Pid }
			r.hooks.afterWait = func() { waits++ }
			if attachFailure {
				r.hooks.beforeAttach = func() error { return errors.New("injected attach failure") }
			}
			result, err := r.Export(context.Background(), engineipc.MarkdownHTML, sourceOf("x"))
			kind := FailureTimeout
			if attachFailure {
				kind = FailureUnavailable
			}
			requireFailure(t, result, err, kind, true)
			if waits != 1 {
				t.Fatalf("Wait count=%d", waits)
			}
			requirePIDGone(t, pid)
		})
	}
}

func TestRuntimeCancelAfterWaitBeforeCompletionWins(t *testing.T) {
	r, _ := makeTestRunner(t, "valid")
	joined := make(chan struct{})
	release := make(chan struct{})
	r.hooks.afterWait = func() { close(joined); <-release }
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		result, err := r.Export(context.Background(), engineipc.MarkdownHTML, sourceOf("x"))
		requireFailure(t, result, err, FailureCanceled, true)
	}()
	<-joined
	closed := make(chan error, 1)
	go func() { closed <- r.Close() }()
	// Lock establishes that Close has marked the runner before releasing Wait.
	for {
		r.mu.Lock()
		done := r.closed
		r.mu.Unlock()
		if done {
			break
		}
		time.Sleep(time.Millisecond)
	}
	close(release)
	if err := <-closed; err != nil {
		t.Fatal(err)
	}
	wg.Wait()
}

func TestRuntimeParentDeathAndUnrelatedSentinel(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	sentinel := exec.Command(exe, "--owned-descendant")
	configureCommand(sentinel)
	if err := sentinel.Start(); err != nil {
		t.Fatal(err)
	}
	t.Logf("unrelated owned sentinel pid=%d", sentinel.Process.Pid)
	defer func() { _ = sentinel.Process.Kill(); _ = sentinel.Wait() }()
	for _, phase := range []string{"before-attach", "after-attach"} {
		t.Run(phase, func(t *testing.T) {
			dir, _ := makeBundle(t)
			parent := exec.Command(exe, "--runtime-parent", phase, dir)
			configureCommand(parent)
			input, err := parent.StdinPipe()
			if err != nil {
				t.Fatal(err)
			}
			output, err := parent.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := parent.Start(); err != nil {
				t.Fatal(err)
			}
			defer func() {
				input.Close()
				if parent.ProcessState == nil {
					_ = parent.Process.Kill()
					_ = parent.Wait()
				}
			}()
			ready := make(chan string, 1)
			go func() { line, _ := bufio.NewReader(output).ReadString('\n'); ready <- line }()
			var line string
			select {
			case line = <-ready:
			case <-time.After(5 * time.Second):
				t.Fatal("parent helper never reached lifecycle gate")
			}
			var childPID int
			if _, err := fmt.Sscan(line, &childPID); err != nil || childPID <= 0 {
				t.Fatalf("invalid helper PID: %q", line)
			}
			h, err := syscall.OpenProcess(syscall.SYNCHRONIZE|syscall.PROCESS_QUERY_INFORMATION, false, uint32(childPID))
			if err != nil {
				t.Fatal(err)
			}
			defer syscall.CloseHandle(h)
			t.Logf("phase=%s exact parent=%d child=%d", phase, parent.Process.Pid, childPID)
			if err := parent.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			_ = parent.Wait()
			wait, err := syscall.WaitForSingleObject(h, 3000)
			if err != nil || wait != syscall.WAIT_OBJECT_0 {
				t.Fatalf("owned child survived parent death: wait=%d err=%v", wait, err)
			}
			var alive bool
			if err := sentinel.Process.WithHandle(func(handle uintptr) {
				state, _ := syscall.WaitForSingleObject(syscall.Handle(handle), 0)
				alive = state == syscall.WAIT_TIMEOUT
			}); err != nil {
				t.Fatal(err)
			}
			if !alive {
				t.Fatal("unrelated sentinel terminated")
			}
		})
	}
}
