# Find keyboard checkpoint

2026-11-09. The initial App/Tester red is preserved in
`find-keyboard-evidence/initial-keyboard-red.log`: Ctrl+F did not focus the
query, query/replacement Enter did not act, Find-field composition did not
protect Escape, and source composition did not block opening.

The first narrow keyboard implementation still failed query focus after Enter
(`narrow-keyboard-pass.log`, despite its filename). Root's diagnosis was
confirmed: the window Escape shortcut ran before field handling, and
`Handle.Focus` requests are applied in handle-registration order, so the source
reveal could win over the query request. The failed follow-up with the
same-frame deferred request is preserved as `focus-fix-focused-1-retry.log`;
an earlier wrapper log-path mistake is preserved as
`focus-fix-logpath-failure.log`.

The adopted target fix is bounded:

- scope the global Escape shortcut before registering it;
- keep Find focus intent generation-scoped and one-shot;
- require the exact post-action tab/doc/editor/buffer identity;
- use the root-authorized neutral `Editor.CancelFocus` forwarding method to
  cancel only the source reveal focus request, never `ReleaseView`;
- cancel both Find handles and any pending focus intent when Find closes.

The currently authored focused tests pass in `focus-fix-focused-2.log`. This is
not the final checkpoint: composition commit/cancel, fresh TextKey replacement,
read-only, Flush identity/lifecycle, toolbar buttons, live Markdown composition,
Ctrl+G/layout regressions, and the one final affected suite/vet remain.
