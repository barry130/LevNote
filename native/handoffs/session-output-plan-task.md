# Session output allocation: bounded read-only plan

2026-11-09. ACTIVE for DeepSeek
`read_native_handoffs_release10_ui_feasibility_task_md`. UTF8 source is accepted
and frozen during .15 delivery. No children. Read only product source; write
only `native/handoffs/session-output-plan.md`. No implementation/tests/builds,
benchmark/launch, new dependencies, product docs or historical evidence edits.

Goal: choose the next small justified memory-related slice. Inspect current
core/session/session.go, builder.go, budget.go and app/state.go persistence.
Root observed dirty capture creates full strings, Save preflights exact JSON
length then json.Marshal materializes encoded output. Identify actual avoidable
copies/retention from source; separate them from process-memory hypotheses.

Recommend at most one bounded change after .15. Prefer reusing existing exact
length/budget logic and immutable strings, preserving public Snapshot/Tab API,
file format, exact encoding/json byte behavior, one atomic private-state write,
all validation-before-write and current rejection/draft retention/retry behavior.
Consider an exact-size output encoder to remove generic encoder growth/copy;
do not assume json.Encoder streams individual tokens (verify its implementation
if relevant). Do not propose broad async persistence, persistent draft caches,
custom compression, new recovery format or multiple optimizations together.

Describe smallest safe seam, ownership, byte oracle (including omitempty,
nil/empty tabs, zero/float32 formatting, escapes, Unicode/U+2028/U+2029, field
order, valid input bounds), failure/atomic safety and decisive tests. Keep
current 4MiB per-draft/16MiB session/100-tab bounds. A benchmark would measure
encoding/output allocation only on deterministic valid 1MiB and multi-tab
fixtures under existing caps, fixed bounded iterations, setup excluded and
exact output equivalence. No whole-app/RSS claim. If exact-size encoding adds
too much correctness surface for a worthwhile benefit, say so and recommend
one simpler evidence-backed next step. Deliver a concise actionable plan, not
an implementation or a generic audit.
