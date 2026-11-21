package app

import (
	"errors"
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/plugins"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/code"
	"levnote.local/native/ui/editor"
)

const (
	sourceLanguageIdle       = "idle"
	sourceLanguageRunning    = "running"
	sourceLanguageStale      = "stale"
	sourceLanguageHostError  = "host-error"
	sourceLanguageFieldBytes = 16 << 10
)

// Intent contains bounded request fields, never a whole source snapshot.
// ByteOffset and editing descriptors are fixed at admission; a queued request
// must not silently follow a later caret.
type sourceLanguageIntent struct {
	operation        code.ServiceOperation
	byteOffset       int
	utf16Offset      int
	selectedEndUTF16 int
	hasUTF16Offset   bool
	hasSelectedRange bool
	prefix, newName  string
	limits           code.ServiceLimits
	// Editing intents retain bounded fields and the original full selection
	// identity. Ordinary B1 read intents deliberately leave this nil.
	completion *sourceCompletionIntent
	rename     *sourceRenameRequest
}

type sourceLanguageOwner struct {
	session, activation, request, route uint64
	tab                                 *Tab
	doc                                 *document.Document
	input                               *editor.Editor
	buffer                              *text.Buffer
	manager                             *plugins.Manager
	target                              editor.SourceTarget
	profile                             code.StructureProfile
	mode, path, name                    string
	encoding                            document.Encoding
	bom, docReadOnly, editorReadOnly    bool
	version                             uint64
}

type sourceLanguagePending struct {
	owner    *sourceLanguageOwner
	intent   sourceLanguageIntent
	dispatch func(func())
}

// Everything here except the synchronized task is UI-owned. No worker reads
// this state, including during cancellation and shutdown.
type sourceLanguageFocusStamp struct {
	findGeneration, findFocusGeneration, goToLineGeneration uint64
	findFocusIntentEpoch                                    uint64
	findFocus                                               findFocusTarget
}

// sourceLanguageShortcut holds no source snapshot or request target. The Editor
// owns its attachment seal; this Host seal owns the actual main presentation.
// Keep this record stable across ordinary frames and caret/text changes.
type sourceLanguageShortcut struct {
	generation, session, route uint64
	tab                        *Tab
	doc                        *document.Document
	input                      *editor.Editor
	buffer                     *text.Buffer
	manager                    *plugins.Manager
	profile                    code.StructureProfile
	mode, path, name           string
	encoding                   document.Encoding
	bom, docReadOnly, readOnly bool
	handler                    func(editor.SourceAction)
	lease                      editor.SourceActionBinding
}

type sourceLanguageState struct {
	shortcutSequence                            uint64
	shortcut                                    *sourceLanguageShortcut
	session, activation, request, routeSequence uint64
	owner                                       *sourceLanguageOwner
	pending                                     *sourceLanguagePending
	task                                        *sourceLanguageTask
	status, reason                              string
	message                                     string
	intent                                      sourceLanguageIntent
	analysisVersion                             uint64
	result                                      code.ServiceResult
	failure                                     error

	visible            bool
	presentation       uint64
	presentationTab    *Tab
	presentationDoc    *document.Document
	presentationInput  *editor.Editor
	presentationBuffer *text.Buffer
	presentationRoute  uint64
	presentationMode   string
	selected           int
	list               ui.ListState
	listLifetime       appListLifetime
	rows               []sourceLanguageRow
	rowsReady          bool
	rowRequest         uint64
	rowPresentation    uint64
	action             *sourceLanguageAction
	completionFocus    bool // consumed only after the public List Handle is built
	rename             *sourceRenameDialog
	cachedPositions    map[string]sourceLanguagePosition
}

func (a *App) configureSourceLanguage(t *Tab) {
	// Configuration declares capability only; folding already did its one
	// existing provider refresh. This must never analyze or refresh it again.
	profile, ok := a.configuredSourceFoldProfile(t)
	if ok {
		t.sourceLanguageProfile = profile
	}
}

