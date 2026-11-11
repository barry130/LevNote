# Targeted Find keyboard review

2026-11-09. Activated for a bounded interim read while owner finishes tests.
Do not claim final acceptance until owner result freezes exact hashes. Publish
actionable interim findings promptly; a short final reconciliation may follow.
Reviewer DeepSeek read_native_handoffs_resource_workload_next_task_md. Read-only
source/evidence review, no product edits/test/build/launch/docs. No child agents.
One bounded review turn; focus on real issues rather than re-inventorying repo.

Read find-keyboard-implementation-task.md, find-keyboard-result.md and actual
changed code/tests/raw test logs. Accepted .13 search/cache and layout are the
baseline; source before/after hashes available in .13 evidence. Review:

- One-shot Ctrl+F/query focus, repeat from replacement, no field remount/focus
  stolen on ordinary frames; query/replacement values/options preserved.
- Event-current text in handlers; no retained ui Element/TextKey; Enter/Shift
  use proper operations; readonly/fresh replace semantics; repeated query Enter
  focus restoration doesn't revive abandoned source/tab/closed panel.
- Window shortcut ordering and source/rich/Find composition across Enter,
  Escape, repeated Ctrl+F and pointer close; toolkit keeps composition ownership.
  Distinguish synthetic behavior from unverified physical OSIME.
- Closed/reopened generation and tab/document/editor/buffer identity guards,
  current-tab semantics, failed/identity-changing Flush preserving drafts and
  preventing wrong target. No unintended search-cache/replace engine changes.
- Existing Ctrl+G, toolbar close/toggle, source focus and min-window layout
  retained; tests exercise actual App/toolkit meaningful behavior, not mirroring
  helpers or asserting an invalid caret position after replace-next.

Report actionable defects with verified file/line/trigger/impact and bounded
recommendation. If none, state no material blocker in this scope and limits.
Do not turn optional polish into blocker, don't repeat accepted tests. Publish
find-keyboard-independent-review.md and notify root. Verify only final changed
file/evidence identities needed to reconcile owner's result.

Current specific questions worth checking: source Reveal and query pending
Handle requests now resolved through tiny Editor.CancelFocus; window Escape
registration is conditional before Shortcut; composing Enter needs consumption
before toolkit default commit. Cross-panel Ctrl+G during Find composition and
field composition flags after a tab changes remain to be tested by owner.
Coordinate directly with owner if payload readable; root receives conclusions.
