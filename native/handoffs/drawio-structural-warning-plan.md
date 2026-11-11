# Draw.io structural-root warning plan

Status: read-only plan, 2026-11-10. No source, test, build, launch, package, or
resource action was performed. The custom-suffix route source and all `.19`
artifacts remain frozen.

## Finding and decision

`native/formats/drawio/parse.go` currently emits `UNSUPPORTED_ELEMENT` for every
`mxCell` without a truthy `vertex` or `edge`. That includes the canonical
container pair used by the sole new-document template:

```xml
<mxCell id="0"/>
<mxCell id="1" parent="0"/>
```

The nodes are structural layers, not missed renderable cells. Suppress the
generic warning only for that exact canonical pair. Do not introduce a general
“root/layer” classification.

Recognition should use both source order and identity:

- The first direct `mxCell` child of `<root>` has `id="0"`, has no `parent`,
  and has no other attributes or content.
- The immediately following direct `mxCell` sibling has `id="1"`,
  `parent="0"`, and has no other attributes or content.
- Both cells are self-closing or otherwise empty apart from XML whitespace and
  comments. They contain no `value`, `style`, geometry, text, or unknown child.
- The pair occurs once, before any other direct `mxCell`. Requiring that
  position keeps reordered or arbitrarily placed containers visible as current
  behavior dictates.

Any deviation keeps the existing warning. This includes `id="2"` or later,
wrong/missing parent links, extra or mis-spelled attributes, labels, styles,
geometry, non-whitespace text, unknown children, reversed order, repeated IDs,
and generic non-vertex cells. Duplicate-ID rejection, ID requirements, parser
budgets, read-only reasons, byte offsets, and all other issue emission remain
unchanged. The parser continues to retain the complete source bytes; this is a
warning-only filter and adds no permissive render/edit model.

## Implementation seam

The minimum production change belongs in
`native/formats/drawio/parse.go`. `parseRootCells` already sees direct-cell
order and has the source start/end offsets needed to establish an empty cell.
It should carry a small parser-local decision for the first two non-renderable
`mxCell` elements and pass an explicit “canonical structural container” result
into `parseCell`, or perform the equivalent suppression only at `parseCell`’s
existing non-vertex/non-edge warning. Keep the pair recognition in one helper
so tests exercise the actual condition rather than string matching.

Do not filter in `native/modules/graphview/sync.go`, `view_ui.go`, or the worker.
Those layers currently forward and summarize real parser issues; hiding
message text there would also conceal malformed cells. No template change is
needed because `native/modules/graphview/adapter.go` already emits the canonical
bytes.

## Regression fixtures

Add or expand parser tests under `native/formats/drawio/`:

1. The exact new-document template and the canonical inline model produce no
   non-vertex `UNSUPPORTED_ELEMENT` issues.
2. The canonical pair remains accepted in a document with valid renderable
   vertices and edges; parse result and source offsets remain exact.
3. Each malformed lookalike still warns: extra `value`, `style`, child,
   non-whitespace text, wrong parent, swapped order, IDs other than `0`/`1`, or
   a third non-vertex cell.
4. A real unsupported non-vertex cell after a valid pair still warns once with
   its own cell ID/page.
5. Duplicate `0`/`1` IDs still fail through the existing duplicate-ID path,
   independently of warning suppression.
6. Round trip/source snapshots show identical bytes and edit offsets.

Update worker literal expectations only in the same accepted future source
slice. In particular, selected-page warning tests and the compressed-page test
in `native/cmd/levnote-drawio-svg/main_test.go` currently count the two
container warnings. Their expected counts should decrease only after the new
parser behavior is frozen. Keep a selected-page fixture with a genuine issue on
the selected page and a larger issue population on an unselected page, proving
that filtering still excludes other pages before warning limits are applied.
Keep exact-limit and over-limit cases, including real issues adjacent to the
suppressed containers.

Only a narrow graphview/template assertion is needed if existing template tests
cannot otherwise prove the clean document surface: the canonical template must
continue to match and render while exposing no container warning. Do not add a
new UI warning framework or suppress all `UNSUPPORTED_ELEMENT` codes.

## Historical evidence and sequencing

The `.19` second-page smoke correctly records the parser behavior of that
immutable binary: two structural-container warnings. Existing `.19` and staged
`.20` logs, images, manifests, and handoffs must not be rewritten, and those
passing runs must not be relabeled as failures. A later clean result may have a
different warning list only after this parser change is implemented, reviewed,
accepted, and recorded in fresh evidence. Implementation should wait for the
current `.20` freeze and an explicit ownership assignment.
