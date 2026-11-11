# Local fixed-engine packages — next design

Status: DESIGN ONLY, 2026-11-09. No implementation, product/test/script edit,
test execution, build, process launch, package mutation, network research, or
child delegation occurred. Current .8 packaging inputs remain untouched.

## Recommendation

Build a separately managed **local fixed-engine package** boundary for
the three existing repository engines:

- `levnote.markdown-export`
- `levnote.mindmap-svg`
- `levnote.mindmap-preview`

Do not add a fourth engine, a generic plugin ABI, scripts, commands, unknown
capabilities, or new engine IDs. This slice should let the user import one of
those fixed engines from a local complete release directory into a receipt,
explicitly activate an exact digest, and explicitly reset to bundled. It does
not claim marketplace, network, publisher authentication, OS sandboxing, or a
resource reduction.

This is the next useful slice because it makes the existing one-shot engine
packages independently managed without pretending the host has been replaced.
The optional Mindmap rename worker remains **not activated**; editable parsing,
outline, layout and ordinary editing stay in the host.

## Package format and destination

Use a content-addressed installed package directory under the app’s private
state root, for example `%APPDATA%/LevNoteNative/engine-packages/`:

```text
engine-packages/
  selection.json
  receipts.json
  levnote.markdown-export/
    <packageSHA256>/
      manifest.json
      levnote-markdown-export.exe
      inventory/release-manifest.json
      assets/
      notices/
  levnote.mindmap-svg/
    <packageSHA256>/
      manifest.json
      levnote-mindmap-svg.exe
  levnote.mindmap-preview/
    <packageSHA256>/
      manifest.json
      levnote-mindmap-preview.exe
```

`selection.json` records only the active source per engine: `bundled` or an
exact 64-hex installed package digest. `receipts.json` is bounded package
inventory metadata; it is not authentication. Both files are strict, size-bound
and read before allocation. Old packages are retained, never deleted or
overwritten. Manifest version remains an opaque string and never becomes a
path.

The package digest is a canonical binary SHA-256 over every copied file:
`u32be(fileCount)` followed, in the fixed allowlist order, by
`u16be(pathLength) + path`, `u64be(size)`, and the file’s raw SHA-256. It is
never string concatenation and never uses the engine version. The final
directory name is the 64-hex digest.

The copied-file allowlist is fixed by the compiled catalog and release
conventions:

- `manifest.json` and the engine executable;
- `release-manifest.json` preserved unchanged as copied release inventory;
- the existing 13 portable source-asset copies from the current packaging
  definition, when present;
- `THIRD_PARTY_NOTICES.md` and `README.md`;
- `licenses/` files listed by the selected release inventory; and
- `attribution/` files listed by that inventory.

The importer verifies only the selected engine and this fixed notice allowlist.
It does not hash unrelated engines or accept arbitrary release paths. Bounds:
manifest 16 KiB, executable 64 MiB, release inventory 2 MiB, `selection.json`
and `receipts.json` 64 KiB each, at most 256 copied files total, and at most
2 MiB total notice bytes. Every bound is checked before allocation.

Import source is a user-selected **complete release directory** containing
`release-manifest.json` and the chosen engine’s `manifest.json` plus
executable. The importer validates:

1. the release manifest’s strict schema and platform fields;
2. the selected engine entry’s path, size, and SHA-256;
3. the engine manifest against the compiled catalog;
4. the executable’s size and streaming SHA-256; and
5. every allowlisted notice/inventory file against the release inventory; and
6. that a final content-addressed destination either does not exist or contains
   exactly the same verified bytes.

Only the fixed allowlist is copied. The importer rejects symlinks, reparse
points, path escapes, unknown capabilities, platform or protocol mismatch,
filename mismatch, size/count/total bounds, and collisions. Every source
file and every destination ancestor is a checked normal local disk path.

Import opens held validated handles, streams in 32 KiB chunks into a fresh
private staging directory, verifies all copied bytes, syncs the files, then
exclusively renames staging to the content-addressed final directory. It never
overwrites an existing final package. Cancellation/failure leaves the active
selection unchanged and cleans only the exact staging paths it created;
cleanup/sharing violations are visible. An existing final package may be
reused only after complete verification.

## Local trust and user consent

Trusted facts are:

- the compiled engine ID, format, protocol, platform, filename and limits;
- the local manifest and executable digests; and
- the release manifest’s local consistency.

Not trusted:

- the directory name;
- the manifest’s claimed version;
- any “local” or “first-party” label in a file; and
- the absence of a signature.

Import alone does not authorize execution. The UI must disclose that
subsequent preview/export runs this code and require two explicit product-UI
choices: first import the local release, then activate the exact digest. For
example: “导入仅保存文件；激活后，此引擎将在导出/只读预览中执行本地代码。清单一致性不代表发行者身份验证或沙箱。”

## Concrete API

Add a small package manager beside the runner in `core/engineruntime`:

