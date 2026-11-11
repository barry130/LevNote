# Independent review: draw.io native route

FROZEN INDEPENDENT REVIEW 2026-11-10. Read-only source/evidence review only:
no product edit, test/build/worker/host launch, package run or physical UI/IME
claim was made. `.19` remains immutable. This review accepts source only as a
review input; it does not accept a `.20` binary or package.

## Result

No demonstrated production blocker was found in the closed route admission,
fresh adapter resolution, app fallback, current-page request, or publication
gate. The implementation is narrow and its capability claims match the existing
draw.io adapter.

The frozen evidence is not yet sufficient for two claims in the implementation
result. Both are test-improvement items; neither requires a production change.

## Required evidence corrections

### 1. Same-ID test can pass after route loss

`native/app/drawio_route_test.go:352-360` saves `oldAdapter`, edits the
manifest name, refreshes, then requires only `call.ctx.Err() != nil` and
`tab.Adapter != oldAdapter`. A failed rematch that leaves `tab.Adapter == nil`
also satisfies that expression. `refreshPlugins` reconfigures tabs even when
refresh reports a load error, and every successful reconfiguration creates a
fresh adapter pointer, so the test does not prove that the changed same-ID
manifest was loaded and rematched.

Before the final freeze, call `requireDrawioRoute(t, tab)` after
`a.refreshPlugins()` and assert that the updated identity is visible, for
example `tab.MatchedPlugin.Name == "图纸笔记 已刷新"` (or another field changed
by that exact manifest revision). The refreshed success message can be checked
as an additional direct observation. This makes the test prove an actual
same-ID replacement rather than a generic presentation reset.

### 2. The “late result” is not delivered by the fake runner

`native/app/drawio_route_test.go:337-368` uses the default
`controlledExporter`. Its `Export` method returns immediately on `ctx.Done`
when `ignoreCancellation` is false
(`native/app/export_worker_test.go:46-63`). After `a.refreshPlugins()` cancels
the job, sending `drawioWorkerOutcome()` into the buffered response channel
does not make the runner return that successful outcome. Consequently, the
`job.result.Payload == nil` assertion observes the ordinary context-canceled
error path, not handling of a worker result that arrives after invalidation.

Set `f.ignoreCancellation = true` before starting this route export, send
`drawioWorkerOutcome()` after the refresh, and assert after completion that:

- the runner actually consumed the successful response;
- `job.result.Payload` and warnings were cleared;
- no save dialog opened and no output file exists;
- `a.exportJob` was released.

The production gate at `native/app/export_worker.go:60-80` and
`native/app/export_worker.go:260-345` appears to handle that case correctly by
clearing invalid results and rechecking identity before publication. The
missing piece is proof through the draw.io same-ID route path.

## Source review

- `native/core/plugins/native_route.go:16-20` adds the exact
  `builtin.drawio` identity. Lines 27-127 retain the schema3 strict field,
  duplicate, null, UTF-8/surrogate, required-field and trailing-data checks.
- `native/core/plugins/declarative.go:333-347` adds draw.io to the closed route
  set without adding outline rules or executable policy. Existing
  normalize/match/install order remains in the registry.
- `native/modules/modules.go:22-35` resolves every route through the existing
  adapter factories and returns a fresh value. The draw.io factory retains the
  existing template, `svg` export descriptor,
  `levnote.drawio-svg`/`RequiresView:true`, and no Outline or Scene capability.
- `native/app/plugins.go:83-103` obtains the route adapter independently of
  builtin enablement and visibly falls back to source when the compiled adapter
  is unavailable. Lines 149-158 also require the returned adapter ID to exist
  in the compiled catalog and to have a constructor.
- `native/app/plugins.go:167-195` gives schema3 truthful, generic native
  capability wording instead of promising outline support, and the install
  dialog at lines 267-280 lists draw.io.

The new app coverage otherwise exercises the intended paths through real app,
Tester and buffer state: new/open modes, BOM/CRLF/unknown metadata preservation,
mapped-suffix SaveAs and readonly copy, second-page field `Compose` plus Flush
before request preparation, source/nil-view refusal, conflicting-draft retention,
builtin/installed enable independence, and explicit missing-adapter fallback.
The inherited graphview test
`native/modules/graphview/safety_test.go:213-245` continues to cover draw.io
page-ID tracking, reorder safety, unrelated-field merge, changed-field refusal
and pending-draft retention. No duplicate worker execution was requested.

## Identities and evidence

All eight frozen source/test identities match the implementation handoff:

| File | Bytes | SHA-256 |
|---|---:|---|
| `native/core/plugins/native_route.go` | 3,915 | `24F7108DB4423AA3C8EB9C50BD0123844AF6088216837DD21CF6323C41B41DB6` |
| `native/core/plugins/declarative.go` | 18,002 | `405AB8406B655C9E3DF8C514422B745D2322EB432F0C28175304CDC8CEF614F8` |
| `native/core/plugins/schema3_test.go` | 19,413 | `C1D451F459E8710DE94A68997A04C7172BC45ACFDCAA724D37ABD9D73694D028` |
| `native/modules/modules.go` | 1,997 | `E8438F294647E0C73522C48141B05DBA0D5DA8BEECCD8B2663C2520FE3CFFCFC` |
| `native/modules/native_route_test.go` | 6,578 | `2A6C3078393BDED0DE90BAB79F7E41278FCD53BF27CAC244C90681175D316BC8` |
| `native/app/plugins.go` | 10,836 | `91D717983A1CAA462D5977572C83BEF0F99A24FF59E275E5C81EA3BDC777D476` |
| `native/app/declarative_outline_test.go` | 4,431 | `5E6D3662D26A7F149FA23EAF633D21C7F9120601CDE84DE175557AF0039AD2E7` |
| `native/app/drawio_route_test.go` | 13,672 | `D81CA768DD4A8A3FCFC019978ED89619FD06DB9D5998894CA3FD327DA6274F5F` |

The preserved focused red log is
`AAB42CA24F041025E6E319677E1E0B358DAD34845A88402EDBCE9E8F196CAE4E`.
Its two failures are consistent with the disclosed test-oracle corrections:
the draw.io template's existing structural-container warnings and the
enablement fixture not yet disabling the builtin. Production source timestamps
predate that red run; the tests were updated afterward, followed by focused
green at `61FABCF23066ADDCF1E480B1842225DECD1DAB500C96BB080751B9297B41DFE9`
and the final affected suite at
`4EFE05254D9347EF4B9AD81348F12C8EAB600B67CCD6BF2C0CF0A432CE7C2827`.

The final suite records exits `0` for `core/plugins` 2.171s, `modules` 0.877s,
and `app` 16.709s. Vet exit is `0`; its zero-byte stream is
`E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855`.
The harness-path failure is preserved separately and produced no test result.
No Tester capture exists for this route task; none was required.

## Scope limits

This review did not execute tests and does not establish physical Windows
input/IME behavior, worker process isolation, resource use, package contents,
or custom-suffix behavior in the immutable `.19` binary. The examples and
portable wording reviewed here were authored by this owner and remain for
separate root review; their structural validation is not product manifest
admission or runtime proof.

## Reconciliation resolution - 2026-11-10

**CLOSED.** Read-only reconciliation of
`drawio-route-test-correction-result.md`, its raw evidence, and the current
`native/app/drawio_route_test.go` confirms that both original test/evidence
findings are resolved. Neither required a production change. The findings above
remain preserved as the initial review record; this dated section supersedes
their open status.

1. The refreshed-route finding is closed. After `a.refreshPlugins()`,
   `drawio_route_test.go:362` calls the full `requireDrawioRoute` contract and
   lines 363-365 require the updated manifest name
   `图纸笔记 已刷新`. A nil adapter, failed rematch, missing capability, or
   stale old manifest can no longer satisfy the test through the previous
   pointer-inequality check alone. The pointer check remains only as supplemental
   evidence that the presentation was replaced.
2. The late-result finding is closed. The target test sets
   `f.ignoreCancellation = true` on its own fake before calling
   `a.exportCurrent` (`drawio_route_test.go:337-339`); the flag is not inherited
   from the preceding test. After refresh invalidation, the test sends a
   successful draw.io outcome and then waits for `job.done`
   (`drawio_route_test.go:366-367`). With cancellation ignored, that completion
   cannot occur until the runner consumes this result.

   The test then drains the queued UI completion and synchronously completes
   lane cleanup. It verifies an empty response channel, `job.ready`, cleared
   payload/warnings/error, released `a.exportJob`, released lane reservation,
   no export dialog, and no output file (`drawio_route_test.go:368-383`). This
   exercises the late-success path rather than the ordinary canceled-error
   shortcut and proves the existing production publication gate refuses it.

The preserved first focused attempt is genuinely red:
`focused.log` records `responseDrained=false` with exit `1`. That failure
matches the disclosed harness correction: the first patch placed
`ignoreCancellation` in the preceding test, so the target runner had already
returned on cancellation. The corrected focused run exits `0` in 1.508s. Final
app tests exit `0` in 17.193s and app vet exits `0` with empty output. The exact
raw command, output, exit and hash records match the correction handoff.

The changed test is 14,177 bytes with SHA-256
`719AB02BE3316FD09334317CF096F2D167048C1B85A89EC89EB1362B50720C86`.
The other seven implementation-file byte counts and SHA-256 values exactly
match the full identities in the initial review table above:
`native_route.go`, `declarative.go`, `schema3_test.go`, `modules.go`,
`native_route_test.go`, `plugins.go`, and `declarative_outline_test.go`.
Thus this correction changed only the reviewed app test, not route production
or the other implementation tests.

Reconciliation is complete and frozen for root acceptance. No tests, builds,
launches, workers, captures, measurements or product edits were performed by
this reviewer. This resolution does not accept a `.20` binary or package.
