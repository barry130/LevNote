# Implement the fixed Mindmap SVG worker

Status: ACTIVE 2026-11-09. Owner read_mindmap_worker_tests_task (GLM); the
previous DeepSeek owner is interrupted. Main source exists and was independently
reviewed; tests/evidence remain. Read native/mindmap_worker_tests_task.txt first.
Read native/AGENTS.md and mindmap-engine-integration-next-stage.md worker scope.
Implement, test and publish a finite command; no architecture exploration task.

Own ONLY new native/cmd/levnote-mindmap-svg/main.go and its tests, plus
native/handoffs/mindmap-engine-worker.md and new evidence. No other product edits.
Root plans/reviews; Astra owns runtime/app; GLM owns catalog/IPC final tests.
No child agents. Compact before 200K context. Work is local-only.

Use frozen APIs: engineipc.ReadRequestFor(engineipc.MindmapSVG,r), existing
WriteResponse/NewResponseDecoder; catalog.Lookup(ID) returns fixed descriptor.
Pure mindmap.NativeLayoutOptions(), PresentationIssues(doc), ExportSVG(doc,
layout,maxBytes) are ready. Source16MiB, default parser depth256/nodes100000,
SVG64MiB. Zero production limit selects hard cap; private small-cap test seam
only. No flags/env/config paths/files/network/subprocesses. engineVersion link
flag like existing Markdown command. EOF and identity strictness come from IPC.

Keep warnings deterministic in parse,layout,presentation order; no duplicates
from doc.Issues. Whole-document ranges [0,len(source)] for all no-location
issues; do not invent node offsets. >256warnings or too-long code/message must
explicitly refuse warning-limit, not truncate. Use stable non-source error
messages. Parse size/depth/node limits refuse with source-limit, invalid source
or schema render-failed, layout severityError refuses no partial bytes. Check
warning bounds before SVG assembly. SVG ErrSVGOutputLimit maps output-limit.
Preserve source exactly; don't strip a literal BOM from document buffer (file
encoding BOM is handled by document.Open upstream). No full file I/O in command.

Tests must be meaningful: actual serve wire with independently asserted XML/
label/geometry/escaping bytes, native parity as secondary oracle, source
nonmutation, deterministic whole-doc warnings including hidden metadata,
unknown layout/default layout, icons/tags/paint/image disclosure, invalid/deep
input, >256warnings and oversized warning message, tiny SVG output cap with
escaped label, foreign request no payload read, failed/short writer.
Use small fixtures and seam bounds; don't allocate huge source merely to assert
header preflight. Existing pure parser limit tests may be cited, not duplicated.

Publish a brief source/API checkpoint promptly, then complete command tests and
vet once when GLM's IPC freeze is confirmed. Preserve actual commands, streams,
exit/time, owned and dependency source fingerprints before/after. No executable
build or real process launch; serve is exercised in-process. Root reviews actual
source/oracles/logs before a later bundle is built. Do not change final launch
documentation yet. Current accepted runnable release is native-worker.20261109.5.
