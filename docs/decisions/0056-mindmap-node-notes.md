# 0056 — 导图节点的备注：内核零渲染，可见性我们自己建

- 状态：已采纳
- 日期：2026-10-13
- 关联：[0025](./0025-mindmap-kernel-integration.md)、[0049](./0049-mindmap-ui-follows-the-language.md)、[0052](./0052-mindmap-outline-panel.md)、[0053](./0053-mindmap-markers.md)、[0054](./0054-mindmap-node-images.md)、[0055](./0055-mindmap-links-and-tags.md)
- 承接：0055 重审条件 5 预留的 `note` —— 本轮就是那次重审：专门一轮做完，不是顺手带。

## 背景

对标 XMind 第一屏的最后一块：主题备注。LevNote 的 `.mindmap` 格式层早就有
`note`（键序第三个，parse / serialize 一直忠实往返），但 mind-elixir **零渲染
零 UI** —— 渲染链一个字都不画它，右键菜单没有它的入口。用户视角看：这个功能
不存在 —— 写进去的东西永远看不见。与标记（0053）、插图（0054）、链接与标签
（0055）不同，这次连「渲染」都要我们自己建。

## 判据（决策依赖的事实）

1. **`note` 是内核数据模型的正字段、渲染链却零提及**：
   `NodeObj.note?: string`（`dist/types/types/index.d.ts:210`，与 `hyperLink`
   相邻）。渲染 `Se`（`MindElixir.js:353-378`）只画 text / image / hyperLink /
   icons / tags 五样 —— 探针实证：带 `note` 的主题挂载后
   `me-tpc` 里没有任何备注痕迹。**格式层无需升格**（它早就是正字段，键序
   `id → topic → note → …`），本轮动的全是「看得见、改得了」。
2. **undo 插件把 `reshapeNode` 都记进历史**：`Kn`（`MindElixir.js:1464-1517`）
   监听 operation（除 `beginEdit` 外）push `{ prev, next, operation }`；
   `undo()` / `redo()` 调 `refresh(prev | next)`，**不 fire operation**。
   键盘 Ctrl+Z / Ctrl+Y 挂在 `.map-container` 上且要求 `editable`（:1509-1513）
   —— 只读没有 undo。
3. **refresh 必经 linkDiv、linkDiv 必 fire 总线事件**：`refresh`（:817）末尾
   `layout() + linkDiv()`；`linkDiv`（`Wn`，:1330-1353）末尾恒定
   `bus.fire("linkDiv")`；`init`（:2755）同链。我们已有 `linkDiv` 监听
   （0049 重建验证用的同一条）⇒ **一切跟着内核重建走的同步都有现成挂点**，
   覆盖 operation / undo / redo / refresh / expand 全部路径。
4. **undo 恢复的文本不是逐字节原文**（内核规范化：children 补 `direction`、
   键序重排）—— 探针实证；语义判据用「root 的 note 回来了」，不用字节比对。
   （0055 轮探到的同一条，本轮复用。）
5. **`exportSvg` 按选择器挑元素**（`zo`，`MindElixir.js:2634-2669`）：只有
   `me-tpc` / `.tags > span` / `.icons > span` / `.hyper-link` / `img` 五种会被
   选走。**未知类名的元素不进 SVG / PNG** —— 我们自挂的徽章天然不进导出，
   不崩不报，是可依赖的语义（见取舍 ⑤）。
6. **`me-tpc` 子序**（探针实证）：`SPAN.text | A.hyper-link | SPAN.icons |
   DIV.tags`，append 末尾即可，不与内核重写的内层打架。jsdom 的
   `innerText` / `textContent` 差异沿用 0055 的教训。

## 取舍

### ① 可见性 = 我们自己的徽章，不 fork 内核渲染

内核零渲染（判据 1），「看得见」只能自己建：`refreshNoteBadges(mind)` 遍历
`mind.nodes` 里每个 `me-tpc`，节点 `nodeObj.note` 是非空字符串就挂
`button.mm-note-badge`（📝），没有就摘。同步时机是 init 后一次 + `linkDiv`
监听（判据 3 的链覆盖一切重建路径，探针已证 undo 之后徽章跟着回来）。
**不改内核渲染**：fork 内核 `Se` 会在升级时丢（0049 的重建命门），外部同步链
反而对升级稳健；代价是徽章在 DOM 上晚一拍（一次微任务级重挂），无感。

### ② 编辑 = 徽章 + 右键「备注…」双入口，落盘走老通道

徽章只解决「已有备注」的查看与修改；**没有备注的节点永远创建不了第一条**
—— 这是侦察中堵上的设计洞。右键菜单补「备注…」（0053–0055 同一条
`contextMenu.extend` 通道，第五项）。

- 弹层：textarea（**普通 Enter 是换行** —— 备注是多行正文；Ctrl+Enter 或
  按钮保存）+ 有 `note` 才渲染移除按钮 + 保存判据 `trim()` 非空。
- **清空 = 移除，不是存空串**：`{ note: undefined }` 被 `JSON.stringify`
  整键丢掉，文件里干净消失 —— 与 0053 `icons` 的 `[]` 纪律**刻意不同**：
  `[]` 是「曾有标记、现在全清」的如实记录（标记画得出来，空数组有视觉语义）；
  `note` 内核零渲染，空串与 undefined 画出来一样，存空串只是文件里的噪声。
- 保存写的是 `trim()` 后的文本（与链接 / 标签同一条输入纪律），多行内部
  原样保留。
- 落盘照走 `reshapeNode` → `operation` → 既有 emit 主通道（0053 起四轮同一条），
  undo 历史自动把它记进去（判据 2）。

