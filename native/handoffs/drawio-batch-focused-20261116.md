# draw.io 克隆／批删：单次聚焦验证（2026-11-16）

结果：**PASS；Go gate exit 0，宿主 wrapper exit 0。14/14 个顶层用例、27/27 个含子测试身份通过，3/3 个目标包通过，0 FAIL、0 SKIP、0 重复运行。** 全程只执行一次聚焦 gate；启动时间为 2026-11-16 19:30:29 +08:00，最后一个包于 19:30:33 +08:00 完成，含输入快照的包装器用时 5.137 秒。实际结果见 [result.json](<F:/AIproject/LevNote/.localtools/drawio-batch-resume-20261116-verify01/result.json>) 和 [host-job-receipt.json](<F:/AIproject/LevNote/.localtools/drawio-batch-resume-20261116-verify01/host-job-receipt.json>)。

验证者：drawio-verify；委派者：lead。本批固定比较点及实际运行前后 HEAD 均为 `f2c2ceb82acecb40fb0502c047e41ba12a12be47`。本任务仅验证当前克隆／批删增量，固定基点后的既有多选、样式等已接受未提交修改不重新列作本批缺陷。指定五个 tracked 文件的完整工作树差异保存在 [tracked-diff.stdout](<F:/AIproject/LevNote/.localtools/drawio-batch-resume-20261116-verify01/tracked-diff.stdout>)，八个指定 untracked 源码／测试均完整读取并纳入 SHA 清单。

已读取 [AGENTS.md](<F:/AIproject/LevNote/native/AGENTS.md>) 的前 43 行和 Development and product constraints、[CONTRIBUTING.md](<F:/AIproject/LevNote/CONTRIBUTING.md>)、[drawio-clone-design-20261116.md](<F:/AIproject/LevNote/native/handoffs/drawio-clone-design-20261116.md>)、[feature-parity-drawio-next-20261115.md](<F:/AIproject/LevNote/native/handoffs/feature-parity-drawio-next-20261115.md>)；使用 code-review 技能的固定基点与合同检查原则，未追加委派。

实际命令（工作目录 F:/AIproject/LevNote）：

```bash
bash .localtools/gate.sh test -json -count=1 -timeout=180s -run '^(TestCloneCellsEmitsVerticesInSelectionOrderThenInternalEdges|TestCloneCellsFailureLeavesSourceOriginalAndClean|TestCloneCellsPreservesOriginalSpansAndRewiresInternalEdge|TestDeleteCellsFailureLeavesWholeDocumentUntouched|TestDeleteCellsRefusesSurvivingPortReference|TestDeleteCellsRemovesSelectionAndIncidentEdgesAtomically|TestDrawioCloneAppExactDiskUndoRedoAndReopen|TestDrawioCloneRejectsInvalidAndLifecycleChanges|TestDrawioMultiCloneOneTransactionSelectionAndExactUndoRedo|TestDrawioMultiDeleteAppExactDiskUndoAndReopen|TestDrawioMultiDeleteInvalidTargetDoesNotWrite|TestDrawioMultiDeleteOneTransactionPreservesRemainingXMLAndUndo|TestDrawioMultiDeleteReplacedBufferKeepsOriginalSource|TestDrawioMultiDeleteSecondNoSelectionKeepsSource)$' ./formats/drawio/ ./modules/graphview/ ./app/
```

正则由六个已读取测试文件中实际存在的 Test 名称构造，先经 grep 核对，并以首尾锚点精确选择 14 个顶层用例。身份到源文件的对应见 [test-selection.json](<F:/AIproject/LevNote/.localtools/drawio-batch-resume-20261116-verify01/test-selection.json>)。执行入口为 [gate.sh](<F:/AIproject/LevNote/.localtools/gate.sh>)，其固定 Windows Go1.27.3、离线 vendor、本地 cache/tmp、GOTOOLCHAIN=local 与 GOTELEMETRY=off 设置保持原样；已另读 [VERSION](<F:/AIproject/LevNote/.localtools/go1.27.3/VERSION>) 确认为 go1.27.3。未执行六根全套测试、vet、build、第二次产品 gate、提交或推送。

