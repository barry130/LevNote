# ADR 0061 · 导图关系线与概括括号 —— 内核原生全链路的验证与钉死

- 日期：2026-11-07
- 状态：已采纳
- 关联：ADR 0007（适配层）、0049（语言包与菜单）、0050（导出落差）、0056（`linkDiv` 兜底通道）、0058（operation 主通道）

## 背景

ROADMAP 候选清单把「关系线 / 概括括号」列为 XMind 对标缺口，理由是
「`arrows` / `summaries` 已透传零 UI」。本轮按惯例先探针内核
（mind-elixir 5.15.1，`dist/MindElixir.js`），逐段核实后发现
**这个缺口不存在**——两个能力的全链路是内核原生的，且早已通：

1. **创建 UI 在右键菜单里，且全开**（`Xn`，:1382-1448）：
   - 「连线」「双向连线」受 `t.link` 开关（:1396
     `t.link && (g.appendChild(m), g.appendChild(x))`）——
     `MindmapEditor` 传的就是 `link: true` ⇒ 两项都渲染；
   - 「概括」**无条件挂**（:1396 `g.appendChild(y)`）——没有开关，一直在；
   - 点「连线」→ `D()` 挂 `clickTips` 提示（语言包的「请点击目标节点」）→
     `map` 上**一次性** click 监听 → 点击的元素是 `me-tpc` 的祖先才
     `createArrow(from, target, { bidirectional? })`（:1433-1448）；
     点空处提示气泡摘掉、箭头不建（`once: true` 的监听自然消费）。
2. **label 编辑无原生对话框**（:1310-1330）：就地 contenteditable 气泡，
   Enter / Tab 收尾、blur 提交、**空值回退原 label**，非空才 fire
   `finishEditArrowLabel` / `finishEditSummary`——Tauri WebView 里
   `prompt()` 缺失的那族问题在这里不存在。
3. **删除走 Delete 键**（`In`，:899-906）：`currentArrow → removeArrow()`、
   `currentSummary → removeSummary(id)`，与节点删除同一条键路；
   菜单里没有「删除关系线」项（点选后 Delete，XMind 同款）。
4. **operation 事件齐全**（`Fn`，:1454-1468）：
   `createArrow` / `removeArrow` / `finishEditArrowLabel` / `reshapeArrow` →
   `type: "arrow"`；`createSummary` / `removeSummary` / `finishEditSummary` →
   `type: "summary"`——走既有 emit 主通道，标脏 / 保存照常，
   与 0053 起的「内核自有 UI 不单独伺候」同一条纪律。
5. **落盘现读**：`getData()` 返回
   `{ nodeData, arrows, summaries, direction, theme, compact, meta }`
   ——`arrows` / `summaries` **在返回里**（不是「写死消失的自定义顶层键」）。
   适配层的顶层 extra 双向直通（`toKernelData` 回填 `adapter.ts:198`、
   `fromKernelData` 进 extra `adapter.ts:275-276`）从 0007 时代就钉着测试。
6. **重画随布局**：`linkDiv` 每次调用先清 label 容器再
   `this.renderArrow()` + `this.renderSummary()`，**然后才** fire `"linkDiv"`
   （:1353）——节点拖动 / 增删 / undo 之后箭头与括号全量重画
   （锚点用相对 delta 重算，:2270-2274），不会悬空。
7. **导出已带**：`exportSvg`（`zo`）克隆 `.topiclinks` 与 `.summary`
   （:2677-2680）——关系线与概括括号都在导出的图里；
   0050「编辑器画得出、导出没有」的落差在这两个能力上**不存在**。

**「零 UI」的旧记录错在哪**：0055 时代写过「内核原生渲染但零 UI」，
那说的是 `hyperLink` / `tags`（锚点与 chip 内核渲染、但没有任何编辑入口，
菜单里的 `link` 项其实是关系线）。把同族印象套到 `arrows` / `summaries`
上就错了——它们有完整的右键菜单入口、编辑气泡、删除键路。
本轮把它当候选缺口排队，是没先探针就写结论的错。

## 决策

1. **不新建 UI、不动生产代码**：能力全链路原生，重复造 UI 是 0053 时代
   否决过的「fork 内核行为」。本轮的全部产出是**验证与登记**。
2. **用集成测试把链路钉死**（`MindmapEditor.test.tsx` 新增
   「关系线与概括括号（ADR 0061）」describe）：菜单三项渲染、
   创建链路（菜单 → 提示 → 点目标 → operation → onChange →
   落盘 + SVG 元素在）、round-trip 重开回填、undo 回滚、
   Delete 删除、`linkDiv` 重画后箭头仍在、只读菜单关门。
   这条链此前**没有任何测试**——内核升级断了是静默的。
3. **登记已知边界**（不改，如实记）：
   - arrow 默认 label 是内核写死的英文 `"Custom Link"`（:2295）——
     用户不编辑 label 时线上就挂着这五个词。不替内核发明默认值
     （改了升级就丢），真机冒烟人工看一眼。
   - 概括括号吃 `currentNodes`（兄弟段），单选点「概括」内核按
     `start === end` 处理——行为如实测、如实记。
   - 删除关系线 / 概括**只有 Delete 键一条路**，菜单无删除项。
4. **白板与文档修正**：「arrows / summaries 已透传零 UI」的旧结论
   随本轮推翻——白板候选清单同步改写。

## 后果

- 正面：XMind 对标清单上「关系线」「概括括号」两项**关闭**
  （原生已覆盖 + 集成测试钉死）；内核升级弄断这条链会被测试当场抓住。
- 成本：测试 +N 条，换一条「内核行为如我们所信」的验证。
- 遗留：arrow 的 `style` 子键（颜色 / 线型）内核按值渲染但无 UI，
  照 §5 原样保留不伺候；概括括号同。后续 XMind 对标轮次若做
  「关系线样式面板」，属新 ADR。
