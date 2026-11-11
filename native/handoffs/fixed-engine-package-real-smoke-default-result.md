# Managed real smoke default-SKIP compile gate

2026-11-09. GLM completed the exact review corrections and the one authorized
default-SKIP gate for its owned
`core/engineruntime/managed_bundle_test.go`. No opt-in smoke execution,
tester/host/worker build, child process, worker launch, or application launch
occurred.

## Corrections applied

- `runnerClosed` cleanup now appears only after `NewManaged` succeeds, so the
  package compiles.
- Manifest JSON size is checked with `len(manifestBytes) == manifestEntry.Size`;
  the manifest's executable-size field is compared only with the executable
  inventory entry.
- Metadata guard cleanup uses a named return and joins its close error with the
  original result error; cleanup failure is never swallowed.
- Both source release and fresh host inventories must list and hold the
  ordinary `levnote-native.exe`, and its actual stat size must match the
  inventory before either directory is treated as a complete release.
- One final JSON logging `t.Cleanup` is registered immediately after record
  creation. It runs after the LIFO runner cleanup fallbacks, derives status
  from `t.Failed()` and retained failures, and preserves evidence after a fatal
  path. It logs exactly once.
- Each runner now records normal close called/error and fallback close
  called/error in separate fields. Fallback close failures also append to the
  record failures.

## Actual gate

Evidence:
`handoffs/evidence/fixed-engine-package-smoke-default-20261109-065958/`

- `scripts/go.ps1 test -run ^TestManagedRealOldReleaseThenBundledResetSmoke$ ./core/engineruntime`
  with `LEVNOTE_MANAGED_REAL_SMOKE` and
  `LEVNOTE_MANAGED_REAL_SMOKE_RELEASE` cleared — exit 0,
  `ok levnote.local/native/core/engineruntime 0.599s`. This proves default
  `SKIP` and compilation, not two-request runtime success.
- `scripts/go.ps1 vet ./core/engineruntime` — exit 0.

## Frozen source

```text
core/engineruntime/managed_bundle_test.go
47e0162df75f73e2146469aabf24bf5c3f976e99a18bfa55cb8c7a913fca0a90
```

The opt-in test remains reserved for a later root-accepted delivery execution.
