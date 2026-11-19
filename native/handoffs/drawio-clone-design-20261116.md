# draw.io 当前页多选克隆：只读设计与 XML 审计（2026-11-16）

范围声明：本文件是只读设计与审计，不构成完成、交付或已批准实现；当前工作区里 lead 仍在改
`native/formats/drawio/delete.go`、`native/formats/drawio/delete_cells_test.go` 和 graphview，相关行号是 2026-11-16 16:16 +08:00 前的“进行中快照”，实现任务须以最终源为准。建议最终格式 API 写作
`func (d *Document) CloneCells(pageIndex int, cellIDs []string, dx, dy float64) ([]string, error)`；返回值只包含克隆出的顶点 vertex ID（顺序对应输入请求的顶点顺序），内部边也分配新 ID，但不需暴露给画布选择。

## 1. 可直接执行的最小 XML 克隆合同

适用输入只限同一未压缩 `Page` 的直接单层 vertex 选择。每个 `cellIDs` 项必须非空、去重、为 `page.Model.Nodes` 已解析 vertex、且满足 `Parent == "1"`；`"0"`、`"1"`、缺失 ID、重复 ID、edge ID、嵌套/父子 vertex、`Unsupported` 非空、缺直接 `mxGeometry @as="geometry"` 都整体拒绝。渲染模型只是资格预筛，完整合同仍要跑 raw token 扫描：`parseRootCells` 只信任 `root` 的直接 `mxCell`（`native/formats/drawio/parse.go:528-615`），`parseCell` 会跳过未知子元素并保留问题（`native/formats/drawio/parse.go:618-748`），不能拿被渲染出的 `Node`/`Edge` 当作“全 XML 中没有其他同名/引用元素”的证明。

复制集只由一个顶点集和内部边生成：

