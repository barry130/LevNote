//go:build windows

package engineruntime

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"levnote.local/native/core/enginecatalog"
)

// Independently authored required names: these assertions must not inherit a
// production omission from packageNotices or a serializer-produced receipt.
var storeNoticeOracle = []string{
	"LICENSE", "README.md", "THIRD_PARTY_NOTICES.md", "attribution/SOURCE_THIRD_PARTY_NOTICES.txt",
	"attribution/mygo-README.md", "attribution/purego-README.md", "attribution/mygo-v0.8.3-no-startup-cleanup.patch",
	"attribution/dependency-patches-README.md", "licenses/Go-BSD-3-Clause.txt", "licenses/go-text-typesetting.txt",
	"licenses/mygo-MIT.txt", "licenses/purego-Apache-2.0.txt", "licenses/x-image-BSD-3-Clause.txt",
	"licenses/x-net-BSD-3-Clause.txt", "licenses/x-text-BSD-3-Clause.txt",
}

type storeFixture struct {
	dir, inventory string
	engine         enginecatalog.EngineID
	version        string
	entries        []map[string]any
	bytes          map[string][]byte
}

func storeWrite(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func storeHash(data []byte) string { hash := sha256.Sum256(data); return hex.EncodeToString(hash[:]) }

func newStoreFixture(t *testing.T, engine enginecatalog.EngineID, version string) *storeFixture {
	t.Helper()
	descriptor, ok := enginecatalog.Lookup(engine)
	if !ok {
		t.Fatal("bad fixture ID")
	}
	f := &storeFixture{dir: t.TempDir(), engine: engine, version: version, bytes: make(map[string][]byte)}
	f.inventory = filepath.Join(f.dir, "release-manifest.json")
	for _, name := range storeNoticeOracle {
		f.put(t, name, []byte("authored attribution: "+name+"\r\n"))
	}
	executable := []byte("inert bytes; never execute this fixture: " + string(engine) + ":" + version)
	f.put(t, descriptor.RelativePath+"/"+descriptor.Filename, executable)
	manifest := fmt.Sprintf(`{"schema":1,"engine":%q,"engineVersion":%q,"protocol":1,"format":%q,"goos":"windows","goarch":"amd64","filename":%q,"size":%d,"sha256":%q}`, engine, version, descriptor.Format, descriptor.Filename, len(executable), storeHash(executable))
	f.put(t, descriptor.RelativePath+"/manifest.json", []byte(manifest))
	// Valid inventory entries whose actual files are absent. Importing one fixed
	// engine must not try to open a host executable or another engine's files.
	f.entries = append(f.entries, map[string]any{"path": "levnote-native.exe", "size": int64(999), "sha256": strings.Repeat("a", 64)})
	f.entries = append(f.entries, map[string]any{"path": "engines/unrelated/unavailable.exe", "size": int64(999), "sha256": strings.Repeat("b", 64)})
	f.save(t)
	return f
}

func (f *storeFixture) put(t *testing.T, path string, data []byte) {
	t.Helper()
	f.bytes[path] = append([]byte(nil), data...)
	entry := map[string]any{"path": path, "size": int64(len(data)), "sha256": storeHash(data)}
	for i, old := range f.entries {
		if old["path"] == path {
			f.entries[i] = entry
			storeWrite(t, filepath.Join(f.dir, filepath.FromSlash(path)), data)
			return
		}
	}
	f.entries = append(f.entries, entry)
	storeWrite(t, filepath.Join(f.dir, filepath.FromSlash(path)), data)
}

func (f *storeFixture) save(t *testing.T) {
	t.Helper()
	data, err := json.Marshal(map[string]any{"schema": 1, "version": f.version, "goos": "windows", "goarch": "amd64", "goamd64": "v1", "excludes": []string{"release-manifest.json"}, "scope": "fixture fixed files", "configuration": "local test state", "files": f.entries})
	if err != nil {
		t.Fatal(err)
	}
	storeWrite(t, f.inventory, data)
	f.bytes["release-manifest.json"] = data
}

func newStoreManager(t *testing.T) (*PackageManager, string, string) {
	t.Helper()
	host, state := t.TempDir(), t.TempDir()
	m, err := NewPackageManager(host, state)
	if err != nil {
		t.Fatal(err)
	}
	return m, host, state
}

func storeAssertNoStage(t *testing.T, m *PackageManager) {
	t.Helper()
	entries, err := os.ReadDir(m.root)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".import-") || strings.HasPrefix(entry.Name(), ".selection-") {
			t.Fatal("owned temporary path leaked", entry.Name())
		}
	}
}

