# Engine package volume-root fix — focused result

2026-11-09. DeepSeek implemented the root-confirmed cross-volume correction in
`native/core/engineruntime/package_files.go`. `packageGuard.lockDir` now opens
and validates each existing ancestor, including the volume root, before deciding
whether creation is legal. It attempts `os.Mkdir` only after the current path
returns a genuine `os.ErrNotExist`, only when `create=true`, and only below the
volume root. The created directory is reopened through the same reparse,
non-directory, identity and handle-validation path. Access-denied and invalid
existing components are not treated as absent directories.

The change is limited to:

- `native/core/engineruntime/package_files.go`
  - prior accepted SHA256:
    `BCA39FF8FF78DAD77E5AEC06137B25B18F1FAEE1544D0B8D332CE08BD25A4CC4`
  - fixed SHA256:
    `BC7A4254A62F4A2E21610EC35EA64CB73951FCB6367EE12A468DA24A3447CA05`
  - 6,632 bytes
- `native/core/engineruntime/package_guard_windows_test.go` (new)
  - SHA256:
    `3B7A12F49B2B06BBA2FC9A2F39D1645FB95BF2DF0E1B372DFCA668D9BDD9EBF7`
  - 3,709 bytes

Raw evidence is under
`native/artifacts/acceptance/engine-package-volume-root-fix-20261109-190635`.
The first regression run against the old behavior is retained as
`red-old-behavior/` with exit 1: both creating below the existing `C:\` volume
root and `NewPackageManager(host F:, state C:)` failed with
`创建引擎包目录失败`. The final focused run is `focused-green-final/`, exit 0:

- `TestPackageGuardCreatesBelowExistingVolumeRoot`
- `TestPackageGuardRefusesMissingWithoutCreateAndFileAncestor`
- `TestNewPackageManagerCrossVolumeHostFStateDefaultC`

The final affected set
`test ./core/engineruntime -run '^(TestPackage|TestManaged)' -count=1 -v`
passed in 6.787s; the opt-in real managed smoke remained skipped without its
environment. Final `vet ./core/engineruntime` passed with the wrapper's recorded
wall duration of 0.591s. Exact wrapper arguments, stdout, stderr and exits are
preserved in `package-managed-tests-final/` and `scoped-vet-final/`. Hardlink
protection passed; the symlink-file and symlink-dir fixtures were skipped
because this Windows account lacks symlink privilege. Those two protections
are not claimed as exercised.

An intermediate `index > 0` creation-first patch also passed focused tests, but
root review found it could trigger creation against an existing ACL-denied
ancestor. That implementation is superseded by the final open-first logic. Its
integer `format`, `focused-green`, `package-managed-tests` and
`scoped-vet` stages remain unchanged as history. A first format-wrapper attempt
that received no arguments because a local function used PowerShell's automatic
`$args` variable is also retained under `format/`.

No TEMP/APPDATA volume was moved. `.8` remains a historical accepted release;
failed `.9` and its evidence remain immutable. Root accepted the completed
`.10` delivery as current in
`fixed-engine-package-delivery-root-acceptance.md`.
