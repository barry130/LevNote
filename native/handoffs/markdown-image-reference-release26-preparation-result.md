# Markdown image-reference .26 preparation result

2026-11-10. Preparation is complete and frozen for root review. The runnable
helpers live in the fresh `.26` evidence directory and are ready for one bounded
package/build/host delivery after explicit activation. This preparation performed
no package call, build, test, vet, worker/host launch, capture or resource
measurement. It preserved one initial PowerShell static parse as historical
evidence and ran exactly one root-authorized correction parse after restoring
the accepted `.25` strict pre-close lifecycle.

Target package:

`native/bin/levnote-native-worker-20261110.26`

Target version: `native-worker.20261110.26`

Evidence root:
[markdown-image-reference-release26-20261110](../artifacts/acceptance/markdown-image-reference-release26-20261110/)

## Final helper set

The twelve copied helpers are frozen with these byte and SHA-256 identities:

| Helper | Bytes | SHA-256 |
|---|---:|---|
|aggregate.ps1|33,235|`36E9B002FEBEEBAF90B812E6D69B74284CE303AF6AAE2EA7502329B5FC5C285D`|
|baseline.ps1|8,987|`4E7EC4401166298E75D75470594B2D7D525D5D91326838269816F4BBBF951AFF`|
|delivery-lib.ps1|29,289|`CF4CF4F46767393635B286378DDE694FBA768ADCAEF1E26D8C8F84FCCDE3419E`|
|initialize-context.ps1|12,078|`AD438359FF80423CC50D52BEFC0B6CFE093B1F24F6D0D7BE7005F76088FCFA3B`|
|initialize-delivery.ps1|2,129|`38AF8B6AFA5B34A2F34946C3DBFB3471FDAA46BA68A43470E91E3ED3A09F5AC6`|
|invoke-stage.ps1|2,653|`97369BF3581B5A31706C87DEE3F584D77FA938D4D47B46D1C98E92CB5B151DA7`|
|run-host-smoke.ps1|32,892|`6521E27A1139927D9878E7FC3F6CA868A33B393B720E503FCDD5A7562AEEB9FF`|
|run-package.ps1|17,324|`23E0A147DF35473FECB273D4FC0E6DE56DB79D38D66833C05FDB8BE167FD89AA`|
|run-preflight.ps1|1,299|`78839F34149B517F2AA3F0DA8D850F6A5193446C6D76D9877F9A7E7C1759252A`|
|run-stage.ps1|2,958|`9631549F240DD5E30A3B7DC8F39F1C4566903A1FF59CE5FABBC7A32B0F31F59A`|
|source-scope.ps1|2,157|`6CEFB378F9027F66892560CE92D3199C1F5F8C5D12FD32E3F40E557A78DE5FC5`|
|static-parse.ps1|1,794|`3CFFA602034A1E3210A62351F5A35DDF4F4E3CBB0CE107416B2E62075A5F8BEE`|

The external close helper
[markdown-image-reference-release26-close-inventory.ps1](markdown-image-reference-release26-close-inventory.ps1)
is 3,842 bytes, SHA-256
`B04A72C88843D129316D524EBF77764E059F16053C168B78CE74C5D2FD988342`.

The final copy/adaptation receipt is
[copied-helper-provenance.json](../artifacts/acceptance/markdown-image-reference-release26-20261110/copied-helper-provenance.json):
7,943 bytes, SHA-256
`7FCB37885B0B49ADBA226242E39F1AD59F7F534E8124AB0A03BC853CFA40A02F`.
It records all helper source/target hashes and the final portable README
identity. The progress checkpoint remains at
[markdown-image-reference-release26-preparation-checkpoint.md](markdown-image-reference-release26-preparation-checkpoint.md).

The helpers preserve the accepted lifecycle and retry rules. The inherited
hidden-host readiness rule was reconciled with the binding decision: after the
exact hidden window is found, require a bounded5000 ms alive settling period,
then capture a fresh strict ordered text+PNG active1 session before posting
exactly one owned `WM_CLOSE`. Normal close must publish the same valid session
with a newer timestamp relative to both the captured pre-close state and process
start. A pre-close collection or shape failure still performs the single normal
close for cleanup, then terminates the result after that close. The held process
and Job handles are explicitly disposed on success and failure.

