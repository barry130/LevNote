# .28 document-growth delivery preparation frozen

Prepared2026-11-10 by
`read_native_handoffs_growth_release28_preparation_task_md`. This is preparation
only. No candidate directory, delivery context, source acceptance or final
source manifest was created. No tests/vet/build/package/host/worker/resource/
screenshot action ran. No child was delegated.

Target: `native/bin/levnote-native-worker-20261110.28`,
version `native-worker.20261110.28`. Helpers are complete in
`native/artifacts/acceptance/growth-release28-20261110`; the external close
helper is `native/handoffs/growth-release28-close-inventory.ps1`.

## Copied provenance and adaptations

The 12 evidence helpers and external close helper were copied from the accepted
immutable `.27` preparation and adapted in place. Exact source and destination
paths, bytes, SHA256 values and timestamps are in
`copied-helper-provenance.json`,10,782B/SHA256
`AB3B06CC7AE395577D25BAFC1F1EEAE6424562CAF8913C5148885B5E94B7D620`.

Necessary changes were limited to:

- `.28` version, candidate directory, evidence/closure paths and scope labels.
- Adding the accepted `.27` root record and the pending
  `document-growth-budget-source-root-acceptance.md` to the existing
  historical source/component gates.
- Replacing the stale reopen/live-tab minimum-file-set requirement with the
  root-accepted final manifest
  `native/handoffs/document-growth-budget-final-evidence-20261110/source-frozen.json`.
  It validates entries with `Path` (native-relative), `Length` and `Sha256`;
  no intermediate source identity is pinned.
- Protecting the complete accepted `.27` package, direct evidence, separate
  aggregate capture and external closed inventory by their exact `.27`
  identities.
- Extending the existing portable README contract with the final growth-policy
  facts. No new helper framework or duplicate guard machinery was added.

| Prepared helper | Bytes | SHA256 |
|---|---:|---|
| aggregate.ps1 | 33015 | 95B92A7D3CA3ECEFA9A8A2AB3EA44152521B299087CB401908C9316E3AC1F643 |
| run-package.ps1 | 17616 | CC27F265DC9E41BBD92260AF8485FB51099B2D8A926A44D173D724A19DFBDD2A |
| initialize-delivery.ps1 | 2226 | 8140FED7BDB143CDBC783D42BBB94B472FB780DAA80DD51D2D4BF33D0D50B324 |
| run-preflight.ps1 | 1299 | 78839F34149B517F2AA3F0DA8D850F6A5193446C6D76D9877F9A7E7C1759252A |
| static-parse.ps1 | 1776 | 9099E2DCC4B95AED96B9EB182C33C0CA58855CB7CA1D4B56CF4D4C9C923FD545 |
| invoke-stage.ps1 | 2653 | 97369BF3581B5A31706C87DEE3F584D77FA938D4D47B46D1C98E92CB5B151DA7 |
| run-stage.ps1 | 2958 | 9631549F240DD5E30A3B7DC8F39F1C4566903A1FF59CE5FABBC7A32B0F31F59A |
| delivery-lib.ps1 | 29586 | 02F02E07D5114F7091DD78716DB51E0C998F6322AA74519F29D4B2F13EA788F3 |
| initialize-context.ps1 | 11849 | CCC856EC3DD5D3D40928B355991EEEAAA227BAE712B5A201E63FA2F72596A67B |
| baseline.ps1 | 8979 | 4F925C41FD98E83F7CEE496F921ECD8F60C26AA1F8A972B1F52892C290F1C8C4 |
| source-scope.ps1 | 2157 | 6CEFB378F9027F66892560CE92D3199C1F5F8C5D12FD32E3F40E557A78DE5FC5 |
| run-host-smoke.ps1 | 33138 | 8F0C11E4018C710F2DED49C731192E07D01608B50DBB9A976B911ED0B4373660 |
| growth-release28-close-inventory.ps1 | 3823 | F4BE512AC5D15FBADC5524B8C912958B8433C3DF8E6BAFD1173EBB1C152B0335 |

The current source acceptance and final manifest intentionally do not exist yet.
Initialization must remain blocked until root publishes both after the affected
final test/vet. The manifest is the sole source-identity authority; earlier
component acceptances remain history and gates, not stale file-set pins.

## Protected .27 baseline

`file-workflow-release27-root-acceptance.md` is the authority for these exact
protected objects:

| Object | Files | Bytes | Inventory identity / SHA256 |
|---|---:|---:|---|
| package `native/bin/levnote-native-worker-20261110.27` | 26 | 27,382,619 | `04740A5B7ABECA69FE5CBDE224B85C26EBC93C9BE418E95B0DC9B0D95AB5B247` |
| direct evidence `native/artifacts/acceptance/file-workflow-release27-20261110` | 147 | 55,816,795 | `EE8F9D01AC05AAD418C7858E143E8AAF56D0B0B53C3B4B1BC2363BA085C75342` |
| capture `native/artifacts/acceptance/file-workflow-release27-20261110-aggregate-capture-attempt-1` | 6 | 3,264 | `B0AC3622B591A12BB7F1A7EA42D3575EFF18D9FF627F18A912A0ACFD19255F87` |
| external `native/artifacts/acceptance/file-workflow-release27-20261110-closed-inventory.json` | 1 | 65,251 | `E4F9FEE38A2F557E954A3E9F2A453B6A55789E2B78E4230668C15328582BD978` |

The protected `.27` host is14,556,160B/SHA256
`3B6734F30033569337A111353BD1BF661FB69F1782983688B0D3D863D0E8212C`;
the release manifest is4,674B/SHA256
`C7783296AEEF59FD6254AF3F9487427658C40227BC5DAC15DF2348ED869926CF`
with version `native-worker.20261110.27`.

The frozen bitmap/vendor/license guards remain unchanged: 35 frozen bitmap
files, the existing pinned dependency-patch manifest and the existing portable
asset/link/license checks are all retained in the copied flow.

## Portable README

`native/portable/README.md` now states the actual implemented policy:

- Ordinary UTF-8 live-content growth is bounded at128MiB, separately from the
  128MiB disk-read bound.
- Initial or explicitly reloaded decoded seeds are retained in full; the
  effective edit ceiling is `max(base, seed bytes)` and does not shrink after
  editing, so accepted undo remains usable.
- Refused replacement/ReplaceAll/indent work is visible and does not commit a
  partial result. A reverse-order intermediate overflow can conservatively
  reject a final-size-fitting batch.
- A rejected rich-format flush keeps the local draft.
- Clipboard, regex expansion, codec/decode strings already in process and
  process memory are outside this content-growth policy.

The prepared README is18,198B/SHA256
`4340A1E0D888BECEC80A765FE63EBA349B2E8A1B631813EDB47ED8152C99245B`.
The one authorized narrow consistency check found88/88 required phrases and
none of10 forbidden phrases, one version marker and the existing three schema3
routes (`builtin.markdown`, `builtin.mindmap`, `builtin.drawio`).

## Static parse

The one authorized completed-helper static parse ran through
`run-preflight.ps1 -Step parse` on PowerShell7.6.9:
14scripts,0parse errors,exit0,stdout4,275B,stderr0B. It covered the12copied
evidence helpers plus generated `parse-command.ps1` and the external close
helper. Raw streams and invocation are preserved in the evidence directory.

`static-parse-result.json` is4,274B/SHA256
`1C8F1B8B55636926A900CC5005AFAED75F37577190591626B294F091ED2F9043`;
`parse-invocation.json` is964B/SHA256
`C6BC61D22999803B1789C808782E6563513CE7B5BE9B6196A567708D960E0C9E`.
No helper changed after this parse.

## Deferred execution

After root publishes both growth source artifacts, accepts this preparation and
explicitly activates one delivery, run this sequence from the repository and
stop on every nonzero exit:

```powershell
$pwsh = (Get-Command pwsh -ErrorAction Stop).Source
$evidence = 'F:\AIproject\LevNote\native\artifacts\acceptance\growth-release28-20261110'
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
& $pwsh -NoProfile -File 'F:\AIproject\LevNote\native\handoffs\growth-release28-close-inventory.ps1'
if ($LASTEXITCODE -ne 0) { throw 'Closure failed; preserve original evidence.' }
```

Execution remains budgeted to one package call/five GUI builds and one exact
copied hidden host. The host keeps the proven isolated `C:` config/plugin,
ordered UTF8+BOM/CRLF custom-route text then PNG,5000ms alive settling, strict
fresh pre-close session, one owned `WM_CLOSE`, newer normal-close publication,
exit0 and exact Job/handle cleanup. No worker request, repeated tests/input/
capture/resource measurement or second restore is authorized.

This startup/session coverage does not test physical size refusal or IME and
does not establish whole-app memory improvement. The content-growth behavior
comes from Astra's focused affected tests and root's final source acceptance.
Preparation ownership is released after this handoff.
