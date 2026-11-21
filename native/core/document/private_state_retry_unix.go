//go:build !windows

package document

func privateStateSharingViolation(error) bool    { return false }
func privateStateTargetReadRetryable(error) bool { return false }
func privateStateTargetReadWait()                {}
func privateStateTargetReadWaitBudget() int      { return 0 }
