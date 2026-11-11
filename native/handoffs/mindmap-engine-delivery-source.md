# Second-engine delivery source checkpoint

Status: SOURCE PREPARATION COMPLETE, 2026-11-09. Source is ready for root review;
no production package/build or worker launch has run in this step. Existing `.5`
and all accepted payload/evidence remain untouched. Runtime/app source stays
frozen at [its completed gate](mindmap-engine-runtime.md).

Three owned files changed:

- `scripts/package-markdown-worker.ps1`: same entry/parameters, path preflight,
  pinned wrapper, target flags, separate evidence folder and asset helpers.
  A fixed two-item definition table feeds one build/PE/manifest loop. Each worker
  receives its own fixed command/path/ID/format, version flag and ≤16KiB manifest;
  each executable is ≤64MiB and checked as Windows amd64 GUI PE. Host is built
  once afterward. `package-result.json` now has `Engines[]` with identity, limits,
  worker bytes/hash/time/PE and manifest path/hash. No singular Worker field is
  reused ambiguously; historical records remain unchanged.
- `portable/README.md`: both worker directories and on-demand one-shot behavior;
  Markdown8MiB/Mindmap16MiB source and64MiB results; native editing/preview,
  source-mode SVG without graphview; literal bounded labels, image placeholder,
  whole-document warning scope and refusal on warning metadata bounds. No new
  resource, sandbox or external plugin claim. Source launch docs still name `.5`.
- `app/export_mindmap_worker_bundle_test.go`: one new opt-in
  `TestMindmapWorkerExportBundledHostPathAcceptance`. It uses the existing
  `LEVNOTE_WORKER_BUNDLE_ACCEPTANCE=1` switch, registered source-mode Mindmap,
  View nil and NewBundled beside the tester. Only SaveDialog is injected.

The new host-path fixture preserves BOM+CRLF, Unicode, custom icons, plain/object
tags, unknown numeric spellings and external image metadata. Both requests check
exact engine version/hash, independently literal ordered warnings with whole
source ranges and shared pure SVG parity. Independent XML checks assert root
viewBox `-24 -24 196 144`,92×40 node at28,28, literal colors, escaped full label
and the image placeholder at112,42; there is no image element/network URL in SVG.
First request publishes exact bytes; second refuses the same target. Each checks
disk/Buffer/version/selection/history/encoding/BOM/dirty/baseline and cleared
payload; Close must succeed. Record reports2worker requests and joined-return
contract without inventing PID observations. Existing Markdown test is unchanged.

## Actual preparation evidence

[delivery-source-attempt01](../artifacts/acceptance/mindmap-engine-20261109-064417-delivery-source-attempt01/result.json)
PASS, SHA256 `8B2DF79A6300699DF8B81263BAE845B1821F46DDBFAB327F3E3A0FDAD49999B5`.
All92 previously frozen inputs match before and after (empty pipeline fields
serialize as null in this small record). The three current source fingerprints
are in [source.json](../artifacts/acceptance/mindmap-engine-20261109-064417-delivery-source-attempt01/source.json),
SHA256 `EE10F1630FF46E70B12C0116A2D51F70F050DF4D496AE51D05C8142D093C458E`.

- Wrapper fmt of new test only, exit0.
- `test ./app -run '^TestMindmapWorkerExportBundledHostPathAcceptance$' -count=1 -v`:
  explicit RUN/SKIP, package PASS1.357s. This compiles the tester; it does not
  prove the real worker path yet. LEVNOTE opt-ins and GOFLAGS were cleared.
- Static PowerShell AST parse, zero errors. Independently read literal fixed
  definitions match exact expected IDs, formats, paths, packages and source caps;
  one shared loop contains the worker build call plus one host call. Required
  portable description strings present. This is static validation, not an
  executed packaging/PE/manifest result. All three stderr streams empty.

| File | SHA256 |
|---|---|
| scripts/package-markdown-worker.ps1 | `2F1E6825F1EE7184787778C0E3D4A720B259CCA25F0DB346FD419DAC8ECD35CB` |
| portable/README.md | `8FA4AE298EE8BAA9B2CFEA7F3ED5E9E8989DEC6D0D01484940BFC5234B163FF8` |
| app/export_mindmap_worker_bundle_test.go | `2382A35951D4817E4D966CFC3758EC85232478200B07CCD8E1DBB9409B6ADDC7` |

After source/review/worker acceptance, the separately gated `.6` delivery plan
will build three binaries, validate the complete21-file set and run the existing
Markdown plus new Mindmap HostPath once each in a disposable complete copy
(four real worker requests total). No such execution is claimed by this checkpoint.
