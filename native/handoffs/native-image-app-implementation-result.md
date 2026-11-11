# Native image app integration result

FROZEN 2026-11-10 for root acceptance. App source and final verification are
complete, including the view owner's test-only floating-point correction.
No product package, native host, external worker, driver or resource run occurred.
The accepted .22 executable and evidence remain unchanged.

## Delivered behavior

- PNG/JPEG suffixes and an eight-byte regular-file signature hint route before
  text decoding. Unknown suffixes with an image signature receive a visible
  explanation. A stable immutable binary snapshot supplies decode authority;
  no mutable BinaryFile is used by a worker after UI ownership transfers.
- Tabs have exclusive text or binary authority. Readonly byte inspection is a
  small presentation buffer over at most64KiB, never an editable document or
  save/session authority. Oversized candidates retain bounded inspection and
  explicitly cannot use the16MiB copy path. Disabled/unsupported/corrupt/mismatched
  images retain byte access and visible reasons instead of text fallback.
- BeginImage calls neutral modules.DecodeImage(ctx, snapshot.ReaderAt(),
  snapshot.Size()). Publication checks current/open tab, owner/token/generation,
  view/adapter/mode and plugin enablement. Only the imageview owns its Bitmap.
- OpenMany stays sequential and rechecks duplicates again at result adoption,
  including the current text Flush barrier. A failed Flush leaves the previous
  tab/view and drops unused new results. Binary commands/status are capability
  aware: readonly image/bytes, copy/reload, format/dimensions/size; no text find,
  replace, GoToLine, outline, split, export or image-template creation.
- Copy captures immutable authority across the save dialog, publishes exclusively
  through PublishBinaryCopy, then uses UI AdoptCopy and its rebound snapshot.
  Held accepted bytes remain copyable after source deletion. Existing destination
  refuses; same-path verifies without rewriting. Completion remains truthful if
  the initiating tab changes/closes, including committed-but-unverified outcomes.
- Reload creates a replacement owner on the worker and transfers it only after
  exact initiating authority validation. Failure preserves old source/pixels.
  Token-based disk observations mark missing/changed state without replacing the
  accepted bytes. Same-tab mode/plugin changes retire presentation but preserve
  the one held source; actual tab deactivation/close drops source and bitmap.
- Session restore uses AddBinaryTab/RestoreBinary and path-only records. Startup
  restore is synchronous, reads binaries one at a time on the lane, drops each
  inactive payload and suppresses partial-session persistence. Clean legacy
  text records with image candidates enter binary routing before text reads;
  dirty text drafts with image suffix retain their text/source recovery.

## Scheduling, close and ownership

One host lane serializes full binary reads, verified reacquisition, decode and
copy. It holds one latest pending preview, one explicit reservation and one
awaiting-result gate. Pending new-tab navigation carries metadata/token only;
a same-active-tab pending preview can alias its already held immutable source.
Slots clear their data/callback/tab/view references when delivered or discarded.
No pending closure retains a large image, and byte reacquisition failure does
not start an idle retry loop.

Native-dialog delivery is deferred. CanClose refuses while a dialog is active,
leaving a ready copy slot and its outcome intact. Outside a dialog it refuses
while work resolves, surfaces the actual outcome while the window exists, then
allows an acknowledged close. It does not join IO inside the UI close callback.
Direct Shutdown cancels/joins the lane and clears presentation/payloads safely.
A timed-out open abandons its output immediately and reclaims the small UI record
and reservation on the next UI frame/command; late callbacks cannot open a tab.
ChooseOpen ends the actual dialog scope before ordered opening, avoiding the
headless modal wait cycle. Nested copy dialogs refuse before another capture.

Steady state holds one active compressed source and at most one image bitmap.
Explicit open/reload can transiently retain the old accepted source/bitmap plus
one replacement source until safe UI adoption or failure cleanup; the awaiting
gate prevents another full task in that interval. A save dialog can retain its
captured immutable source while another tab's task executes. The pending alias
does not allocate another compressed source. A canceled codec may keep its
working data until it returns; no next codec runs concurrently. These are actual
ownership limits, not a measured process-memory or VRAM ceiling.

## Verification and preserved failures

Exact generated commands, combined stdout/stderr and explicit nested-PowerShell
exit records are in native-image-app-evidence-20261110. All Go ran through
native/scripts/go.ps1. No accepted prerequisite suite was run independently here.

- focused1:8 lifecycle cases PASS1.642s. focused2:12 cases+2subtests PASS2.320s.
  focused3:15 cases+2subtests PASS2.420s, capture test explicitly skipped, exit0.
  Repeats followed concrete integration corrections. Cases cover mixed routes,
  source deletion/copy, stale publication, serial/latest-only cancellation,
  transactional reload, modal/close cleanup, failed Flush, path-only restore,
  duplicate adoption, legacy dirty recovery and readonly capability behavior.
