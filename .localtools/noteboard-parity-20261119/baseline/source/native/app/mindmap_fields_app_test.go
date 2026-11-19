package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/formats/mindmap"
)

const mindmapFieldsFixture = `{
  "format": "levnote.mindmap",
  "version": 1,
  "futureTop": 123456789012345678901234567890,
  "layout": "logicalStructure",
  "root": {
    "id": "root",
    "topic": "中心",
    "note": "原备注",
    "hyperLink": "target.md",
    "unknownNode": {"kept": 1.00},
    "children": []
  }
}`

func fieldControl(t *testing.T, tester *ui.Tester, label string) (ui.Rect, string) {
	t.Helper()
	for _, candidate := range []string{label, label + "（当前）", "（当前）" + label} {
		if rect, ok := tester.Find(candidate); ok {
			return rect, candidate
		}
	}
	t.Fatalf("%q (including active variant) is missing: %#v", label, tester.Texts())
	return ui.Rect{}, ""
}

func clickFieldControl(t *testing.T, tester *ui.Tester, label string) {
	t.Helper()
	rect, _ := fieldControl(t, tester, label)
	tester.ClickAt(rect.X+rect.W/2, rect.Y+rect.H/2)
	tester.Frame()
}

func beginGraphFieldComposition(t *testing.T, tester *ui.Tester, fieldLabel, editorLabel, value string) {
	t.Helper()
	clickFieldControl(t, tester, fieldLabel)
	if err := tester.Click(editorLabel); err != nil {
		t.Fatalf("field editor %q is missing: %v", editorLabel, err)
	}
	tester.Command("selectAll")
	tester.Compose(value, utf8.RuneCountInString(value))
}

