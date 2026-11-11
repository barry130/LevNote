# UTF-8 save allocation implementation

2026-11-09. Active for GLM
`read_native_handoffs_release10_docs_task_md`. Root has accepted `.14` and the
bounded planning handoff `utf8-save-resource-plan.md`. Read `native/AGENTS.md`
and that plan if the task payload is opaque. Root coordinates/reviews only; no
children. Implement, test and benchmark only the scoped `native/core/document`
files. Do not build/package/launch, edit product docs, or touch `native/app` or
other packages except where the focused Go benchmark must live in the same
document package.

Goal: reduce full-buffer allocations in dirty UTF-8/BOM Save and SaveAs while
preserving the existing save contract byte-for-byte. Use a shared bounded
temp-file writer seam in `core/document` as planned. The writer must stream the
optional UTF-8 BOM and `Buffer.Snapshot().WriteTo` through a bounded buffered
writer, hash the exact bytes at that writer boundary, flush/sync/close before
conflict check and atomic publication, remove temp files on every failure, and
leave final `didSave` readback verification authoritative. Keep untouched saves
without serialization/timestamp changes. Keep UTF-16/GBK/GB18030 on their
existing encode path; do not change `WriteNewFile` in this stage.

Required implementation checks:

1. Preserve existing `Save`, `SaveAs`, `writeAtomic`, platform publishing,
   permissions, exclusive target creation, conflict/readonly/composition errors,
   dirty cache, undo/redo, selection, baseline and error-detail semantics.
2. Add exact-byte tests for UTF-8 with/without BOM, LF/CRLF, Unicode and
   4KiB/chunk boundaries, multiple Snapshot pieces, reopen, temp cleanup and
   unchanged timestamp for no-op saves. Add regression coverage proving legacy
   encodings and unrepresentable-character errors remain unchanged.
3. Add failure tests for writer/hash failure or target replacement/removal during
   save where practical using existing hooks; original target must remain intact
   and temp files must be removed.
4. Add one bounded benchmark comparing current full-buffer `Buffer.String()+encode`
   output path with the streaming UTF-8 path for deterministic local 1MiB and
   8MiB fixtures. Record fixture hashes, raw wrapper command/output, allocs and
   B/op only. Do not claim RSS, working-set, private bytes, app speed or memory
   reduction from benchmark values.

Run gofmt on changed Go files. Use only approved commands through
`native/scripts/go.ps1`: focused document tests, existing affected document
suites, benchmark, then `go vet ./core/document` (or narrower affected package).
Capture raw logs with new filenames under
`native/handoffs/utf8-save-evidence/`. Publish:
`native/handoffs/utf8-save-implementation-result.md` with exact changed-file
hashes, commands/exits, tests, benchmark fixture hashes and B/op/alloc results,
limits, and any remaining concern. Freeze source after publishing and notify
root; root will independently review and accept before any package/docs stage.

Do not weaken tests or silently alter error behavior to make allocation numbers
look better. If a writer failure seam is not safely injectable without changing
public behavior, document that limit and test the existing atomic failure hooks
instead.

Root clarifications to the planning handoff (authoritative):

- `.14` is now accepted, not `.13`. A bufio.Writer has no digest of its own;
  explicitly hash accepted bytes with SHA-256. Put the bounded buffer before
  the hashing destination so Snapshot.WriteTo's io.WriteString uses the buffer's
  StringWriter path without converting every chunk into a fresh byte slice.
- Prefer factoring existing writeAtomic through one callback-based temp lifecycle
  rather than copying that lifecycle. Its []byte callers keep identical behavior.
  No Windows/Unix publishing edit needed unless an actual defect demands it.
- Successful save must advance its baseline and clear dirty as before, while
  preserving undo/redo/selection. Snapshot descriptors can remain as the saved
  baseline; do not claim all descriptors die when the save call returns.
- Before publication, refusal/write errors preserve the original target (or an
  external writer's current target). After publication, verification can fail
  and cannot promise rollback. Preserve current verification semantics and do
  not invent a rollback over external content. SHA256.Write cannot normally fail;
  test injected sink errors/short writes and digest mismatch meaningfully, no
  invented hash failure framework.
- The benchmark compares equivalent output work (same snapshot capture policy,
  same digest, same writer/flush boundary) with setup outside timed iterations.
  Use one bounded fixed iteration count, e.g.10x, and modest repetitions only
  if variance changes the allocation conclusion. Preserve honest ns/op output
  but do not infer UI speed from it. No full application resource experiment.

## Root in-progress review finding — required correction before freeze

Read at22:46: current draft `writeUTF8Atomic` creates a hash but discards it,
and draft `didSave()` deletes comparison against intended output. This violates
the task: stable readback alone only tells us what is now on disk, not whether
it equals our bytes. An external write immediately after publication could be
silently marked as our saved baseline. Do not keep that change.

Return the exact encoded SHA-256 from the streaming helper on successful atomic
publication, pass it to `didSave` (e.g. expected `[sha256.Size]byte`), and require
readback hash equality before ANY saved baseline/dirty state update. Legacy
path passes `sha256.Sum256(encodedBytes)` to the same verifier. No fake
`_ = sha256.New()`, nil sentinel or skipped verification. Snapshot must be the
source of actual written bytes. A direct wrong-expected-digest verifier test
must return ErrConflict without advancing baseline, dirty or snapshot state.
Preserve existing post-publication failure behavior, no rollback claim.

After completing source corrections, one final affected suite should include
`./core/document ./core/session ./app` and vet those packages, because Save is
used by lifecycle/recovery/format draft actions. No full unrelated package or
managed-engine test rerun. Source implementation ownership stays document-only.

Root review at23:05 of still-uncompiled draft tests: preserve undo on Save, so
`doc.Dirty() || doc.Buffer.CanUndo()` is an invalid failure condition (CanUndo
must remain true after an edit/save). Snapshot contains slice fields and cannot
be compared with `!=`; compare intended contents/identity with existing public
methods and token/metadata. In injected writer test, errors.Is(err,sentinel)
should establish preserved error, not fail. Short-write test must drive the
real bounded write path so n<len with nil becomes io.ErrShortWrite; ignoring
returned n in a callback and writing nothing to the provided temp destination
does not test streaming behavior. Factor a small actual snapshot-to-writer
helper for meaningful sink tests/benchmark if needed, keep atomic lifecycle
single. Address source return signature and SaveAs lexical bytes scope before
running the focused suite. Do not change product semantics to meet wrong tests.
