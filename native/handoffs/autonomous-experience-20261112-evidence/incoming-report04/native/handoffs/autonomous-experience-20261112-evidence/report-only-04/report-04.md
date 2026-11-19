# Autonomous native experience · task-6 · 2026-11-12

## Status and ownership

**Source implementation frozen; product validation not executed by this author.** This is fresh task-6, not a resumed historical picker/list or .30 stage. The implementation owner is native-experience. Native-reliability exclusively owns Go/gofmt/test/vet/render in its separately isolated box lane; lead owns review and acceptance. No child delegation, local product execution, commit, push, vendor change or package activation was performed here.

The authoritative full source for revision03 is `/home/user/levnote-dev-experience-20261112-01/revision-03-source`. Its eight scoped Go inputs are frozen. Revision01 and revision02 remain separately preserved at `/home/user/levnote-dev-experience-20261112-01/source` and `/home/user/levnote-dev-experience-20261112-01/revision-02-source`. The baseline is `8897244ffa4f00f5ec51f95545e7a3ca27d37d97` from the ordinary authorized box clone, read with hooks disabled; the old guarded run and .29 are untouched.

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

Revision01→02 changes only the Outline render buffer-version condition. Revision02→03 changes exactly two test-only lines: the synthetic acceptance fixture records its admitted root, and PNG capture uses the existing public `Tester.Image()` after the existing frame. Geometry, window sizes, all assertions and production guards are unchanged.

Revision03 inputs and diffs: [manifest03](</home/user/levnote-dev-experience-20261112-01/provenance/task6-freeze-03/source-inputs.json>) · [aggregate patch03](</home/user/levnote-dev-experience-20261112-01/provenance/task6-freeze-03/source.patch>) · [exact two-line revision02→03 delta](</home/user/levnote-dev-experience-20261112-01/provenance/task6-freeze-03/revision-02-to-03.patch>). All originals remain immutable; future formatting and runs have separately captured identities.

| Revision03 source | Bytes | SHA256 |
|---|---:|---|
| [app.go](</home/user/levnote-dev-experience-20261112-01/revision-03-source/native/app/app.go>) | 39471 | `b4e8b96ac45a336d3cc2524f513fe652f6cd638c075ad53fb590c509d09fdbde` |
| [view.go](</home/user/levnote-dev-experience-20261112-01/revision-03-source/native/app/view.go>) | 24252 | `a102bd5266bcf35296ba5973c2856a8211b1118ef7068f3f2470502a0d8fb935` |
| [workspace.go](</home/user/levnote-dev-experience-20261112-01/revision-03-source/native/app/workspace.go>) | 8220 | `3a2863c49554a4574d45d650c5d388c33d64599f428241366c700baed2e50be9` |
| [outline.go](</home/user/levnote-dev-experience-20261112-01/revision-03-source/native/app/outline.go>) | 15407 | `7caed4b54a69af411f916b42b1e5d5a9de7db7fb9480799c38d3552d2299bf3a` |
| [list_lifetime.go](</home/user/levnote-dev-experience-20261112-01/revision-03-source/native/app/list_lifetime.go>) | 2345 | `14c4d6cc436a6633ca5055fb18b6bc5b63703ebce0d7c8a817092ea5bdf24bb5` |
| [list_lifetime_test.go](</home/user/levnote-dev-experience-20261112-01/revision-03-source/native/app/list_lifetime_test.go>) | 17378 | `982e3ec121e9d5c020df3e9d4661748e894186cf91976de118d32be47da120a2` |
| [list_handle_lifetime_test.go](</home/user/levnote-dev-experience-20261112-01/revision-03-source/native/app/list_handle_lifetime_test.go>) | 7900 | `ba98054601b1bfd421d1fe721cf8ea6cb97a01081f89a073507f17e3d293d5e5` |
| [acceptance_test.go](</home/user/levnote-dev-experience-20261112-01/revision-03-source/native/app/acceptance_test.go>) | 25229 | `0d54cd2308b545d98cb3f2d81b35ef367c310b66658e2b3ea4672dd7edd6e362` |

The standalone [priority Tree workflow test](</home/user/levnote-dev-experience-20261112-01/revision-03-source/native/app/list_handle_lifetime_test.go>) remains exactly `7900 B`, SHA256 `ba98054601b1bfd421d1fe721cf8ea6cb97a01081f89a073507f17e3d293d5e5`. The eighth input is the one-line acceptance fixture correction, not broader fixture or production edits.

## Requested real validation, not claimed results

