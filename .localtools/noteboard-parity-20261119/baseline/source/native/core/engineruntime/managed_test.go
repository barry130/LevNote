//go:build windows

package engineruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"levnote.local/native/core/enginecatalog"
	"levnote.local/native/core/engineipc"
)

func managedInstall(t *testing.T, m *PackageManager, engine enginecatalog.EngineID, version string) (*storeFixture, PackageReceipt) {
	t.Helper()
	f := newStoreFixture(t, engine, version)
	r, err := m.ImportFromReleaseDir(context.Background(), f.dir, engine)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Activate(context.Background(), engine, r.Digest); err != nil {
		t.Fatal(err)
	}
	return f, r
}

func TestManagedConstructorAndCorruptStateRefusal(t *testing.T) {
	for _, manager := range []*PackageManager{nil, {}} {
		if r, err := NewManaged(manager); r != nil || err == nil {
			t.Fatal("uninitialized manager accepted", r, err)
		}
	}
	m, host, state := newStoreManager(t)
	// No bundle exists; constructor must still succeed and inspect no engine.
	r, err := NewManaged(m)
	if err != nil || r == nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	storeWrite(t, filepath.Join(m.root, packageStateName), []byte(`{"schema":1,"schema":1}`))
	blocked, err := NewPackageManager(host, state)
	if err != nil || blocked.LoadError() == nil {
		t.Fatal("bad state fixture", err)
	}
	loadErr := blocked.LoadError()
	r, err = NewManaged(blocked)
	if err != nil {
		t.Fatal("constructor must retain repairable corrupt manager", err)
	}
	defer r.Close()
	starts, verifies := 0, 0
	r.hooks = &testHooks{afterVerify: func(*verifiedBundle) { verifies++ }, afterStart: func(*os.Process) { starts++ }}
	src := &untouchedSource{}
	result, err := r.Export(context.Background(), engineipc.MarkdownHTML, Source{Size: 1, Data: src})
	requireFailure(t, result, err, FailureIntegrity, false)
	if !errors.Is(err, loadErr) || src.calls != 0 || starts != 0 || verifies != 0 || r.active != nil {
		t.Fatal("corrupt state lost exact cause or proceeded", err, starts, verifies, src.calls)
	}
	if err := blocked.RepairResetAll(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, receipt := managedInstall(t, blocked, enginecatalog.MarkdownHTML, "repaired.v1")
	ctx, cancel := context.WithCancel(context.Background())
	r.hooks.afterVerify = func(*verifiedBundle) { verifies++; cancel() }
	result, err = r.Export(ctx, engineipc.MarkdownHTML, sourceOf("same runner after repair"))
	requireFailure(t, result, err, FailureCanceled, true)
	if result.Engine.PackageDigest != receipt.Digest || starts != 0 || verifies != 1 {
		t.Fatal("repair not visible on next admission", result.Engine)
	}
}

func TestManagedAdmissionSnapshotAndNextRequestMetadata(t *testing.T) {
	m, _, _ := newStoreManager(t)
	_, first := managedInstall(t, m, enginecatalog.MarkdownHTML, "first.v1")
	secondFixture := newStoreFixture(t, enginecatalog.MarkdownHTML, "second.v2")
	second, err := m.ImportFromReleaseDir(context.Background(), secondFixture.dir, secondFixture.engine)
	if err != nil {
		t.Fatal(err)
	}
	r, err := NewManaged(m)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	entered, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	releaseAdmission := func() { releaseOnce.Do(func() { close(release) }) }
	defer releaseAdmission()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	starts := 0
	r.hooks = &testHooks{afterAdmission: func() { close(entered); <-release }, afterVerify: func(*verifiedBundle) { cancel() }, afterStart: func(*os.Process) { starts++ }}
	type outcome struct {
		result Result
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		result, err := r.Export(ctx, engineipc.MarkdownHTML, sourceOf("immutable"))
		done <- outcome{result, err}
	}()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("admission not reached")
	}
	// Selection changes before any verification. Capturing at afterVerify alone
	// would not prove the required admission-time snapshot.
	if err := m.Activate(context.Background(), enginecatalog.MarkdownHTML, second.Digest); err != nil {
		t.Fatal(err)
	}
	busySource := &untouchedSource{}
	busy, busyErr := r.Export(context.Background(), engineipc.MindmapSVG, Source{Size: 1, Data: busySource})
	requireFailure(t, busy, busyErr, FailureBusy, false)
	if busySource.calls != 0 {
		t.Fatal("busy request read source")
	}
	releaseAdmission()
	var completed outcome
	select {
	case completed = <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("snapshot request failed to join")
	}
	requireFailure(t, completed.result, completed.err, FailureCanceled, true)
	if completed.result.Engine.PackageDigest != first.Digest || completed.result.Engine.Version != "first.v1" || completed.result.Engine.Source != PackageInstalled || starts != 0 {
		t.Fatal("in-flight request retargeted", completed.result.Engine, starts)
	}
	r.hooks.afterAdmission = nil
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()
	r.hooks.afterVerify = func(*verifiedBundle) { cancel2() }
	next, err := r.Export(ctx2, engineipc.MarkdownHTML, sourceOf("next"))
	requireFailure(t, next, err, FailureCanceled, true)
	if next.Engine.PackageDigest != second.Digest || next.Engine.Version != "second.v2" || next.Engine.SHA256 == completed.result.Engine.SHA256 {
		t.Fatal("new request did not capture new bytes", next.Engine)
	}
}