| 目标包 | 顶层测试 | 含子测试 PASS | 包结果 | Go 报告耗时 |
|---|---:|---:|---|---:|
| formats/drawio | 6 | 14 | PASS | 0.862s |
| modules/graphview | 6 | 11 | PASS | 1.660s |
| app | 2 | 2 | PASS | 2.473s |
| 合计 | 14 | 27 | PASS | — |

[stdout.jsonl](<F:/AIproject/LevNote/.localtools/drawio-batch-resume-20261116-verify01/stdout.jsonl>) 共 120 行、26,451 字节，完整 JSON 解码无错误；[stderr.log](<F:/AIproject/LevNote/.localtools/drawio-batch-resume-20261116-verify01/stderr.log>) 为 0 字节。[tests.json](<F:/AIproject/LevNote/.localtools/drawio-batch-resume-20261116-verify01/tests.json>) 保存每个 test 的 run 次数和终态。重新从原始 stdout 核对三包终态与全部 27 个测试终态，均为 pass；没有遗漏预期顶层用例。

输入保全：[inputs-before.json](<F:/AIproject/LevNote/.localtools/drawio-batch-resume-20261116-verify01/inputs-before.json>) 与 [inputs-after.json](<F:/AIproject/LevNote/.localtools/drawio-batch-resume-20261116-verify01/inputs-after.json>) 各含 1,275 项，覆盖六个产品源码根、全部 vendor 文件、go.mod/go.sum、固定 gate、Go VERSION/go.exe/compile.exe/link.exe/asm.exe 及四份指导合同。两文件均为 210,833 字节且 SHA256 完全相同：`4054cd9b6b00ab80aa3b4c337fae24cf518e402f7f54e22270c6b27f628d2643`。变更输入列表为空。本子代理未改生产源码、测试断言、工具链、vendor、旧证据或既有包；新文件只创建在本次 resume 命名空间及本报告路径。缓存与临时目录由既定 gate 正常使用。

指定源清单（运行前后相同）：

| 文件 | 字节 | SHA256 |
|---|---:|---|
| [formats/drawio/delete.go](<F:/AIproject/LevNote/native/formats/drawio/delete.go>) | 7029 | `3ac8b7fd53d1ecf573c02a214d982b45daf40cf3840678993ad5e13c19fbec8d` |
| [modules/graphview/adapter.go](<F:/AIproject/LevNote/native/modules/graphview/adapter.go>) | 10051 | `a9cbf991e038f82d1edfcbae4b748deca51448f7654363ac5928c214f670d2a6` |
| [modules/graphview/input.go](<F:/AIproject/LevNote/native/modules/graphview/input.go>) | 8868 | `ee7e430d7c2e3818a834d6f824c480d29663292449ea6763da0de9854fafc928` |
| [modules/graphview/drawio_delete.go](<F:/AIproject/LevNote/native/modules/graphview/drawio_delete.go>) | 7928 | `c6ec7a7d6def380d102c3619ab2d7c87fb87fad5329b8810216aac30de559bd1` |
| [modules/graphview/sync.go](<F:/AIproject/LevNote/native/modules/graphview/sync.go>) | 7099 | `8a898f91478d9262aca4a8bf14d72c7fb8fd010b6fe3c7936177512425def6c2` |
| [formats/drawio/clone_cells.go](<F:/AIproject/LevNote/native/formats/drawio/clone_cells.go>) | 10778 | `50102470f91ca933203f6c5586582aa67f580ac74c574127ed81d5b67c695277` |
| [formats/drawio/clone_cells_test.go](<F:/AIproject/LevNote/native/formats/drawio/clone_cells_test.go>) | 5342 | `a14eac7055ad963d906a28134e39fc5d9f03e0da8f862d5679fb7e1d7c3e5c3c` |
| [formats/drawio/delete_cells_test.go](<F:/AIproject/LevNote/native/formats/drawio/delete_cells_test.go>) | 4441 | `ea141b678d5c7ca40f399061217260721a6e5cffdd8e404abd27f7780db3d056` |
| [modules/graphview/drawio_clone.go](<F:/AIproject/LevNote/native/modules/graphview/drawio_clone.go>) | 7027 | `20756d624facc8410d49111fe59f29b2b9b0081b7ce6c9de7c494bde5d2dd16a` |
| [modules/graphview/drawio_multi_clone_test.go](<F:/AIproject/LevNote/native/modules/graphview/drawio_multi_clone_test.go>) | 3986 | `9db44489e9470f5f4800dd85e2f94f094f48d8ceda4ce672020f2448c4b78eca` |
| [modules/graphview/drawio_multi_delete_test.go](<F:/AIproject/LevNote/native/modules/graphview/drawio_multi_delete_test.go>) | 3446 | `3c41ba29622615ed115627756cd15474812d9f26b8091fceff8808ed5dc529f2` |
| [app/drawio_clone_app_test.go](<F:/AIproject/LevNote/native/app/drawio_clone_app_test.go>) | 4634 | `09912b840f12b249017e1acc2ab179b6ede1e9b960aaf6499511497dee5926b8` |
| [app/drawio_multi_delete_app_test.go](<F:/AIproject/LevNote/native/app/drawio_multi_delete_app_test.go>) | 3409 | `e3fc991c2623b3661f2c75297c89669e109ccfc3093f453aac9ae45bd12bad10` |

