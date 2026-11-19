package engineruntime

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"levnote.local/native/core/enginecatalog"
)

type packageGuard struct {
	files []*os.File
	dirs  map[string]bool
}

func (g *packageGuard) close() error {
	var errs []error
	for i := len(g.files) - 1; i >= 0; i-- {
		if err := g.files[i].Close(); err != nil {
			errs = append(errs, err)
		}
	}
	g.files, g.dirs = nil, nil
	if err := errors.Join(errs...); err != nil {
		return failure(FailureCleanup, "关闭引擎包验证句柄失败", err)
	}
	return nil
}

// Every ancestor is held without delete/write sharing, from volume root down.
// Creation occurs only below an already held parent. This prevents ordinary
// directory replacement while checking/copying; it is not an OS sandbox.
func (g *packageGuard) lockDir(ctx context.Context, path string, create bool) error {
	if g.dirs == nil {
		g.dirs = make(map[string]bool)
	}
	for index, component := range packagePathParts(path) {
		if err := contextFailure(ctx); err != nil {
			return err
		}
		key := strings.ToLower(component)
		if g.dirs[key] {
			continue
		}
		f, err := openLockedFile(component, true)
		if err != nil {
			// Validate and hold every existing ancestor first. Only a genuine
			// absence below the held volume root authorizes creation.
			if !create || index == 0 || !errors.Is(err, os.ErrNotExist) {
				return packageIOError("无法持有引擎包目录", err)
			}
			if err := os.Mkdir(component, 0700); err != nil && !errors.Is(err, os.ErrExist) {
				return packageIOError("创建引擎包目录失败", err)
			}
			f, err = openLockedFile(component, true)
			if err != nil {
				return packageIOError("无法持有引擎包目录", err)
			}
		}
		g.files = append(g.files, f)
		g.dirs[key] = true
	}
	return nil
}

func (g *packageGuard) open(ctx context.Context, path string) (*os.File, error) {
	if err := g.lockDir(ctx, filepath.Dir(path), false); err != nil {
		return nil, err
	}
	if err := contextFailure(ctx); err != nil {
		return nil, err
	}
	f, err := packageOpenFile(path)
	if err != nil {
		return nil, packageIOError("无法持有引擎包文件", err)
	}
	g.files = append(g.files, f)
	return f, nil
}

func packageIOError(message string, err error) error {
	var typed *Failure
	if errors.As(err, &typed) {
		return err
	}
	return failure(FailureIntegrity, message, err)
}

func packageRead(ctx context.Context, f *os.File, limit int64) ([]byte, error) {
	info, err := f.Stat()
	if err != nil || info.Size() <= 0 || info.Size() > limit {
		return nil, packageIOError("引擎包元数据超过限制或为空", err)
	}
	if err := contextFailure(ctx); err != nil {
		return nil, err
	}
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(data)) > limit || int64(len(data)) != info.Size() {
		return nil, packageIOError("读取引擎包元数据失败或大小已变化", err)
	}
	if err := contextFailure(ctx); err != nil {
		return nil, err
	}
	return data, nil
}

func packageHash(ctx context.Context, f *os.File, expectedSize int64, output io.Writer, progress func(int64)) (string, error) {
	info, err := f.Stat()
	if err != nil || info.Size() != expectedSize {
		return "", packageIOError("引擎包文件大小与清单不符", err)
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", packageIOError("无法读取引擎包文件起点", err)
	}
	hash := sha256.New()
	chunk := make([]byte, 32<<10)
	var count int64
	for {
		if err := contextFailure(ctx); err != nil {
			return "", err
		}
		n, readErr := f.Read(chunk)
		if int64(n) > expectedSize-count {
			return "", failure(FailureIntegrity, "引擎包文件增长超过已验证大小", nil)
		}
		count += int64(n)
		_, _ = hash.Write(chunk[:n])
		if output != nil && n > 0 {
			written, writeErr := output.Write(chunk[:n])
			if writeErr != nil || written != n {
				return "", packageIOError("写入引擎包文件失败", errors.Join(writeErr, io.ErrShortWrite))
			}
		}
		if progress != nil && n > 0 {
			progress(count)
		}
		if err := contextFailure(ctx); err != nil {
			return "", err
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", packageIOError("读取引擎包文件失败", readErr)
		}
	}
	if count != expectedSize {
		return "", failure(FailureIntegrity, "引擎包文件长度与清单不符", nil)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func packageDigest(files []PackageFile) string {
	// Exact v1 identity: ASCII domain including NUL, u32 BE entry count, then
	// entries sorted by stored UTF-8 path bytes: u16 BE path byte length, path,
	// u64 BE byte size, raw 32-byte SHA256. No JSON or version sorting involved.
	ordered := append([]PackageFile(nil), files...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	h := sha256.New()
	_, _ = h.Write([]byte("LevNoteEnginePackageV1\x00"))
	var number [8]byte
	binary.BigEndian.PutUint32(number[:4], uint32(len(ordered)))
	_, _ = h.Write(number[:4])
	for _, file := range ordered {
		binary.BigEndian.PutUint16(number[:2], uint16(len(file.Path)))
		_, _ = h.Write(number[:2])
		_, _ = h.Write([]byte(file.Path))
		binary.BigEndian.PutUint64(number[:], uint64(file.Size))
		_, _ = h.Write(number[:])
		digest, _ := hex.DecodeString(file.SHA256) // callers supply validated hashes
		_, _ = h.Write(digest)
	}
	return hex.EncodeToString(h.Sum(nil))
}

func validPackageDigest(digest string) bool {
	if len(digest) != 64 {
		return false
	}
	for _, c := range digest {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func packageDescriptor(id enginecatalog.EngineID) (enginecatalog.Descriptor, error) {
	descriptor, ok := enginecatalog.Lookup(id)
	if !ok {
		return enginecatalog.Descriptor{}, failure(FailureUnavailable, "未知的固定引擎", nil)
	}
	return descriptor, nil
}

func packageEntries(path string, limit int) ([]os.DirEntry, error) {
	// Called only while the directory and its ancestors are held against rename.
	f, err := os.Open(path)
	if err != nil {
		return nil, packageIOError("读取引擎包目录失败", err)
	}
	entries, readErr := f.ReadDir(limit + 1)
	closeErr := f.Close()
	if closeErr != nil {
		return nil, failure(FailureCleanup, "关闭引擎包目录枚举失败", closeErr)
	}
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return nil, packageIOError("枚举引擎包目录失败", readErr)
	}
	if len(entries) > limit {
		return nil, failure(FailureIntegrity, "引擎包目录数量超过限制", nil)
	}
	return entries, nil
}