```go
type PackageSource string
const (
    PackageBundled   PackageSource = "bundled"
    PackageInstalled PackageSource = "installed"
)

type PackageStatus struct {
    Engine  enginecatalog.EngineID
    Source  PackageSource
    Version string
    Digest  string
}

type PackageReceipt struct {
    Engine  enginecatalog.EngineID
    Digest  string
    Version string
    Files   []PackageFile
}

type PackageManager struct{ /* private state */ }

func NewPackageManager(hostDir, stateDir string) (*PackageManager, error)
func (m *PackageManager) ImportFromReleaseDir(
    ctx context.Context, releaseDir string, engine enginecatalog.EngineID,
) (PackageReceipt, error)
func (m *PackageManager) Installed(engine enginecatalog.EngineID) []PackageReceipt
func (m *PackageManager) Activate(
    ctx context.Context, engine enginecatalog.EngineID, digest string,
) error
func (m *PackageManager) Reset(ctx context.Context, engine enginecatalog.EngineID) error
func (m *PackageManager) Snapshot() []PackageStatus
```

`ImportFromReleaseDir` only installs and returns a verified receipt; it does
not change the active selection. `Installed` exposes every installed digest so
the UI can switch or roll back. `Activate(engine,digest)` persists that exact
digest as active. `Reset(engine)` persists bundled. No method exposes a generic
caller-supplied executable path. A failed activation or reset keeps the old
live selection; imported inactive bytes may remain and are shown as retained.
There is no promise of atomic rollback across two different filesystem
objects.

Selection persistence uses `document.WritePrivateState`. A persistence failure
keeps the old live selection and installed package. No document or schema3
suffix routing changes.

## Runner resolution

The app owns the package manager and constructs the runner/lane with it. The
runner does not expose a generic path override.

For each request, the runner:

1. captures a selection snapshot on the UI thread at request admission;
2. if selected installed and the package is missing or corrupt, returns a
   visible `FailureUnavailable` or `FailureIntegrity` with no fallback; and
3. verifies that exact resolved directory with the current lock/hash logic
   before starting the process.

There is **no silent fallback** to a different version. If the installed
package is missing or corrupt, the operation refuses and the UI says the
selected engine package is unavailable. Reset-to-bundled is a user action.

Switching while work runs must:

- mark a switch pending and reject all new engine work;
- asynchronously cancel and join the existing single app-owned `engineLane`;
- invalidate export/preview owner generations;
- persist the new selection;
- recreate the runner and lane from the persisted selection; and
- only then allow new work.

The lifecycle contract lives in the app layer, not in `enginecatalog` or the
runner. The switch uses a bounded background job and UI completion, so the UI
never blocks on a worker. On persistence failure, recreate from the old valid
selection. Preserve cleanup poison: a failed cleanup must not be revived by
recreating a lane. Drafts remain in their views; switching package source does
not reconfigure adapters or discard view state. A readonly scene should be
explicitly refreshed/re-requested after the new lane is active, and stale
scene results remain invalid.

## UI workflow

Add a compact “固定引擎” panel to the existing settings/plugin area:

1. List the three fixed engines with current source and version.
2. For each engine, offer:
   - **导入本地发布目录** (choose a complete release directory);
   - list every installed digest and **激活** an exact digest;
   - **恢复随包**.
3. Import opens a local directory picker only; no marketplace or network.
4. The UI shows one explicit local-code-execution warning before import.
5. A failed import or switch leaves the old selection visible.

## Tests and one real smoke

Focused tests:

- reject unknown engine IDs, wrong format/protocol/platform/filename;
- reject links/reparse, path escape, collision, oversized file, digest mismatch;
- import produces a receipt without changing the active selection;
- activate an exact digest and fail without changing the old live selection;
- selection persistence failure leaves the old live selection;
- list every installed digest and support switching/rollback;
- missing or corrupt installed package refuses without fallback;
- reset returns to bundled;
- switch while a lane task runs drains/cancels and joins before activation,
  rejects new work while pending, and preserves cleanup poison.

One exact real package smoke:

- import `levnote.markdown-export` from a real complete release directory,
  select installed, run one real Markdown HTML export, verify the exact
  result bytes and identity, then reset to bundled and verify the bundled
  path is used again.

## Staged file plan

1. **Core package manager**
   - `native/core/engineruntime/package_manager.go`
   - `native/core/engineruntime/package_manager_test.go`

2. **Runner selection integration**
   - `native/core/engineruntime/runner.go`
   - `native/core/engineruntime/bundle.go`

3. **App lifecycle and UI**
   - `native/app/engine_package.go`
   - `native/app/view.go`

4. **Docs**
   - `native/portable/README.md`

Likely owner split: engine/runtime owner for `core/engineruntime/**`, app owner
for `app/**`, and docs owner for `native/portable/README.md`. No plugin/schema3
or document routing owner is needed.

## Risks

- A local release directory still is not provenance; only the user’s explicit
  import action authorizes execution.
- Switching must drain the single lane correctly, or a task could observe a
  half-changed selection.
- One installed package per engine keeps the model small, but replacing an
  installed package must stay atomic.
- This slice affects exports and readonly scene only; it does not reduce
  host parsing, outline, or editing resource use.
