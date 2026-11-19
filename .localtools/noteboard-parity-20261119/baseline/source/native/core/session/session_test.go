package session

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levnote.local/native/core/document"
)

func TestDirtyRecoveryCannotOverwriteChangedOriginal(t *testing.T) {
	dir := t.TempDir()
	original := filepath.Join(dir, "note.md")
	path := filepath.Join(dir, "state", "session.json")
	if err := os.WriteFile(original, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	doc, err := document.Open(original)
	if err != nil {
		t.Fatal(err)
	}
	_ = doc.Buffer.ReplaceUTF16(0, 8, "local draft🙂")
	_ = doc.Buffer.SetSelection(13, 13)
	tab, err := CaptureBounded(doc, "note.md", 120)
	if err != nil {
		t.Fatal(err)
	}
	if tab.Draft == nil || *tab.Draft != "local draft🙂" {
		t.Fatalf("dirty draft capture failed: %q", *tab.Draft)
	}
	s := Snapshot{Root: dir, Tabs: []Tab{tab}, Active: 0}
	if err = Save(path, s); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(original, []byte("external version"), 0600); err != nil {
		t.Fatal(err)
	}
	loaded, issues := Load(path)
	if len(issues) != 0 || len(loaded.Tabs) != 1 {
		t.Fatal(loaded, issues)
	}
	recovered, err := RestoreTab(loaded.Tabs[0])
	if err != nil {
		t.Fatal(err)
	}
	if recovered.OriginalPath != original || recovered.Document.Path != "" || !recovered.Document.IsNew || recovered.Document.Buffer.String() != "local draft🙂" {
		t.Fatal(recovered)
	}
	if err = recovered.Document.Save(); !errors.Is(err, document.ErrNoPath) {
		t.Fatal("draft borrowed original save token", err)
	}
	if err = recovered.Document.SaveAs(original); !errors.Is(err, document.ErrExists) {
		t.Fatal("draft clobbered original", err)
	}
	data, _ := os.ReadFile(original)
	if string(data) != "external version" {
		t.Fatal(string(data))
	}
}
func TestCleanTabsReopenAndFailedSavePreservesRecovery(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "note.txt")
	path := filepath.Join(dir, "session.json")
	_ = os.WriteFile(file, []byte("saved"), 0600)
	doc, err := document.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	tab, err := CaptureBounded(doc, "note", 0)
	if err != nil {
		t.Fatal(err)
	}
	s := Snapshot{Tabs: []Tab{tab}}
	if err = Save(path, s); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	large := strings.Repeat("x", MaxDraftBytes+1)
	s.Tabs[0].Draft = &large
	if err = Save(path, s); err == nil {
		t.Fatal("oversized draft accepted")
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("failed save destroyed recovery")
	}
	saved, issues := Load(path)
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	reopened, err := RestoreTab(saved.Tabs[0])
	if err != nil || reopened.Document.Dirty() {
		t.Fatal(reopened, err)
	}
}
func TestMalformedTabDoesNotDiscardOtherDrafts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	data := `{"version":1,"active":2,"tabs":[{"draft":"first"},{"path":"relative.txt"},{"draft":"third"}]}`
	_ = os.WriteFile(path, []byte(data), 0600)
	snapshot, issues := Load(path)
	if len(snapshot.Tabs) != 2 || snapshot.Active != 1 || len(issues) != 1 || issues[0].Reason != "invalid-tab" {
		t.Fatal(snapshot, issues)
	}
	if *snapshot.Tabs[1].Draft != "third" {
		t.Fatal(snapshot)
	}
}

func TestCaptureBoundedRejectsOversizedDraftBeforeCopying(t *testing.T) {
	doc := document.New("")
	_ = doc.Buffer.ReplaceUTF16(0, 0, strings.Repeat("x", MaxDraftBytes+1))
	tab, err := CaptureBounded(doc, "large", 0)
	if err == nil || tab.Draft != nil {
		t.Fatalf("oversized draft accepted: tab=%#v err=%v", tab, err)
	}
	if !strings.Contains(err.Error(), "recovery limit") {
		t.Fatal("error did not disclose the recovery limit", err)
	}
}

