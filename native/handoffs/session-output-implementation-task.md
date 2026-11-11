# Session JSON output: remove the Marshal returned copy

ACTIVE after root accepted the immutable .15 package. Owner
DeepSeek read_native_handoffs_release10_ui_feasibility_task_md. No children.
Read native/AGENTS.md, session-output-plan.md and the clarifications below.
Only core/document/state.go plus focused state tests, core/session/session.go,
one small output helper and substantive tests/benchmark. No app/format/UI or
budget/builder changes absent a demonstrated bug. No docs/builds/launches.

Accept the plan's small direct-Encoder alternative. Reuse the shared atomic
writer lifecycle through a private-state callback seam; the existing []byte
WritePrivateState caller keeps semantics. The Encoder still owns a full JSON
buffer; only its extra returned Marshal copy is removed. Do not call this a
token-streaming or fixed-memory JSON encoder. No format, snapshot, draft-limit,
retry, cleanup, recovery or document-save behavior changes.

Save sets Version and fully validates via preflightSnapshot before any output
or directory/temp creation. Use its exact expected length as a postcondition.
Remove only Encoder's one mandatory terminal LF, without changing any JSON
bytes, escaping defaults, field ordering or omitempty/null behavior. Use a
bounded forwarding adapter, no full output buffer. Handle multiple Write calls
and empty writes deliberately. Detect short writes yourself: bundled Encoder
checks returned error but not short n. Preserve exact sentinel and short-write
errors through the helper and atomic callback. Wrong expected length, absent
terminal LF, excess bytes, truncated output or finalization failure must abort
before publication. Do not assume one Write or expose partial recovery files.
Reject an invalid/nil callback before making filesystem changes if the seam is
exported. Preserve private permissions, create/replace, sync/close and existing
post-publication directory-sync semantics.

Tests should compare new output byte-for-byte with json.Marshal for VALID
snapshots: nil/empty tabs, omitted/present fields, explicit LineNumbers false,
controls/quotes/slash/HTML escapes, CRLF/CJK/emoji/U+2028/U+2029, saved paths and
dirty draft copies, selection, BOM/encodings. Float32 extremes belong in Scroll
(nonnegative finite) or a direct helper oracle, not invalid FontSize/SidebarWidth
fixtures. -0 omitempty should retain standard behavior. Cover meaningful
4MiB/100tab/near16MiB bounds with existing fixture patterns, just-over and invalid
input rejection before creation, old recovery bytes/time preserved, successful
Save/Load, wrong lengths and short/sentinel failures reaching a real temp writer
then removing it. No extra framework; reuse established tests.

Benchmark encoded-output only: two deterministic valid fixtures (1MiB draft,
several tabs totalling about1MiB), identical snapshot/preflight outside timing,
old Marshal plus actual io.Discard write versus new helper to same sink. Same
oracle policy in both; setup, fixture hashes and byte-equivalence oracle outside
timing. Use b.N with -benchtime=10x. Preserve raw output and state stdlib pooled
buffer limitations, exact emitted byte counts, B/op and allocs/op. No fake
fixed10 loop, arbitrary forced GC, whole-Save/CPU/RSS inference or repeated
benchmarks unless a real failure requires correction.

Use approved native/scripts/go.ps1 wrapper only. Preserve actual focused failures
and fixes under native/handoffs/session-output-evidence with new filenames.
Finish with one affected document/session/app test suite and vet after final
corrections; no unrelated all-packages or worker repetition. Publish frozen
session-output-implementation-result.md with source/evidence hashes, commands,
equivalent-work benchmark and limitations. Independent read-only review then
root acceptance are required before a future package. Keep .15 immutable.
