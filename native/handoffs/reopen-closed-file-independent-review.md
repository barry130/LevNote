# Closed-file workflow independent source review

2026-11-10 read-only review for
`reopen-closed-file-independent-review-task.md`. Astra remains the
implementation and focused-test owner. This review made no source, test,
build, process, package, or product-document changes and ran no tests.

## Result

No source-grounded lifecycle, adoption, or ownership defect was found in the
reviewed revision. There is therefore no outstanding owner repair request.

The latest reviewed source also includes the blocked-menu `KeepFocus` correction
needed to avoid transferring focus away from an active utility IME composition
when the disabled menu button is pressed.

## Reviewed source identities

Any later edit is outside this review boundary.

| File | Bytes | SHA256 |
|---|---:|---|
| `native/app/reopen_closed.go` | 7895 | `BFCA75BA0E2A878483133ECE43530742C424623CBE6B8809F1285C9E04C62E92` |
| `native/app/app.go` | 38029 | `C1061C650ED8A5FF3231F3642B003225D2155508F1C8B435B40981CE5052F1D3` |
| `native/app/binary.go` | 10170 | `AE2F059F0E91C812A82FFE3F0317EAFE7E2F0B0CE9AEB3AA5056F713C555226D` |
| `native/app/binary_actions.go` | 9626 | `84129D789A02A16B57B467A52E5C6B0E99278FE7A4F5F8DBBC02ED06EB701E11` |
| `native/app/view.go` | 23002 | `195AFA3D82E7612A0CDBEC6A9D0F0432777F492CAB4D94596B0C6F2329C83456` |
| `native/app/workspace.go` | 6715 | `B1492575D6C64673BF02E633C0D7049F47BF151391B4CA58F58F3779599AF259` |

Relevant existing tests inspected without execution:

| File | SHA256 |
|---|---|
| `native/app/reopen_closed_test.go` | `480E666DF7E2D82268AA79C5702C164EA9BDA30B2062513C2AB7A68D3DCBD012` |
| `native/app/app_test.go` | `7E685C4E3544EA850E9524A7AEB42E6CF022294A104AB38FBF443F424D2FD2B1` |
| `native/app/binary_test.go` | `1D150577CF49C01F9C14D18A18611171243492EECB412B60AB544DBBE78D4981` |

## Lifecycle checks

- `CloseTab` computes the candidate path only after `prepareClose` returns, so a
  SaveAs path is current. It revalidates the exact original tab at the removal
  slot after nested dialog work, tests the exact removal receipt, and calls
  `recordClosedFile` only after both persistence and removal succeed.
- Whole-window close removes tabs directly and never records a normal closed-file
  entry. Shutdown cancels an in-flight reopen, drops its small receipt, clears
  the full history backing array, and then tears down binary work.
- A history entry is only `{id,path}`. IDs are monotonic, re-closes receive a new
  ID, paths are canonicalized and cloned, same-path entries deduplicate
  case-insensitively, and removed/evicted backing slots are explicitly cleared.
- The history is capped at 16 entries. Stored path admission is capped at 128 KiB
  UTF-8, while menu labels/tooltips are separately bounded without changing the
  path used to open. Relative, UNC, overlong, invalid, and otherwise unsupported
  paths are omitted with a visible notice only when relevant.

## Adoption and request identity

- Queue completion is kept distinct from UI adoption by an atomic terminal
  outcome and a finished flag. A worker finishing does not consume history.
- Every late text, duplicate-select, and binary adoption path rechecks the exact
  receipt pointer, terminal state, and exact entry ID. A newer close, same-path
  reclose, eviction, cancellation, shutdown, or stale callback cannot consume a
  replacement entry or select the current top by accident.
- A new text tab is acknowledged only after the checked add path succeeds.
  Selecting an already-open text tab is acknowledged only after the exact tab
  still occupies the captured slot and the checked selection succeeds. Binary
  duplicate and new-tab paths have equivalent receipts after actual selection
  or attachment to the new tab.
- Failed lookup, Flush refusal, refused selection, dispatch timeout, binary
  cancellation, and modal/utility-composition barriers leave the entry in
  history for explicit retry. The binary job still releases its task, tab/view
  references, payload slot, and lane reservation on completion or abandonment.
- A valid current-disk byte fallback remains a successful open. Unknown legacy
  text returns through the normal explicit encoding-inspector route rather than
  restoring an old code page.

## Composition, focus, and normal callers

- Shortcut and menu triggers share `reopenClosedFile`; only one request can be
  active. The repeated-key guard prevents held-key duplication.
- Modal, restore, shutdown, and visible Find/GoToLine composition states are
  checked at command entry and again at final adoption. The disabled toolbar
  button keeps utility-field focus when blocked.
- Normal list ordering remains path-only: the newest missing item is retained
  rather than silently skipped, and an older menu item can be chosen by stable
  ID.
- The receipt is optional on the private open request. Normal `Open`,
  `OpenMany`, and `OpenAt` signatures and batch order remain unchanged; duplicate
  text reuse and `OpenAt` reveal still pass through their existing lifecycle.

## Evidence limits

This is a static source review. It does not establish a passing test run,
compiler result, real Windows menu behavior, physical IME behavior, or package
behavior. Astra's focused tests must still establish actual execution. The
existing tests visibly cover successful text/PNG/byte adoption, SaveAs/discard,
failed close and persistence, exact stale IDs, reclose/eviction, Flush recovery,
modal/timeout/shutdown/late callbacks, binary duplicate refusal, bounds/backing
cleanup, blocked utility composition, and the 760x480 menu geometry, but this
reviewer did not run them.
