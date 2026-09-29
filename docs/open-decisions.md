# 待验证条件与过程中被推迟的项

状态：**核心决策已定**（见 `decisions/0005`–`0012`，共八条）。本文不再是「待决事项对比」，改为跟踪
**两件事**：① 已决策事项的出口条件（什么条件下要推翻它）；② 为收窄范围而推迟的功能项。

决策的理由与对比已归档进 ADR，不要在这里重复论证。

---

## 零之前、P3 执行状态（2026-09-29）

**本阶段的主题是「工作区化」**：从「单文档编辑器」变成「一个窗口装下一个文件夹」。

| 交付 | 位置 | 验证方式 |
|---|---|---|
| 多标签工作区 | `src/core/workspace/store.ts` | `store.test.ts`（19 条：去重打开、关闭接管邻居、重排、脏状态互不串台、快照引用稳定） |
| 文件树 | `src/core/workspace/tree.ts` + `ui/Sidebar.tsx` | `tree.test.ts`（15 条：排序、重目录折叠、懒加载、祖先链） |
| 代码内核（CodeMirror 6） | `src/modules/code/` | 由 `App.test.tsx` 的「front matter 降级到代码内核」与 `.cm-content` 断言覆盖 |
| 命令 / 快捷键 / 命令面板 | `src/core/commands/` | `keybinding.test.ts`（18 条）+ `registry.test.ts`（11 条）+ `fuzzy.test.ts`（11 条） |
| 查找替换 | `src/core/search/find.ts` | `find.test.ts`（17 条，含零宽匹配不死循环、非法正则不抛错） |
| 大纲 | `src/core/outline/outline.ts` | `outline.test.ts`（16 条，含围栏代码块与 front matter 里的「假标题」） |
| 主题 / 偏好 | `src/core/theme/theme.ts` + `src/core/settings.ts` | `theme.test.ts`（7 条）+ `settings.test.ts`（8 条） |
| 导出动作抽出 | `src/core/export/export-document.ts` | `export-document.test.ts`（5 条） |

合计 **307 条前端测试**（P2 结束时是 162 条），`tsc` / `eslint` / `vite build` 全绿。

**本阶段修掉的两处真实缺陷**（不是重构，是 bug）：

1. `core/editor/registry.ts` 的 `registerAdapter` 用 `unshift` 插入，
   实际优先级与注释**相反**。只有一个内核时看不出来；接上第二个认得 `.md` 的内核后，
   备胎会抢在正主前面把所有 `.md` 吃掉。已改为「原地替换 + push」并补了说明。
2. **查找替换在 Markdown 内核上会静默丢数据**：替换不经过键盘事件，
   原先的「变更上报」判定把它当成程序写入而不上报，外壳持有的内容停留在替换前，
   用户按保存时写回磁盘的是旧内容。已加 `USER_EDIT_META`。

**P3 新发现的推迟项**：

| 事项 | 为什么推迟 | 风险 / 何时该做 |
|---|---|---|
| **切标签页会重建编辑器内核** | 正确性优先：让内核跟着标签页走比在同一个面板里换内核安全得多（ADR 0010） | 中。代价是撤销栈、光标、滚动位置在切换时丢失。改法（全部挂载 + CSS 隐藏）需要先实测内存占用，尤其是一份 10 万字文档 |
| **i18n** | 现在只有中文，做 i18n 骨架需要先确定文案的组织方式 | **中高。这件事越晚做越贵** —— 每加一个功能就多一批散落的字符串。建议在 P3-B 的第 17 项做「骨架」而不是「翻译」 |
| **主包体积**（886 kB → 现在更大） | 桌面端从本地磁盘加载，与网页端性质不同 | 低。拆分方案已明确：`React.lazy` 加载内核，外壳首屏目标 < 200 kB |
| **命令面板的拼音关键词要手工维护** | 每条命令手写 `keywords` | 低。命令数量还少（15 条）。超过 ~50 条时应改为自动生成拼音 |

---

## 零、P2 执行状态（2026-09-28）

