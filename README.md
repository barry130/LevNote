# 轻笺 · levnote

> **Typora / Notepad++ / 思维导图 / draw.io —— 四合一，纯本地。**
>
> 这四类文档不是四个拼在一起的独立工具，而是**同一个工作区里的一等公民**：
> 可以互相引用、互相嵌入、块级就地编辑。这是本项目的产品定位，不是「功能清单」。

- **多标签 + 文件夹工作区**：一个窗口装下一个文件夹，每个打开的文件是一个标签页。
- **所见即所得的 Markdown**（Milkdown / ProseMirror）：GFM、代码高亮、表格、任务列表；
  选中浮层工具条、`/` 斜杠菜单、表格可视化编辑、代码块语言选择器与行号；
  **粘贴图片自动落到文档旁边的 `<文档名>.assets/` 并写相对路径**，点图看大图；
  **LaTeX 公式**（`$…$` 与 `$$…$$`，点一下改源码）；
  **Mermaid 图表**（```` ```mermaid ```` 围栏里写，光标进块看源码、离开看画好的图，
  可全屏放大）。公式与图表**导出时一起带走**（公式内联字体、图表画成 SVG）。
- **代码 / 文本**（CodeMirror 6）：语法着色、行号、折叠、括号匹配。
- **思维导图**（mind-elixir 5.15.1）：`.mindmap` 是一份**结构自持的 JSON**
  （`id` / `topic` / `children` 是三家候选内核的最小公因式），
  换内核不让用户的文件失效；四种布局、折叠、拖动重排、右键菜单、箭头与摘要；
  **导出成 PNG / SVG 图片**（导图画出来的样子只有内核知道）；**大纲面板**
  （侧栏平铺整棵树，点条目跳到主题、折叠的分支自动展开）；
  **标记**（优先级 / 色标 / 符号，右键挂上，字形直接存进文件、大纲里跟着显示）；
  **插图**（Ctrl+V 把图片粘上主题，存进 `<文档名>.assets/` 不进 JSON，
  图丢了画一张写明路径的占位图，右键可看引用、可移除）；
  **链接与标签**（右键挂 🔗 与标签 chip，🔗 单击即走，指向工作区里的
  文件时点开就是那一份；导出 PNG / SVG 时链接保留、标签画成文字）；
  **打开不改再保存，逐字节一致**；坏文件自动降级成有 JSON 着色的源码视图，
  不会打不开。也可以切到源码视图直接改 JSON（`Ctrl+/`），或与导图并排分屏。
- **自包含 HTML 导出**：单文件、内联 CSS 与图片、断网双击即开。
- **绝不静默转码**：探测失败如实标记，写回原编码，BOM 与行尾是文件属性。
- **原子写入 + mtime 乐观锁**：写到一半崩溃不会毁掉文件，也绝不静默覆盖别人的改动。
- 安装包约 **3.2 MB**（Tauri 2，不是 Electron）；KaTeX 与 Mermaid 都按需加载，不进主程序。

> **当前进度**：四类一等公民 **4/4 已落地** —— Markdown、代码 / 文本、
> 思维导图（P4 第 20–23 项）、draw.io 只读查看器（P5-25）。
> **思维导图**（2026-10-04）：`.mindmap` 是 JSON、结构自持（`docs/format-spec.md` §7.1），
> 编辑走 mind-elixir 5.15.1，**打开不改再保存逐字节一致**；工作区**全文检索**
> （第 23 项）与侧栏第二栏同批落地。
> **draw.io**（2026-10-04，ADR 0027）：Markdown 里的 `kind: diagram` doc-block
> 渲染成真图，reference / inline 两种模式；查看器 2.654 MiB（`public/drawio/`）
> 按需注入、不进 bundle。
> 后续项见 [`ROADMAP.md`](ROADMAP.md)。

---

## 快速开始

```bash
corepack enable       # 首次：让 pnpm 命令可用（版本钉在 package.json 的 packageManager）
pnpm install
pnpm run desktop      # 真桌面版（Tauri 2，需要 Rust 工具链）
pnpm run dev          # 只跑前端（浏览器预览，内存虚拟盘，用于调界面）
pnpm run test         # 单元测试（1,467 条）
pnpm run typecheck    # 类型检查
pnpm run check        # 上面两条加 lint 一起跑 —— 提交前必须全绿
pnpm run lint         # ESLint（类型感知）。**阻塞**，配置里没有关掉的规则
```

从源码构建安装包：

```bash
pnpm run desktop:build
```

### 环境要求

| | 版本 |
|---|---|
| Node.js | ≥ 20 |
| Rust | ≥ 1.77（仅桌面版需要） |
| Windows | WebView2 运行时（Win11 自带；Win10 由安装包引导下载） |
| macOS | 13+ |
| Linux | WebKitGTK 4.1 |

---

## 项目结构

```
src/
├── core/                     # 与界面无关的内核逻辑，全部可单测
│   ├── document/             #   单文档生命周期（脏判定 / 保存 / 冲突 / 编码）
│   ├── workspace/            #   多标签工作区 + 文件树
│   ├── editor/               #   编辑器适配层契约 + 注册表
│   ├── commands/             #   命令注册表 / 快捷键 / 模糊匹配
│   ├── search/               #   查找替换的纯函数
│   ├── outline/              #   大纲提取（Markdown 与导图两份纯函数）
│   ├── export/               #   自包含 HTML 导出
│   ├── highlight/            #   代码高亮（编辑器与导出共用）
│   ├── theme/ settings.ts    #   主题与偏好
│   └── platform/             #   唯一有平台差异的地方
├── modules/                  # 编辑器内核。外壳不认识它们
│   ├── markdown/             #   Milkdown / ProseMirror
│   ├── code/                 #   CodeMirror 6（也是 mindmap / drawio 的源码视图）
│   ├── mindmap/              #   .mindmap：MindElixir，JSON 往返内核
│   ├── drawio/               #   .drawio：随包只读查看器 + 源码视图即编辑入口
│   └── plaintext/            #   兜底内核：任何文件都必须打得开
├── ui/                       # React 外壳（工作区 / 标签栏 / 侧栏 / 面板 / 浮层）
└── ../packages/interop-spec/ # 四类文档互操作规范的可执行实现
src-tauri/                    # Rust 侧：编码探测 / 原子写入 / 文件读写
docs/                         # 规范、ADR、差距分析、开放问题
```

---

## 架构要点

### 1. 依赖方向是单向的

```
modules/（内核）  →  core/editor/types.ts（契约）  ←  ui/（外壳）
```

外壳**不 import 任何具体内核**。想知道当前用的是 Milkdown 还是 CodeMirror，
得去问 `adapter.label`。加一个内核只需要在 `modules/index.ts` 注册一次。

### 2. 平台差异只有一个出口

`DocumentProvider` 是唯一有平台差异的接口。桌面端转发给 Rust 命令，
浏览器端是内存虚拟盘。**内核与 UI 都不许直接引用 `@tauri-apps/*`。**

### 3. 能力只有一份清单

`core/commands/definitions.ts` 是「这个软件能做什么」的唯一来源。
工具栏按钮、命令面板、快捷键都从它派生，因此不可能出现
「按钮上有但面板里搜不到」或「两个入口行为不一致」。

### 4. 降级要掉到「次优」，不是「最底」

内核通过 `canHandle` 可以否决自己，注册表会继续找下一个候选。
带 YAML front matter 的 `.md` 就是这条规则的受益者：Markdown 富文本内核退位后，
**代码内核接住它** —— 用户拿到的是有行号、有着色的源码，而不是一个白底 textarea。

### 5. 四类文件各自是一等公民

`.md` / `.mindmap` / `.drawio` / 代码文本各有自己的内核、自己的新建入口，
单文件打开就画出来，不是「只能在 Markdown 里顺手做一下」。
新文件该写什么内容只有一份出口（`core/new-file-content.ts`），
新建的三个入口（命令面板 / 侧栏 / 示例文档）共用一条编排（取舍见
[ADR 0048](docs/decisions/0048-mindmap-and-drawio-are-first-class-too.md)）。

**`.drawio` 现在是只读的** —— 随包只有 `public/drawio/viewer.min.js`（2.654 MiB），
编辑器本体没随包。改图纸的入口是 **源码视图（Ctrl+2）**，那里面的 XML 就是文档内容。
面板上一直挂着这句话，不靠「没那个按钮」让你自己发现（硬约束 2）。

实测读数（`node scripts/v4-run.mjs`）：随包那份产物里的 `Editor` 是
`function () { return new mxCellEditor(this) }` 这个替身，**跑不起 draw.io 的编辑器**。
要接真编辑器，取 draw.io **webapp** 那份未改动产物放进 `public/drawio/` 下面即可
（需要 `index.html` + `js/app.min.js` + `styles/` + `images/`，与已有的 `shapes/` `stencils/`
同源；`viewer.min.js` 就是这么取的，下载命令见 `scripts/v1-run.mjs`，体积与许可记录见
[ADR 0027](docs/decisions/0027-drawio-viewer.md)）——
**确切的上游地址本轮没联网核过，动手前先照 `viewer.min.js` 的来源确认一遍**。
接进来之后 `modules/drawio/` 换一个 `Component` 就行，**内核契约一个字都不用改**。

---

## 文档

| 文档 | 内容 |
|---|---|
| [`docs/format-spec.md`](docs/format-spec.md) | 四类文档的互操作规范（`doc-block` 语法、编码红线、`.drawio` 往返硬要求） |
| [`docs/decisions/`](docs/decisions/README.md) | ADR：每个有长期影响的取舍，**都带「什么条件下该重新考虑」** |
| [`docs/open-decisions.md`](docs/open-decisions.md) | 待验证项（V1–V4）、被推迟的功能、被阻塞的前置条件 |
| [`docs/gap-analysis-horsemd.md`](docs/gap-analysis-horsemd.md) | 与成熟对标项目的差距分析与技术路线 |
| [`ROADMAP.md`](ROADMAP.md) | 已完成 / 进行中 / 计划中 |
| [`AGENTS.md`](AGENTS.md) | 给 AI 助手与新成员的上手手册 |

---

## 许可与商标边界

- 本项目**只做功能对标**，不接触以下项目的代码：
  **Typora**（闭源商业软件）、**Notepad++**（GPLv3，任何代码复用都会传染整个项目）。
- `draw.io` 源码为 Apache-2.0，但**商标另有约定**：不得使用其名称与图标作为产品标识，
  不得以暗示关联或背书的方式使用。UI 中只写「支持 `.drawio` 格式」。
- 本项目以 **[MIT](LICENSE)** 许可证发布；第三方依赖的许可证分组清单在
  [`docs/dependency-licenses.md`](docs/dependency-licenses.md)（`pnpm run licenses` 重新生成）。