// revokeSourceLanguageShortcut runs even when there has never been a request.
// Detach before any release/callback can reenter and install a successor.
func (a *App) revokeSourceLanguageShortcut(t *Tab) {
	s := &a.sourceLanguage
	binding := s.shortcut
	if binding == nil || t != nil && binding.tab != t {
		return
	}
	s.shortcut = nil
	s.shortcutSequence++
	// Only clear the handler we installed. An expired lease cannot detach a
	// replacement handler on the same Editor.
	if binding.input.SourceActionBindingCurrent(binding.lease) {
		binding.input.SetSourceActionHandler(nil)
	}
}

func (a *App) sourceLanguageShortcutCurrent(binding *sourceLanguageShortcut) bool {
	if binding == nil || a.sourceLanguage.shortcut != binding ||
		a.sourceLanguage.shortcutSequence != binding.generation || a.sourceLanguage.session != binding.session ||
		a.shuttingDown.Load() || a.restoring || a.ShowPlugins {
		return false
	}
	t := binding.tab
	return t != nil && a.Current() == t && a.tabIsOpen(t) && t.textCapable() &&
		t.Doc == binding.doc && t.Editor == binding.input && t.Doc.Buffer == binding.buffer &&
		t.Editor.Buffer == binding.buffer && a.pluginManager == binding.manager &&
		t.sourceLanguageRoute == binding.route && t.sourceLanguageProfile == binding.profile &&
		t.Mode == binding.mode && (t.Mode == "source" || t.Mode == "split") && !t.ByteInspector &&
		t.Doc.Path == binding.path && t.Name == binding.name && t.Doc.Encoding == binding.encoding &&
		t.Doc.BOM == binding.bom && t.Doc.ReadOnly == binding.docReadOnly && t.Editor.ReadOnly == binding.readOnly &&
		binding.input.SourceActionBindingCurrent(binding.lease)
}

// bindSourceLanguageShortcut is called only immediately before drawing the real
// main source or split-left Editor. Project, find, rename and preview fields
// never pass through this boundary. It performs no source materialization.
func (a *App) bindSourceLanguageShortcut(t *Tab) {
	s := &a.sourceLanguage
	if t == nil || a.Current() != t || !a.tabIsOpen(t) || !t.textCapable() || t.Editor == nil ||
		t.Doc.Buffer == nil || t.Editor.Buffer != t.Doc.Buffer || t.ByteInspector ||
		(t.Mode != "source" && t.Mode != "split") || a.shuttingDown.Load() || a.restoring || a.ShowPlugins {
		a.revokeSourceLanguageShortcut(nil)
		return
	}
	if a.sourceLanguageShortcutCurrent(s.shortcut) {
		return
	}
	a.revokeSourceLanguageShortcut(nil)
	s.shortcutSequence++
	binding := &sourceLanguageShortcut{generation: s.shortcutSequence, session: s.session,
		tab: t, doc: t.Doc, input: t.Editor, buffer: t.Doc.Buffer, manager: a.pluginManager,
		route: t.sourceLanguageRoute, profile: t.sourceLanguageProfile, mode: t.Mode,
		path: t.Doc.Path, name: t.Name, encoding: t.Doc.Encoding, bom: t.Doc.BOM,
		docReadOnly: t.Doc.ReadOnly, readOnly: t.Editor.ReadOnly}
	binding.handler = func(action editor.SourceAction) {
		if !a.sourceLanguageShortcutCurrent(binding) {
			return // A retained old callback cannot retire or message a successor.
		}
		if a.modalDepth > 0 || a.ShowSettings || a.commandPalette.open || a.openDocuments.open {
			a.report(errors.New("当前面板或对话框暂不接受源码语言快捷键"))
			return
		}
		// Editor performed pure input admission before invoking us. Keep the
		// explicit command's own live checks and its exact one-slot queue path.
		// Each call is terminal: no stale Host writes after callback reentry.
		switch action {
		case editor.SourceActionComplete:
			a.runSourceCompletionCommand()
			return
		case editor.SourceActionReferences:
			a.runSourceLanguageCommand("references")
			return
		case editor.SourceActionRename:
			a.runSourceRenameCommand()
			return
		}
	}
	s.shortcut = binding
	binding.lease = t.Editor.SetSourceActionHandler(binding.handler)
}

