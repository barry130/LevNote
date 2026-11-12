# LevNote native · 自主开发交付就绪度审计

日期：2026-11-12。审计者：native-readiness。任务：task-7。性质：只读产品源码审计与后续工程清单，不是产品测试、完整安全证明或发布验收。

## 1. 结论与授权边界

**普通开发 GO；当前 application-deliverable NO-GO。** 已经存在可用的四格式原生应用骨架和大量真实工作流，不是从零开始，也不能把“全部后续开发”解释成无限复刻商业编辑器。必须补齐已支持命令的实际行为、编辑闭环、失败可携带性，以及具体有界工作／句柄生命周期缺口；Windows 最终验收后移，不成为这些实现的暂停理由。

依据：[最新原生协作规则][agents-native]、[自主交付计划：17–26][delivery-scope]、[审阅与收敛：44–50][delivery-gates]、[迁移决策][migration]。原始四类文档、纯本地、保真和不静默要求来自[根上手手册][agents-root]与[格式规范：170–177、270–324][format-spec]；旧 Tauri/Milkdown/mind-elixir 实现细节仅作产品行为参考，不是重引入框架的授权。

本次仅写本报告。未执行任何 Go/test/vet/build/gofmt、产品运行、静态验收、远程命令、委派、提交或推送，未改产品／测试／全局规划／历史证据。审计期间 app-list 所属源码正在由另一明确 owner 修改；引用行号是对应读取时的位置，稳定定位应同时使用所列函数名。没有冻结当前树或取得新的源文件摘要，也没有声称已审阅并接受这些并行补丁。

### 已回传并获 lead 决定全部补齐的五项 required 缺口

1. **P1 draw.io Delete/Backspace/native delete 被消费却不删除。** 有添加、移动、连接工作流，却缺格式层删除 API 和原生删除入口。
2. **P1 Markdown 阅读编辑缺主文档用户 Undo/Redo 入口。** 已有可撤销事务，但测试直接调用 Buffer.Undo 不等于用户能在完成编辑后的阅读界面撤销。
3. **P1 功能缺口：本地问题清单缺 Copy All；附带 P2 偏好动作标签反向。** 复制是原始纯本地支持工作流的明确要求，不是新增联网诊断。
4. **P1 普通历史的 Truncated 缺 UI 披露。** 256项／32MiB 的既有预算保持，不能只有“未保存”而让用户不知道较早或本次 Undo 未保留。
5. **P1 Save 同步 fingerprint 缺 expected-length+1 字节工作边界。** 修复不得削弱 exact digest／文件 identity／编码+BOM输出／发布后基线合同。

前三项已优先回传；后两项随后回传且 lead 已明确接受为 required 收敛工作。对应共享任务板：draw.io task-8（blocked by task-7，新的显式 assignment 后执行）；Markdown task-9（blocked by task-6）；Copy All／设置标签／history notice 合并 task-10（experience 完成 task-6 后）；Save fingerprint task-11（reliability 完成 task-5 后）。第6节 T01–T10 为审计工程序号，不是任务板 task-1–task-10；T05与T06属于同一个 task-10。

app UI 文件仍由 native-experience 独占，报告不授予任何新写权限。**后续产品源码和测试的语义实现 origin 仅在既有授权 main box 的新独占普通 source work；本机仅只读／handoff／镜像 box 最终原字节供 lead Git review，不做本机语义改源或 Go fallback。native-reliability 仍是唯一 Go／format／test 命令 owner，按匹配源身份统一 capture。** task-8 预定 source work 为 `/home/user/levnote-dev-drawio-20261112-01`；从普通 clone/snapshot 初始化、关闭 hooks，不碰旧 guard。task-7 在本机完成只读报告后仍须等新的 task-8 authority 才能切 remote。

上述五项不得用安全源码退路取消；G3的 list 生命周期与G7的路径修复是现有工作方向，另作明确 scope／验收收口。其他高级 legacy 能力按第5节明确保留边界，不作无限 parity。

## 2. 判据：能力、证据和严重度不可混用

- **S（source）**：本次读取的实现和现有测试源码所表达的契约。只能支持“有实现／有测试”，不能支持“当前源执行通过”。
- **H（historical）**：已闭合的旧版本执行和验收。仍真实有效，但仅限其源／包／命令身份，不转移给新树。
- **B（boundary）**：已明确的原生有限子集。保留原文、可见限制和安全源／字节退路成立时，不自动变成 P0/P1 开发任务。
- **V（validation gap）**：当前源、真实平台、交互或包还缺相应验证；不是从“没验证”推出“必有缺陷”。
- **P0**：已证实的直接内容毁损／危险执行／普遍不可用阻断；本次未新增一个足以归为 P0 的源码事实。
- **P1**：当前交付要求内的关键工作流、保真／失败出口、生命周期或有界工作缺口。须在 feature-ready 前修复或由 lead 给出有依据的明确范围裁定。
- **P2**：真实但非关键的文案／入口一致性与文档问题；有承诺补齐的项仍须完成，不能假称已经修复。

安全 source fallback 可以合理保留高级编辑边界，**不能**替一个已被处理的 Delete 命令或已宣布“一步撤销”的基本编辑闭环辩护。反之，缺任意高级样式／图片／关系线／IDE 功能，不能仅凭旧产品具备它就升格为 P0。

## 3. 需求 → 实际能力 → 源码／证据 → 差距 → 下一步矩阵

所有“已实现”均为 S，除明确标注 H 外不包含本次执行证明。表中 T01–T10 是第 6 节的依赖排序任务，不是未经 lead 指派就可开始的工作。