func TestPackageStoreFourEnginesExactCopiesSelectionAndReopen(t *testing.T) {
	m, host, state := newStoreManager(t)
	ctx := context.Background()
	expectedInitial := bundledPackageStatuses()
	if m.LoadError() != nil || !reflect.DeepEqual(m.Snapshot(), expectedInitial[:]) {
		t.Fatal("initial selection", m.LoadError(), m.Snapshot())
	}
	initial := m.Snapshot()
	if len(initial) != 4 {
		t.Fatal(initial)
	}
	for _, engine := range []enginecatalog.EngineID{enginecatalog.MarkdownHTML, enginecatalog.MindmapSVG, enginecatalog.MindmapPreview, enginecatalog.DrawioSVG} {
		f := newStoreFixture(t, engine, "test.v1")
		receipt, err := m.ImportFromReleaseDir(ctx, f.dir, engine)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(m.Snapshot(), initial) || receipt.Engine != engine || receipt.Version != "test.v1" || len(receipt.Files) != 18 || !validPackageDigest(receipt.Digest) {
			t.Fatal("import activated or bad receipt", receipt, m.Snapshot())
		}
		descriptor, _ := enginecatalog.Lookup(engine)
		expectedNames := append(append([]string(nil), storeNoticeOracle...), "manifest.json", descriptor.Filename, "release-manifest.json")
		sort.Strings(expectedNames)
		var gotNames []string
		for _, file := range receipt.Files {
			gotNames = append(gotNames, file.Path)
			source := file.Path
			if source == "manifest.json" || source == descriptor.Filename {
				source = descriptor.RelativePath + "/" + source
			}
			actual, err := os.ReadFile(filepath.Join(m.installedPath(engine, receipt.Digest), filepath.FromSlash(file.Path)))
			if err != nil || !bytes.Equal(actual, f.bytes[source]) || int64(len(actual)) != file.Size || storeHash(actual) != file.SHA256 {
				t.Fatal("not exact copied bytes", file, err)
			}
		}
		if !reflect.DeepEqual(gotNames, expectedNames) {
			t.Fatal(gotNames, expectedNames)
		}
		repeat, err := m.ImportFromReleaseDir(ctx, f.dir, engine)
		if err != nil || !reflect.DeepEqual(receipt, repeat) {
			t.Fatal("exact repeat mismatch", err)
		}
		receipt.Files[0].Path = "caller corruption"
		list, err := m.Installed(ctx, engine)
		if err != nil || len(list) != 1 || list[0].Files[0].Path == "caller corruption" {
			t.Fatal("receipt aliases store", err)
		}
		list[0].Files[0].SHA256 = "caller corruption"
		if err = m.Activate(ctx, engine, repeat.Digest); err != nil {
			t.Fatal(err)
		}
		selected, err := m.selection(engine)
		if err != nil || selected.Source != PackageInstalled || selected.Digest != repeat.Digest || selected.Version != "test.v1" {
			t.Fatal(selected, err)
		}
		copy := m.Snapshot()
		copy[0].Digest = "caller corruption"
		reopened, err := NewPackageManager(host, state)
		if err != nil || reopened.LoadError() != nil || !reflect.DeepEqual(reopened.Snapshot(), m.Snapshot()) {
			t.Fatal("state reopen", err)
		}
		if err = m.Reset(ctx, engine); err != nil {
			t.Fatal(err)
		}
		if lenMustInstalled(t, m, engine) != 1 {
			t.Fatal("reset deleted retained package")
		}
		initial = m.Snapshot()
	}
	storeAssertNoStage(t, m)
}

