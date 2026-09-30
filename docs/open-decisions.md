# 待验证条件与过程中被推迟的项

状态：**核心决策已定**（见 `decisions/0005`–`0016`）。本文不再是「待决事项对比」，改为跟踪
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
| 错误收敛（`asAppError` / `AppError`） | `src/core/document/types.ts` | `types.test.ts`（9 条，含「抛字符串 / `null` / 对象也不能让它自己崩」） |
| 文本统计（字数 / 字符数 / 阅读时长） | `src/core/statistics.ts` | `statistics.test.ts`（20 条，按 Word 口径逐条钉住）；状态栏接线由 `App.test.tsx` 的「状态栏统计」（3 条）覆盖 |
| 源码模式切换（Ctrl+/） | `src/core/editor/types.ts`（`sourceView`）+ `core/document/store.ts`（`ViewMode`）+ `ui/` | `store.test.ts` 的「视图模式」（7 条）+ `modules/wiring.test.ts`（5 条跨模块名单一致性）+ `App.test.tsx` 的「源码模式切换」（8 条内核交棒）。决策见 [ADR 0014](./decisions/0014-markdown-source-view.md) |
| 选中浮层工具条（bubble menu） | `src/core/editor/types.ts`（`onSelectionChange` + `formatting`）+ `modules/markdown/selection.ts`（格式名翻译）+ `ui/SelectionToolbar.tsx` | `modules/markdown/formatting.test.ts`（13 条，走真实内核、断言序列化出来的 markdown）+ `App.test.tsx` 的「选中浮层工具条」（9 条，用真实 DOM 选区驱动）。决策见 [ADR 0015](./decisions/0015-selection-toolbar.md) |
| 斜杠菜单（`/` 插入块） | `src/core/editor/types.ts`（`onSlashChange` + `insertBlock`）+ `modules/markdown/selection.ts`（`describeSlash`）+ `ui/SlashMenu.tsx` | `modules/markdown/slash.test.ts`（18 条，**逐字真的打字**驱动，覆盖六种触发条件 + 九个可插入块）+ `App.test.tsx` 的「斜杠菜单」（8 条，覆盖筛选 / 回车 / 上下键 / Esc / 无匹配 / 鼠标 / 切源码视图）。决策见 [ADR 0016](./decisions/0016-slash-menu.md) |
| 表格增强 + 代码块（语言选择器 / 行号） | `src/core/editor/types.ts`（`onTableChange` / `onCodeBlockChange` + `EditorTable` / `EditorCodeBlock`）+ `modules/markdown/table.ts`、`code-block-view.ts`（NodeView）+ `ui/TableToolbar.tsx`、`CodeBlockPicker.tsx` | `modules/markdown/table.test.ts`（27 条）+ `code-block.test.ts`（29 条）+ `App.test.tsx` 的「表格工具条」（12 条）与「代码块语言选择器」（14 条）+ `ui/floating.test.ts`（12 条，**新建** —— 浮层算术提取成纯函数后第一次有测试）。**两半都自研** —— 官方组件是 Vue 3 的，理由与重审条件见 [ADR 0017](./decisions/0017-m5-tables-and-code-blocks.md) |
| 图片：粘贴落盘 + 显示 + 灯箱 | `src/core/document/images.ts`（**新建**，两个无状态的文档级动作）+ `core/document/assets.ts`（目录与命名规则）+ `core/image.ts`（MIME ↔ 扩展名、字节 → `data:` URI）+ `modules/markdown/image-view.ts`（NodeView）+ `ui/Lightbox.tsx`（**新建**）+ `src-tauri/src/fs/mod.rs` 的 `write_binary_file` | `core/document/images.test.ts`（18 条，**新建**：六种读取降级、撞名重试、非 CONFLICT 不重试、写入被截断要报出来）+ `core/image.test.ts`（29 条）+ `core/document/assets.test.ts`（38 条）+ `modules/markdown/image.test.ts`（16 条，内核半边）+ `App.test.tsx` 的「图片」（10 条，**跨四层的端到端**）+ `core/path.test.ts` +4（含「写读路径逐字符相同」）+ Rust 侧 4 条（`percent_decode`）。决策见 [ADR 0018](./decisions/0018-m6-images.md) |
| 位置记忆 + 设置页 | `src/core/positions.ts`（**新建**：独立存储键、逐字段校验、LRU 200 条）+ `core/editor/position.ts`（**新建**：滚动比例算术，三个内核共用）+ `core/editor/types.ts`（`EditorPosition` + `getPosition` / `setPosition` + `onPositionChange`）+ `ui/SettingsDialog.tsx`（**新建**）+ `vite.config.ts`（`define` 注入版本号）+ 三个内核各自实现位置读写 | `core/positions.test.ts`（21 条，**新建**：坏条目整条丢、LRU 淘汰、存储抛错不炸）+ `core/editor/position.test.ts`（10 条，**新建**：除零、`NaN`、往返一致）+ `core/settings.test.ts` +3 + `App.test.tsx` 的「设置页」（5 条）与「位置记忆」（4 条，**跨四层的端到端**）。决策见 [ADR 0021](./decisions/0021-m9-position-memory-and-settings.md) |

