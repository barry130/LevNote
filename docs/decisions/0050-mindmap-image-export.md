# 0050 导图画出来的样子归内核 —— 新增「导出为 PNG / SVG 图片」

- 状态：已采纳
- 日期：2026-10-12
- 关联：硬约束 2、7、11；[ADR 0047](./0047-words-that-outlive-the-language-switch.md)（跨层消息用 `LocalizedText`）；[ADR 0048](./0048-mindmap-and-drawio-are-first-class-too.md)（导图是一等公民）
- 承接：用户让本项目自己对比主流思维导图软件找差距（「你自己比较一下现在主流的思维导图软件，看看你还欠缺什么，然后自己给自己需求继续开发」）。差距清单里，「完全不能把导图导出成图片」是最实的一条 —— XMind、FreeMind、MindMaster 都把「导出 PNG」放在第一屏。图标 / 优先级标记那一族是格式层的事，另立决定。

## 背景

轻笺已有的两条导出都在文本管线上：自包含 HTML（`core/export/export-document.ts`）与打印 PDF（`core/export/print-document.ts`），起点都是内核的 `toHtml()`。而导图内核的 `toHtml()` 依契约**合法地返回 `null`**（`MindmapEditor.tsx` 的注释当年就写了：「真要做富导出，正解是内核的 `exportSvg()` 那条路，那是另一个决定」）。

本轮做的就是那个决定。要回答的第一个问题是：它该是 `toHtml()` 的扩展，还是另一条路？

**另一条路。** `toHtml()` 的语义是「外壳拿**语义 HTML** 去拼自包含网页」；而导图画出来的样子 —— 连线走向、节点盒子、分支配色、折叠关系 —— 长在内核的 DOM 与布局里，外壳没有能把它复现的信息，硬凑的结果一定不像。所以这一轮的形状是「**请内核把自己画一张**」，产物是二进制，外壳只管问存到哪、写盘、转达内核的实话。

## 判据

`mind-elixir` 5.15.1 自己带导出，但直接用会踩两个坑，都被源码钉死了：

1. **底色是 `var()`**。内核把 `theme.cssVar['--bgcolor']`（本项目配的是 `'var(--surface)'`）**原样**写进导出 SVG 底板矩形的 `fill`（`dist/MindElixir.js:2671`）。在应用里没问题 —— 容器上挂着 `--surface`；可一份**独立打开**的 SVG 文件里没有这个定义，`var()` 解析不出来，底板黑掉。
2. **外链图片不自包含**。内核把 `<img>` 的 src **原样**写成 SVG 的 `href`（`Ko`，`dist/MindElixir.js:2654-2663`）。`data:` 的无所谓；引用外部文件的，SVG 里只是个引用（打开时要原文件还在），PNG 里干脆**画不出来** —— SVG 作为图片加载时不取外部资源。静默缺图是硬约束 2 的违约。

## 取舍

### 1. 契约：可选方法 `exportImage`，产物带 `warnings`

```
exportImage?(format: 'png' | 'svg'): Promise<EditorImageExport | null>
EditorImageExport = { readonly blob: Blob; readonly warnings: readonly LocalizedText[] }
```

- **可选**：markdown / code / plaintext 导「图」没有意义（逐字排文本不构成一张图），它们不实现；命令按能力禁用（与 `diagnostics.roundTrip` 同一个形状，`enabled` 里只认 `typeof ctx.ui.editor()?.exportImage === 'function'`，**绝不**按内核 id 分支 —— diagram / drawio 将来接上同一个入口自动亮起来）。
- **`warnings` 为什么在契约里**：判据第 2 条那种实话只有内核知道（它才有 `mind.nodes`），外壳问不出来。「绝不静默」要求它跨层传出来，而 ADR 0047 的 `LocalizedText` 正是「会跨过下一次 `setLocale` 的消息」的载体 —— 警告从内核发出到用户看见，中间要过导出编排、命令层、通知栏三层，早就跨过渲染了。
- **`null` 的语义**：这一次没能画出来（位图编码失败等），是**失败**，调用方必须如实报；整个方法缺失才是「不支持」。
- **警告不拒绝**：带外链图片的导图也导得出去，警告如实跟着。拒绝的话用户无从修复 —— 内核没有读盘能力，内联不了外部图片。

### 2. 底色：画图前换成真实解析色，画完还原

修法是画图前 `getComputedStyle(mind.container).getPropertyValue('--bgcolor')` —— 浏览器对自定义属性的计算值会把 `var(--surface)` 替换成**此刻真实的色值**（浅色 `#ffffff`、深色 `#1f1f1f`，还跟着主题走）。解析不出来（空串或还含 `var(` —— jsdom 没有层叠样式就是这样）就**保留原样**：宁可导出底色可疑，也不编一个颜色糊弄。

两条实现纪律：

- **整体重赋值，不原地改**：`mind.theme` 与模块级 `LEVNOTE_THEME` 是**同一个引用**（内核 installTheme 就是 `this.theme = e`），原地改会污染它，下一个实例被牵连。所以 `mind.theme = { ...prev, cssVar: { ...prev.cssVar, '--bgcolor': resolved } }`，`finally` 里还原。
- **为什么还原不会抢在「画」之前**：内核 `exportPng` 第一行是**同步**调 `exportSvg` 拿到 SVG 字符串再开始异步转换（`dist/MindElixir.js:2710-2711`）。换主题 → `await exportPng()` → finally 还原，SVG 在还原前就已生成。

### 3. PNG 不重画，交给内核

