# Native migration: current coordination

Updated2026-11-12. User requests Go + MyGo native Surface UI, project-authored
engines, local plugins, simple UI and low resource use. Since2026-11-11 the
DSH lead session is the main responsible party (user ruling); it plans,
assigns, reviews and accepts. Children execute explicitly assigned product
work only on the authorized remote box; no local fallback. Responsibility
transfer does not relax source freezes, evidence or platform gates. Root
may edit planning/handoff files. Commits/pushes to CNB+GitHub main are
user-authorized; no branches.

## Active ownership

Latest sequential stage2026-11-12 (takes precedence over historical entries):
- Interrupted review delivery reconciled: local/CNB/GitHub and the authorized
  box clone all matched 23054e7. The prepared product run root remains separate
  from that clone. Before the 2026-11-12 caller-launch request, only
  prepare-01 existed and no test/vet/guard process was present. The single
  launch request returned box-agent-unreachable without a PID receipt;
  report01 preserves that connector fault, not a Go failure or pass. The
  lead's fresh 02:52:42Z primary readback found no caller/phase/process and
  unchanged receipt/wrapper pins. The explicit manual transport resumption
  created caller PID6819 at 03:11:27Z in the original create-only path.
  That original test is now FAILED: Go PID6830 exit1, wrapper exit2,
  TimedOut=false, cleanup/postguards empty, inputs/toolchain identities equal.
  Go ran 03:13:00.727256Z to 03:21:52.160271Z within its original 900s limit.
  Complete stdout has 54 top-level/102 nested FAIL headers and terminates
  in reopen_closed_test.go:234 index-out-of-range panic; remaining suite and
  unreported picker coverage are not accepted. All failures are not assumed
  platform-only. The negative fallback-log gate does not prove a backend.
  Task-4 is closed as failed execution evidence, not product acceptance;
  report02 contains actual outcomes, report01/old evidence stay frozen.
  DSH lead has accepted child guard-evidence-review's task-3 real-receipt
  review in handoffs/remote-box-guard-prepare-01-independent-review-20261112.md.
  The remote-transfer startup failed without claiming task-4 or product work;
  do not restart it. Former task-4 executor guard-evidence-review has stopped
  all product work. Current review and report:
  handoffs/remote-box-product-validation-01-root-review-20261112.md and
  handoffs/remote-box-product-validation-02-20261112.md.
  No further test launch, vet, new run-root, source/assertion/skip/helper edit,
  packaging or .30 activation is authorized. Only closed-evidence preservation
  and documentation synchronization continue; a compatible remote Windows
  environment and separate approval are required for Windows delivery.
  The .30 Windows GUI PE/C:config/WM_CLOSE/Job delivery gates cannot be replaced
  by Linux test, cross-compilation or Wine; .29 remains latest accepted runnable.
- 2026-11-11 user ruling: the DSH lead session is the main responsible
  party; Codex-root review duties transfer here, the historical baseline
  and all freeze rules continue. Remote guard prepare01 returned
  GUARD_READY and is
  [accepted on review](handoffs/remote-box-guard-prepare-01-root-review-20261111.md)
  after delivery-fidelity fix 12b420b (git text=auto eol=lf normalization
  had rewritten the pinned inputs-02.json in transit; native/handoffs and
  mindmap testdata are now stored byte-exact via -text). Product stage is
  authorized once per phase with the reviewed guard receipt
  (da70b0421c48165f9c3db880c75cac31c59f7901f9f55de583f05b0a135e0853) and
  --activate-product-stage picker-final-app-linux-20261111 on run root
  /home/user/levnote-remote-20261111-01: single test, then vet only after
  a passed test; no retry on failure, a fresh run root plus a new review
  is required. .30 stays inactive until test/vet results and its own
  activation; Linux results prove no Windows IME or delivery.
  User-authorized commits/pushes to CNB and GitHub replace the old
  no-commit rule.
- Picker key-correction independent review is COMPLETE; reviewer released.
  Remote preparation02 is COMPLETE and accepted as an unexecuted draft:
  [root preparation review](handoffs/remote-box-preparation-20261111/root-preparation-acceptance-02.md).
  Final transfer inputs: source ZIP02, inputs-02.json and Linux wrapper02,
  pinned by transfer-files-02.json in the same new preparation directory.
  One replacement preparation child completed/released ownership after the
  new Astra task-delivery route failed; no historical child was restarted.
  The five source files/all26patches and original source/helpers/.29/old evidence
  remain frozen. DSH exclusively owns handoffs/remote-box-preflight-20261111.md.
  NEXT is controlled private transfer and separately activated remote migration
  guards on the authorized main box. No box/byte-transfer tool is connected here.
  App final test/vet and .30 remain inactive. All future product operations are
  remote-only; no local fallback. Prior ACTIVE entries below are historical only.
