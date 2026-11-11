# Editor handoff

Acknowledged 2026-11-08: taking ownership of `ui/editor/**`, `core/text/**`, and `scripts/**` only. I have read `native/AGENTS.md`, `handoffs/plan.md`, and `editor_finish_task.txt`.

First stage: publish optional Flush / RecoverDraft / Outline contracts, repair the unfinished layout integration, then run focused editor/text checks. Geometry, input/data safety, bounded-work tests, and measured benchmarks follow. No subagents, commits, or pushes.

API ready: contract.go now exports Flusher / FlushView, Host.RecoverDraft, OutlineItem and OutlineProvider (Outline() []OutlineItem; GoToOutlineItem(string) bool). Host must retain the view on Flush failure. The optional interfaces leave existing View implementations source-compatible.

Build stage: editor/text now compile. Focused tests currently have only the three task-listed obsolete assertions failing (two undo expected wrong text, hardcoded wrap-click offset, long-line typing without focus); replacing them with substantive geometry/tail safety tests. Native cut now waits for Context.WriteClipboardThen success, guards delayed deletion by buffer/version/selection/generation; style invalidation retains focus.

## Completed editor/text stage — 2026-11-08

- Optional shared API is ready: `Flusher`, `FlushView`, `Host.RecoverDraft`, `OutlineItem`, `OutlineProvider.Outline()` and `GoToOutlineItem(string) bool`. Existing `View` implementations remain source-compatible. `Editor.Flush()` finalizes native composition; source wrappers should delegate it before save/replacement.
- Geometry now shares paragraph layout coordinates across paint, selection, caret, pointer and IME, with wrap affinity for Home/End, persistent desired X for vertical navigation, actual-row Y hit-testing, resize/font invalidation, empty lines, and lazy sparse row counts. Each paragraph paints once. Style changes preserve focus; release frees geometry and finalizes IME without discarding buffer/history.
- Long physical lines retain all text and use visible previous/next controls plus explicit UTF-16 range. The ordinary-paragraph shaping limit stays 32 Ki UTF-16; longer lines use 2 Ki windows (plus at most two surrogate-boundary units). This reduced measured long-line edit allocation materially. Tail typing, segment buttons, cross-boundary arrows/Down, surrogate deletion, combining/ZWJ deletion and undo are covered.
- Indent/outdent excludes a selection's final line when it ends at that line's start, preserves forward/reverse mapped endpoints, closes only its own transaction, treats an empty EndGroup as success, removes a tab or 1–4 spaces, and supports Shift+Tab at a caret. Enter preserves CRLF at the final line and copies only leading whitespace before the insertion point.
- Clipboard failure cannot cut content. Default native copies use `Context.WriteClipboardThen`; asynchronous cuts also verify buffer/version/selection/view generation before deletion. Existing app `mygo.Clipboard.Write(transfer.TextData(...))` returns the Windows `OleSetClipboard` HRESULT failure through `WriteData`, so its synchronous error-returning `Copy` callback is compatible.
- Buffer lookups now binary-search cumulative piece metadata and inspect one bounded chunk. `LineRange`, `LineColumn`, journal invalidation, undo/cancel/restore and random edits have coverage. Grapheme deletion is independent of font shaping; a narrow owned-code correction covers go-text v0.3.5's GB11 state defect for an emoji immediately followed by a ZWJ family (without editing the dependency).
- Wrapper sets bundled Go's `TEST_TELEMETRY_DIR` to this repository's build-cache/telemetry with mode `off`. No global telemetry setting is changed. `go env GOTELEMETRY GOTELEMETRYDIR` returned `off` and `F:\AIproject\LevNote\native\build-cache\telemetry`.

### Validation

Passed:

- `native/scripts/go.ps1 test ./ui/editor ./core/text`
- `native/scripts/go.ps1 vet ./ui/editor ./core/text`
- `native/tools/go/bin/gofmt.exe -w native/ui/editor native/core/text`
- `native/scripts/go.ps1 -GoArguments @('test','./ui/editor','-run','TestRenderAcceptance','-count=1')` with `LEVNOTE_EDITOR_SCREENSHOT_DIR` set to `native/build-cache/editor-acceptance`.

