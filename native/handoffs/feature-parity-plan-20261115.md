# 成熟软件能力对齐 · 第一批

用户已委托后续开发技术与产品决策。原生MyGo、纯本地、自研内核、字节保真、已有四固定worker结构保持。对标功能行为，不宣称完整产品等价。

## 本批实现合同

- VS Code：命令面板，Ctrl+Shift+P/F1和工具栏入口，中文/英文关键词过滤、上下键/Enter/Esc、即时可用状态、原有动作入口、IME/草稿/模态/过期目标保护。
- VS Code：Alt+Up/Down移动逻辑行块；Shift+Alt+Up/Down复制。UTF16选区方向/下一行列0排除、CRLF/LF/EOF、readonly/IME/容量、一动作一undo。
- Typora：Markdown块草稿粗体/斜体/行内代码，按钮与局部快捷键，复用草稿同步、冲突及保存链。以现有parser实际语义为边界。
- XMind：节点文字色、底色、边框色与逐项恢复默认，画布/SVG一致，未知样式键保留、no-op零历史、一次操作一undo，目标捕获/Flush后重验。

## 所有权与验证

lead拥有app命令面板与外壳接线；gap-editor拥有ui/editor与core/text新增行规划；gap-markdown拥有formats/markdown与modules/markdownview，新增app专属测试；gap-graph拥有formats/mindmap与modules/graphview限定颜色面板。无并发同文件写入。

基线：.localtools/feature-parity-20261115-baseline，405产品文件。测试从固定gate执行，首轮失败和每次修正均保存新的证据目录。最后统一冻结后完成一次六根回归、vet、五GUI目标构建和包哈希核验；截图为MyGo Tester，不计作实体Windows输入验收。

## 后续队列

Markdown嵌套列表与图片管理；导图字段搜索、关系线/概括和图片；draw.io多选/对齐、分组和复杂形状；PNG/PDF导出；代码折叠、多光标；国际化、无障碍和跨系统验收。先验证当前实现和数据可靠性，再按用户价值与依赖推进。
