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

## 二、硬约束

**这份清单是编号的唯一来源。** 代码注释里写的「硬约束 N」指的就是下面第 N 条 ——
改任何一条之前，先去找它的完整理由（在对应的 ADR、或那一行代码旁边的注释里）。

前三条是地基（破坏它们会让整个架构塌掉），后面十一条是「换个写法就会坏」的具体约束。

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

### 4–14

4. **注册顺序即优先级**：`registerAdapter` 是 `push` + 先注册先匹配。
   **备胎必须排在正主后面** —— 现在的顺序是 `markdown` → `mindmap` → `code`，
   而 `code` 认得 `.md` 与 `.mindmap`（它是两者的源码形态）。
   把它排前面会把两个正主全吃掉：`matches` 先命中、它又没有 `canHandle`，
   注册表直接返回它，后面那个内核**一次都不会被调用**。
   踩过一次（第一版就是 `markdown` → `code` → `mindmap`），
   `modules/wiring.test.ts` 的备胎链测试当场抓住 —— 见 ADR 0025。
5. **Markdown 扩展名名单只有一份**：`core/path.ts` 的 `MARKDOWN_EXTENSIONS`
   （`isMarkdownPath()`）。抄多份的后果是**双重落空** —— 用户拿到白底 textarea，
   而两边代码都没写错。`modules/wiring.test.ts` 跨模块断言钉住它。
6. **`EditorAdapter.sourceView` 不注册进注册表**：注册表按路径挑内核，
   同一路径只能有一个答案（ADR 0014）。源码视图是「同一路径的第二种看法」。
7. **内核里任何绕过键盘事件改文档的能力，必须自己给事务打 `USER_EDIT_META`。**
   `markdownEmitPlugin` 只在「刚发生过用户输入」的窗口内上报，而查找替换 /
   浮层按钮 / 斜杠菜单**不产生 `input` / `paste`**。漏了 = **静默丢数据**
   （文档不变脏，Ctrl+S 写回改动**之前**的文本，界面还显示「已保存」）。
   已踩**六次**。`withUserEdit` 与 `withUserEditOnView` 是同一个东西的两半，不要各写一份。
   **图表预览是唯一不需要打标记的**（只改选区、不改文档）。
8. **`beforeinput` 是内核判定「这次变更是不是用户干的」的唯一可靠依据。**
   ProseMirror **不监听 `input`**（靠 `MutationObserver`，回调是微任务，
   跑在任何事件派发之前）。只挂 `input` 会让**第一次**变更被安静丢掉。
9. **浮层菜单的键盘必须挂在 `window` 的捕获阶段**（`addEventListener(..., true)`）。
   默认监听的话回车会先把段落拆开 —— ProseMirror 的 `handleKeyDown`
   在 `view.dom` 的冒泡路径上先到。配套：菜单项按钮要 `preventDefault` 掉 `mousedown`。
10. **浮层 / 面板不许持有文档坐标。** `setNodeAttribute` **不检查节点类型** ——
    位置过期会安静写到另一个不相干的节点上。用 `resolvePos: () => number | null`，
    **写的那一刻现问**。
11. **外壳不许按类名 / DOM 结构去认内核画的东西。** 正确做法是**加一条契约方法**
    （`EditorMountProps` 上的 `onImageClick` / `onDiagramOpen` …），
    监听挂在内核自己造的元素上。判断依据：**只要需要外壳知道内核内部的 DOM 结构 /
    类名 / 节点类型名，它就该是一条契约方法。**
12. **`joinPath` 把相对部分的分隔符统一成 `base` 的风格。** Markdown 图片地址必须
    正斜杠、Windows 文档路径是反斜杠；不统一则「写入路径」与「回读路径」是**两个字符串**，
    只在**按字符串比较**处出问题（缓存键 / 去重）→「图写进去了但显示不出来」。
    **只换风格**，不折叠重复分隔符、不解析 `..`。
13. **样式表按「属性」划所有权，不按文件。** 内核根节点的**滚动**由内核自己的样式表声明
    （Markdown 是 `.qm-editor`，CodeMirror 是 `.cm-scroller`）；外壳只声明
    「把内核根节点压进 flex 容器」（`flex` / `min-height`）。
    两边都声明同一个属性时**谁赢只看打包顺序**，而 `styles.css` 在产物里靠后、
    会**安静地赢** —— 表现是「打开长文档滚不动」，且**从 `a224ec4` 就存在**
    （jsdom 没有布局引擎，照不出来）。回归测试 `src/ui/styles.test.ts`。
    > 硬约束 11 的同胞：11 说「外壳不许按类名认内核画的东西」，这条说「也不许替内核决定属性」。
