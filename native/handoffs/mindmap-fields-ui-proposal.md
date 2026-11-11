# Mindmap note/local-link inspector — read-only UI proposal

Status: **planning only, not active**. Read `mindmap_fields_stage_plan.txt`,
the pure-format `mindmap-fields-contract-proposal.md`, and the current
graphview draft/sync/inspector/input/edit/Release paths. No product edits,
tests, builds, benchmarks or process launches were performed for this proposal.
The accepted link product stays frozen during the active resource checkpoint.

## Smallest useful UI

Keep the current224px scrollable, clipped inspector. Add one row of three
short native buttons, `文本 / 备注 / 链接`, above the **single** existing
independent `editor.Editor`. The field is selected before editing; there is
no dialog and no three-editor cache. The current field is visually marked.
Proposed bounded accessible names are `字段：主题文本`, `字段：纯文本备注`,
`字段：本地链接`, with `（当前）` on the selected button. They remain ordinary
native buttons with their existing Tab/Enter/Space behavior; do not invent
tab semantics without native support.

The topic input keeps `节点文本` so current topic/drawio tests and behavior
remain meaningful. The new inputs use `节点备注（纯文本）` and
`节点链接（本地）`. Drawio retains its single label editor and never shows
the mindmap field selector. Notes are literal text; neither Markdown nor URL
content is interpreted inside this editor.

Reuse ShapeLimit512 and the existing76/150px bounded field viewport, inspector
scrolling, error summary with full error tooltip, and same-row retry/recover
controls. Long field contents remain complete in the editor and source, using
the existing segmented text path. The field label/help distinguishes an empty
note from an absent note when relevant, without exposing JSON implementation
detail in every normal interaction. The local-link help says `仅打开本地引用；
网页和其他协议会被拒绝` and clearing help says `清空后移除链接`.

Ordinary node selection keeps the chosen field as a light view preference,
after successfully flushing the previous node. Double-clicking a node and
adding a child/sibling explicitly select/focus the topic field; these actions
currently call `focusLabel`, so that helper cannot blindly focus a previously
selected note/link. Outline selection may keep the selected field and does not
focus it. The selector remains usable for reading in a readonly document;
clean fields are readonly and existing failed editable drafts remain available
for correction/recovery, following the existing inspector behavior.

Switching fields always calls Flush first, even when the next field looks
empty. Failure retains the prior field, node, editor text/focus and recovery
controls. A successful switch releases the former editor view and constructs
only the requested field. A clean unchanged field selection has no source,
dirty, selection or undo side effect.

## Exact draft identity and optional state

Extend the private draft descriptor, rather than adding editor.Host methods:

```text
Buffer pointer
node ID (+ existing drawio page identity, unchanged)
private field kind: topic / note / hyperLink / drawio label
source presence bool + exact source value
editor baseline text + current editor + existing source version
```

The private kind covers topic/drawio as well as the two optional mindmap
fields. It can map to the format proposal's `FieldNote`/`FieldHyperLink` at
the graphview boundary; it must not be inferred from visible labels. Keep
`base` (editor text) separate from `sourceBase`/presence, because drawio HTML
labels already depend on that distinction.

| Existing field | User action | Authoritative result |
|---|---|---|
| note absent | Select/read/save or leave empty without a committed edit | Still absent, original bytes retained |
| note present empty | Select/read/save | Still present empty, original bytes retained |
| note nonempty | Clear text | Present empty note |
| note any | Edit Unicode, CRLF, spaces | Exact authored string; no trimming |
| link absent | Select/read/save empty field | Still absent |
| link nonempty | Clear text | Remove `hyperLink` member |
| link any | Enter nonempty text | Store exact string; validity of opening stays with host policy |

Typing and then returning to the baseline before any commit is a no-op.
Returning to empty after a previous real note commit leaves an explicit empty
note. No separate “remove note” button is needed in this finite stage.

Flush keeps the existing sequence: flush native Compose, compare with the
editor baseline, reject readonly/foreign group/replaced Buffer/oversize before
whole-source materialization, parse the current bounded source, find the exact
node, compare **presence and value of the captured field**, and apply only that
field to the current candidate. Another field's source edit may therefore
merge; same-field changed state, missing node or replaced Buffer conflicts.
Clean input yields to authoritative source on refresh/Undo instead of writing
its old value back. This is semantic identity in the current parsed document;
it does not claim historical tracking of a node deleted and recreated with
the same ID and exact field state between observations.

Candidate mutation/serialization must complete before the shared Buffer group
is opened. Each actual Flush creates one ordinary Buffer transaction/Changed
notification; a later app barrier with no new text creates none. Normal
non-composing input currently commits via OnChange; this proposal does not
promise one Undo for an entire multi-keystroke editing session. Pure read/open/
export/save and rejected mutations do not dirty the source. Real field edits
retain the existing whole-JSON reserialization boundary; unknown metadata
survives, but JSON whitespace is not preserved after a real edit. Do not claim
semantic source-caret mapping across that serialization.

