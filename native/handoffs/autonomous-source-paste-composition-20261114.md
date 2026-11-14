# Task43 · 源码粘贴读取回调的组合所有权

## 范围与契约

固定基线 `4122b9faa4810f8d861e2f6ccd87b712405eebd5`。本组仅修改 `native/ui/editor/editor.go`、新增 `native/app/source_paste_composition_test.go` 与本文。完整 Spec 为 `.localtools/source-paste-composition-spec-20261114.md`，SHA256 `8389c0cf7568824a7b3da5bdb7a103339f0006ba33955847d0080e058e2587e9`。

公开 seam 是内置 App 源码视图、`ui.Tester.Command/Type/Key`、`Editor.Paste/SetMarkedText/MarkedRange/Flush` 与 Buffer 文本、选区、版本、历史、Undo/Redo。新测试不读取私有组合 token、布局缓存或平台内部。它覆盖 UI 线程同步主机回调，不建立真实 Windows 剪贴板或物理 IME 接受。

`Paste` 成功读取后，原粘贴只可操作读取开始时的 Buffer、版本、选区、视图和组合所有权。回调新启空/相同文本预编辑可保持版本和选区，却仍属于新组合；回调结束原无改动组合同样使原粘贴失去所有权。原粘贴保持主机状态并报告错误。回调新产生的组合错误优先保留原对象；剪贴板自身读错误仍先报告，同时保留待恢复组合错误。

## 两个实际 RED → 最小 GREEN

第一条53行2094B原测试保存在 `red-01/first-test.raw`，SHA256 `4ec4ad0680147bbe5b1942e267a46a4568d356c2c5bb13547f801ccd8f7d7362`；最终测试以前缀原样保留，不修改字面断言。

- `red-01`：Go8432，UTC `2026-11-14T11:15:52.820887Z`–`11:15:58.253365Z`，exit1/1FAIL/0SKIP；原 editor 与1215其它产品保持。空预编辑的版本/选区/marked 前置检查通过，实际源变成 `aPbc\r\ndef\r\n`，预期字面 `abc\r\ndef\r\n`。原 stdout1793B，SHA256 `32cbf903d473b45f2ddbd8e75f1f809f900d1b5b59a8d0fd1b2eb76c3304a391`。
- 第一片最小实现：捕获原组合 token，成功 read 后 token 改变则报告且结束。`green-01` Go2244，UTC11:17:16.069330Z–11:17:22.967624Z，exit0/1PASS；原空组合仍可提交“中”，一条 Undo/Redo。
- `refusal-red-01`：第一片已绿的生产代码下，回调新拒收“中”没有 token，仍插入 P并清掉原错误；Go3732，UTC11:18:26.399342Z–11:18:32.087823Z，exit1/1FAIL。stdout1772B，SHA256 `ad4f595269a3608c81ea98eea1dead9a9d0586d0f6aeb1608b1931e4f67d3e97`。
- 第二片最小实现：成功 read 后首先重新检查 `compositionIssue()`，原错误对象报告后结束。`refusal-green-01` Go19208，UTC11:19:45.427342Z–11:19:51.501423Z，exit0/1PASS；Esc明确恢复后正常粘贴/撤销保持。

生产共新增9行，不改原 Task40身份守卫、剪贴板 read-error 分支、普通替换语义和 entry-composition 阻塞。没有格式化写入、私有 token 测试、降低断言、新 SKIP、setup失败或不相关重构。

## 逐片控制与唯一 focused union

其余四控制逐片新增后 first GREEN，不冒充新的 RED：相同文本预编辑/两种粘贴命令；读错误与待恢复原组合错误；原未变化的空/非空 entry owner接受 Unicode/CRLF粘贴；回调 Flush结束原空 owner 后旧粘贴停止、后续新粘贴恢复。全部具有独立字面文本和选区/历史 oracles。

`same-text-01` Go5948/3PASS、`read-error-01` Go24252/1PASS、`entry-owner-01` Go24352/7PASS、`settled-owner-01` Go23336/3PASS，均exit0/0SKIP且前后全输入守卫一致。

唯一 `focused-union-01`：Go9568，UTC `2026-11-14T11:24:47.787113Z`–`11:24:51.363020Z`，exit0；6顶层+10嵌套=16PASS/0FAIL/0SKIP。stdout16554B，SHA256 `0e33286d8ca6055e5b2e5da71865d292406cab6c0be82ceb09e8489eccc7f5ec`，stderr0。产品1217聚合 `42ebbed2ea326603f0d25c3fe0ecc2649d393507daa6b0ba51a7266c67c71174`。

## 冻结和接受边界

原1216产品除 editor.go保持（1215保护产品）；新 App测试使产品1217。所有开发命令经固定离线Go1.27.3 gate；7 pins、历史534excluded与CON仅metadata守卫不变。每次新证据目录create-only，原stdout/stderr、actualGoPID、UTC、exit、全输入前后守卫保持。helper复用历史只读函数，不执行历史main、不重写旧证据。

实现已停止写入，独立 Standards/Spec须绑定同base、Spec、三file SHA/OID与1217聚合；唯一affected App/editor/modules全回归、vet、build由lead在接受后执行，实际结果留于fresh命名证据，不通过追加文字假定其成功。

本组不修改受保护保存/搜索文件、Windows权限、vendor、工具链或CI；不接受fresh5PE包、四realworker/强Jobcleanup、ownedhost关闭重启、物理GUI/IME/clipboard/pickers、资源或全应用。CI保持inactive、应用NO-GO，旧.29仍latestacceptedrunnable。