An intermediate review draft weakened this requirement to nullable descriptive
pre-close evidence. Root rejected that unsupported policy drift. The final `.26`
copies restore the accepted `.25` strict predicates; only target version, source
identities, protected `.25` inputs, and portable wording remain as intentional
`.26` adaptations.

## Portable README

[native/portable/README.md](../portable/README.md) is 16,673 bytes, SHA-256
`66066F3E032BC7CB13B132C1E540A88437CE463AA116C01B40AAD444680FCE7A`.

The image-reference paragraph now says in plain Chinese that Ctrl/Cmd+点击 or
Tab+Enter uses the existing host open path; preview does not automatically load
the target image and an active click enters that path. It states the
1..4096 UTF-8 byte destination bound and explicitly rejects ASCII whitespace,
without claiming all Unicode whitespace is rejected. Preview and HTML do not
render inline pixels or fetch network resources; HTML keeps escaped literal
syntax and emits no `img` element.

Ordinary local relative targets preserve literal Windows backslashes and `%xx`.
The format layer performs no unescape/percent decode and passes the target text
literally. The README is narrowly accurate about the existing host: only
explicit `file:` URLs and anchors use its existing decoding rules, so no
universal “host never decodes” claim is made. Missing or unreadable targets
report failure, and readable unsupported files retain the bounded byte-inspection
entry.

The copied helper's README contract was run read-only against the final text
before static parse: no required Chinese/contract fragment was missing and no
forbidden stale wording was present. The inherited F6 sentence now says that F6
focuses the first row action, followed by Tab/Shift+Tab and Enter/Space.

## Source binding

Initialization requires fourteen exact source gates: the nine existing
image/table/component gates plus the image-reference root decision, format
result, UI result, final validation result and forthcoming image-reference root
acceptance. Composition and app integration acceptances remain separate required
bindings.

The thirty-eight accepted source identities are disjoint and current:

| Family | Count | Provenance |
|---|---:|---|
|Autosave/image|15|`native-autosave-evidence-20261110/10-corrected-source-after.json`, manifest SHA-256 `1DCDA557B8AD218DA8D31224937066C9CD5E4588D1AB09CF371F0D06B4D55008`|
|Markdown table|11|`markdown-table-columns-ui-evidence-20261110/08-final-source-after.json`, manifest SHA-256 `2432812FA53197C42B2431BD965810AAB7C864C3A5FF24DFCD9F5318C82D2377`|
|Image reference|12|`markdown-image-reference-ui-evidence-20261110/08-final-source-after.json`, manifest SHA-256 `D95C29FB86AF3F8E32E5871A77228B7545202B0813EB57F0CAAE1DB41FE54FFD`|

The final image-reference inventory is12 files /138,719 bytes, with byte-identical
pre/post validation and inventory SHA-256
`D95C29FB86AF3F8E32E5871A77228B7545202B0813EB57F0CAAE1DB41FE54FFD`.
No product source was edited by this preparation.

## Protected .25 baseline

The acceptance helpers protect the complete accepted `.25` package, direct
evidence, aggregate-capture sibling and external closed inventory, preserving
prior baselines transitively:

| Protected object | Files/bytes or size | Identity |
|---|---:|---|
|Package directory|26 files /27,259,765 B|`07391CBF37FF6406D5D4279CCED96CDBA0FDD3F5829B9685AA0066AA46CC7204`|
|Direct .25 evidence|146 files /55,539,894 B|`F383C6A160D3B56F55A1EEF0B4A083693C1C09EF2F97910AE06C37DDC5BC9646`|
|Aggregate capture attempt1|6 files /3,354 B|`FCD2AE8E96784802DBA7C3A193D350634F1F8E281EC97507799B70CA3E27ABAF`|
|External closed inventory|66,343 B|SHA-256 `39BADBA047C552252A13AD7645FA19D93B70DDF4FA8414FEB9B8E8362B3C99DF`|
|`.25` host|14,443,520 B|SHA-256 `2421D8FC98B144DCCB0ECC1CFF239A18E11DDEF05F12FF0505B307058419AE49`|
|`.25` release manifest|4,674 B|SHA-256 `CA064C3E7824EDCB179465CD8E60D982D057D2EA1D45A0E8407BD44508CF9BC2`|

The protected version is `native-worker.20261110.25`.

## Static parse history

The initial parse is preserved unchanged as historical evidence:

```powershell
& 'F:\AIproject\LevNote\native\artifacts\acceptance\markdown-image-reference-release26-20261110\run-preflight.ps1' -Step parse
```

