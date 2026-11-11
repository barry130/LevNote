# Read-only composition contract review

ACTIVE2026-11-10 for /root/read_native_handoffs_live_tab_admission_implementation_task_md.
No children, source edits or execution. Current growth source remains frozen
for final validation and .28 delivery. Read the next implementation task and
active-transaction-memory-root-plan.md; this is a small review for the already
selected increment, not another memory audit or architecture proposal.

Inspect current Buffer group/history and Editor marked/final/cancel paths.
Write active-composition-coalescing-contract-review.md with at most a few
concrete constraints needed for correctness: group ownership if Restore or
CancelGroup/BeginGroup changes the current transaction on the same buffer;
no-op/same-text initial composition and return-to-original text; suffix
coalescing after an irregular explicit replacement; per-update revision vs
undo/cancel revision behavior; obsolete string references and safe byte counts.
Recommend the smallest API that preserves ordinary group semantics and does
not retain a stale transaction through an ownership token after cancellation.
Distinguish host-reachable behavior from unsupported direct Buffer mutation.
No invented blocker without a source path. Existing implementation task already
sets the policy, so no new design round is needed. Save a concise actionable
artifact promptly, then stop. You may perform the independent final source
review later only after root follow-up; no tests or build now. Keep context below200K.
