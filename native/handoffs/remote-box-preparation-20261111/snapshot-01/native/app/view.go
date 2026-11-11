package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
)

func (a *App) View(c *ui.Context) {
	a.deliverBinary()
	a.expireFindCache()
	a.deliverPackageWork()
	if a.exportJob != nil {
		a.deliverEngineExport(a.exportJob)
	}
	a.deliverScene()
	theme := ui.LightTheme()
	if a.Dark {
		theme = ui.DarkTheme()
	}
	theme.Accent = ui.Hex("#33766c")
	theme.AccentHover = ui.Hex("#28655c")
	theme.AccentPressed = ui.Hex("#20584f")
	theme.Radius = 5
	theme.FontSize = 13
	theme.Font = "Microsoft YaHei UI"
	c.SetTheme(theme)
	a.expireGoToLine()
	a.expireFindFocus()
	a.processOpenDocuments()
	if a.ShowPlugins && a.ShowFind {
		a.clearFindTransient()
	}
	if a.ShowPlugins && a.ShowGoToLine {
		a.closeGoToLine()
	}
	a.shortcuts(c)
	ui.Column(c).Fill().Background(theme.Background).Children(func() {
		ui.Row(c.Key("toolbar")).Height(46).Padding(7, 16).Gap(8).Background(theme.Surface).Children(func() {
			ui.Text(c, "轻笺").Bold().FontSize(16).Margin(0, 12, 0, 0)
			if ui.Button(c, "新建").Clicked() {
				a.NewDocument("markdown")
			}
			if ui.Button(c, "打开").Clicked() {
				a.ChooseOpen()
			}
			a.closedFilesMenu(c)
			if ui.Button(c, "文件夹").Clicked() {
				a.ChooseFolder()
			}
			if ui.Button(c, "保存").Disabled(a.Current() == nil).Clicked() {
				a.Save(a.Current(), false)
			}
			ui.Spacer(c)
			if ui.Button(c, "查找").Disabled(!a.Current().textCapable()).KeepFocus().Clicked() {
				a.toggleFind()
			}
			if ui.Button(c, "插件").Clicked() {
				a.closeGoToLine()
				a.ShowPlugins = !a.ShowPlugins
			}
			if ui.Button(c, "设置").Clicked() {
				a.ShowSettings = !a.ShowSettings
			}
		})
		ui.Divider(c)
		ui.Row(c).Grow(1).AlignItems(ui.Stretch).Children(func() {
			a.sidebar(c)
			// The fixed status row remains visually authoritative when a rich
			// editor is squeezed below its preferred content height.
			ui.Column(c.Key("main")).Grow(1).MinWidth(300).ClipY().Children(func() {
				if len(a.Tabs) > 0 {
					ui.Row(c.Key("tab-navigation")).Shrink(0).Children(func() {
						ui.Column(c).Grow(1).MinWidth(0).Children(func() { a.tabs(c) })
						if ui.Button(c, "已打开文档").Height(30).Margin(4, 6).
							Tooltip("搜索已打开的文档 · Ctrl+P").KeepFocus().Clicked() {
							a.openDocumentPicker()
						}
					})
					ui.Divider(c)
				} else {
					a.revealedTabID = 0
				}
				if a.exportJob != nil {
					ui.Row(c.Key("export-progress")).Height(36).Shrink(0).Padding(4, 12).Gap(8).Children(func() {
						ui.Text(c, "正在导出…").FontSize(12).Grow(1)
						if ui.Button(c, "取消导出").Height(28).Clicked() {
							a.cancelEngineExport("已取消导出")
						}
					})
				}
				a.packageProgress(c)
				if a.ShowPlugins {
					a.clearFindTransient()
					a.pluginPanel(c)
					return
				}
				if a.ShowFind {
					a.findbar(c)
				}
				if a.ShowSettings {
					a.settings(c)
				}
				if a.ShowGoToLine {
					a.goToLineBar(c)
				}
				if a.ShowIssues {
					a.issues(c)
				}
				if a.ShowWorkspaceSearch {
					a.workspaceResults(c)
				}
				if t := a.Current(); t != nil {
					if t.Conflict {
						a.conflictBar(c, t)
					}
					if t.Warning != "" {
						ui.Text(c, t.Warning).FontSize(12).TextColor(theme.Warning).Padding(10, 18)
					}
					if t.FormatWarning != "" {
						ui.Text(c, t.FormatWarning).FontSize(12).TextColor(theme.Warning).Padding(10, 18)
					}
					if t.SyntaxNotice != "" {
						ui.Text(c, t.SyntaxNotice).FontSize(11).TextColor(theme.TextMuted).Padding(7, 18)
					}
					if t.ByteInspector && t.CanReinterpret {
						a.encodingChoices(c, t)
					}
					a.documentView(c, t)
				} else {
					a.welcome(c)
				}
			})
		})
		ui.Divider(c)
		width, _ := c.Size()
		compactStatus := width < 820 && a.Message != ""
		ui.Row(c.Key("status")).Height(29).Padding(4, 14).Gap(14).Clip().Children(func() {
			if a.Message != "" {
				ui.Text(c, a.Message).FontSize(11).Grow(1).Shrink(1).MaxWidth(650).SingleLine()
			} else {
				ui.Text(c, "本地文件 · 原生编辑").FontSize(11).TextColor(theme.TextMuted).Grow(1)
			}
			if t := a.Current(); t != nil && !compactStatus {
				if t.isBinary() {
					a.binaryStatus(c, t)
				} else {
					line, col, _ := t.Doc.Buffer.LineColumn(t.Doc.Buffer.Selection().Head)
					position := fmt.Sprintf("第 %d 行，第 %d 列", line+1, col+1)
					if ui.Button(c, position).
						Label("转到源码行；当前" + position).
						KeepFocus().
						Height(24).
						Clicked() {
						a.openGoToLine()
					}
					encodingLabel := string(t.Doc.Encoding)
					if t.ByteInspector {
						encodingLabel = "二进制（字节视图）"
						if t.CanReinterpret {
							encodingLabel = "原始编码未知（字节视图）"
						}
					}
					ui.Text(c, encodingLabel).FontSize(11)
					state := "已保存"
					if t.Doc.ReadOnly {
						state = "只读"
					} else if t.Doc.Dirty() {
						state = "未保存"
					}
					ui.Text(c, state).FontSize(11).TextColor(theme.TextMuted)
				}
			}
			if len(a.Issues) > 0 && ui.Button(c, fmt.Sprintf("问题 %d", len(a.Issues))).Height(24).Clicked() {
				a.ShowIssues = !a.ShowIssues
			}
		})
	})
	a.openDocumentsDialog(c)
}
func (a *App) sidebar(c *ui.Context) {
	t := c.Theme()
	ui.Column(c.Key("sidebar")).Width(a.SidebarWidth).FillHeight().Background(t.Surface).Border(1, t.Border).Children(func() {
		ui.Row(c).Padding(10, 8).Gap(5).Children(func() {
			label := "工作区"
			if a.ShowOutline {
				label = "文档大纲"
			} else if a.Root != "" {
				label = filepath.Base(a.Root)
			}
			ui.Text(c, label).Bold().Grow(1).SingleLine()
			outlineLabel := "大纲"
			if a.ShowOutline {
				outlineLabel = "文件"
			}
			if ui.Button(c, outlineLabel).Label("切换文件树与文档大纲").Disabled(a.Current().isBinary() && !a.ShowOutline).Height(25).Clicked() {
				a.setOutlineVisible(!a.ShowOutline)
			}
			if !a.ShowOutline && ui.Button(c, "↻").Label("刷新文件树").Height(25).Width(28).Clicked() {
				a.refreshTree()
			}
		})
		if a.ShowOutline {
			a.outlineSidebar(c)
		} else if a.Root == "" {
			ui.Column(c).Padding(12).Gap(12).Children(func() {
				ui.Text(c, "打开一个文件夹，\n把资料放在一起。").FontSize(12).TextColor(t.TextMuted)
				if ui.Button(c, "打开文件夹").Clicked() {
					a.ChooseFolder()
				}
			})
		} else {
			if a.TreeTruncated {
				ui.Text(c, fmt.Sprintf("文件树按目录最多显示 %d 项，总计最多 %d 项；其余内容未载入。", treeDirectoryEntryLimit, treeEntryLimit)).FontSize(11).TextColor(t.Warning).Padding(8, 12)
			}
			entries := a.Entries
			if len(entries) == 0 {
				ui.Text(c, "此文件夹为空。").FontSize(12).TextColor(t.TextMuted).Padding(10, 12)
			} else {
				a.TreeList.Selected = &a.TreeSelected
				a.TreeList.Key = func(i int) any {
					if i < 0 || i >= len(entries) {
						return ""
					}
					return entries[i].Path
				}
				a.TreeList.Label = func(i int) string {
					if i < 0 || i >= len(entries) {
						return ""
					}
					return entries[i].Name
				}
				list := ui.List(c, &a.TreeList, len(entries), func(i int) {
					if i < 0 || i >= len(entries) {
						return
					}
					if a.treeRowBuilt != nil {
						a.treeRowBuilt(i)
					}
					entry := entries[i]
					prefix := "  "
					if entry.Directory {
						prefix = "› "
						if a.Expanded[entry.Path] {
							prefix = "⌄ "
						}
					}
					ui.Row(c).FillWidth().Height(29).Padding(4, 10+float32(entry.Depth)*14).Children(func() {
						ui.Text(c, prefix+entry.Name)
					})
				}).Grow(1)
				if list.Submitted() {
					a.activateTreeEntry(a.TreeSelected)
				}
			}
		}
		ui.Spacer(c)
		ui.Divider(c)
		ui.Column(c).Padding(12).Gap(5).Children(func() {
			ui.Text(c, "新建文档").FontSize(11).TextColor(t.TextMuted)
			for _, item := range []struct{ label, kind string }{{"Markdown", "markdown"}, {"纯文本", "text"}, {"思维导图", "mindmap"}, {"图纸", "drawio"}} {
				if ui.Button(c, item.label).FillWidth().Clicked() {
					a.NewDocument(item.kind)
				}
			}
		})
	})
}

