# Next useful app workflow: reopen a closed saved file

ACTIVE read-only planning task2026-11-10. Owner Astra
/root/read_native_handoffs_tab_workflow_resume_task_md. No child delegation.
Image-reference source is frozen for .26 packaging; no runtime edits/tests or
launches while that work proceeds. DeepSeek owns .26 preparation/execution and
GLM owns final independent reference review/current docs.

Review a small next workflow informed by mature multi-document editors:
Ctrl+Shift+T and a discoverable native command to reopen the last successfully
closed saved local file. Root's proposed direction is a bounded in-memory stack
(up to16 lightweight entries), no persistence or retained Tab/Document/Buffer/
undo/pixels. Route through existing open/select/Flush barriers; already-open
targets select existing tabs. Explicit discarded drafts must never resurrect.
Unsaved documents without an authoritative saved path are not candidates.

Read actual CloseTab/commitClose/removeTab/CanClose, async binary open and text
encoding handling, existing shortcut/utility composition guards and layout.
Decide minimal metadata needed (e.g. path plus previously explicit encoding)
without preserving content. Do not guess GBK/GB18030 or bypass read-only/file
checks. Record only after successful close persistence/cleanup; failed/canceled
close must not enqueue. Whole-window quit must not build history. Do not pop a
candidate until open/select really succeeds; avoid coupling a stale async image
completion to another request. Missing/replaced/disabled-plugin files should
follow current truthful open behavior with a visible outcome and a recoverable
stack, with a defined way past missing entries if needed.

Deliver a concise source-grounded plan at
native/handoffs/reopen-closed-file-next-plan.md with actual seams, smallest
changes, user wording, bounded storage, input/composition/modal behavior and
meaningful focused regressions. If this is not safely small, identify the exact
blocking seam and suggest the smallest prerequisite, not a new framework.
No online browsing is needed; cite current source lines instead of claiming
new external research. No implementation until root activates after .26 closes.