The sole reliability owner reported a real **baseline behavioral FAIL**, using pure baseline plus only the unchanged standalone workflow test: `End selected -1, want the current root last row` (app `0.636s`). Its [baseline evidence directory](</home/user/levnote-dev-reliability-20261112-01/runs/experience-tree-baseline-red-01/evidence>) has input manifest `199957 B`, SHA256 `93bb524f3819ca11e4c515fe0659add7bf07611e86b3d785a9ce8afdb935b8e7` (`1192 files / 16976016 B`). Exact PID/final raw readback remains the sole owner’s report responsibility.

The first current-source attempt is a **compilation failure, not Tree behavior GREEN**: `tester.Render undefined`. Its [original failed evidence directory](</home/user/levnote-dev-reliability-20261112-01/runs/experience-tree-current-green-01/evidence>) and formatted snapshot remain unchanged. The formatted test was `17585 B`, SHA256 `babf29d0e52cc6d56134c70978cff1eddd2255eff1d25c291e318d25e0b32e92`; input manifest `202045 B`, SHA256 `46f404eaebd16797c5721ed3c28fc13cfc9af9356f9053cdee8fa4021dc51916` (`1194 files / 17000363 B`).

Revision03 corrects only that public API call (`17379 B / a1fa56c46ce0dd9863dc77cf830fe8b6e90a195eaf2b3aeb0a0f61e6682cd634` → `17378 B / 982e3ec121e9d5c020df3e9d4661748e894186cf91976de118d32be47da120a2`) and the separately authorized fixture metadata. Fresh current GREEN02 and the 25-case union are submitted to the sole owner; neither is claimed as passed here.

Prioritize **one actual public Tree loop**: `TestTreeListHandleRootSwitchKeepsFreshKeyboardTarget`. The standalone [workflow test file](</home/user/levnote-dev-experience-20261112-01/revision-03-source/native/app/list_handle_lifetime_test.go>) references existing App/public Tester fields and can be injected alone into the named baseline for a behavioral RED, then run against the current source for GREEN. No new lifetime helper needs to be inserted merely to compile that baseline slice. Both outcomes must be reported from real captured calls. Source and test authoring preceded that request, so this is **not a chronological RedBeforeGreen claim**.

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

