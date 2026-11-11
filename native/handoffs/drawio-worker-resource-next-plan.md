# Draw.io worker resource comparison

Status: read-only future plan. No implementation?\par

**No implementation. No execution.** Root activates this stage only after the
immutable `.21` package/evidence and host checks are accepted; `.20` stays unchanged.
Reuse `native/core/engineruntime/worker_resources_test.go` (`resourceProbe`,
`PROCESS_MEMORY_COUNTERS_EX`, `K32GetProcessMemoryInfo`, `GetProcessTimes`),
`run.go` hooks, `process_windows.go`'s `WithHandle` duplicate before request,
Job assignment/cleanup, image/creation identity unseen exit and
`drawio_managed_bundle_test.go`'s `NewManaged`/`Export`/`Close` path. Do not
repeat source tests, package builds, or Markdown measurements.

Before launching, verify the accepted `.20` and `.21` package+manifest digests
and exact image paths; import each copied bundle and activate its receipt digest
in a disposable config root. Keep verification separate from measurement. A
fixed opt-in tester must use `NewManaged`, one fresh tester process per request,
the exact owned process object, a duplicated handle for `PeakWorkingSetSize` and
`PeakPagefileUsage`, identity/creation/exit proof, correct page selected, and
joined Runner `Close`; any hook/probe failure is a failed sample, never a zero.

Two deterministic, moderately sized Windows fixtures (both <1 MiB and well
inside the existing 32 MiB source/250k-element/64-attribute caps), same selected
page per fixture:

1. **NormalMultiPage**: 8 pages/170 vertices, 48 edges, unknown XML/comments;
   page 3 selected (recent editor/zoom/viewState/image-ish metadata). Exercises
   ordinary page parsing and retained XML.
2. **MetadataHeavy**: 6 pages/48 vertices, 72 edges, repeated unknown custom
   metadata up to 32 attributes/element, page 4 selected p4-v7. Forces parser/
   source/model retention rather than SVG output amplification. These are
   telemetry fixtures, not product performance thresholds.

For each fixture require exact source hash, exact SVG bytes/hash, ordered warning
shape/count, unchanged input, zero exit, and identical expected SVG for `.20`
memories. The `.21` warning identity may differ only by the already accepted two
canonical template-pair warnings; any other output or cleanup difference fail? Yes.

Cold order, no warmup, at most six pairs (12 requests): A `.20,.21`, A `.21,.20`,
A `.20,.21`, B `.21,.20`, B `.20,.21`, B `.21,.20`. Fresh tester/config copy per
request. Keep first samples/order/failures; never rerun or average away an error.
Report raw per-pair and medians only descriptively (n=3/fixture/version), never a
confidence, winner, or speed claim.

Measured: worker-only OS `PeakWorkingSetSize` and `PeakPagefileUsage`, sampled
current working set/private commit and kernel+user time; request elapsed includes
launch/transport/serialization/cleanup. Not measured: host/UI/Tauri/editor RSS,
whole-process tree, retained heap, cumulative renderer allocation, GPU, physical
UI/IME. This may show whether the narrower source-ownership slice visibly affects
these raw worker peaks; it does not prove the host editor saves RAM.
