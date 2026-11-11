# Native image `.23` preparation result

2026-11-10. Status: preparation helpers, static parse evidence, and the
portable image section are frozen for root review. Product delivery is not
activated. This task did not package, build, launch a host, request a worker,
run tests/vet/captures/benchmarks, or perform a resource run. The planning task
`native-image-release23-delivery-task.md` remains `PLANNED`.

## Frozen identities

Helper paths are under
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

## Static parse

The final corrected-file parse command was:

```powershell
pwsh -NoProfile -File F:\AIproject\LevNote\native\handoffs\native-image-release23-preparation-20261110\static-parse.ps1 -ResultPath F:\AIproject\LevNote\native\handoffs\native-image-release23-preparation-20261110\static-parse-corrected-20261110.json
```

It parsed 10 scripts with 0 parse errors, `Pass=true`, empty stderr, and exit 0
under PowerShell 7.6.9. The result `CreatedAtUtc` was
`2026-11-10T05:25:27.4996572Z`.

| Evidence | Bytes | SHA-256 |
|---|---:|---|
| `static-parse-corrected-command.txt` | 251 | `A186BD9349CAE95D35038C9557429B8312746A37B5F97D9556AEC9CDCDEF9ED6` |
| `static-parse-corrected-stdout.txt` | 3166 | `E3979DBE382D903EA21874B86FA6666B4E624DA0DFAA058AC84C99B30ACD7DC1` |
| `static-parse-corrected-stderr.txt` | 0 | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |
| `static-parse-corrected-exit.txt` | 3 | `13BF7B3039C63BF5A50491FA3CFD8EB4E699D1BA1436315AEF9CBE5711530354` |
| `static-parse-corrected-20261110.json` | 3165 | `1BAABD32C02EA0C21F478DDA95F631A1D3DC3A35824859B0E17C7ED38E884043` |

The earlier successful parse remains preserved as
`static-parse-final-20261110.json` (3165 bytes, SHA-256
`7EC8E430DCC72FF0F60306CA10E1C2F29BC6D89A72F63813F9E062E698744671`)
with 0 errors and exit 0. Review item 14 changed the optional pre-close schema
and process disposal, so that parse is historical and the separately named
corrected-file parse above is the frozen identity.

## Acceptance gates and helper behavior

`initialize-context.ps1` binds these eight component/source gates:

1. `raster-admission-root-acceptance.md`
2. `raster-decode-root-acceptance.md`
3. `binary-core-root-acceptance.md`
4. `binary-copy-receipt-root-acceptance.md`
5. `binary-session-root-acceptance.md`
6. `image-bitmap-lifecycle-root-acceptance.md`
7. `native-image-view-root-acceptance.md`
8. `native-image-view-narrow-toolbar-root-acceptance.md`

It separately requires composition acceptance
`native-image-composition-root-acceptance.md` and final app acceptance
`native-image-app-root-acceptance.md`.

The two planned hidden runs use one byte-identical copied candidate and one
isolated C: config. Run 1 requires an absent pre-run session and opens the
UTF-8/BOM+CRLF `route-gate.diagramnote` followed by the reviewed light image
fixture. Run 2 has no CLI paths and requires its preserved pre-run session to
equal run 1's publication byte-for-byte and timestamp-for-timestamp.

Each run finds only its exact held process/window, checks that process remains
alive through one bounded 5000 ms settling period, records an optional
pre-close session observation descriptively, posts exactly one owned
`WM_CLOSE`, and does not retry or post a second close. A PASS requires normal
exit 0, no forced cleanup, zero remaining owned processes/worker requests, and
a fresh ordered text+binary session publication after that process started.
The post-close session requires active index 1, original UTF-8/BOM text data,
`png`/`image`/`preview` binary data, absolute paths, and no forbidden source,
pixel, bitmap, or encoded binary payload fields. Run 2 additionally requires
the fresh publication to differ from its captured baseline. Null optional
pre-close facts are accepted; post-close facts remain mandatory.