- Picker raw-key correction is frozen after13behavior tests PASS, source
  manifest under open-document-picker-key-correction-evidence-20261110.
  DeepSeek reviewer owns ACTIVE [narrow correction review](handoffs/open-document-picker-key-correction-review-task.md).
  Astra source ownership released; final app test/vet and `.30` still wait.
- Independent picker review found raw Left/Right can commit native preedit and
  leave app guard stale. Astra now solely owns ACTIVE [query key correction](handoffs/open-document-picker-query-key-correction-task.md)
  in picker runtime/tests. Closed initial picker evidence stays immutable; new
  correction evidence gets separate identities. Final app test/vet and `.30`
  delivery remain held. Next list-lifetime planning is secondary/read-only.
- Picker implementation is frozen after07focused union and08contrast capture.
  DeepSeek reviewer now solely owns ACTIVE [exact-source final review](handoffs/open-document-picker-final-review-task.md).
  Root [accepted .30 preparation](handoffs/open-document-picker-release30-preparation-root-review.md)
  for later activation; no context/build/host yet. Astra waits for review before
  the planned final app test/vet. Source and portable inputs are frozen.
- [.29 docs accepted](handoffs/composition-release29-docs-root-acceptance.md).
  Both original33/37 and narrow3/4 verifier exits1 remain preserved; named-file
  root diff resolves the final leading-blank-line mismatch without a third run.
  DeepSeek delivery owner now solely owns ACTIVE [.30 preparation](handoffs/open-document-picker-release30-preparation-task.md)
  in new helpers/portable README. Astra continues picker implementation/tests;
  DeepSeek reviewer is read-only. No context/build/host until source accepted.
- [.29 runnable accepted](handoffs/composition-release29-root-acceptance.md),
  package/evidence immutable. Current runnable is
  bin/levnote-native-worker-20261110.29/levnote-native.exe.
  DeepSeek delivery owner now solely owns ACTIVE [.29 six product docs](handoffs/composition-release29-docs-task.md).
  Its first saved docs check is33/37,exit1; resolve only the
  [four concrete verifier mismatches](handoffs/composition-release29-docs-narrow-correction.md)
  and preserve original outputs. No product or broad check rerun.
  Astra solely owns ACTIVE [open-document picker implementation](handoffs/open-document-picker-implementation-task.md)
  in its named app files. DeepSeek independent reviewer owns only
  [focused picker review](handoffs/open-document-picker-independent-review-task.md).
  No next build/host/resource work yet; historical delivery freezes below have ended.
- .29 initialize attempt1 failed before context/product execution because a
  required README literal crossed a line break. Delivery owner follows
  [narrow initialization correction](handoffs/composition-release29-initialize-correction.md),
  preserves attempt1, fixes only that check and performs a separately recorded
  retry. Runtime/portable inputs stay frozen; package/host budgets unchanged.
- [Composition source accepted](handoffs/active-composition-coalescing-source-root-acceptance.md)
  after final9package test/vet;4/4inputs unchanged. DeepSeek delivery owner now
  solely executes ACTIVE [.29 delivery](handoffs/composition-release29-delivery-task.md).
  Source/portable inputs frozen. Astra may finish read-only picker plan; no
  feature implementation or additional product checks before .29 closure.
- Composition independent final review found no blocker,4/4identities match.
  Astra now owns ACTIVE [one final affected validation](handoffs/active-composition-coalescing-final-validation-task.md).
  Save any picker plan checkpoint, then prioritize the single test/vet. No
  runtime edits. .29 delivery remains held pending actual final result/acceptance.
- While composition source is frozen, Astra performs only
  [read-only next open-document picker planning](handoffs/open-document-picker-next-plan-task.md).
  This does not authorize source edits; final validation waits for independent
  review. Reviewer and .29 preparation owners continue their existing scopes.
- Composition implementation is frozen after two focused passes (2runtime+2tests).
  DeepSeek `/root/read_native_handoffs_live_tab_admission_implementation_task_md`
  now owns ACTIVE [independent final review](handoffs/active-composition-coalescing-final-review-task.md).
  Astra released source ownership pending that review and final validation;
  .29 preparation continues separately, no product execution yet.
- DeepSeek `/root/read_native_handoffs_growth_release28_preparation_task_md`
  now owns [.29 delivery preparation](handoffs/composition-release29-preparation-task.md)
  in fresh helpers/portable README only. No context/build/host until source accepted.
  Astra owns composition runtime/tests; final reviewer remains read-only/idle.
- [.28 docs accepted](handoffs/growth-release28-docs-root-acceptance.md);
  delivery/docs owner released. Original docs check exit1 and missing raw log
  are disclosed; narrow correction4/4passed. Astra continues composition
  implementation; accepted .28 package/evidence remain immutable.
