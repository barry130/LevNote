# Markdown task-checkbox contract

Status: **active and approved**. Root accepted the15:12 oversized-preview
release and activated `native/markdown_tasks_stage_plan.txt`. This contract,
including the bodyEnd-bounded separator scan, is authoritative for
`formats/markdown/**`; product implementation is now authorized for that scope.

## Additive format API

Add task metadata to the existing pure format model:

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

`Task == nil` retains the current non-task `ListItem` semantics byte-for-byte.
For a task item, `Task.Range` is exactly three ASCII source bytes: `[ ]`,
`[x]`, or `[X]`. `Checked` is true for `x` and `X`, false for ASCII space.
`Text`, `TextRange`, and `Inline` describe the label after the marker and its
separator whitespace. All ranges remain half-open, absolute byte ranges into
`Document.Raw`; `ListItem.Range`, `Block.Range`, and `Document.Raw` preserve
the complete original source, including line endings and trailing whitespace.

## Recognition rule

Recognize a task only at the start of the body already recognized by the current
simple list parser, for both unordered and ordered items. Let:

- `bodyStart` be the current parser's item body start after the list marker and
  its ASCII space/tab separator.
- `bodyEnd` be the current parser's trimmed item body end: before the LF, before
  a CR preceding LF, and after removing trailing ASCII spaces and tabs.

The item is a task when all of the following hold:

1. `bodyEnd-bodyStart >= 3`.
2. The three bytes at `bodyStart` are exactly `[ ]`, `[x]`, or `[X]`.
3. Either those three bytes end at `bodyEnd`, or the next byte is an ASCII
   space or tab.

When valid:

- `Task.Range = [bodyStart, bodyStart+3)`.
- `Checked` reflects the middle byte.
- Scan only for immediately following ASCII spaces and tabs, and stop the scan
  at `bodyEnd`; never advance to the physical line ending and then compare a
  label start with `bodyEnd`.
- The label starts after the separator scan and ends at `bodyEnd`.
- If no non-whitespace label remains, `TextRange.Start == TextRange.End`; this
  zero-length range is valid and lies inside `ListItem.Range`.
- `Text` is the label slice and `Inline` is parsed from that label with absolute
  source offsets. An empty label has no inline spans.

Examples:

- `- [ ]` and `1. [X]` are empty-label tasks.
- `- [ ]\t` is an empty-label task; the tab is separator whitespace.
- `- [ ]   ` is an empty-label task. Since `bodyEnd` already excludes those
  trailing spaces, the marker ends at `bodyEnd`; the separator scan is empty,
  and `TextRange` is the valid empty range `[markerEnd, markerEnd)`. The
  trailing spaces remain preserved outside that empty label in
  `ListItem.Range` and `Document.Raw`.
- `- [x] **done** 🙂` and ``0. [X] `code``` are nonempty tasks with normal
  inline formatting.

## Whitespace, ordered items, and current-parser boundaries

- The list marker itself keeps its current rules: `-`, `*`, or `+` followed by at
  least one ASCII space/tab, or one to nine ASCII digits followed by `.` or `)`
  and at least one ASCII space/tab.
- Ordered starts remain exactly as the current parser reports them. `0. [ ]`,
  `3. [x]`, leading-zero starts, and `1)` markers can be tasks; task recognition
  does not normalize or rewrite numbering. Existing HTML export continues to
  emit `start` for every value other than `1`.
- A leading-space list such as `  - [ ] label` is not a task because the current
  parser does not recognize indented list starts.
- A task label is one physical item line. Current continuation/nesting behavior
  is unchanged; this stage does not add nested lists or multiline item labels.
- Mixed ordered and unordered items continue to stop the current list block.
- Existing `maxListItems` and document partial semantics are unchanged. A task
  in a partial or oversized document may be represented in a bounded prefix, but
  the view must not activate it and export must refuse the document.

## Invalid lookalikes

The following remain ordinary text with `Task == nil` and current non-task
`TextRange` semantics:

- `\[ ] label`, `\\[x] label`, or any body whose first byte is not `[`.
- `[ ]label`, `[x]word`, and `[X]word`: no ASCII space/tab or item end follows
  the three-byte marker.