**已完成**（ADR 0008 划定的 P2 范围）：

| 交付 | 位置 | 验证方式 |
|---|---|---|
| 编辑器适配层契约 | `src/core/editor/` | `registry.test.ts`（9 条：兜底、`canHandle` 否决与抛错、优先级） |
| Markdown 内核（Milkdown 7.22.2） | `src/modules/markdown/` | `markdown-core.test.ts`（46 条广谱语料 + 规范化行为） |
| 自包含 HTML 导出 | `src/core/export/` | `inline-html.test.ts`（16 条）+ `App.test.tsx` 端到端 |
| 外壳接线与导出入口 | `src/ui/App.tsx` | `App.test.tsx`（18 条，含 StrictMode 双调用） |
| Rust 二进制读取命令 | `src-tauri/src/fs/mod.rs` | `cargo test`（11 条） |

合计 **162 条前端测试 + 11 条 Rust 测试**，`tsc` / `vite build` / `cargo check` 全绿。

**P2 中改了主意或发现问题的三处**，均已落成 ADR 或代码注释：

1. 不用 `@milkdown/plugin-listener` —— 它的 `markdownUpdated` 是防抖 200ms 触发的，
   恰好让「这次变更是程序写入的」判断在最需要它的时刻失效。改为自研上报插件
   （`plugin.ts` 的 `markdownEmitPlugin`，判据是「用户交互之后的变更才算用户改动」）。
2. 移除两个从未被导入的死依赖：`katex`、`@milkdown/react`。后者的移除与
   `setup.ts` 里「不用 React 封装层」的既定选择一致 —— 依赖表里的它本来就是个矛盾。
3. 打开路径上的两个静默毁文件问题，见 [ADR 0009](./decisions/0009-markdown-open-guards.md)。

**尚未做、且必须由人来做的一件事**：**真实浏览器里的视觉确认**。
本机没有可用的无头浏览器（装一个要下载约 500 MB），所以以下两点只有逻辑保证、
没有眼见为实：导出 HTML 的实际观感、编辑器在真实 WebView 里的排版。
请跑 `npm run dev`（浏览器预览）或 `npm run desktop`（真桌面版）目视一遍。

**其中「导出观感」这一半已经被降到只需双击**：`docs/samples/` 下有两份**用真实管线**
（内核 `toHtml()` → `inlineAssets` → `buildSelfContainedHtml`）生成的产物，不启动应用即可打开：

| 文件 | 用途 |
|---|---|
| `export-sample.html` | 富文本导出：标题/行内标记/引用（含嵌套）/列表/任务列表/表格/两种代码块/图片/分隔线 |
| `export-sample-plaintext.html` | 纯文本兜底导出（`bodyHtml === null`），应显示为等宽 `<pre>` 而非伪造的富文本 |

它的验收价值在于**断网可读**：两份文件里没有任何 `http(s)` 资源引用，
两张图中「存在的」被内联成 `data:image/png;base64,…`、「不存在的」被标成
`data-qm-missing` 并由 CSS 画出可见占位。生成时实测：1 张内联 + 1 张标注缺失、
2 个代码块全部着色（77 个 `hljs-*` span）、无外部资源、156 KB。

这两份是**一次性的样例快照**，不随代码演进自动更新（生成脚本跑完即删，避免在
测试套件里留下一个「跑测试就写文件」的副作用）。需要重新生成时，照
`App.tsx` 的 `exportHtml` 走一遍即可。

**仍然只有人能做的**：编辑器在真实 WebView 里的排版观感，以及 V3 的延迟实测。

---

## 一、待执行的验证（有明确判据，可直接开工）

