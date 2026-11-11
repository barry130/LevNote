# Mindmap tags — focused implementation handoff

Status: FOCUSED/CONSUMER/.4 DELIVERY ROOT ACCEPTED; FINAL DOCS COMPLETE2026-11-09.
Root accepted the focused checkpoint595570D5 and final consumer7ED7253B below,
including actual full graph7.607s/app6.957s/vet0 and the two final narrow captures.
The authorized narrow
recovery-layout, confirmed-input guidance and conditional-warning identity fixes
are included. The active mindmap-tags-delivery-next-stage.md execution below
completed portable usage text, one fresh native-worker.20261109.4 package and
one existing HostPath tester invocation/two workers in a disposable copy. Root
read and accepted exact aggregate415F685C, validator/source, raw RUN/PASS and
shipped README/frozen inventories. Source launch links now point to.4; final
README/VALIDATION/API/architecture/roadmap/changelog describe exact tag behavior
and limitations. Product/tests/scripts, accepted packages and evidence remain
frozen. The independent actual-time root-acceptance record is linked below.

`formats/mindmap/tags.go` implements Tags/AddTag/RemoveTag/ClearTags as designed:
ordered defensive copies including nested Raw/keys, exact append, ordinal-only
removal, last removal present[], absent/empty clear no-op, validation rollback.
It imposes no view-specific line/count/byte limits. The existing parser and
serializer are unchanged; real edits canonicalize envelopes while preserving
unknown member Raw spellings and order. Unchanged/refused/no-op bytes stay exact.

`tags_test.go` independently checks mixed string/object/empty/duplicate lists,
class/style presence, nested numeric lexemes and member order after reparse,
unrelated metadata, all mutable return layers and pre-mutation survivor aliases,
exact original bytes/no-op, missing/index/UTF8 failures and document-depth
validation rollback for all three mutators. A65-item/513byte+CRLF programmatic
case proves the pure API does not silently impose inspector limits.

Actual pure command `go.ps1 -GoArguments @('test','./formats/mindmap','-run',
'^TestTags','-count=1','-v')` passed0.691s, exit0. Exact streams/command/source
fingerprints and UTC/wall record are in
`../artifacts/acceptance/mindmap-tags-20261109-042028-pure-attempt01/`.

Native source now implements the independent pending tag input and public Flush
barrier before reentrancy, private ordinary-field0/1version receipt (including
source owner/input identity/selection and post-Changed exact bytes), prospective
budget checks both before and after field serialization, and actual candidate
budget after reparse. A completed field write followed by refusal is reported
separately; its undo remains intact. Completed tag commits clear only the exact
applied input and refresh only unchanged input/field/lifecycle identities.

The optional tag section is after the established ID/local-link/structure
controls, keeping their default compact reachability. Input/actions precede the
ordered list. F6 from the ordinary input reaches the tag section; F6/Tab from
the tag input reaches Add. While composing, F6/Tab/ShiftTab keep focus, bytes,
selection and the input group with confirmation guidance. Existing noncomposing
ordinary Tab traversal still works. The existing keyboard-link test now confirms
composition using native Type before Tab, reflecting this deliberate policy.

Add/remove/clear/discard/recovery all use native down-event identity and reject
repeats/continuation clicks. Scroll cancels pending gestures without consuming
the normal inspector scroll. Recovery captures both exact pending strings and
retains the old field-only envelope; callback replacement drafts survive. The
ordinary error-row recovery works even when tags are collapsed, with captured
open-state/epoch proof. Release retains pending input buffers and drops caches.

Actual native attempts, all through the reviewed go wrapper and explicit arrays:

- `../artifacts/acceptance/mindmap-tags-20261109-043453-native-attempt01/`:
  FAIL1.397s. New section initially displaced links; moved after primary controls.
  Empty legacy text assertion used painted text rather than its ordinal-access
  descriptor; corrected after explicitly revealing the row. Composition guidance
  was offscreen; a fixed local hint now accompanies the input.
