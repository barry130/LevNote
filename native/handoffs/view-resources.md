# View resource acceptance — acknowledgement

Astra starts the queued `view_resource_task.txt` after session/Partial focused
checkpoint. Scope: owned Markdown/graph/editor view packages only; no parser,
app or vendor edits. No subagents, commits or pushes.

First measure cold parse/layout separately from repeated frames for 2,000
headings, one 1,000-item list, and 500/2,000-node graphs. Then implement bounded
visible rows/labels/cache only where the measurements justify it. Preserve
draft IME/conflict/release behavior and verify it with actual native inputs.

## Implementation checkpoint (focused work still running)

Confirmed baseline measured before fixes (Windows/amd64, headless, 5 requested
unchanged frames; these are total allocated bytes, not RSS):

| Workload | Built / shaped per frame | Five frames | Allocated |
| --- | ---: | ---: | ---: |
| 2,000 Markdown headings | 2,000 rows | 31.337ms | 5,831,024 B |
| 1,000 list items | 1,001 rows | 21.451ms | 1,155,904 B |
| 500 mindmap nodes | 500 labels | 73.164ms | 9,284,016 B |
| 500 drawio nodes | 500 labels | 27.482ms | 9,050,424 B |
| 2,000 mindmap nodes | 2,000 labels | 241.866ms | 37,060,720 B |
| 2,000 drawio nodes | 2,000 labels | 122.966ms | 36,181,424 B |

All idle probes settled (`FramePending=false`); the defect was cost of a
requested frame, not an unbounded idle invalidation loop.

Implemented:
- Markdown cached flattened block/list-item descriptor rows + MyGo `ui.List`.
  Numeric stable identities map through the bounded coordinate journal after
  outside source edits. No copied source retained by row keys. Active draft is
  a fixed visible panel, not recycled into preview rows. Escape/finish flushes
  and preserves list anchor. Release drops list caches while retaining scroll.
- Graph viewport node culling, clipped straight mindmap connectors, cached page
  node index (removed per-frame maps), reusable connector scratch, 256-entry
  LRU label geometry cache keyed by node/font/size/width and invalidated on source
  refresh/page/Release. Tiny overview labels are omitted with on-canvas guidance;
  a per-frame label limit prevents all-visible fitted diagrams thrashing cache.
  Long displayed label prefixes are bounded to 512 UTF8 bytes; full field/source
  remains preserved and editable. Colors are drawn from current theme.

First same-workload post-change results (before final small field/edge refinements):

| Workload | Actual row/node builds, new shaped labels | Five frames | Allocated |
| --- | --- | ---: | ---: |
| 2,000 headings | 12 rows | 0.602ms | 41,904 B |
| 1,000 list items | 17 rows | 1.052ms | 65,520 B |
| 500 mindmap nodes | 18 nodes, 0 new labels | 3.526ms | 28,312 B |
| 500 drawio nodes | 35 nodes, 0 new labels | 1.057ms | 14,584 B |
| 2,000 mindmap nodes | 18 nodes, 0 new labels | 4.532ms | 27,088 B |
| 2,000 drawio nodes | 35 nodes, 0 new labels | 1.076ms | 14,584 B |

Tests landed for far-row scroll+source insertion preserving identity, real
IME in a far block while preview scrolls away, Escape retaining preview anchor,
1,000-item list editing at 265x480, graph pan/cache/theme/zoom/content/font/width,
all-visible 500-node cache limit, crossing edge culling and hit-testing, Release
cache clear/preferences retention. All existing Markdown and graph tests passed
in first run (Markdown1.439s, graph53.997s).

The 54s graph result exposed a substantive additional field bottleneck: a 30k
UTF16 CJK+emoji label was below the independent editor's 32k ordinary-line
threshold and therefore shaped as one huge field (isolated45.54s). Fix in
progress adds a lower per-field `Editor.ShapeLimit` (graph512, Markdown2048),
keeping the same segment navigation/IME/source text. A zoom test also incorrectly
read only the last settled frame's shape count; fixed it to inspect the geometry
key actually cached, because a second settling pass correctly shapes zero.

Remaining before final acceptance: rerun focused editor/views tests after field
limit, ensure long segment navigation works at small width, measure fitted
unchanged and pan frames with final code, fresh 265x480 draft/conflict screenshots
and visual inspection, focused vet, notify DeepSeek for full integration. No
new features or other owner files modified.

## Final checkpoint — stable for integration

All work above is complete. Final focused command:

```
native/scripts/go.ps1 -GoArguments @('test','./core/text','./core/session','./ui/editor','./modules/markdownview','./modules/graphview','./dependency-patches','-count=1')
native/scripts/go.ps1 -GoArguments @('vet','./core/text','./core/session','./ui/editor','./modules/markdownview','./modules/graphview')
```

Passed: text0.894s, session1.181s, editor3.330s, Markdown1.599s,
graph2.727s, dependency patch1.764s. Vet clean. Full integration/build is
DeepSeek's task; Astra has not run another whole-repository suite.

Final measurement command:

```
native/scripts/go.ps1 -GoArguments @('test','./modules/markdownview','./modules/graphview','-run','FrameResourceProbe|LongLabels','-count=1','-v')
```

