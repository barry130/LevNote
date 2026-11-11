# Mindmap tag list editing — queued root plan

Status: FOCUSED ACCEPTED2026-11-09. Root reviewed final source/oracles and
actual format0.534s/graphview6.696s/vet0 evidence in
../artifacts/acceptance/mindmap-tags-20261109-044422-focused-final/.
Result595570D5050852387CA4700909478BE8E18DE25A1FBF96AE45FB986F41ED326F;
13owned files unchanged. Product now freezes for the separate app consumer
stage. The implementation instructions below remain historical context.

Original focused activation: Table-row package and final
source docs accepted. Root read mindmap-tags-checkpoint.md against current
format/draft/commit/sync/input code and accepts its design for this finite stage.
Astra alone implements new pure tags files and graphview tag controls plus
narrow private draft/commit-receipt and lifecycle hooks. Publish pure/native
checkpoints while proceeding with meaningful focused tests; full owned
mindmap/graphview tests and scoped vet after fixes. App consumer/captures and
new package need the later focused gate; no broader execution is authorized.
Root plans/reviews, no child delegation. Existing package/resource freezes hold.

Implementation clarification: action identity must also be invalidated by
selection/field changes and section-collapse/new gestures, including away and
back to the same node. A successful ordinary field write followed by callback
rejection leaves that write undoable; do not relabel it as an untouched source.
Successful tag commit followed by Release must not rebuild the released view,
and must not clear replacement or subsequently changed drafts. Keep existing
drawio ordinary-draft behavior with meaningful coverage of the shared helper.

## Useful bounded slice

Make existing mindmap tags manageable in the native node inspector. Keep the
canvas, SVG, layout, parser and source schema unchanged. Add a collapsed
`标签` section beside the existing collapsed markers section. Show the current
count and a bounded literal summary. Expanded section lists tags in source
order, each with a distinct ordinal/accessibility name and a remove button.
Add one small native text input plus explicit `添加标签`, and `清空全部标签`.
New tags are plain strings; existing string/object tags remain their original
kind. Duplicate and empty legacy strings must display distinctly (ordinal,
and visible empty-value notation). Do not deduplicate or remove by text.

An explicit add stores the exact nonempty valid UTF-8 input, with no trim,
case-folding, comma splitting or hidden normalization. Restrict this input to
one physical line and at most512 UTF-8 bytes, disclosing the limit. Reject an
invalid value visibly and retain the draft. Existing tags need not satisfy
the new-input restriction: show literal bounded summaries and keep them intact.
No object style/class editor, renaming, reordering, global tag search or network
feature in this slice. New text is never interpreted as HTML or styling.

Addition, removal and clear each make one undoable source transaction, after
the existing pending node field has successfully flushed. A prior field write
remains a separate undo. Removing the last tag or clearing a nonempty list
writes present `[]`; clear on absent/already-empty is a no-op. A node with an
unrelated pending topic/note/link draft cannot lose that draft. A pending tag
add input is independently recoverable and must never be silently applied to
another selected node, replaced Buffer or released view.

## Source and metadata contract

Read `formats/mindmap/types.go`, `parse.go:parseTags`, `serialize.go:encodeTags`,
and existing `fields.go` before choosing the additive pure API. Publish the API
and a concise draft/Flush/lifecycle design checkpoint before implementation.
Prefer purpose-specific add/remove/clear or a defensively copied list API;
do not ask UI callers to reconstruct existing tag objects just to remove one.
Caller-owned slices/Raw/Style/Extra must not alias stored document state.

Preserve all surviving object tag className/style presence flags and unknown
members, including exact numeric Raw literals and order. Existing canonical
document serialization may reformat the envelope after an edit; unchanged
open/save and rejected/no-op actions must retain the exact original bytes.
Document this distinction honestly. Do not add a competing tag parser or change
accepted parser fallback behavior. Malformed tags already reject rich parsing;
they continue to use the visible source path.

Budget all view snapshots and candidates BEFORE cloning. At most64 tags and
8KiB of total tag payload including unknown nested metadata;512-byte display
summary and at most64 native row controls. An oversize existing list is shown
with the current count/explicit source-only limitation, without cloning full
tag metadata each frame. Define one bounded estimator with early exit for
Raw/string/Object member bytes; public programmatic tags with no Raw must be
covered too. This view budget is not a new format schema limit. Full source
materialization/parse only on a valid activation within existing16MiB preview
guard, never on unchanged draw. Release drops tag caches/gestures/input layouts.

## Identity, draft and input rules

Capture the exact Buffer, selected stable nodeID, view lifecycle, displayed
version and ordered tag list/presence at pointer/key down. Remove also captures
the exact ordinal; equal tag text is not identity. Input captures include its
Buffer/version/selection. Cancel on scroll/zoom, pointer cancel, selection/mode/
tab changes and Release. Repeated Enter/Space must not apply twice. Keyboard
route must be reachable from a focused node/tag text input without unexpected
indentation or implicit composition commit. Reuse local native form conventions
and check real MyGo focus behavior; do not change shared editor/vendor APIs.

Before and after the normal field Flush, re-resolve current authoritative data.
Accept only the same Buffer/node/list/presence and active draft/lifecycle. An
own topic/note/link write can be accepted only with a proven exact receipt or
equivalent checks; arbitrary synchronous Changed callbacks must not redirect
the action. Never hold a parsed node pointer across Flush/Changed. Distinguish
successful field write followed by refused tag action in the visible message.
Foreign groups stay untouched; only cancel a transaction this action began.

Use this narrow staged-input policy unless the design checkpoint demonstrates
a blocker: while tag-add input is nonempty or composing, external `Flush`
refuses before modifying either field and shows `新标签尚未添加；请添加或放弃输入`.
Offer explicit `添加标签`, `放弃输入` and a recovery action. Remove/clear/selected
node/field switches also refuse while an add input is pending. The add action
alone may flush the ordinary node field via a small private helper extracted
from existing Flush, then apply its own exact tag operation after identity
checks. Existing controls already use Flush barriers, so no app/core contract
change is expected. A successful add clears only that tag input after success;
failed add retains it. A recovered copy includes exact pending strings, node
identity and field labels; if both node-field and tag input are pending, recover
both before clearing either. Host.RecoverDraft failure retains both. Emptying
the tag input is an explicit discard of its typed value, not a change to the
document's tag list. Guard native composition on focus escape as in table rows.
If this policy needs wider changes, publish the concrete blocker before editing
app/core. Uncommitted tag input is not claimed as crash-recovery persistence.

## Focused then consumer acceptance

Pure tests need independent mixed string/object/duplicate/empty list fixtures;
exact survivor metadata, no-op bytes, last-removal[], rollback on invalid/limit
and alias protection. Native tests need pointer and keyboard, composition,
selected-node/Buffer replacement, duplicate ordinal, concurrent source change,
Flush callbacks, foreign group, recovery and undo/redo of text+selection. Test
bounded unchanged frames and actual scroll cancellation. Do not implement tests
that merely restate helper code. Keep previous stage outputs immutable.

After root focused review, real registered-adapter app tests cover BOM+CRLF
save/reopen, source/preview/split, literal escaped SVG semantics already supplied
by the shared label path, and narrow light/dark views with recovery. Only then
authorize a fresh package. Reuse unchanged worker/runtime/editor acceptance;
no resource repetition or full-suite expansion without a demonstrated need.
No performance comparison, legal immunity or full XMind parity claim.