14. **构建期注入的常量必须写成 `import.meta.env.X`，不能写成裸标识符。**
    Vite dev server **不替换裸标识符形式的 `define`**（`vite build` 与 vitest 都替换
    → **单测全绿而开发时崩**）。踩过：设置页的版本号渲染时抛 `ReferenceError`，
    没有错误边界，**React 卸掉整棵渲染树** —— 点一下设置应用就没了。
    **凡是依赖构建期替换的东西，都要在真正跑起来的那条路径上验一次。**

---

## 三、代码地图

| 想改什么 | 去哪 |
|---|---|
| 文档的打开/保存/冲突/编码 | `core/document/`（**先读 `store.ts` 顶部的「核心不变量」注释**） |
| 多标签、文件树 | `core/workspace/` |
| 快捷键、命令、命令面板 | `core/commands/`（能力清单在 `definitions.ts`） |
| 查找替换的匹配语义 | `core/search/find.ts`（三个内核共用） |
| **视图偏好（主题 / 侧栏 / 行号 / 位置记忆开关）** | `core/settings.ts`（`localStorage`，逐字段校验）。**设置页只是它的一个界面** —— 加偏好项改这里，别在 `SettingsDialog.tsx` 里另起一份 |
| **「上次读到哪儿」（光标 / 滚动）** | `core/positions.ts`（存储：独立键 + LRU 200 条）。**位置的坐标系由内核自己定**，外壳只做往返 —— 见 ADR 0021，别想着在外壳里把它换算成「第几个字符」 |
| **滚动比例那四行算术** | `core/editor/position.ts`（`readScrollRatio` / `applyScrollRatio`）。三个内核共用；**别各写一份**（其中两份一定会忘了判除零） |
| 大纲提取 | `core/outline/outline.ts` |
| 字数 / 阅读时长 | `core/statistics.ts`（Word 口径，内核无关） |
| **哪些文件算 Markdown** | `core/path.ts` 的 `MARKDOWN_EXTENSIONS` / `isMarkdownPath()`。**名单只有这一份**，四个地方在用它 |
| **所有浮层摆在哪**（位置算术） | `ui/floating.ts` 的 `computeFloatingPlacement`（**纯函数**，12 条单测）。四个浮层共用：选中工具条（`center`/上）、斜杠菜单（`start`/下）、表格工具条（`center`/下）、代码块语言面板（`end`/下）。**改摆放规则改这里，不要在组件里再抄一份** —— 它被抄过两遍才提取出来的 |
| 选中浮层工具条（格式按钮、浮层位置） | `ui/SelectionToolbar.tsx`（外壳画浮层）+ `modules/markdown/selection.ts`（内核报选区状态） |
| 斜杠菜单（`/` 插入块） | `ui/SlashMenu.tsx`（外壳画菜单与键盘导航）+ `modules/markdown/selection.ts` 的 `describeSlash`（内核判断触发） |
| 表格工具条（加行加列、对齐、删除） | `ui/TableToolbar.tsx` + `modules/markdown/table.ts` 的 `describeTable`（纯函数，把光标处的表格算成数据）。**对齐是按列的**，见坑 12 |
| 代码块（行号 / 语言标签） | `modules/markdown/code-block-view.ts`（NodeView，**画在文档树里**）+ `ui/CodeBlockPicker.tsx`（外壳画语言面板） |
| **代码高亮**（编辑器与导出共用） | `core/highlight/hljs.ts` + `core/highlight/cache.ts`（**LRU 缓存**）。高亮在**每次文档变更**时整篇重跑（`codeHighlightPlugin` 的 `apply`），是按键路径上的常客 —— 所以**一律走 `highlightCode`**（它带缓存），不要绕开它直接调 hljs。详见 `docs/open-decisions.md` 的「V3 的后续」 |
| **图片存在哪、叫什么名** | `core/document/assets.ts`（目录布局与命名，**纯函数**）。`<文档名>.assets/` + `image-<时间戳>-<盐><后缀>`；Markdown 里一律写相对路径 |
| **图片的读与写** | `core/document/images.ts`（两个**无状态**的文档级动作：`readImageAsUrl` / `writePastedImage`）。**「已存在就换名字重试，绝不覆盖」这条在这里** |
| **图片类型（MIME ↔ 扩展名）** | `core/image.ts`。**只有这一份**，落盘 / 显示 / 导出三处共用 —— 抄成三份的下场是「编辑器里能看、导出后打不开」 |
| 图片显示（NodeView）与灯箱 | `modules/markdown/image-view.ts`（画占位与加载态）+ `ui/Lightbox.tsx`（外壳画全屏预览） |
| **什么算 Mermaid 图表** | `core/mermaid.ts` 的 `isMermaidLanguage` / `MERMAID_STARTER`。**判定只有这一份**，三处共用（NodeView、导出、斜杠菜单插入）—— 抄成三份的下场是「编辑器画得出、导出不画」 |
| **Mermaid 怎么渲染** | `core/mermaid-render.ts`。动态 `import('mermaid')`、**必须传容器**（不传它会往真实 `document.body` 挂临时节点）、容器必须**参与布局**（`display: none` 会让 `getBBox()` 全 0）。jsdom 里画不出图，所以测试用假渲染器 |
| **`doc-block` 怎么解析** | `core/docblock.ts`。属性区 + `---` + 载荷，载荷**逐字节原样**；不认识的键进 `extra` 原样保留（规范 §5 向前兼容，丢了就是单向数据损失）。永远不抛 —— 内容来自用户文件，失败走 `errors` |
| **`.drawio` 缺哪些 stencil 库** | `core/drawio.ts` 的 `stencilLibsOf` / `missingStencilLibs`。**渲染前从 XML 风格串里算**（`shape=mxgraph.<库名>.`），与内置清单求差 —— 不要改成「事后看请求 404」（并发归属不了，XHR 打点是全局猴子补丁） |
| **draw.io 查看器怎么渲染** | `core/drawio-render.ts`。上游产物在 `public/drawio/`（ESLint 全量忽略），**出现图表引用才注入**脚本；注入前十条资源路径必须全部占住（「无网络」的唯一开关，漏一条就静默联网）。渲染器可注入（`DrawIoIo`），与 `MermaidIo` 同一条纪律。**不接主题**（颜色在图的 XML 里，画布恒白） |
| 图 / 源码双态与全屏入口 | `modules/markdown/code-block-view.ts`（`data-mode`）+ `ui/DiagramViewer.tsx`（外壳画全屏） |
| 导出 | `core/export/`（`document.ts` 组装，`inline-html.ts` 加工，`export-document.ts` 动作） |
| **写回磁盘时被改写成什么样**（列表符号 / 分隔线 / 表格） | `modules/markdown/setup.ts` 的 `useDashSymbols`（`bullet` / `rule` 都设成 `'-'`）。**`remark-stringify` 的默认是 `'*'`**，改回去会让「打开一份用 `-` 写的文档 → **改一个字** → 保存」把全篇的 `-` 静默换成 `*`。清单与实测见 `docs/open-decisions.md` 的「原文保真的实测与收窄」 |
| Markdown 编辑体验 | `modules/markdown/` |
| 代码编辑体验 | `modules/code/` |
| **`.mindmap` 的读写** | `core/mindmap/`（纯函数、**不 import 任何内核** —— 「结构自持，不绑定具体内核」是 §7.1 的设计意图）。字段对应表见 `docs/format-spec.md` §7.1.2 |
| **`.mindmap` ⇄ 内核数据** | `modules/mindmap/adapter.ts`（ADR 0007 后果 2 要的薄适配层）。契约是「**带旧信封进去、拿新信封出来**」：顶层 `extra` 只能从上一份信封继承（内核的 `getData()` 返回写死的字段列表），节点层才是现读的 |
| **导图编辑体验** | `modules/mindmap/MindmapEditor.tsx`。两条变更通道（`operation`/`expandNode` 主通道 + `linkDiv` 防抖兜底，后者是为 `expandNodeAll` 那种**改了数据不报事件**的操作准备的）；`getValue` 在用户没动过时交回**原文** |
| **导图能不能用某个文件** | `modules/mindmap/adapter.ts` 的 `checkKernelSafety`（id 含引号 / id 重复都要拦）+ `modules/mindmap/index.ts` 的 `canHandle`。退路是**代码内核**（有 JSON 着色），不是纯文本 |
| 界面布局 | `ui/App.tsx` + `ui/styles.css`。**样式表按「属性」划所有权**（硬约束 13）：外壳只声明 `flex` / `min-height` 这类布局约束，`overflow` 一律归内核 |
| 活动栏（最左边那条图标带） | `ui/ActivityBar.tsx` + `ui/icons.tsx`（手写的十来个 path，**刻意不引图标库**）。五个入口都走命令，可用性来自 `CommandRegistry` 的 `enabled` |
| **视图三态（富文本 / 源码 / 分屏）** | `core/document/store.ts` 的 `ViewMode`（**枚举，不是两个布尔**）+ `ui/ViewSwitcher.tsx`（状态栏里的三档）。分屏的两栏在 `ui/DocumentPane.tsx`：左 = `primary.sourceView`（可编辑），右 = `primary` 加 `readOnly` |
| **块手柄（悬停命中 / 上移下移 / 转为 / 拖动落点）** | `modules/markdown/block.ts`（纯逻辑 + 几何，**悬停走 DOM 元素、落点走坐标**，两条路的分工写在文件头）+ `modules/markdown/plugin.ts` 的 `blockHoverPlugin`（**不共用浮层骨架**：触发源是鼠标移动）+ `ui/BlockHandle.tsx`（显隐合三个条件） |
| **什么时候清掉「该操作哪一块」** | `ui/BlockHandle.tsx` 里那个 `visible === false` 的 effect。**不是「菜单关了」** —— 清早了第二次打开菜单会「点了没反应」，见 ADR 0024 |
| 某个面板 | `ui/DocumentPane.tsx` / `TabBar.tsx` / `Sidebar.tsx` / `FindBar.tsx` … |
| 设置页 | `ui/SettingsDialog.tsx`。**快捷键表与版本号都不许手写**：前者由 `CommandRegistry` 派生，后者经 `vite.config.ts` 的 `define` 从 `package.json` 注入（**必须写成 `import.meta.env.X`**，见硬约束 14） |
| 文件读写、编码探测 | `src-tauri/src/fs/` |

