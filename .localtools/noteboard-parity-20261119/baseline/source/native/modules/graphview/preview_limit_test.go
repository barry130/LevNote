package graphview

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/drawio"
	"levnote.local/native/formats/mindmap"
	"levnote.local/native/ui/editor"
)

func graphAllocationBytes(t *testing.T, label string, run func()) uint64 {
	t.Helper()
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()
	run()
	elapsed := time.Since(start)
	runtime.ReadMemStats(&after)
	allocated := after.TotalAlloc - before.TotalAlloc
	t.Logf("%s elapsed=%s allocated=%d B allocations=%d", label, elapsed, allocated, after.Mallocs-before.Mallocs)
	return allocated
}

var oversizeGraphCases = []struct {
	name   string
	kind   documentKind
	limit  int
	source string
	err    error
}{
	{"mindmap", kindMindmap, mindmap.DefaultMaxBytes, testMindmap, mindmap.ErrLimitExceeded},
	{"drawio", kindDrawio, drawio.DefaultMaxBytes, testDrawio, drawio.ErrLimitExceeded},
}

func TestOversizeGraphRefusesBeforeCopyAndRecovers(t *testing.T) {
	for _, tc := range oversizeGraphCases {
		t.Run(tc.name, func(t *testing.T) {
			// Allocate fixtures before measuring the guarded path. One String
			// measures the avoidable copy only, not former parsing/layout work.
			source := strings.Repeat(" ", tc.limit+1)
			buffer := text.New(source)
			_ = buffer.SetSelection(11, 4)
			selection, history, version := buffer.Selection(), buffer.History(), buffer.Version()
			baseline := graphAllocationBytes(t, "former whole-source copy only", func() {
				raw := buffer.String()
				runtime.KeepAlive(raw)
			})
			if baseline < uint64(len(source)) {
				t.Fatal("baseline did not observe a source copy")
			}
			reports := 0
			v := newView(tc.kind, editor.Host{Buffer: buffer, Report: func(error) { reports++ }})
			if got := graphAllocationBytes(t, "first refusal", func() { v.ensureSynced(nil) }); got > 64<<10 {
				t.Fatalf("preview copied oversized source: %d", got)
			}
			var output []byte
			var exportErr error
			got := graphAllocationBytes(t, "100 cached warnings/export refusals", func() {
				for i := 0; i < 100; i++ {
					_ = v.Warnings()
					output, exportErr = v.Export("svg")
				}
			})
			if got > 128<<10 || len(output) != 0 || !errors.Is(exportErr, tc.err) || reports != 1 {
				t.Fatalf("cached refusal alloc=%d output=%d reports=%d err=%v", got, len(output), reports, exportErr)
			}
			if v.mind != nil || v.diagram != nil || len(v.layout.Nodes) != 0 || v.labels.layouts != nil || v.nodeIndex != nil || v.indexPage != nil || v.edgeScratch != nil {
				t.Fatal("refusal retained parsed/render state")
			}
			if buffer.Selection() != selection || buffer.History() != history || buffer.Version() != version || buffer.String() != source {
				t.Fatal("refusal changed source state")
			}
			tester := ui.NewTester(v.Draw, 532, 420)
			if !strings.Contains(strings.Join(tester.Texts(), " "), "源码视图") {
				t.Fatal("oversize UI did not offer source guidance")
			}
			// Replacement at the same version must recover before any edit.
			replacement := text.New(tc.source)
			v.host.Buffer = replacement
			tester.Frame()
			if v.syncFailed || v.syncError != nil || v.mind == nil && v.diagram == nil {
				t.Fatal("same-version buffer replacement remained poisoned")
			}
			v.host.Buffer = buffer
			v.ensureSynced(nil)
			_ = buffer.ReplaceUTF16(0, buffer.LenUTF16(), tc.source)
			v.ensureSynced(nil)
			if v.syncFailed || v.syncError != nil || v.mind == nil && v.diagram == nil {
				t.Fatal("shrink did not recover preview")
			}
		})
	}
}

