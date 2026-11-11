# Engine app lifecycle review

2026-11-09. Read-only source review only: no test, build, vet, process launch,
or product/test edit was performed. No PASS is claimed.

Sources read: `native/app/engine_packages_lifecycle_test.go`,
`native/app/engine_packages.go`, `native/app/engine_lane.go`,
`native/app/export_worker.go`, `native/app/preview_worker.go`,
`native/app/export_worker_test.go`, `native/app/preview_worker_test.go`, and
the lifecycle portions of `native/app/app.go`,
`native/app/workspace.go`, `native/app/state.go`, and
`native/modules/graphview/scene_preview.go`.

## Findings

### 1. P1: close-refusal assertion is scheduling-dependent

`native/app/engine_packages_lifecycle_test.go:227-243` starts a real package
import and then requires `a.packageJob == job` after `CanClose()` returns.
However, `CanClose` defers `a.deferExportDelivery()()`
(`native/app/app.go:939-950`), and the deferred unwind calls
`deliverPackageWork()` at `native/app/export_worker.go:350-352`. The import can
finish while `CanClose` is persisting the session. In that case the deferred
delivery clears `a.packageJob`, and line 242 fails even though the persistence
refusal correctly stayed before `closeEngineExport`.

GLM repair: make the package result delivery intentionally deferred during this
assertion. For example, increment `a.modalDepth` before calling `CanClose`;
the nested `CanClose` defer will return it to the pre-existing nonzero depth,
so the same job is guaranteed to remain in `a.packageJob`. Then assert that
identity, explicitly unwind the test-held modal depth, and drain/deliver the
job in order. This tests the refusal ordering without depending on import I/O
timing.

### 2. P2: poison test does not test its stated post-delivery invariants

`native/app/engine_packages_lifecycle_test.go:129-142` proves that a poisoned
transition is refused and that list is admitted, but ends immediately after
`a.deliverPackageWork()` with no assertion. It therefore does not prove that
list delivery clears the job or that `enginePoison` remains visible and still
blocks the next transition. A regression that clears poison during list
delivery would pass.

GLM repair: retain the poison in a local sentinel, then after delivery assert
that `a.packageJob == nil`, `errors.Is(a.enginePoison, poison)` is true, and
the delivered list state is present (`checked`, expected empty receipts, nil
error). Finally retry `packageActivateAction` and assert that it is still
refused with no job.

### 3. P2: the “stale generation” intermediate assertion is vacuous

At `native/app/engine_packages_lifecycle_test.go:147-162`, `modalDepth` is
still one when the test increments `packageGeneration` and calls
`deliverPackageWork`. `native/app/engine_packages.go:205-207` returns at the
modal guard, so generation validation at line 227 is never reached. Lines
158-162 would pass if generation checking were removed; only the later call
after decrementing `modalDepth` exercises it.

GLM repair: decrement `modalDepth` and bump `packageGeneration` before the
generation-specific delivery assertion, then verify the job slot is cleared
without importing data. Alternatively, remove the intermediate generation
claim and leave generation coverage solely in the post-unwind call.

## Conclusion

No source-proven compile failure was found, and the reviewed controlled-runner
helpers provide bounded waits and do not establish a lifecycle deadlock from
source inspection. The findings above are the concrete GLM-owned test fixes;
no product defect or data/draft-loss defect was established in this pass.
