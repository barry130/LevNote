# Drawio route late-result test correction

ACTIVE 2026-11-10 for implementation owner
`/root/read_native_handoffs_resource_workload_next_task_md`. This takes priority
over the read-only warning plan; finish any short plan handoff then correct.
No children, no production edits or new feature work. Only drawio_route_test.go
and fresh test evidence/handoff. Preserve original implementation/review/logs.

Independent review `drawio-route-independent-review.md` identifies two real
test gaps in TestDrawioRouteSameIDRefreshRejectsLateExport. Fix both:

1. After refresh assert requireDrawioRoute and the actually updated plugin Name,
   so nil/failed rematch cannot pass the replacement assertion.
2. Set f.ignoreCancellation=true before export. After refreshing/canceling,
   send success, await actual completion, assert response channel consumed
   (with synchronization from job.done), payload/warnings cleared, no dialog,
   no output and released job. Keep lane cleanup joined on success/failure.

Run one focused corrected test, then one final app suite and app vet. Do not
repeat plugins/modules tests, examples validation, graph tests or captures.
Use wrapper and absolute evidence paths. Preserve any failures.
Publish `drawio-route-test-correction-result.md` including final test hash,
raw commands/logs/exits and claim correction. Freeze. A reviewer will append
resolution; no `.20` delivery yet.
