# `.21` drawio efficiency delivery preparation

**Status: PREPARED; EXECUTION INACTIVE; HARNESS FROZEN.**

No product build, source suite/vet, tester compile, worker request, host launch,
UI/input run, capture, benchmark, resource measurement, package invocation, or
product inventory was performed. The `.21` candidate is absent. Execution waits
for root to accept warning and source retention, release the source freeze, and
explicitly activate this harness.

## Harness

Preparation root:
`native/handoffs/drawio-efficiency-release21-preparation-20261109-193152`

At activation, `initialize-context.ps1 -ExecuteDelivery` creates a fresh
evidence directory, records the supplied warning/retention result, review, and
acceptance gates (at least six; maximum three per category), plus accepted
source-retention benchmark fixtures in the scoped source set. The candidate must
be absent and exactly:

`native/bin/levnote-native-worker-20261110.21`

The evidence-local stage harness runs once each, in this order:

1. `run-stage.ps1 -Name baseline`
2. `run-stage.ps1 -Name package`
3. `run-stage.ps1 -Name managed-smoke`
4. `run-stage.ps1 -Name host-smoke`
5. `run-stage.ps1 -Name aggregate`

Actual gates:

- **baseline:** capture scoped current source/build inputs, accepted gate files,
  benchmark fixtures, and the managed-smoke tester source after freeze; require
  the fresh `.21` candidate to be absent; protect the accepted `.20` package and
  complete 139-file direct evidence.
- **package:** one `package-markdown-worker.ps1` invocation, producing exactly
  five Windows amd64 GUI builds: `worker-markdown`, `worker-mindmap`,
  `worker-mindmap-preview`, `worker-drawio`, and `host`. Require exit 0, empty
  build streams, 25 candidate files, 24 payload entries in the schema1 `.21`
  manifest, four fixed engine manifests/identities, five GUI PE images, portable
  source copies and authored links, one README version marker, and the three
  schema3 routes. Require the exact byte-identical package copy.
- **managed-smoke:** make a complete byte-identical candidate bundle copy;
  compile the tester once through `native/scripts/go.ps1`; enable and run
  `TestDrawioManagedRealBundleSmoke` once. Require the 516-byte second-page
  source, the unchanged 598-byte SVG oracle
  `3F990A4AAC3209BAFAD94F44026BF14F3CEC27EA44BDBD3169555947936472BF`,
  page index `1`, ID `second`, zero warnings, no fallback/failure, one joined
  worker lifecycle, and a candidate-derived bundle of 26 files after adding
  only the tester.
- **host-smoke:** from a separate byte-identical copied bundle, start exactly one
  hidden exact `.21` host on the reviewed `.diagramnote` custom-suffix fixture
  with isolated `C:` APPDATA/LOCALAPPDATA and the reviewed 179-byte
  `diagram-notes.json`. Require one native window, exactly one owned `WM_CLOSE`,
  exit 0, one clean preview tab with UTF-8+BOM and no draft/terminal newline,
  zero worker operations, and unchanged fixture/plugin/bundle identities.
- **aggregate:** recheck candidate and protected `.20` identities, unchanged
  scoped source/build inputs and gates, benchmark fixtures, package evidence,
  and host session. For warnings, read exact uppercase `Warnings` from both raw
  and normalized records; accept only JSON `null` or an array at the raw shape
  boundary; require a normalized empty array; validate
  `WarningPropertyKind`, zero `WarningCount`, `WarningsJson == []`, and
  `WarningsSHA256`; require zero warnings. Emit `validator.json` and
  `aggregate.json`, failing on any false gate.

No extra worker, UI automation, resource matrix, installer/cross-platform build,
source suite/vet, benchmark replay, dirty recovery, or package-input edit is
scheduled.

## Raw JSON correction

`core/engineruntime/drawio_managed_bundle_test.go` defines an untagged
`drawioSmokeRecord`, and the immutable `.19` record also emits uppercase
`Warnings`. The earlier lower-case assumption was wrong. Both
`run-managed-smoke.ps1` and `aggregate.ps1` now read exact uppercase `Warnings`;
the normalized result keeps the same uppercase name in separate
`managed-smoke/record.json` and `result.json` paths. Production Go JSON and old
evidence remain unchanged.

The strict helper uses `System.Text.Json`, case-sensitive property matching,
exactly one required property, and only JSON `null` or an array with no null
elements. Missing, duplicate, lower-case, scalar, object, null-element, wrong-root,
and malformed inputs fail. The preparation library also defines the required
`Get-ManagedRecord` marker/object extraction helper; the prior preparation draft
omitted it.

## Evidence

Earlier outputs are preserved. The correction added one no-product sanity run
and one corrected static parse.