- `../artifacts/acceptance/mindmap-tags-20261109-043849-native-attempt02/`:
  FAIL3.124s. A refused field switch requested ordinary-input focus; pending-tag
  refusal now leaves the focused tag editor unchanged. The ordinal assertion now
  verifies visible bounds plus its literal accessibility description.
- `../artifacts/acceptance/mindmap-tags-20261109-043947-native-attempt03/`:
  PASS3.455s, exit0. Native tags, prior fields/composition/group/conflict cases.
- `../artifacts/acceptance/mindmap-tags-20261109-044247-native-attempt04/`:
  PASS3.997s, exit0. Adds collapsed dual-draft native recovery, exact64/65/10,000
  list bounds, post-serialization/canonical16MiB rejection, host4MiB recovery
  refusal retention, Buffer replacement/Release, and UTF16 selection-clamp undo.

Other meaningful oracles include duplicate ordinal/object Raw survival, exact
literal Unicode/space/comma add, two separate field/tag undos, private-helper
reentrant public barriers, post-field Raw growth over prospective8192, arbitrary
Changed owner/source/selection/replacement/Release rejection, success preserving
clean replacement fields, held discard/recovery identity, actual native scroll,
rapid reindexed remove continuation plus a fresh single click, and settled idle
frames. Helpers directly stage callback/budget cases; real native input coverage
is separately identified. No OS IME/app/capture/package/resource claim.

Final owned-package gate is complete. Evidence is
`../artifacts/acceptance/mindmap-tags-20261109-044422-focused-final/`:

- `go.ps1 -GoArguments @('test','./formats/mindmap','./modules/graphview','-count=1','-v')`:
  mindmap PASS0.534s, graphview PASS6.696s, exit0; wall8.3494835s.
- `go.ps1 -GoArguments @('vet','./formats/mindmap','./modules/graphview')`:
  exit0, empty stdout/stderr; wall1.1065894s.
- Thirteen source files unchanged between recorded pre/post hashes. Three
  preexisting opt-in image tests skipped without their environment variables;
  no new image/app/worker/package/UI launch was run.
- `result.json` SHA256
  `595570D5050852387CA4700909478BE8E18DE25A1FBF96AE45FB986F41ED326F`.
- `source-after.json` SHA256
  `11D1CC4DE019D51F4F2A9EDAA6CE8F4A4BF2625F5E41907C86D1F19DE7C22376`.
- `test.stdout.log` SHA256
  `D8CE4F72E3F3D1C1A6FE16EC45AB9A8B6ACEA140ABAD7DB636F834E5E3FB0CCF`.

Final source list (full bytes/SHA256 per file in source-after.json):
`formats/mindmap/{tags.go,tags_test.go}`;
`modules/graphview/{tags.go,tags_ui.go,tags_test.go,draft.go,adapter.go,sync.go,
input.go,fields.go,fields_test.go,outline.go,view_ui.go}`.
No parser/schema/canvas/SVG/app/core/vendor/runtime/protocol/package-script edits.

New graphview test names actually RUN/PASS in the final stream:

- TestTagsNativeOrdinalMetadataAndSeparateUndo
- TestTagsNativeKeyboardComposeBarriersAndDraftFlush
- TestTagsNativeNodeCompositionFocusGuard
- TestTagsNativeDualPendingAddAndReentrantBarrier
- TestTagsNativePointerCompositionRepeatCancelAndScroll
- TestTagsReceiptRejectsChangedCallbackMutations
- TestTagsPostFieldRawGrowthProspectiveBudget
- TestTagsNativeHeldDraftControlsCannotRetarget
- TestTagsNativeRapidContinuationCannotDeleteNextOrdinal
- TestTagsNativeOpenerContinuationAndExplicitDiscard
- TestTagsNativeIdentityStaleWithoutDrawAndAwayBack
- TestTagsSuccessfulCommitKeepsReplacementInputsAndReleasedCache
- TestTagsRecoveryExactBothLegacyAndReentrantReplacement
- TestTagsBudgetInputPresenceAndIdle
- TestTagsSharedDrawioReceiptAndFieldOnlyRecovery
- TestTagsNativeCollapsedDualDraftRecovery
- TestTagsActualLimitsCandidateAndRecoveryRemainExact
- TestTagsPendingReleaseRefreshAndSelectionClamp

