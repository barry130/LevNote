# Markdown 表格单元格会话恢复交接 · 2026-11-19

## 停止点

用户要求暂停开发并准备其他 Harness 接手。md-drafts 完成只读源码核对并获得 lead 对接线范围及测试接缝的批准，随后在第一处实现或测试写入前停止。本代理唯一写入为本交接文件。没有半成品代码、Git 操作或在运行的后台任务。Go 测试尚未启动，没有测试命令记录、日志、退出码或新证据目录。

## 当前行为与已核对位置

- 表格单元格使用独立编辑缓冲区。该缓冲区的 `OnChange` 当前走空路径。完成编辑时才校验并写入主文档。当前工具提示说明单元格暂存不进入会话快照。[table_cell.go](<../modules/markdownview/table_cell.go#L350>) · [view.go](<../modules/markdownview/view.go#L465>)
- App 的定时会话恢复在 UI 线程采集，由 `noteEdit` 重置 900 ms 静默期。`noteEdit` 不修改文档缓冲区。持久化循环当前只记录已有标签和文档，恢复循环已经能打开路径为空的独立草稿。[autosave.go](<../app/autosave.go#L38>) · [state.go](<../app/state.go#L124>) · [state.go](<../app/state.go#L200>)
- `SetMarkedText` 也会触发共享编辑器的 `OnChange`。`ReplaceText` 确认组合后会通知，直接 `UnmarkText` 则只使视图失效。因此需要区分已确认文本与预编辑，另补观察仅走 `UnmarkText` 的结束路径。[editor.go](<../ui/editor/editor.go#L332>)
- 会话模型现有独立 `Draft`、`OriginalPath`、UTF-16 选区与 `View`。实际常量为单份 4 MiB、原始草稿合计 16 MiB、精确 JSON 上限 16 MiB、最多 100 个会话标签。`SnapshotBuilder.AddTab` 对已有恢复记录执行上述检查。[session.go](<../core/session/session.go#L19>) · [builder.go](<../core/session/builder.go#L93>)

## 已批准的接入方案，尚待实现

- 新建 `native/ui/editor/draft_recovery.go`，定义可选的中性捕获接口 `CaptureSessionDraft(maxBytes) (draft, ok, err)` 和包含标签、内容、选区的数据结构。在 [contract.go](<../ui/editor/contract.go#L84>) 加 `DraftRecoveryChanged func(owner View)`。在 [reference.go](<../app/reference.go#L177>) 的 `editorHost` 仅接受仍打开且视图身份相同的通知，并调用 `noteEdit`。
- 新建 `native/modules/markdownview/draft_recovery.go`，实现只读捕获。采集前核对 `LenBytes`，超限报错，在组合输入期间使用上一次确认态的有界不可变快照和相应选区。每次只保留必要缓冲。稳定的普通输入读取当前已确认内容。等于打开时的基线就省略记录，从非空清为零字节仍生成记录。恢复捕获不调用 Flush，不移动焦点或改动宿主选区、撤销栈、主文档。修改 [table_cell.go](<../modules/markdownview/table_cell.go#L350>) 接入独立草稿通知，[draft.go](<../modules/markdownview/draft.go#L34>) 清理缓存并让显式放弃刷新会话，[view.go](<../modules/markdownview/view.go#L441>) 补观察仅经 `UnmarkText` 的确认并更新提示。Draw 仅补观察状态转换。
- 在 [state.go](<../app/state.go#L200>) 主标签后加入独立记录，由新增 `native/app/draft_recovery.go` 适配。建议以清楚命名的路径为空 `.txt` 源码标签恢复，采用 UTF-8，将原路径仅作为 `OriginalPath`，活动标签对应主标签。留意临时主文档、重复保存、显式放弃和退出时已选丢弃的路径。捕获或预算失败走现有可见错误并保留上份快照。沿用会话模型，无需 schema 迁移或依赖变更。

## 恢复开发后的第一步

1. 用现有真实 Markdown 表格 Tester 辅助方法和假时钟新增 `native/app/draft_recovery_test.go`。输入确认文本，验证 900 ms 定时触发后保存独立副本，并用新 App 实际恢复。覆盖 clean 主文档与原文件编码和字节，UTF-16 选区、空文本、放弃后无残留、预算失败时旧快照完整。现有 [markdown_cells_app_test.go](<../app/markdown_cells_app_test.go#L143>) 有一组旧产品断言要求快照完全排除单元格草稿。新行为应更新该断言为验证独立副本，同时保留主文档不写回的断言。该既有测试文件应由接手负责人明确分配后再编辑。
2. 新建 `native/modules/markdownview/draft_recovery_test.go`，用现有原生编辑输入和中性契约验证预编辑保持活跃、先前确认态可恢复、确认后更新以及放弃后消失。物理 Windows IME 仍待专门验证。
3. 恢复写入授权后按批次约定运行小包。使用 `bash .localtools/gate.sh test -count=1 -timeout=300s ...`，在 `.localtools/noteboard-parity-20261119/md-drafts/` 为每次执行新建独立命令、日志和退出码记录。源码核对期间有读取路径与行范围的工具错误，随后以绝对路径和有效源码片段完成调查。没有产品执行失败可交接。

写入权已释放，由接手 Harness 及 lead 重新分配。
