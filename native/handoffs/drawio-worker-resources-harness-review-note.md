# Early fixture/oracle review

2026-11-10 root read the first saved draft; apply before final compile/checkpoint.
No worker runs are activated. These are concrete draft issues, not new scope.

- Both selected pages currently add the single `two` vertex AND the ordinary
  other nodes/edges. They cannot emit the literal598-byte single-rectangle SVG.
  Keep only that small vertex on the selected page; put measured source/model
  load on other pages, or independently specify a larger exact SVG. Use the
  first option for a small reliable harness and report actual counts.
- MetadataCell uses fmt `%q` with integer `enabled`, producing a quoted rune
  instead of XML `vertex="1"` / `edge="1"`. Use explicit valid attribute text.
  Metadata selected-page endpoints also refer to p3-n7 after decrementing count;
  keep nonselected graphs internally valid and selected page simple.
- Actual wire code is uppercase `UNSUPPORTED_ELEMENT`; actual generic message
  must match parse.go, not a paraphrase. Inspect both cmd worker and parse.go.
- Root's earlier all-pages warning note was overly broad: parse produces all
  page issues, but cmd/levnote-drawio-svg/main.go wireParseWarnings filters to
  selected page plus DocumentPage. Correct the plan/report to actual protocol.
  Predeclare exact selected/document warnings, preserving any chosen unknown
  selected-page content. Unknown nonselected content is not emitted in the
  selected-page wire warnings. No change to production behavior.
- Ensure frozen fixture checks assert size<1MiB, actual expected page/node/edge
  counts, selected index/ID and literal expected hash lengths. Computing and
  recording values alone can bless a malformed fixture as valid.

The preparation must include the future opt-in NewManaged/Export/Close sample
path reusing resourceProbe, not only fixture preflight. It may compile/default
SKIP but must not execute that request path before root review/activation.
No need for another independent preflight framework or many helper files.

Root checkpoint05:25: a measurement test containing only unconditional t.Fatal
is NOT the requested implementation. Preparation forbids EXECUTION, not code.
Implement the real future opt-in sample path now (NewPackageManager/import/
activate/NewManaged/Export/Close with existing resourceProbe and guards), then
compile once for the final checkpoint and run only its default SKIP. Do not
continue recompiling a stub. Root activates actual requests after review.
The changed fixtures imply exact expected counts: Normal8pages/148nodes/42edges,
Metadata6pages/41nodes/60edges if the current constructors remain. Assert those
and selected page identity in the XML-only preflight, not just counts>0.

Root05:30 review of the new actual sample path; fix before another compile/run:
- Grouped struct fields share the same json tag (Status,CreatedAtUTC etc.).
  encoding/json drops conflicting tagged fields. Give each field its own name
  and unique tag, and assert a sentinel marshaled record includes every needed
  actual/error/time/version/ordinal field in no-process validation.
- NewManaged activates an INSTALLED package. expectedImage must use
  manager.installedPath(enginecatalog.DrawioSVG,receipt.Digest)+filename,
  not the original bin bundle. Otherwise every real sample fails by design.
- Add one-request-per-tester atomic guard and actual operation counts.
- Record actual returned SVG size/hash and ordered warnings, not only expected
  values/boolean comparison; source after hash and imported receipt identity.
- Probe must finish on every path after acquisition, even early failure. Reuse
  cleanup carefully (Close then probe finish before record is serialized).
  Validate creation/final times and failed memory reads without substituting
  zeros. Flag incomplete final peak coverage explicitly as existing probe does.
- Pin/check fixture source hashes from the final preflight in the eventual
  execution wrapper; do not silently bless regenerated different fixtures.

Pause repeated go test/default-SKIP/preflight invocations while fixing source.
There have already been multiple snapshots; final validation should use one
explicit test -c output, that exact tester default SKIP, then one no-process
preflight. Publish source for root review BEFORE that final compile if needed.

Root continuation review of saved measurement source2026-11-10:
- MatrixPosition is still never assigned; admission/verify/start/wait counters
  are checked locally but not serialized. Record their actual values and derive
  the unique1..12 position from the frozen fixture/version/ordinal order.
- The JSON sentinel is currently called only INSIDE the disabled measurement
  test. Call it in the no-process preflight too so the promised preparation
  actually exercises it. Keep an empty actual warnings array as [] (not null).
- Require a preflight-pinned expected source digest before any import/launch;
  the future wrapper must pass the accepted preflight fixture digest and exact
  tester digest. Do not silently record and bless regenerated different input.
- Verify initial/final creation identities match, and exit is at or after
  creation. LifetimeNS defaults to0 on bad ordering in the shared probe, so
  checking LifetimeNS>=0 alone is insufficient.
- Do not make zero/reduced final Windows peak counters alone a product failure.
  The existing probe explicitly marks incomplete coverage after exit; retain
  that raw limitation and report it, never replace unavailable peaks with0 or
  claim complete lifetime peaks. Actual failed WinAPI reads remain visible
  failures. FinalMemoryReadOK/FinalCPUReadOK and exact cleanup are required, but
  LifetimePeakCountersComplete=false can be a valid limited observation. This
  clarifies the earlier note's 'flag incomplete coverage' wording.

Apply these small concrete corrections before the final compile/preflight.
No new helper framework or product changes; no worker measurement activated.

Follow-through check of saved corrections: reading probe.InitialTimes.Times or
FinalTimes.Times BEFORE a nil guard can panic on the exact probe failure that
must produce evidence. Use nil-safe conditional checks. Adding incomplete
coverage to record.Failures still forces FAIL in cleanup; record a separate
limitation or retain the existing probe.PeakCoverage, not a failure. Finally,
append([]Warning(nil), empty...) still writes null, so initialize an empty
slice for the zero-warning output. Publish corrected SOURCE checkpoint first;
preserve previous tester/evidence and await root before any new compilation.

MatrixPosition must reflect the frozen ALTERNATING order, not .20 then .21 in
every pair. A ordinal2 reverses; B ordinal1 and3 reverse. Assert all12 explicit
fixture/version/ordinal positions in no-process preflight, preserving the
declared order. This is the last source-read correction before checkpoint.
