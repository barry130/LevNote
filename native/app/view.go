package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/text"
)

func (a *App) View(c *ui.Context) {
	a.beginAppListFrame()
	a.beginSourceLanguageFrame()
	if binding := a.sourceLanguage.shortcut; binding != nil && !a.sourceLanguageShortcutCurrent(binding) {
		a.revokeSourceLanguageShortcut(nil)
	}
	a.deliverBinary()
	a.expireFindCache()
	a.deliverPackageWork()
	if a.exportJob != nil {
		a.deliverEngineExport(a.exportJob)
	}
	a.deliverScene()
	style := workbenchAppearance(a.Dark)
	theme := style.Theme
	c.SetTheme(theme)
	a.expireGoToLine()
	a.expireFindFocus()
	a.processOpenDocuments()
	a.processCommandPalette()
	a.processSourceLanguageActions()
	if a.ShowPlugins && a.ShowFind {
		a.clearFindTransient()
	}
	if a.ShowPlugins && a.ShowGoToLine {
		a.closeGoToLine()
	}
	a.shortcuts(c)
	a.cancelHiddenListFocus()
	a.cancelHiddenSourceLanguageListFocus()
	a.consumeHistoryNotices()
	ui.Column(c).Fill().Background(theme.Background).Children(func() {
		a.workbenchToolbar(c)
		ui.Row(c).Grow(1).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
			a.workbenchActivityBar(c)
			a.sidebar(c)
			// The fixed status row remains visually authoritative when a rich
			// editor is squeezed below its preferred content height.
			ui.Column(c.Key("main")).Grow(1).MinWidth(0).MinHeight(0).ClipY().Children(func() {
				if len(a.Tabs) > 0 {
					ui.Row(c.Key("tab-navigation")).Height(workbenchTabsHeight).Shrink(0).AlignItems(ui.Center).Background(style.TabBar).Children(func() {
						ui.Column(c).Grow(1).MinWidth(0).Children(func() { a.tabs(c) })
						if workbenchButton(c, "已打开文档").Height(32).Shrink(0).Margin(6, 8).
							Tooltip("搜索已打开的文档 · Ctrl+P").KeepFocus().Clicked() {
							a.openDocumentPicker()
						}
					})
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
					// The palette or activity bar may have opened this panel in this frame.
					a.revokeSourceLanguageShortcut(nil)
					a.clearFindTransient()
					a.pluginPanel(c)
					return
				}
				a.sourceLanguagePanel(c)
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
					if t.textCapable() && t.Doc.Buffer != nil && t.Doc.Buffer.History().Truncated {
						ui.Row(c.Key("history-budget-notice")).Height(30).Shrink(0).Padding(7, 18).Children(func() {
							ui.Text(c, historyBudgetNotice).FontSize(11).TextColor(theme.Warning).
								SingleLine().Tooltip(historyBudgetNotice).Grow(1)
						})
					}
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
		a.workbenchStatus(c)
	})
	a.openDocumentsDialog(c)
	a.commandPaletteDialog(c)
}
func (a *App) sidebar(c *ui.Context) {
	s := workbenchAppearance(a.Dark)
	t := c.Theme()
	ui.Column(c.Key("sidebar")).Width(a.workbenchSidebarWidth(c)).Shrink(0).FillHeight().MinHeight(0).
		Background(s.Sidebar).BorderWidth(0, 1, 0, 0).BorderColor(t.Border).Children(func() {
		ui.Row(c).Height(workbenchTabsHeight).Shrink(0).Padding(6, 12).Gap(4).AlignItems(ui.Center).Children(func() {
			label := "工作区"
			if a.ShowOutline {
				label = "文档大纲"
			} else if a.Root != "" {
				label = filepath.Base(a.Root)
			}
			ui.Text(c, label).FontSize(14).TextColor(t.Text).Grow(1).MinWidth(0).SingleLine().Tooltip(label)
			outlineLabel := "大纲"
			if a.ShowOutline {
				outlineLabel = "文件"
			}
			if workbenchButton(c, outlineLabel).Label("切换文件树与文档大纲").Disabled(a.Current().isBinary() && !a.ShowOutline).Height(32).Shrink(0).Clicked() {
				a.setOutlineVisible(!a.ShowOutline)
			}
			if !a.ShowOutline && workbenchButton(c, "↻").Label("刷新文件树").Height(32).Width(28).Shrink(0).Clicked() {
				a.refreshTree()
			}
		})
		if a.ShowOutline {
			a.outlineSidebar(c)
		} else if a.Root == "" {
			ui.Column(c).Padding(16, 12).Gap(14).Children(func() {
				ui.Text(c, "打开一个文件夹，\n把资料放在一起。").FontSize(13).TextColor(t.TextMuted)
				if ui.PrimaryButton(c, "打开文件夹").Height(32).Radius(6).FontSize(13).FillWidth().Clicked() {
					a.ChooseFolder()
				}
			})
		} else {
			if a.TreeTruncated {
				ui.Text(c, fmt.Sprintf("文件树按目录最多显示 %d 项，总计最多 %d 项；其余内容未载入。", treeDirectoryEntryLimit, treeEntryLimit)).FontSize(11).TextColor(t.Warning).Padding(8, 12)
			}
			entries := a.Entries
			if len(entries) == 0 {
				a.TreeList.Handle.CancelFocus()
				ui.Text(c, "此文件夹为空。").FontSize(13).TextColor(t.TextMuted).Padding(10, 12)
			} else {
				root, generation := a.Root, a.treeGeneration
				current := func() bool {
					return !a.shuttingDown.Load() && !a.ShowOutline && a.Root == root &&
						a.treeRoot == root && a.treeGeneration == generation
				}
				// List input may finish after a row callback changes the root. A
				// pass-local choice cannot write back into the replacement model.
				selected := a.TreeSelected
				a.TreeList.Selected = &selected
				// A built frame still owns all captured row identities after a
				// root change. Content and actions, not structural keys, expire.
				a.TreeList.Key = func(i int) any {
					if i < 0 || i >= len(entries) {
						return i
					}
					return entries[i].Path
				}
				a.TreeList.Label = func(i int) string {
					if !current() || i < 0 || i >= len(entries) {
						return ""
					}
					return entries[i].Name
				}
				a.treeListLifetime.built = true
				list := ui.List(c, &a.TreeList, len(entries), func(i int) {
					if !current() || i < 0 || i >= len(entries) {
						return
					}
					if a.treeRowBuilt != nil {
						a.treeRowBuilt(i)
					}
					if !current() {
						return
					}
					entry := entries[i]
					ui.Row(c).FillWidth().Height(32).Padding(6, 12, 6, 12+float32(entry.Depth)*14).Gap(6).AlignItems(ui.Center).Children(func() {
						if entry.Directory {
							chevron := ui.Icon(c, workbenchChevronIcon).Size(10, 10).TextColor(s.Subtle)
							if a.Expanded[entry.Path] {
								chevron.Rotate(90)
							}
							ui.Icon(c, workbenchFolderIcon).Size(16, 16).TextColor(s.Subtle)
						} else {
							ui.Box(c).Width(10).Shrink(0)
							ui.Icon(c, workbenchFileIcon(entry.Name)).Size(16, 16).TextColor(t.Accent)
						}
						name := ui.Text(c, entry.Name).FontSize(13).SingleLine().Grow(1).MinWidth(0).Tooltip(entry.Path)
						if entry.Directory {
							prefix := "› "
							if a.Expanded[entry.Path] {
								prefix = "⌄ "
							}
							name.Label(prefix + entry.Name)
						}
					})
				}).Grow(1)
				if current() {
					a.TreeSelected = selected
					if list.Submitted() && selected >= 0 && selected < len(entries) &&
						selected < len(a.Entries) && a.Entries[selected].Path == entries[selected].Path {
						a.activateTreeEntry(selected)
					}
				}
			}
		}
		if !a.ShowOutline && (a.Root == "" || len(a.Entries) == 0) {
			ui.Spacer(c)
		}
		ui.Column(c).Padding(0, 12, 12, 12).Gap(2).Shrink(0).Children(func() {
			ui.Row(c).Height(1).FillWidth().Margin(0, 0, 8, 0).Background(s.Theme.Border)
			ui.Text(c, "新建文档").FontSize(11).TextColor(s.Subtle).Padding(2, 8, 4, 8).FillWidth()
			items := []struct {
				label, kind string
				icon        *ui.SVG
			}{
				{"Markdown", "markdown", workbenchMarkdownIcon},
				{"纯文本", "text", workbenchDocumentIcon},
				{"思维导图", "mindmap", workbenchDiagramIcon},
				{"图纸", "drawio", workbenchShapesIcon},
			}
			for _, item := range items {
				b := workbenchButton(c.Key("sidebar-new-"+item.kind), "").Label(item.label).
					Tooltip("新建 "+item.label).Height(30).Padding(0, 8).Gap(8).Radius(6).FillWidth().
					Justify(ui.Start).AlignItems(ui.Center)
				b.Children(func() {
					ui.Icon(c, item.icon).Size(14, 14).Shrink(0).TextColor(s.Subtle)
					ui.Text(c, item.label).FontSize(13).SingleLine()
				})
				if b.Clicked() {
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
	s := workbenchAppearance(a.Dark)
	var activeButton ui.Element
	activeButtonID, closeIndex := 0, -1
	ui.ScrollHorizontal(c.Key("tabs")).Height(workbenchTabsHeight).Shrink(0).Padding(6, 8).Background(s.TabBar).Children(func() {
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
			for i, t := range a.Tabs {
				i, t := i, t
				label := t.Name
				if t.Doc != nil && !t.Doc.ReadOnly && t.Doc.Dirty() {
					label = "• " + label
				}
				tab := ui.Row(c.Key(t.ID)).Height(32).Shrink(0).Padding(0, 3).Gap(0).AlignItems(ui.Center).Radius(8)
				if i == a.Active {
					tab.Background(s.TabActive)
				}
				tab.Children(func() {
					b := workbenchButton(c, label).Height(32).Radius(6).MaxWidth(230).Padding(5, 8, 5, 29).Tooltip(t.Name).
						Draw(func(p *ui.Painter, r ui.Rect) {
							p.Icon(workbenchFileIcon(t.Name), ui.Rect{X: r.X + 6, Y: r.Y + (r.H-16)/2, W: 16, H: 16}, s.Theme.Focus)
						})
					if i == a.Active {
						b.TextColor(s.Theme.Text)
					} else {
						b.TextColor(s.Subtle)
					}
					if b.Clicked() {
						a.Select(i)
					}
					if current := a.Current(); current != nil && current.ID == t.ID {
						activeButton, activeButtonID = b, t.ID
					}
					if workbenchButton(c, "×").Label("关闭 " + t.Name).Height(26).Width(26).Shrink(0).Padding(2).Clicked() {
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
	s := workbenchAppearance(a.Dark)
	width, height := c.Size()
	inset, top := float32(28), float32(24)
	if width-workbenchActivityWidth-a.workbenchSidebarWidth(c) >= 720 && height >= 640 {
		inset, top = 56, 60
	}
	ui.Scroll(c.Key("welcome")).Grow(1).MinHeight(0).Children(func() {
		ui.Column(c).FillWidth().MaxWidth(800).Padding(top, inset, 28, inset).Gap(8).Children(func() {
			ui.Text(c, "LevNote").FontSize(16).TextColor(s.Subtle)
			ui.Text(c, "留一点空间给想法。").FontSize(30).Bold().TextColor(s.Theme.Text)
			ui.Text(c, "在同一个工作区阅读、记录和整理。").FontSize(13).TextColor(s.Subtle)
			ui.Text(c, "开始").FontSize(14).Margin(8, 0, 4, 0)
			ui.Row(c).FillWidth().MaxWidth(460).Gap(12).Wrap().Children(func() {
				ui.Column(c).Grow(1).MinWidth(160).Gap(8).Children(func() {
					if ui.PrimaryButton(c, "新建 Markdown").FillWidth().Height(44).Shrink(0).Padding(10, 16).Radius(10).FontSize(14).Clicked() {
						a.NewDocument("markdown")
					}
					ui.Text(c, "Ctrl+N").FontSize(12).TextColor(s.Subtle)
				})
				ui.Column(c).Grow(1).MinWidth(160).Gap(8).Children(func() {
					open := workbenchButton(c, "打开文件").FillWidth().Height(44).Shrink(0).Padding(10, 16).Radius(10).FontSize(14).
						Border(1, s.Theme.Border)
					if !open.Hovered() && !open.Pressed() {
						open.Background(s.Sidebar)
					}
					if open.Clicked() {
						a.ChooseOpen()
					}
					ui.Text(c, "Ctrl+O").FontSize(12).TextColor(s.Subtle)
				})
			})
			ui.Text(c, "快捷操作").FontSize(14).Margin(12, 0, 4, 0)
			ui.Text(c, "Ctrl+P  切换已打开文档     Ctrl+S  保存\nCtrl+F  查找与替换          Ctrl+G  转到行").FontSize(13).TextColor(s.Subtle)
		})
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
			if a.LineNumbers {
				lineLabel = "隐藏行号"
			}
			if ui.Button(c, lineLabel).Clicked() {
				a.LineNumbers = !a.LineNumbers
				a.applyEditorPreferences()
				a.Persist()
			}
			wrapLabel := "自动换行"
			if a.SoftWrap {
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
		if ui.Button(c, "复制全部").Clicked() {
			a.copyIssues(a.writeIssuesClipboard)
		}
		if ui.Button(c, "收起问题").Clicked() {
			a.ShowIssues = false
		}
	})
}
func (a *App) shortcuts(c *ui.Context) {
	if c.Shortcut(ui.Cmd|ui.Shift, ui.KeyP) || c.Shortcut(0, ui.KeyF1) {
		a.openCommandPalette()
	}
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
	observer0 := t.Editor.OnChange
	hostTerminalSeal, hostSealValid := captureEditorTerminalSeal(t.Editor)
	mountTabID, mountDoc, mountInput := t.ID, t.Doc, t.Editor
	mountDocBuffer, mountInputBuffer, mountAdapter := mountDoc.Buffer, mountInput.Buffer, t.Adapter
	mountManager, mountRoute, mountGeneration, mountHostEpoch := a.pluginManager, t.sourceLanguageRoute, t.OutlineFactoryGeneration, t.formatHostEpoch
	mountPath, mountName, mountEncoding, mountBOM := mountDoc.Path, t.Name, mountDoc.Encoding, mountDoc.BOM
	mountDocReadOnly, mountInputReadOnly, mountInspector := mountDoc.ReadOnly, mountInput.ReadOnly, t.ByteInspector
	mountProfile, mountConfigDir := t.sourceLanguageProfile, a.configDir
	mountSourceSession, mountSourceActivation := a.sourceLanguage.session, a.sourceLanguage.activation
	mountActiveIndex, mountCurrentTab := a.Active, a.Current()
	mountMode, mountInputBinding, mountInputEpoch := t.Mode, t.inputBinding, t.inputEpoch
	mountModeTarget := ""
	mountPhaseBinding := t.inputBinding
	mountStampValid := mountDocBuffer != nil && mountDocBuffer == mountInputBuffer
	var mountStamp text.ChangeStamp
	var mountVersion uint64
	var mountSelection text.Range
	var mountSelectionGeneration uint64
	if mountStampValid {
		mountStamp, mountVersion, mountSelection = mountDocBuffer.ChangeStamp(), mountDocBuffer.Version(), mountDocBuffer.Selection()
		mountSelectionGeneration = mountDocBuffer.SelectionGeneration()
	}
	mountTerminalSeal, mountSealValid := hostTerminalSeal, hostSealValid
	mountPhaseCurrent := func(builderEntered bool) bool {
		if mountModeTarget == "" {
			return (mountMode == "preview" || mountMode == "split") && t.formatHostActive == builderEntered &&
				t.Mode == mountMode && t.inputBinding == mountInputBinding &&
				t.inputEpoch == mountInputEpoch && mountTerminalSeal.current(mountInput)
		}
		return mountMode == "source" && (mountModeTarget == "preview" || mountModeTarget == "split") &&
			t.Mode == mountModeTarget && t.inputBinding != nil && t.inputBinding != mountInputBinding &&
			t.inputEpoch == mountInputEpoch+3 && a.sourceInputConsumerCurrent(t.inputBinding) &&
			t.formatHostActive == builderEntered && mountTerminalSeal.currentHostAttachment(mountInput)
	}
	mountEntryCurrent := func(builderEntered bool) bool {
		return !a.shuttingDown.Load() && !a.restoring && a.tabIsOpen(t) &&
			t.ID == mountTabID && t.Doc == mountDoc && t.Editor == mountInput && t.View == nil &&
			mountDoc.Buffer == mountDocBuffer && mountInput.Buffer == mountInputBuffer && t.Adapter == mountAdapter &&
			a.pluginManager == mountManager && t.sourceLanguageRoute == mountRoute && t.OutlineFactoryGeneration == mountGeneration &&
			t.formatHostEpoch == mountHostEpoch && mountDoc.Path == mountPath && t.Name == mountName &&
			mountDoc.Encoding == mountEncoding && mountDoc.BOM == mountBOM && mountDoc.ReadOnly == mountDocReadOnly &&
			mountInput.ReadOnly == mountInputReadOnly && t.ByteInspector == mountInspector && t.sourceLanguageProfile == mountProfile &&
			a.configDir == mountConfigDir && a.sourceLanguage.session == mountSourceSession &&
			a.sourceLanguage.activation == mountSourceActivation && a.Active == mountActiveIndex && a.Current() == mountCurrentTab &&
			mountAdapter != nil && mountAdapter.Create != nil && mountStampValid && mountSealValid && mountPhaseCurrent(builderEntered) &&
			mountDocBuffer.ChangeStamp() == mountStamp && mountDocBuffer.Version() == mountVersion &&
			mountDocBuffer.Selection() == mountSelection && mountDocBuffer.SelectionGeneration() == mountSelectionGeneration
	}
	// Observe attachment changes only; this does not poll stamps or readiness.
	if t.Mode == "source" {
		t.formatHostActive = false
		a.bindSourceInput(t)
	} else if t.Adapter == nil || t.View != nil {
		a.bindSourceInput(t)
	}
	if t.Adapter == nil {
		if t.MatchedPlugin.ID != "" && (t.MatchedPlugin.Declarative || t.Editor.Highlight != nil) {
			label := t.MatchedPlugin.Name + " · " + t.MatchedPlugin.ID
			ui.Row(c).Height(42).Shrink(0).Padding(5, 12).Gap(8).AlignItems(ui.Center).Clip().Children(func() {
				ui.Text(c, label).FontSize(13).TextColor(c.Theme().TextMuted).
					SingleLine().Tooltip(label).Grow(1)
				ui.Text(c, t.SourceCapabilityBadge).FontSize(12).TextColor(c.Theme().Text).
					SingleLine().Tooltip(t.SourceCapabilityDetail).Padding(2, 8).
					Background(c.Theme().Surface).Border(1, c.Theme().Border).Radius(4)
			})
		}
		a.bindSourceLanguageShortcut(t)
		t.Editor.View(c.Key(t.ID))
		return
	}
	s := workbenchAppearance(a.Dark)
	ui.Row(c.Key("document-toolbar")).Height(42).Shrink(0).Padding(3, 12).Gap(8).AlignItems(ui.Center).Background(s.TabActive).Children(func() {
		name, detail := t.Adapter.Name, t.Adapter.Name
		if t.MatchedPlugin.Declarative {
			name = t.SourceCapabilityBadge
			detail = t.MatchedPlugin.Name + " · " + t.MatchedPlugin.ID + " · " + t.SourceCapabilityDetail
		}
		ui.Text(c, name).FontSize(13).TextColor(c.Theme().TextMuted).Grow(1).MinWidth(0).SingleLine().Tooltip(detail)
		currentModeClicked := false
		ui.Row(c.Key("document-modes")).Height(36).Shrink(0).Padding(2).Gap(2).Radius(8).Background(s.Sidebar).Children(func() {
			for _, mode := range []struct{ id, label string }{{"preview", "阅读 / 编辑"}, {"source", "源码"}, {"split", "分屏"}} {
				b := workbenchButton(c, mode.label).Height(32).Radius(6)
				if mode.id == t.Mode {
					b.Background(c.Theme().Selection).TextColor(c.Theme().Text)
				}
				if b.Clicked() {
					if mode.id == t.Mode {
						currentModeClicked = true
						return
					}
					if a.setMode(t, mode.id) {
						if mountModeTarget == "" {
							mountModeTarget = mode.id
								mountPhaseBinding = t.inputBinding
						}
					}
				}
			}
		})
		if currentModeClicked {
			return
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
			button := workbenchButton(c, label).Label(access).Height(32).Shrink(0).Disabled(disabled)
			if tooltip != "" {
				button.Tooltip(tooltip)
			}
			if button.Clicked() {
				a.exportCurrent()
			}
		}
	})
	if t.Mode == "source" {
		a.bindSourceLanguageShortcut(t)
		t.Editor.View(c.Key(t.ID))
		return
	}
	if t.View == nil {
		if mountDocBuffer == nil || mountInputBuffer == nil || mountDocBuffer != mountInputBuffer {
			return
		}
		if !mountEntryCurrent(false) {
			return
		}
		factoryTerminalSeal, factorySealValid := captureEditorTerminalSeal(mountInput)
		if !factorySealValid || !factoryTerminalSeal.current(mountInput) {
			return
		}
		host, attachCreatedView := a.editorHostForMount(t, observer0, hostTerminalSeal, hostSealValid)
		createdView := mountAdapter.Create(host)
		if createdView == nil || !mountEntryCurrent(true) || !factoryTerminalSeal.current(mountInput) {
			return
		}
		t.View = createdView
		if !attachCreatedView(createdView) {
			return
		}
		a.bindSourceInput(t)
		bindingAfter := t.inputBinding
		expectedMountMode := mountMode
		expectedBindEpoch := mountInputEpoch + 1
		if mountModeTarget != "" {
			expectedMountMode = mountModeTarget
			expectedBindEpoch = mountInputEpoch + 4
		}
		bindTerminalSeal, bindingSealed := factoryTerminalSeal.replaceRegistration(mountInput)
		mountEntryAfterBindCurrent := func() bool {
			return !a.shuttingDown.Load() && !a.restoring && a.tabIsOpen(t) &&
				t.ID == mountTabID && t.Doc == mountDoc && t.Editor == mountInput && t.View == createdView &&
				mountDoc.Buffer == mountDocBuffer && mountInput.Buffer == mountInputBuffer && t.Adapter == mountAdapter &&
				a.pluginManager == mountManager && t.sourceLanguageRoute == mountRoute && t.OutlineFactoryGeneration == mountGeneration &&
				t.formatHostEpoch == mountHostEpoch && t.formatHostActive && mountDoc.Path == mountPath && t.Name == mountName &&
				mountDoc.Encoding == mountEncoding && mountDoc.BOM == mountBOM && mountDoc.ReadOnly == mountDocReadOnly &&
				mountInput.ReadOnly == mountInputReadOnly && t.ByteInspector == mountInspector && t.sourceLanguageProfile == mountProfile &&
				a.configDir == mountConfigDir && a.sourceLanguage.session == mountSourceSession &&
				a.sourceLanguage.activation == mountSourceActivation && a.Active == mountActiveIndex && a.Current() == mountCurrentTab &&
				t.Mode == expectedMountMode && t.inputBinding == bindingAfter && t.inputEpoch == expectedBindEpoch &&
				mountStampValid && mountDocBuffer.ChangeStamp() == mountStamp &&
				mountDocBuffer.Version() == mountVersion && mountDocBuffer.Selection() == mountSelection &&
				mountDocBuffer.SelectionGeneration() == mountSelectionGeneration
		}
		if !bindingSealed || bindingAfter == nil || bindingAfter == mountPhaseBinding ||
			!a.sourceInputConsumerCurrent(bindingAfter) || !bindTerminalSeal.current(mountInput) ||
			bindingAfter.stamp != mountDocBuffer.ChangeStamp() ||
			!mountEntryAfterBindCurrent() {
			return
		}
	}
	warnings := t.View.Warnings()
	t.ViewWarnings = append(t.ViewWarnings[:0], warnings...)
	inlineWarnings := false
	if reporter, ok := t.View.(interface{ WarningsInline() bool }); ok {
		inlineWarnings = reporter.WarningsInline()
	}
	if len(warnings) > 0 && !inlineWarnings {
		_, windowHeight := c.Size()
		compact := windowHeight < 600
		row := ui.Row(c.Key("format-warnings")).Padding(7, 18).Gap(8)
		if compact {
			// Leave a predictable height for the editor even if a warning arrives mid-draft.
			row.Height(28).Shrink(0).Padding(2, 18).AlignItems(ui.Center)
		}
		row.Children(func() {
			message := warnings[0]
			detail := strings.Join(warnings, "\n")
			if len(warnings) > 1 {
				message += fmt.Sprintf("（另有 %d 项）", len(warnings)-1)
			}
			label := ui.Text(c, message).FontSize(11).TextColor(c.Theme().Warning).Grow(1).
				Description(detail).Tooltip(detail)
			if compact {
				label.MinWidth(0).SingleLine()
			}
			if ui.Button(c, "查看详情").Height(24).Shrink(0).Description(detail).Clicked() {
				a.ShowIssues = true
			}
		})
	}
	// The retained format view may reactivate after Release. This authorizes
	// its same factory Host only; route/dispose epochs never reactivate.
	t.formatHostActive = true
	if t.Mode == "split" {
		ui.Row(c.Key("document-split")).Grow(1).AlignItems(ui.Stretch).Children(func() {
			ui.Column(c).WidthPercent(50).Children(func() {
				a.bindSourceLanguageShortcut(t)
				t.Editor.View(c.Key(t.ID))
			})
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
	if !a.releaseTabPresentation(t, t.isBinary() && t == a.Current()) {
		return false
	}
	t.Mode = mode
	a.bindSourceInput(t)
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
