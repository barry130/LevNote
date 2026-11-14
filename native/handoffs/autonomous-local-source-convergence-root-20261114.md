# 本地原生源码收敛摘要 · 2026-11-14

**状态：Root 已全文审阅并接收本篇的有限源码结论及三份当前说明段，不是应用交付接受。** 本文由 `windows-delivery` 按 task-24 的四文档范围编写，根代理完成原始回执核对、全文与差分审阅并整合 Task26 显示歧义澄清。Task22/23 双轴补篇及独立原始回执审计均已接收；旧历史内容保持，源码、命令与应用接受边界分开。

**结论：当前有限源码收敛与六产品根 test/vet/build 已完成；不是新可运行包，也不是应用完整交付 GO。** 当前已接受的可运行版本仍为 `native-worker.20261110.29`，见[原生说明](<F:/AIproject/LevNote/native/README.md>)及[原 `.29` 根验收](<F:/AIproject/LevNote/native/handoffs/composition-release29-root-acceptance.md>)。历史包、原报告、FAILED 调用、Task20 准备快照和旧 box 证据均不改写；旧 box 不是当前开发或验收执行面。

## 证据来源与身份

本篇作者完整读取了[根独立审计回执](<F:/AIproject/LevNote/.localtools/root-integrated-audit-20261114-01/result.json>)和两份双轴补篇，读取了下表各命令回执的身份/环境/退出段及两份失败原始 stderr；没有重新执行产品、Go、formatter、脚本、Git、网络、MCP 或其他代理。完整原始 test JSON 的逐事件重计和全树逐成员校核由既有根审计完成，不冒充本篇作者重跑或再次独立重计。

| 既有证据 | 字节 | SHA256 |
|---|---:|---|
| [Root 独立原始回执审计](<F:/AIproject/LevNote/.localtools/root-integrated-audit-20261114-01/result.json>) | 28,952 | `415388947b57e847921f47aef6537358ac6fdb50fa7f52e17cc5eed5783619a9` |
| [六根 test 回执](<F:/AIproject/LevNote/.localtools/final-native-20261114/test-01/result.json>) | 15,876 | `1a3c4abf56d7d05280db4ac2a9ff694c891c7dfb68d4850d910ed8933f097e5a` |
| [六根 vet 回执](<F:/AIproject/LevNote/.localtools/final-native-20261114/vet-01/result.json>) | 1,445 | `952d7fcbd7f8725c9d9f1fed6c8c0ebc7a76107d33887796fa7d00d2b515f2ae` |
| [六根 build 回执](<F:/AIproject/LevNote/.localtools/final-native-20261114/build-01/result.json>) | 1,448 | `d249cdeb72af537bcbafcff8c8280df5715585ae912b0781605a5e7d6a182bc0` |
| [只读格式回执](<F:/AIproject/LevNote/.localtools/final-native-20261114/format-01/result.json>) | 2,400 | `2b8533d67f5a5488bb6fb7021345e18694292e83f005aad71bcc6a088181288c` |
| [打包失败回执](<F:/AIproject/LevNote/.localtools/package-root-20261114-01/result.json>) | 1,247 | `8b0d658ea6c254934bf792621d4d6a9429159b88f3a897ba15bd6a6c0ecdcfef` |
| [构建守卫失败回执](<F:/AIproject/LevNote/.localtools/build-guard-root-20261114-01/result.json>) | 985 | `105e741d6f384fcb828c9570580262b82e705658f44da334b23664036af15f0a` |
| [Task22 Standards 补篇](<F:/AIproject/LevNote/native/handoffs/autonomous-final-standards-addendum-20261114.md>) | 8,741 | `256cd7582b77e0947bf976b28d2caedf1bdb34f82ca00d01aaa99581ce20ccf5` |
| [Task23 Spec 补篇](<F:/AIproject/LevNote/native/handoffs/autonomous-final-spec-addendum-20261114.md>) | 4,114 | `dbc2304f3dcefb88f21d39cdc2b1ed24a9e6bfda951317646fc25d4ab8a41788` |

