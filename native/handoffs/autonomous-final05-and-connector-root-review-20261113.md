# FINAL05 与 draw.io 连接源限定回归 · Root 独立复核

日期：2026-11-13。普通开发基线为 `8897244ffa4f00f5ec51f95545e7a3ca27d37d97`。
本文件记录独立源码与证据复核，不是正式 Standards/Spec 双轴审阅、全任务关闭、打包或平台验收。

## 执行与来源边界

- 所有 Go/gofmt/test/vet/build/run 仅由既有授权 box 的唯一命令 owner native-reliability 执行。Root 未执行产品、测试或验证 helper，未修改语义源码。
- Root 实读已有 raw、64 行 numeric 和 142 行 result；只读 Node fs/crypto/assert 比对返回 exit0、1298ms。该调用没有文件写入、进程启动或产品运行。
- 四次结果的完整 primary SHA、PID/PGID、argv、数值退出码、隔离环境、expected roots/files 与三个完整清单均核对。每次 stderr 为零，timed_out=false、owned_cleanup=[]、owned_group_remaining_after_wait=false。
- 清单逐项核对覆盖全部1194或1195条记录；实际源码另外核对下述九个 App 文件或六个 draw.io 文件。此处不声称该脚本重新遍历全部仓库源文件。
- 已消费的旧标签不重跑，FAILED 与后继 PASS 各自保留。

## FINAL05：focused 通过，原25项保留一个真实布局失败

[冻结05源码](</home/user/levnote-dev-experience-20261112-01/revision-05-source>)的九个 App 文件与 focused/affected 实际 run/source 及 input/native 各自完整长度/SHA相等。
原7900B公开Tree/Outline oracle保持 `ba98054601b1bfd421d1fe721cf8ea6cb97a01081f89a073507f17e3d293d5e5`，与真实03父文件逐字节相等。测试与断言未为Linux放宽。

| 结果 | Go PID/PGID | Go / wrapper exit | 已到达覆盖 | 清单记录 / 字节 |
|---|---:|---:|---|---:|
| [focused05](</home/user/levnote-dev-reliability-20261112-01/runs/experience-list-final05-focused-01/evidence/result.json>) |30369|0 / 0|原Tree与Outline两项PASS|1194 / 17000636|
| [affected05](</home/user/levnote-dev-reliability-20261112-01/runs/experience-list-final05-affected-01/evidence/result.json>) |31061|1 / 2|原25项全部到达，24PASS/1FAIL；nested20PASS/2FAIL|1194 / 17000636|

sole failure为 `TestWorkspaceResultsMinimumWindowControls` 的light/dark：
`list_lifetime_test.go:376 results+Find leave no usable source input: ui.Rect{X:228, Y:477, W:0, H:0}`。
Root实读实际失败测试316–399及View1–181：Find在Results之前，主栏ClipY下不可收缩结果面板使源码输入被裁至零尺寸。该结果不是点击或焦点PASS，也不是25项通过。没有panic、skip、timeout或未启动项。

| Primary | focused05 SHA256 | affected05 SHA256 |
|---|---|---|
| numeric |61298c546fc3f732581224db3149a41857991169d3d14387898d1cd83242b366|84490fa4a9374201e17889c8f38672cb82fed0253f2650741232880d1d3faa53|
| result |b351d395a4165ec25c7194ba214b0b8e82a6a40997414429c68a5c4794ebeb22|a0e1e93520122ef06a1f41db0cb0dd1ada7f54e9376820bc243de5032bfa2ec0|
| stdout |d4e4eb37abea31d72f0dbd12a03cfc431e41f7a0d42c545c99af27eeeefcdc8b|cedc8eaa472b51e81b30ec9333c36aeb9d0e0203f224c9ef5737f30304fc6583|
| expected |5ef542c70b996fe30cbc3a96d685ebab611766b998ef34320bc31ff1a300047e|0253fb9e0412ca392a6ae45ecca339ccac869aeb6ac119f0f60f782f2307cfef|
| each of three maps |d7a413c025e6eafccd497ea4ff5430b61bfae85d91c32e5eee6bd5042ac7f1b4|d7a413c025e6eafccd497ea4ff5430b61bfae85d91c32e5eee6bd5042ac7f1b4|

## draw.io：相同公开连接按钮oracle的行为RED→限定GREEN

[连接源RED](</home/user/levnote-dev-reliability-20261112-01/runs/drawio-connector-source-ui-red-01/evidence/result.json>) Go/PGID31763、exit1、wrapper2：
真实边线点击及源码/caret7..23/version/history不变前置检查通过；点击真实“连接两个节点”后visible=true、Disabled=false并进入“选终点”模式，132行严格拒绝断言失败。顶点正控制未到达。

