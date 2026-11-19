package app

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/session"
	"levnote.local/native/modules"
	"levnote.local/native/ui/editor"
)

func imageFixture(t *testing.T, path, format string) []byte {
	t.Helper()
	pixels := image.NewRGBA(image.Rect(0, 0, 40, 24))
	for y := 0; y < 24; y++ {
		for x := 0; x < 40; x++ {
			pixels.SetRGBA(x, y, color.RGBA{uint8(x * 5), uint8(y * 9), 140, 255})
		}
	}
	var data bytes.Buffer
	var err error
	if format == "PNG" {
		err = png.Encode(&data, pixels)
	} else {
		err = jpeg.Encode(&data, pixels, &jpeg.Options{Quality: 95})
	}
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, data.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}

func waitBinaryTask(t *testing.T, task *binaryTask) {
	t.Helper()
	select {
	case <-task.done:
	case <-time.After(5 * time.Second):
		t.Fatal("binary task failed to finish")
	}
}

func decodeCurrentImage(t *testing.T, a *App) editor.ImageResult {
	t.Helper()
	tab := a.Current()
	owner := &failingView{}
	tab.View = owner
	var got editor.ImageResult
	var decodeErr error
	called := false
	_, err := a.beginImage(tab, owner, editor.ImageRequest{}, func(result editor.ImageResult, err error) { got, decodeErr, called = result, err, true })
	if err != nil {
		t.Fatal(err)
	}
	j := a.binaryPreview
	waitBinaryTask(t, j.task)
	a.deliverBinary()
	if !called || decodeErr != nil {
		t.Fatal("decode callback", called, decodeErr)
	}
	return got
}

func TestNativeImageOpenMixedAuthorityAndCapabilities(t *testing.T) {
	dir := t.TempDir()
	pngPath, jpegPath := filepath.Join(dir, "first.png"), filepath.Join(dir, "photo.jpg")
	imageFixture(t, pngPath, "PNG")
	imageFixture(t, jpegPath, "JPEG")
	textPath := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(textPath, []byte("text file"), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(dir, "config"))
	a.OpenMany([]string{filepath.Join(dir, "missing.png"), pngPath, textPath, jpegPath})
	if len(a.Tabs) != 3 || !a.Tabs[0].isBinary() || a.Tabs[0].Doc != nil || a.Tabs[1].Doc == nil || a.Current() != a.Tabs[2] {
		t.Fatal("mixed open order/authority", len(a.Tabs))
	}
	if a.binarySnapshotTab != a.Current() || a.Tabs[0].Editor.Buffer.LenBytes() != 0 {
		t.Fatal("inactive image retained payload")
	}
	if result := decodeCurrentImage(t, a); result.Format != "JPEG" || result.Width != 40 || result.Height != 24 || result.Pixels == nil {
		t.Fatal(result)
	}
	if a.Save(a.Current(), false) || a.openFind() || a.openGoToLine() || a.setMode(a.Current(), "split") {
		t.Fatal("image exposed text capabilities")
	}
	a.find(true)
	a.replace(true)
	a.exportCurrent()
	a.setOutlineVisible(true)
	a.applyEditorPreferences()
	a.Open(pngPath)
	if len(a.Tabs) != 3 || a.Current() != a.Tabs[0] || a.binarySnapshot != nil {
		t.Fatal("duplicate activation or inactive payload")
	}
	if result := decodeCurrentImage(t, a); result.Format != "PNG" || result.Width != 40 {
		t.Fatal(result)
	}
	if !a.CloseTab(a.Active) || len(a.Tabs) != 2 {
		t.Fatal("clean binary close prompted or failed")
	}
}

