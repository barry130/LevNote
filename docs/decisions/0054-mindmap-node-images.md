# 0054 — 导图节点插图：`image` 升格为正字段，粘贴进图与占位如实

- 状态：已采纳
- 日期：2026-10-12
- 关联：[0025](./0025-mindmap-kernel-integration.md)、[0048](./0048-mindmap-and-drawio-are-first-class-too.md)、[0049](./0049-mindmap-ui-follows-the-language.md)、[0050](./0050-mindmap-exports-what-it-draws.md)、[0053](./0053-mindmap-markers.md)
- 承接：0050–0053 四轮自派需求补完了导图的「导出图片 / 大纲 / 标记」三块；本轮补第四块 —— 对标 XMind 第一屏的「插入图片」。

## 背景

对比主流思维导图工具，XMind 主题上可以挂一张图（截图、照片、示意图），
这是与标记并列的第一屏能力：有些东西文字说不清，一张图就是那个分支本身。
LevNote 的导图节点此前**零图片能力** —— 而 Markdown 内核早就有一整套
图片机制（粘贴落盘、相对路径引用、`data:` 内嵌、丢图占位）。
本轮把这套现成的机制接到导图上，而不是给导图另造一套。

## 判据（决策依赖的事实）

1. **内核本来就认 `image`**：`NodeObj.image?: { url, width, height, fit? }`
   （mind-elixir `dist/types/types/index.d.ts:185-190`），渲染由内核自己做：
   `Se` 在主题文本前画 `<img src={imageProxy ? imageProxy(url) : url}>`
   （`MindElixir.js:353-359`），`url` / `width` / `height` **三者齐才画**。
   `imageProxy?: (url) => string` 是构造选项（`dist/types/index.d.ts:56`），
   **同步**、每次节点渲染都调 —— 这就是「文件 src → 可画 src」的转换点。
2. **外壳的图片机制是现成的、与内核无关的**：`EditorProps` 里早就有
   `resolveImageUrl`（丢图时返回 null，外壳已有占位纪律）与 `onImagePaste`
   （内核交 `{ bytes, mimeType }`，外壳存盘后返回文件 src；返回 null 表示
   外壳已向用户报告过、内核不得再报第二条）。`DocumentPane` 对**每个**
   内核统一传这两个回调 —— 内核只要解构，外壳一个字都不用改。
3. **内核 paste 有官方让位口**：内核自身的 paste 先吃「复制的节点」
   （magic JSON，`MindElixir.js:1016-1027`），吃不下才把**原始 ClipboardEvent**
   交给 `pasteHandler`（:1028，构造选项）—— 我们的粘贴处理从这里进，
   不与内核的节点复制打架。
4. **内核对图片的读取是「缺则不画」**：三键缺一**静默不画**
   （判据 1 的另一半）。「图丢了」在我们的外壳语境里是**常见状态**
   （文件挪了、图删了），Markdown 内核的既有纪律就是**占位 + 路径如实**，
   导图照搬，不由内核的静默不画说了算。
5. **jsdom 不解码图片**：`Image` 不触发 `onload`，无布局 ——
   单元测试只能证**接缝**（字节交对了、src 换对了、文件动没动），
   像素观感必须人工冒烟。`decodeImageSize` 因此是 `node-images.ts`
   里的独立函数（测试里 `vi.stubGlobal('Image')` 假货可控）。
6. **各平台剪贴板填充方式不一致**（markdown 内核 `imageFilesOf`
   的既有教训）：有的把图片放进 `files`，有的只放进 `items` 且 `files`
   是空的 —— 取图片两处都要看，只看一处就是「只在别人的机器上坏」。

## 取舍

### ① `image` 升格为正字段，形状与内核一致

照 `icons`（ADR 0053）的先例，`image` 从 `extra` 透传桶升格：
parse / serialize / adapter 三层 + `KNOWN_NODE_KEYS` + `NON_EXTRA_NODE_KEYS`。
形状**原样采用内核的** `{ url, width, height, fit? }`，不发明中间表示 ——
升格前它走 extra 本就原样透传，链路已通；升格后若继续走 extra 会与正字段
重复写键（serialize 的 taken 守卫当场抛错，这条是 0053 的教训）。

