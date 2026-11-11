package search

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"levnote.local/native/core/text"
)

type fakeDirEntry struct {
	name string
	dir  bool
	link bool
}

func (e fakeDirEntry) Name() string { return e.name }
func (e fakeDirEntry) IsDir() bool  { return e.dir }
func (e fakeDirEntry) Type() fs.FileMode {
	if e.link {
		return fs.ModeSymlink
	}
	if e.dir {
		return fs.ModeDir
	}
	return 0
}
func (e fakeDirEntry) Info() (fs.FileInfo, error) { return fakeFileInfo{e}, nil }

type fakeFileInfo struct {
	entry fakeDirEntry
}

func (i fakeFileInfo) Name() string           { return i.entry.name }
func (i fakeFileInfo) Size() int64            { return 0 }
func (i fakeFileInfo) Mode() fs.FileMode      { return i.entry.Type() }
func (i fakeFileInfo) ModTime() (t time.Time) { return }
func (i fakeFileInfo) IsDir() bool            { return i.entry.dir }
func (i fakeFileInfo) Sys() interface{}       { return nil }

type fakeDirectoryBatch struct {
	entries []fs.DirEntry
	read    int
	onRead  func(int)
	onClose func()
}

func (b *fakeDirectoryBatch) Read(n int) ([]fs.DirEntry, error) {
	if b.onRead != nil {
		b.onRead(n)
	}
	if b.read >= len(b.entries) {
		return nil, io.EOF
	}
	end := min(len(b.entries), b.read+n)
	entries := b.entries[b.read:end]
	b.read = end
	return entries, nil
}

func (b *fakeDirectoryBatch) Close() error {
	if b.onClose != nil {
		b.onClose()
	}
	return nil
}

type fakeDirectoryReader struct {
	directories map[string][]fs.DirEntry
	onRead      func(int)
	onClose     func()
}

