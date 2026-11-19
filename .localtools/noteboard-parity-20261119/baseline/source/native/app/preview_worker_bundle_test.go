package app

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/engineipc"
	"levnote.local/native/core/engineruntime"
	"levnote.local/native/core/enginescene"
	"levnote.local/native/ui/editor"
)

// This observer delegates every operation to the actual bundled runtime. It
// counts admitted calls only; it cannot fabricate payloads or choose a path.
// No per-PID or physical-window observation is claimed by the test.
type observedPreviewRunner struct {
	runner *engineruntime.Runner
	calls  atomic.Int32
}

func (r *observedPreviewRunner) Export(ctx context.Context, id engineipc.EngineID, source engineruntime.Source) (engineruntime.Result, error) {
	r.calls.Add(1)
	return r.runner.Export(ctx, id, source)
}
func (r *observedPreviewRunner) Close() error { return r.runner.Close() }

// Ordinary test runs skip before reading a bundle, creating files, or starting
// a worker. Delivery runs this tester in a disposable complete bundle copy.
func TestMindmapPreviewBundledHostPathAcceptance(t *testing.T) {
	if os.Getenv("LEVNOTE_WORKER_BUNDLE_ACCEPTANCE") != "1" {
		t.Skip("explicit packaged readonly preview integration gate")
	}
	captureDir := os.Getenv("LEVNOTE_PREVIEW_BUNDLE_CAPTURE_DIR")
	if captureDir == "" {
		t.Fatal("explicit fresh preview capture directory is required")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	bundleDir := filepath.Dir(exe)
	for _, relative := range []string{"levnote-native.exe", "engines/mindmap-preview/levnote-mindmap-preview.exe", "engines/mindmap-preview/manifest.json"} {
		if info, err := os.Stat(filepath.Join(bundleDir, relative)); err != nil || !info.Mode().IsRegular() {
			t.Fatalf("tester needs complete fixed bundle: %s %v", relative, err)
		}
	}
	var manifest struct{ Engine, EngineVersion, Format, SHA256 string }
	if err := json.Unmarshal(mustReadFile(t, filepath.Join(bundleDir, "engines/mindmap-preview/manifest.json")), &manifest); err != nil || manifest.Engine != "levnote.mindmap-preview" || manifest.Format != "mindmap-scene-v1" || manifest.EngineVersion == "" {
		t.Fatal("wrong preview manifest", err)
	}
	root := t.TempDir()
	source := "{\r\n" +
		`"format":"levnote.mindmap","version":1,"layout":"logicalStructureRight","futureTop":1.00,` + "\r\n" +
		`"root":{"id":"r","topic":"根<&🙂","icons":["⭐"],"tags":["甲&"],"note":"private note","hyperLink":"https://invalid/private","style":{"background-color":"#aBc","border":"#1357","color":"ReBeccAPurple"},"branchColor":"#0F08","image":{"url":"private/missing.png","width":40,"height":30,"future":2.50},"children":[{"id":"a","topic":"折叠A","collapsed":true,"style":{"background-color":"none","color":"red"},"children":[{"id":"hidden","topic":"不应显示"}]},{"id":"b","topic":"可见B","branchColor":"#12345678","futureNode":9007199254740993}]}` + "\r\n}\r\n"
	originalBytes := append([]byte{0xef, 0xbb, 0xbf}, []byte(source)...)
	path := filepath.Join(root, "preview-readonly.mindmap")
	if err := os.WriteFile(path, originalBytes, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0400); err != nil {
		t.Fatal("readonly fixture mode", err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(path, 0600); err != nil {
			t.Error("restore readonly fixture", err)
		}
	})
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path) // document.Open must detect the actual filesystem readonly bit.
	tab := a.Current()
	if tab == nil || !tab.Doc.ReadOnly || !tab.Editor.ReadOnly || tab.Mode != "preview" || tab.Adapter == nil || tab.Adapter.SceneEngine == nil ||
		*tab.Adapter.SceneEngine != (editor.SceneCapability{ID: "levnote.mindmap-preview", Format: "mindmap-scene-v1"}) {
		t.Fatal("real readonly detection/registered scene capability missing")
	}
	if err := tab.Doc.Buffer.SetSelection(8, 2); err != nil {
		t.Fatal(err)
	}
	version, selection, history := tab.Doc.Buffer.Version(), tab.Doc.Buffer.Selection(), tab.Doc.Buffer.History()
	dirty, baseline := tab.Doc.Dirty(), tab.Doc.DiskToken()
	runner, err := engineruntime.NewBundled()
	if err != nil {
		t.Fatal(err)
	}
	observed := &observedPreviewRunner{runner: runner}
	a.exporter = observed
	queue := make(chan func(), 4)
	a.uiDispatch = func(fn func()) { queue <- fn }
	a.ShowOutline = true
	// Hold only delivery while NewTester settles initial frames. This uses the
	// existing modal deferral state; it neither substitutes provider nor result.
	a.modalDepth++
	started := time.Now()
	tester := ui.NewTester(a.View, 760, 480)
	job := a.previewJob
	if job == nil {
		t.Fatal("registered scene did not start", a.Message)
	}
	select {
	case <-job.done:
	case <-time.After(40 * time.Second):
		t.Fatal("production preview did not join within runner bound")
	}
	job.mu.Lock()
	runErr, result := job.err, job.result
	job.mu.Unlock()
	if runErr != nil || result.Engine.ID != "levnote.mindmap-preview" || result.Engine.Version != manifest.EngineVersion ||
		len(result.Engine.SHA256) != 64 || !strings.EqualFold(result.Engine.SHA256, manifest.SHA256) {
		t.Fatal("real preview identity/outcome", result.Engine, runErr)
	}
	// Authored from the fixed 92x40 nodes, 72px horizontal/24px vertical gaps
	// and28px padding, independently of Parse/Layout/Encode/Decode expectations.
	want := enginescene.Scene{Schema: 1, Layout: "logicalStructureRight", Bounds: enginescene.Rect{Width: 312, Height: 160},
		Nodes: []enginescene.Node{
			{ID: "r", Label: "⭐ 根<&🙂 [甲&]", X: 28, Y: 60, Width: 92, Height: 40, Shape: "rounded", Fill: "#aabbccff", Stroke: "#11335577", Text: "#663399ff", Root: true},
			{ID: "a", Label: "折叠A", X: 192, Y: 28, Width: 92, Height: 40, Shape: "rounded", Fill: "#00000000", Text: "#ff0000ff", Collapsed: true},
			{ID: "b", Label: "可见B", X: 192, Y: 92, Width: 92, Height: 40, Shape: "rounded"},
		}, Edges: []enginescene.Edge{
			{FromID: "r", ToID: "a", From: enginescene.Point{X: 120, Y: 80}, To: enginescene.Point{X: 192, Y: 48}, Stroke: "#00ff0088"},
			{FromID: "r", ToID: "b", From: enginescene.Point{X: 120, Y: 80}, To: enginescene.Point{X: 192, Y: 112}, Stroke: "#12345678"},
		}}
	wantWarnings := []editor.SceneWarning{
		{Code: "UNKNOWN_KEY", Message: `node "b" has 1 preserved unknown field(s)`, End: uint32(len(source))},
		{Code: "UNKNOWN_KEY", Message: "preserved 1 unknown top-level field(s)", End: uint32(len(source))},
		{Code: "IMAGE_PLACEHOLDER", Message: "节点图片引用已保留；此只读预览不加载或显示图片，完整引用请在源码中查看", End: uint32(len(source))},
	}
	if !reflect.DeepEqual(result.Scene, want) || !reflect.DeepEqual(result.Warnings, wantWarnings) {
		t.Fatalf("independent real scene/warnings: scene=%+v warnings=%+v", result.Scene, result.Warnings)
	}
	engine := result.Engine
	result = editor.SceneResult{} // do not retain a second delivered scene in test.
	a.modalDepth--
	tester.Frame() // production App.View drains the real bounded job slot.
	if a.previewJob != nil || len(queue) != 0 {
		t.Fatal("preview delivery used queued callback or retained job")
	}
	job.mu.Lock()
	retained := len(job.result.Scene.Nodes) != 0 || len(job.result.Warnings) != 0 || job.complete != nil
	job.mu.Unlock()
	if retained || job.task.source.Data != nil {
		t.Fatal("delivered job retained payload/callback/snapshot")
	}
	warnings := tab.View.Warnings()
	if len(warnings) != 3 || warnings[2] != wantWarnings[2].Message || !tester.HasText("此视图只读；节点属性、标签与图片引用请在源码中查看。") {
		t.Fatal("actual scene limitations/warnings missing", warnings)
	}
	if !a.activateOutlineTarget(tab, "b", false, "大纲定位") {
		t.Fatal("visible scene outline target", a.Message)
	}
	if a.activateOutlineTarget(tab, "hidden", false, "大纲定位") || !strings.Contains(a.Message, editor.ErrOutlineReadOnlyHidden.Error()) {
		t.Fatal("hidden readonly target not visibly refused", a.Message)
	}
	tester.Frame()
	assertReadonlySceneBounds(t, tester, a.Message)
	if err := tester.Click("适应画布"); err != nil {
		t.Fatal(err)
	}
	tester.Move(12, 470)
	tester.Key(0, ui.KeyEscape)
	images := []previewCapture{savePreviewBundleCapture(t, captureDir, "preview-readonly-light-760x480.png", tester)}
	a.Dark = true
	tester.Frame()
	assertReadonlySceneBounds(t, tester, a.Message)
	images = append(images, savePreviewBundleCapture(t, captureDir, "preview-readonly-dark-760x480.png", tester))
	if observed.calls.Load() != 1 || a.previewJob != nil {
		t.Fatal("theme/outline started another worker")
	}
	if err := tester.Click("查看只读文档源码"); err != nil || tab.Mode != "source" {
		t.Fatal("source action unreachable", err, tab.Mode)
	}
	tester.Frame()
	images = append(images, savePreviewBundleCapture(t, captureDir, "preview-readonly-source-760x480.png", tester))
	if tab.Doc.Buffer.String() != source || tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.Selection() != selection ||
		tab.Doc.Buffer.History() != history || tab.Doc.Dirty() != dirty || !tab.Doc.SameBaseline(baseline) || !tab.Doc.ReadOnly || !tab.Editor.ReadOnly ||
		tab.Doc.Encoding != document.UTF8 || !tab.Doc.BOM || !bytes.Equal(mustReadFile(t, path), originalBytes) {
		t.Fatal("real scene/navigation/theme/source view changed exact readonly document")
	}
	if err := a.closeEngineExport(); err != nil {
		t.Fatal(err)
	}
	if observed.calls.Load() != 1 || len(queue) != 0 {
		t.Fatal("close required dispatch or started another worker")
	}
	sourceDigest := sha256.Sum256(originalBytes)
	record := struct {
		Scope, Tester, EngineVersion, EngineSHA256, DiskSourceSHA256             string
		Nodes, Edges, Warnings, WorkerRequests                                   int
		ElapsedSeconds                                                           float64
		RuntimeJoined, ReadonlyDetected, ExactSourcePreserved, ResultSlotDropped bool
		Captures                                                                 []previewCapture
	}{"registered readonly app; real bundled preview; headless native images, no physical UI/PID observation", exe, engine.Version, engine.SHA256, hex.EncodeToString(sourceDigest[:]),
		3, 2, 3, int(observed.calls.Load()), time.Since(started).Seconds(), true, true, true, true, images}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(data))
}

