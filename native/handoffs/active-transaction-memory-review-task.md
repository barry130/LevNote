# Next resource increment: active edit transaction audit

ACTIVE READ-ONLY2026-11-10. Owner DeepSeek
/root/read_native_handoffs_live_tab_admission_implementation_task_md. Previous
growth draft review is accepted and routed to Astra. Do not edit current code,
tests, docs, packages or run anything; no child delegation; compact before200K.

While Astra completes the current growth/refused-composition stage, inspect one
separate potential resource gap: text.Buffer grouped edits append every old/
next string during composition; committed history is limited32MiB/256entries,
but an active group may retain intermediates until EndGroup/CancelGroup. Verify
actual implementation and usage. Distinguish finite logical content bound,
transaction metadata, unique backing strings, and history trimming. Do not
claim measured memory or automatically demand a broad transaction redesign.

If a source-grounded unbounded retention path exists, propose ONE smallest
safe follow-up that can be tested with tiny limits. Preserve original source,
one-action composition undo, cancellation to the exact original, prior redo,
foreign group ownership and accepted-history guarantees. Ordinary multi-edit
group commands and IME have different shapes; an IME-specific coalescing seam
may be smaller than a universal transaction memory cap. Account for changing
replacement ranges, surrogate boundaries and prior emitted/source callbacks.
Never silently discard current group or force commit just to meet a budget.

Publish active-transaction-memory-review.md promptly with exact code trigger,
known finite bounds, minimal scope, and test strategy; if no reachable issue,
say so. This is next-stage planning only and must not expand the current .28
implementation. Do not repeat broad document-growth design or whole-app audit.