### 内核能力怎么加（不要破坏外壳的边界）

外壳**不认识任何具体内核**。要给内核加一种能力，走这条路：

1. 加在 `core/editor/types.ts` 的 `EditorAdapter` / `EditorHandle` 上（**契约**）；
2. 各内核按需实现（不实现的要能被外壳识别为「没有这个能力」）；
3. 外壳只问契约，不问内核。

已有八例可以照着看：

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
- `EditorMountProps.onTableChange` + `EditorHandle.table` —— 表格工具条。
  `describeTable` 把「光标在哪一格、这张表几行几列、能不能删」算成纯数据，
  外壳只负责画。**注意它的生命周期与浮层不同**：`TableState` 非 null 的含义是
  「光标在表格里」，会跟着光标来去。
- `EditorMountProps.onCodeBlockChange` + `EditorHandle.codeBlock` —— 代码块语言面板。
  它的状态非 null 的含义是「**面板开着**」（点击触发），不是「光标在代码块里」。
  另外注意这里**不是纯外壳画**：代码块的行号与语言标签是 NodeView 画进文档树的，
  外壳只画那个弹出的面板。
- `EditorMountProps.onDiagramOpen` —— Mermaid 图表的全屏查看。交出来的是
  **已经画好的那份 SVG 字符串**（不是路径：图没有文件，只有此刻屏幕上的这一份）。
  与 `onImageClick` 同属「监听必须挂在内核自己造的元素上」那一类。
