# Image host integration independent review

ACTIVE 2026-11-10 for the GLM composition owner after its source freeze.
Read-only review only: no product source edits, no test/build/launch/resource run,
no child delegation. Write findings/checkpoint in
native/handoffs/native-image-app-independent-review.md and message root. Astra is
the sole app owner and is adding tests; source is WIP, not yet accepted. Read its
checkpoint and the app integration task. Do not report every still-unwired branch
as a final defect; distinguish concrete implemented-path issues from pending work.

Review binary.go, binary_lane.go, binary_actions.go, app.go, state.go, plugins.go,
view.go, workers.go and affected command paths against the accepted document/
session/view contracts. Focus on one lane and memory ownership (including queues
and result slots), modal-delayed callbacks, canceled publication, explicit copy
outcomes, no live BinaryFile mutations on workers, duplicate opens, external
change, inactive release, startup restoration order and nil Doc/readonly gates.
Trace actual callers, not just individual functions. Cite exact trigger, path and
likely consequence for actionable findings; no hypothetical blanket warnings.

Root noticed a potential ownership mismatch to investigate: during an explicit
open/reload, existing active snapshot/bitmap may remain held while a new full
source is loaded into the result slot. The task's payload model is one active
source plus in-flight canceled work. Assess actual worst live references and
whether normal A->B opens introduce an undisclosed second noncanceled snapshot.
Reload must preserve original on failure; distinguish necessary transient buffers
from retained caches and propose the smallest truthful policy/implementation.

The current synchronous startup Restore intentionally blocks startup while
bounded reads run sequentially; this needs disclosure, not an automatic async
rewrite. Check that no partial persistence or draft loss occurs. No new generic
framework or added heavy validation run. Publish a first bounded review now;
update only for new frozen source when root explicitly requests final follow-up.