func (a *App) retireSourceLanguage(t *Tab) {
	a.revokeSourceLanguageShortcut(t)
	s := &a.sourceLanguage
	if t != nil && (s.owner == nil || s.owner.tab != t) && (s.pending == nil || s.pending.owner.tab != t) && s.presentationTab != t {
		return
	}
	a.releaseSourceRename()
	s.activation++
	s.request++
	s.owner, s.pending = nil, nil
	s.completionFocus = false
	s.result = code.ServiceResult{}
	s.failure = nil
	s.intent = sourceLanguageIntent{}
	s.analysisVersion = 0
	s.status, s.reason, s.message = sourceLanguageStale, "source-owner-changed", ""
	if s.task != nil {
		s.task.abandon()
	}
	a.invalidateSourceLanguageRows()
	if t != nil && s.presentationTab == t || a.shuttingDown.Load() {
		a.closeSourceLanguagePresentation()
	}
}

func (a *App) resetSourceLanguageSession() {
	a.retireSourceLanguage(nil)
	a.closeSourceLanguagePresentation()
	a.sourceLanguage.session++
}

func (a *App) cancelSourceLanguage() {
	a.retireSourceLanguage(nil)
	a.sourceLanguage.status, a.sourceLanguage.reason = string(code.ServiceCanceled), "canceled-by-host"
}

// sourceLanguageInputError is deliberately input-only. Route, readonly and
// byte-budget checks remain at their own later admission boundaries.
func (a *App) sourceLanguageInputError() error {
	if d := a.sourceLanguage.rename; d != nil && d.field != nil {
		if err := d.field.InputActionError(); err != nil {
			return fmt.Errorf("请先完成或取消名称输入：%w", err)
		}
	}
	t := a.Current()
	if t == nil || !t.textCapable() || t.Editor == nil {
		return nil
	}
	if err := t.Editor.InputActionError(); err != nil {
		return fmt.Errorf("请先完成或取消源码输入：%w", err)
	}
	return nil
}

func languageReadCommandAvailable(a *App) bool {
	if a == nil {
		return false
	}
	_, _, err := a.sourceLanguageAdmission()
	return err == nil
}

func languageGoCommandAvailable(a *App) bool {
	if a == nil {
		return false
	}
	profile, ok := a.configuredSourceFoldProfile(a.Current())
	return ok && profile == code.StructureGo
}

func languageCancelCommandAvailable(a *App) bool {
	if a == nil {
		return false
	}
	s := &a.sourceLanguage
	return s.task != nil || s.pending != nil || s.owner != nil
}

func (a *App) sourceLanguageFocusStamp() sourceLanguageFocusStamp {
	return sourceLanguageFocusStamp{
		findGeneration:       a.findGeneration,
		findFocusGeneration:  a.findFocusGeneration,
		findFocusIntentEpoch: a.findFocusIntentEpoch,
		goToLineGeneration:   a.goToLineGeneration,
		findFocus:            a.findFocus,
	}
}

func (a *App) sourceLanguageDispatcher() func(func()) {
	if a.sourceLanguageHooks != nil && a.sourceLanguageHooks.dispatch != nil {
		return a.sourceLanguageHooks.dispatch
	}
	if a.Window != nil {
		return a.Window.Update
	}
	return nil
}