- `EditorMountProps.resolveImageUrl` / `onImagePaste` / `onImageClick` —— 图片的显示、
  落盘、灯箱。前两个是**异步**的（读盘 / 写盘），第三个是同步上报。
  这里有一个容易做错的分工：**内核只做它唯一能做的那部分**（从剪贴板里把字节取出来），
  「存到哪、叫什么名、绝不覆盖」全在外壳 —— 内核连自己正在编辑哪个文件都不该知道。
  灯箱为什么由内核上报「点了哪张图」而不是外壳按类名做事件委托：见下面的坑 14。
- `EditorHandle.getPosition` / `setPosition` + `EditorMountProps.onPositionChange` ——
  位置记忆。这是**最不像浮层的一例**：前六例都是「内核报事件、外壳画界面」，
  这一例是「**内核读写状态、外壳只记住**」。
  两处容易做错的地方：① `offset` 的坐标系**由内核自己定**，外壳只做往返 ——
  想在外壳里把它换算成「第几个可见字符」等于重新实现一遍 ProseMirror 的
  `textBetween`，两份实现一定会在边界上分叉（见 ADR 0021）；
  ② 滚动容器归内核所有（`.qm-editor` / `.cm-scroller` / textarea 自身），
  所以滚动**只能**由内核读写 —— 外壳自己去 `querySelector` 就是按类名认 DOM。
  另外它**不是**「离开时问一次」：那一刻编辑器可能已经销毁，
  依赖卸载时序的东西会在某次重构里安静地坏掉。
