# Markdown task-checkbox app acceptance

## ACTIVE — app test plan before dependent UI work

Acknowledged 2026-11-08. Read `native/AGENTS.md`,
`native/markdown_tasks_stage_plan.txt`, and the approved additive contract in
`native/handoffs/markdown-tasks-contract.md`.

Ownership is limited to `native/app/**` plus this app/integration evidence.
GLM owns `formats/markdown/**`; Astra owns `modules/markdownview/**`. Installed
plugins, graph previews, search/document/session, editor and parser product
code remain frozen. No shell change is planned until a real-adapter app test
demonstrates a composition defect.

## Compact app acceptance plan

These cases run against the actual registered Markdown adapter and native
MyGo UI. Parser/export matrices, stale-journal mapping, duplicate-label
resolution, target changes/deletion, fence context, far-list virtualization and
checkbox command validation remain GLM/Astra responsibilities. This app layer
verifies the finite end-to-end workflows that require the real shell.

1. **Real preview control and accessibility**
   - Open a compact BOM/CRLF Markdown fixture containing unchecked `[ ]`,
     checked `[x]`/`[X]`, ordered and Unicode-label tasks.
   - Assert only agreed real accessibility names/roles/states are exposed, with
     positive on-screen geometry and current source-derived checked state.
   - Exercise pointer activation and focus plus Space from the real app view. A
     control must not rely on a test-only mutation shortcut.

2. **Exact source, save and view agreement**
   - Toggle unchecked to lowercase `x`, and checked `x`/`X` to ASCII space.
   - Compare complete files and require a one-byte diff while brackets, list
     marker, numbering, indentation, CRLF, Unicode, trailing whitespace and
     neighboring bytes remain exact.
   - Save through the app and reopen: preview, source and split views report the
     same state and the saved bytes match the in-memory source.
   - Undo and redo use one transaction each, preserve the source selection and
     direction, and update preview/source/split without an extra save edit.
   - Opening and exporting without interaction leaves source bytes and dirty
     state unchanged.

3. **Readonly behavior**
   - Set `Document.ReadOnly` before configuring/restoring the tab. The current
     host captures readonly at view construction; no unsupported live field-only
     transition is asserted as a product defect.
   - Require visible controls to be disabled both visually and through MyGo's
     accessibility tree (`AccessInfo.Disabled`). Pointer and keyboard attempts
     must leave version, history and source bytes unchanged.
   - Writable Save As of a readonly document remains writable after the existing
     editor readonly synchronization.

4. **Pending native composition barrier**
   - Start a real Markdown Compose draft in a different block, then activate the
     task checkbox from the real preview control.
   - A successful Flush completes first; the draft and then the marker change
     have their documented transaction/notification behavior.
   - Include one app-visible failed mode/view boundary if Astra publishes such a
     reachable path; stale-journal, duplicate and target-context refusals remain
     in Astra's focused ownership.

5. **Mode integration**
   - Verify source mode renders no checkbox; preview and split route activation
     through the same real adapter.
   - After toggle and Undo/Redo, the source editor and split preview agree with
     the same authoritative buffer.

6. **Narrow visual acceptance**
   - Capture one representative narrow preview at `760x480` containing checked
     and unchecked controls plus a bounded long label. Choose light/dark after
     Astra's contrast checkpoint; do not duplicate the broader visual matrix.
   - Assert tab/mode/status geometry, checkbox hit targets and accessibility
     bounds before capture. The image is for root review.

## Contract coordination requested

The app needs Astra to publish these concrete UI semantics with the view
checkpoint:

- exact accessible names for unchecked, checked, empty and clipped labels
  (proposed `未完成任务：<label>` / `已完成任务：<label>`, with the view's
  bounded `…` suffix);
- whether the control is `CheckboxBase` plus a custom themed box or styled
  `Checkbox`;
- disabled behavior for readonly/partial states and the role/accessibility
  fields tests should assert;
- the supported keyboard path (focus then Space) and pointer activation target;
- how a refused toggle reports through `Host.Report`/app status.

GLM has published the additive `TaskMarker` shape in current source, but its
focused parser/export checkpoint is not yet published. At this plan revision,
`formats/markdown` compiles and its existing tests pass. App code waits for the
peer checkpoints instead of depending on source that may still change.

No app product code was changed. The accepted15:12 binary/screenshots/smoke
remain historical for this stage; full integration will run only after root
freezes reviewed focused checkpoints.

## Focused app implementation checkpoint

Added `native/app/markdown_tasks_app_test.go` against the actual registered
Markdown adapter and `ui.NewTester`. It covers:

- compact UTF-8 BOM + CRLF source with unchecked, lowercase/uppercase checked,
  ordered, empty, bold/Unicode and neighboring tasks;
