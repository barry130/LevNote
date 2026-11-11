# Search navigation and Find layout: .13 delivery

2026-11-09. Root accepted this delivery as the current runnable package in
[search-navigation-release13-root-acceptance.md](search-navigation-release13-root-acceptance.md).
The candidate facts, checks and limits below remain the frozen evidence record;
`.12` and all earlier releases remain immutable history.

## Candidate

Directory:

```text
F:\AIproject\LevNote\native\bin\levnote-native-worker-20261109.13
```

Version: `native-worker.20261109.13`

The candidate and its evidence copy each contain 23 files totaling 23,655,494
bytes and are byte-identical.

Host:

```text
14,031,872 bytes
4B86002034A493789A1747E343542FD5A94BCD9DCEB5060662A5DD55DCF4150D  levnote-native.exe
```

Release manifest:

```text
4,137 bytes, 22 payload entries plus itself
0F1825348BD3EEFDB8264079405DBA23019692D22EA7962098B3A4FC4470D5FB  release-manifest.json
```

The three engine records matched their package files and manifests:

| engine | worker bytes | worker SHA-256 | manifest SHA-256 |
|---|---:|---|---|
| `levnote.markdown-export` | 3,300,352 | `F8BFC4527D5A64DCA661175DD0D7C825DF5388CA3CBBA648614722B0AE8D98C5` | `C4F171A31557F712AB8757FFE3A0B4037DDFFFDC8027454ADCBD3E3F019703B8` |
| `levnote.mindmap-svg` | 3,122,688 | `7F470C7A5D49C772B384CF2B8EA6AAC0F304A3C6A0DE715042B66FF0B306E1FA` | `2FD9BEFD6877BC1E6C96272962427FBB59F4C60F311847583E8026D6DD14169E` |
| `levnote.mindmap-preview` | 3,136,512 | `791B56B5C3A02ABFAA262DF191F2E9BC1BA4E23CAC6D4E0770B6CF654886ED2F` | `544129040C006DA20C891525209EB6E08B7A7CEDE6B137C8FC4C8AFB4B9FAF03` |

## Package and builds

The delivery used one existing package-script invocation:

```text
native\scripts\package-markdown-worker.ps1
  -OutputDirectory F:\AIproject\LevNote\native\bin\levnote-native-worker-20261109.13
  -EvidenceDirectory F:\AIproject\LevNote\native\artifacts\acceptance\search-navigation-release13-delivery-20261109-210329\package
  -Version native-worker.20261109.13
```

The invocation exited 0 in 6.7232141 seconds. Its outer package stage exited 0
in 7.8768647 seconds. It made exactly four hidden Windows amd64 GUI builds:

| build | wall seconds |
|---|---:|
| `worker-markdown` | 1.1742651 |
| `worker-mindmap` | 0.9678526 |
| `worker-mindmap-preview` | 1.2294546 |
| `host` | 1.7505707 |

All four builds exited 0 with empty stdout and stderr. No managed runtime tester
or production worker request was run.

## Preservation and accepted evidence

The scoped source/build-input inventory contained 291 files / 2,912,652 bytes,
including 60 `native/formats` files and the referenced
`native/portable/NOTICE-PREAMBLE.md`. `inputs-before.json` and
`inputs-after.json` are identical:

```text
70,362 bytes
8B82C9BDA899C800F016778AB4FF4D38E11FD2FC28BA3B14128128F8567E2597
```

The eight frozen search/layout sources and tests retained their exact identities:

| file | bytes | SHA-256 |
|---|---:|---|
| `app.go` | 32,768 | `5017BCC975F396EFD966D0D76159BC8BF0114671ED06B5E3657C95A184C06984` |
| `plugins.go` | 10,756 | `E64F408A4EFFBA065923B73136A6515BCAFA295468199EF0A9A625E3325E87C6` |
| `state.go` | 7,475 | `59E1DF87541AC786D2CE5603AD10CBCE14CF4B1D9775358B1241E66FAB461D7E` |
| `view.go` | 20,547 | `5501670B56CD876D9E1252B03E31F434DC9DC9E65B9A6C13212AC5D4E2404125` |
| `workspace.go` | 6,594 | `F7898A151F2850FA34C78669E2265C2DE8951BC77C805601F93ED20B306D5A2A` |
| `search_navigation.go` | 4,992 | `8F77367B137135618CEB65BE0513ED5670033E2500A867C87DD8185CB5854E6D` |
| `search_navigation_test.go` | 20,784 | `CEF83EA00713A9896C6FCD117BB7A710BFD184FB788235119121F9C99CF2F79D` |
| `findbar_layout_test.go` | 6,509 | `F20B0A9F777D8DE80632092DB3A1BE5FAEEF5E32AF392FF9DF03896C82193D1C` |

