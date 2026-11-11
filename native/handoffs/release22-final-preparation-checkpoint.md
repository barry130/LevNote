# .22 final preparation checkpoint

2026-11-10, Astra. Helpers frozen for root execution activation. No app/source
edit, app test/vet, tester, package/build, host launch or worker request occurred.
Candidate `.22`, final execution evidence directory and preparation context.json
are still absent. Product docs/README remain unchanged.

Continued the existing ten helpers under
`tab-workflow-release22-preparation-20261110/`; no replacement harness.

Corrections completed:

- run-stage/invoke-stage derive the aggregate capture sibling from the actual
  evidence leaf: `<evidence-directory>-aggregate-capture`. Both refuse existing
  stdout/stderr/invocation files; package capture also refuses existing logs.
  Aggregate records that capture directory; its active logs are outside the
  evidence inventory. After activation this becomes
  `native/artifacts/acceptance/tab-workflow-release22-20261110-aggregate-capture`.
- Test-FingerprintEqual again compares Path, Bytes, SHA256 and optional timestamp.
  Installed plugin copy admission still uses Test-FileContentEqual. Same-path
  fixture/plugin and bundle-before/after checks preserve timestamps; comparison
  between distinct copy paths continues to use relative names/content.
- Full read found two stale Release21 generic types inside host Add-Type C#;
  Marshal.SizeOf now references the declared Release22JobExtendedLimit and
  Release22JobAccounting. PowerShell parsing alone cannot catch that defect.
- Static parser now requires a fresh absolute ResultPath, preventing overwrite
  of prior static-parse.json. Accepted tab gate diagnostic names were corrected.

Reviewed all helpers end to end against actual producers: initialize-context
object fields, baseline/package/host/aggregate consumers, package-markdown-worker
and portable-package-lib result/manifest/source-copy/build records, app config
path and core/session JSON names. ReviewedDiagramNotes and
PortableSourceReadmeFingerprint consistently remain objects. Referenced custom
helpers are defined/dot-sourced. Source inventory keeps Path fallback; protected
.21 directory inventory serialization is unchanged. This was source review,
not execution of delivery stages or embedded C# compilation.

Exactly one final parser invocation after these corrections: PowerShell7.6.9,
10scripts, zero parse errors, exit0. New preserved files in the helper directory:

- `static-parse-astra-final.json`:3162B, SHA256
  F33541C4962A37AD833F905576BB0D3C2BE021E991555B7A295F6C3CE19D4D6C.
- `static-parse-astra-final.stdout.log`:3163B; stderr log:0B.
- `static-parse-astra-final.exit.txt`:1B, hex30 (ASCII `0`).
- `static-parse-astra-final-invocation.json`:1372B, actual command/arguments,
  paths, times, exit and stream lengths.
- `static-parse-astra-final-prior-records.json`:3799B, identities of13existing
  parse records; all13 remained byte/time identical through this invocation.

Evidence gap inherited from preparation: the earliest failed parse had already
been overwritten before this assignment. The file named first-attempt.json is
a passing parse, not that failure. Prior final.exit.txt and static-parse.exit.txt
are both zero-length, not ASCII `0`. All prior files/prose remain preserved;
this checkpoint and the new final parse supersede their stale claims/hashes.

Final helper identities are recorded in the new parse JSON. Changed helpers:

| File | Bytes | SHA256 |
|---|---:|---|
| aggregate.ps1 |19599|CDB7573773CCE36E94E486BE59AD3C5E5F13F7DB4B488E0161A1FE503EB54D78|
| delivery-lib.ps1 |8754|C4483486A780C987EF40198D3CEFACB63BA1DD782229BA02895B1D007791EE68|
| invoke-stage.ps1 |2180|495ED87B22748C775185914B0E3B355763E3E1BD87611139750C7C4E39C986D1|
| run-host-smoke.ps1 |19842|E6F465B22C51F3E396ED0D28B9C21C2146EF313F8632F34344CF3453CB72E7C2|
| run-stage.ps1 |2479|914D481402F0036F1A2CDB03504E49F46AE75C552200441BAFC21CD5F69E02BB|
| static-parse.ps1 |1666|FC9EA3E3AB5A6A36A80880C9C62E85D495B7E892E3ADE4758B3EEF0AAC2D9E68|

Next, only after root activates: copy complete helpers to fresh
`native/artifacts/acceptance/tab-workflow-release22-20261110`, initialize the
real final paths and accepted gate files, then baseline -> package -> host ->
aggregate. One package invocation/five GUI builds and one exact hidden host;
metadata failures preserve evidence and never repeat passed product steps.
No physical input/IME or whole-app resource claim. Stopped before activation.
