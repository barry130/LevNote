# FINAL06 列表增量原字节导入与 Root 有限闭合

日期：2026-11-13。Root：`lead`；Task6 owner：`9e20bc49-fddb-490b-a752-c3b11da86ccc`；唯一产品命令 owner：`a753a04c-1383-4870-877c-e70639eb0422`。

## 1. 决定与边界

type:progress

Task6 的列表生命周期、最小窗口有限源码/Linux 回归及原字节镜像收取完成，可关闭共享任务并移交 Task10。该决定不是完整 app 测试、持久图像视觉审查、package build、Windows GUI/IME/PE/C:config/WM_CLOSE/Job/物理输入或整应用可交付验收。组合 host WIP 不是某个 lane 的已测试快照；产品源码仍未提交或推送。

冻结开发基线仍为 `8897244ffa4f00f5ec51f95545e7a3ca27d37d97`。原 guard、原 FAILED、05 失败、旧报告、未执行本机草稿 provenance 和已闭合 labels 均保留；无重新执行 decoder/importer、refreeze、product command 或本机 Go。

## 2. 已接受的有限回归与实际源审查

- FINAL06 focused：Go/PGID `2058`，Go0/wrapper0，真实 selector `^TestWorkspaceResultsMinimumWindowControls$ ./app`，light/dark 两分支通过。
- FINAL06 affected：Go/PGID `2782`，Go0/wrapper0，逐字复用原25 selector；25 top-level 与22 nested 全 PASS，无 FAIL/SKIP/panic/timeout/cleanup/remaining。Root 已全文读取 raw/numeric/final，并独立核验全部1194实际 input/run files、三份 roots/files 清单，`17000651` B。
- 05 的原最小窗口失败仍独立存在：source Rect `{X:228,Y:477,W:0,H:0}`；06 只将 results Column 的 `Shrink(0)` 改为 `Shrink(1).MinHeight(110)`，保留230与height-260公式、原tests、source minimum、worker/Keys/guards。
- Root 审查了实际 host working-tree 六个 tracked文件 diff，而不是空的 base...HEAD；新 helper86行、standalone245行、combined399行已全文审读。保留 window-lifetime Handle，立即 CancelFocus，下一安全 prebuild boundary 才替换 obsolete ListState；late frame 的活指针不被 reentrant Release/Flush破坏。
- Tree/Search/Outline 的 captured Key 在 stale 后仍逐行唯一；内容、动作与 pass-local selection 通过 owner/version/tab/path/generation再确认。bounded retained entries只按Name稳定排序，dirs-first、读取limit+1、truncation、取消、错误与交付generation不变。Acceptance仅补真实 `searchRoot=workspace`。
- t.TempDir PNG正常清理后已经不存在；断言通过不等于Root持久图像视觉审查，不为截图重跑。

先前回归与源码审读细节见[05/connector review](<F:/AIproject/LevNote/native/handoffs/autonomous-final05-and-connector-root-review-20261113.md>)和[06/mtime review](<F:/AIproject/LevNote/native/handoffs/autonomous-final06-and-document-mtime-root-review-20261113.md>)。

## 3. 原字节导入及独立 archive→manifest→target 核验

[归档](<F:/AIproject/LevNote/native/handoffs/autonomous-experience-20261112-evidence/transport-06/final06-review.tar.gz>)：85709 B，SHA256 `25b935ae97a020bed1552c4fb5468130dabd8a884a6080826bfde8d91e288621`；tar listing实际52 regular成员，无directory/link/absolute/..成员，按路径排序。

[清单](<F:/AIproject/LevNote/native/handoffs/autonomous-experience-20261112-evidence/delivery-06-incoming/payload/native/handoffs/autonomous-experience-20261112-evidence/delivery-06/transport-inputs.json>)：655行/37274 B，SHA256 `ac267abf0777dd81eaf7bef2f9cc64fa862e3850909719f0df706972553d4cdf`，51记录加manifest自身为第52文件，payload总375017 B。Root全文读完655行，使用原生stat逐项核验52个payload、11个backups、九个current source及新旧reports的bytes/fullSHA。

Root独立只读222ms的box fs/crypto/assert audit核对全部51 actual origin、52 actual payload与manifest；整包按实际成员顺序拼接 SHA256为 `f0ead14ae69d14b254a31b77dc14f9d00dac95221d91320b1e051bd9e49c3b47`，与host直接 `tar -xOzf` 的真实payload一致。之后host对实际52成员逐个 `tar -xOzf` 计算SHA并与已native-stat校验的target逐个比较，完整52/52返回0，直接闭合各成员body而非仅接受importer自述。

第一次 while方案因tar读取pipeline stdin仅核1项，未用作52项通过；随后for方案因Windows tar列表CR导致找不到成员、exit1，未写任何文件。最终只对已listing证明无空格/控制路径的52固定成员去除列表行尾CR，全部成员逐项通过；归档与payload未改写。

[decoder receipt](<F:/AIproject/LevNote/native/handoffs/autonomous-experience-20261112-evidence/transport-06/decoder-receipt.json>)：PID19840/exit0，114280encoded characters/10parts，仅解码。原179行[importer](<F:/AIproject/LevNote/native/handoffs/autonomous-experience-20261112-evidence/transport-06/import-final06.py>)只执行一次，PID22144/exit0，04:06:25.666876Z–04:06:26.201269Z；[import receipt](<F:/AIproject/LevNote/native/handoffs/autonomous-experience-20261112-evidence/delivery-06-incoming/import-receipt.json>)228行/9458 B，SHA256 `12f3686ce555980295060236652ed6a751016642768ed35fffd41727ccd6be7c`，productCommands0。