func (a *App) failSourceLanguage(status, reason string, err error) error {
	a.retireSourceLanguage(nil)
	a.sourceLanguage.status, a.sourceLanguage.reason, a.sourceLanguage.failure = status, reason, err
	if status == sourceLanguageStale {
		a.Message = "语言结果已过期；请重新分析当前源码"
	}
	return err
}

func (a *App) sourceLanguageAdmission() (*Tab, code.StructureProfile, error) {
	t := a.Current()
	if a.shuttingDown.Load() || a.restoring || a.modalDepth > 0 {
		return nil, "", errors.New("语言宿主正在关闭、恢复或显示对话框")
	}
	if t == nil || !a.tabIsOpen(t) || !t.textCapable() || t.Doc == nil || t.Editor == nil ||
		t.Doc.Buffer == nil || t.Editor.Buffer != t.Doc.Buffer || t.ByteInspector || (t.Mode != "source" && t.Mode != "split") {
		return nil, "", errors.New("当前标签没有可用的源码语言服务")
	}
	profile, ok := a.configuredSourceFoldProfile(t)
	if !ok || profile != t.sourceLanguageProfile || profile == "" {
		return nil, "", errors.New("当前实际格式路由不支持 Go 或严格 JSON 语言服务")
	}
	return t, profile, nil
}

func (a *App) sourceLanguageOwnerValid(owner *sourceLanguageOwner, requireSelection bool) error {
	if owner == nil || a.shuttingDown.Load() || a.restoring {
		return errors.New("语言请求目标已失效")
	}
	s, t := &a.sourceLanguage, owner.tab
	if s.session != owner.session || s.activation != owner.activation || s.request != owner.request ||
		a.Current() != t || !a.tabIsOpen(t) || t.Doc != owner.doc || t.Editor != owner.input ||
		t.Doc.Buffer != owner.buffer || t.Editor.Buffer != owner.buffer || a.pluginManager != owner.manager ||
		t.sourceLanguageRoute != owner.route || t.sourceLanguageProfile != owner.profile || t.Mode != owner.mode ||
		t.Doc.Path != owner.path || t.Name != owner.name || t.Doc.Encoding != owner.encoding || t.Doc.BOM != owner.bom ||
		t.Doc.ReadOnly != owner.docReadOnly || t.Editor.ReadOnly != owner.editorReadOnly || t.ByteInspector {
		return errors.New("语言请求目标已变化，请重新分析")
	}
	profile, ok := a.configuredSourceFoldProfile(t)
	if !ok || profile != owner.profile {
		return errors.New("语言请求格式路由已变化，请重新分析")
	}
	return owner.input.CheckSourceTarget(owner.target, requireSelection)
}

// canonicalSourceLanguageOperation returns the actual constants so even short
// caller substrings cannot hold a large parent string in the host.
func canonicalSourceLanguageOperation(operation code.ServiceOperation) code.ServiceOperation {
	switch operation {
	case code.ServiceDiagnostics:
		return code.ServiceDiagnostics
	case code.ServiceSymbols:
		return code.ServiceSymbols
	case code.ServiceCompletion:
		return code.ServiceCompletion
	case code.ServiceDefinition:
		return code.ServiceDefinition
	case code.ServiceReferences:
		return code.ServiceReferences
	case code.ServiceInfo:
		return code.ServiceInfo
	case code.ServiceRename:
		return code.ServiceRename
	default:
		return "" // AnalyzeService reports unsupported operation.
	}
}

// requestSourceLanguage is B1's internal UI admission boundary. B2 supplies the
// explicit command/presentation layer; no automatic analyzer is installed here.
func (a *App) requestSourceLanguage(intent sourceLanguageIntent) error {
	return a.admitSourceLanguageRequest(intent, a.launchSourceLanguage)
}

