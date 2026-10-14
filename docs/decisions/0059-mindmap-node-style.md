# 0059 — 导图节点的样式：内核一直在渲染它，只是没人改得了它

- 状态：已采纳
- 日期：2026-10-14
- 关联：[0025](./0025-mindmap-kernel-integration.md)、[0049](./0049-mindmap-ui-follows-the-language.md)、[0050](./0050-mindmap-image-export.md)、[0052](./0052-mindmap-outline-panel.md)、[0053](./0053-mindmap-markers.md)、[0054](./0054-mindmap-node-images.md)、[0055](./0055-mindmap-links-and-tags.md)、[0056](./0056-mindmap-node-notes.md)
- 承接：0056 那一族「内核零 UI、我们自己建」的最后一格 —— 但这一格与前三格**性质不同**：前几轮都要先动格式层，这一轮格式层一个字都不用改。

## 背景

对标 XMind 主题样式面板的第一屏：文字颜色、填充、字号、加粗。

`.mindmap` 格式层**早就有** `style`（§7.1 骨架键，不透明直通），
mind-elixir 也**一直在渲染它**（`Se` 把每个键原样塞给 CSSOM）。
缺的只有一样：全库没有任何一处能**写**它 —— 右键菜单十一条内置项里没有样式，
`contextMenu.extend` 此前只挂了标记（0053）、插图（0054）、链接与标签（0055）、
备注（0056）。用户视角看：从别的工具拿来的图，颜色显示得出来；
自己画的图，一个颜色也刷不上。

所以这一轮不是「加一个字段」，而是「**给一个早已在渲染的字段开一个入口**」——
这也是它与 0053–0056 最大的差别：没有 parse / serialize / adapter 的改动，
风险全部集中在「怎么写才不留脏值」和「编辑器与导出两头都要对」上。

## 判据（决策依赖的事实）

1. **`style` 是格式层的正字段、且刻意不透明**：
   `MindmapNode.style?: Record<string, unknown>`（`src/core/mindmap/types.ts:152-153`，
   注释就是「不解释只保留」），`KNOWN_NODE_KEYS` 里早已有它
   （`src/core/mindmap/types.ts:90-101`）。`src/core/mindmap/parse.ts:307-309`
   只收纯对象（否则 `badField('style', …)`）；`src/core/mindmap/serialize.ts:96-99`
   键序在 `collapsed` 之后、`icons` 之前。
   ⇒ **格式层一个字都不用改。**

2. **适配层也是双向直通，且空对象不进文件**：
   `src/modules/mindmap/adapter.ts:92-105` 的 `NON_EXTRA_NODE_KEYS` 含 `style`
   （所以不会与 `extra` 双写 —— `serialize.ts:135-142` 的重复键守卫会抛）；
   `:143` 与 `:217` 都是 `if (node.style !== undefined && Object.keys(node.style).length > 0)`。
   ⇒ 词表撤空之后文件里连 `"style"` 都不留，不需要我们额外擦。

3. **内核渲染是「不透明直通」**：`Se`（`MindElixir.js:343-348`）
   `if (t.style) { const n = t.style; for (const o in n) e.style[o] = n[o]; }`。
   三条后果都要记住：① 清空 `children` **不**清内联样式；② 非法值被 CSSOM
   **静默忽略**（无单位的非零长度就是）；③ 值写 `''` 能删掉一条声明，
   写 `undefined` **不能**（无效值直接忽略）。

4. **内核导出侧只认四样**：`LayoutSsr.js:45-47` 只把 `color` / `background` /
   `font-size: ${fontSize}px` / `font-weight` 拼进独立 SVG 的内联样式
   （`:16` 还抄了一份 `branchColor`）。⇒ 可编辑词表**必须**恰好是这四样：
   多一个键就是「编辑器里画得出、导出的图里没有」，正是 ADR 0050 要拦的落差。

5. **`reshapeNode` 是同步的，`operation` 在函数体里同步 fire**：
   `Xt`（`MindElixir.js:507-515`）通篇没有 `await` ——
   `o.style && t.style && (t.style = Object.assign(o.style, t.style))` →
   `Object.assign(n, t)` → `Se` → `linkDiv()` → `bus.fire('operation', …)`。
   而 `MindmapEditor` 的 `onOperation` 是同步 `emit()`（→ 序列化 → `onChange`）。
   ⇒ 补丁交出去的那一刻**文本已经写出去了**，摘键必须在补丁里就干净 ——
   不能交个空串等 300ms 的 `scheduleVerify` 兜底，那会留下一个把
   `"color": ""` 写进文件的窗口。

