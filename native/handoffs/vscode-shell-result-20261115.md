# VS Code 工作台外壳结果 · 2026-11-15

## 交付
- 仅修改 app/view.go，新增 app/workbench_chrome.go、app/workbench_icons.go、app/workbench_layout_test.go；主题消费 workbenchAppearance，不复制颜色字面量。
- 顶部 44px 平直命令栏、48px 活动栏、窄窗 204px 侧栏、34px 标签页及蓝色活动顶线、32px 文档工具条、29px 蓝色状态栏、左对齐欢迎页。
- 活动栏保留文件树、大纲、查找（含现有工作区搜索入口）、插件、设置；SVG 本地内嵌、无依赖；目录按深度缩进并有文件类型图标，新建类型收为两行。
- 窄窗长状态消息保留编码/只读信息并提供完整 tooltip；状态文字及按钮显式使用 StatusText。29px 状态栏与既有回归几何一致；44px 顶栏满足现有 tab Y 范围。

## 兼容与范围
- 保留列表 captured entries/root/generation/current、TreeList.Key/Label、owner lifetime、选中提交逻辑；保留标签 ID、延迟关闭和一次性 ScrollIntoView。
- 活动查找和状态行跳转保留 KeepFocus；既有查找/设置/插件/标签操作文本或 Label 保留。
- documentView 中原编辑器/格式视图调用、setMode、shortcuts、未知编码选择及其他数据路径保持；app.go/state.go/theme.go/editor/vendor 均未写入，现有围栏/重载修复未触碰。
- 设置入口从插件页切出后显示设置，避免点击后面板仍被插件页遮住。

## 验证
- 新增四个 TestWorkbench*：活动入口实际导航；暗/亮 760×480 正文与长错误下编码/只读状态；目录文本缩进/选中跨 resize；欢迎页新建文档和模式入口。
- 只读 gofmt -d 已检查；首次 exit1 为两个文件的纯格式差异，已通过正式 edit 修正；未使用 gofmt -w、未变更权限。
- 尚未运行产品测试、构建或应用。根代理独占串行 gate.sh test 与截图；建议首批 TestWorkbench|TestFindbar|TestTabReveal，并覆盖既有列表 lifetime/组合输入/已打开文档回归。
- 原生物理桌面输入、IME、窗口关闭与发行包未在本子任务验收。

## 所有权
STOPPED_WRITING：以上外壳源码与测试可由根代理审阅/串行验证；收到明确修正任务后才恢复写入。
