package engineruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"unicode/utf8"

	"levnote.local/native/core/enginecatalog"
)

const packageStateName = "selections.json"

type packageStateEntry struct {
	Engine  enginecatalog.EngineID `json:"engine"`
	Digest  string                 `json:"digest"`
	Version string                 `json:"version"`
}

type packageState struct {
	Schema     int                  `json:"schema"`
	Selections [4]packageStateEntry `json:"selections"`
}

func bundledPackageStatuses() [4]PackageStatus {
	var statuses [4]PackageStatus
	for i, engine := range packageEngines {
		statuses[i] = PackageStatus{Engine: engine, Source: PackageBundled}
	}
	return statuses
}

func NewPackageManager(hostDir, stateDir string) (_ *PackageManager, err error) {
	if err := packagePlatform(); err != nil {
		return nil, err
	}
	hostDir, err = packageAbsolute(hostDir)
	if err != nil {
		return nil, err
	}
	stateDir, err = packageAbsolute(stateDir)
	if err != nil {
		return nil, err
	}
	m := &PackageManager{hostDir: hostDir, root: filepath.Join(stateDir, "engine-packages"), gate: make(chan struct{}, 1), statuses: bundledPackageStatuses()}
	var guard packageGuard
	defer func() {
		if closeErr := guard.close(); closeErr != nil {
			err = closeErr
		}
	}()
	ctx := context.Background()
	if err = guard.lockDir(ctx, hostDir, false); err != nil {
		return nil, err
	}
	if err = guard.lockDir(ctx, m.root, true); err != nil {
		return nil, err
	}
	for _, engine := range packageEngines {
		if err = guard.lockDir(ctx, filepath.Join(m.root, string(engine)), true); err != nil {
			return nil, err
		}
	}
	f, readErr := guard.open(ctx, filepath.Join(m.root, packageStateName))
	if errors.Is(readErr, os.ErrNotExist) {
		return m, nil
	}
	if readErr != nil {
		m.loadErr = readErr
		return m, nil
	}
	data, readErr := packageRead(ctx, f, packageStateLimit)
	if readErr == nil {
		m.statuses, readErr = parsePackageState(data)
	}
	if readErr != nil {
		// Keep the unusable file verbatim; a later explicit all-selection repair
		// is the only operation permitted to replace corrupt preferences.
		m.statuses, m.loadErr = bundledPackageStatuses(), readErr
	}
	return m, nil
}

func parsePackageState(data []byte) ([4]PackageStatus, error) {
	bad := func() ([4]PackageStatus, error) {
		return [4]PackageStatus{}, failure(FailureIntegrity, "引擎包选择状态格式无效", nil)
	}
	if len(data) == 0 || len(data) > packageStateLimit || !utf8.Valid(data) || !validJSONEscapes(data) {
		return bad()
	}
	root, err := packageObject(data, []string{"schema", "selections"})
	if err != nil {
		return bad()
	}
	schema := string(root["schema"])
	var allowed []enginecatalog.EngineID
	switch schema {
	case "1":
		allowed = legacyPackageEngines[:]
	case "2":
		allowed = packageEngines[:]
	default:
		return bad()
	}
	d := json.NewDecoder(bytes.NewReader(root["selections"]))
	token, err := d.Token()
	if err != nil || token != json.Delim('[') {
		return bad()
	}
	statuses := bundledPackageStatuses()
	seen := make(map[enginecatalog.EngineID]bool, len(allowed))
	known := make(map[enginecatalog.EngineID]bool, len(allowed))
	for _, engine := range allowed {
		known[engine] = true
	}
	for d.More() {
		if len(seen) >= len(allowed) {
			return bad()
		}
		var raw json.RawMessage
		if d.Decode(&raw) != nil {
			return bad()
		}
		fields, err := packageObject(raw, []string{"engine", "digest", "version"})
		var entry packageStateEntry
		if err != nil || json.Unmarshal(fields["engine"], &entry.Engine) != nil || json.Unmarshal(fields["digest"], &entry.Digest) != nil || json.Unmarshal(fields["version"], &entry.Version) != nil {
			return bad()
		}
		if !known[entry.Engine] || seen[entry.Engine] {
			return bad()
		}
		seen[entry.Engine] = true
		status := PackageStatus{Engine: entry.Engine, Source: PackageBundled}
		if entry.Digest == "" {
			if entry.Version != "" {
				return bad()
			}
		} else {
			if !validPackageDigest(entry.Digest) || !validVersion(entry.Version) {
				return bad()
			}
			status.Source, status.Digest, status.Version = PackageInstalled, entry.Digest, entry.Version
		}
		for i := range statuses {
			if statuses[i].Engine == entry.Engine {
				statuses[i] = status
			}
		}
	}
	if token, err = d.Token(); err != nil || token != json.Delim(']') || len(seen) != len(allowed) {
		return bad()
	}
	if _, err := d.Token(); err != io.EOF {
		return bad()
	}
	return statuses, nil
}

