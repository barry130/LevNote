# Markdown table columns: full available evidence transcript

2026-11-10. Metadata-only evidence completion for the accepted and frozen format implementation.

## Provenance and completeness

No test runner wrote a dedicated stdout or stderr log file to the workspace. The exact process records still exist in the implementation thread's Codex session JSONL:

`C:\Users\canace\.codex\sessions\2026\11\10\rollout-2026-11-10T10-34-52-01a1c34e-ec81-7f63-b207-269a9757d86e.jsonl`

The command events below are the records whose ordinals are 6708 through 7335. The selection includes every actual `go.ps1` invocation against `./formats/markdown` in that interval: one compile-only check, one formatter command, every focused test attempt, every full-format-suite attempt, both intermediate vet attempts, and the final vet attempt. The accidental WSL command that failed before Go started is included as an attempt.

Each `command argv` is the exact JSON command array recorded by the executor. Each `effective command` is the `-Command` or `-lc` script passed to PowerShell or WSL. Each output block is copied from the event item's `aggregated_output` field. Unified execution combined stdout and stderr; separate stream logs are unavailable.

- `SESSION-RECORDED OUTPUT` means the event itself contains the output block reproduced here. It is not reconstructed from memory.
- `LONG DISPLAY WAS TRUNCATED` means the interactive tool wrapper around that event reported truncation. The longer `aggregated_output` stored on the command event is still reproduced here; this distinction is retained so nobody mistakes the prior chat display for the process capture.
- The JSONL snapshot used by this evidence write contained 26 selected command records.
- Output text is written as UTF-8 bytes without newline translation. The failed WSL attempt retains the replacement characters and NUL bytes present in the JSONL event; those are session-decoding artifacts, not reconstructed output.

## Attempt index

| # | Record ordinal | Kind | Process status | Exit | Display note |
|---:|---:|---|---|---:|---|
| 1 | 6708 | compile-only test command | `completed` | `0` | SESSION-RECORDED OUTPUT |
| 2 | 6736 | format | `completed` | `0` | SESSION-RECORDED OUTPUT |
| 3 | 6806 | focused test | `failed` | `1` | SESSION-RECORDED OUTPUT |
| 4 | 6827 | focused test | `failed` | `1` | SESSION-RECORDED OUTPUT |
| 5 | 6846 | full format suite | `failed` | `1` | SESSION-RECORDED OUTPUT |
| 6 | 6888 | focused test | `failed` | `1` | SESSION-RECORDED OUTPUT |
| 7 | 6911 | focused test | `failed` | `1` | SESSION-RECORDED OUTPUT |
| 8 | 6939 | focused test | `failed` | `1` | SESSION-RECORDED OUTPUT |
| 9 | 6960 | focused test | `failed` | `1` | SESSION-RECORDED OUTPUT |
| 10 | 6974 | focused test | `completed` | `0` | SESSION-RECORDED OUTPUT |
| 11 | 7032 | focused test | `failed` | `1` | SESSION-RECORDED OUTPUT |
| 12 | 7046 | focused test | `completed` | `0` | SESSION-RECORDED OUTPUT |
| 13 | 7062 | full format suite | `completed` | `0` | SESSION-RECORDED OUTPUT |
| 14 | 7069 | vet | `completed` | `0` | SESSION-RECORDED OUTPUT |
| 15 | 7108 | focused test | `failed` | `1` | SESSION-RECORDED OUTPUT |
| 16 | 7122 | focused test | `failed` | `1` | LONG DISPLAY WAS TRUNCATED |
| 17 | 7164 | full format suite | `failed` | `1` | SESSION-RECORDED OUTPUT |
| 18 | 7171 | full format suite | `failed` | `1` | SESSION-RECORDED OUTPUT |
| 19 | 7185 | full format suite | `failed` | `1` | SESSION-RECORDED OUTPUT |
| 20 | 7200 | full format suite | `failed` | `1` | SESSION-RECORDED OUTPUT |
| 21 | 7249 | focused test | `completed` | `0` | SESSION-RECORDED OUTPUT |
| 22 | 7265 | full format suite | `completed` | `0` | SESSION-RECORDED OUTPUT |
| 23 | 7272 | vet | `completed` | `0` | SESSION-RECORDED OUTPUT |
| 24 | 7323 | focused test | `completed` | `0` | SESSION-RECORDED OUTPUT |
| 25 | 7330 | full format suite | `completed` | `0` | SESSION-RECORDED OUTPUT |
| 26 | 7335 | vet | `completed` | `0` | SESSION-RECORDED OUTPUT |

A failed exit is part of the evidence. No failure block below was rerun or replaced; later passing commands appear as separate attempts.

## Exact command events

### Attempt 1: record ordinal 6708 (compile-only test command)

- Call ID: `call_00_uINVJUKCsOIyxsaGViuZ1055`
- Process status: `completed`
- Exit code: `0`
- Captured output length: `68` characters
- Display completeness: `SESSION-RECORDED OUTPUT`

Command argv:

~~~json
["C:\\Users\\canace\\.cache\\codex-runtimes\\codex-primary-runtime\\dependencies\\native\\powershell\\pwsh.exe", "-Command", "& '.\\native\\scripts\\go.ps1' -GoArguments @('test','./formats/markdown','-run','^$','-count=1')"]
~~~

Effective command:

~~~powershell
& '.\native\scripts\go.ps1' -GoArguments @('test','./formats/markdown','-run','^$','-count=1')
~~~

Session-recorded output:

~~~text
ok  	levnote.local/native/formats/markdown	0.496s [no tests to run]

~~~

### Attempt 2: record ordinal 6736 (format)

- Call ID: `call_00_yXHb0KDccymIhhczyVmj9311`
- Process status: `completed`
- Exit code: `0`
- Captured output length: `64` characters
- Display completeness: `SESSION-RECORDED OUTPUT`

Command argv:

~~~json
["C:\\Users\\canace\\.cache\\codex-runtimes\\codex-primary-runtime\\dependencies\\native\\powershell\\pwsh.exe", "-Command", "& '.\\native\\scripts\\go.ps1' -GoArguments @('fmt','./formats/markdown')"]
~~~

Effective command:

~~~powershell
& '.\native\scripts\go.ps1' -GoArguments @('fmt','./formats/markdown')
~~~

Session-recorded output:

~~~text
formats\markdown\table.go
formats\markdown\table_column_edit.go

~~~

### Attempt 3: record ordinal 6806 (focused test)

- Call ID: `call_00_E3mAvlx43SbGjBJKez2y5135`
- Process status: `failed`
- Exit code: `1`
- Captured output length: `462` characters
- Display completeness: `SESSION-RECORDED OUTPUT`

Command argv:

~~~json
["C:\\Users\\canace\\.cache\\codex-runtimes\\codex-primary-runtime\\dependencies\\native\\powershell\\pwsh.exe", "-Command", "& '.\\native\\scripts\\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumn','-count=1','-timeout=120s','-v')"]
~~~

Effective command:

~~~powershell
& '.\native\scripts\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumn','-count=1','-timeout=120s','-v')
~~~

Session-recorded output:

~~~text
# levnote.local/native/formats/markdown [levnote.local/native/formats/markdown.test]
formats\markdown\table_column_edit_test.go:326:48: invalid operation: patch != (TableColumnPatch{}) (struct containing []TableColumnEdit cannot be compared)
formats\markdown\table_column_edit_test.go:339:31: invalid operation: patch != (TableColumnPatch{}) (struct containing []TableColumnEdit cannot be compared)
FAIL	levnote.local/native/formats/markdown [build failed]
FAIL

~~~

### Attempt 4: record ordinal 6827 (focused test)

- Call ID: `call_00_pcR9k6GRvggEeD7AnFT14550`
- Process status: `failed`
- Exit code: `1`
- Captured output length: `7313` characters
- Display completeness: `SESSION-RECORDED OUTPUT`

Command argv:

~~~json
["C:\\Users\\canace\\.cache\\codex-runtimes\\codex-primary-runtime\\dependencies\\native\\powershell\\pwsh.exe", "-Command", "& '.\\native\\scripts\\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumn','-count=1','-timeout=120s','-v')"]
~~~

Effective command:

~~~powershell
& '.\native\scripts\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumn','-count=1','-timeout=120s','-v')
~~~

Session-recorded output:

~~~text
=== RUN   TestTableColumnPlanExactBytes
=== RUN   TestTableColumnPlanExactBytes/outer-before-middle
=== RUN   TestTableColumnPlanExactBytes/outer-after-last
    table_column_edit_test.go:238: candidate bytes:
         got "| H1 | H2 ||\n| --- | --- |---|\n| A | B ||\n"
        want "| H1 | H2 ||\n| --- | ---|---|\n| A | B ||\n"
=== RUN   TestTableColumnPlanExactBytes/outer-delete-middle
=== RUN   TestTableColumnPlanExactBytes/no-outer-before-first
=== RUN   TestTableColumnPlanExactBytes/no-outer-after-last
    table_column_edit_test.go:238: candidate bytes:
         got "H1 | H2 | H3||\n:--- | ---: | :---:|---|\nA | B | C||"
        want "H1 | H2 | H3||\n:--- | ---: | :---: |---|\nA | B | C||"
=== RUN   TestTableColumnPlanExactBytes/no-outer-delete-middle
=== RUN   TestTableColumnPlanExactBytes/leading-only-insert-last
=== RUN   TestTableColumnPlanExactBytes/trailing-only-insert-first
    table_column_edit_test.go:238: candidate bytes:
         got "||H1 | H2 |\n|---|--- | --- |\n||A | B |\n"
        want "||H1 | H2 |\n|---| --- | --- |\n||A | B |\n"
