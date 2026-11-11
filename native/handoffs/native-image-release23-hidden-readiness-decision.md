# .23 hidden lifecycle acceptance decision

ACTIVE2026-11-10 root decision following Astra's read-only source trace.
This narrowly overrides preparation review item2 and all mandatory pre-close
session-readiness language. No product/runtime/vendor source is reopened.

Hidden MyGo Surface draws its first frame but does not reliably redraw after
later Update callbacks. App automatic persistence runs in View, while CLI binary
adoption can complete independently through updateWait(deliverBinary). Therefore
a session-before-close poll can time out despite successful app state adoption.
Run2's old session proves only the preserved baseline, not current restoration.

Use the existing held-process/window/Job lifecycle with a bounded5000ms settling
period after finding the exact hidden window. Check the owned process remains
alive during this period. Record duration and completion honestly as settling,
not semantic readiness. Optionally read pre-close session once descriptively;
its absence or old shape is not a failure and must not postpone the single close.
No paint/input automation, second close, probe API or product logging is added.

After that period post exactly one owned WM_CLOSE. Require normal exit0, no
forced cleanup, zero remaining owned processes/worker requests and a fresh
post-close session publication after this process start relative to its captured
pre-run state. Require exact ordered text+binary records, active1, original
encoding/BOM and absent binary/text payload leakage for each run. Preserve
run2's byte/time-identical baseline copy with the correct DateTime tick epoch.
If the single close is refused or times out, preserve that product failure and
stop; do not retry it silently. This is startup/close/restore evidence only.

Preparation owner: update run-host-smoke and aggregate together, using nullable
descriptive pre-close facts and mandatory settling/post-close facts. Update
checkpoint/result planned commands to the copied artifacts directory. Finish
one static parse of final helpers and report frozen identities; execution remains
inactive until root accepts this bounded correction. Do not add another
preparation phase or general retry mechanism.
