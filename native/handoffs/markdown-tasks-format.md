# Markdown task metadata/parser/export checkpoint

Scope: `formats/markdown/**` only, under the active
`native/markdown_tasks_stage_plan.txt` and approved
`native/handoffs/markdown-tasks-contract.md`.

Early compiling API checkpoint:

```go
type TaskMarker struct {
    Range   SourceRange
    Checked bool
}

type ListItem struct {
    Range     SourceRange
    TextRange SourceRange
    Text      string
    Inline    []InlineSpan
    Task      *TaskMarker
}
```

`Task == nil` preserves current non-task semantics. The parser recognizes tasks
only at an already recognized unordered/ordered item body start; valid markers
are exactly `[ ]`, `[x]`, or `[X]` followed by ASCII space/tab or the item's
already-trimmed body end. Separator scanning is bounded by `bodyEnd`, yielding
a valid empty `TextRange` for trailing-whitespace tasks. `Document.Raw`, item
ranges, list starts, parser budgets, and partial semantics are unchanged.

HTML task items emit an inert `<input type="checkbox" disabled>` plus `checked`
only when checked, followed by the existing escaped/formatted label. Partial
documents still return no HTML and `ErrPartialDocument`.

Final checkpoint: task recognition, checked HTML export, and contract-focused
tests are complete and frozen.

Focused results (latest run after the authorized formatting-only pass):

```powershell
native/scripts/go.ps1 -GoArguments @('test','./formats/markdown','-count=1')
# ok levnote.local/native/formats/markdown 0.521s

native/scripts/go.ps1 -GoArguments @('vet','./formats/markdown')
# exit status 0, no diagnostics
```

Coverage added: all three markers; empty labels; space/tab separators; CRLF and
trailing whitespace; unordered/ordered numbering including `0` and parenthesized
markers; Unicode and inline formatting; explicit numeric absolute byte ranges;
invalid, escaped, fenced, quoted, inline-code, indented, paragraph, quote, and
continuation lookalikes with ordinary `Text`/`TextRange` preservation; unsafe
link refusal with escaped label output; exact non-task list export compatibility;
inert checked/unchecked HTML; and `Partial` export refusal with
`ErrPartialDocument`.

Astra can consume the exact three-byte `Task.Range`, whole `ListItem.Range`,
post-marker/separator `Text`, `TextRange`, `Inline`, and `Task.Checked` fields.
DeepSeek can proceed against the published API. No further source changes are
planned in `formats/markdown/**` unless root assigns a concrete defect.
Housekeeping completed under the explicit root exception:
`gofmt -w formats/markdown/parser.go formats/markdown/tasks_test.go`;
`gofmt -d` then reported no remaining diff, the focused package test passed, and
vet remained clean. `native/` is an untracked checkout within the outer Git
repository, so a tracked-file `git diff` is unavailable; the formatting commands
and status check above cover the requested changed-file verification.
