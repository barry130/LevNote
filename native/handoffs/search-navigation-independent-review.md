# Search navigation independent review

2026-11-09. Final read-only review of the owner's frozen
`search-navigation-result.md` checkpoint. One source/evidence pass only. The
reviewer ran no tests, benchmarks, vet, build, launch or source/doc/evidence
edit.

## Result

**No material defect found in the bounded search-navigation/cache scope.**
No correction or owner action is requested.

The frozen source, tests, result and seven affected app files match the
recorded checkpoint. Recomputed SHA-256 values:

| File | SHA-256 |
|---|---|
| `native/app/search_navigation.go` | `8F77367B137135618CEB65BE0513ED5670033E2500A867C87DD8185CB5854E6D` |
| `native/app/search_navigation_test.go` | `CEF83EA00713A9896C6FCD117BB7A710BFD184FB788235119121F9C99CF2F79D` |
| `native/handoffs/search-navigation-result.md` | `66022881A7E35982B58D6D2447FAB3D32C5E50DE3CE181EFB1783A964EC32D79` |
| `native/app/app.go` | `5017BCC975F396EFD966D0D76159BC8BF0114671ED06B5E3657C95A184C06984` |
| `native/app/plugins.go` | `E64F408A4EFFBA065923B73136A6515BCAFA295468199EF0A9A625E3325E87C6` |
| `native/app/state.go` | `8E8181C0FC8CB581767784C914B81B09B1A1414912CA9E9376C41DE31D2431CF` |
| `native/app/view.go` | `5501670B56CD876D9E1252B03E31F434DC9DC9E65B9A6C13212AC5D4E2404125` |
| `native/app/workspace.go` | `F7898A151F2850FA34C78669E2265C2DE8951BC77C805601F93ED20B306D5A2A` |

## Material behavior checked

- Selection progression matches the task. A nonempty exact match advances one
  result in either selection direction; otherwise next uses `Start >= End` and
  previous uses `End <= Start`, both on ordered selection bounds, with wrap.
  A recorded unchanged selection advances zero-width results. See
  `native/app/search_navigation.go:102-138`.
- Every find flushes first and only then checks the exact Tab, Document, Editor
  and Buffer identities. The cold result is accepted only if the Buffer version
  did not change during the scan, and identity is checked again before reveal.
  Thus a flush cannot be bypassed and an abandoned synchronous result cannot act
  on the current tab. See `native/app/search_navigation.go:65-85` and
  `native/app/search_navigation.go:140-193`.
- Empty queries, search errors and no-result scans clear `FindIndex`; a
  too-many-matches refusal retains no partial offsets. Query/options/version or
  identity mismatch drops the cache on the next action/frame. See
  `native/app/search_navigation.go:34-50` and
  `native/app/search_navigation.go:155-181`.
- The retained result shape is only copied `{Start,End int}` offsets plus
  small identity/version/options/error/cursor state. It does not retain
  `search.Match`, a source string, `text.Snapshot`, submatch data, or a map of
  tabs. See `native/app/search_navigation.go:13-28` and
  `native/app/search_navigation.go:87-99`.
- Committed switch/add/remove, Find close, encoding reconfiguration, plugin
  rematch, successful reload, replacement and shutdown invalidate the shared
  cache. Failed switch/close returns before the committed invalidation and
  preserves the active draft and valid cache. See
  `native/app/app.go:344-365`, `native/app/app.go:857-893`,
  `native/app/app.go:931-956`, `native/app/app.go:1011-1029`,
  `native/app/plugins.go:43-48`, `native/app/state.go:221-229` and
  `native/app/workspace.go:56-62`.
- Replacement remains independent of navigation offsets: each edit path uses a
  fresh `search.Find`/`search.ReplaceAll`; the navigation cache is dropped only
  after an actual change, then Find recomputes. See `native/app/state.go:184-235`.
- The frozen tests cover adjacent `aa` matches, exact and reversed bounds,
  nearest match, wrap, one/no result, zero-width and UTF-16 offsets, cache
  reuse, options/version/edit/undo/restore/new-buffer changes, too-many refusal,
  rich Flush mutation, identity-changing Flush, failed Flush, lifecycle
  invalidation, reload/rematch, readonly reveal and replacement freshness. See
  `native/app/search_navigation_test.go:43-301`,
  `native/app/search_navigation_test.go:334-484` and
  `native/app/search_navigation_test.go:486-604`.

The owner's raw `final-affected-suite.log`, `final-scoped-vet.log` and
`warm-cold-benchmark.log` are consistent with the retained result: app 16.286s
and core/search 0.716s exit 0; vet exit 0 with no output; warm 30.00 ns/op,
0 B/op, 0 allocs/op and cold 2,468,115 ns/op, 1,063,316 B/op, 4,137 allocs/op.
These are owner-run evidence, not reviewer reruns.

## Limits

The active-tab-change-during-scan test uses a deterministic injected synchronous
seam to establish that an abandoned result is not applied. It does not establish
general concurrent-app safety.

The benchmark compares this app's warm cache lookup with its cold search path.
It does not establish physical UI behavior, app speedup, CPU reduction, process
RSS or release readiness.

The separately noted fixed-width Find bar row
(`native/app/state.go:239-247`) is a layout concern outside this cache and
navigation source-review scope and was not assessed here.
