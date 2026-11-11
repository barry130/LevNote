# Stage2 independent review

FROZEN-SOURCE REVIEW 2026-11-10 for
`/root/read_native_handoffs_drawio_worker_stage2_independent_task_md`.
This is a read-only review of the frozen Stage2 production source, tests,
logs, and both 760x480 captures. I did not edit product files or run tests,
builds, launches, or a repro.

## Conclusion

No source blocker. Stage2’s current-page draw.io worker route satisfies the
required identity, preparation, publication, and stale-result gates. The app
route uses the advertised worker exclusively and has no synchronous draw.io
fallback.

One non-source issue blocks clean closure of the active review process: the
test-followup owner has not published the required
`drawio-worker-stage2-test-followup-result.md`. Its evidence directory contains
a passing final app test and vet, and an earlier failed focused attempt is
retained. The followup evidence itself is coherent, but the absent final handoff
leaves its case truth and run scope unreconciled.

## Findings

1. **Identity and admission are sound.** `startEngineExport` captures the
   active tab, document, buffer, editor, adapter, capability, name, mode, and
   view before Flush. It rechecks those owners after Flush, rejects group and
   oversize source before preparation, saves the post-Flush buffer version,
   and rechecks the same owners, version, group, mode, view, capability, and
   token after preparation. Page metadata is copied into
   `engineipc.RequestPage` before async enqueue.
2. **Preparation semantics are correct, but the original result wording is
   wrong.** `PrepareExportRequest` synchronizes a stale cache and may cancel a
   stale gesture; it is not side-effect-free. It does not Flush, publish, or
   discard a pending draft. Only `ExportRequestToken` is side-effect-free.
   This is the wording correction requested by the test-followup task, not a
   production defect.
3. **Final publication gates are sound.** `validExportIdentity` requires the
   exact document, buffer, editor, adapter, capability, name, version, mode,
   view, and export token for view-required exports. `recheckExport` performs
   one Flush and then repeats the pure identity checks. The pre-dialog and
   post-dialog paths both use that gate. Stale jobs clear their payload and
   warnings and do not write a file.
4. **View invalidation is sound.** The graphview token advances on actual page
   transitions (including B-A-B and A-B-A), real parsed-cache invalidation,
   and Release. Cache hits, unchanged Draw/Warnings, and same-index assignment
   do not advance it. A released view pointer can prepare again under a newer
   token, and a retained source view cannot authorize export because mode is
   checked independently from view nilness.
5. **Old engines remain independent of a mounted view.** Markdown and mindmap
   requests carry `Page=nil`, and source-mode exports work without a view. The
   stage tests also assert that these exports preserve buffer, history,
   selection, dirty state, and original disk/BOM state.
6. **Lifecycle and package semantics are retained.** Existing lane
   reservation, exclusive publication, cancellation, Close/shutdown joining,
   modal deferral, poison blocking, and import-without-activation behavior are
   preserved. The fourth fixed selector, count-neutral reset wording, and
   visible source-mode instruction match both captures.

## Test and evidence reconciliation

The frozen Stage2 test identity drift is authorized and expected: only
`app/export_drawio_worker_test.go` changed, because the followup corrected the
inaccurate B-A-B case into real B-A-B navigation and added a real pending
second-page field through Flush, Prepare, snapshot, publication, and one
intended undo transaction. All other frozen source/test hashes match the
original result exactly.

The original final affected suite, final vet, and both capture hashes match the
frozen result. Both captures are exactly 760x480, readable, in bounds, and
match the claimed source-mode and fourth-engine panel behavior. They are
synthetic MyGo Tester captures and do not establish physical input/IME, latency,
or whole-process resource behavior.

The followup’s current evidence shows a first focused app attempt failing to
compile because the test used the nonexistent `text.History` type, followed by a
passing focused attempt, passing gofmt, final app suite exit 0, and final vet
exit 0. The required final followup markdown is absent, so I cannot reconcile
its final test hash and exact claims from owner-written case truth; the source
and preserved evidence support acceptance after that handoff is published.

## Review limits

This is source/evidence review only. It does not authorize or perform the
`.19` package build, actual managed worker request, host launch, portable asset
delivery, or physical-UI/resource testing. The exact SVG renderer oracle remains
lower-layer/real-worker work and must not be conflated with the synthetic
runner forwarding assertions reviewed here.

## Final reconciliation and closure

CLOSURE 2026-11-10. The test owner has now published
`drawio-worker-stage2-test-followup-result.md`. Its stated source identity
(`app/export_drawio_worker_test.go`, 14,855 bytes, SHA-256
`F8FF9ADBF52F1ED68161306FB4D472A91C500FC59B3A90BA24278C579F4C2BD2`), unchanged
original result identity (5,966 bytes, SHA-256
`19DD58AB7589CC1DBDFE7A6B454A1C28F76968D24C2A4D74A30CF3BDC9BEA283`), and final
test/vet hashes match the evidence reviewed above. The record also accurately
describes the failed first compile attempt, duplicate failure preservation,
passing focused retry, passing final app suite and vet, real B-A-B navigation,
real second-page field Flush, one intended undo transaction, stale-result
clearing, and inherited conflicting-draft coverage.

The earlier absent-record finding remains historically accurate for its review
time. The non-source handoff issue is now resolved. No remaining review blocker
exists. This independent review is closed and frozen; Stage2 source acceptance
does not authorize or perform `.19` delivery.
