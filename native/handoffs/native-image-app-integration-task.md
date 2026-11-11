# Native image tabs and bounded host scheduling

ACTIVE2026-11-10 for Astra after binary-copy-receipt root acceptance. No child
delegation. Root reviews;
implementation/tests stay child-owned. Current .22 package/evidence immutable.

Own app/** and only necessary cmd/levnote wiring (normally unnecessary), plus
handoff/evidence. No document/session/formats/vendor/editor/module/plugin edits:
their contracts are separate owners. Read native/AGENTS.md and accepted core,
copy, session, decode and bitmap lifecycle handoffs before implementation.
Imageview owner is DeepSeek; composition/registry owner will be GLM. No package,
real host, worker or driver/resource run in this stage.

## Binary tab authority, opening and recovery

Use exclusive Tab.Doc or Tab.Binary authority. Never store compressed bytes or
pretend the hex presentation is an editable document. A small readonly Editor
over <=64KiB hex prefix may be used only as presentation; it is not save/conflict/
session authority. Keep binary path/size/kind helpers centralized and audit all
app Doc/Editor dereferences, especially NewFromPlugin/recoverDraft, save dialogs,
Select/close, reference, documentView/status, preferences, watchers and shutdown.
Preserve existing text behavior/undo/flush failures. Binary clean close has no
dirty-text prompt. Don't grow a universal document framework.

Route image candidates before existing text Open's128MiB read. Use only a tiny
regular-file signature probe plus current compiled .png/.jpg/.jpeg suffixes;
probe is a hint, actual stable snapshot authorizes image decoding. Signature may
identify PNG/JPEG with another suffix; expose it as binary and state how the
image plugin was selected, rather than treating it as text. Disabled image plugin
does not fall through to another text adapter. Corrupt/mismatched/unsupported
images show a concrete byte-view reason. >16MiB candidates retain existing
bounded byte inspection and explicitly cannot copy via the bounded image path.
No guessing encoding, silent first-frame selection or downscaling.

OpenMany stays sequential with per-file failures and duplicate activation. Every
binary full read/reload/reacquire/decode/copy shares the single host lane below;
no parallel full snapshots from the old open goroutine. A new BinaryFile may be
created in a worker then transferred once to UI; never capture/mutate an already
UI-owned BinaryFile from workers. Flush old text before any destructive UI
transition; if flush refuses, keep old tab/view and release unused results.

Use accepted session.AddBinaryTab/RestoreBinary branch before text RestoreTab.
Path-only restore cannot retain compressed snapshots/pixels for all tabs. Read
restored binary files one at a time, drop each inactive payload immediately,
retain only small baseline metadata and map active index after failures. Do not
silently restore binaries via text decoding. Preserve mixed record order and
dirty-text recovery. If restoring asynchronously, protect restore ownership
against subsequent user opens/closes and avoid auto-persisting a partial restore
as if complete. Prefer a small explicit state over broad startup changes.

## One execution lane and UI publication

One app-wide binary execution goroutine/slot, latest pending preview only.
All long binary I/O/decode/copy operations enter it. Reuse proven completion-slot
patterns from preview_worker/engine_lane, but do not misuse process-engine IPC.
Only immutable token/snapshot/request inputs cross threads; no view/UI/current
tab reads in worker. On deactivation release bitmap through View.Release, drop
active snapshot and cancel/invalidate request. Old CPU codec may finish between
guarded reads, but the next full read/decode starts only after it returns. A
pending navigation request holds metadata/token only, no next compressed source.

Completion slots clear data, callback and retired tab/view references promptly.
Delivery on UI checks tab still open/active, exact owner, baseline token,
generation, enabled plugin and view mode. Stale callbacks only drop results,
never create a Bitmap. Modal dialogs must not permit stale publication; prefer
owned slot + window invalidate and UI delivery, not queued closures retaining
large images. No busy loop/continuous idle invalidation. Shutdown cancels/joins
lane and releases final owned presentation/payloads, even when called directly
by tests without CanClose. Do not block ordinary tab switching to join codec.

The host is allowed one active immutable compressed snapshot and one view bitmap.
No inactive bitmap/thumbnail/full-source cache. In-flight canceled result may
exist until that single lane finishes; disclose actual bound. Failed reacquire
after external change cannot silently replace the accepted baseline. Existing
active image/snapshot can remain visible until explicit reload/close; explain
that leaving the tab drops the held pixels/bytes and return requires verification.

Root-reviewed integration clarification (2026-11-10): a pending same-active-tab
preview may alias the already held immutable snapshot so byte-to-image viewing
still works after source deletion; it allocates no second source. New-tab pending
navigation carries metadata/token only. Explicit open/reload may temporarily hold
the old accepted source plus one replacement source until safe UI adoption or
failure cleanup. An awaited result blocks another read/decode. Capturing accepted
bytes across a native save dialog also has a bounded transient lifetime; disclose
these exceptions instead of claiming exactly one source throughout. Real inactive
tabs still retain no full source or bitmap. No RSS/VRAM ceiling is established.

Wire editor.Host.BeginImage using the settled editor.ImageRequest/ImageResult
contract. Call neutral modules.DecodeImage (composition owner creates it), not
formats/raster directly. It returns owned RGBA; only view creates/disposes Bitmap.

Settled composition API (2026-11-10 root coordination):
`modules.DecodeImage(ctx context.Context, r io.ReaderAt, size int64)
(editor.ImageResult, error)`. Pass snapshot.ReaderAt() and snapshot.Size(); keep
the immutable snapshot/token lifecycle in app. Registry/adapter ID is `image`.
GLM composition owner is active and will publish API availability first. Session
and imageview source are frozen; root is reviewing their saved results.

## Copy, reload, command capabilities

Ctrl+S states image is readonly; provide visible 另存副本 and byte-view action.
SaveAs captures current snapshot/token before dialog and revalidates afterward.
If no active snapshot, queue verified reacquire on lane. Use PublishBinaryCopy
worker then AdoptCopy on UI; rebound snapshot shares source bytes. Same path
verifies/no rewrite; existing destination refuses. A completed destination is
reported truthfully even if tab closes/changes before adoption. Do not hide a
committed-but-unverified result, delete it, or mutate a new tab's authority.
Accepted explicit copy is not overwritten by navigation preview coalescing;
reserve lane through adoption/outcome handling. Closing may cancel uncommitted
work, but shutdown/close must account for committed results and report outcome.

Explicit Reload can OpenBinary a replacement owner on the lane and adopt only
if the initiating tab/token/generation still matches; this avoids mutating a
live owner on a worker without adding another core contract. Failure preserves
existing accepted authority/pixels. Disk checks use immutable BinaryToken and
existing Change/Missing UI; no text dirty state or invented encoding.

Image preview and readonly bytes are the only two modes. Gate find/replace,
GoToLine, text selection shortcuts, outline, split, export and template creation
by actual tab capabilities. Show format/dimensions/byte size/readonly in status,
not line/column/encoding. Plugin disable/refresh cancels view work and retires
bitmap, with unconditional byte fallback and exact binary authority retained.
Old text flush failure must prevent the plugin transition. No empty image New.

## Verification and source delivery

Meaningful tests with actual public APIs and controlled delayed work: mixed
open/restore/save/close, PNG/JPEG success, signature/suffix mismatch and disabled
fallback, corrupt/oversized admission, duplicate opens, source-deleted held copy,
exclusive destination, stale receipt/adoption, same path no rewrite, external
change/reload, cancellation A->B->close and newest pending request, maximum one
active task, completion/payload cleanup, modal stale callback, shutdown join,
failed rich-text Flush retention, readonly controls and all nil Doc paths.
Use existing text suites once at final after fixes; no unrelated worker smoke.
At most two integrated760x480 Tester captures (image+byte fallback, light/dark)
after actual provider integration, clearly synthetic. No performance claim.

Save checkpoint before checks and exact commands/stdout/stderr/exits/failures
under native/handoffs/native-image-app-evidence-20261110. All Go uses go.ps1.
One final affected app/modules/plugins/session/document/raster/editor suite and
vet after integration sources stable; coordinate with root so standalone owners
are finished. Freeze native-image-app-implementation-result.md with source
identities, actual limits and remaining verification. One portable release comes
only after root acceptance of integrated source, not after private helpers.
