# Independent core-store review — source read only

2026-11-09. ACTIVE for DeepSeek read_native_handoffs_resource_workload_next_task_md.
Your resource plan is complete; root corrected lane/process wording and counts.
Do not edit it again. This is a new bounded read-only source review, no execution.

Read current package_*.go in native/core/engineruntime, excluding unfinished
tests unless helpful, existing bundle openLockedFile/platform helpers and
handoffs/fixed-engine-package-core-activation.md. Astra owns remaining store
implementation/test; GLM owns package_inventory.go/test. Do not edit their files,
run tests/builds, start processes or spawn agents. Context below200K.

Review only concrete correctness that can lose/change selected state, publish
wrong package bytes, leave owned resources improperly, make normal import fail
or hang, or block UI metadata reads. Root already read source and noted:
- single-manager serialization; multi-instance transactions not promised;
- staged exact cleanup/rename releases its handles before filesystem mutation,
  not an adversarial filesystem/sandbox guarantee;
- post-commit cleanup error may mean new state is already durable/live, so error
  cannot universally mean no commit;
- known path parser cases are actively being corrected by GLM.

Do not treat these disclosed boundaries as fresh defects or demand generalized
locking/signatures/security architecture. Need at most3 concrete findings with
file+function, triggering sequence and expected behavior; distinguish confirmed
from hypotheses and check the actual guard order. Report 'none found' if none,
without claiming tests or proof. Optional test coverage suggestion only when it
addresses actual unverified behavior in this slice. Limit output roughly1page
at fixed-engine-package-core-independent-review.md, then finish. Read current
source once; don't chase code changes indefinitely. No new gate for owners.
