# Closed-file workflow implementation — frozen

2026-11-10 09:25:55Z. Assigned implementation and focused validation complete.
Six runtime files plus one focused test file frozen. Root can review/accept this
slice, then activate the separately owned tab-admission increment and one combined
full app test/vet per reopen-and-tab-admission-stage-plan.md.

## Behavior and ownership

- Ctrl+Shift+T and the compact native popover beside Open use one guarded command.
  Up to16 cloned canonical local paths/IDs live only in memory. Reclose dedups
  case-insensitively with a new ID; removal/eviction/shutdown clear backing slots.
- Only successful individual CloseTab records. Capture follows SaveAs; exact
  tab/slot and shutdown state are rechecked after dialog reentry. Recovery must
  persist and that exact tab must actually be removed. Window quit never records.
- Saved discarded edits are not retained; unknown encoding uses the existing
  explicit readonly inspector. Reopen reads current disk bytes/current routing.
  Unsaved/recovered drafts have no history authority; OriginalPath is unused.
- Small optional receipts carry entryID + atomic outcome/finished fields, no
  closed resources. Text, image and duplicate selection consume the exact ID
  only after real UI adoption. One in-flight reservation, no automatic retry.
  Newer closes/reclose/eviction, timeout/cancel/shutdown cannot consume replacement
  records. Existing Open/OpenMany/OpenAt signatures/ordering and lane remain.
- Source/rich input uses Flush. Synchronous recovery that changes current tab
  refuses adoption and preserves the new draft. Utility preedit blocks both entry
  and final adoption; blocked menu uses public Disabled+KeepFocus before pointer
  focus transfer. Modal/restore/shutdown are also guarded.
- Menu exposes older files by stable ID, current-disk/no-discarded-edits wording,
  empty/busy state, bounded labels/path tooltips. Failures keep records and explain
  retry/older-item availability. Missing newest entries are not silently skipped.

## Preserved focused execution

Evidence: [reopen-closed-file-evidence-20261110](reopen-closed-file-evidence-20261110).
Every command is saved with stdout/stderr, UTC start/end and numeric exit. Inner
PowerShell scripts call scripts/go.ps1 and immediately `exit $LASTEXITCODE`.

| Evidence | Actual result |
|---|---|
| 00-format | exit0; initial named-file formatting |
| 01-focused | exit1, app2.619s; initial failure kept intact |
| 02-format | exit0; focused corrections |
| 03-focused | all then-current16 TestReopenClosed groups PASS, app3.144s, exit0 |
| 04-format | exit0; final shutdown guard + cancellation test |
| 05-correction-capture | changed close/shutdown cases, actual running image cancellation and menu capture PASS, app2.142s, exit0 |
| 06-menu-identity | stale native-menu ID and successful shortcut repeat/nonmatching modifiers PASS, app1.648s, exit0 |

01 exposed a real disabled-button pointer-focus leak, fixed with public KeepFocus
only when blocked. Fixture corrections preserved existing CRLF semantics, used
valid UTF8+NUL for binary rather than unknown-encoding routing, and stopped a test
from manually invoking an already-executed dispatcher callback twice (panic).
All initial logs remain. After03 only the narrow shutdown-after-close-dialog
guard changed runtime;05 covers it. After05 only one new menu/shortcut test was
added and run by06. Total18 focused top-level groups; no full app run or vet.

Coverage includes current replaced UTF8/BOM/UTF16 bytes, readonly bytes and legacy
encoding, PNG adoption, SaveAs/discard/drafts, failed close/save/persist/slot and
shutdown reentry, duplicate lookup/publication Flush refusal, synchronous recovery,
newer close/reclose/eviction, modal/restore/preedit, text timeout/shutdown/late UI,
PNG timeout/cancel/duplicate, actual CanClose cancellation of a running image job,
16-entry bound/dedup/backing cleanup, real Tester source/rich composition and
Find query/replacement/GoToLine pointer focus/retry, stale menu ID and held keys.

## Geometry and source freeze

One optional synthetic capture produced and visually reviewed:
[recent-menu-760x480.png](reopen-closed-file-evidence-20261110/recent-menu-760x480.png),
45,173B, SHA256 `FFF50FEC6A4760474BE4F9EE81D54E384716C952DAA8CF823D45620915C16AAE`.
Toolbar controls, menu limitation/header, two file rows, path subtitles and
accelerator fit; zero-tab welcome remains. This is public Tester geometry only.

Final exact identities are in
[source-frozen.json](reopen-closed-file-evidence-20261110/source-frozen.json):
7 files,124,304B; manifest SHA256
`5875F6B4BC504348475E71427DD25CE56ED89C255177DF219FF6FD3D486F464E`.
05 source-before/source-after matched all7 exactly; subsequent test-only addition
is explicitly represented in final manifest below.

| File | Bytes | SHA256 |
|---|---:|---|
| app/app.go |38029| C1061C650ED8A5FF3231F3642B003225D2155508F1C8B435B40981CE5052F1D3 |
| app/reopen_closed.go |7895| BFCA75BA0E2A878483133ECE43530742C424623CBE6B8809F1285C9E04C62E92 |
| app/binary.go |10170| AE2F059F0E91C812A82FFE3F0317EAFE7E2F0B0CE9AEB3AA5056F713C555226D |
| app/binary_actions.go |9626| 84129D789A02A16B57B467A52E5C6B0E99278FE7A4F5F8DBBC02ED06EB701E11 |
| app/view.go |23002| 195AFA3D82E7612A0CDBEC6A9D0F0432777F492CAB4D94596B0C6F2329C83456 |
| app/workspace.go |6715| B1492575D6C64673BF02E633C0D7049F47BF151391B4CA58F58F3779599AF259 |
| app/reopen_closed_test.go |28867| 3D160E6CCA9602F285D1F0A19A1DC450EE8109F5E3B81DEDDC33C6DF037B2C0B |

Independent static review reports no concrete outstanding defect and matches all
six runtime identities; its earlier test hash predates only the06-added case.
No core/session/format/decoder/vendor changes, no product docs, package/build/host,
physical input, driver/resource run, children or .26 evidence mutations. Live-tab
cap is excluded and awaits its separate owner. Implementation is now frozen.
