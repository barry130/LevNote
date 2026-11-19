package app

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/engineipc"
	"levnote.local/native/core/plugins"
	"levnote.local/native/core/text"
)

const routeMarkdown = `{"schema":3,"id":"task-notes","name":"任务笔记","extensions":["TASKMD",".taskmd",".taskmore"],"nativeRoute":"builtin.markdown","lineComment":"#"}`
const routeMindmap = `{"schema":3,"id":"thoughts","name":"想法导图","extensions":[".thoughts",".ideas"],"nativeRoute":"builtin.mindmap"}`

func routingApp(t *testing.T) (*App, string) {
	t.Helper()
	root := t.TempDir()
	a := newTestApp(t, filepath.Join(root, "config"))
	installAppPluginManifest(t, a, "task-notes.json", routeMarkdown)
	installAppPluginManifest(t, a, "thoughts.json", routeMindmap)
	return a, root
}

func routingFile(t *testing.T, root, name, content string) (string, []byte) {
	t.Helper()
	data := append([]byte{0xef, 0xbb, 0xbf}, []byte(content)...)
	path := filepath.Join(root, name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path, data
}

func requireRoute(t *testing.T, tab *Tab, pluginID, adapterID string) {
	t.Helper()
	if tab == nil || tab.Adapter == nil || tab.Adapter.ID != adapterID || tab.MatchedPlugin.ID != pluginID || !tab.MatchedPlugin.Declarative || tab.OutlineFactory != nil || tab.Adapter.Outline == nil || tab.Editor.Highlight == nil {
		t.Fatalf("configuration/native adapter identities: %+v", tab)
	}
	if strings.Contains(tab.SourceCapabilityDetail, "schema2") || !strings.HasPrefix(tab.SourceCapabilityBadge, "使用 ") {
		t.Fatal("schema3 mislabeled", tab.SourceCapabilityBadge, tab.SourceCapabilityDetail)
	}
}

func TestFormatRoutingNativeModesOutlineRoundTripAndSaveAs(t *testing.T) {
	for _, fixture := range []struct{ name, pluginID, adapterID, alias, content, edit, replacement string }{
		{"note.taskmd", "task-notes", "markdown", "copy.taskmore", "# 任务🙂\r\n\r\n正文\r\n", "正文", "编辑后的正文🙂"},
		{"map.thoughts", "thoughts", "mindmap", "copy.ideas", strings.ReplaceAll(mindmapFieldsFixture, "\n", "\r\n"), "中心", "中心已改🙂"},
	} {
		t.Run(fixture.adapterID, func(t *testing.T) {
			a, root := routingApp(t)
			path, disk := routingFile(t, root, fixture.name, fixture.content)
			a.Open(path)
			tab := a.Current()
			requireRoute(t, tab, fixture.pluginID, fixture.adapterID)
			if tab.Mode != "preview" || tab.Doc.Encoding != document.UTF8 || !tab.Doc.BOM || tab.Doc.Dirty() {
				t.Fatal("initial route/encoding", tab.Mode)
			}
			if tab.Adapter.ExportEngine == nil || len(tab.Adapter.Exports) != 1 {
				t.Fatal("native export absent")
			}
			if fixture.adapterID == "markdown" {
				if tab.Adapter.ExportEngine.ID != "levnote.markdown-export" || tab.Adapter.Exports[0] != "html" || tab.Adapter.SceneEngine != nil {
					t.Fatal("Markdown capability")
				}
				spans := tab.Editor.Highlight("# configured comment")
				if len(spans) != 1 || !spans[0].Italic || spans[0].Text != "# configured comment" {
					t.Fatal("alias skipped configured highlight", spans)
				}
			} else if tab.Adapter.ExportEngine.ID != "levnote.mindmap-svg" || tab.Adapter.SceneEngine == nil || tab.Adapter.SceneEngine.ID != "levnote.mindmap-preview" {
				t.Fatal("Mindmap capability")
			}
			a.ShowOutline = true
			tester := ui.NewTester(a.View, 1180, 780)
			for _, mode := range []string{"preview", "source", "split"} {
				if !a.setMode(tab, mode) {
					t.Fatal(a.Message)
				}
				tester.Frame()
				if a.OutlineError != "" || len(a.OutlineItems) != 1 {
					t.Fatal("native outline missing", a.OutlineError, a.OutlineItems)
				}
				item := a.OutlineItems[0]
				if fixture.adapterID == "markdown" && (item.Text != "任务🙂" || item.AnchorID != "任务" || item.Offset != 0) {
					t.Fatal("Markdown semantic outline", item)
				}
				if fixture.adapterID == "mindmap" && (item.ID != "root" || item.Text != "中心" || item.Offset != -1) {
					t.Fatal("Mindmap semantic outline", item)
				}
			}
			if !a.Save(tab, false) {
				t.Fatal(a.Message)
			}
			if saved, err := os.ReadFile(path); err != nil || !bytes.Equal(saved, disk) {
				t.Fatal("no-op save changed bytes", err)
			}
			if !a.setMode(tab, "source") {
				t.Fatal(a.Message)
			}
			start := sourceOffset(t, fixture.content, fixture.edit)
			tab.Editor.ReplaceText(&ui.TextInputRange{Start: start, End: start + text.UTF16Len(fixture.edit)}, fixture.replacement)
			want := strings.Replace(fixture.content, fixture.edit, fixture.replacement, 1)
			if tab.Doc.Buffer.String() != want || !tab.Doc.Dirty() || tab.Doc.Buffer.History().UndoDepth != 1 {
				t.Fatal("source edit not one transaction")
			}
			if !tab.Doc.Buffer.Undo() || tab.Doc.Buffer.String() != fixture.content || !tab.Doc.Buffer.Redo() || tab.Doc.Buffer.String() != want {
				t.Fatal("alias undo/redo")
			}
			copyPath := filepath.Join(root, fixture.alias)
			a.saveDialog = func(*Tab) (string, error) { return copyPath, nil }
			if !a.Save(tab, true) {
				t.Fatal(a.Message)
			}
			requireRoute(t, tab, fixture.pluginID, fixture.adapterID)
			saved, err := os.ReadFile(copyPath)
			if err != nil || !bytes.Equal(saved, append([]byte{0xef, 0xbb, 0xbf}, []byte(want)...)) {
				t.Fatal("SaveAs lost BOM/CRLF/content", err)
			}
			if fixture.adapterID == "mindmap" {
				requireUnknownMindmapMetadata(t, saved[3:])
			}
			if original, _ := os.ReadFile(path); !bytes.Equal(original, disk) {
				t.Fatal("SaveAs overwrote source")
			}
			b := newTestApp(t, a.configDir)
			b.Open(copyPath)
			requireRoute(t, b.Current(), fixture.pluginID, fixture.adapterID)
			if b.Current().Doc.Buffer.String() != want || !b.Current().Doc.BOM {
				t.Fatal("reopen differs")
			}
			// Path rematching is deliberately not format conversion. JSON stays
			// JSON even when a Mindmap alias is explicitly saved as .md.
			mdPath := filepath.Join(root, "explicit-path-change.md")
			a.saveDialog = func(*Tab) (string, error) { return mdPath, nil }
			if !a.Save(tab, true) || tab.MatchedPlugin.ID != "markdown" || tab.Adapter.ID != "markdown" || tab.Doc.Buffer.String() != want {
				t.Fatal("path rematch converted content", a.Message)
			}
			if final, _ := os.ReadFile(mdPath); !bytes.Equal(final, saved) {
				t.Fatal("path rematch changed disk bytes")
			}
		})
	}
}

func TestFormatRoutingCreationFallbackRecoveryAndReferences(t *testing.T) {
	a, root := routingApp(t)
	for _, id := range []string{"markdown", "mindmap"} {
		a.setPluginEnabled(id, false)
	}
	if !a.NewFromPlugin("task-notes") {
		t.Fatal(a.Message)
	}
	first := a.Current()
	requireRoute(t, first, "task-notes", "markdown")
	if first.Name != "未命名.taskmd" || first.Doc.Buffer.String() != "# 新文档\n\n" {
		t.Fatal("normalized suffix/template", first.Name, first.Doc.Buffer.String())
	}
	if !a.NewFromPlugin("thoughts") {
		t.Fatal(a.Message)
	}
	requireRoute(t, a.Current(), "thoughts", "mindmap")
	parsed := parseMindmapBytes(t, []byte(a.Current().Doc.Buffer.String()))
	if parsed.Root.ID != "root" || parsed.Root.Topic != "中心主题" {
		t.Fatal("native template changed")
	}
	a.setPluginEnabled("thoughts", false)
	if a.Current().Adapter != nil || a.Current().Mode != "source" || a.Current().FormatWarning == "" {
		t.Fatal("disabled route did not visibly fall back")
	}
	before := len(a.Tabs)
	if a.NewFromPlugin("thoughts") || len(a.Tabs) != before {
		t.Fatal("disabled configuration created tab")
	}
	a.setPluginEnabled("thoughts", true)
	requireRoute(t, a.Current(), "thoughts", "mindmap")
	// A first suffix occupied by a builtin is not silently skipped for .free.
	a.setPluginEnabled("markdown", true)
	installAppPluginManifest(t, a, "shadow.json", `{"schema":3,"id":"shadow","name":"Shadow","extensions":[".md",".free"],"nativeRoute":"builtin.markdown"}`)
	if a.NewFromPlugin("shadow") || len(a.Tabs) != before || !strings.Contains(a.Message, "首个扩展名 .md") {
		t.Fatal("shadowed creation accepted", a.Message)
	}
	if err := a.recoverDraft("saved.taskmd", "# 恢复\r\n"); err != nil {
		t.Fatal(err)
	}
	requireRoute(t, a.Current(), "task-notes", "markdown")
	if a.Current().Name != "恢复-saved.taskmd" || a.Current().Doc.Buffer.String() != "# 恢复\r\n" {
		t.Fatal("recovered alias")
	}
	a.Persist()
	restored := newTestApp(t, a.configDir)
	restored.Restore()
	requireRoute(t, restored.Current(), "task-notes", "markdown")
	if restored.Current().Doc.Buffer.String() != "# 恢复\r\n" {
		t.Fatal("session lost recovered alias")
	}
	sourcePath, _ := routingFile(t, root, "source.taskmd", "# 来源\r\n")
	targetPath, _ := routingFile(t, root, "目标.taskmore", "# 目标\r\n\r\n段落\r\n")
	a.Open(sourcePath)
	a.editorHost(a.Current()).OpenReference("目标.taskmore")
	if a.Current().Doc.Path != targetPath {
		t.Fatal("alias local reference did not open target")
	}
	requireRoute(t, a.Current(), "task-notes", "markdown")
	a.setMode(a.Current(), "source")
	a.editorHost(a.Current()).OpenReference("#目标")
	if !strings.Contains(a.Message, "目标") || a.Current().Doc.Buffer.Selection().Start() != 0 {
		t.Fatal("native anchor through alias", a.Message)
	}
	// Capability absence is a host composition problem, not another format.
	a.adapters = nil
	a.reconfigureTabs()
	if a.Current().Adapter != nil || !strings.Contains(a.Current().FormatWarning, "原生格式不可用") {
		t.Fatal("missing capability silently rerouted")
	}
}

func TestFormatRoutingReadonlySceneSaveAsAndDisabledFallback(t *testing.T) {
	a, root := routingApp(t)
	path, disk := routingFile(t, root, "readonly.thoughts", strings.ReplaceAll(mindmapFieldsFixture, "\n", "\r\n"))
	if err := os.Chmod(path, 0o400); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o600) })
	a.Open(path)
	tab := a.Current()
	if !tab.Doc.ReadOnly || !tab.Editor.ReadOnly {
		t.Skip("filesystem did not expose readonly fixture")
	}
	f := newControlledExporter()
	a.exporter = f
	tester, call, job := mountPreview(t, a, f, 760, 480)
	var snapshot bytes.Buffer
	if _, err := call.source.Data.WriteTo(&snapshot); err != nil || snapshot.String() != string(disk[3:]) {
		t.Fatal("alias scene snapshot", err)
	}
	call.response <- previewSuccess()
	awaitPreviewDone(t, job)
	tester.Frame()
	if !tester.HasText("只读场景") || !tester.HasText("此视图只读；节点属性、标签与图片引用请在源码中查看。") {
		t.Fatal("alias readonly scene unavailable", tester.Texts())
	}
	a.setPluginEnabled("thoughts", false)
	if tab.Adapter != nil || tab.Mode != "source" || !strings.Contains(tab.FormatWarning, "另存为可写副本") || strings.Contains(tab.FormatWarning, "仍可编辑") {
		t.Fatal("readonly fallback overclaims", tab.FormatWarning)
	}
	a.setPluginEnabled("thoughts", true)
	copyPath := filepath.Join(root, "writable.ideas")
	a.saveDialog = func(*Tab) (string, error) { return copyPath, nil }
	if !a.Save(tab, true) || tab.Doc.ReadOnly || tab.Editor.ReadOnly {
		t.Fatal("readonly copy failed", a.Message)
	}
	requireRoute(t, tab, "thoughts", "mindmap")
	if original, _ := os.ReadFile(path); !bytes.Equal(original, disk) {
		t.Fatal("readonly original changed")
	}
	if copyBytes, _ := os.ReadFile(copyPath); !bytes.Equal(copyBytes, disk) {
		t.Fatal("readonly copy lost exact BOM/CRLF")
	}
	noEngineCall(t, f)
}