func TestPackageStateSchemaOneMigrationAndSchemaTwoPersistence(t *testing.T) {
	host, state := t.TempDir(), t.TempDir()
	stateRoot := filepath.Join(state, "engine-packages")
	path := filepath.Join(stateRoot, packageStateName)
	digest := strings.Repeat("a", 64)
	schemaOne := []byte(`{"schema":1,"selections":[` +
		`{"engine":"levnote.markdown-export","digest":"` + digest + `","version":"installed.v1"},` +
		`{"engine":"levnote.mindmap-svg","digest":"","version":""},` +
		`{"engine":"levnote.mindmap-preview","digest":"","version":""}]}`)
	storeWrite(t, path, schemaOne)
	m, err := NewPackageManager(host, state)
	if err != nil || m.LoadError() != nil {
		t.Fatal(err, m.LoadError())
	}
	beforeLoad, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	statuses := m.Snapshot()
	if len(statuses) != 4 {
		t.Fatalf("schema1 snapshot=%+v", statuses)
	}
	for index, status := range statuses {
		if status.Engine != packageEngines[index] {
			t.Fatalf("status order=%+v", statuses)
		}
		if status.Engine == enginecatalog.MarkdownHTML {
			if status.Source != PackageInstalled || status.Digest != digest || status.Version != "installed.v1" {
				t.Fatalf("old installed selection lost: %+v", status)
			}
		} else if status.Source != PackageBundled || status.Digest != "" || status.Version != "" {
			t.Fatalf("old/new bundled selection=%+v", status)
		}
	}
	if !bytes.Equal(beforeLoad, schemaOne) {
		t.Fatal("schema1 load rewrote state file")
	}

	if err := m.Reset(context.Background(), enginecatalog.DrawioSVG); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var persisted packageState
	if json.Unmarshal(written, &persisted) != nil || persisted.Schema != 2 {
		t.Fatalf("persisted state schema=%d data=%s", persisted.Schema, written)
	}
	if len(persisted.Selections) != 4 || persisted.Selections[0].Digest != digest ||
		persisted.Selections[3].Engine != enginecatalog.DrawioSVG || persisted.Selections[3].Digest != "" {
		t.Fatalf("schema2 selections=%+v", persisted.Selections)
	}
	reopened, err := NewPackageManager(host, state)
	if err != nil || reopened.LoadError() != nil || !reflect.DeepEqual(reopened.Snapshot(), statuses) {
		t.Fatalf("schema2 reopen=%+v err=%v load=%v", reopened.Snapshot(), err, reopened.LoadError())
	}

	invalidSchemaOne := bytes.Replace(schemaOne, []byte("levnote.mindmap-preview"), []byte("levnote.drawio-svg"), 1)
	got, err := parsePackageState(invalidSchemaOne)
	if err == nil || got != ([4]PackageStatus{}) {
		t.Fatalf("schema1 accepted drawio in place of legacy preview: %+v %v", got, err)
	}

	entry := func(engine enginecatalog.EngineID) string {
		return `{"engine":"` + string(engine) + `","digest":"","version":""}`
	}
	stateJSON := func(schema int, engines ...enginecatalog.EngineID) []byte {
		entries := make([]string, len(engines))
		for index, engine := range engines {
			entries[index] = entry(engine)
		}
		return []byte(fmt.Sprintf(`{"schema":%d,"selections":[%s]}`, schema, strings.Join(entries, ",")))
	}
	for name, data := range map[string][]byte{
		"schema1-short": stateJSON(1, enginecatalog.MarkdownHTML, enginecatalog.MindmapSVG),
		"schema2-short": stateJSON(2, enginecatalog.MarkdownHTML, enginecatalog.MindmapSVG, enginecatalog.MindmapPreview),
		"schema2-extra": stateJSON(2, enginecatalog.MarkdownHTML, enginecatalog.MindmapSVG, enginecatalog.MindmapPreview, enginecatalog.DrawioSVG, enginecatalog.DrawioSVG),
	} {
		t.Run(name, func(t *testing.T) {
			got, err := parsePackageState(data)
			if err == nil || got != ([4]PackageStatus{}) {
				t.Fatalf("wrong engine-set size accepted: %+v %v", got, err)
			}
		})
	}
}

