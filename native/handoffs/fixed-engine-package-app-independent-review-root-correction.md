# Root adjudication of independent app findings

2026-11-09. Root checked current code after receiving the independent report.
Findings1 and3 are withdrawn; the prior fix task was narrowed immediately.

1. packageJob.run assigns captured poison only inside `if j.transition`.
   Import/list are not transitions and can legitimately use the store even if
   a process cleanup poison blocks new executions. No reason to disable them.
2. ALL3 repair is a transition, so its button (currently only Disabled(busy))
   is enabled even though poisoned execution will refuse it. This is a valid
   narrow UX issue. Disable it under poison and guard transition admission.
3. RefreshScene has no selectedID assignment. The assignment at scene_preview.go
   replacement callback occurs only after the new scene's index is built. The
   accepted TestNativeSceneRefreshSameVersionDrawPreservesDocument and
   TestNativeSceneRefreshFailureRetainsStaleThenDropsAbsentSelection already
   exercise exactly the stated preservation/drop boundary. No product defect.

The report is not a test result. No extra tests are required to prove absence
of these two misread defects; app integration preservation remains in the
original authorized finite test slice.
