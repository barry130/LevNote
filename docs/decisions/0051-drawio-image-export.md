# 0051 图纸画出来的样子也带得走 —— draw.io 接上同一条图片导出路

- 状态：已采纳
- 日期：2026-10-12
- 关联：硬约束 2（绝不静默）、ADR 0001（零联网）；[ADR 0048](./0048-mindmap-and-drawio-are-first-class-too.md)（图纸是一等公民；`toHtml()` 返回 `null` 的重审条件）；[ADR 0050](./0050-mindmap-image-export.md)（`exportImage` 契约与导出编排，本轮原样复用）
- 承接：用户让本项目「继续对比，完善我们的项目」。上一轮对比的另一半差距在本轮兑现：`.drawio` 文档此前**完全不能**导出图片 —— 而 draw.io 官方工具把「导出 PNG / SVG」放在第一屏。导图那轮（ADR 0050）铺好的路，这轮让图纸走上去。

## 背景

ADR 0050 定下的契约是给**所有**「画出来的内核」的：可选方法 `exportImage(format)`、`EditorImageExport { blob, warnings }`、`null` 表示这一次没能画出来。它当时的重审条件就写了：「diagram / drawio 接入时复用同一契约与 `export-image.ts`，**不另开导出路径**」。本轮就是那一条的兑现 —— 也是检验：契约接第二家内核时，外壳真的**一行没改**。

`file.exportPng` / `file.exportSvg` 两条命令按能力启用（`enabled` 里只认 `typeof ctx.ui.editor()?.exportImage === 'function'`，ADR 0050 原样）。图纸内核一实现，`.drawio` 文档上的这两条命令**自动亮起** —— 命令定义、菜单、导出编排、写盘纪律（create_new 绝不覆盖、半成品如实报），全都是上一轮已经在跑的代码。

要答的问题只剩内核侧的：图纸的「画出来的样子」从哪来？

## 判据

图纸内核（`DrawioEditor`）是只读查看器：`renderDrawio` 把 XML 交给 mxGraph 的 GraphViewer，画完的整张 `<svg>` 留在画布容器里。三个事实都从源码钉死：

1. **查看器没有导出 API**。`viewer.min.js` 只有「把 XML 画进容器」这一个职责 —— mind-elixir 那种现成的 `exportPng()` 在这里**不存在**，位图转换只能自己做。
2. **底色已经写在行内**。查看器把**解析好的**底色直接写进 svg 根的行内 style（`this.svg.style.backgroundColor = (source.background || defaultPageBackgroundColor).cssText`，默认 `#ffffff`）。序列化天然带着走 —— 导图那轮「画前换真实色、画后还原」的整套动作在这里**不需要**。
3. **图片形状用的是 `xlink:href`**（老产物；新产物用 `href`）。`outerHTML` 不补命名空间声明，那样的「SVG」独立打开时按 XML 一解析就报错。

## 取舍

### 1. 序列化用 `XMLSerializer`，不是 `outerHTML`

导出去的文件要能在**别人机器上**独立打开。`XMLSerializer` 会补上 `xmlns`（以及用到的 `xmlns:xlink`）声明；`outerHTML` 不会。判据第 3 条钉死了这一选择 —— 用错序列化器，产物在测试里看着挺好，离开本项目就打不开。jsdom 里它由 w3c-xmlserializer 实现，测试能真的跑过序列化这条路。

### 2. PNG 自己转，诚实边界与导图同一条

SVG 串 → `URL.createObjectURL` → `new Image()` → canvas 2d → `drawImage` → `toBlob('image/png')`，`finally` 里 revoke。位图链上任何一环失败（图加载不出、拿不到上下文、`toBlob` 交回 `null`、尺寸解析不出 0×0）一律**交回 `null`**，让外壳如实报「内核没能把图画出来，没有生成文件」。自建这条链不是越过谁 —— 是查看器根本没有这条链（判据第 1 条）；将来拿到 draw.io 编辑器资产时它该退役（见重审条件）。

位图这边没有「底色行内」的待遇：SVG 当位图源用时，根节点的行内 backgroundColor **不参与绘制**，画出来会变成透明底。所以画布先铺 `getComputedStyle(svg).backgroundColor` —— 解析出来是 `rgb` 系且非全透明才铺，解析不出（jsdom 就是）就不铺：宁可底色可疑，也不编一个颜色糊弄。SVG 分支不需要这步，行内 style 自己会跟着走。

### 3. 导「此刻画在画布上的那张」

`drawnDiagram` 取画布上的第一个 `<svg>`（`instanceof SVGSVGElement` 收窄）。查看器画的就是**当前页、当前缩放**的那张图 —— 序列化它就是「把用户正看着的导出去」，所见即所得。多页文档此刻只有一页在画布上，导整册是另一件事（见重审条件）。

**导出要等正在画的那一趟**：查看器是异步画完的，`renderRef` 挂着在途的 `renderDrawio` Promise，`exportImage` 先 `await` 它 —— 「刚打开就立刻导出」不许拍到半张图；画完才有的机会窗口（失败 / 空文件）里画布上没有 svg，按契约交 `null`，不是编一张空图。

### 4. 警告三句，都是「文件离开面板后还得跟着走」的实话

