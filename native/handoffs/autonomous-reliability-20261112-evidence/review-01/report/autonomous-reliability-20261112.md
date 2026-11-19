# Autonomous native reliability — task-5 / 2026-11-12

Status: **IN PROGRESS. Focused development tests passed; original affected validation, vet, and application acceptance are pending.**
Owner: `native-reliability`, the sole new remote Go/gofmt/test/vet/build lane.

## Authorization and immutable boundaries

- This assignment follows [the autonomous delivery plan](<native/handoffs/autonomous-native-delivery-plan-20261112.md>), not a restart of task-4 or a historical owner.
- All semantic source/test/helper work and product commands originate in the authorized main box's new lane. Local files are byte-exact review/Git mirrors and planning documents; no local Go, gofmt, tests, vet, build, application execution, or semantic fallback.
- The ordinary box clone stays at authorization baseline `8897244ffa4f00f5ec51f95545e7a3ca27d37d97`; the subsequent CI-only workflow-disable commit does not authorize fast-forwarding it or altering any frozen input.
- Original prepare02, failed caller/run-root/source/toolchain/raw evidence and accepted `.29` package remain immutable. [Original report02](<native/handoffs/remote-box-product-validation-02-20261112.md>) is still FAILED: Go1/wrapper2, no timeout or cleanup/postguard discrepancy. Its 54 top-level/102 nested FAIL headers are printed headers, not complete suite totals; the panic aborted later coverage.
- Owned production is [reopen_closed.go](<native/app/reopen_closed.go>), [reference.go](<native/app/reference.go>), and the native-identity comparison plus necessary unused-import removal in [binary_actions.go](<native/app/binary_actions.go>). Own regressions are [reference_test.go](<native/app/reference_test.go>) and [autonomous_reliability_test.go](<native/app/autonomous_reliability_test.go>). Original [reopen_closed_test.go](<native/app/reopen_closed_test.go>) and its Windows assertions remain unchanged.
- Experience owns its app/view/workspace/outline/list inputs; readiness owns drawio production/tests. Reliability freezes and executes their exact new groups serially, without editing their semantic source.

## Failure classification: cause is not an acceptance waiver

| Class | Concrete evidence and diagnosis | Disposition |
|---|---|---|
| Genuine native history admission defect | Production admitted only full Windows drive paths, excluding real Unix absolute files. Original `TestReopenClosedWindowCloseCancelsRunningImage` then indexed empty `closedFiles[0]` at line234. | Actual public native reopen RED/GREEN; native admission now runs before/after canonicalization and retains UTF8/NUL/length, cloning and bounded metadata-only history. No panic-only guard. Original affected test still awaits execution. |
| Genuine native file identity defect | Real Unix `case.txt` and `CASE.txt` were falsely merged by folded Open/close/history identity. | Actual real-file behavioral RED and corrected GREEN. Windows still uses `strings.EqualFold`; Unix uses exact native identity. Binary duplicate comparison shares this policy. |
| Native references and Windows lexical interpretation | Unix absolute references were classified as Windows root-relative; `file:///...` lost its genuine root; Windows literals used host-dependent cleaning. The existing URL fixture added a second slash on Unix. | Actual public Host RED/GREEN plus original19 classification assertions passed. Unix root stays; only `/C:/...` loses its URL separator; Windows literals clean lexically on either host; decode once; refuse double/mixed roots. Fixture construction is native-volume aware. |
| File URL token defect | Bare literal `#` had an empty parsed fragment and opened the target; encoded `%23` is instead a legitimate filename. | Actual behavioral RED retained. Minimal raw-literal fragment guard passed focused validation with independent bare-`#` negative and encoded filename positive, alongside query/NUL refusals and failed-reopen retry/history retention. `ForceQuery` bare-`?` guard remains. |
| Original Windows case fixture | Bound/dedup history fixture uppercases the full real temp path. Case-sensitive Unix must not impersonate Windows to satisfy it. | Original fixture/assertions unchanged; no skip or fabricated folded filesystem. Full affected result and exact limitation are pending. |
| Fixed Windows worker/store contracts | Windows/amd64 engine catalog match precedes controlled runners; non-Windows verified-runner, locking and package-store implementations explicitly refuse. | Catalog/platform gates remain unchanged. No fake-runner movement or widened platform descriptor; Windows backend validation pending. |
| Linux startup issue accounting | `initializeEnginePackages` adds the explicit unavailable-store issue. A later205-issue fixture therefore drops six rather than its Windows five; exact-one autosave issue fixtures also include the startup issue. | Source-confirmed cause, not executed autosave proof, acceptance, or permission to suppress errors/weaken assertions. |
| Font/layout/focus/guidance failures | Original output contains failures beyond paths. Absence of `text will not show` is a negative veto, not rendering/backend proof. | Specific source and affected-execution diagnosis still pending; not blanket platform-only. Experience Tree already has a separate actual public behavioral RED, but its first current GREEN did not compile. |
| Panic-aborted coverage | Original app package terminated in the index panic. | Original later/picker coverage is unexecuted/unknown; neither focused groups nor source accounting retroactively complete it. |

