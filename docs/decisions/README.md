# 决策记录（ADR）

每个文件记录一个有长期影响的取舍：**当时有哪些选项、为什么选它、什么条件下该重新考虑**。

最后一条最重要 —— 没有「重审条件」的决策记录，会变成后人不敢碰的祖训。

| 编号 | 决策 | 状态 |
|---|---|---|
| [0001](./0001-desktop-only-and-self-contained-export.md) | 只做桌面端，把「自包含 HTML 导出」提为一等公民 | 已采纳 |
| [0002](./0002-doc-block-as-fenced-code.md) | 用围栏代码块 `doc-block` 承载互操作，而非 YAML front-matter 或自定义标记 | 已采纳 |
| [0003](./0003-encoding-and-atomic-write.md) | 编码/行尾不静默转码，写入一律原子替换 | 已采纳 |
| [0004](./0004-editor-cores.md) | 四类编辑器内核的选型与许可边界 | 部分待定 |
| [0005](./0005-drawio-integration-route.md) | draw.io：内置只读查看器 + 编辑器按需下载，不自建 | 已采纳 |
| [0006](./0006-macos-release-timing.md) | macOS：延迟签名发布，不延迟渲染验证 | 已采纳 |
| [0007](./0007-mindmap-core-choice.md) | 思维导图内核选 mind-elixir，锁定 5.15.1 | 已采纳 |
| [0008](./0008-scope-and-order.md) | 按「小投入」执行，P2 只做 Markdown + 自包含导出 | 已采纳 |
| [0009](./0009-markdown-open-guards.md) | Markdown 打开路径的两道防线：mdast 规整 + front matter 降级 | 已采纳 |
| [0010](./0010-multi-tab-workspace.md) | 多标签工作区：**组合**多个 `DocumentStore`，不改造它 | 已采纳 |
| [0011](./0011-code-kernel-and-graceful-downgrade.md) | 代码内核用 CodeMirror 6；降级要掉到「次优」而不是「最底」 | 已采纳 |
| [0012](./0012-command-registry.md) | 命令是能力的唯一清单：注册表 + 快捷键 + 命令面板 | 已采纳 |
| [0013](./0013-lint-alongside-typescript-7.md) | 用 npm 别名让 TS6 与 TS7 并存，使 ESLint 能在 TypeScript 7 上运行 | 已采纳 |
| [0014](./0014-markdown-source-view.md) | 源码模式：源码视图是适配器属性（不注册进注册表），视图模式住 `StoreState` | 已采纳 |
| [0015](./0015-selection-toolbar.md) | 选中浮层工具条：内核只上报选区状态与屏幕位置，浮层由外壳用 React 渲染 | 已采纳 |
| [0016](./0016-slash-menu.md) | 斜杠菜单：触发判定留在内核，菜单与键盘都在外壳 | 已采纳 |
| [0017](./0017-m5-tables-and-code-blocks.md) | M5 表格与代码块：自研组件，不引入 Vue 版官方实现 | 已采纳 |
| [0018](./0018-m6-images.md) | M6 图片：`<文档名>.assets/` + `data:` URI 显示 + 自研灯箱 | 已采纳 |
| [0019](./0019-m7-math.md) | M7 数学公式：不用已废弃的官方插件，自研节点 + 输入规则 + NodeView | 已采纳 |
| [0020](./0020-m8-mermaid.md) | M8 Mermaid 图表：图与源码是同一块的两种看法，导出时画成 SVG | 已采纳 |
| [0021](./0021-m9-position-memory-and-settings.md) | M9 位置记忆 + 设置页：位置由内核往返，外壳只负责记；快捷键表与版本号都从唯一来源派生 | 已采纳 |
| [0022](./0022-vscode-shell-and-split-view.md) | VSCode 式外壳 + 视图三态：分屏 = 源码可编辑栏 + 首选内核的只读渲染，共用一份 `doc.text` | 已采纳 |
| [0023](./0023-session-restore.md) | 会话恢复：草稿在标签页激活前写进 store；打不开的文件整份清单交回外壳；丢了几条草稿必须存下来告知 | 已采纳 |
| [0024](./0024-block-handle.md) | M11 块手柄：悬停命中走 DOM 元素（不走坐标）、目标块由内核记、清目标块的时机交给外壳 | 已采纳 |
| [0025](./0025-mindmap-kernel-integration.md) | 导图内核接入：注册顺序把 `code` 压到最后；`layout` 是四个取值；「没改过就不写」的基准取内核稳定之后 | 已采纳 |
| [0026](./0026-workspace-search.md) | 工作区全文检索：Rust 流式 + 可取消 + 尊重 `.gitignore`；跳转落地在内核就绪那一刻（与位置记忆同时刻） | 已采纳 |
| [0027](./0027-drawio-viewer.md) | draw.io 只读查看器：上游产物走 `public/` 按需注入；十条资源路径是「无网络」唯一开关；缺 stencil 从 XML 里算、渲染前说清楚 | 已采纳 |
| [0028](./0028-main-bundle-split.md) | 主包拆分：内核走动态 import、注册在首帧之前完成（等待态在 main.tsx）；单个内核加载失败是降级，说出来但不挡启动 | 已采纳 |
| [0029](./0029-fs-watch.md) | 文件监听：事件是通知、mtime 才是判决；整表替换 API（幂等、崩了重发即收敛）；根目录递归 + 散文件非递归；焦点全量比对保留作兜底 | 已采纳 |
| [0030](./0030-print-pdf.md) | 打印 / 导出 PDF：与自包含 HTML 共用渲染、分道出口；HTML 走内存自定义协议（不落临时文件）；出口是系统打印对话框，不自造排版引擎 | 已采纳 |
| [0031](./0031-i18n-skeleton.md) | i18n 骨架：类型安全目录 + 查表 + 插值，回退到 key 不给空白；首批文案 = 命令面；不引入 i18n 库、语言偏好不进设置页（第二份目录落地时再接） | 已采纳 |
| [0032](./0032-custom-keybindings.md) | 自定义快捷键：录制走捕获阶段 + 规范写法；冲突不阻止、显式顶掉（置空三态）；覆盖表整表替换进偏好；设置页每行可改 | 已采纳 |
| [0033](./0033-roundtrip-fidelity.md) | 往返保真自检：诊断动作只读不写；「解析 → 序列化」由内核实现（含 trailing 复刻），外壳不另起管线；行级 LCS 比对，超限如实标注近似 | 已采纳 |
| [0034](./0034-keepalive-panes.md) | 面板常驻：访问过的标签不销毁内核（切回即离开时的样子，撤销栈保留）；先实测内存（Markdown ≈0.27 / CodeMirror ≈1.9 MB 每实例）；隐藏面板收浮层、句柄按面板分桶 | 已采纳 |
| [0035](./0035-block-interop-l3.md) | 块级互操作第一阶段：提取为引用 / 拉平引用（L3，内核 blockTransform 契约 + 外壳编排）；markdown/mindmap/text 引用画卡片、点击打开；`resolveRelative` 修路径风格边界 | 已采纳 |
| [0036](./0036-block-interop-l2.md) | 块级互操作第二阶段（L2 就地编辑 + 内嵌预览）：就地编辑 = 同面板钻取 + 返回栈（编辑走被引用文档自己的真实路径，单一编辑面不变量不破）；**嵌套内核否决**（六条理由）；预览 = 结构摘要（不渲染全文） | 已采纳 |
| [0037](./0037-menu-bar.md) | 菜单栏：自绘 web 菜单（不用原生菜单 —— enabled 是前端实时状态），菜单只是命令的另一个入口，MENUS 只存命令 id；键盘纪律沿浮层菜单旧例（window 捕获 + 拦截） | 已采纳 |
| [0038](./0038-rename-to-levnote.md) | 项目改名 轻墨 qingmo → 轻笺 LevNote：发版前一次改到底（含 localStorage 键、`.mindmap` 的 `format`、包名/标识符），不留兼容层 —— 此刻没有用户，兼容包袱留到有真实用户再评估 | 已采纳 |
| [0039](./0039-focus-and-typewriter-modes.md) | 专注 / 打字机模式：两个视图偏好**共用一个插件状态对象**；切换走只改插件状态的事务（`addToHistory: false`）**不重建内核**；装饰每次现算不走增量；打字机只写 `scrollTop` 绝不 dispatch；滚动算术只有一份；浓淡归内核样式表 | 已采纳 |
| [0040](./0040-link-activation.md) | 链接激活：`preventDefault` 无条件（编辑器里绝不放行导航），打开要 Ctrl/Cmd（保住「光标能进链接里改字」）；地址分类是 `core/` 里的纯函数（盘符先于协议、走浏览器同款归一化探测）；锚点新增可选契约 `goToAnchor`；五类去向各自一句话，成功的两支安静 | 已采纳 |
| [0041](./0041-command-and-drop-failure-exits.md) | 失败要有出口：命令的意外 throw 由 `runCommandSafely` 兜底成一句话（不再是 `console.warn`），会失败的异步能力类型不许是 `void`；拖入多文件全部打开、只汇报没打开的那几个，成功时安静 | 已采纳 |
| [0042](./0042-preference-write-failure-exit.md) | 偏好写不进去：写入一律返回 `PersistOutcome`（`no-storage` 静默 / `write-failed` 说话）；四处写偏好汇成一句横幅，不各弹各的；副作用搬出 `setSettings` 的 updater | 已采纳 |
| [0043](./0043-degradation-must-say-why.md) | 降级要说出理由：`canHandle` 之外加 `canHandleReason`（`KernelRejectReason` 是**数据**不是翻译好的话）；判定与理由走同一个 `describe…` 纯函数；`resolveAdapter` 行为不变 | 已采纳 |
| [0044](./0044-stored-data-read-failure-exit.md) | 存储里的东西读不出来：四个 `load*` 返回 `LoadResult<T>`（ADR 0042 的对偶），`no-storage` 静默 / `read-failed` / `corrupt` 说话；**只有「键在、值用不了」才算丢失**（缺键=版本升级、夹取=意图已兑现，都不报）；键名是人能读懂的路径才点名，位置封顶 3 条；与写侧横幅分两条，按丢失代价排序 | 已采纳 |
| [0045](./0045-watcher-silence-has-no-recovery.md) | 监听没起来要说出来：`WatchIssue{reason, failedPaths}` 两个槽**整份替换**（不累加、因此能自愈）；`reason` 三态（`null` 没问题 / `''` 有问题但说不出原因 / 原文）；两档措辞（整条链可能断 vs 只是几条路径没监视上）；按掉存**指纹**不存布尔；丢的是「早点知道的机会」而不是数据 —— 这类故障**不会自愈** | 已采纳 |
| [0046](./0046-issues-this-run.md) | 本次运行的问题：0042–0045 那一族当场说的话在这里**留得住**（纯函数记录本 `core/health.ts`，**只活一次运行、不落盘**）；存 key+values 不存句子（否则语言被烤在记账那一刻）；恢复显式且**永不删除**；横幅族走 `syncFamily`（有的记上、消失的标成已恢复）、一次性事件直接 `recordIssue`；内容没变就什么都不做（双挂不是又出一次问题）；写失败要加闩（位置记忆每 600ms 落一次盘）；按族分组、「发生过 n 次」只在 n>1 时出现；**复制全部**是纯本地应用唯一一条把状态交给别人的路 | 已采纳 |
| [0047](./0047-words-that-outlive-the-language-switch.md) | 切了语言，话还是旧的：ADR 0031 记下的七条「`t()` 的结果被烤进跨渲染结构」全部交回 key（文件树 / 跨文件替换 / 设置页错误 / 粘贴图片，`images.ts` 一交 key，`legacyText()` 品牌门整个删掉）；判据是「`t()` 的结果是否跨渲染存活 + 能不能不改内核契约就修」，白名单是**允许动的边界**不是清单；共享形状 `LocalizedText{key, params}`（`HealthLine` 收成别名、参数名取 `params`）；抛物原话当参数走绝不翻；句子只在一处拼——连逗号都是一个 key；`TreeProblem` 不留恒定的 `code`；block-preview 与三个 `Render.message` 出口**登记不做**（落点在内核绘制侧） | 已采纳 |
| [0048](./0048-mindmap-and-drawio-are-first-class-too.md) | 另外两类文件也是一等公民：`.mindmap` 的内核早就写好，缺的是**入口**（新增 `file.newMindmap` + 侧栏 + `file.new` 选后缀），`.drawio` 连内核都没有 —— 新增只读图纸内核（单文件打开就画，`CODE_EXTENSIONS` 里的 `.drawio` 留着当 `sourceView`，**内核契约一个字没改**）；起始内容只有一份（`core/new-file-content.ts`，模板**生成**不抄字面量，`.mindmap` 用解析器验 `canonical`，`.drawio` 用 `missingStencilLibs` 验）；三个新建入口共用一条编排；图纸内核**刻意不实现 `canHandle`**（查看器自己说「不是合法 XML」更准）**也从不调用 `onChange`**；能力缺失写在面板上而不是「没那个按钮」；**编辑位缺资产有实测读数**（`scripts/v4-run.mjs`：`Editor` 是 `mxCellEditor` 替身，跑不起编辑器），改动路线按读数重写 | 已采纳 |
| [0049](./0049-mindmap-ui-follows-the-language.md) | 内核自己画的字也得跟着语言变：右键菜单十一条是 `innerHTML` **在安装那一刻**烤死的，所以传 `true` 就只能用内核硬编码的英文包 —— 传对象才带得上自己的语言包；「切语言当场生效」因此只能走**重建内核**（`useSyncExternalStore` 订阅语言，全项目第一处）；自建 `mindmapLangPack()`（现查现变，不用内核自带 `cn`/`en`：口径统一叫「主题」，且目录只有一个来源）；重建的三条命门 —— 种子取 `latestRef`（拿 `initialValue` 会静默回滚用户的修改）、`getValue()` 记的是**交出去的那个值**不是 `lastText`（否则下一次保存把用户文件重排版，「打开→保存逐字节一致」当场破掉）、缩放还原（`scaleVal` 是公开 API；选中项/专注模式/平移**不**保，如实写出来） | 已采纳 |
| [0050](./0050-mindmap-image-export.md) | 导图画出来的样子归内核：新增可选契约 `exportImage`（`toHtml()` 对导图依旧返回 null —— 富导出的正解是「请内核把自己画一张」）；产物带 `warnings`（外链图片不自包含，内核的实话要跨层传给用户，`LocalizedText` 是载体）；底色在画图前换成**真实解析色**（内核把 `var(--surface)` 原样写进独立 SVG 会黑底，`getComputedStyle` 解析 + 整体重赋值 + finally 还原，解析不出不强编）；PNG 不重画交给内核；先问后画；导出家族统一**绝不覆盖**；两条命令按能力启用、不给快捷键 | 已采纳 |
| [0051](./0051-drawio-image-export.md) | 图纸画出来的样子也带得走：draw.io 接上 ADR 0050 的同一条路，**外壳一行没改**（能力检查认 `typeof exportImage === 'function'`，`.drawio` 文档上两条命令自动亮起 —— 契约放 `core/editor/types.ts` 的意义第一次被兑现检验）；图纸内核三个事实定形状 —— 查看器**无导出 API**（PNG 只能内核层自建 Image+canvas，失败一律 null）、底色已写进 svg 根**行内 style**（导图那套换色/还原整个不需要，PNG 画布解析出才铺、解析不出不编色）、图片形状用 `xlink:href`（序列化必须 `XMLSerializer`，`outerHTML` 不补命名空间声明，独立文件按 XML 解析报错）；导「此刻画布上的那张」（renderRef 等在途渲染，打开立刻导出不许拍半张）；警告三句随文件走（外链图片 PNG 画不出 / SVG 保留原引用、缺 stencil 库按矩形显示与屏幕一致）；ADR 0048「明确标注」在此兑现、内联未做、`toHtml()` 维持 null | 已采纳 |
| [0052](./0052-mindmap-outline-panel.md) | 导图也有大纲面板（对标 XMind 第一屏的大纲视图）：大纲从 `doc.text` 用**纯函数**算（`extractMindmapOutline` 复用 `parseMindmap`，与喂给内核的数据同一套语义 —— 自造第二份判定迟早漂移，跳转会悄悄指错地方），条目带**文件层 id**（`goToOutlineItem` 契约加可选 `id`，主题可重名、id 才指得准；Markdown 永不传，数据来源不同，缺着比编假值诚实）；折叠的子树照样列出（导航要能去到看不见的地方），内核跳转时**展开折叠祖先**而展开就是文档变更（`collapsed` 是文件字段，onChange 与标脏是诚实的）；两个如实的例外 —— **根**不展开（内核对根的折叠不做视觉隐藏、也没有根的展开器，跳过去但不假装展开过）、**只读**不展开（那是编辑，交 false；可见的照跳，选中不是编辑）；`findEle` 对看不见的节点抛错，抓住交 false 不白屏；面板按 `isMindmapPath` 分派提取器与文案（外壳知道「哪些文件是导图」是产品范围，不知道「用哪个内核」） | 已采纳 |
| [0053](./0053-mindmap-markers.md) | 导图也有标记（对标 XMind 第一屏的优先级 / 标记）：格式层把 `icons` 升格为已知字段，存**显示字形本身**（与 `topic` 存显示文本同理 —— **字形即内容**，不发明语义 id + 双向映射表：那要永久维护两张表，还与 extra 桶透传的 icons 形成双表示冲突；能排文本的内核都能显示，词表外字形照 §5 原样保留，调色盘只是词表知识、住内核模块不进格式层语义）；词表三组共 19 字形（优先级 `1️⃣`–`9️⃣` 互斥 / 颜色 `🟥🟧🟨🟩🟦🟪` 互斥 / 符号 `⭐✅❗❓` 不互斥，`applyMarkerToggle` 纯函数：同字形再点取消、互斥组先移旧再追加、词表外按非互斥）；右键「标记…」开调色盘，选中走 `reshapeNode` → `operation` 事件 → 既有 emit 主通道（标脏 / 保存不用单独伺候）；**全清过 = `[]` 一路到底**（`JSON.stringify` 丢 undefined 键，传 `{icons: undefined}` 会把「全清过」状态静默丢掉）；标记是内核自有 UI 不进 `EditorHandle`，只读时 `contextMenu: false` 自然关门；大纲条目把标记内联显示 | 已采纳 |
| [0054](./0054-mindmap-node-images.md) | 导图节点也能插图（对标 XMind 第一屏的「插入图片」，接 Markdown 已有的整套图片机制而非另造一套）：格式层把 `image` 升格为正字段，形状照内核 `{ url, width, height, fit? }` 不发明中间表示（`url` 存**文件 src**、尺寸即内容（不存它下次打开就是内核的默认猜值）；整对象照收 —— 不认识的子键原样跟着走；类型不对整值转 extra）；**显示 = 预热 + `imageProxy` 同步查表**（内核创建前逐个 `resolveImageUrl` 填表，`data:` 透传、查得到换 data:、**查不到画占位图**（内嵌 SVG 虚线框 + 🖼 + 路径如实写在里面）—— 图丢了是常见状态，内核的「缺则不画」是静默、占位才是如实；http 外链也走占位，ADR 0001 零联网不破例）；**插入 = 粘贴**（内核 paste 的官方让位口 `pasteHandler`：文本粘贴归内核，图片字节交外壳 `onImagePaste` 存盘 → 返回文件 src → 内核层 `decodeImageSize` 解码宽高 → 最长边 320 缩好 → `reshapeNode({ image })` 落既有主通道；剪贴板 `files` 与 `items` 两处都看 —— 平台填充不一致是 markdown `imageFilesOf` 的既有教训）；**移除 = 右键「图片…」弹层**（有图显示引用 + 尺寸 + 移除；无图显示「按 Ctrl+V」提示 —— 空弹层是静默）；只读不接招；外壳零改动（`resolveImageUrl` / `onImagePaste` 本就统一传给每个内核）；effect 改 `boot` 闭包（预热是异步的，销毁必须对得上「还没创建出来」的状态） | 已采纳 |
| [0055](./0055-mindmap-links-and-tags.md) | 导图主题也能挂链接与标签（对标 XMind 第一屏的最后两块；第三个候选 `note` 内核完全不渲染，刻意跳过——重审条件单列）：格式层照内核渲染合同升格 `hyperLink`（**非空字符串**，`""` 画不出来整值转 extra，原文不 trim 不解码——分类与打开归外壳 `classifyLink`）与 `tags`（**文字即内容**：`string \| { text, className?, style? }`，未知子键照收、坏形状**整条**转 extra 不逐项过滤、空数组照写=全清过）；内核对 hyperLink **零 UI**（菜单里的 `link` 项是节点连线 `createArrow`，grep 全库锚点无任何点击监听）——编辑=右键弹层（无链接时**没有**移除按钮、空输入禁保存、连挂不关弹层、重复标签 no-op、`tagAdd`/`tagRemove` 从内核现取 `nodeObj.tags` 算新组——marker 先例，不依赖弹层快照）；点击=🔗 **单击即走**（与 markdown 的 Ctrl+点击 刻意不同：显式 chip 无歧义，XMind 同款；host 捕获阶段无条件 `preventDefault`——WebView 不能整页飞走，`href` 从 `nodeObj` 读**不从 DOM 读**，`text` 读 `innerText`——内核拿它画的、jsdom 里 `textContent` 是空）；外壳 `openLink` 五出口现成零改动，**文件路径可打开工作区另一份 `.mindmap`**；激活是读不是编辑——只读也激活但 `contextMenu: false` 编辑关门；导出自带零改动（`exportSvg` 把 tags 画成文本、🔗 写成 `<a href>`——URL 本来就在文件里，是能力不是泄漏）；大纲不伺候（只取 `topic`，0052 的刻意） | 已采纳 |
| [0056](./0056-mindmap-node-notes.md) | 导图节点的备注终于看得见（0055 重审条件 5 兑现——`note` 是格式层正字段但内核**零渲染零 UI**，探针实证 `me-tpc` 里一个字都不画：本轮动的全是「看得见、改得了」）：可见性=**自挂徽章**（`refreshNoteBadges` 遍历 `me-tpc`，有非空 `note` 挂 📝 button、无则摘；同步挂 init 后 + `linkDiv` 监听两处——**内核 `undo`/`redo` 走 `refresh` 且不发 operation**，探针实证徽章被重建抹掉后由 `linkDiv` 兜底重挂，`operation` 是编辑主通道、`linkDiv` 是重建兜底两条腿缺一不可；**不 fork 内核渲染**，升级丢不得）；编辑=徽章 + 右键「备注…」**双入口**（徽章唯一入口的设计洞：没备注的节点永远创建不了第一条）；弹层 textarea 普通 Enter 是换行、Ctrl+Enter/按钮保存、保存写 trim 后文本；**清空=移除 `{note: undefined}` 干净丢键**——与 0053 `icons` 的 `[]` 纪律刻意不同（空数组有视觉语义是「全清过」的记录，note 零渲染空串只是噪声）；落盘照走 `reshapeNode`→`operation`→emit 主通道；undo 后脏标记**迟到不是缺席**（`linkDiv`→`scheduleVerify` 300ms 追上，探针钉死）；大纲联动（`OutlineItem.note` + 面板 📝 指示——指示是「有」正文不进大纲，0052 纯函数路线）；导出不伺候（`exportSvg` 按选择器挑元素、未知类名不选走⇒徽章天然不进图，是文档化语义：导出文件内容、徽章是编辑器部件同右键菜单）；搜索不伺候（`replace` 是 topic 同步通道，note 是异步 reshape——旧注释理由过期已改写，note 搜索记重审）；只读徽章照显、弹层只读无按钮（读不是编辑，🔗 同款） | 已采纳 |
| [0057](./0057-markdown-kernel-ui-follows-the-language.md) | markdown 内核自己画的字也得跟着语言走（0047 登记在案的「落点在内核绘制侧」这轮补上；0049 让导图跟随语言、markdown 整模块却**零 i18n 导入**——英文界面里代码块标签提示、图区纸条、图片占位全是中文）：三处回路形状不同机制只能各自配——badge 走 `chromeLocale` 域（`drawnLabel=null` 重画）、图区纸条走 `painted.locale` 失效→`ensureDiagram` 整链重渲染（`paintDiagramNote` 签名不改收 string——纸条本来就是渲染产物，重跑渲染才是正路）、图片占位记 `failLine`（`ImageView` **没有重渲染回路**只能原地换字）；**语言切换不重建编辑器**（与 0049 导图重建刻意不同：ProseMirror 状态、undo 栈、选区全丢不起）—— 照主题切换的既有通道 `subscribeLocale` + live 注册表（`$prose` 工厂闭包订阅、NodeView 构造 add/destroy delete、`view.destroy` 退订）；zh 逐字照抄旧硬编码⇒存量断言一行不动（兼容即契约）；**别人的话不翻**（mermaid 与 `renderDrawio` 的 `result.message`、`DocBlockPreview.reason` 原样转述——用户拿它搜上游 issue，翻译了反而害人）；26 对 `ui.markdown.*` 键 | 已采纳 |
| [0058](./0058-mindmap-note-search.md) | 导图查找也搜备注（0056 取舍 ⑤ 留的边界、0057 重审条件 2 登记的「另一轮」—— 本轮兑现）：`TopicHit` 加 `field`（`topic` \| `note`），契约零改动（`EditorSearch` 是索引制的，命中表是内核私有的）；`collectHits` 同一节点内 topic 先、note 后；`reveal` 到 note 亮弹层、到 topic 收弹层（只读亮只读弹层）；`replace` 按 `field` 分诊（note 走 `patchNote` + `reshapeNode`，到空串 = 清空即移除，写回必关弹层）；`replaceAll` 按「字段 + 节点」分段键控、组内倒序、报命中数；`NotePopover` 挂 `key={nodeId}`（草稿挂载播种，键盘流切节点不端陈旧草稿）；大纲与导出不伺候（照 0056 旧例） | 已采纳 |
| [0059](./0059-mindmap-node-style.md) | 导图节点的样式可编辑（对标 XMind 主题样式面板第一屏：文字色 / 填充 / 字号 / 加粗）。**与 0053–0056 性质不同的一轮：格式层与适配层零改动** —— `style` 早就是正字段（`types.ts:152-153` 不透明直通、`KNOWN_NODE_KEYS` 里有它），内核也**一直在渲染**（`Se` 把每个键原样塞 CSSOM），缺的只是「写」；风险全在「怎么写才不留脏值」：① 词表**恰好是内核导出认得的那四键**（`LayoutSsr.js:45-47` 只拼 `color`/`background`/`font-size: ${v}px`/`font-weight`，多一个键就是「编辑器画得出、导出没有」，0050 的落差）——`branchColor` 刻意不做（它住 `extra` 桶，可编辑 = 一次升格，该单独一轮）；② **`reshapeNode` 是同步的**（`Xt` 无 `await`，`operation` 在函数体里同步 fire ⇒ 补丁交出去时文本已写出）⇒ 摘键必须**在补丁里就干净**：设值 `{style:{[k]:v}}`、摘一键 `{style:{...rest,[k]:undefined}}`（`Object.assign` 是加/改语义，`JSON.stringify` 丢 `undefined`）、摘最后一键或「清除」走 `{style: undefined}` —— 逐键置 `undefined` 会留下 `"style": {}` 空壳（适配层 `Object.keys().length > 0` 守卫拦不住，§7.1 说等价但**等价不等于该写**）；③ 词表外键照 §5 原样保留，只有用户显式「清除」才整份撤；④ **`fontSize` 存数字、屏幕上补 `px`**（导出按 `${v}px` 拼要数字、CSSOM 忽略无单位长度要单位）——补丁挂 `linkDiv`（init + 每次重画，覆盖 undo/redo 的 `refresh` 不发 operation 那条路，0056 同款时机），DOM 擦除用 `removeProperty(cssPropertyName(key))`（`removeProperty` 只认 CSS 属性名，camelCase 静默不删，转换同 `LayoutSsr.js:62`）；⑤ UI 照内核自有界面纪律（右键 `contextMenu.extend`「样式…」+ `MindmapEditor` 内 `StylePopover`，**不进 `EditorHandle`**），亮不亮由 `node-style.ts` 纯函数判（**词表外的值不亮也不假装摘得掉**），点色块不关弹层、点外面/Escape 关，只读靠 `contextMenu: false` 自然关门（样式无常驻可见物，不需要只读弹层）；已知边界：深色主题下浅底对比度不保证（样式跟文件走不跟主题翻）、带单位字符串字号导出会拼成非法值（如实保留不替别人改写） | 已采纳 |

