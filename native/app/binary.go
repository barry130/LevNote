package app

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/plugins"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

func (t *Tab) isBinary() bool { return t != nil && (t.Binary != nil || t.BinaryPath != "") }
func (t *Tab) path() string {
	if t == nil {
		return ""
	}
	if t.Binary != nil {
		return t.Binary.Path()
	}
	if t.BinaryPath != "" {
		return t.BinaryPath
	}
	if t.Doc != nil {
		return t.Doc.Path
	}
	return ""
}
func (t *Tab) binarySize() int64 {
	if t.Binary != nil {
		return t.Binary.Size()
	}
	return t.BinarySize
}
func (t *Tab) textCapable() bool { return t != nil && t.Doc != nil && !t.isBinary() }

// This eight-byte regular-file probe is only a route hint. OpenBinary supplies
// stable bytes and the neutral decoder independently admits their real format.
func probeImage(path string) (bool, document.BinaryKind, string) {
	ext := strings.ToLower(filepath.Ext(path))
	kind := document.BinaryKindUnknown
	switch ext {
	case ".png":
		kind = document.BinaryKindPNG
	case ".jpg", ".jpeg":
		kind = document.BinaryKindJPEG
	}
	candidate := kind != document.BinaryKindUnknown
	f, err := os.Open(path)
	if err != nil {
		return candidate, kind, ""
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return candidate, kind, ""
	}
	var signature [8]byte
	n, _ := io.ReadFull(f, signature[:])
	actual := document.BinaryKindUnknown
	if n == 8 && bytes.Equal(signature[:], []byte{137, 80, 78, 71, 13, 10, 26, 10}) {
		actual = document.BinaryKindPNG
	}
	if n >= 3 && signature[0] == 0xff && signature[1] == 0xd8 && signature[2] == 0xff {
		actual = document.BinaryKindJPEG
	}
	if !candidate && actual != document.BinaryKindUnknown {
		return true, actual, "按文件签名选择内置图片阅读器；扩展名未声明为图片。"
	}
	return candidate, kind, ""
}

type binaryJob struct {
	op           string
	task         *binaryTask
	tab          *Tab
	owner        *document.BinaryFile
	token        document.BinaryToken
	generation   uint64
	view         editor.View
	adapter      *editor.Adapter
	mode         string
	complete     func(editor.ImageResult, error)
	done         chan struct{}
	notice, name string
	err          error
	abandoned    atomic.Bool
	reopen       *reopenReceipt
}

func (a *App) ensureBinaryLane() *binaryLane {
	if a.binaryLane == nil {
		a.binaryLane = newBinaryLane(nil)
	}
	return a.binaryLane
}

func (a *App) binaryNotify() func() {
	if a.Window != nil {
		return a.Window.Invalidate
	}
	return nil
}

func (a *App) imageEnabled() bool {
	if a.pluginManager == nil {
		return false
	}
	manifest, err := a.pluginManager.Manifest("image")
	_, compiled := a.adapterByID("image")
	return err == nil && manifest.Enabled && manifest.Active && compiled
}

func (a *App) configureBinary(t *Tab) {
	t.Adapter = nil
	t.MatchedPlugin = plugins.Manifest{}
	t.FormatWarning = ""
	t.Editor.Highlight = nil
	t.OutlineFactory = nil
	if t.Binary == nil {
		t.Mode = "source"
		t.FormatWarning = "图片超过 16 MiB 读取上限；仅显示前 64 KiB 字节，不能通过此图片路径另存副本。"
		return
	}
	if !a.imageEnabled() {
		t.Mode = "source"
		t.FormatWarning = "内置图片阅读器未启用；保留二进制权限并显示只读字节。"
		return
	}
	adapter, _ := a.adapterByID("image")
	t.Adapter = &adapter
	t.MatchedPlugin, _ = a.pluginManager.Manifest("image")
	if t.Mode != "source" {
		t.Mode = "preview"
	}
}