- [.28 runnable accepted](handoffs/growth-release28-root-acceptance.md), package
  and evidence immutable. Current native runnable is bin/levnote-native-worker-20261110.28/levnote-native.exe.
  Astra solely owns ACTIVE [composition coalescing](handoffs/active-composition-coalescing-implementation-task.md)
  in text/editor plus focused tests. Read the root refinements; no next build yet.
  DeepSeek `/root/read_native_handoffs_growth_release28_preparation_task_md`
  solely owns ACTIVE [.28 product docs](handoffs/growth-release28-docs-task.md).
  Docs owner now resolves the [narrow final correction](handoffs/growth-release28-docs-narrow-correction.md);
  original consolidated check exit1 stays disclosed; no broad/product reruns.
  Other review owner is idle; no overlapping runtime edits or product runs.
- [Growth source accepted](handoffs/document-growth-budget-source-root-acceptance.md)
  after final9package test/vet and independent review. Source/portable inputs frozen.
  DeepSeek `/root/read_native_handoffs_growth_release28_preparation_task_md` now
  solely owns ACTIVE [.28 delivery](handoffs/growth-release28-delivery-task.md).
  Astra final validation ownership is released; next IME implementation stays
  planned until .28 package/evidence closure. No concurrent product runs.
- Growth independent final source review is complete with no remaining blocker.
  Astra `/root/read_native_handoffs_tab_workflow_resume_task_md` now owns the
  ACTIVE [one final affected validation](handoffs/document-growth-final-validation-task.md).
  All11growth source/test inputs and portable README are frozen. No product
  builds/launches until root source acceptance and separate .28 activation.
  DeepSeek final reviewer and .28 preparation owner have released ownership.
- DeepSeek `/root/read_native_handoffs_live_tab_admission_implementation_task_md`
  now owns only the [next composition contract review](handoffs/active-composition-coalescing-review-task.md).
  Read-only; .28 runtime inputs remain frozen and next implementation stays planned.
- [.27 docs accepted](handoffs/file-workflow-release27-docs-root-acceptance.md).
  DeepSeek docs ownership is released; .27 package/evidence remain immutable.
- Astra `/root/read_native_handoffs_tab_workflow_resume_task_md` now solely owns
  [document growth implementation](handoffs/document-growth-budget-implementation-task.md)
  in the specified text/document/search/editor paths and narrow integration tests.
  Focused source checks only, no next package or product launches yet.
- GLM growth audit owner was interrupted after no concrete design artifact;
  no source ownership. Root's implementation task supplies the current policy.
- DeepSeek `/root/read_native_handoffs_live_tab_admission_implementation_task_md`
  now owns [read-only growth draft review](handoffs/document-growth-draft-review-task.md),
  no source/docs edits or product execution.
- Fresh DeepSeek `/root/read_native_handoffs_growth_release28_preparation_task_md`
  owns [.28 delivery preparation](handoffs/growth-release28-preparation-task.md)
  in new helper files/portable README only. No .28 product execution yet.
- Growth draft review completed; its IME Flush finding is accepted in
  [root correction](handoffs/document-growth-composition-root-correction.md)
  and assigned to Astra. DeepSeek review owner now performs the read-only
  [next active transaction audit](handoffs/active-transaction-memory-review-task.md),
  not implementation. .28 preparation is saved; awaits final source and narrow
  portable composition guidance before activation.
- [.27 runnable accepted](handoffs/file-workflow-release27-root-acceptance.md),
  package/evidence immutable. Current entry is
  bin/levnote-native-worker-20261110.27/levnote-native.exe.
- DeepSeek `/root/read_native_handoffs_live_tab_admission_implementation_task_md`
  completed source/final validation; now owns only
  [.27 product docs](handoffs/file-workflow-release27-docs-task.md).
- Astra completed .27 delivery and is idle pending next implementation design.
- GLM is refining [growth audit design](handoffs/document-growth-budget-root-review.md),
  read-only until root assigns implementation. Earlier ACTIVE entries below
  record completed stages and do not authorize further .27 execution.
- Reopen source is [accepted](handoffs/reopen-closed-file-root-acceptance.md),
  Astra has frozen/released its app files. .26 product docs are
  [accepted](handoffs/markdown-image-reference-release26-docs-root-acceptance.md).
- Fresh DeepSeek `/root/read_native_handoffs_live_tab_admission_implementation_task_md` solely
  implements [live-tab admission](handoffs/live-tab-admission-implementation-task.md)
  in app source/tests. No package or full app test/vet yet.
  The prior raster-admission owner ended its cap follow-up with empty output
  and no saved changes; do not resume it concurrently.
  [Cap source accepted](handoffs/live-tab-admission-root-acceptance.md), now
  performs the active [single combined final validation](handoffs/file-workflow-final-validation-task.md);
  all runtime/test source remains frozen. No builds/hosts yet.
- Astra `/root/read_native_handoffs_tab_workflow_resume_task_md` solely owns
  [.27 delivery preparation](handoffs/file-workflow-release27-preparation-task.md),
  new helpers and portable README only; no app changes or product execution.
  Preparation and [combined source](handoffs/file-workflow-source-root-acceptance.md)
  are now accepted; [.27 actual delivery](handoffs/file-workflow-release27-delivery-task.md)
  is ACTIVE. All runtime/test sources frozen; no extra tests/worker requests.
