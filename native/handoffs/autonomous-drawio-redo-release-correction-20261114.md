# Task21 · draw.io Redo 同步 Release 屏障

## 任务状态

type:state
- 有限修正与源码冻结完成；生产仅 [input.go:270–274](<F:/AIproject/LevNote/native/modules/graphview/input.go#L270-L274>) 五行，新增外包公开 [drawio_redo_release_test.go](<F:/AIproject/LevNote/native/modules/graphview/drawio_redo_release_test.go>)。
- CtrlY/nativeRedo 公开 RED 两子 FAIL/exit1/1.283s → 同测试 GREEN 两子 PASS/exit0/1.593s；最终受影响 focused union exit0/2.556s，19主 PASS+30子 PASS/0FAIL/1原opt-in SKIP。
- exact2 Go1.27.3 `gofmt -l` 空输出/exit0；16份原测试 SHA 16/16 OK；无运行中 job、无再委派、未改旧证据或根 PLAN。

## 目标

type:goal
- 修复 Standards S1 保留基线缺陷：draw.io Redo Changed 同步 Flush/Release 后不得 `ensureSynced` 复活已释放缓存、解析 retired 内容并向旧 Host.Report 上报，且不得抢替换视图焦点。
- 共用相邻 Undo 已有即时回执条件：Buffer/releaseEpoch/pageEpoch/pageIndex/readonly/group/version/selection；不泛化 mindmap、不新增抽象或削弱原断言。
- 测试只用 public Adapter.Create、ui.NewTester genuine canvas click→Delete→Undo→CtrlY/native redo；两键间不 refocus，不以 private redo 调用冒充用户证据。

## 已试方案与失败原因

type:dead-end
- 旧 Redo 仅在 `ensureSynced` 后保护 focus→合法 Changed 回调 Flush/Release→mount new→oldBuffer 坏 XML 后仍 obsoleteReports=1，`invalid draw.io XML ... unexpected EOF`。
- 最小前置五行 draw.io receipt barrier→obsoleteReports=0；两种公开路由 oldChanged3/newChanged0、oldV4 sel{3,13} historyUndo2/Redo0/810B、newV0 sel{5,19} history0、两份字节 SHA 完全不变，newfocus 与 FramePending=false 保持。
- 该轮无 compile/fixture 失败；MarkersInspectorImagesAndRecoveryReachable 仅缺原截图 opt-in 环境变量而 SKIP，未修改/重跑截图或声称已覆盖。

## 进度与下一步

type:progress
- 完整命令/数值退出码/raw/SHA/边界见 [最终收据](<F:/AIproject/LevNote/.localtools/graph-redo-20261114/10-final-receipt.md>)；[final pins](<F:/AIproject/LevNote/.localtools/graph-redo-20261114/07-final-pins.sha256>) 钉住 source 与 RED/GREEN/final 输出。
- final input 7781B SHA `4f696abfcfbc5d8c2b9fa4812441f38a1a794ee1532e8157d3f8d7ad5eff008d`；newtest 8151B SHA `c4db22dec9cc7b617e769be551107350b80ca0494271aaeb0cfd84bcb8ac8f18`。
- root 下一步先实读 input 五行与完整新test、核收据/raw/finalpins，再更新共享 PLAN 并进行根整合门禁。该轮没有 fullgraph/fullproduct/vet/build/Git/ACL/PE/GUI/worker，public synthetic native命令不是物理OS菜单/IME/新包接受。