func TestNativeImageSignatureDisabledCorruptMismatchAndOversize(t *testing.T) {
	dir := t.TempDir()
	a := newTestApp(t, filepath.Join(dir, "config"))
	signature := filepath.Join(dir, "renamed.dat")
	imageFixture(t, signature, "PNG")
	a.Open(signature)
	if a.Current().Doc != nil || a.Current().Binary == nil || !strings.Contains(a.Current().BinaryNotice, "签名") {
		t.Fatal("signature route fell through text")
	}
	decodeCurrentImage(t, a)
	a.setPluginEnabled("image", false)
	tab := a.Current()
	if tab.Adapter != nil || tab.Mode != "source" || tab.Binary == nil || a.binarySnapshot == nil || tab.View != nil {
		t.Fatal("disabled fallback lost authority or retained presentation")
	}
	a.requestBinaryBytes(tab)
	if a.binaryPreview != nil {
		waitBinaryTask(t, a.binaryPreview.task)
		a.deliverBinary()
	}
	if tab.Editor.Buffer.LenBytes() == 0 || !strings.Contains(tab.FormatWarning, "未启用") {
		t.Fatal("byte fallback absent")
	}
	a.setPluginEnabled("image", true)
	a.setMode(tab, "preview")
	decodeCurrentImage(t, a)
	for _, item := range []struct{ name, format string }{{"wrong.jpg", "PNG"}, {"broken.png", ""}} {
		path := filepath.Join(dir, item.name)
		if item.format != "" {
			imageFixture(t, path, item.format)
		} else if err := os.WriteFile(path, []byte("not an image"), 0600); err != nil {
			t.Fatal(err)
		}
		a.Open(path)
		tab = a.Current()
		owner := &failingView{}
		tab.View = owner
		var result editor.ImageResult
		var decodeErr error
		if _, err := a.beginImage(tab, owner, editor.ImageRequest{}, func(r editor.ImageResult, e error) { result, decodeErr = r, e }); err != nil {
			t.Fatal(err)
		}
		waitBinaryTask(t, a.binaryPreview.task)
		a.deliverBinary()
		if decodeErr == nil || result.Pixels != nil || a.heldBinary(tab) == nil || tab.Doc != nil {
			t.Fatal("bad image silently decoded or lost bytes", decodeErr)
		}
		if !a.setMode(tab, "source") {
			t.Fatal("unconditional fallback missing")
		}
	}
	large := filepath.Join(dir, "large.png")
	if err := os.WriteFile(large, []byte("oversize"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(large, document.MaxBinaryBytes+1); err != nil {
		t.Fatal(err)
	}
	a.Open(large)
	if tab = a.Current(); tab.Binary != nil || tab.Doc != nil || tab.binarySize() != document.MaxBinaryBytes+1 || !strings.Contains(tab.FormatWarning, "16 MiB") || a.Save(tab, true) {
		t.Fatal("oversize path gained full/text authority")
	}
}

func TestNativeImageCopiesHeldBytesAndReloadsExplicitly(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.png")
	original := imageFixture(t, source, "PNG")
	a := newTestApp(t, filepath.Join(dir, "config"))
	a.Open(source)
	tab := a.Current()
	initial := tab.Binary.Token()
	a.saveDialog = func(*Tab) (string, error) { return source, nil }
	if !a.Save(tab, true) || !tab.Binary.SameBaseline(initial) || !strings.Contains(a.Message, "没有重写") {
		t.Fatal(a.Message)
	}
	if err := os.Remove(source); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(dir, "copy.png")
	a.saveDialog = func(*Tab) (string, error) { return destination, nil }
	if !a.Save(tab, true) || tab.path() != destination || a.heldBinary(tab) == nil || tab.Binary.SameBaseline(initial) {
		t.Fatal(a.Message)
	}
	if got, err := os.ReadFile(destination); err != nil || !bytes.Equal(got, original) {
		t.Fatal(err, "copy differs")
	}
	other := filepath.Join(dir, "exists.png")
	if err := os.WriteFile(other, []byte("winner"), 0600); err != nil {
		t.Fatal(err)
	}
	a.saveDialog = func(*Tab) (string, error) { return other, nil }
	a.Save(tab, true)
	if got, _ := os.ReadFile(other); string(got) != "winner" || tab.path() != destination {
		t.Fatal("overwrote existing destination")
	}
	old := tab.Binary
	imageFixture(t, destination, "JPEG")
	a.Reload()
	if tab.Binary == old || a.heldBinary(tab) == nil || tab.Conflict {
		t.Fatal("explicit reload didn't transfer replacement authority")
	}
	if err := os.Remove(destination); err != nil {
		t.Fatal(err)
	}
	old = tab.Binary
	held := a.heldBinary(tab)
	a.Reload()
	if tab.Binary != old || a.heldBinary(tab) != held {
		t.Fatal("failed reload lost accepted content")
	}
}

func TestNativeImageModalStaleAndFailedFlushKeepAuthority(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.png")
	imageFixture(t, source, "PNG")
	a := newTestApp(t, filepath.Join(dir, "config"))
	a.NewDocument("markdown")
	textTab := a.Current()
	view := &failingView{flushErr: errors.New("pending draft")}
	textTab.View = view
	a.Open(source)
	if len(a.Tabs) != 1 || a.Current() != textTab || view.releases != 0 || a.binarySnapshot != nil {
		t.Fatal("failed text Flush destroyed view")
	}
	view.flushErr = nil
	a.Open(source)
	tab := a.Current()
	destination := filepath.Join(dir, "stale.png")
	a.saveDialog = func(*Tab) (string, error) { a.Select(0); return destination, nil }
	if a.Save(tab, true) {
		t.Fatal("dialog-stale copy accepted")
	}
	if _, err := os.Stat(destination); !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	a.Select(1)
	owner := &failingView{}
	tab.View = owner
	called := false
	if _, err := a.beginImage(tab, owner, editor.ImageRequest{}, func(editor.ImageResult, error) { called = true }); err != nil {
		t.Fatal(err)
	}
	j := a.binaryPreview
	waitBinaryTask(t, j.task)
	finishModal := a.deferExportDelivery()
	a.deliverBinary()
	if a.CanClose() {
		t.Fatal("window closed inside native dialog")
	}
	if called {
		t.Fatal("modal delivered pixels")
	}
	a.Select(0)
	finishModal()
	if called || j.complete != nil || j.tab != nil || a.binarySnapshot != nil {
		t.Fatal("stale callback retained owner/payload")
	}
	if _, ready := j.task.take(); ready {
		t.Fatal("abandoned result slot retained data")
	}
}

func TestNativeImageSessionOrderAndPathOnlyRestore(t *testing.T) {
	dir := t.TempDir()
	first, second := filepath.Join(dir, "a.png"), filepath.Join(dir, "b.jpg")
	imageFixture(t, first, "PNG")
	imageFixture(t, second, "JPEG")
	config := filepath.Join(dir, "config")
	a := newTestApp(t, config)
	a.Open(first)
	a.NewDocument("text")
	a.Current().Editor.ReplaceText(nil, "unsaved recovery")
	a.Open(second)
	a.Select(0)
	a.Persist()
	raw, err := os.ReadFile(filepath.Join(config, "session.json"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("IHDR")) || bytes.Contains(raw, []byte("00000000")) {
		t.Fatal("binary session contains bytes")
	}
	if err := os.Remove(second); err != nil {
		t.Fatal(err)
	}
	b := newTestApp(t, config)
	b.Restore()
	if len(b.Tabs) != 2 || b.Active != 0 || b.Tabs[0].Binary == nil || b.Tabs[0].Doc != nil || b.Tabs[1].Doc.Buffer.String() != "unsaved recovery" || b.binarySnapshot != nil {
		t.Fatal("mixed restore authority/order/retention")
	}
	decodeCurrentImage(t, b)
	b.Persist()
	records, issues := session.Load(filepath.Join(config, "session.json"))
	if len(issues) > 0 || len(records.Tabs) != 2 {
		t.Fatal(issues)
	}
	if _, err := session.RestoreBinary(records.Tabs[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := session.RestoreTab(records.Tabs[1]); err != nil {
		t.Fatal(err)
	}
}

func TestNativeImageLaneLatestPendingAndShutdownJoin(t *testing.T) {
	dir := t.TempDir()
	paths := []string{filepath.Join(dir, "a.png"), filepath.Join(dir, "b.png"), filepath.Join(dir, "c.png")}
	a := newTestApp(t, filepath.Join(dir, "config"))
	for _, path := range paths {
		imageFixture(t, path, "PNG")
		a.Open(path)
	}
	started := make(chan struct{}, 3)
	release := make(chan struct{}, 3)
	var active, peak atomic.Int32
	a.binaryDecode = func(ctx context.Context, r io.ReaderAt, n int64) (editor.ImageResult, error) {
		count := active.Add(1)
		defer active.Add(-1)
		for p := peak.Load(); count > p; p = peak.Load() {
			if peak.CompareAndSwap(p, count) {
				break
			}
		}
		started <- struct{}{}
		<-release
		return modules.DecodeImage(ctx, r, n)
	}
	begin := func(index int) *binaryTask {
		a.Select(index)
		tab := a.Current()
		owner := &failingView{}
		tab.View = owner
		if _, err := a.beginImage(tab, owner, editor.ImageRequest{}, func(editor.ImageResult, error) { t.Error("stale image delivered") }); err != nil {
			t.Fatal(err)
		}
		return a.binaryPreview.task
	}
	first := begin(0)
	<-started
	second := begin(1)
	third := begin(2)
	waitBinaryTask(t, second)
	if _, ready := second.take(); ready {
		t.Fatal("superseded pending payload retained")
	}
	release <- struct{}{}
	waitBinaryTask(t, first)
	<-started
	if peak.Load() != 1 {
		t.Fatal("parallel full image task")
	}
	if !a.CloseTab(a.Active) {
		t.Fatal("close active image")
	}
	release <- struct{}{}
	waitBinaryTask(t, third)
	if a.binaryPreview != nil || a.binarySnapshot != nil {
		t.Fatal("close retained image request")
	}
	last := begin(0)
	<-started
	finished := make(chan struct{})
	go func() { a.Shutdown(); close(finished) }()
	select {
	case <-last.ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("shutdown did not cancel")
	}
	select {
	case <-finished:
		t.Fatal("shutdown did not join codec")
	default:
	}
	release <- struct{}{}
	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("shutdown did not finish")
	}
	if active.Load() != 0 || a.binaryPreview != nil || a.binarySnapshot != nil {
		t.Fatal("shutdown retained task/payload")
	}
}

func TestNativeImageCopySurvivesNavigationAndClosedTabOutcome(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.png")
	original := imageFixture(t, source, "PNG")
	a := newTestApp(t, filepath.Join(dir, "config"))
	a.Open(source)
	entered, release := make(chan struct{}), make(chan struct{})
	a.binaryLane.close()
	a.binaryLane = newBinaryLane(func(ctx context.Context, r binaryRequest) binaryOutput {
		out := executeBinary(ctx, r)
		if r.op == "copy" {
			close(entered)
			<-release
		}
		return out
	})
	// Controlled UI delivery: publication completes but its outcome is held.
	a.uiDispatch = func(fn func()) { fn() }
	tab := a.Current()
	destination := filepath.Join(dir, "copy.png")
	a.saveDialog = func(*Tab) (string, error) { return destination, nil }
	if !a.Save(tab, true) {
		t.Fatal(a.Message)
	}
	j := a.binaryJob
	<-entered
	if !a.CloseTab(0) {
		t.Fatal("close failed")
	}
	if j.tab != nil || j.owner != nil {
		t.Fatal("closed tab retained by explicit job")
	}
	close(release)
	waitBinaryTask(t, j.task)
	a.deliverBinary()
	if got, _ := os.ReadFile(destination); !bytes.Equal(got, original) || !strings.Contains(a.Message, "已完成") || a.binaryJob != nil {
		t.Fatal(a.Message)
	}
}

func TestNativeImageExternalObservationAndReacquireFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "image.png")
	imageFixture(t, path, "PNG")
	a := newTestApp(t, filepath.Join(dir, "config"))
	a.Open(path)
	tab := a.Current()
	held := a.heldBinary(tab)
	token := tab.Binary.Token()
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	task := diskCheckTask{generation: a.diskGeneration, targets: []diskCheckTarget{{tabID: tab.ID, path: path, binaryToken: token}}}
	if !a.applyDiskCheck(task, checkDiskTask(context.Background(), task)) || !tab.Conflict || a.heldBinary(tab) != held {
		t.Fatal("external deletion lost held data")
	}
	a.NewDocument("text")
	a.Select(0)
	a.setMode(tab, "source")
	a.requestBinaryBytes(tab)
	waitBinaryTask(t, a.binaryPreview.task)
	a.deliverBinary()
	if a.binarySnapshot != nil || !strings.Contains(tab.Warning, "必须验证") {
		t.Fatal(tab.Warning)
	}
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	tester.Frame()
	if a.binaryPreview != nil {
		t.Fatal("failed reacquire retried every frame")
	}
}

func TestNativeImageChooseOpenHeadlessAndDeliveryTimeout(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "image.png")
	imageFixture(t, path, "PNG")
	a := newTestApp(t, filepath.Join(dir, "config"))
	a.openDialog = func(mygo.OpenDialogOptions) ([]string, error) { return []string{path}, nil }
	a.ChooseOpen()
	if a.Current() == nil || a.Current().Binary == nil || a.modalDepth != 0 || a.binaryJob != nil {
		t.Fatal("headless dialog did not unwind")
	}
	a.CloseTab(0)
	queued := make(chan func(), 8)
	a.uiDispatch = func(fn func()) { queued <- fn }
	a.updateWaitTimeout = 40 * time.Millisecond
	done := a.openRequests([]openRequest{{path: path}})
	next := func() func() {
		select {
		case fn := <-queued:
			return fn
		case <-time.After(time.Second):
			t.Fatal("missing UI callback")
			return nil
		}
	}
	next()() // duplicate check
	next()() // explicit lane reservation
	j := a.binaryJob
	late := next() // deliberately withhold result delivery beyond its deadline
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("open queue did not time out")
	}
	if !j.abandoned.Load() {
		t.Fatal("timed-out job was not marked abandoned")
	}
	late()
	a.deliverBinary()
	if len(a.Tabs) != 0 || a.binaryJob != nil || a.binarySnapshot != nil {
		t.Fatal("abandoned result activated or held reservation")
	}
	if out, ready := j.task.take(); ready || out.snapshot != nil || out.image.Pixels != nil {
		t.Fatal("abandoned slot held payload")
	}
	a.uiDispatch = nil
	a.Open(path)
	if a.Current() == nil || a.Current().Binary == nil {
		t.Fatal("lane remained unusable after timeout", a.Message)
	}
}

