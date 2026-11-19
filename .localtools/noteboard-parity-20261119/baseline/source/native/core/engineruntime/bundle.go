package engineruntime

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"

	"levnote.local/native/core/enginecatalog"
	"levnote.local/native/core/engineipc"
)

const (
	manifestLimit   = 16 << 10
	executableLimit = 64 << 20
)

type bundleManifest struct {
	Schema        int                `json:"schema"`
	Engine        engineipc.EngineID `json:"engine"`
	EngineVersion string             `json:"engineVersion"`
	Protocol      int                `json:"protocol"`
	Format        string             `json:"format"`
	GOOS          string             `json:"goos"`
	GOARCH        string             `json:"goarch"`
	Filename      string             `json:"filename"`
	Size          int64              `json:"size"`
	SHA256        string             `json:"sha256"`
}

type verifiedBundle struct {
	info      EngineInfo
	exe, dir  string
	files     []*os.File
	installed *heldPackage
}

func (b *verifiedBundle) close() error {
	var errs []error
	for i := len(b.files) - 1; i >= 0; i-- {
		if err := b.files[i].Close(); err != nil {
			errs = append(errs, err)
		}
	}
	b.files = nil
	if b.installed != nil {
		errs = append(errs, b.installed.close())
		b.installed = nil
	}
	return errors.Join(errs...)
}

func admittedPlatform(descriptor enginecatalog.Descriptor) (enginecatalog.Descriptor, error) {
	if _, ok := enginecatalog.Match(descriptor.Engine, descriptor.Format, runtime.GOOS, runtime.GOARCH); !ok {
		return enginecatalog.Descriptor{}, failure(FailureUnavailable, "当前平台不支持此导出程序包", nil)
	}
	return descriptor, nil
}

func verifyBundle(ctx context.Context, hostDir string, id engineipc.EngineID) (_ *verifiedBundle, err error) {
	descriptor, admissionErr := admittedEngine(id)
	if admissionErr != nil {
		return nil, admissionErr
	}
	b := &verifiedBundle{dir: filepath.Join(hostDir, filepath.FromSlash(descriptor.RelativePath))}
	b.exe = filepath.Join(b.dir, descriptor.Filename)
	defer func() {
		if err != nil {
			if closeErr := b.close(); closeErr != nil {
				err = failure(FailureCleanup, "关闭导出验证句柄失败", closeErr)
			}
		}
	}()
	if err = lockBundleDirectories(ctx, b); err != nil {
		return nil, err
	}
	manifestFile, openErr := openLockedFile(filepath.Join(b.dir, "manifest.json"), false)
	if openErr != nil {
		return nil, bundleOpenFailure(openErr)
	}
	b.files = append(b.files, manifestFile)
	info, statErr := manifestFile.Stat()
	if statErr != nil || info.Size() <= 0 || info.Size() > manifestLimit {
		return nil, failure(FailureIntegrity, "导出清单大小无效", statErr)
	}
	metadata, readErr := io.ReadAll(io.LimitReader(manifestFile, manifestLimit+1))
	if readErr != nil || len(metadata) > manifestLimit {
		return nil, failure(FailureIntegrity, "读取导出清单失败", readErr)
	}
	m, parseErr := parseManifest(metadata, id)
	if parseErr != nil {
		return nil, parseErr
	}
	if err = contextFailure(ctx); err != nil {
		return nil, err
	}
	exe, openErr := openLockedFile(b.exe, false)
	if openErr != nil {
		return nil, bundleOpenFailure(openErr)
	}
	b.files = append(b.files, exe)
	info, statErr = exe.Stat()
	if statErr != nil || info.Size() != m.Size {
		return nil, failure(FailureIntegrity, "导出程序大小与清单不符", statErr)
	}
	hash := sha256.New()
	chunk := make([]byte, 32<<10)
	var count int64
	for {
		if err = contextFailure(ctx); err != nil {
			return nil, err
		}
		n, readErr := exe.Read(chunk)
		count += int64(n)
		if count > m.Size {
			return nil, failure(FailureIntegrity, "导出程序大小已变化", nil)
		}
		_, _ = hash.Write(chunk[:n])
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, failure(FailureIntegrity, "读取导出程序失败", readErr)
		}
	}
	digest := hex.EncodeToString(hash.Sum(nil))
	if count != m.Size || !strings.EqualFold(digest, m.SHA256) {
		return nil, failure(FailureIntegrity, "导出程序SHA256与清单不符", nil)
	}
	b.info = EngineInfo{ID: m.Engine, Version: m.EngineVersion, SHA256: digest, Source: PackageBundled}
	return b, nil
}

