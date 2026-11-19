# 原生格式与插件 API

本页对应 `native/` 的 Go API。`formats/` 为纯数据解析、修改、序列化和导出
层，不依赖 MyGo/UI，不直接读文件。宿主负责打开字节、持有文档缓冲、处理
失败和发布导出文件。视图与输入契约见 [架构说明](../ARCHITECTURE.md)。

## 包与坐标

| 包 | 职责 | 坐标 |
|---|---|---|
| `formats/markdown` | 实用 Markdown 子集、警告、HTML 导出 | `SourceRange{Start, End}` 是原始 UTF-8 源码的半开字节范围 |
| `formats/code` | 内置语言与声明式规则的词法高亮 | `Span.Start/End` 是传入字符串的半开字节范围 |
| `formats/mindmap` | JSON 树、结构编辑、布局、展示规则、序列化与有界SVG | 节点稳定 ID；布局为文档坐标 |
| `formats/drawio` | XML 多页模型、安全字段修改、SVG | 页面和 cell ID；几何为文档坐标 |
| `formats/sourceoutline` | 有界的字面物理行扫描 | Entry.Start/End 是整条物理行的半开 UTF-16 范围（含终止 CR/LF） |
| `core/text` | 缓冲、选区、撤销、版本和范围映射 | 编辑 API 使用 UTF-16 单位；不能直接传格式层字节偏移 |
| `core/plugins` | 有序匹配、JSON高亮/大纲/固定原生格式选择、启停偏好 | 路径扩展名与配置 ID，不等同适配器 ID |
| `core/enginecatalog` | 四个固定引擎的ID、格式、平台、路径与预算 | 按值返回Descriptor，无文件/注册/插件状态 |
| `core/engineipc` | 单请求有界协议、严格元数据与原始字节载荷 | `Warning.Start/End` 为提交快照的UTF-8字节范围 |
| `core/engineruntime` | 固定随包引擎验证及进程生命周期 | 不接收源文件路径；输入为不可变`io.WriterTo`与字节数 |
| `core/enginescene` | 无格式/UI依赖的中性只读场景编解码 | 有限float64文档几何、opaque节点ID；没有源码坐标或宿主身份 |

## 固定 HTML、导图 SVG与draw.io SVG引擎

本节是当前已验收的仓库内实现接口，不是任意可执行插件安装协议。普通
schema1/2 JSON安装配置依然只有源码语法/大纲数据，schema3只选择三个已知
原生适配器；三者均不能指定可执行命令或路径。
可选适配器元数据位于`ui/editor/contract.go`：

```go
type ExportEngine struct {
    ID, Format string
    RequiresView bool
}
// Adapter.ExportEngine *ExportEngine
// Markdown：ID="levnote.markdown-export", Format="html", RequiresView=false。
// Mindmap：ID="levnote.mindmap-svg", Format="svg", RequiresView=false。
// draw.io：ID="levnote.drawio-svg", Format="svg", RequiresView=true。
```

宿主只按能力分派。Markdown/导图在source/noView也可调用；draw.io要求
阅读/编辑或分屏中的已选页面，源码入口如实禁用。启用此能力时失败不回退
到`View.Export`。Markdown worker调用纯`Parse/ExportHTMLBytes`，原字符串
`ExportHTMLChecked`保留；Mindmap worker调用纯`Parse/Layout/ExportSVG`；
draw.io worker调用纯`Parse/ExportSVG`。可编辑导图、语义大纲解析和
预览仍在宿主；只读导图场景接口见下节。停用配置后按剩余配置重匹配，
源码兜底不带执行能力。

```go
// core/enginecatalog；EngineID是string定义类型，engineipc.EngineID是其别名。
const MarkdownHTML EngineID = "levnote.markdown-export"
const MindmapSVG EngineID = "levnote.mindmap-svg"
const MindmapPreview EngineID = "levnote.mindmap-preview"
const DrawioSVG EngineID = "levnote.drawio-svg"
func Lookup(engine EngineID) (Descriptor, bool)
func Match(engine EngineID, format, goos, goarch string) (Descriptor, bool)
```

`Descriptor`按值携带Engine/Format/Protocol/RelativePath/Filename、GOOS/GOARCH
与SourceMaxBytes/PayloadMaxBytes；固定为Windows/amd64。没有注册或调用方
自定义描述符入口，返回副本的修改不会改变catalog。

```go
type Source struct {
    Size int64
    Data io.WriterTo // Data必须不可变
    Page *engineipc.RequestPage // draw.io必须提供页对象；旧三引擎为nil
}
type RequestPage struct { Index int; ID string } // ID仅在所选源页真实无ID时为空
type EngineInfo struct {
    ID engineipc.EngineID
    Version, SHA256 string
    Source PackageSource
    PackageDigest string
}
type Result struct { Payload []byte; Warnings []engineipc.Warning; Engine EngineInfo }
func NewBundled() (*Runner, error)
func NewManaged(manager *PackageManager) (*Runner, error)
func (*Runner) Export(context.Context, engineipc.EngineID, Source) (Result, error)
func (*Runner) Close() error
```

四个引擎共用一个runner任务槽，busy直接拒绝。Close取消、停止新任务并等待
所有已启动的所属进程及管道/写入协程，不依赖UI派发；重复Close保留同一
清理结果。每次成功返回前完成清理，所有错误Payload为nil，没有HTML别名。
成功验证包之后的后续失败仍带`Result.Engine`，验证失败前为零值。`Failure.Kind`区分
busy/closed/unavailable/integrity/source-limit/source-size/protocol/stdout-limit/
stderr-limit/worker-exit/unexpected-stderr/engine/canceled/timeout/cleanup；
`Code`可带有界worker错误码，`Detail`不包含原始stderr或文档内容。

实际宿主目录内固定布局如下，四处各自带`manifest.json`：

| ID / format | 相对目录 / 程序 | 源 / 输出预算 |
| --- | --- | --- |
| `levnote.markdown-export` / `html` | `engines/markdown-export/levnote-markdown-export.exe` | 8MiB / 64MiB |
| `levnote.mindmap-svg` / `svg` | `engines/mindmap-svg/levnote-mindmap-svg.exe` | 16MiB / 64MiB |
| `levnote.mindmap-preview` / `mindmap-scene-v1` | `engines/mindmap-preview/levnote-mindmap-preview.exe` | 16MiB / 32MiB |
| `levnote.drawio-svg` / `svg` | `engines/drawio-svg/levnote-drawio-svg.exe` | 32MiB / 64MiB |

清单最多16KiB，含schema1、固定ID/format、protocol1、Windows/amd64、
文件名、1..64ASCII可见字符版本、大小和SHA256。
runtime验证目录/常规文件句柄并拒绝reparse，持锁覆盖校验/启动/运行。
验证以请求ID对应的catalog为准，不能借另一引擎的清单/路径通过。安装入口
只接受`PackageManager`中的固定catalog包；bundled请求没有额外安装入口，
installed请求只接受已显式选中的已验证包digest。没有可配置shell；
SHA一致性不是签名或沙箱。

