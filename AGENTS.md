# AGENTS.md · 上手手册

> 给 AI 助手与**新加入的人**。目标：十分钟内能安全地改代码，而不是先踩一遍别人踩过的坑。
>
> 人类贡献者的流程在 [`CONTRIBUTING.md`](CONTRIBUTING.md)，决策的理由在
> [`docs/decisions/`](docs/decisions/README.md)。本文只讲**动手时必须知道的事**。

---

## 一、这个项目是什么

一款**纯本地**的桌面编辑器，四类文档（Markdown / 代码文本 / 思维导图 / draw.io 图表）
在同一个工作区里作为一等公民共存。技术栈：**Tauri 2 + React 19 + TypeScript（严格模式）**。

**没有任何网络请求。** 没有服务端、没有账号、没有遥测。
看到「加一个上传/同步/登录」的需求时，先读 ADR 0001 —— 那是有意砍掉的。

---

## 二、三条硬约束

### 1. 依赖方向是单向的

```
modules/（内核）  →  core/editor/types.ts（契约）  ←  ui/（外壳）
```

- 外壳**不许** import 任何具体内核（Milkdown / CodeMirror 的名字不许出现在 `ui/` 里）。
- 内核**不许**直接碰文件系统。
- 内核与 UI **不许** import `@tauri-apps/*`（只有 `core/platform/desktop.ts` 可以）。

### 2. 绝不静默

不静默转码、不静默覆盖、不静默丢内容、不静默降级。
任何降级都要在 UI 上说清楚（「N 张图片读不到，只保留了原链接」）。
这是 ADR 0003 与 `docs/format-spec.md` §8 定的红线。

### 3. 打开任何文件都不能失败

`modules/plaintext` 是兜底内核，**不注册进注册表**，而是作为常量直接交给外壳 ——
这样它不可能被某个内核的错误配置遮蔽掉。任何改动都不能破坏这条。

---

## 三、代码地图

| 想改什么 | 去哪 |
|---|---|
| 文档的打开/保存/冲突/编码 | `core/document/`（**先读 `store.ts` 顶部的「核心不变量」注释**） |
| 多标签、文件树 | `core/workspace/` |
| 快捷键、命令、命令面板 | `core/commands/`（能力清单在 `definitions.ts`） |
| 查找替换的匹配语义 | `core/search/find.ts`（三个内核共用） |
| 大纲提取 | `core/outline/outline.ts` |
| 字数 / 阅读时长 | `core/statistics.ts`（Word 口径，内核无关） |
| **哪些文件算 Markdown** | `core/path.ts` 的 `MARKDOWN_EXTENSIONS` / `isMarkdownPath()`。**名单只有这一份**，四个地方在用它 |
| 选中浮层工具条（格式按钮、浮层位置） | `ui/SelectionToolbar.tsx`（外壳画浮层）+ `modules/markdown/selection.ts`（内核报选区状态） |
| 斜杠菜单（`/` 插入块） | `ui/SlashMenu.tsx`（外壳画菜单与键盘导航）+ `modules/markdown/selection.ts` 的 `describeSlash`（内核判断触发） |
| 导出 | `core/export/`（`document.ts` 组装，`inline-html.ts` 加工，`export-document.ts` 动作） |
| Markdown 编辑体验 | `modules/markdown/` |
| 代码编辑体验 | `modules/code/` |
| 界面布局 | `ui/App.tsx` + `ui/styles.css` |
| 某个面板 | `ui/DocumentPane.tsx` / `TabBar.tsx` / `Sidebar.tsx` / `FindPanel.tsx` … |
| 文件读写、编码探测 | `src-tauri/src/fs/` |

### 内核能力怎么加（不要破坏外壳的边界）

外壳**不认识任何具体内核**。要给内核加一种能力，走这条路：

1. 加在 `core/editor/types.ts` 的 `EditorAdapter` / `EditorHandle` 上（**契约**）；
2. 各内核按需实现（不实现的要能被外壳识别为「没有这个能力」）；
3. 外壳只问契约，不问内核。

已有四例可以照着看：

- `EditorHandle.search` / `goToOutlineItem` —— 可选能力，**缺了显示为禁用**；
- `EditorAdapter.sourceView` —— 源码视图，**不注册进注册表**
  （同一路径只能有一个内核答案，见 ADR 0014），缺了显示为禁用；
- `EditorHandle.formatting` + `EditorMountProps.onSelectionChange` —— 选中浮层工具条，
  **缺了就不显示浮层**（不是禁用）。与上面两例的差别是刻意的：
  源码视图是「有，但这份文档用不上」，格式化是「这个内核根本没有」——
  一个纯文本编辑器弹出加粗按钮，用户点下去只会什么都没发生。