func bundleOpenFailure(err error) error {
	if errors.Is(err, os.ErrNotExist) {
		return failure(FailureUnavailable, "未找到随应用打包的导出程序或清单", err)
	}
	var known *Failure
	if errors.As(err, &known) {
		return known
	}
	return failure(FailureIntegrity, "无法锁定或验证随应用打包的导出文件", err)
}

func parseManifest(data []byte, id engineipc.EngineID) (bundleManifest, error) {
	var m bundleManifest
	bad := func() (bundleManifest, error) {
		return m, failure(FailureIntegrity, "导出清单字段无效或不兼容", nil)
	}
	descriptor, known := enginecatalog.Lookup(id)
	if !known {
		return bad()
	}
	if len(data) == 0 || len(data) > manifestLimit || !utf8.Valid(data) || !validJSONEscapes(data) {
		return bad()
	}
	allowed := map[string]bool{"schema": true, "engine": true, "engineVersion": true, "protocol": true, "format": true, "goos": true, "goarch": true, "filename": true, "size": true, "sha256": true}
	d := json.NewDecoder(bytes.NewReader(data))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return bad()
	}
	seen := make(map[string]bool, len(allowed))
	for d.More() {
		token, err = d.Token()
		key, ok := token.(string)
		if err != nil || !ok || !allowed[key] || seen[key] {
			return bad()
		}
		seen[key] = true
		var value json.RawMessage
		if d.Decode(&value) != nil || bytes.Equal(value, []byte("null")) {
			return bad()
		}
	}
	if _, err = d.Token(); err != nil || len(seen) != len(allowed) {
		return bad()
	}
	if _, err = d.Token(); err != io.EOF {
		return bad()
	}
	if json.Unmarshal(data, &m) != nil {
		return bad()
	}
	if m.Schema != 1 || m.Engine != descriptor.Engine || m.Protocol != descriptor.Protocol || m.Format != descriptor.Format ||
		m.GOOS != descriptor.GOOS || m.GOARCH != descriptor.GOARCH ||
		m.Filename != descriptor.Filename || m.Size <= 0 || m.Size > executableLimit || !validVersion(m.EngineVersion) || len(m.SHA256) != 64 {
		return bad()
	}
	if _, err := hex.DecodeString(m.SHA256); err != nil {
		return bad()
	}
	return m, nil
}

func validVersion(v string) bool {
	if len(v) == 0 || len(v) > engineipc.EngineVersionMaxBytes {
		return false
	}
	for i := range len(v) {
		if v[i] < 0x21 || v[i] > 0x7e {
			return false
		}
	}
	return true
}

// encoding/json accepts unmatched surrogate escapes; manifest validation must
// reject them, just as the wire metadata validator does.
func validJSONEscapes(data []byte) bool {
	for i := 0; i < len(data); i++ {
		if data[i] != '\\' {
			continue
		}
		i++
		if i >= len(data) {
			return false
		}
		if data[i] != 'u' {
			continue
		}
		if i+4 >= len(data) {
			return false
		}
		v, ok := hexQuad(data[i+1 : i+5])
		if !ok {
			return false
		}
		i += 4
		if v >= 0xdc00 && v <= 0xdfff {
			return false
		}
		if v < 0xd800 || v > 0xdbff {
			continue
		}
		if i+6 >= len(data) || data[i+1] != '\\' || data[i+2] != 'u' {
			return false
		}
		v, ok = hexQuad(data[i+3 : i+7])
		if !ok || v < 0xdc00 || v > 0xdfff {
			return false
		}
		i += 6
	}
	return true
}

func hexQuad(data []byte) (uint16, bool) {
	var v uint16
	for _, c := range data {
		v <<= 4
		switch {
		case c >= '0' && c <= '9':
			v |= uint16(c - '0')
		case c >= 'a' && c <= 'f':
			v |= uint16(c - 'a' + 10)
		case c >= 'A' && c <= 'F':
			v |= uint16(c - 'A' + 10)
		default:
			return 0, false
		}
	}
	return v, true
}
