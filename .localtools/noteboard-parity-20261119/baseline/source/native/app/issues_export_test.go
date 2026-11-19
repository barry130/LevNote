package app

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"

	"levnote.local/native/core/document"
	"levnote.local/native/core/search"
)

// The snapshot is the only channel that hands the run's state to somewhere
// outside the app, so it must read on its own: every family keeps its lead
// line, every bound keeps its omission count, and the current tab names which
// document the warnings belong to.
func TestIssuesSnapshotCarriesEveryFamilyAndOmission(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	tab := a.appendTab(document.New("notes"), "notes.md", "", false, false)
	if !a.Select(0) {
		t.Fatal("fixture tab was not activated")
	}
	tab.ViewWarnings = []string{"图片引用缺失", "表格列数不一致"}
	a.SearchIssues = []search.FileIssue{
		{Path: "skip-a.bin", Reason: "binary", Detail: "not a text document"},
		{Path: "skip-b.bin", Reason: "binary", Detail: ""},
	}
	a.SearchIssuesDropped = 3
	a.report(fmt.Errorf("%s：%s", "notes.md", historyBudgetNotice))
	a.report(errors.New("保存 notes.md 失败：磁盘已满"))
	a.IssuesDropped = 7

	snapshot := a.issuesSnapshot()
	for _, want := range []string{
		"当前标签：notes.md",
		fmt.Sprintf("问题清单只保留本次运行最近 %d 条（当前 %d 条）。另有 %d 条较早记录未保留。",
			issueHistoryLimit, len(a.Issues), 7),
		"当前格式的提示",
		"图片引用缺失",
		"表格列数不一致",
		"工作区搜索跳过项（显示 2 项，另有 3 项未保留）",
		"skip-a.bin · binary · not a text document",
		"skip-b.bin · binary\n",
		"问题记录",
		"notes.md：" + historyBudgetNotice,
		"保存 notes.md 失败：磁盘已满",
	} {
		if !strings.Contains(snapshot, want) {
			t.Fatalf("snapshot lost %q:\n%s", want, snapshot)
		}
	}
}

// The snapshot copies exactly the retained window: dropped records stay
// dropped, and the omission count is the honest line about what is missing.
func TestIssuesSnapshotStaysBoundedAndDisclosesOmissions(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	for i := 0; i < issueHistoryLimit+7; i++ {
		a.report(fmt.Errorf("issue-%03d", i))
	}
	snapshot := a.issuesSnapshot()
	if strings.Contains(snapshot, "issue-006") {
		t.Fatal("snapshot resurrected a dropped record")
	}
	if !strings.Contains(snapshot, "issue-007") || !strings.Contains(snapshot, "issue-206") {
		t.Fatalf("snapshot lost the retained window edges:\n%s", snapshot)
	}
	if !strings.Contains(snapshot, fmt.Sprintf("另有 %d 条较早记录未保留。", 7)) {
		t.Fatalf("snapshot hid the omission count:\n%s", snapshot)
	}
}

// With nothing to show, the snapshot stays minimal but never empty: the header
// still states the bound, and absent families add no lead lines.
func TestIssuesSnapshotWithoutFamiliesStaysMinimal(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	snapshot := a.issuesSnapshot()
	for _, want := range []string{
		"当前标签：无",
		fmt.Sprintf("问题清单只保留本次运行最近 %d 条（当前 0 条）。", issueHistoryLimit),
	} {
		if !strings.Contains(snapshot, want) {
			t.Fatalf("snapshot lost %q:\n%s", want, snapshot)
		}
	}
	for _, absent := range []string{"当前格式的提示", "工作区搜索跳过项", "问题记录", "未保留"} {
		if strings.Contains(snapshot, absent) {
			t.Fatalf("snapshot invented %q:\n%s", absent, snapshot)
		}
	}
}

// A successful copy goes through the real seam exactly once, and success is not
// logged as an issue — only the visible message confirms it.
func TestCopyIssuesWritesSnapshotThroughClipboardSeam(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.report(errors.New("保存失败示例"))
	var captured string
	writes := 0
	write := func(text string) error {
		writes++
		captured = text
		return nil
	}
	a.copyIssues(write)
	if writes != 1 {
		t.Fatalf("clipboard writes = %d, want exactly 1", writes)
	}
	if captured != a.issuesSnapshot() {
		t.Fatal("clipboard received something other than the current snapshot")
	}
	if a.Message != "问题清单已复制到剪贴板。" {
		t.Fatalf("success message = %q", a.Message)
	}
	if len(a.Issues) != 1 {
		t.Fatalf("success was logged as an issue: %#v", a.Issues)
	}
}

// A failed copy must have a face: the failure enters the issues channel with
// the underlying error, the seam is never retried, and the next snapshot still
// carries the disclosure — the report never re-enters the clipboard, so a
// failing clipboard cannot recurse.
func TestCopyIssuesDisclosesClipboardFailureWithoutRecursion(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	calls := 0
	write := func(text string) error {
		calls++
		return errors.New("剪贴板被其他程序锁定")
	}
	a.copyIssues(write)
	if calls != 1 {
		t.Fatalf("clipboard attempts = %d, want exactly 1", calls)
	}
	if len(a.Issues) != 1 {
		t.Fatalf("clipboard failure left no trace: %#v", a.Issues)
	}
	if !strings.Contains(a.Issues[0], "复制问题清单失败") ||
		!strings.Contains(a.Issues[0], "剪贴板被其他程序锁定") {
		t.Fatalf("failure entry = %q", a.Issues[0])
	}
	if !strings.Contains(a.Message, "复制问题清单失败") {
		t.Fatalf("failure message = %q", a.Message)
	}
	if !strings.Contains(a.issuesSnapshot(), "复制问题清单失败") {
		t.Fatal("the failure is hidden from the next snapshot")
	}
}

// A missing clipboard seam must fail loudly; treating it as success would be
// the silent-data-loss failure mode this app exists to avoid.
func TestCopyIssuesRejectsMissingClipboardLoudly(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.copyIssues(nil)
	if len(a.Issues) != 1 || !strings.Contains(a.Issues[0], "复制问题清单失败") {
		t.Fatalf("missing clipboard was not disclosed: %#v", a.Issues)
	}
}

// The copy is wired to one explicit button. Rendering alone never touches the
// clipboard, and the headless click path surfaces the real seam's failure
// through the issues channel instead of vanishing.
func TestIssuesCopyAllIsExplicitClickOnly(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.report(errors.New("示例问题"))
	a.ShowIssues = true
	base := len(a.Issues)
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	if len(a.Issues) != base {
		t.Fatalf("rendering alone attempted a clipboard write: %#v", a.Issues)
	}
	if err := tester.Click("复制全部"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if len(a.Issues) != base+1 {
		t.Fatalf("explicit click did not surface the clipboard outcome: %#v", a.Issues)
	}
	last := a.Issues[len(a.Issues)-1]
	if !strings.Contains(last, "复制问题清单失败") || !strings.Contains(last, "窗口未就绪") {
		t.Fatalf("headless click failure = %q", last)
	}
}