1. 顶点集：按匹配输入顺序收集所有合法顶层 vertex 的完整 `<mxCell>` span。
2. 内部边：扫描 `page.Model.Edges` 及其他 raw `mxCell edge` 单元；只有 `parent="1"` 且 `source`、`target` 均在选中顶点集内时复制。显式选中的独立 edge 不在这轮最小合同里，因 graphview 多选状态成员本就排除 edge（`native/modules/graphview/drawio_selection.go:9-16`），而支持“点 edge 或 mixed 选择”需要另行处理 edge label/端点/路线。
3. 外部边：任一 endpoint 不在顶点集内的边保留在原始单元上、不生成克隆；它的 `source/target` 继续引用原件，不替它新增克隆端点。仅当外部边的属性或 style 端口键显式引用被克隆 ID 时触发第 4 步的拒收，不能把这种引用悄悄续到旧 vertex 上。
4. Port 与原始引用：最小策略固定为整体拒收——任一被复制 span 内出现 `sourcePort`/`targetPort`（属性和 style）即拒绝，不论其当前值；任一未复制单元的上述 port 引用、或 `defaultParent/currentRoot`、或未知 wrapper 内上述任一引用命中选区旧 ID 也拒绝。外部边普通的 `source/target` 指向旧 ID 则保留。现有格式层在 `native/formats/drawio/delete.go:142-172` 检查同类属性/style 引用；clone 也按 raw token 判断，不扩展为重映射能力。
5. 未知结构：HTML/image/stencil 标记出的 vertex 不属于最小可编辑顶点，因为解析器已经在 `Unsupported` 增加值（`native/formats/drawio/parse.go:652-670`）。批删当前也拒绝带这类标记的目标（`native/formats/drawio/delete.go:34-49`）；clone 同样整体拒绝，避免把“能理解位置的矩形/椭圆/菱形/圆角矩形”合同悄悄扩大成“任意未知子树复制”。
6. ID 保真生成：每个被克隆顶点/内部边获得唯一新 ID，不能保留旧 ID；必须扫描**全部本页 XML** 的 `id` 属性（不止 `pageHasCell`，后者只看已渲染模型，见 `native/formats/drawio/edit.go:211-234`），并将本次分配结果也加入占用集，禁止与未知未渲染元素重名。沿用现有前缀策略：从 `levnote-1` 起逐个探测，但占用判定来自本次 raw 扫描再加局部 `used` 集；旧 ID 绝不复用。
7. Span 字节内容除下列允许补丁外逐字节保持：`id=<new>`、vertex `mxGeometry x/y += (dx,dy)`、内部边 `source/target` 指向对应新 ID；`parent` 固定 `"1"`（这是资格校验后的原值保持，不是额外新建）。目标 vertex 与内部边的允许子树限定为唯一合法 geometry、可选 `Array as=points` 和空白/注释；未知子树、非空白文本、port 属性和 unsupported geometry 在 clone 专用 raw scanner 下整体拒绝。未知非目标元素、未被重写属性、属性顺序、引号风格、自闭合斜杠、注释、处理指令均原样保留。补丁只走 start-tag 级别重建，沿用 `setTagAttributes` 的属性边界与 `xml.EscapeText` 契约（`native/formats/drawio/edit.go:256-376`），不得用字符串正则跨整个 span 搜索。
8. 几何只支持绝对 vertex 几何：`x/y/width/height` 有限、正宽正高、无 `relative/rotation/transform/scale/offset/alternateBounds`；严格判定可复用 `positionTargets`/`validatePositionCell`/`readPositionGeometry` 的模式（`native/formats/drawio/positions.go:177-359`、:388-443`）。最小 clone 不移动 edge 的 absolute endpoint、waypoint 或 sourcePoint/targetPoint；这些点的原值保持，与批删/移动现有边界一致。
9. 插入顺序与位置：顶点按输入选择顺序复制，内部边按原文顺序附后；每次只生成一个插入字节串，放在 `Page.rootClose` 之前。`rootClose` 是未压缩可编辑 canonical root 的精确闭合标签起点；编辑入口先走 `writablePage` 锁只读页（`native/formats/drawio/edit.go:160-169`），`AddShape`/`AddConnector` 已用同一 anchor（`native/formats/drawio/edit.go:76-116`）。
10. 只读优先级：先查页/index/`writablePage`，再校验有限 `dx/dy`，再计算目标坐标；任何只读/compressed/multiple-root/不支持结构失败都要留下原 `Document` 字节、`OriginalBytes()` 和 `Dirty()` 完全不变。

对 `(dx,dy)` 的合同：一次调用共用同一个有限位移；UI 默认传固定可见偏移（如 10,10）。每个目标 vertex 的新 `x/y` 先过 `finite` 与 `validPositionBounds` 等效检查，保证目标 bounds 不溢出（`native/formats/drawio/positions.go:169-172`、:118-143`），再生成任何补丁。序列化沿用 `formatFloat`；不允许写 NaN/Inf。

## 2. 复制 span、插入位置和原子引用验证顺序

执行顺序必须验证全部条件后再产生任何 source 字节：