func TestOversizeGraphClearsValidCacheAndBlocksMutationWithoutCopy(t *testing.T) {
	for _, tc := range oversizeGraphCases {
		t.Run(tc.name, func(t *testing.T) {
			buffer := text.New(tc.source)
			v := newView(tc.kind, editor.Host{Buffer: buffer})
			tester := ui.NewTester(v.Draw, 760, 480)
			selected, page := v.selectedID, v.pageIndex
			v.panX, v.panY, v.zoom = 13, 21, .8
			// Append valid whitespace, so the only rejection reason is size.
			_ = buffer.ReplaceUTF16(buffer.LenUTF16(), buffer.LenUTF16(), strings.Repeat(" ", tc.limit+1-buffer.LenBytes()))
			_ = buffer.SetSelection(13, 5)
			selection, history, version := buffer.Selection(), buffer.History(), buffer.Version()
			var err error
			called := false
			got := graphAllocationBytes(t, "stale graph mutation refusal", func() {
				if tc.kind == kindMindmap {
					err = v.applyMindmapMutation("root", func(*mindmap.Document) error { called = true; return nil })
				} else {
					err = v.applyDrawioMutation(0, "a", func(*drawio.Document) (string, error) { called = true; return "a", nil })
				}
			})
			if called || !errors.Is(err, tc.err) || got > 64<<10 {
				t.Fatalf("mutation bypassed guard: callback=%t alloc=%d err=%v", called, got, err)
			}
			tester.Frame()
			if v.frameNodes != 0 || v.mind != nil || v.diagram != nil || len(v.layout.Nodes) != 0 || v.labels.layouts != nil || v.nodeIndex != nil || v.draft != nil {
				t.Fatal("obsolete graph/cache/clean field remained visible")
			}
			if v.selectedID != selected || v.pageIndex != page || v.panX != 13 || v.panY != 21 || v.zoom != .8 || buffer.Selection() != selection || buffer.History() != history || buffer.Version() != version {
				t.Fatal("refusal changed navigation/source state")
			}
			if !buffer.Undo() || buffer.String() != tc.source {
				t.Fatal("source Undo did not restore valid document")
			}
			tester.Frame()
			if v.syncFailed || v.frameNodes == 0 {
				t.Fatal("Undo did not recover normal canvas")
			}
		})
	}
}

func TestOversizeGraphCompositionDraftSurvivesFlushReleaseAndRecovery(t *testing.T) {
	for _, tc := range oversizeGraphCases {
		t.Run(tc.name, func(t *testing.T) {
			buffer := text.New(tc.source)
			v := newView(tc.kind, editor.Host{Buffer: buffer})
			tester := ui.NewTester(v.Draw, 760, 480)
			focusDraft(t, tester)
			tester.Compose("待恢复🙂", 4)
			draft, selected, page := v.draft, v.selectedID, v.pageIndex
			_ = buffer.ReplaceUTF16(buffer.LenUTF16(), buffer.LenUTF16(), strings.Repeat(" ", tc.limit+1-buffer.LenBytes()))
			_ = buffer.SetSelection(13, 5)
			selection, history, version := buffer.Selection(), buffer.History(), buffer.Version()
			var err error
			got := graphAllocationBytes(t, "composition + guarded Flush", func() { err = v.Flush() })
			if got > 256<<10 || !errors.Is(err, tc.err) || v.draft != draft || !v.pendingDraft() {
				t.Fatal("Flush copied source or lost draft", got, err)
			}
			tester.Frame()
			if v.frameNodes != 0 || v.mind != nil || v.diagram != nil || len(v.layout.Nodes) != 0 || !tester.HasText("恢复为新文档") {
				t.Fatal("stale drawable survived or recovery control missing")
			}
			// Match the actual 760x480 app's graph content below its toolbar,
			// tab/mode/banner rows and above the global status bar.
			narrow := ui.NewTester(v.Draw, 532, 285)
			for _, label := range []string{"重试写回", "恢复为新文档"} {
				rect, ok := narrow.Find(label)
				if !ok || rect.W <= 0 || rect.H <= 0 || rect.Y+rect.H > 285 {
					t.Fatal("recovery control outside bounded content", label, rect)
				}
			}
			if output, err := v.Export("svg"); len(output) != 0 || !errors.Is(err, tc.err) {
				t.Fatal("failed draft exported content", len(output), err)
			}
			v.Release()
			if v.draft != draft || v.selectedID != selected || v.pageIndex != page || buffer.Selection() != selection || buffer.History() != history || buffer.Version() != version {
				t.Fatal("Release lost draft/navigation or modified source")
			}
			var recovered string
			v.host.RecoverDraft = func(_, content string) error { recovered = content; return nil }
			if err := v.RecoverDraft(); err != nil || !strings.Contains(recovered, "待恢复🙂") || v.draft != nil || buffer.History() != history || buffer.Selection() != selection {
				t.Fatal("draft recovery failed", err, recovered)
			}
			if !buffer.Undo() || buffer.String() != tc.source {
				t.Fatal("recovery damaged source Undo")
			}
			v.ensureSynced(nil)
			if v.syncFailed {
				t.Fatal(fmt.Sprintf("%s failed to recover after undo", tc.name))
			}
		})
	}
}
