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
