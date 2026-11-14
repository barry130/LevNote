//go:build !windows

package document

import "os"

func replaceExisting(from, to string, _ func() error) error { return os.Rename(from, to) }
func publishNew(from, to string) error {
	// Linking is an atomic create-if-absent within the same filesystem. Rename
	// alone would silently replace a target created between the check and save.
	return os.Link(from, to)
}
func syncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
