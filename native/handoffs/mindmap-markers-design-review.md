# Mindmap markers — next finite design review

Status: DESIGN ONLY, 2026-11-08. This refines the next slice to **icons only**;
tags remain displayed but deferred. No product, test, documentation, or build
changes were made for this review.

This is not activated. Implementation starts only after root accepts the
current Markdown single-cell release and explicitly activates this slice.

## Current capability audit

The native format already treats `icons` as an ordered, optional string field:

- `Node.Icons` is `*[]string`; `nil` means absent and a present empty slice
  means `[]`.
- `Parse` accepts arbitrary glyph strings and does not apply a marker
  vocabulary or semantic-ID mapping.
- `Serialize` writes `icons` in the existing canonical node key order and
  preserves arbitrary unknown glyphs, order, and duplicates.
- Graphview already displays icons in `boundedMindmapLabel`, with a 512-byte
  UTF-8-safe display prefix and the existing bounded label-layout cache.
- Existing note/hyperlink edits establish the required mutation pattern:
  find the node, validate before mutation, preserve no-op bytes/dirty state,
  roll back on validation failure, and mark dirty only after success.

Historical ADR 0053 remains useful as product semantics, but its frontend
palette is not the native implementation. The native slice should keep the
same principle: the file stores display glyphs; the 19-glyph vocabulary and
mutual-exclusion rules belong only to graphview.

## Missing behavior

There is no native way to add, remove, or clear mindmap markers. The inspector
can edit topic, note, and hyperlink, but icons are read-only display metadata.
Tags also have no editor, but their `Tag.Object`, `ClassName`, `Style`,
`Extra`, and original raw-object concerns require a separate design; they are
explicitly outside this slice.

## Nil-slice serialization review

`Node.Clone` currently uses `append([]string(nil), (*n.Icons)...)`, so a
present-empty inner slice becomes nil. The initially suspected `json.Marshal`
pitfall is **not** reachable, however: `encodeNode` passes the slice directly,
`writeJSONValue` handles every `[]string` by constructing an `encodeArray`, and
`writeJSONArray` writes `[]` when its length is zero. The pointer to `Icons`
remains non-nil after clone, so presence is preserved and serialization writes
`[]`, not `null`.

No product correction is required for this behavior. Keep the evidence as a
focused regression only if the icons API work touches clone/serialization
anyway:

1. Clone a node whose `Icons` points to an empty present slice and assert the
   clone pointer remains non-nil.
2. Serialize that clone and assert the exact node text contains `"icons": []`,
   not `"icons": null`.
3. Repeat through `AddNode`, which is the current reachable clone consumer.

Do not normalize `Node.Clone` merely to avoid the inner nil slice unless an
actual consumer proves a defect. The new `SetIcons` API should still store a
non-nil clone so its own boundary is predictable.

## Proposed pure-format API

Keep the seam glyph-agnostic and additive:

```go
// Icons returns a defensive copy of the exact ordered glyph list.
// present is false only when the optional field is absent.
func (d *Document) Icons(id string) ([]string, bool, error)

// SetIcons stores an exact, present icon list. A nil argument means [].
func (d *Document) SetIcons(id string, values []string) error
```

Required semantics:

- Missing node and invalid UTF-8 in any glyph reject before mutation.
- `Icons` never changes the document or its serialization.
- A present empty list returns a non-nil zero-length copy and `present == true`.
- `SetIcons(id, nil)` always stores present `[]`; turning an absent field into
  present-empty is a real semantic change.
- Setting a present list to the same ordered values is a semantic no-op:
  retain the original serialization and dirty state. A nil argument and an
  empty non-nil slice are both an empty candidate, but neither is a no-op while
  the field is absent; both create present `[]`.
- Clone the caller's slice before storing it. Later mutation of the caller's
  slice must not alias into the document.
- Store the clone as a non-nil slice so JSON encoding writes `[]`, never
  `null`.
- Save the old `*[]string`, assign the new pointer, run the existing document
  validation, and restore the old pointer on failure. Only a real change marks
  the document dirty.
- Do not alter the current `Icons` pointer merely because the new value is
  empty. A real empty-list change should assign a fresh non-nil slice so the
  semantic transition remains explicit and caller-side aliasing cannot affect
  the document.
- Do not add a format-side glyph vocabulary, semantic marker IDs, ordering
  rule, duplicate rejection, count cap, or aggregate-byte cap. Existing files
  must continue to parse and serialize with their arbitrary metadata intact.

This API is deliberately smaller than a general field API. There is no
`ClearIcons`: clearing all markers means writing present-empty `[]`; making
the field absent is not a user action in this slice.

## Graphview vocabulary and toggle semantics

Graphview owns a fixed 19-glyph palette:

- Priority `1️⃣` through `9️⃣`: mutually exclusive.
- Color `🟥 🟧 🟨 🟩 🟦 🟪`: mutually exclusive.
- Symbols `⭐ ✅ ❗ ❓`: independent toggles.

For a click on known glyph `G`, derive the candidate from the captured
baseline list:

- If `G` is absent: remove every known marker from `G`'s exclusive group, then
  append one `G` at the end. For a non-exclusive symbol, just append one `G`.
- If `G` is present: remove every exact occurrence of `G`, preserving the
  relative order of all other entries.
- Duplicates of the targeted glyph are targeted by the toggle. Duplicates of
  unrelated known or unknown glyphs are untouched.
- Unknown glyphs, original order, and unrelated duplicates survive every
  untargeted toggle.
- “清空全部标记” writes `SetIcons(id, nil)`, producing present `[]`, and is a
  no-op/disabled action when the current list is already present-empty.

