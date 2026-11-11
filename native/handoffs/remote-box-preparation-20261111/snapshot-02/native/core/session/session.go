// Package session persists bounded workspace recovery. Dirty drafts recover as
// new copies; they never acquire the current disk file's save/conflict token.
package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"levnote.local/native/core/document"
	"levnote.local/native/core/text"
)

const Version = 1
const MaxBytes = 16 << 20
const MaxDraftBytes = 4 << 20
const MaxDraftAggregateBytes = MaxBytes
const MaxTabs = 100

type Preferences struct {
	Theme        string  `json:"theme,omitempty"`
	FontSize     float32 `json:"fontSize,omitempty"`
	LineNumbers  *bool   `json:"lineNumbers,omitempty"`
	WordWrap     bool    `json:"wordWrap"`
	SidebarWidth float32 `json:"sidebarWidth,omitempty"`
}
type Tab struct {
	Path         string            `json:"path,omitempty"`
	OriginalPath string            `json:"originalPath,omitempty"`
	Title        string            `json:"title,omitempty"`
	Draft        *string           `json:"draft,omitempty"`
	Encoding     document.Encoding `json:"encoding,omitempty"`
	BOM          bool              `json:"bom"`
	Selection    text.Range        `json:"selection"`
	Scroll       float32           `json:"scroll,omitempty"`
	PluginID     string            `json:"pluginId,omitempty"`
	View         string            `json:"view,omitempty"`
	binary       *BinaryTab
}

type BinaryTab struct {
	Path     string
	Kind     document.BinaryKind
	Title    string
	PluginID string
	View     string
}

type RestoredBinary struct {
	Path     string
	Kind     document.BinaryKind
	Title    string
	PluginID string
	View     string
}
type Snapshot struct {
	Version     int         `json:"version"`
	Root        string      `json:"root,omitempty"`
	Active      int         `json:"active"`
	Tabs        []Tab       `json:"tabs"`
	Preferences Preferences `json:"preferences"`
}

var (
	// ErrBinaryDispatchRequired keeps existing text callers from dereferencing
	// a nil Document when a restored tab is path-only binary authority.
	ErrBinaryDispatchRequired = errors.New("session tab requires binary dispatcher")
	ErrTextDispatchRequired   = errors.New("session tab requires text dispatcher")
)

type Issue struct{ Path, Reason, Detail string }
type Restored struct {
	Document                            *document.Document
	OriginalPath, Title, PluginID, View string
	Scroll                              float32
}

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "LevNote", "native-session.json"), nil
}

func CaptureBounded(doc *document.Document, title string, scroll float32) (Tab, error) {
	builder := NewSnapshotBuilder("")
	if err := builder.AddDocument(doc, title, scroll); err != nil {
		return Tab{}, err
	}
	return builder.snapshot.Tabs[0], nil
}

// Capture is retained only for source compatibility. It cannot report an
// oversized or invalid draft; callers that persist recovery must migrate to
// CaptureBounded and surface its error.
func Capture(doc *document.Document, title string, scroll float32) Tab {
	tab, err := CaptureBounded(doc, title, scroll)
	if err != nil {
		tab.Draft = nil
		return tab
	}
	return tab
}
func Save(path string, snapshot Snapshot) error {
	snapshot.Version = Version
	expected, err := preflightSnapshot(snapshot)
	if err != nil {
		return err
	}
	return document.WritePrivateStateTo(path, func(w io.Writer) error {
		return writeSnapshotJSON(w, snapshot, expected)
	})
}

func Load(path string) (Snapshot, []Issue) {
	result := Snapshot{Version: Version}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err != nil {
		return result, []Issue{{path, "read-failed", err.Error()}}
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, MaxBytes+1))
	if err != nil {
		return result, []Issue{{path, "read-failed", err.Error()}}
	}
	if len(data) > MaxBytes {
		return result, []Issue{{path, "too-large", "recovery file exceeds size limit"}}
	}
	if !utf8.Valid(data) {
		return result, []Issue{{path, "corrupt", "invalid UTF-8 in recovery file"}}
	}
	// Decode tabs independently; one broken item must not discard other drafts.
	var raw struct {
		Version     int         `json:"version"`
		Root        string      `json:"root"`
		Active      int         `json:"active"`
		Tabs        boundedTabs `json:"tabs"`
		Preferences Preferences `json:"preferences"`
	}
	err = json.Unmarshal(data, &raw)
	if err != nil {
		return result, []Issue{{path, "corrupt", err.Error()}}
	}
	if raw.Version != Version {
		return result, []Issue{{path, "unsupported-version", fmt.Sprintf("version %d", raw.Version)}}
	}
	result.Root, result.Preferences = raw.Root, raw.Preferences
	issues := []Issue{}
	if result.Root != "" && (!filepath.IsAbs(result.Root) || strings.IndexByte(result.Root, 0) >= 0) {
		issues = append(issues, Issue{path, "invalid-root", result.Root})
		result.Root = ""
	}
	for i, entry := range raw.Tabs.entries {
		tab := entry.tab
		if entry.err != nil {
			issues = append(issues, Issue{tab.Path, "invalid-tab", fmt.Sprintf("tab %d: %v", i+1, entry.err)})
			continue
		}
		if i == raw.Active {
			result.Active = len(result.Tabs)
		}
		result.Tabs = append(result.Tabs, tab)
	}
	if raw.Tabs.omitted > 0 {
		issues = append(issues, Issue{path, "too-many-tabs", fmt.Sprintf("%d additional tabs omitted", raw.Tabs.omitted)})
	}
	if result.Preferences.FontSize < 9 || result.Preferences.FontSize > 40 {
		result.Preferences.FontSize = 15
	}
	if result.Preferences.SidebarWidth < 160 || result.Preferences.SidebarWidth > 640 {
		result.Preferences.SidebarWidth = 230
	}
	if result.Preferences.Theme != "dark" && result.Preferences.Theme != "light" && result.Preferences.Theme != "system" {
		result.Preferences.Theme = "system"
	}
	return result, issues
}

func RestoreTab(tab Tab) (Restored, error) {
	if err := validateTab(tab); err != nil {
		return Restored{}, err
	}
	if tab.binary != nil {
		return Restored{}, ErrBinaryDispatchRequired
	}
	var doc *document.Document
	var err error
	original := tab.OriginalPath
	if tab.Draft != nil {
		doc = document.New(*tab.Draft)
		if original == "" {
			original = tab.Path
		}
		doc.Encoding = tab.Encoding
		doc.BOM = tab.BOM
		if doc.Encoding == "" {
			doc.Encoding = document.UTF8
		}
	} else {
		doc, err = document.OpenWithEncoding(tab.Path, tab.Encoding)
		if err != nil {
			return Restored{}, err
		}
	}
	anchor, head := tab.Selection.Anchor, tab.Selection.Head
	n := doc.Buffer.LenUTF16()
	if anchor > n {
		anchor = n
	}
	if head > n {
		head = n
	}
	if err = doc.Buffer.SetSelection(anchor, head); err != nil {
		_ = doc.Buffer.SetSelection(0, 0)
	}
	return Restored{doc, original, tab.Title, tab.PluginID, tab.View, tab.Scroll}, nil
}

func RestoreBinary(tab Tab) (RestoredBinary, error) {
	if tab.binary == nil {
		return RestoredBinary{}, ErrTextDispatchRequired
	}
	if err := validateTab(tab); err != nil {
		return RestoredBinary{}, err
	}
	record := *tab.binary
	return RestoredBinary(record), nil
}
