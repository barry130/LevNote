# LevNote 原生主线当前状态

更新：2026-11-19。**用户已要求停止开发，整理并提交上传现有工作，交由其他 Harness 接手。** 主入口：[仓库交接](../HANDOFF.md)。三个实现子代理已停止并释放写入权；当前工作快照包含尚未验收的 draw.io 增量，不代表新发行版。开发主线为 Go 1.27.3 + MyGo 原生 Surface；旧 Tauri/React 路径保留为历史实现。

## 当前结论

- 当前源码基点：`f2c2ceb82acecb40fb0502c047e41ba12a12be47`。成熟功能首批已完成命令面板、行移动/复制、Markdown行内格式、导图颜色及只读显示一致性；最终3375 PASS/49原SKIP/0 FAIL，vet与五GUI构建exit0。前版全部3264测试身份保留并新增160，426测试输入前后同。详见[本批验收](<F:/AIproject/LevNote/native/handoffs/feature-parity-root-acceptance-20261115.md>)。
- 现代工作台UI已完成：深浅分层背景、圆角标签、真实文档搜索、分段模式切换、舒展欢迎页与灰阶状态。最终完整回归3215 PASS/49原SKIP/0 FAIL，vet及五GUI构建退出0，前版全部测试身份保留。详见[现代UI验收](handoffs/modern-ui-root-acceptance-20261115.md)。此前[VS Code版验收](handoffs/vscode-ui-root-acceptance-20261115.md)与[3183项验收](handoffs/continuation-root-acceptance-20261115.md)保留。
- flowix 风格工作台已升级为最新体验版：浅色冷白纸面配紫色品牌锚点、深色冷暗画布配金黄品牌锚点、8px 圆角，顶栏重构为 48px 紧凑动作区，左下四类新建入口改为左对齐列表。六根门禁3577 PASS/49原SKIP/0 FAIL、vet exit0、身份审计零退化。最新[Windows x64 工作台布局体验ZIP](<F:/AIproject/LevNote/native/downloads/LevNote-Flowix-preview-20261116.6-windows-x64.zip>)版本native-flowix.20261116.6-preview，11,746,501字节；五GUI PE、26文件、八引擎路径及全部清单哈希通过独立核验。详见[布局验收](handoffs/flowix-chrome-root-acceptance-20261116.md)；上一[主题体验ZIP](<F:/AIproject/LevNote/native/downloads/LevNote-Flowix-preview-20261116.5-windows-x64.zip>)及[主题验收](handoffs/flowix-restyle-root-acceptance-20261116.md)保留。
- 最新正式接受包仍为 `native-worker.20261110.29`，见[包接受记录](handoffs/composition-release29-root-acceptance.md)。最新[Windows x64 draw.io多选体验ZIP](<F:/AIproject/LevNote/native/downloads/LevNote-Drawio-preview-20261115.4-windows-x64.zip>)版本native-drawio.20261115.4-preview，11,743,909字节；五GUI PE、26文件、八引擎路径及全部清单哈希通过独立核验。此前[功能增强体验ZIP](<F:/AIproject/LevNote/native/downloads/LevNote-Features-preview-20261115.3-windows-x64.zip>)保留。此前[现代UI体验ZIP](<F:/AIproject/LevNote/native/downloads/LevNote-Modern-UI-preview-20261115.2-windows-x64.zip>)保留；完整桌面发行验收仍待完成。
- GitHub CI 保持停用，待开发与平台交付验收完成后再启用。

## 已有能力

