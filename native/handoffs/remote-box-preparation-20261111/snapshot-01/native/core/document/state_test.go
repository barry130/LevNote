package document

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestWritePrivateStateToRejectsNilBeforeCreatingDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing", "state.json")
	if err := WritePrivateStateTo(path, nil); err == nil {
		t.Fatal("nil private-state writer accepted")
	}
	if _, err := os.Stat(filepath.Dir(path)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("nil callback created the destination directory: %v", err)
	}
}

func TestWritePrivateStateToFailuresPreserveTargetAndRemoveTemp(t *testing.T) {
	injected := errors.New("injected private-state failure")
	for _, test := range []struct {
		name  string
		write func(io.Writer) error
		want  error
	}{
		{
			name: "sentinel",
			write: func(w io.Writer) error {
				if _, err := io.WriteString(w, "partial replacement"); err != nil {
					return err
				}
				return injected
			},
			want: injected,
		},
		{
			name: "short-write",
			write: func(w io.Writer) error {
				if _, err := io.WriteString(w, "partial replacement"); err != nil {
					return err
				}
				return io.ErrShortWrite
			},
			want: io.ErrShortWrite,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "state.json")
			if err := os.WriteFile(path, []byte("old recovery"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := WritePrivateStateTo(path, test.write); !errors.Is(err, test.want) {
				t.Fatalf("WritePrivateStateTo error = %v, want %v", err, test.want)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != "old recovery" {
				t.Fatalf("failed callback changed target: %q", got)
			}
			assertNoSaveTemp(t, dir)
		})
	}
}

func assertNoSaveTemp(t *testing.T, dir string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, ".levnote-save-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary save files remain: %v", matches)
	}
}
