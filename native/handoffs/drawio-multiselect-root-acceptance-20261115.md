# draw.io 多选批次根验收 · 2026-11-15/16

## 范围与结论

实现 draw.io 编辑视图的 Ctrl 多选、六向对齐（左/水平居中/右/顶/垂直居中/底）、Shift 空白框选与整组移动，含应用层保存重开与守卫回归。本验收闭合该批次：全部六门通过、身份审计无退化、.4 体验包构建并经独立核验。

## 六门证据（均在 .localtools/ 下，输入前后哈希不变）

| 门 | 轮次 | 结果 | 证据目录 |
|---|---|---|---|
| 格式 | format-04 | 172 PASS / 0 FAIL，436+23 输入同 | drawio-multiselect-20261115-format-04 |
| 视图 | view-03 | 385 PASS / 3 原 SKIP / 0 FAIL | drawio-multiselect-20261115-view-03（含 4 张多选截图：760×480/1180×780 × 深/浅） |
| 应用 | app-04 | 61 PASS / 1 原 SKIP / 0 FAIL | drawio-multiselect-20261115-app-04 |
| 六根全量 | test-01 | 3574 PASS / 49 原 SKIP / 0 FAIL，3623 项 | drawio-multiselect-20261115-test-01 |
| vet | vet-01 | exit 0 | drawio-multiselect-20261115-vet-01 |
| 身份审计 | final-audit-01 | 基线 3424 身份零丢失、零退化、零新 SKIP；source.diff 留档 | drawio-multiselect-20261115-final-audit-01 |

## 实测纠正史（原始失败均保留）

- view-01：354 PASS / 31 FAIL / 3 SKIP。根因聚类：Find 只匹配标签文本（修正为按无障碍标签查找）、Tester 指针事件不携带修饰键（改用 InputEvent 白盒缝隙，vendor 未动）、stageStyleField 窗口尺寸硬编码（参数化）、edgeScratch 共享缓冲别名（测试侧 slices.Clone）、Release 误清选择、draggingID 未置位、框选屏幕裁剪（改世界空间全包含）、切页后选择未恢复（changePage 自动选首节点 + syncDrawioSelection 双向分支）。
- app-01：42 PASS / 19 FAIL / 1 SKIP。方向用例失败由 Array 路由点解析修复（format 层）闭合；旧切页测试由切页自动选中契约闭合；恢复断言修正为既有 RecoverDraft 信封契约（与 mindmap 侧既有断言同款）。
- view-02 仅剩 1 FAIL（edgeScratch 别名），view-03 全绿；app-02/03 为测试编辑中间态失败（字符串字面量与信封分隔），app-04 全绿。

## 交付物

[Windows x64 draw.io 多选体验 ZIP](<F:/AIproject/LevNote/native/downloads/LevNote-Drawio-preview-20261115.4-windows-x64.zip>)

- 版本：native-drawio.20261115.4-preview
- ZIP：11,743,909 字节，SHA256 a28aa504239b84f011e61d690615044b01c7566677f6b962e5a9f59130512b07
- 宿主：14,986,240 字节，SHA256 d2992144c2480a4b1912cfb437ec7b3a087c899c49b6e4917dd5536e1b050213
- 26 文件 / 27,839,072 字节；五 amd64 GUI PE（host + 四 worker）exit0；八引擎路径完整唯一
- 独立核验（drawio-multiselect-preview-final-verify-20261115.py）：PASS——ZIP 重读、全清单哈希、构建与测试 436 输入同源一致

## 契约与边界

- 框选：移动仅预览候选集，松开才发布选择；4px 阈值；世界空间全包含；Ctrl+Shift 并集。
- 整组移动：公共吸附四格增量；单次撤销事务；手动折点保持绝对坐标，预览仅移动端点不动折点。
- 多选期间单节点删除/连线/字段写入关闭；六对齐按钮键 drawio.align.*，两行，≥45×29.9px，标题显示「已选择 N 个图形」。
- 恢复内容沿用既有 RecoverDraft 信封契约；原缓冲区与磁盘不动。
- 未声明物理桌面输入/IME/资源验收；.29 仍为最新正式接受包；本批未提交推送。
- 文档：STATUS.md / README.md 能力表 / portable/README.md 已更新多选能力描述。