合计 **880 条前端测试 + 15 条 Rust 测试**（P2 结束时是 162 条），
`tsc` / `eslint` / `vite build` / `cargo test` 全绿。

> 这张表在 M7 / M8 那两批**漏更了**（表停在 M6），所以「合计」这一行直到 M9
> 才跟着对上一次。现在的 880 是 `vitest run` 的实测值，不是累加出来的。

> 其中 `eslint` 的「全绿」是 **2026-09-29 才真正成立**的：此前它虽然被写在这一行，
> 但实际上是非阻塞的、且还有 33 项欠账。现在它是阻塞门禁，且配置里没有关掉的规则。
> 明细见 ADR 0013 末尾的表。

**本阶段修掉的三处真实缺陷**（不是重构，是 bug）：

1. `core/editor/registry.ts` 的 `registerAdapter` 用 `unshift` 插入，
   实际优先级与注释**相反**。只有一个内核时看不出来；接上第二个认得 `.md` 的内核后，
   备胎会抢在正主前面把所有 `.md` 吃掉。已改为「原地替换 + push」并补了说明。
2. **查找替换在 Markdown 内核上会静默丢数据**：替换不经过键盘事件，
   原先的「变更上报」判定把它当成程序写入而不上报，外壳持有的内容停留在替换前，
   用户按保存时写回磁盘的是旧内容。已加 `USER_EDIT_META`。
3. **图片的写入路径与回读路径不是同一个字符串**（M6 暴露）：
   Markdown 里的地址必须是正斜杠，Windows 上的文档路径是反斜杠 ——
   `joinPath` 只拼接、不统一，于是写入时算出 `…\a.assets\image-x.png`、
   回读时算出 `…\a.assets/image-x.png`。文件系统两种都认，所以**只在按字符串
   比较的地方出问题**（缓存键、去重、相等判断），表现为「图写进去了但显示不出来」。
   已让 `joinPath` 统一分隔符风格（只换风格，不折叠重复分隔符、不解析 `..`），
   并补了一条「写读路径逐字符相同」的用例。

   > 这一条**在单元测试里看不到**：内核那一层拿到的「读文件」是测试自己给的假函数，
   > 返回什么都是测试说了算。只有端到端（真写盘 → 真回读）才把它照出来 ——
   > 这也是「接缝必须单独测」的一个具体例子。

**P3 新发现的推迟项**：

