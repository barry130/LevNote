# Native image `.23` preparation checkpoint

2026-11-10. Status: final helper and portable README identities are frozen for
the separately named scoped parse after review item 14. Root's
`native-image-release23-hidden-readiness-decision.md` resolves review notes
10 and 13: the hidden host helper now uses a bounded alive-settling period,
treats any pre-close session observation as descriptive, and requires the
strict fresh mixed-session publication only after the single close. No delivery
execution is activated. No package, build, host, worker, resource, screenshot,
or product source execution has been performed by this preparation task.

The prior static parse is preserved as `static-parse-final-20261110.json`; it
passed before item 14 and is superseded by the corrected-file parse recorded in
the final result. No helper identity from the earlier parse is being mistaken
for the frozen set.

## Frozen helper identities

All paths are under
`native/handoffs/native-image-release23-preparation-20261110/`.

| File | Bytes | SHA-256 |
|---|---:|---|
| `aggregate.ps1` | 34089 | `9750DAC0D4F696E75D75A924968A6EFB27A8870E37DFE0E4F0E3BEFDD4CD5524` |
| `baseline.ps1` | 10582 | `88508D9B9F18D501C49EFF3E7DF3374A93A9962D3FA3029ED25AB54A928E925E` |
| `delivery-lib.ps1` | 21192 | `3CC43ADB150FBFCB42848C20A77C3D30FDEA0F880E55E95E94AF03662061BC4A` |
| `initialize-context.ps1` | 14720 | `75F7F44E7E0462FC392FB627693F373A64F166460C00EDCBFEF2E32883498F2D` |
| `invoke-stage.ps1` | 2653 | `97369BF3581B5A31706C87DEE3F584D77FA938D4D47B46D1C98E92CB5B151DA7` |
| `run-host-smoke.ps1` | 31257 | `16B2CF59FA52947B32166A6FC192F4D821CB6D2DC0FEB8E1394BBB1EDA529781` |
| `run-package.ps1` | 16751 | `1E3F7573E862BF7CB6F6C4C9788E685A6D6B48BD014D7133BFB7F24B797BB567` |
| `run-stage.ps1` | 2958 | `9631549F240DD5E30A3B7DC8F39F1C4566903A1FF59CE5FABBC7A32B0F31F59A` |
| `source-scope.ps1` | 2157 | `6CEFB378F9027F66892560CE92D3199C1F5F8C5D12FD32E3F40E557A78DE5FC5` |
| `static-parse.ps1` | 1666 | `FC9EA3E3AB5A6A36A80880C9C62E85D495B7E892E3ADE4758B3EEF0AAC2D9E68` |

`native/portable/README.md`: 14185 bytes, SHA-256
`D9096B53268B5D0C15B859DB6DB33F13945CD73D1AD251518979698FECC88141`.

The new directory was adapted from the accepted
`native/handoffs/tab-workflow-release22-preparation-20261110/` harness. The
accepted `.22` helpers were not modified. Their frozen source identities are
preserved in that directory; the `.22` package, 134-file direct evidence,
6-file aggregate capture, and closed inventory remain the protected delivery
baseline.

## Corrections completed

- Initialization now binds eight component/source acceptances:
  raster admission, raster decode, binary core, binary copy receipt, binary
  session, bitmap lifecycle, native image view, and narrow-toolbar layout.
  Composition and final-app acceptance remain separate mandatory gates.
- The README describes saved-absolute-path recovery and one-at-a-time image
  read/decode/copy in plain Chinese. It excludes the former `path-only`,
  `single-lane`, and `单车道行为` labels. It records the baseline 8-bit JPEG
  grayscale/standard-three-component subset, CMYK/progressive exclusion,
  16 MiB/8192-side/8,388,608-pixel bounds, 64 KiB bounded byte inspection for
  oversized sources, unavailable bounded copy for those sources, retained old
  image/source during replacement or a native Save As dialog, and no RSS/VRAM
  guarantee. Initialization, baseline, and aggregate share one required and
  forbidden prose contract.
- Each hidden run captures its exact pre-run session state before process
  start. Run 1 must find no session. Run 2 must find the exact preserved run-1
  publication and byte/time-identical baseline copy. The pre-run copy
  reconstructs its timestamp with `[DateTime]::new(ticks, DateTimeKind.Utc)`;
  file-state ticks use the year-1 `DateTime` epoch, while `FromFileTimeUtc`
  expects the year-1601 Windows FILETIME epoch.
- Each run checks that the exact hidden host remains alive through one bounded
  5000 ms settling period. It may record one pre-close session observation, but
  absence or an old shape is descriptive and cannot postpone the one owned
  `WM_CLOSE`. No pixels/decode success is inferred from settling or session
  state.
- The aggregate permits `SessionFactsBeforeClose` and
  `SessionBeforeCloseFingerprint` to be null and validates their fields only
  when an observation object exists. Post-close facts and fingerprint remain
  mandatory. The owned `Process` object is explicitly disposed after its exit
  facts and session captures are recorded, including a recorded failure path;
  `ProcessDisposed` is part of the run/aggregate schema.
