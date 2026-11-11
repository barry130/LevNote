# Session output allocation: bounded implementation plan

2026-11-09. Read-only planning result. No source, tests, builds, benchmarks,
launches, product docs, or measurements were run. UTF-8 save source and the
`.14` package remain frozen.

## Recommendation

Take one slice after `.15`: write session JSON directly into the temporary file
owned by the existing private-state atomic writer. Keep the current snapshot,
matching buffer, and atomic lifecycle byte-oriented:

1. Refactor `core/document/state.go` so `WritePrivateState` delegates to a small
   callback writer API, tentatively
   `WritePrivateStateTo(path string, write func(io.Writer) error) error`.
   It must keep `MkdirAll(..., 0700)`, create/replace selection, `0600` mode,
   temp sync/close, publication, directory sync, and failure cleanup identical.
2. In `core/session`, use the exact length already returned by
   `preflightSnapshot` as an output postcondition. Encode the validated snapshot
   with the standard `json.Encoder` through a tiny trailing-newline-dropping
   writer that writes to the atomic temp writer. Verify the emitted length
   equals the preflight length and fails before publication on any mismatch.
3. Keep the existing `Snapshot`, `Tab`, JSON field order/tags, `encoding/json`
   escaping, and `WritePrivateState` behavior. `app/state.go` continues to call
   `SnapshotBuilder.Save`; it needs no change.

This is one change: “session JSON writes directly to the existing atomic temp
file.” It avoids adding a recovery format, cache, worker, dependency, or second
persistence path.

## Confirmed allocation finding

The bundled Go 1.27 implementation shows:

- `json.Marshal` builds JSON in a pooled `encodeState` buffer, then explicitly
  copies `e.Bytes()` into a second returned `[]byte`: `encode.go:204-215`.
  During session Save those two encoded buffers can coexist up to the 16 MiB
  cap. The returned copy is currently passed to `WritePrivateState`, written to
  the temp file, then released.
- `json.Encoder.Encode` does **not** stream individual fields or tokens. It
  builds the complete value in the same kind of `encodeState`, appends a newline,
  and calls the destination writer: `stream.go:216-245`. Therefore it does not
  remove encoder growth or the one complete internal encode buffer.
- Directing that one completed encoder buffer to the existing temp writer does
  remove `json.Marshal`'s extra full returned copy. With a 16 MiB session cap,
  this is a bounded potential transient reduction of roughly one encoded
  session buffer. No RSS/working-set result follows from source inspection.

Dirty capture separately calls `Buffer.String()` after preflight
(`builder.go:AddDocumentWithOptions`). That materializes one immutable draft
string retained through the public `Tab.Draft *string`. Removing it would change
the public Snapshot/Tab ownership model and is outside this slice. The
per-save `Snapshot()` tab clone copies descriptors/pointers, not draft bytes, so
optimizing it is not worthwhile here.

## Rejected alternative

Do not add a handwritten exact-size encoder in this stage. It could allocate one
exactly sized output slice and append fields into it, but it would duplicate
`encoding/json` rules for field order, `omitempty`, float32 formatting, HTML
escaping, controls, Unicode/U+2028/U+2029, nil versus empty slices, and future
public struct fields. Existing length accounting already has a substantial
oracle suite; making it also the sole production encoder adds too much
maintenance and compatibility risk for one bounded allocation improvement.

Using `json.Marshal` plus a pre-sized destination is also rejected because the
Marshal result and destination would still coexist. `json.Encoder` into a local
full buffer is rejected because it adds the same second buffer.

## Exact source and API scope

Production:

- `native/core/document/state.go`: add the private-state callback writer seam;
  make the current `[]byte` API a thin delegate with short-write detection.
- `native/core/session/session.go`: replace `json.Marshal` in `Save` with the
  bounded output helper using the preflight size.
- Add one small `native/core/session/output.go` for the newline-dropping exact
  length writer and Encoder call.

Tests/benchmark:

- Add one focused session output test file.
- Add the callback-writer short-write/cleanup case beside private-state tests.
- Add one bounded session output benchmark file.

Do not edit `builder.go` draft ownership, `budget.go` field accounting, app
persistence behavior, the recovery format, or unrelated packages unless a
focused regression proves a directly required correction.

