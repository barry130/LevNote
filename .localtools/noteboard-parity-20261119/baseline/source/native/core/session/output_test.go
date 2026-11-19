package session

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"levnote.local/native/core/document"
	"levnote.local/native/core/text"
)

func TestSessionOutputMatchesJSONMarshal(t *testing.T) {
	root := t.TempDir()
	falseValue := false
	trueValue := true
	draft := "first\r\n中文🙂\u2028\u2029\b\f\t\x00\x1f\"\\/<>&"
	for _, test := range []struct {
		name     string
		snapshot Snapshot
	}{
		{
			name:     "nil-tabs",
			snapshot: Snapshot{Version: Version},
		},
		{
			name: "empty-tabs-and-false-line-numbers",
			snapshot: Snapshot{
				Version: Version,
				Tabs:    []Tab{},
				Preferences: Preferences{
					LineNumbers: &falseValue,
				},
			},
		},
		{
			name: "true-line-numbers",
			snapshot: Snapshot{
				Version: Version,
				Preferences: Preferences{
					LineNumbers: &trueValue,
				},
			},
		},
		{
			name: "dirty-and-saved-tabs",
			snapshot: Snapshot{
				Version: Version,
				Root:    root,
				Active:  1,
				Tabs: []Tab{
					{
						Path:         filepath.Join(root, "<&>\"中文.md"),
						OriginalPath: filepath.Join(root, "original\r\n.md"),
						Title:        "saved <&> \"quoted\" \\",
						Encoding:     document.UTF8,
						BOM:          true,
						Selection:    text.Range{Anchor: 17, Head: 9},
						Scroll:       1e21,
						PluginID:     "levnote.markdown",
						View:         "source",
					},
					{
						Path:      filepath.Join(root, "dirty.txt"),
						Title:     "controls \b\f\n\r\t\x00\x1f",
						Draft:     &draft,
						Encoding:  document.UTF16LE,
						BOM:       true,
						Selection: text.Range{Anchor: 123, Head: 45},
						Scroll:    math.MaxFloat32,
					},
				},
				Preferences: Preferences{
					Theme:        "dark",
					FontSize:     15.125,
					LineNumbers:  nil,
					WordWrap:     true,
					SidebarWidth: 230.5,
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			assertSessionOutputMatchesMarshal(t, test.snapshot)
		})
	}

	for _, scroll := range []float32{
		0,
		float32(math.Copysign(0, -1)),
		math.SmallestNonzeroFloat32,
		1e-8,
		1e-6,
		1e20,
		1e21,
		math.MaxFloat32,
	} {
		value := ""
		assertSessionOutputMatchesMarshal(t, Snapshot{
			Version: Version,
			Tabs:    []Tab{{Draft: &value, Scroll: scroll}},
		})
	}
}

func TestSessionOutputAdapterHandlesChunkingAndEmptyWrites(t *testing.T) {
	value := "chunked 中文🙂"
	snapshot := Snapshot{Version: Version, Tabs: []Tab{{Draft: &value}}}
	want, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	var got bytes.Buffer
	writer := newEncodedSessionWriter(&got, len(want))
	encoded := append(append([]byte(nil), want...), '\n')
	for len(encoded) != 0 {
		step := min(7, len(encoded))
		if n, err := writer.Write(nil); n != 0 || err != nil {
			t.Fatalf("empty write = %d, %v", n, err)
		}
		if n, err := writer.Write(encoded[:step]); n != step || err != nil {
			t.Fatalf("chunk write = %d, %v", n, err)
		}
		encoded = encoded[step:]
	}
	if err := writer.Finish(); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Bytes(), want) {
		t.Fatalf("chunked output differs\n got: %q\nwant: %q", got.Bytes(), want)
	}
}

