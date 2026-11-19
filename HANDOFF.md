# LevNote · Harness 接手入口

更新：2026-11-19。用户最新指令：**先不要开发，把所有东西整理、提交并上传，交给其他 Harness 接手。** 本提交是包含未完成工作的保存点，不是新的已验收发行版。三个实现子代理均已停止，写入权已释放。旧计划中的 ACTIVE、自动继续与历史负责人名称不自动恢复。

## 先看这几项

1. [当前原生状态](native/STATUS.md)：现有能力、体验包与正式交付缺口。
2. [原生协作约束](native/AGENTS.md)、[原生功能说明](native/README.md)、[验证记录](native/VALIDATION.md)。其中早期日期段落是历史记录；以本交接的暂停状态为准。
3. 本页“暂停时的三个任务”及各代理交接，确认尚未实现和未验证部分，再按用户下一次指令推进。
4. [本次证据索引](native/handoffs/harness-transfer-20261119/README.md)：保存本轮开发前快照、真实失败与此前预览的验证依据。

## 主线与不可变约束

- 主线为 Windows 上的 Go 1.27.3 + MyGo 原生 Surface，自研文本、Markdown、思维导图与 draw.io 内核。根目录 Tauri 2/React 实现保留为历史参考，不作为当前开发主线。
- 纯本地，无账号、同步或遥测。四个固定离线引擎配非执行 JSON 清单；不接任意可执行插件。内核不直接访问文件系统，通过中性宿主契约交互。
- 保存未改字节、未知字段、编码/BOM/EOL、只读、撤销历史和草稿；禁止静默覆盖、丢弃、转码或降级。真实失败与已接受产物不可覆盖。
- NoteBoard 仅借鉴功能和工作流程；上游 AGPL-3.0-only，本仓库 MIT，本轮没有复制其实现。
- 若后续委派：最多三个并行子代理、禁止孙代理、先划清互不重叠的写域。上轮使用星驰 GPT-6.1 Sol，lead 负责分工、审查与集成；本轮停止后没有活动实现代理。
- ≤10 个工程文件逐文件 read/edit/write；不要用 Python 或批量脚本改工程源码。旧包、旧证据和已消费的运行目录保持原样。

## 版本与当前快照

交接前提交基点：`f2c2ceb82acecb40fb0502c047e41ba12a12be47`。本次把该基点之后累计的源码、测试、文档、交接材料和七个 Windows ZIP 一并保存。相对基点的全部差异包含多天既有工作；不能把它们全部归为 11 月 19 日新实现。

| 对象 | 状态与依据 |
|---|---|
| 正式接受包 `native-worker.20261110.29` | 仍为最新正式接受版本；[接受记录](native/handoffs/composition-release29-root-acceptance.md)、[Windows ZIP](native/downloads/LevNote-native-worker-20261110.29-windows-x64.zip) |
| 最新体验包 `native-flowix.20261116.6-preview` | [Windows ZIP](native/downloads/LevNote-Flowix-preview-20261116.6-windows-x64.zip)、[布局验收](native/handoffs/flowix-chrome-root-acceptance-20261116.md)；不是当前未验收克隆修改的构建 |
| 当前源码 | 保留全部既有工作与 draw.io 最新修改；尚未完成最新源码集成检查，不宣称编译或测试通过 |

最新体验 ZIP：11,746,501 字节，SHA256 `10d0cb6ea220831fd77a4a7e1d556a05db4ee7faf01a8e7491690386adc05171`。其历史六根验证为 3577 PASS / 49 原有 SKIP / 0 FAIL，vet exit 0，436 项输入不变，3424 个旧测试身份保留，11 张测试器截图。该结果不覆盖之后的克隆/批删工作，也不代替物理输入、IME、剪贴板、进程生命周期或整应用资源验收。

## 暂停时的三个任务

| 任务 | 实际停点 | 接续依据 |
|---|---|---|
| Markdown 嵌套列表 | 没有修改源码或测试，没有运行测试；仍未实现 | [列表交接](native/handoffs/noteboard-parity-md-lists-20261119.md) |
| Markdown 表格单元格草稿恢复 | 完成契约方案，未写源码或测试、未运行验证；现有未提交单元格草稿仍不参与崩溃恢复 | [草稿交接](native/handoffs/noteboard-parity-md-drafts-20261119.md) |
| draw.io 克隆/批删收尾 | 三个文件有新修改；末次测试后继续修改，最新状态未重新编译、格式化、测试或 vet；六项 P2 未宣告关闭 | [draw.io 交接](native/handoffs/noteboard-parity-drawio-finish-20261119.md)、[既有设计](native/handoffs/drawio-clone-design-20261116.md) |

本轮真实源码增量限于：