- `EditorMountProps.onBlockHandleChange` + `EditorHandle.blockHandle` —— 块手柄。
  **它与前七例的触发源都不一样**：前面几个都挂在「选区 / 文档 / storedMarks 变了」上，
  而这一例挂在**鼠标移动**上（鼠标移动完全不碰 state，那套骨架一辈子不会被唤醒），
  所以它有自己的插件 `blockHoverPlugin`，不共用 `floatingReporterPlugin`。
  三条容易做错的地方（完整版见 ADR 0024）：
  ① 悬停命中**走 DOM 元素**（`event.target` → `posAtDOM`），不走 `posAtCoords` ——
  后者依赖 `document.elementFromPoint`，而 jsdom 没有它，走坐标等于把
  「上移 / 删除 / 转为」全部命令挡在测试之外；拖动算落点**反过来必须用坐标**，
  因为那时指针下面压着的是手柄自己；
  ② **目标块由内核记，但清它的时机在外壳** —— 判据是「手柄彻底不可交互」，
  **不是「菜单关了」**（菜单关掉时指针还在手柄上，清早了第二次打开菜单就「点了没反应」）；
  ③ **按住鼠标时不许换目标块**（判据 `event.buttons !== 0`）——
  拖动期间 `mousemove` 照常发，不挡的话用户拖第 1 块、松手却搬走了第 3 块。

**反例**（不要这样做）：在外壳里写 `if (adapter.id === 'markdown')` 或
`resolveAdapterById('code')`。那等于把内核知识漏进外壳，
换内核时外壳要跟着改 —— 那正是这套抽象要避免的事。

**另一条容易忘的**：内核里任何**绕过键盘事件**去改文档的能力，
都必须自己给事务打上「这是用户发起的编辑」标记（`USER_EDIT_META`），
否则文档不会变脏、用户按 `Ctrl+S` 写回磁盘的是改动**之前**的内容。

已经踩过四次：查找替换（`setup.ts` 的 `replace` / `replaceAll`，逐个事务打标记）、
选中浮层工具条与斜杠菜单（`asUserEdit`）、**表格的每一个操作**、
**代码块改语言**。后三者的统一入口是 `withUserEdit(editor, run)` ——
它临时替换 `view.dispatch`，给这期间派发的每个事务都盖上标记，`finally` 里还原。

图片插入（落盘回来时早已过了那个窗口，所以另有一个 `withUserEditOnView`）踩过第六次。

**Mermaid 预览是唯一一个「不需要打标记」的**：它只改**选区**（点图 → 光标进块），
不改文档 —— 而「文档没变就不该变脏」正是这个标记要保证的事。
所以判断依据不是「是不是用户动作」，而是**「这次事务改了文档没有」**。
新加一条这类能力时，先问自己这一句。

---

## 四、十六个已经踩过的坑（省下你的两小时）

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

正确做法在 `src/test/typing.ts`（`typeInto`），**三个顺序不能乱**：

1. **先派发 `beforeinput`，再改 DOM。** 这是浏览器的顺序，也是内核判定
   「这次变更是用户干的」的**唯一可靠依据** —— 理由见坑 11。
2. **改完 DOM 要先让出一个宏任务，再派发 `input`。**
   `MutationObserver` 的回调是微任务，ProseMirror 靠它把变更记进队列；
   `input` 处理器只负责「立刻结算队列」。顺序反了队列是空的，
   这次输入被**完全忽略而且不报错** —— 表现为「测试里打字没反应」。
3. **之后还要等一帧。** 浮层上报是用 `requestAnimationFrame` 合并的
   （拖选时每个 mousemove 都触发一次，逐次去量会强制布局），
   `setTimeout` 走不到它。

同理，**任何断言「内核上报了什么」的用例都要先等一帧** ——
不等就会断言到上一个状态，那是假绿。

### 11. ProseMirror **不监听 `input`** —— 少一次 `beforeinput` 就静默丢字