- GLM `/root/read_native_handoffs_binary_image_glm_task_md` has completed .26
  docs and owns [independent admission review](handoffs/live-tab-admission-independent-review-task.md),
  read-only route audit and later exact frozen-source review; no product runs.
  Final admission review complete. Now performs the read-only
  [document growth admission audit](handoffs/document-growth-budget-next-review-task.md)
  while final validation/.27 delivery use frozen source; no implementation yet.

Historical progress follows; its old ACTIVE wording does not override above.

- DeepSeek `/root/read_native_handoffs_binary_image_prerequisite_task_md`:
  Two provider failures without saved result; inactive, no ownership. Replaced
  by GLM below; do not resume both.
- GLM `/root/read_native_handoffs_binary_image_glm_task_md`:
  [Binary core accepted](handoffs/binary-core-root-acceptance.md), source frozen.
  [Path-only binary session records accepted](handoffs/binary-session-root-acceptance.md),
  source frozen. Now owns
  [image plugin composition accepted](handoffs/native-image-composition-root-acceptance.md),
  source frozen. Now performs
  [read-only image app review](handoffs/native-image-app-independent-review-task.md)
  completed. Now reviews [.23 mixed-session helpers](handoffs/native-image-release23-helper-review-task.md)
  complete. [Hidden autosave review](handoffs/native-hidden-autosave-review.md)
  delivered. [.23 product docs accepted](handoffs/native-image-release23-docs-root-acceptance.md).
  Table-column plan review complete; [.24 docs accepted](handoffs/native-reading-release24-docs-root-acceptance.md).
  Next reading review complete. Now owns
  [.25 product docs](handoffs/markdown-table-columns-release25-docs-task.md).
  Now corrects the remaining F6 wording and performs the read-only
  [image-reference lifecycle review](handoffs/image-reference-independent-review-task.md).
  [.25 docs accepted](handoffs/markdown-table-columns-release25-docs-root-acceptance.md);
  independent lifecycle review complete, no additional reachable defect.
  Now performs short read-only [closed-file plan review](handoffs/reopen-closed-file-plan-review-task.md)
  while .26 delivers. .26 product docs wait for root package acceptance.
  Closed-file plan review is complete. Now owns the bounded read-only
  [.26 metadata-repair consistency review](handoffs/markdown-image-reference-release26-repair-review-task.md).
  Repair review completed. [.26 package accepted](handoffs/markdown-image-reference-release26-root-acceptance.md).
  Now owns active [.26 product docs](handoffs/markdown-image-reference-release26-docs-task.md).
  Root requested narrow [docs corrections](handoffs/markdown-image-reference-release26-docs-review-correction.md)
  before acceptance: no fabricated attempt1 count; preserve historical sections.
  No current runtime edits or product execution. DecodeImage takes
  context, io.ReaderAt and int64 size; builtin ID is image.

- DeepSeek `/root/read_native_handoffs_release10_ui_feasibility_task_md`:
  [.21 delivery accepted](handoffs/drawio-efficiency-release21-root-acceptance.md).
  Final turn disconnected after result saved; product work already verified.
  Inactive after transport failure; no ownership. Do not repeat .21.
- DeepSeek `/root/read_native_handoffs_resource_workload_next_task_md`:
  [Source ownership accepted](handoffs/drawio-source-retention-root-acceptance.md).
  Future resource plan delivered. Inactive after two transport failures while
  starting tab work; no tab implementation was found. No file ownership.