func TestNativeImageWindowCloseSurfacesCopyOutcomeBeforeClosing(t *testing.T) {
	for _, committed := range []bool{false, true} {
		name := "before publication"
		if committed {
			name = "after publication"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			source := filepath.Join(dir, "source.png")
			imageFixture(t, source, "PNG")
			a := newTestApp(t, filepath.Join(dir, "config"))
			a.Open(source)
			started, release := make(chan struct{}), make(chan struct{})
			a.binaryLane.close()
			a.binaryLane = newBinaryLane(func(ctx context.Context, r binaryRequest) binaryOutput {
				if r.op != "copy" {
					return executeBinary(ctx, r)
				}
				var out binaryOutput
				if committed {
					out = executeBinary(ctx, r)
				}
				close(started)
				<-release
				if !committed {
					out = executeBinary(ctx, r)
				}
				return out
			})
			a.uiDispatch = func(fn func()) { fn() }
			destination := filepath.Join(dir, "copy.png")
			a.saveDialog = func(*Tab) (string, error) { return destination, nil }
			if !a.Save(a.Current(), true) {
				t.Fatal(a.Message)
			}
			j := a.binaryJob
			<-started
			if a.CanClose() || len(a.Tabs) != 1 || !strings.Contains(a.Message, "确认副本") {
				t.Fatal("window closed before copy outcome", a.Message)
			}
			close(release)
			waitBinaryTask(t, j.task)
			if committed {
				finishModal := a.deferExportDelivery()
				a.deliverBinary()
				if a.CanClose() || a.binaryJob != j || len(a.Tabs) != 1 || a.Current().path() != source {
					t.Fatal("modal close consumed pending copy outcome", a.Message)
				}
				j.task.mu.Lock()
				ready := j.task.ready
				j.task.mu.Unlock()
				if !ready {
					t.Fatal("modal close dropped ready copy slot")
				}
				finishModal()
				if a.binaryJob != nil || len(a.Tabs) != 1 || a.Current().path() != destination {
					t.Fatal("copy outcome missing after dialog closed", a.Message)
				}
			} else if a.CanClose() || a.binaryJob != nil || len(a.Tabs) != 1 {
				t.Fatal("completion was hidden by closing", a.Message)
			}
			if committed && !strings.Contains(a.Message, "已另存") || !committed && !strings.Contains(a.Message, "未写入") {
				t.Fatal(a.Message)
			}
			if !a.CanClose() || len(a.Tabs) != 0 {
				t.Fatal("second acknowledged close failed", a.Message)
			}
		})
	}
}

