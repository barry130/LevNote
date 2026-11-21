# 完整交付前失败基线审阅 · 2026-11-19

状态：**失败分类与测试身份清点完成，产品未通过交付验收**。Lead 只进行调度、源码审阅和证据整理，没有修改产品源码、测试断言、skip、旧证据或发行包。本文替代“app 关闭死锁”“Markdown 产品运行时直接 panic”等未经核实的归因。

## 身份与方法

当前提交为 `2ebec3d8ef47338f4efa2f913cc64015f61253d6`，环境为 Linux amd64。原始六根测试的 [JSON 日志](../../.localtools/setup-20261119-linux/native-tests.stdout.log) SHA256 为 `72f50503978b2936aaad5f266db5469004385afb2471c910c9bab76d077de68d`，只读解析，未覆盖或重跑原检查。

新证据位于 [本次摘要](../../.localtools/full-delivery-20261119/lead-review/round02/review-summary.json)。[审阅脚本](../../.localtools/full-delivery-20261119/lead-review/round02/audit-baseline.py)只读取源码/旧日志并以 create-only 方式生成新证据，不编辑工程文件。原始失败按包/完整测试名保存在 [逐项失败证据](../../.localtools/full-delivery-20261119/lead-review/round02/failed-test-evidence.json)，panic/timeout 保留原日志行号于 [栈记录](../../.localtools/full-delivery-20261119/lead-review/round02/panic-and-timeout-stacks.json)。

新增检查仅为 Linux 顶层测试名称列举，实际命令为：

```bash
bash .localtools/native-go.sh test -json -list . -timeout=60s ./app/ ./cmd/... ./core/... ./formats/... ./modules/... ./ui/...
```

[命令](../../.localtools/full-delivery-20261119/lead-review/round02/test-list.command.txt)、[stdout](../../.localtools/full-delivery-20261119/lead-review/round02/test-list.stdout.jsonl)、[stderr](../../.localtools/full-delivery-20261119/lead-review/round02/test-list.stderr.log)、[退出码](../../.localtools/full-delivery-20261119/lead-review/round02/test-list.exit)、[开始时间](../../.localtools/full-delivery-20261119/lead-review/round02/test-list.started.txt)、[结束时间](../../.localtools/full-delivery-20261119/lead-review/round02/test-list.finished.txt)均单独保存；exit 0、stderr 为空。**该命令没有执行测试用例，包的 list 成功不代表行为测试通过。**

## 覆盖缺口

| 统计口径 | 实际结果 |
| --- | --- |
| 六产品根当前输入 | 442 个文件，包含 236 个测试源码文件、14 个 testdata 文件 |
| 所有平台测试源码声明扫描 | 1481 项；文本声明扫描，不是 AST 或目标平台实际执行结果 |
| Linux 编译后实际列举 | 1390 个 Test + 17 个 Benchmark |
| 原基线含子测试身份 | 2469 项：2349 PASS / 103 FAIL / 15 SKIP / 2 未终态 |
| 原基线未执行的 Linux 顶层测试 | **426 项：app 341 项，markdownview 85 项** |
| 原基线包结果 | 20 PASS / 5 FAIL / 2 无测试包 |

原先只强调“2 个测试未终态”低估了截断的影响。2 个已开始未终态测试属于同一个 app 父测试及其 `cleanup-error` 子测试；另有 426 个顶层测试根本未到达，不能纳入通过比例或接受结论。完整名单见 [未执行测试](../../.localtools/full-delivery-20261119/lead-review/round02/unreached-top-level-tests.json)。

防退化材料分成三类，各自作用不能混用：

- [产品输入 SHA256](../../.localtools/full-delivery-20261119/lead-review/round02/product-input-sha256.json) 固定实际工作树文件字节，供修复后逐文件审阅断言、平台约束与 testdata 变化。
- [Linux 顶层身份](../../.localtools/full-delivery-20261119/lead-review/round02/linux-top-level-identities.json) 固定该平台真实可发现的入口；[全部平台声明扫描](../../.localtools/full-delivery-20261119/lead-review/round02/static-test-declarations.json)补充 Windows 条件文件等源码身份，但不宣称它们已执行。
- [原始运行身份](../../.localtools/full-delivery-20261119/lead-review/round02/observed-test-identities.json) 固定已到达的动态子测试及终态。后续重命名/拆分必须给出对应关系，缺席、skip 或减少覆盖不能算修复。

## 根因与证据分级

### R1：插件配置根目录符号链接检查缺失——可定位的产品缺陷

