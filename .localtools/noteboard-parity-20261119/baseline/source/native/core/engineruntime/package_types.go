package engineruntime

import (
	"context"
	"sync"

	"levnote.local/native/core/enginecatalog"
)

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

// PackageManager stores only the four fixed catalog engines. The cancellable I/O
// gate is separate from mu: UI metadata reads never wait for a file hash/copy.
// Packages are integrity-consistent local bytes, not publisher authentication.
type PackageManager struct {
	hostDir, root string
	gate          chan struct{}
	mu            sync.RWMutex
	statuses      [4]PackageStatus
	loadErr       error
	hooks         *packageTestHooks
}

// Private deterministic failure seams; no executable or verifier substitution.
type packageTestHooks struct {
	afterCopyChunk    func(string, int64)
	beforePublish     func() error
	beforeStateCommit func() error
}

var packageEngines = [...]enginecatalog.EngineID{
	enginecatalog.MarkdownHTML, enginecatalog.MindmapSVG, enginecatalog.MindmapPreview, enginecatalog.DrawioSVG,
}

var legacyPackageEngines = [...]enginecatalog.EngineID{
	enginecatalog.MarkdownHTML, enginecatalog.MindmapSVG, enginecatalog.MindmapPreview,
}

const (
	packageNoticeLimit = 2 << 20
	packageStateLimit  = 64 << 10
	packageCountLimit  = 12
)

var packageNotices = [...]string{
	"LICENSE", "README.md", "THIRD_PARTY_NOTICES.md",
	"attribution/SOURCE_THIRD_PARTY_NOTICES.txt",
	"attribution/mygo-README.md", "attribution/purego-README.md",
	"attribution/mygo-v0.8.3-no-startup-cleanup.patch",
	"attribution/dependency-patches-README.md",
	"licenses/Go-BSD-3-Clause.txt", "licenses/go-text-typesetting.txt",
	"licenses/mygo-MIT.txt", "licenses/purego-Apache-2.0.txt",
	"licenses/x-image-BSD-3-Clause.txt", "licenses/x-net-BSD-3-Clause.txt",
	"licenses/x-text-BSD-3-Clause.txt",
}

func (m *PackageManager) acquire(ctx context.Context) error {
	if m == nil || m.gate == nil || m.root == "" || ctx == nil {
		return failure(FailureUnavailable, "引擎包仓库或请求上下文不可用", nil)
	}
	if err := contextFailure(ctx); err != nil {
		return err
	}
	select {
	case m.gate <- struct{}{}:
		if err := contextFailure(ctx); err != nil {
			<-m.gate
			return err
		}
		return nil
	case <-ctx.Done():
		return contextFailure(ctx)
	}
}

func (m *PackageManager) release() { <-m.gate }

func (m *PackageManager) Snapshot() []PackageStatus {
	if m == nil || m.gate == nil {
		return nil
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]PackageStatus(nil), m.statuses[:]...)
}

func (m *PackageManager) LoadError() error {
	if m == nil || m.gate == nil {
		return failure(FailureUnavailable, "引擎包仓库未初始化", nil)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.loadErr
}

// selectedPackage is an immutable admission snapshot for the later Runner
// integration. Source, digest, version and roots are not executable overrides.
// Selection and load error are read under the same small lock.
type selectedPackage struct {
	PackageStatus
	hostDir, storeRoot string
}

func (m *PackageManager) selection(engine enginecatalog.EngineID) (selectedPackage, error) {
	if m == nil || m.gate == nil {
		return selectedPackage{}, failure(FailureUnavailable, "引擎包仓库未初始化", nil)
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.loadErr != nil {
		return selectedPackage{}, failure(FailureIntegrity, "引擎包选择状态不可用，请明确重置全部选择", m.loadErr)
	}
	for _, status := range m.statuses {
		if status.Engine == engine {
			return selectedPackage{PackageStatus: status, hostDir: m.hostDir, storeRoot: m.root}, nil
		}
	}
	return selectedPackage{}, failure(FailureUnavailable, "未知的固定引擎", nil)
}

func copiedPackageReceipt(r PackageReceipt) PackageReceipt {
	r.Files = append([]PackageFile(nil), r.Files...)
	return r
}
