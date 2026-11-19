package app

import (
	"bytes"
	"encoding/xml"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/mindmap"
)

const mindmapTagsInput = "新标签（单行，最多512字节）"

const mindmapTagsCleanBody = `{"format":"levnote.mindmap","version":1,"layout":"logicalStructure","root":{"id":"root","topic":"主题","icons":["⭐"],"tags":["同名","",{"text":"同名","className":"","style":{"color":"red"}},"同名"],"children":[]}}`

// This fixture is independent of the new helpers. Tag2 has a deliberately empty
// literal; tag3 shares text with tags1/4 but owns distinct ordered raw metadata.
const mindmapTagsBody = `{
  "format": "levnote.mindmap",
  "version": 1,
  "layout": "logicalStructure",
  "futureTop": {"n":1.00,"large":12345678901234567890},
  "root": {
    "id": "root",
    "topic": "主题",
    "note": "原备注",
    "icons": ["⭐"],
    "tags": ["同名", "", {"text":"同名","className":"","style":{"future":1.20,"color":"red"},"first":2.00,"second":{"n":3e2}}, "同名"],
    "image": {"url":"images/local.png","width":24,"height":16,"future":0.30},
    "futureNode": {"raw":4.00},
    "children": []
  }
}`

func mindmapTagsDiskFixture() []byte {
	return append([]byte{0xef, 0xbb, 0xbf}, []byte(strings.ReplaceAll(mindmapTagsBody, "\n", "\r\n"))...)
}

func openRealMindmapTags(t *testing.T, width, height float32) (*App, *Tab, *ui.Tester, string) {
	t.Helper()
	return openRealMindmapTagsBody(t, width, height, mindmapTagsBody)
}

func openRealMindmapTagsBody(t *testing.T, width, height float32, body string) (*App, *Tab, *ui.Tester, string) {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "tags.mindmap")
	disk := append([]byte{0xef, 0xbb, 0xbf}, []byte(strings.ReplaceAll(body, "\n", "\r\n"))...)
	if err := os.WriteFile(path, disk, 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	if tab == nil || tab.Adapter == nil || tab.Adapter.ID != "mindmap" || tab.Mode != "preview" || tab.Doc.Encoding != document.UTF8 || !tab.Doc.BOM {
		t.Fatal("registered mindmap/BOM fixture did not open")
	}
	tester := ui.NewTester(a.View, int(width), int(height))
	tester.Frame()
	return a, tab, tester, path
}

func revealTagAppControl(t *testing.T, tester *ui.Tester, width, height float32, label string, role ui.Role, minHeight float32) ui.AccessInfo {
	t.Helper()
	tester.Wheel(width-6, height-35, 0, -6000)
	tester.Advance(time.Second)
	for attempt := 0; attempt < 65; attempt++ {
		info, ok := tester.AccessByRole(role, label)
		if ok && !info.Offscreen && info.Bounds.W > 0 && info.Bounds.H >= minHeight-0.1 && info.Bounds.X >= 0 && info.Bounds.Y >= 0 && info.Bounds.X+info.Bounds.W <= width+0.1 && info.Bounds.Y+info.Bounds.H <= height-29+0.1 {
			t.Logf("tag control %q bounds=%+v disabled=%t", label, info.Bounds, info.Disabled)
			return info
		}
		tester.Wheel(width-6, height-35, 0, 20)
		tester.Advance(time.Second)
	}
	info, _ := tester.AccessByRole(role, label)
	t.Fatalf("tag control %q not reachable above status: %+v texts=%q", label, info, tester.Texts())
	return ui.AccessInfo{}
}

func clickTagAppButton(t *testing.T, tester *ui.Tester, width, height float32, label string) {
	t.Helper()
	button := revealTagAppControl(t, tester, width, height, label, ui.RoleButton, 28)
	if button.Disabled || !button.Focusable {
		t.Fatal("action not enabled/focusable", label, button)
	}
	tester.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
	tester.Frame()
}

