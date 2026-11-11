# .19 harness review correction result

Status: **CORRECTED HARNESS PREPARED AND FROZEN; NO DELIVERY EXECUTED**

Date: 2026-11-10

The corrected harness is in the fresh sibling directory
`native/handoffs/drawio-worker-release19-preparation-corrected/`.
The original `native/handoffs/drawio-worker-release19-preparation/` directory,
its result handoff, the accepted `.18` package/evidence, product files, and
accepted gates were not edited.

No `.19` package, build, test executable, test, worker, host, screenshot, or
capture was executed or created. The `.19` candidate remains absent:

`native/bin/levnote-native-worker-20261110.19`

The only executable check was the permitted PowerShell parser-only pass.

## Corrected runtime variables

- `run-package.ps1`: renamed local `$matches` to `$engineMatches`, and
  `$host` to `$packageHost`; the result property remains `Host`.
- `aggregate.ps1`: renamed local `$host` to `$hostResult` and updated every
  read of the hidden-host result.
- A case-insensitive static scan found no remaining listed automatic-variable
  collisions for `Host`, `host`, `matches`, `input`, `args`, `error`, `pwd`,
  `home`, `profile`, or `this`.

| Changed file | Old SHA256 | Corrected bytes | Corrected SHA256 |
|---|---|---:|---|
| `aggregate.ps1` | `690E11BE57446BDDD9B4B0F6DD7C293F22C0A43B7D2D0D4C9230782993F69420` | 21,292 | `4911A4E9A82B936EC3BA90E660D522E323C191F8A73AE6DA79DCA31E2F1E6131` |
| `initialize-context.ps1` | `38DB3BE12516BA10AD42A29C80EBA5FF18B55D940516766AE729FF8239F63AFF` | 10,828 | `4C3A26FC67D8E48CFCC6F6DE23C0C0A69B0AFA279DB1AF77CF128CF185F23FC2` |
| `README.md` | `D0AFE47E7E133A73398527848DB62A82256AC621C812B569F21F4E659B87B0FB` | 1,929 | `523556FD8A415E5BA91FD693559471B6B72B8219BAEE426C7E7C3C1BA42EF3A7` |
| `run-package.ps1` | `088B4F03F509A51E07BCD774424E558A46B5215F01E3015A2FB27CAC3C6E3DFB` | 10,950 | `E5F2016AD32972E770AE85A3CCFEA0EC33029C9E4CB4384EDD451379E8DE33E5` |

The other seven copied PowerShell scripts remain byte-identical to the original
frozen harness.

## Stage 2 protection correction

`initialize-context.ps1` now always inventories both frozen Stage 2 evidence
directories before adding the activation-supplied final evidence directory:

- `native/handoffs/drawio-worker-stage2-evidence`
- `native/handoffs/drawio-worker-stage2-test-followup-evidence`

Paths are normalized absolutely and inserted through a
`StringComparer.OrdinalIgnoreCase` set. A supplied final evidence path equal to
either fixed directory is skipped, including path-case differences, so each
directory is inventoried once. A distinct final evidence path is retained.

The initializer retains all 15 previous fixed gate files and adds the now frozen:

- `native/handoffs/drawio-worker-stage2-test-followup-result.md`
- `native/handoffs/drawio-worker-stage2-independent-review.md`

The fixed gate set therefore contains 17 files. Activation must still supply at
least one final Stage 2 gate path through `-FinalStage2GatePaths`.

## Static parser evidence

Exactly ten corrected `.ps1` files were parsed. Every file reported zero parse
errors; there were no parser diagnostics:

- `evidence/static-parse.stdout.log`: 438 bytes, SHA256
  `4BA0A382EB8944119FF692FAAAF744FABA020793AD7A48C854836C6629D6C4EC`
- `evidence/static-parse.stderr.log`: empty, SHA256
  `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855`
- `evidence/static-parse.exit.txt`: `0`, SHA256
  `9A271F2A916B0B6EE6CECB2426F0B3206EF074578BE55D9BC94F6F3FE3AB86AA`

The corrected `harness-manifest.sha256` covers 14 files: ten scripts, the
README, and three parser-evidence files. It has 14 entries, covers 93,089 bytes,
and all entries match:

- 1,203 bytes
- SHA256
  `3F50584200F28D62B9EBC6CFD5589309C83D2EF575321DDF433F80E08611F8CB`

The original frozen harness manifest still has zero mismatches, and the original
preparation result remains unchanged.

## Exact activation command shape

Delivery remains deferred until root supplies the accepted final Stage 2
manifest/evidence/gates and activates
`drawio-worker-release19-delivery-task.md`. The activation sequence is:

```powershell
$corrected = 'F:\AIproject\LevNote\native\handoffs\drawio-worker-release19-preparation-corrected'
$candidate = 'F:\AIproject\LevNote\native\bin\levnote-native-worker-20261110.19'
$evidence = '<fresh absolute .19 evidence root>'

& "$corrected\initialize-context.ps1" `
  -CandidateDirectory $candidate `
  -EvidenceDirectory $evidence `
  -FinalStage2FilesManifest '<frozen Stage 2 file manifest>' `
  -FinalStage2EvidenceDirectory '<final Stage 2 evidence directory>' `
  -FinalStage2GatePaths @('<final Stage 2 gate file>') `
  -Version 'native-worker.20261110.19' `
  -ExecuteDelivery

$runStage = Join-Path $evidence 'run-stage.ps1'
& $runStage -Name baseline
& $runStage -Name package
& $runStage -Name managed-smoke
& $runStage -Name host-smoke
& $runStage -Name aggregate
```

Initialization copies the corrected scripts and README into the fresh evidence
root. The five stage commands therefore use the evidence-local `run-stage.ps1`
and `context.json`, not the preparation copies.

## Preserved scope

The correction does not authorize or change package contents, package workflow,
managed-smoke assertions, host-smoke lifecycle, aggregate checks, product code,
or build inputs. Those remain the previously reviewed prepared behavior.

The corrected harness is frozen after publication. No further tests, parser
runs, builds, launches, resource measurements, or delivery entrypoints are
authorized by this result.
