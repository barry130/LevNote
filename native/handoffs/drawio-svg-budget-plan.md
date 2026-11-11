# Draw.io SVG output budget plan

Status: READ-ONLY PLAN delivered 2026-11-10. No source, test, build, capture,
launch, or measurement was performed for this handoff. Whole-word Find and
release .17 remain owned by the active delivery task.

## Founding evidence

`native/formats/drawio/svg.go:11` exports one selected page into an unbounded
`bytes.Buffer`. Its final `svg.Bytes()` is therefore only as small as the SVG
expansion happens to be. Parser limits in `parse.go` bound source input
(`DefaultMaxBytes = 32 << 20`), decompressed XML, elements, depth, attributes,
and text; none bounds generated SVG. In contrast, the accepted mindmap exporter
at `native/formats/mindmap/svg.go:12` establishes the needed local pattern: a
64 MiB default/hard cap, a sentinel error, a counting pass, an exact-size fill
pass, and no partial payload on failure.

The only production callers are:

- `native/formats/drawio/drawio_test.go:223`, using zero-valued options.
- `native/modules/graphview/graphview_test.go:207`, using zero-valued options.
- `native/modules/graphview/adapter.go:301`, passing
  `drawio.SVGOptions{}` and the already-selected `v.pageIndex`.

No multi-page, parser, geometry, graphcanvas, or graphview page-selection change
is part of this slice.

## Exact API and limit contract

In `native/formats/drawio/types.go`, extend `SVGOptions` with:

```go
MaxBytes int
```

and add a public sentinel beside the other draw.io errors:

```go
ErrSVGOutputLimit = errors.New("draw.io SVG output exceeds byte limit")
```

In `native/formats/drawio/svg.go`, define `const MaxSVGBytes = 64 << 20`.
Keep `Document.ExportSVG(pageIndex int, options SVGOptions) ([]byte, []Issue,
error)` unchanged.

- `MaxBytes == 0`: use `MaxSVGBytes`.
- `0 < MaxBytes <= MaxSVGBytes`: use the requested lower cap.
- `MaxBytes > MaxSVGBytes`: clamp to `MaxSVGBytes`; callers cannot raise the cap.
- `MaxBytes < 0`: return `nil, nil, ErrSVGOutputLimit`.
- An output whose exact byte length exceeds the resolved cap returns
  `nil, nil, ErrSVGOutputLimit`; never return partial SVG or an incomplete issue
  sequence.
- A successful export preserves the current exact bytes and issue order.

Page lookup remains first, preserving the existing precedence for a missing
page. No source, `Document.Source`, page model, or unknown XML metadata is
mutated.

## Smallest implementation

Use the mindmap pattern locally in `svg.go`; do not extract a shared framework.

1. Replace the direct `bytes.Buffer` target with a private `svgWriter`
   containing `data []byte`, `limit`, `written`, and `err`. A nil `data` slice is
   count mode; a non-nil slice is fill mode. `reserve`, `Write`, `text`, and
   `printf` must reject a fragment before advancing if
   `size > limit-written`.
2. Change the local SVG helpers `writeShape` and `writeSVGText` to accept
   `*svgWriter`, and change edge output to write path coordinates directly.
   `edgePath` already computes the point sequence but currently allocates the
   whole path string. Preserve its exact `M… L…` spelling while emitting each
   admitted coordinate to the writer.
3. Add an SVG-local escaped-attribute writer that passes through `xml.EscapeText`
   into `svgWriter`. Keep `edit.go:357 escapeAttr` unchanged because XML edit
   production uses it. SVG labels similarly pass through `xml.EscapeText`
   directly; do not build an escaped whole-label string. Iterate text lines
   without `strings.Split`, preserving the current output for a trailing newline
   and the existing `dy`/`tspan` bytes.
4. Count pass: run the complete emitter with `svgWriter{limit: limit}` and
   collect issues. Every fixed tag, option, number, newline, escaped attribute,
   escaped label byte, and generated path fragment must go through the same
   writer checks.
5. Fill pass: allocate exactly `make([]byte, count.written)` and rerun the same
   emitter with that slice. Suppress issue collection on the fill pass and return
   the count-pass issues, preserving warning order. If the fill length differs,
   return nil output and an internal export error.

Both passes operate on the same immutable `page` and normalized options.
`graphview.adapter.go` needs no production edit: zero-valued options mean the
64 MiB default, and its existing `v.pageIndex` continues to choose the page.

## Verification

Before changing the emitter during implementation, capture the current
`richDrawio` default SVG as a pre-change oracle: exact bytes plus exact
`[]Issue` values. Put the compact rich fixture's exact expected byte string (or
an equally reviewable literal oracle) in the test, rather than duplicating the
exporter in test code.

Tests in `native/formats/drawio/drawio_test.go`:

- Default and option normalization: zero resolves to `MaxSVGBytes`, values above
  it clamp to the hard cap, and negative fails with `ErrSVGOutputLimit`.
- Successful output after the refactor byte-for-byte matches the pre-change
  oracle, including Unicode and XML escaping, and warnings match in exact order.
- Export once to obtain a small fixture's `want`; `MaxBytes == len(want)`
  succeeds with exactly `want`, while `MaxBytes == len(want)-1` returns nil
  output/issues plus the sentinel error.
- A label containing many newline-separated lines crosses a deliberately small
  cap because each line emits a `tspan`; the failure proves amplification is
  counted rather than bounding only source text.
- The same tests cover escaped `FontFamily`/background attributes and a generated
  multi-point edge path.
- A two-page fixture exports the requested page only; success and limit failure
  leave `Serialize()` bytes equal before export.

One small `native/modules/graphview` regression should exercise
`view.Export("svg")`, verify it exports the current `v.pageIndex` with unchanged
default bytes, and verifies the selected page and source remain unchanged. No
new page-selection behavior is added.

After substantive corrections, run exactly:

```powershell
.\native\scripts\go.ps1 -GoArguments @('test','./formats/drawio','./modules/graphview','-count=1','-timeout=180s')
.\native\scripts\go.ps1 -GoArguments @('vet','./formats/drawio','./modules/graphview')
```

No build is required. A one-off `testing.B` allocation observation is optional
only if a concrete allocation claim needs evidence; it may describe bytes/op for
this function and its named fixture, never process RSS or whole-application
memory.

## Forward dependency

`native/core/engineipc/protocol.go:40` currently carries only `Engine` and
`Format`, while Draw.io export depends on the selected page. A later worker
extraction must carry validated page identity and enforce host
version/lifecycle checks. It must not silently default to page 0 or rewrite the
source merely to export one page. That protocol/work item is outside this
output-bounding slice.
