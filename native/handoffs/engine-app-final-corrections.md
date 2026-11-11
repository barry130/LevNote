# App final corrections before the runnable .9

2026-11-09. GLM takeover remains sole app production/test owner. Root read the
actual final tests/logs and installed-light/pending-dark images. Previous PASS
is real but acceptance is not complete. Finish this finite correction, not a
new feature stage. No workers/store/graphview/portable changes. No subdelegation.

1. export_worker.go exportStatusText: reverting to unconditional '随包外部'
   to satisfy pre-managed legacy tests is a product regression. An installed
   package is not bundled. Restore neutral '本地引擎 HTML/SVG 导出' (or derive
   truthful current source from cached selection). Update ONLY stale copy
   expectations in export_worker_test.go and export_mindmap_worker_test.go;
   preserve no-I/O/no-call, capability, stale/history assertions. Add a compact
   installed-selection status assertion that prevents unconditional bundled
   labeling. Keep lastVerified source/digest distinct from current selection.

2. Apply concrete independent review in engine-app-lifecycle-review.md:
   close-refusal race needs controlled delivery, poison needs post-list
   retention+second-refusal assertions, remove vacuous intermediate generation
   claim. Do not mutate production just to satisfy a timing-dependent test.

3. Original finite contract needs actual pending native draft plus untouched
   buffer/undo, and ready/reserved stale export transition. Existing 'pending '
   source edits do not test an uncommitted view draft. Reuse real graph tag input
   helpers from TestFormatRoutingRealPendingDraftBlocksNewDisableAndInstall or
   TestMindmapWorkerFieldFlushAndPendingTagRecovery: native pending tag, original
   view identity, exact buffer/history/selection/dirty unchanged by a successful
   package reset; draft still visible after. One focused case is enough.
   Add/reuse one ready-but-not-published export transition: capture old job/lane,
   make result ready with lane reserved, reset, establish a new controlled lane,
   deliver stale old callback and prove no dialog/publication and no release of
   new reservation. Existing tests cover generic lane behavior, but no current
   package lifecycle case proves this new integration. No expanded matrix.

4. Captures: pending-hidden images were saved AFTER reopening panel. Capture
   actual hidden progress row before clicking reopen; optionally then assert
   reopening. Avoid fast list completion race: a controlled pending transition
   with bounded Close barrier can hold real packageJob pending while frames run,
   then release/join during cleanup. Do not put fake pending booleans in product.
   Scroll reachability must check visible/hit-test bounds, not merely existence
   in AccessByRole (clipped items can remain in access tree). Installed controls
   at 760x480 already look good. Reuse those images if source is unchanged there;
   no extra blocked-state screenshot needed in addition to genuine pending.

5. Use native/scripts/go.ps1 -GoArguments for ALL runs/builds. Direct tools/go
   bypassed required wrapper context. Prior native files are untracked, so an
   empty git diff is not a before/after source baseline. Retain old logs exactly,
   take a small actual Get-FileHash inventory of app sources+neutral contracts
   before/after corrected runs, and use a fresh timestamped evidence directory.
   Run focused TestEnginePackage with -v -count=1 plus captures, then final full
   ./app once and vet ./app ./ui/editor ./modules/graphview. Capture command args,
   real stdout/stderr/exit. If failure occurs preserve it and retry affected stage.

Update fixed-engine-package-app-result.md truthfully: current draft tests don't
prove pending view preservation; original focused failures aren't present in the
listed focused directory (only final37byte stdout). Locate retained originals if
available, otherwise state missing logs, never recreate/fabricate them. Do not
claim all earlier failures preserved unless actual files exist. No .9 build yet.
After publishing correction result/images root will authorize already queued
single package, two real worker requests, one hidden start/close. Stop expanding
scope; user needs that concrete runnable release.
