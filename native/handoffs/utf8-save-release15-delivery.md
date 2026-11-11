# UTF-8 streaming save: .15 delivery

Status: CANDIDATE-READY for root binary acceptance, 2026-11-09. The accepted
`.14` package remains the current runnable package until root accepts this
candidate. Product source, tests, package scripts and product documentation
were frozen during delivery.

## Candidate

Directory:

```text
F:\AIproject\LevNote\native\bin\levnote-native-worker-20261109.15
```

Version: `native-worker.20261109.15`

The candidate contains 23 files totaling 23,680,582 bytes:

```text
1490F8DF8FA07D6EAE05DCCB2C92947DB086B392263D005BC29A87CCDE9AEF16
```

The evidence copy also contains 23 files / 23,680,582 bytes and every relative
path, size and file SHA-256 matches the candidate.

Host:

```text
14,056,960 bytes
3C5B07646C2E50A82304B88A02B59C2EB170F067599B7FFAB044DCCEBF3DD804  levnote-native.exe
```

Release manifest:

```text
4,137 bytes, 22 payload entries plus itself
3A451E08D9394F829CEE0E85D7C7F14F1FFB735C47049DB6CEF502EE22E35D32  release-manifest.json
```

The three engine records independently match their packaged worker files and
manifests, including the `.15` version:

| engine | worker bytes | worker SHA-256 | manifest SHA-256 |
|---|---:|---|---|
| `levnote.markdown-export` | 3,300,352 | `C2AC3F50B9ED7F21CEC601F245C90A792680C1F8FEAC366E97ACDFEAE6F2CE5C` | `0AA9F22AD1FCD725829FD4DF2FDF18888A98C2D98B8B5CA5FFBD6862BD05525C` |
| `levnote.mindmap-svg` | 3,122,688 | `0F12D56B4D149C4E769DE0B3F234FECC7D69E79CD51AC923FFE38D53B414E329` | `5666E428E0D9B0912C68C52114D76D6E2EF8801AFBDE109C77F51C4E2BBB73D6` |
| `levnote.mindmap-preview` | 3,136,512 | `B5BD4B0C28CCFE5DD4BFF3446F2728428B57CD7A4CFF9062DBBC456A49A67429` | `5D56585E1C6E20A0B5A76C1B405906F15A8854CCF37FF48740B78BC454D4CC9D` |

All four executables are Windows amd64 GUI PE images. The 13 portable source
assets retain their copied SHA-256 identities.

## Package and builds

Delivery used exactly one existing package-script invocation:

```text
native\scripts\package-markdown-worker.ps1
  -OutputDirectory F:\AIproject\LevNote\native\bin\levnote-native-worker-20261109.15
  -EvidenceDirectory F:\AIproject\LevNote\native\artifacts\acceptance\utf8-save-release15-20261109-234608\package
  -Version native-worker.20261109.15
```

The package invocation exited 0 in 6.9633088 seconds; its outer stage exited 0
in 8.289054 seconds. It made exactly four hidden Go wrapper GUI builds:

| build | wall seconds |
|---|---:|
| `worker-markdown` | 1.207411 |
| `worker-mindmap` | 1.0064216 |
| `worker-mindmap-preview` | 1.2096655 |
| `host` | 2.1176615 |

Every build exited 0 with empty stdout and stderr. No managed tester compile,
managed tester run or production worker request was made.

## Preservation and accepted source

The scoped source/build-input inventory contains 295 files / 2,975,431 bytes,
including 60 `native/formats` files and the referenced
`native/portable/NOTICE-PREAMBLE.md`. `inputs-before.json` and
`inputs-after.json` are byte/time identical:

```text
71,335 bytes
F9AB7389E8C5FF94C4AFD62C999BC6DACC0C5B8A01A680C6D1C329925ECD63BC
```

The five accepted UTF-8 source/test files retain their frozen identities:

| file | bytes | SHA-256 |
|---|---:|---|
| `native/core/document/atomic.go` | 2,990 | `872E1467775737A341AB202563276F48A4EF255C278312CB006403C1716281C0` |
| `native/core/document/document.go` | 12,493 | `C405449A2BD981EEE09DEFB89F7C113CF40E0254D6673DC99E2945171C08E1F4` |
| `native/core/document/document_test.go` | 7,489 | `CF6B0A37B7874FC6070375F59C4244A219C40B44FD452A9FE4D8E4E4036F1B38` |
| `native/core/document/utf8_save_test.go` | 8,197 | `446F7542B0447FEA4F29B92796F6CB4B32B3AEB249E914BFB279731F78302863` |
| `native/core/document/utf8_save_benchmark_test.go` | 2,223 | `3BE4BC99538EFF5EAEF03E118BFBD12A1B6510C2B65806B24A810FF6EF2FB4EE` |

The ten accepted inputs were retained. Their identities include:

```text
7,927  142EEE3090FA6A3D2CC950915D35CCAE49C34EE565186B87F8DC0BE7A90209A8  utf8-save-implementation-result.md
2,423  D6328D2F0D356F8B746BC986FFDA8019AF58279A4E17E31C8853AA8FA2BF42DC  utf8-save-root-acceptance.md
7,075  045CDA71B9945A7D486E5A703F4B4280AEB459368843E092B571AD77FCE9F9B1  utf8-save-independent-reconciliation.md
```

