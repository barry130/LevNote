# 轻笺 · levnote

**当前开发主线是 LevNote 原生版（Go + MyGo 原生 Surface，2026-11-08 迁移）。**
它是纯本地桌面应用：没有账号、同步、遥测或网络请求。窗口和编辑区不在浏览器
页面中运行；文本编辑、Markdown、思维导图和 draw.io 格式处理由本项目实现。
当前能力是有限但明确的子集，尚未等价迁移旧版全部功能。

**2026-11-14 当前源码状态：有限源码收敛，不是新应用交付 GO。**
六产品根在本地 Windows / Go1.27.3 / offline vendor 下的一次 test 为
3,019 PASS（1,181 parent + 1,838 nested）、0 FAIL、49 SKIP（47 parent + 2 nested），
26 测试包 PASS、1 host 包无测试，wall45.295191s；vet/build 各 exit0、空输出。
这些是[根独立审计回执](<F:/AIproject/LevNote/.localtools/root-integrated-audit-20261114-01/result.json>)中的源码/命令结果；
49个跳过不是通过，旧1,918项frontend测试不替代native门禁。
Root已报告阅读并接收 [Standards补篇](<F:/AIproject/LevNote/native/handoffs/autonomous-final-standards-addendum-20261114.md>)
和 [Spec补篇](<F:/AIproject/LevNote/native/handoffs/autonomous-final-spec-addendum-20261114.md>)：旧draw.io Redo S1有限闭合、
新增hard0、H1为P3非阻断启发项；两项旧P2/T05在源码层修正、无新actionable发现。
格式8项债务仍held；guard Junction和新打包初始证据目录创建均权限失败，未绕过，
本次打包候选实际GUI PE构建0、该候选host/worker启动0，不排除源码测试另建/启动helper fixture。
CI只完成禁用草案的限定语法解析，仍inactive；
[路径字面量静态复核](<F:/AIproject/LevNote/native/handoffs/autonomous-native-ci-preparation-errata-20261114.md>)已否定显示转义造成的疑似缺陷；CI仍禁止激活，不宣称runtime validated或ready。
物理GUI/IME/剪贴板、四worker真实请求、资源、当前新包进程/Job/关闭/会话及完整接受仍待证。
详情与原始SHA见[源码收敛摘要（root有限审阅已接收）](<F:/AIproject/LevNote/native/handoffs/autonomous-local-source-convergence-root-20261114.md>)；
原`.29`、历史报告和FAILED证据不改写。

当前已接受的可运行交付是 `.29`，版本 `native-worker.20261110.29`：

```powershell
.\native\bin\levnote-native-worker-20261110.29\levnote-native.exe
```

`.29` 收束连续输入法组合更新的撤销历史：同一个完整组合范围的连续更新只保留
原始文本与当前文本作为端点，取消仍可回到组合前，单步撤销/重做仍按一个事务
工作；输入回到原文时不增加空撤销项，也不会抹掉更早的重做。非连续的显式范围
替换仍按普通后缀方式追加历史，因此一次组合中出现多个不规则范围时仍可能保留
很多中间编辑。该优化只作用于文本历史，不增加后台任务、界面控件或持久格式；
没有测量整应用进程内存节省。

此前 `.28` 加入的普通文档 UTF-8 逻辑内容准入继续有效：实时内容通常以
128 MiB 为上限，
它与打开文件时 128 MiB 的磁盘读取上限相互独立。初次打开或明确重新载入得到
的已解码种子即使更大也会完整保留；这类文档以
`max(固定基值, 种子字节数)` 作为不随缩小的编辑高水位，撤销/重做/取消仍可
回到接受过的较大状态。ReplaceAll 与缩进在提交前预检，拒绝不会留下部分命令；
反向批量替换的中间状态超限时，即使最终大小可容纳也会保守拒绝。格式字段的
本地草稿在写回被拒绝后会保留，可缩短后重试或另行恢复。

输入法组合更新因大小被拒绝时，已接受的部分文本保持待处理；保存、切换视图
或标签、关闭会等待有效重试，或等待用户在编辑器中按 Esc 明确取消。焦点丢失
不会静默清掉拒绝，用户可重新聚焦编辑器，完成有效重试或按 Esc 后再继续；
被拒绝的过大字符串本身不保存。普通粘贴拒绝不会形成永久 IME 拒绝。

128 MiB 是逻辑内容边界，不是进程内存或全部文档总量上限；剪贴板、正则展开、
编解码临时字符串以及当前活动组历史仍各自占用内存。4 MiB 单草稿、16 MiB
编码后恢复、32 MiB 历史/图形解析和 100 个标签上限也仍是独立边界。