stdin/stdout各只有一帧：8字节magic`LVEXREQ\0`/`LVEXRES\0`、uint16版本1、
uint16头长24、uint32元数据字节数、uint64载荷字节数，全为小端；随后
精确JSON元数据、原始载荷、EOF。先核对长度再分配，成功还要求exit0。
JSON拒绝重复/未知/大小写不同键、null必需数组/字段、错误类型、无效UTF8/
surrogate或尾随内容。源按catalog为8/16/32MiB，request meta4KiB；draw.io
页面索引为0..249,999、页面ID最多512字节；导出输出64MiB、场景32MiB，response
meta1MiB；警告最多256，code64B/message512B，范围必须位于提交源内；
错误message1024B，stderr16KiB。任何不完整或额外输出均不得发布部分结果。

```go
// core/engineipc
func ReadRequest(r io.Reader) (RequestMeta, []byte, error)
func ReadRequestFor(expected EngineID, r io.Reader) (RequestMeta, []byte, error)
```

`ReadRequest(r)`保持Markdown包装器；`ReadRequestFor(expected,r)`在读取
载荷前按期望ID核对身份和源预算。既有请求编码/响应写入/增量解码器支持
上述四个固定身份，响应必须匹配请求ID、format和已验证的清单版本。
历史`SourceMaxBytes/HTMLMaxBytes`仍是Markdown别名，不用于推断导图预算。

