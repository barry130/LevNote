# Native source line navigation: implementation plan

2026-11-09. Activated by source-navigation-implementation-task.md. The initial
proposal below is corrected to the accepted implementation constraints; final
code/test evidence lives in source-navigation-result.md. Scope is Ctrl+G/source
line navigation in the host app. No command-palette or navigation framework.

## Files and APIs

- `native/app/app.go`
  - Add app state: `ShowGoToLine`, `GoToLineQuery`, `GoToLineError`.
  - Capture Tab/Document/Editor/Buffer pointers and a panel generation.
  - Add `closeGoToLine()` to clear state and all target references.
  - Call it after successful active-tab changes in `Select`,
    `addDocumentChecked`/`recoverDraft`, and after committed `removeTab`.
- `native/app/view.go`
  - Handle `Ctrl+G` in the existing `shortcuts` path.
  - Escape closes the panel without changing mode, selection or document.
  - Add a compact inline `gotolinebar` near the existing find/settings bars.
  - Turn the current line/column status text into an accessible button whose
    action opens the same panel.
- New `native/app/source_navigation.go`
  - Strict parser, exact range validation, open/cancel/submit helpers.
  - Use `ui.TextInput` bound to `GoToLineQuery`, with one-time `.AutoFocus()`
    and a generation-scoped Context.Key set before construction.
  - Use its `OnTextKey` for unmodified Enter and Escape; return `true` after
    handling either key. This runs before normal input submission, while MyGo
    keeps IME composition keys private, so candidate-selection Enter is not a
    navigation submit.
- New `native/app/source_navigation_test.go`
  - Parser/range unit tests plus focused App tests through `ui.NewTester`.
- A tiny optional `editor.Composer` interface and rich-view implementations
  expose composition state to prevent focus changes while a draft is composing.
- No production changes to `native/core/text`, format semantics, build scripts
  or dependencies.

This is the neutral current editor API; no new shared navigation contract is
justified.

## Input and coordinate contract

Grammar is exactly `line` or `line:column`. Trim only surrounding spaces.
Both fields are positive ASCII decimal integers; omitted column means `1`.
Reject empty input, signs, zero, an empty field after `:`, extra separators,
non-digits and integer overflow. There is no clamping.

Coordinates are 1-based source lines and 1-based UTF-16 code-unit columns,
matching the existing status bar. CR and LF are not addressable content
columns. A surrogate pair occupies two columns, and a column inside that pair
is rejected. Columns between combining code points remain addressable code-unit
positions; SetPosition does not normalize entire grapheme clusters. Panel help
states that columns are UTF-16 units. Leading zeroes are rejected explicitly.

After a successful draft Flush, resolve against that final `text.Buffer`:

1. `span, err := buffer.LineRange(line-1)`.
2. Let `units = span.End()-span.Start()`; require `1 <= column <= units+1`.
3. Set `offset = span.Start()+column-1`.
4. Call `buffer.SliceUTF16(offset, offset)` and reject `text.ErrSurrogate`.
5. Only after all checks succeed, call
   `t.Editor.SetPosition(line-1, column-1)` and then `t.Editor.Focus()`.

`LineRange` excludes CRLF and exposes an empty final line after a final
newline. The empty slice is the surrogate-boundary guard before the existing
`SetPosition` path, whose own clamp/UnmarkText behavior is therefore reached
only with already validated final coordinates.

## Lifecycle and focus

Opening captures the current tab/doc/editor/buffer, clears the query/error, and
does not Flush, write, change mode/selection, release a presentation or add
undo history. The source input gets focus and the panel help says navigation
targets the source text, including JSON/XML.

On submit:

1. Parse; syntax errors stay local and do no work on the document.
2. Recheck that the captured tab is current and both tab and editor still use
   the captured buffer.
3. Call existing `flushTab(t, "转到源码行")` once. On failure retain the panel,
   query, captured target and pending view/content; no mode, selection or
   navigation change occurs.
4. Recheck current tab/buffer identity, then perform the final-buffer range and
   surrogate validation above. Invalid final coordinates retain the panel and
   query and report locally.
5. Source mode stays source; split stays split; preview changes to source by
   releasing the rich presentation and setting `Mode = "source"`. Do not call
   `setMode`, because that would Flush a second time.
6. Reposition/focus the source editor and close the panel on success.

Parser/range errors use only `GoToLineError`, not global issue history. A real
Flush failure keeps the existing global diagnostic and mirrors a concise
failure in the panel. Cancel clears the panel and returns focus only to a
visible source editor in source/split mode; it never changes a preview's mode.
Tab switch/close and other committed active-tab changes close this transient
panel and clear its tab/buffer pointers, avoiding stale presentation state.
Existing modal/export/package behavior remains authoritative; opening the
full-pane plugin/package surface closes this inline panel.

Readonly source navigation uses the same selection/reveal path. Byte-inspection
tabs navigate the displayed, read-only hexadecimal logical lines, with panel
text explicitly identifying the target as the read-only byte view rather than
decoded document text.

## Test matrix

- Parser: both forms, surrounding spaces, omitted-column default, empty/zero/
  sign/separator/non-digit/overflow rejection.
- Range: empty lines, empty final line, CRLF boundary, end-of-line position,
  `ErrSurrogate`, and no clamping.
- App input: `Ctrl+G`, focused input, Enter submit and Escape cancel through
  `ui.NewTester`; assert final selection, focus, mode and unchanged history.
- Discoverability: clicking the current line/column status button opens the
  panel.
- Draft barrier: use a test `Flusher` that changes the buffer so coordinates
  are interpreted after Flush; use `failingView` to verify failed Flush keeps
  panel/query/view/selection/history and does not change mode.
- Readonly source and byte-inspection tabs.
- Tab switch and close while the panel is open, including cleared target
  references.
- A segmented long line with reduced `Editor.ShapeLimit`, navigating near its
  tail through `LineRange` without a whole-line/whole-file conversion.
- Required 760x480 light/dark tester capture and bounds check for the added
  panel, reusing `captureAcceptanceTester`. This is not physical OS/IME
  verification; no physical IME test is planned.

After activation, run the focused new tests, then the scoped existing App suite
and `vet ./app` through
`native/scripts/go.ps1 -GoArguments @(...)`. No release build or full app launch
is part of this feature's implementation check.
