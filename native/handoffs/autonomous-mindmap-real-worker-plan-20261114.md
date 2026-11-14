# 四引擎实际 worker 门禁计划：复用既有 mindmap 双入口

日期：2026-11-14。Task27 只读核对与独立计划；执行状态 **NOT RUN / HELD**。本文件是唯一项目写入产物；产品、测试、原 CI 准备、旧包、旧证据保持原样。根代理独立复核后决定执行与有限验收。

## 1. 结论与当前证据层

**mindmap SVG 与 readonly scene 均已有 actual `NewBundled` app-host-path opt-in 测试及独立字面 oracle。缺的是当前候选执行与更强进程/Job 观察证据，不是缺 scene 测试，不需新造底层 driver。** 四引擎成功门禁可直接复用第 2 节的四个实际测试名，其中三项走 App + `NewBundled`，draw.io 一项走 `NewManaged` + 安装的候选引擎。它们不是四项 packaged GUI host 请求，也不是四项 managed-installed 测试。

当前源码基点 `75a5c20ea42d614dc89255ca150a47b35c3ff646` 来自根代理；Task27 未用 Git 独立核定。第 9 节列的是实际读取/复查的磁盘 SHA256，不把提交声明替代磁盘身份。根代理最新交接：Medium integrity8192 下原 gofmt-w 仍是 eight Access denied / 源码1211无修改；原 guard 在 Junction 预备阶段 exit1、0cases；原 packager 重核仍在 line24 NewItem exit1，候选 .32 与脚本 evidence02 absent、0新PE/host/worker。这些原入口原始收据由root独立保留；Task27未重复执行或扩大读取该收据。**三入口仍held，不启动worker；Medium 或 source PASS 不改写为新包 acceptance。** .29 仍是本任务已知的最新 accepted runnable。

[当前原始 source-suite 输出](<F:/AIproject/LevNote/.localtools/final-native-20261114/test-01/stdout.raw>) 的实际 `Action:skip`：

