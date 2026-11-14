package app

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/search"
	"levnote.local/native/core/session"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

const historyBudgetNotice = "较早或本次撤销历史因预算未保留"

// consumeHistoryNotices runs on the normal UI frame, not on inspector queries.
// Reading a budget flag must not flush input, end a group or reset core history.
func (a *App) consumeHistoryNotices() {
	for _, t := range a.Tabs {
		if t == nil {
			continue
		}
		var buffer *text.Buffer
		if t.textCapable() {
			buffer = t.Doc.Buffer
		}
		if buffer != t.historyNoticeBuffer {
			t.historyNoticeBuffer = buffer
			t.seenTruncated = false
		}
		if buffer == nil || !buffer.History().Truncated {
			// Reload/ResetHistory can keep the same buffer. Only an observed
			// false flag (or a different buffer) rearms this disclosure.
			t.seenTruncated = false
			continue
		}
		if !t.seenTruncated {
			t.seenTruncated = true
			a.report(fmt.Errorf("%s：%s", t.Name, historyBudgetNotice))
		}
	}
}

func (a *App) Restore() {
	// Startup restore is one synchronous ownership transaction. Binary IO still
	// executes on the shared lane; no nested UI callbacks can replace its tabs,
	// and partial state is never persisted while it is being assembled.
	if a.restoring || a.binaryJob != nil {
		a.report(fmt.Errorf("已有文件操作，未开始会话恢复"))
		return
	}
	a.cancelAutosave()
	a.restoring = true
	defer func() { a.restoring = false }()
	sessionPath := filepath.Join(a.configDir, "session.json")
	snapshot, issues := session.Load(sessionPath)
	for _, issue := range issues {
		a.report(fmt.Errorf("恢复会话：%s · %s", issue.Reason, issue.Detail))
	}
	a.Root = snapshot.Root
	a.Dark = snapshot.Preferences.Theme == "dark"
	if snapshot.Preferences.FontSize >= 10 {
		a.FontSize = snapshot.Preferences.FontSize
	}
	a.SoftWrap = snapshot.Preferences.WordWrap
	if snapshot.Preferences.LineNumbers != nil {
		a.LineNumbers = *snapshot.Preferences.LineNumbers
	}
	if snapshot.Preferences.SidebarWidth >= 160 {
		a.SidebarWidth = snapshot.Preferences.SidebarWidth
	}
	a.applyEditorPreferences()
	mappedActive := -1
	for index, saved := range snapshot.Tabs {
		if err := a.admitNewTab(); err != nil {
			a.report(err)
			break
		}
		binary, binaryErr := session.RestoreBinary(saved)
		if errors.Is(binaryErr, session.ErrTextDispatchRequired) && saved.Draft == nil {
			if candidate, kind, _ := probeImage(saved.Path); candidate {
				binary = session.RestoredBinary{Path: saved.Path, Kind: kind, Title: saved.Title, View: saved.View, PluginID: "image"}
				binaryErr = nil
			}
		}
		if binaryErr == nil {
			task := newBinaryTask(binaryRequest{op: "restore", path: binary.Path, kind: binary.Kind}, nil)
			lane := a.ensureBinaryLane()
			if err := lane.submit(task, true); err != nil {
				a.report(err)
				continue
			}
			<-task.done
			out, _ := task.take()
			lane.release(task, true)
			if out.err != nil {
				a.report(out.err)
				continue
			}
			name := binary.Title
			if name == "" {
				name = filepath.Base(binary.Path)
			}
			t := a.appendBinary(out, name, "")
			if binary.View == "source" {
				t.Mode = "source"
			}
			a.startWatch(t.path(), false)
			// Baseline metadata alone survives inactive restoration. The selected
			// image will be verified/reacquired on first presentation.
			out.snapshot = nil
			if index == snapshot.Active {
				mappedActive = len(a.Tabs) - 1
			}
			continue
		} else if !errors.Is(binaryErr, session.ErrTextDispatchRequired) {
			a.report(binaryErr)
			continue
		}
		restored, err := session.RestoreTab(saved)
		if err != nil {
			a.report(err)
			continue
		}
		warning := ""
		if restored.OriginalPath != "" {
			warning = "已恢复未保存草稿；原文件保留在 " + restored.OriginalPath + "。保存时请选择副本位置。"
		}
		t := a.add(restored.Document, restored.Title, warning)
		if t == nil {
			continue
		}
		if index == snapshot.Active {
			mappedActive = len(a.Tabs) - 1
		}
		t.OriginalPath = restored.OriginalPath
		t.Editor.ScrollY = restored.Scroll
		t.Mode = restored.View
		if t.Mode == "" {
			t.Mode = "source"
		}
	}
	if len(a.Tabs) > 0 {
		if mappedActive < 0 {
			mappedActive = min(snapshot.Active, len(a.Tabs)-1)
		}
		a.Select(mappedActive)
	}
	if a.Root != "" {
		a.refreshTree()
		a.startWatch(a.Root, true)
	}
	a.persistPending = false
}
func (a *App) Persist() {
	// Explicit persistence requests may retry an unchanged automatic failure.
	if a.persistTabs(a.Tabs, a.Active, "保存恢复状态") != nil {
		// The explicit attempt is the retry. Do not schedule another automatic
		// capture of the same unchanged state.
		a.persistPending = false
	}
}

