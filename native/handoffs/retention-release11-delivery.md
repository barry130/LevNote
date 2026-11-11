# Native worker .11 retention-fix delivery

2026-11-09. The finite `.11` delivery completed with all 13 aggregate checks
passing and is accepted as the current runnable native package in
[retention-release11-root-acceptance.md](retention-release11-root-acceptance.md).
`.10` remains an immutable historical accepted release.

## Accepted release

Directory:

```text
F:\AIproject\LevNote\native\bin\levnote-native-worker-20261109.11
```

Version: `native-worker.20261109.11`

The package contains 23 files totaling 23,622,214 bytes.

Host:

```text
13,998,592 bytes
F1D1FC76AC1A5649709B6AD8FDB44A1928174280F766625F7226856493E299B1  levnote-native.exe
```

Release manifest:

```text
4,137 bytes
A4E37C4E65B38C7B46ACB1FBC80455B604EE4207E7D3072580BD37DE0103E3E6  release-manifest.json
```

The manifest has 22 payload entries plus itself. The aggregate verification
confirmed four Windows amd64 GUI PE executables, three fixed engine manifests,
and all 13 portable source assets byte for byte.

## Frozen source

The combined dependent check published by the text owner passed before the
package build:

```text
ok  levnote.local/native/core/document  0.713s
ok  levnote.local/native/ui/editor      2.395s
ok  levnote.local/native/app            15.388s
exit 0
```

The four reviewed retention files matched the frozen identities before and
after delivery:

```text
562F1293800CF147FE466F1B64E29FA5627591F1E8CCC698FEAE9163534E2E9A  native/core/text/buffer.go
8AE4894024E97B135FD49D32774AFAD7EDCA4CE01A87E7CF62101AD67F62A5A2  native/core/text/retention_test.go
6387CDF9938D3546C26CD8D387CA4B334A011A2A47456CBCB300E0AA5890527A  native/app/app.go
B0CA0D1E70A2C58F3F81EA943081550F6B0D3DF368A25B702E0562BCE241C316  native/app/tab_retention_test.go
```

The build-source scope contains 286 files / 2,843,545 bytes. Its before and
after inventories were identical, including the same inventory SHA-256
`2BA382AA03FD9EA3001D9E035FFA47E6E2912273318163927FAB03D19D890468`.
No product source was edited during delivery.

## Package build

The existing package script was invoked exactly once. It completed with exit
0, empty stderr, and wall time 8.536497 seconds. The four required GUI builds
used `native/scripts/go.ps1` and each exited 0 with empty stdout and stderr:

| Build | Wall seconds |
|---|---:|
| `worker-markdown` | 1.2257154 |
| `worker-mindmap` | 1.0986789 |
| `worker-mindmap-preview` | 1.1075904 |
| `host` | 2.994493 |

The package result status is:

```text
PACKAGED; build and asset checks passed, product acceptance is separate
```

## Managed integration

The existing runtime tester was compiled once into a 23-file evidence-owned
copy of `.11`; the tester executable made the post-compile bundle 24 files /
30,840,390 bytes. Compile exit was 0 in 2.3238694 seconds. Runtime invocation
used exactly:

```text
-test.run ^TestManagedRealOldReleaseThenBundledResetSmoke$
-test.v
-test.count=1
```

The named test passed with exit 0 in 2.2877238 seconds. It made exactly two
production Markdown worker requests:

1. Import and activate the accepted `.10` package, then export.
2. Explicitly Reset to bundled `.11`, then export.

Both outputs were 104 bytes with independent-oracle SHA-256
`681487B9A59F3D162BC719ED242B2242AB4A5B16CCD61C5A945D15EEAEDC50A0`.
The 20-byte source remained `282788C7BCE41070F5F0391BBC70AE69B62F9926ECC071100881DDE0D0450A5A`.
The old and new Markdown worker identities were
`56A5BB8CF883B4336B2AFD5C1085491E2FE92F265596F04A2A321CFFC0D28536`
and
`1B68EA5F3A0735BF1D4A65755874CB6A623FC2076D865B5A90184F3317DD53D6`.
Both workers called their normal close path, with no fallback close or close
error. The managed record status was PASS and its `Failures` value was null.

## Hidden host smoke

One app start used the existing held-handle, owned-window `WM_CLOSE` method.
It executed a byte-identical 23-file copy of `.11`, launched with
`Start-Process -WindowStyle Hidden`, kept the executable on `F:` and used an
isolated run-owned `APPDATA` / `LOCALAPPDATA` under the normal `C:` temp
volume. The isolated config root was retained for evidence.

Observed result:

```text
AppStarts:              1
WindowVisible:          false
GracefulWMClosePosted:  true
GracefulCloseTimedOut:  false
ForcedAppCleanup:       false
AppExitCode:            0
OwnedAppStillLive:      false
RequestedWorkerOperations: 0
```

The unchanged Markdown source stayed at 60 bytes,
`7CBEA90B48A026B9B0704F5F37F4AF7FDBB912BB15ABD6A4FC421D9003EFB6A1`,
with the same last-write ticks. The persisted `session.json` is 417 bytes,
SHA-256
`20050A68832F559159910BE058B01743096658783BB784AEBF9A61E33BDA7F74`,
and names the one opened tab with no draft. App stdout and stderr were empty.

## Aggregate verification

`validator.json` and `aggregate.json` report PASS 13/13. The checks cover:

- one package invocation and four wrapper GUI builds;
- the exact `.11` target/version and clean 23-file inventory;
- manifest entries, bytes, hashes, PE facts and engine manifests;
- all portable source asset hashes;
- the accepted `.10` inventory and identities being unchanged;
- the 286-file build-source inventory being unchanged;
- the four frozen retention source hashes;
- the exactly two-request managed integration;
- the one hidden host start, source/session preservation and owned close.

Accepted `.10` remained exactly 23 files / 23,622,214 bytes:

```text
4EB096FAEA7CA79E71F0C9FF30E659478C0E2AA135803DDBF7C0AEE47A0AEF6C  .10/levnote-native.exe
38D79797ABA4EDFB0DD12E3ED78476D8CC956D1A3069921E447690E842A0670D  .10/release-manifest.json
```

## Evidence

Evidence root:

```text
F:\AIproject\LevNote\native\artifacts\acceptance\retention-release11-delivery-20261109-193512
```

Start with:

```text
validator.json
aggregate.json
inputs-before.json
inputs-after.json
protected-before.json
protected-after.json
package\package-result.json
package\*-build.json
managed-real\compile-command.json
managed-real\run-command.json
managed-real\run.stdout.log
managed-real\record.json
managed-real\status.json
host-smoke\result.json
host-smoke\session-after-close.json
```

## Limits

This evidence covers package construction, the two managed worker identities,
and one hidden owned-host lifecycle. It does not include physical UI input,
IME, GPU, CPU, RSS or working-set/resource-efficiency measurement, and it makes
no such claim.

The four retained 760x480 tester screenshots are historical `.10` captures.
No `.11` recapture was performed, and they are not evidence that `.11` was
rendered through the physical UI.
