# Resource stage handoff

## CURRENT RESOURCE FOLLOW-UP RESULT — 2026-11-08 17:42 +08:00

The original hidden matrix and its clean-restore continuation are separate
facts. Read the canonical
`handoffs/resource-followup-app.md` section **FINAL RESOURCE CHECKPOINT —
2026-11-08 17:42 +08:00** for complete method, hashes and limitations.

- Six rows yielded ten `samples.observations[]` samples each. Exact
  `workingSet64`/`privateMemorySize64` means in MiB are: blank
  58.453125000/86.074218750, plain 61.824218750/88.785156250, Markdown
  63.785156250/90.382812500, mindmap 59.250000000/86.566406250, draw.io
  59.035156250/87.082031250, ordered-four 64.503906250/91.824218750.
- The overall original result remains **FAIL**: no pre-close session appeared
  within the declared 20-second hidden observation bound. The evidence does
  not establish whether persistence occurs under other timing or interaction.
  Automatic live-hidden
  persistence was not observable in the hidden seed window. That failure is
  preserved unchanged.
- The bounded same-app-data continuation passed product-loader validation,
  10 observations, `WM_CLOSE` exit 0, second validation, unchanged fixtures,
  identical session hash and zero remaining app processes. The primary 17:40
  continuation has 10 observations and means of 70.382812500 MiB working set /
  96.679687500 MiB private bytes, with zero CPU delta over 10.142 seconds.
- Scope is the hidden startup main process only. No foreground visibility,
  GPU paint, or automatic live-hidden persistence is claimed.

The first continuation is primary and is preserved together with its logs and
script under `followup-clean-restore-attempt-01-no-logical-cpu/`. Its delayed
metadata correction arrived after launch, so the 17:42 run is supplemental and
explicitly an unplanned repeat: 10 observations with 71.730078125 MiB working
set / 96.453125000 MiB private bytes, zero CPU delta over 10.182 seconds, and
16 logical processors. It is not averaged with the primary restore. No launch
or sample occurred after the repeat.

Current artifact:
`native/artifacts/acceptance/resource-followup-20261108-165230/`. The accepted
13,252,608-byte link executable was reused unchanged at SHA-256
`5BFB5A445CC619FEA9380F0363D3DF26EC4F3FF944C8549B1992F5B462EF3F10`.

## Current accepted runtime checkpoint — 2026-11-08 12:56 +08:00

This replaces the older pending-worker notes below. Source is frozen after the
full test/vet/build gates recorded in `handoffs/integration.md`; both resource
scenarios ran the final `13,084,160`-byte executable with SHA-256
`8ADB375D3AE400B57AA07C73E441D4A13DACE7ADC7124F5F0F38DD457FDC98B9`.

Fixture:
`native/artifacts/acceptance/resource-sample-20261108-1256`.

Method and scope:

- Each scenario waited 10 seconds after window readiness, then collected 10
  samples at approximately one-second intervals.
- Values are for the main process only.
- Working set and private bytes are reported separately; they are not the
  same metric.
- CPU deltas use all 16 logical processors. A one-core percentage and a
  percent-of-16-logical-processors figure are both recorded so the basis is
  explicit.
- These are OS process measurements, not Go/tester allocation counters.
  They cannot be compared directly with Tauri without an identical workload
  and matching measurement scope.

Blank idle, with no command-line documents:

```text
Window: 轻笺 · 原生版
Sample interval: 10.179s
Working set avg/peak/final: 59.25 / 59.25 / 59.25 MiB
Private bytes avg/peak/final: 86.50 / 86.50 / 86.50 MiB
CPU delta: 0.0000s
CPU: 0.000% one core; 0.000% of 16 logical processors
Exit code: 0
```

Four representative documents (`source.go`, `notes.md`, `ideas.mindmap`, and
`flow.drawio`):

```text
Window: 轻笺 · 原生版
Sample interval: 10.148s
Working set avg/peak/final: 86.24 / 87.11 / 87.11 MiB
Private bytes avg/peak/final: 109.48 / 109.57 / 109.57 MiB
CPU delta: 0.1562s
CPU: 1.540% one core; 0.096% of 16 logical processors
Exit code: 0
```

Both processes closed with exit code `0`. The worker/list stage remains bounded
as described below: coalesced generation-checked disk/tree workers, immutable
disk tokens, shutdown-aware callbacks, the explicit started/completed
`updateWait` handshake, actual-read-limited search, and virtualized list rows
with stable selection/expansion identity.

