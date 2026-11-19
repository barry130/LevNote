# drawio-finish 停止开发交接 · 2026-11-19

## 状态与接手约束

用户通过 lead 最新指令要求停止开发、整理交接并准备由其他 Harness 接手。drawio-finish 已停止源码写入，不再实现、格式化、增加测试或运行新检查。本文件是停止后的唯一写入。源码不是已验收终态，六项 P2 不得标记全部闭合。全部原 dirty/untracked 内容及历史证据均保留；未重置、未删除、未改基线副本、未操作 Git、未修改共享记忆、未新增依赖。原生主线 Go1.27.3/MyGo，纯本地、自研内核；旧接受包不受本任务替换。

## 本轮实际修改的源码（仅三个文件）

1. `native/modules/graphview/drawio_clone.go`
   - 删除原基线两处 `fmt.Printf("LEAD ...")` 调试打印；其中第一处在声明前引用 ids/raw 导致编译错误，另一处输出选区 ID/内部状态。已核 `.localtools/noteboard-parity-20261119/baseline/source/native/modules/graphview/drawio_clone.go` 同样包含两处打印。
   - 02 实测连续两次克隆、两次撤销后选择为 []。定位为 beginDrawioCloneHistory 的 epoch fallback 用回放后的目标 raw 查询边界见证，画布仍持有回放前一侧的 IDs/epoch。现已改为 Undo 查询 record.afterText、Redo 查询 record.beforeText；Buffer 文本仍精确匹配目标 raw，owner/IDs 检查保留。
   - 此边界修正发生在最后一次测试之后，尚未重新运行。
2. `native/modules/graphview/drawio_clone_correction20261117_test.go`
   - 原测试把 XML 插入位移加到 caret 预期，实际实现及设计合同是保留数值 UTF-16 坐标；02 实测 buffer.Selection={376 376}，克隆选择已成功，原夹具没有真正触发夹取。
   - 将首 vertex 开始标签改为 `<mxCell value="🙂" id="a" ...>`，旧 caret 设为 rootClose_UTF16+16，在原关闭标签中合法，插入副本后相同数值位置应进入 emoji surrogate pair。预期为 caret-1。
   - 保留 changed/version、新选择/primary、恢复历史、精确成功提示、非法代理对验证、Undo 源码/原 caret/旧多选与 Redo 源码/夹取 caret/新多选断言。SliceUTF16 检查改为实际位置的空区间。
   - 新夹具与断言尚未运行，不能宣称通过。
3. `native/formats/drawio/clone_cells.go`
   - 按 lead 的额外静态发现，将每次从 1 探测 ID 改为本次调用单调 nextIndex 游标，仍用全页 occupied/used 集避重。
   - 旧逻辑先全量复制 insertion，再验证 MaxBytes；现改为保存原始 span 引用与重写标签 chunks，按 int64 累计最终字节数，逐段检查剩余预算，全部通过后才一次分配最终 next 并依序写入。避免先分配注定超限的整份插入副本。重写标签仍会预先分配；不是零额外内存或进程峰值测量结论。
   - 此修改刚完成即收到停止指令，未运行格式测试、预算边界新测试、编译或格式化。

未修改 adapter.go、input.go、delete 源码或测试、两个 app 测试、Markdown 或其他实现者文件。未创建新工程测试文件。

## 已完成的阅读与现态发现

已读首批计划、native/AGENTS.md 最新首段及 Development constraints、drawio-clone-design-20261116.md、drawio-batch-correction-20261117.md、两份 drawio-batch-*-review-20261116.md、clone_cells.go 关键路径、既有两份 correction20261117 tests、drawio_multi_clone_test.go 关键行为和 input.go 的 Undo/Redo 接线。

