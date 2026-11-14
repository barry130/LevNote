# LevNote · 本地原生源码交付回执与 Windows 后续门禁

2026-11-14；root（lead）有限源码交付记录。**源码双 main 同步完成；新应用交付仍 NO-GO。** 本文只补记后续事实，不改写原始报告、失败回执或已接受 `.29`。

## 任务状态（type:state）

源码提交为 `75a5c20ea42d614dc89255ca150a47b35c3ff646`，父提交 `8d4f43239dc30bfeb083dafeec7a02ed0f2da2d3`。实际 commit exit0；95 个精确审阅文件、8,958 insertions / 173 deletions：50 个 live 实现/测试变化（22 code + 28 tests）、必要说明和历史 lineage、两组各九个 **inactive** 准备文件等。没有 broad add、force-add、force-push、新分支或 CI 激活。

根复核暂存及提交树的 95 个 canonical Git OID，并通过 `git cat-file --batch` 核对全部 1,211 个冻结源码成员的完整 bytes/SHA 与磁盘。冻结聚合为 `874bc1fd23a3d02e432c9c6305842ee540afdca1cdad23692905da80d01b2bdf`。这不是逐行语义审阅全树的声明。所有 handoff 保持原字节；两份 README 只依既有 Git 属性从 CRLF 规范化为 LF。

本次原有 534 个排除路径全部保留、未暂存、未删除；包括三组历史证据、旧 author/transport 文件和三个 remote helper。`.localtools/**` 不入库。推送认证失败后 Git 额外列出 `CON` 保留设备名，仅记录 metadata，未打开/哈希/删除/暂存它。受跟踪文件及暂存区在源码提交和远端核验后均干净；不称整个磁盘无 untracked 内容。

## 目标与实际同步（type:goal）

CNB 与 GitHub 均执行非 force `refs/heads/main:refs/heads/main` 推送。首轮两端 exit128，第二轮各 exit0；随后一次只读远端核验实际 exit0，两个 `refs/heads/main` 均为上述完整源码提交，与本地一致：

