# LevNote 原生主线当前状态

更新：2026-11-21。**用户已授权持续完成原生交付；若干源码批次已获有界接受，整产品及新正式发行包仍未接受。** 开发主线为 Go 1.27.3 + MyGo 原生 Surface；旧 Tauri/React 路径保留为历史实现。[仓库交接](../HANDOFF.md)记录本轮接续之前的暂停点，当前任务与文件归属以[接续记录](handoffs/full-delivery-continuation-20261119.md)、[本轮最新分工](../.localtools/full-delivery-20261119/task-lead-independent-freeze-review/PARALLEL-IMPLEMENTATION-LANES-20261120.md)及[2026-11-21所有权账本](../.localtools/full-delivery-20261121/lead-resumption/OWNERSHIP.md)为准。

## 当前开发与交付状态

- Markdown 表格草稿已实现只读捕获已确认文本和配套 UTF-16 选区，7项定向测试通过；该次104个列明输入前后一致。嵌套列表解析、HTML与原生预览已实现；Task28 RED→GREEN、Task29/37 联合回归有界通过，但 formats+markdownview 同次全包运行及 Markdown 模块批次接受仍缺。Task113 修复表格单元格 Changed-successor 回执、Task127 静态候选补普通块回执均未集成或运行；Task127 当前 live 比其 Task113 baseline 的五个既有源均不同，测试文件也尚不存在，须重基线并保留前置所有权修复后再验证。当前 Markdown 不作整批接受。
- 图形标记的过期操作反馈、克隆前字段写回后的拒绝反馈已有定向回归，10个交付文件经 lead 哈希复核一致。大文档克隆的撤销预算与视觉选区恢复已由 [Task20 修复并经独审接受](../.localtools/full-delivery-20261119/task-graph-history-lead-review/20261119T1629Z/REPORT.md)，[Task130 在 Task94 Core 后重新执行 graphview 包测试并由Root有界接受](../.localtools/full-delivery-20261121/lead-resumption/TASK130-ROOT-ACCEPTANCE.json)：199 top + 384 nested PASS、0 FAIL/SKIP、17,019输入稳定；不含 vet、App 或 Windows 接受。
- 关闭事务和恢复文件发布正在修复发布后错误分类、Windows文件身份、内容变更检测及重复关闭错误累积，尚未接受。
- 多选区文本模型已交付并进入独审；多光标输入法及用户交互尚未完成。Go/严格JSON折叠和本地语言服务正在接入与修正，不代表其他语言或跨项目语义已支持。
- 当前证据来自 Linux 开发与定向检查，完整受影响回归仍待依赖收敛；新包的 Windows 键鼠/IME、四类随包引擎、正常退出、重启恢复和资源测量均未完成。原始失败和历史跳过保留，不以交叉编译替代实机接受。