func TestPackageStateLoadWithoutFileDoesNotWrite(t *testing.T) {
	host, state := t.TempDir(), t.TempDir()
	m, err := NewPackageManager(host, state)
	if err != nil || m.LoadError() != nil || len(m.Snapshot()) != 4 {
		t.Fatalf("initial manager=%+v err=%v load=%v", m.Snapshot(), err, m.LoadError())
	}
	path := filepath.Join(m.root, packageStateName)
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("no-state load wrote selections file: %v", err)
	}
}

func lenMustInstalled(t *testing.T, m *PackageManager, engine enginecatalog.EngineID) int {
	t.Helper()
	items, err := m.Installed(context.Background(), engine)
	if err != nil {
		t.Fatal(err)
	}
	return len(items)
}

func TestPackageStoreDigestIndependentFixedVector(t *testing.T) {
	files := []PackageFile{{Path: "z", Size: 2, SHA256: "202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f"}, {Path: "A", Size: 1, SHA256: "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"}}
	// Independently computed from literal domain/count/path/size/hash bytes with
	// SHA256.HashData, not the production digest helper or JSON representation.
	if got := packageDigest(files); got != "bf4009f0f694959d3b89df54f43214228419a5c40d44175a288a80ba105cb16b" {
		t.Fatal(got)
	}
	if files[0].Path != "z" {
		t.Fatal("digest sorted caller-owned slice")
	}
}

func TestPackageStoreRefusesMissingAndInconsistentFiles(t *testing.T) {
	for _, mode := range []string{"missing-license", "hash", "size", "manifest-engine", "manifest-version", "missing-manifest", "extra-installed", "corrupt-existing"} {
		t.Run(mode, func(t *testing.T) {
			m, _, _ := newStoreManager(t)
			f := newStoreFixture(t, enginecatalog.MarkdownHTML, "v1")
			ctx := context.Background()
			if mode == "extra-installed" || mode == "corrupt-existing" {
				r, err := m.ImportFromReleaseDir(ctx, f.dir, f.engine)
				if err != nil {
					t.Fatal(err)
				}
				name := "extra.txt"
				if mode == "corrupt-existing" {
					name = "LICENSE"
				}
				storeWrite(t, filepath.Join(m.installedPath(f.engine, r.Digest), name), []byte("tampered"))
			} else {
				switch mode {
				case "missing-license":
					if err := os.Remove(filepath.Join(f.dir, "licenses", "mygo-MIT.txt")); err != nil {
						t.Fatal(err)
					}
				case "hash":
					storeWrite(t, filepath.Join(f.dir, "LICENSE"), bytes.Repeat([]byte("x"), len(f.bytes["LICENSE"])))
				case "size":
					storeWrite(t, filepath.Join(f.dir, "LICENSE"), []byte("x"))
				case "missing-manifest":
					if err := os.Remove(filepath.Join(f.dir, "engines", "markdown-export", "manifest.json")); err != nil {
						t.Fatal(err)
					}
				case "manifest-engine", "manifest-version":
					name := "engines/markdown-export/manifest.json"
					value := string(f.bytes[name])
					if mode == "manifest-engine" {
						value = strings.Replace(value, "levnote.markdown-export", "levnote.mindmap-svg", 1)
					} else {
						value = strings.Replace(value, `"engineVersion":"v1"`, `"engineVersion":"v2"`, 1)
					}
					f.put(t, name, []byte(value))
					f.save(t)
				}
			}
			if _, err := m.ImportFromReleaseDir(ctx, f.dir, f.engine); err == nil {
				t.Fatal("bad package imported")
			}
			if status, _ := m.selection(f.engine); status.Source != PackageBundled {
				t.Fatal("refusal activated", status)
			}
			storeAssertNoStage(t, m)
		})
	}
}

