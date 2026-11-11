# Active source documentation — accepted .7 readonly scene

2026-11-09. Astra owns this finite source-documentation step. Root accepted the
actual .7 delivery in `native-preview-delivery-root-acceptance.md`, including
all three captures. Finish your delivery handoff, then proceed now without an
additional source-only approval round.

Update README.md, ROADMAP.md, CHANGELOG.md, native/README.md,
native/VALIDATION.md, native/ARCHITECTURE.md and native/formats/API.md. No product,
tests, scripts, shipped bundle or retained evidence edits. Check links and
current-versus-historical wording by read-only inspection; no test/build repeat.

Use .7 launch paths and host + three fixed workers. Describe readonly Mindmap
parse/layout in its worker and neutral native host rendering, source-only full
attributes, Save As writable-copy inspector, no preview image/badge, separate
SVG badge behavior. Keep writable Mindmap/outline parsing and drawio in host.
Describe shared one-task lane, preview coalescing, export priority and stale
result guards. Missing worker refuses preview visibly with source access.

Keep actual limits (Markdown source8MiB, Mindmap source16MiB, preview DTO32MiB,
export results64MiB, label/cache/warning limits) distinct from process RSS.
Do not claim all source baseline unchanged: one test-helper delta is recorded.
Include two invocations/six requests with the first status-oracle failure
retained, five requests in passing oracles, final14/14 validation and three
native headless captures. Do not imply physical UI/IME/resources were rerun.

Replace obsolete current-scope statements that all previews run in host or
readonly Mindmap has editable inspector fields. Preserve historical .6 evidence
as history and avoid rewriting old results. Explain local JSON plugins still
extend source highlighting/outline only at this accepted version; schema3 is
next work, not shipped. The executable/interactive plugin boundary remains.

Write `native-preview-docs-completion.md` listing actual changed docs and any
limitations. Do not start route implementation until root reads this brief gate.
