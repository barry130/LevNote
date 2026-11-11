# Draw.io route `.20` delivery

Status: **CANDIDATE READY FOR ROOT BINARY ACCEPTANCE** 2026-11-10.

`native-worker.20261110.20` completed the authorized delivery sequence: one
portable README wording update, one package-script invocation with five Windows
GUI builds, one hidden byte-identical host route/lifecycle gate, and a final
12/12 aggregate validation. No source suite, test executable, worker request,
resource matrix, capture, or additional package build was run. Launch
documentation still names the accepted package and must remain unchanged until
root accepts this actual `.20` binary.

No product source, test, package script, asset, candidate, or evidence file was
edited after aggregate execution. This handoff is the only delivery artifact
written after that aggregate.

## Portable README input

The sole authorized product prose edit is in
`native/portable/README.md`. It now states that the bundle supports all three
schema3 routes and retains `.19`'s two-route support as history.

| State | Bytes | SHA-256 |
|---|---:|---|
| Before | 12,469 | `8398D4B5D4821204EEF456D7F392887A06DCA8D27059573EC14D58622FF67AB8` |
| After | 12,455 | `700243D30C97FD8B72110FBC5D96C4794CEAF44461A5197CBDE5E4EC5F0BA80E` |

The source still has exactly one `{{VERSION}}` marker and the same three inline
schema3 manifests. The rendered package README is 12,469 bytes, SHA-256
`9BEDC3CB386F1EA5F02117D22D9FA8606F00D93D17978E2A1792A1D8A4A6BB5E`.

Raw wording evidence:

- [before source](../artifacts/acceptance/drawio-route-release20-20261110-031456/portable-readme-before.md)
- [exact diff](../artifacts/acceptance/drawio-route-release20-20261110-031456/portable-readme.diff)

## Candidate and builds

`native/bin/levnote-native-worker-20261110.20` contains **25 files /
26,854,177 bytes**. Its complete inventory identity, including relative path,
byte count, SHA-256 and modification ticks, is
`6B899EFB2FB2A90CB2FB93CFE9BC48A3B94B4CFA2AB16C61C5AB005B05DF14ED`.

| Item | Bytes | SHA-256 |
|---|---:|---|
| `levnote-native.exe` | 14,095,360 | `4B776F634A3626F2BDD15303E175FE38CC4908D566AC54A6D25F51D989B4F34E` |
| `release-manifest.json` | 4,490 | `53C0B71C317734001596988F6AB2BF44201879CC322BA9DB3B9AB494E553DBFA` |

The one package-script invocation exited 0. Every build exited 0 with zero
stdout and stderr bytes:

| Build | Payload bytes | Wall time | Payload SHA-256 |
|---|---:|---:|---|
| markdown export | 3,303,424 | 1.137 s | `B32B43668E5072623981A6C5627E960AB1CA8088D0205C0D6232EDCC6D889856` |
| mindmap SVG | 3,126,272 | 1.297 s | `9FB4EB306825B4C8B678DE790D7FD42D12666CEDF745DCE92BC7D4ED496CB193` |
| mindmap preview | 3,140,608 | 1.011 s | `705F7C0CAA83F785E55D6CD5FB4D92A054DC57F1CFB7EC8BFC21A0E69424F8F7` |
| draw.io SVG | 3,121,664 | 1.119 s | `70C15942339ECF074F81192C01CE652B19FDBC7DBDFABA139FED22E66C46FE9A` |
| host | 14,095,360 | 1.975 s | `4B776F634A3626F2BDD15303E175FE38CC4908D566AC54A6D25F51D989B4F34E` |

The complete package copy is byte-identical at **25 files / 26,854,177
bytes**. The schema1 release manifest names `.20`; all 24 non-manifest payload
entries match their exact sizes and hashes. The four engine manifests match
their payloads and version. All five executables are Windows amd64 PE32+
GUI images. Thirteen portable assets/notices and all authored local links
passed.

Raw package records:

- [package invocation](../artifacts/acceptance/drawio-route-release20-20261110-031456/package-command.json)
- [package result](../artifacts/acceptance/drawio-route-release20-20261110-031456/package-result.json)
- [inner package record](../artifacts/acceptance/drawio-route-release20-20261110-031456/package/package-result.json)
- [package stdout](../artifacts/acceptance/drawio-route-release20-20261110-031456/stage-package.stdout.log)

## Hidden route and lifecycle gate

The gate copied the complete candidate once, then started that exact copy with
`Start-Process -WindowStyle Hidden`. It used a fresh run-owned configuration
root on the normal C: temp volume, the reviewed 179-byte
`diagram-notes.json` seed, and one 893-byte UTF8+BOM/CRLF draw.io XML fixture
with unknown metadata and the custom suffix `.diagramnote`.

- Start count / PID / held handle: `1` / `31892` / `2312`
- Path identity: exact held-handle `QueryFullProcessImageName`
- Window found / visible: `true` / `false`
- Owned `WM_CLOSE` posts: `1`
- Normal host exit: `0`; timeout `false`; forced cleanup `false`
- Job processes after assignment / after exit: `1` / `1`
- Worker requests: `0`
- App stdout / stderr: `0` / `0` bytes
- Fixture before and after: 893 bytes,
  `7A166D51274D6868463F6D2CF671D2B8680D05047A24D2A85156E28843E4A1FB`
- Installed plugin JSON before and after: 179 bytes,
  `317FB43C43863A092FF7AA346C1A436A26C75A8BE41DC5E74C7F184AAE529A07`
