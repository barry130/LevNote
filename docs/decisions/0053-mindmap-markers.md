# 0053 — 导图标记：`icons` 升格为正字段，词表与调色盘归内核

- 状态：已采纳
- 日期：2026-10-12
- 关联：[0025](./0025-mindmap-kernel-integration.md)、[0047](./0047-words-that-outlive-the-language-switch.md)、[0048](./0048-mindmap-and-drawio-are-first-class-too.md)、[0049](./0049-mindmap-ui-follows-the-language.md)、[0052](./0052-mindmap-outline-panel.md)
- 承接：0050–0052 三轮自派需求补完了导图的「导出图片 / 大纲」两块；本轮补第三块 —— 对标 XMind 第一屏的优先级与标记。

## 背景

对比主流思维导图工具，XMind 的主题上挂着**优先级 1–9** 与各类标记图标，
这是它第一屏的核心能力：一眼扫过去就知道哪个分支最急、哪个已完成。
LevNote 此前连格式层的标记字段都没有 —— `.mindmap` 的节点表（format-spec §7.1.2）
里没有任何可以承载「这个主题很重要」的字段，内核也就无从显示。

本轮给三个来源补上这一整条链：格式层有地方存、适配层原样带、内核有 UI 可以点。

## 判据（决策依赖的事实）

1. **内核本来就认 `icons`**：`NodeObj.icons?: string[]`（mind-elixir
   `dist/types/types/index.d.ts:181`），渲染是内核自己做的：`Se` 全量重建主题载荷
   （`MindElixir.js:368-371`），字形字符串过 `be` 转义（`& < "`，:59-61）——
   没有注入面，清空传 `{ icons: [] }` 就行。**我们不需要在内核渲染层写任何代码。**
2. **`reshapeNode` 是现成的落盘通道**（`d.ts:153`，实现 `Xt` :507-515）：
   `Object.assign(nodeObj, patch)` → 重渲染 → fire `'operation'` 事件
   `{ name: 'reshapeNode', obj, origin }` —— 正好落进我们既有的 emit 主通道
   （`onOperation` 只跳 `beginEdit`）。标脏、保存、撤销不需要单独伺候。
3. **右键菜单有官方扩展点**：`contextMenu.extend?: { name, key?, onclick }[]`
   追加在菜单末尾；内核 `g` 处理 `contextmenu`（`MindElixir.js:1198-1203`）
   只认 `button === 2`，且 `setTimeout(…, 200)` 之后先 `selectNode`（:1202）
   再弹菜单 —— 所以 extend 项的 `onclick` 里 `mind.currentNode`
   就是右键到的那个节点。
4. **「清空」必须是 `[]` 而不是 `undefined`**：`getData` = `JSON.parse(getDataString())`
   深拷贝（:733-735），而 `JSON.stringify` 会把 `undefined` 键整个丢掉 ——
   若用 undefined 表示清空，一次内核往返它就变回「从来没有」，是一次静默改写。
5. **内核给节点 id 加 `me` 前缀**（:397 `t.dataset.nodeid = "me" + e.id`）——
   DOM 测试要找 `[data-nodeid="men1"]` 而不是 `n1`（本轮测试的三坑之一，见后果）。

## 取舍

### ① 字形即内容，不发明语义 id

`icons` 从 `extra` 透传桶**升格为正字段**（与 `style` / `note` 同级），
存**显示字形本身**（`"1️⃣"`、`"⭐"`）。XMind 的做法是存语义 id（`priority-1` 一类）
加它自己的图标资产 —— 我们不学，理由：

- **最内核无关**：文件里存的是能直接排版的文本，任何能排文本的内核都能显示；
  换内核不换文件。
- **不造双表示**：适配层头部本来就写着 icons 原样透传 —— 升格前它走 `extra`
  桶，链路已通；若再发明 `markers` 语义 id，就要维护 id↔emoji 双表，
  且与 extra 透传的 `icons` 形成同一件事的两套写法。
- **§5 的既有承诺顺延**：文件里出现词表之外的字形（用户手写、别的工具写的）
  照样原样保留。词表只是**界面的知识**，不是文件的裁判。
- XMind 互导真要做时，id↔字形对照表住**互导那一层**，不回渗格式层。

### ② 空数组是「全清过」，如实写

`undefined` 不写（没这回事）；`[]` 照写（全清过）。两者在文件里是**可区分的**，
丢掉前者就是保存后 diff 里少一句话。parse 读入时同样原样带回。

### ③ 类型不对整值转 `extra`