- [source freeze02](<F:/AIproject/LevNote/.localtools/final-native-20261114/source-frozen-02/source-frozen.json>)：1,211 个源码/构建输入文件；清单声明 aggregate SHA256 `874bc1fd23a3d02e432c9c6305842ee540afdca1cdad23692905da80d01b2bdf`，清单文件 SHA256 `5315612dc8654f158dae1a4aff7a3ab28a4d7c79180dcb09349b7034ad58aedc`。Root 审计时与当时成员匹配；该 aggregate 是清单身份，不另造聚合算法，也不把后来文档变更冒充原输入不变。
- [shipping 输入清单](<F:/AIproject/LevNote/.localtools/final-native-20261114/source-shipping-02.json>)：1,239 个文件，SHA256 `405d71a737a83f9563fb0e992ab4b9e78b24df37e8e129e30108ac9f55e5128a`。Root 审计确认其 ordinary/hidden/untracked 范围和 UTC ticks 匹配；这是调用前输入，不是已生成发行包，也不覆盖本轮稍后增加的文档。
- Root 报告审计实际退出 0；回执最终状态为 `RAW_SOURCE_AND_FINITE_COMMAND_EVIDENCE_VERIFIED_APPLICATION_NO_GO`。它逐项校核当前源码/shipping 输入、Task19 的 88 个回执成员及原件/8 个当前最终 pin、Task21 的 16 个原测试身份和当前修正，保留 RED/GREEN 的原始不同结果，不将 RED 当最终通过。原件保护仅指 Task13/15/19/21 各冻结 correction 入口中的指定原件；不宣称全部原测试树/断言从基线未变。Root 已逐项读取并审核六个既有测试文件 diff；[设置宽度测试](<F:/AIproject/LevNote/native/app/app_test.go#L1296-L1307>)的两行动作标签更正已有[原精确批准记录:66](<F:/AIproject/LevNote/native/handoffs/autonomous-experience-final06-import-root-review-20261113.md#L66>)：`!a.SoftWrap`→`a.SoftWrap`、默认true时“显示行号”→“隐藏行号”，全部bounds断言保留，并要求另以公开路径验证真实toggle/持久化。不是无依据改变原断言，也不能据此宣称整个原测试树未变。

## 已完成的有限源码与双轴复核

[Markdown 历史与生命周期整合](<F:/AIproject/LevNote/native/handoffs/autonomous-history-conflict-and-encoding-root-review-20261113.md>)及[Windows 保存冲突修正](<F:/AIproject/LevNote/native/handoffs/autonomous-windows-save-conflict-correction-20261113.md>)保留各自有限验证范围。当前补篇没有将旧 F1–F3 或已更正缺陷重新列为新失败。

- **Standards：** Root 已报告接收完整 [Task22 补篇](<F:/AIproject/LevNote/native/handoffs/autonomous-final-standards-addendum-20261114.md>)。旧 S1 在冻结的 draw.io Redo 同步回调释放/替换路径闭合，两个真实公开命令入口有已存 PASS；新增硬规则违例 0。H1 Issues 复制文本与面板重复为 **P3 heuristic / non-blocking**，不是强制重构或当前文案不匹配，也不是所有图生命周期都已验收。
- **Spec：** Root 已报告接收完整 [Task23 补篇](<F:/AIproject/LevNote/native/handoffs/autonomous-final-spec-addendum-20261114.md>)。两项旧 P2/T05 Issues snapshot 缺口在源码层闭合：实际应用身份/每次真实 UTC 时间及全输出 64 KiB 构造预算、UTF-8 摘录与三类省略/损失区分；本次未建立新的 actionable 缺失、部分实现、错误实现或越界发现。原跨秒 snapshot 相等断言仍有测试 flakiness 风险，不通过缓存时间或 fake clock 改生产语义。
- 旧 [Standards 报告](<F:/AIproject/LevNote/native/handoffs/autonomous-final-standards-review-20261113.md>)与旧 [Spec 报告](<F:/AIproject/LevNote/native/handoffs/autonomous-final-spec-review-20261113.md>)仍是原快照；新结论由补篇承接，不覆盖旧文件。以上是指定修正及 tracked/untracked 工作树范围的有限审阅，不宣称重新逐行语义审阅全部 1,211 文件。

## 六根本地命令门禁

已有执行使用 **Go1.27.3 windows/amd64**、vendor/offline/telemetry-off，隔离项目内 cache/tmp，`CGO_ENABLED=0`；以[本地 gate 的实际选择](<F:/AIproject/LevNote/.localtools/gate.sh#L6-L21>)为准。六个产品模式严格为 `./app/ ./cmd/... ./core/... ./formats/... ./modules/... ./ui/...`，不对含 handoff 快照的 native 整库使用 `./...`。在仓库根目录的当前对应命令是：

```bash
bash .localtools/gate.sh test -json -count=1 -timeout=300s ./app/ ./cmd/... ./core/... ./formats/... ./modules/... ./ui/...
bash .localtools/gate.sh vet
bash .localtools/gate.sh build
```

这是已存调用的说明，不是要求失败后重跑。gate 的 `fmt` 分支实际执行 vet，不能拿它宣称 formatter PASS；旧注释的 “eleven roots” 也不改变实际六模式。Offline Go 选择不构成 OS 防火墙或跨平台/默认 CGO 验收。

| 门禁 | 实际结果与有限范围 |
|---|---|
| test，一次 | exit0；RUN 3,068（1,228 parent + 1,840 nested）；**PASS 3,019（1,181 parent + 1,838 nested）/ FAIL 0 / SKIP 49（47 parent + 2 nested）**；26 个测试包 PASS、1 个 host 包无测试；wall **45.295191s**；stderr 0B，源码不变。parent 包含其 subtest 结果，不能把 3,019 当互相独立的 top-level 测试数。 |
| vet，一次 | exit0，stdout/stderr 均 0B；同一源码 freeze 前后不变。 |
| build，一次 | exit0，stdout/stderr 均 0B；同一源码 freeze 前后不变；不是 GUI host+四 worker 的完整打包。 |

49 个 SKIP 的名字和真实原因完整保存在[根审计回执](<F:/AIproject/LevNote/.localtools/root-integrated-audit-20261114-01/result.json#L34-L279>)：包括未显式启用的视觉/真实包/worker/资源门、symlink 权限不可用和 C: TEMP 条件不满足。SKIP 不算 PASS，也不代表这些平台门已通过。Tester、writer seam、allocation-only 结果不替代物理设备/真实剪贴板或整应用测量。旧 1,918 个 frontend 测试不作为 native 交付门禁。

## 格式、构建守卫与打包：失败保留，未接受

- **格式仍 held：** 原只读 `gofmt -l` 实际 exit0、stdout478B、stderr0B，报告 **8 个债务（5 个原有 + 3 个新增）**；包装调用由 root 报告退出 1、回执 `FAILED_PRESERVED`，不是格式 PASS。三项新增见[原格式回执](<F:/AIproject/LevNote/.localtools/final-native-20261114/format-01/result.json#L57-L60>)。已发生的写入/ACL修复通道停止，本篇不再次修复、不请求提权、不绕过、不改旧基线。
- **构建守卫 All 未运行任何 case：** [原 stderr](<F:/AIproject/LevNote/.localtools/build-guard-root-20261114-01/stderr.raw>)在原脚本 line20 创建 tools Junction 时拒绝访问，exit1，stdout0B/stderr266B，SHA256 `80a19c758990c9ec4ef2c57b5bc29d76047d98dde86734194027c18e7d0bd1fe`。这是独立环境失败，不升格为 guard PASS，也不换 copy/另一个 shell。
- **新打包没有 PE：** [原 stderr](<F:/AIproject/LevNote/.localtools/package-root-20261114-01/stderr.raw>)在原打包脚本 line24 创建 evidence 目录时 Access denied，exit1，stdout0B/stderr385B，SHA256 `39df0306422b71fc88a02baae14574149d58a0a09c539b76a866697016ad0cb1`。Root 审计确认拟议 `native/bin/levnote-native-worker-20261114.31` 与该次产品证据目录均不存在；**本次打包实际 GUI PE 构建 0、该候选 host/worker 启动 0**。这不表示整个会话没有任何 PE/进程：源码测试可能另建/启动 test helper fixtures，但不构成真实 shipping worker 接受。未预建目录、换输出、copy、改 helper 或用 ACL/shell 绕过重试；不得称 `.31` 已存在或 accepted。

## 可继续源码交付与应用 NO-GO 分开

[Task20 准备收据](<F:/AIproject/LevNote/native/handoffs/autonomous-native-ci-preparation-20261114/PREPARATION-RESULT.md>)及[禁用 CI 草案](<F:/AIproject/LevNote/native/handoffs/autonomous-native-ci-preparation-20261114/native-windows-ci.yml.disabled>)保持冻结、disabled/inactive，合同仍 false/空；本次不激活 CI、不改任何 workflow。准备 checker 是五真实 GUI PE/全部25载荷+自身清单的草案，不使用旧三-worker fake fixture 冒充四-worker接受；hash/header 一致性不等于真实请求执行。Root 随后已全文阅读两份新 PS，并在 PowerShell7.7.0 进行两文件 `ParseFile` 和模板 JSON 数据检查，实际 exit0、parser errors0；[单独 parser 回执](<F:/AIproject/LevNote/.localtools/ci-preparation-parser-20261114-01/result.json>)为4,464B，SHA256 `b63b6f6c69d88ff39a3dd91a6b3c9f73d01a868da0c0bb658239d6fdb28179ca`，9个准备输入的bytes/SHA/UTCticks前后一致，状态 `PREPARATION_PARSE_PASS_NOT_CI_OR_APPLICATION_ACCEPTANCE`。没有 dot-source、compiler、native方法/Go/worker/产品进程执行；此结果只证实语法/模板数据可解析，不是 runner运行、checker或CI验证通过，不能激活CI。原准备收据仍保留本来未执行的历史身份，后续结果不回写它。Root 已全文接收[Task26 显示歧义澄清](<F:/AIproject/LevNote/native/handoffs/autonomous-native-ci-preparation-errata-20261114.md>)（3,450B/SHA256 `6ebaba39642f30608c953be0fdc86ecad6b3e0b0dbdd7c728a1188f77e43eac6`）：源码分隔符码点均为一个U+005C，疑似双反斜杠缺陷来自工具显示转义，未新增prepared-runner或产品缺陷；独立PS5.1纯字面表达式核验exit0不等于PS7 helper或CI运行PASS。Root 仍明确禁止激活：不宣称 runtime validated、ready 或实际 checker/runner 通过。被冻结的九个准备文件不改写。

**新应用交付仍 NO-GO：** 当前新包的五 PE 与许可证/完整 manifest 门、四 worker 的真实请求/失败清理、owned host/Job/正常关闭/会话恢复/第二次启动、物理 Windows 键鼠/IME/剪贴板/选择器与视觉结果、真实资源工作量仍须各自独立证据。Markdown 与 draw.io opt-in smoke 不能冒充 mindmap SVG/scene 两条真实路径或物理 GUI。旧 `.29` 有其原有限接受，但不把旧接受外推到当前源码。

Root 已全文阅读本篇、三份新增说明段及 Task26 澄清，核对实际 diff 后接收有限文档结论。下一步依精确候选清单校核新增与tracked源码，审阅最终暂存差分，再提交并同步 CNB/GitHub main；本篇不提前宣称 Git 已执行。现有权限下包装/格式通道不重跑不绕过；后续平台验收是未完成条件，不取消，也不阻断已经可继续的有限源码收敛工作。
