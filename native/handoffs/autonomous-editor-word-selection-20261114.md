# Task32 · 双击词内部选区保留分隔符 · 本地实现交接

## 任务状态（type:state）

- 固定基点：`257ef9215fb999b1e154d64a076efe7a7c977b0a`。实施 owner：`next-editor-scout`。
- 仅修改[editor.go](<../ui/editor/editor.go>)、新增[word_selection_test.go](<../ui/editor/word_selection_test.go>)及本篇；ignored create-only 执行证据位于工作区根 `.localtools/editor-word-20261114/`，其记录器不进入产品。
- 首公开双击选区实际 RED → 最小 GREEN 已完成；最终 focused 为 **8 个顶层 + 2 个嵌套 PASS、0 FAIL、0 SKIP、exit0**。仅首选区测试形成 RED→GREEN，其余垂直扩展第一次执行即 GREEN。
- 本地固定 Go1.27.3 Windows/amd64、CGO_ENABLED=0、offline vendor、workspace cache/tmp、telemetry off；未运行 Editor/App/modules full、vet/build、产品 GUI、Git、network/box、ACL 或包装器。
- 本篇与源码/测试冻结后停止写入；root负责冻结六文件的双轴审阅和统一 App+editor+modules affected full 及后续门禁。本成果为源码/Tester 有限证据，不是物理 Windows GUI/IME 或 runnable 交付接受。

## 目标与最小实现（type:goal）

普通 built-in `Editor.View` 双击 CRLF 中间行词内部位置：源 `one\r\ntwo\r\nthree`、UTF-16 offset6，选区必须字面 `[5,8)`、选中文字 `two`。输入 `X` 后必须为 `one\r\nX\r\nthree`；真实键盘 Undo 还原原正文和 `[5,8)`，Redo 还原替换正文与 caret `[6,6)`。

