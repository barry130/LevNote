# Native worker .12 source-navigation delivery

2026-11-09. The finite `.12` delivery completed and is accepted as the current
runnable native package in
[source-navigation-release12-root-acceptance.md](source-navigation-release12-root-acceptance.md).
One existing package-script call produced the candidate, one byte-identical
copied host completed a hidden owned startup/close smoke, and the final scoped
validator passed 11/11. `.11` remains unchanged.

## Candidate

Directory:

```text
F:\AIproject\LevNote\native\bin\levnote-native-worker-20261109.12
```

Version: `native-worker.20261109.12`

The package contains 23 files totaling 23,645,766 bytes.

Host:

```text
14,022,144 bytes
43FC7E86004DC3602F6AA367B6BC55007A1AE28614484E5F7DD3BADF0A782768  levnote-native.exe
```

Release manifest:

```text
4,137 bytes
9AC85B63DE204FE9B3FE4CEEE865838FAA04B7B810213374727987EF50F0C445  release-manifest.json
```

The manifest has 22 payload entries plus itself. The validator confirmed all
four executables are Windows amd64 GUI PE images and that all 13 portable
source assets retain their copied identities.

The three `.12` engine records independently matched their executable files
and manifests:

| engine | worker bytes | worker SHA-256 | manifest SHA-256 |
|---|---:|---|---|
| `levnote.markdown-export` | 3,300,352 | `C5AB6A00128D8D596E57F796B78D8E68F14DCAECA1B2CD87E8EF1AEEEFAD1BD3` | `7A8BA0E5BD272B47D5B72A904BDDD12040353A86CFC0928E81AD2FD06CEFE6AE` |
| `levnote.mindmap-svg` | 3,122,688 | `31B50BB0B61B8ED99CAC47DC759FE7DB4EEB01725FFC5EB28AC981BF58D729ED` | `FC6846BF1CE2C652BF2091196EB5A89F91E3D62E8072F7576D6BA3C0A3AAA197` |
| `levnote.mindmap-preview` | 3,136,512 | `44B7033A63F7C745B8B5EFAFA242139309D06CEAEB4ADDBB563C033D2B8C20A6` | `79DA01B129D48A090606FB2D7E859E8249CB791D395CEE3CFE07F80F14E670F2` |

## Commands and package build

The candidate directory was absent at baseline. Delivery stages used the
absolute PowerShell 7 runtime with `-NoProfile` and captured stdout/stderr:

```text
pwsh -NoProfile -File ...\invoke-stage.ps1 -Name baseline -ScriptPath ...\baseline.ps1
pwsh -NoProfile -File ...\invoke-stage.ps1 -Name package -ScriptPath ...\run-package.ps1
pwsh -NoProfile -File ...\invoke-stage.ps1 -Name host-smoke -ScriptPath ...\run-host-smoke.ps1
pwsh -NoProfile -File ...\invoke-stage.ps1 -Name aggregate -ScriptPath ...\aggregate.ps1
```

Baseline exited 0 in 1.5000610 seconds with empty stderr.

The package stage made exactly one call to:

```text
native\scripts\package-markdown-worker.ps1
  -OutputDirectory F:\AIproject\LevNote\native\bin\levnote-native-worker-20261109.12
  -EvidenceDirectory F:\AIproject\LevNote\native\artifacts\acceptance\source-navigation-release12-delivery-20261109-201756\package
  -Version native-worker.20261109.12
```

That call exited 0 in 8.9392441 seconds with empty stderr. The outer package
stage exited 0 in 10.4214083 seconds with empty stderr. The package result says:

```text
PACKAGED; build and asset checks passed, product acceptance is separate
```

Exactly four hidden wrapper GUI builds exited 0 with empty stdout and stderr:

| build | wall seconds |
|---|---:|
| `worker-markdown` | 1.4672781 |
| `worker-mindmap` | 1.3689200 |
| `worker-mindmap-preview` | 1.2570794 |
| `host` | 3.2578835 |

## Preservation

The scoped source/build-input inventory contained 288 files / 2,880,940 bytes,
including 60 files under `native/formats` and the existing
`native/portable/NOTICE-PREAMBLE.md`. `inputs-before.json` and
`inputs-after.json` were byte-identical:

```text
9619ED8D7ADC7B52962CB6BDE4C40D42ACABAE35C1068D281F87CEF437B481AC
```