这一条是 M5 做表格测试时发现的**真实数据丢失缺陷**，不是测试问题。

现象：**用户敲的第一个字存不进磁盘**。界面显示「已保存」，文件里却没有那个字。

根因：ProseMirror 靠 `MutationObserver` 读 DOM 变更并**自己派发事务**，
而 MutationObserver 的回调是**微任务**，会跑在任何事件派发之前。
于是 `markdownEmitPlugin` 的 `armedByUser` 在 `input` 到达时**还是 false**，
那次变更被当成「程序写入」安静丢掉。

它最坏的地方是**只错第一次**：从第二次输入起一切正常（标志已经上膛），
所以人工测试几乎不可能发现 —— 只有「打开文件 → 敲一个字 → 保存」
这条最短路径会暴露它。

修法：`plugin.ts` 的 `markdownEmitPlugin` 改为监听 **`beforeinput`**
（按规范在 DOM 修改**之前**派发，因此一定早于那次事务），`input` 保留作补充
（不是所有输入路径都发 `beforeinput`）。

> **这个缺陷在 M3/M4 时就已经存在**，只是没有任何一条测试踩到它。
> 结论：**「测试全绿」不等于「没有数据丢失」** —— 测试覆盖的是我们想到的路径。

### 12. Milkdown 的表格 schema 有三条读文档读不出来的约束

三条都是**实测**出来的，写错了不会报错，只会**静默丢内容**：

1. **最小表格是 2 行**（`table_header_row` + 至少一个 `table_row`）。
   空表头行在 schema 上**仍然合法**，但 `tableHeaderRowSchema.toMarkdown`
   遇到空内容直接 `return` —— 于是「只有表头的表格」存盘后**一行都不剩**，
   没有任何报错。所以删行的下限是 `map.height >= 3`，与光标在哪一行**无关**。
2. **对齐是按列、且由表头决定的**。`tableSchema.toMarkdown` 用**第一行**的单元格
   拼出对齐串，`keepTableAlignPlugin` 再把表头的 `alignment` 抄给同列的每一格。
   所以「给某一列设对齐」必须**写整列** —— 只写光标所在那格，
   下一次事务就被改回表头的值，用户会觉得「设了没生效」。
3. **`TableRole` 里没有 `header_row`**（只有 `table | row | cell | header_cell`）。
   表头行和正文行共用 `row`，想按「这是第几行」判断必须自己数。

完整取舍见 [ADR 0017](docs/decisions/0017-m5-tables-and-code-blocks.md)。

### 13. `setNodeAttribute` **不检查节点类型** —— 过期的位置会写到别的节点上

`tr.setNodeAttribute(pos, 'language', lang)` 只认位置，不认节点是什么。
位置一旦过期（面板开着的时候用户在别处打字、或者 `setMarkdown` 换掉了整篇文档），
它会**安静地把属性写到另一个不相干的节点上**，而且不报错。

所以 `CodeBlockChromeRequest` 里给的不是 `pos` 数字，而是
`resolvePos: () => number | null` —— **写的那一刻现问**，拿不到就放弃。
给函数就没有「记下的坐标过期」这个窗口。

> 这条是踩出来的：第一版存的是裸 `pos`，`setMarkdown` 之后设语言写到了
> 一个无关段落上，测试表现为「什么都没发生」，查起来像是面板没生效。
>
> **推广**：任何「面板 / 浮层持有文档坐标」的设计都要问一句 ——
> 这段时间里文档会不会变？会的话就存函数，不要存数字。

### 14. 外壳想按类名从 DOM 里认出内核画的东西 —— 那是耦合，不是捷径

图片的灯箱要「点一张图 → 开全屏预览」。最省事的写法是外壳在编辑器容器上挂
一个 `click`，命中 `img.qm-img-el` 就开。**不要这么做。**

那要求外壳认识**内核画的类名**，而依赖方向是
`modules/` → `core/editor/types.ts` ← `ui/`：**外壳不许 import 任何具体内核**。
把类名挪进 `core/` 只是把耦合换个地方藏起来 —— 那份类名本质上是内核的渲染细节，
它随内核改而改，而外壳会在毫无察觉的情况下跟着坏。

正确做法是**让内核上报**：契约加 `EditorMountProps.onImageClick(image)`，
监听挂在内核自己造的那个元素上。内核知道「点了哪个节点、它的 src 是什么、
解析出来的 URL 是什么」，外壳只需要画。

