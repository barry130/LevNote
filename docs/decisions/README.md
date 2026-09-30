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