## Output and failure contract

- Validation remains entirely before temp-file creation: `Save` sets Version,
  calls `preflightSnapshot`, and only then invokes `WritePrivateStateTo`.
- The Encoder default must match `json.Marshal`: HTML escaping enabled,
  no indentation, struct declaration order, and no extra bytes in the committed
  file. Its mandatory final newline is removed by the wrapper.
- The wrapper should delay the final byte rather than assume one destination
  `Write`; it commits the previous pending byte on the next write and drops the
  final `\n` at finish. It must preserve short-write/error propagation.
- The wrapper counts committed JSON bytes, rejects output beyond the preflight
  size, and at finish requires both `pending == '\n'` and committed count equal
  to the preflight size. A mismatch returns an error, so the atomic temp file is
  removed and the existing recovery file is untouched.
- `WritePrivateState` keeps its current behavior through delegation; existing
  private-state callers do not move onto Encoder.
- Any encoding/write/sync/check failure before publication preserves the old
  recovery file. Publication remains the commit point; post-publication
  directory-sync failure keeps the existing actual-state semantics.
- Draft validation, aggregate/per-draft/tab limits, rejection text, explicit
  retry behavior, and draft retention remain unchanged.

## Byte oracle and focused regressions

For every valid table case, set `snapshot.Version = Version`, compare
`json.Marshal(snapshot)` with the new helper's captured bytes byte-for-byte, and
confirm no trailing newline. Include:

- `Snapshot{}` with nil tabs versus `Tabs: []Tab{}` so `null` and `[]` remain
  distinct; field order and all omitted/present fields.
- `Preferences.LineNumbers` nil, explicit true, and explicit false; zero and
  populated `FontSize`, `SidebarWidth`, `Theme`, and `WordWrap`.
- float32 zero, `-0`, smallest nonzero, `1e-8`, `1e-6`, `1e20`, `1e21`, and
  `math.MaxFloat32`, matching the existing preflight oracle.
- strings with quotes, backslashes, every JSON control escape, other `<0x20`,
  `<>&`, CRLF/LF, Chinese, emoji/surrogate pair, U+2028/U+2029, and metadata
  paths/titles. Invalid UTF-8 and NaN/Inf remain rejected before output.
- one and multiple tabs, saved-only tabs, dirty drafts, `OriginalPath`,
  `PluginID`, `View`, BOM/encoding, selection, and scroll.
- representative values at the current 100-tab, 4 MiB draft, and 16 MiB
  encoded-session boundaries, plus one just-over rejection proving the old
  recovery remains byte-identical and no temp file is created.

Failure tests:

- The callback private-state writer receives `io.ErrShortWrite` and an injected
  sentinel after writing to the real temp destination; original target remains
  intact and the temp file is removed.
- An intentionally wrong expected output length fails before publication.
- Existing Save/Load, malformed-tab, omitted-draft, and explicit persistence
  regressions remain green; one successful round trip must load the newly
  written file and preserve Unicode/escapes exactly.

## Bounded benchmark after implementation

Measure output encoding only, not full Save, disk, RSS, app speed, or GC:

- Build two deterministic valid fixtures locally: one 1 MiB dirty draft and a
  multi-tab aggregate near 1 MiB under the 4 MiB/16 MiB/100-tab caps.
- Compare current `json.Marshal(snapshot)` with the Encoder-to-`io.Discard`
  helper after preflight. Setup, snapshot construction, oracle generation, and
  fixture hashing stay outside timed loops.
- Run through `native/scripts/go.ps1` with `b.N`, `b.ReportAllocs`, and a fixed
  `-benchtime=10x`; record fixture bytes/SHA-256 and raw command/output.
- Report only B/op and allocs/op at this output boundary. Preserve ns/op in raw
  output if emitted but make no application-level speed claim.

## Acceptance

The slice is accepted only if the implementation preserves exact committed
bytes, all validation-before-write and rejection behavior, one atomic private
state write, and existing draft/file ownership. Run focused output/state tests,
one final affected `./core/document ./core/session ./app` suite and vet after
corrections, plus the single bounded benchmark above. Do not build, launch, or
measure process memory as part of this stage.