四类文档共享多标签、源码编辑、保存/另存、查找替换、工作区搜索、问题记录和有预算的会话恢复。原生格式视图提供 Markdown 块/表格编辑、导图节点/标记/标签编辑，以及 draw.io 的有限形状/连线编辑，含 Ctrl 多选、六向对齐、Shift 框选与整组移动。四个固定本地引擎承担 HTML、SVG 导出和只读导图场景；PNG/JPEG 可只读查看。完整能力与边界见[功能表](README.md#已有能力与边界)。

最近完成的源码修复包括图形草稿恢复的输入所有权、恢复失败的单次报告，以及工作区搜索结果与实际搜索文本快照的绑定。旧搜索结果遇到保存后变动或未打开文件的内容变化时，会提示重新搜索，保留当前选择和文档状态。

## 本轮修复与验证

| 修复 | 用户可见变化 | 当前证据 |
|---|---|---|
| Markdown 闭合代码块写回 | 替换正文时保留关闭围栏的独立行和原 LF/CRLF，后续段落保持正常；空正文与未闭合块保留其语义 | 真实失败后同断言通过；含真实Enter的19项公开交互与失败保留回归通过，见[围栏记录](handoffs/continuation-format-audit-20261115.md) |
| 重载后的会话恢复 | 已明确放弃的草稿在成功重载后更新恢复快照；失败、取消及确认期间的文档变化另作保留检查 | 两个入口已各完成真实失败与修复验证，12项成功/失败/确认变化回归通过；见[重载记录](handoffs/continuation-app-audit-20261115.md) |
| 四个固定引擎的包检查 | 资产 fixture 补齐 draw.io 可执行文件和 manifest；八路径必须完整、唯一且没有额外成员 | AST 与实际路径判据 13 项通过；完整目录 fixture 待平台环境修复，见[发布审计](handoffs/continuation-release-audit-20261115.md) |

上表描述 2026-11-15 已完成的历史批次：该批完整源码门禁与双轴审阅通过，CRLF 真实回车问题已复审关闭。之后积累的 draw.io 克隆/批删工作另属未验收增量。2026-11-19 的最新三处修改未重跑编译、测试或 vet；最近两次定向检查均 exit 1，原始失败保留。Markdown 嵌套列表、表格单元格草稿恢复仅完成交接设计，未实现。当前提交用途是完整工作快照交接，参见[停点与验证范围](../HANDOFF.md#暂停时的三个任务)。

## 交付前仍需完成

| 工作 | 当前事实 | 完成判据 |
|---|---|---|
| 新 Windows 包 | UI体验包已完成五个GUI PE、26文件、四引擎八路径及ZIP独立核验；历史`.33`失败保留 | 体验包可交付；正式发行还需下列运行验收 |
| 实际 worker 生命周期 | 新源码的真实随包 worker 验收仍待执行 | 四类请求及失败/取消路径通过，进程与 Job 清理完成 |
| 宿主与会话 | 新包的正常关闭、第二次启动与会话恢复仍待执行 | 关闭不丢草稿；恢复内容/次序/选区正确；所属进程退出 |
| Windows 桌面输入与布局 | 当前宿主未提供可用的 Windows 桌面自动操作入口 | 实际键鼠、中文 IME、剪贴板、原生选择器、窄窗和显示缩放验收 |
| 资源表现 | 源码预算与局部基准已有覆盖；当前整应用指标待测 | 固定工作负载下记录启动、编辑、切换与关闭时的 CPU/内存，并解释异常 |
| CI | 原生 CI 草案已准备，未激活 | 上述开发与交付检查收敛后启用并记录实际运行结果 |

历史`.33`的PowerShell目录创建失败及Low完整性标签诊断保留。新UI体验包使用已审阅的固定五目标本地Go/Python包装器，沿用工具/补丁pin与离线约束，单次构建成功；未修改工具链标签、系统权限或旧产物。七场景十一张MyGo测试器实渲图覆盖深浅主题和760×480窄窗；物理输入与显示缩放仍归完整桌面验收。

## 后续功能完善方向

这些是公开说明的原生能力缺口，需要分别设计、实现与验收，不计入当前功能已经完成：

1. Markdown 嵌套列表、图片内嵌显示/粘贴资源管理、Mermaid、更多富文本操作。
2. 导图图片、关系线/概括完整交互，以及标签样式、排序和字段搜索。
3. draw.io的Ctrl多选、六向对齐、Shift框选与整组移动已完成，接口及保真合同见[实施计划](<F:/AIproject/LevNote/native/handoffs/feature-parity-drawio-next-20261115.md>)、验收见[多选验收](<F:/AIproject/LevNote/native/handoffs/drawio-multiselect-root-acceptance-20261115.md>)；之后补克隆/批删、分组、复杂形状、图片和stencil支持。
4. PNG/PDF 导出与打印；代码折叠、多光标与语言服务。
5. 原生国际化、无障碍和其他操作系统验收。

优先顺序是保存/草稿/撤销等数据可靠性、现有命令的完整行为、实际发行验收，再推进新增格式能力。避免把有限子集描述成完整旧版功能对等。

## 开发验证入口

以下为用户重新授权开发后的复现入口，本次交接未执行。仓库保留[原门禁脚本](../.localtools/gate.sh)及[迁移说明](../HANDOFF.md#环境与验证入口)；原脚本固定 F:/AIproject/LevNote 和本地 Go 1.27.3，新机器先准备工具链并核对路径、离线 vendor、缓存与临时目录。先运行与改动有关的回归，收敛后执行一次适当的完整受影响检查；不要把交接快照纳入测试。

```bash
bash .localtools/gate.sh test -count=1 -timeout=300s ./app/ ./cmd/... ./core/... ./formats/... ./modules/... ./ui/...
bash .localtools/gate.sh vet
bash .localtools/gate.sh build
```

历史验证记录见[验证文档](VALIDATION.md)。本页提供当前结论，历史记录保留原时间、范围和实际结果。
