package app

import (
	"errors"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

const commandPaletteQueryLabel = "搜索命令"

type workbenchCommand struct {
	id, title, keywords, shortcut string
	enabled                       func(*App) bool
	run                           func(*App)
}

func commandHasText(a *App) bool {
	t := a.Current()
	return t != nil && t.textCapable() && t.Editor != nil && t.Doc.Buffer != nil
}
func commandHasView(a *App) bool {
	return commandHasText(a) && a.Current().Adapter != nil && a.Current().Adapter.Create != nil
}

// Only real application actions belong here. The command ID, never a filtered
// row index or a captured tab, is resolved again at the point of execution.
var workbenchCommands = []workbenchCommand{
	{"file.new.markdown", "文件：新建 Markdown", "new markdown note", "Ctrl+N", nil, func(a *App) { a.NewDocument("markdown") }},
	{"file.new.text", "文件：新建文本", "new text code", "", nil, func(a *App) { a.NewDocument("text") }},
	{"file.new.mindmap", "文件：新建思维导图", "new mindmap xmind", "", nil, func(a *App) { a.NewDocument("mindmap") }},
	{"file.new.drawio", "文件：新建图纸", "new drawio diagram", "", nil, func(a *App) { a.NewDocument("drawio") }},
	{"file.open", "文件：打开文件", "open file", "Ctrl+O", nil, func(a *App) { a.ChooseOpen() }},
	{"file.folder", "文件：打开文件夹", "open folder workspace", "", nil, func(a *App) { a.ChooseFolder() }},
	{"file.save", "文件：保存", "save", "Ctrl+S", func(a *App) bool { return commandHasText(a) && !a.Current().Doc.ReadOnly && !a.Current().Saving }, func(a *App) { a.Save(a.Current(), false) }},
	{"file.saveAs", "文件：另存为", "save as", "Ctrl+Shift+S", func(a *App) bool { return commandHasText(a) && !a.Current().Saving }, func(a *App) { a.Save(a.Current(), true) }},
	{"file.close", "文件：关闭当前文档", "close tab", "Ctrl+W", func(a *App) bool { return a.Current() != nil }, func(a *App) { a.CloseTab(a.Active) }},
	{"navigation.documents", "导航：搜索已打开文档", "open documents tabs switch", "Ctrl+P", func(a *App) bool { return a.Current() != nil }, func(a *App) { a.openDocumentPicker() }},
	{"navigation.line", "导航：转到行号", "go to line column", "Ctrl+G", commandHasText, func(a *App) { a.ShowPlugins = false; a.openGoToLine() }},
	{"navigation.find", "编辑：查找与替换", "find search replace", "Ctrl+F", commandHasText, func(a *App) { a.ShowPlugins = false; a.openFind() }},
	{"fold.current", "编辑：折叠当前位置", "fold collapse current block", "", sourceFoldCommandAvailable, func(a *App) { a.runSourceFoldCommand(foldCurrentPosition) }},
	{"fold.unfoldCurrent", "编辑：展开当前位置", "unfold expand current block", "", sourceFoldCommandAvailable, func(a *App) { a.runSourceFoldCommand(unfoldCurrentPosition) }},
	{"fold.unfoldAll", "编辑：展开全部", "unfold expand all", "", sourceFoldCommandAvailable, func(a *App) { a.runSourceFoldCommand(unfoldEverything) }},
	{"fold.refresh", "编辑：重新分析折叠", "fold refresh reanalyze", "", sourceFoldCommandAvailable, func(a *App) { a.runSourceFoldCommand(reanalyzeFolds) }},
	{"language.analyze", "语言：分析当前文档", "language analyze diagnostics Go JSON syntax", "", languageReadCommandAvailable, func(a *App) { a.runSourceLanguageCommand("diagnostics") }},
	{"language.symbols", "语言：转到文档符号", "language symbols Go JSON", "", languageReadCommandAvailable, func(a *App) { a.runSourceLanguageCommand("symbols") }},
	{"language.definition", "语言：转到本文件定义", "language definition Go", "", languageGoCommandAvailable, func(a *App) { a.runSourceLanguageCommand("definition") }},
	{"language.references", "语言：查找本文件引用", "language references Go", "", languageGoCommandAvailable, func(a *App) { a.runSourceLanguageCommand("references") }},
	{"language.info", "语言：查看符号信息", "language info declaration documentation Go", "", languageGoCommandAvailable, func(a *App) { a.runSourceLanguageCommand("info") }},
	{"language.complete", "语言：源码补全", "language completion vocabulary Go JSON", "", languageCompletionCommandAvailable, func(a *App) { a.runSourceCompletionCommand() }},
	{"language.rename", "语言：重命名本文件符号", "language rename local binding Go", "", languageRenameCommandAvailable, func(a *App) { a.runSourceRenameCommand() }},
	{"language.cancel", "语言：取消当前分析", "language cancel", "", languageCancelCommandAvailable, func(a *App) { a.runSourceLanguageCancel() }},

	{"view.source", "视图：源码", "source code", "", commandHasView, func(a *App) { a.ShowPlugins = false; a.setMode(a.Current(), "source") }},
	{"view.preview", "视图：阅读与编辑", "preview read edit", "", commandHasView, func(a *App) { a.ShowPlugins = false; a.setMode(a.Current(), "preview") }},
	{"view.split", "视图：分屏", "split preview source", "", commandHasView, func(a *App) { a.ShowPlugins = false; a.setMode(a.Current(), "split") }},
	{"view.outline", "视图：显示文档大纲", "outline headings", "", commandHasText, func(a *App) { a.setOutlineVisible(true) }},
	{"view.files", "视图：显示文件树", "explorer files", "", nil, func(a *App) { a.setOutlineVisible(false) }},
	{"view.theme", "外观：切换明暗主题", "theme dark light", "", nil, func(a *App) { a.Dark = !a.Dark; a.markPersistPending() }},
	{"view.wrap", "外观：切换自动换行", "word wrap", "", nil, func(a *App) { a.SoftWrap = !a.SoftWrap; a.applyEditorPreferences(); a.markPersistPending() }},
	{"view.lineNumbers", "外观：切换行号显示", "line numbers", "", nil, func(a *App) { a.LineNumbers = !a.LineNumbers; a.applyEditorPreferences(); a.markPersistPending() }},
	{"view.font.increase", "外观：增大编辑字号", "font zoom increase", "", nil, func(a *App) { a.font(1) }},
	{"view.font.decrease", "外观：减小编辑字号", "font zoom decrease", "", nil, func(a *App) { a.font(-1) }},
	{"workbench.settings", "工作台：外观设置", "settings preferences", "", nil, func(a *App) { a.ShowPlugins = false; a.ShowSettings = true }},
	{"workbench.plugins", "工作台：插件管理", "plugins extensions", "", nil, func(a *App) { a.ShowPlugins = true }},
	{"workbench.issues", "帮助：本次运行的问题", "help issues diagnostics", "", nil, func(a *App) { a.ShowPlugins = false; a.ShowIssues = true }},
}

type commandPaletteState struct {
	open, composing               bool
	generation                    uint64
	origin                        *Tab
	query, filteredQuery, message string
	matches                       []int
	selected                      int
	selectedID, pendingID         string
	pendingClose, reveal          bool
	list                          ui.ListState
	queryHandle                   ui.Handle
}

func (a *App) commandPaletteAllowed() error {
	if a.shuttingDown.Load() || a.restoring || a.modalDepth > 0 {
		return errors.New("请先结束当前对话框或工作区恢复，再打开命令面板")
	}
	// This pure input-only check also sees a first refused native preedit,
	// before any focus, flush, palette close, or command side effect.
	if err := a.sourceLanguageInputError(); err != nil {
		return err
	}
	if a.commandPalette.composing || a.openDocuments.composing || a.findCompositionActive() || a.GoToLineComposing || a.sourceCompositionActive(a.Current()) {
		return errors.New("请先完成或取消输入法组合，再打开命令面板")
	}
	return nil
}

func (a *App) openCommandPalette() bool {
	p := &a.commandPalette
	if err := a.commandPaletteAllowed(); err != nil {
		a.report(err)
		return false
	}
	if p.open {
		if !a.commandPaletteCurrent(p.generation) {
			a.closeCommandPalette()
			return false
		}
		p.queryHandle.Focus()
		return true
	}
	origin := a.Current()
	if origin != nil && !a.flushTab(origin, "打开命令面板") {
		return false
	}
	if a.Current() != origin {
		a.Message = "同步输入期间文档已变化；请重新打开命令面板"
		return false
	}
	if err := a.commandPaletteAllowed(); err != nil {
		a.report(err)
		return false
	}
	a.closeOpenDocuments()
	a.closeCommandPalette()
	p.open, p.origin = true, origin
	p.queryHandle.Focus()
	a.refreshCommandPalette("")
	return true
}

func (a *App) closeCommandPalette() {
	p := &a.commandPalette
	p.queryHandle.CancelFocus()
	p.list.Handle.CancelFocus()
	*p = commandPaletteState{generation: p.generation + 1, selected: -1, queryHandle: p.queryHandle, list: ui.ListState{Handle: p.list.Handle}}
}
func (a *App) commandPaletteCurrent(generation uint64) bool {
	p := &a.commandPalette
	return p.open && p.generation == generation && a.Current() == p.origin && !a.shuttingDown.Load() && !a.restoring && a.modalDepth == 0
}

func (a *App) refreshCommandPalette(query string) {
	p := &a.commandPalette
	if query == p.filteredQuery && p.matches != nil {
		return
	}
	terms := strings.Fields(strings.ToLower(strings.TrimPrefix(strings.TrimSpace(query), ">")))
	// Replace rather than mutate the backing array: MyGo can still be laying
	// out a list built earlier in this frame.
	matches := make([]int, 0, len(workbenchCommands))
	for i, command := range workbenchCommands {
		haystack := strings.ToLower(command.title + " " + command.keywords + " " + command.shortcut)
		match := true
		for _, term := range terms {
			if !strings.Contains(haystack, term) {
				match = false
				break
			}
		}
		if match {
			matches = append(matches, i)
		}
	}
	// Filtering changes membership and intentionally selects the first result.
	// Discard the previous stable row key while retaining the widget handle.
	p.list = ui.ListState{Handle: p.list.Handle}
	p.matches, p.filteredQuery, p.selected, p.selectedID = matches, query, -1, ""
	if len(matches) > 0 {
		p.selected = 0
		p.selectedID = workbenchCommands[matches[0]].id
	}
	p.reveal = true
}

// Consume actions before constructing any new list rows. Deferred callbacks
// retain only the generation and command ID, and execution checks live state.
func (a *App) processCommandPalette() {
	p := &a.commandPalette
	if !p.open {
		return
	}
	if !a.commandPaletteCurrent(p.generation) {
		a.closeCommandPalette()
		return
	}
	// Do not consume close/execute intent until every input owner has passed
	// its pure admission check. In particular a markless first-refusal must
	// retain the original editor's Escape/final-text route and the palette's
	// exact pending action/list identity.
	if p.composing {
		return
	}
	if err := a.commandPaletteAllowed(); err != nil {
		if a.commandPaletteCurrent(p.generation) {
			// Drop this refused action so it cannot fire automatically after the
			// source owner later settles; keep the same open palette/list state.
			p.pendingID, p.pendingClose = "", false
			p.message = err.Error()
		}
		return
	}
	id, close := p.pendingID, p.pendingClose
	p.pendingID, p.pendingClose = "", false
	if close {
		a.closeCommandPalette()
		return
	}
	if id == "" {
		return
	}
	for _, command := range workbenchCommands {
		if command.id != id {
			continue
		}
		if command.enabled != nil && !command.enabled(a) {
			p.message = "此命令当前不可用，请先打开适用的文档"
			return
		}
		a.closeCommandPalette()
		command.run(a)
		return
	}
}

func (a *App) commandPaletteKey(generation uint64, k ui.TextKey) bool {
	if !a.commandPaletteCurrent(generation) {
		return false
	}
	p := &a.commandPalette
	if p.composing {
		return true
	} // only native Compose/Text may finish preedit
	if k.Mods != 0 {
		return false
	}
	switch k.Key {
	case ui.KeyEscape:
		p.pendingClose = true
		return true
	case ui.KeyUp, ui.KeyDown, ui.KeyEnter:
		p.query = k.Text()
		a.refreshCommandPalette(p.query)
		if k.Key == ui.KeyEnter {
			p.pendingID = p.selectedID
			return true
		}
		if len(p.matches) > 0 {
			delta := 1
			if k.Key == ui.KeyUp {
				delta = -1
			}
			p.selected = max(0, min(len(p.matches)-1, p.selected+delta))
			p.selectedID, p.reveal = workbenchCommands[p.matches[p.selected]].id, true
		}
		return true
	}
	return false
}

func (a *App) commandPaletteDialog(c *ui.Context) {
	p := &a.commandPalette
	if !p.open {
		return
	}
	if !a.commandPaletteCurrent(p.generation) {
		a.closeCommandPalette()
		return
	}
	a.refreshCommandPalette(p.query)
	generation := p.generation
	shown, cancel := true, false
	var input ui.Element
	width, height := c.Size()
	panel := ui.DialogBase(c.Key(fmt.Sprintf("command-palette-%d", generation)), &shown, func(backdrop, panel ui.Element) {
		backdrop.KeepFocus().Background(ui.RGBA(0, 0, 0, 0.25))
		panel.Width(min(float32(600), max(240, width-32))).Height(min(float32(420), max(160, height-32))).
			Background(c.Theme().Surface).Border(1, c.Theme().Border).Radius(10).Label("命令面板").
			ObserveInput(func(ev ui.InputEvent) {
				if !a.commandPaletteCurrent(generation) {
					return
				}
				switch ev.Kind {
				case ui.InputCompose:
					p.composing = ev.Text != ""
				case ui.InputText:
					p.composing = false
				}
			})
		if panel.Shortcut(ui.Cmd|ui.Shift, ui.KeyP) || panel.Shortcut(0, ui.KeyF1) {
			if err := a.commandPaletteAllowed(); err != nil {
				if a.commandPaletteCurrent(generation) {
					p.message = err.Error()
				}
			} else if a.commandPaletteCurrent(generation) {
				p.queryHandle.Focus()
			}
		}
		ui.Column(c).Fill().Padding(14).Gap(9).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "命令面板").FontSize(16).Bold().Grow(1)
				ui.Text(c, fmt.Sprintf("%d 项", len(p.matches))).FontSize(12).TextColor(c.Theme().TextMuted)
				if workbenchButton(c, "关闭").Label("关闭命令面板").KeepFocus().Clicked() {
					cancel = true
				}
			})
			input = ui.TextInput(c.Key("query"), &p.query).Label(commandPaletteQueryLabel).
				Placeholder("输入命令，例如：分屏、主题、查找").FillWidth().Height(36).MaxLength(128).Bind(&p.queryHandle).
				OnTextKey(func(k ui.TextKey) bool { return a.commandPaletteKey(generation, k) })
			if p.query != p.filteredQuery {
				a.refreshCommandPalette(p.query)
			}
			rows := p.matches
			p.list.Selected = &p.selected
			p.list.Key = func(i int) any { return workbenchCommands[rows[i]].id }
			p.list.Label = func(i int) string { return workbenchCommands[rows[i]].title }
			if p.reveal && p.selected >= 0 {
				p.list.ScrollIntoView(p.selected)
				p.reveal = false
			}
			list := ui.List(c.Key("commands"), &p.list, len(rows), func(i int) {
				command := workbenchCommands[rows[i]]
				enabled := command.enabled == nil || command.enabled(a)
				button := ui.ButtonBase(c).Role(ui.RoleButton).Label("执行命令："+command.title).
					FillWidth().Height(38).Padding(6, 9).KeepFocus().Disabled(!enabled)
				button.Children(func() {
					ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Children(func() {
						ui.Text(c, command.title).SingleLine().Grow(1).MinWidth(0).FontSize(14)
						badge := command.shortcut
						if !enabled {
							badge = "当前不可用"
						}
						if badge != "" {
							ui.Text(c, badge).FontSize(11).Shrink(0)
						}
					})
				})
				if button.Clicked() && a.commandPaletteCurrent(generation) {
					if p.composing {
						p.message = "请先完成或取消输入法组合"
					} else {
						p.pendingID = command.id
					}
					c.Invalidate()
				}
			}).Grow(1).MinHeight(0).KeepFocus()
			if p.selected >= 0 && p.selected < len(rows) {
				p.selectedID = workbenchCommands[rows[p.selected]].id
				if list.Submitted() && !p.composing {
					p.pendingID = p.selectedID
					c.Invalidate()
				}
			}
			if len(rows) == 0 {
				ui.Text(c, "没有匹配的命令，请尝试其他关键词").FontSize(12)
			}
			hint := "↑↓ 选择   Enter 执行   Esc 返回"
			if p.message != "" {
				hint = p.message
			}
			ui.Text(c, hint).FontSize(12).TextColor(c.Theme().TextMuted).MaxLines(2)
		})
	})
	if p.composing && !input.Focused() && !input.Composing() {
		p.composing = false
	}
	if panel.Dismissed() || cancel {
		if p.composing {
			p.message = "请先完成或取消输入法组合"
		} else {
			p.pendingClose = true
			c.Invalidate()
		}
	}
}