1. 冷资格：`writablePage(pageIndex)`；读 `.ReadOnly/Compressed/Model` 预筛，校验请求 IDs。此处先用现有解析模型拒绝简单错误，但还不能断言 span。
2. raw 核对：以 `newXMLParser(d.source, d.options)` 从文档根扫到目标 model（`positionTargets` 的命名空间安全做法，`native/formats/drawio/positions.go:174-204`），不要把 `page.modelStart` 当独立 XML 片段交给 `encoding/xml`，否则继承命名空间会丢失；一次性收集：direct mxCell 起止 span、所有现存 ID、目标 vertex/内部边资格、所有 raw `parent/source/target/sourcePort/targetPort/defaultParent/currentRoot` 引用。`deleteCellRanges` 对完整直接单元的 token 级 span 与 raw 引用检查正是这个模式（`native/formats/drawio/delete.go:115-205`）；clone 应实现自己的 scanner，因为删除要的“完整移除”与 clone 要的“保真复制+局部重写”输出不同。
3. Span 排序与原子性：选中 spans 必须完整 direct、`start < end`、互不重叠；同一 ID 重复、span 不能闭合、非 direct 或 root/layer 身份异常都拒绝。`DeleteCells` 已在 `native/formats/drawio/delete.go:185-201` 检查重叠并依赖 `parse.go:578-584` 的 duplicate-ID 硬错；clone 对 local-name ambiguity 同样零容忍。
4. 引用判定：最小合同下，任何 raw `sourcePort/targetPort/defaultParent/currentRoot` 或未知 wrapper 内的上述引用命中选区旧 ID 即整体拒绝；被复制 span 内 port 属性/style 键一律拒绝。style 检查必须按 `;` 拆分并处理重复 key（参照 `native/formats/drawio/delete.go:160-170`）。目标外普通 `source/target` 不改写；它们仍指向原件。
5. 分配与重写：为每个旧 ID 分配唯一新 ID 并建立完整 remap。vertex geometry 的 `x/y` 补丁复用 `setTagAttributes` 只改 start tag；内部边的 `id/source/target` 也在同一 start-tag 补丁中换掉。所有补丁生成完后，先用 `int64` 计算总长并对 `d.options.MaxBytes` 预检查（`native/formats/drawio/positions.go:115-143`）；再组装 `next = source[:page.rootClose] + insertion + source[page.rootClose:]`。
6. 原子提交：仅调用一次 `rewriteAndReparse(next)`（当前在 `native/formats/drawio/edit.go:197-208`）；完整 reparse 成功后才替换 document，失败必须不改 `d.source`、`d.original`、`dirty`、模型、Issues。测试断言：成功态只一次状态变化；失败态 `Serialize()` 字节与原始输入一致（参考 `native/formats/drawio/delete_cells_test.go:54-68` 的失败原子断言与 `source_retention_test.go:178+` 的失败原子覆盖）。
7. Buffer 层一致性：格式 API 完成后 graphview 再 `commitBufferSelection(serialized, true)`；这个 commit 只能由 graphview 做，不在格式 API 内完成。

## 3. 不能复用 AddShape 或现有 Serialize 作为克隆写回路径

