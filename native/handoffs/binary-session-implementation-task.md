# Path-only binary recovery records

ACTIVE2026-11-10 for GLM read_native_handoffs_binary_image_glm_task_md after
binary-core-root-acceptance.md. Ownership is core/session/** only, plus
evidence/handoff. No app/document/vendor changes,
no child agents, no product package or host/worker runs. Existing binary core
and .22 package stay unchanged. Read binary-session-app-next-plan.md for context.

Implement a bounded additive Version1 branch for binary records. Keep all
existing text-only JSON bytes/order/escaping, recovery limits, draft accounting,
writer error/atomic behavior and old text RestoreTab results. Binary records
retain path and small validated metadata only (explicit binary discriminator,
validated kind, optional title/plugin/view). No document bytes/snapshot/pixels,
draft, encoding, BOM, OriginalPath, text selection or scroll. Choose exact key
names once and record the contract for the later app owner; unknown discriminator
or kind is a reported invalid record, never coerced into text.

Do not add MarshalJSON to the draft-bearing Tab: it causes extra full draft
encoding/copies. Keep the existing ordinary text Save path as-is. For mixed
sessions use a small snapshot wire projection containing original ordinary Tab
values and distinct small binary wire values; copying descriptors/string headers
is fine. Bound the projection by MaxTabs. Retain independent stdlib Marshal
oracle tests for text-only exact bytes and a separately declared binary/mixed
wire oracle, including actual output length equal to preflight.

Load must inspect wire key presence for binary records, rejecting even empty,
null or false forbidden text fields. Do this only within the existing first100
boundedTabs entries. Avoid copying the full tabs array into RawMessages and
avoid duplicate draft-sized decodes on ordinary text tabs. A bounded key/type
scan or metadata-only discriminator scan over each existing slice is suitable;
then use the appropriate branch. A malformed binary entry must not discard
neighboring text drafts or shift a valid active record incorrectly.

Expose a pure small binary restore reference for the later host dispatcher;
do not open files or retain full snapshots in session restoration. Until the
app owner wires that dispatcher, the EXISTING RestoreTab API must report an
explicit binary-dispatch-required error for binary records, not return success
with Document=nil (current callers dereference it). Keep ordinary text API
behavior and avoid changing all existing callers in this isolated stage.

Add meaningful regressions for mixed capture/save/load, no forbidden keys,
unknown kind/discriminator, explicit forbidden key values, damaged binary between
good drafts, active mapping, nil/empty tabs, MaxTabs/MaxBytes boundaries and text
byte-compatibility. Maintain builder clone independence and rejection before
filesystem creation. Reuse existing suites rather than duplicate tests. One
focused run then one final session suite/vet after actual fixes; save exact
commands/stdout/stderr/exits and retained failures in
native/handoffs/binary-session-evidence-20261110. All Go through scripts/go.ps1.

Publish binary-session-implementation-result.md with actual contract, remaining
app integration dependency, limits and frozen scoped file identities. No claim
that images are already enabled in the application. Root then assigns the next
bounded implementation step without user choices.
