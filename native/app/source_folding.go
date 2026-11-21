package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	formatscode "levnote.local/native/formats/code"
	"levnote.local/native/ui/editor"
)

type sourceFoldCommand uint8

const (
	foldCurrentPosition sourceFoldCommand = iota
	unfoldCurrentPosition
	unfoldEverything
	reanalyzeFolds
)

// configuredSourceFoldProfile derives support from the actual matched builtin
// registry entry, never from a highlight language alias or a filename alone.
// Installed declarative/native routes can therefore occupy .go without
// inheriting Go grammar. Strict JSON is deliberately limited to .json.
func (a *App) configuredSourceFoldProfile(t *Tab) (formatscode.StructureProfile, bool) {
	if t == nil || !t.textCapable() || t.ByteInspector || t.Doc == nil ||
		t.Doc.Buffer == nil || t.Editor == nil || t.Editor.Buffer != t.Doc.Buffer ||
		a.pluginManager == nil {
		return "", false
	}
	matched := t.MatchedPlugin
	if matched.ID != "code" || !matched.Builtin || !matched.Enabled || !matched.Active {
		return "", false
	}
	path := t.Doc.Path
	if path == "" {
		path = t.Name
	}
	actual, err := a.pluginManager.Match(path)
	if err != nil || actual.ID != "code" || !actual.Manifest.Builtin ||
		!actual.Manifest.Enabled || !actual.Manifest.Active {
		return "", false
	}
	builtin, err := a.pluginManager.Manifest("code")
	if err != nil || !builtin.Builtin || !builtin.Enabled || !builtin.Active {
		return "", false
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return formatscode.StructureGo, true
	case ".json":
		return formatscode.StructureJSON, true
	default:
		return "", false
	}
}

func sourceFoldProvider(profile formatscode.StructureProfile) editor.FoldProvider {
	limits := formatscode.DefaultStructureLimits()
	return func(snapshot editor.FoldSnapshot) (editor.FoldResult, error) {
		result, err := formatscode.AnalyzeStructure(snapshot.Source, profile, limits)
		if err != nil {
			return editor.FoldResult{}, err
		}
		folds := make([]editor.FoldRange, 0, len(result.Folds))
		for _, fold := range result.Folds {
			folds = append(folds, editor.FoldRange{
				OpenLine: fold.Open.Line, CloseLine: fold.Close.Line,
				OpenStart: fold.Open.UTF16.Start, OpenEnd: fold.Open.UTF16.End,
				CloseStart: fold.Close.UTF16.Start, CloseEnd: fold.Close.UTF16.End,
			})
		}
		return editor.FoldResult{
			Complete: result.Complete, Unsupported: result.Unsupported,
			Diagnostics: len(result.Diagnostics), Folds: folds,
		}, nil
	}
}

// disableSourceFolding is called before any configureTab early return. It
// releases old targets for binary/inspector transitions and A→B→A changes.
func (a *App) disableSourceFolding(t *Tab) {
	if t == nil || t.Editor == nil {
		return
	}
	_ = t.Editor.SetFoldProvider(nil, 0, editor.MaxFoldBytes)
}

func (a *App) installConfiguredSourceFolding(t *Tab) {
	if t == nil || t.Editor == nil {
		return
	}
	profile, ok := a.configuredSourceFoldProfile(t)
	if !ok {
		return
	}
	// SetFoldProvider reports ordinary analysis failures through Editor.OnError
	// and keeps full-source status in the toolbar. Preserve that single report;
	// a provider-install error without a callback still needs a visible exit.
	previousOnError := t.Editor.OnError
	reported := false
	t.Editor.OnError = func(err error) {
		reported = true
		if previousOnError != nil {
			previousOnError(err)
		} else {
			a.report(err)
		}
	}
	err := t.Editor.SetFoldProvider(sourceFoldProvider(profile), t.OutlineFactoryGeneration, editor.MaxFoldBytes)
	t.Editor.OnError = previousOnError
	if err != nil && !reported {
		a.report(err)
	}
}

func (a *App) sourceFoldCommandError(t *Tab) error {
	if a.shuttingDown.Load() || a.restoring || a.modalDepth > 0 {
		return errors.New("工作区正在关闭、恢复或显示对话框；未执行折叠命令")
	}
	if t == nil || !a.tabIsOpen(t) || a.Current() != t || !t.textCapable() ||
		t.ByteInspector || t.Doc == nil || t.Doc.Buffer == nil || t.Editor == nil || t.Editor.Buffer != t.Doc.Buffer {
		return errors.New("当前活动标签没有可用的源码编辑区；未执行折叠命令")
	}
	if t.Mode != "source" && t.Mode != "split" {
		return errors.New("折叠命令仅适用于源码或分屏视图")
	}
	if t.Doc.Buffer.InGroup() || a.sourceCompositionActive(t) || a.findCompositionActive() || a.GoToLineComposing || a.openDocuments.composing {
		return errors.New("文档事务或输入法组合尚未结束；请先完成或取消，再执行折叠命令")
	}
	if _, ok := a.configuredSourceFoldProfile(t); !ok {
		return errors.New("当前格式没有受支持的结构折叠提供器")
	}
	return nil
}

func sourceFoldCommandAvailable(a *App) bool {
	return a != nil && a.sourceFoldCommandError(a.Current()) == nil
}

func foldContainsPosition(candidate editor.FoldCandidate, offset int) bool {
	return offset >= candidate.Range.OpenStart && offset <= candidate.Range.CloseEnd
}

func (a *App) runSourceFoldCommand(action sourceFoldCommand) {
	// Resolve the active tab, editor and caret now. No workbench command retains
	// a tab, source offset or FoldTarget from palette rendering.
	t := a.Current()
	if err := a.sourceFoldCommandError(t); err != nil {
		a.report(err)
		return
	}
	ed := t.Editor
	offset := ed.Buffer.Selection().Head
	var runErr error
	var reported bool
	previousOnError := ed.OnError
	ed.OnError = func(err error) {
		reported = true
		if previousOnError != nil {
			previousOnError(err)
		} else {
			a.report(err)
		}
	}
	func() {
		defer func() { ed.OnError = previousOnError }()
		switch action {
		case foldCurrentPosition:
			runErr = ed.FoldAtPosition(offset)
		case unfoldCurrentPosition:
			found := false
			for _, candidate := range ed.FoldCandidates() {
				if candidate.Collapsed && foldContainsPosition(candidate, offset) {
					found = true
					break
				}
			}
			if !found {
				runErr = errors.New("当前位置没有可展开的折叠块；如文档刚变化，请先重新分析折叠")
				return
			}
			runErr = ed.UnfoldAtPosition(offset)
		case unfoldEverything:
			collapsed := false
			for _, candidate := range ed.FoldCandidates() {
				collapsed = collapsed || candidate.Collapsed
			}
			if !collapsed {
				runErr = errors.New("当前没有已折叠内容；如文档刚变化，请先重新分析折叠")
				return
			}
			runErr = ed.UnfoldAll()
		case reanalyzeFolds:
			runErr = ed.RefreshFolds()
		default:
			runErr = fmt.Errorf("未知折叠命令 %d", action)
		}
	}()
	if runErr != nil && !reported {
		a.report(runErr)
	}
}