### ③ undo / redo：徽章重挂与脏标记迟到，都接在 linkDiv 上

undo 走 `refresh` 且不 fire operation（判据 2）⇒ 两个后果：
徽章被重建抹掉、onChange 不来。两条都由 `linkDiv` 兜底：徽章重同步（取舍 ①）
+ 既有 `scheduleVerify`（300ms 迟到的 onChange）。**迟到不是缺席** —— 探针
实证 Ctrl+Z 后约一拍 changes 里拿到恢复文本。这里没有新增机制，全是既有
链路的重新论证。

### ④ 大纲联动：📝 指示进面板，正文不进

0052 的判据：大纲来自 `extractMindmapOutline`（`doc.text` 纯函数），不碰内核
DOM —— 所以联动改的是纯函数与面板：`OutlineItem` 加 `note?: string`，
面板在 markers 之后加 `.outline-note` 的 📝。**指示是「有」，正文不进大纲** ——
大纲是导航不是阅读器。图上看不见的备注，大纲里看得见，正好补 0052
「大纲要去图上没有的地方」的另一半。

### ⑤ 导出与搜索不伺候

- 导出：判据 5 —— 徽章不进 SVG / PNG。这是**文档化的语义选择**：导出的是
  文件内容的呈现（同 0050「画出来的样子」），徽章是编辑器部件（同右键菜单
  不进导出）。`note` 本身在文件里，查看器侧没有渲染义务（与 0055 导出边界
  同款：「加载与否归查看器」）。
- 搜索：`EditorSearch` 的 `from/to` 是 topic 段内偏移、`replace` 走
  `setNodeTopic` 同步通道；`note` 写入是 `reshapeNode` 异步操作 —— 塞进同一
  契约要么伪造同步、要么动契约。`collectHits` 旧注释「只搜 topic 因为内核
  不渲染 note」的理由已过期（本轮起看得见了），注释改成新边界：note 可见
  仍不搜 —— replace 通道是 topic 专用；note 搜索记重审条件。

### ⑥ 只读：徽章照显，弹层只读

读备注是读不是编辑（0055 🔗 激活同款）：只读挂载徽章照常同步，点击弹层
textarea `readOnly`、整套按钮不渲染。右键入口随 `contextMenu: false` 关门。

## 后果

**测到的**：`MindmapEditor` 7 条 jsdom 全链路（右键末尾有「备注…」且空弹层
无移除按钮、空输入禁保存、保存后徽章长出 / 有 note 挂徽章、无 note 不挂、
title 走 i18n / 点徽章预填原文、Ctrl+Enter 落盘多行 / 移除后 `note` 键干净
消失、徽章摘掉 / **Ctrl+Z 后徽章重现 + 迟到的 onChange 追上**、undo 恢复判据
用语义不用字节 / 全空白保存禁而移除还在 / 只读徽章照显、弹层无按钮）；
`mindmap-outline` 的 note 透传；`App` 大纲 📝 挂对条目（研发有、另两条没有）
且正文不进大纲；parse / serialize 的 note 键位回归照旧全绿。全量 check
全绿（MindmapEditor +7、mindmap-outline +1、App +1）。

**没测到的**：观感 —— 徽章在主题右侧的实际位置、弹层尺寸；真 WebView 里的
Ctrl+Z 键盘路径（探针证的是 jsdom dispatch 的同一条链）。需要
`pnpm run tauri dev` 人工冒烟：给两个主题挂备注、徽章点开改一条、Ctrl+Z 移了
又找回来、导出 PNG 确认徽章不在图里、大纲里 📝 对得上。

**已知边界**：备注保存的是 `trim()` 后的文本（首尾空白不存），文件里已有的
历史 `note` 原样往返不重排（与所有正字段同一条：改写只在用户编辑时发生）；
`note` 里可以放任意文本（`be` 转义纪律归内核渲染侧，徽章只存与否、不渲染
正文，弹层 textarea 是 React 受控节点）。

**本轮的工程教训**：内核的 undo 走 `refresh` 且**不发 operation** ——
一切「跟着内核重建走」的自挂 DOM 与脏标记，兜底要挂 `linkDiv` 事件；
`operation` 是编辑主通道，`linkDiv` 是重建兜底，两条腿缺一不可。

## 重审条件

1. **换思维导图内核**：新内核若原生渲染 note，徽章退役（取舍 ① 的同步链
   随之删除）；届时「导出图片带不带备注」重定 —— 内核若把备注画进
   `me-tpc`，按判据 5 它就会进导出，语义会翻过来。格式层 `note` 不变。
2. **备注要进导出图片**：今天的边界是徽章不进（取舍 ⑤）。若产品要
   「导出的图里看得见备注」，那是导出管线的内容增补（exportSvg 后处理或
   PNG 角标），另立 ADR，别在徽章上顺手改。
3. **note 要可搜索 / 可替换**：要动 `EditorSearch` 契约（取舍 ⑤ 的边界）——
   `from/to` 的语义、异步 replace 的时序都要重新设计，届时与大纲搜索一起审。
4. **备注要富文本 / 样式**（XMind 备注是富文本）：先过 0053「字形即内容」
   的同款路线 —— 文本住文件层，样式住内核模块词表；结构化富文本是格式层
   breaking change，必须走 format version 迁移。
5. **大纲点条目要能「跳到备注」**：跳的是主题（0052 契约的 id 指主题）；
   「打开备注弹层」是新契约方法或面板内展开，别塞 `goToOutlineItem` 的
   可选参数。
