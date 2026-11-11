# Find keyboard independent reconciliation

Status: FINAL READ-ONLY RECONCILIATION, 2026-11-09.

The original material finding is resolved. No remaining material defect was
found in this focus-leave scope. This review made no source, test, documentation,
build, launch, or product change.

## Resolution

`search_navigation.go` now provides three bounded lifecycle operations:

- `clearFindTransient()` clears both App composition flags, cancels both Find
  field focus handles, and drops the pending Find focus request.
- `expireFindFocus()` invalidates a pending request when Find is hidden, Plugins
  is open, the Find generation changed, or the captured tab/document/editor/
  buffer identity is stale.
- `reconcileFindFieldComposition()` clears a field's flag only after the public
  `Element.Focused()` and `Element.Composing()` results show both that focus has
  left and that marked text has ended.

`state.go` binds the rendered query and replacement elements, reconciles both
composition flags, and only then resolves the one-shot focus request. `view.go`
expires pending Find focus on each frame and clears the same transient state when
Plugins hides the Find panel. Values and options remain independent of this
cleanup.

This directly repairs the original case where focus transfer calls
`UnmarkText()`, commits the field, and emits no `InputText` observer callback.
It uses public toolkit state rather than relying on a callback that the toolkit
does not guarantee.

## Lifecycle evidence

The red log reproduces stale query and replacement composition after source
focus, stale query lifecycle after a tab switch, and stale query composition
through Plugins. The corrected green log passes all seven cases in
`app/find_keyboard_lifecycle_test.go`: query/replacement to source,
query/replacement to another tab, query/replacement through Plugins hide/show,
and cancellation of a hidden pending focus request.

The cases check the committed field value, preserved field/options state,
unchanged source text/version/history, immediate Ctrl+F/Enter/Escape use, no
later focus steal, and no retained stale pending target identity. The final
affected app and editor suites passed at 14.326s and 2.083s with exit 0; the
following vet passed with exit 0 and no output.

## Verified frozen identities

All values below were hashed in this review and match the owner checkpoint.

| File | SHA256 |
|---|---|
| `native/app/app.go` | `F57743A4CAD59C95B3FB13D8B8C982C99CBFE43B979ED7EB447ACD51B53C6AC3` |
| `native/app/search_navigation.go` | `57D56DA4B266E9381D8F2268A26D8CFC33ABA4B968937D3AB753F32ED4763E77` |
| `native/app/state.go` | `63BBC1372A26D862D8B69D755E50FC7880C6474C36121F4A0A637DDA852E8D38` |
| `native/app/source_navigation.go` | `49FA79D5D3FC7B38B46363874780E421E3DABE363F6C28F1C3FAE78B12806113` |
| `native/app/view.go` | `CA2D62B0F873A86FADB92DBA525BE09E25DA04F44A7969D6D9D44DA532EFD640` |
| `native/app/find_keyboard_test.go` | `1C9D0408DDE6899E134351763E9373D61E7A0537E8B150A1EB2CB2AEC64FAD81` |
| `native/app/find_keyboard_lifecycle_test.go` | `04C55F900DA46173EA1A757B808B1106C78F0196F295948E6CFD6B94A284D680` |
| `native/ui/editor/editor.go` | `6438B438F0409BEEB26A9427C5CC8EBF971132242933B1D927BEBFEA4371AC81` |

| Evidence | SHA256 |
|---|---|
| `focus-leave-lifecycle-red.log` | `550592E5F9145AA471D78DB3F20E705B8E16CE4A4E29C58581386DC10962A601` |
| `focus-leave-lifecycle-focused-green.log` | `D09D1C6B7515918A3386F2421073F9DF4290E2500303226A77E5CC732B4C8C6B` |
| `final-app-ui-editor-test-3.log` | `49F7F745B49E4D8551C48C74FB8F696EFBA6D1306BE99B59F8E1D29EBDE86B1B` |
| `final-app-ui-editor-vet-3.log` | `55136E372CC06096357CBA87F588CCC89781D15DC38B9BC7BEEF9078BF7A4185` |

## Scope limit

The lifecycle tests provide App/Tester synthetic composition evidence. They do
not prove physical Windows IME candidate selection or physical Escape behavior,
because no application launch or physical IME test was performed. No memory or
performance claim is made.

The original `find-keyboard-independent-review.md` remains unchanged as the
historical finding. `.14` delivery remains separately staged and was not
activated by this review.
