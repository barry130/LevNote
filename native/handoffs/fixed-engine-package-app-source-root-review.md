# App package integration first source — root review

2026-11-09. Read current engine_packages.go/UI, preview/export changes, App.New,
View and SceneEngineInfo. Captured old lane, short metadata snapshot, separate
result slot and no editable-view reconfiguration match the planned boundary.
Continue finite app tests/captures; no additional idle approval gate.

Two concrete presentation corrections before final capture:

- Corrupt state: Manager.Snapshot uses placeholder bundled statuses while
  LoadError blocks resolution. Panel currently prints “当前选择：随包版本” above
  the corrupt-state warning. Don't call that a known current choice: when
  blocked, show the selection as unreadable/blocked and the explicit ALL3 reset.
  This is presentation only, not a change to manager state/fallback semantics.
- Background package job only displays its status inside the plugin panel.
  Closing the panel leaves the old ordinary a.Message and no progress/cancel
  entry, despite a transition still blocking requests. Show a compact global
  package-operation row or status with a reopen/cancel action while pending,
  identify the captured engine/action (not whichever engine panel is now selected).
  Preserve normal editor layout and avoid duplicate tall progress regions.

Also correct remaining scene view wording “应用的随包 SVG 导出” since an installed
package may now serve it; a narrow user-copy change only. DeepSeek independent
read-only review is running; route its concrete findings before final gate.

Runtime followup remains blocked solely by GLM's test source. The exact
default-SKIP compile gate is assigned to GLM under real-smoke-second-review.md.
Don't retry another runtime invocation until that stable compile passes.
