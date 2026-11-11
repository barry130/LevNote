# Mindmap note and local-link fields — additive mutation contract proposal

Status: design proposal only. `mindmap_fields_stage_plan.txt` is queued and
not active. This proposal was prepared from the current read-only source
(`formats/mindmap/{types,parse,serialize,edit}.go` and the existing graph
inspector) without product edits, tests, or benchmarks. Resource
fixture/validator/sampler review takes priority when DeepSeek publishes those
artifacts.

## Current format facts

- `mindmap.Node.Note` is `*string`. Absence and an explicit JSON `""` are both
  valid and are represented differently: `nil` means the `note` member is
  absent; a non-nil pointer to `""` means `note` is present and empty.
- `mindmap.Node.HyperLink` is `*string`, but the format rejects a present empty
  string during parse and serialization. Therefore absence and `""` are not
  equivalent for this field: a cleared link must be represented by removing the
  optional member, not by storing `hyperLink: ""`.
- Serialization emits known optional fields in a fixed order (`note`, ...,
  `hyperLink`, ...), then preserves unknown node members, then emits
  `children`. Unknown top-level/node metadata is already preserved.
- Existing mutators such as `RenameNode` and `SetCollapsed` perform a semantic
  no-op check before setting `dirty`. They validate the document after
  structural mutation and roll back on failure. Note/link mutators should use
  the same discipline.
- The current inspector only displays a bounded note excerpt and an open-link
  button. It does not yet edit these fields.

## Proposed public API

Keep this additive and small in `formats/mindmap/edit.go`:

```go
type FieldKind string

const (
    FieldNote     FieldKind = "note"
    FieldHyperLink FieldKind = "hyperLink"
)

func (d *Document) SetNote(id string, value string) error
func (d *Document) SetHyperLink(id string, value string) error
func (d *Document) Field(id string, field FieldKind) (string, bool, error)
func (d *Document) ClearField(id string, field FieldKind) error
```

`SetNote(id, value)` always stores a present note, including `""`. It is the
correct operation when the user edits/clears a note and the semantic result is
`note` present with the exact literal text. `SetHyperLink(id, "")` is invalid
and returns a validation error without mutation. Use `ClearField` to remove the
link member. For symmetry and caller simplicity, `ClearField(FieldNote)` is
valid too, but the normal note editor should prefer `SetNote` so an edited
empty note remains explicit unless the UI deliberately offers “remove note”.

Exact error wrapping should follow existing mutators:

- missing document/root/node: `ErrNodeNotFound` (wrapped with the node id);
- empty hyperlink: `ErrInvalidDocument` wrapped with a precise message, or a
  small exported `ErrEmptyHyperLink` if the owner wants a stable programmatic
  check;
- any final document validation failure: the validation error.

No method silently trims, normalizes, canonicalizes URLs, or converts line
endings. The caller receives an error when validation fails and no part of the
document has changed.

## Optional state and no-op matrix

| Current state | Request | Result | Dirty? |
|---|---|---|---|
| `note` absent | `SetNote("")` | present `""` | yes |
| `note` present `""` | `SetNote("")` | present `""` | no |
| `note` present `abc` | `SetNote("abc")` | present `abc` | no |
| `note` present `abc` | `SetNote("abc\r\n")` | present exact authored text | yes |
| `note` present | `ClearField(FieldNote)` | absent | yes |
| `note` absent | `ClearField(FieldNote)` | absent | no |
| `hyperLink` absent | `SetHyperLink("a.md")` | present nonempty literal | yes |
| `hyperLink` absent | `SetHyperLink("")` | unchanged | no; error |
| `hyperLink` present | `SetHyperLink("")` | unchanged | no; error |
| `hyperLink` present | `ClearField(FieldHyperLink)` | absent | yes |
| `hyperLink` absent | `ClearField(FieldHyperLink)` | absent | no |
| `hyperLink` present `a.md` | `SetHyperLink("a.md")` | unchanged | no |
| `hyperLink` present `a.md` | `SetHyperLink("a.md ")` | exact authored text | yes |
| unknown fields present | any note/link edit | unchanged | only if field changes |

