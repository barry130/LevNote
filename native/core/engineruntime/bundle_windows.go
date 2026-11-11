//go:build windows

package engineruntime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var kernel32 = syscall.NewLazyDLL("kernel32.dll")
var getFinalPathName = kernel32.NewProc("GetFinalPathNameByHandleW")

func lockBundleDirectories(ctx context.Context, b *verifiedBundle) error {
	absolute, err := filepath.Abs(b.dir)
	if err != nil {
		return bundleOpenFailure(err)
	}
	volume := filepath.VolumeName(absolute)
	// The v1 package target is a normal local Windows installation directory.
	// UNC/device namespace launch would require a separate reviewed policy.
	if len(volume) != 2 || volume[1] != ':' || strings.HasPrefix(absolute, `\\`) {
		return failure(FailureIntegrity, "导出程序必须位于本地应用目录", nil)
	}
	current := volume + `\`
	parts := strings.Split(strings.TrimPrefix(absolute, current), `\`)
	for index := -1; index < len(parts); index++ {
		if err := contextFailure(ctx); err != nil {
			return err
		}
		if index >= 0 {
			current = filepath.Join(current, parts[index])
		}
		file, openErr := openLockedFile(current, true)
		if openErr != nil {
			return bundleOpenFailure(openErr)
		}
		b.files = append(b.files, file)
	}
	return nil
}

func openLockedFile(path string, directory bool) (*os.File, error) {
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	access := uint32(syscall.GENERIC_READ)
	flags := uint32(syscall.FILE_FLAG_OPEN_REPARSE_POINT)
	if directory {
		access = 0x80 // FILE_READ_ATTRIBUTES, sufficient for identity checks.
		flags |= syscall.FILE_FLAG_BACKUP_SEMANTICS
	}
	h, err := syscall.CreateFile(name, access, syscall.FILE_SHARE_READ, nil, syscall.OPEN_EXISTING, flags, 0)
	if err != nil {
		return nil, err
	}
	valid := false
	defer func() {
		if !valid {
			_ = syscall.CloseHandle(h)
		}
	}()
	var info syscall.ByHandleFileInformation
	if err = syscall.GetFileInformationByHandle(h, &info); err != nil {
		return nil, err
	}
	if info.FileAttributes&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0 || (info.FileAttributes&syscall.FILE_ATTRIBUTE_DIRECTORY != 0) != directory {
		return nil, failure(FailureIntegrity, "导出路径含重解析点或文件类型不符", nil)
	}
	kind, err := syscall.GetFileType(h)
	if err != nil || kind != syscall.FILE_TYPE_DISK {
		return nil, failure(FailureIntegrity, "导出文件不是普通磁盘文件", err)
	}
	resolved, err := finalPath(h)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(filepath.Clean(resolved), filepath.Clean(path)) {
		return nil, failure(FailureIntegrity, "导出文件句柄路径与固定目录不符", nil)
	}
	file := os.NewFile(uintptr(h), path)
	if file == nil {
		return nil, failure(FailureIntegrity, "无法持有导出文件句柄", nil)
	}
	valid = true
	return file, nil
}

func finalPath(h syscall.Handle) (string, error) {
	// A bounded Windows path buffer; no path-provided allocation size is trusted.
	buf := make([]uint16, 32768)
	n, _, err := getFinalPathName.Call(uintptr(h), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), 0)
	if n == 0 {
		return "", err
	}
	if n >= uintptr(len(buf)) {
		return "", failure(FailureIntegrity, "导出路径超过Windows长度限制", nil)
	}
	return strings.TrimPrefix(syscall.UTF16ToString(buf[:n]), `\\?\`), nil
}
