# Task40 · 源码粘贴同步主机回调身份保真

## 任务状态

(type:state) 实现与公开回归已完成，当前仅源码/Tester阶段。固定HEAD为 `a055cb6f5c3a3d72e650e2d06943f92c853664d5`；生产改动仅[editor.go:715–734](<F:/AIproject/LevNote/native/ui/editor/editor.go#L715-L734>)，新回归为[源粘贴身份测试](<F:/AIproject/LevNote/native/app/source_paste_identity_test.go>)，第三交付物为本交接。
已完成11个fresh focused调用：1次fixture编译失败、1次真实product RED、1次最小GREEN、7次必要控制、1次最终联集。联集8个顶层+6个子用例=14 PASS、0 FAIL、0 SKIP。所有原raw与收据保持原字节。
生产25988B SHA256 `4377e4652b7d173245ad0e96d0079fcb738b1cbc58bfd1de4af1cb759a9b6eac`；测试12504B SHA256 `ac84c3a9e1bded22435755d88ea31b217d07ed8b6622e6f8197cba48c3275aa0`。三文件原始副本及本交接SHA由[冻结结果](<F:/AIproject/LevNote/.localtools/source-paste-20261114/freeze-01/result.json>)集中记录，避免交接自指摘要。

## 目标

(type:goal) 对已接受的可写paste/pasteAndMatchStyle，在同步读取剪贴板前绑定Buffer/Version/Selection/View lifetime，成功读后只有原身份仍有效才插入；主机导航、编辑、Buffer替换与ReleaseView均保留，不回滚。原read error、入口composition error及初始readonly语义保持优先。
预先确认的公开seam为真实built-in Markdown App.View源码编辑区、`ui.Tester.Command`、公开`Tab.Editor.Paste` callback、公开Buffer文本/选区/版本/历史以及App Message/Issues，详见[root固定spec](<F:/AIproject/LevNote/.localtools/source-paste-review-spec-20261114.md>)。这是同步UI线程host重入合同，不是内建Windows剪贴板并发或物理IME结论。
最小修正只有5行净增：读取前记录buffer/version/generation，读取成功后比较Readonly、Buffer指针、Version、原Selection、viewGeneration；过期时报告“剪贴板已读取；文档或选区已改变，未粘贴内容”，保持命令已处理并停止该次插入。读取失败分支先报告原error并return，沿用原ReplaceText、history与IME流程。

## 已试方案与失败原因

(type:dead-end) 首fixture采用未核实的App/Tester接口→[red-01编译原stderr](<F:/AIproject/LevNote/.localtools/source-paste-20261114/red-01/stderr.raw>)记录 `a.current undefined`、`ui.TestConfig`/NewTester签名及Key参数错误。该调用不计product RED；生产原始字节未改，首次fixture与original-editor副本保留于同目录。
更正为实际 `App.Current()`、`ui.NewTester(a.View,1180,780)`、`tester.Key(mod,key)` 后，[red-02原stdout](<F:/AIproject/LevNote/.localtools/source-paste-20261114/red-02/stdout.raw>)真实暴露 `source = "abc\r\ndPef\r\n"`，独立literal期望为 `"abc\r\ndef\r\n"`。callback一次、selection从1到6、host版本/历史不变的前置断言已通过。
真实首用例[1–66行](<F:/AIproject/LevNote/native/app/source_paste_identity_test.go#L1-L66>)2375B SHA256 `a5420fb6e01225eed610985611301a610ae0ddae1a153992137e651ed646ecfe`，作为完整byte prefix保留。后续controls只追加，imports及首oracle均无改写；[exact-prefix/生产投影审计](<F:/AIproject/LevNote/.localtools/source-paste-20261114/audit-01/result.json>)exit0，gofmt仅 `-d` 两文件、无写入、无差分。

## 进度与下一步

(type:progress) [原raw审计](<F:/AIproject/LevNote/.localtools/source-paste-20261114/raw-audit-01/result.json>)12172B SHA256 `8632652fba6b1898d7654537a9f7f208934f26bb7cb5d08fad2bc6d8f918740d`；[联集原stdout](<F:/AIproject/LevNote/.localtools/source-paste-20261114/focused-union-01/stdout.raw>)2568B SHA256 `db3b8c85a185b618d987675d1d6e5e1d91ea29c6b6a0a8bc130f4b9ce9b0e8c2`。没有重复Go、全affected、vet、build、包装、Git或网络调用。
原1215产品基线聚合 `3023befb009c177558a6f69bddd7e6fc40ff56717675ea6f632061de7b1ac68a`；只改其中editor、保护原1214其余产品，再加新测试=1216产品。本交接作为独立第三文件。7个gate/executable/modsum pins、原534excluded路径metadata与CON lstat-only metadata逐run前后相同；缓存/tmp/历史toolchain不是产品源码。
下一步由lead读取[三文件冻结结果](<F:/AIproject/LevNote/.localtools/source-paste-20261114/freeze-01/result.json>)及[1216产品状态](<F:/AIproject/LevNote/.localtools/source-paste-20261114/freeze-01/state.json>)，启动Task41/42独立Standards与Spec双轴，再执行root唯一affected test和顺序vet/build。writer在冻结后声明 `STOPPED_WRITING`，Task40完成；闭合历史证据/.29/CIinactive与application NO-GO保持。

## 公开用例与调用账

每个调用均使用固定reviewed Bash + [gate.sh](<F:/AIproject/LevNote/.localtools/gate.sh>)，cwd `F:/AIproject/LevNote`，Go image为 `.localtools/go1.27.3/bin/go.exe`，离线vendor/local缓存/tmp、无遥测。精确argv、wrapper/gate PID、Go创建UTC/image、起止UTC、exit及before/after/raw SHA在各receipt中。以下时间均为2026-11-14 UTC，非物理GUI验收。

| fresh receipt | Go PID | UTC start → end | exit | 分类/结果 |
|---|---:|---|---:|---|
| [red-01](<F:/AIproject/LevNote/.localtools/source-paste-20261114/red-01/receipt.json>) | 29732 | 09:10:28.075227 → 09:10:29.381327 | 1 | fixture编译失败，非product RED |
| [red-02](<F:/AIproject/LevNote/.localtools/source-paste-20261114/red-02/receipt.json>) | 29108 | 09:12:15.263264 → 09:12:21.667376 | 1 | selection-only host导航产生不期望P插入，真实RED |
| [green-01](<F:/AIproject/LevNote/.localtools/source-paste-20261114/green-01/receipt.json>) | 8100 | 09:14:01.608821 → 09:14:08.061379 | 0 | 原66行oracle，最小guard GREEN |
| [buffer-01](<F:/AIproject/LevNote/.localtools/source-paste-20261114/buffer-01/receipt.json>) | 29864 | 09:15:03.998336 → 09:15:09.106235 | 0 | 原/新Buffer同字面、Version、Selection，保留两个文档及host替换 |
| [version-01](<F:/AIproject/LevNote/.localtools/source-paste-20261114/version-01/receipt.json>) | 9740 | 09:15:58.222967 → 09:16:03.943397 | 0 | 同Buffer/Selection，保留H host编辑，Undo/Redo只有host编辑 |
| [release-01](<F:/AIproject/LevNote/.localtools/source-paste-20261114/release-01/receipt.json>) | 29036 | 09:16:40.902431 → 09:16:46.513187 | 0 | ReleaseView后无插入，新live view普通paste恢复 |
| [error-01](<F:/AIproject/LevNote/.localtools/source-paste-20261114/error-01/receipt.json>) | 26348 | 09:17:30.411186 → 09:17:35.674063 | 0 | host selection变更且read ErrRange时，原error优先 |
| [ordinary-01](<F:/AIproject/LevNote/.localtools/source-paste-20261114/ordinary-01/receipt.json>) | 9228 | 09:21:37.642837 → 09:21:43.123826 | 0 | paste/matchStyle×host/services四子用例，Unicode/CRLF选区替换+Undo/Redo |
| [readonly-01](<F:/AIproject/LevNote/.localtools/source-paste-20261114/readonly-01/receipt.json>) | 15088 | 09:23:36.119569 → 09:23:50.071009 | 0 | 初始readonly不read；callback readonly无edit且可见error；恢复普通paste |
| [composition-01](<F:/AIproject/LevNote/.localtools/source-paste-20261114/composition-01/receipt.json>) | 20356 | 09:24:42.739654 → 09:24:48.272790 | 0 | 入口已拒尺寸composition原issue优先/0read，Escape后有限paste |
| [focused-union-01](<F:/AIproject/LevNote/.localtools/source-paste-20261114/focused-union-01/receipt.json>) | 24000 | 09:26:39.953272 → 09:26:43.689358 | 0 | 8top+6nested=14 PASS /0FAIL /0SKIP |

所有控制为最小GREEN之后逐片首次PASS，不额外宣称RED→GREEN。首red-02 stderr空；red-01编译stderr760B SHA256 `38623e4212ae7c63ea96e337546d781777d7bdef3e75b1462e5207b814678480`独立保留。focused运行器来自实读的旧runner字节，在新scope中create-only派生；唯一后续helper变更为启用red-02原fixture/original副本，详见[runner派生记录](<F:/AIproject/LevNote/.localtools/source-paste-20261114/derivation.json>)及[审计记录](<F:/AIproject/LevNote/.localtools/source-paste-20261114/audit-01/result.json>)。
