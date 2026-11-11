# Independent review: preview consumer lifecycle

2026-11-09. Read-only review of the finite shared-lane/app lifecycle slice. I
did not edit product files, run tests, build, launch a process, or delegate.
The files were read on 2026-11-09 around 07:57–08:02 local time; source was
actively changing, so this is a snapshot review, not final integration or
visual/process approval.

## Reviewed snapshot

- `native/app/engine_lane.go` — SHA-256
  `734F3681DE1FD8283125EC7115013C651837FFEE4656E6D5E7B4C2100D1E5EF3`.
- `native/app/preview_worker.go` — SHA-256
  `668F95957A62D0243F5BFC15FDBC2CF50B9DC9FB64DC347284388A03638CC3AF`.
- `native/app/export_worker.go` — initial SHA-256
  `5FA1A6B89F92D162FA3575CC7527E37FC0555B314F36E635106F6A0115DD438C`; a later
  read observed SHA-256
  `8014472548B171740C2291DAE69932063122006F6E8C176FFCEDF6B42A18D20F`.
- Other scoped reads included `reference.go`, `app.go`, `view.go`,
  `workspace.go`, `ui/editor/contract.go`, and `ui/editor/scene.go`.
- `native/vendor/github.com/egoist/mygo/content.go:159-195` and
  `native/vendor/github.com/egoist/mygo/loop.go:13-37` were read for queue
  behavior.

## Withdrawn finding

I withdraw the earlier close/completion concern. The reviewed interleaving
itself shows the active task is completed exactly once: `engineLane.close()`
cancels it and removes `pending == active`; the worker loop then performs the
one normal completion and exits. Direct close completion applies only to a
non-active pending task or export reservation, so those paths do not overlap.

The production completion callbacks are not arbitrary view callbacks:
`previewJob.finish` is UI-free, bounded by the fixed payload, checks its cancel
slot around decode, and closes `j.done` exactly once; `exportJob.complete`
likewise fills and closes its result slot once. `Window.Invalidate` does not
wait for UI: it uses a CAS to coalesce calls, and `postMain` only locks,
appends, and signals the main loop, returning false after shutdown. Therefore
`engineLane.close()` waiting for the already-bounded result-slot handoff is the
intended join, not a UI dependency or double completion.

## Limited no-finding

Within the reviewed snapshot, I found no concrete defect in:

- shared runner exclusion (`engineLane.run` serializes `Runner.Export`);
- preview preemption/reservation ordering (`reserveExport` promotes the active
  preview back to pending and wakes the lane);
- one-latest pending preview replacement (`submitPreview`);
- preview snapshot/identity capture and delivery recheck
  (`beginScene`, `validPreview`, `deliverScene`);
- export reservation held through publication
  (`deliverEngineExport` releases only in its deferred cleanup);
- cleanup-error poisoning (`cleanupErr` blocks new tasks and `close` preserves
  it);
- payload lifetime (invalidated preview jobs do not retain decoded payloads,
  and delivered results are cleared before callback);
- modal unwind draining both export and scene delivery
  (`deferExportDelivery`);
- `ShowSource` owner/current-tab recheck (`reference.go:172-174`).

This limited reviewed slice has no additional actionable finding. It does not
approve final integration, visual behavior, process packaging, or unobserved
renderer files.
