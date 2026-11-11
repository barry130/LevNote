# Native image app integration checkpoint

2026-11-10, implementation in progress; no checks have run yet. App-only edits.

- Added exclusive binary tab authority, tiny eight-byte signature/suffix route
  ahead of text decoding, metadata-only inactive state and active held snapshot.
- Added single binary lane with one latest pending navigation request and an
  explicit reservation through UI outcome delivery. Task slots contain no live
  tab/view/owner; UI records own validity/callback checks. Copy uses accepted
  PublishBinaryCopy/AdoptCopy, reload transfers a new owner once.
- Added neutral BeginImage/DecodeImage wiring, native image/readonly-byte views,
  readonly controls/status, publication outcome reporting and shutdown joining.
- Session restore is a synchronous startup ownership transaction, each binary
  read running serially on the same lane and dropping inactive source payload.
  It preserves mixed record order and blocks persistence of partial restore.
- Audited main Doc dereferences in save/close/reload/view/reference/Find/template
  and added binary disk-token watching. Text failure/flush paths stay in place.

Concrete remaining concerns before final checks: compile complete integration;
exercise modal and shutdown result slots with real delayed work; verify no idle
byte-reacquire loop; check inactive tab payload/bitmap release and same-path copy
outcome; verify mixed restore active mapping after failed entries and failed rich
Flush transitions. Add focused tests and only then final combined affected suite
and vet when root confirms all composition sources frozen. At most two synthetic
760x480 captures, no product/host/worker/driver/resource run.

Two malformed patch applications made no changes; one file read requested a
nonexistent inspect.go (Inspect is in document.go). Both corrected in place.

## Implementation ready for final integrated verification

After root's concrete review, notes1–7 are implemented and tested. CanClose
does not block on IO: it cancels/refuses while pending, delivers copy truth while
the window exists, and requires another close after the outcome. A timed-out
open marks its slot abandoned, drops payload immediately and reclaims the small
UI record/reservation on the next frame/command. ChooseOpen's actual dialog
scope ends before ordered file opening. Same-tab mode/plugin changes retire
presentation but retain the one active snapshot; true deactivation/close drops
it. Byte preferences remain readonly, unwrapped, without text line numbers.

Duplicate paths are rechecked at UI adoption using the existing Select/Flush
barrier. Clean legacy image session records enter the bounded binary branch
before any text read; dirty text drafts with image suffix remain text/source.
Nested SaveAs dialogs are refused before capturing another snapshot. Nil image
callback owners reject. Failed image decode reason remains through byte fallback.

Focused1:8 tests PASS1.642s. Focused2:12 tests+2subtests PASS2.320s.
Focused3:15 lifecycle tests+2subtests PASS2.420s, one explicit capture test SKIP;
exit0 with corrected explicit nested-PowerShell exit propagation. No prerequisite
standalone suite or product run. Focused repeats followed concrete source fixes.
Independent review found no additional blocker beyond root notes.

Two original synthetic760x480 captures are preserved: image light and disabled
plugin byte fallback dark. Capture1 failed a test-only pending-job assertion
before producing files; capture2 passed after allowing already-delivered tiny
decode. The first runner's missing explicit LASTEXITCODE propagation masked its
failure in outer-shell metadata; stdout FAIL is authoritative and preserved in
captures1-correction.md. Runner now propagates correctly. Root viewed captures
and assigned the view owner's clipped toolbar control a narrow correction.

App source is stable. Awaiting that view freeze, then root-authorized one
replacement image capture and one final combined app/modules/.../plugins/session/
document/raster/editor test+vet. No package/host/resource run.

Truthful resource accounting: steady state one active accepted source+bitmap;
single codec task may linger after cancellation. Explicit open/reload can retain
old accepted source+bitmap plus one new source until safe UI adoption. The lane
awaiting gate blocks any next full task before the result is consumed/dropped.
A modal SaveAs captures its one immutable source and can transiently overlap
another tab's in-flight source; nested SaveAs cannot accumulate source captures.
Pending new-tab requests carry metadata only; same-active-snapshot aliases are
approved and add no bytes. No RSS/VRAM or physical-driver bound is claimed.

## Final verification checkpoint

The approved close-modal guard is saved: CanClose refuses before any result
adoption while modalDepth is positive. The existing lifecycle test additionally
holds a committed copy result through deliverBinary and CanClose, checks that
the ready slot and old path remain, then verifies adoption only after dialog
unwind. No separate focused rerun was added.

The view owner's narrow toolbar result/root acceptance is frozen. Its exact
ui.go2B168A93... and imageview_test.goE4FE9CA6... identities were verified before
integration. capture-fixed1 passed1.506s/exit0; the separately named
image-light-760x480-toolbar-fixed.png was inspected and all toolbar controls are
visible inside the532-DIP pane. Both original image files remain unchanged.

final-suite1 ran the full affected app/modules/.../plugins/session/document/
raster/editor suite. App passed19.490s; every other package passed except
modules/imageview, where TestManualResizePreservesCenterThroughTester asserted
exact floating-point zoom1 and received0.9999999999999999 after the toolbar
geometry change. Actual exit1/output are preserved. Root was notified for a
view-owned narrow correction; no imageview source was edited by the app owner.
Vet awaits that corrected source. Fifteen app files are frozen in
native-image-app-evidence-20261110/frozen-app-source-identities.json; there was no
before-implementation identity snapshot, and none is claimed.

Final completion: root reviewed the view owner's test-only tolerance correction
(imageview_test.goCD2506B1..., runtime ui.go unchanged). Per root's narrow scope,
only imageview was retested: imageview-final1 PASS1.203s/exit0. Combined final-vet1
exit0, empty preserved output. Already-passing app/other packages and captures
were not rerun. All15 app identities still match. Source, verification, limits
and preserved failures are now frozen in native-image-app-implementation-result.md.