| 实际测试 | 原始输出行 | 该轮实际 packaged request |
|---|---:|---:|
| `TestDrawioManagedRealBundleSmoke` | [3265](<F:/AIproject/LevNote/.localtools/final-native-20261114/test-01/stdout.raw#L3265>) | 0 |
| `TestMindmapWorkerExportBundledHostPathAcceptance` | [7740](<F:/AIproject/LevNote/.localtools/final-native-20261114/test-01/stdout.raw#L7740>) | 0 |
| `TestWorkerExportBundledHostPathAcceptance` | [8314](<F:/AIproject/LevNote/.localtools/final-native-20261114/test-01/stdout.raw#L8314>) | 0 |
| `TestManagedRealOldReleaseThenBundledResetSmoke`（附加旧版/reset） | [9846](<F:/AIproject/LevNote/.localtools/final-native-20261114/test-01/stdout.raw#L9846>) | 0 |
| `TestMindmapPreviewBundledHostPathAcceptance` | [11604](<F:/AIproject/LevNote/.localtools/final-native-20261114/test-01/stdout.raw#L11604>) | 0 |

`RUN` 事件后紧接 `SKIP` 不等于 worker Start。该轮全 source-suite 的 PASS/exit0 与49个 SKIP应分开计；0 actual-run 只针对这些当前候选 opt-in 门禁，非整个项目历史 helper/process 总数。[命令收据](<F:/AIproject/LevNote/.localtools/final-native-20261114/test-01/command-start.json>) 明确该轮仅本地 source evidence。

[冻结 Task20 局部计划](<F:/AIproject/LevNote/native/handoffs/autonomous-native-ci-preparation-20261114/REAL-WORKER-PLAN.md#L17-L34>) 当时只列 Markdown old-release/reset 与 draw.io 两个 contracts；“这两个测试不覆盖 mindmap”是正确的局部边界，但不据此称当前源码没有 mindmap bundle tests。Task27 已阅读全文 SVG212行、preview256行、Markdown212行、draw.io544行及 Markdown managed328行。原准备保持冻结；本文件补充其覆盖索引，不覆盖它。

技能目录已检查并加载 `code-review`；采用 Standards/Spec 双轴证据分离。其通常要求 fixed-point diff 和并行子代理，本任务是指定单文件只读计划、禁止 Git/再委派，故未执行完整并行 diff review。匹配的本地文件/检索工具已用于核对；本任务没有已运行的新验收流程，技能库不激活未运行计划。

## 2. 四个既有成功入口：名称、env、路线与有限覆盖

| 引擎 / 固定预算 | 实际测试与完整源码 | 唯一该 case opt-in / 额外参数 | 实际路线、请求数与现有检查 | 执行状态 |
|---|---|---|---|---|
| `levnote.markdown-export` / HTML；8MiB source、64MiB payload | `TestWorkerExportBundledHostPathAcceptance`；[app test:27–140](<F:/AIproject/LevNote/native/app/export_worker_bundle_test.go#L27-L140>) | `LEVNOTE_WORKER_BUNDLE_ACCEPTANCE=1` | 完整 bundle copy 旁 test image；注册 source-mode app；实际 `NewBundled`；只替换 SaveDialog。2次实际导出：首次独占发布，第二次成功 worker 后同目标写入失败；原目标/源 BOM+CRLF/状态保持；close/join合约 | NOT RUN |
| `levnote.mindmap-svg` / SVG；16MiB、64MiB | `TestMindmapWorkerExportBundledHostPathAcceptance`；[app test:26–212](<F:/AIproject/LevNote/native/app/export_mindmap_worker_bundle_test.go#L26-L212>) | `LEVNOTE_WORKER_BUNDLE_ACCEPTANCE=1` | 注册 source-mode app + 实际 `NewBundled`；2次请求；准确 manifest identity、字面 XML geometry、三项完整范围 warnings、成功后独占写失败、slot释放、源与UI状态保持 | NOT RUN |
| `levnote.mindmap-preview` / `mindmap-scene-v1`；16MiB、32MiB | `TestMindmapPreviewBundledHostPathAcceptance`；[app test:42–256](<F:/AIproject/LevNote/native/app/preview_worker_bundle_test.go#L42-L256>) | `LEVNOTE_WORKER_BUNDLE_ACCEPTANCE=1` **且** `LEVNOTE_PREVIEW_BUNDLE_CAPTURE_DIR` 非空 | 实际 filesystem readonly属性+BOM+CRLF；注册 readonly adapter；实际 `NewBundled`，包装器仅委托 Export/Close、计数。1次请求；独立三节点两边 Scene；生产 bounded slot投递；大纲隐藏节点诊断、source/fit controls、light/dark/source native Tester图、slot与snapshot释放、关闭无需dispatch | NOT RUN |
| `levnote.drawio-svg` / SVG；32MiB、64MiB | `TestDrawioManagedRealBundleSmoke`；[runtime test:120–544](<F:/AIproject/LevNote/native/core/engineruntime/drawio_managed_bundle_test.go#L120-L544>) | `LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE=1`；绝对 `LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_CANDIDATE`、`LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_EVIDENCE`、`LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_CONFIG` | test image在 evidence root下完整 copy；导入候选第四引擎、显式激活 digest，old3保持；实际 `NewManaged` 从 installed path做1次 `{Index:1,ID:"second"}` 请求；独立 exact SVG/空 warnings/完整 EngineInfo；actual root handle/image/creation/exit观测与Close | NOT RUN |

正常 `go test` 临时 image 不满足相邻完整 bundle要求。测试编译产物仅进 evidence copies，绝不追加到 shipping candidate。表内成功 cases 总计6次 worker requests；数量指完整运行后的既有代码预期，不是当前已观察次数。

Preview test仅检查 capture env非空，使用 `MkdirAll` 和每张图的 `O_EXCL`；**绝对、fresh、evidence-owned目录由外层计划额外强制**，不错误声称源码已经验证其绝对路径。

三项 App测试没有actual PID/creation/held root/Job accounting记录；`RuntimeJoined=true` 是其返回合约标签。Preview capture是 `ui.NewTester` 的 native headless图，而非物理 Windows画面。Draw.io成功 test已有精确 duplicated root handle，但未导出 Job membership/member signal/ActiveProcesses统计，也未包含实际取消/timeout/descendant矩阵。四项均非 host启动/WM_CLOSE/session/relaunch/物理IME/resource/全应用交付。

## 3. 独立 payload、geometry、warning、metadata oracle

### 3.1 Markdown

沿用 [既有字面片段:99–106](<F:/AIproject/LevNote/native/app/export_worker_bundle_test.go#L99-L106>)：`<h1 id="字">字🙂</h1>`、disabled checked task、A/B表头、`<td>&lt;&amp;</td>`、字面 `**字**`、`<p>external</p>`、escaped script。active external anchor及raw script缺席。完整 payload/warnings parity另由 `markdown.ExportHTMLChecked(markdown.Parse(source))`核对；**warning oracle这里是共享实现，不宣称独立 warning literal**。独立片段仍是共享 parser/export parity之外的检查。

两次请求后disk bytes、UTF8/BOM、buffer/version/selection/history/dirty/baseline均保留，首次target bytes不被第二次覆盖。记录 engine version/SHA、HTML bytes/SHA、warning/dialog数、elapsed；现有 App test并未独立断言 Engine.Source/PackageDigest或持有worker handle。

### 3.2 Mindmap SVG（现有独立 oracle，不新增测试）

[字面 XML assertion](<F:/AIproject/LevNote/native/app/export_mindmap_worker_bundle_test.go#L175-L212>)必须仍覆盖：

- SVG namespace `http://www.w3.org/2000/svg`；viewBox `-24 -24 196 144`；width196、height144；2 rect、2 text。
- 正文node `(x28,y28,width92,height40)`，fill `#ff0000`、stroke `#123`；label x74，text fill `#663399`；tspan值 `⭐ custom<& 根<&🙂 [甲& · 乙]`。
- 图片仅字形 `图`，位置 `(112,42)`；XML实体准确；没有 `<image`、`https://`、`<script`。
- 共享 Parse/Layout/ExportSVG parity额外保留，但不从其计算独立期望坐标。
- 三个 warnings按下表准确顺序与完整 wording比较；各 Start0，End=`uint32(len(source))`，按送入worker的无disk-BOM source bytes计。

| 顺序 | Code | Message |
|---:|---|---|
| 1 | `UNKNOWN_KEY` | `node "r" has 1 preserved unknown field(s)` |
| 2 | `UNKNOWN_KEY` | `preserved 1 unknown top-level field(s)` |
| 3 | `IMAGE_PLACEHOLDER` | `1 个节点图片已保留；原生画布不访问文件或网络，显示为“图”占位` |

Engine.ID为SVG ID，version/SHA准确匹配manifest且两次请求稳定；payload经过成功worker返回后，才进行首次发布/第二次独占写失败。未曾调用 in-process view fallback。fixture未知数值 `1.00/2.50/3.00/4.00`、BOM+CRLF和编辑历史逐字保持。第二次target失败不是worker/process失败测试。

### 3.3 Readonly scene（既有三节点独立 Scene）

[既有手写 Scene和warnings:123–141](<F:/AIproject/LevNote/native/app/preview_worker_bundle_test.go#L123-L141>)不从 Parse/Layout/Encode/Decode生成 expected：schema1、layout `logicalStructureRight`、bounds `(0,0,312,160)`。

| Node | 字面 Label | 几何 x/y/w/h | Paint fill/stroke/text | 标志 |
|---|---|---|---|---|
| r | `⭐ 根<&🙂 [甲&]` | 28/60/92/40 | `#aabbccff` / `#11335577` / `#663399ff` | rounded、root=true |
| a | `折叠A` | 192/28/92/40 | `#00000000` / empty / `#ff0000ff` | rounded、collapsed=true |
| b | `可见B` | 192/92/92/40 | all empty | rounded |

r→a：`(120,80)`→`(192,48)`，stroke `#00ff0088`；r→b：`(120,80)`→`(192,112)`，stroke `#12345678`。hidden child缺席，source-only note/link/image reference/unknown fields无 DTO字段；标签、节点计数和paint由上述独立literal比较。实际结果经production Decode，然后与manual Scene比较；这是独立期望geometry，不是另一个独立decoder，也不是记录raw-scene payload hash的测试。

warnings按 `UNKNOWN_KEY(node b)` → `UNKNOWN_KEY(top-level)` → `IMAGE_PLACEHOLDER`，Start0/End=`len(source)`；后者exact message为 `节点图片引用已保留；此只读预览不加载或显示图片，完整引用请在源码中查看`。

[生产slot与文档保留:143–195](<F:/AIproject/LevNote/native/app/preview_worker_bundle_test.go#L143-L195>)：投递后nodes/warnings/callback/source snapshot释放；真正readonly属性仍在；outline target b可定位，hidden产生 `editor.ErrOutlineReadOnlyHidden`；主题/导航/source操作不再Start；source/fitcontrols在760×480 status row上方可用；三张PNG产生bytes/SHA。frame/Tester图之外的physical GUI仍待独立执行。

现有[producer literal test:87–98](<F:/AIproject/LevNote/native/cmd/levnote-mindmap-preview/scene_test.go#L87-L98>)比较完整 authored escaped wire JSON，并做重复帧确定性；[四layout独立geometry test:14–84](<F:/AIproject/LevNote/native/cmd/levnote-mindmap-preview/scene_test.go#L14-L84>)覆盖四layout。这些调用 `serve`，属于source/in-process层；它们不替代本节actual bundle case。

### 3.4 Draw.io

[实际managed exact oracle:403–419](<F:/AIproject/LevNote/native/core/engineruntime/drawio_managed_bundle_test.go#L403-L419>)以源码手写完整 `expectedSVG` 比较bytes，不通过renderer生成want：选second而非first；viewBox `-14 -4 128 88`，w128/h88；rect `(10,20,80,40)`、fill `#dae8fc`、stroke `#6c8ebf`；`two`在 `(50,44.9)`；末尾LF准确；warnings empty。

EngineInfo必须准确等于 `{ID:drawioSVG, Version:candidateVersion, SHA256:candidateExeSHA, Source:installed, PackageDigest:importDigest}`。Import不激活；Activate仅第四项；legacy schema1 old3 load保持state原bytes，old3 selections不变。candidate executable前后hash相同，source前后相同。完整candidate/copy inventory保留另由外层第 6 节检查，test本身只比相关draw.io metadata/workerhash，不扩大成全包身份检查。

## 4. 复用生产 public Managed 与 owned Job 契约

### 4.1 公共路径和源快照

[public types](<F:/AIproject/LevNote/native/core/engineruntime/types.go>)、[Runner:63–181](<F:/AIproject/LevNote/native/core/engineruntime/runner.go#L63-L181>)的正常路线是：

```text
NewPackageManager(bundleHostDir, freshStateDir)
  → LoadError / Snapshot
  → ImportFromReleaseDir(ctx, candidate, fixedEngineID)     // 仅安装，不激活
  → Activate(ctx, fixedEngineID, receipt.Digest)             // 精确选择
  → NewManaged(manager)
  → Export(ctx, fixedEngineID, Source{Size, immutable Data, Page})
  → Close()                                                // 相同cleanup barrier
```

[Import](<F:/AIproject/LevNote/native/core/engineruntime/package_store.go#L140-L231>)重验actual18-file installed package、独占publish，rename后取消保持已安装而未激活版本。[Activate/Reset](<F:/AIproject/LevNote/native/core/engineruntime/package_state.go#L278-L330>)只替换指定selection；损坏state保留，显式全selection repair才替换。未来四引擎 installed/reset扩展可复用此生产路线，**本任务不造新CLI、底层exec/stdio driver、任意exeprovider或hook返回假payload**。

`NewBundled`依据当前test image的 `os.Executable`定位bundle，没有runtime arbitrary path/command覆盖。三项既有App bundle tests显式注入真实 `NewBundled`，draw.io test显式 `NewManaged`，故必须保持路线标签。[export入口](<F:/AIproject/LevNote/native/app/export.go#L24-L29>)在有ExportEngine时走worker；[preview coordinator](<F:/AIproject/LevNote/native/app/preview_worker.go#L39-L67>)只对成功结果Decode并转neutral DTO。

`Source.Size`是UTF8字节；Data为immutable `io.WriterTo` snapshot。Page在admission时复制；仅draw.io允许且要求page，index0..249999、ID≤512B valid UTF8；其他三引擎Page=nil。默认request timeout30s、pipe wait2s、stderr≤16KiB；同Runner单active operation，成功join之后复用；selection、unknownID、invalidPage、closed/busy可在launch前失败。

### 4.2 实际launch/Wait/Job：既有事实和未导出的观察

[生产run](<F:/AIproject/LevNote/native/core/engineruntime/run.go#L49-L165>)：held-file verification后执行固定worker `--stdio-v1`；cwd为固定engine目录；[Windows command配置](<F:/AIproject/LevNote/native/core/engineruntime/process_windows.go#L203-L210>)使用HideWindow、creationflags `0x08000000`，env仅SystemRoot/WINDIR，无PATH、LEVNOTE/config/profilingoverride。每个成功Start都恰好Wait，即使Job attach失败；normal Wait join stdout/stderr copyloops，随后writerDone join。

[Job implementation](<F:/AIproject/LevNote/native/core/engineruntime/process_windows.go#L59-L200>)固定KILL_ON_JOB_CLOSE/no breakaway，清理前枚举至多256实际members、用precise heldhandles验证membership，TerminateJob，然后ActiveProcesses0并等待held member对象signal，最后close Job/member handles。只看到PID消失或active0不足以替代held对象signal。Start到attach间隙由精确root Kill + Wait覆盖，不虚称suspended-before-assignment。

[Runner completion:127–181](<F:/AIproject/LevNote/native/core/engineruntime/runner.go#L127-L181>)在child/writer/Job/bundlehandles清理之后才linearize结果。cleanup failure覆盖为 `FailureCleanup`、Payload=nil、runner permanentlyclosed；所有Close等待同barrier并返回保存的cleanup错误。App [close:383–419](<F:/AIproject/LevNote/native/app/export_worker.go#L383-L419>)joinpackagework、取消preview、lane close并joinexport、clear结果；preview owner失效不会保留releasedviewgraph或触发stale投递。

已有draw.io [afterAdmission/Verify/Start/Wait hooks:319–335](<F:/AIproject/LevNote/native/core/engineruntime/drawio_managed_bundle_test.go#L319-L335>)仅观测真实production run，不替换exe/算法/provider。它从actual `os.Process.WithHandle` DuplicateHandle，QueryFullProcessImageNameW/GetProcessTimes记录image与creation FILETIME，在Wait后GetExitCodeProcess，期望0并closePrimary。尚无explicit `WAIT_OBJECT_0`/Job accounting输出，故actual root identity/exit和Runtime joined合约应分栏。

### 4.3 更强 process gate 的待补字段（不另建driver）

**更强 managed/owned Job 门禁状态：NOT_RUN；本计划和冻结 CI 草案均不自动激活root held入口。** 优先接受现有测试的原范围；若根代理要求更强four-worker进程证据，复用上面的production Managed/ownedJob与既有只观测hook模式，另立审定任务增加观察，而非在Task27实施。每request evidence至少区分：

```text
engineID, selected source/digest/version/SHA, exact expected image
request source-byte hash, payload-byte hash(or decoded-scene oracle), warnings
admission/verify/start/attach/wait counts, UTC order, runtime return, Close outcome
actual root PID + creation FILETIME + duplicated held process identity/image
root WAIT_OBJECT_0 + exitCode, exact Job membership observed after attach
Job ActiveProcesses before/final; bounded captured members and held signals
writer/copy-loop join basis, Job/member/bundle-handle close results
cleanup/fallback attempted + result; protected inputs before/after identities
```

三项App currently `observed=false`的PID/Job字段要写 `UNOBSERVED`，不可填0或从 `RuntimeJoined=true`合成true。draw.io目前已observe的rootimage/creation/exit填actual值；尚无fields保持UNOBSERVED。若normalworker太快而未捕获Job/对象signal，该强门禁保持INCOMPLETE，非补查回收PID。当前production workers不spawndescendants；“本次ownedJob无descendant”与“descendant cleanup路径被实测”分开，后者只能由既有helper测试提供单独层证据。

## 5. 失败、prestart拒收与清理矩阵

以下是生产契约oracle和现有证据归属；未运行的actual四engine失败矩阵保持pending，不创建假测试名，不把helper或内联 `serve`当current shipped-worker失败验收。

| 阶段/fixture | 预期error/result | 启动与清理oracle | 现有层/实际缺口 |
|---|---|---|---|
| unknownID、非draw.io带Page、draw.io缺/非法Page、closed/busy、损坏selection | `FailureUnavailable` / `FailureProtocol` / `FailureClosed` / `FailureBusy` / selection integrity | 本request Start0；Data未读；Payloadnil；无新dialog/target；closed/busy不影响既有ownedwork的join | public admission + unit/helper，不是all4actual失败报告 |
| Size<0、超过各sourcecap、Data=nil、已取消context | `FailureSourceLimit` / `FailureSourceSize` / `FailureCanceled` | preverify/prestart error应Engine empty，Start0；源/disk保持 | [runtime preflight test](<F:/AIproject/LevNote/native/core/engineruntime/runner_test.go#L534-L573>)；真实bundlecase待执行 |
| missing manifest/worker、hash/size/identity或path/reparse错误 | unavailable / integrity，Payloadnil | verification失败Start0；Engine empty；无parser/viewfallback，保持candidate；仅独立故障copy修改 | 既有[Markdown refusal:145–212](<F:/AIproject/LevNote/native/app/export_worker_bundle_test.go#L145-L212>)有两个opt-in；不是mindmap/all4refusalcoverage |
| verify成功后Start/Job建立/attach失败 | unavailable，Payloadnil | Engine可已含verifiedmetadata；successfulStart才要求Wait1；attach失败精准rootKill/Wait，无writer启动；heldhandles全部close | [实际production分支](<F:/AIproject/LevNote/native/core/engineruntime/run.go#L41-L114>)；[helper timeout/attach](<F:/AIproject/LevNote/native/core/engineruntime/runner_test.go#L646-L669>)；未注入真实worker |
| source短/长/虚报WriteTocount/写error | `FailureSourceSize`，Payloadnil | verifymetadata保留；已Start的child/copy/writer/Job完全join；无publish | [exact writer helper cases](<F:/AIproject/LevNote/native/core/engineruntime/runner_test.go#L588-L602>) |
| schema malformed、duplicateID、depth257、warning257或超longwarning | coherent wire `status:error`、no warnings/no payload；runtime `FailureEngine`并保留producer code | productionworker通常exit0表示完整错误帧，非exit1；真实启动必须Wait/Job清理；源不回显/不变 | SVG/preview内联serve tests已有；真实bundle failure尚无本轮证据 |
| scene IDs512/513、真实32MiB escaped-output上限 | 512可用；513或真正escaped-overcap → `output-limit`、nilpayload | 真实预算，不拿tiny-limitseam冒充production32MiB/64MiB门禁 | [preview source/output tests:295–358](<F:/AIproject/LevNote/native/cmd/levnote-mindmap-preview/main_test.go#L295-L358>)是in-process；SVGtinycap仅私有testseam |
| malformed/truncated/extra/wrong-engine/version响应、stdout超cap、stderr或非zeroexit | protocol/stdoutlimit/unexpectedstderr/stderrlimit/workerexit；Payloadnil | 精确heldroot与ownedJob清理；copyloops/writer完成；verifiedEngine可保留 | [frameoutcomes helper](<F:/AIproject/LevNote/native/core/engineruntime/runner_test.go#L306-L343>)；不修改候选worker制造这些输出 |
| activecancel、timeout、Close、Wait后completion前cancel | canceled/timeout；Payloadnil；所有Close同barrier | Start1/Wait1，heldroot与Jobmembers signal、Active0、writerjoin；stale callback不投递，source可达 | [Close helper](<F:/AIproject/LevNote/native/core/engineruntime/runner_test.go#L605-L701>)及[preview controlledtests](<F:/AIproject/LevNote/native/app/preview_worker_test.go#L322-L401>)；actual scene bundle只成功case |
| cleanup自身失败 | `FailureCleanup`，Payloadnil；下一次`FailureClosed` | 记录真实failure；Close持续返回相同barrier错误；既有失败不因fallback清理完成改PASS | [cleanup helper](<F:/AIproject/LevNote/native/core/engineruntime/runner_test.go#L496-L516>)；源与target保持 |
| 成功worker后exclusive-existingtarget写失败 | App visible file-writefailure；原target原bytes保留 | worker已成功join，这不是runtimeFailure；jobpayload/previewslots释放、nodocumentmutation | Markdown/SVGactual成功tests的第二次request（本轮未跑） |

[独立held-descendant oracle](<F:/AIproject/LevNote/native/core/engineruntime/runner_test.go#L345-L483>)在stdout到达、Job终止前验证membership并留handle，Export返回时检验signal；它是repository helper证据，不作为shippedmindmapworker descendant evidence。部分较旧helper调用laterPID检查，第4.3更强门禁不沿用该较弱依据。

全部failure均要求Payloadnil，但**EngineMetadata是否empty由phase决定**：只对preverify failure断言empty；verify成功后运行/取消/cleanup失败仍可保留Engine。warnings亦不能对任意Failure笼统推空；只对上述producercoherent错误fixture按具体oracle要求empty。

## 6. 一次受控未来执行顺序（仅命令计划）

### 6.1 先决条件与隔离

根代理确认packageguard、packager及currentcandidate身份/审定入口后，另立一个唯一 `RUN_ID`。有入口held则本轮保持 `NOT RUN / HELD`，不以外壳、ACL修补、改helper路径、预建被拒目录或重用旧namespace替代原结果。根代理采集的 .32结果是独立证据，本文不追改历史失败。

未来由root审定：

1. 核定当前source inventory、toolchain(项目Go1.27.3 windows/amd64)与候选完整shippinginventory；sourcecommit/磁盘/compileinputs一致。守护入口与五GUI-PE检查完成后再启worker门禁；hash/PE仅身份一致，非publisher签名或执行证明。
2. 使用absent普通localdrive `E=F:/AIproject/LevNote/native/handoffs/autonomous-real-worker-<RUN_ID>`，与candidate不嵌套。建立 `E/app-bundle`、`E/drawio-bundle` 两个完整evidencecopies及 `E/logs`、`E/test-tmp`。candidate所有文件copy前后hash相同；除testimages之外，copy相对member-set/bytes/hash与candidate完全相同。无reparse/linked目录。testimage不得进candidate。**两个disposable tester不是第六/第七shippingPE；五GUI-PE候选checker只检查原host+四worker，tester只列evidence附件。**
3. compile用项目localcache/tmp与offlinevendor；runtime `TMP/TEMP=E/test-tmp`，App的 `t.TempDir/config`完全run-owned；draw.io `E/drawio-config`保持absent至test创建。profile/APPDATA/LOCALAPPDATA如caller采用隔离也只使用该freshrun，不读取/改真实用户配置。
4. `E/drawio-result.json`必须absent、outsidecopy；capture `E/preview-captures`为绝对absent目录，由previewtest创建。draw.io evidence/config均belowE，config与copy互不嵌套；test本身再次检查这些约束。
5. 每次tester launch启动freshcaller，删除该caller继承的全部 `LEVNOTE_*`，再仅加入下表当前case env；清理继承GOselectors，固定offlinevendor/Go1.27.3/CGO0/GOOSwindows/GOARCHamd64/GOAMD64v1/telemetryoff。不设workerpathoverride；productionchild env保持第4.2的最小值。
6. root外层记录exactcommand、cwd、env、UTCstart/end、testimageSHA、stdout.raw/stderr.raw无截断、testerexitCode、超时/cleanup。外层监督沿用已审定owner/job执行方式，不新写底层workerdriver；外层终止只针对所启动的ownedtestprocess树。force/fallback发生时该case保持FAIL。

### 6.2 编译两个既有testimages（只编译，后续独立运行）

下面变量由root绑定到真实fresh绝对路径；命令在仓库root执行，所有输出各自一次收据，不运行通配 `./...`。

```bash
# PLAN ONLY. E and CANDIDATE are reviewed absolute Windows paths.
# inherited LEVNOTE_* and Go selectors were removed by the reviewed caller.
export GOOS=windows GOARCH=amd64 GOAMD64=v1 CGO_ENABLED=0
export GOENV=off GOWORK=off GOFLAGS=-mod=vendor GOPROXY=off GOSUMDB=off
export GOVCS='*:off' GOTOOLCHAIN=local GOTELEMETRY=off
bash .localtools/gate.sh test -c -o "$E/app-bundle/levnote-app.test.exe" ./app/
bash .localtools/gate.sh test -c -o "$E/drawio-bundle/engineruntime.test.exe" ./core/engineruntime
```

[现有gate](<F:/AIproject/LevNote/.localtools/gate.sh#L5-L21>)会进入native并选项目Go；它本身不是本门禁controller。compileexit缺失/非0、source变化、testimage不匹配均停止后续launch并保存failreceipt；不重复一次耗用namespace。

### 6.3 按四个actual test name各运行一次

表内每项 `-test.run` 精确anchor，`-test.count=1`；caller case env全新，TMP/TEMP为run-owned。每项独立保存stdout/stderr/exit；expected顶层 `PASS` exactly1、`SKIP`0、`FAIL`0、testerexit0；“匹配不到test也exit0”不算PASS。任何非0/missingreceipt/SKIP/cleanupfailure/timeout则停止并保留，余项NOT RUN。

| 序号 | Fresh caller添加的env | Exact argv | 既有预期workerrequests |
|---:|---|---|---:|
| 1 | `LEVNOTE_WORKER_BUNDLE_ACCEPTANCE=1` | `E/app-bundle/levnote-app.test.exe -test.run ^TestWorkerExportBundledHostPathAcceptance$ -test.v -test.count=1 -test.timeout=2m` | 2 |
| 2 | `LEVNOTE_WORKER_BUNDLE_ACCEPTANCE=1` | `E/app-bundle/levnote-app.test.exe -test.run ^TestMindmapWorkerExportBundledHostPathAcceptance$ -test.v -test.count=1 -test.timeout=2m` | 2 |
| 3 | `LEVNOTE_WORKER_BUNDLE_ACCEPTANCE=1`; `LEVNOTE_PREVIEW_BUNDLE_CAPTURE_DIR=E/preview-captures` | `E/app-bundle/levnote-app.test.exe -test.run ^TestMindmapPreviewBundledHostPathAcceptance$ -test.v -test.count=1 -test.timeout=2m` | 1 |
| 4 | `LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE=1`; `LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_CANDIDATE=CANDIDATE`; `LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_EVIDENCE=E/drawio-result.json`; `LEVNOTE_DRAWIO_MANAGED_REAL_SMOKE_CONFIG=E/drawio-config` | `E/drawio-bundle/engineruntime.test.exe -test.run ^TestDrawioManagedRealBundleSmoke$ -test.v -test.count=1 -test.timeout=2m` | 1 |

2m为既有test整体上限，runner每request仍30s；App等待done的40s仍沿用source，不缩短或改大来掩盖failure。外层超时需大于testdeadline以留cleanup/stream收尾，最终rawexit与cleanup均记账。

运行后根代理逐项审核：四test存在且PASS0SKIP、logs内容与第3oracle吻合、metadata匹配candidate、drawioJSON actualrootidentity/exit、三张previewPNGbytes/SHA、source/candidate/fixture/plugin inventory一致、copies只多testimages、所有Close/ownedprocess监督无fallback。计数不从PASS日志反推PID；缺process/Job观测按第4.3保留UNOBSERVED。

### 6.4 附加既有contracts独立执行，不并入四成功case

- `TestWorkerExportBundledHostPathRefusal`：[实际env:145–156](<F:/AIproject/LevNote/native/app/export_worker_bundle_test.go#L145-L156>)为 `LEVNOTE_WORKER_BUNDLE_REFUSAL=missing`或`integrity`，不需要acceptance=1。各使用新的完整故障copy和同一已hash app testimage，只在故障copy删除相关manifest或改workerbytes并记录diff；test本身不改bundle。各一次 `-test.run ^TestWorkerExportBundledHostPathRefusal$ -test.v -test.count=1 -test.timeout=2m`；expectedWorkerStart0。这是Markdownpreverifyroute，不假装all4拒收门禁。
- `TestManagedRealOldReleaseThenBundledResetSmoke`：[actualtest:62–328](<F:/AIproject/LevNote/native/core/engineruntime/managed_bundle_test.go#L62-L328>)；另建完整copy、run-ownedTMP/config，设置 `LEVNOTE_MANAGED_REAL_SMOKE=1`和绝对 `LEVNOTE_MANAGED_REAL_SMOKE_RELEASE=OLD_DIFFERENT_RELEASE`；old/new版本必须不同。exact `-test.run ^TestManagedRealOldReleaseThenBundledResetSmoke$ -test.v -test.count=1 -test.timeout=2m`。预期两次Markdown请求：installedold→Reset→currentbundled；exactHTML、Source/digest/version/SHA、Close和source保留。非mindmapreset或all4installed执行证据。
- 第5节actualall4failure/activecancel/Job更强矩阵尚未存在完整当前候选runreceipt，保持pending；如需新增观察由root另立productionAPI任务审定。先执行已有测试、不在Task27写观察实现。

## 7. Wire/scene admission约束与source-only测试边界

[protocol](<F:/AIproject/LevNote/native/core/engineipc/protocol.go>)：protocol1、24Bheader、`LVEXREQ\x00`/`LVEXRES\x00`；littleendian version/header/meta/payloadlength offsets8/10/12/16；requestmetadata≤4096B、response≤1MiB、warnings≤256/code≤64B/message≤512B、error≤1024B、version≤64 printableASCII。request需准确EOF+validUTF8；success无error；coherenterrorpayload0。严格JSONfieldset/duplicate/unknown/null/surrogates拒收见[metadata reader](<F:/AIproject/LevNote/native/core/engineipc/json.go>)；fragmentedstdout通过[ResponseDecoder](<F:/AIproject/LevNote/native/core/engineipc/decode.go>)，extra/incomplete/wrongengineversion释放payload；Finish仅在成功Wait/copyloop/EOF后。

[scene DTO](<F:/AIproject/LevNote/native/core/enginescene/scene.go>) schema1、四layout、nodes≤100000/edges≤99999、ID≤512B/label≤515B、roundedshape、paintempty或lowercaseRGBA；无sourceonlynote/link/image/metadatafields。[Encode](<F:/AIproject/LevNote/native/core/enginescene/encode.go>)确定性、失败nil；[Decode](<F:/AIproject/LevNote/native/core/enginescene/decode.go>)只固定浅schema、字段齐全/无duplicate/unknown、count准确/validrootedtree、boundedstring/number，全失败零Scene。

SVG [worker tests](<F:/AIproject/LevNote/native/cmd/levnote-mindmap-svg/main_test.go>)和preview [main tests](<F:/AIproject/LevNote/native/cmd/levnote-mindmap-preview/main_test.go>)已全文核对；foreignrequest不读payload、invalidUTF8/trailing/version、depth/source/warning/errorcode、writerfail及preview真实32MiBescapedoutput均有source测试。SVGtinycap是私有seam，production64MiB固定cap未被该seam实跑；preview真实32MiBcase虽然使用真预算，依然是内联serve而非packaged.exe。准确保留已有scope，新增进程证据待补不转成产品缺口。

## 8. 有限验收判据与下一步

分层记录状态，不取并集冒充wholeapplicationPASS：

| 层 | 可由本计划未来获得的证据 | 当前 |
|---|---|---|
| source/in-process/helper | 已有source-suite与literal/frame/helperjoin测试的原范围 | source证据既有；本任务未重跑 |
| packageidentity/5GUIPE | 必须从root当前candidatechecker原收据取得 | 入口held/新采集由root核定 |
| 四actualworker成功 | 四实际测试各PASS、0SKIP；共6预期requests；三NewBundled+一installedNewManaged | NOT RUN |
| all4actualfailure/heldroot/Job/member/join强门禁 | 第4.3/5节完整actual收据 | NOT_RUN / PENDING，既有部分hook/helperscope分开 |
| packagedhost/Job/WM_CLOSE/session/secondlaunch | 另立root审定run；可参考[已准备host流程](<F:/AIproject/LevNote/native/handoffs/autonomous-local-windows-delivery-20261113/README.md>)，不从worker PASS推出 | 本任务未运行 |
| physicalGUI/IME/keyboard/clipboard/picker/rendering | 真机独立检查；TesterPNG/hiddenstartup不替代 | PENDING |
| resource/response/wholeappdelivery | 真实workload+完整产品判据 | PENDING / application NO_GO |

下一步第一动作：root读本文件并独立核定第9身份及已有 SVG/preview正文；结合新packager原始收据决定G6.1是否可进入。若入口held，记录具体失败phase/exit/rawstreams，四test继续NOT RUN，不复用耗用证据目录。若解锁且currentcandidate完整，按第6序列只编译两个existingtestimages，再按四actualnames各一次；root审核完成之前不把Task27文档交付标成worker或应用验收。

## 9. 已读证据身份（SHA256，非执行或签名）

所有路径为本地磁盘实际read/stat结果；四成功test全部正文已读，managedold-release正文亦读完。最后四test和protocol/Runner复查未变化；未来compile前仍需重新核定源inventory。

| 文件 | Bytes / Lines | SHA256 |
|---|---:|---|
| [Markdown app bundle test](<F:/AIproject/LevNote/native/app/export_worker_bundle_test.go>) | 8878 / 212 | `ad93ce0c181f771849361219489125b221989db3ebedc7e71c28aca93146b39e` |
| [SVG app bundle test](<F:/AIproject/LevNote/native/app/export_mindmap_worker_bundle_test.go>) | 9261 / 212 | `2382a35951d4817e4d966cfc3758ec85232478200b07ccd8e1dbb9409b6addc7` |
| [Readonly preview app bundle test](<F:/AIproject/LevNote/native/app/preview_worker_bundle_test.go>) | 12227 / 256 | `272bbf326b5509befc6448d53588022e8e561c25e076667c1f23fa897a5ecf0b` |
| [Draw.io managed candidate test](<F:/AIproject/LevNote/native/core/engineruntime/drawio_managed_bundle_test.go>) | 20800 / 544 | `cb678b7da1d2c4f5b91e5873d156b97cc4fcbd248f0785fd7045674dbf1dd695` |
| [Markdown managed/reset](<F:/AIproject/LevNote/native/core/engineruntime/managed_bundle_test.go>) | 12065 / 328 | `47e0162df75f73e2146469aabf24bf5c3f976e99a18bfa55cb8c7a913fca0a90` |
| [Runner](<F:/AIproject/LevNote/native/core/engineruntime/runner.go>) | 5668 / 197 | `e917779fe5768984cd11c288ca150f5ff0e74d9f6b214599b914877581aebcd6` |
| [run coordinator](<F:/AIproject/LevNote/native/core/engineruntime/run.go>) | 7703 / 259 | `bc54377fb32cfb3d6a9b7eeb298bf34c775d2dc349d1394c5d340347f0c5bbbc` |
| [Windows process/Job](<F:/AIproject/LevNote/native/core/engineruntime/process_windows.go>) | 6456 / 210 | `a43d0ba27aed86e7195d97c4d968040d097cb55a47592e799b8783a9b320b486` |
| [public package state/Activate/Reset](<F:/AIproject/LevNote/native/core/engineruntime/package_state.go>) | relevantboundedranges | `6dfd18c9ff5d5f50c6579f780e7e080633a7a7d5bb89195d72a07c4fc27a485b` |
| [public Import/Installed](<F:/AIproject/LevNote/native/core/engineruntime/package_store.go>) | relevantboundedranges | `2e17e8fa07a31c8b49108315ef0afe2e4b7adcbfa9a76aec071c9d525cccd852` |
| [protocol](<F:/AIproject/LevNote/native/core/engineipc/protocol.go>) | 8944 / 279 | `6196bf4d35c733f4d1db048293f19e3a55a7b50eb3fce8a2b6405d5f2a41ed77` |
| [metadataJSON](<F:/AIproject/LevNote/native/core/engineipc/json.go>) | 5405 / 198 | `e6560e806c9314f94174c2b85ff75d1abf2f8362b254ffe0ddcfbd44015c7157` |
| [frameDecoder](<F:/AIproject/LevNote/native/core/engineipc/decode.go>) | 3431 / 121 | `1f3547f2f3c2bbf8d6705928f42cc7de1c87f7a852f93aca3e02e822f760e258` |
| [sceneDecoder](<F:/AIproject/LevNote/native/core/enginescene/decode.go>) | 13173 / 580 | `73ab1613cb66e64110334cf441f95aa721b8b4b91abdf4a79c238795c6416148` |
| [SVG producer tests](<F:/AIproject/LevNote/native/cmd/levnote-mindmap-svg/main_test.go>) | 16558 / 408 | `ddc6f1012e207cdcd1f91ffc28f787b67a250d91cef37b17457bf02f3cb45415` |
| [Preview producer main tests](<F:/AIproject/LevNote/native/cmd/levnote-mindmap-preview/main_test.go>) | 16065 / 370 | `f1aec813ca727c3221721b0e984763c1fa4e57b8c1f9a22cba5720cffa2015de` |
| [Preview scene tests](<F:/AIproject/LevNote/native/cmd/levnote-mindmap-preview/scene_test.go>) | 9057 / 169 | `b4b001d43ae9fafa996ea3f5fc05f9189ee9d22af1c4ea20ec746d48c4131533` |
| [frozenTask20plan](<F:/AIproject/LevNote/native/handoffs/autonomous-native-ci-preparation-20261114/REAL-WORKER-PLAN.md>) | 5632 / 38 | `28c24b519ab03eaa76d20de60e8002a6c94d1d643df125eb3c227f8a66bb3504` |
| [sourcegate](<F:/AIproject/LevNote/.localtools/gate.sh>) | 928 / 24 | `5fecb22a44f54f64e5bd65c25773179568bebd6fd8840ac2e875546782120f35` |
| [actualsource-suite raw output](<F:/AIproject/LevNote/.localtools/final-native-20261114/test-01/stdout.raw>) | 2915861 / 12741 | `013b7f58da8f6005918b7005078ebc48fce4fde8087dab46a894126ac0aba56e` |
| [source-suite result](<F:/AIproject/LevNote/.localtools/final-native-20261114/test-01/result.json>) | 15876 / 534 | `1a3c4abf56d7d05280db4ac2a9ff694c891c7dfb68d4850d910ed8933f097e5a` |

Task27未build/launch、未执行Go/产品/helper/网络命令、未访问真实用户config、未改冻结Task16/20/24产物、未Git或再委派。文档完成与root独立acceptance分开；本文不激活CI、不晋升candidate。