type routingFlushView struct {
	failingView
	onFlush func() error
}

func (v *routingFlushView) Flush() error {
	v.flushes++
	if v.onFlush != nil {
		return v.onFlush()
	}
	return nil
}

func TestFormatRoutingNewFlushReceiptAndInstallModalRecheck(t *testing.T) {
	for _, action := range []string{"conflict", "recovery", "replacement", "route-change", "success-edit"} {
		t.Run(action, func(t *testing.T) {
			a, _ := routingApp(t)
			a.NewFromPlugin("task-notes")
			tab := a.Current()
			before := tab.Doc.Buffer.String()
			selection := tab.Doc.Buffer.Selection()
			view := &routingFlushView{}
			tab.View = view
			var replacement *failingView
			view.onFlush = func() error {
				switch action {
				case "conflict":
					return errors.New("pending cell conflict")
				case "recovery":
					return a.recoverDraft("pending.taskmd", "# recovered")
				case "replacement":
					replacement = &failingView{}
					tab.View = replacement
				case "route-change":
					return a.pluginManager.SetEnabled("task-notes", false)
				case "success-edit":
					return tab.Doc.Buffer.ReplaceUTF16(tab.Doc.Buffer.LenUTF16(), tab.Doc.Buffer.LenUTF16(), "committed")
				}
				return nil
			}
			ok := a.NewFromPlugin("task-notes")
			if action == "success-edit" {
				if !ok || len(a.Tabs) != 2 || tab.Doc.Buffer.String() != before+"committed" || tab.Doc.Buffer.History().UndoDepth != 1 || view.releases != 1 {
					t.Fatal("own flush not preserved", a.Message)
				}
				if !tab.Doc.Buffer.Undo() || tab.Doc.Buffer.String() != before {
					t.Fatal("own flush not undoable")
				}
			} else {
				wantTabs := 1
				if action == "recovery" {
					wantTabs = 2
				}
				if ok || len(a.Tabs) != wantTabs || view.releases != 0 || tab.Doc.Buffer.String() != before || tab.Doc.Buffer.Selection() != selection {
					t.Fatal("stale/failed flush appended/released", action, a.Message)
				}
				if replacement != nil && (tab.View != replacement || replacement.releases != 0) {
					t.Fatal("replacement overwritten")
				}
				if action == "recovery" && a.Current().Doc.Buffer.String() != "# recovered" {
					t.Fatal("recovery tab lost")
				}
			}
		})
	}
	t.Run("post-modal-new-draft", func(t *testing.T) {
		a, root := routingApp(t)
		a.NewDocument("markdown")
		tab := a.Current()
		view := &failingView{}
		tab.View = view
		path := filepath.Join(root, "new.json")
		if err := os.WriteFile(path, []byte(`{"schema":3,"id":"new-route","name":"New","extensions":[".newroute"],"nativeRoute":"builtin.markdown"}`), 0o600); err != nil {
			t.Fatal(err)
		}
		a.openDialog = func(mygo.OpenDialogOptions) ([]string, error) {
			view.flushErr = errors.New("draft created inside modal")
			return []string{path}, nil
		}
		a.installPlugin()
		if view.flushes != 2 || view.releases != 0 || tab.View != view || a.pluginManager.Enabled("new-route") {
			t.Fatal("post-modal draft bypassed", a.Message)
		}
		if _, err := os.Stat(filepath.Join(a.configDir, "plugins", "new.json")); !os.IsNotExist(err) {
			t.Fatal("refused install published", err)
		}
	})
}

