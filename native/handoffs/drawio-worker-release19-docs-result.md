# `.19` release documentation result

Date: 2026-11-10. This documents the accepted immutable
`native-worker.20261110.19` package only and freezes the seven touched
documentation files for root acceptance. No product code, examples, portable
assets, accepted packages or direct evidence were modified.

## Scope and changes

- Updated the root release overview, roadmap and changelog for the current
  `.19` package, the fourth fixed draw.io SVG worker, current-page export
  guidance, four-engine package semantics and the accepted delivery evidence.
- Updated `native/README.md` for the `.19` launch path, four-worker build,
  current-page draw.io requirement and `.1`–`.18` historical-package boundary.
- Updated `native/ARCHITECTURE.md` for the fourth worker, `RequiresView`,
  `RequestPage`, page/token/preparation and publication guards, the accepted
  `.19` build/run evidence, schema1/schema2 package-selection semantics and
  the retained historical `.18` package.
- Added the accepted `.19` validation record with package identities, five
  GUI builds, the one installed-engine request, the separate hidden host run,
  synthetic-versus-actual coverage and explicit unverified physical/resource
  claims.
- Corrected `native/formats/API.md` because its previous three-engine
  host-side draw.io wording was false for `.19`. It now documents the four
  fixed engines, `DrawioSVG`, `RequiresView`, `Source.Page`, required page
  metadata/bounds and schema1/schema2 package semantics.
- Applied root review notes 1–4 before freeze: `startEngineExport` owns the
  initial owner capture and Flush; `PrepareExportRequest` only synchronizes a
  stale parsed cache and reports page/epoch; pending-draft Flush intentionally
  changes Buffer/dirty/history while clean export preserves those in-memory
  states; the fourth bundled path is included; the no-flatten claim is scoped
  to snapshot transfer. Draw.io's required page object permits an empty ID only
  when the selected source page truly has no ID.

## Final file identities

| File | Bytes | SHA-256 |
|---|---:|---|
| `README.md` | 17,693 | `0131B1654867230325E6DF2C7E39A477D9BFEA978C88540B3DDC8A3A0544386C` |
| `ROADMAP.md` | 119,176 | `78F0045E505F1EC3DCFBC0FEB1C301EEA6C01F87B297C50AFDBFFAA4BEA6CD26` |
| `CHANGELOG.md` | 144,155 | `8A641C9F1CADD5C484C91052CFA56DA21703DA62BB6C318A4531EC253C1735DC` |
| `native/README.md` | 59,126 | `8E1F0D8A990EA82D0B10FD5EAC74B345BEA55CBF3B8E78875F41CF31D23634BA` |
| `native/ARCHITECTURE.md` | 62,407 | `6E793D4D3D4FDF581D4D635024657006FB2D19D6D86271940A59EB3E87A3A08E` |
| `native/VALIDATION.md` | 137,743 | `4A13A19423DA5F773001A1D0A3D6AE4F466670808CA61444FD1213CD5C28A1B7` |
| `native/formats/API.md` | 53,097 | `1E4114CDC1A618E9FCFFD007388AA93961741D6B7B77BA445FAE02C752B4416F` |

## Link check

One narrow local-link check covered all Markdown file links in the seven
touched documents: 347 targets, all existing. No inherited link outside the
touched content was reported missing.

The first invocation of the checker had a path-resolution bug and produced no
valid result; it was corrected before the valid pass above. No application,
test, build, capture, resource measurement or other execution was run.

## Limitations and source boundary

This is a documentation-only reconciliation. It reuses the accepted `.19`
package and source evidence; it does not re-execute or revalidate the product.
The documented limits are protocol/output/request bounds, not process peak
memory or speed guarantees. Raw draw.io XML parsing is bounded but allocates.
The accepted runs do not establish physical UI/IME, dirty recovery, process
memory/CPU, latency or packaged interactive export behavior.

The parallel custom-suffix source work that adds `builtin.drawio` to schema3
is not part of the immutable `.19` binary. These documents keep the accepted
`.19` schema3 routes as `builtin.markdown` and `builtin.mindmap`, and do not
claim examples, portable assets or API route extensions from that separate
source task.

Frozen for root acceptance. This handoff does not self-accept.
