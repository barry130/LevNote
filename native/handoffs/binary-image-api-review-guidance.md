# Binary prerequisite boundary notes

2026-11-10 root planning guidance for the active binary design owner.

The first code increment can stay entirely in core/document. No session/app/
plugins/view/vendor changes yet. Small binary baseline plus a separately owned
immutable snapshot; the baseline must not retain compressed bytes after caller
drops the snapshot. Reacquiring a snapshot must verify saved baseline, including
identity/hash, and refuse changed/missing source. Explicit Reload establishes a
new generation and invalidates old tokens/snapshots for state adoption.

Copying the already accepted active snapshot is allowed even if the original
later changed or disappeared: it copies the bytes the user had opened. Do not
silently substitute newly read source bytes. Caller must receive an explicit
result showing publication and whether new baseline/path was verified/adopted.
Same destination as current canonical path is nonoverwriting/no rewrite.

Review immutable API carefully: returning bytes.Reader as io.Reader still lets
callers type-assert its WriterTo, whose Write can expose the backing slice to a
custom writer. Public WriterTo must not hand private authoritative storage to
an untrusted writer; either omit it in the public reader shape or use bounded
scratch/copy. Decoder access should use a read-only reader/ReaderAt abstraction,
not duplicate the whole source solely to hide a slice. Tests should probe this
actual aliasing boundary where applicable. No security sandbox claim.

Publication and post-commit verification/durability failure are distinct. A
small private outcome-returning atomic helper may underlie existing wrappers,
which keep their existing signatures/error behavior. A failed post-commit step
must not pretend no destination exists, erase a committed file, or invent a
verified new baseline. Keep injected failure seams scoped; avoid global mutable
hook complexity or unrelated save refactoring. Final affected document suite
once after meaningful focused regressions; no package build for this foundation.
