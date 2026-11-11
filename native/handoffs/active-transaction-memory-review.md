# Active IME transaction retention review

2026-11-10T10:51:10Z. Read-only, static source audit. No implementation,
test/build/product run, benchmark, or memory measurement. This is a candidate
for a later increment and does not expand the current .28 work.

## Finding

A reachable IME sequence can retain every accepted intermediate marked string
until composition ends. The current 128 MiB logical-content ceiling and the
32 MiB/256-entry history limits do not bound that active-group sequence.

Exact production path:

1. `native/ui/editor/editor.go:296-332`, `SetMarkedText`, calls `BeginGroup` on
   the first accepted composition and retains that group across subsequent
   marked updates.
2. `native/ui/editor/editor.go:622-630`, the actual `InputCompose` handler
   always passes a nil replacement range. While composing,
   `replacement()` at260-267 resolves that to the complete current marked
   range.
3. Every accepted `Buffer.ReplaceUTF16` call appends one
   `change{start, old, next}` to `b.group.edits` at
   `native/core/text/buffer.go:297-320`. `old` is materialized by
   `sliceBytes`; `next` retains the event value. The same call clones accepted
   `next` text into document pieces through `piecesOf`, so they can be distinct
   backings even though sharing is also possible.
4. The group ends only at a successful commit/cancel route
   (`editor.go:270-294`, `334-345`, Escape at696-702). Nothing prunes
   `group.edits` while the group is active.

Trigger: start one composition over a selected or empty span, then deliver
many accepted marked updates whose replacement range is the complete prior
marked span, for example `a`, `ab`, `🙂`, `你🙂`, back to `a`, and so on. Each
state may be small and legal while the sum
`len(change.old)+len(change.next)` over the unbounded number of callbacks grows
without limit.

## Bounds and accounting

- The current buffer is finite by policy: ordinary accepted content cannot
  exceed the 128 MiB base ceiling, with the documented oversized-seed ceiling.
  That bounds each logical state, not the number of intermediate states.
- The active group also holds an edits slice and its backing capacity. Its
  size is linear in the number of accepted marked updates.
- `old` strings are newly materialized by `sliceBytes`. `next` strings may
  retain the caller's event backing; some may be shared, and there is no
  promise that all `old`/`next` values are uniquely allocated. Source inspection
  therefore shows reachable retention, not a measured byte or heap total.
- `History()` at `buffer.go:402-410` reports only `undo`/`redo`, not the active
  group. `SetHistoryLimits` and `trimHistory` at392-430 do not inspect or trim
  `b.group`.
- `commit()` trims only after `EndGroup`. If the committed group is within the
  budget it remains one undo action; if it exceeds the budget, existing policy
  may trim it or older transactions and set `Truncated`. No new history rule is
  needed.

## Smallest safe follow-up

Add an IME-only coalescing path used by `Editor.SetMarkedText`; leave ordinary
`BeginGroup` callers and every command transaction unchanged.

The first accepted marked replacement should record one aggregate change:
`start`, the original replaced text, and the current marked value. While the
group is active, a subsequent accepted update may replace that aggregate when
its normalized range exactly covers
`[start, start+UTF16Len(current marked value))`. For that case:

- Keep `transaction.before` and the first `old` text unchanged.
- Replace only the aggregate `next` with the new accepted value.
- Set `transaction.after` to the new accepted selection as today.
- Recompute its retained-byte count as `len(old)+len(next)`, releasing the
  previous intermediate `next` reference from the active transaction.
- Keep exactly one edit in the IME transaction and leave version/current buffer
  already updated by the normal replacement path.

Changing lengths are handled by recomputing the UTF-16 end after every accepted
value. The range comparison uses UTF-16 units and the buffer's existing
validation, so surrogate pairs are never split. The current production
`InputCompose` route is the exact full-span shape above. An explicit public
`SetMarkedText` range that does not exactly cover the aggregate must fall back
to the existing append path; it must not guess or silently discard a range.

This preserves the required behavior without a universal size cap:

- Cancel replays the single aggregate change in reverse, restoring the exact
  original source and `transaction.before` selection.
- EndGroup commits one ordinary transaction, so composition undo/redo remains
  one action and normal history trimming/truncation is unchanged.
- Prior redo remains untouched during the active group and cancellation, as it
  is today; only a successful normal commit clears redo.
- `SetMarkedText` keeps its existing `InGroup` refusal, so a foreign group is
  never adopted or modified.
- Views still receive the same per-update versions and `OnChange` calls. The
  compaction changes only transaction metadata after an accepted replacement
  and must not emit another callback.
- A rejected replacement or the current sticky refused-composition state must
  leave the aggregate, bytes, selection, and source unchanged.

Do not add a group-level byte cap, force a commit to meet a budget, drop the
current group, or retroactively compact ordinary commands. Those would change
transaction or history semantics for reasons unrelated to this path.

## Tiny-limit test strategy

In `native/core/text`, use a small buffer and history limit, inspect the
package-private group, and:

1. Begin an IME composition over existing text and apply several accepted
   full-span updates with changing ASCII, Unicode, and surrogate-pair lengths.
   After each, assert one edit, unchanged original `old`/`before`, correct
   current `next`/after-selection, and retained bytes equal to only the original
   plus current marked strings. Assert data/version behavior still follows each
   accepted replacement.
2. Cancel with an earlier redo entry present. Assert exact original text and
   selection, no group, one action's transaction state was not committed, and
   the prior redo still replays.
3. Repeat accepted updates, EndGroup, and assert one undo/redo step. Exercise a
   subsequent rejected update and foreign-group refusal without changing the
   owned or foreign group.

In `native/ui/editor`, extend the existing composition tests with
`ui.Tester.Compose` values such as `a`, `ab`, `🙂`, and `你🙂`; assert
`OnChange` still fires once per accepted update, no extra callback comes from
coalescing, Escape restores the exact source/selection, and successful commit
undoes/redoes in one action. These structural assertions prove the bounded
transaction representation; they are not a process-memory benchmark.

## Inspected snapshot

```text
3568bbae46d0931446ee59a211c23386ad9e68ecc0e2b95f8ed827c729b2bfc8  native/core/text/buffer.go
c108fc2107da573f0919a73acd71fdea1bdb5f2b4456b491dba07043609af9c5  native/ui/editor/editor.go
8ae4894024e97b135fd49d32774afad7edca4ce01a87e7cf62101ad67f62a5a2  native/core/text/retention_test.go
10063c656ce4a2c0b0ba0d1c81ffb236485735e5f2a5126ec6549978f6fc5abe  native/ui/editor/growth_test.go
0ac7fb907c83dc3c2b7642b197441492274cbb1cbc75e6688264d577d995adb4  native/app/app.go
```
