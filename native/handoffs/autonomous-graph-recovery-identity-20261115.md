# 图形恢复输入所有权与普通等字输入通知 · Task58 交接

## 任务状态（type:state）

- BASE：`a1e6dc810fd4847b4204b1703b88b99a49b38d14`；实施者 `graph-recovery`，Task58 revision3。原固定 [spec01](<F:/AIproject/LevNote/.localtools/graph-recovery-identity-spec-20261115-01.md>) 保持20行5790字节，SHA256 `57ea665b4d4b2578790467f5f3d436242e72d6da462a4f95cb74424e64964714`。
- 后继输入识别及旧错误隔离已完成；23个 graph 与8个 Editor 测试逐项执行后，通过唯一一次精确31名双包联集。全部为公开接口行为测试，无 Skip。联集是 focused 证据，不替代 root 的最终受影响全套门禁。
- control26 的 ordinary same-literal Type 真正 RED 触发 `GR58-ACCEPTED-EQUAL-INPUT-05`；[scope02](<F:/AIproject/LevNote/.localtools/graph-recovery-identity-20261115/scope-02.json>) 及 [bridge03](<F:/AIproject/LevNote/.localtools/graph-recovery-identity-20261115/bridge-03.py>) 已由 root 全文读取；production 仅额外允许 Editor 原等字成功分支45字节通知。
- 五文件的最终字节、raw blob OID、范围与原 oracle 保护由桥03 `freeze-01` 登记；root canonical Git OID、独立 Standards/Spec、唯一最终 test/vet/build、提交与双远端交付由 Task59负责。CI inactive、应用 NO-GO、Task35 HELD、`.29` 已接受包及旧平台入口保持。

## 目标与公开边界（type:goal）

`RecoverDraft` 只复制发起恢复时的待确认字段/未添加标签字节及既有信封。同步 Host 回调中接受的新输入属于后继所有者，即使最后 text/version/selection 完全相同，旧恢复也只处置发起输入；发起恢复的旧错误只返回/报告，避免附着到回调安装的替代输入。

**不同text/version/selection与既有owner/release/main Buffer guard不回归**。

- graph 测试为 external `package graphview_test`：既有 `MindmapAdapter().Create(editor.Host{Buffer, Path, RecoverDraft, Report, Changed})`，返回的 `editor.View.Draw/Release`、导出 `RecoverDraft/CompositionActive/Warnings`、公开 `editor.OutlineNavigator` 与真实 `ui.NewTester` Click/Type/Compose/Command/Key/clipboard。固定有限 `.mindmap` 是本组实际运行 fixture；共享函数修复没有宣称逐个 draw.io 页面或物理 IME 已验收。
- Editor 测试为 external `package editor_test`：`editor.New(Buffer)`、真实 `ui.NewTester(Editor.View)`、既有 `OnChange/OnError`、公开 Buffer bytes/version/selection/history/snapshot/group/token、`MarkedRange/Flush`；不触及私有 Editor 状态或 handler，也不加 API、identity/token 抽象、reflection、unsafe 或 toolkit-tree 观察。
- 自绘输入的 accessibility `Value` 未绑定文本。接受证据来自真实事件、CompositionActive、公开 Buffer 状态、clipboard 与 literal Host 信封，而不是以空 `Value` 推断内容。
- Host 按值复制；在测试闭包里重新赋值外部 Host 并非挂载 view 的 main Buffer replacement 入口。本组保留主 Buffer身份 guard，未伪造这个路径。公开 RecoverDraft 是约定 seam；按钮外层既有 `setDraftError/setTagError` 接线保持原样。

## 实施与精确文件

