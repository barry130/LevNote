package session

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levnote.local/native/core/document"
	"levnote.local/native/core/text"
)

func TestPreparedSessionSaveUsesExactIndependentSnapshot(t *testing.T) {
	dir := t.TempDir()
	filename := filepath.Join(dir, "session.json")
	old := "old recovery"
	if err := Save(filename, Snapshot{Tabs: []Tab{{Draft: &old}}}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	value := "confirmed <&>🙂\r\n"
	snapshot := Snapshot{Version: Version, Active: 0, Tabs: []Tab{{
		Draft: &value, Encoding: document.UTF8, Selection: text.Range{Anchor: 1, Head: 4},
		OriginalPath: filepath.Join(dir, "original.md"), Title: "independent.txt", View: "source",
	}}}
	want, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := PrepareSave(filename, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := prepared.Abort(); err != nil {
			t.Error(err)
		}
	}()
	value = "caller mutated after Prepare"
	if got, err := os.ReadFile(filename); err != nil || !bytes.Equal(got, before) {
		t.Fatalf("Prepare published or changed the previous snapshot: %q %v", got, err)
	}
	if err := prepared.Commit(nil); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(filename); err != nil || !bytes.Equal(got, want) {
		t.Fatalf("prepared output differs from exact independent JSON oracle:\n got %q\nwant %q\n%v", got, want, err)
	}
	saved, issues := Load(filename)
	if len(issues) != 0 || len(saved.Tabs) != 1 {
		t.Fatalf("Load=%#v issues=%v", saved, issues)
	}
	restored, err := RestoreTab(saved.Tabs[0])
	if err != nil || restored.Document.Path != "" || !restored.Document.IsNew || restored.Document.Buffer.String() != "confirmed <&>🙂\r\n" || restored.Document.Buffer.Selection() != (text.Range{Anchor: 1, Head: 4}) {
		t.Fatalf("prepared snapshot did not really restore its independent draft: %#v %v", restored, err)
	}
}

func TestPreparedSessionBudgetFailureDoesNotCreateDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "must-not-exist")
	oversized := strings.Repeat("x", MaxDraftBytes+1)
	prepared, err := PrepareSave(filepath.Join(dir, "session.json"), Snapshot{Tabs: []Tab{{Draft: &oversized}}})
	if err == nil || prepared != nil {
		t.Fatal("oversized prepared session was accepted", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("failed preflight created a directory: %v", err)
	}
}
