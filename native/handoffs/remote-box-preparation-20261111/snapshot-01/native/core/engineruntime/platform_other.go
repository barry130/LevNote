//go:build !windows

package engineruntime

import (
	"context"
	"os"
	"os/exec"
)

type processJob struct{}

func newProcessJob() (*processJob, error) {
	return nil, failure(FailureUnavailable, "当前平台不支持已验证的导出进程运行器", nil)
}
func (*processJob) attach(*os.Process) error {
	return failure(FailureUnavailable, "不支持的平台", nil)
}
func (*processJob) terminate() error    { return nil }
func (*processJob) closeAndWait() error { return nil }
func configureCommand(*exec.Cmd)        {}
func lockBundleDirectories(context.Context, *verifiedBundle) error {
	return failure(FailureUnavailable, "当前平台不支持已验证的导出进程运行器", nil)
}
func openLockedFile(string, bool) (*os.File, error) {
	return nil, failure(FailureUnavailable, "不支持的平台", nil)
}
