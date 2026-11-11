# Mindmap markers — pure-format checkpoint

Status: **API READY / FORMAT CHECKPOINT PASS**, 2026-11-08 22:17 local time.
GLM owns only `native/formats/mindmap/**`. No tags, parser, graphview, app,
vendor, script, or documentation product files were changed.

## Exact seam for graphview

```go
// Icons returns an exact ordered copy plus presence. Absent returns
// (nil,false,nil); present-empty returns a non-nil empty slice and true.
func (d *Document) Icons(id string) ([]string, bool, error)

// SetIcons stores an exact, present icon list. Nil means present [].
func (d *Document) SetIcons(id string, values []string) error
```

Important semantics:

- `SetIcons` clones the input; later caller mutation cannot affect the model.
- `Icons` returns a defensive copy; later mutation of the returned slice cannot
  affect the model.
- Invalid UTF-8 and missing node reject before assignment, leaving exact
  serialization and dirty state unchanged.
- A present list receiving the same ordered values is a semantic no-op: old
  pointer, dirty state, and original bytes are retained. Nil and empty are the
  same empty candidate, but that candidate is not a no-op when the field is
  absent: it creates present `[]`.
- A real change assigns the cloned list, runs the existing whole-document
  validation, and restores the exact old pointer on failure. Dirty state is
  set only after successful validation, so an already-dirty document remains
  dirty on rollback.
- There is no remove-field API. Actual marker clearing writes present `[]`.
- No glyph vocabulary, sorting, deduplication, count limit, or aggregate-byte
  limit exists in the format layer. The format accepts 65 icons and larger
  valid arrays; graphview owns the UI 64/4096 action budget.

## Files

- `native/formats/mindmap/fields.go` — additive `Icons`, `SetIcons`, and the
  private ordered-list equality helper.
- `native/formats/mindmap/fields_test.go` — marker fixture and focused
  regressions.

No nil-slice normalization or `Node.Clone` change was made. The existing
canonical `[]string` writer safely serializes the present-empty clone case as
`[]`; the tests explicitly pin this behavior.

## Verification

Commands and results:

```powershell
native/scripts/go.ps1 test ./formats/mindmap -count=1 -timeout=120s
# ok levnote.local/native/formats/mindmap 0.498s

native/scripts/go.ps1 vet ./formats/mindmap
# exit 0, no output
```

After root's test-oracle correction, one focused run was executed:

```powershell
native/scripts/go.ps1 test ./formats/mindmap -count=1 -timeout=120s -run '^TestIcons|TestSetIcons'
# ok levnote.local/native/formats/mindmap 0.496s

native/scripts/go.ps1 vet ./formats/mindmap
# exit 0, no output
```

The earlier full owned-package result remains 0.498s (JSON evidence run
0.502s); no redundant JSON rerun was performed after the test-only correction.
No unrelated package, full integration, build, or benchmark was run.

## Covered regressions

- Read absent, present non-empty, and present-empty without dirtying or changing
  original bytes.
- Returned/input slice alias safety in both directions.
- Absent-to-present-empty via `SetIcons(nil)`.
- Same-list no-op retains old pointer, dirty state, and exact original bytes.
- Nil and empty are equivalent only against a present-empty field.
- Present-empty survives `SetIcons`, `Node.Clone`, and `AddNode` round trips and
  serializes as `"icons": []`, never `"icons": null`.
- Arbitrary Unicode/custom glyphs, order, and duplicate values survive a real
  edit; rich tag `className`, `style`, unknown tag members, and unknown node
  metadata remain present.
- Invalid UTF-8 and missing node reject atomically.
- Validation rollback restores the exact pointer/value from both clean and
  already-dirty states.
- 65 valid icons are accepted by the pure format, proving no UI cap leaked into
  the parser.
- Test oracles use the standard library's independent `slices.Equal`, not the
  product's private list-equality helper.
- Absent `Icons("plain")` is explicitly read as `(nil,false,nil)`.
- A same-list no-op on an already-dirty document preserves the prior real note,
  its dirty state, and the old icon pointer.
- Rich metadata is compared structurally and from its raw JSON: tag
  `className`, ordered `style`, unknown `riskLevel`, and unknown node
  `unknownNode`/`1.00` number spelling survive exactly.

The seam is frozen for graphview/app integration. GLM is now read-only unless
root identifies a concrete format defect.