[populateAcceptanceLists](<F:/AIproject/LevNote/native/app/acceptance_test.go#L580-L607>) now records `a.searchRoot = workspace` immediately before synthetic SearchResults creation in revision03. Lead authorized exactly that fixture metadata line and the public PNG API correction after reading their actual sources. Revision03 was copied only after the absent destination was confirmed; no old snapshot, assertion, vendor code or production guard was weakened.

Task-9 Markdown Handle/preview history remains separate. After task-6 source handoff, task-10 revision2 is explicitly assigned next: local issues CopyAll with visible clipboard outcome; existing committed-history truncation notice with once/current-tab/reset-on-reload semantics and small-budget public triggers; settings action-label correction. Existing 256/32MiB limits, cap/composition engines, worker scheduling and activity-group policy are not reimplemented here.

Linux public Tester evidence cannot establish physical Windows IME, native OS folder-dialog preedit, Windows GUI/PE/C:config/WM_CLOSE/Job acceptance or whole-process memory improvements. No result from the old failed namespace, .29, Wine or cross-build is substituted. Source convergence continues while consolidated platform acceptance remains later work.

## Byte-exact review delivery

Freeze01’s six real Base64 tool-result chunks were decoded from this author’s own persisted session, not recreated by the model. Archive `50873 B`, SHA256 `91b48f6504c42eb48966d3878bee3351dcab8dc64cc056a1b7df8a1de8664641`, and every mirrored file matched the real box receipt. Existing target identities were checked against explicitly allowed previous SHA values and rechecked immediately before writing. The seven original frozen source files, [manifest01](<F:/AIproject/LevNote/native/handoffs/autonomous-experience-20261112-evidence/freeze-01/source-inputs.json>), [patch01](<F:/AIproject/LevNote/native/handoffs/autonomous-experience-20261112-evidence/freeze-01/source.patch>) and disclosed ancestry are available locally for review.

Latest delivery transfers complete original bytes of the three changed files, this report, and both revision02/revision03 metadata under [evidence](<F:/AIproject/LevNote/native/handoffs/autonomous-experience-20261112-evidence/>), with the same archive, per-file and previous-target identity checks. Local mirror completion is recorded in the actual transfer receipt, not inferred from this preparation statement. No local Go/gofmt/test/vet/build/render/product execution or source semantic implementation accompanies byte transport. Source handoff completion is not full-app or Windows delivery acceptance.

## Completed revision03 mirror · report-only update04

This section records the completed delivery, not an inferred transfer preparation. The exact three changed source inputs, immutable revision02/revision03 metadata and the revision03 report were mirrored to F only after all allowed previous-target SHA checks. Every write was preceded by a fresh old-identity check and followed by a byte-length/SHA readback; all eight final source inputs matched manifest03.

[Actual working-mirror receipt03](<F:/AIproject/LevNote/native/handoffs/autonomous-experience-20261112-evidence/working-mirror-receipt-03.json>): `7624 B`, SHA256 `136735d88a33052945aeacc5e2ebf5ecd81ea669501a220b3fa9c3627e95cecd`, verified `2026-11-12T08:37:52.375Z`. The real archive was `48175 B`, SHA256 `520e57570ca53d598a434f5fdbeaa477d64921b8b46d234671f9a9156acd79b4`; six original tool-result parts were decoded from this author's own persisted session only. [Manifest03](<F:/AIproject/LevNote/native/handoffs/autonomous-experience-20261112-evidence/freeze-03/source-inputs.json>) is `7681 B`, SHA256 `ccc1461d5b26a34fe4c7788b4fff7dfa9d40cd11812bdd1ae2745f28586ae71b`; [the exact two-line delta](<F:/AIproject/LevNote/native/handoffs/autonomous-experience-20261112-evidence/freeze-03/revision-02-to-03.patch>) is `917 B`, SHA256 `07c3649c54788be2c7599f102af093101dcb0f8ed9d1f92f89728bbc631815b5`.

The original report02 is preserved in [freeze02](<F:/AIproject/LevNote/native/handoffs/autonomous-experience-20261112-evidence/freeze-02/report-02.md>). The complete original report03 (`14089 B`, SHA256 `ee31ec3a5cfa5be3b83422c6848830fb407a109d7d73ba50f6188b97027c123b`) remains untouched in its revision03 box source and incoming03 archive. This report-only update04 appends documentation to those original bytes; it creates no source revision04, formatting or product execution.

## Outline post-callback candidate · read-only closure

**No contract-conforming public App.View/Tester behavioral RED was established. No production guard or source/test revision04 was created.** The extra candidate is closed as unproved, not fixed or accepted safe for every possible future implementation.

[OutlineSource's neutral contract](<F:/AIproject/LevNote/native/ui/editor/contract.go#L60-L69>) explicitly requires inspection without changing selection, mounting a view or flushing drafts. A Snapshot/Resolve stub that directly Select/Open/NewDocument-switches the active tab cannot automatically count as legal product RED, even with both Buffers at version0 and old/new captured literal IDs.

Concrete factories were read: [Markdown newOutlineSource](<F:/AIproject/LevNote/native/modules/markdownview/outline.go#L30-L31>), [Mindmap newMindmapOutlineSource](<F:/AIproject/LevNote/native/modules/graphview/outline.go#L22-L24>), [InstalledSourceOutlineFactory](<F:/AIproject/LevNote/native/modules/modules.go#L78-L101>) and [sourceview.NewOutline](<F:/AIproject/LevNote/native/modules/sourceview/outline.go#L36-L69>) only retain Buffer/copied configuration and never invoke lifecycle-capable Host callbacks. The actual CurrentOutlineItem bodies for [Markdown](<F:/AIproject/LevNote/native/modules/markdownview/outline.go#L166-L185>), [Mindmap](<F:/AIproject/LevNote/native/modules/graphview/outline.go#L209-L215>) and [scene preview](<F:/AIproject/LevNote/native/modules/graphview/scene_preview.go#L190-L196>) are queries without a tab/Buffer/lifecycle transition. Host.OpenReference/RecoverDraft/ShowSource exist, but their mere presence does not establish invocation from those query/constructor bodies.

The lack of a post-callback check in [outlineSourceFor/ensureOutlineSnapshot](<F:/AIproject/LevNote/native/app/outline.go#L58-L90>) is a source observation, not an executed reachable defect. Different-tab same-version callbacks also encounter the existing owner metadata in [outlineNavigationContextMatches](<F:/AIproject/LevNote/native/app/outline.go#L422-L437>) before row label/submission. No global claim that future side-effectful factories are safe is made. Empty/error focus already cancels pending requests and is not a new defect.

[Detailed read-only review04](<F:/AIproject/LevNote/native/handoffs/autonomous-experience-20261112-evidence/outline-callback-contract-review-04.md>) records concrete function paths, contract limits and source identity. The F reference.go belongs to task5's separately mirrored source; actual revision03 editorHost/openReference were therefore read directly on the box rather than assumed identical. One narrow box metadata call printed the full review inventory but ended in outer transport timeout; it is not labeled a successful product run.

The sole native-reliability owner has retained priority Tree GREEN02 followed by the unchanged 25-case union in its serial lane. Their results are still pending. Source convergence and the subsequently assigned task10 continue without treating this candidate closure or exact-byte delivery as whole-app/Windows/IME/PE/WM_CLOSE/Job/measured-memory acceptance.