func (a *App) admitSourceLanguageRequest(intent sourceLanguageIntent, launch func(*sourceLanguageOwner, sourceLanguageIntent, string, func(func()))) error {
	// InputActionError is a pure, input-only gate. It must precede route,
	// snapshot, retirement, focus, or any other effect so a first refused
	// markless preedit remains owned by its original native client.
	if err := a.sourceLanguageInputError(); err != nil {
		return err
	}
	if intent.rename != nil {
		return a.admitSourceRename(intent.rename, launch)
	}
	if a.sourceLanguage.rename != nil {
		if err := a.sourceRenameInputError(); err != nil {
			return err
		}
		a.releaseSourceRename()
	}
	dispatch := a.sourceLanguageDispatcher()
	if dispatch == nil {
		return a.failSourceLanguage(sourceLanguageHostError, "ui-dispatch-unavailable", errors.New("当前宿主没有可用的 UI 排队调度器"))
	}
	t, profile, err := a.sourceLanguageAdmission()
	if err != nil {
		return a.failSourceLanguage(string(code.ServiceUnsupported), "route-or-view", err)
	}
	if len(intent.prefix) > sourceLanguageFieldBytes || len(intent.newName) > sourceLanguageFieldBytes {
		err := errors.New("语言请求字段超过 16 KiB 上限")
		a.failSourceLanguage(string(code.ServiceLimit), "field-bytes", err)
		a.sourceLanguage.result = code.ServiceResult{Status: code.ServiceLimit, Reason: "resource-limit", Resource: "field-bytes", Limit: sourceLanguageFieldBytes}
		return err
	}
	maxBytes := min(intent.limits.Structure.Bytes, code.DefaultServiceLimits().Structure.Bytes)
	if maxBytes <= 0 {
		return a.failSourceLanguage(string(code.ServiceInvalid), "invalid-limits", errors.New("语言请求必须声明正的源码预算"))
	}
	if t.Doc.Buffer.LenBytes() > maxBytes {
		err := errors.New("语言请求源码超过字节预算")
		a.failSourceLanguage(string(code.ServiceLimit), "source-bytes", err)
		a.sourceLanguage.result = code.ServiceResult{Status: code.ServiceLimit, Reason: "resource-limit", Resource: code.StructureBytes, Limit: maxBytes}
		return err
	}
	if intent.completion != nil {
		if err := sourceCompletionSelectionValid(t, intent.completion); err != nil {
			return err
		}
		intent.completion.focusStamp = a.sourceLanguageFocusStamp()
		copy := *intent.completion
		copy.oldText = strings.Clone(copy.oldText)
		intent.completion = &copy
	}
	// Own small caller strings without retaining an arbitrarily large parent.
	intent.operation = canonicalSourceLanguageOperation(intent.operation)
	intent.prefix, intent.newName = strings.Clone(intent.prefix), strings.Clone(intent.newName)
	s := &a.sourceLanguage
	if s.task != nil {
		// Frozen B0 issues an opaque token only together with a source snapshot.
		// Reuse the already admitted read target; never materialize a second
		// source while the physical slot is occupied. Changed source/attachment
		// is explicitly stale and requires a fresh request after retirement.
		if err := a.sourceLanguageOwnerValid(s.owner, intent.completion != nil); err != nil {
			return a.failSourceLanguage(sourceLanguageStale, "pending-target-changed", err)
		}
		owner := *s.owner
		s.request++
		owner.request = s.request
		s.owner = &owner
		if intent.hasUTF16Offset {
			// Keep only the exact admitted UTF-16 scalar; the physical slot is
			// still occupied, so do not capture or materialize another source.
			intent.byteOffset = 0
		}
		s.pending = &sourceLanguagePending{owner: &owner, intent: intent, dispatch: dispatch}
		s.completionFocus = false
		s.intent = intent
		s.analysisVersion = owner.version
		s.result, s.failure = code.ServiceResult{}, nil
		s.status, s.reason = sourceLanguageRunning, "latest-pending"
		a.invalidateSourceLanguageRows()
		s.task.abandon()
		return nil
	}
	snapshot, err := t.Editor.CaptureSourceSnapshot(maxBytes)
	if err != nil {
		return a.failSourceLanguage(sourceLanguageStale, "source-input-unavailable", err)
	}
	if intent.hasUTF16Offset {
		intent.byteOffset, err = sourceLanguageByteOffset(snapshot.Source, intent.utf16Offset, intent.hasSelectedRange)
		if err != nil {
			a.retireSourceLanguage(nil)
			s.status, s.reason = string(code.ServiceInvalid), "invalid-offset"
			s.result = code.ServiceResult{Status: code.ServiceInvalid, Reason: "invalid-offset"}
			return err
		}
	}
	s.request++
	owner := &sourceLanguageOwner{session: s.session, activation: s.activation, request: s.request, route: t.sourceLanguageRoute,
		tab: t, doc: t.Doc, input: t.Editor, buffer: t.Doc.Buffer, manager: a.pluginManager, target: snapshot.Target,
		profile: profile, mode: t.Mode, path: t.Doc.Path, name: t.Name, encoding: t.Doc.Encoding, bom: t.Doc.BOM,
		docReadOnly: t.Doc.ReadOnly, editorReadOnly: t.Editor.ReadOnly, version: snapshot.Version}
	launch(owner, intent, snapshot.Source, dispatch)
	return nil
}

