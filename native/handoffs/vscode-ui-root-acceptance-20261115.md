# VS Code 风格工作台与 Windows 体验包验收

日期：2026-11-15。基点：f2c2ceb82acecb40fb0502c047e41ba12a12be47。当前改动留在未提交工作区。

## 交付结果

已完成受 VS Code 启发的原生工作台：48px 图标活动栏、紧凑文件树、34px 标签栏、蓝色状态栏、深浅主题与简化欢迎页。新用户默认深色；已保存的 light 偏好优先。原编辑、保存、只读、编码/BOM、组合输入、标签身份、延后关闭与手动滚动语义保留。

[Windows x64 体验 ZIP](<F:/AIproject/LevNote/native/downloads/LevNote-VSCode-UI-preview-20261115.1-windows-x64.zip>)：11,641,997 字节，26 文件；版本 native-ui.20261115.1-preview。解压完整文件夹后运行 levnote-native.exe。

ZIP SHA256：93caaf6a7325811c3603ae86ef04e52b51b737f2a5a3a41b7a4449a6a2aaa8f1。
宿主 14,761,984 字节，SHA256：b252ba81265931a3e272a686bf82d12645979118c3df306b760cc470451dd153。

该包包含本次 UI 和当前工作区修复，定义为体验包。正式发行接受仍保留 .29；新包尚未完成实体输入法、剪贴板、正常关闭与新 worker 生命周期的综合发行验收。

## 测试与审阅

- [最终聚焦回归](<F:/AIproject/LevNote/.localtools/vscode-ui-20261115-focused-04/result.json>)：103 PASS、4 SKIP、0 FAIL。
- [完整六根 test03](<F:/AIproject/LevNote/.localtools/vscode-ui-20261115-test-03/result.json>)：3258 RUN、3208 PASS、49 SKIP、1 FAIL，退出1。唯一失败为 core/document/disk_test.go:117 的 fixture os.Rename Access is denied，发生在被测 Revert 断言之前。
- [同一用例窄复核](<F:/AIproject/LevNote/.localtools/vscode-ui-20261115-disk-01/result.json>)：原源码/原断言未变，1 PASS、退出0。保留完整失败，不将 test03 改记为全绿。
- [精确身份汇总](<F:/AIproject/LevNote/.localtools/vscode-ui-final-audit-20261115/result.json>)：3209 个唯一 PASS = 3183 原 PASS + 26 新工作台用例/子用例；无遗漏、重复或新增 SKIP，49 原 SKIP 相同。
- [vet](<F:/AIproject/LevNote/.localtools/vscode-ui-20261115-vet-01/result.json>) 与 [build](<F:/AIproject/LevNote/.localtools/vscode-ui-20261115-build-01/result.json>) 均退出0。test03/disk/vet/build/capture 的六产品根及所采工具/依赖 pin 前后一致，gofmt 只读检查与 git diff --check 通过。
- [Standards 独审](<F:/AIproject/LevNote/native/handoffs/vscode-ui-standards-review-20261115.md>)与[Spec 独审](<F:/AIproject/LevNote/native/handoffs/vscode-ui-spec-review-20261115.md>)无剩余实质问题，已停止写入。lead另审最终差异、配套测试位置适配和新增工作台文件。

独审及回归中修复了从插件返回查找/设置的反向关闭、浅色状态栏悬停/按下对比度、固定可访问名称遮盖真实状态消息、目录展开名称丢失。旧菜单与标签测试只适配新几何，核心行为断言保留。各次失败原流仍在其独立目录。

## 视觉证据

[最终 capture](<F:/AIproject/LevNote/.localtools/vscode-ui-20261115-capture-01/result.json>)：7 PASS、0 FAIL，六场景十张 PNG。lead查看深色源码、浅色源码、深色阅读和最小浅色欢迎页；前轮另查状态栏交互，最终截图自动像素断言再次确认对比度。

- [深色源码](<F:/AIproject/LevNote/.localtools/vscode-ui-20261115-capture-01/images/dark-source.png>)，1180×780。
- [浅色源码](<F:/AIproject/LevNote/.localtools/vscode-ui-20261115-capture-01/images/light-source.png>)，1180×780。
- [深色阅读](<F:/AIproject/LevNote/.localtools/vscode-ui-20261115-capture-01/images/dark-reading.png>)，1180×780。
- [最小浅色欢迎页](<F:/AIproject/LevNote/.localtools/vscode-ui-20261115-capture-01/images/light-welcome-narrow.png>)，760×480。

截图由真实 MyGo 测试器渲染；未据此宣称物理键鼠/IME或整应用资源表现通过。

## 新体验包构建与核验

[固定目标包装器](<F:/AIproject/LevNote/.localtools/vscode-ui-preview-build-20261115.py>)沿用本地 Go1.27.3、离线 vendor、telemetry off、缓存/tmp、26补丁manifest固定SHA及逐文件校验。仅构建宿主和既有四个固定worker，使用 trimpath、buildvcs=false、windowsgui；未修改权限/完整性标签、工具链或旧包。

[构建结果](<F:/AIproject/LevNote/.localtools/vscode-ui-preview-build-20261115-01/result.json>)：五次编译各退出0，1258项构建输入前后一致；五个 PE 均为 amd64 GUI。26文件共27,590,861字节，含14份原样许可/归属资产和四引擎manifest；README明确预览状态，所有本地链接通过。原始 argv、环境、stdout/stderr、PE与hash记录在相邻文件。

包装器独审在执行前发现 release-manifest 生成晚于本地链接检查，已修正为 assets→manifest→links，窄复审通过；五目标构建仅执行一次。

[独立ZIP重读结果](<F:/AIproject/LevNote/.localtools/vscode-ui-preview-final-verify-20261115.json>)：CRC、26唯一成员、25条发布清单、八个精确引擎路径、四引擎版本/字节/hash、五GUI PE全部通过；409个共同测试/构建输入摘要一致，六产品根路径集合完整相同。构建另采vendor和资产；测试收据未采整个vendor，不扩大其证明范围。

首次独立核验在读ZIP前将六根测试清单与包含vendor的构建清单比较，因覆盖范围不同退出1；[原核验失败](<F:/AIproject/LevNote/.localtools/vscode-ui-preview-verify-01.failure.json>)保留。只修正核验范围后继续，未重编、改包或重跑产品测试。

## 后续

用户可直接体验新界面。正式发行尚待四worker实际生命周期、宿主正常关闭与恢复、物理中文输入/剪贴板/选择器及显示缩放验收；这些不阻碍交付明确标识的UI预览。全软件其他能力缺口继续见[当前状态](<F:/AIproject/LevNote/native/STATUS.md>)。本轮未提交、推送或启用CI。
