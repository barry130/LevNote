# Fixed-engine package core — implementation checkpoint

2026-11-09. GLM implementation is active under
`fixed-engine-package-core-activation.md`; the older design-only gate is no
longer in force. This checkpoint records the concrete API being implemented now;
it is not integration or delivery acceptance.

## Actual public API

The implementation is owned by `native/core/engineruntime`. The source manifest
truth is the accepted `.8` `release-manifest.json`; that package remains
read-only.

```go
type PackageSource string

const (
	PackageBundled   PackageSource = "bundled"
	PackageInstalled PackageSource = "installed"
)

type PackageFile struct {
	Path   string
	Size   int64
	SHA256 string
}

type PackageReceipt struct {
	Engine  enginecatalog.EngineID
	Digest  string
	Version string
	Files   []PackageFile
}

type PackageStatus struct {
	Engine  enginecatalog.EngineID
	Source  PackageSource
	Version string
	Digest  string
}

type PackageManager struct { /* private state and operation lock */ }

func NewPackageManager(hostDir, stateDir string) (*PackageManager, error)
func (m *PackageManager) ImportFromReleaseDir(ctx context.Context, releaseDir string, engine enginecatalog.EngineID) (PackageReceipt, error)
func (m *PackageManager) Installed(ctx context.Context, engine enginecatalog.EngineID) ([]PackageReceipt, error)
func (m *PackageManager) Activate(ctx context.Context, engine enginecatalog.EngineID, digest string) error
func (m *PackageManager) Reset(ctx context.Context, engine enginecatalog.EngineID) error
func (m *PackageManager) RepairResetAll(ctx context.Context) error
func (m *PackageManager) Snapshot() []PackageStatus
func (m *PackageManager) LoadError() error
```

`Snapshot` and `LoadError` copy the small live-selection metadata under a short
lock and never wait behind package I/O. Import/Installed/Activate/Reset/repair
operations use a separately cancellable operation gate. `Installed` can return
healthy receipts together with an error identifying corrupt packages; a single
bad directory never hides every other usable installed choice.

`ImportFromReleaseDir` imports without activation. `Installed` returns verified
receipts and does not silently omit corrupt packages. Activation verifies the
exact content-addressed package before changing state. A corrupt selection
state yields a usable manager with `LoadError() != nil`; ordinary
activate/reset is refused until `RepairResetAll` succeeds.

Only this package-store source/tests and its evidence/result handoffs are in
scope. Runner, app, workers, catalog, scripts and package inputs remain
unchanged.
