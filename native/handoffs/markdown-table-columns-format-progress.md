# Markdown table column format: progress checkpoint

2026-11-10, implementation checkpoint before final package verification.

The new planner API and tests exist only in:

- `native/formats/markdown/table_column_edit.go`
- `native/formats/markdown/table_column_edit_test.go`

Implemented the shared action/edit/patch API, authoritative pre/post parsing,
zero-based target validation, total-column minimum/maximum checks, exact raw
cell and delimiter preservation, structural outer-pipe additions, queued
ascending nonoverlapping edits, odd trailing-backslash refusal, and zero-patch
error returns.

Known edge corrections are complete:

- `InsertBefore` increments the candidate column count.
- A bare one-pipe row inserts as `|||` in either direction; an empty leading
  marker is distinguished from nonempty `| H` rows.
- Inserting before the one-column leading-only padding row `|  ` produces
  `||  |` so the original two-space cell range survives.
- Appending after an ordinary leading-only one-column row gains its trailing
  boundary, while a bare `|` shared marker becomes `|||` in either direction.
- Deleting the last column in a leading-only larger row preserves an empty or
  padding-only middle cell by adding the necessary trailing outer pipe, for
  example `|H||B` -> `|H||` and `|H|  |B` -> `|H|  |`.
- A normal append after `|H` gains the trailing boundary, and the seven-column
  exact fixture reaches eight columns with exact expected bytes.
- Odd trailing backslashes refuse, while an even run remains structural and is
  covered by an exact append-after fixture with unchanged surviving bytes.
- Candidate table growth is checked from the applied source delta before the
  candidate parse. A supported table whose source is exactly 32 KiB therefore
  returns `ErrTableColumnLimit` with a zero patch instead of being classified
  through the parser's fallback warning.
- Descriptor promotion/demotion fixtures exercise target tables below the
  eight-column limit and separate later descriptor budgets.
- Exact oracle spacing distinguishes all outer-pipe forms, including the
  required `|---|` and `|---|---|` spellings.

A focused run was attempted after the initial edge patch:

```text
.\native\scripts\go.ps1 -GoArguments @('test','./formats/markdown','-run','^TestTableColumn','-count=1','-timeout=120s','-v')
```

The first attempt stopped at a test-helper compile error while collecting the
planner's multi-edit row result. After fixing that test plumbing and the
concrete exact-output assertions, the focused matrix passes with exit 0,
including the later deletion-boundary and shared-marker additions. Temporary
diagnostic logging was removed.

Preserved first focused failures (abridged only around the reported line):

```text
# levnote.local/native/formats/markdown [levnote.local/native/formats/markdown.test]
formats\markdown\table_column_edit_test.go:231:30: cannot use edit (variable of type []TableColumnEdit) as TableColumnEdit value in argument to append
FAIL
FOCUSED_EXIT=1
```

After repairing that test helper, the first concrete edge oracle failed while
`|  ` insertion was still being validated:

```text
debug candidate="||  |\n|---|---|\n||  |\n"
PlanTableColumnEdit: table column edit cannot preserve the complete source structure
FAIL
FOCUSED_EXIT=1
```

Once planning succeeded, the first candidate-byte assertion exposed the stale
expected outer boundary:

```text
candidate bytes:
 got "||  |\n|---|---|\n||  |\n"
want "||  \n|---|---|\n||  \n"
FAIL
FOCUSED_EXIT=1
```

The exact-limit fixture then exposed why parser fallback was unsuitable for
error classification:

```text
exact-limit source=32768 candidate=33029 edits=258 afterBlocks=1 partial=false
firstProblem="table exceeds the 32 KiB preview limit; complete source remains visible"
candidate-over-exact-limit patch={Edits:[]} err=table column edit cannot preserve the complete source structure
FAIL
EXACT_LIMIT_TEST_EXIT=1
```

The final pre-parse size guard corrects that classification.

During earlier package-wide formatting, frozen `table.go` was reformatted. It
was restored from the frozen comparison copy and is currently SHA256
`AF49F0DD750CB490C2E8E78B1ACA80F58B01739C691B347323CF19360B6FE898`.
No further package-wide formatting will run.

The final focused, full-format and vet commands pass with exit 0 after the
last parity fixture. After the accepted source freeze, the implementation
thread's command records were preserved in
[markdown-table-columns-format-evidence-transcript.md](markdown-table-columns-format-evidence-transcript.md).
No separate raw stdout/stderr log files were saved; the transcript copies the
available unified command-event output and labels the one long display that
the tool truncated.