- A host run cannot report `Status=PASS` unless its post-close session is a
  fresh publication after that process start and differs from the captured
  baseline and contains the exact ordered text+binary shape. Run 2 additionally
  verifies its baseline against run 1. Aggregate independently checks settling,
  one close, fresh publication, and the same recorded facts.
- Sequential lifetime evidence now uses process creation/completion ordering:
  run 1 started and completed before run 2 started. PID and handle values remain
  descriptive only.
- Aggregate writes validator/support/aggregate files under
  `aggregate-attempts/attempt-N/` and stage output under a distinct
  `...-aggregate-capture-attempt-N/` directory. Attempt 1 is required first.
  At most attempt 2 is allowed after a preserved failed attempt, including an
  exception before attempt 1's aggregate JSON. A successful root
  `aggregate.json` prohibits further attempts. Failed files are not replaced,
  and no baseline/package/host stage is rerun for aggregate metadata repair.
- Runtime parameter traps in aggregate `Test-Path` and `Join-Path` list
  expressions were corrected before this freeze.

## Product execution plan, not activated

Final delivery must first copy the reviewed helper set into the fresh
`native/artifacts/acceptance/native-image-release23-20261110/` directory, then
initialize and execute from those copied artifacts. The preparation directory
is source evidence and must not execute delivery. The copied
`initialize-context.ps1` must supply these eight root source acceptance files
together with the separate composition and final-app acceptances, the accepted
893-byte text fixture, the 17635-byte image fixture, the fresh `.23` candidate
path, and the copied delivery directory as evidence:

```powershell
& F:\AIproject\LevNote\native\artifacts\acceptance\native-image-release23-20261110\initialize-context.ps1 `
  -Version native-worker.20261110.23 `
  -CandidateDirectory F:\AIproject\LevNote\native\bin\levnote-native-worker-20261110.23 `
  -RootSourceAcceptancePath @(
    F:\AIproject\LevNote\native\handoffs\raster-admission-root-acceptance.md
    F:\AIproject\LevNote\native\handoffs\raster-decode-root-acceptance.md
    F:\AIproject\LevNote\native\handoffs\binary-core-root-acceptance.md
    F:\AIproject\LevNote\native\handoffs\binary-copy-receipt-root-acceptance.md
    F:\AIproject\LevNote\native\handoffs\binary-session-root-acceptance.md
    F:\AIproject\LevNote\native\handoffs\image-bitmap-lifecycle-root-acceptance.md
    F:\AIproject\LevNote\native\handoffs\native-image-view-root-acceptance.md
    F:\AIproject\LevNote\native\handoffs\native-image-view-narrow-toolbar-root-acceptance.md
  ) `
  -CompositionAcceptancePath F:\AIproject\LevNote\native\handoffs\native-image-composition-root-acceptance.md `
  -AppAcceptancePath F:\AIproject\LevNote\native\handoffs\native-image-app-root-acceptance.md `
  -MixedTextFixturePath F:\AIproject\LevNote\native\artifacts\acceptance\tab-workflow-release22-20261110\host-smoke\workspace\route-gate.diagramnote `
  -ImageFixturePath F:\AIproject\LevNote\native\handoffs\native-image-view-evidence-20261110\imageview-alpha-toolbar-light-760x480-review-fix.png `
  -EvidenceDirectory F:\AIproject\LevNote\native\artifacts\acceptance\native-image-release23-20261110 `
  -ExecuteDelivery
```

After separate root activation, execute once in order:

```powershell
pwsh -NoProfile -File F:\AIproject\LevNote\native\artifacts\acceptance\native-image-release23-20261110\run-stage.ps1 -Name baseline
pwsh -NoProfile -File F:\AIproject\LevNote\native\artifacts\acceptance\native-image-release23-20261110\run-stage.ps1 -Name package
pwsh -NoProfile -File F:\AIproject\LevNote\native\artifacts\acceptance\native-image-release23-20261110\run-stage.ps1 -Name host-smoke
pwsh -NoProfile -File F:\AIproject\LevNote\native\artifacts\acceptance\native-image-release23-20261110\run-stage.ps1 -Name aggregate -AggregateAttempt 1
```

Only if aggregate attempt 1 preserves a failed invocation may the explicitly
chosen metadata-only retry run:

```powershell
pwsh -NoProfile -File F:\AIproject\LevNote\native\artifacts\acceptance\native-image-release23-20261110\run-stage.ps1 -Name aggregate -AggregateAttempt 2
```

## Remaining dependencies

Actual product execution still requires separate root activation. The exact
`.23` candidate count, manifest payload count, five GUI build results, four
engine manifests, five PE facts, both hidden-run schemas, and final aggregate
outcome cannot be established by static parsing. No physical input/IME, image
rendering, worker request, or RSS/VRAM claim is planned. Current runnable
delivery remains immutable `.22`.
