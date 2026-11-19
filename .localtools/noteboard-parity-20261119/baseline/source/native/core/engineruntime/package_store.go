package engineruntime

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"levnote.local/native/core/enginecatalog"
)

func packageRandomName(prefix string) (string, error) {
	var entropy [16]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		return "", packageIOError("创建引擎包唯一临时名称失败", err)
	}
	return prefix + hex.EncodeToString(entropy[:]), nil
}

type packageStaging struct {
	root, path  string
	files, dirs []string
	guard       packageGuard
	published   bool
}

func newPackageStaging(ctx context.Context, root string) (*packageStaging, error) {
	name, err := packageRandomName(".import-")
	if err != nil {
		return nil, err
	}
	s := &packageStaging{root: root, path: filepath.Join(root, name)}
	if err := os.Mkdir(s.path, 0700); err != nil {
		return nil, packageIOError("创建引擎包暂存目录失败", err)
	}
	if err := s.guard.lockDir(ctx, s.path, false); err != nil {
		return nil, errors.Join(err, s.cleanup())
	}
	return s, nil
}

// Cleanup is deliberately nonrecursive and visits only paths this import
// created. Never follow a computed recursive remove or erase foreign entries.
func (s *packageStaging) cleanup() error {
	closeErr := s.guard.close()
	if s.published {
		return closeErr
	}
	abs, err := packageAbsolute(s.path)
	if err != nil || abs != s.path || filepath.Dir(abs) != s.root || !strings.HasPrefix(filepath.Base(abs), ".import-") {
		return failure(FailureCleanup, "拒绝清理非本次引擎包暂存目录", err)
	}
	errs := []error{closeErr}
	for i := len(s.files) - 1; i >= 0; i-- {
		if err := os.Remove(s.files[i]); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	for i := len(s.dirs) - 1; i >= 0; i-- {
		if err := os.Remove(s.dirs[i]); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	if err := os.Remove(s.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		errs = append(errs, err)
	}
	if err := errors.Join(errs...); err != nil {
		return failure(FailureCleanup, "清理本次引擎包暂存文件失败", err)
	}
	return nil
}

func (s *packageStaging) copy(ctx context.Context, p *heldPackage, hooks *packageTestHooks) error {
	for _, name := range []string{"attribution", "licenses"} {
		path := filepath.Join(s.path, name)
		if err := os.Mkdir(path, 0700); err != nil {
			return packageIOError("创建引擎包署名目录失败", err)
		}
		s.dirs = append(s.dirs, path)
		if err := s.guard.lockDir(ctx, path, false); err != nil {
			return err
		}
	}
	for _, entry := range p.receipt.Files {
		if err := contextFailure(ctx); err != nil {
			return err
		}
		path := filepath.Join(s.path, filepath.FromSlash(entry.Path))
		out, err := packageCreateFile(path)
		if err != nil {
			return packageIOError("创建引擎包副本失败", err)
		}
		s.files = append(s.files, path)
		var progress func(int64)
		if hooks != nil && hooks.afterCopyChunk != nil {
			progress = func(count int64) { hooks.afterCopyChunk(entry.Path, count) }
		}
		digest, copyErr := packageHash(ctx, p.files[entry.Path], entry.Size, out, progress)
		if copyErr == nil && digest != entry.SHA256 {
			copyErr = failure(FailureIntegrity, "复制时引擎包摘要变化", nil)
		}
		if copyErr == nil {
			copyErr = out.Sync()
		}
		closeErr := out.Close()
		if closeErr != nil {
			return failure(FailureCleanup, "关闭引擎包副本失败", errors.Join(copyErr, closeErr))
		}
		if copyErr != nil {
			return packageIOError("复制引擎包失败", copyErr)
		}
		// Keep copied bytes locked until the final complete verification.
		if _, err := s.guard.open(ctx, path); err != nil {
			return err
		}
	}
	return nil
}

func packageDirectoryDigests(path string) ([]string, error) {
	entries, err := packageEntries(path, packageCountLimit)
	if err != nil {
		return nil, err
	}
	digests := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || !validPackageDigest(entry.Name()) || entry.Type()&os.ModeSymlink != 0 {
			return nil, failure(FailureIntegrity, "引擎仓库存在非摘要目录或链接", nil)
		}
		digests = append(digests, entry.Name())
	}
	sort.Strings(digests)
	return digests, nil
}

func (m *PackageManager) ImportFromReleaseDir(ctx context.Context, releaseDir string, engine enginecatalog.EngineID) (receipt PackageReceipt, err error) {
	if _, err := packageDescriptor(engine); err != nil {
		return receipt, err
	}
	if err := m.acquire(ctx); err != nil {
		return receipt, err
	}
	defer m.release()
	var destinationGuard packageGuard
	defer func() {
		if closeErr := destinationGuard.close(); closeErr != nil {
			err = closeErr
		}
	}()
	engineDir := filepath.Join(m.root, string(engine))
	if err := destinationGuard.lockDir(ctx, engineDir, false); err != nil {
		return receipt, err
	}
	digests, err := packageDirectoryDigests(engineDir)
	if err != nil {
		return receipt, err
	}
	p, err := verifyPackage(ctx, releaseDir, engine, false, "")
	if err != nil {
		return receipt, err
	}
	defer func() {
		if closeErr := p.close(); closeErr != nil {
			err = closeErr
		}
	}()
	for _, digest := range digests {
		if digest == p.receipt.Digest {
			existing, verifyErr := m.verifiedInstalled(ctx, engine, digest)
			if verifyErr != nil {
				return receipt, verifyErr
			}
			receipt = copiedPackageReceipt(existing.receipt)
			return receipt, existing.close()
		}
	}
	if len(digests) >= packageCountLimit {
		return receipt, failure(FailureIntegrity, "每种引擎最多保留12个包；没有删除或覆盖旧包", nil)
	}
	stage, err := newPackageStaging(ctx, m.root)
	if err != nil {
		return receipt, err
	}
	defer func() {
		if cleanupErr := stage.cleanup(); cleanupErr != nil {
			err = cleanupErr
		}
	}()
	if err := stage.copy(ctx, p, m.hooks); err != nil {
		return receipt, err
	}
	// Recheck actual destination bytes, including its exact 18-file set, before
	// publication. Close only handles required by the Windows rename; the store
	// root and target engine directory stay held throughout publication.
	check, err := verifyPackage(ctx, stage.path, engine, true, p.receipt.Digest)
	if err != nil {
		return receipt, err
	}
	if err := check.close(); err != nil {
		return receipt, err
	}
	if err := stage.guard.close(); err != nil {
		return receipt, err
	}
	if m.hooks != nil && m.hooks.beforePublish != nil {
		if err := m.hooks.beforePublish(); err != nil {
			return receipt, packageIOError("发布引擎包失败", err)
		}
	}
	if err := contextFailure(ctx); err != nil {
		return receipt, err
	}
	if err := packageRename(stage.path, m.installedPath(engine, p.receipt.Digest), false); err != nil {
		// Another manager may have published the exact package. Never replace it;
		// equality is established by actual files, not existence alone.
		existing, verifyErr := m.verifiedInstalled(ctx, engine, p.receipt.Digest)
		if verifyErr != nil {
			return receipt, packageIOError("独占发布引擎包失败", errors.Join(err, verifyErr))
		}
		receipt = copiedPackageReceipt(existing.receipt)
		return receipt, existing.close()
	}
	stage.published = true
	// Cancellation after atomic rename does not remove a published version. It
	// stays installed without activation, even if the caller cancels UI delivery.
	receipt = copiedPackageReceipt(p.receipt)
	return receipt, nil
}

func (m *PackageManager) Installed(ctx context.Context, engine enginecatalog.EngineID) (receipts []PackageReceipt, err error) {
	if _, err := packageDescriptor(engine); err != nil {
		return nil, err
	}
	if err := m.acquire(ctx); err != nil {
		return nil, err
	}
	defer m.release()
	var guard packageGuard
	defer func() {
		if closeErr := guard.close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}()
	dir := filepath.Join(m.root, string(engine))
	if err := guard.lockDir(ctx, dir, false); err != nil {
		return nil, err
	}
	entries, err := packageEntries(dir, packageCountLimit)
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	statuses, loadErr := m.stateSnapshot()
	errs := []error{loadErr}
	selected := ""
	for _, status := range statuses {
		if status.Engine == engine {
			selected = status.Digest
		}
	}
	seenSelection := selected == ""
	for _, entry := range entries {
		if err := contextFailure(ctx); err != nil {
			return receipts, errors.Join(append(errs, err)...)
		}
		if entry.Name() == selected {
			seenSelection = true
		}
		if !entry.IsDir() || !validPackageDigest(entry.Name()) || entry.Type()&os.ModeSymlink != 0 {
			errs = append(errs, packageIssue(engine, entry.Name(), failure(FailureIntegrity, "安装包目录无效", nil)))
			continue
		}
		p, verifyErr := m.verifiedInstalled(ctx, engine, entry.Name())
		if verifyErr != nil {
			errs = append(errs, packageIssue(engine, entry.Name(), verifyErr))
			continue
		}
		receipts = append(receipts, copiedPackageReceipt(p.receipt))
		if err := p.close(); err != nil {
			errs = append(errs, packageIssue(engine, entry.Name(), err))
		}
	}
	if !seenSelection {
		errs = append(errs, packageIssue(engine, selected, failure(FailureUnavailable, "当前选中的安装包已缺失；未回退随包版本", nil)))
	}
	return receipts, errors.Join(errs...)
}
