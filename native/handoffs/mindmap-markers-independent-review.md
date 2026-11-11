# Mindmap markers — independent current-source review

Status: read-only review against `native/handoffs/mindmap-markers-contract.md`.
No product file was edited and no tests, builds, or screenshots were run. This
review reflects the current marker sources and the owner's published focused
checkpoint, not older proposals or the superseded Release-data-loss concern.

## Resolution recheck — SVG correction

Current status: **RESOLVED — no remaining independent findings.** Both the
graphview SVG correction and final app acceptance evidence are adequate. This
final recheck was read-only; the reviewer did not rerun tests or inspect the
images independently.

The SVG omission is fixed correctly:

- `native/modules/graphview/mindmap_svg.go:61-62` now renders
  `boundedMindmapLabel(node.Node, node.Text)`, matching the canvas presentation
  and preserving marker/tag content before XML escaping.
- `native/modules/graphview/render_cache.go:120-155` builds only the bounded
  512-byte prefix without allocating a complete joined label; rune boundaries
  are respected.
- `native/modules/graphview/mindmap_svg.go:108-114` escapes the bounded text at
  the XML boundary with `xml.EscapeText`.
- `native/modules/graphview/sync.go:96` now truthfully states that canvas and
  SVG show the bounded prefix and that full topic/marker/tag content remains in
  source. It no longer misleadingly points icons/tags to the topic field
  editor.
- `native/modules/graphview/adapter.go:233-250` retains the existing
  Flush/sync/sync-error guard, so the updated SVG uses current authoritative
  content rather than a stale cache.

The new oracles in `native/modules/graphview/export_safety_test.go:152-234`
are meaningful: they parse the generated SVG as XML and compare exact decoded
labels for Unicode/keycap/custom markers, duplicate order, toggle/clear
transitions, child markers, and rich tags. They assert `<` and `&` are escaped,
oversized single/many/empty marker inputs and a large tag stay bounded and
valid UTF-8, output remains under12KiB, and export preserves source/history/
selection. The retained low-level multiline writer test intentionally checks
that its input is written exactly; that distinction is correct because the node
export caller now chooses the bounded presentation.

No further graphview defect was found in this correction.

### Final app resolution

The final app source resolves the split/SVG evidence gap:

- `native/app/mindmap_markers_app_test.go:322-345` enters actual split mode at
  760×480, scrolls to the full-height clear control, performs a real pointer
  click, verifies host mutation and rich-metadata preservation, exports SVG,
  and confirms the cleared SVG label no longer contains the removed markers.
- Lines 220–237 assert edited and reopened SVG labels reflect the marker state,
  including that the removed star does not reappear after reopen.
- Lines 442–492 preserve a UTF-8 BOM, reopen the saved document writable in
  preview, then perform a real priority-4 action and assert the exclusive
  result `["中文🙂", "⭐", "4️⃣"]`.

The owner's final focused log records `PASS ... 2.817s` for the marker suite;
root reviewed all four final images and accepted the focused gate. No current
app acceptance defect remains.

## Findings

### P1 — SVG export silently omitted markers that the canvas shows (resolved)

The correction above resolves this finding. The following text records the
original defect for provenance.

`native/modules/graphview/mindmap_svg.go:60` exports only `node.Text`. By
contrast, the native canvas uses `boundedMindmapLabel(node.Node, node.Text)`
at `native/modules/graphview/draw.go:113`, and
`native/modules/graphview/render_cache.go:120` includes `node.Icons` in that
bounded label.

Reachable sequence: open a mindmap containing icons, add or remove one through
the marker UI, then export SVG. The authoritative source and reopened document
contain the edited `icons`, and the native canvas displays the marker, but the
exported SVG contains only the topic text.

That is a user-visible export downgrade with no warning or documented
limitation. Either SVG export should render the same bounded marker content as
the canvas, or it should return/export a visible warning stating that markers
are not represented in SVG. Silent omission is not contract-compatible.

Current app marker tests verify source metadata and save/reopen, but do not
assert that an exported SVG preserves or visibly discloses marker state.

### P2 — Split-mode marker action/reachability was not covered (resolved)

The final app resolution above closes this evidence gap. The original issue is
retained below for provenance.

`TestMindmapMarkersPendingFlushModeSplitAndSourcePreview` in
`native/app/mindmap_markers_app_test.go` transitions through `split` and
`source`, but does not perform a marker action while `split` is active. The
narrow 760×480 screenshot test captures only preview mode.

The contract asks app acceptance to cover source/preview/split and readonly/
pending/conflict barriers. This is an acceptance-oracle gap, not proof that
split controls are unreachable; no split-mode screenshot or action was
available to make that stronger claim.

## Current-source conclusions

The reviewed implementation otherwise matches the active contract:

- Pointer and key capture records action, Buffer identity, selected node,
  presence, and a bounded ordered glyph copy before Flush.
- Activation flushes the independent field draft, reparses the authoritative
  source, and refuses on Buffer/node/icons identity mismatch. Readonly,
  foreign-group, preview-size, count, and byte barriers are enforced.
- Candidate count and byte checks precede candidate allocation. A semantic
  no-op performs no transaction or serialization.
- A successful action uses one host transaction, preserves legal numerical
  source selection, and uses marker-specific clamp wording.
- Collapsed presentation does not build all palette controls, and idle drawing
  does not clone source or icon lists per frame.
- Release clears marker gesture, presentation, disclosure, error, and focus
  state. Existing field-draft recovery is not altered.
- The two root-noted recent issues are fixed in current source: near-cap marker
  summaries append an ellipsis, and candidate checks count before allocating.

No other current product defect was found in the reviewed marker source or app
acceptance file.
