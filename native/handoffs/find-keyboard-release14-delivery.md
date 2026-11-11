# Find keyboard flow: .14 delivery

Status: CANDIDATE-READY for root binary acceptance, 2026-11-09. `.13` remains
the current accepted runnable package until root accepts this candidate.
Product source, tests, package scripts and product documentation were frozen
during delivery.

## Candidate

Directory:

```text
F:\AIproject\LevNote\native\bin\levnote-native-worker-20261109.14
```

Version: `native-worker.20261109.14`

The candidate and its evidence copy each contain 23 files totaling 23,673,926
bytes and are byte-identical.

Host:

```text
14,050,304 bytes
D037EF7B3DC700E84AA9484AE9A0A7A706662DB8A834D8D3CC4C4B53A3C4FC9A  levnote-native.exe
```

Release manifest:

```text
4,137 bytes, 22 payload entries plus itself
A876F6E298DCDFA176178E5B2749383AF0CD5053984355751C0A0F10F9BF5379  release-manifest.json
```

The three engine records independently match their package files and manifests:

| engine | worker bytes | worker SHA-256 | manifest SHA-256 |
|---|---:|---|---|
| `levnote.markdown-export` | 3,300,352 | `D41B000CA08BA28220665E0727B79781B9410CB8561DF617F9324A18D464E680` | `F4537F6C2AF8630392FA7919738063B902136D188D0C2952ED50D3A04AC8EE55` |
| `levnote.mindmap-svg` | 3,122,688 | `CA8B1E2E6EB2518B01DED73FF2CAD950AAF6B4DA018AB40BB4768B32DCAFF8E2` | `24A4F67858FB728348C3D991572670412E60CA7929019FF3FE4ED47ECC745B50` |
| `levnote.mindmap-preview` | 3,136,512 | `DE119B4294AA1F00B9D4B29876E9E7CE3BA9110AFBC81FEBA63BBD3D1C46C087` | `05B1E35355DC44C724142701935F420C5ED40749D4B564D10F3127A361CDBF81` |

## Package and builds

The delivery used exactly one existing package-script invocation:

```text
native\scripts\package-markdown-worker.ps1
  -OutputDirectory F:\AIproject\LevNote\native\bin\levnote-native-worker-20261109.14
  -EvidenceDirectory F:\AIproject\LevNote\native\artifacts\acceptance\find-keyboard-release14-delivery-20261109-215637\package
  -Version native-worker.20261109.14
```

The package invocation exited 0 in 7.9638189 seconds; its outer stage exited 0
in 9.3949979 seconds. It made exactly four hidden Windows amd64 GUI builds:

| build | wall seconds |
|---|---:|
| `worker-markdown` | 1.2194189 |
| `worker-mindmap` | 0.9725370 |
| `worker-mindmap-preview` | 1.2524333 |
| `host` | 3.0209420 |

All four builds exited 0 with empty stdout and stderr. No managed tester compile,
managed tester run or production worker request was made.

## Preservation and accepted source

The scoped source/build-input inventory contained 293 files / 2,962,619 bytes,
including 60 `native/formats` files and the referenced
`native/portable/NOTICE-PREAMBLE.md`. `inputs-before.json` and
`inputs-after.json` are identical:

```text
70,843 bytes
5C7F9787268CACAAD33D79FB904EA9A69F221EF256252E76DBAC01D7E49D108A
```

The eight accepted Find keyboard source/test files retained their frozen
identities:

| file | bytes | SHA-256 |
|---|---:|---|
| `native/app/app.go` | 33,365 | `F57743A4CAD59C95B3FB13D8B8C982C99CBFE43B979ED7EB447ACD51B53C6AC3` |
| `native/app/search_navigation.go` | 9,759 | `57D56DA4B266E9381D8F2268A26D8CFC33ABA4B968937D3AB753F32ED4763E77` |
| `native/app/state.go` | 11,057 | `63BBC1372A26D862D8B69D755E50FC7880C6474C36121F4A0A637DDA852E8D38` |
| `native/app/source_navigation.go` | 8,175 | `49FA79D5D3FC7B38B46363874780E421E3DABE363F6C28F1C3FAE78B12806113` |
| `native/app/view.go` | 20,768 | `CA2D62B0F873A86FADB92DBA525BE09E25DA04F44A7969D6D9D44DA532EFD640` |
| `native/app/find_keyboard_test.go` | 21,741 | `1C9D0408DDE6899E134351763E9373D61E7A0537E8B150A1EB2CB2AEC64FAD81` |
| `native/app/find_keyboard_lifecycle_test.go` | 18,867 | `04C55F900DA46173EA1A757B808B1106C78F0196F295948E6CFD6B94A284D680` |
| `native/ui/editor/editor.go` | 19,929 | `6438B438F0409BEEB26A9427C5CC8EBF971132242933B1D927BEBFEA4371AC81` |