## Lane, environment and byte-exact review mirrors

```text
lane = /home/user/levnote-dev-reliability-20261112-01
ordinary baseline = /home/user/LevNote/native
baseline HEAD = 8897244ffa4f00f5ec51f95545e7a3ca27d37d97
consumed helper02 = lane/tools/dev-remote-validation-20261112-02.py
helper02 bytes / SHA256 = 17046 / 0085947e72015013a67720d90798fe71d2f32ddf3a8f3e51e9d94d9487555c21
latest own focused input = lane/inputs/native-path-focused-01/native
focused manifest = lane/inputs/native-path-focused-01/expected-manifest.json
manifest bytes / SHA256 = 201208 / 114e90ec2c516c9c203f7e8e56ad5f6843e7d1d183cb906e94ee584fb4c831af
focused runtime/build/test inputs = 1192 files / 16978935 bytes
focused source-before/source-after = 199410 bytes / 2a087081672f38a6161a8e9f25aff421a00f6e8b9ac8d0829ba803779b5c6bda
```

The input roots are exactly `go.mod`, `go.sum`, `app`, `cmd`, `core`, `formats`, `modules`, `ui`, `vendor`, `dependency-patches`, `examples`. Original RED was pure889 plus the1470-byte public test, not a claim that the regression was committed in889.

Effective runs use Go1.27.3 Linux/amd64, `GOTOOLCHAIN=local`, `CGO_ENABLED=0`, `GOWORK=off`, `GOVCS=*:off`, `GOFLAGS=-mod=vendor`, `GOPROXY=off`, `GOSUMDB=off`, owned HOME/XDG/cache/tmp. Owned XDG `go/telemetry/mode` contains `off` before Go. `GOENV=off` correctly has effective empty filename. No user/ordinary telemetry configuration changed.

Five local review mirrors were checked against the complete box-authored formatted file identities, not reconstructed semantic suggestions:

| Mirror | Bytes | SHA256 |
|---|---:|---|
| [reopen_closed.go](<native/app/reopen_closed.go>) | 8564 | `a6b8013854d4c27d68e2ae1153525b0764d39b8892a700ce8804b206e485da44` |
| [reference.go](<native/app/reference.go>) | 8049 | `9c5ebfd54b1bc1ed1de659bef13d34b29018a0cc2cfd572af91598e7e57f98cd` |
| [reference_test.go](<native/app/reference_test.go>) | 7133 | `50451754b6389d8642c78295df32876f5b21ca103bd60c07d9117f3c7acabb4a` |
| [binary_actions.go](<native/app/binary_actions.go>) | 10163 | `8174e9a8b3b89fd22a349292fd8d7c39cb905d655400fe7ba581255ffef8111b` |
| [autonomous_reliability_test.go](<native/app/autonomous_reliability_test.go>) | 9467 | `2a3c83ffb229270aec79840aa072da13507f260e5d37261d65f5c92c20f5ce33` |

Box metadata export is create-only under `evidence/own-path-source-mirror-02`; the readable exact four-file delta is `evidence/own-path-source-readable-mirror-01/source.diff`. Neither export is product execution. The new public test contains five native reopen/identity/Host/token/failed-retry tests and preserves the original1470-byte public reopen body.

## Saved actual new executions

Every product row has fresh label, pinned source/manifest, actual PID/owned PGID, raw streams, timestamps, numeric exit, timeout/cleanup, and source/toolchain/environment checks. No consumed label was relaunched.

