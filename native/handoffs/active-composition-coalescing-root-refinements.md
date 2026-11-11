# Root refinements to composition contract

2026-11-10, read-only review accepted as direction after .28 closure. Read the
contract review with these corrections; they override ambiguous reviewer prose.

1. Preserve actual callback behavior. Current Editor.SetMarkedText calls
   changed() for every accepted marked update, including same-text preedit.
   Buffer does not advance text revision on same text. Do not interpret review
   item6 as permission to suppress this existing callback. Coalescing must add
   no extra callback; accepted marked/final paths retain their current callbacks.
   Aggregate cancel/undo/redo need not reproduce N intermediate revisions.
2. A token must be bound to both its issuing Buffer and the exact current
   composition group. A merely per-buffer integer counter is not enough if
   equal counters on two buffers accept a foreign token. Use a small private
   identity design; token must not hold a transaction, edits, or original text.
   Clearing a stale editor token must release obsolete owners. Avoid a pointer
   to a zero-sized allocation as a unique token (Go may equate such pointers).
3. Typed composition groups may reject tokenless mutation/end/cancel; ordinary
   groups must retain their current API and behavior. Restore/ResetHistory
   invalidate typed ownership. Escape against stale ownership must not cancel
   a new foreign group. These direct same-buffer mutations are defensive tests,
   not a newly claimed UI interaction or transaction framework.
4. Net-zero suffix handling is accepted: if the active aggregate returns to its
   original text, a completed no-text-change composition should create no fake
   undo action and should preserve prior redo. Keep the before-selection and
   marked baseline for a later update or cancellation. Irregular previous
   suffixes must still undo/cancel correctly. Test this explicitly; do not
   conflate no text revision with no accepted-input callback.
5. Saturated accounting must not lose an unknown overflow after a suffix shrinks.
   If using saturation, retain an overflow state or safely recompute from
   retained edits when needed; otherwise do checked totals. Normal full-span
   path remains O(1) transaction retention. Irregular append histories are a
   disclosed fallback, not a universal bounded-memory promise.

These refine the existing small implementation task; no extra design stage or
pre-.28 source edits are authorized.
