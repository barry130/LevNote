# Managed engine app source review — read-only

2026-11-09. DeepSeek owns this read-only review after Astra's first app source
slice. Do not edit Go/production/tests, do not run tests/build/processes and do
not spawn children. Read native/AGENTS.md current top, fixed-engine-package-
app-activation.md, then current app/app.go, engine_packages.go,
engine_packages_ui.go, view.go, preview_worker.go, export_worker.go,
ui/editor/scene.go and any focused tests already present.

Report at native/handoffs/fixed-engine-package-app-source-review.md. Limit to
at most 5 concrete defects with file/function and a minimal correction. Focus:

* manager initialization uses os.Executable/configDir without blocking frames;
* one bounded package operation/result slot, cancellation and shutdown joins;
* transitions capture the original lane/runner/job and cannot cancel/release a
  new lane; cleanup poison and postcommit selection semantics remain visible;
* managed runner is used for subsequent export/readonly preview and bundled
  fallback is not silent; EngineInfo source/digest reaches scene/status;
* modal dialog owner revalidation, readonly SceneRefresher only, writable draft
  and undo preservation, and close/CanClose refusal behavior;
* panel actions are keyboard accessible, scrollable at 760x480, distinct import
  vs activate/reset/ALL3 repair, and do not scan package bytes per frame.

Treat known single-manager and store stage-close limits as already disclosed,
not fresh findings. If no concrete issue, say so and identify any uncertainty.
Do not claim tests or builds. Root will route fixes to Astra after reading.