### Preserved runtime evidence detail

The successful sampling command was recovered from the task execution record
without rerunning it. It is reproduced exactly below:

```powershell
function Measure-LevNote([string]$label, [string[]]$openPaths) {
  $root = 'F:\AIproject\LevNote\native\artifacts\acceptance\resource-sample-20261108-1256'
  $exe = Join-Path $root 'bin\levnote-native.exe'
  $appData = Join-Path $root ("appdata-{0}" -f $label.ToLowerInvariant())
  $workspace = 'F:\AIproject\LevNote\native\artifacts\acceptance\runtime-fixture\workspace'
  New-Item -ItemType Directory -Path $appData -Force | Out-Null
  $env:APPDATA = $appData
  $env:LOCALAPPDATA = $appData
  if ($openPaths.Count -gt 0) {
    $process = Start-Process -FilePath $exe -ArgumentList $openPaths -WorkingDirectory $workspace -PassThru
  } else {
    $process = Start-Process -FilePath $exe -WorkingDirectory $workspace -PassThru
  }
  $deadline = [DateTime]::UtcNow.AddSeconds(20)
  $title = ''
  while ([DateTime]::UtcNow -lt $deadline) {
    Start-Sleep -Milliseconds 200
    $process.Refresh()
    if ($process.HasExited) { break }
    $title = $process.MainWindowTitle
    if ($title -eq '轻笺 · 原生版') { break }
  }
  if ($title -ne '轻笺 · 原生版') { $process.Kill($true); $process.WaitForExit(); throw "resource window did not become ready for ${label}: '$title'" }
  Start-Sleep -Seconds 10
  $process.Refresh()
  $cpuStart = $process.TotalProcessorTime.TotalSeconds
  $sampleStart = [DateTime]::UtcNow
  $ws = [System.Collections.Generic.List[long]]::new()
  $private = [System.Collections.Generic.List[long]]::new()
  for ($i = 0; $i -lt 10; $i++) {
    Start-Sleep -Seconds 1
    $process.Refresh()
    $ws.Add($process.WorkingSet64)
    $private.Add($process.PrivateMemorySize64)
  }
  $sampleEnd = [DateTime]::UtcNow
  $process.Refresh()
  $cpuDelta = $process.TotalProcessorTime.TotalSeconds - $cpuStart
  $elapsed = ($sampleEnd - $sampleStart).TotalSeconds
  $closeRequested = $process.CloseMainWindow()
  if (-not $closeRequested -or -not $process.WaitForExit(15000)) {
    $process.Kill($true)
    $process.WaitForExit()
    throw "resource process did not close gracefully: $label"
  }
  if ($process.ExitCode -ne 0) { throw "resource process exit=$($process.ExitCode) for $label" }
  [pscustomobject]@{
    Scenario = $label
    OpenDocuments = $openPaths.Count
    WindowTitle = $title
    SettlementSeconds = 10
    SampleIntervalSeconds = [math]::Round($elapsed, 3)
    SampleCount = 10
    WorkingSetAvgMiB = [math]::Round(($ws | Measure-Object -Average).Average / 1MB, 2)
    WorkingSetPeakMiB = [math]::Round(($ws | Measure-Object -Maximum).Maximum / 1MB, 2)
    WorkingSetFinalMiB = [math]::Round($ws[$ws.Count - 1] / 1MB, 2)
    PrivateBytesAvgMiB = [math]::Round(($private | Measure-Object -Average).Average / 1MB, 2)
    PrivateBytesPeakMiB = [math]::Round(($private | Measure-Object -Maximum).Maximum / 1MB, 2)
    PrivateBytesFinalMiB = [math]::Round(($private[$private.Count - 1] / 1MB), 2)
    CpuSecondsDelta = [math]::Round($cpuDelta, 4)
    CpuPercentOneCore = [math]::Round(($cpuDelta / $elapsed) * 100, 3)
    CpuPercentSystem16Logical = [math]::Round(($cpuDelta / ($elapsed * [Environment]::ProcessorCount)) * 100, 3)
    ExitCode = $process.ExitCode
  }
}
$workspace = 'F:\AIproject\LevNote\native\artifacts\acceptance\runtime-fixture\workspace'
$results = @()
$results += Measure-LevNote 'blank-idle' @()
$results += Measure-LevNote 'four-documents' @(
  (Join-Path $workspace 'source.go'),
  (Join-Path $workspace 'notes.md'),
  (Join-Path $workspace 'ideas.mindmap'),
  (Join-Path $workspace 'flow.drawio')
)
$results | ConvertTo-Json -Compress
```