On success update both the editor baseline and authoritative presence/value.
In particular, a cleared hyperlink's new baseline is absent, not present-empty.
On failure keep the independent draft outside parse/layout caches. Release
continues freeing those heavy caches and retaining a failed draft. Recovery
names the original node and the exact field, includes original presence, and
appends the complete draft text unmodified to a separate unsaved plain document.
Recovery does not apply a conflicting source mutation and clears the draft
only after the host accepts the new document.

The existing `pendingDraft` compares `Buffer.String()` repeatedly. Long note
fields make that relevant: cache the exact dirty comparison by the independent
input Buffer version, with composition checked separately, so unchanged frames
need not materialize the whole field. Full field/base ownership remains honest;
actual edit/Flush/serialization/recovery can still require linear work. This
is a proposed narrowly related implementation detail, not a measured fix yet.

## Pure-format seam

The existing GLM proposal is sufficient for the view without additional public
UI APIs:

```go
func (d *Document) SetNote(id string, value string) error
func (d *Document) SetHyperLink(id string, value string) error
func (d *Document) Field(id string, field FieldKind) (string, bool, error)
func (d *Document) ClearField(id string, field FieldKind) error
```

Use SetNote only after a real text change; use ClearField(FieldHyperLink) for
empty edited links. SetHyperLink("") remains an atomic validation error.
Field reads supply presence/value; Set/Clear perform semantic no-op checks and
preserve rollback/unknown metadata. Final enum/error names remain subject to
root/GLM publication on activation. No parser grammar change is needed.

## Opening an exact local link

The current `drawMindmapInspector` button passes `*node.HyperLink` straight to
Host.OpenReference. It has no view-level Flush, captured-node/field recheck or
copied-target bound. The app's later flush cannot fix the already chosen href;
this is the concrete seam to replace after activation.

Keep the explicit button available when another field is selected. Use
`打开链接` normally; while the **same node's link field** has pending text or
Compose, label the action `写回并打开链接` so post-Flush behavior is explicit.
A new nonempty draft link may use this action before an authoritative link
exists. An empty result yields `链接已移除，未打开链接`; a failed Flush blocks
navigation and leaves the draft recoverable. Readonly clean links remain
navigable. Missing Host.OpenReference is visibly unavailable.

Capture at the gesture's control identity the Buffer, node ID and complete
optional link state. Do not resolve from a later `selectedID` or a label. The
command is:

1. Flush the current independent field. On failure, stop and report through
   the existing visible draft error/Host.Report path.
2. Recheck Buffer and original node, bounded parse status and foreign-group
   state. Re-read the exact current link field.
3. For an ordinary `打开链接`, require the current presence/value to equal the
   captured link. If a different field was flushed, that cannot authorize a
   different link. External link changes refuse visibly and require a fresh
   action.
4. For explicit `写回并打开链接`, accept the committed link value only if the
   same captured draft/Buffer/node/field survived Flush, its original field
   comparison succeeded, and the current parsed value equals that draft's new
   post-commit authoritative baseline. This is the user's just-committed field,
   not a fallback to whichever link happens to be current. A failed/conflicting
   or superseded draft does not authorize navigation.
5. Require a complete nonempty destination within4096 UTF-8 bytes, then call
   Host.OpenReference once with those exact bytes. Return immediately, because
   opening another tab may synchronously Release this view.

4096 matches the accepted native Markdown destination budget and bounds the
one actionable snapshot; it is not a parser/storage cap. A longer stored or
draft link remains fully readable/editable but cannot be opened by this button.
Show a nearby explicit `链接超过4096字节，请在源码中查看或修改目标` notice;
never copy a prefix as an href. Tooltip/accessible metadata use at most512
UTF-8 bytes plus an explicit ellipsis, and never hold another full long target.
The button name remains a fixed bounded command, not the href.

Do not add URL normalization, whitespace trimming, scheme classification,
browser launch or OS protocol calls in graphview. The existing host handles
literal relative paths/local file URLs, anchors and visible policy refusals.
Opening/readonly navigation itself creates no document transaction; a clearly
labeled pending field commit may create its one normal transaction first.

## Focused evidence after activation only

Owner tests should use actual native selector buttons and Compose/input, not
only direct mutator calls. Cover note absence/empty and clearing link, Unicode
and CRLF, unchanged byte-preserving save, other-field source merge, same-field
presence-only conflict, deletion/replaced Buffer, readonly clean navigation,
foreign group, oversize refusal, Release/recovery and retry. Verify one commit
is one undo/redo step and a later barrier is a no-op. Retain a drawio label
regression plus double-click/new-node topic focus when the prior field is note.

Link cases need actual pending-link `写回并打开链接`, pending-other-field Flush,
external href change before activation, empty/over-budget/missing-handler
refusal, readonly success, exact once callback, unchanged source/history for
clean open, and synchronous Release during the callback. App owner verifies
the registered adapter's save/reopen/source-mode lifecycle and existing local
reference policy; no shell edit is assumed.

Inspect760×480 light/dark/readonly geometry with long fields and a conflict:
selector, field name, input, link action and both recovery controls must remain
reachable by scroll and within the content area. Keep this to owned field
behavior; no unrelated parser matrix or full release until root review.
