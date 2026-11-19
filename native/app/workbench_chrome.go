package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"
)

const (
	workbenchToolbarHeight float32 = 48
	workbenchTabsHeight    float32 = 44
	workbenchStatusHeight  float32 = 29
	workbenchActivityWidth float32 = 56
)

// Keep a useful document width without changing the persisted sidebar preference.
func (a *App) workbenchSidebarWidth(c *ui.Context) float32 {
	width, _ := c.Size()
	limit := max(float32(180), width-workbenchActivityWidth-508)
	return min(max(a.SidebarWidth, 180), min(float32(280), limit))
}

// Buttons keep MyGo's keyboard, disabled and accessibility behavior; only their
// face changes. Explicit hover/press feedback also works on non-surface chrome.
func workbenchButton(c *ui.Context, label string) ui.Element {
	b := ui.Button(c, label).Height(32).Padding(4, 10).Border(0, ui.Color{}).Radius(6).FontSize(14)
	b.Background(ui.Color{})
	if b.Pressed() {
		b.Background(c.Theme().SurfacePressed)
	} else if b.Hovered() {
		b.Background(c.Theme().SurfaceHover)
	}
	return b
}

// Status controls use the same quiet hover/press surfaces as the workbench.
func workbenchStatusButton(c *ui.Context, label string) ui.Element {
	return workbenchButton(c, label).Height(24).Padding(2, 6).FontSize(12).TextColor(c.Theme().TextMuted)
}

// workbenchActionButton is a toolbar ghost action: icon + label, one height,
// one radius, theme surfaces on hover/press. The label stays the click and
// accessibility identity of the button.
func workbenchActionButton(c *ui.Context, key, label string, icon *ui.SVG, tooltip string) ui.Element {
	b := workbenchButton(c.Key(key), "").Label(label).Tooltip(tooltip).
		Height(32).Padding(0, 10).Gap(6).Radius(8).Shrink(0)
	b.Children(func() {
		ui.Icon(c, icon).Size(15, 15).Shrink(0)
		ui.Text(c, label).FontSize(13).SingleLine()
	})
	return b
}

func (a *App) workbenchToolbar(c *ui.Context) {
	s := workbenchAppearance(a.Dark)
	width, _ := c.Size()
	ui.Row(c.Key("toolbar")).Height(workbenchToolbarHeight).Shrink(0).Padding(0, 12).Gap(2).
		Background(s.Chrome).AlignItems(ui.Center).Children(func() {
		ui.Row(c.Key("workbench-brand")).Gap(7).Padding(0, 10, 0, 0).Shrink(0).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, workbenchBrandIcon).Size(20, 20).TextColor(s.Theme.Focus)
			ui.Text(c, "轻笺").FontSize(14).Bold()
		})
		ui.Row(c.Key("toolbar-brand-divider")).Width(1).Height(20).Shrink(0).Margin(0, 6, 0, 4).Background(s.Theme.Border)
		if workbenchActionButton(c, "toolbar-new", "新建", workbenchPlusIcon, "新建 Markdown · Ctrl+N").Clicked() {
			a.NewDocument("markdown")
		}
		if workbenchActionButton(c, "toolbar-open", "打开", workbenchFolderIcon, "打开文件 · Ctrl+O").Clicked() {
			a.ChooseOpen()
		}
		a.closedFilesMenu(c)
		if workbenchActionButton(c, "toolbar-folder", "文件夹", workbenchFilesIcon, "打开工作区文件夹").Clicked() {
			a.ChooseFolder()
		}
		if workbenchActionButton(c, "toolbar-save", "保存", workbenchSaveIcon, "保存 · Ctrl+S").Disabled(a.Current() == nil).Clicked() {
			a.Save(a.Current(), false)
		}
		ui.Spacer(c)
		search := workbenchButton(c.Key("toolbar-document-search"), "").Label("搜索已打开的文档").
			Tooltip("搜索已打开的文档 · Ctrl+P").KeepFocus().Width(min(float32(340), max(float32(240), width*0.28))).
			MinWidth(224).Shrink(1).Height(32).Padding(0, 12).Gap(8).Radius(16).
			Border(1, s.Theme.Border).Background(s.Theme.Surface).TextColor(s.Subtle)
		if search.Pressed() {
			search.Background(s.Theme.SurfacePressed)
		} else if search.Hovered() {
			search.Background(s.Theme.SurfaceHover)
		}
		search.Children(func() {
			ui.Icon(c, workbenchSearchIcon).Size(15, 15).Shrink(0)
			ui.Text(c, "搜索已打开的文档").FontSize(12.5).SingleLine().MinWidth(0).Shrink(1)
			ui.Spacer(c)
			ui.Text(c, "Ctrl+P").FontSize(11).Padding(2, 6).Background(s.Chrome).Radius(4).Shrink(0)
		})
		if search.Clicked() {
			a.openDocumentPicker()
		}
		if workbenchActionButton(c, "toolbar-commands", "命令", workbenchCommandIcon, "命令面板 · Ctrl+Shift+P / F1").Label("打开命令面板").KeepFocus().Clicked() {
			a.openCommandPalette()
		}
		if width >= 1040 {
			workspace := "LevNote"
			if a.Root != "" {
				workspace = filepath.Base(a.Root)
			}
			ui.Text(c, workspace).FontSize(12).TextColor(s.Subtle).Padding(0, 2, 0, 8).
				SingleLine().MinWidth(0).MaxWidth(180).Shrink(1).Tooltip(a.Root)
		}
	})
}