`mind.exportPng()` 内部完成「SVG 串 → data URL → `Image` → canvas → PNG Blob」，`onerror` 会 reject（如实报原因），`toBlob` 给 null 交 `null`（如实报「没能画出来」）。外壳不自己重实现这条链 —— 重实现就要追着内核的序列化语义跑，升级即烂。SVG 更简单：`exportSvg()` 是同步的，直接包 Blob。

**保留内核默认的 `foreignObject` 文字**（不传 `noForeignObject`）：保真度高、字号字重都内联。代价如实写在这里：部分老 SVG 查看器不支持 `foreignObject`，在那类工具里文字会不见 —— 用户有 SVG/PNG 两个格式可选，PNG 不受此限。

### 4. 编排：先问存到哪，再画

`core/export/export-image.ts` 与 `export-document.ts` 同级并列（不是它的分支）。顺序是 `pickSavePath(标题.格式)` → 内核画 → `blob.arrayBuffer()` → `writeBinaryFile`。**先问后画**：PNG 要位图编码，用户在对话框上点取消的话这笔时间不该白花。三分支结果形状与 HTML 导出**同形**（saved / cancelled / failed），外壳的通知逻辑零新概念。

### 5. 导出家族统一「绝不覆盖」

`writeBinaryFile` 是 create_new 语义：目标已存在以 CONFLICT 原文失败（「目标文件已存在，为避免覆盖已有内容，本次写入已取消」）。这不是图片导出的新规矩 —— HTML 导出走 `writeTextFile({ create: true })`，同为 create_new，**从来就不覆盖**。`DocumentProvider` 也没有删除 / 覆盖 API。如实转达 CONFLICT 原文即可；用户在系统对话框里看到的「确认替换」由系统对话框自己负责，轮不到写盘层猜。

### 6. 两条命令、不给快捷键

`file.exportPng` / `file.exportSvg`，按能力启用，同进「导出」类与文件菜单。不给 keybinding：`Mod+Shift+E` 已归 HTML 导出，图片导出是低频动作，再排一个组合键只会平添一个谁也记不住的绑定。

## 后果

- 导图是第一个实现 `exportImage` 的内核；`toHtml()` 依旧返回 `null` —— 本 ADR 开的「富导出」路是「导一张图文件」，不是「给自包含 HTML 提供内容片段」，`<pre>` 呈现 JSON 原文的那条降级不变。
- 新增 6 对文案：`command.file.exportPng/Svg.title`、`core.export.imageBuildFailed`（{reason}）、`core.export.imageDrawFailed`、`ui.mindmap.exportPng/SvgImagesExternal`（{n}）；复用 `core.export.dialogFailed/writeFailed/exported` 与 `core.images.partialWrite`（半成品写盘与图片落盘是同一条纪律）。

### 测到的

- `src/core/export/export-image.test.ts`（9 条）：先问后画（取消不画不写）、建议名带对后缀、字节**原样**落盘、警告以「；」拼进通报、draw 抛 / draw null / 对话框炸 / 写盘炸 / 半成品各自如实报。
- `src/core/commands/definitions.test.ts`（4 条）：enabled 只认能力（无 `exportImage` 即禁用）、PNG 全链（建议名剥扩展名、字节进盘、警告进通知）、内核缺失时安静 no-op、draw null 按失败通报。
- `src/modules/mindmap/MindmapEditor.test.tsx`（3 条）：handle 真的有 `exportImage`；SVG 交得出真的 `image/svg+xml` Blob 且无外链图片时 `warnings` 为空；连导两次一致（主题还原没还原好第二次就画不出）。

**没有测到的，如实写出来**：

- **PNG 路径完全没测**：内核转 PNG 用 `new Image()` 等加载事件，jsdom 的图片永远不加载 —— 那条路在测试环境里不会 resolve，只会挂起。这不是可测而未测，是不可测。
- **像素一个都没测**：jsdom 没有布局引擎，导出的 SVG 里画布尺寸是 200×200（节点全量成 0）、底色保留 `var(--surface)`（解析不出，走回退）。底板真实颜色、外链图片的两种下场、文字保真，都要靠 `pnpm run tauri dev` 里的人工冒烟：开一份 `.mindmap` → 两个格式各导一次 → 独立打开验底色、深浅两个主题各来一遍。
- 单测里 SVG 的 `blob.size > 0` 只证明「画了东西」，不证明「画对了」。

## 重审条件

- **内核升级**：若 `mind-elixir` 新版改了底板 `fill` 的来源（不再原样读 `theme.cssVar['--bgcolor']`）或 `exportPng` 不再先同步后异步，`withResolvedBackground` 的换色窗口与还原时序要重新核过 —— 判据是那两处源码行为，不是版本号。
- **第三家内核接入**：diagram / drawio 实现 `exportImage` 时，直接复用本契约与 `export-image.ts`，**不**新开导出路径；若它们的实现发现契约缺字段（比如需要报告 DPI），先扩契约再实现，不许在命令层塞私货。
- **覆盖导出的需求出现**：若用户提出「导出时允许覆盖同名文件」，那要动的是导出**家族**（HTML 一起改），在 `DocumentProvider` 上加一个显式的「覆盖」入口，而不是图片这条偷偷放宽。
- **外链图片想内联**：若将来内核拿到读盘能力（或外壳愿意把文件字节递给内核），`exportSvgImagesExternal` 那条警告应升级为「内联 + 报数」，`countExternalImages` 的判定（`src` 非 `data:`）保持不变。
- **foreignObject 兼容性实际造成抱怨**：若真有用户的 SVG 查看器不支持 `foreignObject`，再评估加「纯文本模式导出」（`noForeignObject` 的逐行 `<text>` 路径），判据是那一类查看器的实际占比，不是猜的。
