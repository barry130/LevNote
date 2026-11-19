# Autonomous native experience · task-6 · 2026-11-12

## Status and ownership

**Source implementation frozen; product validation not executed by this author.** This is fresh task-6, not a resumed historical picker/list or .30 stage. The implementation owner is native-experience. Native-reliability exclusively owns Go/gofmt/test/vet/render in its separately isolated box lane; lead owns review and acceptance. No child delegation, local product execution, commit, push, vendor change or package activation was performed here.

The authoritative full source for revision02 is `/home/user/levnote-dev-experience-20261112-01/revision-02-source`. Its seven scoped Go files are frozen. Revision01 remains separately preserved at `/home/user/levnote-dev-experience-20261112-01/source`. The baseline is `8897244ffa4f00f5ec51f95545e7a3ca27d37d97` from the ordinary authorized box clone, read with hooks disabled; the old guarded run and .29 are untouched.

## Origin disclosure

Initial unexecuted host drafts existed before the box-only correction. They were disclosed and saved, not relabeled as box-first work. Their original paths, lengths and SHA256 are in [unexecuted draft provenance](</home/user/levnote-dev-experience-20261112-01/provenance/unexecuted-local-drafts.json>), with the saved [draft patch](</home/user/levnote-dev-experience-20261112-01/provenance/unexecuted-local-draft.patch>) and [source origin](</home/user/levnote-dev-experience-20261112-01/provenance/source-origin.txt>). The two original helper/test draft bodies are separately preserved in [helper draft](</home/user/levnote-dev-experience-20261112-01/provenance/local-draft-list_lifetime.go>) and [test draft](</home/user/levnote-dev-experience-20261112-01/provenance/local-draft-list_handle_lifetime_test.go>).

[View](</home/user/levnote-dev-experience-20261112-01/source/native/app/view.go>) still has the original unexecuted draft's exact bytes (`24252 B`, SHA256 `a102bd5266bcf35296ba5973c2856a8211b1118ef7068f3f2470502a0d8fb935`); it is not a box-first-authored claim. App fields, the lifetime helper and the original workflow test follow that disclosed ancestry with later box edits. Workspace/Outline changes and the new combined regression file were authored on the box. After the correction there was no further host implementation; later host source changes are exact frozen-byte review mirrors only.

## Implemented invariants

- Each of Tree/Search/Outline retains exactly its comparable public `ui.Handle`, never a whole old ListState, private references, per-row handles or forced GC.
- Semantic invalidation cancels old pending focus immediately. If a List already owns the frame, its callback/pointer state survives late layout; the next App.View prebuild drains the reset. A later intentional focus/reveal survives that drain.
- Lists use a per-build local selected index, guarded Key/Label/row callbacks, and guarded app publication/submission. Old root/search ID/controller/owner/snapshot callbacks cannot act on replacement models. Revision02 additionally rejects a late outline row when the captured buffer version changed after sidebar construction.
- Different-root Tree resets selection/scroll and obsolete model rows. Same-root scans preserve expansion and anchor behavior. Long row labels are shrinkable single lines with complete path tooltips.
- An admitted search records its root. New search clears old selection/summary/callback state while retaining its Handle. Collapse cancels pending focus and stops building but preserves live results/selection/scroll. Old-root results are explicitly nonactionable, not silently erased.
- Outline detaches all app ownership before controller Release. Hide/tab/retry/error/empty transitions retain Handle and clear obsolete callback state safely. A fresh controller is not redundantly invalidated merely to create it, so later deliberate focus survives. Successful same-controller reparses retain selected stable ID. Existing Flush and navigation revalidation remain in place.
- Shutdown clears all three app-owned callback, selected-pointer, row and hook references. This is a source ownership invariant, not measured heap/RSS improvement.
- At 760×480 the workspace result header uses single-line shrinkable summary text, nonsquashed cancel/details/collapse controls, full row/path tooltips and a reduced low-height panel budget to keep source input available.

## Exact scoped source inputs

Revision01 inputs and diff: [manifest01](</home/user/levnote-dev-experience-20261112-01/provenance/task6-freeze-01/source-inputs.json>) · [patch01](</home/user/levnote-dev-experience-20261112-01/provenance/task6-freeze-01/source.patch>).

Revision02 inputs and diff: [manifest02](</home/user/levnote-dev-experience-20261112-01/provenance/task6-freeze-02/source-inputs.json>) · [patch02](</home/user/levnote-dev-experience-20261112-01/provenance/task6-freeze-02/source.patch>).

Only the Outline render version condition differs between revisions. Both are unformatted/unexecuted inputs; future formatting and integration need their own matching manifests. Revision01 evidence is not overwritten.

