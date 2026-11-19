//go:build !windows

package engineruntime

import "os"

func packagePlatform() error {
	return failure(FailureUnavailable, "当前平台不支持本地引擎包仓库", nil)
}
func packageAbsolute(string) (string, error)     { return "", packagePlatform() }
func packagePathParts(string) []string           { return nil }
func packageOpenFile(string) (*os.File, error)   { return nil, packagePlatform() }
func packageCreateFile(string) (*os.File, error) { return nil, packagePlatform() }
func packageRename(string, string, bool) error   { return packagePlatform() }