| # | 验证 | 判据 | 命中后怎么办 | 阻塞谁 | 状态 |
|---|---|---|---|---|---|
| V1 | draw.io `viewer.min.js` 能否**独立**渲染 | 能否独立运行（不依赖完整 bundle）；能否正确渲染我们样本里的**多页**与**压缩变体** `.drawio`；viewer + 必要 stencils 实测体积 | 不能独立 → 退化为「连 viewer 也按需下载」；体积 > 6 MB → 重估是否值得内置 | P5 | 未开始 |
| V2 | mind-elixir 500 节点压测 | 输入到反馈 ≤ 50ms；`.mindmap` 读写与 §7.1 骨架的对齐损耗 | 卡顿或停更 → **自研**；XMind 往返成刚需 → 迁 simple-mind-map | P4 | 未开始 |
| V3 | Milkdown 长文档输入延迟 | 1 万字以上文档，输入延迟 ≤ 50ms | 超出 → 改为分栏预览方案（保守但更不像 Typora） | 回填 0004 | **仍未测**（需要真实浏览器，jsdom 的耗时没有参考价值） |
| V4 | Milkdown 表格可视化编辑的成本 | 达到「像 Typora 一样编表格」需要多少人日 | 明显超出预算 → 评估改用 `@milkdown/crepe`（开箱含块手柄与表格 UI），代价是控制力下降 | P2 收尾 | 未开始 |

**V1 与 V2 是各自阶段的开工前置**，应在动手写代码之前先做，避免为可能被否的方案投入沉没成本。

**V3 是唯一一项「推迟了但会反噬」的验证** —— 它的结论要回填 ADR 0004，
而 0004 的待定状态会一直挂着。V3 无法在 jsdom 里做（jsdom 的渲染耗时与真实 WebView 无关），
必须先用 `npm run desktop` 跑起来、拿一份 1 万字以上的真实文档手动测。

---

## 二、为收窄范围而推迟的功能（ADR 0008）

这些**没有被删掉，只是排到 P2 之后**。每项都必须有明确的落点，否则会无声消失。

| 功能 | 原计划 | 现落点 | 风险 |
|---|---|---|---|
| Mermaid 渲染 | P2 | P2 之后的独立小阶段 | 低（Markdown 生态标准能力，用户会主动要） |
| 大纲面板 / 标题跳转 | P2 | P2 之后 | 中（长文档实际必需，可能需提回 P2） |
| 专注模式 / 打字机模式 | P2 | P2 之后 | 低（体验项，非能力项） |
| PDF 导出 | P2 | P2 之后（与自包含导出共用渲染结果，紧接其后） | 低 |
| 数学公式渲染 | P2 | P2 之后 | 低 |

**关于推迟项的保真度**：Mermaid 与数学公式虽然不渲染，但**内容不会丢** ——
Mermaid 以代码块形式原样保留，公式以文本形式原样保留，两者都有测试钉住
（见 `markdown-core.test.ts` 的「内部不做处理但也不该丢的结构」）。
这一点是本项目对推迟功能的底线要求：**不处理可以，丢掉不行。**

**触发把某项提回 P2 的条件**：若「没有它就无法把成果交付给一个真实用户」，就提回来。

---

## 二之二、P2 过程中新发现的推迟项

| 事项 | 为什么推迟 | 风险 / 何时该做 |
|---|---|---|
| **front matter 的富文本编辑** | P2 里 front matter 文件走纯文本，是**有意的降级**（见 ADR 0009） | 中。Obsidian / Hugo 用户会想让这类文件也有富文本视图。做法：把 front matter 当不透明前缀，编辑区只呈现正文、保存时原样拼回。P3 候选 |
| **打开时的往返保真度自检** | 实测序列化器的规范化是真实存在的（setext → ATX、行尾空白收敛…），正常文档也会触发，运行时告警噪音太大 | 低。已改用测试保证（`markdown-core.test.ts` 的「序列化器的规范化行为」一组），并且那组测试同时是回归保护 |
| **主包体积 886 kB** | 桌面端从本地磁盘加载，与网页端性质完全不同，不值得现在拆 | 低。真要拆就 lazy-load Markdown 模块（`React.lazy` + `Suspense`），外壳首屏能降到百 kB 级 |
| **`reload()` 后光标位置** | 目前靠 ProseMirror 的事务映射，可能落到替换区域末尾而不是原处 | 低。P3 做多标签时顺手处理 |
| **图片粘贴落盘 + 相对路径管理** | 只做了「读回来内联」这一半（导出需要）；「粘贴时落盘」是编辑器功能 | 中。用户粘贴截图后需要有一个落盘位置与相对路径策略，否则导出的自包含文件里那张图永远内联不了 |

