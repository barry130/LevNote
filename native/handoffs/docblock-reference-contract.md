# Doc-block reference safety — root decision, active

ACTIVE after root acceptance of the mindmap field release. This final contract
supersedes tentative metadata choices in the GLM proposal. No child delegation.

## Small parser seam

Use existing DocBlockRef.HeaderRange and correct it to the exact byte interval
after the opening fence line up to the separator line start. No second identity
struct or duplicate Src string. Add `ReferenceProblem string` on DocBlockRef:
empty means the recognized reference header is structurally usable; it does NOT
promise host support, local path existence or navigation permission. Source,
payload and existing Attrs/Extra remain authoritative; do not serialize edits.

The parser sets a bounded, explanatory problem for unterminated fence, duplicate
recognized headers (id/kind/mode/src/title), missing/invalid id, missing kind,
missing/unknown mode or missing/empty recognized src in reference mode. Retain
existing parsed values/source for display; new diagnostics must not duplicate
the same existing warning. Detect recognized duplicates using a fixed small set,
not an unbounded list/copy of arbitrary unknown key names. Unknown kind stays
literal/preserved, with host file routing independent of the kind string.

An explicit inline block is not a reference; no Open button for a normal inline
block without src. An inline block with src must display an explanation that it
will not open that header as a reference. Missing/unknown mode is visible and
not silently treated as reference. This intentionally tightens the old button's
`src != ""` condition; document it. Unknown extra header duplication is not an
opening ambiguity and need not block a valid recognized reference. Two blocks
with the same id/title remain independently navigable through exact source
ranges; no cross-document/ID registry or global uniqueness validation.

Malformed missing-separator input remains the newly accepted raw-code fallback.
Valid source/header normalization remains existing grammar (lowercase accepted
keys and TrimSpace value); do not claim raw untrimmed target bytes. No extra
grammar, nested editor, automatic file loading or network behavior.

## Native view

Retain one ordinary native `打开引用` button per existing reference block, with
pointer and focused Enter/Space behavior. Capture Buffer/version, immutable
preview row identity and enclosing/header UTF16 ranges plus bounded COMPLETE
recognized src from the actual gesture; never take current data because a
reused key reported Clicked. Capture no full header/raw copy or Block/Document
closure that outlives the frame. Drop canceled/superseded/released gestures.

Flush first, then map unchanged ranges through MapRangeSince, require the same
buffer and bounded complete parse, and verify exact current enclosing/header
range, BlockDocBlock type, ReferenceProblem empty, mode reference and full src.
Mapping proves the captured block bytes were untouched; no extra retained raw
header/hash or second parser needed. New enclosing fences, target edits, failed
range history or changed Buffer refuse visibly, no same-id/title fallback.
Foreign groups and incomplete/oversize source refuse. A changed draft in the
same block cannot authorize a stale open; a fresh gesture is required.

Allow readonly navigation; pure open adds no history. Host.OpenReference is the
only classifier/opener. Call exactly once and return without dereferencing
cleared caches; preserve the accepted ListState snapshot/reentrant Release fix.
Use current rows, existing inline-link safety helpers when they fit, but do not
refactor already accepted inline behavior into a generic framework.

Full src above4096UTF8bytes is disabled with a visible reason, never shortened
into a destination. Title/kind/mode/tooltip UI copies are rune-safe bounded:
title at most256bytes, each kind/mode64bytes, tooltip512bytes plus explicit
ellipsis. Keep32KiB block guard and complete source. Status/problem text bounded
and visible; no full target appended to an accessible label. Normal inline
payload remains the existing readable source presentation.

## Ownership and acceptance

GLM: formats/markdown/{types,docblock}.go small metadata and focused tests, API
published first. DeepSeek reference-view owner (capacity replacement for Astra):
modules/markdownview/** native button/capture/lifecycle,
focused actual-input tests and narrow normal/refused image; final docs. DeepSeek:
real app registered-adapter local file/refusal/synchronous Release and pending
draft acceptance, then final full test/vet/build after root gate. No broad parser
matrix/resource process run, no vendor/app policy changes absent actual defect.

Keep same-id/title blocks distinct; test insertion ahead, target/construct edit,
newBuffer/Restore, pending unrelated Compose success and same-block conflict,
readonly/no-op,4096/4097byte boundaries, duplicate recognized headers and inline
src refusal, host unavailable, canceled native action and callback Release.
Header ranges include exact CRLF boundaries and exclude the separator itself.
