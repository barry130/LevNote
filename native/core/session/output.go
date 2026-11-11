package session

import (
	"encoding/json"
	"errors"
	"io"
)

var (
	errEncodedSessionTooLarge  = errors.New("encoded session exceeds preflight length")
	errEncodedSessionTruncated = errors.New("encoded session does not match preflight length and terminator")
)

// exactSessionWriter forwards every encoded byte except the final Encoder
// newline. It keeps one pending byte, enforces the preflight length, and turns
// short destination writes into errors before the atomic temp file is published.
type exactSessionWriter struct {
	dst        io.Writer
	expected   int
	received   int
	pending    byte
	hasPending bool
	err        error
}

func newEncodedSessionWriter(dst io.Writer, expected int) *exactSessionWriter {
	return &exactSessionWriter{dst: dst, expected: expected}
}

func (w *exactSessionWriter) Write(data []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	if len(data) == 0 {
		return 0, nil
	}
	if w.received > w.expected+1-len(data) {
		w.err = errEncodedSessionTooLarge
		return 0, w.err
	}
	if w.hasPending {
		if err := writeExactly(w.dst, []byte{w.pending}); err != nil {
			w.err = err
			return 0, w.err
		}
	}
	if len(data) > 1 {
		if err := writeExactly(w.dst, data[:len(data)-1]); err != nil {
			w.err = err
			return 0, w.err
		}
	}
	w.pending = data[len(data)-1]
	w.hasPending = true
	w.received += len(data)
	return len(data), nil
}

func (w *exactSessionWriter) Finish() error {
	if w.err != nil {
		return w.err
	}
	if !w.hasPending || w.pending != '\n' || w.received != w.expected+1 {
		w.err = errEncodedSessionTruncated
		return w.err
	}
	return nil
}

func writeExactly(w io.Writer, data []byte) error {
	n, err := w.Write(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return io.ErrShortWrite
	}
	return nil
}

func writeSnapshotJSON(w io.Writer, snapshot Snapshot, expected int) error {
	exact := newEncodedSessionWriter(w, expected)
	var encoded any = snapshot
	if hasBinaryTab(snapshot) {
		encoded = newSnapshotWire(snapshot)
	}
	if err := json.NewEncoder(exact).Encode(encoded); err != nil {
		return err
	}
	return exact.Finish()
}

type snapshotWire struct {
	Version     int         `json:"version"`
	Root        string      `json:"root,omitempty"`
	Active      int         `json:"active"`
	Tabs        []any       `json:"tabs"`
	Preferences Preferences `json:"preferences"`
}

func newSnapshotWire(snapshot Snapshot) snapshotWire {
	wire := snapshotWire{
		Version: snapshot.Version, Root: snapshot.Root, Active: snapshot.Active,
		Tabs: make([]any, 0, len(snapshot.Tabs)), Preferences: snapshot.Preferences,
	}
	for _, tab := range snapshot.Tabs {
		if tab.binary != nil {
			wire.Tabs = append(wire.Tabs, binaryTabWire{
				Record:   binaryRecordValue,
				Kind:     tab.binary.Kind,
				Path:     tab.binary.Path,
				Title:    tab.binary.Title,
				PluginID: tab.binary.PluginID,
				View:     tab.binary.View,
			})
		} else {
			wire.Tabs = append(wire.Tabs, tab)
		}
	}
	return wire
}