| Workload | Cold parse/layout | Cold native view | 5 normal unchanged frames / allocated | Rows/nodes built, new labels |
| --- | ---: | ---: | --- | --- |
| 2,000 headings | 1.640ms / 3,164,728 B | 15.522ms / 1,222,400 B | 1.127ms / 42,936 B | 12 rows |
| 1,000 list items | 1.610ms / 455,304 B | 2.614ms / 1,032,520 B | 1.053ms / 68,368 B | 18 rows |
| 500 mindmap nodes | 4.240ms / 2,320,768 B | 43.024ms / 24,510,488 B | 2.466ms / 26,624 B | 18 nodes, 0 labels |
| 500 drawio nodes | 2.549ms / 1,606,320 B | 4.678ms / 1,652,264 B | 1.517ms / 14,584 B | 35 nodes, 0 labels |
| 2,000 mindmap nodes | 17.636ms / 9,897,816 B | 34.952ms / 3,607,344 B | 3.710ms / 26,832 B | 18 nodes, 0 labels |
| 2,000 drawio nodes | 10.148ms / 7,010,376 B | 2.566ms / 2,124,808 B | 1.008ms / 14,952 B | 35 nodes, 0 labels |

Cold native-view numbers include headless renderer/font initialization and are
order-sensitive (first graph case initializes shared renderer/font resources).
No OS RSS comparison or Tauri claim is implied. Every idle view settled without
pending frame requests. Normal/pan geometry is already cached before measuring.

Five pan frames: mindmap500 16.746ms/62,208B; drawio500 8.775ms/127,112B;
mindmap2000 13.907ms/60,576B; drawio2000 7.390ms/126,264B. Culling preserves
crossing connectors and partially visible node hit-testing. Page+buffer identity
changes rebuild cached geometry even when Buffer.Version is equal.

Fitted overview, five unchanged frames (explicit test order to avoid ambiguity):

| Fixture | Five frames / allocated | Visible nodes | Hidden tiny labels | New labels |
| --- | --- | ---: | ---: | ---: |
| mindmap500 | 15.676ms / 288,592B | 106 | 105 | 0 |
| drawio500 | 2.660ms / 238,928B | 500 | 499 | 0 |
| mindmap2000 | 35.353ms / 371,208B | 106 | 105 | 0 |
| drawio2000 | 2.010ms / 364,840B | 800 | 800 | 0 |

Documentation review reran only `TestGraphFrameResourceProbe/kind1-2000`:
confirmed106 visible/105 hidden/0 new labels/1 cached, package1.049s.
That verification run measured26.0366ms/431,152B for five fitted frames; it is
not mixed into the original same-run table. The 500-overlapping-visible-node regression proves
the 256-entry LRU does not thrash; changing selection settles again. Fitted
mindmap still loops its edge list and draws intersecting connectors, hence its
remaining higher work. No spatial index or edge simplification is claimed.

### Source and property field shaping

Root correctly noted the long-field bottleneck also affected source view.
Measured the same 30,000 UTF16 CJK+emoji line in the independent source editor:
800px unwrapped cold frame **46.848s,39,500,448B** at the old32k threshold.
Default ordinary paragraph threshold is now4k UTF16, longer lines use2k segments;
graph property field uses512, Markdown draft2048. Segment controls preserve full
content, surrogate boundaries, tail edit/undo and native IME. Stable explicit
keys also fixed focus loss when a segment toolbar first appears.

After default limit, same source fixture:
- 800px unwrapped:241.929ms /2,608,528B /14,884allocs.
- 800px wrapped:489.725ms /2,175,904B /22,611allocs.
- 265px unwrapped:214.642ms /1,645,032B /11,267allocs.
- 265px wrapped:464.537ms /2,232,176B /23,524allocs.

These are bounded cold frames, not claims of sub-frame typing latency on that
path. `TestSourceLongUnicodeResourceProbe` preserves source and settles; actual
field next/previous/tail/undo/focus/IME tests pass. Long graph label full test
dropped45.54s to0.08s. Displayed note/page-name/selected-ID prefixes are bounded
before shaping, with ellipsis; note no longer copies a full []rune every frame.

### Narrow conflict layout inspection

Root's actual app760x480 conflict screenshot showed preview text over the status
bar. Fixed owned view: outer/list clipping, shrinkable zero-min-height preview,
compact fixed draft, two-line concise conflict with full tooltip, bounded
buttons/footer. New `TestConflictDraftFitsActualSplitContentBounds` reproduces
the shell's265x300 content at y150..450, verifies every conflict control/footer
is within it, and scrolls to a far preview row while the draft remains.

Generated and visually inspected:
- `build-cache/markdown-resource-acceptance/markdown-conflict-bounded-split.png`
  shows the complete conflict panel + preview + footer ending above STATUS BAR.
- `.../markdown-virtual-draft-narrow.png` at265x480 shows native preedit and
  scrollable original list rows; no repeated placeholder wall.
- `.../markdown-conflict-narrow.png` and `markdown-narrow.png`.
- `build-cache/graph-resource-acceptance/drawio-narrow.png` and existing graph
  narrow/conflict screenshot test outputs; toolbar/page-name and zoom fit.

DeepSeek has been told to recapture actual minimum app split conflict after the
owned fix and can run full integration now. Remaining limits: no OS IME/window
automation repeated in this stage; tests use MyGo actual native input transport
in headless tester. Graph cold parse/layout remains proportional to node count,
and very long source lines disclose segmented editing. No new features begun.
