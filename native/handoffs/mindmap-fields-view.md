# Mindmap fields — native view implementation

## FINAL OWNED CHECKPOINT — product frozen for root/app integration

Final full owned package: PASS2.631s; full owned vet exit0/no diagnostics.

```powershell
native/scripts/go.ps1 -GoArguments @('test','./modules/graphview','-count=1','-timeout','120s')
native/scripts/go.ps1 -GoArguments @('vet','./modules/graphview')
```

All current graphview tests passed, including previous draft/oversize/resource/
outline/drawio regressions. No full repository test or release build run here.
No known owned blocker remains. Final field native matrix/images passed1.495s;
bounded long-field532x285 reachability passed1.355s after letting the test
clock advance through normal wheel scrolling (no product layout workaround).

New/changed product files: fields.go, field_link.go, draft.go, adapter.go,
view_ui.go, input.go and edit.go. Test file: fields_test.go. Pure-format seam
and app files remain under their respective owners. Three stable selectors
share one independent Editor; GLM's setters preserve literal optional state.
Pending identity includes Buffer/node/field/presence/value; native pointer and
Enter/Space capture exact original link/draft. Post-Flush own input token and
baseline must agree. No current-node/label fallback. Canceled/outside/released,
field-switched, unavailable and view-released gestures drop their draft pointer.
Ordinary graph collapse retains field; doubleclick/new child/sibling focus topic.

Actual native coverage completed:
- absent/present-empty no-op, exact Unicode/CRLF/trailing spaces, one committed
  history step, clean barriers, clear note/link, unknown metadata and Undo/Redo;
- other-field source merge, same-field/presence-only conflict, deletion, new
  Buffer, foreign group, readonly, oversize, failed switching and Release/recovery;
- pending link pointer/Tab+Enter/Space, other-field Compose then open, empty
  write/remove/refusal, missing handler, readonly navigation, 4096/4097/30000B
  limits, native source-changing keyboard and press/release identity matrices;
- synchronous OpenReference->Release during real current Draw, another fresh
  action after rebuild, exact once callback, pure-open source/history/selection;
- clean source Undo refresh, double-click/new-node topic focus, drawio isolation;
- long-field segment controls/input/link/retry/recover reachable via inspector
  scroll at532x285, full source/draft text retained and idle frames settle.

Pictures: artifacts/acceptance/mindmap-fields-view/
mindmap-fields-{light,dark,readonly,conflict}-760x480.png. Astra inspected
light/readonly/conflict; root inspected all four and accepted visible geometry.
These are component native tester images; app owns the integrated760x480 image.
Input is native tester transport, not real Windows candidate-window acceptance.

Introduced200,000B readonly-note workload confirms the pending comparison cache:
before five unchanged frames1,036,200B/201allocs/546.5µs; intermediate after
cache12,216B/197allocs/528.1µs; final no-duplicate-note-summary path9,736B/
182allocs/528.2µs. Fixture/first layout outside measurement, FramePending=false.
Only cumulative headless allocation/time; no new process-resource launch.
Full field/base remains owned; edits/parse/serialization still have linear work.

Native Inspector paints Editor glyphs via Painter.TextLayout, so Tester.Texts
does not expose its full contents. The former note summary ui.Text was removed
to avoid duplicate display. App verifies exact selected note by real copy and
clipboard plus PNG/geometry, not a duplicate text-label assertion.

Resource and feature documentation follows this freeze; final binary metadata
waits for DeepSeek/root acceptance. No additional graph feature expansion.

## Compiling native-control checkpoint

Current tree compiles; actual native fields/input tests pass1.047s:
`go.ps1 -GoArguments @('test','./modules/graphview','-run','TestMindmapFieldsNative|TestMindmapLongNoteUnchangedFieldProbe','-count=1','-timeout','120s','-v')`.
This is focused only, not full owner/integration completion. App may consume
the exact controls now; no guessed internal IDs are needed.

- Stable selector labels: `字段：主题文本`, `字段：纯文本备注`, `字段：本地链接`.
  No `（当前）` suffix. Selected button has border/background + current tooltip.
  Native RoleButton, enabled for readonly reading.
- Single RoleTextField input: `节点文本`, `节点备注（纯文本）`,
  `节点链接（本地）`. ShapeLimit512, bounded inspector scroll/field viewport.
- Action RoleButton: `打开链接`, or `写回并打开链接` for same-node pending
  hyperlink input. Actual primary click and focused Enter/Space capture native
  identity, Flush/revalidate, then call Host.OpenReference once. A callback
  synchronously Release is covered through actual Draw/native click.
- Mindmap property field Tab now leaves the text field for the enabled link
  action (or selected field selector if unavailable); Shift+Tab returns to the
  selector. This is scoped to the property container; source/drawio indent is
  unchanged. Enter/Space on the link button activate through native input.
- Over4096-byte destination/missing host/no destination is visibly disabled,
  with reason. Navigation refusals use Host.Report; failed Flush also retains
  existing draft error/retry/recovery. Readonly editing is blocked; navigation
  and reading remain available.

At the early checkpoint, native focused coverage already passed absent/present-empty no-op, exact
Unicode+CRLF note Compose followed by field switch, one undo/redo transaction,
clear-note/clear-link semantics, other-field merge, same-field conflict,
Release/recovery and readonly. Then-remaining (now complete above): stale native activation matrix,
presence-only/deletion/new-buffer/foreign-group/oversize field coverage,
topic focus regressions, bounded geometry/screenshots and full owned test/vet.

Measured introduced200,000B readonly note workload, fixture/initial view outside:
five unchanged frames before dirty-comparison cache1,036,200B/201allocs/546.5µs;
after buffer-version-keyed exact comparison cache12,216B/197allocs/528.1µs,
FramePending=false both. This narrowly avoids repeated Buffer.String copies;
full field/base still retained and actual mutation/serialization remains linear.
These are headless cumulative allocations, not OS memory/CPU samples.

ACTIVE acknowledgement. Read the current AGENTS.md, mindmap_fields_stage_plan.txt
and final mindmap-fields-contract.md. Astra owns graphview only: one native
topic/note/link editor, presence-aware source drafts, exact native link gesture
identity, bounded field presentation and focused actual-input/images. No peer
implementation edits, resource reruns or full integration build.

Implementation is proceeding from the approved UI proposal. GLM publishes the
pure Field/SetNote/SetHyperLink/ClearField seam; app owner receives exact native
control labels and behavior as soon as compilation/input checkpoint is ready.
Resource documentation waits for DeepSeek's arithmetic reconciliation of the
existing JSON. The first17:40 restore is primary,17:42 supplemental; original
matrix FAIL and missing-preclose-persistence limitation remain visible.
