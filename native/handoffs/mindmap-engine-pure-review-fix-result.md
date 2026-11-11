# SVG escaped-boundary review reproduction

Status: complete, 2026-11-09. The independent P1 is invalid for the pinned Go
implementation and unchanged current SVG writer. No product fix was made.
Root independently read the same toolchain functions and authorized retaining
the regression plus full Mindmap/vet results instead of a defensive rewrite.

The review claimed that `xml.EscapeText` turns the writer's
`ErrSVGOutputLimit` into `io.ErrShortWrite`. The pinned
`tools/go/src/encoding/xml/xml.go:1956–1966` returns the exact `Write` error;
`tools/go/src/fmt/print.go:214–219` does the same. `svgWriter.Write` returns its
stored sentinel on refusal, so the existing assignments do not overwrite it
with a different error. This is corroborated by execution, not only inspection.

Only `formats/mindmap/svg_test.go` changed. New
`TestSVGOutputLimitEveryEscapedBoundary` uses a669-byte single-node SVG with
independently asserted `<`, `&`, `>`, double/single quote, Tab/CR entities,
invalid-XML character replacement, Unicode, ordinary prefix/suffix and a second
line. Every positive insufficient cap1–668 is checked through public ExportSVG
and direct count/fill emitters. Each returns the sentinel, public data is nil,
counts stay within budget, unwritten buffer/canary bytes remain intact, and
subsequent text/printf/multiline/Write calls cannot resume a refused writer.
Exact/full+1 boundaries remain successful. These checks prove output/error and
write behavior; they do not measure process memory or prove allocation bounds.

Actual prechange run:
[`mindmap-pure-review-20261109-062507-prechange-attempt01`](../artifacts/acceptance/mindmap-pure-review-20261109-062507-prechange-attempt01)
passed0.537s, exit0. It retains the unmodified product source, new test source,
commands/streams/timing and result SHA-256
`75D98D15C8F227A2A9BD6B89B9E6747AB381B1A1283F14620B3D8CC25B2478BC`.
There was no red run to report or a product change between red and green.

Actual final run:
[`mindmap-pure-review-20261109-062553-final-attempt02`](../artifacts/acceptance/mindmap-pure-review-20261109-062553-final-attempt02):

```powershell
./scripts/go.ps1 -GoArguments @('test','./formats/mindmap','-count=1','-v')
./scripts/go.ps1 -GoArguments @('vet','./formats/mindmap')
```

Full Mindmap0.691s PASS; outer test1.7625067s/vet0.8718421s, both exit0,
stderr empty. All four retained pre-extraction SVG bytes still match. All56
inventoried inputs remain unchanged during execution;55 prior inputs excluding
the augmented test match the earlier frozen checkpoint. Graphview was not rerun.

- Final result: `4FD82149B567BD848A2167A46A36AEAD46FE5B3A9743647C97FCC2AA136F5BD7`
- Final inventory: `AB4911A1B4958ECA7143920D0E7C08A070D6BC078F4F2AB8F3EB4C7D56EE4345`
- Unchanged svg.go: `106D55A6FDFDB7D3E39C35087E82E802DDF53E6702DE6BD82326AFE5F42FE2D0`
- Augmented svg_test.go: `D76BFB317CD7A20F7078DEF916738F9EABFFA3AE9865604BE51FED27C6DDEBD6`

Original review and previous evidence remain intact as historical records.
This result corrects that review's error-propagation claim. No app/runtime,
catalog/IPC, graphview, packaging, UI/process launch or resource work occurred.
