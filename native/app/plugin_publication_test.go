package app

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levnote.local/native/core/document"
	"levnote.local/native/core/plugins"
)

// pluginPublicationOutcome returns an actual prepared commit refusal, or a
// cleanup warning after a real commit of the manager's published bytes. The
// manager transition itself is covered by core/plugins' instance-local tests;
// this package verifies the UI result boundary without a public injection API.
func pluginPublicationOutcome(t *testing.T, path string, committed bool) error {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := document.PreparePrivateStateTo(path, func(w io.Writer) error {
		_, err := w.Write(data)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = prepared.Abort() })
	temps, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".levnote-save-*"))
	if err != nil || len(temps) != 1 {
		t.Fatalf("expected one exact prepared temp: %v, %v", temps, err)
	}
	refused := errors.New("plugin commit refused before publication")
	commitErr := prepared.Commit(func() error {
		if !committed {
			return refused
		}
		return nil
	})
	if !committed {
		var publication *document.StatePublicationError
		if !errors.As(commitErr, &publication) || publication.Committed || !errors.Is(commitErr, refused) {
			t.Fatalf("real prepared commit was not refused: %v", commitErr)
		}
		if err := prepared.Abort(); err != nil {
			t.Fatal(err)
		}
		return commitErr
	}
	if commitErr != nil {
		t.Fatalf("real settings replacement did not commit: %v", commitErr)
	}
	if _, err := os.Lstat(temps[0]); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("real replacement did not consume the temp: %v", err)
	}
	file, err := os.OpenFile(temps[0], os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteString("foreign plugin temp"); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	cleanupErr := prepared.Abort()
	if cleanupErr == nil {
		t.Fatal("real cleanup did not reject a foreign temp identity")
	}
	if foreign, err := os.ReadFile(temps[0]); err != nil || string(foreign) != "foreign plugin temp" {
		t.Fatalf("Abort changed the foreign temp: %q %v", foreign, err)
	}
	return &document.StatePublicationError{Path: path, Committed: true, CleanupErr: cleanupErr}
}

func TestPluginPublicationResultReconfiguresPublishedTabs(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("enabled=%v", enabled), func(t *testing.T) {
			a := newTestApp(t, t.TempDir())
			a.NewDocument("markdown")
			a.NewDocument("markdown")
			if len(a.Tabs) != 2 {
				t.Fatal("expected two real markdown tabs")
			}
			if enabled {
				a.setPluginEnabled("markdown", false)
			}
			oldViews := make([]*failingView, len(a.Tabs))
			verify := make([]func(), len(a.Tabs))
			for index, tab := range a.Tabs {
				view := &failingView{}
				tab.View = view
				oldViews[index] = view
				if err := tab.Doc.Buffer.SetSelection(1, 1); err != nil {
					t.Fatal(err)
				}
				before := tab.Doc.Buffer.Snapshot()
				version, history, selection := tab.Doc.Buffer.Version(), tab.Doc.Buffer.History(), tab.Doc.Buffer.Selection()
				generation := tab.OutlineFactoryGeneration
				verify[index] = func() {
					wantPlugin := "code"
					if enabled {
						wantPlugin = "markdown"
					}
					if tab.View != nil || view.releases != 1 || tab.OutlineFactoryGeneration <= generation || tab.MatchedPlugin.ID != wantPlugin {
						t.Fatalf("published result kept an old tab route: plugin=%q generation=%d/%d releases=%d message=%q",
							tab.MatchedPlugin.ID, tab.OutlineFactoryGeneration, generation, view.releases, a.Message)
					}
					if !tab.Doc.Buffer.EqualSnapshot(before) || tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.History() != history || tab.Doc.Buffer.Selection() != selection {
						t.Fatal("published reconfiguration changed source, selection, version or undo history")
					}
				}
			}
			// Production's SetEnabled call is real. A second real prepared write
			// yields a cleanup warning for exactly those same published bytes.
			if err := a.pluginManager.SetEnabled("markdown", enabled); err != nil {
				t.Fatal(err)
			}
			outcome := pluginPublicationOutcome(t, plugins.SettingsPath(a.configDir), true)
			wrapped := fmt.Errorf("outer result: %w", outcome)
			a.finishSetPluginEnabled("markdown", enabled, wrapped)
			for _, check := range verify {
				check()
			}
			if !strings.Contains(a.Message, outcome.Error()) || !strings.Contains(a.Message, "was published") ||
				!strings.Contains(strings.Join(a.Issues, "\n"), outcome.Error()) || strings.Contains(a.Message, "回滚") {
				t.Fatalf("postpublication warning disappeared or claimed rollback: %q issues=%v", a.Message, a.Issues)
			}
			settings, err := plugins.LoadSettings(a.configDir)
			if err != nil || settings.Disabled["markdown"] == enabled || a.pluginManager.Enabled("markdown") != enabled {
				t.Fatalf("UI/manager/disk result disagree: %+v %v", settings, err)
			}
			for _, view := range oldViews {
				if view.flushes != 0 {
					t.Fatal("result handling flushed the old view a second time")
				}
			}
		})
	}
}

