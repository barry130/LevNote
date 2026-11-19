# flowix 工作台布局批次根验收（2026-11-16）

## 任务
用户在 view 主题移植后指出布局仍杂乱难看，要求继续。根代理把任务从重新配色收敛为工作台结构整改：顶部动作区、文档搜索入口与左下新建文档区的可见布局重建，保留 flowix 设计令牌和既有文档能力。

## 实现
- 顶栏固定 48px：品牌、竖线、分类后的 新建/打开/最近关闭/文件夹/保存，右侧为胶囊文档搜索、命令面板与宽窗工作区名；动作统一为 32px 图标+文字的幽灵按钮，保留快捷键、可用性与 Label 不变。
- 左下「新建文档」改为分隔线、小节标题和四个 30px 左对齐列表项；Markdown、纯文本、思维导图、图纸分别带独立语义图标，点击仍走既有 NewDocument。
- 「最近关闭」文本改为 28×32 向下箭头按钮，Label 仍是「最近关闭的文件」，菜单与最近关闭能力未变；补齐顶栏与活动的图标资产。
- native/app/workbench_layout_test.go 的 TestWorkbenchCompactToolbarAndCreationList 覆盖 760/1180 两种宽度：八个顶栏入口均在 48px 内；四个新建入口高度为 30、位于状态栏上方；每次点击实际创建对应扩展名文档。

## 验证证据
- 批次六根校验器：test-03 exit0，Run 3626、Pass 3577、原 Skip 49、Fail 0（目录 .localtools/flowix-chrome-20261116-test-03）；vet-03 exit0（目录 .localtools/flowix-chrome-20261116-vet-03）。
- 身份审计 final-audit-05 关联 test-03：436 输入前后一致，3424 个历史测试身份零删除/零退化/零新增跳过；3626 项没有重复运行或结果，新增测试 202 项（含本批布局测试），changedProductFiles 27。
- gofmt 对 workbench_chrome.go、workbench_icons.go、view.go、reopen_closed.go、workbench_layout_test.go 输出为空；测试与构建输入 436 项同终态源码。
- 截图：.localtools/flowix-restyle-20261116/images3/ 共 11 张（深浅 × 源码/阅读/欢迎/窄窗 + 状态栏悬停/按下），人工检视确认顶栏不再挤叠、胶囊搜索带 Ctrl+P 提示、文档列表项与四类新建入口左对齐、深浅主题与命令入口可分辨。深色欢迎与阅读视图见[深色欢迎](<F:/AIproject/LevNote/.localtools/flowix-restyle-20261116/images3/dark-welcome.png>)与[浅色阅读](<F:/AIproject/LevNote/.localtools/flowix-restyle-20261116/images3/light-reading.png>)。

## 交付
- [Windows x64 工作台布局体验ZIP](<F:/AIproject/LevNote/native/downloads/LevNote-Flowix-preview-20261116.6-windows-x64.zip>)：11,746,501 字节，26 文件，解压总计 27,843,472 字节，SHA256 `10d0cb6ea220831fd77a4a7e1d556a05db4ee7faf01a8e7491690386adc05171`；版本 native-flowix.20261116.6-preview。
- 五个 amd64 GUI PE 均构建 exit0；宿主 levnote-native.exe 14,990,336 字节，SHA256 `ad85b57a6922d7a5009c7e0e7688a901c526fa064780d998d37ea8f0e26439af`；固定 Go 与 MyGo 补丁 pin 通过，1,286 构建输入前后相同。
- .localtools/flowix-chrome-preview-final-verify-20261116.py 对 ZIP 重读、26 文件/八引擎路径、全部清单哈希、五 GUI PE 与构建/测试 436 输入同源核验 PASS；回执 .localtools/flowix-chrome-preview-final-verify-20261116.json。

## 修复史
- view.go 直接声明 BorderTop 不存在 → 改用 1px Border 背景行作为新建区分隔。
- gofmt 不接受手工制表列格式 → 用最终 gofmt 输出收敛图标常量与测试文件尾部空行；修复后所有触及文件 gofmt -l 为空。
- validator 复用脚本后审计目录曾指向错误/占位目录 → 最终审计脚本固定历史 drawio-multiselect-20261115-baseline 并允许 audit 输出尝试号与 与 test 输入尝试号分别记录；保留失败目录编号，最终采用 test-03 / vet-03 / audit-05。
- 构建器原把 audit-attempt 绑定到 test-attempt → 增加 --audit-attempt；一次构建 exit0，无覆盖旧包。

## 边界
- 预览构建：未做 Windows 物理输入法/剪贴板/正常关闭的完整发行验收；正式接受包仍为 native-worker.20261110.29。
- 未提交推送；.3/.4/.5 体验包与全部历史验收证据原样保留；本次只调整工作台布局与测试，不声明 Typora/VS Code/XMind/draw.io 全功能对等。