- Astra `/root/read_native_handoffs_tab_workflow_resume_task_md`:
  [Tab workflow source accepted](handoffs/tab-workflow-root-acceptance.md),
  including root-requested GoToLine pointer-focus composition correction.
  Source/tests frozen; [image-view plan](handoffs/native-image-view-next-plan.md)
  delivered. Now sole [.22 delivery owner](handoffs/release22_delivery_resume_task.md)
  after DeepSeek transport failures. [.22 delivery accepted](handoffs/tab-workflow-release22-root-acceptance.md)
  with14/14 gates; [.22 docs accepted](handoffs/tab-workflow-release22-docs-root-acceptance.md).
  [Final matrix wrappers accepted](handoffs/matrix-wrapper-final-root-acceptance.md).
  [One fixed12 run](handoffs/matrix-fixed12-execution-task.md) finished12/12PASS,
  all owned processes ended; [resource observations accepted](handoffs/drawio-worker-resources-fixed12-root-acceptance.md).
  [Bitmap disposal accepted](handoffs/image-bitmap-lifecycle-root-acceptance.md),
  vendor/guard/patch evidence frozen.
  [Binary copy receipts accepted](handoffs/binary-copy-receipt-root-acceptance.md),
  document source frozen. Now owns
  [image app integration accepted](handoffs/native-image-app-root-acceptance.md).
  App/runtime/tests frozen with final combined evidence, no further source work.
  [Keyboard reader plan](handoffs/native-image-keyboard-next-plan.md) delivered.
  [Hidden readiness trace](handoffs/native-image-hidden-readiness-review.md)
  delivered; .23 package/normal close accepted separately.
  [Autosave and image keyboard source accepted](handoffs/native-autosave-keyboard-root-acceptance.md),
  corrected affected test/vet passed; source frozen. Now owns
  [.24 bounded delivery](handoffs/native-reading-release24-task.md) is now
  [accepted13/13](handoffs/native-reading-release24-root-acceptance.md), package/evidence frozen.
  [Table-column UI and app tests](handoffs/markdown-table-columns-ui-task.md)
  [implemented and accepted](handoffs/markdown-table-columns-root-acceptance.md);
  final affected test/vet passed with11/11source identities unchanged.
  [.25 delivery accepted13/13](handoffs/markdown-table-columns-release25-root-acceptance.md).
  Now owns [image-reference UI/app implementation](handoffs/markdown-image-reference-ui-task.md)
  under [root decision](handoffs/markdown-image-reference-root-decision.md).
  Focused implementation frozen; now runs the single
  [final affected validation](handoffs/markdown-image-reference-final-validation-task.md).
  Final affected test/vet passed; now performs read-only
  [closed-file workflow planning](handoffs/reopen-closed-file-next-plan-task.md)
  while .26 preparation/delivery belongs to DeepSeek. No runtime edits yet.
  [Image-reference source accepted](handoffs/markdown-image-reference-root-acceptance.md).
  Closed-file plan complete; implementation awaits .26 closure and activation.
  .26 is now accepted/closed. Owns active [closed-file implementation](handoffs/reopen-closed-file-implementation-task.md).
  Runtime source edits and focused app tests only; no .27 build yet.
  No more fixed12 requests.
- GLM `/root/read_native_handoffs_drawio_worker_resources_harness_task_md`:
  [Resource preparation accepted](handoffs/drawio-worker-resources-harness-root-acceptance.md),
  source/tester frozen. Wrapper preparation delivered; root found remaining
  runtime defects and assigned final correction to Astra. No wrapper ownership
  or requests. Raster draft/checkpoint handed off; interrupted2026-11-10 after
  repeated invalid fixture corrections. No ownership; do not resume alongside
  replacement below.
- DeepSeek `/root/read_native_handoffs_raster_admission_finish_task_md`:
  [Raster admission accepted](handoffs/raster-admission-root-acceptance.md).
  [Raster decode accepted](handoffs/raster-decode-root-acceptance.md), source frozen.
  [Native image presentation accepted](handoffs/native-image-view-root-acceptance.md)
  including requested geometry/wording corrections; source frozen.
  [Narrow image toolbar accepted](handoffs/native-image-view-narrow-toolbar-root-acceptance.md);
  runtime source frozen again. Test-only resize correction passed in
  [final app acceptance](handoffs/native-image-app-root-acceptance.md). Now resumes
  [.23 preparation](handoffs/native-image-release23-preparation-task.md) in
  new helper/evidence files and portable/README.md is
  [accepted](handoffs/native-image-release23-preparation-root-acceptance.md).
  [.23 delivery accepted](handoffs/native-image-release23-root-acceptance.md),
  package/evidence frozen. Now owns [image keyboard increment](handoffs/native-image-keyboard-implementation-task.md)
  in input.go/ui.go/imageview tests and app/binary_keyboard_test.go only.
  [Keyboard source accepted](handoffs/native-autosave-keyboard-root-acceptance.md).
  Image source frozen; [table column planning](handoffs/markdown-table-columns-next-task.md)
  delivered. [Column format planner](handoffs/markdown-table-columns-format-task.md)
  implemented/frozen with final format test/vet. No further source edits;
  Source is [accepted](handoffs/markdown-table-columns-root-acceptance.md).
  Metadata-only format-evidence clarification is complete. Now owns the
  Image-reference contract review complete. Now owns
  [image-reference format implementation](handoffs/markdown-image-reference-format-task.md)
  under the root decision; .25 build/evidence already closed and accepted.
  Format source/final test+vet frozen. [.26 preparation](handoffs/markdown-image-reference-release26-preparation-task.md)
  is followed by active [.26 bounded delivery](handoffs/markdown-image-reference-release26-delivery-task.md).
  First finish required strict lifecycle correction/reparse, then execute once;
  root source acceptance exists. Runtime source stays frozen through closure.
  The sole package call exited1 at README example link validation after all five
  GUI builds exited0. Now owns active [metadata repair and remaining delivery](handoffs/markdown-image-reference-release26-repair-activation.md).
  No repeated package/build; retain original failure, repair only README/manifest
  and narrow evidence helpers, then the one remaining strict hidden host.
  [.26 delivery accepted](handoffs/markdown-image-reference-release26-root-acceptance.md).
  Package/evidence frozen; no further product execution or source ownership.
  Now owns read-only [next memory increment selection](handoffs/native-memory-next-increment-review-task.md).
  Memory selection complete; live-tab admission planned after reopen freeze.
  Now owns [independent reopen source review](handoffs/reopen-closed-file-independent-review-task.md).
