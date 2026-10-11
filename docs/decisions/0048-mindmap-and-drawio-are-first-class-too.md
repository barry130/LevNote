# 0048 另外两类文件也是一等公民 —— 思维导图与 draw.io 要有自己的入口

- 日期：2026-10-13
- 状态：已接受
- 关联：硬约束 1（依赖方向）、2（绝不静默）、11、13；ADR 0001（运行时不发网络请求）、
  ADR 0005（stencil 缺库是静默画错，实测结论 4）、ADR 0027（draw.io 只读查看器随包）、
  ADR 0043（降级要说出理由）、ADR 0047（存 key 不存句子）；ROADMAP P5 第 24/25/26 项

## 背景

用户看了一眼应用，说：

> 现在我只看到对于 MD 文件的大规模兼容和使用了，其他如思维导图、drawi 等文件呢？
> 我也需要单独支持他们，而不仅仅在 md 文件里面进行这些处理。

这句话说得对，而且比「补一个入口」更重：**「一等公民」是一整条链**，
少一环就还是「只能在 Markdown 里顺手做一下」。逐条核实之后，实际情况是：

| 类型 | 能单独打开吗 | 有新建入口吗 | 有独立内核吗 |
|---|---|---|---|
| `.md` | ✅ | ✅（`file.new`） | ✅ |
| `.mindmap` | ✅（打开对话框有 `filterMindmap`） | ❌ **零个** | ✅ `modules/mindmap` |
| `.drawio` | ✅（`filterDrawio`） | ❌ **零个** | ❌ 只有 doc-block 里的一种画法 |

两条缺口的性质完全不同，这决定了这一轮要分开处理：

- **`.mindmap` 的内核早就写好了**（ADR 0007 / 0025 / 0043 那一族），
  `mindmapAdapter` 认领 `.mindmap`、`sourceView` 是代码内核、被否决时还会说清楚为什么退。
  它只是**入口为零**：`file.new` 只会落一个空文本文件（`writeTextFile({ text: '' })`），
  侧栏「新建文件」走 Rust 的 `create_file`，同样是空文件。于是用户想新建一张导图，
  唯一的办法是「先新建一个 .md，再改名成 .mindmap」—— 一个没人会记住的姿势。
- **`.drawio` 连内核都没有**。`.drawio` / `.xml` / `.svg` / `.mindmap` 现在都被
  `CODE_EXTENSIONS` 认领，单独打开就是一段 XML 文本。随包有 `public/drawio/viewer.min.js`
  （2,783,375 字节 ≈ 2.654 MiB）与 `shapes/` `stencils/`，但那份产物**只被 doc-block 用**
  （`modules/markdown/code-block-view.ts` 里的私有实现），单文件打开这条路上没有它。

## 判据

「一等公民」不是一个形容词，是四条能验收的事。缺一条，用户就还得绕路：

1. **能新建**：不用「先建别的再改名」这种没人记得住的姿势。
2. **有独立内核**：单文件打开就画出来，不是 doc-block 的一种载荷。
3. **不在 Markdown 里也一样**：`.drawio` / `.mindmap` 各按各的渲染，各按各的编辑。
4. **缺能力要说出来**：不能编辑就是不能编辑，但面板上写着，且给出下一步。

第 4 条决定了这一轮**必须**把「编辑位」这件事摆到台面上，而不是闷头做一个只读版本就收工。

## 取舍

### 1. 「起始内容只有一份」，且判据只用 `core/path.ts`

`core/new-file-content.ts` 是本轮唯一的出口：

```ts
export function newFileContent(path: string): string {
  if (isMindmapPath(path)) return emptyMindmapText(titleOf(path));
  if (isDrawioPath(path)) return drawioTemplateXml();
  return '';
}
```

