# Open-document picker `.30` delivery preparation

2026-11-10. Status: **PREPARED-ONLY; FINAL PICKER SOURCE ACCEPTANCE PENDING**.
The twelve successful `.29` delivery helpers and one external closure helper
were copied into a fresh evidence directory and adapted for candidate `.30`.
One saved static parse ran and passed. No delivery context, candidate, build,
package, host, worker, test, capture, resource operation or product run was
performed.

## Candidate, source bindings and protection

- Candidate: `native/bin/levnote-native-worker-20261110.30`.
- Version: `native-worker.20261110.30`.
- Future source manifest:
  `native/handoffs/open-document-picker-final-evidence-20261110/source-frozen.json`.
  It must be a nonempty array with native-relative `Path`, `Length`, `Sha256`;
  every entry is validated without a hardcoded file count or intermediate
  hashes.
- Future source acceptance:
  `native/handoffs/open-document-picker-source-root-acceptance.md`.
- At preparation time the future source manifest and picker source acceptance
  did not exist. The candidate directory, delivery context and package do not
  exist either. Those files must supply final facts in the later delivery
  assignment; this report invents none of them.
- Initialization retains all earlier component/source gates, the separate
  historical native-image composition/final-app acceptances, the frozen bitmap
  lifecycle list, fixed engine/license/notice gates and both pinned MyGo patch
  identities. It appends the future picker source acceptance as the eighteenth
  required source gate.

Accepted `.29` is protected byte-for-byte:

| Protected object | Files | Bytes | Identity |
| --- | ---: | ---: | --- |
| `bin/levnote-native-worker-20261110.29` | 26 | 27,408,380 | `9FAEE40604A7DBA31ED14003011938911661E8BBBE0CCBDC5D9EBCDAA640962B` |
| `artifacts/acceptance/composition-release29-20261110` | 158 | 55,825,749 | `0F46070C6DFDD5CD30824376C2E6D4D9E459849DB45F9353366093EDF8388F89` |
| `artifacts/acceptance/composition-release29-20261110-aggregate-capture-attempt-1` | 6 | 3,243 | `1F4E2193425B76BBD06BF5C929B7662BF43758E20EF4236D65D8B643EEA81242` |
| `composition-release29-20261110-closed-inventory.json` | 1 | 69,492 | `5BB1C746F4590C1DC40A01F255E4F88E1D5E1C24F4D230047A4BA9A73AA78854` |

The helper also pins the accepted `.29` host at 14,580,224 bytes,
`9462194DDC7A310834D564A0A5DE2281A7C799BCE323989D1755CE698CD4CFC6`,
and its manifest at 4,674 bytes,
`8CCFC58E3D1A19847EF7A55289DE947C7C35D493DCEB1681772250D07AB00722`.

## Copied helpers and portable usage

[Copied-helper provenance](../artifacts/acceptance/open-document-picker-release30-20261110/copied-helper-provenance.json)
records source and destination absolute paths, source/destination byte lengths,
SHA-256 values, UTC write ticks and changed/unchanged status for all 13 helpers.
Four are byte-identical and nine contain only the authorized `.30`
candidate/source/protection/portable-contract adaptation.

Provenance identity: 10,087 bytes,
`3EA3ABF18D5BE5ED4ADCD69B03CDC20E3ABDFE40CBE25B932592A2885D0243A3`.

| Helper | Bytes | SHA-256 |
| --- | ---: | --- |
| `aggregate.ps1` | 33,144 | `0CB5038189885FAE5358168EAC84D4EE64002C481A71353F2F3840B811F4E3FC` |
| `run-package.ps1` | 17,596 | `50910BE7464BAE11DA8A31BDD744AB2D76235E12367AC1311736233CD9E3D835` |
| `initialize-delivery.ps1` | 2,506 | `0E8252EAC9F8E2F6215AA761772B7C2D4A989623C50A74FDAF9F97BB34CB2893` |
| `run-preflight.ps1` | 1,299 | `78839F34149B517F2AA3F0DA8D850F6A5193446C6D76D9877F9A7E7C1759252A` |
| `static-parse.ps1` | 1,790 | `B34156124693948B5FCF99155CD24CD489884AED1F47F4CF1D4273648943171A` |
| `invoke-stage.ps1` | 2,653 | `97369BF3581B5A31706C87DEE3F584D77FA938D4D47B46D1C98E92CB5B151DA7` |
| `run-stage.ps1` | 2,958 | `9631549F240DD5E30A3B7DC8F39F1C4566903A1FF59CE5FABBC7A32B0F31F59A` |
| `delivery-lib.ps1` | 30,481 | `1BB850C3E43527F4E3426B21C4EA2132E9707676EE1F9C8B78E4EEAE45DE82C4` |
| `initialize-context.ps1` | 12,095 | `93D25F673299719B0BB5DC532E0F63F2B86C918B59EDE67F9029F4CEF89472E8` |
| `baseline.ps1` | 8,963 | `CCD47A720E686730488ECA62BB0AC4D18DEFA99EF83D98C7E68942CD5ACFF11D` |
| `source-scope.ps1` | 2,157 | `6CEFB378F9027F66892560CE92D3199C1F5F8C5D12FD32E3F40E557A78DE5FC5` |
| `run-host-smoke.ps1` | 33,197 | `3EF211726C83C460589773FEB6C44797D2AEAEEA66CAF594BD258BADB97B5DAF` |
| `open-document-picker-release30-close-inventory.ps1` | 3,844 | `E0892ED069795A8E35CF3B6A466EC57E6CDBC3C07B695F2C0C70DD7F8E0E71EA` |