- exact role/name/tooltip/checked/disabled/focusable/bounds assertions;
- box pointer activation and focus-plus-Space activation;
- exact one-middle-marker-byte source diffs, reversed selection preservation,
  one undo/redo transaction, explicit save, BOM/CRLF preservation and reopen;
- preview/source/split lifecycle, including a second checkbox activation in
  split;
- readonly configured before mount, unchanged refused activation, writable
  Save As control, and another-block native Compose -> activation barrier;
- unchanged open/export bytes and dirty state;
- an opt-in `760x480` dark screenshot with chrome/checkbox geometry checks.

Compile-only check:

```powershell
native\scripts\go.ps1 -GoArguments @('test','./app','-run','^$','-count=1')
```

PASS: `levnote.local/native/app 1.458s [no tests to run]`.

Pointer/undo/redo/save/views and Space/readonly focused tests:

```powershell
native\scripts\go.ps1 -GoArguments @('test','./app','-run','TestMarkdownTaskCheckboxRealAdapter(PointerUndoRedoSaveAndViews|SpaceAndReadonly)$','-count=1','-timeout','120s','-v')
```

PASS: `levnote.local/native/app 1.392s`.

The following focused app regression previously exposed a Markdown view stale-row
defect:

```powershell
native\scripts\go.ps1 -GoArguments @('test','./app','-run','TestMarkdownTask','-count=1','-timeout','120s')
```

`TestMarkdownTaskCheckboxRealAdapterFlushesOtherBlockCompose` establishes a real
native Compose draft in the heading, activates the unchecked task box, and
observes the required source (`# 中文🙂` plus `[x]`). After executing one Undo,
the authoritative buffer correctly returns to `[ ]`, but the accessibility tree
still exposes `已完成任务：未完成`; `未完成任务：未完成` is absent. The open clean
draft editor is still mounted. This agrees with `View.refresh`: after
`rebaseDraft()` maps the unrelated undo and returns true, the early
`v.draft != nil && v.loaded` return keeps the old parsed checkbox rows until
the draft is dismissed. The same command therefore fails at
`markdown_tasks_app_test.go:395`.

Routed review request: after a clean/mapped external host-version change while
a draft remains open, invalidate the cached parsed rows so source, preview and
split agree immediately; do not discard or rebind the clean draft and do not
fix this in app.

The failing case was re-run directly against the current tree after formatting.
It failed at the same post-Undo accessibility assertion; the source and Undo
behavior before that assertion remained correct:

```powershell
native\scripts\go.ps1 -GoArguments @('test','./app','-run','TestMarkdownTaskCheckboxRealAdapterFlushesOtherBlockCompose$','-count=1','-timeout','120s','-v')
```

FAIL: `levnote.local/native/app 1.314s`; the tree still contained
`已完成任务：未完成` and omitted `未完成任务：未完成`.

## Latest focused evidence — narrow visual

After reducing only the representative fixture's repeated label from 30 to 20
fragments and removing its duplicate empty row, the actual app screenshot and
geometry assertions pass:

```powershell
$out = 'F:\AIproject\LevNote\native\build-cache\markdown-tasks-acceptance-20261108'
$env:LEVNOTE_MARKDOWN_TASKS_SCREENSHOT_DIR = $out
native\scripts\go.ps1 -GoArguments @('test','./app','-run','TestMarkdownTaskCheckboxesNarrowScreenshot$','-count=1','-timeout','120s','-v')
```

PASS: `levnote.local/native/app 1.455s`.

Output:

`native/build-cache/markdown-tasks-acceptance-20261108/markdown-tasks-narrow-dark-760x480.png`

The image was inspected at its actual `760x480` size. It shows three checked
and unchecked controls plus the bounded long Unicode label, preserves the tab
and mode-control rows above the status region, wraps the long task naturally,
and leaves the `y=451..480` status row unobscured. The test also asserts every
shown checkbox and chrome label has positive in-bounds geometry and that the
app status text is inside the bottom row.

The failed-Flush regression is covered by the focused app test:

```powershell
native\scripts\go.ps1 -GoArguments @('test','./app','-run','TestMarkdownTaskCheckboxRealAdapterFailedFlushRetainsDraftAndSource$','-count=1','-timeout','120s','-v')
```

PASS: `levnote.local/native/app 1.290s`. It uses the same direct overlapping
host edit as the existing real-adapter app integration test
`TestNarrowMarkdownConflictKeepsViewOutOfStatusRow`, clicks the real checkbox,
and verifies unchanged buffer/version/selection/history, retained tab/view and
file, plus the visible `草稿仍保留` and recovery controls.

At that intermediate point the clean-draft stale-row defect still blocked the
stage freeze; the resolution and current evidence follow below. No full suite,
release build, or replacement of the historical accepted binary has been run.

## Clean-draft refresh regression resolved

