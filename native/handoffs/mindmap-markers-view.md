# Native mindmap markers — graphview checkpoint

Status: **FOCUSED CHECKPOINT PASS / PRODUCT FROZEN FOR ROOT REVIEW**.
Sole owner `/root/astra_marker_view`; active contract
`mindmap-markers-contract.md`. Scope: `native/modules/graphview/**` and this
handoff. No app, format, core, editor, vendor or script changes. No delegation,
full integration, release build, process launch or resource sampling.

## Root-authorized SVG correction — latest checkpoint

Root reopened only the verified SVG omission after independent review:
`mindmap_svg.go` previously rendered `node.Text`, silently omitting canvas
markers/tags. SVG now uses the existing `boundedMindmapLabel(node.Node,
node.Text)` before the existing XML escaping/multiline writer. Node dimensions,
layout, source and renderer budget are unchanged. `sync.go`'s long-label message
now explicitly says **canvas and SVG show a512byte prefix; full topic/marker/tag
content remains in source**. It no longer tells users icons/tags are in the
topic field editor. Draw.io's separate existing message is untouched.

Independent expected SVG text assertions cover Unicode/keycap/custom markers,
order/duplicates, XML escaping, priority toggle, symbol removal and clear;
unmodified child markers remain present. Large single glyphs,30,000custom/empty
glyph entries and a large rich tag verify the exact bounded prefix, valid XML,
bounded output and truthful warning without source/history/selection changes.
The low-level multiline writer still preserves its passed text exactly; the
node's bounded presentation is chosen by the export caller.

Actual latest commands/results:

```powershell
native/scripts/go.ps1 -GoArguments @('test','./modules/graphview','-run','^TestMindmapSVG|TestLongGraphLabelExport','-count=1','-timeout=120s')
# ok levnote.local/native/modules/graphview 1.040s ; exit0

native/scripts/go.ps1 -GoArguments @('test','./modules/graphview','-count=1','-timeout=120s')
# ok levnote.local/native/modules/graphview 4.477s ; exit0

native/scripts/go.ps1 -GoArguments @('vet','./modules/graphview')
# exit0, no output
```

All three runs passed first time after this correction. No screenshot rerun:
existing component images have short labels and show the same canvas/UI state;
only exported SVG and the warning on truncated-label documents changed. Product
is frozen again. App owner owns actual-adapter SVG acceptance. Worker follow-up
is design-only and cannot authorize additional graphview edits.

## Native controls and behavior

Stable accessibility names, shared with root and the app owner:

- `展开标记面板` / `收起标记面板`
- `优先级 1` through `优先级 9`
- `红色标记`, `橙色标记`, `黄色标记`, `绿色标记`, `蓝色标记`, `紫色标记`
- `星标标记`, `完成标记`, `重要标记`, `疑问标记`
- `清空全部标记`

`当前节点标记摘要` has the bounded summary as its accessibility description;
`标记操作问题` has the bounded operation-specific failure description. These
labels replace raw text in Tester's `Texts` lookup, so adapter tests should
inspect `Access(name).Description` when asserting the full message.

One `标记` section starts collapsed in the existing224DIP scrolling inspector,
after field/error controls. Collapsed state builds no19-button palette. Opening
and closing are read-only. The original independent topic/note/link editor stays
in place. Native Tab from that editor now reaches the marker disclosure; further
Tab reaches the finite controls. Existing link keyboard tests remain passing.

The19 glyphs use native buttons with Chinese names/tooltips, checkmark/empty-box
faces and selected borders, plus accessibility Checked. Colors also display
Chinese text. Clear explains that it removes custom glyphs too and writes[];
it is disabled on absent/present-empty fields. Existing oversized/read-only
content stays readable and explains the source-edit path/refusal.

## Action and lifecycle boundary

- `markers.go` owns the vocabulary, presentation, capture, action and refusal
  code. `adapter.go`, `view_ui.go` and `input.go` contain the scoped integration.
- A pointer-down or initial Enter/Space captures the action, Buffer identity,
  selected node ID, icons presence and exact ordered bounded list. Capture
  checks the displayed Buffer/version against the current buffer and parser
  cache; source changed without a Draw is refused rather than adopted as a
  new unseen baseline. No parsed Node pointer crosses Flush.
- Enter/Space activates once; auto-repeat is consumed. Native cancellation,
  outside release, scroll, collapse and Release discard the snapshot. Reused
  fixed-key controls cannot retarget a held press to a different selected node.
- Activation first Flushes the existing field draft. It then verifies buffer,
  node, read-only, foreign group and source-size barriers, refreshes the parsed
  authority and requires exact current icons presence/list equality. Unrelated
  topic or other source edits merge while that baseline still matches.
- Both current and prospective lists are limited to64entries/4096UTF8bytes.
  Count precedes bounded length summing and allocation. No UI limit was added
  to the pure format. Unknown values, order and duplicates stay intact except
  for the user's specific toggle/clear semantics.
