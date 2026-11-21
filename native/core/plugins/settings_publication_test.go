package plugins

import (
	"bytes"
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

	"levnote.local/native/core/document"
)

// settingsPublicationFault exercises real private-state preparation and commit.
// Only the manager that owns this callback is affected. A postcommit cleanup
// refusal comes from Abort protecting a real foreign temp, after a successful
// replacement; the typed outcome is assembled as WritePrivateStateTo does.
func settingsPublicationFault(t *testing.T, guardErr error, cleanupFault bool, outcome *error) func(string, Settings) error {
	t.Helper()
	return func(root string, settings Settings) error {
		disabled := make([]string, 0, len(settings.Disabled))
		for id, value := range settings.Disabled {
			if value {
				disabled = append(disabled, id)
			}
		}
		sort.Strings(disabled)
		data, err := json.MarshalIndent(map[string][]string{"disabled": disabled}, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		path := SettingsPath(root)
		prepared, err := document.PreparePrivateStateTo(path, func(w io.Writer) error {
			_, err := w.Write(data)
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = prepared.Abort() })
		temps, err := filepath.Glob(filepath.Join(root, ".levnote-save-*"))
		if err != nil || len(temps) != 1 {
			t.Fatalf("expected one owned prepared temp: %v, %v", temps, err)
		}
		commitErr := prepared.Commit(func() error { return guardErr })
		if guardErr != nil {
			var publication *document.StatePublicationError
			if !errors.As(commitErr, &publication) || publication.Committed || !errors.Is(commitErr, guardErr) {
				t.Fatalf("real commit did not refuse before publication: %v", commitErr)
			}
			if err := prepared.Abort(); err != nil {
				t.Fatal(err)
			}
			*outcome = commitErr
			return fmt.Errorf("prepared settings: %w", commitErr)
		}
		if commitErr != nil {
			t.Fatalf("real replacement did not commit: %v", commitErr)
		}
		if !cleanupFault {
			t.Fatal("fault callback needs a guard or a cleanup fault")
		}
		// The seeded target forces replacement, consuming its exact owned temp.
		// A foreign file at that now-free name must never be removed by Abort.
		if _, err := os.Lstat(temps[0]); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("replacement did not consume its temp: %v", err)
		}
		file, err := os.OpenFile(temps[0], os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.WriteString("foreign temporary owner"); err != nil {
			_ = file.Close()
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
		cleanupErr := prepared.Abort()
		if cleanupErr == nil {
			t.Fatal("real Abort accepted a foreign temp identity")
		}
		if data, err := os.ReadFile(temps[0]); err != nil || string(data) != "foreign temporary owner" {
			t.Fatalf("Abort changed the foreign temp: %q, %v", data, err)
		}
		publication := &document.StatePublicationError{Path: path, Committed: true, CleanupErr: cleanupErr}
		*outcome = publication
		return fmt.Errorf("prepared settings: %w", publication)
	}
}

func readPublicationSettings(t *testing.T, root string) []byte {
	t.Helper()
	data, err := os.ReadFile(SettingsPath(root))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestSetEnabledSettingsPublicationOutcomes(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, committed := range []bool{false, true} {
			t.Run(fmt.Sprintf("enabled=%v/committed=%v", enabled, committed), func(t *testing.T) {
				root := t.TempDir()
				settingsRoot, installed := filepath.Join(root, "settings"), filepath.Join(root, "installed")
				initial := map[string]bool{"drawio": true}
				if enabled {
					initial["markdown"] = true
				}
				if err := SaveSettings(settingsRoot, Settings{Disabled: initial}); err != nil {
					t.Fatal(err)
				}
				manager, err := NewManager(settingsRoot, installed)
				if err != nil {
					t.Fatal(err)
				}
				before := readPublicationSettings(t, settingsRoot)
				manifests := manager.Manifests()
				guardErr := errors.New("stop this settings commit before publication")
				var outcome error
				var guard error
				if !committed {
					guard = guardErr
				}
				manager.saveSettings = settingsPublicationFault(t, guard, committed, &outcome)
				err = manager.SetEnabled("markdown", enabled)
				wrapped := fmt.Errorf("host: %w", err)
				var publication *document.StatePublicationError
				if !errors.As(wrapped, &publication) || publication.Committed != committed || !errors.Is(wrapped, outcome) {
					t.Fatalf("publication outcome/cause lost: %v", wrapped)
				}
				if errors.Is(wrapped, ErrEnabledNotLoaded) || (!committed && !errors.Is(wrapped, guardErr)) ||
					(committed && !errors.Is(wrapped, publication.CleanupErr)) {
					t.Fatalf("independent cause classification changed: %v", wrapped)
				}
				wantEnabled := enabled
				if !committed {
					wantEnabled = !enabled
					if !bytes.Equal(before, readPublicationSettings(t, settingsRoot)) || !reflect.DeepEqual(manager.Manifests(), manifests) {
						t.Fatal("precommit refusal changed disk bytes or live manifests")
					}
				} else if bytes.Equal(before, readPublicationSettings(t, settingsRoot)) || !strings.Contains(err.Error(), "was published") {
					t.Fatalf("postcommit warning did not describe actual new disk state: %v", err)
				}
				loaded, loadErr := LoadSettings(settingsRoot)
				if loadErr != nil || loaded.Disabled["markdown"] == wantEnabled || !loaded.Disabled["drawio"] ||
					manager.Enabled("markdown") != wantEnabled || manager.disabled["markdown"] == wantEnabled || manager.Enabled("drawio") {
					t.Fatalf("disk/disabled/registry disagree: disk=%v enabled=%v disabled=%v err=%v", loaded.Disabled, manager.Enabled("markdown"), manager.disabled, loadErr)
				}
				fresh, err := NewManager(settingsRoot, installed)
				if err != nil {
					t.Fatal(err)
				}
				wantMatch := "code"
				if wantEnabled {
					wantMatch = "markdown"
				}
				for _, candidate := range []*Manager{manager, fresh} {
					match, err := candidate.Match("example.md")
					if err != nil || match.ID != wantMatch {
						t.Fatalf("live/restarted matching disagrees with actual publication: %+v, %v", match, err)
					}
				}
				// Another live object does real persistence while this fault is armed.
				otherRoot := t.TempDir()
				other, err := NewManager(filepath.Join(otherRoot, "settings"), filepath.Join(otherRoot, "installed"))
				if err != nil {
					t.Fatal(err)
				}
				if err := other.SetEnabled("markdown", false); err != nil || other.Enabled("markdown") {
					t.Fatalf("instance fault leaked to another manager: %v", err)
				}
			})
		}
	}
}

func TestSetEnabledSettingsPublicationAndLoadFailuresCompose(t *testing.T) {
	for _, broken := range []string{"removed", "corrupt"} {
		for _, phase := range []string{"precommit", "committed-cleanup", "clean"} {
			t.Run(broken+"/"+phase, func(t *testing.T) {
				root := t.TempDir()
				settingsRoot, installed := filepath.Join(root, "settings"), filepath.Join(root, "installed")
				if err := SaveSettings(settingsRoot, Settings{Disabled: map[string]bool{"publication-source": true}}); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(installed, 0700); err != nil {
					t.Fatal(err)
				}
				manifestPath := filepath.Join(installed, "publication-source.json")
				if err := os.WriteFile(manifestPath, []byte(`{"schema":1,"id":"publication-source","name":"Publication Source","extensions":[".go"]}`), 0600); err != nil {
					t.Fatal(err)
				}
				manager, err := NewManager(settingsRoot, installed)
				if err != nil {
					t.Fatal(err)
				}
				before := readPublicationSettings(t, settingsRoot)
				manifests := manager.Manifests()
				if _, ok := manager.Syntax("publication-source"); !ok {
					t.Fatal("fixture did not load the disabled syntax")
				}
				if broken == "removed" {
					if err := os.Remove(manifestPath); err != nil {
						t.Fatal(err)
					}
				} else if err := os.WriteFile(manifestPath, []byte("{"), 0600); err != nil {
					t.Fatal(err)
				}
				guard := errors.New("do not publish settings")
				var outcome error
				if phase == "precommit" {
					manager.saveSettings = settingsPublicationFault(t, guard, false, &outcome)
				} else if phase == "committed-cleanup" {
					manager.saveSettings = settingsPublicationFault(t, nil, true, &outcome)
				}
				err = manager.SetEnabled("publication-source", true)
				wrapped := fmt.Errorf("host: %w", err)
				var publication *document.StatePublicationError
				if errors.As(wrapped, &publication) != (phase != "clean") || errors.Is(wrapped, ErrEnabledNotLoaded) != (phase != "precommit") {
					t.Fatalf("publication and enabled-not-loaded did not compose independently: %v", wrapped)
				}
				if outcome != nil && !errors.Is(wrapped, outcome) {
					t.Fatalf("original publication error lost: %v", wrapped)
				}
				if phase == "precommit" {
					if !errors.Is(wrapped, guard) || !bytes.Equal(before, readPublicationSettings(t, settingsRoot)) || !reflect.DeepEqual(manager.Manifests(), manifests) {
						t.Fatal("precommit failure reloaded or changed settings")
					}
					if _, ok := manager.Syntax("publication-source"); !ok {
						t.Fatal("precommit failure discarded the live syntax")
					}
				} else {
					settings, loadErr := LoadSettings(settingsRoot)
					if loadErr != nil || settings.Disabled["publication-source"] || manager.disabled["publication-source"] || manager.Enabled("publication-source") {
						t.Fatalf("published enable/load failure state disagrees: %+v %v", settings, loadErr)
					}
					if _, ok := manager.Syntax("publication-source"); ok {
						t.Fatal("published reload kept removed/corrupt syntax")
					}
					if phase == "committed-cleanup" && (!publication.Committed || !errors.Is(wrapped, publication.CleanupErr)) {
						t.Fatalf("postcommit cleanup cause lost on failed enable: %v", wrapped)
					}
				}
				if match, err := manager.Match("example.go"); err != nil || match.ID != "code" {
					t.Fatalf("fallback no longer matches the actual registry state: %+v %v", match, err)
				}
			})
		}
	}
}