- Fresh GLM `/root/read_native_handoffs_drawio_worker_stage2_independent_task_md`:
  [Next tab keyboard plan](handoffs/tab-keyboard-next-plan.md) delivered and
  accepted as a narrow implementation direction. Independent source review and
  handoff repair complete. [.21 docs accepted](handoffs/drawio-efficiency-release21-docs-root-acceptance.md).
  .22 preparation handed off, now inactive with no ownership. Final runtime
  consistency review and delivery assigned to the fresh DeepSeek owner below.
  Do not resume this owner alongside its replacement.
- DeepSeek `/root/read_native_handoffs_release22_delivery_resume_task_md`:
  Two transport failures before completion, inactive/no ownership. .22 final
  preparation/delivery pending reassignment; do not resume alongside a new owner.
- GLM `/root/read_native_handoffs_release10_docs_task_md`: UTF8 save plan delivered;
  implementation interrupted after prolonged uncompiled draft. Inactive, no file
  ownership; DeepSeek takeover must not overlap this agent.
- GLM `/root/read_native_handoffs_drawio_worker_next_plan_task_md`:
  Stage1/view reviews accepted. Inactive after two repeated-read loop guards on
  next planning task; no ownership, don't resume. Root finished
  [next drawio data-route plan](handoffs/drawio-route-next-plan.md).
  Stage2 independent review will use a fresh context when result freezes.
- Astra is user-authorized and newly assigned tab workflow after transport
  failures in the replaced agent. Don't resume an old owner alongside its
  replacement. Actual concurrency3children plus root.
  No child may spawn children. GLM/DeepSeek compact before200K context.

Opaque task payloads: read the exact active task above and continue; publish
named handoffs. Historical task instructions are inactive unless reactivated.

## Accepted runnable package

Current `bin/levnote-native-worker-20261110.27/levnote-native.exe`:
native-worker.20261110.27,26files/27,382,619bytes; host14,556,160bytes SHA256
3B6734F30033569337A111353BD1BF661FB69F1782983688B0D3D863D0E8212C.
Manifest4,674bytes SHA256
C7783296AEEF59FD6254AF3F9487427658C40227BC5DAC15DF2348ED869926CF.
[.27 accepted13/13](handoffs/file-workflow-release27-root-acceptance.md);
one package/five builds and one strict copied hidden host exited0; no retries.
Reopen/cap behavior verified by separate combined app tests/vet. Package and
all evidence immutable; docs may update under the active task above.

Historical `bin/levnote-native-worker-20261110.26/levnote-native.exe`:
native-worker.20261110.26,26files/27,281,492bytes; host14,455,808bytes SHA256
B219024B7C5B444DA0307579C1500821A6B24FDCDD9B4D47E0FC14E863A51B67.
Manifest4,674bytes SHA256
EFF8DD2FAF05EC620AB5976C2A72E647F12570876B15B13606062B7DC9C4E20E.

- [.26 accepted13/13](handoffs/markdown-image-reference-release26-root-acceptance.md)
- [.26 delivery](handoffs/markdown-image-reference-release26-delivery.md)

.26 adds explicit native Markdown image-reference opening. Source/synthetic
view/app/in-process worker tests accepted separately. One package invocation
exited1 after five successful GUI builds; exact README/manifest metadata repair
adopted those binaries. One hidden copied host/strict mixed-session close passed.
Original failures and metadata-only retry preserved. No physical input/resource
claim. Package/evidence immutable; next closed-file implementation is assigned.

Historical `bin/levnote-native-worker-20261110.25/levnote-native.exe`:
native-worker.20261110.25,26files/27,259,765bytes; host14,443,520bytes SHA256
2421D8FC98B144DCCB0ECC1CFF239A18E11DDEF05F12FF0505B307058419AE49.
Manifest4,674bytes SHA256
CA064C3E7824EDCB179465CD8E60D982D057D2EA1D45A0E8407BD44508CF9BC2.

- [.25 accepted13/13](handoffs/markdown-table-columns-release25-root-acceptance.md)
- [.25 delivery](handoffs/markdown-table-columns-release25-delivery.md)

.25 adds before/after/delete table-column actions,1..8column limits and raw-cell
preservation with separate cell/structural undo. Source affected tests/vet passed.
One package/five GUI builds and one exact hidden host/clean mixed-session close;
no physical table/IME/resource claim. Package/evidence immutable. Workspace source
is released only for assigned image-reference work, which is not in this package.

Historical `bin/levnote-native-worker-20261110.24/levnote-native.exe`:
native-worker.20261110.24,26files/27,226,613bytes; host14,410,752bytes SHA256
A4D679FB0F35DB9258410DCC35995853780DB4E915C928E8D6434A92E6A0BD08.
Manifest4,674bytes SHA256
B1609AE10D15DBEA63DD40CC7E699E131E49156FC48821D8FD63891E565D935E.

