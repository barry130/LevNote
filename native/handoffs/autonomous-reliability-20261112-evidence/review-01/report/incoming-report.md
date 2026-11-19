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
