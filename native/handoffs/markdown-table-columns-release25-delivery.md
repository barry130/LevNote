# Markdown table columns .25 delivery — frozen for root acceptance

Completed 2026-11-10. Final aggregate **PASS 13/13 on attempt 1, exit 0**.
One package invocation produced five GUI builds; one exact copied hidden host
completed pre-close recovery and normal close. All writers have exited and
the external evidence inventory is closed. No failure, retry or metadata repair
was required in this delivery. Root owns acceptance/current-launch docs.

## Package

Candidate: `native/bin/levnote-native-worker-20261110.25`, version
`native-worker.20261110.25`.

| Artifact | Bytes / count | SHA256 or inventory identity |
|---|---:|---|
| Complete candidate | 26 files / 27,259,765 B | `07391CBF37FF6406D5D4279CCED96CDBA0FDD3F5829B9685AA0066AA46CC7204` |
| `levnote-native.exe` | 14,443,520 B | `2421D8FC98B144DCCB0ECC1CFF239A18E11DDEF05F12FF0505B307058419AE49` |
| `release-manifest.json` | 4,674 B | `CA064C3E7824EDCB179465CD8E60D982D057D2EA1D45A0E8407BD44508CF9BC2` |
| Scoped build inputs | 385 files / 4,177,632 B | `7253A48B1A393CAB2A1542088428890E22410C2EDA992C1518B02E9FAD608CF1` |
| Portable README source | 15,433 B | `D4F49C27D53BEF2046A9EFD71334D2A23BD74B3EE67E8A4A92DF2AD1F8CA79ED` |

The manifest is schema 1 with 25 payload entries, excluding itself. All five
Windows amd64 PE32+ GUI builds exited 0 with empty build streams. Four fixed
engine manifests, 14 actual portable source copies, notices/licenses, local links
and both dependency patches passed. The package stage took 11.717 seconds.

The portable README now documents before/after/delete-column, 1–8 columns,
preserved raw cells/line endings, F6/Tab/ShiftTab/Enter/Space, separate cell and
structure undo, and explicit refusal/whole-table source editing. Existing
image/recovery/worker contracts remain. No runtime source changed for delivery.

[Table source acceptance](markdown-table-columns-root-acceptance.md) and
[final integration](markdown-table-columns-final-validation-result.md) establish
table editing/persistence. Their exact 11 identities plus the unchanged 15
image/autosave identities were checked through packaging, host and aggregate.
The source tests/vet were inherited, never repeated for delivery.

## One hidden host

[Host result](../artifacts/acceptance/markdown-table-columns-release25-20261110/host-smoke/run-1-mixed-cli-open/result.json)
is PASS. The byte-identical copied host ran in a fresh isolated C: configuration
with no prior session. It opened accepted UTF-8+BOM/CRLF `.diagramnote` text
through the installed custom-format JSON, then PNG. Exact held process image
and owned window were checked (PID 23240, held handle 2404, window 2430280).

- Process start: `2026-11-10T07:37:28.1978167Z`.
- Alive settling: 5,012.6499 ms, 46 alive checks.
- Strict fresh pre-close session captured: `2026-11-10T07:37:33.5751458Z`.
- Single owned WM_CLOSE: `2026-11-10T07:37:33.5804387Z`.
- Session: 675 B; SHA256
  `21A967D39B5F1D2A0C530C776D30782969ED72EE58A507EA588D8F60AE2A3E2C`.
- Version 1 / active index 1; clean UTF-8+BOM `diagram-notes` preview first,
  path-only binary/png/image preview second, with no draft or binary payload.

[Pre-close bytes](../artifacts/acceptance/markdown-table-columns-release25-20261110/host-smoke/run-1-mixed-cli-open/session-before-close.json),
[capture facts](../artifacts/acceptance/markdown-table-columns-release25-20261110/host-smoke/run-1-mixed-cli-open/session-before-close-facts.json)
and [normal-close bytes](../artifacts/acceptance/markdown-table-columns-release25-20261110/host-smoke/run-1-mixed-cli-open/session-after-close.json)
are separately preserved. Normal close published the same bytes with a newer
write timestamp (639298930484978095 → 639298930535853579 ticks). Host exit 0,
Job total 1 / active 0 / terminated 0, no forced cleanup, no owned live process,
process and Job handles disposed. Process stdout/stderr are empty. Candidate,
copied bundle, fixtures, installed plugin, workspace and source identities match.

The retained isolated configuration is
`C:\Users\canace\AppData\Local\Temp\levnote-native-release25-columns-smoke-0b8839ade1e346dfb8b62986dcb5ab91`.
This host proves package lifecycle and clean-session dispatch before close.
It does not establish physical table editing, dirty/minimized input recovery,
PNG rendering, physical OS IME, drivers or resource improvements. No second
restore launch, worker request, forced input/repaint or capture occurred.

## Evidence and closure

Evidence root:
`native/artifacts/acceptance/markdown-table-columns-release25-20261110`.
Twelve helpers came from actual corrected .24 evidence; source fingerprints are
in `copied-helper-provenance.json`. Direct DateTimeOffset conversions and explicit
inventory summation loops were retained. One static parse covered 14 scripts,
including the external closure helper: zero errors, exit 0. Initialization,
baseline, package, host, aggregate and external closure each exited 0. Exact
commands, streams, start/end times and exit records are saved; aggregate-stage
writer logs are in its separate sibling directory.

| Evidence artifact | Bytes | SHA256 |
|---|---:|---|
| [Final aggregate](../artifacts/acceptance/markdown-table-columns-release25-20261110/aggregate.json) | 49,349 | `993C171AD4BD07772CE6695C1A9F7B62275504708C7CADA56F4B7A152E1FE2A5` |
| [Attempt-1 validator](../artifacts/acceptance/markdown-table-columns-release25-20261110/aggregate-attempts/attempt-1/validator.json) | 11,505 | `D8E187B41D779F4A5D21ECA57351CBB89B7BBCFFB1AE1C4B212F91A4FCD8B0F9` |
| [Static parse](../artifacts/acceptance/markdown-table-columns-release25-20261110/static-parse-result.json) | 4,498 | `635DFCCD30441130D5858CE55C9036035A3BA78401A6109D48C61BF1151DAEC5` |
| [Closed external inventory](../artifacts/acceptance/markdown-table-columns-release25-20261110-closed-inventory.json) | 66,343 | `39BADBA047C552252A13AD7645FA19D93B70DDF4FA8414FEB9B8E8362B3C99DF` |

[Closure result](markdown-table-columns-release25-close-inventory-result.json)
records exit 0 at `2026-11-10T07:38:49.8454376Z`; its
[stdout](markdown-table-columns-release25-close-inventory.stdout.log) records
the final fingerprint and explicit totals. Closure first checked every recorded
aggregate evidence fingerprint and all completed stage exits, then inventoried
the roots after their writers had ended:

| Closed root | Files | Bytes | Inventory identity |
|---|---:|---:|---|
| Direct .25 evidence | 146 | 55,539,894 | `F383C6A160D3B56F55A1EEF0B4A083693C1C09EF2F97910AE06C37DDC5BC9646` |
| Attempt-1 capture sibling | 6 | 3,354 | `FCD2AE8E96784802DBA7C3A193D350634F1F8E281EC97507799B70CA3E27ABAF` |
| Total | 152 | 55,543,248 | Per-root exact records in the closed inventory |

The complete accepted .24 package, direct evidence, both capture siblings and
both original/corrected external inventories retain accepted bytes and times.
The .25 package, portable input, helpers and closed evidence are now frozen.
No .26 or further product execution follows from this task.
