# Native reading .24 delivery — frozen for root acceptance

Completed 2026-11-10. Final aggregate **PASS 13/13, exit 0**. The package,
product evidence, both aggregate capture siblings and external inventory are
closed. Root owns acceptance and any current-launch documentation update.

[Source acceptance](native-autosave-keyboard-root-acceptance.md) supplies the
corrected affected app/imageview tests and vet; those checks were not repeated
during delivery. The exact current 15 source files and scoped build inputs
matched before and after delivery. Earlier image component acceptances remain
references, without comparing new app source to superseded .23 hashes.

## Package and inputs

Candidate: `native/bin/levnote-native-worker-20261110.24`, version
`native-worker.20261110.24`. One existing package invocation produced five GUI
PEs, each build exit 0 with empty build stdout/stderr. Four engine manifests,
14 portable source copies, notices/licenses and both patches passed validation.
The release manifest is schema 1 with 25 payload entries, excluding itself.

| Artifact | Bytes / count | SHA256 or inventory identity |
|---|---:|---|
| Complete candidate | 26 files / 27,226,613 B | `C17C919EB5F71C0643CAA9BBF1F5C9CCD509EE2CC91DD8A7FD704F15D06427B5` |
| `levnote-native.exe` | 14,410,752 B | `A4D679FB0F35DB9258410DCC35995853780DB4E915C928E8D6434A92E6A0BD08` |
| `release-manifest.json` | 4,674 B | `B1609AE10D15DBEA63DD40CC7E699E131E49156FC48821D8FD63891E565D935E` |
| Scoped build inputs | 380 files / 4,106,059 B | `FCD190C6A53D8D4AA818FCE6D0AB4465DE09A3A31F684038016AFEAAB44CB628` |
| Portable README source | 15,049 B | `B10440FF39A3CAC9744BEDC36961BB54B86BD103E96EB05D183B4562DEC5B260` |

The README records focused image controls, toolbar focus return, explicit 100%
mode and paint-independent recovery scheduling after 900 ms quiet time. UI
dispatch may delay persistence; uncommitted format drafts still require explicit
completion, and recovery never autosaves to original files. Helpers came from
the actual corrected .23 delivery. Their one preparation parse covered 13
scripts under PowerShell 7.6.9: zero errors, exit 0. This parse predates the
preserved three-expression aggregate metadata correction described below.

## One hidden host and pre-close recovery

[Host result](../artifacts/acceptance/native-reading-release24-20261110/host-smoke/run-1-mixed-cli-open/result.json)
records exactly one hidden copied host in a fresh isolated C: configuration,
initially without `session.json`. It opened the ordered UTF-8+BOM/CRLF
`.diagramnote` fixture and PNG. The owned process remained alive through
5,015.0757 ms settling. A newly written strict session was captured before
the single owned `WM_CLOSE`:

- Start: `2026-11-10T06:42:07.5356941Z`.
- Pre-close capture: `2026-11-10T06:42:12.8862702Z`.
- Close requested: `2026-11-10T06:42:12.8920834Z`.
- Session: version 1, active index 1; clean `diagram-notes` preview first,
  path-only `binary`/`png`/`image` preview second, without drafts or binary payload.
- Exact pre-close bytes: 659 B, SHA256
  `800045175388CAB4DF70B11A5650D8072CA70CB8A7AE0EF02C943B71C5CE2E04`.

[Pre-close bytes](../artifacts/acceptance/native-reading-release24-20261110/host-smoke/run-1-mixed-cli-open/session-before-close.json)
and [capture facts](../artifacts/acceptance/native-reading-release24-20261110/host-smoke/run-1-mixed-cli-open/session-before-close-facts.json)
are preserved separately from
[normal-close bytes](../artifacts/acceptance/native-reading-release24-20261110/host-smoke/run-1-mixed-cli-open/session-after-close.json).
Normal close published the same strict bytes with a newer write timestamp;
`FreshNormalClosePublication` is true. Host exit 0, no forced cleanup, Job
total 1 / active 0 / terminated 0, held process and Job handles disposed, no owned
live process, app stdout/stderr empty. Candidate/copied-bundle, fixture, plugin,
workspace and current-source identities stayed unchanged.

