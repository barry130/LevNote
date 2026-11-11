//go:build windows

package engineruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levnote.local/native/core/engineipc"
)

func TestBundleManifestFailuresNeverStart(t *testing.T) {
	for _, mode := range []string{"hash", "size", "version", "protocol", "unknown", "duplicate", "case", "null", "nonascii", "surrogate", "utf8", "oversize", "missing-exe", "file-size-changed"} {
		t.Run(mode, func(t *testing.T) {
			dir, exe := makeBundle(t)
			path := filepath.Join(filepath.Dir(exe), "manifest.json")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var m map[string]any
			if err := json.Unmarshal(data, &m); err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "hash":
				m["sha256"] = strings.Repeat("0", 64)
			case "size":
				m["size"] = 1
			case "version":
				m["engineVersion"] = "has space"
			case "protocol":
				m["protocol"] = 2
			case "unknown":
				m["command"] = "arbitrary.exe"
			case "case":
				m["Engine"] = m["engine"]
				delete(m, "engine")
			case "null":
				m["size"] = nil
			case "nonascii":
				m["engineVersion"] = "版本"
			case "missing-exe":
				if err := os.Remove(exe); err != nil {
					t.Fatal(err)
				}
			case "file-size-changed":
				if err := os.WriteFile(exe, []byte("not-executable"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			data, _ = json.Marshal(m)
			switch mode {
			case "duplicate":
				data = append([]byte(`{"schema":1,`), data[1:]...)
			case "surrogate":
				data = []byte(strings.Replace(string(data), helperVersion, `\uD800`, 1))
			case "utf8":
				data = append(data, 0xff)
			case "oversize":
				data = append(data, []byte(strings.Repeat(" ", manifestLimit))...)
			}
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			r := newRunner(dir)
			defer r.Close()
			starts := 0
			r.hooks = &testHooks{afterStart: func(*os.Process) { starts++ }}
			result, err := r.Export(context.Background(), engineipc.MarkdownHTML, sourceOf("x"))
			kind := FailureIntegrity
			if mode == "missing-exe" {
				kind = FailureUnavailable
			}
			requireFailure(t, result, err, kind, false)
			if starts != 0 {
				t.Fatal("invalid bundle launched")
			}
		})
	}
}

func TestRuntimeVerifiedButUnlaunchableBundleRetainsInfo(t *testing.T) {
	dir, exe := makeBundle(t)
	data := []byte("not a Windows executable")
	if err := os.WriteFile(exe, data, 0600); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(filepath.Dir(exe), "manifest.json")
	metadata, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var m bundleManifest
	if err := json.Unmarshal(metadata, &m); err != nil {
		t.Fatal(err)
	}
	m.Size = int64(len(data))
	digest := sha256.Sum256(data)
	m.SHA256 = hex.EncodeToString(digest[:])
	metadata, _ = json.Marshal(m)
	if err := os.WriteFile(manifestPath, metadata, 0600); err != nil {
		t.Fatal(err)
	}
	r := newRunner(dir)
	defer r.Close()
	result, err := r.Export(context.Background(), engineipc.MarkdownHTML, sourceOf("x"))
	requireFailure(t, result, err, FailureUnavailable, true)
	if result.Engine.SHA256 != m.SHA256 {
		t.Fatal("verified metadata lost after Start refusal")
	}
}

func TestBundleLocksReplacementAndDirectoryRenameUntilClose(t *testing.T) {
	dir, exe := makeBundle(t)
	bundle, err := verifyBundle(context.Background(), dir, engineipc.MarkdownHTML)
	if err != nil {
		t.Fatal(err)
	}
	defer bundle.close()
	for _, path := range []string{exe, filepath.Join(filepath.Dir(exe), "manifest.json")} {
		file, err := os.OpenFile(path, os.O_WRONLY, 0)
		if err == nil {
			file.Close()
			t.Fatalf("verified file writable: %s", path)
		}
		if err := os.Rename(path, path+".moved"); err == nil {
			t.Fatalf("verified file rename succeeded: %s", path)
		}
	}
	for _, path := range []string{filepath.Dir(exe), filepath.Join(dir, "engines"), dir} {
		if err := os.Rename(path, path+".moved"); err == nil {
			t.Fatalf("locked directory rename succeeded: %s", path)
		}
	}
	if err := bundle.close(); err != nil {
		t.Fatal(err)
	}
	file, err := os.OpenFile(exe, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("lock retained after close: %v", err)
	}
	file.Close()
	engineDir := filepath.Dir(exe)
	if err := os.Rename(engineDir, engineDir+".moved"); err != nil {
		t.Fatalf("directory lock retained: %v", err)
	}
	if err := os.Rename(engineDir+".moved", engineDir); err != nil {
		t.Fatal(err)
	}
}

func TestBundleRejectsReparse(t *testing.T) {
	dir, exe := makeBundle(t)
	original := exe + ".original"
	if err := os.Rename(exe, original); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(original, exe); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	r := newRunner(dir)
	defer r.Close()
	result, err := r.Export(context.Background(), engineipc.MarkdownHTML, sourceOf("x"))
	requireFailure(t, result, err, FailureIntegrity, false)
}

func TestBundleCanceledBeforeVerifyAndFailureLeavesHandlesClosed(t *testing.T) {
	dir, exe := makeBundle(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := verifyBundle(ctx, dir, engineipc.MarkdownHTML); err == nil {
		t.Fatal("canceled verification succeeded")
	}
	manifest := filepath.Join(filepath.Dir(exe), "manifest.json")
	if err := os.WriteFile(manifest, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := verifyBundle(context.Background(), dir, engineipc.MarkdownHTML); err == nil {
		t.Fatal("invalid manifest succeeded")
	}
	if err := os.Rename(filepath.Dir(exe), filepath.Dir(exe)+".moved"); err != nil {
		t.Fatalf("failed verification leaked locks: %v", err)
	}
}
