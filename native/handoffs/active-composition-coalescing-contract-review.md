# Active composition coalescing contract review

STATIC REVIEW ONLY, 2026-11-10. No source edits, tests, builds, products,
benchmarks, or children. This is a narrow contract review for the already
selected post-.28 retention increment, not a new memory audit or architecture
round.

## Finding

The implementation brief is sound. The only contract that needs to be fixed
before implementation is ownership: an Editor-held `*text.Buffer` plus a Boolean
`composing` flag cannot prove that the Buffer's current group is still the group
Editor opened. A same-buffer `Restore`, ordinary `CancelGroup`/`EndGroup`, or a
new group can make `cancelComposition` target unrelated current state. Do not
carry a `*transaction` or copied transaction in Editor.

## Smallest API

Use one opaque, Buffer-issued token and keep the transaction exclusively in
Buffer:

```go
type CompositionToken struct { id uint64 } // fields inaccessible outside text

func (b *Buffer) BeginCompositionGroup() (CompositionToken, error)
func (b *Buffer) ReplaceCompositionUTF16(token CompositionToken, start, end int, value string) error
func (b *Buffer) EndCompositionGroup(token CompositionToken) error
func (b *Buffer) CancelCompositionGroup(token CompositionToken) error
```

The exact names are flexible; the four capabilities and opaque token are the
contract. `BeginCompositionGroup` captures `transaction.before` exactly as
`BeginGroup` does. The token contains an identity/generation value, never a
transaction pointer, edit slice, old/current string, or document snapshot.

The Buffer must invalidate that identity whenever the owned group ends, cancels,
is reset by `Restore`/`ResetHistory`, or is otherwise replaced. Every typed
operation must validate the token before touching state. Ordinary `BeginGroup`
must still reject an existing group. While a typed composition group is active,
ordinary group/replace methods must not silently append to, end, or cancel it;
conversely typed methods must never adopt an ordinary/foreign group. A stale
token should fail without mutation, and Editor must clear its local token even
when cancellation finds it stale.

This extra kind/token check only changes the new typed group. Existing ordinary
group callers retain their current behavior.

## Coalescing constraints

1. The first accepted composition replacement records the normalized UTF-16
   start, the original replaced selection/text, and the current accepted value.
   A same-text first preedit must still establish this baseline so the original
   direction-bearing `transaction.before` selection and the initial full span
   are recoverable even when no text edit was created initially.
2. Only a consecutive replacement whose ordered range exactly equals the
   previous accepted marked segment, including its current UTF-16 length, may
   replace the last aggregate `change.next`. Keep its first `old` unchanged and
   update the aggregate end, current value, selection, byte contribution, and
   per-update version/callback behavior.
3. An explicit range that is not the exact previous marked span appends with
   today's ordinary semantics and becomes the new coalescible suffix. Later
   full-span updates may compact only that final suffix. Never claim arbitrary
   irregular ranges are bounded or retroactively rewrite earlier commands.
4. If accepted updates return the current text to the original aggregate text,
   remove the net-zero text change from the active transaction. EndGroup then
   commits no fake undo record; the Buffer version remains the result of the
   actual accepted updates. Cancellation still restores the captured original
   selection.
5. The final native `ReplaceText` path must, while composing, call
   `ReplaceCompositionUTF16` for the current marked span and then
   `EndCompositionGroup`; it must not use ordinary `ReplaceUTF16` followed by a
   tokenless end. Same-text final commit must still close ownership and emit the
   existing final-selection/callback behavior.
6. A text-changing accepted marked update still advances the Buffer revision
   through the normal `replace` path and Editor still emits exactly its existing
   one change callback. A same-text update adds no text revision and no extra
   callback. Cancel/undo/redo replay the aggregate edit(s), so a coalesced
   successful edit has one undo step and cancellation can advance by fewer
   revisions than an N-intermediate implementation. Do not synthesize the old
   intermediate revision count.
7. On suffix replacement, overwrite the previous `change.next` reference,
   release any temporary previous-value reference, and never retain an
   intermediate-value list for later repair. Size admission still uses existing
   UTF-8/range/surrogate ordering. Retained-byte accounting must use checked or
   saturating arithmetic so a large irregular sequence cannot overflow into a
   negative value and evade the existing 32 MiB history limit.
8. Refusal remains atomic: preflight/admission failures, rejected strings,
   foreign groups, stale tokens, and a replaced Buffer leave source, selection,
   history/redo, and existing group contents unchanged. An ongoing refusal
   keeps its token/source only for explicit retry or local Escape; Escape clears
   the local token after the one guarded cancellation attempt.

## Reachability boundary

Host-reachable composition uses `InputCompose` with a nil range, resolved to the
current complete marked range, and the final `InputText` commit through
`ReplaceText`. Those are the paths that must prove bounded coalescing, sticky
refusal, retry, local/global Escape, focus/release, and lifecycle blocking.

Direct same-Buffer calls to `CancelGroup`, `EndGroup`, `BeginGroup`, or
`Restore` underneath an active Editor composition are unsupported ownership
mutations. The token contract must fail closed, but it need not promise a
coherent composition after a caller violates that ownership. Those paths are
regression tests for accidental cross-cancel protection, not additional UI
features or a universal transaction scheduler.

## Reviewed source identities

```text
0AF9281A6051B8715854665043CBCC3EA9DFBB89BA5D028BA719B312B244FB21  native/core/text/buffer.go
09BEA640C8F1953E615B43088A7F58FA25AA8342B59694CC4023CBD36A885EBD  native/ui/editor/editor.go
```
