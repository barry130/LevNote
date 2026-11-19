package app

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/session"
)

// Reload must supersede a previously captured draft even when the user has
// undone back to the saved version and no later edit or normal close occurs.
func TestReloadRecoveryPublicReloadSupersedesSavedDraft(t *testing.T) {
	root := t.TempDir()
	path, config := filepath.Join(root, "reload.txt"), filepath.Join(root, "config")
	original := append([]byte{0xef, 0xbb, 0xbf}, []byte("original disk\r\n")...)
	external := append([]byte{0xef, 0xbb, 0xbf}, []byte("external disk\r\n")...)
	if err := os.WriteFile(path, original, 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, config)
	clock := installAutosaveClock(a)
	a.Open(path)
	tab := a.Current()
	if tab == nil {
		t.Fatalf("open failed: %s", a.Message)
	}
	tab.Editor.ReplaceText(&ui.TextInputRange{Start: 0, End: tab.Doc.Buffer.LenUTF16()}, "old recovered draft\r\n")
	clock.now = clock.now.Add(autosaveDelay)
	clock.latest(t).fn()
	clock.drain()
	before := readRecovery(t, a)
	if len(before.Tabs) != 1 || before.Tabs[0].Draft == nil || *before.Tabs[0].Draft != "old recovered draft\r\n" {
		t.Fatal("fixture did not persist the original dirty draft")
	}
	tab.Editor.Command("undo")
	if tab.Doc.Dirty() {
		t.Fatal("undo did not return to the saved version")
	}
	if err := os.WriteFile(path, external, 0600); err != nil {
		t.Fatal(err)
	}
	a.Reload()
	if tab.Doc.Buffer.String() != "external disk\r\n" || tab.Doc.Dirty() {
		t.Fatalf("reload did not adopt external content: %s", a.Message)
	}
	// Restore directly, without CanClose/Persist, models recovery after an
	// abrupt exit. Normal closing would hide the stale-snapshot defect.
	restored := newTestApp(t, config)
	restored.Restore()
	got := restored.Current()
	if got == nil || got.Doc.IsNew || got.Doc.Dirty() || got.Doc.Buffer.String() != "external disk\r\n" {
		if got == nil {
			t.Fatalf("reload recovery restored no document: %s", restored.Message)
		}
		t.Fatalf("reload recovery resurrected old draft: new=%v dirty=%v text=%q", got.Doc.IsNew, got.Doc.Dirty(), got.Doc.Buffer.String())
	}
	actual, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(actual, external) {
		t.Fatalf("reload changed external file bytes: %x, %v", actual, err)
	}
}

