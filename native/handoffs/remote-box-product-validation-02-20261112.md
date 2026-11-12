# Remote frozen product validation 02 — manual transport resumption

Date: 2026-11-12. Owner: `guard-evidence-review`, task-4.
Target: **authorized main box**; real box identifier and connection URLs are omitted.

## Final outcome: FAILED — VET HELD

The retained test phase actually executed and failed. **Original Go ReturnCode=1;
wrapper exit=2.** ProductAttempted=true, ProductExecuted=true, TimedOut=false,
Command.Error=null, Signal=null, CleanupEvents=[], CleanupErrors=[], PostGuardErrors=[].
The source and toolchain guard identities match before/after. A failed command with
clean guards is **not** COMMAND_PASSED_WITH_GUARDS or product acceptance.

The product operation stopped. No vet, retry, third launch, new run root, source/
assertion/skip repair, installation, upgrade, plugin change, local product execution,
commit or push was performed. This report closes execution evidence, not a passing
product gate. The current root decision is TEST_FAILED_VET_HELD.

## Authority, original fault and immutable report01

The complete manual decision in the [root review](<F:/AIproject/LevNote/native/handoffs/remote-box-product-validation-01-root-review-20261112.md>)
was read as the then-117-line document, together with task-4 revision 3. It authorized
one conditional additional transport request to complete the still-unobserved single
Go phase, not to rerun an existing command. After caller creation, the lead restricted
all subsequent work to reading the same attempt. No additional launch was requested.

[Report01](<F:/AIproject/LevNote/native/handoffs/remote-box-product-validation-01-20261112.md>)
was uncertainty-corrected and frozen before resumption: 94 lines, 6,417 bytes, SHA256
`61f803093ec2df658b0b342f59e156bf8e09ccf553826275715a05955bc27360`.
Its first-request `box agent unreachable` remains an unconfirmed historical start,
not an invented Go exit/pass/failure. All resumed outcomes are recorded only here.
The independent audit, prepare evidence, frozen input/helpers and accepted `.29`
were not edited.

## Fresh preflight and atomic same-path caller

The primary read-only check at `2026-11-12T03:06:27.489995+00:00` found the original
caller parent/target and test/vet phase directories absent, including symlinks;
no Go/compile/vet/link/.test/LevNote/Python peer process was found by comm/exe/cwd.
Process arguments and environment were not read. Receipt and wrapper were regular,
non-symlink files with the exact pinned length/SHA pairs:

| Pin | Bytes | Actual SHA256 |
| --- | ---: | --- |
| Guard receipt | 723 | `da70b0421c48165f9c3db880c75cac31c59f7901f9f55de583f05b0a135e0853` |
| Frozen wrapper | 33,772 | `7d53c6b7374a52c9d3e6749e423449d073778ac1c33c49888a121b97d43e17d7` |

Immediately before launch, the script repeated the same gates and saved the exact
snapshot at `2026-11-12T03:11:26.922451+00:00`. It atomically created the original
caller parent/target, then created capture files exclusively. A delayed original
request could not obtain another caller at that same path. Existing capture or phase
files were never overwritten or cleaned; the wrapper's separate create-only phase
interlock remained unchanged.

```text
Caller parent: /home/user/levnote-remote-caller-evidence-20261112-01
Same caller: /home/user/levnote-remote-caller-evidence-20261112-01/test-01
Run root: /home/user/levnote-remote-20261111-01
Frozen wrapper: /home/user/levnote-inbox-20261111-01/linux-guarded-validation-02.py
```

## Actual caller, wrapper and Go invocation

| Layer/field | Actual evidence |
| --- | --- |
| Launch request | Transport request 2; successful launch-query exit 0 is not a product exit |
| Launch UTC | `2026-11-12T03:11:27.226870+00:00` |
| Caller PID | 6819; independent session, stdin DEVNULL, redirected outer streams |
| Outer UTC-before | `2026-11-12T03:11:29.969693+00:00` |
| Outer UTC-after | `2026-11-12T03:22:13.923756+00:00` |
| Caller result | WrapperSpawned=true; WrapperPid=6821; WrapperReturnCode=2; CallerError=null |
| Wrapper phase start | `2026-11-12T03:11:36.632394+00:00` |
| Wrapper phase finish | `2026-11-12T03:22:13.897942+00:00` |
| Wrapper result | Status=FAILED; WrapperExitCode=2; ProductAttempted/Executed=true |
| Go PID / Spawned | 6830 / true |
| Go start | `2026-11-12T03:13:00.727256+00:00` |
| Go finish | `2026-11-12T03:21:52.160271+00:00` |
| Original Go return code | **1**, separately preserved from wrapper exit **2** |
| Go timeout / capture error / signal | TimedOut=false / Error=null / Signal=null |
| Go cleanup | CleanupEvents=[]; CleanupErrors=[] |
| Post-guards | PostGuardErrors=[] |

