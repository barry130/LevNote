# Fourth fixed worker, stage 1: transport and package infrastructure

ACTIVE after .18 root binary acceptance,2026-11-10. Owner DeepSeek
`/root/read_native_handoffs_release10_ui_feasibility_task_md`. No
children; compact before200K. Root plans/reviews only. Every Go command uses
native/scripts/go.ps1. Preserve all old packages/evidence. No product docs or
application launches. This stage is source-only, no new release packaging yet.

Read `drawio-worker-next-plan.md` and root's refinement task/result (the root
refinement decisions prevail). Implement lower layers only:

- Fixed catalog ID `levnote.drawio-svg`, format svg,32MiB source/64MiB payload,
  engines/drawio-svg/levnote-drawio-svg.exe, Windows amd64. Existing identities,
  version1 frames/manifest schema1 and old exact request bytes unchanged.
- Drawio-only required page `{index,id}` with strict object and field types,
  index0..249999 and ID<=512UTF8bytes; empty ID permitted as actual identity.
  No unknown/null/duplicate/case-folded fields. Old3 refuse any page. Enforce
  validation before payload read/copy/bundle lookup or child launch. Bound
  marshaled request metadata too. `invalid-page` is well-formed-but-unavailable
  index or exact-ID mismatch, no lookup/fallback. Reuse strict JSON parser.
- Preserve Runner.Export(ctx,id,Source) API by additive typed optional Page
  metadata; copy into owned value at admission, document immutable ownership.
  Runtime prefix uses it. Invalid options must never call Source.Data or launch.
- New one-shot worker reads full unmodified source, parses with existing drawio
  limits, verifies requested page/index/ID, calls accepted bounded ExportSVG.
  No serialize/rewrite, filesystem/network calls or arbitrary options. Process
  itself is not sandboxed. Compressed/readonly pages allowed per current parser.
  Bounded issue conversion retains true page/cell context and coarse whole-input
  span; don't silently drop relevant issues or truncate IDs/messages. Cap256
  entries,64code/512message bytes; fail with warning-limit on excess. Keep
  warning order deterministic and avoid claiming exact source positions.
- Package schema2 exactly4 entries; schema1 still requires all old3 and maps new
  engine to bundled in memory, preserving installed selections and original
  state file until explicit persistence. Writes schema2. Corrupt state remains
  blocked; reset explicit; retention12 and cleanup/poison unchanged.
- Add fourth fixed build definition to package-markdown-worker.ps1 and its
  scope text. Existing verification/notices/manifest behavior maintained.

Own core/enginecatalog/**, core/engineipc/**, core/engineruntime/**,
cmd/levnote-drawio-svg/**, relevant existing worker tests only when shared API
requires, scripts/package-markdown-worker.ps1 and directly related packaging
tests. Narrow parser exception below. No changes to app/**,
modules/graphview/**, ui/editor/**, product docs or portable assets. Original
accepted SVG emitter/oracle/budget tests frozen.

Root refinement decisions:

- Runtime must copy Page at Export admission before any asynchronous/hook work;
  app will independently copy before enqueuing in stage2. No pointer mutation
  can change admitted metadata later. app/engine_lane.go owns runner call and
  task clearing; preview_worker.go constructs lane/preview tasks.
- Existing parseMxFile labels unknown root children/text with a mutable page
  counter, so selected-page filtering alone would falsely attribute/drop these.
  Own ONLY parse.go and one new focused issue-scope test plus a tiny new constant
  file if useful: mark its two mxfile-wide issue sites with Page=-1 (document
  scope). Leave page-local issues/strings/order unchanged. Worker includes
  document-scope plus selected-page parse issues, then selected-page emitter
  issues in that order; converts -1 to an explicit document label, never a
  fake page0. No new Issue fields or schema changes required. Add regression
  around two pages and outside-page unknown content. This narrow additive
  metadata distinction is the only allowed format implementation change.
- New worker error responses must contain nonnil empty warnings where the
  protocol requires; never expose partial SVG on parse/page/warning/output
  failure. Convert/bound parse warnings before expensive export when possible;
  combine count with emitter warnings afterward without silent truncation.
  Precheck raw cell-ID/message bytes before formatting bounded warning text;
  don't allocate a giant combined message just to discover it exceeds512B.
- Existing source snapshot captures decoded UTF8 text; worker source tests don't
  claim original disk-encoding/BOM fidelity. Host tests later verify disk bytes
  are untouched by export. Keep release version1 frame and package schema2
  compatibility decisions explicit.

Decisive tests (substantive behavior, not mirrored helpers): exact old3 request
prefix bytes and rejection of page; drawio present0 vs absent/null/extra/type/
negative/over-limit metadata, invalid encode rejection; runtime invalid-page
options before source/launch; worker second-page exact SVG/ordered relevant
warnings, invalid-index/ID refusal, compressed/readonly success, output-limit
nil output, protocol EOF/UTF8; parser source bytes unchanged. App undo/BOM/dirty
tests belong to stage2, don't claim them from worker-only checks. Package old3
installed-state migration/no-write-on-load/schema2/corrupt-reset/count/retention.

Run targeted regressions then one final affected suite/vet, using real command
outputs/exits. Choose packages from actual changes; preserve failures. Don't
repeat already-green large suites without new changes. No application package
build or real-worker delivery here: those follow stage2 integration. Existing
runtime tests may build their established helper binaries as required.

Write `drawio-worker-stage1-result.md` with precise files/hashes, commands and
raw logs, meaningful assertions, compatibility and limitations. Freeze for
independent review. App still advertises the old drawio host exporter in this
intermediate source stage; it is not a standalone user delivery.

Write `drawio-worker-stage1-progress.md` at the first compile/test milestone
with implementation scope, current blocker if any, and next check. Keep it
brief; this is coordination, not a substitute for the final evidence/result.
