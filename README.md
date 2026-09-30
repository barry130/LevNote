# 轻墨 · qingmo

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
- **自包含 HTML 导出**：单文件、内联 CSS 与图片、断网双击即开。
- **绝不静默转码**：探测失败如实标记，写回原编码，BOM 与行尾是文件属性。
- **原子写入 + mtime 乐观锁**：写到一半崩溃不会毁掉文件，也绝不静默覆盖别人的改动。
- 安装包约 **3.2 MB**（Tauri 2，不是 Electron）；KaTeX 与 Mermaid 都按需加载，不进主程序。

> **当前进度**：四类里 **2/4 已落地**（Markdown、代码 / 文本）；
> 思维导图（mind-elixir）与 draw.io 是 P4 / P5，各自被一项验证前置
> （V2 压测 / V1 查看器独立渲染）。见 [`ROADMAP.md`](ROADMAP.md)。

---

## 快速开始

```bash
npm install
npm run desktop      # 真桌面版（Tauri 2，需要 Rust 工具链）
npm run dev          # 只跑前端（浏览器预览，内存虚拟盘，用于调界面）
npm run test         # 单元测试（621 条）
npm run typecheck    # 类型检查
npm run check        # 上面两条加 lint 一起跑 —— 提交前必须全绿
npm run lint         # ESLint（类型感知）。**阻塞**，配置里没有关掉的规则
```

从源码构建安装包：

```bash
npm run desktop:build
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
│   ├── outline/              #   Markdown 大纲提取
│   ├── export/               #   自包含 HTML 导出
│   ├── highlight/            #   代码高亮（编辑器与导出共用）
│   ├── theme/ settings.ts    #   主题与偏好
│   └── platform/             #   唯一有平台差异的地方
├── modules/                  # 编辑器内核。外壳不认识它们
│   ├── markdown/             #   Milkdown / ProseMirror
│   ├── code/                 #   CodeMirror 6
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
- 第三方依赖的许可证清单见 `ROADMAP.md` 的 P7 项。
