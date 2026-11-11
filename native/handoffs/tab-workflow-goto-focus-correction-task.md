# Tab workflow: reconcile GoToLine composition after pointer focus leave

ACTIVE root review correction2026-11-10. Owner remains Astra tab-workflow agent.
Root has not accepted the tab stage yet. The completed result explicitly records
that GoToLine's composition flag persists after pointer-only focus leave. The
saved07 failure and the final test's Type(preedit) exception show that this is
an unresolved behavior, now affecting the new Ctrl+Tab guard.

Reproduction: open GoToLine, Compose preedit, Ctrl+Tab refuses correctly, click
the source editor to leave the field, then explicitly retry Ctrl+Tab. The native
input composition has ended, but GoToLineComposing stays true and the visible
panel causes an indefinite inaccurate refusal.

Ownership is extended narrowly to native/app/source_navigation.go and focused
tests. Reconcile the field's flag using public actual Focused()/Composing()
state after pointer focus leave, following the already working Find lifecycle.
Keep actual focused preedit protected. No vendor changes, no blanket removal
of the guard, no synthetic text reconstruction. A frame-local input handle is
permitted; don't retain widgets. Keep the existing keyboard commit/cancel cases
and add the real pointer-leave path (value/preedit survive, no automatic switch,
explicit subsequent shortcut succeeds, no source edits/undo loss).

Preserve all prior logs. Run the new focused regression then one affected app
test/vet after the correction. No package/host/resource run or docs changes.
Update tab-keyboard-implementation-result.md and final-file-identities.json
with the new file and final evidence; publish concise progress and freeze.
