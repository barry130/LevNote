# Table columns: active format implementation

ACTIVE2026-11-10. Owner DeepSeek raster_admission_finish. .24 is accepted and
workspace source freeze released for this scope. No child delegation. Read
native/AGENTS.md, markdown-table-columns-next-plan.md and any saved independent
review; the concrete corrections here override ambiguous statements in that plan.
Do the implementation, not another planning phase.

Own new formats/markdown/table_column_edit.go and table_column_edit_test.go only.
Reuse the existing parser and preservation helpers without changing their source.
Existing row implementation is table_edit.go. Do not touch the UI/app/docs,
accepted package/evidence, parser, text.Buffer, dependencies or .24 portable file.

Contract shared with Astra UI owner:

- TableColumnAction uint8, TableColumnInsertBefore/TableColumnInsertAfter/
  TableColumnDelete constants, starting1.
- TableColumnEdit { Range SourceRange; Text string }.
- TableColumnPatch { Edits []TableColumnEdit }.
- PlanTableColumnEdit(source string, tableRange, anchorRowRange SourceRange,
  column int, action TableColumnAction) (TableColumnPatch,error).
- Sentinel errors ErrTableColumnAction, ErrTableColumnTarget,
  ErrTableColumnLimit, ErrTableColumnContext, ErrTableColumnMinimum.
- column is a zero-based selected index; min1/max8 checks use total column
  count, never selected index. Error returns zero patch.
- Successful edits are original-source byte ranges, nonoverlapping, ordered
  ascending. Merge same-position edits deterministically inside the planner.
  Caller applies descending. No reserialization and no line-ending edits.

Safety and preservation decisions:

1. Full authoritative Parse before and candidate Parse after, exact table/anchor
   identities, invalid UTF8/partial/limits/stale/ragged/unsupported rejection.
   Validate the header, delimiter and each body row; retain all existing bounds.
2. Preserve every surviving cell's raw Range bytes INCLUDING padding, TextRange
   text, delimiter spelling/alignment, row line ending and original indentation.
   Neighbor blocks, other-table fallback state and warnings retain semantics
   with correctly translated offsets. Reuse alignmentBlockPreserved where valid.
3. Only necessary structural outer pipes may be added at the original indent or
   line edge. Empty new header/body cells contain no text; new delimiter is ---.
   No new padding inside an existing cell and no escaping/normalizing its bytes.
4. Before-first insertion into no-leading rows needs a new leading outer pipe
   as well as the new column separator. A single pipe would swallow the empty
   cell as an outer marker. Same care for a new empty last cell without a
   trailing pipe. Prove retained cells/raw through candidate descriptors.
5. If an unchanged final cell ends with an odd backslash run, newly appended
   separator may be escaped. Refuse with a context error if strict raw-byte
   preservation cannot represent the operation. Do not add padding silently.
6. First-column deletion in no-leading rows may expose >3 spaces or block
   syntax. It is permissible to add a leading outer pipe at the original indent
   so surviving raw cells retain exact padding. For two->one in any no-leading
   row, add the required leading boundary; preserve any existing outer marker.
   If a candidate still cannot prove the contract, return context refusal.
7. Retain all surviving delimiter raw cells byte-for-byte, not merely semantic
   alignment. Existing parser needs >=3 hyphens: plan examples :-- etc are not
   supported markers; use :--- / ---: / :---: and longer runs in valid fixtures.

Tests: meaningful exact-output cases for before/after/delete, all outer-pipe
forms including single-pipe empty cells, first/middle/last, one/two/seven/eight
columns, Unicode/escaped pipes/backslash parity, indentation and exposed block
starts, LF/CRLF/CR/missing final newline, unchanged raw padding/alignment,
neighbor structures/warnings and descriptor-budget promotion/demotion.
Malformed identity/action/source and every relevant limit must return zero
patch. Expected bytes independent from the implementation, not a second copy
of its algorithm. No new parser support. Preserve any real failure logs.

Run focused new format tests while implementing via scripts/go.ps1 only; once
stable run one full formats/markdown test and vet, record exact commands/exits.
No app/view suite, builds, hosts, captures or benchmarks. Write a short
markdown-table-columns-format-progress.md after first substantive milestone,
then markdown-table-columns-format-result.md with API/refusal decisions,
actual source identities and evidence links. Freeze these files for integration.
Coordinate through task/progress files if collaboration payloads are unreadable.
