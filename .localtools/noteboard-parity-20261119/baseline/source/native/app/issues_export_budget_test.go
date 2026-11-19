package app

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"levnote.local/native/core/document"
	"levnote.local/native/core/search"
)

// The clipboard writer is the agreed public action seam: the exported header
// must identify the real native app and the current copy occasion, not an issue.
func TestCopyIssuesIdentifiesApplicationAndExportMoment(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	before := time.Now().UTC().Truncate(time.Second)
	var copied string
	a.copyIssues(func(text string) error {
		copied = text
		return nil
	})
	after := time.Now().UTC()
	if !strings.Contains(copied, "应用：轻笺 · 原生版\n") {
		t.Fatalf("copy lacks the native application identity: %q", copied)
	}
	_, rest, ok := strings.Cut(copied, "导出时刻：")
	if !ok {
		t.Fatal("copy lacks an export moment")
	}
	stamp, _, _ := strings.Cut(rest, "\n")
	got, err := time.Parse(time.RFC3339, stamp)
	if err != nil || got.Before(before) || got.After(after) {
		t.Fatalf("export moment %q is outside [%s, %s]: %v", stamp, before, after, err)
	}
	if strings.Contains(copied, "发生时刻") || strings.Contains(copied, "恢复时刻") {
		t.Fatal("copy invented timestamps for historical issues")
	}
}

// Oversized data in each field and family is observed at the actual copy seam.
// The total budget is a spec literal, independent of production constants.
func TestCopyIssuesBoundsOversizedFamiliesWithoutChangingOriginals(t *testing.T) {
	giant := strings.Repeat("甲🙂乙", 100000)
	for _, family := range []string{"tab", "warnings", "search-path", "search-reason", "search-detail", "issues", "mixture"} {
		t.Run(family, func(t *testing.T) {
			a := newTestApp(t, t.TempDir())
			tab := a.appendTab(document.New("notes"), "notes.md", "", false, false)
			if !a.Select(0) {
				t.Fatal("fixture tab was not activated")
			}
			tab.ViewWarnings = []string{"short-warning"}
			a.SearchIssues = []search.FileIssue{{Path: "short-path", Reason: "short-reason", Detail: "short-detail"}}
			a.Issues = []string{"short-issue"}
			a.IssuesDropped, a.SearchIssuesDropped = 7, 3
			switch family {
			case "tab":
				tab.Name = giant
			case "warnings":
				tab.ViewWarnings[0] = giant
			case "search-path":
				a.SearchIssues[0].Path = giant
			case "search-reason":
				a.SearchIssues[0].Reason = giant
			case "search-detail":
				a.SearchIssues[0].Detail = giant
			case "issues":
				a.Issues[0] = giant
			case "mixture":
				tab.Name = giant
				tab.ViewWarnings[0] = giant
				a.SearchIssues[0] = search.FileIssue{Path: giant, Reason: giant, Detail: giant}
				a.Issues[0] = giant
			}
			name := tab.Name
			warnings := append([]string(nil), tab.ViewWarnings...)
			searches := append([]search.FileIssue(nil), a.SearchIssues...)
			issues := append([]string(nil), a.Issues...)
			var copied string
			a.copyIssues(func(text string) error { copied = text; return nil })
			if len(copied) > 64*1024 || !utf8.ValidString(copied) {
				t.Fatalf("copy is not bounded valid UTF-8: bytes=%d valid=%v", len(copied), utf8.ValidString(copied))
			}
			for _, want := range []string{"复制预算：65536 UTF-8 字节", "当前标签：", "当前 1 条", "另有 7 条较早记录未保留", "当前格式的提示", "工作区搜索跳过项（显示 1 项，另有 3 项未保留）", "问题记录", "本次复制截短", "本次复制摘要"} {
				if !strings.Contains(copied, want) {
					t.Errorf("copy lost %q (bytes=%d)", want, len(copied))
				}
			}
			warningTruncated, searchTruncated, issueTruncated := 0, 0, 0
			if family == "warnings" || family == "mixture" {
				warningTruncated = 1
			}
			if strings.HasPrefix(family, "search-") || family == "mixture" {
				searchTruncated = 1
			}
			if family == "issues" || family == "mixture" {
				issueTruncated = 1
			}
			for label, truncated := range map[string]int{"格式提示": warningTruncated, "搜索跳过": searchTruncated, "问题记录": issueTruncated} {
				want := fmt.Sprintf("%s：复制 1/1 项，本次预算省略 0 项，单条截短 %d 项。", label, truncated)
				if !strings.Contains(copied, want) {
					t.Errorf("copy has a false truncation/omission count: %q", want)
				}
			}
			for _, short := range []string{"short-warning", "short-path", "short-reason", "short-detail", "short-issue"} {
				presentInOriginal := warnings[0] == short || searches[0].Path == short || searches[0].Reason == short || searches[0].Detail == short || issues[0] == short
				if presentInOriginal && !strings.Contains(copied, short) {
					t.Errorf("oversized field hid unrelated short text %q", short)
				}
			}
			if tab.Name != name || !reflect.DeepEqual(tab.ViewWarnings, warnings) || !reflect.DeepEqual(a.SearchIssues, searches) || !reflect.DeepEqual(a.Issues, issues) || a.IssuesDropped != 7 || a.SearchIssuesDropped != 3 {
				t.Fatal("copy changed retained originals or historical counts")
			}
		})
	}
}