| 事项 | 为什么推迟 | 风险 / 何时该做 |
|---|---|---|
| **切标签页会重建编辑器内核** | 正确性优先：让内核跟着标签页走比在同一个面板里换内核安全得多（ADR 0010） | 中。代价是撤销栈、光标、滚动位置在切换时丢失。改法（全部挂载 + CSS 隐藏）需要先实测内存占用，尤其是一份 10 万字文档 |
| **切视图也会重建内核** | 同一个理由（ADR 0014）。两种视图共用 `doc.text` 一份权威副本，所以**内容**无损；丢的是撤销栈、光标与滚动位置 | 中。horseMD 明确把「切源码模式保留滚动位置」当卖点，所以这一条迟早要补。难点是**坐标翻译**：源码偏移 → ProseMirror 位置需要序列化器的逆映射，而它一旦不精确就会「跳到错的地方」——按本项目一贯的取舍（跳错比不跳更伤人），**宁可不做也不做错**。廉价的近似（按滚动比例）可以后续单独评估 |
| **i18n** | 现在只有中文，做 i18n 骨架需要先确定文案的组织方式 | **中高。这件事越晚做越贵** —— 每加一个功能就多一批散落的字符串。建议在 P3-B 的第 17 项做「骨架」而不是「翻译」 |
| **选中浮层点到编辑器之外不会自己收起来** | 刻意为之：点浮层按钮必然先让编辑器失焦，一失焦就藏，按钮会在 `click` 到达之前消失（ADR 0015）。**不做，而不是没做** | 低。只会让浮层多留一会儿，不涉及数据。改法是监听 `.editor-area` 的 `blur` 并在 `relatedTarget` 落在浮层内部时放过 —— 但这个判断在 jsdom 里验证不了，容易写成「真机上偶发点不动」，所以等有人真的抱怨再说 |
| **浮层的屏幕几何只有人眼能验** | `coordsAtPos()` 依赖布局引擎，jsdom 返回全 0（`src/test/jsdom-gaps.ts` 只能补成空矩形）。测试钉住的是「空选区返回 null」「不抛错」 | 低（但确实没验）。要在 `npm run desktop` 里目视确认：浮层摆在哪、会不会压住文字、贴边时有没有翻到下方。与 V3 同类 —— 测试环境的结论不能替代真机结论 |
| **空段落套进容器后会序列化成 `<br />`** | Milkdown 的 `remarkPreserveEmptyLinePlugin` 把空段落序列化成一个 `<br />` 的 html 节点当**空行占位符**：mdast 里空段落序列化出来什么都没有，没有占位符的话「一个空引用」存盘再打开就消失了。**往返稳定**（`> <br />` → `> <br />`），只影响「插入了但一个字没打就保存」，打字（`> 甲`）之后占位符就没了 | 低。**不是 M4 引入的** —— 手敲 `> ` 触发输入规则就能造出同样产物。两条改法都有真实损失（关掉插件 = 放弃空行保留；只省掉纯占位符容器 = 空引用存盘后消失），所以现在不动。详见 [ADR 0016](./decisions/0016-slash-menu.md) 代价 2 |
| **主包体积**（886 kB → 现在更大） | 桌面端从本地磁盘加载，与网页端性质不同 | 低。拆分方案已明确：`React.lazy` 加载内核，外壳首屏目标 < 200 kB |
| **命令面板的拼音关键词要手工维护** | 每条命令手写 `keywords` | 低。命令数量还少（15 条）。超过 ~50 条时应改为自动生成拼音 |
| **代码块语言的中文名 / 拼音关键词是手工的** | 候选列表来自 `supportedLanguages()`（直接读 highlight.js 的 `listLanguages()`，不会过期），但**中文名与拼音是手写的一张表**。新增语言只会显示成英文 id，**不会坏**，只是不友好 | 低。这张表的规模由 `highlight.js/lib/common` 的 36 种语言决定，是**固定集合**，不随我们的代码增长。真需要时按语言 id 的常见程度补 |
| **代码块语言面板的屏幕几何只有人眼能验** | 同 M3 / M4：`getBoundingClientRect()` 在 jsdom 里全是 0 | 低（但确实没验）。面板要**右对齐到语言标签的右边缘**（`horizontal: 'end'`），贴右边缘时会被夹回视口 —— 夹取逻辑有单测，观感要在 `npm run desktop` 里看 |
| **图片没有尺寸调整与右键菜单** | 拖动改宽高、`width` 属性、复制路径、在文件管理器里打开 —— 这些都是纯交互层的增量，不需要新的内核能力。M6 先把「能粘进来、存得住、看得见」这条主干打通 | 低。等有人真的需要在文档里控制图片大小时再做。难点在「改了尺寸之后怎么存」—— 要么写进 Markdown 的 HTML 形式（会牺牲可移植性），要么只在预览里生效（那就不算功能） |
| **图片的显示观感与灯箱只有人眼能验** | 同 M3–M5：jsdom 没有布局引擎，也不解码图片 | 低（但确实没验）。要在 `npm run desktop` 里确认：占位的虚线框大小、加载中那块底色的位置、灯箱在大图 / 长图上的表现、深底与浅色主题的对比度 |
| **图片走 `data:` URI 而非 `asset:` 协议** | `asset:` 要打开 `app.security.assetProtocol` 并配一份路径白名单 —— 那是一块需要长期维护的安全面。`data:` 走已有的 `read_binary_file`，权限收口在同一个地方。代价（base64 大 1/3、常驻 DOM）由上游 24 MB 的单文件上限兜住 | 低到中。**重审条件**：一份文档里几十张大图、打开明显变慢时，重新评估 —— 届时要一并给出白名单方案，而不是只把开关打开。见 [ADR 0018](./decisions/0018-m6-images.md) |
| **显示图片不做缓存** | 缓存对象是 `data:` URI（1 MB 的图 → 约 1.4 MB 字符串），**常驻且无上界**。一份三十张截图的文档就能吃掉几十 MB。不缓存的代价有限：NodeView 只在节点创建时解析一次，真正会重读的只有「切换视图模式」 | 低。**重审条件**：实测切视图出现可感知卡顿时，加一个**按字节总量**（而不是条目数）设上界的缓存 |
| **设置页的版式与长列表滚动只有人眼能验** | 同 M3–M6：jsdom 没有布局引擎，`getBoundingClientRect()` 全是 0，也**没有滚动**。测试钉住的是「工具栏能开、Esc / 点遮罩能关、点内部不关、快捷键表由注册表派生、改主题立即生效且落盘、版本号来自构建期常量」——全是**行为**，不是**观感** | 低（但确实没验）。要在 `npm run desktop` 里目视确认：四段分组的分隔是否清楚、开关的选中态在深浅两套主题下都看得见、快捷键表在窗口变窄时会不会溢出、以及**清除位置记忆**那个按钮的破坏性提示是否足够显眼。见 [ADR 0021](./decisions/0021-m9-position-memory-and-settings.md) |
| **位置记忆的实际手感只有人眼能验** | 位置（光标偏移 + 滚动比例）的**往返**在单测里钉得很死（三个内核各一条 + `position.test.ts` 10 条 + `positions.test.ts` 21 条），但「**回到哪儿才算对**」是个观感问题：`textLength` 对不上时只恢复滚动、不对齐光标，这在实际文档上落点是否符合直觉，只有真机能判断 | 低到中。要在 `npm run desktop` 里确认三件事：① 长文档读到一半切走再切回，落点是否就是刚才那一段；② 在**源码模式**与富文本之间来回切，滚动比例是否稳定（两边 `textLength` 口径不同，只在同一视图内保证）；③ 编辑完立刻关窗（`beforeunload` 兜底那一次写盘）再打开，位置有没有丢。与 V3 同类 —— **测试环境的结论不能替代真机结论** |

