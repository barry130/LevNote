# .27 file-workflow delivery preparation frozen

Prepared2026-11-10 by Astra. Preparation only; app/runtime/test ownership remains
with the admission owner. No tests/vet/builds/package/host/worker/resource run,
no new screenshot, no source acceptance manifest and no delivery context were
created. The .27 candidate directory is still absent. .26 files were read only.

Target: native/bin/levnote-native-worker-20261110.27,
version native-worker.20261110.27. Helpers are complete in
native/artifacts/acceptance/file-workflow-release27-20261110.

The normal successful .25 package/aggregate/preflight/stage wrappers and external
close helper were copied; final .26 library/context/baseline/source scope and
strict held-process host were copied. The normal package gate requires PASS,
one package invocation and five GUI builds; .26 repair-only schema is removed.
The copied originals were rehashed and match the saved copy origins. Exact
source and destination identities are in copied-helper-provenance.json:
10,267B SHA256
6F8B8B9BBF360CC521410FED1CBDDD09515E8A3A256F0766AA25B97083C65BFF.
The original copy record is copied-helper-origins.json,5,467B SHA256
201EA05A1C97BBE4F2373EEB8ECB4ABD5281F70BE3DAE1A2D66EA9EFAA6A49BD.

Final input binding intentionally remains deferred. Initialization requires
native/handoffs/file-workflow-source-root-acceptance.md and reads the accepted
native/handoffs/file-workflow-final-evidence-20261110/source-frozen.json array
with Path(native-relative),Length,Sha256. It validates every listed source,
requires the reopen files and live_tab_admission_test.go, and freezes the normal
full scoped source/build/portable/vendor inputs then. No pre-cap app hash is
treated as final. Prior component acceptances remain reference gates.

Complete accepted .26 package/direct evidence/both aggregate capture siblings/
external closed inventory use markdown-image-reference-release26-root-acceptance.md
as authority, including corrected package inventory identity7DB6E266...5EE1FB64.
All four protected directories and the external inventory are checked again
by future baseline/aggregate/closure. No old package or evidence was altered.

Portable README adds recent16 saved-local path history, current disk content,
no discarded edits or untitled-draft restoration, explicit retry, transient
history, utility composition commit/cancel, and100live tabs with existing-path
reuse/close-and-retry. Existing capabilities and limitations remain. No RSS
claim. README is17,455B SHA256
392902C420712FB18ECDE3AFD8BAC9CD3CDC768DEC513699E23D396DD71F9F29.
Read-only prose checks passed76required/10forbidden phrases, one version marker,
the existing three schema3 routes and exactly five intended local links. The
Markdown image example remains split across inline-code spans, avoiding .26's
link-check false positive.

One authorized completed-helper static parse ran through run-preflight.ps1:
PowerShell7.6.9,14scripts(12copied helpers + generated parse command + external
close helper),zero errors,exit0;2026-11-10T09:42:05.5256642Z to
09:42:06.2716677Z,0.7430374s. Raw parse.stdout.log4,373B and
parse.stderr.log0B are preserved. static-parse-result.json4,372B SHA256
7AE12305C8D09DC75E2D819725B62B9D624F2B873EDD999A0DEFC27D063A1636;
parse-invocation.json992B SHA256
9059B071D3BFCBCB01D4ABFB8DE6AE8E47975717E70E671660AA73DAE19AF636.
No helper changed after this parse.

