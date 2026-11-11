# Session output: .16 delivery

Status: CANDIDATE-READY for root binary acceptance, 2026-11-10. The accepted
`.15` package remains the current runnable package until root accepts this
candidate. Product source, tests, package scripts and product documentation
were frozen during delivery.

## Candidate

Directory:

```text
F:\AIproject\LevNote\native\bin\levnote-native-worker-20261110.16
```

Version: `native-worker.20261110.16`

The candidate contains 23 files totalling 23,690,310 bytes:

```text
F9937B33552496EAEDA1FB2A7F68B7F27FB769915A6E4FEFC546E566D10E9CF5
```

The package-copy and hidden-host copies contain the same 23 relative paths,
bytes and SHA-256 values and have the same complete inventory identity. The
host-copy and package-copy identities were checked again after the hidden
startup/close.

Host:

```text
14,066,688 bytes
76A60252CFCC3394386B4DAF87AF8B845086DE35ED5DCCACC6CC79179E60B03D  levnote-native.exe
```

Release manifest:

```text
4,137 bytes, 22 payload entries plus itself
53994DB6861F4B9918D33F9FF90330744862730C838417E0C10501ADA050CD90  release-manifest.json
```

The three engine manifests independently match their packaged worker files and
the `.16` version:

| engine | worker bytes | worker SHA-256 | manifest SHA-256 |
|---|---:|---|---|
| `levnote.markdown-export` | 3,300,352 | `F2D971EAE32D655D006D2D5AF8EAA258B0678FE0450C954FF81062A6D608E349` | `F7BC1806A1560CB7F57BE1181B6857D6A5761DE548C6B7D4415AE5A17E7F037A` |
| `levnote.mindmap-svg` | 3,122,688 | `59B620A0F63BA6EC7E5F24A932A3FAB81433496D38B17BA9C590491E62F45389` | `B792D23B125B62B5BACF1AC260937CBF81C8EEBD18E7E1C1DDDD7C86FFE42B8B` |
| `levnote.mindmap-preview` | 3,136,512 | `FB5582028336DBBE12115AA65A5D9E3BDAEC38C9D83B8A72B451DFC7AE304139` | `F4F4733FE5904AAEBE278B8347765C35E7D6806164827C85A68121AD62238F13` |

All four executables are Windows amd64 GUI PE images. The 15 packaged notice,
license and attribution assets are present, and all 13 source-copy SHA-256
identities match their repository inputs.

## Package and builds

Delivery used exactly one invocation of the existing fixed package script:

```text
native\scripts\package-markdown-worker.ps1
  -OutputDirectory F:\AIproject\LevNote\native\bin\levnote-native-worker-20261110.16
  -EvidenceDirectory F:\AIproject\LevNote\native\artifacts\acceptance\session-output-release16-20261110-001354\package
  -Version native-worker.20261110.16
```

The package invocation exited 0 in 7.398546 seconds; the outer stage exited 0
in 8.7133451 seconds. Its captured stdout was the candidate path plus newline:

```text
F:\AIproject\LevNote\native\bin\levnote-native-worker-20261110.16
```

Its stderr was empty. It made exactly four hidden Go wrapper GUI builds:

| build | wall seconds |
|---|---:|
| `worker-markdown` | 1.23231 |
| `worker-mindmap` | 1.3591193 |
| `worker-mindmap-preview` | 0.9856612 |
| `host` | 2.3731005 |

Every build exited 0 with empty stdout and stderr. No managed tester compile,
tester run, production worker request, package retry or second package invocation
was made.

## Preservation and accepted source

The pre-build baseline passed before the candidate directory was created. The
scoped source/build-input inventory contains 299 files / 2,996,205 bytes,
including 60 `native/formats` files and the referenced existing
`native/portable/NOTICE-PREAMBLE.md`. `inputs-before.json` and
`inputs-after.json` are byte-identical:

```text
72,290 bytes
DB7F0FB618EE3732FCD4EFB0A7E259839D0B4F1DE16D9CE7AD7BC1F5D234D39E
```

The six accepted session-output source/test files retain their frozen
identities:

| file | bytes | SHA-256 |
|---|---:|---|
| `native/core/document/state.go` | 1,083 | `979DFD1E0945EBC57341712C2CE9CF8D5A55B279447382ACCD1F03748C9FB368` |
| `native/core/session/session.go` | 6,085 | `BA3580DBDA6B92984EA9E1844B211FC5FF7ED690CF0E19D30F799620E5402132` |
| `native/core/session/output.go` | 2,009 | `AA06E8E3C5C85BAE8940FABA92673582DF41B8E9F7FBD64AAD1B47DED45407EA` |
| `native/core/document/state_test.go` | 1,971 | `940C51F8ABD1CF731C196276C6A77B4F5EBADE570DE44DD42A56E25AAAB35E24` |
| `native/core/session/output_test.go` | 14,054 | `43739380C6C5E9208FE18D94A9194DA2D812465D678702A2371A0EA33D7DDFA7` |
| `native/core/session/output_benchmark_test.go` | 2,280 | `D3C6DD70CFF82D20F129BC3A9BF273DEA5E111F6CEF1EAF5125B27AD7850B7B0` |

The implementation result, root acceptance, independent reconciliation and six
final raw evidence logs were unchanged:

```text
BE517D20BB9E127FFA791EB5EC264B41584FF03132BBAD1DA04E27497C8E208E  session-output-implementation-result.md
6C45ABE9CCE28F3DCBA9C6C5DAB3C026B673B6229E823043154134CF389E5ACF  session-output-root-acceptance.md
DBDC93638F1E92328551255EEDA0C100C18F69CDE516CA0F79D4F565C0C5DA42  session-output-independent-reconciliation.md
F970B73C3D09D505A5692515E43F00652E8C8AAB4F52823B19EF503EE7AE5E23  review-focused-red-1.log
964CAE1AAC6B7AFC07181CB8C0337938742EBD2E1EA5F9987FBE3918F34C4BFF  review-focused-red-2.log
A35C416D36A39D6C5A32CFB8CC7F9955A5DEC2B1F93C061AD1E2EE61187D79AD  review-focused-green-1.log
4457C3757892F8BCD9B3AB5B8122B028B49008519DE1296E2C1ECC72B49FE8D3  review-session-output-benchmark-1.log
B2268E1D62703561D65604CD2F519F035AD660C944A3F291E42321893269F524  review-final-affected-suite-1.log
EAFAF8D4E06C5830A9BD826C69B9E66867C2C8A4AE7D257361E9738061B88CBC  review-final-affected-vet-1.log
```

The complete accepted `.15` package remains 23 files / 23,680,582 bytes:

```text
1490F8DF8FA07D6EAE05DCCB2C92947DB086B392263D005BC29A87CCDE9AEF16  package inventory identity
3C5B07646C2E50A82304B88A02B59C2EB170F067599B7FFAB044DCCEBF3DD804  levnote-native.exe
3A451E08D9394F829CEE0E85D7C7F14F1FFB735C47049DB6CEF502EE22E35D32  release-manifest.json
```

Its complete 109-file / 47,758,090-byte direct evidence remains unchanged:

```text
4FA520D0A383397F1C4AD5F0B7AA59DEFC2A3B1020C3C2313D020DB2E6796EDD  inventory identity
C5A536C1FD5F175F3E9AE96057F3BB24F340BE3AB55C8973E18218C8CD465D1E  aggregate.json
```

## Hidden host smoke

One hidden smoke launched a byte-identical 23-file copy of the `.16` bundle
from `F:`, with fresh run-owned configuration on the normal `C:` temp volume
and a generated 60-byte Markdown source:

```text
PID:                       27980
HeldProcessHandle:         1568
Start UTC:                 2026-11-09T16:15:58.9765481Z
Executable volume:         F:\
Config volume:             C:\
Window title:              轻笺 · 原生版
WindowVisible:             false
Owned WM_CLOSE posted:     true
AppExitCode:               0
ForcedAppCleanup:          false
OwnedAppStillLive:         false
RequestedWorkerOperations: 0
```

Only the held process identity and its owned window were used. The source kept
its 60 bytes and SHA-256
`7CBEA90B48A026B9B0704F5F37F4AF7FDBB912BB15ABD6A4FC421D9003EFB6A1`.
The candidate and copied bundle were preserved byte-for-byte.

The post-close session parsed as exactly one UTF-8 Markdown tab without BOM or
draft. Its final byte is `}`; it has no terminal LF or CR:

```text
413 bytes
B66C699AD0C5D0FD11E34E86FCD0FE3037AC9CB9D183C7A07E3F0138A9A8D0C8  session-after-close.json
```

Application stdout and stderr were empty. The smoke operation exited 0 in
3.8339934 seconds; its outer stage exited 0 in 5.5361049 seconds. The accepted
`.11` managed old-release/bundled-reset execution remains historical evidence
and was not rerun.

## Aggregate

The final validator and aggregate passed **12/12 without retry**. They checked:

- one package invocation and exactly four successful GUI builds;
- the exact candidate and byte-identical package copy;
- the release-manifest payload;
- four Windows amd64 GUI PE images and three engine manifests;
- 15 packaged notice/license/attribution assets and 13 source-copy identities;
- the complete protected `.15` package and direct evidence inventories;
- unchanged scoped source/build inputs and the six frozen session files;
- unchanged accepted result, raw logs, reconciliation and root gates; and
- one exact-copy hidden host start with owned normal close, no terminal session
  newline, and preserved source and bundle.

```text
Validator: 9,786 bytes
EBA1A54A1EF2A5AC4AB797EFD846D7CB3FDD61CB6791B015DBEADF526C8961E4  validator.json

Aggregate: 16,743 bytes
00BF4482CB0C660A98A60C7CAD7A7CA0E1D61757B30D57CD4AD673EB90B5C0A5  aggregate.json
```

The outer aggregate stage exited 0 in 1.4256575 seconds.

## Evidence

Evidence root:

```text
F:\AIproject\LevNote\native\artifacts\acceptance\session-output-release16-20261110-001354
```

At completion it contains 104 files / 47,774,882 bytes:

```text
15A3050C4B0F1C471798B4A958B457963D44A50A634E8113550B34001A2BB9F2
```

The package result is 7,316 bytes SHA-256
`4D8DCF6D80CADD680293802A371BBBF5742D7271778AA5508A33C7D3586C0C6D`.
Start with:

```text
validator.json
aggregate.json
inputs-before.json
inputs-after.json
protected-before.json
protected-after.json
protected-evidence-before.json
protected-evidence-after.json
package-command.json
package/package-result.json
package/worker-*-build.json
host-smoke/result.json
host-smoke/session-after-close.json
stage-*-invocation.json
```

## Limits

This evidence establishes package construction, frozen-source preservation,
manifest/asset consistency and one hidden exact-host startup with an owned
normal close. It does not execute a dirty save in the hidden host, import or
execute a packaged engine, or repeat any focused test, affected suite, vet,
benchmark, managed tester compile/run or production worker request.

The accepted output benchmark measures encoded-output allocations, not whole
Save latency. `json.Encoder` still holds its complete internal output and draft
strings remain in memory. This delivery makes no token-streaming, fixed-memory,
RSS/working-set, physical IME, visible-UI, CPU, GPU or application-speed claim.