- Earlier sanity run:
  [stdout](drawio-efficiency-release21-preparation-20261109-193152/json-shape-sanity.stdout.log),
  [stderr](drawio-efficiency-release21-preparation-20261109-193152/json-shape-sanity.stderr.log),
  [exit](drawio-efficiency-release21-preparation-20261109-193152/json-shape-sanity.exit.txt).
- Corrected sanity run:
  [stdout](drawio-efficiency-release21-preparation-20261109-193152/json-shape-sanity.corrected.stdout.log),
  [stderr](drawio-efficiency-release21-preparation-20261109-193152/json-shape-sanity.corrected.stderr.log),
  [exit](drawio-efficiency-release21-preparation-20261109-193152/json-shape-sanity.corrected.exit.txt).
  Exit `0`; stdout 1,654 bytes; stderr 0; `Pass: true`; 3 accepted and 16
  rejected cases. Accepted uppercase raw `Warnings: null`, uppercase raw
  `Warnings: []`, and uppercase normalized `Warnings: []` all serialize stably
  to the exact JSON bytes `[]`.
- Initial parse:
  [result](drawio-efficiency-release21-preparation-20261109-193152/static-parse.initial-static-parse.json),
  [stdout](drawio-efficiency-release21-preparation-20261109-193152/static-parse.initial-static-parse.stdout.log),
  [stderr](drawio-efficiency-release21-preparation-20261109-193152/static-parse.initial-static-parse.stderr.log),
  [exit](drawio-efficiency-release21-preparation-20261109-193152/static-parse.initial-static-parse.exit.txt):
  11 scripts, 0 errors, pass, exit `0`, 3,600-byte stdout, 0-byte stderr.
- Earlier final parse:
  [stdout](drawio-efficiency-release21-preparation-20261109-193152/static-parse.final.stdout.log),
  [stderr](drawio-efficiency-release21-preparation-20261109-193152/static-parse.final.stderr.log),
  [exit](drawio-efficiency-release21-preparation-20261109-193152/static-parse.final.exit.txt):
  12 scripts, 0 errors, pass, exit `0`, 3,915-byte stdout, 0-byte stderr.
- Corrected static parse:
  [result](drawio-efficiency-release21-preparation-20261109-193152/static-parse.json),
  [stdout](drawio-efficiency-release21-preparation-20261109-193152/static-parse.corrected.stdout.log),
  [stderr](drawio-efficiency-release21-preparation-20261109-193152/static-parse.corrected.stderr.log),
  [exit](drawio-efficiency-release21-preparation-20261109-193152/static-parse.corrected.exit.txt).
  PowerShell 7.6.9; 12 scripts; 0 parse errors; pass; exit `0`; 3,915-byte
  stdout; 0-byte stderr.

Post-parse read-only checks found no current hash mismatch between
`static-parse.json` and the 12 scripts, candidate absent, and no stale aggregate
symbols (`FinalStage2Files`, `AcceptedEvidenceDirectories`, `acceptedGatePaths`,
`stage2Before`, `acceptedDirectoriesBefore`, or `Release20JobAccounting`).

## Protected `.20` identities

- Package: 25 files / 26,854,177 bytes / inventory
  `6B899EFB2FB2A90CB2FB93CFE9BC48A3B94B4CFA2AB16C61C5AB005B05DF14ED`.
- Host: 14,095,360 bytes /
  `4B776F634A3626F2BDD15303E175FE38CC4908D566AC54A6D25F51D989B4F34E`.
- Manifest: 4,490 bytes /
  `53C0B71C317734001596988F6AB2BF44201879CC322BA9DB3B9AB494E553DBFA`.
- Direct evidence: 139 files / 54,304,954 bytes / inventory
  `5ED2F19531E178E33F355164546AB50C5C31F37E032F9E8B7CCBB6791C9C8621`;
  `aggregate.json`
  `C1CAEC5205F8A833A4DFF86B84485E4D84187CAC3BD6D3DA88DA1F47D7E59611`.

The accepted fixture oracle is 516 source bytes and a 598-byte SVG (SHA-256
above), page index `1`, ID `second`, zero warnings.

## Frozen final identities

33 files / 140,044 bytes:

| File | Bytes | SHA-256 |
|---|---:|---|
| `aggregate.ps1` | 25,119 | `69E12FE352AC4811B351DF36A8AB2DCC63652549E368FF169C332BEF1CDD167C` |
| `baseline.ps1` | 8,916 | `C4436F02D521F6958A25AA0BC76DD925909BAE6F3FF7EAADE439B6D3E207C620` |
| `delivery-lib.ps1` | 12,628 | `6A6FF0CCC12263770E3FA32D0A488F8FD5FB846C7DA9420EAD2D7523296C06ED` |
| `initialize-context.ps1` | 8,965 | `4121F096D0A4697CB6A17089AFF957E0EE223E6889AB0CFC21477EF0202471AE` |
| `invoke-stage.ps1` | 1,700 | `B791685AC92FE8225F1780705E9B15492601186CD833C4A85A6252D7C98914FB` |
| `json-shape-sanity.corrected.exit.txt` | 2 | `9A271F2A916B0B6EE6CECB2426F0B3206EF074578BE55D9BC94F6F3FE3AB86AA` |
| `json-shape-sanity.corrected.stderr.log` | 0 | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |
| `json-shape-sanity.corrected.stdout.log` | 1,654 | `9627327FF16F59A256839C65276C78CB3531CA12AAF0A5711D79C1FC5FE679D5` |
| `json-shape-sanity.exit.txt` | 2 | `9A271F2A916B0B6EE6CECB2426F0B3206EF074578BE55D9BC94F6F3FE3AB86AA` |
| `json-shape-sanity.ps1` | 3,930 | `95B0C0F1A45E6A644AB4A09938C2D98443B12A8538955D7A19CDB87309D7B68C` |
| `json-shape-sanity.stderr.log` | 0 | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |
| `json-shape-sanity.stdout.log` | 1,654 | `9627327FF16F59A256839C65276C78CB3531CA12AAF0A5711D79C1FC5FE679D5` |
| `README.md` | 2,681 | `E8C80A885038C6CC687723D30988A599399218B97EEA2DFC963D043628517E42` |
| `run-host-smoke.ps1` | 19,887 | `89E70EED238FE6AA44225ECFE6B831A11F5E5944B3D251F3EDF7EC6FE314E056` |
| `run-managed-smoke.ps1` | 12,441 | `BC6660252CC8E63DA0B331D59496B5356A26014A6DBA6480DF1C6A35934701B0` |
| `run-package.ps1` | 12,656 | `5553AFECED1B99052A76F0CE96E00342B15840EFCCE3DAA0A8C246ABBDD9DE3F` |
| `run-stage.ps1` | 2,062 | `C619C043ED4612572BCB4AE87165CC321188C856C1EFC599C70442B3881BBDAD` |
| `source-scope.ps1` | 1,599 | `95E9EE57648CDC88815856FD77C5C9D17F5526036FF0A270063DD2009C5B1AC7` |
| `static-parse.corrected.exit.txt` | 2 | `9A271F2A916B0B6EE6CECB2426F0B3206EF074578BE55D9BC94F6F3FE3AB86AA` |
| `static-parse.corrected.stderr.log` | 0 | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |
| `static-parse.corrected.stdout.log` | 3,915 | `C9DF7D959737A643770D05D0C85535CE360091CC6C332B25151A1965AD38D71C` |
| `static-parse.exit.txt` | 2 | `9A271F2A916B0B6EE6CECB2426F0B3206EF074578BE55D9BC94F6F3FE3AB86AA` |
| `static-parse.final.exit.txt` | 2 | `9A271F2A916B0B6EE6CECB2426F0B3206EF074578BE55D9BC94F6F3FE3AB86AA` |
| `static-parse.final.stderr.log` | 0 | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |
| `static-parse.final.stdout.log` | 3,915 | `9A30F9C8AF84CA94D87F9B02AFB2B4B9A180EDAB95CF2274014B6D49D051111B` |
| `static-parse.initial-static-parse.exit.txt` | 2 | `9A271F2A916B0B6EE6CECB2426F0B3206EF074578BE55D9BC94F6F3FE3AB86AA` |
| `static-parse.initial-static-parse.json` | 3,599 | `B5C75AAB97C7280A1FA453EF02AFD9B9088B5AE892601BF41E85216DB1A5F6E1` |
| `static-parse.initial-static-parse.stderr.log` | 0 | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |
| `static-parse.initial-static-parse.stdout.log` | 3,600 | `57F511294A6249F9F7A8B0B76065548F0B1FB8D1EE26DE57905F5797DD0B40F1` |
| `static-parse.json` | 3,914 | `858144F93BF3765AA65FDD60BA1542E051DD946CD1D893382D07A629075F8DF0` |
| `static-parse.ps1` | 1,597 | `3A9FB8EA090530348A01CC8EE5B3B3568A2230F302931DE35BEA029ED55BC47F` |
| `static-parse.stderr.log` | 0 | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |
| `static-parse.stdout.log` | 3,600 | `57F511294A6249F9F7A8B0B76065548F0B1FB8D1EE26DE57905F5797DD0B40F1` |

The preparation directory is frozen at these identities. Execution must copy
the harness into fresh evidence rather than edit this preparation source.
