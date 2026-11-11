# 原生验证记录与测量口径

本页记录 2026-11-08–10 原生迁移验收的证据，功能边界以 [README](README.md)
为准。所有时间为单台 Windows amd64 机器的样本，不能代表所有设备。

## 输入法组合更新合并与 `.29` 当前交付：2026-11-10

当前入口
[bin/levnote-native-worker-20261110.29](bin/levnote-native-worker-20261110.29/README.md)，
版本`native-worker.20261110.29`。根代理已接受最终源码、实际包、宿主结果和外部
闭包；`.28`及更早包与证据保持不变。

| 记录 | 字节 | SHA256 / inventory identity |
|---|---:|---|
| 26文件候选包 |27,408,380|`9FAEE40604A7DBA31ED14003011938911661E8BBBE0CCBDC5D9EBCDAA640962B`|
| 宿主`levnote-native.exe` |14,580,224|`9462194DDC7A310834D564A0A5DE2281A7C799BCE323989D1755CE698CD4CFC6`|
| `release-manifest.json`，25载荷项，不含自身 |4,674|`8CCFC58E3D1A19847EF7A55289DE947C7C35D493DCEB1681772250D07AB00722`|
| aggregate attempt1 |51,587|13/13PASS，SHA256`E5D72F92DE5486118D85A8FDC048E955615FB20FF2E0BB34EF6B9173EEEF6C47`|
| attempt validator |10,529|PASS，SHA256`529ADD2C88D89142D5068AE1B37B4E190AD466829A2A4DBC03AC1EB798C9DB7D`|

### 冻结源码与最终test/vet

[根源码接受](handoffs/active-composition-coalescing-source-root-acceptance.md)为
2,775字节，SHA256
`59F8AF4AEF765C9E0C725E6759DA2E96C6DC03171998774CA50B9EFD4CD3B7BF`。
[最终源码清单](handoffs/active-composition-coalescing-final-evidence-20261110/source-frozen.json)
为628字节，SHA256
`5673AB5368E21B7F81631418448C0D4B20B8ECC3A274906E6BE3EBF77EB47213`，
包含2个runtime和2个test文件：

| 源文件 | 字节 | SHA256 |
|---|---:|---|
| `core/text/buffer.go` |23,353|`D02837C2AD8EC739E8058B4ECFC479A3C537B85233653902C4E1069079ECE057`|
| `ui/editor/editor.go` |24,397|`B5D98594BF9213C0D2B0EF2B69C1C00040ED5E365FF133A11BF813AABFD9366A`|
| `core/text/composition_test.go` |12,624|`C57A16E814798699705F7C3F1062967E6E37E4434C80603E133E3C792F88A3B3`|
| `ui/editor/composition_test.go` |7,948|`CDACCCD4EBDE965CD3CCD2B8366B843FD483FDFB39B94AC8EB63B87D39ACFF7D`|

冻结后恰好运行一次组合test和一次匹配vet：

```powershell
native/scripts/go.ps1 -GoArguments @('test','./core/text','./core/document','./core/search','./core/session','./ui/editor','./modules/sourceview','./modules/markdownview','./modules/graphview','./app','-count=1')
native/scripts/go.ps1 -GoArguments @('vet','./core/text','./core/document','./core/search','./core/session','./ui/editor','./modules/sourceview','./modules/markdownview','./modules/graphview','./app')
```

| 运行 | 实际退出 | UTC区间 | 结果 |
|---|---:|---|---|
| 9包test |Go0 / shell0|11:32:49.5106114Z–11:33:33.3885214Z|9/9包pass；text0.966s、document1.353s、search1.323s、session2.636s、editor2.920s、sourceview2.406s、markdownview5.889s、graphview16.495s、app41.451s；test stderr为空|
| 匹配vet |Go0 / shell0|11:33:58.2801608Z–11:34:01.0655179Z|无诊断；stdout/stderr为空|

4/4源文件前后Length/Sha256一致。600次更新结构序列和240次Tester更新序列
建立给定表示与输入/回调行为；它们是源码测试，不是物理Windows IME事件。

### 组合历史合同

Buffer拥有的类型化组合token只持有事务身份，并绑定签发Buffer与当前组。
陈旧、跨Buffer或reset后的身份不能修改或取消外来事务；无token的类型化
更新/结束/取消会拒绝。连续完全相同的完整标记范围更新保留原始/当前后缀
端点，不切片中间旧文本；净零组合清理死引用，不增加假撤销，也不擦除此前
重做。相同文本的首/末输入和既有已接受标记回调保持原行为。取消、单步
撤销和重做仍作为一个事务工作，溢出核算/裁剪通过有意义测试。

非连续或不规则显式范围仍追加普通后缀历史，可能保留很多中间编辑。直接
reset时的token拒绝属于防御性覆盖，不是新支持的UI序列。没有通用活动历史、
进程RSS/堆或物理输入上限结论。

### 打包、宿主与闭包

一次正常打包调用完成五次Windows amd64 GUI构建，全部exit0且各流为空；
四个fixed engine清单/PE、14个portable source copy、26文件package-copy和
25载荷release manifest通过。首次aggregate即13/13，无重试或修复：五个
build、四个engine、一个hidden host和零worker request。401个scoped
source/build input共4,355,553字节，identity
`09A051CD60CCC8911EF93FE06DA00CA82A02199FD9EC1C232DF910C4090B4C0A`；
17个根源码门、35个冻结bitmap和26个patch-manifest项也未变。

一次精确复制的隐藏宿主使用隔离`C:`配置和已安装本地diagram-notes插件，按
命令行顺序打开UTF-8+BOM/CRLF `.diagramnote` 后PNG。运行始于
`11:40:08.5548603Z`、结束于`11:40:15.3618901Z`，wall6.8055567s；实际
存活settling为5,009.7743ms、46次检查。唯一owned`WM_CLOSE`前观察到两标签、
活动索引1的新653字节严格会话，SHA256
`B81A6532CA60C86C601E9CD0CB458B2D1899380B4B96AAB511126F0CE5F674D4`。
正常关闭以更新写入时间发布同一字节；host exit0，Job total1/active0，
process/Job句柄释放，无强制清理或worker请求。

该运行证明有序CLI打开/会话发布和正常关闭，不是第二次启动恢复。它不证明
物理IME、运行时大小拒绝交互、PNG渲染比较或整应用内存改善。原始记录见
[宿主结果](artifacts/acceptance/composition-release29-20261110/host-smoke/result.json)、
[`.29`交付](handoffs/composition-release29-delivery.md)和
[`.29`根验收](handoffs/composition-release29-root-acceptance.md)。

外部闭包共164文件/55,828,992字节：直接证据
`composition-release29-20261110`158文件/55,825,749字节，identity
`0F46070C6DFDD5CD30824376C2E6D4D9E459849DB45F9353366093EDF8388F89`；
attempt1捕获6文件/3,243字节，identity
`1F4E2193425B76BBD06BF5C929B7662BF43758E20EF4236D65D8B643EEA81242`。
[external closed inventory](artifacts/acceptance/composition-release29-20261110-closed-inventory.json)
为69,492字节，SHA256
`5BB1C746F4590C1DC40A01F255E4F88E1D5E1C24F4D230047A4BA9A73AA78854`。
闭包同时复核157个aggregate证据记录未变，并保留`.28`的26文件
27,395,153字节包、146文件55,786,384字节直接证据、6文件3,193字节捕获和
63,713字节外部清单原身份。

初始化attempt1在产品执行前因README必需短语跨行而exit1；原始command、
invocation、空stdout、336字节stderr和summary保存在
`artifacts/acceptance/composition-release29-20261110/initialize-attempt-1/`。
仅把检查拆成两个实际字面量并对changed helper运行一次语法parse；独立
attempt2随后exit0。失败记录和重试流均保留，产品阶段没有重复。

`.29`的端点保留只适用于连续完整标记范围，不构成通用历史上限或进程内存上限。
`.28`大小拒绝、有效重试、Esc取消和保存/切换/关闭屏障继续适用；已进入进程的
剪贴板、正则/编解码临时串及当前活动组仍占内存。没有物理Windows IME、第二次
启动恢复、worker执行、PNG渲染或RSS/工作集改善结论。

## 文档增长与 `.28` 历史交付：2026-11-10

历史入口
[bin/levnote-native-worker-20261110.28](bin/levnote-native-worker-20261110.28/README.md)，
版本`native-worker.20261110.28`。根代理已接受实际包、宿主结果和外部闭包；
`.27`包、直接证据、捕获目录和外部清单继续按原身份保护。

| 记录 | 字节 | SHA256 / inventory identity |
|---|---:|---|
| 26文件候选包 |27,395,153|`B2BB8C7EB09C2D917882FBFA424D8B921F4504619E37A5DE17E3B10B18E5D8CB`|
| 宿主`levnote-native.exe` |14,567,424|`8DB245432AB4C2358667B837A8D0CA7F164EA6CC8696C09D585AF194A2F04C64`|
| `release-manifest.json`，25载荷项，不含自身 |4,674|`999B4EAB52C6BC2E54D9ED74121CA3DBD4DFEE85CE9AAE678EF0C6297C895E24`|
| aggregate attempt1 |46,972|13/13PASS|
| attempt aggregate |46,305|`3F4983C85DD9B4A6E4EEE8080C33631F569181E705EA72B28A04B780AD07C478`|

### 冻结源码与最终test/vet

根源码接受和交付使用同一个11文件权威清单，而不是较早的implementation
清单。11个`Path/Length/Sha256`项共92,225字节；manifest
[source-frozen.json](handoffs/document-growth-budget-final-evidence-20261110/source-frozen.json)
为1,727字节，SHA256
`B0A212FF61660DDADD9C5A040289FA6057E52B36E3A030B91BA32196A7C8D275`。
[根源码接受](handoffs/document-growth-budget-source-root-acceptance.md)只
审阅delegated实际结果，没有重跑产品检查。

冻结后恰好运行一次组合test和一次匹配vet：

```powershell
native/scripts/go.ps1 -GoArguments @('test','./core/text','./core/document','./core/search','./core/session','./ui/editor','./modules/sourceview','./modules/markdownview','./modules/graphview','./app','-count=1')
native/scripts/go.ps1 -GoArguments @('vet','./core/text','./core/document','./core/search','./core/session','./ui/editor','./modules/sourceview','./modules/markdownview','./modules/graphview','./app')
```

| 运行 | 实际退出 | UTC区间 | 结果 |
|---|---:|---|---|
| 9包test |Go0 / shell0|11:06:35.6012270Z–11:07:07.5550564Z|9/9包pass；`app`27.455s，`graphview`16.658s，test stderr为空|
| 匹配vet |Go0 / shell0|11:07:24.7043468Z–11:07:25.9977153Z|无诊断；stdout/stderr为空|

测试前后11/11文件Length/Sha256一致，没有源修改或重复最终运行。最终验证结果
[document-growth-final-validation-result.md](handoffs/document-growth-final-validation-result.md)
为4,283字节，SHA256
`3447C944A9451DD8F3474B52FA67250A554F5F805DD075ECEC733BA1381ED1CC`；
证据文件闭包为
[evidence-files.json](handoffs/document-growth-budget-final-evidence-20261110/evidence-files.json)，
SHA256`ECFFD261571C83A5420530E28982B13B48F8E3DE5B5166BC256B74A16BEA37D6`。

这组源码测试建立以下实际行为：普通UTF-8逻辑内容有128MiB基值；较大已解码
种子完整保留并形成固定`max(基值,种子字节)`高水位；显式Restore重设种子；
撤销/重做/取消重放已接受状态。替换、反向ReplaceAll和缩进在局部提交前预检，
拒绝不改写部分命令；反向中间态超限可保守拒绝最终可容纳批次。Markdown/图形
字段写回被拒绝时本地草稿保留。大小拒绝的首个或进行中IME组合须有效重试或
显式Esc取消；焦点丢失不能清除屏障，被拒绝字符串不保存。后续IME合并/保留
不在`.28`。这些是MyGo headless源码seams，不是物理Windows IME交互。

### 打包、宿主与闭包

一次正常打包调用完成五次Windows amd64 GUI构建，全部exit0且各流为空；
四个fixed engine清单/PE和26文件package-copy通过。首次aggregate即13/13
通过，无元数据修复或重试：五个build、四个engine、一个hidden host和零worker
request。399个scoped source input共4,328,941字节，identity
`F01EED43E83BF3B54B295EBB575C953D537F9D1D155619F7D3818F526D4AC875`。

一次精确复制的隐藏宿主使用隔离`C:`配置和已安装本地diagram-notes插件，按
命令行顺序打开UTF-8+BOM/CRLF `.diagramnote` 后PNG。实际存活settling为
5,014.6165ms、46次检查；唯一owned`WM_CLOSE`前观察到两标签、活动索引1
的新643字节严格会话，SHA256
`CC71B5B507B9605EF277BAD701A86EE5C574DD7C0D2ADBDA489A6253FA9F28B9`。
正常关闭以更新写入时间发布同一字节；host exit0，Job total1/active0，
process/Job句柄释放，无强制清理或worker请求。

该运行证明命令行打开/会话发布和正常关闭，不是第二次启动的会话恢复测试。
它不证明物理IME、运行时大小拒绝交互、PNG渲染比较或整应用内存改善。
原始记录见[宿主结果](artifacts/acceptance/growth-release28-20261110/host-smoke/result.json)、
[`.28`交付](handoffs/growth-release28-delivery.md)和
[`.28`根验收](handoffs/growth-release28-root-acceptance.md)。

外部闭包共152文件/55,789,577字节：直接证据
`growth-release28-20261110`146文件/55,786,384字节，identity
`854427F529599D5912F316C61CD313147D0D15F1105EA9E78ED0C79394F96075`；
attempt1捕获6文件/3,193字节，identity
`504D8519A9A4DC2BAFFA9EA86F75B57BE2C0B5A427B711C6A8310669E3541A7E`。
[external closed inventory](artifacts/acceptance/growth-release28-20261110-closed-inventory.json)
为63,713字节，SHA256
`7ADEE2D137C4FC9CB61DCE6FC5C5B3BAF306F3C587B8E4D0700D7973C3FAEA46`。
闭包同时复核145个aggregate证据记录未变，并保留`.27`的26文件
27,382,619字节包、147文件55,816,795字节直接证据、6文件3,264字节捕获和
65,251字节外部清单原身份。交付后两个`.28`证据根冻结，文档任务不修改它们。

128MiB是普通逻辑内容边界，不是进程内存或全部文档总量上限。4MiB单草稿、
16MiB编码后会话、32MiB历史/图形解析、100标签和worker预算保持独立；已进入
进程的剪贴板、正则展开、编解码临时串及当前活动组历史仍占内存。一次不可用
shell的pre-package分派在PowerShell脚本启动前exit1，未产生stage输出或产品
执行；随后唯一package调用正常exit0。该包装失败保留在交付与docs记录。

## 文件工作流与 `.27` 历史交付：2026-11-10

当前入口
[bin/levnote-native-worker-20261110.27](bin/levnote-native-worker-20261110.27/README.md)，
版本`native-worker.20261110.27`。根代理接受完整26文件、27,382,619字节
目录，包目录identity为
`04740A5B7ABECA69FE5CBDE224B85C26EBC93C9BE418E95B0DC9B0D95AB5B247`。

| 记录 | 字节 | SHA256 |
|---|---:|---|
| 宿主levnote-native.exe |14,556,160|3B6734F30033569337A111353BD1BF661FB69F1782983688B0D3D863D0E8212C|
| release-manifest.json，25载荷项，不含自身 |4,674|C7783296AEEF59FD6254AF3F9487427658C40227BC5DAC15DF2348ED869926CF|
| 首次及最终validator，13/13 |10,800|781BC985F146387F6A8D81908D3C231CCE100EC3E8EA24F271FC5BEA393259D9|
| 最终aggregate |48,311|A9D5464CFEE88B424C8005D891FBFE17072938940A9F92B20527D96A50E2C077|

冻结的9文件组合先运行一次`app`测试，再运行一次`app` vet；两次都从
`F:\AIproject\LevNote`通过`native/scripts/go.ps1`调用，没有管道包装，
退出码就是实际Go进程退出码。

```powershell
& '.\native\scripts\go.ps1' -GoArguments @('test','./app','-count=1')
& '.\native\scripts\go.ps1' -GoArguments @('vet','./app')
```

| 运行 | 退出 | UTC区间 | 捕获结果 |
|---|---:|---|---|
| combined `test ./app -count=1` |0|2026-11-10T10:01:52.2123277Z–10:02:17.6814041Z；25.467s wall|`ok levnote.local/native/app 24.204s`；stderr0字节|
| `vet ./app` |0|2026-11-10T10:02:24.9932495Z–10:02:26.3645992Z；1.370s wall|stdout/stderr均0字节|

测试前后清单包含同样的9个`{Path,Length,Sha256}`身份，共160,306字节，
0项不一致。规范清单
[source-frozen.json](handoffs/file-workflow-final-evidence-20261110/source-frozen.json)
为1,348字节，SHA256
`940C698E0B0195258649E6A0747A8F1EBAF1F754BAE05E6539B790B023098DAF`。
命令、退出、四路输出和前后比较的原始记录见
[最终验证](handoffs/file-workflow-final-validation-result.md)与
[组合源码验收](handoffs/file-workflow-source-root-acceptance.md)。

一次正常打包调用完成五次Windows amd64 GUI构建，全部exit0；五个构建的
stdout/stderr为空。调用UTC
10:06:12.9303733Z–10:06:21.4554494Z（8.5207973s），exit0，外层
stdout67字节/stderr0字节。完整目录、四个引擎清单、五个PE、392个构建输入、
便携资产、相对链接、许可和补丁归于检查均通过。首次aggregate即13/13通过，
无失败尝试、元数据修复或重试。实际调用和产物见
[package-command.json](artifacts/acceptance/file-workflow-release27-20261110/package-command.json)、
[首次validator](artifacts/acceptance/file-workflow-release27-20261110/aggregate-attempts/attempt-1/validator.json)、
[aggregate.json](artifacts/acceptance/file-workflow-release27-20261110/aggregate.json)、
[`.27`交付](handoffs/file-workflow-release27-delivery.md)和
[`.27`验收](handoffs/file-workflow-release27-root-acceptance.md)。

一次精确复制的隐藏宿主使用隔离配置，按顺序打开UTF-8+BOM/CRLF自定义
格式文本和PNG，46个存活检查共等待5,005.7892ms。它在唯一owned
`WM_CLOSE`之前已写入两标签、活动索引1的657字节严格会话；关闭请求前捕获
UTC10:07:40.5695238Z，关闭请求为10:07:40.5751476Z，正常关闭以更新的
写入时间重新发布同一字节，SHA256
`D796A3CCABF6C13FC35A0DBD81298593847B5009B038A633942BE2519B97CE43`。
宿主exit0，Job total1/active0，process/Job句柄已释放，无强制清理或
worker请求。原始记录见
[宿主结果](artifacts/acceptance/file-workflow-release27-20261110/host-smoke/run-1-mixed-cli-open/result.json)。

重开与标签准入行为由冻结源码的聚焦测试建立：`Ctrl+Shift+T`/原生菜单使用
本次运行最近16个已保存本地文件路径与接收标识，重开当前磁盘内容；重复路径
移到顶部，失败保留对应记录并可显式重试，不恢复已放弃修改或草稿。真实
Markdown冲突视图在容量拒绝两次后保留草稿，关闭一个标签后恢复。新建、
模板、恢复、普通/批量打开、字节回退和异步二进制采用共用
`session.MaxTabs=100`；满额只拒绝新路径，已有路径复用，批量打开继续，
拒绝不改写源文/版本/历史/选区/恢复文件或二进制所有权，混合恢复按序接受
恰好100条。最终准入组为app `2.420s`，既有open-many/重开/图片回归组为
app `3.267s`，均exit0。证据中的004–007是保留的测试构造失败，不是产品
回归；无原始日志的初始release-order报告只作为报告历史，不作为测量失败
证据。合同与独立审查见
[重开源码验收](handoffs/reopen-closed-file-root-acceptance.md)和
[准入源码验收](handoffs/live-tab-admission-root-acceptance.md)。

这些证据验证了`.27`已交付的程序包和一处有界启动/会话/关闭生命周期，但
不证明交付程序中的物理键盘/IME交互、内嵌图片渲染、100标签RSS或其它资源
表现、dirty恢复、第二次恢复或跨机器可复现性。100只是标签数量上限；128 MiB
只限制文本源文件读取，活动缓冲区增长准入未在`.27`实现，不能把两者合成为
内容或内存上限。详细功能边界见[原生README](README.md)。

## Markdown图片引用与 `.26` 历史交付：2026-11-10

历史入口
[bin/levnote-native-worker-20261110.26](bin/levnote-native-worker-20261110.26/README.md)，
版本`native-worker.20261110.26`。根代理接受完整26文件、27,281,492字节
候选目录，identity为
`7DB6E2660206EABC4D689894A0B1D50400C6DFA8BAA71CBD3689C6055EE1FB64`。

| 记录 | 字节 | SHA256 |
|---|---:|---|
| 宿主levnote-native.exe |14,455,808|B219024B7C5B444DA0307579C1500821A6B24FDCDD9B4D47E0FC14E863A51B67|
| release-manifest.json，25载荷项，不含自身 |4,674|EFF8DD2FAF05EC620AB5976C2A72E647F12570876B15B13606062B7DC9C4E20E|
| 最终aggregate |59,808|CA89586C05515313CEB75AA17744FA9A3EA04C11AA247B686AC5F7C37C9ACC8E|
| 修复记录portable-link-repair-result.json |39,262|7BE8218A41ACDD2AB4C9C59F889E785D9DE8EC19C097D4EF683D9B23C5280014|
| closed external inventory |84,468|DAFE2B31DF426BAC111AB2179A031234EA90BD28E912E624E055661C6928B27D|

最终validator为13/13通过。一次打包调用先完成五次Windows amd64 GUI
构建，五个构建均exit0且输出流为空；随后便携链接检查把README中的行内
代码示例误判为本地链接，整个打包调用exit1。修复只改源README中的该示例
行及候选README/发布清单；五个可执行文件、四个引擎清单和其余包资产保持
逐字节一致。原始上下文、输入清单、失败退出/输出流、缺失的原始
package-result文件和失败清单保留，未重复任何产品构建或运行。

修复解析历史保留：第一次辅助脚本语法解析失败且摘要计数错误，修正后
通过；一次元数据采用顺序失败；最终受控解析26个脚本零错误。聚合attempt1
缺少实际HostSHA256，在validator/aggregate创建前exit1，没有12/13结论；
attempt2仅修正聚合元数据，13/13通过。
未重复打包、构建、测试、vet、worker请求或产品运行。

一次隐藏精确复制的宿主使用全新隔离配置，按顺序打开UTF-8+BOM/CRLF
自定义格式文本和PNG；在它唯一拥有的`WM_CLOSE`之前，写入679字节严格
混合会话，随后正常关闭以更新的写入时间发布同一字节。宿主exit0，Job
total1/active0，owned handles已释放，无强制清理、进程流或worker请求。
该运行证明这个严格生命周期和两次会话字节一致，不证明物理图片引用
输入、内嵌图片渲染、物理IME、dirty恢复、第二次恢复或进程资源表现。

显式图片引用的源码/合成UI、应用和in-process worker测试建立行为：
原始`![label](target)`语法保持可见，Ctrl/Cmd+点击或Tab+Enter可打开；
目标边界、本地路径保留、HTML转义、PNG/JPEG边界和字节回退被覆盖。
精确源码验收、交付和证据边界见
[源码验收](handoffs/markdown-image-reference-root-acceptance.md)、
[包验收](handoffs/markdown-image-reference-release26-root-acceptance.md)、
[交付](handoffs/markdown-image-reference-release26-delivery.md)和
[修复审查](handoffs/markdown-image-reference-release26-repair-review.md)。

闭合证据共189文件/55,735,872字节：直接177文件/55,729,781字节identity
`ED5BE339D567A45C5F954AC42DAB3B5F78254BDBB818C6BE9949FB7FCBCF1296`；
attempt1捕获6文件/2,716字节identity
`0EF46BFEE7C164FD711C8BC7401EE93EEF35758887532A50C57613FEDC7AB8D5`；
attempt2捕获6文件/3,375字节identity
`2BD33DA06E8EC8E6C30C0C40106A300F985420A88DD39B00149F464A61B4B2F3`。

## Markdown表格列与 `.25` 历史交付：2026-11-10

历史入口
[bin/levnote-native-worker-20261110.25](bin/levnote-native-worker-20261110.25/README.md)，
版本`native-worker.20261110.25`。根代理接受完整26文件、27,259,765字节
目录，包目录identity为
`07391CBF37FF6406D5D4279CCED96CDBA0FDD3F5829B9685AA0066AA46CC7204`。

