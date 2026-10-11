# 0040 · 链接激活：编辑器里绝不放行导航，五类去向都给反馈

日期：2026-10-10
状态：已采纳
关联：0003（不静默，是本 ADR 的母题）、0014（源码视图复用代码内核）、
0018（图片：`onImageClick` 那一类「监听挂在内核自己造的元素上」）、
0021（位置记忆：位置由内核算，外壳只做往返）、
0031（i18n：文案只有一份 key）、
0034（面板常驻：句柄按面板分桶 —— 「点击那一刻的句柄」必须是活动面板那一个）、
AI助手手册硬约束 1（依赖方向单向）/ 2（绝不静默）/
11（外壳不许按类名认内核画的东西）/ 12（`joinPath` 只统一分隔符）

## 背景

Markdown 文档里的链接一直是**能看不能点**的：渲染出一个 `<a>`，
点下去什么事也没有。这不是「少一个功能」，而是踩了硬约束 2 ——
一个用户看得见、点得着、点下去没有任何反馈的东西，比没有它还糟：
用户没法区分「没生效」「我不支持」「点偏了」这三种情况。

顺带还有一条更硬的事实：**不拦住的话它会导航走。**
编辑器跑在 WebView2 里，`<a>` 的默认行为是让整个 webview 跳走 ——
正在编辑的文档连同未保存的改动一起消失，
而保存前的确认逻辑（`DocumentStore` 那条保护）根本不会被问到。

所以这件事有两半，缺一半就是 bug：① 掐掉默认导航；② 五种去向都给出话。

它确实值得一条 ADR，因为其中至少六处有第二个选项，都被否掉了
（见每个决策里的「否掉的选项」）。

## 决策

### 1. 内核上报，外壳决定 —— 走契约 `EditorMountProps.onLinkClick`

点击链接是**文档内的就地动作**：与「点一张图开灯箱」（ADR 0018）同一类。
按那条纪律办：

- 契约加在 `src/core/editor/types.ts`：`LinkActivation { href, text }` +
  `EditorMountProps.onLinkClick?`，监听挂在**内核自己造的那个 `<a>` 上**；
- 外壳（`src/ui/App.tsx` 的 `openLink`）负责「相对谁」——
  只有外壳知道这个文档此刻在哪个目录；
- 外壳不许按 `a[href]` 做事件委托（硬约束 11 的同胞：
  那要求外壳认识内核画的 DOM，依赖方向就是这么漏掉的）。

### 2. 读**文档模型**里的 `href`，不读 DOM 上的

`@milkdown/preset-commonmark` 内部有一个 `sanitizeLinkHref`：不安全协议
（`javascript:` / `data:` …）写进 DOM 时被替换成空串。也就是说
**DOM 上的 `href` 会说谎** —— 用户写的 `javascript:alert(1)` 在 DOM 上是 `''`。

所以 `linkMarkAt` 从 `view.state.doc` 的 link mark 上取 `attrs.href`，
而不是 `anchor.getAttribute('href')`：否则那条地址的真相在到达数据模型之前
就被白名单蒸掉了，外壳只能拿到一个空串，说不出「这个链接用的是 javascript 协议」。

### 3. 挂 `handleDOMEvents.click`，不挂 `handleClickOn`

`handleClickOn` 由 ProseMirror 的 **mouseup** 处理器调用，
它那里的 `event.preventDefault()` **取消不掉随后的 `click`** ——
webview 的导航照样发生。
这是实测出来的，所以 `src/modules/markdown/link.test.ts` 的文件头
把「preventDefault 无条件发生」钉成三条不变量里的第一条。

### 4. `preventDefault` 是**无条件**的，打开才需要修饰键

```ts
if (!(event instanceof MouseEvent)) return false;
const anchor = anchorElementOf(view, event.target);
if (anchor === null) return false;
event.preventDefault();                   // ← 无条件
if (!isActivation(event)) return false;   // ← 之后才问修饰键
```

两件事被刻意分成两步：

- **导航无条件掐掉**：编辑器里没有任何链接该让 WebView 导航走。
  哪怕是「不支持的协议」「空地址」，留给浏览器的默认行为都是错的 ——
  它们要的是**一句话**，不是一个新的页面。
- **打开需要 Ctrl / Cmd**：这是富文本编辑器的惯例
  （Typora / VSCode / Notion 都是），而且它解决了一个真实存在的冲突 ——
  用户想把光标放到链接**里面**去改字。

代价是第一次可能不知道要按 Ctrl —— 这是**可发现性**问题，
不是要不要支持的问题：修它的正确方式是加一句 hover 提示（见重审条件），
而不是取消修饰键。

