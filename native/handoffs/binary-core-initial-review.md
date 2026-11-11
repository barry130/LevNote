# Initial binary WIP review

2026-11-10 root read the active draft, not frozen acceptance. Please address
these concrete issues before final checks. Focused execution window is open.

1. Remove the unrequested Changed/channel/Close machinery. BinaryFile owns no
   open OS handles or bytes; a single-goroutine baseline requires no channel or
   sync.Once. Current publishChange receives from an open unsignaled channel,
   so Reload/SaveAs deadlock. Host will own async generations. Keep small token
   authority only, and document baseline invalidation. No new general framework.
2. binary.go currently includes an orphaned second SaveAs tail and duplicated
   Token/Reader/readStable functions after the first readStableContext. Clean
   this into one implementation before gofmt/compile. binary_test.go lacks io,
   has placeholder imports/init and wrong atomic helper argument list.
3. SaveAs ctx currently checks only at entry; snapshot.WriteTo does not observe
   it and atomic check=nil. Use private context-aware chunk streaming and check
   ctx immediately before publication; public io.WriterTo can remain ordinary.
   Cancellation after commit must report Committed and retain destination.
4. Stable reads/digest verification must verify actual total bytes equals the
   accepted before/after size; no known mismatch may establish a baseline.
   Check cancellation at final boundary as well. Use small per-call read seam
   for substantive short-read/mutation/cancellation cases, no global hook.
5. Tests need real expectations: after SaveAs increments path/revision, the old
   immutable snapshot is readable but its adoption token is stale. Current test
   expects SameBaseline(old)==true and then reuses it for another SaveAs. Capture
   the new snapshot explicitly for follow-on cases; don't change token rules
   just to satisfy this assumption. Consider small no-copy retoken result only
   if the eventual host needs it; publish exact API choice in handoff.
6. Exclusive race test precreates destination so beforeWrite never runs; create
   it inside the hook, and assert the hook was reached, existing bytes retained.
   Oversize fixture must create file before Truncate. Add successful empty file,
   midcopy cancellation, source deleted but accepted-copy succeeds, stale token
   rejected before destination creation, same-size/restored-mtime replacement,
   and read failure/bound checks. Use independent expected bytes/statuses.

Do not broaden other core packages. Existing text atomic signatures and errors
remain unchanged. Please freeze coherent source/results after correction rather
than continue speculative API additions. Root handles next integration planning.
