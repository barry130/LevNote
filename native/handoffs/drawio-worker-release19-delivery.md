# Fourth worker `.19` delivery

Status: **CANDIDATE READY FOR ROOT BINARY ACCEPTANCE** 2026-11-10.
`native-worker.20261110.19` has passed the corrected delivery sequence: one
package invocation, five Windows GUI builds, one real managed draw.io
fourth-worker request, one separate hidden host lifecycle, and a 12/12
aggregate validation. `.18` remains the accepted package until root accepts
this actual binary and updates product documentation.

No product source, test, package script, asset, candidate, or evidence file was
edited after execution. This handoff is the only delivery artifact written
after the aggregate.

## Candidate and package

`native/bin/levnote-native-worker-20261110.19` contains **25 files /
26,852,621 bytes**. Its complete inventory identity, including relative path,
byte count, SHA-256 and modification ticks, is
`7925942AFDE49F63C2D77D79EBE34FD3F1BAD18F535745E30171AFA3CD27A24B`.

| Item | Bytes | SHA-256 |
|---|---:|---|
| `levnote-native.exe` | 14,094,848 | `BB845BF6FDE47D16804AEFA033FBAF5B85A507FF71147E06F467A636E9F61CE2` |
| `release-manifest.json` | 4,490 | `523C2BF682C9E9527FDDF972BC3CEE2C93D22E49114CA8FB52A2DF5D4C4ABFA4` |

The one package-script invocation exited 0. Every build exited 0 with zero
stdout and stderr bytes:

| Build | Payload bytes | Wall time | Payload SHA-256 |
|---|---:|---:|---|
| markdown export | 3,303,424 | 1.380 s | `F70B1FAD4817ED64E179139CD2CFEEDDE06D15AFDEF9E93F01D4FD5155C95CCF` |
| mindmap SVG | 3,126,272 | 1.096 s | `97CDBD2060F1609A935BFD59A73954A37871A563B0989AEDD6D4757EC33B756C` |
| mindmap preview | 3,140,608 | 0.954 s | `8DE0E2D25559FA9FE491B4A70E676E0CDC7C0121B095F753E796EC47BA1625BE` |
| draw.io SVG | 3,121,664 | 1.286 s | `DF646F752E39D42ACD154F16B80D04D7636978895E3CD36F06E70E50A55EB319` |
| host | 14,094,848 | 2.728 s | `BB845BF6FDE47D16804AEFA033FBAF5B85A507FF71147E06F467A636E9F61CE2` |

The complete package copy is byte-identical at 25 files / 26,852,621 bytes.
The schema-1 release manifest names the `.19` version and all 24 non-manifest
payload entries match their exact sizes and hashes. The four fixed engine
manifests carry the same version and match their worker payloads. All five
executables are Windows amd64 PE32+ GUI images (`machine=0x8664`,
`PE32+=0x20b`, `subsystem=2`). All 13 portable source-copy assets retain their
source hashes, and the package's local authored links were checked.

Raw package records:

- [package invocation](../artifacts/acceptance/drawio-worker-release19-20261110-023148/package-command.json)
- [package result](../artifacts/acceptance/drawio-worker-release19-20261110-023148/package-result.json)
- [inner package result](../artifacts/acceptance/drawio-worker-release19-20261110-023148/package/package-result.json)
- [package stdout](../artifacts/acceptance/drawio-worker-release19-20261110-023148/stage-package.stdout.log)

## Actual fourth-worker smoke

The reviewed fixture was compiled once and then run once against a complete
evidence-owned copy of the `.19` bundle. The tester was placed in that copy, so
the copy grew from 25 to 26 files; the accepted candidate stayed at 25 files.
The fixture source remained 21,150 bytes with the pre/post SHA-256
`074C4894D1D05F0FEBE7B49F61BF958980BDDF6DFEF9151179A18C05A2FF4548`.

