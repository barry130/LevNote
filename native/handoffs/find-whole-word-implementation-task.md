# Whole-word Find: active implementation

ACTIVE2026-11-10 after .16 root package acceptance. Owner DeepSeek
`/root/read_native_handoffs_release10_ui_feasibility_task_md`. Root coordinates
and reviews only. No children. Compact before200K context. All Go execution uses
the workspace native/scripts/go.ps1 wrapper. Product docs belong to other owner.

Implement the reviewed scope in [find-whole-word-plan-review.md](find-whole-word-plan-review.md).
Current accepted package .16 and all historical package/evidence directories are
immutable. Own app.go/state.go plus narrowly needed app tests and fresh evidence.
No core search algorithm/cache design/session schema/dependency changes.

Add runtime WholeWord bool, shared findOptions wiring and a native 全词 checkbox
with existing KeepFocus/Escape behavior. Existing full Options cache equality
already invalidates a warm scan when toggled. Document Find/navigation/Replace
and workspace search must share this option. Unicode letter/number/mark/_
boundaries are the existing semantics, not Chinese linguistic segmentation.

Verify actual checkbox next/previous/wrap, regex+case interaction and Unicode
boundaries; warm cache invalidation; Replace Current/All with grouped undo;
readonly and failed Flush preserve buffer/version/selection; one app workspace
fixture. Extend existing layout/focus/Tab/Escape tests rather than invent a new
framework or broad duplicate test matrix. Run relevant focused tests, preserve
real failures, then one final app+core/search suite and vet after fixes. Do not
repeat already-passing unrelated gates. No build/host launch at this source stage.

At760x480 retain both-theme bounds/overlap/editor/status assertions and the max
editor-font case. Write TWO fresh light/dark tester screenshots with existing
LEVNOTE_FINDBAR_LAYOUT_SCREENSHOT_DIR hook to a new evidence folder, never old
.13 captures. Root will view actual PNGs before acceptance; tester images are
not physical OS input evidence.

Publish find-whole-word-implementation-result.md with changed source inventory,
actual commands/raw outputs/exit codes, assertions exercised, screenshot paths
and hashes, and limits. State readiness for independent review; don't self-accept.
If a review finds a defect, fix within this ownership, retain failed attempts,
and rerun only meaningful affected checks. After result wait for review/next task.