The last opener test proves its bounds stay fixed on opening, the rapid second
click still hits the opener and changes no source, and later explicit discard
retains the ordinary pending draft. The separate reindexed-remove regression
proves a rapid continuation really intersects the next ordinal's remove button
but cannot delete it; a later independent click commits once.

That focused checkpoint was accepted by root before the consumer below. Its
native Tester evidence alone did not establish host-level recovery integration.

## Registered app consumer and demonstrated corrections

New `app/mindmap_tags_app_test.go` uses the registered mindmap adapter and real
native controls. Its rich fixture independently verifies duplicate/empty/object
tag ordinals, class/style presence, ordered unknown Raw numeric spellings,
unrelated top/node/image fields, literal Unicode/spaces/comma/HTML-like input,
clear/last removal present[], exact undo/selection, unchanged BOM+CRLF save,
actual-edit canonical LF save/reopen and XML-parsed SVG escaping. Pending tag
input blocks Save/export/mode/tab/close before dialogs. Ordinary field+tag Add
is two separate undo entries. Split-source conflict preserves both inputs;
the actual synchronous app RecoverDraft creates the exact labeled unsaved copy.
Post-field Raw growth refusal retains the tag input and the independently
undoable field write. Draft text is verified through native select-all/copy,
because this repository's custom Editor does not populate AccessInfo.Value.

Consumer failures justified three narrowly approved production corrections:

- Ordinary draft-error recovery is a96pxremaining-row control at760width,
  accessible as `恢复节点及标签草稿`; the lower panel retains its fullwidth
  `恢复标签及节点草稿`. Both use the existing captured identity/lifecycle guards.
- `errTagConfirm` is transient local guidance exposed in Warnings, not a
  persistent Host.Report failure. Actual confirmation clears only that sentinel;
  real validation/budget/conflict errors remain. Empty input or explicit discard
  clears its resolved pending-tag warning while retaining any ordinary draft.
- `app/view.go` gives the conditional format-warning row and split container
  stable keys, and passes the tab key to native Draw. MyGo's unkeyed IDs depend
  on sibling ordinal; the first warning formerly changed the native view ID,
  dropping composition focus and inspector scroll. The new always-run test
  starts with no warnings, creates/removes the first warning in preview/split,
  updates the marked composition, confirms different exact text and adds it
  once. It verifies source/history/disk and no stale host status/issue.

The two760x480captures use duplicate/empty/object tags without unrelated
warning-producing top/node/image fields, allowing input+Add/discard/recovery
in the same frame. Rich metadata remains in functional tests. Every relevant
control is separately scrolled into view and checked against actual Bounds,
Offscreen and the29pxstatus boundary. The dark case also checks the compact
ordinary recovery control and activates real app recovery after the capture.
The light case commits exact confirmed input after capture. No OSIME/GPU or
external accessibility claim follows from MyGo tester images.

Preserved consumer attempts under `../artifacts/acceptance/`:

- `mindmap-tags-20261109-044940-consumer-attempt01`: compile failure, tester
  dimensions required int; test-only conversion corrected.
- `mindmap-tags-20261109-045006-consumer-attempt02`: FAIL3.849s. Incorrect
  accessibility Value oracle corrected to native clipboard. Actual ordinary
  recovery button overflowedX783 in760window; distinct names/width fixed it.
- `mindmap-tags-20261109-045150-consumer-dark-diagnostic`: retained failing
  dark screenshot before the narrow production correction.
- `mindmap-tags-20261109-045711-consumer-attempt03`: fmt harness rejected files
  from two directories in one invocation; no tests ran. Separate fmt calls used.
- `mindmap-tags-20261109-045732-consumer-attempt04`: affected native PASS1.332s
  and four rich functional consumers passed. New clean capture fixture exposed
  the first-warning composition/focus failure; capture checks failed.
