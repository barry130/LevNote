# Mindmap fields — pure-format seam checkpoint

Status: **API READY / FORMAT CHECKPOINT PASS**

GLM owns only `native/formats/mindmap/**`. The implemented seam is exactly the
active root-approved contract:

```go
type FieldKind string

const (
    FieldNote      FieldKind = "note"
    FieldHyperLink FieldKind = "hyperLink"
)

func (d *Document) Field(id string, field FieldKind) (string, bool, error)
func (d *Document) SetNote(id string, value string) error
func (d *Document) SetHyperLink(id string, value string) error
func (d *Document) ClearField(id string, field FieldKind) error
```

Files:

- `native/formats/mindmap/fields.go`
  (`731F3D308AF01747D3420ABCDCDCB505A1A44B773826B8E5984C7CEF11F92D9C`)
- `native/formats/mindmap/fields_test.go`
  (`5278E35FB388EAEFCD3B28FF1F4F7AF7865E50578449186D521C00B53F2E798C`)

## Semantics for Astra

- `Field` returns the exact current literal string and presence; it never
  dirties the document.
- `SetNote` always stores a present note. `SetNote(id, "")` turns an absent
  note into present-empty, and setting the same present value is a semantic
  no-op.
- `SetHyperLink` rejects an empty value before mutation. It preserves exact
  literal nonempty strings without trimming, normalization, or URL parsing.
- `ClearField(FieldHyperLink)` removes the optional member; clearing an absent
  link is a no-op. `ClearField(FieldNote)` is also supported, although the UI
  can use `SetNote("")` for normal note clearing.
- Invalid field kind, missing node, invalid UTF-8, empty hyperlink, and
  whole-document validation failure all reject atomically before changing
  dirty state or serialization.
- Real semantic changes mark the document dirty; semantic no-op reads/clears
  retain exact original bytes.
- Unknown top-level/node fields and number spellings survive a real edit, but
  the existing canonical JSON reserialization boundary applies after a real
  edit.

## Focused verification

Commands and results:

- `native/scripts/go.ps1 -GoArguments @('test','-count=1','./formats/mindmap')`
  — `ok levnote.local/native/formats/mindmap 0.627s`
- `native/scripts/go.ps1 -GoArguments @('vet','./formats/mindmap')`
  — pass, no output

The tests cover absent/present-empty note, exact Unicode/CRLF/trailing spaces,
hyperlink set/clear/no-op, semantic no-op original-byte preservation, unknown
members/number lexemes, invalid kind/node/UTF-8/empty-link rejection, and
rollback on whole-document validation failure. The compact regressions also
prove absent-to-present-empty `SetNote` survives serialize/reparse and that
rejected `SetHyperLink`/`ClearField` calls preserve the previous field pointer,
value, unrelated edit, and already-dirty flag.