## 与其他文档的关系

- **决策定下来的写在本目录**（0001 起顺序编号）。
- **尚未决策、或已决策但有待验证的出口条件**，记在 [`../open-decisions.md`](../open-decisions.md)。
- **格式与互操作的技术规范**见 [`../format-spec.md`](../format-spec.md)（规范与决策分开：
  规范说「是什么」，ADR 说「为什么这样选」）。
- **与外部成熟项目的对标**见 [`../gap-analysis-horsemd.md`](../gap-analysis-horsemd.md)。

## 0004 的遗留部分

0004 把「Markdown 内核能否达到 Typora 体验」和「思维导图内核选型」列为待定。
其中**思维导图已由 0007 解决**；Markdown 内核仍待 P2 实测后回填
（判据：长文档 1 万字以上输入延迟不超过 50ms，超出则改为分栏预览方案）。

> **这条至今仍挂着，而且它是本目录里最该被清掉的一项。**
> 参考项目 horseMD 已经把 Milkdown Crepe 用起来了，同时又在 `spike/live-preview/`
> 里实测 CM6 + Lezer 的「源码即模型」方案（12.8 万汉字：解析 5.4ms、首屏 8ms、
> 单键重解析 0.35ms），并规划了迁移路径。
> 也就是说：**一个已经把 Milkdown 用起来的项目正在评估离开它**。
> 这个信号应该被纳入 V3 的判断，见 [`../gap-analysis-horsemd.md`](../gap-analysis-horsemd.md) §3.3。

