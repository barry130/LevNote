# Image reference format implementation

PLANNED until markdown-table-columns-release25-root-acceptance.md exists.
Then ACTIVE for DeepSeek /root/read_native_handoffs_raster_admission_finish_task_md.
Read markdown-image-reference-root-decision.md; it overrides both review options.
No child delegation, compact before200Kcontext, all Go through scripts/go.ps1.

Own formats/markdown/types.go,inline.go,export.go and a new focused image test
file; existing markdown_test.go changes only for new intended warning/descriptor
expectations. No table parser/planner, view/app/core/vendor/docs edits. Format
only files may be formatted, never package-wide fmt. Preserve .25 package/evidence.

Implement InlineImage exact range/raw fields and the bounded actionable subset
in the root decision, using existing work/span budgets. Accepted and rejected
forms leave complete source unchanged. Exact HTML escaped original image syntax
and warnings must be tested for both ExportHTMLChecked and bounded byte export.
Existing link behavior (including outer link with image-looking label) must
remain. Test forward progress and no partial action for malformed/budget cases.
Coordinate API freeze via handoff; Astra consumes the exact constant/fields.

Use substantive focused tests for syntax/escaping/nesting/newlines/ranges,
source preservation, boundary4096/4097, work/span limits, code/table/bold/outer
link ownership, literal %, Windows path bytes, unsafe target syntax versus host
responsibility, HTML no-img/no-src preservation, and unchanged ordinary links.
Capture exact command/stdout/stderr/exit files for runs from the start; keep
initial failures if any. Do not rerun merely to improve log formatting.

Root review addendum: new warning semantics also cross the existing Markdown
worker response boundary. You may add ONE focused cmd/levnote-markdown-export/
image_reference_test.go using the existing in-process callWorker helper. Assert
independent literal escaped HTML/no img, exact warning code/range and source
unchanged for accepted/rejected local/remote syntax. No worker runtime edit or
OS worker launch. Include this affected command package in final coordinated
test/vet once; do not run runtime/resource matrix tests.

After focused pass freeze source and write
native/handoffs/markdown-image-reference-format-result.md with concise behavior,
limitations, actual identities and evidence paths. App/view final integrated
verification will be separately coordinated; no package/launch/resource run.
