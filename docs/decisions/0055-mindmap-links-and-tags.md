# 0055 — 导图节点的链接与标签：`hyperLink` 与 `tags` 升格为正字段，内核零 UI 就补 UI

- 状态：已采纳
- 日期：2026-10-12
- 关联：[0025](./0025-mindmap-kernel-integration.md)、[0048](./0048-mindmap-and-drawio-are-first-class-too.md)、[0049](./0049-mindmap-ui-follows-the-language.md)、[0050](./0050-mindmap-exports-what-it-draws.md)、[0053](./0053-mindmap-markers.md)、[0054](./0054-mindmap-node-images.md)
- 承接：0050–0054 五轮自派需求补完了导图的「导出图片 / 大纲 / 标记 / 插图」；本轮补对标 XMind 第一屏的最后两块 —— 主题超链接与标签。第三个候选 `note` 内核完全不渲染，本轮刻意跳过（见重审条件 5）。

## 背景

对比主流思维导图工具，XMind 的主题上除了标记和图片，还能挂**超链接**（网页、
mailto、指向本地文件）和**标签**（「重要」「风险」这类小 chip）。LevNote 的
`.mindmap` 此前只有标记与插图，主题挂不上一条链接，也没有标签 —— 而 Markdown
内核早有一整套链接机制（`LinkActivation`、外壳分类打开、文件路径可跳到工作区
里的另一份文档）。本轮把这套现成机制接到导图上，不另造。

## 判据（决策依赖的事实）

1. **内核原生渲染这两样，形状现成**：`NodeObj.hyperLink?: string`
   （`dist/types/types/index.d.ts:182`，**纯字符串**，无 title、无显示文字）；
   `NodeObj.tags?: (string | TagObj)[]`（:180，`TagObj { text, style?, className? }`
   :156-160）。渲染在 `Se`（`MindElixir.js:364-378`）：🔗 是
   `a.hyper-link`（`target=_blank`、`innerText` 画 🔗、`href` 原样赋值）；tags 是
   `div.tags > span` 逐个（字符串直接 textContent，对象元素 text + className +
   style）。假值 / 空数组**不画**并清掉 DOM 引用。
2. **内核对 hyperLink 零 UI**：右键菜单里的 `link` / `linkBidirectional` 项是
   `createArrow` 节点间连线（`Xn`，`MindElixir.js:1382-1453`），与 hyperLink
   无关；grep 全库 `hyper-link` 只有渲染（:366）与导出（:2687）两处 ——
   **内核没给锚点挂任何点击监听**，WebView 里裸点会整页导航走（markdown
   `link.ts` 文件头记的同一条教训）。编辑弹层、点击拦截都得我们自己写。
3. **外壳的链接机制是现成的、与内核无关的**：`LinkActivation { href, text }`
   （`src/core/editor/types.ts:774-779`）与 `onLinkClick` 在 `EditorMountProps`
   里，`DocumentPane` 对**每个**内核统一传 —— 内核只要解构，外壳一个字都不用改。
   `App.tsx` 的 `openLink` 五出口（空链接如实说 / 不支持的协议如实说 / 锚点 /
   文件 / 外部），`classifyLink` 只认 http(s) / mailto 为外部 —— **文件路径可以
   打开工作区里的另一份 `.mindmap`**，这是导图链接最有用的形态。
4. **导出已自带这两样**：`exportSvg` 把 `.tags > span` 画成 SVG 文本、把
   `.hyper-link` 交给 `Fo` 写成 `<a href>`（`MindElixir.js:2681-2690`）——
   导出零改动，链接在导出的 SVG 里仍然可点（URL 本来就在文件里，是能力
   不是泄漏），与 ADR 0050「导出画出来的样子」同一句话。
5. **tags 的对象元素带 `style` / `className`**：内核渲染时 `Object.assign` 到
   span 上（判据 1）—— 这是内核渲染合同的一部分。文件层照存原样，
   不解释也不清洗（与 `note` / `icons` 同一条：一个字段要么信，要么整个不信）。
