# Whole-word Find `.17`: candidate-ready delivery

2026-11-10 00:49 +08:00. Status: **CANDIDATE-READY; root binary acceptance
required**. This delivers the source accepted in
[find-whole-word-root-acceptance.md](find-whole-word-root-acceptance.md).
No product source, test, packaging script or product documentation was edited
or rerun. `.16` remains the current accepted package pending root acceptance of
the actual `.17` binary.

## Candidate and immutable evidence

- Candidate:
  `native/bin/levnote-native-worker-20261110.17`
- Version: `native-worker.20261110.17`
- Package: 23 files, 23,690,310 bytes
- Host: `levnote-native.exe`, 14,066,688 bytes, SHA-256
  `E4E81A30B0CB47F080379BD75EEEBD003F4621B3A8393D344073D32DA94EBC1C`
- Release manifest: 4,137 bytes, 22 payload entries plus itself excluded,
  SHA-256 `668C94BCDAC492BE88D96AE45A73157B7B5B7427C001FC2F4452DF0BB6E3BD08`
- Fresh evidence:
  `native/artifacts/acceptance/find-whole-word-release17-20261110-004642`
- Evidence inventory at final aggregation context: 117 files, 47,784,783
  bytes, identity
  `41FCCE7072CD36F3955C859DB54E056D453EA2A21BC19552916624601A8367B4`
- [validator.json](../artifacts/acceptance/find-whole-word-release17-20261110-004642/validator.json):
  12/12 PASS, SHA-256
  `9528DBBDC809915FA9CD2C4251700226365E2EF10E79F97B616D8C38A9F23038`
- [aggregate.json](../artifacts/acceptance/find-whole-word-release17-20261110-004642/aggregate.json):
  PASS, SHA-256
  `B7094AFE989AD9D35A59CCBB4E7312A4BCDC55440AFF2061A50A08FACD3253ED`

The package directory and separate 23-file copy are byte-identical. The
release manifest names every shipped payload with matching size and SHA-256.
The three engine manifests independently match the packaged `.17` workers:

| Engine | Worker bytes | Worker SHA-256 | Manifest SHA-256 |
|---|---:|---|---|
| markdown-export | 3,300,352 | `6E367C919CA7897C5CD768AAACDB031EF085D22CB4E39E33312CEADEABA10072` | `1C07567478456408C0FED7EF4064BDC0FC3629A2F6EF2471F5A0342799EC3EE4` |
| mindmap-svg | 3,122,688 | `D3094CC7919E9245E2B34809EAE5DA4E4494AD13CC3FF38E298E7E87D8BDAB4C` | `20DB91078D1895AA7527E5B51C2A887FBBD496F68B9AE5B32342795F7142E7CD` |
| mindmap-preview | 3,136,512 | `B850DB6971EE28B83753DEB44146846CA13B6AA2EEE409B687864B4E8B6D0664` | `76C87F480EF27A69C122E01A4394643AB6389C00548E1819BE70D0429132FB3E` |

All four executables are Windows amd64 GUI PE images. All 13 direct portable
source-asset copies retained their source hashes, and the packaged authored
links and notices/licenses resolved.

## Execution

The existing package script was invoked exactly once:

```powershell
native\scripts\package-markdown-worker.ps1 `
  -OutputDirectory F:\AIproject\LevNote\native\bin\levnote-native-worker-20261110.17 `
  -EvidenceDirectory F:\AIproject\LevNote\native\artifacts\acceptance\find-whole-word-release17-20261110-004642\package `
  -Version native-worker.20261110.17
