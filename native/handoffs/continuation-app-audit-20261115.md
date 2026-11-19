# App 重载恢复修复交接

基线：`f2c2ceb82acecb40fb0502c047e41ba12a12be47`。实施范围仅两个生产文件、新增公开流程回归、本交接及独立证据目录。状态：**STOPPED_WRITING（生产源码与测试）**；root 已完成12项最终聚焦验收，全部通过；最终受影响门禁与独审由root统一收尾。

## 已修复的问题（type:goal）

P2：用户确认放弃修改并重载磁盘内容后，已落盘的旧 dirty Draft 未被及时清除。异常退出后的新 App.Restore 按 Draft 优先规则重新打开被放弃文本。正常关闭可能重写恢复快照，故行为断言直接建立新 App.Restore，不以正常退出充当复现。

根因是 Revert/Buffer.Restore 不产生 Editor.OnChange，先前成功恢复保存已消耗 persistPending，模态退出仅安排 autosave 而未重新标记请求。公共 App.Reload 在 Undo 回到 clean 后也会遗留同一旧 Draft。

## 最终实现（type:state）

- [app.go](<F:/AIproject/LevNote/native/app/app.go#L1284>)：公开重载在 Flush 前后固定目标所有者；成功后调用唯一 finishTextReload 收尾。该收尾同步 Buffer/ReadOnly/Conflict/source 呈现，直接调用既有 persistTabs，不新增保存通道。
- [workspace.go](<F:/AIproject/LevNote/native/app/workspace.go#L368>)：真实冲突栏在 native MessageOptions/MessageResult 边界设置可替换 hook，默认仍调用原生消息框。确认期间复查 tab/document/buffer/editor/view/adapter/path/mode/name、版本、编码、BOM、磁盘基线；确认后再次 Flush 并检查，过期确认保留新内容。
- 快照失败明确显示“已采用磁盘版本；恢复快照更新失败，旧恢复快照仍保留”。消耗本次待保存请求，既有错误去重保持；每次失败显式重设 Message，避免此前相同错误被去重后藏在其他状态文案后面。
- [reload_recovery_test.go](<F:/AIproject/LevNote/native/app/reload_recovery_test.go>)：只经公开 App.Reload、真实 App.View/Tester 冲突按钮、原生消息框边界和新 App.Restore 验证；Document/Buffer/session 全部为生产实现。原磁盘文件字节单独比对。

## 已执行证据（type:progress）

每轮都保留完整 stdout.jsonl、stderr.txt、exit.txt、实际命令、前后源码及工具链 SHA256；失败记录未覆盖。执行入口为固定本地 Go1.27.3 的 gate。

| 证据 | 实际结果 |
|---|---|
| [red-01](<F:/AIproject/LevNote/.localtools/continuation-reload-20261115/red-01/stdout.jsonl>) | exit1，公开 Reload 后 fresh Restore 重现 `new=true dirty=true text="old recovered draft\r\n"` |
| [green-01](<F:/AIproject/LevNote/.localtools/continuation-reload-20261115/green-01/stdout.jsonl>) | exit1，编译失败 `a.Persist() (no value) used as value`；误用 void 返回已改为 persistTabs |
| [green-02](<F:/AIproject/LevNote/.localtools/continuation-reload-20261115/green-02/stdout.jsonl>) | exit0，公开 Reload 的同一行为 oracle PASS |
| [red-02](<F:/AIproject/LevNote/.localtools/continuation-reload-20261115/red-02/stdout.jsonl>) | exit1，真实冲突栏确认后 fresh Restore 重现 `confirmed discard resurrected old draft` |
| [green-03](<F:/AIproject/LevNote/.localtools/continuation-reload-20261115/green-03/stdout.jsonl>) | exit0，公开 Reload 与冲突栏两个同一 oracle 均 PASS，5/5 前后 pin 相同 |
| [review-01](<F:/AIproject/LevNote/.localtools/continuation-reload-20261115/review-01/production.diff>) | 最终完整生产差异、三份源码快照；gofmt exit0，限定 diff-check exit0 |

[final-01](<F:/AIproject/LevNote/.localtools/continuation-reload-20261115/final-01/stdout.jsonl>)：root实际运行 `^TestReloadRecovery`，exit0，12 PASS、0 FAIL、0 SKIP，stderr空，5/5 before/after pin相同。app-audit另行完整读取52行JSON原流、实际命令、退出码、stderr及前后pin，确认12个Test身份与最终冻结源码一致。

最终三份源码 SHA256（与 green-03 的中间实现不同，已由root最终聚焦验证）：

```text
42db50a2d868d604b6a266dc950efdc2c9987d1dbdcf51f527d0c26eeac6b220  native/app/app.go
e0ef1828da249652c72b4e4828fbe9fcd1ab1531fdc21ce3443b2007e58f3c1a  native/app/workspace.go
fb90f7b76ec98ae8ff56fdc2c79347f2e38f6c6b17063c03bbd6da2b1be650ac  native/app/reload_recovery_test.go
```

完整 [pin 清单](<F:/AIproject/LevNote/.localtools/continuation-reload-20261115/review-01/source.sha256>) 同时包括 gate 与 Go 可执行文件。

## 新增控制与剩余验收（type:progress）

6 个顶层测试、6 个子测试，共 12 个 Test 身份。两个原始成功断言已有 RED→GREEN；随后加入的控制按 lead 指令冻结，由 root 一次执行 `^TestReloadRecovery`，最终全部 PASS。

- 取消、原生对话框错误、确认后真实移除磁盘文件造成读取失败、公开 Reload 遇 dirty 文档：原草稿、冲突状态、旧快照字节和 fresh Restore 保持。
- 原生确认期间公开新建并激活另一文档：原目标草稿与新目标草稿保持；旧恢复快照仍可恢复。
- 原生确认期间同一 Buffer 收到新编辑：新编辑保持，过期确认明确报告。
- 两个入口分别面对真实其他草稿超过 4MiB：使用实际 SnapshotBuilder 预算失败，不替换持久化实现；磁盘版本已采用而旧恢复快照保留；相同错误只占一个 Issues 条目、Message 仍可见；旧 timer 投递不重写；修正其他草稿后显式重试恢复完整会话。

失败控制覆盖的是真实恢复预算错误，未声称覆盖 OS 磁盘空间耗尽或 Windows ACL 拒写。新原生对话框 hook 检查程序流程，不代表物理 Windows 输入法或真实弹窗验收。

下一步：root完成统一受影响套件及独审；聚焦验收已按上述final-01执行并独立核读原流，此子代理保持源码冻结。旧已接受包、旧证据、共享白板及其他归属文件均保持原状。
