# Autonomous native experience · task-6 · report06 · 2026-11-13

## Current status and authority

**The frozen revision06 passed the original minimum-window regression and the unchanged original 25-test affected union on Linux. This is finite source/test convergence, not whole-application or Windows delivery acceptance.** Task6 remains in progress until lead closes this report and exact-byte delivery. Task10 has not been started; its App/settings/history work requires the lead's fresh assignment after this handoff.

Implementation owner: `9e20bc49-fddb-490b-a752-c3b11da86ccc` (historical display name native-experience). Sole Go/gofmt/test/vet/build/run owner: `a753a04c-1383-4870-877c-e70639eb0422` (native-reliability). Lead owns review, integration and acceptance. The author executed **zero product commands**. Metadata-only Python audits do not compile, import or execute the application. No child delegation, Git commit/push, alternate box, vendor/private-state change or package activation accompanied this work.

This is a **new report version**, not a rewritten old evidence namespace. [Report04](</home/user/levnote-dev-experience-20261112-01/report-only-04/native/handoffs/autonomous-experience-20261112.md>) remains `19105 B`, SHA256 `5f16c8c368b043475bc3fab36217bbbd72de8748ce0751918d704033edfe55aa`. Its pending-validation and guarded-structural-Key wording is historical, superseded here. The original failed calls, snapshots, manifests, READY files and .29 package remain unchanged.

## Source lineage and origin disclosure

Authoritative source: [revision06](</home/user/levnote-dev-experience-20261112-01/revision-06-source/>), derived from [revision05](</home/user/levnote-dev-experience-20261112-01/revision-05-source/>), not from F. Baseline is `8897244ffa4f00f5ec51f95545e7a3ca27d37d97`, from the ordinary authorized box clone. This development lineage is not the separately synchronized CI-pause commit and does not activate .30.

Initial unexecuted F drafts existed before the box-only correction. Their [identity inventory](</home/user/levnote-dev-experience-20261112-01/provenance/unexecuted-local-drafts.json>), [draft patch](</home/user/levnote-dev-experience-20261112-01/provenance/unexecuted-local-draft.patch>), [origin declaration](</home/user/levnote-dev-experience-20261112-01/provenance/source-origin.txt>), [lifetime-helper draft](</home/user/levnote-dev-experience-20261112-01/provenance/local-draft-list_lifetime.go>) and [Tree-test draft](</home/user/levnote-dev-experience-20261112-01/provenance/local-draft-list_handle_lifetime_test.go>) are preserved. Original View bytes were `24252 B / a102bd5266bcf35296ba5973c2856a8211b1118ef7068f3f2470502a0d8fb935`; their later box-derived lineage is not relabeled box-first. Workspace/Outline and combined regressions were box-authored. Subsequent F updates were authorized original-byte mirrors, never continued semantic implementation.

Revision01/02/03 and their failed runs remain immutable. Revision02 added the actual captured Buffer-version guard. Revision03 made only the authorized compatibility/fixture corrections: `Tester.Image()` instead of nonexistent `Tester.Render()`, and `a.searchRoot = workspace` for the synthetic acceptance fixture. The standalone Tree workflow oracle remains **7900 B**, SHA256 `ba98054601b1bfd421d1fe721cf8ea6cb97a01081f89a073507f17e3d293d5e5`.

The [03→05 delta](</home/user/levnote-dev-experience-20261112-01/provenance/task6-freeze-05/revision-03-to-05.patch>) is `2059 B / 50b604f7210c65ab2f82967eff52504b76fb87a976a07ca807adaa433293e334`: only the three captured structural-Key corrections in View/Search/Outline and the bounded scanner Name-sort/import. [Freeze05](</home/user/levnote-dev-experience-20261112-01/provenance/task6-freeze-05/source-inputs.json>) is `3036 B / 74ec096142fd683f4deeea41c59c66e2697accb068ed8c48c448e1c1065a74e7`.

[Freeze06](</home/user/levnote-dev-experience-20261112-01/provenance/task6-freeze-06/source-inputs.json>) is `4605 B / f864c426b7d200533641839fea12b4907241050778ba46a4470256773fbbb2dd`, created `2026-11-12T12:29:29.755167+00:00`. Its historical `FROZEN_UNFORMATTED_UNEXECUTED_NOT_ACCEPTANCE` status is not rewritten after later execution. The [05→06 patch](</home/user/levnote-dev-experience-20261112-01/provenance/task6-freeze-06/revision-05-to-06.patch>) is `517 B / a24f2833b397deccf45c308c6ddd3b743fba4fdb253419fd86a19b130405a64e`.