func (a *App) appendBinary(out binaryOutput, name, notice string) *Tab {
	a.nextID++
	input := editor.New(text.New(""))
	input.ReadOnly, input.FontSize, input.LineNumbers = true, a.FontSize, false
	input.Label = "只读原始字节"
	input.OnError = a.report
	t := &Tab{ID: a.nextID, Name: name, Binary: out.file, Editor: input, Mode: "preview", BinaryNotice: notice}
	if out.file != nil {
		t.BinaryKind = out.file.Kind()
	}
	if out.inspection != nil {
		t.BinaryPath, t.BinarySize = out.inspection.Path, out.inspection.Size
		input.Buffer = text.New(hex.Dump(out.inspection.Bytes))
	}
	a.configureTab(t)
	a.Tabs = append(a.Tabs, t)
	a.invalidateDiskChecks()
	return t
}

func (a *App) attachBinarySnapshot(t *Tab, snapshot *document.BinarySnapshot) {
	if snapshot == nil || t == nil || t.Binary == nil || !t.Binary.SameBaseline(snapshot.Token()) {
		return
	}
	a.binarySnapshotTab, a.binarySnapshot = t, snapshot
	prefix := make([]byte, min(snapshot.Size(), 64<<10))
	_, _ = snapshot.ReaderAt().ReadAt(prefix, 0)
	t.Editor.Buffer = text.New(hex.Dump(prefix))
}

func (a *App) heldBinary(t *Tab) *document.BinarySnapshot {
	if a.binarySnapshotTab == t && t.Binary != nil && a.binarySnapshot != nil && t.Binary.SameBaseline(a.binarySnapshot.Token()) {
		return a.binarySnapshot
	}
	return nil
}

func (a *App) deactivateBinary(t *Tab) {
	if !t.isBinary() {
		return
	}
	t.BinaryGeneration++
	a.cancelBinaryPreview(t)
	if a.binarySnapshotTab == t {
		a.binarySnapshotTab, a.binarySnapshot = nil, nil
	}
	t.ImageFormat, t.ImageWidth, t.ImageHeight = "", 0, 0
	if t.Binary != nil {
		t.Editor.Buffer = text.New("")
	}
}

func (a *App) cancelBinaryPreview(t *Tab) {
	j := a.binaryPreview
	if j == nil || t != nil && j.tab != t {
		return
	}
	a.binaryPreview = nil
	j.complete, j.tab, j.owner, j.view, j.adapter = nil, nil, nil, nil, nil
	a.binaryLane.cancelTask(j.task, true)
}

func (a *App) beginImage(t *Tab, owner editor.View, request editor.ImageRequest, callback func(editor.ImageResult, error)) (func(), error) {
	if a.shuttingDown.Load() || t == nil || owner == nil || t != a.Current() || !a.tabIsOpen(t) || t.Binary == nil || t.Mode != "preview" ||
		t.View != owner || callback == nil || !a.imageEnabled() {
		return nil, errors.New("图片视图已变化或插件不可用")
	}
	a.cancelBinaryPreview(nil)
	j := &binaryJob{op: "preview", tab: t, owner: t.Binary, token: t.Binary.Token(), generation: t.BinaryGeneration,
		view: owner, adapter: t.Adapter, mode: t.Mode, complete: callback}
	j.task = newBinaryTask(binaryRequest{op: "preview", token: j.token, kind: t.BinaryKind, snapshot: a.heldBinary(t), decode: a.binaryDecode}, a.binaryNotify())
	a.binaryPreview = j
	if err := a.ensureBinaryLane().submit(j.task, false); err != nil {
		a.binaryPreview = nil
		return nil, err
	}
	return func() {
		if a.binaryPreview == j {
			a.cancelBinaryPreview(t)
		}
	}, nil
}

func (a *App) requestBinaryBytes(t *Tab) {
	if t == nil || t.Binary == nil || a.Current() != t || a.heldBinary(t) != nil || a.binaryPreview != nil {
		return
	}
	j := &binaryJob{op: "bytes", tab: t, owner: t.Binary, token: t.Binary.Token(), generation: t.BinaryGeneration, mode: t.Mode}
	j.task = newBinaryTask(binaryRequest{op: "bytes", token: j.token}, a.binaryNotify())
	a.binaryPreview = j
	if err := a.ensureBinaryLane().submit(j.task, false); err != nil {
		a.binaryPreview = nil
		a.report(err)
	}
}

