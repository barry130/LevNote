# Active tab reveal result

Completed 2026-11-10 with the [combined keyboard stage](tab-keyboard-implementation-result.md).
Source/tests frozen; .21 runnable package is unchanged.

`app/app.go:94` holds only `revealedTabID int`. `app/view.go:274` uses a
frame-local button with public `ScrollIntoView()` and default minimal instant
scrolling. It resolves the active identity after click handling and requests
reveal once when that identity changes. Ordinary frames, same-tab focus and
closing an earlier inactive tab do not undo a user's manual horizontal scroll.
Empty tabs clear the marker; add/open/restore and active close select the new
identity without retaining documents or widgets through the marker.

The real close-button route formerly called CloseTab inside the live slice
range; CloseTab compacts and nil-clears its backing tail. Close now executes
after all tab buttons are built and invalidates the next frame. The focused
early active/inactive close tests pass without stale tail access, retain
correct neighbors and verify the removed tab is absent from the backing slice.

Five focused reveal tests passed in
[09-reveal-focused.log](../artifacts/tab-workflow-20261110/09-reveal-focused.log),
with the expanded inactive-close identity case passing in 11 and the final
full app suite. Coverage: 14 tabs in 760x480, first/last keyboard wrap,
manual scroll after reveal and later redraws, same-tab selection, add/open/
restore, both click directions, actual early close buttons, last close,
empty/reopen, and failed-Flush keyboard/click leaving scroll unchanged.

One opt-in synthetic capture was produced and visually inspected:
[tab-active-wrap-760x480.png](../artifacts/tab-workflow-20261110/tab-active-wrap-760x480.png).
The tab-13 button is fully visible after first-to-last wrap. This is Tester
geometry only; physical Windows UI/IME and resource use were not measured.
PNG 26620 bytes, SHA256
`774F3C582CA95C49D879A42FF3ADF0C96C237399099C523C23403B110E72CFB5`.

The original combined app test/vet (13/14 logs) and the final app test after
root's GoToLine focus correction (17.278s, exit 0) plus vet (exit 0), final
identities and retained logs are recorded in the keyboard result. Reveal source
and capture are unchanged by that correction. No package/build/host launch or
README edits.
