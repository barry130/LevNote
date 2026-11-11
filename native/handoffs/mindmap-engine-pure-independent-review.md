# Mindmap pure extraction independent review

Status: COMPLETE 2026-11-09. This is a read-only source review of the frozen
final pure-extraction implementation; I did not run tests, builds, processes,
launches, or network operations, and did not edit implementation/evidence.

## Finding

**No actionable findings. The one P1 I initially reported is withdrawn.**
The proposed failing path assumed `encoding/xml.EscapeText` would replace the
budgeted writer's refusal with `io.ErrShortWrite`. The pinned Go source proves
that assumption false. At
[native/tools/go/src/encoding/xml/xml.go:1957](/F:/AIproject/LevNote/native/tools/go/src/encoding/xml/xml.go:1957),
every internal `w.Write` error is returned unchanged; the tail write does the
same at line 1965–1966. Likewise,
[native/tools/go/src/fmt/print.go:214](/F:/AIproject/LevNote/native/tools/go/src/fmt/print.go:214)
propagates the single `w.Write` error unchanged at lines 217–218.

The concrete over-budget path is therefore:

1. `svgWriter.reserve` refuses the fragment and sets
   `w.err = ErrSVGOutputLimit` at
   [native/formats/mindmap/svg.go:59](/F:/AIproject/LevNote/native/formats/mindmap/svg.go:59).
2. `svgWriter.Write` returns `(0, w.err)` — the same sentinel — at
   [native/formats/mindmap/svg.go:66](/F:/AIproject/LevNote/native/formats/mindmap/svg.go:66).
3. `xml.EscapeText` (and similarly `fmt.Fprintf`) returns that writer error
   unchanged.
4. The assignment at
   [native/formats/mindmap/svg.go:206](/F:/AIproject/LevNote/native/formats/mindmap/svg.go:206)
   stores `ErrSVGOutputLimit` again, already matching `dst.err`; it does not
   replace it with another error.

Thus `ExportSVG` returns `(nil, ErrSVGOutputLimit)` as required. There is no
concrete failing path for escaped text or `printf` fragments. A boundary
regression can still pin this stdlib propagation contract, but it is guarding
an existing correct behavior, not fixing a product defect.

## Source reviewed

- Current: `native/formats/mindmap/presentation.go`,
  `native/formats/mindmap/svg.go`,
  `native/formats/mindmap/presentation_test.go`,
  `native/formats/mindmap/svg_test.go`,
  `native/modules/graphview/mindmap_svg.go`,
  `native/modules/graphview/color.go`,
  `native/modules/graphview/render_cache.go`,
  `native/modules/graphview/sync.go`,
  `native/modules/graphview/adapter.go`,
  `native/modules/graphview/mindmap_presentation_test.go`.
- Retained pre-extraction evidence:
  `native/artifacts/acceptance/mindmap-pure-20261109-060918-baseline-attempt01/`,
  including `before-mindmap_svg.go`, `before-color.go`,
  `before-render_cache.go`, `before-sync.go`,
  `before-mindmap_presentation_test.go`, and `before-export_safety_test.go`.
- Final evidence metadata and recorded streams:
  `native/artifacts/acceptance/mindmap-pure-20261109-061746-owned-final-attempt03/`.
- Fixture byte identity was checked read-only: the four current
  `.mindmap`, `.svg`, and `.json` fixture sets exactly match the retained
  baseline hashes in `baseline-manifest.json`.
- I also verified that the 56 entries in the final `source-after.json` match
  the current files by SHA-256; I did not re-run the recorded gate.

## Behavioral comparison

- Native dimensions remain exactly the required value options at
  [native/formats/mindmap/presentation.go:23](/F:/AIproject/LevNote/native/formats/mindmap/presentation.go:23),
  while generic zero-value `LayoutOptions` behavior is independently checked.
- `DisplayLabel`, `DisplayLabelTruncated`, and `LiteralColor` preserve the
  old 512-byte rune-safe display policy, ordered/duplicate icons and tags,
  64-byte color admission, named/hex/transparent/none semantics, and warning
  order/count/content. They do not mutate the document.
- `PresentationIssues` reproduces the old image/long-label/unsupported-color
  warnings in the same order and preserves parse/layout issues separately.
  The current corrected test distinguishes nil from an empty warning slice.
- The SVG geometry, multiline layout, label/icon/tag rendering, collapsed
  indicator, image placeholder, color fallbacks, XML escaping, and numeric
  formatting match the retained `before-mindmap_svg.go` implementation.
- The count-then-fill writer checks each fragment before counting or copying.
  It stops emission promptly at loops and the nested text writer, returns nil
  bytes on error, and uses an exact allocation after the count pass.
- `mindmap_svg.go`, `literalGraphColor`, `boundedMindmapLabel`, sync layout,
  and warning wiring are thin delegates to the shared pure implementation.
  The canvas does not invoke full SVG assembly during drawing or idle frames.
- Existing render-cache limits and idle semantics remain unchanged; the
  current graphview regression checks no reparse/re-layout/re-shaping on
  subsequent idle frames.
- No new product dependency was introduced by this extraction:
  `golang.org/x/image/colornames` was already used by the old graphview color
  helper and only moved into the pure package.

## Oracle and review limits

- The final artifact reports `test` and `vet` exit 0 and unchanged source
  inventories; this review relied on those recorded streams and did not rerun
  them. The reported aggregate is not independently recomputed here.
- The four retained fixtures cover right/left geometry, Unicode, multiline,
  markers/tags, literal and unsupported paints, negative explicit bounds,
  collapse/image, blank and long labels. They are byte-identical to the
  retained baseline, but they cannot exhaust all source shapes or all output
  budgets.
- Independent literal/XML assertions cover geometry, fallback dimensions,
  escaping, duplicate/empty metadata, hidden-child omission, and unsafe-paint
  omission. Existing graphview tests retain source/history/selection
  nonmutation and idle-cache behavior.
- The negative-bounds fixture intentionally supplies an explicit `LayoutResult`
  because the source has an unknown layout; this validates the exporter's
  negative `Bounds` handling, not a change to `Layout`'s general bounds policy.
  The normal left/right fixtures do compare native `Layout` bounds.
- I found no unresolved concrete question beyond the P1 finding.
