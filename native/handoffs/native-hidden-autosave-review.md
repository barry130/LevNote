# Hidden/minimized autosave review

2026-11-10, read-only source review. No product/runtime/test/helper file was
changed, and no test, build, host, worker, parser, physical-input, paint, or
resource process was run. This does not reopen or alter `.23`.

## WIP cutoff

```text
native/handoffs/native-image-hidden-readiness-review.md 4761 B  2026-11-10 13:20:40
native/handoffs/native-hidden-autosave-review-task.md    1707 B  2026-11-10 13:31:47

native/app/state.go                                      13553 B  2026-11-10 12:43:42
native/app/view.go                                       22988 B  2026-11-10 12:30:00
native/app/app.go                                        36131 B  2026-11-10 12:46:32
native/app/workspace.go                                   6673 B  2026-11-10 12:19:27

vendor/github.com/egoist/mygo/content.go                  9307 B  2026-11-08 11:42:29
vendor/github.com/egoist/mygo/window.go                  77294 B  2026-11-08 11:42:29
vendor/github.com/egoist/mygo/loop.go                     3304 B  2026-11-08 11:42:29
vendor/github.com/egoist/mygo/ui/runtime.go              55029 B  2026-11-08 11:42:29
vendor/github.com/egoist/mygo/ui/context.go               34487 B  2026-11-08 11:42:29
vendor/github.com/egoist/mygo/ui/window.go               30539 B  2026-11-08 11:42:29
vendor/github.com/egoist/mygo/internal/windows/surface.go 41844 B  2026-11-08 11:42:29
vendor/github.com/egoist/mygo/internal/windows/window.go 49263 B  2026-11-08 11:42:29
```

## Finding: source establishes a real minimized-autosave defect

The hidden-readiness result was not only a synthetic hidden-launch helper
limitation. The same application path also affects an ordinary user sequence:

1. Edit text, open/reload an image, switch a tab, or change another persisted
   preference. These paths set `persistPending` and update `lastEdit`
   (`app/app.go:397`, `app/app.go:450`, `app/app.go:462`,
   `app/binary_actions.go:197-231`, `app/view.go:344-374`).
2. Minimize (or otherwise lose the frame that would have serviced the debounce)
   before the next `View` execution after the 900 ms deadline.
3. The only automatic scheduler is `View -> persistWhenIdle`
   (`app/view.go:32`, `app/state.go:229-238`). Before the deadline,
   `Context.After` merely schedules a future frame; at/after the deadline, the
   next `View` performs `persistAutomatic`.
4. `persistAutomatic` runs one attempt and clears `persistPending` on failure
   (`app/state.go:241-254`). Therefore, if no frame runs, a pending draft or
   tab/preference change remains unsaved indefinitely until a later user-visible
   frame or a successful normal close. If the process is killed before close,
   that draft is not in `session.json`.

This is an established application defect: persistence is tied to Content frame
execution even though it is not inherently a drawing task.

The boundary is that source alone does not establish the exact real-world
frequency of paint delivery while minimized. MyGo explicitly marks a minimized
Surface as occluded (`internal/windows/surface.go:378-381`), and its comments
establish that hidden windows receive no `WM_PAINT`; the hidden-launch review
already established the no-paint behavior after the first frame. Whether a
minimized Surface sometimes receives a paint because of system-driven
invalidation is Windows runtime behavior that this read-only review did not and
could not establish. It does not change the product conclusion: the app has no
paint-independent guarantee for its own autosave deadline.

Opening files while minimized has the same source-level issue. CLI/drop/open
paths eventually set `persistPending`, but without a subsequent Content frame
there is no application-owned mechanism that executes `persistAutomatic`.
Restoring the window would normally produce a frame and then persist, but
requiring a visible window contradicts the recovery contract for a long-lived
minimized session.

## Why the existing public MyGo APIs do not solve it directly

- `Window.Update(fn)` posts `fn` to MyGo’s main FIFO, executes it independently
  of `WM_PAINT`, then calls `contentChanged` to request a frame
  (`content.go:181-206`). The callback itself is paint-independent.
- `Window.Invalidate`, `Context.After`, and `Context.Invalidate` all request a
  frame; they do not synchronously rebuild `Content`
  (`content.go:156-175`, `ui/context.go:494-505`, `ui/runtime.go:1192-1214`).
- `Context.After` arms a frame timer through `windowHost.afterTimer`
  (`ui/window.go:747-757`). A timer can wake the runtime, but the resulting
  `View` still depends on the Surface supplying `SurfaceFrame`.
- On Windows, `surface.RequestFrame` invalidates and posts `drawHidden` only
  before the first frame of an invisible window; after `FirstFrame` it depends
  on `WM_PAINT` (`surface.go:275-280`, `341-355`). The minimized window is
  occluded, but MyGo does not provide a public “build Content without paint”
  operation other than executing a callback through the main loop.

The useful public primitive is therefore the callback in `Window.Update`, not
the redraw half of that API.

## Smallest application-owned correction

Add an application-owned, paint-independent one-shot autosave timer that, when
the 900 ms deadline passes, dispatches through `Window.Update` and runs the
existing persistence logic. Do not call `View` directly and do not add polling.

