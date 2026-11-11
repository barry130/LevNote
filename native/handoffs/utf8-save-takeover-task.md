# Finish the bounded UTF8 save slice

2026-11-09. ACTIVE for DeepSeek
`read_native_handoffs_release10_ui_feasibility_task_md`. GLM writer has been
interrupted and has no ownership; do not resume it. Root remains planning/review
only. No children. Read utf8-save-implementation-task.md (including corrections)
and actual current core/document files. Own only core/document source/tests and
utf8-save-evidence/result. `.14` immutable and current; no builds/docs/launches.

The unfinished implementation exists but has not run any actual tests/benchmark
yet. Finish it, not redesign. Root saw several still-outstanding issues:

- atomic.go `writeUTF8Atomic` returns `(digest,error)` but still returns only
  `writeAtomicWriter(...)` error; fix return and ensure digest usable only on
  success. Remove redundant writeAtomicData trampoline if not useful.
- didSave expected-digest comparison has been restored; do not delete it.
- draft wrong-digest test compares dirty live Buffer with old saved snapshot,
  so assertion fails even for correct behavior. Compare saved state pre/post,
  token/disk metadata/cache flags without running Dirty at a point that mutates
  cache before your assertion; verify separately that source remains dirty.
- short/sink tests now drive streamUTF8 but should check exact ErrShortWrite/
  sentinel. Test both <32KiB flush failure and >buffer forwarding where useful.
  Use actual provided temp writer composed with injected sink to prove written
  partial temp is removed; do not invent hash-error framework.
- utf8_save_benchmark_test.go and utf8_save_test.go are drafts. Inspect carefully
  for equal output work, no setup in timed region, correct digest/fixture oracle,
  independent1/8MiB fixtures and fixed10x iterations. Benchmark may be output
  generation, not full Save; state exact boundary honestly.

Required behavior/tests remain in original task: UTF8/BOM exact bytes/chunks/
Unicode/line endings, undo/redo/selection/baseline, untouched save timestamp,
nonUTF8 old encode path, composition/readonly/errors, final conflict check,
exclusive creation and digest mismatch. Existing suites cover many; add only
missing substantive cases. Prefer separate new test file for additions.

Run approved Go wrapper now to get compile feedback and preserve actual errors;
do not spend another long design pass without executable progress. Then focused
fixes/tests, one bounded benchmark, final document/session/app test and vet.
No package/worker/physical UI repetition. Publish checkpoint promptly if blocked,
then complete frozen utf8-save-implementation-result.md with final hashes/raw
evidence. Root and independent reviewer will reconcile.

Root read benchmark draft: it currently loops literal `i<10` instead of b.N,
which makes Go benchmark denominators wrong. Use b.N and constrain invocation
with `-benchtime=10x`. bytes.Buffer sink grows/retains whole output; for narrow
output-generation allocation use io.Discard plus exact digest in timed path,
validate full bytes independently outside timing. Specify snapshot-capture
inclusion/exclusion identically. Fixture truncates a pattern byte arbitrarily,
which can yield invalid UTF8 and panic text.New; ensure exact target sizes via
ASCII padding or rune-safe construction. Report SHA256 of actual source fixture.
No benchmark evidence exists yet; draft is not valid measurement.
