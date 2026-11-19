package markdownview

import (
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/markdown"
	"levnote.local/native/ui/editor"
)

func markdownAllocationBytes(t *testing.T, name string, run func()) uint64 {
	t.Helper()
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()
	run()
	elapsed := time.Since(start)
	runtime.ReadMemStats(&after)
	allocated := after.TotalAlloc - before.TotalAlloc
	t.Logf("%s elapsed=%s allocated=%d B allocations=%d", name, elapsed, allocated, after.Mallocs-before.Mallocs)
	return allocated
}

func TestOversizeMarkdownRefusalBeforeCopyAndCacheRecovery(t *testing.T) {
	// Fixture construction, source preservation checks and UI creation stay
	// outside the measured regions. The baseline isolates the removed copy.
	source := strings.Repeat("x", markdown.MaxParseBytes+1)
	buffer := text.New(source)
	_ = buffer.SetSelection(7, 3)
	selection, history, version := buffer.Selection(), buffer.History(), buffer.Version()
	baseline := markdownAllocationBytes(t, "former whole-source copy only", func() {
		raw := buffer.String()
		runtime.KeepAlive(raw)
	})
	if baseline < uint64(len(source)) {
		t.Fatal("baseline did not observe the whole-buffer copy")
	}
	reports := 0
	v := Adapter().Create(editor.Host{Buffer: buffer, Report: func(error) { reports++ }}).(*View)
	if got := markdownAllocationBytes(t, "first refusal", v.refresh); got > 64<<10 {
		t.Fatalf("oversize preview allocated %d bytes before rejection", got)
	}
	var exportErr error
	var output []byte
	got := markdownAllocationBytes(t, "100 cached warnings/export refusals", func() {
		for i := 0; i < 100; i++ {
			_ = v.Warnings()
			output, exportErr = v.Export("html")
		}
	})
	if got > 128<<10 || len(output) != 0 || !errors.Is(exportErr, markdown.ErrPartialDocument) {
		t.Fatalf("cached refusal: allocation=%d output=%d err=%v", got, len(output), exportErr)
	}
	if reports != 1 || !v.doc.Partial || v.doc.Raw() != "" || len(v.rows) != 0 || len(v.doc.Blocks) != 0 {
		t.Fatal("refusal re-reported or retained source/preview")
	}
	if buffer.Version() != version || buffer.Selection() != selection || buffer.History() != history || buffer.String() != source {
		t.Fatal("refusal mutated authoritative source")
	}
	tester := ui.NewTester(v.Draw, 265, 360)
	if !tester.HasText(oversizeSourceGuidance) || tester.HasText("添加段落") {
		t.Fatal("refusal omitted guidance or offered incomplete block edits", tester.Texts())
	}
	// Buffer identity, not version alone, must invalidate the refused cache.
	replacement := text.New("# new buffer\n")
	if replacement.Version() != version {
		t.Fatal("fixture must use matching versions")
	}
	v.host.Buffer = replacement
	tester.Frame()
	if !tester.HasText("new buffer") || v.previewError != nil || v.doc.Partial {
		t.Fatal("same-version replacement remained poisoned")
	}
	v.host.Buffer = buffer
	v.refresh()
	if err := buffer.ReplaceUTF16(0, buffer.LenUTF16(), "# shrink\n"); err != nil {
		t.Fatal(err)
	}
	v.refresh()
	if v.previewError != nil || v.doc.Partial || v.doc.Raw() != "# shrink\n" {
		t.Fatal("shrinking did not recover")
	}
	if !buffer.Undo() {
		t.Fatal("refusal/recovery damaged Undo")
	}
	v.refresh()
	if !v.doc.Partial || buffer.Selection() != selection || buffer.String() != source {
		t.Fatal("Undo did not restore original authoritative state")
	}
}

func TestOversizeMarkdownDropsStaleRowsBeforeBegin(t *testing.T) {
	buffer := text.New("# original\n")
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	v.refresh()
	v.outlineCurrent, v.outlineCaret = "old-outline-id", 3
	staleBlock := v.doc.Blocks[0]
	_ = buffer.ReplaceUTF16(0, 0, strings.Repeat(" ", markdown.MaxParseBytes))
	selection, history, version := buffer.Selection(), buffer.History(), buffer.Version()
	if got := markdownAllocationBytes(t, "stale double-click guard", func() { v.begin(0, staleBlock) }); got > 64<<10 {
		t.Fatalf("stale begin copied large prefix/source: %d", got)
	}
	if v.draft != nil || v.doc.Raw() != "" || len(v.rows) != 0 || v.outline != nil || v.outlineCurrent != "" || v.outlineCaret != 0 {
		t.Fatal("stale block entered or retained old preview")
	}
	if buffer.Selection() != selection || buffer.History() != history || buffer.Version() != version {
		t.Fatal("rejected stale begin changed source state")
	}
}