2026-11-20 至 21 的阶段记录（后续结论见下段）：[七路径文本 Core](../.localtools/full-delivery-20261119/task-lead-independent-freeze-review/TASK94-ACCEPTANCE-20261120.md)和[十路径 A1.1 原生取消传输](../.localtools/full-delivery-20261119/task-lead-independent-freeze-review/TASK103-ACCEPTANCE-20261120.md)已分别经独审及 Root 有界接受，最终所选回归为 154 top + 344 sub 与 67 top + 265 sub PASS。[B2 源码结果过期显式披露](../.localtools/full-delivery-20261121/lead-resumption/TASK109-B2-ACCEPTANCE.md)现经 Root 有界接受：保留真实行为 RED→GREEN，最终 56 top + 137 sub PASS；范围不含整 App、自动语言服务、Editor、Windows 或发行。[Task119 六项真实 RED→GREEN 已获 Root 有界接受](../.localtools/full-delivery-20261121/lead-resumption/TASK119-GREEN-ROOT-ACCEPTANCE.json)：相同测试字节/selector，RED的5/6 top与16/21 nested失败转为6top+21nested全PASS，0FAIL/SKIP；精确两源promotion、一次gofmt及2283 before/after/current和11tools闭合。历史清单含1个GOCACHE工件，已单列缓存身份，非cache输入为2282项。Task136缓存排除v2及Task119新的union-cache-excluded冻结已由Root分别接受；独立冻结28项核对后，唯一94声明Editor union真实运行exit0：94 top +157 nested PASS，0 FAIL/SKIP/build失败，JSON与printed的251次start/terminal精确闭合，2282稳定输入/11工具/config前后相同，mutable cache旁路未入稳定guard。详见[TASK119冻结Root复核](../.localtools/full-delivery-20261121/lead-resumption/TASK119-CACHE-EXCLUDED-FREEZE-ROOT-REVIEW.json)及[94项联测Root接受](../.localtools/full-delivery-20261121/lead-resumption/TASK119-CACHE-EXCLUDED-UNION-ROOT-ACCEPTANCE.json)。该结果仅限所选Editor Linux测试，不代表完整Editor/App/Windows接受。Task143补全候选的Find ABA与fixture问题经Task146焦点epoch修订后，Task153基线1top+7nested为行为RED（4有效Find焦点请求失败、3控制PASS）；最终85精确union exit1，84topPASS/1topFAIL+174nested PASS，唯一失败是补全取消控制。Task162在新17,035项工作区对该测试仅诊断一次：真实Cancel Click返回nil且目标存在，但取消后Visible/Owner/Intent及3候选保留、无排队action、无View状态变化，exit1；capture guard闭合但未接受，原因未区分。Task166在独立17,035项工作区追加View pass/守卫/无障碍观测，经Task171独审及Root有界接受后，已单独format/freeze/运行一次：exit1，capture闭合0guard；Click确有3次View+1额外Frame，11组状态均current/未禁用/无组合或输入错误，但无排队action。Cancel按钮由Y265移到Y237（-28），原centerY278在最终按钮外；中途pressed/Clicked未观测，不据此直接断定释放丢失。Task175静态追踪到虚拟List缓存估高→真实测量导致auto-height收缩，正在新私有候选把固定动作行置于List前并保持原动作入队优先级；仍需新对照与85union接受。Task169失败审阅的partial记录、Task162及Task166全部原始证据保持不变。Task154 Markdown Release新双phase均17,037项，baseline真实RED：一个stale Release点击产生两条同sentinel报告；candidate 292 top +705 nested完整启动且JSON/printed闭合，但exit1、候选不接受：formats 86+237 PASS；markdownview 4 top/10 nested只在新增Task142 Enter路径焦点fixture失败，3个需要显式截图目录的视觉测试SKIP。Task170在新私有完整pair仅修Enter测试真实键盘聚焦路径；旧Task154 raw/freeze/run不可重跑。Task158发现no-frame changed-client disabled-scope覆盖空缺；Task165追加3种转换同phase测试，经Task167静态独审与Root有界接受，仅限未来准备，未format/执行（3 RED、10 final提案）。Task163/164的B5取消fixture静态修订只清除saved后的live确认action，Root接受静态；因补全union仍未接受，不运行重命名。以上均非Windows物理IME、整App或产品发行接受。旧freeze/raw/exit和所有Root脚本错误记录保留。Task140 Markdown动作候选经Task144独审无新增静态blocker，但继承table-cell Enter/PointerUp和reference pointer-up→queued-click各有重复Report P2。Task141私有修订经Task149追踪仍有Release入口清receipt、Changed安装successor致早退时旧Clicked重报P2；Task150两套17,036项准备保持未格式/未执行HOLD。Task151窄修Release receipt lifetime并补真实Tester回归，经Task157独审无blocker；Task154全新完整baseline/candidate各17,037输入已Root完整重哈希，唯一阶段差异view.go、同一新回归及Task142修正测试均一致，待固定formatter/新freeze/真实RED及一次86+206声明全包联测。Task162从Task153失败candidate另建17,035项诊断树，仅controls测试变化，Task159 overlay经Task160独审无blocker；仍等待新formatter/freeze及一次诊断执行。Task156 B5重基线117选择器Root静态pin已闭合，但F156-1取消fixture时序阻断、Task153基础 union 也未接受，Task163仅修测试前置。Task155 no-frame独审正在闭合并标注changed-client缺少独立子例。上述新候选均未集成或运行。首次green-prep于源码写入前因Root证据格式/引用错误停止，完整保留后由新v2完成，旧失败未覆盖。Task121确认 B5 List 双击误确认与名称 Editor 焦点被抢；[Task128三源私有修订经Task131独审、Root接受为无P1静态候选](../.localtools/full-delivery-20261121/lead-resumption/TASK128-ROOT-STATIC-REVIEW.json)，仍须B2/B4/Editor重基线和运行验证。[Task129 no-frame四路径候选经Task132独审、Root接受为无P1静态候选](../.localtools/full-delivery-20261121/lead-resumption/TASK129-ROOT-STATIC-REVIEW.json)；Root另独立发现Task129三份consumer stage基于旧Task106，live文件不同，且manifest由6项增至7项；不得整份回拷旧stage，须以当前消费者文件rebase headless成员及最终manifest pin，依据[TASK129当前基线复核](../.localtools/full-delivery-20261121/lead-resumption/TASK129-CURRENT-INTEGRATION-BASELINE.json)。格式化后patch/manifest/三个consumer pin闭合、运行验证及Windows语义仍待完成。Task134已按当前B2重采样完成B4六源私有候选与真实combined diff，Root机械身份/diff闭合通过；Task139独审经Root接受HOLD两项P2：异步补全可在用户转到idle Find/GoToLine后抢回焦点；服务最多512候选但UI仅256可选，JSON尾部无筛选/翻页出口。另需执行双击/重复pointer提交覆盖。独审确认Task134两测试行数metadata561/468应为378/355，内容SHA未变。未promote/运行。Task133已重建Markdown六源私有候选（Task100+113+127链，live基线精确匹配），Root机械diff闭合；Task135发现709行新测试含baseline不存在符号，不能做RED；其中rejected-replace预期也与Task119 sealed Editor settlement语义不符（静态预期ErrSizeLimit/单次Report）。Task138在新私有域拆分7+3测试、19子案例并修正oracle；Root88项身份/差异复核通过，但发现第二测试文件有未使用ui import，编译前静态阻断，候选HOLD。旧Task138工件不修改；Task142在新私有域仅删一行unused import，经Root独立36/36身份/字节检查后接受为静态测试源修正，receipt SHA12c519e9…70f2d3；未typecheck/运行。Task140另修task toggle在cell Changed-successor后续写、link/reference旧动作继续OpenReference及实际UI单次Report；Task120 table-cell UI双Host.Report另Task141。均未promote/执行。Task136已完成私有cache-excluded v2 recorder/policy候选，103项Python AST/隔离内存/fake-OS静态自检通过；Root静态接受、重建新freeze与94声明执行授权仍待，未调用recorder入口或Go。Task143拟修B4 P2焦点所有权/512项可达性，须独立新候选和Tester回归。Task119六项GREEN已接受，但旧94项union freeze曾只读生成一次后因稳定清单误含mutable GOCACHE工件而HOLD，未执行测试；新缓存排除录制器与新freeze仍待复核。以上局部结果不代表整App、Windows实机、资源或发行接受；[当前分工](../.localtools/full-delivery-20261119/task-lead-independent-freeze-review/PARALLEL-IMPLEMENTATION-LANES-20261120.md)和[2026-11-21所有权账本](../.localtools/full-delivery-20261121/lead-resumption/OWNERSHIP.md)记录当前路径权限与未完成门禁。