| 记录 | 字节 | SHA256 |
|---|---:|---|
| 宿主levnote-native.exe |14,443,520|2421D8FC98B144DCCB0ECC1CFF239A18E11DDEF05F12FF0505B307058419AE49|
| release-manifest.json，25载荷项，不含自身 |4,674|CA064C3E7824EDCB179465CD8E60D982D057D2EA1D45A0E8407BD44508CF9BC2|
| 最终aggregate |49,349|993C171AD4BD07772CE6695C1A9F7B62275504708C7CADA56F4B7A152E1FE2A5|
| attempt-1 validator |11,505|D8E187B41D779F4A5D21ECA57351CBB89B7BBCFFB1AE1C4B212F91A4FCD8B0F9|
| closed external inventory |66,343|39BADBA047C552252A13AD7645FA19D93B70DDF4FA8414FEB9B8E8362B3C99DF|

最终validator为13/13通过，attempt1无重试或元数据修正。四个引擎manifest、
385个构建输入匹配，14个便携副本核对。一次打包调用生成五个GUI可执行
文件，全部exit0。

一次隐藏精确复制的宿主使用全新隔离配置，按顺序打开UTF-8+BOM/CRLF
自定义格式文本和PNG；在它唯一拥有的`WM_CLOSE`之前，已发布675字节严格
有序混合会话，活动索引1，会话SHA256
`21A967D39B5F1D2A0C530C776D30782969ED72EE58A507EA588D8F60AE2A3E2C`。
捕获时间为2026-11-10T07:37:33.5751458Z，关闭请求时间为
2026-11-10T07:37:33.5804387Z。正常关闭以更新的写入时间发布同一字节，
exit0，Job total1/active0，无强制清理、owned handles已释放，进程流
为空，且无worker请求。

源码测试建立表格列编辑、事务与BOM+CRLF保存重开行为；该隐藏宿主证明
干净文本+PNG会话在正常关闭前通过真实调度完成恢复写入，随后正常关闭
重新发布。它不证明物理表格交互、物理输入/IME、截图/驱动行为、渲染或
进程资源改善。表格列源码验收、一次合成UI截图和最终11/11源身份保持见
[源码验收](handoffs/markdown-table-columns-root-acceptance.md)、
[最终验证](handoffs/markdown-table-columns-final-validation-result.md)。

闭合证据共152文件/55,543,248字节：直接146文件/55,539,894字节identity
`F383C6A160D3B56F55A1EEF0B4A083693C1C09EF2F97910AE06C37DDC5BC9646`；
捕获同级6文件/3,354字节identity
`FCD2AE8E96784802DBA7C3A193D350634F1F8E281EC97507799B70CA3E27ABAF`。
精确验收、交付和证据边界见
[包验收](handoffs/markdown-table-columns-release25-root-acceptance.md)、
[交付](handoffs/markdown-table-columns-release25-delivery.md)、
[源码验收](handoffs/markdown-table-columns-root-acceptance.md)和
[最终验证](handoffs/markdown-table-columns-final-validation-result.md)。

## 图片键盘、恢复调度与 `.24` 历史交付：2026-11-10

历史入口
[bin/levnote-native-worker-20261110.24](bin/levnote-native-worker-20261110.24/README.md)，
版本`native-worker.20261110.24`。根代理接受完整26文件、27,226,613字节
目录，包目录inventory identity为
`C17C919EB5F71C0643CAA9BBF1F5C9CCD509EE2CC91DD8A7FD704F15D06427B5`。

| 记录 | 字节 | SHA256 |
|---|---:|---|
| 宿主levnote-native.exe |14,410,752|A4D679FB0F35DB9258410DCC35995853780DB4E915C928E8D6434A92E6A0BD08|
| release-manifest.json，25载荷项，不含自身 |4,674|B1609AE10D15DBEA63DD40CC7E699E131E49156FC48821D8FD63891E565D935E|
| 最终aggregate |51,028|D512E569D75D5250F69A6F358D9455BB48E56A8444A439BA53D4B93233E3DA9E|
| attempt-2 validator |10,605|07ACBC85744DA61AC6348D156294F7226B6026D148BBE637047B1DC282FFF77F|

最终validator为13/13通过。四个引擎manifest、380个构建输入匹配，
14个便携副本核对。一次打包调用生成五个GUI可执行文件，全部exit0。

一次隐藏精确复制的宿主使用全新隔离配置，按顺序打开UTF-8+BOM/CRLF
自定义格式文本和PNG；在它唯一拥有的`WM_CLOSE`之前，已发布659字节严格
有序混合会话，活动索引1，会话SHA256
`800045175388CAB4DF70B11A5650D8072CA70CB8A7AE0EF02C943B71C5CE2E04`。
捕获时间为2026-11-10T06:42:12.8862702Z，关闭请求时间为
2026-11-10T06:42:12.8920834Z。正常关闭以更新的写入时间发布同一字节，
exit0，Job total1/active0，无强制清理、owned handles已释放，进程流
为空，且无worker请求。

源码测试建立脏缓冲与图片键盘逻辑；该隐藏宿主证明干净文本+PNG会话
在正常关闭前通过真实调度完成恢复写入，随后正常关闭重新发布。它不
证明物理最小化编辑/IME、截图/驱动行为、渲染或进程资源改善。首次
合并测试中的图片100%模式回归失败已修正，修正后通过见
[源码验收](handoffs/native-autosave-keyboard-root-acceptance.md)。

aggregate attempt1仅因时间转换丢失小数秒而12/13通过、仅1项失败；attempt2
只修正三个时间转换。初始外部inventory footer总计为零，修正后由已经
正确的逐根记录推导总计。两次修正均未重复执行产品工作。精确验收、
交付和证据边界见
[包验收](handoffs/native-reading-release24-root-acceptance.md)、
[交付](handoffs/native-reading-release24-delivery.md)和
[源码验收](handoffs/native-autosave-keyboard-root-acceptance.md)。

## 图片阅读与 `.23` 历史交付：2026-11-10

历史入口
[bin/levnote-native-worker-20261110.23](bin/levnote-native-worker-20261110.23/README.md)，
版本`native-worker.20261110.23`。根代理接受完整26文件、27,214,997字节目录，
包目录identity为
`180786B864A37AAC6423B42C95F079D8AA0AF5B18BE4504F2A124DDECF184040`。

| 记录 | 字节 | SHA256 |
|---|---:|---|
| 宿主levnote-native.exe |14,400,000|837A29744F0C833308B5ACC668B0F5FB2628B0710A816BF8D74DB0B516742C86|
| release-manifest.json，25载荷项，不含自身 |4,674|B0650A1E9A90ACAC7A68457252DA559938B1CB6D33E4B1279DE23E2688325FD6|

35个受保护源/构建输入已核验，它们不是随包发布的35个资产；实际便携
副本为14。一次打包调用完成五次Windows amd64 GUI构建。两次隐藏宿主
运行共用一份精确复制的包与隔离配置：第一次以CLI混合打开文本和PNG，
第二次无参数恢复已保存的混合会话；均通过同一个owned `WM_CLOSE`正常
退出，且无worker请求。根代理接受13/13验收门。

源码测试与合成Tester截图建立行为与布局证据；隐藏宿主运行建立打包启动、
干净关闭和混合会话恢复。它们不证明物理像素、物理输入/IME、驱动行为、
RSS/VRAM/CPU或广泛内存优势。

已知`.23`限制：正常关闭持久化已验证；自动恢复当前依赖View帧，没有
独立于绘制的期限保证。后续分配的源码修正不追溯改变这个不可变`.23`
二进制。当前合同只覆盖已提交的document-buffer恢复，不宣称捕获未提交的
格式本地草稿。见
[包验收](handoffs/native-image-release23-root-acceptance.md)、
[交付](handoffs/native-image-release23-delivery.md)和
[隐藏自动保存审查](handoffs/native-hidden-autosave-review.md)。

## 标签工作流与 `.22` 历史交付：2026-11-10

历史入口
[bin/levnote-native-worker-20261110.22](bin/levnote-native-worker-20261110.22/README.md)，
版本`native-worker.20261110.22`。根代理接受完整25文件、26,863,393字节目录，
包目录identity为
`BA43209FF7F6CFBEDD8287DBEA7B642008AFA839D133FECD8CC5C9D065FD9CF6`。

| 记录 | 字节 | SHA256 |
|---|---:|---|
| 宿主levnote-native.exe |14,101,504|085E65F7BA1386A01E58CFCB26F0C069AACBFFD47AA063974CCDF0CC99C0C19F|
| release-manifest.json，24载荷项 |4,490|4F66B7DC96E7213DF728E44A44B3DA77AE2C293BC5C08E10315156DB4793671F|
| validator.json，14/14 |6,178|BD66B4463E43ED0F04DF40D664AB8BFDF53FB923B32C7844F752CE4B6D1FCA22|
| aggregate.json |43,361|8438EC87DD42319A103C8F27B9CD5AB9DB0921355323957E45C1424E2C158D4D|
| 独立closed-inventory.json |57,245|E61FB97AA24C5DAD6673F365D32EB77398C0D98EE32FE456A13F9B45A75373E2|

`Ctrl+Tab`/`Ctrl+Shift+Tab`按视觉顺序回绕，零/一标签无副作用。实际选择
变化时活动标签按钮显示一次，手动横向滚动不被普通重绘拉回。源码/富视图
继续通过既有Select/Flush提交；失败保留当前标签、草稿、撤销及导出任务。
可见Find查询/替换或GoToLine字段正在组合输入时，快捷键显示明确提示并
保留焦点/preedit，完成或取消后必须重按，不排队切换。关闭按钮在遍历后
执行，避免同步关闭清空切片尾部后继续访问。

根代理发现GoToLine鼠标失焦后仍残留组合标记；保留的15号日志先复现失败，
随后按当帧公开焦点/组合状态清理，16号双方向指针回归通过。最终一次
`go.ps1 -GoArguments @('test','./app','-count=1')`为17.278s/exit0，
`go.ps1 -GoArguments @('vet','./app')`exit0；见
[18测试日志](artifacts/tab-workflow-20261110/18-corrected-final-app-test.log)、
[19静态检查](artifacts/tab-workflow-20261110/19-corrected-final-app-vet.log)。
[760×480测试器图](artifacts/tab-workflow-20261110/tab-active-wrap-760x480.png)
显示首尾回绕后tab13按钮完整可见；覆盖真实关闭按钮路由和手动滚动的合成
测试及全部保留失败见[实现结果](handoffs/tab-keyboard-implementation-result.md)、
[显示结果](handoffs/tab-active-reveal-result.md)和
[源码验收](handoffs/tab-workflow-root-acceptance.md)。

一次既有打包脚本调用完成五次Windows amd64 GUI构建，均exit0且输出流为空；
四个引擎清单、五个PE、13项资源复制、许可/相对链接及README版本渲染通过。
一次精确复制的宿主隐藏启动，PID2480、held handle1720，创建时间
2026-11-10T01:24:37.8337669Z；通过该句柄核对映像路径。一次owned WM_CLOSE
后exit0，无强制清理或残留owned进程，空stdout/stderr。Job TOTAL processes1
表示没有worker请求。隔离C:配置中只有一条diagram-notes路由的干净
UTF-8/BOM预览会话，无draft/终止换行；CRLF源文件、安装JSON、候选及副本
前后字节/时间相同。321输入共3,248,618字节及11个标签验收门保持不变；
受保护`.21`包25文件、直接证据202文件及retry兄弟目录4文件均未改。

本次交付无失败或重试。[最终证据](artifacts/acceptance/tab-workflow-release22-20261110/aggregate.json)
134文件/54,340,909字节；外置aggregate-capture兄弟目录6文件/2684字节，
不在运行时哈希活动日志。所有外层脚本退出后，133条聚合记录再次核对未变；
[关闭后清单](artifacts/acceptance/tab-workflow-release22-20261110-closed-inventory.json)
覆盖两个目录合计140文件/54,343,593字节，不含清单自身。
准备阶段最早解析失败已被前任覆盖、旧exit记录为空，证据缺口如实保留；
最终十脚本解析通过，exit文件为一字节0x30，见
[准备检查点](handoffs/release22-final-preparation-checkpoint.md)。

[包验收](handoffs/tab-workflow-release22-root-acceptance.md)、
[交付](handoffs/tab-workflow-release22-delivery.md)和
[14/14核验](artifacts/acceptance/tab-workflow-release22-20261110/validator.json)
区分行为测试与打包初始化/关闭证据。没有物理Windows输入法、打包交互导出、
dirty恢复或整应用内存/CPU结论；worker行为沿用`.21`，本次未运行资源矩阵，
也未提供PNG/JPEG阅读器。文档更新不修改冻结包、便携说明或证据。

## draw.io告警收窄、源码所有权与 `.21` 历史交付：2026-11-10

历史入口
[bin/levnote-native-worker-20261110.21](bin/levnote-native-worker-20261110.21/README.md)，
版本`native-worker.20261110.21`。根代理接受完整25文件、26,859,809字节目录；
宿主14,097,920字节，SHA256
`2C400D3730AEE64FC380CA0312B44BAAB1C0803AA70BA96525804BB4B436B329`；
发布清单4,490字节、24个载荷项，SHA256
`C2556B0E372466B5F67FEBF7C9FBACBA0D44050B00F9892C64837ED16CE9A824`；
validator7,382字节，SHA256
`48DB08D815E5A4E0888E29D79BF797BA4EDD6C92576B71F1768B0F5B35ECBD69`；
aggregate18,214字节，SHA256
`E34D2FE39096978E88FB5DB2B868EEB9B96315EA141D73E879B0910F191D1FB8`。
包目录identity为
`3126E7898073CEBBC02EAB907EEF541C8A3D5538F0B77A45D58B6CE9DBE72ED7`。

`.21`包含两项已接受的draw.io源码变更。结构容器告警只识别按顺序出现的
精确0/1空容器对，XML空白和注释不影响判定；新建空图纸不再报告两条泛化
容器告警，其他不支持内容仍可见，这不是通用图层/分组支持。源码所有权优化
先归一化选项并检查上限再克隆调用方输入；私有原始/初始当前源文可共享
不可变自有字节，公共`Source`独立，实际编辑后原始/当前/公共源文保持三份
不同字节，超限拒绝、失败回滚、只读压缩页、序列化字节、元数据和防御性
返回保持不变。

同一33,985字节基准fixture在前后使用，SHA256
`04E533B0DA533713F51E106F820A7F4A62D0E0F06CA910EB6FEAA9857A3D3E9F`，
包含8页、169个模型节点、7个未压缩页和1个压缩只读页。固定20次、三样本、
`-benchmem`的中位累计分配：

| 子基准 | pre B/op | post B/op | pre allocs/op | post allocs/op |
|---|---:|---:|---:|---:|
| `ParseRetained` | 764,244 | 722,520 | 8,335 | 8,333 |
| `UpdateSerialize` | 1,733,705 | 1,487,174 | 16,688 | 16,681 |

`B/op`和`allocs/op`只表示每次操作的累计分配，不是保留堆、工作集、RSS、
整应用内存或速度。观察到的中位耗时为Parse 1,082,480→1,244,525 ns/op、
Update 2,291,545→2,478,320 ns/op；这些短样本较慢，但不是速度结论。

一次打包调用完成五次Windows amd64 GUI构建，全部exit0；319个源/构建输入
共3,183,116字节、六个源码门和两个基准fixture保持不变。实际installed
draw.io worker来自导入的bundled-origin包，请求516字节源文、页索引1、
页面ID`second`，返回同一598字节SVG（SHA256
`3F990A4AAC3209BAFAD94F44026BF14F3CEC27EA44BDBD3169555947936472BF`），
原始`Warnings:null`/规范化`[]`、计数0，exit0并joined关闭，无fallback。

另一次隐藏精确宿主副本使用run-owned普通`C:`配置和179字节
`diagram-notes.json`，加载`.diagramnote`路由，持久化425字节干净UTF-8+BOM
预览会话；一次owned `WM_CLOSE`后exit0，无超时/强制清理或残留owned进程。
Job的TOTAL processes为1只表示本次宿主启动未派生worker，不表示退出后仍
有进程存活。

直接证据目录202文件、91,722,328字节，identity
`55B150C3A423BC3D213E7CF7282F1D87F9247AD35768236144D9217E7F4CE79B`；
单独的aggregate重试兄弟目录4文件、1,635字节，identity
`FA8CDD03BA1AC32257A97403BE861DC82E9EE3EF2443866D7860B6BC1EF6AB16`。
首次聚合15/16/exit1保留：复制件错误读取`SourcePreserved`并尝试哈希自身
活动重定向日志；修正为`FixturePreserved`并排除活动流后，仅重试聚合，exit0
且stderr为空，未重放产品阶段。源码、包与原始核验见
[包验收](handoffs/drawio-efficiency-release21-root-acceptance.md)、
[交付记录](handoffs/drawio-efficiency-release21-delivery.md)、
[结构警告验收](handoffs/drawio-structural-warning-root-acceptance.md)、
[结构警告结果](handoffs/drawio-structural-warning-result.md)、
[结构警告独立审查](handoffs/drawio-structural-warning-independent-review.md)、
[源码保留验收](handoffs/drawio-source-retention-root-acceptance.md)、
[源码保留结果](handoffs/drawio-source-retention-result.md)、
[独立审查](handoffs/drawio-source-retention-independent-review.md)、
[validator.json](artifacts/acceptance/drawio-efficiency-release21-20261110/validator.json)
和[aggregate.json](artifacts/acceptance/drawio-efficiency-release21-20261110/aggregate.json)。
无物理UI/IME、打包交互式导出、dirty恢复、安装器、跨平台或进程内存/CPU
结论。后续CtrlTab标签工作流见上方`.22`记录，不包含在`.21`本包。

## schema3 draw.io数据路由与 `.20` 历史交付：2026-11-10

历史入口
[bin/levnote-native-worker-20261110.20](bin/levnote-native-worker-20261110.20/README.md)，
版本`native-worker.20261110.20`。根代理接受完整25文件、26,854,177字节目录；
宿主14,095,360字节，SHA256
`4B776F634A3626F2BDD15303E175FE38CC4908D566AC54A6D25F51D989B4F34E`；
发布清单4,490字节、24个载荷项，SHA256
`53C0B71C317734001596988F6AB2BF44201879CC322BA9DB3B9AB494E553DBFA`；
validator5,418字节，SHA256
`5B4FD713AD5A14BAE8E2D5266CBF55B66802D3161151660509844A8662CC1455`；
aggregate24,651字节，SHA256
`C1CAEC5205F8A833A4DFF86B84485E4D84187CAC3BD6D3DA88DA1F47D7E59611`。
完整清单identity（相对路径、字节数、SHA256和记录修改ticks）为
`6B899EFB2FB2A90CB2FB93CFE9BC48A3B94B4CFA2AB16C61C5AB005B05DF14ED`。
证据目录139文件、54,304,954字节，identity
`5ED2F19531E178E33F355164546AB50C5C31F37E032F9E8B7CCBB6791C9C8621`。
源码、包与原始核验见
[源码路由验收](handoffs/drawio-route-root-acceptance.md)、
[实现结果](handoffs/drawio-route-implementation-result.md)、
[独立核对](handoffs/drawio-route-independent-review.md)、
[包验收](handoffs/drawio-route-release20-root-acceptance.md)、
[交付记录](handoffs/drawio-route-release20-delivery.md)和
[validator.json](artifacts/acceptance/drawio-route-release20-20261110-031456/validator.json)。

`.20`把schema3扩展到三个数据目标：`builtin.markdown`、`builtin.mindmap`
与`builtin.drawio`。配置是本地JSON，不安装可执行程序；`builtin.drawio`
让自定义后缀使用内置draw.io编辑/预览与当前页SVG，不提供Outline或
只读Scene。文件内容必须本身是draw.io XML；SaveAs按新路径重匹配但不转换
正文。配置ID与适配器ID分开，安装配置独立启停，被遮蔽、禁用或同ID重扫
会使旧能力身份失效；失败或待写草稿按既有Flush/复核屏障阻止操作。

一次精确复制隐藏宿主启动使用run-owned普通`C:`配置、179字节
`diagram-notes.json`和893字节UTF-8+BOM/CRLF、含未知元数据的
`.diagramnote` XML。PID31892收到一次owned`WM_CLOSE`后exit0；无超时或
强制清理，job前后进程数均为1，未发生worker请求。会话427字节，恰有一个
匹配路径、`pluginId=diagram-notes`、`view=preview`、UTF-8与BOM标记的
干净单标签，无草稿。文件、安装配置、候选与完整副本前后字节不变。
该运行证明配置加载、路径路由、会话持久化与干净生命周期，不证明可见
渲染、物理UI/IME、交互式导出、dirty恢复或资源表现。

`.20`的一次打包调用完成五次Windows amd64 GUI构建，全部exit0且
stdout/stderr为空；12/12聚合核验、315个源/构建输入（3,153,585字节）、
六个路由门、三个路由证据目录、七个插件示例和完整`.19`历史包均通过或
保持不变。实际managed installed draw.io worker运行证据继承自`.19`；
`.20`没有重新请求worker。后续警告/source-retention源码变更不在本包。

## draw.io当前页worker与 `.19` 历史交付：2026-11-10

历史入口
[bin/levnote-native-worker-20261110.19](bin/levnote-native-worker-20261110.19/README.md)，
版本`native-worker.20261110.19`。根代理接受完整25文件、26,852,621字节目录；
宿主14,094,848字节，SHA256
`BB845BF6FDE47D16804AEFA033FBAF5B85A507FF71147E06F467A636E9F61CE2`；
发布清单4,490字节、24个载荷项，SHA256
`523C2BF682C9E9527FDDF972BC3CEE2C93D22E49114CA8FB52A2DF5D4C4ABFA4`；
validator5,616字节，SHA256
`2FDF7329E289719ABCC14361A916E0E95442D3BB52E98CF05B057C409611AFE5`；
aggregate28,725字节，SHA256
`91DB69CDA656C177A925178EC30239D693166DFB0BD722BF79B587ED1D10B2BC`。
完整清单identity（相对路径、字节数、SHA256和记录修改ticks）为
`7925942AFDE49F63C2D77D79EBE34FD3F1BAD18F535745E30171AFA3CD27A24B`。
证据目录195文件、91,652,331字节，identity
`4CFA46B862E2798CCEB0A49645E1096044465C9D77AC291509D072C48523D7F4`。
源码、包与原始核验见
[Stage2源码验收](handoffs/drawio-worker-stage2-root-acceptance.md)、
[Stage2结果](handoffs/drawio-worker-stage2-result.md)、
[测试补验](handoffs/drawio-worker-stage2-test-followup-result.md)、
[独立审查](handoffs/drawio-worker-stage2-independent-review.md)、
[包验收](handoffs/drawio-worker-release19-root-acceptance.md)、
[交付记录](handoffs/drawio-worker-release19-delivery.md)和
[validator.json](artifacts/acceptance/drawio-worker-release19-20261110-023148/validator.json)。

`.19`新增第四固定draw.io SVG worker：阅读/编辑或分屏中选择当前页后，
宿主准备零基页面索引与页面ID并交给worker。页面索引为0..249,999、
ID最多512字节，源32MiB、SVG64MiB、默认全请求30秒；raw XML解析有界
但会分配解析模型。页面缺失、超限或协议失败没有部分结果。宿主在
导出开始时由`startEngineExport`捕获标签/文档/Buffer/编辑器/适配器/
能力/名称/模式/视图所有者并执行输入Flush；随后的`PrepareExportRequest`
只可能同步过期的解析缓存，并报告当前页与视图epoch，不Flush、发布或
丢弃待写草稿。发布前用`ExportRequestToken`读取已捕获身份并复核。
页面A-B-A、同索引重解析、Release、视图/模式/标签/文档/Buffer/
编辑器或版本变化都拒绝发布并释放载荷与警告。待写回字段的初始Flush
合并为一次有意的撤销事务，因此改变Buffer、脏状态和撤销历史；无待写回
草稿的干净导出不改Buffer、选区、脏状态或历史。原始磁盘字节、编码
和BOM不被导出本身改写。

本地引擎包固定catalog为Markdown HTML、导图SVG、只读导图场景和draw.io
SVG四个引擎；每个引擎最多保留12个已安装包，导入不启用。schema1旧
三项选择按原字节读取，draw.io默认随包；显式启用/重置/修复后持久化
schema2四个精确选择项。选择文件损坏、单个坏包与cleanup poison语义
保持，不提供删除UI。

一次打包调用完成五次Windows amd64 GUI构建，全部exit0且stdout/stderr
为空：Markdown导出3,303,424字节、导图SVG3,126,272字节、导图预览
3,140,608字节、draw.io SVG3,121,664字节、宿主14,094,848字节。五份产物
均为Windows amd64 GUI PE。314个源/构建输入共3,133,071字节前后相同。