[backup receipt](<F:/AIproject/LevNote/native/handoffs/autonomous-experience-20261112-evidence/delivery-06-incoming/backup-receipt.json>)197行/8029 B，SHA256 `f6ca74f166d56b79e86520a99e56fd52fa8eff7d10cb2e78ce1f9b13b8ca2e1d`，在任何mirror前持久保存11原文件及premirrorreceipt。四差异源view/workspace/outline/workers替换，另五已final源未重写；四旧draft和全部其他备份均独立bytes/SHA核验。两旧[experience report](<F:/AIproject/LevNote/native/handoffs/autonomous-experience-20261112.md>)19105 B/`5f16c8c368b043475bc3fab36217bbbd72de8748ce0751918d704033edfe55aa`与[reliability report](<F:/AIproject/LevNote/native/handoffs/autonomous-reliability-20261112.md>)34988 B/`503dbf1c32d21865774379ebfeea419a9fa57ac77994cb2ddc51c6f2de9aaf27`保持。

新[FINAL06 report](<F:/AIproject/LevNote/native/handoffs/autonomous-experience-final06-20261113.md>)另命名创建，127行/21483 B，SHA256 `320979a87b66032f88b33d467ce0f44eb837c59ad868edec58528cec3e1611c8`。本small package不复制完整native/vendor/toolchain/maps；大清单原文以box全路径/bytes/fullSHA引用保留，不能称完整备份。

## 4. 九份最终源身份

| file | bytes | SHA256 |
|---|---:|---|
| [app.go](<F:/AIproject/LevNote/native/app/app.go>) | 39471 | `b4e8b96ac45a336d3cc2524f513fe652f6cd638c075ad53fb590c509d09fdbde` |
| [view.go](<F:/AIproject/LevNote/native/app/view.go>) | 24375 | `8b702c15dfcd265821ec10fe7a8f52927f6ffe667a349ff04b8b281c67ba3b77` |
| [workspace.go](<F:/AIproject/LevNote/native/app/workspace.go>) | 8339 | `d9f458f1cb4d597932b58161f546f1e43874aa5eeb9bf25eb9c47ca40519177a` |
| [outline.go](<F:/AIproject/LevNote/native/app/outline.go>) | 15523 | `c75435237f5d9eb9b466ed8b755cc8c22818f5dc4a6a57ac2045ae7136cad8cf` |
| [list_lifetime.go](<F:/AIproject/LevNote/native/app/list_lifetime.go>) | 2345 | `14c4d6cc436a6633ca5055fb18b6bc5b63703ebce0d7c8a817092ea5bdf24bb5` |
| [list_lifetime_test.go](<F:/AIproject/LevNote/native/app/list_lifetime_test.go>) | 17378 | `982e3ec121e9d5c020df3e9d4661748e894186cf91976de118d32be47da120a2` |
| [list_handle_lifetime_test.go](<F:/AIproject/LevNote/native/app/list_handle_lifetime_test.go>) | 7900 | `ba98054601b1bfd421d1fe721cf8ea6cb97a01081f89a073507f17e3d293d5e5` |
| [acceptance_test.go](<F:/AIproject/LevNote/native/app/acceptance_test.go>) | 25229 | `0d54cd2308b545d98cb3f2d81b35ef367c310b66658e2b3ea4672dd7edd6e362` |
| [workers.go](<F:/AIproject/LevNote/native/app/workers.go>) | 8225 | `73c65d6aedc8b517c808314aae439f38c1c4137b013c2ca2db885e1fc88d9b83` |

合计148785 B。box frozen originals、payload、host current九源完全一致。

## 5. Task10 移交决定

type:goal

Task10的新[workroot](</home/user/levnote-dev-experience-task10-20261113-01/source/native>)由实际已格式化06继承，不来自组合host WIP。Root完整读取44行[input derivation](</home/user/levnote-dev-experience-task10-20261113-01/coordination/input-derivation.json>)与86行metadata helper；另573ms只读全树核验1194文件/17000651 B，source-before/source-after/target全部与原expected roots/files一致，保留head metadata。

derivation PID6266/productCommands0/semanticChanges0；expected原202703 B/`ef86a56bd0d035060a9ff403df4e0afc29c6757b2fe3244b62cbbe275a44a73b`、start-map199750 B/`b6260b46019c40e14bed3ae443c258cda019a026b13ca244b14c965d7b4b33b5`、derivation1562 B/`8f369117e4062bf4d0b8935ec7db7c6dacd7e09c9b6e4a3a4b1667d04104ee5a`独核。

同experience owner接续Task10，vertical顺序为history notice→bounded CopyAll真实clipboard acknowledgment/failure→toggle动作。已确认实际public seams为 `Buffer.SetHistoryLimits`、`Context.WriteClipboardThen`、`Tester.FailClipboard`；不得使用不存在的SetHistoryBudget或Clipboard.Set，不改core预算/flag/group或MyGo/vendor。

Root实读[旧settings test:1296–1316](<F:/AIproject/LevNote/native/app/app_test.go#L1296-L1316>)，批准窄扩仅两行：1301条件`!a.SoftWrap`→`a.SoftWrap`、1306默认true期待`显示行号`→`隐藏行号`。全部bounds assertions及其他tests保持；新动作label需另公开验证真实toggle与持久化，不能把旧错误期望当规范。

最终全应用box语义集成、正式独立parallel Standards/Spec、全suite/vet、真实平台与包门禁、文档/CI恢复及双main产品交付仍待完成。
