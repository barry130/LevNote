package search

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"levnote.local/native/core/document"
	"levnote.local/native/core/text"
)

type WorkspaceOptions struct {
	Options
	MaxFileBytes int64
	MaxResults   int
	MaxEntries   int
}
type FileMatch struct {
	Path         string
	Line, Column int
	Start, End   int
	Preview      string
}
type FileIssue struct{ Path, Reason, Detail string }
type Batch struct {
	Matches []FileMatch
	Issues  []FileIssue
}
type Summary struct {
	Files, Matches, Skipped int
	Entries, Dirs            int
	Cancelled, Truncated     bool
}

const previewRunes = 240
const defaultMaxWorkspaceEntries = 10000
const directoryReadBatch = 256

// directoryReader opens a stateful directory session. The session must remain
// open between bounded Read calls so the directory cursor can advance.
type directoryReader interface {
	Open(path string) (directoryBatch, error)
}

type directoryBatch interface {
	Read(n int) ([]fs.DirEntry, error)
	Close() error
}

type osDirectoryReader struct{}

func (osDirectoryReader) Open(path string) (directoryBatch, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return &osDirectoryBatch{file: file}, nil
}

type osDirectoryBatch struct {
	file *os.File
}

func (b *osDirectoryBatch) Read(n int) ([]fs.DirEntry, error) {
	return b.file.ReadDir(n)
}

func (b *osDirectoryBatch) Close() error { return b.file.Close() }

type traversal struct {
	reader directoryReader
	// calls is the number of bounded ReadDir calls, not the number of
	// entries seen. Reuse of the same batch is deliberate.
	calls int
}

func sortDirEntries(entries []fs.DirEntry) {
	sort.Slice(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})
}

var errLimit = errors.New("workspace result limit")
var workspaceOpenHook func(path string)

// Workspace streams bounded batches on the calling goroutine. Run it in a
// worker and marshal batches onto the UI thread. Links are never followed.
func Workspace(ctx context.Context, root string, opts WorkspaceOptions, emit func(Batch)) (Summary, error) {
	var summary Summary
	if opts.Query == "" {
		return summary, nil
	}
	re, err := compile(opts.Options)
	if err != nil {
		return summary, err
	}
	if opts.MaxFileBytes <= 0 {
		opts.MaxFileBytes = 4 << 20
	}
	if opts.MaxResults <= 0 {
		opts.MaxResults = 2000
	}
	if opts.MaxEntries <= 0 {
		opts.MaxEntries = defaultMaxWorkspaceEntries
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return summary, err
	}
	batch := Batch{}
	flush := func() {
		if emit != nil && (len(batch.Matches) > 0 || len(batch.Issues) > 0) {
			emit(batch)
		}
		batch = Batch{}
	}
	issue := func(path, reason, detail string) {
		summary.Skipped++
		batch.Issues = append(batch.Issues, FileIssue{path, reason, detail})
		if len(batch.Issues)+len(batch.Matches) >= 64 {
			flush()
		}
	}
	rootInfo, err := os.Lstat(root)
	if err != nil {
		return summary, err
	}
	if rootInfo.Mode()&os.ModeSymlink != 0 {
		issue(root, "symbolic-link", "workspace root must not be a symbolic link")
		flush()
		return summary, nil
	}
	if !rootInfo.IsDir() {
		return summary, &fs.PathError{Op: "open workspace", Path: root, Err: fs.ErrInvalid}
	}
	traverse := traversal{reader: osDirectoryReader{}}
	if workspaceReader != nil {
		traverse.reader = workspaceReader
	}
	err = traverse.walk(ctx, root, opts, &summary, issue, func(path string, entry fs.DirEntry) bool {
		if entry.Type()&fs.ModeSymlink != 0 {
			issue(path, "symbolic-link", "")
			return true
		}
		info, err := entry.Info()
		if err != nil {
			issue(path, "read-failed", err.Error())
			return true
		}
		if !info.Mode().IsRegular() {
			issue(path, "not-regular", "")
			return true
		}
		if workspaceOpenHook != nil {
			workspaceOpenHook(path)
		}
		if info.Size() > opts.MaxFileBytes {
			issue(path, "too-large", "")
			return true
		}
		doc, err := document.OpenLimitedRead(path, "", opts.MaxFileBytes)
		if err != nil {
			reason := "read-failed"
			if errors.Is(err, document.ErrTooLarge) {
				reason = "too-large"
			} else if errors.Is(err, document.ErrBinary) {
				reason = "binary"
			} else if errors.Is(err, document.ErrUnsupportedEncoding) {
				reason = "unknown-encoding"
			} else if errors.Is(err, document.ErrConflict) {
				reason = "changed-while-reading"
			}
			issue(path, reason, err.Error())
			return true
		}
		summary.Files++
		source := doc.Buffer.String()
		remaining := opts.MaxResults - summary.Matches
		matches, findErr := findCompiled(source, opts.Options, re, remaining)
		if findErr != nil && !errors.Is(findErr, ErrTooManyMatches) {
			issue(path, "search-failed", findErr.Error())
			return true
		}
		previewLine := -1
		preview := ""
		for _, m := range matches {
			line, column, _ := doc.Buffer.LineColumn(m.Start)
			if line != previewLine {
				previewLine = line
				preview = boundedPreview(doc.Buffer, line, m.Start, previewRunes)
			}
			batch.Matches = append(batch.Matches, FileMatch{path, line + 1, column + 1, m.Start, m.End, preview})
			summary.Matches++
			if len(batch.Matches)+len(batch.Issues) >= 64 {
				flush()
			}
		}
		if errors.Is(findErr, ErrTooManyMatches) || summary.Matches >= opts.MaxResults {
			summary.Truncated = true
			return false
		}
		return true
	})
	flush()
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		summary.Cancelled = true
		return summary, err
	}
	if errors.Is(err, errLimit) {
		return summary, nil
	}
	return summary, err
}

