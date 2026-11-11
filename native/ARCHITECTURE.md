# 原生实现架构

本页描述 `native/` 的现有实现。产品能力和未迁移项见 [README](README.md)，
格式与安装配置的接口见 [API](formats/API.md)。旧 TypeScript/Rust 架构文档
保留其历史意义，不作为 Go 包的导入约束或实现说明。

## 从窗口到文档

```text
cmd/levnote ── MyGo 窗口与原生 Surface
     │
    app ───── ui/editor 的 View / Adapter / Host 契约
     │                 ▲
     │             modules/markdownview、modules/graphview
     │                 │
     ├── core/*        └── formats/*（无 UI、无文件读写）
     └── modules（内置视图的组合入口）
```

`cmd/levnote/main.go` 通过 `Content: ui.View(s.View)` 创建窗口，挂接关闭、
焦点和文件拖入事件。默认 1180×780，最小 760×480。没有用网页承载编辑器。
MyGo 提供窗口、绘制、字体排版、输入与平台服务；LevNote 没有使用 MyGo
现成的 Markdown、代码或富文本编辑引擎。

| 位置 | 所拥有的状态与职责 |
|---|---|
| `core/text` | UTF-8 分块 piece table、UTF-16 编辑坐标、选区、版本、撤销与范围日志 |
| `core/document` | 路径、编码、BOM、脏状态、保存基准、磁盘读取/原子写入/冲突判定 |
| `core/session` | 有预算的恢复快照、偏好和草稿副本；不冒用原文件保存基准 |
| `core/search` | 文档/工作区匹配、受限读取、取消、结果与遍历预算 |
| `core/plugins` | 有序匹配、声明式语法配置、本地安装和启停状态 |
| `core/enginecatalog` | 编译期固定的四个引擎ID、格式、平台、路径及输入/输出预算；按值返回描述符 |
| `core/engineipc` | 无UI/进程/文件依赖的一次性有界帧、严格JSON/UTF8和类型化结果 |
| `core/engineruntime` | 固定worker包验证、隐藏进程/Windows Job、取消/超时/Wait与有界stdio |
| `core/enginescene` | 无UI/文件/格式模型依赖的有界中性场景DTO、严格解码与精确分配编码 |
| `cmd/levnote-markdown-export` | 仅继承stdio的单请求Markdown HTML worker，复用纯解析/导出器 |
| `cmd/levnote-mindmap-svg` | 仅继承stdio的单请求导图SVG worker，复用纯解析、布局和有界SVG |
| `cmd/levnote-mindmap-preview` | 仅继承stdio的只读导图场景worker，解析/布局后输出中性树、几何、字面标签/颜色 |
| `cmd/levnote-drawio-svg` | 仅继承stdio的单请求draw.io当前页SVG worker，复用有界解析与导出器 |
| `formats/*` | Markdown、代码、导图、图纸的纯数据逻辑；不直接调用宿主文件操作 |
| `ui/editor` | 自研文本输入与绘制，向格式视图提供统一的小型文本字段 |
| `modules/*view` | 解析缓存、预览/画布、局部草稿、格式动作和导出 |
| `app` | 标签、文件树、查找面板、设置、问题记录、生命周期与平台操作编排 |

`app` 经 `modules.Builtins()` 获得内置适配器，再按插件 ID 接线；不从
控件树结构猜测内核类型。代码高亮作为纯函数回调接到文本编辑器。没有启用
的适配器匹配时，应用直接保留独立的文本编辑器，不再依赖 Code 插件。

## 本地配置与固定原生格式

schema1/2仍是源码高亮/字面大纲数据。schema3的`NativeRoute`只允许
`builtin.markdown`、`builtin.mindmap`与`builtin.drawio`，由
`modules.NativeRouteAdapter`返回新的已知适配器值；app不导入具体视图。
安装配置ID用于匹配、启停和会话记录，`Adapter.ID`仍是`markdown`、
`mindmap`或`drawio`，能力来自固定描述符；draw.io路由没有大纲或只读场景。
安装配置可独立启用，内置同名配置停用不妨碍其使用已提供的能力。优先级
和先注册先匹配规则不变，配置不能提供可执行路径、命令或新的引擎ID。

`App.NewFromPlugin`只用第一个规范化扩展名与该适配器模板，先检查当前
实际Match，已有视图Flush后再次检查Manager、标签/Doc/Buffer/Editor/View、
generation和新模板/路由，随后才释放旧视图并添加新文档；首后缀被遮蔽
明确拒绝。重配即使保持同一配置ID，也释放旧大纲、场景和导出来源身份。
安装对话框返回后重做Flush；SaveAs也在对话框返回后同步并复核来源、
模式/路径及generation，避免恢复或替换视图后误写。SaveAs只按新路径匹配，
不转换内容。未添加标签/写回失败仍保留草稿，显式恢复可以换到新标签。