func TestNativeImageDeletedSourceKeepsBytesThroughModeAndPlugin(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "image.png")
	original := imageFixture(t, source, "PNG")
	a := newTestApp(t, filepath.Join(dir, "config"))
	a.Open(source)
	tab := a.Current()
	decodeCurrentImage(t, a)
	held := a.heldBinary(tab)
	if err := os.Remove(source); err != nil {
		t.Fatal(err)
	}
	if !a.setMode(tab, "source") || a.heldBinary(tab) != held || tab.Editor.Buffer.LenBytes() == 0 {
		t.Fatal("mode discarded accepted bytes")
	}
	a.setPluginEnabled("image", false)
	if a.heldBinary(tab) != held || tab.Adapter != nil || tab.Editor.Buffer.LenBytes() == 0 {
		t.Fatal("plugin fallback discarded accepted bytes")
	}
	a.SoftWrap, a.LineNumbers = true, true
	a.applyEditorPreferences()
	if tab.Editor.SoftWrap || tab.Editor.LineNumbers {
		t.Fatal("byte inspector inherited text preferences")
	}
	destination := filepath.Join(dir, "copy.custom")
	a.saveDialog = func(*Tab) (string, error) { return destination, nil }
	a.Save(tab, true)
	if tab.path() != destination || tab.Doc != nil || a.heldBinary(tab) == nil {
		t.Fatal("custom suffix copy lost binary authority", a.Message)
	}
	if got, _ := os.ReadFile(destination); !bytes.Equal(got, original) {
		t.Fatal("copy reencoded bytes")
	}
	a.setPluginEnabled("image", true)
	a.setMode(tab, "preview")
	decodeCurrentImage(t, a)
	a.NewDocument("text")
	if a.binarySnapshot != nil || tab.Editor.Buffer.LenBytes() != 0 {
		t.Fatal("real deactivation retained bytes")
	}
}