func TestCopyIssuesManyRecordsPreserveFamilyHeadersAndExactOmissions(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	tab := a.appendTab(document.New("notes"), "many.md", "", false, false)
	if !a.Select(0) {
		t.Fatal("fixture tab was not activated")
	}
	for i := 0; i < 200; i++ {
		tab.ViewWarnings = append(tab.ViewWarnings, strings.Repeat("W", 2047))
		a.SearchIssues = append(a.SearchIssues, search.FileIssue{Path: strings.Repeat("P", 640), Reason: strings.Repeat("R", 640), Detail: strings.Repeat("D", 640)})
		a.Issues = append(a.Issues, strings.Repeat("I", 2047))
	}
	a.IssuesDropped, a.SearchIssuesDropped = 7, 3
	var copied string
	a.copyIssues(func(text string) error { copied = text; return nil })
	if len(copied) > 65536 || !utf8.ValidString(copied) {
		t.Fatalf("many records exceed budget or break Unicode: %d bytes", len(copied))
	}
	for _, want := range []string{
		"当前标签：many.md", "最近 200 条（当前 200 条）。另有 7 条较早记录未保留。",
		"当前格式的提示", "工作区搜索跳过项（显示 200 项，另有 3 项未保留）", "问题记录",
		"格式提示：复制 10/200 项，本次预算省略 190 项，单条截短 0 项。",
		"搜索跳过：复制 10/200 项，本次预算省略 190 项，单条截短 0 项。",
		"问题记录：复制 10/200 项，本次预算省略 190 项，单条截短 0 项。",
	} {
		if !strings.Contains(copied, want) {
			t.Errorf("missing truthful count/header %q", want)
		}
	}
	if len(tab.ViewWarnings) != 200 || len(a.SearchIssues) != 200 || len(a.Issues) != 200 || a.IssuesDropped != 7 || a.SearchIssuesDropped != 3 {
		t.Fatal("export omissions changed retained or historical counts")
	}
	t.Logf("three full families: %d bytes; copied 10/200 each; omitted 190 each; truncated 0", len(copied))
}