func (a *App) activateTreeEntry(i int) {
	if i < 0 || i >= len(a.Entries) {
		return
	}
	a.TreeSelected = i
	entry := a.Entries[i]
	if entry.Directory {
		a.Expanded[entry.Path] = !a.Expanded[entry.Path]
		a.refreshTree()
		return
	}
	a.Open(entry.Path)
}
func (a *App) tabs(c *ui.Context) {
	var activeButton ui.Element
	activeButtonID, closeIndex := 0, -1
	ui.ScrollHorizontal(c.Key("tabs")).Height(38).Shrink(0).Children(func() {
		ui.Row(c).Gap(1).Children(func() {
			for i, t := range a.Tabs {
				i, t := i, t
				label := t.Name
				if t.Doc != nil && !t.Doc.ReadOnly && t.Doc.Dirty() {
					label = "• " + label
				}
				ui.Row(c.Key(t.ID)).Padding(3, 4).Gap(2).Children(func() {
					b := ui.Button(c, label).Height(30)
					if i == a.Active {
						b.Background(c.Theme().Selection).TextColor(c.Theme().Text)
					}
					if b.Clicked() {
						a.Select(i)
					}
					if current := a.Current(); current != nil && current.ID == t.ID {
						activeButton, activeButtonID = b, t.ID
					}
					if ui.Button(c, "×").Label("关闭 " + t.Name).Height(26).Width(24).Clicked() {
						closeIndex = i
					}
				})
			}
		})
	})
	// Closing compacts and clears the live backing slice. Finish building it
	// first, then let the next pass choose the surviving active button.
	if closeIndex >= 0 && a.CloseTab(closeIndex) {
		c.Invalidate()
		return
	}
	// Resolve after every click has run: an earlier button may have become
	// active. Keep only its ID across frames so manual scrolling stays free.
	if current := a.Current(); current != nil && current.ID == activeButtonID && current.ID != a.revealedTabID {
		activeButton.ScrollIntoView()
		a.revealedTabID = current.ID
	}
}
func (a *App) welcome(c *ui.Context) {
	ui.Column(c).Grow(1).Center().Gap(16).Children(func() {
		ui.Text(c, "留一点空间给想法。").FontSize(28).Bold()
		ui.Text(c, "在同一个工作区阅读、记录和整理。").TextColor(c.Theme().TextMuted)
		ui.Row(c).Gap(10).Children(func() {
			if ui.PrimaryButton(c, "打开文件").Clicked() {
				a.ChooseOpen()
			}
			if ui.Button(c, "新建 Markdown").Clicked() {
				a.NewDocument("markdown")
			}
		})
		ui.Text(c, "Ctrl+O 打开    Ctrl+N 新建    Ctrl+S 保存").FontSize(12).TextColor(c.Theme().TextMuted)
	})
}
func (a *App) settings(c *ui.Context) {
	ui.Column(c).Padding(10, 18).Gap(7).Background(c.Theme().Surface).Children(func() {
		ui.Row(c).Gap(8).Children(func() {
			ui.Text(c, "外观").Bold().Grow(1)
			if ui.Button(c, "关闭设置").Clicked() {
				a.ShowSettings = false
			}
		})
		ui.Row(c).Gap(8).Wrap().Children(func() {
			if ui.Button(c, "切换明暗").Clicked() {
				a.Dark = !a.Dark
				a.markPersistPending()
			}
			if ui.Button(c, "A−").Clicked() {
				a.font(-1)
			}
			ui.Text(c, fmt.Sprintf("%g px", a.FontSize))
			if ui.Button(c, "A+").Clicked() {
				a.font(1)
			}
			lineLabel := "显示行号"
			if !a.LineNumbers {
				lineLabel = "隐藏行号"
			}
			if ui.Button(c, lineLabel).Clicked() {
				a.LineNumbers = !a.LineNumbers
				a.applyEditorPreferences()
				a.Persist()
			}
			wrapLabel := "自动换行"
			if !a.SoftWrap {
				wrapLabel = "关闭换行"
			}
			if ui.Button(c, wrapLabel).Clicked() {
				a.SoftWrap = !a.SoftWrap
				a.applyEditorPreferences()
				a.Persist()
			}
		})
	})
}
func (a *App) pluginPanel(c *ui.Context) {
	ui.Column(c.Key("plugins-panel")).Grow(1).MinHeight(0).Children(func() {
		ui.Row(c.Key("plugin-kinds")).Padding(5, 18).Gap(8).Height(38).Children(func() {
			if ui.Button(c, "格式配置").Label("查看格式配置").Height(28).Clicked() {
				a.showEnginePackages = false
			}
			if ui.Button(c, "引擎包").Label("查看本地引擎包").Height(28).Clicked() {
				a.showEnginePackages = true
			}
		})
		if a.showEnginePackages {
			a.enginePackagePanel(c.Key("engine-packages"))
		} else {
			a.formatPluginPanel(c.Key("format-config"))
		}
	})
}

