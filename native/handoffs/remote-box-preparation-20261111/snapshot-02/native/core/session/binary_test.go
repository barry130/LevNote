package session

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levnote.local/native/core/document"
	"levnote.local/native/core/text"
)

func binaryTabFixture(t *testing.T) (Tab, string) {
	t.Helper()
	dir := t.TempDir()
	record := BinaryTab{
		Path:     filepath.Join(dir, "photo.png"),
		Kind:     document.BinaryKindPNG,
		Title:    "photo <&>",
		PluginID: "images",
		View:     "preview",
	}
	tab, err := newBinaryTab(record)
	if err != nil {
		t.Fatal(err)
	}
	return tab, dir
}

func TestBinarySessionCaptureSaveLoadAndRestore(t *testing.T) {
	binary, dir := binaryTabFixture(t)
	draft := "text draft🙂"
	text := Tab{
		Path:      filepath.Join(dir, "note.md"),
		Title:     "note",
		Encoding:  document.UTF8,
		BOM:       true,
		Draft:     &draft,
		Selection: text.Range{Anchor: 2, Head: 4},
		Scroll:    12,
	}
	builder := NewSnapshotBuilder(dir)
	if err := builder.AddTab(text); err != nil {
		t.Fatal(err)
	}
	if err := builder.AddBinaryTab(*binary.binary); err != nil {
		t.Fatal(err)
	}
	if err := builder.SetActive(1); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "session.json")
	if err := builder.Save(path); err != nil {
		t.Fatal(err)
	}

	want := mixedSessionWireOracle{
		Version: Version, Root: dir, Active: 1,
		Tabs: []any{text, mixedBinaryWireOracle{
			Record:   "binary",
			Kind:     document.BinaryKindPNG,
			Path:     binary.Path,
			Title:    "photo <&>",
			PluginID: "images",
			View:     "preview",
		}},
	}
	wantJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := preflightSnapshot(mustBuilderSnapshot(t, builder))
	if err != nil || expected != len(wantJSON) {
		t.Fatalf("preflight=%d oracle=%d err=%v", expected, len(wantJSON), err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != string(wantJSON) {
		t.Fatalf("mixed output differs\n got: %s\nwant: %s", got, wantJSON)
	}

	loaded, issues := Load(path)
	if len(issues) != 0 || len(loaded.Tabs) != 2 || loaded.Active != 1 {
		t.Fatalf("Load=%#v issues=%v", loaded, issues)
	}
	if *loaded.Tabs[0].Draft != draft || loaded.Tabs[0].Scroll != 12 {
		t.Fatalf("text tab changed: %#v", loaded.Tabs[0])
	}
	restored, err := RestoreBinary(loaded.Tabs[1])
	if err != nil || restored != (RestoredBinary(*binary.binary)) {
		t.Fatalf("RestoreBinary=%#v err=%v", restored, err)
	}
	if _, err := RestoreTab(loaded.Tabs[1]); !errors.Is(err, ErrBinaryDispatchRequired) {
		t.Fatalf("text restore accepted binary: %v", err)
	}
	if _, err := RestoreBinary(loaded.Tabs[0]); !errors.Is(err, ErrTextDispatchRequired) {
		t.Fatalf("binary restore accepted text: %v", err)
	}

	// RestoreTab remains an explicit text API. Its existing result must not
	// change for ordinary records now that binary authority exists.
	textRestored, err := RestoreTab(loaded.Tabs[0])
	if err != nil || textRestored.Title != "note" || textRestored.Scroll != 12 {
		t.Fatalf("ordinary text restore changed: %#v err=%v", textRestored, err)
	}
}

type mixedSessionWireOracle struct {
	Version     int         `json:"version"`
	Root        string      `json:"root,omitempty"`
	Active      int         `json:"active"`
	Tabs        []any       `json:"tabs"`
	Preferences Preferences `json:"preferences"`
}

type mixedBinaryWireOracle struct {
	Record   string              `json:"record"`
	Kind     document.BinaryKind `json:"kind"`
	Path     string              `json:"path"`
	Title    string              `json:"title,omitempty"`
	PluginID string              `json:"pluginId,omitempty"`
	View     string              `json:"view,omitempty"`
}

func mustBuilderSnapshot(t *testing.T, builder *SnapshotBuilder) Snapshot {
	t.Helper()
	snapshot, err := builder.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func TestBinaryRecordStrictWireContract(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.json")
	base := binaryWireFixture(t, "record", "binary")
	for _, test := range []struct {
		name string
		data string
	}{
		{"unknown-kind", strings.Replace(base, `"kind":"png"`, `"kind":"gif"`, 1)},
		{"missing-kind", strings.Replace(base, `"kind":"png",`, ``, 1)},
		{"unknown-record", strings.Replace(base, `"record":"binary"`, `"record":"text"`, 1)},
		{"null-kind", strings.Replace(base, `"kind":"png"`, `"kind":null`, 1)},
		{"empty-path", strings.Replace(base, `"path":"C:\\a.png"`, `"path":""`, 1)},
		{"relative-path", strings.Replace(base, `"path":"C:\\a.png"`, `"path":"a.png"`, 1)},
		{"unknown-field", binaryWireFixture(t, "pixels", nil)},
		{"draft-empty", binaryWireFixture(t, "draft", "")},
		{"encoding-null", binaryWireFixture(t, "encoding", nil)},
		{"bom-false", binaryWireFixture(t, "bom", false)},
		{"originalPath-empty", binaryWireFixture(t, "originalPath", "")},
		{"selection-zero", binaryWireFixture(t, "selection", text.Range{})},
		{"unknown-nested-array", binaryWireFixture(t, "unknown", map[string]any{
			"nested": []any{1, 2, map[string]any{"x": "text"}, map[string]any{"y": true}},
		})},
		{"last-primitive-unknown", binaryWireFixture(t, "unknown", nil)},
		{"last-string-unknown", binaryWireFixture(t, "unknown", "last")},
		{"scroll-zero", binaryWireFixture(t, "scroll", float32(0))},
		{"wrong-discriminator-type", strings.Replace(base, `"record":"binary"`, `"record":null`, 1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			if !json.Valid([]byte(test.data)) {
				t.Fatal("invalid fixture")
			}
			if err := os.WriteFile(path, []byte(test.data), 0600); err != nil {
				t.Fatal(err)
			}
			snapshot, issues := Load(path)
			if len(snapshot.Tabs) != 0 || len(issues) != 1 || issues[0].Reason != "invalid-tab" {
				t.Fatalf("invalid record accepted: snapshot=%#v issues=%v", snapshot, issues)
			}
		})
	}
}

func TestDamagedBinaryDoesNotDiscardNeighborDrafts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")
	data := `{"version":1,"active":2,"tabs":[{"draft":"first"},{"record":"binary","kind":"gif","path":"C:\\bad.png"},{"draft":"third"}]}`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	snapshot, issues := Load(path)
	if len(snapshot.Tabs) != 2 || snapshot.Active != 1 || len(issues) != 1 || issues[0].Reason != "invalid-tab" {
		t.Fatalf("neighbor recovery failed: snapshot=%#v issues=%v", snapshot, issues)
	}
	if *snapshot.Tabs[0].Draft != "first" || *snapshot.Tabs[1].Draft != "third" {
		t.Fatalf("drafts shifted: %#v", snapshot.Tabs)
	}
}

func TestWireScannerHandlesEscapedAndNestedTextFields(t *testing.T) {
	escaped, err := objectFields([]byte(`{"record":"binary","kind":"png","path":"C:\\a.png"}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(escaped) != 3 || string(escaped[binaryRecordKey].value) != `"binary"` {
		t.Fatalf("ordinary scanner fields=%#v", escaped)
	}
	escaped, err = objectFields([]byte(`{"rec\u006frd":"binary","kind":"png","path":"C:\\a.png"}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(escaped) != 3 || string(escaped[binaryRecordKey].value) != `"binary"` {
		t.Fatalf("escaped key fields=%#v", escaped)
	}

	value := "draft with }, ] and \" quotes"
	tabJSON, err := json.Marshal(Tab{
		Draft:     &value,
		Selection: text.Range{Anchor: 2, Head: 4},
	})
	if err != nil {
		t.Fatal(err)
	}
	fields, err := objectFields(tabJSON)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) == 0 || &fields["draft"].value[0] != &tabJSON[indexOfJSONField(t, tabJSON, "draft")] {
		t.Fatal("draft value slice is not an alias of entry input")
	}
	var decoded Tab
	if err := json.Unmarshal(tabJSON, &decoded); err != nil || *decoded.Draft != value ||
		decoded.Selection != (text.Range{Anchor: 2, Head: 4}) {
		t.Fatalf("nested text decode failed: %#v err=%v", decoded, err)
	}
}

func binaryWireFixture(t *testing.T, key string, value any) string {
	t.Helper()
	entry := map[string]any{
		"record": "binary",
		"kind":   "png",
		"path":   `C:\a.png`,
	}
	entry[key] = value
	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatal(err)
	}
	return `{"version":1,"active":0,"tabs":[` + string(data) + `],"preferences":{}}`
}

func indexOfJSONField(t *testing.T, data []byte, field string) int {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(data))
	if token, err := decoder.Token(); err != nil || token != json.Delim('{') {
		t.Fatal("tab fixture is not an object")
	}
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			t.Fatal(err)
		}
		name, _ := token.(string)
		if name == field {
			var value json.RawMessage
			if err := decoder.Decode(&value); err != nil {
				t.Fatal(err)
			}
			return indexBytes(data, value)
		}
		var skipped any
		if err := decoder.Decode(&skipped); err != nil {
			t.Fatal(err)
		}
	}
	t.Fatalf("field %q not found", field)
	return 0
}