func workbenchActivity(c *ui.Context, key, label string, icon *ui.SVG, active, disabled bool) bool {
	s := c.Theme()
	b := workbenchButton(c.Key(key), "").Label(label).Tooltip(label).
		Width(40).Height(40).Padding(10).Radius(8).Shrink(0).Disabled(disabled)
	if key == "activity-search" {
		b.KeepFocus()
	}
	if active {
		b.TextColor(s.Focus)
		if !b.Hovered() && !b.Pressed() {
			b.Background(s.Selection)
		}
	} else {
		b.TextColor(s.TextMuted)
	}
	b.Children(func() { ui.Icon(c, icon).Size(20, 20) })
	return b.Clicked()
}

func (a *App) workbenchActivityBar(c *ui.Context) {
	s := workbenchAppearance(a.Dark)
	ui.Column(c.Key("activity-bar")).Width(workbenchActivityWidth).Shrink(0).FillHeight().Padding(8).Gap(6).
		Background(s.Activity).AlignItems(ui.Center).Children(func() {
		if workbenchActivity(c, "activity-files", "文件树", workbenchFilesIcon, !a.ShowOutline && !a.ShowPlugins && !a.ShowFind, false) {
			a.setOutlineVisible(false)
			a.ShowPlugins = false
		}
		if workbenchActivity(c, "activity-search", "查找", workbenchSearchIcon, a.ShowFind && !a.ShowPlugins, !a.Current().textCapable()) {
			if a.ShowPlugins {
				a.ShowPlugins = false
				a.openFind()
			} else {
				a.toggleFind()
			}
		}
		if workbenchActivity(c, "activity-outline", "文档大纲", workbenchOutlineIcon, a.ShowOutline && !a.ShowPlugins, a.Current().isBinary() && !a.ShowOutline) {
			a.setOutlineVisible(!a.ShowOutline)
			a.ShowPlugins = false
		}
		if workbenchActivity(c, "activity-plugins", "插件", workbenchPluginsIcon, a.ShowPlugins, false) {
			a.closeGoToLine()
			a.ShowPlugins = !a.ShowPlugins
		}
		ui.Spacer(c)
		if workbenchActivity(c, "activity-settings", "设置", workbenchSettingsIcon, a.ShowSettings && !a.ShowPlugins, false) {
			if a.ShowPlugins {
				a.ShowPlugins = false
				a.ShowSettings = true
			} else {
				a.ShowSettings = !a.ShowSettings
			}
		}
	})
}

func workbenchFileIcon(name string) *ui.SVG {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".md", ".markdown", ".mdown":
		return workbenchMarkdownIcon
	case ".mindmap", ".drawio":
		return workbenchDiagramIcon
	case ".go", ".js", ".ts", ".tsx", ".json", ".css", ".html", ".py":
		return workbenchCodeIcon
	default:
		return workbenchDocumentIcon
	}
}

func (a *App) workbenchStatus(c *ui.Context) {
	s := workbenchAppearance(a.Dark)
	width, _ := c.Size()
	ui.Row(c.Key("status")).Height(workbenchStatusHeight).Shrink(0).Padding(2, 12).Gap(12).Clip().
		Background(s.Status).TextColor(s.StatusText).AlignItems(ui.Center).Children(func() {
		message := a.Message
		if message == "" {
			message = "本地文件 · 原生编辑"
		} else {
			ui.Icon(c, workbenchNoticeIcon).Size(14, 14).TextColor(s.StatusText).Shrink(0)
		}
		ui.Text(c.Key("status-message"), message).FontSize(12).TextColor(s.StatusText).
			Grow(1).Shrink(1).MinWidth(0).SingleLine().Tooltip(message)
		if t := a.Current(); t != nil {
			if t.isBinary() {
				a.binaryStatus(c, t).FontSize(12).TextColor(s.StatusText).Shrink(0)
			} else {
				if width >= 900 || a.Message == "" {
					line, col, _ := t.Doc.Buffer.LineColumn(t.Doc.Buffer.Selection().Head)
					position := fmt.Sprintf("第 %d 行，第 %d 列", line+1, col+1)
					if workbenchStatusButton(c.Key("status-position"), position).Label("转到源码行；当前" + position).
						TextColor(s.StatusText).KeepFocus().Height(24).Shrink(0).Clicked() {
						a.openGoToLine()
					}
				}
				encodingLabel := string(t.Doc.Encoding)
				if t.ByteInspector {
					encodingLabel = "二进制（字节视图）"
					if t.CanReinterpret {
						encodingLabel = "原始编码未知（字节视图）"
					}
				}
				ui.Text(c, encodingLabel).FontSize(12).TextColor(s.StatusText).Shrink(0)
				state := "已保存"
				if t.Doc.ReadOnly {
					state = "只读"
				} else if t.Doc.Dirty() {
					state = "未保存"
				}
				ui.Text(c, state).FontSize(12).TextColor(s.StatusText).Shrink(0)
			}
		}
		if len(a.Issues) > 0 && workbenchStatusButton(c.Key("status-issues"), fmt.Sprintf("问题 %d", len(a.Issues))).
			TextColor(s.StatusText).Height(24).Shrink(0).Clicked() {
			a.ShowIssues = !a.ShowIssues
		}
	})
}
