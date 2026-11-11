# Draw.io fourth worker, stage 1 result

Status: COMPLETE and source-frozen for independent review, 2026-11-10.

This stage implements the fixed `levnote.drawio-svg` lower layers only. It does
not advertise the worker from the Draw.io adapter or App, does not package a
fourth worker, and does not build or launch the product. The accepted `.18`
release and all earlier release/evidence directories remain unchanged.

## Implemented contract

- Added the fixed Draw.io SVG catalog identity and Windows amd64 descriptor:
  `engines/drawio-svg/levnote-drawio-svg.exe`, format `svg`, protocol 1,
  32 MiB source and 64 MiB payload limits.
- Kept wire protocol version 1 and old request bytes unchanged. The old three
  engines reject any `page` field before the source payload is read.
- Added required Draw.io page metadata `{index,id}`. The strict decoder accepts
  exactly `engine`, `format`, and `page`; the page accepts exactly `index` and
  `id`. Index is `0..249999`; ID is valid UTF-8 of at most 512 bytes, with an
  empty ID valid for an actual no-ID page. Null, absent, duplicate, unknown,
  case-folded, wrong-type, and out-of-range values are refused before payload
  read/copy, package lookup, or launch.
- Preserved `Runner.Export(ctx,id,Source)` and added only owned
  `Source.Page`. `Runner.Export` copies the page value before any admission
  hook, package lookup, child start, or source write.
- Added the one-shot `levnote-drawio-svg` worker. It resolves only
  `Pages[index]`, requires the exact ID, parses without rewriting, and exports
  the requested page through the accepted bounded SVG emitter.
- Parser `mxfile`-wide issues now use `DocumentPage = -1`; selected-page issues
  retain their real page and cell context. Page-local parser and string/order
  behavior is otherwise unchanged.
- Worker warnings are document/page scoped with coarse whole-source spans.
  Parser warnings are bounded before export and precede selected-page emitter
  warnings. More than 256 warnings or an oversized code/message/cell component
  refuses with `warning-limit`; component strings are not truncated and a huge
  combined message is not allocated just to discover the overflow.
- Parse, page, warning, and output failures return an empty nonnil warning array
  and nil payload. `invalid-page`, `source-limit`, `invalid-utf8`,
  `output-limit`, `warning-limit`, and render failures remain coherent protocol
  errors.
- Package selection schema 1 still requires exactly the old three entries and
  maps Draw.io to bundled in memory without writing on load. Explicit
  persistence writes schema 2 with exactly four entries. Corrupt-state
  blocking, explicit repair, installed selection retention, poison/cleanup
  behavior, and the per-engine 12-package limit are unchanged.
- Added the fourth fixed build definition and updated scope wording in the
  packaging script. The script was statically parsed but not executed.

## Files changed

Production/source:

- `native/core/enginecatalog/catalog.go`
- `native/core/engineipc/protocol.go`
- `native/core/engineipc/json.go`
- `native/core/engineipc/decode.go`
- `native/core/engineruntime/types.go`
- `native/core/engineruntime/run.go`
- `native/core/engineruntime/runner.go`
- `native/core/engineruntime/package_types.go`
- `native/core/engineruntime/package_state.go`
- `native/formats/drawio/parse.go`
- `native/formats/drawio/issue_scope.go`
- `native/cmd/levnote-drawio-svg/main.go`
- `native/scripts/package-markdown-worker.ps1`

Tests:

- `native/core/enginecatalog/catalog_test.go`
- `native/core/engineipc/protocol_test.go`
- `native/core/engineipc/drawio_request_test.go`
- `native/core/engineruntime/runner_test.go`
- `native/core/engineruntime/drawio_runtime_test.go`
- `native/core/engineruntime/package_store_test.go`
- `native/formats/drawio/issue_scope_test.go`
- `native/cmd/levnote-drawio-svg/main_test.go`

## Evidence and results

All raw logs are under
`native/handoffs/drawio-worker-stage1-evidence/`. Earlier logs were preserved
and were not rerun for presentation.

