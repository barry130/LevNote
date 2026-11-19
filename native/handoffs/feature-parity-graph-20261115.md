# Native node color inspector · 2026-11-15

## Delivery state

The inspector passed root graph-02 and full test-01. The subsequently authorized read-only scene consistency increment passed root graph-03 and remains frozen while root runs full test-02. The child ran pinned gofmt and scoped git diff checks only; all Go execution remains owned by lead.

Final approved scope contains thirteen Go files plus this handoff: the original eight plus two read-only production files and three related test files. The root explicitly authorized the one existing selected-stroke assertion to follow the new product contract. Existing packages, toolchain, vendor and closed evidence remain unchanged. No commit or push was created.

## Behavior

- The mindmap inspector has a collapsed Colors section after Tags: text, fill and border, each with six choices, actual current value and independent reset. Native file keys are color, background-color, border.
- Only the selected property changes. Unknown ordered style members, unknown node/top-level metadata and original numeric spellings survive. Removing the last member removes the style object; an already absent property preserves optional/empty-object presence and original bytes.
- Equivalent supported color literals, including named/short/full hex, cause no serialization or undo transaction. Palette clicks use fixed six-digit literal colors; custom and unsupported existing values remain visible and preserved until that property is explicitly changed/reset.
- One color action produces one source transaction and preserves source selection with surrogate-aware clamping. A confirmed ordinary field draft flushes first as its own transaction; an error after a successful field write says the field was already written and the color was not.
- Captured target: owner buffer, source version/selection, node id, field/tag-input identities, tag/style/release epochs. Revalidate before Flush and against its exact receipt; parse current source for the action. Buffer/selection/node round-trip changes, readonly, active source group, composition and pending tag drafts prevent a color write. Changed reentry cannot restore an old draft or bind the old status to a replacement document.
- Draw and SVG share StyleTextFallback: only explicit opaque background without any color member derives readable ink. Transparent/custom fills and all explicit ink remain untouched. No-style rendering keeps prior defaults. A selected node with valid explicit border now draws that real border plus an outer selection ring.
- Native host-read-only graph views use the same drawing path. The installed-engine scene is an independent renderer, not a call through svgColors: its current worker now derives missing DTO Text via StyleTextFallback, and its native scene painter preserves explicit Stroke plus an outer selection ring. An existing installed worker binary keeps its prior paint projection until replaced by a newly built package; the host still does not parse source or synthesize a missing worker response.

## Files

| File | Purpose |
| --- | --- |
| [style.go](../formats/mindmap/style.go) | Typed per-property edits, raw preservation/no-op, shared fill contrast fallback |
| [style_test.go](../formats/mindmap/style_test.go) | Five format tests: original-byte no-op, independent reset, unknown/raw retention, rollback, equivalent colors and fallback |
| [svg.go](../formats/mindmap/svg.go) | Calls shared text fallback before existing literal color mapping |
| [styles.go](../modules/graphview/styles.go) | Palette UI, bounded presentation, input capture and exact-receipt activation |
| [styles_test.go](../modules/graphview/styles_test.go) | Six native test families with pointer/keyboard, histories, readonly, Flush/composition/conflict, held gestures/reentry, SVG/painter/optional PNG |
| [adapter.go](../modules/graphview/adapter.go) | Style state, input observer, release lifecycle and issue reporting |
| [view_ui.go](../modules/graphview/view_ui.go) | Inspector placement only; existing Focus color changes predate this scope |
| [draw.go](../modules/graphview/draw.go) | Shared text fallback and separate selection outline for explicit borders |

## Checks and next action

Pinned formatter: .localtools/go1.27.3/bin/gofmt.exe. Initial -w failed for the newly created style.go/styles.go with Access is denied on gofmt temporary files. No ACL/toolchain repair was used. Formatting those and all final eight files via stdout and CAS write succeeded; syntax parsed. Scoped git diff --check exited 0. The existing tracked diff of view_ui.go includes two earlier Accent→Focus changes already present in the frozen baseline; this child added only drawStyles(c).

Lead owns all Go execution through the current local gate. Focused selection: TestStyle across ./formats/mindmap and ./modules/graphview. TestStylesNativeCanvasSVGAndOptionalCapture optionally writes four real Tester.Image PNGs when LEVNOTE_GRAPH_STYLE_SCREENSHOT_DIR is set (light/dark × text/fill-border, each 760×480). No test skips are added when the variable is absent: behaviors always run, only file capture is conditional.