func packageObject(data []byte, keys []string) (map[string]json.RawMessage, error) {
	bad := failure(FailureIntegrity, "引擎包状态包含无效字段", nil)
	d := json.NewDecoder(bytes.NewReader(data))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return nil, bad
	}
	allowed := make(map[string]bool, len(keys))
	for _, key := range keys {
		allowed[key] = true
	}
	fields := make(map[string]json.RawMessage, len(keys))
	for d.More() {
		token, err = d.Token()
		key, ok := token.(string)
		if err != nil || !ok || !allowed[key] || fields[key] != nil {
			return nil, bad
		}
		var value json.RawMessage
		if d.Decode(&value) != nil || bytes.Equal(value, []byte("null")) {
			return nil, bad
		}
		fields[key] = value
	}
	if token, err = d.Token(); err != nil || token != json.Delim('}') || len(fields) != len(keys) {
		return nil, bad
	}
	if _, err := d.Token(); err != io.EOF {
		return nil, bad
	}
	return fields, nil
}

func (m *PackageManager) stateSnapshot() ([4]PackageStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.statuses, m.loadErr
}

func (m *PackageManager) persistState(ctx context.Context, next [4]PackageStatus) (err error) {
	var state packageState
	state.Schema = 2
	for i, status := range next {
		state.Selections[i] = packageStateEntry{Engine: status.Engine, Digest: status.Digest, Version: status.Version}
	}
	data, err := json.Marshal(state)
	if err != nil || len(data) > packageStateLimit {
		return packageIOError("无法编码引擎包选择状态", err)
	}
	var guard packageGuard
	defer func() {
		if closeErr := guard.close(); closeErr != nil {
			err = closeErr
		}
	}()
	if err := guard.lockDir(ctx, m.root, false); err != nil {
		return err
	}
	path := filepath.Join(m.root, packageStateName)
	old, openErr := packageOpenFile(path)
	if openErr == nil {
		if closeErr := old.Close(); closeErr != nil {
			return failure(FailureCleanup, "关闭旧引擎选择文件失败", closeErr)
		}
	} else if !errors.Is(openErr, os.ErrNotExist) {
		return packageIOError("旧引擎选择路径不是普通文件", openErr)
	}
	name, err := packageRandomName(".selection-")
	if err != nil {
		return err
	}
	temp := filepath.Join(m.root, name)
	f, err := packageCreateFile(temp)
	if err != nil {
		return packageIOError("创建引擎选择临时文件失败", err)
	}
	published := false
	defer func() {
		if !published {
			if removeErr := os.Remove(temp); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				err = failure(FailureCleanup, "清理引擎选择临时文件失败", errors.Join(err, removeErr))
			}
		}
	}()
	n, writeErr := f.Write(data)
	if writeErr == nil && n != len(data) {
		writeErr = io.ErrShortWrite
	}
	if writeErr == nil {
		writeErr = f.Sync()
	}
	closeErr := f.Close()
	if writeErr != nil || closeErr != nil {
		return packageIOError("持久化引擎选择失败", errors.Join(writeErr, closeErr))
	}
	if m.hooks != nil && m.hooks.beforeStateCommit != nil {
		if err := m.hooks.beforeStateCommit(); err != nil {
			return packageIOError("发布引擎选择失败", err)
		}
	}
	if err := contextFailure(ctx); err != nil {
		return err
	}
	if err := packageRename(temp, path, true); err != nil {
		return packageIOError("原子更新引擎选择失败", err)
	}
	published = true
	// Rename is the commit point. Cancellation arriving afterwards cannot undo
	// an already published preference; the small live snapshot must match it.
	m.mu.Lock()
	m.statuses, m.loadErr = next, nil
	m.mu.Unlock()
	return nil
}

func (m *PackageManager) Activate(ctx context.Context, engine enginecatalog.EngineID, digest string) (err error) {
	if err := m.acquire(ctx); err != nil {
		return err
	}
	defer m.release()
	next, loadErr := m.stateSnapshot()
	if loadErr != nil {
		return failure(FailureIntegrity, "选择状态损坏；必须明确重置全部选择", loadErr)
	}
	p, err := m.verifiedInstalled(ctx, engine, digest)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := p.close(); closeErr != nil {
			err = closeErr
		}
	}()
	for i := range next {
		if next[i].Engine == engine {
			next[i] = PackageStatus{Engine: engine, Source: PackageInstalled, Digest: digest, Version: p.receipt.Version}
		}
	}
	return m.persistState(ctx, next)
}

func (m *PackageManager) Reset(ctx context.Context, engine enginecatalog.EngineID) error {
	if _, err := packageDescriptor(engine); err != nil {
		return err
	}
	if err := m.acquire(ctx); err != nil {
		return err
	}
	defer m.release()
	next, loadErr := m.stateSnapshot()
	if loadErr != nil {
		return failure(FailureIntegrity, "选择状态损坏；必须明确重置全部选择", loadErr)
	}
	for i := range next {
		if next[i].Engine == engine {
			next[i] = PackageStatus{Engine: engine, Source: PackageBundled}
		}
	}
	return m.persistState(ctx, next)
}

func (m *PackageManager) RepairResetAll(ctx context.Context) error {
	if err := m.acquire(ctx); err != nil {
		return err
	}
	defer m.release()
	return m.persistState(ctx, bundledPackageStatuses())
}
