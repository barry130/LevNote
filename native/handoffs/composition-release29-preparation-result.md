# Composition coalescing `.29` delivery preparation

2026-11-10. Status: **PREPARED-ONLY; FINAL SOURCE ACCEPTANCE PENDING**.
The twelve `.28` delivery helpers and one external closure helper were copied
into the fresh evidence directory and adapted for candidate `.29`. One static
parse ran and passed. No delivery context, candidate, build, package, host,
worker, test, capture or resource operation was run.

## Candidate, source bindings and protection

- Candidate: `native/bin/levnote-native-worker-20261110.29`.
- Version: `native-worker.20261110.29`.
- Future source manifest:
  `native/handoffs/active-composition-coalescing-final-evidence-20261110/source-frozen.json`.
  It must be a nonempty array with native-relative `Path`, `Length`, `Sha256`;
  the helper validates every entry and does not assume a file count.
- Future source acceptance:
  `native/handoffs/active-composition-coalescing-source-root-acceptance.md`.
- Final source facts remain pending. At preparation time neither the future
  `source-frozen.json` nor the future source acceptance existed. Independent
  review has completed without a blocker; the active final validation and root
  acceptance still own those facts. Nothing in this report invents them.
- Source gates also require the existing composition and final-app integration
  acceptances named by `initialize-delivery.ps1`.

Accepted `.28` is protected byte-for-byte:

| Protected object | Files | Bytes | Identity |
| --- | ---: | ---: | --- |
| `bin/levnote-native-worker-20261110.28` | 26 | 27,395,153 | `B2BB8C7EB09C2D917882FBFA424D8B921F4504619E37A5DE17E3B10B18E5D8CB` |
| `artifacts/acceptance/growth-release28-20261110` | 146 | 55,786,384 | `854427F529599D5912F316C61CD313147D0D15F1105EA9E78ED0C79394F96075` |
| `artifacts/acceptance/growth-release28-20261110-aggregate-capture-attempt-1` | 6 | 3,193 | `504D8519A9A4DC2BAFFA9EA86F75B57BE2C0B5A427B711C6A8310669E3541A7E` |
| `growth-release28-20261110-closed-inventory.json` | 1 | 63,713 | `7ADEE2D137C4FC9CB61DCE6FC5C5B3BAF306F3C587B8E4D0700D7973C3FAEA46` |

The helper also pins the accepted `.28` host at 14,567,424 bytes,
`8DB245432AB4C2358667B837A8D0CA7F164EA6CC8696C09D585AF194A2F04C64`,
and its manifest at 4,674 bytes,
`999B4EAB52C6BC2E54D9ED74121CA3DBD4DFEE85CE9AAE678EF0C6297C895E24`.

## Copied helpers

[Copied-helper provenance](../artifacts/acceptance/composition-release29-20261110/copied-helper-provenance.json)
records both source and destination absolute paths, source/destination byte
lengths and SHA-256 values, UTC write ticks, and changed/unchanged status for all
13 helpers. Four are byte-identical and nine contain the authorized `.29`
version/path/source-gate/protection/portable-wording adaptation.

Provenance identity: 9,931 bytes,
`BB5289C4D47C9BF46F14A3F49526FBB0A2993A2578B52A4C26CEDCBCF3EB69F4`.

| Helper | Bytes | SHA-256 |
| --- | ---: | --- |
| `aggregate.ps1` | 33,096 | `0FCCE152C1E1A7947267CCEAF045699B48EE474369FE9E8F4E30F0390D6183D5` |
| `run-package.ps1` | 17,611 | `E6F072A88AD08D43BCD9A27EB1C5160D0CDCCFD2B15F2BB5127B6DA882B90E33` |
| `initialize-delivery.ps1` | 2,412 | `35FF82102B823873710F6C773DA60057A3A785B1165459AE95220B936E3E87C8` |
| `run-preflight.ps1` | 1,299 | `78839F34149B517F2AA3F0DA8D850F6A5193446C6D76D9877F9A7E7C1759252A` |
| `static-parse.ps1` | 1,781 | `DC94E62FA70ABA5128B50B671005866AA685FFB258A9F206F615BAD7760A7F20` |
| `invoke-stage.ps1` | 2,653 | `97369BF3581B5A31706C87DEE3F584D77FA938D4D47B46D1C98E92CB5B151DA7` |
| `run-stage.ps1` | 2,958 | `9631549F240DD5E30A3B7DC8F39F1C4566903A1FF59CE5FABBC7A32B0F31F59A` |
| `delivery-lib.ps1` | 30,040 | `6E810172D72C0EEF23A5F357A7BB8ED833B0F13F9742E9E51D03DCAF78FAD7BB` |
| `initialize-context.ps1` | 12,022 | `F7A7A725DC6C575B452C47D82B5CA4F332CDA5CB66B39D41C0763458379261B8` |
| `baseline.ps1` | 8,973 | `1E457E0147C6011712BE0C0F4A3F577F4052F737BAAE5BB9B6EE7794A0BFA7D8` |
| `source-scope.ps1` | 2,157 | `6CEFB378F9027F66892560CE92D3199C1F5F8C5D12FD32E3F40E557A78DE5FC5` |
| `run-host-smoke.ps1` | 33,159 | `F36C55268019F5450157E614164C9837A8D3491AEEA6BB31B5540079334A2D80` |
| `composition-release29-close-inventory.ps1` | 3,828 | `84BC9C79795A7F1DE3AF59A7F2E6EA76D7FCBA226826F81934335363A843F56D` |