func parseMindmapBytes(t *testing.T, data []byte) *mindmap.Document {
	t.Helper()
	document, _, err := mindmap.Parse(data, mindmap.ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return document
}

func requireUnknownMindmapMetadata(t *testing.T, data []byte) {
	t.Helper()
	var top map[string]json.RawMessage
	if err := json.Unmarshal(data, &top); err != nil {
		t.Fatal(err)
	}
	if string(top["futureTop"]) != "123456789012345678901234567890" {
		t.Fatalf("top-level unknown metadata changed: %s", top["futureTop"])
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(top["root"], &root); err != nil {
		t.Fatal(err)
	}
	if string(root["unknownNode"]) != `{"kept": 1.00}` {
		t.Fatalf("node unknown metadata changed: %s", root["unknownNode"])
	}
}

func TestMindmapFieldRealAdapterDiskSaveAndReopen(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "fields.mindmap")
	if err := os.WriteFile(path, []byte(mindmapFieldsFixture), 0600); err != nil {
		t.Fatal(err)
	}
	const exactNote = "中文🙂\r\n第二行  "

	a := newTestApp(t, filepath.Join(root, "config-a"))
	a.Open(path)
	tab := a.Current()
	if tab == nil || tab.Adapter == nil || tab.Adapter.ID != "mindmap" || tab.Mode != "preview" {
		t.Fatalf("real mindmap adapter did not open: %#v", tab)
	}
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	for _, label := range []string{
		"字段：主题文本", "字段：纯文本备注", "字段：本地链接",
	} {
		fieldControl(t, tester, label)
	}
	beginGraphFieldComposition(t, tester, "字段：纯文本备注", "节点备注（纯文本）", exactNote)
	if strings.Contains(tab.Doc.Buffer.String(), "中文🙂") {
		t.Fatal("pending note composition reached the source before Save")
	}
	if !a.Save(tab, false) {
		t.Fatalf("field save failed: %s", a.Message)
	}
	if tab.Mode != "preview" || tab.Doc.Dirty() {
		t.Fatalf("field save changed mode or dirty state: mode=%q dirty=%v", tab.Mode, tab.Doc.Dirty())
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	parsed := parseMindmapBytes(t, saved)
	note, present, err := parsed.Field("root", mindmap.FieldNote)
	if err != nil || !present || note != exactNote {
		t.Fatalf("saved note = %q present=%v err=%v", note, present, err)
	}
	requireUnknownMindmapMetadata(t, saved)

	reopened := newTestApp(t, filepath.Join(root, "config-b"))
	reopened.Open(path)
	reopenedTab := reopened.Current()
	reopenedTester := ui.NewTester(reopened.View, 1180, 780)
	reopenedTester.Frame()
	clickFieldControl(t, reopenedTester, "字段：纯文本备注")
	if err := reopenedTester.Click("节点备注（纯文本）"); err != nil {
		t.Fatalf("reopened note editor is missing: %v", err)
	}
	reopenedTester.Command("selectAll")
	reopenedTester.Command("copy")
	reopenedTester.Frame()
	if copied := reopenedTester.Clipboard(); copied != exactNote {
		t.Fatalf("reopened note editor clipboard = %q, want %q", copied, exactNote)
	}
	beforeUnchangedSave, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reopened.Save(reopenedTab, false) {
		t.Fatalf("clean reopened field save failed: %s", reopened.Message)
	}
	afterUnchangedSave, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(beforeUnchangedSave, afterUnchangedSave) {
		t.Fatal("clean field selection and save reserialized the reopened file")
	}
}

func TestMindmapPendingLinkWriteOpenSurvivesSynchronousTabRelease(t *testing.T) {
	root := t.TempDir()
	graphPath := filepath.Join(root, "graph.mindmap")
	if err := os.WriteFile(graphPath, []byte(mindmapFieldsFixture), 0600); err != nil {
		t.Fatal(err)
	}
	targetDir := filepath.Join(root, "子 目录")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		t.Fatal(err)
	}
	targetPath := filepath.Join(targetDir, "目标 文件.md")
	if err := os.WriteFile(targetPath, []byte("# 打开成功\n"), 0600); err != nil {
		t.Fatal(err)
	}

	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(graphPath)
	graphTab := a.Current()
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	const relativeTarget = "子 目录/目标 文件.md"
	beginGraphFieldComposition(t, tester, "字段：本地链接", "节点链接（本地）", relativeTarget)
	if _, ok := tester.Find("打开链接"); ok {
		t.Fatal("pending link draft incorrectly retained the ordinary open action")
	}
	if _, ok := tester.Find("写回并打开链接"); !ok {
		t.Fatalf("pending write-and-open action is missing: %#v", tester.Texts())
	}
	if err := tester.Click("写回并打开链接"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()

	if len(a.Tabs) != 2 || a.Active != 1 || a.Current() == graphTab {
		t.Fatalf("synchronous link open did not switch exactly once: active=%d tabs=%d", a.Active, len(a.Tabs))
	}
	if len(a.Tabs) > 1 && canonical(a.Tabs[1].Doc.Path) != canonical(targetPath) {
		t.Fatalf("opened destination = %q, want %q", a.Tabs[1].Doc.Path, targetPath)
	}
	parsed := parseMindmapBytes(t, []byte(graphTab.Doc.Buffer.String()))
	link, present, err := parsed.Field("root", mindmap.FieldHyperLink)
	if err != nil || !present || link != relativeTarget {
		t.Fatalf("committed link = %q present=%v err=%v", link, present, err)
	}
	requireUnknownMindmapMetadata(t, []byte(graphTab.Doc.Buffer.String()))

	if !a.Select(0) {
		t.Fatalf("returning to graph failed: %s", a.Message)
	}
	if !a.Save(graphTab, false) {
		t.Fatalf("saving graph after synchronous release failed: %s", a.Message)
	}
	saved, err := os.ReadFile(graphPath)
	if err != nil {
		t.Fatal(err)
	}
	parsed = parseMindmapBytes(t, saved)
	link, present, err = parsed.Field("root", mindmap.FieldHyperLink)
	if err != nil || !present || link != relativeTarget {
		t.Fatalf("saved link = %q present=%v err=%v", link, present, err)
	}
	requireUnknownMindmapMetadata(t, saved)
}

func TestMindmapPendingNoteAcrossModeTabAndClose(t *testing.T) {
	const exactNote = "生命周期🙂\r\n备注  "
	for _, action := range []string{"mode", "tab", "close"} {
		t.Run(action, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "graph.mindmap")
			if err := os.WriteFile(path, []byte(mindmapFieldsFixture), 0600); err != nil {
				t.Fatal(err)
			}
			a := newTestApp(t, filepath.Join(root, "config"))
			a.Open(path)
			tab := a.Current()
			tester := ui.NewTester(a.View, 1180, 780)
			tester.Frame()
			beginGraphFieldComposition(t, tester, "字段：纯文本备注", "节点备注（纯文本）", exactNote)
			if strings.Contains(tab.Doc.Buffer.String(), "生命周期🙂") {
				t.Fatal("pending note composition reached source before the lifecycle action")
			}

			switch action {
			case "mode":
				if !a.setMode(tab, "source") {
					t.Fatalf("source-mode barrier failed: %s", a.Message)
				}
				if tab.Mode != "source" || len(a.Tabs) != 1 || a.Current() != tab {
					t.Fatalf("source-mode result = %q active=%d tabs=%d", tab.Mode, a.Active, len(a.Tabs))
				}
			case "tab":
				a.NewDocument("text")
				if len(a.Tabs) != 2 || a.Active != 1 || a.Current() == tab || tab.Mode != "preview" {
					t.Fatalf("tab barrier result = active=%d tabs=%d graphMode=%q", a.Active, len(a.Tabs), tab.Mode)
				}
			case "close":
				a.closeDialog = func(*Tab) (closeDecision, error) { return closeDiscard, nil }
				if !a.CloseTab(0) {
					t.Fatalf("close barrier failed: %s", a.Message)
				}
				if len(a.Tabs) != 0 || a.Active != -1 {
					t.Fatalf("close barrier result = active=%d tabs=%d", a.Active, len(a.Tabs))
				}
			}

			parsed := parseMindmapBytes(t, []byte(tab.Doc.Buffer.String()))
			note, present, err := parsed.Field("root", mindmap.FieldNote)
			if err != nil || !present || note != exactNote {
				t.Fatalf("lifecycle note = %q present=%v err=%v", note, present, err)
			}
			if depth := tab.Doc.Buffer.History().UndoDepth; depth != 1 {
				t.Fatalf("pending note committed %d transactions, want 1", depth)
			}
			requireUnknownMindmapMetadata(t, []byte(tab.Doc.Buffer.String()))

			if action == "mode" {
				if !a.setMode(tab, "preview") {
					t.Fatalf("return to preview failed: %s", a.Message)
				}
				if tab.Mode != "preview" || a.Current() != tab {
					t.Fatalf("returning preview state = mode=%q active=%d", tab.Mode, a.Active)
				}
			}
			if action == "tab" && !a.Select(0) {
				t.Fatalf("return to graph tab failed: %s", a.Message)
			}
			if action == "tab" && (a.Current() != tab || tab.Mode != "preview") {
				t.Fatalf("returned graph tab state = mode=%q active=%d", tab.Mode, a.Active)
			}
		})
	}
}

