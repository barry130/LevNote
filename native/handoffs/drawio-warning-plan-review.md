# Structural-warning plan review

Date: 2026-11-10. Read-only review of
[drawio-structural-warning-plan.md](drawio-structural-warning-plan.md)
against the current parser, graph template, draw.io SVG oracle, parser tests
and selected-page worker tests. No product/docs edits, tests, builds, launches
or measurements were run, and no historical evidence or oracle was changed.

## Decision

The root-approved narrow first step is sound: suppress the generic
non-vertex warning only for the exact ordered first pair
`<mxCell id="0"/><mxCell id="1" parent="0"/>` used by the new-document
template. Do not add general layer/group semantics or suppress other
`UNSUPPORTED_ELEMENT` issues.

The ordered-first-pair policy is a disclosed conservative subset, not a claim
that reordered or later structural cells are invalid draw.io. It deliberately
leaves those visible pending a separate generalization design. Record that
limitation in implementation/review notes; do not silently widen stage scope.

## Minimal implementation ownership

- Production change: `native/formats/drawio/parse.go` only.
- Put recognition in one small parser-local helper used by the actual
  `parseRootCells`/`parseCell` path. Do not add a second decoder or copy the
  whole source.
- Keep duplicate-ID checking in `parseRootCells` exactly as-is: each cell's
  required ID is checked and inserted into the existing per-root `duplicates`
  map before any warning-suppression decision.
- Do not alter graphview synchronization, UI warning rendering, the worker,
  the template, or serialization. Those layers should continue receiving the
  parser's issue list unchanged.

A helper should validate both cells only after both start elements have been
consumed through the existing XML path. The first cell's warning must remain if
the immediately following cell is not the exact `id="1" parent="0"` cell, and
all surrounding warning order must remain relative to existing parser output.
Never pre-commit suppression from the first start tag alone.

## Exact recognition and failure rules

Qualify only when all of the following hold:

- The first direct `mxCell` child of `<root>` has local name `mxCell`, exactly
  the required attributes `id="0"`, no `parent`, no text, comments, processing
  instructions, children, geometry, metadata or namespace declarations, and is
  empty either as a self-closing tag or with no content between explicit open
  and close.
- The immediately following direct `mxCell` sibling is likewise empty and has
  exactly `id="1"` and `parent="0"`.
- Attribute order, quoting style, XML whitespace and self-closing versus
  explicit close do not change qualification.
- The pair is the first two direct `mxCell` children; no prior direct
  `mxCell` exists.

Any deviation retains current warnings. In particular:

- wrong or missing `parent`, reversed order, IDs other than `0`/`1`, extra or
  misspelled attributes, namespace declarations/attributes, `value`, style,
  geometry, text, comments, processing instructions, unknown children or
  metadata;
- a later or repeated `0`/`1` pair, duplicate `0`/`1` IDs, or any third
  non-vertex cell;
- non-XML whitespace or non-whitespace content.

XML comments and processing instructions are already accepted parser tokens
inside cells/root, but this canonical-pair definition must not treat them as
empty content. Keep the existing directive rejection and all text accounting.
Unknown root siblings such as `<future/>` must retain their own warning even
when the pair is clean.

## Budgets and accounting

Do not change normal parser budgets, element/depth/text accounting, duplicate
checks, issue severity, byte offsets, page scope, serialization, or unknown
metadata retention. The pair recognition should consume the same token stream
and use existing offsets; no second pass over the full source is needed.

The worker currently parses the full document, filters issues to the selected
page or document scope, and checks the selected-page count before SVG
 emission. Removing two issues may change exact/over-limit fixture arithmetic;
it does not relax `engineipc.WarningsMaxCount`, component bounds or the
selected-page-first filtering behavior.

## Tests and fixtures

Add/expand parser tests in `native/formats/drawio`:

1. canonical new-document template and inline model produce no structural
   warnings;
2. canonical pair remains clean alongside valid vertices/edges, with exact
   round trip and offsets;
3. malformed lookalikes each retain both/first warnings and relative order:
   extra `value`/`style`, child, text, wrong parent, swapped order, wrong IDs,
   third non-vertex cell;
4. duplicate `0`/`1` IDs still hit the existing `ErrDuplicateID` path before
   any suppression effect;
5. clean pair plus an unknown sibling still emits that sibling's warning;
6. attribute order/quotes/XML whitespace/explicit close variants qualify while
   semantically different variants do not.

Selected-page worker expectations in
`native/cmd/levnote-drawio-svg/main_test.go` need an intentional two-warning
subtraction only in fixtures whose selected page contains the canonical pair.
Preserve exact SVG/source-byte assertions separately from changed issue
expectations. Update:

- `TestDrawioWorkerExportsExactSecondPageAndPreservesSource` (2→0 warnings);
- `TestDrawioWorkerWarningOrderAndScope` (selected-page 2 structural warnings
  disappear, remaining order unchanged);
- `TestDrawioWorkerCompressedReadOnlySuccess` (inner pair 2→0, compressed
  warning remains);
- `TestDrawioWorkerFiltersLargeUnselectedWarningPopulation` (selected page
  2→0 while the unselected page remains filtered before budget checks);
- `TestDrawioWorkerCombinedWarningBound` (250/251 `<future/>` arithmetic must
  be recalculated to preserve exact and over-limit cases);
- output-limit and protocol fixtures that use a clean canonical pair.

Keep the bounded SVG oracle
`native/formats/drawio/testdata/svg-prechange-oracle.json` immutable. If a new
source slice intentionally changes issue expectations, add a fresh oracle or
fixture in that slice; do not rewrite the pre-change oracle. The old SVG bytes
and source bytes remain separate assertions from warning-count changes.

Minimal graphview coverage is needed only if existing tests cannot prove the
new-document clean surface: canonical template still matches/renders with no
container warning. No new UI warning framework is needed.

## Affected final suites

After implementation and corrections, run the substantive focused packages and
one final affected suite:

- `native/formats/drawio`
- `native/cmd/levnote-drawio-svg`
- `native/modules/graphview` only if its assertions are updated or it is the
  final affected app-facing package

Use the fixed `native/scripts/go.ps1` wrapper. Preserve actual command output,
exit codes and any failure; do not run these during `.20` packaging. No
historical `.19`/`.20` result may be relabeled as covering the changed
behavior.

## Sequencing

Implementation may start only after `.20` is frozen and root explicitly assigns
the source slice. Keep this stage warning-only; general layer/group support,
broader root semantics, and route/UI behavior are follow-up designs, not part
of this implementation.