**否掉的选项是「不用修饰键，把链接整体做成只读的 NodeView」。**
那等于让用户想改 `[文字](url)` 里的那几个字时先切到源码视图 ——
用一个更贵的功能（源码视图）去救一个便宜的动作（把光标挪进去）。

### 5. 地址分类是 `core/` 里的一个**纯函数**

`src/core/link.ts` 的 `classifyLink(href)` → 五类：
`anchor` / `file` / `external` / `empty` / `unsupported`。

放在 `core/` 是因为判定只依赖 `href` 这个字符串，**不需要知道文档在哪**
（「相对谁」那一步归外壳）。于是它能配一大批单测：`src/core/link.test.ts` 30 条。

三条顺序不是随便写的，写反了就是 bug：

1. **盘符先于协议**（`DRIVE_ABSOLUTE = /^[A-Za-z]:[\\/]/` 早于 `SCHEME`）——
   `C:` 与 `https:` 长得一模一样。先按协议判会让用户写的绝对路径链接
   得到一句「协议不受支持」，即**把一个可用的东西说成不可用**。
2. **先拆锚点再判其余** —— 否则 `other.md#sec` 会被当成一个含 `#` 的文件名。
3. **空路径 + 有锚点 = 本文档锚点** —— `#标题` 只有这一种写法。

另一条与安全有关：**探测协议要走 `stripIgnoredChars`**
（`code <= 0x20` / `0x7f–0xa0` / `0x200b–0x200d` / `2028` / `2029` / `feff`
一律不算数）。浏览器就是这么解析的，`java\tscript:` 在真实浏览器里
仍然解析成 `javascript:` —— 我们的判定必须与浏览器一致，
否则「我说能打开」与「浏览器怎么理解它」会分叉。
归一化**只用于探测**，交出去的永远是原文（单测里有两条钉住这件事）。

还有一份刻意**不共用**的名单：`EXTERNAL_SCHEMES = http / https / mailto`，
比 Milkdown 的渲染白名单（它还带 `ftp` / `tel`）**更窄**。
宽进严出：渲染时认得的协议照常显示，我们主动交出去的只有这三个。
（`src/core/link.ts` 的文件头写了为什么两份名单不合并。）

### 6. 外壳决定「文件用谁开」：注册表说了算

```ts
hasSpecializedAdapter(absolute) ? workspace.open(absolute) : openExternal(absolute)
```

轻笺自己认得的那几类（`.md` / `.mindmap` / 代码后缀）**在工作区里打开** ——
它们是一等公民，交给系统默认程序只会得到另一个编辑器里的源码；
其余（png / pdf / xlsx …）交给系统的关联程序 ——
在纯文本内核里显示一张 png 的字节不是「打开」。

判据问的是**注册表**（`hasSpecializedAdapter`）而不是某个后缀清单：
前者是我们真正的答案。它异步填充，所以启动极早期的一次点击会落到
「交给系统」那一支；那时内核自己也还没就绪，可接受。

外部调用藏在 `core/platform/desktop.ts` 后面（新增 Rust 依赖
`tauri-plugin-opener`，见「后果」）：
`core/platform/index.ts` 的 `openExternal` 在**非桌面**环境走
**明确 reject**（不是静默返回），所以浏览器预览下点击会走到「说一句打不开」那一支。

### 7. 锚点：新增可选契约 `EditorHandle.goToAnchor(anchor): boolean`

与既有的 `goToHeading` 只差「那个位置怎么算出来」。
`findAnchorPos` 刻意**不限定为标题**：判别依据是「有没有这个 `id`」本身，
与浏览器 hash 跳转的口径一致。写成「只看 heading」等于替明天的 schema 做决定
（哪天代码块也带上了 id，那份判定就得跟着改一遍）。

返回**布尔**而不是让外壳去 catch：内核不支持锚点（比如源码视图）时外壳收到
`false`，说的是「这份文档里没有叫 X 的锚点」，**不会**去开一个新标签页 ——
用户点的是「跳到这一节」，不是「打开另一个文件」。

> **这里揪出来过一个真实缺陷**：`findAnchorPos('')` 会命中那些 `id` 默认是空串的
> 节点（一条还没被 `syncHeadingIdPlugin` 写到 id 的空标题），
> 于是 `[占位](#)` 这类「先占个位」的写法会**跳到一个无关的空标题上去**，
> 还假装成功了。修法是第一行 `if (anchor === '') return null;` ——
> 空串是「没写锚点」，不是「锚点指向这里」，
> 它由外壳判成 `empty` 并说一句话。这是测出来的，不是读代码读出来的。