---

## 三、被阻塞的前置条件

| # | 事项 | 阻塞原因 | 影响 |
|---|---|---|---|
| B1 | 每周 macOS 渲染冒烟（ADR 0006 第 1 条） | **需要一台可用的 Mac**（或云 Mac 的预算）。目前未确认是否存在 | 0006 的核心动作无法启动；跨平台渲染差异会积压 |
| B2 | 排期收敛 | **每周实际可投入小时数、是否有人并行 —— 未确认** | 按 ADR 0008 的保底情形执行，不阻塞开发，但无法给出确定日期 |
| B3 | Apple 开发者账号（99 美元/年） | 采购周期 | P7 才用得上，但**申请有周期，应提前启动**，不占开发关键路径 |
| B4 | Windows 代码签名证书（OV / EV） | 采购周期 + 需企业主体 | 同上，P7 用，应提前启动 |

B3 / B4 是原规划就指出「应尽早启动申请」的事项 —— 它们不占开发工时，但会占**日历时间**。

---

## 四、用户侧待办（无法由开发完成）

- 跑一次 `npm run dev` 或 `npm run desktop`，目视确认 Markdown 编辑器与导出 HTML 的样子
  （这是 P2 唯一没有被自动化覆盖的验收项，见本文第零节）
- 中国商标网查询「轻墨」在第 9 类（软件）与第 42 类（技术服务）的可注册性
- GitHub / npm / crates.io 的名称占用检查
- 域名可用性：`qingmo.app` / `.dev` / `.io`
- 品牌 lockup 中的中文**转曲**后再对外使用（`qingmo-lockup.svg` 目前用系统字体）

---

## 五、已决策速查

| 决策 | 结论 | ADR |
|---|---|---|
| draw.io 资源路线 | 内置只读查看器（~3 MB）+ 编辑器按需下载（~60–100 MB）；完整离线包作为可选构建目标；**不自建** | [0005](./decisions/0005-drawio-integration-route.md) |
| macOS 发布时机 | 延迟签名与公证；**不延迟**每周渲染冒烟 | [0006](./decisions/0006-macos-release-timing.md) |
| 思维导图内核 | **mind-elixir**，精确锁定 `5.15.1`（npm `latest` 标签是坏的） | [0007](./decisions/0007-mindmap-core-choice.md) |
| 范围与排序 | 按「小投入」执行；**P2 = Markdown 内核 + 自包含 HTML 导出** | [0008](./decisions/0008-scope-and-order.md) |
| Markdown 打开路径 | mdast 规整 null（否则无标题图片整张丢失）+ front matter 文件降级纯文本 | [0009](./decisions/0009-markdown-open-guards.md) |

---

## 六、构建期踩过的坑（省下下次的半小时）

| 症状 | 原因 | 正确做法 |
|---|---|---|
| `cargo check` 通过但 `cargo test` 报 `LNK1181: 无法打开输入文件 …\out\resource.lib` | 手删过 `target/debug/build/qingmo-*/out`。那里除了 codegen 资产还有 `tauri-build` 生成的 `resource.lib`（图标与 manifest）。`cargo check` 不链接所以看不出来 | 用 `cargo clean -p qingmo`（793 文件 / 2.0 GiB，但在可接受范围内），**不要手删 cargo 的输出目录** |
| `cargo check` 报 `failed to write asset … 拒绝访问 (os error 5)` | 上一轮 codegen 的产物被占用（编辑器 / 杀软 / 残留句柄） | 同上，`cargo clean -p qingmo` |
| vitest 里 `import css from './x.css?raw'` 拿到空字符串，测试假绿 | vitest 默认 `css: false`，CSS 模块被替换成空实现 | `vite.config.ts` 的 `test` 里显式开 `css: true`。不开的话「导出的版式 CSS 是否真的内联」这类断言会永远通过 |

