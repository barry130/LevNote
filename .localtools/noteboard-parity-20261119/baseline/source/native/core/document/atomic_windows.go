//go:build windows

package document

import (
	"syscall"
	"time"
	"unsafe"
)

var kernel32 = syscall.NewLazyDLL("kernel32.dll")
var replaceFileW = kernel32.NewProc("ReplaceFileW")
var moveFileExW = kernel32.NewProc("MoveFileExW")

// Windows reports a destination that another process holds open with these
// codes. Antivirus scanners, indexers and backup agents hold files for tens of
// milliseconds at a time, so an immediate failure would turn an ordinary
// background scan into a save the user cannot complete.
const (
	errorSharingViolation       = syscall.Errno(32)
	errorLockViolation          = syscall.Errno(33)
	errorUnableToRemoveReplaced = syscall.Errno(1175)
)

// replaceRetryBudget bounds the total wait. It is deliberately short: a file
// held for longer than this is a real conflict that must be reported, not
// hidden behind an unbounded retry loop.
const (
	replaceRetryAttempts = 6
	replaceRetryDelay    = 25 * time.Millisecond
)

// The wait boundary permits deterministic Windows filesystem fault scheduling
// in tests; production retains the bounded delay before each delayed attempt.
var replaceRetryWait = time.Sleep

func isTransientReplaceError(err error) bool {
	return err == errorSharingViolation || err == errorLockViolation || err == errorUnableToRemoveReplaced
}

func replaceExisting(from, to string, check func() error) error {
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
	var callErr error
	for attempt := 0; attempt < replaceRetryAttempts; attempt++ {
		if attempt > 0 {
			replaceRetryWait(replaceRetryDelay)
			// Reuse the caller's original expectations after every wait. Never
			// adopt a changed destination as a new baseline. The immediate
			// check-to-ReplaceFileW interval still is not an OS-level CAS.
			if check != nil {
				if err := check(); err != nil {
					return err
				}
			}
		}
		var ok uintptr
		ok, _, callErr = replaceFileW.Call(uintptr(unsafe.Pointer(dst)), uintptr(unsafe.Pointer(src)), 0, 0, 0, 0)
		if ok != 0 {
			return nil
		}
		if !isTransientReplaceError(callErr) {
			return callErr
		}
	}
	return callErr
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
