# Closed-file implementation checkpoint

2026-11-10, before first focused run. Source now on disk; no full app suite,
vet, package, host, worker or resource execution. .26 evidence is untouched.

- Own changes: app/app.go, reopen_closed.go, binary.go, binary_actions.go,
  view.go, workspace.go; focused tests in app/reopen_closed_test.go.
- History retains16 cloned canonical local paths plus monotonically increasing
  IDs. Reclose gets a new ID; remove/eviction/shutdown zero backing entries.
  CloseTab verifies its original target/slot after dialog/SaveAs and records
  only after recovery persistence and actual removal. CanClose never records.
- Private receipt holds only entryID and atomic outcome/queue-finished fields.
  One reservation lasts until queue completion. Only actual UI Select/append
  consumes the exact ID; timeout/abandonment/stale ID/refusal cannot consume a
  newer close. Public Open/OpenMany/OpenAt signatures and binary lane preserved.
- Check shared guard at command and adoption, after Flush where appropriate.
  A synchronous recovery that changes the current tab refuses adoption while
  keeping the recovered draft/current presentation. Utility preedit disables
  the menu before focus transfer; source/rich input uses its established Flush.
- Native compact popover beside Open and Ctrl+Shift+T share the guarded path.
  Menu IDs are stable entry IDs; labels/tooltips bounded without truncating
  stored paths. Wording states current disk contents, no discarded edits.
  Failed requests retain records and offer explicit retry/older menu selection.
- Focused tests exercise real text/PNG/byte opens, unknown encoding, SaveAs,
  discard/no-path and close failures; duplicate Select/Flush, pending newer
  close/reclose/eviction, final modal/preedit guards, timeout/shutdown/late
  callbacks, synchronous recovery, bounded retention and real Tester key/menu
  focus/geometry. First focused run and optional single760x480capture next.

Known validation questions are test assertions/geometry, not unresolved scope.
The later live-tab admission cap is explicitly excluded. Final full affected
app test/vet belongs to the later combined root activation after both freezes.
