# Task31 · 工作区磁盘结果与已开草稿 · 2026-11-14

## 任务状态（type:state）
- 实施已冻结，仅修改[workspace.go](<F:/AIproject/LevNote/native/app/workspace.go#L237-L301>)，新建[回归测试](<F:/AIproject/LevNote/native/app/workspace_result_stale_test.go>)与本交接；基点257ef9215fb999b1e154d64a076efe7a7c977b0a，无Git或产品启动。
- 最终源码10460B SHA256 `877351825bbab40477718b15a4030d258dafb7311b31a1ed2e124226ccaf7cf4`；测试15855B/325行 SHA256 `cb85daa0e89d19601ebe70c57d4fa0f39fb7a33725f5f0aab4b3911807c1119a`。
- [最终focused收据](<F:/AIproject/LevNote/.localtools/workspace-stale-20261114/final-union-10/receipt.json>)：Go5100、测试21112、wrapper29204，06:52:04.316915Z→06:52:08.702687Z，exit0；9顶层+4子项PASS、0skip；[完整raw](<F:/AIproject/LevNote/.localtools/workspace-stale-20261114/final-union-10/stdout.jsonl>)12270B SHA256 `cb82a6e5720ac45146ee5c53dfa5cb91e28db195aed12bb9fc90e1f9fb9121ed`，stderr0B。
- 仅 `bash .localtools/gate.sh test -json -count=1 -timeout=120s -run '^TestWorkspaceDiskResult' ./app`；既有Go1.27.3、离线vendor、局部cache/tmp、CGO0、遥测关闭。未单独运行vet/build或Appfull；Go test自动vet子进程按实际进程记录保留。

## 目标与限定实现（type:goal）
- 真实磁盘 `zero\nneedle\n` 搜索得5..11；公开源码编辑成 `xxxxx\nzero\nneedle\n` 后真实列表Enter原选 `\nzero\n`。现在保留18..18及草稿/历史/编码BOM/ReadOnly/所有者，显示“工作区结果来自磁盘；文档已有未保存修改，未跳转”。
- 已脏目标在Flush/Select/Release/Reveal前早退；干净已开目标捕获Tab/Doc/Editor/Buffer/path/version及searchID/root/result，目标Flush后重检、`selectChecked`源Flush后的callback重检，再同步`revealSource`，无第二次目标Flush。未开文件保留原OpenAt路径。
- 干净当前/非当前正常选择5..11 `needle`；公共Flusher同步产生草稿后保留其有效18..18与undo/redo；源Flush令目标变脏、失败Flush、已有脏非当前目标、搜索代/root/Buffer身份/净零双编辑版本控制各有公开Enter测试。
- Flusher为同步host-contract夹具，不代表内置并发或physicalIME；本切片未实现search时磁盘快照版本、搜索后保存/重载/外部改动重新验证，同行Preview P2仍held；原生平台交付仍NO-GO，.29不变。

## 已试方案与保留失败（type:dead-end）
- [首RED](<F:/AIproject/LevNote/.localtools/workspace-stale-20261114/red-01/receipt.json>) Go21876/wrapper30000，06:21:56.897317Z→06:22:03.780852Z exit1：[raw](<F:/AIproject/LevNote/.localtools/workspace-stale-20261114/red-01/stdout.jsonl>)实选5..11 `\nzero\n`；原生产d9f458…、首94行3868B/f97e812f…先运行后最小dirty gate。[首GREEN](<F:/AIproject/LevNote/.localtools/workspace-stale-20261114/green-01/receipt.json>) Go32688 exit0，同首test不变。
- 仅dirty预检查→[pending RED](<F:/AIproject/LevNote/.localtools/workspace-stale-20261114/red-pending-02/receipt.json>) Go29628 exit1，合法同步后的草稿仍被5..11旧范围定位；捕获/Flush重检修正后[green-pending-02](<F:/AIproject/LevNote/.localtools/workspace-stale-20261114/green-pending-02/receipt.json>)实exit1/1PASS+1FAIL：选区18..18及可见失败已正确，但新增fixture版本误填2；公开NewBuffer版本0+一次Replace实际1，新增premise0及literal1修正后[03](<F:/AIproject/LevNote/.localtools/workspace-stale-20261114/green-pending-03/receipt.json>)2PASS。首94行和原raw均未改。
- [control04](<F:/AIproject/LevNote/.localtools/workspace-stale-20261114/control-04/receipt.json>)实exit1/3FAIL：新增clean oracle误以Release清空View指针；两inactive fixture公开异步Open未drain。修正为原指针保留与真实队列等待后[05](<F:/AIproject/LevNote/.localtools/workspace-stale-20261114/control-05/receipt.json>)实build-fail/0test，新增helper漏queue参数；签名/调用修正后[06](<F:/AIproject/LevNote/.localtools/workspace-stale-20261114/control-06/receipt.json>)3PASS。以上fixture纠正期间生产未改。
- [07](<F:/AIproject/LevNote/.localtools/workspace-stale-20261114/control-07/receipt.json>)首次3PASS；[08](<F:/AIproject/LevNote/.localtools/workspace-stale-20261114/control-08/receipt.json>)首次1顶层+4子项PASS。它们是新增控制，不另称真实RED；所有原失败单独create-only保留。

## 进度与下一步（type:progress）
- [format09](<F:/AIproject/LevNote/.localtools/workspace-stale-20261114/format-09/receipt.json>)gofmt exit0，源码/测试字节完全不变；[fresh审核](<F:/AIproject/LevNote/.localtools/workspace-stale-20261114/after-format-09/audit.json>)确认[首94行raw](<F:/AIproject/LevNote/.localtools/workspace-stale-20261114/after-format-09/first-red-prefix.raw>)3868B SHA256 `f97e812f1a65c7011e602a2943eedf59842b23d2887a92874884c1e30ddd6c67`，与首RED一致。
- 全部测试局部前后pin均相等；最终聚合 `1b68bea1d7ee4e32a4f2d62f6330fd40a2f40e5602c4e2e2f485f036bf3a53f3`。8原denied文件、gate、Go/gofmt、go.mod/sum逐项相对首RED完全不变；并行editor工作未纳入局部相等宣称。
- 执行者停止全部源码/测试/交接写入；lead先读本交接及最终raw、freeze11收据，再收两实施停写快照，启动独立Standards/Spec及唯一App+editor+modules受影响门禁。
