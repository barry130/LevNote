# Harness 交接证据 · 2026-11-19

主入口：[HANDOFF.md](../../../HANDOFF.md)。用户要求停止功能开发并提交上传当前全部工作。证据保留在原路径，通过显式加入 Git 随仓库提供；不移动、不覆盖原件。

| 内容 | 已保留的仓库路径 | 含义 |
|---|---|---|
| 本轮前快照 | [baseline 清单](../../../.localtools/noteboard-parity-20261119/baseline/manifest.json) 与同目录 source | 444 项输入；用于区分本轮三处修改，不是待应用的旧源码 |
| 本轮真实失败 | [第一次日志](../../../.localtools/noteboard-parity-20261119/drawio-finish/01-log.txt)、[第二次日志](../../../.localtools/noteboard-parity-20261119/drawio-finish/02-log.txt) 及同目录 command/exit/run 脚本 | 编译失败、行为失败均 exit 1；随后修改尚未验证 |
| 最近预览验证 | [.6 历史验收](../flowix-chrome-root-acceptance-20261116.md)，原证据目录 `.localtools/flowix-chrome-20261116-test-03/` | 只覆盖此前预览，不覆盖当前 WIP |
| 原离线门禁 | [gate.sh](../../../.localtools/gate.sh) | 固定本机路径，迁移前审阅 |
| 旧验证/构建脚本 | [验证脚本](../../../.localtools/flowix-chrome-validate-20261116.py)、[预览构建脚本](../../../.localtools/flowix-chrome-preview-build-20261116.py) | 仅保存复现依据，不重新执行旧命名空间 |
| 遗留失败 | [pass.log](../../../pass.log) | 文件名含 pass，实际为 FAIL，缺完整上下文 |

曾尝试用文件批处理把上述七组证据复制到本目录，均报告 Windows ACCESS_DENIED，复制状态 unknown；[原始结果](copy-receipt.json)保留，后续检查未见目标证据文件。本次改为把允许读取的原路径证据直接纳入 Git，未通过其他文件接口重试复制，未修改 Windows 权限。正式依据是本仓库原路径文件和本次 Git 提交，不是失败复制的回执。

本轮三个文件的完整差异另存为 [drawio-current-batch.patch](drawio-current-batch.patch)，是相对开发前快照的审阅材料，不是已验证修复。Git 属性已为选中的原路径证据关闭文本行尾转换，避免清单与 CRLF fixture 在上传时变样。

本目录及 `.localtools` 的源码快照不参与产品编译。只检查 app、cmd、core、formats、modules、ui 六个产品根，不使用 `go test ./...`。工具链与缓存不会随这些选定证据入库。
