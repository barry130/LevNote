# Task28 · Find 替换 Flush 后只读合同加固

## 任务状态
(type:state) 实现及一次最终 Appfull 已完成，待 lead 固定基点 f2a5756038a3fa255eacf09a57840dbb5fd38bc6 的独立审阅；P2 合同加固，不是已证实的普通用户 P1 生产并发/磁盘损坏链。
[最小守卫](<F:/AIproject/LevNote/native/app/state.go#L302-L305>) 仅新增 4 行，SHA256 b99b611c13e291c85c2e105a2e8371383d201449c01e7ba9edd7ab9f37f94475；[四个新 public tests](<F:/AIproject/LevNote/native/app/replace_readonly_flush_test.go>) 8944B/SHA256 b76d7d5d429c6bb93c862ce02274074b0a16545e689cc53da86f92ad3d3ef722。
[完整证据目录](<F:/AIproject/LevNote/.localtools/find-readonly-flush-20261114/>) 独立 create-only；原测试、8 个历史 format-denied 源码、.29 和旧失败证据保持原字节；无新 vet/build/package/host/Git/network。

## 目标
(type:goal) `ui.NewTester(a.View,1180,780)` → 替换字段 Enter / 替换当前 / 全部替换，公开 `editor.View` + `editor.Flusher` callback 同步改变同一 `Doc.ReadOnly`；fixture 只设置 Doc，只读权威不依赖 Editor 同步。
写权限检查位于成功 Flush 和目标身份重检后，先于 Unmark、搜索写入、Release、后续 Find 和 noteEdit；保留合法 Flush 编辑、source/version、逆向选区、Undo/Redo、pending input、live target 和 Find 状态/可见结果。
Enter 首例严格 RED→最小 GREEN；其余逐个追加 all-button 合法 Flush 编辑保留、writable-all 成功/分离历史、current-button pending MarkedRange/InGroup 保留，均合同扩展首跑 GREEN。

## 已试方案与失败原因
(type:dead-end) 02-red-current：caller 的预设 Git Bash 路径缺失→`FileNotFoundError [WinError 2]`，无 gate/Go PID、非测试 RED；原失败留存，仅 runner 改用 `command -v bash` 找到的既有 MSYS 可执行文件。
03-red-current：`bash .localtools/gate.sh test -json -count=1 -timeout=300s -run '^TestReplaceReadOnlyAfterFlushCurrentEnter$' ./app/`→exit1/1 FAIL，实际 source=`match needle!`、v5、selection6..12、Undo2/Redo0；[原流](<F:/AIproject/LevNote/.localtools/find-readonly-flush-20261114/03-red-current/stdout.jsonl#L4>)。
05-green-current：相同命令→exit0/1 PASS，App2.309s；07-all-synchronized、09-writable-all、11-current-pending 各 exit0/1 PASS，App2.647/1.995/1.912s；只有首 Enter 有实际 RED。
13-focused-find-replace：同 gate 的 `-run '^Test(Find|Replace)'`→exit0，parents47 PASS/1既有SKIP（all nodes90 PASS/1 SKIP），App2.890s/wall4.4979459s；无断言弱化或新 skip。

## 进度与下一步
(type:progress) 14-app-full：`bash .localtools/gate.sh test -json -count=1 -timeout=300s ./app/` 唯一最终 Appfull→exit0，parents391 PASS/28既有SKIP，child nodes428 PASS/0 SKIP，all nodes819 PASS/28 SKIP；App20.361s/wall21.8117779s，前后当前 scoped inventory 相同 SHA256 9db98042a30780d78d38dcc4929675be05c705901e38dc5a110246cabdf8320a。
[Appfull 原流](<F:/AIproject/LevNote/.localtools/find-readonly-flush-20261114/14-app-full/stdout.jsonl>) 800028B/SHA256 751740af6e2d7b0abf54d215b3874d87971a70aefbe3fdbedd8f96de6f3ce06a；[回执](<F:/AIproject/LevNote/.localtools/find-readonly-flush-20261114/14-app-full/receipt.json>) SHA256 86341c0b02a33dae92f90bde1692e541d7f60889ae3d8b0cc99621ee3fd50848；28 opt-in SKIP 不计作 Windows/IME/PE/WM_CLOSE 或物理 GUI 验收。
全部 labels：00-baseline、01-format-red、02-red-current（caller fault）、03-red-current、04-format-green、05-green-current、06-format-all、07-all-synchronized、08-format-writable、09-writable-all、10-format-pending、11-current-pending、12-toolchain-version（实际 Go1.27.3 windows/amd64）、13-focused-find-replace、14-app-full、15-final-summary（纯 pins/派生汇总，无产品 gate）。
下一步：lead 读取 [final-summary](<F:/AIproject/LevNote/.localtools/find-readonly-flush-20261114/15-final-summary/summary.json>)，建立新 scope freeze 后双轴只读审阅；执行方冻结 source/test/handoff，应用整体与平台验收仍待 lead 的独立门禁。