func TestPackageStoreCapsAndBoundedStreaming(t *testing.T) {
	for _, mode := range []string{"inventory", "manifest", "exe-declared", "notices-total", "state"} {
		t.Run(mode, func(t *testing.T) {
			m, host, state := newStoreManager(t)
			f := newStoreFixture(t, enginecatalog.MarkdownHTML, "v1")
			switch mode {
			case "inventory":
				storeWrite(t, f.inventory, bytes.Repeat([]byte(" "), releaseInventoryLimit+1))
			case "manifest":
				f.put(t, "engines/markdown-export/manifest.json", bytes.Repeat([]byte(" "), manifestLimit+1))
				f.save(t)
			case "exe-declared":
				for _, entry := range f.entries {
					if strings.HasSuffix(entry["path"].(string), "levnote-markdown-export.exe") {
						entry["size"] = int64(executableLimit + 1)
					}
				}
				f.save(t)
			case "notices-total":
				for _, entry := range f.entries {
					if entry["path"] == "LICENSE" {
						entry["size"] = int64(packageNoticeLimit)
					}
				}
				f.save(t)
			case "state":
				storeWrite(t, filepath.Join(m.root, packageStateName), bytes.Repeat([]byte(" "), packageStateLimit+1))
				blocked, err := NewPackageManager(host, state)
				if err != nil || blocked.LoadError() == nil {
					t.Fatal("oversized state not blocked", err)
				}
				return
			}
			copied := false
			m.hooks = &packageTestHooks{afterCopyChunk: func(string, int64) { copied = true }}
			if _, err := m.ImportFromReleaseDir(context.Background(), f.dir, f.engine); err == nil || copied {
				t.Fatal("cap did not refuse before copy", err, copied)
			}
			storeAssertNoStage(t, m)
		})
	}
	path := filepath.Join(t.TempDir(), "data")
	storeWrite(t, path, bytes.Repeat([]byte("x"), 70001))
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	writer := &storeChunkWriter{}
	if _, err := packageHash(context.Background(), file, 70001, writer, nil); err != nil || writer.largest != 32<<10 || writer.total != 70001 {
		t.Fatal(writer, err)
	}
}

type storeChunkWriter struct{ largest, total int }

func (w *storeChunkWriter) Write(b []byte) (int, error) {
	if len(b) > w.largest {
		w.largest = len(b)
	}
	w.total += len(b)
	return len(b), nil
}

func TestPackageStoreCancellationPublicationFailureAndNonblockingSnapshot(t *testing.T) {
	m, _, _ := newStoreManager(t)
	f := newStoreFixture(t, enginecatalog.MarkdownHTML, "v1")
	f.put(t, "README.md", bytes.Repeat([]byte("a"), 100000))
	f.save(t)
	ctx, cancel := context.WithCancel(context.Background())
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	released := false
	defer func() {
		cancel()
		if !released {
			close(release)
		}
	}()
	m.hooks = &packageTestHooks{afterCopyChunk: func(path string, count int64) {
		if path == "README.md" && count == 32<<10 {
			close(entered)
			<-release
		}
	}}
	go func() { _, err := m.ImportFromReleaseDir(ctx, f.dir, f.engine); done <- err }()
	select {
	case <-entered:
	case err := <-done:
		t.Fatal("import failed before held copy", err)
	case <-time.After(3 * time.Second):
		t.Fatal("copy did not start")
	}
	// Only this test goroutine reads metadata while the import goroutine owns I/O.
	metadata := make(chan bool, 1)
	go func() { metadata <- len(m.Snapshot()) == 4 && m.LoadError() == nil }()
	select {
	case ok := <-metadata:
		if !ok {
			t.Fatal("metadata")
		}
	case <-time.After(time.Second):
		t.Fatal("Snapshot blocked behind import")
	}
	waitCtx, waitCancel := context.WithCancel(context.Background())
	waitStarted, waitDone := make(chan struct{}), make(chan error, 1)
	go func() {
		close(waitStarted)
		_, err := m.Installed(waitCtx, f.engine)
		waitDone <- err
	}()
	<-waitStarted
	waitCancel()
	select {
	case err := <-waitDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatal("waiting I/O gate did not honor cancellation", err)
		}
	case <-time.After(time.Second):
		t.Fatal("canceled I/O waiter remained blocked behind the copy")
	}
	cancel()
	close(release)
	released = true
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatal("copy cancellation", err)
	}
	storeAssertNoStage(t, m)
	if lenMustInstalled(t, m, f.engine) != 0 {
		t.Fatal("canceled package published")
	}
	m.hooks = &packageTestHooks{beforePublish: func() error { return errors.New("fixture publish denied") }}
	if _, err := m.ImportFromReleaseDir(context.Background(), f.dir, f.engine); err == nil {
		t.Fatal("publication failure hidden")
	}
	storeAssertNoStage(t, m)
	if lenMustInstalled(t, m, f.engine) != 0 {
		t.Fatal("failed publication retained package")
	}
	m.hooks = nil
	if _, err := m.ImportFromReleaseDir(context.Background(), f.dir, f.engine); err != nil {
		t.Fatal("store unusable after cleanup", err)
	}
}

