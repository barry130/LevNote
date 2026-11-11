# Doc-block reference integration

## Current status — ACCEPTED 2026-11-08 19:24 +08:00

Focused parser/view/app evidence, the final clicked-row correction and GLM's
no-findings review were accepted by root. Product source remained frozen while
DeepSeek executed the single authorized full test, vet, release build and
isolated hidden native startup/close sequence.

Root read the saved `result.json`, raw 20-package test stdout,
`executable-meta.json`, 12/12 evidence validation and smoke result, then
accepted this exact executable as the current doc-block reference release. The
18:17 mindmap-fields build below is now the previous accepted binary.

## Exact commands

Run once from the repository root with the bundled Go 1.27.3 wrapper:

```powershell
native\scripts\go.ps1 -GoArguments @('test','./...','-count=1','-timeout','300s')
native\scripts\go.ps1 -GoArguments @('vet','./...')
native\scripts\go.ps1 -GoArguments @('build','-trimpath','-buildvcs=false','-ldflags','-s -w -H=windowsgui','-o','bin\levnote-native.exe','./cmd/levnote')
```

- Full test: exit 0, command wall time 11.613s.
- Full vet: exit 0, command wall time 1.903s; both live-captured streams are
  intentionally zero bytes.
- Release GUI build: exit 0, command wall time 2.444s; both live-captured
  streams are intentionally zero bytes.
- The 1,062-byte test stdout log contains 20 package summary lines, all `ok` or
  `[no test files]`; test stderr is a live-captured zero-byte file.
- Notable package durations: `app` 7.128s, `core/plugins` 3.128s,
  `modules/markdownview` 4.281s, `modules/graphview` 5.533s.

The final affected view guard checkpoint remains `1.120s`; that duration is not
the full `markdownview` package duration. The full package result in this
integration is `4.281s`.

## Accepted executable

- Artifact copy:
  `native/artifacts/acceptance/docblock-reference-20261108-192000/levnote-native.exe`
- Source path: `native/bin/levnote-native.exe`
- Bytes: `13,298,176`
- SHA-256:
  `DF911A7F2C37E5548CDC6BF6E92D0FE93CB45283277D010020C267095A3D5D82`
- UTC mtime: `2026-11-08T11:20:17.6539304Z`
- PE machine `0x8664`, optional-header magic `0x020B`, Windows GUI subsystem 2.
- Go 1.27.3, Windows amd64/v1, `CGO_ENABLED=1`, `trimpath=true`, MyGo v0.8.3.
- Source and artifact-copy size/hash values match.

The previous accepted 18:17 mindmap-fields executable is preserved at:

`native/artifacts/acceptance/docblock-reference-20261108-191758/previous-accepted-mindmap-fields.exe`

- Bytes: `13,274,624`
- SHA-256:
  `F06ABEF11653EF93F03D3017A62215CACD8364536E9EDF900242DC9283950A6C`
- UTC mtime: `2026-11-08T10:17:14.9536389Z`

## Isolated hidden smoke

Evidence root:
`native/artifacts/acceptance/startup-smoke-docblock-reference-20261108-192100`

- Harness exit 0; wall time 3.689s.
- The app was launched with `Start-Process -WindowStyle Hidden`.
- PID-scoped `EnumWindows` found owned PID 35788, title `轻笺 · 原生版`.
- `WindowVisibleAfterHiddenLaunch=false`.
- PID-scoped `WM_CLOSE` was accepted; the process exited 0 within the 15s bound,
  without forced cleanup and with no remaining `levnote-native` process.
- A realistic local `mode: reference`, `src: target.txt` fixture was opened.
- Unrelated updater-shaped file and directory sentinels survived with exact
  contents `preserve-file` and `preserve-directory`.
- Fixture and target hashes were unchanged. Source/copied executable hashes
  matched the accepted SHA-256 above.
- This proves hidden startup/graceful close, not visible GPU rendering.

## Evidence and validation

Main artifact:
`native/artifacts/acceptance/docblock-reference-20261108-192000`

- `result.json`: aggregate command, package, executable, prior-artifact,
  screenshot and smoke evidence.
- `test.stdout.log`, `test.stderr.log`, `vet.stdout.log`, `vet.stderr.log`,
  `build.stdout.log`, `build.stderr.log`: actual live captures, including
  intentional zero-byte streams.
- `test-packages.json`: 20 parsed package lines with literal tabs preserved in
  each `RawLine`.
- `executable-meta.json`, `go-version-meta.stdout.log`,
  `go-version-meta.stderr.log`: executable identity and Go build metadata.
- `previous-accepted-metadata.json`, `source-fingerprints.json`,
  `screenshots-meta.json`, `smoke-summary.json`, `OUTPUT-PROVENANCE.txt`.
- `evidence-validation.json`: PASS, 12/12 checks. It independently revalidated
  exits, captured stream sizes, all 20 package summaries, executable and PE
  identity, the preserved prior binary, four screenshots, smoke state and the
  absence of resource sampling.
- The first validator invocation exposed a local PowerShell expression error
  before producing a verdict. It is preserved as
  `evidence-validation-attempt-01-failure.json`; only the validator script was
  corrected, and no product command ran again.
- `root-acceptance.json` records the accepted exact identity and evidence
  reviewed by root; it is 993 bytes, SHA-256
  `2D0B2B0EC893C01FE6CE97708644314CA735C141BCCBF02EB39BDDFF8E2FB489`.

The four already-reviewed 760x480 normal/refused light/dark images were copied
unchanged into the main artifact. The final clicked-row guard did not require
new visual evidence because it changes source identity validation, not layout.

No process-resource sampling, benchmark, product source edit, accepted-command
rerun, rebuild or child-agent execution occurred during integration.

## Post-acceptance documentation audit

After root accepted the exact executable above, a read-only audit covered the
nine owned product documents: `native/README.md`, `native/formats/API.md`,
`native/ARCHITECTURE.md`, `native/VALIDATION.md`, root `README.md`,
`ROADMAP.md`, `CHANGELOG.md`, `docs/decisions/0063-native-migration.md` and
`docs/decisions/README.md`.

- All 9 files decoded as strict UTF-8 and all 9 had balanced Markdown fences.
- All 197 relative Markdown links resolved; there were 0 failures.
- The accepted hash, main artifact root and hidden-smoke root are present.
- The executable, aggregate result, 12/12 evidence validation,
  `root-acceptance.json` and smoke result paths all exist.
- `root-acceptance.json` records `ACCEPTED`, 13,298,176 bytes, SHA-256
  `DF911A7F2C37E5548CDC6BF6E92D0FE93CB45283277D010020C267095A3D5D82`
  and UTC mtime `2026-11-08T11:20:17.6539304Z`.
- The audit found no stale current-release wording such as “candidate
  complete” or “pending root acceptance”.

The native overview and validation record now link the preserved
`root-acceptance.json`. This documentation-only correction did not rerun tests,
vet, build or smoke.

