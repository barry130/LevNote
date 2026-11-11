//go:build windows

package document

import (
	"syscall"
	"unsafe"
)

var kernel32 = syscall.NewLazyDLL("kernel32.dll")
var replaceFileW = kernel32.NewProc("ReplaceFileW")
var moveFileExW = kernel32.NewProc("MoveFileExW")

func replaceExisting(from, to string) error {
	src, err := syscall.UTF16PtrFromString(from)
	if err != nil {
		return err
	}
	dst, err := syscall.UTF16PtrFromString(to)
	if err != nil {
		return err
	}
	// ReplaceFile preserves the destination ACL and attributes. No DELETE+MOVE
	// fallback: a failed replacement must never destroy the original document.
	ok, _, callErr := replaceFileW.Call(uintptr(unsafe.Pointer(dst)), uintptr(unsafe.Pointer(src)), 0, 0, 0, 0)
	if ok == 0 {
		return callErr
	}
	return nil
}
func publishNew(from, to string) error {
	src, err := syscall.UTF16PtrFromString(from)
	if err != nil {
		return err
	}
	dst, err := syscall.UTF16PtrFromString(to)
	if err != nil {
		return err
	}
	// Omit MOVEFILE_REPLACE_EXISTING. A target created after the file dialog
	// still wins; this call can never overwrite it. WRITE_THROUGH=0x8.
	ok, _, callErr := moveFileExW.Call(uintptr(unsafe.Pointer(src)), uintptr(unsafe.Pointer(dst)), 0x8)
	if ok == 0 {
		return callErr
	}
	return nil
}
func syncDirectory(string) error { return nil }