func reloadRecoveryConflict(t *testing.T) (*App, *Tab, *autosaveClock, string, []byte) {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "conflict.txt")
	if err := os.WriteFile(path, []byte("original disk\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	clock := installAutosaveClock(a)
	a.Open(path)
	tab := a.Current()
	if tab == nil {
		t.Fatalf("open failed: %s", a.Message)
	}
	tab.Editor.ReplaceText(&ui.TextInputRange{Start: 0, End: tab.Doc.Buffer.LenUTF16()}, "discarded draft\r\n")
	clock.now = clock.now.Add(autosaveDelay)
	clock.latest(t).fn()
	clock.drain()
	before := readRecovery(t, a)
	if len(before.Tabs) != 1 || before.Tabs[0].Draft == nil || *before.Tabs[0].Draft != "discarded draft\r\n" {
		t.Fatal("fixture did not capture the dirty draft")
	}
	external := []byte("external disk is now different\r\n")
	if err := os.WriteFile(path, external, 0600); err != nil {
		t.Fatal(err)
	}
	// Keep the real disk worker; only dispatch its completion on this test's UI thread.
	dispatch := make(chan func(), 16)
	a.uiDispatch = func(fn func()) { dispatch <- fn }
	a.CheckExternal()
	select {
	case fn := <-dispatch:
		fn()
	case <-time.After(time.Minute):
		t.Fatal("external file check did not complete")
	}
	if !tab.Conflict {
		t.Fatalf("external change did not reach the conflict bar: %s", a.Message)
	}
	return a, tab, clock, path, external
}

func clickReloadRecovery(t *testing.T, a *App) {
	t.Helper()
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	if err := tester.Click("重新载入"); err != nil {
		t.Fatal(err)
	}
}

func TestReloadRecoveryConflictDiscardSupersedesSavedDraft(t *testing.T) {
	a, tab, _, path, external := reloadRecoveryConflict(t)
	confirmed := false
	a.reloadDialog = func(options mygo.MessageOptions) (mygo.MessageResult, error) {
		if options.Title != "重新载入" || len(options.Buttons) != 2 || options.Buttons[1] != "放弃并重新载入" || options.CancelButton != 0 {
			t.Fatalf("unexpected discard prompt: %#v", options)
		}
		confirmed = true
		return mygo.MessageResult{Button: 1}, nil
	}
	clickReloadRecovery(t, a)
	if !confirmed || tab.Conflict || tab.Doc.Dirty() || tab.Doc.Buffer.String() != string(external) {
		t.Fatalf("confirmed reload failed: confirmed=%v conflict=%v dirty=%v text=%q message=%q", confirmed, tab.Conflict, tab.Doc.Dirty(), tab.Doc.Buffer.String(), a.Message)
	}
	restored := newTestApp(t, a.configDir)
	restored.Restore()
	got := restored.Current()
	if got == nil {
		t.Fatalf("reload recovery restored no document: %s", restored.Message)
	}
	if got.Doc.IsNew || got.Doc.Dirty() || got.Doc.Buffer.String() != string(external) {
		t.Fatalf("confirmed discard resurrected old draft: new=%v dirty=%v text=%q", got.Doc.IsNew, got.Doc.Dirty(), got.Doc.Buffer.String())
	}
	actual, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(actual, external) {
		t.Fatalf("discard reload changed external bytes: %x, %v", actual, err)
	}
}

func TestReloadRecoveryConfirmationOwnerChangePreservesDraft(t *testing.T) {
	a, tab, _, _, _ := reloadRecoveryConflict(t)
	var next *Tab
	a.reloadDialog = func(mygo.MessageOptions) (mygo.MessageResult, error) {
		// Native message boxes may pump nested UI callbacks. A completed open
		// can activate a different tab before the original confirmation returns.
		a.NewDocument("text")
		next = a.Current()
		next.Editor.ReplaceText(nil, "new active draft")
		return mygo.MessageResult{Button: 1}, nil
	}
	clickReloadRecovery(t, a)
	if next == nil || a.Current() != next || next.Doc.Buffer.String() != "new active draft" {
		t.Fatal("confirmation changed the replacement owner")
	}
	if tab.Doc.Buffer.String() != "discarded draft\r\n" || !tab.Doc.Dirty() || !tab.Conflict {
		t.Fatalf("stale confirmation discarded the previous owner's draft: dirty=%v text=%q", tab.Doc.Dirty(), tab.Doc.Buffer.String())
	}
	restored := newTestApp(t, a.configDir)
	restored.Restore()
	got := restored.Current()
	if got == nil || got.Doc.Buffer.String() != "discarded draft\r\n" || !got.Doc.Dirty() {
		t.Fatal("stale confirmation overwrote the existing recovery draft")
	}
}

func TestReloadRecoveryCancelledOrFailedPreservesDraft(t *testing.T) {
	for _, scenario := range []string{"cancel", "dialog error", "disk read error", "public dirty"} {
		t.Run(scenario, func(t *testing.T) {
			a, tab, _, path, external := reloadRecoveryConflict(t)
			before, err := os.ReadFile(filepath.Join(a.configDir, "session.json"))
			if err != nil {
				t.Fatal(err)
			}
			a.reloadDialog = func(mygo.MessageOptions) (mygo.MessageResult, error) {
				switch scenario {
				case "cancel":
					return mygo.MessageResult{Button: 0}, nil
				case "dialog error":
					return mygo.MessageResult{}, errors.New("reload dialog fixture failure")
				case "disk read error":
					if err := os.Remove(path); err != nil {
						t.Fatal(err)
					}
					return mygo.MessageResult{Button: 1}, nil
				default:
					t.Fatal("public Reload unexpectedly asked to discard dirty text")
					return mygo.MessageResult{}, nil
				}
			}
			if scenario == "public dirty" {
				a.Reload()
			} else {
				clickReloadRecovery(t, a)
			}
			if tab.Doc.Buffer.String() != "discarded draft\r\n" || !tab.Doc.Dirty() || !tab.Conflict {
				t.Fatalf("cancelled/failed reload changed draft: dirty=%v conflict=%v text=%q", tab.Doc.Dirty(), tab.Conflict, tab.Doc.Buffer.String())
			}
			after, err := os.ReadFile(filepath.Join(a.configDir, "session.json"))
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("cancelled/failed reload changed recovery bytes: %v", err)
			}
			if strings.Contains(a.Message, "已重新载入") {
				t.Fatalf("cancelled/failed reload claimed success: %s", a.Message)
			}
			if scenario != "disk read error" {
				actual, err := os.ReadFile(path)
				if err != nil || !bytes.Equal(actual, external) {
					t.Fatalf("cancelled/failed reload changed external bytes: %x, %v", actual, err)
				}
			}
			restored := newTestApp(t, a.configDir)
			restored.Restore()
			got := restored.Current()
			if got == nil || !got.Doc.Dirty() || got.Doc.Buffer.String() != "discarded draft\r\n" {
				t.Fatal("cancelled/failed reload lost the recovery draft")
			}
		})
	}
}

