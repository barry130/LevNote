# Release 10 native UI feasibility plan

2026-11-09. Prepared by the second DeepSeek owner. **Not executed and not
authorized to execute until root accepts `.10` and explicitly activates this
plan.** Product source, tests, builds, binaries and user configuration remain
unchanged during preparation.

Inputs read:

- `native/AGENTS.md`
- `native/handoffs/release10-ui-feasibility-task.md`
- `native/handoffs/resource-workload-next-plan.md`
- `native/handoffs/resource-workload-next-root-review.md`
- Computer Use `SKILL.md`, `docs/guidance.md`, `docs/api.md` and
  `docs/confirmations.md` below
  `C:/Users/canace/.codex/plugins/cache/openai-bundled/computer-use/26.1103.32239`

The actual UI control labels in the accepted app are `阅读 / 编辑`, `源码`,
`分屏`, `导出`, and `取消导出`. The window title is `轻笺 · 原生版`. The
command line accepts file paths after the executable path.

## Activation gate and fixed scope

Execution starts only after root supplies all of these:

1. An explicit activation after reading the real `.10` delivery result.
2. The exact accepted `.10` package directory, executable, size and SHA-256
   inventory.
3. Confirmation that the package is complete and remains the accepted `.10`
   candidate.

If any item is absent or a package hash disagrees with the accepted inventory,
stop before process creation and report the discrepancy. Do not substitute
`.9`, `.8`, a source build or a hidden-startup helper.

This activation permits exactly one real-window pilot run. It does not activate
memory, CPU, working-set or six-run resource measurement. It does not permit
product changes, test-only hooks, builds, test reruns or a Windows UI Automation
helper.

## Prerequisites and capability gate

After activation, perform these checks in order without launching the app:

1. Create a fresh, absent run directory:
   `native/artifacts/acceptance/release10-ui-feasibility-<UTC timestamp>`.
2. Copy the entire accepted `.10` package into that directory. Hash every copied
   file before and after the run. This run-owned copy is the only executable
   tree used by the pilot.
3. Create fresh run-owned `APPDATA` and `LOCALAPPDATA` directories below the run
   directory. The resulting config directory must be the run-owned
   `APPDATA/LevNoteNative`; never use the real `%APPDATA%` or
   `%LOCALAPPDATA%`.
4. Confirm the active PowerShell `Start-Process` supports
   `-Environment` and `-WindowStyle Hidden`. If it does not, stop and ask root
   for a design decision. Do not launch first and try to isolate afterward.
5. Initialize Computer Use exactly once in a fresh `node_repl` session:

   ```js
   if (!globalThis.sky) {
     const { sky } = await import("@oai/sky");
     globalThis.sky = sky;
   }
   ```

   Then make one lightweight `sky.list_apps()` or `sky.list_windows()` call to
   prove the helper is available. If it fails, retry once after two seconds;
   if the lightweight call still fails, reset the JavaScript session if
   available, initialize once more, retry once, then stop without launching.

`sky.launch_app({ app })` accepts only an app identifier/path; it has no argv or
environment option. The accepted launch split is therefore:

- ordinary process creation through the existing execution shell for isolated
  environment and the two fixture argv entries;
- every window observation and action through `@oai/sky`, using only returned
  app/window objects.

PowerShell may create and inspect only the exact run-owned process. It must not
enumerate windows, use UI Automation, send messages, call `CloseMainWindow`, or
guess window handles.

## Deterministic fixtures

Create two non-private Markdown fixtures, each exactly **30 KiB (30,720 bytes)**:

- `fixtures/ui-feasibility-a.md`
- `fixtures/ui-feasibility-b.md`

Both are generated ASCII with LF line endings, no BOM and no trailing newline.
Each contains a unique heading, paragraphs, an unordered list, a fenced code
block, and a short table. Remaining bytes are deterministic short ASCII lines or
padding; no line approaches the editor's long-line segmentation threshold.

Before launch:

- assert each initial byte length is exactly 30,720;
- record SHA-256 for both fixtures;
- create `evidence/expected-a-after.md` as the exact initial bytes of A followed
  by the seven ASCII bytes `EDIT_OK`, with no newline or whitespace;
- record the expected post-edit length and SHA-256;
- retain unchanged copies of both initial fixtures for byte comparisons.

The UI edit will append `EDIT_OK` at the exact end of source A. Successful save
therefore requires:

