# Paint-independent recovery scheduling result

2026-11-10. Autosave source frozen. All eight new focused tests passed on their
first run. The corrected final combined app/imageview test and vet both passed.
The first combined imageview failure remains preserved; its owner corrected
explicit100% mode and added a meaningful public resize regression. See
native-autosave-keyboard-final-validation-result.md.

## Implemented

Recovery no longer waits for View or Context.After. Every production mutation
that formerly set persistPending now calls markPersistPending; source edits use
noteEdit to preserve the last-edit+900ms deadline. Tab and preference changes
reuse the remaining deadline, including an already queued callback.

One cancellable one-shot timer captures Window.Update on UI. Its background
callback only posts through that captured dispatcher. UI delivery checks request
identity, pending state, shutdown, restore, modal nesting and the deadline. An
early callback schedules only the remainder. Existing headless uiDispatch is
never assumed safe for real timers; deterministic tests explicitly inject their
fake timer/clock and queued dispatcher.

Modal exit rearms after deferred results. Restore cancels before its transaction
and retains its final pending=false decision. Explicit persistence consumes the
request on success or failure, so failed close/modal unwinding does not retry an
unchanged state. The intentional failed-cleanup recovery path marks a fresh
request as before. Shutdown cancels before joining owned work. Queued/stopped
callbacks cannot resurrect a later request.

Automatic capture neither Flushes format views nor commits source preedit.
It retains the existing committed-document/session serialization contract;
format-local drafts still need the existing explicit commit paths. Binary
records remain path/kind/title/plugin/view metadata without payload.

## Verification

All commands used scripts/go.ps1 via the preserved pwsh7 runner in
native-autosave-evidence-20261110. No failure was discarded.

- 01-format: go run cmd/gofmt -w on the11 owned files, exit0.
- 02-focused: go test ./app -run ^TestRecoveryAutosave -count=1 -v,
  exit0; app1.971s. Eight tests cover no-View capture, queued UI dispatch,
  no Flush/preedit commit, debounce/stale/early delivery, non-edit coalescing,
  nested modal cancellation, Restore/Shutdown, explicit-close supersession,
  failure/new-edit/explicit retry, binary wire metadata, and inline dispatcher
  exclusion.
- 04-final-test: go test ./app ./modules/imageview -count=1,
  command exit1; app PASS21.808s, imageview FAIL16.323s at the100% toolbar action.
  No app failure. Vet was held until the authorized narrow correction froze.
- 08-corrected-final-test: same combined command, exit0; app23.863s,
  imageview15.447s, wall27.773s.
- 09-final-vet: go vet ./app ./modules/imageview, exit0, empty output,
  wall1.533s. No additional test or vet run.

All11 autosave source identities in source-freeze.json match the pre/post15-file
combined inventories03-source-before.json and06-source-after.json. Corrected
07-corrected-source-before.json and10-corrected-source-after.json match all15
current files; only the image owner's ui.go/test correction changed between
the two validations. Autosave source was unchanged. Ownership was autosave.go, autosave_test.go,
app.go, state.go, view.go, workspace.go, export_worker.go, binary_actions.go,
outline.go, plugins.go and source_navigation.go, all under app/.

No session/document/module/vendor change, package build, host/worker launch,
capture, benchmark or resource run. Deterministic queue/fake-time tests establish
the app scheduling mechanism; they are not physical Windows minimize/IME or
driver/resource evidence. The accepted .23 artifact/evidence remains immutable.
