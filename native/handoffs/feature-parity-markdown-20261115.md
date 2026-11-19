# Markdown 行内格式交接 · 2026-11-15

## 交付与写域

本轮实现块草稿的粗体、斜体、行内代码：三个原生按钮及仅草稿编辑区内生效的 Ctrl+B / Ctrl+I / Ctrl+E。界面继承工作台深浅主题、字体与圆角，未引入新依赖或编辑器。

仅下列 7 个文件属于本任务写域：

- native/formats/markdown/inline_edit.go（新增）
- native/formats/markdown/inline_edit_test.go（新增）
- native/modules/markdownview/inline_actions.go（新增）
- native/modules/markdownview/inline_actions_test.go（新增）
- native/modules/markdownview/view.go（窄接线）
- native/app/markdown_inline_app_test.go（新增）
- native/handoffs/feature-parity-markdown-20261115.md（本记录）

未修改共享 app 生产实现、shared editor、旧测试、vendor、工具链或既有证据；未运行 Go、提交、推送或子委派。Go 测试由 lead 串行执行。

## 行为与守卫

- 普通段落、标题、引用的块草稿显示三项格式按钮。表格草稿、围栏代码、列表原始结构和其他结构源码不启用该工具条。
- 完整普通选区包裹标记；已有同种格式的内容/完整标记选区或内容内光标取消该格式。支持既有下划线形式的粗体/斜体和多反引号代码。
- 空选区插入标记对，光标位于中间。编辑器 UTF-16 选区转换为格式层 UTF-8 字节区间；反向选区及 undo/redo 前后选区保持。
- 单次动作先复核编辑器、宿主、版本、选区和生命周期，再执行草稿和宿主大小预检。仅调用草稿 ReplaceText，选择位置纳入一个本地事务；事务完成后恢复并调用原 OnChange，经既有 syncDraft 写回宿主。不会持有宿主事务跨输入事件。
- 按钮 KeepFocus；输入法预编辑存在时保留组合文本/焦点，仅提示先确认文字。Save、模式切换和换文档继续使用既有 Flush。
- readonly、外来事务、版本/选区变化、替换或 Release、未同步冲突草稿均阻止旧动作落到新对象。Draw 末尾释放过期手势及已结束草稿的新引用。
- 纯计划使用生产 parseInline 再解析候选：新 span 必须对应请求，其他 span 的 kind/text/href 与相对位置保持。跨行、已有格式嵌套/部分选择、边界歧义、代码两端反引号/双端空格、超过64KiB块/扫描预算都会返回可见说明，原文保持。
- 原始 BOM、CRLF、未知块由原文档保存通道保留。本轮不改变解析语法、不增加递归嵌套格式支持。

## 回归与实际结果

新增测试：

- 格式层：TestInlineEditWrapToggleAndByteFidelity、TestInlineEditEmptyCaretAndExistingAlternateMarks、TestInlineEditCodeDelimiterAndExplicitBoundaries。
- 模块层：TestMarkdownInlineButtonsKeysUndoRedoAndFinish、TestMarkdownInlineReversedUTF16AndEmptyCaret、TestMarkdownInlineCompositionRetainedUntilExplicitCommit、TestMarkdownInlineAdmissionReadonlyConflictAndHeldSelection、TestMarkdownInlineMultilineAndNestedFeedback。
- App：TestMarkdownInlineRealAdapterBOMCRLFSaveReopen、TestMarkdownInlineRealAdapterIMEBarriers、TestMarkdownInlineLayoutAndCapture。

固定 .localtools/go1.27.3/bin/gofmt.exe 已解析并格式化6个 Go 文件。最初 -w 尝试临时文件 Access denied/exit2，随后按既定 stdout + 文件工具写回完成，未修改 ACL。

lead 的 markdown-01 实际运行（证据 .localtools/feature-parity-20261115-markdown-01/stdout.jsonl）：143 RUN / 138 PASS / 3 原有 SKIP / 2 FAIL（父子），exit1。全部新增格式层与模块层通过；App 首个保存重开子例在已通过磁盘 BOM/CRLF 和原文断言后，于导出调用 panic：fixture 仅 Open 没有创建首帧，other.View 为 nil。已补真实 ui.NewTester(reopened.View)+Frame 再检查导出，未改 app 生产行为，原失败保留。

lead 的 markdown-02 实际运行（证据 .localtools/feature-parity-20261115-markdown-02/stdout.jsonl）：213 RUN / 204 PASS / 7 原有 SKIP / 2 FAIL，exit1。新增 Inline 全部通过，包括 app BOM/CRLF 重开导出、IME save/mode/tab 和4种主题/宽度布局。两个旧回归发现上方新增工具条挤压窄窗预览：TestMarkdownTaskCheckboxRealAdapterFailedFlushRetainsDraftAndSource 的复选框失去可见面积，TestNarrowMarkdownConflictKeepsViewOutOfStatusRow 的正在编辑此块落到状态栏下方。已修产品布局：三个格式按钮并入原操作行，紧凑面文案保持原 Label，错误反馈复用原状态行；旧测试零修改。上述修正已在 markdown-03 通过；原失败保留，不改旧断言。

截图可选环境变量 LEVNOTE_MARKDOWN_INLINE_SCREENSHOT_DIR；TestMarkdownInlineLayoutAndCapture 始终执行布局断言，设置变量时输出 dark=true/false × 760/1180 宽、480高的4张真实 MyGo 帧。没有新增 skip。

## 最终聚焦结果与读图

lead 的第三次实际运行：[markdown-03 原始日志](<F:/AIproject/LevNote/.localtools/feature-parity-20261115-markdown-03/stdout.jsonl>)，exit0。按该日志 Test 事件复核：213 RUN / 206 PASS / 7 原有 SKIP / 0 FAIL。lead 回执确认424份输入前后完全相同。新增 Inline 回归与两个旧窄窗/冲突回归均通过。

四张原生 MyGo 实际帧已逐张读取：

- [深色760×480](<F:/AIproject/LevNote/.localtools/feature-parity-20261115-markdown-03/images/markdown-inline-true-760.png>)
- [浅色760×480](<F:/AIproject/LevNote/.localtools/feature-parity-20261115-markdown-03/images/markdown-inline-false-760.png>)
- [深色1180×480](<F:/AIproject/LevNote/.localtools/feature-parity-20261115-markdown-03/images/markdown-inline-true-1180.png>)
- [浅色1180×480](<F:/AIproject/LevNote/.localtools/feature-parity-20261115-markdown-03/images/markdown-inline-false-1180.png>)

两种宽度下，三个格式按钮和完成按钮同排完整显示，格式选区保留在编辑区，现有预览在草稿下方可见，操作行与状态栏无重叠。深浅主题下文字、选区、边框、操作面均可分辨，沿用现有配色。截图是 Tester 绘制结果，不作为实体桌面输入法验收。

## 接续与冻结

本任务6份生产/测试 Go 文件保持冻结，本记录为唯一收尾写入。Markdown 行内格式任务已交付 lead；根继续统一全量 test/vet 与整体交付验收。未把聚焦结果扩大为全产品验收。
