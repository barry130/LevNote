# Paint-independent recovery scheduling

ACTIVE2026-11-10 after .23 package/root acceptance. Astra owns
this next safety increment. No child delegation. Read the source-grounded
native-hidden-autosave-review.md and the actual existing implementation; its
proposed mechanics are recommendations, not permission to introduce races.

Problem: recovery persistence is started only from View. A pending document or
tab/preference change has no application-owned execution guarantee when a hidden
or minimized window stops producing frames. Normal close already persists.
Do not change accepted .23 helpers/package. Fix application scheduling through
public MyGo main-thread dispatch, without calling View or changing vendor.

Own app/autosave.go and substantive autosave tests, and the minimum app runtime
edits needed to replace direct persistPending=true sites with one scheduling
helper. App state/lifecycle fields, modal-exit, Restore and Shutdown hooks are
in scope. Keep session/document serializers, format views, raster and package
scripts frozen. Do not edit app/binary_keyboard_test.go (separate view owner).

Use one cancellable one-shot debounce scheduler. Preserve the existing lastEdit
deadline semantics: source edits wait900ms after the latest edit; tab/preferences
use the remaining existing delay and do not silently reset editing history.
Every real pending-state transition must schedule even if there is no future
View. Duplicate requests coalesce, late callbacks validate the latest request.
The background timer must not read/mutate tabs, lastEdit, modalDepth, restoring,
Window or other UI state. It only posts through a captured safe UI dispatcher;
the callback performs state checks on UI. App.update's no-Window/no-uiDispatch
fallback runs synchronously, so never let a real timer use that fallback to
mutate headless App from a background goroutine. Inject clock/timer/dispatch
only as needed for deterministic tests, not a general job framework.

On callback recheck shutdown, request identity, pending/deadline and modal/restore
barriers. A modal exit must re-arm a deferred request independently of another
paint; merely leaving pending=true and hoping for View repeats the bug. Preserve
Restore's full-transaction suppression and final pending decision. Explicit
save/close persistence supersedes obsolete timer requests. Shutdown cancels the
timer before joining owned work; queued callbacks cannot resurrect persistence.
Keep one-attempt failure suppression via persistAutomatic, no retry loop and no
automatic Flush or commit of live composition/rich-local drafts. Session still
captures committed document buffers; uncommitted format-local drafts remain the
existing documented limitation. No polling, per-frame allocations or cache.

Verify behavior with a few substantive tests using actual session bytes and a
queued dispatcher: dirty committed text persists without View; later edits move
deadline and stale callbacks do nothing; modal exit resumes without View; restore
and shutdown do not publish partial/late state; write failure is not repeated
until a new mutation. Exercise at least one non-edit mutation and binary metadata
without copying payload into session. Reuse the existing session/dispatch seams.

Preserve failures, exact command/exit logs and final source identities under
native/handoffs/native-autosave-evidence-20261110. Coordinate final tests with
root and the image-keyboard owner: implementation may proceed in disjoint files,
but the combined affected app/imageview suite and vet run once after both final
source sets are frozen. Focused new autosave tests during debugging are allowed.
No package/host/worker/resource/capture runs in this source task. Write checkpoint
and native-autosave-result.md; no product documentation ownership here.
