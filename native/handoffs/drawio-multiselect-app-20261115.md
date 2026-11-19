# draw.io 多选对齐 App 纵向测试交接

日期：2026-11-15；负责人：gap-editor；状态：初版源码冻结，产品测试待根代理执行。

## 写域与验证状态

仅新增 [主测试](<F:/AIproject/LevNote/native/app/drawio_alignment_app_test.go>)、[保护路径测试](<F:/AIproject/LevNote/native/app/drawio_alignment_app_guards_test.go>) 和本交接文档。既有测试、生产代码、旧证据与已接受包均未改动；未创建 commit、未 push、未派生代理、未写共享记忆或共享白板。

采用本会话已加载的 tdd 技能及公开接口测试原则。此子任务未启动 Go test、vet、build 或产品进程；根代理统一执行与记录。当前无 PASS 结论。

固定格式器为 F:/AIproject/LevNote/.localtools/go1.27.3/bin/gofmt.exe，以 spawnSync(exe, [绝对文件路径], {encoding:'utf8'}) 读取 stdout，经文件工具写回，未用 -w。首次格式检查发现生成文本中两处 CRLF 转义变成源码换行，status=2；针对性修复后两个文件的最终格式检查均 status=0、stderr 为空。该结果只证明语法格式检查。

## 冻结输入

| 文件 | 字节 | SHA-256 |
| --- | ---: | --- |
| [主测试](<F:/AIproject/LevNote/native/app/drawio_alignment_app_test.go>) | 15104 | 0830cc0fdac80d8243840c115ff743d1cb30ea03936a1d4d08e2b4084e10216b |
| [保护路径测试](<F:/AIproject/LevNote/native/app/drawio_alignment_app_guards_test.go>) | 11834 | 6e0f496257152d256a77c3de8d8bdcfa2d6e3ca5d8c0efd451fb613c75528b9c |

## 真实路径与独立 oracle

- 临时工作区真实 UTF-8 BOM 文件 → App.Open → 已注册 draw.io Adapter → App.View / Tester.Frame；容量样本在首帧、真实 Host 创建前安装公开 text.NewWithLimit，并保持 Editor/Document 同一 buffer。
- 从 Tester.Image 中只扫描已渲染的“draw.io 画布：对齐页”区域，用三枚独有填色与面积门定位真实矩形，再发 ClickAtWith(ui.Cmd, ...)；没有读取或修改 graphview 私有状态，没有复制 fit/pan/zoom 算法。
- 以真实画布 Escape 清空旧多选，再普通选 a，Ctrl 选 c、b。选择顺序刻意不同于 XML 顺序。普通点击已选成员保留集合的语义不被测试辅助函数误当成“单选重置”。
- 六方向期待几何标签和浮点数均为手算固定字面量，没有调用 PlanAlignment、UpdatePositions 或序列化器生成期望值。L=-12.5、R=205.25、T=8.25、B=221.25；三节点尺寸依次 81×42、62×25、99×31。
- 样本同时包含小数与科学计数法尺寸、正号/尾零、混合单/双引号、属性乱序、CRLF、BOM、中文与 astral scalar、未知属性与带命名空间元数据、CDATA、未选顶点、带人工路由点的连线、有效压缩第二页。
- 压缩页固定载荷已由独立 Node raw-DEFLATE 解码为含 compressed-shape/保留🙂/17,23,90,40 的合法 XML；产品 parser 验证仍待根执行。
- 既检查完整 XML/BOM 字节，也用公开 drawio.Parse 检查实际节点坐标/尺寸/parent、连线端点与两个人工路由点、压缩页。Save 前原盘保持，Save 后逐字节一致，再 CloseTab/Open/Frame 重开实际文件。

## 测试矩阵

6 个顶层测试、18 个叶场景：

| 测试 | 场景与关键断言 |
| --- | --- |
| [SixCommandsSaveReopen](<F:/AIproject/LevNote/native/app/drawio_alignment_app_test.go#L277>) | 六按钮 × preview/light 与 split/dark，12 场景。Ctrl 选择零文档变更；按钮可达/启用；一次整组事务与一步 undo/redo；UTF-16 反向选区包含 emoji；精确保存、关闭、重开、几何/BOM/编码保留。 |
| [NoopKeepsRedoAndSelectionHistory](<F:/AIproject/LevNote/native/app/drawio_alignment_app_guards_test.go#L14>) | 左对齐、顶对齐、undo 后保留 redo；Ctrl 移除/加回节点及重复左对齐不改版本/选区/history/dirty；多选 Delete/Backspace 不删除 primary；属性编辑禁用/隐藏；redo 与真实保存。 |
| [CapacityFailureKeepsRedoAndDisk](<F:/AIproject/LevNote/native/app/drawio_alignment_app_guards_test.go#L70>) | 原文大小容量上限；公开源编辑器建立“删末尾 CRLF → undo”的真实 redo；底对齐须增长 2 字节而失败；版本、选区、history、snapshot、dirty、三选状态、原盘保持；错误可见；随后原 redo 仍恢复其独立文本修改。 |
| [ReadonlyCustomRouteSaveAs](<F:/AIproject/LevNote/native/app/drawio_alignment_app_guards_test.go#L109>) | 真实文件 Chmod 0400 → .diagramnote 插件路由；只读仍可三选、六按钮禁用、点击不变；SaveAs .diagramnote2 后可编辑，底对齐仅改副本；关闭重开副本，原盘保持。无新增 Skip。 |
| [LabelThenGeometryHaveSeparateUndo](<F:/AIproject/LevNote/native/app/drawio_alignment_app_guards_test.go#L153>) | 原生“节点文本”编辑器真实 Type 标签 → Ctrl 多选 → 水平居中；两条历史按几何/标签分别 undo/redo，最后逐字节保存。 |
| [FailedDraftBlocksCtrlSelectionAndLifecycle](<F:/AIproject/LevNote/native/app/drawio_alignment_app_guards_test.go#L192>) | source-conflict 与 capacity 两场景；真实 Compose 草稿 → 源字段冲突/容量失败 → Ctrl 点另一节点保留 owner；Save/切源模式/关闭均停止，源文、version、选区、history、原盘保持；实际点击“恢复为新文档”，再 SaveAs 文本并核对草稿字节。 |

## 根代理下一步

等待 graphview 写域冻结后，由根代理在当前 Windows 固定 gate 下对 ./app/ 使用 ^TestDrawioAlignmentApp 聚焦过滤，保存实际命令、stdout/stderr、exit 与失败；此阶段没有子代理生成的测试结果可继承。

若出现失败，先区分真实行为缺陷与 fixture/Tester 前置条件问题，保留首失败，按原独占写域串行修复。图层私有 guard、Shift 框选/整组拖动以及最小窗口截图由 graphview 负责人覆盖，本文件不宣称替代这些证据。最终 affected suite、vet、构建和交付均由根代理管理。
