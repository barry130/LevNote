# Next resource workload plan

2026-11-09. Planning only. This file proposes a small measurement stage after `.8`; it
does not claim a run, comparison, baseline, threshold, or acceptance. Evidence read:
`.8` resource notes in `README.md` and `VALIDATION.md`, `core/text/buffer.go`,
`core/text/range_map.go`, and the tab/presentation/engine lifecycle in `app/app.go`,
`app/view.go`, `app/plugins.go`, `app/outline.go`, and `app/engine_lane.go`.

## 1. User workloads

Generate fixtures locally with a deterministic, non-private fixture builder: fixed
seed, UTF-8 text, no user files. Keep the generator and fixture hashes as evidence,
but run operations through the real UI. All three cases should be separate sessions.

| Case | Fixture and operations | Operational settle points | Correctness, separate from timing |
| --- | --- | --- | --- |
| A. Mixed tabs | 5 files in one workspace: 15 KB Markdown, 10 KB `.mindmap`, 4 KB plain text, 6 KB Go source, 12 KB `.drawio`. Open all; visit preview/split on Markdown, mindmap, and drawio; return each to source; edit two Markdown headings; start HTML export; switch tabs during export; close all. | Each open: tab count increases and editor can focus. Each view switch: status text stabilizes and the former view is released. Export: status becomes completed or cancelled and its one-shot child exits. Close: session write finishes; the persistent host engine-lane goroutine may remain until app shutdown. | Tab count/order, mode, Markdown buffer content/hash, dirty state, and saved UTF-8 bytes must match the declared edit. A cancelled export must leave its target absent or byte-identical to its pre-run state. Supported-limit warnings are accepted and recorded; fatal or unexplained errors are not. |
| B. Large edit/undo | One about 500 KB, 8,000-line Markdown file with headings, lists, tables, and fences. Open; select the end; insert one 512-byte block as transaction 1; save; then make 100 independent single-character insertions at the current end, each as its own transaction; undo 100 times; redo 50 times; save; reload. | Open: buffer byte count/hash matches fixture. Each transaction: version advances once and the view reaches paint-idle. Undo/redo settles after the expected final text is visible; save/reload settles after disk read and status clear. | After 100 undos the buffer equals the first saved fixture-plus-512-byte block. After 50 redos it equals that block plus 50 known characters. Selection follows the declared transaction positions, not a generic “command history” claim. Saved bytes and reload hash equal the final visible text; `Truncated` is recorded if the defined 32 MiB/256-entry budget is reached. |
| C. Repeated lifecycle | Three 30 KB Markdown files. Open all, alternate source/preview across tabs 20 times, close each after a short export, reopen all, close all. Include one observed 20-second idle tail before the final close. | Switch: flush and release complete before the new tab is ready. Export: status settles and its one-shot child has exited before close. Close-tab: tab removal and session persistence finish while the host engine-lane goroutine remains alive by design. Final app close: engine-lane close and host exit finish. | Every switch preserves the buffer hash and selection. Reopened files equal the last saved bytes. A settled export/cancellation requires its child joined; persistent lane is not a persistent worker process. Idle drift is observational and not called a leak. |

Record timestamps at operation start, settled UI, and close. Keep correctness logs
independent of timing samples. Use no performance pass threshold in the first stage.

## 2. Collectible Windows process metrics

Use a run-owned process handle obtained from the launched application, never a
name-based sweep. Record executable path, PID, creation time, parent PID, and job or
fixture directory. Treat each simulation worker as owned only when it was launched by
that exact run and its manifest/executable identity matches. Close copied handles
after exit and preserve unrelated sentinel processes.

Collect these fields:

- `PROCESS_MEMORY_COUNTERS_EX.PrivateUsage`: private bytes at each settle point and
  during sampling. Report as MiB. Do not call it RSS, nonpageable RSS, committed
  memory, pagefile usage, or “working set” without the working-set qualifier.
  This is an OS field value; keep the name and units aligned with worker evidence.
- `WorkingSetSize` and `PeakWorkingSetSize`: sampled and peak working set, separately.
  Do not add bytes-per-operation, file size, private bytes, or commit memory to it.