func TestReloadRecoveryConfirmationEditPreservesNewDraft(t *testing.T) {
	a, tab, _, path, external := reloadRecoveryConflict(t)
	a.reloadDialog = func(mygo.MessageOptions) (mygo.MessageResult, error) {
		tab.Editor.ReplaceText(&ui.TextInputRange{Start: 0, End: tab.Doc.Buffer.LenUTF16()}, "edited while confirming")
		return mygo.MessageResult{Button: 1}, nil
	}
	clickReloadRecovery(t, a)
	if tab.Doc.Buffer.String() != "edited while confirming" || !tab.Doc.Dirty() || !tab.Conflict {
		t.Fatalf("stale confirmation discarded a newer edit: dirty=%v text=%q", tab.Doc.Dirty(), tab.Doc.Buffer.String())
	}
	if !strings.Contains(a.Message, "已变化") {
		t.Fatalf("stale confirmation was not disclosed: %s", a.Message)
	}
	actual, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(actual, external) {
		t.Fatalf("stale confirmation changed external file: %x, %v", actual, err)
	}
}

func TestReloadRecoveryPersistenceFailureKeepsSnapshotAndReportsAdoptedDisk(t *testing.T) {
	for _, entry := range []string{"public", "conflict bar"} {
		t.Run(entry, func(t *testing.T) {
			a, tab, clock, path, external := reloadRecoveryConflict(t)
			before, err := os.ReadFile(filepath.Join(a.configDir, "session.json"))
			if err != nil {
				t.Fatal(err)
			}
			if entry == "public" {
				tab.Editor.Command("undo")
			}
			// A real unrelated draft exceeding the recovery budget makes the
			// whole snapshot fail; no persistence implementation is replaced.
			a.NewDocument("text")
			other := a.Current()
			other.Editor.ReplaceText(nil, strings.Repeat("x", session.MaxDraftBytes+1))
			if !a.Select(0) {
				t.Fatal("could not select the reload target")
			}
			a.reloadDialog = func(mygo.MessageOptions) (mygo.MessageResult, error) {
				return mygo.MessageResult{Button: 1}, nil
			}
			if entry == "public" {
				a.Reload()
			} else {
				clickReloadRecovery(t, a)
			}
			if tab.Doc.Dirty() || tab.Doc.Buffer.String() != string(external) {
				t.Fatal("persistence failure rolled back the adopted disk version")
			}
			if !strings.Contains(a.Message, "已采用磁盘版本") || !strings.Contains(a.Message, "旧恢复快照仍保留") {
				t.Fatalf("partial success was not explained: %s", a.Message)
			}
			issues := len(a.Issues)
			a.Message = "intervening status message"
			a.Reload()
			if len(a.Issues) != issues || !strings.Contains(a.Message, "旧恢复快照仍保留") {
				t.Fatalf("repeated failure duplicated issues or hid the current error: issues=%d/%d message=%q", len(a.Issues), issues, a.Message)
			}
			// Deliver even the old stopped callback: an unchanged failed state
			// must neither retry automatically nor replace the retained snapshot.
			clock.now = clock.now.Add(3 * autosaveDelay)
			clock.latest(t).fn()
			clock.drain()
			if len(a.Issues) != issues {
				t.Fatal("unchanged recovery failure retried automatically")
			}
			after, err := os.ReadFile(filepath.Join(a.configDir, "session.json"))
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("failed persistence replaced the old recovery file: %v", err)
			}
			actual, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(actual, external) {
				t.Fatalf("reload persistence failure changed external bytes: %x, %v", actual, err)
			}
			restored := newTestApp(t, a.configDir)
			restored.Restore()
			got := restored.Current()
			if got == nil || got.Doc.Buffer.String() != "discarded draft\r\n" || !got.Doc.Dirty() {
				t.Fatal("failed snapshot did not retain the previously recoverable draft")
			}
			other.Editor.ReplaceText(&ui.TextInputRange{Start: 0, End: other.Doc.Buffer.LenUTF16()}, "retained other draft")
			a.Reload()
			retry := newTestApp(t, a.configDir)
			retry.Restore()
			if len(retry.Tabs) != 2 || retry.Tabs[0].Doc.Dirty() || retry.Tabs[0].Doc.Buffer.String() != string(external) || retry.Tabs[1].Doc.Buffer.String() != "retained other draft" {
				t.Fatalf("explicit retry did not capture the repaired complete session: tabs=%d message=%s", len(retry.Tabs), a.Message)
			}
		})
	}
}