2026-11-21 后续进展：[Task170 Markdown](../.localtools/full-delivery-20261121/lead-resumption/TASK170-ROOT-BOUNDED-INTEGRATION-ACCEPTANCE.json)、[Task175 补全取消布局](../.localtools/full-delivery-20261121/lead-resumption/TASK175-ROOT-BOUNDED-INTEGRATION-ACCEPTANCE.json)及[Task172 原生取消传输](../.localtools/full-delivery-20261121/lead-resumption/TASK172-ROOT-BOUNDED-INTEGRATION-ACCEPTANCE.json)已分别获得限定范围接受。[Task198 重命名联测](../.localtools/full-delivery-20261121/lead-resumption/TASK198-ROOT-BOUNDED-INTEGRATION-ACCEPTANCE.json)已真实执行117个顶层、232个嵌套测试，全PASS、零SKIP/guard，17,039输入与文件身份稳定；Task191原始因果RED和两个fixture/oracle失败保留。只修正两个测试的前置条件/精确错误理由；双击预览不会确认，显式Enter/按钮确认均通过。格式化实际exit0，因本地计数缓存造成的原false receipt另有独立闭合补充，未重复运行格式化。已将四个接受来源合成为[Task199隔离工作区](../.localtools/full-delivery-20261121/task199-final-compose/PREPARATION.json)，17,045文件、相对live有35条晋升差异；尚未晋升live。[Task202完整六根](<../.localtools/full-delivery-20261121/lead-resumption/TASK202-ROOT-SIX-ROOT-FAILED-CLASSIFICATION.json>)已实际失败：27包、1825顶层PASS/15FAIL/4SKIP，3248嵌套PASS/17FAIL；输入稳定、进程闭合且无guard或event错误，未继续vet/build。失败分为隔离工作区漏带插件示例、旧输入所有权测试前置条件、窄窗口真实裁切及超大文档导出拒绝顺序，分别补齐或修复；四项跳过仍为随包Windows门禁，不计PASS。B6主源码快捷键的[Task212联测已失败并闭合](<../.localtools/full-delivery-20261121/lead-resumption/TASK212-ROOT-FAILED-CLASSIFICATION.json>)：17,054输入稳定，计划339top实际280PASS/2FAIL、626nestedPASS/1FAIL；Editor新fixture重复key导致57top未到达，App窗口失焦用例失败，plugins44全部通过。[独立Host审阅](<../.localtools/full-delivery-20261121/task214-host-review/REPORT.md>)还发现同帧SaveAs后旧按键被新绑定接收，已由[Task219四文件修复](<../.localtools/full-delivery-20261121/task219-b6-recovery/REPORT.md>)接续失败的215交付，并经[Task221独立静态审查](<../.localtools/full-delivery-20261121/task221-b6-final-review/REPORT.md>)确认无新增blocker，仍待执行。原Task210布局代理服务失败后，[Task216](<../.localtools/full-delivery-20261121/task216-layout-recovery/REPORT.md>)交付六runtime和三项新回归；[Task220独审](<../.localtools/full-delivery-20261121/task220-layout-run-review/REPORT.md>)发现警告Details弹窗可能结束健康组合输入，222保存纯输入门禁后服务失败，225补齐真实App/graph组合输入与原生会话回归，226修正自绘Editor不属于文本label的测试观测；[Root联合静态复核](<../.localtools/full-delivery-20261121/lead-resumption/TASK225-226-ROOT-STATIC-REPAIR-REVIEW.json>)通过。Task209输入测试与211导出顺序修复均进入[Task217冻结组合](<../.localtools/full-delivery-20261121/task217-repaired-integration/FREEZE.json>)：17,056输入、17覆盖文件、1,880预期top，格式化exit0仅两文件空白变化，完整六根测试已启动；记录器已补强独立bytecode路径与精确平台skip/no-test身份检查，未启动新Go运行。Task218服务失败后B7由223在独立路径接续；B6/B7均未获行为接受，其余产品能力、Windows与新发行继续待完成。

