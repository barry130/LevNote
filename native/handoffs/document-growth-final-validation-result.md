# Growth final affected validation — PASS

2026-11-10 Astra. Executed exactly one nine-package test invocation followed by
one matching vet invocation under the active final-validation task. Both actual
Go/wrapper and child-shell exits are0. No source edits, fixes or repeat runs.
All11frozen source identities matched before and after.

Evidence root: [document-growth-budget-final-evidence-20261110](document-growth-budget-final-evidence-20261110/).

## Commands and outcomes

From native/ through the repository wrapper:

```powershell
native/scripts/go.ps1 -GoArguments @('test','./core/text','./core/document','./core/search','./core/session','./ui/editor','./modules/sourceview','./modules/markdownview','./modules/graphview','./app','-count=1')
native/scripts/go.ps1 -GoArguments @('vet','./core/text','./core/document','./core/search','./core/session','./ui/editor','./modules/sourceview','./modules/markdownview','./modules/graphview','./app')
```

The saved scripts use `./scripts/go.ps1` after setting the native working
directory. Each captures `$LASTEXITCODE` immediately after the wrapper, writes
the actual Go result, then exits with that captured code. The wrapper directly
exits with its captured native go.exe result; its inspected SHA256 was
`A3EFCD7DC4079641F1767341446168E6778AB07E1C4DD09A968C768D1CC1B178`.
Separate child-process records additionally confirm shell exits0. Log writers
finished before final hashing.

| Command | UTC start → finish inside command script | Go / shell exit | Result |
| --- | --- | --- | --- |
| test | 2026-11-10T11:06:35.6012270Z → 11:07:07.5550564Z | 0 / 0 | 9/9 packages pass |
| vet | 2026-11-10T11:07:24.7043468Z → 11:07:25.9977153Z | 0 / 0 | No diagnostics |

Package test times: core/text1.510s, core/document2.261s, core/search1.707s,
core/session2.683s, ui/editor4.216s, sourceview2.082s, markdownview7.515s,
graphview16.658s, app27.455s. Test stderr and both vet streams are empty.

- [Test script](document-growth-budget-final-evidence-20261110/01-test.ps1),
  [actual Go result](document-growth-budget-final-evidence-20261110/01-test.go-result.json),
  [process result](document-growth-budget-final-evidence-20261110/01-test.process-result.json),
  [stdout](document-growth-budget-final-evidence-20261110/01-test.stdout.log),
  [stderr](document-growth-budget-final-evidence-20261110/01-test.stderr.log).
- [Vet script](document-growth-budget-final-evidence-20261110/02-vet.ps1),
  [actual Go result](document-growth-budget-final-evidence-20261110/02-vet.go-result.json),
  [process result](document-growth-budget-final-evidence-20261110/02-vet.process-result.json),
  [stdout](document-growth-budget-final-evidence-20261110/02-vet.stdout.log),
  [stderr](document-growth-budget-final-evidence-20261110/02-vet.stderr.log).

## Frozen inputs and scope

[Preflight](document-growth-budget-final-evidence-20261110/preflight.json)
matched implementation's exact11file union. [Before](document-growth-budget-final-evidence-20261110/source-before.json),
[after](document-growth-budget-final-evidence-20261110/source-after.json) and
[comparison](document-growth-budget-final-evidence-20261110/source-comparison.json)
prove11/11unchanged Length/Sha256. No additional correction files exist.

[Final source-frozen.json](document-growth-budget-final-evidence-20261110/source-frozen.json)
uses native-relative Path,Length,Sha256 and is byte-identical to the implementation
manifest: SHA256 `B0A212FF61660DDADD9C5A040289FA6057E52B36E3A030B91BA32196A7C8D275`.
[Closed evidence-file identities](document-growth-budget-final-evidence-20261110/evidence-files.json)
SHA256 `ECFFD261571C83A5420530E28982B13B48F8E3DE5B5166BC256B74A16BEA37D6`.

No LEVNOTE opt-in environment variable was set. No opt-in actual worker,
resource, screenshot or benchmark execution was enabled; no product package or
host launched. Normal package tests exercise their existing headless/test seams.
This validates affected source behavior, not physical Windows IME, process
memory measurements or a delivered .28 executable. Earlier focused failure and
correction evidence remains intact in the implementation evidence directory.

Validation is frozen and owner stops here. Root acceptance and separate .28
delivery activation remain the next gates.