func TestSessionOutputAdapterRejectsMalformedFinalization(t *testing.T) {
	want := []byte(`{"version":1}`)
	for _, test := range []struct {
		name     string
		expected int
		input    []byte
		wantErr  error
	}{
		{
			name:     "absent-final-newline",
			expected: len(want),
			input:    want,
			wantErr:  errEncodedSessionTruncated,
		},
		{
			name:     "wrong-final-byte",
			expected: len(want),
			input:    append(append([]byte(nil), want[:len(want)-1]...), '\r'),
			wantErr:  errEncodedSessionTruncated,
		},
		{
			name:     "truncated",
			expected: len(want),
			input:    append(append([]byte(nil), want[:len(want)-2]...), '\n'),
			wantErr:  errEncodedSessionTruncated,
		},
		{
			name:     "excess",
			expected: len(want),
			input:    append(append([]byte(nil), want...), 'x', '\n'),
			wantErr:  errEncodedSessionTooLarge,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			writer := newEncodedSessionWriter(&output, test.expected)
			_, writeErr := writer.Write(test.input)
			finishErr := writer.Finish()
			if !errors.Is(writeErr, test.wantErr) && !errors.Is(finishErr, test.wantErr) {
				t.Fatalf("write=%v finish=%v want=%v", writeErr, finishErr, test.wantErr)
			}
		})
	}
}