传入的是文档层解码后的精确UTF8 Buffer快照；文件编码BOM已由文档层
处理，worker不再去BOM或转码。Buffer中的字面U+FEFF是内容。worker只用
继承stdio与Exit，不接路径、读取文件、解析外部引用、联网或派生子进程。
错误码为invalid-request/source-limit/invalid-utf8/output-limit/warning-limit/
render-failed/partial-document（最后一项为Markdown）。导图解析错误或无效
schema为render-failed，解析限额为source-limit。导图警告按解析、布局、展示
顺序返回，各项范围均为`[0,len(source))`；数量或字段大小超限为warning-limit，
不截断、漏报或伪造节点源码位置。当前HTML/SVG生成器先在64MiB预算内计数，再
一次分配精确结果并填充；output-limit无部分载荷/警告。它不是进程峰值
内存或流式保证；历史`.1`仅在完整字符串构建后检查。30秒全请求超时和Windows Job清理详见
[架构](../ARCHITECTURE.md#一次性-html-导图-svg与drawio当前页导出进程)。

app负责Flush、版本/来源身份、模态对话框延后与不覆盖目标的最终发布；
worker不接收文件路径，也不调用文件发布接口，但仍有普通用户的OS权限。
最终WriteNewFile同步执行，开始后不能取消。
当前`.21`交付、历史包与失败修正证据见[验证记录](../VALIDATION.md)。

## 本地引擎包管理

```go
type PackageSource string
const PackageBundled PackageSource = "bundled"
const PackageInstalled PackageSource = "installed"

type PackageFile struct { Path string; Size int64; SHA256 string }
type PackageReceipt struct {
    Engine enginecatalog.EngineID
    Digest, Version string
    Files []PackageFile
}
type PackageStatus struct {
    Engine enginecatalog.EngineID
    Source PackageSource
    Version, Digest string
}

func NewPackageManager(hostDir, stateDir string) (*PackageManager, error)
func (m *PackageManager) ImportFromReleaseDir(ctx context.Context, releaseDir string, engine enginecatalog.EngineID) (PackageReceipt, error)
func (m *PackageManager) Installed(ctx context.Context, engine enginecatalog.EngineID) ([]PackageReceipt, error)
func (m *PackageManager) Activate(ctx context.Context, engine enginecatalog.EngineID, digest string) error
func (m *PackageManager) Reset(ctx context.Context, engine enginecatalog.EngineID) error
func (m *PackageManager) RepairResetAll(ctx context.Context) error
func (m *PackageManager) Snapshot() []PackageStatus
func (m *PackageManager) LoadError() error
```

`NewPackageManager`的构造参数是宿主/状态目录；管理操作只接受固定
catalog的四个引擎ID。已安装包根为
`stateDir/engine-packages`，每引擎最多12个版本。`ImportFromReleaseDir`
复制并验证完整本地发布目录中的对应引擎，不启用；重复digest会重新验证
既有安装并原样复用，不新增、不覆盖。`Installed`列出当前可验证的包；
单个坏包/缺失包返回问题，健康包继续可用。`Activate`、`Reset`和
`RepairResetAll`分别是显式启用、恢复随包和显式修复全部选择。

`Snapshot()`返回选择快照：installed项带版本与包digest，bundled项不预填
版本/摘要；实际版本和摘要在运行时验证后出现在`Result.Engine`中。选择
状态文件损坏时，执行与普通转换拒绝，列表/导入仍可用于诊断。单个已安装
包损坏时，被选中的包拒绝执行且不回退，健康替代包和显式恢复随包仍可用。
schema1旧选择文件按原字节读取旧的三个引擎选择，draw.io默认随包；
显式启用、重置或修复全部选择后写入schema2的四个精确选择项。
选择转换由app层负责join旧工作、处理poison与UI状态；格式包和runtime
API没有任意可执行路径或命令覆盖，也没有引擎插件API。SHA/大小一致性是本地完整性检查，
不是发布者认证或沙箱。

## 中性场景与只读预览能力

`core/enginescene`只携带展示值，公开API如下；不加入worker身份、源码或
文件/图片路径，不保留调用方输入的可变别名：

```go
type Point struct { X, Y float64 }
type Rect struct { X, Y, Width, Height float64 }
type Node struct {
    ID, Label string
    X, Y, Width, Height float64
    Shape string
    Fill, Stroke, Text string
    Root, Collapsed bool
}
type Edge struct {
    FromID, ToID string
    From, To Point
    Stroke string
}
type Scene struct {
    Schema int
    Layout string
    Bounds Rect
    Nodes []Node
    Edges []Edge
}
func Encode(Scene) ([]byte, error)
func Decode([]byte) (Scene, error)
```

schema1固定字段，JSON含nodeCount/edgeCount，解码可接受任意字段顺序，
但拒绝重复/未知/大小写变体/缺项/null、错误类型、无效UTF8/surrogate与
尾随内容。计数按词法验证精确整数，不能让float64舍入将小数变成合法
计数；长指数和系数有界处理。树要求唯一ID、恰好一个根、n−1边、有效
父节点及全连通，使用迭代验证。paint仅空主题角色或小写`#rrggbbaa`，
shape为`rounded`；布局限四个已支持名称。坐标绝对值≤1e9，正尺寸≤1e7，
有限浮点；不能下溢为零尺寸。

固定上限：wire32MiB、100000节点/99999边、ID512B、Label515B、Layout64B。
载荷/字符串/数组在超过相应预算时立即拒绝，不靠声明计数预分配。
`ErrInvalidScene`与`ErrLimitExceeded`支持errors.Is；任何错误返回nil编码
字节或零Scene。Encode共享计数/填充器、一次精确len==cap分配，无公开
自定义预算入口；不承诺总进程内存，调用者不可并发修改输入Scene。

`cmd/levnote-mindmap-preview`使用`ReadRequestFor(MindmapPreview,r)`，按
16MiB解析/原生布局后映射到DTO。共享有界标签512字节前缀加可选省略号，
字面颜色规范化，空颜色由宿主主题填充；不传主题默认、完整字段或图片。
图片warning保留code/order/整篇范围，文案明确此预览不加载或显示图片；
SVG图片占位另属原接口。数量/字段超限不静默截断，错误零载荷。

`ui/editor/scene.go`与Host的中性能力为：

```go
type SceneCapability struct { ID, Format string }
type SceneRequest struct { Capability SceneCapability; Epoch uint64 }
type SceneWarning struct { Code, Message string; Start, End uint32 }
type SceneEngineInfo struct { ID, Version, SHA256 string }
type SceneResult struct {
    Scene enginescene.Scene
    Warnings []SceneWarning
    Engine SceneEngineInfo
}
// Adapter.SceneEngine *SceneCapability
// Host.SceneEngine *SceneCapability
// Host.BeginScene func(owner View, request SceneRequest,
//     complete func(SceneResult, error)) (cancel func(), err error)
// Host.ShowSource func(owner View) bool
```

BeginScene/cancel只在UI线程使用且不阻塞；异步完成至多一次，先取消则
不再回调。宿主验证owner/能力、只读状态与全部来源身份，捕获Buffer快照，
后台运行/解码；先取走并清空任务槽，再把结果所有权交给仍匹配的视图。
视图不提供源码。
ShowSource核对owner，切换不会解除只读。场景只显示文字/颜色/几何，完整
属性在源码查看，另存可写副本后使用编辑面板。

app共享单执行队列，至多一个最新待处理预览；导出优先并占用至发布结束，
被抢占的有效预览随后恢复。取消/Release丢弃旧引用、快照和迟到结果，清理
失败仍报告；Close不依赖UI派发。固定worker失效可切源码，绝不宿主解析
回退。schema3选择Mindmap的自定义后缀使用相同能力；schema1/2仍只有
源码高亮/大纲，任意可执行/交互编辑引擎安装未交付。实际边界和证据见
[架构](../ARCHITECTURE.md#只读导图场景与共享队列)。

## Markdown

```go
doc := markdown.Parse(source)
// doc.Raw() 与 source 逐字节相同。
// doc.Blocks、doc.Warnings 是解析结果；doc.Partial 表示解释被预算截断。
html, warnings, err := markdown.ExportHTMLChecked(doc)
if err != nil {
    // 向用户显示 warnings，保留完整源码；不要发布 html。
    return err
}
// 由宿主处理 warnings，再以“不覆盖已有文件”的写入入口发布 html。
```

实际常量是 `BlockHeading`、`BlockParagraph`、`BlockCode`、`BlockList`、
`BlockQuote`、`BlockHTML`、`BlockDivider`、`BlockDocBlock`。块的 `Range`
包含语法，`TextRange` 指向可编辑正文；`Level`、`Language`、`Ordered`、
`Start`、`Items`、`DocBlock` 都是 `Block` 的直接字段，没有单独的
包围这些字段的 `Heading` 或 `Code` 子对象。

行内常量是 `InlineBold`、`InlineItalic`、`InlineCode`、`InlineLink`。
`InlineSpan` 含 `Kind/Range/Text/Href`。解析器识别的是有限子集；不支持
的语法保留在原文中并按可见文本和警告处理，不以重新序列化偷偷修正原文。
`DocBlockRef` 保留属性、额外键、载荷和来源范围，不自行加载引用文件。
缺少必需 `---` 分隔符时返回含完整原文的 `BlockCode` 并发出警告，
缺失结束围栏也会警告；不会访问不存在的下一行或形成重叠块范围。
有结束围栏时下一块从围栏后的物理行开始，合法 doc-block 后的正文同样
继续解析。这是错误处理和范围修正，没有扩展 doc-block 语法。

`DocBlockRef.HeaderRange` 是从开围栏整行之后起、到 `---` 分隔行首之前的
精确半开 UTF-8 字节范围，不包含分隔符或载荷。`Attrs` 只识别小写
规范键 `id`、`kind`、`mode`、`src`、`title`；输入键名先归一化为小写，
值沿用现有 `TrimSpace` 规则。未知属性保留在 `Extra`，不要把它当成未
裁剪的原始目标字节。

`ReferenceProblem` 为空**只**表示已识别的引用头结构可用，不承诺宿主支持、
本地目标存在、允许读取/打开、目标长度或只读状态。引用模式缺少/清空
`src`、重复识别键、缺失或无效 `id`、缺失 `kind`、缺失/未知 `mode`、
inline 模式带 `src`、未闭合围栏都会给出有界诊断，同时保留已解析值和完整
原文。普通 `mode: inline` 且没有 `src` 可以有空的
`ReferenceProblem`，但它仍不是可导航引用。引用导航还要求显式
`mode: reference` 与非空 `src`；缺失分隔符仍是原始代码回退，不构造引用。
相同 `id`/`title` 没有全局唯一约束，引用身份是精确源码范围，不是这些字段
或列表下标。

```go
type TaskMarker struct {
    Range SourceRange
    Checked bool
}
// ListItem.Task *TaskMarker；nil 表示普通列表项。
```

任务仅在现有简单无序/有序列表正文开头识别：`[ ]`、`[x]`、`[X]` 后接
ASCII 空格、tab 或已裁去行尾空白的正文结尾。`Task.Range` 包含原始三个
ASCII 字节；Checked 对 x/X 为 true。任务的 Text/TextRange/Inline 是
标记及紧随的空格/tab 之后的标签；空标签范围有效且长度为零。整项 Range
和 doc.Raw 保留原编号、行尾和尾随空白。普通列表项语义不变，缩进/嵌套、
围栏、引用及 `[x]word` 不借此扩展为任务。

HTML 任务输出 `<input type="checkbox" disabled>`，完成项再加 `checked`，
非空格式化标签前加一个空格；空任务仅输出方框。不含脚本、事件或网络
资源。原生控件切换须先 Flush，再按 Buffer/版本映射整项和标记 UTF-16
范围，并以当前完整解析重新确认任务上下文；不得由标签或下标定位。
仅把标记中间一字符在空格与小写 x 之间切换，用一个新事务保留源选区
方向，成功后 Changed 一次。只读/Partial/超限或失效目标拒绝；原生控件
的可访问标签/tooltip 只复制最多128 UTF-8 字节的原标签及省略号。

Markdown 管道表第一切片增加以下纯数据元数据：

```go
const BlockTable BlockKind = "table"

type TableAlignment uint8
const (
    TableAlignDefault TableAlignment = iota
    TableAlignLeft
    TableAlignCenter
    TableAlignRight
)
type TableCellRef struct {
    Range, TextRange SourceRange
}
type TableRowRef struct {
    Range SourceRange
    Cells []TableCellRef
}
type TableRef struct {
    Rows []TableRowRef
    DelimiterRange SourceRange
    Alignments []TableAlignment
    Problem string
}
// Block.Table TableRef
```

所有范围都是原始 UTF-8 字节的半开区间。`Rows` 第一项是表头，分隔行
不进入显示行；`TableCellRef.Range` 包含原 ASCII 空格/tab 填充，
`TextRange` 去掉两端同一填充。有效表的 `Problem` 为空且对齐项数与列数
一致；识别到但结构或预算失败的连续候选仍是一个 `BlockTable`，保留完整
`Raw`、有界 `Problem` 和不构造行列。表头加分隔行、零正文行有效。

声明子集为 1–8 列；允许每行独立可选外管道、最多三个前导 ASCII 空格/tab、
空单元格和奇偶反斜杠转义。反引号不保护未转义管道。上限为 256 个正文行、
每单元格 `TextRange` 4096 UTF-8 字节、完整表 `Raw` 32 KiB、每文档累计
16,384 个 cell 描述项。预算避免无界描述项保留；完整 `Raw` 有意作为
源码回退保留，超过 32 KiB 时视图只给有界源码入口而不物化整张预览。

有效表可导出语义 `<table>/<thead>/<tbody>/<tr>/<th>/<td>`。单元格只输出
HTML 转义的 `TextRange` 字面文字，只把固定对齐枚举转换为 `text-align`；
不解释其中的 Markdown 转义、链接、图片或 HTML。无效/超限表把完整 `Raw`
转义放进 `<pre>`；描述项损坏时也先验证范围/计数/枚举再写表格标记。
`ExportHTMLChecked` 对 `Partial` 仍返回空内容与错误。

第一切片已通过格式、原生视图和真实注册适配器聚焦门，并在冻结源码上完成
一次全量 `test ./...`（20/20包摘要通过）、全量vet、去符号GUI构建、
隔离隐藏启动关闭和12/12证据核验；根代理已接受20:49产物13,384,192字节，
SHA-256
`7632B1C50D9536B25A12CF9FA7E288B62F903A3B4ABA9896EF8849356864BBCD`。
该产物现为上一验收产物。精确产物与边界见
[验证记录](../VALIDATION.md)与
[应用交接](../handoffs/markdown-tables-app.md)。

第二切片的单单元格编辑不增加纯格式API：视图继续使用上述
`TableCellRef.TextRange`、完整`BlockTable.Range`和原始UTF-8范围，先以
当前完整解析核对候选，再由宿主对精确UTF-16范围执行一次写回。单格草稿
最大4096 UTF-8字节，拒绝换行、未转义结构管道、候选表超过32 KiB或文档
解析上限，以及行列数/对齐等表形变化。无改动完成不写宿主。该草稿属于
视图本地状态，不进入自动会话/崩溃快照；无效草稿通过现有恢复回调显式
打开为未同步副本。真实应用聚焦门、最终视图检查、全量test/vet/build、
12/12证据核验和隔离隐藏启动关闭均已通过；根代理已接受13,425,152字节
单元格发布产物。

第三切片提供纯数据行补丁规划，不修改传入源码或文档：

```go
type TableRowAction uint8
const (
    TableRowInsertAfter TableRowAction = iota + 1
    TableRowDelete
)
type TableRowPatch struct {
    Range SourceRange // 完整 source 中的 UTF-8 半开字节范围
    Text string
}
func PlanTableRowEdit(source string, tableRange, rowRange SourceRange,
    action TableRowAction) (TableRowPatch, error)
```

输入范围是身份，不信任外部描述项，也不按文字查找同名行。函数重新解析
权威源码并要求同一受支持表和物理行；表头插入锚定分隔行之后，表头删除
返回`ErrTableRowHeader`。插入生成相同列数的空字面行（两列为`| | |`），
保留选中行缩进和锚定行LF/CRLF/CR；最后一行无行尾时在原尾与新行间添加
最近的表行尾，而新末行继续无行尾。删除只覆盖数据物理行及其自己的行尾。
补丁外的源码保持逐字节一致，调用方负责转UTF-16并执行事务。

候选沿用8MiB源、8列、256正文行、4096字节格、32KiB表及全篇描述项等
限制，并完整解析比较其他块/表的描述状态。全篇16,384cell预算可能让后表
回退但不设置`Partial`，因此不能只检查`Partial`。删除释放预算而让另一
回退表恢复可解析时也保守拒绝`ErrTableRowContext`。其他可判定错误是
`ErrTableRowTarget`、`ErrTableRowAction`、`ErrTableRowLimit`；失败补丁为零值。

原生调用还在pointer/key down捕获Buffer、草稿/单元格对象、版本、选区与
生命周期。自身单元格Flush仅接受已证明的0/1版本结果及预期文本/选区，
同步回调改源/选区、替换或Release都拒绝。成功单元格写回与行补丁是两条
独立撤销；后者与映射后的有方向选区在同一事务提交。若前者已成功而后者
拒绝，会报告该区别，不声称单元格尚未写入或仍有失败草稿。未成功写回的
草稿仍可显式恢复。行操作的`.3`交付及实际证据见[行操作交接](../handoffs/markdown-table-rows.md)。

所选列对齐提供另一纯补丁规划接口，复用既有`TableAlignment`枚举：

```go
type TableAlignmentPatch struct {
    Range SourceRange // 权威 source 中 UTF-8 半开字节范围
    Text string
    Noop bool
}
func PlanTableAlignmentEdit(source string, tableRange SourceRange,
    column int, alignment TableAlignment) (TableAlignmentPatch, error)
```

`column`从0起；仅接受`TableAlignDefault/Left/Center/Right`。重解完整源码，
要求精确同一受支持表，再复用原分隔行scanner与对齐判据。相同语义返回
`TableAlignmentPatch{Noop:true}`，Range/Text保持零值，不改不规范但合法的
原始拼写，也不构造候选。真实修改只替换分隔格的非填充`TextRange`：保留
原短横线数、调整两端冒号，补丁以外所有字节保持。调用方负责转UTF-16和事务。

无效枚举为`ErrTableAlignmentValue`，源UTF-8/范围/列或受支持表身份不符为
`ErrTableAlignmentTarget`；解析/增长预算问题为`ErrTableAlignmentLimit`，
影响其它解析语义为`ErrTableAlignmentContext`。错误返回零补丁。正增长
在新标记/候选分配前检查4096字节分隔格、32KiB表和8MiB源；现有8列/
256正文行/16,384描述项预算不变。候选完整解析核对所选表形状/值/对齐、
其它块语义/表回退状态、警告及范围平移，不能只看Partial。

原生行/对齐共享`table_actions.go`生命周期，捕获列/原对齐、source/draft
版本/选区与Release世代。自身Flush仅接受可证明的0/1版本receipt；成功
与Noop都完成单格面板。干净Noop无source/version/history/Changed；先写
单元格后Noop只保留单元格编辑，真实对齐另记撤销。回调释放/替换/原位
改源或输入不能被旧local error覆盖，也不能重造旧draft。源码选区在替换
标记内时移到标记起点，其后按delta平移，保留方向。实际`.5`交付、独立
源码审查、真实适配器/窄窗/包证据见[列对齐交接](../handoffs/markdown-table-alignment.md)。
无新表格语法、列插入/删除、排序或合并API。

HTML 导出转义正文和属性；原始 HTML、doc-block 以文本输出，不执行脚本。
链接仅允许相对路径、锚点和 `mailto:`，其他链接保留文字并报告警告。
这与应用打开引用的策略不同：应用不启动邮件程序，也不打开外部 URL。

原生视图的 `InlineLink` 通过手动构建的 `RoleLink` 行内文本和
`Host.OpenReference` 激活，不调用工具包的外部 URL 控件。Ctrl/Cmd+主键
点击或 Tab+Enter 激活；普通选择/双击仍交给段落，Space 不激活。原生
可访问名为 `链接：<标签>`，空标签用 `（空链接）`；只读允许导航。
手势捕获 Buffer/版本、包围块或列表项与完整行内语法的 UTF-16 范围，先
Flush，再按日志和当前解析核对同一完整 href，不用标签/下标重定位。

完整目标副本最多4096 UTF-8 字节，超出时明确禁用而不截断地址；名称和
目标提示分别最多128/512字节加安全省略号。每行可交互条件是链接数≤16
且解析 InlineSpan 数≤64；超过后整行仍格式化显示，但取消下划线/RoleLink
并在上方给出源码查看提示。文档范围/原文保留，HTML 格式导出不受这个
UI 控件预算影响。相对路径按字面文件名；本地 file: URL 与同文档锚点
在宿主解码一次。未增加链接 title 语法或跨文件 fragment 路由。

原生 Markdown 视图为每个结构有效、`mode: reference` 且 `src` 非空的
doc-block 提供一个普通「打开引用」按钮。pointer-down 和聚焦后的
Enter/Space 会记录 Buffer 身份、版本、预览行 ID、整块及属性区 UTF-16
范围和有界完整 `src`。激活先 Flush，再用 `MapRangeSince` 映射两个范围，
并在当前完整解析上核对同 Buffer、精确整块/属性范围、`BlockDocBlock`、
空 `ReferenceProblem`、reference 模式和完整 `src`。无关内容插在目标前
可以重定位；目标/引用头/围栏构造变化、映射过期、Buffer 替换、Partial、
超限或异组编辑会拒绝，且不会按相同 ID/标题或旧下标另找目标。

只读引用可导航，不创建历史；Flush 本身仍可能产生一次普通提交。
`Host.OpenReference` 只调用一次，并允许它同步切标签和 `Release` 当前视图。
完整目标上限为 4096 UTF-8 字节，4097 字节起禁用；标题、kind 和 mode
副本各最多 256/64/64 字节，tooltip 连同前缀与省略号总计最多 512 字节，
均使用 UTF-8 安全省略。超长目标或宿主缺少打开能力时按钮禁用并显示原因；
结构无效的块不显示按钮。
已有 32 KiB 内容块显示保护和 8 MiB Markdown 完整解析上限不变。
宿主仍只允许已保存文档目录下的字面相对名、本地绝对 Windows 路径、解码
一次的本地 `file:` URL 和当前文档 `#锚点`；网页 URL、UNC、远程 file URL
与未保存文档相对路径拒绝，不静默截短成另一个目标。

该实现已通过最终聚焦门、GLM no-findings 源码复核和一次完整集成；根代理
已接受本轮exe及日志，18:17字段版转为上一验收产物。
证据入口见[集成交接](../handoffs/docblock-reference-integration.md)。

`ExportHTMLChecked(Document) (string, []Warning, error)` 会合并解析警告和
导出警告，调用方不要重复追加 `doc.Warnings`。任何解析预算导致的
`Partial` 都返回空字符串和 `ErrPartialDocument`。兼容方法
`ExportHTML(Document) (string, []Warning)` 同样返回空内容，但无法返回
error；持久化导出必须用 checked 版本并处理错误。

新增字节接口供有界输出调用方使用：

```go
func ExportHTMLBytes(doc Document, maxBytes int) ([]byte, []Warning, error)
var ErrOutputLimit error
```

成功HTML字节与警告内容/顺序/范围与字符串接口相同；返回自有切片，
`len == cap <= maxBytes`。先计数、后精确分配/填充，共享渲染及转义规则，
不重新解析。负预算或任何不足以容纳完整HTML的预算（包括空文档外壳）
返回`ErrOutputLimit`、nil载荷/nil警告。`Partial`优先拒绝，返回
`ErrPartialDocument`和解析警告副本；无部分HTML。64MiB是worker传入的
固定协议上限；API没有文件、进程或协议副作用，结果容量不等于RSS上限。

标题元数据通过 `Headings(doc Document, limit int) ([]Heading, bool)` 提取，
只采用解析器的 `BlockHeading`。`Heading{Range, Level, Text, AnchorID}`
中的范围/文字引用 doc；长期缓存必须另存有界副本。bool 表示解析不完整、
项数限制或 anchor 裁剪。`MaxParseBytes` 暴露已有 8 MiB 解析保护值，方便
调用方在 `Buffer.String()` 分配前拒绝过大文档。

`HeadingAnchors.Next(text) (string, bool)` 与 HTML 标题 ID 共用：Unicode
转小写，保留字母/数字/组合标记，空白与短横线合为一个 `-`，去除其他标点；
空结果为 `section`。基础 slug 最多 256 UTF-8 字节，重复项选择未使用的
`-2`、`-3` 等后缀，连字面标题 `a-2` 也不会碰撞。HTML 属性再做安全转义。
这是当前文档链接规则，不是编辑前后的稳定导航身份。

## 可选大纲契约

```go
type OutlineItem struct {
    ID, AnchorID, Text string
    Level, Offset int
}
type OutlineSnapshot struct {
    Version uint64
    Items []OutlineItem
    Truncated bool
}
type OutlineSource interface {
    OutlineSnapshot() (OutlineSnapshot, error)
    ResolveOutlineItem(id string) (OutlineItem, error)
    ResolveOutlineAnchor(anchor string) (OutlineItem, error)
    Release()
}
type OutlineNavigator interface { NavigateOutlineItem(id string) error }
type CurrentOutlineProvider interface { CurrentOutlineItem() (id string, ok bool) }
```

以上类型位于 `ui/editor`。`Adapter.Outline func(Host) OutlineSource` 可选，
nil 表示不支持；空 Items 表示有效空集，error 表示读取失败，`Truncated`
表示条目/文字/slug 或解析预算不完整。`Version` 等于该 Buffer 的版本，
缓存必须同时核对 Buffer 指针。最多 2048 项，Text 最多 512 UTF-8 字节；
超出 512 字节的导图 ID 被省略，不能截断身份。tooltip/无障碍标签复用
同一有界文字，完整内容在文档中。

Level 为 1 基：Markdown 1–6，导图根1、子节点2。Offset 是 Markdown
标题语法块起点的 UTF-16 偏移；导图为 -1。AnchorID 为空表示没有链接锚点。
Markdown ID 是进程内 Buffer/版本/范围身份，不落盘；Flush 之后须把旧 ID
交给 Resolve，并采用返回的新 ID/偏移。过期范围或目标自身改变返回
`ErrOutlineStale`；能力缺失为 `ErrOutlineUnsupported`，只读隐藏节点为
`ErrOutlineReadOnlyHidden`。不得用旧 slug 或旧列表下标兜底跳转。

Source 的读取与 Release 不提交草稿，不修改选择或折叠，也不创建画布布局。
mounted View 的 Navigate 防御性 Flush；宿主仍需先做自己的生命周期校验。
来源模式和分屏的负偏移策略、折叠展开及选区限制见
[架构说明](../ARCHITECTURE.md#大纲与定位)。Current 是快照提示，不是订阅。

## 代码高亮

```go
result := code.Highlight(source, code.LanguageGo)
// result.Spans []code.Span: Start、End、Token、Kind
// result.Warnings []code.Warning: 不支持语言或预算保护等提示

configured := code.HighlightDeclarative(source, code.DeclarativeSyntax{
    LineComment: "#",
    Keywords: []string{"true", "false"},
})
```

内置 `Language` 为 Go、TypeScript、JavaScript、Python、JSON、Rust、XML，
`LanguageForPath(path)` 可根据扩展名查询。`Token` 常量为
`TokenKeyword/String/Number/Comment/Tag/Name/Punct/Text`。lexer 只返回
数据，不解析工程或执行源程序；单次调用上限为 2 MiB，并有工作量保护。

原生源码视图把每一行单独交给高亮回调，调用之间**没有跨行词法状态**。
跨行注释和字符串可能显示不准确；长行分段也不会推断段前状态。
底层 API 能处理一个字符串内的注释，不等于视图已经实现整篇增量高亮。

## 导图与图纸

```go
mind, mindIssues, err := mindmap.Parse(data, mindmap.ParseOptions{})
if err != nil {
    return err // 将 mindIssues 和错误交给宿主展示
}
diagram, diagramIssues, err := drawio.Parse(data, drawio.ParseOptions{})
if err != nil {
    return err // 将 diagramIssues 和错误交给宿主展示
}
```

两者均返回模型、问题列表和 error。宿主不能忽略问题列表：未知构造保留
不代表原生画布能完整显示。导图通过 `Document.Serialize(SerializeOptions)`
序列化，未修改时保留原输入；修改后的 JSON 会重新组织格式并保留未知字段
（零值选项默认为两空格缩进、LF）。
图纸的受支持编辑基于源 XML 范围修改；压缩页为只读，避免把没有可靠写回
规则的预览变成编辑。具体方法以各包的导出类型为准。

`drawio.Parse`先归一化`ParseOptions`并检查字节上限，再克隆调用方输入；超限
拒绝不触发该克隆。私有原始/初始当前源文可以共享同一份不可变自有字节，
导出的可变`Document.Source`始终独立。受支持编辑构造新的自有当前源文并
整体重解析，成功后才提交；失败保留原文档。提交后原始、当前和公共源文是
三份不同字节，`Serialize()`与`OriginalBytes()`返回防御性副本。结构容器
告警只收窄到按顺序出现的精确0/1空容器对，XML空白和注释不影响判定，其他
不支持内容仍可见。

导图可选字段使用以下纯数据 API：

```go
type FieldKind string
const (
    FieldNote      FieldKind = "note"
    FieldHyperLink FieldKind = "hyperLink"
)
func (d *Document) Field(id string, field FieldKind) (string, bool, error)
func (d *Document) SetNote(id string, value string) error
func (d *Document) SetHyperLink(id string, value string) error
func (d *Document) ClearField(id string, field FieldKind) error
```

`Field` 返回字面值和存在性，读取不改变 dirty 状态。`SetNote("")` 创建
或保留显式空备注；`SetHyperLink("")` 拒绝，移除链接使用 `ClearField`。
字符串不 trim、不解析 URL，不把换行或尾随空格规范化；是否可导航由宿主
决定。字段种类错误、节点不存在、无效 UTF-8 或实际修改后的模型校验失败
返回错误并保持原状态。同值写入和清除已缺失字段不改变原始序列化字节；
实际修改保留未知成员/数值字面量，但遵循上述 JSON 重新排版边界。

原生属性面板只为真实文字变化调用 setter：查看缺失空备注不创建字段，
清空已有备注保存 `""`，清空已有链接移除成员。草稿身份包含 Buffer、节点、
字段种类和原存在性/完整值，按最新源码只合并该字段；其他字段可同时变化。
成功 Flush 是一次文档事务。链接激活在原生按下手势捕获目标，Flush 后
重验；待同步链接使用显式「写回并打开链接」动作，不回退到当前节点地址。
目标超过4096 UTF-8 字节拒绝激活，提示副本最多512字节加省略号。
这些限制只约束导航元数据，不截断存储字段，也不使格式包具备文件访问能力。

标记列表使用独立的纯格式接口：

```go
func (d *Document) Icons(id string) ([]string, bool, error)
func (d *Document) SetIcons(id string, values []string) error
```

`Icons`返回精确有序副本与存在性；缺失为`nil,false`，显式空列表为非nil
空副本与`true`。`SetIcons(nil)`也写成存在的`[]`，不是删除字段。任意有效
UTF-8字形、顺序和重复项保留，调用者数组不会被借用；相同存在列表是
无改动操作。节点缺失、无效UTF-8或模型校验失败返回错误并保持原状态。
格式层没有优先级/色标词表，也没有64项/4096字节UI限制。

原生视图提供19个固定字形按钮：优先级与色标分别互斥，符号可多选，
同字形重按取消。当前和候选均须≤64项且字形总计≤4096 UTF-8字节；
先检查再复制，超限全清也拒绝并指向源码。手势记录Buffer/当前解析版本、
节点ID、icons存在性与完整有序列表，Flush后重新解析核对，不能保留旧节点
指针跨Flush。候选序列化/重解析/文档上限检查成功才以一次事务提交，
前置字段Flush自身可另有一次历史。未知Raw成员数值字面量、标签与图片
附加字段保留；实际编辑仍会规范化JSON排版。

标签列表使用以下纯格式接口：

```go
func (d *Document) Tags(id string) ([]Tag, bool, error)
func (d *Document) AddTag(id, value string) error
func (d *Document) RemoveTag(id string, ordinal int) error
func (d *Document) ClearTags(id string) error
```

`Tags`返回深复制的有序值和存在性，Raw、Style/Extra成员及字符串均不借用
源模型；缺失为`nil,false`，存在空列表为非nil/true。`AddTag`追加一个非空
有效UTF-8字面字符串，不trim/拆分/去重，不修改已有对象/字符串标签种类。
`RemoveTag`删除唯一零基序号，最后一项删除后保留`[]`；`ClearTags`对缺失
和已空列表无操作，对非空列表写存在的`[]`。节点不存在、索引越界、无效
输入或修改后完整模型校验失败返回错误并保持原状态。未知Raw数值字面量、
对象class/style存在性和成员顺序保留；实际修改仍重新组织JSON排版。

纯API允许有效非空多行/长文字，不施加UI的512字节单行、64项或8KiB限制。
graphview Add检查LenBytes后才物化输入，拒绝CR/LF、空串和无效UTF-8。
列表预算逐项为`1+len(Raw)+len(Text)+len(ClassName)`，再加Style/Extra每个
成员的`1+len(Key)+len(Raw)`；当前/候选均至多64项/8192字节，克隆前先验，
自身字段写回后重算Raw，再验证真实序列化候选。它不是仅可见文字字数，
超限需使用源码，拒绝保留原输入，全清不能绕过预算。

未添加输入在公共Flush重入早退之前拒绝保存/导出/切换/关闭，不会隐式Add。
Add私下调用ordinary-field helper并接受严格自身0/1版本回执，随后重解完整
有序标签/存在性，不跨回调保存可变节点。普通字段与标签各记一次撤销；
字段已写回而标签拒绝必须区分报告。成功/恢复仅清理捕获的原输入，宿主
同步Release或替换草稿不能被覆盖。恢复两份草稿时追加明确标签区，field-only
信封不变。待添加标签不承诺进入会话/崩溃恢复快照，无新增editor.Host API。

纯导图展示和有界SVG接口为：

```go
func NativeLayoutOptions() LayoutOptions
const MaxDisplayLabelBytes = 512
func DisplayLabel(node *Node, fallback string) string
func DisplayLabelTruncated(node *Node) bool
func LiteralColor(value string) (string, bool)
func PresentationIssues(doc *Document) []Issue
const MaxSVGBytes = 64 << 20
var ErrSVGOutputLimit error
func ExportSVG(doc *Document, layout LayoutResult, maxBytes int) ([]byte, error)
```

原生布局选项按值返回（节点高40、宽92–380、间隔72/24、padding28），不改
通用Layout默认值。显示标签只构造UTF-8安全的512字节前缀加省略号，不先
拼接整份icons/tags。颜色接受有界十六进制/命名纯色；展示问题在隐藏节点
也检查，顺序为图片占位、标签截断、不支持颜色；nil文档/root返回空问题。
这些廉价规则不生成SVG，图形同步/Draw不为展示元数据调用导出器。

`ExportSVG`零预算用64MiB，正数夹到64MiB，负数返回ErrSVGOutputLimit；
先有界计数，再一次分配精确长度结果并填充，任意错误返回nil而非部分SVG。
调用期间不得修改doc/layout；这不是AST/进程内存上限。图片只占位，用户
标签经XML转义，未知元数据仍留在源文而不会因此全部进入图片。

图形视图的参考 `Export("svg")` 使用当前布局或当前图纸页；`"png"` 返回
`graphview.ErrPNGUnsupported`。不要把底层解析能力、视图编辑能力和导出
保真度视为同一个承诺；应用的导图和draw.io当前页导出均由能力分派到
固定worker，宿主保留编辑与预览。用户能力清单见 [README](../README.md)。
导图SVG和画布共用有界主题/标记/标签摘要，前缀最多512 UTF-8字节并安全转义；
超长时明确提示到源码查看完整内容，导出不修改文档或放大节点布局。

`mindmap.DefaultMaxBytes`（16 MiB）和 `drawio.DefaultMaxBytes`（32 MiB）
是零值 ParseOptions 使用的实际默认值。视图与轻量大纲从 Buffer 读取整篇
前用这些常量检查 `LenBytes`；Markdown 对应 `MaxParseBytes`（8 MiB）。
这不改变纯解析器可接受的显式 ParseOptions，也不扩大富视图的处理上限。

超限视图仍通过已有 Warnings/Draw 返回源码指导，Export 返回 nil 数据及
错误。Markdown 使用可被 `errors.Is` 识别的 `markdown.ErrPartialDocument`；
图形使用各自的 `ErrLimitExceeded`。无需在宿主新增格式专用接口。图形有
待同步字段时 Flush 也会拒绝超限完整解析；Markdown 的安全范围写回可继续。

## 注册表与插件管理

```go
registry := plugins.NewRegistry()
match, err := registry.Match("README.MARKDOWN")
if err != nil {
    return err // 应用在这里接独立的纯文本入口
}
manifest, err := registry.Manifest(match.ID)
```

`Match(path) (Match, error)` 按扩展名精确匹配，忽略大小写。优先级是
Markdown（Order 1）→ Mind Map（2）→ draw.io（3）→ 安装的声明式配置（50）
→ Code（100）；相同 Order 保留注册顺序。代码格式重复声明 Markdown、
导图和图纸后缀，作为它们停用后的候选，但它不是不可停用的兜底。

```go
manager, err := plugins.NewManager(settingsRoot, installRoot)
if err != nil {
    return err
}
err = manager.InstallFile(manifestPath) // 验证并复制一个本地 JSON；不覆盖
if err != nil {
    return err
}
err = manager.SetEnabled("markdown", false)
if err != nil {
    return err
}
match, err := manager.Match(path)
if err != nil {
    return err // 宿主以纯文本回退，并显示未匹配原因
}
manifests := manager.Manifests()
syntax, ok := manager.Syntax(match.ID)
loadErrors := manager.LoadErrors()
err = manager.Refresh()
if err != nil {
    return err // 刷新后的完整问题可从 LoadErrors() 查询
}
```

示例省略了导入、变量使用和宿主 UI；每个有 error 的操作都需要宿主独立
处理。`NewManager` 可在偏好损坏时继续创建，将问题留在
`LoadErrors()`；因此构造成功不等于没有加载问题。

**内置与安装的插件都可启停。** 停用项仍列出，安装项的有效语法数据仍
可查询；匹配会跳过它们。Markdown 与 Code 都停用，且无其他声明匹配时，
`Match` 返回 error，应用以不属于注册表的纯文本编辑器打开。
没有匹配结果不会阻止打开可读取的文本。

`Manifest()`/`Manifests()`的扩展名按匹配规则补点、去首尾空白、转小写和
去重；`Syntax()`保留配置声明的独立副本。`Manifests()` 和 `Syntax()`
返回可变切片的副本。`Enabled` 是保存的选择，
`Active` 是实际能参与匹配的状态，`Builtin`/`Declarative` 区分来源。
`Capabilities{Read, Edit, Source, Export}` 是描述数据；应用的可用导出
入口由对应 `editor.Adapter.Exports` 与可选 `ExportEngine` 决定，不能仅据 `Caps.Export` 推断
具体格式或已支持的编辑操作。

`ErrEnabledNotLoaded`只表示`SetEnabled`已保存偏好并reload，但启用的目标
仍无法加载。调用者应用`errors.Is(err, plugins.ErrEnabledNotLoaded)`识别，
报告问题并重新配置视图；普通保存偏好失败不会改变旧registry/view，不能
因任意error一律释放草稿。详细错误仍由`LoadErrors()`提供。

## 可安装的 JSON 格式配置

```json
{
  "schema": 1,
  "id": "toml",
  "name": "TOML",
  "apiVersion": 1,
  "version": "1.0.0",
  "extensions": [".toml"],
  "lineComment": "#",
  "keywords": ["true", "false"]
}
```

`blockStart`/`blockEnd` 可同时提供或同时为空。未知 JSON 字段、无效 ID、
重复 ID、错误 API 版本等会报错；插件 ID 由小写字母、数字及单个连字符
分词组成。安装不覆盖已有 ID 或目标文件，也不接受符号链接。

- 单个配置最多 64 KiB；目录最多检查 1024 个条目、加载 256 个插件。
- 注释标记最多 8 字节；关键字最多 512 项，每项最多 64 字节。
- `plugins.LoadInstallDir(root)` 返回 `LoadResult{Manifests, Files, Errors}`。
- `plugins.SettingsPath(root)` 指向 `plugins.json`；`LoadSettings(root)`
  和 `SaveSettings(root, Settings)` 管理停用 ID 集合。
- 应用把配置安装在 `<UserConfigDir>/LevNoteNative/plugins/`，偏好在父目录。

schema1 只扩展源码匹配与高亮元数据。schema2 增加逐行字面大纲规则：

```json
{
  "schema": 2,
  "id": "ini-outline",
  "name": "INI outline",
  "apiVersion": 1,
  "version": "1.0.0",
  "extensions": [".ini"],
  "lineComment": "#",
  "outline": [
    { "prefix": "[", "suffix": "]", "level": 1 },
    { "prefix": "# ", "level": 2 }
  ]
}
```

`schema: 1` 不能含 `outline`（即使是空数组），不会自动升级。schema2
继续使用 API 1；已有 API/version 默认行为保留，显式 `apiVersion: 1`
最清楚。`outline` 省略或 `[]` 表示没有大纲能力，null/非数组拒绝。
每条规则必须有非空 prefix 和整数 level，suffix 可省略或为空，但不能
为 null。未知顶层/规则键、错误类型、错误 Unicode 转义、无效 UTF-8、
控制字节、重复 prefix/suffix 对均拒绝；level 不参与解除重复判定。
最多 8 条，prefix 为 1–16 UTF-8 字节、suffix 为 0–16 字节、level 为 1–6。
配置和 manager 返回的规则切片各有独立所有权。

匹配先去掉一条物理行前面的 ASCII 空格/tab，再按声明顺序查找 prefix；
末尾空格/tab 去掉后必须以 suffix 结束。两标记之间的标签再去掉两端
ASCII 空格/tab。第一个完整标记匹配拥有这行；标签为空就省略，不能再
落到后面的宽规则。CRLF/EOF 前恰好一个末端 CR 不参与匹配，内部 CR
保留；没有 LF 的末行要读到真正 EOF 才完整。前后缀是字面值，不是正则。

**规则不理解多行注释、字符串、围栏或 heredoc。** 其中的同形行也可能
匹配；高亮结果不能充当排除规则。纯源码格式没有 Markdown AnchorID，
也没有阅读/分屏/图形导出。安装示例：
[INI](../examples/plugins/ini-outline.json)、[配置标题](../examples/plugins/config-headings.json)。

```go
type Rule struct { Prefix, Suffix string; Level int }
type Config struct { PluginID string; Schema int; Rules []Rule }
type Entry struct { Start, End, Level int; Text string }
type Result struct { Entries []Entry; Truncated bool; Bytes, Lines int }
func Scan(reader io.Reader, rules []Rule) (Result, error)
// modules/sourceview 的中立配置适配入口：
func NewOutline(host editor.Host, config sourceoutline.Config) editor.OutlineSource
// modules 的宿主组合入口，负责 plugins DTO -> sourceoutline DTO：
func InstalledSourceOutlineFactory(pluginID string, syntax plugins.SyntaxManifest) func(editor.Host) editor.OutlineSource
```

Scan 最多解释 2 MiB、65,536 行、2048 项，标签最多 512 UTF-8 字节；
Truncated 披露预算或标签裁剪。4096 字节固定缓冲可预读过行/项数边界，
但总消费最多 2 MiB + 1，未解释数据不输出。预算中断的半行绝不输出；
连续 8 次 `(0,nil)` 返回 `io.ErrNoProgress`，读取失败与非法 UTF-8 返回
error，不伪装成空大纲。调用方提供者从 Buffer 分段读取，不复制完整行。
旧 ID 绑定 Buffer/版本/完整行范围以及插件/schema/有序规则指纹；规则
变化、范围重叠、重载或 64 次日志过期会返回 stale。Release 释放条目和
扫描状态，保留最多 8 条不可变规则以便再次读取。

## Schema3固定原生格式选择

```go
// core/plugins
type NativeRoute string
const NativeRouteMarkdown NativeRoute = "builtin.markdown"
const NativeRouteMindmap NativeRoute = "builtin.mindmap"
const NativeRouteDrawio NativeRoute = "builtin.drawio"
// SyntaxManifest.NativeRoute NativeRoute

// modules：每次返回独立的固定适配器值，不查询内置配置启用状态。
func NativeRouteAdapter(route plugins.NativeRoute) (editor.Adapter, bool)

// app：先验实际Match，再Flush并重验来源和Match，成功添加文档返回true。
func (a *App) NewFromPlugin(id string) bool
```

完整样例为[task-notes.json](../examples/plugins/task-notes.json)、
[thoughts.json](../examples/plugins/thoughts.json)与
[diagram-notes.json](../examples/plugins/diagram-notes.json)。必需字段为`schema:3`、
`id`、`name`、非空`extensions`、`nativeRoute`。只接受上述三个目标；
version/apiVersion仅在省略时分别默认`1.0.0`/`1`，显式错误值不默认。
可保留注释/keywords高亮元数据，不接受`outline`，原生语义大纲由适配器
提供；draw.io路由按能力描述不提供大纲。直接解码/安装均先受64KiB上限
保护；schema3严格拒绝重复键
（含解码后同名）、未知或大小写错误字段、null、错误类型、非法UTF-8和
孤立代理转义。出现schema3/nativeRoute时不能靠最后一个schema键降级到
旧解码规则；schema1/2原有默认及重复键行为保持。

匹配配置ID如`task-notes`写入会话和启停状态，适配器ID仍为`markdown`；
`thoughts`对应`mindmap`，`diagram-notes`对应`drawio`。NativeRouteAdapter
按固定适配器描述提供实际能力，不注册可执行命令；Markdown/导图提供
模板、视图、大纲与导出，draw.io提供模板、视图与当前页SVG，没有大纲或
只读场景。`InstalledSourceOutlineFactory`仍只
处理schema2，schema3不混用字面规则。优先级不变：新建使用规范化后的
首扩展名，必须实际Match回原配置ID；被遮蔽则拒绝，不改用第二后缀。
Flush后复核Manager与来源视图、文档/缓冲/Editor/generation以及新模板、
route、Match；SaveAs在对话框后也重新Flush并复核，按新路径重匹配但不
转换正文。禁用或同ID重扫会使旧大纲/场景/导出身份失效。

`.8`的[核心验收](../handoffs/format-routing-core-root-acceptance.md)、
[应用验收](../handoffs/format-routing-app-root-acceptance.md)和
[精确交付](../handoffs/format-routing-delivery.md)覆盖前两条路由。`.20`
[源码验收](../handoffs/drawio-route-root-acceptance.md)、
[实现结果](../handoffs/drawio-route-implementation-result.md)、
[独立核对](../handoffs/drawio-route-independent-review.md)、
[包验收](../handoffs/drawio-route-release20-root-acceptance.md)与
[交付](../handoffs/drawio-route-release20-delivery.md)覆盖
`builtin.drawio`严格解码、真实安装、模板、SaveAs、失效与路由生命周期；
隐藏运行只证明加载/路由/干净生命周期。

这里的插件是数据配置，不是可下载的可执行编辑引擎。
配置不能提供 Go 函数、脚本、动态库、文件读写、进程启动或网络能力。
