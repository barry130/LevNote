# 更新日志

本文件记录**面向使用者的变化**。决策的理由与重审条件在 [`docs/decisions/`](docs/decisions/README.md)，
不做详细记录（避免两处各写一遍然后互相矛盾）。

格式参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.1.0/)。

---

## [0.3.0] — 2026-09-29

**主题：从「单文档编辑器」变成「工作区编辑器」。**

### 新增

- **多标签工作区**：一个窗口里同时打开多个文档，支持拖拽重排、中键关闭、Ctrl+Tab 循环切换。
- **文件夹工作区与文件树**：打开一个文件夹作为工作区；子目录懒加载；
  `node_modules` / `target` / `.git` 等重目录默认折叠（可一键显示）；
  打开文件时自动展开它的祖先目录链。侧栏宽度可拖拽。
- **代码 / 文本内核**（CodeMirror 6）：语法着色、行号、代码折叠、括号匹配、
  多光标、软换行开关。支持 JavaScript / TypeScript / JSX / TSX / JSON / CSS / HTML /
  XML / Markdown / Python / Rust / YAML / SQL。
- **命令面板**（`Ctrl+P`）：按标题、拼音、英文名模糊搜索；禁用的命令会显示但说明原因。
- **查找与替换**（`Ctrl+F`）：区分大小写、全词匹配、正则表达式；
  正则模式下替换文本支持 `$1` / `$&`；三个内核共用同一套匹配语义。
- **大纲面板**（`Ctrl+Shift+L`）：从 Markdown 提取标题树，点击跳转；
  自动跳过围栏代码块与 front matter 里的「假标题」。
- **主题**：宣纸（浅色）/ 松烟（深色）/ 跟随系统（`Ctrl+Shift+T` 循环）。
- **偏好持久化**：主题、侧栏宽度与可见性、大纲可见性、软换行。

### 变更

- **`.md` 文件的降级行为改善**：带 YAML front matter 的 Markdown 不再退化为白底纯文本框，
  而是由代码内核接管 —— 有行号、有语法着色的源码。元数据仍然一个字都不会被动。
- **`.txt` / `.json` / `.yml` 等文本文件**改由代码内核打开（原先走纯文本框）。
- **`Ctrl+O` / `Ctrl+S` / `Ctrl+Shift+S` 的行为不变**，但实现改走统一的命令层，
  因此命令面板里的同一个功能与快捷键的行为保证一致。

### 修复

- **查找替换在 Markdown 内核上会静默丢数据**：替换操作不经过键盘事件，
  原先的「变更上报」判定会把它当成程序写入而不上报，导致外壳持有的内容停留在替换前 ——
  用户按保存时写回磁盘的是旧内容。现在替换会显式标记为「用户发起的编辑」。
- **内核注册表的优先级与注释相反**：原实现让**最后**注册的内核排在最前，
  一旦接上第二个认得 `.md` 的内核，它会抢在 Markdown 内核前面。
- **`systemPrefersDark` 在缺少 `matchMedia` 的环境下会抛错**，导致应用无法渲染。

### 工程

- 初始化 git 仓库；新增 GitHub Actions CI（类型检查 + 测试 + 构建 + Rust clippy/test）。
- 新增 ESLint 扁平配置与 `.editorconfig`。
  **lint 目前是「非阻塞」的**：刚接入，首轮全量扫描还剩一批历史问题，
  因此 CI 里是 `continue-on-error`。收尾动作见 `ROADMAP.md` 第 20 项。
- 为让 `typescript-eslint` 能跑起来，`typescript` 暂时指向 TS 6
  （`@typescript/native` 保留 TS 7）。取舍与重审条件写在 `eslint.config.js` 头部。
- 新增 `README.md` / `ROADMAP.md` / `CONTRIBUTING.md` / `SECURITY.md` / `AGENTS.md`。
- 新增 ADR [0010](docs/decisions/0010-multi-tab-workspace.md) /
  [0011](docs/decisions/0011-code-kernel-and-graceful-downgrade.md) /
  [0012](docs/decisions/0012-command-registry.md)。
- 新增 [`docs/gap-analysis-horsemd.md`](docs/gap-analysis-horsemd.md)：与成熟对标项目的差距分析与技术路线。
- 测试从 162 条增加到 300+ 条（前端）。

---

## [0.2.0] — 2026-09-28

**主题：Markdown 内核 + 自包含 HTML 导出。**

- Markdown 所见即所得编辑（Milkdown / ProseMirror 7.22.2）：GFM、表格、任务列表、
  代码块高亮、引用、图片。
- **自包含 HTML 导出**：单文件、内联 CSS 与图片、断网双击即开；
  导出时如实通报「N 张图片已内联 / M 张读不到」。
- 打开路径的两道防线：mdast 规整（否则没有标题的图片会被静默丢出文档）、
  带 front matter 的文件降级处理。
- 变更上报自研（不用 `@milkdown/plugin-listener`）：它的 200ms 防抖会让
  「这次变更是程序写入的」判断失效，导致「打开后一个字没改就显示成脏」。

---

## [0.1.0] — 2026-09-28

**主题：文档生命周期跑通。**

- 打开 / 编辑 / 保存 / 另存为 / 重新加载。
- **脏判定只认内容与格式**，不看时间戳 —— 「打开又什么都没改」不会被当成待保存。
- **绝不静默转码**：BOM 优先探测，其次字符集统计；探测失败如实标记 `lossy`；
  写回原编码；不可表示时报错并取消写入；BOM 与行尾是文件属性。
- **原子写入**（临时文件 → fsync → rename）+ **mtime 乐观锁**：
  磁盘被外部改写时保存失败并让用户选「重新加载 / 覆盖 / 另存为副本」。
- 平台抽象：`DocumentProvider` 是唯一的平台差异点，纯 `vite dev` 下可跑内存虚拟盘。
- 编辑器适配层契约 + 兜底内核：**任何文件都必须能被打开**。