新增[selectionWordBoundary](<../ui/editor/editor.go#L498-L514>)在移动 `pos` **之前**检查旧分隔符词表 `" \t\r\n.,;:()[]{}"`。保持 `GraphemeBoundary` 步进、UTF-16 坐标和4096上限；CRLF、emoji surrogate 与组合字素不被拆分。新增[insideSelectionWord](<../ui/editor/editor.go#L516-L531>)要求点的左右相邻 grapheme 都存在且都不是旧词表分隔符。

[PointerDown分支](<../ui/editor/editor.go#L804-L811>)仅 `Clicks == 2 && insideSelectionWord(pos)`走新边界；其余 `Clicks >= 2`保留原 `word`。原 `word`、所有导航/命令/ReadOnly/历史路径不改，Ctrl/Alt/方向键语义无新增调整。

**有限语义**：词边缘、分隔符与文档边缘仍走旧解释；三击及更多仍旧行为。不是所有点击位置的完整选词重设。例如 `one two three` offset4仍选 `[3,8)`，两空格之间仍 `[3,5)`，第三次真实点击 CRLF offset6仍 `[3,10)`；这些是保守兼容字面回归，未宣称也已修正。

## 实际 RED 与逐步扩展（type:dead-end）

首test采用公开 `BoundsForRange` → `IndexForPoint`有效 geometry → `Tester.Find("文档编辑区")`屏幕坐标 → 两次普通 `ClickAt`，backend自行计数。没有私有 word/boundary 测试、内部重算 oracle、伪造 `Clicks`、改现有断言或物理输入声称。

[red-01原始stdout](<../../.localtools/editor-word-20261114/red-01/stdout.raw>)逐字失败：

```text
=== RUN   TestDoubleClickWordKeepsCRLFBoundaries
    word_selection_test.go:37: double-click selected {Range:{Start:3 End:10} Reversed:false}; want literal two at [5,8)
--- FAIL: TestDoubleClickWordKeepsCRLFBoundaries (0.01s)
FAIL
FAIL	levnote.local/native/ui/editor	1.176s
FAIL
```

[首42行原字节](<../../.localtools/editor-word-20261114/audit-01/first-red-42-lines.raw>)与 RED 输入相等：1664B，SHA256 `e99b1775f7e1c8c2e916612a57dbd180e1affcef9533bd4bd11bd51a180b63e3`。后续只追加测试，首42行未弱化或重写。

所有测试通过实际已读[gate.sh](<../../.localtools/gate.sh>)执行。命令公共前缀为 `bash .localtools/gate.sh test -count=1 -v ./ui/editor/ -run PATTERN`；`PATTERN`如下表。raw是 `-v` stdout/stderr，**不是 Go test JSON**；JSON只有证据 receipt/maps。UTC起止是外层 gate 窗口；receipt另列实查 Go PID/image/created_utc，并非推测 PID。

| create-only run / receipt | PATTERN | 实查 Go PID | gate UTC起止（2026-11-14） | exit / 结果 |
|---|---|---:|---|---|
| [red-01](<../../.localtools/editor-word-20261114/red-01/receipt.json>) | `^TestDoubleClickWordKeepsCRLFBoundaries$` | 24092 | 06:22:08.595402Z–06:22:10.939063Z | 1 / 真RED |
| [green-01](<../../.localtools/editor-word-20261114/green-01/receipt.json>) | 同首test | 30304 | 06:24:45.045935Z–06:24:47.713460Z | 0 / 最小GREEN |
| [crlf-history-01](<../../.localtools/editor-word-20261114/crlf-history-01/receipt.json>) | `^TestDoubleClickWordReplacementUndoRedoKeepsCRLF$` | 21388 | 06:27:12.955709Z–06:27:15.412703Z | 0 / 首GREEN |
| [lf-01](<../../.localtools/editor-word-20261114/lf-01/receipt.json>) | `^TestDoubleClickWordKeepsLFBoundaries$` | 6360 | 06:28:25.644333Z–06:28:27.908718Z | 0 / 首GREEN |
| [spaces-01](<../../.localtools/editor-word-20261114/spaces-01/receipt.json>) | `^TestDoubleClickWordKeepsSpaces$` | 21700 | 06:29:45.590692Z–06:29:47.998906Z | 0 / 首GREEN |
| [punctuation-01](<../../.localtools/editor-word-20261114/punctuation-01/receipt.json>) | `^TestDoubleClickWordKeepsPunctuation$` | 23520 | 06:30:47.592732Z–06:30:49.996573Z | 0 / 首GREEN |
| [unicode-01](<../../.localtools/editor-word-20261114/unicode-01/receipt.json>) | `^TestDoubleClickWordKeepsEmojiAndCombiningGraphemes$` | 24208 | 06:31:58.699805Z–06:32:01.097784Z | 0 / 首GREEN |
| [edge-compat-01](<../../.localtools/editor-word-20261114/edge-compat-01/receipt.json>) | `^TestDoubleClickDelimiterAndWordEdgeRetainSelection$` | 27920 | 06:33:17.382620Z–06:33:19.589973Z | 0 / 顶1+嵌2首GREEN |
| [third-click-01](<../../.localtools/editor-word-20261114/third-click-01/receipt.json>) | `^TestThirdClickRetainsExistingWordSelection$` | 32652 | 06:34:24.952923Z–06:34:27.460149Z | 0 / 首GREEN |
| [focused-union-01](<../../.localtools/editor-word-20261114/focused-union-01/receipt.json>) | `^(TestDoubleClickWord\|TestDoubleClickDelimiterAndWordEdgeRetainSelection\|TestThirdClickRetainsExistingWordSelection)`（实际regex为未转义的三个 `|`） | 14656 | 06:36:01.743402Z–06:36:07.893120Z | 0 / 顶8+嵌2PASS |

[最终stdout](<../../.localtools/editor-word-20261114/focused-union-01/stdout.raw>)1262B，SHA256 `5fe22e91fb33ee31c2a317109a3313d060cda0bea9a3fac61d62609c05a463c9`。最终[receipt](<../../.localtools/editor-word-20261114/focused-union-01/receipt.json>)SHA256 `3b8864324191f353378312c5e39e773c6873d0092c0c79d3b8eff52c6d42faf7`；每次 stderr均0B。

## 冻结身份、审计与下一步（type:progress）

[新audit-01结果](<../../.localtools/editor-word-20261114/audit-01/result.json>)SHA256 `8511f766bb5fbeb3e0fc72dcfc0630bbffc3a899401d016169cad84a693141bc`，实际exit0；每个run前后map相同。限定 inventory覆盖 editor、core/text、vendor、固定toolchain和原8保护文件：16512项中仅2个本任务源码/测试从首RED到最终改变，其余16510项各run、最终均与首RED相同。该scope排除了同行独立写入的其他app源，不作整工作区不变声明。

从最终[editor.go](<../ui/editor/editor.go>)删去两个新增helper并将exact2分支还原，得到[原源码投影](<../../.localtools/editor-word-20261114/audit-01/original-source-projection.raw>)SHA256 `b5d98594bf9213c0d2b0ef2b69c1c00040ed5e365ff133a11bf813aabfd9366a`，与首RED原生产源逐字相等。原导航/命令/三击等未改由此确认。原vendor/工具链、existing assertions、原八、旧证据/.29保留。

只读固定 `gofmt.exe -d`两本任务文件：PID27516，06:38:41.105667Z–06:38:41.148891Z，exit0，stdout/stderr均0B，未写源。该检查不是新增build/test。

| 冻结文件 | bytes / lines | SHA256 |
|---|---:|---|
| [editor.go](<../ui/editor/editor.go>) | 25584 / 960 | `d40cfa709063f060e34e03c64a4cb4e09745a18b2b9f17b44d743cd2104aa24d` |
| [word_selection_test.go](<../ui/editor/word_selection_test.go>) | 6806 / 175 | `e4106d6bdd5c0fcf5f1dea076aeb674a587868deea188ae7bd1fac13d91de76d` |
| [run.py](<../../.localtools/editor-word-20261114/run.py>) | 记录器 | `bce4e419fbaf974fa1d9139b422cbd6f6253cd45a25abba7f56842563063884b` |
| [audit.py](<../../.localtools/editor-word-20261114/audit.py>) | 只读核验器 | `c75e9b213d60db2cd218917675d7bb5ac95000ec34e17a4a565bde3a2ad662a4` |

本篇SHA由创建后的freeze-final-01清单外置固定，避免自引用。下一步：root按[审计结果](<../../.localtools/editor-word-20261114/audit-01/result.json>)核对这3个交付文件及执行证据，等另一实施lane停写后冻结exact6文件，完成独立双轴审阅，再统一运行一次 App+editor+modules affected full。child停止写入/门禁，无后台任务待收。
