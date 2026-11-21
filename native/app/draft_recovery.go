package app

import (
	"errors"
	"path"
	"strings"

	"levnote.local/native/core/document"
	"levnote.local/native/core/session"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

// appendSessionDraft runs on the UI thread when the host captures a session.
// A separate record owns the confirmed text without carrying a writable path.
func appendSessionDraft(builder *session.SnapshotBuilder, tab *Tab) error {
	capturer, ok := tab.View.(editor.SessionDraftCapturer)
	if !ok {
		return nil
	}
	valid := sealSessionTab(tab, true)
	draft, ok, err := capturer.CaptureSessionDraft(session.MaxDraftBytes)
	if !valid() {
		return errCloseOwnerChanged
	}
	if err != nil || !ok {
		return err
	}
	if len(draft.Content) > session.MaxDraftBytes {
		return errors.New("独立草稿超过 4 MiB 恢复上限")
	}
	units := text.UTF16Len(draft.Content)
	if draft.Anchor < 0 || draft.Head < 0 || draft.Anchor > units || draft.Head > units {
		return errors.New("独立草稿选区不属于已确认内容；现有恢复快照保持不变")
	}
	name := path.Base(strings.ReplaceAll(strings.TrimSpace(draft.Title), `\`, "/"))
	if name == "." || name == ".." || name == "/" {
		name = "独立草稿"
	}
	if !strings.EqualFold(path.Ext(name), ".txt") {
		name += ".txt"
	}
	originalPath := tab.Doc.Path
	if originalPath == "" {
		originalPath = tab.OriginalPath
	}
	return builder.AddTab(session.Tab{
		OriginalPath: originalPath,
		Title:        "恢复-" + name,
		Draft:        &draft.Content,
		Encoding:     document.UTF8,
		Selection:    text.Range{Anchor: draft.Anchor, Head: draft.Head},
		View:         "source",
	})
}
