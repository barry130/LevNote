# Draw.io route example and portable guidance

ACTIVE 2026-11-10 for DeepSeek
`/root/read_native_handoffs_release10_ui_feasibility_task_md`.
The `.19` package and direct evidence are accepted and immutable. The separate
implementation owner is adding `builtin.drawio` to the existing schema3 allowlist.
You own only `native/examples/plugins/diagram-notes.json`,
`native/examples/plugins/README.md`, and `native/portable/README.md`. No shared
root/native docs or formats/API edits; a GLM owner is updating `.19` docs there.
No children, compact before200K. Root plans/reviews only.

Create a small complete local JSON example, schema3/API1, id diagram-notes,
route builtin.drawio, extension `.diagramnote` (optional second alias only if
useful). Follow existing templates and project naming. Avoid arbitrary code,
commands, network or custom syntax assumptions. Example text must truthfully
explain that existing draw.io XML uses the shipped native editor and selected-
page SVG engine; changing suffix does not convert other formats into draw.io.

Update examples README and portable README schema3 sections to show three
routes and the new example, including a complete inline JSON in portable docs.
Do not promise draw.io outline/scene capabilities. State source-mode export
guidance, supported subset/metadata preservation, own config enable state,
normal draft Flush safety, and import-data vs installed-worker distinction
concisely without duplicating the entire limit/reference discussion. Retain
portable version placeholder exactly once and all existing local notice links.

Source examples are intended for the NEXT build. They must explicitly state
the accepted `.19` bundle only supports Markdown/mindmap schema3 routes;
draw.io suffix routing requires a later build. Portable README is a future-build
template: do not edit any `.19` shipped file or claim a `.20` release exists.

No Go suite, build, product launch, actual worker or screenshot. A narrow JSON
structural parse and local authored-link check is allowed; production manifest
admission will be checked by the implementation owner's tests and later review.
Publish `drawio-route-examples-result.md` with exact changed files/hashes,
actual check scope and pending integration boundary, then freeze. Root may
request a small final wording reconciliation after source acceptance.
