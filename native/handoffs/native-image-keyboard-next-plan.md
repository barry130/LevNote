# Focused keyboard controls for the native image reader

2026-11-10, read-only recommendation after app source acceptance. Implement only
after root activates a post-.23 slice. This task changes this handoff only; no
source, tests, product docs, captures, builds or launches were changed/run.
No live competitor research is claimed.

## Proposed interaction

Keep keys local to the focused `图片查看区域` canvas. Do not add app shortcuts,
ancestor input observation, global key polling, a host contract or a focus flag.

| Exact key/modifiers | Action |
| --- | --- |
| `KeyEqual`, modifiers0 or exactly Shift | Zoom in by1.25 (`=` and US `+`) |
| `KeyMinus`, modifiers0 | Zoom out by1/1.25 |
| `CodeNumpadAdd` / `CodeNumpadSubtract`, modifiers0 | Same zoom actions |
| `Key0`, modifiers0 | Fit entire image |
| `Key1`, modifiers0 |100% actual-size view |
| Left/Right/Up/Down, modifiers0 | Move reading position32 DIPs in that direction |

Left/Up add32 to image panX/panY; Right/Down subtract32, matching existing
graph-preview reading direction. No Shift-fast-pan, F alias, Ctrl/Cmd zoom,
Home/End, PageUp/Down, Escape or additional commands in this increment. All
Ctrl/Cmd/Alt/Super chords and Shift chords except Shift+Equal remain untouched.
Normalize each Key/Code to at most one action so keypad minus is not applied
twice when both public fields name it. Do not infer printable commands from
PhysicalKey or InputText; literal symbols on untested layouts are not guaranteed.

Handle only InputKeyDown, using existing nil-bitmap/released guard. During an
active pointer drag leave key actions alone until release, avoiding a stale
drag-origin jump. Accept OS repeat as one bounded action per delivered keydown;
no held-key timer. A recognized action returns true even when clamped/no-op,
so arrows cannot fall through to an outer scroll container. Other events/keys
return false; InputText/InputCompose/InputCommand and key-up stay untouched.

## Focus and public API evidence

- `modules/imageview/view.go:69` already builds a Focusable canvas, binds
  `v.canvas` and installs `HandleInput(v.handleInput)`. Focusable participates
  in Tab/ShiftTab and pointer focus (`ui/element.go:1938`). Click or Tab into
  the canvas is the entry rule; keyboard control does not depend on hover.
- `ui/handler.go:133` documents focused-only key delivery. Its `focusHandler`
  at293 requires the exact focused element plus window keyboard focus; events
  are not bubbled from text-input children. `ui/input.go:954` gives declared
  app/ancestor shortcuts priority. The existing handler needs no retained
  Context or transient Element to recognize focus.
- App shortcuts (`app/view.go:537`) keep Ctrl/Cmd+N/O/S/ShiftS/W/Tab/ShiftTab/F/G
  and Escape. A focused sidebar/settings/search field receives its own typed
  text/composition. Plugin panel replaces document drawing. Native file/message
  dialogs own keyboard focus, so parent canvas keys are not delivered; test
  window blur via public Tester.SetFocused(false), without claiming OS proof.
- After an enabled fit/100%/decrease/increase toolbar action, request
  `v.canvas.Focus()` from that action's Clicked branch. This explicitly returns
  both pointer and Enter/Space activation to the canvas for follow-up keys.
  Tab/ShiftTab still leave normally. No autofocus on draw, decode completion,
  tab selection, window reactivation or plugin re-enable; returning users may
  click/Tab into the canvas. No focus request after the byte action, and no
  changes to app SaveAs/reload focus behavior. Do not use KeepFocus, which would
  retain whichever unrelated input previously had focus.
- `ui/ref.go:60` exposes deferred Handle.Focus; CancelFocus at98 cancels a
  pending request rather than blurring current focus. Existing Release calls
  CancelFocus, clears pixels and marks released. `app/view.go:673` and
  `app/app.go:993` release the old image presentation on byte mode/deactivation;
  the omitted canvas cannot take later keys. Draw after Release is intentional
  reactivation in current code, so a release test must also stop drawing it.