- [drawio_clone.go](native/modules/graphview/drawio_clone.go)：去除两处调试打印，修正连续撤销/重做时对同步前 owner/selection 与 before/after 文本的检查方向。
- [drawio_clone_correction20261117_test.go](native/modules/graphview/drawio_clone_correction20261117_test.go)：调整 Unicode fixture，让固定 UTF-16 光标在插入后落到 emoji 内部，保留完整历史、选择、反馈与撤销重做断言。
- [clone_cells.go](native/formats/drawio/clone_cells.go)：单调 ID 分配；先累计原始跨度与标签片段并做 int64 总预算检查，再组装最终输出；标签改写仍有分配。

这三处修改仅做停点核对，未作为功能完成接受。11 月 19 日前快照见[开发前清单](.localtools/noteboard-parity-20261119/baseline/manifest.json)。图形测试最近一次实际执行仍有两个失败：`TestDrawioMultiCloneClampCompletesSelectionHistoryAndFeedback`、`TestDrawioMultiCloneContinuousHistoryRespectsOwnerAndSelectionEpoch`；另一次更早执行为调试代码编译失败，两次 exit 均为 1，原始日志保留。最新更改能否修复它们仍待验证。

已确认的草稿恢复设计：中性可选 `CaptureSessionDraft(maxBytes) (draft, ok, err)`；`Host.DraftRecoveryChanged func(owner View)` 由 app 检查标签仍打开且 `t.View == owner` 后触发恢复调度；UI 线程 900ms 安静期保存，不 Flush/Unmark IME，只捕获已确认内容；恢复为无路径 `.txt` 副本，原路径只进 `OriginalPath`；单草稿 4MiB、总原始文本及精确 JSON 均 16MiB、最多 100 标签。早期提及的 32MiB 是错误值，不能采用。

## 环境与验证入口

原工作区：`F:/AIproject/LevNote`。证据原有绝对路径保持原文；迁移后按仓库相对路径查找。Go 工具链、缓存、临时目录不入库，新机器准备 Go 1.27.3；vendored 依赖在仓库中。

[原门禁脚本](.localtools/gate.sh)已作为交接证据纳入版本控制，脚本仍固定本机路径，不能在新机器未经审阅直接执行。原检查环境如下；这些是后续复现说明，本次整理没有运行新产品测试、构建或打包：

```bash
cd /f/AIproject/LevNote/native
export GOENV=off GOWORK=off GOFLAGS=-mod=vendor GOPROXY=off GOSUMDB=off
export GOVCS='*:off' GOTOOLCHAIN=local GOTELEMETRY=off CGO_ENABLED=0
export GOCACHE=F:/AIproject/LevNote/.localtools/cache
export GOTMPDIR=F:/AIproject/LevNote/.localtools/tmp
../.localtools/go1.27.3/bin/go.exe test -count=1 -timeout=300s ./app/ ./cmd/... ./core/... ./formats/... ./modules/... ./ui/...
../.localtools/go1.27.3/bin/go.exe vet ./app/ ./cmd/... ./core/... ./formats/... ./modules/... ./ui/...
../.localtools/go1.27.3/bin/go.exe build ./app/ ./cmd/... ./core/... ./formats/... ./modules/... ./ui/...
```

始终明确这六个产品根，不运行 `go test ./...` 将 handoff 中的历史源码快照纳入。后续若获继续开发指令，第一步是阅读 draw.io 交接与本轮前后差异，在新证据目录运行相关定向回归，再决定修正；不要恢复已消费的旧运行脚本。

Windows 正式发行仍缺新包真实 worker 生命周期、正常退出和再次启动恢复、物理键鼠/中文 IME/剪贴板/选择器/显示缩放、固定工作负载资源测量。GitHub CI 暂停状态保持。

## 保存范围与远端

目标分支 `main`，CNB `origin`：https://cnb.cool/canace/LevNote.git；GitHub `github`：https://github.com/barry130/LevNote.git。不改写历史、不强推。实际交接提交可由 `git log -1 --format=%H` 获取；最终推送回读结果由本次交付答复记录。

源码、测试、文档、已有 handoff 证据、七个下载 ZIP 与本次明确选中的本地证据入库。通过显式纳入 Git 保留证据原路径和字节；缓存、工具链、临时运行目录仍按原忽略规则保留本地，不整体上传。

根目录 [pass.log](pass.log) 名称误导，实际内容是历史 draw.io 格式测试 FAIL；保留原件，不当成通过证明。根目录 [CON](CON) 是已有 Windows 保留名称条目，文件操作工具以 Reserved/ambiguous/device path 拒绝检查；内容未核验，未删除或改名，单独保留本地，不纳入跨平台仓库。除该异常条目与忽略目录外，所有待提交项目工作均在本次保存范围。
