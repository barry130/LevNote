# VS Code 工作台主题结果 · 2026-11-15

STOPPED_WRITING。主题源码已完成；产品测试、格式化与截图由 root 统一集成执行。

## 改动

- 新增 [workbench_theme.go](<F:/AIproject/LevNote/native/app/workbench_theme.go>)，提供约定的 `workbenchStyle` 与 `workbenchAppearance(dark bool)`。每次调用返回独立的 `*ui.Theme`，布局可复制并局部覆盖状态栏主题。
- [app.go:251](<F:/AIproject/LevNote/native/app/app.go#L251>) 仅在 `New` 默认字段增加 `Dark: true`。原有未提交重载修复保持原样。
- [state.go:63](<F:/AIproject/LevNote/native/app/state.go#L63>) 仅改 `Restore` 主题分支：显式 `light` 为亮色，其余为深色。`session.Load` 对缺文件返回空 Theme，对有效快照中的空/未知 Theme 归一为 `system`；因此新配置、未选择主题与旧 system 缺省均得到深色。保存流程本身保持原样，继续写出明确的 light/dark。
- 现有语法配色实际位于 [plugins.go:17](<F:/AIproject/LevNote/native/app/plugins.go#L17>) 的 `syntaxPalette` / `syntaxColors`，本轮保持原样。

## 视觉与对比度

已加载 `frontend-design`，按 [设计稿](<F:/AIproject/LevNote/native/handoffs/vscode-ui-design-20261115.md>) 实现紧凑 VS Code 工作台，无布局文件改动。

- 深色编辑区 #1f1f1f，Chrome/Sidebar/Activity/TabBar #181818，激活页签同编辑区；控件 #252526、悬浮 #2a2d2e、边界 #2b2b2b、正文 #cccccc。
- 亮色编辑区白色，周围工作台及侧栏 #f3f3f3；正文 #3b3b3b。
- 强调 #0078d4，状态 #007acc，强调与状态文字均为白色。焦点独立采用深色 #75beff / 亮色 #005a9e，以保证控件底色上的可见性。
- UI Microsoft YaHei UI 13，Radius 3，Spacing 3，FocusRingWidth 1。编辑字体与字号继续走现有偏好。
- 数值预核：白/状态蓝 4.510561、白/强调蓝 4.528515；正文、弱化字、焦点与语义色在实际普通表面上的最小比值分别为亮色 4.730040、深色 4.885002。这是配色算术，产品测试仍待 root 执行。

## 测试源码与集成

新增 [workbench_theme_test.go](<F:/AIproject/LevNote/native/app/workbench_theme_test.go>) 三项顶层测试：

1. `TestWorkbenchThemeRestoreAndPersist`：八个真实磁盘输入（新配置、缺 preferences、缺 theme、空 theme、system、未知、light、dark），调用 New → Restore → Persist → Shutdown → New → Restore；同时检查恢复期间原偏好文件未被改写，并保留已有编辑字号。
2. `TestWorkbenchThemeSelectionPersists`：从默认深色明确切换亮色、再切回深色，两次真实保存/重启后保持用户选择。
3. `TestWorkbenchThemeReadableColors`：检查实际主题文字、弱化字、状态、强调、焦点、语义色、选区、tooltip 与既有语法色的 WCAG 对比度 >=4.5；不做镜像色值断言。

状态栏继承已直接与 vscode-shell 协调：MyGo `Button` 在 [widgets.go:155](<F:/AIproject/LevNote/native/vendor/github.com/egoist/mygo/ui/widgets.go#L155>) 显式取 `Theme.Text`，父容器白字不覆盖按钮。布局需逐按钮设 StatusText 或使用状态栏局部主题；#cccccc 在 #007acc 上仅 2.808693。强调蓝作暗底小号文字也不足4.5，已提示 shell 使用 Theme.Focus/Text 表达可读文字。

本代理未调用任何产品 test/vet/build/run 或截图，未调用 gofmt；遵从 root 当前统一格式化/门禁安排。已执行的源码检查只有 `git diff --check -- native/app/app.go native/app/state.go`（exit0，stdout/stderr 均空）及只读差异/身份检查。

root 下一步：合并 shell 对 `workbenchAppearance` 的使用后，统一格式化新增源码，通过既有 gate 执行 `^TestWorkbenchTheme` 与相关应用回归，再做最终完整门禁及真实 View 的截图检查。记录每次实际命令、原始输出及退出码。

## 交付时源码 SHA256（格式化前）

| 文件 | SHA256 |
| --- | --- |
| [主题](<F:/AIproject/LevNote/native/app/workbench_theme.go>) | a7169f7bbf01aa809e7e40993f7cf266a46e1b422864bcea704d160d2da5015f |
| [测试](<F:/AIproject/LevNote/native/app/workbench_theme_test.go>) | 5a9d0e665d2be75e611b40021f303911ab75174108dfade4d2255bc397dcad9b |
| [构造函数所在文件](<F:/AIproject/LevNote/native/app/app.go>) | 0628fbec0b753403cd3e753116d02cbcec801d87cf83cf62ff3eef7ff8e5f557 |
| [恢复分支所在文件](<F:/AIproject/LevNote/native/app/state.go>) | b0179e65905eed857a0a58de7f83788cb043d7574360a1efa88d12d5348f6a46 |
| [原有语法配色文件（只读）](<F:/AIproject/LevNote/native/app/plugins.go>) | fa881deb3ae0435b85b9eb2df6b27ead1ce2e1deeaed9647691c990996944f23 |

写入范围为以上四个源码/测试文件及本交接；无子委派、无 commit/push。共享记忆与总验收记录留给 root 合并更新。
