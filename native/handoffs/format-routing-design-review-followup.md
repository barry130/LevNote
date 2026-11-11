# Route design review — finite refinement, no more searches

2026-11-09. Root read the saved format-routing-design-checkpoint.md and current
core/plugins, app/plugins/app/state/outline/reference, modules and adapters.
The two-route schema3 direction is accepted in principle. Your previous turn
was stopped by the repeated-identical-tool-call detector. Do not repeat that
rg call or restart exploration. Use the saved checkpoint and these concrete
review findings; update only that design handoff, then finish.

1. Adapter.Matches is not called by the current app route path. configureTab
   already uses registry.Match, then adapterByID. No new bypass mechanism is
   needed. Say this accurately. NewDocument's fixed kind/name map is the actual
   extra path seam. Choose one small discoverable entry in the plugin panel,
   e.g. enabled native-route row's 新建文档 action, resolving current manifest
   and using its first normalized extension plus fixed adapter.Template.
   All draft/Flush/add barriers remain shared. No custom template authority.
2. Schema3 default version/api applies only when omitted; explicit empty,
   null, wrong types or apiVersion0 must fail. Declare how schema detection
   cannot be evaded by duplicate schema keys such as 3 then2, while legacy
   schema1/2 compatibility is retained. A bounded ordered top-level field pass
   can discover any schema3 declaration or nativeRoute key before choosing
   strict validation. Invalid surrogate escapes must not be silently replaced
   in new schema3 strings. No generic arbitrary-JSON framework is needed.
3. Capabilities must have one authority. Plugins may own the closed route-key
   constants/validation, modules maps those keys to a freshly returned adapter,
   app consumes the neutral adapter. Avoid two independent executable/feature
   catalogs or redundant mutable capability bits. Native syntax notices/badges
   must not label schema3 as schema2 via the existing Schema>=2 branches.
4. State actual failure semantics: successful Flush may write a draft before a
   later installation/settings failure; do not promise absolute zero buffer
   mutation across that action. Failed Flush retains unsynchronized content;
   later persistence failure keeps selected routing/views and saved settings,
   and does not throw away the now-written edit. Existing Refresh may recover
   with load issues and reconfigure usable items; don't invent a transactional
   refresh guarantee that the implementation doesn't have.
5. Save As rematches by destination path without converting the format. A
   Mindmap saved as .md is JSON opened by Markdown, not a Markdown conversion.
   Route-aware tests should choose same-format aliases except a deliberate
   explicit cross-format path test with truthful wording. Local references are
   generic path-based in app/reference.go, so custom suffixes should work via
   Open/registry; include one real alias-to-alias local link test, no new URL
   handling. Native semantic outline remains authoritative.
6. Refine finite implementation split for later activation: GLM only
   core/plugins schema/loader/tests and modules fixed-route composition/tests;
   Astra only app wiring/UI/tests/docs after core contract checkpoint. Fixed
   exported route types/API declared before both edit. Core can test during
   app source prep; no same-file ownership. No implementation until .7 package
   and docs are accepted and root explicitly activates the next slice.

This is a design edit only. You already have enough source evidence; no need
to run additional searches, tests or commands beyond saving/reading your one
design file. Publish the corrected checkpoint and stop cleanly without a
progress-only wait loop.
