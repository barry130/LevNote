# Native search navigation: implementation result

2026-11-09. The bounded implementation under
`search-navigation-implementation-task.md` is complete and frozen for root and
one independent read-only review. No build, launch, capture, resource matrix,
product docs, search-engine change, editor/text API change, dependency change
or `.12` package/evidence change was performed.

## Behavior and ownership

Previous/next now uses ordered UTF-16 selection bounds. A nonempty selection
that exactly equals a match advances, either selection direction, with wrap.
Otherwise next chooses the first match whose `Start >= selection.End`, and
previous chooses the last whose `End <= selection.Start`. Exact wrapping,
zero-width progress, reversed selections, UTF-16 emoji and boundary matches
have focused regressions. One result wraps to itself.

The app owns one active-document cache with exact `Tab`, `Document`, `Editor`
and `Buffer` identities, the final buffer version, `search.Options`, result
error, last index and last returned selection. Retained match data is only a
new `[]findMatchOffset` of `{Start,End int}` values. It retains no `Snapshot`,
source string, `search.Match`, submatch array or per-tab cache. The cold path
temporarily uses the existing shared `search.Find`, copies offsets, then lets
the snapshot/matches die. Successful cold search allocates the offset slice;
an `ErrTooManyMatches` result allocates no offset capacity and never navigates
on partial matches.

Every navigation still flushes the current editor/rich view first. A failed
flush keeps the active document, presentation, selection, history and still
valid warm cache. Tab/document/editor/buffer identity is checked before the
cold scan and before reveal, so a tab switch or identity-changing Flush cannot
apply an abandoned result. Replace/ReplaceAll remain on fresh search authority;
an actual replacement invalidates navigation and the subsequent find rebuilds
from the edited buffer.

The cache is dropped on committed tab add/switch/remove, successful reload,
plugin reconfigure/rematch, encoding reconfigure, shutdown, Find close and
replacement. Query/options/version mismatch is checked cheaply and dropped on
the next rendered frame or by the next find action. Failed tab switch, close
and Find flush preserve the active document and cache. There is no timer,
worker or per-frame source conversion.

## Actual red and focused evidence

Run order and exact command/output/exit records are preserved in
`native/handoffs/search-navigation-evidence/`:

1. `red-adjacent-navigation.log` is the original product behavior red. The old
   selector stayed on `text.Range{Anchor:2, Head:4}` instead of wrapping to
   `[0,2)`; exit 1.
2. `focused-adjacent-pass.log` passes the adjacency regression; exit 0.
3. `focused-search-navigation.log` records a real expanded-test failure caused
   by the test expecting `"needle"` in `"alpha needle"` at `[2,8)` instead of
   the correct UTF-16 `[6,12)`; exit 1.
4. `focused-search-navigation-pass.log` and
   `focused-search-navigation-constraints-pass.log` pass after correcting that
   test expectation; exit 0.
5. `focused-search-navigation-final.log` is an assertion/setup failure distinct
   from the original product red. The new active-tab-during-scan test assumed
   the newly created second buffer's caret was zero, but the buffer already
   placed it at UTF-16 offset 6 after insertion. Production behavior was not
   failing this assertion. The test now captures and compares the existing
   selection, which is the property being protected; exit 1.
6. `focused-search-navigation-final-pass.log` passes the complete focused set
   after that test-only correction; exit 0.

The final focused command was:

```text
.\native\scripts\go.ps1 -GoArguments @("test","./app","-run","^(TestFind|TestReplaceCurrentInvalidatesNavigationCacheAndFindsFreshText|TestShutdownClearsFindCache)","-count=1")
ok  	levnote.local/native/app	2.035s
EXIT_CODE=0
```

The focused set includes real `Reload()` and `reconfigureTabs()` cache drops,
next-frame query/version expiration, identity-changing `Flush`, source-changing
`Flush`, failed warm Flush, one-result wrapping, concurrent active-tab change,
readonly/source reveal and replacement freshness.

## Final affected checks

Exactly one affected suite was run after the substantive correction:

```text
.\native\scripts\go.ps1 -GoArguments @("test","./app","./core/search","-count=1")
ok  	levnote.local/native/app	16.286s
ok  	levnote.local/native/core/search	0.716s
EXIT_CODE=0
```

The scoped vet then ran once:

```text
.\native\scripts\go.ps1 -GoArguments @("vet","./app","./core/search")
EXIT_CODE=0
```

Exact logs are `final-affected-suite.log` and `final-scoped-vet.log`. No whole
release suite or build was run.

## Bounded allocation evidence

`warm-cold-benchmark.log` preserves the raw 20x result for
`BenchmarkSearchNavigationCacheWarmCold`, using a 57,344-byte source with 4,096
matching results:

```text
warm-16  30.00 ns/op      0 B/op      0 allocs/op
cold-16  2,468,115 ns/op  1,063,316 B/op  4,137 allocs/op
```

This isolates cache lookup versus the cold search-function path. It does not
claim an application speedup, percentage improvement or lower process working
set. Cold search still scans the whole document. A successful maximum-size
cache stores 200,000 pairs of amd64 `int` values, about 3.2 MB for the offset
array plus slice/object overhead; refusal stores no partial offsets.

## Frozen source and test hashes

| File | Bytes | SHA-256 |
|---|---:|---|
| `native/app/search_navigation.go` | 4,992 | `8F77367B137135618CEB65BE0513ED5670033E2500A867C87DD8185CB5854E6D` |
| `native/app/search_navigation_test.go` | 20,784 | `CEF83EA00713A9896C6FCD117BB7A710BFD184FB788235119121F9C99CF2F79D` |
| `native/app/app.go` | 32,768 | `5017BCC975F396EFD966D0D76159BC8BF0114671ED06B5E3657C95A184C06984` |
| `native/app/plugins.go` | 10,756 | `E64F408A4EFFBA065923B73136A6515BCAFA295468199EF0A9A625E3325E87C6` |
| `native/app/state.go` | 7,461 | `8E8181C0FC8CB581767784C914B81B09B1A1414912CA9E9376C41DE31D2431CF` |
| `native/app/view.go` | 20,547 | `5501670B56CD876D9E1252B03E31F434DC9DC9E65B9A6C13212AC5D4E2404125` |
| `native/app/workspace.go` | 6,594 | `F7898A151F2850FA34C78669E2265C2DE8951BC77C805601F93ED20B306D5A2A` |

Source and tests are frozen at these hashes after the passing focused,
affected-suite and vet runs.

## Limits

- Navigation proves only the active tab/document/editor/buffer at the final
  version. It is intentionally not a per-tab search cache.
- A manual caret move away from and back to the same coordinates is not
  observable; the recorded selection then behaves as the same progression
  position.
- Version/query changes are permitted to remain lazy until the next frame or
  next find action, but can never authorize a stale reveal.
- The benchmark covers this app function only. It does not establish physical
  UI behavior, process RSS, CPU reduction or release-package readiness.

## Frozen-ready

The implementation, tests and evidence are stable and ready for the one
read-only reviewer pass. The reviewer is
`/root/read_native_handoffs_resource_workload_next_task_md`; the expected
handoff is `native/handoffs/search-navigation-independent-review.md`.
