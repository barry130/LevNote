package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"levnote.local/native/core/document"
	"levnote.local/native/core/engineipc"
	"levnote.local/native/core/engineruntime"
	"levnote.local/native/formats/markdown"
)

// Root's later full-integration gate compiles this package's test executable
// beside a freshly packaged host/engines bundle. This uses real NewBundled and
// registered app routing, with only the native SaveDialog replaced. It is
// explicitly a host-path tester, not a physical dialog/packaged-host UI claim.
// Ordinary focused tests skip without building or launching a production worker.
func TestWorkerExportBundledHostPathAcceptance(t *testing.T) {
	if os.Getenv("LEVNOTE_WORKER_BUNDLE_ACCEPTANCE") != "1" {
		t.Skip("explicit packaged bundle integration gate")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	bundleDir := filepath.Dir(exe)
	for _, relative := range []string{"levnote-native.exe", filepath.Join("engines", "markdown-export", "manifest.json")} {
		if info, err := os.Stat(filepath.Join(bundleDir, relative)); err != nil || !info.Mode().IsRegular() {
			t.Fatalf("integration tester must sit beside current packaged host+engine: %s %v", relative, err)
		}
	}
	root := t.TempDir()
	source := "# 字🙂\r\n\r\n- [x] task\r\n\r\n| A | B |\r\n| --- | --- |\r\n| <& | **字** |\r\n\r\n[external](https://example.com)\r\n\r\n<script>literal</script>\r\n"
	path := filepath.Join(root, "bundled.md")
	originalBytes := append([]byte{0xef, 0xbb, 0xbf}, []byte(source)...)
	if err := os.WriteFile(path, originalBytes, 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	if tab == nil || tab.Adapter == nil || tab.Adapter.ExportEngine == nil || tab.Adapter.ExportEngine.ID != string(engineipc.MarkdownHTML) {
		t.Fatal("actual Markdown export descriptor missing")
	}
	if !a.setMode(tab, "source") || tab.View != nil {
		t.Fatal("source/noView capability route changed")
	}
	runner, err := engineruntime.NewBundled()
	if err != nil {
		t.Fatal(err)
	}
	a.exporter = runner
	queue := make(chan func(), 4)
	a.uiDispatch = func(fn func()) { queue <- fn }
	destination := filepath.Join(root, "bundled.html")
	dialogs := 0
	a.exportDialog = func(mygo.SaveDialogOptions) (string, error) { dialogs++; return destination, nil }
	version, selection, history, dirty, baseline := tab.Doc.Buffer.Version(), tab.Doc.Buffer.Selection(), tab.Doc.Buffer.History(), tab.Doc.Dirty(), tab.Doc.DiskToken()
	start := time.Now()
	a.exportCurrent()
	job := a.exportJob
	select {
	case <-job.done:
	case <-time.After(40 * time.Second):
		t.Fatal("production worker did not complete within runner bound")
	}
	job.mu.Lock()
	runtimeErr := job.err
	warnings := append([]engineipc.Warning(nil), job.result.Warnings...)
	engine := job.result.Engine
	job.mu.Unlock()
	if runtimeErr != nil {
		t.Fatalf("production runner: %v", runtimeErr)
	}
	drainOneExportCallback(t, queue)
	want, wantWarnings, err := markdown.ExportHTMLChecked(markdown.Parse(source))
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(destination)
	if err != nil || string(got) != want || !bytes.Contains(got, []byte("&lt;&amp;")) || len(warnings) != len(wantWarnings) || len(warnings) == 0 {
		t.Fatalf("bundled export parity/warnings: %v", err)
	}
	for i, warning := range warnings {
		oracle := wantWarnings[i]
		if warning.Code != oracle.Code || warning.Message != oracle.Message || int(warning.Start) != oracle.Range.Start || int(warning.End) != oracle.Range.End {
			t.Fatal("production warning metadata drift")
		}
	}
	for _, fragment := range []string{`<h1 id="字">字🙂</h1>`, `<input type="checkbox" disabled checked> task`, `<th>A</th>`, `<td>&lt;&amp;</td>`, `<td>**字**</td>`, `<p>external</p>`, `&lt;script&gt;literal&lt;/script&gt;`} {
		if !bytes.Contains(got, []byte(fragment)) {
			t.Fatalf("independent bundled HTML fragment missing: %s", fragment)
		}
	}
	if bytes.Contains(got, []byte(`<a href="https://example.com"`)) || bytes.Contains(got, []byte("<script>")) {
		t.Fatal("unsafe/raw input became active HTML")
	}
	if dialogs != 1 || tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.Selection() != selection || tab.Doc.Buffer.History() != history || tab.Doc.Dirty() != dirty || !tab.Doc.SameBaseline(baseline) || tab.Doc.Buffer.String() != source {
		t.Fatal("production export changed source/baseline")
	}
	// The second real run attempts the same exclusive target; first bytes remain.
	a.exportCurrent()
	second := a.exportJob
	select {
	case <-second.done:
	case <-time.After(40 * time.Second):
		t.Fatal("second production worker did not join")
	}
	drainOneExportCallback(t, queue)
	still, err := os.ReadFile(destination)
	if err != nil || !bytes.Equal(still, got) || !strings.Contains(a.Message, "导出失败") {
		t.Fatal("production app overwrote export target")
	}
	onDisk, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(onDisk, originalBytes) || tab.Doc.Encoding != document.UTF8 || !tab.Doc.BOM || tab.Doc.Buffer.String() != source || tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.Selection() != selection || tab.Doc.Buffer.History() != history || tab.Doc.Dirty() != dirty || !tab.Doc.SameBaseline(baseline) {
		t.Fatal("two real exports changed exact source bytes/encoding/state")
	}
	if err := a.closeEngineExport(); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(got)
	record := struct {
		Scope, Tester, EngineVersion, EngineSHA256, HTMLSHA256 string
		HTMLBytes, Warnings, Dialogs                           int
		ElapsedSeconds                                         float64
		RuntimeJoined                                          bool
	}{
		"registered app host-path tester with real bundled worker; injected SaveDialog only", exe, engine.Version, engine.SHA256, hex.EncodeToString(digest[:]), len(got), len(warnings), dialogs, time.Since(start).Seconds(), true}
	data, _ := json.MarshalIndent(record, "", "  ")
	t.Log(string(data))
}

// Integration prepares the deliberately missing/altered bundle in a separate
// disposable evidence copy. This test never mutates bundle files and cannot
// choose an executable: NewBundled still resolves relative to this tester.
func TestWorkerExportBundledHostPathRefusal(t *testing.T) {
	mode := os.Getenv("LEVNOTE_WORKER_BUNDLE_REFUSAL")
	if mode == "" {
		t.Skip("explicit disposable missing/altered bundle gate")
	}
	if mode != "missing" && mode != "integrity" {
		t.Fatal("refusal mode must be missing or integrity")
	}
	root := t.TempDir()
	path := filepath.Join(root, "refusal.md")
	raw := []byte("# unchanged🙂\n")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	if !a.setMode(tab, "source") || tab.Adapter == nil || tab.Adapter.ExportEngine == nil {
		t.Fatal("real descriptor unavailable")
	}
	forbidden := &exportForbiddenView{}
	tab.View = forbidden
	runner, err := engineruntime.NewBundled()
	if err != nil {
		t.Fatal(err)
	}
	a.exporter = runner
	queue := make(chan func(), 1)
	a.uiDispatch = func(fn func()) { queue <- fn }
	destination := filepath.Join(root, "must-not-exist.html")
	dialogs := 0
	a.exportDialog = func(mygo.SaveDialogOptions) (string, error) { dialogs++; return destination, nil }
	version, selection, history, baseline := tab.Doc.Buffer.Version(), tab.Doc.Buffer.Selection(), tab.Doc.Buffer.History(), tab.Doc.DiskToken()
	a.exportCurrent()
	job := a.exportJob
	select {
	case <-job.done:
	case <-time.After(40 * time.Second):
		t.Fatal("refusal did not join")
	}
	job.mu.Lock()
	runErr, html, info := job.err, job.result.Payload, job.result.Engine
	job.mu.Unlock()
	var failed *engineruntime.Failure
	kind := engineruntime.FailureUnavailable
	if mode == "integrity" {
		kind = engineruntime.FailureIntegrity
	}
	if !errors.As(runErr, &failed) || failed.Kind != kind || html != nil || info.ID != "" {
		t.Fatalf("refusal=%v engine=%+v", runErr, info)
	}
	drainOneExportCallback(t, queue)
	if dialogs != 0 || forbidden.exports != 0 || !strings.Contains(a.Message, "导出失败") {
		t.Fatal("missing/altered bundle fell back or opened dialog")
	}
	if _, err := os.Stat(destination); !os.IsNotExist(err) {
		t.Fatal("refused export created target")
	}
	onDisk, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(onDisk, raw) || tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.Selection() != selection || tab.Doc.Buffer.History() != history || !tab.Doc.SameBaseline(baseline) {
		t.Fatal("refusal changed source")
	}
	if err := a.closeEngineExport(); err != nil {
		t.Fatal(err)
	}
	exe, _ := os.Executable()
	t.Logf("host-path refusal mode=%s tester=%s kind=%s no target/dialog/fallback", mode, exe, failed.Kind)
}