func openTagsAppPanel(t *testing.T, tester *ui.Tester, width, height float32) {
	t.Helper()
	clickTagAppButton(t, tester, width, height, "展开标签面板")
}

func typeTagAppInput(t *testing.T, tester *ui.Tester, width, height float32, value string, compose bool) {
	t.Helper()
	field := revealTagAppControl(t, tester, width, height, mindmapTagsInput, ui.RoleTextField, 40)
	tester.ClickAt(field.Bounds.X+field.Bounds.W/2, field.Bounds.Y+field.Bounds.H/2)
	tester.Command("selectAll")
	if compose {
		tester.Compose(value, text.UTF16Len(value))
	} else {
		tester.Type(value)
	}
}

func typeTagOrdinaryField(t *testing.T, tester *ui.Tester, width, height float32, value string) {
	t.Helper()
	field := revealTagAppControl(t, tester, width, height, "节点文本", ui.RoleTextField, 48)
	tester.ClickAt(field.Bounds.X+field.Bounds.W/2, field.Bounds.Y+field.Bounds.H/2)
	tester.Command("selectAll")
	tester.Type(value)
}

// The repository editor draws its own text and exposes a native text-input
// client, but does not set accessibility Value. Read through the actual native
// select-all/copy path rather than pretending an empty AccessInfo.Value is loss.
func requireTagAppInputText(t *testing.T, tester *ui.Tester, width, height float32, label, want string) {
	t.Helper()
	field := revealTagAppControl(t, tester, width, height, label, ui.RoleTextField, 40)
	tester.ClickAt(field.Bounds.X+field.Bounds.W/2, field.Bounds.Y+field.Bounds.H/2)
	tester.Command("selectAll")
	tester.SetClipboard("__not_copied__")
	tester.Command("copy")
	if got := tester.Clipboard(); got != want {
		t.Fatalf("native copy of %q = %q, want %q", label, got, want)
	}
}

func requireAppTags(t *testing.T, tab *Tab, want ...string) *mindmap.Document {
	t.Helper()
	doc := parseMindmapBytes(t, []byte(tab.Doc.Buffer.String()))
	if doc.Root.Tags == nil {
		t.Fatal("tags presence lost")
	}
	got := make([]string, len(*doc.Root.Tags))
	for i, tag := range *doc.Root.Tags {
		got[i] = tag.Text
	}
	if !reflect.DeepEqual(got, want) && !(len(got) == 0 && len(want) == 0) {
		t.Fatalf("tags=%#v want=%#v", got, want)
	}
	return doc
}

func requireTagsAppMetadata(t *testing.T, doc *mindmap.Document, objectOrdinal int) {
	t.Helper()
	if raw, ok := doc.Extra.Get("futureTop"); !ok || string(raw) != `{"n":1.00,"large":12345678901234567890}` {
		t.Fatal("top raw/order lost", string(raw))
	}
	if raw, ok := doc.Root.Extra.Get("futureNode"); !ok || string(raw) != `{"raw":4.00}` {
		t.Fatal("node raw/order lost", string(raw))
	}
	if doc.Root.Note == nil || *doc.Root.Note != "原备注" || doc.Root.Icons == nil || !reflect.DeepEqual(*doc.Root.Icons, []string{"⭐"}) {
		t.Fatal("unrelated fields changed")
	}
	image := doc.Root.Image
	if image == nil || image.URL != "images/local.png" || image.Width != "24" || image.Height != "16" || len(image.Extra.Members) != 1 || string(image.Extra.Members[0].Raw) != "0.30" {
		t.Fatal("image metadata changed", image)
	}
	if objectOrdinal < 0 {
		return
	}
	tag := (*doc.Root.Tags)[objectOrdinal]
	if !tag.Object || tag.Text != "同名" || !tag.ClassNameSet || tag.ClassName != "" || !tag.StyleSet || len(tag.Style.Members) != 2 || tag.Style.Members[0].Key != "future" || string(tag.Style.Members[0].Raw) != "1.20" || tag.Style.Members[1].Key != "color" || string(tag.Style.Members[1].Raw) != `"red"` || len(tag.Extra.Members) != 2 || tag.Extra.Members[0].Key != "first" || string(tag.Extra.Members[0].Raw) != "2.00" || tag.Extra.Members[1].Key != "second" || string(tag.Extra.Members[1].Raw) != `{"n":3e2}` {
		t.Fatal("surviving object presence/member order/exact raw altered", tag)
	}
}