`Manager.SetEnabled`保存偏好失败时不改变已加载状态。偏好成功落盘但
启用目标在reload后不可用，返回包装的`ErrEnabledNotLoaded`；app用
`errors.Is`识别并按真实注册表重配，同时报告失败。这样不会将仍显示的
旧适配器误当作已启用配置，也不会在持久化失败时丢弃现有输入。
规范与有限实际验证见[API](formats/API.md#schema3固定原生格式选择)和
[应用交接](handoffs/format-routing-app-result.md)。

## 二进制与本地图片

内置图片阅读器作为格式插件组合，而不是独立文档引擎。二进制权威
以不可变的已接受文件状态为基准；需要读取或复制时，由单独拥有的不可变
快照提供`Reader`、`ReaderAt`与`WriterTo`式访问，公开接口不逃出原始载荷
切片。重载会重新核对基准，原文件之后的改动不会被静默当作当前内容。

图片打开、重载、解码和复制操作串行执行并核对标签、文档、来源与请求
身份；待处理请求可被更新的请求取代或取消，迟到结果不会发布到已变化的
标签或文档。只有活动图片保留解码来源/位图呈现，非活动载荷释放；
打开、重载或另存期间的短暂交接会按已知基准同时保留新旧快照的必要
引用，交接完成后释放旧基准。模态、关闭、恢复和事务屏障会明确拒绝或
延后不安全操作。公开的位图释放是资源释放合同，不是整进程RSS、VRAM
或GPU占用的承诺。

`.26`的显式图片引用是本地 Markdown 路由，不是新的图片渲染/解码路径。
格式层在遇到 `![label](target)` 时保留原始语法并识别可在阅读/分屏中
激活的本地图片引用；标题式与引用式形式不支持。目标为 1–4096 UTF-8
字节，排除 ASCII 空白/控制字符、尖括号和圆括号；本地相对路径、百分号
编码与 Windows 反斜杠按原文传递。视图手势先捕获来源身份与有界目标，
Flush 后重验；激活复用现有 `Host.OpenReference` 的本地文件打开、读取器
与字节回退路径，不自动内嵌解码，不引入图片缓存。HTML 导出继续转义并
保留原 Markdown 文本。

只读图片模式不编辑、转码或联网读取，失败与支持子集边界明示；不能解码
但仍可读取原始字节的文件可回退字节视图，读取失败的文件如实报告。
「另存副本」使用已核验的原始字节并采用不覆盖目标的新
文件发布。会话记录只保存路径、顺序与活动状态；恢复时重新读取保存的路径。
校验和与来源记录保留原格式引擎和授权解码/工具包的区分，不因哈希或许可
记录产生版权豁免。

`.24`的恢复调度由应用层拥有：最新编辑后约900毫秒静默启动一次待处理
保存，并通过应用计时器派发，因此不需要View绘制帧。待处理任务仍走既有
会话发布和身份检查，可能被重UI工作或原生对话框延后；它只写恢复状态，
不自动写回原文件，也不宣称无损崩溃恢复或精确墙钟期限。

## 一次性 HTML、导图 SVG与draw.io当前页导出进程

`Adapter.ExportEngine *ExportEngine{ID, Format string; RequiresView bool}`
是可选能力元数据，Markdown声明`levnote.markdown-export`/`html`且不需要
View，Mindmap声明`levnote.mindmap-svg`/`svg`且不需要View，draw.io声明
`levnote.drawio-svg`/`svg`且`RequiresView=true`。`enginecatalog.Lookup`
返回固定描述符副本，
`Match`验证格式和平台；清单或调用方不能增加ID、命令、路径与预算。
外壳按能力分派，不导入具体格式视图。Markdown/导图不要求挂载View，
源码模式导图也可导出而不创建画布；draw.io必须先在阅读/编辑或分屏中
选择页面，源码视图如实禁用入口。
已启用的引擎失败时不调用进程内View.Export；后者保留为测试/参考纯路径。
schema1/2/3 JSON配置均无法声明可执行程序。禁用后按剩余注册表重匹配，
若落到源码兜底则移除旧导出/场景能力。
可编辑导图、语义大纲解析及Markdown/draw.io预览仍在宿主；只读导图使用
下述独立场景路径，UI绘制始终在宿主。

app先验证能力与源预算，捕获来源标签/文档/Buffer/编辑器/适配器/能力身份，
Flush后重验这些所有者，再捕获不可变Snapshot/LenBytes与写回后的版本；
快照传输不把整篇源文flatten为字符串。后台Snapshot.WriteTo把精确字节数写入
一条24字节小端头+有界JSON元数据+原始载荷。响应同样一帧，必须长度、
UTF8、字段大小写/类型/必需项、重复键、身份/版本、EOF和exit0全部成立。
Markdown源8MiB，导图源16MiB，结果各64MiB，最多256警告；stderr16KiB，
错误不引用源文。runtime统一返回唯一的`Result.Payload`字节切片。自`.2`起
worker调用ExportHTMLBytes：共享渲染器先有界计数，再分配精确长度缓冲
填充；五种实体按片段转义，不先构造完整转义临时串。计数与填充复用同一
描述符检查/实体映射，警告仅填充时收集，顺序不变。输出缓冲len==cap且
不超过预算，失败无部分载荷。兼容字符串接口仍共享规则单遍写Builder。
这约束结果容量，不涵盖源文/AST/宿主副本或进程RSS；历史`.1`仅在完整
字符串生成后检查上限。

导图worker按`Parse → Layout(NativeLayoutOptions()) → PresentationIssues →
ExportSVG`处理，保留原生布局与SVG字节语义；SVG同样先有界计数、再精确
分配/填充，错误无部分载荷。解析、布局、显示警告依序转换为整份源码
`[0,len(source))`范围，不伪造节点坐标；警告数量/字节超限直接拒绝。
图片只是占位，不读本地/外部资源。纯展示层把廉价`DisplayLabel`、
`LiteralColor`、`PresentationIssues`与SVG生成分开；graphview的Draw/sync
不会为画布元数据生成SVG。`NativeLayoutOptions`按值返回，通用Layout默认
值不变。共享提取前保留的四组SVG/几何/警告基线继续作为字节回归依据。

`NewBundled`按请求ID只从实际宿主目录下`engines/markdown-export/`、
`engines/mindmap-svg/`、`engines/mindmap-preview/`或
`engines/drawio-svg/`寻找对应固定清单/worker。
`NewManaged`从`PackageManager`读取当前选择：`bundled`走上述宿主目录，
`installed`只走已验证的
`stateRoot/<engine>/<packageDigest>`。Markdown清单不能满足Mindmap请求。
runtime不接受shell、PATH或用户指定命令。每次运行锁定本地驱动器路径目录及常规
文件，拒绝reparse并核对句柄最终位置，持有拒绝写入/删除共享的句柄完成
SHA256与大小验证直至运行结束。清单与程序一致是完整性一致性，不是签名
或同时替换两者的防护，更不是恶意程序沙箱。

draw.io请求额外携带当前页的`RequestPage{Index int, ID string}`：
索引是零基页序，ID是页面元素ID；页面对象对draw.io必填，旧三引擎仍传
`nil`。页面索引0..249,999、ID最多512字节；ID只有当所选源页没有ID时
才可为空，缺失页面在worker内如实失败，不回退到第一页。
`startEngineExport`先捕获标签/文档/Buffer/编辑器/适配器/能力/名称/
模式/视图所有者并执行输入Flush；随后的`PrepareExportRequest`只可能
同步过期的解析缓存，并报告当前页与视图epoch，不Flush、发布或丢弃
待写草稿。`ExportRequestToken`是纯getter，返回这份已捕获身份供后续
核对。
页面A-B-A、同索引重解析、Release、视图/模式/标签/文档/Buffer/编辑器
或版本变化都会拒绝发布并释放载荷与警告。待写回字段的初始Flush合并为
一次有意的撤销事务，因此改变Buffer、脏状态和撤销历史；无待写回草稿的
干净导出不改Buffer、选区、脏状态或历史。原始磁盘字节、编码和BOM
不被导出本身改写。
draw.io源32MiB、SVG64MiB、页面索引/ID与警告按协议边界拒绝；默认
30秒仍是全请求超时。raw XML解析有界但会分配解析模型，这些上限不是
进程峰值内存或速度承诺。draw.io的Prepare可能在解析缓存中字符串化源文，
不应把快照传输的免flatten规则扩大成整应用零拷贝或全程不解析承诺。

启动使用隐藏/no-console标志，发送输入前通过精确WithHandle附加到
KILL_ON_JOB_CLOSE Job；失败杀掉该已持有子进程并Wait，不重试未受管进程。
每次Start对应一次Wait，默认全请求30秒、管道退出WaitDelay2秒。已知协议/
stdout溢出立即取消；终止前按最多256成员取得并核实Job内句柄，终止后
等待活动数归零及这些句柄真正signaled。清理失败会返回类型化错误并使runner
拒绝新任务。Close与完成共享锁决定线性化，重复Close保存同一清理结果，
不依赖UI回调才能结束。此机制不是敌对进程树沙箱，仓库worker不派生后代。

app保留一个导出任务槽，四个引擎共用一条执行队列和一个runner准入槽，
取消后仍须等前一任务清理完成。导出结果放在可清空槽中，排队UI回调只捕获任务身份；
迟到或取消结果释放内容，清理错误仍可见。交付前Flush/核对版本，SaveDialog
后再Flush/核对；编辑后撤销也不能发布旧版本。Save/Close/Open/安装等原生
模态路径使用深度计数延后交付，关闭取消不破坏仍有效任务。确认退出先
持久化并等待runner，清理失败拒绝退出并重新保存仍打开的草稿恢复状态。
Shutdown幂等并等待同一完成屏障。最终发布仍是现有同步、不覆盖目标的
WriteNewFile，一旦调用不再承诺取消，磁盘延迟仍影响UI。

### draw.io SVG输出预算与当前页worker

格式层的`ExportSVG(pageIndex, SVGOptions{MaxBytes, ...})`仍保留纯导出
能力，供worker复用。`MaxBytes==0`选择64MiB`MaxSVGBytes`默认硬限，正数按请求值
生效、超过硬限夹取，负数返回`ErrSVGOutputLimit`。页码查找保持原有优先级，
无效`pageIndex`先于输出预算失败。

生成器分两遍：第一遍只计数输出字节并在计数路径收集有序列`Issue`，不创建
完整SVG缓冲；第二遍按精确大小分配并填充。填充长度与计数不一致时返回错误。
超限和每个外层错误都返回nil SVG、nil issues，不泄漏部分输出。恰好等于上限
时成功路径仍返回既有SVG字节、警告和所选页。XML属性、标签、绘图/字体值与
边路径片段按固定UTF-8安全块写入；计数与填充复用同一描述符、实体和数字格式。

这里的64MiB是序列化输出容量，不是进程内存或峰值内存上限。源文、解析模型、
页面索引、问题/点序列、临时字符串片段和成功输出缓冲仍占内存；两遍会重复
部分确定性几何计算。宿主现有的错误提示是普通UI的可见失败出口。测试与
旧代码逐字节oracle覆盖精确填满、超一字节、负数/夹取、换行放大、页码优先级、
警告顺序和所选页；不据此宣称CPU、速度、整应用内存或打包宿主SVG交互改善。
合同、实现与核对见
[源码验收](handoffs/drawio-svg-budget-root-acceptance.md)、
[实现结果](handoffs/drawio-svg-budget-result.md)和
[独立核对](handoffs/drawio-svg-budget-independent-reconciliation.md)。

当前`.29`交付在文本历史层合并同一完整IME标记范围的连续更新，并保留`.28`
普通UTF-8逻辑内容准入、命令预检和大小拒绝的IME屏障，以及`.27`最近关闭文件
重开与100活动标签准入。历史`.28`交付加入内容增长准入与大小拒绝IME屏障；
历史`.27`交付加入
最近关闭文件重开和100活动标签准入；`.26`交付显式图片引用打开则转为历史
能力，`.25`交付在既有单元格/行/对齐面板上加入前后
插入列和删除当前列。
列操作从权威源码生成精确补丁，保留幸存单元格原始字节、填充、分隔线、
缩进、行尾与相邻内容，仅在需要边界时添加外层管道；单元格写回与列结构
调整分离为两个撤销事务。F6/Tab/ShiftTab/Enter/Space沿用组合输入保护，
不安全/不支持/超预算候选明确拒绝并保留整表源码回退。表格仍为1–8列。
`.24`的图片键盘、明确100%和恢复调度继续有效；worker行为沿用`.21`。
构建、隐藏宿主与行为测试的不同范围及精确身份集中在[验证记录](VALIDATION.md)。

历史`.21`交付继承schema3 draw.io路由和四个固定引擎，并加入已接受的
draw.io结构容器告警收窄与源码所有权优化。一次打包调用完成五次GUI构建；
实际installed draw.io worker零警告返回同一598字节SVG，另一次隐藏自定义
后缀宿主启动干净关闭。详细身份、基准与证据见
[`.21`交付](handoffs/drawio-efficiency-release21-delivery.md)、
[包验收](handoffs/drawio-efficiency-release21-root-acceptance.md)、
[16/16核验](artifacts/acceptance/drawio-efficiency-release21-20261110/validator.json)
和[验证记录](VALIDATION.md)。这些运行不证明物理UI/IME、dirty recovery、
进程内存/CPU、延迟、可见渲染或打包交互式导出表现。

`.20`历史交付是宿主加四个引擎的25文件目录，共26,854,177字节，完整清单
identity为`6B899EFB2FB2A90CB2FB93CFE9BC48A3B94B4CFA2AB16C61C5AB005B05DF14ED`；
宿主14,095,360字节，SHA256
`4B776F634A3626F2BDD15303E175FE38CC4908D566AC54A6D25F51D989B4F34E`；
发布清单4,490字节、24个载荷项，SHA256
`53C0B71C317734001596988F6AB2BF44201879CC322BA9DB3B9AB494E553DBFA`。
根代理接受12/12交付核验。一次打包调用完成五次Windows amd64 GUI构建。
一份精确副本用run-owned普通`C:`配置隐藏启动，加载已安装的
`builtin.drawio`配置与`.diagramnote`文件，持久化干净预览/UTF-8/BOM
单标签会话，owned`WM_CLOSE`后exit0，无worker请求；315个源码/构建输入
共3,153,585字节保持不变。`.19`的实际installed draw.io worker执行
证据保持有效并由本包继承。这些运行没有证明物理UI/IME、dirty recovery、
进程内存/CPU、延迟、可见渲染或打包交互式导出表现。证据见
[`.20`交付](handoffs/drawio-route-release20-delivery.md)、
[包验收](handoffs/drawio-route-release20-root-acceptance.md)、
[12/12核验](artifacts/acceptance/drawio-route-release20-20261110-031456/validator.json)、
[`.19`交付](handoffs/drawio-worker-release19-delivery.md)、
[Stage2源码验收](handoffs/drawio-worker-stage2-root-acceptance.md)、
[Stage2结果](handoffs/drawio-worker-stage2-result.md)和
[独立审查](handoffs/drawio-worker-stage2-independent-review.md)。

`.18`交付保持历史：宿主加三个引擎的23文件目录共23,699,526字节，完整清单
identity为`4A4F9BD0B6DF2DDCD1E597F026AAC985A9D50711CDD85F3E9310F7DC5B0B2991`；
宿主14,075,904字节，SHA256
`BFA6F460F49087390F114CEC2CF2ADC034307B07769D8B64E3E70C78BBDCB44A`；
发布清单4,137字节、22个载荷项，SHA256
`8CA2016C0064638BE965FDFE74468F8004A37143BE568A189FD7FEDAB2D12FDD`。
其隐藏启动只验证启动和干净关闭，没有执行打包SVG导出、dirty recovery、
managed tester、worker请求、物理UI/IME自动化或资源测量。证据见
[`.18`交付](handoffs/drawio-svg-budget-release18-delivery.md)与
[包验收](handoffs/drawio-svg-budget-release18-root-acceptance.md)。

`.17`完整23文件23,690,310字节继续作为历史；其全词查找选项、宿主
`14,066,688`字节、300个源码/构建输入共3,005,261字节及117文件/47,784,783
字节直接证据均保留。Find测试是合成应用/Tester事件，隐藏启动只验证启动和
干净关闭，没有证明物理IME、打包Find交互或资源表现，[交付](handoffs/find-whole-word-release17-delivery.md)、
[包验收](handoffs/find-whole-word-release17-root-acceptance.md)和
[12/12核验](artifacts/acceptance/find-whole-word-release17-20261110-004642/validator.json)
未改写。

`.16`完整23文件23,690,310字节继续作为历史；其会话恢复输出合同、
299个源码/构建输入共2,996,205字节、宿主SHA256
`76A60252CFCC3394386B4DAF87AF8B845086DE35ED5DCCACC6CC79179E60B03D`与
104文件/47,774,882字节直接证据均保留，[交付](handoffs/session-output-release16-delivery.md)、
[包验收](handoffs/session-output-release16-root-acceptance.md)和
[12/12核验](artifacts/acceptance/session-output-release16-20261110-001354/validator.json)
未改写。

`.15`完整23文件23,680,582字节继续作为历史；其UTF-8/BOM有界缓冲保存、
[包验收](handoffs/utf8-save-release15-root-acceptance.md)与
[交付](handoffs/utf8-save-release15-delivery.md)均未改写。`.14`完整
23文件23,673,926字节继续作为历史；其查找键盘焦点/组合输入生命周期、
[包验收](handoffs/find-keyboard-release14-root-acceptance.md)与
[交付](handoffs/find-keyboard-release14-delivery.md)均未改写。`.13`完整
23文件23,655,494字节继续作为历史；其相邻结果导航、活动文档缓存、
760×480两张tester图及
[包验收](handoffs/search-navigation-release13-root-acceptance.md)均未改写。
`.18`继续包含`.17`全词查找、`.16`会话输出、`.15`UTF-8保存、`.14`查找键盘生命周期、
`.12`源码行列导航、`.13`查找导航/布局，
以及`.11`两项最小指针
所有权修正：`Buffer.replace`在piece合并后清理
超出新逻辑长度的死槽位；`removeTab`移动剩余指针、清空被移除的尾部指针
后再缩短切片。两者不改变逻辑内容、快照、撤销/重做或关闭/恢复语义，也
不构成RSS或工作集测量。`.11`完整23文件23,622,214字节及其13/13、两次
受控请求与隐藏启动记录继续保留在[验证记录](VALIDATION.md)。

## 本地引擎包与选择切换

引擎包管理只面向固定catalog的四个ID。导入读取一个完整本地发布目录，
复制该引擎所需的清单、程序、许可与说明，逐文件验证大小和SHA256，并把
收据写入用户配置目录；不触发执行，也不改变当前选择。重复digest会重新
验证既有安装并原样复用，不新增、不覆盖。启用和恢复随包是显式选择转换：
先取消/等待旧导出与预览请求并关闭旧runner，再持久化新选择并替换运行时。
等待期间新任务被阻止；可写草稿、撤销栈和只读源码不被Flush、Release或
回滚，只读场景在成功切换后按需刷新。

这种顺序避免新引擎与旧请求并存导致的版本/来源身份混用，也避免在清理
失败时假装选择已切换。清理失败会把引擎服务标记为poison，阻止后续执行
和新的选择转换，但导入与已安装列表仍可用于诊断。若清理失败发生在选择
状态已经提交之后，UI显示实际选择与实际失败，不推断回滚。

选择状态文件本身损坏时，执行与普通选择转换被阻断，导入/列表仍可用于
诊断；「重置全部选择」是明确修复出口，只恢复随包选择，不删除已装包。
单个已安装包损坏是另一回事：若它正被选中，执行明确失败且不回退随包；
列表继续显示健康替代包及其具体问题，显式启用健康包或恢复随包仍可用。
`Snapshot()`中的随包状态只声明固定catalog身份和`bundled`来源，
`Version`/`Digest`为空；实际版本与摘要在运行时验证后出现在导出结果中。

schema1旧选择文件按原字节读取旧的三个引擎选择，draw.io回落随包；
显式启用/重置/修复后持久化为schema2的四个精确选择项，不重写历史文件
本身。schema2损坏或包含未知项时按现有损坏状态语义处理。

校验和是本地完整性检查，不提供发布者认证或进程沙箱。schema1/2/3 JSON
配置与worker包是两套不同机制：前者不执行代码；后者中已安装包需要显式
启用，随包引擎默认被选择，而任一被选中的worker都只会在实际导出/预览
请求时启动。

## 只读导图场景与共享队列

`Adapter.SceneEngine *SceneCapability{ID,Format}`声明固定
`levnote.mindmap-preview`/`mindmap-scene-v1`。Host提供带owner的异步
`BeginScene`和`ShowSource`；视图不提供源码、文件路径或执行命令。app核对
当前标签/Doc/Buffer/Editor/Adapter/View、版本及能力后捕获不可变Snapshot，
后台调用runtime并解码DTO，UI不flatten整篇。请求源16MiB、场景32MiB，
严格codec拒绝重复/未知字段、无效UTF8/数值、超限和非单根树，不发布部分
结果；节点ID512B、显示标签最多515B，最多100000节点/99999边。DTO没有
源码、图片URL、格式模型指针、主题默认值或宿主身份。

`app/engine_lane.go`只有一处Runner.Export执行循环，至多一项最新待处理
预览；新请求取代旧待处理并取消旧执行。导出占用队列优先级，取消正在
执行的预览并等其joined返回，在SaveDialog/写文件结束前不让预览进入。
被抢占且仍有效的同代预览会恢复；更新的一代替换它，避免永远停在加载。
取消待处理立即释放快照；执行中的不可变输入仅保留到清理完成。清理失败
使队列拒绝后续任务，错误在视图已经失效时仍报告；Close/Shutdown不依赖
UI排队回调并幂等等待，包括尚未执行就被取消的导出。

`app/preview_worker.go`持有单个可清空结果槽。生产完成只发非阻塞Invalidate，
App.View/模态退出在UI线程取结果，先核对完整所有者与版本再交付。切模式、
标签、重载、重配或Release取消并立即清空callback/result/文档引用；迟到
完成不得重建旧视图，结果交付后释放快照和回调。Save/Open/Close等模态
深度屏障共用，关闭被用户拒绝不破坏仍有效的请求。

配置了场景能力的只读Mindmap选择独立`sceneView`，缺少provider/worker
仍显示失败与源码入口，绝不落回宿主格式解析。它只用DTO原生绘制，支持
有限变换、缩放/平移/适应、选择和可见ID导航；首帧无几何时延后居中，
隐藏ID定位在修改选择/平移前拒绝。按视口剔除，256项标签缓存/每帧布局
预算；缩小或预算隐藏有语义文字提示，静止帧不循环Invalidate。空颜色由
主题补齐，字面颜色保持，因此任意文档颜色未必有良好暗色对比度。

只读视图不提供属性编辑器，完整备注、链接、标签与图片引用在源码查看。
图片既不加载也不画占位徽标；SVG既有「图」占位不变。切源码不解除只读，
另存为可写副本后恢复普通编辑面板。宿主大纲仍解析源文，可列出折叠后
未进入DTO的节点，点击时如实拒绝。schema3选择Mindmap的自定义后缀也遵循
相同只读场景路径；schema1/2仍仅提供源码高亮/大纲。此固定DTO不是任意
交互编辑插件协议，可写导图解析/编辑仍在宿主。

## 文本、视图与草稿

共享文档的唯一内容依据是 `Host.Buffer`。解析模型是带 Buffer 身份和版本
的缓存；不能只比较版本号，因为替换后的另一个 Buffer 可能有相同版本。
格式层的字节范围和编辑器的 UTF-16 范围必须显式转换；代理对边界不可拆开。

宿主使用的现有接口位于 `ui/editor/contract.go`：

```go
type View interface {
    Draw(*ui.Context)
    Release()
    Warnings() []string
    Export(format string) ([]byte, error)
}

type Flusher interface { Flush() error }
func FlushView(view View) error
```

`Host` 提供 `Buffer`、`ReadOnly`、`Path`，以及 `Changed`、`Report`、
`OpenReference`、`RecoverDraft(title, content) error` 回调。视图通过回调
请求宿主操作，不直接写文件或自行启动外部程序。

保存、导出、模式/标签切换、插件重配、重新载入和关闭都必须先经过 Flush。
普通组合输入会在这里提交到对应字段；独立待添加标签在公共Flush入口先
拒绝，必须明确Add/discard/recover。错误时保留当前视图、文档和草稿，不接着
Release 或导航。成功恢复为新文档后，原视图才可清除那份未同步草稿。

### 标签快捷键与活动标签显示

`selectAdjacentTab`先检查零/一标签，无副作用返回；其它情况按视觉顺序
计算相邻索引并调用既有`Select`，不另写Flush、导出清理或持久化流程。
精确`Ctrl+Tab`/`Ctrl+Shift+Tab`触发；源码/富视图仍由Select同步输入，
失败保留当前视图、草稿、撤销与导出任务。

快捷键入口仅对可见Find查询/替换字段或GoToLine字段的实际组合输入拒绝，
显示完成/取消后重试提示，保留焦点/preedit且不排队。GoToLine与Find一样
依据当帧公开Focused/Composing状态清理失焦后已结束的组合，避免陈旧标记
永久阻止快捷键；不重建字符串、不改Select的普通源码/富视图提交行为。

标签栏仅记已显示的稳定tab ID；实际活动身份变化时，用当帧按钮的公开
ScrollIntoView显示一次，不跨帧保存控件/文档引用。普通重绘、重复选中及
关闭更早的非活动标签不拉回手动滚动。关闭按钮延后到遍历结束才执行
CloseTab，避免压缩并清空切片尾部后继续访问旧遍历元素。证据见
[标签源码验收](handoffs/tab-workflow-root-acceptance.md)与[验证记录](VALIDATION.md)。

### 源码行列导航

`Ctrl+G`与状态栏当前行列入口打开同一个源码导航面板；目标由
`app/source_navigation.go`捕获标签、Document、Editor、主Buffer与当前显示
Buffer身份。`ui/editor/contract.go`提供可选`editor.Composer`，Markdown与
graph视图分别从`modules/markdownview/view.go`、
`modules/graphview/adapter.go`实现，不窥探工具包控件树。打开导航时，
源码编辑器或任一中性富视图的组合输入会阻止；提交时只有导航查询字段自身
的组合输入会阻止。语法解析通过后，普通`Flush`再同步目标视图自己的文档
草稿，避免把“查询字段未组合”误写成“目标视图没有待同步输入”。

输入只接受正值ASCII`line`或`line:column`，去除外围空白后省略列按1处理；
前导零、零、符号、空字段、额外分隔符和整数溢出拒绝且不夹取，最多32个
UTF-16码元。坐标是一基UTF-16码元位置；CR/LF不是列，末尾空行可定位，
代理对内部的列在`SetPosition`前拒绝，组合码点之间可定位但不承诺字形簇
语义。先解析输入语法，再Flush当前视图一次；行列范围只在Flush后检查，
同时重新核对捕获身份，用有界`LineRange`与空的`SliceUTF16`检查最终缓冲，
不复制整篇文档。

源码和分屏保持原模式，预览只在最终目标有效后切到源码。只读源码允许导航；
当前显示是只读十六进制字节视图时仍按显示行列工作，并在面板中说明目标属于
字节视图。Flush拒绝保留面板、原始查询、目标引用、模式、选区与历史；
Flush成功但目标无效时明确说明文本已同步。成功导航请求一次位置持久化。

面板、输入、回调、观察者和按钮快捷键共享generation；输入元素键也含该代际，
因此关闭/重开不会复用已关闭字段或让旧回调操作新目标。标签选择、新/恢复
标签、提交移除、重载、编码重匹配和插件重配清理面板及捕获指针；失败的切换
或关闭保留原目标。回归入口为`app/source_navigation_test.go`；冻结源码、
测试与两张760×480 tester图见[实现结果](handoffs/source-navigation-result.md)、
[源码验收](handoffs/source-navigation-root-acceptance.md)与
[独立审查](handoffs/source-navigation-independent-review.md)。这些不是
物理Ctrl+G/OS IME或资源测量。

### 文档内查找导航与活动文档缓存

`app/search_navigation.go`把上一处/下一处改为有序UTF-16选区边界上的确定性
推进：非空选区正好等于一个匹配时，按请求的“上一处/下一处”操作方向移动
一个结果，正向或反向选区均适用；其它情况下，
下一处选第一个`Start >= selection.End`的结果，上一处选最后一个
`End <= selection.Start`的结果，并在首尾回绕。零宽结果使用已返回选区来
继续前进，单结果可以回绕到自身。搜索核心、编辑器与格式视图接口不变。

应用只保存当前活动文档的一份缓存：精确的Tab、Document、Editor、Buffer
身份，最终Buffer版本，查询选项、结果错误、上一次索引与选区，以及复制出的
`{Start,End}`偏移。它不保留`search.Match`、源字符串、`text.Snapshot`、
子匹配或按标签分表。每次查找仍先`Flush`，再核对最终身份；冷扫描期间版本
或身份变化会丢弃结果，发布前再核对一次。提交标签加入/切换/移除、成功重载、
插件重配、编码重配、关闭查找、关闭应用或实际替换会清理缓存；失败的Flush、
切换或关闭保留当前文档、草稿和仍有效的缓存。查询/选项/版本不匹配可以懒到
下一帧或下一次查找时清理，但旧缓存不能授权跨文档揭示。替换与ReplaceAll
继续每次使用新的搜索，不复用导航偏移。

活动偏移缓存只覆盖命中后的定位，冷路径仍扫描整篇文本。20次benchmark的
warm lookup为0B/op、0alloc/op，cold path为1,063,316B/op、4,137alloc/op；
这是函数级证据，不是整篇搜索、进程RSS、CPU或整应用提速结论。

最小760×480窗口下，`app/state.go`只给现有查找栏两行增加两个`.Wrap()`；
字段宽度、标签、动作、焦点、Tab顺序和搜索语义不变。原布局的“正则”控件
越过主区域右边界，修正后所有控件落在状态栏上方，浅色/深色与32px编辑字号
均通过tester几何检查。该布局与缓存实现见[查找实现结果](handoffs/search-navigation-result.md)、
[源码验收](handoffs/search-navigation-root-acceptance.md)、
[独立审查](handoffs/search-navigation-independent-review.md)、
[查找栏结果](handoffs/findbar-layout-result.md)与
[查找栏验收](handoffs/findbar-layout-root-acceptance.md)。这些`.13`证据不证明
物理OS IME、`Ctrl+F`焦点或`Enter`快捷键行为；键盘流程由后续`.14`单独交付。

### 全词查找选项

`App.WholeWord`是与`CaseSensitive`、`Regex`并列的运行期Find状态；
`App.findOptions()`是唯一映射点，把三者与查询文本一起组成
`search.Options`。文档上一处/下一处、Replace Current/All和工作区搜索都消费
这同一份选项。活动文档导航缓存把完整选项纳入身份，因此切换全词后不能复用
过滤前的偏移。选项不写入会话或设置，也不引入新依赖。

`core/search`继续使用原有Unicode边界谓词：字母、数字、记号与下划线属于
词字符；查询候选的起止任一侧仍紧邻词字符时，全词过滤不选中它。例如`cat`
命中独立`cat`，不命中`category`、`_cat`或`cat_`；`猫`不会切分`猫咪`，组合
记号也不会被拆成独立词。这是字符边界规则，不是中文分词，也不是语法感知的
标识符匹配。搜索算法、缓存结构与会话schema未因选项增加而改变。

Find的`全词`复选框沿用`KeepFocus`与Escape生命周期；组合输入确认前动作仍不
执行。导航与替换继续每次先Flush并核对最终身份，Replace All继续保持单撤销组；
失败Flush保留文本、版本、选区、历史与字段焦点。Tester实际控件覆盖导航/回绕、
760×480布局、Tab与Escape；Unicode/大小写/正则、工作区、替换和生命周期证据
通过运行期选项与应用方法建立，不宣称物理OS IME或打包宿主交互。实现、核对与
验收见[全词查找结果](handoffs/find-whole-word-implementation-result.md)、
[独立核对](handoffs/find-whole-word-independent-reconciliation.md)和
[源码验收](handoffs/find-whole-word-root-acceptance.md)。

### 查找键盘、焦点与组合输入

`Ctrl+F`和工具栏入口共用同一个Find面板。打开/聚焦请求带generation和一次性
消费标记，并在callback时核对当前标签、Document、Editor与Buffer；重复触发
复用现有面板，不清空Query、Replacement、大小写、全词或正则状态。控件绑定后再次
核对身份，旧代际、旧标签或旧Buffer的请求只被丢弃，不把焦点、导航或替换
重定向到后来打开的目标。源码揭示成功时先取消该揭示自己的pending source
focus，再应用Find的一体化查询框焦点请求；这不是另一个状态机。

查询框在读取当前`TextKey.Text()`后处理Enter为下一处、Shift+Enter为上一处，
成功后恢复查询框焦点；替换框Enter沿用现有replace-current路径，
Shift+Enter不动作。Escape只关闭Find并把焦点还给源码/分屏中的当前源编辑器；
阅读预览保持原状，不向不可见源编辑器投递焦点。Escape先由可见Find字段或
动作处理；全局快捷键只在焦点位于可见Find面板之外且GoToLine不存在时参与，
避免抢先关闭整个面板。

活动组合输入时，原始Enter/Escape由Find字段消费；Find的导航、替换、打开/关闭
及相关焦点动作与`Ctrl+G`暂不执行。焦点仍可转移到其他控件、标签或Plugins；
工具包可能在结束组合时提交却不发InputText，因此App用公开Focus/Compose状态
核对已失焦字段，只有失去焦点且不再组合时才清除组合标记。
工具包可能在焦点转移、其他控件聚焦或子树隐藏时结束组合却不发InputText，
因此App在帧内用公开`Element.Focused()`与`Element.Composing()`核对已失焦
字段，只有两者都结束才清除组合标记。`ShowPlugins`隐藏Find时同时取消两个
字段handle和精确的pending focus目标；查询、替换、选项、源文本、选区、版本、
历史与查找缓存不变。

替换在Flush前捕获完整身份，Flush后再核对身份并使用新的匹配/版本结果，
不在陈旧结果上提交；失败Flush保留源文本、选区、历史和字段焦点。
`Editor.CancelFocus()`仅取消当前源码揭示的排队焦点，不结束文档或编辑器
生命周期。回归入口为`app/find_keyboard_test.go`与
`app/find_keyboard_lifecycle_test.go`；证据是合成App/Tester事件，不是物理
Windows IME、候选窗或资源测量。结果、独立核对和包验收见
[查找键盘结果](handoffs/find-keyboard-result.md)、
[独立核对](handoffs/find-keyboard-independent-reconciliation.md)、
[源码验收](handoffs/find-keyboard-root-acceptance.md)与
[`.14`包验收](handoffs/find-keyboard-release14-root-acceptance.md)。

Markdown 局部编辑持有正文范围和包围语法的保护范围，并按版本映射。
`Buffer.MapRangeSince(version, Range) (Range, bool)` 只记最近 64 次变化的
坐标，不保存额外文本：

- 在非空范围起点插入视为插在目标之前，两端一起移动；终点插入视为目标
  之后，两端不动。
- 插入目标内部或空范围所在点、重叠替换、历史过期、Restore 或无效边界
  都使映射失败。Undo、Redo 和取消事务的实际逆向修改也进入日志。
- 调用方还须验证 Buffer 身份和目标字段。来源变化时，干净草稿让位于
  新源码；有本地改动的冲突草稿留在面板中，可恢复为副本或显式丢弃。

图形字段使用 Buffer、页面身份、节点/cell ID 和源字段基准判定目标。
导图的主题、纯文本备注、本地链接共用一个独立 Editor；草稿另记字段种类
及原存在性/完整字面值。切换字段先 Flush；写回时重新解析当前文档，只
合并捕获字段，其他字段的新变化保留。缺失与显式空备注是不同状态，链接
清空则移除成员。来源已改同一字段或节点消失时，旧草稿保持可恢复，不能
借用当前选中节点。干净草稿让位于源码 Undo。一次真实 Flush 产生一个
普通文档事务，不承诺整个跨多次保存的输入过程只有一次撤销。

普通属性只保留一个字段编辑器，固定512单位排版预算并在224px属性区内
滚动。是否有未同步文字按输入 Buffer 版本缓存精确比较结果，避免静止帧
反复复制长备注；完整字段和来源基线仍被持有。字段和选择偏好在 Release
后可恢复，解析/排版缓存释放；冲突草稿在恢复成功前保留。

导图链接按钮在原生 pointer/key down 捕获 Buffer、节点、原链接存在性与
值，待同步链接还绑定确切草稿及其输入版本。普通打开要求源链接未变；
「写回并打开链接」仅可使用同一草稿成功 Flush 后核对的权威值。取消、
更换字段/节点和 Release 清理手势引用。完整目标最多4096字节，超限禁用，
提示副本最多512字节；Host.OpenReference 调用一次后立即返回，允许宿主
同步切换标签并释放当前视图。纯导航不创建文档事务，前置写回仍可能创建。

标记是graphview自有UI，不扩展外壳的具体格式命令。既有224DIP滚动检查器
增加默认收起的19字形面板，原生按钮带中文名称/Checked状态；只有优先级
与色标各自互斥，未知字形顺序与重复项原样保留，全清保存显式`[]`。
当前/候选列表先检查64项及4096 UTF-8字节预算，摘要最多512字节；纯格式
`Icons/SetIcons`不包含词表或这些UI上限。面板静止状态按版本缓存，不循环
请求帧。折叠、滚动/取消、切节点与Release清理手势。

pointer/key down记录当前Buffer/解析版本、节点ID、icons存在性与有序值；
源码已变但尚未Draw时拒绝旧显示目标。先Flush独立字段，再以新解析验证
同一列表，不跨Flush保留节点指针。候选模型序列化、重解析与文档上限通过
后一次提交并保留合法源码选区；需夹取时用标记专属提示。前置Flush仍可
产生自己的历史。未知成员Raw数值、标签/图片附加字段保留，真实图形编辑
仍会规范化JSON排版。画布与SVG共用512字节有界标签并做XML转义；完整
主题/标记/标签需在源码查看编辑，节点尺寸与布局不因本轮改变。

标签使用独立输入与默认收起的滚动面板，不扩展外壳的格式命令。纯API
`Tags/AddTag/RemoveTag/ClearTags`按顺序复制、追加、移除并保留对象/未知Raw；
新标签不trim、拆分或去重。视图在物化输入前检查LenBytes≤512，再检查
有效UTF-8与无CR/LF。当前/候选至多64项，保留内容预算8192字节逐项累计
`1+len(Raw)+len(Text)+len(ClassName)+ΣStyle/Extra(1+len(Key)+len(Raw))`，
在克隆前以及自身字段写回后的新Raw上重算，序列化/重解析后再验候选。
纯格式层不携带UI上限；超限拒绝不截断输入/源数据，全清也不能绕过。

公共Flush的pending-tag检查先于重入早退，Add直接调用私有ordinary helper，
不设任何全局绕过标记。手势捕获源Buffer/版本/选区、节点、完整有序标签
及存在性、两份输入身份和生命周期。字段写回只接受0/1版本的自身receipt；
Host.Changed后立刻证明确切序列化字节、Buffer/草稿owner/input版本/选区、
节点/字段/页面/epoch仍相同。随后重新解析，不把节点指针跨Flush保存。
合法字段写回与标签提交分别记历史；后者拒绝明确报告字段已写回。

selection/field/section折叠、离开再返回、scroll/cancel/Release使旧手势失效；
按钮拒绝重复按键和连续多击。discard/recovery同样不重定向到新输入。
成功Changed可同步切tab/Release，清理仅限已应用的原输入，不复活缓存或
覆盖替换输入。恢复同时捕获两份原文，普通field-only信封保持兼容；宿主
同步新建tab后也只清理捕获的旧草稿。未提交标签仅在运行窗口持有，不进入
会话/崩溃快照，显式恢复受宿主既有4MiB预算约束。

普通和tag输入的F6/Tab/Shift+Tab在组合期间保持焦点并显示本地指导。确认后
只清理确认sentinel；真实验证/预算/冲突错误仍上报，清空或放弃只解除已解决
的pending提示。临时指导不写入本次运行的Host.Report问题历史。普通错误行的
紧凑恢复按钮和标签区全宽恢复按钮有不同可访问名称、共享身份守卫。
app为格式warning行/split容器/具体tab视图设置稳定Key：首次条件warning的
出现或消失不能借由MyGo兄弟序号变化重建输入和滚动身份。

draw.io 保留原有标签输入和页面身份，不使用导图字段选择器。拖动等暂态
几何以每次释放为一次文档编辑；来源变化会取消未完成手势并提示。

Markdown 任务方框使用原生 Checkbox，状态每次来自格式层 TaskMarker。
单次动作捕获 Buffer/版本及整项、标记的数字范围，先 Flush，再映射并用
当前有界解析核对完整上下文；在目标前插入未闭合围栏也会被拒绝。最终
只替换中间一个字符，以新事务保存反向选区和撤销，不序列化整篇文档。
当前虚拟列表构造使用不可变解析/行快照，回调不会清空正在遍历的数组。
仅已验证的自身切换可以保留该行控件身份，连续空格不会丢焦点；同名标签
不参与身份判断。干净独立草稿跟随无关源变化时会刷新预览并脱离旧块下标，
因此 Undo/Redo 不会被仍打开的字段冻结；冲突草稿继续保留恢复路径。

Markdown 管道表沿用同一 List 和局部草稿生命周期。解析器保存表头/正文的
物理行与单元格字节范围，分隔行只映射显示位置；视图把每行摊平为稳定
source-range 身份，不保留可变的行下标记号。实际可用宽度至少为每列
112 DIP 时显示等宽网格，否则显示带行号/列名的堆叠字面记录；未知首帧和
尺寸变化首帧先选堆叠，一次布局变化只请求一次收敛帧。单元格最多三行并
显示省略，完整值只在最多 512 字节的安全 tooltip 中，完整源文仍在源码/
整表编辑器中。没有全文档 RowWidth 或隐式横向平移。

表格单元格是明确可聚焦的原生 RoleColumnHeader/RoleCell 目标。双击或
Tab+Enter 捕获 Buffer/版本、稳定的预览行、行列和精确源码范围，先 Flush
已有草稿，再按当前完整解析重验；无关前插可重定位，表内改动或替换拒绝。
单格编辑复用唯一的独立 Editor，但输入只留在视图本地草稿，不持续写宿主
Buffer 或历史。Finish、Save、导出、模式/标签切换和关闭统一走 Flush；
成功候选只做一次精确 ReplaceUTF16 和一个 Changed，无改动完成不写历史。
结构管道、换行或预算失败保留同一草稿、选区、历史和恢复入口；可直接继续
修改、显式放弃或打开未同步副本。未写回的本地值不进入会话/崩溃恢复。

选中单元格面板的插入/删除行先以`PlanTableRowEdit`对权威源码规划精确
字节补丁；表头插入位于分隔行后，不能删除表头。候选完整解析还比较所有
其他块/表，防止全篇描述项预算在未标记Partial时改变另一表的回退状态。
删除释放预算而使别的表恢复预览也保守拒绝。其他原文不重新序列化。

所选列对齐以`PlanTableAlignmentEdit`从权威源码重解同一表及分隔行，复用
既有scanner，只替换目标格去填充后的标记，保留短横线数并调整两端冒号。
语义相同返回Noop，不构造候选；真实修改先检查增长预算，再完整解析候选，
核对所选表/其它块和表的语义、范围平移、警告及全篇描述项回退状态。

`table_actions.go`是行/对齐唯一的手势、Flush receipt、提交与完成管线，
各动作仅提供planner和按钮。手势绑定草稿/单元格对象、Buffer、源与输入
版本/选区、Release世代、原表/行及列/对齐身份。自身Flush只接受没有写源的0版本或恰好一次单元格替换的
1版本，并核对预期候选和正常写回后的折叠选区；任意Changed回调改源、
选择、释放或替换都不能把命令转给新目标。单元格写回独立撤销，随后
行/对齐补丁与映射的有方向源选区另起一个自有事务；不取消外来事务。后续拒绝
明确显示单元格是否已经写回，只有尚未成功写回的草稿才保留恢复入口。
成功/对齐Noop由正常Flush完成面板，不重建旧draft；干净Noop无写入/history/
Changed，有待写单元格的Noop只留下其自身编辑。错误仍报告Host，但local
error只有捕获source/lifecycle及预期版本/选区一致时才更新；未完成draft还
须保持输入版本/选区。Report后的同步释放、替换或原位输入改动不接收旧状态。

单格输入采用固定48px编辑区和固定30px提示位，避免结束组合时移动已按下
按钮。标题只读有界descriptor中的所选列/当前对齐，源版本变化显示待重验，
idle不整篇parse。四个46×26px对齐按钮占196px，短face保留准确可访问名；
完成/放弃/恢复与行按钮同样紧凑，表头禁删说明留在固定提示位。F6在无组合
时把焦点移到行操作，再Tab到对齐；组合中保留焦点/草稿并请求先确认。
Tab/ShiftTab遍历按钮，Enter/Space执行，repeat不重复提交。祖先观察
backend原始ClickCount，在MyGo为新命中控件重算前拒绝打开面板手势的连续
多击；实际滚动handler取消捕获后返回false，原滚动继续。Release清理捕获
和焦点handle；稳定帧不持续Invalidate。无共享编辑器或vendor修改。

「编辑表格源码」、聚焦 Enter/Space 和表格行的双击都在手势开始捕获
Buffer、版本、稳定行身份、整表 UTF-16/字节范围及完整有界源文。实际激活
先 Flush，再按范围日志映射，并核对当前完整解析仍是同一个 `BlockTable`
及相同源文。无关前插可以重定位；表内改动、替换/过期 Buffer、异组事务、
Partial 或超限可见拒绝。失败保留同一 View、草稿、选区、历史和恢复控件。
只读表只显示字面预览和「只读表格」说明，不创建草稿。整表源码编辑允许
用户主动破坏表结构，保存后预览重解为明确回退，不在第一切片阻止管道或
换行。

表头/分隔行加零正文行仍有效。解析上限为8列、256正文行、每单元格
4096 UTF-8字节、32 KiB完整表和每文档16,384个cell描述项；失败候选只做
一次完整 `Raw` 回退。`Block.Raw` 有意保留权威源文，因此超过32 KiB时预览
不物化行列布局，而是有界源码提示和现有整块源码入口。HTML 导出复用相同
范围校验，只输出转义后的字面单元格/完整回退原文，不激活链接、图片或
HTML。反引号不保护未转义管道，也没有完整 GFM、排序/合并工具、公式或嵌套表
承诺；行/对齐/列操作共享下述手势与事务管线。

表格第一切片与单元格编辑都已在冻结源码上通过完整验收，其产物分别保留。
历史行操作`.3`宿主13,591,552字节（SHA256`61512C03…`）与worker
3,295,744字节（`6CFE7157…`）已接受；完整Markdown视图/app、consumer vet、
两张760×480测试器图、新包真实宿主路径的一次测试/两次worker请求和12/12
核验通过。1025输入、15657工具链文件、179旧产物/证据保持。此前`.1`全量
及生命周期验收、`.2`生成器/worker验收仍保留，本轮没有重跑无关套件。
`.1/.2`固定解析夹具的累计分配约减少80%/66.8%、转义样本中位时间约增加
12.8%，与另外记录的`.1/.2`进程峰值对照不同，均不能移用为`.3`资源测量。
精确范围见[验证记录](VALIDATION.md)与[行操作交接](handoffs/markdown-table-rows.md)。

历史列对齐`.5`完整格式/视图/app、scoped vet、独立源码审查与两张760×480
分屏图已验收。实际preview264px、输入226×48px，动作最下沿443.82px在
真实status451px之前。新包一次HostPath/两worker请求及12/12核验通过，
19文件17,038,894字节；1037输入、完整工具链与530旧文件不变。没有新的
物理UI/OSIME或资源比较，详情见[列对齐交接](handoffs/markdown-table-alignment.md)。

行内链接用公开的 RichText 子文本和 RoleLink，父行 ObserveInput 在命中
测试前记录修饰键；普通鼠标仍命中可选择段落，Ctrl/Cmd 才命中链接。
激活只使用手势时的 Buffer/版本、包围范围、语法范围和有界完整 href，
Flush 后重解并检查当前上下文。无捕获/已取消手势不回落到新绘制目标。
宿主检查来源标签仍存在且当前激活，再执行本地引用策略；视图不自行
分类、打开浏览器或操作磁盘。导航没有文档事务，前置 Flush 仍可能编辑。

显式图片引用复用同一条手势/来源身份纪律，但不复用普通链接的角色。
`![label](target)` 保持原文显示并可聚焦；Ctrl/Cmd+点击或Tab+Enter激活。
格式层只识别无标题的内联形式，目标为1–4096 UTF-8字节且排除ASCII空白/
控制字符、尖括号和圆括号；本地相对路径、百分号编码和Windows反斜杠原样
进入宿主。激活仍先Flush并重验当前上下文，再调用现有本地打开路径。外层
普通链接包住图片语法时不提供独立图片动作，保留原链接行为。

每行最多16链接且64行内 span 才构造行内控件。更密集行完整格式化显示、
去掉链接下划线与交互并在上方说明源码入口。依据是固定 MyGo 版本的
下划线绘制每段重复 DecodeSpans 的实测开销；不通过修改依赖或截断目标
规避。正常链接前景色独立于按钮 Accent，明暗主题背景/Surface 对比度
均验证≥4.5:1。同步打开文档可能在当前 List 构建中 Release：释放时交换
ListState 指针，让当前不可变行快照和旧状态存活到该帧结束，下一帧使用
新状态，避免清空正在使用的 MyGo 状态。

doc-block 引用复用同一条“来源身份先于宿主动作”原则。格式层的
`HeaderRange` 是属性区的精确 UTF-8 半开范围，`ReferenceProblem` 只描述
结构可用性；原生视图再要求显式 `mode: reference` 与非空 `src`。按钮在
pointer-down 或聚焦 Enter/Space 时捕获 Buffer/版本、不可变预览行 ID、
整块/属性区 UTF-16 范围和有界完整目标。释放保留按压时的行快照；
激活先 Flush，再把范围交给 `MapRangeSince`，只在当前完整解析上核对
同 Buffer、精确范围、doc-block 类型、空 `ReferenceProblem`、模式和完整
`src`。无关插入可映射；目标、引用头或构造改变、历史过期、Partial、超限
和异组事务拒绝。相同 ID/title 不参与目标选择，不保留 Block/Document
闭包、原始头副本或第二解析器。

引用目标的完整副本最多4096 UTF-8 字节，超限在克隆目标前禁用；
标题/kind/mode 的显示副本分别以256/64/64字节有界，tooltip连同前缀和
省略号总计最多512字节。只读导航允许且不增加历史，前置 Flush 仍可能提交
草稿。`Host.OpenReference` 是唯一分类和打开出口，只调用一次；调用后不再
读取当前视图，因此宿主可同步切换标签并 Release。目标策略仍是保存目录内
的字面相对文件、本地绝对路径、解码一次的本地 `file:` URL 或当前文档
锚点，不访问网络、UNC 或远程 file URL。普通 inline 无 src 即使没有解析
问题也不显示打开按钮；结构无效的块同样没有打开按钮。

最终点击行来源守卫、完整test/vet/build和隔离隐藏启动关闭已执行并独立
核验，根代理已接受本轮exe；18:17字段版转为上一验收产物。
证据与精确边界见[引用集成交接](handoffs/docblock-reference-integration.md)。

`Release` 表示释放重资源，并非销毁文档。应用在暂时切走视图时保留轻量
View 对象，图形选中项、页码、平移缩放等可继续使用；解析模型、可见列表
索引和标签排版缓存释放。文本缓冲和撤销属于文档，不随预览释放。

## 大纲与定位

`Adapter.Outline func(Host) OutlineSource` 为 Markdown 与导图提供轻量提取。
它只读 Buffer，按指针和版本缓存描述项；源码模式不调用 `Adapter.Create`
或构造图形布局。快照最多 2048 项、显示文字最多 512 UTF-8 字节，调用方
取得自己的切片，字符串只保留有界副本。错误、有效空集和不完整快照分别
表示；未支持的格式没有 factory。切走/重配时释放缓存。

Markdown 大纲身份包含 Buffer 身份、源版本和整段标题 UTF-16 范围。
`ResolveOutlineItem(oldID)` 用范围日志映射，再核对当前解析器的标题范围，
返回新的当前条目。它不以标题文字、slug 或序号查找，因此在前面插入重名
标题不会把旧点击重定向。`AnchorID` 则是当前文档的链接 slug，与 HTML
导出的标题 ID 共用生成器；它不作为跨修改身份。导图直接使用文件节点 ID，
没有可信源码偏移时返回 -1，没有 Markdown anchor 时返回空字符串。

应用先 Flush，再校验标签、适配器、路径、模式、Buffer 与控制器代数，
重新解析捕获的旧 ID。源码 Markdown 只 Reveal；阅读模式调用可选
`OutlineNavigator`；分屏先导航预览再定位有效源码偏移。导图分屏保留
源码选区，源码单模式明确提示切换后定位。视图还会防御性 Flush 与重解，
失败保留局部草稿。旧 `OutlineProvider` 接口仍保留兼容。

导图定位隐藏节点仅展开折叠祖先，放进一次文档事务，保留未知字段；
选择根节点本身不展开。只读隐藏目标在修改选择、平移或布局之前失败。
展开事务保存原数值 UTF-16 选区，并让 Undo/Redo 一起恢复；序列化后文档
变短或数值落在代理对中间时夹取到前一个合法边界并提示。它不宣称把源码
选区语义映射到重新排版后的相同 JSON 字段。

侧栏使用虚拟列表，缓冲版本不变时复用快照。`CurrentOutlineProvider`
只在快照刷新且没有稳定选中项时提供提示；列表高亮是选择/导航状态，
没有连续的源码光标跟随订阅。

安装格式的源码大纲与富视图分开：`Tab.OutlineFactory` 由
`modules.InstalledSourceOutlineFactory` 组合，`tab.Adapter` 保持 nil，
所以没有伪造的阅读/分屏/导出能力。`core/plugins.OutlineRule` 只在这层
映射为中立的 `formats/sourceoutline.Rule`；`modules/sourceview` 负责
通用 OutlineSource 契约，既不导入插件注册表，也不挂载图形。

纯扫描器从 `io.Reader` 固定块读取，最多解释 2 MiB/65,536 行；保留有界
前缀、标签和后缀状态，不缓存整行。Provider 从 Buffer 的二分索引
`SliceUTF16` 读取小窗口，处理代理对端点，无全篇 String 或 Snapshot。
固定 4096 字节缓冲可能预读过行/项数上限，未解释数据不生成条目；总读取
上限 2 MiB + 1 字节。一次极小探测可在 Provider 内复制两个 UTF-16
单位（最多 6 个 UTF-8 字节），只返回调用方要求的字节。

规则遵循 LF 物理行；匹配前只删除终止 LF/EOF 前的一个 CR，内部 CR 保留。
预算截断的半行不输出。旧条目还绑定插件 ID/schema/有序规则的 SHA-256，
规则刷新不能将旧范围重新解释成另一项；宿主的 factory/controller 代数
另外拒绝已失效的异步或 Flush 跨越上下文。规则不识别多行语言上下文，
界面持续说明可能命中注释、字符串或块内的同形文本。

## 文件与恢复安全

普通保存检查磁盘内容哈希、文件身份与元数据基准，再原子替换目标。另存为
和导出统一走不覆盖已有目标的发布入口。读取过程中检测文件或路径变化；
编码不能无损转换时返回错误，原始字节不会被猜测后覆盖。

脏 UTF-8/UTF-8 BOM `Save` 和 `SaveAs` 从现有 `Buffer.Snapshot()` 经 32 KiB
写缓冲送入临时文件和 SHA-256 writer，不再先取得整份编码字节。共享
`writeAtomicWriter` 负责临时文件模式、写回调、`Sync`、关闭、发布前检查、
原子/独占发布和延迟清理；UTF-8 helper 只有在整个发布生命周期成功后才返回
摘要，任何失败都返回零摘要和保留的错误。`didSave` 随后重新 stat 并完整读取
已发布文件，只有读回 SHA-256 等于预期输出摘要时，才更新磁盘 token、已保存
快照、编码/BOM元数据、基准版本和脏缓存。`SaveAs`保留`.14`的
Path/IsNew/ReadOnly发布后时机，所以发布后校验失败不承诺回滚。UTF-16、GBK
和GB18030仍使用原有整缓冲编码及`writeAtomic`路径；快照描述符及不可变底层
分块仍会在保存期间保留，32 KiB缓冲只界定编码输出阶段。

后台检查在 UI 线程捕获不可变 `document.DiskToken`，工作线程只读取 token，
不触碰可变 `Document`/`Buffer`。结果回到 UI 后再以 `SameBaseline` 核对。
成功保存或恢复会推进基准代数，即使字节相同也不能接纳旧任务结果。
后台检查是提前提示；保存前完整校验仍然负责最终冲突判定。

目录和磁盘检查任务各自只运行一个工作项，并合并为最新待处理请求；切换
根目录、关闭和新请求使用取消/代数隔离。文件监视失败、截断和读取失败
进入用户可见问题记录。工作区搜索只做明确的目录排除和符号链接跳过，
尚未实现旧版的完整 `.gitignore` 解释。

会话使用 `SnapshotBuilder` 在生成整份草稿字符串之前计算 JSON 编码预算，
包括引号、控制字符、HTML 转义、元数据和偏好；拒绝不合法或过大的输入时
不消耗已有预算。现有 API：

```go
builder := session.NewSnapshotBuilder(root)
err := builder.SetPreferences(preferences)
err = builder.AddDocumentWithOptions(doc, session.CaptureOptions{
    Title: title, Scroll: scroll, OriginalPath: originalPath,
    PluginID: pluginID, View: viewMode,
})
err = builder.SetActive(activeIndex)
err = builder.Save(sessionPath)
```

这是调用顺序示意；实际宿主每步检查 error，失败不进入下一步。
`AddDocument` 是精简入口，`AddTab` 用于已有记录或已明确丢弃草稿的记录。
`Snapshot()` 返回独立可变容器。读取会话最多保留前 100 个数组项并统计
省略数，不为任意数量的后续标签创建对象。草稿恢复成新的未保存文档；
`OriginalPath` 只用于说明来源，不授予覆盖原路径的身份。

### 最近关闭与标签准入

`Ctrl+Shift+T`和原生菜单只保存本次运行中最近单独成功关闭的16个已保存
本地文件路径及接收标识，不保存正文、图片字节、撤销、选区或草稿。重开
重新读取当前磁盘和当前格式/插件配置；同一路径再次关闭会移到顶部，缺失、
不可读、Flush失败或界面采用失败会保留对应记录，允许用户改选或显式重试。
重开不恢复已放弃修改、未保存文档或恢复草稿；整个窗口关闭会取消待处理
重开，也不会把当时全部标签误记为最近关闭。来源与接收身份在同步/异步
打开、Flush和最终采用前后重新核对，旧结果不能消费后来的记录或改到别的
文档。

`session.MaxTabs`为100，会话快照的标签容量与运行时准入共用这个值。新建、
模板、恢复草稿、普通/批量打开、字节回退、异步二进制采用和启动会话恢复
都经过同一标签数量检查：已有路径先复用，满额只拒绝真正的新标签；批量
打开记录拒绝后继续处理后续项。尽早在IO/模板/Flush前检查，并在实际
Flush或异步回调后再次检查，避免旧准入结果越过标签变化。拒绝不逐出或
清空既有标签，也不改写源内容、版本、撤销、选区、恢复文件或二进制
所有权；会话恢复按顺序接受恰好100条记录。这里限制标签数量，不提供活动
内容或进程内存预算。

会话写盘在`preflightSnapshot`完成后，把已校验的Snapshot与精确输出长度交给
`core/document`的私用状态回调写入接缝；接缝复用既有临时文件权限、同步、
关闭、原子替换、清理和发布点。`core/session`用标准`json.Encoder`经一字节
待处理适配器输出到该接缝：运行期核对预检长度、拒绝短写和溢出；除最后一个
待处理字节外，其余字节交给底层writer；`Finish`要求该待处理字节正是Encoder
强制的LF且总长度等于预检长度加一。运行期不显式检查首个字节，也不做oracle
哈希比较。测试另行把完整输出与`json.Marshal`oracle逐字节比较，证明字段、
顺序、转义、`omitempty`/nil及无末尾LF格式相同。

预算与快照校验先于目录或临时文件效果。临时文件写入、sync/close、发布前
check、`publishNew`或`replaceExisting`任一失败都发生在提交点前，保留旧恢复
文件与时间并按既有生命周期清理临时文件。发布成功就是提交点；随后
`syncDirectory(dir)`仍可能返回错误。Windows当前把该目录同步实现为no-op，
Unix可能失败；一旦发布后同步失败，保存返回错误，但新文件已经发布，不会
回滚。
适配器记住第一个writer/finish错误，后续操作返回同一错误且不再写入sink；
这是辅助合同加固，不暗示当前生产调用方曾丢失数据。Encoder在Go 1.27仍
内部构造完整JSON值，脏草稿的`Buffer.String()`捕获也仍在，所以该边界只
删除了`json.Marshal`额外返回的编码副本，不是token流式编码、固定内存或
整次保存/RSS/工作集测量。源码验收、结果和独立核对见
[源码验收](handoffs/session-output-root-acceptance.md)、
[实现结果](handoffs/session-output-implementation-result.md)和
[独立核对](handoffs/session-output-independent-reconciliation.md)。

## 资源工作范围

原生版以低内存和低 CPU 为目标，主要策略是限定输入、限制累计工作和只
构造可见 UI。Markdown 解释超限以 `Partial` 明示并拒绝 HTML 导出；源码
仍完整保留。Markdown 预览虚拟列表、图形视口剔除和有限 LRU 标签缓存减少
未变化帧的工作。极小图形标签隐藏并显示说明；超长文本通过有界分段访问。

文本编辑器的源文件读取上限仍是128 MiB。`.28`在`core/text`另设普通UTF-8
逻辑内容的默认128 MiB基值，并在可复用构造器上保留不可变基值。缓冲区有效
高水位为`max(基值,初始或Restore种子字节数)`：较大的种子不截断，缩小后也不能
降低该高水位，撤销/重做/取消组按已接受历史重放。显式Restore在保持缓冲区
身份的同时用目标基值和来源种子重新设置高水位。该设计允许单个已解码文档
超过128 MiB，所以128 MiB不是全部文档、RSS或工作集上限。

替换先验证UTF-8、UTF-16范围/代理对及使用`byteOffset`计算的前瞻字节数，
在复制删除片段或写历史前拒绝；稳定错误可让源码/格式视图显示拒绝。ReplaceAll
按实际反向编辑顺序预检整个组，缩进先累计正增长，都在`BeginGroup`前拒绝。
因此反向中间态超限时可以保守拒绝一个最终大小可容纳的批次，且不留下部分
命令。普通编辑失败不回滚此前已接受的事务；既有格式多步操作的修订语义不被
扩大为“所有失败修订不变”。

源码输入的大小拒绝若发生在IME组合期间，只保留有界错误与所有者元数据，不复制
被拒绝字符串。未完成的首个或进行中组合使`Flush`持续返回同一大小拒绝；
只有接受的有效更短重试或编辑器中明确的Esc取消才放行保存、切换和关闭。
焦点丢失、导航、删除/复制/撤销和视图释放引起的`UnmarkText`不能静默清除该
屏障，重新聚焦后可重试或取消。普通粘贴拒绝不是持久的IME屏障。格式本地草稿
被写回拒绝时继续保留，可缩短后重试或另行恢复。

`.29`把连续完整IME标记范围的更新记录为Buffer拥有的类型化组合token；token
只持有事务身份，不把另一Buffer或重置后的事务当作当前历史。陈旧、跨Buffer
或重置身份不能修改或取消外来事务，无token的类型化更新/结束/取消请求会拒绝。
连续完全相同的完整标记范围更新只保留原始与当前后缀端点，不切片复制中间旧
文本；净零组合清理死引用，不制造空撤销项，也不删除此前已有的重做。相同文本
的首/末输入和既有已接受标记回调保持原行为。非连续或不规则显式范围继续使用
普通后缀历史，可能随编辑次数增长。直接重置时拒绝外来token属于防御性覆盖，
不是新增支持的UI序列。这里的“端点保留”不是通用活动历史或进程内存上限。

4 MiB单草稿、16 MiB编码后会话、32 MiB历史/图形解析、100标签以及各worker
输入/输出预算继续是彼此独立的边界。已进入进程的剪贴板、正则展开、编解码
临时串和活动组历史仍占内存；128 MiB内容准入不能被表述为进程内存有界。

这些措施未消除所有线性工作：图形解析/布局与节点数有关，边列表仍需
遍历；接近解析上限的文档未做完整交互验收。测量的环境与口径见
[验证记录](VALIDATION.md)，不据测试器分配量推导运行进程内存。

宿主解析的富视图先用 `Buffer.LenBytes()` 对照 `markdown.MaxParseBytes`、
`mindmap.DefaultMaxBytes`、`drawio.DefaultMaxBytes`，再允许整篇
`String`/字节转换；Markdown 打开旧块前的前缀读取也在检查之后。
超限拒绝按 Buffer 指针和版本缓存，清理旧模型、行、布局、标签及大纲
描述项。独立草稿及其字段基线仍保留，不能据此声称视图所有内存均已释放。
缩小内容、撤销或换成新 Buffer 后可重新解析。独立只读sceneView先查同一
源预算；更新失败时可保留旧场景并标为stale，显示明确说明且拒绝旧导航，
不把旧场景当成新源码结果。

Markdown Flush 只按已验证的字段范围写回，不因预览超限而禁止安全的局部
同步；导出随后仍拒绝超限源码。图形 Flush 需要解析来核对节点/页面，
所以超限时先拒绝复制并保留草稿。现有 View.Warnings/Draw/Export 负责
说明和错误，外壳无需另行猜测格式上限；导出失败不产生输出字节。

## 平台与第三方边界

当前 Windows 字体路径由 MyGo 使用 DirectWrite，图形后端使用 D3D11。
Windows 包依赖图同时含 MyGo 的 `internal/webview2`、`internal/update`、
`netproxy` 等共享实现；这不改变编辑界面运行在 Surface 的事实，也不能
作为这些功能在运行时被启用或被链接器完全裁掉的证明。

LevNote 没有启用 MyGo 更新器。vendored v0.8.3 的唯一启动补丁移除自动
清理可执行文件旁 `.*.old-*` 的调用，避免影响同目录的无关文件。
`go.ps1` 的构建保护和补丁回归必须随依赖升级维护。

应用的文件/图像引用路线不请求远程内容，安装配置没有可执行能力。
未进行完整 OS 级网络跟踪，也未穷尽 Windows 输入法、辅助技术和其他平台。
许可证、派生来源以及应随分发保留的文本见
[第三方声明](THIRD_PARTY_NOTICES.md)。
