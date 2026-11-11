# Next-stage active composition retention decision

2026-11-10. Root accepts source finding in active-transaction-memory-review.md:
committed32MiB history and fixed logical-content ceiling do not bound count of
active IME group intermediates. This is not measured process memory. It is a
next source stage AFTER .28 closure; no implementation authorized now.

Direction: opt-in composition group/coalescing contract at text layer used only
by Editor-owned IME lifecycle. Preserve original replaced bytes plus current
marked bytes for consecutive exact full-span updates, rather than every
intermediate string. Keep same source revisions/callbacks/selection and one-step
undo/cancel/redo. Ordinary BeginGroup commands retain behavior.

Refinements before dispatch:
- Native final ReplaceText commit also passes through the marked span and
  must participate or safely append exactly once; don't optimize only Compose
  then accumulate a new series through a forgotten input seam.
- A composition may begin with a no-op/same-text preedit and selection movement;
  preserve its original selection and handle zero edit state coherently.
- If normalized explicit range is irregular, retain existing append semantics
  for that transition. Consecutive later replacements of the most recent marked
  segment may compact that suffix if provably safe, but never claim arbitrary
  native sequences are universally bounded unless they actually are.
- Refused input/sticky error, foreign group/changed buffer, failed/retried edit,
  cancellation, prior redo, selected surrogate pairs and callbacks are required
  test cases. Update active byte accounting when replacing a retained next.
- Prefer avoiding the old intermediate slice allocation in the proven fast
  path as well as releasing obsolete references; no snapshot/full-document copy
  or group-wide serializer introduced. Avoid overflow in history byte counts.
- A typed/owned composition group mode is clearer than permitting arbitrary
  code to rewrite ordinary group history. Keep API small and compatibility
  clear; implementation owner may choose exact method names.

Tiny structural tests should establish retained edit/string count for hundreds
of legal updates and exact cancellation/undo. One source allocation benchmark
is optional only if it substantiates this specific change; no whole-app RSS
claim and no repeat existing worker matrix. Root assigns after current .28 is
frozen; current source owner must not broaden into this stage prematurely.
