# 0017 · M5 表格与代码块：自研组件，不引入 Vue 版官方实现

状态：**已采纳**（M5）

## 背景

M5 的交付是「表格增强 + 代码块（语言选择器 / 行号）」。`ROADMAP.md` 在这一行
留了一句备注：`` `kit/component/table-block` + `code-block` `` ——
指的是 Milkdown 自带的两个成品组件。Milkdown 的官方文档把它们作为
「开箱即用的表格 / 代码块」推荐，看起来是最省事的路。

先把它们到底带来了什么查清楚（`node_modules/@milkdown/components/package.json`）：

| 项 | 内容 |
|---|---|
| 运行时依赖 | **`vue@^3.5.20`**、`@floating-ui/dom`、`clsx`、`dompurify`、`lodash-es`、`nanoid`、`unist-util-visit` |
| 额外内核插件 | `@milkdown/plugin-diff`、`@milkdown/plugin-tooltip` |
| peer 依赖 | `@codemirror/{language,state,view}`（这三条我们已经有了） |
| 实现方式 | Vue 3 组件。`code-block` 的 `NodeView` 里 `createApp(...)` 挂一个 Vue 应用，再把一个 **CodeMirror 6 实例**嵌进 ProseMirror 的文档树 |

也就是说，采纳它的代价是**在一个 React 19 工程里引入一整套 Vue 运行时**，
外加五个通用工具库和一个第二编辑器实例。这不是「顺手用个官方组件」的量级。

## 决策

**M5 的两半都自研。表格用「浮层工具条 + ProseMirror 命令」，代码块用
「NodeView + 外壳面板」。不引入 `@milkdown/components`。**

这与 M1–M4 是同一条线：**四批功能都没有引入任何新依赖**，
而 M5 继续做到了这一点 —— 代码块用到的 `@codemirror/*` 早就因为代码内核而存在，
行号与语言标签是我们自己画的 DOM。

## 理由

### 1. 为一个代码块引入 Vue 不划算

官方 `code-block` 提供的语言选择器确实不错（可搜索的候选列表）。
但我们已经有：

- `core/highlight/hljs.ts` —— 语言清单（`hljs.listLanguages()`，36 种）与高亮实现，
  **编辑器与导出共用同一份**；
- `core/commands/fuzzy.ts` —— 模糊搜索（M4 为斜杠菜单写的）；
- `ui/floating.ts` —— 浮层摆放算术。**这个文件是本次才提取出来的**：
  M3 的 `SelectionToolbar` 与 M4 的 `SlashMenu` 各自内联了一份几乎相同的
  clamp 逻辑（「上方放不下翻到下方」「水平夹进视口」），到第三次要写语言面板时
  才按「三次法则」提取成纯函数，并补了 12 条单测。
  提取的收益不只是少抄一遍 —— 它让「摆在哪」从**副作用里的算术**
  变成**可断言的输入输出**，而后者在 jsdom 里能测（`getBoundingClientRect()`
  仍然全是 0，但纯函数的输入是我们自己造的）。

把这几样拼起来就是一个语言选择器。为了它引入 Vue，等于用 35 KB（gzip）
的第二个 UI 框架换掉大约 150 行我们自己的代码。

> 提取时顺手给 `computeFloatingPlacement` 加了第三个水平模式 **`end`**
> （右对齐）。原来的两种是 `center`（工具条跟着选区居中）与 `start`（菜单左对齐光标），
> 而语言面板要**右对齐到语言标签的右边缘** —— 标签在代码块右上角，
> `start` 对齐会让面板从标签开始向右长，长出正文栏外面。
> 新增分支的同时补了 2 条单测（右对齐的算式、贴左边缘时的夹取）。

### 2. 官方实现会**替换掉**我们的高亮管线

`code-block` 组件把代码本体交给 CodeMirror 渲染，也就是说语法着色从
「`highlightCode` → inline 装饰」变成「CM6 + Lezer」。

后果不是「多了一个依赖」，而是**编辑器与导出不再同源**：
导出走的是 `core/export/inline-html.ts` → `highlightCode` → hljs 的 class，
编辑器会走 CM6 的 `tags` → 另一套 class 名。两边的配色要么对不上，
要么得再维护一张映射表。而「编辑器里是彩色、导出的文件里是黑白」
正是这份产品明令要避免的（`core/highlight/hljs.ts` 文件头第一条）。

### 3. 官方实现会带进我们**明确拒绝过**的插件

