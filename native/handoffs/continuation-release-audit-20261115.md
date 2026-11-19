# 原生版接续发布审计 · 2026-11-15

基点：`f2c2ceb82acecb40fb0502c047e41ba12a12be47`。本报告检查当前打包代码、产品范围与交付证据；未执行产品测试、构建、打包或桌面操作。

## 当前判断

[最新源码快照:29](<F:/AIproject/LevNote/native/handoffs/autonomous-workspace-snapshot-20261115.md#L29>)记录固定 Windows/Go1.27.3 六根门禁 test/vet/build exit 0，3152 PASS、49 个原 SKIP、0 FAIL；这些是既有证据，本轮未重跑。当前 `.29` 仍是最新接受包。`.33` 在创建 evidence 目录时退出 1，0 PE 构建、0 worker/host 启动，整应用发布仍 NO-GO。

## 必须修代码：一项验收脚本覆盖缺口

**已修复的 P2：四 worker 的包结构原来由三 worker fixture 校验。** 旧资产检查仅建立 Markdown HTML、导图 SVG、只读导图场景三个 worker，原路径断言只要求六个文件。当前[资产fixture:82](<F:/AIproject/LevNote/native/scripts/check-portable-package.ps1#L82>)包含四个引擎，[路径检查:124](<F:/AIproject/LevNote/native/scripts/check-portable-package.ps1#L124>)独立列出八个必需文件。生产[打包定义:34](<F:/AIproject/LevNote/native/scripts/package-markdown-worker.ps1#L34>)和[固定目录表:59](<F:/AIproject/LevNote/native/core/enginecatalog/catalog.go#L59>)均已包含第四个 draw.io worker。

- 触发/影响：现有 fixture 完全没有 draw.io exe/manifest，仍满足三引擎路径断言；该 PASS 只证明旧三引擎资产结构，覆盖不到当前四引擎要求。静态结论，本轮未运行 fixture，也没有证据表明生产打包器实际漏装。
- 修法：补 draw.io 的两个 fixture 文件；独立列出四引擎八个必需路径，以精确集合、总数和唯一数共同判定，发现重复、缺失及额外引擎路径。
- 回归：完整八路径通过；分别移除 draw.io exe/manifest、添加重复/额外路径的纯数据控制均失败；既有目录/Junction/十四项资产/字节摘要检查保持。
- 实施状态：检查脚本已补齐并冻结，SHA256 `dd031915a6022759a2f3f0cf3ed4754683664e7d3f1e92384b759b192a733477`。采用 Ordinal HashSet 与 SetEquals，保留原目录/资产检查。固定 PowerShell 7.7.0 执行实际helper AST及内存数据检查，13项通过、exit 0、stderr空、前后源码SHA相同；[原始回执](<F:/AIproject/LevNote/.localtools/continuation-package-check-20261115/attempt-02-receipt.json>)保存完整结果。首次 PATH 查找 exit 127 已单独保留。完整目录fixture、构建与产品启动均未执行；纯数据结果不证明包已通过平台验收。

生产打包路径未确认新的确定缺陷：[新目录校验:18](<F:/AIproject/LevNote/native/scripts/portable-package-lib.ps1#L18>)先排除既有/嵌套/重定向目录；[打包器:91](<F:/AIproject/LevNote/native/scripts/package-markdown-worker.ps1#L91>)逐个生成四 worker，再构建宿主并校验 amd64 GUI PE，最后复制资产、生成全文件 manifest 与检查本地链接。Go 包装器仍走固定 `native/tools/go` 与 26 项补丁 pin，当前打包前需按现有合同核对该身份，零工具安装。

## 主动范围决定：有限子集，不作为新增回归缺陷

[交付范围:38](<F:/AIproject/LevNote/native/handoffs/autonomous-native-delivery-plan-20261112.md#L38>)要求现有四类文档的支持路径可靠，并未要求无限复制 legacy 能力。保留以下边界，扩充须另立产品目标：

| 领域 | 尚未包含的能力 / 当前决定 | 依据 |
|---|---|---|
| Markdown | 完整 GFM/富文本、Mermaid、公式、图片内嵌与粘贴管理；当前有有限排版、表格与明确图片引用导航 | [便携说明:237](<F:/AIproject/LevNote/native/portable/README.md#L237>)、[当前能力表](<F:/AIproject/LevNote/native/README.md#L404>) |
| 图形 | 导图图片/完整关系线与概括、draw.io 完整 stencil/分组及压缩页图形编辑；保留未知源数据和源码入口 | [当前能力表](<F:/AIproject/LevNote/native/README.md#L405>) |
| 输出 | PNG、PDF、打印尚非当前发布能力；固定导出为 HTML/两类 SVG | [图形 PNG 明确失败:318](<F:/AIproject/LevNote/native/modules/graphview/adapter.go#L318>)、[固定四引擎:38](<F:/AIproject/LevNote/native/core/enginecatalog/catalog.go#L38>) |
| 插件/便携数据 | 任意可执行插件、网络/账号/同步/遥测排除在范围外；移动程序不移动 APPDATA 配置 | [便携说明:10](<F:/AIproject/LevNote/native/portable/README.md#L10>)、[固定包边界:230](<F:/AIproject/LevNote/native/portable/README.md#L230>) |

## 平台验收：尚未完成的发布必要条件

1. **恢复已定位的工具执行身份前置条件。** [Task75:7](<F:/AIproject/LevNote/.localtools/windows-package-blocker-20261115-02.md#L7>)已定位 Medium 父→Low 镜像→Low pwsh 子→Medium 包目录 ADD=false；不是路径参数或 Everyone DELETE_CHILD ACE 的问题。只按[现有最小维护合同:128](<F:/AIproject/LevNote/.localtools/windows-package-blocker-20261115-02.md#L128>)由宿主维护渠道处理，当前任务零标签/ACL/owner/工具链变更、零重复 Low 诊断。后来的只读 token 样本不冒充 `.33` 历史现场。
2. **新候选五 PE 与包完整性。** 在全新版本/证据目录保存宿主与四 worker 的实际 build exit、GUI PE、版本/长度/SHA256、四份引擎 manifest、release manifest、许可证及本地链接；源码/工具输入前后身份守卫。原 `.33` 失败与 `.29` 包保留。
3. **四个真实 worker。** 对精确新包分别验证 HTML、导图 SVG、只读导图场景、draw.io 当前页请求；再覆盖缺失/损坏、取消/超时与所有权 join/Job 清理，留真实进程/退出证据。源码进程内测试不替代这一步。
4. **真实 Windows 宿主。** 新包正常启动、owned window 正常关闭、第二次启动与会话恢复；UTF-8/BOM/只读/草稿/撤销等跨重启语义及失败提示。物理 GUI/IME/剪贴板/picker、最小窗口与 DPI/字体另留真实操作证据。
5. **支持的桌面入口与资源观察。** [Task76:7](<F:/AIproject/LevNote/.localtools/windows-desktop-blocker-20261115-02.md#L7>)确认当前 DSH 官方 CUA 为 macOS/unsupported-platform；旧 Codex native-pipe 失败是另一宿主的历史样本。后续按[入口要求:52](<F:/AIproject/LevNote/.localtools/windows-desktop-blocker-20261115-02.md#L52>)接入受支持的 Windows 工具，再做固定工作量资源测量；headless Tester/隐藏启动不证明物理输入或整进程资源表现。
6. **最终交付与 CI。** 最新状态入口由 lead 更新；GitHub CI 仍停用，准备/parser 结果不等于 runner 运行通过。由 lead 在开发完成后恢复对应流程，并按精确候选完成平台/包验收再取代 `.29`；本轮无 commit/push。

审计方法：先查技能库并加载 code-review；HEAD 等于基点，当前任务按既有范围做现行代码审计，不启动该技能的子代理流程。初始 tracked clean，历史 untracked 保留；唯一报告路径为本文件。
