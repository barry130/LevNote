# Oversized preview boundary checkpoint

## Final15:12 integrated record

DeepSeek full test PASS7.165s wall, full vet clean, release build successful.
Current exe independently verified13,209,600B, UTC2026-11-08T07:12:01.8261018Z,
SHA25636CDC562D0E3C7F1793EDC935154B3AD7FF726716FCC409A736B04BD0FCAF3C6.
Hidden PID-scoped window enumeration/WM_CLOSE smoke exit0 with both unrelated
sentinels preserved and copied hash matching. Focused app tests and corrected
760x480 images passed; exact records are integration.md and
artifacts/acceptance/oversize-preview-20261108-150938/result.json.
Product stays frozen; later sections preserve the focused development evidence.

Acknowledged after root accepted schema2 integration at14:43. Astra owns only
minimal exported graph parser default byte constants, Markdown/graph view
pre-materialization guards and focused lifecycle/allocation regressions.
No parser algorithm or optional feature change; DeepSeek owns app acceptance.

Implementation contract: inspect Buffer.LenBytes before whole String/[]byte or
Markdown begin prefix copies. Cache refusal by Buffer identity/version, clear
obsolete parsed/layout/row/label caches even with an independent dirty draft,
and show concise source guidance. Source bytes/selection/undo stay authoritative.
Safe Markdown range-only Flush remains allowed above its preview budget;
graph Flush must parse to locate a node, so it refuses above the default before
copying and keeps the recoverable draft. Shrink/new Buffer recovers normally.

Planned API: existing markdown.MaxParseBytes remains the single Markdown bound;
mindmap.DefaultMaxBytes and drawio.DefaultMaxBytes expose their current parser
defaults (16MiB/32MiB), which the parsers and views will both use. No limit change.
Focused tests will construct oversize fixtures outside allocation measurement,
cover cold/repeated refusal, stale-cache release, draft/Undo/source preservation,
recovery after shrink and replacement, and export refusal with no output.

## Focused implementation checkpoint — stable for peer/app review

Product changes are limited to the accepted boundaries:
- `formats/mindmap/parse.go`: exported `DefaultMaxBytes = 16 << 20` is used by
  normalized defaults; `formats/drawio/parse.go` exposes/uses `DefaultMaxBytes =
  32 << 20`. Existing `markdown.MaxParseBytes` remains8MiB. No limit/algorithm
  change. Graph outline's copied16MiB constant was replaced by the parser value.
- `modules/markdownview/preview_limit.go`: pre-copy check, cached refusal and
  obsolete preview/row/list/outline release; refresh/export/stale block begin
  call it before whole source/prefix materialization. Independent draft survives.
  If source shrinks with a pending draft, the preview rebuilds without assigning
  that draft to any new row index. Safe field Flush still uses range mapping.
- `modules/graphview/preview_limit.go`: per-kind length check and parsed/cache
  release. ensureSynced checks before String/[]byte; Flush and writable mutation
  paths reject before their additional parses. Export returns refusal with no
  bytes. Pending field survives/recovery is still accessible; clean obsolete
  field is released. Selection/page/pan/zoom are retained.

No new neutral interface or shell size guard is required. Create stays lazy.
Warnings/Draw expose authoritative current-buffer refusal; Export fails with
nil output. Markdown text is “Markdown 超过8MiB预览上限。完整原文仍保留，请切到
源码视图；HTML导出已暂停。” (UI includes normal spacing), wrapping
`markdown.ErrPartialDocument`. Graph text identifies kind and16/32MiB limit,
“完整原文仍保留；请切到源码视图”, wrapping the corresponding parser's
`ErrLimitExceeded`. App production consumes only existing View warnings/errors;
test fixtures may import exported parser constants. No implicit mode switch.