The run loaded an isolated legacy schema-1 state with the old three selections,
confirmed the fourth bundled engine without implicit persistence, imported and
explicitly activated the `.19` draw.io engine, and issued exactly one
`NewManaged` / `Runner.Export` request. The request selected zero-based page
index `1`, page ID `second`; no other engine ran.

- Test: `^TestDrawioManagedRealBundleSmoke$`
- Compile / run exits: `0` / `0`
- `NewManaged`, export, admission, verify, start and wait calls: `1` each
- Payload: 598 bytes, SHA-256
  `3f990a4aac3209bafad94f44026bf14f3cec27ea44bdbd3169555947936472bf`
- Warnings: two exact ordered `UNSUPPORTED_ELEMENT` entries, SHA-256
  `5CA8CA9007E0154EB8E7E37DF132987115FADB245F1239D8EC982F00CBD48A5D`
- Worker PID / held handle: `23728` / `984`
- Creation UTC: `2026-11-09T18:32:46.8381216Z`
- Exact imported image path, start, wait and normal exit `0`: passed
- Joined lifecycle: primary close succeeded; fallback close was not attempted

Raw worker records:

- [managed result](../artifacts/acceptance/drawio-worker-release19-20261110-023148/managed-smoke/result.json)
- [structured record](../artifacts/acceptance/drawio-worker-release19-20261110-023148/managed-smoke/record.json)
- [compile invocation](../artifacts/acceptance/drawio-worker-release19-20261110-023148/managed-smoke/compile-invocation.json)
- [run invocation](../artifacts/acceptance/drawio-worker-release19-20261110-023148/managed-smoke/run-invocation.json)
- [run stdout](../artifacts/acceptance/drawio-worker-release19-20261110-023148/managed-smoke/run.stdout.log)

This proves one installed fourth-worker request and its joined managed process
lifecycle. Broader draw.io parser, renderer, manifest, and failure behavior is
established by the previously accepted source tests and Stage 1/2 evidence;
it was not rerun in this delivery.

## Hidden host lifecycle

The separate host smoke copied the complete `.19` bundle, started that exact
copy once with `Start-Process -WindowStyle Hidden`, and verified its path through
the held process handle. It used an isolated run-owned APPDATA/LOCALAPPDATA
root on the normal C: temp volume and one unchanged 60-byte generated Markdown
source.

- Start count / PID / held handle: `1` / `3712` / `2296`
- Owned `WM_CLOSE` posts: `1`
- Normal exit: `0`; no timeout or forced cleanup
- Worker operations requested: `0`
- Source SHA-256 before and after:
  `7CBEA90B48A026B9B0704F5F37F4AF7FDBB912BB15ABD6A4FC421D9003EFB6A1`
- Clean one-tab session: 412 bytes, SHA-256
  `8CDBFEEB79E7BBBECF8477834F4773100972A385A80BA1409F6E219C26AF4C6A`
- App stdout and stderr: zero bytes
- Candidate and copied bundle remained unchanged

Raw host records:

- [host result](../artifacts/acceptance/drawio-worker-release19-20261110-023148/host-smoke/result.json)
- [session after close](../artifacts/acceptance/drawio-worker-release19-20261110-023148/host-smoke/session-after-close.json)
- [host outer invocation](../artifacts/acceptance/drawio-worker-release19-20261110-023148/outer-host-smoke-invocation.json)

This run establishes hidden startup and clean close only. It does not establish
physical UI or OS IME behavior, dirty Save/recovery, packaged interactive SVG
behavior, process memory, CPU use, or a resource improvement.

## Execution and aggregate

The successful execution used the corrected prepared harness copied into the
fresh evidence directory. Initialization reported `AUTHORIZED`, 12 final
Stage 2 files, 314 scoped source inputs, 9 protected evidence directories and
20 protected gate files. The stages were then run sequentially:

```text
./run-stage.ps1 -Name baseline
./run-stage.ps1 -Name package
./run-stage.ps1 -Name managed-smoke
./run-stage.ps1 -Name host-smoke
./run-stage.ps1 -Name aggregate
```

