# Draw.io fourth engine: read-only contract plan

ACTIVE READ-ONLY2026-11-10 for GLM agent whose task name references this file.
Root coordinates; two DeepSeek owners implement bounded SVG and .17 docs. No
children. GLM context max256K, compact before200K. Do not edit product source,
tests/docs/evidence, run commands except bounded source reads, build/launch or
measure. Only write drawio-worker-next-plan.md. Keep roughly three pages.

User wants self-authored native formats as plugins, low memory and simple UI.
Current catalog has three fixed engines: Markdown HTML, mindmap SVG, readonly
mindmap scene. Drawio SVG remains synchronous graphview.Export(selectedPage).
Bounded64MiB SVG emitter is separate active work; don't duplicate or change it.
Design smallest fourth fixed drawio SVG worker slice, ready for root decision.

Inspect actual enginecatalog, engineipc RequestMeta/json decode, engineruntime
Source/run/managed/store, app export_job/lane, editor contract, graphview page
selection/export and package script. Identify only concrete required changes.

Key constraints to resolve:
- Snapshot capture after Flush and all source limits before copy. Worker parses
  complete untouched XML using project drawio parser, exports exact requested
  page with bounded emitter, returns bounded warnings. Never default every
  request to page0 or rewrite user source. Existing compressed/readonly pages
  export according to existing format behavior, no edit permission needed.
- Current request has Engine/Format only, strict unknown/duplicate rejection.
  Propose minimal typed page descriptor optional ONLY for new Drawio engine;
  distinguish absent from0, reject extra/missing/invalid options. Preserve old
  engine wire bytes and manifest compatibility; decide whether version change
  necessary and justify. No arbitrary executable/path/plugin options.
- Current app exportJob checks tab/buffer/version/adapter but not selected page
  or view identity. Capture neutral view-owned request token/epoch after Flush,
  and revalidate at result and after native dialog. Page A->B->A should invalidate
  stale invocation if meaningful; switching page doesn't change text version.
  View.Release/reparse/source-mode behavior must have explicit policy. Shell
  must not inspect concrete graphview fields. Avoid new global routing framework.
- Current export button visible in source mode yet no view; prior drawio
  exportCurrent simply returned. Choose clear compatible behavior (e.g. disabled
  with visible select-page instruction), not silent change to first page export.
- Extend manager/catalog/script fixed list to4; remove literal three-count user
  copy with accurate count. Existing3-engine installed selections must load and
  new engine default bundled; corrupt selection behavior/poison stays. Preserve
  bounded retention12versions, explicit imports, queue/cancel/close.
- Worker stays one-shot, local, no direct FS/network. Metadata limits256warnings,
  source32MiB/output64MiB; warnings retain actual page/cell meaning via bounded
  message or source range without inventing precise offsets. Unknown metadata
  source preservation; no arbitrary plugin sandbox or memory-peak claim.

Deliver proposed concrete neutral interfaces/data fields and ownership by file,
compatibility/migration choices, at most two implementable stages and decisive
tests: exact second page bytes/warnings, invalid page/id, unchanged source,
page switch/release/Flush/modal changes cancel or refuse publication, old engine
wire no-change, manager4selection behavior, actual4th-worker run with owned process
cleanup. Reuse existing tests/script, don't design redundant giant test matrix.
Record unresolved code facts honestly rather than inventing availability.