- [.24 accepted13/13](handoffs/native-reading-release24-root-acceptance.md)
- [.24 delivery](handoffs/native-reading-release24-delivery.md)

.24 adds focused image keyboard controls, explicit100% mode and900ms quiet-time
recovery scheduling independent of painting. Recovery never autosaves original
files; UI work can delay it and uncommitted local format drafts need completion.
One package/five GUI builds and one hidden exact host with fresh pre-close clean
mixed-session recovery and normal close. No physical input/resource claim.
Package/evidence immutable. Workspace source is released for separately assigned work.

Historical `bin/levnote-native-worker-20261110.23/levnote-native.exe`:
native-worker.20261110.23,26files/27,214,997bytes; host14,400,000bytes SHA256
837A29744F0C833308B5ACC668B0F5FB2628B0710A816BF8D74DB0B516742C86.
Manifest4,674bytes SHA256
B0650A1E9A90ACAC7A68457252DA559938B1CB6D33E4B1279DE23E2688325FD6.

- [.23 accepted13/13](handoffs/native-image-release23-root-acceptance.md)
- [.23 delivery](handoffs/native-image-release23-delivery.md)

.23 adds bounded native PNG/JPEG reading, original-byte copy and mixed session
restore. One package/five builds, two exact hidden hosts/normal close; no worker
requests or physical input/driver/resource claim. Frame-dependent autosave is a
known historical .23 limitation, corrected in .24. Evidence frozen.

Historical `bin/levnote-native-worker-20261110.22/levnote-native.exe`:
native-worker.20261110.22,25files/26,863,393bytes; host14,101,504bytes SHA256
085E65F7BA1386A01E58CFCB26F0C069AACBFFD47AA063974CCDF0CC99C0C19F.
Manifest4,490bytes SHA256
4F66B7DC96E7213DF728E44A44B3DA77AE2C293BC5C08E10315156DB4793671F.

- [.22 accepted14/14](handoffs/tab-workflow-release22-root-acceptance.md)
- [.22 delivery](handoffs/tab-workflow-release22-delivery.md)

Historical `bin/levnote-native-worker-20261110.21/levnote-native.exe`:
native-worker.20261110.21,25files/26,859,809bytes; host14,097,920bytes SHA256
2C400D3730AEE64FC380CA0312B44BAAB1C0803AA70BA96525804BB4B436B329.
Manifest4,490bytes SHA256
C2556B0E372466B5F67FEBF7C9FBACBA0D44050B00F9892C64837ED16CE9A824.

- [.21 accepted16/16](handoffs/drawio-efficiency-release21-root-acceptance.md)
- [.21 delivery](handoffs/drawio-efficiency-release21-delivery.md)

- [.20 accepted12/12](handoffs/drawio-route-release20-root-acceptance.md)
- [.20 delivery](handoffs/drawio-route-release20-delivery.md)

- [.19 accepted12/12](handoffs/drawio-worker-release19-root-acceptance.md)
- [.19 delivery](handoffs/drawio-worker-release19-delivery.md)

- [.18 accepted12/12](handoffs/drawio-svg-budget-release18-root-acceptance.md)
- [.18 delivery](handoffs/drawio-svg-budget-release18-delivery.md)
- [.17 accepted12/12](handoffs/find-whole-word-release17-root-acceptance.md)
- [.17 delivery](handoffs/find-whole-word-release17-delivery.md)
- [.16 accepted12/12](handoffs/session-output-release16-root-acceptance.md)
- [.16 delivery](handoffs/session-output-release16-delivery.md)
- [.15 accepted12/12](handoffs/utf8-save-release15-root-acceptance.md)
- [.15 delivery](handoffs/utf8-save-release15-delivery.md)
- [.14 accepted12/12](handoffs/find-keyboard-release14-root-acceptance.md)
- [.14 delivery](handoffs/find-keyboard-release14-delivery.md)
- [.13 accepted11/11](handoffs/search-navigation-release13-root-acceptance.md)
- [.12 accepted11/11](handoffs/source-navigation-release12-root-acceptance.md)
- [.11 historical13/13](handoffs/retention-release11-root-acceptance.md)
- [retention fixes accepted](handoffs/retention-root-acceptance.md)
- [.10 manager/package accepted](handoffs/fixed-engine-package-delivery-root-acceptance.md)
- [.11 docs accepted](handoffs/release11-docs-result.md)