`@milkdown/components` 依赖 `@milkdown/plugin-tooltip`（0015 里被否掉的
「内核持有用户界面」那一路）与 `plugin-diff`。`@milkdown/kit` 那一侧还挂着
`plugin-slash`（我们已经在 M4 自研了斜杠菜单）与 `plugin-listener`
（它的 `markdownUpdated` 防抖 200ms，会让「这次变更是程序写入的」判断失效，
见 `plugin.ts` 的 `markdownEmitPlugin`）。

装上一个成品组件，就要为它带来的这一串邻居一起负责。

## 表格半边：三条被实测逼出来的结论

表格的全部命令面在 `modules/markdown/setup.ts` 的 `createMarkdownTable`，
三条路各走各的：

### 1. Milkdown 的 gfm 表格 schema 是 `table_header_row table_row+`

**必须有表头行，且至少一行正文。** 于是：

- 表格最少 2 行 ——「只剩 1 行」不是一种可表示的状态，删到 2 行就该停手
  （`describeTable` 的 `canDeleteRow` 因此是 `map.height >= 3`，
  与光标在哪一行**无关**）；
- 表头行不能凭空消失 —— 删掉它就得让下一行顶上。

这条约束的坑在于它**不在 `deleteRow` 的视野里**：`prosemirror-tables` 的
`deleteRow` 删掉表头行之后，剩下的空表头行在 schema 上**仍然合法**
（`tableHeaderRowSchema` 的 content 是 `(table_header)*`），
但它的 `toMarkdown` 遇到空内容会直接 `return`。实测：文档里 4 行、
`getMarkdown()` 只剩 2 行，**没有任何报错**，用户也看不出少了哪两行。

所以删行改成**重建表格节点**（`deleteTableRow`）：丢掉目标行，
必要时把第 1 行提升为表头行。

### 2. 删列**不能**重建

`deleteColumn` 会按 `TableMap` 正确地收缩 `colspan`，而重建做不到这一点
（`colspan` 是跨列的，删一列要把它从 2 改成 1，而不是删掉整个格子）。
所以删列继续走 `prosemirror-tables`，只是要额外修光标：
`deleteColumn` 在删掉光标所在的那一列后会把选区映射到**表格之外**，
于是工具条自己消失了 —— 表现为「点了一下，按钮不见了」。

修法是 `takeTransaction`（把命令要派发的事务截下来）+ `keepCaretInTable`
（先把光标夹回新表格内再派发）。

### 3. 对齐在 Markdown 里本质是**按列**的

`tableSchema.toMarkdown` 是用**第一行**的单元格决定分隔行的：
`align.push(cell.attrs.alignment)`。而 `keepTableAlignPlugin` 的
`appendTransaction` 会把表头单元格的对齐**抄到同列的每个单元格**上。

于是预设的 `setAlignCommand` 在正文单元格里看着像坏了 ——
用户只改了光标所在那格，下一次事务就被改回表头行的值。

所以 `setAlign` 自己写：**整列一起改**（`writeColumnAlign`）。

顺带补了一处预设的不对称：加行会抄相邻单元格的对齐
（`addRowWithAlignment`），**加列不会**。不补的话，在一张整列居中的表格里
「加一列」会得到一列左对齐的格子。`createMarkdownTable` 的 `colBefore` / `colAfter`
因此要在插入前记下参考列的对齐，插入后再写上去。

## 代码块半边：NodeView + 外壳面板

### 为什么这里必须用 NodeView（表格那边用的是装饰）

表格工具条是**跟着光标**的浮层。代码块的两样东西都不是这个形态：

- **行号要和代码的每一行对齐。** 代码块内部是一个带 `\n` 的文本节点，
  DOM 里没有「一行一个元素」这种结构，只能自己造。装饰做不到 ——
  `Decoration.inline` 造不出块边界，而且空行（`from === to`）
  根本没法被 inline 装饰命中，行号会从空行那里开始错位。
- **语言标签要常驻在每个代码块的右上角**（Typora 就是这样），
  与光标在不在里面无关。做成浮层就得为每个块跟踪位置、滚动时逐个重算。

两件事都要求「块的内部有我们自己的 DOM」，那就是 NodeView 的定义。

### 三个不能省的 NodeView 细节

1. **`ignoreMutation` 必须声明。** 行号与标签是我们自己往 DOM 里写的，
   每写一个字都会触发 `MutationObserver`。不声明忽略的话，ProseMirror
   会把行号当成用户输入**读进文档** —— 存出来的 Markdown 里凭空多出一列数字。
   判据只有一条：不在 `contentDOM` 里的都不算。

2. **`update` 里要判「值真的变了没有」。** 这个方法在**每一次文档变更**时都会被调用。
   不判就重画的话，在一个一千行的代码块里敲一个字就要新建一千个行号元素。
   加上判断之后，在代码块里打字产生的 DOM 变更是**零**。