- Final app source adds the approved modal ready-copy assertion to the existing
  close lifecycle case. final-suite1 includes it and records app PASS19.490s.
  The same run records modules PASS1.041s, graphview13.736s, markdownview4.333s,
  sourceview2.567s, plugins4.533s, session2.879s, document1.280s, raster2.472s and
  editor3.832s. Overall exit1 is preserved: imageview's existing manual resize
  test compared zoom exactly with1 and received0.9999999999999999 after the
  narrow-toolbar geometry change. No app source fix was required.
- Root narrowed follow-up to the view owner's test-only tolerance correction.
  imageview-final1 ran the full ./modules/imageview package once: PASS1.203s,
  exit0. Its frozen imageview_test.go hash is
  CD2506B146E45D6AD663D9EC6DCA8BBF5DDDC39A42AB1EA37254C908459AB021;
  runtime ui.go remains2B168A93E3D3B0140D3114CD282113B02CE4CA1944820A7FA37F9F853071FBAC.
  final-vet1 checked the entire planned affected package set, exit0 with an
  empty preserved output file. Final acceptance combines the first suite's
  passing packages with this corrected imageview pass; final-suite1 remains
  a real failed run. No already-passing package test or capture was repeated.
- captures1 failed a test-only pending-job assertion before any PNG was written.
  The tiny decode had already completed. Its initial runner lacked explicit
  nested LASTEXITCODE propagation, so that run's outer metadata says0 despite
  stdout FAIL. Both are preserved with captures1-correction.md. The runner was
  corrected; subsequent tests preserve actual propagated exits. Earlier focused
  stdout showed PASS and was not repeated merely to replace metadata.

Combined final command was:

```powershell
native/scripts/go.ps1 -GoArguments @('test','./app','./modules/...','./core/plugins','./core/session','./core/document','./formats/raster','./ui/editor','-count=1')
native/scripts/go.ps1 -GoArguments @('test','./modules/imageview','-count=1')
native/scripts/go.ps1 -GoArguments @('vet','./app','./modules/...','./core/plugins','./core/session','./core/document','./formats/raster','./ui/editor')
```

Synthetic captures use the public MyGo Tester at760x480. Original image-light
and byte-fallback-dark captures remain preserved (captures2 PASS2.369s/exit0).
Root found the imageview byte action clipped in the532-DIP content pane and
assigned its owner the two-row correction. capture-fixed1 passed1.506s/exit0;
the separately named image-light-760x480-toolbar-fixed.png checks actual action
bounds and was visually inspected: all toolbar controls are fully visible.
It replaces only the current image-layout evidence; both originals remain.

| Capture | Bytes | SHA256 |
| --- | ---: | --- |
| bytes-dark-760x480.png |62382|28DBBA1A892E386F812152BDD6D4891B148258E02D6B6B5EEBFDCE05D9218A24|
| image-light-760x480.png |30030|B9899C8F56E603DA1E34ADDAAD1511AAEB3A8D6E5D7E6C98C18B201275F2C02C|
| image-light-760x480-toolbar-fixed.png |30620|9CC430CDE23FF506F73B9B11A58CA917E53FC4DD2DD45490F2AAEC86239BD8DD|

## Source identities and remaining product scope

The exact15-file inventory, byte counts and SHA256 values are frozen in
native-image-app-evidence-20261110/frozen-app-source-identities.json. New files:
app/binary.go, binary_lane.go, binary_actions.go and binary_test.go. Other edits:
app/app.go, state.go, view.go, plugins.go, reference.go, workspace.go, workers.go,
export.go, export_worker.go, search_navigation.go and outline.go. No pre-edit
app identity snapshot exists; none is claimed. No other owner's source was edited.
All15 file byte counts/hashes were checked again after final vet and still match.

The accepted raster subset remains noninterlaced PNG with at-most8-bit legal
color depths and8-bit baseline JPEG, with existing EXIF orientation handling.
Compressed input is capped16MiB, each dimension8192, pixels8,388,608, modeled
RGBA mip chain48MiB and modeled decode payload128MiB. Unsupported formats/modes
fall back visibly to bytes, without silent frame selection or downscaling.
Codec CPU between guarded reads is not preemptible. Synchronous startup restore
can wait for bounded disk reads. Tests and synthetic captures do not establish
physical OS input/IME, actual GPU-driver retirement, RSS/CPU improvements or
packaged image support. Product packaging/host verification awaits root source
acceptance and separate activation.
