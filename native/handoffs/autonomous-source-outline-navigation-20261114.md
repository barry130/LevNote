# Task37 · 源码大纲跳转后的垂直目标列修正

## 任务状态（type:state）

- 实施 owner：native-input-scout；基线 HEAD `831d8a642632b0f5904fb603d3753d41f2ec73fc`。唯一原产品修改为 [editor.go:951–964](<F:/AIproject/LevNote/native/ui/editor/editor.go#L951-L964>)，新增 [源码大纲测试](<F:/AIproject/LevNote/native/app/source_outline_navigation_test.go>) 与本交接。
- 真实首 RED 成立后完成最小 GREEN；四个公开行为控制逐片通过，单一 focused 联集实际 **5 PASS / 0 FAIL / 0 SKIP，exit0**。这不是最终 affected gate 或 runnable/platform 接收。
- 原1214产品中只 editor.go 改变，其余1213逐项 bytes/SHA256不变；原534excluded与 CON 仅 lstat metadata，前后及首 RED 对照一致。七个官方exe/gate/modsum pin均保持。
- 实现与新测试已定稿；本交接写毕后执行一次只读冻结审计，声明 STOPPED_WRITING 后由 lead 独立双轴审阅、独有最终 affected gate。

## 目标与公开契约（type:goal）

首测试 [1–64行](<F:/AIproject/LevNote/native/app/source_outline_navigation_test.go#L1-L64>) 使用真实内建 Markdown App、公开源码按钮、真实大纲 List Click/Enter、Tester Key。固定1180×780；先打开侧栏再建立目标列，跳转后没有 editor click。原64行/2043bytes完整保存于 [first-test.raw](<F:/AIproject/LevNote/.localtools/source-outline-20261114/red-01/first-test.raw>)，SHA256 `7e4b8035ee20904039021a506e0d03cd7ccc49907dd23ba89ef034a652463e94`；追加控制后仍为逐字节前缀。

```go
"iiiiiiiiiiii\r\niiiiiiiiiiii\r\n# Target\r\niiiiiiiiiiii\r\n"
```

UTF-16行起点字面值0、14、28、38。CmdHome、8×Right、Down得到22；真实大纲 Target Enter得到28；下一次Down应38。首 RED实得46，确证旧第9列被沿用。期望来自固定字符数，不从布局/私有desiredX/应用解析器生成。

仅在 Buffer.SetSelection成功后清 `hasDesiredX`；组合错误仍先报告/Focus/return，失败range仍报告并保持旧目标列、ensureCaret与Focus。既有word导航、Task32双击修正、affinity、数据/历史逻辑无改动。`fail(nil)`没有外部副作用，成功分支不调用它保持同等语义。

公开控制均在 [同一新增测试文件](<F:/AIproject/LevNote/native/app/source_outline_navigation_test.go>)：

| 控制 | 字面行为及保真观测 |
|---|---|
| [连续Down:66–76](<F:/AIproject/LevNote/native/app/source_outline_navigation_test.go#L66-L76>) | 22→短heading36→46，仍保留初始preferred column；源与history不变 |
| [ShiftDown:78–94](<F:/AIproject/LevNote/native/app/source_outline_navigation_test.go#L78-L94>) | 大纲28后ShiftDown选28..38，字面选中 `# Target\r\n`；源/history不变 |
| [失败Reveal:96–133](<F:/AIproject/LevNote/native/app/source_outline_navigation_test.go#L96-L133>) | 真实Type/Undo先建立RedoDepth1，Right×2/Down选36；Reveal(-1,-1)报告ErrRange，selection/source/version/history不变；Down48确认第11列保存，Redo恢复字面插入与23，Undo恢复原源与22 |
| [组合拒绝:135–177](<F:/AIproject/LevNote/native/app/source_outline_navigation_test.go#L135-L177>) | 同源52byte公开Buffer限额，真实Tester.Compose("中",1)超限；Flush得到同一pending error，Reveal28仍选36且source/version/history不变；Escape恢复后Down48保留旧列，再成功Reveal28/Down38重置列 |

## 已试方案与失败原因（type:dead-end）

- 首 [RED raw:1–6](<F:/AIproject/LevNote/.localtools/source-outline-20261114/red-01/stdout.raw#L1-L6>)：`selection = {Anchor:46 Head:46}, want literal [38,38)`；22和28的前置断言通过。此为真实产品RED；本阶段零fixture失败。
- Find旧列候选已排除：查找revealSource会ReleaseView→releaseGeometry，现有清列路径生效。本修正只针对可达源码大纲direct Reveal，不引入第二缺陷。
- 原始测试/证据、534excluded、.29和所有闭合历史包保留；未启动Git、网络、full suites、vet/build/打包、物理Windows/IME验收或child delegation。

## 实际执行与冻结证据（type:progress）

固定离线 Go1.27.3、gate、vendor，所有命令 cwd `F:/AIproject/LevNote`，有效 Go cwd由gate固定到native。统一argv形状：

```text
<BASH> .localtools/gate.sh test -v -count=1 -run <EXACT_PATTERN> ./app/ -timeout 90s
```

各次完整argv、wrapper/gate/actualGo PID、actual image/created UTC、执行UTC、exit及raw SHA/bytes均在独立create-only receipt。每次before/after对原1214+newtest/7pins/excluded534+CON/HEAD完整守卫。所有七次 `inputs_equal=true`；没有重跑已消费目录。

| 调用/收据 | wrapper / gate / Go PID | UTC起止 | exit | stdout bytes / SHA256 |
|---|---|---|---:|---|
| [red-01](<F:/AIproject/LevNote/.localtools/source-outline-20261114/red-01/receipt.json>) | 29724 / 31968 / 32296 | 07:59:33.531062Z→07:59:39.196780Z | 1 | 293 / `e9e278d455d3be6f2309b86c0359608eba79a867827fcfb286d5227a95c069a2` |
| [green-01](<F:/AIproject/LevNote/.localtools/source-outline-20261114/green-01/receipt.json>) | 29980 / 21232 / 25452 | 08:00:53.831370Z→08:01:32.588806Z | 0 | 192 / `81c60587b044bbc3b75bd6e16801ae5e508328636b2eb2836e6456b287e7594d` |
| [sticky-column-01](<F:/AIproject/LevNote/.localtools/source-outline-20261114/sticky-column-01/receipt.json>) | 20964 / 15460 / 28032 | 08:03:23.485276Z→08:03:31.065596Z | 0 | 192 / `73293147863b625beaf100c31343db5a0a0be63844a021bacadb8c58a69cb31d` |
| [shift-down-01](<F:/AIproject/LevNote/.localtools/source-outline-20261114/shift-down-01/receipt.json>) | 22300 / 18328 / 30588 | 08:04:23.764652Z→08:04:30.672389Z | 0 | 190 / `313aa5861e6cb27304b3c027f69b47ce699dca04859b269ddc364aee1c2705ff` |
| [invalid-reveal-01](<F:/AIproject/LevNote/.localtools/source-outline-20261114/invalid-reveal-01/receipt.json>) | 4104 / 26008 / 28808 | 08:06:19.525750Z→08:06:26.811065Z | 0 | 206 / `261a3c6eeb623e58930f8235a1a6d01f21a142fedafa8786ac0455f55abf5bfe` |
| [composition-01](<F:/AIproject/LevNote/.localtools/source-outline-20261114/composition-01/receipt.json>) | 27280 / 14040 / 10168 | 08:08:14.948963Z→08:08:22.610713Z | 0 | 222 / `b9fa6e53e20573cd44c6001b4b20a5fdda397b182c4c12556cb19becaecb8a39` |
| [focused-union-01](<F:/AIproject/LevNote/.localtools/source-outline-20261114/focused-union-01/receipt.json>) | 28516 / 18084 / 31476 | 08:09:12.653955Z→08:09:24.016985Z | 0 | 834 / `83d277dfb5f4732ed429c6892b2d59f59fd913b7fe02f59ab37590e05846e34f` |

全部stderr为0bytes，SHA256 `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`。联集 [raw:1–12](<F:/AIproject/LevNote/.localtools/source-outline-20261114/focused-union-01/stdout.raw#L1-L12>)逐项保存5PASS。

[限定审计结果](<F:/AIproject/LevNote/.localtools/source-outline-20261114/audit-01/result.json>)实际exit0：08:15:07.915719Z→08:15:08.378465Z。原source去替换唯一成功Reveal分支后与备份byte全等；首64行前缀全等；原1213不变；7pins/excluded534/CON与首RED全等；gofmt -d PID24292/exit0/stdout+stderr空，无format写入。

| 冻结源输入 | bytes | SHA256 |
|---|---:|---|
| [editor.go](<F:/AIproject/LevNote/native/ui/editor/editor.go>) | 25651 | `52d536f5c0ad7e166406a9e21b07b10fbee1a779aa7ba5ec4f017320529eeb6f` |
| [source_outline_navigation_test.go](<F:/AIproject/LevNote/native/app/source_outline_navigation_test.go>) | 6715 | `1ea8615e0fc6b2bd65a81c4519dd763a6c70c9c1fd51ff432e9f650eec363099` |

下一步由lead核对单独冻结收据、本交接及首RED/联集raw，再执行独立Standards+Spec；修正通过后root专属一次最终affected gate，旧1214和本5test focused证据不替代该阶段。
