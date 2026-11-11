# Schema3 app gate — acceptance ready

2026-11-09. Implementation, finite workflows and three native headless captures
complete. Product/tests/examples are frozen for root review. Accepted `.7` and
historical evidence are unchanged; no package, real worker, physical UI or
resource measurement was executed in this stage.

Final evidence directory:
`artifacts/acceptance/format-routing-app-20261109-105444-final04/`.
`aggregate.json` PASS10/10 SHA256
`77DF322DE9DCE9C762173E642931671934E942839E3AB1FBA4CDAD7CF87D14E4`;
validator `aggregate.ps1` SHA256
`EA656E7D2A2E95A66320950B669B825730D531E5AA26CD09C855C0D7C9976064`.

Actual final pinned-wrapper commands, retained in `go-check.ps1`:

```
test ./app -count=1 -v
vet ./app ./core/plugins
```

Complete app PASS9.819s; actual child exit0/wall11.6184497s. Vet exit0/
1.5439924s, empty stdout/stderr. All54 recorded app/core/composition/example
inputs are unchanged before/after and still match current disk. Result SHA256
`F98BFC4EF9A11E3F031C1BF3FFAAE2FEE8666662C8333FAFC3F603C46274895D`.
Both scripts explicitly forward LASTEXITCODE in final/capture followups.
Worker/older acceptance opt-in cases remain SKIP in the full app run.

Eight new non-opt-in workflow tests actually PASS:

- `TestFormatRoutingNativeModesOutlineRoundTripAndSaveAs`: real two-route
  adapters, configured highlight, semantic outline in preview/source/split,
  exact BOM/CRLF/no-op save, edit+undo+redo, same-format aliases/reopen, unknown
  Mindmap raw metadata, and deliberate `.md` path rematch without conversion.
- `TestFormatRoutingCreationFallbackRecoveryAndReferences`: normalized suffix
  and shipped templates, disabled builtin versus route, disabled/absent
  capability fallback, shadowed first suffix refusal, recovery/session reload,
  alias local reference and native heading anchor.
- `TestFormatRoutingReadonlySceneSaveAsAndDisabledFallback`: real chmod0400
  detected by Open (did not skip), controlled neutral scene result, truthful
  source-only fallback and exact writable SaveAs preserving original bytes.
- `TestFormatRoutingNewFlushReceiptAndInstallModalRecheck`: conflict, recovery
  activation, replacement View, disabled route and successful own-write
  receipts; actual second Flush after native-dialog seam creates a new draft.
- `TestFormatRoutingRealPendingDraftBlocksNewDisableAndInstall`: real routed
  Mindmap tag input, no route/new/install mutation, clipboard recovery oracle.
- `TestFormatRoutingSaveAsPostDialogNativeDraftAndOwner`: real native note
  composition committed once after dialog, exact on-disk note/raw metadata and
  undo; real pending tag refusal, replacement/closed/recovered/reconfigured
  owners publish no file and keep current owners untouched.
- `TestFormatRoutingPostSaveLoadFailureReconcilesButPersistenceFailureRetains`:
  real removed/corrupt files trigger reconcile; settings write refusal keeps
  the live route/view. Added core sentinel test actually PASS0.140s in the first
  raw log, including errors.Is through a host wrapper.
- `TestFormatRoutingSameIDRefreshRejectsLateSceneAndExport`: bounded controlled
  runner ignores cancel until joined; old scene callback/result is cleared,
  fresh generation completes, stale export opens no dialog or destination.

Visual gate:
`artifacts/acceptance/format-routing-app-20261109-104832-capture03/`.
`TestFormatRoutingPluginWorkflowCaptures` actual PASS1.622s, exit0; light/dark
configuration identity and target plus full button within panel clip; actual
button clicks create the selected route. Three760x480 PNGs were inspected:

- `captures/routing-plugins-light-760x480.png`
- `captures/routing-plugins-dark-760x480.png`
- `captures/routing-document-taskmd-760x480.png`

Capture inputs equal the final full-app inputs. Result SHA256
`A1CE9DB774A2436D42BB0D640333EC38BE539D26D6BB8FA51C021380E0AFDD42`.

Failure history remains explicit. Original103224-attempt01 raw log passed
all workflows but failed both captures because wheel x748 was outside the
padded scroll content. Its result.json wrongly claimed exit0 because its
wrapper did not propagate LASTEXITCODE; immutable result is corrected by the
separate `evidence-notes.md`, not rewritten. Capture02 (103824) correctly
records exit1: light succeeded, dark action was below the scroll clip despite
being above status. Capture03 tightened the helper to the actual padded panel
bottom438 and wheel x700; both real clicks passed. No product change or failed
oracle suppression was needed. Preliminary direct checks are disclosed in
attempt01/preliminary-checks.txt; final actual raw streams supersede them.

Source: app/app.go, plugins.go, view.go, format_routing_test.go;
core/plugins/registry.go + schema3_test.go sentinel followup;
examples/plugins/task-notes.json and thoughts.json. Final SHA256 values are
all stored in final04/source-after.json and aggregate.CurrentInputs.
Native capabilities still belong to `modules.NativeRouteAdapter`; schema3 is
a local configuration for shipped formats, not an independently installed
editable engine or executable plugin.
