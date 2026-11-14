# Independent Standards review — native convergence WIP

Task: task-17. Reviewer: standards-review. Review snapshot: 2026-11-13; report finalized 2026-11-14. Standards only; Spec findings remain separate. No source acceptance or execution certification.

## Findings — under 400 words

### S1 — P2 documented-rule violation: Redo rehydrates a released draw.io view

In the modified [Redo hunk](<F:/AIproject/LevNote/native/modules/graphview/input.go#L263-L274>), `Host.Changed()` is followed unconditionally by `ensureSynced(nil)`; the captured ownership receipt guards only the later focus request. A lawful synchronous callback can Flush/Release this view, mount a replacement, and change the retired Buffer. [Release](<F:/AIproject/LevNote/native/modules/graphview/adapter.go#L262-L269>) clears parsed caches and invalidates synchronization. [ensureSynced](<F:/AIproject/LevNote/native/modules/graphview/sync.go#L55-L71>) then captures the *new* epoch and reparses retired content, potentially reporting its parse failure through the old host ([draw.io branch](<F:/AIproject/LevNote/native/modules/graphview/sync.go#L123-L130>)).

This breaches the [Release cache contract](<F:/AIproject/LevNote/native/ARCHITECTURE.md#L768-L770>); the documented synchronous-Changed rule expressly prohibits reviving released caches ([ownership rule](<F:/AIproject/LevNote/native/ARCHITECTURE.md#L610-L617>)). The adjacent [Undo path](<F:/AIproject/LevNote/native/modules/graphview/input.go#L239-L244>) correctly rejects obsolete receipts before synchronization. Apply that immediate barrier to Redo and add its public callback-release regression. High static confidence; no executed reproduction or demonstrated content loss. **The unsafe continuation already existed at baseline: this is retained nonconformance in a modified hunk, not a newly introduced regression.**

### H1 — P3 heuristic smell: duplicated Issues presentation rules

The new [snapshot formatter](<F:/AIproject/LevNote/native/app/issues_export.go#L39-L65>) repeats the panel's retention/dropped header, search-omission header, and path/reason/detail assembly ([panel](<F:/AIproject/LevNote/native/app/view.go#L536-L559>)). Changing one disclosure now requires parallel edits, inviting copied text to diverge from visible text. Consider small shared pure formatting helpers while keeping UI rendering and clipboard publication separate. This is **Duplicated Code**, a maintainability judgement, not a mandatory repository rule or an asserted present mismatch. T05 content/output-budget compliance belongs to the separate Spec review.

## Source identity appendix

- Fixed baseline: `8897244ffa4f00f5ec51f95545e7a3ca27d37d97`.
- Reviewed HEAD, independently rechecked before finalization work: `8d4f43239dc30bfeb083dafeec7a02ed0f2da2d3`.
- Commits above baseline: `79bd88e` (native convergence/source-authority documentation), `8d4f432` (temporary CI pause).
- Semantic scope was the complete tracked working-tree comparison `git diff 8897244ffa4f00f5ec51f95545e7a3ca27d37d97 -- native AGENTS.md .gitignore`, plus live untracked product/source/test/helper files enumerated with `git ls-files --others --exclude-standard`. This is **not** a commits-only or three-dot WIP review.
- Inventory/raw Git identity evidence: [initial capture](<C:/Users/canace/AppData/Local/Temp/dsh-subprocess-FOjVcS/dsh-subprocess-32416-3-0f981684c566-stdout.log>) and [later capture](<C:/Users/canace/AppData/Local/Temp/dsh-subprocess-FOjVcS/dsh-subprocess-32416-7-796b65ccc5e0-stdout.log#L600-L636>). Finding identities below are read-only Git blob hashes, **not SHA-256**; they pin the inspected bytes even if later implementation moves the live tree.

| Inspected finding/support file | Git blob identity |
|---|---|
| [Graph input](<F:/AIproject/LevNote/native/modules/graphview/input.go>) | `6f1d912b8c6657495826a72caa6810aabb686973` |
| [Graph synchronization](<F:/AIproject/LevNote/native/modules/graphview/sync.go>) | `7399c0f9a6a8c8d3e9a3d37067049fe4c8b15599` |
| [Graph adapter](<F:/AIproject/LevNote/native/modules/graphview/adapter.go>) | `ead867bf4220776da56515258831cb0f1ac4c6af` |
| [Public deletion/Undo callback tests](<F:/AIproject/LevNote/native/modules/graphview/drawio_delete_test.go>) | `052bdce57b1ff79f5ac0ab0c99bd2da757b2422a` |
| [Issues snapshot](<F:/AIproject/LevNote/native/app/issues_export.go>) | `1d552b306388a465dcb81aca95eae56443199678` |
| [App view](<F:/AIproject/LevNote/native/app/view.go>) | `b728b50bb9fa733e3db92468a18183738dbe9268` |

S1's callback scenario is supported by the legal host lifecycle used in the existing public [Undo-release regression](<F:/AIproject/LevNote/native/modules/graphview/drawio_delete_test.go#L157-L264>), not claimed as an executed Redo test. S1 is narrowly scoped to draw.io; no generalized claim about all graph actions is made.

## Coverage and limitations appendix

- Reviewed tracked app lifecycle/list/navigation/reference/history/UI changes, document fingerprint/atomic publication/retry changes, graph input/UI and Markdown preview/lifecycle changes, changed tests, instruction/configuration and convergence-document hunks. Read all enumerated live untracked app/document/draw.io/graph/Markdown implementation and Go test bodies. Read all three historical remote-validation helpers and compared their versions without execution.
- Applicable rules were selected from [root instructions](<F:/AIproject/LevNote/AGENTS.md#L1-L65>), [native instructions](<F:/AIproject/LevNote/native/AGENTS.md#L27-L43>), [native current scope](<F:/AIproject/LevNote/native/AGENTS.md#L566-L600>), [migration ADR](<F:/AIproject/LevNote/docs/decisions/0063-native-migration.md>), [architecture](<F:/AIproject/LevNote/native/ARCHITECTURE.md>), [format contracts](<F:/AIproject/LevNote/native/formats/API.md>), [contributor guidance](<F:/AIproject/LevNote/CONTRIBUTING.md>) and [editor configuration](<F:/AIproject/LevNote/.editorconfig>). Large instruction/architecture/API files were mapped and relevant sections read, not claimed exhaustively read. Latest native/local Windows authority overrides legacy framework and historical box-only wording.
- Archived handoff payloads/backups/raw runs/packages were classified as historical evidence, not live source. The [readiness audit](<F:/AIproject/LevNote/native/handoffs/autonomous-delivery-readiness-20261112.md>) is dated historical analysis; its old gap/authority descriptions are not recast as current defects or renewed commands. The [delivery-plan amendment](<F:/AIproject/LevNote/native/handoffs/autonomous-native-delivery-plan-20261112.md#L7-L24>) supplies the current authority. Temporary CI disablement is recorded, not mistaken for native release acceptance.
- No tests, vet, build, formatter, helper execution, product launch, network/MCP action, staging, commit, push, ACL repair, or delegation was performed. Formatting/compiler/tooling-enforced issues and access/environment limitations are excluded from findings. Private deterministic seams and necessary lifecycle receipt tuples are not automatically smells.
- This report preserves the inspected old snapshot. Subsequent Issues corrections or other edits require a new review; no later patch is silently accepted here. No full-tree byte freeze, physical Windows GUI/clipboard/IME verification, package acceptance, or measured RSS/latency proof was established by this reviewer. Those are explicit limitations, not inferred rule violations.