=== RUN   TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-first
=== RUN   TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-last
=== RUN   TestTableColumnPlanExactBytes/indented-no-leading-delete-first
=== RUN   TestTableColumnPlanExactBytes/single-column-normal-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-leading-only-insert-before
    table_column_edit_test.go:238: candidate bytes:
         got "|| A\n|---|---\n|| C\n"
        want "|| A\n|---|----\n|| C\n"
=== RUN   TestTableColumnPlanExactBytes/single-column-leading-only-insert-after
=== RUN   TestTableColumnPlanExactBytes/single-column-degenerate-insert-after
=== RUN   TestTableColumnPlanExactBytes/single-column-padding-insert-before
    table_column_edit_test.go:225: PlanTableColumnEdit: table column edit cannot preserve the complete source structure
=== RUN   TestTableColumnPlanExactBytes/single-column-padding-insert-after
    table_column_edit_test.go:225: PlanTableColumnEdit: table column edit cannot preserve the complete source structure
=== RUN   TestTableColumnPlanExactBytes/escaped-pipe-and-unicode-delete-first
--- FAIL: TestTableColumnPlanExactBytes (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-before-middle (0.00s)
    --- FAIL: TestTableColumnPlanExactBytes/outer-after-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-delete-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-before-first (0.00s)
    --- FAIL: TestTableColumnPlanExactBytes/no-outer-after-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-delete-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/leading-only-insert-last (0.00s)
    --- FAIL: TestTableColumnPlanExactBytes/trailing-only-insert-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/indented-no-leading-delete-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-normal-insert-before (0.00s)
    --- FAIL: TestTableColumnPlanExactBytes/single-column-leading-only-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-leading-only-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-degenerate-insert-after (0.00s)
    --- FAIL: TestTableColumnPlanExactBytes/single-column-padding-insert-before (0.00s)
    --- FAIL: TestTableColumnPlanExactBytes/single-column-padding-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/escaped-pipe-and-unicode-delete-first (0.00s)
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/LF
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/_LF
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/_
--- PASS: TestTableColumnPlanNewlineFormsAndNeighbors (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/LF (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/_LF (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/_ (0.00s)
=== RUN   TestTableColumnPlanWarningAndDescriptorState
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-warning-remains
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-demotion-refused
    table_column_edit_test.go:327: descriptor demotion patch={Edits:[]} err=table column edit exceeds a source or table limit
--- FAIL: TestTableColumnPlanWarningAndDescriptorState (0.01s)
    --- PASS: TestTableColumnPlanWarningAndDescriptorState/neighbor-warning-remains (0.00s)
    --- FAIL: TestTableColumnPlanWarningAndDescriptorState/neighbor-demotion-refused (0.01s)
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/unknown-action
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/invalid-utf8
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-table
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-anchor
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/column-bounds
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/one-column-minimum
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/eight-column-maximum
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/ragged-and-unsupported
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/odd-trailing-backslash
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/candidate-over-document
    table_column_edit_test.go:406: fixture is partial: [{Code:too-large Message:Markdown over 8 MiB is shown as raw text Range:{Start:0 End:8388609}}]
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/partial-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-table-bytes
--- FAIL: TestTableColumnPlanRefusesInvalidTargetsAndLimits (0.04s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/unknown-action (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/invalid-utf8 (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-table (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-anchor (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/column-bounds (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/one-column-minimum (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/eight-column-maximum (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/ragged-and-unsupported (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/odd-trailing-backslash (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-document (0.00s)
    --- FAIL: TestTableColumnPlanRefusesInvalidTargetsAndLimits/candidate-over-document (0.01s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/partial-document (0.01s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-table-bytes (0.02s)
FAIL
FAIL	levnote.local/native/formats/markdown	0.654s
FAIL

~~~

### Attempt 5: record ordinal 6846 (full format suite)

- Call ID: `call_00_OhTQX2mqlGSSsO0HQzEu9469`
- Process status: `failed`
- Exit code: `1`
- Captured output length: `3276` characters
- Display completeness: `SESSION-RECORDED OUTPUT`

Command argv:

~~~json
["C:\\Users\\canace\\.cache\\codex-runtimes\\codex-primary-runtime\\dependencies\\native\\powershell\\pwsh.exe", "-Command", "& '.\\native\\scripts\\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumnPlanExactBytes$','-count=1','-timeout=120s','-v')"]
~~~

Effective command:

~~~powershell
& '.\native\scripts\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumnPlanExactBytes$','-count=1','-timeout=120s','-v')
~~~

Session-recorded output:

~~~text
=== RUN   TestTableColumnPlanExactBytes
=== RUN   TestTableColumnPlanExactBytes/outer-before-middle
=== RUN   TestTableColumnPlanExactBytes/outer-after-last
=== RUN   TestTableColumnPlanExactBytes/outer-delete-middle
=== RUN   TestTableColumnPlanExactBytes/no-outer-before-first
=== RUN   TestTableColumnPlanExactBytes/no-outer-after-last
=== RUN   TestTableColumnPlanExactBytes/no-outer-delete-middle
=== RUN   TestTableColumnPlanExactBytes/leading-only-insert-last
=== RUN   TestTableColumnPlanExactBytes/trailing-only-insert-first
=== RUN   TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-first
=== RUN   TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-last
=== RUN   TestTableColumnPlanExactBytes/indented-no-leading-delete-first
=== RUN   TestTableColumnPlanExactBytes/single-column-normal-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-leading-only-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-leading-only-insert-after
=== RUN   TestTableColumnPlanExactBytes/single-column-degenerate-insert-after
=== RUN   TestTableColumnPlanExactBytes/single-column-padding-insert-before
    table_column_edit_test.go:235: debug candidate="||  \n|---|---|\n||  \n"
    table_column_edit_test.go:237: PlanTableColumnEdit: table column edit cannot preserve the complete source structure
=== RUN   TestTableColumnPlanExactBytes/single-column-padding-insert-after
    table_column_edit_test.go:235: debug candidate="|  |\n|---|---|\n|  |\n"
    table_column_edit_test.go:237: PlanTableColumnEdit: table column edit cannot preserve the complete source structure
=== RUN   TestTableColumnPlanExactBytes/escaped-pipe-and-unicode-delete-first
--- FAIL: TestTableColumnPlanExactBytes (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-before-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-after-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-delete-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-before-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-after-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-delete-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/leading-only-insert-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/trailing-only-insert-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/indented-no-leading-delete-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-normal-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-leading-only-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-leading-only-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-degenerate-insert-after (0.00s)
    --- FAIL: TestTableColumnPlanExactBytes/single-column-padding-insert-before (0.00s)
    --- FAIL: TestTableColumnPlanExactBytes/single-column-padding-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/escaped-pipe-and-unicode-delete-first (0.00s)
FAIL
FAIL	levnote.local/native/formats/markdown	0.688s
FAIL

~~~

### Attempt 6: record ordinal 6888 (focused test)

- Call ID: `call_00_U8RsKfEIJ3twdHujRkZa3837`
- Process status: `failed`
- Exit code: `1`
- Captured output length: `315` characters
- Display completeness: `SESSION-RECORDED OUTPUT`

Command argv:

~~~json
["C:\\Users\\canace\\.cache\\codex-runtimes\\codex-primary-runtime\\dependencies\\native\\powershell\\pwsh.exe", "-Command", "& '.\\native\\scripts\\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumn','-count=1','-timeout=120s','-v'); $exit=$LASTEXITCODE; Write-Output \"FOCUSED_EXIT=$exit\"; exit $exit"]
~~~

Effective command:

~~~powershell
& '.\native\scripts\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumn','-count=1','-timeout=120s','-v'); $exit=$LASTEXITCODE; Write-Output "FOCUSED_EXIT=$exit"; exit $exit
~~~

Session-recorded output:

~~~text
# levnote.local/native/formats/markdown [levnote.local/native/formats/markdown.test]
formats\markdown\table_column_edit_test.go:231:30: cannot use edit (variable of type []TableColumnEdit) as TableColumnEdit value in argument to append
FAIL	levnote.local/native/formats/markdown [build failed]
FAIL
FOCUSED_EXIT=1

~~~

### Attempt 7: record ordinal 6911 (focused test)

- Call ID: `call_00_vR8Bjpl99W6MJn6TVVtt5933`
- Process status: `failed`
- Exit code: `1`
- Captured output length: `6525` characters
- Display completeness: `SESSION-RECORDED OUTPUT`

Command argv:

~~~json
["C:\\Users\\canace\\.cache\\codex-runtimes\\codex-primary-runtime\\dependencies\\native\\powershell\\pwsh.exe", "-Command", "& '.\\native\\scripts\\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumn','-count=1','-timeout=120s','-v'); $exit=$LASTEXITCODE; Write-Output \"FOCUSED_EXIT=$exit\"; exit $exit"]
~~~

Effective command:

~~~powershell
& '.\native\scripts\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumn','-count=1','-timeout=120s','-v'); $exit=$LASTEXITCODE; Write-Output "FOCUSED_EXIT=$exit"; exit $exit
~~~

Session-recorded output:

~~~text
=== RUN   TestTableColumnPlanExactBytes
=== RUN   TestTableColumnPlanExactBytes/outer-before-middle
=== RUN   TestTableColumnPlanExactBytes/outer-after-last
=== RUN   TestTableColumnPlanExactBytes/outer-delete-middle
=== RUN   TestTableColumnPlanExactBytes/no-outer-before-first
=== RUN   TestTableColumnPlanExactBytes/no-outer-after-last
=== RUN   TestTableColumnPlanExactBytes/no-outer-delete-middle
=== RUN   TestTableColumnPlanExactBytes/leading-only-insert-last
=== RUN   TestTableColumnPlanExactBytes/trailing-only-insert-first
=== RUN   TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-first
=== RUN   TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-last
=== RUN   TestTableColumnPlanExactBytes/indented-no-leading-delete-first
=== RUN   TestTableColumnPlanExactBytes/single-column-normal-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-leading-only-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-leading-only-insert-after
=== RUN   TestTableColumnPlanExactBytes/single-column-degenerate-insert-after
=== RUN   TestTableColumnPlanExactBytes/single-column-padding-insert-before
    table_column_edit_test.go:235: debug candidate="||  |\n|---|---|\n||  |\n"
    table_column_edit_test.go:237: PlanTableColumnEdit: table column edit cannot preserve the complete source structure
=== RUN   TestTableColumnPlanExactBytes/single-column-padding-insert-after
=== RUN   TestTableColumnPlanExactBytes/escaped-pipe-and-unicode-delete-first
--- FAIL: TestTableColumnPlanExactBytes (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-before-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-after-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-delete-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-before-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-after-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-delete-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/leading-only-insert-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/trailing-only-insert-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/indented-no-leading-delete-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-normal-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-leading-only-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-leading-only-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-degenerate-insert-after (0.00s)
    --- FAIL: TestTableColumnPlanExactBytes/single-column-padding-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-padding-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/escaped-pipe-and-unicode-delete-first (0.00s)
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/LF
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/_LF
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/_
--- PASS: TestTableColumnPlanNewlineFormsAndNeighbors (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/LF (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/_LF (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/_ (0.00s)
=== RUN   TestTableColumnPlanWarningAndDescriptorState
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-warning-remains
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-demotion-refused
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-promotion-refused
--- PASS: TestTableColumnPlanWarningAndDescriptorState (0.01s)
    --- PASS: TestTableColumnPlanWarningAndDescriptorState/neighbor-warning-remains (0.00s)
    --- PASS: TestTableColumnPlanWarningAndDescriptorState/neighbor-demotion-refused (0.00s)
    --- PASS: TestTableColumnPlanWarningAndDescriptorState/neighbor-promotion-refused (0.00s)
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/unknown-action
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/invalid-utf8
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-table
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-anchor
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/column-bounds
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/one-column-minimum
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/eight-column-maximum
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/ragged-and-unsupported
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/odd-trailing-backslash
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/candidate-over-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/partial-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-table-bytes
--- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits (0.05s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/unknown-action (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/invalid-utf8 (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-table (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-anchor (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/column-bounds (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/one-column-minimum (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/eight-column-maximum (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/ragged-and-unsupported (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/odd-trailing-backslash (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-document (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/candidate-over-document (0.02s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/partial-document (0.01s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-table-bytes (0.02s)
FAIL
FAIL	levnote.local/native/formats/markdown	0.685s
FAIL
FOCUSED_EXIT=1

~~~

### Attempt 8: record ordinal 6939 (focused test)

- Call ID: `call_00_81xl6myvp1IkxYtcBKme1849`
- Process status: `failed`
- Exit code: `1`
- Captured output length: `6458` characters
- Display completeness: `SESSION-RECORDED OUTPUT`

Command argv:

~~~json
["C:\\Users\\canace\\.cache\\codex-runtimes\\codex-primary-runtime\\dependencies\\native\\powershell\\pwsh.exe", "-Command", "& '.\\native\\scripts\\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumn','-count=1','-timeout=120s','-v'); $exit=$LASTEXITCODE; Write-Output \"FOCUSED_EXIT=$exit\"; exit $exit"]
~~~

Effective command:

~~~powershell
& '.\native\scripts\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumn','-count=1','-timeout=120s','-v'); $exit=$LASTEXITCODE; Write-Output "FOCUSED_EXIT=$exit"; exit $exit
~~~

Session-recorded output:

~~~text
=== RUN   TestTableColumnPlanExactBytes
=== RUN   TestTableColumnPlanExactBytes/outer-before-middle
=== RUN   TestTableColumnPlanExactBytes/outer-after-last
=== RUN   TestTableColumnPlanExactBytes/outer-delete-middle
=== RUN   TestTableColumnPlanExactBytes/no-outer-before-first
=== RUN   TestTableColumnPlanExactBytes/no-outer-after-last
=== RUN   TestTableColumnPlanExactBytes/no-outer-delete-middle
=== RUN   TestTableColumnPlanExactBytes/leading-only-insert-last
=== RUN   TestTableColumnPlanExactBytes/trailing-only-insert-first
=== RUN   TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-first
=== RUN   TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-last
=== RUN   TestTableColumnPlanExactBytes/indented-no-leading-delete-first
=== RUN   TestTableColumnPlanExactBytes/single-column-normal-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-leading-only-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-leading-only-insert-after
=== RUN   TestTableColumnPlanExactBytes/single-column-degenerate-insert-after
=== RUN   TestTableColumnPlanExactBytes/single-column-padding-insert-before
    table_column_edit_test.go:250: candidate bytes:
         got "||  |\n|---|---|\n||  |\n"
        want "||  \n|---|---|\n||  \n"
=== RUN   TestTableColumnPlanExactBytes/single-column-padding-insert-after
=== RUN   TestTableColumnPlanExactBytes/escaped-pipe-and-unicode-delete-first
--- FAIL: TestTableColumnPlanExactBytes (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-before-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-after-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-delete-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-before-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-after-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-delete-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/leading-only-insert-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/trailing-only-insert-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/indented-no-leading-delete-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-normal-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-leading-only-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-leading-only-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-degenerate-insert-after (0.00s)
    --- FAIL: TestTableColumnPlanExactBytes/single-column-padding-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-padding-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/escaped-pipe-and-unicode-delete-first (0.00s)
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/LF
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/_LF
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/_
--- PASS: TestTableColumnPlanNewlineFormsAndNeighbors (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/LF (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/_LF (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/_ (0.00s)
=== RUN   TestTableColumnPlanWarningAndDescriptorState
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-warning-remains
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-demotion-refused
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-promotion-refused
--- PASS: TestTableColumnPlanWarningAndDescriptorState (0.01s)
    --- PASS: TestTableColumnPlanWarningAndDescriptorState/neighbor-warning-remains (0.00s)
    --- PASS: TestTableColumnPlanWarningAndDescriptorState/neighbor-demotion-refused (0.01s)
    --- PASS: TestTableColumnPlanWarningAndDescriptorState/neighbor-promotion-refused (0.00s)
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/unknown-action
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/invalid-utf8
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-table
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-anchor
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/column-bounds
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/one-column-minimum
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/eight-column-maximum
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/ragged-and-unsupported
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/odd-trailing-backslash
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/candidate-over-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/partial-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-table-bytes
--- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits (0.05s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/unknown-action (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/invalid-utf8 (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-table (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-anchor (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/column-bounds (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/one-column-minimum (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/eight-column-maximum (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/ragged-and-unsupported (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/odd-trailing-backslash (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-document (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/candidate-over-document (0.02s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/partial-document (0.01s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-table-bytes (0.01s)
FAIL
FAIL	levnote.local/native/formats/markdown	0.546s
FAIL
FOCUSED_EXIT=1

~~~

### Attempt 9: record ordinal 6960 (focused test)

- Call ID: `call_00_ZiRRqv0BxxXnSy4sM2Xu5693`
- Process status: `failed`
- Exit code: `1`
- Captured output length: `6616` characters
- Display completeness: `SESSION-RECORDED OUTPUT`

Command argv:

~~~json
["C:\\Users\\canace\\.cache\\codex-runtimes\\codex-primary-runtime\\dependencies\\native\\powershell\\pwsh.exe", "-Command", "& '.\\native\\scripts\\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumn','-count=1','-timeout=120s','-v'); $exit=$LASTEXITCODE; Write-Output \"FOCUSED_EXIT=$exit\"; exit $exit"]
~~~

Effective command:

~~~powershell
& '.\native\scripts\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumn','-count=1','-timeout=120s','-v'); $exit=$LASTEXITCODE; Write-Output "FOCUSED_EXIT=$exit"; exit $exit
~~~

Session-recorded output:

~~~text
=== RUN   TestTableColumnPlanExactBytes
=== RUN   TestTableColumnPlanExactBytes/outer-before-middle
=== RUN   TestTableColumnPlanExactBytes/outer-after-last
=== RUN   TestTableColumnPlanExactBytes/outer-delete-middle
=== RUN   TestTableColumnPlanExactBytes/no-outer-before-first
=== RUN   TestTableColumnPlanExactBytes/no-outer-after-last
=== RUN   TestTableColumnPlanExactBytes/no-outer-delete-middle
=== RUN   TestTableColumnPlanExactBytes/leading-only-insert-last
=== RUN   TestTableColumnPlanExactBytes/trailing-only-insert-first
=== RUN   TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-first
=== RUN   TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-last
=== RUN   TestTableColumnPlanExactBytes/indented-no-leading-delete-first
=== RUN   TestTableColumnPlanExactBytes/single-column-normal-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-leading-only-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-leading-only-insert-after
=== RUN   TestTableColumnPlanExactBytes/single-column-degenerate-insert-after
=== RUN   TestTableColumnPlanExactBytes/single-column-degenerate-insert-before
    table_column_edit_test.go:248: PlanTableColumnEdit: table column edit cannot preserve the complete source structure
=== RUN   TestTableColumnPlanExactBytes/single-column-padding-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-padding-insert-after
=== RUN   TestTableColumnPlanExactBytes/escaped-pipe-and-unicode-delete-first
--- FAIL: TestTableColumnPlanExactBytes (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-before-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-after-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-delete-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-before-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-after-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-delete-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/leading-only-insert-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/trailing-only-insert-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/indented-no-leading-delete-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-normal-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-leading-only-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-leading-only-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-degenerate-insert-after (0.00s)
    --- FAIL: TestTableColumnPlanExactBytes/single-column-degenerate-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-padding-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-padding-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/escaped-pipe-and-unicode-delete-first (0.00s)
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/LF
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/_LF
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/_
--- PASS: TestTableColumnPlanNewlineFormsAndNeighbors (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/LF (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/_LF (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/_ (0.00s)
=== RUN   TestTableColumnPlanWarningAndDescriptorState
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-warning-remains
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-demotion-refused
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-promotion-refused
--- PASS: TestTableColumnPlanWarningAndDescriptorState (0.01s)
    --- PASS: TestTableColumnPlanWarningAndDescriptorState/neighbor-warning-remains (0.00s)
    --- PASS: TestTableColumnPlanWarningAndDescriptorState/neighbor-demotion-refused (0.00s)
    --- PASS: TestTableColumnPlanWarningAndDescriptorState/neighbor-promotion-refused (0.00s)
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/unknown-action
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/invalid-utf8
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-table
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-anchor
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/column-bounds
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/one-column-minimum
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/eight-column-maximum
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/ragged-and-unsupported
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/odd-trailing-backslash
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/candidate-over-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/partial-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-table-bytes
--- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits (0.05s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/unknown-action (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/invalid-utf8 (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-table (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-anchor (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/column-bounds (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/one-column-minimum (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/eight-column-maximum (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/ragged-and-unsupported (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/odd-trailing-backslash (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-document (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/candidate-over-document (0.02s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/partial-document (0.01s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-table-bytes (0.01s)
FAIL
FAIL	levnote.local/native/formats/markdown	0.639s
FAIL
FOCUSED_EXIT=1

~~~

### Attempt 10: record ordinal 6974 (focused test)

- Call ID: `call_00_pcPAE1THpqQQWlIGSahf4116`
- Process status: `completed`
- Exit code: `0`
- Captured output length: `6491` characters
- Display completeness: `SESSION-RECORDED OUTPUT`

Command argv:

~~~json
["C:\\Users\\canace\\.cache\\codex-runtimes\\codex-primary-runtime\\dependencies\\native\\powershell\\pwsh.exe", "-Command", "& '.\\native\\scripts\\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumn','-count=1','-timeout=120s','-v'); $exit=$LASTEXITCODE; Write-Output \"FOCUSED_EXIT=$exit\"; exit $exit"]
~~~

Effective command:

~~~powershell
& '.\native\scripts\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumn','-count=1','-timeout=120s','-v'); $exit=$LASTEXITCODE; Write-Output "FOCUSED_EXIT=$exit"; exit $exit
~~~

Session-recorded output:

~~~text
=== RUN   TestTableColumnPlanExactBytes
=== RUN   TestTableColumnPlanExactBytes/outer-before-middle
=== RUN   TestTableColumnPlanExactBytes/outer-after-last
=== RUN   TestTableColumnPlanExactBytes/outer-delete-middle
=== RUN   TestTableColumnPlanExactBytes/no-outer-before-first
=== RUN   TestTableColumnPlanExactBytes/no-outer-after-last
=== RUN   TestTableColumnPlanExactBytes/no-outer-delete-middle
=== RUN   TestTableColumnPlanExactBytes/leading-only-insert-last
=== RUN   TestTableColumnPlanExactBytes/trailing-only-insert-first
=== RUN   TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-first
=== RUN   TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-last
=== RUN   TestTableColumnPlanExactBytes/indented-no-leading-delete-first
=== RUN   TestTableColumnPlanExactBytes/single-column-normal-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-leading-only-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-leading-only-insert-after
=== RUN   TestTableColumnPlanExactBytes/single-column-degenerate-insert-after
=== RUN   TestTableColumnPlanExactBytes/single-column-degenerate-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-padding-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-padding-insert-after
=== RUN   TestTableColumnPlanExactBytes/escaped-pipe-and-unicode-delete-first
--- PASS: TestTableColumnPlanExactBytes (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-before-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-after-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-delete-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-before-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-after-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-delete-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/leading-only-insert-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/trailing-only-insert-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/indented-no-leading-delete-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-normal-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-leading-only-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-leading-only-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-degenerate-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-degenerate-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-padding-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-padding-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/escaped-pipe-and-unicode-delete-first (0.00s)
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/LF
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/_LF
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/_
--- PASS: TestTableColumnPlanNewlineFormsAndNeighbors (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/LF (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/_LF (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/_ (0.00s)
=== RUN   TestTableColumnPlanWarningAndDescriptorState
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-warning-remains
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-demotion-refused
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-promotion-refused
--- PASS: TestTableColumnPlanWarningAndDescriptorState (0.01s)
    --- PASS: TestTableColumnPlanWarningAndDescriptorState/neighbor-warning-remains (0.00s)
    --- PASS: TestTableColumnPlanWarningAndDescriptorState/neighbor-demotion-refused (0.00s)
    --- PASS: TestTableColumnPlanWarningAndDescriptorState/neighbor-promotion-refused (0.00s)
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/unknown-action
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/invalid-utf8
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-table
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-anchor
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/column-bounds
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/one-column-minimum
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/eight-column-maximum
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/ragged-and-unsupported
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/odd-trailing-backslash
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/candidate-over-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/partial-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-table-bytes
--- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits (0.05s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/unknown-action (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/invalid-utf8 (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-table (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-anchor (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/column-bounds (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/one-column-minimum (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/eight-column-maximum (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/ragged-and-unsupported (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/odd-trailing-backslash (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-document (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/candidate-over-document (0.02s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/partial-document (0.01s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-table-bytes (0.02s)
PASS
ok  	levnote.local/native/formats/markdown	0.642s
FOCUSED_EXIT=0

~~~

### Attempt 11: record ordinal 7032 (focused test)

- Call ID: `call_00_Pdxfw1XKKlnIGNhm2z6G8372`
- Process status: `failed`
- Exit code: `1`
- Captured output length: `7282` characters
- Display completeness: `SESSION-RECORDED OUTPUT`

Command argv:

~~~json
["C:\\Users\\canace\\.cache\\codex-runtimes\\codex-primary-runtime\\dependencies\\native\\powershell\\pwsh.exe", "-Command", "& '.\\native\\scripts\\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumn','-count=1','-timeout=120s','-v'); $exit=$LASTEXITCODE; Write-Output \"FOCUSED_EXIT=$exit\"; exit $exit"]
~~~

Effective command:

~~~powershell
& '.\native\scripts\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumn','-count=1','-timeout=120s','-v'); $exit=$LASTEXITCODE; Write-Output "FOCUSED_EXIT=$exit"; exit $exit
~~~

Session-recorded output:

~~~text
=== RUN   TestTableColumnPlanExactBytes
=== RUN   TestTableColumnPlanExactBytes/outer-before-middle
=== RUN   TestTableColumnPlanExactBytes/outer-after-last
=== RUN   TestTableColumnPlanExactBytes/outer-delete-middle
=== RUN   TestTableColumnPlanExactBytes/no-outer-before-first
=== RUN   TestTableColumnPlanExactBytes/no-outer-after-last
=== RUN   TestTableColumnPlanExactBytes/no-outer-delete-middle
=== RUN   TestTableColumnPlanExactBytes/leading-only-insert-last
=== RUN   TestTableColumnPlanExactBytes/leading-only-append-after
=== RUN   TestTableColumnPlanExactBytes/leading-only-delete-last-empty-middle
    table_column_edit_test.go:282: candidate bytes:
         got "| H1 ||\n| --- | --- \n| A1 ||\n"
        want "| H1 ||\n| --- | ---\n| A1 ||\n"
=== RUN   TestTableColumnPlanExactBytes/leading-only-delete-last-padding-middle
    table_column_edit_test.go:282: candidate bytes:
         got "| H1 |  |\n| --- | --- \n| A1 |  |\n"
        want "| H1 |  |\n| --- | ---\n| A1 |  |\n"
=== RUN   TestTableColumnPlanExactBytes/trailing-only-insert-first
=== RUN   TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-first
=== RUN   TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-last
=== RUN   TestTableColumnPlanExactBytes/indented-no-leading-delete-first
=== RUN   TestTableColumnPlanExactBytes/single-column-normal-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-leading-only-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-leading-only-insert-after
=== RUN   TestTableColumnPlanExactBytes/single-column-degenerate-insert-after
=== RUN   TestTableColumnPlanExactBytes/single-column-degenerate-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-padding-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-padding-insert-after
=== RUN   TestTableColumnPlanExactBytes/escaped-pipe-and-unicode-delete-first
--- FAIL: TestTableColumnPlanExactBytes (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-before-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-after-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-delete-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-before-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-after-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-delete-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/leading-only-insert-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/leading-only-append-after (0.00s)
    --- FAIL: TestTableColumnPlanExactBytes/leading-only-delete-last-empty-middle (0.00s)
    --- FAIL: TestTableColumnPlanExactBytes/leading-only-delete-last-padding-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/trailing-only-insert-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/indented-no-leading-delete-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-normal-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-leading-only-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-leading-only-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-degenerate-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-degenerate-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-padding-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-padding-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/escaped-pipe-and-unicode-delete-first (0.00s)
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/LF
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/_LF
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/_
--- PASS: TestTableColumnPlanNewlineFormsAndNeighbors (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/LF (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/_LF (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/_ (0.00s)
=== RUN   TestTableColumnPlanWarningAndDescriptorState
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-warning-remains
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-demotion-refused
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-promotion-refused
--- PASS: TestTableColumnPlanWarningAndDescriptorState (0.02s)
    --- PASS: TestTableColumnPlanWarningAndDescriptorState/neighbor-warning-remains (0.00s)
    --- PASS: TestTableColumnPlanWarningAndDescriptorState/neighbor-demotion-refused (0.01s)
    --- PASS: TestTableColumnPlanWarningAndDescriptorState/neighbor-promotion-refused (0.01s)
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/unknown-action
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/invalid-utf8
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-table
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-anchor
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/column-bounds
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/one-column-minimum
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/eight-column-maximum
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/ragged-and-unsupported
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/odd-trailing-backslash
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/candidate-over-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/partial-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-table-bytes
--- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits (0.13s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/unknown-action (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/invalid-utf8 (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-table (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-anchor (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/column-bounds (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/one-column-minimum (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/eight-column-maximum (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/ragged-and-unsupported (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/odd-trailing-backslash (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-document (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/candidate-over-document (0.04s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/partial-document (0.04s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-table-bytes (0.05s)
FAIL
FAIL	levnote.local/native/formats/markdown	0.874s
FAIL
FOCUSED_EXIT=1

~~~

### Attempt 12: record ordinal 7046 (focused test)

- Call ID: `call_00_45WKqY6LyeglBpWQc1sY9394`
- Process status: `completed`
- Exit code: `0`
- Captured output length: `6975` characters
- Display completeness: `SESSION-RECORDED OUTPUT`

Command argv:

~~~json
["C:\\Users\\canace\\.cache\\codex-runtimes\\codex-primary-runtime\\dependencies\\native\\powershell\\pwsh.exe", "-Command", "& '.\\native\\scripts\\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumn','-count=1','-timeout=120s','-v'); $exit=$LASTEXITCODE; Write-Output \"FOCUSED_EXIT=$exit\"; exit $exit"]
~~~

Effective command:

~~~powershell
& '.\native\scripts\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumn','-count=1','-timeout=120s','-v'); $exit=$LASTEXITCODE; Write-Output "FOCUSED_EXIT=$exit"; exit $exit
~~~

Session-recorded output:

~~~text
=== RUN   TestTableColumnPlanExactBytes
=== RUN   TestTableColumnPlanExactBytes/outer-before-middle
=== RUN   TestTableColumnPlanExactBytes/outer-after-last
=== RUN   TestTableColumnPlanExactBytes/outer-delete-middle
=== RUN   TestTableColumnPlanExactBytes/no-outer-before-first
=== RUN   TestTableColumnPlanExactBytes/no-outer-after-last
=== RUN   TestTableColumnPlanExactBytes/no-outer-delete-middle
=== RUN   TestTableColumnPlanExactBytes/leading-only-insert-last
=== RUN   TestTableColumnPlanExactBytes/leading-only-append-after
=== RUN   TestTableColumnPlanExactBytes/leading-only-delete-last-empty-middle
=== RUN   TestTableColumnPlanExactBytes/leading-only-delete-last-padding-middle
=== RUN   TestTableColumnPlanExactBytes/trailing-only-insert-first
=== RUN   TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-first
=== RUN   TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-last
=== RUN   TestTableColumnPlanExactBytes/indented-no-leading-delete-first
=== RUN   TestTableColumnPlanExactBytes/single-column-normal-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-leading-only-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-leading-only-insert-after
=== RUN   TestTableColumnPlanExactBytes/single-column-degenerate-insert-after
=== RUN   TestTableColumnPlanExactBytes/single-column-degenerate-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-padding-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-padding-insert-after
=== RUN   TestTableColumnPlanExactBytes/escaped-pipe-and-unicode-delete-first
--- PASS: TestTableColumnPlanExactBytes (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-before-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-after-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-delete-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-before-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-after-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-delete-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/leading-only-insert-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/leading-only-append-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/leading-only-delete-last-empty-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/leading-only-delete-last-padding-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/trailing-only-insert-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/indented-no-leading-delete-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-normal-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-leading-only-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-leading-only-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-degenerate-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-degenerate-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-padding-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-padding-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/escaped-pipe-and-unicode-delete-first (0.00s)
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/LF
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/_LF
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/_
--- PASS: TestTableColumnPlanNewlineFormsAndNeighbors (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/LF (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/_LF (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/_ (0.00s)
=== RUN   TestTableColumnPlanWarningAndDescriptorState
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-warning-remains
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-demotion-refused
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-promotion-refused
--- PASS: TestTableColumnPlanWarningAndDescriptorState (0.01s)
    --- PASS: TestTableColumnPlanWarningAndDescriptorState/neighbor-warning-remains (0.00s)
    --- PASS: TestTableColumnPlanWarningAndDescriptorState/neighbor-demotion-refused (0.01s)
    --- PASS: TestTableColumnPlanWarningAndDescriptorState/neighbor-promotion-refused (0.00s)
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/unknown-action
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/invalid-utf8
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-table
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-anchor
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/column-bounds
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/one-column-minimum
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/eight-column-maximum
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/ragged-and-unsupported
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/odd-trailing-backslash
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/candidate-over-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/partial-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-table-bytes
--- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits (0.06s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/unknown-action (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/invalid-utf8 (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-table (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-anchor (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/column-bounds (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/one-column-minimum (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/eight-column-maximum (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/ragged-and-unsupported (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/odd-trailing-backslash (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-document (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/candidate-over-document (0.03s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/partial-document (0.01s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-table-bytes (0.02s)
PASS
ok  	levnote.local/native/formats/markdown	0.583s
FOCUSED_EXIT=0

~~~

### Attempt 13: record ordinal 7062 (full format suite)

- Call ID: `call_00_nH30EUEqQmeyjVNu1FDF8589`
- Process status: `completed`
- Exit code: `0`
- Captured output length: `75` characters
- Display completeness: `SESSION-RECORDED OUTPUT`

Command argv:

~~~json
["C:\\Users\\canace\\.cache\\codex-runtimes\\codex-primary-runtime\\dependencies\\native\\powershell\\pwsh.exe", "-Command", "& '.\\native\\scripts\\go.ps1' -GoArguments @('test','./formats/markdown','-count=1','-timeout=120s'); $exit=$LASTEXITCODE; Write-Output \"FULL_FORMAT_TEST_EXIT=$exit\"; exit $exit"]
~~~

Effective command:

~~~powershell
& '.\native\scripts\go.ps1' -GoArguments @('test','./formats/markdown','-count=1','-timeout=120s'); $exit=$LASTEXITCODE; Write-Output "FULL_FORMAT_TEST_EXIT=$exit"; exit $exit
~~~

Session-recorded output:

~~~text
ok  	levnote.local/native/formats/markdown	0.695s
FULL_FORMAT_TEST_EXIT=0

~~~

### Attempt 14: record ordinal 7069 (vet)

- Call ID: `call_00_UVQHeLx2Yvz5LEYo8RC23315`
- Process status: `completed`
- Exit code: `0`
- Captured output length: `19` characters
- Display completeness: `SESSION-RECORDED OUTPUT`

Command argv:

~~~json
["C:\\Users\\canace\\.cache\\codex-runtimes\\codex-primary-runtime\\dependencies\\native\\powershell\\pwsh.exe", "-Command", "& '.\\native\\scripts\\go.ps1' -GoArguments @('vet','./formats/markdown'); $exit=$LASTEXITCODE; Write-Output \"FORMAT_VET_EXIT=$exit\"; exit $exit"]
~~~

Effective command:

~~~powershell
& '.\native\scripts\go.ps1' -GoArguments @('vet','./formats/markdown'); $exit=$LASTEXITCODE; Write-Output "FORMAT_VET_EXIT=$exit"; exit $exit
~~~

Session-recorded output:

~~~text
FORMAT_VET_EXIT=0

~~~

### Attempt 15: record ordinal 7108 (focused test)

- Call ID: `call_00_eC9uWh5SV6wXVumFqGad7891`
- Process status: `failed`
- Exit code: `1`
- Captured output length: `7324` characters
- Display completeness: `SESSION-RECORDED OUTPUT`

Command argv:

~~~json
["C:\\Users\\canace\\.cache\\codex-runtimes\\codex-primary-runtime\\dependencies\\native\\powershell\\pwsh.exe", "-Command", "& '.\\native\\scripts\\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumn','-count=1','-timeout=120s','-v'); $exit=$LASTEXITCODE; Write-Output \"FOCUSED_EXIT=$exit\"; exit $exit"]
~~~

Effective command:

~~~powershell
& '.\native\scripts\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumn','-count=1','-timeout=120s','-v'); $exit=$LASTEXITCODE; Write-Output "FOCUSED_EXIT=$exit"; exit $exit
~~~

Session-recorded output:

~~~text
=== RUN   TestTableColumnPlanExactBytes
=== RUN   TestTableColumnPlanExactBytes/outer-before-middle
=== RUN   TestTableColumnPlanExactBytes/outer-after-last
=== RUN   TestTableColumnPlanExactBytes/outer-delete-middle
=== RUN   TestTableColumnPlanExactBytes/seven-to-eight-insert-after
=== RUN   TestTableColumnPlanExactBytes/no-outer-before-first
=== RUN   TestTableColumnPlanExactBytes/no-outer-after-last
=== RUN   TestTableColumnPlanExactBytes/no-outer-delete-middle
=== RUN   TestTableColumnPlanExactBytes/leading-only-insert-last
=== RUN   TestTableColumnPlanExactBytes/leading-only-append-after
=== RUN   TestTableColumnPlanExactBytes/leading-only-delete-last-empty-middle
=== RUN   TestTableColumnPlanExactBytes/leading-only-delete-last-padding-middle
=== RUN   TestTableColumnPlanExactBytes/trailing-only-insert-first
=== RUN   TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-first
=== RUN   TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-last
=== RUN   TestTableColumnPlanExactBytes/indented-no-leading-delete-first
=== RUN   TestTableColumnPlanExactBytes/single-column-normal-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-leading-only-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-leading-only-insert-after
=== RUN   TestTableColumnPlanExactBytes/single-column-degenerate-insert-after
=== RUN   TestTableColumnPlanExactBytes/single-column-degenerate-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-padding-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-padding-insert-after
=== RUN   TestTableColumnPlanExactBytes/escaped-pipe-and-unicode-delete-first
--- PASS: TestTableColumnPlanExactBytes (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-before-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-after-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-delete-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/seven-to-eight-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-before-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-after-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-delete-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/leading-only-insert-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/leading-only-append-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/leading-only-delete-last-empty-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/leading-only-delete-last-padding-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/trailing-only-insert-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/indented-no-leading-delete-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-normal-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-leading-only-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-leading-only-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-degenerate-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-degenerate-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-padding-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-padding-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/escaped-pipe-and-unicode-delete-first (0.00s)
=== RUN   TestTableColumnCandidateExactRawLimitRefusal
    table_column_edit_test.go:310: exact-limit fixture bytes=32512 want=32768
--- FAIL: TestTableColumnCandidateExactRawLimitRefusal (0.00s)
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/LF
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/_LF
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/_
--- PASS: TestTableColumnPlanNewlineFormsAndNeighbors (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/LF (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/_LF (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/_ (0.00s)
=== RUN   TestTableColumnPlanWarningAndDescriptorState
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-warning-remains
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-demotion-refused
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-promotion-refused
--- PASS: TestTableColumnPlanWarningAndDescriptorState (0.01s)
    --- PASS: TestTableColumnPlanWarningAndDescriptorState/neighbor-warning-remains (0.00s)
    --- PASS: TestTableColumnPlanWarningAndDescriptorState/neighbor-demotion-refused (0.00s)
    --- PASS: TestTableColumnPlanWarningAndDescriptorState/neighbor-promotion-refused (0.00s)
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/unknown-action
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/invalid-utf8
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-table
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-anchor
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/column-bounds
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/one-column-minimum
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/eight-column-maximum
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/ragged-and-unsupported
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/odd-trailing-backslash
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/candidate-over-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/partial-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-table-bytes
--- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits (0.04s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/unknown-action (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/invalid-utf8 (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-table (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-anchor (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/column-bounds (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/one-column-minimum (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/eight-column-maximum (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/ragged-and-unsupported (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/odd-trailing-backslash (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-document (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/candidate-over-document (0.02s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/partial-document (0.01s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-table-bytes (0.01s)
FAIL
FAIL	levnote.local/native/formats/markdown	0.526s
FAIL
FOCUSED_EXIT=1

~~~

### Attempt 16: record ordinal 7122 (focused test)

- Call ID: `call_00_nc4ltDklG5CNHwvayjxF1952`
- Process status: `failed`
- Exit code: `1`
- Captured output length: `42563` characters
- Display completeness: `LONG DISPLAY WAS TRUNCATED; SESSION-RECORDED OUTPUT BELOW`

Command argv:

~~~json
["C:\\Users\\canace\\.cache\\codex-runtimes\\codex-primary-runtime\\dependencies\\native\\powershell\\pwsh.exe", "-Command", "& '.\\native\\scripts\\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumn','-count=1','-timeout=120s','-v'); $exit=$LASTEXITCODE; Write-Output \"FOCUSED_EXIT=$exit\"; exit $exit"]
~~~

Effective command:

~~~powershell
& '.\native\scripts\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumn','-count=1','-timeout=120s','-v'); $exit=$LASTEXITCODE; Write-Output "FOCUSED_EXIT=$exit"; exit $exit
~~~

Session-recorded output:

~~~text
=== RUN   TestTableColumnPlanExactBytes
=== RUN   TestTableColumnPlanExactBytes/outer-before-middle
=== RUN   TestTableColumnPlanExactBytes/outer-after-last
=== RUN   TestTableColumnPlanExactBytes/outer-delete-middle
=== RUN   TestTableColumnPlanExactBytes/seven-to-eight-insert-after
=== RUN   TestTableColumnPlanExactBytes/no-outer-before-first
=== RUN   TestTableColumnPlanExactBytes/no-outer-after-last
=== RUN   TestTableColumnPlanExactBytes/no-outer-delete-middle
=== RUN   TestTableColumnPlanExactBytes/leading-only-insert-last
=== RUN   TestTableColumnPlanExactBytes/leading-only-append-after
=== RUN   TestTableColumnPlanExactBytes/leading-only-delete-last-empty-middle
=== RUN   TestTableColumnPlanExactBytes/leading-only-delete-last-padding-middle
=== RUN   TestTableColumnPlanExactBytes/trailing-only-insert-first
=== RUN   TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-first
=== RUN   TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-last
=== RUN   TestTableColumnPlanExactBytes/indented-no-leading-delete-first
=== RUN   TestTableColumnPlanExactBytes/single-column-normal-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-leading-only-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-leading-only-insert-after
=== RUN   TestTableColumnPlanExactBytes/single-column-degenerate-insert-after
=== RUN   TestTableColumnPlanExactBytes/single-column-degenerate-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-padding-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-padding-insert-after
=== RUN   TestTableColumnPlanExactBytes/escaped-pipe-and-unicode-delete-first
--- PASS: TestTableColumnPlanExactBytes (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-before-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-after-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-delete-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/seven-to-eight-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-before-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-after-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-delete-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/leading-only-insert-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/leading-only-append-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/leading-only-delete-last-empty-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/leading-only-delete-last-padding-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/trailing-only-insert-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/indented-no-leading-delete-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-normal-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-leading-only-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-leading-only-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-degenerate-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-degenerate-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-padding-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-padding-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/escaped-pipe-and-unicode-delete-first (0.00s)
=== RUN   TestTableColumnCandidateExactRawLimitRefusal
    table_column_edit_test.go:314: exact-limit fixture shape partial=false blocks=[{Kind:table Range:{Start:0 End:32768} TextRange:{Start:0 End:32768} Text: Inline:[] Level:0 Language: Ordered:false Start:0 Items:[] DocBlock:{Attrs:{ID: Kind: Mode: Src: Title: Extra:map[]} HeaderRange:{Start:0 End:0} Payload: FenceLength:0 PayloadRange:{Start:0 End:0} ReferenceProblem:} Table:{Rows:[] DelimiterRange:{Start:0 End:0} Alignments:[] Problem:table body row has a different column count} Raw:|a|b|c|d|e|f|g|
        |---|---|---|---|---|---|---|
        |xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx|xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx|xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx|xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx|xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx|xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx|xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx|xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx|
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||
        |||||||||}]
--- FAIL: TestTableColumnCandidateExactRawLimitRefusal (0.00s)
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/LF
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/_LF
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/_
--- PASS: TestTableColumnPlanNewlineFormsAndNeighbors (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/LF (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/_LF (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/_ (0.00s)
=== RUN   TestTableColumnPlanWarningAndDescriptorState
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-warning-remains
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-demotion-refused
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-promotion-refused
--- PASS: TestTableColumnPlanWarningAndDescriptorState (0.01s)
    --- PASS: TestTableColumnPlanWarningAndDescriptorState/neighbor-warning-remains (0.00s)
    --- PASS: TestTableColumnPlanWarningAndDescriptorState/neighbor-demotion-refused (0.00s)
    --- PASS: TestTableColumnPlanWarningAndDescriptorState/neighbor-promotion-refused (0.00s)
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/unknown-action
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/invalid-utf8
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-table
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-anchor
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/column-bounds
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/one-column-minimum
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/eight-column-maximum
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/ragged-and-unsupported
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/odd-trailing-backslash
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/candidate-over-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/partial-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-table-bytes
--- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits (0.04s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/unknown-action (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/invalid-utf8 (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-table (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-anchor (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/column-bounds (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/one-column-minimum (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/eight-column-maximum (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/ragged-and-unsupported (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/odd-trailing-backslash (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-document (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/candidate-over-document (0.02s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/partial-document (0.01s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-table-bytes (0.01s)
FAIL
FAIL	levnote.local/native/formats/markdown	0.554s
FAIL
FOCUSED_EXIT=1

~~~

### Attempt 17: record ordinal 7164 (full format suite)

- Call ID: `call_00_HoWmNijiuh1sVG8n25qc8717`
- Process status: `failed`
- Exit code: `1`
- Captured output length: `255` characters
- Display completeness: `SESSION-RECORDED OUTPUT`

Command argv:

~~~json
["C:\\WINDOWS\\system32\\bash.exe", "-lc", "& '.\\native\\scripts\\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumnCandidateExactRawLimitRefusal$','-count=1','-timeout=120s','-v'); $exit=$LASTEXITCODE; Write-Output \"EXACT_LIMIT_TEST_EXIT=$exit\"; exit $exit"]
~~~

Effective command:

~~~powershell
& '.\native\scripts\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumnCandidateExactRawLimitRefusal$','-count=1','-timeout=120s','-v'); $exit=$LASTEXITCODE; Write-Output "EXACT_LIMIT_TEST_EXIT=$exit"; exit $exit
~~~

Session-recorded output:

~~~text
�(u�N  L i n u x   �v  W i n d o w s   P[�|�~�l	g�]�[ň�vR�S0 
 �S	cgq�NN�f�[ňR�Seg㉳Qdk�:  
  
 O(u  ' w s l . e x e   - - l i s t   - - o n l i n e '   R�Q�S(u�vR�S 
 �T  ' w s l . e x e   - - i n s t a l l   < D i s t r o > '   ۏL��[ň0 
 
~~~

### Attempt 18: record ordinal 7171 (full format suite)

- Call ID: `call_00_xLxaSLZ1QzYpQAfJbFwT4649`
- Process status: `failed`
- Exit code: `1`
- Captured output length: `801` characters
- Display completeness: `SESSION-RECORDED OUTPUT`

Command argv:

~~~json
["C:\\Users\\canace\\.cache\\codex-runtimes\\codex-primary-runtime\\dependencies\\native\\powershell\\pwsh.exe", "-Command", "& '.\\native\\scripts\\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumnCandidateExactRawLimitRefusal$','-count=1','-timeout=120s','-v'); $exit=$LASTEXITCODE; Write-Output \"EXACT_LIMIT_TEST_EXIT=$exit\"; exit $exit"]
~~~

Effective command:

~~~powershell
& '.\native\scripts\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumnCandidateExactRawLimitRefusal$','-count=1','-timeout=120s','-v'); $exit=$LASTEXITCODE; Write-Output "EXACT_LIMIT_TEST_EXIT=$exit"; exit $exit
~~~

Session-recorded output:

~~~text
=== RUN   TestTableColumnCandidateExactRawLimitRefusal
    table_column_edit_test.go:318: exact-limit line=0 bytes=15 pipes=8 cells=7 valid=true mask={row:{Range:{Start:0 End:16} Cells:[{Range:{Start:1 End:2} TextRange:{Start:1 End:2}} {Range:{Start:3 End:4} TextRange:{Start:3 End:4}} {Range:{Start:5 End:6} TextRange:{Start:5 End:6}} {Range:{Start:7 End:8} TextRange:{Start:7 End:8}} {Range:{Start:9 End:10} TextRange:{Start:9 End:10}} {Range:{Start:11 End:12} TextRange:{Start:11 End:12}} {Range:{Start:13 End:14} TextRange:{Start:13 End:14}}]} indent:0 pipes:8 valid:true tooIndented:false overColumns:false overCell:false oneColumnNoLeading:false}
--- FAIL: TestTableColumnCandidateExactRawLimitRefusal (0.00s)
FAIL
FAIL	levnote.local/native/formats/markdown	0.521s
FAIL
EXACT_LIMIT_TEST_EXIT=1

~~~

### Attempt 19: record ordinal 7185 (full format suite)

- Call ID: `call_00_nIgj1RymQ0NIUoI8KgjC0458`
- Process status: `failed`
- Exit code: `1`
- Captured output length: `350` characters
- Display completeness: `SESSION-RECORDED OUTPUT`

Command argv:

~~~json
["C:\\Users\\canace\\.cache\\codex-runtimes\\codex-primary-runtime\\dependencies\\native\\powershell\\pwsh.exe", "-Command", "& '.\\native\\scripts\\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumnCandidateExactRawLimitRefusal$','-count=1','-timeout=120s','-v'); $exit=$LASTEXITCODE; Write-Output \"EXACT_LIMIT_TEST_EXIT=$exit\"; exit $exit"]
~~~

Effective command:

~~~powershell
& '.\native\scripts\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumnCandidateExactRawLimitRefusal$','-count=1','-timeout=120s','-v'); $exit=$LASTEXITCODE; Write-Output "EXACT_LIMIT_TEST_EXIT=$exit"; exit $exit
~~~

Session-recorded output:

~~~text
=== RUN   TestTableColumnCandidateExactRawLimitRefusal
    table_column_edit_test.go:333: candidate-over-exact-limit patch={Edits:[]} err=table column edit cannot preserve the complete source structure
--- FAIL: TestTableColumnCandidateExactRawLimitRefusal (0.00s)
FAIL
FAIL	levnote.local/native/formats/markdown	0.476s
FAIL
EXACT_LIMIT_TEST_EXIT=1

~~~

### Attempt 20: record ordinal 7200 (full format suite)

- Call ID: `call_00_QyaAlwtTlAC5E7X1jFcp7806`
- Process status: `failed`
- Exit code: `1`
- Captured output length: `551` characters
- Display completeness: `SESSION-RECORDED OUTPUT`

Command argv:

~~~json
["C:\\Users\\canace\\.cache\\codex-runtimes\\codex-primary-runtime\\dependencies\\native\\powershell\\pwsh.exe", "-Command", "& '.\\native\\scripts\\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumnCandidateExactRawLimitRefusal$','-count=1','-timeout=120s','-v'); $exit=$LASTEXITCODE; Write-Output \"EXACT_LIMIT_TEST_EXIT=$exit\"; exit $exit"]
~~~

Effective command:

~~~powershell
& '.\native\scripts\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumnCandidateExactRawLimitRefusal$','-count=1','-timeout=120s','-v'); $exit=$LASTEXITCODE; Write-Output "EXACT_LIMIT_TEST_EXIT=$exit"; exit $exit
~~~

Session-recorded output:

~~~text
=== RUN   TestTableColumnCandidateExactRawLimitRefusal
    table_column_edit_test.go:349: exact-limit source=32768 candidate=33029 edits=258 afterBlocks=1 partial=false firstProblem="table exceeds the 32 KiB preview limit; complete source remains visible"
    table_column_edit_test.go:357: candidate-over-exact-limit patch={Edits:[]} err=table column edit cannot preserve the complete source structure
--- FAIL: TestTableColumnCandidateExactRawLimitRefusal (0.00s)
FAIL
FAIL	levnote.local/native/formats/markdown	0.514s
FAIL
EXACT_LIMIT_TEST_EXIT=1

~~~

### Attempt 21: record ordinal 7249 (focused test)

- Call ID: `call_00_A2dP0EHfh91joQuOvE6I6833`
- Process status: `completed`
- Exit code: `0`
- Captured output length: `7241` characters
- Display completeness: `SESSION-RECORDED OUTPUT`

Command argv:

~~~json
["C:\\Users\\canace\\.cache\\codex-runtimes\\codex-primary-runtime\\dependencies\\native\\powershell\\pwsh.exe", "-Command", "& '.\\native\\scripts\\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumn','-count=1','-timeout=120s','-v'); $exit=$LASTEXITCODE; Write-Output \"FOCUSED_EXIT=$exit\"; exit $exit"]
~~~

Effective command:

~~~powershell
& '.\native\scripts\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumn','-count=1','-timeout=120s','-v'); $exit=$LASTEXITCODE; Write-Output "FOCUSED_EXIT=$exit"; exit $exit
~~~

Session-recorded output:

~~~text
=== RUN   TestTableColumnPlanExactBytes
=== RUN   TestTableColumnPlanExactBytes/outer-before-middle
=== RUN   TestTableColumnPlanExactBytes/outer-after-last
=== RUN   TestTableColumnPlanExactBytes/outer-delete-middle
=== RUN   TestTableColumnPlanExactBytes/seven-to-eight-insert-after
=== RUN   TestTableColumnPlanExactBytes/no-outer-before-first
=== RUN   TestTableColumnPlanExactBytes/no-outer-after-last
=== RUN   TestTableColumnPlanExactBytes/no-outer-delete-middle
=== RUN   TestTableColumnPlanExactBytes/leading-only-insert-last
=== RUN   TestTableColumnPlanExactBytes/leading-only-append-after
=== RUN   TestTableColumnPlanExactBytes/leading-only-delete-last-empty-middle
=== RUN   TestTableColumnPlanExactBytes/leading-only-delete-last-padding-middle
=== RUN   TestTableColumnPlanExactBytes/trailing-only-insert-first
=== RUN   TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-first
=== RUN   TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-last
=== RUN   TestTableColumnPlanExactBytes/indented-no-leading-delete-first
=== RUN   TestTableColumnPlanExactBytes/single-column-normal-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-leading-only-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-leading-only-insert-after
=== RUN   TestTableColumnPlanExactBytes/single-column-degenerate-insert-after
=== RUN   TestTableColumnPlanExactBytes/single-column-degenerate-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-padding-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-padding-insert-after
=== RUN   TestTableColumnPlanExactBytes/escaped-pipe-and-unicode-delete-first
--- PASS: TestTableColumnPlanExactBytes (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-before-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-after-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-delete-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/seven-to-eight-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-before-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-after-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-delete-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/leading-only-insert-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/leading-only-append-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/leading-only-delete-last-empty-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/leading-only-delete-last-padding-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/trailing-only-insert-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/indented-no-leading-delete-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-normal-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-leading-only-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-leading-only-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-degenerate-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-degenerate-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-padding-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-padding-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/escaped-pipe-and-unicode-delete-first (0.00s)
=== RUN   TestTableColumnCandidateExactRawLimitRefusal
--- PASS: TestTableColumnCandidateExactRawLimitRefusal (0.00s)
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/LF
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/_LF
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/_
--- PASS: TestTableColumnPlanNewlineFormsAndNeighbors (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/LF (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/_LF (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/_ (0.00s)
=== RUN   TestTableColumnPlanWarningAndDescriptorState
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-warning-remains
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-demotion-refused
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-promotion-refused
--- PASS: TestTableColumnPlanWarningAndDescriptorState (0.01s)
    --- PASS: TestTableColumnPlanWarningAndDescriptorState/neighbor-warning-remains (0.00s)
    --- PASS: TestTableColumnPlanWarningAndDescriptorState/neighbor-demotion-refused (0.00s)
    --- PASS: TestTableColumnPlanWarningAndDescriptorState/neighbor-promotion-refused (0.00s)
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/unknown-action
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/invalid-utf8
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-table
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-anchor
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/column-bounds
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/one-column-minimum
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/eight-column-maximum
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/ragged-and-unsupported
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/odd-trailing-backslash
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/candidate-over-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/partial-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-table-bytes
--- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits (0.04s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/unknown-action (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/invalid-utf8 (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-table (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-anchor (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/column-bounds (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/one-column-minimum (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/eight-column-maximum (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/ragged-and-unsupported (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/odd-trailing-backslash (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-document (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/candidate-over-document (0.02s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/partial-document (0.01s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-table-bytes (0.01s)
PASS
ok  	levnote.local/native/formats/markdown	0.637s
FOCUSED_EXIT=0

~~~

### Attempt 22: record ordinal 7265 (full format suite)

- Call ID: `call_00_2oVIfrtbgL7vktKsBG0L3414`
- Process status: `completed`
- Exit code: `0`
- Captured output length: `81` characters
- Display completeness: `SESSION-RECORDED OUTPUT`

Command argv:

~~~json
["C:\\Users\\canace\\.cache\\codex-runtimes\\codex-primary-runtime\\dependencies\\native\\powershell\\pwsh.exe", "-Command", "& '.\\native\\scripts\\go.ps1' -GoArguments @('test','./formats/markdown','-count=1','-timeout=120s'); $exit=$LASTEXITCODE; Write-Output \"FINAL_FULL_FORMAT_TEST_EXIT=$exit\"; exit $exit"]
~~~

Effective command:

~~~powershell
& '.\native\scripts\go.ps1' -GoArguments @('test','./formats/markdown','-count=1','-timeout=120s'); $exit=$LASTEXITCODE; Write-Output "FINAL_FULL_FORMAT_TEST_EXIT=$exit"; exit $exit
~~~

Session-recorded output:

~~~text
ok  	levnote.local/native/formats/markdown	0.912s
FINAL_FULL_FORMAT_TEST_EXIT=0

~~~

### Attempt 23: record ordinal 7272 (vet)

- Call ID: `call_00_k8QGV9saEGmwW0NJNOwX7197`
- Process status: `completed`
- Exit code: `0`
- Captured output length: `25` characters
- Display completeness: `SESSION-RECORDED OUTPUT`

Command argv:

~~~json
["C:\\Users\\canace\\.cache\\codex-runtimes\\codex-primary-runtime\\dependencies\\native\\powershell\\pwsh.exe", "-Command", "& '.\\native\\scripts\\go.ps1' -GoArguments @('vet','./formats/markdown'); $exit=$LASTEXITCODE; Write-Output \"FINAL_FORMAT_VET_EXIT=$exit\"; exit $exit"]
~~~

Effective command:

~~~powershell
& '.\native\scripts\go.ps1' -GoArguments @('vet','./formats/markdown'); $exit=$LASTEXITCODE; Write-Output "FINAL_FORMAT_VET_EXIT=$exit"; exit $exit
~~~

Session-recorded output:

~~~text
FINAL_FORMAT_VET_EXIT=0

~~~

### Attempt 24: record ordinal 7323 (focused test)

- Call ID: `call_00_liOWs6K7jp45M6PwUyO68023`
- Process status: `completed`
- Exit code: `0`
- Captured output length: `7407` characters
- Display completeness: `SESSION-RECORDED OUTPUT`

Command argv:

~~~json
["C:\\Users\\canace\\.cache\\codex-runtimes\\codex-primary-runtime\\dependencies\\native\\powershell\\pwsh.exe", "-Command", "& '.\\native\\scripts\\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumn','-count=1','-timeout=120s','-v'); $exit=$LASTEXITCODE; Write-Output \"FOCUSED_EXIT=$exit\"; exit $exit"]
~~~

Effective command:

~~~powershell
& '.\native\scripts\go.ps1' -GoArguments @('test','./formats/markdown','-run','^TestTableColumn','-count=1','-timeout=120s','-v'); $exit=$LASTEXITCODE; Write-Output "FOCUSED_EXIT=$exit"; exit $exit
~~~

Session-recorded output:

~~~text
=== RUN   TestTableColumnPlanExactBytes
=== RUN   TestTableColumnPlanExactBytes/outer-before-middle
=== RUN   TestTableColumnPlanExactBytes/outer-after-last
=== RUN   TestTableColumnPlanExactBytes/outer-delete-middle
=== RUN   TestTableColumnPlanExactBytes/seven-to-eight-insert-after
=== RUN   TestTableColumnPlanExactBytes/even-trailing-backslash-insert-after
=== RUN   TestTableColumnPlanExactBytes/no-outer-before-first
=== RUN   TestTableColumnPlanExactBytes/no-outer-after-last
=== RUN   TestTableColumnPlanExactBytes/no-outer-delete-middle
=== RUN   TestTableColumnPlanExactBytes/leading-only-insert-last
=== RUN   TestTableColumnPlanExactBytes/leading-only-append-after
=== RUN   TestTableColumnPlanExactBytes/leading-only-delete-last-empty-middle
=== RUN   TestTableColumnPlanExactBytes/leading-only-delete-last-padding-middle
=== RUN   TestTableColumnPlanExactBytes/trailing-only-insert-first
=== RUN   TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-first
=== RUN   TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-last
=== RUN   TestTableColumnPlanExactBytes/indented-no-leading-delete-first
=== RUN   TestTableColumnPlanExactBytes/single-column-normal-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-leading-only-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-leading-only-insert-after
=== RUN   TestTableColumnPlanExactBytes/single-column-degenerate-insert-after
=== RUN   TestTableColumnPlanExactBytes/single-column-degenerate-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-padding-insert-before
=== RUN   TestTableColumnPlanExactBytes/single-column-padding-insert-after
=== RUN   TestTableColumnPlanExactBytes/escaped-pipe-and-unicode-delete-first
--- PASS: TestTableColumnPlanExactBytes (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-before-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-after-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/outer-delete-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/seven-to-eight-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/even-trailing-backslash-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-before-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-after-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/no-outer-delete-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/leading-only-insert-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/leading-only-append-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/leading-only-delete-last-empty-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/leading-only-delete-last-padding-middle (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/trailing-only-insert-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/two-to-one-no-leading-delete-last (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/indented-no-leading-delete-first (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-normal-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-leading-only-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-leading-only-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-degenerate-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-degenerate-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-padding-insert-before (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/single-column-padding-insert-after (0.00s)
    --- PASS: TestTableColumnPlanExactBytes/escaped-pipe-and-unicode-delete-first (0.00s)
=== RUN   TestTableColumnCandidateExactRawLimitRefusal
--- PASS: TestTableColumnCandidateExactRawLimitRefusal (0.00s)
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/LF
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/_LF
=== RUN   TestTableColumnPlanNewlineFormsAndNeighbors/_
--- PASS: TestTableColumnPlanNewlineFormsAndNeighbors (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/LF (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/_LF (0.00s)
    --- PASS: TestTableColumnPlanNewlineFormsAndNeighbors/_ (0.00s)
=== RUN   TestTableColumnPlanWarningAndDescriptorState
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-warning-remains
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-demotion-refused
=== RUN   TestTableColumnPlanWarningAndDescriptorState/neighbor-promotion-refused
--- PASS: TestTableColumnPlanWarningAndDescriptorState (0.01s)
    --- PASS: TestTableColumnPlanWarningAndDescriptorState/neighbor-warning-remains (0.00s)
    --- PASS: TestTableColumnPlanWarningAndDescriptorState/neighbor-demotion-refused (0.00s)
    --- PASS: TestTableColumnPlanWarningAndDescriptorState/neighbor-promotion-refused (0.00s)
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/unknown-action
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/invalid-utf8
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-table
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-anchor
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/column-bounds
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/one-column-minimum
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/eight-column-maximum
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/ragged-and-unsupported
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/odd-trailing-backslash
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/candidate-over-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/partial-document
=== RUN   TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-table-bytes
--- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits (0.04s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/unknown-action (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/invalid-utf8 (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-table (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/wrong-anchor (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/column-bounds (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/one-column-minimum (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/eight-column-maximum (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/ragged-and-unsupported (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/odd-trailing-backslash (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-document (0.00s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/candidate-over-document (0.02s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/partial-document (0.01s)
    --- PASS: TestTableColumnPlanRefusesInvalidTargetsAndLimits/over-table-bytes (0.01s)
PASS
ok  	levnote.local/native/formats/markdown	0.531s
FOCUSED_EXIT=0

~~~

### Attempt 25: record ordinal 7330 (full format suite)

- Call ID: `call_00_UL894SG6UGco9SKmpCxo3346`
- Process status: `completed`
- Exit code: `0`
- Captured output length: `81` characters
- Display completeness: `SESSION-RECORDED OUTPUT`

Command argv:

~~~json
["C:\\Users\\canace\\.cache\\codex-runtimes\\codex-primary-runtime\\dependencies\\native\\powershell\\pwsh.exe", "-Command", "& '.\\native\\scripts\\go.ps1' -GoArguments @('test','./formats/markdown','-count=1','-timeout=120s'); $exit=$LASTEXITCODE; Write-Output \"FINAL_FULL_FORMAT_TEST_EXIT=$exit\"; exit $exit"]
~~~

Effective command:

~~~powershell
& '.\native\scripts\go.ps1' -GoArguments @('test','./formats/markdown','-count=1','-timeout=120s'); $exit=$LASTEXITCODE; Write-Output "FINAL_FULL_FORMAT_TEST_EXIT=$exit"; exit $exit
~~~

Session-recorded output:

~~~text
ok  	levnote.local/native/formats/markdown	0.692s
FINAL_FULL_FORMAT_TEST_EXIT=0

~~~

### Attempt 26: record ordinal 7335 (vet)

- Call ID: `call_00_ET_TZvn8Wui2f0PyC69kODT0585`
- Process status: `completed`
- Exit code: `0`
- Captured output length: `25` characters
- Display completeness: `SESSION-RECORDED OUTPUT`

Command argv:

~~~json
["C:\\Users\\canace\\.cache\\codex-runtimes\\codex-primary-runtime\\dependencies\\native\\powershell\\pwsh.exe", "-Command", "& '.\\native\\scripts\\go.ps1' -GoArguments @('vet','./formats/markdown'); $exit=$LASTEXITCODE; Write-Output \"FINAL_FORMAT_VET_EXIT=$exit\"; exit $exit"]
~~~

Effective command:

~~~powershell
& '.\native\scripts\go.ps1' -GoArguments @('vet','./formats/markdown'); $exit=$LASTEXITCODE; Write-Output "FINAL_FORMAT_VET_EXIT=$exit"; exit $exit
~~~

Session-recorded output:

~~~text
FINAL_FORMAT_VET_EXIT=0

~~~

## Formatter note

Attempt 2 above is the package formatter command. Its output names both `table.go` and `table_column_edit.go`. The accepted result therefore says **no net parser change; the incidental formatter edit to `table.go` was restored byte-for-byte**. The final `table.go` identity in the result handoff matches the frozen comparison copy.

## Limits of this evidence

This file does not claim that the Codex session JSONL is a product runtime log, does not recover any stream separation the executor did not preserve, and does not turn tool-display excerpts into raw process logs. `SESSION-RECORDED OUTPUT` labels the exact unified output available from the command event; the long display truncation is disclosed where it occurred.
