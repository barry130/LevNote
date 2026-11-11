# Final picker review decisions

ACTIVE2026-11-10 for Astra only. Root read independent WIP review and owner's
handle finding. Complete these before freeze; no generic framework or vendor edit.

1. Same-tab mutation coverage: existing same/remove and same/reorder tests
   correctly accept because they leave the active target untouched. Preserve
   them as unrelated-target cases and add real same-target removal/replacement/
   reorder during Flush (origin moved/replaced at captured slot). Assert visible
   refusal, no selecting another slot, no wrong release or byte/history mutation.

2. Public MyGo Handle has no disposal API and appending a new Handle binding
   adds a ref retained by the UI engine until window close. Preserve just two
   stable public handles (query and list) across filter ListState resets AND
   close/reopen for this app window, cancelling pending focus on every close.
   This explicitly refines earlier 'clear handles' wording: clear all metadata,
   selected IDs, messages, pending actions, callbacks and ListState internals,
   but keep/reuse opaque two-handle identity in separate app-owned state or
   clean picker reset. These handles must hold no app document/view/content
   callbacks. Generation still changes every opening; bind same handle to the
   new query/list element. Shutdown cancels requests. Prove repeated openings
   reuse handles and restore opener/focus; do not inspect private toolkit fields.
   Root prefers this small bounded identity retention over one new handle/window
   ref per opening. No toolkit workaround/fork.

3. Metadata refresh now reuses strings/folds and has measured zero allocations
   when unchanged. Root accepts one bounded scan of up to100 retained-length
   title/path prefixes while visible to observe current external title/path/
   readonly/dirty changes. It is not a skipped CPU-work claim. Do not spread
   change counters through existing app/format code or retain full original
   paths to avoid this bounded scan. Remove duplicate same-query refresh in the
   same view pass if not already done; immediate TextKey freshness remains.
   Document helper allocation evidence and bounded scan honestly. This resolves
   the reviewer CPU concern as an explicit small tradeoff, not a benchmark claim.

4. Tab/ShiftTab composition regression plus real draft/focus tests must pass
   after the latest runtime changes. One final focused union is justified after
   these concrete refinements; then freeze/publish result+source identities for
   independent review. The later final app test/vet remains separate planned
   acceptance. Save a single optional layout image if still needed. No package.

5. Root viewed the actual06 capture. Selected teal row uses gray explicit
   TextMuted for path and badges, making them difficult to read against the
   selection background. Set selected-row secondary text to theme.AccentText
   (public list selected-state policy), keep unselected muted. This is a small
   readability fix, not restyling. A single replacement narrow capture is justified
   by this observed defect; preserve06 original. Include selected row plus normal
   path/duplicate-name rows. No global theme change or new visual dependency.
