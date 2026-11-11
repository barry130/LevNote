package session

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"levnote.local/native/core/document"
	"levnote.local/native/core/text"
)

func TestJSONCounterMatchesMarshalAcrossEscapesAndPieceSplits(t *testing.T) {
	values := []string{"", "ordinary Chinese 中文🙂", "\b\f\n\r\t\x00\x1f\"\\<>&\u2028\u2029", "�", "\xff\xc0\xe2\x82", strings.Repeat("中🙂\b<&", 1000)}
	for _, value := range values {
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if got := draftEncodedBytes(value); got != len(encoded) {
			t.Fatalf("string %q: encoded=%d want=%d", value[:min(len(value), 24)], got, len(encoded))
		}
		// Arbitrary UTF-8 splits, alternating both io.Writer entry points.
		for step := 1; step <= 5; step++ {
			var counter jsonStringCounter
			for i := 0; i < len(value); i += step {
				piece := value[i:min(i+step, len(value))]
				if i%2 == 0 {
					_, err = counter.WriteString(piece)
				} else {
					_, err = counter.Write([]byte(piece))
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			if got := counter.Finish(); got != len(encoded) {
				t.Fatalf("step %d %q: encoded=%d want=%d", step, value[:min(len(value), 24)], got, len(encoded))
			}
		}
	}
	value := strings.Repeat("🙂中文<&\b", 1000)
	buffer := text.New(value)
	encoded, _ := json.Marshal(value)
	if size, err := draftEncodedLength(buffer); err != nil || size != len(encoded) {
		t.Fatal(size, len(encoded), err)
	}
}

func TestSnapshotPreflightEqualsSerializedSize(t *testing.T) {
	root := t.TempDir()
	draft := "中🙂\b\f<&>\u2028\u2029\x00\"\\"
	falseValue := false
	for _, scroll := range []float32{0, 1, 1e-8, 1e-6, 1e20, 1e21, math.MaxFloat32, math.SmallestNonzeroFloat32} {
		snapshots := []Snapshot{
			{Version: Version},
			{Version: Version, Tabs: []Tab{}, Preferences: Preferences{LineNumbers: &falseValue}},
			{Version: Version, Root: root, Tabs: []Tab{{Draft: &draft, Scroll: scroll}}},
			{Version: Version, Root: root, Active: 1, Tabs: []Tab{
				{Path: filepath.Join(root, "<&>\"中.txt"), Title: "\b\f\n", Scroll: scroll},
				{Draft: &draft, Path: filepath.Join(root, "a.txt"), OriginalPath: filepath.Join(root, "original.txt"), Title: "中文", Encoding: document.UTF16LE, BOM: true, Selection: text.Range{Anchor: 12345, Head: 42}, PluginID: "example.plugin", View: "split", Scroll: scroll},
			}, Preferences: Preferences{Theme: "dark", FontSize: 15.125, WordWrap: true, LineNumbers: &falseValue, SidebarWidth: 230.5}},
		}
		for _, snapshot := range snapshots {
			encoded, err := json.Marshal(snapshot)
			if err != nil {
				t.Fatal(err)
			}
			size, err := preflightSnapshot(snapshot)
			if err != nil || size != len(encoded) {
				t.Fatalf("scroll %g size %d want %d err %v JSON %s", scroll, size, len(encoded), err, encoded)
			}
		}
	}
}

func TestPreflightUsesActualBytesWithoutMetadataReserve(t *testing.T) {
	value := strings.Repeat("x", MaxDraftBytes-256)
	snapshot := Snapshot{Version: Version, Tabs: []Tab{{Draft: &value}, {Draft: &value}, {Draft: &value}, {Draft: &value}}}
	size, err := preflightSnapshot(snapshot)
	if err != nil || size <= MaxBytes-(1<<20) || size > MaxBytes {
		t.Fatal("ordinary drafts were overcharged by a metadata reserve", size, err)
	}
	// A single real metadata change can cross the precise remaining budget.
	snapshot.Tabs[0].Title = strings.Repeat("a", MaxBytes-size+1)
	if _, err := preflightSnapshot(snapshot); err == nil {
		t.Fatal("metadata above remaining budget accepted")
	}
}

func TestBuilderFailuresAreTransactionalAndNilSafe(t *testing.T) {
	builder := NewSnapshotBuilder("")
	if err := builder.AddDocument(nil, "nil", 0); err == nil {
		t.Fatal("nil document accepted")
	}
	if err := builder.AddDocument(&document.Document{}, "nil-buffer", 0); err == nil {
		t.Fatal("nil buffer accepted")
	}
	if _, err := CaptureBounded(nil, "nil", 0); err == nil {
		t.Fatal("nil capture accepted")
	}
	doc := document.New(strings.Repeat("x", 1<<20))
	doc.Encoding = "invalid"
	for i := 0; i < 20; i++ {
		if err := builder.AddDocument(doc, "invalid", 0); err == nil {
			t.Fatal("invalid encoding accepted")
		}
	}
	if builder.Len() != 0 || builder.tabsBytes != 0 || builder.rawDraftBytes != 0 {
		t.Fatal("failed additions consumed budget", builder)
	}
	doc.Encoding = document.UTF8
	if err := builder.AddDocument(doc, "corrected", 0); err != nil {
		t.Fatal("failed add poisoned later valid capture", err)
	}
	before := builder.tabsBytes
	if err := builder.SetActive(10); err == nil || builder.snapshot.Active != 0 {
		t.Fatal("failed active index changed builder", err)
	}
	if err := builder.SetPreferences(Preferences{Theme: "bad"}); err == nil || builder.snapshot.Preferences.Theme != "" || builder.tabsBytes != before {
		t.Fatal("failed preferences changed builder", err)
	}
	if err := builder.AddTab(Tab{}); err == nil || builder.Len() != 1 || builder.tabsBytes != before {
		t.Fatal("failed tab changed builder", err)
	}
	if err := NewSnapshotBuilder("").SetActive(0); err != nil {
		t.Fatal("empty session cannot select index zero", err)
	}
}

func TestBuilderSnapshotHasIndependentMutableFields(t *testing.T) {
	builder := NewSnapshotBuilder("")
	draft, numbers := "retained draft", true
	if err := builder.SetPreferences(Preferences{Theme: "dark", LineNumbers: &numbers}); err != nil {
		t.Fatal(err)
	}
	if err := builder.AddTab(Tab{Draft: &draft, View: "split"}); err != nil {
		t.Fatal(err)
	}
	draft, numbers = "mutated input", false
	first, err := builder.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if *first.Tabs[0].Draft != "retained draft" || !*first.Preferences.LineNumbers {
		t.Fatal("builder retained caller pointers")
	}
	*first.Tabs[0].Draft = "mutated output"
	first.Tabs[0].Title = "mutated title"
	*first.Preferences.LineNumbers = false
	second, err := builder.Snapshot()
	if err != nil || *second.Tabs[0].Draft != "retained draft" || second.Tabs[0].Title != "" || !*second.Preferences.LineNumbers {
		t.Fatal("snapshot changed builder", second, err)
	}
	if got, err := preflightSnapshot(second); err != nil {
		t.Fatal(err)
	} else if expected, _ := snapshotEnvelopeLength(builder.snapshot, builder.Len(), builder.tabsBytes, false); got != expected {
		t.Fatal("snapshot mutation invalidated accounting", got, expected)
	}
}

func TestBuilderCountsViewMetadataAndDiscardedRecordsBeforeCapture(t *testing.T) {
	root := t.TempDir()
	builder := NewSnapshotBuilder(root)
	if err := builder.SetPreferences(Preferences{Theme: "light", FontSize: 15, SidebarWidth: 230}); err != nil {
		t.Fatal(err)
	}
	options := CaptureOptions{Title: "draft", Scroll: 12, OriginalPath: filepath.Join(root, "original.md"), PluginID: "markdown", View: "preview"}
	if err := builder.AddDocumentWithOptions(document.New("draft🙂"), options); err != nil {
		t.Fatal(err)
	}
	if err := builder.AddTab(Tab{Path: filepath.Join(root, "discarded.md"), Title: "saved only", View: "source"}); err != nil {
		t.Fatal(err)
	}
	if err := builder.SetActive(1); err != nil {
		t.Fatal(err)
	}
	snapshot, err := builder.Snapshot()
	if err != nil || snapshot.Active != 1 || len(snapshot.Tabs) != 2 || snapshot.Tabs[1].Draft != nil || snapshot.Tabs[0].OriginalPath != options.OriginalPath || snapshot.Tabs[0].View != options.View {
		t.Fatal(snapshot, err)
	}
	// Metadata alone can exceed the JSON budget, even with a tiny draft.
	options.Title = strings.Repeat("\x00", 3<<20)
	before := builder.tabsBytes
	if err := builder.AddDocumentWithOptions(document.New("tiny"), options); err == nil || builder.Len() != 2 || builder.tabsBytes != before {
		t.Fatal("view metadata bypassed budget", err)
	}
}

func TestInvalidPreferencesAndMetadataPreserveExistingRecovery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")
	if err := Save(path, Snapshot{}); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	for _, p := range []Preferences{{Theme: "unknown"}, {FontSize: float32(math.NaN())}, {FontSize: 1}, {SidebarWidth: float32(math.Inf(1))}, {SidebarWidth: 1024}} {
		if err := Save(path, Snapshot{Preferences: p}); err == nil {
			t.Fatal("invalid preferences accepted", p)
		}
	}
	draft := "small"
	for _, tab := range []Tab{{Draft: &draft, Title: "\xff"}, {Draft: &draft, Scroll: float32(math.NaN())}, {Draft: &draft, PluginID: strings.Repeat("\x00", 3<<20)}} {
		if err := Save(path, Snapshot{Tabs: []Tab{tab}}); err == nil {
			t.Fatal("invalid metadata accepted")
		}
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("failed metadata/preference save replaced recovery")
	}
}

func TestRejectedSessionPreflightAllocationsAreBounded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	controls := strings.Repeat("\x00", 3<<20)
	doc := document.New(controls)
	huge := strings.Repeat("x", MaxBytes+1)
	cases := []struct {
		name string
		run  func() error
	}{
		{"direct-draft", func() error { return Save(path, Snapshot{Tabs: []Tab{{Draft: &controls}}}) }},
		{"buffer-draft", func() error { return NewSnapshotBuilder("").AddDocument(doc, "controls", 0) }},
		{"preferences", func() error { return Save(path, Snapshot{Preferences: Preferences{Theme: huge}}) }},
		{"path", func() error { return Save(path, Snapshot{Tabs: []Tab{{Path: huge}}}) }},
		{"title", func() error { return NewSnapshotBuilder("").AddDocument(doc, huge, 0) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			runtime.GC()
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			err := tc.run()
			runtime.ReadMemStats(&after)
			if err == nil {
				t.Fatal("resource rejection missing")
			}
			// Piece descriptor snapshots are ~40KiB for this 3MiB buffer.
			// Permit runtime noise, but reject any draft-length string/byte copy.
			if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 256<<10 {
				t.Fatalf("preflight copied large input: %d bytes, err=%v", allocated, err)
			}
		})
	}
}