历史交付 `.27` 新增最近关闭文件的临时重开：`Ctrl+Shift+T` 或「打开」旁的
「最近关闭的文件」会按当前磁盘内容重开最后成功关闭的已保存本地文件，
菜单保留最近16个路径。成功关闭同一路径会移到顶部；文件缺失或不可读时
记录仍保留，可改选其他记录后显式重试。该记录只存在于本次运行，不恢复
已放弃的修改、未保存文档或独立恢复草稿，也不保存正文、撤销、选区或图片
字节。若可见工具字段仍在组合输入，先完成或取消，再重新触发，不排队执行。

共享标签准入上限现在是100个活动标签，与恢复容量使用同一个
`session.MaxTabs`。满100个时，新标签会提示先关闭一个再重试；已经打开的
同一路径仍直接选中，多文件打开会在逐项报告拒绝后继续处理其余路径。
拒绝不会逐出已有标签，也不会消耗最近关闭记录或静默丢弃当前草稿。
100是标签数量上限，不是内容或内存上限：128 MiB限制的是文本源文件读取，
`.28` 起另行限制普通 UTF-8 实时逻辑内容，`.29` 继续保留该边界。

历史交付 `.26` 新增明确的本地 Markdown 图片引用打开：原始
`![label](target)` 保持可见；
通过 Ctrl/Cmd+点击或 Tab+Enter 可显式打开目标，走当前本地文件打开、读取器和
字节回退路径。PNG/JPEG 大小/格式边界与图片阅读器启停不变；不会自动把该语法
渲染成内嵌图片，也不引入图片缓存。目标为 1–4096 UTF-8 字节，排除 ASCII
空白/控制字符、尖括号和圆括号；标题式与引用式图片形式不支持。格式层原样
传递目标，本地相对路径、百分号编码和 Windows 反斜杠保留；已有 `file:` URL
和标题锚点继续遵循原有宿主规则。缺失或不可读目标会如实报告；若原始字节仍
可读取，可通过字节回退建立标签。HTML 导出保留转义后的原 Markdown 文本，不
解码图片。

历史交付 `.25` 为 Markdown 表格增加列操作：在当前单元格前后插入列、删除当前列；
表格保持1–8列。操作保留幸存单元格原始字节、填充、分隔线写法、缩进、
行尾和相邻内容，仅在需要边界时添加外层管道。不安全、不支持或超预算的
操作明确拒绝，可回到整表源码编辑。单元格修改与列结构调整分别撤销。
单元格面板沿用 F6、Tab/Shift+Tab、Enter/Space；F6 前需确认输入法组合。

`.24` 在有边界的本地 PNG/JPEG 只读图片模式上增加键盘操作；这些快捷键
在图片画布获得键盘焦点时生效：`+`/`=`/`-`和数字键盘加减缩放，方向键
平移；`0` 进入适应窗口，`1` 进入明确100%（一个图片像素对应一个 DIP）。
明确100%在窗口大小变化时保留该手工缩放；按`0`切回适应窗口。激活已启用
的工具栏缩放/适应/100%按钮后，键盘焦点返回图片画布；仅聚焦工具栏不是
动作。

`.23` 引入的图片边界继续适用：「另存副本」把已核验的原始字节复制到
新目标，且不覆盖已有文件。不能解码但仍可读取原始字节的文件会回退到
字节视图；读取失败的文件如实报告，不保证其内容可检查。内置图片阅读器
可在格式插件配置中启用或停用。

图片会话恢复保存并重开路径与标签元数据，不保存图片字节或像素。编辑
后约900毫秒静默由应用计时器写入恢复状态；该调度独立于绘制，但原生
对话框或重UI工作可能延后本次写入。它从不自动写回原文件；尚未显式完成
的格式本地草稿仍不在自动捕获内，继续通过完成、放弃或恢复流程处理。
支持的子集为：源文件不超过16 MiB，单边不超过8192像素，总像素不超过
8,388,608；JPEG 仅支持 baseline SOF0、8位灰度或标准三通道色，不支持
CMYK或progressive；PNG 仅支持不超过8位、非交错、非动画。超过16 MiB
的文件仍可进行有界字节检查，但不能经该路径使用有界图片复制操作。带
图片后缀但已有文本草稿的内容仍按文本处理。

