# Tab workflow source accepted

2026-11-10 root read the final source/tests, retained failing and passing logs,
final identities, combined result and760x480Tester capture. Accepted the small
combined app stage: visual-order Ctrl+Tab/Ctrl+Shift+Tab, one-frame active-tab
reveal, deferred close-button mutation, and actual GoToLine focus-leave
composition reconciliation. Current runnable release remains .21 until a
separate .22 delivery is accepted.

Zero/one tabs have no effects. Existing Select/Flush handles source and rich
draft commit/refusal, export cleanup and persistence. Visible composing utility
fields retain preedit and request explicit completion/retry; no queued switch.
The discovered stale GoToLine flag was reproduced in15 (exit1), fixed using
the frame-local public Focused/Composing state, and passed both pointer-leave
directions in16. Source text/history/version remain unchanged by that utility
navigation. The guarded active preedit tests still pass.

Final affected app test18 passed17.278s/exit0 and vet19 exited0. No root rerun.
Earlier failure logs are retained. The existing capture shows tab13 button
fully visible at760x480; it is synthetic geometry, not physical Windows/IME.
No product package, host launch or resource measurement occurred in this stage.

Accepted product/test identities (bytes/SHA256):

- app/app.go:34047,
  0148CFA928154E440C22C3D3C9AE08CC5B34A791F1732C615F1932B3021BC1D3
- app/view.go:22464,
  BE5B492D1DA34AF246F447F5B61989BC90DD7ABFC4281DA6FBAC7FE4BCB62909
- app/source_navigation.go:8453,
  76998D57326DB1ED75BC27C2E7C34BEB68DD204E54E16316002F0A928C8CB2F5
- app/tab_workflow_test.go:22725,
  83F835E597A23238FE45408BDA55262995F77252ADB80122E971460EB8722A2C

Combined result4268B:
706C8D2FEC34A163CCDC54A1DD0787C09B7D24FF6501C6BB8EB4AA49E76FA702.
Reveal result2309B:
0231C47B324DF74381F6F000BB017E7DC5EEC0E8F941F83B7113ECF00D2E2614.
Machine-readable source/capture identities and all logs:
[tab-workflow evidence](../artifacts/tab-workflow-20261110/final-file-identities.json).

.22 execution remains pending completed helper review and resource test-source
freeze. No source/docs mutation while packaging. Unchanged engine behavior
inherits .21; do not repeat worker requests for this host-only change.
