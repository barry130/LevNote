# Next resource increment: document growth admission audit

ACTIVE read-only planning2026-11-10. Owner GLM
/root/read_native_handoffs_binary_image_glm_task_md. Current reopen/cap source
is frozen for final validation/.27 delivery. No runtime/test/product-doc edits,
test execution, builds, launches, benchmarks or child delegation. This task
does not activate any new source work. Keep the result short and source-based.

Root observation to verify: core/document.MaxEditableBytes128MiB is applied
to stable file reads, while core/text.ReplaceUTF16 and editor ReplaceText/paste
may grow a live document without the same ceiling. Existing32MiB history limit
is a history budget, not necessarily the document content limit. The current
100-tab cap addresses count only; do not claim its worst-case content is bounded
at128MiB per tab unless actual write admission guarantees that.

Trace real edit origins: normal typing/IME, paste, replace current/all, format
transactions, recovered draft/template, undo/redo/snapshot restore. Distinguish
reader limit, parser limits, recovery draft limit, history bytes and actual
logical-buffer growth. Check overflow/huge incoming value handling and when
allocation occurs, without constructing huge fixtures or running anything.

Publish document-growth-budget-next-review.md: concrete present gap or explain
why no gap; one smallest useful next increment with ownership/API/error UX and
preservation requirements; focused small-limit test strategy if needed. Preserve
normal below-limit editing, shrinking/replacing content, IME transaction owner,
undo/redo and source fidelity. Avoid global-memory promise, broad scheduler/cache
framework or silent truncation. If a reader-limit label in current docs is
overbroad, note exact text for the later docs owner rather than editing it.

Prioritize this one audit; when complete stop and release ownership. Root will
decide next implementation only after .27 package closure. Do not invent a
fixed number merely to appear strict; reuse existing product bounds where
appropriate and describe any compatibility issue explicitly.