This is a narrow real UI-dispatch check of clean mixed-session persistence before
close. It does not establish dirty-draft or minimized physical-input recovery,
PNG rendering, physical keyboard/IME, driver behavior or resource improvements.
There was no second restore launch, worker request, forced repaint/input, capture,
benchmark or additional test/build run.

## Preserved metadata failures and final evidence

Aggregate attempt 1 exited 1 with 12/13. Its sole failed predicate was
`CaptureBeforeClose`: deserialized `DateTime` values were converted to strings
before reparsing, dropping fractional seconds. The raw capture and close times
above were already correctly ordered. The original aggregate/helper and all
16 diagnostic predicates were preserved. Root authorized one metadata-only
attempt 2. Three scoped time conversions now cast directly to `DateTimeOffset`
and normalize to UTC, preserving ticks/timezone. No product operation repeated.

- [Attempt 1 aggregate](../artifacts/acceptance/native-reading-release24-20261110/aggregate-attempts/attempt-1/aggregate.json),
  [saved predicates](../artifacts/acceptance/native-reading-release24-20261110/aggregate-attempts/attempt-1-preclose-diagnostic.json),
  [helper correction](../artifacts/acceptance/native-reading-release24-20261110/helper-corrections/aggregate-time-precision/correction.json).
- [Attempt 2 validator](../artifacts/acceptance/native-reading-release24-20261110/aggregate-attempts/attempt-2/validator.json):
  13/13, 10,605 B, SHA256 `07ACBC85744DA61AC6348D156294F7226B6026D148BBE637047B1DC282FFF77F`.
- [Final aggregate](../artifacts/acceptance/native-reading-release24-20261110/aggregate.json):
  51,028 B, SHA256 `D512E569D75D5250F69A6F358D9455BB48E56A8444A439BA53D4B93233E3DA9E`.

After all stage writers exited, the external closed inventory recorded the direct
evidence root and both sibling aggregate captures. Its initial top-level totals
were zero because `Measure-Object` did not read ordered-dictionary keys; all saved
per-root arrays/counts/identities were correct. Root authorized preserving that
external original and deriving only the footer totals from saved per-root facts
with explicit loops. The correction retained every product, aggregate and
evidence-root byte. Final read-only verification passed at
`2026-11-10T06:48:53.6982122Z`.

- [Original external inventory](../artifacts/acceptance/native-reading-release24-20261110-closed-inventory-attempt-1.json):
  72,432 B, SHA256 `0C575F33A29EB69C39FD181750D80540D3B23A94FFA32A80DD498C0BE7C902A8`.
- [Corrected external inventory](../artifacts/acceptance/native-reading-release24-20261110-closed-inventory.json):
  73,016 B, SHA256 `D9FA85BBC0F9906FF75874B1851ACD0E1046256CD5B5FB99D29976F2CE94328C`.
- [Corrected closure result](native-reading-release24-close-inventory-result.json)
  records the saved correction, artifact fingerprints and final verification exit 0.

| Closed evidence root | Files | Bytes | Inventory identity |
|---|---:|---:|---|
| Direct .24 evidence | 155 | 55,785,316 | `E44D08DF592A38A7BF26309E3E50052F9EE39A61330B54B18ED7FA1146DF36A9` |
| Aggregate capture attempt 1 | 6 | 2,851 | `EFE0346259C5BD19709646F30CBDADAFD8484A4FB20BEA03A851746E9A6C7870` |
| Aggregate capture attempt 2 | 6 | 3,274 | `F37278A498615CB9821DA29758A0FACBE8FE2479A6FBE0167BBF466183E656C4` |
| Total | 167 | 55,791,441 | Per-root records retained in the external inventory |

Complete .23 package, direct evidence, both capture siblings and external closed
inventory retain their accepted identities. The .24 runtime, portable build
input, package and evidence are frozen; no further product execution or evidence
rewrite is authorized by this delivery task.