The recorded wrapper argv exactly matched the original authorized vector and binding;
only that boolean is reported locally. The complete real bound argv remains in remote
capture. Connector target was `main`; no --inbox or extra flags were added.

```text
Wrapper request: python3 -I -B <frozen wrapper> test
Original authorized box-id binding retained exactly (not printed here)
--run-root /home/user/levnote-remote-20261111-01
--guard-receipt-sha256 da70b0421c48165f9c3db880c75cac31c59f7901f9f55de583f05b0a135e0853
--activate-product-stage picker-final-app-linux-20261111

Actual Go argv:
/home/user/levnote-remote-20261111-01/toolchain/go/bin/go test -mod=vendor ./app -count=1
Actual Go Cwd: /home/user/levnote-remote-20261111-01/source/native
Actual TimeoutSeconds: 900
```

The actual captured environment is Linux/amd64/v1, CGO_ENABLED=0, local run-owned
toolchain and HOME/XDG/cache/temp/GOPATH, GOTOOLCHAIN=local, GOENV/GOWORK=off,
GOFLAGS empty, GOPROXY/GOSUMDB off, GOVCS=*:off, telemetry off, DISPLAY=:0.
No ambient environment bypass, raw replacement Go command or backend probe was used.

A same-PID read-only wait request returned `Request timed out`. This was a connector
request fault, not a Go timeout or wrapper exit. The detached caller was not signalled
or restarted. At `03:19:06.641576+00:00` it remained active; outer streams were then
0 bytes and completion files absent. Those were running snapshots, not final empty
streams. Subsequent reads collected the original completed files. The 900-second
limit ran from the Go start, not the caller start; no API/caller timeout replaced it.

## Original raw evidence and fingerprints

The outer and Go streams are distinct. The complete 32,050-byte Go stdout was read,
including the portion beyond the initial previews; original remote bytes were retained.
The final wrapper stdout is a failure summary, not the Go assertion stream:

```json
{"Phase": "test", "Status": "FAILED", "Evidence": "/home/user/levnote-remote-20261111-01/evidence/test-01/result.json", "ExitCode": 2}
```

| Evidence | Bytes | Actual raw SHA256 |
| --- | ---: | --- |
| Go stdout | 32,050 | `9d87e0a85ca1c0253dd5936dbd3a7baad5323e954217f20119304f7b6268afd6` |
| Go stderr | 0 | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| Outer wrapper stdout | 135 | `4b518c338e7ad15fbcc7b132a66c76e446822670e56a6cfe99dd2e9755be67e0` |
| Outer wrapper stderr | 0 | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| Go exit record | 2,317 | `ecb3b50841088d50d1c750ab0c3ba5e6d829d738d1f95ca981e8d9946f459404` |
| Phase result | 15,898 | `fc1022e5e48e77f4483c5754bf1f5c91287d099aee1bf18d332241cdd28b2253` |
| Caller result | 357 | `5177b4e4628d01730bc9aed795c6899d735d010cb6e9e31926cfd09faf5bdee2` |
| Bound wrapper argv | 374 | `9c30d83c2202b88ee9a5c53bd82ee780dbca306e96bf11dc136c9b78afc1bdf2` |
| Wrapper exit text | 2 | `53c234e5e8472b6ac51c1ae1cab3fe06fad053beb8ebfd8977b010655bfdd3c3` |
| UTC-before text | 33 | `5dc5718d7a4c5790caa818d244fb6d6a19380ad43b64864871b1fb815cb0eb2a` |
| UTC-after text | 33 | `68afa844842d9f6085455f3a916d661cd3f72e4164b65031a59496ca4a2dfab9` |
| Wrapper exception | 491 | `435bd081badfb6cc8e273a369da3c3ee16125b97553b40bf7f704e99a3a95650` |

```text
Outer captures under /home/user/levnote-remote-caller-evidence-20261112-01/test-01/:
preflight.json, wrapper-argv.json, caller-launch.json, caller-self.json,
wrapper-pid.json, utc-before.txt, utc-after.txt, wrapper-exit.txt,
wrapper-stdout.bin, wrapper-stderr.bin, caller-result.json, caller-runner.py,
launcher-stdout.bin, launcher-stderr.bin

Phase captures under /home/user/levnote-remote-20261111-01/evidence/test-01/:
result.json, invocation.json, environment.json, exception.txt,
inputs-before.json, inputs-after.json, toolchain-before.json, toolchain-after.json,
prerequisites-before/, go-command/command.json, go-command/exit.json,
go-command/stdout.bin, go-command/stderr.bin
```

The physical final caller-directory listing also shows both launcher streams at
0 bytes. Their launch-query exit 0 remains only a launch acknowledgement.

