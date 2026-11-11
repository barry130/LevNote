package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/enginecatalog"
	"levnote.local/native/core/engineruntime"
)

func (a *App) enginePackagePanel(c *ui.Context) {
	ui.Column(c).Grow(1).MinHeight(0).Padding(8, 18).Gap(6).Background(c.Theme().Surface).Children(func() {
		ui.Row(c).Gap(8).Children(func() {
			ui.Text(c, "本地引擎包").Bold().FontSize(15).Grow(1)
			if ui.Button(c, "关闭插件").Height(28).Clicked() {
				a.ShowPlugins = false
			}
		})
		ui.Scroll(c.Key("package-scroll")).Grow(1).MinHeight(0).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Gap(7).Children(func() {
				ui.Text(c, "导入会复制本地可执行程序；启用后，应用可运行它完成对应的预览或导出。请只导入你信任的发布文件夹。").FillWidth().FontSize(11).TextColor(c.Theme().TextMuted)
				ui.Text(c, "格式配置是另一类不执行代码的 JSON；导入引擎不会自动启用。").FillWidth().FontSize(11).TextColor(c.Theme().TextMuted)
				ui.Row(c).Wrap().Gap(6).Children(func() {
					for _, item := range []struct {
						engine enginecatalog.EngineID
						face   string
					}{
						{enginecatalog.MarkdownHTML, "Markdown"}, {enginecatalog.MindmapSVG, "导图 SVG"},
						{enginecatalog.MindmapPreview, "只读预览"}, {enginecatalog.DrawioSVG, "图纸 SVG"},
					} {
						if ui.Button(c.Key(string(item.engine)), item.face).Label("选择" + packageEngineLabel(item.engine)).Height(28).Clicked() {
							a.packageEngine = item.engine
						}
					}
				})
				ui.Text(c, packageEngineLabel(a.packageEngine)).Bold().FontSize(13).FillWidth()
				if a.packageInitErr != nil || a.packageManager == nil {
					message := "引擎包仓库不可用"
					if a.packageInitErr != nil {
						message += "：" + a.packageInitErr.Error()
					}
					ui.Text(c, message).FillWidth().FontSize(12).TextColor(c.Theme().Warning)
					return
				}
				blocked := a.packageManager.LoadError()
				for _, status := range a.packageStatuses {
					if status.Engine != a.packageEngine {
						continue
					}
					message := "当前选择：随包版本（运行时验证）"
					if blocked != nil {
						message = "当前选择无法读取；引擎请求已阻断"
					} else if status.Source == engineruntime.PackageInstalled {
						message = "当前选择：已安装 " + status.Version + " · " + shortPackageDigest(status.Digest) + "（运行时验证）"
					}
					ui.Text(c, message).FillWidth().FontSize(12).Tooltip(string(status.Engine) + "\n" + status.Digest)
				}
				busy := a.packageJob != nil
				if blocked != nil {
					ui.Text(c, "选择状态损坏，所有引擎请求已阻断："+blocked.Error()).FillWidth().FontSize(11).TextColor(c.Theme().Warning)
					if ui.Button(c, "重置全部引擎").Label("将所有引擎恢复为随包选择并保留安装包").Height(28).Disabled(busy || a.enginePoison != nil).Clicked() {
						a.startPackageWork(packageRepairAction, a.packageEngine, "")
					}
				}
				if a.enginePoison != nil {
					ui.Text(c, "引擎清理失败，新任务已停止："+a.enginePoison.Error()).FillWidth().FontSize(11).TextColor(c.Theme().Warning)
				}
				ui.Row(c).Wrap().Gap(6).Children(func() {
					if ui.Button(c, "导入文件夹").Label("从可信本地发布文件夹导入" + packageEngineLabel(a.packageEngine)).Height(28).Disabled(busy).Clicked() {
						a.chooseEngineImport()
					}
					if ui.Button(c, "检查列表").Label("检查已安装的" + packageEngineLabel(a.packageEngine)).Height(28).Disabled(busy).Clicked() {
						a.startPackageWork(packageListAction, a.packageEngine, "")
					}
					if ui.Button(c, "恢复随包").Label("恢复" + packageEngineLabel(a.packageEngine) + "的随包版本").Height(28).Disabled(busy || blocked != nil || a.enginePoison != nil).Clicked() {
						a.startPackageWork(packageResetAction, a.packageEngine, "")
					}
				})
				if a.packageOutcome != "" {
					ui.Text(c, a.packageOutcome).FillWidth().FontSize(11).TextColor(c.Theme().TextMuted)
				}
				if busy {
					ui.Text(c, packageEngineLabel(a.packageJob.engine)+" · "+packageActionLabel(a.packageJob.action)).FillWidth().FontSize(11)
					ui.Text(c, "操作正在后台完成；关闭面板不会停止它。").FillWidth().FontSize(11)
					if ui.Button(c, "取消操作").Label("取消当前引擎包操作并等待清理").Height(28).Clicked() {
						a.packageJob.cancel()
						a.packageOutcome = "正在取消；等待文件与进程清理"
					}
				}
				list := a.packageLists[a.packageEngine]
				if !list.checked {
					ui.Text(c, "尚未完整检查已安装列表；导入成功的包会先显示在下方。").FillWidth().FontSize(11).TextColor(c.Theme().TextMuted)
				} else if len(list.receipts) == 0 {
					ui.Text(c, "没有通过本次检查的已安装包。").FillWidth().FontSize(12)
				}
				if list.err != nil {
					ui.Text(c, "列表问题："+list.err.Error()).FillWidth().FontSize(11).TextColor(c.Theme().Warning)
				}
				for _, receipt := range list.receipts {
					receipt := receipt
					ui.Column(c.Key(receipt.Digest)).FillWidth().Padding(6).Gap(3).Border(1, c.Theme().Border).Children(func() {
						ui.Text(c, receipt.Version).Bold().FontSize(12).FillWidth()
						ui.Text(c, fmt.Sprintf("包 %s · %d 个文件；运行时重新验证", shortPackageDigest(receipt.Digest), len(receipt.Files))).FontSize(11).FillWidth().Tooltip(receipt.Digest)
						if ui.Button(c, "启用此包").Label("启用" + packageEngineLabel(receipt.Engine) + " " + receipt.Version + " " + shortPackageDigest(receipt.Digest)).Height(28).Disabled(busy || blocked != nil || a.enginePoison != nil).Clicked() {
							a.startPackageWork(packageActivateAction, receipt.Engine, receipt.Digest)
						}
					})
				}
			})
		})
	})
}

// Keep the captured operation reachable when the user returns to a document
// or the format configuration tab. This reads cached state only.
func (a *App) packageProgress(c *ui.Context) {
	j := a.packageJob
	if j == nil || a.ShowPlugins && a.showEnginePackages {
		return
	}
	ui.Row(c.Key("package-progress")).Height(36).Shrink(0).Padding(4, 12).Gap(6).Children(func() {
		ui.Text(c, packageEngineLabel(j.engine)+" · "+packageActionLabel(j.action)+"…").FontSize(11).Grow(1).Shrink(1).SingleLine().Tooltip(a.packageOutcome)
		if ui.Button(c, "查看操作").Label("查看当前引擎包操作").Height(28).Clicked() {
			a.ShowPlugins, a.showEnginePackages = true, true
			a.packageEngine = j.engine
		}
		if ui.Button(c, "取消").Label("取消当前引擎包操作并等待清理").Height(28).Clicked() {
			j.cancel()
			a.packageOutcome = "正在取消；等待文件与进程清理"
		}
	})
}