func TestMindmapTagsRealAdapterExactSaveUndoModesAndSVG(t *testing.T) {
	a, tab, tester, path := openRealMindmapTags(t, 1180, 780)
	original := tab.Doc.Buffer.String()
	if !a.Save(tab, false) || !bytes.Equal(mustReadFile(t, path), mindmapTagsDiskFixture()) || tab.Doc.Buffer.CanUndo() || tab.Doc.Dirty() {
		t.Fatal("unchanged BOM/CRLF save changed bytes", a.Message)
	}
	openTagsAppPanel(t, tester, 1180, 780)
	selection := text.Range{Anchor: 17, Head: 3}
	_ = tab.Doc.Buffer.SetSelection(selection.Anchor, selection.Head)
	clickTagAppButton(t, tester, 1180, 780, "移除第4个标签")
	requireTagsAppMetadata(t, requireAppTags(t, tab, "同名", "", "同名"), 2)
	if tab.Doc.Buffer.Selection() != selection || tab.Doc.Buffer.History().UndoDepth != 1 || !tab.Doc.Dirty() {
		t.Fatal("ordinal removal selection/dirty/history")
	}
	removed := tab.Doc.Buffer.String()
	if !tab.Editor.Command("undo") || tab.Doc.Buffer.String() != original || tab.Doc.Buffer.Selection() != selection || !tab.Editor.Command("redo") || tab.Doc.Buffer.String() != removed {
		t.Fatal("ordinal source/selection undo/redo not exact")
	}
	tester.Frame()
	const literal = "  中文🙂,<x>&  "
	typeTagAppInput(t, tester, 1180, 780, literal, false)
	clickTagAppButton(t, tester, 1180, 780, "添加标签")
	typeTagAppInput(t, tester, 1180, 780, "同名", false)
	tester.Key(0, ui.KeyF6)
	if !tester.Focused("添加标签") {
		t.Fatal("real adapter F6 route failed")
	}
	tester.Key(0, ui.KeySpace)
	requireTagsAppMetadata(t, requireAppTags(t, tab, "同名", "", "同名", literal, "同名"), 2)
	if tab.Doc.Buffer.History().UndoDepth != 3 {
		t.Fatal("duplicate/Unicode adds were not independent single edits")
	}
	data, err := tab.View.Export("svg")
	if err != nil {
		t.Fatal(err)
	}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var spans []string
	for {
		token, decodeErr := decoder.Token()
		if decodeErr == io.EOF {
			break
		}
		if decodeErr != nil {
			t.Fatal("SVG invalid XML", decodeErr)
		}
		if element, ok := token.(xml.StartElement); ok {
			if element.Name.Local == "x" || element.Name.Local == "script" {
				t.Fatal("tag text became an XML element")
			}
			if element.Name.Local == "tspan" {
				var value string
				if err := decoder.DecodeElement(&value, &element); err != nil {
					t.Fatal(err)
				}
				spans = append(spans, value)
			}
		}
	}
	wantSpan := "⭐ 主题 [同名 ·  · 同名 ·   中文🙂,<x>&   · 同名]"
	if len(spans) != 1 || spans[0] != wantSpan || !bytes.Contains(data, []byte("&lt;x&gt;&amp;")) {
		t.Fatalf("literal bounded SVG tag oracle: spans=%q", spans)
	}
	for _, mode := range []string{"source", "split", "preview"} {
		before, history := tab.Doc.Buffer.String(), tab.Doc.Buffer.History()
		if !a.setMode(tab, mode) {
			t.Fatal("mode refused after complete action", mode, a.Message)
		}
		tester.Frame()
		if tab.Mode != mode || tab.Doc.Buffer.String() != before || tab.Doc.Buffer.History() != history {
			t.Fatal("mode mutated source/history")
		}
	}
	edited := tab.Doc.Buffer.String()
	if !bytes.Equal(mustReadFile(t, path), mindmapTagsDiskFixture()) {
		t.Fatal("edit/export/mode wrote disk before Save")
	}
	if strings.Contains(edited, "\r\n") || !strings.Contains(edited, "\n") {
		t.Fatal("real edit canonical line ending expectation changed")
	}
	if !a.Save(tab, false) || tab.Doc.Dirty() || !tab.Doc.BOM || tab.Doc.Encoding != document.UTF8 || !bytes.Equal(mustReadFile(t, path), append([]byte{0xef, 0xbb, 0xbf}, []byte(edited)...)) {
		t.Fatal("edited exact canonical+BOM save mismatch", a.Message)
	}
	reopened := newTestApp(t, filepath.Join(t.TempDir(), "config"))
	reopened.Open(path)
	rtab := reopened.Current()
	rtester := ui.NewTester(reopened.View, 1180, 780)
	if rtab.Adapter.ID != "mindmap" || !rtab.Doc.BOM || rtab.Doc.Buffer.String() != edited || rtab.Doc.Dirty() {
		t.Fatal("saved tags failed real reopen")
	}
	requireTagsAppMetadata(t, requireAppTags(t, rtab, "同名", "", "同名", literal, "同名"), 2)
	openTagsAppPanel(t, rtester, 1180, 780)
	clickTagAppButton(t, rtester, 1180, 780, "清空全部标签")
	requireTagsAppMetadata(t, requireAppTags(t, rtab), -1)
	typeTagAppInput(t, rtester, 1180, 780, "最后一个", false)
	clickTagAppButton(t, rtester, 1180, 780, "添加标签")
	clickTagAppButton(t, rtester, 1180, 780, "移除第1个标签")
	requireAppTags(t, rtab)
	if rtab.Doc.Buffer.History().UndoDepth != 3 {
		t.Fatal("clear/add/last removal not separate")
	}
	readonly, err := document.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	readonly.ReadOnly = true
	ro := newTestApp(t, filepath.Join(t.TempDir(), "config"))
	ro.add(readonly, "tags.mindmap", "")
	runner := newControlledExporter()
	ro.exporter = runner
	rot := ui.NewTester(ro.View, 1180, 780)
	call, job := awaitExportCall(t, runner), ro.previewJob
	call.response <- previewSuccess()
	awaitPreviewDone(t, job)
	rot.Frame()
	for _, name := range []string{"添加标签", "清空全部标签", "移除第1个标签"} {
		info, ok := rot.AccessByRole(ui.RoleButton, name)
		if ok {
			t.Fatal("scene-only readonly view exposed tag editing action", name, info)
		}
	}
	if !rot.HasText("此视图只读；节点属性、标签与图片引用请在源码中查看。") {
		t.Fatal("readonly source limitation missing")
	}
	version, selection := readonly.Buffer.Version(), readonly.Buffer.Selection()
	if err := rot.Click("只读思维导图画布"); err != nil {
		t.Fatal(err)
	}
	rot.Type("不能添加标签🙂")
	if readonly.Buffer.String() != edited || readonly.Buffer.Version() != version || readonly.Buffer.Selection() != selection || readonly.Buffer.CanUndo() || readonly.Dirty() {
		t.Fatal("readonly controls mutated source")
	}
}