- Windows `internal/windows/surface.go:1152` maps main/keypad minus and main
  equals, but not keypad add. Public `InputEvent.Code`/CodeNumpadAdd handles that
  via `ui/handler.go:204` raw focused delivery. Tester.CodeDown/CodeUp exercise
  this path. No introspection, vendor patch or backend key-table change needed.

All vendor references above are under `native/vendor/github.com/egoist/mygo/`;
other paths are relative to native/. These are source-read findings, not new
runtime observations. No public API blocker was found for the proposed keys.

## Geometry and lifecycle

Reuse `zoomAroundCenter(1.25 or1/1.25)`, `zoomToCenter(1)` and `fitMode()` from
imageview/ui.go. Keyboard anchor is the visible viewport center; pointer wheel
and pinch keep `zoomAt(ev.X,ev.Y,...)`. Anchor preservation is subject to the
existing edge clamp. Never use keyboard event X/Y (the last pointer position).

For arrows call existing panBy with canvas dimensions, image dimensions and
the32-DIP delta. Assign its returned viewport only when valid. Set fitMode=false
only if pan actually changes; a small centered image or fitted image remains
in fit mode on a clamped no-op. No alternative pan arithmetic or geometry edits.
Manual zoom remains2%..800%; fit may be below2% for tall images and never exceeds
100%. Existing zoomAround prevents zoom-out below a tiny fit from snapping up
to2%; first zoom-in enters the manual range. Preserve these semantics exactly.
Controls only change viewport state: no decode request, bitmap/source/cache
allocation, document mutation, persistence, cancellation or publication change.

## Small ownership and verification proposal

Runtime edits: `modules/imageview/input.go` (key branch) and `ui.go` (focus return,
small tooltips). Keep view.go, geometry.go, app runtime and all contracts frozen.
Tests: `modules/imageview/imageview_test.go`, plus one new
`app/binary_keyboard_test.go` for actual host mode/tab routing only. Same owner
may own these exact files once activated; no composition/core/vendor ownership.
Prefer tooltips on existing controls (`适合窗口：0`, `100%：1`, `增大：+ / =`,
`减小：-`) and the canvas (`图片区域获焦后：方向键移动`). Keep labels and layout
unchanged; no extra hint row in the already narrow toolbar.

Use a few public Tester scenarios, not duplicate geometry helper unit tests:

1. Focus containment: completed image beside a real TextInput; Tab/click canvas,
   then Key/TypeKey plus/minus/0/1 and keypad CodeDown/CodeUp. Assert one action
   each. Tab to field, type/Compose, send navigation/modifier keys and confirm
   its normal text/preedit behavior and unchanged image viewport. Window blur
   blocks canvas actions. Toolbar Enter/Space returns focus to canvas; Tab exits.
2. Geometry through events: large image at100%; repeated +/- and arrows reach
   manual limits and all pan edges, retaining finite bounded state. Check the
   center image coordinate before/after a zoom away from clamps. Fit/100% and
   one tall sub-floor fit fixture prove fit mode and zoom-out no-op. Use existing
  1e-6 tolerance for computed zoom/pan values, not exact floating equality.
3. Release/byte boundary: fixture callback releases and stops drawing the view;
   subsequent keys cannot change the old viewport, request pixels or create a
   bitmap. Pending focus is canceled. In the app case use the real byte button,
   readonly byte editor and CtrlTab into text: image keys no longer act, plain
   text input still works, tab navigation/authority/snapshot lifecycle holds.
   Preserve bytes/version/token; do not forbid normal readonly caret movement.

After implementation/corrections, run affected imageview/app test suites and vet
once via scripts/go.ps1, preserving actual exits and any failures. No screenshots,
benchmark, package/host/worker/driver run or resource claim is needed for this
slice. Physical IME and hardware layouts remain separate verification scope.
