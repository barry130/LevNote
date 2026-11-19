package app

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
)

// issuesClipboardWriter is the real seam between the issues snapshot and the
// clipboard. Production wiring goes through mygo.Clipboard; tests inject
// outcomes. Copying stays an explicit user action: nothing outside copyIssues
// may call the writer.
type issuesClipboardWriter func(text string) error

// writeIssuesClipboard mirrors the editor Copy seam: clipboard access requires
// a live window, and a missing window must fail loudly instead of silently.
func (a *App) writeIssuesClipboard(text string) error {
	if a.Window == nil {
		return errors.New("窗口未就绪，剪贴板不可用")
	}
	return mygo.Clipboard.Write(transfer.TextData(text))
}

const (
	// Three independent 20 KiB body shares leave 4 KiB for the bounded tab,
	// fixed headers and numeric summaries. One large family cannot hide another.
	issuesExportMaxBytes    = 64 * 1024
	issuesExportFamilyBytes = 20 * 1024
	issuesExportRecordBytes = 2 * 1024
	issuesExportFieldBytes  = 640
	issuesExportTabBytes    = 512
	issuesExportMarker      = "…[本次复制截短]"
)

// issuesExportExcerpt touches only a bounded prefix and at most three boundary
// bytes, never scanning or copying the unexported tail. Inputs are Unicode text;
// cutting at a rune boundary preserves it without rewriting the retained value.
func issuesExportExcerpt(text string, limit int) (string, bool) {
	if len(text) <= limit {
		return text, false
	}
	end := limit - len(issuesExportMarker)
	for end > 0 && !utf8.RuneStart(text[end]) {
		end--
	}
	return text[:end] + issuesExportMarker, true
}

type issuesExportFamily struct {
	bytes, copied, truncated int
}

// append stops at the first row that cannot fit. Callers derive omitted records
// from the original slice length, so omitted tails need no iteration or copies.
func (f *issuesExportFamily) append(snapshot *strings.Builder, row string, truncated bool) bool {
	if len(row)+1 > issuesExportFamilyBytes-f.bytes {
		return false
	}
	snapshot.WriteString(row)
	snapshot.WriteByte('\n')
	f.bytes += len(row) + 1
	f.copied++
	if truncated {
		f.truncated++
	}
	return true
}

func (f issuesExportFamily) summary(snapshot *strings.Builder, label string, total int) {
	fmt.Fprintf(snapshot, "%s：复制 %d/%d 项，本次预算省略 %d 项，单条截短 %d 项。\n",
		label, f.copied, total, total-f.copied, f.truncated)
}

// issuesSnapshot samples this export's current time; no render clock is stored
// and no historical occurrence time is invented. The retained record counts and
// old dropped counts remain panel facts; export omissions are disclosed apart.
func (a *App) issuesSnapshot() string {
	var snapshot strings.Builder
	snapshot.Grow(issuesExportMaxBytes)
	snapshot.WriteString("应用：轻笺 · 原生版\n")
	fmt.Fprintf(&snapshot, "导出时刻：%s\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Fprintf(&snapshot, "复制预算：%d UTF-8 字节；正文每族 %d 字节；单条 %d 字节；搜索字段 %d 字节；标签 %d 字节（含截短标记）。\n",
		issuesExportMaxBytes, issuesExportFamilyBytes, issuesExportRecordBytes, issuesExportFieldBytes, issuesExportTabBytes)
	current := a.Current()
	tabTruncated := false
	if current != nil {
		var name string
		name, tabTruncated = issuesExportExcerpt(current.Name, issuesExportTabBytes)
		fmt.Fprintf(&snapshot, "当前标签：%s\n", name)
	} else {
		snapshot.WriteString("当前标签：无\n")
	}
	fmt.Fprintf(&snapshot, "问题清单只保留本次运行最近 %d 条（当前 %d 条）。", issueHistoryLimit, len(a.Issues))
	if a.IssuesDropped > 0 {
		fmt.Fprintf(&snapshot, "另有 %d 条较早记录未保留。", a.IssuesDropped)
	}
	snapshot.WriteByte('\n')
	var warnings, searches, issues issuesExportFamily
	warningCount := 0
	if current != nil && len(current.ViewWarnings) > 0 {
		warningCount = len(current.ViewWarnings)
		snapshot.WriteString("当前格式的提示\n")
		for _, warning := range current.ViewWarnings {
			row, truncated := issuesExportExcerpt(warning, issuesExportRecordBytes-1)
			if !warnings.append(&snapshot, row, truncated) {
				break
			}
		}
	}
	if len(a.SearchIssues) > 0 {
		fmt.Fprintf(&snapshot, "工作区搜索跳过项（显示 %d 项", len(a.SearchIssues))
		if a.SearchIssuesDropped > 0 {
			fmt.Fprintf(&snapshot, "，另有 %d 项未保留", a.SearchIssuesDropped)
		}
		snapshot.WriteString("）\n")
		for _, issue := range a.SearchIssues {
			path, pathTruncated := issuesExportExcerpt(issue.Path, issuesExportFieldBytes)
			reason, reasonTruncated := issuesExportExcerpt(issue.Reason, issuesExportFieldBytes)
			row := path + " · " + reason
			truncated := pathTruncated || reasonTruncated
			if issue.Detail != "" {
				detail, detailTruncated := issuesExportExcerpt(issue.Detail, issuesExportFieldBytes)
				row += " · " + detail
				truncated = truncated || detailTruncated
			}
			if !searches.append(&snapshot, row, truncated) {
				break
			}
		}
	}
	if len(a.Issues) > 0 {
		snapshot.WriteString("问题记录\n")
		for _, issue := range a.Issues {
			row, truncated := issuesExportExcerpt(issue, issuesExportRecordBytes-1)
			if !issues.append(&snapshot, row, truncated) {
				break
			}
		}
	}
	if tabTruncated || warningCount > 0 || len(a.SearchIssues) > 0 || len(a.Issues) > 0 {
		snapshot.WriteString("本次复制摘要（预算省略与单条截短仅影响副本，不改变历史计数或原文）：\n")
		if tabTruncated {
			snapshot.WriteString("当前标签：单条截短 1 项。\n")
		}
		if warningCount > 0 {
			warnings.summary(&snapshot, "格式提示", warningCount)
		}
		if len(a.SearchIssues) > 0 {
			searches.summary(&snapshot, "搜索跳过", len(a.SearchIssues))
		}
		if len(a.Issues) > 0 {
			issues.summary(&snapshot, "问题记录", len(a.Issues))
		}
	}
	return snapshot.String()
}

// copyIssues is the explicit 复制全部 action: the write happens synchronously
// through the real seam and the outcome is disclosed either way. A failed copy
// is reported through the issues channel itself — the report never re-enters
// the clipboard, so a failing clipboard cannot recurse.
func (a *App) copyIssues(write issuesClipboardWriter) {
	if write == nil {
		a.report(errors.New("复制问题清单失败：剪贴板不可用"))
		return
	}
	if err := write(a.issuesSnapshot()); err != nil {
		a.report(fmt.Errorf("复制问题清单失败：%v", err))
		return
	}
	a.Message = "问题清单已复制到剪贴板。"
}