func (a *App) persistTabs(tabs []*Tab, active int, action string) error {
	return a.persistTabsWithDiscard(tabs, active, nil, action)
}

func (a *App) persistQuitTabs(tabs []*Tab, active int, discarded []bool) error {
	return a.persistTabsWithDiscard(tabs, active, discarded, "关闭未完成：无法持久化恢复状态")
}

func (a *App) persistTabsWithDiscard(tabs []*Tab, active int, discarded []bool, action string) error {
	// Explicit save/close and automatic attempts supersede queued timer work,
	// including callbacks already posted before the timer could be stopped.
	a.cancelAutosave()
	if a.restoring {
		return fmt.Errorf("会话恢复尚未完成；未覆盖现有恢复状态")
	}
	// This attempt consumes the current request even on failure. In particular,
	// unwinding a failed close dialog must not silently retry unchanged state.
	a.persistPending = false
	theme := "light"
	if a.Dark {
		theme = "dark"
	}
	lineNumbers := a.LineNumbers
	builder := session.NewSnapshotBuilder(a.Root)
	if err := builder.SetPreferences(session.Preferences{
		Theme: theme, FontSize: a.FontSize, LineNumbers: &lineNumbers,
		WordWrap: a.SoftWrap, SidebarWidth: a.SidebarWidth,
	}); err != nil {
		return a.reportPersistFailure(fmt.Errorf(
			"%s：%w；草稿未写入恢复快照，现有恢复快照保持不变", action, err,
		))
	}
	mappedActive := -1
	for i, t := range tabs {
		if t.isBinary() {
			if i == active {
				mappedActive = builder.Len()
			}
			if err := builder.AddBinaryTab(session.BinaryTab{Path: t.path(), Kind: t.BinaryKind, Title: t.Name, PluginID: t.MatchedPlugin.ID, View: t.Mode}); err != nil {
				return a.reportPersistFailure(fmt.Errorf("%s：%s：%w；现有恢复快照保持不变", action, t.Name, err))
			}
			continue
		}
		if t.ByteInspector {
			continue
		}
		dropDraft := i < len(discarded) && discarded[i]
		if dropDraft && t.Doc.Path == "" {
			continue
		}
		if i == active {
			mappedActive = builder.Len()
		}
		var err error
		if dropDraft {
			err = builder.AddTab(session.Tab{
				Path: t.Doc.Path, Title: t.Name, Encoding: t.Doc.Encoding, BOM: t.Doc.BOM,
				Selection: t.Doc.Buffer.Selection(), Scroll: t.Editor.ScrollY,
				OriginalPath: t.OriginalPath, PluginID: t.MatchedPlugin.ID, View: t.Mode,
			})
		} else {
			err = builder.AddDocumentWithOptions(t.Doc, session.CaptureOptions{
				Title: t.Name, Scroll: t.Editor.ScrollY, OriginalPath: t.OriginalPath,
				PluginID: t.MatchedPlugin.ID, View: t.Mode,
			})
		}
		if err != nil {
			return a.reportPersistFailure(fmt.Errorf(
				"%s：%s：%w；草稿未写入恢复快照，现有恢复快照保持不变",
				action, t.Name, err,
			))
		}
	}
	activeToSave := 0
	if builder.Len() > 0 {
		if mappedActive >= 0 {
			activeToSave = mappedActive
		} else {
			activeToSave = max(0, min(active, builder.Len()-1))
		}
	}
	if err := builder.SetActive(activeToSave); err != nil {
		return a.reportPersistFailure(fmt.Errorf(
			"%s：%w；草稿未写入恢复快照，现有恢复快照保持不变", action, err,
		))
	}
	if err := builder.Save(filepath.Join(a.configDir, "session.json")); err != nil {
		return a.reportPersistFailure(fmt.Errorf("%s：%w", action, err))
	}
	a.lastPersistError = ""
	a.persistPending = false
	return nil
}

