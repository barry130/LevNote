# Table columns: active native UI and integration

ACTIVE2026-11-10. Owner Astra tab_workflow_resume after .24 acceptance. No child
delegation. Read native/AGENTS.md, markdown-table-columns-next-plan.md and
markdown-table-columns-format-task.md. Format owner is DeepSeek and owns only
the new format planner/test; this owner handles the UI and app integration.
Do not edit each other's files. .24 package, portable README and evidence frozen.

Implement new modules/markdownview/table_columns.go/table_columns_test.go and
app/markdown_columns_app_test.go. Narrow edits allowed in table_actions.go,
table_rows.go, table_alignment.go; their existing row/alignment behavior remains.
No app runtime/parser/vendor/core edits. If a meaningful existing test fixture
depends on the changed focus order, preserve its intent and report the narrow
adaptation, do not weaken behavior. No unrelated tests or UI redesign.
Specifically the existing focusAlignment helper in table_alignment_test.go and
activateAlignmentApp plus optional capture in app/markdown_alignment_app_test.go
use fixed old Tab counts. Ownership includes these two tests for exact new
focus-route updates, retaining target/focus assertions.

Use the exact format API/error names/order defined in format-task.md. Extend
the common patch representation to multiple edits minimally, keeping existing
single-edit row/alignment contracts stable where possible. Validate original
byte ranges/UTF8 boundaries and convert to original UTF16 offsets before edits;
apply descending in one owned group. Map both selection endpoints with the
existing insertion affinity in that same order; set once before EndGroup.
No callbacks inside the group; Changed exactly once afterward. Foreign groups
and stale source/selection refuse. CancelGroup restores text/selection/history
but may advance revision: do not claim unchanged version after internal failure.

Keep existing gesture capture, normal draft Flush, receipt/epoch/owner checks,
release/recovery and post-Flush invalidation. A changed cell and structural
operation remain two independent undo steps. A post-Flush column refusal says
the cell was written and column unchanged. A failed draft must remain reachable.
Do not invent a separate draft flush path.

Add one short wrapped row between row and alignment controls: insert before,
insert after, delete this column. Stable accessible labels identify each action.
Total columns1/8 control delete/insert disabled state; readonly all disabled.
F6 first target remains existing insert-row button. Tab/ShiftTab deterministic
through enabled controls, Enter/Space once, repeat/held pointer/scroll/cancel
semantics retained. Preserve input-method composition checks and fixed hint
height so normal commit cannot move a pressed button. Verify265/300/760DIP
as appropriate with actual public Tester geometry.

Cover new-column behavior meaningfully: pointer + keyboard, narrow layout,
clean one-undo/changed-cell two-undo, exact source and selection through all
edits, composition/stale/conflict/readonly/foreign-group/release/callback
replacement, post-flush refusal, boundary disabled states. Reuse existing
helpers, don't duplicate the entire row/alignment suite. Add one real app
BOM+CRLF route through actual adapter, disk unchanged before Save, dirty state,
Save/reopen bytes and unchanged neighboring content. No physical IME claims.

You may prepare tests/UI in parallel with format implementation. Run focused
view/app new tests after the shared format file exists; preserve failures and
correct concrete causes. Do not rerun accepted source/image checks separately.
Once both owners freeze and root reviews, one combined affected app/view/format
test+vet will be separately activated for the final integrated source. No
package/host/resource run in this task. One optional native Tester screenshot
of the actual column panel may accompany the focused test, with visible layout
review and no additional capture harness.

Write markdown-table-columns-ui-progress.md after a substantive milestone and
markdown-table-columns-ui-result.md with exact files, evidence and limitations.
Use named handoffs for fallback when messages unreadable. All Go uses the
workspace wrapper. No further plan-only phase; implement this increment.
