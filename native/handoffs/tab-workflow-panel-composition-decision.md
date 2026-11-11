# Narrow tab-shortcut composition barrier

ACTIVE decision2026-11-10 after Astra isolated the synthetic failure. Root
accepts the bounded guard for this shortcut only. This overrides the earlier
plan expectation that composing Find/GoToLine fields could always switch.

Keep zero/one-tab no-op before all checks. For two or more tabs, if an active
visible Find query/replacement or GoToLine field is composing, refuse adjacent
navigation with a concise visible message asking to finish composition and
try the shortcut again. Preserve field focus, committed value and preedit,
document/tab/draft/undo/export state; do not automatically queue a later switch.
Use existing visible-panel composition state, not stale hidden-field flags.

This guard is not a blanket editor IME refusal. Source composition still uses
existing Select/Flush commit behavior, and rich Flush failures still preserve
their drafts. Do not alter vendor or generic Select, reconstruct marked text
by string concatenation, or introspect toolkit internals. There is no public
commit API for the built-in utility field, and keeping its draft is preferable
to silently clearing it while switching.

Add focused synthetic cases for query, replacement and GoToLine; both
directions refuse during composition, values/preedit/focus remain, and an
explicit later shortcut after commit/cancel works. Verify no stale guard after
panel close. Preserve the original failing log. Continue active reveal and one
final affected app test/vet. No physical OS/IME claim. Briefly disclose this
exception in the result; usage documentation follows accepted delivery.