- `AddShape` 是四种内置形状的模板化新建，不负责复制任意 start-tag 属性、未知 XML 子树或内部边；它用 `styleForShape` 归一样式并建立单节点 geometry（`native/formats/drawio/edit.go:76-116`、:237-248`）。逐个调用会产生 N 次完整 reparse、N 个 buffer 修改和 N 个 undo 项，不是一次原子提交。
- `nextCellID` 不适合作为 clone allocator 的完整依据：它只扫描 `page.Model` 中已渲染 node/edge 的 ID（`native/formats/drawio/edit.go:211-234`），未知 XML、被跳过的 wrapper、root 之外的同名字段不会入模型。
- 当前 `Serialize` 是“既有私有 source 的读出口”（`native/formats/drawio/edit.go:14-19`），不是能把模型修改编码回 XML 的通用序列化器；它不调用 `xml.Encoder`、不规范化 quote，也不编码属性。
- graphview 不能逐条调用 `AddShape` 后把结果字符串再 parse 拼目标：这样会丢失未映射的内部边、未知子树语义和选择顺序，也无法在 flush/失败后保持无部分写入。
- `rewriteAndReparse` 本身是接受“完整 next 字节”的提交口；clone 的新职责只是生成一组合法的插入补丁。补丁生成做不到位时不能用 Serialize 替代，因为 Serialize 只克隆 `d.source`。

## 4. graphview 的 Ctrl+D 行动、选择与撤销生命周期

- 快捷键绑定位于 canvas 输入路由 `handleCanvasInput` 的 `InputKeyDown` 分支（`native/modules/graphview/input.go:10-104`）：条件为 `v.kind == kindDrawio && ev.Kind == ui.InputKeyDown && ev.Mods&ui.Cmd != 0 && ev.Mods&^(ui.Cmd|ui.Shift) == 0 && ev.Key == ui.KeyD`（即 Ctrl/Cmd+D），不抢 Delete/方向键；`ui.Cmd` 是既有快捷键修饰符（对照 `native/ui/editor/editor.go:897-915`）。不接受 `Cmd+Shift+D` 或 Alt 组合，避免与将来的编辑命令竞争。
- 入口前后都使用严格 owner 快照：参照 `captureDrawioAction`/`flushDrawioAction`（`native/modules/graphview/drawio_alignment.go:13-71`）。先捕获 buffer/version/selection/releaseEpoch/pageEpoch/selectionEpoch/page/primary/ids/field/input，再检查 `host.ReadOnly`、group、composition、sync 状态；格式拒绝必须先于 ordinary-field flush，避免失败 clone 先提交一个草稿；普通字段 flush 后如版本已变化，用新 raw 重新 parse 并重算 clone（对照批删 `deleteDrawioCell` 在 `native/modules/graphview/drawio_delete.go:15-108` 的顺序）。
- 行动只在 `len(v.drawSelection.ids) >= 2` 时启用；空/单选顶点先不在最小合同里，edge-only 选择因 `drawioSelection` 成员不含 edge（`native/modules/graphview/drawio_selection.go:9-16`）自然落空且不写源。
- 成功调用格式层后，`commitBufferSelection(serialized, true)` 以 `BeginGroup → ReplaceUTF16 → SetSelection(clamped original caret) → EndGroup` 形成一个 undo/redo 项（`native/modules/graphview/edit.go:13-64`；`native/core/text/buffer.go:476-523` 证明 EndGroup 记录 before/after、Undo 恢复 before、Redo 恢复 after）。提交完成且 buffer 版本恰好 +1、owner 快照仍一致后，先取消手势，再把画布多选改为克隆出的 vertex IDs，primary 为最后一个新 ID（`setDrawioSelection` 会递增 selection epoch 并清 alignment 缓存，`drawio_selection.go:31-62`）。
- 撤销/重做生命周期：undo 把源码回滚到 clone 之前；当前 `syncDrawioSelection` 会把已经不存在的新 ID 裁剪掉，再把旧选择中仍存在的顶点组成新 IDs（`native/modules/graphview/drawio_selection.go:86-135`）；若撤销前已经另选无关联顶点，却不能指望它恢复 clone 前的完整多选。因此 clone 行动需要在 view 里登记“本事务前的旧多选 / 本事务后的新多选”与 buffer version 绑定的恢复记录：undo 完成重同步后，当且仅当 owner/epoch 快照未变、旧 IDs 全部仍在当前页时恢复旧多选；redo 后同理恢复新多选。对齐测试已展示“几何 undo/redo 后选择保留”的断言风格（`native/modules/graphview/drawio_alignment_test.go:79-123`），clone 要新增同强度断言而不是把它交给普通同步碰运气。
- 用户先点空白/换页/改选择时，selection epoch 已变化，恢复记录作废，不强制恢复任何选择。
- 成功消息放在已有 warnings/status 顶部，不虚构二次 Report；失败消息以“未克隆：...；原文仍保留，可切到源码视图继续编辑。”为壳，保持批删的说明方式（`native/modules/graphview/drawio_delete.go:30-36`）。source mode 或 readonly/compressed 页禁用动作并给只读原因；页面 issue 明确提示，不静默。

## 5. 最小成对测试与冲突面

格式层成功/失败成对（建议新增单独 clone 测试文件如 `clone_cells_test.go`，不改 lead 正改的 `delete_cells_test.go`）：

| 对 | 输入 | 断言 |
|---|---|---|
| 成功保真 | `a,b` 顶层 vertex + 内部 edge `a→b`，外层 edge 指向未选 `c`，未知属性、注释、单引号属性、未知 wrapper 均在场 | 一次调用后旧块不变；只有两个新 vertex span + 一个新 edge span；`x/y` 目标值为 `formatFloat` 输出；内部 edge 的 `id/source/target` 全部替换；外部边仍引用旧 `a`；`rootClose` 之后字节不变；`Dirty()==true` |
| ID 冲突 | 未知 wrapper 中含 `levnote-1` 或外层未渲染元素占 ID | clone 分配跳过全部已占 ID；若外层 raw 引用克隆目标且最小策略不支持则整体拒绝、字节不变 |
| 边界不全 | `a,b` + `b→c` 外连边 + `a→b` 内部边 | 只复制 `a→b`；`b→c` 保留原 endpoint 且不复制；返回的 vertex IDs 不混入 edge ID |
| Port/style 引用 | 选择集外的边 style 含 `sourcePort=a` 或选中的边含 port 键 | 按最小策略整体拒绝；`Serialize==original`、`Dirty()==false` |
| 失败原子 | readonly/compressed/重复 ID/单层限定不符/非法 geometry/缺 mxGeometry | 在改写前退出；`Serialize()==输入原文`、`Dirty()==false`、issues 不新增 |
| reparse 原子 | 构造注入后重解析必败的中间态（测试用内部钩子或重复 span） | `rewriteAndReparse` 失败不替换 document，caller 只得到错误，buffer 不变化 |

graphview 成功/失败成对（建议新增 `native/modules/graphview/drawio_multi_clone_test.go`，沿用 `newMultiDrawio/selectMultiNodes` 模式）：

| 对 | 输入 | 断言 |
|---|---|---|
| 一键成功 | Ctrl 多选 a,b，buffer 有 caret，按 `tester.Key(ui.Cmd, ui.KeyD)` | changed/version 只增 1；新 XML 存在；原 XML 完整；画布选择为两个新 vertex；undo 完整恢复旧 XML/caret/旧多选；redo 完整恢复新 XML/caret/新多选；历史只有一项 |
| 草稿原子 | 先打开属性草稿再按 Ctrl+D | 格式拒绝时源/version/selection 不变，草稿未写入也未丢历史；成功路径 flush receipt 恰 0/1 次版本变化，随后用新 raw 重算 clone |
| 禁用路径 | 单选 / edge-only / readonly / composition active / group active / syncFailed | key 被消费但不写字，提示原因；不虚构成功，不清原选择 |
| 外部字面量保真 | fixture 含外部 edge、自定义属性、注释 | 成功后这些字面量逐项仍在；clone 后 edge 数量与 endpoint 引用符合第 1 节合同 |

潜在冲突/协调点：

1. `delete.go` 正处于 lead 修改中；clone 可复用其 raw token 思路，但不要直接改 `deleteCellRanges` 的语义或把它改名成公共函数。若最终需要共享，应先提取一个通用 direct-mxCell token scanner，再由 delete/clone 各自维护输出与验证；这属于另行实现任务。
2. `positions.go` 的严格位置白名单（`validatePositionStyle`/`validatePositionCell`）偏窄，适合对齐/移动但会误拒合法的可显示样式；clone 的最小合同把 style 当不透明数据复制，只拒绝 `Unsupported` 标记与非法 geometry 形态。
3. `Page.Model.Edges` 不保证覆盖所有同名 ID 的原始元素；ID 分配必须另做全 raw 扫描。
4. `Serialize` 永远只克隆 `d.source`；whole-buffer 替换仍由 graphview 的 buffer transaction 完成，格式层不直接碰 text.Buffer。
5. 本次设计不含跨页 clone、剪贴板序列化、group/parent clone、edge-only/mixed selection、外部边重连、edge waypoint 偏移或样式语义复制；这些都应在后续独立任务中扩合同。
6. 本批次未运行任何测试，未改生产源码，上述文件与行号仅是后续实现的设计依据；实现任务的定向 gate 与最终全量验收由 lead 另行指定，不在此推断脚本位置或追加参数。
