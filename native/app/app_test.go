package app

import (
	"errors"
	"fmt"
	"github.com/egoist/mygo/ui"
	"image/png"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"levnote.local/native/core/document"
	"levnote.local/native/core/search"
	"levnote.local/native/core/session"
)

func newTestApp(t *testing.T, configDir string) *App {
	t.Helper()
	a := New(configDir)
	t.Cleanup(a.Shutdown)
	return a
}

func clickTesterText(t *testing.T, tester *ui.Tester, text string) {
	t.Helper()
	rect, ok := tester.Find(text)
	if !ok {
		t.Fatalf("%q is not visible: %#v", text, tester.Texts())
	}
	tester.ClickAt(rect.X+rect.W/2, rect.Y+rect.H/2)
}

type failingView struct {
	flushErr error
	flushes  int
	releases int

	cacheLoaded bool
	pageIndex   int
	selectedID  string
	zoom        float32
	panX        float32
	warnings    []string
}

func (v *failingView) Draw(*ui.Context)              {}
func (v *failingView) Release()                      { v.releases++; v.cacheLoaded = false }
func (v *failingView) Warnings() []string            { return append([]string(nil), v.warnings...) }
func (v *failingView) Export(string) ([]byte, error) { return []byte("export"), nil }
func (v *failingView) Flush() error                  { v.flushes++; return v.flushErr }

type testDirEntry string

func (e testDirEntry) Name() string               { return string(e) }
func (e testDirEntry) IsDir() bool                { return false }
func (e testDirEntry) Type() fs.FileMode          { return 0 }
func (e testDirEntry) Info() (fs.FileInfo, error) { return nil, nil }

type partialDirReader struct{}

func (partialDirReader) ReadDir(int) ([]os.DirEntry, error) {
	return []os.DirEntry{testDirEntry("visible.txt")}, io.ErrUnexpectedEOF
}

func TestWorkspaceEditingAcrossTabs(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.NewDocument("markdown")
	first := a.Current()
	first.Mode = "source"
	tt := ui.NewTester(a.View, 1180, 780)
	if err := tt.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyEnd)
	tt.Type("一次修改")
	a.NewDocument("text")
	tt.Frame()
	a.Select(0)
	tt.Frame()
	first.Editor.Command("undo")
	if first.Doc.Buffer.String() != "# 新文档\n\n" {
		t.Fatalf("undo across tabs: %q", first.Doc.Buffer.String())
	}
	tt.Frame()
	tt.Advance(time.Second)
	if tt.FramePending() {
		t.Fatal("idle view should sleep")
	}
}

func TestEmptyDirectoryAndPartialReadErrors(t *testing.T) {
	entries, truncated, err := readDirectoryBounded(t.TempDir(), 8)
	if err != nil || truncated || len(entries) != 0 {
		t.Fatalf("empty directory = entries=%#v truncated=%v err=%v", entries, truncated, err)
	}

	entries, truncated, err = readDirectoryEntries(partialDirReader{}, 8)
	if len(entries) != 1 || entries[0].Name() != "visible.txt" || truncated {
		t.Fatalf("partial entries = %#v truncated=%v err=%v", entries, truncated, err)
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("partial read error was hidden: %v", err)
	}
}