6. **合并是「加 / 改」语义，摘键只能显式置 `undefined`**：`Object.assign(o.style, t.style)`
   只加不改不删；`Object.assign(n, t)` 会**照抄** `undefined` 值，
   而 `JSON.stringify` 丢 `undefined` 键 ⇒ `{ style: undefined }` 与
   `{ style: { ...rest, [key]: undefined } }` 都能干净摘除
   （`{ note: undefined }` 是 0056 已经验过的同一条）。

7. **`findEle` 对看不见的节点抛异常**：`FindEle: Node ${e} not found, maybe it's collapsed.`
   （`MindElixir.js:339-342`）—— 补丁路径必须接住并如实报，不许让它逃逸（0052 先例）。

8. **`refresh()` 重建元素、且不发 `operation`**：`Ht`（`MindElixir.js:395-398`）
   造**新的** `me-tpc`（新 `nodeObj`）；内核的 undo / redo 走 `refresh(prev)`。
   ⇒ 任何「内核画不出来的那一半」都得挂在 `linkDiv` 上重放（0056 徽章同一条）。

9. **只读时 `contextMenu: false` 已经关掉整个右键菜单**（0052 起的写法）
   ⇒ 编辑入口自然消失，不需要单独的只读弹层。

## 取舍

### ① 词表恰好是内核导出认得的那四个键，且住内核模块

`color` / `background` / `fontSize` / `fontWeight`，写在
`src/modules/mindmap/node-style.ts`。**词表是界面知识，不是格式语义** ——
文件里出现词表外的键（别的工具写的 `borderColor` 一类）照样原样往返，
与 `markers.ts` 同一条纪律（ADR 0053 ②）。

`branchColor` 刻意**不做**：它不在格式层的已知字段里（`NON_EXTRA_NODE_KEYS`
里没有它），因此住在 `extra` 桶 —— 把它变成可编辑的键等于一次**升格**
（0053 / 0054 / 0055 那种格式层改动），该单独一轮，不顺手带。

### ② 具体色值，不走主题变量

样式是**文件内容**：换台机器、换个主题都该长一个样。而 `var()` 进了独立 SVG
解析不出来（ADR 0050 给 `--bgcolor` 踩过的同一条）。
所以词表里是 8 个偏墨低饱和的文字色 + 8 个淡彩底，都是字面量
（`LEVNOTE_THEME.cssVar` 那套桥接只服务内核自己的画布底色，不服务节点样式）。

### ③ `fontSize` 存**数字**，屏幕上补 `px`

两头都要对，而两头要的形状相反：导出按 `${v}px` 拼接（要数字，字符串会拼成
非法值），CSSOM 对无单位长度按无效值忽略（要单位）。取舍是**文件里存数字**
（文件优先、导出优先），编辑器这一半由 `syncStyleFontSizes(mind.nodes)`
在渲染之后补一次 `n + 'px'`（`MindmapEditor.tsx:322-329`）。

时机与备注徽章完全同一条（ADR 0056）：挂 `linkDiv`（初始化一次 + 每次重画一次，
`MindmapEditor.tsx:1042-1046`），顺带覆盖 undo / redo 的 `refresh(prev)` 那条
不发 `operation` 的路。带单位的字符串值（别的工具写的 `'20px'`）内核自己画得对，
这里不碰 —— 不替别人改写。

### ④ 摘键 = 显式置 `undefined`，DOM 那一半用 `removeProperty`

落盘走 `styleApplyRef` / `styleClearRef` → `reshapeNode` → `operation` →
**既有 emit 主通道**（标脏 / 保存 / undo 历史全部免费，不单独伺候，0053 起的纪律）。
判据 5 决定了补丁必须是「交出去就干净」的形状：

- 设值：`{ style: { [key]: value } }`；
- 摘一个键：`{ style: { ...rest, [key]: undefined } }`；
- 摘掉**最后一个**键 / 「清除」：`{ style: undefined }` —— 不逐键置 `undefined`，
  因为一串 `undefined` 值的键会让 `Object.keys(...).length` 仍大于 0，
  适配层的守卫拦不住，文件里会留下一个 `"style": {}` 的空壳
  （规范 §7.1 说它与「没有」等价，但**等价不等于该写**）。

DOM 那一半必须自己擦：判据 3 ① 说清空 `children` 不清内联样式，
而 `t.style` 变成假值时 `Se` 整段跳过 —— 旧内联样式会**赖在元素上**。
擦法是 `el.style.removeProperty(cssPropertyName(key))`：`removeProperty` 只认
CSS 属性名（`font-size`），拿 camelCase（`fontSize`）去删会**静默什么都不删**；
转换函数与内核自己写的那份逐字同形（`LayoutSsr.js:62`，
`MindmapEditor.tsx:338-340`）。

