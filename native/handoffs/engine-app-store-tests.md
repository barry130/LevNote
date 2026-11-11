# Native app store tests — frozen source

2026-11-09. DeepSeek source-only slice. No product source changed, no agent
delegation, and no tests/build/vet/processes were run while GLM owns app
lifecycle source.

## Files

- Added `native/app/engine_packages_store_test.go` only.
- Added this handoff.

Frozen source SHA256:
`8F1D5BB8B31F59815B3435737180DEB3F437679C9A5D755BF7817E334DDD04A2`.

## Shared fixture helper

Exact name and signature for GLM or later app tests:

```go
func enginePackageFixture(t *testing.T, engine enginecatalog.EngineID, version string) string
```

It returns a fresh absolute release directory containing the catalog manifest
and executable, all fifteen required authored/attribution files, and
`release-manifest.json`. The executable is inert text and is never executed.
The test-owned notice list mirrors the fixed eighteen-file stored receipt
(`release-manifest.json`, `manifest.json`, executable, fifteen notices).

The file is Windows-tagged to match the real Windows-only package store.
GLM lifecycle tests that call the helper should carry the same build constraint.

## Tests written

- `TestEnginePackageStoreImportActivateResetAndRestart` imports all three fixed
  engines through `App.startPackageWork`/`deliverPackageWork`, proves import is
  inert, keeps the cached receipt available, activates/resets only the selected
  engine, and verifies a new `App` restores the persisted choice.
- `TestEnginePackageStoreListReturnsHealthyPackagesAndCorruption` imports two
  distinct Markdown packages, corrupts one stored notice, and proves list
  delivery retains the healthy receipt while exposing the corrupt package
  through `packageLists[...].err` and the outcome.
- `TestEnginePackageStoreCorruptStateStaysBlockedUntilRepairRetainsPackage`
  starts from corrupt selection state, proves ordinary activation fails without
  changing the corrupt file or falsely changing status, then runs explicit
  ALL3 repair and verifies the imported package remains listed.

All waits use `packageJob.done` with a bounded ten-second timeout. No real
engine, OS UI, resource measurement, or fake successful manager is used.

## Verification status

Source is complete and `gofmt`-clean. **Results are not yet run**; this handoff
does not claim PASS, compile success, or test execution. The intended later
focused command is the `TestEnginePackageStore` prefix in `native/app` after
GLM's source is stable.
