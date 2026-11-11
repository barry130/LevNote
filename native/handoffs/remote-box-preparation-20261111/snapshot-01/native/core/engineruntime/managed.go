package engineruntime

import (
	"context"
	"path/filepath"
	"strings"
)

func verifySelectedBundle(ctx context.Context, selected selectedPackage) (*verifiedBundle, error) {
	switch selected.Source {
	case PackageBundled:
		return verifyBundle(ctx, selected.hostDir, selected.Engine)
	case PackageInstalled:
		if !validPackageDigest(selected.Digest) || selected.storeRoot == "" {
			return nil, failure(FailureIntegrity, "选中的引擎包摘要或仓库无效", nil)
		}
		p, err := verifyPackage(ctx, filepath.Join(selected.storeRoot, string(selected.Engine), selected.Digest), selected.Engine, true, selected.Digest)
		if err != nil {
			return nil, err
		}
		if p.receipt.Version != selected.Version {
			if closeErr := p.close(); closeErr != nil {
				return nil, closeErr
			}
			return nil, failure(FailureIntegrity, "选中引擎包版本与已验证文件不一致", nil)
		}
		// Retain the entire verifier owner, including every notice/inventory and
		// ancestor handle. run closes it only after process/pipe/job cleanup.
		return &verifiedBundle{
			info: EngineInfo{ID: p.manifest.Engine, Version: p.manifest.EngineVersion,
				SHA256: strings.ToLower(p.manifest.SHA256), Source: PackageInstalled, PackageDigest: selected.Digest},
			exe: p.exe, dir: p.dir, installed: p,
		}, nil
	default:
		return nil, failure(FailureIntegrity, "引擎包选择来源无效", nil)
	}
}