| New namespace inside lane | Actual result | Scope / retained failure |
|---|---|---|
| `evidence/environment-01` | Go metadata0 / wrapper2 | Original helper incorrectly expected effective GOENV `off`; corrected metadata02/03 passed. Not product acceptance. |
| `runs/reopen-red-01` | Go10901/1, wrapper2 | Real public user reopen: `没有可重新打开的记录`; 06:05:49.356514Z–06:08:48.976494Z; no timeout, cleanup empty/group gone. Connector wait did not cause a relaunch. |
| `runs/reopen-green-01` | Go14013/0, wrapper0 | Same unchanged1470-byte public test; 06:26:44.952458Z–06:29:06.057346Z; no timeout/cleanup/group. Only native admission changed. |
| `runs/native-case-red-01` | Go14785/1, wrapper2 | Real distinct native file falsely deduplicated: tabs1. |
| `runs/native-case-green-01` | Go1, wrapper2 | Actual compile failure: `app/binary_actions.go:8:2: "strings" imported and not used`. Preserved. |
| `runs/native-case-green-02` | Go0, wrapper0 | Same native case oracle after only necessary unused-import removal; app0.129s. |
| `runs/native-reference-red-01` | Go16951/1, wrapper2 | Native absolute misclassified; native file URL root stripped. 06:47:55.948488Z–06:48:19.401050Z; no timeout/cleanup/group. |
| `runs/native-reference-green-01` | Go0, wrapper0 | Public two-form Host test plus original classification; app0.010s. |
| `runs/native-token-red-01` | Go1, wrapper2 | Bare `#` opened the `%23` target: tabs2/message已打开. Failed-reopen history-retention test was in the selector; no separate RED claim for that passing test. |
| `runs/native-path-format-01` | Command0, wrapper0 | Five exact own files only. Separate create-only `evidence/native-path-format-byte-equivalence-01` proved the entire formatted snapshot inventory equaled own current work, not just selected file hashes. |
| `runs/native-path-focused-01` | Go19128/0, wrapper0 | 07:02:21.292365Z–07:03:00.103299Z,38.810953s, app0.428s; no timeout/cleanup/group. Ten top-level/29 nested PASS headers, not whole-suite totals. |
| `runs/experience-tree-baseline-red-01` | Go19847/1, wrapper2 | Pure889+unchanged7900-byte public Tree test: `End selected -1, want the current root's last row`; 07:14:52.643348Z–07:15:30.176931Z; no timeout/cleanup/group. No chronological RED-before-authored-peer-source claim. |
| `runs/experience-tree-format-01` | Command0, wrapper0 | Only list_lifetime_test formatting changed17379→17585 bytes; priority7900-byte public oracle unchanged. Raw freeze02 preserved; separate formatted derivative. |
| `runs/experience-tree-current-green-01` | Go1, wrapper2 | Real compile failure at list_lifetime_test.go:448: `tester.Render undefined`; no Tree behavior executed. Peer freeze03 fixes only public `Image()` API and authorized synthetic searchRoot fixture. New GREEN and25case group pending. |
| `runs/drawio-red-01` | Go1, wrapper2 | `Document.DeleteCell undefined`: compile RED, not behavioral RED. |
| `runs/drawio-green-01` | Go16155/0, wrapper0 | Original unchanged1738-byte literal source-preservation oracle; 06:40:31.832003Z–06:40:33.716281Z; no timeout/cleanup/group. |
| `runs/drawio-incident-red-02` | Go21271/1, wrapper2 | Actual behavioral refusal of known out/in/self-loop edges at delete_test.go:52. 07:23:22.580149Z–07:23:27.855506Z; no timeout/cleanup/group. Peer's new6219-byte production is frozen for fresh two-case GREEN02. |

Latest own focused command:

```text
go test -v -count=1 -timeout=120s
  -run '^(TestAutonomousReliability.*|TestResolveLocalReferenceClassification|TestEditorHost.*)$'
  ./app
```

Its stdout is5464 bytes/SHA256 `e1a0c138d7e53a512856975f23963137eb6239ce8e6127384d9b9fe8b882ca1e`; stderr0. It covers five new public tests, original19 classification subtests, stale closed/inactive Host refusals, saved/unsaved source barriers, external/remote refusals and outline-anchor Flush. The empty-suffix encoded-`#` positive is verbose subtest `#00`, distinct from the bare-`#` negative.

## Runner ownership and remaining work

[Immutable helper02 mirror](<native/scripts/dev-remote-validation-20261112-02.py>) pins explicit argv/cwd/env/input snapshots and refuses alternate tool/module/platform/output inputs. Inspection found its post-Popen `.started.json` write is outside the owned wait/cleanup block. Lead authorized only a new helper03 to preserve lifecycle/error recording on that writer failure; helper02 and all consumed results are unchanged. The new helper03 and real owned-child fault validation are still pending, not an accomplished acceptance claim.

Next work, in the sole serial lane:

1. Create box-authored helper03 and actual narrowly owned started-receipt fault RED/GREEN/control evidence; mirror its exact bytes. Preserve helper02 and every old label.
2. Freeze/format/execute readiness's unchanged two-oracle incident GREEN02, then the requested affected format group; no readiness-owned Go run.
3. Freeze/format/execute experience03's unchanged7900-byte Tree priority GREEN02, then exact new8+affected17 selector. Preserve freeze01/02/raw/format/failedGREEN01; no skip or assertion deletion.
4. Execute the original case/reference/reopen/binary affected group on own corrected source, then affected vet with actual outcomes. Classify remaining original fixture/platform/font/layout/focus failures specifically; do not label all failures platform-only.
5. Only after path affected closure can task-5 complete and task-11 begin. `.29` stays latest accepted runnable; Linux development, focused success or cross-build cannot replace Windows/IME/PE/WM_CLOSE/Job/backend/whole-app acceptance.


