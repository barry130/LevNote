# Prepare one real managed draw.io worker acceptance fixture

ACTIVE 2026-11-10 for DeepSeek
`/root/read_native_handoffs_release10_ui_feasibility_task_md` after warning
correction is frozen. Disjoint from graphview correction and independent review.
Root plans/reviews; no child delegation. No builds, tests or launches yet: this
prepares the real fixture for later .19 delivery, where execution is authorized.

Read `drawio-worker-release19-delivery-task.md`, existing Windows
core/engineruntime/managed_bundle_test.go and .11 managed smoke harness. Implement
one narrowly env-gated Windows test in new
`native/core/engineruntime/drawio_managed_bundle_test.go` for actual .19 drawio
execution. It must default Skip, never run ordinary suite children. Preserve
old smoke test and runtime production files. Don't modify completed evidence.

Fixture validates valid old3schema1 load without write and fourth bundled;
import exact chosen fresh candidate fourth engine without activation; explicit
Activate; one and only one real NewManaged Export with exact page1+ID; independent
literal SVG plus warning ordering, unchanged original XML; verified installed
version/hash/package digest and unchanged old3 selections. Source below1KiB is
enough. Reuse accepted main_test.go exact second-page fixture/oracle directly as
literal, never call format renderer for expected output. Record all facts and
Start/Wait/Close (and exact owned process handle/path/creation identity where
existing safe observation helpers allow) through private observation-only hooks.
Never override argv/executable/stdio/lifecycle, process-name-kill or synthesize
response. All process paths come from existing production package manager.

Require explicit env enable + absolute candidate release/evidence/config paths
with run-owned roots; test exe will later sit in evidence-owned bundle COPY,
candidate remains immutable. Ensure no cleanup erases evidence. Use existing
safe handle observation Windows primitives, keep handle closed and record close
errors. One Runner Export; cleanup fallback may Close same runner idempotently,
never export again to repair a failed run. JSON record should distinguish error
and absent observation; record failures before exiting. No test of huge input,
resources, UI export or physical input. Keep code small and not a generic harness.

Publish `drawio-worker-real-smoke-preparation-result.md` with design, exact test
path/hash, env contract, deferred command for wrapper test -c, and evidence naming.
Do not execute the future command now. Freeze. Root will review this test and
activate actual .19 delivery after accepted stage2/portable assets. No production
source/worker/parser/app/script/docs changes under this task.