6. **jsdom 的 `innerText` 有实现、`textContent` 是空**（探针实证）：内核拿
   `innerText` 画 🔗，jsdom 里读回 `textContent` 拿到空串 —— 激活上报的
   `text` 必须读 `innerText`。右键菜单三坑（button 必须显式 2、菜单
   `setTimeout(200)` 后才弹、节点 id 有 `me` 前缀）沿用 0053 的既有结论。

## 取舍

### ① 升格形状照内核，不发明中间表示

照 `icons`（0053）/ `image`（0054）的先例，`hyperLink` 与 `tags` 从 `extra`
透传桶升格：parse / serialize / adapter 三层 + `KNOWN_NODE_KEYS` +
`NON_EXTRA_NODE_KEYS`。升格前它们走 extra 本就原样透传，链路已通；升格后若
继续走 extra 会与正字段重复写键（serialize 的 taken 守卫当场抛错）。

- `hyperLink` 是**非空字符串**：`""` 整值转 `extra` —— 内核对假值不渲染，
  空串是画不出来的链接。原文**不 trim 不解码**，分类与打开归外壳
  `classifyLink`（与 markdown 同一出口）。
- `tags` 是 `(string | { text, className?, style? })[]` —— **文字即内容**
  （与 `icons` 存显示字形同理）。对象元素我们不认识的子键**原样照收**
  （`riskLevel` 之类跟着走）；混入一个坏形状**整条**转 `extra`，不逐项过滤 ——
  防「悄悄只留好的一半」。空数组 `[]` 照写：那是「全清过」的如实记录。
- 键序在 `image` 之后、`extra` 之前。

### ② 编辑 = 右键弹层（内核零 UI，我们补）

- 「链接…」：提示行（支持什么）+ 输入框（**预填当前值**）+ 保存 + 移除。
  没链接时移除按钮**压根没有**（与 0054 无图弹层同款「不是禁用，是没有」）；
  输入为空时保存禁用 —— 空串落盘是画不出来的链接（取舍 ①）。
- 「标签…」：chip 列表（✕ 按索引摘）+ 输入 + 添加（回车同义）。**挂一个不关
  弹层**（连挂是正常用法，0053 字形同款）；重复标签 no-op —— 现值原样返回，
  不落第二份；摘到最后一个留下 `"tags": []`，空列表提示与空输入并存。
- 落盘统一走 `reshapeNode` → `operation` → 既有 emit 主通道；`tagAdd` /
  `tagRemove` **从内核现取 `nodeObj.tags` 算新组**（marker 先例）—— 不依赖
  弹层快照，避免 setState updater 里的副作用。只读模式 `contextMenu: false`，
  菜单整个关门 —— 挂链接与标签是编辑。

### ③ 点击 = 🔗 单击即激活（与 markdown 的 Ctrl+点击 刻意不同）

内核不拦锚点的默认行为（判据 2），我们在 host 上挂**捕获阶段** click 监听：
命中 `a.hyper-link` 就无条件 `preventDefault` + `stopPropagation`（WebView
不能整页飞走），`href` 从 `nodeObj` 读、不从 DOM 读（DOM 会说谎），`text` 读
`innerText`（内核拿它画的，判据 6），交给外壳 `onLinkClick` 分类打开。

**为什么单击就走、markdown 却要 Ctrl**：markdown 正文里的链接嵌在文字中间，
单击首先要能「进去改字」；导图的 🔗 是用户亲手挂在主题上的**显式 chip**，
点它没有别的意图 —— XMind 同款，单击即走。激活是**读不是编辑**：只读模式
同样激活（查链接不是改文件）。

### ④ 导出与大纲不伺候

- 导出：判据 4 已证内核自带，零改动。
- 大纲：`extractMindmapOutline` 只取 `topic`（0052 的刻意），tags 不参与 ——
  大纲是文字导航，chip 挤进大纲是另一个需求。

