# .14 documentation wording correction

2026-11-09. Active for DeepSeek resource/delivery owner, docs only, no tests/
builds/launches. Keep .14 package and original delivery immutable. Root has read
all current added text; fix only these precise wording errors, then append a
short correction entry to find-keyboard-release14-docs-result.md and freeze.

1. CHANGELOG .14 second bullet currently ends `活动组合输入期间 Enter/Escape 动作
   和 Ctrl+G；` with no predicate. Add `暂不执行`.
2. native/ARCHITECTURE.md Find section says `导航、替换、面板切换与关闭动作暂不执行`.
   `面板切换` overstates protection: Plugins and tab focus transfers are allowed
   and explicitly tested. Narrow to Find navigation/replacement/open/close
   actions and Ctrl+G while active Find composition; acknowledge focus may leave
   and toolkit commits/reconciles.
3. native/README.md line near122 says actions `会等待输入确认`; no request queue
   exists. Say they temporarily refuse/do not execute and can be invoked again
   after input confirmation, not automatic deferred action.

No full link/test rerun needed because links unchanged. Confirm exact relevant
text and report changed doc identities only. All other .14 docs already meet
root acceptance scope.