Astra fixed the owned Markdown view refresh path. A successful mapped external
host-version change now invalidates parsed preview rows when the independent
draft is clean, retains the field editor/ranges, and detaches the old editing
index before reparse. Astra's full focused view command passed:

```powershell
native\scripts\go.ps1 -GoArguments @('test','./modules/markdownview','-count=1','-timeout','120s')
```

PASS: `levnote.local/native/modules/markdownview 1.621s`.

The exact previously failing app regression now passes:

```powershell
native\scripts\go.ps1 -GoArguments @('test','./app','-run','TestMarkdownTaskCheckboxRealAdapterFlushesOtherBlockCompose$','-count=1','-timeout','120s','-v')
```

PASS: `levnote.local/native/app 1.490s`.

The consolidated real-adapter suite passes as well; the screenshot test is
normally skipped without its opt-in output directory:

```powershell
native\scripts\go.ps1 -GoArguments @('test','./app','-run','TestMarkdownTask','-count=1','-timeout','120s','-v')
```

PASS after Astra's final view freeze: `levnote.local/native/app 1.401s`. All
four real-adapter tests passed;
`TestMarkdownTaskCheckboxesNarrowScreenshot` skipped only because
`LEVNOTE_MARKDOWN_TASKS_SCREENSHOT_DIR` was intentionally unset.

The screenshot was regenerated after the view fix:

```powershell
$out = 'F:\AIproject\LevNote\native\build-cache\markdown-tasks-acceptance-20261108'
$env:LEVNOTE_MARKDOWN_TASKS_SCREENSHOT_DIR = $out
native\scripts\go.ps1 -GoArguments @('test','./app','-run','TestMarkdownTaskCheckboxesNarrowScreenshot$','-count=1','-timeout','120s','-v')
```

PASS after Astra's final view freeze: `levnote.local/native/app 1.351s`.
Current image evidence:

- Path:
  `native/build-cache/markdown-tasks-acceptance-20261108/markdown-tasks-narrow-dark-760x480.png`
- Size: `41,841` bytes
- SHA-256:
  `D9EE88C32A220CF730A4B8C458A508E2FC268B29BB101C5D75CA4FAEA3393143`
- Timestamp: `2026-11-08T15:43:10.7782066+08:00`

`native\scripts\go.ps1 -GoArguments @('vet','./app')` also passes. Root
visually accepted the dark narrow task rows and control geometry.

Astra's final view checkpoint changed the readonly instruction to `只读预览`;
its complete focused view suite passed in `1.846s` with vet clean. The app
focused suite and screenshot were then rerun once as recorded above. GLM's
current-source review subsequently found no actionable defect, and root froze
the product and authorized the final integration.

## Final integration checkpoint — accepted

The single authorized final sequence completed:

```powershell
native\scripts\go.ps1 -GoArguments @('test','./...','-count=1','-timeout','300s')
native\scripts\go.ps1 -GoArguments @('vet','./...')
native\scripts\go.ps1 -GoArguments @('build','-trimpath','-buildvcs=false','-ldflags','-s -w -H=windowsgui','-o','bin\levnote-native.exe','./cmd/levnote')
```

- Full test PASS, exit0,6.399s wall; every native package passed.
- Full vet PASS, exit0,1.083s wall, no diagnostics.
- Release GUI build PASS, exit0,1.480s wall.

Final release binary:

- `native\bin\levnote-native.exe`;
- `13,227,008` bytes;
- SHA-256
  `8D71EC6AEF8F0DAE91C125AB87F234B439FEBF111712CEFC79751692412175ED`;
- UTC mtime `2026-11-08T07:48:42.2959080Z`;
- Go1.27.3 Windows amd64/v1, CGO1, trimpath, MyGo0.8.3;
- PE32+ x64 Windows GUI subsystem2.

Final evidence directory:
`native\artifacts\acceptance\markdown-tasks-20261108-154805\`. Its
`result.json` records all test/vet/build/smoke fields and artifact hashes. The
directory also contains the current binary copy, root-accepted screenshot, raw
test output, and the preserved prior accepted binary. Vet and build emitted no
bytes, so their zero-byte logs are explicitly marked in `result.json` as
post-run placeholders rather than raw captures; `OUTPUT-PROVENANCE.txt` records
the distinction.

Hidden isolated smoke:
`native\artifacts\acceptance\startup-smoke-markdown-tasks-20261108-154842\`.
PID-scoped EnumWindows found `轻笺 · 原生版` with the `Start-Process
-WindowStyle Hidden` launch still invisible, `WM_CLOSE` exited0, and two
unrelated updater-shaped sentinels survived with exact contents. Release and
copied hashes match; `result.json` includes UTC start/finish times.

No product source was changed for the final run. Root accepted this15:48
artifact at15:50 after reading the full result, test log, and smoke JSON. The
15:12 binary/screenshot/smoke is historical and remains preserved under
`previous-accepted-oversize.exe`.