func TestMindmapTagsRealAdapterPublicBarriersAndSeparateFieldUndo(t *testing.T) {
	a, tab, tester, path := openRealMindmapTags(t, 1180, 780)
	a.add(document.New("other"), "other.txt", "")
	if !a.Select(0) {
		t.Fatal("fixture cannot return to graph")
	}
	tester.Frame()
	openTagsAppPanel(t, tester, 1180, 780)
	typeTagAppInput(t, tester, 1180, 780, "待添加🙂", false)
	buffer, view := tab.Doc.Buffer, tab.View
	before, version, history, selected := buffer.String(), buffer.Version(), buffer.History(), buffer.Selection()
	dialogs := 0
	a.exportDialog = func(mygo.SaveDialogOptions) (string, error) {
		dialogs++
		return filepath.Join(t.TempDir(), "unexpected.svg"), nil
	}
	a.closeDialog = func(*Tab) (closeDecision, error) { dialogs++; return closeDiscard, nil }
	if a.Save(tab, false) || a.setMode(tab, "source") || a.setMode(tab, "split") || a.Select(1) || a.CloseTab(0) || a.CanClose() {
		t.Fatal("pending tag escaped a host barrier")
	}
	a.exportCurrent()
	tester.Frame()
	if dialogs != 0 || a.Current() != tab || len(a.Tabs) != 2 || tab.Mode != "preview" || tab.View != view || tab.Doc.Buffer != buffer || buffer.String() != before || buffer.Version() != version || buffer.History() != history || buffer.Selection() != selected || !bytes.Equal(mustReadFile(t, path), mindmapTagsDiskFixture()) || !strings.Contains(a.Message, "新标签尚未添加") {
		t.Fatal("barrier altered authority/input or opened dialog", a.Message)
	}
	requireTagAppInputText(t, tester, 1180, 780, mindmapTagsInput, "待添加🙂")
	typeTagOrdinaryField(t, tester, 1180, 780, "本地主题")
	if buffer.String() != before || buffer.CanUndo() {
		t.Fatal("ordinary OnChange bypassed pending tag")
	}
	clickTagAppButton(t, tester, 1180, 780, "添加标签")
	doc := requireAppTags(t, tab, "同名", "", "同名", "同名", "待添加🙂")
	if doc.Root.Topic != "本地主题" || buffer.History().UndoDepth != 2 || !tab.Doc.Dirty() {
		t.Fatal("Add did not independently Flush ordinary field", a.Message)
	}
	completed, afterSelection := buffer.String(), buffer.Selection()
	if !tab.Editor.Command("undo") {
		t.Fatal("tag undo failed")
	}
	fieldOnly := buffer.String()
	if requireAppTags(t, tab, "同名", "", "同名", "同名").Root.Topic != "本地主题" || buffer.Selection() != afterSelection {
		t.Fatal("tag undo lost independently written field/selection")
	}
	if !tab.Editor.Command("undo") || buffer.String() != before || buffer.Selection() != selected {
		t.Fatal("ordinary field undo not exact")
	}
	if !tab.Editor.Command("redo") || buffer.String() != fieldOnly || !tab.Editor.Command("redo") || buffer.String() != completed {
		t.Fatal("separate redo failed")
	}
	tester.Frame()
	if !a.Select(1) || !a.Select(0) || !a.Save(tab, false) {
		t.Fatal("completed input did not release host barriers", a.Message)
	}
}

