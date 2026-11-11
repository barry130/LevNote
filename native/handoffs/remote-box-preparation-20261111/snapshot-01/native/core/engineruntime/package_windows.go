//go:build windows

package engineruntime

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var packageMoveFileEx = kernel32.NewProc("MoveFileExW")

func packagePlatform() error { return nil }

func packageAbsolute(path string) (string, error) {
	if path == "" || len(path) > 30000 || !filepath.IsAbs(path) || strings.HasPrefix(path, `\\`) {
		return "", failure(FailureIntegrity, "引擎包路径必须为本地绝对磁盘路径", nil)
	}
	path = strings.ReplaceAll(path, "/", `\`)
	volume := filepath.VolumeName(path)
	if len(volume) != 2 || volume[1] != ':' || !((volume[0] >= 'A' && volume[0] <= 'Z') || (volume[0] >= 'a' && volume[0] <= 'z')) {
		return "", failure(FailureIntegrity, "引擎包路径不支持网络或设备命名空间", nil)
	}
	// Reject ambiguous components instead of cleaning aliases into acceptance.
	relative := strings.TrimSuffix(strings.TrimPrefix(path, volume+`\`), `\`)
	if relative != "" && !inventoryValidPath(strings.ReplaceAll(relative, `\`, "/")) {
		return "", failure(FailureIntegrity, "引擎包路径含无效或歧义目录名", nil)
	}
	return filepath.Clean(path), nil
}

func packagePathParts(path string) []string {
	volume := filepath.VolumeName(path) + `\`
	parts := []string{volume}
	current := volume
	for _, part := range strings.Split(strings.TrimPrefix(path, volume), `\`) {
		if part != "" {
			current = filepath.Join(current, part)
			parts = append(parts, current)
		}
	}
	return parts
}

func packageOpenFile(path string) (*os.File, error) {
	f, err := openLockedFile(path, false)
	if err != nil {
		return nil, err
	}
	var info syscall.ByHandleFileInformation
	err = syscall.GetFileInformationByHandle(syscall.Handle(f.Fd()), &info)
	if err != nil || info.NumberOfLinks != 1 {
		closeErr := f.Close()
		if closeErr != nil {
			return nil, failure(FailureCleanup, "关闭引擎包文件句柄失败", closeErr)
		}
		return nil, failure(FailureIntegrity, "引擎包文件不允许硬链接", err)
	}
	return f, nil
}

func packageCreateFile(path string) (*os.File, error) {
	name, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	h, err := syscall.CreateFile(name, syscall.GENERIC_READ|syscall.GENERIC_WRITE, syscall.FILE_SHARE_READ, nil, syscall.CREATE_NEW, syscall.FILE_ATTRIBUTE_NORMAL|syscall.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(h), path)
	resolved, err := finalPath(h)
	if err != nil || !strings.EqualFold(filepath.Clean(resolved), filepath.Clean(path)) {
		_ = f.Close()
		return nil, failure(FailureIntegrity, "新引擎包文件句柄位置不符", err)
	}
	return f, nil
}

func packageRename(from, to string, replace bool) error {
	oldName, err := syscall.UTF16PtrFromString(from)
	if err != nil {
		return err
	}
	newName, err := syscall.UTF16PtrFromString(to)
	if err != nil {
		return err
	}
	flags := uintptr(0x8) // MOVEFILE_WRITE_THROUGH; no copy across volumes.
	if replace {
		flags |= 0x1 // Atomic replacement of the selection file only.
	}
	result, _, callErr := packageMoveFileEx.Call(uintptr(unsafe.Pointer(oldName)), uintptr(unsafe.Pointer(newName)), flags)
	if result == 0 {
		return callErr
	}
	return nil
}
