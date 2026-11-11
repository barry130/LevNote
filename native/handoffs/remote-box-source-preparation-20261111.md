# 远程最终验收输入准备结果 · 2026-11-11

原主会话：`01a1b6b1-3caf-76b1-81e2-183c090b7b86`。
受限实现子代理：`/root/remote_snapshot_preparation`。准备已完成，释放写入归属，交根代理验收。
唯一后续远端为 `main / cb-698f697fed7c`；本阶段没有远程执行或传输。
最后未完成的产品阶段仍是 picker 最终完整 app test/vet，`.30` 尚未启动，`.29` 保持已接受状态。

本轮实际完成静态测试引用审阅、原字节复制、ZIP 归档、Path/Length/Sha256 清单与逐文件核对，以及独立 Linux 守卫包装器草稿。没有执行任何 Go/list/version、test、vet、build、产品、语法检查、包装器、截图、worker 或资源测量；没有下载、上传、凭据访问、提交、推送或分支操作。冻结源码、既有 helper、旧证据和 `.29` 未修改。

## 最终快照与身份

以下文件均位于 `native/handoffs/remote-box-preparation-20261111/`：

| 文件 | 字节 | SHA256 |
|---|---:|---|
| `levnote-source-inputs-20261111-02.zip` | 5,227,577 | `457503DEECBCF0A534CC3240027048454BDF758076E0EE8D3A0A64F7773DB7F9` |
| `inputs-02.json` | 244,715 | `A51734C0E9B40051AD1BA06AD4B5E0ACC4D92EF1A3DE5261259FA435B5DF2E51` |
| `linux-guarded-validation-02.py` | 33,772 | `7D53C6B7374A52C9D3E6749E423449D073778AC1C33C49888A121B97D43E17D7` |

展开副本为 `snapshot-02/`；实际输入 **1,208 文件 / 17,045,848 字节**。完整输入聚合身份是：

`96e0a103b8538f9810333082ed05217e47b1360e963a7dbc8dbef0585654e12d`

聚合算法为按 ordinal Path 排序，将每项 `Path + TAB + 十进制 Length + TAB + 小写 Sha256 + LF` 编成 UTF-8 后求 SHA256；Path 以仓库根为基准、使用正斜杠。它与上表 JSON 文件本身的 SHA256 是两种独立身份。

02 的普通复制/归档/哈希命令 UTC 为 `2026-11-11T02:50:31.1203449Z` 至 `02:50:43.5476690Z`，exit 0。原文件前清单、复制文件、每个 ZIP 成员及原文件后清单完全一致，无 reparse 或路径碰撞。记录见 `snapshot-attempt-02.invocation.json`、分离文本流和 `snapshot-verification-02.json`；这些是 PowerShell 准备文本捕获，不是产品原始进程流。根代理另在 `root-input-review-02.json` 独立核对当前源文件/副本/全部 ZIP 成员，零差异。

冻结依据 `native/handoffs/open-document-picker-key-correction-evidence-20261110/source-frozen.json` 为 748 字节、SHA256 `AB90451D3003E713E6A5260087FA36B0E32DC2B3EDA23DD517F8B0502E9B59F6`；其中五个 app 源码/测试身份逐项一致。补丁清单 SHA256 `9113120BD2EBCD9A86842153239CDDFF84D1F1BD3F46F5D4F0F152EB4560E16A`，其中全部 26 项身份一致，模块/vendor/startup 守卫未改变。

01 全部产物保留。它原为 1,212 文件 / 17,166,862 字节，字节核对正确，但误带四张 Markdown 表格历史输出 PNG。02 仅精确排除该目录的四张截图，共 121,014 字节，未增加或修改任何保留输入。`scope-correction-02.json` 列明差异；01 是范围被替代，不是产品失败。早期 1,305 的估算为算术错误，不是实际清单计数。

## 范围与独立守卫草稿