Concretely:

1. Extend `App` with a small autosave timer state (for example one
   `*time.Timer` and a mutex-protected due time); no document/image data is
   stored.
2. Whenever a path currently sets `persistPending`, or through one existing
   helper where appropriate, arm/reset a timer for
   `lastEdit + 900ms`. Re-arm/reset only on later changes, preserving the
   current debounce.
3. When the timer fires, call `a.Window.Update(func() { ... })` if the window
   exists. In tests/headless mode, use the existing `uiDispatch`/fallback
   branch already in `App.update` rather than inventing a second dispatch.
4. The dispatched callback must:
   - return immediately when shutdown has started;
   - return while `restoring` is true;
   - return when `modalDepth > 0`, leaving `persistPending` true;
   - return before the 900 ms deadline if the timer fired early or a newer edit
     moved the deadline;
   - otherwise call the existing `persistAutomatic()`.
5. Leave the frame-dependent `persistWhenIdle` path in place. It can remain a
   fallback and will eventually coalesce with the timer path; remove it only if
   tests show the new timer makes it dead on every supported dispatch.
6. Stop/reset the timer during shutdown and keep it inactive after the app is
   closed. A late callback must not resurrect work.

This uses only the public `Window.Update` main-thread queue. The unavoidable
redraw request it makes is harmless; correctness comes from the callback, not
from a subsequent paint. It needs no vendor change, polling, extra cache, UI
automation, or new allocation architecture.

### Barrier semantics to preserve

- **Debounce:** one timer, reset on each edit/state change, executes at the same
  `lastEdit + 900ms` contract as `persistWhenIdle`.
- **Restore:** `persistTabsWithDiscard` already refuses while `restoring`
  (`state.go:141-144`). The dispatched timer should also re-check `restoring`
  before doing work.
- **Modal:** automatic persistence currently runs from `View`, including during
  ordinary panel state, but a native modal can block redraw and `CanClose`
  explicitly refuses with `modalDepth > 0` (`app.go:1046-1049`). A paint-free
  timer must not create surprise persistence from inside a modal. Re-check
  `modalDepth`; if nonzero, keep `persistPending` and rely on existing explicit
  lifecycle persistence or a later frame.
- **Shutdown:** `Shutdown` first sets `shuttingDown` and joins owned work
  (`workspace.go:56-76`). The timer callback must be shutdown-safe and stop the
  timer on app shutdown. `Window.Update` posts into MyGo’s main loop; MyGo
  drains/rejects queued callbacks when the application loop shuts down
  (`loop.go:17-31`, `64-78`), so a late post will not run after loop shutdown.
  The app-level `shuttingDown` check remains the application-owned guard.
- **Failure suppression:** call `persistAutomatic`, not `Persist`.
  `persistAutomatic` intentionally makes one attempt for a state, reports the
  failure through the existing `reportPersistFailure` path, and clears
  `persistPending` so unchanged failed state is not retried on every callback
  (`state.go:241-254`, `app_test.go:1587-1612`). A new document/session change
  re-arms it.
- **Draft Flush:** `persistTabsWithDiscard -> AddDocumentWithOptions` already
  captures the current document/draft state through the session builder
  (`state.go:180-198`). Running on the main/UI thread is important so editor and
  rich-view state is not read concurrently. A paint-free timer must not call
  `Flush` as a side effect of persistence; the existing session capture contract
  must remain unchanged unless a focused test finds a real need.

## Existing meaningful test seams

No physical input automation is required:

- `newTestApp` already supports a headless app and explicit `a.uiDispatch`;
  `App.update` has a public-window, injected-dispatch, and direct-call fallback
  (`app.go:278-294`).
- Existing pending-state tests cover edits, opens, tab selection, and mode
  changes (`app_test.go:562-593`).
- Existing automatic failure-suppression semantics are pinned by
  `TestAutomaticRecoveryFailureIsNotRetriedUntilStateChanges`
  (`app_test.go:1587-1612`).
- Existing binary pending/delivery and shutdown-join tests exercise main-thread
  delivery and cancellation without a physical window
  (`binary_test.go:327-400`).

The smallest focused tests would use injected dispatch/timer seams to prove:

1. edit marks pending and arms one timer;
2. subsequent edits reset rather than multiply timers;
3. the callback runs after the deadline and calls `persistAutomatic`;
4. an early callback leaves pending state and re-arms;
5. `restoring`, modal, and shutdown states suppress execution without clearing
   pending state (except the existing automatic-failure semantics);
6. a failed automatic save clears pending and is not retried until new state;
7. the new path produces the same session bytes for a text draft as the current
   `persistWhenIdle` path.

A separate runtime verification could minimize a real owned window and observe
`session.json` after an injected edit, but this source correction does not need
physical-input automation to establish its mechanism.

## Verdict

There is an actionable product defect: the 900 ms autosave has no application-
owned, paint-independent execution guarantee. The smallest correction is an
application-owned one-shot debounce timer whose main-thread callback uses the
existing `App.update` dispatch and existing `persistAutomatic` semantics. The
hidden `.23` host helper should not be changed by that correction and should
continue to prove close-side session publication rather than rely on pre-close
automatic persistence.