- `url` 存**文件 src**（相对路径或 `data:`）—— 与 Markdown 的纪律一致：
  文件里写的路径就是引用，上屏前才解析成可画的 data:。
- `width` / `height` 存**显示尺寸**（尺寸即内容 —— 不存它，下次打开就是
  内核的默认猜值）；粘贴时按最长边 320 缩好（只缩不放，`Math.round` 保整）。
- **整对象照收**：`image` 里我们不认识的子键（`caption` 之类）原样跟着走；
  类型不对（非对象、`url` 非非空字符串、尺寸非正有限数、`fit` 出枚举）
  **整值转 `extra`** —— 与 `note` / `icons` 同一条纪律：一个字段要么信，要么整个不信。
- 键序在 `icons` 之后、`extra` 之前。

### ② 显示 = 预热 + `imageProxy` 同步查表，查不到画占位

`imageProxy` 是同步的，而解析（读盘、转 data:）是异步的 —— 所以**预热必须
发生在内核创建之前**：解析文档 → 收集 `image.url`（跳过 `data:` 与空串）→
`Promise.all` 逐个 `resolveImageUrl` → 填 `Map<src, dataUrl>` → 再装内核。
查表方式：`data:` 原样透传（本来就能画，不劳驾外壳）；查得到换 data:；
**查不到画占位图** —— 内嵌小 SVG（虚线框 + 🖼 + 文件路径），路径如实写在
占位图里，「图不在这里」这件事本身上屏。不去 fetch 外链（http 图片同样走
占位）：ADR 0001 零联网是文件级承诺，导图渲染不破例。

语言切换重建内核（ADR 0049 的命门）顺带重跑预热 —— 占位图里的说明文字
走 `t()`，烘焙后语言总是对的。

`setValue` 换文档时补预热：新引用逐个解析 → 填表 → 对带图节点
`reshapeNode(el, {})` 空 patch 重渲染（`Object.assign` 同值不产生 diff，
但 `Se` 全量重建载荷让占位换真图）。折叠着的节点看不见、不急，
展开时自然查到新表。

### ③ 插入 = 粘贴，走内核让位口，尺寸解码在内核层自建

右键菜单没有「插入图片」（mind-elixir 没这个 UI），XMind 用户的手最熟的是
**粘贴**。链路：`pasteHandler` 收原始 ClipboardEvent → 两处找图片文件
（判据 6）→ 不是图片就放手（文本粘贴归内核）→ 没选中主题如实说
（`onError`，绝不静默）→ 字节交 `onImagePaste` → 外壳返回文件 src →
`decodeImageSize` 解码宽高（`createObjectURL` + `Image`，失败如实说）→
缩到最长边 320 → 粘贴字节内联 data: 填显示缓存 → `reshapeNode(el, { image })`。
落盘走 `operation` → 既有 emit 主通道，与标记同一条链；只读模式
`onPaste` 第一行直接 return —— 图片是编辑，不悄悄放行。

PNG 光栅化**不借内核的 `exportPng`**（那是导出整张图的事，见 ADR 0050）；
内核没有「把一张图片转成 data:」的 API，`bytesToDataUrl`（FileReader）
就是全部 —— 三个平台（Tauri WebView / 浏览器 / jsdom）都有 FileReader，
不自建原生桥。

### ④ 移除 = 右键「图片…」弹层，两项都有事可做

复用 0053 调色盘的弹层形状：有图显示**引用路径 + 显示尺寸 + 移除按钮**
（`reshapeNode(el, { image: undefined })` —— `JSON.stringify` 把 undefined
键整个丢掉，文件里不剩 `"image": null` 半句话）；无图显示「选中它，按
Ctrl+V」的提示 —— 空弹层是静默，提示才是如实。`fit` 刻意不进弹层状态：
弹层纯展示，不提供编辑（改 `fit` 是未来需求，别先造 UI）。

