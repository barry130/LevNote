# Independent review: prepared view export contract

ACTIVE for GLM `/root/read_native_handoffs_drawio_worker_next_plan_task_md`
after finishing initial stage1 review. Both layers are frozen except the
separate worker-warning correction assigned in native/AGENTS.md. No product
edits, execution/build/repeated gates, or child delegation.

Read view-contract-task/result and actual eight changed files. Review neutral
contract, exact current page index+ID, no fallback/no second Flush/no source
mutation, pre/post synchronization identities, monotonic epoch, lifecycle and
tests. Specifically inspect ensureSynced callback reentrancy: it reports canceled
gestures before using host.Buffer again; can a callback set buffer nil, replace
it, Release, or modify source and crash or lose a newly pending draft before
Prepare's after-check runs? Distinguish existing lifecycle behavior from a newly
exposed acceptance guarantee; only report concrete triggers. Root has verified
ordinary page/cache epoch code and read final test/vet logs; do not duplicate
passing suites. A callback that restores same buffer/version or pageA->B->A
must not silently retarget accepted page. Getter stays pure.

Check handoff coverage claims against actual tests, especially reentrant state
changes (test presently exercises one page change), draft restoration and
Release reuse. Missing redundant tests are not automatically defects; require
a concrete behavior or unproved material guarantee. Review source/error handling
without changing app/adapter advertisement. .18 remains immutable.

Write `drawio-worker-view-contract-independent-review.md` with concrete findings
or no-blocker conclusion and exact scoped identity. Retain original stage1 review
separately. When worker correction freezes, reconcile it in
`drawio-worker-stage1-warning-independent-reconciliation.md`; narrow source and
existing evidence only unless reproduction is required to decide a real issue.
