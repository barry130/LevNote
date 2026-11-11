# Queued app slice — local native format configuration

2026-11-09. Root accepted .7 source docs and read actual core API/source.
ACTIVE for Astra immediately after core/plugins + direct modules tests/vet
pass and the actual result is published; no extra approval wait. Read alongside
`format-routing-implementation-next-stage.md`. Astra now owns core and app
sequentially after the interrupted GLM/DeepSeek attempts; no other code owner
is active. Finish the core gate before app source work.

## User workflow

Install a local JSON configuration for `.taskmd` or `.thoughts`, see its name,
ID and selected Markdown/Mindmap format in the plugin panel, then use its
`新建文档` action or open an existing alias file. Native modes, semantic outline,
export and readonly scene come from the existing selected adapter. Tell users
that the configuration uses a shipped format; implementation identifiers can
remain in concise details rather than dominate normal editing controls.

Own app/plugins.go, app/view.go, app/app.go and new/affected app tests. Add
two example JSON manifests under native/examples/plugins/ and final source docs after
functional acceptance. No engine/runtime/format module/editor changes without
first reporting a concrete necessary dependency. No package or worker launch.

## Specific seams root inspected

- configureTab already uses registry matching; there is no Adapter.Matches
  recheck to work around. Use modules.NativeRouteAdapter; keep MatchedPlugin.ID
  distinct from Adapter.ID and nil schema2 OutlineFactory for native routes.
- Current Schema>=2 branches mislabel schema3. Add deliberate schema3 wording,
  retaining existing schema1/2 highlight semantics and tests.
- Manager.Manifest must expose the normalized extension list. Root asked the
  core owner to align installed-list metadata with the normalized Registry
  result. Use that contract for `未命名` + first extension, not Syntax.Extensions
  or an app-local normalization copy. Recheck current active/enabled manifest,
  route and actual match after any Flush that can reenter/recover. If the first
  suffix is shadowed by another matching registration, refuse visibly. Do not
  silently choose a later suffix or force a hidden per-document route override.
- Existing addDocument flushes/release the old view before append, so use the
  common path. A failed Flush adds no tab and retains the failed draft. A
  successful earlier Flush is an undoable edit even if a later operation fails.
- installPlugin currently only flushes before native chooseOpenPaths. Recheck
  after the modal returns, before InstallFile, because nested input may have
  made a new draft. A cancel does not install. No new global transaction system.
- SaveAs already rematches changed paths; it does not convert format bytes.
  Tests prove same-format aliases, readonly writable copy and a clearly labeled
  deliberate cross-format path case. Recovery derives route from title/path.
- configureTab invalidates preview/export; disposePresentation releases outline
  and view. Same-ID refresh changes adapter/factory generation. Use controlled
  runner completion to demonstrate old results cannot attach or publish; do not
  introduce new runtime catalogs for routing.

## Finite checks

Use real installed manifests/adapters and existing test seams. Cover both route
types, disabled builtin mapping versus disabled configuration, shadowed first
suffix, normalized suffix creation, native semantic outline/source/split,
configured highlight, exact BOM/CRLF/unknown fields/save/reopen/undo, recovered
alias, local alias link/anchor, actual filesystem readonly/SaveAs, conflicting
draft refusal, post-dialog Flush and stale same-ID scene/export. Reuse existing
shared fixtures/helpers where their oracle is applicable. Prefer a compact
group of workflow tests over repeating broad format/runtime suites.

Publish source checkpoint, run focused new tests, then full app tests/scoped
app vet once stable. Keep actual streams and a before/after source checkpoint
in fresh evidence; retain failures and repair only affected causes. Capture
760x480 light/dark plugin-panel workflow with identity and create action
reachable, then one actual custom-suffix document view. Native headless only.
After these pass root reads source/results/images before a new bundle stage.

No child delegation; no repeat of .7 physical-worker tests just for path
routing. Accepted .7 delivery and historical evidence remain immutable.