```text
final(A) == initial(A) + ASCII("EDIT_OK")
final(B) == initial(B)
```

No user document, screenshot, clipboard, token, credential or unrelated file
enters the fixtures or export.

## Process creation and exact ownership

Create the process only after the helper and isolation checks pass. The command
shape is:

```powershell
$proc = Start-Process `
  -FilePath (Join-Path $packageCopy 'levnote-native.exe') `
  -WorkingDirectory $packageCopy `
  -ArgumentList @($fixtureA, $fixtureB) `
  -PassThru `
  -WindowStyle Hidden `
  -Environment @{
    APPDATA = $runAppData
    LOCALAPPDATA = $runLocalAppData
  }
```

Immediately persist the exact process identity in evidence: PID, UTC creation
time, executable path, parent PID, command arguments, working directory,
`APPDATA`, `LOCALAPPDATA`, package-copy path, and run-directory path. All later
process checks must match both PID and creation time. Never identify the process
by executable name alone.

Passing the two fixture paths on argv is deliberate and disclosed: it prepares
the initial two tabs only. All subsequent tab, view, edit, save, export and close
actions are real UI input. `-WindowStyle Hidden` is only a safe launch state; it
is not counted as a visible-window result. The app must be selected and activated
through Computer Use before any interaction claim.

## Real-window selection and action discipline

Refresh `sky.list_apps()` for a short bounded startup window. Select exactly one
returned app/window whose app identity matches the run-owned executable path or
whose returned title is `轻笺 · 原生版` and whose process identity is already
proven exact. If zero or more than one candidate remains, stop; do not guess.

Rehydrate and activate the selected window:

```js
globalThis.targetWindow = await sky.get_window({
  id: returnedWindow.id,
  app: returnedWindow.app,
});
await sky.activate_window({ window: targetWindow });
globalThis.state = await sky.get_window_state({
  window: targetWindow,
  include_screenshot: true,
  include_text: true,
});
globalThis.targetWindow = state.window;
```

The first captured state must visibly show the two fixture tabs. Record the
returned window `id`, app identity, title, and whether the screenshot was
observed. Inspect the screenshot directly; do not decode, save or re-emit its
data URL.

Use one observe/action/refresh transaction per input:

1. Observe the latest state and stop to inspect it.
2. Choose exactly one returned accessibility element or one point in the latest
   returned screenshot.
3. Perform one `click`, `press_key`, `type_text` or `scroll`.
4. Refresh immediately and retain the new returned window object.

Indexes, screenshot IDs and coordinates expire after every action. If an input
or refresh outcome is unknown, do not blindly repeat it: capture a fresh state,
determine whether the intended effect occurred, and stop if that cannot be
established. If a modal or save dialog is expected, use `sky.list_windows()`
and require exactly one newly returned owned dialog; do not derive a handle from
a title or PID alone.

## One pilot sequence

Allow at most ten minutes and one app launch. Timestamps are diagnostic
boundaries only, never performance samples.

1. **Initial state.** Record the two tab titles and active tab. Both must be
   visible and the window must be foreground and capturable.
2. **Select A.** Click the tab labeled `ui-feasibility-a.md`; refresh and verify
   A is active.
3. **Exercise reading and source controls.**
   - If A starts in reading mode, click `源码`; otherwise click
     `阅读 / 编辑`. Refresh and record the mode change.
   - End in source mode by clicking `源码` as needed. Refresh and verify the
     source control is active and the editor is visible.
4. **Focus the editor.** Click a stable point in the observed source work
   surface and refresh. Typing is permitted only when `focused_element`, selected
   text, or the screenshot establishes that the source editor has focus. If
   focus cannot be established, stop before typing and report the exact state.
5. **Make the known edit.** Press `Control_L+End`, refresh, then use
   `type_text` with exactly `EDIT_OK`. Refresh and verify the marker is visible
   at the source end.
6. **Save.** Press `Control_L+s` and refresh. Observe until one of these is true,
   using at most three two-second observations:
   - the visible state reports the document saved / not dirty;
   - the exact on-disk bytes match `expected-a-after.md`.

   Then read A and B with ordinary file reads. Require exact byte equality for
   both formulas above. A visible status alone is insufficient; file bytes alone
   do not establish that the UI save path ran.
7. **Switch tabs and views.** Click B, refresh, switch to `阅读 / 编辑` and back
   to `源码`, refreshing after each action. Click A, refresh. Click B, refresh.
   Verify the active title and that neither switch changes A or B on disk.