func TestMindmapTagsRealAdapterSplitConflictAndRecovery(t *testing.T) {
	a, tab, tester, path := openRealMindmapTags(t, 1180, 780)
	if !a.setMode(tab, "split") {
		t.Fatal(a.Message)
	}
	tester.Frame()
	openTagsAppPanel(t, tester, 1180, 780)
	typeTagAppInput(t, tester, 1180, 780, "标签原样🙂", false)
	typeTagOrdinaryField(t, tester, 1180, 780, "字段原样🙂")
	// Use the visible split source editor, not a private graph mutation.
	if err := tester.Click("文档编辑区"); err != nil {
		t.Fatal(err)
	}
	tester.Command("selectAll")
	external := strings.Replace(tab.Doc.Buffer.String(), `"topic": "主题"`, `"topic": "外部主题"`, 1)
	tester.Type(external)
	if tab.Doc.Buffer.String() != external {
		t.Fatal("actual split source input fixture failed")
	}
	version, history, selection, view := tab.Doc.Buffer.Version(), tab.Doc.Buffer.History(), tab.Doc.Buffer.Selection(), tab.View
	clickTagAppButton(t, tester, 1180, 780, "添加标签")
	if tab.Doc.Buffer.String() != external || tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.History() != history || tab.Doc.Buffer.Selection() != selection || tab.View != view || !strings.Contains(a.Message, "节点的源字段已改变") {
		t.Fatal("stale add retargeted input or hid conflict", a.Message)
	}
	for _, pair := range [][2]string{{"节点文本", "字段原样🙂"}, {mindmapTagsInput, "标签原样🙂"}} {
		requireTagAppInputText(t, tester, 1180, 780, pair[0], pair[1])
	}
	if a.Save(tab, false) || a.setMode(tab, "preview") || a.CloseTab(0) {
		t.Fatal("conflicting dual drafts escaped barrier")
	}
	clickTagAppButton(t, tester, 1180, 780, "收起标签面板")
	clickTagAppButton(t, tester, 1180, 780, "恢复节点及标签草稿")
	want := "tags.mindmap\n节点 ID: root\n页面 ID: \n字段: 节点文本\n原字段存在: true\n\n字段原样🙂\n\n--- 未添加的标签输入 ---\n节点 ID: root\n字段: 新标签（尚未添加）\n\n标签原样🙂"
	recovered := a.Current()
	if len(a.Tabs) != 2 || recovered == tab || !recovered.Doc.IsNew || recovered.Doc.Path != "" || recovered.Doc.Buffer.String() != want || !strings.Contains(recovered.Warning, "未保存副本") {
		t.Fatal("real synchronous RecoverDraft did not create exact unsaved copy", recovered.Doc.Buffer.String())
	}
	if tab.Doc.Buffer.String() != external || tab.Doc.Buffer.History() != history || !bytes.Equal(mustReadFile(t, path), mindmapTagsDiskFixture()) {
		t.Fatal("recoverDraft callback changed original source/disk")
	}
	if !a.Select(0) {
		t.Fatal("cannot return after recovery", a.Message)
	}
	tester.Frame()
	if _, err := tab.View.Export("svg"); err != nil {
		t.Fatal("recovered pending state left stale barrier", err)
	}
}

