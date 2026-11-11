//go:build ignore

// Run explicitly with scripts/go.ps1 run ./scripts/qa_probe.go. This is a
// read-only, bounded cross-package acceptance probe; files use a temp folder.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"levnote.local/native/core/document"
	"levnote.local/native/core/session"
	"levnote.local/native/formats/markdown"
)

func measure(label string, run func()) {
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()
	run()
	elapsed := time.Since(start)
	runtime.ReadMemStats(&after)
	fmt.Printf("%s: elapsed=%s allocated=%d B allocs=%d\n", label, elapsed, after.TotalAlloc-before.TotalAlloc, after.Mallocs-before.Mallocs)
}

func main() {
	for _, tc := range []struct{ name, value string }{
		{"64KiB newlines", strings.Repeat("\n", 64<<10)},
		{"256KiB newlines", strings.Repeat("\n", 256<<10)},
		{"64KiB headings", strings.Repeat("# a\n", 16<<10)},
		{"256KiB headings", strings.Repeat("# a\n", 64<<10)},
	} {
		measure("Markdown "+tc.name, func() {
			doc := markdown.Parse(tc.value)
			fmt.Printf("  blocks=%d warnings=%d exact=%v\n", len(doc.Blocks), len(doc.Warnings), doc.Raw() == tc.value)
		})
	}
	for _, tail := range []int{0, 1 << 20} {
		value := strings.Repeat("[", 32) + strings.Repeat("x", 65536) + strings.Repeat("`", tail)
		measure(fmt.Sprintf("Inline exhausted then %d-byte backtick run", tail), func() {
			doc := markdown.Parse(value)
			budget, afterBudget := -1, false
			for _, warning := range doc.Warnings {
				if warning.Code == "scan-budget" {
					budget = warning.Range.Start
				} else if budget >= 0 && warning.Code == "unmatched-code" {
					afterBudget = true
				}
			}
			fmt.Printf("  budget=%v warnings=%d continued-code-scan=%v\n", budget >= 0, len(doc.Warnings), afterBudget)
		})
	}

	folder, err := os.MkdirTemp("", "levnote-qa-")
	if err != nil {
		panic(err)
	}
	path := filepath.Join(folder, "recovery.json")
	defer os.Remove(folder)
	defer os.Remove(path)
	_ = session.Save(path, session.Snapshot{})
	original, _ := os.ReadFile(path)
	value := strings.Repeat("\x00", 3<<20)
	snapshot := session.Snapshot{Tabs: []session.Tab{{Draft: &value}}}
	measure("Session rejects 3MiB NUL JSON expansion", func() {
		err := session.Save(path, snapshot)
		after, _ := os.ReadFile(path)
		fmt.Printf("  err=%v old-preserved=%v\n", err, string(after) == string(original))
	})
	largeDoc := document.New(value)
	measure("Builder rejects 3MiB NUL before capture", func() {
		builder := session.NewSnapshotBuilder("")
		err := builder.AddDocument(largeDoc, "controls", 0)
		fmt.Printf("  err=%v\n", err)
	})
	poisoned := session.NewSnapshotBuilder("")
	doc := document.New(strings.Repeat("x", 1<<20))
	doc.Encoding = "invalid"
	for i := 0; i < 14; i++ {
		_ = poisoned.AddDocument(doc, "invalid", 0)
	}
	doc.Encoding = document.UTF8
	poisonErr := poisoned.AddDocument(doc, "corrected", 0)
	freshErr := session.NewSnapshotBuilder("").AddDocument(doc, "corrected", 0)
	fmt.Printf("Builder retry after14 invalid captures: reused=%v fresh=%v\n", poisonErr, freshErr)
	func() {
		defer func() { fmt.Printf("Builder nil doc panic=%v\n", recover()) }()
		_ = session.NewSnapshotBuilder("").AddDocument(nil, "nil", 0)
	}()

	metadata := session.Snapshot{Preferences: session.Preferences{Theme: strings.Repeat("x", session.MaxBytes+1)}}
	measure("Session oversized Preferences.Theme preflight", func() {
		err := session.Save(path, metadata)
		fmt.Printf("  err=%v rejected-before-marshal=%v\n", err, err != nil && strings.Contains(err.Error(), "before JSON"))
	})
}
