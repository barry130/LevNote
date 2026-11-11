# Canonical draw.io containers: warning-only increment

ACTIVE2026-11-10 after .20 root binary acceptance; see native/AGENTS.md.
Implementation owner DeepSeek resource
agent; no children, compact before200K. Root plans/reviews only. All .19/.20
packages and evidence immutable, no builds/launches/captures in this slice.

Read `drawio-structural-warning-plan.md` and
`drawio-warning-plan-review.md`. Implement the narrow canonical first ordered
pair 0/1 recognition in formats/drawio/parse.go only. This does not claim general
layer/group support: reordered/later/custom structural cells keep current
warnings until a separately designed generalization. No new dependency,
second XML decoder, full-source copy or UI/worker warning filters.

Root reconciliation of reviewer/original-plan differences: ordinary XML
whitespace and comments inside otherwise empty cells are harmless and MUST be
allowed (original plan). Review's prohibition of comments is not adopted.
Processing instructions, namespaces on element/attrs (including inherited
namespace Name.Space), extra attrs/value/style, geometry/children, metadata,
non-XML whitespace/text disqualify. Exactly id=0 on the first direct mxCell and
id=1,parent=0 on the immediately next direct element; comments/XML whitespace may separate
them. Other root element siblings between them prevent the pair; outside the
pair they retain their own warnings. A preceding non-mxCell sibling need not
invalidate the first-mxCell rule but must remain visible. Attribute order,
quote spelling and explicit close vs self-closing must not matter.

Both cells must fully validate before removing EITHER generic nonvertex
warning. If second fails recognition, preserve first and all surrounding issue
order. Keep duplicate-ID map/insertion, depth/elements/text/decompression budgets,
readonly status, source offsets, IDs, parsing errors and edit semantics. Errors
while validating the pair must not suppress only one warning. A later unrelated
parse error still fails normally; no new whole-parse warning rollback required. Source and
SVG bytes remain unchanged. Model/node/edge output is unchanged.

Tests owned: formats/drawio focused parser/alias-free warning tests and changed
existing SVG oracle assertion; cmd/levnote-drawio-svg/main_test.go expectations;
one modules/graphview template contract test if needed. Never edit frozen
svg-prechange-oracle.json or old handoff/evidence. For exact old SVG oracle,
assert old SVG/source byte identity and explicitly subtract just the two known
canonical-container warnings for the new issue expectation. Do not broadly
filter all nonvertex warnings in a test helper. Include invalid second pair,
missing second, namespaces, comments/whitespace/PI, unrelated siblings, third
cell, duplicate IDs, unchanged edit offsets and budget/error paths.

Worker tests: selected page loses only canonical two warnings; compressed
readonly warning remains. Genuine unsupported cells/metadata retain ordered
issues. Exact256 and257 selected/document warnings plus large unselected
population must use substantive real issues, no weakening count limits.
Root found one additional current-source oracle: in
core/engineruntime/drawio_managed_bundle_test.go, the gated real smoke currently
expects two canonical container warnings at lines410-421. You own only that
literal expectation change too: require zero warnings for its unchanged clean
second-page fixture; keep exact598byte SVG/source and process/selection checks.
This is future tester behavior, not permission to run it against immutable .19
or .20. No compatibility branch pretending one oracle covers both parsers.
The next .21 delivery will compile/use the updated test on its own candidate.
No engineruntime production changes. Add this package to final compile/test/vet
scope, with real-smoke environment unset so the actual worker gate stays skipped.
Update graphview/app assertions only if actually affected; describe any extra
scope before changing production beyond parse.go (none expected).

Use native/scripts/go.ps1 -GoArguments. Run focused regressions during work,
then once final affected test/vet: ./formats/drawio ./cmd/levnote-drawio-svg
./modules/graphview (and ./app only if its affected test expectation changes).
Do not rerun full native tree or passing package/launch gates. Preserve every
failed command/log/exit and correct evidence narrowly.

Publish `drawio-structural-warning-result.md` with exact files/sizes/hashes,
raw evidence paths/commands/exits, intentional warning change and unaffected
SVG/source/offset behavior. Freeze for independent review; no self-acceptance.
After acceptance the retained-source optimization is next; no standalone new
package is required for this tiny warning increment before that optimization.
