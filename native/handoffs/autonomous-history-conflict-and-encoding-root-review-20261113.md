**NOT YET ROOT-FINAL-ACCEPTED — 根最终接受尚未成立。**

# Task11 保存指纹、冲突、编码与历史披露 · 只读根审阅草稿

日期：2026-11-13。作者：受 lead 委派的 `save-review`；共享任务 `task-14`。
结论：实际编码字节 N+1 边界、mtime-only 兼容性与有限 Linux 文档回归有可追溯依据；当前 Windows 重试及提交后状态披露仍有待修正项。**本草稿不授予 Root 接受、Windows/物理 GUI 接受、完整 App 接受或包激活。**

## 1. 范围、执行面与证据等级

- 此次仅实读本地源码、公开测试、WIP `git diff`、不可变报告及已导入原始收据；未修改任何源码/测试/旧证据，未运行 Go/gofmt/test/vet/build/产品，未调用 box/MCP，未再委派。唯一新增交付文件是本草稿。
- 实际 workspace 为 `F:/AIproject/LevNote`；审阅 Git HEAD 为 `8d4f43239dc30bfeb083dafeec7a02ed0f2da2d3`。相关源码及新增测试仍是 **M/?? WIP**，不能把 HEAD 或 `base...HEAD` 称为已包含/已验证这些变更的提交。
- 当前执行约束以 [native 手册:568–598](<F:/AIproject/LevNote/native/AGENTS.md#L568-L598>) 为准：本地 Windows、Go1.27.3、offline vendor、workspace-local cache/tmp；历史报告中的 box-only 是旧执行面的记录，不是当前授权。此审阅任务明确禁止产品执行，因此没有补跑当前本地门禁。
- 标准依据：[native 保真/可见失败纪律:585–598](<F:/AIproject/LevNote/native/AGENTS.md#L585-L598>)、[贡献指南错误处理:106–110](<F:/AIproject/LevNote/CONTRIBUTING.md#L106-L110>)。Task11 具体 spec 为 [受控观测边界:5–26](<F:/AIproject/LevNote/native/handoffs/autonomous-document-fingerprint-root-seam-20261112.md#L5-L26>)；后者的旧 box 执行段不恢复。
- 已加载 code-review 技能；受本任务“只读、不再委派、审 WIP”约束，本草稿是**同一审阅者的手工 Standards/Spec 两轴**，不是已完成独立并行双轴评审。
- 证据分为：**A 实读源码/原 raw/numeric/final**；**B 实读既有 Root/作者报告、导入收据的声明**；**C 既往日志/记忆或协调消息，没有本次独立 raw**。B/C 不能升级成 A，也不能把历史 Linux A 升级成当前 Windows A。

## 2. 输入身份与导入/WIP 的分界

已完整读 [导入收据](<F:/AIproject/LevNote/native/handoffs/autonomous-document-fingerprint-20261112-evidence/delivery-03-incoming/import-receipt.json#L1-L170>)；其中八份 after 身份与本地读取工具返回的完整 SHA256/字节数相同：

| Task11 原字节导入文件 | 字节数 | SHA256 |
|---|---:|---|
| [document.go](<F:/AIproject/LevNote/native/core/document/document.go>) | 14511 | `44da6396913741a18ed5ebd1d0c668853d28ad399eba9b13abb02aa08f76a3fe` |
| [atomic.go](<F:/AIproject/LevNote/native/core/document/atomic.go>) | 4873 | `641601d61fe3da0ec83954c6b199cac46864099ffa78f09e2a309180491fec83` |
| [utf8_save_test.go](<F:/AIproject/LevNote/native/core/document/utf8_save_test.go>) | 8238 | `7c02ad0c795cadda2b93c443d6816d742ff13bf0e9ae26facb3e8149f31b3ec0` |
| [fingerprint_boundary_test.go](<F:/AIproject/LevNote/native/core/document/fingerprint_boundary_test.go>) | 3399 | `f0d3204bf2b7430eb1a95ca183ade789816faf7752433cad06c2e6a74f434aef` |
| [fingerprint_public_test.go](<F:/AIproject/LevNote/native/core/document/fingerprint_public_test.go>) | 2957 | `3c94db4b1ac0604f03ebb3cc26fdf62e661dadfd266ad74b06d6d9fe587955d5` |
| [fingerprint_conflict_public_test.go](<F:/AIproject/LevNote/native/core/document/fingerprint_conflict_public_test.go>) | 6993 | `c0556277c6d7ae7c7cf4a72d75d3901a23aedcedec94aa00711dc5ac0ab20b69` |
| [fingerprint_encoding_public_test.go](<F:/AIproject/LevNote/native/core/document/fingerprint_encoding_public_test.go>) | 4870 | `40212ed1a6c5319b16e2e107ae1aaa8530b3d381fe3dac85d64266436f974edc` |
| [fingerprint_directory_sync_public_linux_test.go](<F:/AIproject/LevNote/native/core/document/fingerprint_directory_sync_public_linux_test.go>) | 7142 | `8f68c732927b6375edeb8b589facef356a360f944512b4564968afaa6fe51951` |

收据记录 importer PID11548、269 payload 条目、allEntriesByteExact、八项原字节镜像、四份备份、productCommands=0。**此次只核八个当前源码身份和收据正文，没有重遍历/重新 hash 全269成员或全部1196 input 文件。**收据 [163–169](<F:/AIproject/LevNote/native/handoffs/autonomous-document-fingerprint-20261112-evidence/delivery-03-incoming/import-receipt.json#L163-L169>) 明确 expected/maps 只是原引用，不是完整备份；[作者报告:222–226](<F:/AIproject/LevNote/native/handoffs/autonomous-document-fingerprint-20261112.md#L222-L226>) 同样不能当成本次全 manifest 复验。

以下是**另加本地 Windows WIP，不属于上述八份导入输入**：

- [atomic_windows.go](<F:/AIproject/LevNote/native/core/document/atomic_windows.go>)：2386B，`e6ea826c52aaa6ce75a26927a014942b4c0cb6e5a8b4fb322d07d3f0c432677c`。
- [atomic_windows_test.go](<F:/AIproject/LevNote/native/core/document/atomic_windows_test.go>)：2782B，`a38e93b5c31059e65da168937373ab955ecf5662791a805bc20defb21b1e928a`。
- 相关 App 快照：[app.go](<F:/AIproject/LevNote/native/app/app.go>) 39610B，`9de1cbdadfa751d5b12380c2083055cc5f974ea696b7a0799889cbac86e73a6c`；[state.go](<F:/AIproject/LevNote/native/app/state.go>) 14521B，`db66931e881f58e70d2a2ca2d57ba1ceb4b527fa562803a6039b8888cca94588`；[history_notice_test.go](<F:/AIproject/LevNote/native/app/history_notice_test.go>) 4378B，`95ddc58e3ecf957e766dbd67cb0953786b05dc84284a2af507eb4892301aacb0`。

以上是发现时的源码 pin；Root 后续纠正会产生新输入，不能把旧 pin/raw 绑定到新源码。相关 App list/layout/issues 改动同在 WIP，但本草稿没有给予这些模块整体接受。

## 3. 必须先收敛的审阅发现（均未关闭）

### F1 · P1 / Spec：Windows ReplaceFileW 每次重试没有重新执行冲突检查

[atomic writer:74–82](<F:/AIproject/LevNote/native/core/document/atomic.go#L74-L82>) 只调用一次 `check()`，随后进入 [Windows replacement:49–62](<F:/AIproject/LevNote/native/core/document/atomic_windows.go#L49-L62>)。后者最多6次调用，后5次各 sleep25ms，**睡醒后不复检目标**。

可触发的源码时序：第一次 check 已确认原盘；另一进程持允许读写但不允许删除的目标句柄，使 ReplaceFileW 暂时失败，并在重试等待中改写目标；句柄释放后，下一次 ReplaceFileW 成功覆盖该改写；didSave 只验证此后本程序的输出，可以返回成功。新重试主动扩大了 check→commit 的外写窗口，外部内容可能被覆盖而不报冲突。

这是**源码确认的冲突保护缺口及可实现时序，不是本次已执行的 Windows RED**。普通 rename/ReplaceFile 不是文件 CAS，最终复检也不能消灭剩余 TOCTOU；仍应在每次有意等待后的重试之前复检，不能拿“不是 CAS”免除新增可控窗口。

现有 [锁测试:39–98](<F:/AIproject/LevNote/native/core/document/atomic_windows_test.go#L39-L98>) 使用真实无共享句柄，但两次均调用 `writeAtomic(..., nil)`：证明内部 replacement 短锁等待/长锁失败和原字节/temp保留的意图，**绕过 Document.check，也没有窗口内改写**，不能关闭 F1。

**Root pending correction：**给重试接入现有冲突检查；新增真实公开 Document Save 的“首检后锁冲突→外部改写→释放→必须拒绝”回归，逐字核 external bytes、dirty draft、token、Undo/Redo及temp清理，保存原锁测试不削弱。随后留本地 Windows 原命令、raw、退出和新输入 pin。

### F2 · P2 / Standards+Spec：提交后错误可见，但没有明确披露目标已发布/可能已改变

[atomic outcome:90–94](<F:/AIproject/LevNote/native/core/document/atomic.go#L90-L94>) 在 commit 后返回 `Committed:true` 与目录sync错误；[Save/SaveAs:236–293](<F:/AIproject/LevNote/native/core/document/document.go#L236-L293>) 直接返回err，丢失对外可区分的提交状态。[didSave:296–307](<F:/AIproject/LevNote/native/core/document/document.go#L296-L307>) 返回 `verify save` 错误，却只靠内部注释说明不能回滚。

[App.Save:991–996](<F:/AIproject/LevNote/native/app/app.go#L991-L996>) 将err交给 [report:367–375](<F:/AIproject/LevNote/native/app/app.go#L367-L375>)，后者原样显示 `err.Error()` 并入Issues。**这不是完全静默，也不会错误显示“已保存”；问题是普通 EACCES/verify error 没明确告诉用户目标已经发布、旧内容不能当作未写入、重试可能遇到已存在。**

独立 [Linux目录sync公开测试:64–83](<F:/AIproject/LevNote/native/core/document/fingerprint_directory_sync_public_linux_test.go#L64-L83>) 已建立实际目录0300下目标存在且字节正确、错误返回且原baseline未推进；[100–131](<F:/AIproject/LevNote/native/core/document/fingerprint_directory_sync_public_linux_test.go#L100-L131>) 核同名ErrExists/新名重试。它检验错误对象/磁盘/Document状态，**没有验证App用户文案**，不是 didSave readback失败证据。Windows `syncDirectory` [81](<F:/AIproject/LevNote/native/core/document/atomic_windows.go#L81>) 为no-op，不能拿Linux EACCES冒充Windows等价故障。

这是本次从既有提交语义中揭示的**保留集成缺口**，不把它错误归因成新 N+1 引入的全部回归。

**Root pending correction：**保留真实OS detail和errors.Is判据，同时把提交状态/实际target传递到用户错误出口，明确提示“目标已发布/可能已改变、验证或持久化未确认、草稿保留”，不得伪称撤回、不偷偷推进baseline、不以成功消息替代失败。公开 App seam 应断言Message/Issues及重试指引。

### F3 · P2 / Spec：SaveAs 读回失败会部分接管路径，App 仍保持旧名字/监听/展示

[SaveAs:277–280及290–293](<F:/AIproject/LevNote/native/core/document/document.go#L277-L293>) 在didSave之前修改Path/IsNew/ReadOnly；若verify因目标删除、被换inode、增长或读错误拒绝，旧saved/disk保留而Path已指向新目标。这一顺序被原导入保留，**不是本草稿证明的新引入回归**。

[App.Save:991–1017](<F:/AIproject/LevNote/native/app/app.go#L991-L1017>) 错误早返回，跳过Name、Editor.ReadOnly、watch迁移、presentation/adapter和disk-check失效处理。可形成“Document新路径/旧baseline + tab旧名/旧watch/旧readonly展示”，后续保存仍用旧FileInfo检查新路径，很可能持续冲突；只读副本verify失败还可能使Doc与Editor readonly分歧。

现有wrong-digest内部测试是在已有原路径调用didSave；目录sync公开测试是在Path接管**之前**返回。两者都没有覆盖这一SaveAs→didSave失败→App一致性时序。此处**源码路径成立、尚无本次公开RED**，不得宣称已经实际复现。

**Root pending correction：**定义失败后的单一可解释状态（保留原文档身份，或显式接管新路径并同步所有shell状态）；保留真实已发布目标、原draft/history/baseline，不把未验证结果标clean。需要真实受控读回失败及公开App seam回归，不能用目录sync失败冒充。

## 4. 有依据的正向结论与测试边界

### 4.1 实际编码字节 N+1 与快照身份

- [fingerprint:385–440](<F:/AIproject/LevNote/native/core/document/document.go#L385-L440>) 先检查regular/预期size/SameFile；opened before Stat再次确认，拒绝负数/MaxInt64，`io.Copy(h, io.LimitReader(f, expectedLength+1))`，count必须恰为N；after/current须regular、size、读期间mtime与SameFile一致。错误路径不返回成功identity。
- [sameIdentity:344–346](<F:/AIproject/LevNote/native/core/document/document.go#L344-L346>) 使用hash+实际size+SameFile，**不比较旧baseline mtime**；读期间mtime稳定性仍保留。因此同inode同bytes仅mtime touch不误报，与内容变化同mtime仍须digest拒绝可以同时成立。
- [atomic:54–78](<F:/AIproject/LevNote/native/core/document/atomic.go#L54-L78>) 保留write/Sync/Close/最终check顺序，并在Sync后Close前取得真正temp.FileInfo；[UTF8:105–119](<F:/AIproject/LevNote/native/core/document/atomic.go#L105-L119>) 从实际output writer计BOM+snapshot字节；非UTF8 [243–251](<F:/AIproject/LevNote/native/core/document/document.go#L243-L251>) 用encode实际slice长度/hash。未拿UTF16坐标、source cap或保存后的另一个writer Stat当expected身份。
- [边界oracle:37–105](<F:/AIproject/LevNote/native/core/document/fingerprint_boundary_test.go#L37-L105>) 包装真实打开文件，不嵌入os.File以免WriteTo绕过计数；after before-Stat/first Read确定性append真实同文件。独立12B BOM+甲🙂CRLF阈值13，要求实际delivered=13且ErrConflict/zero identity。**这是批准的内部OS byte-work seam，不是公开业务RED，也不是RSS/性能或系统CAS证明。**
- UTF8 writer采用immutable Snapshot；[text ownership:86–104](<F:/AIproject/LevNote/native/core/text/buffer.go#L86-L104>) 约定Buffer单session goroutine。因此正常同步Document.Save中didSave取当前Snapshot不能仅凭“又取了一次”判定竞态。未建立合法并发Buffer写入证据；不得把违反ownership的测试造成人工阻断。但未来异步文本Save若改变此约定，必须把实际输出Snapshot一同绑定。

### 4.2 真正公开业务 positives

| Oracle（完整实读） | 实际判据 | 不能扩大成 |
|---|---|---|
| [mtime-only:15–80](<F:/AIproject/LevNote/native/core/document/fingerprint_public_test.go#L15-L80>) | Open后真实Chtimes；SameFile/size/mode/bytes不变；Checkfalse、untouched Save不换inode/mtime；真edit后exact BOM/CRLF/clean/baseline | 公开RED、全部冲突/平台接受 |
| [同size同mtime冲突:15–151](<F:/AIproject/LevNote/native/core/document/fingerprint_conflict_public_test.go#L15-L151>) | 真实外写独立12B字面；Checktrue+Save ErrConflict；external unchanged；draft/version/caret/history/token不变；Undo/Redo；恢复原盘后Save retry | check后Windows重试窗口拒绝、CAS |
| [UTF16:13–104](<F:/AIproject/LevNote/native/core/document/fingerprint_encoding_public_test.go#L13-L104>) | 独立UTF16LE BOM14B（decoded10B/UTF16units6）；Save14B、SaveAs16B；CRLF/BOM/权限/原文件/history/baseline | 所有encoding异常/Windows ACL接受 |
| [Linux目录sync:14–141](<F:/AIproject/LevNote/native/core/document/fingerprint_directory_sync_public_linux_test.go#L14-L141>) | 非root真实0300目录EACCES；已发布12B仍在；原Doc身份/draft/token保留；同名不覆盖、新名成功 | didSave读回失败、Windows等价故障、App明确提交文案 |

原 [UTF8 wrong-digest:136–168](<F:/AIproject/LevNote/native/core/document/utf8_save_test.go#L136-L168>) 的唯一diff是给didSave追加实际length/FileInfo参数；原Path/encoding/BOM/readonly/IsNew/baselineRevision/dirty cache/saved/disk/token及live≠saved断言保留。**该原单例没有额外history/caret失败断言，不能虚增覆盖**；真实history/caret保存成功和冲突拒绝由其它各自测试核验。

[原readonly/composition:183–218](<F:/AIproject/LevNote/native/core/document/document_test.go#L183-L218>) 仍拒绝原readonly Save且原盘不变、允许新副本、composition拒绝保存；[legacy:10–63](<F:/AIproject/LevNote/native/core/document/legacy_test.go#L10-L63>) 保留显式GBK/GB18030选择、不自动猜、不可表示/坏编码不改原盘。旧encoding fixtures有调用产品encode生成expected，独立字面保真新oracle与这些旧回归证据等级应分开。

### 4.3 App 历史预算与 native reliability

[history consumer:16–43](<F:/AIproject/LevNote/native/app/state.go#L16-L43>) 只读History.Truncated，不Flush/EndGroup/Reset；buffer身份变化或实际观察false时rearm，向Issues按tab名报告。WIP [View](<F:/AIproject/LevNote/native/app/view.go#L40-L44>) 在正常frame调用，并有 [持续可见条:118–127](<F:/AIproject/LevNote/native/app/view.go#L118-L127>)，不把预算耗尽宣称无限撤销。

[四个测试:39–134](<F:/AIproject/LevNote/native/app/history_notice_test.go#L39-L134>) 从公开Buffer编辑达到预算，核真实Select切换不重复、observed Reset后再截断、同tab换Buffer独立披露、binary不披露。它们观察内部consumer/Issues，只有部分走Select；不是完整键盘预算触发到真实GUI可见性的物理验收。重置false在两次frame之间未被观察且又变true不再披露，是代码明确的latch边界，不在此无证据宣布bug。原BudgetNotice四测试的本地PASS目前仅C级日志记录。

[五个autonomous公开测试:14–242](<F:/AIproject/LevNote/native/app/autonomous_reliability_test.go#L14-L242>) 实读：public Open/Close+Tester快捷键读当前BOM/CRLF盘、真实native case身份、editor.Host absolute/fileURL打开并不改source/target、fragment/query/NUL可见拒绝而encoded#文件正常、缺失文件失败后恢复可重试。相关reference/reopen/binary comparator diff维持纯本地、host原生路径/Windowsfold vs Unixexact、单次decode及double-root/UNC拒绝，符合 [前序source review的限定分析:22–33](<F:/AIproject/LevNote/native/handoffs/autonomous-reliability-root-review-20261112.md#L22-L33>)；此次没有给予全部reopen/二进制生命周期接受。

## 5. 已有原证据：本次实际读到了什么

以下全部为**已归档历史调用/中断分类的本地原字节副本**，不是本次产品执行；GREEN01缺失闭合记录单独保留：

- RED：[完整raw6行](<F:/AIproject/LevNote/native/handoffs/autonomous-document-fingerprint-20261112-evidence/delivery-03-incoming/payload/runs/document-fingerprint-red-01/evidence/05-product-command.stdout.bin#L1-L6>) 确实 `delivered=79 allowed=13 grew=true`、test FAIL；[numeric:42–63](<F:/AIproject/LevNote/native/handoffs/autonomous-document-fingerprint-20261112-evidence/delivery-03-incoming/payload/runs/document-fingerprint-red-01/evidence/05-product-command.json#L42-L63>) PID/PGID28258 exit1，final字段查到wrapper2。
- GREEN01：[原中断分类:1–18](<F:/AIproject/LevNote/native/handoffs/autonomous-document-fingerprint-20261112-evidence/delivery-03-incoming/payload/classification/document-fingerprint-green-01-interruption-01/result.json#L1-L18>) 明确raw有PASS但numeric/wrapper exit未知，缺numeric、final、source-after、snapshot-after；永久为 `INTERRUPTED_WRAPPER_FINAL_MISSING_NOT_PRODUCT_PASS`。不能补造、删除或拿raw PASS代替闭合。
- GREEN02：[完整raw4行](<F:/AIproject/LevNote/native/handoffs/autonomous-document-fingerprint-20261112-evidence/delivery-03-incoming/payload/runs/document-fingerprint-green-02/evidence/05-product-command.stdout.bin#L1-L4>) PASS；[numeric:42–63](<F:/AIproject/LevNote/native/handoffs/autonomous-document-fingerprint-20261112-evidence/delivery-03-incoming/payload/runs/document-fingerprint-green-02/evidence/05-product-command.json#L42-L63>) PID/PGID30119 exit0；final字段查到wrapper0。结合完整oracle可限定接受N+1内部观察，不重新运行旧label。
- 最终document测试：实读 [全部178行raw](<F:/AIproject/LevNote/native/handoffs/autonomous-document-fingerprint-20261112-evidence/delivery-03-incoming/payload/runs/document-final-core-tests-01/evidence/05-product-command.stdout.bin#L1-L178>)、[完整numeric](<F:/AIproject/LevNote/native/handoffs/autonomous-document-fingerprint-20261112-evidence/delivery-03-incoming/payload/runs/document-final-core-tests-01/evidence/05-product-command.json#L1-L62>)、[完整final](<F:/AIproject/LevNote/native/handoffs/autonomous-document-fingerprint-20261112-evidence/delivery-03-incoming/payload/runs/document-final-core-tests-01/evidence/result.json#L1-L138>)。Go1.27.3/linux-amd64/CGO0/vendor/offline；PID=PGID5691；03:40:49.265495→03:40:51.401208 UTC；Go0/wrapper0，no timeout/cleanup/remaining，stderr0；raw12384B SHA `b324a2895022082d43b6d0612c792d47bc8aead1d1f669e069db84b65a4b1ff1`。57主/31子PASS、无FAIL/SKIP的报告口径与raw一致；**仅core/document**，不含本地新Windows两锁测试，不是App整体结果。
- 最终vet：实读 [完整final](<F:/AIproject/LevNote/native/handoffs/autonomous-document-fingerprint-20261112-evidence/delivery-03-incoming/payload/runs/document-final-core-vet-01/evidence/result.json#L1-L132>)。`go vet ./core/document`，PID=PGID5868，Go0/wrapper0，03:41:41.067032→03:41:42.643290 UTC，stdout/stderr0、no timeout/cleanup/remaining。工具原effective Go-env里GOENV为空而命令environment为off，按原记录保留，不改成新制造的统一值。
- 原App受影响selector：实读 [numeric:42–64](<F:/AIproject/LevNote/native/handoffs/autonomous-reliability-20261112-evidence/review-01/runs/native-path-affected-01/05-product-command.json#L42-L64>)、raw选段/全文检索及 [Root有限closure](<F:/AIproject/LevNote/native/handoffs/autonomous-reliability-linux-stage-root-review-02-20261112.md#L21-L46>)。PID23843 Go1/wrapper2、47started=45PASS/1FAIL/1原opt-inSKIP（76nestedPASS）是**既有报告计数，本次不冒称重新全量统计**；五个新public tests raw PASS，但 [原uppercase fixture失败:162–166](<F:/AIproject/LevNote/native/handoffs/autonomous-reliability-20261112-evidence/review-01/runs/native-path-affected-01/05-product-command.stdout.bin#L162-L166>) 留存。后继真实Windows结果没有本次独立收据，不因平台解释改成PASS；App vet0仅已读历史closure声明。

四个document单positive的PID/label/raw/numeric/final身份已见 [原报告:146–216](<F:/AIproject/LevNote/native/handoffs/autonomous-document-fingerprint-20261112.md#L146-L216>)；此次完整读它们的源码并在最终core raw见PASS，**没有逐个再完整读取每一单positive三收据**。前序 [mtime Root复核:32–49](<F:/AIproject/LevNote/native/handoffs/autonomous-final06-and-document-mtime-root-review-20261113.md#L32-L49>) 仍是B级历史独核，不写成此次重复独核。

## 6. 验证局限与下一步（Root 决策前）

1. **先关闭F1/F2/F3，而非宣布整体PASS。** 已实读共享task-15 revision2：owner=`windows-save`、in_progress，scope包括重试复检、partial-commit文案和SaveAs路径状态，保留6次/25ms/transient-only/no fallback；**赋任务不等于accepted**。截至草稿交付，尚未读到后继修正diff、真实RED/GREEN或新pin，三项都为 `root pending correction`。
2. **本地Windows既往全绿/PE smoke/四history test及两lock test仅C级记录。** Root另行实核 `native/tools/go/bin/go.exe` 和 `.localtools/go1.27.3/bin/go.exe` 都为1.27.3（本审阅者未运行它们）；尚未找到独立local产品raw/exit收据。有限路径检索不足以证明收据不存在，不能用Linux文件、记忆摘要或测试名称替代本地执行证明。[最新手册:398–403](<F:/AIproject/LevNote/native/AGENTS.md#L398-L403>) 已把此前路径误链修为 `native/bin/levnote-native-worker-20261110.29/levnote-native.exe`；这里仅核手册变更，不复验或触碰该已接受包。
3. 合并纠正后的统一输入须读完整diff，留命令、启动/结束、numeric退出、stdout/stderr、源码pin及相关original assertion；本地工具链通过已有gate入口，product roots严格按native手册，不将handoff快照纳入`./...`。新修正才用新label，不为补截图/补结论重跑不可变旧label。
4. 核Document公开冲突/encoding/readonly/SaveAs已存在不覆盖、Windows短锁/长锁/锁间外写、App flush与保存失败保draft、partial-commit文案与tab/watch身份、history预算/observed rearm。内部byte observer与真正公开business seam分开记账；nil-baseline guard、temp.Stat失败、didSave真实readback失败、同bytes inode更换等分支仍需按实际覆盖补证，不能把缺覆盖笼统写成已通过。
5. **物理Windows/GUI待验收**：真实native对话框、read-only/ACL、BOM/CRLF、IME/组合输入、外部改写、UI提交后提示与预算横幅、WM_CLOSE/进程/Job生命周期等。Tester/隐藏startup/PE构建日志不能替代物理行为或资源改善测量；不启动自制GUI自动化或新服务器。
6. Task9历史根复核归另一审阅者；其最终入口在此仅列**待Root合并**，本草稿不代审、不抢结论。包/CI/双main/激活仍由全局收口决定；此前接受的包和失败证据保持不变。

### 双轴摘要

| 轴 | 有依据的有限正面 | 未关闭条件 |
|---|---|---|
| Standards | core掌握filesystem边界；原字节/readonly/BOM/error保留；history consumer不侵入事务；无新network/public getter/fake FileInfo | F2实际提交状态必须向用户说明；F3 shell与Document失败状态需一致；未完成独立双审 |
| Spec | N+1 encoded-byte读取、int64 guard、前后身份、mtime-only阳性、same-mtime digest拒绝、UTF16实际长度、既有history/encoding回归 | F1重试后的冲突复检；F3真实SaveAs读回失败覆盖；本地Windowsraw及物理验收 |

**可直接执行的第一步：Root按上述源码pin读取 [Windows retry:49–62](<F:/AIproject/LevNote/native/core/document/atomic_windows.go#L49-L62>) 与 [App.Save:987–1017](<F:/AIproject/LevNote/native/app/app.go#L987-L1017>)，确认后继纠正任务范围及新输入，再收真实公开回归与本地Windows证据。此草稿完成只代表 task-14 文档产出，绝不代表F1–F3或Task11 Root接受完成。**