func (r *fakeDirectoryReader) Open(path string) (directoryBatch, error) {
	entries, ok := r.directories[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return &fakeDirectoryBatch{entries: entries, onRead: r.onRead, onClose: r.onClose}, nil
}

type shortDirectoryReader struct {
	entries []fs.DirEntry
	next    int
	reads   int
}

func (r *shortDirectoryReader) Open(path string) (directoryBatch, error) {
	return &shortDirectoryReader{entries: r.entries, next: r.next, reads: r.reads}, nil
}

func (r *shortDirectoryReader) Read(n int) ([]fs.DirEntry, error) {
	r.reads++
	if r.next >= len(r.entries) {
		return nil, io.EOF
	}
	end := min(len(r.entries), r.next+1)
	entries := r.entries[r.next:end]
	r.next = end
	return entries, nil
}

func (r *shortDirectoryReader) Close() error { return nil }

type stalledDirectoryReader struct {
	closes int
}

func (r *stalledDirectoryReader) Open(string) (directoryBatch, error) {
	return &stalledDirectoryBatch{reader: r}, nil
}

type stalledDirectoryBatch struct {
	reader *stalledDirectoryReader
}

func (*stalledDirectoryBatch) Read(int) ([]fs.DirEntry, error) {
	return nil, nil
}

func (b *stalledDirectoryBatch) Close() error {
	b.reader.closes++
	return nil
}

func TestUnicodeCaseAndWholeWords(t *testing.T) {
	source := "🙂 CAT cat category _cat cat_ 猫 猫咪 e\u0301"
	hits, err := FindString(source, Options{Query: "cat", WholeWord: true})
	if err != nil || len(hits) != 2 || hits[0].Start != 3 || hits[0].End != 6 {
		t.Fatal(hits, err)
	}
	hits, err = FindString(source, Options{Query: "猫", WholeWord: true, CaseSensitive: true})
	if err != nil || len(hits) != 1 {
		t.Fatal(hits, err)
	}
	hits, err = FindString(source, Options{Query: "e", WholeWord: true})
	if err != nil || len(hits) != 0 {
		t.Fatal("combining mark split", hits, err)
	}
}
func TestRegexReplacementIsOneUndoAndRejectsStale(t *testing.T) {
	b := text.New("🙂 a=12 b=34")
	opts := Options{Query: `(\w)=(\d+)`, Regex: true, CaseSensitive: true}
	hits, err := Find(b.Snapshot(), opts)
	if err != nil {
		t.Fatal(err)
	}
	n, err := ReplaceAll(b, opts, "${2}:$1")
	if err != nil || n != 2 || b.String() != "🙂 12:a 34:b" {
		t.Fatal(b.String(), n, err)
	}
	if b.History().UndoDepth != 1 {
		t.Fatal(b.History())
	}
	if err = Replace(b, hits[0], opts, "bad"); !errors.Is(err, ErrStale) {
		t.Fatal(err)
	}
	b.Undo()
	if b.String() != "🙂 a=12 b=34" {
		t.Fatal("replace undo")
	}
	if err = Replace(b, hits[0], opts, "${2}"); err != nil || b.String() != "🙂 12 b=34" {
		t.Fatal(b.String(), err)
	}
}
func TestLiteralDollarZeroWidthAndInvalidRegex(t *testing.T) {
	b := text.New("x x")
	n, err := ReplaceAll(b, Options{Query: "x"}, "$1")
	if err != nil || n != 2 || b.String() != "$1 $1" {
		t.Fatal(b.String(), err)
	}
	b = text.New("甲\n乙")
	n, err = ReplaceAll(b, Options{Query: "(?m)^", Regex: true}, ">")
	if err != nil || n != 2 || b.String() != ">甲\n>乙" {
		t.Fatal(b.String(), err)
	}
	if _, err = FindString("abc", Options{Query: "[", Regex: true}); err == nil {
		t.Fatal("invalid regex accepted")
	}
}
func TestWorkspaceSkipsAreReportedAndCancellationIsExplicit(t *testing.T) {
	root := t.TempDir()
	put := func(path, content string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	put(filepath.Join(root, "a.txt"), "🙂 find\nfind")
	put(filepath.Join(root, "node_modules", "skip.txt"), "find")
	put(filepath.Join(root, "binary"), "a\x00find")
	put(filepath.Join(root, "large"), "find find find find find find find find")
	var found []FileMatch
	var issues []FileIssue
	summary, err := Workspace(context.Background(), root, WorkspaceOptions{Options: Options{Query: "find"}, MaxFileBytes: 24}, func(b Batch) { found = append(found, b.Matches...); issues = append(issues, b.Issues...) })
	if err != nil || summary.Matches != 2 || summary.Skipped != 3 || len(issues) != 3 || found[0].Column != 4 {
		t.Fatal(summary, found, issues, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	summary, err = Workspace(ctx, root, WorkspaceOptions{Options: Options{Query: "find"}}, nil)
	if !errors.Is(err, context.Canceled) || !summary.Cancelled {
		t.Fatal(summary, err)
	}
	summary, err = Workspace(context.Background(), root, WorkspaceOptions{Options: Options{Query: "find"}, MaxResults: 1}, nil)
	if err != nil || !summary.Truncated || summary.Matches != 1 {
		t.Fatal(summary, err)
	}
}

func TestWorkspaceEnforcesActualReadLimitAndEntryBudget(t *testing.T) {
	root := t.TempDir()
	grower := filepath.Join(root, "grower.txt")
	if err := os.WriteFile(grower, []byte("find"), 0600); err != nil {
		t.Fatal(err)
	}
	var issues []FileIssue
	summary, err := Workspace(context.Background(), root, WorkspaceOptions{
		Options: Options{Query: "find"}, MaxFileBytes: 8,
	}, func(batch Batch) { issues = append(issues, batch.Issues...) })
	if err != nil || summary.Files != 1 || summary.Matches != 1 {
		t.Fatal(summary, issues, err)
	}

	// A stat-sized file is still rejected when it grows before the actual read.
	// The limited reader enforces the byte boundary rather than trusting Stat.
	oldOpenHook := workspaceOpenHook
	workspaceOpenHook = func(path string) {
		if path == grower {
			if err := os.WriteFile(path, []byte(strings.Repeat("x", 9)), 0600); err != nil {
				t.Errorf("grow workspace file: %v", err)
			}
		}
	}
	defer func() { workspaceOpenHook = oldOpenHook }()
	issues = nil
	summary, err = Workspace(context.Background(), root, WorkspaceOptions{
		Options: Options{Query: "x"}, MaxFileBytes: 8,
	}, func(batch Batch) { issues = append(issues, batch.Issues...) })
	if err != nil || summary.Files != 0 || len(issues) != 1 || issues[0].Reason != "too-large" {
		t.Fatal(summary, issues, err)
	}

	// A deterministic traversal budget is disclosed rather than accumulating
	// an unbounded directory listing before result caps help.
	workspaceOpenHook = nil
	if err := os.WriteFile(filepath.Join(root, "second.txt"), []byte("find"), 0600); err != nil {
		t.Fatal(err)
	}
	summary, err = Workspace(context.Background(), root, WorkspaceOptions{
		Options: Options{Query: "find"}, MaxEntries: 1,
	}, nil)
	if err != nil || !summary.Truncated || summary.Entries != 1 {
		t.Fatal(summary, err)
	}
}

func TestWorkspaceDirectoryReadsAreBoundedAndCancellable(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "find.txt"), []byte("find"), 0600); err != nil {
		t.Fatal(err)
	}

	readSizes := make([]int, 0, 4)
	closeCalls := 0
	reader := &fakeDirectoryReader{
		onRead: func(n int) { readSizes = append(readSizes, n) },
		onClose: func() {
			closeCalls++
		},
		directories: map[string][]fs.DirEntry{
			root: {
				fakeDirEntry{name: "node_modules", dir: true},
				fakeDirEntry{name: "z-link", link: true},
				fakeDirEntry{name: "a.txt"},
				fakeDirEntry{name: "nested", dir: true},
			},
			filepath.Join(root, "nested"): {
				fakeDirEntry{name: "b.txt"},
			},
		},
	}
	oldReader := workspaceReader
	workspaceReader = reader
	defer func() { workspaceReader = oldReader }()

	var issues []FileIssue
	summary, err := Workspace(context.Background(), root, WorkspaceOptions{
		Options: Options{Query: "find"},
	}, func(batch Batch) { issues = append(issues, batch.Issues...) })
	if err != nil {
		t.Fatal(err)
	}
	for _, size := range readSizes {
		if size != directoryReadBatch {
			t.Fatalf("directory read batch was not bounded: %#v", readSizes)
		}
	}
	if len(readSizes) != 4 {
		t.Fatalf("reader call count = %d; want root, root EOF, nested, and nested EOF (%#v)", len(readSizes), readSizes)
	}
	if closeCalls != 2 {
		t.Fatalf("directory sessions closed = %d; want root and nested to close exactly once", closeCalls)
	}
	if summary.Entries != 5 || summary.Dirs != 2 || summary.Skipped != 4 {
		t.Fatalf("summary = %#v; issues = %#v", summary, issues)
	}
	if len(issues) != 4 {
		t.Fatalf("excluded directory and symlink evidence is wrong: %#v", issues)
	}

	ctx, cancel := context.WithCancel(context.Background())
	reader = &fakeDirectoryReader{
		onRead: func(int) { cancel() },
		onClose: func() {
			closeCalls++
		},
		directories: map[string][]fs.DirEntry{
			root: {fakeDirEntry{name: "stopped.txt"}},
		},
	}
	workspaceReader = reader
	summary, err = Workspace(ctx, root, WorkspaceOptions{Options: Options{Query: "find"}}, nil)
	if !errors.Is(err, context.Canceled) || !summary.Cancelled || summary.Entries != 0 {
		t.Fatalf("cancellation did not stop traversal immediately: %#v %v", summary, err)
	}
	if closeCalls != 3 {
		t.Fatalf("cancelled directory session closed = %d total; want 3", closeCalls)
	}
}

func TestWorkspaceDirectoryTraversalContinuesPastOneBatch(t *testing.T) {
	root := t.TempDir()
	entries := make([]fs.DirEntry, 0, directoryReadBatch+1)
	for i := 0; i < directoryReadBatch+1; i++ {
		entries = append(entries, fakeDirEntry{name: fmt.Sprintf("%04d.txt", i)})
	}
	batches := 0
	reader := &fakeDirectoryReader{
		directories: map[string][]fs.DirEntry{root: entries},
		onRead:      func(int) { batches++ },
	}
	oldReader := workspaceReader
	workspaceReader = reader
	defer func() { workspaceReader = oldReader }()

	summary, err := Workspace(context.Background(), root, WorkspaceOptions{
		Options: Options{Query: "find"}, MaxFileBytes: 1,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if batches != 3 || summary.Entries != directoryReadBatch+1 {
		t.Fatalf("continued read failed: batches=%d summary=%#v", batches, summary)
	}
	if reader.directories[root] == nil {
		t.Fatal("fixture mutated")
	}
}

func TestWorkspaceDirectoryTraversalContinuesShortNilBatchesAndReportsStall(t *testing.T) {
	root := t.TempDir()
	entries := make([]fs.DirEntry, 0, 257)
	for i := 0; i < 256; i++ {
		entries = append(entries, fakeDirEntry{name: fmt.Sprintf("%04d.txt", i)})
	}
	entries = append(entries, fakeDirEntry{name: "last.txt"})
	reader := &shortDirectoryReader{entries: entries}
	oldReader := workspaceReader
	workspaceReader = reader
	defer func() { workspaceReader = oldReader }()

	summary, err := Workspace(context.Background(), root, WorkspaceOptions{
		Options: Options{Query: "find"}, MaxFileBytes: 1,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Entries != len(entries) {
		t.Fatalf("short nil batches stopped early: summary=%#v reads=%d", summary, reader.reads)
	}

	stalled := &stalledDirectoryReader{}
	workspaceReader = stalled
	var issues []FileIssue
	summary, err = Workspace(context.Background(), root, WorkspaceOptions{
		Options: Options{Query: "find"},
	}, func(batch Batch) { issues = append(issues, batch.Issues...) })
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 1 || issues[0].Reason != "read-stalled" || summary.Truncated || summary.Entries != 0 {
		t.Fatalf("no-progress read was silent: summary=%#v issues=%#v", summary, issues)
	}
	if stalled.closes != 1 {
		t.Fatalf("stalled directory session closed = %d; want 1", stalled.closes)
	}
}

func TestWorkspaceRejectsSymbolicLinkRoot(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	var issues []FileIssue
	summary, err := Workspace(context.Background(), link, WorkspaceOptions{
		Options: Options{Query: "find"},
	}, func(batch Batch) { issues = append(issues, batch.Issues...) })
	if err != nil || summary.Entries != 0 || len(issues) != 1 || issues[0].Reason != "symbolic-link" {
		t.Fatalf("workspace followed root symlink: %#v %#v %v", summary, issues, err)
	}
}

func TestWorkspacePreviewIsBoundedAndUnicodeSafe(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "large.txt")
	lineStart := strings.Repeat("甲", 1000)
	lineMiddle := strings.Repeat("🙂", 500)
	lineEnd := lineMiddle + strings.Repeat("乙", 1000)
	hits := 500
	pattern := strings.Repeat("find ", hits)
	source := lineStart + " " + pattern + lineEnd
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	var matches []FileMatch
	summary, err := Workspace(context.Background(), root, WorkspaceOptions{
		Options: Options{Query: "find", CaseSensitive: true},
	}, func(batch Batch) { matches = append(matches, batch.Matches...) })
	if err != nil || summary.Matches != hits {
		t.Fatal(summary, err)
	}
	for _, match := range matches {
		utf16Len := text.UTF16Len(match.Preview)
		if utf16Len > 242 || strings.Count(match.Preview, "find") < 1 {
			t.Fatalf("bad preview %q (UTF-16 length %d)", match.Preview, utf16Len)
		}
	}
}

func TestBoundedPreviewUsesAbsoluteUTF16CoordinatesAndKeepsBothEllipses(t *testing.T) {
	source := "first line\nsecond 甲🙂甲 line that continues for a long time after the emoji boundary "
	source += strings.Repeat("tail ", 120)
	buffer := text.New(source)
	span, err := buffer.LineRange(1)
	if err != nil {
		t.Fatal(err)
	}
	preview := boundedPreview(buffer, 1, span.Start()+(span.End()-span.Start())/2, previewRunes)
	if !strings.HasPrefix(preview, "…") || !strings.HasSuffix(preview, "…") {
		t.Fatalf("both ellipses were not preserved: %q", preview)
	}
	if strings.Contains(preview, "first line") {
		t.Fatalf("preview crossed a physical line: %q", preview)
	}
	if value, err := buffer.SliceUTF16(span.End()-3, span.End()); err != nil || !strings.Contains(preview, value) {
		t.Fatalf("match not retained: preview=%q value=%q err=%v", preview, value, err)
	}
	if text.UTF16Len(preview) > previewRunes+2 {
		t.Fatalf("preview is unbounded: %q (%d UTF-16 units)", preview, text.UTF16Len(preview))
	}
}

func TestBoundedPreviewDoesNotSplitSurrogatePairs(t *testing.T) {
	source := strings.Repeat("甲", 120) + "🙂" + strings.Repeat("乙", 120)
	buffer := text.New(source)
	span, err := buffer.LineRange(0)
	if err != nil {
		t.Fatal(err)
	}
	preview := boundedPreview(buffer, 0, span.Start()+120, previewRunes)
	if strings.ContainsRune(preview, utf8.RuneError) {
		t.Fatalf("preview split UTF-8 or surrogate boundaries: %q", preview)
	}
	if !strings.Contains(preview, "🙂") {
		t.Fatalf("astral match omitted: %q", preview)
	}
}