The exact successful aggregate output was:

```json
[{"Scenario":"blank-idle","OpenDocuments":0,"WindowTitle":"轻笺 · 原生版","SettlementSeconds":10,"SampleIntervalSeconds":10.179,"SampleCount":10,"WorkingSetAvgMiB":59.25,"WorkingSetPeakMiB":59.25,"WorkingSetFinalMiB":59.25,"PrivateBytesAvgMiB":86.5,"PrivateBytesPeakMiB":86.5,"PrivateBytesFinalMiB":86.5,"CpuSecondsDelta":0.0,"CpuPercentOneCore":0.0,"CpuPercentSystem16Logical":0.0,"ExitCode":0},{"Scenario":"four-documents","OpenDocuments":4,"WindowTitle":"轻笺 · 原生版","SettlementSeconds":10,"SampleIntervalSeconds":10.148,"SampleCount":10,"WorkingSetAvgMiB":86.24,"WorkingSetPeakMiB":87.11,"WorkingSetFinalMiB":87.11,"PrivateBytesAvgMiB":109.48,"PrivateBytesPeakMiB":109.57,"PrivateBytesFinalMiB":109.57,"CpuSecondsDelta":0.1562,"CpuPercentOneCore":1.54,"CpuPercentSystem16Logical":0.096,"ExitCode":0}]
```

The executed script retained the ten working-set and private-byte values only
in local arrays and emitted aggregates; those individual values were not
written to an artifact and therefore are not reconstructed here.

The four fixture files and exact byte sizes were:

| File | Bytes |
| --- | ---: |
| `source.go` | 86 |
| `notes.md` | 388 |
| `ideas.mindmap` | 336 |
| `flow.drawio` | 976 |

The blank scenario restored an empty session and had no active document.
`OpenMany` appends and activates each requested path in order, so the
four-document scenario ended with `flow.drawio` active. Its builtin adapter
sets startup mode to `preview`; the other three tabs were inactive. The tiny
fixture opens only these four files and does not construct a large workspace.

## Acknowledgement — 2026-11-08

- Owner: DeepSeek plugins/app agent.
- Read `native/resource_stage_task.txt`, `native/resource_workflows_task.txt`,
  `native/AGENTS.md`, and the current format/editor handoffs.
- Scope: `native/app/**`, app-facing use of the session/document contracts,
  plus resource tests and evidence. Astra owns `core/session/**`; GLM owns
  `core/document/**`; this agent integrates their published APIs and requests
  any missing contract through this handoff.
- Finite order: (1) encoded recovery preflight and failed-state retry control;
  (2) immutable disk tokens plus coalesced/cancellable disk and tree workers;
  (3) bounded visible-window tree/search rows using native list facilities.
- No child agents, commits, pushes, network activation, or engine ownership
  changes. Compact before 200K context.

## Current work

- Astra owns `core/session/**`; GLM retains `core/document/**`. The exact
  builder API in `handoffs/session.md` has landed and is integrated in app.
- The remaining disk-worker prerequisite is GLM's immutable disk token/check;
  it is not present in current `core/document` source.

### Refresh script verification

- Started with `native/vendor` genuinely absent: moved it to the ignored backup
  `native/build-cache/vendor-pre-refresh-c4eb71b48cf04c319fd5e1441be70df5`.
- Ran
  `powershell.exe -NoProfile -ExecutionPolicy Bypass -File F:\AIproject\LevNote\native\dependency-patches\refresh.ps1`.
- Exit code 0; the script recreated 814 vendor files, applied the
  `LEVNOTE PATCH` to vendored MyGo, and `go test ./dependency-patches` passed.
  `go.ps1`'s trailing `exit` therefore does not terminate the calling refresh
  script before patching. The ignored backup remains for audit; no workaround
  was attempted after recursive cleanup was denied.

### Encoded session integration

The landed and integrated builder path is:

- `CaptureOptions{Title, Scroll, OriginalPath, PluginID, View}`;
- `AddDocumentWithOptions(doc, options)` for a normal tab, preflighting all raw
  and encoded draft/metadata budget before materialization;
- `AddTab` for a clean saved/reopen record or explicit close-discard with
  `Draft=nil`;
- `Len`, transactional `SetPreferences` / `SetActive`, independent
  `Snapshot`, and preserving `Save`.