## Current receipt closure append — 2026-11-12

**Current status: own source corrections and Linux affected/vet execution are complete for Root review. The original affected invocation is FAILED (one preserved Windows-only case fixture); final Windows acceptance is PENDING, not a development hold or a PASS. Task-5 final acceptance remains a Root decision.**

All paragraphs above are the preserved incoming 14017-byte chronology, SHA256 `678ba7a178818ecf63a0f7097e120d92cd59000394eff10c96eaed45487cf69e`; their then-pending statements do not describe the completed receipts below. Original report02, helpers01/02, all consumed inputs/runs/raw receipts, original assertions/skips and `.29` remain immutable. No source was reauthored locally and no consumed invocation was repeated.

### Helper03 actual real-child writer-fault closure

- Helper02 RED caller21533, real child/PGID21534, 2026-11-12T08:21:42.419006+00:00–2026-11-12T08:21:43.009335+00:00, metadata exit1. Injected started-receipt OSError escaped while the child was alive; only the caller's newly spawned group was SIGTERM-cleaned and confirmed gone. The old defective helper was preserved.

- Helper03 GREEN caller21592, 2026-11-12T08:33:12.410258+00:00–2026-11-12T08:33:12.485759+00:00, metadata exit0. Fault child21593/PGID21593 finalized numeric -15, recorded `started_receipt_error`, helper-owned SIGTERM reason `started receipt write failed`, no timeout or remaining group. Normal child21594 finalized0 with exact14-byte `owned control\n` stdout SHA256 `6588bc0131893cb76a0a27aea154412692680a28a4fbd01ccbe15e943cb7594c`, stderr0 and no cleanup/error.

- Helper03 before/after identity17964 bytes/SHA256 `6f3dd79076e9f284691f24be1a8e3b0c91d57d9c0ed5766d07cc87445acdff89`; [exact local mirror](<native/scripts/dev-remote-validation-20261112-03.py>) and harness already independently reviewed by Root. This is actual Python process/evidence-boundary verification, explicitly `product_executed=false`; it is not Go or product acceptance. Harness and source mirrors were not repeated.

- RED/GREEN original metadata and raw streams are included in the new review evidence bundle; full original source paths and numeric receipts remain in those records.

### Original path/reference/reopen/binary affected and vet closure