原基线 `TestPluginRootsRejectSymlinks` 在 [测试:650](../core/plugins/plugins_test.go#L650) 报 `settings root symlink was not reported`。夹具已成功建立指向现有目录的 symlink；如果创建失败，既有测试会 skip，但本次实际走到 FAIL。

[NewManager:70](../core/plugins/registry.go#L70) 检查安装根目录，随后调用 `LoadSettings(settingsRoot)`；[LoadSettings:524](../core/plugins/declarative.go#L524)直接对拼接后的配置文件执行 Lstat。若根目录是 symlink 且目标内没有该文件，ENOENT 被转换成空设置、无错误。根目录未被校验。因此本项有明确源码与失败证据，**不能归为 Linux 平台噪声**。[SaveSettings:576](../core/plugins/declarative.go#L576)已调用根目录检查，但它不能补回读取路径上的缺口。

最小实现候选为同一配置实现及现有测试文件。需要覆盖目标内配置不存在/存在的根 symlink、普通空根和正常配置读取，保留“不存在配置可以正常初始化”的契约。现有测试在第一个 Fatal 处停止，后面的安装根与写入拒绝断言本轮未被该用例执行到；修复后应全部到达。实现与新验证交给宿主代理。

### R2：app 超时是包总预算耗尽，当前栈未进入关闭——不能直接归为死锁

原日志显示整个 app 包 300.015 秒失败；报警时父测试运行 22 秒，`cleanup-error` 子测试仅 2 秒。栈位于 [awaitExportCall:75](../app/export_worker_test.go#L75)，调用点是 [关闭测试:531](../app/export_worker_test.go#L531)。测试尚未进入后面的 shutdown/close 分支。

原基线有 **58 项失败**直接报 `export did not reach runner`，其等待函数每次最多 5 秒。源码解释了 Linux 上不会进入 fake exporter 的前置拒绝：[startEngineExport:99](../app/export_worker.go#L99)先调用 [exportDescriptor:484](../app/export_worker.go#L484)，它按 `runtime.GOOS/GOARCH` 校验。[四项目录描述](../core/enginecatalog/catalog.go#L38)均为 Windows amd64。测试虽然注入了 fake exporter，但没有越过更早的描述符平台校验。

这是已定位的平台前置条件与测试夹具不一致，不证明 Windows 导出、取消或关闭正常。**不得通过放宽产品平台门禁、删除生命周期断言、统一新增 skip 或单纯加长 300 秒来宣称修复。**宿主代理应设计保持真实 Windows 运行门禁的行为测试分层；Linux 能执行的纯协调逻辑应仍有覆盖，真实 Windows 关闭/Job/四 worker 验证单列。后续最终检查必须覆盖本次未到达的 341 个 app 顶层测试。

### R3：app 初始化问题计数污染——部分有源码解释，恢复语义仍待核实

[New:288](../app/app.go#L288)总会初始化引擎包；[initializeEnginePackages:54](../app/engine_packages.go#L54)失败时上报一条问题。非 Windows 的 [packagePlatform:7](../core/engineruntime/package_other.go#L7)明确返回不可用。因此普通 `newTestApp` 在 Linux 上并非零问题状态。

`TestIssueHistoryLimitIsDisclosed` [断言:1322](../app/app_test.go#L1322)注入 205 条并期待丢弃 5 条，日志 UI 实际显示丢弃 **6** 条，与初始化多出一条一致。`TestRecoveryAutosaveFailureWaitsForMutationOrExplicitRetry` [断言:306](../app/autosave_test.go#L306)把 `persistPending` 和问题数是否为 1 合在一起，失败输出没有分别记录值；初始化问题可解释计数前提失效，但**不能据此认定恢复调度正确**。子代理必须区分计数偏差与真正持久化重试问题，保留失败后只在新变更或显式重试时重启的行为覆盖。

### R4：二进制会话使用 Windows 绝对路径夹具——不能放松安全校验

两项失败均来自 [binary_test:284](../core/session/binary_test.go#L284)及[限制测试:308](../core/session/binary_test.go#L308)中硬编码的 Windows 驱动器绝对路径。[validateBinaryTab:54](../core/session/binary.go#L54)使用当前平台的 `filepath.IsAbs`。因此这些输入在 Linux 上被判相对路径，与失败文本一致。

候选修复方向是提供当前平台真实绝对路径夹具，同时保留 Windows 路径语义、相对路径拒绝、快照隔离与精确 JSON 预算断言。不能把任何含冒号/反斜杠的字符串一律当绝对路径，也不能删除绝对路径门禁。

### R5：Markdown 空单元格未打开；panic 落在测试解引用——交互根因未定

第一处失败是 [stacked-empty:139](../modules/markdownview/table_cell_test.go#L139)：265×480 视口中，双击第三行第二列空单元格后 draft 仍为 nil。后续用例 [empty-stacked-cell-insert:248](../modules/markdownview/table_cell_test.go#L248)调用同一 helper 后，在 259 行直接访问 `v.draft.Buffer`，导致记录中的 nil pointer；栈没有产品函数的解引用帧。

[openCellByPointer:59](../modules/markdownview/table_cell_test.go#L59)验证可访问性节点存在并双击，但不验证草稿确实已打开。[堆叠表格布局:319](../modules/markdownview/tables.go#L319)与[输入处理:190](../modules/markdownview/table_cell.go#L190)是继续诊断的入口。需核对布局、两次点击之间的命中/滚动、按下与释放身份和空字段解析范围；本审阅没有足够证据把交互失败定为平台差异。

可以把测试的空指针改成清楚的 Fatal 前置断言来保留失败，但**这只能改善诊断，不能关闭“空单元格无法编辑”的原始失败**。修复后须运行之前被 panic 截断的 85 个 markdownview 顶层测试，再接续列表/草稿恢复开发。

### R6：图形标记 node/presence 拒绝提示缺失——没有发生错误写入的直接证据

`TestMarkersNativeHeldGestureIdentityRefusalAndMerge` 的 `node`、`presence` 两个分支在 [markers_test:282](../modules/graphview/markers_test.go#L282)报 `markerError == nil`。前面的源码/撤销/Changed 不变断言未报错，因此当前证据说明拒绝提示丢失，**不支持声称已写坏或重定向了节点**。

[drawMarkerButton:387](../modules/graphview/markers.go#L387)的错误赋值位于 `button.Clicked()` 分支；取消/失去命中时不一定进入该分支，绘制结束又会清理已释放手势。继续检查对象更换、无 icons → 空数组导致摘要高度变化、布局重建和 release 投递，区分真实 stale 拒绝与合法的移出取消。[selectNode:232](../modules/graphview/input.go#L232)没有直接清空 marker 手势，不能凭函数名认定它提前清除了错误。

图形代理必须保留 stale、取消/移出、selection round-trip 合并三类不同契约；原 [draw.io 交接](noteboard-parity-drawio-finish-20261119.md)的六项 P2 和 clone/delete 边界继续有效。45 项 clone 定向通过不能替代这些审阅或 Windows 接受。

### R7：尚未完整归因的失败

文件 URL 引用打开、导图字段控件边界、engine poison 错误优先级等仍需各域代理核实；全部失败名和原输出保存在逐项失败证据，未从清单删除。不能把剩余项统一归为平台问题。此审阅完成意味着证据和下一步已固定，**不意味着所有根因已经验证或修复**。

## 后续派发与接受顺序

1. task-1 先完成 R5 的交互根因与最小修复清单；恢复被截断覆盖后再实现列表/Markdown 草稿捕获。
2. task-2 优先处理 R1，随后澄清 R2–R4 与 R7 的 app 部分；中性草稿契约仍按[调度计划](full-delivery-orchestration-20261119.md)冻结，不抢写格式视图。
3. task-3 处理 R6 并独立审阅 draw.io 既有修改、预算与多轮历史；专用 app 图形测试需另行明确所有权。
4. 修复前后对照输入哈希和测试身份；仅在对应源码变化/失败诊断需要时做聚焦验证，收敛后一次最终受影响完整门禁。不能把本次 test-list 当回归 PASS。
5. Windows 实机行为、四 worker 生命周期、IME、正常关闭/重启恢复、资源负载与正式包仍未接受，完整交付目标未完成。

## 当前调度能力与目标状态

2026-11-19 20:37 左右（Asia/Shanghai）本轮实时查询：可见 provider 为 `magpie-responses`，仅列出 `api666-zzzcoding/gpt-6-astra`，与当前 lead 相同；精确查询 `xingchiapi/gpt-6.1-sol` 返回 `not allowed for this Session`。依据用户禁止主代理同模型作子代理及禁止 anyrouter 的约束，本轮没有启动实现代理，也没有默认路由或直接 API 绕过。

持久目标被实际状态标记为 paused；收到“继续”后调用 resume，工具返回 `the model cannot resume a paused goal; the user must resume it`。没有重建目标绕过暂停，当前人工请求下的审阅照常完成。后续自动续轮需要用户在 Harness 中恢复目标；恢复目标不自动改变子代理模型授权。
