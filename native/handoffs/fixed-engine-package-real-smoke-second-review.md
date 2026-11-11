# Smoke test — remaining exact corrections and default gate

2026-11-09. GLM remains sole owner of managed_bundle_test.go. Your reported
first review completion is not borne out by the file. Fix these exact issues:

- runnerClosed/t.Cleanup currently appears immediately after Import, before
  `runner` is declared. Move that block immediately AFTER NewManaged succeeds.
  It currently makes the entire package fail compile (`undefined: runner`).
- Metadata compares `manifest.Size != manifestEntry.Size`. manifest.Size is the
  EXECUTABLE length; manifestEntry.Size is JSON length. Compare actual
  len(manifestBytes) to manifestEntry.Size, plus existing executable consistency.
- Metadata deferred close deliberately discards its error (`_ = closeErr`).
  Use a named returned error joined with close error, or explicit structured
  closes; preserve both and never hide cleanup failure.
- Require ordinary current host levnote-native.exe existence in fixture before
  assuming tester is beside a complete release. Current helper never checks it.
- Register one final JSON logging cleanup when record is made, before fallible
  setup; it runs after runner cleanup (LIFO). Use t.Failed() to set status. Normal
  and fallback close outcomes of BOTH runners need explicit fields; cleanup
  errors append to failures. The current end-only log is lost on Fatal, and one
  empty shared error cannot prove both joins. Do not duplicate log output.

After source correction, you are NOW authorized for one exact DEFAULT-SKIP
compile/test of `^TestManagedRealOldReleaseThenBundledResetSmoke$` with opt-in
variables cleared, then `vet ./core/engineruntime`. Use go.ps1 argument array,
real raw stdout/stderr/exit records and fresh small evidence dir. No actual
smoke, tester/release build or child process. Fix any compile issue and preserve
the failed invocation. Publish actual default-SKIP source result and notify
Astra only once your source is stable. Root will read source+actual evidence.

Astra should avoid another runtime followup until this default-SKIP compile
passes. Its app tests use a different package and can proceed. This gate is
source compilation, NOT two-request runtime success; do not mislabel it.