The first twelve helpers are under
`native/artifacts/acceptance/open-document-picker-release30-20261110/`; the
closure helper is under `native/handoffs/`.

[Portable README](../portable/README.md) remains UTF-8 without BOM with LF line
endings and is 19,649 bytes,
`44DA10ECD9912E95D1CB5B5FF68EFBA0FB48D128F0CFA2C7F9EA59286EEBD239`.
Its exact six-line picker paragraph begins at line 21 and occurs once. The
495-byte paragraph hashes to
`CD616A1590EC3FD44B0DFE1502A77A3B8D87BEC39AE1425C422EC268893A77AE`.
It describes `Ctrl+P` / “已打开文档”, current-tab-only case-insensitive
title/full-path substring filtering, arrows, Enter/click and Escape, query IME
key/pointer ownership, completion or cancellation of pending source/rich/
utility input, a bounded searchable metadata portion, and no body index, disk
search or new tabs. Existing `.28` size/IME/save/switch/close barriers and the
`.29` composition limits remain. This wording remains provisional until the
implementation owner publishes the final picker result.

[Portable identity evidence](../artifacts/acceptance/open-document-picker-release30-20261110/portable-readme-identity.json)
is 2,845 bytes,
`2AE0A83021791F5BDD0227BCD0E5F7A35CC1F4B4A2F148B02811F16C1A5889AF`.

## One static parse

Exactly one saved invocation was made:

```powershell
& native/artifacts/acceptance/open-document-picker-release30-20261110/run-preflight.ps1 -Step parse
```

It started `2026-11-10T12:06:33.0645977Z`, finished
`2026-11-10T12:06:33.8396744Z`, exited 0, and used PowerShell 7.6.9. The result
records 14 parsed scripts: the twelve copied helpers, the generated
`parse-command.ps1`, and the external closure helper. All 14 have zero parse
errors. Stdout is 4,471 bytes; stderr is the zero-byte empty stream. No parse
failure or corrective parse occurred.

| Saved object | Bytes | SHA-256 |
| --- | ---: | --- |
| `parse-command.ps1` | 305 | `0805E48D7AC249DEAD945393C135E8C46DE14705883CDEB9148D4E5702BC7AD9` |
| `parse-invocation.json` | 1,020 | `EEEB0BDBB8032A8E6174251ADDC6B3E65DA5B648E1BF22A28F29CDFB6DC3FC68` |
| `parse.stdout.log` | 4,471 | `3D32FAFD23F3FA31FC5831577D932AD2313EE38F1A544A77758B7483BB087FA4` |
| `parse.stderr.log` | 0 | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |
| `static-parse-result.json` | 4,470 | `AEEEC2F75886C5907E9B0EDA586F0DBC3D0245BDA9FA9B55FFAAC62726E6987D` |

## Prepared delivery sequence

After separate activation and creation of the future source manifest/root
acceptance, run this sequence once:

1. `run-preflight.ps1 -Step initialize`
2. `run-stage.ps1 -Name baseline`
3. `run-stage.ps1 -Name package`
4. `run-stage.ps1 -Name host-smoke`
5. `run-stage.ps1 -Name aggregate -AggregateAttempt 1`
6. `native/handoffs/open-document-picker-release30-close-inventory.ps1`

The delivery budget is one package invocation, five GUI builds, one copied
hidden host, and zero worker requests, captures, resource measurements or
second-restore runs. The isolated `C:` host opens ordered UTF-8+BOM/CRLF text
then PNG through CLI requests, remains alive for 5,000 ms, requires a fresh
strict two-tab session before one owned `WM_CLOSE`, then requires newer
normal-close publication, exit 0 and exact owned cleanup. This proves CLI
open/session publication and normal close, not picker interaction, physical
IME behavior or a second-launch restore.

Preserve every actual failure. A failed aggregate attempt is inspected before
any metadata-only correction; no blind retry, and no baseline, package, host,
parser or source/test/build rerun. Run external closure only after a successful
aggregate and exited writers.

## Nonclaims

This is preparation evidence only. It creates no `.30` context, candidate,
package, host evidence, picker source acceptance or root acceptance. No runtime
source, test, accepted `.29` artifact or product output was changed. The
portable paragraph remains provisional pending the picker owner’s final
implementation result and root source acceptance.