| Revision02 source | Bytes | SHA256 |
|---|---:|---|
| [app.go](</home/user/levnote-dev-experience-20261112-01/revision-02-source/native/app/app.go>) | 39471 | `b4e8b96ac45a336d3cc2524f513fe652f6cd638c075ad53fb590c509d09fdbde` |
| [view.go](</home/user/levnote-dev-experience-20261112-01/revision-02-source/native/app/view.go>) | 24252 | `a102bd5266bcf35296ba5973c2856a8211b1118ef7068f3f2470502a0d8fb935` |
| [workspace.go](</home/user/levnote-dev-experience-20261112-01/revision-02-source/native/app/workspace.go>) | 8220 | `3a2863c49554a4574d45d650c5d388c33d64599f428241366c700baed2e50be9` |
| [outline.go](</home/user/levnote-dev-experience-20261112-01/revision-02-source/native/app/outline.go>) | 15407 | `7caed4b54a69af411f916b42b1e5d5a9de7db7fb9480799c38d3552d2299bf3a` |
| [list_lifetime.go](</home/user/levnote-dev-experience-20261112-01/revision-02-source/native/app/list_lifetime.go>) | 2345 | `14c4d6cc436a6633ca5055fb18b6bc5b63703ebce0d7c8a817092ea5bdf24bb5` |
| [list_lifetime_test.go](</home/user/levnote-dev-experience-20261112-01/revision-02-source/native/app/list_lifetime_test.go>) | 17379 | `a1fa56c46ce0dd9863dc77cf830fe8b6e90a195eaf2b3aeb0a0f61e6682cd634` |
| [list_handle_lifetime_test.go](</home/user/levnote-dev-experience-20261112-01/revision-02-source/native/app/list_handle_lifetime_test.go>) | 7900 | `ba98054601b1bfd421d1fe721cf8ea6cb97a01081f89a073507f17e3d293d5e5` |

The seventh exact path is `native/app/list_handle_lifetime_test.go`; it is not omitted from the inventory.

## Requested real validation, not claimed results

Prioritize **one actual public Tree loop**: `TestTreeListHandleRootSwitchKeepsFreshKeyboardTarget`. The standalone [workflow test file](</home/user/levnote-dev-experience-20261112-01/revision-02-source/native/app/list_handle_lifetime_test.go>) references existing App/public Tester fields and can be injected alone into the named baseline for a behavioral RED, then run against the current source for GREEN. No new lifetime helper needs to be inserted merely to compile that baseline slice. Both outcomes must be reported from real captured calls. Source and test authoring preceded that request, so this is **not a chronological RedBeforeGreen claim**.

Retained new regression union (8 top-level tests):

```text
TestAppListHandlesSurviveSemanticResets
TestTreeListHandleRootSwitchKeepsFreshKeyboardTarget
TestSearchListHandleNewSearchRejectsStaleDelivery
TestOutlineListHandleReleaseAfterSidebarBuild
TestOutlineListHandleReentrantFlushAndRetry
TestHiddenListFocusDoesNotStealEditorInput
TestTreeRootSwitchKeepsComposingQueryInput
TestWorkspaceResultsMinimumWindowControls
```

The manifest contains an anchored exact-name selector for those eight plus 17 affected existing tests: Tree virtualization/anchor/limit, Search virtualization/replacement/issues, source and builtin Outline lifecycle/Flush/stale navigation/empty/truncated/error retry/virtualization, Find focus-leave composition, and minimum-window Find controls/reflow. The 25-case union is requested serially, not represented as executed. Real queued scanner/search callbacks are drained on the test/UI owner; the timer is a deadlock guard, not a timing-based success criterion.

The new regressions retain late release, failed/reentrant Flush, retry and empty transitions, three-list hidden/new focus, root transition during composing query input, original bytes/version/committed-history invariants, and minimum-window controls. The minimum-window test requests public Tester renders. No render files exist as accepted evidence here; the sole validation owner may retain them when that test actually runs.

## Follow-up and boundaries

[populateAcceptanceLists](<F:/AIproject/LevNote/native/app/acceptance_test.go#L580-L606>) prepares synthetic search rows but has no admitted search-root metadata. Its later capture needs the one-line fixture assignment `a.searchRoot = workspace`. That file was outside this task's write scope and has not been edited. Lead has accepted the fixture direction and will authorize a separate snapshot update after reading it; production guards were not relaxed for old fixtures. Any such follow-up requires separate frozen inputs, not a rewrite of revision01.

Task-9 Markdown Handle/preview history remains separate. After task-6 source handoff, task-10 revision2 is explicitly assigned next: local issues CopyAll with visible clipboard outcome; existing committed-history truncation notice with once/current-tab/reset-on-reload semantics and small-budget public triggers; settings action-label correction. Existing 256/32MiB limits, cap/composition engines, worker scheduling and activity-group policy are not reimplemented here.

Linux public Tester evidence cannot establish physical Windows IME, native OS folder-dialog preedit, Windows GUI/PE/C:config/WM_CLOSE/Job acceptance or whole-process memory improvements. No result from the old failed namespace, .29, Wine or cross-build is substituted. Source convergence continues while consolidated platform acceptance remains later work.
