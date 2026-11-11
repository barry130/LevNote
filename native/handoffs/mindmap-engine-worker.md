# Mindmap SVG worker

Status: ACTUAL COMMAND GATE PASS, 2026-11-09, executed by `astra_marker_view`
after root accepted the corrected test source. The earlier owner's summaries
had no retained raw streams/before-after inventory and are preliminary only.
Current command/test source is unchanged and frozen pending root delivery gate.

Owned scope is new `native/cmd/levnote-mindmap-svg/main.go` and tests plus this
handoff and new evidence. The command reads exactly one operation-bound
request with `engineipc.ReadRequestFor(engineipc.MindmapSVG, r)`, use the frozen
catalog identity and bound, parse with the default 16 MiB/depth256/100000-node
limits, lay out via `mindmap.NativeLayoutOptions`, and emit through
`mindmap.ExportSVG` with the 64 MiB hard cap and a private small-budget seam.
Warnings are ordered parse, layout, presentation and use whole-document
ranges because the frozen model has no source spans. Responses use the existing
`engineipc.WriteResponse`, preserve `engineVersion`, and keep diagnostics free of
source bytes.

No product source outside the new command was changed. The actual final gate
ran in-process serve tests and scoped vet using the frozen accepted IPC. It
built/ran Go's test executable as usual; it did not build a production worker,
launch a worker subprocess, package a release or run resources/physical UI.

## Source/API checkpoint

2026-11-09: the first production checkpoint is complete at
[`../cmd/levnote-mindmap-svg/main.go`](../cmd/levnote-mindmap-svg/main.go).
Root has confirmed that the IPC product source is frozen; GLM now owns only
tests/logs, so command tests and vet are not waiting on a product edit.

`main.go` SHA-256:

```text
7207D0BBD20F218ADEE8215ADC56F4D9919C8583FE30FA2777A4FF057F3B687E
```

Frozen dependency hashes read at the final command gate:

| Product source | SHA-256 |
| --- | --- |
| `core/enginecatalog/catalog.go` | `B064E0165F5E6577B460616465BE6456DA2A0027EE02A378827DE3CDC87D46B2` |
| `core/engineipc/protocol.go` | `E1DC708E0391899430FEB43B4ED2755938D570F11D3B0FA713778B1DD5A2A7DA` |
| `core/engineipc/decode.go` | `663A6B44EE185DD8B1383955DB529441BA63B2F10AD66946FBB72DC04F8F6EA5` |
| `core/engineipc/json.go` | `FE829AE271E28D6D237E6605F03663DBD740295818511CEB68D425403E5E802C` |
| `formats/mindmap/parse.go` | `4B4AC072080E1462B85ABF225CC6B3C9BB9842AD17C0B59B80E024D1FC950B2A` |
| `formats/mindmap/layout.go` | `87245A07934051C05499FE5D3331E9E4C7B03FD5788FBAB46280F06BD3312547` |
| `formats/mindmap/presentation.go` | `0F54E697771ED666554ED0C51545025E9FA212886F5249225D0534442EB12296` |
| `formats/mindmap/svg.go` | `106D55A6FDFDB7D3E39C35087E82E802DDF53E6702DE6BD82326AFE5F42FE2D0` |

Final test source SHA-256:

```text
DDC6F1012E207CDCD1F91FFC28F787B67A250D91CEF37B17457BF02F3CB45415
```

## Final command gate

2026-11-09: the finite test file now exercises the actual in-process serve wire
with independent literal/XML geometry and escaping oracles, pure renderer parity
as a secondary check, exact source nonmutation, deterministic successful output,
default and unknown layouts, whole-source warning ranges, hidden metadata
disclosure, real-serve `warning-limit` refusals, invalid/deep/oversize/foreign
requests, tiny SVG output caps with an escaped label, broken and short writers,
and frozen catalog/parser/output limits. Root's test-review corrections were
implemented; no `main.go` defect was exposed, so its reviewed fingerprint above
is unchanged.

Actual exact runners, direct process stdout/stderr, programmatic exit/time
metadata and before/after inventories are in
[mindmap-command-20261109-070556-actual-attempt01](../artifacts/acceptance/mindmap-command-20261109-070556-actual-attempt01/result.json).
The runner clears GOFLAGS and LEVNOTE opt-ins and uses child invocations of the
pinned `go.ps1 -GoArguments` with explicit arrays:

- `@('test','./cmd/levnote-mindmap-svg','-count=1','-v')`:9 actual RUN/PASS,
  package0.537s, actual wall1.6124803s, exit0,1166byte stdout and empty stderr.
- `@('vet','./cmd/levnote-mindmap-svg')`:actual wall0.8305406s, exit0,
  empty stdout/stderr.

All20 owned/frozen production dependency/wrapper/module/go.exe inputs stayed
identical. `main.go` and `main_test.go` match the expected fingerprints above;
no fixes were needed. Result SHA256:
`79B5E4F0161D53E83D2C1028528B44F7BEDFE5DEC3C47AE41E02D5457E1BEF47`.
Identical source-before/source-after SHA256:
`481E7C18BBD58791B8C532EDD9971E9C3DD35D054EFFFE1603F1A9F116F83F83`.
Actual test stdout SHA256:
`CC5D2C88E9CB45B0BB40C9BA161A03FBAB3C0C8E6455DCB01F7A440804D66242`.

The preserved earlier directory
[mindmap-worker-command-tests-20261109](../artifacts/acceptance/mindmap-worker-command-tests-20261109/)
contains preliminary hand-authored summaries/source hashes, not retained raw
test/vet streams or before/after evidence. Its claimed times/pass results are
not used for final acceptance; the actual record above supersedes them without
rewriting those historical files. No full suite/runtime/app/resource/physical UI
was repeated, and production `.6` packaging remains separately gated.

The command has inherited stdio only (`os.Stdin`, `os.Stdout`, `os.Exit`), no
flags, environment, paths, files, subprocesses, networking or format UI
dependency. Its production `serve` path uses the 64 MiB hard SVG cap; only the
unexported `serveWithLimit` test seam accepts a smaller budget.
