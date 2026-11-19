# 格式交互审计与围栏修复 · 2026-11-15

源码审计基点：`f2c2ceb82acecb40fb0502c047e41ba12a12be47`。本轮收敛到一个已由公开输入回归复现的 P2 缺陷；lead 随后授权实施该修复。采用 `code-review` 与 `tdd` 技能，测试 seam 为外部测试包使用 `ui.Tester` 驱动 `View.Draw` 的双击、输入、完成按钮，以及 `Flush/Export/Host.Buffer` 合同。

## 发现与影响

**P2：替换已闭合 Markdown 代码块正文会把关闭围栏接到正文末尾，吞并后续段落。**

1. 打开源码 `"```go\nold\n```\n\nafter\n"` 的 Markdown 阅读视图。
2. 双击代码块，执行全选并输入 `new`，输入中没有末尾换行。
3. 点击“完成编辑”。修复前实际源码是 `"```go\nnew```\n\nafter\n"`，后续 `after` 被未闭合代码块吞入。

可达链路是 [rows.go:120](<F:/AIproject/LevNote/native/modules/markdownview/rows.go#L120>) 的真实双击 → [draft.go](<F:/AIproject/LevNote/native/modules/markdownview/draft.go>) 的 `begin/openDraft/syncDraft`。原因是 [parser.go:348](<F:/AIproject/LevNote/native/formats/markdown/parser.go#L348>) 将关闭围栏前行尾包含在 `TextRange`；旧写回直接将整个字段替换为输入，漏保关闭围栏的独立行语义。该问题属于已有可编辑代码块能力，不是新增 Markdown 语法或文档化的原生子集限制。

## 修复范围与行为

产品变更仅两文件：

- [draft.go](<F:/AIproject/LevNote/native/modules/markdownview/draft.go>)：为已闭合普通代码块草稿记录原 LF/CRLF 与精确源字段；非空输入的最后一个结构分隔采用原行尾，输入没有末尾换行时补齐。正文内行尾与本地原输入不改动。空正文不加空行；未闭合块不补关闭符。`editBase` 仍保存真实输入，源字段比较单独保存补齐后的值；成功才更新基值、范围和版本，失败沿用现有草稿保留机制。清理草稿时同时清除此状态。
- [view.go:35](<F:/AIproject/LevNote/native/modules/markdownview/view.go#L35>)：新增一个可空的局部草稿状态指针。解析器、格式导出、公共契约与 app 均未改动。没有每键全篇重解析。

新增 [fenced_draft_test.go](<F:/AIproject/LevNote/native/modules/markdownview/fenced_draft_test.go>) 使用外部 package，首条 RED/GREEN 测试不访问视图私有字段，不改原有断言或 SKIP。后续对照覆盖 LF、CRLF、初始空正文、全删、较长围栏及关闭围栏无末尾换行、显式行尾、未闭合 EOF、重复 Flush 清洁、继续输入、clean host Undo、只读/未编辑、预算失败恢复与重试、外来事务重试、外部冲突原输入恢复。

## 原始执行证据

所有产品命令仅经过固定 [gate.sh](<F:/AIproject/LevNote/.localtools/gate.sh>) 的 Go1.27.3/offline vendor 配置。证据位于 [run-focused.sh](<F:/AIproject/LevNote/.localtools/continuation-fence-20261115/run-focused.sh>) 所在的新目录；每次保存完整命令、stdout、stderr、exit，以及本 scope 三份 Go 源码和 gate 的 before/after SHA256。该 pin 只覆盖本任务，不作为全仓快照声明。

| 阶段 | 精确产品命令 | 结果与证据 |
|---|---|---|
| RED | `bash .localtools/gate.sh test -json -count=1 -timeout=120s ./modules/markdownview/ -run TestFencedDraftReplacementPreservesFollowingParagraph` | exit 1；[red-03 原流](<F:/AIproject/LevNote/.localtools/continuation-fence-20261115/red-03.stdout.jsonl>)、[退出码](<F:/AIproject/LevNote/.localtools/continuation-fence-20261115/red-03.exit.txt>)。真实失败为 `new` 与关闭围栏相接。 |
| 首个 GREEN | 同 RED 命令，同一测试原断言 | exit 0；[green-01 原流](<F:/AIproject/LevNote/.localtools/continuation-fence-20261115/green-01.stdout.jsonl>)、[退出码](<F:/AIproject/LevNote/.localtools/continuation-fence-20261115/green-01.exit.txt>)。一个测试 PASS，验证精确源码、后段 HTML、单步 Undo/Redo。 |
| 初版聚焦 | root 使用同一助手，以 `-run '^TestFencedDraft'` 执行初版对照 | exit 0，16 PASS、0 FAIL、0 SKIP；[final-01 原流](<F:/AIproject/LevNote/.localtools/continuation-fence-20261115/final-01.stdout.jsonl>)、[退出码](<F:/AIproject/LevNote/.localtools/continuation-fence-20261115/final-01.exit.txt>)，stderr空，scope pins前后一致。 |

RED 和首个 GREEN 的 before/after 四项 pin 均一致；两阶段首条测试 SHA256 同为 `32f78af271bdb81ce499baa16456920c2d577aea26b179b33c23137673a144ae`。修复前 [RED pins](<F:/AIproject/LevNote/.localtools/continuation-fence-20261115/red-03.before.sha256>) 与修复后 [GREEN pins](<F:/AIproject/LevNote/.localtools/continuation-fence-20261115/green-01.before.sha256>) 均保留。

证据助手最初两个 phase 在 `set -C` 与 MSYS 外部命令重定向处报 `cannot overwrite existing file`：[red-01 命令](<F:/AIproject/LevNote/.localtools/continuation-fence-20261115/red-01.command.txt>)、[red-01 空 pin](<F:/AIproject/LevNote/.localtools/continuation-fence-20261115/red-01.before.sha256>)、[red-02 命令](<F:/AIproject/LevNote/.localtools/continuation-fence-20261115/red-02.command.txt>)、[red-02 空 pin](<F:/AIproject/LevNote/.localtools/continuation-fence-20261115/red-02.before.sha256>) 原样保留。这两次 wrapper exit 2，未进入 Go 测试；助手仅取消 `noclobber`，后续使用全新 phase，未覆写旧失败。

## 独审发现的真实回车边界

Spec独审发现：CRLF块全选替换为单行后，原生编辑器的真实Enter默认产生LF，初版会把关闭围栏前分隔换成LF。root新增 `TestFencedDraftEnterPreservesClosingLineEnding`（已有正文/初始空正文两个子例），[review-red-01](<F:/AIproject/LevNote/.localtools/continuation-fence-20261115/review-red-01.stdout.jsonl>) exit1，两个子例精确复现 `new\n` 而预期 `new\r\n`。

仅修最后一个结构行尾后，同一测试 [review-green-01](<F:/AIproject/LevNote/.localtools/continuation-fence-20261115/review-green-01.stdout.jsonl>) exit0。测试还验证继续键入时原输入内LF保持、末尾结构CRLF保持、完成不重复改写。最终 [final-02](<F:/AIproject/LevNote/.localtools/continuation-fence-20261115/final-02.stdout.jsonl>) 19 PASS、0 FAIL、0 SKIP，exit0；原16项结果与新RED均保留。

## 审计边界与交接

此次也阅读了 mindmap/drawio 的解析、变更写回、图形字段草稿与 Markdown 链接/任务/块草稿路径，排除已文档化的有限原生子集、JSON 结构编辑重排和此前已修复的恢复所有权问题。drawio `html=0` 下含 `<`/`>` 字面标签存在进一步调查候选，本轮没有运行回归或实现，不列入本轮已确认缺陷。

现有最小修复的首个 RED→GREEN 已闭合；app-audit释放执行槽后，root完成初版16项聚焦，并在独审回车修正后完成最终19项聚焦，全部通过。没有全量/vet/build、桌面操作、包构建、提交推送、工具链或权限改动；原证据及旧未跟踪文件保持。没有发现必须通过 app 交叉修改才能完成的依赖。最终完整门禁与独立审阅由 lead 统一执行。