The two accepted Find-panel captures also remained unchanged:

```text
1860EB9E8E96F8F1EA51052CC907AA1D3D188B73122C565969D9045C8EA599CB  findbar-light-760x480.png
BF139E982E55183C515FA0965033643B748C611598ACDF5B28A89F3CA6DAA2F3  findbar-dark-760x480.png
```

The complete accepted `.12` package remained 23 files / 23,645,766 bytes:

```text
54D598FD6E8FAB321614B76DE54D40DBEDE7CC044D74965A8D7699DAF6A652F3  full .12 inventory identity
43FC7E86004DC3602F6AA367B6BC55007A1AE28614484E5F7DD3BADF0A782768  levnote-native.exe
9AC85B63DE204FE9B3FE4CEEE865838FAA04B7B810213374727987EF50F0C445  release-manifest.json
```

The protected inventory remained identical at 7,759 bytes with SHA-256
`81BBA328E7F9F3A48D46EE932B07B268CFE2BB5A474177839B0740CB56F36C61`.
No source, test, build script, product documentation, `.12` file or earlier
evidence was edited.

## Hidden host smoke

One hidden smoke launched an exact 23-file copy of the `.13` bundle from `F:`,
with fresh run-owned configuration on `C:` and a generated Markdown fixture.

```text
PID:                       38700
HeldProcessHandle:         2196
Start UTC:                 2026-11-09T13:05:28.1686788Z
WindowVisible:             false
OwnedWMClosePosted:        true
AppExitCode:               0
ForcedAppCleanup:          false
OwnedAppStillLive:         false
RequestedWorkerOperations: 0
```

Only the held process identity and its owned window were used. There was no
process-name sweep, forced cleanup or second host launch.

The 60-byte source was retained with SHA-256
`7CBEA90B48A026B9B0704F5F37F4AF7FDBB912BB15ABD6A4FC421D9003EFB6A1`.
The copied bundle and candidate were preserved. The recovered session contained
exactly one tab and no draft:

```text
425 bytes
224C70518D42B283CE40E631019BE10C4CB21C59EF39E0EDD47EAA03C05EA2B3  session-after-close.json
```

Application stdout and stderr were empty. The outer host stage exited 0 in
5.5464364 seconds; the smoke operation exited 0 in 3.7847295 seconds.

## Aggregate

The final validator and aggregate passed **11/11 without retry**. They checked
the exact package and evidence copy, manifest payload hashes, four GUI PE
images, three engine manifests and files, 13 portable assets, the complete `.12`
package, scoped source/build-input preservation, eight frozen files, accepted
results/acceptances/captures, and the one hidden host smoke.

```text
Validator: 9,509 bytes
BC4EA3DEFBDAA7E67A5406D4A779ECBE8E8E6518362B6A72909CCF5F1A5EFFC0E  validator.json

Aggregate: 16,189 bytes
8604FFD235EF95481E47925A6000C6AE697CA50457726BE74B77DB0BCA9BE343  aggregate.json
```

The outer aggregate stage exited 0 in 1.2835339 seconds.

## Evidence

Evidence root:

```text
F:\AIproject\LevNote\native\artifacts\acceptance\search-navigation-release13-delivery-20261109-210329
```

Start with:

```text
validator.json
aggregate.json
inputs-before.json
inputs-after.json
protected-before.json
protected-after.json
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
normal close. It does not execute `.13` workers or managed runtime tests, and
does not verify visible Find/search actions, physical keyboard/IME behavior or
any CPU, GPU, RSS, working-set or other resource-results. The inherited `.11`
managed protocol evidence remains historical integration evidence and is not a
`.13` runtime claim.
