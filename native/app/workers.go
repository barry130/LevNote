package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"levnote.local/native/core/document"
)

// coalescedWorker runs one task at a time and keeps only the newest pending
// task. Bursts replace pending work instead of creating a goroutine per event.
type coalescedWorker[T any] struct {
	execute func(context.Context, T)

	mu      sync.Mutex
	pending *T
	running bool
	stopped bool
	wake    chan struct{}
	done    chan struct{}
	cancel  context.CancelFunc
}

func newCoalescedWorker[T any](execute func(context.Context, T)) *coalescedWorker[T] {
	ctx, cancel := context.WithCancel(context.Background())
	w := &coalescedWorker[T]{
		execute: execute,
		wake:    make(chan struct{}, 1),
		done:    make(chan struct{}),
		cancel:  cancel,
	}
	go w.run(ctx)
	return w
}

func (w *coalescedWorker[T]) submit(task T) bool {
	if w == nil {
		return false
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.stopped {
		return false
	}
	value := task
	w.pending = &value
	if !w.running {
		w.running = true
	}
	select {
	case w.wake <- struct{}{}:
	default:
	}
	return true
}

func (w *coalescedWorker[T]) run(ctx context.Context) {
	defer close(w.done)
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.wake:
		}
		for ctx.Err() == nil {
			w.mu.Lock()
			if w.pending == nil {
				w.running = false
				w.mu.Unlock()
				break
			}
			task := *w.pending
			w.pending = nil
			w.mu.Unlock()

			w.execute(ctx, task)
		}
	}
}

func (w *coalescedWorker[T]) close() {
	if w == nil {
		return
	}
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return
	}
	w.stopped = true
	w.pending = nil
	w.mu.Unlock()
	w.cancel()
	<-w.done
}

type treeScanTask struct {
	root       string
	expanded   map[string]bool
	generation uint64
}

type treeScanResult struct {
	entries   []Entry
	truncated bool
	issues    []string
}

type diskCheckTarget struct {
	tabID       int
	path        string
	token       document.DiskToken
	binaryToken document.BinaryToken
}

type diskCheckTask struct {
	generation uint64
	targets    []diskCheckTarget
}

type diskCheckResult struct {
	tabID       int
	path        string
	token       document.DiskToken
	binaryToken document.BinaryToken
	changed     bool
	missing     bool
	invalid     bool
	errText     string
}

func checkDiskTask(ctx context.Context, task diskCheckTask) []diskCheckResult {
	results := make([]diskCheckResult, 0, len(task.targets))
	for _, target := range task.targets {
		if ctx.Err() != nil {
			return nil
		}
		check, err := target.token.CheckDisk(ctx)
		if target.binaryToken.Valid() {
			check, err = target.binaryToken.CheckDisk(ctx)
		}
		result := diskCheckResult{
			tabID: target.tabID, path: target.path, token: target.token,
			binaryToken: target.binaryToken,
			changed:     check.Changed, missing: check.Missing, invalid: check.Invalid,
		}
		if err != nil {
			result.errText = err.Error()
		}
		results = append(results, result)
	}
	return results
}

func (a *App) invalidateDiskChecks() {
	a.diskGeneration++
}

func (a *App) scheduleDiskCheck() {
	if a.shuttingDown.Load() {
		return
	}
	a.invalidateDiskChecks()
	generation := a.diskGeneration
	targets := make([]diskCheckTarget, 0, len(a.Tabs))
	for _, t := range a.Tabs {
		if t != nil && t.Binary != nil {
			token := t.Binary.Token()
			targets = append(targets, diskCheckTarget{tabID: t.ID, path: token.Path(), binaryToken: token})
			continue
		}
		if t == nil || t.ByteInspector || t.Doc == nil || t.Doc.IsNew {
			continue
		}
		token := t.Doc.DiskToken()
		if !token.Valid() {
			continue
		}
		targets = append(targets, diskCheckTarget{tabID: t.ID, path: token.Path(), token: token})
	}
	if len(targets) == 0 || a.diskWorker == nil {
		return
	}
	a.diskWorker.submit(diskCheckTask{generation: generation, targets: targets})
}

