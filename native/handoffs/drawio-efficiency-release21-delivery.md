# `.21` drawio efficiency delivery

**Status: DELIVERED AND FROZEN 2026-11-10.**

Warning plus source/retention source was accepted and frozen. The prepared
harness ran each product stage once. Aggregate required one evidence-only
correction and retry; no package build, tester compile/worker request, host
launch, source suite/vet, benchmark, or capture was repeated.

## Final package and evidence

- Candidate:
  `F:\AIproject\LevNote\native\bin\levnote-native-worker-20261110.21`
- Candidate identity: 25files /26,859,809bytes /
  `3126E7898073CEBBC02EAB907EEF541C8A3D5538F0B77A45D58B6CE9DBE72ED7`.
- Host: `levnote-native.exe`,14,097,920bytes /
  `2C400D3730AEE64FC380CA0312B44BAAB1C0803AA70BA96525804BB4B436B329`.
- Release manifest:4,490bytes /
  `C2556B0E372466B5F67FEBF7C9FBACBA0D44050B00F9892C64837ED16CE9A824`;
  24 payload entries.
- Evidence:
  `F:\AIproject\LevNote\native\artifacts\acceptance\drawio-efficiency-release21-20261110`
- Complete frozen inventory:202files /91,722,328bytes /
  `55B150C3A423BC3D213E7CF7282F1D87F9247AD35768236144D9217E7F4CE79B`.
  It includes the preserved first-attempt evidence and the final aggregate
  and validator; the corrected invocation streams are in the separate4-file
  retry sibling.
- [Validator](../artifacts/acceptance/drawio-efficiency-release21-20261110/validator.json):7,382bytes,
  SHA-256
  `48DB08D815E5A4E0888E29D79BF797BA4EDD6C92576B71F1768B0F5B35ECBD69`,
  PASS16/16.
- [Aggregate](../artifacts/acceptance/drawio-efficiency-release21-20261110/aggregate.json):18,214bytes,
  SHA-256
  `E34D2FE39096978E88FB5DB2B868EEB9B96315EA141D73E879B0910F191D1FB8`,
  PASS16/16.

Six accepted source gates and two accepted benchmark fixture paths were
fingerprinted. The candidate contains the fixed four engine manifests, five
Windows amd64 GUI PE images, a 24-entry schema1 `.21` manifest payload set,
portable assets/notices, and the rendered portable README with three schema3
routes. The
complete `.20` package and139-file direct evidence remained unchanged.

## Managed installed worker

One tester compile and one actual installed draw.io request passed. The worker
ran from a complete byte-identical25-file candidate bundle copy; adding only the
tester produced26files.

- PID `14228`; duplicated handle `648`; creation UTC
  `2026-11-09T20:59:07.8135391Z`.
- Image:
  `F:\AIproject\LevNote\native\artifacts\acceptance\drawio-efficiency-release21-20261110\managed-smoke\config\engine-packages\levnote.drawio-svg\b740d5cb782e78fe7f348bcd31f8d5dd34005b36e4d46933110e0808ab6bd7c1\levnote-drawio-svg.exe`.
- Request/page: one new managed selection, one export, one admission, one verify,
  one start, one wait; page index `1`, ID `second`.
- Source `516` bytes; payload `598` bytes, SHA-256
  `3F990A4AAC3209BAFAD94F44026BF14F3CEC27EA44BDBD3169555947936472BF`.
- Raw uppercase `Warnings` was JSON `null`; normalized uppercase `Warnings` was
  retained `[]`; count `0`; SHA-256
  `4F53CDA18C2BAA0C0354BB5F9A3ECBE5ED12AB4D8E11BA873C2F11161202B945`.
- Primary handle closed; no fallback close; runner close called.

Raw result/record:
[managed result](../artifacts/acceptance/drawio-efficiency-release21-20261110/managed-smoke/result.json),
[record](../artifacts/acceptance/drawio-efficiency-release21-20261110/managed-smoke/record.json).

## Hidden host

One separately copied byte-identical `.21` host started hidden on
`route-gate.diagramnote` with isolated `C:` configuration and the reviewed179-byte
`diagram-notes.json`.

- PID `7660`; held handle `2348`; native window found and not visible.
- Exactly one owned `WM_CLOSE`; no timeout or forced cleanup; exit `0`; app not
  live afterward.
- Job TOTAL processes was1 after assignment and after exit; requested worker
  operations were`0`, and no owned process remained active after exit.
- Session observed one clean `diagram-notes` preview tab: UTF-8, BOM, no draft,
  no terminal newline.
- Fixture, installed plugin, candidate bundle, and copied host bundle remained
  unchanged.
- Run-owned config root:
  `C:\Users\canace\AppData\Local\Temp\levnote-native-release21-route-smoke-eaadfcad9fa841aababf79d64971f612`.

Raw records:
[host result](../artifacts/acceptance/drawio-efficiency-release21-20261110/host-smoke/result.json),
[session](../artifacts/acceptance/drawio-efficiency-release21-20261110/host-smoke/session-after-close.json).

## Aggregate retry

The first aggregate invocation consumed the single stage-wrapper attempt and
returned exit `1` without replaying any product stage. It was15/16 because the
harness read non-existent `SourcePreserved` instead of the recorded
`FixturePreserved`; it then tried to hash its own redirected
`outer-aggregate.stderr.log`.

The first `validator.json`, aggregate script, copied helpers, invocation records,
and logs are retained under
`../artifacts/acceptance/drawio-efficiency-release21-20261110/aggregate-first-attempt/`;
the first failed
validator remains there with SHA-256
`7E89B7EF1BDC27AC214FC9006657CC2F446518BDE1729B2D5B4C9F9867BD7E96`.

The copied evidence aggregate was corrected only to use `FixturePreserved` and
exclude the actively-open aggregate invocation streams/records. It was directly
captured outside the scanned evidence root:

- Retry record/command:
  `../artifacts/acceptance/drawio-efficiency-release21-20261110-aggregate-retry/aggregate-retry-invocation.json`
- Raw stdout: same directory,
  `aggregate-retry.stdout.log` (562bytes)
- Raw stderr: same directory,
  `aggregate-retry.stderr.log` (0bytes)
- Exit: same directory, `aggregate-retry.exit.txt` (`0`).
- Retry-directory complete inventory: 4 files / 1,635 bytes /
  `FA8CDD03BA1AC32257A97403BE861DC82E9EE3EF2443866D7860B6BC1EF6AB16`.

## Limits

No physical UI/IME, interactive SVG, dirty-save/recovery, whole-process or
interactive memory/CPU, installer, cross-platform, or resource-matrix claim is
made. This delivery inherits accepted source test/benchmark evidence; no source
suite, vet, benchmark, or capture was rerun here.