func (a *App) formatPluginPanel(c *ui.Context) {
	ui.Column(c).Grow(1).Padding(12, 18).Gap(7).Background(c.Theme().Surface).Border(1, c.Theme().Border).Children(func() {
		ui.Row(c).Gap(8).Children(func() {
			ui.Text(c, "格式插件").Bold().FontSize(15).Grow(1)
			if ui.Button(c, "安装").Clicked() {
				a.installPlugin()
			}
			if ui.Button(c, "重新扫描").Clicked() {
				a.refreshPlugins()
			}
			if ui.Button(c, "安装目录").Clicked() {
				a.Message = "插件安装目录：" + filepath.Join(a.configDir, "plugins")
			}
			if ui.Button(c, "关闭插件").Clicked() {
				a.ShowPlugins = false
			}
		})
		ui.Text(c, "本地 JSON 配置不执行插件代码。可配置高亮、逐行源码大纲，或使用内置 Markdown / 思维导图；未匹配时显示源码，只读文件可另存副本。").FontSize(11).TextColor(c.Theme().TextMuted)
		if a.pluginManager == nil {
			ui.Text(c, "插件管理器不可用。").TextColor(c.Theme().Warning)
			return
		}
		ui.Scroll(c).Grow(1).Children(func() {
			for index, manifest := range a.pluginManager.Manifests() {
				manifest := manifest
				status := "可用"
				if manifest.Builtin {
					status = "内置启用"
					if !manifest.Enabled {
						status = "内置停用"
					}
				} else if manifest.Enabled && manifest.Active {
					status = "已启用"
				} else if manifest.Enabled {
					status = "载入失败"
				} else {
					status = "已停用"
				}
				ui.Column(c.Key(index)).Padding(4, 0).Gap(3).Children(func() {
					ui.Row(c).Gap(8).Children(func() {
						ui.Text(c, manifest.Name).Bold().Grow(1)
						ui.Text(c, manifest.ID).FontSize(11).TextColor(c.Theme().TextMuted).Width(105)
						ui.Text(c, status).FontSize(11).TextColor(c.Theme().TextMuted)
						label := "启用"
						if manifest.Enabled {
							label = "停用"
						}
						if ui.Button(c, label).Label(label + "插件 " + manifest.Name).Height(24).Clicked() {
							a.setPluginEnabled(manifest.ID, !manifest.Enabled)
						}
					})
					ui.Text(c, extensionLines(manifest.Extensions)).FillWidth().FontSize(11).TextColor(c.Theme().TextMuted)
					if manifest.Builtin {
						if adapter, ok := a.adapterByID(manifest.ID); ok && adapter.ExportEngine != nil {
							ui.Text(c, a.exportStatusText(*adapter.ExportEngine)).FillWidth().FontSize(11).TextColor(c.Theme().TextMuted)
						}
					}
					if manifest.Declarative && a.pluginManager != nil {
						if syntax, ok := a.pluginManager.Syntax(manifest.ID); ok {
							if syntax.Schema == 3 {
								ui.Row(c).Gap(8).Children(func() {
									ui.Text(c, "使用 "+nativeRouteLabel(syntax.NativeRoute)).Grow(1).FontSize(12).
										Tooltip(manifest.Name + " · " + manifest.ID + " · schema3 · " + string(syntax.NativeRoute))
									if ui.Button(c, "新建文档").Label("使用 " + manifest.Name + " 新建文档").Height(27).
										Disabled(!manifest.Enabled || !manifest.Active).Clicked() {
										a.NewFromPlugin(manifest.ID)
									}
								})
							} else {
								ui.Text(c, declarativeCapability(syntax)).FillWidth().FontSize(11).TextColor(c.Theme().TextMuted)
							}
						}
					}
				})
			}
			for i, issue := range a.pluginManager.LoadErrors() {
				ui.Text(c.Key(fmt.Sprintf("plugin-error-%d", i)), "载入错误 · "+filepath.Base(issue.Path)+" · "+issue.Message).FontSize(11).TextColor(c.Theme().Warning)
			}
		})
	})
}

