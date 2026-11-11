# Queued implementation — schema3 native format routes

2026-11-09. QUEUED until root accepts .7 actual delivery and its source docs.
Do not edit product/tests/scripts now. Root has read the corrected GLM design
checkpoint and current implementation. This file fixes scope and API choices
for later explicit activation; the user has delegated implementation decisions.

## Product slice

A local schema3 JSON manifest selects exactly builtin.markdown or
builtin.mindmap using nativeRoute. Custom extensions then open in the existing
native views with source/split/semantic outline and actual supported export;
readonly Mindmap uses the already-shipped scene worker. A small 新建文档 action
on an enabled schema3 plugin row uses its first extension and adapter template.
The UI describes a local format configuration using a shipped engine, rather
than displaying implementation jargon or claiming installed executable code.
Keep name/manifest ID/target format visible in the plugin details. Plugin/new
document controls must remain usable at760x480 in both themes.

No new process engine, template language, permission model, URL handling,
format conversion, arbitrary binary installation, drawio route or parser/UI
rewrite is part of this slice. Plaintext fallback remains unconditional.

## Fixed additive API

GLM owns core/plugins and modules/modules.go plus direct package tests only.
Before app edits publish the actual signatures/source checkpoint:

- plugins.NativeRoute string type with NativeRouteMarkdown and
  NativeRouteMindmap constants. SyntaxManifest.NativeRoute carries that type.
- modules.NativeRouteAdapter(route plugins.NativeRoute) (editor.Adapter, bool)
  returns a fresh corresponding existing adapter, false for unknown values.
  Feature/export/scene/outline/template truth remains in those adapters.
  Loader owns the small admission whitelist; there is no user capability list.
- No Manifest flag duplication is needed; Manager.Syntax already returns an
  owned copied DTO by installed manifest ID. Preserve deep-copy behavior.
- Schema1/2 stay compatible. Schema3 accepts exact field names/types, requires
  nativeRoute and rejects outline, unknown/duplicate keys, present nulls,
  invalid surrogate escapes and duplicate-schema evasion. Absent version/api
  use legacy defaults; present empty/zero/invalid values fail. Keep all existing
  byte/count/path limits. Do not widen builtin matching priority or installed
  order. Top-level duplicate discovery must be bounded by the64KiB input.

GLM publishes its short source checkpoint, then runs meaningful core/plugins
and modules package tests/vet with actual streams and before/after owned inputs.
No app edits, full-repo suite, bundle or real worker. Root reviews focused source
and admission oracles while Astra can prepare the app after API acceptance.

## App owner after core contract read

Astra owns app/plugins.go, view.go, app.go and related new/affected app tests;
source docs/examples at final completion. Consume modules.NativeRouteAdapter;
never copy route→engine/feature tables in App. Keep schema3 native semantic
outline exclusive; schema3 source highlighting can use its declared line-local
rules but must not inherit Schema>=2 source-outline claims. Successful route
binding keeps fixed adapter.ID; MatchedPlugin retains configuration identity.

New plugin-row document action resolves fresh enabled+active manifest/syntax
and actual registry match for the proposed filename before creating a document.
If another higher-priority registration shadows that extension, refuse visibly
rather than silently creating a template routed to another format. Let the
user use an unclaimed suffix or change enabled registrations; do not force a
one-off hidden route override. Reuse add/Flush and adapter.Template. A refused
Flush must not append a tab or leak a new source buffer/view.

Reuse configureTab generation, reconfigureTabs release, SaveAs/path rematching,
recovery and worker cancellation. UI install dialog must recheck draft/Flush
after modal return before mutation if it can have processed nested input.
Successful per-tab Flush before a later failure remains a valid undoable edit;
do not promise all-tabs transaction rollback. Preserve drafts, saved settings
and routing on persistence failure; existing partial Refresh behavior stays
visible rather than being relabeled atomic.

Save As only changes path/encoding under the existing core; it does not convert
JSON to Markdown. Cover same-format aliases, exact BOM/CRLF/save/reopen/undo,
unknown Mindmap fields, readonly→writable copy, recovered suffix, alias local
reference/anchor, disabled builtin vs fixed available capability, shadowed
new-file action, pending draft refusal and same-ID refresh stale scene/export
suppression. Use real native adapters and controlled runners for lifecycle;
the accepted.7 real worker tests need no repeat merely for path routing.

After narrow source/focused review, full app tests/scoped vet and760x480 native
tester images are the app gate. No bundle/build stage until root reads them.
Final docs state schema3 is an installable configuration for shipped formats;
the editable-engine package/protocol gap remains explicit. Stages continue
autonomously; no user choice or approval round is required for this scope.
