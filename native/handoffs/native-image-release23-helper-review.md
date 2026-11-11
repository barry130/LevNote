# .23 mixed-session helper independent review

2026-11-10, read-only review by GLM. This is a helper checkpoint for the
preparation owner, not execution, product acceptance, or source acceptance.

## WIP cutoff

The findings below were taken from this working-tree state. No helper, product
source, test, build, host, worker, parser, or resource process was executed.

```text
native/AGENTS.md                                              14728 B  2026-11-10 12:53:21
native/handoffs/native-image-release23-helper-review-task.md  1598 B  2026-11-10 12:45:55
native/handoffs/native-image-release23-preparation-task.md     5068 B  2026-11-10 12:11:41
native/handoffs/native-image-release23-preparation-review-notes.md 3105 B 2026-11-10 12:43:58

aggregate.ps1          26683 B  2026-11-10 12:34:50
baseline.ps1           10581 B  2026-11-10 12:31:20
delivery-lib.ps1       18908 B  2026-11-10 12:19:52
initialize-context.ps1 14319 B  2026-11-10 12:31:20
invoke-stage.ps1        2210 B  2026-11-10 12:31:20
run-host-smoke.ps1     24441 B  2026-11-10 12:32:07
run-package.ps1        16751 B  2026-11-10 12:22:28
run-stage.ps1           2509 B  2026-11-10 12:31:20
source-scope.ps1        2157 B  2026-11-10 12:19:52
static-parse.ps1        1666 B  2026-11-10 09:11:24

native/portable/README.md 13903 B  2026-11-10 12:33:34
native/cmd/levnote/main.go   882 B  2026-11-08 10:32:07
native/app/state.go        13553 B  2026-11-10 12:43:42
native/app/app.go          36131 B  2026-11-10 12:46:32
```

## Resolved from the current source

- The seven required source gates now include
  `binary-copy-receipt-root-acceptance.md`
  (`initialize-context.ps1:69-95`), resolving root note 1.
- Candidate counts are not blindly inherited from `.22`; package and aggregate
  schemas include the actual candidate count, expected count, expected set,
  engine checks, PE facts, source copies, and README identities.
- The host readiness path is a bounded semantic poll of the actual mixed
  session, with process-exit checks, rather than a fixed delay.
- Both runs share one byte-identical copied bundle and one isolated config
  root; each run issues exactly one owned `WM_CLOSE` only after its readiness
  gate, and cleanup records failures rather than hiding them.
- The aggregate captures logs outside the evidence inventory, requires the
  actual package/host result fields, and explicitly scopes the evidence away
  from decode/rendering, physical input/IME, and RSS/VRAM claims.

## Finding 1: run 2 can pass by rereading run 1’s session

The current pre-close check cannot distinguish an actual run-2 restoration
from stale run-1 JSON:

- `run-host-smoke.ps1:335-342` calls `Get-ReleaseSessionFacts` and marks the
  run ready as soon as the expected mixed shape is readable.
- For run 2, run 1 has already written that exact shape. There is no per-run
  pre-launch session baseline or post-close publication proof in
  `run-host-smoke.ps1:396-458` or `aggregate.ps1:341-387`.
- The product behavior explains why this is more than a theoretical naming
  problem. `native/cmd/levnote/main.go:20-35` creates the window, calls
  `Restore()` synchronously, then registers close handlers and opens CLI paths.
  `native/app/state.go:17-122` loads the old session into memory but does not
  publish a new `session.json` during restore. Publication happens later:
  `native/app/app.go:1046-1064` persists still-open tabs during `CanClose`,
  and `native/app/state.go:208-215` saves the session. Therefore run 2 can
  observe valid old JSON immediately even if restoration failed internally.

Smallest truthful correction, without another host run or UI automation:

1. Before each launch, capture the exact config `session.json` fingerprint
   (including bytes, SHA-256, and `LastWriteUtcTicks` where available):
   run 1 should require absence in a fresh config; run 2 should preserve a copy
   of the run-1 baseline in the run-2 evidence directory.
2. Keep the current expected-shape pre-close observation, but record it as
  `SessionFactsBeforeClose`; do not name or treat it as proof that run 2
   restored those tabs.
3. After exit 0 from the owned `WM_CLOSE`, capture the final fingerprint and
   semantic session facts. Add explicit run fields such as:
   `SessionBeforeRun`, `SessionBeforeRunFingerprint`,
   `SessionAfterCloseFingerprint`, `FreshSessionPublication`, and
   `ClosePublicationAfterProcessStart`.
4. Require each run to have published a fresh session after its own process
   start. For run 2, additionally require the post-close identity to differ
   from its captured pre-run baseline, especially `LastWriteUtcTicks`.
5. Update `aggregate.ps1`’s run contract and `Test-HostRun` accordingly. Its
   run-2 claim should require: expected pre-close shape + fresh close-side
   publication after run-2 start + post-close identity differs from run-2’s
   pre-run baseline.

This uses the actual lifecycle and does not require startup publication,
another process, or any new UI automation.

