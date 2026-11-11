# Active composition coalescing — implementation frozen

2026-11-10 Astra. Implemented the assigned compact IME history contract in
exactly2runtime files and2new focused test files. Both focused runs passed first
attempt; source is frozen/released for independent review and the root-activated
final gate. No children, full suite/vet, package/host, worker/resource run,
screenshot or benchmark. Accepted .28 artifacts remain immutable.

## API and ownership

`core/text.Buffer` now exposes `BeginCompositionGroup`,
`ReplaceCompositionUTF16`, `EndCompositionGroup`, `CancelCompositionGroup` and
the side-effect-free `OwnsComposition`. `CompositionToken` is opaque and contains
only a private pointer to a nonzero-sized identity allocation, no transaction,
buffer, snapshot or text. Buffer's current composition state validates both the
identity and issuing-buffer pointer. Tokens from another buffer, a shallow copy,
an ended/cancelled group, Restore or ResetHistory cannot mutate/end/cancel state.

Tokenless Replace/End/Cancel refuse a typed composition; ordinary BeginGroup
remains append-based and rejects existing ownership. Typed state is dropped on
end/cancel/reset. Editor retains its original buffer owner+token, uses the typed
replacement for marked and final input, and guarded end/cancel for Flush,
UnmarkText and Escape. Stale Escape makes one guarded cancellation attempt and
clears local owner/token without cancelling the replacement group.

## Retention and behavior

Only a consecutive replacement of exactly the previous full UTF16 marked span
coalesces. It reuses the last retained `next` as current text, avoiding the old
slice allocation, and replaces that suffix's `next` while preserving its first
`old`. No whole-document snapshot/intermediate list is introduced. An irregular
explicit range appends and begins a new suffix; preceding suffixes stay intact.

A same-text first preedit retains a baseline and original direction-bearing
selection without a fake edit. Returning a suffix to its original text removes
and zero-clears its edit slot while retaining one baseline for later updates or
cancellation. Completing a net-zero group creates no undo action and preserves
prior redo. Text-changing accepted updates retain per-update revisions; every
accepted marked update, including same text, still emits its existing callback.
Same-text final commit still closes ownership and emits the existing callback.
Aggregate undo/redo/cancel replays the retained endpoint edits, not intermediate
revisions. Commit continues to use normal history truncation.

Retained-byte addition saturates with an explicit overflow bit. Shrinking or
removing a suffix recounts all retained edits when the old count overflowed;
subtracting from an unknown saturated total never clears overflow blindly.
History totals also saturate and overflow forces existing trimming, even with
the configured byte limit at max-int. Ordinary finite accounting is unchanged.

All .28 size refusal semantics remain: atomic failed replacement, sticky first/
ongoing/final IME refusal, retry/Escape resolution, no rejected string retention,
and Save/Select/Close blocking. First-refusal final retry also refuses a foreign
ordinary group opened in the interim. Existing focus/release and local/global
Escape behavior is preserved.

## Verification

Fresh [evidence directory](active-composition-coalescing-evidence-20261110/).
Both commands used scripts/go.ps1, captured its native-Go `$LASTEXITCODE`
immediately, saved separate stdout/stderr and Go/process UTC+exit records, then
exited with that captured code. No failures or corrective reruns occurred.

| Run | Focus | UTC command start → end | Go/shell | Output |
| --- | --- | --- | --- | --- |
| 01-focused | text/editor new tests + existing growth/composition/indent seams | 11:23:26.1815874Z → 11:23:29.1689428Z | 0/0 | text0.594s; editor1.286s |
| 02-lifecycle-seams | selected Markdown/graph/app growth and composition lifecycle seams | 11:24:16.1154356Z → 11:24:28.8714228Z | 0/0 | Markdown1.782s; graph1.685s; app2.857s |

Date for both rows2026-11-10. Exact package/regex arguments are in
[01-focused.ps1](active-composition-coalescing-evidence-20261110/01-focused.ps1)
and [02-lifecycle-seams.ps1](active-composition-coalescing-evidence-20261110/02-lifecycle-seams.ps1).
Both stderr files are empty; result JSONs distinguish Go from shell status.

Twelve new top-level tests prove600accepted full-span updates retain at most
one edit for selected and insertion spans, original/current strings only,
net-zero cleared backing slots, revisions/selections, final/cancel/redo and
irregular suffix fallback. Tiny tests exercise validation atomicity, cross-buffer
and reset token refusal, saturation/recount and ordinary history budget policy.
Public Tester cases send240updates through native transport, assert exact
callbacks (same-text included), local/global Escape, final commit and undo/redo.
Restore/reset tests prove stale Escape cannot cancel ordinary or typed foreign
groups. Existing real app Save/Select/Close and Markdown/graph recovery gates
all passed without app/format runtime changes.

## Frozen source identities

Native-relative Path,Length,Sha256 in
[source-frozen.json](active-composition-coalescing-evidence-20261110/source-frozen.json),
SHA256 `5673AB5368E21B7F81631418448C0D4B20B8ECC3A274906E6BE3EBF77EB47213`.
All4files are unchanged from the first focused-run inventory.

| Path | Bytes | SHA256 |
| --- | ---: | --- |
| core/text/buffer.go | 23353 | D02837C2AD8EC739E8058B4ECFC479A3C537B85233653902C4E1069079ECE057 |
| ui/editor/editor.go | 24397 | B5D98594BF9213C0D2B0EF2B69C1C00040ED5E365FF133A11BF813AABFD9366A |
| core/text/composition_test.go | 12624 | C57A16E814798699705F7C3F1062967E6E37E4434C80603E133E3C792F88A3B3 |
| ui/editor/composition_test.go | 7948 | CDACCCD4EBDE965CD3CCD2B8366B843FD483FDFB39B94AC8EB63B87D39ACFF7D |

[Closed evidence identities](active-composition-coalescing-evidence-20261110/evidence-files.json)
SHA256 `EF7D505085AD51E0D935E020BEC84F7AF200F9AABED8EEA0ED54338C4E668D16`.

## Limits

This is a structural retention proof for consecutive full-span IME updates,
not an RSS/heap or physical Windows IME measurement. Irregular explicit ranges
may retain an unbounded number of distinct suffix edits; no universal group cap,
forced commit or data drop was added. Buffer admission still bounds logical
growth separately from caller allocations, history and snapshots. Direct reset
under active input is a defensive ownership case, not a supported UI sequence.
No additional implementation or execution follows this handoff until activated.