The first twelve helpers are under
`native/artifacts/acceptance/composition-release29-20261110/`; the closure
helper is under `native/handoffs/`.

## One static parse

Exactly one saved invocation was made:

```powershell
& native/artifacts/acceptance/composition-release29-20261110/run-preflight.ps1 -Step parse
```

It started `2026-11-10T11:31:54.6189506Z`, finished
`2026-11-10T11:31:55.6830722Z`, exited 0, and used PowerShell 7.6.9. The result
records 14 parsed scripts: the twelve copied helpers, the generated
`parse-command.ps1`, and the external closure helper. All 14 have zero parse
errors. Stdout is 4,345 bytes; stderr is the zero-byte empty stream. No parse
failure or corrective parse occurred.

| Saved object | Bytes | SHA-256 |
| --- | ---: | --- |
| `parse-command.ps1` | 287 | `06A62AE7EE5BA40DE346D9D35D7E124D353F97E6F525796A74F0D98021BB0834` |
| `parse-invocation.json` | 984 | `76A3EC74198577A9B470F9905A0F12A1B4085FEED37909FCA975B9B310AEB750` |
| `parse.stdout.log` | 4,345 | `3BF58F11F3A8AC55736A215D2B8BB951C0D1B0C3A657CB6A6A8CB9888BC7297E` |
| `parse.stderr.log` | 0 | `E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855` |
| `static-parse-result.json` | 4,344 | `2BEA9E54B5F7CDB4705FDFE236C46B03172160504B94E3EC727E3F7462E23805` |

## Portable README

[Portable README](../portable/README.md) is UTF-8 without BOM, uses LF line
endings, and is 19,152 bytes with SHA-256
`5C209387A537E1307467792B68F08E5FB756178034E89F226D70E4C5DD3FC6B7`.

The exact four-line composition paragraph begins at line 44 and occurs once.
Its 425 UTF-8 bytes hash to
`F5A0B1D70C4FD49680BCFF8B4086A0971D21AC5A36CB7D0EB78581719988C336`.
It says that consecutive full-span composition updates retain the group's
original and current text, regular undo/cancel/redo still acts as one group,
and irregular explicit ranges safely append suffixes that may retain more
history. It explicitly makes no whole-process memory bound and preserves the
`.28` size refusal, IME retry/Escape, and save/switch/close barriers.

[Portable identity evidence](../artifacts/acceptance/composition-release29-20261110/portable-readme-identity.json)
is 1,224 bytes,
`35C4B583B935BC18744C4DC8B82F1A5CEF302A4B6B6AB82FF59FA669D30FD1CC`.

## Prepared execution sequence

After a separate activation and the future source manifest/root acceptance,
run this sequence once:

1. `run-preflight.ps1 -Step initialize`
2. `run-stage.ps1 -Name baseline`
3. `run-stage.ps1 -Name package`
4. `run-stage.ps1 -Name host-smoke`
5. `run-stage.ps1 -Name aggregate -AggregateAttempt 1`
6. `native/handoffs/composition-release29-close-inventory.ps1`

The later package budget is one package invocation, five GUI builds, one copied
hidden host, and zero worker requests/captures/resources. The host uses an
isolated `C:` configuration, opens UTF-8+BOM/CRLF text and then PNG through
ordered CLI requests, settles alive for 5,000 ms, requires fresh strict
two-tab/active-index-1 pre-close publication, receives one owned `WM_CLOSE`,
then requires newer normal-close publication, exit 0 and exact cleanup. This
proves CLI open/session publication/normal close, not a second-launch restore
or physical IME behavior.

Attempt 1 is the aggregate opportunity. If it actually fails, preserve it and
run only the prepared aggregate attempt 2 wrapper; do not rerun baseline,
package, host, parser or helpers. The external closure runs only after a
successful aggregate.

## Nonclaims

This is preparation evidence only. It creates no `.29` context, candidate,
package, host evidence, source acceptance or root acceptance. No runtime source,
test, accepted `.28` artifact, product documentation or portable product claim
was executed or changed beyond the authorized composition paragraph.