- `mindmap-tags-20261109-045824-capture-attempt05`: retained diagnostic failure
  and images showing focus lost and inputY385→626 on first conditional warning.
- `mindmap-tags-20261109-050035-capture-attempt06`: stable-key regression and
  captures PASS3.702s. Visual review found historical host status still retained
  confirmation guidance; subsequent change keeps only that sentinel local.
- `mindmap-tags-20261109-050143-consumer-attempt07`: final affected native
  PASS2.531s and first-warning/capture consumer PASS4.294s. Fifteen inputs
  unchanged; result SHA256
  `27CDA7A5ACDE4776232E51A37587964B1A990BEE76D2E285D297D5715C9AD64D`.

Final images in attempt07/images, visually inspected after the final code:

- `mindmap-tags-light-expanded-760x480.png`:43,401bytes, SHA256
  `A8CC3DD92B0A47036DC306D0FD886178AEF8B9D9524873D17129E6AA69A9D8FA`.
- `mindmap-tags-dark-recovery-760x480.png`:46,478bytes, SHA256
  `17A74B42FB8C8590331779F27464B415DAB2BF8DC45A4A9DE8B2723C9FC6153D`.

## Final consumer gate

`../artifacts/acceptance/mindmap-tags-20261109-050233-consumer-final/` contains
the exact scripts, raw streams, UTC/wall results and15pre/post source hashes.

- `go.ps1 -GoArguments @('test','./modules/graphview','./app','-count=1','-v')`:
  graphview PASS7.607s, app PASS6.957s, exit0; wall8.8931777s.
- `go.ps1 -GoArguments @('vet','./modules/graphview','./app')`: exit0, empty
  stdout/stderr; wall1.3526358s.
- `result.json` SHA256
  `7ED7253BCECD36FD5E459B0DCAB8F0D48E1C4CE1B40CEB8664264255FD63313A`.
- `source-after.json` SHA256
  `1A218DDD145CC02A74A5EF114F0B38A340B43E4321951ABF12816D61E49ADDD2`.
- `test.stdout.log` SHA256
  `12053ABDD200DFC91C27E2CE4BD042557E41A9565783AED9DE98760E990619A5`.

The15file list is the original13focused inputs plus
`app/{mindmap_tags_app_test.go,view.go}`. Consumer production changes are limited
to `modules/graphview/{tags.go,tags_ui.go,draft.go,view_ui.go}` and `app/view.go`;
`tags_test.go` adds the exact sentinel/host-report/resolved-pending regression.
The pure implementation/parser/schema/canvas/SVG and other frozen layers did
not change during consumer work. The existing pure format acceptance is reused.

Six app top-level tests exist; the first five RUN/PASS in the final ordinary
app suite, while the last opt-in capture passed both subtests in attempt07:

- TestMindmapTagsRealAdapterExactSaveUndoModesAndSVG
- TestMindmapTagsRealAdapterPublicBarriersAndSeparateFieldUndo
- TestMindmapTagsRealAdapterSplitConflictAndRecovery
- TestMindmapTagsRealAdapterPostFieldBudgetRefusal
- TestMindmapTagsRealAdapterFirstWarningPreservesComposition (preview/split)
- TestMindmapTagsAppNarrowCaptures (light-expanded/dark-recovery)

Ordinary final tests remove LEVNOTE opt-in variables in their child process:
existing image/runtime/bundled-worker acceptances correctly SKIP. No real worker,
new app process, resource matrix, full repository suite or package was run for
this consumer gate. Root accepted its exact source/evidence/images before
activating the separate.4delivery gate above.

## Fresh.4 delivery checkpoint

Candidate clean program folder: `../bin/levnote-native-worker-20261109.4/`.
Evidence: `../artifacts/acceptance/mindmap-tags-20261109-050613-package/`.
The unchanged packaging script ran once, version`native-worker.20261109.4`;
no product/test/script edits occurred after the accepted consumer. Portable
README now documents tag limits, explicit Add/discard/recovery, separate undo,
unchanged-vs-reserialized bytes and running-window-only staged-input retention.