- `native-path-affected-01` reached its final selected case without panic: **47 top-level RUN headers,45 PASS,1 FAIL,1 original SKIP;76 nested PASS headers**. The only FAIL is `TestReopenClosedBoundDedupAndShutdownClearBacking` at unchanged [reopen_closed_test.go:276–279](<native/app/reopen_closed_test.go#L276-L279>): it uppercases the entire actual temporary file path before Open. Native Unix identity must remain case-sensitive; the original Windows assertion was neither weakened nor skipped and no fake case-folded filesystem was introduced.

- Original `TestReopenClosedWindowCloseCancelsRunningImage` actually PASSes in this affected invocation; genuine native absolute admission is fixed, not an index guard. All five new native tests, original classification/Host cases, other original reopen cases, selected image/binary byte/admission cases PASS.

- The sole original SKIP is [TestNativeImageCaptures](<native/app/binary_test.go#L696-L701>): its pre-existing `LEVNOTE_IMAGE_CAPTURE_DIR` opt-in was unset and `explicit synthetic image/byte captures` was printed. It is not new, not PASS, and does not establish image capture/physical UI coverage.

- The new matching-input `go vet ./app` invocation actually exited0 with empty streams, no timeout/cleanup/remaining group and unchanged input/snapshot inventories. This limited vet does not erase the affected test exit1 or prove whole-app, Windows, backend or IME acceptance.

- Root latest sequencing permits source+Linux evidence closure to release the task-11 dependency after original review, while the unchanged Windows-uppercase scenario remains in consolidated platform acceptance. Task-11 is not auto-started or self-authorized by this report.

### Peer runs: concrete current failures and retained coverage limits

- Drawio incident GREEN02 actually passed both original source-preservation and known out/in/self-loop incident-edge oracles. It does not complete the graphview public UI or final full-format gate.

- Experience03 Tree current GREEN02 is a genuine behavioral FAIL: unchanged7900-byte public oracle `End selected -1, want the current root's last row` at line64. The preceding current GREEN01 was compile1 (`tester.Render undefined`), not behavior. Revision03 changed only the public `Image()` call and approved synthetic search-root fixture; its formatted lifetime test is17584 bytes/SHA256 `643c72e1f6bbcec4b90e3f6659854eed544fc66f851affe6889cef43774e687b`, priority Tree oracle unchanged.

- The exact25 selector then executed11 top-level tests (9 PASS,2 FAIL) and10 nested PASS headers before `TestOutlineListHandleReleaseAfterSidebarBuild` produced `ui: two elements under one parent have the key ""` at `outline.go:151`;14 selected top-level cases had not started. No25/25 completion or retroactive full coverage is claimed. Revision04 is report-only, not source04. Root authorized only a new revision05 for the two real failures; frozen03, old consumed runs and oracles stay unchanged.

- Drawio public canvas RED01 actually failed at line38: real Adapter/View/Tester click then Delete left changed0, original CRLF XML, reports[]. The format-only public-test derivative is2689 bytes/SHA256 `8938202e1a2e94d156a42a1f3d00dfa6153628b8fb513d6186bff9c878831541`; original2687-byte prepared test remains preserved.

- Narrow drawio public UI GREEN01 reached the exact Delete transaction assertions but failed at line45: native Undo key left changed1 and the deleted XML unchanged. Redo was not reached. Same2689-byte public oracle remained exact; only new `input.go`6000 bytes/SHA256 `402188fdfebb2d8c1668f99040f3466b3118321da75374abd25f35ba7af99b4e` and `drawio_delete.go`5583 bytes/SHA256 `1e9c5b40cee91417e089cbed973b4931d452c09bc63b42ce6bc6b8d56dcc4650` were authored by readiness and gofmt caused zero byte changes. Readiness owns only a new narrow public Undo/focus correction, no assertion/history/input overwrite.

- After native-UI/edge/button slices converge, one full `./formats/drawio ./modules/graphview` plus vet remains to run. No early repeated full suite, unrelated app packages or readiness-owned Go executions were added.

### Exact completed command receipts


#### `drawio-incident-green-02`

```text
/home/user/go/bin/go test -v -count=1 -timeout=120s -run ^TestDeleteCell(PreservesUnrelatedSource|RemovesIncidentEdgesOnce)$ ./formats/drawio
```

- Actual PID/PGID21736/21736; launched `2026-11-12T08:53:28.343356+00:00`; ended `2026-11-12T08:53:30.931140+00:00`; elapsed2.589414s; numeric command exit0; wrapper exit0; outcome `COMMAND_PASSED_LIMITED_SCOPE`. No timeout, cleanup or remaining group.

- Input manifest200417 bytes/SHA256 `5e69cf222b84953b1f249d1ba25578a3c056f59a9237ac703402a20dcdbd6de8`;1193 files/16978725 bytes. Source-before/after/snapshot-after199550 bytes/SHA256 `a70511188f4bbc0a0153d5e22c308cc5c375db82d31f0f0f27ada443a511b955`; actual maps and both current input/run snapshots are equal.

- Raw stdout265 bytes/SHA256 `ec4995e3b8397f4f71f0349544f3c427695a918725a21002409f00d32e3b6079`; stderr0 bytes/SHA256 `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`. Top printed PASS/FAIL/SKIP2/0/0; headers are not unselected-suite coverage.

- [Raw final command](<native/handoffs/autonomous-reliability-20261112-evidence/review-01/runs/drawio-incident-green-02/05-product-command.json>), [raw stdout](<native/handoffs/autonomous-reliability-20261112-evidence/review-01/runs/drawio-incident-green-02/05-product-command.stdout.bin>), [full primary result](<native/handoffs/autonomous-reliability-20261112-evidence/review-01/runs/drawio-incident-green-02/result.json>). These are new byte-exact copies of consumed receipts, not new launches.


#### `experience-tree-current-green-02`

```text
/home/user/go/bin/go test -v -count=1 -timeout=120s -run ^TestTreeListHandleRootSwitchKeepsFreshKeyboardTarget$ ./app
```

- Actual PID/PGID21905/21905; launched `2026-11-12T09:04:17.129302+00:00`; ended `2026-11-12T09:04:48.087628+00:00`; elapsed30.959846s; numeric command exit1; wrapper exit2; outcome `COMMAND_FAILED`. No timeout, cleanup or remaining group.

- Input manifest202370 bytes/SHA256 `b3ed0e6264d4492c3ffedeec481b7a695b6f4fac179f4e4b136f5e9f465bd650`;1194 files/17000388 bytes. Source-before/after/snapshot-after199696 bytes/SHA256 `c01240a81ddda2b4d9b8b7bbd5efc9887fe2fb1b02d8d22fccf78ba47a48ba89`; actual maps and both current input/run snapshots are equal.

- Raw stdout268 bytes/SHA256 `e1a01a3155070e4c00615c69cc25467c6e34af8608c2320e37dc5a902f86cdc3`; stderr0 bytes/SHA256 `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`. Top printed PASS/FAIL/SKIP0/1/0; headers are not unselected-suite coverage.

- [Raw final command](<native/handoffs/autonomous-reliability-20261112-evidence/review-01/runs/experience-tree-current-green-02/05-product-command.json>), [raw stdout](<native/handoffs/autonomous-reliability-20261112-evidence/review-01/runs/experience-tree-current-green-02/05-product-command.stdout.bin>), [full primary result](<native/handoffs/autonomous-reliability-20261112-evidence/review-01/runs/experience-tree-current-green-02/result.json>). These are new byte-exact copies of consumed receipts, not new launches.


#### `experience-list-affected-01`

```text
/home/user/go/bin/go test -v -count=1 -timeout=180s -run ^(TestAppListHandlesSurviveSemanticResets|TestTreeListHandleRootSwitchKeepsFreshKeyboardTarget|TestSearchListHandleNewSearchRejectsStaleDelivery|TestOutlineListHandleReleaseAfterSidebarBuild|TestOutlineListHandleReentrantFlushAndRetry|TestHiddenListFocusDoesNotStealEditorInput|TestTreeRootSwitchKeepsComposingQueryInput|TestWorkspaceResultsMinimumWindowControls|TestWorkspaceTreeDisclosesPerDirectoryLimit|TestWorkspaceTreeVirtualizesRowsAndKeepsExpansionAnchor|TestWorkspaceSearchVirtualizesFarResultAndReplacement|TestWorkspaceSearchIssueDetailsAreBounded|TestSourceOutlineFactoryDoesNotRequireRichAdapterAndInvalidatesByGeneration|TestOutlineSourceModeResolvesOldIdentityAfterFlushWithoutCreatingView|TestOutlineActivationFlushesPendingNativeComposition|TestOutlineFlushFailureRetainsStateAndNegativeOffsetIsActionable|TestOutlineActivationRejectsLifecycleChangesDuringFlush|TestOutlineEmptyButTruncatedSnapshotKeepsPartialNotice|TestOutlineVirtualListReachesFarRowAndActivatesByKeyboard|TestOutlineSnapshotFailureDoesNotRetryEveryFrameAndReleaseInvalidates|TestBuiltinOutlineReleasedOnPluginModeTabAndReload|TestBuiltinMarkdownLargeOutlineCachesSnapshotAndVirtualizesRows|TestFindKeyboardFocusLeaveReconcilesComposition|TestFindbarControlsFitMainPaneAtMinimumWindow|TestFindbarActionsAndFocusSurviveMinimumReflow)$ ./app
```

- Actual PID/PGID22596/22596; launched `2026-11-12T09:07:32.536079+00:00`; ended `2026-11-12T09:08:00.583027+00:00`; elapsed28.048415s; numeric command exit1; wrapper exit2; outcome `COMMAND_FAILED`. No timeout, cleanup or remaining group.

- Input manifest202365 bytes/SHA256 `4a74107191d89dcdb24ed955e88ee95ef0bce0cfdd0332399c95b520fbbdef42`;1194 files/17000388 bytes. Source-before/after/snapshot-after199696 bytes/SHA256 `c01240a81ddda2b4d9b8b7bbd5efc9887fe2fb1b02d8d22fccf78ba47a48ba89`; actual maps and both current input/run snapshots are equal.

- Raw stdout14421 bytes/SHA256 `8c9c7f67aef58329f17aa494e5152232c57f59339602f961a5d09aebfab9e289`; stderr0 bytes/SHA256 `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`. Top printed PASS/FAIL/SKIP9/2/0; headers are not unselected-suite coverage.

- [Raw final command](<native/handoffs/autonomous-reliability-20261112-evidence/review-01/runs/experience-list-affected-01/05-product-command.json>), [raw stdout](<native/handoffs/autonomous-reliability-20261112-evidence/review-01/runs/experience-list-affected-01/05-product-command.stdout.bin>), [full primary result](<native/handoffs/autonomous-reliability-20261112-evidence/review-01/runs/experience-list-affected-01/result.json>). These are new byte-exact copies of consumed receipts, not new launches.


#### `drawio-ui-red-01`

```text
/home/user/go/bin/go test -v -count=1 -timeout=120s -run ^TestDrawioCanvasDeleteUndoRedoPreservesSource$ ./modules/graphview
```

- Actual PID/PGID23334/23334; launched `2026-11-12T09:13:55.119698+00:00`; ended `2026-11-12T09:14:09.337433+00:00`; elapsed14.219004s; numeric command exit1; wrapper exit2; outcome `COMMAND_FAILED`. No timeout, cleanup or remaining group.

- Input manifest200784 bytes/SHA256 `773637259d7eda579a8992cfe99a7468e7515afd010c2477edff98a1dd929531`;1194 files/16981414 bytes. Source-before/after/snapshot-after199710 bytes/SHA256 `8e033e300e8985f74b46a0f64c25f42def6b9a9dc63d21f7a6530dfff434b886`; actual maps and both current input/run snapshots are equal.

- Raw stdout706 bytes/SHA256 `7321405dc99f86f6c8ff4edd3379c0538301e4c778287503dd304dfcf9d5e84e`; stderr0 bytes/SHA256 `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`. Top printed PASS/FAIL/SKIP0/1/0; headers are not unselected-suite coverage.

- [Raw final command](<native/handoffs/autonomous-reliability-20261112-evidence/review-01/runs/drawio-ui-red-01/05-product-command.json>), [raw stdout](<native/handoffs/autonomous-reliability-20261112-evidence/review-01/runs/drawio-ui-red-01/05-product-command.stdout.bin>), [full primary result](<native/handoffs/autonomous-reliability-20261112-evidence/review-01/runs/drawio-ui-red-01/result.json>). These are new byte-exact copies of consumed receipts, not new launches.


#### `native-path-affected-01`

```text
/home/user/go/bin/go test -v -count=1 -timeout=300s -run ^(TestAutonomousReliability.*|TestReopenClosed.*|TestResolveLocalReferenceClassification|TestEditorHost.*|TestNativeImage.*|TestBinaryInspectorLabelsOriginalBytes|TestLiveTabAdmissionAsyncBinaryCleanupAndDuplicate)$ ./app
```

- Actual PID/PGID23843/23843; launched `2026-11-12T09:16:00.042582+00:00`; ended `2026-11-12T09:16:26.094242+00:00`; elapsed26.052711s; numeric command exit1; wrapper exit2; outcome `COMMAND_FAILED`. No timeout, cleanup or remaining group.

- Input manifest201209 bytes/SHA256 `f118f82abf54b7083289f953eb0b65097b15e3314caa999973df5b764b29a86f`;1192 files/16978935 bytes. Source-before/after/snapshot-after199410 bytes/SHA256 `2a087081672f38a6161a8e9f25aff421a00f6e8b9ac8d0829ba803779b5c6bda`; actual maps and both current input/run snapshots are equal.

- Raw stdout17463 bytes/SHA256 `d0440f9b2a8b9ac995985b175ff120998c80c49be80ecb1e5caa38b92291575e`; stderr0 bytes/SHA256 `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`. Top printed PASS/FAIL/SKIP45/1/1; headers are not unselected-suite coverage.

- [Raw final command](<native/handoffs/autonomous-reliability-20261112-evidence/review-01/runs/native-path-affected-01/05-product-command.json>), [raw stdout](<native/handoffs/autonomous-reliability-20261112-evidence/review-01/runs/native-path-affected-01/05-product-command.stdout.bin>), [full primary result](<native/handoffs/autonomous-reliability-20261112-evidence/review-01/runs/native-path-affected-01/result.json>). These are new byte-exact copies of consumed receipts, not new launches.


#### `native-path-vet-01`

```text
/home/user/go/bin/go vet ./app
```

- Actual PID/PGID24554/24554; launched `2026-11-12T09:24:11.825077+00:00`; ended `2026-11-12T09:24:46.435805+00:00`; elapsed34.614990s; numeric command exit0; wrapper exit0; outcome `COMMAND_PASSED_LIMITED_SCOPE`. No timeout, cleanup or remaining group.

- Input manifest201204 bytes/SHA256 `72aaa56953f8603b95e64ff4d45fb0ca21158f0cab5c286e517fe6bf1c99e642`;1192 files/16978935 bytes. Source-before/after/snapshot-after199410 bytes/SHA256 `2a087081672f38a6161a8e9f25aff421a00f6e8b9ac8d0829ba803779b5c6bda`; actual maps and both current input/run snapshots are equal.

- Raw stdout0 bytes/SHA256 `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`; stderr0 bytes/SHA256 `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`. Top printed PASS/FAIL/SKIP0/0/0; headers are not unselected-suite coverage.

- [Raw final command](<native/handoffs/autonomous-reliability-20261112-evidence/review-01/runs/native-path-vet-01/05-product-command.json>), [raw stdout](<native/handoffs/autonomous-reliability-20261112-evidence/review-01/runs/native-path-vet-01/05-product-command.stdout.bin>), [full primary result](<native/handoffs/autonomous-reliability-20261112-evidence/review-01/runs/native-path-vet-01/result.json>). These are new byte-exact copies of consumed receipts, not new launches.


#### `drawio-ui-green-01`

```text
/home/user/go/bin/go test -v -count=1 -timeout=120s -run ^TestDrawioCanvasDeleteUndoRedoPreservesSource$ ./modules/graphview
```

- Actual PID/PGID26273/26273; launched `2026-11-12T09:39:19.407594+00:00`; ended `2026-11-12T09:39:36.406772+00:00`; elapsed17.000601s; numeric command exit1; wrapper exit2; outcome `COMMAND_FAILED`. No timeout, cleanup or remaining group.

- Input manifest201490 bytes/SHA256 `da6b463ab7e61c32f8e10769c0e1973fa1316ff07f4a6a995970cf8096cb69ec`;1195 files/16987405 bytes. Source-before/after/snapshot-after199865 bytes/SHA256 `14c98a13191f4650b6478c80dd0a4bff1ba80cb59fd2926fda738203210d07f4`; actual maps and both current input/run snapshots are equal.

- Raw stdout538 bytes/SHA256 `76f06df775f0c3f572eba49d6af39d118e918fba85d5210787f71e5469464580`; stderr0 bytes/SHA256 `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`. Top printed PASS/FAIL/SKIP0/1/0; headers are not unselected-suite coverage.

- [Raw final command](<native/handoffs/autonomous-reliability-20261112-evidence/review-01/runs/drawio-ui-green-01/05-product-command.json>), [raw stdout](<native/handoffs/autonomous-reliability-20261112-evidence/review-01/runs/drawio-ui-green-01/05-product-command.stdout.bin>), [full primary result](<native/handoffs/autonomous-reliability-20261112-evidence/review-01/runs/drawio-ui-green-01/result.json>). These are new byte-exact copies of consumed receipts, not new launches.


### Other original report02 diagnoses are not blanket platform waivers

- Original stdout32050 bytes/SHA256 `9d87e0a85ca1c0253dd5936dbd3a7baad5323e954217f20119304f7b6268afd6` was read without modifying its historical root. Plugin manager failure is actual code extension metadata bounds `{X:247 Y:481 W:0 H:0}` outside760x480 ([original test](<native/app/app_test.go#L1238-L1263>)); layout/render-specific investigation and a later real affected check remain, not a font diagnosis or task5 path defect.

- Original issue-history205-report fixture expected5 omitted but observed6 because native non-Windows package-store initialization reports a real unavailable-store issue. The original assertion is exact message/count ([test](<native/app/app_test.go#L1318-L1333>)); source-confirmed accounting is not proof that autosave is correct or permission to suppress the startup issue.

- Original Mindmap preview/split first-warning failure at [line476](<native/app/mindmap_tags_app_test.go#L449-L486>) checks six independent values together: focus, warnings, exact source/history, app guidance and zero Issues. Raw warnings[] alone neither proves focus loss nor stale guidance; the same startup issue can violate zero-Issues, and a separate actual focus/guidance diagnostic remains required. The original output cannot isolate those branches. No monkeypatched platform or weaker test was used.

- Absence of `text will not show` remains only a negative log veto. Real physical MyGo/Pango/input/backend/Windows behavior was not exercised by these Linux/CGO0 receipts. Original report02 index panic and later original picker/full-suite coverage remain FAILED/unknown despite the new selected original panic-case PASS.


### Evidence preservation and next concrete work

- Corrected read-only collection02 is36873 bytes/SHA256 `f59bfc331e713caf348fe9e65ff1878943fa3c0b3a011a4f5ee37d308441fc65`. Collection01 is preserved with its RUN-header subtest-count error;02 explicitly excludes slash subtest names. No primary receipt was edited or product command repeated for that correction.

- Box incoming-transfer draft01 printed a truncated SHA field and was not adopted; the new incoming02 corrected only those three characters and exactly matches the full14017-byte local incoming report hash. Both drafts remain recorded; no semantic source change or product run occurred.

- Next: Root review of the named report and complete primary/raw/closure mirrors can close task5 source+Linux evidence without falsely marking the retained Windows fixture PASS. Continue exact new experience05 and readiness Undo/focus slices in fresh labels after owner-provided source identities. Windows/IME/PE/WM_CLOSE/Job/backend/whole-app delivery gates remain PENDING; `.29` remains accepted runnable.
