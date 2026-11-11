# Doc-block reference view independent review

Status: **final frozen current-source review; no findings**.

Reviewed: `native/modules/markdownview/reference.go`,
`reference_test.go`, and the current `view.go` lifecycle changes. No product
files were edited and no duplicate peer tests were run.

Owner-reported focused evidence after the final narrow correction:

```text
affected reference tests: 1.120s PASS
prior full owned modules/markdownview package: 1.907s PASS
vet: clean
```

Root reviewed both results and all app screenshots. The `1.120s` result is the
affected reference subset after the final guard change, not a repeat of the full
package. This independent review accepts that evidence; it does not claim a new
execution.

## Final verification

- The `Clicked` branch now requires `captured.buffer == target.buffer` and
  `captured.rowID == target.rowID` before activation. A reused fixed-key button
  cannot consume another reference row’s captured gesture.
- A second visible, structurally valid reference with the same ID/title is
  tested through both a real pointer press and focused Enter. It captures the
  second row and opens `targets/b.txt`, proving source-range identity rather
  than ID/title matching.
- Held-gesture invalidation tests are meaningful:
  - nil and replacement buffers change the actual host buffer and verify the
    capture and pending token are dropped with a visible refusal;
  - an oversize destination is changed after pointer-down and refuses without
    opening;
  - a partial parse is produced after pointer-down and refuses at activation;
  - `Restore`, an expired 65-edit journal, a foreign group, and a replaced
    buffer all reject while preserving buffer text, version, selection, and
    history.
- The release-outside test now asserts both the capture and pending token are
  cleared and that a later frame rebuilds to the current buffer version.
- Canceled pointer input and virtualized-out rows clear the captured gesture
  without opening or retaining obsolete identity.
- The current code no longer contains the unused `referenceCurrent` helper or
  its unreachable trailing return.
- `activateReference` validates the target bound, flushes first, requires the
  same buffer, rejects groups and oversize source, maps both enclosing and
  header ranges, requires a current exact `BlockDocBlock` with empty
  `ReferenceProblem`, explicit reference mode, exact header/enclosing ranges and
  exact full `src`, then calls `Host.OpenReference` once. No stale source data or
  view cache is used after synchronous Release.
- Failed Flush retains the draft and recovery controls; navigation adds no
  history transaction. Oversize targets, metadata, and tooltips stay bounded
  and rune-safe.

No actionable source defect remains. The doc-block reference view is ready for
the integration gate.
