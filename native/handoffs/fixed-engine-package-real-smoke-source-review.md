# Real managed smoke source — concrete corrections

2026-11-09. GLM owns only managed_bundle_test.go as before. Fix now; runtime
followup compile is blocked by this newly added file. No execution yet.

1. Remove unused strings import and unused metadata Inventory field. Format
   with project gofmt. Notify root/Astra promptly once compilation blocker is
   gone; finish all following source fixes before declaring the file frozen.
2. Each NewManaged success needs immediate t.Cleanup Close fallback before any
   possible Fatal/early exit. Explicit normal Close remains required and its
   joined success must be captured separately for each runner. Don't rely solely
   on reaching the happy-path Close line. Error record cannot be overwritten by
   a later success. Log only one structured JSON record, not duplicate %+v plus
   JSON. Final cleanup logging should still retain failures after a Fatal.
3. Assert actual engine IDs and receipt.Engine/Version, source bytes unchanged
   after the second request too, exact output parity (the independent literal
   oracle is consistent with current emitter). Reject same source/host directory
   and require different release versions, otherwise this cannot prove older
   package selection vs new bundled restoration.
4. The helper currently reads only release inventory, not the engine manifest
   nor expected complete host. Require the fresh host executable and parse its
   and old release's fixed Markdown manifests; check inventory/manifest version,
   size/hash consistency and requested identity. Read metadata with hard caps
   (reuse limited file read, not unbounded ReadFile), do not execute/hash extra
   binaries: real Manager/Runner supplies actual held-file verification. No
   need to duplicate verifier implementation or hash entire releases here.
5. Correct handoff opt-in wording: only flag missing skips; flag=1 with missing
   source release FAILs. Record source-only, not proof of joined runs. No default
   test or vet run until root says source is frozen and coordinates with Astra.

This is a finite delivery test correction, not new product scope. Keep earlier
runtime compile failure evidence untouched; no more files outside ownership.
