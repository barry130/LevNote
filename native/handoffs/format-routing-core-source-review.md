# Root source review — schema3 API ready for app

2026-11-09. Root read actual native_route.go, decodeManifest/validateSyntax and
surrogate scanning changes, manifestOf normalization, modules mapping and the
published core checkpoint. The actual API matches the planned two-route
contract; app can depend on it. This is source/API acceptance, not a claim that
the pending tests have passed.

Top-level discovery is behind the64KiB/UTF8 boundary, retains duplicate keys
and escaped names before dispatch, and sees schema3 before a later schema2.
Strict route objects reject unknown/null/prohibited fields and present invalid
defaults. Whole-input surrogate scanning now advances after closing strings;
the established schema2 cases must remain in the package regression. Module
composition returns existing adapters and does not add executable authority.
The normalized Manifest metadata fixes the new-document suffix seam without
an app-local normalizer. No actionable source defect found in this finite slice.

Astra continues the already planned core tests/vet. Once those scoped commands
pass, publish core result and proceed directly to the app activation; no idle
approval round is needed. Root will read retained core output while app work
proceeds. If a substantive product/test failure appears, report its cause and
repair within ownership before dependent implementation.

App scope is `format-routing-app-activation.md`, now active after this core
gate; source examples belong under `native/examples/plugins/` (the earlier
relative path was from the native workspace). Existing .7 delivery immutable.
