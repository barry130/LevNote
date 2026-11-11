# Binary session path-only recovery result

2026-11-10. Ownership was `native/core/session/**` only. No app, document,
view, vendor, product package, host, or worker files were changed.

## Actual Version 1 wire contract

Ordinary text tabs are unchanged and have no `record` key. Text-only session
JSON bytes, field order, escaping, and `RestoreTab` behavior are unchanged.

A binary tab is a distinct object in the existing ordered `tabs` array:

```json
{
  "record": "binary",
  "kind": "png",
  "path": "C:\\absolute\\path.png",
  "title": "optional",
  "pluginId": "optional",
  "view": "optional"
}
```

`record` is required and must be exactly `"binary"`. `kind` is required and may
be `"png"`, `"jpeg"`, or `""` for a path whose kind is not yet known. `path` is
required, absolute, NUL-free, and valid UTF-8. `title`, `pluginId`, and `view`
are optional bounded UTF-8 metadata. Binary records accept no other key.

Presence of `draft`, `encoding`, `bom`, `originalPath`, `selection`, or
`scroll` is rejected even when the value is empty, zero, null, or false. A
duplicate field, unknown discriminator, unknown kind, or unknown payload key is
reported as `invalid-tab`; it is never coerced into a text tab. A malformed
binary entry does not discard neighboring text drafts or shift the valid active
mapping. Only the first 100 tab entries are decoded; later entries are counted
and omitted as before.

The binary save projection is used only for mixed/binary snapshots. Ordinary
text `Tab` values are projected directly into a bounded `[]any`; there is no
per-text `MarshalJSON` and no extra full-draft encode/copy. Preflight uses a
separate exact binary object counter and remains equal to actual encoder bytes.
The load scanner slices key names and small discriminator values from the
already validated per-tab bytes. It does not copy draft values through
`json.RawMessage`; ordinary text has one draft-sized decode.

## API and restoration boundary

- `SnapshotBuilder.AddBinaryTab(BinaryTab)` validates and captures small
  path/metadata authority before any filesystem operation.
- `BinaryTab` holds `Path`, `Kind`, `Title`, `PluginID`, and `View`.
- `RestoreBinary(Tab) (RestoredBinary, error)` is pure. It does not open a
  file, read bytes, create a snapshot, or retain pixels.
- Existing `RestoreTab(Tab)` returns `ErrBinaryDispatchRequired` for binary
  tabs, so current text callers cannot dereference a nil document.
- `RestoreBinary` on a text tab returns `ErrTextDispatchRequired`.

App integration remains later ownership: the host dispatcher must inspect
binary authority and call `RestoreBinary`, then queue its own bounded binary
read. Images are not yet enabled in the application by this session change.

## Validation

- Final affected session suite:
  `native/scripts/go.ps1 -GoArguments test,./core/session`
  (actual wrapper: `& .\scripts\go.ps1 -GoArguments @('test','./core/session')`)
  — `ok levnote.local/native/core/session 1.538s`, exit 0.
- Final affected vet:
  `native/scripts/go.ps1 -GoArguments vet,./core/session`
  — no output, exit 0.
- Evidence files are in
  `native/handoffs/binary-session-evidence-20261110`.

Focused regressions cover mixed capture/save/load, strict binary keys and
explicit forbidden values, unknown kind/discriminator, damaged binary between
good drafts, active-index mapping, clone independence, rejection before
filesystem creation, `MaxTabs`, `MaxBytes`, text-only byte compatibility,
mixed Marshal/preflight oracle, nested/last-field scanner cases, escaped keys,
and absence of the forbidden text-marshal/RawMessage copy paths.

## Scoped identities

SHA256:

- `native/core/session/binary.go`
  `6af880be6931d0284c8b971c452ed9d0c632a1da257eef7867cf202acd9fbb77`
- `native/core/session/binary_test.go`
  `0b1b4da5f0c59815f7653167138551113a7211ecbf35a5f79a7e0da68591923a`
- `native/core/session/binary_structural_test.go`
  `4c5d6fe049a84033635b2f296f9697651a7858966bf4b262f65bafda37eb6eab`
- `native/core/session/builder.go`
  `3488eead7285501324da9d75c418b9fc5d93eebe9b22eff1e4556480931d7121`
- `native/core/session/load.go`
  `8a1d86e7dffc6edf74d0d7ad43b7d72af2be3d09474b6d2aea4c657d34a0c610`
- `native/core/session/output.go`
  `672355d4087018592aa0e9ea0d15a4e4c06d2198b36571ef973914ea7bbdbb4b`
- `native/core/session/session.go`
  `bd3593bcba8e22fc96b1a27b8bd772c9e183d413723d9346d0ae4173afce9fe3`
- `native/core/session/budget.go`
  `026fca5ce0db9fb9dd91864dc104cf25b586043735d628c35e2f5351b99301f0`

Existing test files in the same scoped package were not content-changed except
by shared formatting: `budget_test.go`, `load_test.go`,
`output_benchmark_test.go`, `output_test.go`, and `session_test.go`.