func TestCopyIssuesManyOversizedRecordsSeparateOmissionAndTruncation(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	tab := a.appendTab(document.New("notes"), "giant.md", "", false, false)
	if !a.Select(0) {
		t.Fatal("fixture tab was not activated")
	}
	giant := strings.Repeat("甲🙂乙", 100000)
	for i := 0; i < 200; i++ {
		tab.ViewWarnings = append(tab.ViewWarnings, giant)
		a.SearchIssues = append(a.SearchIssues, search.FileIssue{Path: giant, Reason: giant, Detail: giant})
		a.Issues = append(a.Issues, giant)
	}
	var copied string
	a.copyIssues(func(text string) error { copied = text; return nil })
	if len(copied) > 65536 || !utf8.ValidString(copied) {
		t.Fatalf("many giant records exceed budget or break UTF-8: %d bytes", len(copied))
	}
	for _, label := range []string{"格式提示", "搜索跳过", "问题记录"} {
		want := label + "：复制 10/200 项，本次预算省略 190 项，单条截短 10 项。"
		if !strings.Contains(copied, want) {
			t.Errorf("budget omission and excerpt truncation are not distinct: %q", want)
		}
	}
	if strings.Contains(copied, "未保留") || a.IssuesDropped != 0 || a.SearchIssuesDropped != 0 {
		t.Fatal("export omission was mislabeled as historical loss")
	}
	if tab.ViewWarnings[199] != giant || a.SearchIssues[199].Detail != giant || a.Issues[199] != giant {
		t.Fatal("unexported tails were changed")
	}
	t.Logf("three giant families: %d bytes; copied 10/200 each; omitted 190 each; truncated 10 each", len(copied))
}

func TestCopyIssuesKeepsEveryShortRecordWithoutFalseOmissions(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	tab := a.appendTab(document.New("notes"), "short🙂.md", "", false, false)
	if !a.Select(0) {
		t.Fatal("fixture tab was not activated")
	}
	for i := 0; i < 200; i++ {
		tab.ViewWarnings = append(tab.ViewWarnings, fmt.Sprintf("warning-%03d-甲🙂", i))
		a.SearchIssues = append(a.SearchIssues, search.FileIssue{Path: fmt.Sprintf("path-%03d", i), Reason: "reason", Detail: "detail🙂"})
		a.Issues = append(a.Issues, fmt.Sprintf("issue-%03d-乙🙂", i))
	}
	var copied string
	a.copyIssues(func(text string) error { copied = text; return nil })
	for i := 0; i < 200; i++ {
		for _, want := range []string{fmt.Sprintf("warning-%03d-甲🙂\n", i), fmt.Sprintf("path-%03d · reason · detail🙂\n", i), fmt.Sprintf("issue-%03d-乙🙂\n", i)} {
			if !strings.Contains(copied, want) {
				t.Fatalf("copy lost short retained record %q", want)
			}
		}
	}
	if len(copied) > 65536 || !utf8.ValidString(copied) || strings.Contains(copied, "未保留") || strings.Contains(copied, "…[本次复制截短]") {
		t.Fatalf("short copy invented loss or exceeded budget: bytes=%d", len(copied))
	}
	for _, label := range []string{"格式提示", "搜索跳过", "问题记录"} {
		if !strings.Contains(copied, label+"：复制 200/200 项，本次预算省略 0 项，单条截短 0 项。") {
			t.Fatalf("short-copy counts are false for %s", label)
		}
	}
}