func TestPackageStoreStateCorruptionAndExplicitRepair(t *testing.T) {
	m, host, state := newStoreManager(t)
	f := newStoreFixture(t, enginecatalog.MarkdownHTML, "v1")
	ctx := context.Background()
	r, err := m.ImportFromReleaseDir(ctx, f.dir, f.engine)
	if err != nil {
		t.Fatal(err)
	}
	if err = m.Activate(ctx, f.engine, r.Digest); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(m.root, packageStateName)
	good, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"duplicate", "case", "unknown", "null", "duplicate-engine", "upper-digest", "surrogate", "utf8", "truncated"} {
		t.Run(mode, func(t *testing.T) {
			bad := append([]byte(nil), good...)
			switch mode {
			case "duplicate":
				bad = append([]byte(`{"schema":1,`), bad[1:]...)
			case "case":
				bad = []byte(strings.Replace(string(bad), `"schema"`, `"Schema"`, 1))
			case "unknown":
				bad = append([]byte(`{"extra":1,`), bad[1:]...)
			case "null":
				bad = []byte(strings.Replace(string(bad), `"selections":[`, `"selections":null,"unused":[`, 1))
			case "duplicate-engine":
				bad = []byte(strings.Replace(string(bad), "levnote.mindmap-svg", "levnote.markdown-export", 1))
			case "upper-digest":
				bad = []byte(strings.Replace(string(bad), r.Digest, strings.ToUpper(r.Digest), 1))
			case "surrogate":
				bad = []byte(strings.Replace(string(bad), `"version":"v1"`, `"version":"\ud800"`, 1))
			case "utf8":
				bad = append(bad, 0xff)
			case "truncated":
				bad = bad[:20]
			}
			storeWrite(t, path, bad)
			blocked, err := NewPackageManager(host, state)
			if err != nil || blocked.LoadError() == nil {
				t.Fatal("corrupt startup not explicit", err)
			}
			if _, err = blocked.selection(f.engine); err == nil {
				t.Fatal("corrupt preferences silently resolved bundled")
			}
			if err = blocked.Activate(ctx, f.engine, r.Digest); err == nil {
				t.Fatal("activate overwrote corrupt other choices")
			}
			if err = blocked.Reset(ctx, f.engine); err == nil {
				t.Fatal("reset overwrote corrupt other choices")
			}
			after, _ := os.ReadFile(path)
			if !bytes.Equal(after, bad) {
				t.Fatal("read/refusal mutated state")
			}
			blocked.hooks = &packageTestHooks{beforeStateCommit: func() error { return errors.New("denied") }}
			if err = blocked.RepairResetAll(ctx); err == nil || blocked.LoadError() == nil {
				t.Fatal("failed repair unblocked state", err)
			}
			after, _ = os.ReadFile(path)
			if !bytes.Equal(after, bad) {
				t.Fatal("failed repair replaced state")
			}
			blocked.hooks = nil
			if err = blocked.RepairResetAll(ctx); err != nil || blocked.LoadError() != nil {
				t.Fatal("repair", err)
			}
			if lenMustInstalled(t, blocked, f.engine) != 1 {
				t.Fatal("repair removed package")
			}
			storeAssertNoStage(t, blocked)
		})
	}
}

