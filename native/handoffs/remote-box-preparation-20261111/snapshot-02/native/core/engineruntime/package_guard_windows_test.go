//go:build windows

package engineruntime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func closePackageGuard(t *testing.T, guard *packageGuard) {
	t.Helper()
	if err := guard.close(); err != nil {
		t.Fatal(err)
	}
}

func TestPackageGuardCreatesBelowExistingVolumeRoot(t *testing.T) {
	base := t.TempDir()
	if !strings.EqualFold(filepath.VolumeName(base), "C:") {
		t.Skip("system TEMP is not on C:")
	}
	target := filepath.Join(base, "missing", "engine-packages", "markdown-html")
	var guard packageGuard
	defer func() {
		if err := guard.close(); err != nil {
			t.Error(err)
		}
	}()
	if err := guard.lockDir(context.Background(), target, true); err != nil {
		t.Fatalf("existing volume root was not held before creating descendants: %v", err)
	}
	for _, path := range []string{
		filepath.Join(base, "missing"),
		filepath.Join(base, "missing", "engine-packages"),
		target,
	} {
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			t.Fatalf("missing descendant was not created as a directory: %s: %v", path, err)
		}
	}
	closePackageGuard(t, &guard)
}

func TestPackageGuardRefusesMissingWithoutCreateAndFileAncestor(t *testing.T) {
	t.Run("create-false", func(t *testing.T) {
		base := t.TempDir()
		missing := filepath.Join(base, "missing", "engine-packages")
		var guard packageGuard
		defer func() {
			if err := guard.close(); err != nil {
				t.Error(err)
			}
		}()
		err := guard.lockDir(context.Background(), missing, false)
		var typed *Failure
		if !errors.As(err, &typed) || typed.Kind != FailureIntegrity || !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("missing path was not refused as a filesystem absence: %v", err)
		}
		if _, statErr := os.Stat(filepath.Join(base, "missing")); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("create=false created a path: %v", statErr)
		}
	})

	t.Run("file-ancestor", func(t *testing.T) {
		base := t.TempDir()
		blocked := filepath.Join(base, "blocked")
		if err := os.WriteFile(blocked, []byte("existing file"), 0600); err != nil {
			t.Fatal(err)
		}
		target := filepath.Join(blocked, "child", "engine-packages")
		var guard packageGuard
		defer func() {
			if err := guard.close(); err != nil {
				t.Error(err)
			}
		}()
		err := guard.lockDir(context.Background(), target, true)
		var typed *Failure
		if !errors.As(err, &typed) || typed.Kind != FailureIntegrity {
			t.Fatalf("file ancestor was not refused as an invalid directory: %v", err)
		}
		data, readErr := os.ReadFile(blocked)
		if readErr != nil || string(data) != "existing file" {
			t.Fatalf("file ancestor changed: %q: %v", data, readErr)
		}
		if _, statErr := os.Stat(filepath.Join(blocked, "child")); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("descendant below a file ancestor was created: %v", statErr)
		}
	})
}

func TestNewPackageManagerCrossVolumeHostFStateDefaultC(t *testing.T) {
	state := t.TempDir()
	if !strings.EqualFold(filepath.VolumeName(state), "C:") {
		t.Skip("system TEMP is not on C:")
	}
	nativeRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, statErr := os.Stat(nativeRoot); statErr != nil {
		t.Fatal(statErr)
	}
	if !strings.EqualFold(filepath.VolumeName(nativeRoot), "F:") {
		t.Skip("repository is not on F:")
	}
	host, err := os.MkdirTemp(nativeRoot, ".engine-package-host-")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Remove(host); err != nil {
			t.Error(err)
		}
	}()
	manager, err := NewPackageManager(host, state)
	if err != nil {
		t.Fatalf("cross-volume manager creation failed: %v", err)
	}
	if manager.root != filepath.Join(state, "engine-packages") {
		t.Fatalf("unexpected state root: %s", manager.root)
	}
}
