# Fourth worker stage1 independent review

Status: complete, 2026-11-10. No remaining blocker found in the reviewed stage1
lower layers after reconciling the frozen warning correction.

This review is based on the original frozen stage1 task/result/evidence and the
separately frozen warning-correction result/evidence. I did not rerun tests,
build, launch, or modify product source. The active view-contract work is a
separate review and is not assessed here.

## Concrete finding and resolution

### Original stage1 warning intermediates were unbounded before conversion

Location: `native/cmd/levnote-drawio-svg/main.go`, in the original stage1
warning-conversion path.

Trigger: a source with many issues on unselected pages, or more than 256
relevant document/selected-page issues, caused the new IPC conversion layer to
allocate intermediate capacity proportional to all parsed issues before it
filtered or rejected them. The wire contract can publish at most 256 warnings,
so those intermediates were avoidable new worker-only storage growth.

Minimal repair: count only document/selected parse warnings and refuse excess
before allocating conversion storage; give emitter conversion only the
remaining combined capacity and check standalone conversion count before
allocating. Parser/emitter internals and all other stage1 behavior stay
unchanged.

Resolution: corrected and frozen in
[drawio-worker-stage1-warning-correction-result.md](drawio-worker-stage1-warning-correction-result.md).
The corrected worker counts relevant parse warnings first, allocates only after
the count is bounded, passes the remaining capacity to emitter conversion, and
checks `wireWarnings` before allocation/conversion. The narrow tests cover a
large unselected population, 257 relevant issues, zero remaining emitter
capacity, exact 256 and 257 combined counts, and existing order/scope behavior.

This was the only concrete stage1 defect found. It is corrected and is not a
remaining acceptance blocker.

## Hash and evidence reconciliation

- At the start of review, all 21 entries in the original
  `drawio-worker-stage1-evidence/final-hashes.txt` matched their recorded
  SHA256 and byte length.
- After the frozen warning correction, the original manifest remains unchanged
  and its own SHA256 is still
  `072828726C48B3BBE390B80FE5BFD0707955307B1CDFA72038CFA22FC2E2AEBE`.
  The original result also remains unchanged at SHA256
  `C476EF99BBAE3F8E7145274BEE0F1F9A7AC0AAA1FAF321D9C5E453F7308BF3BA`.
- Of the original 21 entries, the 19 non-worker entries still match exactly.
  The two expected exceptions are the correction-owned worker files:
  - `native/cmd/levnote-drawio-svg/main.go`
    `1B8EA397373D1CD3A2CC6F09D4FA6B9EA27E81B6F2FC0241722E7D7D4B85BE20`,
    7,228 bytes.
  - `native/cmd/levnote-drawio-svg/main_test.go`
    `06007F93A9CCF105A16943B0F780322D5103266D84808145F2BF28FDE03E88E5`,
    16,601 bytes.
- The correction manifest, focused worker test log, and worker vet log match
  their recorded hashes. The focused worker test and vet both record exit 0.
  No passing suite was repeated for this review.

## Reviewed behavior

Protocol and request admission:

- The old three request metadata byte sequences remain exact and reject any
  `page` field before reading the source payload.
- Draw.io requires a page object with exactly `index` and `id`. The strict
  decoder rejects absent, null, duplicate, unknown, case-folded, wrong-type,
  out-of-range, invalid-UTF8, and malformed entries before source read/copy.
- The runtime copies `Source.Page` into operation-owned storage before the
  admission hook, package lookup, child start, or source write. Invalid options
  return `FailureProtocol` without calling `Source.Data`, running hooks, selecting
  a package, or launching a child.
- Well-formed but absent or mismatched page index/identity fails as
  `invalid-page`; there is no page-zero or ID fallback and no failed payload.

Worker and warnings:

- The worker resolves only `document.Pages[index]` and requires the exact ID;
  source bytes are read as provided, parsed without rewriting, and the selected
  page is exported through the existing bounded SVG emitter.
- Parse failures, page failures, warning-limit failures, and output-limit
  failures produce nil payload and an empty nonnil warning array.
- Parser warnings are checked before export, and document plus selected-page
  warnings precede selected-page emitter warnings. Unselected-page warnings do
  not leak. Raw code/message/cell components are checked before prefix
  concatenation and no oversized component is truncated.
- After the frozen correction, new warning conversion intermediates are bounded
  by the wire warning count. This is a worker-conversion bound, not a
  whole-process memory claim.

Parser and package state:

- Only the two existing mxfile-wide parser issue sites use `DocumentPage = -1`;
  page-local issues retain their real page and cell context, wording, and
  order.
- Schema 1 requires exactly the three legacy engine selections regardless of
  order, maps Draw.io to bundled in memory, and does not rewrite the state file
  during load. Explicit persistence writes schema 2 with exactly four entries.
- Corrupt-state blocking, explicit repair/reset, installed selection retention,
  atomic persistence, poison/cleanup behavior, and the 12-package per-engine
  limit remain present in the reviewed source and tests.

Catalog and packaging script:

- The fourth catalog identity agrees with the result: engine
  `levnote.drawio-svg`, SVG format, protocol 1, Windows amd64,
  `engines/drawio-svg/levnote-drawio-svg.exe`, 32 MiB source and 64 MiB payload.
- The packaging script has the matching fourth fixed build definition. It keeps
  fixed repository inputs, rejects preexisting output/evidence directories, and
  makes no executable/path override or sandbox claim. Its recorded check was
  AST parsing only; it was not executed.

## Review conclusion

The stage1 lower layers are acceptable with the frozen warning correction
applied. No independent protocol, runtime, worker, parser, package migration,
or packaging-script defect remains from this review.

The evidence still does not claim app integration, source-mode policy, App
undo/BOM/dirty preservation, packaged fourth-worker delivery, whole-process
memory bounds, physical UI/input behavior, or OS sandboxing. Those remain
outside this stage.
