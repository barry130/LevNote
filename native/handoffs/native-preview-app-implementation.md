# Native readonly preview integration — root activation

2026-11-09. Root read the actual editor/app/graphview/runtime lifecycle source
and `native-preview-app-source-checkpoint.md`. The additive neutral API and
separate scene view in that checkpoint are accepted. Astra owns implementation
and meaningful tests in exactly its listed editor/app/graphview files. Production
catalog/IPC and the command/codec are frozen; GLM owns transport test evidence.
The current accepted runnable `.6` and historical evidence stay immutable.

## Implement now

Proceed continuously with the checkpoint and `native-preview-app-root-plan.md`.
Do not wait for another design round. Publish a short source-ready handoff after
the shared lane and app lifecycle wiring, then implement the separate native
scene consumer. Root will read source while work continues. No other agent edits
these product files. No child delegation.

The scene capability is a fixed catalog operation; no command/path/source authority
is accepted from a View. Readonly production Mindmap always selects sceneView when
the capability is advertised, including missing provider/bundle cases. Those
fail visibly. Editable Mindmap/drawio and source-only disabled-plugin behavior
remain covered by relevant regressions.

## Review clarifications

- Lane execution, cancellation joins and Close must finish without a functioning
  UI dispatcher. Queueing a UI notification must not be a dependency of process
  cleanup or closing its completion channel. No worker reads mutable App/Tab or
  Buffer; no closure accidentally retains raw payload after the decoded result
  has been transferred or invalidated.
- Reserve export only after its existing flush/identity checks. Maintain the
  reservation through modal path choice and publication. Its final UI cleanup
  revalidates the one newest pending preview before admission. Canceled exports
  and immediate failures release the reservation exactly once. A cleanup failure
  poisons this execution lane and remains visibly reportable even if its original
  view/tab has gone. Close repeats the same stored cleanup outcome.
- Clear pending ownership and callbacks immediately on cancellation. A canceled
  running request may retain only what execution/join still needs. Bounded stale
  display is separate from at most one current result slot; stale callbacks must
  not hold an extra large scene/source indefinitely in the UI queue.
- `Release` is reusable. Warnings-before-Draw is the first request point; no
  synchronous completion through BeginScene. Failure does not trigger an idle
  frame retry. Modal unwind drains preview as well as export exactly once.
- `ShowSource` must remain reachable with keyboard and recheck current owner.
  Scene-only outline resolves visible IDs; hidden resolved source target returns
  ErrOutlineReadOnlyHidden without invoking a concrete parser or changing source.
- Keep theme roles and literal color precedence. Guard tiny positive dimensions
  before float32 conversion; transforms and fitting stay finite. Label values are
  already bounded by producer; preserve them without a second truncation.

## Verification boundary

Add the finite fake-runner and native literal-scene tests described by checkpoint.
Run focused checks as needed to implement correctly, retaining real raw output
for claimed results. Final changed editor/graphview/app package tests and scoped
vet follow root source review and transport gate; preserve failing attempts and
rerun only justified scope. Record before/after owned inputs and exact commands.
No bundle/build/real-worker launch/resource/physical UI test yet; those form the
next stage after this consumer is accepted. Root does not execute any test.