The wrapper exception is `RuntimeError: Go command failed; original return
code/streams retained`, raised by its fixed Go-success gate. Command.Error=null
means no command capture/spawn error was recorded; it does not erase test assertions
or the terminal panic.

## Before/after guards and completed process state

| Guard record | Before files/bytes | After files/bytes | Before and after aggregate Identity |
| --- | --- | --- | --- |
| Inputs | 1,208 / 17,045,848 | 1,208 / 17,045,848 | `96e0a103b8538f9810333082ed05217e47b1360e963a7dbc8dbef0585654e12d` |
| Toolchain | 15,657 / 244,139,206 | 15,657 / 244,139,206 | `9806db80da121ddc6891941ef94ce2da8e15855f15af3ee0e1e33eb51249f7b9` |

Both input records contain exactly 5 picker files and 26 patch files. The saved
before/after JSONs are byte-identical and parsed-identical, with these separate
raw JSON fingerprints; these are not substituted for aggregate input identities:

| Saved JSON pair | Each bytes | Before SHA256 = after SHA256 |
| --- | ---: | --- |
| Inputs before/after | 215,722 | `8a522eef4901375db812d97f20f7c030b62c5821d5c4900f481c7d0deb50a881` |
| Toolchain before/after | 2,619,707 | `ea06e7739c3743026d540ab75b503dd152f48467f2c1429145f19d45ea81dffc` |

The complete aggregate identities and saved JSON hashes match the accepted prepare
baseline. Readback compared the saved evidence only; it did not repeat a broad
source/toolchain tree hash or execute additional prerequisite probes.

Final metadata-only inspection at `2026-11-12T03:30:12.911475+00:00` found no relevant
active Go/compile/vet/link/.test/LevNote/Python peer process. Vet evidence was absent.
No process was killed or cleaned by the reviewer. The caller, Go and wrapper return
records and UTC-after were already complete.

## Failure classification — symptoms, not a blanket platform diagnosis

Full stdout has **332 lines, 54 top-level FAIL headers and 102 indented FAIL headers**.
These count observed printed failure headers, not total executed tests, distinct leaf
failures, passed/skipped counts or coverage. Categories below are descriptive and may
overlap; no failure was ignored, converted to a pass or repaired.

| Category | Actual observed symptoms |
| --- | --- |
| App layout and issue disclosure | Minimum-width plugin metadata rectangle outside 760x480; issue-history disclosure assertion failed. Raw issue list includes a Chinese 200-current/6-omitted sentence, but the asserted test still failed. |
| Recovery, live-tab and closed-file admission | Autosave failed-attempt pending/report-once assertions; missing reopen fixture receipts []; reopen not started for UTF8/BOM/UTF16/binary/legacy cases; SaveAs path not captured. |
| Reference paths and file URLs | Local file-URL target refused as encoded UNC; C:/ versus C:\ separator expectation; decoded local file-URL and double-encoded UNC classification mismatches. |
| Worker/package/export/preview/routing | Repeated export-did-not-reach-runner failures; unsupported-platform package messages; unavailable engine repository/poison assertion; preflight/status/barrier/oversize expectations differ before publication. Lifecycle, modal, cancellation, late-edit and shutdown tests consequently fail their own assertions. |
| Mindmap tag barriers and focus/guidance | Public barrier assertion fails with unsupported-platform message; preview/split first-warning cases report removing warning lost focus or retained stale guidance []. Marker/tag SVG paths also fail to reach runner. Informational control bounds in the log are not themselves labelled as failed assertions. |
| Terminal panic | TestReopenClosedWindowCloseCancelsRunningImage indexes an empty collection at the captured test stack location, causing a recovered/repanicked runtime index-out-of-range panic and package failure. |

Selected terminal lines (intervening stack frames omitted):

```text
--- FAIL: TestReopenClosedWindowCloseCancelsRunningImage (0.02s)
panic: runtime error: index out of range [0] with length 0 [recovered, repanicked]
levnote.local/native/app.TestReopenClosedWindowCloseCancelsRunningImage(0x36430e466248)
	/home/user/levnote-remote-20261111-01/source/native/app/reopen_closed_test.go:234 +0x605
FAIL	levnote.local/native/app	447.038s
FAIL
```

**The panic aborted the package test execution.** This was not a normally completed
whole suite. Later suite/picker coverage is unknown; no separate picker pass, complete
suite run or uppercase-temp-path result is claimed. The observed failures are broader
than the previously identified Windows-path qualifications; the log alone cannot
establish that all failures are platform-only.

## Backend/platform limits and next gate

A full case-insensitive scan of both final Go streams found no `text will not show`
substring. This is only absence of the negative veto token, not positive MyGo/Pango
backend/rendering proof. The fixed nonverbose test can suppress successful-test logs.
The prerequisite record reports FontMapAndContextCreated=true but MyGoBackendObserved
and PhysicalUIOrIMEObserved=false; those are prerequisite provenance flags, not an
observation of the actual Go backend. No additional backend/font probe was run.