- `[y] label`, `[ x] label`, `[X ] label`, full-width brackets, or non-ASCII
  `x`.
- A task marker in a code fence, block quote, paragraph, table warning block, or
  continuation line rather than at a recognized list item body start.
- A body beginning with inline code or quotation, such as `` `[ ]` label`` or
  `"[ ]" label`.
- A line that is not a current-parser list item, including `-[ ]` and `1.[ ]`.

Valid and invalid markers add no parser warning. Invalid syntax is not silently
rewritten; it remains visible source text.

## HTML export contract

For each task item, emit an inert checkbox before the existing formatted label:

```html
<li><input type="checkbox" disabled checked> <!-- rendered label --></li>
<li><input type="checkbox" disabled> <!-- rendered label --></li>
```

Exact rules:

- Checked tasks include one ` checked` attribute; unchecked tasks omit it.
- Always include `type="checkbox"` and `disabled`.
- Emit one ASCII space between the checkbox and a nonempty label. An empty-label
  task emits only the checkbox inside `<li>`.
- Do not emit `id`, `class`, `onclick`, script, network assets, or event
  handlers.
- Reuse the existing `renderInline` label output, escaping and unsafe-link
  behavior unchanged. The three-byte marker is not emitted as text.
- Non-task list output remains byte-for-byte the current output.
- `Document.Partial` continues to cause `ExportHTMLChecked` to return no HTML
  and `ErrPartialDocument`; task metadata does not weaken that refusal.

## Native activation and mutation contract

The visible checkbox is a control over source metadata, never an independent
boolean authority. Its checked state comes from the current parser's
`ListItem.Task.Checked`; readonly, partial, or oversized views show it disabled.

Before activation:

1. Capture host buffer identity and version, the exact `ListItem.Range`, and
   exact `TaskMarker.Range`. Do not identify the target by list index, label, or
   old byte offset alone.
2. Complete pending composition or block draft through the existing `Flush`.
   If flush fails, visibly refuse and retain the draft, source, and caret.
3. Require the same buffer pointer, non-readonly mode, and no pre-existing edit
   group.
4. If the version changed, map both the complete item range and marker range
   through the edit journal. Restore, expired history, target overlap, or invalid
   boundaries visibly refuse.
5. Obtain the current bounded parser result for the authoritative buffer. If it
   is partial or oversize, refuse. It must contain a recognized task whose item
   and marker ranges equal the mapped ranges. Use the current task state, not the
   stale captured state.

Mapping alone is not sufficient. A newly inserted **unclosed** fence above an
otherwise untouched task line makes the current parser consume that line as code;
the mapped item/marker ranges will not match a current task, so activation must
refuse. If the inserted fence closes before the target, the current parser can
recognize the shifted task and activation may proceed. Duplicate labels never
participate in target resolution.

After validation, replace only the middle marker byte:

- unchecked `[ ]` -> `[x]`
- checked `[x]` or `[X]` -> `[ ]`

Use one new edit group containing the one UTF-16 replacement and restoration of
the pre-toggle selection and direction. A failed replacement or selection
restore cancels the group. One successful activation is exactly one undo action;
undo restores the original `x`/`X` spelling and selection, redo restores the
normalized new spelling and selection. Never join an existing group. Call
`Host.Changed` once only after the successful transaction. No whole-document
serialization or additional whole-buffer `String` is needed for the final
one-character mutation after current-model validation.

## Focused acceptance outline

- Parser matrix: all three markers, empty labels, spaces/tabs, CRLF, Unicode,
  ordered starts including `0`, inline formatting, exact absolute ranges, and
  `Document.Raw` equality.
- Negative matrix: escaped/non-ASCII/unclosed markers, `[x]word`, fenced, quoted,
  inline-code, indented, paragraph, quote, and continuation-line lookalikes.
- Export: checked and unchecked attributes, empty/nonempty labels, escaping,
  unsafe links, unchanged non-task output, and partial refusal.
- Native activation: pointer/keyboard, readonly disabled state, exact one-byte
  diff, one undo/redo with reversed selection, duplicate labels, unrelated
  insertion, target edit/deletion, Restore/expired journal/new buffer, pending
  IME, failed flush, and the unclosed-fence contextual refusal described above.