func extensionLines(extensions []string) string {
	const perLine = 5
	var text strings.Builder
	for i, extension := range extensions {
		if i > 0 {
			if i%perLine == 0 {
				text.WriteByte('\n')
			} else {
				text.WriteString("  ")
			}
		}
		text.WriteString(extension)
	}
	return text.String()
}

func (a *App) font(delta float32) {
	a.FontSize = max(10, min(32, a.FontSize+delta))
	a.applyEditorPreferences()
	a.Persist()
}
func (a *App) issues(c *ui.Context) {
	ui.Scroll(c).Height(150).Padding(12).Children(func() {
		limitText := fmt.Sprintf("问题清单只保留本次运行最近 %d 条（当前 %d 条）。", issueHistoryLimit, len(a.Issues))
		if a.IssuesDropped > 0 {
			limitText += fmt.Sprintf("另有 %d 条较早记录未保留。", a.IssuesDropped)
		}
		ui.Text(c, limitText).FontSize(11).TextColor(c.Theme().TextMuted)
		if current := a.Current(); current != nil && len(current.ViewWarnings) > 0 {
			ui.Text(c, "当前格式的提示").Bold().FontSize(12)
			for i, warning := range current.ViewWarnings {
				ui.Text(c.Key(fmt.Sprintf("view-warning-%d", i)), warning).FontSize(12).TextColor(c.Theme().Warning)
			}
		}
		if len(a.SearchIssues) > 0 {
			searchIssueText := fmt.Sprintf("工作区搜索跳过项（显示 %d 项", len(a.SearchIssues))
			if a.SearchIssuesDropped > 0 {
				searchIssueText += fmt.Sprintf("，另有 %d 项未保留", a.SearchIssuesDropped)
			}
			searchIssueText += "）"
			ui.Text(c, searchIssueText).Bold().FontSize(12).TextColor(c.Theme().Warning)
			for i, issue := range a.SearchIssues {
				detail := issue.Path + " · " + issue.Reason
				if issue.Detail != "" {
					detail += " · " + issue.Detail
				}
				ui.Text(c.Key(fmt.Sprintf("search-issue-%d", i)), detail).FontSize(12)
			}
		}
		for i, issue := range a.Issues {
			ui.Text(c.Key(i), issue).FontSize(12)
		}
		if ui.Button(c, "收起问题").Clicked() {
			a.ShowIssues = false
		}
	})
}
func (a *App) shortcuts(c *ui.Context) {
	if c.Shortcut(ui.Cmd, ui.KeyP) {
		a.openDocumentPicker()
	}
	if c.Shortcut(ui.Cmd, ui.KeyN) {
		a.NewDocument("markdown")
	}
	if c.Shortcut(ui.Cmd, ui.KeyO) {
		a.ChooseOpen()
	}
	if c.Shortcut(ui.Cmd, ui.KeyS) {
		a.Save(a.Current(), false)
	}
	if c.Shortcut(ui.Cmd|ui.Shift, ui.KeyS) {
		a.Save(a.Current(), true)
	}
	if c.Shortcut(ui.Cmd, ui.KeyW) {
		a.CloseTab(a.Active)
	}
	if c.Shortcut(ui.Cmd|ui.Shift, ui.KeyT) && !c.ShortcutRepeated() {
		a.reopenClosedFile(0)
	}
	if c.Shortcut(ui.Cmd, ui.KeyTab) {
		a.selectAdjacentTab(true)
	}
	if c.Shortcut(ui.Cmd|ui.Shift, ui.KeyTab) {
		a.selectAdjacentTab(false)
	}
	if c.Shortcut(ui.Cmd, ui.KeyF) {
		a.openFind()
	}
	if c.Shortcut(ui.Cmd, ui.KeyG) && !a.ShowGoToLine {
		a.openGoToLine()
	}
	// The focused numeric field must receive Escape so an active IME can
	// cancel its candidates without closing the panel underneath it.
	findOwnsEscape := a.ShowFind && a.findPanel.FocusWithin(c)
	if !a.ShowGoToLine && !findOwnsEscape && c.Shortcut(0, ui.KeyEscape) {
		if a.ShowFind {
			a.cancelFind()
		}
		a.ShowSettings = false
		a.ShowIssues = false
		a.ShowPlugins = false
	}
}
func (a *App) documentView(c *ui.Context, t *Tab) {
	if t.isBinary() {
		a.binaryDocumentView(c, t)
		return
	}
	if t.Adapter == nil {
		if t.MatchedPlugin.ID != "" && (t.MatchedPlugin.Declarative || t.Editor.Highlight != nil) {
			label := t.MatchedPlugin.Name + " · " + t.MatchedPlugin.ID
			ui.Row(c).Height(30).Shrink(0).Padding(4, 16).Gap(8).Clip().Children(func() {
				ui.Text(c, label).FontSize(12).TextColor(c.Theme().TextMuted).
					SingleLine().Tooltip(label).Grow(1)
				ui.Text(c, t.SourceCapabilityBadge).FontSize(11).TextColor(c.Theme().Text).
					SingleLine().Tooltip(t.SourceCapabilityDetail).Padding(2, 8).
					Background(c.Theme().Surface).Border(1, c.Theme().Border).Radius(4)
			})
		}
		t.Editor.View(c.Key(t.ID))
		return
	}
	ui.Row(c).Height(38).Shrink(0).Padding(5, 16).Gap(6).Children(func() {
		name, detail := t.Adapter.Name, t.Adapter.Name
		if t.MatchedPlugin.Declarative {
			name = t.SourceCapabilityBadge
			detail = t.MatchedPlugin.Name + " · " + t.MatchedPlugin.ID + " · " + t.SourceCapabilityDetail
		}
		ui.Text(c, name).FontSize(12).TextColor(c.Theme().TextMuted).Grow(1).SingleLine().Tooltip(detail)
		for _, mode := range []struct{ id, label string }{{"preview", "阅读 / 编辑"}, {"source", "源码"}, {"split", "分屏"}} {
			b := ui.Button(c, mode.label).Height(28)
			if mode.id == t.Mode {
				b.Background(c.Theme().Selection)
			}
			if b.Clicked() {
				if mode.id == t.Mode {
					return
				}
				a.setMode(t, mode.id)
			}
		}
		if len(t.Adapter.Exports) > 0 {
			label, access, tooltip := "导出", "导出", ""
			disabled := a.exportJob != nil
			if capability := t.Adapter.ExportEngine; capability != nil && capability.RequiresView {
				tooltip = "导出当前选择的图纸页面（SVG）"
				label, access = "导出页面", "导出图纸当前页面"
				if !exportViewModeValid(t.Mode) {
					label, access = "选择页面后导出", "选择页面后导出"
					tooltip = "图纸导出需要阅读或分屏视图；请切换到阅读或分屏并选择页面"
					disabled = true
				} else if t.View == nil {
					tooltip = "图纸视图尚未准备好；请等待阅读或分屏视图显示后选择页面"
					disabled = true
				}
			}
			button := ui.Button(c, label).Label(access).Height(28).Disabled(disabled)
			if tooltip != "" {
				button.Tooltip(tooltip)
			}
			if button.Clicked() {
				a.exportCurrent()
			}
		}
	})
	if t.Mode == "source" {
		t.Editor.View(c.Key(t.ID))
		return
	}
	if t.View == nil {
		t.View = t.Adapter.Create(a.editorHost(t))
	}
	warnings := t.View.Warnings()
	t.ViewWarnings = append(t.ViewWarnings[:0], warnings...)
	if len(warnings) > 0 {
		ui.Row(c.Key("format-warnings")).Padding(7, 18).Gap(8).Children(func() {
			message := warnings[0]
			if len(warnings) > 1 {
				message += fmt.Sprintf("（另有 %d 项）", len(warnings)-1)
			}
			ui.Text(c, message).FontSize(11).TextColor(c.Theme().Warning).Grow(1)
			if len(warnings) > 1 && ui.Button(c, "查看详情").Height(24).Clicked() {
				a.ShowIssues = true
			}
		})
	}
	if t.Mode == "split" {
		ui.Row(c.Key("document-split")).Grow(1).AlignItems(ui.Stretch).Children(func() {
			ui.Column(c).WidthPercent(50).Children(func() { t.Editor.View(c.Key(t.ID)) })
			ui.Column(c).Grow(1).Border(1, c.Theme().Border).Children(func() { t.View.Draw(c.Key(t.ID)) })
		})
	} else {
		// The first warning can appear while native input is composing. A
		// conditional sibling must not change the view's identity and drop
		// its focus, marked input or inspector scroll position.
		t.View.Draw(c.Key(t.ID))
	}
}