- `compile-milestone-1.log`: exit 1. A compile-only pass found three existing
  positional `RequestMeta` test literals after the additive `Page` field.
  Those test literals were changed to named fields.
- `focused-1.log`: exit 1. Exposed test-fixture XML escaping, parser warning
  wording/count assumptions, and nil-payload expectations; no lower-layer
  compatibility defect was established by that run.
- `focused-2.log`: exit 1. Exposed remaining fixture wording and nil-payload
  expectations.
- `focused-3.log`: exit 0 for `core/engineipc`, `cmd/levnote-drawio-svg`, and
  `formats/drawio`.
- `focused-runtime-package-1.log`: exit 1 because the new package test compared
  the zero status slice with the zero array incorrectly.
- `focused-runtime-package-2.log`: exit 0 for the focused runtime/migration/
  package cases after correcting that assertion.
- `final-affected-suite.log`: exit 0. One final affected suite covered
  `./core/enginecatalog`, `./core/engineipc`, `./core/engineruntime`,
  `./formats/drawio`, and all four fixed worker commands:
  Markdown, Mindmap SVG, Mindmap Preview, and Draw.io SVG. Observed package
  times included `core/engineruntime` 38.381s and Markdown 1.445s; the remaining
  packages were cached or under 2s. No application package build was run.
- `final-affected-vet.log`: exit 0 with no diagnostic output for the same eight
  affected packages.
- `final-package-script-parse.log`: PowerShell AST parse, 0 errors, exit 0.
  This validates syntax only; the packaging script was not executed.

The decisive tests assert:

- independent literal bytes for all three old requests and Draw.io's
  `{index,id}` request, plus exact old-engine `page` rejection before payload;
- strict Draw.io metadata key/type/null/duplicate/range/UTF-8/ID handling and
  exact page/IP copy ownership before the admission hook;
- invalid runtime options return `FailureProtocol` without calling
  `Source.Data`, hooks, package selection, or launch;
- exact second-page SVG bytes, source bytes unchanged, ordered document plus
  selected-page warnings, no other-page warning leakage, exact-index/ID
  refusal, compressed read-only success, output/warning limits, protocol
  errors, and no payload allocation for output failures;
- parser page-local issue attribution and explicit document scope;
- schema 1 no-write migration, schema 2 four-entry persistence/reopen, exact
  wrong-count refusal, invalid legacy replacement refusal, corrupt reset, and
  12-package retention.

## Compatibility and limitations

- `ProtocolVersion` remains 1; engine bundle manifest schema remains 1; the
  exact old three request JSON/frame bytes are unchanged.
- Package selection state is migrated to schema 2 only when an explicit
  selection operation persists it. Loading schema 1 does not rewrite the file.
- The worker receives decoded UTF-8 text through the existing source snapshot.
  It makes no promise about the original disk encoding or BOM, and this stage
  does not claim App-level byte/BOM/undo/dirty preservation. Those remain
  Stage 2 App integration assertions.
- The worker source performs no filesystem or network calls. The OS process is
  not sandboxed, and this result makes no memory-peak or physical-input claim.
- The app still uses the existing Draw.io host-side export path in this
  intermediate stage. Draw.io does not advertise this worker, has no source-mode
  worker policy yet, and has no packaged fourth-worker/real-run evidence. Those
  are Stage 2.
- The package manager exposes four fixed statuses, but this stage does not add a
  Draw.io selector or alter App package UI. Stage 2 owns App exposure.
- The existing 30-second runner timeout is unchanged. No 32 MiB real-worker run
  was attempted in Stage 1.

## Frozen hashes

SHA256, byte length, and path are also preserved in
`native/handoffs/drawio-worker-stage1-evidence/final-hashes.txt`, whose own
SHA256 is
`072828726C48B3BBE390B80FE5BFD0707955307B1CDFA72038CFA22FC2E2AEBE`.