3. **面板不该记文档坐标。** `CodeBlockChromeRequest` 里给的是
   `resolvePos: () => number | null`，不是一个 `pos` 数字。
   面板开着的时候用户完全可能在别处打字，记下的数字就指向了别的节点，
   而 `setNodeAttribute` **不检查节点类型** —— 它会安静地把语言写到
   那个不相干的节点上。给函数就没有这个窗口：写的那一刻现问。

### 语言选择器是**点击触发**，不是「光标进去就出现」

语言标签常驻在每个代码块上，既然它一直在，触发就是「用户点了它」。
所以 `CodeBlockState` 非 null 的含义是「面板开着」，
与 `TableState`（跟着光标走）的生命周期不同。

### 面板上还有一个视图偏好：行号

行号是**纯展示**，所以：

- 内核只读这个偏好来**选一个类名**（`qm-no-line-numbers`），
  行号槽本身照建不误、由 CSS 负责显隐；
- 换来的好处是**改这个偏好不需要重建内核** —— 重建会丢掉光标、
  滚动位置与撤销栈，而用户只是在面板上勾了一个框。

### 一个真实缺陷：`beforeinput`

做表格测试时发现**用户敲的第一个字存不进磁盘**：界面显示「已保存」，
文件里却没有那个字。

根因是 ProseMirror **并不监听 `input` 事件** —— 它靠 `MutationObserver`
读 DOM 变更、自己派发事务，而 MutationObserver 的回调是微任务，
先于任何事件派发跑完。于是 `markdownEmitPlugin` 的 `armedByUser` 在
`input` 到达时还是 false，那次输入被安静丢掉。而且它是**攒着的**：
从第二次输入开始一切正常 —— 这种「只错第一次」的 bug 极难在人工测试里发现。

修法是补上 `beforeinput`（按规范在 DOM 修改**之前**派发，所以一定早于那次事务），
并让 `src/test/typing.ts` 也按这个顺序派发。
`input` 保留着作为补充 —— 不是所有输入路径都发 `beforeinput`。

**这个缺陷在 M3/M4 就已经存在**（`markdownEmitPlugin` 是那时写的），
只是没有任何一条测试踩到它。

## 代价

### 1. 行号槽的 DOM 在关掉行号后仍然存在

关掉只是 CSS 不画。每个行号是一个空 `div`，几千行的代码块上仍是可忽略的开销，
但它是**真实存在**的。

### 2. 语言候选列表的显示名要我们自己维护

`supportedLanguages()` 给的是 `hljs` 的语言标识，中文名与拼音别名在
`ui/CodeBlockPicker.tsx` 里。**只补常见的十几种**，其余照原样显示标识 ——
给 `gams` 编一个中文名只会让它更难被找到。

### 3. 代码块的编辑器 DOM 与导出 DOM 不是同一棵树

编辑器是 `<div class="qm-cb"><div class="qm-cb-gutter">…</div><pre><code>…</code></pre><button>js</button></div>`，
导出是 `<pre data-language="js"><code>…</code></pre>`。
**观感一致靠的是样式抄同一组值**（`markdown.css` 里那些注释标了对应关系），
不是靠结构相同。这意味着改 `document-styles.css` 里代码块的排版时，
两边都要看一眼。

### 4. 浮层几何仍然只有人眼能验

同 0015 / 0016。`computeFloatingPlacement` 的算术有单测，
但「内核量出来的坐标对不对」在 jsdom 里测不了
（`getBoundingClientRect()` 全是 0）。

## 重审条件

1. **Milkdown 的官方组件改成框架无关**（比如提供 headless / Web Component 版本）
   → 重新评估。当前拒绝的唯一理由是 Vue 运行时，不是它的功能不好。
2. **代码块要支持「实时预览」**（M8 的 Mermaid 就是这种：
   围栏里写的不是代码而是要渲染的图）→ 那时 `renderPreview` 那套会变得有吸引力。
   届时的判断依据是：**能不能只取它的渲染管线、不取它的 Vue 外壳**。
   取不了就继续自研（`previewOnlyByDefault` 那部分逻辑只有几十行）。
3. **行号槽的 DOM 开销被实测证明影响性能** → 把偏好挪进内核、
   由 NodeView 决定建不建，并强制重绘（代价：改偏好要重建内核，
   光标与撤销栈会丢）。
4. **换掉 Milkdown** → 表格的契约只有 `TableState` / `EditorTable`，
   代码块的只有 `CodeBlockState` / `EditorCodeBlock`，都是外壳自己的词。
   换内核时外壳一行都不用改。**做不到就说明这次抽象失败了**，
   要在这里如实记下来。