func TestNativeImageTransactionalReloadOverlapAndNewestPending(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.png")
	imageFixture(t, source, "PNG")
	a := newTestApp(t, filepath.Join(dir, "config"))
	a.Open(source)
	tab := a.Current()
	held := a.heldBinary(tab)
	started, release := make(chan struct{}), make(chan struct{})
	a.binaryLane.close()
	a.binaryLane = newBinaryLane(func(ctx context.Context, r binaryRequest) binaryOutput {
		out := executeBinary(ctx, r)
		if r.op == "reload" {
			close(started)
			<-release
		}
		return out
	})
	a.uiDispatch = func(fn func()) { fn() }
	a.Reload()
	j := a.binaryJob
	<-started
	if a.heldBinary(tab) != held {
		t.Fatal("transaction dropped accepted image before replacement ready")
	}
	owner := &failingView{}
	tab.View = owner
	called := false
	if _, err := a.beginImage(tab, owner, editor.ImageRequest{}, func(editor.ImageResult, error) { called = true }); err != nil {
		t.Fatal(err)
	}
	pending := a.binaryPreview.task
	select {
	case <-pending.done:
		t.Fatal("preview ran before reload outcome")
	default:
	}
	close(release)
	waitBinaryTask(t, j.task)
	j.task.mu.Lock()
	replacement := j.task.output.snapshot
	j.task.mu.Unlock()
	if replacement == nil || replacement == held || a.heldBinary(tab) != held {
		t.Fatal("expected bounded old/new transaction pair missing")
	}
	a.deliverBinary()
	waitBinaryTask(t, pending)
	if called || a.heldBinary(tab) != replacement || a.binaryPreview != nil {
		t.Fatal("reload did not invalidate pending view")
	}
}

