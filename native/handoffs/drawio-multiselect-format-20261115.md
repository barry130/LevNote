# draw.io 多选位置格式实现交接 · 2026-11-15

## 状态与写域

三 API、六常量和公开接口回归已实现。lead 的 format01 发现既有自闭合 mxPoint 解析缺陷后，已按追加授权窄修 [parse.go](<F:/AIproject/LevNote/native/formats/drawio/parse.go#L906>) 三行并新增独立公开 [点解析回归](<F:/AIproject/LevNote/native/formats/drawio/positions_parse_points_test.go>)，代码再次冻结交 format02。原新增 [positions.go](<F:/AIproject/LevNote/native/formats/drawio/positions.go>)、[positions_test.go](<F:/AIproject/LevNote/native/formats/drawio/positions_test.go>) SHA 保持，原 offset 负资格案例保留；类型、单节点编辑及旧测试均未修改。无 commit/push、子委派或共享记忆写入。

根合同为 [drawio-multiselect-plan-20261115.md](<F:/AIproject/LevNote/native/handoffs/drawio-multiselect-plan-20261115.md>)；格式资格与原子补丁按已审 [feature-parity-drawio-next-20261115.md](<F:/AIproject/LevNote/native/handoffs/feature-parity-drawio-next-20261115.md>) 落地。

## 冻结接口

```go
type PositionUpdate struct { CellID string; X, Y float64 }
type Alignment uint8
// iota 0..5: AlignLeft, AlignCenterX, AlignRight,
//           AlignTop, AlignCenterY, AlignBottom
func (d *Document) ValidatePositionTargets(pageIndex int, cellIDs []string) error
func (d *Document) PlanAlignment(pageIndex int, cellIDs []string, alignment Alignment) ([]PositionUpdate, error)
func (d *Document) UpdatePositions(pageIndex int, updates []PositionUpdate) error
```

Validate 至少一个唯一 ID；Plan 至少两个，输出仅含坐标发生变化的节点并保持请求顺序。Plan 不修改对象，已对齐返回空列表。Update 的空列表在有效可写页面是 no-op；无效页、nil Document、只读页仍保留对应错误。非空 Update 重新验证全部目标与坐标再构造补丁。错误包装既有 sentinel，支持 errors.Is，并包含相关节点/属性/样式原因。

## 实现语义

- 原始资格扫描从私有完整 source 开始，到所选 model 结束，保留祖先 namespace 上下文。导出的 Source、Nodes 数值与 Issue 回退值不作为写入事实，其他页的压缩内容不在预检中解压。
- canonical 0/1 容器按真实 XML 的精确非命名空间属性、顺序、空内容检查。仅接纳 root 直接子项、parent=1、vertex、无子项的绝对叶节点；同页隐藏 wrapper 内重复 ID/子项也参与资格检查。
- 每个目标须有且仅有一个直接、非命名空间 mxGeometry，as=geometry，有限且正的宽高。x/y 缺省为零，显式空值、无效数值、NaN/Inf、端点溢出均报错。relative、变换属性和 geometry 子元素阻止该批操作。
- cell/geometry 关键属性须非命名空间；任何 local-name 碰撞与关键名 namespace declaration 都报错。精确重复属性仍由原 Parse.acceptStart 处理，该语义保持。自闭合 mxPoint 的 token 消费修正见下文。
- 支持矩形、圆角矩形、椭圆、菱形；普通展示 style 保留。rotation=0、flipH/flipV=0 可用，组、swimlane、布局/未知命名 style、未知 shape、图片、stencil 等有具体错误。未知 style 键采用保守合同；普通 XML 扩展属性和非几何 metadata 子树可保持原样，其他节点的无效或未知内容不全局阻止该批操作。
- 对齐以选择整体外包框为基准，不吸附。所有节点端点、整体跨度和目标位置均检查有限性；输入顺序影响输出顺序，不影响结果坐标。
- 每个标签仅修改有数值变化的 x/y。未变化的 `1e2`、`100.0`、`-0` 与缺省属性继续保留；尺寸、另一轴、引号、空格、属性序、标签/边/metadata 保持字节。
- 先完整验证，再生成标签补丁、排序与区间检查、拼接一份新候选、一次 rewriteAndReparse。失败不触碰 source/original/Source/Dirty/模型；成功沿用既有 source 所有权。未调用逐节点 UpdateGeometry。

## 回归覆盖

位置操作新增 11 个顶层公开接口测试，入口为 Parse → Validate/Plan/Update → Serialize/OriginalBytes/Dirty/公开模型；JSON 快照包含导出模型和 issues，未替换内部协作者。另新增 1 个独立 drawio_test 外包测试、9 个点闭合子例，只调用公开 Parse/模型/Serialize/OriginalBytes/Dirty。

| 范围 | 独立判断依据 |
|---|---|
| 六向对齐 | a=(-10,20,30,10)、b=(40,-5,10,20)，手算左 b.x=-10；水平中心 a.x=5/b.x=15；右 a.x=20；顶 a.y=-5；垂直中心 a.y=7.5/b.y=2.5；底 b.y=10 |
| 最小轴补丁 | 完整字面 XML oracle，反向请求顺序、单/双引号、科学计数法和原宽高词法 |
| no-op | clean/dirty 两种状态，空列表、数值等值、负零、缺省 x、已对齐空计划；缺省轴仅在变化时插入 |
| 私有源所有权 | 修改 Parse 输入、导出 Source/Nodes、Serialize/OriginalBytes 返回值，计划仍取真实源且私有字节不串改 |
| 目标原子性 | 缺失 ID 位于首中末，重复/空/容器 ID、失败后合法编辑；clean/dirty 均比较全公开快照 |
| 真实 XML 资格 | 缺失/重复/包装 geometry、非法数/尺寸、相对/offset/alternateBounds/未知变换、namespace 属性/声明/元素、边/组/布局/shape、canonical 容器及 wrapper 内子项/重 ID |
| 多页保真 | 前置压缩页、编辑页、后置普通页；CRLF、XML 声明、CDATA、注释、未知 metadata、未选中异常节点与边关系保持完整字节 |
| 候选失败 | MaxBytes 增长失败；增加缺省轴触发 MaxAttributes 的真实候选重解析失败，随后编辑证明旧偏移仍有效 |
| 数值/参数边界 | NaN、正负 Inf、新目标端点溢出、选区跨度溢出；无效页、nil Document、无效枚举、短/空选择 |

精确重复 x/y/id 属性另用公开 Parse 验证既有 ErrInvalidXML；可被原 Parse 接受的 namespaced 同名和重复 geometry 则由三新 API 验证整批零变更。

## 格式化、身份与执行边界

本 agent 共执行四次固定工具链的 gofmt stdout，均 foreground exit 0、stderr 为空，并通过文件工具写回完整输出。Go test/vet/build 和产品进程均由 lead 统一；本 agent 未执行。lead 已保存 format01 真实 RED，窄修后的行为结果待 format02。

```text
.localtools/go1.27.3/bin/gofmt.exe native/formats/drawio/positions.go
.localtools/go1.27.3/bin/gofmt.exe native/formats/drawio/positions_test.go
.localtools/go1.27.3/bin/gofmt.exe native/formats/drawio/parse.go
.localtools/go1.27.3/bin/gofmt.exe native/formats/drawio/positions_parse_points_test.go
```

| 文件 | 字节 / 行 | SHA-256 |
|---|---:|---|
| [positions.go](<F:/AIproject/LevNote/native/formats/drawio/positions.go>) | 17647 / 501 | 8fdbe1a5fb5b16f09812980540b61a77e5a9517cbf3433745a8d42fcf8435b10 |
| [positions_test.go](<F:/AIproject/LevNote/native/formats/drawio/positions_test.go>) | 23858 / 482 | 06bd37a01002a401646cbb037f0a26bc3ca96f62558bfd4e322d0583c88fac14 |
| [parse.go](<F:/AIproject/LevNote/native/formats/drawio/parse.go>) | 36353 / 1182 | 7b5c6d693017bdea4b9c3b7b238332a800cb45c9680ad575662f1d188765df0a |
| [positions_parse_points_test.go](<F:/AIproject/LevNote/native/formats/drawio/positions_parse_points_test.go>) | 3917 / 75 | 98943d18ed278b20a49b0dc04ba2aa248e0ec346d7ea83af930b4f86627e1adf |

资格预检当前是一次 source 前缀扫描，并非索引缓存；消费者应在动作/选择改变时调用，避免 paint 对每个节点反复扫描。本轮没有性能基准或 UI/物理输入证据。格式文件已向 lead、gap-graph、gap-editor 发出稳定冻结通知；后续源修改先通知 lead，避免与检查并写。

## format01 RED 与自闭合 mxPoint 窄修

已实际读取 [format01 result](<F:/AIproject/LevNote/.localtools/drawio-multiselect-20261115-format-01/result.json>) 全 30 行及 [失败日志](<F:/AIproject/LevNote/.localtools/drawio-multiselect-20261115-format-01/stdout.jsonl#L370>)：162 RUN / 160 PASS / 2 FAIL；两个失败身份是 offset-point 子例及其父项；22 个相关输入、435 个全量输入前后相同，exit 1。原始证据保持。

根因：encoding/xml 对 `<mxPoint .../>` 仍返回匹配 EndElement；旧分支直接 continue，使该 token 落入 mxGeometry 的结束分支，报 `invalid draw.io XML: mismatched mxGeometry end element`。修正只在该分支调用既有 consumeEmptyEnd 并透传错误；git diff 确认为新增三行，其余 parser 语义保持。

新增正向回归用独立字面坐标 source=(12.5,-7.25)、target=(120,90)、offset=(-3,4.5) 验证各角色的自闭合/显式闭合六例，以及相邻全自闭合/全显式/混合三例；逐例检查后续节点 (1,2,3,4) 的同步、issues 为空、公开点模型、未 dirty、Serialize/OriginalBytes/Source 与完整输入逐字节相同。offset 沿既有通用 Points 表示，不扩展角色语义。原位置测试的 offset-point 负资格 fixture 字节和测试身份均保持。

再次冻结后由 lead 执行 format02；此处记录实现及已知 RED，不提前声明 GREEN。
