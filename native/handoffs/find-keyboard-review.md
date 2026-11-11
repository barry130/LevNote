# Find keyboard behavior: read-only review

2026-11-09. Scope: current `native/app/view.go`, `native/app/state.go`,
`native/app/search_navigation.go`, the accepted Ctrl+G pattern in
`native/app/source_navigation.go`, and current MyGo TextInput semantics. No
source/test/doc edit, test, build, or launch was performed.

## Current behavior

1. **Ctrl+F reveals the bar but does not establish query focus.**
   `view.go:524-526` only assigns `a.ShowFind = true`. Unlike
   `openGoToLine` (`source_navigation.go:96-115`), it has no open helper,
   generation, source/rich composition check, or focus request. Both Find
   inputs are plain `TextInput`s (`state.go:236-266`): neither has
   `AutoFocus`, `OnTextKey`, `ObserveInput`, or a `Composing` check. Thus the
   source supports no query focus on the first or a repeated Ctrl+F. Repeated
   Ctrl+F leaves visibility unchanged. `AutoFocus` would not repair repetition
   because it acts only when the element first appears.

2. **Enter does not run Find.** The query and replacement fields have no key
   callback. Enter may receive normal single-line TextInput handling, but no
   app code calls `a.find(true)` or `a.replace(false)`; Shift+Enter likewise
   has no previous-result action. Only the buttons at `state.go:240-260`
   invoke Find/Replace.

3. **Normal Escape closes the bar without restoring document focus.**
   `view.go:532-536` handles window Escape by calling `closeFind()`, which
   clears visibility and invalidates the cache (`search_navigation.go:52-55`).
   It does not focus the current source or rich editor. This also closes the
   other panels in the same branch. Ctrl+G is different: `cancelGoToLine`
   explicitly restores focus only when its captured tab is still current,
   open, and not preview (`source_navigation.go:146-151`).

4. **No Find-specific composition state exists.** MyGo documents that
   `OnTextKey` sees non-text keys, but an active input method owns its keys
   (`ui/textinput.go:143-160`), and `Composing()` exposes that state
   (`ui/textinput.go:281-288`). Because the Find fields adopt neither the
   observation nor a key handler, source code cannot preserve Find behavior
   around composition. In the synthetic tester route, Compose followed by
   Escape reaches the window shortcut and closes Find because there is no
   field handler to consume it. Exact physical IME interception was not
   launched or observed here; source review alone does not justify claiming
   that behavior.

The navigation/cache implementation already owns source/rich Flush, source
reveal, query/version checks, and active-tab abandonment
(`search_navigation.go:39-95,140-193`). A keyboard change should call those
existing paths rather than duplicate them.

## One justified next slice

Add a small app-owned Find keyboard flow, following the proven Ctrl+G shape:

- Add generation-scoped Find open/close helpers. Ctrl+F should reveal Find and
  focus the query for first and repeated invocation, but should not shift focus
  away from an active source/rich composition or an actively composing Find
  field.
- Give the generation-keyed query field `AutoFocus`, `ObserveInput`, and
  `OnTextKey`. Enter runs `find(true)`; Shift+Enter runs `find(false)`; Escape
  closes and restores focus to the current live editor when it is not preview.
  Do nothing while that field is composing.
- Apply the same composition-aware Escape and Enter-to-replace behavior to the
  generation-keyed replacement field. Use `KeepFocus` for controls that should
  retain field focus. No new search algorithm, cache, panel framework, or
  warning system.
- Leave `find`, `replace`, Flush, cache identity checks, and current-tab
  semantics unchanged. Scope product changes to `app/state.go`,
  `app/search_navigation.go`, and the Find shortcut/Escape wiring in
  `app/view.go`.

Put focused acceptance tests in a new `app/find_keyboard_test.go`, keeping the
layout owner's `findbar_layout_test.go` conflict-free. Synthetic coverage:
first/repeated Ctrl+F focus, Enter/Shift+Enter direction, Escape focus return,
composing Enter/Escape in both fields doing nothing, and stale callbacks after
close/reopen. Physical OS IME behavior remains an explicit delivery-time UI
check, not a claim from this review.
