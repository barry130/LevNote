# Fixed engine package core — independent source review

2026-11-09. Read-only review. Sources read once: `package_types.go`,
`package_state.go`, `package_files.go`, `package_verify.go`, `package_store.go`,
`package_inventory.go`, `package_windows.go`, the shared Windows
`openLockedFile`/`finalPath` helpers, and `fixed-engine-package-core-activation.md`.
No tests or builds were run, so this records source inspection rather than proof.

## Result

**None found** within the requested correctness slice. I found no concrete
state-loss/state-change, wrong-package-bytes, improper-resource-leave,
normal-import failure/hang, or UI-metadata-blocking defect in the current sources.

## Guard order checked

- **Selection commit:** `PackageManager.persistState` writes and syncs a fresh
  temporary file, checks cancellation, atomically replaces `selections.json`, then
  updates `statuses` and clears `loadErr` under `mu`. `Activate` verifies the
  installed package before calling it. A failed write/rename leaves the live and
  persisted selection unchanged; a post-rename cleanup failure can report an error
  after commit, which matches the disclosed boundary. `Reset` uses the same path;
  corrupt startup state blocks activate/reset until `RepairResetAll`.
- **Package identity and publication:** `verifyPackage` requires the fixed
  18-file set, hashes actual bytes against the release inventory, checks the engine
  manifest/exe identity, and computes the package digest. `ImportFromReleaseDir`
  copies through bounded streams into a fresh staging directory, verifies that
  staged copy again, closes handles needed for mutation, and publishes with a
  non-replacing rename. An existing destination is accepted only after independent
  verification of its actual files. I found no path where differently valued bytes
  become the receipt for the published digest.
- **Cleanup and UI reads:** `packageStaging.cleanup` is nonrecursive, verifies that
  its target is the import's own `.import-*` child, deletes only recorded
  files/directories, and does nothing after publication. Its guard is closed before
  deletion, matching the staged-cleanup boundary. `Snapshot` and `LoadError` use
  only the short metadata `RWMutex`, while bulk import/verification uses the separate
  cancellable operation gate; large I/O does not hold the metadata lock.

The disclosed single-manager serialization, non-adversarial filesystem limits,
post-commit cleanup ambiguity, and in-progress path-parser corrections are not
counted as new findings. No additional test is proposed: the current commit-order,
staging, verification, and metadata-read paths are directly visible in source, and
an added test would not resolve an identified unverified behavior in this slice.