func TestSessionOutputAdapterPreservesWriteErrors(t *testing.T) {
	injected := errors.New("injected output failure")
	for _, test := range []struct {
		name  string
		write io.Writer
		want  error
	}{
		{
			name:  "sentinel",
			write: &failureWriter{limit: 2, err: injected},
			want:  injected,
		},
		{
			name:  "short-write",
			write: &failureWriter{limit: 2},
			want:  io.ErrShortWrite,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			writer := newEncodedSessionWriter(test.write, len(`{"version":1}`))
			if _, err := writer.Write([]byte(`{"version":1}` + "\n")); !errors.Is(err, test.want) {
				t.Fatalf("Write error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestSessionOutputAdapterErrorsAreSticky(t *testing.T) {
	want := []byte(`{"version":1}`)
	encoded := append(append([]byte(nil), want...), '\n')
	t.Run("excess", func(t *testing.T) {
		writer := newEncodedSessionWriter(io.Discard, len(want))
		if n, err := writer.Write(encoded); n != len(encoded) || err != nil {
			t.Fatalf("valid Write = %d, %v", n, err)
		}
		if _, err := writer.Write([]byte("x")); !errors.Is(err, errEncodedSessionTooLarge) {
			t.Fatalf("excess Write error = %v", err)
		}
		if err := writer.Finish(); !errors.Is(err, errEncodedSessionTooLarge) {
			t.Fatalf("Finish after excess = %v", err)
		}
		if _, err := writer.Write(nil); !errors.Is(err, errEncodedSessionTooLarge) {
			t.Fatalf("Write after excess = %v", err)
		}
	})

	injected := errors.New("injected sticky failure")
	for _, test := range []struct {
		name string
		sink *stagedFailureWriter
		want error
	}{
		{
			name: "sentinel",
			sink: &stagedFailureWriter{failCall: 2, err: injected},
			want: injected,
		},
		{
			name: "short-write",
			sink: &stagedFailureWriter{failCall: 2},
			want: io.ErrShortWrite,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			writer := newEncodedSessionWriter(test.sink, len(want))
			split := len(encoded) / 2
			if n, err := writer.Write(encoded[:split]); n != split || err != nil {
				t.Fatalf("valid Write = %d, %v", n, err)
			}
			if _, err := writer.Write(encoded[split:]); !errors.Is(err, test.want) {
				t.Fatalf("failed Write error = %v, want %v", err, test.want)
			}
			calls := test.sink.calls
			if err := writer.Finish(); !errors.Is(err, test.want) {
				t.Fatalf("Finish after sink failure = %v, want %v", err, test.want)
			}
			if _, err := writer.Write([]byte("again")); !errors.Is(err, test.want) {
				t.Fatalf("Write after sink failure = %v, want %v", err, test.want)
			}
			if test.sink.calls != calls {
				t.Fatalf("sticky failure emitted again: calls %d -> %d", calls, test.sink.calls)
			}
		})
	}
}

func TestSessionOutputRepresentativeBounds(t *testing.T) {
	maxDraft := strings.Repeat("x", MaxDraftBytes)
	assertSessionOutputToDiscard(t, "4MiB-draft", Snapshot{
		Version: Version,
		Tabs:    []Tab{{Draft: &maxDraft}},
	})

	tabs := make([]Tab, MaxTabs)
	value := "draft"
	for i := range tabs {
		tabs[i] = Tab{Draft: &value}
	}
	assertSessionOutputToDiscard(t, "100-tabs", Snapshot{
		Version: Version,
		Active:  MaxTabs - 1,
		Tabs:    tabs,
	})

	nearLimit := strings.Repeat("x", MaxDraftBytes-256)
	nearLimitSnapshot := Snapshot{
		Version: Version,
		Tabs: []Tab{
			{Draft: &nearLimit},
			{Draft: &nearLimit},
			{Draft: &nearLimit},
			{Draft: &nearLimit},
		},
	}
	if size, err := preflightSnapshot(nearLimitSnapshot); err != nil || size <= MaxBytes-(1<<20) || size > MaxBytes {
		t.Fatalf("near-limit fixture size = %d, err = %v", size, err)
	}
	assertSessionOutputToDiscard(t, "near-16MiB", nearLimitSnapshot)
}

func TestSaveRejectsInvalidAndOverLimitBeforeCreatingDirectory(t *testing.T) {
	root := t.TempDir()
	invalid := "\xff"
	baseDraft := "draft"
	base := Snapshot{Tabs: []Tab{{Draft: &baseDraft}}}
	baseSize, err := preflightSnapshot(base)
	if err != nil {
		t.Fatal(err)
	}
	tooLarge := base
	tooLarge.Tabs = append([]Tab(nil), base.Tabs...)
	tooLarge.Tabs[0].Title = strings.Repeat("a", MaxBytes-baseSize-10)
	tooLargeSize := optionalStringBytes("title", tooLarge.Tabs[0].Title)
	if baseSize+tooLargeSize != MaxBytes+1 {
		t.Fatalf("fixture crosses limit by %d bytes", baseSize+tooLargeSize-MaxBytes)
	}
	for _, test := range []struct {
		name     string
		snapshot Snapshot
	}{
		{
			name: "invalid",
			snapshot: Snapshot{
				Tabs: []Tab{{Draft: &invalid}},
			},
		},
		{
			name:     "just-over",
			snapshot: tooLarge,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(root, test.name, "state.json")
			if err := Save(path, test.snapshot); err == nil {
				t.Fatal("invalid snapshot saved")
			}
			if _, err := os.Stat(filepath.Dir(path)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("rejected save created destination directory: %v", err)
			}
		})
	}
}

func TestSaveOutputFailuresPreserveRecoveryAndTimestamp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := document.WritePrivateState(path, []byte("old recovery")); err != nil {
		t.Fatal(err)
	}
	oldTime := time.Unix(1_700_000_000, 0)
	if err := os.Chtimes(path, oldTime, oldTime); err != nil {
		t.Fatal(err)
	}
	beforeInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := Snapshot{Version: Version, Tabs: []Tab{{Path: filepath.Join(dir, "saved.txt")}}}
	expected, err := preflightSnapshot(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	injected := errors.New("injected atomic callback failure")
	for _, test := range []struct {
		name  string
		write func(io.Writer) error
		want  error
	}{
		{
			name: "wrong-expected-length",
			write: func(w io.Writer) error {
				return writeSnapshotJSON(w, snapshot, expected+1)
			},
			want: errEncodedSessionTruncated,
		},
		{
			name: "real-pipeline-sentinel",
			write: func(w io.Writer) error {
				return writeSnapshotJSON(&prefixFailureWriter{
					dst:    w,
					prefix: 7,
					err:    injected,
				}, snapshot, expected)
			},
			want: injected,
		},
		{
			name: "real-pipeline-short-write",
			write: func(w io.Writer) error {
				return writeSnapshotJSON(&prefixFailureWriter{
					dst:    w,
					prefix: 7,
				}, snapshot, expected)
			},
			want: io.ErrShortWrite,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := document.WritePrivateStateTo(path, test.write); !errors.Is(err, test.want) {
				t.Fatalf("WritePrivateStateTo error = %v, want %v", err, test.want)
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != "old recovery" {
				t.Fatalf("failed output published bytes: %q", got)
			}
			afterInfo, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if !afterInfo.ModTime().Equal(beforeInfo.ModTime()) {
				t.Fatalf("failed output changed timestamp: before=%v after=%v", beforeInfo.ModTime(), afterInfo.ModTime())
			}
			matches, err := filepath.Glob(filepath.Join(dir, ".levnote-save-*"))
			if err != nil {
				t.Fatal(err)
			}
			if len(matches) != 0 {
				t.Fatalf("temporary files remain: %v", matches)
			}
		})
	}
}

func TestSessionOutputSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	draft := "round trip\r\n中文🙂\u2028\u2029\b<&>\"\\"
	snapshot := Snapshot{
		Version: Version,
		Tabs: []Tab{{
			Path:      filepath.Join(filepath.Dir(path), "saved.txt"),
			Title:     "controls \f\t",
			Draft:     &draft,
			Encoding:  document.UTF8,
			BOM:       true,
			Selection: text.Range{Anchor: 9, Head: 3},
			Scroll:    1e20,
		}},
	}
	if err := Save(path, snapshot); err != nil {
		t.Fatal(err)
	}
	loaded, issues := Load(path)
	if len(issues) != 0 || loaded.Version != Version || len(loaded.Tabs) != 1 {
		t.Fatalf("Load = %#v, issues = %v", loaded, issues)
	}
	got := loaded.Tabs[0]
	if got.Draft == nil || *got.Draft != draft || got.Title != "controls \f\t" || got.Selection != snapshot.Tabs[0].Selection {
		t.Fatalf("round-trip tab = %#v", got)
	}
}

func assertSessionOutputMatchesMarshal(t *testing.T, snapshot Snapshot) {
	t.Helper()
	want, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := preflightSnapshot(snapshot)
	if err != nil || expected != len(want) {
		t.Fatalf("preflight = %d, Marshal = %d, err = %v", expected, len(want), err)
	}
	var got bytes.Buffer
	if err := writeSnapshotJSON(&got, snapshot, expected); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got.Bytes(), want) {
		t.Fatalf("output differs\n got: %q\nwant: %q", got.Bytes(), want)
	}
	if bytes.HasSuffix(got.Bytes(), []byte{'\n'}) {
		t.Fatal("published session JSON retained Encoder's terminal newline")
	}
}

func assertSessionOutputToDiscard(t *testing.T, name string, snapshot Snapshot) {
	t.Helper()
	expected, err := preflightSnapshot(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	t.Run(name, func(t *testing.T) {
		if err := writeSnapshotJSON(io.Discard, snapshot, expected); err != nil {
			t.Fatal(err)
		}
	})
}

type failureWriter struct {
	limit int
	err   error
}

func (w *failureWriter) Write(data []byte) (int, error) {
	if len(data) <= w.limit {
		return len(data), nil
	}
	return w.limit, w.err
}

type stagedFailureWriter struct {
	calls    int
	failCall int
	err      error
}

func (w *stagedFailureWriter) Write(data []byte) (int, error) {
	w.calls++
	if w.calls != w.failCall {
		return len(data), nil
	}
	return 0, w.err
}

type prefixFailureWriter struct {
	dst    io.Writer
	prefix int
	err    error
}

func (w *prefixFailureWriter) Write(data []byte) (int, error) {
	n := min(w.prefix, len(data))
	if n == 0 {
		return 0, w.err
	}
	written, err := w.dst.Write(data[:n])
	if err != nil {
		return written, err
	}
	if written != n {
		return written, io.ErrShortWrite
	}
	return n, w.err
}