func (a *App) launchSourceLanguage(owner *sourceLanguageOwner, intent sourceLanguageIntent, source string, dispatch func(func())) {
	if intent.completion != nil {
		// Prove local token/range facts against this one admitted materialization.
		// Proof failure is not a syntax diagnostic: always run the real service.
		copy := *intent.completion
		copy.proven, copy.proofFailure = proveSourceCompletion(source, owner.profile, &copy)
		intent.completion = &copy
	}
	s := &a.sourceLanguage
	task := newSourceLanguageTask(owner.request, code.ServiceRequest{Source: source, Profile: owner.profile,
		Operation: intent.operation, ByteOffset: intent.byteOffset, Prefix: intent.prefix, NewName: intent.newName, Limits: intent.limits})
	s.owner, s.pending, s.task = owner, nil, task
	s.completionFocus = false
	s.intent, s.analysisVersion = intent, owner.version
	s.result, s.failure, s.message = code.ServiceResult{}, nil, ""
	s.status, s.reason = sourceLanguageRunning, ""
	a.invalidateSourceLanguageRows()
	// Capture the genuine dispatcher and an ID-only UI callback at admission.
	// Workers do not call App.update (whose no-window fallback is inline).
	notify := func(id uint64) { dispatch(func() { a.completeSourceLanguage(id) }) }
	var hooks sourceLanguageHooks
	if a.sourceLanguageHooks != nil {
		hooks = *a.sourceLanguageHooks
		hooks.dispatch = nil
	}
	go task.run(notify, hooks)
}