func TestMindmapMissingAndRemoteLinksPreserveSource(t *testing.T) {
	tests := []struct {
		name      string
		link      string
		wantError string
	}{
		{name: "missing local", link: "missing.md", wantError: "missing.md"},
		{name: "remote", link: "https://example.com/never", wantError: "不会打开网络地址"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "graph.mindmap")
			source := strings.Replace(mindmapFieldsFixture, `"hyperLink": "target.md"`, `"hyperLink": "`+test.link+`"`, 1)
			if err := os.WriteFile(path, []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
			a := newTestApp(t, filepath.Join(root, "config"))
			a.Open(path)
			tab := a.Current()
			tester := ui.NewTester(a.View, 1180, 780)
			tester.Frame()
			clickFieldControl(t, tester, "字段：本地链接")
			before := tab.Doc.Buffer.String()
			history := tab.Doc.Buffer.Version()
			if err := tester.Click("打开链接"); err != nil {
				t.Fatalf("clear link action is missing: %v", err)
			}
			tester.Frame()
			if len(a.Tabs) != 1 || a.Current() != tab || tab.Doc.Buffer.String() != before ||
				tab.Doc.Buffer.Version() != history || tab.Doc.Dirty() {
				t.Fatalf("refused link mutated source/tab identity: tabs=%d active=%d bufferChanged=%v version=%d dirty=%v",
					len(a.Tabs), a.Active, tab.Doc.Buffer.String() != before, tab.Doc.Buffer.Version(), tab.Doc.Dirty())
			}
			if !strings.Contains(a.Message, test.wantError) {
				t.Fatalf("refusal message = %q, want %q", a.Message, test.wantError)
			}
		})
	}
}