## 0013 的遗留部分（lint 欠账）—— 已清零

0013 让 ESLint 第一次真正跑了起来。第一次全量扫描的结果是 **64 项**，
其中 31 项是 `unbound-method` 的误报（当时选择先关掉规则），**剩余 33 项是真的**。

**现在 0 项，而且配置里不再有任何被关掉的规则。**

选择的是下面两条路里的**第一条**：把欠账清零，然后把 lint 设为强制门禁。
`npm run check` 与 `.github/workflows/ci.yml` 现在都会因 lint 失败而失败。

清零过程与每一条的去向记在 [`../../ROADMAP.md`](../../ROADMAP.md) 的
「lint 清尾（第 20 项）」一节。其中两条值得在这里单独留一笔：

1. **`unbound-method` 是被「修好」的，不是被「豁免」的。**
   11 个文件里的回调 prop 从方法简写改成属性签名之后，规则不再误报，于是打开。
   附带效果是回调参数开始按**逆变**检查，类型安全变严了 ——
   而且实测没有暴露新的类型错误，说明原来的标注本来就是对的。

2. **它暴露了一个双编译器方案的系统性分歧。**
   同一行代码，**TS 6 算出 `never`，TS 7 算出 `number`**
   （TS 6 的控制流分析对「在回调里赋值 → 回到外层收窄」这种模式失效）。
   也就是说 **lint 看到的类型不一定等于 `tsc` 看到的类型**，
   于是「`tsc` 通过」不再蕴含「lint 通过」。这是 0013 这个方案的固有代价，
   已写进 0013 的「代价」一节与 ROADMAP。