2026-11-21 最新整合检查：[Task217完整六根失败分类](<../.localtools/full-delivery-20261121/lead-resumption/TASK217-ROOT-SIX-ROOT-FAILED-CLASSIFICATION.json>)已核对全部1880个顶层测试：1865 PASS、11 FAIL、4项Windows随包SKIP，嵌套3351 PASS/14 FAIL；17,056输入稳定、进程闭合、guard/event零错误。Task202原15个失败中13个已通过，当前失败集中于图形警告消息/控件身份、两处App测试前置条件与Markdown窄窗裁切，正在Task230/231私有候选修复；Task217不继续vet/build。Task227正在补B7独立500ms语言服务与900ms恢复时序，已扩展到真实面板/补全/重命名入口，以保护定时器回调重入后的会话继任者。Task228原生取消三源经[Root静态审查](<../.localtools/full-delivery-20261121/lead-resumption/TASK228-ROOT-STATIC-NATIVE-CANCEL-REVIEW.json>)后，已在[Task232完整Editor包运行](<../.localtools/full-delivery-20261121/task232-native-cancel-validation/test/RESULT.json>)中通过166 top +349 nested，零FAIL/SKIP、17,057输入稳定且进程闭合；随后[Task233独审](<../.localtools/full-delivery-20261121/task233-native-cancel-review/REPORT.md>)发现首单点无token拒绝挡住原生短重试、合法全选后receipt未同步挡住正常终结两项未覆盖的兼容性阻断，Root保留既有PASS并暂停取消行为接受；Task234正在新三路径候选修复和补真实原生对照。Task230/231已在[Task235完整六根](<../.localtools/full-delivery-20261121/lead-resumption/TASK235-ROOT-SIX-ROOT-FAILED-CLASSIFICATION.json>)中复验：1874 top PASS/2 FAIL/4 Windows SKIP，3363 nested PASS/2 FAIL；5245个JSON/打印身份精确闭合，17,056输入与元数据稳定、零guard/event错误。旧11失败有9通过，Markdown与导图标签的四张760×480浅深截图已核看。剩余两个健康备注新fixture错误假定原生final不会立即写回Host，实际旧合同会准确提交；Task237仅修两测试以区分合法final与Details额外副作用，保留所有拒绝与所有权断言；不运行vet/build。B7已扩展至11个必要生命周期和富文本Host提交路径、23组新测试，尚待封存执行。Task236已完整审阅Task229原件并封存四文件未完成工具修订，明确inactive；早期错误收据、capture精确清单和阶段证据绑定仍待接续，真正Windows解析/编译/执行未完成。