func TestMindmapReadonlySaveAsCreatesEditableFieldView(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "readonly.mindmap")
	destination := filepath.Join(root, "writable.mindmap")
	if err := os.WriteFile(source, []byte(mindmapFieldsFixture), 0600); err != nil {
		t.Fatal(err)
	}
	doc, err := document.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	doc.ReadOnly = true
	a := newTestApp(t, filepath.Join(root, "config"))
	a.add(doc, "readonly.mindmap", "")
	tab := a.Current()
	if tab == nil || tab.Adapter == nil || tab.Adapter.ID != "mindmap" || tab.Mode != "preview" ||
		!tab.Doc.ReadOnly || !tab.Editor.ReadOnly {
		t.Fatalf("readonly graph was not configured before its preview view: %#v", tab)
	}
	runner := newControlledExporter()
	a.exporter = runner
	tester := ui.NewTester(a.View, 1180, 780)
	call, job := awaitExportCall(t, runner), a.previewJob
	call.response <- previewSuccess()
	awaitPreviewDone(t, job)
	tester.Frame()
	if !tester.HasText("此视图只读；节点属性、标签与图片引用请在源码中查看。") {
		t.Fatal("readonly scene limitation missing")
	}
	if err := tester.Click("只读思维导图画布"); err != nil {
		t.Fatal(err)
	}
	tester.Command("selectAll")
	tester.Type("不应写入")
	if tab.Doc.Buffer.String() != mindmapFieldsFixture {
		t.Fatal("readonly scene accepted direct input")
	}

	a.saveDialog = func(*Tab) (string, error) { return destination, nil }
	if !a.Save(tab, false) {
		t.Fatalf("readonly Save As failed: %s", a.Message)
	}
	if tab.Doc.ReadOnly || tab.Editor.ReadOnly || tab.Mode != "preview" ||
		canonical(tab.Doc.Path) != canonical(destination) {
		t.Fatalf("writable copy state is wrong: readonly=%v editorReadonly=%v mode=%q path=%q",
			tab.Doc.ReadOnly, tab.Editor.ReadOnly, tab.Mode, tab.Doc.Path)
	}

	tester.Frame()
	beginGraphFieldComposition(t, tester, "字段：纯文本备注", "节点备注（纯文本）", "复制后🙂")
	if !a.Save(tab, false) {
		t.Fatalf("field save on writable copy failed: %s", a.Message)
	}
	saved, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	parsed := parseMindmapBytes(t, saved)
	note, present, err := parsed.Field("root", mindmap.FieldNote)
	if err != nil || !present || note != "复制后🙂" {
		t.Fatalf("copy note = %q present=%v err=%v", note, present, err)
	}
	requireUnknownMindmapMetadata(t, saved)
}