`scope-review-02.md` 记录实际测试/辅助代码/fixture/embed 引用。快照保守包含完整 native app、cmd、core、formats、modules、ui、vendor、patch、examples、scripts、portable 与许可证树及必要单文件；不声称依赖包的测试会由 `./app` 执行。排除工具链、缓存、运行包、历史非依赖证据、旧 web/Tauri、私有配置，以及所有本次准备和 DSH 预检报告。没有用 Git 提交或重新 vendor 代替当前字节。

`linux-guarded-validation-02.py` 仅经源码阅读审阅，**从未执行或做语法检查**。它固定 Python `-I -B`、Go 1.27.3 Linux amd64、`CGO_ENABLED=0`、离线 vendor 和干净 HOME/XDG/cache/temp，限制命令参数，完整核对 source/toolchain 前后身份，并对 test 和 vet 同样执行五源码、26补丁、startup 与模块守卫。工具链须以官方归档的实际字节校验并逐文件比对提取物；仅有官方清单/HEAD 不够。CGO=0 是单独的 Linux 验证配置，不代表 Windows 或默认 Linux CGO 行为一致。

包装器把 prepare 与待授权 test/vet 分开；没有产品阶段激活 token 和根代理审核过的真实 guard-ready 身份，不运行产品。后续命令仅允许 `test -mod=vendor ./app -count=1` 和 `vet -mod=vendor ./app`，失败后不自动重试或改测试，test 失败时 vet 保持等待审阅。原始二进制 stdout/stderr、UTC argv/cwd/env、真实 child exit/signal、清理错误及 Attempted/Spawned 区别由草稿保留。

DSH 的独立预检已观察到 Ubuntu 24.04.5、DISPLAY=:0、共享库和字体；报告由它独占维护，未进入输入快照。本子代理无 box/原字节传输工具，也没有实际下载 Go、准备远端源树或验证 MyGo 后端。草稿中的库/字体哈希及 ctypes font-map 检查只是单独进程前提，不是 MyGo 渲染/IME 证明，也不是整个 Linux 环境的封闭身份。非 verbose 成功 test 可能隐藏日志，缺少 fallback 日志不能证明有效文本后端。

现有 Linux 风险保留：`app/reference_test.go` 的 C: 路径期望、reopen 测试对真实临时路径转大写后重开，以及四个 Windows-tagged app 测试文件的平台选择。没有运行 Go 验证真实选中文件，也没有新增 skip 来适应 Linux。

## 明确下一步

唯一下一阶段是由 DSH 提供或用户交付的授权私有原字节通道，将上述三份传输文件送往指定 box 的全新 inbox，核对长度/哈希；再在远端取得官方 Go 归档，并独立激活 **迁移守卫 prepare**。精确命令、路径约束、外层流捕获和产物审核要求见 `remote-runbook-02.md`，传输身份见 `transfer-files-02.json`。当前缺口是安全传输通道和真实远程守卫结果；不得回退本机、读取令牌或开启公开上传服务。根代理审阅真实守卫结果后，才决定激活 Linux 最终 app test/vet。

Windows `go.ps1`/bootstrap 及 `.30` helper 依赖 Windows go.exe、GUI PE/五次构建、C: 配置、WM_CLOSE 与 Job 清理。Linux 测试、交叉编译或 Wine 均不能代替该交付验收；`.30` 保持等待，不改变旧 Windows 证据。

准备顶层文件的闭合身份清单为 `preparation-artifacts-02.json`，8,097 字节，SHA256 `A05C5E9FB28F0E9B59A0ED1729EAB5031171A444D23FD6CF436953721387041C`。它覆盖生成时全部顶层准备文件，排除自身；展开目录由各输入清单与归档核对覆盖，本结果位于目录外并单独计算身份。收尾更新 progress 的 apply_patch 两次返回写入失败，随后使用允许范围内的 UTF-8 文本写入成功；未触及快照、源码或旧记录。