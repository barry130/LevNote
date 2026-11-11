# Mindmap tags — proposed API and lifecycle checkpoint

Status: ROOT DESIGN ACCEPTED; FOCUSED IMPLEMENTATION ACTIVE, 2026-11-09.
Table-row.3delivery and final documentation are accepted. This checkpoint
follows `mindmap-tags-next-stage.md`; root reviewed it against actual format
and graph lifecycle sources. No parser/schema/canvas/SVG/app/core/vendor change
is authorized. Consumer/package execution waits the later root focused gate.

## Pure API and exact preservation

Add `formats/mindmap/tags.go` and focused tests, using existing `Tag`, `Object`,
`parseTags`, `encodeTags`, document validation and serialization:

```go
func (d *Document) Tags(id string) ([]Tag, bool, error)
func (d *Document) AddTag(id, value string) error
func (d *Document) RemoveTag(id string, ordinal int) error // zero-based
func (d *Document) ClearTags(id string) error
```

`Tags` is an ordered defensive copy: absent -> nil/false; present empty ->
non-nil empty/true. Clone strings, Raw, and every Style/Extra member key/Raw;
mutating a returned slice/nested Raw cannot change the document. Add appends a
plain-string tag, requiring nonempty valid UTF-8; no trim/split/deduplication.
The one-line/512-byte input and64/8KiB editing limits belong to the view, not
the schema. Existing empty/duplicate strings and object tags remain valid.
Remove addresses exactly one ordinal, including duplicate texts. Invalid index
wraps ErrInvalidDocument; missing node uses ErrNodeNotFound. Last removal writes
present `[]`; Clear on absent/already-empty is a true no-op. Mutations build owned
survivors, validate before success, restore the old pointer on failure, and set
dirty only for a real edit. No arbitrary `SetTags` reconstructive API is needed.

Every survivor retains Object/ClassNameSet/StyleSet, decoded strings, ordered
Style/Extra members and exact Raw numeric spellings. Parsed Tag.Raw is retained
in model copies; existing serializer rebuilds the tag envelope on a real edit.
Tests compare modeled fields/presence plus exact unknown-member Raw and order
after reparse, rather than require identical whole-object whitespace. Unchanged
open/save, rejected and no-op actions still return exact original source bytes.

## One pre-clone budget and idle presentation

Graphview defines one overflow-safe early-exit estimator over borrowed tags:

```
count <= 64
cost = sum over tags (
    1 + len(tag.Raw) + len(tag.Text) + len(tag.ClassName)
      + sum over tag.Style.Members and tag.Extra.Members (
          1 + len(member.Key) + len(member.Raw)))
cost <= 8192 bytes
```

Lengths are UTF-8 byte lengths; Raw **and** decoded fields are both counted.
The fixed1per tag/member bounds even public programmatic empty-key/empty-Raw
objects. Raw=nil is not an escape. This is an editing retention/work budget,
not a serialized-size or process-memory promise. Check count first; compare
each length with remaining budget before addition, no JSON marshal/clone to
measure. Validate current and prospective borrowed survivor/add sequence before
any deep copy, including clear/remove from an oversize current list. After full
candidate serialization/reparse, recheck the actual parsed list budget before
source commit, since its Raw representation can have a different size.

Presentation caches by Buffer/version/nodeID/sync validity; only bounded text,
ordinals and flags are cached, never full tag metadata on each frame. Summary
max512bytes and64iterations; each expanded literal label is separately bounded
and ordinal-named, with visible empty-string notation. Oversize lists show true
count and source-only guidance; no truncation is committed. Current cached
version must match before a gesture snapshot; source changed without Draw is
rejected. Full-source parsing/materialization happens only after valid activation
and the existing16MiB guard. Source-change Draw retains existing parse behavior;
unchanged Draw adds neither parsing nor tag metadata copies.

## Draft barrier and two commit receipts

Tag input has its own Editor/Buffer, owner source Buffer/nodeID, lifecycle epoch
and cached pending state. It is pending when nonempty **or composing**. Public
`Flush` checks that state **before** its existing reentrancy early-return and
returns `新标签尚未添加；请添加或放弃输入` without changing either field. This also
covers ordinary field OnChange, which currently calls public Flush. Add alone
calls a private ordinary-field helper directly; no global bypass flag is set.
Remove/clear/node/field switches keep using the public barrier. Emptying the
input or explicit discard removes only the staged tag text, never source tags.

