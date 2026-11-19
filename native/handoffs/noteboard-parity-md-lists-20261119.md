# Markdown 列表实现交接 · 2026-11-19

## 状态
- 本代理已停止所有功能实现、修复和新测试；未执行 Git 操作。
- 本轮未写入或修改任何格式层/预览源文件或测试文件；没有未完成的源码半成品。
- 本轮未启动后台 job。停止时 job_list 可见条目：0；无待停止 job。

## 已完成的只读检查
- 阅读 `native/handoffs/noteboard-parity-batch1-plan-20261119.md` 分工与验收要求；确认 Markdown 列表任务的写域限定为 native/formats/markdown 与 native/modules/markdownview 指定文件。
- 阅读 `native/AGENTS.md` 最新首段及 Development and product constraints；本地 Go 门禁使用 `bash .localtools/gate.sh test <packages>`，要求离线 Go 1.27.3、保存实际输出与退出码。
- 阅读现有 Markdown parser/types/export 与 markdownview rows/blocks/tasks 的相关段落。当前格式层列表解析只收集同一有序/无序类型的平面 item；item 有源字节 Range、TextRange 与 checkbox 的精确字节 Range。HTML 按顶层 BlockList 输出 ul/ol 和 ordered start，但没有嵌套树；预览 rows 按平面 item 展开，taskTarget 通过 source range/version 做精确操作。
- 检查记忆流程库：无匹配的 native Go gate test 流程。

## 检查失败与未执行项
- 一次只读读取请求使用了超出 rows.go 文件总行数的 offset，工具返回 out-of-range；没有产生文件改动。
- 另一次目标 handoff 读取返回 not found，之后按本交接新建该专属文件。
- 没有运行任何测试/构建/vet；因此没有产品验证结果或测试退出码。
- 未实现嵌套列表树、缩进预览、嵌套任务定位/撤销及嵌套 HTML；没有执行 CRLF/emoji、宽窄布局、预算或不支持样本测试。

## 后续接手建议
- 先从当前工作区与 .localtools/noteboard-parity-20261119/baseline/source/ 对照核实共享基线，不覆盖/回滚任何既有 dirty/untracked 文件。
- 以一个统一的有界列表节点树承载父子关系、有序起始值、marker/checkbox 原始字节范围；格式层解析作为显示、导出、任务操作的唯一结构来源。超深度、项目数超限、嵌套多段与代码围栏必须显式 warning/保留原文，不得静默压平或冒称完整 CommonMark。
- 接手者完成源码后新增独立命名的命令/log/真实 exit 证据，并只跑负责小包的门禁；本代理没有可复用的测试结果或代码补丁。

## 文件清单
- 新增：`native/handoffs/noteboard-parity-md-lists-20261119.md`（本交接）。
- 源码/测试：本代理未修改任何文件。
