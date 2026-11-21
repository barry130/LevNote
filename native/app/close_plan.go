package app

import (
	"errors"
	"fmt"
	"path/filepath"

	"levnote.local/native/core/document"
	"levnote.local/native/core/session"
)

var errCloseOwnerChanged = errors.New("关闭或恢复采集期间文档、视图或草稿已变化；未发布旧状态，请重新操作")

type preparedSessionState interface {
	Commit(func() error) error
	Abort() error
}

func (a *App) prepareCloseSnapshot(path string, snapshot session.Snapshot) (preparedSessionState, error) {
	if a.prepareSessionState != nil {
		return a.prepareSessionState(path, snapshot)
	}
	return session.PrepareSave(path, snapshot)
}

func closePublicationResult(path string, commitErr, cleanupErr error) error {
	if cleanupErr == nil {
		return commitErr
	}
	var publication *document.StatePublicationError
	if errors.As(commitErr, &publication) {
		return &document.StatePublicationError{
			Path: publication.Path, Committed: publication.Committed, Err: publication.Err,
			CleanupErr: errors.Join(publication.CleanupErr, cleanupErr),
		}
	}
	if commitErr == nil {
		return &document.StatePublicationError{Path: path, Committed: true, CleanupErr: cleanupErr}
	}
	return &document.StatePublicationError{Path: path, Err: commitErr, CleanupErr: cleanupErr}
}

// sessionSeal pins both the tab order and the actual data owners. Topology-only
// seals allow the intended Flush/Save edit but never a replacement owner. Full
// seals additionally pin content, confirmed-draft notifications and metadata.
func (a *App) sessionSeal(full bool) func() bool {
	tabs := append([]*Tab(nil), a.Tabs...)
	active, config, root := a.Active, a.configDir, a.Root
	draftGeneration := a.draftRecoveryGeneration
	dark, font, wrap, numbers, sidebar := a.Dark, a.FontSize, a.SoftWrap, a.LineNumbers, a.SidebarWidth
	checks := make([]func() bool, len(tabs))
	for i, t := range tabs {
		checks[i] = sealSessionTab(t, full)
	}
	return func() bool {
		if a.shuttingDown.Load() || a.Active != active || a.configDir != config || a.Root != root || len(a.Tabs) != len(tabs) {
			return false
		}
		if full && (a.draftRecoveryGeneration != draftGeneration || a.Dark != dark || a.FontSize != font || a.SoftWrap != wrap || a.LineNumbers != numbers || a.SidebarWidth != sidebar) {
			return false
		}
		for i, t := range tabs {
			if a.Tabs[i] != t || !checks[i]() {
				return false
			}
		}
		return true
	}
}

func sealSessionTab(t *Tab, full bool) func() bool {
	if t == nil {
		return func() bool { return false }
	}
	id, doc, input, view, adapter := t.ID, t.Doc, t.Editor, t.View, t.Adapter
	name, mode, original, outline := t.Name, t.Mode, t.OriginalPath, t.OutlineFactoryGeneration
	binary, binaryPath, binaryKind, binaryGeneration := t.Binary, t.BinaryPath, t.BinaryKind, t.BinaryGeneration
	inspector, plugin := t.ByteInspector, t.MatchedPlugin.ID
	base := func() bool {
		return t.ID == id && t.Doc == doc && t.Editor == input && t.View == view && t.Adapter == adapter && t.Name == name &&
			t.Mode == mode && t.OriginalPath == original && t.OutlineFactoryGeneration == outline && t.ByteInspector == inspector &&
			t.MatchedPlugin.ID == plugin && t.Binary == binary && t.BinaryPath == binaryPath && t.BinaryKind == binaryKind && t.BinaryGeneration == binaryGeneration
	}
	if doc == nil {
		return base
	}
	buffer, path := doc.Buffer, doc.Path
	encoding, bom, readonly, isNew := doc.Encoding, doc.BOM, doc.ReadOnly, doc.IsNew
	token := doc.DiskToken()
	if buffer == nil {
		return func() bool { return base() && doc.Buffer == nil && doc.Path == path }
	}
	version, selection, history := buffer.Version(), buffer.Selection(), buffer.History()
	return func() bool {
		return base() && doc.Buffer == buffer && input.Buffer == buffer && doc.Path == path &&
			(!full || buffer.Version() == version && buffer.Selection() == selection && buffer.History() == history &&
				doc.Encoding == encoding && doc.BOM == bom && doc.ReadOnly == readonly && doc.IsNew == isNew && (token.Valid() && doc.SameBaseline(token) || !token.Valid() && !doc.DiskToken().Valid()))
	}
}

// closePreparedSession is entered only after all explicit save/discard choices.
// Its temp file is complete before any worker is stopped. Until Commit every
// failure leaves the previous recovery bytes untouched and keeps live tabs.
func (a *App) closePreparedSession(discarded []bool) bool {
	valid := a.sessionSeal(true)
	builder, err := a.captureSessionTabs(a.Tabs, a.Active, discarded, "关闭未完成：无法准备恢复状态")
	if err != nil {
		return false
	}
	if !valid() {
		a.report(errCloseOwnerChanged)
		return false
	}
	snapshot, err := builder.Snapshot()
	if err != nil {
		a.reportPersistFailure(err)
		return false
	}
	path := filepath.Join(a.configDir, "session.json")
	prepared, err := a.prepareCloseSnapshot(path, snapshot)
	if err != nil {
		a.reportPersistFailure(fmt.Errorf("关闭未完成：无法准备恢复状态；现有恢复快照保持不变：%w", err))
		return false
	}
	if !valid() {
		err = closePublicationResult(path, errCloseOwnerChanged, prepared.Abort())
		a.reportPersistFailure(err)
		return false
	}
	if err := a.closeEngineExport(); err != nil {
		// Retain cleanup causes even when the runner's later Close is a no-op.
		a.enginePoison = err
		err = closePublicationResult(path, err, prepared.Abort())
		a.report(fmt.Errorf("关闭未完成：导出进程清理失败；现有恢复快照保持不变：%w", err))
		return false
	}
	check := func() error {
		if !valid() {
			return errCloseOwnerChanged
		}
		return nil
	}
	err = closePublicationResult(path, prepared.Commit(check), prepared.Abort())
	var published *document.StatePublicationError
	if err != nil && (!errors.As(err, &published) || !published.Committed) {
		a.reportPersistFailure(fmt.Errorf("关闭未完成：恢复状态未发布；现有恢复快照保持不变：%w", err))
		return false
	}
	// Publication is the close commit point. A directory-sync or temp-cleanup
	// error cannot undo it. Complete the close and disclose the exact outcome.
	a.lastPersistError = ""
	a.cancelReopen()
	for len(a.Tabs) > 0 {
		a.removeTab(len(a.Tabs) - 1)
	}
	a.cancelAutosave()
	a.persistPending = false
	if err != nil {
		a.report(fmt.Errorf("关闭已提交，恢复文件已发布：%w", err))
	}
	return true
}
