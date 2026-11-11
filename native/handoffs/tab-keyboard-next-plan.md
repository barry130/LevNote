# Next tab keyboard plan (read-only result)

Date: 2026-11-10

## Scope

Add the conventional wrapped visual-order tab bindings:

- `Ctrl+Tab`: next tab.
- `Ctrl+Shift+Tab`: previous tab.

This slice has no MRU history, switcher popup, animation, dependency, setting,
or persistent state. Plain `Tab` and `Shift+Tab` keep their existing focus
movement behavior. It is app-level behavior only; no `.21` drawio source,
warning, or source-retention work belongs in the same change.

## Binding and target

Register both chords in `App.shortcuts` with the existing exact-match API:

```go
if c.Shortcut(ui.Cmd, ui.KeyTab) {
    a.selectAdjacentTab(true)
}
if c.Shortcut(ui.Cmd|ui.Shift, ui.KeyTab) {
    a.selectAdjacentTab(false)
}
```

`ui.Cmd` is the existing Windows control modifier used by the neighboring
bindings. MyGo routes a registered window shortcut before an editable focused
element can claim the key. It deliberately does not affect plain `Tab`, which
the toolkit only treats as focus movement when the modifiers are exactly zero
or `Shift`.

Use a small helper rather than a duplicate switching lifecycle:

```go
func (a *App) selectAdjacentTab(forward bool) bool {
    if len(a.Tabs) < 2 {
        return false
    }
    delta := 1
    if !forward {
        delta = -1
    }
    target := (a.Active + delta + len(a.Tabs)) % len(a.Tabs)
    return a.Select(target)
}
```

The method name is an implementation choice; the important properties are the
two-tab guard and delegation to `Select`. Do not add a separate Flush/release/
focus path.

## Lifecycle and reentry

`App.Select(index)` already provides the required switch semantics:

1. rejects an out-of-range index;
2. on the same index, focuses the current editor and closes GoToLine;
3. on a different index, Flushes the old tab for「切换标签」;
4. on Flush failure, reports it and returns without changing `Active`;
5. cancels the old tab's export, releases its presentation, updates `Active`,
   invalidates Find cache, closes GoToLine, focuses the new editor, and marks
   session persistence pending.

The two-tab guard must run before `Select`. `Select(currentIndex)` is not a
pure no-op because it focuses the editor and closes GoToLine, so calling it for
zero or one tab would violate the no-op requirement.

No production `Flush` path currently mutates `a.Tabs` or invokes draft
recovery synchronously. Markdown `View.Flush` can call `Host.Changed`, but the
host only updates edit time and persistence state. Graph `view.Flush` guards
synchronous reentry. `Host.RecoverDraft` is invoked by explicit recovery
buttons, not by `Flush`. Therefore this slice should not redesign generic
`Select` or add a post-Flush pointer revalidation barrier.

If a future production Flush path gains synchronous recovery or tab-array
replacement, it must then capture the target tab pointer before Flush and
refuse the switch if that pointer is no longer open. That is a separate
hardening change; do not bundle it with this keyboard slice.

## IME, Find, and GoToLine

The existing `Select` behavior is correct for input state:

- It closes GoToLine on success; failure preserves both the active tab and the
  current GoToLine target, as existing lifecycle tests require.
- It invalidates the Find cache. It does not close Find, which preserves the
  existing Find-panel semantics.
- `sourceCompositionActive` covers both source `Editor.MarkedRange` and rich
  view `Composer.CompositionActive`, and `flushTab` finalizes source composition
  before switching.

Do not add a blanket IME refusal to the new shortcut. The existing switch
barrier is deliberate: source composition is committed before leaving the old
editor, and a rich-view Flush error must leave its active draft/view alive.

Find and GoToLine field compositions require no new special case for this
binding. They only explicitly claim Enter and Escape while composing, so
`Ctrl+Tab` and `Ctrl+Shift+Tab` route as window shortcuts. A focused text field
does not claim them merely because it is editable. This synthetic routing
statement does not establish physical OS or IME behavior.

If implementation reveals that a real host text input needs to own
`Ctrl+Tab`, add only that focused field's explicit key claim and document the
actual precedence. Do not invent a new modal framework.

## Offscreen tab visibility

The tab strip is `ui.ScrollHorizontal(c.Key("tabs"))`. It currently does not
auto-reveal the newly active tab.

MyGo has a public `Element.ScrollIntoView`, but it must be called on the
element in the frame that should reveal it. The current tab buttons are local
variables inside the tab build closure and the app does not retain their
elements. A bounded implementation could retain the `ui.Button` result for the
active tab and call `ScrollIntoView()` in the same frame that observes the
index change, but this is not necessary for switching to be correct.

Recommendation: defer tab-strip auto-reveal to a later UX slice. The public
element-reference approach is the only acceptable MyGo API route; do not use
widget-tree introspection, private runtime state, coordinate arithmetic, or a
new dependency. The immediate behavior remains correct even when the selected
tab is offscreen: `Select` changes the active document, focuses its editor, and
the tab strip stays wherever the user left it.

## Substantive future tests

Add one focused app test, with ordinary method cases plus one synthetic Tester
routing case. Do not repeat existing full suites.

Method cases:

1. Zero tabs: `selectAdjacentTab(true)` and `(false)` return false and do not
   call `Select`, focus anything, Flush, close a panel, or mark persistence.
2. One tab: both directions return false and are no-ops for the same reasons.
3. Two tabs: forward wraps from index 1 to 0; backward wraps from index 0 to
   1; ordinary middle movement works in both directions.
4. Three tabs: forward wraps 2 → 0 and backward wraps 0 → 2.
5. Failed Flush: switching returns false, leaves `Active` and the current tab
   pointer unchanged, preserves source text/draft/view, and reports the
   existing「切换标签」failure.
6. Successful switch: old export is invalidated, old presentation is released,
   Find cache is invalidated, GoToLine closes, new editor gains focus, and
   persistence becomes pending. Reuse existing counting seams where possible.

Synthetic Tester case:

1. With at least two tabs, `tester.Key(ui.Cmd, ui.KeyTab)` advances and
   `tester.Key(ui.Cmd|ui.Shift, ui.KeyTab)` returns to the prior tab. State
   that this verifies synthetic key routing only, not physical Windows OS
   behavior.

Do not add a reentry test for a hypothetical synchronous tab replacement. No
current production path can perform it.

## Help update

After implementation, update the operations table in `native/README.md` with a
single concise row, for example:

| 切换标签 | `Ctrl+Tab` / `Ctrl+Shift+Tab` 按可视顺序切换并循环 |

Do not describe an MRU switcher, popup, history, or physical-input validation.

## Implementation boundary

- `native/app/view.go`: two shortcut branches.
- `native/app/app.go`: one small adjacent-selection helper.
- One focused app test file or narrowly scoped test addition.
- `native/README.md`: one operations-table row.
- No changes to generic `Select`, Find, GoToLine, IME, tab-scroll state, or
  release documentation in this slice.

This handoff is a plan only. No product source, documentation, tests, builds,
or executions were changed or performed for this result.