## 后果

**测到的**：parse（hyperLink + tags 往返、对象元素未知子键 `riskLevel` 原样
跟着走、空数组照读照写；BAD_FIELD 五例：`hyperLink: 42` / `''` / `tags` 非数组 /
`['重要', 2]` / 缺 `text` 的对象，全部整值转 extra 原样往返）；serialize（键序
`image` 后 `extra` 前、`hyperLink` undefined 不写、`tags` 空数组照写）；adapter
（extra 用例换 `hyperLinkHref` / `tagList` 样本 —— 真未知键；两字段正字段用例：
双向直通、extra 里不再有它们、`tags: []` 从内核拿回来还是空数组）；
`MindmapEditor` 9 条 jsdom 全链路（预填改链落盘弹层自关 / 无链接给提示且无移除
按钮且空输入禁保存 / 移除后文件里不再有 `hyperLink` / 点 🔗 交 `{ href, text: '🔗' }`
且 `defaultPrevented` 且文件一个字节不动 / 只读可点 🔗 但无编辑菜单 / 回车与
按钮都能挂标签、挂后清 draft 不关弹层 / 重复标签 no-op 文件不动 / ✕ 按索引摘、
最后一个留下 `"tags": []` 与空列表提示 / chip 文字在独立 span 里）。
全量 check 全绿。

**没测到的**：观感 —— chip 圆角与弹层布局、🔗 在主题上的实际位置；
真 WebView 里 `innerText` 与导航拦截的最终表现（jsdom 证的是「我们的监听
先拦住」，不是「WebView 真的不飞」）。需要 `pnpm run tauri dev` 人工冒烟：
挂一条链接点它、指向另一份 `.mindmap` 再点、挂两个标签导出 SVG 用浏览器打开。

**已知边界**：`hyperLink` 存**原文**（不 trim 不解码）—— `mailto:` / 未知协议
交外壳后由 `openLink` 如实说「不支持」，与 markdown 同一条；导出 SVG 里的
`<a href>` 可点性由打开 SVG 的查看器决定（与 0054 的外链 `<image href>`
同一条：导出的是文件层的引用，加载与否归查看器）。`tags` 的 `style` /
`className` 是内核渲染合同的一部分，我们不解释也不清洗 —— 文件里写什么
就渲染什么（`dangerouslySetInnerHTML` 一类的键本来就只可能出现在 `extra`）。

**本轮的工程教训**：jsdom 实现了 `innerText` 但不回填 `textContent` ——
「读回」要顺着「写进」的那条 API 走，别想当然换成姐妹属性。

## 重审条件

1. **换思维导图内核**：新内核没有 `hyperLink` / `tags` 等价物时，渲染与
   弹层随内核换；格式层两字段**不变** —— 「字符串链接」与「文字或带样式的
   文字」是文件层承诺，与内核无关（适配层吸收对齐损耗，同 `expanded` ↔
   `collapsed` 先例）。
2. **链接要 title / 显示文字**（内核 `hyperLink` 是纯字符串）：升级成对象
   `{ href, title? }` 是**格式层 breaking change** —— 届时必须走 format
   version 迁移，不能悄悄改形状；显示文字上屏另算，编辑 UI 先改
   `LinkPopoverState`。
3. **标签要颜色 / 样式词表**（XMind 有标签样式库）：别发明语义 id —— 走
   0053「字形即内容」的同款路线：样式住内核模块词表供 UI 调色，文件存文字
   与样式原样。
4. **大纲要显示标签**：先回 `extractMindmapOutline` —— 大纲面板的数据源是
   纯函数（0052 的判据），不是内核 DOM；大纲条目的 `markers` 内联（0053）
   与 `tags` 内联届时一起审视。
5. **`note` 内核不渲染仍是已知差距**：本轮刻意跳过（三个候选里风险最高 ——
   渲染、编辑、上屏全要自建）。重审时连同「note 要不要上屏」一起定，
   别在下一轮顺手做。
