# Independent review: draw.io native route

ACTIVE 2026-11-10 with implementation and examples results frozen.
DeepSeek owner `/root/read_native_handoffs_release10_ui_feasibility_task_md`;
this owner authored examples only and is independent of the code implementation.
No child delegation, no product edits/test execution/builds/launches. Context
below200K. Read current implementation result and concrete changed files only,
not historical release inventories or vendor/toolchain trees.

Review against `drawio-route-implementation-task.md` and source acceptance for
the fourth worker. Confirm the new schema3 route changes no strict manifest
validation/priority/persistence or executable policy, adapter capabilities are
fresh and truthful, and custom suffix uses the existing template/native view/
current-page export contract. Independent config enable state must survive
disabled builtin, source fallback must be explicit, and SaveAs must not convert
content. Existing Markdown/mindmap tests must not lose outline assertions just
to accommodate drawio.

Check meaningful app tests for the real native graph field/Flush, second-page
request, readonly copy, failed draft retention, late export invalidation and
same-ID rematch. Distinguish new checks, inherited lower-layer coverage and
unexecuted cases. Read raw final affected-suite/vet exits and any failure
chronology; inspect new Tester capture if generated. Do not request broad repeat
matrices for already covered shared code. Review docs only for new route claims
from the frozen examples task; note you authored those docs and root independently
reviews them. `.19` general release docs remain a separate GLM task.

Publish `drawio-route-independent-review.md` with actual actionable findings
and source lines or a clear no-blocker conclusion plus limits. Exact source
identities should match the implementation handoff. This is source review and
does not accept future `.20` binaries or claim physical UI/resource validation.