func (a *App) setMode(t *Tab, mode string) bool {
	if t == nil || mode == t.Mode {
		return t != nil
	}
	if t.isBinary() && mode != "source" && mode != "preview" {
		a.Message = "图片只提供图片阅读与只读字节视图"
		return false
	}
	if mode != "source" && (t.Adapter == nil || t.Adapter.Create == nil) {
		a.report(errors.New("当前格式只提供源码编辑，没有阅读或分屏视图。"))
		return false
	}
	if !a.flushTab(t, "切换格式视图") {
		return false
	}
	a.releaseTabPresentation(t, t.isBinary() && t == a.Current())
	t.Mode = mode
	a.markPersistPending()
	return true
}

func (a *App) applyEditorPreferences() {
	for _, t := range a.Tabs {
		t.Editor.SetFontSize(a.FontSize)
		if t.isBinary() {
			t.Editor.SetSoftWrap(false)
			t.Editor.LineNumbers = false
			continue
		}
		t.Editor.SetSoftWrap(a.SoftWrap)
		t.Editor.LineNumbers = a.LineNumbers
	}
}

func (a *App) encodingChoices(c *ui.Context, t *Tab) {
	ui.Column(c).Padding(7, 18).Gap(5).Background(c.Theme().Surface).Children(func() {
		ui.Text(c, "原始编码无法自动识别。明确选择后重新打开：").FontSize(11)
		ui.Row(c).Gap(6).Wrap().Children(func() {
			for _, encoding := range []document.Encoding{document.UTF8, document.UTF16LE, document.UTF16BE, document.GBK, document.GB18030} {
				encoding := encoding
				if ui.Button(c, string(encoding)).Height(25).Clicked() {
					a.reopenWithEncoding(t, encoding)
				}
			}
		})
	})
}
