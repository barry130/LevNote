# 原生编辑能力首批接续 · 2026-11-19

> 已停止：同日用户要求暂停开发并交由其他 Harness 接手。下文是原计划，不是当前执行授权。三个子代理已释放写入权；实际停点、证据和接续入口见 [HANDOFF.md](../../HANDOFF.md)。

## 目标与授权
用户要求以 LevNote 为基础借鉴 NoteBoard，继续后续开发；最多三个星驰 GPT-6.1 Sol 子代理实现，lead 分工、质量审查与验收。主线 Go1.27.3/MyGo，自研格式/编辑内核，纯本地，不引入账号、同步、在线编辑器或新的第三方文档内核。借鉴功能行为与流程，按本项目结构独立实现。

## 当前基线
HEAD f2c2ceb82acecb40fb0502c047e41ba12a12be47 加现有未提交、未跟踪源码，均须保留。正式接受包 .29 与 .6 体验包、全部旧证据不覆盖。draw.io 克隆/批删已有增量，先核实际源码与 20261117 修正判据，再补尚未闭合项。

## 并行分工与写域
1. md-lists：native/formats/markdown/{parser.go,types.go,export.go}、新增列表格式文件/测试；native/modules/markdownview/{blocks.go,rows.go,tasks.go,links.go} 及新增嵌套列表测试。不得改 draft.go、table_*、view.go、app、graphview。完成有序/无序/任务混合的常用嵌套列表解析、原生缩进展示、任务操作和 HTML 语义导出；原始字节范围保留。嵌套多段/代码围栏若不能完整支持须保留原文并清楚界定；深度、元素数沿用或明确限制，不能递归耗尽栈或截断不提示。
2. md-drafts：native/modules/markdownview/{draft.go,table_cell.go,view.go}、新增草稿捕获文件/测试；native/ui/editor 内中性新增契约文件；native/app/state.go、必要的独立恢复适配文件/新测试；native/core/session 仅在必要时新增后向兼容模型/测试。不得改列表上述写域、graphview、ui/editor/editor.go。设计并实现未写回表格单元格草稿的有预算会话恢复，捕获不改变主文档、不隐式完成编辑/组合输入、不覆盖原文件，恢复成明确的独立草稿。先回报契约与所需精确文件，lead 审核后实施。
3. drawio-finish：native/formats/drawio 的 clone/delete 实现与测试；native/modules/graphview 对应 clone/delete/adapter 文件和新测试；native/app/{drawio_clone_app_test.go,drawio_multi_delete_app_test.go}。先审六项既有 P2 是否已修，再完成实际缺口及其回归。不扩分组、跨页或任意 edge 克隆。不得改 Markdown 或 app 通用文件。

跨写域需求先给 lead；不擅自改他人文件。每人独占自己的新增 handoff 文件与 .localtools/noteboard-parity-20261119/<agent>/ 证据。根独占计划、公共文档与集成证据。不得子代理再委派、Git变更、修改共享记忆或旧产物。

## 验收
- 单文件 read→edit/write；禁止 Python/批量脚本修改工程文件（≤10文件尤其严格）；工具生成格式化仅限本代理明确拥有的单文件。
- 开发先保存相关反例与失败，修复后验证真实用户行为、撤销/只读/冲突/CRLF/Unicode/资源限制；不改断言弱化覆盖、不新增 skip、不按实现抄测试。
- 仅使用 bash .localtools/gate.sh test ...，固定 Go1.27.3、离线依赖。每人只跑自己相关小包并保存新命令、输出及真实 exit；禁止同时全量 build/vet。
- 完成时给出文件清单、行为与边界、测试原始路径及 exit，停止写入。lead 检视相对本轮快照的完整 diff，安排规范/需求双轴独审，处理阻塞，再统一六根 test、vet、build。
- 新包由 lead 在源码接受后安排。源码测试与预览构建不冒充 Windows 物理输入、输入法、剪贴板或整应用性能验收。
