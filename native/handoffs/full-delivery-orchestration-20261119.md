# LevNote 完整交付调度与质量门禁 · 2026-11-19

> **当前接续，2026-11-19 21:45（Asia/Shanghai）**：用户要求接续 `session-8d654edc-04ab-4648-b5ce-9cdaf7a45d5f`。已读取原会话直接指令：只使用星驰 GPT-6.1 Sol，不允许其他子代理模型，子代理数量不限；lead 负责安排任务与最终质量检查，不实现产品功能。下文旧“三个代理上限”、备用模型和授权阻断均为历史记录，不再是当前规则。新会话已成功查询 `magpie-responses / xingchiapi/gpt-6.1-sol`（high 可用），当前无遗留代理或任务，已建立新的持久目标。具体新写域与任务见[本次接续调度](full-delivery-continuation-20261119.md)。未变的产品边界与完整交付判据继续适用。

## 本轮授权与负责人

用户明确要求继续全部后续开发，由 lead 负责子智能体调度和最终质量审阅，**lead 不实现产品功能**。不使用 lead 当前模型作子智能体，也不使用此前明确禁止的 anyrouter/gpt-6.1-sol；优先星驰 GPT-6.1 Sol。模型持续服务报错时更换可用且符合用户约束的模型。本轮主模型已发生切换，不能把换主模型视为解除同模型限制；后续始终以实际会话与用户最新授权核对。旧交接暂停说明描述历史停点，不阻止本轮新授权；旧代理、旧任务脚本和旧运行目录不自动恢复。

当前源码提交：`2ebec3d8ef47338f4efa2f913cc64015f61253d6`。当前工作区 `/data/workspace/harness/home/LevNote` 为 Linux amd64；开发工具见 [工作区说明](../../DEVELOPMENT.md)。正式原生主线仍为 Go 1.27.3 + MyGo Surface；旧 Tauri 路径为历史参考，不能用其功能或构建冒充原生交付。

## 当前模型调度限制