Actual commands all exit0 with empty stderr:

- Package command3.8862651s; worker build0.9362357s, host build2.0892809s.
- Existing app tester compile1.7653516s into a complete19-file evidence copy.
- One `TestWorkerExportBundledHostPathAcceptance` invocation, PASS0.78s;
  command wall1.632111s. Its two real worker requests produce331HTMLbytes,
  one warning and two injected SaveDialogs, then refuse existing target overwrite.
  Exact source/BOM/history/baseline/HTML oracles pass; `RuntimeJoined=true`.
  This is the source-reviewed joined-runner contract, not a per-PID observer or
  packaged-host UI launch. No other process/resource/acceptance suite was run.

Independent read-only validator: **12/12PASS**. Complete release has19files,
17,014,847bytes,13exact license/attribution assets,18relative manifest entries
and19valid authored local links. Both actual PE headers are Windows amd64 GUI.
The19disposable copies remain unchanged; only the tester is extra in that copy.
All1031frozen inputs,15,657toolchain files and373protected prior files retain
their captured inventories/bytes/SHA256; prior deliverables also retain UTC
mtime ticks. This includes preexisting bin payloads and earlier tag/row evidence.

Actual payload identities:

| File | Bytes | SHA256 |
| --- | ---: | --- |
| `levnote-native.exe` | 13,663,232 | `29E6148B844FDACB3CF12854ED95F1F2D3F779B05E711BB5C3A30DAA964C6A6E` |
| `engines/markdown-export/levnote-markdown-export.exe` | 3,295,744 | `9EF039FE01C81CE7445DE3A571947D752D611C439A081B4E35B660731A6F4537` |
| `engines/markdown-export/manifest.json` | 337 | `839D5D88DC308C67FAF846BC4B779503228840D6988E293861918DE5F8294027` |
| `release-manifest.json` | 3,411 | `BC08D9EFDC43997B1956ED1C95BCF2FC9D84A6B60A582E26DAE800823EF52AFC` |

The worker embeds a new version string; its new hash is expected despite
unchanged renderer/runtime. No binary-equivalence or resource-improvement claim.

- `aggregate.json`:46,948bytes, SHA256
  `415F685CD76CECC50AB5F9D52BFDC21E6662243B782CBDAF96D3B75B19F8D330`.
- `validator.json`:4,452bytes, SHA256
  `FD05487F2206BBF86F30C2663AE8AF8F2F94F4A771ED76EDE0FFE2B250384598`.
- `aggregate.ps1`:19,407bytes, SHA256
  `39BE1F98A68B3F573BE64EA10721F1D01271B7A9EDC0759FED9B1CA1BAB00E04`.
- `host-path.stdout.log`:825bytes, SHA256
  `90F2901BF1F8BF71F3358A43AD2212B8F1D187A6297FA2C6353C2E54F0830271`.

Root accepted this exact package after reading aggregate/validator source,
original HostPath RUN/PASS/HTML331B/two dialogs/RuntimeJoined, substituted README
and all frozen inventories. Aggregate, validator, candidate and prior evidence
remain immutable. No additional test/build/launch followed acceptance.

Final source documentation: root `README.md`, `ROADMAP.md`, `CHANGELOG.md`;
native `README.md`, `VALIDATION.md`, `ARCHITECTURE.md`, `formats/API.md` and this
handoff. Both outdated native README tag-unavailable statements are corrected;
node images remain placeholders. Current launch is.4; historical.1/.2resource
and.3row results retain their original version/scope. Staged tags are not claimed
as crash-recovery data. [Separate root acceptance](../artifacts/acceptance/mindmap-tags-20261109-050613-package/root-acceptance.json)
records actual creation UTC and hashes read from accepted paths, without editing
the accepted aggregate. Selected-column alignment remains queued for a later
design checkpoint; no alignment code or validation was performed in this stage.