NoPhysicalUIIMEOrWindowsDeliveryClaim=true is retained. Linux/CGO0 evidence does not
replace the four Windows-tagged files, C:/case semantics, GUI PE/five builds,
physical IME, WM_CLOSE/owned-process Job cleanup or separate `.30` delivery acceptance.
The accepted runnable remains `.29` and its evidence stays immutable.

**Next permitted step: lead reads this failed-command evidence and closes task-4 as
execution-evidence complete, not product passed. Vet remains held.** No source/skip/
assertion repair, rerun or fresh run root is authorized by this report.

## Appendix — all 54 observed top-level failure headers, in captured order

Names below identify printed top-level failure headers only. Nested 102 headers and
full diagnostics remain in the original raw stdout. The final panic limits coverage.

```text
TestPluginManagerRowsReflowAtMinimumWidth
TestIssueHistoryLimitIsDisclosed
TestReadonlyMindmapAtMinimumSizeKeepsTabModeAndCanvasUsable
TestRecoveryAutosaveFailureWaitsForMutationOrExplicitRetry
TestDocBlockReferenceRealAdapterReadonlyFileURLAndRefusal
TestDrawioRouteSecondPageFlushesOneTransactionBeforePrepare
TestDrawioRouteSameIDRefreshRejectsLateExport
TestEnginePackagePoisonRefusesTransitionsButLeavesInspectionAvailable
TestDrawioWorkerSecondPageExactPublicationPreservesState
TestDrawioWorkerFlushesRealSecondPageDraftBeforePrepare
TestDrawioWorkerRefusesBeforePreparationWithoutRunner
TestDrawioWorkerRefusesPreparationReentryWithoutRunner
TestDrawioWorkerRejectsStalePageViewAndSourceBeforePublish
TestMindmapWorkerModesReadonlyAndExactPublication
TestMindmapWorkerFieldFlushAndPendingTagRecovery
TestMindmapWorkerPreflightPairSourceCapAndFlushOwners
TestMindmapWorkerStaleSnapshotNoFallbackAndStatus
TestMindmapWorkerAndMarkdownShareAppAdmission
TestMindmapWorkerDisableRetainsSourceFallback
TestWorkerExportActualDescriptorAllModesAndPreservesSource
TestWorkerExportRefusesWithoutFallbackAndCachesHistoricalVerification
TestWorkerExportPreflightAndExclusivePublishing
TestWorkerExportLateEditsUndoAndDialogGuards
TestWorkerExportLifecycleSuccessAndFailure
TestWorkerExportNestedModalDeferral
TestWorkerExportShutdownAndAcceptedCloseJoinWithoutDispatch
TestWorkerExportNativeProgressCancelAndStatusImages
TestWorkerExportCanceledCleanupFailureIsNotHidden
TestWorkerExportFailedCloseRestoresDiscardedRecovery
TestWorkerExportFlushesRealPendingCompositionBeforeSnapshot
TestWorkerExportBusyReuseAndSnapshotSurvivesLiveEdit
TestFormatRoutingReadonlySceneSaveAsAndDisabledFallback
TestFormatRoutingSameIDRefreshRejectsLateSceneAndExport
TestLiveTabAdmissionByteFallbackAndClosedReceiptRetry
TestLiveTabAdmissionClosedReceiptReusesOpenPathAtCap
TestMindmapReadonlySaveAsCreatesEditableFieldView
TestMindmapMarkersRealAdapterSaveReopenUndoAndReadonly
TestMindmapTagsRealAdapterExactSaveUndoModesAndSVG
TestMindmapTagsRealAdapterPublicBarriersAndSeparateFieldUndo
TestMindmapTagsRealAdapterFirstWarningPreservesComposition
TestBuiltinMindmapReadonlyHiddenTargetRefusesWithoutMutation
TestOversizeRichViewsSupportSourceToRichModeLifecycle
TestScenePreviewRegisteredReadonlySnapshotAndNoSourceFallback
TestScenePreviewExportPriorityJoinReservationAndResume
TestScenePreviewOwnerReleaseAndCanceledNewestPending
TestScenePreviewNewestPendingSnapshotAndStaleSuppression
TestScenePreviewNestedModalAndInvalidatedCleanupRemainVisible
TestScenePreviewMalformedAndUnavailableDoNotRetry
TestScenePreviewShutdownAndCanceledQueuedExportNoDispatch
TestResolveLocalReferenceClassification
TestReopenClosedReadsCurrentDiskAndPreservesBytes
TestReopenClosedDoesNotRememberLegacyEncoding
TestReopenClosedSaveAsDiscardAndNoDraftAuthority
TestReopenClosedWindowCloseCancelsRunningImage
```
