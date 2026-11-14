package app

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/search"
)

func (a *App) startWatch(path string, recursive bool) {
	if path == "" || a.Window == nil {
		return
	}
	if _, exists := a.watches[path]; exists {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.watches[path] = cancel
	go func() {
		watch, err := mygo.WatchFiles(ctx, path, mygo.FileWatchOptions{Recursive: recursive, Debounce: 250 * time.Millisecond, Ignore: func(path string, dir bool) bool {
			name := filepath.Base(path)
			return dir && (name == ".git" || name == "node_modules" || name == "target" || name == "tools" || name == "build-cache")
		}})
		if err != nil {
			a.update(func() {
				a.report(fmt.Errorf("文件监视未启动，外部改动可能发现不了：%s：%w", path, err))
				delete(a.watches, path)
			})
			return
		}
		defer watch.Close()
		for {
			_, err = watch.Next(ctx)
			if err != nil {
				if ctx.Err() == nil {
					a.update(func() { a.report(fmt.Errorf("文件监视已中断：%s：%w", path, err)); delete(a.watches, path) })
				}
				return
			}
			a.update(func() {
				a.CheckExternal()
				if recursive {
					a.refreshTree()
				}
			})
		}
	}()
}
func (a *App) CheckExternal() {
	a.scheduleDiskCheck()
}
func (a *App) Shutdown() {
	if !a.shuttingDown.CompareAndSwap(false, true) {
		<-a.shutdownJoined
		return
	}
	defer close(a.shutdownJoined)
	a.closeOpenDocuments()
	a.clearClosedFiles()
	a.cancelAutosave()
	a.shutdownBinary()
	a.invalidateFindCache()
	if err := a.closeEngineExport(); err != nil {
		a.report(fmt.Errorf("导出进程清理失败：%w", err))
	}
	a.releaseOutline()
	a.closeAppLists()
	close(a.shutdownDone)
	if a.searchCancel != nil {
		a.searchCancel()
	}
	for _, cancel := range a.watches {
		cancel()
	}
	a.treeWorker.close()
	a.diskWorker.close()
}
func (a *App) searchWorkspace() {
	if a.searchCancel != nil {
		a.searchCancel()
	}
	if a.Root == "" || a.Query == "" {
		a.Message = "先打开文件夹并输入查找内容"
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.searchCancel = cancel
	a.searchID++
	id := a.searchID
	a.Searching = true
	a.SearchResults = nil
	a.SearchIssues = nil
	a.SearchIssuesDropped = 0
	a.SearchSelected = -1
	a.searchListLifetime.reset(&a.SearchList)
	a.SearchSummary = ""
	a.ShowWorkspaceSearch = true
	root, opts := a.Root, a.findOptions()
	a.searchRoot = root
	go func() {
		summary, err := search.Workspace(ctx, root, search.WorkspaceOptions{Options: opts}, func(batch search.Batch) {
			a.update(func() {
				if id != a.searchID {
					return
				}
				a.SearchResults = append(a.SearchResults, batch.Matches...)
				a.addSearchIssues(batch.Issues)
			})
		})
		a.update(func() {
			if id != a.searchID {
				return
			}
			a.Searching = false
			if err != nil && ctx.Err() == nil {
				a.report(err)
			}
			a.SearchSummary = fmt.Sprintf("%d 个匹配 · %d 个文件 · 跳过 %d 项", summary.Matches, summary.Files, summary.Skipped)
			if summary.Truncated {
				a.SearchSummary += " · 结果已截断，请缩小范围"
			}
			if summary.Cancelled {
				a.SearchSummary += " · 已取消"
			}
		})
	}()
}

func (a *App) addSearchIssues(issues []search.FileIssue) {
	if len(issues) == 0 {
		return
	}
	room := max(0, searchIssueLimit-len(a.SearchIssues))
	keep := min(room, len(issues))
	a.SearchIssues = append(a.SearchIssues, issues[:keep]...)
	a.SearchIssuesDropped += len(issues) - keep
}

func (a *App) workspaceResults(c *ui.Context) {
	panelHeight := float32(230)
	if _, height := c.Size(); height > 0 && height < 600 {
		// Leave source input usable beside Find at the 760x480 minimum.
		panelHeight = min(panelHeight, max(float32(110), height-260))
	}
	ui.Column(c).Height(panelHeight).Shrink(1).MinHeight(110).Border(1, c.Theme().Border).Children(func() {
		ui.Row(c).Padding(8, 12).Gap(8).Children(func() {
			label := a.SearchSummary
			if a.Searching {
				label = "正在搜索工作区…"
			}
			ui.Text(c, label).FontSize(12).SingleLine().Grow(1).Shrink(1).MinWidth(0).Tooltip(label)
			if a.Searching && ui.Button(c, "取消搜索").Shrink(0).Clicked() && a.searchCancel != nil {
				a.searchCancel()
			}
			if len(a.SearchIssues) > 0 && ui.Button(c, "查看跳过详情").Shrink(0).Clicked() {
				a.ShowIssues = true
			}
			if ui.Button(c, "收起结果").Shrink(0).Clicked() {
				a.ShowWorkspaceSearch = false
				a.SearchList.Handle.CancelFocus()
			}
		})
		if !a.ShowWorkspaceSearch {
			return // Collapse preserves the current results and scroll place.
		}
		if a.searchRoot != a.Root {
			a.SearchList.Handle.CancelFocus()
			ui.Text(c, "工作区已变化，请重新搜索。").FontSize(12).TextColor(c.Theme().TextMuted).Padding(10, 14)
			return
		}
		results := a.SearchResults
		if len(results) == 0 {
			a.SearchList.Handle.CancelFocus()
			label := "没有匹配结果。"
			if a.Searching {
				label = "正在读取结果…"
			}
			ui.Text(c, label).FontSize(12).TextColor(c.Theme().TextMuted).Padding(10, 14)
		} else {
			id, root := a.searchID, a.searchRoot
			current := func() bool {
				return !a.shuttingDown.Load() && a.ShowWorkspaceSearch && !a.ShowPlugins &&
					a.searchID == id && a.searchRoot == root && a.Root == root
			}
			selected := a.SearchSelected
			a.SearchList.Selected = &selected
			// Captured keys stay distinct until this frame finishes, even
			// when a newer search makes its labels and actions obsolete.
			a.SearchList.Key = func(i int) any {
				if i < 0 || i >= len(results) {
					return i
				}
				result := results[i]
				return searchRowKey{Path: result.Path, Start: result.Start, End: result.End}
			}
			a.SearchList.Label = func(i int) string {
				if !current() || i < 0 || i >= len(results) {
					return ""
				}
				result := results[i]
				return fmt.Sprintf("%s:%d  %s", filepath.Base(result.Path), result.Line, result.Preview)
			}
			a.searchListLifetime.built = true
			list := ui.List(c, &a.SearchList, len(results), func(i int) {
				if !current() || i < 0 || i >= len(results) {
					return
				}
				if a.searchRowBuilt != nil {
					a.searchRowBuilt(i)
				}
				if !current() {
					return
				}
				result := results[i]
				label := fmt.Sprintf("%s:%d  %s", filepath.Base(result.Path), result.Line, result.Preview)
				ui.Row(c).FillWidth().Height(28).Padding(4, 8).Children(func() {
					ui.Text(c, label).SingleLine().Grow(1).MinWidth(0).Tooltip(result.Path + "\n" + label)
				})
			}).Grow(1)
			if current() {
				a.SearchSelected = selected
				if list.Submitted() && selected >= 0 && selected < len(results) && selected < len(a.SearchResults) {
					old, now := results[selected], a.SearchResults[selected]
					if old.Path == now.Path && old.Start == now.Start && old.End == now.End {
						a.activateSearchResult(selected)
					}
				}
			}
		}
	})
}

func (a *App) activateSearchResult(i int) {
	if i < 0 || i >= len(a.SearchResults) {
		return
	}
	a.SearchSelected = i
	result := a.SearchResults[i]
	path := canonical(result.Path)
	for index, t := range a.Tabs {
		if t == nil || t.Doc == nil || t.Doc.Path == "" || !sameNativeFilePath(canonical(t.Doc.Path), path) {
			continue
		}
		if !t.textCapable() {
			a.Message = "工作区搜索目标已变化；未跳转"
			return
		}
		doc, input, buffer := t.Doc, t.Editor, t.Doc.Buffer
		capturedPath, version := doc.Path, buffer.Version()
		id, root, origin := a.searchID, a.searchRoot, a.Current()
		check := func() error {
			if a.shuttingDown.Load() || !a.ShowWorkspaceSearch || a.ShowPlugins || a.searchID != id ||
				a.searchRoot != root || a.Root != root || i >= len(a.SearchResults) ||
				a.SearchResults[i].Path != result.Path || a.SearchResults[i].Start != result.Start || a.SearchResults[i].End != result.End {
				return fmt.Errorf("工作区搜索结果已变化；未跳转")
			}
			if a.Current() != origin || index >= len(a.Tabs) || a.Tabs[index] != t ||
				t.Doc != doc || t.Editor != input || doc.Buffer != buffer || input.Buffer != buffer || doc.Path != capturedPath {
				return fmt.Errorf("工作区搜索目标已变化；未跳转")
			}
			// Disk-result coordinates do not describe an unsaved live draft.
			// Check before flushing too: preserve already-dirty presentations.
			if doc.Dirty() {
				return fmt.Errorf("工作区结果来自磁盘；文档已有未保存修改，未跳转")
			}
			if buffer.Version() != version {
				return fmt.Errorf("同步输入期间文档已变化；工作区结果未跳转")
			}
			return nil
		}
		if err := check(); err != nil {
			a.report(err)
			return
		}
		if !a.flushTab(t, "工作区搜索定位") {
			return
		}
		if err := check(); err != nil {
			a.report(err)
			return
		}
		// Switching may flush another tab. Recheck before Select releases it,
		// and reveal synchronously without OpenAt's second target flush.
		if !a.selectChecked(index, check) {
			return
		}
		origin = t
		if err := check(); err != nil {
			a.report(err)
			return
		}
		a.revealSource(t, revealTarget{Start: result.Start, End: result.End})
		return
	}
	// Unopened files retain the existing open/load path.
	a.OpenAt(result.Path, result.Start, result.End)
}
func (a *App) conflictBar(c *ui.Context, t *Tab) {
	ui.Row(c).Padding(8, 14).Gap(8).Background(c.Theme().Warning.Alpha(.12)).Children(func() {
		ui.Text(c, "磁盘上的文件已改变。").Grow(1).FontSize(12)
		if ui.Button(c, "重新载入").Clicked() {
			if t.isBinary() {
				a.reloadBinary(t)
				return
			}
			defer a.deferExportDelivery()()
			if !a.flushTab(t, "重新载入") {
				return
			}
			if t.Doc.Dirty() {
				r, err := mygo.Dialog.Message(mygo.MessageOptions{Parent: a.Window, Title: "重新载入", Message: "放弃当前修改并读取磁盘版本？", Buttons: []string{"取消", "放弃并重新载入"}, CancelButton: 0, DestructiveButtons: []int{1}})
				if err != nil {
					a.report(err)
					return
				}
				if r.Button != 1 {
					return
				}
			}
			if err := t.Doc.Revert(); err != nil {
				a.report(err)
				return
			}
			a.invalidateTabExport(t, "文档已重新载入，已取消导出")
			a.invalidateDiskChecks()
			a.disposePresentation(t)
			t.Editor.Buffer = t.Doc.Buffer
			t.Editor.ReadOnly = t.Doc.ReadOnly
			t.Conflict = false
			t.Mode = "source"
			a.configureTab(t)
			a.Message = "已重新载入 " + t.Name
		}
		if ui.Button(c, "另存副本").Clicked() {
			a.Save(t, true)
		}
	})
}