func (a *App) completeSourceLanguage(id uint64) {
	s := &a.sourceLanguage
	task := s.task
	if task == nil || task.id != id {
		return // A late/duplicate notice cannot release a replacement slot.
	}
	out, ready := task.take()
	if !ready {
		return
	}
	s.task = nil
	if a.shuttingDown.Load() {
		task.abandon()
		return
	}
	if pending := s.pending; pending != nil {
		s.pending = nil
		if pending.intent.rename != nil {
			// Rename refusal retires only the obsolete preview. A live name or
			// source native input must keep its field and cancellation route.
			if err := a.sourceRenameRequestValid(pending.intent.rename); err != nil {
				a.rejectSourceRename(pending.intent.rename, err)
				return
			}
		} else if err := a.sourceLanguageOwnerValid(pending.owner, pending.intent.completion != nil); err != nil {
			a.failSourceLanguage(sourceLanguageStale, "pending-target-changed", err)
			return
		}
		// Only now is a new immutable source snapshot allowed. Check both the
		// original opaque target and the fresh capture, never just equal bytes.
		snapshot, err := pending.owner.input.CaptureSourceSnapshot(min(pending.intent.limits.Structure.Bytes, code.DefaultServiceLimits().Structure.Bytes))
		if err != nil {
			a.failSourceLanguage(sourceLanguageStale, "pending-capture-failed", err)
			return
		}
		intent := pending.intent
		if intent.hasUTF16Offset {
			intent.byteOffset, err = sourceLanguageByteOffset(snapshot.Source, intent.utf16Offset, intent.hasSelectedRange)
			if intent.rename != nil {
				if err == nil && intent.byteOffset != intent.rename.dialog.byteOffset {
					err = errors.New("重命名原始字节位置已变化")
				}
				if err != nil {
					a.rejectSourceRename(intent.rename, err)
					return
				}
			}
			if err != nil {
				a.retireSourceLanguage(nil)
				s.status, s.reason = string(code.ServiceInvalid), "invalid-offset"
				s.result = code.ServiceResult{Status: code.ServiceInvalid, Reason: "invalid-offset"}
				return
			}
		}
		a.launchSourceLanguage(pending.owner, intent, snapshot.Source, pending.dispatch)
		return
	}
	if s.intent.rename != nil {
		a.completeSourceRename(id, out.result, out.err)
		return
	}
	if s.owner == nil || s.owner.request != id {
		return
	}
	if err := a.sourceLanguageOwnerValid(s.owner, s.intent.completion != nil); err != nil {
		a.failSourceLanguage(sourceLanguageStale, "completion-target-changed", err)
		return
	}
	if out.err != nil {
		a.failSourceLanguage(sourceLanguageHostError, "analyzer-failed", out.err)
		return
	}
	if err := validateSourceLanguageResult(out.result); err != nil {
		a.failSourceLanguage(sourceLanguageHostError, "invalid-service-payload", err)
		return
	}
	s.result, s.status, s.reason = out.result, string(out.result.Status), out.result.Reason
	if out.result.Status != code.ServiceComplete && out.result.Status != code.ServiceInvalid {
		// Failed results contain only status/reason/resource, no usable target.
		s.owner = nil
	}
	if s.intent.completion != nil {
		if out.result.Status != code.ServiceComplete {
			s.intent.completion = nil
		} else if err := validateSourceCompletionResult(s.owner, s.intent.completion, out.result); err != nil {
			a.failSourceLanguage(sourceLanguageHostError, "completion-not-applicable", err)
			return
		}
	}
	a.invalidateSourceLanguageRows()
	if s.intent.completion != nil && len(s.result.Completion) > 0 && a.sourceLanguageInputError() == nil && !a.commandPalette.composing && !a.openDocuments.composing && !a.findCompositionActive() && !a.GoToLineComposing && !a.sourceCompositionActive(a.Current()) && a.sourceLanguageFocusStamp() == s.intent.completion.focusStamp {
		s.completionFocus = true
	}
}

func validateSourceLanguageResult(result code.ServiceResult) error {
	switch result.Status {
	case code.ServiceComplete:
		return nil
	case code.ServiceInvalid, code.ServiceUnsupported, code.ServiceAmbiguous, code.ServiceLimit, code.ServiceCanceled:
		if len(result.Symbols) != 0 || len(result.Completion) != 0 || result.Definition != nil || len(result.References) != 0 || result.Info != nil || len(result.Edits) != 0 || result.NoOp ||
			(result.Status != code.ServiceInvalid && len(result.Diagnostics) != 0) {
			return errors.New("语言服务失败结果含有成功载荷")
		}
		return nil
	default:
		return fmt.Errorf("未知语言服务状态 %q", result.Status)
	}
}

func (a *App) stopSourceLanguage() *sourceLanguageTask {
	task := a.sourceLanguage.task
	a.cancelSourceLanguage()
	a.closeSourceLanguagePresentation()
	return task
}