The exact eight accepted navigation files matched their frozen SHA-256 and byte
counts before and after delivery. The two accepted captures also retained their
recorded identities:

```text
2932E7800A6C9062636DDDA1180F4A876ACBCDA1B7769F7F413427B614F0229A  source-navigation-error-760x480-light.png
17E75F906CF7B4C54F6D1708E46EF9EBEE714B576B2FC84799F542D361960D7B  source-navigation-error-760x480-dark.png
```

The complete accepted `.11` inventory remained 23 files / 23,622,214 bytes:

```text
47AA937D0BC239E82BF029DC2D0367C94F841F7C9AEFA14C7CCB0CCC3FE4D3C4  full .11 inventory identity
F1D1FC76AC1A5649709B6AD8FDB44A1928174280F766625F7226856493E299B1  .11 levnote-native.exe
A4E37C4E65B38C7B46ACB1FBC80455B604EE4207E7D3072580BD37DE0103E3E6  .11 release-manifest.json
```

No product source, test, build script, product document, `.11` file or earlier
evidence directory was edited.

## Hidden host smoke

One host start used a byte-identical 23-file copy of `.12` under the evidence
directory. It was launched from `F:` with
`Start-Process -WindowStyle Hidden`, while isolated run-owned
`APPDATA`/`LOCALAPPDATA` stayed under the normal `C:` temp volume.

Observed result:

```text
AppStarts:                 1
PID:                       29644
HeldProcessHandle:         2396
WindowVisible:             false
OwnedWMClosePosted:        true
ForcedAppCleanup:          false
AppExitCode:               0
OwnedAppStillLive:         false
RequestedWorkerOperations: 0
```

Only the exact held process handle and exact PID-owned window were used. There
was no process-name scan or kill, no second app PID and no helper retry.

The unchanged 60-byte sample remained:

```text
7CBEA90B48A026B9B0704F5F37F4AF7FDBB912BB15ABD6A4FC421D9003EFB6A1
```

The copied bundle was unchanged. Normal close retained one isolated session tab
for that source with no draft; `session-after-close.json` is 425 bytes with
SHA-256
`E25931FBEEB8553D89C439C84298997EC93CDE13CE56004ECE934A6364A67C86`.
App stdout and stderr were empty.

## Aggregate

The final validator and aggregate pass 11/11. They check:

- one package call and four GUI builds;
- the exact fresh `.12` target, 23-file inventory and evidence copy;
- manifest entries, sizes, hashes and four GUI PE facts;
- all three `.12` engine version/hash/manifests;
- all portable source-asset identities;
- the complete unchanged `.11` inventory;
- unchanged scoped inputs including `native/formats`;
- the exact eight frozen navigation source/test files;
- the accepted navigation result, acceptance and two captures;
- the one hidden owned host startup, session/source/bundle preservation and
  normal close.

The first aggregate attempt exited 1 in 1.7159136 seconds because the
evidence-only build-name list said `worker-markdown-preview` instead of the
package script's actual `worker-mindmap-preview`. Its raw records are preserved
as `stage-aggregate-attempt01.*`. After changing only that filename, the final
aggregate exited 0 in 4.7928265 seconds. No package build or host start was
replayed.

```text
Validator SHA-256: 6FC92787461EE3F89EC11EA73161ABAC859E19DD2282FF52BE39C8CF32E006F3
Aggregate SHA-256: E6A950BA60C5483BA3478740625EE08344EE7373F3CEBB687D2EDA2F39000D34
```

This run compiled and ran zero managed runtime testers and requested zero
production worker operations. The accepted `.11` managed evidence remains the
inherited engine/protocol integration record and is not presented as `.12`
runtime execution. The frozen synthetic source-navigation tests and captures
establish the feature's tested behavior; the hidden host smoke establishes only
normal startup and owned close.

## Evidence

Evidence root:

```text
F:\AIproject\LevNote\native\artifacts\acceptance\source-navigation-release12-delivery-20261109-201756
```

Start with:

```text
validator.json
aggregate.json
baseline-result.json
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
stage-aggregate-attempt01.stderr.log
```

## Limits

This evidence covers package construction, source preservation and one hidden
owned-host startup/close. It does not include `.12` managed worker execution,
physical Ctrl+G/IME/OS candidate-window automation, visible UI interaction,
GPU, CPU, RSS, working-set or other resource measurement, and it makes no such
claim.