## Finding 2: `DistinctRunProcesses` does not prove distinct lifetimes

Current calculation in `run-host-smoke.ps1:538-539`:

```powershell
DistinctRunProcesses = [int64]$run1.App.Id -ne [int64]$run2.App.Id -and
    [int64]$run1.App.HeldHandle -ne [int64]$run2.App.HeldHandle
```

Windows can reuse a PID after the first process exits. Handle values are also
scoped and reusable once the first held handle is released. The runs are
sequential, so unequal PID/handle values are descriptive identity evidence,
not lifetime proof.

Smallest correction:

- `run-host-smoke.ps1` already records `App.StartTimeUtc`
  (`run-host-smoke.ps1:282-287`), but `aggregate.ps1:63-65` omits it from the
  required process fields. Require it.
- Replace the boolean proof with sequential lifetime ordering:
  run 1 `StartTimeUtc` < run 2 `StartTimeUtc`, and run 1 completed before run 2
  started. The per-run `FinishedAtUtc` is already available.
- Keep PID and held-handle values as descriptive evidence. A clearer summary
  field name is `DistinctSequentialRunLifetimes`; retaining the old name with a
  corrected calculation also avoids unnecessary schema churn.

## Finding 3: aggregate-only retry is blocked by old outer capture files

`run-stage.ps1` always creates a sibling `...-aggregate-capture` directory and
requires the same three paths to be absent:

```powershell
outer-aggregate.stdout.log
outer-aggregate.stderr.log
outer-aggregate-invocation.json
```

If packaging and both host runs pass but aggregate then fails for a
metadata/schema error, those failure logs exist. Reusing the same evidence root
would require deleting preserved failure evidence; creating a new evidence root
would discard or duplicate the already completed product operations. This
repeats the exact failure mode `.21` corrected by allowing one aggregate retry
without repeating product work.

Smallest correction:

- Add an explicit aggregate retry attempt, or auto-number the aggregate capture
  files/directories. Require each new attempt’s paths to be absent while
  preserving prior attempt logs unchanged.
- Keep `aggregate.ps1:8-10` guarding against an already completed
  `aggregate.json`; retry should be possible only while the aggregate has not
  succeeded.
- Keep package and host-smoke evidence immutable so a metadata-only correction
  cannot repeat packaging or hidden host runs.

## Finding 4: helper still requires forbidden README implementation labels

Root note 5 asks for plain Chinese user instructions and specifically removes
`path-only`, `single-lane`, and `单车道行为` as literal user-facing labels.
The current README still contains:

- `path-only` (`native/portable/README.md:58`)
- `single-lane` (`native/portable/README.md:64`)
- `单车道行为` (`native/portable/README.md:66`)

More importantly, the helpers themselves still require the first two literals:

- `initialize-context.ps1:199-204`
- `baseline.ps1:137-143`
- `aggregate.ps1:229-232`

Thus a corrected README would fail context, baseline, and aggregate checks.

Smallest correction:

- Replace the literal assertions with plain meaning: session recovery reads the
  saved absolute path again; image reading, decoding, and copying run one at a
  time.
- Keep the truthful bounds in Chinese:
  baseline 8-bit JPEG supports grayscale or standard three-component color,
  not CMYK or progressive JPEG; PNG is at most 8-bit, noninterlaced, and
  nonanimated. Candidates over 16 MiB remain inspectable as bytes but cannot
  use the bounded image copy operation.
- Keep the wording that an accepted snapshot may also be retained during a
  native Save As dialog, so explicit open/reload is not the only possible
  transient source reference.
- Keep the statement that inactive payload release and one-at-a-time scheduling
  are not RSS/VRAM guarantees.
- Update all three helper checklists consistently, then perform the one final
  static parse.

## Future activation condition, not a current preparation blocker

The current seven-gate source set in `initialize-context.ps1:69-95` is coherent
for preparation. It does not include
`native-image-view-narrow-toolbar-root-acceptance.md`, which root note 6 says
must bind delivery after the narrowed toolbar is accepted.

Do not treat its current absence as a preparation defect. The important
execution condition is that a context initialized now freezes only those seven
gates. Once the narrow-toolbar acceptance exists, execution must be bound by
reinitializing the context or explicitly extending its frozen gate list before
the `.23` run. The currently prepared context must not be reused unchanged for
actual delivery after that new gate lands.

## Conclusion

The major package-count, source-scope, readiness, close, and no-resource-claim
corrections from the root notes are implemented. Four bounded helper defects
remain before the owner’s one final static parse:

1. Add per-run session baseline/close-publication proof so run 2 cannot pass
   from run 1’s stale JSON.
2. Prove sequential distinct lifetimes from timestamps and completion order,
   not unequal PID/handle values.
3. Permit a preserved, aggregate-only retry attempt without deleting old
   failure logs.
4. Replace the literal `path-only`/`single-lane` README assertions and wording
   with plain Chinese meaning.

No additional architecture, broad test gates, or extra host/UI evidence is
needed for these corrections.