func (a *App) reportPersistFailure(err error) error {
	if err == nil || err.Error() == a.lastPersistError {
		return err
	}
	a.report(err)
	a.lastPersistError = err.Error()
	return err
}

// persistAutomatic makes at most one automatic attempt for a given state.
// A failure leaves the previous recovery file intact and clears the automatic
// request; a document/session preference change marks it pending again, while
// explicit user actions call Persist (or a lifecycle persistence method) and
// intentionally request a fresh attempt.
func (a *App) persistAutomatic() error {
	if !a.persistPending {
		return nil
	}
	err := a.persistTabs(a.Tabs, a.Active, "保存恢复状态")
	if err != nil {
		a.persistPending = false
	}
	return err
}

func (a *App) findOptions() search.Options {
	return search.Options{Query: a.Query, Regex: a.Regex, CaseSensitive: a.CaseSensitive, WholeWord: a.WholeWord}
}
func (a *App) replace(all bool) {
	t := a.Current()
	if !t.textCapable() || t.Doc.ReadOnly {
		return
	}
	doc, input, buffer := t.Doc, t.Editor, t.Doc.Buffer
	if !a.flushTab(t, "替换") {
		return
	}
	if !a.findTargetCurrent(t, doc, input, buffer) {
		a.invalidateFindCache()
		a.Message = "替换时文档已变化；未替换"
		return
	}
	input.UnmarkText()
	opts := a.findOptions()
	changed := false
	if all {
		count, err := search.ReplaceAll(buffer, opts, a.Replacement)
		if err != nil {
			a.report(err)
			return
		}
		changed = count > 0
		a.Message = fmt.Sprintf("已替换 %d 处", count)
	} else {
		matches, err := search.Find(buffer.Snapshot(), opts)
		if err != nil {
			a.report(err)
			return
		}
		s := buffer.Selection()
		for _, m := range matches {
			if m.Start == s.Start() && m.End == s.End() {
				if err = search.Replace(buffer, m, opts, a.Replacement); err != nil {
					a.report(err)
					return
				}
				changed = true
				break
			}
		}
	}
	if changed {
		a.invalidateFindCache()
		if t.View != nil {
			a.releasePresentation(t)
		}
	}
	if !all {
		a.find(true)
	}
	if changed {
		input.ReleaseView()
		a.noteEdit()
	}
}
func (a *App) findbar(c *ui.Context) {
	generation := a.findGeneration
	callbackTab := a.Current()
	var callbackDoc *document.Document
	var callbackInput *editor.Editor
	var callbackBuffer *text.Buffer
	if callbackTab != nil {
		callbackDoc = callbackTab.Doc
		callbackInput = callbackTab.Editor
		if callbackDoc != nil {
			callbackBuffer = callbackDoc.Buffer
		}
	}
	fieldCurrent := func() bool {
		return a.ShowFind && a.findGeneration == generation &&
			a.findTargetCurrent(callbackTab, callbackDoc, callbackInput, callbackBuffer)
	}
	cancel := func() {
		if fieldCurrent() {
			a.cancelFindAt(generation)
		}
	}
	var queryInput, replacementInput ui.Element
	panel := ui.Column(c.Key("findbar")).Padding(8, 12).Gap(6).Background(c.Theme().Surface).
		Bind(&a.findPanel)
	panel.Children(func() {
		ui.Row(c).Wrap().Gap(8).Children(func() {
			queryInput = ui.TextInput(c.Key("find-query"), &a.Query).
				Placeholder("查找文本或正则表达式").Label("查找内容").Width(260).
				ObserveInput(func(ev ui.InputEvent) {
					if !fieldCurrent() {
						return
					}
					switch ev.Kind {
					case ui.InputCompose:
						a.FindQueryComposing = ev.Text != ""
					case ui.InputText:
						a.FindQueryComposing = false
					}
				}).
				OnTextKey(func(k ui.TextKey) bool {
					if !fieldCurrent() {
						return false
					}
					if a.findCompositionActive() && (k.Key == ui.KeyEnter || k.Key == ui.KeyEscape) {
						return true
					}
					if k.Key == ui.KeyEnter && (k.Mods == 0 || k.Mods == ui.Shift) {
						return a.findFromQueryKeyAt(generation, k, k.Mods == 0)
					}
					if k.Key == ui.KeyEscape && k.Mods == 0 {
						return a.cancelFindAt(generation)
					}
					return false
				})
			queryInput.Bind(&a.findQueryHandle)
			if ui.Button(c, "上一个").KeepFocus().
				OnShortcut(0, ui.KeyEscape, cancel).
				Clicked() {
				if fieldCurrent() && a.findActionAllowedAt(generation) {
					a.find(false)
				}
			}
			if ui.Button(c, "下一个").KeepFocus().
				OnShortcut(0, ui.KeyEscape, cancel).
				Clicked() {
				if fieldCurrent() && a.findActionAllowedAt(generation) {
					a.find(true)
				}
			}
			ui.Checkbox(c, &a.CaseSensitive, "区分大小写").KeepFocus().
				OnShortcut(0, ui.KeyEscape, cancel)
			ui.Checkbox(c, &a.Regex, "正则").KeepFocus().
				OnShortcut(0, ui.KeyEscape, cancel)
			ui.Checkbox(c, &a.WholeWord, "全词").KeepFocus().
				OnShortcut(0, ui.KeyEscape, cancel)
			ui.Spacer(c)
			if ui.Button(c, "×").Label("关闭查找").KeepFocus().
				OnShortcut(0, ui.KeyEscape, cancel).
				Clicked() {
				cancel()
			}
		})
		ui.Row(c).Wrap().Gap(8).Children(func() {
			replacementInput = ui.TextInput(c.Key("find-replacement"), &a.Replacement).
				Placeholder("替换为").Label("替换内容").Width(260).
				ObserveInput(func(ev ui.InputEvent) {
					if !fieldCurrent() {
						return
					}
					switch ev.Kind {
					case ui.InputCompose:
						a.FindReplacementComposing = ev.Text != ""
					case ui.InputText:
						a.FindReplacementComposing = false
					}
				}).
				OnTextKey(func(k ui.TextKey) bool {
					if !fieldCurrent() {
						return false
					}
					if a.findCompositionActive() && (k.Key == ui.KeyEnter || k.Key == ui.KeyEscape) {
						return true
					}
					if k.Key == ui.KeyEnter {
						if k.Mods == ui.Shift {
							return true
						}
						if k.Mods == 0 {
							return a.replaceFromReplacementKeyAt(generation, k)
						}
					}
					if k.Key == ui.KeyEscape && k.Mods == 0 {
						return a.cancelFindAt(generation)
					}
					return false
				})
			replacementInput.Bind(&a.findReplacementHandle)
			if ui.Button(c, "替换当前").KeepFocus().
				OnShortcut(0, ui.KeyEscape, cancel).
				Clicked() {
				if fieldCurrent() && a.findActionAllowedAt(generation) {
					a.replace(false)
				}
			}
			if ui.Button(c, "全部替换").KeepFocus().
				OnShortcut(0, ui.KeyEscape, cancel).
				Clicked() {
				if fieldCurrent() && a.findActionAllowedAt(generation) {
					a.replace(true)
				}
			}
			ui.Spacer(c)
			if ui.Button(c, "搜索工作区").KeepFocus().
				OnShortcut(0, ui.KeyEscape, cancel).
				Clicked() {
				if fieldCurrent() && a.findActionAllowedAt(generation) {
					a.searchWorkspace()
				}
			}
		})
	})
	a.reconcileFindFieldComposition(queryInput, &a.FindQueryComposing)
	a.reconcileFindFieldComposition(replacementInput, &a.FindReplacementComposing)
	a.resolveFindFocus(generation)
}