- Session after close: 427 bytes,
  `42438072AC5F00E8DF12FC16AC69B73B1E88A1DE5BBB8B0AD2FBD8F507113561`
- Candidate and complete copied bundle remained unchanged

The clean session contains exactly one tab with the exact fixture path,
`pluginId=diagram-notes`, `view=preview`, `encoding=UTF-8`, `bom=true`, and no
draft. The fixture and installed JSON retain exact bytes and write ticks.

Raw host records:

- [host result](../artifacts/acceptance/drawio-route-release20-20261110-031456/host-smoke/result.json)
- [session](../artifacts/acceptance/drawio-route-release20-20261110-031456/host-smoke/session-after-close.json)
- [outer invocation](../artifacts/acceptance/drawio-route-release20-20261110-031456/outer-host-smoke-invocation.json)
- [stage stdout](../artifacts/acceptance/drawio-route-release20-20261110-031456/stage-host-smoke.stdout.log)

This establishes packaged custom-suffix configuration loading, path routing,
session persistence, and clean process lifecycle only. It does not establish
visible rendering, physical input/IME behavior, interactive SVG export, dirty
Save/recovery, or process memory/CPU use.

## Execution and aggregate

The formal harness used PowerShell 7.6.9. Its single static parser pass covered
10 scripts with zero parse errors:

- [static parse](../artifacts/acceptance/drawio-route-release20-20261110-031456/static-parse.json)
- [static parse stdout](../artifacts/acceptance/drawio-route-release20-20261110-031456/static-parse.stdout.log)

Formal initialization reported `AUTHORIZED`, six route gate files, and three
accepted route evidence directories. The stages were run once each:

```text
pwsh.exe -NoProfile -File ./run-stage.ps1 -Name baseline
pwsh.exe -NoProfile -File ./run-stage.ps1 -Name package
pwsh.exe -NoProfile -File ./run-stage.ps1 -Name host-smoke
pwsh.exe -NoProfile -File ./run-stage.ps1 -Name aggregate
```

| Stage | Outer seconds | Stage seconds | Exit | Stage stderr |
|---|---:|---:|---:|---:|
| baseline | 2.273 | 1.577 | 0 | 0 bytes |
| package | 10.688 | 9.954 | 0 | 0 bytes |
| host-smoke | 6.867 | 5.706 | 0 | 0 bytes |
| aggregate | 2.556 | 1.912 | 0 | 0 bytes |

The applicable-not-dead validator passed **12/12**:

- [validator.json](../artifacts/acceptance/drawio-route-release20-20261110-031456/validator.json),
  5,418 bytes, SHA-256
  `5B4FD713AD5A14BAE8E2D5266CBF55B66802D3161151660509844A8662CC1455`
- [aggregate.json](../artifacts/acceptance/drawio-route-release20-20261110-031456/aggregate.json),
  24,651 bytes, SHA-256
  `C1CAEC5205F8A833A4DFF86B84485E4D84187CAC3BD6D3DA88DA1F47D7E59611`

The validator covered the five builds and exact candidate/copy/manifest, four
engine manifests, five Windows GUI PE images, portable assets/links and README
routes, immutable `.19`, frozen scoped inputs and accepted gates, the clean
session, the exact hidden process lifecycle, and post-run byte preservation.

## Protected and frozen inputs

The accepted `.19` package remains **25 files / 26,852,621 bytes**, complete
inventory identity
`7925942AFDE49F63C2D77D79EBE34FD3F1BAD18F535745E30171AFA3CD27A24B`.
Its complete direct evidence remains **195 files / 91,652,331 bytes**,
identity
`4CFA46B862E2798CCEB0A49645E1096044465C9D77AC291509D072C48523D7F4`.
Its accepted aggregate remains SHA-256
`91DB69CDA656C177A925178EC30239D693166DFB0BD722BF79B587ED1D10B2BC`.

The scoped product/build inventory remained unchanged at **315 files /
3,153,585 bytes**, including 64 `native/formats` files. Six accepted route gate
files, three route evidence directories, and all seven plugin example files
remained byte/time unchanged. The reviewed `diagram-notes.json` is 179 bytes
with SHA-256
`317FB43C43863A092FF7AA346C1A436A26C75A8BE41DC5E74C7F184AAE529A07`.

After the final aggregate, the complete evidence root contains **139 files /
54,304,954 bytes**, identity
`5ED2F19531E178E33F355164546AB50C5C31F37E032F9E8B7CCBB6791C9C8621`.

## Setup notes

Before formal initialization, two shell-only attempts to copy the adapted `.19`
harness emitted setup/runtime metadata errors. The first wildcard copy failed
after creating the fresh evidence root; the second copied the scripts but
emitted a `Continue=Stop` parameter-parse message. Neither attempt reached
formal initialization or started any product operation. The single static
parse and formal initialization then passed, and all stage/candidate/evidence
records above come from that formal sequence.

## Scope and remaining acceptance gate

The draw.io route behavior is established by the previously accepted affected
source/app evidence plus this packaged hidden route gate. Draw.io runtime,
renderer, parser, and worker behavior are inherited from accepted `.19`
evidence; this delivery made no worker request.

No full release build, source suite, `go vet`, test executable compilation,
capture, visible UI automation, physical input/IME test, interactive SVG
export, dirty recovery test, process-memory measurement, CPU measurement, or
resource matrix was performed. `native/examples/plugins/README.md`, launch
documentation, and all `.19` artifacts remain unchanged. Root review and
acceptance of the actual `.20` binary remain the package gate.