func TestNativeImageCaptures(t *testing.T) {
	dir := os.Getenv("LEVNOTE_IMAGE_CAPTURE_DIR")
	if dir == "" {
		t.Skip("explicit synthetic image/byte captures")
	}
	root := t.TempDir()
	path := filepath.Join(root, "示例图片.png")
	imageFixture(t, path, "PNG")
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	a.Message = ""
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	if a.binaryPreview != nil {
		waitBinaryTask(t, a.binaryPreview.task)
	}
	tester.Frame()
	tester.Frame()
	if a.Current().ImageWidth != 40 || !tester.HasText("适合窗口") || !tester.HasText("另存副本") {
		t.Fatal(tester.Texts())
	}
	if os.Getenv("LEVNOTE_IMAGE_CAPTURE_REVISION") != "" {
		control, ok := tester.Find("查看只读字节")
		if !ok || control.X < 228 || control.Y < 0 || control.X+control.W > 760 || control.Y+control.H > 451 {
			t.Fatal("byte control clipped", control, ok)
		}
		captureAcceptanceTester(t, filepath.Join(dir, "image-light-760x480-toolbar-fixed.png"), tester)
		return
	}
	captureAcceptanceTester(t, filepath.Join(dir, "image-light-760x480.png"), tester)
	a.Dark = true
	a.setPluginEnabled("image", false)
	a.Message = ""
	tester.Frame()
	tester.Frame()
	if a.Current().Adapter != nil || a.heldBinary(a.Current()) == nil || !strings.Contains(strings.Join(tester.Texts(), " "), "只读字节") {
		t.Fatal(tester.Texts())
	}
	captureAcceptanceTester(t, filepath.Join(dir, "bytes-dark-760x480.png"), tester)
}