- Real mutation uses a newly parsed candidate, `SetIcons`, canonical serialize,
  candidate document-byte bound and reparse before `commitBufferSelection(...,
  true)`. Failure cannot partially mutate the displayed document cache.
  One successful marker action adds one Changed callback/undo transaction;
  an earlier pending-field Flush legitimately adds its separate transaction.
- Semantic no-op clear does not serialize, change field presence, create
  history or call Changed. Numerical source selection is preserved where legal;
  any clamp uses `修改标记后源码已重新序列化…`, not the outline wording.
- Summary and palette selected flags are cached per Buffer/version/node. Idle
  frames do not clone icon lists, copy source, parse, build candidates or
  serialize. Summaries/tooltips are rune-safe and bounded to512UTF8bytes and
  direct the user to source. Near-boundary omitted entries always show `…`.
- Release clears marker snapshot, presentation cache, disclosure, error and
  native handle along with existing graph caches. Failed field drafts continue
  through the existing recovery path with exact text/selection/history.

## Verification actually run

Existing module helpers and tests were inspected before the first test run;
new tests reuse `chooseField`, `composeField`, `revealFieldControl`,
`replaceSource` and `parseMind`. Every Go invocation used the existing
`native/scripts/go.ps1` Go1.27.3 wrapper. Formatting used the bundled gofmt.

Final commands, after the bounded-summary/count correction:

```powershell
native/scripts/go.ps1 -GoArguments @('test','./modules/graphview','-run','^TestMarkersBudgetCurrentCandidateAndSummary$','-count=1','-timeout=120s')
# ok levnote.local/native/modules/graphview 1.390s ; exit0

native/scripts/go.ps1 -GoArguments @('test','./modules/graphview','-count=1','-timeout=120s')
# ok levnote.local/native/modules/graphview 3.462s ; exit0

native/scripts/go.ps1 -GoArguments @('vet','./modules/graphview')
# exit0, no output
```

Prior actual results, retained as failure/correction history:

1. Initial marker-only run failed in2.087s: the independent rich JSON fixture
   omitted the child `children:[]` that existing canonical serialization adds;
   message assertions used HasText despite fixed accessibility labels.
2. Added explicit child `children:[]` to the fixture (kept the complete
   independent JSON semantic comparison), and one action/disclosure repaint.
   The next marker-only run failed in1.914s only on those labeled-text assertions.
3. Assertions now read the native accessibility description. Exact raw checks
   additionally verify top-level `{"number":1.00}`, node`{"number":1e3}`,
   tag-style`1.20`, tag-extra`2.00` and image-extra`0.30` after reparse.
   Marker-only suite passed in1.954s.
4. Complete graphview package with image generation passed in4.457s; vet exit0.
5. Final source review caught the summary near-cap omitted-entry ellipsis edge.
   Added an independent458..466byte-prefix regression, fixed the bounded append,
   made candidate count checking explicitly precede byte summing, and cleared
   gesture state directly on button scroll/zoom too. Final commands above pass.

The complete package includes existing topic/note/link, draw.io, SVG safety,
outline, preview-size, bounded-label/cache and settled-idle regression tests.
No separate process resource probe or benchmark was added/run.

New focused coverage includes actual pointer, Tab+Enter, Space and auto-repeat;
collapse/discovery/selected-state accessibility; priority/color exclusivity,
targeted duplicate removal, arbitrary-glyph/order preservation and independent
symbols; clear[]/absent-empty no-op; one Changed/undo, exact undo/redo and legal
source selection; held-press Buffer/node/list reorder/content/presence/deletion
and stale-without-Draw guards; unrelated source/pending-note merge; failed-note
recovery; foreign-group/read-only/invalid/oversized-source refusal;64/65 and
4096/4097 current/prospective limits; serialized candidate document-size guard;
marker-specific source-selection clamp; Release/cache/idle and draw.io isolation.

## Final component images and evidence limits

Generated by the complete4.457s package run with:

```powershell
$env:LEVNOTE_GRAPH_MARKERS_SCREENSHOT_DIR = 'F:\AIproject\LevNote\native\modules\graphview\images'
native/scripts/go.ps1 -GoArguments @('test','./modules/graphview','-count=1','-timeout=120s')
```

All four files were opened and visually inspected:

- `modules/graphview/images/mindmap-markers-light-760x480.png` (33,268bytes)
- `modules/graphview/images/mindmap-markers-dark-760x480.png` (33,424bytes)
- `modules/graphview/images/mindmap-markers-recovery-light-760x480.png` (39,645bytes)
- `modules/graphview/images/mindmap-markers-recovery-dark-760x480.png` (39,889bytes)

The final summary-boundary/count/scroll correction does not change any state
shown in these images. Per root's instruction they were retained without a
redundant screenshot rerun. Final3.462s validation skipped only the optional
image-generation test because the environment variable was not set.

The image test actually scrolls to every named marker row, clear, note editor,
link and conflict recovery control; the existing inspector reaches them within
760x480. They are not all visible simultaneously. The selected state remains
legible in both themes. Screenshots show native tester rendering and synthetic
input transport; they do not establish OS IME, GPU or production-window behavior.
Root review and the sole app integrator's registered-adapter/full native gate
remain required; this checkpoint makes no release/full-integration claim.
