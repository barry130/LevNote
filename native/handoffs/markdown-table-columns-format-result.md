# Markdown table column format: result

Status: COMPLETE and frozen, 2026-11-10. Scope is format-only.

## Delivered

Added the shared column-edit contract and planner in
`native/formats/markdown/table_column_edit.go`:

- `TableColumnAction`, `TableColumnInsertBefore`,
  `TableColumnInsertAfter`, and `TableColumnDelete`.
- `TableColumnEdit`, `TableColumnPatch`, and
  `PlanTableColumnEdit(source, tableRange, anchorRowRange, column, action)`.
- `ErrTableColumnAction`, `ErrTableColumnTarget`, `ErrTableColumnLimit`,
  `ErrTableColumnContext`, and `ErrTableColumnMinimum`.

The planner reparses the authoritative source, verifies the exact table and
anchor identities, uses a zero-based selected column, checks the one/eight
limits against total table width, plans every header/delimiter/body row, and
reparses the complete candidate. Successful edits are ascending,
nonoverlapping original-source byte ranges for descending application. All
errors return the zero patch.

Surviving cells retain their raw range bytes, text range, padding, delimiter
spelling, indentation, row ending, and neighbors. New non-delimiter cells are
empty and new delimiter cells are exactly `---`. The planner adds only the
outer pipe boundaries needed to preserve those raw cells.

## Refusal and edge decisions

- Unsupported, ragged, partial, stale, malformed, non-UTF8, over-limit, or
  invalid-source targets refuse without a patch.
- Deleting the final column of a one-column table returns
  `ErrTableColumnMinimum`; inserting into an eight-column table returns
  `ErrTableColumnLimit`.
- A last cell ending in an odd backslash run refuses insertion with
  `ErrTableColumnContext`; an even run remains structural and is accepted.
- A bare shared marker `|` becomes `|||` for either insertion direction.
- Inserting before the leading-only padding cell `|  ` produces `||  |` so the
  original padding remains inside the surviving cell.
- Deleting the last column preserves an empty or padding-only retained middle
  cell by adding a trailing outer boundary:
  `|H||B` -> `|H||`, `|H|  |B` -> `|H|  |`.
- Candidate table growth is checked from the applied source delta before the
  candidate parse. A supported table at exactly 32 KiB refuses an insert whose
  candidate crosses that table limit with `ErrTableColumnLimit`, not through a
  parser fallback warning.

## Verification

Focused column matrix:

```text
.\native\scripts\go.ps1 -GoArguments @('test','./formats/markdown','-run','^TestTableColumn','-count=1','-timeout=120s','-v')
ok levnote.local/native/formats/markdown 0.531s
FOCUSED_EXIT=0
```

Final affected suite:

```text
.\native\scripts\go.ps1 -GoArguments @('test','./formats/markdown','-count=1','-timeout=120s')
ok levnote.local/native/formats/markdown 0.692s
FINAL_FULL_FORMAT_TEST_EXIT=0
```

Final vet:

```text
.\native\scripts\go.ps1 -GoArguments @('vet','./formats/markdown')
FINAL_FORMAT_VET_EXIT=0
```

The focused tests cover exact before/after/delete bytes; one, two, seven and
eight-column boundaries; every outer-pipe form; bare and padding-only markers;
empty-middle deletion; no-leading and indented rows; UTF-8 and escaped pipes;
odd/even backslash parity; LF/CRLF/CR/final-unterminated input; neighbor
blocks and warnings; descriptor promotion/demotion; exact 32 KiB refusal; and
zero-patch invalid action/source/range/table limits. The first compile and
failed oracle outputs remain recorded in
[markdown-table-columns-format-progress.md](markdown-table-columns-format-progress.md).
All 26 relevant command records from the implementation session, including
every failure and success block, are preserved in
[markdown-table-columns-format-evidence-transcript.md](markdown-table-columns-format-evidence-transcript.md).
That transcript records that no separate raw stdout/stderr files were saved,
uses the command-event output rather than reconstructing text, and identifies
the one long model display that the tool had truncated.

## Frozen identities

```text
native/formats/markdown/table_column_edit.go
18964 bytes
SHA256 8DEB679C85C1351A1119A16DB4B0B41295CDD1D34A1173E304EC84A72D7ECC4F

native/formats/markdown/table_column_edit_test.go
19752 bytes
SHA256 6A372B8E406FAC406D33803ABBF6367D0BF4BEC589B09F0AF0D72BAEBA213CCD
```

Evidence transcript:

```text
native/handoffs/markdown-table-columns-format-evidence-transcript.md
145345 bytes
SHA256 48FD3DE7B0CBBCF98899870F29E5002F3B9373B1268DCFC2E598ECDAD54D9DA3
```

An earlier package-wide formatting command incidentally reformatted frozen
`native/formats/markdown/table.go`. That change was disclosed, the file was
restored byte-for-byte, and no package-wide formatter was run afterward.

```text
current  native/formats/markdown/table.go
12689 bytes
SHA256 AF49F0DD750CB490C2E8E78B1ACA80F58B01739C691B347323CF19360B6FE898

baseline native/artifacts/acceptance/markdown-export-memory-20261109-024336-optimized/benchmark new/formats/markdown/table.go
12689 bytes
SHA256 AF49F0DD750CB490C2E8E78B1ACA80F58B01739C691B347323CF19360B6FE898
```

Current and baseline bytes match. No net parser change occurred; the
incidental formatter edit to `table.go` was disclosed and restored
byte-for-byte. No app, UI, core, module wiring, vendor, package, or product
documentation file was edited.