A semantic no-op must not mark the document dirty or reserialize it on save.
Since `Serialize` already returns the exact original bytes while the canonical
hash still equals the baseline, preserving `dirty == false` is essential. A
field read must not dirty the document.

## Atomic mutation rules

1. Resolve the node before modifying anything.
2. Capture the old pointer/value for rollback.
3. Apply the pointer/value or remove it.
4. Call `d.validate()` with the candidate state.
5. On validation failure, restore the exact old state (including nil versus
   pointer-to-empty) and return the error. `dirty` remains false for a
   rejected no-op and remains unchanged for an already-dirty document.
6. On success, mark `dirty` only when the semantic optional-state pair changed:
   presence AND value must both be considered.
7. Never move unknown members into `Extra`, never delete/reorder them, and
   never rewrite numeric spellings or unrelated raw JSON.

Because `validate()` calls `encodeNodeMetadata`, an empty `HyperLink` is the
only field-specific validation expected for this stage. The mutator should
still use the document-wide validation path so a candidate cannot leave a
temporarily invalid model behind. No new parser grammar is required.

## Serialization and undo expectations

- Editing a note/link produces the normal canonical node serialization when a
  real change is made. This can reformat the JSON file even though unrelated
  unknown values survive; that is the existing serialization boundary, not a
  new promise of byte-preserving JSON editing.
- A rejected input leaves `Serialize()` byte-for-byte unchanged.
- A semantic no-op leaves `Serialize()` unchanged and does not dirty the doc.
- The view should wrap one committed field mutation in one text Buffer group
  so it is one appropriate undo transaction. The format mutator itself does not
  own text history; graphview/app owns that mapping.

## Field identity for independent drafts

A draft identity should include all of the following:

```text
buffer identity
node ID (and the field enum, not a display label)
field kind
original semantic state: presence + exact original string
```

For note/link this must distinguish absent from present-empty. The UI should
use the same enum values as the format API.

- An external edit in another field can merge by flushing first and then
  re-resolving the same node/field; the draft remains the user's exact edited
  text.
- A same-field external change must conflict unless the editor is clean and can
  yield to authoritative source.
- Deleting the node or replacing the Buffer must conflict and retain the draft.
- A failed Flush never drops the draft or partially replaces source.
- Recovery names the node and field and contains the exact draft text in a
  separate unsaved copy. It must not silently serialize the draft into the
  authoritative document.

## View-layer consequences (for Astra/DeepSeek, not this stage's product edit)

- Selecting a field must be an explicit user action. Switching topic/note/link
  flushes the current single draft first; if Flush fails, the switch does not
  silently replace it with another field draft.
- Note remains literal plain text. No Markdown parsing or remote content is
  introduced.
- HyperLink remains a literal string. Opening it must Flush, re-resolve the
  same node/link identity and exact current value, then call the existing
  `Host.OpenReference` exactly once. If Flush changes the hyperlink itself, use
  the post-Flush current value and document that choice; never navigate a stale
  captured value.
- A pending conflict blocks opening.
- The inspector can bound display/accessible metadata, but source and draft text
  remain complete. An over-budget link is visibly unavailable and never
  shortened into another destination.
- Existing topic editing and draw.io label editing must continue to work with
  the same one-active-field lifecycle.

## Focused acceptance cases to specify before activation

- absent note -> set empty stays explicit `"note": ""`;
- present empty note -> no-op set empty remains non-dirty;
- clearing/removing note changes presence without touching unknown fields;
- note with Unicode/CRLF and trailing whitespace is preserved byte-semantic;
- hyperlink set/clear/no-op and rejected empty link;
- unknown node/top-level members preserved after a real edit;
- failed validation leaves no mutation/dirty change;
- one real edit is one undo transaction; no-op edit does not add history;
- source switch, field switch, node deletion, Buffer replacement, readonly and
  recovery routes preserve draft/source truth;
- link open uses post-Flush current identity/value and existing host policy.

These are contract cases, not a request to run or add them while the stage is
queued or during resource measurement.
