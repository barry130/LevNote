# .19 documentation acceptance

Accepted by root2026-11-10 after read-only review. The seven actual document
sizes/hashes match `drawio-worker-release19-docs-result.md` (4,081bytes,
SHA25606D24499879F0309669899E80A01DB9B82C90698233BAB7B78F66BC6343A380C).
All four review corrections are represented: initial Flush/owner capture in
app, Prepare/cache/page/epoch semantics, draft transaction effects separate
from clean export and original disk bytes, and the four bundled worker paths.
Required drawio page metadata and conditional empty ID are explicit.

The author's scoped347-target link check is documentation evidence only.
No test, build, application or worker was rerun by root. Immutable .19 runtime
acceptance remains `drawio-worker-release19-root-acceptance.md`; no custom
suffix capability or resource/physical-input claim was added to that binary.

These are accepted current .19 documents, not permanently frozen living docs.
A later assigned release-doc task may update the current guidance while
preserving dated historical acceptance and delivery evidence.

Subsequent read-only sweep found stale current-usage paragraphs in
native/README.md around126-152 (three workers/two exporters and host-generated
drawio SVG), despite the corrected introduction/architecture. This is a
documentation follow-up, not a changed binary claim; explicitly assigned in
`drawio-route-release20-docs-task.md`. Root does not rely on those stale lines
as evidence of runtime behavior. The next docs acceptance must close them.
