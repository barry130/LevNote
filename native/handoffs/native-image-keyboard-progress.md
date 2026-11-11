# Native image keyboard progress

2026-11-10. The `.23` package/evidence remains frozen. This increment owns only
`native/modules/imageview/input.go`, `native/modules/imageview/ui.go`,
`native/modules/imageview/imageview_test.go`, and the new
`native/app/binary_keyboard_test.go`.

Implemented and reconciled against:

- `native/AGENTS.md`
- `native/handoffs/native-image-keyboard-implementation-task.md`
- `native/handoffs/native-image-keyboard-next-plan.md`
- `native/handoffs/native-image-keyboard-review-notes.md`

The focused image canvas now handles the reviewed equals/plus, minus, keypad,
fit, actual-size and arrow keys with exact modifiers and one normalized action
per event. Typing, IME, commands, modified shortcuts, key-up and pointer-drag
keys remain isolated. Keyboard zoom is centered on the viewport; clamped pan is
a fit-preserving no-op. Enabled fit/100%/decrease/increase toolbar actions
return focus to the canvas. Tooltips expose the key guidance without adding a
toolbar row.

Root review corrections are applied:

1. The app test verifies keyboard zoom through rendered image geometry at
   `centerX+22`, replacing the repeated unavailable `HasText("125%")` assertion.
2. The containment test compares the complete `viewport` value before and after
   text/IME/command input in the adjacent field.
3. The public sub-floor fixture now uses the admitted `1x8192` side limit and a
   short valid Tester viewport; no raster limit was widened.
4. The evidence directory records every invoked Go command, exit, and failed
   attempt, including the repeated app failure history.

The corrected focused imageview test passed in `14.186s`/exit0. The corrected
focused app boundary test passed in `2.409s`/exit0. Logs and a command manifest
are under `native/handoffs/native-image-keyboard-evidence-20261110/`.

The later combined suite exposed the zoom-equal `100%` mode defect. `zoomToCenter`
now clears fit mode on an explicit equal-geometry target, while relative
zoom/pan no-op behavior is unchanged. A public button plus `1`-key resize
regression passed with the previously failing toolbar test in `1.204s`/exit0.
The final regression uses an admitted `400x200` image, proves the resized 100%
view is cropped/clamped at zoom1 in both activation modes, then proves public
`0` restores centered fit below100%.

Source is frozen again for root review. The combined app/imageview suite and
vet remain held for Astra's sole coordinated run. No build, host, package,
capture, worker, or resource run was performed.
