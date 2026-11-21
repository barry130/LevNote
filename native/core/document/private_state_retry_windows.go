//go:build windows

package document

import (
	"errors"
	"syscall"
)

func privateStateSharingViolation(err error) bool {
	return errors.Is(err, syscall.Errno(32)) || errors.Is(err, syscall.Errno(33))
}

func privateStateTargetReadRetryable(err error) bool {
	targetErr, ok := err.(*privateStateTargetReadRetryError)
	if !ok {
		return false
	}
	openErr, ok := targetErr.err.(*privateStateFileOpenError)
	if !ok {
		return false
	}
	return privateStateSharingViolation(openErr.err)
}

func privateStateTargetReadWait()           { replaceRetryWait(replaceRetryDelay) }
func privateStateTargetReadWaitBudget() int { return replaceRetryAttempts - 1 }