外链图片形状导出去之后**不再是自包含的**，用户必须知道；缺 stencil 库的形状**按矩形显示**，这个「与屏幕上看到的一致」的约定离开应用后没人解释。三句话都通过 ADR 0047 的 `LocalizedText` 随文件带出去：

- 外链图片 + PNG：画不出来（转换时不去读它们）—— `ui.drawio.exportPngImagesExternal` {n}；
- 外链图片 + SVG：保留的是原引用，打开时要那些文件还在 —— `ui.drawio.exportSvgImagesExternal` {n}；
- 缺 stencil 库：按矩形显示，与屏幕上看到的一致 —— `ui.drawio.exportMissingLibs` {libs}，库名单用 `t('ui.drawio.libJoin')` 拼（分隔符也是 key，ADR 0047 的纪律）。

`countExternalImages` 的判定：`href` 或 `xlink:href` 非 `data:` 即外链 —— 两处属性都看（判据第 3 条），`data:` 的已内嵌，不算。

### 5. `toHtml()` 维持 `null` —— 与 ADR 0048 重审条件的关系

ADR 0048 给 `toHtml()` 返回 `null` 留的重审条件是：要做 HTML 导出，先答「外链图片怎么办」。本轮**不是**那条的路：`exportImage` 导的是二进制图文件，不进自包含 HTML，外链图片不会在导出物里变成联网请求（ADR 0001 不受影响）。但那条条件里的「**明确标注**」先在这条路上兑现了 —— 外链图片数出来进 `warnings`。把图片**换成内联**仍未做：那要真去读外部文件的字节，内核没有读盘能力。两条路各答各的：`toHtml()` 继续 `null`（自包含 HTML 退回 `<pre>` 呈现 XML），`exportImage` 走二进制快照。

## 后果

- 外壳零改动兑现：`src/core/commands/definitions.ts` 与 `src/core/export/export-image.ts` 本轮只有注释改口（「导图、图纸先后接上的就是这同一条路；diagram 接上时还是它」），能力检查、菜单、编排、写盘一个字没动 —— 这正是 ADR 0050 当初把契约放 `core/editor/types.ts` 的意义。
- 新增 3 对文案（`ui.drawio.exportPng/SvgImagesExternal`、`ui.drawio.exportMissingLibs`，zh/en 同占位符）；复用 `core.export.imageDrawFailed`（null ⇒ 「内核没能把图画出来，没有生成文件」）、`core.export.*` 导出家族与 `core.images.partialWrite`。

### 测到的

- `src/modules/drawio/DrawioEditor.test.tsx` 新增 7 条：handle 真的有 `exportImage`；SVG 分支交得出 `image/svg+xml` Blob、`blob.text()` 含 `xmlns="http://www.w3.org/2000/svg"` 声明（`XMLSerializer` 的证据）、无外链无缺库时 `warnings` 为空；外链夹具（1 外链 + 1 `data:`）恰报 1 条 `ui.drawio.exportSvgImagesExternal` 且原引用留在串里；缺库 XML 报 `ui.drawio.exportMissingLibs`；渲染失败交 `null`；空文件交 `null`；连导两次一致。
- `pnpm run check` 全绿：84 文件 / 1731 测（基线 1724 + 本轮 7）。

**没有测到的，如实写出来**：

- **PNG 路径完全没测**：jsdom 的 `Image` 永不加载，`svgToPng` 在测试环境里不会 resolve。与导图轮同一条记录 —— 不可测，不是未测。
- **像素一个都没测**：jsdom 没有布局引擎，底色铺不上（`getComputedStyle` 解析不出，走「不铺」分支）、尺寸来自夹具的宽高属性。底色、清晰度、缺库形状的实际观感，要靠 `pnpm run tauri dev` 人工冒烟：开一份 `.drawio`（带外链图片的、带缺库形状的各来一份）→ 两个格式各导一次 → 独立打开验。

## 重审条件

- **拿到 draw.io 编辑器资产时**（与 ADR 0048 的第一条重审条件同一天到来）：真编辑器有自己的导出入口 —— 那时 `exportImage` 应改为**借编辑器自己的导出**（与导图借 `mind-elixir` 的 `exportPng` 同理），本轮自建的 Image+canvas 转换链退役；判据是「编辑器有没有现成的导出入口」，不是版本号。
- **外链图片想内联**：若内核拿到读盘能力（或外壳愿意把文件字节递给内核），两条 `exportPng/SvgImagesExternal` 警告应升级为「内联 + 报数」；`countExternalImages` 的判定（`href` / `xlink:href` 非 `data:`）保持不变。与 ADR 0050 的同名条件同步。
- **导出倍率 / DPI 的需求出现**：当前位图按画布上的内容尺寸原样导出，没有倍率概念。若用户要「导大图 / 高分辨率」，那要谈的是**契约扩展**（`EditorImageExport` 加字段），先扩契约再实现 —— 不许在图纸内核里塞私货。
- **多页文档要「导整册」**：当前导的是画布上的当前页。真编辑器接入后多页才成为现实，届时该有的是「逐页导出」或「选中页导出」，仍然走同一条契约。
- **PNG 底色在真机上缺席**：若人工冒烟发现底色没铺上（`getComputedStyle(svg).backgroundColor` 在 Tauri WebView 里解析为空或透明），优先修「解析 → 铺」这条链的取值来源，不许改成硬编码白色糊弄。
