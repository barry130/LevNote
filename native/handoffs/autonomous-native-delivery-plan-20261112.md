# LevNote native autonomous delivery plan

Date: 2026-11-12. Lead: DSH. Development baseline: main 77454291ce1aad7c5a617ab19d507e192154baf6.

## Authority and boundary

The latest direct user request authorizes all remaining development, ordinary technical/product decisions and continued task sequencing without repeated phase approval. Windows and other final platform acceptance are consolidated after feature convergence; their absence does not hold ordinary development. This plan supersedes the former docs-only/current-source freeze, not historical evidence.

Lead plans, assigns, reviews and accepts. At most three explicitly assigned children implement source/tests/helpers and execute new product commands on the existing authorized main box; children must not delegate. Local product implementation, tests, vet, builds, runs and static-acceptance fallback remain prohibited. The local checkout is for source reading, planning/handoffs, byte-exact mirrors of box-authored output for review, and Git delivery, not an implementation lane. CNB and GitHub main are the existing delivery channels; no new branch.

## Protected historical artifacts

The closed original Linux invocation remains FAILED, not a pass: Go1/wrapper2, no Go timeout, cleanup/postguards clear, empty closedFiles indexing panic after the recorded failure headers. That invocation and its capture paths are consumed; never relaunch them.

Original prepare02 files, the guarded original source/toolchain snapshot, all raw captures and closed reports, accepted .29 package and older accepted release directories stay immutable. New working-tree edits and new development work/evidence paths are authorized and distinctly identified. Existing Windows assertions may not be removed, weakened or skipped simply to make Linux green. A platform limitation is reported separately from a product defect.

## Deliverable scope

Converge the existing Go1.27.3 + MyGo native Surface application rather than inventing an unlimited feature backlog:

1. Reliable open/new/save/save-as/close/reopen, multi-file order, tabs, workspace navigation and recovery. Preserve untouched bytes, unknown format metadata, readonly state, encoding/BOM, draft/undo and explicit conflict/failure behavior.
2. Useful native Markdown, code text, mindmap and draw.io workflows through their implemented editor/view contracts. Every supported command must work; missing/limited capabilities must be visible and have a usable safe source/byte fallback.
3. Coherent keyboard/focus/composition, file picker and list navigation, Find/outline/reference navigation, minimum-window layout, readable failure history and preferences. Native input primitives only; no replacement document editor or toolkit-tree introspection.
4. Bounded retention and responsive ordinary interactions. Fix concrete lifecycle/retention defects; do not claim RSS/CPU/driver/IME improvements from unit or synthetic tests.
5. Local fixed-catalog plugins/engine packages with explicit import/activation, safe ownership/join/poison handling and visible actual failures. No arbitrary executable overrides, account, network runtime, synchronization or telemetry.
6. Consistent user/developer docs, reproducible development evidence and a final platform/package acceptance record describing exactly what was executed.

## Initial parallel assignments

- native-reliability: diagnose the closed Linux failure symptom families against current source; repair verified shared reliability defects and improve deterministic new regression coverage without weakening Windows semantics. Own only named lifecycle/reference files and a separate reliability handoff; declare any extra scope before editing. Prepare and operate a new remote development validation lane, never the original guard lane.
- native-experience: continue the unfinished app-list handle/state lifetime work, then its directly related keyboard/minimum-window usability corrections. Own named list/UI files and its separate handoff; coordinate shared app state before edits. No whole-app product execution or Windows packaging in this initial assignment.
- native-readiness: independently audit actual originating requirements against current native capabilities and evidence. Read-only product source; own a new readiness report and ordered, concrete follow-up backlog. Do not turn inherited historical claims into current passes.

Shared task board descriptions supply the exact current scopes and acceptance criteria. Old named children/tasks are not automatically resumed.

## Source-confirmed convergence backlog

The initial readiness audit reports five concrete supported-workflow/bounded-work gaps. They are engineering tasks, not evidence of a passed implementation or an instruction to replicate unlimited legacy parity:

- task-8, after task-7: native-readiness implements draw.io selected vertex/edge deletion. Protect root/layer and readonly pages, preserve untouched source ranges, refuse unsupported parent/reference structures, handle incident known edges as one explicit undoable transaction, and wire actual canvas key/button entry.
- task-9, after the current list group: add Markdown preview main-document Undo/Redo user entry while preserving local draft-editor history/composition priority, Flush refusal and readonly state. Also retain that module's public List Handle across Release/dropPreviewCache, cancel obsolete focus and reset only outside a live frame; abandoned handles are a source lifecycle gap, not measured RSS.
- task-10, after task-6: native-experience adds explicit local Copy All issues, clipboard-failure disclosure, visible ordinary-history `Truncated` notices, and correct preference action labels. Keep the existing 256-item/32 MiB budget; do not infer a lost-history signal identifies one exact edit.
- task-11, after task-5: native-reliability bounds synchronous save fingerprint reads by authoritative expected encoded lengths plus one, with stat/identity revalidation and exact digest. Preserve save conflict and post-publication semantics, actual BOM/encoding lengths and visible failures.

Source origin is the existing authorized box: isolated reliability, experience and draw.io work roots, with disjoint shared-task file scopes. Native-reliability is the sole product-command owner and combines only explicitly frozen input bytes into fresh validation snapshots. The local initial unexecuted test/list drafts were declared and moved to box-authoritative work before further implementation; they are not claimed as accepted source or local execution.

Coordination commit 8897244ffa4f00f5ec51f95545e7a3ca27d37d97 is synchronized to both delivery remotes and the ordinary box clone. New source remains unaccepted until concrete review and recorded affected regression; the ordinary clone and original guarded namespace are not interchangeable.

## Development validation policy

Use the existing box ordinary clone/development environment only in a new clearly identified lane. Verify the actual Go1.27.3/vendor/telemetry-off configuration and source revision; never source ordinary development env into the protected old guard context. The historical command is not reauthorized.

Focused development regression may use explicitly named affected cases/packages and records its selection; it is not full acceptance. Preserve the initial failure and every correction result with actual command, cwd, env identity, source identity, start/end, stdout/stderr and exit/timeout/owned-cleanup outcome. Avoid repeated unrelated suites. Once a feature group converges, run one appropriate complete affected suite, then the consolidated final matrix.

Product-command owner serializes shared remote lane use. Long work is owned, timed and recorded; connector readback failure is not a Go timeout. No unrelated process killing, shared-cache deletion, OS/box replacement, credential inspection or public upload of private material.

## Review and convergence

Lead reviews concrete diffs and required child reports before accepting changes. Unimplemented requirements and verified failures receive an owner and next executable step, not an indefinite manual-permission hold. Each source group is integrated with explicit tests, limitations and dual-remote commit identity.

The final matrix includes core/format/editor/session/plugin behavior, integration paths, normal close and owned-process cleanup, packaged artifacts and actual supported platform behavior. Windows GUI/PE/C:config/WM_CLOSE/Job/physical input gates remain genuine deferred requirements; Linux, cross-compilation, hidden startup or Wine do not substitute for them.

.29 remains the latest accepted runnable until a new candidate completes its own acceptance. Development-complete, command-pass, package-created and application-deliverable are separate statuses. Do not declare the whole application deliverable while a required critical workflow or final platform gate remains unverified.