```

Raw package output is [package.stdout.log](../artifacts/acceptance/find-whole-word-release17-20261110-004642/package.stdout.log);
stderr is empty; exit was 0. Its single output line is the candidate directory.
The four builds used the repository `go.ps1` wrapper and each recorded exit 0
with empty stdout/stderr:

| Build | Seconds |
|---|---:|
| worker-markdown | 1.16 |
| worker-mindmap | 1.02 |
| worker-mindmap-preview | 1.15 |
| host | 1.64 |

The wrapper record is
[package-command.json](../artifacts/acceptance/find-whole-word-release17-20261110-004642/package-command.json).
Outer and stage stdout/stderr/exit records are retained separately. No package
invocation, test, vet, benchmark, capture, managed tester compilation, managed
worker request or product build was repeated.

## Inputs and protection

- Scoped source/build inputs: 300 files, 3,005,261 bytes, including 60 files
  under `native/formats` and the existing
  `native/portable/NOTICE-PREAMBLE.md`.
- Inputs before/after are byte/time identical:
  [before](../artifacts/acceptance/find-whole-word-release17-20261110-004642/inputs-before.json)
  and
  [after](../artifacts/acceptance/find-whole-word-release17-20261110-004642/inputs-after.json),
  both SHA-256
  `AB9ABFC1139447C65DCA995D068F4B26CA9CA50129A8C487AB7177E339ECABB2`.
- Frozen app/test identities remained:

| File | SHA-256 |
|---|---|
| `native/app/app.go` | `58B6A783968C8860BBE6E8EF87E74CE9C6BEA3560BD4C19A569FC44465557931` |
| `native/app/state.go` | `023C76A143CB572D0A2B45D783A20E105996FF570615357BB1C3F31032E79984` |
| `native/app/find_whole_word_test.go` | `BEBFEBC560AA78D8A47B1390A2CA4E29E956C8328CB4AFEE0B080BF81328E744` |
| `native/app/search_navigation_test.go` | `FA8EF3C1DAF50B67AA03F7ADE032F465EB0535202FAB4328EB7C28DFC4E72A40` |
| `native/app/findbar_layout_test.go` | `662C167BFFA6CC344EF2E2A73E3566824E03B0DA3BA3D19FA463279FFCF12180` |

- Accepted whole-word result, review/correction files, raw logs, light/dark
  captures and root acceptance remained unchanged. The final reconciliation
  is `16C1A31CF96E958C221F9995E794C2CC5348DF41BEE2F3F977B604BCCE7D2E5E`;
  the second corrected focused log is
  `5F50F57B160F074F46D04688300C2AC68AE903758F14A1E40CC96021E0F51065`.
- Accepted `.16` remained exactly 23 files / 23,690,310 bytes with host
  SHA-256 `76A60252…E60B03D` and manifest SHA-256
  `53994DB6…050CD90`. Its complete direct delivery evidence also remained
  exactly 104 files / 47,774,882 bytes with inventory identity
  `15A3050C4B0F1C471798B4A958B457963D44A50A634E8113550B34001A2BB9F2`
  and aggregate SHA-256
  `00BF4482CB0C660A98A60C7CAD7A7CA0E1D61757B30D57CD4AD673EB90B5C0A5`.

## Hidden host smoke

The one allowed smoke used an exact copied 23-file bundle from `F:` and a
run-owned `APPDATA`/`LOCALAPPDATA` root under normal `C:` TEMP. The generated
60-byte Markdown source has SHA-256
`7CBEA90B48A026B9B0704F5F37F4AF7FDBB912BB15ABD6A4FC421D9003EFB6A1`.

- One hidden start; exact held process image checked by
  `QueryFullProcessImageName`.
- Owned hidden window found; one owned `WM_CLOSE`; normal exit code 0.
- No timeout or forced cleanup; owned PID is absent after close.
- Source and candidate/copy bundles remained unchanged.
- Post-close session is 414 bytes, 0 terminal LF/CR, exactly one clean tab for
  the generated source, with no draft.
- App stdout/stderr and requested worker operations were zero.

See
[host-smoke/result.json](../artifacts/acceptance/find-whole-word-release17-20261110-004642/host-smoke/result.json)
and
[session-after-close.json](../artifacts/acceptance/find-whole-word-release17-20261110-004642/host-smoke/session-after-close.json).

## Scope

Frozen source/tests, focused and final affected suite/vet evidence, and both
760x480 captures establish the whole-word Find slice through the accepted
source gates. This package run proves packaging, manifest/PE/asset integrity,
and one hidden startup/clean close. It does not rerun Find UI, and the packaged
host smoke does not prove visible Find interaction, physical IME/focus,
accessibility, dirty Save/recovery, worker execution, plugin/package import,
GPU, CPU, memory or resource behavior.