---

## 零、P2 执行状态（2026-09-28）

**已完成**（ADR 0008 划定的 P2 范围）：

| 交付 | 位置 | 验证方式 |
|---|---|---|
| 编辑器适配层契约 | `src/core/editor/` | `registry.test.ts`（9 条：兜底、`canHandle` 否决与抛错、优先级） |
| Markdown 内核（Milkdown 7.22.2） | `src/modules/markdown/` | `markdown-core.test.ts`（46 条广谱语料 + 规范化行为） |
| 自包含 HTML 导出 | `src/core/export/` | `inline-html.test.ts`（16 条）+ `App.test.tsx` 端到端 |
| 外壳接线与导出入口 | `src/ui/App.tsx` | `App.test.tsx`（38 条，含 StrictMode 双调用、源码模式、选中浮层工具条） |
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
| V4 | Milkdown 表格可视化编辑的成本 | 达到「像 Typora 一样编表格」需要多少人日 | 明显超出预算 → 评估改用 `@milkdown/crepe`（开箱含块手柄与表格 UI），代价是控制力下降 | P2 收尾 | ✅ **已由 M5 回答**，见下 |

**V1 与 V2 是各自阶段的开工前置**，应在动手写代码之前先做，避免为可能被否的方案投入沉没成本。

**V3 是唯一一项「推迟了但会反噬」的验证** —— 它的结论要回填 ADR 0004，
而 0004 的待定状态会一直挂着。V3 无法在 jsdom 里做（jsdom 的渲染耗时与真实 WebView 无关），
必须先用 `npm run desktop` 跑起来、拿一份 1 万字以上的真实文档手动测。