If a node contains multiple markers from one “exclusive” group, the file is
still valid. The palette should say so (for example “已有多个同类标记”) and the
next click still applies the deterministic group rule above; the format layer
never rejects the existing file.

## Bounded view preflight

Graphview should enforce editing budgets before cloning or building a
candidate list:

- At most **64 icons**.
- At most **4096 UTF-8 bytes total** across all icon strings.

Use a cheap direct count/sum over the parsed node's current `Icons` before
calling `Icons`; an over-budget existing list must not be copied into panel
state. If either the current list or the derived candidate exceeds a budget,
disable the offending marker action and show a bounded reason such as
“标记过多或过长，请切到源码编辑”。 The node and its bounded display remain visible.

These are UI budgets, not parser limits. A format-level test can still assert
that 65 valid icons remain accepted by the pure API; graphview must visibly
refuse to edit that list.

The current-marker summary should be bounded (for example at 512 UTF-8 bytes
with a rune-safe ellipsis) and report the total count. The fixed palette has 19
buttons; membership checks and toggle derivation must not allocate a new full
candidate on every frame.

## Graphview integration and lifecycle

Use the existing whole-document mutation boundary rather than a second
independent editor:

1. Flush any pending note/hyperlink/topic draft first. On failed Flush, retain
   that draft and refuse the marker action with the existing visible error.
2. Ensure the parsed cache is current, then capture the host `Buffer`, node ID,
   and a bounded copy of the current icons at the native gesture.
3. Compute the next list from that captured baseline under the 64/4096 budgets.
4. Skip a semantic no-op before touching the host; no serialization, history
   step, `Changed`, or cache invalidation may occur.
5. For a real change, parse the current bounded host source, call `Icons`,
   and require the current presence/list to equal the captured baseline. A
   missing node or changed list returns a visible stale/conflict error rather
   than applying the click to a current list.
6. Call `SetIcons`, serialize, reparse, and commit exactly one ordinary host
   transaction. Preserve the selected node and invalidate the parsed/render
   caches as the existing mutation path does.

Read-only, sync-failed, missing-buffer, oversize-preview, and no-selected-node
states disable marker actions visibly. The buttons must be reachable by native
pointer and keyboard focus, with stable Chinese labels/tooltips. No marker
action may silently alter a different selected node or fall back to a current
node after a redraw.

## Memory and fidelity constraints

- Do not materialize the whole source in a per-frame marker panel. The existing
  parse/serialize work is confined to an actual user action.
- Avoid `Read`-to-`Set` round trips on unchanged frames. Membership checks can
  read the already-parsed bounded node without allocating a new list.
- The action-time copy is bounded by the 64/4096 UI budget.
- No-op actions preserve exact original bytes and dirty state.
- A real action preserves unknown top-level/node fields, rich tags, image data,
  number spellings, and all unrelated icons.
- The format layer canonicalizes an edited document as it already does for
  note/hyperlink; this is not an additional marker-specific rewrite.
- Invalid UTF-8, validation failure, stale node, foreign host buffer, and
  rejected budgets must be atomic and leave no partial mutation.

## Focused acceptance matrix

Pure format:

1. Absent `Icons`: `present == false`, no dirty change.
2. Present `icons: []`: non-nil empty copy, `present == true`.
3. `SetIcons(nil)` on absent: stores present `[]`, marks dirty, survives
   serialize/reparse.
4. Same-list `SetIcons`: no dirty change and exact original bytes.
5. Caller-side mutation after `SetIcons` cannot mutate the document.
6. Invalid UTF-8 and missing node reject atomically, including on an
   already-dirty document.
7. Validation rollback restores the old pointer/value and dirty flag.
8. Unknown glyph, order, and duplicates survive a no-op and an unrelated real
   edit.
9. `Node.Clone` and `AddNode` preserve present-empty `[]` as `[]`; assert the
   clone pointer remains non-nil and serialization contains `"icons": []`,
   not `"icons": null`.
10. A 65-glyph list remains valid at the format layer, proving no new parser
    cap leaked into the format.

Graphview:

1. All 19 palette actions are present with correct labels and keyboard access.
2. Priority selection removes the previous priority and appends the new one.
3. Color selection removes the previous color and appends the new one.
4. Symbol toggles are independent.
5. Unknown glyphs, order, and duplicates survive an untargeted toggle.
6. Targeted duplicate glyph removal preserves all other relative order.
7. Clear-all writes present `[]`; clear on absent/present-empty is a visible
   no-op and creates no history.
8. Existing lists over 64 icons or 4096 aggregate bytes display a bounded count
   and refuse editing visibly.
9. A candidate that would exceed either budget is refused before cloning.
10. Readonly, sync failure, oversize preview, missing node, and stale source
    refuse without a host transaction.
11. Failed pending-field Flush retains that draft and refuses the marker action.
12. A same-node icons change after capture produces a stale/conflict error, not
    a lost toggle.
13. Each real marker action is one undo step; undo/redo restores exact icon
    state and selection.
14. A real edit preserves rich tag metadata and unknown node fields.
15. A no-op click preserves exact source bytes and no history.
16. Light/dark and readonly inspector images show the palette, bounded current
    summary, conflict/refusal state, and retained graph.

## Non-goals

- No tag editing API or UI in this slice.
- No marker search/filter, outline integration, drag reorder, custom
  vocabulary, XMind import, semantic IDs, or style/color tag library.
- No context-menu framework or per-node popover framework; a bounded inspector
  section is enough.
- No parser grammar/version change and no format-side marker cap.