最新正式接受包仍是 [Windows x64 `.29` ZIP](downloads/LevNote-native-worker-20261110.29-windows-x64.zip)；[Flowix `.6` 工作台体验 ZIP](downloads/LevNote-Flowix-preview-20261116.6-windows-x64.zip)是历史布局体验包。两者均不包含本轮新增源码。CI尚未启用，当前没有新正式发行包。

## 历史已接受批次

- 当时源码基点：`f2c2ceb82acecb40fb0502c047e41ba12a12be47`。成熟功能首批已完成命令面板、行移动/复制、Markdown行内格式、导图颜色及只读显示一致性；最终3375 PASS/49原SKIP/0 FAIL，vet与五GUI构建exit0。前版全部3264测试身份保留并新增160，426测试输入前后同。详见[本批验收](<F:/AIproject/LevNote/native/handoffs/feature-parity-root-acceptance-20261115.md>)。
- 现代工作台UI已完成：深浅分层背景、圆角标签、真实文档搜索、分段模式切换、舒展欢迎页与灰阶状态。最终完整回归3215 PASS/49原SKIP/0 FAIL，vet及五GUI构建退出0，前版全部测试身份保留。详见[现代UI验收](handoffs/modern-ui-root-acceptance-20261115.md)。此前[VS Code版验收](handoffs/vscode-ui-root-acceptance-20261115.md)与[3183项验收](handoffs/continuation-root-acceptance-20261115.md)保留。
- flowix 风格工作台已升级为最新体验版：浅色冷白纸面配紫色品牌锚点、深色冷暗画布配金黄品牌锚点、8px 圆角，顶栏重构为 48px 紧凑动作区，左下四类新建入口改为左对齐列表。六根门禁3577 PASS/49原SKIP/0 FAIL、vet exit0、身份审计零退化。最新[Windows x64 工作台布局体验ZIP](<F:/AIproject/LevNote/native/downloads/LevNote-Flowix-preview-20261116.6-windows-x64.zip>)版本native-flowix.20261116.6-preview，11,746,501字节；五GUI PE、26文件、八引擎路径及全部清单哈希通过独立核验。详见[布局验收](handoffs/flowix-chrome-root-acceptance-20261116.md)；上一[主题体验ZIP](<F:/AIproject/LevNote/native/downloads/LevNote-Flowix-preview-20261116.5-windows-x64.zip>)及[主题验收](handoffs/flowix-restyle-root-acceptance-20261116.md)保留。
- 最新正式接受包仍为 `native-worker.20261110.29`，见[包接受记录](handoffs/composition-release29-root-acceptance.md)。最新[Windows x64 draw.io多选体验ZIP](<F:/AIproject/LevNote/native/downloads/LevNote-Drawio-preview-20261115.4-windows-x64.zip>)版本native-drawio.20261115.4-preview，11,743,909字节；五GUI PE、26文件、八引擎路径及全部清单哈希通过独立核验。此前[功能增强体验ZIP](<F:/AIproject/LevNote/native/downloads/LevNote-Features-preview-20261115.3-windows-x64.zip>)保留。此前[现代UI体验ZIP](<F:/AIproject/LevNote/native/downloads/LevNote-Modern-UI-preview-20261115.2-windows-x64.zip>)保留；完整桌面发行验收仍待完成。
- GitHub CI 保持停用，待开发与平台交付验收完成后再启用。