func TestMindmapTagsRealAdapterPostFieldBudgetRefusal(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "budget.mindmap")
	large := strings.Repeat("x", 3830)
	source := `{"format":"levnote.mindmap","version":1,"root":{"id":"root","topic":"根","tags":[{"text":"` + large + `"}]}}`
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	tester := ui.NewTester(a.View, 1180, 780)
	openTagsAppPanel(t, tester, 1180, 780)
	input := strings.Repeat("a", 512)
	typeTagAppInput(t, tester, 1180, 780, input, false)
	typeTagOrdinaryField(t, tester, 1180, 780, "写回字段")
	clickTagAppButton(t, tester, 1180, 780, "添加标签")
	doc := requireAppTags(t, tab, large)
	if doc.Root.Topic != "写回字段" || tab.Doc.Buffer.History().UndoDepth != 1 || !strings.Contains(a.Message, "节点字段已写回，标签未修改") {
		t.Fatal("post-field refusal not truthful", a.Message)
	}
	requireTagAppInputText(t, tester, 1180, 780, mindmapTagsInput, input)
	if a.Save(tab, false) {
		t.Fatal("post-field refusal dropped tag/barrier")
	}
	clickTagAppButton(t, tester, 1180, 780, "放弃标签输入")
	if !tab.Editor.Command("undo") || tab.Doc.Buffer.String() != source || !bytes.Equal(mustReadFile(t, path), []byte(source)) {
		t.Fatal("already written field not independently undoable")
	}
}

