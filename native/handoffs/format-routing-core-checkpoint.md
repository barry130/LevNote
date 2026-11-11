# Schema3 core API/source checkpoint

2026-11-09. Core scoped gate passed; actual results and final hashes are in
`format-routing-core-result.md`. Astra is the sole implementation owner.
App was not edited by this step and now proceeds under its separate activation.

The actual additive API is:

```go
type NativeRoute string
const (
    NativeRouteMarkdown NativeRoute = "builtin.markdown"
    NativeRouteMindmap  NativeRoute = "builtin.mindmap"
)
// SyntaxManifest now has NativeRoute NativeRoute.
func modules.NativeRouteAdapter(route plugins.NativeRoute) (editor.Adapter, bool)
```

`core/plugins/native_route.go` discovers all bounded top-level keys before
dispatch. Any schema3 or nativeRoute declaration selects strict validation;
schema3-to2 duplicates cannot evade it. The route decoder rejects duplicate,
unknown/wrongcase, null and outline fields, null array items, ill-formed
surrogate escapes and unrecognized routes. Defaults apply only to omitted
version/apiVersion. `decodeManifest` itself now enforces the existing64KiB cap.
Legacy schema1/2 decoding/defaults remain on their existing path. The shared
surrogate scanner advances past closing quotes so whole-object inspection
checks each actual JSON string, retaining literal escaped backslashes.

`declarative.go` uses the existing `normalizeExtensions` at `manifestOf`, so
Manager Manifest/Manifests and Registry Match expose the same normalized,
deduplicated suffixes. Syntax preserves its declared spelling and copied slices.

`modules/modules.go` returns the existing fresh Markdown/Mindmap adapter from
the fixed route catalog; it does not look up builtin enablement or invent
capability metadata. `InstalledSourceOutlineFactory` remains schema2-only.
There are no app, concrete format-module, worker or packaging changes.

Completed: independent strict-admission and installed-lifecycle/copy/rollback
oracles, direct adapter template/outline/capability isolation, then only
`test ./core/plugins ./modules` and corresponding vet with real retained streams.
