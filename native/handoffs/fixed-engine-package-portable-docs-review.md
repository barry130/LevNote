# Portable documentation — small factual corrections

2026-11-09. GLM owns only portable/README.md + handoff. Root read current text.
Fix these specific facts; no test/build or broader rewrite:

- MindmapSVG supports readonly documents too. Remove `可写` from its engine
  description (the earlier readonly paragraph correctly permits SVG export).
- Engine switching schedules readonly scene refresh automatically on next
  Draw/Warnings. Replace “请按提示刷新或重新请求” with automatic update wording
  and retained old-content warning on failure; there is no new manual Refresh
  button promised by current scene UI.
- Background cancel is not rollback after atomic commit. Say explicitly that
  if import/selection already committed, cancel may retain it and current status
  is authoritative. It never silently activates on import.
- If process cleanup has failed, transitions are blocked for this run, including
  ALL3 repair; import/list remain independent. Don't promise repair usable in
  that state. One concise sentence suffices, no new warning checklist.
- Final export file write is synchronous and cannot cancel after starting; avoid
  describing disk latency as bounded. Metadata/content size bounds are separate.

Keep {{VERSION}} and remaining limits/links. Publish updated docs handoff and
freeze portable README for later source-reviewed .9 packaging.