func TestMindmapTagsRealAdapterFirstWarningPreservesComposition(t *testing.T) {
	for _, mode := range []string{"preview", "split"} {
		t.Run(mode, func(t *testing.T) {
			a, tab, tester, path := openRealMindmapTagsBody(t, 1180, 780, mindmapTagsCleanBody)
			if !a.setMode(tab, mode) {
				t.Fatal(a.Message)
			}
			tester.Frame()
			if warnings := tab.View.Warnings(); len(warnings) != 0 {
				t.Fatal("fixture must start with no conditional warning row", warnings)
			}
			openTagsAppPanel(t, tester, 1180, 780)
			typeTagAppInput(t, tester, 1180, 780, "未确认", true)
			version, selection, history := tab.Doc.Buffer.Version(), tab.Doc.Buffer.Selection(), tab.Doc.Buffer.History()
			tester.Key(0, ui.KeyF6)
			if !tester.Focused(mindmapTagsInput) || !tester.HasText("请先确认正在输入") || len(tab.View.Warnings()) == 0 || tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.Selection() != selection || tab.Doc.Buffer.History() != history {
				t.Fatal("first conditional warning lost focus or changed source", tab.View.Warnings())
			}
			// Updating then confirming composition must replace the marked
			// range, not append after an implicitly unmarked earlier value.
			tester.Compose("再次组合🙂", 6)
			tester.Key(ui.Shift, ui.KeyTab)
			if !tester.Focused(mindmapTagsInput) {
				t.Fatal("second composition escaped its input")
			}
			tester.Type("确认🙂")
			if !tester.Focused(mindmapTagsInput) || len(tab.View.Warnings()) != 0 || tab.Doc.Buffer.String() != mindmapTagsCleanBody || tab.Doc.Buffer.History() != history || strings.Contains(a.Message, "请先确认正在输入") || len(a.Issues) != 0 {
				t.Fatal("removing warning lost focus or retained stale guidance", tab.View.Warnings())
			}
			tester.Key(0, ui.KeyF6)
			if !tester.Focused("添加标签") {
				t.Fatal("confirmed input cannot reach actions")
			}
			tester.Key(0, ui.KeyEnter)
			requireAppTags(t, tab, "同名", "", "同名", "同名", "确认🙂")
			if tab.Doc.Buffer.History().UndoDepth != 1 || !bytes.Equal(mustReadFile(t, path), append([]byte{0xef, 0xbb, 0xbf}, []byte(mindmapTagsCleanBody)...)) {
				t.Fatal("marked input confirmation/commit duplicated or touched disk")
			}
		})
	}
}

