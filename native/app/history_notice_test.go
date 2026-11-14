package app

import (
	"path/filepath"
	"strings"
	"testing"

	"levnote.local/native/core/document"
	"levnote.local/native/core/text"
)

// truncateHistory forces the buffer's retained undo history past its budget so
// that History().Truncated becomes true, using only the public text seam.
func truncateHistory(t *testing.T, buffer *text.Buffer) {
	t.Helper()
	buffer.SetHistoryLimits(2, 1<<20)
	for _, s := range []string{"x", "y", "z"} {
		if err := buffer.ReplaceUTF16(0, 0, s); err != nil {
			t.Fatal(err)
		}
	}
	if !buffer.History().Truncated {
		t.Fatalf("fixture did not reach the history budget: %#v", buffer.History())
	}
}

func historyNoticeCount(a *App, tabName string) int {
	n := 0
	for _, issue := range a.Issues {
		if strings.Contains(issue, historyBudgetNotice) && strings.Contains(issue, tabName) {
			n++
		}
	}
	return n
}

// A tab switch is a normal frame boundary: the disclosure is about the buffer's
// budget, so re-showing the tab must not re-announce the same truncation.
func TestHistoryBudgetNoticeDoesNotRepeatAfterTabSwitch(t *testing.T) {
	root := t.TempDir()
	a := newTestApp(t, filepath.Join(root, "config"))

	doc := document.New("budget")
	truncateHistory(t, doc.Buffer)
	a.appendTab(doc, "budget.txt", "", false, false)
	a.appendTab(document.New("other"), "other.txt", "", false, false)

	a.consumeHistoryNotices()
	if got := historyNoticeCount(a, "budget.txt"); got != 1 {
		t.Fatalf("first observation reported %d notices, want 1: %#v", got, a.Issues)
	}

	// Switch away and back through the real tab-selection entry point.
	if !a.Select(1) || !a.Select(0) {
		t.Fatal("tab switch was refused")
	}
	a.consumeHistoryNotices()
	if got := historyNoticeCount(a, "budget.txt"); got != 1 {
		t.Fatalf("tab switch repeated the notice: %d: %#v", got, a.Issues)
	}
}

// ResetHistory clears the truncation flag; once the app has actually observed a
// non-truncated history, a later truncation is new information and must be said.
func TestHistoryBudgetNoticeRearmsAfterObservedReset(t *testing.T) {
	root := t.TempDir()
	a := newTestApp(t, filepath.Join(root, "config"))

	doc := document.New("budget")
	truncateHistory(t, doc.Buffer)
	a.appendTab(doc, "budget.txt", "", false, false)

	a.consumeHistoryNotices()
	if got := historyNoticeCount(a, "budget.txt"); got != 1 {
		t.Fatalf("first observation reported %d notices, want 1", got)
	}

	doc.Buffer.ResetHistory()
	a.consumeHistoryNotices()
	if got := historyNoticeCount(a, "budget.txt"); got != 1 {
		t.Fatalf("reset alone must not add a notice: %d", got)
	}

	truncateHistory(t, doc.Buffer)
	a.consumeHistoryNotices()
	if got := historyNoticeCount(a, "budget.txt"); got != 2 {
		t.Fatalf("re-truncation after an observed reset was silent: %d", got)
	}
}

// A different buffer is a different history, even inside the same tab (reload or
// revert). The latch belongs to the buffer, not the tab identity.
func TestHistoryBudgetNoticeTracksBufferIdentityNotTabIdentity(t *testing.T) {
	root := t.TempDir()
	a := newTestApp(t, filepath.Join(root, "config"))

	doc := document.New("budget")
	truncateHistory(t, doc.Buffer)
	tab := a.appendTab(doc, "budget.txt", "", false, false)
	a.consumeHistoryNotices()
	if got := historyNoticeCount(a, "budget.txt"); got != 1 {
		t.Fatalf("first observation reported %d notices, want 1", got)
	}

	// Same tab, brand-new buffer: its history is its own and is truncated.
	replacement := text.New("budget")
	truncateHistory(t, replacement)
	tab.Doc.Buffer = replacement
	a.consumeHistoryNotices()
	if got := historyNoticeCount(a, "budget.txt"); got != 2 {
		t.Fatalf("a replaced buffer did not get its own disclosure: %d", got)
	}

	// Re-observing that same buffer stays quiet.
	a.consumeHistoryNotices()
	if got := historyNoticeCount(a, "budget.txt"); got != 2 {
		t.Fatalf("the same buffer was announced twice: %d", got)
	}
}

// A tab with no text buffer has no history to disclose, and must not panic.
func TestHistoryBudgetNoticeIgnoresTabsWithoutTextBuffer(t *testing.T) {
	root := t.TempDir()
	a := newTestApp(t, filepath.Join(root, "config"))

	binary := a.appendTab(document.New("bin"), "bin.bin", "", false, false)
	binary.BinaryPath = "C:\\images\\bin.bin"
	binary.Doc.ReadOnly = true

	a.consumeHistoryNotices()
	if got := historyNoticeCount(a, "bin.bin"); got != 0 {
		t.Fatalf("binary tab produced a history notice: %#v", a.Issues)
	}
}