func TestCaptureBoundedSupportsUnsavedAndExistingTabs(t *testing.T) {
	for _, test := range []struct {
		name    string
		content string
		file    bool
	}{
		{name: "unsaved-empty", content: ""},
		{name: "unsaved-nonempty", content: "draft🙂"},
		{name: "existing-dirty", file: true, content: "dirty draft"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var doc *document.Document
			if test.file {
				path := filepath.Join(t.TempDir(), "note.md")
				if err := os.WriteFile(path, []byte("saved"), 0600); err != nil {
					t.Fatal(err)
				}
				var err error
				doc, err = document.Open(path)
				if err != nil {
					t.Fatal(err)
				}
			} else {
				doc = document.New(test.content)
			}
			if test.file {
				if err := doc.Buffer.ReplaceUTF16(0, doc.Buffer.LenUTF16(), test.content); err != nil {
					t.Fatal(err)
				}
			}
			if !doc.Dirty() {
				t.Fatalf("dirty test setup produced a clean document: %q", doc.Buffer.String())
			}
			tab, err := CaptureBounded(doc, test.name, 12)
			if err != nil {
				t.Fatal(err)
			}
			if tab.Draft == nil || *tab.Draft != test.content || tab.Scroll != 12 {
				t.Fatalf("dirty capture: %#v %q", tab, *tab.Draft)
			}
		})
	}
}

func TestPreferencesLineNumbersKeepsDefaultAndExplicitFalse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prefs.json")
	falseValue := false
	if err := Save(path, Snapshot{Preferences: Preferences{LineNumbers: &falseValue}}); err != nil {
		t.Fatal(err)
	}
	loaded, issues := Load(path)
	if len(issues) != 0 || loaded.Preferences.LineNumbers == nil || *loaded.Preferences.LineNumbers {
		t.Fatalf("explicit false lost: %#v issues=%v", loaded.Preferences.LineNumbers, issues)
	}
	data, _ := os.ReadFile(path)
	var raw struct {
		Preferences Preferences `json:"preferences"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if raw.Preferences.LineNumbers == nil || *raw.Preferences.LineNumbers {
		t.Fatal("explicit false was not serialized")
	}
}
func TestMissingCorruptAndFutureVersionsHaveDistinctOutcomes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	if _, issues := Load(path); len(issues) != 0 {
		t.Fatal(issues)
	}
	for _, test := range []struct{ data, reason string }{{"{", "corrupt"}, {`{"version":1} null`, "corrupt"}, {`{"version":2}`, "unsupported-version"}} {
		_ = os.WriteFile(path, []byte(test.data), 0600)
		_, issues := Load(path)
		if len(issues) != 1 || issues[0].Reason != test.reason {
			t.Fatal(issues)
		}
	}
	// A successful second save exercises replacement, not only first creation.
	value := ""
	s := Snapshot{Tabs: []Tab{{Draft: &value}}}
	if err := Save(path, s); err != nil {
		t.Fatal(err)
	}
	value = "updated"
	if err := Save(path, s); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	var decoded Snapshot
	if err := json.Unmarshal(raw, &decoded); err != nil || *decoded.Tabs[0].Draft != "updated" {
		t.Fatal(err)
	}
}

func TestSnapshotBuilderPreflightsAggregateBeforeMaterializingDrafts(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "session.json")
	builder := NewSnapshotBuilder(root)
	small := document.New("draft🙂")
	if err := builder.AddDocument(small, "small", 0); err != nil {
		t.Fatal(err)
	}
	if err := builder.SetActive(0); err != nil {
		t.Fatal(err)
	}
	if err := builder.Save(path); err != nil {
		t.Fatal(err)
	}

	draft := strings.Repeat(`x"<&>`, MaxDraftBytes/5)
	large := document.New(draft)
	err := builder.AddDocument(large, "large", 0)
	if err == nil || len(builder.snapshot.Tabs) != 1 {
		t.Fatalf("aggregate preflight failed to stop early: tabs=%d err=%v", len(builder.snapshot.Tabs), err)
	}
	if !strings.Contains(err.Error(), "existing recovery file kept") {
		t.Fatalf("error does not preserve recovery: %v", err)
	}
	loaded, issues := Load(path)
	if len(issues) != 0 || len(loaded.Tabs) != 1 || *loaded.Tabs[0].Draft != "draft🙂" {
		t.Fatalf("recovery changed: %#v issues=%v", loaded, issues)
	}
}

func TestSaveRejectsJSONEscapesBeforeMarshalling(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")
	if err := Save(path, Snapshot{}); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	// One 3 MiB raw draft is below the raw aggregate limit, but JSON encodes
	// each NUL as six bytes, proving the preflight accounts for escapes.
	value := strings.Repeat("\x00", 3<<20)
	snapshot := Snapshot{Tabs: []Tab{{Draft: &value}}}
	err := Save(path, snapshot)
	if err == nil {
		t.Fatal("JSON escaping and metadata budget was not preflighted")
	}
	if !strings.Contains(err.Error(), "before JSON marshalling") {
		t.Fatalf("error does not name preflight: %v", err)
	}
	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Fatal("failed preflight destroyed recovery")
	}
}
