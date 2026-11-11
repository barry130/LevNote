package engineruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"levnote.local/native/core/enginecatalog"
)

type heldPackage struct {
	guard    packageGuard
	receipt  PackageReceipt
	manifest bundleManifest
	files    map[string]*os.File
	dir, exe string
}

func (p *heldPackage) close() error { return p.guard.close() }

// verifyPackage holds every selected file and directory until its caller has
// finished copying, activating or (in the later runner slice) joining a run.
// An inventory is a consistency statement, never publisher authentication.
func verifyPackage(ctx context.Context, root string, engine enginecatalog.EngineID, stored bool, expectedDigest string) (_ *heldPackage, err error) {
	descriptor, err := packageDescriptor(engine)
	if err != nil {
		return nil, err
	}
	path, err := packageAbsolute(root)
	if err != nil {
		return nil, err
	}
	p := &heldPackage{files: make(map[string]*os.File, 18), dir: path}
	defer func() {
		if err != nil {
			if closeErr := p.close(); closeErr != nil {
				err = closeErr
			}
		}
	}()
	if err = p.guard.lockDir(ctx, path, false); err != nil {
		return nil, err
	}
	inventoryFile, err := p.guard.open(ctx, filepath.Join(path, "release-manifest.json"))
	if err != nil {
		return nil, err
	}
	inventoryBytes, err := packageRead(ctx, inventoryFile, releaseInventoryLimit)
	if err != nil {
		return nil, err
	}
	inventory, err := parseReleaseInventory(inventoryBytes)
	if err != nil {
		return nil, err
	}
	paths := make(map[string]string, 17)
	paths["manifest.json"] = descriptor.RelativePath + "/manifest.json"
	paths[descriptor.Filename] = descriptor.RelativePath + "/" + descriptor.Filename
	for _, name := range packageNotices {
		paths[name] = name
	}
	var noticeSize int64
	for destination, source := range paths {
		file, ok := inventory.Files[source]
		if !ok {
			return nil, failure(FailureIntegrity, "发布清单缺少必需的引擎或署名文件", nil)
		}
		limit := int64(packageNoticeLimit)
		switch destination {
		case "manifest.json":
			limit = manifestLimit
		case descriptor.Filename:
			limit = executableLimit
		default:
			if file.Size > packageNoticeLimit-noticeSize {
				return nil, failure(FailureIntegrity, "引擎包署名文件总量超过2MiB", nil)
			}
			noticeSize += file.Size
		}
		if file.Size <= 0 || file.Size > limit {
			return nil, failure(FailureIntegrity, "引擎包声明的文件大小无效", nil)
		}
	}
	if stored {
		if !validPackageDigest(expectedDigest) {
			return nil, failure(FailureIntegrity, "引擎包摘要格式无效", nil)
		}
		if err := p.checkExactFiles(ctx, paths); err != nil {
			return nil, err
		}
	}
	actualInventoryHash := sha256.Sum256(inventoryBytes)
	p.receipt = PackageReceipt{Engine: engine, Files: []PackageFile{{Path: "release-manifest.json", Size: int64(len(inventoryBytes)), SHA256: hex.EncodeToString(actualInventoryHash[:])}}}
	p.files["release-manifest.json"] = inventoryFile
	names := make([]string, 0, len(paths))
	for name := range paths {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		entry := inventory.Files[paths[name]]
		location := paths[name]
		if stored {
			location = name
		}
		file, openErr := p.guard.open(ctx, filepath.Join(path, filepath.FromSlash(location)))
		if openErr != nil {
			return nil, openErr
		}
		digest, hashErr := packageHash(ctx, file, entry.Size, nil, nil)
		if hashErr != nil {
			return nil, hashErr
		}
		if digest != entry.SHA256 {
			return nil, failure(FailureIntegrity, "引擎包文件SHA256与发布清单不符", nil)
		}
		p.files[name] = file
		p.receipt.Files = append(p.receipt.Files, PackageFile{Path: name, Size: entry.Size, SHA256: digest})
	}
	manifestFile := p.files["manifest.json"]
	if _, err := manifestFile.Seek(0, 0); err != nil {
		return nil, packageIOError("读取引擎清单失败", err)
	}
	manifestBytes, err := packageRead(ctx, manifestFile, manifestLimit)
	if err != nil {
		return nil, err
	}
	p.manifest, err = parseManifest(manifestBytes, engine)
	if err != nil {
		return nil, err
	}
	exeEntry := inventory.Files[paths[descriptor.Filename]]
	if p.manifest.Size != exeEntry.Size || !strings.EqualFold(p.manifest.SHA256, exeEntry.SHA256) || p.manifest.EngineVersion != inventory.Version {
		return nil, failure(FailureIntegrity, "引擎清单与发布清单的程序身份不一致", nil)
	}
	p.receipt.Version = p.manifest.EngineVersion
	sort.Slice(p.receipt.Files, func(i, j int) bool { return p.receipt.Files[i].Path < p.receipt.Files[j].Path })
	p.receipt.Digest = packageDigest(p.receipt.Files)
	if stored && p.receipt.Digest != expectedDigest {
		return nil, failure(FailureIntegrity, "安装目录摘要与实际引擎包不一致", nil)
	}
	p.exe = p.files[descriptor.Filename].Name()
	return p, nil
}

func (p *heldPackage) checkExactFiles(ctx context.Context, files map[string]string) error {
	children := map[string]map[string]bool{"": {"release-manifest.json": false}}
	for name := range files {
		directory, base := filepath.ToSlash(filepath.Dir(name)), filepath.Base(name)
		if directory == "." {
			directory = ""
		}
		if children[directory] == nil {
			children[directory] = make(map[string]bool)
			children[""][directory] = true
		}
		children[directory][base] = false
	}
	for relative, expected := range children {
		directory := filepath.Join(p.dir, filepath.FromSlash(relative))
		if err := p.guard.lockDir(ctx, directory, false); err != nil {
			return err
		}
		entries, err := packageEntries(directory, len(expected))
		if err != nil {
			return err
		}
		if len(entries) != len(expected) {
			return failure(FailureIntegrity, "安装的引擎包文件集合不完整", nil)
		}
		for _, entry := range entries {
			dir, ok := expected[entry.Name()]
			if !ok || entry.IsDir() != dir || entry.Type()&os.ModeSymlink != 0 {
				return failure(FailureIntegrity, "安装的引擎包含额外或别名文件", nil)
			}
		}
	}
	return nil
}

func (m *PackageManager) installedPath(engine enginecatalog.EngineID, digest string) string {
	return filepath.Join(m.root, string(engine), digest)
}

func (m *PackageManager) verifiedInstalled(ctx context.Context, engine enginecatalog.EngineID, digest string) (*heldPackage, error) {
	if _, err := packageDescriptor(engine); err != nil {
		return nil, err
	}
	if !validPackageDigest(digest) {
		return nil, failure(FailureIntegrity, "引擎包摘要必须是64位小写十六进制", nil)
	}
	return verifyPackage(ctx, m.installedPath(engine, digest), engine, true, digest)
}

func packageIssue(engine enginecatalog.EngineID, digest string, err error) error {
	return fmt.Errorf("engine package %s/%s: %w", engine, digest, err)
}