## 已有能力

四类文档共享多标签、源码编辑、保存/另存、查找替换、工作区搜索、问题记录和有预算的会话恢复。原生格式视图提供 Markdown 块/表格编辑、导图节点/标记/标签编辑，以及 draw.io 的有限形状/连线编辑，含 Ctrl 多选、六向对齐、Shift 框选与整组移动。四个固定本地引擎承担 HTML、SVG 导出和只读导图场景；PNG/JPEG 可只读查看。完整能力与边界见[功能表](README.md#已有能力与边界)。

最近完成的源码修复包括图形草稿恢复的输入所有权、恢复失败的单次报告，以及工作区搜索结果与实际搜索文本快照的绑定。旧搜索结果遇到保存后变动或未打开文件的内容变化时，会提示重新搜索，保留当前选择和文档状态。

## 本轮修复与验证

| 修复 | 用户可见变化 | 当前证据 |
|---|---|---|
| Markdown 闭合代码块写回 | 替换正文时保留关闭围栏的独立行和原 LF/CRLF，后续段落保持正常；空正文与未闭合块保留其语义 | 真实失败后同断言通过；含真实Enter的19项公开交互与失败保留回归通过，见[围栏记录](handoffs/continuation-format-audit-20261115.md) |
| 重载后的会话恢复 | 已明确放弃的草稿在成功重载后更新恢复快照；失败、取消及确认期间的文档变化另作保留检查 | 两个入口已各完成真实失败与修复验证，12项成功/失败/确认变化回归通过；见[重载记录](handoffs/continuation-app-audit-20261115.md) |
| 四个固定引擎的包检查 | 资产 fixture 补齐 draw.io 可执行文件和 manifest；八路径必须完整、唯一且没有额外成员 | AST 与实际路径判据 13 项通过；完整目录 fixture 待平台环境修复，见[发布审计](handoffs/continuation-release-audit-20261115.md) |

上表描述 2026-11-15 已完成的历史批次：该批完整源码门禁与双轴审阅通过，CRLF 真实回车问题已复审关闭。随后 2026-11-19 的交接快照包含未验收的 draw.io 增量，当时最近两次定向检查均 exit1，Markdown 嵌套列表与表格草稿恢复也只有设计；这些原始事实保留在[停点与验证范围](../HANDOFF.md#暂停时的三个任务)。重新授权后的实现和验证进度见本页顶部，不能用历史通过或失败代替对当前源码的检查。

## 交付前仍需完成

| 工作 | 当前事实 | 完成判据 |
|---|---|---|
| 新 Windows 包 | UI体验包已完成五个GUI PE、26文件、四引擎八路径及ZIP独立核验；历史`.33`失败保留 | 体验包可交付；正式发行还需下列运行验收 |
| 实际 worker 生命周期 | 新源码的真实随包 worker 验收仍待执行 | 四类请求及失败/取消路径通过，进程与 Job 清理完成 |
| 宿主与会话 | 新包的正常关闭、第二次启动与会话恢复仍待执行 | 关闭不丢草稿；恢复内容/次序/选区正确；所属进程退出 |
| Windows 桌面输入与布局 | 当前宿主未提供可用的 Windows 桌面自动操作入口 | 实际键鼠、中文 IME、剪贴板、原生选择器、窄窗和显示缩放验收 |
| 资源表现 | 源码预算与局部基准已有覆盖；当前整应用指标待测 | 固定工作负载下记录启动、编辑、切换与关闭时的 CPU/内存，并解释异常 |
| CI | 原生 CI 草案已准备，未激活 | 上述开发与交付检查收敛后启用并记录实际运行结果 |

历史`.33`的PowerShell目录创建失败及Low完整性标签诊断保留。新UI体验包使用已审阅的固定五目标本地Go/Python包装器，沿用工具/补丁pin与离线约束，单次构建成功；未修改工具链标签、系统权限或旧产物。七场景十一张MyGo测试器实渲图覆盖深浅主题和760×480窄窗；物理输入与显示缩放仍归完整桌面验收。

## 后续功能完善方向

这些是公开说明的原生能力缺口，需要分别设计、实现与验收，不计入当前功能已经完成：

1. Markdown 嵌套列表已实现并有 RED→GREEN 与定向联合回归，但 formats/markdownview 同次全包门禁和整批接受未完成；剩余为图片内嵌显示/粘贴资源管理、Mermaid 渲染和更多富文本操作。
2. 导图图片、关系线/概括完整交互，以及标签样式、排序和字段搜索。
3. draw.io的Ctrl多选、六向对齐、Shift框选与整组移动已完成，接口及保真合同见[实施计划](<F:/AIproject/LevNote/native/handoffs/feature-parity-drawio-next-20261115.md>)、验收见[多选验收](<F:/AIproject/LevNote/native/handoffs/drawio-multiselect-root-acceptance-20261115.md>)；之后补克隆/批删、分组、复杂形状、图片和stencil支持。
4. PNG/PDF 导出与打印；代码折叠、多光标与语言服务。
5. 原生国际化、无障碍和其他操作系统验收。

优先顺序是保存/草稿/撤销等数据可靠性、现有命令的完整行为、实际发行验收，再推进新增格式能力。避免把有限子集描述成完整旧版功能对等。

## 开发验证入口

当前 Linux 开发从仓库根目录使用[项目内入口](../.localtools/native-go.sh)，固定本地 Go1.27.3、离线vendor、缓存和临时目录。仓库保留的[原Windows门禁脚本](../.localtools/gate.sh)固定历史本机路径，不能直接当跨机器入口。先运行与改动相关的定向回归，待依赖收敛后执行下列完整检查；不要把交接快照纳入测试。Windows真实运行与发行接受另按本页门禁完成。

```bash
bash .localtools/native-go.sh test -json -count=1 -timeout=300s ./app/ ./cmd/... ./core/... ./formats/... ./modules/... ./ui/...
bash .localtools/native-go.sh vet
bash .localtools/native-go.sh build
```

历史验证记录见[验证文档](VALIDATION.md)。本页提供当前结论，历史记录保留原时间、范围和实际结果。