Gesture captures Buffer/node/epoch/displayed version, exact list presence and
full ordered values, remove ordinal, tag input object/Buffer/version/selection,
and ordinary field draft identity/state. Re-resolve authoritative tags before
and after the field helper, never carry a parsed node across Changed. Add first
validates its complete nonempty UTF-8 one-line <=512byte value and prospective
budget; invalid add keeps input. CR/LF are refused; spaces and commas are literal.

Extract the existing ordinary Flush body into a private helper returning a
receipt of whether it actually wrote. Immediately after `commitBuffer` returns
from synchronous `host.Changed`, **before** updating draft base/selectedID/cache,
prove same view epoch, Buffer, selectedID, field draft/input identity, unchanged
input version/selection, and exact expected source version/text/selection. A
no-op is0source versions; a field write is exactly1 with its serialized result
and normal whole-buffer replacement caret. Check again after input.Flush, which
can itself callback. A foreign edit/selection/Release/replacement stops the tag
action and must not restore old selectedID or reparse a released view. If the
field write already occurred, report `节点字段已写回，标签未修改`; it is separately
undoable. The public nested Flush still sees the pending tag even while helper
reentrancy protection is set, so callback save/switch cannot bypass the barrier.

Then build/serialize/reparse a fresh authoritative candidate, enforce16MiB plus
tag budgets, and commit tags in one owned transaction via selection-preserving
commit. Preserve the source selection at tag-transaction start numerically and
directionally, clamp only to valid UTF-16 boundaries with a tag-specific notice.
Never cancel a foreign group. After the tag commit's Changed callback, success
may already have written even if lifecycle changed: clear **only** the captured
input whose exact version/content was applied; a replaced or subsequently edited
input survives. Do not call unconditional finishCommitted or resurrect caches.
Receipt identity controls refresh; a completed source transaction is never
misreported as an unperformed tag action. Input is cleared only after success.

## Recovery, Release, focus and finite verification

`refreshDraft`/source refresh preserve both drafts while either is pending;
`ensureSynced` preserves selected node for pending tag input, even if the node
disappears. Drawio's draftPage branch continues to require its ordinary draft,
not a new aggregate pending boolean. Separate clear helpers avoid accidental
cross-clearing. Recovery captures both exact staged strings, node identity and
field labels before calling Host.RecoverDraft; it does not invoke public Flush
or retry the mutation. Failure retains both; success clears only the captured
unchanged draft objects, never callback replacements. Composition text already
present in each input Buffer is included. Uncommitted input is not crash recovery.

Release increments epoch, drops cached presentations/gestures/handles/layouts,
and keeps pending input buffers. Its public Flush still refuses tag-pending state,
so direct Release cannot silently write the ordinary draft. Return rebinds to the
same owner only; changed Buffer/node requires recovery/discard, never reassignment.

Collapsed 标签 sits beside markers in the existing scroll inspector. A local F6
route from node/tag input focuses the tag section/add action; Tab/ShiftTab then
traverse native controls. During composition, local focus-escape shortcuts keep
focus/text/selection/group and show confirm-first guidance. The existing node
Tab override receives the same local composition guard; source/drawio unchanged.
Enter/Space activates buttons once; raw continuation clicks and repeats cannot
apply twice. Real scroll/zoom handlers cancel tag gestures and returnfalse so
normal inspector scrolling/canvas behavior continues. Pending values remain.

Implementation scope: new pure tags files/tests; new graphview tags files/tests;
narrow adapter/draft/sync/view_ui/input hooks and private field receipt. No new
host contract. Pure independent metadata/alias/no-op/index/rollback tests first;
then native actual pointer/keyboard/composition/duplicate-ordinal, callback and
dual-draft recovery, read-only/foreign-group, UTF-16 undo/redo, budget+idle and
scroll tests. Full owned format/graphview packages and vet once after fixes.
App captures/consumer tests and fresh package wait the later root gate; no
runtime/resource/full-repository replay.