func TestCopyIssuesMultibyteExcerptBoundaries(t *testing.T) {
	for _, family := range []string{"tab", "warning", "path", "reason", "detail", "issue"} {
		limit := 640
		if family == "tab" {
			limit = 512
		} else if family == "warning" || family == "issue" {
			limit = 2047
		}
		for _, extra := range []int{0, 1, 2, 3, 4} {
			t.Run(fmt.Sprintf("%s/%d", family, extra), func(t *testing.T) {
				a := newTestApp(t, t.TempDir())
				tab := a.appendTab(document.New("notes"), "boundary.md", "", false, false)
				if !a.Select(0) {
					t.Fatal("fixture tab was not activated")
				}
				text := strings.Repeat("甲🙂", (limit+extra)/7) + strings.Repeat("x", (limit+extra)%7)
				tab.ViewWarnings = []string{"warning"}
				a.SearchIssues = []search.FileIssue{{Path: "path", Reason: "reason", Detail: "detail"}}
				a.Issues = []string{"issue"}
				switch family {
				case "tab":
					tab.Name = text
				case "warning":
					tab.ViewWarnings[0] = text
				case "path":
					a.SearchIssues[0].Path = text
				case "reason":
					a.SearchIssues[0].Reason = text
				case "detail":
					a.SearchIssues[0].Detail = text
				case "issue":
					a.Issues[0] = text
				}
				var copied string
				a.copyIssues(func(value string) error { copied = value; return nil })
				if len(copied) > 65536 || !utf8.ValidString(copied) {
					t.Fatalf("boundary broke UTF-8 or total budget: %d bytes", len(copied))
				}
				if extra == 0 {
					if !strings.Contains(copied, text) || strings.Contains(copied, "…[本次复制截短]") {
						t.Fatal("exactly fitting Unicode field was changed")
					}
				} else if strings.Contains(copied, text) || !strings.Contains(copied, "…[本次复制截短]") {
					t.Fatal("oversized Unicode field was not explicitly truncated")
				}
			})
		}
	}
}

func TestCopyIssuesDoesNotAllocateWholeOversizedInputs(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	tab := a.appendTab(document.New("notes"), "notes.md", "", false, false)
	if !a.Select(0) {
		t.Fatal("fixture tab was not activated")
	}
	giant := strings.Repeat("甲🙂", 1<<20)
	tab.Name, tab.ViewWarnings = giant, []string{giant}
	a.SearchIssues = []search.FileIssue{{Path: giant, Reason: giant, Detail: giant}}
	a.Issues = []string{giant}
	write := func(value string) error {
		if len(value) > 65536 {
			t.Fatalf("writer received oversized report: %d", len(value))
		}
		return nil
	}
	a.copyIssues(write) // Warm formatter state before measuring export-only work.
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	for i := 0; i < 3; i++ {
		a.copyIssues(write)
	}
	runtime.ReadMemStats(&after)
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 1<<20 {
		t.Fatalf("three bounded copies allocated %d bytes; whole giant concatenation is forbidden", allocated)
	} else {
		t.Logf("three bounded copies export-only allocated %d bytes (not whole-app/RSS evidence)", allocated)
	}
	runtime.KeepAlive(giant)
}

func TestCopyIssuesResamplesTimeAndWaitsForWriteAcknowledgement(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	a.Message = "before copy"
	var first string
	a.copyIssues(func(value string) error {
		if a.Message != "before copy" {
			t.Fatal("success was reported before the writer acknowledged")
		}
		first = value
		return nil
	})
	_, rest, _ := strings.Cut(first, "导出时刻：")
	stamp, _, _ := strings.Cut(rest, "\n")
	firstTime, err := time.Parse(time.RFC3339, stamp)
	if err != nil {
		t.Fatal(err)
	}
	// Await one real wall-clock rollover, not a cached or injected render clock.
	timer := time.NewTimer(time.Until(firstTime.Add(time.Second)) + 10*time.Millisecond)
	defer timer.Stop()
	<-timer.C
	a.Message = "before second copy"
	before := time.Now().UTC().Truncate(time.Second)
	var second string
	a.copyIssues(func(value string) error {
		if a.Message != "before second copy" {
			t.Fatal("second success was reported before the writer acknowledged")
		}
		second = value
		return errors.New("explicit second-write failure")
	})
	after := time.Now().UTC()
	_, rest, _ = strings.Cut(second, "导出时刻：")
	stamp, _, _ = strings.Cut(rest, "\n")
	secondTime, err := time.Parse(time.RFC3339, stamp)
	if err != nil || !secondTime.After(firstTime) || secondTime.Before(before) || secondTime.After(after) {
		t.Fatalf("second export time is stale/outside copy interval: %q, %v", stamp, err)
	}
	if !strings.Contains(a.Message, "explicit second-write failure") || len(a.Issues) != 1 || !strings.Contains(a.Issues[0], "复制问题清单失败") {
		t.Fatal("second failed write was not explicit and nonrecursive")
	}
}