Refusal caches by pointer+version, reports once per refused revision, and keeps
no raw source/model/rows/layout/label/index. Later shrink, Undo or replacement
at the same version recovers. A clean graph Flush succeeds so source switching
is available; a changed graph field Flush cannot locate its target without a
parse and fails preserving the draft. Markdown safe range-only Flush remains
successful above8MiB, including pending Compose, unless its existing conflict/
readonly/group guards fail. Release never silently discards a dirty draft.

Focused tests construct fixtures outside measurement. Exact command:
`native/scripts/go.ps1 -GoArguments @('test','./modules/markdownview','./modules/graphview','-run','^TestOversize','-count=1','-v','-timeout=180s')`
PASS markdownview1.301s/graphview1.767s. Measurements (cumulative heap allocation,
single headless sample, not retained/process memory; sub-ms0s is timer resolution):

| Path | Time | Bytes | Allocs |
|---|---:|---:|---:|
| Markdown8MiB+1, former String-copy-only baseline |1.5584ms|8,396,800|1|
| Markdown first refusal |1.0069ms|0|0|
| Markdown100 cached warning/export refusals |0s|1,600|100|
| Mindmap16MiB+1, String-copy-only baseline |3.5057ms|16,785,408|1|
| Mindmap first refusal |0s|16|1|
| Mindmap100 cached warning/export refusals |0s|1,600|100|
| Drawio32MiB+1, String-copy-only baseline |10.7734ms|33,562,624|1|
| Drawio first refusal |0s|16|1|
| Drawio100 cached warning/export refusals |0s|1,600|100|
| Stale Markdown double-click guard |0s|0|0|
| Stale graph mutation refusal, each kind |0s|176|5|
| Graph Compose + refused Flush, each kind |0s|96|3|

Baseline measures only the former unavoidable String call; it does not claim
complete before/after parser/UI or double String/[]byte allocation measurements.
Tests enforce generous64/128/256KiB budgets, not exact implementation counts.
They check source/selection/history/version, stale canvas removal, same-version
replacement, shrink/Undo recovery, actual native Compose safe range and conflict
paths, Release/recovery, nil export and mutations not reaching callbacks.

Added one post-probe regression for source Undo back in budget while the same
Markdown composition draft remains open. Final owned gate after that addition:
`native/scripts/go.ps1 -GoArguments @('test','./modules/markdownview','./modules/graphview','./formats/mindmap','./formats/drawio','-count=1','-timeout=180s')`
PASS markdownview1.817s/graphview3.149s/mindmap1.026s/drawio0.587s.
Same four packages `vet` clean. Owned git diff --check clean.

No whole-suite/build was run. Product is frozen for root/GLM review and DeepSeek
app lifecycle acceptance. Remaining evidence: app tests/screenshot/integrated
artifact and updated docs once accepted; no outstanding owned focused failure.

### Integration followups

Root review requested clearing the lightweight stale outlineCurrent/caret
descriptor along with preview caches. Done, stale-row regression asserts it.
Markdown package test PASS1.381s; vet/diff check clean.

DeepSeek's actual760x480 app image showed the graph recovery button below the
status bar. Inspected the image and corrected only view sizing: bounded/clipped
outer graph and content row, scrolling inspector, two-line error summary with
full tooltip, retry/recover together in one28px row. Added both-kind geometry
regression at532x285 (actual remaining content size). Graph package test
PASS2.148s, vet clean. Inspected DeepSeek's regenerated
`artifacts/acceptance/oversize-preview-20261108-150938/oversize-graph-draft-recovery-760x480.png`:
retry/recover both fully visible at roughly y397–424 above the y451 status bar;
inspector scrollbar and clipping are present. No further owned product edit.
Native README/API/architecture/VALIDATION and root ROADMAP/CHANGELOG now describe
the correction, with full integrated artifact still explicitly pending.
Read-only docs check:12 files balanced fences,160 relative links resolve,
owned diff check clean. Independent field source/base retained for recovery is
explicitly documented; no zero whole-process allocation claim.