- App persistence now removes its raw aggregate preflight and per-tab
  `CaptureBounded`, calls `SetPreferences` first, uses
  `AddDocumentWithOptions` for normal tabs, uses `AddTab(Draft=nil)` for
  explicitly discarded disk-backed tabs, maps the active tab through
  `builder.Len()`, then calls transactional `SetActive` and `Save`.
- `Tab.OriginalPath` is preserved from `Restore` into every subsequent
  persistence; recovered drafts remain unsaved copies and never reacquire the
  original disk path.
- A failed explicit `Persist` no longer remains scheduled for another
  automatic attempt of the same unchanged state.

Focused session checkpoint passed after the concurrent edit settled:

`native\scripts\go.ps1 test .\core\session -count=1`

Result: `ok levnote.local/native/core/session 0.717s`.

### Immutable document disk-check request for GLM

The app must hash every open disk-backed text tab in one worker, including
read-only text tabs, without calling methods on a mutable `*Document` from
that worker. `document.CheckExternalChange` cannot be used there because its
method reads the document baseline concurrently with Save/SaveAs/Revert.

Please publish an opaque, value-copyable baseline plus a package-level check,
equivalent to:

- `func (d *Document) DiskToken() DiskToken`;
- `func CheckDisk(path string, token DiskToken) (changed bool, next DiskToken, err error)`;
- `func (d *Document) Revision() uint64`, incremented whenever `Path`,
  `IsNew`, saved encoding/BOM or the saved disk baseline changes.

`CheckDisk` must be callable without a `*Document`, use the same canonical path
and stable-read/hash semantics as Save, return a clear contradiction error for
an unstable file, and represent missing/unreadable files distinguishably.
`DiskToken` must not contain mutable slices/maps. The app will capture
`{tab ID, canonical path, Revision, DiskToken}` on the UI thread, hash only
immutable values in one coalesced worker, and apply a change/error only if all
captured values still match. Existing full Save conflict checking remains
authoritative.

## Checkpoints

### Recovery retry policy

Automatic recovery now makes one attempt for one unchanged state. A failed
idle attempt clears `persistPending` but leaves the last error and previous
recovery file intact. A new document/session/preference change marks it pending
again. Explicit `Persist` performs its one immediate retry and clears pending
when it fails; lifecycle close retries explicitly and keeps all tabs if durable
state cannot be established.

I added a close-workflow regression using one 3 MiB NUL draft. Its raw size is
legal, but JSON escaping exceeds the 16 MiB recovery limit before
materialization; close fails, both tabs and their order remain, and the prior
recovery bytes remain unchanged. Repeated-restore regression also checks that
OriginalPath survives two persist/restore cycles while the recovered document
stays new and the original disk bytes stay untouched.

Focused commands passed:

`native\scripts\go.ps1 test .\app -run '^TestRecoveredCopyPreservesOriginalPathAcrossRepeatedRestores$' -count=1 -timeout=20s`

`native\scripts\go.ps1 test .\app -run TestAutomaticRecoveryFailureIsNotRetriedUntilStateChanges -count=1`

### Bounded native lists and tree worker preparation

- Tree and workspace-search rows now use MyGo `ui.List`/`ListState` with stable
  path/composite result keys and observer hooks for visible-row assertions.
- Tree scanning is off the UI thread in a generic one-active/one-newest-pending
  coalesced worker. Requests capture root, copied expansion state and a
  generation; apply is rejected when root/generation changed. Old displayed
  rows stay until a complete bounded replacement is ready.
- `updateWait` now has a bounded dispatch acknowledgement timeout and prevents
  a late callback from applying after timeout. Queued callbacks recheck
  shutdown when they execute, shutdown releases an outstanding wait
  immediately, and timeout/report paths marshal App mutation back through the
  acknowledged dispatcher.
- Tree and search lists now wire `Selected` and element `Submitted`, so
  Up/Down/Home/End plus Enter operate through MyGo's real list keyboard path.
  A root replacement resets selection; stable path/composite keys retain tree
  identity across expansion and remove a vanished search row's selection.
- Virtualization tests previously passed for 3000 tree and 2000 search rows
  (both Windows app suite and focused list tests), including expansion while
  scrolled, far result selection, replacement, empty state and stable keys.
  The keyboard acceptance versions and exact visible-row/allocation logs are
  ready but cannot execute while current peer Markdown source is transiently
  uncompilable.

Focused shutdown regressions pass:

`native\scripts\go.ps1 test .\app -run 'Test(QueuedUpdateRechecksShutdownAtExecution|UpdateWaitIsReleasedAndLateCallbackSuppressedByShutdown)$' -count=1 -timeout=20s`

Current transient blocker at latest exact command: GLM's in-progress
`formats/markdown/inline.go:249` reports `p.limit.warned undefined`. This is
peer work in progress, not an app source failure; no full suite will be run
until their checkpoint lands.

Checkpoint results, exact commands, workloads and latency/allocation evidence
will be appended here as each increment completes.

### Virtualization and dispatcher checkpoint — 2026-11-08

The keyboard versions now pass against the actual 3,000-entry tree and
2,000-result search models. The allocation measurements retain the full model
during measurement, reset the row-built observer for every measured frame, and
compare against a five-row list. Values are from five `AllocsPerRun` frames and
ten fixed memory-counter frames on one Windows/amd64 run; they are tester
allocation costs, not process memory.

```text
tree virtualization: rows=3000 visible=3 maxRowsBuilt/frame=5 allocs/frame=50.0 bytes/frame=1831
tree small-list comparison: rows=5 visible=3 maxRowsBuilt/frame=5 allocs/frame=43.0 bytes/frame=1759
search virtualization: rows=2000 visible=7 maxRowsBuilt/frame=9 allocs/frame=99.0 bytes/frame=3263
search small-list comparison: rows=5 visible=5 maxRowsBuilt/frame=5 allocs/frame=66.0 bytes/frame=2453
```

Passing command:

```powershell
native\scripts\go.ps1 -GoArguments @('test','./app','-run','Test(WorkspaceTreeVirtualizesRowsAndKeepsExpansionAnchor|WorkspaceSearchVirtualizesFarResultAndReplacement)$','-count=1','-timeout','60s','-v')
```

The tests cover stable tree selection identity across expansion, root reset,
far-row `Home`/`End` plus `Enter`, search replacement, removed-search
selection, empty state, and visible row construction bounded well below the
full row counts.

`updateWait` now uses a started/completed handshake. Timeout and shutdown may
abandon only a callback which has not started. Once a callback starts, the
worker waits for its bounded completion and reports the mutation as applied;
the callback cannot begin after abandonment and is not queued or applied a
second time. Focused regression:

```powershell
native\scripts\go.ps1 test .\app -run 'Test(UpdateWaitTimeoutDoesNotAbandonStartedCallback)$' -count=1 -timeout=30s
```

### Disk-token worker integration — final API wiring

The app now captures only `{tab ID, token path, document.DiskToken}` on the UI
thread and submits all saved text tabs to one coalesced worker. Readonly text
tabs are included; `ByteInspector` tabs are excluded. The worker calls only
`DiskToken.CheckDisk(ctx)` and never receives a `*Document`. Results are
applied only when shutdown did not begin, the batch generation is current,
the tab ID still exists, its path still matches, and the UI thread confirms
`Document.SameBaseline(token)`. Missing/invalid/error states remain visible.
App add/remove/save/reload/encoding-reopen routes invalidate stale generations;
the document revision comparison also rejects an unchanged Save or Revert
whose baseline revision changes.

GLM published this exact API as final together with focused tests/vet:

```powershell
native\scripts\go.ps1 test .\app -run 'Test(DiskCheck|DiskWorker)' -count=1 -timeout=60s
```

Additional app regressions prove an unchanged `Save` and a same-byte `Revert`
invalidate the old token/generation, a same-size external replacement is
detected for a readonly text tab, ByteInspector tabs are excluded, stale
dispatches fail closed, and shutdown cancels an active hash worker.

The consolidated app and owned peer-package checkpoint passed:

```powershell
native\scripts\go.ps1 test .\app -count=1 -timeout=120s
native\scripts\go.ps1 test .\core\plugins .\formats\code .\core\document .\core\session -count=1 -timeout=120s
native\scripts\go.ps1 vet .\app .\core\plugins .\formats\code .\core\document .\core\session .\cmd\levnote
```

Results: app `2.735s`; plugins `1.965s`; code `0.906s`; document `0.676s`;
session `1.361s`; vet produced no diagnostics.

### Real draw.io lifecycle regression

Added app coverage equivalent to the real Markdown and mindmap routes:
double-click/select the actual draw.io node field, start a native `Compose`
without committing it to the host, then exercise ordinary `Save`, preview to
source mode, and tab switching. The save route asserts persisted bytes contain
the composed `中文🙂` value and an unrelated `vendor="keep"` XML attribute;
ordinary Save remains in preview mode.