The [2026-11-13 independent input receipt](</home/user/levnote-dev-experience-20261112-01/coordination/resume-06-input-review-20261113-01.json>) is `5490 B / c3aef40fb2155996f61343f10e7c709deeb3fea7df262157b131679e8fac3206`. Its metadata-only helper verified all **5967 source files**, identical file sets, every nonchanged byte exact05, and the exact one-line replacement. Whole-source totals are **80238552 B** for05 and **80238567 B** for06. All **533 test files** are exact03/05; the diagnostic observer is excluded. These inventories are input sizes, not measured process memory.

## Implemented invariants and justified final corrections

- Tree/Search/Outline retain their comparable public `ui.Handle`, not the whole old ListState, private references or per-row Handles. Semantic reset preserves only Handle.
- Obsolete pending focus is canceled immediately. Already-built Lists keep live state/callback pointers through late layout; full reset is deferred to the next safe App.View prebuild. A later deliberate focus/reveal survives that drain.
- **Structural Keys retain captured identities until the built frame ends.** Valid Tree rows return captured Path, Search rows return captured `searchRowKey`, Outline rows return captured OutlineID, even after owner invalidation during construction. Out-of-range fallback is integer index, distinct from valid string/struct keys. Stale **content, selection publication, submit and actions**, not structural uniqueness, are suppressed by current-owner guards.
- Tree guards Root/treeRoot/treeGeneration. Search guards visibility/plugins/searchID/searchRoot/current Root. Outline guards tab/TabID/adapter/factory generation/path/mode/Buffer/controller generation/snapshot version and actual Buffer version. Existing Flush revalidation remains intact.
- Different-root Tree resets stale selection/scroll; same-root scans retain expansion and anchor behavior. New Search retains Handle but resets old selection/summary/action state. Collapse cancels obsolete focus and stops building while preserving live results/position.
- Outline detaches app ownership before captured Release, allowing legitimate reentrancy. A nonexistent old source/controller is not redundantly released, so newer deliberate focus survives. Same-controller reparses preserve stable selected IDs. Shutdown clears app-owned List callbacks, selected pointers and hooks; this is not a measured heap/RSS improvement.
- Scanner sorts **only the already retained, bounded `ReadDir(limit+1)` subset** by Name before the unchanged directories-first traversal. Admission/truncation subset, recursion, cancellation/errors, budgets, watches and generation delivery do not change. This stabilizes the real scan path, not a fixture-specific View workaround.

Inspection-only Outline Snapshot/Resolve must not mutate tab ownership, mount views, change selection or flush drafts. The earlier side-effectful callback candidate is closed as **unproved**, not a contract-conforming behavioral RED and not a production fix. Legitimate Release/Flush/factory-generation transitions remain covered by the public App.View/Tester oracle. No observer replaces production callbacks or injects focus/selection merely to obtain GREEN.

## Actual failure chain, preserved rather than replayed

| Actual isolated run | Observed result and limitation |
|---|---|
| [baseline Tree RED01](</home/user/levnote-dev-reliability-20261112-01/runs/experience-tree-baseline-red-01/evidence/>) | Pure baseline plus original7900 Tree oracle: `End selected -1`; genuine behavior RED. |
| [current GREEN01](</home/user/levnote-dev-reliability-20261112-01/runs/experience-tree-current-green-01/evidence/>) | Compile failure: `tester.Render undefined`. Never behavior GREEN. |
| [current GREEN02](</home/user/levnote-dev-reliability-20261112-01/runs/experience-tree-current-green-02/evidence/>) | Go21905 exit1/wrapper2; unchanged03 Tree `End selected -1`. Failed namespace not rerun. |
| [affected01](</home/user/levnote-dev-reliability-20261112-01/runs/experience-list-affected-01/evidence/>) | Go22596 exit1/wrapper2; 11 top-level started,9PASS/2FAIL,14 unstarted after `ui: two elements under one parent have the key ""` Outline panic. Not 25-test completion. |
| [public-input observer05](</home/user/levnote-dev-reliability-20261112-01/runs/experience-tree-public-input-observer-05/evidence/>) | Go29020 exit1/wrapper2. Literal00 had Find=true but clipped rectangle `{1,209,0,0}` outside List `{1,93,226,80.5}`; first model04/last29. Click/End produced no focus, selected binding writes or notifications; Enter not reached. This is **not** proof of dropped App publication after a successful bound write. |
| [final05 focused](</home/user/levnote-dev-reliability-20261112-01/runs/experience-list-final05-focused-01/evidence/>) | Go30369 exit0/wrapper0: original Tree and Outline-release tests both PASS. Actual nine source inputs/maps independently closed. |
| [final05 original25](</home/user/levnote-dev-reliability-20261112-01/runs/experience-list-final05-affected-01/evidence/>) | Go31061 exit1/wrapper2: all25 started,24PASS/1FAIL;20 nestedPASS/2FAIL. Only minimum-window light/dark source editor was clipped to `{X:228,Y:477,W:0,H:0}` at original line376. No panic/timeout/cleanup/remaining group. Later action/render branches were not reached in05. |