### ⑤ 单键编辑不碰外来键，「清除」整份撤掉

`Object.assign` 的合并语义天然保留词表外的键：刷一个文字色，
文件里别的工具写的 `borderColor` 原样还在（§5 向前兼容）。
而「清除」是用户**显式**要「回到默认外观」—— 那一下连外来键一起撤掉。
§5 保护的是「我们不静默改用户的文件」，不是「用户明确要求的清除也不许做」。

### ⑥ UI 是内核自有界面，亮不亮由纯函数判

同 0053–0056 纪律：入口是右键 `contextMenu.extend` 末尾一条「样式…」
（`MindmapEditor.tsx:720`，标签直接用 `t()` —— 语言一变内核整个重建，ADR 0049），
界面是 `MindmapEditor` 内的 `StylePopover`（React，画在内核容器**之外**），
**不进 `EditorHandle`**。三行色块 + 一行字号档位 + 加粗 + 清除。

「哪个值算当前挂着」一律由 `node-style.ts` 的纯函数判
（`activeSwatchIndex` / `activeFontSize` / `isBold`）：**词表外的值不亮、
也不假装摘得掉** —— 一个亮着的色块意味着「再点一下能摘掉它」，
对 `'600'` 这种外来字重说谎比不亮更糟。

弹层里点色块**不关**（一次配好几样是正常用法，标记调色盘同款），
点外面 / Escape 关。弹层里那份 `style` 是打开那一刻的快照，只负责「亮哪个」；
写入之后由 `applyStyle` 回填（`styleApplyRef` 返回写完的现值）。

### ⑦ 只读：入口自然消失，不做只读弹层

`contextMenu: false` 已经把整个右键菜单关掉（判据 9）。
与备注（0056 徽章照显、弹层只读）不同：样式**没有**常驻可见物 ——
没有徽章这种「只读也该看得见」的东西，也就不需要只读弹层。

## 后果

- 用户能在 LevNote 里刷文字色 / 底色 / 字号 / 加粗，文件里就是 `style` 本身 ——
  与 XMind 一类工具双向互通（这个字段本来就在文件里）。
- **格式层与适配层零改动**：`style` 一直是正字段。本轮只补了「改得了」。
- 新增 `src/modules/mindmap/node-style.ts`（词表 + 三个纯判定函数）
  与 `node-style.test.ts`（17 条：词表自洽 —— 键清单用
  `Record<StyleKey, true>` 钉死、色值唯一且是六位十六进制、
  文字色够深 / 底色够浅、字号升序且是数字）、
  `MindmapEditor.test.tsx` 的 `describe('主题样式（ADR 0059）')` 11 条
  （菜单入口 / 三行词表齐全 / 刷色落盘 / 再点摘掉 / 摘最后一个键连字段都不留 /
  单键互不干扰 / 底色与文字色各查各的 / 数字字号 + 屏幕上补 `px` /
  加粗 / 外来键保留 + 清除整份撤 / 点外面关 / 只读无入口 / undo 恢复）。
- 已知边界（如实记，不假装没有）：
  ① 深色主题下浅底块与浅色文字的对比度**不保证** —— 样式是用户亲手刷的漆，
  跟着文件走不跟着主题翻；
  ② 带单位的字符串字号（`'20px'`）渲染侧正常、导出侧会被拼成非法值 ——
  原样保留，不替别人改写。

## 重审条件

1. **内核升级后 `LayoutSsr.js` 的映射范围变了**（多了或少了键）——
   词表要跟着调，否则立刻回到「编辑器里画得出、导出的图里没有」（判据 4）。
2. **若把 `branchColor` 升格为正字段**（或别的工具开始往 `style` 里写
   我们没认的键并期望可编辑）—— 词表要重开，这一轮的四键是导出侧的镜像，
   不是「样式该有什么」的完整答案。
3. **深色主题的对比度被当成缺陷报上来** —— 那时要决定「样式跟主题翻」
   还是「保持文件自持」，这是一个**语义**决定（会改变文件的含义），
   不是调色板调整，必须单独一轮。
4. **`fontSize` 的 px 补丁依赖 `linkDiv`**：若内核某次升级不再在重画后 fire
   `linkDiv`，编辑器里的数字字号会整片失效（导出不受影响，因为导出读文件）。
   那时改挂显式同步（`refresh` 之后 + `operation` 之后双保险）。
5. **若内核开始自己认数字字号**（渲染侧补单位）—— 本轮的 `syncStyleFontSizes`
   就是冗余，该删掉而不是留着两套。
