# Managed engine app review fixes — active Astra slice

ROOT CORRECTION after direct current-source read: DeepSeek findings1 and3 are
not supported as written. `packageJob.run` consults captured poison only inside
`if j.transition`; list/import can work independently. `RefreshScene` does NOT
clear selectedID; replacement callback does, correctly, and accepted scene
tests already assert preservation. Do NOT change scene code or add duplicate
tests for finding3. Do NOT block list/import merely because runner is poisoned.
Root withdraws the overly broad initial task below. Only the ALL3 repair button
is a demonstrated enabled-but-unusable transition when enginePoison is set.
Disable repair under poison and guard transition action entry consistently;
show sticky service failure. Existing Import/List stay usable. Reuse scene
tests and continue the originally assigned finite app tests/captures. Any new
finding needs an actual reachable source trace. No extra graphview gate.

2026-11-09. Astra owns only the concrete findings from
handoffs/fixed-engine-package-app-source-review.md plus focused tests. Read
current AGENTS top and app activation plan. Do not edit store/runtime/GLM
smoke files. No broad suite/build until fixes and focused tests are reviewed.

Fix all three findings:

1. In startPackageWork and package panel, persistent `enginePoison` must block
   new import/list/activate/reset/repair actions if the current worker path
   cannot actually perform them. The current job captures poison and exits
   before manager work, leaving clickable actions that always fail. Use one
   visible durable message. If you decide RepairResetAll should bypass poison,
   implement and test that explicit boundary instead; otherwise disable all
   actions including ALL3 repair and explain the service must be restarted.
   Keep the poison sticky across lane recreation.
2. A readonly scene RefreshScene should retain selectedID across unchanged
   source/version transitions, as well as zoom/pan. Only validate/drop it when
   replacement scene actually lacks the node, or source identity/version has
   changed. Add a focused graphview/editor regression for selected node plus
   refresh; existing replacement-absent test must continue to pass.
3. Add app focused tests with controlled runner/store seams for poison action
   refusal and transition preservation of selected readonly preview/writable
   draft/undo. Reuse existing package fixtures. Don't use a giant matrix.

Publish `native/handoffs/fixed-engine-package-app-review-fix-result.md` with
source summary, focused test/vet raw evidence and changed-input hashes. Root
will review before full app gate. Keep existing source review and failed runs.