### V4 的结论（2026-09-29，M5 落地时得出）

**结论：成本可接受，自研；不评估 Crepe。**

M5 已经把表格的可视化编辑做出来了（加行加列、删行删列、设对齐、插入最小表格），
实现是「一个纯函数 `describeTable()` + 一个 React 工具条 + 装饰层」，
**没有引入任何新依赖**。也就是说 V4 原本担心的「成本明显超出预算」并没有发生 ——
真正的成本不在 UI，而在**读懂 Milkdown 的表格 schema**（见下）。

同时，「改用 `@milkdown/crepe`」这个备选分支**被 ADR 0017 关闭了**：
Crepe 开箱的表格 UI 正是建立在 `@milkdown/components` 上的，而那是 Vue 3。
采纳它等于采纳 ADR 0017 已经否掉的那条路（第二套 UI 框架 + 高亮管线分叉）。

三条在实现过程中**实测**到的、读文档读不出来的约束（已写进 ADR 0017）：

1. 最小表格是 **2 行**（表头行 + 至少一个正文行）；**空表头行会在序列化时被静默丢掉**；
2. 对齐是**按列、且由表头决定**的 —— 设对齐必须**写整列**，只写光标所在格会在存盘时丢；
3. `TableRole` 里**没有 `header_row`**，表头行与正文行共用 `row`。

**仍未做、且属于 V4 范围的剩余部分**：鼠标拖拽改列宽、表格悬浮 `+` 按钮、
单元格内快捷导航（Tab 跳格是 ProseMirror 自带，跨行插入未做）。
这些**不需要新的内核能力**，是纯交互层的增量，可以随时补 —— 所以不再挂着一个待验证项。

---

## 二、为收窄范围而推迟的功能（ADR 0008）

这些**没有被删掉，只是排到 P2 之后**。每项都必须有明确的落点，否则会无声消失。

| 功能 | 原计划 | 现落点 | 风险 |
|---|---|---|---|
| ~~Mermaid 渲染~~ | P2 | ✅ **已补**（M8，ADR 0020） | — |
| 大纲面板 / 标题跳转 | P2 | P2 之后 | 中（长文档实际必需，可能需提回 P2） |
| 专注模式 / 打字机模式 | P2 | P2 之后 | 低（体验项，非能力项） |
| PDF 导出 | P2 | P2 之后（与自包含导出共用渲染结果，紧接其后） | 低 |
| ~~数学公式渲染~~ | P2 | ✅ **已补**（M7，ADR 0019） | — |

