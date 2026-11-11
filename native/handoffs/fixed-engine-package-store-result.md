# Fixed engine package store — focused result

2026-11-09. Implementation and scoped validation complete; root source/evidence
acceptance pending. This is a source-only store slice, not shipped or connected
to the Runner/app. Accepted .8/.7 packages and prior evidence remain unchanged.

## Source and behavior

New owned production files in `core/engineruntime/`: `package_types.go`,
`package_files.go`, `package_verify.go`, `package_state.go`, `package_store.go`,
`package_windows.go`, `package_other.go`; new tests `package_store_test.go`.
No existing Runner/bundle helper was changed. GLM owns the separate inventory
decoder/test, consumed at its frozen hashes from
`fixed-engine-package-inventory-result.md`.

The API and private seams are in `fixed-engine-package-store-checkpoint.md`.
Imports verify/copy exactly18 fixed files, preserve the original release inventory
and all15notices, and do not activate. Content identity includes actual inventory
bytes and all17other files; an independent authored binary digest vector is tested.
Same-package reuse revalidates bytes. The12-per-engine limit refuses a new package
without deleting old versions and still permits exact reuse. Installed scans
return healthy owned receipts alongside identified corrupt/missing selection
errors. Activation re-verifies actual contents, with no verification cache.

Selection persistence is bounded strict JSON and atomic. Corrupt preferences
remain untouched and block selection/ordinary Activate/Reset. Explicit
RepairResetAll replaces all three choices with bundled while retaining installed
packages. Failed writes leave live and disk state unchanged. Successful rename
is the commit point: a later cleanup error is FailureCleanup and does not roll
back the committed selection. Future app handling must inspect that outcome
truthfully rather than treating every error as an uncommitted operation.

The context-aware operation gate is independent from the short selection lock.
Snapshot/LoadError remain responsive during a blocked copy. Local paths and
held ancestors/files reject reparse/type mismatch; regular-file hardlinks are
refused. All declarations are capped before unbounded work, reads retain hard
limits, and copy/hash use32KiB chunks. Staging cleanup is nonrecursive and names
only files/directories this import created. Its handles close for the final
rename/cleanup: no hostile filesystem/admin guarantee is claimed for that
window. This is one PackageManager's serialization, not cross-process state
merging or a strong multi-manager combined quota. Non-Windows entry returns
explicit unsupported behavior; no non-Windows toolchain gate was executed.

## Actual tests and evidence

All commands use `scripts/go.ps1 -GoArguments @(<explicit array>)`; each evidence
`go-check.ps1` exits `$LASTEXITCODE`. `run.ps1` captures real raw stdout/stderr,
process exit codes, times, and18 scoped input fingerprints before/after.

1. `artifacts/acceptance/fixed-engine-package-store-20261109-131851-attempt01/`
   - `test ./core/engineruntime -run ^TestPackageStore -count=1 -v -timeout 60s`
     —10top-level cases PASS; package3.145s; command4.925095s; exit0.
   - `vet ./core/engineruntime` —empty output;1.2336001s; exit0.
   - result PASS,18inputs unchanged.
2. `artifacts/acceptance/fixed-engine-package-store-20261109-132244-followup02/`
   - Tests only
     `^TestPackageStore(CancellationPublicationFailureAndNonblockingSnapshot|InstalledKeepsHealthyChoicesBesideCorruptSelection)$`
     —2cases PASS; package1.024s; command2.8455141s; exit0.
   - vet —empty output;1.090057s; exit0.
   - result PASS,18inputs unchanged. Production source is identical to attempt01;
     only the enhanced waiting-cancellation oracle and new partial-list case
     changed. The previous unaffected nine cases were not repeated.

The combined gate covers11distinct top-level TestPackageStore cases. Subtests
exercise all three IDs, exact copied bytes/18names, independent digest vector,
receipt/snapshot ownership, import-only, activation/reset/reopen, missing license,
hash/length/manifest mismatch, corrupt exact reuse, extra installed files,
inventory/manifest/exe/notices/state caps, bounded writes, cancellation while
copying and waiting at the gate, publication failure cleanup, nonblocking UI
metadata, malformed/corrupt state and failed/successful repair, persistence
refusal,12package quota/exact reuse, missing selected package with no fallback,
healthy choices beside a corrupt selected package, and explicit input refusal.

Hardlink refusal actually PASS. File and directory symlink tests explicitly
SKIP because this Windows session lacks the required privilege. No symlink test
success, OS sandbox, imported executable execution or real app integration is
claimed. The fixture exe files are inert text and never launched. No failed
store test run occurred. The inventory owner's preserved no-tests invocation
remains separate and is not counted as store validation.

Exact source/result/stream digests are recorded in the adjacent
`fixed-engine-package-store-fingerprints.json`, computed from current files.
The evidence before/after JSON and original streams are immutable. No full
runtime/process/resource suite, application build, package or launch was run.
