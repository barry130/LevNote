# Package store — source checkpoint

2026-11-09. Store source and focused filesystem tests/vet are complete; see
`fixed-engine-package-store-result.md` for actual results and frozen evidence.
No Runner/app integration or imported process launch.

Public API matches `fixed-engine-package-core-checkpoint.md`: NewPackageManager,
ImportFromReleaseDir, Installed, Activate, Reset, RepairResetAll, Snapshot and
LoadError. Import does not activate. Installed returns healthy copied receipts
plus identified package errors. Nil contexts/uninitialized managers are refused.

Actual files: `package_types.go`, `package_files.go`, `package_verify.go`,
`package_state.go`, `package_store.go`, `package_windows.go`, `package_other.go`.
GLM retains the two `package_inventory*` files; Runner/bundle files are unchanged.

Private integration seams:

- `selection(engine) (selectedPackage, error)` atomically reads source, digest,
  opaque version, private host/store roots and load error under the small lock.
- `verifiedInstalled(ctx,engine,digest) (*heldPackage,error)` revalidates exact
  stored18files and digest. Its `manifest`, `receipt`, `dir`, `exe` and held
  `files`/directory guard remain owned until `close()`. Future Runner consumes
  this private held result, never a public executable-path override.
- Separate context-aware `gate` serializes filesystem operations; Snapshot and
  LoadError only take the short live-state lock. Startup reads <=64KiB selection
  metadata, never hashes installed executables. Corrupt selection data remains
  on disk and blocks selection/Activate/Reset until explicit RepairResetAll.

Package identity is SHA256 of ASCII `LevNoteEnginePackageV1\x00`, u32BE count,
then lexicographically sorted stored UTF-8 paths, each u16BE path length + path
bytes + u64BE size + raw32byte SHA256. Exactly manifest, executable, original
release inventory and15required notices participate. Inventory is preserved
unchanged and remains the original release inventory, not publisher authentication.

Files are bounded before read/copy; hash/copy uses32KiB chunks. Directory/file
handles reject reparse/type mismatch and regular-file hard links. Ancestors stay
held while accessed. Staging cleanup is nonrecursive and only visits paths this
import created. Its own handles close for the final rename/cleanup; no hostile
filesystem or administrator resistance is claimed for that window. The retained
store root and engine directory prevent ordinary ancestor replacement. This is
a single-Manager serialized store, not cross-process transactions: simultaneous
independent managers are not promised a combined12package quota or merged state.

Selection-file atomic rename is the commit point. Live state updates immediately
after successful rename. A subsequent handle-close cleanup error reports
FailureCleanup with the new live/disk selection retained; callers must not treat
every returned error as proof of rollback. Package publication similarly keeps a
committed new directory installed (never activated) if cancellation comes later.

Completed finite checks use small inert executable bytes and all three fixed IDs:
exact18files/receipt copies/digest, import-only/activation/reset/reopen, corrupt
selection and write refusal, independent source/hash/cap/link errors,12package
limit+exact reuse, missing selected package, cancellable gate/copy+exact cleanup.