上述 13 项的机器可读清单为 [source-manifest.json](<F:/AIproject/LevNote/.localtools/drawio-batch-resume-20261116-verify01/source-manifest.json>)，SHA256 `1fe882a4ddab98af9cb0cde77cd70cd3881bb30ae9b3d138b1f133cca640c497`。

旧记录检查：先读 [drawio-batch-validate-20261116.py](<F:/AIproject/LevNote/.localtools/drawio-batch-validate-20261116.py>)，确认它原本以 `.localtools/drawio-batch-20261116-{phase}-{attempt}` 作为结果路径。当前文件查询与目录核对只找到原有四个 drawio-batch 包装／审计／预览脚本，未找到任何该命名的结果目录；旧 clone/delete 原始 JSONL 也未定位。[old-evidence-check.json](<F:/AIproject/LevNote/.localtools/drawio-batch-resume-20261116-verify01/old-evidence-check.json>) 保留本次实际检查。历史叙述里的“专项绿”在本报告中不替代原始证据；此刻可核验的最新实际结果是本报告这一次运行。

本次通过的直接覆盖：格式克隆的原始片段／内部边重接／外部边保留／请求顺序／raw ID 避让与失败原子；格式批删的关联边闭包／未知内容保留／非法目标与存活 port 引用；视图单次克隆／批删事务、单次克隆的反复 undo/redo 与显式选择保护；应用层 BOM/CRLF、显式保存、重开与磁盘写入时机。

现有聚焦用例的覆盖边界：

- 克隆视图测试执行一次 Ctrl+D 后往返历史，未覆盖连续两次克隆再跨越两个历史项的选择恢复。
- 现有 port 用例是未复制单元的 style 引用命中选择；未覆盖被复制 span 上值不命中选择的 port 键、内部边未知子树、diagram 层 ID 冲突。
- 克隆应用用例保存的是 undo 后的原状态；克隆后新状态的直接保存／重开不在该用例内。
- ordinary-field 草稿、composition/group/syncFailed 及回调重入 owner 变化等合同场景未由这 14 个顶层用例逐项覆盖。

这些边界保留给 lead 的合同审阅与后续修正，不由本次绿色结果推导完整合同、物理输入或新包交付结论。本次没有为补覆盖临时改断言、加测试或重复运行。

证据封存：[command.json](<F:/AIproject/LevNote/.localtools/drawio-batch-resume-20261116-verify01/command.json>)、[launch.json](<F:/AIproject/LevNote/.localtools/drawio-batch-resume-20261116-verify01/launch.json>)（PID 17508）、[runner-manifest.json](<F:/AIproject/LevNote/.localtools/drawio-batch-resume-20261116-verify01/runner-manifest.json>)、[evidence-manifest.json](<F:/AIproject/LevNote/.localtools/drawio-batch-resume-20261116-verify01/evidence-manifest.json>) 保存命令、启动、执行器和产物身份。stdout SHA256 为 `44fcf72b2e53e312b6470c4e58935a176b311cd3eaa88f327357d6f03d604402`；result SHA256 为 `f59834b3d0f6f718aba2e2ff1dc229ef0c06bee7e6e10113db3d5f36c112bbe9`。包装器自身与证据文件均用 create-only 创建，宿主后台任务 `bash-651e18fd5b324afd` 已完成并收取，当前没有本任务运行中的 job。

验证任务完成；交由 lead 汇总独立审阅。验证者释放执行与证据写入 ownership，等待下一项明确指派。