- input.go 的 beginDrawioCloneHistory 在 Buffer.Undo/Redo 成功之后、ensureSynced 之前。该现态允许 begin 同时核对回放后的源码和仍在画布上的回放前选择；本轮未改调用顺序。不要未经行为验证改成 Undo 前调用。
- 原基线已有 256 条/32 MiB 快照预算、淘汰清空物理槽、Release 清除 buffer/text/IDs 引用；02 的相关测试通过。预算辅助计算按 beforeText/afterText 字节，不是全进程内存上限。
- 原基线已有 copied-port、内部 edge 子树和全选中页 XML ID 扫描修正及格式反例。当前任务未执行格式包，不能据静态阅读宣称这三项通过。
- 原基线 continuous-history 反例实际 RED；修改后的回放修复未验证。不要只引用 KnownBoundary 的人工记录测试代替真实 Ctrl+D/Undo/Redo。

## 实际执行与原始证据

证据独占目录：`.localtools/noteboard-parity-20261119/drawio-finish/`。以下两次都通过 gate 使用固定离线工具链，日志和退出码分别保存，没有覆盖旧证据。

1. `bash .localtools/gate.sh test -count=1 -timeout=300s ./modules/graphview -run DrawioClone -v`
   - `run-01.sh`；`01-command.txt`、`01-log.txt`、`01-exit.txt`。exit **1**。
   - 原始错误：drawio_clone.go:208 undefined: ids / undefined: raw；build failed。运行时源码为受保护基线原态。
2. `bash .localtools/gate.sh test -count=1 -timeout=300s ./modules/graphview -run "Drawio.*Clone" -v`
   - `run-02.sh`；`02-command.txt`、`02-log.txt`、`02-exit.txt`。exit **1**。
   - 此时只删除了两处调试打印，尚未修改回放/夹具/格式层。
   - FAIL TestDrawioMultiCloneClampCompletesSelectionHistoryAndFeedback：real clone does not clamp and select result 1 {376 376} [levnote-1 levnote-2]。
   - FAIL TestDrawioMultiCloneContinuousHistoryRespectsOwnerAndSelectionEpoch：two undos did not restore the original source and selection []。
   - PASS TestDrawioCloneHistoryEntryAndByteBudgets、TestDrawioCloneHistoryWitnessOwnerAndReleaseReferences、TestDrawioMultiCloneOneTransactionSelectionAndExactUndoRedo、TestDrawioCloneRejectsInvalidAndLifecycleChanges（empty/single/missing/replaced-buffer/readonly）。
   - 后台 job `bash-e2070dcbf72f415a` 已收集，completed/exit 1；停止时 job_list 无运行任务。

没有新的 GREEN 终态；没有跑全量 test/vet/build、app 测试、GUI、真实 Windows 输入法/剪贴板/物理操作或打包。不得把 02 的部分 PASS 套用到后来改过的最终源码。

## 尚未完成与建议首步（需新接手授权后执行）

1. 先审相对 `.localtools/noteboard-parity-20261119/baseline/source/` 的上述三个文件 diff，保留所有既存改动；核对新 clamp 夹具是否在保持数值坐标合同下真实落到代理对内部。
2. 保存新的独立 evidence，再跑 `bash .localtools/gate.sh test -count=1 -timeout=300s ./modules/graphview -run "Drawio.*Clone" -v`；验证多轮 clone→undo→redo、中途明确换选、页/owner 生命周期 veto。不能复用 01/02 路径。
3. 校验 clone_cells.go 的 chunks 字节总数、精确 MaxBytes 边界与超限失败原子、连续已占 ID 避重；需要时新增聚焦反例，然后跑 ./formats/drawio。此项原计划添加的测试未动笔。
4. 通过聚焦修正后再按 lead 新授权跑完整受影响 formats/drawio、modules/graphview 与相关 app clone/multi-delete 测试，记录最终命令/log/exit；若只允许本次整理，保持以上未完成状态。
5. 六根最终检查、独立审阅与提交上传由 lead/后续 Harness 处理。本代理无 Git 操作或提交上传权限。

合同边界继续保持：仅当前页单层受支持 vertex 及内部 edge；不扩分组、跨页、剪贴板、任意 edge/mixed selection、外部边重连或 waypoint 位移。

## 所有权释放

源码写入已停止，无运行 job；本 handoff 写完即释放 drawio-finish 全部所有权，等待接手。