| 需求／来源 | 当前实际原生能力 | 精确源码与已有证据 | 缺口、严重度／边界 | 可直接执行的下一步 |
|---|---|---|---|---|
| 四格式一等文档；纯本地；迁移计划 21–25 | Go + MyGo Surface；项目自研 text、Markdown、mindmap、draw.io；新建四格式；无账户／同步工作流 | [app.go][app] `NewDocument`、[view.go：282–288][new-document-ui]；[contract.go][editor-contract]；[migration][migration] | S：四格式不是只剩源码文本。B：没有无限编辑器等价承诺 | 保留结构和原生输入约束；不重新排队“四格式新建”（已存在） |
| 打开／拖入多文件／同路径复用／新建／只读退路 | 有顺序打开、100 活动标签准入、同路径复用；未知编码／超限有只读字节检查；普通 I/O 错误可见 | [app.go][app] `OpenMany`、`loadForOpen`、`admitNewTab`、`addDocumentChecked`；[document.go][document] `Open`、`Inspect`；[README：352–356][capabilities] | S：100 tab cap、增长 cap 已存在。B：“打开不能失败”不意味着绕过权限或完整装载任意大文件。V：新树各入口尚未重新完整执行 | T01 核对本机路径语义；T08 覆盖文本／恢复／异步二进制／满额与后续项继续，不再重写 cap |
| 保存保真、只读、冲突、另存不覆盖；格式规范 §8 | 保留未动文件字节；UTF-8/BOM 流式写；其他支持编码无损写；双摘要检查；SaveAs create-only；Windows ReplaceFile 保留目标属性 | [document.go：194–310][save-contract]、[atomic.go：14–116][atomic]、[atomic_windows.go：14–46][atomic-windows] | S：关键保存保护已有。已承认非合作外部写者的最终检查／替换窗口不能由普通文件系统 API 完全消灭。**P1：fingerprint 整读当前外部文件，无字节上界** | T07 加基线／实际输出长度约束和增长检测；不以仅 size/mtime 代替 exact digest，不给 SaveAs 添加隐式覆盖 |
| 字符编码／BOM／混合行尾；格式规范 §8 | UTF-8、UTF-16LE/BE、GBK/GB18030 的受控重解释／保存；无法无损处理时拒绝；原字节退路 | [document.go][document]、[README：356][capabilities]；[view.go][app-view] `encodingChoices` 调用 | B：不是任意字符集自动猜测器；GBK/UTF-16 等整缓冲路径不是流式实现。未发现新证据证明已支持编码会被静默替换 | T08 纳入非法序列、不可表示字符、BOM／行尾／readonly 与 SaveAs 竞态；T09 文案精确匹配实际清单 |
| 关闭／重开／关闭取消／恢复草稿身份 | Flush 后关闭／切换、失败保留；近期关闭 16 条仅路径与 receipt；重开从当前磁盘读取，不复活放弃的改动 | [app.go][app] `CloseTab`、`CanClose`、`flushTab`；[reopen_closed.go：14–55][reopen]、[reopen_closed_test.go][reopen-tests] | S：不是缺重开功能。**P1：读取时 closedFilePath 只准 Windows drive absolute，真实 Unix 已保存文件不入历史**；旧 Linux 测试最终空 slice panic 不是普通用户生产 panic 的证明 | T01 已属 reliability：真实 Open→Close→Reopen、本机 case 规则、receipt 成败、满额保留；不能只给旧测试 `[0]` 加 guard 遮蔽失败 |
| 本地链接／锚点／doc-block／图像引用安全打开 | 主文档 source-path 上的本地引用；拒绝外部 scheme／网络 UNC／host URL；范围／版本／owner 守卫 | [reference.go：31–80、117–128][references]、[reference_test.go][reference-tests]；[Markdown view][markdown-view]；[README：358][capabilities] | **P1：读取时 `/` 本机绝对路径在早期 Windows 根相对拒绝分支被挡，Windows 词法与本机 filepath 混用**。B：拒绝网页／系统 shell 打开符合纯本地，不是功能失败 | T01 分离本机绝对路径、Windows 词法与 file URL；同时拒绝 encoded UNC、drive-relative、NUL、host/query/fragment，不以 cwd 猜相对引用 |
| 可恢复而不覆盖原文件；选择／滚动与偏好 | 有界 JSON 恢复、独立未保存副本、binary path-only lane；900ms quiet-time recovery，不自动写原文件 | [state.go][state] `Restore`、`Persist`；[session.go：19–44、91–118][session]；[autosave.go][autosave]；[autosave_test.go][autosave-tests] | B：100 tabs、单 draft 4MiB、JSON 16MiB；未写回本地块／单元格／新 tag 草稿不承诺 crash recovery；不是已丢原文。V：旧 autosave report-once/pending 失败族尚未在新树闭合 | T08 验证 quiet-time 不依赖重绘、失败闩与重试、上限时保留旧 recovery、独立副本身份、close cleanup 拒绝不丢恢复 |
| 自研文本编辑：输入／选区／剪贴板／IME／历史 | UTF-8 piece table、UTF-16 坐标、undo/redo、IME 连续端点合并、增长拒绝、长行分段与有界 shaping | [buffer.go：95–150][text-history]、[buffer.go：469–588][history-operations]；[editor.go][editor]、[composition_test.go][editor-composition-tests]、[growth_test.go][editor-growth-tests] | S：普通 history 默认 256 项／32MiB，已有 trim。**P1：Truncated 没有 shell/view 用户披露**。B：不规则活动 group 不具通用 cap；物理 IME 未据此验收 | T06 消费截断信号，不降低历史／内容上限，也不宣称普通 history 无界；T08 维持增长拒绝／Esc／Flush 屏障；T10 真 IME |
| code/text 的日常编辑与高亮 | 文本编辑全链；Go/TS/JS/Python/JSON/Rust/XML 单行词法高亮，本地 syntax JSON | [editor.go][editor]；[plugins.go][app-plugins] `configureTab`；[README：357、361][capabilities] | B：单行独立高亮，跨行串／注释可能不准；无 LSP、折叠、多光标、完整 IDE，不执行代码 | 保留明确 code 边界；普通编辑验证并入 T08。没有把“整套 IDE”列为交付 P1 |
| Markdown：读／改／任务／表格／源分屏／HTML | 常用块、inline 子集、任务切换、双击字面编辑；表格 cell、行／列／对齐；引用导航；有界 HTML | [markdownview/view.go：227–304][markdown-view]；[table_columns_test.go][markdown-column-tests]、[table_rows_test.go][markdown-row-tests]、[table_alignment_test.go][markdown-alignment-tests]、[tasks_test.go][markdown-task-tests]；[README：358][capabilities] | **P1：完成编辑回到 preview 后无主 Buffer Undo/Redo 用户入口**。现有 tests 直接 buffer.Undo 不能填补 UI。B：不完整 CommonMark/GFM/Mermaid/富文本 toolbar；source 可继续编辑 | T04 区分 local draft history 与 host history；native command／键／按钮可撤销任务／cell／结构；拒绝保留草稿和 selection |
| Markdown 帧／缓存／List 生命周期与超限 | virtual preview、rows/source identity 守卫；超 8MiB 先拒绝再物化，不整篇偷偷解析；local draft 保留 | [view.go：151–187][markdown-release]、[preview_limit.go：16–54][markdown-preview-limit]；[preview_limit_test.go][markdown-limit-tests] | **P1：Release:182、dropPreviewCache:48 每次新 ListState 指针放弃已经 bound 的 public Handle**；旧 frame 不能当场清零。不是已测得 RSS leak 数字 | T04 同 owner 顺带修 Markdown list-lifetime，保留 Handle identity + CancelFocus + 安全 reset 时机；不能直接覆盖 active build 引用的旧 ListState |
| Mindmap：结构、属性、标记、标签、导出／只读 | 新建、topic/note/local-link、子／同级／删除／reparent／promote、四布局、collapse、markers、tags、canvas、SVG；readonly scene worker 与节点导航 | [graphview/adapter.go][graph-adapter]、[view_ui.go：45–99、159–271][graph-ui]、[input.go：210–242][graph-history]；[mindmap/serialize.go][mindmap-serialize]；[README：359][capabilities] | S：图形 Undo/Redo 确实存在，不是 gap。B：未完整绘制／编辑图片、style object、关系线／summary；tags search/reorder 等未实现。未知字段保留不等于画出。readonly scene 不是可写插件 | 不无界迁移 mind-elixir UI。T09/T10 明确有限子集、保留未知／坏形字段与 safe source；若另扩高级内容，另立有界合同和 scope |
| draw.io：原生基本创建／移动／标签／连接／多页 | 四种基本形状、节点文本、节点拖坐标、连接、页面切换、当前页 SVG；格式层 edit+reparse 原子发布 | [formats/drawio/edit.go：28–208][drawio-edit]、[graphview/view_ui.go：101–148][drawio-toolbar]、[graphview/input.go：46–73、244–249][drawio-delete]；[drawio_test.go][drawio-tests] | **P1：deleteKey 对 draw.io 返回 true 却不操作，无 DeleteCell API 或按钮**。B：readonly compressed pages、complex shape/stencil/group 等已有限支持，不要求整套 draw.io 编辑器 | T03 精确删除选定 cell 和已知 incident edges（同事务且披露）；保护 0/1、readonly、未知 parent refs／raw XML；host 一步 Undo |
| 格式往返／unknown 元数据／不顺手美化 | 未修改时原 bytes；draw.io precise source patch 保留其他页／unknown XML；mindmap canonical edit 保留 raw extra，校验后发布 | [drawio-edit：14–19、197–208][drawio-edit]；[mindmap/serialize.go：9–57、71–133][mindmap-serialize]；[source_retention_test.go][drawio-retention-tests]；[README：382–384][format-boundaries] | B：导图结构编辑后 JSON 排版会变，已披露；压缩 draw.io 页图形只读而非“解压编辑并压回”；source 模式仍可编辑 XML。不能把上述描述当字节保真无限承诺 | T03 新删除延续 exact untouched ranges；T08 fixtures 覆盖 unknown attrs/elements、其他页面／compressed 页和 stale reparse atomicity；T09 区分源保留与 SVG 保真 |
| 独立本地 PNG/JPEG 与失败 byte fallback | 已有 bounded readonly reader、fit/100%/zoom/pan、keyboard、original-byte copy、binary restore、bitmap disposal | [imageview/view.go][image-view]、[imageview/input.go][image-input]、[imageview/imageview_test.go][image-tests]；[README：362][capabilities] | B：16MiB、8192 单边、8,388,608 pixels、baseline JPEG、8-bit/non-interlaced PNG；无图片编辑／SVG image route。不重新排队“图片查看器” | T08 binary reopen/adoption/readonly-copy 组合；T10 渲染／驱动真实核验。Markdown inline image 管理仍是明确高级边界 |
| Find／replace／workspace search／outline／picker／keyboard | literal/regex/case/Unicode whole-word；选区推进／wrap；有界 workspace results；Markdown/mindmap/schema2 outline；native document picker | [workspace.go][workspace]、[outline.go][outline]、[find.go][find]、[search/workspace.go][workspace-search]；[README：363–364][capabilities] | S：不是缺 Find/outline。B：graph Find 基于 JSON/XML，不是 node/note 专用；cold path 仍扫描文本；outline 2048 等 limits。**P1：app-list Handle 生命周期已有未完成方向**；V：min-window／native picker 新树需闭合 | T02 当前 experience owner 完成 handle/state/reset 修正；T08 真实 callback、selection、query guard、旧 frame、760×480 关键控制路径回归 |
| failure history／不静默／纯本地支持 | report 维护最近 200 字符串并计 dropped；issues 显示当前 ViewWarnings、SearchIssues／跳过数和运行期错误；“查看详情”真的能展开 warnings | [app.go：229–235、363–371][issues-model]、[view.go：525–560][issues-ui]、[app_test.go：1184–1200](<F:/AIproject/LevNote/native/app/app_test.go#L1184-L1200>) `TestViewWarningSummaryOpensFullDetails`；[ADR0046：99–112][issues-requirement] | **P1 功能：没有 Copy All 控制或可携带 snapshot**。不存在“warning详情只打开空列表”缺口，读取到完整循环与测试。B：不是新建联网／自动上传／永久遥测；native 当前没有完整 legacy grouped health 状态模型 | T05 有界脱离 UI 可读 snapshot，带应用／时间、数量和 dropped/truncation；成功回执前不报 copied，失败本地可见；不伪造并不存在的每条历史时刻／恢复状态 |
| 用户常用偏好与诚实入口 | 设置可调明暗／字号／行号／wrap；source Editors 应用 shared preferences；session 存 sidebar width 与位置 | [view.go：363–403][settings-ui]、[view.go][app-view] `applyEditorPreferences`；[session.go：25–31][session]、[state.go：32–44][restore-preferences] | **P2：lineLabel/wrapLabel 的动作方向与当前布尔反向关系写反**。B：有 sidebarWidth 存储不等于有用户 drag/settings 入口；未原生迁移全部国际化、字体族、专注/打字机/position opt-out。不能假称全 legacy settings 完成 | T05 修两种当前状态下的 action text 和值/persist；T09 精确列默认可用 prefs 与未迁移项；额外入口由 lead 有界选取，不私自扩 schema |
| plugin 配置安全／失败明确／可继续 source | schema1 syntax、schema2 literal outline、schema3 fixed native route；install/enable/disable/rescan；受控 template 优先级；JSON 不执行代码 | [plugins/registry.go：312–455][plugin-manager]、[declarative.go：59–107][plugin-limits]；[plugins.go][app-plugins]；[README：365][capabilities] | S：已有 bounded install 和失败 rollback；不将“enable 后未加载成功”的显式报告误当全部插件 no-op。B：没有 executable plugin editor API，符合当前安全范围 | T08 preserve malformed/oversize/duplicate/shadowed route 的原始错误与 source fallback；T09 确保菜单仅公布真实能力 |
| 四 fixed workers／本地 package／owner/join/poison | 固定 HTML、mindmap SVG、readonly scene、draw.io current-page SVG；source/payload cap；单 runner admission、30s、失败清 payload、cleanup poison、shared Close barrier；Windows Job 所有权 | [catalog.go：9–63][catalog]、[protocol.go][protocol]、[runner.go：79–181][runner]、[managed.go：9–36][managed]、[process_windows.go][process-windows]；[README：366][capabilities] | S：不是缺第四 worker。V：当前源真实请求／包切换／清理仍需对应验收。B：哈希不是签名，Job 不是已证明网络隔离或全进程 memory sandbox；不支持任意可执行路径 | T08 以 fixed identities 做协议／stale publish／cancel／poison 回归；T10 真实 packaged worker request 与所有 owned 进程／管道／handle join；不使用旧 requests 补当前证据 |
| bounded retention／响应性／低资源目标 | 已有内容／parse／history／cache／tab／recovery／worker 输出 budgets，cleared backing tails 和释放链 | [text-history][text-history]、[preview_limit][markdown-preview-limit]、[catalog][catalog]；[list lifetime review][list-review]；[VALIDATION：2588–2596][remaining-validation] | **P1：app/Markdown abandoned List Handle、Save fingerprint 上界、history truncation 披露**。B：单项 budgets 不是总 RSS 上界，活动 irregular history/全编码临时串仍有不同成本。V：真实长负载和同口径对照未完成 | T02/T04/T06/T07 修源问题；T10 按明确负载测实际 workset/RSS/latency，不能用 exe size、累计 alloc 或 cap 代称 whole-app 改善 |
| 文档／包／current-source acceptance | 大量旧闭合源／包证据；最新 accepted runnable 仍 .29；自主迭代授权独立于旧 held stage | [README][readme]、[ARCHITECTURE][architecture]、[VALIDATION][validation]、[delivery-gates][delivery-gates]；[旧 Linux report02][linux-report] | V：新树并无继承 pass；.30 preparation 不是 .30 acceptance。P2：协作手册较深“Accepted runnable”旧 current .27 小节与顶部 latest .29 并存，读者容易误选；旧历史不能改成新通过 | T09 单独更新当前索引／主线说明，保留旧结果；T10 exact source/package/version/manifest、fresh evidence、平台全链，不以文档检查代产品验收 |

## 4. 七个源码事实：修复方向与可验证失败形状

### G1. draw.io Delete：基本编辑链上的真实 no-op（P1）

[graphview/input.go：46–73][drawio-delete] 将 native `delete` 与 Delete/Backspace 路由到 `deleteKey`；[244–249][drawio-delete] 仅处理 mindmap 然后无条件返回 true。[draw toolbar：101–148][drawio-toolbar] 没有删除；[format edit API：28–158][drawio-edit] 只有更新／添加。因此既不是“按钮没做”这种高级愿望，也不是“图形根本只读”这种边界。

required：用户可以在已有可写简单图中纠正已添加的错误形状／连线；被处理的命令必须做事或明确拒绝。安全最小实现应只覆盖已理解的 cell 结构，并拒绝不能安全证明的引用／XML 形状；不能为了实现 Delete 去重写整篇 XML。

验收形状：选择节点→Delete→已知关联边随同一事务删除且可见说明→Undo 恢复 exact original；删 edge；别页不变；0/1／compressed readonly／unknown dependent parent refs 拒绝且 source/version/history 不变；local field draft Flush 失败不能删；回调造成 host/view 替换后不能写旧对象。

### G2. Markdown 的“一步撤销”必须成为用户能力（P1）

本次读取 [View.Draw/Export/Release][markdown-view] 并检索整个 markdownview 运行时：没有主 Buffer Undo/Redo、KeyZ/Y、native undo/redo 路由。现有 [任务][markdown-task-tests]、[列][markdown-column-tests]、[行][markdown-row-tests]、[对齐][markdown-alignment-tests] 测试大多由测试代码直接执行 Buffer.Undo。local draft 使用 Editor 自己的 history，并不补足 draft 清理之后的主文档 history。

required：在原生 reading/edit 子集完成任务／cell／结构变更后，用户能不切 source 地撤销。主文档与局部输入历史不能混成一条抢键盘路径。source/split 仍是退路，不是这个最基本编辑环节的替代品。

验收形状：focus preview 的 native command / Ctrl+Z / Ctrl+Y / Ctrl+Shift+Z 与 visible actions；任务单步；cell 和 structural edit 两步关系；local draft 内先走 draft history；Flush refusal／IME refusal／readonly/stale owner 不改 host；undo 后 rows/cache/selection 与 Changed 同步。

### G3. MyGo List Handle：已有 app 任务之外还有 Markdown 实例（P1）

既有 [115 行 source-trace][list-review] 已指出 MyGo public ListState.Handle 注册后会留在 engine.refs 直到窗口关闭，无 public list disposal；`CancelFocus` 只取消 focus request。layout 会在 row 构建之后消费 state；app 先 build sidebar，再 main，因此简单当场清零旧 state 会破坏当前 frame。

app Tree/Search/Outline 目前属于 experience 正在修改的独占 scope；不能重复派“还没开始的 list cap”。独立遗漏在 [Markdown Release：179–182][markdown-release] 和 [dropPreviewCache：48][markdown-preview-limit]：为了保旧 frame 指针而新建 state，仍放弃每个已 bound Handle。

required：保持 public Handle identity、取消过期 focus、释放大 geometry/cache，且不改坏仍在 build/layout 的 immutable frame。只保持 state pointer 不释放几何也不是正确收口；只保存 Handle 却在错误时机重置也不正确。本次没有测量泄漏字节／RSS／长期窗口句柄数。

### G4. Copy All 与实际偏好动作：用户能带走失败／看懂开关（P1/P2）

[issues UI：525–560][issues-ui] 完整显示 current ViewWarnings 与 SearchIssues，本次排除了“详情按钮空转”假设；但只有 Text 和收起按钮，没有 copy/export。原始 [ADR0046：99–109][issues-requirement] 明确 pure-local Copy All 自足且失败可见。native bounded snapshot 可以先实现当前数据模型拥有的信息，不能编造旧模型的 occurrence/resolvedAt。

[settings：383–399][settings-ui] 在 LineNumbers 已 true 时写“显示行号”、false 时写“隐藏行号”，点击实际是反转；wrap 同样反向。应显示正确当前状态或下一动作，测试必须断言点击后的值与可见文本，不能只断言某标签存在。

required：copy 成功仅在确认之后显示；失败以本地问题出口可见；snapshot 包含 app/时间/当前计数/未保留计数与 warnings/search 跳过，不抹掉拒绝。不得为了修 UI 同时写 experience 正独占的文件。

### G5. 普通 history 已有 cap；缺的是截断披露（P1）

[buffer.go：95–99、138][text-history] 提供 HistoryStatus.Truncated、256 entries、32MiB 默认预算；[552–588][history-operations] 总计 undo+redo retained bytes，trim 清 discarded slots 并设置 truncated。app status、editor 与 module 运行时未消费该 flag。一个大于 32MiB 但仍符合 128MiB 内容准入的单次真实编辑，可以让新 transaction 本身不留在 Undo 中。

required：用户必须看见“部分／本次撤销未保留”的真实限制；不能假称无限 Undo，也不能重新发明已经完成的 cap。优先每 tab 状态／问题 snapshot 披露，保持 Truncated 到 ResetHistory 的真实 lifecycle；不得把独立 local draft 的 history 与主文档混算。

[README：355、490–492][readme] 的“活动 group 无通用 cap”不等于普通 committed history 无 cap。活动 irregular group 可继续作为明确有风险的独立预算工作，但本次没有运行 adversarial IME 或证明其真实内存量，因此不列为“已证实泄漏 P0”。

### G6. Save fingerprint 的外部文件工作量没有上界（P1）

[Save：204–220][save-contract] 同步调用 `fingerprint`，发布前 check 和 [didSave：289–301][save-contract] 也调用；[fingerprint：379–410][fingerprint] 对当前 regular file 先 `io.Copy` SHA-256 到 EOF，再做 size/mtime/identity 检查。外部把已打开小文件替换成巨大文件，或边读边增长时，可以在发现冲突之前做远超原入场预算的工作。异步 cancellation-aware DiskToken 不覆盖这条 Save 路径。

required：保留 digest 与 pre/postpublish correctness，但按 baseline／已知编码输出长度限工作；stat 已经能证明基线不一致时应先拒绝。读至少需 count/limit+1/增长核对，不是仅加 `before.Size() > 128MiB`（真实 UTF-16／GB18030 输出长度可能不同）。postpublish 验证仍不得把另一个外部写者的 digest 当保存成功，也不得承诺无法实现的原子 CAS。

验收形状：小 baseline→外部巨大／增长文件→有界 conflict，不写原文件；晚增长仍拒绝；相同 stat 而 bytes 不同仍由 exact hash 发现；正确 UTF-8/BOM 与其他编码输出保存；published-but-verification-failed 的 Doc dirty/baseline 语义不误前进。源码确定的是缺边界，不是已执行的 UI 卡顿或 RSS 数值。

### G7. 路径修复和新回归必须闭合真实执行差异（P1/V）

当前 reliability 已有实际本机 Open→Close/Reopen 和 Host.OpenReference seam 方向；读取时旧 Windows-only 分支仍可见于 [closedFilePath][reopen] 与 [resolveLocalReference][references]。这是原生本机路径错误，不应把所有旧 Linux FAIL 都归因于“不是 Windows”。

[旧失败报告02][linux-report] 原调用已经消耗且 FAILED，末尾空索引 panic 中断了剩余测试；不意味着剩余 packages pass。新开发必须采用新 lane／新原始记录，保留旧 Windows 正反断言，具体区分 path bug、minimum-window/layout、autosave、barrier、实际 Windows-only worker/package 和 helper 环境问题。

## 5. 明确保留的有限子集：不是本轮 P0/P1 实现清单

| 原始／legacy 能力 | 原生实际边界 | 为什么可保留；交付时不能少说什么 |
|---|---|---|
| 完整 CommonMark/GFM、slash/format/block 拖动、复杂嵌入互操作 | 有常用块／inline／任务／bounded table 字面编辑；doc-block 显式本地引用，无嵌套编辑器／extract-flatten-drill 全链 | 用源／分屏继续编辑完整字面内容；unknown/partial 提示真实，导出原 HTML/doc-block 为 escaped 可见文本，不能宣称 Typora/Milkdown parity |
| Mermaid、Markdown inline 图片、paste assets、完整 HTML/PDF/print | 图片引用打开与独立 PNG/JPEG reader 已存在；无 inline renderer/assets manager/Mermaid/PNG export/PDF/print | 不执行原 HTML，不联网取外链；已有原链接／源码保留且披露，不能把图片阅读器误写成 Markdown 图片管理全部完成 |
| 完整 IDE、跨行准确词法、语义 symbols/node-aware Find | 自研 text 编辑 + 单行高亮；Find 使用源 JSON/XML；schema2 literal outline | 当前 native 必须明确源文本搜索而非主题／备注命中；不新增执行代码、LSP 或无界高亮来满足“四类文档” |
| mindmap 全 style／图片／arrows／summaries／富备注 | 已有 topic/note/local-link、结构/布局/collapse/markers/tags、SVG／readonly scene；未知字段保留，部分只占位 | 全链高级关系线等是 legacy，不能靠旧“已做／零 UI”描述推断新 Go 实现；source 仍能查看／修改，但必须说明 canvas/SVG 未完整表示 |
| 完整 draw.io group/stencil/image/多选／compressed 图形写回 | 多页读、有限形状／文本／basic connector／坐标／SVG；compressed 页图形只读 | readonly + source 是可用边界；写入只解释过的精确 XML；本轮 Delete 必须在该子集内做实，不以“完整 draw.io 未做”解释吞键 |
| 全部 legacy 偏好：字体族、compact/focus/typewriter、locale、位置 opt-out、侧栏拖调 | 主题／字号／行号／wrap，位置与 width 的会话字段；native 设置入口有限 | 数据字段与用户入口分开列；原始设置不全等于 P0。已决定修标签必须实现；若选扩常用入口，用单一验证/持久化合同，不另写 UI 私有设置 |
| 完整无障碍／屏幕阅读器／跨系统 | 原生 label/focus/keyboard 基础；未完成所有真实平台／screen reader 核验 | 不伪造 fully accessible/cross-platform claim；按声明支持平台完成必要 input/contrast/layout gate，其他未支持范围必须公开 |
| 全局硬 RSS ceiling／完整安全 sandbox／签名执行信任 | 多项 parser/payload/history/cache budgets；固定 local engines+显式 activation、integrity checks+owned Job cleanup | 源码有界不等于进程 RSS ceiling；hash 不等于签名或隔离。必须保持 local-only 和 known owners，资源测量按负载记录，不夸大 |

此表不是取消原需求。它给 lead 明确的 native v1 收口边界：若决定扩其中一项，先定义可验证合同／资源边界和独立 scope，而不是让“全部后续”变成永远无法完成的隐式 backlog。

## 6. 前 10 项按依赖排序的工程任务

1/2 可以并行；3/4/5 各自先做 formats/view/helper disjoint work，app 的所有 UI 接线须在其当前 owner 释放后由明确 owner 做。序号表达集成拓扑，不要求空等前一个完全结束。验证和包工作只由获新 assignment 的 remote product-command owner 在既有 box 的新 lane 执行。

| 任务 | required、优先级、当前归属／依赖 | 第一个可执行动作与范围 | 收口验收（不是本次已运行） |
|---|---|---|---|
| **T01 本机绝对路径与 file URL／重开闭环** | G7；P1；**native-reliability 正在负责**，不重复派；无 T02 依赖 | 在 [reference.go][references]、[reopen_closed.go][reopen] 对真实 native absolute、Windows lexical、file URL 分开判定；沿已建真实 seam 补正反 tests | 本机真实 Open→Close→Reopen；Windows absolute/relative/UNC 保留语义；encoded escape/host/query/NUL 禁止；receipt失败保留、同路径复用、满100仍不消费拒绝记录；新远端原始结果 |
| **T02 app Tree/Search/Outline public Handle 生命周期与 late-frame reset** | G3/G7；P1；**native-experience 独占现有 app scope**；无 T01 依赖 | 依 [list lifetime implementation direction][list-task] 完成 [app.go][app]/[workspace.go][workspace]/[outline.go][outline]/[view.go][app-view]；保存 Handle、CancelFocus、reset timing，保留 generation/searchID/controller guards | 旧 row/callback 跨导航/close/reload/query/outline toggle安全；新 state 不新增旧 public Handle；释放几何/大 backing；selection/focus/native input/minimum window 回归，不能以 RSS假数字代证据 |
| **T03 draw.io 精确原子删除** | G1；P1；**lead 已排队，待 task-7 后新的显式 assignment**；不依 app shared edit | 在 [drawio/edit.go][drawio-edit] 新 safe DeleteCell API/tests，再 [graphview/input.go][drawio-delete]＋[view_ui.go][graph-ui] 接 Delete／按钮；不要同步动 app | 可写子集删 node/edge；已知 incident edge 同一事务且解释；保护0/1、readonly compressed、unknown parent refs；staged reparse拒绝不变；host draft失败不删、一次Undo exact restore |
| **T04 Markdown preview host history 与该模块 list lifetime** | G2/G3；P1；**lead queued**；module owner 必须与 T03 disjoint；不并写 app | 在 [markdownview/view.go][markdown-view] 与 [preview_limit.go][markdown-preview-limit] 增 history controls/input routing 和安全 list reset；新 tests | preview 用户 Undo/Redo真可达；draft内先local undo；Flush/IME/readonly/stale/reentrant不丢；task/cell/structure事务分开；Release/dropPreviewCache不 abandoned Handle或破坏本帧 |
| **T05 本地 issues Copy All＋偏好动作标签** | G4；P1功能/P2文案；**任务板 task-10，lead queued 给 native-experience，blocked by task-6**；与 T06同scope收口 | 在该 owner 新remote source work 先新 `native/app/issues_export.go` 和独立 tests 写 bounded snapshot/clipboard receipt，再在 [issues/settings UI][app-view] 接线；不新增网络/export telemetry | app/时间/运行计数/dropped、当前 warnings/search skip完整可读；超长受控并明确省略；clipboard失败可见，未确认不报成功；两种布尔状态label/点击值/persist正确，150px issues区/760×480控制可达 |
| **T06 各主文档 history budget 截断披露** | G5；P1；**合并任务板 task-10，native-experience 完成 task-6后继续**；与 T05同scope收口 | 从 [HistoryStatus][text-history] 接当前tab可见状态／issue snapshot，按Buffer identity与Truncated transition去重；新case用小history limits模拟，不分配巨文件 | entries与bytes cap仍有效；“较早或本次历史因预算未保留”不伪造具体edit；record-once、切tab、ResetHistory/reload、localdraft隔离；不清coreflag、不改256/32MiB、不每render重复issues |
| **T07 Save fingerprint 有界工作** | G6；P1；**任务板 task-11，lead queued 给 native-reliability，blocked by task-5，独占 core/document**；不与 T01 app 文件撞写 | 在新remote source work 对 [fingerprint][fingerprint]/[Save、didSave][save-contract] 加baseline/known-output expected-length+1 budget与early mismatch；构造reader/stat seam tests；保留digest和atomic writer合同 | 外部巨文件/读时增长有界拒绝，原磁盘/Doc baseline不误推进；同stat不同bytes仍detect；合法各encoding/BOM输出；pre/postpublish conflict与失败状态；不以stat-only削弱安全 |
| **T08 新树必要回归族与一致集成验证** | G7与所有S/V；P1交付证据；依 T01–T07相应组；**仅当前获授权remote owner串行** | 从[旧 report02][linux-report] 的 symptom families到现有 [autosave tests][autosave-tests]/[reopen tests][reopen-tests]/[app tests][app-tests] 建新选择；按新dev lane实际源身份执行focused，修正后一次完整affected suite | 每个required failure有修复/明确platform理由；不能删除Windows断言／靠空sliceguard掩盖；minimum760×480、flush/refusal/persist/read-only/queued adoption/worker stale/cleanup覆盖；actual cwd/env/source/stdout/stderr/exit/timeout完整；未运行其余项如实列出 |
| **T09 source-converged 文档、当前索引、包输入与双远端身份** | scope与不静默；P2文档＋P1发布输入；依 T01–T08实现收口 | 被授权docs/lead owner更新 [README][readme]/[ARCHITECTURE][architecture]/[VALIDATION][validation] 的当前章节及 [AGENTS current index][agents-native]；为新candidate制定精确version/manifest/NOTICE/source identity | 普通history/activity边界说准；功能缺口已闭合或有限边界公开；.29与旧FAIL全部不可变；.30 prep不当pass；两个远端读回同commit（审计未执行Git）；包命名不暗指未验证新版本 |
| **T10 功能收敛后的统一平台／包／真实使用验收** | 三层gate；P1 application-deliverable；依 T08/T09；**最终阶段，不阻断T01–T07** | 在既有授权流程提供真实Windows target后，新candidate执行PE/完整portable inventory、launch/open/edit/save/restore/normal-close、四worker requests和native interaction；新原始路径 | exactWindowsGUI/PE/C:config/fileURL/WM_CLOSE/Job；物理IME候选／预编辑／focus/clipboard；readonly/冲突/failedFlush；第二次启动恢复；渲染/最小窗口/真实负载资源；owned进程/join clean；未过关键项不能宣布application-deliverable |

T08 不要求为每次小改动反复全量 test；按明确 affected group 集成一次。T10 也不要求反复跑旧发布目录，而是接受新的精确 candidate。任何动作都不得使用消耗过的原 guard / caller / phase 命名空间。

## 7. 三层 stop/go 门：不要把“写完代码”叫“应用交付”

### Gate A — feature-ready（native v1 范围内的开发收敛）

**当前：NO-GO，但继续开发 GO。**

必须具备：

- 原生四格式的已声明简单工作流可完成 open/new/edit/undo/save/save-as/close/reopen/recover；G1/G2不能作为“有source退路”保留。
- 当前 P1 source缺口均已修复／经 lead 明确、有依据的范围裁定；T01/T02/T03/T04/T05/T06/T07与其 required regressions收口；已承诺 settings labels也完成。
- Flush／IME／readonly／conflict／stale owner／dispatch failure拒绝保留源、草稿、selection/history与恢复身份；不静默。
- 具体 parser/history/list/cache/worker/source工作边界和限制向用户说明；可读 byte/source退路不被插件或format错误遮蔽。
- 新源受到相应focused和最终affected验证，并由lead review；单个成功的旧测试、源码存在或合成图片不能补缺失结果。
- 第5节有限子集公开，不制造无限legacy parity指标；新增高级范围另立合同。

Windows final可以仍是待测而 Gate A独立收口；这只是 development-complete，不发布“已通过真实Windows”的声明。

### Gate B — platform-tested（声明支持平台、精确 candidate）

**当前：NOT VERIFIED／NO-GO。**

Windows amd64 fixed catalog是当前真实目标：必须用本 candidate 的GUI PE、真实本机Windows path/config/readonly/atomic save/fileURL、实际窗口 `WM_CLOSE` 和Job-owned child cleanup验证。需要物理键盘／鼠标／剪贴板／IME候选与focus/preedit失败拒绝等输入；Tester事件与hidden启动不能替代。PNG/JPEG实际render、minimum-window滚动、格式字段与失败按钮也须在真实app核对。

源单测可保留为低层证据；Linux／CGO0／cross-build／Wine均不能替代上述目标。原 `-race` 曾初始化 `0xc0000139` 失败，[验证待办][remaining-validation] 已明确，不能宣称race通过。若额外OS未声明支持，公开未支持／未验收，不以无限跨平台测试拖住native v1。

### Gate C — application-deliverable（可以真实交给使用者）

**当前：NO-GO。** Gate A + Gate B均满足后，还必须：

1. 新源commit、两个远端、构建环境、version、5个GUI/worker产物及完整portable license/manifest/source inventory身份一致且可复现；不是只看 exe存在或包目录创建。
2. 从新干净profile完成四格式真实使用、失败重试、只读副本、dirty恢复，第二次launch核对会话；normal close没有未joined owned jobs/pipes/handles。旧 clean hidden close不是dirty恢复证据。
3. 四fixed引擎运行请求分别产出真实格式/当前页/scene信息；cancel/timeout/invalid response/poison/切包失败可见且不发布部分产物，不以旧版本请求当新请求。
4. 有明确负载与时间范围的资源／响应观测，覆盖多文档普通编辑/切换/反复格式release/导出/关闭。exe大小、累计alloc、解析cap与workset/RSS分开报告；不要求制造未经测量的“更轻更快”结论。
5. 没有未关闭的required重大failure，限制和支持范围写入用户文档；source/history/bytefallback与Copy All真正可走。当前用户能够独立复现结果，而不是必须依赖开发者test seam。

`.29`仍是最新已accepted runnable；本报告没有接受 `.30`或任何新包。代码完成、command-pass、package-created、platform-tested、application-deliverable是不同状态。

## 8. 最终交接

- task-7输出是一份22项 source-backed矩阵、7个真实问题方向、10项拓扑任务和3层gates。三个首要独立gap已优先发送；普通history披露和同步Save字节预算随后回传，五项全部获lead required收口决定并映射task-8/9/10/11。
- 当前无新增P0证明；G5/G6是实现边界缺失，不伪造运行期卡顿/RSS/物理IME结果；app正在修改的list patch没有被本报告顺带接受。
- 不能重排已完成 live-tab/growth/composition、四格式新建、图形Undo/Redo、Markdown引用／图片reader或第四worker；不能把已显示完整warnings的“查看详情”误报为空操作。
- lead下一步可直接完成task-7审阅，然后给draw.io原子删除新assignment；auditor在该assignment前不转产品写权限。app接线与remote产品命令继续遵守各自当前独占owner。

## 源码与需求引用

[agents-native]: <F:/AIproject/LevNote/native/AGENTS.md>
[agents-root]: <F:/AIproject/LevNote/AGENTS.md>
[delivery-scope]: <F:/AIproject/LevNote/native/handoffs/autonomous-native-delivery-plan-20261112.md#L17-L26>
[delivery-gates]: <F:/AIproject/LevNote/native/handoffs/autonomous-native-delivery-plan-20261112.md#L44-L50>
[migration]: <F:/AIproject/LevNote/docs/decisions/0063-native-migration.md>
[format-spec]: <F:/AIproject/LevNote/docs/format-spec.md#L170-L324>
[issues-requirement]: <F:/AIproject/LevNote/docs/decisions/0046-issues-this-run.md#L99-L112>
[app]: <F:/AIproject/LevNote/native/app/app.go>
[app-view]: <F:/AIproject/LevNote/native/app/view.go>
[new-document-ui]: <F:/AIproject/LevNote/native/app/view.go#L282-L288>
[settings-ui]: <F:/AIproject/LevNote/native/app/view.go#L363-L403>
[issues-ui]: <F:/AIproject/LevNote/native/app/view.go#L525-L560>
[issues-model]: <F:/AIproject/LevNote/native/app/app.go#L229-L371>
[app-tests]: <F:/AIproject/LevNote/native/app/app_test.go>
[reopen]: <F:/AIproject/LevNote/native/app/reopen_closed.go#L14-L55>
[reopen-tests]: <F:/AIproject/LevNote/native/app/reopen_closed_test.go>
[references]: <F:/AIproject/LevNote/native/app/reference.go#L31-L128>
[reference-tests]: <F:/AIproject/LevNote/native/app/reference_test.go>
[workspace]: <F:/AIproject/LevNote/native/app/workspace.go>
[outline]: <F:/AIproject/LevNote/native/app/outline.go>
[state]: <F:/AIproject/LevNote/native/app/state.go>
[restore-preferences]: <F:/AIproject/LevNote/native/app/state.go#L32-L44>
[autosave]: <F:/AIproject/LevNote/native/app/autosave.go>
[autosave-tests]: <F:/AIproject/LevNote/native/app/autosave_test.go>
[app-plugins]: <F:/AIproject/LevNote/native/app/plugins.go>
[document]: <F:/AIproject/LevNote/native/core/document/document.go>
[save-contract]: <F:/AIproject/LevNote/native/core/document/document.go#L194-L310>
[fingerprint]: <F:/AIproject/LevNote/native/core/document/document.go#L379-L410>
[atomic]: <F:/AIproject/LevNote/native/core/document/atomic.go#L14-L116>
[atomic-windows]: <F:/AIproject/LevNote/native/core/document/atomic_windows.go#L14-L46>
[text-history]: <F:/AIproject/LevNote/native/core/text/buffer.go#L95-L150>
[history-operations]: <F:/AIproject/LevNote/native/core/text/buffer.go#L469-L588>
[session]: <F:/AIproject/LevNote/native/core/session/session.go#L19-L118>
[find]: <F:/AIproject/LevNote/native/core/search/find.go>
[workspace-search]: <F:/AIproject/LevNote/native/core/search/workspace.go>
[editor]: <F:/AIproject/LevNote/native/ui/editor/editor.go>
[editor-contract]: <F:/AIproject/LevNote/native/ui/editor/contract.go>
[editor-composition-tests]: <F:/AIproject/LevNote/native/ui/editor/composition_test.go>
[editor-growth-tests]: <F:/AIproject/LevNote/native/ui/editor/growth_test.go>
[markdown-view]: <F:/AIproject/LevNote/native/modules/markdownview/view.go>
[markdown-release]: <F:/AIproject/LevNote/native/modules/markdownview/view.go#L151-L187>
[markdown-preview-limit]: <F:/AIproject/LevNote/native/modules/markdownview/preview_limit.go#L16-L54>
[markdown-limit-tests]: <F:/AIproject/LevNote/native/modules/markdownview/preview_limit_test.go>
[markdown-task-tests]: <F:/AIproject/LevNote/native/modules/markdownview/tasks_test.go>
[markdown-column-tests]: <F:/AIproject/LevNote/native/modules/markdownview/table_columns_test.go>
[markdown-row-tests]: <F:/AIproject/LevNote/native/modules/markdownview/table_rows_test.go>
[markdown-alignment-tests]: <F:/AIproject/LevNote/native/modules/markdownview/table_alignment_test.go>
[graph-adapter]: <F:/AIproject/LevNote/native/modules/graphview/adapter.go>
[graph-ui]: <F:/AIproject/LevNote/native/modules/graphview/view_ui.go>
[drawio-toolbar]: <F:/AIproject/LevNote/native/modules/graphview/view_ui.go#L101-L148>
[graph-history]: <F:/AIproject/LevNote/native/modules/graphview/input.go#L210-L242>
[drawio-delete]: <F:/AIproject/LevNote/native/modules/graphview/input.go#L46-L249>
[drawio-edit]: <F:/AIproject/LevNote/native/formats/drawio/edit.go#L14-L208>
[drawio-tests]: <F:/AIproject/LevNote/native/formats/drawio/drawio_test.go>
[drawio-retention-tests]: <F:/AIproject/LevNote/native/formats/drawio/source_retention_test.go>
[mindmap-serialize]: <F:/AIproject/LevNote/native/formats/mindmap/serialize.go>
[image-view]: <F:/AIproject/LevNote/native/modules/imageview/view.go>
[image-input]: <F:/AIproject/LevNote/native/modules/imageview/input.go>
[image-tests]: <F:/AIproject/LevNote/native/modules/imageview/imageview_test.go>
[plugin-manager]: <F:/AIproject/LevNote/native/core/plugins/registry.go#L312-L455>
[plugin-limits]: <F:/AIproject/LevNote/native/core/plugins/declarative.go#L59-L107>
[catalog]: <F:/AIproject/LevNote/native/core/enginecatalog/catalog.go#L9-L63>
[protocol]: <F:/AIproject/LevNote/native/core/engineipc/protocol.go>
[runner]: <F:/AIproject/LevNote/native/core/engineruntime/runner.go#L79-L181>
[managed]: <F:/AIproject/LevNote/native/core/engineruntime/managed.go#L9-L36>
[process-windows]: <F:/AIproject/LevNote/native/core/engineruntime/process_windows.go>
[list-review]: <F:/AIproject/LevNote/native/handoffs/native-list-handle-lifetime-next-review.md>
[list-task]: <F:/AIproject/LevNote/native/handoffs/native-list-handle-lifetime-implementation-task.md>
[readme]: <F:/AIproject/LevNote/native/README.md>
[capabilities]: <F:/AIproject/LevNote/native/README.md#L348-L367>
[format-boundaries]: <F:/AIproject/LevNote/native/README.md#L369-L394>
[architecture]: <F:/AIproject/LevNote/native/ARCHITECTURE.md>
[validation]: <F:/AIproject/LevNote/native/VALIDATION.md>
[remaining-validation]: <F:/AIproject/LevNote/native/VALIDATION.md#L2588-L2596>
[linux-report]: <F:/AIproject/LevNote/native/handoffs/remote-box-product-validation-02-20261112.md>
