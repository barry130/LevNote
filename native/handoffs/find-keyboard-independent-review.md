# Find keyboard independent review

Status: FINAL SOURCE REVIEW of the owner-frozen checkpoint, 2026-11-09.
Read-only source/evidence inspection; no test, build, launch, or product/docs
edit was performed. The seven frozen source/test SHA-256 values in
`find-keyboard-result.md` match the current files, as do the two final log
hashes. This review found one concrete lifecycle defect in the requested scope.

## Finding 1 (material): a Find composition flag can survive focus leaving its field

`state.go:273-282` and `state.go:328-337` establish the two flags from
`InputCompose` and clear them only on a later `InputText`, while `closeFind`
clears them as part of closing the panel. The toolkit does not necessarily
deliver either observer event when focus leaves a composing text input:

- `ui/input.go:1393-1400` detects a changed text-input client and calls
  `old.release()`.
- `ui/text_input_client.go:98-103` implements release by calling the old text
  client's `UnmarkText()`.
- `ui/editor_input.go:257-265` commits/clears that input's internal composition
  and calls `publish()`, but does not emit `InputCompose` or `InputText` through
  `ObserveInput`.

Concrete trigger: focus `查找内容` or `替换内容`, start a composition, then click
the source editor or another tab while Find remains open. The toolkit ends the
old input target and commits/clears its marked text, but the corresponding app
flag can remain true. On a tab switch the old field callback would also reject
any late event because `fieldCurrent()` requires the captured tab/document/
editor/buffer to remain current (`state.go:258-260`).

Impact: after composition has visibly ended, the App still reports that an IME
is composing. Repeated Ctrl+F is refused; Enter/Escape are consumed by the
composition guards; toolbar close and Find actions are blocked until Find is
closed by some path that clears both flags. This is a stale UI state, not a
physical-IME uncertainty.

The same lifecycle hole exists when `ShowPlugins` hides the entire Find subtree:
`view.go:83-88` returns before `findbar`, removes the focused field without an
App-level reconciliation, and can also leave a one-shot `findFocus*` request
unresolved until Find is rendered again. The latter alone was not observed as
an ordinary-frame focus steal, but it should be invalidated with the same
transient-state cleanup.

Narrow recommendation: when neither relevant Find field remains focused, or
when the Find subtree is hidden/recreated because of tab or presentation
changes, clear the abandoned composition flags and pending `findFocus*`
reference while preserving `Query`, `Replacement`, and options. Do not invoke
`Editor.ReleaseView` or otherwise mutate the source draft. Add one actual App/
Tester regression that composes in query/replacement, moves focus to the source
or another tab, then verifies a normal Ctrl+F/Enter path is immediately usable.
No additional framework is needed.

## Checked and not reported as defects

The implementation correctly guards Ctrl+G before taking the shortcut, consumes
Find Enter/Escape while the tracked composition is active, and keeps generation
plus exact tab/document/editor/buffer checks around navigation and focus
restoration. Replacement checks target identity after Flush before mutation;
failed and identity-changing Flush tests preserve the relevant drafts and avoid
wrong-target replacement. Query and replacement pointer buttons preserve focus
by `KeepFocus`, and the conditional global Escape registration leaves focused
Find actions on their own path.

The existing tests cover Tester synthetic composition, failed Flush, stale
buffer callbacks, Ctrl+G, Escape restoration, and replacement identity. They do
not cover the focus-leaves-composing-field trigger above, so the passing frozen
suite does not close this finding. Physical OS IME candidate/Escape behavior
remains an explicit scope limit.

Evidence reconciled without execution:

- `find-keyboard-evidence/final-app-ui-editor-test-2.log`:
  `6BF9A32F9C724525DE82354B24D05995D9343322543F5A811CF496500F4529D5`
- `find-keyboard-evidence/final-app-ui-editor-vet-2.log`:
  `55136E372CC06096357CBA87F588CCC89781D15DC38B9BC7BEEF9078BF7A4185`
