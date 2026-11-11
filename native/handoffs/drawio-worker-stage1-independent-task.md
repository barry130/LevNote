# Fourth worker stage1 independent review

ACTIVE after completed frozen stage1-result.md,2026-11-10. Reviewer GLM
`/root/read_native_handoffs_drawio_worker_next_plan_task_md`. Root review
and independent review are separate from implementation; no child delegation.

Read active implementation task, original/refined plan, frozen result and actual
changed files/logs. No product edits/build/launch or repeat passing suites.
Write `drawio-worker-stage1-independent-review.md` with concrete findings,
locations/triggers and minimal repairs. No speculative checklist or cosmetic
style findings. Review all final source hashes and factual scope; original
.18/source-oracle records immutable.

Prioritize:

- old3 wire bytes unchanged; duplicate/unknown/null/type/UTF8/index/ID bounds
  enforced in encoder and strict decoder, request meta size capped; invalid
  metadata must not read source or launch. Well-formed absent/mismatched page
  fails invalid-page with nil payload, never falls back to page0 or ID search.
- Source.Page copied at runtime admission BEFORE hooks/async work, app-owned
  queue copy deferred to stage2. Source.Size/Data/cleanup/poison behavior same.
- Worker exact second page SVG vs direct API, expected issue scope/order and
  upper bounds, readonly/compressed export, unchanged input and bounded errors.
  Parse -1 denotes mxfile-wide only; no arbitrary relabeling/dropping. Bound
  warning raw strings BEFORE concatenation, no truncation. Empty warnings[] on
  protocol outputs; failed output is never publishable. Decode to EOF.
- schema1 requires exactly old3 IDs regardless of order, migrates in memory with
  fourth bundled and no file write; schema2 exactly4; installed IDs/digests and
  explicit reset/corrupt-state/12version retention/atomic persistence preserved.
- fourth catalog/script/build definition agrees, no new source executable/path
  override or sandbox claims. Existing installed3 manifests continue supported.
- test fixtures independent enough to catch wrong selected page, aliasing,
  load mutation/old state loss, output/warning limits. Retain real failed
  compile and other attempts; final tests tied to actual final hashes. No app
  undo/BOM/memory/physical-UI claim from worker-only tests.

No giant test matrices. Request one narrow reproduction only if source/evidence
can't resolve a substantive concern. Any authorized repro uses wrapper and a
fresh review evidence file, never modifies completed original evidence.
