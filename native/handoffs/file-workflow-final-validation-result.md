# File workflow final validation — PASS

2026-11-10. The single combined app test and vet were run once each,
sequentially, on the frozen reopen/live-admission union. Both exited0. The
pre-run and post-run inventories contain the same nine
`{Path,Length,Sha256}` identities. No source correction or evidence rerun was
needed.

## Actual runs

Both exact commands were invoked from `F:\AIproject\LevNote` through
`native/scripts/go.ps1`. No pipeline wrapped either command, so each numeric
exit is the propagated Go process exit.

| Evidence | Exact command | Exit | UTC interval | Captured output |
|---|---|---:|---|---|
| `01-test` | `& '.\native\scripts\go.ps1' -GoArguments @('test','./app','-count=1')` |0|2026-11-10T10:01:52.2123277Z–2026-11-10T10:02:17.6814041Z;25.467s wall|`ok levnote.local/native/app 24.204s`; stderr0bytes|
| `02-vet` | `& '.\native\scripts\go.ps1' -GoArguments @('vet','./app')` |0|2026-11-10T10:02:24.9932495Z–2026-11-10T10:02:26.3645992Z;1.370s wall|stdout and stderr0bytes|

Raw records are in
[`file-workflow-final-evidence-20261110`](file-workflow-final-evidence-20261110):

- `/01-test.command.txt`, `/01-test.exit.json`, `/01-test.stdout.txt`,
  `/01-test.stderr.txt`
- `/02-vet.command.txt`, `/02-vet.exit.json`, `/02-vet.stdout.txt`,
  `/02-vet.stderr.txt`

## Frozen identities

The union is nine files and160,306bytes. `before-manifest.json` was read and
inventoried before the commands; `after-manifest.json` was recomputed after
vet. Their parsed identities match exactly, with zero mismatches. The evidence
comparison is recorded in `before-after-comparison.json`.

The final canonical manifest is
[`source-frozen.json`](file-workflow-final-evidence-20261110/source-frozen.json),
1,348bytes, SHA256
`940C698E0B0195258649E6A0747A8F1EBAF1F754BAE05E6539B790B023098DAF`.
The pre-run JSON was written with CRLF, while the canonical final manifest uses
LF; their serialized bytes therefore differ, but every parsed identity field is
identical.

| File (native/) | Bytes | SHA256 |
|---|---:|---|
|`app/app.go`|38935|`0AC7FB907C83DC3C2B7642B197441492274CBB1CBC75E6688264D577D995ADB4`|
|`app/reopen_closed.go`|7895|`BFCA75BA0E2A878483133ECE43530742C424623CBE6B8809F1285C9E04C62E92`|
|`app/binary.go`|10170|`AE2F059F0E91C812A82FFE3F0317EAFE7E2F0B0CE9AEB3AA5056F713C555226D`|
|`app/binary_actions.go`|10173|`D9DB7FAA2CF674ED3428F67479D6377B4901ED7C7A14C404F0536F325A9609BB`|
|`app/view.go`|23002|`195AFA3D82E7612A0CDBEC6A9D0F0432777F492CAB4D94596B0C6F2329C83456`|
|`app/workspace.go`|6715|`B1492575D6C64673BF02E633C0D7049F47BF151391B4CA58F58F3779599AF259`|
|`app/state.go`|13647|`7F3F8E338E5619241FA3D0ACC72BA791778F1CE2A328272CBAED12E40B9400FD`|
|`app/reopen_closed_test.go`|28867|`3D160E6CCA9602F285D1F0A19A1DC450EE8109F5E3B81DEDDC33C6DF037B2C0B`|
|`app/live_tab_admission_test.go`|20902|`285244AD5A24FFB927F57CDA28A8AD2F5CDB7F09FE7A52960EDDBD74BAA8766D`|

## Inherited focused evidence

The focused evidence remains in
[`live-tab-admission-evidence-20261110`](live-tab-admission-evidence-20261110/README.md)
and was not repeated:

- Final admission group: exit0,5.540s wall, app2.420s.
- Existing open-many/reopen/native-image affected set: exit0,4.406s wall,
  app3.267s.
- The real Markdown conflict view retained its draft across two capacity
  refusals, then recovered it after closing one tab.
- Refusals preserve source, version, history, selection, prior recovery/session
  bytes, Flush/release state and binary ownership as applicable. Mixed restore
  accepts exactly100 records in order.
- Evidence entries004-007 remain explicitly preserved test-construction
  failures, not product regressions. The unretained initial release-order
  report is reported history because no original log exists; the retained
  post-Flush cases independently verify the final ordering.

## Independent-review boundary

After the shared implementation/source review, GLM performed the exact
read-only review of the same nine frozen identities and found no outstanding
source-grounded defect. That review performed no tests, build, launch, resource
measurement, package change or source edit. Its result SHA256 is
`B0F499211A7FC126B05A497F5ED9FCD8BE93AA8D2C8B5FF725EB5750B5ABB07F`;
root accepted it in
[`live-tab-admission-root-acceptance.md`](live-tab-admission-root-acceptance.md).

The route decision also remains the review boundary: Restore is synchronous on
the UI owner, does not pump callbacks while waiting, and the binary worker has
no App authority. No worker-to-App test hook, manufactured cross-goroutine race
or disposable binary metadata was added. Final coverage concentrates on the
reachable asynchronous open/Flush routes and the successful mixed100-record
restore.

## Limitations

This run validates the frozen Go app source and tests only. It did not build a
package, launch the app or host, run a worker request, capture physical input,
or measure resources. It makes no physical keyboard/IME claim, no100-tab RSS or
other resource/performance claim, and no binary-delivery or `.27` package
claim. Root source acceptance and delivery preparation remain separate.