The acceptance PNG `native/build-cache/editor-acceptance/editor-wrap-ime.png` was opened and visually inspected: Chinese/emoji/combining characters, wrapped paragraph, CRLF empty lines, line numbers, active row and composition underline/caret display coherently. This uses MyGo's actual headless renderer/native-input adapter, not a desktop OS IME interaction; full desktop acceptance remains the app owner's integration step.

The bounded-layout test uses 100,000 lines: initial viewport shapes at most 12 paragraphs, ten idle clock advances/frames cause zero additional shapes and no pending animation, a far jump shapes at most 16 more paragraphs, the retained layout cache stays at 256 entries, and a tail edit preserves an unaffected cached prefix. Separate tests verify sparse row counts after a prefix insertion/undo/redo, caret-pointer-IME coordinate round-trips at 3 widths and 2 fonts, focus retention/release, and actual native clipboard failures.

### Measured benchmarks

Command: `native/scripts/go.ps1 -GoArguments @('test','./core/text','./ui/editor','-run','^$','-bench','BenchmarkLarge|BenchmarkLongLine','-benchmem','-benchtime=200ms','-count=1')`.
Windows amd64; AMD Ryzen 7 6800H; one local run (microbenchmarks, not a process RSS claim).

| Workload | Time/op | Bytes/op | Allocs/op |
| --- | ---: | ---: | ---: |
| 50,000-line lookup (LineColumn + LineRange + 10 UTF-16 slice) | 11.828 µs | 24 | 2 |
| 500,000-line lookup | 7.716 µs | 24 | 2 |
| 50,000-line tail insertion + undo | 58.110 µs | 38,850 | 9 |
| 500,000-line tail insertion + undo | 611.322 µs | 378,305 | 9 |
| 100,000-line cached caret query | 12.822 µs | 0 | 0 |
| 2 MiB single-line tail insertion + undo, including segment reshaping | 1.816 ms | 1,327,058 | 181 |
| 100,000-line headless idle frame | 84.086 µs | 67 | 12 |

The first 32 Ki long-line window run measured 15.851 ms / 24,072,619 B for tail insertion+undo; the 2 Ki window change above is included in final results.

### Explicit limits / integration notes

- Unvisited wrapped physical lines initially count as one row; the scrollbar estimate refines as they are visited. Long-line segments represent only the disclosed current part, with all document bytes retained.
- Piece metadata is rebuilt per edit; the 500,000-line insertion+undo measurement reflects that O(piece count) work. Grapheme context may grow for pathological arbitrarily long combining/RI sequences to preserve correct deletion.
- Toolkit shaping still owns native font fallback and glyph placement. Tests cover representative Unicode and IME transport; no claim of exhaustive OS-specific IME or bidi conformance is made.
- No optional editor feature expansion, commits or pushes. Owned editor stage is ready for root review/integration; available for a small reassigned acceptance/core task.

## Outline contract published — implementation enabled

Root approved `handoffs/outline-contract.md`. `ui/editor/contract.go` now exposes:
- `Adapter.Outline func(Host) OutlineSource`.
- `OutlineSource.OutlineSnapshot() (OutlineSnapshot,error)`, `ResolveOutlineItem(string) (OutlineItem,error)`, `ResolveOutlineAnchor(string) (OutlineItem,error)`, `Release()`.
- `OutlineNavigator.NavigateOutlineItem(string) error`, `CurrentOutlineProvider.CurrentOutlineItem() (string,bool)`.
- `OutlineItem.AnchorID`, one-based Level, UTF16 block-start Offset or -1; snapshot Version/Items/Truncated.
- Caps: 2048 items, 512 UTF8 bytes for labels and file node IDs. Errors distinguish stale, unsupported and readonly-hidden targets.

DeepSeek can implement app now. Resolve captured old ID AFTER Flush and adopt the returned current item; never require the old ID to remain in a regenerated snapshot. Source Markdown reveals its offset without mounting View. Source mindmap fails with actionable mode guidance; split mindmap navigates the existing graph and preserves source caret. Astra is implementing shared parser heading/slug extraction plus lightweight cached providers and defensive navigation. Focused validation remains pending.

Outline owned implementation/tests/vet checkpoint is complete and stable for app
integration. Exact validation, measurements, root-authorized folded-root layout
fix, and remaining app acceptance are recorded at the end of
`handoffs/outline-contract.md`. No commits/pushes or unrelated feature changes.