Root-reported actual outcomes are recorded below. The synthetic native pointer/keyboard/painter evidence does not establish physical OS IME or a delivered package.

## First root run and narrow correction

Root graph-01 exited 1: 365 RUN / 352 PASS / 3 original SKIP / 10 FAIL identities (including parent tests). Original stdout.jsonl remains in .localtools/feature-parity-20261115-graph-01. Graph inputs stayed identical; root reported aggregate allInputsSame=false because three independent Markdown tests were created during the run.

The failed custom-value assertion exposed a presentation accessibility issue: MyGo indexes explicit Label before visible text. The old label named only the property. The label now contains the bounded actual summary too; visible text and tooltip stay unchanged. The exact current values remain asserted.

Conflict and six Changed-reentry fixtures incorrectly assumed Tester.Type leaves an ordinary field pending. Existing draft.go OnChange immediately calls Flush. These fixtures now enter a public source Buffer group, type through the real native editor, end the empty group and assert unchanged source/history plus pending draft and ErrGroup before attempting the color. The previously passing separate-field-undo fixture also uses this setup, so it now exercises a real pending field. Core error, no-color and transaction assertions remain. Production transaction code did not change in this correction.

Only styles.go and styles_test.go changed for graph-02; both pinned gofmt executions parsed and exited 0. Lead and independent reviewer received the new stable point.


## Root graph-02 and full test-01

Root graph-02 exited 0: 365 RUN / 362 PASS / 3 original SKIP / 0 FAIL. All 424 input files stayed identical before and after. Root full test-01 also exited 0: 3347 PASS / 49 original SKIP / 0 FAIL, with the same 424 inputs unchanged. The original graph-01 failures remain preserved.

Four actual graph-02 PNGs were inspected under .localtools/feature-parity-20261115-graph-02/images: light/dark fill-border captures show the real green border inside the blue selection ring; the pale fill and explicit blue text remain legible. The text captures only expose the top edge of the swatches at the bottom of the viewport, so those two images are not evidence of a fully visible palette. This finding prompted the dedicated screenshot helper correction described below.

Independent reviewer gap-editor found no substantive inspector production issue after inspecting the transaction/Flush receipts, raw preservation, input identities, selected border and SVG path. The reviewer independently confirmed that the worker scene does not call svgColors and initially lacked the new fallback/outer ring. Lead authorized closing that actual visual difference in this batch.

## Read-only scene consistency increment

- [Worker projection](<F:/AIproject/LevNote/native/cmd/levnote-mindmap-preview/scene.go>): only an absent color member asks the shared StyleTextFallback and normalizes the admitted result into the existing eight-digit DTO Text. All explicit color members, including unsupported/malformed values, retain their prior path. No-style and transparent/translucent/custom fill keep their prior defaults. Protocol and worker admission/lifecycle are unchanged.
- [Scene paint](<F:/AIproject/LevNote/native/modules/graphview/scene_preview_draw.go>): an explicit nonempty DTO Stroke remains the node border when selected; the selection ring is drawn 4px outside it, matching native graph behavior. Missing Stroke retains the old single-border selection. Root-authorized existing assertion in [scene_preview_test.go](<F:/AIproject/LevNote/native/modules/graphview/scene_preview_test.go>) now expects its literal #112233ff instead of theme Accent; all other assertions remain.
- [Worker regression](<F:/AIproject/LevNote/native/cmd/levnote-mindmap-preview/scene_style_test.go>): one family with thirteen authored IPC cases checks opaque light/dark and opaque-alpha fallback, explicit/transparent/invalid/custom ink, transparent/translucent/custom fill, empty/no-style, and border-only. Decoded DTO colors are compared to independent constants; original bytes remain equal.
- [Native scene regression](<F:/AIproject/LevNote/native/modules/graphview/scene_colors_test.go>): light/dark times explicit-style/no-style use the real pointer and Painter. Pixels prove green actual border plus blue outer ring coexist, dark fallback ink appears on pale fill, no-style old single-border baseline remains, and source bytes/version/history/selection plus worker request count stay unchanged. Optional capture adds two 760x480 read-only images.
- [Owned capture helper](<F:/AIproject/LevNote/native/modules/graphview/styles_test.go>): a full 28px button must be visible rather than accepting any positive clipped height. The text capture targets the reset below both text palette rows. Shared field helper and production inspector layout are unchanged.

