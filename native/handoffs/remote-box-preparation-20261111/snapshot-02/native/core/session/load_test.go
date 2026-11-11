package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestLoadDenseTabsKeepsFirstDraftsAndCountsOmissions(t *testing.T) {
	const count = 100000
	path := filepath.Join(t.TempDir(), "dense.json")
	// 1.3 MiB, deliberately much smaller than the 16 MiB file budget.
	data := `{"version":1,"active":1,"tabs":[{"draft":"first"},` + strings.Repeat(`{"draft":""},`, count-2) + `{"draft":"last"}]}`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	start := time.Now()
	snapshot, issues := Load(path)
	elapsed := time.Since(start)
	runtime.ReadMemStats(&after)
	if len(snapshot.Tabs) != MaxTabs || snapshot.Active != 1 || *snapshot.Tabs[0].Draft != "first" {
		t.Fatal("bounded load lost initial drafts/selection", len(snapshot.Tabs), snapshot.Active)
	}
	if len(issues) != 1 || issues[0].Reason != "too-many-tabs" || issues[0].Detail != fmt.Sprintf("%d additional tabs omitted", count-MaxTabs) {
		t.Fatal("omitted count missing", issues)
	}
	allocated, allocations := after.TotalAlloc-before.TotalAlloc, after.Mallocs-before.Mallocs
	t.Logf("dense tabs input=%d elapsed=%s allocated=%d B allocations=%d", len(data), elapsed, allocated, allocations)
	// Input reading/JSON scanning may copy the bounded file, but omitted tabs
	// must not each allocate a RawMessage or decoded object.
	if allocations > 5000 || allocated > uint64(len(data))*12+(1<<20) {
		t.Fatal("load allocated per omitted tab", allocated, allocations)
	}
}

func TestBoundedTabScannerSkipsNestedUnknownFieldsAndInvalidEntries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested.json")
	data := `{"version":1,"active":2,"tabs":[{"draft":"first ],\\\" ok","unknown":[{"array":[1,2]}]},false,{"draft":"third"},null]}`
	if !json.Valid([]byte(data)) {
		t.Fatal("bad fixture")
	}
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	snapshot, issues := Load(path)
	if len(snapshot.Tabs) != 2 || snapshot.Active != 1 || *snapshot.Tabs[1].Draft != "third" || len(issues) != 2 {
		t.Fatal(snapshot, issues)
	}
	for _, issue := range issues {
		if issue.Reason != "invalid-tab" {
			t.Fatal(issues)
		}
	}
	for _, invalid := range []string{`{"version":1,"tabs":{}}`, `{"version":1,"tabs":[{},]}`, `{"version":1,"tabs":[]} false`} {
		_ = os.WriteFile(path, []byte(invalid), 0600)
		if _, issues := Load(path); len(issues) != 1 || issues[0].Reason != "corrupt" {
			t.Fatal("invalid array accepted", invalid, issues)
		}
	}
}

func TestLoadRejectsNULRootWithoutLosingDraft(t *testing.T) {
	path := filepath.Join(t.TempDir(), "root.json")
	value := "draft"
	data, err := json.Marshal(Snapshot{Version: Version, Root: t.TempDir() + "\x00suffix", Tabs: []Tab{{Draft: &value}}})
	if err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(path, data, 0600)
	snapshot, issues := Load(path)
	if snapshot.Root != "" || len(snapshot.Tabs) != 1 || len(issues) != 1 || issues[0].Reason != "invalid-root" {
		t.Fatal(snapshot, issues)
	}
}