此版保留 `.21` 的 draw.io 告警修正，以及 schema3 `builtin.drawio` 数据路由和四个固定
worker：Markdown HTML、导图SVG、只读导图场景和 draw.io当前页SVG。请先在
图纸的「阅读 / 编辑」或「分屏」中选择页面，再使用「导出页面」；源码视图
会禁用该入口并说明原因。图纸源码上限为32 MiB，SVG结果上限为64 MiB，请求
默认超时30秒。导出前会同步待写回草稿；结果发布前会复核页面、视图、标签、
文档和版本身份。

更早的表格列操作、全词查找、会话恢复、UTF-8 保存、查找键盘/导航、本地引用
修正和 schema3路由继续保留。实现合同、历史包和详细证据见
[原生架构](native/ARCHITECTURE.md)、[验证记录](native/VALIDATION.md)、
[更新日志](CHANGELOG.md)和
[`.29`组合合并包验收](native/handoffs/composition-release29-root-acceptance.md)。

请整体保留目录中的 `engines/`、许可文本和说明；程序目录可移动，配置仍在
`%APPDATA%\LevNoteNative`。详细构建、能力边界、验证和第三方来源见：

- [原生版中文说明、构建与运行](native/README.md)
- [能力与实现边界](native/README.md#已有能力与边界) · [架构](native/ARCHITECTURE.md)
- [格式与插件 API](native/formats/API.md) · [验证记录](native/VALIDATION.md)
- [第三方来源与许可](native/THIRD_PARTY_NOTICES.md) · [迁移决策 ADR 0063](docs/decisions/0063-native-migration.md)
- [`.29` 组合合并包验收](native/handoffs/composition-release29-root-acceptance.md) ·
  [交付记录](native/handoffs/composition-release29-delivery.md) ·
  [源码验收](native/handoffs/active-composition-coalescing-source-root-acceptance.md) ·
  [`.28` 文档增长包验收](native/handoffs/growth-release28-root-acceptance.md) ·
  [历史交付记录](native/handoffs/growth-release28-delivery.md) ·
  [`.27` 文件工作流包验收](native/handoffs/file-workflow-release27-root-acceptance.md) ·
  [交付记录](native/handoffs/file-workflow-release27-delivery.md) ·
  [`.26` 图片引用包验收](native/handoffs/markdown-image-reference-release26-root-acceptance.md) ·
  [交付记录](native/handoffs/markdown-image-reference-release26-delivery.md) ·
  [`.25` 表格列包验收](native/handoffs/markdown-table-columns-release25-root-acceptance.md) ·
  [交付记录](native/handoffs/markdown-table-columns-release25-delivery.md)
- [`.24` 图片键盘/恢复包验收](native/handoffs/native-reading-release24-root-acceptance.md) ·
  [`.24`交付记录](native/handoffs/native-reading-release24-delivery.md)
- [历史包与失败证据](native/VALIDATION.md)

原生扩展有两类。**本地 JSON 格式配置**不执行代码：schema1/2 为代码后缀提供
高亮和字面大纲，schema3 把自定义后缀映射到固定的 Markdown、思维导图或
draw.io能力。
**本地引擎包**面向固定四个导出/预览引擎，从「插件 → 引擎包」打开
「本地引擎包」面板；导入只复制和校验完整兼容发布目录，不自动启用，显式
启用后的包会在实际导出/预览请求中启动对应本地 worker。可恢复随包或重置
全部选择，不支持任意可执行路径或命令覆盖，也没有引擎插件 API。每个引擎
最多保留12个已安装包；schema1旧选择文件按原字节读取，显式选择后持久化为
schema2的四个引擎项。

四类文档仍各有边界：代码文本是自研编辑器，Markdown 支持块编辑、有限表格和
本地链接，导图支持主题结构与基础属性，draw.io 支持基础形状与当前页 SVG；
未迁移能力在能力文档中逐项列出，不会静默降级。

`pnpm run desktop` 仍启动旧 Tauri 版。下面的功能、安装包大小、快捷键、测试数
和平台要求均为 **legacy Tauri 历史记录**，不代表原生版验收结果。

## 旧 Tauri 版（legacy）

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
  可全屏放大）。公式与图表**导出时一起带走**（公式内联字体、图表画成 SVG）；
  **内核画的字跟随界面语言**（代码块标签提示、图表与图片的说明纸条，
  切语言重画不重建 —— 光标与撤销历史不动）。
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
  **备注**（有备注的主题带 📝，点开即看即改，多行正文；大纲里同样有
  📝 指示；撤销历史找得回来；徽章是编辑器部件、不进导出图片；
  查找栏连备注一起搜，替换按主题 / 备注分通道）；
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