func TestPackageStoreWriteRefusalPreservesLiveAndDisk(t *testing.T) {
	m, _, _ := newStoreManager(t)
	ctx := context.Background()
	f := newStoreFixture(t, enginecatalog.MarkdownHTML, "v1")
	r, err := m.ImportFromReleaseDir(ctx, f.dir, f.engine)
	if err != nil {
		t.Fatal(err)
	}
	if err = m.Reset(ctx, f.engine); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(m.root, packageStateName)
	before, _ := os.ReadFile(path)
	status := m.Snapshot()
	m.hooks = &packageTestHooks{beforeStateCommit: func() error { return errors.New("publish denied") }}
	if err = m.Activate(ctx, f.engine, r.Digest); err == nil {
		t.Fatal("state write refusal hidden")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) || !reflect.DeepEqual(status, m.Snapshot()) {
		t.Fatal("failed activation mutated choice")
	}
	storeAssertNoStage(t, m)
}

func TestPackageStoreInstalledKeepsHealthyChoicesBesideCorruptSelection(t *testing.T) {
	m, _, _ := newStoreManager(t)
	ctx := context.Background()
	first := newStoreFixture(t, enginecatalog.MarkdownHTML, "healthy.v1")
	good, err := m.ImportFromReleaseDir(ctx, first.dir, first.engine)
	if err != nil {
		t.Fatal(err)
	}
	second := newStoreFixture(t, first.engine, "corrupt.v2")
	bad, err := m.ImportFromReleaseDir(ctx, second.dir, second.engine)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Activate(ctx, first.engine, bad.Digest); err != nil {
		t.Fatal(err)
	}
	storeWrite(t, filepath.Join(m.installedPath(first.engine, bad.Digest), "LICENSE"), []byte("corrupt"))
	list, err := m.Installed(ctx, first.engine)
	if len(list) != 1 || list[0].Digest != good.Digest || err == nil || !strings.Contains(err.Error(), bad.Digest) {
		t.Fatal("one bad package hid healthy options or its own issue", list, err)
	}
	selected, err := m.selection(first.engine)
	if err != nil || selected.Source != PackageInstalled || selected.Digest != bad.Digest {
		t.Fatal("corrupt selected package silently switched", selected, err)
	}
	if err := m.Activate(ctx, first.engine, bad.Digest); err == nil {
		t.Fatal("activation trusted cached verification of corrupt package")
	}
	if err := m.Activate(ctx, first.engine, good.Digest); err != nil {
		t.Fatal("healthy explicit selection unusable", err)
	}
	selected, _ = m.selection(first.engine)
	if selected.Digest != good.Digest {
		t.Fatal("explicit healthy selection not applied")
	}
}

func TestPackageStoreTwelveLimitExactReuseAndMissingSelected(t *testing.T) {
	m, host, state := newStoreManager(t)
	ctx := context.Background()
	var first *storeFixture
	var receipts []PackageReceipt
	for i := 0; i < 12; i++ {
		f := newStoreFixture(t, enginecatalog.MarkdownHTML, fmt.Sprintf("v%d", i))
		if i == 0 {
			first = f
		}
		r, err := m.ImportFromReleaseDir(ctx, f.dir, f.engine)
		if err != nil {
			t.Fatal(i, err)
		}
		receipts = append(receipts, r)
	}
	if lenMustInstalled(t, m, first.engine) != 12 {
		t.Fatal("not exactly twelve")
	}
	if r, err := m.ImportFromReleaseDir(ctx, first.dir, first.engine); err != nil || r.Digest != receipts[0].Digest {
		t.Fatal("exact existing refused at quota", err)
	}
	extra := newStoreFixture(t, first.engine, "v13")
	copied := false
	m.hooks = &packageTestHooks{afterCopyChunk: func(string, int64) { copied = true }}
	if _, err := m.ImportFromReleaseDir(ctx, extra.dir, extra.engine); err == nil || copied {
		t.Fatal("thirteenth copied/published", err, copied)
	}
	m.hooks = nil
	if lenMustInstalled(t, m, first.engine) != 12 {
		t.Fatal("quota deleted previous content")
	}
	if err := m.Activate(ctx, first.engine, receipts[0].Digest); err != nil {
		t.Fatal(err)
	}
	path := m.installedPath(first.engine, receipts[0].Digest)
	moved := filepath.Join(t.TempDir(), "held-missing-package")
	if err := os.Rename(path, moved); err != nil {
		t.Fatal(err)
	}
	list, err := m.Installed(ctx, first.engine)
	if err == nil || len(list) != 11 || !strings.Contains(err.Error(), receipts[0].Digest) {
		t.Fatal("missing selected package silently omitted", len(list), err)
	}
	reopened, err := NewPackageManager(host, state)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := reopened.selection(first.engine)
	if err != nil || selected.Source != PackageInstalled || selected.Digest != receipts[0].Digest {
		t.Fatal("missing selection fell back", selected, err)
	}
	if _, err := reopened.verifiedInstalled(ctx, first.engine, selected.Digest); err == nil {
		t.Fatal("missing selected package verified")
	}
	storeAssertNoStage(t, m)
}

