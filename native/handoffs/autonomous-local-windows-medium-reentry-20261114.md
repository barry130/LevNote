# 本地 Windows 执行条件变化后的原入口重核

## 结论（type:state）

本篇记录用户继续请求后的一次有限复核，**不是应用接受、不解除原平台 held，也不激活 CI**。当前 Windows `whoami /groups` 主进程显示 Medium mandatory label `S-1-16-8192`，与历史官方诊断的 Low `S-1-16-4096` 不同；这只建立重新观察原入口的条件变化，未证明子进程写入能力或 Windows 文件权限已恢复。没有修改 ACL、owner、UAC、执行壳或原脚本，也没有用预建目录、直接文件覆写、Junction→copy 代替失败入口。

源码保持 [`75a5c20`](<F:/AIproject/LevNote/native/handoffs/autonomous-local-source-delivery-receipt-20261114.md>) 所记录的 1,211-member 冻结输入，聚合 `874bc1fd23a3d02e432c9c6305842ee540afdca1cdad23692905da80d01b2bdf`；新观察未改产品或测试文件。原有限 test 3,019 PASS / 0 FAIL / 49 SKIP 与 vet/build exit0 的范围不扩张。

## 真实执行与失败（type:dead-end）

| 原入口 | 本次命令结果 | 实际到达边界 |
|---|---|---|
| [Go1.27.3 gofmt -w 八文件](<F:/AIproject/LevNote/.localtools/format-medium-20261114-01/04-original-gofmt-write.json>) | actual exit2，stdout0B / stderr829B | 八个原源码目录的临时文件 `open … Access is denied`；0 文件改变，全部 1,211 输入 before=after |
| [check-build-guard.ps1](<F:/AIproject/LevNote/native/scripts/check-build-guard.ps1#L20>)，`-CaseSet All` | [actual exit1](<F:/AIproject/LevNote/.localtools/guard-medium-20261114-01/result.json>)，stdout0B / stderr266B | line20 原 Junction 创建失败；0 guard cases，无产品运行 |
| [package-markdown-worker.ps1](<F:/AIproject/LevNote/native/scripts/package-markdown-worker.ps1#L24>) | [actual exit1](<F:/AIproject/LevNote/.localtools/package-medium-20261114-01/result.json>)，stdout0B / stderr385B | line24 创建原 `native/artifacts/packages` 父目录下的新证据目录失败，先于 candidate mkdir/build |

原 `gofmt -w` 显式集合为 `native/app/app.go`、`native/app/list_lifetime_test.go`、`native/core/document/fingerprint_boundary_test.go`、`native/core/document/publication_error_test.go`、`native/core/document/windows_retry_conflict_test.go`、`native/core/search/workspace.go`、`native/formats/markdown/table.go`、`native/formats/markdown/table_column_edit.go`。before/after 扫描、原命令与所有流在[本次 formatter 回执目录](<F:/AIproject/LevNote/.localtools/format-medium-20261114-01/>)保留。调用者最终 exit1 / `FAILED_PRESERVED` 源于期望原格式入口成功的断言失败，不改变真实 gofmt exit2。为检查纯格式变化而预备的 Go token 比较器仅比较了未改变的源码，没有产生格式化成功或新应用测试证据。

本次 packager 消耗的 candidate 名为 `levnote-native-worker-20261114.32`、脚本 evidence 名为 `autonomous-local-windows-package-20261114-02`：两者在失败后均 absent。**0 新 GUI PE、0 host/worker launch**；没有复用失败 01、旧 `.31` 名、旧 prepare02、已闭合 `.30` 或已接受 `.29` 包。新的调用者证据位于 workspace `.localtools`，不是将被拒绝的原脚本输出改写到另一路径。

工具链保持：本地与打包 Go 都为 `go version go1.27.3 windows/amd64`，reviewed native x64 PowerShell7.7.0、CGO0、GOENV/GOWORK off、GOFLAGS `-mod=vendor`、GOPROXY/GOSUMDB/GOVCS off、GOTOOLCHAIN local、telemetry off，临时目录限定原 workspace。包装器 `go.ps1` 及 vendor26pin 未改。PowerShell 的现有进程级 PATH 仅前置已审阅 pwsh 目录以满足原脚本 `Get-Command pwsh`，没有持久配置或 alternate shell。

## 原始证据身份

| 证据 | Bytes | SHA256 |
|---|---:|---|
| [formatter 原 stderr](<F:/AIproject/LevNote/.localtools/format-medium-20261114-01/04-original-gofmt-write.stderr.raw>) | 829 | `312729a1a5689eaa0cc24f3656151147e7495f2b3accedbf401d5515560a1658` |
| [guard 原 stderr](<F:/AIproject/LevNote/.localtools/guard-medium-20261114-01/02-original-entry.stderr.raw>) | 266 | `80a19c758990c9ec4ef2c57b5bc29d76047d98dde86734194027c18e7d0bd1fe` |
| [guard caller result](<F:/AIproject/LevNote/.localtools/guard-medium-20261114-01/result.json>) | 988 | `b0199f67975a440fa2d2fd83d1214a4c1ec98b5e39ca9345d597190b2c53a698` |
| [packager 原 stderr](<F:/AIproject/LevNote/.localtools/package-medium-20261114-01/02-original-entry.stderr.raw>) | 385 | `096304e29b502eb7b7f876f0a3535f27f3d6ec50903ff537d9bc498ed0fd30d1` |
| [packager caller result](<F:/AIproject/LevNote/.localtools/package-medium-20261114-01/result.json>) | 1405 | `29354e4b46ddbd66a66f80f30b275a2bc931efcce30eb03e84a395556f89c3a0` |

[root 独立重核](<F:/AIproject/LevNote/.localtools/medium-reentry-root-audit-20261114-01/result.json>)实际 exit0 / `MEDIUM_ORIGINAL_GATE_FAILURES_INDEPENDENTLY_VERIFIED_APPLICATION_NO_GO`，1,289B / SHA `3df0bb272e59290b2405a36d0070c1848736b0f0cb2077629f18a4605f09a1d4`：复核完整 1,211 源成员、format before/after 与 token 比较、三原入口 actual exits/原流及工具 pins、Medium 观察、candidate/evidence absent、tracked/index 干净和 disabled CI。这是失败证据核验成功，不是三个产品门禁成功。

历史失败、官方 `nextAction=stop` / zerochanges、原有限 gate rawstreams 不改写。三个原入口本次再失败之后不自动重试、不换 writer/shell/output、不再尝试修 ACL 或所有权。

## 已有测试的反证与真实下一步（type:progress）

Task27 与 root 全文核对确认两个现有测试均已存在：

- [TestMindmapWorkerExportBundledHostPathAcceptance](<F:/AIproject/LevNote/native/app/export_mindmap_worker_bundle_test.go#L26-L173>)：`LEVNOTE_WORKER_BUNDLE_ACCEPTANCE=1`，真实 NewBundled SVG 两请求、独立 literal XML/geometry 与 exact warnings、BOM/CRLF/source/state 保持、exclusive target refusal、joined-return；声明没有 PID sampling。原文件 9,261B / SHA `2382a35951d4817e4d966cfc3758ec85232478200b07ccd8e1dbb9409b6addc7`。
- [TestMindmapPreviewBundledHostPathAcceptance](<F:/AIproject/LevNote/native/app/preview_worker_bundle_test.go#L42-L210>)：同 opt-in，另须 fresh `LEVNOTE_PREVIEW_BUNDLE_CAPTURE_DIR`，真实 filesystem readonly 检测、NewBundled scene、独立 literal geometry/warnings、3 张 headless Tester 图和结果 slot 丢弃；明确没有 physical UI/PID observation。原文件 12,227B / SHA `272bbf326b5509befc6448d53588022e8e561c25e076667c1f23fa897a5ecf0b`。

因此缺口不是“缺 mindmap 测试”，而是这些 opt-in 对新候选的实际运行、全四 worker 受控失败及强进程/Job/cleanup 接受仍未完成。前默认 suite SKIP 未变成 PASS；局部 CI preparation 的两项 managed smoke 不是全仓覆盖清单。不修改已有测试或原冻结准备来制造成功。

root 已全文读取并有限接收 [Task27 门禁计划](<F:/AIproject/LevNote/native/handoffs/autonomous-mindmap-real-worker-plan-20261114.md>)，终稿38,408B / 267行 / SHA `a5286885333cfe2b5e4cce891e670155635b3fa2792f8480f215840e4dd4fb4e`。[独审02](<F:/AIproject/LevNote/.localtools/mindmap-plan-root-audit-20261114-02/result.json>)实际exit0，核对21来源bytes/SHA、五个真实defaultsuite opt-in SKIP和终稿pin，结果4,271B / SHA `9bce702e8b9049afc6d36ed965874cba26ae04b6df3f57d70b9c7d186a5d5c41`。独审01在UTF-8 rawstream误用Python默认GBK读取时exit1，原[失败补记](<F:/AIproject/LevNote/.localtools/mindmap-plan-root-audit-20261114-01/failure.json>)保留；02只纠正caller encoding并使用新命名空间，产品与plan未为审计修改。

计划精确区分三项App/NewBundled和一项draw.io/NewManaged-installed、6次预期而未观察worker请求、独立 literal SVG/scene oracle、disposable tester非第6/7 shipping PE以及更强PID/Job/异常清理 **NOT_RUN**。root接受仅指入口索引及未运行计划，不执行或激活冻结准备，不替代原独立 Standards/Spec 源码审阅。

平台 lane 继续 held，等待原入口真实条件另有变化。依次为五 PE 完整包 → disposable tester bundle 四 worker 接受 → owned host/session/close/restart → 物理 Windows GUI/IME/剪贴板/选择器及资源负载 → reviewed native CI。当前应用 **NO-GO**，`.29` 仍 latest accepted runnable；本篇以及新门禁计划属于文档补记，不代表新产品源码或平台接受。