func TestOversizeMarkdownCompositionSafeRangeFlushAndConflictRecovery(t *testing.T) {
	for _, conflict := range []bool{false, true} {
		name := "safe-range"
		if conflict {
			name = "conflict"
		}
		t.Run(name, func(t *testing.T) {
			buffer := text.New("# 标题\n")
			v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
			tester := ui.NewTester(v.Draw, 265, 360)
			beginDraft(t, v, tester, "标题")
			tester.Command("selectAll")
			tester.Compose("本地🙂", 3)
			draft := v.draft
			if conflict {
				_ = buffer.ReplaceUTF16(2, 4, "外部")
			}
			_ = buffer.ReplaceUTF16(buffer.LenUTF16(), buffer.LenUTF16(), strings.Repeat("x", markdown.MaxParseBytes))
			before := buffer.History()
			tester.Frame()
			if v.draft != draft || v.doc.Raw() != "" || len(v.rows) != 0 || !v.doc.Partial || !tester.HasText(oversizeSourceGuidance) {
				t.Fatal("oversize refresh lost draft or retained stale preview")
			}
			output, err := v.Export("html")
			if len(output) != 0 {
				t.Fatal("oversize export produced output")
			}
			prefix, _ := buffer.SliceUTF16(0, 7)
			if !conflict {
				if !errors.Is(err, markdown.ErrPartialDocument) || !strings.HasPrefix(prefix, "# 本地🙂") || v.pendingDraft() || buffer.History().UndoDepth != before.UndoDepth+1 {
					t.Fatalf("safe field flush was blocked: %q %v", prefix, err)
				}
				if !buffer.Undo() {
					t.Fatal("safe Flush not undoable")
				}
				prefix, _ = buffer.SliceUTF16(0, 5)
				if prefix != "# 标题\n" {
					t.Fatal("Undo lost previous field", prefix)
				}
				return
			}
			if !errors.Is(err, errDraftConflict) || !v.pendingDraft() || !strings.HasPrefix(prefix, "# 外部\n") || buffer.History() != before {
				t.Fatal("conflict did not retain independent draft/source", err)
			}
			v.Release()
			if v.draft != draft || v.doc.Raw() != "" || len(v.rows) != 0 {
				t.Fatal("Release discarded draft or retained render state")
			}
			var recovered string
			v.host.RecoverDraft = func(_, content string) error { recovered = content; return nil }
			if err := v.RecoverDraft(); err != nil || recovered != "本地🙂" || v.draft != nil || buffer.History() != before {
				t.Fatal("oversize conflict could not recover", err, recovered)
			}
		})
	}
}

func TestOversizeMarkdownShrinksWhileIndependentCompositionStaysOpen(t *testing.T) {
	buffer := text.New("# 标题\n\n正文\n")
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	tester := ui.NewTester(v.Draw, 265, 360)
	beginDraft(t, v, tester, "标题")
	tester.Command("selectAll")
	tester.Compose("未完成🙂", 4)
	draft := v.draft
	_ = buffer.ReplaceUTF16(buffer.LenUTF16(), buffer.LenUTF16(), strings.Repeat("x", markdown.MaxParseBytes))
	tester.Frame()
	if v.doc.Raw() != "" || !v.pendingDraft() {
		t.Fatal("oversize did not clear source cache/keep draft")
	}
	if !buffer.Undo() {
		t.Fatal("source append was not undoable")
	}
	tester.Frame()
	if v.previewError != nil || v.doc.Partial || v.doc.Raw() != "# 标题\n\n正文\n" || len(v.rows) == 0 || v.draft != draft || !v.pendingDraft() {
		t.Fatal("in-budget preview did not resume with independent draft still open")
	}
	if err := v.Flush(); err != nil {
		t.Fatal("restored range could not flush", err)
	}
	if buffer.String() != "# 未完成🙂\n\n正文\n" {
		t.Fatal("preview recovery rebound or changed the draft", buffer.String())
	}
}
