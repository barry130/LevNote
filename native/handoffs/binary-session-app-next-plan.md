# Binary session and app continuation plan

PLANNED2026-11-10 by root. Not implementation ownership yet. Activate only
after binary core acceptance; image rendering additionally waits on accepted
raster admission and bitmap retirement. Current .22 package stays immutable.

## First continuation: path-only binary session records

Keep text Version1 JSON bytes and draft/size accounting stable. Add a small
explicit binary discriminator/validated kind and path-only record branch.
Binary JSON must not contain fictitious draft, encoding, BOM or text selection
fields. Use a small custom Tab wire encoder only if needed; avoid converting all
documents into a generic framework. Update tabEncodedLength and preflight to
match actual encoded bytes, and retain independent Marshal oracle tests for
existing text and new mixed sessions. Unknown kinds/mixed invalid fields must
report an issue and leave neighboring text drafts recoverable.

Restore should dispatch binary records before text OpenWithEncoding. Return
explicit mutually exclusive document/binary authority; do not invent a writable
Document for image bytes. Prefer path record restoration by the host so it can
queue one bounded binary read at a time rather than retain all restored full
snapshots. Restored inactive images retain only small accepted baseline metadata;
no compressed bytes, decoded pixels or thumbnails. Session cap and path checks
continue to apply. This stage does not claim pixels are implemented.

For additive Version1 compatibility, unknown old readers may attempt text reopen
and report failure; current code must never silently coerce binary into UTF8.
Presence of forbidden binary fields must be validated from wire keys, including
explicit empty/false values if strict path-only contract is chosen. Use a bounded
per-tab raw decode only for already bounded first100 records, not full-array
RawMessage copying. Encoding/json custom marshal needs an alias for ordinary text
to avoid recursion and preserve text order/escaping. Independent serialization
length assertions must still account for both branches exactly.

Serialization memory caution: a custom Tab.MarshalJSON for every text tab can
add a second draft-sized byte allocation, undoing earlier session-output work.
Prefer a snapshot wire projection with unchanged text Tab values and distinct
small binary wire values (no per-text-tab marshal), or another measured bounded
approach. Do not add MarshalJSON to the full draft-bearing Tab solely to omit
binary fields. Choose and document the smallest branch in session output rather
than silently introducing extra copies of dirty text.

## Host integration after contracts settle

Add BinaryFile authority to Tab with either text or binary ownership. A bounded
hex presentation buffer may exist solely for inspector display, never as saved
source, dirty authority, file identity or persisted draft. Centralize small tab
capability/path helpers to avoid scattered unsafe t.Doc dereferences; do not
refactor unrelated text features. Read existing Select/flush, modal ownership,
disk worker, save, restore, close and plugin lifecycles before editing.

Candidate suffix/signature probe must be bounded before the128MiB text load.
Probe is only routing hint; accepted full stable snapshot authorizes image
admission/decoding. Disabled/unsupported/corrupt/oversized images stay visibly
inspectable. Format registration must allow image plugin disablement with
unconditional byte fallback; no image New/template/edit/remote capability.

One app-wide decode/copy lane with immutable tokens, latest-request coalescing,
generation/owner/tab checks. Don't capture live BinaryFile in worker goroutines.
Codec cancellation invalidates publication; finish in-flight codec before next
work, drop canceled results and stale completion closures. SaveAs must preserve
the active accepted snapshot even if original changes/deletes. Inactive reacquire
verifies baseline. Snapshot source memory is16MiB maximum; later bitmap budgets
are explicit payload limits, not process RSS. Same path never rewritten; existing
destination not overwritten; committed-but-unverified outcome remains visible.

The current core SaveAs mutates BinaryFile on its owning goroutine. Do not call
it concurrently from a worker to implement asynchronous copying. Before host copy
lane integration, decide a narrow token-based publication receipt + UI adoption
extension (preferred if copy is asynchronous) or explicitly document a synchronous
bounded first slice. No shared-mutable authority races or forged adoption. This
is a later core-owner extension, not justification to delay current pure tests.

Ctrl+S explains readonly image; SaveAs uses exact snapshot bytes. Reload is
explicit, change/missing notifications retain current accepted image until
decision. Image status shows format/dimensions/bytes, not line/encoding. Text
find/replace/GoToLine/outline/split/export are gated by capabilities. Tab close,
mode change and plugin disable cancel work and retire bitmap; old text Flush
failure must leave active tab/view/draft intact. Clean binary close has no dirty
text prompt. Restore order/active mapping and path duplicate detection include
binary tabs.

## Verification and delivery

Use pure/method/Tester tests for mixed session compatibility, exact copy/conflict
states, duplicate opening, stale async A->B->close, one lane, plugin disable,
readonly commands, no retained inactive payloads and view geometry. Physical
driver/IME/resource claims require separate bounded evidence. Build one next
portable package only after meaningful source checks pass; do not make a release
for each private helper. Keep docs concise and consolidate actual limits rather
than expanding repetitive validation chronology.