| Prepared helper | Bytes | SHA256 |
|---|---:|---|
| aggregate.ps1 | 33022 | 2897FC53BFEB0B289F96DF7995D4CAC0CD49F85F882CF533E708D87B260D4C77 |
| run-package.ps1 | 17614 | 9E753B99DBF8350FC37D275A6C101431DE926BB040269D07E68AAD630B936FBE |
| initialize-delivery.ps1 | 2040 | 4F0858D09CF9F0CD62AE71F6642BCD0A4576F771B70B6E6183BE5BA809A08FC5 |
| run-preflight.ps1 | 1299 | 78839F34149B517F2AA3F0DA8D850F6A5193446C6D76D9877F9A7E7C1759252A |
| static-parse.ps1 | 1783 | 6DB6B42C9D8C5AC481EFBA567E0F67CEF197973460DD1F3C38435F54E3D7D26A |
| invoke-stage.ps1 | 2653 | 97369BF3581B5A31706C87DEE3F584D77FA938D4D47B46D1C98E92CB5B151DA7 |
| run-stage.ps1 | 2958 | 9631549F240DD5E30A3B7DC8F39F1C4566903A1FF59CE5FABBC7A32B0F31F59A |
| delivery-lib.ps1 | 29437 | 028AF309F517008D26EADED36DE94E1E5D71284573C50877B05BB8CC98D2ABFB |
| initialize-context.ps1 | 11766 | 83BCADA339216CA09DA39C1A96F2E495D12347FF4D62366E4E75EDFA1C394D48 |
| baseline.ps1 | 9000 | 2703633C7E751CB00B5B7E964C32A5B8E56653372357EB6F230F293B0434C912 |
| source-scope.ps1 | 2157 | 6CEFB378F9027F66892560CE92D3199C1F5F8C5D12FD32E3F40E557A78DE5FC5 |
| run-host-smoke.ps1 | 33147 | BB4CA8E4AE3DBE5A6CE06A9BE95BC5F5AFD70061D2376C8D53626A9708F884B7 |
| handoffs/file-workflow-release27-close-inventory.ps1 | 3831 | 0F8EA204474C2DF3D4CE65CFF2536E817064C5ACADAE8325C49AC27E28FBF2E1 |

After cap review, the one combined final app test/vet, root source acceptance
and explicit delivery activation, execute this sequence from the repository.
Stop at a nonzero exit; do not repeat parse or any passed product operation.

```powershell
$pwsh = (Get-Command pwsh -ErrorAction Stop).Source
$evidence = 'F:\AIproject\LevNote\native\artifacts\acceptance\file-workflow-release27-20261110'
& $pwsh -NoProfile -File (Join-Path $evidence 'run-preflight.ps1') -Step initialize
if ($LASTEXITCODE -ne 0) { throw 'Initialize failed; preserve evidence.' }
& $pwsh -NoProfile -File (Join-Path $evidence 'run-stage.ps1') -Name baseline
if ($LASTEXITCODE -ne 0) { throw 'Baseline failed; preserve evidence.' }
& $pwsh -NoProfile -File (Join-Path $evidence 'run-stage.ps1') -Name package
if ($LASTEXITCODE -ne 0) { throw 'Package failed; no repeated builds.' }
& $pwsh -NoProfile -File (Join-Path $evidence 'run-stage.ps1') -Name host-smoke
if ($LASTEXITCODE -ne 0) { throw 'Host failed; no repeated launch.' }
& $pwsh -NoProfile -File (Join-Path $evidence 'run-stage.ps1') -Name aggregate -AggregateAttempt 1
if ($LASTEXITCODE -ne 0) { throw 'Aggregate failed; preserve attempt1.' }
& $pwsh -NoProfile -File 'F:\AIproject\LevNote\native\handoffs\file-workflow-release27-close-inventory.ps1'
if ($LASTEXITCODE -ne 0) { throw 'Closure failed; preserve original evidence.' }
```

Future budget is one package/five GUI builds and one exact copied hidden host.
Ordered accepted UTF8+BOM/CRLF custom-route text then PNG, isolated C: config/
plugin,5000ms alive settling, strict fresh pre-close snapshot, one owned WM_CLOSE,
newer normal-close snapshot,exit0/Job cleanup. Held process/window identity,
typed three-time comparisons and saved-array closure totals are inherited.
No second restore launch or worker request. Aggregate writer logs use separate
capture siblings; close only after those writers exit, keeping any closure
stdout/stderr outside all scanned roots. Existing metadata-only attempt2 may
be used only under its failure-preservation conditions, without product reruns.

This lifecycle does not establish physical CtrlShiftT/IME interaction, image
rendering,100-tab memory/RSS, dirty recovery or resource improvement. Reopen/
admission behavior must come from the separately accepted source regressions.
Preparation files and portable README are now frozen pending root review;
product execution remains held.