抽出来是因为**两个新建入口各写各的**：`file.new` 在前端 `writeTextFile`，
侧栏「在根目录新建文件」走 Rust `create_file`（`src-tauri/src/fs/mod.rs:398`，
注释里明写「创建**空**文件……空文件是合法的，打开它由前端决定，这里不替调用方预填任何内容」）。
将来每加一种有格式的文件都要改两处 —— 而漏掉的那一处是**静默**地给用户一个空文件。

- **模板是生成的，不是抄字面量的**：`.mindmap` 走
  `core/mindmap/template.ts` 的 `emptyMindmapDocument()` → `serializeMindmap()`。
  抄一份 JSON 字面量的话，格式一演进，新落盘的文件就是旧版的。
  用解析器自己的眼睛验：模板往返 `parseMindmap(...).canonical === true` ——
  「打开 → 保存」不产生 diff。
- **中心主题用文件名**（`titleOf(path)`），不编内容。导图节点的 id 固定是 `root`
  且不含 `"` / `\`、不重复 —— 判据抄自 `modules/mindmap/adapter.ts` 的 `UNSAFE_ID_CHARS`，
  但**不 import 那一层**（core 不该知道内核，硬约束 1）。
- **`.drawio` 的模板不引用任何 stencil 库**：随包只内置 `bpmn` 与 `flowchart`，
  新文件一打开就顶一条「未内置的形状库」警告是迎接用户的方式吗。
  判据用 `core/drawio.ts` 自己的 `missingStencilLibs` 量，不是照抄字面量。
- **`isDrawioPath` 刻意不收 `.xml`**：`.xml` 是泛型容器，绝大多数 `.xml` 不是图纸。
  打开对话框的 `filterDrawio` 列 xml 只是**入口**，认领要的是「按图渲染」这件事。

### 2. 三个新建入口共用一条编排，差异只留数据

`file.new` / `file.newMindmap` / `help.sampleDocument` 现在都走
`definitions.ts` 里的 `writeNewFileThenOpen(ctx, spec)`。判定顺序是
「存不存在 → 写不写得进去 → 打不打得开」，三者各有一句能说清楚的话。

`file.newMindmap` 之外，**侧栏新建 `.mindmap` 不改**：它落 0 字节，
而 `describeMindmapVeto('')` 返回 `null`（空文件**不是坏文件**，是「还没开始写」，
`adapter.test.ts:372` 已经钉住），所以打开就是一张带「中心主题」的导图画布。
两条路都合法，不必统一 —— 但这是**说清楚了的不统一**，不是漏掉。

### 3. `.drawio` 内核刻意**不实现** `canHandle`

`modules/mindmap/index.ts` 有一整套 `canHandle` / `canHandleReason`，
因为导图是「内核吃文本自己解析」，格式不对时只能由适配层判。
图纸不一样：随包的查看器自己会把「这不是合法的 XML」写进元素里，
`renderDrawio` 读出来就是 `DrawioRenderFail.message` —— **比适配层再判一遍更准**，
而且不必为它去顶 ADR 0043 那张 kind 表的 5 个上限。

所以 `modules/drawio/index.ts` 只有四行半：`id` / `label` / `matches` / `sourceView` / `Component`。
任何 `.drawio` 都归它，剩下的交给查看器说。

### 4. 只读内核**从不调用 `onChange`**

这是一个只读内核：它不能改文档，就**没有「改了什么」可报**。
一旦报了，外壳会把文档标脏、Ctrl+S 就把没变的文件重写一遍。

`toHtml()` 返回 `null` 也是有理由的，不是偷懒：图纸里可以有外链图片形状，
塞进自包含导出会让它去联网，那与 ADR 0001 直接冲突。
（导图内核同处理，两处都写了这个理由。）

### 5. 编辑位现在就铺好，形状就是 `sourceView: codeAdapter`

**`.drawio` 的文档内容本来就是那段 XML** —— 与 `.mindmap` 的 JSON 完全同构。
所以「可编辑」这件事在本轮不是没做，而是**有一个已经能用的入口**：
`drawioAdapter.sourceView = codeAdapter`，Ctrl+2 切到源码视图，XML 就是可编辑的文本，
Ctrl+S 走的就是所有文件都在走的那条落盘路。

**内核契约一个字都没改**（`EditorMountProps` 不动），这条路今天就能走。
将来真的接上 draw.io 编辑器，也只是把同一个 `EditorMountProps` 的另一个实现放进同一个位置。

### 6. 编辑器资产：**有实测读数，不是一句推测

随包那份 `viewer.min.js` 是不是也能当编辑器使？ROADMAP 第 26 项一直把它写成
「编辑器按需下载」，理由是体积；但在写这句话之前没人量过它到底能不能用。
这一轮量了（`scripts/v4-run.mjs`，与 `v1`/`v2`/`v3` 探针同一套：
`scripts/lib/harness.mjs` 起静态服务与浏览器、`scripts/lib/cdp.mjs` 走 CDP）。

四条判据的读数：

```
Editor: function   EditorUi: function   Graph: function   GraphViewer: function
mxUtils: object    mxGraph: function
App: undefined     EmbedFile: undefined
bootError: "Cannot set properties of undefined (setting 'lightbox')"
```

根因在产物里 @556907：

```js
Editor = function () { return new mxCellEditor(this) }
```

这是 **mxGraph 单元格编辑器的替身**，不是 draw.io 的 `Editor` 类。
`EditorUi` 第一个参数是 Editor 实例（`this.editor = b || new Editor; this.container = d || document.body;`），
于是它拿到一个 `mxCellEditor`，`u.lightbox` 直接炸在 `undefined` 上。
`App.MODE_EMBED` 与 `EmbedFile` 也因此都不在 —— 它们正是 `initializeEmbedMode` 回调要用的东西。

**这份 viewer 产物跑不起 draw.io 的编辑器，这不是引导写错能救的。**
（第一轮还踩了一个自己的坑：探针页的路径必须与随包分发时一致 ——
把 `editor.html` 挂在 `/editor.html` 上，它页面里的相对路径就解析到 `/viewer.min.js`，
而资源在 `/drawio/**`。）探针页因此**不留在 `public/`**：跑不起来的编辑器页面不进发行物。

沙箱无出网（`Invoke-WebRequest` 到 raw.githubusercontent.com 超时），
本机也没有 draw.io 安装（`Program Files` / `LOCALAPPDATA` / `Get-Command drawio` 全部落空），
所以**编辑器资产这一轮拿不到**。拿到它的命令见 README 与本 ADR 的「重审条件」。

### 7. 能力缺失写在面板上，不靠「没有那个按钮」

只读这件事在 `DrawioEditor` 里**一直**挂着一条说明，并给出可走的下一步
（切源码视图 Ctrl+2）。空图纸、渲染失败、缺 stencil 库三种占位各说各的话：

- **空图纸**：给的是「怎么让它有内容」，而且**根本不去调渲染器** ——
  渲染器会回一句「图表内容是空的」，那是一句准确的诊断，
  但对着一份还没开始写的文件毫无用处。
- **渲染失败**：渲染器的**原话一个字都不改、不翻**。那是 mxGraph 报的错，
  翻译它等于替系统说话（ADR 0043 的降级理由必须是真理由）。
- **缺 stencil 库**：说出来是哪些库。图**画出来了**，但那些形状退化成矩形，
  而 mxGraph 既不报错也不提示 —— 不说的话就是**静默地画错**
  （ADR 0005 的 V1 实测结论 4）。

这三条占位存的是**数据**（ADR 0047）：

```ts
type Blank =
  | { readonly kind: 'empty' }
  | { readonly kind: 'failed'; readonly reason: string }
  | { readonly kind: 'missing-libs'; readonly libs: readonly string[] };
```

于是切语言时前两种自己的话跟着变、缺库警告里的分隔符也跟着变（`ui.drawio.libJoin`
是一个 key，英文里顿号不对），而**渲染器的原话保持原样**。

## 一处只在这张图上才复现的死角

画布 `<div>` **一直挂着**，占位时给它加 `is-tucked`（`display: none`），
而不是条件渲染。因为条件挂载会让「从空图纸改成有图纸」这条路整个失效：
`xml` 变化时 effect 重跑，而那一刻 `canvasRef.current` 恰好是 `null`（正在显示占位），
于是什么也不画，且不会自己好。`DrawioEditor.test.tsx` 里有一条用例专门钉它。

## 后果

- 新增 `core/mindmap/template.ts`、`core/new-file-content.ts`（+ 测试）
  与 `modules/drawio/`（内核 + 样式表 + 测试）。
- 命令多一个 `file.newMindmap`（菜单与命令面板都有，关键词含「思维导图 / 导图 /
  mind map / siweidaotu」）；`file.new` 选 `.mindmap` 后缀也会落同一份模板。
- `core/drawio.ts` 的 doc-block 起始内容改为复用 `drawioTemplateXml()`，
  那个标签从「双击改内容，或直接编辑下面的 XML」改成「标题」——
  随包是只读查看器，双击并不会让你改到任何东西，**一句承诺做不到的话比不写更糟**。
- `modules/mindmap/MindmapEditor.tsx` 的 `createEmptyDocument()` 改为委托
  `emptyMindmapDocument('中心主题')`；两份各自抄字段，格式演进时就会有一份先掉队。
- `EditorModuleId` 加 `'drawio'`；注册表顺序变成 markdown → mindmap → **drawio** → code
  （`.drawio` 要在 code **之前**，而 `CODE_EXTENSIONS` 里**必须继续留着** `.drawio` ——
  那是 `sourceView` 指着的东西）。
- 探针 `scripts/v4-run.mjs` / `v4-host.html` / `v4-editor.html` 进仓，与 v1–v3 同一套。
- 门禁：82 个测试文件 / 1691 条用例全绿。

### 登记不做（连同理由一起钉住，别让人重新扫一遍）

- **`modules/markdown/code-block-view.ts` 里 6 处硬编码中文**（doc-block 渲染
  draw.io 时的说明文字与「未内置的形状库」那一条），它们没走 `t()`，
  切语言后仍是中文。这是 ADR 0047 那一族的**漏网**，要收就在下一次
  「`t()` 的结果跨渲染存活」那一轮里一起收，别单开一轮。
- **`App.tsx` 侧栏「新建文件」给 `.mindmap` 落模板**：要改就得让 Rust 的
  `create_file` 认识格式，那是另一侧的事；而空文件已经合法（见取舍 2）。

## 重审条件

- **拿到 draw.io 编辑器资产时**（`README.md` 里有获取命令）：把 `modules/drawio/`
  的 `Component` 换成真编辑器实现，`sourceView: codeAdapter` 保留（纯文本编辑
  永远要留着），「只能查看」那条说明删掉。接的时候判据不变 ——
  **内核把 XML 当文档内容经 `onChange` 交回外壳落盘**，
  契约（`EditorMountProps`）一个字都不用改，这是本轮提前铺好的原因。
- **`toHtml()` 一直返回 `null`**：一旦有用户要求「把图纸导出成 HTML」，
  先问一句外链图片怎么办 —— 直接做就会在导出物里埋下联网请求（ADR 0001）。
- **`isDrawioPath` 开始收 `.xml`**：那说明产品决定「`.xml` 按图渲染」，
  届时 `filterDrawio` 里那列 xml 就不再只是入口，而 `CODE_EXTENSIONS` 里
  `.xml` 与 `.drawio` 谁先谁后也要重排一次。
- **`newFileContent()` 长出第四个分支时**：先问一句「这条判据真的只该看后缀吗」。
  四个分支以内是四种文件类型；五个以上说明格式判定本身该挪出 `core/path.ts`。
- **`modules/drawio/` 的 `canHandle` 将来被补上**：那一天查看器不再自己
  报「不是合法 XML」，理由与取舍 3 就反过来了，届时要连同 ADR 0043 的 kind 表一起改。