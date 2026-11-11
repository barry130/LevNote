# Active composition coalescing — independent final review

2026-11-10. Static read-only review of the frozen composition source and the
owner's focused evidence. I made no source edits and ran no tests, builds,
product instances, resource measurements, or physical-input checks.

## Identity

All four current source/test files match
[source-frozen.json](active-composition-coalescing-evidence-20261110/source-frozen.json)
exactly. The manifest SHA-256 is
`5673AB5368E21B7F81631418448C0D4B20B8ECC3A274906E6BE3EBF77EB47213`.

| Native-relative path | Bytes | SHA-256 |
| --- | ---: | --- |
| `core/text/buffer.go` | 23353 | `D02837C2AD8EC739E8058B4ECFC479A3C537B85233653902C4E1069079ECE057` |
| `ui/editor/editor.go` | 24397 | `B5D98594BF9213C0D2B0EF2B69C1C00040ED5E365FF133A11BF813AABFD9366A` |
| `core/text/composition_test.go` | 12624 | `C57A16E814798699705F7C3F1062967E6E37E4434C80603E133E3C792F88A3B3` |
| `ui/editor/composition_test.go` | 7948 | `CDACCCD4EBDE965CD3CCD2B8366B843FD483FDFB39B94AC8EB63B87D39ACFF7D` |

The freeze verification records the same four files unchanged from the first
focused run through the final evidence capture.

## Conclusion

No remaining source-grounded blocker found in the exact four-file union.

The implementation satisfies the root refinements:

- `CompositionToken` binds a nonzero private identity to the issuing `Buffer`
  and current composition group. Foreign, stale, shallow-copy, ended, cancelled,
  `Restore`, and `ResetHistory` ownership cannot mutate, end, or cancel the live
  group. Tokenless typed mutation and end/cancel are refused; ordinary groups
  retain their append-based API and cannot own/cancel a typed group.
- Exact consecutive full UTF-16 span replacement reuses the retained suffix
  endpoints. Full-span histories retain original/current transaction text in
  O(1), while irregular explicit ranges take the disclosed append fallback.
- Same-text first preedit retains the baseline and direction-bearing original
  selection without a fake edit. A net-zero active aggregate clears its edit
  slot, creates no undo action, and preserves prior redo. Overwrite and
  end/cancel/reset release the previous/current ownership references.
- Accepted `SetMarkedText` continues to call `changed` once, including same-text
  input; coalescing adds no callback. Final native replacement, `Flush`,
  `UnmarkText`, and local Escape use the guarded typed operations. A stale local
  Escape cannot cancel a newly opened ordinary or typed foreign group.
- Retained-byte and history-total accounting saturate safely. Suffix shrink or
  removal recounts retained edits after overflow; an unknown saturated total is
  never incorrectly cleared, and history trimming remains forced while overflow
  is known.

## Verification boundary

The source tests substantively exercise 600 full-span updates for insertion and
selected replacement, endpoint-only retention, same-text/net-zero behavior,
irregular suffix fallback, ownership refusal, prior-redo preservation, reset and
restore staleness, accounting overflow/recount, and native callback/final/Escape
paths. Owner evidence reports both focused commands passing on their first
attempt, with Go and shell exit `0`, empty stderr, and these timings:

- `01-focused`: 2026-11-10 11:23:26.1815874Z to 11:23:29.1689428Z; `core/text`
  0.594s, `ui/editor` 1.286s.
- `02-lifecycle-seams`: 2026-11-10 11:24:16.1154356Z to 11:24:28.8714228Z;
  Markdown 1.782s, graph 1.685s, app 2.857s.

There was no failed composition run or corrective source rerun to review.
These checks support the structural history, rejection, lifecycle, and callback
claims only. They do not establish RSS/heap behavior, whole-app memory, physical
Windows IME behavior, or a universal bound for irregular-range suffix history.
That retention fallback is disclosed and is not a blocker for this increment.
