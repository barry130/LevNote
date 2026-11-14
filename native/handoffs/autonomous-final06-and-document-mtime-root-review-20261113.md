# FINAL06 及公开 mtime-only 保存兼容性 · Root 限定复核

日期：2026-11-13。普通开发基线 `8897244ffa4f00f5ec51f95545e7a3ca27d37d97`。
本记录是实际源码/原始证据独立审阅，不是正式两轴Standards/Spec、完整App/core验证、Windows或整应用交付接受。所有产品命令由授权box唯一owner native-reliability执行；Root没有产品执行或语义源码改动。

## FINAL06：同一原失败单例与原25项后继PASS

冻结05的原25项Go31061/1、wrapper2、24PASS/1FAIL（nested20PASS/2FAIL）保持，原最低窗口light/dark源码输入0×0失败不抹除。
06只把[结果面板149行](</home/user/levnote-dev-experience-20261112-01/revision-06-source/native/app/workspace.go#L143-L154>) `.Shrink(0)` 改为 `.Shrink(1).MinHeight(110)`；未改230/height-260偏好公式、110下限、Find controls、source min80、capture Keys、worker bounded排序、任何oracle或断言。
Root实际manifest/patch/READY与九actual05父源的独核见[前序记录](<F:/AIproject/LevNote/native/handoffs/autonomous-final05-and-connector-root-review-20261113.md#L57-L63>)。作者[今日恢复审查](</home/user/levnote-dev-experience-20261112-01/coordination/resume-06-input-review-20261113-01.json>)5490B/SHA `c3aef40fb2155996f61343f10e7c709deeb3fea7df262157b131679e8fac3206`记录5967源/80238567B与533tests03/05全保留；Root读全文108行，不把作者遍历说成Root5967复遍历。

| 调用 | Go PID=PGID | Go / wrapper退出 | UTC launched→ended | 覆盖 |
|---|---:|---:|---|---|
| [focused06](</home/user/levnote-dev-reliability-20261112-01/runs/experience-list-final06-focused-01/evidence/result.json>) |2058|0 / 0|02:21:04.522909→02:21:43.583137|原最低窗口single top PASS，light/dark2 nested PASS|
| [affected06](</home/user/levnote-dev-reliability-20261112-01/runs/experience-list-final06-affected-01/evidence/result.json>) |2782|0 / 0|02:24:51.043734→02:25:17.577309|原25 top全started、25PASS、22 nestedPASS|

Root分别完整实读raw10/138行、numeric64行、result142行。
两个独立只读Node fs/crypto/assert审查均exit0（focused与mtime一起2361ms；affected806ms），不启动产品/helper、不写文件。全部1194 actual input/native和run/source文件各自完整长度/SHA与expected相等；所有三个清单roots/files逐项与expected一致，1194文件/17000651B。affected实际argv逐字等于Go31061原25；actual source逐字等于focused06；相对05只workspace15B一行，全部测试原样。
两次timed_out=false、owned_cleanup=[]、remaining=false，stderr零，无panic/skip/未达分支；Go1.27.3/linux-amd64/vendor/offline/CGO0/isolated HOME/XDG/tmp。

| Primary | focused06 bytes / SHA256 | affected06 bytes / SHA256 |
|---|---|---|
| numeric |2714 / 36225dfd49227d1fcad2905f699533a581b7b84fdaf22b7c074dfd5d69dce22c|4003 / 59eadfbe643c71d45048ce8a78a918799fceccf18e980c8a1e900c272f2def31|
| result |5816 / 68c36c8d02b4c4915680f652b2d4e68a72092d89ee57f3f21047eeada23aa6a7|8388 / 571a8983002ee89a41203ddb34ac8af3ec17a7c476ac9472f516eb380cef9619|
| stdout |939 / c5f67322fec093f1ce8cb6ca9a602e7ba220997d9126bbf1ff249f614e6b8bf0|11812 / 5ad7121a72fca05ec734e437fe2fdfc02c52b104a1affbcb111c398d790c20e1|
| expected |202703 / ef86a56bd0d035060a9ff403df4e0afc29c6757b2fe3244b62cbbe275a44a73b|202704 / 9a0580ed2abcd5420c18c97cba116eb49c2aa375cfca9812549f3a69adf6b4a9|
| each of three maps |199696 / b16c7cfa0e8ddfd29f77104b744cc6bb58fd89b69330e1fc59bd5016a3473efb|199696 / b16c7cfa0e8ddfd29f77104b744cc6bb58fd89b69330e1fc59bd5016a3473efb|

**视觉证据限制**：原public Tester在t.TempDir记录PNG路径，正常cleanup删除；Root实际查为ENOENT，没有持久图可审。原single test的source≥200×64及三个action分支断言通过，不声称source实测113/80或PNG肉眼审图。不得为截图重跑已闭合标签。
**调用前故障**：affected的外部metadata生成器曾以固定argv索引发生IndexError；soleowner记录在任何run-root/intent/PID创建前、检查run-root absent，改用 `argv.index('-run')+1` 后仅实际执行Go2782一次。这是prelaunch metadata故障，不虚称产品重跑，也不覆盖旧结果。

## mtime-only：真实公开Open/Check/Save兼容性阳性

Root完整读[公开oracle](</home/user/levnote-dev-reliability-20261112-01/runs/document-public-mtime-positive-01/source/native/core/document/fingerprint_public_test.go#L1-L80>)：external `document_test`，2957B/SHA `3c94db4b1ac0604f03ebb3cc26fdf62e661dadfd266ad74b06d6d9fe587955d5`。
独立literal12B为UTF8 BOM+“甲🙂\r\n”，expected为BOM+“乙🙂\r\n”；实际Open后同一file Chtimes从time.Unix(1600000000,0)到1600086400，Stat/SameFile/size/mode/mtime及ReadFile literal证明仅时间戳变化。
公开CheckExternalChange返回false；untouched Save不替换inode、不抹touched时间；公开ReplaceUTF16甲→乙变dirty，Save exact encoded bytes/BOM/CRLF及metadata/clean/新baseline Checkfalse均通过。无私有disk注入、I/O hook、fake FileInfo、sleep/计时或生成expected。
这是**兼容性positive**，不是伪造RED、digest冲突拒绝、已发布验证失败/重试、系统CAS或完整core/document接受。

[原调用结果](</home/user/levnote-dev-reliability-20261112-01/runs/document-public-mtime-positive-01/evidence/result.json>)：Go/PGID1757、Go0/wrapper0，UTC 02:04:08.114849→02:04:13.670453，wrapper180s/Go120s，raw4行PASS/test0.02s/pkg0.029s；无timeout/cleanup/remaining，stderr零。
Root完整实读raw4、numeric64、result142；只读script0/2361ms核完整primary、全部1193实际input+run文件16977844B、roots与三maps，全部原GREEN02的1192inputs原样，仅新增公开oracle。
[事后收取索引](</home/user/levnote-dev-reliability-20261112-01/coordination/document-public-mtime-positive-01-receipt.json>)1884B/SHA `546b8987702b89b45e213efff77ca0b80bbf6a5e10ae7bc33d47dd9d5dde0d26`只指向已有actual，不假造额外request/authorization/产品receipt。

| Primary | bytes | SHA256 |
|---|---:|---|
| numeric |2725|b21d70bae2717605345092c90e411638bc3a01a07e35b143249500faa0e8f5ff|
| result |5841|44fd49377477f88f427360c9bcfc21a3394bad456ee3e057bebe075b65aa9444|
| stdout |186|e5127baedbc0e65268bb742230a8fa47a6f79cccbe7b0e2efa34eca52541e5bb|
| expected |201480|bfa22cc9d13bf66f9aca1974a699ded4846e46b81be1d27269433f14e28158b0|
| each of three maps |199579|3ce7b33473c52e5bb9d051c459d28a4d7a22b002bc66e85679e9bb8af0de0d8f|

## 接续边界

Task6原25Linux有限回归已闭合；新作者报告、九源smallopaque精确镜像待Root收取后关闭有限任务，再启用同owner Task10 Appissues/settings/historynotice。Task9 Markdown仍待明确owner，不通过此记录自动授权实现。
Task8新增[公开Undo/Release oracle](</home/user/levnote-dev-drawio-20261112-01/source/native/modules/graphview/drawio_delete_test.go#L157-L264>)15445B/SHA `57ae1fb006fc66b6ae77173ea60f74667a55f8897386524ffec239de3dc13a2f`，Root已读108行并readonly核原9041prefix+五源exactconnectorGREEN。普通KeyZ触发Changed#2后真实Flush/Release-return/activeView替换再public写旧Buffer；只计算Release返回且switch后的旧Report，wrapper不draw released view，无nested Tester input。实际RED/GREEN另记，不提前当PASS。
Task11下一为同samefile/same-size改digest并恢复mtime的公开拒绝/Dirty/Undo/Redo/retry单positive，然后合法真实已发布验证失败边界与最终完整core tests/vet；内部byte-work/nil-disk必要guard不冒称公共失败证明。
整体integration、正式独立parallel Standards/Spec、真实platform/GUI/IME/PE/C:config/WM_CLOSE/Job、package/docs/CI恢复与双main交付仍待；.29是latestacceptedrunnable，.30未激活，整体NO-GO。
