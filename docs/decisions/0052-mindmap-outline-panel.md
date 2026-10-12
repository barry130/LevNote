# 0052 — 导图大纲面板：从 `doc.text` 用纯函数算条目，跳转契约带上文件层 id

- 状态：已采纳
- 日期：2026-10-14
- 关联：[0007](./0007-mindmap-core-choice.md)、[0025](./0025-mindmap-kernel-integration.md)、[0047](./0047-localized-text.md)、[0048](./0048-mindmap-drawio-first-class.md)、[0049](./0049-mindmap-ui-follows-the-language.md)
- 承接：ADR 0048 让 `.mindmap` 成为一等公民；本轮补上对标产品第一屏就有、而 LevNote 完全没有的能力。

## 背景

对比主流思维导图工具（XMind、幕布、GitMind 等），大纲视图是第一屏功能：
XMind 打开导图即可在侧栏看到整棵树的平铺列表，点击条目画布滚动定位到对应节点。

LevNote 的大纲面板（`src/ui/OutlinePanel.tsx`）此前只认 Markdown：
`outlineEnabled = isMarkdownPath(path)`（App.tsx），条目来自 `extractOutline(markdown)`。
`.mindmap` 文档打开时大纲按钮置灰 —— 明明手里有一棵树，却给不出大纲。

## 判据（侦察实证）

1. 现有跳转契约是 `EditorHandle.goToOutlineItem?(item: { level, text }): boolean`
   （`src/core/editor/types.ts`）。参数刻意不带行号：内核没有「行」的概念就别硬翻译；
   `id` 与 `level/text` 的拆分理由同 `goToAnchor` —— 数据来源不同，
   合成一个方法必有一边传假参数。
2. mind-elixir 的定位原语（`node_modules/mind-elixir/dist/types/index.d.ts:167-210`）：
   - `findEle(id)` 找不到会 **throw**（`MindElixir.js:341`
     `FindEle: Node ${e} not found, maybe it's collapsed.`）；
   - `selectNode(tpc)` 选中但不发 operation 事件 —— 选中不是编辑；
   - `scrollIntoView(el)` 用 `getBoundingClientRect` + `move`（jsdom 安全）；
   - `expandNode(el, isExpand)` 只对带 +/− 气泡的分支节点有效
     （`MindElixir.js:785-804` 读 `i.children[1]`），**根没有气泡** —— 对根调用会抛 TypeError；
   - `focusNode(el)` 不是滚动定位，是**专注模式重定根**（换 `nodeData` + `initRight`），不能用。
3. 文件层已有完整语义：`parseMindmap(text)`（`src/core/mindmap/parse.ts`）绝不抛，
   致命则返回 null；缺 id 的文件按先序补 `auto-0/1/2…`（确定性）；topic 数字/布尔转字符串。
   适配器把文件层 `collapsed: true` 映射为内核 `expanded: false`（`adapter.ts:113-130`），
   `collapsed` 是**文件里的字段**，展开/折叠经 onChange 回写并标脏。
4. 内核不隐藏根的子分支：探测实证（jsdom 挂载后 `[data-nodeid="mea1"]` 已在 DOM 里）
   —— 根折叠在视觉上不生效，子分支永远渲染；但对**中层**折叠，被折的孙节点不在 DOM 里，
   `findEle` 会 throw。
5. jsdom 无布局：滚动、像素定位不可断言；App 测试里大纲面板默认关
   （`outlineVisible: false`，`settings.ts:146`），测试须显式传设置。

## 取舍

### 1. 大纲从 `doc.text` 算，而不是从内核要

新纯函数 `extractMindmapOutline(text)`（`src/core/outline/mindmap-outline.ts`）复用
`parseMindmap` —— 与喂给内核的数据**同一套语义**：topic 强转、auto-id、容错行为全部一致。
自造第二份「什么算一个主题」的判定迟早漂移，跳转会悄悄指错地方。

条目 = 先序 DFS：root 的 `level` 是 1，`line` 是**先序序号**（不是行号 ——
导图文件是 JSON，行号没有意义；沿用 `OutlineItem.line` 字段名是为了不再造一个形状）。

好处顺带成立：未挂载也能算（面板不依赖内核就绪）、纯函数可全量测试。
源码视图手改 JSON 的中间态下 `parseMindmap` 返回 null，面板如实说
「暂时读不出导图的结构」（新 key `ui.outline.mmUnreadable`），不给假大纲。

### 2. 契约加可选 `id`：文件层 id，不是渲染层 DOM id

`goToOutlineItem(item: { level, text; id?: string })`。
主题文字可重名，`(level, text)` 只能找到首个匹配；文件层的 `id` 才指得准。
有 `id` 按 `id` 找，没有则回退 `(level, text)` —— Markdown 永不传 id
（`OutlineItem.id?` 只有导图设置；数据来源不同，缺着比编一个假值诚实）。

