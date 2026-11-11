# Final affected composition validation — PASS

2026-11-10 Astra. Exactly one full affected9package test followed by one matching
vet completed. Actual native Go/wrapper and shell exits are0 for both. All4
reviewed source identities match before/after; no source edits or corrective
runs. Source and validation evidence are now frozen for root acceptance.

Evidence: [active-composition-coalescing-final-evidence-20261110](active-composition-coalescing-final-evidence-20261110/).

## Actual commands

Saved scripts set native/ as working directory and invoke:

```powershell
./scripts/go.ps1 -GoArguments @('test','./core/text','./core/document','./core/search','./core/session','./ui/editor','./modules/sourceview','./modules/markdownview','./modules/graphview','./app','-count=1')
./scripts/go.ps1 -GoArguments @('vet','./core/text','./core/document','./core/search','./core/session','./ui/editor','./modules/sourceview','./modules/markdownview','./modules/graphview','./app')
```

Each saves `$LASTEXITCODE` immediately after the wrapper and exits with that
value after writing Go-result metadata. The wrapper's native go.exe exit
propagation was inspected; preflight SHA256 is
`A3EFCD7DC4079641F1767341446168E6778AB07E1C4DD09A968C768D1CC1B178`.
Separate process-result records confirm the same shell exit. Log writers ended
before hashing the closed evidence files.

| Command | UTC start → finish within script | Go/shell exit | Outcome |
| --- | --- | --- | --- |
| test | 2026-11-10T11:32:49.5106114Z → 11:33:33.3885214Z | 0/0 | 9/9 packages pass |
| vet | 2026-11-10T11:33:58.2801608Z → 11:34:01.0655179Z | 0/0 | No diagnostics |

Test times: text0.966s, document1.353s, search1.323s, session2.636s,
editor2.920s, sourceview2.406s, markdownview5.889s, graphview16.495s,
app41.451s. Test stderr and both vet streams are empty.

- [Test script](active-composition-coalescing-final-evidence-20261110/01-test.ps1),
  [Go result](active-composition-coalescing-final-evidence-20261110/01-test.go-result.json),
  [process result](active-composition-coalescing-final-evidence-20261110/01-test.process-result.json),
  [stdout](active-composition-coalescing-final-evidence-20261110/01-test.stdout.log),
  [stderr](active-composition-coalescing-final-evidence-20261110/01-test.stderr.log).
- [Vet script](active-composition-coalescing-final-evidence-20261110/02-vet.ps1),
  [Go result](active-composition-coalescing-final-evidence-20261110/02-vet.go-result.json),
  [process result](active-composition-coalescing-final-evidence-20261110/02-vet.process-result.json),
  [stdout](active-composition-coalescing-final-evidence-20261110/02-vet.stdout.log),
  [stderr](active-composition-coalescing-final-evidence-20261110/02-vet.stderr.log).

## Input freeze and limits

[Preflight](active-composition-coalescing-final-evidence-20261110/preflight.json)
matched the exact four-file independent-review union.
[Before](active-composition-coalescing-final-evidence-20261110/source-before.json),
[after](active-composition-coalescing-final-evidence-20261110/source-after.json) and
[comparison](active-composition-coalescing-final-evidence-20261110/source-comparison.json)
record4/4unchanged Length/Sha256. No correction files were added.

[Final source-frozen.json](active-composition-coalescing-final-evidence-20261110/source-frozen.json)
uses native-relative Path,Length,Sha256 and matches the implementation manifest
byte-for-byte: `5673AB5368E21B7F81631418448C0D4B20B8ECC3A274906E6BE3EBF77EB47213`.
[Closed evidence identities](active-composition-coalescing-final-evidence-20261110/evidence-files.json)
SHA256 `93C8106899363E3CFCBDF2722DD10C58C7BF9E073E87D62B51C80B282A27ED43`.

No LEVNOTE opt-in environment variables were set; no opt-in worker, resource,
screenshot or benchmark tests enabled. No product build or host run. These are
source/integration checks, not physical Windows IME or process-memory evidence,
nor a delivered .29 executable. Earlier focused evidence remains untouched.

Picker planning was checkpointed before this prioritized run at
[open-document-picker-next-plan-checkpoint.md](open-document-picker-next-plan-checkpoint.md).
No picker implementation has begun. This validation owner stops here pending
root acceptance; next delivery needs separate activation.
