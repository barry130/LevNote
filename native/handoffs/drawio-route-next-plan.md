# Planned next increment: data-only draw.io format routing

ROOT PLAN 2026-11-10. Implementation remains INACTIVE until .19 source/package
stage completes or root explicitly assigns a disjoint source interval. Root
does planning only; all code/tests/docs delegated. No children. The prior GLM
planning turn was halted twice for identical-read loops; do not resume it for
this plan. No code change is attributed to that failed turn.

## Scope and decision

Extend existing schema3 closed NativeRoute with exact `builtin.drawio`, retaining
schema3/API1 and all validation/limits. A user's local JSON format config can map
a custom suffix such as `.diagramnote` to the shipped project-authored drawio
adapter. Builtin drawio retains independent enable state; config route itself
still follows installed registry order and own enable state. This selects a
compiled capability, not executable paths or new engine installation.

Source map verified by root:
- core/plugins/native_route.go constants; declarative.go validateSyntax currently
  accepts only two routes. Keep strict duplicate/null/unknown/type/UTF8 validation.
- modules/modules.go NativeRouteAdapter returns fresh shipped adapters; add
  DrawioAdapter there. App continues neutral adapter matching, no parser imports.
- app/plugins.go configureTab already assigns schema3 adapter and source mode
  fallback; New template/SaveAs/lifecycle tests exist in format_routing_test.go.
- app declarativeCapability currently says every route reuses '编辑、大纲与导出'.
  Drawio has no outline provider. Adjust generically from capability presence or
  use truthful count-neutral '原生格式能力' wording; don't promise drawio outline.

## Minimal implementation

Add constant, validate allowlist, composition factory branch and meaningful route
tests. Strict unknown-route test presently uses builtin.drawio; replace its bad
value with an actually unsupported identity without deleting the rejection.
Current .19 drawio adapter already carries RequiresView worker export capability:
route inherits exact current page contract and source-mode refusal automatically.
No new schema/executable/renderer/template/IPC copy. Generated drawio template is
the sole origin; raw SaveAs changes pathname routing, not body format conversion.

New config example under examples/plugins plus example README/user/portable
guide will describe current-page selection, disable fallback, source limits and
no arbitrary plugin execution. Do not edit accepted .19 assets after packaging.

## Focused verification and stage sequence

1. Plugin decode/lifecycle + modules factory: exact route success, normalization
   and copy/enable semantics, real unknown route refusal, template valid drawio,
   no invented outline capability.
2. App custom suffix new/open/read/edit/split selected second-page export metadata;
   source/nil view refuses; disable or unavailable compiled route returns editable
   plaintext/read-only inspect as appropriate. Builtin disabled doesn't silently
   disable separately installed route. No disk/BOM/metadata mutation absent edits.
3. Failed pending graph draft Flush must block disable/new/install/rematch as
   existing routes; successful SaveAs rematches by new suffix without converting
   bytes. Same-ID route refresh invalidates old exported result. Use existing
   lifecycle helpers, don't duplicate global transactions or broad matrices.
4. One final affected plugins/modules/app suite and vet after corrections, review
   code and narrow UI capture only if exposed capability text actually changes.
   New package/docs follow actual source acceptance; .19 stays historical.

This is deliberately small: no custom drawio parser formats, new unsupported
diagram editing, permanent process, new UI framework, schema4, arbitrary code
plugins, network, installer/download or extra idle work. Resource improvement
claims require a separate measured workload; this routes existing capabilities.
