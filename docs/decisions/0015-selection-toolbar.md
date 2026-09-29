# 0015 · 选中浮层工具条：内核只上报「选区状态」，浮层由外壳渲染

状态：**已采纳**（M3）

## 背景

M2 做完源码模式之后，Markdown 内核在「写」这一侧还剩一个最大的缺口：
**没有任何办法把一段文字变成加粗**。用户要么自己敲 `**`，要么去记快捷键。
Typora 的答案是一个选中文字后浮出来的工具条（bubble menu）。

Milkdown 生态里有现成的实现：`@milkdown/plugin-tooltip` 提供 `tooltipFactory` +
`TooltipProvider`，`@milkdown/components` 里也有 `link-tooltip`。用它似乎是「零成本」。

问题在于它**返回的是一段由内核用原生 DOM 拼出来的元素**。Milkdown 官方的用法是
在插件里 `document.createElement('div')`、填 `innerHTML`、自己绑事件，再交给
`TooltipProvider` 去摆位置。这条路会把两件事同时破坏：

1. **样式不再单一来源**。主题色板与深色模式变量全在 `ui/styles.css`（`:root` 与
   `[data-theme='dark']`），内核里另起一套必然对不齐 —— 而「导出的 HTML 在别的机器上
   观感一致」是 0001 定下的硬要求，编辑器里的观感也不能例外。
2. **内核开始持有用户界面**。本项目的红线是 `模块 → 契约 ← 外壳`：
   内核不认识外壳、外壳不认识内核。一个内核里的 `innerHTML` 就是这条红线上的第一个缺口，
   而缺口一旦开了，下一个功能就会照着它写。

## 决策

**内核只上报「选区现在是什么格式」和「它在屏幕上的哪儿」，浮层由外壳用 React 画。**

具体分成六条：

1. **契约加两个东西**（`core/editor/types.ts`）：
   - `EditorMountProps.onSelectionChange?: (snapshot: SelectionSnapshot | null) => void`
   - `EditorHandle.formatting?: EditorFormatting`

   `SelectionSnapshot` = `{ format: SelectionFormat; anchor: SelectionRect }`，
   其中 `SelectionFormat` 是 `{ inline: ReadonlySet<InlineFormat>; block: BlockFormat;
   headingLevel: number | null; linkHref: string | null }`。

2. **`formatting` 是可选能力，缺了就不显示浮层**（不是禁用）。
   与 `sourceView` 的处理**故意相反**：源码视图是「有，但这份文档用不上」，
   所以显示成禁用；格式化是「这个内核根本没有」，一个纯文本编辑器弹出加粗按钮，
   用户点下去只会什么都没发生。

3. **`anchor` 用视口坐标**（与 `getBoundingClientRect()` 同一套），浮层用
   `position: fixed` 摆。内核能拿到的只有 `coordsAtPos()`，它给的就是视口坐标；
   要换算成「相对编辑器容器」的坐标，内核就得知道容器是谁、滚到哪儿了 ——
   那是外壳的布局知识。

4. **浮层是 `.editor-area` 的兄弟节点，不是内核挂载容器的子节点。**
   内核挂载时会替换掉自己那层容器的内容，放进去会被抹掉。

5. **格式名在外壳与内核之间翻译一次**（`modules/markdown/selection.ts` 的映射表）。
   外壳只认 `strong / emphasis / strikethrough / inlineCode / link` 这五个词，
   不认 `strike_through`、`inlineCode` 这些 ProseMirror / Milkdown 的名字。

6. **格式化事务必须打 `USER_EDIT_META`**（见下面「代价」一节，这是本 ADR 最要紧的一条）。

## 理由

### 为什么 `formatting` 要放在 `EditorHandle` 上，而不是加一堆命令

`core/commands/` 里已经有一套命令注册表（0012）。看上去「加粗」应该是一个命令，
这样命令面板、快捷键、工具栏按钮都能共用。

但**命令的 `run(ctx)` 拿不到编辑器命令面** —— `ctx.ui.editor()` 给的是
「当前活动文档的 `EditorHandle`」，而一次格式化必须作用在**那个具体内核**上。
把它做成命令，就要在 `CommandContext` 里再开一条通向活动内核的路，
而那条路已经有 `ui.editor()` 了。所以选择把能力挂在 `EditorHandle` 上，
由浮层（它本来就只对当前活动文档有意义）直接调用。

**顺带一个好处**：浮层不需要知道「现在活动的是第几个标签页」。
它拿到的是 `DocumentPane` 给它的那份文档的 `formatting`，切标签页就换一份。

### 为什么 `inline` 用「选区里每个字符都带」而不是「起点带」

选中 `普通**加粗**` 时，起点带 strong、终点不带。如果按起点算，加粗按钮会亮起来，
而用户点下去不是「取消加粗」而是「整段加粗」—— 按钮的**承诺**和它的**行为**不一致。