`"icons": 42`、`["1️⃣", 2]` 整个值转 `extra`，不逐项过滤 —— 与
`note` / `collapsed` / `style` 同一条纪律（一个字段要么信，要么整个不信）。

### ④ 词表住内核模块，不进 EditorHandle，不进外壳

`src/modules/mindmap/markers.ts`：19 个字形三组 —— 优先级 1️⃣–9️⃣（互斥）、
色标 🟥🟧🟨🟩🟦🟪（互斥）、符号 ⭐✅❗❓（不互斥）。调色盘 UI（右键「标记…」
→ 分组面板 → 点字形）是 `MindmapEditor` 内部实现，同右键菜单纪律：
**内核自己的 UI 不上 `EditorHandle` 契约**，外壳一个字都不知道。

互斥只是调色盘的交互语义（`applyMarkerToggle`）：标优先级 2 就先摘优先级 1。
文件层从不说「不许」—— 手写两个优先级进文件，内核照样两个都显示
（诚实呈现，不由词表做裁判）。追加一律排末尾，「后标的排在后面」是稳定规则，
文件往返不乱序。

### ⑤ 落盘走既有主通道，只读自然关门

点字形 → `mind.reshapeNode(topic, { icons })` → `operation` 事件 → 既有 emit →
标脏/保存，与手动编辑**同一条链**。只读模式 `contextMenu: false`，
右键菜单整条不存在 —— 标记是编辑，不悄悄放行（测试覆盖了这条）。

### ⑥ 大纲面板跟着亮

`OutlineItem` 加可选 `markers`（只有导图大纲设，Markdown 永不设）；
面板条目里内联显示字形（`aria-hidden` —— 不劳读屏逐个念）。
这是 0052 大纲契约的自然延伸，不动提取器结构。

## 后果

**测到的**：parse（icons 空数组照读照写、两例坏类型整值转 extra）；
serialize（键序含 icons、undefined 不写 `[]` 照写、词表外字形照写）；
adapter（icons 是正字段、extra 里不再有它 —— 升格前掉 extra 会与正字段
重复写键；空数组原样带回）；markers 词表 12 条（字形唯一、组引用自洽、
i18n 键真空、互斥组集合、`markerGroupOf` 词表外 null、`applyMarkerToggle` 六态）；
`MindmapEditor` 7 条 jsdom 右键全链路（菜单 → 调色盘 → 落盘 → 摘掉 → 互斥 →
清除 → 点外关闭 → 只读无菜单）。全量 check **86 文件 / 1775 测试全绿**。

**没测到的**：观感 —— 调色盘的实际布局、字形在主题上的大小、悬停文案；
jsdom 无布局无 hover，与 ADR 0050 的 PNG 像素同一句老话：需要
`pnpm run tauri dev` 人工冒烟。

**已知边界**：语言切换靠内核重建（ADR 0049 同一条命门，菜单与调色盘文案
全部走 `t()`）；词表外字形在调色盘里不亮（不认识就不装认识），文件里不动。

**本轮踩的三个 jsdom 坑**（都记在测试注释里，留给下一个写内核交互测试的人）：
内核右键只认 `button === 2`；菜单是 `setTimeout(200)` 之后才弹；
节点 DOM 的 `data-nodeid` 带 `me` 前缀。另有一条工具教训入档：
本机 PowerShell 对无 BOM 文件按 GBK 解码，`Get-Content | Set-Content` 文本往返
会把 UTF-8 文件毁成 mojibake —— 改非 ASCII 文件一律用 edit/write 工具。

**顺手修正**：ADR 0047–0052 的「日期」字段此前误标为未来日期（10-13 / 10-14），
本轮一并改回真实提交日（0047–0049：2026-10-11；0050–0052：2026-10-12），
连同 ROADMAP / AGENTS 里引用这些日期的句子。

## 重审条件

1. **换思维导图内核**：新内核没有 `icons` 等价物时，markers.ts 词表与调色盘 UI
   随内核一起换；格式层 `icons` 字段**不变** —— 字形即内容，这条是文件层承诺。
2. **XMind 互导提上日程**：id↔字形对照表住在互导层（`markers.ts` 文件头已预留
   这条边界），不回改格式层。
3. **标记要参与搜索 / 过滤**：`collectHits` 目前只搜 `topic`；届时先回词表模块
   加能力，字形解析不散进 `core`。
4. **调色盘要支持自定义词表**：`MINDMAP_MARKERS` / `MINDMAP_MARKER_GROUPS`
   已是数据不是代码，届时评估外置进偏好。
5. **大纲面板要按标记过滤 / 着色**：`OutlineItem.markers` 契约已带载荷，
   面板侧自行处理，不动提取器。