func TestPluginPublicationNormalSetEnabledUsesResultBoundary(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.NewDocument("markdown")
	tab := a.Current()
	for _, enabled := range []bool{false, true} {
		view := &failingView{}
		tab.View = view
		generation := tab.OutlineFactoryGeneration
		before := tab.Doc.Buffer.Snapshot()
		history, selection := tab.Doc.Buffer.History(), tab.Doc.Buffer.Selection()
		a.setPluginEnabled("markdown", enabled)
		wantPlugin, wantState := "code", "已停用"
		if enabled {
			wantPlugin, wantState = "markdown", "已启用"
		}
		settings, err := plugins.LoadSettings(a.configDir)
		if err != nil || settings.Disabled["markdown"] == enabled || a.pluginManager.Enabled("markdown") != enabled ||
			tab.MatchedPlugin.ID != wantPlugin || tab.OutlineFactoryGeneration <= generation || tab.View != nil ||
			view.flushes != 1 || view.releases != 1 || a.Message != "插件 markdown "+wantState {
			t.Fatalf("real setPluginEnabled was not wired through result handling: enabled=%v match=%q flush=%d release=%d disk=%v err=%v message=%q",
				enabled, tab.MatchedPlugin.ID, view.flushes, view.releases, settings.Disabled, err, a.Message)
		}
		if !tab.Doc.Buffer.EqualSnapshot(before) || tab.Doc.Buffer.History() != history || tab.Doc.Buffer.Selection() != selection {
			t.Fatal("normal plugin transition changed source, undo or selection")
		}
	}
}

func TestPluginPublicationPrecommitRetainsTabAndExactSettings(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.NewDocument("markdown")
	tab := a.Current()
	view := &failingView{}
	tab.View = view
	generation, adapter := tab.OutlineFactoryGeneration, tab.Adapter
	if err := plugins.SaveSettings(a.configDir, plugins.Settings{Disabled: map[string]bool{}}); err != nil {
		t.Fatal(err)
	}
	path := plugins.SettingsPath(a.configDir)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	outcome := pluginPublicationOutcome(t, path, false)
	a.finishSetPluginEnabled("markdown", false, fmt.Errorf("outer result: %w", outcome))
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) || tab.View != view || view.releases != 0 ||
		tab.Adapter != adapter || tab.OutlineFactoryGeneration != generation || !a.pluginManager.Enabled("markdown") {
		t.Fatalf("precommit refusal changed settings or tab presentation: %v message=%q", err, a.Message)
	}
	if !strings.Contains(a.Message, outcome.Error()) || !strings.Contains(strings.Join(a.Issues, "\n"), outcome.Error()) {
		t.Fatalf("precommit cause was hidden: %q issues=%v", a.Message, a.Issues)
	}
}

func TestPluginPublicationCombinedLoadFailureKeepsBothWarnings(t *testing.T) {
	a, _ := routingApp(t)
	a.NewFromPlugin("task-notes")
	tab := a.Current()
	view := &failingView{}
	tab.View = view
	generation := tab.OutlineFactoryGeneration
	if err := os.Remove(filepath.Join(a.configDir, "plugins", "task-notes.json")); err != nil {
		t.Fatal(err)
	}
	// This real SetEnabled publishes and detects the missing manifest.
	loadErr := a.pluginManager.SetEnabled("task-notes", true)
	if !errors.Is(loadErr, plugins.ErrEnabledNotLoaded) {
		t.Fatalf("missing plugin did not produce enabled-not-loaded: %v", loadErr)
	}
	publicationErr := pluginPublicationOutcome(t, plugins.SettingsPath(a.configDir), true)
	outcome := fmt.Errorf("combined result: %w", errors.Join(publicationErr, loadErr))
	var publication *document.StatePublicationError
	if !errors.Is(outcome, plugins.ErrEnabledNotLoaded) || !errors.As(outcome, &publication) || !publication.Committed {
		t.Fatal("combined fixture lost a real outcome")
	}
	a.finishSetPluginEnabled("task-notes", true, outcome)
	if tab.View != nil || view.releases != 1 || tab.Adapter != nil || tab.Mode != "source" ||
		tab.OutlineFactoryGeneration <= generation || a.pluginManager.Enabled("task-notes") {
		t.Fatalf("combined failure did not reconcile stale tabs: %q", a.Message)
	}
	for _, warning := range []string{publicationErr.Error(), plugins.ErrEnabledNotLoaded.Error()} {
		if !strings.Contains(a.Message, warning) || !strings.Contains(strings.Join(a.Issues, "\n"), warning) {
			t.Fatalf("combined outcome lost %q: %q issues=%v", warning, a.Message, a.Issues)
		}
	}
}
