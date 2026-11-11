# Native Find keyboard flow result

2026-11-09. Implementation and scoped verification are complete. Production
source and tests are frozen for root and independent review. No package build,
worker execution, launch, physical OS IME check, or product-doc edit was made.
The accepted `.13` package, screenshots, and evidence remain unchanged.

## Delivered behavior

- Ctrl+F and the toolbar Find action open Find and use a generation-scoped,
  one-shot query focus request. Repeated invocation focuses the existing query
  without clearing query, replacement, or options, and no later frame steals
  focus back.
- The query field handles Enter as next and Shift+Enter as previous. It reads
  the current `TextKey.Text()` before taking the key and restores query focus
  after an accepted action, including repeated navigation and cache rebuilds.
- The replacement field handles Enter as replace-current through the existing
  fresh match/hash validation. Shift+Enter does nothing, readonly documents stay
  unchanged, and after replacement the field retains focus for further actions.
- Normal Escape closes Find and restores focus to the current live source editor
  in source/split presentation. Preview stays in preview and does not receive
  invisible source focus. Escape from a focused Find action also works.
- Find query/replacement composition consumes raw Enter/Escape before the
  single-line control can commit or submit marked text. Buttons retain composition
  focus; toolbar toggle/close and pointer actions neither navigate, replace, nor
  close while the relevant Find field is composing.
- Source/rich composition blocks opening Find. Ctrl+G while either Find field is
  composing is refused with a visible reason and does not move focus; Ctrl+G works
  after the composition ends.
- Focus requests capture the exact tab, document, editor, and buffer and are
  revalidated after the action and after controls bind. Successful source reveal
  cancels its own pending source focus through the authorized neutral
  `Editor.CancelFocus()` before the one-shot Find focus is applied.
- Find callbacks reject stale generation, tab, document, editor, or buffer
  identities. Closing Find cancels both field handles and its pending focus
  target. Replacement captures identity before Flush and rechecks it before any
  mutation; failed Flush retains source/history/selection and field focus, while
  an identity-changing Flush leaves no stale focus or navigation request.

The global Escape shortcut is registered only when focus is outside the visible
Find panel and GoToLine is absent. Its prior preemption of the field/action path
was the initial Enter/Escape failure. The source-navigation owner has the
corresponding narrow guard against Ctrl+G while Find is composing.

## Post-review lifecycle correction

The independent review found that the toolkit can release a composing Find
field, commit its value, and emit no `InputText` observer event when focus moves
to the source, another tab, or another focused control. The App could therefore
retain `FindQueryComposing` or `FindReplacementComposing` after the field had
visibly ended composition, blocking Ctrl+F, Enter, Escape, and actions.

The correction keeps document identity as the guard for source actions while
reconciling each Find field's transient composition state from public
`Element.Focused()` and `Element.Composing()` results. A field flag clears only
after that field has lost focus and the toolkit no longer marks text. Hiding Find
under `ShowPlugins` clears the same transient flags, cancels both field focus
handles, and drops the exact pending tab/document/editor/buffer focus request.
`Query`, `Replacement`, case sensitivity, and regex state are unchanged, and
source text, selection, version, history, and the Find cache are not mutated by
cleanup. Invalid pending focus requests expire without changing active
composition.

The actual corrected red is
`find-keyboard-evidence/focus-leave-lifecycle-red.log` (SHA256
`550592E5F9145AA471D78DB3F20E705B8E16CE4A4E29C58581386DC10962A601`).
It reproduced stale query composition after source focus, stale replacement
composition after source focus, stale query composition after tab switch, and
stale query composition hidden by Plugins. Two pre-capture tab-label fixture
attempts are preserved separately as
`focus-leave-lifecycle-red-fixture-attempt.log` and
`focus-leave-lifecycle-red-fixture-attempt-2.log`; their replacement text made
the tab label dirty, so they are fixture evidence rather than product results.

After correction, the focused lifecycle run
`find-keyboard-evidence/focus-leave-lifecycle-focused-green.log` (SHA256
`D09D1C6B7515918A3386F2421073F9DF4290E2500303226A77E5CC732B4C8C6B`)
passed query and replacement composition-to-source, query and replacement
composition-to-tab, query and replacement through Plugins hide/show, and hidden
cancellation of a staged pending query-focus request. These cases assert one
committed field value, preserved field/options state, unchanged source
text/selection/version/history, immediate Ctrl+F/Enter/Escape use, no delayed
query/replacement focus, and no stale pending focus identity.