var workspaceReader directoryReader

func (t *traversal) walk(ctx context.Context, root string, opts WorkspaceOptions, summary *Summary,
	issue func(path, reason, detail string), visit func(path string, entry fs.DirEntry) bool,
) error {
	type directoryWork struct {
		path string
	}
	stack := []directoryWork{{path: root}}
	var currentPath string
	var current directoryBatch
	defer func() {
		if current != nil {
			_ = current.Close()
		}
	}()
	for len(stack) > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		work := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if current == nil {
			currentPath = work.path
			handle, err := t.reader.Open(work.path)
			if err != nil {
				issue(work.path, "read-failed", err.Error())
				continue
			}
			current = handle
		} else if currentPath != work.path {
			return errors.New("search traversal opened an unexpected directory")
		}
		t.calls++
		entries, readErr := current.Read(directoryReadBatch)
		if len(entries) > 0 {
			sortDirEntries(entries)
		}
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return err
			}
			if summary.Entries >= opts.MaxEntries {
				summary.Truncated = true
				return errLimit
			}
			summary.Entries++
			path := filepath.Join(work.path, entry.Name())
			if entry.IsDir() {
				summary.Dirs++
				if path != root {
					switch entry.Name() {
					case ".git", "node_modules", "target", "dist", "build-cache", "tools":
						issue(path, "excluded-directory", "")
						continue
					}
				}
				stack = append(stack, directoryWork{path: path})
				continue
			}
			if !visit(path, entry) {
				return errLimit
			}
		}
		if readErr != nil {
			if !errors.Is(readErr, io.EOF) {
				issue(work.path, "read-failed", readErr.Error())
			}
			_ = current.Close()
			current = nil
			continue
		}
		if len(entries) == 0 {
			issue(work.path, "read-stalled", "directory read made no progress")
			_ = current.Close()
			current = nil
			continue
		}
		stack = append(stack, work)
	}
	return nil
}

func boundedPreview(buffer *text.Buffer, line, matchStart, limit int) string {
	span, err := buffer.LineRange(line)
	if err != nil {
		return ""
	}
	start := max(span.Start(), min(matchStart-limit/2, span.End()-limit))
	end := min(span.End(), start+limit)
	start = max(span.Start(), end-limit)
	start = alignUTF16Start(buffer, start)
	end = alignUTF16End(buffer, end)
	preview, err := buffer.SliceUTF16(start, end)
	if err != nil {
		return ""
	}
	leading := span.Start() < start
	trailing := end < span.End()
	if leading {
		preview = "…" + preview
	}
	if trailing {
		preview += "…"
	}
	return preview
}

func alignUTF16Start(buffer *text.Buffer, offset int) int {
	if _, err := buffer.SliceUTF16(offset, offset); err != nil {
		if _, err := buffer.SliceUTF16(offset-1, offset-1); err == nil {
			return offset - 1
		}
	}
	return offset
}

func alignUTF16End(buffer *text.Buffer, offset int) int {
	if _, err := buffer.SliceUTF16(offset, offset); err != nil {
		if _, err := buffer.SliceUTF16(offset+1, offset+1); err == nil {
			return offset + 1
		}
	}
	return offset
}
