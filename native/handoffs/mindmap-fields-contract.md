# Native mindmap field stage — root-approved design, active

ACTIVE after root accepted the bounded process-resource checkpoint with its
recorded limitation. Read mindmap_fields_stage_plan.txt and the two proposal
files for source facts. This final contract supersedes their tentative choices.
No further resource sampling; implement and verify the assigned field work now.

## Pure-format seam and state

GLM owns the small additive API proposed in
mindmap-fields-contract-proposal.md: FieldKind with FieldNote/FieldHyperLink,
Field(id,field)->(value,present,error), SetNote, SetHyperLink, ClearField.
Reject an invalid field kind/missing node/invalid requested value before any
mutation. Preserve no-op dirty state, literal strings and unknown metadata.
Use existing error conventions; no new parser grammar or format version.
Full model validation, if used, must not partially mutate on failure or add
unnecessary repeated scans beyond the existing serialization path. Existing
RenameNode does not perform full document validation; do not claim otherwise.

SetNote("") stores a present empty string; empty SetHyperLink rejects and
ClearField(FieldHyperLink) removes it. The UI calls a setter only for a real
text change. Thus merely reading an absent empty note keeps it absent, and
clearing a nonempty note stores explicit empty. No separate remove-note UI is
required. Unknown fields and valid existing metadata remain authoritative.

## Native view

Astra owns graphview only: three simple native field buttons sharing ONE
existing independent Editor, exact field input labels from the UI proposal,
224px inspector clipping/scrolling and existing bounded shaping. Drawio keeps
the original label editor with no mindmap selector. Notes are plain text.
Read-only can select/read/copy fields and navigate valid links. Double-click
and new-child/sibling focus topic; ordinary selection keeps the selected field.

The draft key includes Buffer pointer,node,private field kind and original
presence+value (plus existing drawio page identity). Field switching must Flush
first, retain the old draft/field on failure, and never replace a conflicting
draft. Only the captured field is merged into freshly parsed current source.
Clean refresh yields to source; pending same-field/source-node conflict keeps
draft recoverable. Recovery identifies original node/field/presence and exact
text. A real Flush is one ordinary history transaction; no promise of one undo
for an entire typing session, and no-op barriers add no history.

Keep the existing pre-copy oversize guard and render cache lifecycle. Check a
long-note unchanged-frame path before adding any dirty/version cache; a narrow
fix is justified only by the introduced field workload. No parser/layout or
generic source-editor rewrite belongs here.

## Link action

Use one visible action: normally `打开链接`, and explicitly
`写回并打开链接` while this same node's hyperlink draft is pending. Capture
Buffer,node,original link state,and the exact pending draft identity at the
native activation gesture for pointer and keyboard; do not choose a current
node/target merely because a fixed-key button reports Clicked after redraw.

Flush first. Ordinary open requires unchanged captured current link. The
explicit write-and-open action may navigate only the same captured draft's
successfully committed authoritative value, verified against current source.
Changed/absent/mismatched draft/Buffer/node/field refuses visibly, with no
current-target fallback. Empty committed link says removed/not opened.

Complete destinations above4096UTF8bytes are visibly unavailable; do not keep
another full target copy or shorten it into a destination. Tooltip metadata
caps512bytes rune-safely, command label is fixed. Full source/draft is retained.
Host.OpenReference remains sole classifier/filesystem route. Its literal paths,
file URLs, network/unsupported refusal and readonly behavior remain unchanged.
Call it once and return; it can synchronously Release this view. Failed Flush
retains the view; pure navigation adds no transaction.

## Ownership, evidence and activation gate

GLM publishes the exact implemented format seam + focused results first.
Astra implements the one-editor UI/lifecycle and native tests. DeepSeek does
actual registered-adapter save/reopen/mode/tab/reference acceptance; shell
product edit only for a demonstrated required issue. No peer file edits or
child subdelegation. Owner completion can transfer to independent review.

Meaningful cases: absent/empty/no-op,Unicode/CRLF/trailing spaces,optional state
conflict,unrelated field merge,node deletion,Buffer replacement,readonly,
native Compose with field/save/mode/close,foreign group,oversize/Release/recovery,
undo/redo,drawio regression,and exact once safe link opening. Narrow light/dark
and readonly inspector image must show reachable input/link/recovery controls.
Full integration/build occurs only after focused gates and root review.