The focused compile, focused UTF-8 tests, final output benchmark, final affected
suite and final vet logs were also unchanged. `accepted-gates.json` is 2,601
bytes SHA-256
`B53EE5ACECF8EB2168D1E9418C8334E285C2A61745CADBD9DE4FC6D62112023A`.

The complete accepted `.14` package remains 23 files / 23,673,926 bytes:

```text
55083524EA6293313503B63CA2212BD0F6D2FE352B3DBF969FBE4643B531261D  full inventory identity
D037EF7B3DC700E84AA9484AE9A0A7A706662DB8A834D8D3CC4C4B53A3C4FC9A  levnote-native.exe
A876F6E298DCDFA176178E5B2749383AF0CD5053984355751C0A0F10F9BF5379  release-manifest.json
```

Its complete 104-file / 47,738,730-byte delivery evidence also remains
unchanged:

```text
F1A659DA07BAB54C006BA8D5457908692276ABF702B0171BB03FF07BBE14535C  full evidence inventory identity
66242F93DC7948DA319438A165EEA433AB4C2E2CD9CA5FC3EA6237A1AE05C765  accepted aggregate.json
```

The initially prepared delivery context contained literal hash placeholders.
The failed baseline attempt is retained as `baseline-attempt01-*` and
`context-placeholder-failure.json`; `context-repair-record.json` records the
metadata-only correction. No package invocation had occurred before that
correction.

## Hidden host smoke

One hidden smoke launched an exact 23-file copy of the `.15` bundle from `F:`,
with fresh run-owned configuration on the normal `C:` temp volume and a
generated 60-byte Markdown source:

```text
PID:                       36012
HeldProcessHandle:         2228
Start UTC:                 2026-11-09T15:50:43.2846580Z
Executable volume:         F:\
Config volume:             C:\
Window title:              轻笺 · 原生版
WindowVisible:             false
OwnedWMClosePosted:        true
AppExitCode:               0
ForcedAppCleanup:          false
OwnedAppStillLive:         false
RequestedWorkerOperations: 0
```

Only the held process identity and its owned window were used. The source
retained its 60 bytes and SHA-256
`7CBEA90B48A026B9B0704F5F37F4AF7FDBB912BB15ABD6A4FC421D9003EFB6A1`.
The copied bundle and candidate were preserved. The recovered session contained
one UTF-8 Markdown tab without BOM or draft:

```text
408 bytes
1028497A55298F71A1D9C1FF469189D9EEA4A92D4E289FA67728FF1E5593DC62  session-after-close.json
```

Application stdout and stderr were empty. The smoke operation exited 0 in
3.0666922 seconds; its outer stage exited 0 in 4.7872024 seconds.

The accepted `.11` managed real old-release/bundled-reset execution remains
historical integration evidence. It was not rerun for `.15`.

## Aggregate

The final validator and aggregate passed **12/12 without retry**. They checked:

- one package invocation and exactly four successful GUI builds;
- the exact candidate and package copy;
- the release-manifest payload;
- four GUI PE images and three engine manifests;
- all 13 portable assets;
- the complete protected `.14` package and delivery evidence;
- unchanged scoped source/build inputs and the five frozen UTF-8 files;
- unchanged accepted results, raw evidence, reconciliation and root gates; and
- one exact-copy hidden host start with owned normal close and preserved data.

```text
Validator: 9,504 bytes
9644A2DA72C850220780F590C9A306D04F57EAAA37522BC02D641421303250BA  validator.json

Aggregate: 16,519 bytes
C5A536C1FD5F175F3E9AE96057F3BB24F340BE3AB55C8973E18218C8CD465D1E  aggregate.json
```

The outer aggregate stage exited 0 in 1.469717 seconds.

## Evidence

Evidence root:

```text
F:\AIproject\LevNote\native\artifacts\acceptance\utf8-save-release15-20261109-234608
```

At completion it contains 109 files / 47,758,090 bytes:

```text
4FA520D0A383397F1C4AD5F0B7AA59DEFC2A3B1020C3C2313D020DB2E6796EDD
```

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
context-repair-record.json
package-command.json
package\package-result.json
package\worker-*-build.json
host-smoke\result.json
host-smoke\session-after-close.json
stage-*-invocation.json
```

## Limits

This evidence establishes package construction, frozen-source preservation,
manifest/asset consistency and one hidden exact-host startup with an owned
normal close. It does not exercise a dirty UTF-8 Save in the hidden host, import
or execute the packaged engines, or repeat any focused test, full suite,
benchmark, managed tester compile/run or production worker request.

The accepted synthetic UTF-8 tests cover the streaming output, exact-digest,
readback, state-commit, cleanup and short-write paths. The bounded benchmark
measured encoded-output allocation, not whole-Save latency. This delivery makes
no physical Windows IME, visible UI automation, worker execution, CPU, GPU,
process-memory or other resource claim.
