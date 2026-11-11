# Hidden host readiness verdict

2026-11-10, read-only trace. The mandatory pre-close mixed-session poll is not
a reliable readiness gate. It can succeed if opening happens before an available
frame, but after the first hidden frame there is no guaranteed frame to publish
automatic session state. Do not make .23 success depend on that timing.

## Concrete path

- MyGo `internal/windows/surface.go:275` invalidates, but posts drawHidden only
  while !drawn and hidden. FirstFrame sets drawn at343; drawHidden at350 refuses
  thereafter. The adjacent comment explicitly states hidden windows get no
  WM_PAINT. Ordinary changed/invalidation does not synthesize another draw.
- `content.go:190` Window.Update posts and executes its callback, then calls
  contentChanged. `ui/runtime.go:1194` changed and `ui/window.go:731` only request
  a frame. The backend's posted callbacks run through its message queue
  (`internal/windows/app.go:281`), independently of paint. Context.After also
  ends in invalidation, not synchronous View execution. Capture/Fit paths can
  build frames but this helper uses neither and should not add such automation.
- `app/binary_actions.go:39` waits for the lane then updateWait(deliverBinary),
  so CLI adoption and explicit-job release can complete without another frame.
  completeBinaryExplicit at166 only marks persistPending. `app/view.go:32`
  invokes persistWhenIdle; `app/state.go:229` is therefore frame-dependent.
  Text opening also marks pending rather than synchronously saving.
- `cmd/levnote/main.go:20` restores synchronously before registering close and
  before queuing CLI opens. `app/state.go:121` ends restore with pending=false.
  `app/app.go:1046` CanClose persists current tabs at1063 before removing them;
  the atomic session writer runs directly, without needing View/WM_PAINT.

All MyGo paths above are under native/vendor/github.com/egoist/mygo/.

Run1 has no session in its fresh config: it may remain absent or partial until
normal close despite both tabs already being adopted. Run2 already has run1's
valid JSON: a shape poll can pass immediately without proving restoration.
Neither longer polling nor that old file resolves readiness. Run2's synchronous
restore and a fresh close-side publication of the exact two records provide
the useful evidence instead.

The accepted .22 helper in artifacts/acceptance/tab-workflow-release22-20261110
waited1200ms after owned hidden-window discovery, posted one WM_CLOSE, then read
session-after-close.json. Its result records PID2480, hidden=true, one close,
exit0, no forced cleanup and job total1. It did not check pre-close session;
this historical pass does not support the new mandatory poll assumption.

## Smallest verification adjustment

Keep existing exact held-process/window/job checks. Replace mandatory session
readiness with a recorded bounded settling interval, e.g.5s after window found,
checking process liveness every100ms. Label it settling, not proof of open/decode
completion. A pre-close session observation is optional and descriptive only.
Then post the one owned WM_CLOSE and retain the existing15s normal-exit bound.
If binary work is still pending, CanClose deliberately refuses; preserve that
failure and clean only owned processes. Do not retry close/start or send paint,
input, show-window or capture commands to manufacture readiness.

Require post-close exact mixed order/active/encoding/BOM/binary shape, exit0,
both owned processes ended and unchanged fixtures/plugin/bundle. Run1 requires
an absent pre-run session. Run2 preserves its exact baseline copy and requires
a fresh timestamp after its own start and newer than that baseline; identical
JSON bytes are expected, so changed SHA is not required. This catches empty or
partial restored state rather than accepting the old run1 file. Retain the
separate per-run records and actual sequential lifetime evidence.

Adjacent concrete helper correction: current run-host-smoke.ps1:284 calls
DateTime.FromFileTimeUtc on LastWriteUtcTicks, while delivery-lib.ps1:49 stores
DateTime.Ticks (year1 epoch). These are different epochs; use a UTC DateTime
constructed from those ticks or copy the source's LastWriteTimeUtc. Otherwise
run2's preserved baseline-copy timestamp comparison fails before launch.

No product persistence fix is needed for this hidden startup/normal-close gate.
Autosave while a real window remains hidden/minimized would be a separately
scoped persistence requirement; this trace does not establish it. No PNG decode,
pixels, GPU, physical input, dirty recovery or resource behavior follows from
session/window presence. Only this handoff was written; no source/helper edits,
tests, builds, captures, host/worker runs or new automation were executed.