- `EditorMountProps.onSlashChange` + `EditorFormatting.insertBlock` —— 斜杠菜单。
  这里有一个容易做错的分工：**触发判定（`describeSlash`）必须留在内核**，
  因为「光标在不在段落末尾」「有没有选中文字」是文档模型的问题；
  **菜单长什么样、有哪些项、选中哪一项、键盘怎么走全在外壳**，
  因为那是用户界面。`insertBlock` 与 `toggleBlock` 分开，是因为前者多一件事要做：
  先删掉 `/query` —— 要删哪几个字符是文档坐标，只有内核知道。

**反例**（不要这样做）：在外壳里写 `if (adapter.id === 'markdown')` 或
`resolveAdapterById('code')`。那等于把内核知识漏进外壳，
换内核时外壳要跟着改 —— 那正是这套抽象要避免的事。

**另一条容易忘的**：内核里任何**绕过键盘事件**去改文档的能力，
都必须自己给事务打上「这是用户发起的编辑」标记（`USER_EDIT_META`），
否则文档不会变脏、用户按 `Ctrl+S` 写回磁盘的是改动**之前**的内容。
已经踩过三次：查找替换（`setup.ts` 的 `replace` / `replaceAll`）、
选中浮层工具条与斜杠菜单（`setup.ts` 的 `asUserEdit` 统一包住）。
表格操作、图片插入会再遇到 —— **新加一条这类能力时，先问自己打没打这个标记。**

---

## 四、十个已经踩过的坑（省下你的两小时）

### 1. Markdown 内核的「变更上报」不能用防抖

`@milkdown/plugin-listener` 的 `markdownUpdated` 是 **200ms 防抖**触发的。
我们在 dispatch 时设的「这次是程序写入」标志，等到 200ms 后回调才去读，早就清掉了。
后果是「从磁盘重新加载后，一个字都没改的文档立刻显示成脏」，用户被引着去保存，
一保存就把文件里他没碰过的排版改掉。

**上报规则是「只有用户交互之后的变更才算用户改动」**（见 `modules/markdown/plugin.ts`）。
键盘输入、输入法上屏、粘贴、剪切、拖放、撤销/重做会「上膛」；
初始化、`setMarkdown`、插件自己补段落的事务都不上报。

**另外**：由用户动作触发但**不经过键盘事件**的编辑（查找替换、将来的格式化）
必须显式打 `USER_EDIT_META`。少了它，替换结果不会上报，
用户按保存时写回磁盘的是替换前的旧内容 —— **这是静默丢数据**。

### 2. 打开路径上的两个「静默毁文件」问题

- **mdast 里「声明为字符串、实际是 null」的字段必须收敛成空字符串**
  （`modules/markdown/normalize-ast.ts`）。不加这一步，**没有标题的图片会被静默丢出文档**。
  注册必须走 `config`（早于 parser 创建），不能等编辑器建好再挂。
- **带 YAML front matter 的文件不能让富文本内核碰**。它会把 `---` 当分隔线、
  把 `键: 值` 当正文，保存时元数据就没了。走 `canHandle` 否决，
  注册表会继续找下一个候选（现在是代码内核接住）。

### 3. `EditorHandle.setValue` 必须避免回声

程序写入的内容变化**不能再回调 `onChange`**，否则「外壳设值 → 内核报变更 → 外壳设值」成环。
两个内核各自有机制：Markdown 用 `SILENT_META`，代码内核用 `SILENT` 注解。

同理，外壳那边用 `lastReportedRef` / `lastPushedRef` 两个基准去重（见 `DocumentPane.tsx`）。

### 4. 内核是异步创建的，句柄可能指向已销毁的实例

Markdown 内核是 `await Editor.make()...create()`。换文件、**切视图**时新的内核还没就绪，
而同步的 effect 已经跑了 —— 只凭「句柄存在」就往上写，会写到一个已经销毁的内核上。

所以 `DocumentPane` 里有 `handleKeyRef`：句柄必须带上它属于**哪一次内核实例**，
标识是 `会话号:视图模式`。

> **别把它简化回 `会话号`。** 切视图时会话号**不变**，只用会话号的话，
> 那一段异步窗口里旧句柄会被误认为「有效」。同一个标识也用在失败标记
> （`failedKey`）上，因此一次内核启动失败不会把用户永久锁在兜底内核上。

### 5. vitest 默认 `css: false`，会让「CSS 是否真的内联」这类断言永远通过