- `PagefileUsage` and `PeakPagefileUsage`: current and peak commit, reported separately
  from working set. The existing worker evidence uses these field names and meanings.
- `GetProcessTimes`: cumulative user/kernel CPU time. Report CPU seconds on one core
  and as a share of this host's logical processors, verified at run time. Preserve
  zero deltas as “below observed timer granularity,” not “no CPU use.”
- Worker counts and lifecycle: requests submitted, processes launched/exited, and
  each worker's peak working set/commit and cumulative CPU time. Report main process
  and workers separately; do not sum unrelated processes or simultaneous peaks as
  though they occurred at one instant.

Sample the main process at operation starts and settles, plus a short high-cadence
window during export and repeated switching. Query final peaks after exit if the
handle remains valid. Calendar wall time and internal stopwatches are diagnostic,
not memory substitutes.

## 3. Paired-run method and remaining obstacle

Prepare two immutable manifests: the `.8` package and the next root-accepted package
when it exists. Each manifest identifies every executable, manifest, fixture, config,
and session directory by SHA-256. Use the same generated fixtures, same workspace
paths, same config, same display scaling, same step script, and fresh session data.

Before measuring, pass an implementation gate proving that the proposed physical
real-UI operations can be driven reproducibly without assuming internal state is
readable through an OS-level UI automation surface. If that cannot be demonstrated,
the stage stops or narrows to operations that can be observed honestly. Do not
substitute hidden startup or internal test hooks for the user workload.

First run one pilot pair total (`.8`, then candidate), not one pilot per workload. If
process ownership, fixture hashes, or step completion fail, stop and correct the
harness before collecting more numbers. Then, for one priority workload only
(recommended C, repeated lifecycle), run one alternating AB/BA pair set: `.8`,
candidate, candidate, `.8`. This is four measured runs plus the two pilot runs,
six total for this first stage, not an 18-run matrix. Add another workload only after those counts and
results are reviewed. Preserve raw samples, logs, operation boundaries, process
identities, exits, and hashes. Compare each version to itself and report paired
deltas; do not compare B/op, file size, or startup idle to editing working set. A
visible-editing session is the unit of interest; hidden idle startup remains only a
control.

The exact obstacle for a legacy Tauri comparison is that the repository records no
matched Tauri/native executable, shared workload recipe, or proven equivalent
format/config routing for these scenes. Existing `.1/.2` worker samples and `.8`
hidden startup are not a same-workload UI baseline. Therefore legacy comparison stays
blocked until a preserved legacy executable and a compatibility gate prove identical
fixture, routing, save/export semantics, and process ownership. The `.8` versus future
native candidate comparison is feasible sooner; before that candidate is accepted,
state that the second binary does not exist and collect no “new package” values.

## 4. Likely hotspots, in priority order

1. **Per-tab text and history retention — high priority hypothesis.** Each tab keeps
   its `Document`, `Buffer`, editor, and up to 256 history transactions or 32 MiB of
   retained edit bytes per buffer (`buffer.go`). Undo/redo transactions store old and
   new strings even though chunks are shared. Several large dirty tabs can therefore
   retain more than visible file size and several times the per-tab budget in total.
   Measure private bytes by tab count and history depth before changing limits.
2. **Outline/view reconstruction on tab switches — medium priority hypothesis.**
   `Select` calls `releasePresentation`, which calls `releaseOutline`; a later outline
   render creates a factory source again for the selected tab. Repeated switching may
   reparse the same unchanged document and churn allocations. `outlineSourceFor`
   already reuses a source only for matching tab/buffer/generation. Measure parse
   work and switch-settled allocations before adding a cache; a cache also has a
   retention cost.
3. **Engine-task snapshots surviving cancellation — medium-low priority hypothesis.**
   `engineTask.source` is an immutable snapshot cleared only by `finishEngineTask`.
   A cancelled active preview/export can remain until the runner returns, while a
   preempted task may be requeued. Measure overlap during repeated export/switch and
   close. Change cancellation cleanup only if traces show the source remains across
   several settled operations.

These are unmeasured hypotheses. Process data, event boundaries, and repeated runs
must decide whether code changes are justified; no hotspot is a confirmed defect.
