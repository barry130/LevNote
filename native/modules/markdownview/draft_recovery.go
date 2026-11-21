package markdownview

import (
	"errors"
	"path/filepath"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

const maxConfirmedCellDraftBytes = 4 << 20

var errCellDraftRecoveryLimit = errors.New("单元格恢复草稿超过允许字节；保留此前有效恢复快照")

// CaptureSessionDraft copies only an independently edited cell. It never
// flushes the editor or touches the host buffer, focus, selection, or history.
func (v *View) CaptureSessionDraft(maxBytes int) (editor.SessionDraft, bool, error) {
	if maxBytes <= 0 {
		return editor.SessionDraft{}, false, errCellDraftRecoveryLimit
	}
	draft := v.draft
	edit := v.cellEdit
	if draft == nil || edit == nil {
		return editor.SessionDraft{}, false, nil
	}
	limit := min(maxBytes, maxConfirmedCellDraftBytes)
	marked, composing := draft.MarkedRange()
	selection := v.recoveryConfirmedSelection
	snapshot := v.recoverySnapshot
	if composing {
		if v.recoveryOverflow || !v.recoverySnapshotSet || v.recoverySnapshotBytes > limit {
			return editor.SessionDraft{}, false, errCellDraftRecoveryLimit
		}
	} else if draft.Buffer.LenBytes() > limit {
		return editor.SessionDraft{}, false, errCellDraftRecoveryLimit
	} else if v.recoveryOverflow || !v.recoverySnapshotSet || draft.Buffer.Version() != v.recoveryVersion ||
		draft.Buffer.Selection() != v.recoveryObservedSelection || marked != v.recoveryMarked ||
		v.recoveryMarkedActive {
		// UnmarkText deliberately has no OnChange callback. When capture is the
		// first observation after that transition, make an ephemeral bounded
		// snapshot without notifying the host or mutating scheduling state.
		snapshot = draft.Buffer.Snapshot()
		selection = draft.Buffer.Selection()
	}
	value := snapshot.String()
	if value == v.editBase {
		return editor.SessionDraft{}, false, nil
	}
	name := filepath.Base(v.host.Path)
	if name == "" || name == "." || name == string(filepath.Separator) {
		name = "Markdown"
	}
	return editor.SessionDraft{
		Title: name + "-表格单元格草稿.txt", Content: value,
		Anchor: selection.Anchor, Head: selection.Head,
	}, true, nil
}

func (v *View) confirmCellDraft(draft *editor.Editor) {
	v.observeCellDraftConfirmation(draft)
}

func (v *View) observeCellDraftConfirmation(draft *editor.Editor) {
	if v.draft != draft || v.cellEdit == nil || v.recoveryDraft != draft {
		return
	}
	marked, composing := draft.MarkedRange()
	selection := draft.Buffer.Selection()
	version := draft.Buffer.Version()
	if version == v.recoveryVersion && selection == v.recoveryObservedSelection &&
		composing == v.recoveryMarkedActive && marked == v.recoveryMarked {
		return
	}
	v.recoveryVersion = version
	v.recoveryObservedSelection = selection
	v.recoveryMarked, v.recoveryMarkedActive = marked, composing
	if composing {
		v.recoveryComposing = true
		return
	}
	v.recoveryComposing = false
	if draft.Buffer.LenBytes() > maxConfirmedCellDraftBytes {
		v.recoveryOverflow = true
	} else {
		v.recoverySnapshot = draft.Buffer.Snapshot()
		v.recoverySnapshotBytes = draft.Buffer.LenBytes()
		v.recoverySnapshotSet = true
		v.recoveryConfirmedSelection = selection
		v.recoveryOverflow = false
	}
	if v.host.DraftRecoveryChanged != nil {
		v.host.DraftRecoveryChanged(v)
	}
}

func (v *View) clearCellDraftRecovery() {
	if v.recoveryDraft == nil {
		return
	}
	v.recoveryDraft = nil
	v.recoveryComposing = false
	v.recoverySnapshot = text.Snapshot{}
	v.recoverySnapshotBytes = 0
	v.recoverySnapshotSet = false
	v.recoveryObservedSelection = text.Range{}
	v.recoveryConfirmedSelection = text.Range{}
	v.recoveryVersion = 0
	v.recoveryMarked = ui.TextInputRange{}
	v.recoveryMarkedActive = false
	v.recoveryOverflow = false
	if v.host.DraftRecoveryChanged != nil {
		v.host.DraftRecoveryChanged(v)
	}
}