func TestInstalledPluginControlsNativeSourceSyntax(t *testing.T) {
	root := t.TempDir()
	pluginRoot := filepath.Join(root, "plugins")
	if err := os.MkdirAll(pluginRoot, 0755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"schema":1,"id":"toml","name":"TOML","version":"1.0.0","apiVersion":1,"extensions":[".toml"],"lineComment":"#","blockStart":"/*","blockEnd":"*/","keywords":["true","false"]}`
	if err := os.WriteFile(filepath.Join(pluginRoot, "toml.json"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "settings.toml")
	if err := os.WriteFile(path, []byte("enabled = true\n# note\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, root)
	a.Open(path)
	current := a.Current()
	if current == nil || current.MatchedPlugin.ID != "toml" || current.Adapter != nil || current.Mode != "source" || current.Editor.Highlight == nil {
		t.Fatalf("plugin did not select native source syntax: %#v", current)
	}
	spans := current.Editor.Highlight("enabled = true")
	text := ""
	for _, span := range spans {
		text += span.Text
	}
	if text != "enabled = true" || len(spans) < 2 {
		t.Fatalf("declarative highlight spans = %#v", spans)
	}
	a.setPluginEnabled("toml", false)
	if current.MatchedPlugin.ID != "code" || current.Editor.Highlight != nil {
		t.Fatalf("disabled plugin did not fall back to source view: %#v", current.MatchedPlugin)
	}
}

func TestSyntaxPalettesMeetContrastAndFollowTheme(t *testing.T) {
	luminance := func(color ui.Color) float64 {
		channel := func(value uint8) float64 {
			component := float64(value) / 255
			if component <= 0.04045 {
				return component / 12.92
			}
			return math.Pow((component+0.055)/1.055, 2.4)
		}
		return 0.2126*channel(color.R) + 0.7152*channel(color.G) + 0.0722*channel(color.B)
	}
	contrast := func(a, b ui.Color) float64 {
		left, right := luminance(a), luminance(b)
		if left < right {
			left, right = right, left
		}
		return (left + 0.05) / (right + 0.05)
	}
	for _, dark := range []bool{false, true} {
		theme := ui.LightTheme()
		if dark {
			theme = ui.DarkTheme()
		}
		colors := syntaxColors(dark)
		values := map[string]ui.Color{
			"keyword": colors.keyword, "string": colors.stringValue,
			"number": colors.number, "comment": colors.comment, "tag": colors.tag,
		}
		for kind, color := range values {
			if ratio := contrast(color, theme.Background); ratio < 4.5 {
				t.Errorf("%s syntax color in dark=%v has contrast %.2f", kind, dark, ratio)
			}
		}
	}
	light, dark := syntaxColors(false), syntaxColors(true)
	if light == dark {
		t.Fatal("source syntax palette does not follow the application theme")
	}
}

func TestBuiltinAdaptersAndUnknownExtensionFallback(t *testing.T) {
	root := t.TempDir()
	a := newTestApp(t, root)
	a.NewDocument("mindmap")
	if current := a.Current(); current == nil || current.Adapter == nil || current.Adapter.ID != "mindmap" || current.Mode != "preview" {
		t.Fatalf("mindmap adapter not wired: %#v", current)
	}
	path := filepath.Join(root, "unknown.levx")
	if err := os.WriteFile(path, []byte("visible source\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a.Open(path)
	current := a.Current()
	if current == nil || current.Adapter != nil || current.Mode != "source" || current.FormatWarning == "" {
		t.Fatalf("unknown extension must visibly fall back to source: %#v", current)
	}
	if current.Doc.Buffer.String() != "visible source\n" {
		t.Fatal("fallback changed file content")
	}
}

func TestOpenManyKeepsRequestOrderContinuesAndReusesTabs(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first.txt")
	second := filepath.Join(root, "second.txt")
	if err := os.WriteFile(first, []byte("first"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("second"), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.OpenMany([]string{filepath.Join(root, "missing.txt"), first, second})
	if len(a.Tabs) != 2 || a.Tabs[0].Name != "first.txt" || a.Tabs[1].Name != "second.txt" {
		t.Fatalf("ordered batch = %#v", a.Tabs)
	}
	a.OpenMany([]string{first})
	if len(a.Tabs) != 2 || a.Active != 0 {
		t.Fatalf("duplicate open created or selected the wrong tab: active=%d tabs=%d", a.Active, len(a.Tabs))
	}
}

func TestOpenManyWaitsForDelayedAsyncUIDispatch(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first.txt")
	second := filepath.Join(root, "second.txt")
	for path, value := range map[string]string{first: "first", second: "second"} {
		if err := os.WriteFile(path, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}

	a := newTestApp(t, filepath.Join(root, "config"))
	dispatched := make(chan func(), 16)
	a.uiDispatch = func(fn func()) { dispatched <- fn }
	done := a.openRequests([]openRequest{
		{path: first},
		{path: first},
		{path: filepath.Join(root, "missing.txt")},
		{path: second},
	})

	var firstLookup func()
	select {
	case firstLookup = <-dispatched:
	case <-time.After(2 * time.Second):
		t.Fatal("first UI dispatch did not arrive")
	}
	if len(a.Tabs) != 0 {
		t.Fatal("UI state changed before the delayed dispatch ran")
	}
	time.Sleep(20 * time.Millisecond)
	firstLookup()

	for {
		select {
		case fn := <-dispatched:
			time.Sleep(10 * time.Millisecond)
			fn()
		case <-done:
			if len(a.Tabs) != 2 || a.Tabs[0].Name != "first.txt" || a.Tabs[1].Name != "second.txt" {
				t.Fatalf("delayed ordered batch = %#v", a.Tabs)
			}
			return
		case <-time.After(2 * time.Second):
			t.Fatal("ordered open queue did not finish")
		}
	}
}

func TestWorkspaceRevealSurvivesOpeningAndReuse(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "search.txt")
	if err := os.WriteFile(path, []byte("zero\nneedle\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.OpenAt(path, 5, 11)
	current := a.Current()
	if current == nil || current.Doc.Buffer.Selection().Start() != 5 || current.Doc.Buffer.Selection().End() != 11 {
		t.Fatalf("initial async-style reveal was not applied: %#v", current)
	}
	a.Select(-1)
	a.NewDocument("text")
	a.OpenAt(path, 5, 11)
	if a.Active != 0 || a.Current() != current {
		t.Fatal("search reveal did not reuse the existing tab")
	}
}

func TestSearchRevealFlushesExistingRichView(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "note.md")
	if err := os.WriteFile(path, []byte("# saved\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	view := &failingView{flushErr: errors.New("draft conflict")}
	tab.View = view
	tab.Mode = "preview"
	if err := tab.Doc.Buffer.SetSelection(1, 3); err != nil {
		t.Fatal(err)
	}

	a.OpenAt(path, 0, 1)
	if tab.View != view || tab.Mode != "preview" {
		t.Fatal("failed reveal flush released or changed the rich view")
	}
	if selection := tab.Doc.Buffer.Selection(); selection.Start() != 1 || selection.End() != 3 {
		t.Fatalf("failed reveal flush changed selection: %#v", selection)
	}
}

func TestReadonlyPreviewAndByteInspectorAreDistinct(t *testing.T) {
	root := t.TempDir()
	a := newTestApp(t, filepath.Join(root, "config"))
	documentValue := document.New("# preview\n")
	documentValue.ReadOnly = true
	a.add(documentValue, "readonly.md", "")
	if current := a.Current(); current == nil || current.Adapter == nil || current.Mode != "preview" {
		t.Fatalf("readonly text did not configure its preview: %#v", current)
	}

	path := filepath.Join(root, "invalid.txt")
	if err := os.WriteFile(path, []byte{0xc3, 0x28}, 0600); err != nil {
		t.Fatal(err)
	}
	a.Open(path)
	inspector := a.Current()
	if inspector == nil || !inspector.ByteInspector || !inspector.CanReinterpret || inspector.Adapter != nil {
		t.Fatalf("invalid encoding did not open the inert byte inspector: %#v", inspector)
	}
	if strings.Contains(inspector.Warning, "encoding is unknown or invalid") ||
		!strings.Contains(inspector.Warning, "不会转码或覆盖原文件") {
		t.Fatalf("main byte-view warning is not actionable Chinese: %q", inspector.Warning)
	}
	foundDetail := false
	for _, issue := range a.Issues {
		if strings.Contains(issue, "原始错误：") && strings.Contains(issue, "encoding is unknown or invalid") {
			foundDetail = true
			break
		}
	}
	if !foundDetail {
		t.Fatalf("raw decode reason is missing from issue details: %#v", a.Issues)
	}
	tester := ui.NewTester(a.View, 1180, 780)
	if !tester.HasText("原始编码未知（字节视图）") {
		t.Fatalf("unknown byte inspector still claims a decoded encoding: %#v", tester.Texts())
	}
	if a.Save(inspector, true) {
		t.Fatal("byte inspector was allowed to save its hex presentation")
	}
}

func TestBinaryInspectorLabelsOriginalBytes(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "binary.dat")
	if err := os.WriteFile(path, []byte("a\x00b"), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	if tab == nil || !tab.ByteInspector || tab.CanReinterpret {
		t.Fatalf("binary content did not open as an inert byte inspector: %#v", tab)
	}
	tester := ui.NewTester(a.View, 1180, 780)
	if !tester.HasText("二进制（字节视图）") || tester.HasText("UTF-8") {
		t.Fatalf("binary inspector status is misleading: %#v", tester.Texts())
	}
}

func TestReadonlySaveAsMakesNewDocumentEditable(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "readonly.txt")
	destination := filepath.Join(root, "writable.txt")
	if err := os.WriteFile(source, []byte("readonly text"), 0600); err != nil {
		t.Fatal(err)
	}
	doc, err := document.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	doc.ReadOnly = true
	a := newTestApp(t, filepath.Join(root, "config"))
	a.add(doc, "readonly.txt", "")
	tab := a.Current()
	tab.Editor.ReadOnly = true
	a.saveDialog = func(*Tab) (string, error) { return destination, nil }

	if !a.Save(tab, false) {
		t.Fatal("readonly Save As failed")
	}
	if tab.Doc.Path != destination || tab.Doc.ReadOnly || tab.Editor.ReadOnly {
		t.Fatalf("saved copy retained readonly state: path=%q docReadOnly=%v editorReadOnly=%v", tab.Doc.Path, tab.Doc.ReadOnly, tab.Editor.ReadOnly)
	}
	tab.Editor.ReplaceText(&ui.TextInputRange{Start: 0, End: tab.Doc.Buffer.LenUTF16()}, "editable copy")
	if tab.Doc.Buffer.String() != "editable copy" {
		t.Fatal("new writable copy still rejected editing")
	}
}

func TestExplicitEncodingReopenNeverGuesses(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "legacy.txt")
	if err := os.WriteFile(path, []byte{0xc4, 0xe3, 0xba, 0xc3}, 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	if tab == nil || !tab.ByteInspector || !tab.CanReinterpret {
		t.Fatalf("legacy bytes did not reach the explicit-choice view: %#v", tab)
	}
	tester := ui.NewTester(a.View, 760, 480)
	labelRect, ok := tester.Find("原始编码无法自动识别。明确选择后重新打开：")
	if !ok {
		t.Fatalf("encoding choice label is missing: %#v", tester.Texts())
	}
	minButtonY := float32(480)
	for _, encoding := range []document.Encoding{document.UTF8, document.UTF16LE, document.UTF16BE, document.GBK, document.GB18030} {
		rect, ok := tester.Find(string(encoding))
		if !ok {
			t.Fatalf("%s encoding button is missing: %#v", encoding, tester.Texts())
		}
		if rect.X < 0 || rect.Y < 0 || rect.X+rect.W > 760 || rect.Y+rect.H > 480 {
			t.Fatalf("%s encoding button falls outside the 760x480 window: %#v", encoding, rect)
		}
		minButtonY = min(minButtonY, rect.Y)
	}
	if labelRect.Y+labelRect.H > minButtonY {
		t.Fatalf("encoding choice label overlaps its buttons: label=%#v first-button-y=%v", labelRect, minButtonY)
	}
	if a.reopenWithEncoding(tab, document.UTF8) {
		t.Fatal("UTF-8 choice silently accepted invalid bytes")
	}
	if !tab.ByteInspector || tab.Doc.Buffer.String() != tab.Doc.Buffer.String() {
		t.Fatal("failed explicit choice changed the byte inspector")
	}
	if !a.reopenWithEncoding(tab, document.GBK) {
		t.Fatal("explicit GBK choice failed")
	}
	if tab.ByteInspector || tab.Doc.Buffer.String() != "你好" || tab.Doc.Encoding != document.GBK {
		t.Fatalf("GBK reopen lost text or metadata: %#v %q", tab, tab.Doc.Buffer.String())
	}
}

func TestFlushFailureRetainsViewForLifecycleActions(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "note.md")
	if err := os.WriteFile(path, []byte("# saved\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	view := &failingView{flushErr: errors.New("draft conflict")}
	tab.View = view
	tab.Mode = "preview"

	if a.Save(tab, false) {
		t.Fatal("save succeeded after flush failure")
	}
	if tab.View != view || view.releases != 0 {
		t.Fatal("save released the view after flush failure")
	}
	a.Reload()
	if tab.View != view || view.releases != 0 {
		t.Fatal("reload released the view after flush failure")
	}
	if a.Select(-1) {
		t.Fatal("invalid select unexpectedly succeeded")
	}
}

func TestReplaceFlushesPendingRichEditFirst(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.NewDocument("markdown")
	tab := a.Current()
	tab.View = &failingView{flushErr: errors.New("draft conflict")}
	tab.Mode = "preview"
	before := tab.Doc.Buffer.String()
	a.Query = "新文档"
	a.Replacement = "replacement"

	a.replace(true)
	if tab.Doc.Buffer.String() != before {
		t.Fatal("replace mutated the document after a failed flush")
	}
}

func TestFindFlushesPendingRichEditFirst(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.NewDocument("markdown")
	tab := a.Current()
	view := &failingView{flushErr: errors.New("draft conflict")}
	tab.View = view
	tab.Mode = "preview"
	if err := tab.Doc.Buffer.SetSelection(1, 3); err != nil {
		t.Fatal(err)
	}
	before := tab.Doc.Buffer.Selection()
	a.Query = "新文档"

	a.find(true)
	if view.flushes != 1 || tab.View != view || view.releases != 0 {
		t.Fatal("failed find flush released or replaced the rich view")
	}
	if selection := tab.Doc.Buffer.Selection(); selection != before {
		t.Fatalf("failed find flush changed selection: %#v", selection)
	}
}

func TestFindRevealsSourceInsteadOfLeavingPreviewVisible(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.NewDocument("markdown")
	tab := a.Current()
	view := &failingView{}
	tab.View = view
	tab.Mode = "preview"
	a.Query = "新文档"

	a.find(true)
	if tab.Mode != "source" || tab.View != view || view.releases != 1 {
		t.Fatal("find left preview visible while revealing a hidden source editor")
	}
	if selection := tab.Doc.Buffer.Selection(); selection.Start() != 2 || selection.End() != 5 {
		t.Fatalf("find selection = %#v", selection)
	}
	if !a.persistPending {
		t.Fatal("source reveal did not mark session persistence pending")
	}
}

func TestReplaceReleasesStaleRichPresentationAfterMutation(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.NewDocument("markdown")
	tab := a.Current()
	view := &failingView{}
	tab.View = view
	tab.Mode = "preview"
	a.Query = "新文档"
	a.Replacement = "替换"

	a.replace(true)
	if tab.Doc.Buffer.String() != "# 替换\n\n" {
		t.Fatalf("replace result = %q", tab.Doc.Buffer.String())
	}
	if tab.View != view || view.releases != 1 {
		t.Fatal("replace did not release caches for the pre-replace buffer while retaining view state")
	}
}

func TestSessionMutationHelpersMarkPending(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "note.txt")
	if err := os.WriteFile(path, []byte("saved"), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.NewDocument("text")
	if !a.persistPending {
		t.Fatal("new tab did not mark session persistence pending")
	}
	a.persistPending = false
	a.Open(path)
	if !a.persistPending {
		t.Fatal("open did not mark session persistence pending")
	}
	if !a.Select(0) {
		t.Fatal("select failed")
	}
	if !a.persistPending {
		t.Fatal("tab selection did not mark session persistence pending")
	}
	tab := a.Current()
	tab.Mode = "preview"
	a.persistPending = false
	if !a.setMode(tab, "source") {
		t.Fatal("mode switch failed")
	}
	if !a.persistPending {
		t.Fatal("mode switch did not mark session persistence pending")
	}
}

func TestOrdinarySavePreservesModeViewAndSelection(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "note.md")
	if err := os.WriteFile(path, []byte("# saved\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	view := &failingView{}
	tab.View = view
	tab.Mode = "source"
	if err := tab.Doc.Buffer.SetSelection(2, 5); err != nil {
		t.Fatal(err)
	}

	if !a.Save(tab, false) {
		t.Fatal("ordinary save failed")
	}
	if tab.Adapter == nil || tab.View != view || tab.Mode != "source" {
		t.Fatal("ordinary save discarded the rich presentation state")
	}
	if selection := tab.Doc.Buffer.Selection(); selection.Start() != 2 || selection.End() != 5 {
		t.Fatalf("ordinary save changed selection: %#v", selection)
	}
}

func TestSaveFlushesPendingNativeIMEComposition(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "ime.txt")
	if err := os.WriteFile(path, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	tester := ui.NewTester(a.View, 760, 480)
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	tester.Compose("中", 1)
	if _, composing := tab.Editor.MarkedRange(); !composing {
		t.Fatal("native IME composition was not active before save")
	}
	if !a.Save(tab, false) {
		t.Fatal("save with pending IME composition failed")
	}
	if _, composing := tab.Editor.MarkedRange(); composing {
		t.Fatal("save left native IME composition open")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "中" || tab.Doc.Buffer.String() != "中" {
		t.Fatalf("save lost pending IME text: file=%q buffer=%q", data, tab.Doc.Buffer.String())
	}
}

func TestOpeningAnotherTabReleasesInactivePresentation(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.NewDocument("markdown")
	first := a.Current()
	view := &failingView{}
	first.View = view
	first.Mode = "preview"

	a.NewDocument("text")
	if view.releases != 1 || first.View != view {
		t.Fatalf("inactive presentation state was discarded: releases=%d view=%#v", view.releases, first.View)
	}
}

func TestModeSwitchReleasesCachesButRetainsViewState(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.NewDocument("mindmap")
	tab := a.Current()
	view := &failingView{
		cacheLoaded: true,
		pageIndex:   2,
		selectedID:  "node-7",
		zoom:        1.75,
		panX:        -84,
	}
	tab.View = view

	if !a.setMode(tab, "source") {
		t.Fatal("mode switch failed")
	}
	if tab.View != view || view.releases != 1 || view.cacheLoaded {
		t.Fatalf("source mode did not release heavy caches while retaining the view: releases=%d cached=%v view=%#v", view.releases, view.cacheLoaded, tab.View)
	}
	if view.pageIndex != 2 || view.selectedID != "node-7" || view.zoom != 1.75 || view.panX != -84 {
		t.Fatalf("source mode lost lightweight graph state: %#v", view)
	}
	view.cacheLoaded = true
	if !a.setMode(tab, "preview") {
		t.Fatal("switch back to preview failed")
	}
	if tab.View != view || view.releases != 2 || view.cacheLoaded {
		t.Fatalf("returning to preview replaced the view or failed to invalidate caches: releases=%d cached=%v view=%#v", view.releases, view.cacheLoaded, tab.View)
	}
	if view.pageIndex != 2 || view.selectedID != "node-7" || view.zoom != 1.75 || view.panX != -84 {
		t.Fatalf("returning to preview lost lightweight graph state: %#v", view)
	}
}

func TestAdapterReconfigureDisposesRetainedView(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.NewDocument("markdown")
	tab := a.Current()
	view := &failingView{}
	tab.View = view

	a.setPluginEnabled("markdown", false)
	if tab.View != nil || view.releases != 1 || tab.Adapter != nil {
		t.Fatalf("adapter replacement retained an obsolete view: releases=%d view=%#v", view.releases, tab.View)
	}
}

func TestPluginStateDoesNotChangeBeforeViewFlush(t *testing.T) {
	root := t.TempDir()
	a := newTestApp(t, root)
	a.NewDocument("markdown")
	tab := a.Current()
	view := &failingView{flushErr: errors.New("unfinished block")}
	tab.View = view
	a.setPluginEnabled("markdown", false)
	if !a.pluginManager.Enabled("markdown") {
		t.Fatal("failed view flush still disabled the builtin")
	}
	if tab.View != view || view.releases != 0 {
		t.Fatal("failed view flush released or replaced the format view")
	}
}

func TestWindowCloseCancellationDoesNotRemoveTabs(t *testing.T) {
	root := t.TempDir()
	a := newTestApp(t, filepath.Join(root, "config"))
	a.NewDocument("text")
	first := a.Current()
	a.NewDocument("text")
	first.View = &failingView{flushErr: errors.New("still editing")}
	if a.CanClose() {
		t.Fatal("window close succeeded after a flush failure")
	}
	if len(a.Tabs) != 2 || a.Tabs[0] != first {
		t.Fatal("cancelled window close mutated the tab list")
	}
}

func TestWindowCloseSavesReopenRecordsAndRestoresWorkspace(t *testing.T) {
	root := t.TempDir()
	firstPath := filepath.Join(root, "first.txt")
	savedPath := filepath.Join(root, "saved.txt")
	discardedPath := filepath.Join(root, "discarded.txt")
	for path, value := range map[string]string{
		firstPath:     "first disk",
		savedPath:     "saved disk",
		discardedPath: "discarded disk",
	} {
		if err := os.WriteFile(path, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	a := newTestApp(t, root)
	a.Open(firstPath)
	a.Open(savedPath)
	saved := a.Current()
	a.NewDocument("text")
	unsaved := a.Current()
	a.Open(discardedPath)
	discarded := a.Current()
	saved.Editor.ReplaceText(&ui.TextInputRange{Start: 0, End: saved.Doc.Buffer.LenUTF16()}, "saved edit")
	unsaved.Editor.ReplaceText(&ui.TextInputRange{Start: 0, End: unsaved.Doc.Buffer.LenUTF16()}, "unsaved draft")
	discarded.Editor.ReplaceText(&ui.TextInputRange{Start: 0, End: discarded.Doc.Buffer.LenUTF16()}, "discarded disk edit")

	decisions := make([]*Tab, 0, 3)
	a.closeDialog = func(tab *Tab) (closeDecision, error) {
		decisions = append(decisions, tab)
		if tab == saved {
			return closeSave, nil
		}
		return closeDiscard, nil
	}
	if !a.CanClose() {
		t.Fatal("clean quit workflow with explicit save/discard decisions failed")
	}
	if len(decisions) != 3 || len(a.Tabs) != 0 {
		t.Fatalf("close decisions or in-memory cleanup incomplete: decisions=%d tabs=%d", len(decisions), len(a.Tabs))
	}

	snapshot, issues := session.Load(filepath.Join(root, "session.json"))
	if len(issues) != 0 || len(snapshot.Tabs) != 3 {
		t.Fatalf("quit recovery records = %#v issues=%#v", snapshot, issues)
	}
	wantPaths := []string{firstPath, savedPath, discardedPath}
	for i, path := range wantPaths {
		if snapshot.Tabs[i].Path != path || snapshot.Tabs[i].Draft != nil {
			t.Fatalf("quit tab %d = %#v; want clean reopen record for %s", i, snapshot.Tabs[i], path)
		}
	}

	restored := newTestApp(t, root)
	restored.Restore()
	if len(restored.Tabs) != 3 {
		t.Fatalf("restored tabs = %d", len(restored.Tabs))
	}
	wantText := []string{"first disk", "saved edit", "discarded disk"}
	for i, text := range wantText {
		if restored.Tabs[i].Doc.Path != wantPaths[i] || restored.Tabs[i].Doc.Buffer.String() != text || restored.Tabs[i].Doc.Dirty() {
			t.Fatalf("restored tab %d = path %q text %q dirty=%v", i, restored.Tabs[i].Doc.Path, restored.Tabs[i].Doc.Buffer.String(), restored.Tabs[i].Doc.Dirty())
		}
	}
}

func TestWindowCloseCancelKeepsTabsAndRecoveryStable(t *testing.T) {
	root := t.TempDir()
	a := newTestApp(t, root)
	for _, name := range []string{"first.txt", "second.txt"} {
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, []byte(name), 0600); err != nil {
			t.Fatal(err)
		}
		a.Open(path)
		a.Current().Editor.ReplaceText(nil, name+" draft")
	}
	if err := a.persistTabs(a.Tabs, a.Active, "准备取消关闭测试"); err != nil {
		t.Fatal(err)
	}
	beforeTabs := append([]*Tab(nil), a.Tabs...)
	beforeSession, err := os.ReadFile(filepath.Join(root, "session.json"))
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	a.closeDialog = func(*Tab) (closeDecision, error) {
		calls++
		if calls == 1 {
			return closeDiscard, nil
		}
		return closeCancel, nil
	}

	if a.CanClose() {
		t.Fatal("window close succeeded after a cancel decision")
	}
	if len(a.Tabs) != len(beforeTabs) || a.Tabs[0] != beforeTabs[0] || a.Tabs[1] != beforeTabs[1] {
		t.Fatal("cancel changed tab identity or order")
	}
	afterSession, err := os.ReadFile(filepath.Join(root, "session.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(afterSession) != string(beforeSession) {
		t.Fatal("a discarded draft mutated recovery before the whole close decision completed")
	}
}

func TestRecoveredDraftIsBoundedUnsavedAndExplicit(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	if err := a.recoverDraft("note.md", "draft"); err != nil {
		t.Fatal(err)
	}
	current := a.Current()
	if current == nil || !current.Doc.IsNew || current.Doc.Buffer.String() != "draft" ||
		!strings.Contains(current.Warning, "未保存副本") {
		t.Fatalf("recovered draft = %#v", current)
	}
	if err := a.recoverDraft("large.md", strings.Repeat("x", session.MaxDraftBytes+1)); err == nil {
		t.Fatal("oversized recovery was accepted")
	}
}

func TestBoundedDirectoryReadDisclosesOverflow(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(name), 0600); err != nil {
			t.Fatal(err)
		}
	}
	entries, truncated, err := readDirectoryBounded(root, 2)
	if err != nil || len(entries) != 2 || !truncated {
		t.Fatalf("bounded read = %d, %v, %v", len(entries), truncated, err)
	}
}

func TestWorkspaceTreeDisclosesPerDirectoryLimit(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < treeDirectoryEntryLimit+1; i++ {
		name := fmt.Sprintf("item-%03d.txt", i)
		if err := os.WriteFile(filepath.Join(root, name), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Root = root
	a.refreshTreeNow()
	if len(a.Entries) != treeDirectoryEntryLimit || !a.TreeTruncated {
		t.Fatalf("tree limit = entries=%d truncated=%v", len(a.Entries), a.TreeTruncated)
	}
	tester := ui.NewTester(a.View, 1180, 780)
	if !tester.HasText(fmt.Sprintf("文件树按目录最多显示 %d 项，总计最多 %d 项；其余内容未载入。", treeDirectoryEntryLimit, treeEntryLimit)) {
		t.Fatalf("rendered tree limit is inaccurate: %#v", tester.Texts())
	}
}

func TestWorkspaceTreeVirtualizesRowsAndKeepsExpansionAnchor(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 5; i++ {
		dir := filepath.Join(root, fmt.Sprintf("folder-%02d", i))
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		for j := 0; j < 12; j++ {
			name := filepath.Join(dir, fmt.Sprintf("child-%02d-%02d.txt", i, j))
			if err := os.WriteFile(name, []byte(name), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Root = root
	for i := 0; i < 4; i++ {
		a.Expanded[filepath.Join(root, fmt.Sprintf("folder-%02d", i))] = true
	}
	a.refreshTreeNow()
	tester := ui.NewTester(a.View, 760, 480)
	target := -1
	for i, entry := range a.Entries {
		if entry.Path == filepath.Join(root, "folder-04") {
			target = i
			break
		}
	}
	if target < 0 {
		t.Fatal("target directory is absent from the bounded tree")
	}
	a.TreeList.ScrollTo(target, ui.Center)
	tester.Frame()
	tester.Frame()
	clickTesterText(t, tester, "› folder-04")
	tester.Frame()
	if a.TreeSelected < 0 || a.TreeSelected >= len(a.Entries) ||
		a.Entries[a.TreeSelected].Path != filepath.Join(root, "folder-04") {
		t.Fatalf("row click did not select the target directory: %d", a.TreeSelected)
	}
	tester.Key(0, ui.KeyEnter)
	a.refreshTreeNow()
	tester.Frame()
	if !a.Expanded[filepath.Join(root, "folder-04")] ||
		a.TreeSelected < 0 || a.TreeSelected >= len(a.Entries) ||
		a.Entries[a.TreeSelected].Path != filepath.Join(root, "folder-04") {
		t.Fatalf("directory click did not expand or retain its identity: selected=%d target=%d", a.TreeSelected, target)
	}
	if _, ok := tester.Find("⌄ folder-04"); !ok {
		t.Fatalf("expansion moved the directory identity away: %#v", tester.Texts())
	}
	childIndex := -1
	for i, entry := range a.Entries {
		if entry.Path == filepath.Join(root, "folder-04", "child-04-00.txt") {
			childIndex = i
			break
		}
	}
	if childIndex < 0 {
		t.Fatal("expanded child is absent from the tree model")
	}
	a.TreeList.ScrollIntoView(childIndex)
	tester.Frame()
	tester.Frame()
	if _, ok := tester.Find("child-04-00.txt"); !ok {
		t.Fatalf("expanded child rows did not appear near the retained anchor: %#v", tester.Texts())
	}

	a.Entries = make([]Entry, treeEntryLimit)
	for i := range a.Entries {
		a.Entries[i] = Entry{
			Path: filepath.Join(root, "synthetic", fmt.Sprintf("%04d.txt", i)),
			Name: fmt.Sprintf("%04d.txt", i),
		}
	}
	farPath := filepath.Join(root, "far.txt")
	if err := os.WriteFile(farPath, []byte("FAR"), 0600); err != nil {
		t.Fatal(err)
	}
	a.Entries[len(a.Entries)-1].Path = farPath
	a.Entries[len(a.Entries)-1].Name = "last.txt"
	var built []int
	a.treeRowBuilt = func(i int) { built = append(built, i) }
	a.TreeList.ScrollTo(0, ui.Start)
	tester.Frame()
	tester.Frame()
	clickTesterText(t, tester, "0000.txt")
	tester.Frame()
	first, last := a.TreeList.Visible()
	if first != 0 || last < 0 || last-first+1 >= 100 {
		t.Fatalf("tree visible range = %d..%d", first, last)
	}
	if len(built) == 0 || len(built) >= 200 {
		t.Fatalf("tree rebuilt too many of %d rows: callbacks=%d visible=%d..%d", len(a.Entries), len(built), first, last)
	}
	built = nil
	tester.Key(0, ui.KeyEnd)
	tester.Frame()
	tester.Frame()
	if a.TreeSelected != len(a.Entries)-1 {
		t.Fatalf("End did not choose the far tree row: selected=%d", a.TreeSelected)
	}
	foundLast := false
	for _, i := range built {
		if i == len(a.Entries)-1 {
			foundLast = true
			break
		}
	}
	if !foundLast || len(built) >= 200 {
		t.Fatalf("far tree row was not built as a bounded visible row: callbacks=%d", len(built))
	}
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	if current := a.Current(); current == nil || !strings.EqualFold(current.Doc.Path, farPath) {
		t.Fatalf("Enter did not activate the far tree row: %#v", current)
	}

	largeEntries := a.Entries
	measureRows := func() (allocs float64, bytesPerFrame uint64, maxRows int) {
		allocs = testing.AllocsPerRun(5, func() {
			built = nil
			tester.Frame()
			maxRows = max(maxRows, len(built))
		})
		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)
		const frames = 10
		for i := 0; i < frames; i++ {
			built = nil
			tester.Frame()
			maxRows = max(maxRows, len(built))
		}
		runtime.ReadMemStats(&after)
		bytesPerFrame = (after.TotalAlloc - before.TotalAlloc) / frames
		return
	}
	visibleFirst, visibleLast := a.TreeList.Visible()
	largeAllocs, largeBytes, largeMaxRows := measureRows()
	t.Logf("tree virtualization: rows=%d visible=%d maxRowsBuilt/frame=%d allocs/frame=%.1f bytes/frame=%d",
		len(largeEntries), visibleLast-visibleFirst+1, largeMaxRows, largeAllocs, largeBytes)

	a.Entries = append([]Entry(nil), largeEntries[:5]...)
	a.TreeSelected = -1
	a.TreeList = ui.ListState{}
	tester.Frame()
	tester.Frame()
	smallFirst, smallLast := a.TreeList.Visible()
	smallAllocs, smallBytes, smallMaxRows := measureRows()
	t.Logf("tree small-list comparison: rows=5 visible=%d maxRowsBuilt/frame=%d allocs/frame=%.1f bytes/frame=%d",
		smallLast-smallFirst+1, smallMaxRows, smallAllocs, smallBytes)
	a.Entries = largeEntries

	a.Root = t.TempDir()
	a.refreshTreeNow()
	if a.TreeSelected != -1 {
		t.Fatalf("root replacement retained an old tree selection: %d", a.TreeSelected)
	}
}

func TestWorkspaceSearchVirtualizesFarResultAndReplacement(t *testing.T) {
	root := t.TempDir()
	a := newTestApp(t, filepath.Join(root, "config"))
	a.ShowWorkspaceSearch = true
	a.SearchResults = make([]search.FileMatch, 2000)
	for i := range a.SearchResults {
		a.SearchResults[i] = search.FileMatch{
			Path: filepath.Join(root, fmt.Sprintf("result-%04d.txt", i)),
			Line: 1, Column: 1, Start: i, End: i + 3,
			Preview: fmt.Sprintf("result-%04d", i),
		}
	}
	firstPath := a.SearchResults[0].Path
	if err := os.WriteFile(firstPath, []byte("RESULT-0000"), 0600); err != nil {
		t.Fatal(err)
	}
	farPath := filepath.Join(root, "far.txt")
	if err := os.WriteFile(farPath, []byte("FAR-RESULT"), 0600); err != nil {
		t.Fatal(err)
	}
	a.SearchResults[len(a.SearchResults)-1] = search.FileMatch{
		Path: farPath, Line: 1, Column: 1, Start: 0, End: 3, Preview: "FAR-RESULT",
	}
	var built []int
	a.searchRowBuilt = func(i int) { built = append(built, i) }
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	tester.Frame()
	first, last := a.SearchList.Visible()
	if first != 0 || last < 0 || last-first+1 >= 100 {
		t.Fatalf("search visible range = %d..%d", first, last)
	}
	if len(built) == 0 || len(built) >= 200 {
		t.Fatalf("search rebuilt too many of %d rows: callbacks=%d visible=%d..%d", len(a.SearchResults), len(built), first, last)
	}
	largeResults := a.SearchResults
	measureRows := func() (allocs float64, bytesPerFrame uint64, maxRows int) {
		allocs = testing.AllocsPerRun(5, func() {
			built = nil
			tester.Frame()
			maxRows = max(maxRows, len(built))
		})
		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)
		const frames = 10
		for i := 0; i < frames; i++ {
			built = nil
			tester.Frame()
			maxRows = max(maxRows, len(built))
		}
		runtime.ReadMemStats(&after)
		bytesPerFrame = (after.TotalAlloc - before.TotalAlloc) / frames
		return
	}
	largeAllocs, largeBytes, largeMaxRows := measureRows()
	t.Logf("search virtualization: rows=%d visible=%d maxRowsBuilt/frame=%d allocs/frame=%.1f bytes/frame=%d",
		len(largeResults), last-first+1, largeMaxRows, largeAllocs, largeBytes)

	a.SearchResults = append([]search.FileMatch(nil), largeResults[:5]...)
	a.SearchSelected = -1
	a.SearchList = ui.ListState{}
	tester.Frame()
	tester.Frame()
	smallFirst, smallLast := a.SearchList.Visible()
	smallAllocs, smallBytes, smallMaxRows := measureRows()
	t.Logf("search small-list comparison: rows=5 visible=%d maxRowsBuilt/frame=%d allocs/frame=%.1f bytes/frame=%d",
		smallLast-smallFirst+1, smallMaxRows, smallAllocs, smallBytes)

	a.SearchResults = largeResults
	a.SearchSelected = -1
	a.SearchList = ui.ListState{}
	tester.Frame()
	tester.Frame()

	firstLabel := "result-0000.txt:1  result-0000"
	clickTesterText(t, tester, firstLabel)
	tester.Frame()
	built = nil
	tester.Key(0, ui.KeyEnd)
	tester.Frame()
	tester.Frame()
	label := "far.txt:1  FAR-RESULT"
	if _, ok := tester.Find(label); !ok {
		t.Fatalf("far search result is not reachable: %#v", tester.Texts())
	}
	if a.SearchSelected != len(a.SearchResults)-1 {
		t.Fatalf("End did not choose the far search row: selected=%d", a.SearchSelected)
	}
	if len(built) >= 200 {
		t.Fatalf("far search result rebuilt too many rows: callbacks=%d", len(built))
	}
	tester.Key(0, ui.KeyEnter)
	tester.Frame()
	if current := a.Current(); current == nil || !strings.EqualFold(current.Doc.Path, farPath) {
		t.Fatalf("Enter did not open the far search result: %#v", current)
	}
	if selection := a.Current().Doc.Buffer.Selection(); selection.Start() != 0 || selection.End() != 3 {
		t.Fatalf("far result reveal range = %#v", selection)
	}

	a.SearchResults = []search.FileMatch{{
		Path: farPath, Line: 1, Column: 1, Start: 4, End: 7, Preview: "replacement",
	}}
	a.SearchList.ScrollTo(0, ui.Start)
	a.searchRowBuilt = nil
	tester.Frame()
	tester.Frame()
	if !tester.HasText("far.txt:1  replacement") || tester.HasText(label) {
		t.Fatalf("search replacement kept stale rows: %#v", tester.Texts())
	}
	if a.SearchSelected != -1 {
		t.Fatalf("search replacement retained the removed result's selection: %d", a.SearchSelected)
	}
	a.SearchResults = nil
	tester.Frame()
	if !tester.HasText("没有匹配结果。") {
		t.Fatalf("empty search state is missing: %#v", tester.Texts())
	}
}

func TestViewWarningSummaryOpensFullDetails(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.NewDocument("markdown")
	tab := a.Current()
	tab.Mode = "preview"
	tab.View = &failingView{warnings: []string{"first warning", "second warning"}}
	tester := ui.NewTester(a.View, 1180, 780)
	if !tester.HasText("first warning（另有 1 项）") || !tester.HasText("查看详情") {
		t.Fatalf("warning summary or detail action missing: %#v", tester.Texts())
	}
	if err := tester.Click("查看详情"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if !a.ShowIssues || !tester.HasText("second warning") || len(tab.ViewWarnings) != 2 {
		t.Fatalf("full warnings unavailable: show=%v warnings=%#v texts=%#v", a.ShowIssues, tab.ViewWarnings, tester.Texts())
	}
}

func TestWorkspaceSearchIssueDetailsAreBounded(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	issues := make([]search.FileIssue, searchIssueLimit+3)
	for i := range issues {
		issues[i] = search.FileIssue{
			Path:   fmt.Sprintf("skip-%03d.bin", i),
			Reason: "binary",
			Detail: "not a text document",
		}
	}
	a.addSearchIssues(issues)
	a.ShowWorkspaceSearch = true

	tester := ui.NewTester(a.View, 1180, 780)
	if len(a.SearchIssues) != searchIssueLimit || a.SearchIssuesDropped != 3 {
		t.Fatalf("search issue storage = %d dropped=%d", len(a.SearchIssues), a.SearchIssuesDropped)
	}
	if !tester.HasText("查看跳过详情") {
		t.Fatalf("search issue details action is missing: %#v", tester.Texts())
	}
	if err := tester.Click("查看跳过详情"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if !a.ShowIssues {
		t.Fatal("search issue details action did not open the Issues panel")
	}
	summary := fmt.Sprintf("工作区搜索跳过项（显示 %d 项，另有 %d 项未保留）", searchIssueLimit, 3)
	if !tester.HasText(summary) ||
		!tester.HasText("skip-000.bin · binary · not a text document") ||
		!tester.HasText("skip-199.bin · binary · not a text document") {
		t.Fatalf("bounded search issue details are incomplete: %#v", tester.Texts())
	}
}

func TestPluginManagerRowsReflowAtMinimumWidth(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.ShowPlugins = true
	tester := ui.NewTester(a.View, 760, 480)
	labels := []string{
		"格式插件", "安装", "重新扫描", "安装目录", "关闭插件",
		"Code / Text", "code", "停用插件 Code / Text",
	}
	for _, label := range labels {
		rect, ok := tester.Find(label)
		if !ok {
			t.Fatalf("%q is missing from the narrow plugin manager: %#v", label, tester.Texts())
		}
		if rect.X < 0 || rect.Y < 0 || rect.X+rect.W > 760 || rect.Y+rect.H > 480 {
			t.Fatalf("%q falls outside the 760x480 plugin manager: %#v", label, rect)
		}
	}
	for _, manifest := range a.pluginManager.Manifests() {
		metadata := extensionLines(manifest.Extensions)
		rect, ok := tester.Find(metadata)
		if !ok {
			t.Fatalf("%s extension metadata is missing: %#v", manifest.ID, tester.Texts())
		}
		if rect.X < 0 || rect.Y < 0 || rect.X+rect.W > 760 || rect.Y+rect.H > 480 {
			t.Fatalf("%s extension metadata falls outside the 760x480 plugin manager: %#v", manifest.ID, rect)
		}
	}
}

func TestPluginManagerKeepsTabIdentityAndReplacesDocumentCanvas(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "settings.toml")
	if err := os.WriteFile(path, []byte("enabled = true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	a.ShowPlugins = true
	tester := ui.NewTester(a.View, 760, 480)
	tabRect, ok := tester.Find("settings.toml")
	if !ok || tabRect.X < 0 || tabRect.Y < 0 || tabRect.X+tabRect.W > 760 || tabRect.Y+tabRect.H > 480 {
		t.Fatalf("active tab identity was lost behind the plugin manager: %#v", tabRect)
	}
	if tester.HasText("文档编辑区") {
		t.Fatalf("document editor remained active beneath the plugin manager: %#v", tester.Texts())
	}
	if err := tester.Click("关闭插件"); err != nil {
		t.Fatal(err)
	}
	// The plugin panel replaced the document branch; render one additional
	// settled frame so the restored source editor is part of the label tree.
	tester.Frame()
	tester.Frame()
	if a.ShowPlugins || !tester.HasText("文档编辑区") {
		t.Fatalf("closing the plugin manager did not restore the document canvas: %#v", tester.Texts())
	}
}

func TestSettingsControlsRemainVisibleAtMinimumWidth(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.ShowSettings = true
	tester := ui.NewTester(a.View, 760, 480)
	wrapLabel := "自动换行"
	if a.SoftWrap {
		wrapLabel = "关闭换行"
	}
	for _, label := range []string{
		"外观", "关闭设置", "切换明暗", "A−", "16 px", "A+",
		"隐藏行号", wrapLabel,
	} {
		rect, ok := tester.Find(label)
		if !ok {
			t.Fatalf("%q is missing from the minimum-width settings panel: %#v", label, tester.Texts())
		}
		if rect.X < 0 || rect.Y < 0 || rect.X+rect.W > 760 || rect.Y+rect.H > 480 {
			t.Fatalf("%q falls outside the 760x480 settings panel: %#v", label, rect)
		}
	}
}

func TestIssueHistoryLimitIsDisclosed(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	for i := 0; i < issueHistoryLimit+5; i++ {
		a.report(fmt.Errorf("issue-%03d", i))
	}
	a.ShowIssues = true
	tester := ui.NewTester(a.View, 1180, 780)
	if !tester.HasText(fmt.Sprintf(
		"问题清单只保留本次运行最近 %d 条（当前 %d 条）。另有 %d 条较早记录未保留。",
		issueHistoryLimit, issueHistoryLimit, 5,
	)) {
		t.Fatalf("issue-history limit is not disclosed: %#v", tester.Texts())
	}
	if len(a.Issues) != issueHistoryLimit || a.IssuesDropped != 5 {
		t.Fatalf("issue history = %d dropped=%d", len(a.Issues), a.IssuesDropped)
	}
}

func TestMinimumWindowKeepsFormatControlsVisible(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.NewDocument("markdown")
	a.Current().Mode = "split"
	tester := ui.NewTester(a.View, 760, 480)
	labels := []string{"新建", "文件夹", "保存", "查找", "插件", "设置", "阅读 / 编辑", "源码", "分屏", "导出"}
	rects := make([]ui.Rect, 0, len(labels))
	for _, label := range labels {
		rect, ok := tester.Find(label)
		if !ok {
			t.Fatalf("%q is clipped out of the minimum window: %#v", label, tester.Texts())
		}
		if rect.X < 0 || rect.Y < 0 || rect.X+rect.W > 760 || rect.Y+rect.H > 480 {
			t.Fatalf("%q falls outside the minimum window: %#v", label, rect)
		}
		rects = append(rects, rect)
	}
	for i := range rects {
		for j := i + 1; j < len(rects); j++ {
			if rects[i].X < rects[j].X+rects[j].W && rects[j].X < rects[i].X+rects[i].W &&
				rects[i].Y < rects[j].Y+rects[j].H && rects[j].Y < rects[i].Y+rects[i].H {
				t.Fatalf("%q and %q overlap at 760x480", labels[i], labels[j])
			}
		}
	}
}

func TestReadonlyMindmapAtMinimumSizeKeepsTabModeAndCanvasUsable(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "planning.mindmap")
	if err := os.WriteFile(path, []byte(builtinCollapsedMindmap), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	tab.Doc.ReadOnly = true
	tab.Editor.ReadOnly = true
	a.ShowOutline = true
	runner := newControlledExporter()
	a.exporter = runner
	tester := ui.NewTester(a.View, 760, 480)
	call, job := awaitExportCall(t, runner), a.previewJob
	call.response <- previewSuccess()
	awaitPreviewDone(t, job)
	tester.Frame()
	tester.Frame()

	for _, label := range []string{
		"planning.mindmap", "关闭 planning.mindmap",
		"阅读 / 编辑", "源码", "分屏", "导出",
	} {
		rect, ok := tester.Find(label)
		if !ok {
			t.Fatalf("readonly mindmap %q is missing at 760x480: %#v", label, tester.Texts())
		}
		wantHeight := float32(12)
		if label == "关闭 planning.mindmap" {
			wantHeight = 24
		}
		if rect.W <= 0 || rect.H < wantHeight ||
			rect.X < 0 || rect.Y < 0 || rect.X+rect.W > 760 || rect.Y+rect.H > 480 {
			t.Fatalf("readonly mindmap %q is clipped at 760x480: %#v", label, rect)
		}
	}
	tabRect, _ := tester.Find("planning.mindmap")
	closeRect, _ := tester.Find("关闭 planning.mindmap")
	modeRect, _ := tester.Find("阅读 / 编辑")
	if tabRect.Y < closeRect.Y || tabRect.Y+tabRect.H > closeRect.Y+closeRect.H ||
		modeRect.Y < closeRect.Y+closeRect.H {
		t.Fatalf("tab/mode rows overlap or shrink: tab=%#v close=%#v mode=%#v", tabRect, closeRect, modeRect)
	}
	canvas, ok := tester.Find("只读思维导图画布")
	if !ok || canvas.H < 100 || canvas.W < 100 {
		t.Fatalf("readonly graph canvas is unusable at 760x480: %#v present=%v", canvas, ok)
	}
	if !tester.HasText("此视图只读；节点属性、标签与图片引用请在源码中查看。") || !tester.HasText("查看只读文档源码") {
		t.Fatalf("readonly scene limitation/source route is missing: %#v", tester.Texts())
	}
}

func TestExportPublisherCompletesBeforePublishingAndNeverOverwrites(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "diagram.svg")
	payload := []byte("<svg>complete</svg>")
	if err := writeExportFile(path, payload); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != string(payload) {
		t.Fatalf("export contents = %q, %v", data, err)
	}
	if err := writeExportFile(path, []byte("replacement")); err == nil {
		t.Fatal("export publisher overwrote an existing target")
	}
	if data, _ := os.ReadFile(path); string(data) != string(payload) {
		t.Fatalf("failed export changed target: %q", data)
	}
}

func TestInvalidPluginManifestIsVisible(t *testing.T) {
	root := t.TempDir()
	pluginRoot := filepath.Join(root, "plugins")
	if err := os.MkdirAll(pluginRoot, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginRoot, "broken.json"), []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, root)
	found := false
	for _, issue := range a.Issues {
		if strings.Contains(issue, "broken.json") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("manifest load error was hidden: %#v", a.Issues)
	}
	a.ShowPlugins = true
	tt := ui.NewTester(a.View, 1180, 780)
	tt.Frame()
}
func TestSessionRestoresUnsavedAsCopy(t *testing.T) {
	dir := t.TempDir()
	a := newTestApp(t, dir)
	a.NewDocument("text")
	a.Current().Editor.ReplaceText(nil, "待保存内容")
	a.LineNumbers = false
	a.Persist()
	b := newTestApp(t, dir)
	b.Restore()
	if len(b.Tabs) != 1 || b.Current().Doc.Buffer.String() != "待保存内容" || !b.Current().Doc.IsNew {
		t.Fatal("draft recovery failed")
	}
	if b.LineNumbers {
		t.Fatal("line-number preference was not restored")
	}
}

func TestRecoveredCopyPreservesOriginalPathAcrossRepeatedRestores(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source.txt")
	if err := os.WriteFile(source, []byte("disk version"), 0600); err != nil {
		t.Fatal(err)
	}

	first := newTestApp(t, dir)
	first.Open(source)
	first.Current().Editor.ReplaceText(
		&ui.TextInputRange{Start: 0, End: first.Current().Doc.Buffer.LenUTF16()},
		"draft version",
	)
	first.Persist()
	first.Shutdown()

	second := newTestApp(t, dir)
	second.Restore()
	current := second.Current()
	if current == nil || !current.Doc.IsNew || current.Doc.Path != "" ||
		current.Doc.Buffer.String() != "draft version" || current.OriginalPath != source {
		if current == nil {
			t.Fatal("first restore did not produce a tab")
		}
		t.Fatalf("first restore lost copy/original identity: isNew=%v path=%q text=%q original=%q want=%q",
			current.Doc.IsNew, current.Doc.Path, current.Doc.Buffer.String(), current.OriginalPath, source)
	}
	second.Persist()
	second.Shutdown()

	snapshot, issues := session.Load(filepath.Join(dir, "session.json"))
	if len(issues) != 0 || len(snapshot.Tabs) != 1 {
		t.Fatalf("second recovery snapshot = %#v issues=%#v", snapshot, issues)
	}
	if snapshot.Tabs[0].Path != "" || snapshot.Tabs[0].OriginalPath != source ||
		snapshot.Tabs[0].Draft == nil || *snapshot.Tabs[0].Draft != "draft version" {
		t.Fatalf("second persist lost original-path metadata: %#v", snapshot.Tabs[0])
	}

	third := newTestApp(t, dir)
	third.Restore()
	defer third.Shutdown()
	current = third.Current()
	if current == nil || !current.Doc.IsNew || current.Doc.Path != "" ||
		current.OriginalPath != source || current.Doc.Buffer.String() != "draft version" {
		t.Fatalf("second restore lost original-path metadata: %#v", current)
	}
	if disk, err := os.ReadFile(source); err != nil || string(disk) != "disk version" {
		t.Fatalf("restored copy changed the original file: %q err=%v", disk, err)
	}
}

func TestSessionRetainsReadonlyTextButNotByteInspector(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "readonly.txt")
	if err := os.WriteFile(path, []byte("readonly text"), 0600); err != nil {
		t.Fatal(err)
	}
	readonly, err := document.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	readonly.ReadOnly = true
	a := newTestApp(t, filepath.Join(dir, "config"))
	a.add(readonly, "readonly.txt", "")
	a.Persist()
	snapshot, issues := session.Load(filepath.Join(dir, "config", "session.json"))
	if len(issues) != 0 || len(snapshot.Tabs) != 1 || snapshot.Tabs[0].Path != path {
		t.Fatalf("readonly text was omitted from recovery: %#v %#v", snapshot, issues)
	}

	restored := newTestApp(t, filepath.Join(dir, "config"))
	restored.Restore()
	if len(restored.Tabs) != 1 || restored.Current().Doc.Buffer.String() != "readonly text" {
		t.Fatalf("readonly text did not restore: %#v", restored.Tabs)
	}

	inspector := document.New("byte view")
	inspector.Path = path
	restored.addDocument(inspector, "readonly.txt", "", true, true)
	restored.Persist()
	snapshot, issues = session.Load(filepath.Join(dir, "config", "session.json"))
	if len(issues) != 0 || len(snapshot.Tabs) != 1 {
		t.Fatalf("byte inspector was persisted as a text tab: %#v %#v", snapshot, issues)
	}
}
func TestFirstRunKeepsDefaultLineNumbers(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.Restore()
	if !a.LineNumbers {
		t.Fatal("missing recovery file reset the default line-number preference")
	}
}
func TestExplicitlyDiscardedTabIsRemovedFromRecovery(t *testing.T) {
	dir := t.TempDir()
	a := newTestApp(t, dir)
	a.NewDocument("text")
	a.Current().Editor.ReplaceText(nil, "discard me")
	a.Persist()
	remaining, active := tabsAfterRemoval(a.Tabs, a.Active, 0)
	if !a.commitClose(remaining, active, func() { a.removeTab(0) }) {
		t.Fatal("explicit discard could not commit its recovery update")
	}

	restored := newTestApp(t, dir)
	restored.Restore()
	if len(restored.Tabs) != 0 {
		t.Fatalf("explicitly discarded draft was restored: %#v", restored.Tabs)
	}
}

func TestAutomaticRecoveryFailureIsNotRetriedUntilStateChanges(t *testing.T) {
	dir := t.TempDir()
	a := newTestApp(t, dir)
	a.NewDocument("text")
	a.Current().Editor.ReplaceText(nil, "draft")
	// Force a persistence failure without relying on a large allocation.
	if err := os.Mkdir(filepath.Join(dir, "session.json"), 0700); err != nil {
		t.Fatal(err)
	}

	a.persistPending = true
	if err := a.persistAutomatic(); err == nil {
		t.Fatal("automatic recovery unexpectedly succeeded")
	}
	if a.persistPending {
		t.Fatal("failed automatic recovery remained queued for every UI frame")
	}
	if err := a.persistAutomatic(); err != nil {
		t.Fatalf("unchanged automatic state retried and failed again: %v", err)
	}

	a.Current().Editor.ReplaceText(nil, "new draft")
	if !a.persistPending {
		t.Fatal("new document change did not request a fresh automatic attempt")
	}
}

func TestClosePersistsOnlyTheRemainingTabs(t *testing.T) {
	dir := t.TempDir()
	a := newTestApp(t, dir)
	a.NewDocument("text")
	a.Current().Editor.ReplaceText(nil, "discard me")
	a.NewDocument("text")
	a.Current().Editor.ReplaceText(nil, "keep me")
	a.Persist()

	remaining, active := tabsAfterRemoval(a.Tabs, a.Active, 0)
	if !a.commitClose(remaining, active, func() { a.removeTab(0) }) {
		t.Fatal("close could not persist its recovery update")
	}
	snapshot, issues := session.Load(filepath.Join(dir, "session.json"))
	if len(issues) != 0 || len(snapshot.Tabs) != 1 {
		t.Fatalf("remaining recovery = %#v issues=%#v", snapshot, issues)
	}
	if snapshot.Tabs[0].Draft == nil || *snapshot.Tabs[0].Draft != "keep me" {
		t.Fatalf("wrong tab was retained after close: %#v", snapshot.Tabs[0])
	}
}

func TestCloseDoesNotDiscardTabWhenRemainingDraftIsOversized(t *testing.T) {
	dir := t.TempDir()
	a := newTestApp(t, dir)
	a.NewDocument("text")
	discard := a.Current()
	discard.Editor.ReplaceText(nil, "discard me")
	a.NewDocument("text")
	retained := a.Current()
	retained.Editor.ReplaceText(nil, "small draft")
	a.Persist()

	retained.Editor.ReplaceText(nil, strings.Repeat("x", session.MaxDraftBytes+1))
	remaining, active := tabsAfterRemoval(a.Tabs, a.Active, 0)
	if a.commitClose(remaining, active, func() { a.removeTab(0) }) {
		t.Fatal("close succeeded while the remaining draft could not be persisted")
	}
	if len(a.Tabs) != 2 || a.Tabs[0] != discard || a.Tabs[1] != retained {
		t.Fatal("failed recovery update removed the explicitly discarded tab")
	}
	snapshot, issues := session.Load(filepath.Join(dir, "session.json"))
	if len(issues) != 0 || len(snapshot.Tabs) != 2 {
		t.Fatalf("old recovery snapshot was overwritten or removed: tabs=%d issues=%#v", len(snapshot.Tabs), issues)
	}
}

func TestCloseRejectsEscapeHeavyRecoveryBeforeChangingWorkflow(t *testing.T) {
	dir := t.TempDir()
	a := newTestApp(t, dir)
	a.NewDocument("text")
	previous := a.Current()
	previous.Editor.ReplaceText(nil, "previous recovery")
	a.Persist()
	before, err := os.ReadFile(filepath.Join(dir, "session.json"))
	if err != nil {
		t.Fatal(err)
	}

	a.NewDocument("text")
	retained := a.Current()
	// Three MiB is below the raw per-draft limit, but each NUL expands sixfold
	// in JSON and exceeds the 16 MiB recovery-file budget.
	retained.Editor.ReplaceText(nil, strings.Repeat("\x00", 3<<20))
	a.closeDialog = func(*Tab) (closeDecision, error) { return closeDiscard, nil }

	if a.CloseTab(0) {
		t.Fatal("close succeeded while the escape-heavy remaining draft could not be persisted")
	}
	if len(a.Tabs) != 2 || a.Tabs[0] != previous || a.Tabs[1] != retained || a.Active != 1 {
		t.Fatalf("failed close changed the tab workflow: active=%d tabs=%#v", a.Active, a.Tabs)
	}
	after, err := os.ReadFile(filepath.Join(dir, "session.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("escape-heavy preflight failure overwrote the previous recovery")
	}
	if !strings.Contains(a.lastPersistError, "existing recovery file kept") {
		t.Fatalf("encoded recovery rejection was not disclosed: %q", a.lastPersistError)
	}
}

func TestCloseStopsBeforeCapturingAggregateOversizedRecovery(t *testing.T) {
	dir := t.TempDir()
	a := newTestApp(t, dir)
	a.NewDocument("text")
	a.Current().Editor.ReplaceText(nil, "previous recovery")
	a.Persist()

	for i := 0; i < session.MaxDraftAggregateBytes/session.MaxDraftBytes+1; i++ {
		a.appendTab(document.New(strings.Repeat("x", session.MaxDraftBytes)),
			fmt.Sprintf("large-%d.txt", i), "", false, false)
	}
	a.Active = len(a.Tabs) - 1
	if a.persistQuitTabs(a.Tabs, a.Active, nil) == nil {
		t.Fatal("aggregate oversized recovery unexpectedly succeeded")
	}
	if len(a.Tabs) != session.MaxDraftAggregateBytes/session.MaxDraftBytes+2 {
		t.Fatal("aggregate recovery failure changed the open tabs")
	}
	if !strings.Contains(a.lastPersistError, "现有恢复快照保持不变") {
		t.Fatalf("aggregate preflight was not disclosed: %q", a.lastPersistError)
	}
	snapshot, issues := session.Load(filepath.Join(dir, "session.json"))
	if len(issues) != 0 || len(snapshot.Tabs) != 1 ||
		snapshot.Tabs[0].Draft == nil || *snapshot.Tabs[0].Draft != "previous recovery" {
		t.Fatalf("previous recovery changed after aggregate preflight failure: %#v issues=%#v", snapshot, issues)
	}
}

func TestCloseDoesNotRemoveTabsWhenRecoveryWriteFails(t *testing.T) {
	dir := t.TempDir()
	a := newTestApp(t, dir)
	a.NewDocument("text")
	tab := a.Current()
	tab.Editor.ReplaceText(nil, "must remain")
	if err := os.Mkdir(filepath.Join(dir, "session.json"), 0700); err != nil {
		t.Fatal(err)
	}

	if a.commitClose(nil, -1, func() { a.removeTab(0) }) {
		t.Fatal("close succeeded after recovery persistence failed")
	}
	if len(a.Tabs) != 1 || a.Tabs[0] != tab {
		t.Fatal("recovery write failure removed the tab before close was durable")
	}
}
func TestScreenshot(t *testing.T) {
	if os.Getenv("LEVNOTE_SCREENSHOT") == "" {
		t.Skip("explicit visual verification")
	}
	a := newTestApp(t, t.TempDir())
	a.NewDocument("markdown")
	a.Current().Editor.ReplaceText(nil, "轻笺 · 原生工作区\n\n在一个窗口里，阅读、记录和整理。\n\n这一页由自研编辑器绘制。\n文本缓冲、选区与撤销历史独立于界面。\n\n打开文件  Ctrl+O\n保存文档  Ctrl+S\n查找替换  Ctrl+F\n")
	tt := ui.NewTester(a.View, 1180, 780)
	path := os.Getenv("LEVNOTE_SCREENSHOT")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err = png.Encode(f, tt.Image()); err != nil {
		t.Fatal(err)
	}
}
