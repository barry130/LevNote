# VS Code 工作台 Spec 独审

基线 f2c2ceb82acecb40fb0502c047e41ba12a12be47；已审指定三文件增量、六个新增 workbench 文件及最终配套修正。[app.go](<F:/AIproject/LevNote/native/app/app.go#L251>)仅审默认深色，既有重载修复另属原任务。

**最终静态结论：无剩余实质 Spec 问题。**

- 对照[设计第9、10、13行](<F:/AIproject/LevNote/native/handoffs/vscode-ui-design-20261115.md#L9>)，插件回返时的[查找](<F:/AIproject/LevNote/native/app/workbench_chrome.go#L95>)／[设置](<F:/AIproject/LevNote/native/app/workbench_chrome.go#L112>)反向关闭及[状态按钮](<F:/AIproject/LevNote/native/app/workbench_chrome.go#L34>)浅色交互对比度问题均已修正；查找保留组合输入检查。
- [状态消息](<F:/AIproject/LevNote/native/app/workbench_chrome.go#L147>)改用稳定Key，保留原文可访问名和完整Tooltip；[目录名称](<F:/AIproject/LevNote/native/app/view.go#L208>)恢复展开前缀。[菜单](<F:/AIproject/LevNote/native/app/reopen_closed_test.go#L795>)与[标签](<F:/AIproject/LevNote/native/app/tab_workflow_test.go#L407>)测试只适配新几何，非零面积、窗口界限、循环切换、手动滚动及reveal行为断言保留。
- [设计第8、13、14行](<F:/AIproject/LevNote/native/handoffs/vscode-ui-design-20261115.md#L8>)对应的[窄窗欢迎入口](<F:/AIproject/LevNote/native/app/view.go#L315>)、[saved light优先](<F:/AIproject/LevNote/native/app/state.go#L65>)及[标签t.ID、延后关闭、按ID显露](<F:/AIproject/LevNote/native/app/view.go#L272>)均保留；未见编辑、只读或字节能力回退。最近关闭入口只换样式，浅色验收截图显式设light。

已读新增测试并查看五张实际Tester截图。独立读取[test02原流](<F:/AIproject/LevNote/.localtools/vscode-ui-20261115-test-02/stdout.raw>)确认旧几何、目录及消息回归通过；唯一新树测试的名称匹配已[修正](<F:/AIproject/LevNote/native/app/workbench_layout_test.go#L124>)。本审阅未执行Go或启动程序；全量test03结果由lead补入最终验收。

STOPPED_WRITING