func TestNativeImageDuplicateRecheckedAtPublication(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "image.png")
	imageFixture(t, path, "PNG")
	a := newTestApp(t, filepath.Join(dir, "config"))
	a.NewDocument("text")
	textTab := a.Current()
	started, release := make(chan struct{}), make(chan struct{})
	a.binaryLane = newBinaryLane(func(ctx context.Context, r binaryRequest) binaryOutput {
		out := executeBinary(ctx, r)
		close(started)
		<-release
		return out
	})
	j := &binaryJob{op: "open", name: "image.png"}
	if err := a.startBinaryExplicit(j, binaryRequest{op: "open", path: path, kind: document.BinaryKindPNG}); err != nil {
		t.Fatal(err)
	}
	<-started
	file, snapshot, err := document.OpenBinary(context.Background(), path, document.BinaryKindPNG)
	if err != nil {
		t.Fatal(err)
	}
	existing := a.appendBinary(binaryOutput{file: file, snapshot: snapshot}, "image.png", "")
	failure := &failingView{flushErr: errors.New("pending rich input")}
	textTab.View = failure
	close(release)
	waitBinaryTask(t, j.task)
	a.deliverBinary()
	if len(a.Tabs) != 2 || a.Current() != textTab || existing.Binary != file || failure.releases != 0 || a.binarySnapshot != nil {
		t.Fatal("duplicate delivery ignored Flush or opened duplicate")
	}
	failure.flushErr = nil
	a.Open(path)
	if len(a.Tabs) != 2 || a.Current() != existing {
		t.Fatal("duplicate activation failed after Flush")
	}
}

func TestNativeImageLegacySessionImageRoutingPreservesDirtyDraft(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.png")
	imageFixture(t, path, "PNG")
	missing := filepath.Join(dir, "missing.png")
	draft := "dirty text with image suffix"
	config := filepath.Join(dir, "config")
	if err := os.MkdirAll(config, 0700); err != nil {
		t.Fatal(err)
	}
	if err := session.Save(filepath.Join(config, "session.json"), session.Snapshot{Active: 1, Tabs: []session.Tab{
		{Path: missing, Title: "missing.png"}, {Path: path, Title: "legacy.png"}, {Path: path, Title: "draft.png", Draft: &draft},
	}}); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, config)
	a.Restore()
	if len(a.Tabs) != 2 || a.Active != 0 || a.Tabs[0].Binary == nil || a.Tabs[0].Doc != nil || a.binarySnapshot != nil {
		t.Fatal("legacy clean image read as text or mapping lost")
	}
	textTab := a.Tabs[1]
	if textTab.Doc == nil || textTab.Binary != nil || textTab.Adapter != nil || textTab.Mode != "source" || textTab.Doc.Buffer.String() != draft || !textTab.Doc.IsNew || textTab.OriginalPath != path {
		t.Fatal("dirty image-suffix draft was discarded or interpreted as binary")
	}
}

func TestNativeImageCallbacksAndReentrantCopyAreBounded(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "source.png")
	imageFixture(t, path, "PNG")
	a := newTestApp(t, filepath.Join(dir, "config"))
	if _, err := a.beginImage(nil, nil, editor.ImageRequest{}, func(editor.ImageResult, error) {}); err == nil {
		t.Fatal("nil callback owner accepted")
	}
	a.Open(path)
	tab := a.Current()
	calls := 0
	destination := filepath.Join(dir, "copy.png")
	a.saveDialog = func(*Tab) (string, error) {
		calls++
		if a.Save(tab, true) {
			t.Fatal("nested modal copy retained another source authority")
		}
		return destination, nil
	}
	if !a.Save(tab, true) || calls != 1 || tab.path() != destination {
		t.Fatal("copy failed after bounded nested refusal", a.Message)
	}
}
