# Markdown task UI — independent current-source review

Reviewer: GLM formats agent. This is a read-only review of the current
`modules/markdownview` checkpoint after Astra's final wording, clean-draft
invalidation, and immutable-snapshot fixes. No product source was modified and
no test suite was rerun by this reviewer; Astra's focused package evidence and
DeepSeek's app/view gates remain the execution evidence.

## Result

**No actionable defect found.** The current implementation satisfies the
reachable data-integrity requirements in `native/markdown_tasks_stage_plan.txt`
and `handoffs/markdown-tasks-contract.md`.

## Reviewed behavior

- `tasks.go` captures numeric buffer/version identity plus the complete item
  range and exact three-unit marker range. Activation never resolves by list
  index, row position, or label. Duplicate labels therefore cannot redirect a
  command.
- `toggleTask` flushes first, then rechecks live buffer identity, readonly
  state, foreign transaction state, and the oversize guard. It maps both the
  item and marker ranges, performs a current bounded parse when the cached
  preview is stale, refuses partial input, and requires the current parser to
  recognize the exact mapped task before mutation.
- The mutation replaces only the middle marker unit, restores the captured
  selection and direction inside one new group, and notifies `Host.Changed`
  once only after success. Failure paths cancel the group and leave source,
  selection, and pending draft intact.
- `findTask` walks ordered list-item ranges with a cursor and compares both the
  whole item and marker ranges. It does not perform a repeated full-prefix UTF-16
  scan for each visible row; `previewRow.span` supplies the item anchor and only
  the short marker prefix is measured.
- The checkbox input handler is the innermost enabled input handler. Pointer
  and Space activation therefore cannot fall through to the parent row's
  double-click draft handler. A control built before `host.ReadOnly` becomes
  true still checks live readonly state and refuses without mutation; the
  rebuilt control is visibly disabled.
- `previewSnapshot` keeps the doc, rows, buffer, and version immutable for the
  active List build. A reentrant `Host.Changed`/`Warnings` refresh cannot mutate
  the arrays being rendered. The exact own-toggle row identity is carried into
  the next parse only after its one-character range maps successfully.
- The clean mapped-draft fix invalidates the parsed preview and detaches the
  old block index when an external edit changes the source version, while
  retaining the mapped independent field. Dirty or conflicting drafts continue
  through the existing recoverable path.
- Readonly wording is now accurate (`未切换任务状态` and `只读预览`), and empty,
  Unicode, clipped, ordered, and large-label cases use bounded source-derived
  metadata.

## Double-click target review

The parent row still calls `begin` with a block captured by its frame snapshot,
while `begin` reads current `v.doc`, `v.version`, and `v.docBuffer`. I found no
reachable sequence where that mismatch edits the wrong target:

1. Input callbacks run between frames, not while the immutable List rows are
   being constructed; reentrant task toggles occur during construction but do
   not invoke the row's double-click callback.
2. `begin` ignores the stale block index and validates the captured block's
   buffer identity, byte bounds, mapped UTF-16 range, and exact current source
   bytes.
3. It then reparses the authoritative buffer and accepts a draft only for a
   current block with the same kind and exact byte range. A contextual fence,
   replacement, deletion, stale buffer, Restore, or expired journal therefore
   reports conflict instead of opening the old block.

This is a current-source conclusion, not a new test requirement.

## Scope note

This review does not expand the supported list grammar or add new behavior. It
supports freezing the task-checkbox stage in its approved bounded scope.