ProseMirror 的 `ResolvedPos.marksAcross($to)` 给的正是「整个选区共有」的 mark，
语义天然对齐，不需要自己遍历文本节点求交集（那会把跨节点边界写错）。

### 为什么 `linkHref` 要单独上报

链接除了「有没有」还有「指向哪」。只上报「有链接」的话，用户想把 `http://` 改成
`https://` 就只能先删掉再重加。上报地址之后，浮层可以回填、内核可以走
`updateLinkCommand` 改地址 —— 而不是用 `toggleLinkCommand`（它的底层是
`toggleMark`，看到选区已经带着 link mark 会执行**移除**，用户的链接会莫名消失）。

## 代价

### 1. 格式化不打 `USER_EDIT_META` 会**静默丢数据**（已修，但值得单独记）

`markdownEmitPlugin` 只在「刚发生过用户输入」的窗口内上报变更，而点浮层按钮
**不产生任何 `input` / `paste` 事件**。第一版实现漏了这一步，后果是：

> 用户选中一段话、点「加粗」、按 Ctrl+S —— 界面全程显示「已保存」，
> 而写回磁盘的是**加粗之前**的文本。

这与 `USER_EDIT_META` 当初为「查找替换」加上的理由是**同一条**
（见 `plugin.ts` 里那段注释），但两处的实现方式不同：查找替换自己 dispatch，
可以直接 `setMeta`；格式化的事务由 Milkdown 的 `CommandManager` 内部派发
（`command(view.state, view.dispatch, view)`），中间没有钩子，也没有用到
`dispatchTransaction` 选项。所以 `setup.ts` 里临时换掉 `view.dispatch`、
在 `finally` 里换回来，作用范围只有这一次同步调用。

**这条之所以写进 ADR 而不是只写在代码注释里**：它是「能力挂在 `EditorHandle` 上」
这个决策的直接后果 —— 内核多了一条「绕过键盘事件改文档」的路径，
而每一条这样的路径都必须自己声明身份。将来再加同类能力（斜杠菜单 M4、
表格操作 M5、图片插入 M6）时会重复遇到同一个坑。

### 2. 光标位置与滚动位置不保留

切视图、点格式化都会走一次内核事务。`setBlockType` 会把光标移到新块的合适位置，
但**不做精确的坐标平移** —— 用户点「二级标题」之后光标可能跳到块首。
与 0014 记的「切视图不保留光标与滚动」是同一类问题，修法也一样（要做坐标翻译）。

### 3. 点到编辑器之外，浮层不会自己收起来

内核只在**选区真的变了**时上报。用户选中文字后去点侧栏或状态栏，
ProseMirror 的选区没变（只是失焦），所以浮层留在原地。

**这是刻意的**：用户点浮层上的按钮必然先让编辑器失焦，一失焦就藏，
按钮会在 `click` 到达之前消失 —— 那是浮层菜单最经典的坑。
链接输入框更是必须能拿到焦点。

代价是「点了别处浮层还在」。它不会造成数据问题，但确实是个毛刺。

### 4. 只用真实 WebView 才能验证的几何

`anchorOf` 依赖 `coordsAtPos()`，它在 jsdom 里没有布局引擎、拿不到真实坐标。
所以测试只能钉住「空选区返回 null」「不抛错」这两条，
**浮层到底摆在哪、会不会压住文字、贴边时有没有翻到下方 —— 只能靠人眼看**。
这一条与 V3（长文档输入延迟）属于同一类：测试环境的结论不能替代真机结论。

## 重审条件

1. **浮层出现时用户抱怨「位置不对 / 压住文字 / 贴边跑出屏幕」** →
   说明 `useLayoutEffect` 里那套「上方放不下就翻到下方 + 水平夹进视口」不够用，
   需要改成测量后按四象限择位。
2. **点到编辑器之外浮层不收起来被反复反馈** → 改成监听 `.editor-area` 的 `blur`，
   并在 `relatedTarget` 落在浮层内部时放过（链接输入框就在里面）。
   现在不做，是因为这个判断在 jsdom 里没法验证，容易写成「在真机上偶发点不动」。
3. **M4（斜杠菜单）或 M5（表格）也需要一个浮层** → 先看它们能不能复用
   `SelectionSnapshot` 这套「内核报状态、外壳画 UI」的模型。
   如果能，就把 `SelectionToolbar` 里的定位逻辑抽成共用的 `FloatingLayer`；
   如果不能，说明这个抽象选错了，要在这里记一笔。
4. **换掉 Milkdown**（0004 / V3 的结论）→ 契约里只有 `InlineFormat` /
   `BlockFormat` 这些**外壳自己的词**，换内核只需要重写 `selection.ts` 的映射表
   与 `setup.ts` 的命令派发，外壳与浮层一行都不用改。
   **如果做不到这一点，说明这次抽象失败了**，要在这里如实记下来。