An earlier run stopped during compilation in transient peer-owned search work:

```text
core\search\workspace.go:122:46: undefined: doc
```

Command:

```powershell
native\scripts\go.ps1 test .\app -run 'TestRealDrawioPendingIMECompositionAcrossSaveModeAndTab' -count=1 -timeout=60s
```

After that peer edit settled, all three real adapter routes passed:

```powershell
native\scripts\go.ps1 -GoArguments @('test','./app','-run','TestReal(Drawio|Markdown|Graph)PendingIMECompositionAcrossSaveModeAndTab','-count=1','-timeout','90s','-v')
```

Result: Markdown, mindmap, and draw.io each passed `save`, `mode`, and `tab`
subtests; total package time `1.332s`.

### Runtime fixture readiness and current acceptance gate — 2026-11-08 12:26

The app acceptance tests now share one fixture builder between visual captures
and the later native-process sample. The opt-in generator writes a
deterministic seven-file workspace plus plugin config and preserves unrelated
entries already present under the requested root:

```powershell
$env:LEVNOTE_ACCEPTANCE_RUNTIME_DIR='F:\AIproject\LevNote\native\artifacts\acceptance\runtime-fixture'
native\scripts\go.ps1 test .\app -run '^TestAcceptanceRuntimeFixture$' -count=1
```

Generated workspace:
`F:\AIproject\LevNote\native\artifacts\acceptance\runtime-fixture\workspace`.
The four representative open documents are `source.go`, `notes.md`,
`ideas.mindmap`, and `flow.drawio`. `settings.toml`, `readonly.txt`, and
`legacy.txt` support the plugin/readonly/encoding routes.

The current screenshots were regenerated after the final far-row fixture
selection was corrected:

```powershell
$env:LEVNOTE_ACCEPTANCE_RUNTIME_DIR='F:\AIproject\LevNote\native\artifacts\acceptance\runtime-fixture'
$env:LEVNOTE_ACCEPTANCE_SCREENSHOT_DIR='F:\AIproject\LevNote\native\artifacts\acceptance'
native\scripts\go.ps1 test .\app -run '^TestAcceptance(RuntimeFixture|Screenshots)$' -count=1 -timeout=120s
```

Result: `ok levnote.local/native/app 1.969s`. The 1180×780 and 760×480
large-tree/search images were inspected and now visibly select the final tree
row and a near-far search row. The tree warning reports both the actual
256-per-directory and 3000-total limits; no clipping was found in the refreshed
images.

Current-source bounded regression:

```powershell
native\scripts\go.ps1 test .\app -run '^(TestEmptyDirectoryAndPartialReadErrors|TestBoundedDirectoryReadDisclosesOverflow|TestWorkspaceTreeVirtualizesRowsAndKeepsExpansionAnchor|TestWorkspaceSearchVirtualizesFarResultAndReplacement|TestUpdateWaitIsReleasedAndLateCallbackSuppressedByShutdown|TestUpdateWaitTimeoutDoesNotAbandonStartedCallback|TestDiskWorkerShutdownTerminatesActiveCheck)$' -count=1 -timeout=90s
```

Result: `ok levnote.local/native/app 2.065s`.

Root rejected the 12:25 `markdown-conflict-760x480.png` because content and
status text visibly collide through y450-480. That PNG remains failed evidence.
The app host now clips the main column vertically and, below 820 DIPs with a
long status message, keeps only the single-line message plus the Issues button
while hiding lower-priority caret/encoding/state fields. A real Markdown
conflict regression checks that draft controls end above the status bar and
that the message does not overlap the Issues entry:
`TestNarrowMarkdownConflictKeepsViewOutOfStatusRow`.

Astra reports its owned view fix ready with bounded outer/list clipping,
zero minimum height, an 80px draft, a two-line conflict summary, compact
buttons/footer, and a 265×300 bounded-content assertion. The first app
recapture attempt after that report was blocked before tests ran by transient
GLM search compile errors, so the accepted PNG has not yet been replaced.
Recapture is pending GLM's compilable search checkpoint.

Runtime process resource sampling is intentionally not started yet. Astra's
final long-field shaping/view checkpoint and GLM's actual bounded workspace
search reader are still pending, and the current executable predates the
resource work. Once both are green, the next accepted binary will be sampled
as idle plus the four-document fixture above, with working set/RSS and private
bytes reported separately from the tester allocation numbers in this file.
