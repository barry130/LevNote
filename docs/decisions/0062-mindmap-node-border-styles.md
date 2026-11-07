# ADR 0062 · 导图节点边框 —— 颜色 / 粗细 / 线型，以及 0059 约束的适用面收窄

- 日期：2026-11-07
- 状态：已采纳
- 关联：ADR 0059（节点样式弹层）、0060（连线颜色）、0050（导出落差与 warnings 管道）、0003（绝不静默）

## 背景

XMind 对标候选清单的「节点边框样式」。按 0061 的教训**先探针内核**
（mind-elixir 5.15.1，`dist/MindElixir.js`）再定范围，四条证据改写了本轮的前提：

1. **渲染侧没有白名单**：`Se`（:343-348）把 `nodeObj.style` 逐键塞给
   CSSOM（`for (const o in n) e.style[o] = n[o]`）——`borderColor` /
   `borderWidth` / `borderStyle` 任何一个键都会原样生效，与 0059 的
   四个文本键走的是同一条路。
2. **导出侧对边框是专门通道，不是白名单**：这是本轮最重要的发现。
   0059 写过「词表恰好是内核导出认得的那四键」，依据是
   `LayoutSsr.js` 的 `foreignObject` 只拼 `fontFamily / fontSize /
   fontWeight / color` 四样——但**那只管文本**。节点矩形走的是另一段
   代码（`Xo` / `se`，`MindElixir.js:2610-2641`）：从
   `getComputedStyle` **现读** `borderColor → stroke`、
   `borderWidth → stroke-width`、`borderRadius → rx / ry` 拼进 SVG
   `rect`。DOM 上长什么样（无论谁写上去的），导出的图里就是什么样。
   ⇒ **0059 的「恰好四键」约束只适用于文本类键**，对边框类键不成立。
3. **编辑框也认边框**：`It`（:405-420，节点文字的就地编辑输入框）从
   computedStyle 拷 `border` / `borderRadius` —— 改边框后编辑中的输入框
   跟着长边框，无需任何补丁。
4. **主题层已有边框概念**（`:5-58` 的 `--root-border-color` /
   `--main-border` / `--root-radius`）——但那是主题级默认，节点级
   `style.borderColor` 塞的是内联样式，优先级高于主题，互不干扰。

两个探针结论直接划定了范围边界：

- **`borderStyle` 是唯一的导出落差**：导出 rect 没有
  `stroke-dasharray` 映射 —— 编辑器里画的虚线，导出的图里静默变实线。
  硬约束 2 不许静默 ⇒ 走 0050 的 `{blob, warnings}` 管道点名。
- **`boxShadow` 内核零支持**（全库 grep 无命中：渲染侧无、导出侧无），
  阴影不进本轮范围，也不排队 —— 除非换内核，做了也是「编辑器画得出、
  导出没有」且没有通道补。

## 决策

1. **边框三键进 `style` 桶，格式层与适配层零改动**。
   `style` 在格式层是不透明直通（§5，0059 已铺好），`borderColor` /
   `borderWidth` / `borderStyle` 与 `color` / `fontSize` 同住一个桶，
   序列化键序、`KNOWN_NODE_KEYS`、适配层透传全部复用。
   `StyleKey` 联合类型扩到七键（node-style.ts 的编译期哨兵同步）。
2. **词表**（`node-style.ts`）：
   - 颜色：**复用 `STYLE_TEXT_COLORS` 那八格**（`STYLE_BORDER_COLORS`
     就是它的别名）—— 名单只有一份的纪律；「偏墨低饱和压得住浅底」
     与文字色是同一条判据，不另抄一张表。
   - 粗细：`['1px', '2px', '3px']`，存**带单位字符串**。这是
     `fontSize` 教训的正面用法：渲染侧塞 CSSOM 必须有单位（数字被
     规范忽略），而导出侧现读 computedStyle、**不像字号那样拼
     `${v}px` 字符串**，带单位两头都成立 ⇒ **不需要**
     `syncStyleFontSizes` 那样的 linkDiv 兜底同步器。这条不对称
     （同在 style 桶里、一个要补丁一个不要）是本轮要钉进文档的
     知识，防后人给 borderWidth 也挂一个用不上的同步器。
   - 线型：`['solid', 'dashed']`。`solid` 在表里、再点一次 = 摘键 ——
     文件里留下显式 `"borderStyle": "solid"` 与「没设过」是两个状态，
     摘掉才是干净（0059 的摘一键纪律顺延）。
3. **落盘通道完全复用 0059**：`styleApplyRef` 对键名是泛型的
   （`reshapeNode(el, { style: { [key]: value } })`），三行 UI 的
   `onApply` 直接接上；摘键补丁（`{...rest, [k]: undefined}`）、
   内联样式收尾（`removeProperty(cssPropertyName(key))`，边框键的
   camelCase → kebab 转换是同一个函数）、「清除」整份撤掉
   （`{style: undefined, branchColor: undefined}` 同一次 reshapeNode），
   全部不需要为边框改一行。
4. **弹层加三行**（StylePopover 第六至八行，「连线」行之前）：
   边框色格画的是**空心框**（格子四边框即色值，mindmap.css 的
   `.mm-style-swatch-border`），粗细 chip 显示档位序号（1/2/3，不是
   `'2px'` 字符串——用户看的是「多粗」），虚线 chip 自身边框就是
   虚线（示意即语义）。四行可辨性问题（0059）由此扩成五种示意。
   `STYLE_EST_HEIGHT` 336 → 468。
5. **虚线导出落差不静默**：`countDashedBorders(root)` 纯函数遍历
   信封数据（不是 DOM——数据面是导出行为的真相源），数
   `borderStyle` 为**非 solid 非空字符串**的节点；`exportImage` 把
   个数交给 `warnings`（`ui.mindmap.exportBorderDashed`）。
   词表外的线型（别的工具写的 `'dotted'` / `'double'`）**照数** ——
   判据是「导出会画成实线」，不是「是不是词表写的」；而数字 / null
   这类 CSSOM 塞不进去的形状不数（编辑器里本来就没画出来，导出
   没丢东西）。
6. **节点形状（borderRadius 的 XMind 形状语义）刻意不做**：候选 ③
   另开一轮——形状的默认档位（根节点圆 / 分支圆角矩形）有主题层
   特例（`--root-radius: 30px` vs `--main-radius: 20px`），与「节点
   级覆写」的语义边界（XMind 的形状是互斥枚举，不是数值半径）需要
   单独探针，不搭边框的便车。
7. **阴影不做也不排队**（探针 4 的结论），除非换内核。

## 后果

- 正面：XMind 对标清单「节点边框」关闭；0059「恰好四键」的适用面
  收窄为「文本类键」—— 边框类键有导出专门通道，这条修正写进了
  node-style.ts 的文件头与 ADR，后人不会再拿白名单错拦边框类扩展
  （下一个就可能是形状 / 主题覆写）。
- 成本：StylePopover 多三行（估高 +132px）；`exportImage` 多一次
  数据遍历（导出本来就是全量画，可忽略）。
- 已知边界如实记：单独设 `borderStyle: 'dashed'` 而不设宽度时，
  CSS 的 `border-width` 默认 `medium`（3px）——虚线可见，词表语义
  自洽（三行独立、自由组合，XMind 同款）；导出图里虚线一律实线、
  由 warnings 点名。
- 测试：node-style +8（词表自洽 + countDashedBorders 四态）、
  MindmapEditor +9（弹层三行 / 三键落盘 / 各摘各的 / 清除收尾 /
  虚线导出警告 + 实线不多话 / undo 重塞）。
