# 成熟功能首批：源码与 Windows 体验包验收

日期：2026-11-15。版本：native-features.20261115.3-preview。此记录接受本批源码与体验包，整软件后续开发继续。

## 已交付行为

- 命令面板提供25个真实动作；Ctrl+Shift+P/F1打开，标题或英文关键词筛选，方向键和Enter执行。执行前重新检查活动文档与可用性。
- 源码编辑器支持Alt+↑/↓移动逻辑行、Shift+Alt+↑/↓复制行；保留混合行尾、UTF-16选择方向与一次操作一次撤销。
- Markdown标题/段落/引用草稿支持粗体、斜体、行内代码按钮与Ctrl+B/I/E；完整格式可取消，空选区插入标记对。单行单层范围之外有具体提示，原文和草稿保持。
- 导图节点提供文字色、底色、边框色预设与逐项恢复默认，保留未知样式。显式不透明底色且缺少文字色时，可编辑画布、只读场景与SVG共享对比度回退；自定义边框与选中外圈同时显示。

## 验证结果

| 检查 | 真实结果 | 证据 |
|---|---|---|
| 最终全量test-02 | 3424 RUN / 3375 PASS / 49原SKIP / 0FAIL；61.467秒，426输入前后同 | [结果](<F:/AIproject/LevNote/.localtools/feature-parity-20261115-test-02/result.json>) |
| vet-01 | exit0；426输入前后同 | [结果](<F:/AIproject/LevNote/.localtools/feature-parity-20261115-vet-01/result.json>) |
| 测试身份与差异审计 | 前版3264身份全部保留，新增160；无遗漏/退化/新增SKIP/重复RUN或终态；30个产品文件变化，无删除 | [审计](<F:/AIproject/LevNote/.localtools/feature-parity-20261115-final-audit-02/audit.json>)、[差异](<F:/AIproject/LevNote/.localtools/feature-parity-20261115-final-audit-02/source.diff>) |
| graph-03 | 包含只读worker的422 RUN / 419 PASS / 3原SKIP / 0FAIL；426输入同 | [结果](<F:/AIproject/LevNote/.localtools/feature-parity-20261115-graph-03/result.json>) |
| Markdown缺陷复现/修复 | inline-01：6PASS/4FAIL含父；inline-02同断言10PASS；新增4兼容正例在全量test-02通过 | [原失败](<F:/AIproject/LevNote/.localtools/feature-parity-20261115-inline-01/result.json>)、[修复](<F:/AIproject/LevNote/.localtools/feature-parity-20261115-inline-02/result.json>) |

独立审阅完成命令面板、编辑器、Markdown和图形变更，最终无剩余阻塞项。Markdown空标记问题由gap-editor发现：含临时文字的probe可通过，而实际空标记可能吞并后方已有粗体或行内代码。修正同时解析真实候选，允许新空span缺席，但严格保留无关span的类型/内容/链接/偏移/数量。取消与非空格式原验证保持。

第一轮完整test-01的3396RUN/3347PASS/49原SKIP/0FAIL保留；它早于上述边界修正与只读场景补齐，不代替最终test-02。palette筛选状态、Markdown窄窗行拥挤和graph fixture的先前失败、修正与原始输出均保留，不覆写为成功。

## 实际绘图检查

Markdown窄窗与宽窗深浅截图已检查。graph-03的760×480只读深浅截图显示浅黄填充上的深色文字、保留绿色原边框与蓝色选中外圈；文字色面板截图中28px恢复默认按钮完整可见。

- [深色只读图](<F:/AIproject/LevNote/.localtools/feature-parity-20261115-graph-03/images/node-colors-dark-true-760-readonly.png>)
- [浅色只读图](<F:/AIproject/LevNote/.localtools/feature-parity-20261115-graph-03/images/node-colors-dark-false-760-readonly.png>)
- [浅色文字面板](<F:/AIproject/LevNote/.localtools/feature-parity-20261115-graph-03/images/node-colors-dark-false-760-text.png>)

这些是MyGo测试器实际渲染与输入事件证据。实体Windows IME、系统剪贴板、窗口关闭/再次启动、四worker随包生命周期与整应用资源测量仍归最终平台验收。

## 交付与完整性

[Windows x64 功能增强体验ZIP](<F:/AIproject/LevNote/native/downloads/LevNote-Features-preview-20261115.3-windows-x64.zip>)：11,699,328字节，26文件，解压总计27,728,525字节。五个amd64 GUI PE均构建exit0，固定Go与26个MyGo补丁pin通过，1276构建输入前后相同。

ZIP SHA256：

`e52fcb9336142438bf81f2919bbf5bc6da2d259b2a965a1e44aa5997b6d6078a`

宿主14,878,720字节，SHA256：091a95fec1ba66efecefc752e6edff6dfc10044ac60b874a98f7c4d11368b1eb。

[构建结果](<F:/AIproject/LevNote/.localtools/feature-parity-preview-build-20261115-01/result.json>)与[独立ZIP核验](<F:/AIproject/LevNote/.localtools/feature-parity-preview-final-verify-20261115.json>)均成功。独立检查重读ZIP、CRC、全部清单哈希、四引擎精确八路径、五个GUI PE以及426个测试/构建共同输入。未启动新包；正式接受包仍为native-worker.20261110.29，旧体验包保持原字节。本轮未创建Git提交或推送。

## 接续开发

按[draw.io实施合同](<F:/AIproject/LevNote/native/handoffs/feature-parity-drawio-next-20261115.md>)形成新批基线后，先做批量x/y原子补丁、Ctrl多选与六对齐，再加入Shift框选和整组拖动。格式、共享graphview、app纵向测试分别设唯一写者；root串行运行产品检查，最终再独立审阅。