func TestFormatRoutingRealPendingDraftBlocksNewDisableAndInstall(t *testing.T) {
	a, root := routingApp(t)
	path, _ := routingFile(t, root, "draft.thoughts", mindmapTagsCleanBody)
	a.Open(path)
	tab := a.Current()
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	openTagsAppPanel(t, tester, 1180, 780)
	typeTagAppInput(t, tester, 1180, 780, "未提交🙂", false)
	before, history, view := tab.Doc.Buffer.String(), tab.Doc.Buffer.History(), tab.View
	if a.NewFromPlugin("task-notes") || len(a.Tabs) != 1 {
		t.Fatal("pending tag allowed new route")
	}
	a.setPluginEnabled("thoughts", false)
	dialogs := 0
	a.openDialog = func(mygo.OpenDialogOptions) ([]string, error) { dialogs++; return nil, nil }
	a.installPlugin()
	if !a.pluginManager.Enabled("thoughts") || dialogs != 0 || tab.View != view || tab.Doc.Buffer.String() != before || tab.Doc.Buffer.History() != history {
		t.Fatal("pending native draft lost", a.Message)
	}
	requireTagAppInputText(t, tester, 1180, 780, mindmapTagsInput, "未提交🙂")
}

func TestFormatRoutingSaveAsPostDialogNativeDraftAndOwner(t *testing.T) {
	for _, action := range []string{"native-note", "native-tag-refusal", "replacement", "closed", "recovery", "reconfigure"} {
		t.Run(action, func(t *testing.T) {
			a, root := routingApp(t)
			path, disk := routingFile(t, root, "dialog.thoughts", mindmapFieldsFixture)
			a.Open(path)
			tab := a.Current()
			tester := ui.NewTester(a.View, 1180, 780)
			tester.Frame()
			originalView := tab.View
			output := filepath.Join(root, "after-dialog.ideas")
			var replacement *failingView
			a.saveDialog = func(*Tab) (string, error) {
				switch action {
				case "native-note":
					beginGraphFieldComposition(t, tester, "字段：纯文本备注", "节点备注（纯文本）", "弹窗期间🙂\r\n新备注")
				case "native-tag-refusal":
					openTagsAppPanel(t, tester, 1180, 780)
					typeTagAppInput(t, tester, 1180, 780, "弹窗期间未提交", false)
				case "replacement":
					replacement = &failingView{flushErr: errors.New("replacement draft")}
					tab.View = replacement
				case "closed":
					a.removeTab(0)
				case "recovery":
					if err := a.recoverDraft("new.taskmd", "# recovered while choosing"); err != nil {
						t.Fatal(err)
					}
				case "reconfigure":
					a.refreshPlugins()
				}
				return output, nil
			}
			ok := a.Save(tab, true)
			if action == "native-note" {
				if !ok || tab.Doc.Path != output || tab.Doc.Buffer.History().UndoDepth != 1 {
					t.Fatal("post-dialog native draft not committed", a.Message)
				}
				saved, err := os.ReadFile(output)
				if err != nil || !bytes.HasPrefix(saved, []byte{0xef, 0xbb, 0xbf}) {
					t.Fatal("save output", err)
				}
				parsed := parseMindmapBytes(t, saved[3:])
				if parsed.Root.Note == nil || *parsed.Root.Note != "弹窗期间🙂\r\n新备注" {
					t.Fatal("post-dialog note missing", parsed.Root.Note)
				}
				requireUnknownMindmapMetadata(t, saved[3:])
				if !tab.Doc.Buffer.Undo() || tab.Doc.Buffer.String() != mindmapFieldsFixture {
					t.Fatal("post-dialog write not separately undoable")
				}
			} else {
				if ok || tab.Doc.Path != path || tab.Doc.Buffer.String() != mindmapFieldsFixture || tab.Doc.Buffer.CanUndo() {
					t.Fatal("stale/failed dialog wrote source", action, a.Message)
				}
				if _, err := os.Stat(output); !os.IsNotExist(err) {
					t.Fatal("refused SaveAs published", err)
				}
				if action == "native-tag-refusal" {
					if tab.View != originalView {
						t.Fatal("refused draft released")
					}
					requireTagAppInputText(t, tester, 1180, 780, mindmapTagsInput, "弹窗期间未提交")
				}
				if replacement != nil && (tab.View != replacement || replacement.releases != 0 || replacement.flushes != 0) {
					t.Fatal("replacement touched")
				}
			}
			if original, _ := os.ReadFile(path); !bytes.Equal(original, disk) {
				t.Fatal("original changed")
			}
		})
	}
}