8. **Export A, conditionally.** Click A, refresh, click the observed `导出`
   control, and refresh. Use a new absent run-owned path
   `exports/ui-feasibility-a.html`.
   - If the save destination can be identified and filled unambiguously, save
     once. A modal path is acceptable only after a fresh focus check identifies
     the filename field; replace its value with the full run-owned path without
     touching other directories.
   - After submission, refresh until the app reports `已导出 ...`, using at
     most three two-second observations.
   - Verify the new HTML file is non-empty, contains the known fixture heading
     text, contains `EDIT_OK`, and differs from both Markdown files.
   - If any overwrite/delete/security/authentication prompt appears, cancel and
     report export as blocked. Never accept a replacement prompt.
   - If the dialog cannot be identified and driven reliably, cancel it through
     the observed app UI and record `export feasibility: unknown`. Do not retry
     with coordinates from an older state.
9. **Close through the app UI.** Return to the main window. Close each remaining
   tab with `Control_L+w`, refreshing after each close and verifying that the tab
   count decreases. Then press `Alt_L+F4` on the returned app window and refresh.
   A clean-close result requires the exact owned main process to exit normally
   with exit code `0`; record the actual exit code.

Do not close by `Stop-Process`, `CloseMainWindow`, `taskkill`, window messages or
a custom helper unless the bounded failure cleanup below is invoked. A forced
cleanup is never reported as a successful clean UI close.

## Evidence to publish

Publish a compact result under the run directory and a short handoff at
`native/handoffs/release10-ui-feasibility-result.md`. It must contain:

- activation source and accepted `.10` version/package hashes;
- run root, copied package root, fixture paths, config paths, export path, and
  SHA-256 for fixtures and expected or produced files;
- returned app/window identity and every observed UI action in sequence;
- initial/final tab state and source/reading transitions;
- exact byte-comparison results for A and B;
- export bytes/content checks or an explicit `unknown`;
- main process PID, creation time, executable path, parent PID, exit method and
  actual exit code;
- confirmation that the package inventory was unchanged and that all config
  writes remained inside the run-owned directories;
- bounded failures and retries exactly as observed, with no fabricated timing,
  screenshot, worker, memory or CPU result.

Do not retain screenshot data URLs solely for evidence. The result can state
which returned screenshots were inspected; it must not call headless test
images or a hidden startup a physical interaction.

## Failure and cleanup bounds

- Lightweight helper timeout: wait two seconds, retry once. If it fails again,
  reset/initialize/retry once, then stop.
- Window selection or state capture failure: discard stale indexes and
  coordinates, reselect a fresh returned window, retry the observation once.
  If that fails, stop.
- A locked desktop, security prompt, authentication UI, terminal prompt, or an
  ambiguous modal is an immediate stop; do not automate it.
- A save/export overwrite prompt is not accepted. Cancel and preserve the prior
  target.
- If the app does not exit after the real UI close attempt, verify the exact
  run-owned PID and creation time, then use one guarded `Stop-Process` only on
  that exact process as cleanup. Mark the close as forced/failed, retain the
  evidence and report any possible descendant uncertainty rather than claiming
  clean shutdown.
- If helper failure prevents any window action after launch, the same guarded
  process cleanup may remove only the exact run-owned main process. Never sweep
  processes by name or touch another LevNote/Codex process.
- Never delete the run directory, accepted package, fixtures or failed evidence
  during the pilot. A later root decision owns any cleanup through UI or
  filesystem action.

## Exit criteria and deferred work

The pilot passes only when all of the following are true:

- the real returned window was observed and activated;
- both generated tabs were selected;
- reading and source modes were exercised;
- the public UI edit and `Ctrl+S` produced exactly the declared A bytes;
- B remained byte-identical;
- tab switching did not alter either file;
- either a new export was verified or export is honestly marked unknown;
- the app closed through UI input with normal exit code `0`;
- package files, run isolation and user-state preservation checks all pass.

This pass establishes only that the real-window workload is driveable and
correct. One run does not establish reliability across runs, physical IME
composition, GPU behavior, resource reduction or a Tauri comparison.

Only after root reviews and separately activates the measurement stage may the
existing resource plan continue. That stage remains limited to workload C,
repeated lifecycle, with one `.8`/accepted-candidate pilot pair followed by one
AB/BA set: six total runs. No resource values, thresholds or claims belong in
this feasibility checkpoint.