func indexBytes(data, value []byte) int {
	for i := range data {
		if len(data[i:]) >= len(value) && bytes.Equal(data[i:i+len(value)], value) {
			return i
		}
	}
	return -1
}

func TestBinaryBuilderAndSnapshotAreIndependent(t *testing.T) {
	builder := NewSnapshotBuilder("")
	record := BinaryTab{Path: `C:\photo.png`, Kind: document.BinaryKindJPEG}
	if err := builder.AddBinaryTab(record); err != nil {
		t.Fatal(err)
	}
	first, err := builder.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	first.Tabs[0].binary.Path = `C:\changed.png`
	first.Tabs[0].Path = `C:\changed.png`
	second, err := builder.Snapshot()
	if err != nil || second.Tabs[0].Path != `C:\photo.png` || second.Tabs[0].binary.Path != `C:\photo.png` {
		t.Fatalf("binary clone leaked: %#v err=%v", second, err)
	}
	before := builder.Len()
	invalid := record
	invalid.Path = "relative.png"
	if err := builder.AddBinaryTab(invalid); err == nil || builder.Len() != before {
		t.Fatalf("invalid binary changed builder: len=%d err=%v", builder.Len(), err)
	}
}

func TestBinaryRecordLimitsAndTextByteCompatibility(t *testing.T) {
	dir := t.TempDir()
	textPath := filepath.Join(dir, "text-session.json")
	draft := "unchanged text draft"
	textSnapshot := Snapshot{Version: Version, Tabs: []Tab{{Draft: &draft, Scroll: 12}}}
	if err := Save(textPath, textSnapshot); err != nil {
		t.Fatal(err)
	}
	want, err := json.Marshal(textSnapshot)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(textPath)
	if string(got) != string(want) {
		t.Fatalf("text bytes changed\n got: %s\nwant: %s", got, want)
	}

	binaryPath := filepath.Join(dir, "binary-session.json")
	builder := NewSnapshotBuilder("")
	for range MaxTabs {
		if err := builder.AddBinaryTab(BinaryTab{Path: `C:\photo.png`, Kind: document.BinaryKindPNG}); err != nil {
			t.Fatal(err)
		}
	}
	if err := builder.Save(binaryPath); err != nil {
		t.Fatal(err)
	}
	if err := builder.AddBinaryTab(BinaryTab{Path: `C:\extra.png`, Kind: document.BinaryKindPNG}); err == nil {
		t.Fatal("MaxTabs boundary bypassed")
	}
	loaded, issues := Load(binaryPath)
	if len(issues) != 0 || len(loaded.Tabs) != MaxTabs {
		t.Fatalf("MaxTabs load failed: tabs=%d issues=%v", len(loaded.Tabs), issues)
	}

	// A metadata-only value crosses the exact remaining JSON budget. Rejection
	// happens before the private-state destination directory is created.
	bounded := NewSnapshotBuilder("")
	if err := bounded.AddBinaryTab(BinaryTab{Path: `C:\photo.png`, Kind: document.BinaryKindPNG}); err != nil {
		t.Fatal(err)
	}
	snapshot, err := bounded.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	size, err := preflightSnapshot(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	record := BinaryTab{
		Path:  `C:\over.png`,
		Kind:  document.BinaryKindPNG,
		Title: strings.Repeat("a", MaxBytes-size+1),
	}
	rejectPath := filepath.Join(dir, "does-not-exist", "session.json")
	if err := Save(rejectPath, Snapshot{Tabs: []Tab{mustBinaryTab(t, record)}}); err == nil {
		t.Fatal("MaxBytes boundary bypassed")
	}
	if _, err := os.Stat(filepath.Dir(rejectPath)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("rejected save created directory: %v", err)
	}
}

func mustBinaryTab(t *testing.T, record BinaryTab) Tab {
	t.Helper()
	tab, err := newBinaryTab(record)
	if err != nil {
		t.Fatal(err)
	}
	return tab
}
