# Source-plugin schema2 loader checkpoint

Owner: GLM. Scope implemented and frozen for independent review:
`native/core/plugins/**` schema2 validation/load/copy tests and
`native/examples/plugins/**` samples. No app, engine, docs, session, or view
files were changed.

## Public loader boundary

```go
type OutlineRule struct {
    Prefix string
    Suffix string
    Level  int
}

type SyntaxManifest struct {
    // existing schema1 fields
    Outline []OutlineRule // schema2 only
}
```

- Schema1 remains behavior-compatible and rejects an `outline` field.
- Schema2 keeps `apiVersion` at1. Unknown top-level and rule fields are rejected.
- `outline` must be an array when present; absent or `[]` means no outline
  capability; `null` is invalid. At most8 rules are accepted.
- `prefix` is required and non-null; `suffix` is optional, but a present `null`
  is invalid. `level` is required and must be an integer1..6.
- Marker strings are valid UTF-8, up to16 bytes, and reject NUL/CR/LF/control
  bytes. Duplicate exact `prefix`+`suffix` signatures are rejected regardless of
  level.
- A literal U+FFFD marker is valid. Ill-formed/unpaired surrogate escapes are
  rejected by walking JSON string escape sequences with correct backslash
  parity. A JSON `\\ud800` marker decodes to the valid literal `\ud800`; actual
  `\ud800`/`\udc00` escapes remain rejected, while paired surrogate escapes and
  ordinary escaped Unicode are valid.
- `Manager.Syntax(id)` and internal loaded syntax state use deep slice copies:
  `Extensions`, `Keywords`, and `Outline` each receive independently allocated
  backing arrays. Mutating a returned `SyntaxManifest` cannot alter matching or
  later returned syntax.
- Existing installation protections remain unchanged:64KiB manifest cap,
  256-plugin cap, bounded directory listing, Lstat regular-file/symlink checks,
  no overwrite, atomic `WriteNewFile`, and rollback on failed load.

## Samples

- `native/examples/plugins/ini-outline.json`
- `native/examples/plugins/config-headings.json`
- `native/examples/plugins/README.md` documents schema2 fields, source-only
  behavior, literal line semantics, and the explicit no-multiline-lexical-state
  limitation.

## Validation

From `native`:

```powershell
F:\AIproject\LevNote\native\tools\go\bin\go.exe test ./core/plugins -count=1 -timeout=180s
F:\AIproject\LevNote\native\tools\go\bin\go.exe vet ./core/plugins
```

Results:

```text
ok  	levnote.local/native/core/plugins	3.998s
```

`go vet ./core/plugins` completed with no diagnostics. The latest rerun after
the sample-README precision correction also completed without diagnostics;
`git diff --check` on the owned paths produced no output. Both commands are the
focused owned-package checks; no broader test/build claim is made.

The core/plugins schema2 loader is now frozen for DeepSeek's app tests and the
independent read-only scanner/provider/app review.