func assertReadonlySceneBounds(t *testing.T, tester *ui.Tester, message string) {
	t.Helper()
	if message == "" {
		message = "本地文件 · 原生编辑"
	}
	status, ok := tester.Find(message)
	if !ok || status.Y < 430 || status.H <= 0 || status.Y+status.H > 480 {
		t.Fatal("actual status missing or outside status row", message, status)
	}
	statusTop := status.Y + status.H/2 - 29/2.0
	canvas, ok := tester.AccessByRole(ui.RoleGroup, "只读思维导图画布")
	if !ok || canvas.Offscreen || canvas.Bounds.W < 100 || canvas.Bounds.H < 100 || canvas.Bounds.Y+canvas.Bounds.H > statusTop+0.1 {
		t.Fatal("readonly canvas unusable above actual status", canvas, statusTop)
	}
	for _, name := range []string{"查看只读文档源码", "适应画布"} {
		control, ok := tester.AccessByRole(ui.RoleButton, name)
		if !ok || control.Offscreen || !control.Focusable || control.Disabled || control.Bounds.W < 28 || control.Bounds.H < 28 || control.Bounds.Y+control.Bounds.H > statusTop {
			t.Fatal("scene action not reachable", name, control, statusTop)
		}
	}
}

type previewCapture struct {
	Path, SHA256 string
	Bytes        int
}

func savePreviewBundleCapture(t *testing.T, directory, name string, tester *ui.Tester) previewCapture {
	t.Helper()
	if err := os.MkdirAll(directory, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, name)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	encodeErr, closeErr := png.Encode(file, tester.Image()), file.Close()
	if encodeErr != nil || closeErr != nil {
		t.Fatal("capture", encodeErr, closeErr)
	}
	data := mustReadFile(t, path)
	digest := sha256.Sum256(data)
	return previewCapture{Path: path, SHA256: hex.EncodeToString(digest[:]), Bytes: len(data)}
}
