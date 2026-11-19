# draw.io 下一批：多选、六种对齐与整组移动

日期：2026-11-15。性质：只读源码调查后的实施提案；本轮唯一写入为本文。生产源码、测试、历史证据和包均未改动，未运行 Go，未派生代理。以下接口、文件和测试中标注“拟新增”的内容尚未实现。

## 结论与建议顺序

用三个小增量完成：① 原子批量位置补丁与六种对齐计划；② Ctrl 多选与六个真实对齐动作；③ Shift 框选与整组选区拖动。前两项构成首个可交付功能闭环，第三项独立跟进。当前 .3 预览交付与冻结由 lead 统一安排，本提案不改变其源码范围。

实际图形视图在 [graphview](<F:/AIproject/LevNote/native/modules/graphview/adapter.go#L64>)，draw.io 与思维导图共享该 view；没有把实现任务分配到假定的 diagramview 路径。已加载 frontend-design 指导，沿用现有原生控件、主题和紧凑检查器，不引入 Web 编辑器、图标库或新依赖。记忆检索返回历史迁移/保真资料；工作区流程库没有匹配的 draw.io 多选流程，以下结论均取自当前源码。

## 已核实的现状

| 事实 | 直接源码入口 | 对实现的约束 |
|---|---|---|
| 当前只有 selectedID、draggingID 和一个拖动基点；pageEpoch、releaseEpoch 已存在 | [adapter.go:84](<F:/AIproject/LevNote/native/modules/graphview/adapter.go#L84>) | 增加 draw.io 专用选择集合；保留思维导图单选行为 |
| 普通左键命中节点后选中并准备单节点拖动；空白左拖和非左键拖动进入平移；连线已有单选 | [input.go:103](<F:/AIproject/LevNote/native/modules/graphview/input.go#L103>) | 框选须明确手势，不直接抢占现有空白平移 |
| 拖动期间只更新 view 偏移；松手计算坐标并按 4 单位网格取整，随后 moveDiagramNode | [input.go:174](<F:/AIproject/LevNote/native/modules/graphview/input.go#L174>) | 保留“预览不写源、松手一次提交”，整组移动只对公共位移吸附 |
| 命中采用倒序节点与轴对齐矩形，使用 screenToWorld；命中宽高最低 8 | [geometry.go:56](<F:/AIproject/LevNote/native/modules/graphview/geometry.go#L56>) | 框选使用文档几何，不拿命中最小宽高或窗口绝对坐标参与对齐 |
| UpdateGeometry 只改一个节点，写 x/y/width/height 四属性 | [edit.go:51](<F:/AIproject/LevNote/native/formats/drawio/edit.go#L51>) | 对齐只需 x/y；新接口应保留 width/height 的原始词法与省略状态 |
| Serialize 克隆当前源；patchStartTag 只替换一个起始标签；rewriteAndReparse 成功后才替换 Document | [edit.go:14](<F:/AIproject/LevNote/native/formats/drawio/edit.go#L14>)、[edit.go:197](<F:/AIproject/LevNote/native/formats/drawio/edit.go#L197>) | 批量必须先收集全部补丁，再构造一次候选并验证，避免部分成功与 N 次整文解析 |
| applyDrawioMutation 克隆解析候选、变更、序列化、重解析，再提交整个文本缓冲区 | [edit.go:302](<F:/AIproject/LevNote/native/modules/graphview/edit.go#L302>) | 可参考数据路径；新交互不要逐节点调用此入口形成多个撤销项 |
| commitBufferSelection 使用 BeginGroup/ReplaceUTF16/EndGroup，再 Changed；可在事务内保留 UTF-16 数值选区并夹取合法边界 | [edit.go:22](<F:/AIproject/LevNote/native/modules/graphview/edit.go#L22>) | 单次几何操作用该事务契约，no-op 在开组前退出 |
| deleteDrawioCell 已覆盖 owner、buffer/version/selection、pageEpoch、releaseEpoch、fieldIdentity、Flush receipt 与回调重入 | [drawio_delete.go:14](<F:/AIproject/LevNote/native/modules/graphview/drawio_delete.go#L14>) | 新批量动作以这个严格入口为参照，避免较旧通用变更入口缺少的身份复核 |
| Flush 调用 flushOrdinaryField；换页先 Flush，成功后清理选择/手势/草稿 | [draft.go:198](<F:/AIproject/LevNote/native/modules/graphview/draft.go#L198>)、[draft.go:436](<F:/AIproject/LevNote/native/modules/graphview/draft.go#L436>) | 多选切换和变更先解决既有属性草稿；失败保留原 owner 与草稿 |
| 源版本或缓冲区替换会同步并取消现有手势；丢失选中对象时当前代码会选第一页中的首节点 | [sync.go:48](<F:/AIproject/LevNote/native/modules/graphview/sync.go#L48>)、[sync.go:164](<F:/AIproject/LevNote/native/modules/graphview/sync.go#L164>) | 多选同步需显式修剪；避免旧集合在新页/新 buffer 同名 ID 上复活 |
| readonly 包含宿主只读、页只读与预览限制；压缩页保留原载荷且只读 | [sync.go:253](<F:/AIproject/LevNote/native/modules/graphview/sync.go#L253>)、[parse.go:396](<F:/AIproject/LevNote/native/formats/drawio/parse.go#L396>) | 多选浏览可用，所有几何写入继续受这些门约束 |
| Node 提供 Parent、Style、Unsupported 和私有源偏移，但没有公开 relative/旋转/分组几何合同；无效坐标可能被解析成零并产生 issue | [types.go:70](<F:/AIproject/LevNote/native/formats/drawio/types.go#L70>)、[parse.go:824](<F:/AIproject/LevNote/native/formats/drawio/parse.go#L824>) | “能画出矩形”不等于“可按绝对矩形移动”；新能力须验证实际源结构 |
| 连线端点从节点当前展示位置计算，折点仍用 XML 坐标 | [draw.go:267](<F:/AIproject/LevNote/native/modules/graphview/draw.go#L267>) | 移动只改变选中顶点，连接关系与手工折点字节保持；不会自动平移整条连线 |

## 增量一：原子位置补丁与纯对齐计划

拟新增 `native/formats/drawio/positions.go` 和 `native/formats/drawio/positions_test.go`；仅在必要时窄改 [types.go](<F:/AIproject/LevNote/native/formats/drawio/types.go>) / [parse.go](<F:/AIproject/LevNote/native/formats/drawio/parse.go>)，记录批量位置编辑资格所需的私有解析事实。优先复用原有标签补丁器与 owned-source 解析路径，不改变公开 Parse/Serialize 合同。

建议接口（以下全部是提案）：

```go
// PositionUpdate changes location only; dimensions remain byte-exact.
type PositionUpdate struct {
    CellID string
    X, Y   float64
}

type Alignment uint8
// AlignLeft, AlignCenterX, AlignRight,
// AlignTop, AlignCenterY, AlignBottom

func (d *Document) ValidatePositionTargets(pageIndex int, cellIDs []string) error
func (d *Document) PlanAlignment(pageIndex int, cellIDs []string, alignment Alignment) ([]PositionUpdate, error)
func (d *Document) UpdatePositions(pageIndex int, updates []PositionUpdate) error
```

ValidatePositionTargets 是 UI 可用性与两种变更共同使用的资格判定；PlanAlignment 只计算，完全不修改 Document；UpdatePositions 重验当前对象并原子提交。避免 UI 和格式层各写一份稍有不同的资格规则。

首批资格固定为同页、普通绝对坐标顶点，宽高为有限正数，具有唯一且可定位的 mxGeometry。默认先支持 canonical 0/1 容器中 parent=1 的叶顶点；父容器身份从实际 XML 核对，不能只比 Parent 字符串。支持矩形/圆角/椭圆/菱形的未变换几何；组、子顶点、相对坐标、旋转、swimlane/布局变换、图片/stencil、未知 shape 退化渲染等保持原样并给出具体不可用原因。重复请求 ID、缺失 ID、边 ID、重复 geometry、无效数字归零 issue、几何未知变换、存在子项的组顶点均使整次动作退出，禁止静默跳过其中一项。普通不参与几何的扩展 metadata 可原样保留，其他页面或节点的未知内容不应全局封禁该动作。

实现资格所需的 relative、几何子项/重复 geometry、实际形状和父层级信息由格式层从原始 XML 取得：可在解析时记录私有事实，也可在一次预检中读对应源片段；不依靠 UI 反查 XML，不把 inferShape 的 fallback 当证据。缺少 x/y 的合法绝对顶点视为零；只有实际发生位置变化的轴才插入该属性。已存在的 x/y 若数值未变，也保留诸如 1e2、100.0、负零等原有字面量。

批量算法固定为：先验证全部目标、坐标和来源范围 → 对各目标起始标签调用现有 setTagAttributes 生成最小 x/y 补丁 → 按源偏移排序并检查互不重叠 → 分段拼接一份候选 → 一次 rewriteAndReparse → 成功后替换 Document。不要在 Document 上顺序调用 N 次 UpdateGeometry；后者不仅多次整文解析，而且失败时格式对象已被前 N-1 次变更。私有 source、original、Source 导出副本、Dirty 的所有权规则照旧；候选失败时它们和源偏移全部保持原状态。

对齐基准是当前选择集合的整体轴对齐外包框，不以首选或最后点击节点为锚。以 L=min(x)、R=max(x+w)、T=min(y)、B=max(y+h) 计算：

| 按钮 | 每个选中节点的新坐标 | 另一轴 |
|---|---|---|
| 左对齐 | x=L | y 不变 |
| 水平居中 | x=L+(R-L)/2-w/2 | y 不变 |
| 右对齐 | x=R-w | y 不变 |
| 顶对齐 | y=T | x 不变 |
| 垂直居中 | y=T+(B-T)/2-h/2 | x 不变 |
| 底对齐 | y=B-h | x 不变 |

计算用 float64；端点、边界跨度和结果均验证有限数，处理加法溢出。对齐不吸附网格，允许半单位中心。少于两个有效顶点时动作不可用；已对齐时返回空补丁并保持 bytes/Dirty/版本/撤销/Changed 均不变。所有节点尺寸、标签、样式、parent、ID、连接关系和层叠顺序保持。

本增量完成标准：格式级六种结果与手算相符；同一批含一个失败目标时整份源零变化；成功差异仅位于被选节点需要变化的 x/y 属性；压缩页及未知 XML 片段逐字节一致。它是后续 UI 的基础，尚不单独宣称用户已获得多选动作。

## 增量二：Ctrl 多选与六对齐闭环

拟新增 `native/modules/graphview/drawio_selection.go`、`drawio_alignment.go` 及对应测试。窄接线涉及 [adapter.go](<F:/AIproject/LevNote/native/modules/graphview/adapter.go>)、[input.go](<F:/AIproject/LevNote/native/modules/graphview/input.go>)、[draw.go](<F:/AIproject/LevNote/native/modules/graphview/draw.go>)、[sync.go](<F:/AIproject/LevNote/native/modules/graphview/sync.go>)、[view_ui.go](<F:/AIproject/LevNote/native/modules/graphview/view_ui.go>)、[draft.go](<F:/AIproject/LevNote/native/modules/graphview/draft.go>)。共享文件必须由同一名 view 实现者串行修改；思维导图颜色功能属于此前已冻结内容，按下一批新基线审阅。

拟新增 view 私有 drawioSelection：有序唯一 IDs、membership、primaryID、selectionEpoch、buffer/page 身份。现有 selectedID 暂作 primary 兼容字段；所有 draw.io 选择变更集中到一个入口，禁止并行维护两个互相漂移的“真值”。思维导图继续原单选。

手势合同：

- 普通单击未选节点：替换为单节点；点击已选集合成员：保持集合，把该节点设为 primary。双击明确收为单节点并进入现有标签编辑。
- Ctrl+单击顶点（沿现有 ui.Cmd 平台修饰语义）：加入/移除集合；移除 primary 时从剩余有序集合稳定选一个；不启动拖动或编辑，不改变文档内容。单选边继续原行为；Ctrl 点边保留原集合并显示“多选仅包含图形”。
- 普通空白单击清空选择，原空白左拖平移仍保留；Escape 首先取消当前手势/连接模式，无手势时清空多选。选择变化本身不进文档 undo。
- 可以选择只读或尚不支持几何变更的顶点来查看；六对齐可用性通过共同 validator 判断。集合含任何不支持对象时整组停用，并显示原因和数量，不只处理可移动子集。
- 多于一个节点时检查器显示“已选择 N 个图形”、六对齐与可用性原因；原单节点文本、删除、连接入口全部收起或停用。尤其 Delete/Backspace 和连接快捷路由必须再次检查集合大小，防止视觉多选却只删除/连接 primary。
- 本增量尚未实现整组拖动时，多选集合的指针拖动不写源，并在检查器明示“整组移动待下一增量”；不悄悄退化为 primary 单节点移动。单节点原有拖动照旧。增量三完成后移除该阶段提示。

六按钮放检查器内两行三列，不继续加宽现有顶栏：第一行左/水平居中/右，第二行顶/垂直居中/底。给每个按钮稳定 Key 与完整中文 Label，保留原生键盘焦点与禁用态；采用与已有图形样式动作一致的 KeepFocus / ObserveInput 手势捕获，避免按钮先提交 IME 再以新对象执行。新功能不占用全局 Ctrl+A/C/V/G，也不扩张 Editor.Host 公共契约。

选择生命周期：同一 buffer、同一页身份的普通源同步仅保留仍存在的 ID；源替换、换页、Release 清理集合、捕获动作和拖动预览。page 身份结合 pageEpoch 与当前页面实际身份，不能只凭可重复的 page ID。source 同名节点替换使待执行手势失效，即使新的集合仍包含该 ID。undo/redo 后按当前源修剪选择；不添加“选择变动撤销项”。初次自动选首节点时同步初始化单元素集合。

六对齐的事务入口拟为 applyDrawioAlignment(mode Alignment)。参照 [deleteDrawioCell](<F:/AIproject/LevNote/native/modules/graphview/drawio_delete.go#L14>) 的严格模式：

1. 在任何同步、Report、Flush 或焦点迁移之前捕获 buffer、version、UTF-16 selection、releaseEpoch、pageEpoch/index、selectionEpoch 与 IDs、primary、fieldIdentity；toolbar 按下/键盘激活绑定同一快照。
2. 检查 readonly、InGroup、flushing、CompositionActive、parse/preview 状态及全部格式目标；格式不适用或 no-op 时在 Flush 前退出，保留未提交属性草稿。
3. 调用 flushOrdinaryField，使用真实 receipt 核验版本/选区的唯一允许变化；必要时在已提交标签的新源上重新计算计划。失败保持草稿和选择，不清理后继续。
4. CheckReplaceUTF16 预检容量；核验 owner/source/selection 再执行一次 commitBufferSelection(value, true)，用已有 flushing 守卫阻止 Changed 同步重入另一几何动作。
5. Changed 返回后复核精确 buffer/version/source/selection/page/selectionEpoch/release 所有权，再同步缓存和保留选择。回调若换 buffer、换页或 Release，不恢复旧草稿、旧集合或旧手势。

“一个事务”指一次六对齐动作的全部坐标只有一个主缓冲区 undo 项与一次 Changed。动作前已有的普通标签草稿仍按原 Flush 单独提交，顺序为“标签事务 → 对齐事务”；一次 undo 撤对齐、再次 undo 撤标签。不能把既有标签历史揉进几何事务，或因对齐失败悄悄丢弃它。容量失败保持几何零修改；若前置标签 Flush 已成功，其独立提交事实如实保留。

新增应用纵向测试拟放 `native/app/drawio_alignment_app_test.go`；该文件和 graphview 共享源码由 lead 分配明确写域。此增量闭环验收后才可声称“Ctrl 多选 + 六对齐可用”。

## 增量三：Shift 框选与整组拖动

复用增量二选择模型与增量一位置补丁。拟新增 `native/modules/graphview/drawio_gesture.go` / 对应 Tester 测试，窄改 [input.go](<F:/AIproject/LevNote/native/modules/graphview/input.go>)、[geometry.go](<F:/AIproject/LevNote/native/modules/graphview/geometry.go>)、[draw.go](<F:/AIproject/LevNote/native/modules/graphview/draw.go>) 及手势清理入口。

Shift+左键从空白处拖动启动框选，Ctrl+Shift 表示与手势起始集合求并；普通 Shift 框选替换集合。采用完全包含的轴对齐包围盒，拖动方向不改变规则，边不进入框选集合。4 屏幕像素阈值以下不生成框选；Move 只画裁剪在画布内的半透明框与候选描边，Up 才写 view 选择状态；Canceled/Escape 恢复起始集合，主 buffer 始终零变化。坐标先减画布窗口原点，再用 screenToWorld；缩放/平移/resize 或页/源身份改变时取消，避免同一框跨坐标系解释。

整组拖动在已选节点上起步，按下即冻结 IDs、各节点原始 float64 矩形及 buffer/version/page/selection owner；Move 只更新展示偏移。以 primary 的目标位置按现有 4 单位网格吸附一次，得到共同 dx/dy，再加到每个基点；不逐节点吸附，保留节点间原始相对距离。展示和最终提交使用同一公共位移。节点 hit-test、diagramNodeBox 与 displayNodePosition 应共用该临时位置查询，连接端点随之预览。

Up 只提交一次 UpdatePositions + 主缓冲区事务；零位移、Canceled、Escape、readonly、IME、源更新、标签页切换、Release、迟到鼠标释放均不写源。进行中的手势不跨普通标签页生命周期存活；现有 ensureSynced 取消单拖的门扩展到多拖与框选，并加入选择 epoch。显式手工折点、游离端点和边 XML 不随顶点组平移；本批不宣称完整 draw.io 分组连线布局。

## 复制、删除与分组的明确边界

复制/粘贴/克隆不进入本批。它们需要独立定义：ID 生成、选中子图的连接线闭包、跨页/跨文档引用、未知 XML 包装器与 metadata 的复制语义、剪贴板平台边界、失败原子性。不能用 Serialize/Parse 后遍历 AddShape 当作保真复制，也不临时接管 Ctrl+C/V。

分组/取消分组不进入本批。它们涉及 parent 图、相对坐标、祖先变换、子树移动、组边连接与 XML 层级；当前 Node 的平面几何和 fallback 渲染不构成已支持这些语义的证据。选择集合仅是 view 状态，不写入 XML，也不命名为 group。已有文件中的组和未知层级原样保留。

批量删除另开小批：当前 DeleteCell 处理单 cell 与关联边，独立的整组选区删除需要闭包去重及关联线策略。当前多选状态下按 Delete 明确不执行单节点删除；单选的现有删除仍照旧。缩放、多选浏览、已有保存/导出、源码视图继续保留原入口。

## 回归与证据：真正操作视图、真正保存，再核对源

本轮没有执行以下测试。实施时由 lead 串行运行当前 Windows gate，保存新证据目录和首次失败；不覆盖 markdown-03 或其他闭合证据，不使用旧阶段 test namespace。

格式层（拟新增 positions_test）：

- 六种对齐用不同尺寸、负坐标、小数、零轴省略、非文档顺序 ID；直接断言手算坐标，width/height 与其词法保持。验证重排选取顺序不改变结果。
- no-op 不改 bytes/Dirty；空更新不改源；少于两个、重复/缺失 ID、edge、group/relative/rotation、非法/缺失 geometry、非有限输入、边界溢出均整批零修改。把失败目标放首/中/末，证明无部分成功。
- 混合 CRLF/LF、单/双引号、实体、属性次序、注释、namespace、处理指令、未知元素/属性、HTML 文本、其他页/压缩载荷、显式连线折点的精确字节 oracle。对选中节点只替换需要变化的 x/y，其他切片相等；不要只验证“重新解析没报错”。
- 原 input、OriginalBytes、导出 Source 隔离，失败重解析的原子性与解析 budget 原样有效。复用 [既有源所有权测试](<F:/AIproject/LevNote/native/formats/drawio/source_retention_test.go>) 和 [文本/几何未知 XML 保真测试](<F:/AIproject/LevNote/native/formats/drawio/drawio_test.go#L92>)；增加多页混合原始片段，保持历史 fixtures 不动。

真实 graphview Tester（拟新增 selection/alignment/gesture tests）：

- 按 [现有真实拖动测试](<F:/AIproject/LevNote/native/modules/graphview/graphview_test.go#L147>) 创建 DrawioAdapter 和 ui.NewTester，通过真实 canvas 几何和输入事件 Ctrl 选/取消，点完整 Label 的六按钮；不只直接调用内部 mutate 方法。
- 测普通点击、已选点击、双击编辑、Ctrl 点边、Escape、空白平移、primary 移除；多选 Delete/连接/属性输入均不落单节点修改。只读/压缩页仍能浏览选择，几何操作无源变更。
- 每个六对齐动作观察 Changed 次数、buffer Version、文本、正向/反向 UTF-16 source selection；一次 undo/redo 覆盖整批。再测前置标签草稿形成两项独立历史。no-op、失败、空选择保持 redo 栈。
- pending IME、Flush 冲突、容量不足、host.InGroup、Report/Changed 中换 buffer/页/只读/Release、按下和点击之间源变化、同 ID 内容替换、重复/迟到事件，不得向新 owner 写入或复活旧选择/草稿。
- 框选正反方向、全包/部分相交、Ctrl 并集、不同 pan/zoom、取消和 resize；整组拖动验证公共 snap delta、Move 阶段主源未变、Up 只有一个事务、连接两端/单端随节点预览而折点字节保持。
- 760×480 与 1180×780，深/浅主题，六按钮与数量可见、可聚焦、可点击，状态/原因不遮画布和底部状态栏；源/分屏模式与属性草稿保留。保存至少四张实渲图并人工读取图像。

应用纵向回归（拟新增 drawio_alignment_app_test）：

- 从真实临时文件 Open → 渲染首帧 → Ctrl 多选 → 点击对齐 → Save/SaveAs → 关闭再打开并渲染 → 校验几何及磁盘逐字节预期。必须是实际 app 文档/Host 路径，不用只调用 Serialize 替代保存。
- UTF-8+BOM、CRLF、多页含压缩第二页、非默认扩展名路由、只读 SaveAs 新副本、分屏 UTF-16 选区、undo/redo 后 Save、外部源变化和失败 Flush 保留。沿用 [drawio_route_test.go](<F:/AIproject/LevNote/native/app/drawio_route_test.go>) 的真实路由/保存方式和 [real_view_integration_test.go](<F:/AIproject/LevNote/native/app/real_view_integration_test.go#L220>) 的真实 IME 草稿通道。
- 最终回归应覆盖既有单选拖动/添加/连接/删除、drawio source retention/压缩页/导出 selected page、graphview 全套及 app 真实图形生命周期。共享 view 同时验证 mindmap 选择、颜色/标记/标签/草稿功能，以发现跨格式回归。

建议聚焦包顺序为 formats/drawio → modules/graphview → app 相关选择测试；通过后执行一次受影响完整 suite 和 vet。具体 gate 命令、证据命名、全产品汇总和打包时点由 lead 统一确定。上述 Tester/文件证据不等同于实体桌面 Ctrl/Shift、系统 IME 或整个进程性能实测；这些继续进入最终平台验收。

## 可执行的分工与验收交接

| 增量 | 唯一实现写域建议 | 接受条件 |
|---|---|---|
| 1 | 格式 owner：新 positions 文件、窄 parser/type 私有事实及其测试 | 所有资格与原子字节 oracle；接口冻结给 view owner |
| 2 | view owner：graphview 选择/对齐新文件及共享接线；app owner 只写新增 app 测试 | Ctrl 多选到保存重开完整闭环、六几何公式、单事务、回调/草稿守卫、四图 |
| 3 | 同一 view owner：框选/多拖新文件与接线；app 测试扩展由原 owner | 整组相对位置不变、完整取消/失效、主源仅 Up 一次提交 |

格式 owner 与 view owner可以并行准备，但共享 graphview 文件只设一个写者；root 串行测试/验收，审阅者只读检查 diff 和独立原始输入输出。所有当前冻结功能先形成新批基线，再启动该提案；不以“已有单节点能力”代替多节点可复现验收。

下一条具体实施指令可直接指向：先新建 native/formats/drawio/positions_test.go 的六对齐与混合未知 XML 原子用例，再实现 positions.go；格式接口通过后接 graphview Ctrl 多选与六动作。未运行的计划测试不得记为 PASS。