func TestMindmapTagsAppNarrowCaptures(t *testing.T) {
	out := os.Getenv("LEVNOTE_MINDMAP_TAGS_APP_SCREENSHOT_DIR")
	if out == "" {
		t.Skip("set LEVNOTE_MINDMAP_TAGS_APP_SCREENSHOT_DIR")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	for _, dark := range []bool{false, true} {
		name := "light-expanded"
		if dark {
			name = "dark-recovery"
		}
		t.Run(name, func(t *testing.T) {
			// Keep rich unknown/image metadata in the functional tests above.
			// This capture fixture omits those warning-producing fields so the
			// input and recovery actions can be read together at760x480; it
			// still contains duplicate, empty and object tags.
			body := mindmapTagsCleanBody
			a, tab, tester, path := openRealMindmapTagsBody(t, 760, 480, body)
			a.Dark = dark
			tester.SetDark(dark)
			tester.Frame()
			openTagsAppPanel(t, tester, 760, 480)
			typeTagAppInput(t, tester, 760, 480, "新标签🙂", true)
			tester.Key(0, ui.KeyF6)
			inputAfter, _ := tester.AccessByRole(ui.RoleTextField, mindmapTagsInput)
			if !inputAfter.Focused || !tester.HasText("请先确认正在输入") || tab.Doc.Buffer.CanUndo() {
				captureAcceptanceTester(t, filepath.Join(out, "mindmap-tags-"+name+"-composition-diagnostic.png"), tester)
				t.Fatalf("app composition guard: input=%+v guidance=%t undo=%t texts=%q", inputAfter, tester.HasText("请先确认正在输入"), tab.Doc.Buffer.CanUndo(), tester.Texts())
			}
			tester.Type("新标签🙂")
			if strings.Contains(a.Message, "请先确认正在输入") {
				t.Fatal("confirmed capture retained stale status guidance", a.Message)
			}
			if problem, ok := tester.Access("标签操作问题"); ok && strings.Contains(problem.Description, "请先确认正在输入") {
				t.Fatal("confirmed capture retained stale focus warning", problem)
			}
			if dark {
				typeTagOrdinaryField(t, tester, 760, 480, "保留字段🙂")
				// A refused multiline tag uses the real input; both drafts stay.
				typeTagAppInput(t, tester, 760, 480, "坏\n标签🙂", false)
				clickTagAppButton(t, tester, 760, 480, "添加标签")
				if !strings.Contains(a.Message, "非空单行UTF-8") {
					t.Fatal("capture lacks actual refusal", a.Message)
				}
				// The compact ordinary error row must also fit this width.
				revealTagAppControl(t, tester, 760, 480, "恢复节点及标签草稿", ui.RoleButton, 28)
			} else {
				tester.Key(0, ui.KeyF6)
				if !tester.Focused("添加标签") {
					t.Fatal("app F6 did not reach Add")
				}
			}
			// Preserve a real failing compact layout before reachability checks.
			imagePath := filepath.Join(out, "mindmap-tags-"+name+"-760x480.png")
			captureAcceptanceTester(t, imagePath, tester)
			// Every action must be reachable above the fixed status bar. This
			// is a scrollable inspector, not a claim all rows fit at once.
			for _, label := range []string{"添加标签", "放弃标签输入", "恢复标签及节点草稿", "移除第4个标签"} {
				info := revealTagAppControl(t, tester, 760, 480, label, ui.RoleButton, 28)
				if label != "移除第4个标签" && (info.Disabled || !info.Focusable) {
					t.Fatal("capture primary action disabled", label, info)
				}
			}
			revealTagAppControl(t, tester, 760, 480, "恢复标签及节点草稿", ui.RoleButton, 28)
			// After scrolling to recovery, require the useful input/actions in
			// the same frame, rather than capturing only a clipped footer.
			for _, control := range []struct {
				label  string
				role   ui.Role
				height float32
			}{
				{mindmapTagsInput, ui.RoleTextField, 40}, {"添加标签", ui.RoleButton, 28}, {"放弃标签输入", ui.RoleButton, 28}, {"恢复标签及节点草稿", ui.RoleButton, 28},
			} {
				info, ok := tester.AccessByRole(control.role, control.label)
				if !ok || info.Offscreen || info.Bounds.H < control.height || info.Bounds.X < 0 || info.Bounds.X+info.Bounds.W > 760.1 || info.Bounds.Y < 0 || info.Bounds.Y+info.Bounds.H > 451.1 {
					t.Fatal("final capture lacks complete input/action context", control.label, info)
				}
			}
			captureAcceptanceTester(t, imagePath, tester)
			if tab.Doc.Buffer.String() != body || tab.Doc.Dirty() || !bytes.Equal(mustReadFile(t, path), append([]byte{0xef, 0xbb, 0xbf}, []byte(body)...)) {
				t.Fatal("staged capture altered source/disk")
			}
			if dark {
				clickTagAppButton(t, tester, 760, 480, "恢复标签及节点草稿")
				if len(a.Tabs) != 2 || a.Current() == tab || !strings.Contains(a.Current().Doc.Buffer.String(), "保留字段🙂") || !strings.HasSuffix(a.Current().Doc.Buffer.String(), "坏\n标签🙂") {
					t.Fatal("visible dark recovery lost exact drafts")
				}
			} else {
				clickTagAppButton(t, tester, 760, 480, "添加标签")
				requireAppTags(t, tab, "同名", "", "同名", "同名", "新标签🙂")
				if tab.Doc.Buffer.History().UndoDepth != 1 {
					t.Fatal("confirmed capture input duplicated on Add")
				}
			}
		})
	}
}
