# UTF-8 save resource plan

2026-11-09. Planning only, for a small implementation stage after the Find-keyboard
`.14` delivery; nothing in this plan is a run, baseline, measured comparison, or
acceptance. This stage follows the source-only review required by
`utf8-save-resource-plan-task.md`. It keeps `.14` source frozen and does not change
the current save contract. The active accepted executable remains the `.13` host
named in `native/AGENTS.md`; no package or process-memory claim is being made here.

## Confirmed whole-buffer work

Confirmed allocations in the current UTF-8 save path are:

1. `native/core/text/buffer.go:133` `Buffer.String` allocates a complete output
   `string` (`out.Grow(b.bytes)` and writes every chunk).
2. `native/core/document/encoding.go:124` `encode` allocates another complete
   output slice for UTF-8 (`[]byte(value)`; BOM is additionally prepended).
3. `native/core/text/buffer.go:162` `Buffer.Snapshot` copies the piece-descriptor
   slice (`append([]piece(nil), b.pieces...)`); it does not copy chunk strings.
   The descriptors live only through the save.

Unmeasured process hypotheses remain outside this stage: RSS, working set,
private bytes, pagefile/commit, UI/app speed, and GC cadence. Do not infer any of
them from allocations or B/op. Benchmarking is scoped to the exact save function
and its encoded-output allocations below.

## Smallest seam

Keep all existing gates, conflict checks, permissions, timestamps, no-op saves,
composition refusal, dirty/undo/selection/baseline, readonly `SaveAs`, and
encoding errors unchanged. Introduce one bounded UTF-8/BOM-only writer in
`core/document`, used by `Save` and `SaveAs` when `Encoding == UTF8`:

- take the already-owned `d.Buffer.Snapshot()`;
- write the optional UTF-8 BOM (`0xEF 0xBB 0xBF`) once, then
  `Snapshot.WriteTo` into a temporary file wrapped in a bounded
  `bufio.Writer` (the same shared chunk strings, no new full-text string or
  full encoded output slice);
- use the `bufio.Writer`'s accumulated digest as the encoded-hash input; do not
  create a second complete digest copy.

Legacy/UTF-16/GBK/GB18030 encodings keep the existing
`encode(Buffer.String(), ...)` path. `WriteNewFile` and the other atomic writers
are outside this stage.

## Shared atomic temp writer

Create one small helper, tentatively `writeAtomicWriter`, shared by `Save` and
`SaveAs`, with the same lifecycle as `writeAtomic`:

1. `os.CreateTemp(filepath.Dir(path), ".levnote-save-*")`.
2. `temp.Chmod(mode.Perm())`.
3. Caller-provided callback drives BOM and `Snapshot.WriteTo` into the wrapped
   destination; short or failed writes fail the callback.
4. `temp.Sync()`, then close.
5. If `check` was supplied, run it after the temp write succeeds and before
   publication.
6. `Save` publishes with `replaceExisting`; `SaveAs` publishes with
   `publishNew`.
7. On any failure, return the actual error and remove the temp file; the original
   target remains unchanged. On success, `syncDirectory(dir)` as today.

Do not change `publishNew` / `replaceExisting` platform behavior, checks, or
final verification. The `didSave` SHA-256 readback and `fingerprint` remain the
authoritative conflict check.

## Preserved save invariants

The implementation stage must verify, not weaken:

- final conflict checks and exclusive creation;
- same-mode (`os.FileMode`) for an existing target, `0600` for `SaveAs`;
- readonly status, composition (`Buffer.InGroup`) refusal, missing-file
  conflict, and `ErrExists` for an existing `SaveAs` target;
- no serializer run and no timestamp change for an untouched save;
- dirty/undo/redo/selection/baseline/session state unchanged by saving;
- unchanged `dirtyCached`/`dirtyVersion` handling on success or failure (reset
  cache only on success, matching the current `didSave` flow);
- errors preserve the actual path and OS detail; failures leave the original
  file byte-identical and remove the temp file;
- readonly `SaveAs` creates an editable new document with `0600`;
- `.8`-style `sameIdentity` disk token and conflict semantics, byte-identical
  same-byte replacement detection, and unchanged save/revert baseline behavior.

## Verification and benchmark scope

Use existing test style; do not introduce new frameworks. Keep
`TestEncodingRoundTripAndUntouchedSave`, `TestConflictIncludesSameTimestampAndMissingFile`,
`TestSaveAsNeverOverwritesAndNewEmptyIsUnsaved`, `TestWriteNewFileNeverOverwrites`,
`TestReadOnlyAndCompositionNeverWrite`,
`TestInvalidEncodingAndBinaryUseBoundedInspector`,
`TestAtomicFailureLeavesOriginalAndNoTemporaryFiles`, and the disk-token suite
green, and add these focused cases in `core/document`:

1. UTF-8 save with and without BOM, CRLF and LF endings, Unicode including
   surrogate-pair and chunk-boundary content: disk bytes, digest, timestamps,
   dirty/undo/selection/baseline, reopen round-trip, temp cleanup.
2. Chunk-boundary (and a ~4 KiB) test that forces multiple `Snapshot.WriteTo`
   pieces through the buffered temp writer; compare against `encode` byte-for-byte.
3. UTF-16/GBK/GB18030 save regression keeps existing byte-for-byte output and
   `ErrUnrepresentableEncoding` behavior.
4. Failure cases: target file replaced/removed during save, composition refusal,
   readonly `Save`, existing `SaveAs` target, hash mismatch on readback,
   writer/hash failure; original stays byte-identical, temp removed, error detail
   preserved.

Benchmark boundary (one function only):

- independently generate two modest deterministic fixtures locally, about 1 MiB
  and 8 MiB, no private/user data, with mixed ASCII/CJK/emoji and enough chunks to
  cross 4 KiB boundaries;
- run the new function and the current full-buffer path (`Buffer.String` +
  `encode`) against the same temp-file/digest writer boundary under the same Go
  version and `testing.B` conditions, to report encoded-output allocations and
  B/op; keep the same API used by `testing.AllocsPerRun`/`testing.Benchmark`
  per fixture size;
- report allocations and B/op only for the save function and its encoded-output
  allocations; do not infer RSS, working set, private bytes, pagefile/commit,
  UI/app speed, or GC cadence from these values;
- record fixture hashes, the exact benchmark command through the approved Go
  wrapper, and raw output in the result handoff.

## Ownership and acceptance checks

Owner: one implementer assigned by root after the `.14` delivery boundary. Reviewer
and acceptor: root. Source files are `core/document/atomic.go`,
`core/document/atomic_windows.go`, `core/document/atomic_unix.go`,
`core/document/document.go`, `core/document/encoding.go`, plus focused tests in
the same package. Do not touch `native/app` or any other package.

Acceptance requires:

1. focused document tests pass;
2. the existing affected suites listed above pass;
3. the benchmark evidence exists with both fixtures' hashes, both paths'
   allocations/B/op, raw output, and no process-memory claim;
4. the result handoff states the UTF-8-only scope and unchanged save semantics;
5. root acceptance before any delivery or docs stage.

Do not claim a physical memory improvement from these values; state the honest
scope as fewer save-path allocations for large UTF-8/BOM saves, with actual
process memory as a separately measured topic if root activates it later.