The complete accepted `.13` package remained 23 files / 23,655,494 bytes:

```text
A9818DB81B64978EA912BABF2E1FA85CAFBBF1C9CEBD4329756355FFDCCBE495  full inventory identity
4B86002034A493789A1747E343542FD5A94BCD9DCEB5060662A5DD55DCF4150D  levnote-native.exe
0F1825348BD3EEFDB8264079405DBA23019692D22EA7962098B3A4FC4470D5FB  release-manifest.json
```

Its complete 102-file / 47,615,997-byte delivery evidence also remained
unchanged:

```text
834F975246D0C9A15A8D69575365681FFCCB280D5B728D7422A6BC78A54A2F4A  full evidence inventory identity
8604FFD235EF95481E47925A6000C6AE697CA50457726BE74B77DB0BCA9BE343  accepted aggregate.json
```

The accepted keyboard result, root acceptance, historical independent review,
final reconciliation, red/green correction logs, final suite/vet logs, `.13`
acceptance/delivery records and the two historical `.13` layout captures were
all retained. `accepted-gates.json` is 3,160 bytes SHA-256
`CA417E38B0790B88B3E985DD8434A5AD287206E9CE654E932520DCB6BFFB32E0`.
No new capture was made because accepted visual markup was unchanged.

## Hidden host smoke

One hidden smoke launched an exact 23-file copy of the `.14` bundle from `F:`,
with fresh run-owned configuration on the normal `C:` temp volume and a
generated 60-byte Markdown source:

```text
PID:                       17036
HeldProcessHandle:         2288
Start UTC:                 2026-11-09T13:58:43.2301207Z
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

Only the held process identity and its owned window were used; there was no
process-name cleanup, helper retry or second host launch. The source retained
its 60 bytes and SHA-256
`7CBEA90B48A026B9B0704F5F37F4AF7FDBB912BB15ABD6A4FC421D9003EFB6A1`.
The copied bundle and candidate were preserved. The recovered session contained
exactly one UTF-8 tab without BOM or draft:

```text
421 bytes
1957AE8D2AA6CA68C014FBE16E82EF8AC9A05D479F5B0640919AAE1EBCA723E4  session-after-close.json
```

Application stdout and stderr were empty. The smoke operation exited 0 in
4.0850289 seconds; its outer stage exited 0 in 5.8768702 seconds.

The accepted `.11` managed real old-release/bundled-reset protocol execution
evidence remains historical integration evidence. It was not rerun for `.14`.

## Aggregate

The final validator and aggregate passed **12/12 without retry**. They checked
the exact package and evidence copy, manifest payload hashes, four GUI PE
images, three engine manifests/files, 13 portable assets, the complete `.13`
package and its complete delivery evidence, scoped source/build-input
preservation, eight frozen keyboard files, accepted results/root gates/captures,
and the one hidden host smoke.

```text
Validator: 10,085 bytes
92E1B7AEF03A99CF082E91740595A80E608D88771514AC7E4037A864EA75786E  validator.json

Aggregate: 17,036 bytes
66242F93DC7948DA319438A165EEA433AB4C2E2CD9CA5FC3EA6237A1AE05C765  aggregate.json
```

The outer aggregate stage exited 0 in 1.9010241 seconds.

## Evidence

Evidence root:

```text
F:\AIproject\LevNote\native\artifacts\acceptance\find-keyboard-release14-delivery-20261109-215637
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
normal close. It does not execute `.14` workers or repeat managed runtime tests.
The accepted synthetic Find keyboard tests establish application/keyboard
lifecycle behavior; this delivery does not verify physical Windows IME
candidate or Escape behavior, visible UI automation, or any CPU, GPU, RSS,
working-set or other resource result. The two `.13` layout captures remain
historical visual evidence.
