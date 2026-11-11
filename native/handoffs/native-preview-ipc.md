# Fixed preview catalog and IPC source

2026-11-09. Source-ready for the command owner. Root accepted the exact minimal
descriptor and alias before formatting. The only production changes are in
`core/enginecatalog/catalog.go` and `core/engineipc/protocol.go`; both are
`gofmt`-clean and frozen while independent transport tests and scoped evidence
are completed. No other production, runtime, command, build, process, UI,
dependency, manifest or script change is in progress.

## Fixed definitions

- `MindmapPreview` engine ID: `levnote.mindmap-preview`.
- Descriptor format: `mindmap-scene-v1`, protocol `1`, relative path
  `engines/mindmap-preview`, executable `levnote-mindmap-preview.exe`,
  platform `windows/amd64`.
- Source budget: `MindmapSourceMaxBytes` (16 MiB).
- Payload budget: `enginescene.MaxPayloadBytes` (32 MiB).
- IPC alias: `engineipc.MindmapPreview = enginecatalog.MindmapPreview`.
- Legacy `ReadRequest` remains the Markdown-only compatibility wrapper.

The command owner may compile and run command tests against these frozen APIs.
Catalog/IPC-specific independent oracles, full scoped package gates and actual
evidence will be appended after source testing.

## Scoped acceptance

- Added the exact preview descriptor assertion, including 16 MiB source and
  32 MiB payload budgets.
- Added cross-format, cross-engine and platform rejection assertions.
- Added an independently constructed preview request frame and fragmented
  preview response frame with literal metadata bytes.
- Proved wrong preview identity, expected-version and format metadata are
  refused before payload allocation.
- Proved preview source-cap and response-cap headers are refused before
  payload allocation/read, with the source error classified as `source-limit`.
- Proved legacy `ReadRequest` still refuses preview requests without reading
  their source payload.
- Production files were formatted and frozen before these test-only additions.

## Actual evidence

Fresh retained evidence is in
`native/handoffs/evidence/preview-ipc-20261109/`.

- Full test: `native/scripts/go.ps1 test ./core/enginecatalog ./core/engineipc -count=1`
  — exit 0, 07:51:40.1952115 to 07:51:41.2973835.
  `catalog 0.508s`, `engineipc 0.598s`; stdout 102 bytes, stderr 0 bytes.
- Scoped vet: `native/scripts/go.ps1 vet ./core/enginecatalog ./core/engineipc`
  — exit 0, 07:51:41.2974825 to 07:51:41.6241958.
  stdout 0 bytes, stderr 0 bytes.
- Inventory: the earlier `test-full.log` is a separate retained passing run
  (`catalog 0.544s`, `engineipc 0.635s`); the authoritative final retained run
  is `test-full.stdout.txt`/`test-full.stderr.txt`
  (`catalog 0.508s`, `engineipc 0.598s`). `inventory.json` originally contained
  only filename and structural notes. It now additionally records the four
  source files' size and SHA-256 as an explicit captured-after inventory;
  no pre-run source hash inventory was retained.
- `ReadRequest` remains the unchanged Markdown-only compatibility wrapper.

No runtime, worker, command, build, package, process, UI, manifest, dependency
or script execution occurred in this scoped gate.
