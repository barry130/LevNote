# Open-document picker final source review

2026-11-10. Read-only review of the frozen picker source and the saved focused
evidence. This reviewer ran no test, build, product instance, benchmark, or
physical-input check and edited only this handoff. The original WIP review,
`open-document-picker-independent-review.md`, remains unchanged.

## Verdict

**One concrete blocker remains.** During a native query composition,
`openDocumentsKey` consumes Tab/Shift+Tab, Enter, Up, Down and Escape, but leaves
Left and Right for MyGo's editor. The public control route then commits the
composition before applying the caret move. This contradicts the implementation
task's requirement that the IME own Enter/arrows/Escape and the source comment's
own rule that a raw navigation key must not submit text or move the widget caret.

Final review ownership is therefore withheld pending a focused correction and
evidence run. The rest of the final delta matches the controlling decisions.

## Frozen source

The actual five files match `source-frozen.json` exactly:

| Native-relative source | Bytes | SHA-256 |
|---|---:|---|
| `app/open_documents.go` | 14010 | `3E0E332285E6B3B4C0A4B23FC56B8CF298A7366333C374DE022B6520DC079390` |
| `app/open_documents_test.go` | 25662 | `D6B88B91A7E7A79A72C49AC2CA6D804E5313FF4FE4A58325660E2782E327DFB2` |
| `app/app.go` | 39003 | `FAD0EDE4C480E0C8F5C56F4E81CF2499BEF0007BA6EAE877FC516623BFFEA050` |
| `app/view.go` | 23425 | `64DCDC194510C6426112A7B6AAFE57A84DDE68E630A20D53C994D77CC34102C0` |
| `app/workspace.go` | 6739 | `7A9ACA4F8E807D7A0563B5595976839AA7DED706EFDC5171B259079D68A16DA4` |

`freeze-verification.json` records that only `open_documents.go` changed after
07, solely for the selected-secondary color correction. Reversing that exact
block reconstructs the 07 runtime hash
`7CB014CDF3365AE0AD62CAF5F2D11E5EC38824B48E1D5C8D2C760C1B77930339`,
and the other four files remain identical.

## Blocking route

At `app/open_documents.go:305-312`, the composing branch returns false for
Left/Right. The exact public MyGo route is:

1. `ui/input.go:976-983` offers a non-text key to the focused editor's
   `OnTextKey`. It does so when the editor wants the key, which includes
   Left/Right at `ui/editor_navigation.go:43-47`.
2. Because the picker callback returned false, `ui/input.go:987-997` queues an
   `editKey` for the focused text input.
3. `ui/editor.go:223-226` processes `editKey` by calling `commitCompose()` before
   applying the key. `commitCompose` clears the native composition/caret state
   (`ui/editor.go:199-209`), commits the preedit into the editor buffer, and then
   the Left/Right handler moves the caret.

Thus public Tester input such as `Compose("B", 1)` followed by
`Key(0, ui.KeyLeft)` or `Key(0, ui.KeyRight)` reaches a reachable early-commit
route without private-toolkit access. The app's `composing` flag is updated by
external `InputCompose`/`InputText` observation, while this internal
`commitCompose` calls only the text-input client. Consequently the app can retain
a stale composing guard after the native composition has already been committed.
This is source/API reasoning, not a reviewer-run reproduction, and it does not
claim a physical Windows IME result.

The existing real composition regression covers Esc, Tab, Shift+Tab, Down and
Enter, but not Left/Right. Add a direct public-Tester regression that composes
text and sends both Left and Right while asserting: picker/focus stay intact;
native `Composing()` and the native caret remain unchanged; the query/bound value
stays uncommitted; the app still recognizes the active composition; active tab,
documents, history and bytes stay unchanged. The smallest source correction is
to consume the intended IME-owned arrows in the composing branch, without
changing the ordinary non-composing Left/Right behavior.

## Accepted final checks

The earlier root decisions resolve the WIP review's other findings:

- The up-to-100 retained title/path prefix scan per visible frame is an accepted
  bounded CPU tradeoff, not a zero-work claim. Unchanged metadata/folded strings
  are reused; this review makes no whole-app CPU or memory claim.
- Same-target remove/replace/move during Flush now has direct regressions and
  visible refusal assertions. The unrelated-target reentry cases remain valid
  coverage for that different identity path.
- Picker rows retain only stable ID, bounded sanitized title/path/fold metadata,
  flags and list fields. They retain no `Tab`, document, buffer, view, content
  snapshot or worker.
- Exactly two app-owned public opaque handles survive list resets and
  close/reopen. Other metadata, callbacks, selection/actions/messages and list
  internals are cleared; close/Shutdown cancels query focus and advances the
  generation.
- Pointer and submitted actions are deferred until after list construction.
  Activation resolves by stable ID, captures origin/target/index, and validates
  generation, origin, exact target pointer/index and ID after Flush. The same-tab
  path explicitly flushes/checks rather than using ordinary Select.
- The real Markdown refusal paths preserve the visible field and draft rather
  than focusing hidden source. The real rich-growth opening refusal and the
  injected post-open Flush refusal cover the two different draft restoration
  seams; the latter does not claim a naturally occurring rich mutation.
- Query Tab/Shift+Tab consumes and focus propagation corrections are present.
  This does not cover the Left/Right blocker above.
- Selected row path/badges use `Theme.AccentText`; unselected secondary text
  remains `Theme.TextMuted`.

## Focused evidence

`07-focused-union` ran from
`2026-11-10T12:11:43.1813438Z` through `12:11:49.8790258Z`. Go exit was `0`
(`app` package `3.318s`); the wrapper process finished at
`12:11:50.1195217Z` with shell exit `0`; stderr is empty. All 12 behavior
top-level tests passed, including the real draft/focus, same-target mutation and
two-handle tests. The optional capture skipped as intended. Its pre/post
manifests match all five source files.

`08-contrast-capture` ran from `2026-11-10T12:13:19.1585794Z` through
`12:13:23.7816726Z`. Go exit was `0` (`app` package `1.655s`); the wrapper
process finished at `12:13:24.0184254Z` with shell exit `0`; stderr is empty.
The original capture remains SHA-256
`78F6F56AEBB088FC69A1F0AD5B3399B5D0514C63CC492D2C0D40DB0F8784CB38`;
the replacement remains
`9C544622D078AEFD50AB295D3FC646B2E46AAB3315A07E2144E7A02712088DE0`.
Inspection shows the selected path/badge legible against the selected row and
normal secondary text still muted. No 08 pre-run manifest is claimed; the
post-07 reconstruction in `freeze-verification.json` supplies the source delta
identity.

These focused tests establish synthetic public-control routes. They do not
establish physical Windows IME delivery, natural rich mutation, RSS/heap use,
or whole-app memory behavior. The later full affected app test and vet remain
unrun by this reviewer and should wait for corrected frozen source.