导出测试要验证「版式 CSS 被内联进单文件 HTML」，而那条链路走
`import styles from './document-styles.css?raw'`。不开 `css: true` 的话，
CSS 模块被替换成空实现，`?raw` 也拿不到内容 —— **测试假绿**。
已经在 `vite.config.ts` 的 `test` 里显式开了。

### 6. `cargo check` 通过但 `cargo test` 报 `LNK1181`

手删过 `target/debug/build/qingmo-*/out` 会导致这个：那里除了 codegen 资产，
还有 `tauri-build` 生成的 `resource.lib`。`cargo check` 不链接所以看不出来。
**用 `cargo clean -p qingmo`，不要手删 cargo 的输出目录。**

### 7. jsdom 缺三个 DOM 接口，报错位置与被测代码无关

`Range.prototype.getClientRects`、`Range.prototype.getBoundingClientRect`、
`HTMLElement.prototype.scrollIntoView` **在 jsdom 30 里都没有**
（`Element.prototype.getClientRects` 倒是有）。

不补的话，任何触发 ProseMirror `scrollIntoView()` 的操作 —— 查找跳转、
浮层工具条格式化 —— 都会抛 `target.getClientRects is not a function`，
栈顶停在 `prosemirror-view` 里，**看起来像编辑器坏了**。
补丁在 `src/test/jsdom-gaps.ts`（`vite.config.ts` 的 `test.setupFiles` 全局加载）。
它只补缺口、不覆盖 jsdom 已有的实现 —— 否则将来 jsdom 补上了，
我们反而用一个更假的版本盖掉它，而且没人会注意到。

### 8. 在测试里造 ProseMirror 选区，有两个坑

写「选中文字」的测试时走**真实 DOM 选区** + `document.dispatchEvent(new Event('selectionchange'))`
是对的（这正是浏览器里发生的事）。但有两处会让用例**假绿**：

1. **不能只 `selection.removeAllRanges()`**。ProseMirror 会看到「焦点不在自己身上」，
   直接跳过这次事件，连事务都不派发 —— 内核根本不知道选区没了。
   要模拟「用户点掉选中」，得在编辑器里放一个**塌缩的光标**。
2. **塌缩点不能落在文档最开头**。`DOMObserver.flush()` 里有一条：若「刚获得焦点 200ms 内」
   +「选区塌缩」+「塌缩到 `Selection.near(doc.resolve(0))`」，它认定这是**浏览器自作主张
   把选区重置到了文档开头**，于是把 state 里的选区写回 DOM、丢弃这次塌缩。
   这是真实存在的浏览器行为，PM 处理得对；但测试正好踩上就会看到
   「选了、取消了、浮层还在」，然后去怀疑一个根本没坏的实现。

（两条都记在 `App.test.tsx` 的 `clearSelection` 注释里，改那里之前先读它。）

### 9. 浮层菜单的键盘必须挂在 `window` 的**捕获阶段**

菜单打开时焦点**还在编辑器里**（用户正在打字），所以 `keydown` 的事件目标是
contenteditable。ProseMirror 的 `handleKeyDown` 挂在 `view.dom` 上，
在**冒泡路径上先到**：默认方式（`window` 冒泡）监听的话，等到我们收到回车时，
ProseMirror 已经把段落拆开了 —— 结果是「插入了块，但也多出一个空行」。

捕获阶段从 `window` 往下走，先于目标元素，所以能先一步
`preventDefault()` + `stopPropagation()` 把这一下拿走。
`stopPropagation` 还顺带拦住了外壳自己的全局快捷键（`App.tsx` 也在 `window` 上监听），
否则 `Esc` 会顺手关掉查找栏。

**配套的另一条**：菜单项按钮上要 `preventDefault` 掉 `mousedown`，
否则浏览器先把焦点挪到按钮上、ProseMirror 的选区随之丢掉，
菜单会在 `click` 到达之前被卸载 —— 表现成「按钮点不动」。
（链接输入框相反：它必须能拿到焦点。）

### 10. 测试里「打字」必须复现浏览器的链路，不能直接塞事务

jsdom 没有真实键盘。直接往 `view.state` 里塞事务能造出文档状态，但会**绕过**
整条输入链路（DOM 变更 → `MutationObserver` → `DOMObserver.flush()` →
`readDOMChange` → `input` 事件 → `markdownEmitPlugin`）——
而斜杠菜单、输入规则、变更上报全都挂在这条链上，绕过它测试就只验证了同义反复。

正确做法在 `src/test/typing.ts`（`typeInto`），两个顺序不能反：

1. **先改 DOM，让出一个宏任务，再派发 `input`。**
   `MutationObserver` 的回调是微任务，ProseMirror 靠它把变更记进队列；
   `input` 处理器只负责「立刻结算队列」。顺序反了队列是空的，
   这次输入被**完全忽略而且不报错** —— 表现为「测试里打字没反应」。
