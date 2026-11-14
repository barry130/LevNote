# Independent Standards addendum — frozen native corrections

Task: task-22. Reviewer: standards-review. Date: 2026-11-14. Standards axis only; Spec remains independent. This addendum does not modify the [original report](<F:/AIproject/LevNote/native/handoffs/autonomous-final-standards-review-20261113.md>) or grant application/package acceptance.

## Findings — under 400 words

### S1 disposition — closed for this frozen draw.io path

The new [immediate Redo barrier](<F:/AIproject/LevNote/native/modules/graphview/input.go#L263-L279>) rejects changed Buffer, release/page ownership, readonly, group, version or selection before `ensureSynced`. A synchronous Release therefore cannot rehydrate the old parsed cache, satisfying the [ownership rule](<F:/AIproject/LevNote/native/ARCHITECTURE.md#L610-L617>) and [Release contract](<F:/AIproject/LevNote/native/ARCHITECTURE.md#L768-L770>). The new [public regression](<F:/AIproject/LevNote/native/modules/graphview/drawio_redo_release_test.go#L13-L151>) performs genuine Delete→Undo→Redo through Ctrl+Y and native command, Flushes/releases the old View, mounts a replacement, changes retired XML, and verifies no obsolete report, receipt mutation or focus theft. Both routes PASS in the [saved full-suite stream](<F:/AIproject/LevNote/.localtools/final-native-20261114/test-01/stdout.raw#L10753-L10770>). This closes the original retained-baseline violation, not a newly introduced regression; no broader graph-lifecycle guarantee is claimed.

### Hard violations — none newly identified in the correction review

[Issues export](<F:/AIproject/LevNote/native/app/issues_export.go#L29-L177>) now separates current export identity/time, bounded Unicode excerpts, family omissions, record truncation and historical losses while retaining originals and explicit writer failure. Its private excerpt/family helpers implement real bounded work, not speculative abstraction. [Save retry](<F:/AIproject/LevNote/native/core/document/atomic_windows.go#L41-L74>) rechecks original expectations after each delay; [publication handling](<F:/AIproject/LevNote/native/app/app.go#L1002-L1021>) discloses actual target/metadata state without success. These agree with [preservation and explicit-failure rules](<F:/AIproject/LevNote/native/AGENTS.md#L585-L591>). The unavoidable check-to-OS race and Windows directory-sync no-op remain explicit boundaries, not newly solved guarantees.

### H1 — P3 heuristic retained, non-blocking

Retention/search headers and detail assembly still overlap between [export](<F:/AIproject/LevNote/native/app/issues_export.go#L98-L134>) and [panel](<F:/AIproject/LevNote/native/app/view.go#L536-L559>): possible **Duplicated Code**, with future text-drift cost. This is judgement, not a mandatory rule or current mismatch. Different bounded-copy and full-text-display responsibilities justify not forcing a shared abstraction merely for reuse. No additional Fowler smell warrants a finding.

## Identity and scope appendix

- Fixed baseline resolves to `8897244ffa4f00f5ec51f95545e7a3ca27d37d97`; HEAD independently rechecked as `8d4f43239dc30bfeb083dafeec7a02ed0f2da2d3`. Commits are `79bd88e51848444187ebf32ef589580dd7487ab1` and `8d4f43239dc30bfeb083dafeec7a02ed0f2da2d3`.
- `git diff 8897244ffa4f00f5ec51f95545e7a3ca27d37d97...HEAD` contains documentation and paused CI only. The scope additionally includes `git diff HEAD`, the ordinary tracked working-tree comparison to baseline, and `git ls-files --others --exclude-standard` across all six native product roots and live helpers. New untracked Redo and Issues tests/implementation were inspected, not omitted because three-dot excludes them.
- The [freeze manifest](<F:/AIproject/LevNote/.localtools/final-native-20261114/source-frozen-02/source-frozen.json>) declares 1,211 files and aggregate `874bc1fd23a3d02e432c9c6305842ee540afdca1cdad23692905da80d01b2bdf`. This reviewer independently matched every listed file's length and SHA-256 to the live tree: zero mismatches, command exit 0. The aggregate is the manifest's declared value, not a separately reconstructed aggregate algorithm. Manifest-file SHA-256: `5315612dc8654f158dae1a4aff7a3ab28a4d7c79180dcb09349b7034ad58aedc`.
- Inherits the original report's completed tracked/untracked semantic coverage; this is a finite correction addendum, not a fresh exhaustive semantic reread of all 1,211 files or old handoffs. Re-read the complete new Issues implementation/budget test and Redo regression, immediate Redo hunk, Document save/verification and atomic publication paths, Windows retry, app partial-publication handling and relevant rules. Task-15 completed scope and [saved correction record](<F:/AIproject/LevNote/native/handoffs/autonomous-windows-save-conflict-correction-20261113.md>) were cross-checked, not taken as renewed execution authority.

| Frozen correction/support file | SHA-256 |
|---|---|
| [Graph input](<F:/AIproject/LevNote/native/modules/graphview/input.go>) | `4f696abfcfbc5d8c2b9fa4812441f38a1a794ee1532e8157d3f8d7ad5eff008d` |
| [Public Redo regression](<F:/AIproject/LevNote/native/modules/graphview/drawio_redo_release_test.go>) | `c4db22dec9cc7b617e769be551107350b80ca0494271aaeb0cfd84bcb8ac8f18` |
| [Issues export](<F:/AIproject/LevNote/native/app/issues_export.go>) | `fd7a4b4a2d9906230291424d9c50e62d8f0b439a928e10465ea483b8754d4d9b` |
| [Issues budget regression](<F:/AIproject/LevNote/native/app/issues_export_budget_test.go>) | `2c2a12bbbb301c1c9cfb7d4849fab9446d5d70e9c0f19d19be6a68ce417fee25` |
| [Document save](<F:/AIproject/LevNote/native/core/document/document.go>) | `86d10215c638d44f74fcebe20dc187e5eedcb8ee9388cfe78061818d5ed39c38` |
| [Windows atomic helper](<F:/AIproject/LevNote/native/core/document/atomic_windows.go>) | `f381a70c61e5aaed52610ea023208178d25c451b80f0e8eb5c6cde5c5d8a4618` |
| [App save orchestration](<F:/AIproject/LevNote/native/app/app.go>) | `47e48c3ae487056c78cc22a47ae469e5e45733c08bbc8e8a46227ce6eec1f7c2` |

Original report remains byte-identical, SHA-256 `9be7de036246174fe7a3a930ad0afa8201717b4dbb412680bd3206db1cd38901`.

## Authority and evidence limitations appendix

- Applicable standards include [root instructions](<F:/AIproject/LevNote/AGENTS.md#L1-L65>), the more specific [native authority](<F:/AIproject/LevNote/native/AGENTS.md#L27-L43>) and [local development constraints](<F:/AIproject/LevNote/native/AGENTS.md#L566-L598>), [CONTRIBUTING](<F:/AIproject/LevNote/CONTRIBUTING.md#L77-L110>), [architecture](<F:/AIproject/LevNote/native/ARCHITECTURE.md>), [format API](<F:/AIproject/LevNote/native/formats/API.md#L1-L16>) and the [migration ADR](<F:/AIproject/LevNote/docs/decisions/0063-native-migration.md>). Native/local authority overrides historical Tauri/box-specific prescriptions. Fowler's twelve baseline smells are judgement calls, subordinate to repository rules; necessary immutable receipt tuples, platform error outcomes and deterministic test seams are not automatically smells.
- Read [full-suite receipt](<F:/AIproject/LevNote/.localtools/final-native-20261114/test-01/result.json#L1-L50>) and independently recounted saved JSON outcomes: 3,019 PASS, 0 FAIL, 49 SKIP, exit 0, source unchanged. Checked stream SHA-256 `013b7f58da8f6005918b7005078ebc48fce4fde8087dab46a894126ac0aba56e`; stderr is empty. All eight new Issues budget/identity test groups and both public Redo routes have explicit PASS events. A first reviewer-only counter comparison exited 1 because absent zero-valued `fail` was not initialized; corrected zero-key initialization matched the unchanged stream. This was no product execution or product failure.
- [Vet receipt](<F:/AIproject/LevNote/.localtools/final-native-20261114/vet-01/result.json>) and [build receipt](<F:/AIproject/LevNote/.localtools/final-native-20261114/build-01/result.json>) each record exit 0, empty streams and unchanged same freeze. The inspected [gate](<F:/AIproject/LevNote/.localtools/gate.sh#L6-L21>) selects local Go1.27.3, six product roots, offline vendor and telemetry-off. These are saved command evidence, not reviewer-launched gates or packaged/physical GUI acceptance; 49 skips are not passes.
- No product/Go/formatter/helper execution, source/old-report modification, Git write, network/MCP action or child delegation occurred. Only this new report was written in the repository. Known original five plus new three held formatting debts remain excluded from semantic findings; no claim of format PASS is made.
- Physical Windows GUI/clipboard/IME, packaged process/close/session and measured resource acceptance remain separate. Neither synthetic Tester/writer seams, allocation-only tests nor current command passes establish those results. Future edits require new identities/review.