## Files frozen

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

## Verification and evidence

The actual initial red is preserved at
`find-keyboard-evidence/initial-keyboard-red.log`: Ctrl+F did not focus query,
Enter/Shift+Enter did not navigate, and composition was not protected.
`narrow-keyboard-pass.log` is also a red run despite its historical filename.
The duplicate-commit composition red and fixture-only failures are preserved at
`broad-regressions-focused-2.log` and `broad-regressions-focused-3.log`.
The fixture corrections distinguish committed `AccessInfo.Value` from marked
text, capture the readonly selection selected by `find(true)`, and select the
exact fresh replacement match. `focused-4.log` and `focused-5.log` preserve
runner plumbing mistakes; they are not product-result evidence.

Final focused command:

```text
$ .\native\scripts\go.ps1 -GoArguments @("test","./app","-run","^(TestFindKeyboard|TestFind|TestSourceNavigation|TestFindbar)","-count=1","-v")
...
PASS
ok levnote.local/native/app 2.776s
EXIT_CODE=0
```

This covered all keyboard cases, Find matching/cache/Flush behavior, the actual
Ctrl+G shortcut path, and minimum-window Findbar reflow. Two capture-only tests
were skipped because their screenshot-directory environment was intentionally
not set; visual markup was not changed in this task.

After the final stale-buffer and failed/identity-changing Flush keyboard-focus
additions, the focused regression run at
`find-keyboard-evidence/new-lifecycle-focus-covered.log` passed all three named
cases with exit 0.

The one final affected suite after all source/test corrections:

```text
$ .\native\scripts\go.ps1 -GoArguments @("test","./app","./ui/editor","-count=1")
ok  levnote.local/native/app       15.805s
ok  levnote.local/native/ui/editor  2.065s
EXIT_CODE=0
```

The one final vet after that suite:

```text
$ .\native\scripts\go.ps1 -GoArguments @("vet","./app","./ui/editor")
EXIT_CODE=0
```

The vet command produced no warning output. Exact raw logs are
`find-keyboard-evidence/final-app-ui-editor-test-2.log` (SHA256
`6BF9A32F9C724525DE82354B24D05995D9343322543F5A811CF496500F4529D5`)
and `find-keyboard-evidence/final-app-ui-editor-vet-2.log` (SHA256
`55136E372CC06096357CBA87F588CCC89781D15DC38B9BC7BEEF9078BF7A4185`).

After the post-review correction, the one final affected suite was:

```text
$ .\native\scripts\go.ps1 -GoArguments @("test","./app","./ui/editor","-count=1")
ok  levnote.local/native/app       14.326s
ok  levnote.local/native/ui/editor  2.083s
EXIT_CODE=0
```

The one final vet after that suite was:

```text
$ .\native\scripts\go.ps1 -GoArguments @("vet","./app","./ui/editor")
EXIT_CODE=0
```

The vet produced no warning output. Exact raw logs are
`find-keyboard-evidence/final-app-ui-editor-test-3.log` (SHA256
`49F7F745B49E4D8551C48C74FB8F696EFBA6D1306BE99B59F8E1D29EBDE86B1B`)
and `find-keyboard-evidence/final-app-ui-editor-vet-3.log` (SHA256
`55136E372CC06096357CBA87F588CCC89781D15DC38B9BC7BEEF9078BF7A4185`).

## Scope and limits

This result proves MyGo Tester synthetic composition behavior. It does not claim
physical OS IME candidate or Escape behavior, because no UI launch was allowed or
performed. The tests verify that Enter/Escape leave the committed accessibility
value unchanged while composition remains active, and that one explicit Tester
commit changes the bound value exactly once. The toolkit's
`AccessInfo.Value` omits marked text by design, so it is not used as a preedit-text
oracle.

Existing matching, search cache, replacement transaction, Flush, reflow, and
document/view lifecycle ownership are preserved. No performance or memory claim
is made.
