# Managed app package result

2026-11-09. The app implementation and final acceptance takeovers are
complete. GLM stopped at a tool boundary after the accepted focused correction;
DeepSeek then finished only the stale capture correction, final wrapper runs,
and this evidence update. No sub-agents were used. Product shell/store/runtime/
graph sources remain frozen.

## Final source

The capture correction is limited to
`native/app/engine_packages_capture_test.go`. It now holds a genuine Reset
transition at the existing `controlledExporter.onClose` seam, hides the engine
panel while that transition is pending, captures the hidden progress row,
checks the progress/reopen/cancel buttons have positive fully visible bounds,
then reopens the panel before releasing the gate and joining/delivering the
job. Its cleanup release is registered after `newTestApp`, so Go runs it before
`App.Shutdown`; timeout paths release it too. Installed scroll checks also
require non-offscreen, nonzero, fully in-window bounds instead of relying on
access-tree presence alone.

Final source SHA256 values:

- `native/app/engine_packages_capture_test.go`:
  `BF30865AC8BADE042CFA1E8F15E88B2134A7E7BF8C38E9C65C6E4272E5C20520`
- `native/app/engine_packages_lifecycle_test.go`:
  `C560EAB6229390F0D2B404BB45AB7FFB1D3C8B2B84699E4A42857ECE9263B2CF`
- `native/app/engine_packages_store_test.go`:
  `8F1D5BB8B31F59815B3435737180DEB3F437679C9A5D755BF7817E334DDD04A2`
- `native/app/engine_packages.go`:
  `664D4F3F9FB9F228E9A825FAE706DEA41D29DC07CDAE767FC108429DBFBA2FFE`
- `native/app/export_worker.go`:
  `A4E31583A7CF78E26E03A19EE7C6CAD2F5BCD7E51EA089764C68D37BA6227CF4`
- unchanged scene source `native/modules/graphview/scene_preview.go`:
  `E3420FC1200B6454014674BBD135284329B7EAEDC146C3EFA17779C3FF8B597B`

The final before/after source inventory covered 117 Go files under
`native/app`, `native/ui/editor`, `native/modules/graphview`,
`native/core/enginecatalog`, `native/core/engineruntime`, and
`native/core/engineipc`. No hash changed during the final capture, full-app,
and vet stages. The comparison is non-vacuous and is recorded in
`native/artifacts/acceptance/fixed-engine-package-app-final-20261109-105040/`.

## Lifecycle coverage

The accepted lifecycle source covers:

- joining and cancelling an old export lane while preserving a writable draft,
  dirty state, and undo history;
- preserving a real uncommitted mindmap tag input, source, history, selection,
  dirty state, and exact view identity across a successful package Reset;
- suppressing a ready stale export and its late callback without releasing the
  new lane reservation;
- refreshing only an owned readonly scene after a transition while leaving a
  separately writable draft untouched;
- retained cleanup poison with diagnostic list/import availability;
- modal deferral, stale-generation suppression, UI-free shutdown joining, and
  close refusal that leaves package work alive.

The peer store file covers real-fixture import, activation/reset/restart,
healthy and corrupt listing, corrupt-state refusal, and ALL3 repair retention.

## Final verification

All final Go commands ran through
`native/scripts/go.ps1 -GoArguments ...` from the repository root. A fresh
timestamped evidence root retains command arguments, environment, UTC start/end
times, duration, exit codes, and raw stdout/stderr.

Focused accepted correction, unchanged in this final pass:

```text
.\scripts\go.ps1 -GoArguments @('test','./app','-run','TestEnginePackage','-count=1','-v')
ok  levnote.local/native/app  2.679s
```

Evidence:
`native/handoffs/evidence/fixed-engine-package-app-final-corrections-20261109-104028/`.
Attempt 1 is retained with exit 1 and the `undefined: tab` compile error;
attempt 2 is the passing raw test stream and exit 0.

Final capture, exact-test execution:

```text
.\scripts\go.ps1 -GoArguments @(
  'test','./app','-run','TestEnginePackagePanelCaptures','-count=1','-v'
)
```

`LEVNOTE_ENGINE_PACKAGE_SCREENSHOT_DIR` pointed at the final image directory.
Exit 0; package result `ok levnote.local/native/app 1.910s`; the test itself
passed in 0.28s. Evidence:
`native/artifacts/acceptance/fixed-engine-package-app-final-20261109-105040/capture-attempt01/`.

Final full app:

```text
.\scripts\go.ps1 -GoArguments @('test','./app','-count=1')
ok  levnote.local/native/app  13.775s
```

Exit 0. Evidence:
`native/artifacts/acceptance/fixed-engine-package-app-final-20261109-105040/full-app-attempt01/`.

Final scoped vet:

```text
.\scripts\go.ps1 -GoArguments @(
  'vet','./app','./ui/editor','./modules/graphview'
)
```

Exit 0 with empty stdout and stderr after 1.036s. Evidence:
`native/artifacts/acceptance/fixed-engine-package-app-final-20261109-105040/vet-attempt01/`.

## Final captures

Exactly four final 760x480 PNGs are in
`native/artifacts/acceptance/fixed-engine-package-app-final-20261109-105040/images/`:

| Image | SHA256 |
|---|---|
| `engine-panel-installed-light-760x480.png` | `F0616112C33C7077ECE5CA044A36D9AD37B72A4F233C32EEE5C2A75A2C5E130F` |
| `engine-panel-installed-dark-760x480.png` | `E96A510A3E51CFD54F95A2887A6ABE204C5E8C5A7CCAC8E6E943BEA89786778B` |
| `engine-panel-pending-hidden-light-760x480.png` | `9265D8BAE8033EF1F1691898C8EA23C97ABB0DB3A63A95473589E304898A6168` |
| `engine-panel-pending-hidden-dark-760x480.png` | `BCD9C0AF85C5B24E889330040F5625095896DEB25C2740861B1AE3B92270675E` |

The installed images show the imported inert package as the active installed
selection and show the installed-package list controls. The pending-hidden
images were saved before reopening: they show the main workspace with the
global Reset progress row, `查看操作`, and `取消`, while the Reset job remains
blocked at the controlled close gate. The test then reopens and observes the
pending panel before releasing the gate.

## Preserved history and limits

- Prior capture attempt04 remains unchanged. Its files named
  `engine-panel-pending-hidden-*` showed a reopened engine-package panel, not a
  genuinely hidden panel; the final images above correct and test that contract.
- Earlier focused evidence under
  `native/handoffs/evidence/fixed-engine-package-app-focused-20261109/`
  contains only the final 37-byte stdout plus metadata; the original
  development failure streams are not present there. They are not recreated or
  claimed as preserved.
- Earlier full-app failures under
  `native/handoffs/evidence/fixed-engine-package-app-full-20261109/` remain in
  place. The final full-app result above did not overwrite them.
- The first focused correction attempt and its compile failure remain in the
  final-corrections evidence directory.

These are MyGo headless tester captures. No physical UI automation, OS IME,
GPU, memory, CPU, resource-sampling, or private process-counter claim was made.

No `.9` candidate, package build, managed real worker, or host launch was run.
`.8` and `.7` remain immutable. Delivery stays stopped until the root owner
reads these final images/result and explicitly activates the prepared `.9`
execution.