| SHA256 | Bytes | File |
|---|---:|---|
| `196794C9FC631C4385607EEE0EEA8FDD4C9FFFFD74E19DCA9B3868D3B23AA91E` | 2534 | `native/core/enginecatalog/catalog.go` |
| `3A13E877754ACA803E4853E1938C81C3BCC16739F065109978FE179C65B8B0A9` | 3144 | `native/core/enginecatalog/catalog_test.go` |
| `6196BF4D35C733F4D1DB048293F19E3A55A7B50EB3FCE8A2B6405D5F2A41ED77` | 8944 | `native/core/engineipc/protocol.go` |
| `E6560E806C9314F94174C2B85FF75D1ABF2F8362B254FFE0DDCFBD44015C7157` | 5405 | `native/core/engineipc/json.go` |
| `1F3547F2F3C2BBF8D6705928F42CC7DE1C87F7A852F93ACA3E02E822F760E258` | 3431 | `native/core/engineipc/decode.go` |
| `582F9CEFC4DBB2D719615F710C5D7EC079280186A800D20488AD891F8B3CDBEF` | 27264 | `native/core/engineipc/protocol_test.go` |
| `0CEB33D43833C74FC871A373CDF4EFD137BD62AC1213328A4537E00E9AA7ED2A` | 6799 | `native/core/engineipc/drawio_request_test.go` |
| `B602AA4FFD2317D902E68D0F9C10C3363498E3AC4600D56C577C4B18BAFF7143` | 2409 | `native/core/engineruntime/types.go` |
| `BC54377FB32CFB3D6A9B7EEB298BF34C775D2DC349D1394C5D340347F0C5BBBC` | 7703 | `native/core/engineruntime/run.go` |
| `E917779FE5768984CD11C288CA150F5FF0E74D9F6B214599B914877581AEBCD6` | 5668 | `native/core/engineruntime/runner.go` |
| `4FF2C6E081F8ED30F529DE3AFD4F8B32EF1BFD5E7F81770D4B6A1F1F62778038` | 4128 | `native/core/engineruntime/package_types.go` |
| `6DFD18C9FF5D5F50C6579F780E7E080633A7A7D5BB89195D72A07C4FC27A485B` | 9472 | `native/core/engineruntime/package_state.go` |
| `3D78EAF829A213E3AEFDBF6D7AE68E2C6E8510382E3F7CA232431B60170000F0` | 23807 | `native/core/engineruntime/runner_test.go` |
| `7500FF4E5947130F814173D66B332E4C12DFACAD4AD8C11AEC89D1FCD565EE3E` | 2580 | `native/core/engineruntime/drawio_runtime_test.go` |
| `242CC3C2E506FDDE59E70598CF39A342669E7BF9C86D9A3C5751BFFA87433C4E` | 28928 | `native/core/engineruntime/package_store_test.go` |
| `DAB4949F7EDEA5593AB7E9F7A5E6AF9F845BF10F19D3E07CA94982AF0B946D25` | 33375 | `native/formats/drawio/parse.go` |
| `1E5B09BA28387D1FA7703E67841855A19DB724BC984604B9654755544324EC8C` | 166 | `native/formats/drawio/issue_scope.go` |
| `1B08250018178F5D5E7E4CDB873EF0880D88DAE11ACAB362E2DF0501C5F4D0BC` | 1217 | `native/formats/drawio/issue_scope_test.go` |
| `B07F8B99701BC6F942238425586FAC02AAA0DAD4C60A737A1B46C7C0C89D6FC1` | 6293 | `native/cmd/levnote-drawio-svg/main.go` |
| `4BE3F6B7A5FD8526D22C39C266C45B1309C953091AF82F62A5E8A283630CD318` | 13375 | `native/cmd/levnote-drawio-svg/main_test.go` |
| `E888150C909962EE8F65F4EA536F5BAA9AF1B1D69AA91E484F53BF72FD98A0F3` | 9323 | `native/scripts/package-markdown-worker.ps1` |

Product source and tests are frozen at these hashes pending independent review.