> **判断依据**：如果一件事需要外壳知道内核内部的 DOM 结构、类名、或者节点类型名，
> 那它就该是一条契约方法。已经按这条走过六次（选区、斜杠、表格、代码块、图片、图表），
> 每次都加在 `core/editor/types.ts` 上。
>
> **顺带一条**：`onImageClick` 给的是**已经解析好的 URL**，不是让外壳拿 `src`
> 再解析一次 —— 再解析意味着把同一个文件读第二遍，而那份字节此刻就在 DOM 上。
> 同理 `onDiagramOpen` 给的是**已经画好的那份 SVG 字符串**：图没有文件路径，
> 只有「此刻屏幕上的这一份」，重新渲染一遍既慢又可能画出不一样的图。

### 15. `nodeView.update()` 比 `PluginView.update()` **早一步** —— 在它里面判断选区会差一帧

Mermaid 图表块要在「光标进块」时显示源码、「光标离块」时显示图。
最自然的地方是在 `update()` 里判断 —— 而 `update()` 只在**节点本身**变了时被调用
（选区变化不是文档变化），所以本来就够不着。于是改用插件持一份活跃视图表，
在 `PluginView.update()` 里推给每个视图。

**坑在后面**：一个事务里 ProseMirror **先**跑 `nodeView.update()`、**后**跑
`PluginView.update()`。所以如果在 `update()` 里顺手做一次「要不要画图」的判断，
它看到的 `editing` 是**上一帧**的值 —— 表现是「在图表块里打第一个字时会多画一张
没人看的图」（第一次错，后面就对了，因为标志已经跟上）。

**修法**：图区的对齐**统一放在插件推过来的那个回调里**，`update()` 只管节点本身。
一句话：**在 `update()` 里读任何与选区有关的状态，都要先问一句「这个值是不是上一帧的」。**

### 16. `getBBox()` 在「没被渲染」的元素上给不出数字 —— 也决定了单测能证明什么

Mermaid 靠 `getBBox()` / `getComputedTextLength()` 量文字尺寸来决定节点框多大。
这两个 API 只在元素**真的参与布局**时才给得出数字：`display: none` 或者游离
（detached）节点会得到全 0（Firefox 上直接抛错），画出来是一片挤在一起、
互相重叠的框。所以给它量尺寸的容器必须挂在文档里、且用 `position: fixed`
挪到屏幕外，**不能用 `display: none` 藏**。

**同一个事实的另一半在测试里**：jsdom 没有布局引擎，`getBBox()` 一律返回 0，
**mermaid 在单测里画不出任何东西**。所以「渲染」这一层必须做成**可注入**的
（`MermaidIo` / `renderMermaid` 的注入点），单测钉住的是**我们这一层的契约**
（谁被换掉、失败留下什么、主题传对了没有、容器摘没摘），
而不是「图画得好不好看」—— 后者只能人工看。

> 判断依据：**只要一个库的渲染依赖真实的排版引擎，它就必须能被替换掉**，
> 否则那个功能在单测里等于没测。

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

## 六、现在的状态（2026-10-04）

**已完成**：P0–P3 的大部分 —— 文档生命周期、Markdown 内核、自包含导出、
多标签工作区、文件树、代码内核、命令面板、查找替换、大纲、主题。

**Markdown 完善（P3.5，对标 horseMD）**：字数统计、源码模式切换（Ctrl+/）、
**选中浮层工具条**、**斜杠菜单（`/` 插入块）**、
**表格可视化编辑 + 代码块语言选择器 / 行号**、
**图片粘贴落盘 + 相对路径 + 灯箱**、**LaTeX 数学公式**、**Mermaid 图表**、
**位置记忆 + 设置页**、**VSCode 式外壳 + 视图三态（含源码/预览分屏）**已交付；
**只剩块手柄**。
清单在 `ROADMAP.md` 的 P3.5。
**前六项都没有引入新依赖** —— `@milkdown/kit` 已经带齐大部分积木，
表格与代码块是自研的（官方那套是 Vue 3，见 ADR 0017），
图片的显示与灯箱也是自己画的（见 ADR 0018），
M10 的图标也是手写的十来个 path（`ui/icons.tsx`）。
后两项（KaTeX、Mermaid）破了这条：`katex` + `remark-math`、`mermaid`，
两者都走动态加载、不进主包（ADR 0019 / 0020）。