.22 adds visual-order CtrlTab/ShiftCtrlTab, one-frame active-tab reveal, safe
close-loop mutation and GoToLine pointer-leave composition reconciliation.
One package/five GUI builds and one exact hidden host close; no worker requests.
Tab source tests/Tester geometry accepted separately; no physical IME/resource
claim. Package/evidence immutable. .22 source freeze ended after acceptance.
.21 adds canonical drawio warning cleanup and source-buffer ownership reduction.
One actual installed worker request:516source/598SVG/zero warnings,exit0/joined;
one separate hidden host,clean UTF8+BOM custom-route session. Five GUI builds;
initial aggregate metadata-only failure corrected once,no product work repeat.
Direct202files91722328B plus retry4files1635B immutable. Allocation evidence
is source microbenchmark only,no whole-app memory/speed/physical UI claim.
.20 adds schema3 builtin.drawio routing for custom suffixes. One hidden exact
host opened .diagramnote using the reviewed installed JSON and persisted clean
preview/BOM session; no worker request, physical UI or resource measurement.
Worker behavior inherits .19. Warning/source-retention changes are future source.
.19 adds fourth fixed drawio SVG worker, exact current-page metadata and view
epoch/owner publication checks. One actual installed-engine request and joined
lifecycle, separate exact hidden host start/owned-close accepted. Five GUI builds,
no physical UI/IME or resource evidence. Data-only drawio suffix route is future
workspace source and NOT in this immutable package.
.18 adds64MiB bounded host-side draw.io SVG emission. Exact old-byte oracle,
selected-page regression and narrow corrections accepted; one hidden exact
host startup/clean close. Not whole-process memory/physical UI proof. Still3
worker engines in this immutable release; fourth-engine work is future source.
.17 exposes existing whole-word Find for document navigation/replacement and
workspace search. App/shared option plus checkbox, no core algorithm/schema.
Two760x480Tester captures and synthetic/method tests; one hidden exact host close,
no physical input/resource claims.
.16 removes Marshal's extra returned JSON copy during session output while
preserving exact JSON bytes and atomic lifecycle. Encoder still buffers a full
JSON value. Output-only allocation benchmark, not whole-Save/process memory.
One hidden exact host/normal close confirms clean-session JSON without terminal
newline; no dirty recovery/worker/physical input or resource measurement.
.15 adds UTF8/BOM snapshot output through32KiB buffer with exact digest gating;
output-only allocation benchmark, not whole-Save/process memory measurement.
One hidden exact host/normal close, no dirty-save/worker/physical-UI run.
.14 adds Find keyboard flow and focus-leave composition cleanup, with one
hidden host/normal close, no worker execution or physical UI/resource claim.
.13 adds search navigation/cache and minimum-window Find row wrapping. One
exact hidden host/normal close; no worker execution repeat or physical UI claim.
.12 adds Ctrl+G/status source navigation and inherits .11 retention fixes.
.12 worker manifests checked; .11 managed execution evidence inherited, not
rerun. One .12 hidden exact host/normal close and two new tester images accepted.
.11 clears obsolete piece/tab backing-array references, not measured process
memory improvement. .10 remains historical accepted; .9 failed history. All
release directories/evidence immutable. .11 real managed smoke uses installed
.10 then bundled .11; one hidden exact host startup/WM_CLOSE passed. Four
engine-panel images remain historical .10 captures, not physical UI/IME tests.

## Development and product constraints

- All Go uses `native/scripts/go.ps1 -GoArguments @(...)` from repo root, selecting
  Go1.27.3/local cache/vendor/startup patch. No raw/system Go. Preserve actual
  command/output/exit and failures. `native/` untracked: empty git diff proves
  nothing. Use small scoped inventories only where needed, no toolchain hashes.
- MyGo drawing/font shaping/native input primitives only; no prebuilt document
  editors. Small utility controls permitted. Format views use neutral host
  contracts, no direct filesystem or introspection of toolkit widget trees.
- Pure local runtime: no network/account/sync/telemetry. JSON format configs are
  nonexecuting. Fixed4catalog engine packages separately imported/activated
  in accepted source (.18 runnable still has3 until .19 delivery);
  import doesn't activate, installed selection explicit, no arbitrary executable
  path/command overrides. Hashes aren't signatures, sandbox or legal immunity.
- Preserve untouched bytes, unknown metadata, readonly/encoding/BOM, undo and
  drafts. No silent overwrite/transcode/loss/fallback. Plaintext/byte inspection
  remains reachable; actual limits and failures must be visible.
- Engine selection joins old work before persistence. Cleanup may fail after
  commit: show actual state. Poison blocks execution/transitions while import/
  list remain usable. Damaged selection state differs from individual bad
  package. Never release writable drafts just to switch engines.
- Avoid duplicate gates/unrelated suites. Run substantive focused tests and one
  final affected full suite after corrections. Root never executes tests/builds.
- Computer Use helper failed3bounded native-pipe probes before launch; see
  [result](handoffs/release10-ui-feasibility-result.md). No helper retry/resource
  matrix active. Don't bypass with homemade UI automation. Tester captures and
  hidden startup don't establish physical OS IME or memory/CPU improvements.

Historical coordination (inactive ownership/freeze instructions):
[through .11](handoffs/coordination-history-20261109-through-release11.md) and
[earlier](handoffs/coordination-history-20261109-before-release9.md).