func TestManagedAllIDsMetadataAndFullGuardOwnership(t *testing.T) {
	for _, engine := range []enginecatalog.EngineID{enginecatalog.MarkdownHTML, enginecatalog.MindmapSVG, enginecatalog.MindmapPreview} {
		t.Run(string(engine), func(t *testing.T) {
			m, _, _ := newStoreManager(t)
			f, receipt := managedInstall(t, m, engine, "held.v1")
			r, err := NewManaged(m)
			if err != nil {
				t.Fatal(err)
			}
			defer r.Close()
			descriptor, _ := enginecatalog.Lookup(engine)
			dir := m.installedPath(engine, receipt.Digest)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var held []*os.File
			var lockErrors []string
			starts := 0
			r.hooks = &testHooks{afterVerify: func(bundle *verifiedBundle) {
				if bundle.installed == nil {
					lockErrors = append(lockErrors, "missing verifier owner")
				} else {
					held = append(held, bundle.installed.guard.files...)
				}
				for _, file := range receipt.Files {
					path := filepath.Join(dir, filepath.FromSlash(file.Path))
					if writable, err := os.OpenFile(path, os.O_WRONLY, 0); err == nil {
						writable.Close()
						lockErrors = append(lockErrors, "writable "+file.Path)
					}
					if err := os.Rename(path, path+".moved"); err == nil {
						_ = os.Rename(path+".moved", path)
						lockErrors = append(lockErrors, "renamed "+file.Path)
					}
				}
				for _, path := range []string{dir, filepath.Join(dir, "attribution"), filepath.Dir(dir), m.root} {
					if err := os.Rename(path, path+".moved"); err == nil {
						_ = os.Rename(path+".moved", path)
						lockErrors = append(lockErrors, "renamed directory "+path)
					}
				}
				cancel()
			}, afterStart: func(*os.Process) { starts++ }}
			result, err := r.Export(ctx, engine, sourceOf("no process"))
			requireFailure(t, result, err, FailureCanceled, true)
			expected := EngineInfo{ID: engine, Version: "held.v1", Source: PackageInstalled, PackageDigest: receipt.Digest, SHA256: storeHash(f.bytes[descriptor.RelativePath+"/"+descriptor.Filename])}
			if result.Engine != expected || len(lockErrors) != 0 || starts != 0 || len(held) <= 18 {
				t.Fatal(result.Engine, expected, lockErrors, starts, len(held))
			}
			for _, file := range held {
				managedAssertClosedFile(t, file)
			}
			if err := os.Rename(dir, dir+".moved"); err != nil {
				t.Fatal("package directory remains held", err)
			}
			if err := os.Rename(dir+".moved", dir); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestManagedInstalledFailureNeverFallsBack(t *testing.T) {
	for _, mode := range []string{"missing-exe", "changed-license", "missing-directory", "state-version"} {
		t.Run(mode, func(t *testing.T) {
			m, host, state := newStoreManager(t)
			f, receipt := managedInstall(t, m, enginecatalog.MarkdownHTML, "chosen.v1")
			// A coherent bundled installation exists as a tempting fallback, but
			// selected-package refusal must not even verify it.
			for _, name := range []string{"engines/markdown-export/manifest.json", "engines/markdown-export/levnote-markdown-export.exe"} {
				storeWrite(t, filepath.Join(host, filepath.FromSlash(name)), f.bytes[name])
			}
			dir := m.installedPath(f.engine, receipt.Digest)
			switch mode {
			case "missing-exe":
				if err := os.Remove(filepath.Join(dir, "levnote-markdown-export.exe")); err != nil {
					t.Fatal(err)
				}
			case "changed-license":
				storeWrite(t, filepath.Join(dir, "LICENSE"), []byte("changed"))
			case "missing-directory":
				if err := os.Rename(dir, filepath.Join(t.TempDir(), "missing")); err != nil {
					t.Fatal(err)
				}
			case "state-version":
				path := filepath.Join(m.root, packageStateName)
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				storeWrite(t, path, bytes.Replace(data, []byte("chosen.v1"), []byte("other.v2"), 1))
				m, err = NewPackageManager(host, state)
				if err != nil || m.LoadError() != nil {
					t.Fatal(err, m.LoadError())
				}
			}
			r, err := NewManaged(m)
			if err != nil {
				t.Fatal(err)
			}
			defer r.Close()
			verifies, starts := 0, 0
			r.hooks = &testHooks{afterVerify: func(*verifiedBundle) { verifies++ }, afterStart: func(*os.Process) { starts++ }}
			source := &untouchedSource{}
			result, err := r.Export(context.Background(), f.engine, Source{Size: 1, Data: source})
			if err == nil || result.Payload != nil || result.Engine.ID != "" || verifies != 0 || starts != 0 || source.calls != 0 {
				t.Fatal("selected error fell back", result, err, verifies, starts, source.calls)
			}
			selected, selectionErr := m.selection(f.engine)
			if selectionErr != nil || selected.Source != PackageInstalled || selected.Digest != receipt.Digest {
				t.Fatal("failure mutated selection", selected, selectionErr)
			}
			if strings.Contains(err.Error(), "随应用打包") {
				t.Fatal("installed failure mislabeled bundled", err)
			}
		})
	}
}

func TestManagedBundledMetadataAndAdmissionResetSnapshot(t *testing.T) {
	m, host, _ := newStoreManager(t)
	f := newStoreFixture(t, enginecatalog.MarkdownHTML, "bundled.v1")
	for _, name := range []string{"engines/markdown-export/manifest.json", "engines/markdown-export/levnote-markdown-export.exe"} {
		storeWrite(t, filepath.Join(host, filepath.FromSlash(name)), f.bytes[name])
	}
	_, installed := managedInstall(t, m, enginecatalog.MarkdownHTML, "installed.v2")
	if err := m.Reset(context.Background(), enginecatalog.MarkdownHTML); err != nil {
		t.Fatal(err)
	}
	r, err := NewManaged(m)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var hookErr error
	r.hooks = &testHooks{afterAdmission: func() { hookErr = m.Activate(context.Background(), enginecatalog.MarkdownHTML, installed.Digest) }, afterVerify: func(*verifiedBundle) { cancel() }}
	result, err := r.Export(ctx, engineipc.MarkdownHTML, sourceOf("x"))
	requireFailure(t, result, err, FailureCanceled, true)
	if hookErr != nil || result.Engine.Source != PackageBundled || result.Engine.PackageDigest != "" || result.Engine.Version != "bundled.v1" {
		t.Fatal("bundled snapshot changed after admission", result.Engine, hookErr)
	}
	if status, _ := m.selection(enginecatalog.MarkdownHTML); status.Digest != installed.Digest {
		t.Fatal("fixture switch not applied")
	}
}

func TestManagedCleanupFailurePoisonsRunnerAndClose(t *testing.T) {
	m, _, _ := newStoreManager(t)
	_, receipt := managedInstall(t, m, enginecatalog.MarkdownHTML, "v1")
	r, err := NewManaged(m)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var handles []*os.File
	r.hooks = &testHooks{afterVerify: func(bundle *verifiedBundle) {
		handles = append(handles, bundle.installed.guard.files...)
		_ = handles[len(handles)-1].Close()
		cancel()
	}}
	result, err := r.Export(ctx, engineipc.MarkdownHTML, sourceOf("x"))
	requireFailure(t, result, err, FailureCleanup, true)
	if result.Engine.PackageDigest != receipt.Digest {
		t.Fatal(result.Engine)
	}
	first, second := r.Close(), r.Close()
	if first == nil || !reflect.DeepEqual(first, second) {
		t.Fatal("cleanup result lost", first, second)
	}
	for _, file := range handles {
		managedAssertClosedFile(t, file)
	}
	result, err = r.Export(context.Background(), engineipc.MarkdownHTML, sourceOf("again"))
	requireFailure(t, result, err, FailureClosed, false)
}

func TestManagedRepositoryHelperLaunchWaitKeepsPackageHeld(t *testing.T) {
	m, _, _ := newStoreManager(t)
	f := newStoreFixture(t, enginecatalog.MarkdownHTML, helperVersion)
	exePath := makeBundleFor(t, f.dir, engineipc.MarkdownHTML)
	exe, err := os.ReadFile(exePath)
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(filepath.Dir(exePath), "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	f.put(t, "engines/markdown-export/levnote-markdown-export.exe", exe)
	f.put(t, "engines/markdown-export/manifest.json", manifest)
	f.save(t)
	receipt, err := m.ImportFromReleaseDir(context.Background(), f.dir, f.engine)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Activate(context.Background(), f.engine, receipt.Digest); err != nil {
		t.Fatal(err)
	}
	r, err := NewManaged(m)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	r.timeout = 5 * time.Second
	starts, waits := 0, 0
	var held []*os.File
	var lockErrors []string
	dir := m.installedPath(f.engine, receipt.Digest)
	checkHeld := func(phase string) {
		for _, file := range held {
			if _, err := file.Stat(); err != nil {
				lockErrors = append(lockErrors, phase+":closed "+file.Name())
			}
		}
		if err := os.Rename(dir, dir+".moved"); err == nil {
			_ = os.Rename(dir+".moved", dir)
			lockErrors = append(lockErrors, phase+":renamed package")
		}
	}
	r.hooks = &testHooks{afterVerify: func(bundle *verifiedBundle) { held = append(held, bundle.installed.guard.files...) }, afterStart: func(p *os.Process) {
		starts++
		t.Logf("owned installed repository test-helper pid=%d", p.Pid)
		checkHeld("start")
	}, afterWait: func() { waits++; checkHeld("wait") }}
	result, err := r.Export(context.Background(), engineipc.MarkdownHTML, sourceOf("中🙂<&\r\n"))
	if err != nil || string(result.Payload) != "<p>中🙂&lt;&amp;\r\n</p>" || result.Engine.Source != PackageInstalled || result.Engine.PackageDigest != receipt.Digest || result.Engine.SHA256 != storeHash(exe) || starts != 1 || waits != 1 || len(lockErrors) != 0 {
		t.Fatal(result.Engine, string(result.Payload), err, starts, waits, lockErrors)
	}
	for _, file := range held {
		managedAssertClosedFile(t, file)
	}
	if err := os.Rename(dir, dir+".moved"); err != nil {
		t.Fatal("package still held after successful return", err)
	}
	if err := os.Rename(dir+".moved", dir); err != nil {
		t.Fatal(err)
	}
	var written bundleManifest
	if err := json.Unmarshal(manifest, &written); err != nil || result.Engine.Version != written.EngineVersion {
		t.Fatal("verified version metadata", err)
	}
}

func managedAssertClosedFile(t *testing.T, file *os.File) {
	t.Helper()
	// Windows File.Stat calls GetFileType on the native descriptor, which returns
	// ERROR_INVALID_HANDLE after Close rather than wrapping os.ErrClosed. Inspect
	// the same owned File object; never reopen a path or recycled numeric handle.
	if file.Fd() != uintptr(syscall.InvalidHandle) {
		t.Fatal("owned file descriptor remains live", file.Name())
	}
	if _, err := file.Stat(); err == nil {
		t.Fatal("closed file still stat-able", file.Name())
	}
}
