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

## 0013 的遗留部分（lint 欠账）

0013 让 ESLint 第一次真正跑了起来。第一次跑的结果是 **64 项问题**，
其中 31 项是误报（已关规则），**剩余约 29 项是真的**，明细见 0013 末尾的表。

这些项**不阻塞提交**，但有一条必须留意：`npm run check` 现在包含 `lint`，
而 CI 里也调了 `npm run lint`。**在把欠账清零之前，CI 的 lint 步骤会红。**
两种选择，都需要一个明确的决定（不要让它无声地挂着）：

- 把剩余项清零，然后把 lint 设为强制门禁；
- 或者先把 CI 里的 lint 步骤标为 `continue-on-error`，并在本文件里登记欠账。

**倾向第一种。** 理由与 `../open-decisions.md` 的 V1–V4 同源：
一份「写了但不执行」的检查清单，比没有清单更糟 —— 它会让人以为覆盖到了。