**关于推迟项的保真度**：这张表里剩下的三项（大纲面板已有、专注模式、PDF 导出）
都还是「不做也不能丢」的状态。底线要求是：**不处理可以，丢掉不行。**

> 这条底线原本也管着数学公式与 Mermaid。现在两个都已经**真正渲染**了：
> 公式是 M7（ADR 0019，导出内联字体保证断网也有字形），
> 图表是 M8（ADR 0020，导出画成 SVG，不依赖 mermaid 本身）。
> 于是这张表里**已经没有「内容不丢但没渲染」的项**。

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
| 多标签工作区 | 内核跟着标签页走（切换时重建）；外壳不持有任何文档字段 | [0010](./decisions/0010-multi-tab-workspace.md) |
| 代码内核与降级 | CodeMirror 6；「降级到次优」而不是「降级到最差」 | [0011](./decisions/0011-code-kernel-and-graceful-downgrade.md) |
| 命令注册表 | 命令只有一个来源；面板由注册表生成，不会过期 | [0012](./decisions/0012-command-registry.md) |
| 双编译器（TS 6 + TS 7） | lint 跑 TS 6、`tsc` 跑 TS 7；**两条都要跑** | [0013](./decisions/0013-lint-alongside-typescript-7.md) |
| Markdown 源码视图 | 复用代码内核，不注册进注册表；`ViewMode` 住在 `StoreState` | [0014](./decisions/0014-markdown-source-view.md) |
| 选中浮层工具条 | 内核只上报「选区是什么格式、在屏幕哪儿」，浮层由外壳画 | [0015](./decisions/0015-selection-toolbar.md) |
| 斜杠菜单 | 触发判定留在内核、菜单与键盘在外壳；**Esc 只收菜单不删字** | [0016](./decisions/0016-slash-menu.md) |
| M5 表格与代码块 | **自研两半，不用官方 Vue 版组件**；表格走装饰、代码块走 NodeView | [0017](./decisions/0017-m5-tables-and-code-blocks.md) |
| M6 图片 | 落 `<文档名>.assets/`、文档里写相对路径、显示走 `data:` URI（不开 `asset:` 协议）、灯箱自研 | [0018](./decisions/0018-m6-images.md) |
| M9 位置记忆 + 设置页 | 位置的 `offset` **由内核定义、外壳只往返**；`textLength` 作一致性凭据（不符则只恢复滚动）；滚动读写进契约（容器归内核）；位置单独一个存储键 + LRU 200 条；切文档立刻落盘；设置页的快捷键表与版本号都从唯一来源派生 | [0021](./decisions/0021-m9-position-memory-and-settings.md) |

---

## 六、构建期踩过的坑（省下下次的半小时）

| 症状 | 原因 | 正确做法 |
|---|---|---|
| `cargo check` 通过但 `cargo test` 报 `LNK1181: 无法打开输入文件 …\out\resource.lib` | 手删过 `target/debug/build/qingmo-*/out`。那里除了 codegen 资产还有 `tauri-build` 生成的 `resource.lib`（图标与 manifest）。`cargo check` 不链接所以看不出来 | 用 `cargo clean -p qingmo`（793 文件 / 2.0 GiB，但在可接受范围内），**不要手删 cargo 的输出目录** |
| `cargo check` 报 `failed to write asset … 拒绝访问 (os error 5)` | 上一轮 codegen 的产物被占用（编辑器 / 杀软 / 残留句柄） | 同上，`cargo clean -p qingmo` |
| vitest 里 `import css from './x.css?raw'` 拿到空字符串，测试假绿 | vitest 默认 `css: false`，CSS 模块被替换成空实现 | `vite.config.ts` 的 `test` 里显式开 `css: true`。不开的话「导出的版式 CSS 是否真的内联」这类断言会永远通过 |