实际managed请求使用从`.19`候选包导入并显式启用的第四引擎，而不是
 bundled active selection。请求选择零基索引1、页面ID`second`，返回
598字节SVG和两条有序`UNSUPPORTED_ELEMENT`警告；worker PID23728正常退出，
joined lifecycle关闭且无回退。另一份字节相同宿主副本从`F:`启动，
run-owned配置位于普通`C:` temp卷，PID3712收到一次owned`WM_CLOSE`后
exit0；无worker操作，保留一个干净UTF-8单标签会话，源文件不变。

Stage2合成应用/Tester证据覆盖页面准备、token、视图/所有者/版本拒绝、
schema1/schema2选择、协议边界和端到端导出路径；这些是测试证据，不等同
于本次交付的两次实际运行。本轮没有验证物理Windows输入/IME、dirty
recovery、进程内存/CPU、延迟或打包应用中的交互式导出；`.18`及更早证据
保持历史。validator与aggregate为12/12、exit0，无重试。

`.19`当时schema3仍只接受`builtin.markdown`与`builtin.mindmap`；
`builtin.drawio`由后续`.20`验收。

## draw.io SVG输出预算与 `.18` 交付：2026-11-10

当前入口
[bin/levnote-native-worker-20261110.18](bin/levnote-native-worker-20261110.18/README.md)，
版本`native-worker.20261110.18`。根代理接受完整23文件、23,699,526字节目录；
宿主14,075,904字节，SHA256
`BFA6F460F49087390F114CEC2CF2ADC034307B07769D8B64E3E70C78BBDCB44A`；
发布清单4,137字节、22个载荷项，SHA256
`8CA2016C0064638BE965FDFE74468F8004A37143BE568A189FD7FEDAB2D12FDD`。
完整清单identity（相对路径、字节数、SHA256和记录修改ticks）为
`4A4F9BD0B6DF2DDCD1E597F026AAC985A9D50711CDD85F3E9310F7DC5B0B2991`。
源码、包与原始核验见
[源码验收](handoffs/drawio-svg-budget-root-acceptance.md)、
[实现结果](handoffs/drawio-svg-budget-result.md)、
[独立核对](handoffs/drawio-svg-budget-independent-reconciliation.md)、
[包验收](handoffs/drawio-svg-budget-release18-root-acceptance.md)、
[交付记录](handoffs/drawio-svg-budget-release18-delivery.md)和
[validator.json](artifacts/acceptance/drawio-svg-budget-release18-20261110-011852/validator.json)。

`drawio.SVGOptions.MaxBytes`成为输出合同的一部分：零值选择64MiB
`MaxSVGBytes`默认硬限，较小的正数被采纳，较大的正数夹取到64MiB，负数返回
`ErrSVGOutputLimit`。`ExportSVG(pageIndex, options)`签名不变，页码查找仍先于
输出预算，因此无效`pageIndex`保持原有错误优先级。生成器先用私有计数writer
遍历全部输出和XML转义，不创建完整SVG计数缓冲；确认边界后再按精确大小分配
一次并填充。填充长度不一致、超限和所有外层错误都返回nil SVG与nil issues，
不返回部分结果。问题列表只在计数遍收集，因此成功字节、警告内容和顺序保持；
恰好等于上限也走成功路径。普通UI仍通过现有导出错误界面显示失败。

这些证据只约束序列化结果字节，不是进程内存上限。源文/解析模型、页面索引、
问题与点序列、固定UTF-8安全转义片段和成功输出缓冲仍消费内存；计数与填充
两遍会重复部分确定性几何计算。没有整应用内存、CPU、速度、物理Windows
输入/IME或打包宿主SVG交互结论。draw.io仍在宿主导出，`.18`包仍是宿主加
三个固定worker；并发源码中的第四worker不属于本包。

生产/UI冻结后的首次完整受影响套件曾通过：draw.io包PASS0.721s、
graphview PASS10.850s，exit0；随后那次scoped vet也无输出。该轮证明当时的
实现、精确oracle、边界和graphview所选页导出，但之后独立审查仍改了
`svg.go`与`svg_budget_test.go`。原始完整绿色记录（含仅时间差异的第二遍）
继续作为历史，不能把后续修正过的全部文件说成由那次套件重新验证。

修正完成后，最终drawio-only包测试PASS0.641s、exit0；最终drawio-only vet
exit0且无输出。`graphview_test.go`没有变化，因此没有重复运行graphview。见
[修正结果](handoffs/drawio-svg-budget-correction-result.md)、
[drawio最终测试](handoffs/evidence/drawio-svg-budget-correction-test.log)与
[drawio最终vet](handoffs/evidence/drawio-svg-budget-correction-vet.log)。
修正后的冻结identity如下：

| 文件 | 字节 | SHA-256 |
|---|---:|---|
| `native/formats/drawio/types.go` | 5,089 | `233025C35D1A6C33832E47D8782A07A367954F1EEF70754FB83A5BE436C7D845` |
| `native/formats/drawio/svg.go` | 15,572 | `47F3E584EEF6E2AD981FAE539EE6E2ECA3E93DDBD16F22B7DEC3D5FA086944CF` |
| `native/formats/drawio/drawio_test.go` | 13,581 | `A1A777227507637928A1920FF9B253F75C6E4A1EFEA697BB087F74CB8D18F189` |
| `native/formats/drawio/svg_budget_test.go` | 10,737 | `79C1F18B12D35C238FA76E5F919BCA65FEE734AA62FC40BEB48FE43797261801` |
| `native/formats/drawio/testdata/svg-prechange-oracle.json` | 6,688 | `14D7C7166E047979ACDBCBE670EE6918CF44892D54680AAFAC7B0B81944A6611` |
| `native/modules/graphview/graphview_test.go` | 9,858 | `3F8B264305F77F1D6EF2D1468742E8F62DB9248631EB7AB675A1EE213CDE18A0` |

`.18`交付用一次现有打包脚本调用完成四次Windows amd64 GUI构建，全部exit0
且stdout/stderr为空；四份产物都是Windows amd64 GUI PE。302个源/构建输入
共3,027,991字节（含62个`native/formats`文件及实际存在的
`native/portable/NOTICE-PREAMBLE.md`）前后相同，inventory SHA-256
`B4681E2EB1E761D2D0D8F811E75D24663727C1050B6C4921B5EE6411976B02B1`；
六个SVG源/测试/oracle和23个已接受证据/门文件保持不变。完整`.17`目录23文件
23,690,310字节及其117文件/47,784,783字节直接证据未改写。validator与
aggregate为12/12、exit0，无重试；[aggregate.json](artifacts/acceptance/drawio-svg-budget-release18-20261110-011852/aggregate.json)
SHA256`D1066FA721005A7F4DC97184BC8E1E802B8FE86013609B5A2F64975D81E14FF0`。

隐藏宿主使用一份字节相同的23文件副本从`F:`启动，run-owned配置位于普通
`C:` temp卷。PID18756、held handle2312和query-path image经精确身份核对；
post owned `WM_CLOSE`后exit0，无超时、强制清理或剩余owned进程，app
stdout/stderr为空，worker操作为0。60字节源文件字节/哈希/时间不变；关闭时
恢复会话416字节，SHA256
`1B9FB3D2E8E5E20F593961F3F7D01838B13841066D9DB4DA8E8DC85C4127DE49`，
只有一个干净UTF-8 Markdown tab，无BOM/draft或终止LF/CR。候选、副本和隔离
配置保留。隐藏启动只验证启动和干净关闭，没有执行打包SVG交互、dirty
recovery、managed tester、production worker、物理IME/UI自动化或资源测量。

## 全词查找与 `.17` 历史交付：2026-11-10

`.17`历史入口
[bin/levnote-native-worker-20261110.17](bin/levnote-native-worker-20261110.17/README.md)，
版本`native-worker.20261110.17`。根代理接受完整23文件、23,690,310字节目录；
宿主14,066,688字节，SHA256
`E4E81A30B0CB47F080379BD75EEEBD003F4621B3A8393D344073D32DA94EBC1C`；
发布清单4,137字节，SHA256
`668C94BCDAC492BE88D96AE45A73157B7B5B7427C001FC2F4452DF0BB6E3BD08`。
源码、包与原始核验见[源码验收](handoffs/find-whole-word-root-acceptance.md)、
[包验收](handoffs/find-whole-word-release17-root-acceptance.md)、
[交付记录](handoffs/find-whole-word-release17-delivery.md)和
[validator.json](artifacts/acceptance/find-whole-word-release17-20261110-004642/validator.json)。

`App.WholeWord`是运行期选项，与大小写、正则一起由`findOptions()`映射到
`core/search`，供文档上一处/下一处、Replace Current/All和工作区搜索共用。
边界谓词为Unicode字母、数字、记号与下划线：`cat`命中独立`cat`，不命中
`category`、`_cat`或`cat_`；`猫`不切分`猫咪`，组合记号不拆成独立词。该规则
不是中文分词。全词开关不写入设置或会话，不新增依赖；核心搜索算法、活动文档
缓存结构与会话schema不变。选项切换会改变缓存身份，使过滤前偏移失效。

实际`ui.Tester`复选框交互覆盖切换、上一处/下一处与回绕，以及760×480布局、
Tab、Escape；替换、Unicode/大小写/正则组合、工作区和只读/失败Flush生命周期
则由测试分配运行期选项并调用应用方法。Replace All仍为一个撤销组，失败Flush
保留文本、Buffer版本、选区、历史和字段焦点。两张760×480浅色/深色图是
MyGo Tester渲染结果，不是物理Windows输入、IME候选窗或打包宿主Find交互证据：
[浅色](handoffs/find-whole-word-evidence/screenshots/findbar-light-760x480.png)、
[深色](handoffs/find-whole-word-evidence/screenshots/findbar-dark-760x480.png)。

生产/UI冻结后的首次受影响套件为`app`15.298s、`core/search`0.742s，exit0；
随后scoped vet exit0且无输出。两轮测试断言修正之后，只重跑已修正的聚焦测试
1.581s与Replace Current 1.740s；生产/UI未改，广泛绿色门没有重跑。文档更新
不触发测试、vet或编译。最终冻结identity为：

| 文件 | 字节 | SHA-256 |
|---|---:|---|
| `app/app.go` | 33,365 | `58B6A783968C8860BBE6E8EF87E74CE9C6BEA3560BD4C19A569FC44465557931` |
| `app/state.go` | 11,176 | `023C76A143CB572D0A2B45D783A20E105996FF570615357BB1C3F31032E79984` |
| `app/find_whole_word_test.go` | 7,548 | `BEBFEBC560AA78D8A47B1390A2CA4E29E956C8328CB4AFEE0B080BF81328E744` |
| `app/search_navigation_test.go` | 21,407 | `FA8EF3C1DAF50B67AA03F7ADE032F465EB0535202FAB4328EB7C28DFC4E72A40` |
| `app/findbar_layout_test.go` | 7,275 | `662C167BFFA6CC344EF2E2A73E3566824E03B0DA3BA3D19FA463279FFCF12180` |

`.17`交付用一次现有打包脚本调用完成四次Windows amd64 GUI构建，全部exit0
且stdout/stderr为空。300个源/构建输入共3,005,261字节（含60个
`native/formats`文件）前后相同；五个冻结全词源/测试、已接受截图/日志/核对，
以及完整`.16`目录23文件23,690,310字节和其104文件、47,774,882字节直接证据
保持不变。validator与aggregate为12/12、exit0，无重试。证据根为
[find-whole-word-release17-20261110-004642](artifacts/acceptance/find-whole-word-release17-20261110-004642/)。

隐藏宿主使用一份字节相同的23文件副本从`F:`启动，run-owned配置位于普通
`C:` temp卷。owned `WM_CLOSE`后exit0，无超时、强制清理或剩余owned进程，
app stdout/stderr为空，worker操作为0；60字节源文件字节/哈希/时间不变。
关闭时恢复会话414字节，SHA256
`7F3066DE1F3A3F9A983DB9806F73EA8D16961AB0EE89484B57BF305587F56B7C`，
只有一个干净UTF-8 Markdown tab，无BOM/draft或终止LF/CR。隐藏启动只验证
启动和干净关闭，没有验证打包Find交互、dirty recovery、managed tester、
production worker、物理IME/UI自动化、进程内存、CPU或资源改善。

## 会话恢复 JSON 输出与 `.16` 交付：2026-11-10

`.16`历史入口
[bin/levnote-native-worker-20261110.16](bin/levnote-native-worker-20261110.16/README.md)，
版本`native-worker.20261110.16`。根代理接受完整23文件、23,690,310字节目录；
宿主14,066,688字节，SHA256
`76A60252CFCC3394386B4DAF87AF8B845086DE35ED5DCCACC6CC79179E60B03D`；
发布清单4,137字节，SHA256
`53994DB6861F4B9918D33F9FF90330744862730C838417E0C10501ADA050CD90`；
包清单身份
`F9937B33552496EAEDA1FB2A7F68B7F27FB769915A6E4FEFC546E566D10E9CF5`。
源码验收、交付记录与12/12原始核验分别见
[源码验收](handoffs/session-output-root-acceptance.md)、
[`.16`交付](handoffs/session-output-release16-delivery.md)和
[validator.json](artifacts/acceptance/session-output-release16-20261110-001354/validator.json)。

