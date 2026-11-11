已在**原会话 `01a1b6b1-3caf-76b1-81e2-183c090b7b86`**继续，ID 来自当前环境的 `CODEX_THREAD_ID`。本轮仅进行了本地文件读取、哈希核对和工具发现，没有写入文件、执行产品操作或启动子代理。以下内容可由 `-o` 保存为接续凭据。

**最后未完成阶段已确认：picker 最终应用测试与 vet。**

[修正审阅](F:/AIproject/LevNote/native/handoffs/open-document-picker-key-correction-review.md)已经落盘，结论为“没有剩余具体阻塞”，审阅者已释放任务。`native/AGENTS.md` 顶部仍记着等待审阅，属于尚未同步的阶段记录。

[最终验证任务](F:/AIproject/LevNote/native/handoffs/open-document-picker-final-validation-task.md)仍为 PLANNED；最终验证结果、源码接受记录和 `.30` 交付报告均不存在。因此：

- 当前接受交付仍为 `.29`。
- `.30` 仅完成交付准备，尚未构建或验收。
- 三个列表控件引用的后续优化继续排队，当前不进入实现。

当前权威冻结清单是 [key-correction/source-frozen.json](F:/AIproject/LevNote/native/handoffs/open-document-picker-key-correction-evidence-20261110/source-frozen.json)，748 字节，SHA-256：

`AB90451D3003E713E6A5260087FA36B0E32DC2B3EDA23DD517F8B0502E9B59F6`

本轮重新读取并核对，五个实际文件全部匹配：

| `native/` 下路径 | 字节数 | SHA-256 |
|---|---:|---|
| `app/open_documents.go` | 13882 | `BF1789ACC0A9B885F358CD46F62251C5E2971A6DD351AC5209A3C82BC15B7DA6` |
| `app/open_documents_test.go` | 30122 | `A70444047DED384477F2DFDE2E83FE98AC53F06304633F4F65304A19CCC5444C` |
| `app/app.go` | 39003 | `FAD0EDE4C480E0C8F5C56F4E81CF2499BEF0007BA6EAE877FC516623BFFEA050` |
| `app/view.go` | 23425 | `64DCDC194510C6426112A7B6AAFE57A84DDE68E630A20D53C994D77CC34102C0` |
| `app/workspace.go` | 6739 | `7A9ACA4F8E807D7A0563B5595976839AA7DED706EFDC5171B259079D68A16DA4` |

此前记录的 13 项行为测试通过属于 **2026-11-10 的既有证据**，本轮没有重跑。

**下一步唯一阶段：将 picker 最终应用测试与 vet 迁至远程 box，先完成执行前守卫。**

目标固定为用户指定的 `main / cb-698f697fed7c`，Ubuntu 24.04、Linux amd64、`/home/user`。本轮没有可调用的 box/DSH 远程执行或文件传输工具，未独立探测远端。需要接入按 box ID 定向的执行、原字节文件传输及结果读取能力；不回退 Windows 本机。

最小迁移方案如下：

1. **传输实际工作区输入。**  
   在远端新建独立运行目录，传入完整所需源码、测试、`vendor/`、`go.mod`、`go.sum`、补丁、测试引用的 fixture/helper 和冻结交接记录。不能仅凭 Git 提交或重新 clone 代替当前工作区。传输保持原始字节及行尾；生成覆盖全部实际输入的路径、长度、SHA-256 清单，远端逐项核对。上述五文件清单只是变更身份，不能代替完整构建输入清单。

2. **固定 Linux Go 1.27.3。**  
   使用经官方发布清单 SHA-256 验证的 Linux amd64 工具链，记录归档与实际工具链身份。版本不存在或校验失败即停止，不能降级或让 `GOTOOLCHAIN` 自动换版。工具链、缓存、临时目录和遥测状态均放在远端本次运行目录。

3. **保留 vendor 与补丁守卫。**  
   新远端执行包装器应独立于冻结源码，等价保留：
   - `GOTOOLCHAIN=local`、`GOWORK=off`、`GOENV=off`，明确 `-mod=vendor`。
   - 禁止通过 `-C`、`-modfile`、`-overlay` 或非 vendor 模式替换输入；不执行 `tidy`、`mod vendor` 或依赖升级。
   - 固定 MyGo `v0.8.3`，检查 startup 标记及禁止恢复 `go cleanUpdateLeftovers(...)` 调用。
   - 核对补丁清单自身及全部 **26 项**文件。当前清单 SHA-256 为  
     `9113120BD2EBCD9A86842153239CDDFF84D1F1BD3F46F5D4F0F152EB4560E16A`。  
     本轮本地只读核对结果为 **26 项、0 不匹配**。
   - 对 `vet` 同样应用守卫；现有 Windows 包装器的 guarded-command 列表没有包含 `vet`。

4. **先检查 Linux 实际依赖和测试范围。**  
   MyGo 的 Linux 文本路径使用 Pango/Cairo/GLib 及字体环境；缺库时源码存在退回 stub 的路径，不能把该状态算作有效 UI 验证。远端需记录库、字体和实际平台选择。现有应用测试还包含 Windows 路径预期、Windows build tags；固定引擎目录目前也只登记 Windows amd64。遇到这些平台差异，应如实报告，不能删除断言、添加跳过或改冻结源码来获得通过。

5. **守卫通过并进入执行阶段后，只做一次目标验证。**  
   经远端包装器执行 Linux 原生的 `go test -mod=vendor ./app -count=1` 和 `go vet -mod=vendor ./app`。关闭截图、worker 和资源测量 opt-in；分别保存命令、UTC 时间、原始 stdout/stderr、退出码、平台选中文件以及输入前后身份。证据写入新的远端 `20261111-linux` 目录，不覆盖旧 Windows 证据。失败保留原样，先分析再决定后续。

旧 [go.ps1](F:/AIproject/LevNote/native/scripts/go.ps1)硬编码 `tools\go\bin\go.exe`，bootstrap 下载的也是 Windows ZIP，不能直接用于 Ubuntu。[`.30` 交付任务](F:/AIproject/LevNote/native/handoffs/open-document-picker-release30-delivery-task.md)还要求 Windows GUI PE、五次 Windows 构建、C: 隔离配置，以及 `user32/kernel32`、`WM_CLOSE`、Job Object 的真实宿主生命周期。**Linux 测试、交叉编译或 Wine 结果都不能替代这份 Windows 交付验收。** 当前唯一 box 为 Linux，故 Windows `.30` 交付继续待定。

交给 DSH 的下一次请求应限定为：

> 保持原会话 `01a1b6b1-3caf-76b1-81e2-183c090b7b86`，仅连接 `main / cb-698f697fed7c`。先只读返回实际 box/OS/架构、工作目录、Go 1.27.3 可用性、Linux 文本库与字体、现有远端工作区及输入清单状态。不得测试、vet、构建、启动产品或修改源码。随后按上述冻结身份制定原字节传输和独立守卫包装器的执行清单；所有后续产品操作只能发生在该 box。本机始终只读，旧证据保持原样。