Final05 failure raw is `11516 B / cedc8eaa472b51e81b30ec9333c36aeb9d0e0203f224c9ef5737f30304fc6583`; numeric `4003 B / 84490fa4a9374201e17889c8f38672cb82fed0253f2650741232880d1d3faa53`; wrapper result `8374 B / a0e1e93520122ef06a1f41db0cb0dd1ada7f54e9376820bc243de5032bfa2ec0`. Its three input maps are each `199696 B / d7a413c025e6eafccd497ea4ff5430b61bfae85d91c32e5eee6bd5042ac7f1b4`, **1194 files /17000636 B**. The observer's different input total must not be substituted.

This history is disclosed, not a retrospective chronological RedBeforeGreen claim for the earlier implementation. The final06 minimum-window correction follows the actual unchanged-oracle05 behavior failure.

## The single revision06 sizing correction

At760×480, preferred Results height220 remained nonshrinkable due to `.Shrink(0)`. Find-only editor started atY257; combined Results moved it toY477, beyond main/status boundary450 and clipped it to zero. Public `Context.Size()` is the window content size, not remaining parent height. The source editor already uses `.Grow(1).MinHeight(80)`.

Only [workspace.go:143–149](</home/user/levnote-dev-experience-20261112-01/revision-06-source/native/app/workspace.go#L143-L149>) changes:

```go
panelHeight := float32(230)
if _, height := c.Size(); height > 0 && height < 600 {
    // Leave source input usable beside Find at the 760x480 minimum.
    panelHeight = min(panelHeight, max(float32(110), height-260))
}
ui.Column(c).Height(panelHeight).Shrink(1).MinHeight(110).Border(1, c.Theme().Border).Children(func() {
```

The exact delta is `.Shrink(0)` → `.Shrink(1).MinHeight(110)`. Preferred230/height−260 formula, existing110 lower bound, editor/Find controls, List Keys/guards/focus/selection, all tests and assertion thresholds remain unchanged. No post-failure fixture relaxation, extra focus, selection seed or oracle alteration was used.

The unchanged [minimum-window test:349–399](</home/user/levnote-dev-experience-20261112-01/revision-06-source/native/app/list_lifetime_test.go#L349-L399>) passed light and dark: cancel/details/collapse rectangles positive, within bounds and nonoverlapping; source rectangle **W≥200,H≥64**; real cancel invocation, details exposure and collapse action reached. Exact combined Results/editor geometry was not logged; **113px Results/80px editor is not reported as measured**.

Public `Tester.Image()` and PNG encoding/close branches were reached, with two render log paths per run. Original `t.TempDir()` cleanup deleted those images; no persistent PNG is available or claimed visually reviewed. There is no rerun merely to recover a screenshot.

## Exact final nine source inputs

| Revision06 source | Bytes | SHA256 |
|---|---:|---|
| [app.go](</home/user/levnote-dev-experience-20261112-01/revision-06-source/native/app/app.go>) | 39471 | `b4e8b96ac45a336d3cc2524f513fe652f6cd638c075ad53fb590c509d09fdbde` |
| [view.go](</home/user/levnote-dev-experience-20261112-01/revision-06-source/native/app/view.go>) | 24375 | `8b702c15dfcd265821ec10fe7a8f52927f6ffe667a349ff04b8b281c67ba3b77` |
| [workspace.go](</home/user/levnote-dev-experience-20261112-01/revision-06-source/native/app/workspace.go>) | 8339 | `d9f458f1cb4d597932b58161f546f1e43874aa5eeb9bf25eb9c47ca40519177a` |
| [outline.go](</home/user/levnote-dev-experience-20261112-01/revision-06-source/native/app/outline.go>) | 15523 | `c75435237f5d9eb9b466ed8b755cc8c22818f5dc4a6a57ac2045ae7136cad8cf` |
| [list_lifetime.go](</home/user/levnote-dev-experience-20261112-01/revision-06-source/native/app/list_lifetime.go>) | 2345 | `14c4d6cc436a6633ca5055fb18b6bc5b63703ebce0d7c8a817092ea5bdf24bb5` |
| [list_lifetime_test.go](</home/user/levnote-dev-experience-20261112-01/revision-06-source/native/app/list_lifetime_test.go>) | 17378 | `982e3ec121e9d5c020df3e9d4661748e894186cf91976de118d32be47da120a2` |
| [list_handle_lifetime_test.go](</home/user/levnote-dev-experience-20261112-01/revision-06-source/native/app/list_handle_lifetime_test.go>) | 7900 | `ba98054601b1bfd421d1fe721cf8ea6cb97a01081f89a073507f17e3d293d5e5` |
| [acceptance_test.go](</home/user/levnote-dev-experience-20261112-01/revision-06-source/native/app/acceptance_test.go>) | 25229 | `0d54cd2308b545d98cb3f2d81b35ef367c310b66658e2b3ea4672dd7edd6e362` |
| [workers.go](</home/user/levnote-dev-experience-20261112-01/revision-06-source/native/app/workers.go>) | 8225 | `73c65d6aedc8b517c808314aae439f38c1c4137b013c2ca2db885e1fc88d9b83` |

Only workspace differs05 (`8324 B / 473f732d2e4d1e9950be852fd6eeb176cbc4188e0f60e1f345ec31ad94ea2d2e`). Eight other scoped files and all533 tests are exact05/03. The nine final files total **148785 B**.

## Actual revision06 commands and execution identities

[READY06](</home/user/levnote-dev-experience-20261112-01/coordination/final06-ready.txt>) remains `5247 B / c7b940b4afb6401076f41ea8654debfdb3367dca5b33df35521e1051c872069d`. Status-first recovery found no prior06 input/intent/run/PID; no create-only freezer or consumed05 failure was repeated.

Soleowner [format06](</home/user/levnote-dev-reliability-20261112-01/runs/experience-list-final06-format-01/evidence/result.json>): actual gofmt2017/PGID2017 exit0/wrapper0, `2026-11-13T02:19:37.627719Z`–`02:19:37.692461Z`, only `gofmt -w app/workspace.go` in a new isolated snapshot. **Zero byte changes**; entire1194 formatted snapshot remains exact original06.

Soleowner [focused06](</home/user/levnote-dev-reliability-20261112-01/runs/experience-list-final06-focused-01/evidence/result.json>): actual Go2058/PGID2058 exit0/wrapper0, `2026-11-13T02:21:04.522909Z`–`02:21:43.583137Z`, elapsed `39.06659982500423s`, wrapper180s:

```text
go test -v -count=1 -timeout=120s -run ^TestWorkspaceResultsMinimumWindowControls$ ./app
```

Raw10 lines: **1 top-level +2 light/dark subtests PASS**, app0.304s. Only after this result fully closed did the soleowner run [affected06](</home/user/levnote-dev-reliability-20261112-01/runs/experience-list-final06-affected-01/evidence/result.json>): actual Go2782/PGID2782 exit0/wrapper0, `2026-11-13T02:24:51.043734Z`–`02:25:17.577309Z`, elapsed `26.535024564000196s`, wrapper240s. Exact argv is captured in its [numeric command](</home/user/levnote-dev-reliability-20261112-01/runs/experience-list-final06-affected-01/evidence/05-product-command.json>); `-v -count=1 -timeout=180s -run <original25 anchored selector> ./app`, **byte-exact selector to actual05 Go31061**. Raw138 lines: **all25 top-level tests started and PASS, all22 nested tests PASS**, app1.088s. There are no unstarted selected tests, panic, timeout, owned cleanup or remaining owned process group.

A soleowner prelaunch metadata-index read failed before any affected06 input/run/intent/PID was created; absence was independently checked before preparation. It is not an extra product run. Actual affected06 product execution is only Go2782. Old failed namespaces remain intact.

Toolchain: Go1.27.3 Linux/amd64, CGO0, `GOFLAGS=-mod=vendor`, `GOPROXY=off`, `GOSUMDB=off`, `GOTOOLCHAIN=local`, `GOTELEMETRY=off`, `GOWORK=off`, `GOVCS=*:off`; HOME/XDG/cache/tmp isolated, no old guard sourcing. Actual runner [dev-remote-validation-20261112-03.py](</home/user/levnote-dev-reliability-20261112-01/tools/dev-remote-validation-20261112-03.py>) is `17964 B / 6f3dd79076e9f284691f24be1a8e3b0c91d57d9c0ed5766d07cc87445acdff89`; ordinary [env input](</home/user/levnote-dev-env.sh>) is `175 B / 9b6b8b1aad396bd0add1d2f469de218ae3df99079b72da1b21e79f99f0b51e47`.

## Independent primary evidence closure

The author read focused raw10/numeric64/result142 and affected raw138/numeric64/result142 in full. Metadata-only [final06 audit](</home/user/levnote-dev-experience-20261112-01/coordination/check-final06-evidence.py>) exited0 for each actual result, checking intent/start/PID/PGID/argv/cwd, outcome, streams, runner/env identities, all source-map roots/files, every actual caller input and run snapshot byte against frozen06, nine pins and original7900. The lead and soleowner also independently audited the primary evidence.

| Evidence | Focused06 bytes / SHA256 | Original25 affected06 bytes / SHA256 |
|---|---|---|
| [raw stdout](</home/user/levnote-dev-reliability-20261112-01/runs/experience-list-final06-focused-01/evidence/05-product-command.stdout.bin>) · [union raw](</home/user/levnote-dev-reliability-20261112-01/runs/experience-list-final06-affected-01/evidence/05-product-command.stdout.bin>) | `939 / c5f67322fec093f1ce8cb6ca9a602e7ba220997d9126bbf1ff249f614e6b8bf0` | `11812 / 5ad7121a72fca05ec734e437fe2fdfc02c52b104a1affbcb111c398d790c20e1` |
| [numeric](</home/user/levnote-dev-reliability-20261112-01/runs/experience-list-final06-focused-01/evidence/05-product-command.json>) · [union numeric](</home/user/levnote-dev-reliability-20261112-01/runs/experience-list-final06-affected-01/evidence/05-product-command.json>) | `2714 / 36225dfd49227d1fcad2905f699533a581b7b84fdaf22b7c074dfd5d69dce22c` | `4003 / 59eadfbe643c71d45048ce8a78a918799fceccf18e980c8a1e900c272f2def31` |
| [result](</home/user/levnote-dev-reliability-20261112-01/runs/experience-list-final06-focused-01/evidence/result.json>) · [union result](</home/user/levnote-dev-reliability-20261112-01/runs/experience-list-final06-affected-01/evidence/result.json>) | `5816 / 68c36c8d02b4c4915680f652b2d4e68a72092d89ee57f3f21047eeada23aa6a7` | `8388 / 571a8983002ee89a41203ddb34ac8af3ec17a7c476ac9472f516eb380cef9619` |
| [expected](</home/user/levnote-dev-reliability-20261112-01/inputs/experience-list-final06-focused-01/expected-manifest.json>) · [union expected](</home/user/levnote-dev-reliability-20261112-01/inputs/experience-list-final06-affected-01/expected-manifest.json>) | `202703 / ef86a56bd0d035060a9ff403df4e0afc29c6757b2fe3244b62cbbe275a44a73b` | `202704 / 9a0580ed2abcd5420c18c97cba116eb49c2aa375cfca9812549f3a69adf6b4a9` |

Both actual source-before/source-after/snapshot-after triples are individually **199696 B**, SHA256 **`b16c7cfa0e8ddfd29f77104b744cc6bb58fd89b69330e1fc59bd5016a3473efb`**, with identical roots and all **1194 files /17000651 B**. Eight other scoped inputs and every test are exact actual05 RED; only the specified workspace line differs. Empty stderr hashes are `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`.

Tree/Search virtualization allocation logs in the union are bounded public-frame observations, not whole-process or Windows memory measurements. No manual screenshot, physical keyboard/IME or operating-system dialog result is inferred from Tester.

## Exact-byte review delivery and remaining work

A separate create-only delivery06 package carries the original nine frozen source Buffers, this new report, primary numeric/raw/result evidence and pinned manifests/origin records. It does not regenerate Go text, reformat it, overwrite original reports or execute product code. Archive/per-entry/manifest identities and fresh destination identity checks must be verified by the integrating lead; preparation of a package is not a claim that F has already been mirrored. Original03/04 report copies and all05 failed evidence remain preserved.

Only task6's finite source/Linux regressions are converged here. Whole-app tests/vet/build/integration, Windows GUI/PE/config/WM_CLOSE/Job behavior, physical Windows IME/OS-dialog preedit and measured whole-process memory acceptance remain later unified work. The read-only Outline candidate closure does not establish safety of every possible future side-effectful factory. Native Markdown task9 remains separate. Task10 issues export/visible clipboard failure, current-buffer history-loss notification and action-label corrections are pending fresh scope and must not be represented as already implemented.
