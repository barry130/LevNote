# 0060 — 导图分支连线的颜色：一次升格，把「住 extra 的渲染合同」扶正

- 状态：已采纳
- 日期：2026-11-07
- 关联：[0025](./0025-mindmap-kernel-integration.md)、[0049](./0049-mindmap-ui-follows-the-language.md)、[0050](./0050-mindmap-image-export.md)、[0052](./0052-mindmap-outline-panel.md)、[0053](./0053-mindmap-markers.md)、[0055](./0055-mindmap-links-and-tags.md)、[0056](./0056-mindmap-node-notes.md)、[0059](./0059-mindmap-node-style.md)
- 承接：0059 刻意留下的那一格（取舍①的原话：「该单独一轮，不顺手携」）。`branchColor` 与 `style` 四键同属「内核一直在渲染、只是没人改得了」的字段，但它是**节点顶层键**、不在 `style` 对象里，且尚未升格 —— 所以这一轮比 0059 多一层格式层改动，比 0059 少一层 DOM 修补。

## 背景

对标 XMind 样式面板的分支连线颜色（branch line color）：用户给某个分支刷一种颜色，
这棵子树的连线跟着变。

`.mindmap` 格式层今天**不认识** `branchColor`（不在 `KNOWN_NODE_KEYS`，走节点
`extra` 桶双向透传）；mind-elixir 却**一直在渲染它** —— `linkDiv` 画连线时读
`nodeObj.branchColor`，没有值就按兄弟序号从 `theme.palette` 轮转。用户视角：
从别的工具拿来的图，分支颜色显示得出来（extra 原样进出 + 内核渲染）；
自己画的图，一个也刷不上 —— 与 0059 的处境一字不差。

差别在于：让 `style` 可编辑只要「写」（格式层与适配层早已就位），让
`branchColor` 可编辑还要**升格** —— 0053（`icons`）/ 0054（`image`）/
0055（`hyperLink` / `tags`）的同一张清单：类型规则、坏形状去 extra、
序列化键序、适配层从「extra 收编」改为「正字段直通」。

## 判据（决策依赖的事实）

1. **内核渲染合同**：`linkDiv`（`Wn`，`MindElixir.js:1331-1353`）开头
   `this.lines.innerHTML = ""`（`:1333`）—— 主分支连线**每次全量重画**；
   子分支 `dt`（`:1354-1367`）对每个节点 `remove` 旧 SVG 再重建（`:1350-1351`）。
   取色规则：主分支 `g = c.nodeObj.branchColor || y[l % y.length]`（`:1347`
   —— `y` 是 `theme.palette`，`l` 是兄弟序号）；子分支
   `C = m.firstChild.nodeObj.branchColor || n`（`:1359` —— `n` 是从上层
   传下来的**已解析**色）。⇒ 三条后果：① **子节点无值时继承祖先最近的
   branchColor**，设值即「这棵子树换色」；② 同色还会刷到主分支节点的
   `me-tpc` 边框（`c.style.borderColor = g`，`:1348`）；③ 颜色由
   `ct`（`:1259-1266`）**烤进 path 的 `stroke` 属性**。

2. **导出侧零词表约束 —— 与 0059 的判据 4 根本不同**：`exportSvg`（`zo`，
   `MindElixir.js:2664-2677`）克隆的是**已渲染**的 `.subLines` / `.lines`
   SVG —— 颜色在 `linkDiv` 那一刻就解析定案，导出照抄 `stroke` 属性。
   ⇒ 任何色值都自动带得走，不存在「编辑器画得出、导出没有」的落差；
   0059「词表必须恰好是导出认得的键」在这里**不适用**，词表是纯界面知识
   （与 `markers.ts` 同一条纪律）。`LayoutSsr.js:16` 只是把 `branchColor`
   抄进 SSR 布局数据（hydration 路径），与 `exportSvg` 无关。

3. **写路径全覆盖、无需 DOM 修补 —— 与 0059 的 `removeProperty` 关键差别**：
   `reshapeNode`（`Xt:507-515`）merge → `Se` → `linkDiv()` → `operation` ——
   落盘主通道本身就会重画连线；undo / redo 的 `refresh` 重建后 `linkDiv`
   同样会跑（连线是内核自己画的，跟数据走）。⇒ 写值、摘值之后 DOM 自动
   正确，**不需要** `removeProperty` 收尾、**不需要** `linkDiv` 监听兜底
   （0059 要做那两样，是因为内联样式是「赖在元素上的状态」而连线是
   「每次全量重建的 SVG」）。

4. **`getData` 本来就带它**：`on`（`MindElixir.js:700-710`）返回整棵活
   `nodeData`，`De`（`:725-729`）序列化时只丢 `parent` 对象引用。⇒ 节点
   上的 `branchColor` 今天就能随 `getData()` 流回适配层（节点层现读收进
   `extra`）。内核类型也声明了它：`NodeObj.branchColor?: string`
   （`dist/types/types/index.d.ts:194`）—— 适配层直通可用**正字段待遇**，
   不用 `setUnknown`。

5. **根节点例外**：`linkDiv` 只对 `me-main > me-wrapper`（根的子节点）读
   `branchColor`（`:1332-1334`），根自己没有连线。⇒ 根上的
   `branchColor` 无视觉效果（0052「根不展开」同族：内核没有那个视觉
   表达，UI 不假装有）。

