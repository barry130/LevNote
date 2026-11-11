# Resource follow-up — existing lifecycle coverage inventory

2026-11-08 read-only inventory during final link integration, updated after
root accepted the16:45 release and16:48 smoke. No new test run, test code,
product change or process sample in this inventory. It supports the now-active
resource_followup_stage_plan.txt. The final integration suite passed8.998s
with all packages, and full vet passed2.452s. Do not launch a redundant suite
during process CPU sampling.

## Existing meaningful assertions

| Area / file | Existing tests | Verified scope in the test source |
|---|---|---|
|Markdown view_test.go|TestReadOnlyAndSameVersionReleaseAreLossless|Readonly/writable parse released; independent clean field removed; scroll retained; Warnings/Frame/HTML rebuild same Buffer version; source bytes and Undo remain unchanged|
|Markdown view_test.go|TestNativeConflictRecoveryReleaseAndBlockedActions|Failed native field Flush preserves independent recovery through Release; navigation/export barriers don't discard it|
|Markdown links_test.go|TestLinkSynchronousReleaseKeepsActiveListBuild|Actual native link callback synchronously Release; old ListState pointer remains usable in active build, later frame recreates view and second link works; settled|
|Markdown outline_test.go|TestOutlineIdentityMapsDuplicateInsertionsAndUTF16|Same old source ID resolves after independent preview Release and new View creation; duplicate insertion and Undo keep correct identity|
|Markdown preview_limit_test.go|TestOversizeMarkdownRefusalBeforeCopyAndCacheRecovery; TestOversizeMarkdownDropsStaleRowsBeforeBegin|No whole-source copy on refusal; repeated refusals cached; old raw/rows/outline descriptors dropped; new same-version Buffer and shrinking recover; Undo restores authority|
|Markdown preview_limit_test.go|TestOversizeMarkdownCompositionSafeRangeFlushAndConflictRecovery; TestOversizeMarkdownShrinksWhileIndependentCompositionStaysOpen|Independent Compose/draft survives refusal; safe range-only Flush; conflict recovery; new bounded parse does not reattach stale block index|
|Graph safety_test.go|TestReleaseRebuildsSameVersionAndRetainsViewState|Both mindmap/drawio release parse/layout/draft; same-version Warnings/Export/Frame rebuild; selected node/page/zoom/pan and source/version unchanged|
|Graph safety_test.go|TestGroupConflictRetainsNativeDraftAcrossRefreshSelectionUndoAndRelease|Foreign group conflict retains native draft across attempted actions and Release; no stolen transaction|
|Graph resources_test.go|TestLabelCacheBoundsAndInvalidation|No reshaping on pan/theme; correct invalidation by zoom/font/width/source; bounded labels; Release drops label layouts/node index/page index/edge scratch while retaining selection/zoom/pan|
|Graph resources_test.go|TestDenseVisibleLabelsDoNotThrashBoundedCache|500 visible overlapping labels settle under cap, selecting previously omitted last node doesn't cause continual cache churn|
|Graph resources_test.go|TestPageAndSameVersionBufferReplacementDropCachedGeometry|Page switch and same-version replacement Buffer invalidate old parse/index/text|
|Graph preview_limit_test.go|TestOversizeGraphRefusesBeforeCopyAndRecovers; TestOversizeGraphClearsValidCacheAndBlocksMutationWithoutCopy; TestOversizeGraphCompositionDraftSurvivesFlushReleaseAndRecovery|Both parsers refuse before whole copy; old drawable removed; failed Compose+Flush/Export/Release preserve independent field/source/history and visible recovery; Undo/shrink rebuild|
|Source outline outline_test.go|TestSourceOutlineObservationalUnicodeAndDuplicateRangeIdentity; TestSourceOutlineBoundedLongLineCacheAndCopiedDescriptors|Release removes entries/spans/read chunks/cached Buffer; same old ID maps after Release/new controller; repeated bounded snapshot doesn't reread chunks; caller slice doesn't alias cache|
|Text editor editor_test.go / geometry_test.go|TestReleaseViewRetainsUndo; TestReleaseCompositionAndStyleDoNotLoseHistoryOrStealFocus|Release commits pending native composition and frees geometry/sparse rows/focus; Undo retained; inactive style change cannot steal another control's focus|
|Text editor geometry_test.go|TestLayoutIsBoundedAndIdleFramesDoNotReshape|100,000-line source only shapes viewport;10 unchanged frames don't reshape/request another frame; far jump bounds sparse rows/cache|
|App app_test.go|TestOpeningAnotherTabReleasesInactivePresentation; TestModeSwitchReleasesCachesButRetainsViewState; TestAdapterReconfigureDisposesRetainedView|Shell lifecycle contract (stub views): retain light View pointer across tab/mode, Release heavy caches, discard on reconfigure|
|App outline_builtin_test.go|TestBuiltinOutlineReleasedOnPluginModeTabAndReload|Real Markdown provider factory recreated across source mode/tab, cleared/recreated by plugin disable-enable and reload|
|App outline_test.go|TestOutlineSnapshotFailureDoesNotRetryEveryFrameAndReleaseInvalidates|Stub provider failure cached, explicit retry releases/rebuilds once; later mode invalidation clears source|
|App oversize_preview_app_test.go|TestOversizeRichViewsSupportSourceToRichModeLifecycle; TestOversizeGraphDraftFailedModeSwitchRetainsThenRecovers|Actual registered rich adapters, source/preview switch refusal/recovery and failed draft retention|
|App markdown_links_app_test.go|TestMarkdownLinkRealAdapterLocalPointerKeyboardSaveReopen|Actual registered local-file route, no-op bytes/history, source-mode no inline controls, preview restoration and synchronous source release; native pointer+keyboard|

## Evidence scope and remaining gap

The owner ran the complete modules/markdownview package after the final link
resource fix: PASS2.050s and vet clean, including all Markdown rows above.
The final DeepSeek integration run now covers the current artifact's entire
suite: PASS8.998s (app5.454s; markdownview2.586s), full vet clean. Exact evidence:
artifacts/acceptance/markdown-links-20261108-164403/result.json and live logs.
Older accepted runs are historical, not newly rerun here. App mode tests that use stubs are labeled as such; they do
not prove actual GPU paint or per-format allocations.

No missing correctness lifecycle assertion currently justifies a new redundant
test. The uncovered resource question is the main process's working set/private
bytes/CPU under current validated medium fixtures and a hidden-window launch.
These code-level drops do not establish allocator/OS memory returning or zero
RSS after Release. Native tester real-input and hidden process sampling remain
separate: session tab/mode state alone cannot prove visible rendering or real
tab-click/reopen behavior. Follow the active stage's exact measurement scope.
