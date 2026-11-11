# Implement a native already-open document picker

ACTIVE2026-11-10 after composition-release29-root-acceptance.md. Owner Astra
/root/read_native_handoffs_tab_workflow_resume_task_md; no children. Root plans/
reviews only. Read open-document-picker-next-plan.md (accepted design baseline)
and this refinement. Implement directly; no further design-only stage.

Own native/app/open_documents.go, native/app/open_documents_test.go and small
app.go/view.go/workspace.go wiring. No core/module/vendor/storage/contracts,
product docs, portable README, historical evidence/package edits. Other agent
owns .29 docs; reviewer may inspect only. Save progress/result under handoffs.

Ship Ctrl+P and a compact native '已打开文档' dialog; toolbar entry is outside
scrolling tabs and KeepFocus. Case-insensitive Unicode title/path substring,
stable visual tab order, real keyboard Up/Down/Enter/Escape and single-click
selection. No new tab, indexing, disk reads, background jobs or row editors.
Use existing100tab admission authority. Bound sanitized/cloned display/folded
metadata (title256bytes/path4096bytes) and query128UTF16 units, show clipping
limitation, duplicate-name distinction, active/dirty/readonly/binary state.
Virtual rows, stable tab IDs, selected ID retention, no content/pointer snapshots.
Clear all picker metadata/list callbacks/focus handles on close and Shutdown.

Reuse existing admission, source/rich/utility composition and flushTab barriers.
Opening may not steal uncommitted preedit or bypass sticky rejected preedit.
Revalidate origin after synchronous Flush. Activation must resolve ID anew and
validate generation/origin/destination in selectChecked's post-Flush callback;
explicit Flush for same-tab path. A failed Flush returns to original draft,
keeps error and correct existing opener focus; do not accidentally focus a hidden
source editor in preview. Async/reentrant tab mutation invalidates stale rows.

Concrete toolkit risk to prove early with focused public Tester regressions:
DialogBase writes open=false for backdrop/Escape BEFORE its child callback.
Apply KeepFocus to backdrop as well as row/cancel buttons. Do not assume
resetting open=true on Dismissed preserves composing input/focus: prove actual
query preedit survives blocked click, backdrop, Escape, then explicit commit or
cancel lets retry work. IME owns Enter/arrows/Escape; no ancestor Escape shortcut
that preempts it. If this exact public-control design cannot satisfy the tests,
document the failing observation and implement the smallest public-API modal
composition solution in assigned app file; no vendor fork or private widget
introspection. Root can review a concrete alternative without asking the user.
Do not add parent/on-close shortcuts that discard underlying source preedit.

Use fresh TextKey.Text() for immediate filtering/Enter; no retained frame
Element. Defer clicked/submitted action until list building ends. Avoid changing
Select lifecycle for unrelated callers. A visible dialog keeps underlying active
document mounted; selection alone/filtering may not mutate bytes/history/order,
create inactive views or persist picker history. Opening under shutdown/restore/
native-modal state is refused. Repeated opening focuses current query only.

Independent contract checkpoint (open-document-picker-independent-review.md)
confirms one scope requirement: top-level App.shortcuts does not fire through
DialogBase's modal scope. Register the repeated Ctrl+P route inside that modal
with public API and test moving focus within the dialog then refocusing query.
Root also requests actual next-frame destination focus after successful Select:
its queued Editor.Focus and modal opener restoration must not fight. Same-tab
preview activation must preserve a real visible opener, not hidden source focus.

Tests should establish real public control routing/focus and state invariants,
not merely mirror helpers: Unicode/emoji/paths/untitled/duplicate/badge/bounds;
100mixed tabs+virtualization and final-row reveal; shortcut/typing/Enter/click/
Escape; source/rich/utilities and rejected preedit barriers; query composition
cases above; real rich failed Flush with correct field focus; same-tab barrier,
reentrant removal/reorder and async active identity change. Include no-result
and query shrink navigation. Use one760x480Tester capture if layout needs proof.
Source tests do not establish physical Windows IME behavior.

Run meaningful focused checks via native/scripts/go.ps1 -GoArguments only,
save exact commands/outputs/exits including failures. Freeze and publish actual
changed-file identities, implementation result and limitations. After root and
independent review, one final affected app test/vet will be assigned; do not run
package/product hosts/workers/resource checks now. Save useful progress before
context compaction; GLM/DeepSeek reviewers compact by200K.