- 已核对准确注册标识：provider `magpie-responses`，model `xingchiapi/gpt-6.1-sol`。
- 当前会话模型工具拒绝星驰上述子代理路由，也拒绝 `auto/gpt-6.1-sol` 和另一已配置 provider `magpie`。
- 早先只列出 anyrouter 路由；本轮主模型切换后重新无参数列 provider，再列其模型，实际仅显示 `magpie-responses / api666-zzzcoding/gpt-6-astra`，与 lead 同模型。星驰精确路由仍返回 `not allowed for this Session`，没有可派发的合规模型。以上是当前工具实测，不能沿用旧名单描述现态。
- 因此本轮尚未启动任何实现代理，以下分工为**待派发计划，不赋予任何历史代理写入权**。lead 不修改配置白名单绕过限制、不偷用默认模型。
- 用户随后说明已修改设置。复核 Harness 实现后确认：授权设置只在**全新顶层会话**首次创建 Agent 时采样；当前会话已有的授权事件不可被新配置替换。恢复/重启当前会话或重启 Harness 服务不会刷新旧授权，不应继续反复尝试同一被拒绝路由。依据：[授权采样实现](</data/workspace/harness/app/lib/node_modules/@deepseek-ai/dsh/node_modules/@deepseek-ai/dsh-tool-subagent/lib/model-selection-settings.js#L83-L100>)、[仅首次事件生效的投影](</data/workspace/harness/app/lib/node_modules/@deepseek-ai/dsh/node_modules/@deepseek-ai/dsh-tool-subagent/lib/model-selection-settings.js#L44-L58>)、[设置页“仅影响新会话”文案](</data/workspace/harness/app/lib/node_modules/@deepseek-ai/dsh/node_modules/@deepseek-ai/dsh-client-ui-settings-subagent/lib/client.js#L74-L76>)。
- 接续方法：保存已开启且含星驰精确路由的子智能体设置，在**同一工作目录新建会话**，不是刷新/恢复旧会话或继承其授权快照的分叉。新会话先查询星驰精确 provider/model；实际查询成功后才能派发。可同时授权另一个非 anyrouter 模型作为服务失败时的备用。
- 空字符串不是未传参数，模型目录查询不要使用 `provider:""` 或 `model:""`；需要列提供方时真正省略参数，或直接查询已核对的精确路由。
- 当前会话的 task-1/2/3 仍待派发；task-4 的只读基线审阅已形成[报告](full-delivery-baseline-review-20261119.md)，测试身份和失败证据已固定。新会话不应假定沿用同一任务板或 goal；先检查实际状态，仅重建不存在的任务。
- 持久目标实测为 paused；本次收到用户“继续”后调用 resume，工具拒绝并要求用户自行恢复。人工请求下的审阅已继续，但自动续轮需要在 Harness 恢复该目标，不能新建另一目标绕过暂停。
- 用户在随后确认中明确选择“新会话启用星驰”，没有选择允许当前千刀 Astra 子代理。接续采用同工作目录全新会话，不再在旧会话试探其它路由。等待答复期间运行时模型显示又发生变化，因此上述模型名单是该次查询的记录，不应推断为下一会话当前能力；新会话仍须实际查询星驰精确路由。
- 本轮已完成 task-4 并验证442个产品输入及原基线日志保持不变。新会话应先读本计划和[基线审阅报告](full-delivery-baseline-review-20261119.md)，检查任务板后派发 task-1/2/3；不要重复已完成的日志解析或把 test-list 当行为回归。

## 交付范围与完成判据

所有公开能力、未完成任务和需求差距进入可追踪清单；不得仅修少数回归就把当前有限子集命名为完整项目。

| 工作流 | 必须完成的核对与交付 |
| --- | --- |
| 文档可靠性 | 打开/保存/另存、冲突、只读、编码/BOM/EOL、原字节与未知字段、撤销重做、错误可见性和源码/字节兜底 |
| 工作区与宿主 | 多标签、文件树/大纲、查找替换与工作区搜索、命令/快捷键、恢复调度、正常关闭及二次启动、不丢未提交编辑 |
| Markdown | 先修表格空单元格失败/panic；接续嵌套列表与任务操作；完成表格独立草稿恢复；核对图片/资源、Mermaid、富文本操作等原生差距 |
| 思维导图 | 先修现有标记身份失败；核对字段、标签、颜色、只读与撤销；登记并补齐图片、关系线/概括、排序和字段搜索差距 |
| draw.io | 独立审阅克隆/批删既有 P2 与预算；实测多轮撤销/选区/owner；核对分组、复杂形状、图片、stencil 等剩余功能，不静默变更格式 |
| 代码与文本 | 保存/选区/Unicode/IME、查找/行命令、性能与原生高亮；核对折叠、多光标和语言服务的需求及实现路径 |
| 导出与本地引擎 | 四个固定离线引擎的真实随包请求、拒绝/失败/取消与清理；HTML/SVG 和尚缺 PNG/PDF/打印；纯本地非执行 JSON 配置 |
| 可用性与发行 | 国际化、无障碍、窄窗/缩放/键盘导航、文档/许可、安装包可搬移、干净机器启动与恢复、资源固定负载、CI 实际运行 |
| 平台 | Windows amd64 为现有正式目标，实机验收必做；其他平台差距明确登记，不将 Linux/交叉编译或 Wine 当 Windows 系统输入/Job 验收 |

更大功能项先做需求/架构核对与实现分片，再逐项验收；不得未经用户明确调整就从完整交付清单剔除。若某项并非当前项目需求，应记录对应需求依据，而不是为赶工自行降范围。

## 第一批待派发任务与不重叠写域

先诊断实际失败、回报根因和精确文件清单，再授权小批实现。每批涉及不超过 10 个工程文件时逐文件 read/edit/write，禁止 Python/批量脚本改源码。最多三个直接子代理，禁止孙代理。

1. **Markdown 可靠性与列表代理**：诊断 `native/modules/markdownview` 的空单元格断言和 nil pointer；读取 [列表交接](noteboard-parity-md-lists-20261119.md)、[草稿交接](noteboard-parity-md-drafts-20261119.md)。候选写域仅 `native/formats/markdown/`、`native/modules/markdownview/` 中派发时列明文件。先修现有崩溃，再列表与模块侧草稿捕获；不写 app/core/ui 公共契约，不扩大第三方内核。
2. **宿主与恢复代理**：诊断 `native/app` 超时与失败、`native/core/session` 路径问题、`native/core/plugins` symlink 问题，区分真实产品缺陷和有证据的平台夹具问题。候选写域为这些包中的明确文件，以及派发时列明的 `native/ui/editor/contract.go` 和新增中性草稿契约文件。草稿宿主接线必须先与 Markdown 代理冻结契约；不写 markdownview、graphview 或 ui/editor/editor.go（需要时重新授权）。
3. **图形可靠性代理**：诊断 graphview 标记身份失败，独立复核 [draw.io 交接](noteboard-parity-drawio-finish-20261119.md) 和未关闭 P2。候选写域仅 `native/modules/graphview/`、`native/formats/drawio/`、必要的 `native/formats/mindmap/` 指定文件及明确划给它的专用 app 图形测试；不得写 app 通用文件、Markdown 或公共契约。

跨域需求先交 lead，未经新任务不得抢写。lead 独占调度计划、公共状态/验收文档和最终集成证据。发布脚本、CI 和新包作为源码收敛后的独立任务另行授权。

## 表格草稿契约的既有安全边界

采用可选中性捕获接口 `CaptureSessionDraft(maxBytes) (draft, ok, err)` 与 `Host.DraftRecoveryChanged func(owner View)`；宿主核对标签仍打开、`t.View == owner`。900ms 安静期在 UI 线程采集，不 Flush/Unmark IME、不改焦点/主文档/选区/撤销。组合输入期间只记录上次确认态，明确放弃/释放不留旧副本。恢复成无路径 `.txt` 独立标签，原路径仅为 `OriginalPath`。单份 4MiB，原始文本合计及精确 JSON 分别 16MiB，最多100标签；超限/失败可见并保留此前有效快照。不得采用历史错误32MiB值。

## 测试与证据

环境准备原始基线保持不变：[汇总](../../.localtools/setup-20261119-linux/baseline-summary.json)。本机六根测试命令 exit1，2349 PASS/103 FAIL/15 SKIP（含子测试），有2个未终态测试、5个失败包。新[基线审阅](full-delivery-baseline-review-20261119.md)确认：Markdown panic 落在测试对 nil draft 的解引用，之前已有空单元格交互失败；app 300s 是包预算耗尽，报警栈仍在等待 fake exporter，尚未进入关闭分支。测试名称列举进一步确认 **426 个 Linux 顶层测试未执行（app 341、markdownview 85）**；已固定442个产品输入哈希和测试身份。新 list exit0只证明列举成功，没有新增行为 PASS。Linux build/vet、五目标 Windows 交叉编译、45项 clone 定向回归及四个独立 Linux worker 冒烟通过，**不等于最终产品接受**。

每个代理使用全新 `.localtools/full-delivery-20261119/<task>/<run-id>/` 证据目录，保存实际命令、输入身份、stdout/stderr、退出码与时间。原失败不覆盖；测试失败不是模型服务失败，不能用换模型/新增 skip/削弱断言伪造通过。只能通过项目内工具：

```bash
source .localtools/env.sh
bash .localtools/native-go.sh test -count=1 -timeout=300s <明确受影响产品包> -run <相关过滤器>
# lead 在源码冻结和独立复核后统一最终六根检查：
bash .localtools/native-go.sh test -json -count=1 -timeout=300s ./app/ ./cmd/... ./core/... ./formats/... ./modules/... ./ui/...
bash .localtools/native-go.sh vet
bash .localtools/native-go.sh build
```

不要运行 `./...` 把 handoff 历史源码纳入；不要动 vendor、旧工具链/证据、`.29` 接受包或现有体验ZIP。平台特定断言只在查明预期语义后做保留原覆盖的跨平台夹具补充，不以 Linux 绿为由改 Windows 合同。

## 独立质量审阅与模型替换

实现者完成后冻结写域，交付精确文件清单、行为/边界、失败与通过原始证据、未完成项。另一个非实现者做数据安全/并发/预算/结构审阅；lead 亲自检查完整 diff 和未跟踪文件，验收用户行为与需求覆盖，并集中运行集成门禁。实现者自报 PASS 不能自动获接受。

同模型连续两次服务错误且没有有效工作进展时不无限重试：保存错误与最近checkpoint，停止旧代理并确认已释放写域，再用已许可的非当前模型启动替代者，只接续明确任务。若所有合规路由都不可用，记录具体模型许可/服务条件并请求调整会话能力，不由 lead 代写产品功能。所有后台job及时收集，不留无人管理进程。

## 最终接受必须同时满足

- 需求清单逐项实现或有用户明确认可的范围决策，安全/恢复/撤销/错误出口无阻断。
- 原有测试身份与覆盖保留，最终目标平台受影响完整回归、vet/build通过；skip和平台例外明确解释，panic/timeout不能当通过。
- 新发行包具有完整宿主+四worker、相对清单/哈希/许可/说明；不覆盖旧接受资产，不把裸交叉编译exe当可发行包。
- Windows 新包真实四类请求及失败/取消/Job清理、正常关闭、再次启动和恢复通过；键鼠、中文IME、剪贴板、选择器、窄窗/缩放与固定资源负载有相应真实证据。
- 安装/运行/开发/故障恢复文档与实际产物一致，CI在最终收敛后按授权启用并记录运行；最终交付包括源码、发行包、验证报告和剩余风险（不得隐去未完成工作）。

当前进度：lead 已完成调度/验收拆分、失败证据分类及测试身份冻结，已形成[基线审阅报告](full-delivery-baseline-review-20261119.md)并据此细化三个待派发任务；**尚无已派发开发代理，尚未开始本轮产品实现或新增行为回归**。本轮查询未取得可用的星驰授权，用户已明确选择在新会话启用星驰后接续；后续以新会话的实际许可结果派发，不重复旧会话的拒绝探测。
