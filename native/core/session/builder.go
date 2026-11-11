package session

import (
	"errors"
	"fmt"

	"levnote.local/native/core/document"
)

// CaptureOptions carries view metadata into the same preflight as the draft.
// Set these fields before capture; adding them to a returned snapshot afterward
// would bypass the builder's early budget check (Save still validates it).
type CaptureOptions struct {
	Title        string
	Scroll       float32
	OriginalPath string
	PluginID     string
	View         string
}

// SnapshotBuilder is owned by the same goroutine as its documents. It counts
// exact JSON bytes before materializing each draft and commits only successful
// additions. No document, buffer or caller-owned mutable pointer is retained.
type SnapshotBuilder struct {
	snapshot      Snapshot
	tabsBytes     int // sum of encoded tab objects, without commas/brackets
	rawDraftBytes int
}

func NewSnapshotBuilder(root string) *SnapshotBuilder {
	return &SnapshotBuilder{snapshot: Snapshot{Version: Version, Root: root}}
}

func (b *SnapshotBuilder) Len() int {
	if b == nil {
		return 0
	}
	return len(b.snapshot.Tabs)
}

func (b *SnapshotBuilder) AddDocument(doc *document.Document, title string, scroll float32) error {
	return b.AddDocumentWithOptions(doc, CaptureOptions{Title: title, Scroll: scroll})
}

func (b *SnapshotBuilder) AddDocumentWithOptions(doc *document.Document, options CaptureOptions) error {
	if b == nil {
		return errNilBuilder
	}
	if doc == nil || doc.Buffer == nil {
		return errors.New("session document or text buffer is nil")
	}
	tab := Tab{Path: doc.Path, Title: options.Title, Encoding: doc.Encoding,
		BOM: doc.BOM, Selection: doc.Buffer.Selection(), Scroll: options.Scroll,
		OriginalPath: options.OriginalPath, PluginID: options.PluginID, View: options.View}
	dirty := doc.Dirty()
	if err := validateTabMetadata(tab, dirty); err != nil {
		return err
	}
	raw, encoded := 0, -1
	if dirty {
		raw = doc.Buffer.LenBytes()
		if raw > MaxDraftBytes {
			return fmt.Errorf("dirty draft is %d bytes; recovery limit is %d bytes and the original draft is not protected", raw, MaxDraftBytes)
		}
	}
	// Reject exhausted aggregate/metadata before even walking piece descriptors.
	minimumEncoded := -1
	if dirty {
		minimumEncoded = 2
	}
	if _, err := b.preflightAddition(tab, minimumEncoded, raw); err != nil {
		return err
	}
	if dirty {
		var err error
		encoded, err = draftEncodedLength(doc.Buffer)
		if err != nil {
			return err
		}
	}
	size, err := b.preflightAddition(tab, encoded, raw)
	if err != nil {
		return err
	}
	if dirty {
		value := doc.Buffer.String()
		tab.Draft = &value
	}
	b.append(tab, size, raw)
	return nil
}

// AddTab handles a saved file whose dirty draft was explicitly discarded, or
// an already-owned recovery record. The caller can safely reuse/change tab and
// its Draft pointer after this returns. Strings themselves are immutable.
func (b *SnapshotBuilder) AddTab(tab Tab) error {
	if b == nil {
		return errNilBuilder
	}
	if err := validateTab(tab); err != nil {
		return err
	}
	raw, encoded := 0, -1
	if tab.Draft != nil {
		raw, encoded = len(*tab.Draft), draftEncodedBytes(*tab.Draft)
	}
	size, err := b.preflightAddition(tab, encoded, raw)
	if err != nil {
		return err
	}
	b.append(cloneTab(tab), size, raw)
	return nil
}

// AddBinaryTab captures only path/metadata authority. Rejection happens before
// any filesystem operation; restoration later opens the file separately.
func (b *SnapshotBuilder) AddBinaryTab(record BinaryTab) error {
	if b == nil {
		return errNilBuilder
	}
	tab, err := newBinaryTab(record)
	if err != nil {
		return err
	}
	size := binaryTabEncodedLength(*tab.binary)
	if _, err := b.preflightAddition(tab, -1, 0); err != nil {
		return err
	}
	b.append(tab, size, 0)
	return nil
}

var errNilBuilder = errors.New("session snapshot builder is nil")

func (b *SnapshotBuilder) preflightAddition(tab Tab, draftSize, raw int) (int, error) {
	if len(b.snapshot.Tabs) >= MaxTabs {
		return 0, fmt.Errorf("session has more than %d tabs; existing recovery file kept", MaxTabs)
	}
	if raw > MaxDraftAggregateBytes-b.rawDraftBytes {
		return 0, errRecoveryLimit
	}
	size := tabEncodedLength(tab, draftSize)
	if _, err := snapshotEnvelopeLength(b.snapshot, len(b.snapshot.Tabs)+1, b.tabsBytes+size, false); err != nil {
		return 0, err
	}
	return size, nil
}

func (b *SnapshotBuilder) append(tab Tab, size, raw int) {
	b.snapshot.Tabs = append(b.snapshot.Tabs, tab)
	b.tabsBytes += size
	b.rawDraftBytes += raw
}

func (b *SnapshotBuilder) SetActive(index int) error {
	if b == nil {
		return errNilBuilder
	}
	next := b.snapshot
	next.Active = index
	if _, err := snapshotEnvelopeLength(next, len(next.Tabs), b.tabsBytes, next.Tabs == nil); err != nil {
		return err
	}
	b.snapshot.Active = index
	return nil
}

func (b *SnapshotBuilder) SetPreferences(preferences Preferences) error {
	if b == nil {
		return errNilBuilder
	}
	next := b.snapshot
	next.Preferences = preferences
	if _, err := snapshotEnvelopeLength(next, len(next.Tabs), b.tabsBytes, next.Tabs == nil); err != nil {
		return err
	}
	b.snapshot.Preferences = clonePreferences(preferences)
	return nil
}

// Snapshot returns independent mutable containers; retaining or editing this
// result cannot invalidate the builder's cached lengths or later snapshots.
func (b *SnapshotBuilder) Snapshot() (Snapshot, error) {
	if b == nil {
		return Snapshot{}, errNilBuilder
	}
	if _, err := snapshotEnvelopeLength(b.snapshot, len(b.snapshot.Tabs), b.tabsBytes, b.snapshot.Tabs == nil); err != nil {
		return Snapshot{}, err
	}
	next := b.snapshot
	next.Preferences = clonePreferences(next.Preferences)
	if next.Tabs != nil {
		next.Tabs = make([]Tab, len(b.snapshot.Tabs))
		for i, tab := range b.snapshot.Tabs {
			next.Tabs[i] = cloneTab(tab)
		}
	}
	return next, nil
}

func (b *SnapshotBuilder) Save(path string) error {
	snapshot, err := b.Snapshot()
	if err != nil {
		return err
	}
	return Save(path, snapshot)
}

func cloneTab(tab Tab) Tab {
	if tab.binary != nil {
		value := *tab.binary
		tab.binary = &value
	}
	if tab.Draft != nil {
		value := *tab.Draft
		tab.Draft = &value
	}
	return tab
}

func clonePreferences(preferences Preferences) Preferences {
	if preferences.LineNumbers != nil {
		value := *preferences.LineNumbers
		preferences.LineNumbers = &value
	}
	return preferences
}
