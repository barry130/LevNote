# Root review — resource workload planning

2026-11-09. Accept as queued planning only, not execution. Root read actual
plan and relevant buffer/lane lifecycle. Corrected two planning inaccuracies:
the lane is a host goroutine, children are one-shot and join after each settled
operation;4 measured runs +2pilot runs =6total. Existing .8 has no matching
interactive measurements, no claim added.

Use only C repeated lifecycle first after a visible interaction feasibility
gate and accepted candidate. Long edit/undo B requires proving actual public UI
transaction boundaries (100 typed characters may coalesce); don't assume its
wording makes undo count true. This is deliberately future work, no need add a
test-only product interface to obtain private history. Source/disk correctness
and stable observable UI steps should precede resource sampling. If screen/UI
automation cannot expose a claimed state, narrow and describe the measurement.

Hypotheses (per-tab histories, repeated outline reconstruction, snapshots until
join) are reasonable priorities for measurement, not confirmed memory defects.
No need to grow the run matrix or duplicate accepted worker microbenchmarks.