func TestMindmapFieldsAppNarrowScreenshot(t *testing.T) {
	outputDir := os.Getenv("LEVNOTE_MINDMAP_FIELDS_APP_SCREENSHOT_DIR")
	if outputDir == "" {
		t.Skip("set LEVNOTE_MINDMAP_FIELDS_APP_SCREENSHOT_DIR")
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatal(err)
	}

	t.Run("normal", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "fields.mindmap")
		// Keep the normal capture free of provider metadata warnings so the
		// mindmap canvas receives its representative 760x480 height.
		normalSource := strings.Replace(mindmapFieldsFixture, ",\n  \"futureTop\": 123456789012345678901234567890", "", 1)
		normalSource = strings.Replace(normalSource, ",\n    \"unknownNode\": {\"kept\": 1.00}", "", 1)
		if err := os.WriteFile(path, []byte(normalSource), 0600); err != nil {
			t.Fatal(err)
		}
		a := newTestApp(t, filepath.Join(root, "config"))
		a.Dark = true
		a.Open(path)
		tab := a.Current()
		if tab == nil || tab.Mode != "preview" {
			t.Fatalf("normal screenshot did not open preview: %#v", tab)
		}
		tester := ui.NewTester(a.View, 760, 480)
		tester.SetDark(true)
		tester.Frame()
		clickFieldControl(t, tester, "字段：纯文本备注")
		if err := tester.Click("节点备注（纯文本）"); err != nil {
			t.Fatalf("normal note field editor is missing: %v", err)
		}
		tester.Command("selectAll")
		tester.Compose("实时备注🙂\r\n第二行", utf8.RuneCountInString("实时备注🙂\r\n第二行"))
		tester.Frame()
		for _, label := range []string{
			"源码", "阅读 / 编辑", "字段：纯文本备注", "节点备注（纯文本）",
		} {
			rect, ok := tester.Find(label)
			if !ok || rect.X < 0 || rect.Y < 0 || rect.X+rect.W > 760 || rect.Y+rect.H > 451 {
				t.Fatalf("normal narrow control %q is missing or clipped: %#v texts=%#v", label, rect, tester.Texts())
			}
		}
		captureAcceptanceTester(t, filepath.Join(outputDir, "mindmap-fields-app-normal-760x480-dark.png"), tester)
	})

	t.Run("conflict", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, "fields.mindmap")
		if err := os.WriteFile(path, []byte(mindmapFieldsFixture), 0600); err != nil {
			t.Fatal(err)
		}
		a := newTestApp(t, filepath.Join(root, "config"))
		a.Dark = true
		a.Open(path)
		tab := a.Current()
		tester := ui.NewTester(a.View, 760, 480)
		tester.SetDark(true)
		tester.Frame()
		clickFieldControl(t, tester, "字段：纯文本备注")
		if err := tester.Click("节点备注（纯文本）"); err != nil {
			t.Fatalf("conflict note field editor is missing: %v", err)
		}
		tester.Command("selectAll")
		tester.Compose("窄屏保留草稿🙂\r\n第二行", utf8.RuneCountInString("窄屏保留草稿🙂\r\n第二行"))

		raw := tab.Doc.Buffer.String()
		external := strings.Replace(raw, `"note": "原备注"`, `"note": "外部改动"`, 1)
		if external == raw {
			t.Fatal("fixture note was not replaced for conflict capture")
		}
		if err := tab.Doc.Buffer.ReplaceUTF16(0, tab.Doc.Buffer.LenUTF16(), external); err != nil {
			t.Fatal(err)
		}
		if a.flushTab(tab, "截图验收") {
			t.Fatal("conflicting note draft unexpectedly flushed")
		}
		tester.Frame()
		for _, label := range []string{"字段：纯文本备注", "节点备注（纯文本）"} {
			rect, _ := fieldControl(t, tester, label)
			if rect.X < 0 || rect.Y < 0 || rect.X+rect.W > 760 || rect.Y+rect.H > 451 {
				t.Fatalf("%q is not initially visible above the status row: %#v", label, rect)
			}
		}
		var recoveryRects [2]ui.Rect
		for attempt := 0; attempt < 8; attempt++ {
			retry, retryOK := tester.Find("重试写回")
			recover, recoverOK := tester.Find("恢复为新文档")
			if retryOK && recoverOK && retry.W > 0 && recover.W > 0 {
				recoveryRects = [2]ui.Rect{retry, recover}
				break
			}
			tester.Scroll(750, 300, 0, 120)
			tester.Frame()
		}
		for index, label := range []string{"重试写回", "恢复为新文档"} {
			rect := recoveryRects[index]
			if rect.W <= 0 || rect.H <= 0 || rect.X < 0 || rect.Y < 0 ||
				rect.X+rect.W > 760 || rect.Y+rect.H > 451 {
				t.Fatalf("%q is not reachable by bounded inspector scrolling: %#v texts=%#v",
					label, rect, tester.Texts())
			}
		}
		captureAcceptanceTester(t, filepath.Join(outputDir, "mindmap-fields-app-conflict-760x480-dark.png"), tester)
	})
}