6. **摘值语义与 0056/0059 同一条**：`Object.assign` 照抄 `undefined` 键、
   `linkDiv` 的 `||` 对 falsy 回落 palette、`JSON.stringify` 丢 `undefined`
   键 ⇒ `{ branchColor: undefined }` 一次写干净：文件无键、连线回到
   默认轮转。单值字段没有 `style` 的「留其余键」分支，比 0059 简单。

7. **默认轮转色就在我们手里**：`LEVNOTE_THEME.palette`
   （`MindmapEditor.tsx`）是主分支的取色环 —— 摘掉一个 branchColor
   回到的就是这张环。

## 取舍

### ① 升格：非空字符串，坏形状整值转 extra

`branchColor` 的形状与 `hyperLink`（0055）同一条纪律：**非空字符串**。
内核对 falsy 值 `||` 回落 —— `""` 是画不出来的颜色，装作读进来了就是
另一种静默丢内容。非字符串或 `""` ⇒ `badField` 整值转 `extra`（一个不丢，
§5）。四处改动照 0053/0054/0055 清单：

- `types.ts`：`KNOWN_NODE_KEYS` 加 `branchColor`；`MindmapNode.branchColor?: string`
  （注释里写明「值优先于 palette 轮转、子树继承、根无连线」的内核合同）；
- `parse.ts`：`typeof raw === 'string' && raw !== ''` 才收；
- `serialize.ts`：键序 `tags` 之后、`extra` 之前（与 parse 同序，往返稳定）；
- `adapter.ts`：`NON_EXTRA_NODE_KEYS` 加它（升格后**不能再走 extra** ——
  `serialize` 的重复键守卫会抛），两个方向都 `undefined` 不交（没这回事）。

### ② 词表与默认轮转是同一张表，且名单只有一份

`BRANCH_COLORS` 八个品牌色放在 `node-style.ts`，`LEVNOTE_THEME.palette`
**从它取** —— 不抄第二份（`MARKDOWN_EXTENSIONS` 的纪律：名单出现两份
的下场是「编辑器给的颜色和默认轮转对不上」）。语义自洽：摘掉 =
「回到默认轮转」，而默认轮转恰好就是这张表。词表是**界面知识**不是
格式语义（判据 2）：文件里词表外的色值照 §5 原样保留，弹层里不亮
（没有格子代表它，与 0059 同一条「不假装摘得掉」）。

### ③ UI：并进样式弹层第五行，根上不渲染

样式弹层加一行「连线」（与文字色 / 背景 / 字号 / 加粗同一张脸），
`canBranch` 为假（根节点）时整行不渲染 —— 给根刷一个画不出来的
颜色是骗人（判据 5）。落盘通道独立于 `styleApplyRef`：`branchColor`
不是 `style` 的子键，混进同一族补丁会让「style 对象补丁」的语义
带上一个顶层键 —— 两族数据一条通道，正是 0053 双写教训的温床。
toggle 判据从内核真值算（`nodeObj.branchColor === hex`，`markerApplyRef`
的先例），快照（`StylePopoverState.branchColor`）只负责「亮哪个」。
点色块不关弹层、点外面 / Escape 关、只读靠 `contextMenu: false`
自然关门 —— 全部与 0059 同款。

### ④ 「清除」一并摘掉 branchColor

「清除这个主题上的全部样式」的语义是**回到默认外观**，连线也在其中：
显式撤（`{ branchColor: undefined }`，与 `style: undefined` 两刀同一次
落盘）。这也给了词表外的 branchColor 一条**显式**摘除路径 —— §5 防的
是我们悄悄动它，不拦用户自己清（0059 判据 ⑥ 原文同款）。

## 测试与验收

- 纯函数（`node-style.test.ts`）：`BRANCH_COLORS` 形状（八项、唯一、
  六位 hex）；`activeBranchIndex` 词表内亮 / 词表外与非字符串不亮。
- 格式层（`parse` / `serialize` / 往返）：正字段读入、非字符串与 `''`
  转 extra、键序在 `tags` 与 `extra` 键之间、往返稳定。
- jsdom 全链路（`MindmapEditor.test.tsx`）：点色块 → `changes` 落盘 +
  连线 `stroke` 变色；再点一次 → 文件无键 + `stroke` 回落该节点序号
  的 palette 色（词表与轮转同源的闭环断言）；子节点继承父分支色；
  子节点自己设值时切换；根上无「连线」行；只读无入口；「清除」
  一并摘 branchColor；undo 后连线跟随数据（linkDiv 重画）。

## 已知边界

- 词表外的色值（别的工具写的 `#123456`）：文件保留、弹层不亮、
  只能靠「清除」显式撤 —— 与 0059 词表外 style 键同一待遇。
- 根上的 `branchColor`：照收照存（格式层不歧视），但内核不渲染、
  弹层不给入口。手工写进文件的照原样保留。
- `me-tpc` 边框同色是内核的副产品（判据 1 ②），不由我们解释。

## 重审条件

1. 内核升级改了 `linkDiv` 的取色或继承规则（判据 1） ⇒ 继承与回落
   断言重跑；
2. 内核给根也画连线 ⇒ `canBranch` 的根例外删除；
3. 词表外的 branchColor 需要逐值管理（自定义色拾色器） ⇒ UI 轮单独开，
   词表纪律（判据 2）届时一并重审；
4. `BRANCH_COLORS` 与 `LEVNOTE_THEME.palette` 若要分家（默认轮转换色
   不影响已有词表） ⇒ 必须同时改 `MindmapEditor` 的取用处，名单纪律
   （取舍②）重审。