### 3. 内核跳转：展开折叠的祖先是文档变更，标脏是诚实的

`MindmapEditor.tsx` 的 `goToOutlineItem`：
沿 `findTopicPath`（先序，同 auto-id 语义）找到根到目标的链，
对链上 `collapsed === true` 的祖先逐个 `expandNode(findEle(id), true)`；
最后 `findEle(target.id)` + `selectNode` + `scrollIntoView` + `host.focus()`。

展开折叠节点会改写文件里的 `collapsed` 字段 —— 这是**文档变更**，onChange 与标脏照常发生。
导航把看不见的地方变可见，本来就该存进文件；不标脏反而是撒谎
（用户看到分支展开了，保存重开却还是折着的）。

### 4. 两个如实的例外：根不展开，只读不展开

- **根**：内核对根的折叠不做视觉隐藏（判据 4），也没有根的展开器气泡；
  对根调 `expandNode` 会抛 TypeError。实现里 `index === 0` 跳过展开 ——
  跳过去，但不假装展开过。
- **只读**（分屏预览）：展开是编辑，只读时折着的祖先就留在折着（返回 false）；
  可见的节点照跳（选中不是编辑，`selectNode` 不发 operation 事件）。

### 5. `findEle` 抛错交 false，不白屏

中途文件被外部改动、专注模式子树外、任何「找不到」的时刻：`findEle` throw 被捕获，
`goToOutlineItem` 交 `false`。外壳的既有行为是「这个内核不支持/跳不过去」，如实。

### 6. 面板分派按 `isMindmapPath`，外壳不认识内核

`OutlinePanel` 按 `isMindmapPath(state.doc.path)` 分派提取器与文案
（计数「{n} 个主题」、tooltip「点击跳到这个主题」、空主题「（空主题）」、
parse 致命的「暂时读不出导图的结构」）；App 的 `outlineEnabled` 改为
`isMarkdownPath(path) || isMindmapPath(path)`。
外壳知道「哪些文件是导图」是**产品范围**，不知道「用哪个内核渲染它」——
与既有 `isMarkdownPath` 决策同一条线（App.tsx 原注释）。

## 后果

**测到的**（85 文件 / 1750 测试全绿）：

- `mindmap-outline.test.ts`（6 条）：空串/坏 JSON → `[]`；先序层级序号 id 全量；
  折叠子树照样列出；auto-id 与内核同源；数字 topic 转字符串。
- `MindmapEditor.test.tsx`（9 条）：按 id 跳 → true 且 changes 空、getValue 逐字节不变
  （选中不是编辑）；`(level,text)` 回退；未知 id → false；根折叠照样跳（文件不动）；
  中层两折跳孙 → changes 恰 2、`collapsed` 字段从文件里消失；无 id 文件经
  auto-id 跳 → true（面板↔内核同源）；只读折着 → false、只读可见 → true。
- `App.test.tsx`（4 条）：`.mindmap` 文档开面板见 3 条主题与「3 个主题」计数；
  Markdown 对照不受影响；点击不抛；偏好默认关时面板不出现。

**没有测到的**（如实）：jsdom 无布局 —— `scrollIntoView` 的实际滚动、选中高亮的像素
位置不可断言，需 `pnpm run tauri dev` 人工冒烟（开 `.mindmap` → 开大纲 → 点深层条目）。

**已知边界**：`styles.css` 的 `.outline-item.level-N` 只定义到 6 级样式（7+ 用默认色）；
导图层级无上限，深层条目缩进靠面板 inline `paddingLeft`（任意深度都能缩进），
颜色分层放弃 —— 记为已知边界，不为导图加 CSS。

## 重审条件

1. **大纲可编辑**（XMind 的大纲是第二编辑视图）：现实现只导航。要做编辑时另立 ADR
   —— 那是「文件层写入通道」的新问题（谁写、undo 归谁），不是本契约的延伸。
2. **换内核**（mind-elixir → 其他）：`id` 的文件层契约不变；若新内核的定位原语
   不认文件层 id，适配器负责映射，不动 `goToOutlineItem` 契约。
3. **drawio 大纲**：`isMindmapPath` 分派线不动；drawio 若要大纲，另立 ADR
   （它的「结构」定义、提取器、跳转原语都是另一套事实，不复用 `extractMindmapOutline`）。
4. **层级过滤 / 大纲搜索**：先回到提取器层做（纯函数，可全量测试），不进内核契约。
5. **跳转后选中态被内核刷新掉**（选中依赖内核内部状态）：若实际使用中出现
   「点了面板但画布上看不出选中」，回来修 `selectNode` 之后的补充高亮，不动契约。