The baseline copy reconstructs its timestamp with
`[DateTime]::new(ticks, DateTimeKind.Utc)`. File-state ticks use the year-1
`DateTime` epoch; `FromFileTimeUtc` expects the year-1601 Windows FILETIME
epoch and would break the exact run-2 baseline comparison. The owned
`Process` object is explicitly disposed after exit facts and captures are
recorded on both success and recorded failure, and `ProcessDisposed` is
enforced by the aggregate. Sequential lifetime proof uses run-1 creation and
completion before run-2 creation; PID and handle inequality are descriptive
only.

The aggregate requires attempt 1. It may accept only one explicitly selected
attempt-2 metadata retry when attempt 1 left a preserved failed invocation,
including a failure before `aggregate.json` existed. Attempt output and
capture directories are distinct, prior failures are not replaced, a
successful aggregate prohibits rerun, and neither baseline/package nor host
work is repeated for an aggregate retry.

The portable image section is plain Chinese. It describes PNG/JPEG readonly
display, fit/100%/zoom/pan, original-byte new-destination copy, byte fallback,
plugin enable/disable, saved-absolute-path rereading, one-at-a-time image
operations, the actual 16 MiB/side/pixel/format subset, unavailable bounded
copy above 16 MiB, retained old image/source during replacement or native Save
As, and no RSS/VRAM guarantee.

## `.22` provenance and protection

The new helpers were adapted without modifying the accepted source directory
`native/handoffs/tab-workflow-release22-preparation-20261110/`. The accepted
`.22` delivery remains the protected baseline:

- package: `native/bin/levnote-native-worker-20261110.22`, 25 files,
  26,863,393 bytes, inventory
  `BA43209FF7F6CFBEDD8287DBEA7B642008AFA839D133FECD8CC5C9D065FD9CF6`;
- direct evidence: 134 files, 54,340,909 bytes, inventory
  `B862D886E30F00D4364972DC8521AA75EB2C266DF81B1250DF1F42A2D1B1CC5C`;
- aggregate capture sibling: 6 files, 2684 bytes, inventory
  `E6D61DCF9B02D9D32F3DC4957DA4747506E0A7E942DD62C55C91B9107011DFE3`;
- closed inventory: 57,245 bytes, SHA-256
  `E61FB97AA24C5DAD6673F365D32EB77398C0D98EE32FE456A13F9B45A75373E2`.

## Frozen activation commands

Delivery must start from a fresh
`native/artifacts/acceptance/native-image-release23-20261110` directory and
copy the reviewed `.ps1` helpers there before initialization. The preparation
directory is source evidence and must not execute delivery.

```powershell
$preparation = 'F:\AIproject\LevNote\native\handoffs\native-image-release23-preparation-20261110'
$delivery = 'F:\AIproject\LevNote\native\artifacts\acceptance\native-image-release23-20261110'
if (Test-Path -LiteralPath $delivery) { throw "Fresh delivery directory already exists: $delivery" }
New-Item -ItemType Directory -Path $delivery -ErrorAction Stop | Out-Null
Get-ChildItem -LiteralPath $preparation -File -Filter '*.ps1' |
  Copy-Item -Destination $delivery -ErrorAction Stop

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

After separate root activation, execute these copied helpers once in order:

```powershell
pwsh -NoProfile -File F:\AIproject\LevNote\native\artifacts\acceptance\native-image-release23-20261110\run-stage.ps1 -Name baseline
pwsh -NoProfile -File F:\AIproject\LevNote\native\artifacts\acceptance\native-image-release23-20261110\run-stage.ps1 -Name package
pwsh -NoProfile -File F:\AIproject\LevNote\native\artifacts\acceptance\native-image-release23-20261110\run-stage.ps1 -Name host-smoke
pwsh -NoProfile -File F:\AIproject\LevNote\native\artifacts\acceptance\native-image-release23-20261110\run-stage.ps1 -Name aggregate -AggregateAttempt 1
```

Only after aggregate attempt 1 preserves a failed invocation may the selected
metadata-only retry run:

```powershell
pwsh -NoProfile -File F:\AIproject\LevNote\native\artifacts\acceptance\native-image-release23-20261110\run-stage.ps1 -Name aggregate -AggregateAttempt 2
```

Static parsing cannot establish the future `.23` candidate count, manifest
payload count, five GUI build results, four engine manifests, five PE facts,
the two hidden-run results, or aggregate outcome. Actual package/host execution
still requires a separate root activation after this preparation is accepted.