func TestPackageStoreLinksPathsAndInputRefusal(t *testing.T) {
	for _, mode := range []string{"hardlink", "symlink-file", "symlink-dir"} {
		t.Run(mode, func(t *testing.T) {
			m, _, _ := newStoreManager(t)
			f := newStoreFixture(t, enginecatalog.MarkdownHTML, "v1")
			outside := filepath.Join(t.TempDir(), "outside")
			storeWrite(t, outside, f.bytes["LICENSE"])
			if mode == "symlink-dir" {
				old := filepath.Join(f.dir, "licenses")
				target := filepath.Join(t.TempDir(), "licenses")
				if err := os.Rename(old, target); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, old); err != nil {
					t.Skipf("directory symlink unavailable: %v", err)
				}
			} else {
				path := filepath.Join(f.dir, "LICENSE")
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				var err error
				if mode == "hardlink" {
					err = os.Link(outside, path)
				} else {
					err = os.Symlink(outside, path)
				}
				if err != nil {
					t.Skipf("link unavailable: %v", err)
				}
			}
			if _, err := m.ImportFromReleaseDir(context.Background(), f.dir, f.engine); err == nil {
				t.Fatal("link imported")
			}
			actual, _ := os.ReadFile(outside)
			if !bytes.Equal(actual, f.bytes["LICENSE"]) {
				t.Fatal("external target changed")
			}
			storeAssertNoStage(t, m)
		})
	}
	m, _, _ := newStoreManager(t)
	ctx := context.Background()
	for _, path := range []string{"relative", `\\server\share\release`, `\\?\C:\release`, `C:\bad:stream`, `C:\bad\..\alias`} {
		if _, err := m.ImportFromReleaseDir(ctx, path, enginecatalog.MarkdownHTML); err == nil {
			t.Fatal("bad path accepted", path)
		}
	}
	if _, err := m.ImportFromReleaseDir(ctx, t.TempDir(), "unknown"); err == nil {
		t.Fatal("unknown ID")
	}
	for _, digest := range []string{"", strings.Repeat("A", 64), strings.Repeat("a", 63), "../escape"} {
		if err := m.Activate(ctx, enginecatalog.MarkdownHTML, digest); err == nil {
			t.Fatal("bad digest", digest)
		}
	}
	var zero PackageManager
	for _, manager := range []*PackageManager{nil, &zero, m} {
		if _, err := manager.Installed(nil, enginecatalog.MarkdownHTML); err == nil {
			t.Fatal("nil context/manager accepted")
		}
	}
}

func TestPackageStoreStreamShortWriteRefused(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bytes")
	storeWrite(t, path, []byte("abc"))
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := packageHash(context.Background(), f, 3, storeShortWriter{}, nil); !errors.Is(err, io.ErrShortWrite) {
		t.Fatal(err)
	}
}

type storeShortWriter struct{}

func (storeShortWriter) Write(p []byte) (int, error) { return len(p) - 1, nil }