### 8. 五个分支**都**终止在一句话上 —— 除了该安静的那两个

| 去向 | 用户看到 |
|---|---|
| 本文档锚点（命中） | 光标真的过去了 —— **一句不说** |
| 文件（成功） | 标签页真的打开了 —— **一句不说** |
| 锚点不存在 | 「这份文档里没有名为「X」的锚点。」—— 点名它，不说「跳不过去」 |
| 空地址（`[占位](#)`） | 「这个链接还没有指向任何地方（地址是空的）。」 |
| 不支持的协议 | 「这个链接用的是 javascript 协议，轻笺无法打开。它仍然可以编辑。」—— 点名**它是什么** |
| 相对地址没有基准 | 「无法确定 ./x.md 的位置 —— 这份文档没有可用的目录信息。」 |
| 打开失败（两种走法都是） | 「打不开 X。」 |

**「空」与「不支持」分成两种**：成因完全不同 ——
前者是作者还没写完，后者是作者写了我们处理不了的东西，
混在一起说只能给出一句两头都不准的废话。

**成功的两个分支刻意安静**（`App.test.tsx` 里钉了通知为空）：
一边干活一边表功，是另一种吵。

## 后果

- **两个新模块**：`src/core/link.ts`（纯函数 + 30 条单测）、
  `src/modules/markdown/link.ts`（插件 + `findAnchorPos` + 14 条 jsdom 单测）。
  外壳侧新增 `src/ui/App.test.tsx` 的「链接激活」一组 8 条
  （五个分支 + 无修饰键的普通点击 + `defaultPrevented` + 成功的分支不许说话）。
- **契约变了三处**：`LinkActivation`、`EditorMountProps.onLinkClick?`、
  `EditorHandle.goToAnchor?`。后两者都是**可选**的 ——
  没有这个能力的内核（源码视图 / 代码内核 / 导图）按「没有锚点」处理，
  而不是画一个点了没反应的东西。
- **一个新 Rust 依赖**：`tauri-plugin-opener`（2.x，`Cargo.toml` 里写了理由与
  重审条件），权限是两条 allow（`opener:allow-open-url` / `allow-open-path`）。
  前端侧不直接 import `@tauri-apps/*`（硬约束 1），只走 `core/platform/`，
  那里是**动态 import**（不进主包的关键路径）。
- **「点击那一刻的句柄」必须是活动面板那一个。** 它依赖既有的一条纪律：
  面板常驻（ADR 0034）之后句柄按面板分桶，`openLink` 读 `handleRef.current`，
  所以它**不能进 `useCallback` 的依赖表**（它每次都在变，进去就等于每次切标签
  都重造回调，而面板是按引用收它的）。`App.tsx` 那段注释里写明了这件事。
- **已知边界**：目前只有 Markdown 富文本视图有 `<a>`。
  源码视图里的链接是语法高亮出来的 token，点它既不被掐也不被处理 ——
  那是「这份视图里没有链接可点」，不需要补偿，也不需要说一句话。

## 重审条件

- **修饰键是不是太隐蔽**：若有反馈表明用户找不到入口（Issue、冒烟清单里
  反复被卡），加一处 hover 提示（状态栏一句「Ctrl+点击打开」）——
  **不要**改成不用修饰键：理由见决策 4，那会把链接变成光标到不了的禁区。
- **`EXTERNAL_SCHEMES` 要加第四个协议之前**：先回答「我们要把它交给谁」
  （系统浏览器？邮件客户端？），并确认 Milkdown 的渲染白名单不会因此产生
  「显示得出来、我们打不开」的分叉。两份名单刻意不共用，改时要一起看。
- **第三个「文档里指向别处的地址」出现时**：图片（相对路径）、
  doc-block 引用（`resolveDocBlockSrc`）、链接（本次）三处都在解同一个问题，
  而且各解一遍。第三处之后再新增，就该先抽一层「相对谁」的公共解法，
  而不是再写一份 `resolveRelative` 的调用。
- **`file` 分支丢掉了 anchor**：`classifyLink` 已经把 anchor 解析出来了，
  外壳目前没用。要支持「打开别的文件并跳到它某一节」时，
  这条才需要补（届时开花的事物是 `WorkspaceStore` 一侧：跳哪儿由谁记录）。
- **是否需要在 webview 那一层拦（导航钩子）而不是在 DOM 的 click 上拦**：
  若将来发现某个写法仍能爬出去（`target="_blank"` 之类），
  那条防线要升到壳层；在那之前，只在事件发生处拦是最便宜的一层。