2. **之后还要等一帧。** 浮层上报是用 `requestAnimationFrame` 合并的
   （拖选时每个 mousemove 都触发一次，逐次去量会强制布局），
   `setTimeout` 走不到它。

同理，**任何断言「内核上报了什么」的用例都要先等一帧** ——
不等就会断言到上一个状态，那是假绿。

---

## 五、工作流

```bash
npm run check        # = typecheck + lint + test。提交前必须全绿
npm run lint         # 单独跑 ESLint（check 与 CI 都已包含）
```

> **lint 是阻塞的。** 它曾经以「非阻塞」引入（首轮有 33 条历史问题），
> 欠账清零后已转为强制门禁。「有规则但不执行」比「没有规则」更糟 ——
> 前者会让人以为已经守住了。
>
> 配置里**刻意只开能抓真问题的规则**：不开风格规则，也不开
> `eslint-plugin-react-hooks` 的 `recommended`（它会把整套 React Compiler
> 规则一起打开）。要加规则，先想清楚它抓的是哪一类真 bug。

> **注意 `package.json` 里有两个 TypeScript 大版本**（`typescript` 指向 TS 6、
> `@typescript/native` 指向 TS 7）。这是为了让 `typescript-eslint` 能跑起来，
> 理由与重审条件写在 `eslint.config.js` 头部。**不要「顺手」把它改回 TS 7。**
>
> 代价要知道：**lint 跑在 TS 6 上，`tsc` 跑在 TS 7 上，两者可能对同一段代码
> 给出不同的类型**（已实测遇到过：TS 6 把闭包赋值后的变量收窄成 `never`，
> TS 7 正确得到 `number`）。所以「`tsc` 通过」不等于「lint 通过」。
> 撞上这种分歧时改写法，不要赌哪个版本更聪明。

改完代码之后：

1. **纯逻辑放 `core/`，配 `*.test.ts`。** core 里的每个模块都该有测试。
2. **有长期影响的取舍 → 新增一条 ADR**（`docs/decisions/NNNN-*.md`），
   并更新索引。**ADR 必须写「重审条件」** —— 没有重审条件的决策记录会变成后人不敢碰的祖训。
3. **注释写「为什么」。** 判断标准：这条注释是否记录了某次踩坑、
   某个反直觉的取舍、或某个「换个写法就会坏」的约束？
   代码里已有的注释密度是有意的，不要以「太啰嗦」为由删掉。
4. 用户可见的变化 → 更新 `CHANGELOG.md`；阶段性的 → 更新 `ROADMAP.md`。

---

## 六、现在的状态（2026-09-29）

**已完成**：P0–P3 的大部分 —— 文档生命周期、Markdown 内核、自包含导出、
多标签工作区、文件树、代码内核、命令面板、查找替换、大纲、主题。

**Markdown 完善（P3.5，对标 horseMD）**：字数统计、源码模式切换（Ctrl+/）、
**选中浮层工具条**、**斜杠菜单（`/` 插入块）**已交付；
表格 / 代码块增强、图片落盘、KaTeX、Mermaid、位置记忆待做。
清单在 `ROADMAP.md` 的 P3.5。
**前四项都没有引入新依赖** —— `@milkdown/kit` 已经带齐大部分积木。

**最该先做的一件事**：跑 `docs/open-decisions.md` 里的 **V3**
（Milkdown 长文档输入延迟 ≤ 50ms）。它的结论决定后续「Markdown 编辑体验怎么走」，
而且**只能在真实 WebView 里测**（jsdom 的耗时没有参考价值）。

> **一条纪律**：`open-decisions.md` 里的 V1–V4 在没有结论之前，**不要再新增待验证项**。

**空壳模块**（返回 `NOT_SUPPORTED`，等对应阶段落地）：
`src-tauri/src/{watch,search,drawio}/mod.rs`。

---

## 七、明确不要做的事

| 做法 | 原因 |
|---|---|
| 参考 Typora 的代码 | 闭源商业软件。只做功能对标 |
| 参考 Notepad++ 的代码 | GPLv3，会传染整个项目 |
| 用 draw.io 的名称/图标做产品标识 | 商标约定（ADR 0005） |
| 引入 GPL 系依赖 | 同上 |
| 用 `.mindmap` 的完整内核 schema | 规范刻意只固定最小骨架，绑定内核等于让换内核时用户的文件失效 |
| 为「可能被否的方案」先写代码 | 先跑验证。ADR 0005 的 R1 就是这么处理的 |
