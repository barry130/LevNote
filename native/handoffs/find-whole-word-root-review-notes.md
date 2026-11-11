# Root review notes2026-11-10

Actual light/dark760x480 captures viewed. Controls fit without overlap, full
Find controls and status visible; editor remains usable. Production wiring is
minimal and consistent with reviewed plan; no behavior defect found in it.

Independent reviewer: assess these narrow evidence/docs gaps; no broad new suite.

1. Cache test's `needle filler needle` fixture has only whole-word hits. It proves
   one cold scan on toggle but not the claimed filtered-offset change. Add an
   in-word hit to that existing fixture and assert offsets before/after toggle,
   retaining scan-count assertions. No new cache algorithm needed.
2. Replace Current test only replaces an eligible standalone cat. Add a selected
   in-identifier occurrence and assert WholeWord refuses it, preserving buffer,
   version and selection. Existing positive/Replace All/grouped undo remain.
3. Result says regex+case interaction uses same checkbox without changing text;
   actual table sets runtime booleans directly. Either accurately describe the
   combined evidence or add one small actual control interaction; don't fabricate
   UI coverage from method-level tests. Workspace fixture verifies shared options
   but directly assigns WholeWord too; use accurate wording.

Separate .16 docs review (same reviewer is docs owner): ARCHITECTURE says adapter
checks `首尾字节`, but output.go checks exact length and final pending LF, not an
initial byte. Tests compare full JSON with oracle; runtime doesn't hash-compare
oracle. Replace the sentence with precise runtime-vs-test wording.
Also distinguish prepublication write/adapter validation failures (old file
preserved) from syncDirectory failure after publication. Don't imply all errors
from the atomic lifecycle roll back a published file. No source change requested
for this docs correction, and no test/build/launch required.

Route concrete test corrections to implementation owner; reviewer may correct
only its six .16 docs/result within existing docs ownership after reporting the
review. Keep raw prior evidence/results and don't edit accepted package content.