[连接源GREEN](</home/user/levnote-dev-reliability-20261112-01/runs/drawio-connector-source-ui-green-01/evidence/result.json>) Go/PGID32393、exit0、wrapper0：
同一9041B公开oracle完整SHA `363c22c7e6e438fe9885c9cc2e1659c0ca0a02ea7a50c4428d26ad5e9c5609ea` 与实际RED相等；边线源Disabled拒绝和真实顶点enabled正控制均到达PASS。原5484B边线Delete/Undo/Redo测试前缀与上次实际edgeGREEN相等。

只有 [view_ui.go](</home/user/levnote-dev-reliability-20261112-01/runs/drawio-connector-source-ui-green-01/source/native/modules/graphview/view_ui.go#L131-L147>) 三行改动：按钮Disabled增加既有 `selectedDrawioNode()==nil`；Flush后仅从当前selectedDrawioNode.ID赋connectSource。不改变Mindmap/其他按钮、字段历史、parser、adapter、canvas geometry、vendor或App。
Root以实际RED全文进行三处唯一literal替换后与实际GREEN全文完全相等；其他五源长度/SHA相等。RED ViewUI10107B/SHA `86c111a29b6ecc87f308fa2660ae738fa9d964402f8875cec326db60860f8328`；GREEN10160B/SHA `d27e5db05a32f107014322cdf5d2509a5d4d8442d5c1748dfd0747c7fcec12b3`。

| Primary | connectorRED SHA256 | connectorGREEN SHA256 |
|---|---|---|
| numeric |c0ba9adef4369d372d0f177955ebcf2b952b01ee2ea834a9fb5c480015e62d07|cd0de56f4765e7ce19bcfa8e82b3f38cdb97b16a38d07bf0cfe906d904f5d3d8|
| result |fbce9ba6c2e837537ace804fdac5289942c0cae804ccbc504e0b7a28b85b59f7|4cdcaf96c341b2e8a54f185b94c0b6f6f1b71a9f6fab339142f19ea06022d125|
| stdout |6411e6256e2efdc3ee47a7fa7826d8d5e6f04c8ece6ad020538e5eda33f9bae2|52a2e4343ec5a6dc87d3b5ecffd5e6f4474ce12c43b67aac641b40f7ccbed0f4|
| expected |05fd4266e548d8c0ab2ddf3617df1f03b5d64285d39f428c9a004e62d8285a78|011c8451335e2fc16c14a0f40ace56650ba0ff490ef8c945d92860021cbf345c|
| each of three maps |f2610c12115ae099e503077432d9ee4f7a2624b79c6d9fb336f94eda003e0871|13243687d7924be5aeeb13b81aaa6f0ffa4ce41416ee26b9be31f8cec739e61d|

RED清单1195文件/16996888B，GREEN1195/16996941B，仅ViewUI增加53B。该结果只接受连接源限制的Linux公开用例，不代表全部Task8、stale-owner历史保护、field/IME、格式完整拒绝、完整两包tests/vet或Windows输入已通过。

## FINAL06冻结输入审阅与下一步

Root实读[06 manifest](</home/user/levnote-dev-experience-20261112-01/provenance/task6-freeze-06/source-inputs.json>)114行/4605B/SHA `f864c426b7d200533641839fea12b4907241050778ba46a4470256773fbbb2dd`，及[05→06 patch](</home/user/levnote-dev-experience-20261112-01/provenance/task6-freeze-06/revision-05-to-06.patch>)11行/517B/SHA `a24f2833b397deccf45c308c6ddd3b743fba4fdb253419fd86a19b130405a64e`。
另一次只读script exit0/510ms核对九个实际父源：八源exact05；workspace8324B→8339B/SHA `d9f458f1cb4d597932b58161f546f1e43874aa5eeb9bf25eb9c47ca40519177a`仅149行 `.Shrink(0)`→`.Shrink(1).MinHeight(110)`。原230/height-260偏好公式、110下限、Find与sourceEditor min80、所有测试/Keys/worker/焦点/selection/guard均保持。
作者5967源/533测试全遍历恢复核对为独立作者证据，Root本次九actualpins核对不冒称第二次5967遍历。

06执行继续由唯一owner以新input/newlabel进行，原失败单例先行，实际完整PASS后原25项只执行新后继一次。Root收取现有结果，不重跑为补截图；temporary测试PNG若已删除，不声明已审图。Task6最终镜像与报告、Task8生命周期及完整包回归、Task11公开保存回归、Markdown与issues收尾仍继续。整体交付未接受。
