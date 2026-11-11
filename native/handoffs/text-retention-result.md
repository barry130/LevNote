# Core/text retained-piece audit result

2026-11-09. Scope stayed in `native/core/text`. Accepted `.10` inputs were
not changed and no build, launch, UI call or resource matrix was run. The
parallel app `removeTab` correction remains with its separate owner; the
single combined dependent fullapp check was run after that owner published
readiness.

## Confirmed defect and correction

`Buffer.replace` made `out`, compacted it in place with `merged := out[:0]`,
and assigned `b.pieces = merged` without clearing entries beyond the new
length. Those entries are invisible to ordinary slice iteration but keep chunk
pointers reachable through the live backing array.

The final focused fixture edits one byte inside a 4 KiB chunk so coalescing
shrinks three output pieces to one, undoes and redoes that edit, then calls
`ResetHistory`. Snapshots taken before and after the edit remain alive across
the whole sequence. Before the correction the test exited 1 with:

```text
--- FAIL: TestReplaceReleasesDeadPieceTailAfterHistoryReset (0.00s)
    retention_test.go:70: piece backing retains 2 dead slots beyond logical length 1 (capacity 5): 2 stale-only slots / 4097 stale-only bytes; 0 duplicate-logical slots; 4097 unique referenced bytes total
FAIL
FAIL	levnote.local/native/core/text	0.532s
```

These two pointers were stale-only: neither referenced chunk occurred in the
live piece slice. One was the replaced 1-byte chunk and one was the original
4,096-byte chunk. This is retention through the live backing array, separate
from undo/redo, since history was reset. An earlier undo-only diagnostic showed
why the distinction matters: one dead slot referenced a chunk that the
remaining logical piece still needed. That slot was waste, but clearing its
pointer did not free the chunk. The final regression counts stale-only bytes
separately from duplicate pointers to legitimately live chunks.

The minimal fix is at `native/core/text/buffer.go:622-624`:

```go
clear(out[len(merged):])
b.pieces = merged
```

It clears only dead slots in the temporary output backing after coalescing.
Logical pieces, undo/redo transactions and snapshots are unchanged. Snapshots
already own copied piece slices and continue to reference immutable chunks.
The regression verifies both the pre-edit and post-edit snapshots after undo,
redo and history reset.

## Substring ownership finding

No production change was made for the second hypothesis.
`ReplaceUTF16` stores `change.next = value` directly, so a caller that passed a
small slice of a much larger string could make `RetainedBytes` count only the
small logical replacement while history kept the larger allocation alive.
The audited production callers do not currently create that condition:

- `ui/editor/editor.go` receives MyGo text input and paste values. Windows
  `WM_CHAR` creates per-rune strings; IME completion uses `imeString`, which
  allocates from the returned UTF-16 length before converting it to a string.
  Paste has its own whole clipboard value.
- `modules/markdownview/draft.go:97` and `table_cell.go:446` materialize the
  complete draft with `Buffer.String()` before flushing it.
- Table row/alignment values are rebuilt patch strings; task markers use the
  constants `x` and space; graph edits pass a whole serialized value.
- `change.old` comes from `SliceUTF16`, whose `sliceBytes` writes into a new
  `strings.Builder`; it does not point into the source chunk or caller string.

Adding `strings.Clone` to every replacement would impose another copy on normal
typing even though no current production caller supplies the problematic
substring. The residual API-level ownership hazard is documented here but was
not treated as sufficient reason for a broad change. `RetainedBytes` is
accurate for the present callers because their retained `next` strings are
materialized at logical replacement size.

## Verification and hashes

All Go commands used the required wrapper.

```text
> native/scripts/go.ps1 -GoArguments @('test','./core/text','-run','^TestReplaceReleasesDeadPieceTailAfterHistoryReset$','-count=1')
ok  	levnote.local/native/core/text	0.554s
exit 0

> native/scripts/go.ps1 -GoArguments @('test','./core/text','-count=1')
ok  	levnote.local/native/core/text	0.635s
exit 0

> native/scripts/go.ps1 -GoArguments @('vet','./core/text')
exit 0 (no output)
```

After the app owner published its green close/fix evidence, the one combined
dependent pass also exited 0:

```text
> native/scripts/go.ps1 -GoArguments @('test','./core/document','./ui/editor','./app','-count=1')
ok  	levnote.local/native/core/document	0.713s
ok  	levnote.local/native/ui/editor	2.395s
ok  	levnote.local/native/app	15.388s
exit 0
```

Original `buffer.go` SHA-256:

```text
603307BE3CC825C97F245579C1D8FA4F242F463FA5B6F5DC8AAE68399856AC67  native/core/text/buffer.go
```

Final changed files:

```text
562F1293800CF147FE466F1B64E29FA5627591F1E8CCC698FEAE9163534E2E9A  native/core/text/buffer.go
8AE4894024E97B135FD49D32774AFAD7EDCA4CE01A87E7CF62101AD67F62A5A2  native/core/text/retention_test.go
```

The existing `buffer_test.go`, `range_map.go` and `grapheme.go` hashes remained
unchanged. This proves deterministic retained-pointer ownership in the package;
it does not claim a process RSS reduction or substitute for the unavailable UI
resource comparison. The combined pass covered both concurrent corrections in
one non-build run. The concurrently changed app files remained:

```text
6387CDF9938D3546C26CD8D387CA4B334A011A2A47456CBCB300E0AA5890527A  native/app/app.go
B0CA0D1E70A2C58F3F81EA943081550F6B0D3DF368A25B702E0562BCE241C316  native/app/tab_retention_test.go
```

Accepted `.10` remained unchanged at 23 files / 23,622,214 bytes. Host hash:

```text
4EB096FAEA7CA79E71F0C9FF30E659478C0E2AA135803DDBF7C0AEE47A0AEF6C  native/bin/levnote-native-worker-20261109.10/levnote-native.exe
```