func (a *App) applyDiskCheck(task diskCheckTask, results []diskCheckResult) bool {
	if a.shuttingDown.Load() || task.generation != a.diskGeneration {
		return false
	}
	tabs := make(map[int]*Tab, len(a.Tabs))
	for _, t := range a.Tabs {
		if t != nil {
			tabs[t.ID] = t
		}
	}
	for _, result := range results {
		t := tabs[result.tabID]
		if t == nil {
			continue
		}
		if result.binaryToken.Valid() {
			if t.Binary == nil || t.path() != result.path || !t.Binary.SameBaseline(result.binaryToken) {
				continue
			}
		} else if t.ByteInspector || t.Doc == nil || t.Doc.IsNew ||
			t.Doc.Path != result.path || !t.Doc.SameBaseline(result.token) {
			continue
		}
		if result.errText != "" {
			a.report(fmt.Errorf("%s 的外部改动检查失败：%s", t.Name, result.errText))
			continue
		}
		if result.invalid {
			a.report(fmt.Errorf("%s 的外部改动状态无法确认；当前内容仍保留，请重新打开或另存副本", t.Name))
			continue
		}
		if !result.changed {
			continue
		}
		if !t.Conflict {
			t.Conflict = true
			if result.missing {
				a.report(fmt.Errorf("%s 对应的磁盘文件已不存在；当前内容仍保留，请另存副本", t.Name))
			} else {
				a.report(fmt.Errorf("%s 已在外部改变，请重新载入或另存副本", t.Name))
			}
		}
	}
	return true
}

func scanTree(ctx context.Context, root string, expanded map[string]bool) treeScanResult {
	result := treeScanResult{}
	var walk func(string, int)
	walk = func(dir string, depth int) {
		if ctx.Err() != nil || depth > treeDepthLimit || len(result.entries) >= treeEntryLimit {
			result.truncated = true
			return
		}
		remaining := treeEntryLimit - len(result.entries)
		if remaining <= 0 {
			result.truncated = true
			return
		}
		directoryLimit := min(remaining, treeDirectoryEntryLimit)
		entries, truncated, err := readDirectoryBounded(dir, directoryLimit)
		if err != nil {
			result.issues = append(result.issues, fmt.Sprintf("%s：%v", dir, err))
		}
		if truncated {
			result.truncated = true
		}
		sort.SliceStable(entries, func(i, j int) bool {
			return entries[i].Name() < entries[j].Name()
		})
		for _, isDir := range []bool{true, false} {
			for _, entry := range entries {
				if ctx.Err() != nil {
					result.truncated = true
					return
				}
				if entry.IsDir() != isDir || entry.Name() == ".git" {
					continue
				}
				if len(result.entries) >= treeEntryLimit {
					result.truncated = true
					return
				}
				path := filepath.Join(dir, entry.Name())
				result.entries = append(result.entries, Entry{
					Path: path, Name: entry.Name(), Directory: entry.IsDir(), Depth: depth,
				})
				if entry.IsDir() && expanded[path] {
					walk(path, depth+1)
				}
			}
		}
		if err != nil || ctx.Err() != nil {
			result.truncated = result.truncated || ctx.Err() != nil
			return
		}
	}
	if root != "" {
		walk(root, 0)
	}
	return result
}

func (a *App) applyTreeScan(task treeScanTask, result treeScanResult) bool {
	if a.shuttingDown.Load() || task.generation != a.treeGeneration || task.root != a.Root {
		return false
	}
	a.Entries = result.entries
	a.TreeTruncated = result.truncated
	a.TreeIssues = append([]string(nil), result.issues...)
	for _, issue := range result.issues {
		a.report(fmt.Errorf("刷新文件树：%s", issue))
	}
	return true
}

func (a *App) refreshTreeNow() {
	a.treeGeneration++
	root := a.Root
	a.resetTreeIdentity(root)
	expanded := make(map[string]bool, len(a.Expanded))
	for path, open := range a.Expanded {
		if open {
			expanded[path] = true
		}
	}
	task := treeScanTask{root: root, expanded: expanded, generation: a.treeGeneration}
	a.applyTreeScan(task, scanTree(context.Background(), root, expanded))
}

type directoryReader interface {
	ReadDir(int) ([]os.DirEntry, error)
}

func readDirectoryBounded(dir string, limit int) ([]os.DirEntry, bool, error) {
	handle, err := os.Open(dir)
	if err != nil {
		return nil, false, err
	}
	defer handle.Close()
	return readDirectoryEntries(handle, limit)
}

// readDirectoryEntries asks for one extra entry so an exactly-full directory
// is distinguishable from one that has more entries than could be retained.
func readDirectoryEntries(handle directoryReader, limit int) ([]os.DirEntry, bool, error) {
	entries, err := handle.ReadDir(limit + 1)
	if err != nil && !errors.Is(err, io.EOF) {
		return entries, false, err
	}
	if len(entries) > limit {
		return entries[:limit], true, nil
	}
	return entries, false, nil
}
