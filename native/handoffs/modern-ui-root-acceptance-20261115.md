# 现代工作台 UI 与 Windows 体验包验收

日期：2026-11-15。用户要求“没有现代化一点的UI吗”。本轮完成更舒展的原生工作台，交付 native-ui.20261115.2-preview。正式接受包 native-worker.20261110.29 继续保留；本次为 UI 体验构建。

## 结果

深浅主题采用分层背景、蓝色主操作、统一线性图标和灰阶状态栏。顶部56px、活动栏56px/按钮40px、标签带44px、文档栏42px；圆角标签和分段模式切换保留实际编辑行为。欢迎页使用30px标题和44px主次入口；顶部搜索实际打开已打开文档选择器，保留焦点及组合输入保护。760×480时主区至少508px；极窄内容继续允许响应式重排。

更改主要在[主题](../app/workbench_theme.go)、[外框](../app/workbench_chrome.go)、[图标](../app/workbench_icons.go)、[视图](../app/view.go)；另将[图形提示](../modules/graphview/view_ui.go#L240)两处文字从Accent改用Focus，对比度深色7.402:1、浅色6.922:1。独立审阅确认搜索焦点、列表稳定标识、标签延后关闭/揭示和模式/导出守卫，无剩余实质发现。

## 验证与修正

| 检查 | 结果 | 证据 |
|---|---|---|
| 最终聚焦 | 186 RUN / 182 PASS / 4 SKIP / 0 FAIL | [focused-03](../../.localtools/modern-ui-20261115-focused-03/result.json) |
| 六产品根完整回归 | 3264 RUN / 3215 PASS / 49 SKIP / 0 FAIL，exit0 | [test-02](../../.localtools/modern-ui-20261115-test-02/result.json) |
| 静态检查 | vet exit0 | [vet-01](../../.localtools/modern-ui-20261115-vet-01/result.json) |
| 测试身份与输入核对 | 保留前版3258身份，新增6；49原SKIP无新增；三阶段410输入一致 | [最终核对](../../.localtools/modern-ui-final-audit-20261115.json) |
| Windows构建 | 宿主+四worker五次构建全部exit0，1259输入前后一致 | [构建结果](../../.localtools/modern-ui-preview-build-20261115-01/result.json) |
| ZIP独立复核 | 26文件、25清单成员、精确8引擎路径、五amd64 GUI PE、CRC及全部哈希通过 | [独立验包](../../.localtools/modern-ui-preview-final-verify-20261115.json) |

首次[完整回归test-01](../../.localtools/modern-ui-20261115-test-01/result.json)保留3207 PASS/49 SKIP/8 FAIL原始记录。修正包括：标签测试从旧46–86边界改为56–100并保留全部行为断言；主区MinWidth回到0，修复360px测试中只读表格的响应式排版；插件最小窗口断言改为实际滚动到每个控件/扩展名，要求正尺寸且位于顶部与状态栏之间。新测试保留搜索的真实鼠标/过滤/Enter/Escape及组合输入不改文档检查。

最早截图还发现窄窗新建Markdown截字，已通过13px文字与较小内边距修复。最终7场景及4状态交互共11张MyGo Tester实渲图，覆盖深浅主题、源码/阅读/欢迎页和760×480窄窗；主要查看：[深色阅读](../../.localtools/modern-ui-20261115-focused-03/images/dark-reading.png)、[浅色源码](../../.localtools/modern-ui-20261115-focused-03/images/light-source.png)、[窄窗欢迎](../../.localtools/modern-ui-20261115-focused-03/images/light-welcome-narrow.png)。

## 交付

[Windows x64体验ZIP](../downloads/LevNote-Modern-UI-preview-20261115.2-windows-x64.zip)，11,643,907字节。解压完整文件夹后双击 levnote-native.exe；默认深色，已有浅色偏好保留，在设置中切换主题。

ZIP SHA256：fb0171d3947ad320b01c5da05deb2e18c1cb0f99999e4e4447a8a5f9d37b2d0f。
宿主 SHA256：6f75fb80205bcbff93992b0650891c9554e31fd3ae9daacea418d9e4bdfe68b5。

截图为原生测试器渲染；新包未执行实体桌面输入、剪贴板、原生对话框及正常关闭/重启的完整发行验收。旧包、既有失败和[VS Code版验收](vscode-ui-root-acceptance-20261115.md)保留。改动留在工作区，未提交或推送。