func TestFormatRoutingPostSaveLoadFailureReconcilesButPersistenceFailureRetains(t *testing.T) {
	for _, problem := range []string{"removed", "corrupt", "settings"} {
		t.Run(problem, func(t *testing.T) {
			a, _ := routingApp(t)
			a.NewFromPlugin("task-notes")
			tab := a.Current()
			view := &failingView{}
			tab.View = view
			generation := tab.OutlineFactoryGeneration
			path := filepath.Join(a.configDir, "plugins", "task-notes.json")
			switch problem {
			case "removed":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			case "corrupt":
				if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "settings":
				if err := os.MkdirAll(plugins.SettingsPath(a.configDir), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			a.setPluginEnabled("task-notes", true)
			if problem == "settings" {
				if tab.View != view || view.releases != 0 || tab.OutlineFactoryGeneration != generation || !a.pluginManager.Enabled("task-notes") {
					t.Fatal("persistence failure replaced view", a.Message)
				}
			} else {
				if tab.View != nil || view.releases != 1 || tab.Adapter != nil || tab.Mode != "source" || tab.OutlineFactoryGeneration <= generation || a.pluginManager.Enabled("task-notes") {
					t.Fatal("post-save load failure retained obsolete route", a.Message)
				}
				if !strings.Contains(a.Message, plugins.ErrEnabledNotLoaded.Error()) {
					t.Fatal("post-save cause hidden", a.Message)
				}
				if _, err := os.Stat(plugins.SettingsPath(a.configDir)); err != nil {
					t.Fatal("fixture didn't persist first", err)
				}
			}
		})
	}
}

func TestFormatRoutingSameIDRefreshRejectsLateSceneAndExport(t *testing.T) {
	t.Run("scene", func(t *testing.T) {
		a, root := routingApp(t)
		path, _ := routingFile(t, root, "late.thoughts", mindmapFieldsFixture)
		a.Open(path)
		tab := a.Current()
		tab.Doc.ReadOnly, tab.Editor.ReadOnly = true, true
		f := newControlledExporter()
		controlledPreviewJoin(a, f)
		tester, call, oldJob := mountPreview(t, a, f, 760, 480)
		oldView, oldAdapter := tab.View, tab.Adapter
		oldGeneration := tab.OutlineFactoryGeneration
		a.refreshPlugins()
		if tab.View != nil || tab.Adapter == oldAdapter || tab.OutlineFactoryGeneration <= oldGeneration || call.ctx.Err() == nil {
			t.Fatal("refresh didn't cancel old scene")
		}
		call.response <- previewSuccess()
		awaitPreviewDone(t, oldJob)
		a.deliverScene()
		oldJob.mu.Lock()
		retained := len(oldJob.result.Scene.Nodes)
		callback := oldJob.complete
		oldJob.mu.Unlock()
		if retained != 0 || callback != nil {
			t.Fatal("late scene retained owner/result")
		}
		tester.Frame()
		next := awaitExportCall(t, f)
		newJob := a.previewJob
		if newJob == nil || tab.View == oldView || next.id != engineipc.MindmapPreview {
			t.Fatal("same-ID new scene didn't start")
		}
		next.response <- previewSuccess()
		awaitPreviewDone(t, newJob)
		tester.Frame()
		if !tester.HasText("只读场景") {
			t.Fatal("fresh generation scene missing")
		}
	})
	t.Run("export", func(t *testing.T) {
		a, root := routingApp(t)
		path, _ := routingFile(t, root, "late.taskmd", "# Old\r\n")
		a.Open(path)
		tab := a.Current()
		f := newControlledExporter()
		controlledPreviewJoin(a, f)
		queue := make(chan func(), 8)
		a.uiDispatch = func(fn func()) { queue <- fn }
		dialogs := 0
		output := filepath.Join(root, "must-not.html")
		a.exportDialog = func(mygo.SaveDialogOptions) (string, error) { dialogs++; return output, nil }
		a.exportCurrent()
		call := awaitExportCall(t, f)
		job := a.exportJob
		if call.id != engineipc.MarkdownHTML {
			t.Fatal("route export descriptor", call.id)
		}
		var snapshot bytes.Buffer
		_, _ = call.source.Data.WriteTo(&snapshot)
		if snapshot.String() != "# Old\r\n" {
			t.Fatal("alias export snapshot")
		}
		a.refreshPlugins()
		if call.ctx.Err() == nil {
			t.Fatal("refresh left export active")
		}
		call.response <- exportSuccess()
		awaitExportDone(t, job)
		drainOneExportCallback(t, queue)
		if dialogs != 0 || a.exportJob != nil {
			t.Fatal("late export opened dialog")
		}
		if _, err := os.Stat(output); !os.IsNotExist(err) {
			t.Fatal("late export published", err)
		}
		if tab.Adapter == job.adapter {
			t.Fatal("same-ID adapter identity reused")
		}
	})
}

func routingButton(t *testing.T, tester *ui.Tester, label string) ui.AccessInfo {
	t.Helper()
	for attempt := 0; attempt < 60; attempt++ {
		info, ok := tester.AccessByRole(ui.RoleButton, label)
		// The panel's 12px bottom padding and 1px border end its scroll
		// viewport before the status row, even when AccessInfo isn't Offscreen.
		if ok && !info.Offscreen && info.Bounds.H >= 26 && info.Bounds.X >= 228 && info.Bounds.X+info.Bounds.W <= 742 && info.Bounds.Y >= 170 && info.Bounds.Y+info.Bounds.H <= 438 {
			if info.Disabled || !info.Focusable {
				t.Fatal("new action inaccessible", info)
			}
			return info
		}
		tester.Wheel(700, 410, 0, 36)
		tester.Advance(time.Second)
	}
	info, _ := tester.AccessByRole(ui.RoleButton, label)
	if out := os.Getenv("LEVNOTE_FORMAT_ROUTING_CAPTURE_DIR"); out != "" {
		captureAcceptanceTester(t, filepath.Join(out, "unreachable-"+label+".png"), tester)
	}
	t.Fatal("plugin action unreachable above status", info, tester.Texts())
	return info
}

func TestFormatRoutingPluginWorkflowCaptures(t *testing.T) {
	out := os.Getenv("LEVNOTE_FORMAT_ROUTING_CAPTURE_DIR")
	if out == "" {
		t.Skip("set LEVNOTE_FORMAT_ROUTING_CAPTURE_DIR for scoped native captures")
	}
	if err := os.MkdirAll(out, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, dark := range []bool{false, true} {
		name, plugin, id, target := "light", "任务笔记", "task-notes", "Markdown"
		if dark {
			name, plugin, id, target = "dark", "想法导图", "thoughts", "思维导图"
		}
		t.Run(name, func(t *testing.T) {
			a, _ := routingApp(t)
			a.NewDocument("text")
			a.Dark = dark
			a.ShowPlugins = true
			tester := ui.NewTester(a.View, 760, 480)
			tester.Frame()
			info := routingButton(t, tester, "使用 "+plugin+" 新建文档")
			for _, text := range []string{plugin, id, "使用 " + target} {
				if rect, ok := tester.Find(text); !ok || rect.X < 228 || rect.X+rect.W > 760 || rect.Y < 90 || rect.Y+rect.H > 451 {
					t.Fatal("configuration identity/target not visible", text, rect)
				}
			}
			t.Logf("%s create control bounds=%+v", name, info.Bounds)
			captureAcceptanceTester(t, filepath.Join(out, "routing-plugins-"+name+"-760x480.png"), tester)
			tester.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
			tester.Frame()
			adapterID := "markdown"
			if dark {
				adapterID = "mindmap"
			}
			requireRoute(t, a.Current(), id, adapterID)
			if a.ShowPlugins || len(a.Tabs) != 2 {
				t.Fatal("native create action failed", a.Message)
			}
			if !dark {
				if !tester.HasText("使用 Markdown") || !tester.HasText("新文档") {
					t.Fatal("custom suffix native view not shown", tester.Texts())
				}
				captureAcceptanceTester(t, filepath.Join(out, "routing-document-taskmd-760x480.png"), tester)
			}
		})
	}
}