| 文件 | 最终字节 | SHA256 |
| --- | ---: | --- |
| [draft.go](<F:/AIproject/LevNote/native/modules/graphview/draft.go#L361-L434>) | 13580 | `f21441138e712cd7f9df2e8b7b88f0cb05d6972b4ee2d1362587396aa40801e2` |
| [recovery_identity_test.go](<F:/AIproject/LevNote/native/modules/graphview/recovery_identity_test.go>) | 50482 | `6273d7e4a315fd37b27758f7bec742d0a5bd9592c58faeb6eb4d4f5956e1b14d` |
| [editor.go](<F:/AIproject/LevNote/native/ui/editor/editor.go#L371-L379>) | 26616 | `45a59a4f0a1c6a77aec5972708ab53dd37932f3217a6f7204423714fb7419e2a` |
| [input_acceptance_test.go](<F:/AIproject/LevNote/native/ui/editor/input_acceptance_test.go>) | 15368 | `75cb96d04eeda2c589dfb4e82e1160272cb501a437e48897e17881e11d941872` |
| [本交接](<F:/AIproject/LevNote/native/handoffs/autonomous-graph-recovery-identity-20261115.md>) | 以冻结登记为准 | 自身值由外部冻结回执登记，避免循环摘要 |

图形 production 全部差异位于 `RecoverDraft`。字段与标签临时 `OnChange` wrapper 在调用 Host 前安装，先记录接受，再转发既有 handler；每个 wrapper 以 defer 恢复原 handler，覆盖嵌套恢复与错误返回。分别结合既有 identity 判断 field/tag；成功独立清理未被替代的发起输入，仅两者及主 Buffer/release epoch 均相同才 Release。错误原对象返回；所有权变化时仅 `report(err)`，不把旧失败塞入新 draft。

Editor production 仅在原 `ReplaceText` 等字成功路径 `ensureCaret()` 之后、`Invalidate()` 之前插入：

```go
if e.OnChange != nil {
    e.OnChange()
}
```

原 bytes/version 变化或 wasComposing 的 `changed()` 分支原样；无 blink/cache reset 扩展。ReadOnly、UTF-8/size admission、foreign group、composition-buffer/token、failed close 的 early-return 及既有 API 原样。普通等字/空字接受后回调看到已收尾的 selection/history/composition 状态；失败输入没有接受通知。

## Oracle、真实 RED 与弯路（type:dead-end）

首 oracle 固定 root `recovery-root`、child `recovery-child`、emoji/CRLF、未知节点/顶层元数据及原 number 字面量。主 Buffer 固定原 bytes/version0、reverse selection `{Anchor:23,Head:7}`、undo0/redo0/no group、Host Changed0；恢复标题固定 `recovery.mindmap-节点草稿.txt`，field-only/tag-only/combined 信封逐字断言。[oracle03](<F:/AIproject/LevNote/.localtools/graph-recovery-identity-20261115/oracle-03/oracle.json>) 的5863字节 projection SHA256 `6d29724e11d9bf86e0acca144fbb09d776b7b18566e383760f08e11c7ccc2259`，冻结时间 `2026-11-14T19:08:54.631216+00:00`。

| 阶段 | 实际结论与后继 | 原始 stdout SHA256 |
| --- | --- | --- |
| [red04 raw](<F:/AIproject/LevNote/.localtools/graph-recovery-identity-20261115/red-04/stdout.raw>) | callback 接受同字新 tag composition 后 `RECOVERY_REPLACED_NEW_TAG_COMPOSITION`；unchanged oracle03 的 green02 通过 | `a2138c4b74072341a464d8f01e2ead14ce5ec038bb2d01f4af80a7accf88db0c` |
| [control04 raw](<F:/AIproject/LevNote/.localtools/graph-recovery-identity-20261115/control-04/stdout.raw>) | `RECOVERY_REPLACED_NEW_FIELD_COMPOSITION`；增加 field-local observation 后 control05 通过 | `732816e614ebef1c9b247d53004408dc0cb4b3fdbdca5cdc077c0127c38c9b1d` |
| [control06 raw](<F:/AIproject/LevNote/.localtools/graph-recovery-identity-20261115/control-06/stdout.raw>) | `RECOVERY_ATTACHED_OLD_FAILURE_TO_NEW_TAG`；owner-aware 错误分支后 control07 通过 | `9ce18cefa61ffae8cfa24393500e47e4d7d2c72e125aadbedaa0e5b5e2bc91a1` |
| [control26 raw](<F:/AIproject/LevNote/.localtools/graph-recovery-identity-20261115/control-26/stdout.raw>) | 非composition同字 Type 已接受并恢复同一end selection、copy为“样”，仍被恢复清掉；GR58-05精确补丁后 unchanged fixture control27 通过 | `e0c4dcf98f543463bd733211c8aea86aa024e6a3397438a0c25e16d4e09b2fbd` |

原失败及观察者修正证据全部保留，不覆写或归类成通过：red01 为未绑定 `Value` 观察错误；red02 为非锚定 regex setup，零产品执行；red03 为初始 genuine RED，但 green01 observer 误期望0 warnings，root `GR58-WARNING-OBSERVER-02` 将真实既存兼容2条纳入 oracle03 后重跑 BASE red04。control20 为两条既存指导误加句号，`GR58-NESTED-GUIDANCE-OBSERVER-03` 只移除指定两个句号，control21通过。control24误删 Release 后既存 pending-tag 指导，`GR58-RELEASE-GUIDANCE-OBSERVER-04` 只将最终 expected warnings 恢复为3，control25通过。所有产品/信封/ownership 断言保持。

## 已执行公开控制（type:progress）

[精确单项映射与 raw 身份](<F:/AIproject/LevNote/.localtools/graph-recovery-identity-20261115/union-02/individual-green-coverage.json>) 登记完整31名、各成功 label/command/result/raw SHA；各名先逐项通过，再执行联集，不批量预写后再寻找实现。

| Graph 成功阶段 | 覆盖行为 |
| --- | --- |
| green02、control01–03 | 首同字新 tag composition、no pending、no Host保留、原error/retry |
| control05、07–11 | field同字新composition、错误不附新tag、同字确认、空tag composition、absent field空composition、field失败保留 |
| control12–14 | selection-only、不同text/version、不同field text/new composition |
| control15–18 | field/tag独立关闭、失败admission不成为owner、field同字确认 |
| control19、21–23 | replacement field error、nested field/tag handler还原、replacement tag/node error |
| control25、27 | public Release epoch错误隔离、非composition same-literal Type保留及后续恢复 |

| Editor 成功阶段与 exact test | 覆盖行为 |
| --- | --- |
| control28 `TestEditorInputAcceptanceSameLiteralTypeNotifiesWithoutMutation` | 等字普通 Type 通知1次，bytes/version/history/最终UTF16selection完全相同，Flush不重放 |
| control29 `TestEditorInputAcceptanceInvalidUTF8DoesNotNotify` | 原 `ErrUTF8` 一次，接受通知0、状态不变；有效retry通知1次 |
| control30 `TestEditorInputAcceptanceSizeFailureDoesNotNotify` |4-byte limit失败原SizeLimitError且通知0；边界合法字节retry通知1次 |
| control31 `TestEditorInputAcceptanceReadOnlyDoesNotNotify` | readonly同字/变化/空字/Compose全部0 callback，选区与源状态保留 |
| control32 `TestEditorInputAcceptanceChangedTypeNotifiesOnce` | 变化 Type一次已收尾通知、version1与一个undo transaction；Undo/Redo无新增输入通知 |
| control33 `TestEditorInputAcceptanceEmptyTypeNotifiesWithoutMutation` | 真正 `Type("")` 接受一次、空bytes/version0/history0/selection0不变 |
| control34 `TestEditorInputAcceptanceForeignCompositionDoesNotNotify` | 他owner composition保留，Type/Compose原ErrGroup且通知0，显式cancel后retry一次 |
| control35 `TestEditorInputAcceptanceLostCompositionDoesNotNotify` | 已接受Compose的token经公开ResetHistory失效，foreign ordinary group保留；Type/Flush/Escape原ErrComposition而无接受通知，显式cancel后retry一次 |

## 唯一 focused union 与冻结入口

[union02 helper](<F:/AIproject/LevNote/.localtools/graph-recovery-identity-20261115/union-02.py>) 279行17164字节 SHA256 `d9f510cbc00342677fd0ec05b925b7c33213b4648a11a7f10f60ff3a3132ebd9` 已由 root 全文实际读取后准以相同字节单次执行。它只加载已读桥03 lower definitions；原helpers/bridge01/02/union01/roothelper main不执行。既有 historical `all/parents/nested` 与新 per-name receipt 均重新核真实JSON单名/终态；最后捕获双包31 exact names。

- [union result](<F:/AIproject/LevNote/.localtools/graph-recovery-identity-20261115/union-02/result.json>)：732行25844字节 SHA256 `1674e5eeedb213c9af106d9b780b33c0614edb4bfcd0404e7e9e6e76cdcb82cc`；exit0、31 RUN/31 PASS/0 FAIL/0 SKIP，graph23与Editor8各包终态PASS一次；`captureComplete/inputsEqual/allPriorChildArtifactsUnchanged/bothPackagesExact/goImagesExact=true`。
- [union raw](<F:/AIproject/LevNote/.localtools/graph-recovery-identity-20261115/union-02/stdout.raw>)：144行32675字节 SHA256 `4a092168e38b18f81d4050ed399754a71f4958aca90f1346287f73b795adcf1a`；[stderr](<F:/AIproject/LevNote/.localtools/graph-recovery-identity-20261115/union-02/stderr.raw>) 0字节。wrapper PID19996/gate14060/Go25696(parent26064)，Go image为pinned本地Go、creation `2026-11-14T21:58:53.124976+00:00`；运行 `2026-11-14T21:58:53.040821+00:00` 至 `2026-11-14T21:58:56.096864+00:00`。Gate推导expected argv，不冒称OS command-line捕获。
- 主产品 tracked∪`git ls-files --others -z`（含ignored）限定八prefix与go.mod/go.sum；BASE1219+两个newtests=1221，aggregate `e6c86fc529679397c5aea4524348bfc4b022f7dc7e9c333422c191030303fb55`，其余1217原产品exact。tracked变化只有两production路径；handoff不在产品aggregate中。before/after同SHA `48b77b5e0f356996044bdebf8440295fb045d523c1c6aa375a69b193d60cc2b9`，prior前后同SHA `54cfeb895b3d5bf5bf7d8eaf42c30d95312dcd21e67db8494a8b478f206be31a`。
- 七tool/module pins、8489历史文件及唯一e80f roothelper例外、1854 prior child文件、534 excluded与CON metadata、首oracle03、control26原projection/raw、所有已执行Editor whole-byte prefixes均保护。excluded/CON只比较metadata，正文未open/hash/stage。
- [新test格式回执](<F:/AIproject/LevNote/.localtools/graph-recovery-identity-20261115/union-02/new-test-format.json>)：只对两个newtests运行pinned gofmt无`-w`，stdout完全等于各冻结wholebytes；每个文件恰好一个final LF。八个保护format债文件和所有原test未格式化。

后续冻结调用仅登记字节/范围，零产品执行；已消费runner与证据不改、不重放：

```bash
"/c/Users/canace/.dsh/dsh-runtimes/dsh-primary-runtime/dependencies/python/python.exe" .localtools/graph-recovery-identity-20261115/bridge-03.py freeze-01 freeze
```

## Root 接续第一步

读取冻结登记的五file raw、production diff、first-oracle preservation，与本篇和union02原始流逐项比对；在子代理 `allwriters STOPPED_WRITING` 与 fresh Task58 get/CAS complete之后，依 Task59进行独立双轴审阅，再执行唯一受影响最终test/vet/build。Root单独管理canonical Git OID与exact五file提交/CNB+GitHub main readbacks，以及共享memory_log/账本/PLAN；子代理没有Git mutation、platform/GUI/MCP、权限/ACL/token变化、shared-memory writes或subdelegation。