| Stage | Outer seconds | Stage seconds | Exit | Stage stderr |
|---|---:|---:|---:|---:|
| baseline | 1.828 | 1.205 | 0 | 0 bytes |
| package | 11.473 | 10.887 | 0 | 0 bytes |
| managed-smoke | 5.748 | 4.762 | 0 | 0 bytes |
| host-smoke | 6.393 | 5.122 | 0 | 0 bytes |
| aggregate | 2.744 | 1.800 | 0 | 0 bytes |

The aggregate validator passed **12/12**, exit 0:

- [validator.json](../artifacts/acceptance/drawio-worker-release19-20261110-023148/validator.json),
  5,616 bytes, SHA-256
  `2FDF7329E289719ABCC14361A916E0E95442D3BB52E98CF05B057C409611AFE5`
- [aggregate.json](../artifacts/acceptance/drawio-worker-release19-20261110-023148/aggregate.json),
  28,725 bytes, SHA-256
  `91DB69CDA656C177A925178EC30239D693166DFB0BD722BF79B587ED1D10B2BC`
- [commands.json](../artifacts/acceptance/drawio-worker-release19-20261110-023148/commands.json)
  records one package invocation, one managed tester compile, one managed
  tester run and one hidden host start

The aggregate confirmed:

- all five builds and the complete 25-file candidate/manifest;
- four engine manifests, five GUI PE images, 13 copied assets and local links;
- the complete accepted `.18` package remained 23 files / 23,699,526 bytes,
  inventory identity
  `4A4F9BD0B6DF2DDCD1E597F026AAC985A9D50711CDD85F3E9310F7DC5B0B2991`;
- the `.18` direct delivery evidence remained 117 files / 47,816,959 bytes,
  inventory identity
  `8D73702A59E34583FB77E567AF7167A16A06698F751A16D623BDD469EE66C948`;
- 314 scoped source/build inputs / 3,133,071 bytes were unchanged, including
  64 `native/formats` files and portable `NOTICE-PREAMBLE.md`; the identical
  before/after inventory record hashes to
  `0807204CD56DA2059E06AA7309FEE4FF9773D71CE67686D80DB0DC2A84170900`;
- all 12 final Stage 2 files matched their accepted manifest;
- all 9 accepted evidence directories and 20 accepted gate files were
  unchanged;
- both the managed fourth-worker request and separate hidden host conditions
  passed.

The completed `.19` evidence directory contains **195 files / 91,652,331
bytes**, complete inventory identity
`4CFA46B862E2798CCEB0A49645E1096044465C9D77AC291509D072C48523D7F4`.

## Preserved initialization failure

The first initialization wrapper used Windows PowerShell 5.1 and exited 1
without starting any product operation because that runtime lacks
`System.IO.Path.IsPathFullyQualified`. This was an evidence-metadata/runtime
selection failure before candidate creation, not a product failure. The
five-file failure record was preserved at
`native/artifacts/acceptance/drawio-worker-release19-init-failure-20261110-023053`,
including [failure.txt](../artifacts/acceptance/drawio-worker-release19-init-failure-20261110-023053/failure.txt)
and [initialize.stderr.log](../artifacts/acceptance/drawio-worker-release19-init-failure-20261110-023053/initialize.stderr.log).
The successful rerun used the harness's intended PowerShell 7 `pwsh` runtime
and the fresh directory listed above.

## Scope

`.19` contains the host plus four fixed workers and adds one actual bundled
draw.io fourth-worker request. The existing markdown and two mindmap engines
were built and package-validated; their broader behavioral evidence is
inherited from earlier accepted source/Stage evidence rather than rerun here.
The separate hidden host run proves only startup and a clean owned close.

No physical UI/IME automation, dirty Save/recovery, packaged interactive SVG
run, process-memory or CPU measurement, resource matrix, full release build,
or product documentation update was performed. Root review of the actual `.19`
binary remains the acceptance gate.