All six revised Go inputs were parsed/formatted with the pinned formatter and CAS writes; scoped git diff --check exited 0. Root graph-03 actually exited 0: 422 RUN / 419 PASS / 3 original SKIP / 0 FAIL, including the cmd worker package. All 426 input files stayed identical before and after. The thirteen owned Go inputs remain frozen; root full test-02 is in progress, including the independently reviewed Markdown P2 correction and nine new test identities. Final capture inspection is not claimed by this receipt. No added skips; optional PNG output remains the only environment-conditional behavior.

Independent reviewer gap-editor completed a read-only review of the narrow increment against the fixed baseline and found no blocker. The reviewer read both production paths, the thirteen worker cases, all four native painter combinations and source/history/selection/request invariants, and the owned capture helper. The single old selected explicit-stroke assertion change matches the new product contract; no behavior assertions were weakened. The review performed no additional run and does not substitute for root execution receipts.

## Frozen source identities after read-only increment

| File | SHA256 |
| --- | --- |
| [formats/mindmap/style.go](<F:/AIproject/LevNote/native/formats/mindmap/style.go>) | b2649aaeb7e4aca12a2a52476f7861e8b4e4c2349089c099419471fddffaf547 |
| [formats/mindmap/style_test.go](<F:/AIproject/LevNote/native/formats/mindmap/style_test.go>) | b69ab2b72d28d6a5e178324ed212db7749a5031753d8415599c635eda16c5020 |
| [formats/mindmap/svg.go](<F:/AIproject/LevNote/native/formats/mindmap/svg.go>) | dbd57297bc5fc962beb8f77b96ceb95b2ed7775109a018e0eeb016907c2cf844 |
| [modules/graphview/styles.go](<F:/AIproject/LevNote/native/modules/graphview/styles.go>) | 4c473875983c269e5f78469e31cf5a4af6f80fe1b1c6fe604f1ea3551666179e |
| [modules/graphview/styles_test.go](<F:/AIproject/LevNote/native/modules/graphview/styles_test.go>) | 7787119ce0ea71c6077a69e4ef276ff84d8e448131e88cc541a0b2fb05e7bf01 |
| [modules/graphview/adapter.go](<F:/AIproject/LevNote/native/modules/graphview/adapter.go>) | e4778939ca73af451ccac43828d59d389edbb1c9d59d7881d53e54e6edced009 |
| [modules/graphview/view_ui.go](<F:/AIproject/LevNote/native/modules/graphview/view_ui.go>) | 1c4bfcfe69880f9861e37b4da04c9e2c05beac1266ca0401c11e1fa6361a97f6 |
| [modules/graphview/draw.go](<F:/AIproject/LevNote/native/modules/graphview/draw.go>) | f3dba71e50f6a495afd4e53fd39e33c37c0e293ba2e23e8f6c6497720fcfae98 |
| [cmd/levnote-mindmap-preview/scene.go](<F:/AIproject/LevNote/native/cmd/levnote-mindmap-preview/scene.go>) | 566074bfef450ffd4ec005605dda06eac82929f847cbd88267a96996668e2ce8 |
| [cmd/levnote-mindmap-preview/scene_style_test.go](<F:/AIproject/LevNote/native/cmd/levnote-mindmap-preview/scene_style_test.go>) | 8a4667e8e81d15d59784284a6ee77184a14dc7b7f9dc7309b55c68ced2d2e3c9 |
| [modules/graphview/scene_preview_draw.go](<F:/AIproject/LevNote/native/modules/graphview/scene_preview_draw.go>) | 24cdc32f053fa12f29041b3f858de0cc575b45220b65740c0ecfe86c18ca9902 |
| [modules/graphview/scene_preview_test.go](<F:/AIproject/LevNote/native/modules/graphview/scene_preview_test.go>) | b0e575ba3f5bcf7132d41c4cd11f7bb55fb70956f2fe05e5182797906b123416 |
| [modules/graphview/scene_colors_test.go](<F:/AIproject/LevNote/native/modules/graphview/scene_colors_test.go>) | 2634aee81fa70f7b9ac1dcfbbd7a7092bb79eb9d67125034ec240c264a530309 |
