# draw.io 克隆批删缺陷修正范围（2026-11-17）

基线：f2c2ceb82acecb40fb0502c047e41ba12a12be47。接续原会话 session-f9a42190-927d-44fa-aad1-26d65e97dfb3；历史审阅仅保留，不覆盖。

## 输入
- drawio-clone-design-20261116.md：公开合同与测试缝。
- drawio-batch-spec-review-20261116.md：五项 P2。
- drawio-batch-standards-review-20261116.md：一项 P2。
- feature-parity-drawio-next-20261115.md：功能范围不变。
- native/AGENTS.md：本地 Windows 当前主线与六根验证约束。

## 确定公开测试缝
格式 CloneCells/Serialize/OriginalBytes/Dirty：验证原始 XML 保留、完整 raw 资格和失败原子。
graphview 画布输入、Buffer 历史及 Release：验证选择恢复、caret 成功收尾、历史预算与生命周期。
app 保存/undo/redo/reopen：沿用合同，不在此轮扩新功能；最终受影响全包检查。
不得放宽已有断言以通过测试；旧缺陷先保留实际 RED，再改实现。

## 文件所有权
格式子代理：native/formats/drawio/clone_cells.go、clone_cells_test.go，必要时新修正测试文件；不改 delete.go 或其他模块。
画布子代理：native/modules/graphview/adapter.go、drawio_clone.go、drawio_multi_clone_test.go，必要时新修正测试文件；不改 app。
根代理：计划、审阅、验收文档，后续独立验证。两个实现任务不追加委派。

## 判据
六项 P2 均有对应反例与行为修复；不扩到 group、剪贴板、跨页、任意 edge 克隆。
运行针对性 Gate，保留命令、真实输出、exit 和输入前后身份；使用新的 20261117 证据目录。
格式压缩/只读/未知子树与失败原子行为不回归。恢复记录有可见 entry/byte 上限并在 Release 清理引用。
完整六根与独立 Spec/Standards 仅在源码冻结后执行；不生成新包，不提交推送，除非根另行验收。
