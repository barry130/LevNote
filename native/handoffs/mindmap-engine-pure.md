# Mindmap pure presentation and bounded SVG

Status: implementation and owned checks complete; product/test source frozen for
root acceptance. No app, worker, IPC/runtime, build/package or process/UI launch
was performed in this gate. The current `.5` delivery remains unchanged.

## API and wiring

`formats/mindmap/presentation.go` now owns:

```go
func NativeLayoutOptions() LayoutOptions
const MaxDisplayLabelBytes = 512
func DisplayLabel(node *Node, fallback string) string
func DisplayLabelTruncated(node *Node) bool
func LiteralColor(value string) (string, bool)
func PresentationIssues(doc *Document) []Issue
```

Native options are returned by value: height40, horizontal72, vertical24,
padding28, width92–380, character8 and line20. General `Layout` defaults remain
unchanged. Label assembly retains icon/tag order, duplicates and empty entries;
only the UTF-8-safe512-byte prefix plus ellipsis is built. Color admission is
the previous64-byte hex/named/none/transparent subset. UI color conversion stays
in graphview, so draw.io retains its existing admission behavior.

`PresentationIssues` is separate from SVG and does not build labels. It visits
hidden descendants too, returning image-placeholder, label-truncated and
unsupported-color warnings in the original order and wording. Codes are
`IMAGE_PLACEHOLDER`, `LABEL_TRUNCATED`, `UNSUPPORTED_COLOR`; severity is warning
and NodeID is empty because these are document-wide counts/diagnoses. Parse and
layout issues remain separate. The existing graphview `addWarning` continues
message deduplication. Nil document/root returns no presentation issues.

`formats/mindmap/svg.go` owns:

```go
const MaxSVGBytes = 64 << 20
var ErrSVGOutputLimit error
func ExportSVG(doc *Document, layout LayoutResult, maxBytes int) ([]byte, error)
```

Zero uses64MiB; positive limits clamp to64MiB; negative limits return
`ErrSVGOutputLimit`. A missing document/root retains the readable error
`mindmap is not readable`. The shared emitter counts with remaining-budget
checks, stops on refusal, then fills one exact-size result. Every error returns
nil bytes. This bounds output, not the document/layout or total process memory.
Callers must not mutate the document or layout concurrently. Existing numeric
formatting, XML text escaping, geometry, fixed fonts/paints and placeholders
remain byte-identical. Admitted colors are literal hex and cannot need XML
attribute escaping; user text still goes through `encoding/xml.EscapeText`.

Only four graphview product files changed: `mindmap_svg.go` delegates export,
`render_cache.go` delegates Mindmap labels, `color.go` delegates literal
admission, and `sync.go` uses shared native options and presentation issues.
No SVG generation occurs in Draw/sync. The UI text cache remains in graphview.
The old long multiline escaping test and split benchmark moved with their
helpers to pure tests; native color, metadata-edit, marker-toggle/clear and
large-label tests remain. The benchmark was not run in this gate.

## Retained baseline and meaningful checks

The original implementation and capture harness are retained under
[`mindmap-pure-20261109-060918-baseline-attempt01`](../artifacts/acceptance/mindmap-pure-20261109-060918-baseline-attempt01).
Before product edits, the explicit capture test passed (graphview0.968s).
Its12 source/SVG/Bounds+Warnings files were copied unchanged into
[`testdata/presentation`](../formats/mindmap/testdata/presentation/README.md).
The ordinary tests can only read these files, not regenerate expected output.

| Original fixture | SVG bytes | Original and current SVG SHA-256 |
| --- | ---: | --- |
| right-rich |1680|60D3122C086D9B1EA702516B924B2F975CE46C9B81F2AB2E4D2B45A3ED6755E4|
| left-plain |1677|918F60DBC2FDF39BE76FA8FA224714541B67252FC33C63947CD833697DC1A4A5|
| negative-unsafe |954|A516029166E17B357371D77EC9431CC0700CB5B8DCE2FCD0A733E72879AF0998|
| blank-long |1731|90DF679E350DF6CBABAEE196133BF036745B904C57BC4B7670E15BFF58500B9B|

`TestSVGPreExtractionFixtures` and
`TestMindmapPresentationRetainedBaselineAndIdle` independently use those old
bytes/metadata. They cover left/right/org/unknown layouts, Unicode/multiline,
duplicate/custom icons and tags, literal/unsafe paint, collapsed/image nodes,
negative bounds and long labels. The native regression also checks shared
layout options, stable parsed/layout/draft identity, no reshaping or pending
frame at idle, and unchanged source/version/history/selection.

Additional pure oracles cover exact native options and caller-local mutation,
unchanged general defaults,512-byte/rune/combined punctuation boundaries,
literal color outputs and rejected fallbacks, nil guards, hidden-descendant
warning counts/order, exact independent viewBox/background/escaping labels,
blank lines and long Unicode escaping. SVG limits at exact output−1/exact/+1,
tiny/negative/default/hard/overhard limits verify complete output or nil error
payload, exact returned capacity, and sticky refusal. Existing native huge
unknown glyph arrays and oversized tag tests continue to verify bounded
presentation and truthful warnings without source mutation.

## Actual checks and preserved failures

All commands ran through the repository wrapper with explicit argument arrays;
child environments cleared `LEVNOTE_*` and GOFLAGS. No arbitrary go binary or
installed executable was used.

1. `mindmap-pure-20261109-061537-focused-attempt01`: fmt refused named files
   from two directories. No tests ran. The next attempt split fmt by directory.
2. `mindmap-pure-20261109-061603-focused-attempt02`: graphview1.219s PASS.
   Pure tests failed only because `reflect.DeepEqual` treated nil and empty
   warning slices as unequal for the warning-free fixture. Actual warning
   contents were both empty. The failed test source/logs remain. Changed only
   that oracle to `slices.Equal`; no product change followed this attempt.
3. [`mindmap-pure-20261109-061746-owned-final-attempt03`](../artifacts/acceptance/mindmap-pure-20261109-061746-owned-final-attempt03):
   full owned packages and scoped vet pass.

```powershell
./scripts/go.ps1 -GoArguments @('test','./formats/mindmap','./modules/graphview','-count=1','-v')
./scripts/go.ps1 -GoArguments @('vet','./formats/mindmap','./modules/graphview')
```

Actual package times: mindmap0.567s, graphview6.811s. Outer test8.1423648s and
vet1.0788532s, both exit0; stderr empty. The full graphview package includes its
existing synthetic cache/resource tests; this is not a real process resource
measurement, screenshot recapture or OS input claim.

Final result SHA-256:
`3096002E8F2AC0B9E002928812F84580816C7104E57A6BF454CE98AABB37E24A`.
`source-after.json` SHA-256:
`1265511B7CDC3327B9137C7CB8F3635B0665E409C30BB2CE13E2BE982CCCA897`.
All56 inventoried Go/fixture inputs match before/after execution; all34 existing
non-owned Go files in the two packages match the pre-extraction inventory.
All12 retained fixtures match their original length and SHA-256.

Pure production hashes:

- `presentation.go`: `0F54E697771ED666554ED0C51545025E9FA212886F5249225D0534442EB12296`
- `svg.go`: `106D55A6FDFDB7D3E39C35087E82E802DDF53E6702DE6BD82326AFE5F42FE2D0`

Exact hashes of all tests/wiring are in the final `source-after.json`.
Root assigned an independent read-only source review after the compiling
checkpoint; this report does not assert that review's outcome. No known
implementation or test failure remains; next catalog/worker/runtime/app gate
is outside this completed finite scope.