测试：**1,259 条前端 + 53 条 Rust**，`tsc` / `lint` / `vite build` / `cargo test` 全绿。
主包 1,892.10 kB（gzip 626.59 kB），已超 Vite 的 500 kB 警戒线 ——
归因与拆分方案在 `ROADMAP.md` 第 18 项，是当前最该做的一件事。
打包产物里另有 mermaid 的懒加载 chunk 3.4 MB（ADR 0020）与 draw.io 查看器
2.654 MiB（`public/drawio/`，按需注入、不进 bundle，ADR 0027）。

**P4 思维导图已全部完成（2026-10-04）**：`.mindmap` 的读写、适配层、内核接入、
**工作区全文检索**（第 23 项，见 ADR 0026）全部落地，四类一等公民里
**Markdown、思维导图、代码文本三类已可独立使用**。

**P5-25 内置只读 draw.io 查看器已完成（2026-10-04，ADR 0027）**：Markdown 里的
`kind: diagram` doc-block 渲染成真图，reference / inline 两种模式，四类一等公民里
draw.io 已可在 Markdown 里引用查看。P5 剩下的按需下载（26）与离线构建（27）
需要真实分发环境先行。

**最该先做的一件事**：**P3-18 主包拆分**（三个内核改成按需动态加载）——
1.89 MB 的主包是现在最大的技术债，拆分方案在 `ROADMAP.md` 第 18 项。

> **✅ 纪律已解除（2026-10-02）**：`open-decisions.md` 里的 V1–V4 全部有结论，
> 待验证清单已清空，**不再有「出结论之前不许新增待验证项」这条约束**。
> 但「**先量再改**」这条不解除 —— 见 `ROADMAP.md` 的六条方法教训。
>
> V1（draw.io 查看器）2026-10-02 已测完 —— 三条判据全过、决策维持，
> 但翻出两条新风险（**stencil 缺失静默降级** / **`stencils.min.js` 不含 `shapes/*.js`**）。
> **给 P5 的四条硬要求写在 ADR 0005 文末**，动手前先读那一段。
> V2（mind-elixir 500 节点）判据修正为 `refresh`，结论是「成本由顶层分支数决定」，
> 维持 mind-elixir（ADR 0007 复评记录）；
> V3（Milkdown 长文档输入延迟）已测完并落地（ADR 0004）；
> V4（表格可视化编辑的成本）已由 M5 回答 —— 自研可行、成本可接受。
>
> **P4 接入内核时新踩的四个坑**（都写进 ADR 0025 与 `PITFALLS.md`）：
> 注册顺序把 `code` 排在了导图前面（它认得 `.mindmap`，于是全被它吃掉）；
> 「没改过就不写」的基准取错了时机（内核跑完首次布局会给顶层节点补 `direction`）；
> jsdom 没有 `matchMedia` 而内核在构造函数里就要它；空文件的「原文」不能当回退目标。
>
> **P4 第 23 项（全文检索）又踩的一个坑**（写进 ADR 0026 决策 7）：
> **React 19 的 StrictMode 会把「模拟卸载→重挂」推迟到挂载之后的某个异步点**。
> 外壳握着句柄「戳一次」的定位刚好落在重挂之前，随后那次重挂把选区与
> 状态栏的光标读数一起销毁，而外壳早已把待办清掉 —— 再也没有人补那一次定位。
> 修法是把「跳到某一行」变成与 `initialPosition` 同构的东西：
> 请求作为 prop 传给面板，**面板在 `handleEditorReady` 里应用**，
> 任何一次内核重建都会重新落地。判断依据由此多一条：
> **凡是「等内核就绪才该落地」的意图，都走 ready 回调 + prop，不走命令式戳一下。**

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
| 引入 `@milkdown/components`（官方表格 / 代码块组件） | 它是 **Vue 3** 写的，会带进第二个 UI 框架并替换掉编辑器的高亮管线（编辑器与导出从此不同源）。理由与重审条件见 [ADR 0017](docs/decisions/0017-m5-tables-and-code-blocks.md) |
| 打开 Tauri 的 `asset:` 协议来显示本地图片 | 它要配一份**路径白名单**（scope），是一块需要长期维护的安全面。`data:` 走已有的 `read_binary_file`，权限收口在同一个地方。代价有上界（上游卡了 24MB）。重审条件见 [ADR 0018](docs/decisions/0018-m6-images.md) |
| 为「可能被否的方案」先写代码 | 先跑验证。ADR 0005 的 R1 就是这么处理的 |
