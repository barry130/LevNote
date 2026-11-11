# Session output: bounded review follow-up

2026-11-10. ACTIVE after independent review confirms the bounded gaps;
owner DeepSeek read_native_handoffs_release10_ui_feasibility_task_md. No children.
Keep original scope, no build/docs/launch, immutable .15. Preserve existing
evidence, make new filenames for actual subsequent commands and failures.

Root reviewed current files and requires these narrow verification corrections:

1. Exercise the actual composition of writeSnapshotJSON -> injected destination
   -> real temp writer inside document.WritePrivateStateTo. Forward a nonzero
   prefix to the provided temp file, then report either n<len/nil or an injected
   sentinel; require exact ErrShortWrite/sentinel, old recovery byte/time
   preservation and no leaked temp. Current callback just writes a prefix then
   manually returns ErrShortWrite, which only covers callback cleanup. Keep that
   honest; use real pipeline for added coverage, no artificial hash framework.
2. Outside benchmark timing, encode each actual benchmark snapshot through the
   new helper and compare byte-for-byte with the Marshal oracle. Current fixture
   hash is old output only. Keep same fixed10x benchmark, setup excluded, pooled
   encoder caveat and actual ns/op logs; rerun once because benchmark setup is
   changed, not for nicer numbers.
3. Fill real byte-oracle coverage gaps with a saved-only tab (Draft nil, valid
   absolute Path) and explicit true LineNumbers. Current new table labels a
   draft-bearing tab "saved" and handoff overstates that. Keep existing valid
   float/Unicode/bounds tests; no redundant matrix.
4. Independent reviewer confirms requirement for sticky adapter errors.
   Root observes one valid output Write followed by excess Write can return an
   error then Finish nil. Current production Encoder returns immediately so
   this alone is not demonstrated data loss. Small helper-level sticky error
   retention is acceptable if reviewed: fail subsequent Write/Finish with original
   error, no second pending-byte emission after a sink failure. Do not add broad
   retries/concurrency/serialization machinery.

Reviewer concluded no demonstrated production correctness defect in current
Encoder path. Implement sticky errors as bounded helper-contract hardening,
not a claimed data-loss fix. Original review is session-output-independent-review.md.

After scoped corrections, focused meaningful tests, one final affected
document/session/app suite+vet; raw final logs should contain exact command,
output and exit. Update result final source/evidence hashes and accurately
distinguish previous callback-only tests, benchmark observations and final
pipeline coverage. Freeze and ask independent reviewer to reconcile only new
files/evidence. Root then accepts source before .16 packaging.