- [CNB 源码提交](<https://cnb.cool/canace/LevNote/-/commit/75a5c20ea42d614dc89255ca150a47b35c3ff646>)。
- [GitHub 源码提交](<https://github.com/barry130/LevNote/commit/75a5c20ea42d614dc89255ca150a47b35c3ff646>)。

已存[源码同步总回执](<F:/AIproject/LevNote/.localtools/root-source-sync-20261114-02/source-sync-receipt.json>)：5,934 bytes，SHA256 `62b099be9d2409cfac6967a6e48dba5f17620892d4795035fa685449323c14aa`；状态 `SOURCE_COMMIT_DUAL_MAIN_SYNCHRONIZED_VERIFIED_APPLICATION_NO_GO`。包含 22 个原始成员的路径/bytes/SHA、实际 push01/02 exit 和远端读回身份。

[提交树独审](<F:/AIproject/LevNote/.localtools/root-source-sync-20261114-02/committed-verification.json>) SHA256 `32eb88938cc20b8d47896e803869a30e6420b005ba86fc730fae928aabc6065c`；[精确暂存独审](<F:/AIproject/LevNote/.localtools/root-source-sync-20261114-02/staged-verification.json>) SHA256 `de5084cbe443414268932945fc8e7f49100cc4a9f8ce808ef72053166d3098a2`。这些是本机证据指针，不是提交到远端的原始证据包。

## 已试方案与失败原因（type:dead-end）

- 准备脚本01误用 manifest key `pendingNames` → `KeyError` / exit1；在任何暂存前失败。原回执保留；新02仅纠正为实际 `pending`、独立保存 `stageExit` 后完成。不是产品 RED，也未通过更改产品/测试掩盖失败。
- 首轮双 push → `failed to execute prompt script (exit code 1)` / `could not read Username` / exit128；读回两端仍为父提交，未误判成功。当前 Niubash Git 缺 helper；只读系统安装元数据定位既有 `E:/Git`（Git2.53.0.2），既有 GCM 版本实际 exit0 为2.7.3。
- 第二轮只在各次 Git 命令传入 `credential.helper=E:/Git/mingw64/bin/git-credential-manager.exe` 与 `GCM_INTERACTIVE=never` / `GIT_TERMINAL_PROMPT=0`；GitHub另传既有 `socks5h://127.0.0.1:10808`。没有改变 PATH 或 Git配置，没有发起交互登录或清除凭据，没有 `credential reject`，未打印凭据。
- 原 formatting / build guard / fresh packaging 失败仍保留：8项格式债务；官方ACL诊断 `nextAction=stop` / zero changes；Junction拒绝导致0 guard case；packager line24证据目录 `Access denied` / exit1，在候选创建和GUI PE build前失败。未换路径/预建目录/换shell/ACL/owner/UAC/elevation绕过，也未重复这些已关闭的失败通道。

## 进度与下一步（type:progress）

[源码收敛摘要](<F:/AIproject/LevNote/native/handoffs/autonomous-local-source-convergence-root-20261114.md>)及两轴补篇原范围保持：一次六产品根 test 为3,019 PASS（1,181 parent + 1,838 nested）、0 FAIL、49 SKIP（47 parent + 2 nested），26测试包PASS + 1host无测试，wall45.295191s；vet/build各exit0。提交树完整匹配该冻结源码，但没有为文档提交重新执行测试。49 SKIP、原跨秒 snapshot 相等断言风险和 held 格式债务没有改成 PASS。

**剩余平台门禁按依赖顺序，不自动激活：**

1. 先由宿主/权限策略恢复原路径的合法执行能力，并确认物理 Computer Use native pipe 可用。当前 fullaccess 不等于 Windows token/ACL 已改变；原官方 `stop` 条件未解除前，不重复 format、guard、package，不以新路径或替代shell规避。
2. 只有前置条件明确改变并获对应新执行授权后，按原失败脚本和全新 create-only 回执重开有限门禁：完整只读 format 核验与获准格式修正、原 build guard case、原 packager五GUI PE（host + 四fixed workers）及完整资产/许可/manifest。工具链仍Go1.27.3/offline vendor/六产品模式，不跑含handoff的整库`./...`。
3. 新候选经独立输入契约审阅后，执行四worker真实请求/拒绝/超时/取消/poison/清理与实际owned Job证据。Markdown/mindmap export、draw.io managed已有各自opt-in来源；mindmap SVG/readonly scene仍需独立reviewed route plan。测试器须在单独disposable bundle，不作为candidate的第六个PE。
4. 仅在对应fresh candidate接受输入后激活[owned-host准备](<F:/AIproject/LevNote/native/handoffs/autonomous-local-windows-delivery-20261113/README.md>)：suspended CreateProcess→非继承kill-on-close Job→membership/image/start核验→Resume；clean ordered two-tab session、恰好一个owned WM_CLOSE、新close publication、exit0、Job total1/active0与输入前后完整inventory。pre-close失败为FAIL/零WM_CLOSE，cleanup须join。
5. 实测Windows键鼠/IME/系统剪贴板/选择器/视觉、第二次启动/会话恢复与明确资源负载。Tester、单项alloc、PE大小、旧`.29`、源码PASS均不替代。最后审阅文档/包/完整接受，再决定恢复native CI；[禁用CI准备](<F:/AIproject/LevNote/native/handoffs/autonomous-native-ci-preparation-20261114/PREPARATION-RESULT.md>)的parser/literal checks不授权runtime或activation。

本文初写时没有可替代`.29`的新安装包；新`.31`候选及原打包证据目录未产生。源码阶段已交付，整体应用目标未完成，平台通道保持held。本文为源码提交之后的文档补记，不属于旧1,239-member shipping输入原字节集合。

## 同日继续请求后的执行条件重核

后续只读 token 观察实际为 Medium `S-1-16-8192`，与上述历史 Low 条件不同。原入口在全新 create-only 调用者回执下各有限执行一次：[重核记录](<F:/AIproject/LevNote/native/handoffs/autonomous-local-windows-medium-reentry-20261114.md>)记录真实 formatter exit2/8 Access denied/0源码改变、guard line20 Junction exit1/0cases、packager line24 mkdir exit1/候选`.32`与脚本evidence02 absent/0新PE或host/workerlaunch。没有修改ACL/owner/UAC，未换writer/shell/output规避失败。原1,211-member源码保持冻结，旧失败/准备/包不改写，三通道继续held，不用Medium元数据推断写入成功。

同轮独立源码核对纠正覆盖判断：mindmap SVG 和 readonly scene **均已有** `LEVNOTE_WORKER_BUNDLE_ACCEPTANCE=1` 的真实 bundle host-path opt-in测试；前有限suite均SKIP，当前缺口是新候选实际执行及更强进程/异常清理接受，不是缺测试。Task27独立计划另列实际入口、oracle和观察边界；没有新造scene测试或激活Task16/20准备。当前仍应用NO-GO、CIinactive、`.29`latestacceptedrunnable。