### ⑤ 导出与大纲：已经安全，不用伺候

- 导出（ADR 0050 的 `exportImage`）：上屏的 src 已是 `data:`（预热换过 /
  粘贴内联），`exportSvg` 写出的 `<image href>` 自包含；`countExternalImages`
  只数非 `data:` 的 —— 占位图（内嵌 SVG 的 data:）不会误报。文件里存相对
  路径不影响：导出的是**画出来的样子**，画出来的是什么就导什么。
- 大纲（ADR 0052）：`extractMindmapOutline` 只取 `topic`，`image` 不参与
  —— 大纲是文字导航，不是缩略图画廊。这条刻意不动。

## 后果

**测到的**：parse（四键齐收、不认识的子键跟着走、`fit` 可省略；四例坏类型
整值转 extra 原样往返）；serialize（键序 `icons` 后 `extra` 前、有则原样写出
undefined 不写）；adapter（image 是正字段、extra 里不再有它、双向直通）；
`node-images.ts` 20 条（引用收集去重跳 data:、占位图前缀 + 转义 + 超长截断、
预热四分支、剪贴板两处取图含 jsdom 空壳 items 的平台差、尺寸钳制只缩不放
保比例、`bytesToDataUrl`、`decodeImageSize` 真假 onload）；`MindmapEditor`
9 条 jsdom 全链路（预热显示 / 丢图占位 / data: 透传 / 粘贴落盘全链 /
无选中如实说 / 外壳失败静默 / 只读不接招 / 弹层移除 / 无图提示）。
全量 check 全绿。

**没测到的**：观感 —— 占位图的字号、虚线框样式、`<img>` 在主题上的实际
布局；`Image` 解码在真 WebView 里的时序。jsdom 无布局不解码，与 ADR 0050
的 PNG 像素同一句老话：需要 `pnpm run tauri dev` 人工冒烟（开一张带图
的 `.mindmap`，粘贴一张截图，挪走图片文件重开看占位）。

**已知边界**：文件里的 `http(s)` 外链图片**上屏走占位**（不联网），
但导出 SVG 时内核把原始 `image.url` 写进 `<image href>` —— 导出的是
文件层的引用，打开导出文件时能否加载取决于查看器自身（与 ADR 0050
「外部引用如实计数并警告」同一条纪律）。粘贴的图片字节**不进 `.mindmap`
文件**（那会让 JSON 爆炸）—— 它们作为独立图片文件住在文档旁边，
这是与 Markdown 一致的既有纪律。

**本轮的工程教训**（记在实现注释里，留给下一个接内核的人）：effect 的
清理函数可能抢在内核创建之前执行（预热是异步的），所以 effect 体改成
`boot` 闭包返回 teardown，`cancelled` 标志守卫预热回调 ——
「创建是异步的，销毁就必须能对上没创建出来的状态」。

## 重审条件

1. **换思维导图内核**：新内核没有 `image` 等价物时，`node-images.ts`
   （预热 / 粘贴 / 占位 / 解码）随内核一起换；格式层 `image` 字段**不变**
   —— `url` + 显示尺寸是文件层承诺，与内核无关。
2. **`.mindmap` 单文件分发提上日程**（把图内嵌进 JSON）：届时在**外壳**
   `onImagePaste` / `resolveImageUrl` 层做（外壳知道分发形态），内核与
   格式层不动 —— `url` 存什么由外壳的返回值决定，这条缝已经在。
3. **粘贴支持拖放 / 文件选择插入**：入口不同，落点相同 —— 都汇到
   `onImagePaste` → `reshapeNode({ image })` 这一条链，别开第二条路。
4. **`fit` 要可编辑 / 宽高要可拖**：弹层此刻纯展示（取舍 ④ 刻意留白），
   届时改 `ImagePopoverState`，落盘同走 `reshapeNode`。
5. **Markdown 内核丢图占位策略变更**：导图占位图应与 Markdown 的占位
   一起审视 —— 两者是同一件事（「图丢了是常见状态」）的两个渲染面，
   别让它们长成两种用户体验。