会话`Save`在`preflightSnapshot`及4 MiB单草稿/100标签/16 MiB编码后预算完成
后，将已校验Snapshot和精确长度交给`WritePrivateStateTo`，复用既有私用状态
临时文件权限、同步、关闭、原子发布和清理。标准`json.Encoder`经一字节待处理
适配器写入该接缝：运行期核对预检长度、拒绝短写/溢出，除待处理末字节外交付
writer，并要求`Finish`看到的末字节是Encoder强制LF且总长度等于预检长度加一。
运行期不显式比较首字节，也不做oracle哈希比较；测试另行证明完整输出与
`json.Marshal`oracle逐字节相同，字段、顺序、转义、`omitempty`/nil、草稿
预算与恢复格式保持不变，完整JSON末尾没有LF。临时写入、发布前校验和发布
操作错误均发生在提交点前，因此旧恢复文件字节与时间保留、临时文件清理。
原子发布成功后`syncDirectory`仍可能失败；Windows当前为no-op，Unix可能返回
错误。此时保存报错，但新文件已发布且不会回滚。现有重试/丢弃行为不变。
适配器对首个错误保持粘滞是合同加固，不宣称修复此前生产调用方缺陷。
Encoder仍内部缓冲完整JSON，脏草稿字符串捕获仍在；这不是token流式或固定
内存实现。详细合同见[架构](ARCHITECTURE.md#文件与恢复安全)，实现与审查见
[实现结果](handoffs/session-output-implementation-result.md)和
[独立核对](handoffs/session-output-independent-reconciliation.md)。

最终受影响套件实际通过`core/document`0.894s、`core/session`1.646s、
`app`17.571s；随后scoped vet exit0且无输出。原始最终证据为
[affected suite](handoffs/session-output-evidence/review-final-affected-suite-1.log)
和[vet](handoffs/session-output-evidence/review-final-affected-vet-1.log)。
这些结果之后没有为文档更新重跑测试或编译。

修正后的输出边界基准固定10次，两条路径使用同一预置输入，先以`json.Marshal`
oracle核对新输出字节、长度和无末尾LF，再分别向`io.Discard`写入，设置与
oracle在计时循环外。

| 夹具 | 编码字节 | `json.Marshal` | Encoder/适配器 |
|---|---:|---:|---:|
| 一条约1 MiB脏草稿 | 1,048,701 | 1,164,120 B/op、5 allocs/op | 106,213 B/op、4 allocs/op |
| 八标签合计约1 MiB | 1,049,114 | 1,469,992 B/op、5 allocs/op | 412,825 B/op、4 allocs/op |

标准库encoder池状态会影响B/op样本，因此这些只是输出边界分配观察，不是
保证、整次`Document.Save`延迟、磁盘/CPU/GC、RSS/工作集/private usage、
进程内存或物理输入结论。原始样本见
[benchmark](handoffs/session-output-evidence/review-session-output-benchmark-1.log)。

`.16`交付用一次现有打包脚本调用完成四次Windows amd64 GUI构建，全部exit0
且stdout/stderr为空。299个源/构建输入共2,996,205字节（含60个
`native/formats`文件及实际存在的`native/portable/NOTICE-PREAMBLE.md`）
前后相同；六个冻结session源/测试和已接受证据保持不变，完整`.15`目录
23文件23,680,582字节及其109文件、47,758,090字节直接证据未改写。最终
validator与aggregate为12/12、exit0，无重试。证据根为
[session-output-release16-20261110-001354](artifacts/acceptance/session-output-release16-20261110-001354/)。

隐藏宿主使用一份字节相同的23文件副本从`F:`启动，run-owned配置位于普通
`C:` temp卷。held process/owned window经精确身份核对，post owned`WM_CLOSE`
后exit0，无超时、强制清理或剩余owned进程，app stdout/stderr为空，worker
请求为0。60字节源文件字节/哈希/时间不变；关闭时恢复会话413字节，SHA256
`B66C699AD0C5D0FD11E34E86FCD0FE3037AC9CB9D183C7A07E3F0138A9A8D0C8`，
只有一个干净UTF-8 Markdown tab、无BOM/draft，最后字节为`}`且没有终止
LF/CR。隐藏启动只验证干净会话输出，没有执行dirty recovery、managed tester
或production worker请求；两次受控引擎请求仍是`.11`历史证据，`.13`视觉
证据也保持历史。本轮没有生成/复核物理UI/IME、可见自动化或资源测量证据。

## UTF-8 流式保存与 `.15` 交付：2026-11-09

`.15`历史入口
[bin/levnote-native-worker-20261109.15](bin/levnote-native-worker-20261109.15/README.md)，
版本`native-worker.20261109.15`。根代理接受完整23文件、23,680,582字节目录；
宿主14,056,960字节，SHA256
`3C5B07646C2E50A82304B88A02B59C2EB170F067599B7FFAB044DCCEBF3DD804`；
发布清单4,137字节，SHA256
`3A451E08D9394F829CEE0E85D7C7F14F1FFB735C47049DB6CEF502EE22E35D32`。
源码验收、交付记录与12/12原始核验分别见
[源码验收](handoffs/utf8-save-root-acceptance.md)、
[`.15`交付](handoffs/utf8-save-release15-delivery.md)和
[validator.json](artifacts/acceptance/utf8-save-release15-20261109-234608/validator.json)。

脏UTF-8/UTF-8 BOM `Save`/`SaveAs`从现有`Buffer.Snapshot()`经32 KiB缓冲写入
临时文件，并用同一写入流计算SHA-256；可选BOM、LF/CRLF与Unicode精确字节、
多分片快照、重新打开及临时文件清理均有断言。发布后读回摘要必须等于预期
摘要，才推进磁盘token、已保存快照、编码/BOM元数据、基准版本和脏缓存；
摘要不匹配返回`ErrConflict`并保持活动编辑缓冲为脏。共享原子写入生命周期
覆盖短写、同步、关闭、发布前检查、独占/替换发布与清理。无改动保存仍保持
字节和时间；现有冲突、只读/组合输入拒绝及`SaveAs`不覆盖目标测试继续通过。
`SaveAs`保留`.14`的Path/IsNew/ReadOnly发布后时机，因此发布后校验失败不能
承诺回滚。UTF-16、GBK和GB18030仍走旧整缓冲路径。详细实现与审查分别见
[实现结果](handoffs/utf8-save-implementation-result.md)和
[独立核对](handoffs/utf8-save-independent-reconciliation.md)。

最终受影响套件实际通过：`core/document`0.818s、`core/session`1.040s、
`app`16.036s；随后scoped vet exit0且无输出。原始最终证据为
[focused tests](handoffs/utf8-save-evidence/focused-utf8-save-tests-6.log)、
[benchmark](handoffs/utf8-save-evidence/benchmark-utf8-output-10x-final.log)、
[affected suite](handoffs/utf8-save-evidence/final-affected-suite-2.log)和
[vet](handoffs/utf8-save-evidence/final-vet.log)。这些结果之后没有为文档
更新重跑测试或编译。

输出生成基准固定10次，两条路径消费同一个预置快照，都向`io.Discard`写入
相同BOM前缀UTF-8字节并以完整输出SHA-256校验；夹具创建、快照捕获和oracle
设置在计时循环外。

| 夹具 | 整缓冲 | 流式 |
|---|---:|---:|
| 1 MiB | 2,105,398 B/op、4 allocs/op | 33,131 B/op、8 allocs/op |
| 8 MiB | 16,785,478 B/op、4 allocs/op | 33,139 B/op、8 allocs/op |

流式路径的分配次数由4增至8，收益限于编码输出的累计分配字节。该基准不测
快照捕获、基准保留、整次`Document.Save`延迟、进程峰值/RSS/工作集/private
usage、UI/CPU/GC或物理输入。

`.15`交付用一次现有打包脚本调用完成四次Windows amd64 GUI构建，全部exit0
且stdout/stderr为空。295个源/构建输入共2,975,431字节（含60个
`native/formats`文件及实际存在的`native/portable/NOTICE-PREAMBLE.md`）
前后相同；五个冻结UTF-8源/测试和已接受证据保持不变，完整`.14`目录23文件
23,673,926字节及其104文件、47,738,730字节交付证据未改写。最终validator与
aggregate为12/12、exit0，无重试。证据根为
[utf8-save-release15-20261109-234608](artifacts/acceptance/utf8-save-release15-20261109-234608/)。

隐藏宿主使用一份字节相同的23文件副本从`F:`启动，run-owned配置位于普通
`C:` temp卷。held process/owned window经精确身份核对，post owned`WM_CLOSE`
后exit0，无超时、强制清理或剩余owned进程，app stdout/stderr为空，worker
请求为0。60字节源文件字节/哈希/时间不变；恢复会话408字节，SHA256
`1028497A55298F71A1D9C1FF469189D9EEA4A92D4E289FA67728FF1E5593DC62`，
只有一个UTF-8 Markdown tab且无draft。隐藏启动没有执行dirty Save、managed
tester或production worker请求；两次受控引擎请求仍是`.11`历史证据。本轮
没有生成/复核物理UI/IME、可见自动化或资源测量证据。

## 文档内查找键盘与 `.14` 历史交付：2026-11-09

当时入口
[bin/levnote-native-worker-20261109.14](bin/levnote-native-worker-20261109.14/README.md)，
版本`native-worker.20261109.14`。根代理接受完整23文件、23,673,926字节目录；
宿主14,050,304字节，SHA256
`D037EF7B3DC700E84AA9484AE9A0A7A706662DB8A834D8D3CC4C4B53A3C4FC9A`；
发布清单4,137字节，SHA256
`A876F6E298DCDFA176178E5B2749383AF0CD5053984355751C0A0F10F9BF5379`。
包验收、交付记录与12/12原始核验分别见
[根代理验收](handoffs/find-keyboard-release14-root-acceptance.md)、
[`.14`交付](handoffs/find-keyboard-release14-delivery.md)和
[validator.json](artifacts/acceptance/find-keyboard-release14-delivery-20261109-215637/validator.json)。

实际行为覆盖首次/重复`Ctrl+F`与工具栏查询框聚焦、查询Enter/Shift+Enter、
替换Enter/Shift+Enter、源码/分屏Escape返回、预览不投递不可见焦点，以及查询、
替换、大小写和正则值保留。活动组合输入时Enter/Escape动作与`Ctrl+G`不生效；
字段失焦或`ShowPlugins`隐藏Find后，公开Focus/Compose状态核对会清理瞬时
组合与焦点标记。焦点/动作callback核对generation与标签/Document/Editor/Buffer；
替换在Flush后重验身份并使用新的匹配结果，失败Flush保留文本、历史、选区与
字段焦点。具体源码/API合同见[架构](ARCHITECTURE.md#查找键盘焦点与组合输入)。

最终生命周期聚焦回归7/7，app1.634s、exit0；最终受影响套件为
`app`14.326s、`ui/editor`2.083s，均exit0；随后scoped vet exit0且无输出。
没有在这些结果之后重跑测试或编译managed tester。原始最终日志为
[suite](handoffs/find-keyboard-evidence/final-app-ui-editor-test-3.log)
SHA256`49F7F745B49E4D8551C48C74FB8F696EFBA6D1306BE99B59F8E1D29EBDE86B1B`
与[vet](handoffs/find-keyboard-evidence/final-app-ui-editor-vet-3.log)
SHA256`55136E372CC06096357CBA87F588CCC89781D15DC38B9BC7BEEF9078BF7A4185`。
接受时独立核对的八个源码/测试身份如下；完整证据与历史red见
[实现结果](handoffs/find-keyboard-result.md)和
[独立核对](handoffs/find-keyboard-independent-reconciliation.md)。

| 文件 | SHA256 |
|---|---|
| `native/app/app.go` | `F57743A4CAD59C95B3FB13D8B8C982C99CBFE43B979ED7EB447ACD51B53C6AC3` |
| `native/app/search_navigation.go` | `57D56DA4B266E9381D8F2268A26D8CFC33ABA4B968937D3AB753F32ED4763E77` |
| `native/app/state.go` | `63BBC1372A26D862D8B69D755E50FC7880C6474C36121F4A0A637DDA852E8D38` |
| `native/app/source_navigation.go` | `49FA79D5D3FC7B38B46363874780E421E3DABE363F6C28F1C3FAE78B12806113` |
| `native/app/view.go` | `CA2D62B0F873A86FADB92DBA525BE09E25DA04F44A7969D6D9D44DA532EFD640` |
| `native/app/find_keyboard_test.go` | `1C9D0408DDE6899E134351763E9373D61E7A0537E8B150A1EB2CB2AEC64FAD81` |
| `native/app/find_keyboard_lifecycle_test.go` | `04C55F900DA46173EA1A757B808B1106C78F0196F295948E6CFD6B94A284D680` |
| `native/ui/editor/editor.go` | `6438B438F0409BEEB26A9427C5CC8EBF971132242933B1D927BEBFEA4371AC81` |

`.14`交付使用一次现有打包脚本调用完成四次wrapper GUI build，全部exit0且
stdout/stderr为空。293个源码/构建输入共2,962,619字节（含60个
`native/formats`文件及实际存在的`native/portable/NOTICE-PREAMBLE.md`）
前后相同；八个冻结键盘源/测试身份和两张`.13`历史布局图均保持。完整`.13`目录23文件
23,655,494字节及其102文件、47,615,997字节交付证据未改写。最终validator
12/12、aggregate exit0，无重试。

隐藏宿主使用一份字节相同的23文件副本从`F:`启动，run-owned
`APPDATA`/`LOCALAPPDATA`位于普通`C:` temp卷。PID17036、held handle2288和
owned window经精确身份核对；post owned`WM_CLOSE`后exit0，无超时、强制清理
或剩余owned进程，app stdout/stderr为空，worker请求为0。60字节源文件
字节/哈希/时间不变；恢复会话421字节，SHA256
`1957AE8D2AA6CA68C014FBE16E82EF8AC9A05D479F5B0640919AAE1EBCA723E4`，
只有一个UTF-8 Markdown tab且无draft。

本轮`.14`没有执行managed tester或production worker请求；两次受控引擎请求
仍是继承自`.11`的历史证据。合成App/Tester键盘证据不证明物理Windows IME、
候选窗、可见UI自动化或Escape行为；`.13`两张布局图仍是历史图像。没有GPU、
CPU、RSS、working set、内存或性能结论。

## 文档内查找、窄窗查找栏与 `.13` 历史交付：2026-11-09

当时入口
[bin/levnote-native-worker-20261109.13](bin/levnote-native-worker-20261109.13/README.md)，
版本`native-worker.20261109.13`。根代理接受完整23文件、23,655,494字节目录；
宿主14,031,872字节，SHA256
`4B86002034A493789A1747E343542FD5A94BCD9DCEB5060662A5DD55DCF4150D`；
发布清单4,137字节，SHA256
`0F1825348BD3EEFDB8264079405DBA23019692D22EA7962098B3A4FC4470D5FB`。
交付证据在
[`.13`交付记录](handoffs/search-navigation-release13-delivery.md)，
包验收与11/11原始核验见
[根代理验收](handoffs/search-navigation-release13-root-acceptance.md)和
[validator.json](artifacts/acceptance/search-navigation-release13-delivery-20261109-210329/validator.json)。

查找实现结果、源码验收和独立审查分别为
[实现结果](handoffs/search-navigation-result.md)、
[源码验收](handoffs/search-navigation-root-acceptance.md)、
[独立审查](handoffs/search-navigation-independent-review.md)；最小窗口布局为
[布局结果](handoffs/findbar-layout-result.md)和
[布局验收](handoffs/findbar-layout-root-acceptance.md)。

最终查找聚焦测试PASS2.035s；随后唯一一次受影响套件为`app`16.286s、
`core/search`0.716s，均exit0；scoped vet exit0且无输出。最终布局聚焦
PASS2.213s，唯一一次最终`app`套件16.235s，scoped vet exit0且无输出。
没有为布局再跑`core/search`、managed tester或worker请求。原始adjacency
red、两次只测试断言/夹具修正、最终日志与冻结hashes分别保留在结果和证据目录，
没有把夹具失败描述成另一个产品失败。

实际行为覆盖相邻结果、精确前进/反向选区、最近结果、零宽结果、单结果回绕、
UTF-16与边界；每次查找都先Flush，冷扫描与发布前核对最终身份/版本。
仅当前活动文档缓存复制出的UTF-16 offset pairs，不保留source、Snapshot、
Match或per-tab表。提交标签加入/切换/移除、成功重载、插件/编码重配、关闭
查找、关闭应用或实际替换会清理缓存；失败Flush、切换或关闭保留活动文档、
草稿和仍有效缓存。查询/选项/版本变化可懒到下一帧/下一次查找才清理，但
不会跨身份或陈旧版本生效。Replace/ReplaceAll继续使用新搜索。

20次函数benchmark中warm lookup为0B/op、0allocs/op；cold path为
1,063,316B/op、4,137allocs/op。cold path仍扫描整篇文本，这组数据不表示
整篇搜索或整应用提速，也不提供进程RSS/CPU结论。during-scan换标签测试用
确定性同步注入证明旧结果不发布，不是通用并发安全测试。

最小760×480的原始产品red是“正则”控件右边界到x=809；两个`.Wrap()`后
最右控件到x=748，最后一行结束于y=238.041，状态栏顶边为451，编辑器区为
532×203.959。浅色与深色PNG证明该窗口下的渲染几何；实际1180×780到
760×480 resize、查询/替换输入、选项、前后导航、replace/close动作和
Tab遍历由MyGo Tester事件检查证明，不能归因于截图。两者都不证明物理OS
IME、可见交互或资源表现：

- [浅色760×480](handoffs/findbar-layout-evidence/findbar-light-760x480.png)，
  SHA256`1860EB9E8E96F8F1EA51052CC907AA1D3D188B73122C565969D9045C8EA599CB`。
- [深色760×480](handoffs/findbar-layout-evidence/findbar-dark-760x480.png)，
  SHA256`BF139E982E55183C515FA0965033643B748C611598ACDF5B28A89F3CA6DAA2F3`。

`.13`交付使用一次现有打包脚本调用完成四次wrapper GUI build，全部exit0且
stdout/stderr为空。291个源码/构建输入共2,912,652字节（含60个
`native/formats`文件）前后相同；八个冻结查找/布局文件与两图继续匹配。
完整`.12`目录23文件23,645,766字节、宿主、清单和原始证据均未改写。

隐藏宿主使用一份字节相同的23文件副本从`F:`启动，run-owned
`APPDATA`/`LOCALAPPDATA`位于普通`C:` temp卷。PID38700、held handle2196和
owned window经精确身份核对；post owned`WM_CLOSE`后exit0，无超时、强制清理
或剩余owned进程，app stdout/stderr为空。60字节源文件字节/哈希/时间不变；
恢复会话425字节，SHA256
`224C70518D42B283CE40E631019BE10C4CB21C59EF39E0EDD47EAA03C05EA2B3`，
只有一个UTF-8 Markdown tab且无draft。最终validator
11/11、aggregate exit0，无重跑。

本轮`.13`没有执行managed tester或production worker请求；两次受控引擎请求
仍是继承自`.11`的历史证据。`.13`只独立验证三个包内引擎文件的版本/大小/摘要
与manifest。没有物理OS IME、`Ctrl+F`焦点/`Enter`快捷键、GPU、CPU、RSS、
working set或其他资源结论；键盘行为是后续独立任务，不属于本交付。

## 源码行列导航与 `.12` 历史交付：2026-11-09

当时入口
[bin/levnote-native-worker-20261109.12](bin/levnote-native-worker-20261109.12/README.md)，
版本`native-worker.20261109.12`。根代理接受完整23文件、23,645,766字节目录；
宿主14,022,144字节，SHA256
`43FC7E86004DC3602F6AA367B6BC55007A1AE28614484E5F7DD3BADF0A782768`；
发布清单4,137字节，SHA256
`9AC85B63DE204FE9B3FE4CEEE865838FAA04B7B810213374727987EF50F0C445`。
源码、测试、图片与包证据分别见
[源码验收](handoffs/source-navigation-root-acceptance.md)、
[实现结果](handoffs/source-navigation-result.md)、
[独立审查](handoffs/source-navigation-independent-review.md)、
[包验收](handoffs/source-navigation-release12-root-acceptance.md)、
[交付记录](handoffs/source-navigation-release12-delivery.md)和
[11/11核验](artifacts/acceptance/source-navigation-release12-delivery-20261109-201756/validator.json)。

源码导航最终顺序为聚焦测试、刷新截图、一次受影响合并测试、scoped vet；
之后实现源码/测试/证据未再变化。独立审查只读源码和两张最终图，没有重复
执行。最终八文件哈希与两张图片身份均被`.12`交付清单再次核对。

| 实际检查 | 结果与范围 |
| --- | --- |
| 聚焦回归 | `test ./app -run 'SourceNavigation|ParseSourceLine' -count=1` PASS2.294s；覆盖实际Ctrl+G/状态入口、严格语法、Flush一次、最终缓冲坐标、活动组合、按钮焦点Escape、陈旧generation、只读/字节视图及标签清理 |
| 刷新截图 | `TestSourceNavigationNarrowErrorCapture` PASS1.546s；保留前两次只读错误断言失败，最终两图由根代理和独立审查直接查看 |
| 受影响合并测试 | app23.694s、ui/editor5.533s、markdownview7.713s、graphview19.048s均PASS；随后scoped vet exit0/无输出 |
| 打包与构建 | 一次现有打包脚本调用完成四次wrapper Windows amd64 GUI build，全部exit0且stdout/stderr为空；最终validator 11/11 |
| 受控宿主启动 | 一份字节相同宿主副本从`F:`、`C:`隔离配置隐藏启动；owned`WM_CLOSE`后exit0，无超时/强制清理；60字节源文件、425字节会话、复制包与app stdout/stderr全部保持预期 |
| 完整性 | 288源/构建输入共2,880,940字节（含60个`native/formats`文件）前后相同；`.11`23文件23,622,214字节、宿主与清单身份及原始证据全部不变 |

最终明暗tester图如下；它们证明760×480面板内输入、错误、帮助、按钮和
状态行可见，不证明物理UI自动输入、系统IME候选窗或资源表现：

- [浅色760×480](handoffs/source-navigation-evidence/source-navigation-error-760x480-light.png)，
  SHA256`2932E7800A6C9062636DDDA1180F4A876ACBCDA1B7769F7F413427B614F0229A`。
- [深色760×480](handoffs/source-navigation-evidence/source-navigation-error-760x480-dark.png)，
  SHA256`17E75F906CF7B4C54F6D1708E46EF9EBEE714B576B2FC84799F542D361960D7B`。

本轮`.12`没有编译或运行managed tester，也没有发出production worker请求；
两次installed`.10`→bundled`.11`受控Markdown请求仍只是`.11`历史证据，不能
描述成`.12`运行。聚合首次exit1仅因证据文件名写成旧
`worker-markdown-preview`，修正为`worker-mindmap-preview`后只重跑聚合，
原始失败保留，未重做构建或启动。没有物理Ctrl+G/OS IME、GPU、CPU、RSS、
working set或其它资源结论。

## 本地引用保留修正与 `.11` 历史交付：2026-11-09

当时入口
[bin/levnote-native-worker-20261109.11](bin/levnote-native-worker-20261109.11/README.md)，
版本`native-worker.20261109.11`。根代理接受完整23文件、23,622,214字节目录；
宿主13,998,592字节，SHA256
`F1D1FC76AC1A5649709B6AD8FDB44A1928174280F766625F7226856493E299B1`；
发布清单4,137字节，SHA256
`A4E37C4E65B38C7B46ACB1FBC80455B604EE4207E7D3072580BD37DE0103E3E6`。
完整证据在
[accepted delivery directory](artifacts/acceptance/retention-release11-delivery-20261109-193512)，
13/13聚合通过；详见[交付交接](handoffs/retention-release11-delivery.md)
和[根代理验收](handoffs/retention-release11-root-acceptance.md)。

两项修正都是确定性指针所有权，不是进程内存测量：

- `Buffer.replace`合并piece后清理超出新逻辑长度的死槽位，避免仍活着的
  backing array保留失效chunk指针；快照、撤销/重做和逻辑内容不变。
- `removeTab`移动剩余指针、清空被移除的尾部指针后再缩短切片，避免关闭
  标签后的backing array继续拥有已移除`*Tab`；关闭/恢复语义不变。

| 实际检查 | 结果与范围 |
| --- | --- |
| 打包与构建 | 一次包装器调用、四次GUI build exit0；完整副本与候选逐字节一致；286个输入从baseline到聚合不变 |
| Managed请求 | 一次tester运行exit0，恰好两次生产请求：导入并启用`.10` Markdown（包digest`977ee81a1f7591a4dba204527c183293c05354f3c64ac14e8342df300f3c130c`）→Reset bundled `.11`；两次均输出104字节HTML，SHA256`681487B9A59F3D162BC719ED242B2242AB4A5B16CCD61C5A945D15EEAEDC50A0`，匹配独立literal oracle。输入是内存中20字节快照，两次Runner.Close均无错误 |
| 跨卷启动 | `.11`宿主/目录在`F:`，run-owned `APPDATA`/`LOCALAPPDATA`在默认`C:`TEMP；隐藏窗口收到owned WM_CLOSE后exit0，60字节源文件字节与时间戳保持不变，session 417字节/SHA256`20050A68832F559159910BE058B01743096658783BB784AEBF9A61E33BDA7F74`保留，无强制清理 |
| UI图片 | 4张760×480测试器图保持历史`.10`来源；`.11`没有重拍，不能当作本版物理UI证据 |
| 不变范围 | `.10`23文件23,622,214字节及全部失败证据完整保留；`.9`仍为失败历史 |

本版没有RSS、working set、CPU、GPU或物理UI/IME测量，也不宣称相对旧版的
资源效率改善。历史`.10`交付与完整证据继续保留在下节。

## 历史`.10`交付：2026-11-09

历史入口
[bin/levnote-native-worker-20261109.10](bin/levnote-native-worker-20261109.10/README.md)，
版本`native-worker.20261109.10`。根代理接受完整23文件、23,622,214字节目录；
宿主13,998,592字节，SHA256
`4EB096FAEA7CA79E71F0C9FF30E659478C0E2AA135803DDBF7C0AEE47A0AEF6C`；
发布清单4,137字节，SHA256
`38D79797ABA4EDFB0DD12E3ED78476D8CC956D1A3069921E447690E842A0670D`。
完整证据在
[accepted delivery directory](artifacts/acceptance/fixed-engine-package-delivery-20261109-20261109-191019)，
14/14聚合通过；详见[交付交接](handoffs/fixed-engine-package-delivery.md)
和[根代理验收](handoffs/fixed-engine-package-delivery-root-acceptance.md)。

`.9` managed尝试在生产worker请求前失败：`lockDir(create=true)`试图对
已有卷根`C:\`调用`Mkdir`，得到`Access is denied`。修正采用open-first
路径遍历：先打开并验证既有祖先（含卷根），仅对真实缺失的子目录创建并
重新打开验证；不再把拒绝访问或无效既有组件解释为缺失目录。精确修复
为`core/engineruntime/package_files.go`，新增Windows回归见
`package_guard_windows_test.go`，证据与原始失败保留在
[volume-root correction](artifacts/acceptance/engine-package-volume-root-fix-20261109-190635)。

| 实际检查 | 结果与范围 |
| --- | --- |
| 修正回归 | 旧行为红：卷根下创建与`NewPackageManager(host F:, state C:)`均exit1；最终三项聚焦测试exit0。affected `TestPackage*`/`TestManaged*` PASS6.787s，scoped vet0；symlink-file/symlink-dir因账户无权限未运行，不宣称验证 |
| 打包与构建 | 一次包装器调用、四次GUI build exit0；完整副本与候选逐字节一致；284个输入从baseline到聚合不变 |
| Managed请求 | 一次tester运行exit0，恰好两次生产请求：导入并启用`.8` Markdown（包digest`fb291276e904a570219cf8d8ce28882b44811f4f4af89bf1cc423cabf929f155`）→Reset bundled `.10`；两次均输出104字节HTML，SHA256`681487B9A59F3D162BC719ED242B2242AB4A5B16CCD61C5A945D15EEAEDC50A0`，匹配独立literal oracle。输入是内存中20字节快照，两次Runner.Close均无错误 |
| 跨卷启动 | `.10`宿主/目录在`F:`，run-owned `APPDATA`/`LOCALAPPDATA`在默认`C:`TEMP；隐藏窗口收到owned WM_CLOSE后exit0，60字节源文件字节与时间戳保持不变，session 427字节/SHA256`6DF53A3D…`保留，无强制清理 |
| UI图片 | 复用根代理已接受的4张760×480安装面板/隐藏后台进度浅色/深色测试器图；app/rendering源码与117文件捕获清单除明确runtime修正外不变 |
| 不变范围 | `.8`23文件23,457,235字节、`.9`23文件23,621,697字节及全部失败证据完整保留；`.9`不是可接受发布 |

真实窗口UI试点的能力门失败：Computer Use helper三次有界探测（初始、
允许的2秒重试、JavaScript会话重置/重导入后再探测）均无法连接native
pipe，执行按要求在启动应用前停止；未复制包、未生成夹具、无真实窗口
观察/输入或资源测量。这是helper可用性限制，不是产品UI失败，精确记录见
[UI feasibility result](handoffs/release10-ui-feasibility-result.md)。

两次证据-only harness问题保留：managed wrapper把`Failures:null`按一个元素
计数，实际tester已PASS，状态仅重新处理现有记录；aggregate首次使用只读
PowerShell`$Host`变量失败，修正后只重读证据，不重跑请求/编译。本轮没有
物理UI/IME、GPU、CPU或资源效率结论；历史资源数字仍只属于各自原始构建。

## Schema3固定原生格式选择 · 历史`.8`交付：2026-11-09

当时入口[bin/levnote-native-worker-20261109.8](bin/levnote-native-worker-20261109.8/README.md)。
根代理已读取[12/12聚合](artifacts/acceptance/format-routing-delivery-20261109-114942/aggregate02.json)、
[核验与代码](artifacts/acceptance/format-routing-delivery-20261109-114942/validator02.json)、
实际build/smoke/session记录及保留的元数据失败，接受23文件完整目录。
总大小为**23,457,235字节**，取自[独立总字节更正](artifacts/acceptance/format-routing-delivery-20261109-114942/total-byte-correction.json)，
不能使用聚合中的`TotalBytes:null`。

| 产物 | 字节 | SHA256 |
| --- | ---: | --- |
| 宿主 | 13,836,288 | `60D5C11A8EECF7626DD692098B92DD0CF4810E7F8F289515849982AE23954FBA` |
| Markdown HTML worker | 3,300,352 | `7487710E2882099F522C4F8A14E61C03A3F1086FA5C70E5A073DAA83C1D85F4C` |
| Mindmap SVG worker | 3,122,688 | `8E62E8B4EF23A725FFB2BCA3E29286FC7B703C965FFECB8F44C8DAF34A758C47` |
| Mindmap场景worker | 3,136,512 | `02F9DA3367C45B774ECB504D8B573E1093305A6274C15107A12D5E748196803C` |

聚合SHA256 `3869CF65CD66CAF6107BF42865CA56EBFCA76E63DB741DCC0CDFB89904F450C9`；
核验SHA256 `7FC86AC0AD5EEDFB3D20291D5DF6EF550ED845F752C60EF1543B022F969D2947`；
发布清单SHA256 `9957E49CA27A42F88182AF14758C47C8D09B6654F9F57C8F68B2288A2D45DC41`。

| 实际检查 | 结果与范围 |
| --- | --- |
| core/plugins与modules | PASS2.463s/0.905s，scoped vet0，14输入不变；strict schema3/legacy、真实安装/持久化/回滚/副本、固定适配器能力；[核心验收](handoffs/format-routing-core-root-acceptance.md) |
| app | 完整app PASS9.819s，app/plugins vet0，54输入不变；八项workflow覆盖真实路由、BOM/只读SaveAs、草稿/模态/身份、同ID迟到结果与启用后加载失败；[应用交接](handoffs/format-routing-app-result.md) |
| 原生测试器图片 | capture03 PASS1.622s，760×480插件浅色/深色和自定义`.taskmd`文档三图由根代理审阅；是headless测试器，不是物理输入/系统IME验收 |
| 包 | 一次脚本wall6.015914s，四次build exit0/空streams；实际四个AMD64 GUIPE、三个engine manifest、22发布项、13原样资产、示例JSON和本地链接 |
| 隐藏启动 | 一次完整副本，smoke wall4.5523857s/exit0；精确持有PID26236/句柄确认程序路径，窗口隐藏，正常WM_CLOSE退出，无超时或强制结束；sentinel存活后通过自己的event退出 |
| 会话与完整性 | 隔离APPDATA安装两份真实样例；关闭后正好两份可写`.taskmd`/`.thoughts`文档，对应`task-notes`/`thoughts`，UTF-8、BOM分别true/false、无draft；源码/配置4项、候选和副本23文件、44输入及旧`.7`23文件保持 |

实际[smoke输出](artifacts/acceptance/format-routing-delivery-20261109-114942/smoke.stdout.log)
及[关闭后会话](artifacts/acceptance/format-routing-delivery-20261109-114942/smoke/session-after-close.json)
支持上述有限结论；不依赖隐藏窗口的空闲自动保存，不验证可见布局，也
没有请求worker操作。既有`.7`真实worker/codec/runtime证据沿用，未重放。

原应用attempt01流程断言通过但截图失败，旧wrapper未传LASTEXITCODE使其
result误报通过；原始失败stdout和单独说明保留。capture02修正后仍有深色
clip夹具失败，capture03只修实际wheel点/完整可见边界后通过。首次交付
聚合的12项检查通过，但末尾证据路径拼接失败exit1；attempt02只改证据
表达式，后续空总字节字段由独立Int64求和记录更正，包/启动均未重跑。
详见[交付及失败范围](handoffs/format-routing-delivery.md)和
[根代理验收](handoffs/format-routing-delivery-root-acceptance.md)。

这只是两个固定原生格式的本地JSON选择，配置ID与适配器身份独立，不是
任意可执行引擎安装或完整迁移；可写导图和语义大纲仍在宿主。没有新增
资源/物理UI/OSIME/GPU对照，历史`.1/.2`资源数字不能移用为`.8`结论。

## 只读导图场景 · 历史`.7`交付：2026-11-09

根代理已读取原始失败、实际三项HostPath及合并核验源码，并查看全部三张
760×480原生测试器图，接受[14/14聚合](artifacts/acceptance/native-preview-delivery-20261109-083326-status-followup/aggregate.json)
SHA256`3DBF1F0221D08D0621F8330A1CA6AEDAB2A6AA203EB7B8C45348B5DB24323EC6`、
[核验记录](artifacts/acceptance/native-preview-delivery-20261109-083326-status-followup/validator.json)
SHA256`DD58DA93E60CAC1209F0A554A58C958382172210F281C345918BEA18D8231F2D`
及[原始脚本](artifacts/acceptance/native-preview-delivery-20261109-083326-status-followup/aggregate.ps1)。
当时入口[bin/levnote-native-worker-20261109.7](bin/levnote-native-worker-20261109.7/README.md)，
宿主加三个固定worker，23文件23,430,283字节，配置仍在APPDATA。

| 产物 | 字节 | SHA256 |
| --- | ---: | --- |
| 宿主 | 13,811,200 | `24BA9D5B11366BDD90B02467889B7272F5A0DD2456D83938BC9A20150CAC706C` |
| Markdown HTML worker | 3,300,352 | `D0BBAECBFAEBC4CA85C757600DF4804190911D30BB017C7F61197254361F327F` |
| Mindmap SVG worker | 3,122,688 | `9A07BE71C94AB1C3CC8E5FCBC1FE10DD536B83BCD8E09D53EFAFD71F0155E0F6` |
| Mindmap场景worker | 3,136,512 | `C1846B47794420151A32C68E66F1D9DDC0C2FA3D5258977CD10F9C24B4276C96` |
| 发布清单 | 4,135 | `DE4489C3A433A434BD6FF07CBFC0243EFADF7F4251400E97A93C795A2283DF6C` |

| 实际检查 | 结果与范围 |
| --- | --- |
| 中性codec | 12顶层测试PASS1.240s/vet0；独立wire、精确整数/IEEE边界、100000节点迭代树、真实32MiB及超一字节拒绝，12输入不变；[交接](handoffs/native-preview-dto.md) |
| 场景命令 | 原13顶层测试PASS1.025s，vet仅28个未具名外部测试字面量失败保留；补X/Y后受影响四布局PASS0.576s/vet0，生产未变、27输入不变；真实16MiB源与32MiB输出拒绝，不用注入小cap，[交接](handoffs/native-preview-worker.md) |
| catalog/IPC | 实际完整catalog0.508s/IPC0.598s、vet0；literal帧/描述符/预算和跨引擎拒绝；历史缺少执行前哈希清单，不用后验指纹假装证明未变，[验收](handoffs/native-preview-ipc-root-acceptance.md) |
| 最终graphview/editor/app | graphview9.785s/editor1.874s与app8.571s通过，scoped vet0；97输入、回调/队列/清理/新旧只读及另存可写安全断言，[交接](handoffs/native-preview-app.md) |
| 包装资产与新包 | 25/25资产夹具exit0；一次宿主+三个worker打包wall6.8074742s、四次实际build exit0/空stderr |
| 原HostPath | SVG PASS0.18s、HTML PASS0.14s，各两次请求；预览worker已完成独立结果校验，后在空闲状态文案测试oracle失败，原exit1保留 |
| 预览唯一续测 | 只改实际状态文案helper，新副本/tester单独预览PASS0.20s、wall1.1039693s，app vet0/空streams；没有导出重跑或产品重建 |
| 完整性 | 14/14；四个实际GUIPE、三个精确manifest、22发布项、13原样资产、19本地链接；1081输入成员不变且只有授权测试文件差异，15657工具链文件/984旧文件及原失败目录保持 |

初次native测试因Painter.Text不进入语义文字集合失败，改为稳定的语义提示
overlay后通过；完整app两项历史只读面板断言也按新source-only契约修正，
原字段/标记/标签可写安全断言保留。队列验证包括一活动/一最新pending、
导出优先及发布保留、被抢占预览恢复、mode/tab/dispose竞态、模态深度、
取消释放结果/快照和无UI派发的Close。限范围的[独立scheduler源码审查](handoffs/native-preview-consumer-independent-review.md)
撤回不成立的重复完成猜测，无行动项；中性codec由根代理直接审阅接受，
没有冒称未完成的独立codec审查通过。

原[HostPath stdout](artifacts/acceptance/native-preview-delivery-20261109-082251-package/host-path.stdout.log)
包含SVG/HTML通过和预览失败。预览已触发预期的隐藏节点定位拒绝，产品
正确显示`a.Message`；helper错误地查找空闲文案。根代理授权只改helper/
两个调用点，保持实际拒绝文字并用其真实居中位置推导29px状态栏边界。
新[preview-only stdout](artifacts/acceptance/native-preview-delivery-20261109-083326-status-followup/host-path.stdout.log)
明确RUN/PASS，未重复两个导出。**两次tester编译、两次调用、六次真实
worker请求**：原四次导出+一次预览，后一次预览；五次属于最终通过的测试。
原失败及其tester/副本/上下文完全保留，合并结果没有把失败请求抹掉。

实际只读fixture由`os.Chmod(0400)`后经document.Open识别，不用直接改flag。
独立断言312×160场景、3节点/2边、字面规范化颜色、Unicode/标记/标签和
三条整篇范围warning；模态暂缓交付先看真实job槽，再用App.View交付。
可见/隐藏节点、主题不重启worker、源码入口、原文件BOM/CRLF/未知数字/
编码、Buffer版本/选区/历史/脏状态/磁盘基准及结果槽释放均验证。
RuntimeJoined来自实际返回/Close契约，没有额外PID观察。

根代理已看[浅色场景](<artifacts/acceptance/native-preview-delivery-20261109-083326-status-followup/scene captures/preview-readonly-light-760x480.png>)、
[深色场景](<artifacts/acceptance/native-preview-delivery-20261109-083326-status-followup/scene captures/preview-readonly-dark-760x480.png>)和
[只读源码](<artifacts/acceptance/native-preview-delivery-20261109-083326-status-followup/scene captures/preview-readonly-source-760x480.png>)。
画布、source/fit、大纲、限制/警告/拒绝文字可见；显式半透明边在暗主题中
对比度偏低是保存的文档颜色，不宣称任意颜色均满足主题对比度。只读场景
没有图片或徽标，SVG既有图片占位另属导出路径。

没有物理窗口/OSIME、完整仓库重跑、资源或Tauri对照；`.1/.2`测量不能
移用到`.7`。该版本宿主仍解析可写导图与语义大纲，schema3路由当时尚未交付。
精确范围见[交付交接](handoffs/native-preview-delivery.md)和[根代理独立验收](handoffs/native-preview-delivery-root-acceptance.md)。
本次source文档更新不改已接受运行包、原始记录或聚合。

## 第二个固定导出引擎 · 历史`.6`交付：2026-11-09

根代理已读实际两项HostPath、原始失败和独立修正，并接受
[聚合attempt02](artifacts/acceptance/mindmap-engine-20261109-065002-package/aggregate-attempt02.json)
SHA256`59CFC6F967968A91D272D91B12218EEDB739C8434B400D77ACD8647983FD39C2`、
[12/12核验](artifacts/acceptance/mindmap-engine-20261109-065002-package/validator-attempt02.json)
及[核验源码](artifacts/acceptance/mindmap-engine-20261109-065002-package/aggregate-attempt02.ps1)。
当时运行目录为[bin/levnote-native-worker-20261109.6](bin/levnote-native-worker-20261109.6/README.md)，
版本`native-worker.20261109.6`，原生宿主加Markdown HTML、导图SVG两个固定
引擎，共21文件20,187,007字节；配置仍在APPDATA。

| 产物 | 字节 | SHA256 |
| --- | ---: | --- |
| 宿主 | 13,706,752 | `8A4D96A0ACE3F6E50A98E38A9B38E2C8E6F2257C25BD5101606569BD12583310` |
| Markdown worker | 3,299,840 | `11BE430F7FBA5DE126DB8658893C9E9143189F4867C728437E13841C6D5E6F1D` |
| Markdown清单 | 337 | `F9CACEF76D5B82879347800D07FBC62511D84BE516103764E4C8E84D0B014C33` |
| Mindmap worker | 3,122,688 | `F36D89C941D4ACEB71EE97F083F5FF2237DD71CC2AAEC189AF4AAF9571E7A964` |
| Mindmap清单 | 328 | `F65A6E42F33CB0B2E0C9EDF19E4B91C5BB1615715CB3FCE350B7803ADFF1E2AC` |
| 发布清单 | 3,767 | `C83ECD84B9960C1ED33E870319D4662818EE386F10BDF3389733246775BBDDE2` |

| 实际检查 | 结果及范围 |
| --- | --- |
| 纯展示/SVG提取 | 四组提取前SVG/几何/警告字节基线、预算/转义边界、native idle/source不变回归；初次完整mindmap0.567s/graphview6.811s/vet0，补充边界后完整mindmap0.691s/vet0，见[纯层交接](handoffs/mindmap-engine-pure.md) |
| catalog/IPC | 两个固定ID/format与独立预算，完整catalog0.435s/IPC0.572s/vet0；已保留执行前一字节gofmt差异，不声称该测试文件全程未变 |
| 最终runtime/graphview/app | PASS26.148s/7.005s/7.389s，scoped vet0，92输入不变；覆盖共享准入、错误身份/清单、源模式、Flush所有者、取消/清理和既有回归 |
| 实际导图命令门 | 九项明确RUN/PASS，包0.537s，test/vet exit0，20输入不变；[原始记录](artifacts/acceptance/mindmap-command-20261109-070556-actual-attempt01/result.json) |
| fresh`.6`package | 一次调用wall5.1502791s；Markdown/Mindmap/host构建1.0642834s/1.1474431s/1.9776715s，全部exit0/空streams |
| 实际HostPath | tester编译1.9284956s，一次调用wall1.1330645s；Mindmap PASS0.17s、Markdown PASS0.13s，四次真实worker请求，stderr空 |
| 独立完整性核验 | 三个实际Windows amd64 GUI PE、两份精确清单、20发布条目、13原样许可/来源资产、19本地链接；1062输入/15657工具链文件/733旧文件及完整副本不变 |

纯层独立审查曾报告XML转义可能吞掉输出上限错误；对照固定Go标准库与
新增每个转义位置预算回归，在未改产品源码时PASS，审查者撤回该问题。
完整过程见[审查复现](handoffs/mindmap-engine-pure-review-fix-result.md)，
没有伪造失败执行或另作防御性产品修改。
[独立集成源码审查](handoffs/mindmap-engine-integration-independent-review.md)
没有可行动问题；它不涵盖当时尚未落盘的命令测试，后者以随后真实执行的
九项记录为准，不以早先手写汇总代替stdout和输入清单。

[实际HostPath stdout](artifacts/acceptance/mindmap-engine-20261109-065002-package/host-path.stdout.log)
含两项明确RUN/PASS，每项先成功再拒绝已有目标，共四个worker请求。
导图返回704字节SVG、3条整篇范围warning、2次dialog；独立XML断言
`viewBox="-24 -24 196 144"`、节点92×40位于28,28、Unicode/标记/标签、
颜色及图片占位，另以纯API作字节比较。Markdown返回331字节HTML、1条
warning、2次dialog，沿用标题/任务/字面表格与转义独立断言。两项均保留
原文件BOM/CRLF/源码、未知元数据、编码、Buffer版本/选区/历史/脏状态和
磁盘基准。SaveDialog被注入，使用实际NewBundled；RuntimeJoined=true依据
已测试的返回契约，没有独立PID观察或物理宿主窗口验收。

原聚合器已写出PASS12/12的`validator.json`，随后递归哈希自身仍打开的
`aggregate.stderr.log`遇到Windows共享冲突，exit1；原脚本/日志/结果全部
保留。根代理只授权新的attempt02排除其当前两个活动stream，退出后另存
[最终stream指纹](artifacts/acceptance/mindmap-engine-20261109-065002-package/aggregate-attempt02-command.json)。
未重跑构建、测试或worker；修正聚合exit0、12/12。先前app中文多余空格
夹具失败及改正亦保留，不影响当前已接受结果。

本轮没有全仓重跑、完整UI启动、资源对照或物理OSIME/GPU声明；runtime
reparse测试因无符号链接权限SKIP，其余所属生命周期helper通过。`.1/.2`
资源数字仍只属于原版本。全部实际命令、精确哈希和限制见
[交付交接](handoffs/mindmap-engine-delivery.md)与
[独立根代理验收记录](artifacts/acceptance/mindmap-engine-20261109-065002-package/root-acceptance.json)。
最终源码文档另存；已接受包、聚合与历史证据不随本次文档修订改变。

## Markdown所选列对齐 · 历史`.5`交付：2026-11-09 05:56构建

根代理已读取并接受[12/12聚合](artifacts/acceptance/markdown-alignment-20261109-055547-package/aggregate.json)，
SHA256`A32B88174E17EF6BA50190EC9C83A4840F1E0C4A762C6AD5B2BFAB2960AA5C42`，
[核验记录](artifacts/acceptance/markdown-alignment-20261109-055547-package/validator.json)及其
[实际源码](artifacts/acceptance/markdown-alignment-20261109-055547-package/aggregate.ps1)。
当时目录为[bin/levnote-native-worker-20261109.5](bin/levnote-native-worker-20261109.5/README.md)，
版本`native-worker.20261109.5`。19文件17,038,894字节，13个原样许可/来源
资产、18项相对发布清单和19个本地链接通过；配置仍在APPDATA。

| 产物 | 字节 | SHA256 |
| --- | ---: | --- |
| 宿主 | 13,686,784 | `1272E5B13D1331B81A92111E242D9ACCF153363C6BFDC9F8748A11DC19BE33C7` |
| worker | 3,295,744 | `DC3E0F42BA27961DFEE714C205AB48A8C0722BADCFB3546CE4C6D25CEA68AAAA` |
| 引擎清单 | 337 | `BA5BBD0B9D747319A549764580D6D93BF5AF967F0162270710B23A427A29E443` |
| 发布清单 | 3,411 | `77D5BE4228ADF28D15637741FF8D966B28B5F6A90295E010D9742AC9D1C123E3` |

| 本轮检查 | 实际结果 |
| --- | --- |
| 最终完整Markdown格式/视图 | PASS0.809s/2.499s，combined wall4.4817699s；10输入不变 |
| 格式/视图vet | exit0/wall1.0456751s，空streams |
| 对齐/既有行consumer | PASS1.741s；三组对齐功能测试及两张opt-in图，既有行测试通过 |
| 修正capture02 | PASS1.675s；保留capture01，正常Escape关闭键盘tooltip后焦点/草稿/Enter路线仍有效 |
| 最终完整app/scoped vet | PASS6.441s，command wall7.8643048s；vet exit0/wall1.3845038s，14输入不变 |
| fresh`.5`package | exit0/wall4.330298s；worker1.1482483s、host2.2525709s |
| 既有真实HostPath | tester编译1.8377442s；一次RUN/PASS0.20s，command wall2.535937s，含两次worker请求 |
| 完整性 | 1037输入、15657工具链文件、530旧文件与19副本清单/尺寸/哈希不变；旧文件UTCmtime ticks亦匹配 |

纯API五组测试覆盖16种对齐转换/无操作、独立精确字节、源/表/格增长限制、
合法收缩与全篇描述项耗尽后的另一回退表。共享行/对齐管线的十一组新增
原生测试覆盖两段undo、完成面板、composition几何、held/repeat/真实scroll、
rapid continuation、Changed/Report释放/替换/同buffer原位改动、选区与idle。
失败的unused变量、非法代理对选区及实际点击未命中夹具保留并修正；完整
旧row/cell回归仍通过，没有为旧内部local error断言削弱回调ownership。

真实注册adapter使用BOM+CRLF、Unicode/转义管道、不同标记填充和两个表：
header/body改列后只变分隔标记，干净Noop不改源/历史/选区/脏状态/磁盘；
待写cell+对齐为两次undo/redo，cell+Noop只有cell一次。Save/reopen与三模式
保留精确字节；参考View.Export独立断言th/td样式及字面HTML，不启动worker。
split冲突保留输入，真实RecoverDraft打开精确未保存副本；自身cell成功后
对齐超限拒绝明确说明已写回。只读无编辑动作。

根代理和实现代理均看过[浅色所选列](artifacts/acceptance/markdown-alignment-20261109-055235-capture-attempt02/images/markdown-alignment-light-column-760x480.png)
及[深色恢复](artifacts/acceptance/markdown-alignment-20261109-055235-capture-attempt02/images/markdown-alignment-dark-recovery-760x480.png)。
actual preview264px、完整输入226×48px，按钮最下沿443.8213px在实际status
上沿451px之前。status按实际居中文本与29px行高推导；原文本Y减padding
公式与遮挡提示的capture01仍保留。正常Escape只关tooltip，不提交/清除
草稿；之后Enter执行精确内容且形成两条撤销。深色实际恢复验证精确输入。

[GLM独立源码审查](handoffs/markdown-alignment-independent-review.md)无可行动问题，
范围是pure/native源码与已有证据，不包括OSIME/package执行。包验收仅注入
SaveDialog，真实`.5`NewBundled路径一次tester包含成功和独占目标拒绝两次
worker请求；HTML331B、一条warning、两个dialog、RuntimeJoined=true。
没有额外PID观察、完整UI启动、资源测量或无关套件。`.1/.2`B/op与进程峰值
数字不属于`.5`，不以测试器截图宣称真实OSIME/GPU或资源优势。

完整命令/输入与保留失败见[对齐交接](handoffs/markdown-table-alignment.md)。
[独立根代理验收记录](artifacts/acceptance/markdown-alignment-20261109-055547-package/root-acceptance.json)
按实际时间另存，接受的聚合、核验记录与运行包不随文档修订改变。

## 导图标签 · 历史`.4`交付：2026-11-09 05:06构建

根代理已读取并接受[12/12聚合](artifacts/acceptance/mindmap-tags-20261109-050613-package/aggregate.json)，
SHA256`415F685CD76CECC50AB5F9D52BFDC21E6662243B782CBDAF96D3B75B19F8D330`，
以及[独立核验记录](artifacts/acceptance/mindmap-tags-20261109-050613-package/validator.json)
`FD05487F2206BBF86F30C2663AE8AF8F2F94F4A771ED76EDE0FFE2B250384598`。
当时运行目录[bin/levnote-native-worker-20261109.4](bin/levnote-native-worker-20261109.4/README.md)，
版本`native-worker.20261109.4`；19文件17,014,847字节，13原样许可/来源资产、
18项发布清单及19本地链接通过。配置仍在APPDATA，旧包和已接受证据未改写。

| 产物 | 字节 | SHA256 |
| --- | ---: | --- |
| 宿主 | 13,663,232 | `29E6148B844FDACB3CF12854ED95F1F2D3F779B05E711BB5C3A30DAA964C6A6E` |
| worker | 3,295,744 | `9EF039FE01C81CE7445DE3A571947D752D611C439A081B4E35B660731A6F4537` |
| 引擎清单 | 337 | `839D5D88DC308C67FAF846BC4B779503228840D6988E293861918DE5F8294027` |
| 发布清单 | 3,411 | `BC08D9EFDC43997B1956ED1C95BCF2FC9D84A6B60A582E26DAE800823EF52AFC` |

| 本轮检查 | 实际结果 |
| --- | --- |
| 纯标签聚焦及完整格式 | PASS0.691s、完整mindmap0.534s；纯格式其后未改，消费阶段沿用 |
| 最终完整graphview/app | PASS7.607s/6.957s，combined wall8.8931777s；15输入不变 |
| 最终consumer vet | exit0/wall1.3526358s，空stdout/stderr |
| 最后受影响回归/两张图 | graphview2.531s、app首warning与capture4.294s；更早失败记录保留 |
| fresh`.4`package | exit0/wall3.8862651s；worker0.9362357s、host2.0892809s |
| 既有真实HostPath | tester编译1.7653516s；一次RUN/PASS0.78s，command wall1.632111s，含两次worker请求 |
| 完整性 | 1031输入、15657工具链文件、373保护文件及19副本清单/尺寸/哈希不变；旧文件UTCmtime ticks亦匹配 |

真实注册适配器的五个非opt-in测试覆盖精确Unicode/空白/逗号添加、重复项
序号移除、最后删除/全清`[]`、对象class/style存在性和未知Raw数值/顺序、
未编辑BOM+CRLF保存及真实编辑canonical LF保存重开、字面SVG XML转义、
只读/三模式、选区/分别撤销。待添加tag阻止Save/export/tab/mode/close，
split源码冲突保持双草稿；真实同步app RecoverDraft打开精确未保存副本。
字段自身写回使Raw预算增长后拒绝tag，已写字段仍可独立撤销。新无警告
fixture在preview/split创建与清除首次warning，连续组合和不同文字确认
仍精确替换marked文本，只添加一次，不改源磁盘或留下过时status。

原生回归还覆盖按下身份、重复/连续多击、scroll/cancel/Release、同步
Changed替换来源/输入、post-field budget及清空pending提示。纯API无UI
上限；视图新输入512B、当前/候选64项/8KiB预算和16MiB导图限制均独立验证。
未提交tag只保留在运行窗口，未承诺进入会话/崩溃恢复。

已审阅[浅色输入与操作](artifacts/acceptance/mindmap-tags-20261109-050143-consumer-attempt07/images/mindmap-tags-light-expanded-760x480.png)
和[深色双草稿拒绝恢复](artifacts/acceptance/mindmap-tags-20261109-050143-consumer-attempt07/images/mindmap-tags-dark-recovery-760x480.png)：
分别43,401B/SHA`A8CC3DD9…`与46,478B/`17A74B42…`。输入、Add/discard/recover
在同帧可读，所有相关按钮单独滚动检验Bounds在状态栏上方；暗图实际激活
恢复后验证两份精确输入。截图fixture保留重复/空/对象标签，省略无关未知
字段/图片警告以展示操作；丰富Raw/BOM测试仍使用独立原始fixture。

consumer-attempt01–07和dark-diagnostic完整保留：包括tester尺寸类型、
空AccessInfo.Value错误测试预期、普通恢复行溢出、fmt跨目录参数错误、
首次warning导致焦点和scroll丢失，以及确认后旧status提示。对应有限修正
为不同可访问名/紧凑恢复按钮、原生copy断言、稳定格式视图Key和本地确认
sentinel；真实验证/预算/冲突错误仍报告。细节见[标签交接](handoffs/mindmap-tags.md)。

包验收只注入SaveDialog，使用真实NewBundled/.4worker；HTML331B、一条
warning、两次位置回调，拒绝同名目标覆盖，精确源BOM/编码/历史不变且
RuntimeJoined=true。没有额外PID观察、完整UI启动、资源实验或无关套件。
测试器截图不证明OSIME/GPU；下方`.1/.2`资源数据仍不属于`.4`。worker因
版本字符串变化而有新哈希，不声称与旧包二进制相同。

[独立根代理验收](artifacts/acceptance/mindmap-tags-20261109-050613-package/root-acceptance.json)
按实际记录时间另存，聚合/核验/交付包均保持接受时原样。

## Markdown数据行 · 历史`.3`交付：2026-11-09 04:05构建

根代理已读取并接受[12/12聚合](artifacts/acceptance/markdown-table-rows-20261109-040306-package/aggregate.json)，
SHA256`9C1FE658598A48D2B07AAB910AE45C22BDD4B0CAFE4572B98E700D78D3FAA345`。
当时运行目录[bin/levnote-native-worker-20261109.3](bin/levnote-native-worker-20261109.3/README.md)，
版本`native-worker.20261109.3`，19文件共16,942,149字节。13个许可/来源原文
副本、19个本地链接、release manifest的18项尺寸/哈希和唯一自排除均通过。
用户配置仍在APPDATA。旧包与已有证据未改写。

| 产物 | 字节 | SHA256 |
| --- | ---: | --- |
| 宿主 | 13,591,552 | `61512C03DDB9F63EB51B59A09C8960B3B52333F7B3E0F44BB0F3410DD3FA2E44` |
| worker | 3,295,744 | `6CFE7157B40F974C08A8DC88D920E2619C5A814D4E1AF88070DFBCC5FF4F43F3` |
| 引擎清单 | 337 | `0DF33AA2FD39B9A606AE4F24BEFFE56D18F21EBF6BFB951A5F0256FBB6D14B08` |
| 发布清单 | 3,411 | `0AE777D6B1BE26095987E7422AAE86C3EC4064FBB4C12D314EA80B3291A3FFE2` |

| 本轮检查 | 实际结果 |
| --- | --- |
| 纯行补丁聚焦 | PASS0.521s；精确LF/CRLF/CR/EOF、重复行、预算和不相关回退状态 |
| 完整格式包 | PASS0.647s；其后格式源码未变化 |
| 最终完整Markdown视图/app | PASS2.306s/5.703s，combined wall8.1742019s |
| 最终consumer vet | exit0/wall1.2836549s，空stderr |
| 集成行操作与截图 | 聚焦app PASS1.549s，wall4.0140581s；两张760×480明暗分屏 |
| fresh`.3`package | exit0/wall3.9761635s；worker1.139006s、host1.9588919s，空build streams |
| 既有真实HostPath | tester编译1.6212914s；一次实际RUN/PASS0.19s，command wall0.8690875s，内含两次真实worker请求 |
| 完整性 | 1025输入、15657工具链文件、179保护文件不变；保护文件UTCmtime ticks亦匹配，19份测试副本不变 |

行操作通过真实注册适配器的pointer/F6/Tab/Enter/Space触发。独立期望验证
BOM/CRLF与邻接原文、表头/正文插入及精确删除、干净格一次undo/改动格
两次undo-redo、有方向选区、保存重开、三模式和字面转义HTML。行补丁只改
所选物理行，沿用8列/256正文行/4096字节格/32KiB表/8MiB源；完整候选还
保护全篇16,384cell预算对其他表的影响。删除使别的回退表恢复也保守拒绝。
单元格写回失败保留草稿；写回已成功而后续行被拒绝时，清楚报告并保留
已写回内容的独立撤销，不声称尚未写回。

本轮保留全部失败：测试类型/焦点预期、组合焦点与滚动分派、变化提示导致
按钮位移、暗色窄窗按钮落入状态栏以下，以及真实连续多击穿透新面板的
删除诊断。对应有限修正为本地F6组合保护、滚动取消、固定提示位、48px
单格编辑器/短错误及原始多击计数守卫。后者有无时间推进的实际红绿回归，
先证明后续点击确实落在删除按钮，再断言拒绝及下一次独立点击只提交一次。

已审阅[亮色暂存图](artifacts/acceptance/markdown-table-rows-20261109-035447-consumer-attempt02/images/markdown-rows-staged-light-760x480.png)
与[暗色拒绝恢复图](artifacts/acceptance/markdown-table-rows-20261109-035447-consumer-attempt02/images/markdown-rows-refused-dark-760x480.png)。
所有恢复/完成/行按钮bounds位于y451状态栏上方；恢复实际产生精确草稿，
组合提交没有重复文字。随后输入守卫不改变布局，复用这两张图，没有重拍。

包验收仅SaveDialog注入，使用实际NewBundled及固定`.3`worker；先成功导出，
再拒绝同名目标覆盖。HTML331字节、一条警告、两次位置回调，源文件BOM/
编码和文档状态均不变，RuntimeJoined=true。未独立观察worker PID，不以
此声称物理对话框或完整宿主UI启动。本轮无新UI启动、资源、runtime或全
仓库重复套件，测试器不证明系统IME/GPU或所有指针序列。下方资源数字仍
只属于标明的`.1/.2`，不是`.3`测量。

源码、全部原始日志和历史失败入口见[行操作交接](handoffs/markdown-table-rows.md)；
[独立根代理验收记录](artifacts/acceptance/markdown-table-rows-20261109-040306-package/root-acceptance.json)
按实际创建时间另存，不修改聚合或发布目录。

## Markdown worker实际进程对照 · 2026-11-09 03:16

根代理接受[进程聚合](artifacts/acceptance/markdown-worker-resources-20261109-031432-matrix-attempt02/aggregate.json)
SHA256`1DE4E8D3682B11FFA6200993C7B186C71F3E0367D2B6A697A3F764F26EE18F2C`，
PASS8/8。同一Windows测试器通过真实`NewBundled/Export/Close`运行已验收
`.1/.2`worker与清单副本；每夹具按旧/新交替三对，每行新测试器进程，
无预热。12次调用、12个真实worker均PASS，源/HTML/警告顺序哈希匹配。
本轮未改产品或运行包，也未新构建/启动应用UI。

| 夹具 / 版本 | OS峰值工作集MiB中位数 [最小,最大] | OS峰值进程提交量MiB中位数 [最小,最大] | runtime耗时ms中位数 [最小,最大] |
| --- | --- | --- | --- |
| 1MiB转义围栏 .1 | 25.273 [25.250,25.875] | 33.918 [31.570,36.422] | 32.615 [32.113,437.402] |
| 1MiB转义围栏 .2 | 15.199 [15.031,15.234] | 21.406 [21.320,21.781] | 33.517 [31.110,579.734] |
| Mixed32 .1 | 7.699 [7.688,7.773] | 13.457 [13.445,13.484] | 22.743 [22.283,26.587] |
| Mixed32 .2 | 7.527 [7.504,7.727] | 13.133 [13.129,13.297] | 21.837 [21.128,23.110] |

转义夹具峰值工作集/提交量中位数降低39.86%/36.89%；Mixed32降低
2.23%/2.41%。这是每组3个本机样本；此前80%/66.8%B/op下降是累计
渲染器分配，不能当作进程峰值的比例。此处是worker自身的Windows
`PeakWorkingSetSize`及`PeakPagefileUsage`，后者为进程提交量峰值，
不是磁盘I/O；单独保留采样`WorkingSetSize/PrivateUsage`，不相加。

Windows10.0 build26200、amd64、Go1.27.3、16逻辑CPU。afterStart用
WithHandle复制精确进程句柄，名义2ms采样，每行实际6–12次；106次
内存查询与24次进程时间查询均成功，退出后最终峰值可读。12个句柄
均观察到exit0，采样器/runner已join并关闭句柄，275个保护文件和6个
副本保持尺寸/哈希/UTCmtime。原始记录保留每次时间、间隔、错误与完整性。
首轮旧/新437.4022/579.734ms未排除，原因未测定；CPU读数为
0/15.625/31.25ms，不能据此说零成本或提速。没有应用UI/测试器内存、
GPU、系统总量或与Tauri的比较，也无跨机器保证。

首次编译的PowerShell参数绑定失败发生于Go之前；首个矩阵调用因探针
manifest期望摘要少一个`b`而在启动前拒绝（调用1、worker0）。原始
失败保留，root批准常量修正、无进程哈希预检5/5及新测试器编译/SKIP后
才执行上述12次；未把失败或短样本补成额外请求。方法、全部原始入口、
更正指纹及单独验收记录见[进程测量交接](handoffs/markdown-worker-resources.md)。

## 有界HTML生成 · 历史`.2`目录：2026-11-09 02:49构建

根代理接受 [bin/levnote-native-worker-20261109.2](bin/levnote-native-worker-20261109.2/README.md)，
版本`native-worker.20261109.2`。该改动先在输出预算内计数，再精确分配
一次结果字节并填充，worker不再从完整HTML字符串复制字节；HTML/警告
字节和顺序保持。旧`.1`目录、标记exe与原证据均保留不变。

| 新产物 | 字节 | SHA256 |
| --- | ---: | --- |
| 宿主 | 13,560,320 | `8B4FDF6FB232A92F6DFA45FB48443B6611F9D572DCB06A126B69C69BDB7ECF06` |
| worker | 3,295,744 | `94DB0FC39A538D78AF06A1A40384D059BFC86A8790A57655424377F7754F16AC` |
| 引擎清单 | 337 | `790DCCC0A968D59A820392DB6C2DA71392B8768EE1A05C05E7B2221B2A132F73` |
| 发布清单 | 3,411 | `3150D782D8E12AD88451E09E9191D4B6DB83E89F6C045703EF4B203D37B63C68` |

宿主/worker均为Windows amd64 GUI PE32+，UTCmtime分别为
2026-11-08T18:49:19.1374243Z、18:49:17.1454938Z。完整目录19文件、
16,910,419字节，13个许可/来源副本逐字节匹配，19本地链接有效，发布清单
18条覆盖全部其他文件且明确排除自身。构建及测试证据位于单独目录。

| 此轮检查 | 实际结果 |
| --- | --- |
| 首轮完整exporter/worker包 | PASS0.644s/0.864s；exact8MiB五倍转义worker测试只运行一次，PASS0.16s |
| 计数优化后的完整exporter包/两包vet | PASS0.499s；vet exit0/wall1.2318809s、空streams |
| 完整Markdown视图/app | PASS1.963s/5.550s，combined wall8.2188577s |
| 最终worker两项parity/tiny-cap framing | PASS0.651s；未重复8MiB测试 |
| consumer vet | exit0/wall1.4379224s、空streams |
| 新`.2`包 | exit0/wall4.0429435s；worker1.1316821s、宿主1.9520156s、空build streams |
| 现有真实host路径测试 | test-c编译exit0/wall1.6497914s；实际RUN/PASS0.19s、command wall2.3513067s |
| 聚合核验 | PASS10/10；183冻结输入、51旧文件尺寸/哈希/UTCticks不变 |

真实host路径测试在新payload的可丢弃证据副本旁运行，用户目录不含tester。
使用实际注册适配器、NewBundled与真实`.2`worker，仅SaveDialog注入。
既有独立HTML语义、警告范围、源文件BOM/编码、版本/选区/历史/脏状态和
同名目标拒绝覆盖断言通过。HTML331B SHA256
`18FBE1F3BE429C7AF1F975E208933F08E154A6C2396FF75CDAB61D516356B758`，
一条警告/两次位置回调，RuntimeJoined=true。未独立观测workerPID，不
据此声称物理对话框/IME/GPU/隐藏启动验收。无关runtime、全仓库、缺失/
损坏副本、startup、截图未重跑。

预改源在独立目录保留16文件，前后使用相同解析夹具、工具链、benchmark
循环、10x benchtime/3samples。解析不在计时内；旧adapter为checked字符串
再转自有字节，新adapter调用字节接口。输入、HTML、警告哈希完全一致。

| 解析夹具 | 输出字节 | B/op中位数旧→新 | allocs/op | ns/op中位数旧→新 |
| --- | ---: | --- | --- | --- |
| 1MiB连续&代码块 | 5,242,960 | 26,257,427→5,251,083（-80.00%） | 11→1 | 5,000,990→5,639,960（+12.78%） |
| 混合文档32段 | 24,203 | 171,386→56,880（-66.81%） | 706→283 | 119,680→114,420（-4.40%） |

这是本机小样本的累计Go分配B/op，不是进程RSS、峰值、CPU使用率或速度
保证。计数/填充需要第二次遍历；源文/AST、allocator、宿主结果副本仍占
内存。第一版计数实现转义样本约2.16倍耗时的原日志保留，依据实测仅改为
共享实体映射的checked增量计数和跳过空写入，最终取上表结果。

[focused记录](artifacts/acceptance/markdown-export-memory-20261109-024336-optimized/focused-checkpoint.json)
SHA256`A572ED95AFEB9F1D8089A235CDF4301BDF80B015DC37A715884CB06F59BB778F`；
[集成聚合](artifacts/acceptance/markdown-export-memory-20261109-025501-metadata/aggregate.json)
SHA256`FCD3293A9C6A8F6757083BC59C2FEFF499E5917B50A1F5B83B1F9982D6AF9F5C`。
首次聚合脚本碰到PowerShell只读`$Host`变量，尚未创建聚合；原脚本/错误
说明保留，只修正变量名后在新目录生成元数据，没有重复产品阶段。
实际命令、原始日志与独立验收记录见[交接](handoffs/markdown-export-memory.md)。

## 自动完整打包 · 历史`.1`目录：2026-11-09 02:23构建

根代理已接受 [脚本组装的运行目录](<bin/levnote native worker 20261109.1 packaged/README.md>)：
19文件共17,020,368字节，固定版本`native-worker.20261109.1`。宿主、worker、
引擎清单的字节/哈希与下面01:38原验收完全相同；原产品功能验收继续适用，
本轮未重复完整产品测试、启动或资源测量。旧人工交付、标记exe和原候选
目录均保持原字节与UTCmtime。

| 检查 | 实际结果 |
| --- | --- |
| 构建守卫 | 32个不同用例通过：26项拒绝、4次实际夹具编译、JSON环境检查、未打补丁时refresh所需mod操作 |
| 目录与资产夹具 | PASS24/24，exit0/wall1.0431872s；含空格路径、不同PowerShell当前位置、拒绝前无副作用、已有重解析目录、完整许可/链接/哈希 |
| 同版本生产打包 | exit0/wall3.1721468s；worker0.9537688s、宿主1.3266586s，构建stdout/stderr均0，GUI PE32+通过 |
| 运行资产 | 3个payload与原验收逐字节相同，13个许可/来源副本与当前源码完全相同，19个本地链接有效 |
| 发布清单 | 18条覆盖全部其他文件，明确排除自身；3,411B，SHA256`A1E511BAC26E57F49103F947D4D197541480D9BD4918ECD4FA9007AA7D6976FB` |
| 冻结/旧产物 | 171个产品+10个审阅过的打包/文档输入构建前后匹配；32个旧资产/证据保持尺寸、哈希、UTCticks |

两处证据错误没有覆盖：首轮守卫30/31、exit1/wall30.6577072s，把Go对
`GOENV=off`返回的空配置路径误期望成`off`；新目录只重跑JSON环境断言与
此前跳过的refresh检查，PASS2/2、exit0/wall1.9602773s。首份包核验11/12
误把完整保护基线32文件写成22；额外10个是原候选中的构建/测试证据，
文件比较均通过。独立4/4修正明确22个必需文件+10个旧证据的集合和哈希，
与首份有效11项共同完成12项包条件。没有重复打包或资产测试。

[首份核验](<artifacts/acceptance/portable build 20261109-022315 production/package-verification.json>)
SHA256`09B2A7AA56AD0FB9DD38A79467399A535EBA915F2A7971BA8628D85AECE7A4D9`；
[独立计数修正](artifacts/acceptance/portable-build-20261109-022549-protected-correction/protected-correction.json)
SHA256`E8C093B9A687FAD0EB380608FF946B6C26F115097513B7AE5909422722B1C8F8`。
命令、当前源指纹、实际streams和有限结论见[打包交接](handoffs/portable-build.md)。
这证明本机固定输入的一次重复打包，不证明跨机器可复现、签名或不可信
源码安全隔离。可移动程序目录继续使用APPDATA用户配置。

## 独立 Markdown HTML worker · 历史`.1`功能验收：2026-11-09 01:38构建

根代理已接受本次精确宿主/worker/清单、真实导出与拒绝路径，以及20/20
证据核验。可运行目录是 [bin/levnote-native-worker-20261109.1](bin/levnote-native-worker-20261109.1/README.md)，
由已验收的原始候选逐字节复制，并补齐许可与来源文档。不是重新构建，
旧 `native/bin/levnote-native.exe` 标记产物保持原尺寸、哈希与mtime。

| 检查 | 实际结果 |
| --- | --- |
| 全量 `test ./... -count=1 -timeout 300s` | exit0，wall26.7777398s；19个测试包+4个无测试包；app6.930s、worker1.083s、protocol2.151s、runtime22.884s |
| 全量 `vet ./...` | exit0，wall1.8217958s，stdout/stderr均0 |
| 固定脚本构建宿主+worker | exit0，wall3.900804s；显式windows/amd64/v1；两者PE32+ GUI核验通过 |
| 真实随包worker应用路径 | 明确RUN/PASS0.21s；331字节HTML、一条警告、两次保存位置回调；第二次拒绝同名覆盖 |
| 缺失/改动worker副本 | 明确RUN/PASS0.01s/0.02s；unavailable/integrity，无对话框、回退或目标文件 |
| 实际宿主隐藏启动与所属WM_CLOSE | 捕获command4.8221416s、内部smoke4.2671225s；PID19384退出0；无超时/强制清理；哨兵36776存活后由自身事件正常退出 |
| 证据核验 | PASS20/20；174个冻结输入、75个证据文件、候选payload和旧标记exe保持一致 |

宿主13,613,568字节，SHA256
`4B5DCECA5E5F5C3D3609336563F138247942E35E0C3BE65B382B32771A80B23C`，
mtimeUTC`2026-11-08T17:38:51.9981206Z`。
worker3,352,576字节，SHA256
`DB8F02E6E187833FB52B4FC02B076B4AC96E357FB3E554533A698D8F050F6AD0`，
mtimeUTC`2026-11-08T17:38:49.8028912Z`；版本`native-worker.20261109.1`。
337字节清单SHA256`64472251BD737D3BB82AFB05C35F648882D2E9D8D4A57BCFD1CDE8CF457CC453`。

[最终聚合](artifacts/acceptance/markdown-worker-integration-20261109-015232-attempt04/aggregate.json)
SHA256`FB0468DADF9F526AFD7BAE72BD1E3893CAD7CEE78176870DDDAE5BB59B12A83A`；
[20项核验](artifacts/acceptance/markdown-worker-integration-20261109-015232-attempt04/validator-result.json)
和[独立验收记录](artifacts/acceptance/markdown-worker-integration-20261109-015232-attempt04/root-acceptance.json)
保留实际时间与文件哈希，接受后不改写原聚合/核验文件。
完整命令、stream、原始候选及失败历史见[集成交接](handoffs/markdown-worker-full-integration.md)。

实际导出使用注册的Markdown源码/noView入口、固定NewBundled和真实worker，
仅保存位置对话框被测试回调替换。除与纯导出结果逐字节/警告范围比较外，
还独立断言标题ID、禁用任务方框、表格字面转义、原始HTML及不支持链接。
两次导出后原文件BOM/字节、编码、Buffer版本、选区、撤销、脏状态和磁盘
基准保持不变。生产workerPID没有独立观测；清理完成依据是已验证的runtime
返回契约。隐藏smoke只启动宿主，不执行导出；关闭后隔离session确认已打开
只读UTF8BOM夹具且无草稿，三个payload及四个夹具/哨兵哈希不变。

聚焦门此前覆盖严格协议/长度/UTF8/JSON、真实Job生命周期和持有的后代
句柄等待；文件symlink分支因Windows权限不足SKIP。后代/父进程死亡是
仓库测试helper证据，worker本身不派生子进程。app聚焦1.773s、完整app包
5.411s、scopedvet1.525383s及四张760×480明暗进度/状态图经根代理核对。
没有独立同伴审查结论或真实OS IME/GPU/系统对话框验收。

完整集成的失败记录均保留：`test -c`被已有包装器大小写不敏感的`-C`守卫
拒绝，改用`test -run '^$' -o`生成测试exe且运行零测试；PowerShell裸点号
参数被解释成`-test`，改为字面字符串数组；首轮smoke立即读取MainModule
为空，持有的所属进程被清理，后续以QueryFullProcessImageName读句柄并在
关闭后核对session。首份聚合552510…因JSON自动日期类型误比较得到19/20，
原文件保留，最终元数据用UTCticks通过20/20。成功的产品测试/构建不重复。

历史`.1`输入8MiB、结果64MiB，但渲染器先构造完整HTML再拒绝超限，不是峰值内存
或流式保证。最终WriteNewFile仍同步占用UI且开始后不可取消。此轮未做
进程资源采样、新旧对照、签名、安装器或可复现打包验证。可移动程序目录
仍使用APPDATA配置，不是自包含用户数据便携模式。

## 导图标记 · 历史验收：2026-11-09 00:04构建（本地时间）

根代理已接受19字形标记的纯格式/视图/真实应用聚焦门及最终四张760×480
明暗/分屏/恢复图片，再授权冻结源码的一次完整序列。最终应用聚焦原始
日志为PASS2.817s，graphview完整包4.477s、vet退出0。新的完整序列结果：

| 检查 | 实际结果 |
| --- | --- |
| `test ./... -count=1 -timeout 300s` | exit0，9.480s，20包摘要；app6.942s，graphview5.419s |
| `vet ./...` | exit0，1.829s，stdout/stderr均0字节 |
| 去符号Windows GUI发布构建 | exit0，2.541s，stdout/stderr均0字节 |
| 隔离隐藏启动、所属窗口WM_CLOSE | exit0，3.450s；PID12100，Visible=false，无强制清理/超时/残留应用进程 |
| 证据核验 | PASS15/15；154个冻结文件指纹全部不变，四图副本匹配聚焦原件 |

精确产物：[验收副本](artifacts/acceptance/mindmap-markers-integration-20261109-000341/levnote-native.exe)，
13,461,504字节，SHA256
`17E014EB6F61E671F6FCCA439719E8831B7591042A6F29EE5354960C120C0466`，
mtimeUTC`2026-11-08T16:04:08.3209774Z`。PE32+ x64 GUI subsystem2，
Go1.27.3。此前34F3F48C…单元格exe在构建前复制并核对，未被覆盖丢失。

[聚合结果](artifacts/acceptance/mindmap-markers-integration-20261109-000341/result.json)
SHA256`6C095CBB82F852897707A3E92AA4A081500303640D43B24A82B610E941B720F8`；
[证据核验](artifacts/acceptance/mindmap-markers-integration-20261109-000341/evidence-validation.json)
SHA256`6071D83B0C7EC0B6B1448DFC30AB268B29CD0B43BC8BA348F229EC7141BFC57D`。
[根代理验收](artifacts/acceptance/mindmap-markers-integration-20261109-000341/root-acceptance.json)
单独记录明确接受与实际建档时间，不改写两份已核对JSON。
[smoke原始记录](artifacts/acceptance/startup-smoke-mindmap-markers-20261109-000341/result.json)
保留实际应用stdout/stderr（均0）、导图fixture/target与更新形状文件/目录
哨兵哈希，均未变化。所有命令流在执行时捕获，没有补造空日志或重跑。

覆盖范围包括64项/4096字节当前与候选上限、512字节摘要、已知组互斥与
未知顺序、显式全清`[]`、草稿/候选身份保护，以及Unicode/自定义字形/XML
转义和有界SVG标签。标签编辑、节点图片渲染和完整旧版功能仍不在范围内。
四图只证明滚到的全清/恢复控件可达，全部19控件不同屏，窄分屏源码区受
挤压，画布标签可裁切。测试器与隐藏启动都不证明真实OS IME/GPU验收。
本轮没有资源采样或新旧实现的资源比较。

历史聚焦失败保留在`mindmap-markers-20261108-234000-current-run`，FAIL2.623s；
修正的是优先级4替换3的预期，以及用RoleButton取得28DIP恢复按钮完整边界。
最终日志在`mindmap-markers-20261108-234731-final-run`。更早230124目录被作者
重跑覆盖；复制成功图到231014不能恢复原始失败字节。详细来源见
[应用交接](handoffs/mindmap-markers-app.md)。聚合结果内早期失败路径少了
`-current-run`后缀，已用[独立勘误](artifacts/acceptance/mindmap-markers-integration-20261109-000341/evidence-notes.json)
记录，保持接受时的原文件哈希不变。

## 导图字段与 doc-block 修正 · 2026-11-08 18:17集成检查点

根代理已接受字段的纯格式、原生视图、真实应用定向证据，以及760×480
明暗/只读/冲突图片。所属完整 graphview 包测试2.631s通过，vet无诊断；
纯 mindmap 包0.627s、Markdown doc-block 修正包0.657s通过，各自vet无诊断。
应用真实适配器定向用例1.933s通过、app vet无诊断。集成负责人随后对冻结
源码执行一次完整序列：全套测试9.738s通过（app5.971s、graphview4.403s），
全套vet2.358s无诊断，去符号GUI构建3.207s通过，隐藏启动关闭3.882s通过。
以下产物已独立核对并经根代理最终验收；此前16:45副本保留。

```powershell
.\native\scripts\go.ps1 -GoArguments @('test','./...','-count=1','-timeout','300s')
.\native\scripts\go.ps1 -GoArguments @('vet','./...')
.\native\scripts\go.ps1 -GoArguments @('build','-trimpath','-buildvcs=false','-ldflags','-s -w -H=windowsgui','-o','bin\levnote-native.exe','./cmd/levnote')
```

| 构建项 | 本轮记录 |
|---|---|
| 文件 | `native/bin/levnote-native.exe`；[验收副本](artifacts/acceptance/mindmap-fields-20261108-181628/levnote-native.exe) |
| 字节数 | 13,274,624 B（约12.66 MiB），未压缩的去符号exe |
| SHA-256 | `F06ABEF11653EF93F03D3017A62215CACD8364536E9EDF900242DC9283950A6C` |
| 文件时间 | 2026-11-08T10:17:14.9536389Z（本地18:17:14 +08:00） |
| 环境 | Go1.27.3、Windows amd64/v1、CGO_ENABLED=1、trimpath、MyGo v0.8.3；PE32+ GUI subsystem2 |

[集成结果](artifacts/acceptance/mindmap-fields-20261108-181628/result.json)
保存实际stdout/stderr、退出码、命令和20个包状态；成功的vet/build日志
确实为空。包摘要和Architecture元数据曾从已有日志/机器信息校正，未重跑
命令；[独立证据核验](artifacts/acceptance/mindmap-fields-20261108-181628/evidence-validation.json)
9/9通过。文档侧另行读取验收副本核对大小、SHA-256和时间。

[隐藏启动记录](artifacts/acceptance/startup-smoke-mindmap-fields-20261108-181739/result.json)
按PID枚举到「轻笺 · 原生版」，Visible=false，WM_CLOSE退出0；无关更新
形状文件/目录哨兵内容保留，源/副本哈希一致，无残留所属应用进程。
这证明隐藏进程启动和关闭，不证明可见GPU绘制。本轮未采样进程资源，
下方17:35–17:42值仍专属16:45链接副本。

所属视图包的独立复现命令：

```powershell
.\native\scripts\go.ps1 -GoArguments @('test','./modules/graphview','-count=1','-timeout','120s')
.\native\scripts\go.ps1 -GoArguments @('vet','./modules/graphview')
```

证据见 [视图交接](handoffs/mindmap-fields-view.md)、
[格式交接](handoffs/mindmap-fields-format.md)、[应用交接](handoffs/mindmap-fields-app.md)
和 [doc-block 修正](handoffs/docblock-safety.md)。原生输入覆盖字段切换与
Compose、指针/Tab+Enter/Space、可选字段存在性、逐字保存/重开、单次
Flush的Undo/Redo、其他字段合并、源冲突恢复、只读与超限；还包括按下
后的源/节点/Buffer/草稿变化、取消及同步OpenReference→Release，不只
直接调用导航函数。draw.io原有标签编辑未改为导图字段模型。

200,000字节只读备注的fixture和首帧在测量外：旧路径五个未变化帧累计
1,036,200 B/201次分配/546.5 µs；最终输入版本缓存与去除重复摘要后为
9,736 B/182次分配/528.2 µs，FramePending=false。完整字段及基线仍保留，
实际修改/解析/序列化仍有线性成本。这是headless累计分配，不是进程内存
或CPU测量。532×285内容区域的长字段分段、链接与恢复控件通过真实滚动
可到达性检查；[四张组件图](artifacts/acceptance/mindmap-fields-view/)
和 [应用正常图](artifacts/acceptance/mindmap-fields-20261108-180356/mindmap-fields-app-normal-760x480-dark.png)、
[应用冲突图](artifacts/acceptance/mindmap-fields-20261108-180356/mindmap-fields-app-conflict-760x480-dark.png)
已检查。应用截图测试1.556s；输入内容由真实选择/复制核验，不能以
Tester.Texts的绘制标签清单替代独立Editor文本。

doc-block回归验证缺少分隔符/EOF/未关闭围栏不再崩溃，完整原文作为代码
显示并警告；关闭围栏只消费一次，合法或错误块之后的正文继续解析。
这没有增加嵌套图表或更广的Markdown语法。

## doc-block引用 · 历史验收：2026-11-08 19:20集成，根代理19:24验收

根代理已接受解析器、原生视图和真实应用聚焦门，并收到GLM最终
no-findings复核。最终来源身份narrow/clicked-row守卫聚焦测试1.120s、
vet通过；该数字是受影响的`TestReference`/守卫范围，不是完整
markdownview包耗时。完整集成中markdownview包4.281s、app包7.128s。

集成负责人随后对冻结源码执行一次完整序列：

```powershell
native\scripts\go.ps1 -GoArguments @('test','./...','-count=1','-timeout','300s')
native\scripts\go.ps1 -GoArguments @('vet','./...')
native\scripts\go.ps1 -GoArguments @('build','-trimpath','-buildvcs=false','-ldflags','-s -w -H=windowsgui','-o','bin\levnote-native.exe','./cmd/levnote')
```

全套测试退出0，命令墙钟11.613s；捕获的1,062字节stdout含20条包摘要，
全部为`ok`或`[no test files]`，stderr为实际捕获的0字节文件。全量vet
1.903s退出0，构建2.444s退出0；成功vet/build的stdout/stderr都是执行时
捕获的0字节流，不是事后占位。

| 构建项 | 本轮记录 |
|---|---|
| 文件 | [验收副本](artifacts/acceptance/docblock-reference-20261108-192000/levnote-native.exe)；源路径`native/bin/levnote-native.exe` |
| 字节数 | 13,298,176 B |
| SHA-256 | `DF911A7F2C37E5548CDC6BF6E92D0FE93CB45283277D010020C267095A3D5D82` |
| 文件时间 | 2026-11-08T11:20:17.6539304Z（本地19:20:17 +08:00） |
| PE/构建 | `0x8664`、`0x020B`、Windows GUI subsystem2；Go1.27.3、trimpath、CGO_ENABLED=1、MyGo v0.8.3 |

[聚合结果](artifacts/acceptance/docblock-reference-20261108-192000/result.json)
保存命令、退出码、包摘要、可执行/PE/Go元数据、源指纹、图片和smoke摘要；
[独立证据核验](artifacts/acceptance/docblock-reference-20261108-192000/evidence-validation.json)
12/12通过。首次validator调用因自身PowerShell表达式在给出结论前失败，失败
记录已保留；只修工具脚本，未重跑任何产品命令。
[根代理验收记录](artifacts/acceptance/docblock-reference-20261108-192000/root-acceptance.json)
固定了根代理读取的证据集合与接受时点。

隔离隐藏启动使用真实本地`mode: reference` fixture。按owned PID 35788
枚举到「轻笺 · 原生版」，Visible=false；WM_CLOSE退出0，3.689s内没有
强制清理或残留进程。更新形状文件/目录哨兵和fixture/target哈希保持不变，
源/副本哈希一致。记录见
[smoke记录](artifacts/acceptance/startup-smoke-docblock-reference-20261108-192100/result.json)。
这只证明隐藏进程启动/关闭，不证明可见GPU绘制。

本轮未采样进程资源，也未做benchmark。四张760×480明暗正常/拒绝应用图
已复制到验收产物；最终点击行守卫改变来源身份校验，不改布局。
根代理已读取聚合结果、原始包摘要、可执行元数据和smoke，并接受这个
13,298,176字节产物；18:17字段版13,274,624字节成为上一验收产物。
输入、限制和精确交接见
[应用交接](handoffs/docblock-reference-app.md)、
[最终视图交接](handoffs/docblock-reference-view.md)与
[集成交接](handoffs/docblock-reference-integration.md)。

## Markdown管道表第一切片 · 2026-11-08 20:49集成，根代理20:50验收

冻结产品源码的解析/导出、原生视图和真实注册适配器聚焦门通过，独立只读
复核没有发现可执行缺陷。随后执行一次全量test/vet、去符号GUI构建和隔离
隐藏启动关闭；根代理读取原始20包stdout、聚合结果、可执行元数据、
provenance、12/12独立核验与smoke后接受精确产物。19:20 doc-block本地引用版
转为上一验收产物。

完整序列：

```powershell
native\scripts\go.ps1 -GoArguments @('test','./...','-count=1','-timeout','300s')
native\scripts\go.ps1 -GoArguments @('vet','./...')
native\scripts\go.ps1 -GoArguments @('build','-trimpath','-buildvcs=false','-ldflags','-s -w -H=windowsgui','-o','bin\levnote-native.exe','./cmd/levnote')
```

全套测试退出0、命令墙钟10.453s，20/20包摘要为`ok`或`[no test files]`；
其中app7.700s、`core/plugins`4.541s、`modules/markdownview`3.910s、
`formats/markdown`2.288s。vet退出0、2.050s；构建退出0、2.441s。
六个stdout/stderr流均在执行时捕获，成功vet/build流按实际保持0字节。

格式解析/导出检查覆盖精确UTF-8字节范围、Unicode、缩进/空单元格、可选
外管道、单列、头/分隔零正文、CRLF、奇偶反斜杠、反引号内未转义管道、
块边界和段落中断；每个上限/上限+1、多小表累计预算、完整单块回退、
Partial导出拒绝、恶意HTML/链接/图片字面转义及损坏描述项均有断言。

```powershell
.\native\scripts\go.ps1 -GoArguments @('test','./formats/markdown','-count=1','-timeout=120s')
.\native\scripts\go.ps1 -GoArguments @('vet','./formats/markdown')
```

格式包0.634s通过，vet退出0。报告中的旧0.601s仅是先前API编译检查，
不计为本阶段验收。

原生视图的表格定向命令墙钟2062ms、Go包1.078s；完整markdownview包
2726ms/1.809s，vet退出0。实际输入覆盖指针、聚焦Enter/Space、取消/
滚动、待提交Compose、无关前插重定位、双表、只读、失败Flush保留恢复、
32 KiB超限双入口、远行虚拟化、闲置帧和Release。解析/视图组件另有
265×480堆叠与760×480网格、明暗和240-DIP冲突共六张图片，均在
[视图交接](handoffs/markdown-tables-view.md)列明并逐张检查。

真实注册Markdown适配器的命令与结果：

```powershell
$env:LEVNOTE_MARKDOWN_TABLES_SCREENSHOT_DIR =
  'F:\AIproject\LevNote\native\artifacts\acceptance\markdown-tables-app-20261108-200223'
.\native\scripts\go.ps1 -GoArguments @(
  'test','./app','-run','^TestMarkdownTable','-count=1','-timeout=120s','-v'
)
```

`PASS`，`ok levnote.local/native/app 1.646s`；`vet ./app`、受影响文件
`gofmt -d`和`git diff --check`均退出0。应用覆盖BOM+CRLF无改动保存/重开、
指针/Enter/Space整表编辑、Compose先Flush、将结构主动改坏后回退、保存/
模式/标签/关闭屏障、失败Flush保留、只读三列堆叠、无效与超限源码可见、
安全与回退HTML导出，以及不扰动表格前后普通段落。

应用产物位于
`artifacts/acceptance/markdown-tables-app-20261108-200223/`。表格进入
首预览块后，760×480明暗分屏图都断言并实际显示`编辑表格源码`和正文
`**字面**`在y451状态栏上方：

| 文件 | 字节数 | SHA-256 |
|---|---:|---|
| `focused-test.stdout.log` | 2,934 | `032D87E4F9784DD94405F7F0C41A0A0E42F19BA9CBDCCCE23B2DFAD1A2DAF834` |
| `focused-vet.stdout.log` | 0 | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |
| `markdown-tables-app-split-light-760x480.png` | 46,219 | `ACE45FD41128DBEB268257CD481D2A01A09CD2DF40824F8EE52BFAEF35C69B78` |
| `markdown-tables-app-split-dark-760x480.png` | 46,639 | `5321F8B3324A2A2A417F67B6E40C0EE5D9D0011BEAA875E48D72D43403B83ABD` |

本轮没有资源采样或benchmark，不把测试器累计分配解释为进程内存/CPU。
隔离隐藏启动使用真实Markdown表格fixture，owned PID2732、Visible=false；
`WM_CLOSE`后退出0，3.804s的捕获command内无强制清理或残留LevNote进程，
fixture/target、更新形状文件/目录哨兵和源/副本哈希保持不变。

当前已验收可执行文件：

| 构建项 | 本轮记录 |
|---|---|
| 文件 | [验收副本](artifacts/acceptance/markdown-tables-20261108-204724/levnote-native.exe)；源路径`native/bin/levnote-native.exe` |
| 字节数 | 13,384,192 B |
| SHA-256 | `7632B1C50D9536B25A12CF9FA7E288B62F903A3B4ABA9896EF8849356864BBCD` |
| 文件时间 | 2026-11-08T12:49:19.7290537Z（本地20:49:19 +08:00） |
| PE/构建 | `0x8664`、`0x020B`、Windows GUI subsystem2；Go1.27.3、trimpath、MyGo v0.8.3 |

[聚合结果](artifacts/acceptance/markdown-tables-20261108-204724/result.json)
的SHA-256为
`823205D739CDA036110ECADD1200BA8AD35F2449B68F8B9883CCAB265296466F`；
[独立核验](artifacts/acceptance/markdown-tables-20261108-204724/evidence-validation.json)
12/12通过；
[根代理验收](artifacts/acceptance/markdown-tables-20261108-204724/root-acceptance.json)
独立记录接受结果，未改写已接受的结果或核验快照。smoke记录见
[result.json](artifacts/acceptance/startup-smoke-markdown-tables-20261108-204724/result.json)。

精确范围、场景与保留的测试脚手架修正见
[格式交接](handoffs/markdown-tables-format.md)、
[视图交接](handoffs/markdown-tables-view.md)和
[应用交接](handoffs/markdown-tables-app.md)。

## Markdown表格单元格编辑 · 聚焦检查与已验收集成

第二切片使用真实注册Markdown适配器验证Header/Body单元格、BOM+CRLF、
宿主源码在输入期间不变、Finish/Save单次写回、独立Undo/Redo和精确保存/
重开。保存、HTML导出、源码/分屏切换、标签切换和关闭的屏障测试均在输入
前记录源码版本/历史；无效结构值阻止动作并保留原文、选区、历史和视图。
真实`打开未同步副本`会新增并激活一个含准确无效草稿的新标签。本轮还覆盖
已有普通Markdown Compose先Flush、显式放弃、无改动完成、只读拒绝和窄窗
明暗分屏图。

```powershell
.\native\scripts\go.ps1 -GoArguments @('test','./app','-run','^TestMarkdownCells','-count=1','-timeout','180s','-v')
.\native\scripts\go.ps1 -GoArguments @('vet','./app')
```

应用单元格定向组通过，`ok levnote.local/native/app 2.010s`；app vet无诊断。
这是一项真实注册适配器的聚焦门。根代理复核该聚焦证据、最终视图交接、
两张集成图和一次已授权失败记录后，批准修正过时的测试选择器并重新执行
完整序列。第二次全量`test ./...`以20/20包摘要通过（9.147s），全量vet
（1.941s）、去符号GUI构建（2.516s）、12/12证据核验和隔离隐藏启动关闭
均通过；根代理已接受13,425,152字节产物。

单元格草稿是有意的视图本地状态：自动会话快照只读取当前宿主文档，因此
输入中但尚未Finish/Save的值不会进入会话/崩溃恢复，界面与文档明确提示
用户先完成写回。正常Save/关闭屏障会先提交有效值；无效值仍阻塞并在同一
视图中保留修改、放弃和打开未同步副本。视图所有者最终检查通过后，应用
只重拍了`TestMarkdownCellsAppNarrowScreenshots`，PASS 0.10s、包1.636s。
两张实际760×480明暗图位于
[markdown-cells-20261108-213013](artifacts/acceptance/markdown-cells-20261108-213013/)：

这些聚焦图先于最终集成并保持不变；最终集成目录另行复制并记录了相同的
两张图片身份。

当前已验收可执行文件：

| 构建项 | 本轮记录 |
|---|---|
| 文件 | [验收副本](artifacts/acceptance/markdown-cells-20261108-213713/levnote-native.exe)；源路径`native/bin/levnote-native.exe` |
| 字节数 | 13,425,152 B |
| SHA-256 | `34F3F48C98300B053B0BA0E69BE9D9F89CDA15E17759E6A26AD817A1E9C1D14C` |
| 文件时间 | 2026-11-08T13:45:35.0697169Z（本地21:45:35 +08:00） |
| PE/构建 | `0x8664`、`0x020B`、Windows GUI subsystem2 |

[聚合结果](artifacts/acceptance/markdown-cells-20261108-213713/result.json)
的SHA-256为
`E3AD9163C07EBA90C6436069BBE751DAACEBB029E8EB8D8FFDBE4DF9112D9BF8`；
[独立核验](artifacts/acceptance/markdown-cells-20261108-213713/evidence-validation.json)
12/12通过；
[根代理验收](artifacts/acceptance/markdown-cells-20261108-213713/root-acceptance.json)
独立记录接受结果，未改写已接受的结果或核验快照。smoke记录见
[result.json](artifacts/acceptance/startup-smoke-markdown-cells-20261108-213713/result.json)，
owned PID25664、隐藏启动、`WM_CLOSE`后退出0，无强制清理或残留进程。

历史失败记录如实保留：21:39第一次全量序列只在测试阶段失败，原因是
过时的`RoleCell`测试选择器，不是产品缺陷；根代理审阅过当时的原始输出并
批准测试-only修正。该次序列的根级test/sequence流因复用目录被成功重跑
覆盖，未保留字节级副本；相关时间、字节数和结论只作为根代理观察记录在
[集成交接](handoffs/integration.md)，不能冒充原始证据。下一次任何集成
尝试必须先创建全新目录。

|---|---:|---|
| `markdown-cells-staged-light-760x480.png` | 45,262 | `4B7E44C1F007ED46DF4C8BE82D353051EA2D1081F3B2E8F2B2E3F667B47F9607` |
| `markdown-cells-refused-dark-760x480.png` | 50,747 | `D3D30CE6927132143A00505A73737C7E3E9CB2E7F58DCC37FF454465B3ECE461` |

本轮没有资源采样或benchmark，也不把测试器累计分配解释为进程内存/CPU。
隐藏启动关闭不是真实系统IME或GPU验收；暗色集成图显示恢复控件，表格正文
行可能位于760×480视口外。

## 隐藏启动进程资源 · 16:45链接副本，17:35–17:42采样

根代理接受以下有限证据及其失败边界。采样使用原封不动的16:45可执行
副本：13,252,608 B，SHA-256
`5BFB5A445CC619FEA9380F0363D3DF26EC4F3FF944C8549B1992F5B462EF3F10`。
它不含后续导图字段修正。全部运行是**隐藏、无输入的启动/恢复主进程**；
没有可见绘制、GPU、交互编辑或旧Tauri同负载比较结论。采样期间没有
产品编译/测试、强制GC或工作集裁剪。

每次窗口就绪后静置10秒，按约1秒间隔取10个观察值。表中MiB是原始
`samples.observations[].workingSet64/privateMemorySize64`的算术平均除以
1,048,576，显示两位小数；不是沿用脚本的四舍五入摘要。

| 场景 | n | 平均工作集 MiB | 平均private bytes MiB | 采样CPU增量 |
|---|---:|---:|---:|---:|
| 空白窗口 | 10 | 58.45 | 86.07 | 0 s |
| 纯文本 | 10 | 61.82 | 88.79 | 0 s |
| Markdown | 10 | 63.79 | 90.38 | 0 s |
| 思维导图 | 10 | 59.25 | 86.57 | 0 s |
| draw.io | 10 | 59.04 | 87.08 | 0 s |
| 顺序打开四文档 | 10 | 64.50 | 91.82 | 0 s |
| 17:40首次恢复Markdown | 10 | 70.38 | 96.68 | 0 s |

负载已用预编译的产品加载器验证：纯文本565,054 B/8,192行；Markdown
994,815 B（含BOM）、2,048块/512标题/512任务/512链接，Partial=false；
导图20,683 B/500节点；图纸39,444 B/1页/100节点/180边，并保留其两个
非绘制结构cell警告。四文档按这些路径顺序启动，末项为活动文档。具体
fixture哈希和实际结构见 [载荷验证](artifacts/acceptance/resource-followup-20261108-165230/fixture-validation.json)。

前六场景CPU窗口约10.1秒；空白同一PID还追加10次、间隔3秒的观察，
实际30.137秒，CPU增量0s，平均工作集58.46 MiB/private86.07 MiB。静止
观察到0不代表任意使用期间没有CPU工作。首个恢复窗口10.142秒，CPU增量
0s；它漏记逻辑处理器分母，所以只报告CPU秒数。同机矩阵记录16个逻辑
处理器，但不回填或改写第一次恢复的原始JSON。

**原始矩阵仍为FAIL。** 第七项在隐藏种子窗口关闭前等了20秒，没有观察
到session.json，故未完成原计划的自动持久化/恢复场景。随后WM_CLOSE
正常退出0并生成正确的382字节会话；17:40的授权续测复用同一app-data，
验证并恢复该Markdown预览标签，再采样和正常关闭，会话前后哈希相同。
这证明该范围内的正常关闭持久化与恢复，不证明自动隐藏存续期间写入；
会话缺失也不能推出未执行View或绘制帧。

第一次恢复已成功，17:42因迟到的分母元数据审查又运行了一次，属于
**非必要、未计划的补充重复**。其工作集71.73 MiB/private96.45 MiB、
10个观察值、10.182秒CPU增量0，记录16逻辑处理器；不把它当第八个独立
场景，不与17:40结果平均。此后没有追加运行。

保留的 [原始FAIL结果](artifacts/acceptance/resource-followup-20261108-165230/result.json)
为212,222 B、SHA-256 `6EA9BC9370318AFEB82CF406E2CDFC5B639F58F918C4E10A5261EFEF6C78935E`；
[首次恢复原始记录](artifacts/acceptance/resource-followup-20261108-165230/followup-clean-restore-attempt-01-no-logical-cpu/followup-clean-restore/clean-restore-followup.json)
为24,340 B、SHA-256 `1740A7D6548E42C96CA77D390F55C3DB8EC4AD7716B91A297BEE4FD0954601AE`；
[补充重复记录](artifacts/acceptance/resource-followup-20261108-165230/followup-clean-restore/clean-restore-followup.json)
为25,318 B、SHA-256 `ED7F414F4BEFB8C0BD8A6C54AC65EFB34029147E2B8A85B767DCDD95CF733C27`。
首轮预检因包装进程暂未读到ExitCode而停止、未启动产品的失败记录也保留。
方法、全部证据副本与独立审查见 [采样交接](handoffs/resource-followup-app.md)
和 [审查记录](handoffs/resource-followup-artifact-review.md)。

## 历史 Markdown 行内链接集成检查点 · 2026-11-08 16:45–16:48 +08:00

根代理已验收此阶段冻结源码的最终产物、窄窗图片和隐藏启动结果。全套
Go 测试通过，墙钟8.998s（app5.454s、core/plugins4.060s、markdownview2.586s）；
全套 vet2.452s 无诊断，发布构建3.066s通过。精确命令和执行时捕获的
stdout/stderr 见 [集成结果](artifacts/acceptance/markdown-links-20261108-164403/result.json)
与 [输出来源](artifacts/acceptance/markdown-links-20261108-164403/OUTPUT-PROVENANCE.txt)。
成功 vet/build 的零字节日志表示没有输出，并非事后补造的检查结果。

```powershell
.\native\scripts\go.ps1 -GoArguments @('test','./...','-count=1','-timeout','300s')
.\native\scripts\go.ps1 -GoArguments @('vet','./...')
.\native\scripts\go.ps1 -GoArguments @('build','-trimpath','-buildvcs=false','-ldflags','-s -w -H=windowsgui','-o','bin\levnote-native.exe','./cmd/levnote')
```

文档收口独立读取验收副本并核对以下大小、哈希和时间：

| 构建项 | 本检查点记录 |
|---|---|
| 文件 | `native/bin/levnote-native.exe`；[保留的验收副本](artifacts/acceptance/markdown-links-20261108-164403/levnote-native.exe) |
| 字节数 | 13,252,608 B（约12.64 MiB），未压缩的去符号 exe |
| SHA-256 | `5BFB5A445CC619FEA9380F0363D3DF26EC4F3FF944C8549B1992F5B462EF3F10` |
| 文件时间 | 2026-11-08T08:45:33.7128592Z（本地16:45:33 +08:00） |
| 环境 | Go1.27.3、windows/amd64、GOAMD64=v1、CGO_ENABLED=1、trimpath、MyGo v0.8.3；PE32+ GUI subsystem2 |

隔离副本隐藏启动，按 PID 枚举到标题「轻笺 · 原生版」，窗口保持不可见；
WM_CLOSE 被接受并以 exit0 退出。两个无关更新形状哨兵及其内容保留，
源/副本哈希相同，见 [启动结果](artifacts/acceptance/startup-smoke-markdown-links-20261108-164536/result.json)。
前两次启动脚本检查分别遇到 Windows PowerShell 5 的无 BOM UTF-8 解析、
子会话找不到 Get-FileHash 的问题；失败记录和修正说明完整保留，未重建
产品来掩盖它们。该证据证明隐藏进程启动/关闭，不证明可见 GPU 文本绘制。

该产物后续隐藏启动进程样本见本页上方17:35–17:42记录；下方12:55
工作集/private bytes/CPU仍只属于更早的资源版，不沿用旧值。

### 定向输入、资源与可见界面

集成前完整 Markdown 视图包测试2.050s通过、vet无诊断；应用真实适配器定向测试1.818s通过、
app vet无诊断。独立当前源码审查没有实质发现。精确入口与场景见
[视图交接](handoffs/markdown-links-view.md)、
[应用交接](handoffs/markdown-links-app.md) 与
[独立审查](handoffs/markdown-links-review.md)。

```powershell
.\native\scripts\go.ps1 -GoArguments @('test','./modules/markdownview','-count=1','-timeout','120s','-v')
.\native\scripts\go.ps1 -GoArguments @('vet','./modules/markdownview')
```

公开原生输入验证普通拖选/复制、双击、Ctrl/Cmd点击、Tab+Enter、Space
滚动、换行片段命中/空隙、取消/修饰键变化。原文身份用例包括实际按键
捕获后的前方重复链接插入、编辑/删除目标、围栏上下文、Restore/过期日志/
新Buffer、Partial/超限、组合输入先Flush及失败保留。同步打开文件触发
Release时，当前列表状态继续存活到帧结束；源文本/选区/撤销不因纯导航改变。

资源诊断确认固定工具包的下划线Decorate会重复解码所有span。700链接
单行旧路径五帧累计2,150,546,456B/1.0331694s；在每行≤16链接且≤64span
才启用控件的显示预算下，整段保留可读格式并提示源码入口，变为
3,935,736B/4.6967ms。1000任务链接行的远端视口只构造16行，五帧
98,256B/1.0583ms，FramePending=false。单个30KiB标签在265px宽度的
五帧为184,208B/2.119ms；16链接与512粗体片段的回退五帧为
3,935,648B/1.2539ms。fixture/首轮解析不在未变化帧测量内。这些是
headless累计分配和耗时，不是进程RSS/CPU或对旧版的等负载比较。

明暗/只读265px视图和密集回退图已检查。链接前景色明色#24675d、暗色
#7dd3c4，在背景/Surface上的最低对比度分别6.021:1和8.502:1；旧暗色
按钮Accent用于文本仅3.331:1。最新760×480应用图44,500B，来自
[16:44截图](artifacts/acceptance/markdown-links-focused-20261108-1644/markdown-links-wrapped-760x480-dark.png)，
只重跑截图测试1.300s，正常布局与前次已审阅图片逐字节相同。

## 历史 Markdown 任务方框集成检查点 · 2026-11-08 15:48 +08:00

当前源码已完成简单任务项的纯格式元数据/HTML 与原生复选控件。定向
格式测试0.521s、vet 无诊断；视图完整包测试1.846s、vet 无诊断，具体
命令和场景见 [格式交接](handoffs/markdown-tasks-format.md) 与
[视图交接](handoffs/markdown-tasks-view.md)。独立当前源码审查没有实质
发现；根代理已接受最终集成产物及窄窗/只读图片。

```powershell
.\native\scripts\go.ps1 -GoArguments @('test','./...','-count=1','-timeout','300s')
.\native\scripts\go.ps1 -GoArguments @('vet','./...')
.\native\scripts\go.ps1 -GoArguments @('build','-trimpath','-buildvcs=false','-ldflags','-s -w -H=windowsgui','-o','bin\levnote-native.exe','./cmd/levnote')
```

全套测试通过，墙钟6.399s（app4.699s、markdown1.929s、markdownview3.338s）；
全套 vet1.083s 无诊断，发布构建1.480s 通过。见
[精确结果](artifacts/acceptance/markdown-tasks-20261108-154805/result.json)。
文档收口独立读取并核对以下发布文件值：

| 构建项 | 该阶段记录 |
|---|---|
| 文件 | [保留的任务方框版](artifacts/acceptance/markdown-links-20261108-164403/previous-accepted-markdown-tasks.exe) |
| 字节数 | 13,227,008 B（约12.61 MiB），未压缩的去符号 exe |
| SHA-256 | `8D71EC6AEF8F0DAE91C125AB87F234B439FEBF111712CEFC79751692412175ED` |
| 文件时间 | 2026-11-08T07:48:42.2959080Z（本地15:48:42 +08:00） |
| 环境 | Go1.27.3、windows/amd64、GOAMD64=v1、CGO_ENABLED=1、trimpath、MyGo v0.8.3；PE32+ GUI subsystem2 |

隔离副本隐藏启动，PID 范围内枚举到标题「轻笺 · 原生版」，窗口仍不可见；
WM_CLOSE 后 exit0，两个无关更新形状哨兵及内容保留，副本哈希与发布文件
相同，见 [启动结果](artifacts/acceptance/startup-smoke-markdown-tasks-20261108-154842/result.json)。
没有新增进程内存或 CPU 测量；12:55 历史样本不代表此产物。

实际原生输入验证点击、六次连续空格、六次快速点击，精确一字符变更、
反向选区、独立 Undo/Redo、重复任务的远端点击及前方插入、已改变/删除
目标、过期日志、Restore、新 Buffer、未闭合围栏上下文、Partial/超限、
只读和外部事务拒绝。另一个块的 Compose 先同步再切换任务，两步分别
可撤销；同列表草稿替换或 Flush 冲突不误切换。干净字段仍打开时，任务
Undo/Redo 的当前勾选状态也会更新；待恢复的冲突草稿不被丢弃。

应用真实适配器任务组1.401s、窄窗截图1.351s、app vet 通过。覆盖
BOM/CRLF 的保存重开、源码/分屏一致性、只读创建、Compose/失败 Flush；
详见 [应用交接](handoffs/markdown-tasks-app.md)。265px 窄视图明暗/
只读矩阵和760×480 暗色应用图已做可见几何检查，根代理已审阅图片。
最终应用图为41,841B，SHA-256 为
`D9EE88C32A220CF730A4B8C458A508E2FC268B29BB101C5D75CA4FAEA3393143`，
保存在上述验收目录，任务内容未覆盖 y451 起的状态栏。

单次 headless 样本：1000任务、五个未变化帧1.0012ms、累计74,008B/
972次分配，构造17行、FramePending=false；fixture/首轮解析在测量外。
这不代表进程内存/CPU，也没有生成与旧 Tauri 的相同负载对照。

## 历史超限预览集成检查点 · 2026-11-08 15:12 +08:00

Markdown/导图/图纸在复制整篇 Buffer 前拒绝超过 8/16/32 MiB 的富视图
输入。所有者测试/vet 已通过，独立审查未发现实质缺陷；应用级窄窗检查
促成图形恢复按钮改为同排、错误摘要限制行数、属性区可滚动。最终全套
测试、vet、发布构建与隐藏启动检查通过。精确记录见
[集成交接](handoffs/integration.md) 和
[验收 result.json](artifacts/acceptance/oversize-preview-20261108-150938/result.json)。

```powershell
.\native\scripts\go.ps1 -GoArguments @('test','./...','-count=1','-timeout','300s')
.\native\scripts\go.ps1 -GoArguments @('vet','./...')
.\native\scripts\go.ps1 -GoArguments @('build','-trimpath','-buildvcs=false','-ldflags','-s -w -H=windowsgui','-o','bin\levnote-native.exe','./cmd/levnote')
```

全套测试 7.165s 墙钟时间（app 5.104s，所有包通过），vet 无诊断，发布
构建成功。文档收口独立核对以下文件值：

| 构建项 | 该阶段记录 |
|---|---|
| 文件 | `native/bin/levnote-native.exe` |
| 字节数 | 13,209,600 B（约 12.60 MiB），未压缩的去符号 exe |
| SHA-256 | `36CDC562D0E3C7F1793EDC935154B3AD7FF726716FCC409A736B04BD0FCAF3C6` |
| 文件时间 | 2026-11-08T07:12:01.8261018Z（本地 15:12:01 +08:00） |
| 环境 | Go 1.27.3、windows/amd64、GOAMD64=v1、CGO_ENABLED=1、trimpath、MyGo v0.8.3 |

隔离副本以隐藏窗口启动，按该进程 PID 枚举窗口找到「轻笺 · 原生版」；
窗口保持不可见，WM_CLOSE 后 exit 0，两个更新命名形状哨兵及内容完整
保留，副本哈希相同，见
[启动结果](artifacts/acceptance/startup-smoke-oversize-preview-20261108-151201/result.json)。
没有新增进程内存/CPU 测量，12:55 历史样本仍不代表本产物。

定向命令与全部场景见 [交接记录](handoffs/oversize-preview.md)。最后四包
测试为 markdownview 1.817s、graphview 3.149s、mindmap 1.026s、drawio
0.587s；随后 Markdown 描述项清理回归 1.381s，图形窄区修正回归 2.148s；
相应 vet 均无诊断。真实 Compose 测试覆盖 Markdown 范围安全写回与冲突
保留、图形拒绝同步/Release/恢复副本、无部分导出、源码 Undo、缩小恢复
以及同版本的新 Buffer 替换。图形两种格式的恢复控件在 532×285 内容区
内具有可见矩形。应用定向测试 2.545s、完整 app 4.179s、app vet 通过。
760×480 截图检查 1.384s，源码/分屏入口均可见，图形「重试写回」与
「恢复为新文档」文字矩形顶边 y403.38、高 13.97，在 y451 状态栏上方；
修正后的实际图片已经检查。应用验证还覆盖源码只读导航/可写输入、
失败导出不发布文件、模式切换失败保留草稿和恢复成新文档。

下表是单次 headless 累计分配样本，fixture 在测量外构造。对照只执行旧
路径必须进行的一次 Buffer.String，不代表完整旧解析器/帧时间。短时
测量有调度噪声，因此以分配量检查是否发生随文档大小增长的复制。

| 路径 | 累计字节 | 分配次数 |
|---|---:|---:|
| Markdown 8 MiB+1，仅整篇 String 对照 | 8,396,800 | 1 |
| Markdown 首次拒绝 | 0 | 0 |
| 导图 16 MiB+1，仅整篇 String 对照 | 16,785,408 | 1 |
| 导图首次拒绝 | 16 | 1 |
| 图纸 32 MiB+1，仅整篇 String 对照 | 33,562,624 | 1 |
| 图纸首次拒绝 | 16 | 1 |
| 每种格式重复 100 次告警/导出拒绝 | 1,600 | 100 |

单次过期 Markdown 块进入检查分配 0 B，图形过期变更拒绝每种 176 B，
图形 Compose 后拒绝 Flush 每种 96 B。草稿样本为短字段；独立字段文本
与基线按需保留，不宣称超大字段为零成本。这里没有进程内存、峰值、
持续编辑 CPU 或旧 Tauri 对照的新测量。

## 历史 Schema2 源码大纲集成检查点 · 2026-11-08 14:43 +08:00

字面行扫描器、配置加载器、源码提供者和应用接线合并后，全套测试通过，
全套 vet 无诊断，发布构建及隔离启动成功。独立代码审查未发现实质缺陷。
精确执行结果见 [集成记录](handoffs/integration.md) 和
[验收 result.json](artifacts/acceptance/source-plugin-20261108-144035/result.json)。

```powershell
.\native\scripts\go.ps1 test ./... -count=1 -timeout=300s
.\native\scripts\go.ps1 vet ./...
.\native\scripts\go.ps1 -GoArguments @('build','-trimpath','-buildvcs=false','-ldflags','-s -w -H=windowsgui','-o','bin\levnote-native.exe','./cmd/levnote')
```

全套运行中的包耗时包括 app 4.270s、core/plugins 2.963s、sourceoutline
2.655s、sourceview 2.741s；所有包通过，包含 dependency-patches。

| 构建项 | 该阶段记录 |
|---|---|
| 文件 | `native/bin/levnote-native.exe` |
| 字节数 | 13,201,920 B（约 12.59 MiB），未压缩的去符号 exe |
| SHA-256 | `4F9145A3E09AA2731F0FD365502CADE53E755467A376AC45BCAE610542848B8F` |
| 文件时间 | 2026-11-08T06:41:35.3481194Z（本地 14:41:35 +08:00） |
| 环境 | Go 1.27.3、windows/amd64、GOAMD64=v1、CGO_ENABLED=1、MyGo v0.8.3 |

文档收口独立核对大小、SHA-256 和时间。隔离副本显示「轻笺 · 原生版」，
接受正常关闭并以 exit 0 退出，两个更新命名形状的无关文件/目录哨兵及
内容均完整保留，副本哈希与构建一致，见
[启动结果](artifacts/acceptance/startup-smoke-source-plugin-20261108-144240/result.json)。
最初隐藏窗口启动未暴露 MainWindowHandle；该自有进程结束后，最终关闭
检查短暂显示普通窗口以发送 WM_CLOSE，没有把前次尝试记为成功。

定向应用回归覆盖真实配置安装、BOM/CRLF 字节保存、Compose 后 Flush/
重解、失败保留、只读导航、工厂失效、启停/规则刷新、内置优先级、源码
格式拒绝阅读/分屏/富导出、2000 项键盘定位与空集截断提示。schema1 及
无大纲规则的 schema2 均保留引号字符串/数字的基础高亮。

最终截图来自 `artifacts/acceptance/source-plugin-20261108-144035/`，单独
截图测试 1.608s：760×480 暗色 INI、大纲远端 Unicode/重复标题/CRLF
实际定位、INI schema2 与 TOML schema1 插件详情。空规则 schema2 由测试
覆盖，未出现在该插件面板 PNG 中。本阶段没有新增进程内存/CPU 样本；
下方历史数值不代表本次可执行文件。

### 扫描器与提供者的定向资源证据

```powershell
.\native\scripts\go.ps1 -GoArguments @('test','./formats/sourceoutline','./modules/sourceview','-count=1','-timeout=120s')
.\native\scripts\go.ps1 -GoArguments @('vet','./formats/sourceoutline','./modules/sourceview')
```

scanner 0.610s、provider 1.004s，vet 无诊断。覆盖 500 个独立简单参考实现
比较、逐字节 UTF-8 分片、CRLF/内部 CR、精确 2 MiB 无 LF 末行、预算跨代理
字符的半行拒绝、16 字节后缀与长空白、首次空标签匹配、读取错误/无进展、
碎片化缓冲、Compose 后重解、配置指纹变化和范围失效。

同一 Ryzen 7 6800H/windows amd64，`-benchtime=5x -benchmem`：

| 工作量 | 时间/op | B/op | allocs/op |
|---|---:|---:|---:|
| 2 MiB 不匹配物理行 | 17.628 ms | 4,896 | 2 |
| 2 MiB 匹配物理行 | 40.477 ms | 5,456 | 4 |
| 2000 字面行标题冷读取 | 2.102 ms | 927,416 | 10,021 |
| 2000 项缓存快照 | 0.01134 ms | 131,072 | 1 |
| 2000 标题、200 次分散编辑后冷读取 | 1.745 ms | 935,001 | 10,023 |
| 上述碎片化缓冲缓存快照 | 0.01084 ms | 131,072 | 1 |

小样本差值不能解释为碎片化更快。分配是累计量，不是进程内存。一次
2 MiB 截断的 Unicode 超长行使用 664 个有界读取窗口，不返回半行条目，
后续快照复用缓存。精确命令与边界见
[source-plugin-contract.md](handoffs/source-plugin-contract.md)。本阶段尚未
新增进程内存/CPU 样本。

## 历史大纲集成检查点 · 2026-11-08 14:03 +08:00

Markdown/导图大纲与应用导航合并后，全套测试通过（8.083s 墙钟时间，
包含 dependency-patches），全套 vet 无诊断，发布构建及隔离启动成功。
这是 schema2 前一个阶段的完整二进制记录，详见
[integration.md](handoffs/integration.md) 的 14:03 检查点。

```powershell
.\native\scripts\go.ps1 -GoArguments @('test','./...','-count=1','-timeout','300s')
.\native\scripts\go.ps1 -GoArguments @('vet','./...')
.\native\scripts\go.ps1 -GoArguments @('build','-trimpath','-buildvcs=false','-ldflags','-s -w -H=windowsgui','-o','bin\levnote-native.exe','./cmd/levnote')
```

| 构建项 | 该阶段记录 |
|---|---|
| 文件 | `native/bin/levnote-native.exe` |
| 字节数 | 13,143,040 B（约 12.53 MiB），未压缩的去符号 exe |
| SHA-256 | `D77BC3F45114957416BE83B361873AC3648708E23B308C3395A35D099FF15925` |
| 文件时间 | 2026-11-08T06:02:54.6755956Z（本地 14:02:54 +08:00） |
| 环境 | Go 1.27.3、windows/amd64、GOAMD64=v1、CGO_ENABLED=1、MyGo v0.8.3；PE32+ GUI 子系统 2 |

文档收口独立核对大小、SHA-256 和时间。隔离目录
`artifacts/acceptance/startup-smoke-outline-20261108-140315` 中的复制品显示
「轻笺 · 原生版」，接受正常关闭并以 exit 0 退出；两个更新命名形状的
无关哨兵及内容均完整保留。本轮没有重新测量进程工作集/private bytes/CPU，
下方 12:55–12:56 样本仅属于上一个资源检查点，不能移用为这个 exe 的数值。

定向回归覆盖真实 Markdown/导图提供者，含 Unicode/CRLF/重复标题、Flush
期间插入重复项、过期范围和重载拒绝、Compose 后导航、冲突草稿保留、
源码/阅读/分屏、隐藏节点一次展开与撤销、只读拒绝、未知字段、反向源码
选区跨展开/Undo/Redo、插件停用和生命周期失效。2000 个标题的末项能在
532×420 预览中出现，每帧构造少于 40 行。源码提取不挂载图形布局。

验收截图包括 Markdown 三模式、暗色窄分屏、2048 项截断提示、图纸不支持
状态、导图隐藏节点及只读 760×480；远端示例截图已实际定位第 90 章，
只读窄窗的标签名/关闭按钮/模式行可见。它们是 MyGo 测试器截图，不代替
物理键盘、系统 IME 候选窗或系统读屏验收。大纲开关当前只在运行期间保留；
选中高亮不连续跟随源码光标。

### 大纲提取的有界 headless 样本

Ryzen 7 6800H，Windows amd64，`-benchtime=5x -benchmem`。冷路径包含解析，
热路径为已缓存快照复制；分配是累计 B/op，不是进程内存或峰值。

| 负载 | 时间/op | B/op | allocs/op |
|---|---:|---:|---:|
| 8192 个重复 Unicode slug | 6.292 ms | 1,364,187 | 24,559 |
| 2000 个 Markdown 标题，冷提取 | 6.329 ms | 4,227,992 | 17,964 |
| 2000 个 Markdown 条目，缓存快照 | 0.0192 ms | 131,072 | 1 |
| 2000 个导图节点，冷提取（无布局） | 14.872 ms | 7,860,992 | 138,026 |
| 2000 个导图条目，缓存快照 | 0.0915 ms | 131,094 | 1 |

宿主只在文档/标签/规则变化时复制快照，未变化帧复用自己的数组。精确命令、
所有者测试和选区夹取边界见 [大纲交接](handoffs/outline-contract.md)。

## 历史资源集成检查点 · 2026-11-08 12:55–12:56 +08:00

编辑器、会话、视图与搜索/磁盘读取修正合并后，全套测试、静态检查和
构建通过。集成执行记录见 [integration.md](handoffs/integration.md) 的
12:55 检查点；搜索/磁盘定向证据见 [search.md](handoffs/search.md) 顶部
12:52 检查点。它们替代这些工作文件中保留的旧阻塞状态。

```powershell
.\native\scripts\go.ps1 -GoArguments @('test','./...','-count=1','-timeout','300s')
.\native\scripts\go.ps1 -GoArguments @('vet','./...')
.\native\scripts\go.ps1 -GoArguments @('test','./dependency-patches','-count=1','-v')
.\native\scripts\go.ps1 -GoArguments @('build','-trimpath','-buildvcs=false','-ldflags','-s -w -H=windowsgui','-o','bin\levnote-native.exe','./cmd/levnote')
```

全套测试通过；vet 无诊断；补丁单独验证 0.501s。全套运行中的包耗时包括
app 3.555s、plugins 4.313s、search 2.592s、session 2.700s、
markdownview 2.764s、graphview 3.138s、editor 3.172s。

| 构建项 | 记录 |
|---|---|
| 文件 | `native/bin/levnote-native.exe` |
| 字节数 | 13,084,160 B（约 12.48 MiB），未压缩的去符号 exe |
| SHA-256 | `8ADB375D3AE400B57AA07C73E441D4A13DACE7ADC7124F5F0F38DD457FDC98B9` |
| 文件时间 | 2026-11-08T04:55:06.1079484Z（本地 12:55:06 +08:00） |
| 构建环境 | Go 1.27.3、windows/amd64、GOAMD64=v1、CGO_ENABLED=1、MyGo v0.8.3 |

该检查点文档收口时独立读取文件大小、时间和 SHA-256，与上述记录一致。
隔离目录启动出现「轻笺 · 原生版」窗口，请求正常关闭后 exit 0，旁边
两个与更新器命名相似的无关文件/目录均保留。

该阶段 MyGo 测试器截图于 12:52 重新生成，fixture/screenshots 测试耗时
1.943s。760×480 的冲突草稿截图已验收，控件和预览不再覆盖底部状态栏；
1180×780 与窄窗口的树、搜索、编码和插件面板未发现新的布局阻塞。

## 历史原生进程资源样本（12:55 资源版）

两种场景都运行上一节13,084,160字节的资源版exe，不是本页最新产物。
窗口就绪后等待 10 秒，再按约 1 秒间隔
采样 10 次；只统计**主进程**，不包含整个系统、GPU 或共享服务的资源。
working set 与 private bytes 含义不同，应各自比较。机器有 16 个逻辑
处理器，CPU 同时列出占一核和占 16 核总容量的百分比。

| 场景 | 实际采样区间 | 工作集平均 / 峰值 / 最终（MiB） | Private bytes 平均 / 峰值 / 最终（MiB） | CPU 增量 | CPU 一核 / 全机容量 |
|---|---:|---|---|---:|---|
| 空白窗口，无命令行文件 | 10.179s | 59.25 / 59.25 / 59.25 | 86.50 / 86.50 / 86.50 | 0.0000s | 0.000% / 0.000% |
| 四个文档标签，图纸为当前标签 | 10.148s | 86.24 / 87.11 / 87.11 | 109.48 / 109.57 / 109.57 | 0.1562s | 1.540% / 0.096% |

四文档是 fixture 的 `source.go`（86 B）、`notes.md`（388 B）、
`ideas.mindmap`（336 B）、`flow.drawio`（976 B）。这是小型代表性文档的
打开后静止样本，不是四个重视图同时绘制，也不是大文档峰值压力测试。
两进程均正常关闭 exit 0。fixture、隔离数据目录与详细方法见
[resources.md](handoffs/resources.md) 顶部 12:56 检查点。

读数为单次实验，不代表稳态上界；没有同设备同内容的 Tauri 样本，因此
不据此作迁移前后的运行内存比较。

## 已完成的定向验证

资源修正后执行：

```powershell
.\native\scripts\go.ps1 -GoArguments @('test','./core/text','./core/session','./ui/editor','./modules/markdownview','./modules/graphview','./dependency-patches','-count=1')
.\native\scripts\go.ps1 -GoArguments @('vet','./core/text','./core/session','./ui/editor','./modules/markdownview','./modules/graphview')
```

依次通过，包耗时为 0.894s、1.181s、3.330s、1.599s、2.727s、1.764s；
vet 无诊断。用例包括范围失效、Undo/Restore、冲突草稿恢复、只读边界、
会话转义预算、字段组合输入、长行代理对边界、缓存释放和窄容器布局。

应用的三个真实适配器都有「组合输入后立即 Save / 切源码 / 切标签」用例：

```powershell
.\native\scripts\go.ps1 -GoArguments @('test','./app','-run','TestReal(Drawio|Markdown|Graph)PendingIMECompositionAcrossSaveModeAndTab','-count=1','-timeout','90s','-v')
```

该检查点三组各三个子例通过，包耗时 1.332s。它们使用实际视图和 MyGo
输入事件，断言保存后的内容及未知字段，而非仅调用格式层方法。它们仍是
测试器输入，不能代替 Windows 系统输入法候选窗的完整人工检查。

## 可见 UI 工作量样本

命令与详细日志见 [视图资源记录](handoffs/view-resources.md)。以下是缓存
预热后的 **连续五帧合计耗时及累计分配**，不含文件打开，不是单帧延迟，
也不是进程工作集或峰值内存。

| 测试器负载 | 五帧耗时 | 五帧累计分配 | 每帧构造的行/节点 |
|---|---:|---:|---|
| 2000 个 Markdown 标题 | 1.127 ms | 42,936 B | 12 行 |
| 1000 项列表 | 1.053 ms | 68,368 B | 18 行 |
| 2000 个导图节点，普通视口 | 3.710 ms | 26,832 B | 18 节点，0 次新标签排版 |
| 2000 个图纸节点，普通视口 | 1.008 ms | 14,952 B | 35 节点，0 次新标签排版 |
| 2000 个导图节点，适应全图 | 35.353 ms | 371,208 B | 106 节点，105 个极小标签隐藏 |
| 2000 个图纸节点，适应全图 | 2.010 ms | 364,840 B | 800 节点，800 个极小标签隐藏 |

表中 2000 节点导图的 106/105 计数已通过单独重跑
`TestGraphFrameResourceProbe/kind1-2000` 核实（0 次新排版，1 条缓存）。
该次只用于核对计数，未把它的不同计时与原同轮表格混合。

各视图静止后测试器的 `FramePending` 为 false，即视图没有自行请求持续
重画。这不等于整个原生进程或 GPU 的空闲 CPU 为零。导图适应全图时仍
遍历边并绘制相交连接线，表中保留其较高耗时。

## 超长输入与恢复预算

同一条 30,000 UTF-16 单位的 CJK/emoji 源码，旧的 32k 整行阈值在 800px
不换行冷帧中耗时 46.848s。改为普通行阈值 4096、长行 2048 单位分段后：

| 测试器视口 | 冷帧耗时 | 累计分配 |
|---|---:|---:|
| 800px，不换行 | 241.929 ms | 2,608,528 B |
| 800px，换行 | 489.725 ms | 2,175,904 B |
| 265px，不换行 | 214.642 ms | 1,645,032 B |
| 265px，换行 | 464.537 ms | 2,232,176 B |

这是有界冷启动排版样本，仍达数百毫秒，不是流畅输入或 16ms 内响应的
承诺。字段分段导航、完整内容、末尾编辑、撤销、焦点和组合输入另有回归。

会话测试使用构造阶段以外的分配量：拒绝 3 MiB NUL 草稿时，Save 分配
6,176 B，builder 分配 45,968 B，均在完整 JSON 编码/草稿复制前拒绝；
100,000 个标签的 1,300,042 B 文件读取样本耗时 15.548ms、分配
3,175,544 B，仅保留前 100 条并报告省略 99,900 条。完整原恢复文件在
失败写入后不变，详见 [会话记录](handoffs/session.md)。

## 仍需独立验证的事项

- 更长采样、真实编辑负载、多文档高峰和同负载 Tauri/原生对照尚未完成。
- 系统级网络观测、真实 Windows 输入法候选窗口、完整屏幕阅读器与其他
  操作系统验收未完成。
- 当前 Windows race 测试曾在运行器初始化时以 `0xc0000139` 退出，不能
  声称 `-race` 已通过。
- exe 字节数、压缩安装包大小、RSS/工作集和累计分配是不同指标。旧版
  3.2 MB Tauri 安装包保留在历史文档，不与本版 exe 作等口径对比。
