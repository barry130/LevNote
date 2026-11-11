package session

import (
	"errors"
	"path/filepath"
	"strings"

	"levnote.local/native/core/document"
)

const binaryRecordValue = "binary"

const (
	binaryRecordKey       = "record"
	binaryKindKey         = "kind"
	binaryPathKey         = "path"
	binaryTitleKey        = "title"
	binaryPluginKey       = "pluginId"
	binaryViewKey         = "view"
	binaryDraftKey        = "draft"
	binaryEncodingKey     = "encoding"
	binaryBOMKey          = "bom"
	binaryOriginalPathKey = "originalPath"
	binarySelectionKey    = "selection"
	binaryScrollKey       = "scroll"
)

// binaryTabWire is the only persisted shape for binary authority. It is a
// distinct type so the draft-bearing Tab keeps its ordinary stdlib encoder and
// text sessions never pay for a custom marshal path.
type binaryTabWire struct {
	Record   string              `json:"record"`
	Kind     document.BinaryKind `json:"kind"`
	Path     string              `json:"path"`
	Title    string              `json:"title,omitempty"`
	PluginID string              `json:"pluginId,omitempty"`
	View     string              `json:"view,omitempty"`
}

func newBinaryTab(record BinaryTab) (Tab, error) {
	if err := validateBinaryTab(record); err != nil {
		return Tab{}, err
	}
	value := record
	return Tab{Path: record.Path, binary: &value}, nil
}

func validateBinaryTab(record BinaryTab) error {
	for _, value := range []string{record.Path, record.Title, record.PluginID, record.View} {
		if err := validMetadataString(value); err != nil {
			return err
		}
	}
	if record.Path == "" || !filepath.IsAbs(record.Path) || strings.IndexByte(record.Path, 0) >= 0 {
		return errors.New("binary tab path must be absolute and contain no NUL")
	}
	switch record.Kind {
	case document.BinaryKindUnknown, document.BinaryKindPNG, document.BinaryKindJPEG:
	default:
		return errors.New("unknown binary tab kind")
	}
	return nil
}

func validateBinaryTabContainer(tab Tab) error {
	if tab.binary == nil {
		return errors.New("binary tab metadata is missing")
	}
	if err := validateBinaryTab(*tab.binary); err != nil {
		return err
	}
	if tab.OriginalPath != "" || tab.Draft != nil || tab.Encoding != "" || tab.BOM ||
		tab.Selection.Anchor != 0 || tab.Selection.Head != 0 || tab.Scroll != 0 {
		return errors.New("binary tab contains text-only fields")
	}
	return nil
}

func binaryTabEncodedLength(record BinaryTab) int {
	total := len(`{"record":"binary","kind":,"path":}`) +
		draftEncodedBytes(string(record.Kind)) + draftEncodedBytes(record.Path)
	total += optionalStringBytes("title", record.Title)
	total += optionalStringBytes("pluginId", record.PluginID)
	total += optionalStringBytes("view", record.View)
	return total
}

func hasBinaryTab(snapshot Snapshot) bool {
	for _, tab := range snapshot.Tabs {
		if tab.binary != nil {
			return true
		}
	}
	return false
}
