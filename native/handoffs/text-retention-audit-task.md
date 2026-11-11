# Next finite stage: verify text ownership before optimizing

2026-11-09. Root accepts the UI helper failure as an environment limitation;
no native launch/input occurred, and no resource workload can be claimed.
DeepSeek read_native_handoffs_release10_ui_feasibility_task_md now owns this
disjoint core/text investigation. No more Computer Use retries in this stage.
GLM owns seven product docs for accepted .10; other DeepSeek is idle. No children;
compact before200K. All tests use scripts/go.ps1 -GoArguments only.

Read core/text/buffer.go, existing text tests/benchmarks and narrow callers for
string ownership. First establish concrete reachable-memory facts, not a claim
of process RSS or global leak. Scope two hypotheses from root source review:

1. replace() compacts out in place via merged := out[:0], then assigns b.pieces
without clearing the no-longer-logical tail. Are obsolete chunk pointers beyond
len(merged) retained by the live backing array? Can real supported edit sequences
leave materially many stale chunks after undo/history is released? Snapshots
that legitimately hold chunks must remain valid.
2. ReplaceUTF16 records change.next=value, while piecesOf independently clones
chunks. Can a small substring supplied by normal callers retain a much larger
source allocation through undo/redo, despite RetainedBytes counting only the
small logical string? Locate real call paths (editor/highlight/format patches),
don't assume all strings have independent backing. Existing old string comes
from SliceUTF16 builder; verify it separately.

Prepare deterministic focused regressions or a bounded test-owned diagnostic to
prove/disprove each hypothesis. Prefer actual retained pointer graph/ownership
or carefully scoped allocation facts over noisy global heap percentages.
No unsafe product code, no broad telemetry, no artificial UI hooks, no huge
fixtures, no time-based flaky GC assertions. Preserve pre-change actual failing
evidence and immutable .10 inputs. Ordinary diagnosis/test source may be added
under core/text; do not edit production until a concrete effect is confirmed.

When a concrete defect is proven, a minimal ownership fix in core/text/buffer.go
is already authorized: clear dead pointer slots before retaining a shortened
piece slice; independently own stored edit strings if needed to make retained
history bytes honest. Choose smallest correction justified by actual evidence.
Don't change undo limits, user-facing undo behavior, snapshots, groups or UI.
Do not add an unmeasured cache. Check bytes/source/Unicode/selection/undo/redo/
group cancel/snapshots using meaningful existing+focused regressions.

Publish findings and changed hashes promptly to handoffs/text-retention-result.md.
Run affected ./core/text tests + vet once after final correction; if product
changes, also existing dependent ./core/document ./ui/editor ./app checks once
to cover history/drafts integration (no unrelated worker/resource matrix).
Small existing benchmark comparison optional only if it helps assess extra
copy cost; distinguish allocations per operation from retained process memory.
No full release build until root accepts the exact correction and assigns it.

If neither hypothesis is materially real, publish negative findings and stop
without speculative changes. Accepted .10 remains runnable regardless. This
stage progresses the user's memory objective while true UI measurement is
blocked; it must not be presented as equivalent to the deferred C workload.