func (a *App) validBinaryJob(j *binaryJob) bool {
	return j != nil && j.tab != nil && a.Current() == j.tab && a.tabIsOpen(j.tab) && j.tab.Binary == j.owner &&
		j.owner != nil && j.owner.SameBaseline(j.token) && j.tab.BinaryGeneration == j.generation
}

func (a *App) deliverBinary() {
	if a.modalDepth > 0 || a.shuttingDown.Load() {
		return
	}
	if j := a.binaryJob; j != nil {
		if j.abandoned.Load() {
			a.discardAbandonedBinary(j)
		} else if out, ready := j.task.take(); ready {
			a.completeBinaryExplicit(j, out)
		}
	}
	j := a.binaryPreview
	if j == nil {
		return
	}
	if !a.validBinaryJob(j) || j.tab.Mode != j.mode || j.op == "preview" &&
		(j.tab.View != j.view || j.tab.Adapter != j.adapter || !a.imageEnabled()) {
		a.cancelBinaryPreview(nil)
		return
	}
	out, ready := j.task.take()
	if !ready {
		return
	}
	a.binaryPreview = nil
	defer a.binaryLane.release(j.task, false)
	t, callback := j.tab, j.complete
	j.tab, j.owner, j.view, j.adapter, j.complete = nil, nil, nil, nil, nil
	if out.snapshot != nil {
		a.attachBinarySnapshot(t, out.snapshot)
	}
	if out.err != nil {
		if errors.Is(out.err, document.ErrConflict) || errors.Is(out.err, os.ErrNotExist) {
			t.Conflict = true
		}
		t.Warning = "图片未更新：" + out.err.Error() + "；离开标签会释放已载入内容，返回时必须验证原文件。"
		a.report(errors.New(t.Warning))
	} else {
		if j.op == "preview" {
			t.Warning = ""
		}
		if j.op == "preview" {
			t.ImageFormat, t.ImageWidth, t.ImageHeight = out.image.Format, out.image.Width, out.image.Height
		}
	}
	if callback != nil {
		callback(out.image, out.err)
	}
	// Failed byte reacquire waits for explicit reload/selection; no idle retries.
	if j.op == "bytes" && out.err != nil {
		t.Editor.Buffer = text.New("无法验证原始字节；请明确重新载入。")
	}
}

func (a *App) binaryDocumentView(c *ui.Context, t *Tab) {
	ui.Row(c).Height(38).Shrink(0).Padding(5, 12).Gap(6).Children(func() {
		ui.Text(c, "图片 · 只读").Grow(1).FontSize(12)
		if ui.Button(c, "图片").Disabled(t.Adapter == nil).Height(28).Clicked() {
			a.setMode(t, "preview")
		}
		if ui.Button(c, "字节视图").Height(28).Clicked() {
			a.setMode(t, "source")
		}
		if ui.Button(c, "另存副本").Disabled(t.Binary == nil || t.Saving).Height(28).Clicked() {
			a.Save(t, true)
		}
		if ui.Button(c, "重新载入").Disabled(t.Binary == nil).Height(28).Clicked() {
			a.Reload()
		}
	})
	if t.BinaryNotice != "" {
		ui.Text(c, t.BinaryNotice).FontSize(11).Padding(5, 12).TextColor(c.Theme().TextMuted)
	}
	if t.Mode == "preview" && t.Adapter != nil {
		if t.View == nil {
			t.View = t.Adapter.Create(a.editorHost(t))
		}
		t.View.Draw(c.Key(t.ID))
		return
	}
	if t.Binary != nil && t.Editor.Buffer.LenBytes() == 0 {
		a.requestBinaryBytes(t)
	}
	t.Editor.View(c.Key(t.ID))
}

func (a *App) binaryStatus(c *ui.Context, t *Tab) {
	label := strings.ToUpper(string(t.BinaryKind))
	if label == "" {
		label = "二进制"
	}
	if t.ImageWidth > 0 {
		label = fmt.Sprintf("%s · %d × %d", t.ImageFormat, t.ImageWidth, t.ImageHeight)
	}
	ui.Text(c, fmt.Sprintf("%s · %d 字节 · 只读", label, t.binarySize())).FontSize(11).TextColor(c.Theme().TextMuted)
}