Result: exit0; PowerShell7.6.9; 14 scripts parsed; 0 parse errors; stdout4,527
bytes; stderr0 bytes; wall time0.9013261 s. The script includes the twelve copied
helpers, the generated parse-command wrapper and the external close helper.
There was no parse retry.

- [parse invocation](../artifacts/acceptance/markdown-image-reference-release26-20261110/parse-invocation.json), 1,036 B, SHA-256 `4E0CFACE578D85CDD5E4D3B8A50D7B2D6459855F350A8124DB9429129E800DA9`
- [static parse result](../artifacts/acceptance/markdown-image-reference-release26-20261110/static-parse-result.json), 4,526 B, SHA-256 `EE75070D4C251D08634430AEE0EE4B0AECF14D2A19FAD12FAD38AA67AB43F7F8`
- [stdout](../artifacts/acceptance/markdown-image-reference-release26-20261110/parse.stdout.log), 4,527 B, SHA-256 `4F362BF9B9154E42B3BC13B83D0749081D9186F4F16D2438E7874F4D624E2F51`
- [stderr](../artifacts/acceptance/markdown-image-reference-release26-20261110/parse.stderr.log), 0 B, SHA-256 `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855`

That result was superseded for final helper-state validity after the strict
lifecycle restoration. The root review authorized exactly one correction-only
reparse into distinct paths, preserving the initial files:

```powershell
& 'F:\AIproject\LevNote\native\artifacts\acceptance\markdown-image-reference-release26-20261110\correction-parse-command.ps1'
```

Result: exit0; PowerShell7.6.9; 15 scripts parsed; 0 parse errors; stdout4,849
bytes; stderr0 bytes; wall time0.7703366 s. The count is one higher because the
correction command wrapper itself is in the scanned evidence root. There was no
second correction or later parse.

- [correction parse invocation](../artifacts/acceptance/markdown-image-reference-release26-20261110/correction-parse-invocation.json), 1,358 B, SHA-256 `E6B0EBBE50BB0EE4CFF2A081EB1CCFA007221308B606C6691BCAD4F882AFFA9E`
- [correction static parse result](../artifacts/acceptance/markdown-image-reference-release26-20261110/correction-static-parse-result.json), 4,848 B, SHA-256 `E7F7CCFE12FDD5681F470424579B84090319032D30EB00E6B2195E15A3947A50`
- [correction stdout](../artifacts/acceptance/markdown-image-reference-release26-20261110/correction-parse.stdout.log), 4,849 B, SHA-256 `C4B53295FA329DC4B947D1E0F29C42DE6ABC165C9BFCA1CD6559AD11737A3CCA`
- [correction stderr](../artifacts/acceptance/markdown-image-reference-release26-20261110/correction-parse.stderr.log), 0 B, SHA-256 `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855`

After the correction artifacts, the evidence root contained23 files /176,516
bytes before delivery began.

## Exact delivery sequence after activation

Run only after root writes
`native/handoffs/markdown-image-reference-root-acceptance.md` and explicitly
activates `.26` delivery. Use these copied artifact helpers, not preparation
files:

```powershell
& 'F:\AIproject\LevNote\native\artifacts\acceptance\markdown-image-reference-release26-20261110\run-preflight.ps1' -Step initialize
& 'F:\AIproject\LevNote\native\artifacts\acceptance\markdown-image-reference-release26-20261110\run-stage.ps1' -Name baseline
& 'F:\AIproject\LevNote\native\artifacts\acceptance\markdown-image-reference-release26-20261110\run-stage.ps1' -Name package
& 'F:\AIproject\LevNote\native\artifacts\acceptance\markdown-image-reference-release26-20261110\run-stage.ps1' -Name host-smoke
& 'F:\AIproject\LevNote\native\artifacts\acceptance\markdown-image-reference-release26-20261110\run-stage.ps1' -Name aggregate -AggregateAttempt 1
& 'F:\AIproject\LevNote\native\handoffs\markdown-image-reference-release26-close-inventory.ps1'
```

The expected budget is one package-script call/five GUI builds and one exact
copied hidden host. No additional worker request, test/vet repeat, physical
input, capture or resource measurement belongs to `.26` delivery. Root source
acceptance and explicit `.26` delivery activation now exist; product execution
starts only through the freshly copied evidence helpers listed above.
