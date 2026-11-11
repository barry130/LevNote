# Independent neutral scene source review

2026-11-09. GLM may read the now-existing core/enginescene scene.go, encode.go
and decode.go under the corrected dto-contract/root next-stage plan. Write only
native/handoffs/native-preview-dto-independent-review.md. No edits to product,
tests, plans, evidence; no tests/builds/launches; no child delegation. Astra
retains sole implementation/execution ownership and is adding meaningful tests.

Review finite source for concrete correctness/resource-boundary defects: exact
JSON syntax/key/missing/null/trailing handling, UTF-8/UTF16 surrogate refusal,
string and count admission before retention, arbitrary key order, exact integral
counts despite float rounding, bounded long-number handling and finite geometry,
rooted-tree invariants/iterative validation, deterministic count/fill escaping,
no partial return/input alias. Do not require a generic framework or speculate
about hostile worker sandboxing. API is pure and caller must not mutate Scene
concurrently with Encode; that is not a new concurrency defect.

Give smallest concrete offending input/expected-versus-actual mechanism and
source lines for actionable findings. Read actual stdlib when claims depend on
its behavior; don't infer them from names. If tests exist by review completion,
inspect independent oracle quality without running them. Mention any unreviewed
later files honestly. Report no actionable findings if that is the result.
