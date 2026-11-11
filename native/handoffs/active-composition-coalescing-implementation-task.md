# Compact active IME transaction history

ACTIVE2026-11-10 after .28 package/evidence closure and
growth-release28-root-acceptance.md. Owner Astra /root/read_native_handoffs_tab_workflow_resume_task_md.
No children. Root owns planning/acceptance; you own implementation and focused
validation. Read active-transaction-memory-root-plan.md and the prior source
review. Those establish reachable intermediate retention, not a heap/RSS result.
Also read active-composition-coalescing-contract-review.md and the overriding
active-composition-coalescing-root-refinements.md, particularly same-text
callback preservation and buffer-bound tokens. No new design round is required.

Small scope: native/core/text/buffer.go, native/ui/editor/editor.go and focused
new tests in those packages. A narrow app regression file is permitted if needed
to prove actual keyboard/lifecycle integration. No document/search/format/vendor
runtime edits or new UI. Preserve .28 source evidence and immutable packages;
new evidence goes under native/handoffs/active-composition-coalescing-evidence-20261110.

Implement a small opt-in composition-group API, exclusively used by the editor's
owned IME lifecycle. Ordinary BeginGroup behavior must remain identical. For a
consecutive accepted replacement covering exactly the previous full marked
segment, keep the original replaced bytes and the current accepted replacement,
not every old/new intermediate. Preserve original selection, current selection,
per-update revision and callbacks, ownership and normal history trimming.
Use UTF16 boundaries with existing validation. Avoid an unnecessary old-segment
allocation on the proven fast path; do not add whole-document snapshots.
Maintain safe retained-byte accounting without arithmetic overflow.

Include the native final ReplaceText commit. No-op/same-text first preedit must
retain the correct original selection/ownership even if there is initially no
edit. Irregular explicit replacement ranges must safely fall back to ordinary
append semantics; compact only a suffix whose full-span relationship is proven.
Document this scope: arbitrary irregular ranges are not universally bounded.
Do not retroactively coalesce ordinary commands or force commit/drop to meet a
budget. Accepted normal input still produces the same text/revisions/events.

Retain all .28 refusal semantics: a size-refused preedit is sticky, valid retry
or explicit Escape resolves it, rejected strings are not retained. Save/select/
close cannot bypass it. Foreign group/changed buffer cannot be adopted or
cancelled accidentally. A failed replacement leaves retained history and prior
redo intact. Cancellation restores the exact original text and selection;
successful commit is one undo/redo action with existing truncation rules.

Meaningful tiny tests: hundreds of exact full-span updates with ASCII, CJK and
surrogate pairs retain a bounded edit count and only original/current strings;
per-update version/callback accounting; selected/nonselected initial span;
same-text/no-op initial preedit; final commit, cancel and earlier redo; irregular
explicit ranges with correct fallback undo/cancel; first/ongoing refused update,
valid retry, foreign group and buffer replacement. Exercise real Tester compose
and local/global Escape routing for the editor integration without claiming OS
IME coverage. Tests may inspect private transaction representation in core/text.

First save a concise implementation approach/progress checkpoint. Run only the
new focused tests plus existing composition/growth seams affected by the change.
Use native/scripts/go.ps1, capture actual exits and separate streams, retain
failures and make justified corrections. No blanket full suite, builds, hosts,
screenshots, worker matrix or benchmark unless root later activates it. A small
structural retention proof is enough; no whole-app memory claims. Publish
active-composition-coalescing-result.md with runtime API, fallback boundaries,
actual verification and frozen source identities. Independent review and one
final affected gate follow before the next release. Keep context under200K.
