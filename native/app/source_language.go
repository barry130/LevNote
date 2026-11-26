package app

import (
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

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

// Superseded continuations are control flow, not an error to present through a
// callback-installed successor. Actual input/service errors remain unchanged.
var errSourceLanguageSuperseded = errors.New("语言操作的原始目标已被回调替换")

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
	automatic  *sourceAutoDescriptor
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
	auto                                        sourceAutoState
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
	defer a.bindSourceInput(t)
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

func (a *App) retireSourceLanguage(t *Tab) bool {
	a.revokeSourceLanguageShortcut(t)
	s := &a.sourceLanguage
	var cancelAuto func()
	if t == nil || s.auto.desired != nil && s.auto.desired.binding.tab == t || s.intent.automatic != nil && s.intent.automatic.binding.tab == t {
		a.invalidateSourceAutoState(nil)
		cancelAuto = a.prepareCancelSourceAutoTimer()
	}
	finish := func() bool {
		current := a.sourceLanguageContinuation()
		if cancelAuto != nil {
			cancelAuto()
		}
		return current()
	}
	if t != nil && (s.owner == nil || s.owner.tab != t) && (s.pending == nil || s.pending.owner.tab != t) && s.presentationTab != t {
		return finish()
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
	return finish()
}

func (a *App) resetSourceLanguageSession() bool {
	if !a.retireSourceLanguage(nil) {
		return false
	}
	if !a.closeSourceLanguagePresentation() {
		return false
	}
	a.sourceLanguage.session++
	return true
}

func (a *App) cancelSourceLanguage() bool {
	s := &a.sourceLanguage
	s.auto.manual = true
	s.auto.cancel++
	s.auto.cancelBinding, s.auto.cancelFloor = nil, text.ChangeStamp{}
	if t := a.Current(); t != nil && t.Doc != nil && t.Doc.Buffer != nil {
		s.auto.cancelBinding, s.auto.cancelFloor = t.inputBinding, t.Doc.Buffer.ChangeStamp()
	}
	request := s.request + 1
	if !a.retireSourceLanguage(nil) {
		return false
	}
	if s.request != request || !s.auto.manual || s.auto.desired != nil {
		return false
	}
	s.status, s.reason = string(code.ServiceCanceled), "canceled-by-host"
	return true
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
	return s.task != nil || s.pending != nil || s.owner != nil || s.auto.desired != nil
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

// Capture one synchronous Host continuation without acquiring source text or a
// target. Its caller first commits its own state, then invokes the seam, then
// returns immediately if a callback installed any successor authority.
func (a *App) sourceLanguageContinuation() func() bool {
	s := &a.sourceLanguage
	session, activation, request, presentation := s.session, s.activation, s.request, s.presentation
	sequence, cancel := s.auto.sequence, s.auto.cancel
	t, manager := a.Current(), a.pluginManager
	restoring, shuttingDown, config := a.restoring, a.shuttingDown.Load(), a.configDir
	var sessionCurrent func() bool
	if !shuttingDown {
		sessionCurrent = a.sessionSeal(true)
	}
	stateCurrent := func() bool {
		return a.restoring == restoring && a.shuttingDown.Load() == shuttingDown && a.configDir == config &&
			(sessionCurrent == nil || sessionCurrent()) && s.session == session && s.activation == activation && s.request == request && s.presentation == presentation &&
			s.auto.sequence == sequence && s.auto.cancel == cancel && a.Current() == t && a.pluginManager == manager
	}
	if t == nil {
		return stateCurrent
	}
	id, doc, input, view, adapter, binding := t.ID, t.Doc, t.Editor, t.View, t.Adapter, t.inputBinding
	boundCurrent := binding != nil && binding.input.InputBindingCurrent(binding.lease)
	epoch, route, profile, mode, name := t.inputEpoch, t.sourceLanguageRoute, t.sourceLanguageProfile, t.Mode, t.Name
	generation, inspector := t.OutlineFactoryGeneration, t.ByteInspector
	binary := t.Binary
	var binaryToken document.BinaryToken
	if binary != nil {
		binaryToken = binary.Token()
	}
	binaryPath, binaryKind, binarySize, binaryGeneration := t.BinaryPath, t.BinaryKind, t.BinarySize, t.BinaryGeneration
	formatHostEpoch, formatHostActive := t.formatHostEpoch, t.formatHostActive
	var docBuffer, inputBuffer *text.Buffer
	var docStamp, inputStamp text.ChangeStamp
	var path string
	var encoding document.Encoding
	var docVersion, docSelection, inputVersion, inputSelection uint64
	var bom, docReadOnly, readOnly, blocked bool
	if doc != nil {
		docBuffer, path, encoding, bom, docReadOnly = doc.Buffer, doc.Path, doc.Encoding, doc.BOM, doc.ReadOnly
		if docBuffer != nil {
			docStamp, docVersion, docSelection = docBuffer.ChangeStamp(), docBuffer.Version(), docBuffer.SelectionGeneration()
		}
	}
	if input != nil {
		inputBuffer = input.Buffer
		readOnly, blocked = input.ReadOnly, input.InputActionError() != nil
		if inputBuffer != nil {
			inputStamp, inputVersion, inputSelection = inputBuffer.ChangeStamp(), inputBuffer.Version(), inputBuffer.SelectionGeneration()
		}
	}
	return func() bool {
		return stateCurrent() && a.tabIsOpen(t) && t.ID == id && t.Doc == doc && t.Editor == input && t.View == view && t.Adapter == adapter &&
			t.inputBinding == binding && t.inputEpoch == epoch && t.sourceLanguageRoute == route && t.sourceLanguageProfile == profile &&
			t.Mode == mode && t.Name == name && t.OutlineFactoryGeneration == generation && t.ByteInspector == inspector &&
			(doc == nil || doc.Buffer == docBuffer && doc.Path == path && doc.Encoding == encoding && doc.BOM == bom && doc.ReadOnly == docReadOnly) &&
			(docBuffer == nil || docBuffer.ChangeStamp() == docStamp && docBuffer.Version() == docVersion && docBuffer.SelectionGeneration() == docSelection) &&
			(input == nil || input.Buffer == inputBuffer && input.ReadOnly == readOnly && (input.InputActionError() != nil) == blocked) &&
			(inputBuffer == nil || inputBuffer.ChangeStamp() == inputStamp && inputBuffer.Version() == inputVersion && inputBuffer.SelectionGeneration() == inputSelection) &&
			(doc == nil || input == nil || docBuffer == inputBuffer) &&
			t.Binary == binary && t.BinaryPath == binaryPath && t.BinaryKind == binaryKind && t.BinarySize == binarySize &&
			t.BinaryGeneration == binaryGeneration && (binary == nil || binary.SameBaseline(binaryToken)) &&
			t.formatHostEpoch == formatHostEpoch && t.formatHostActive == formatHostActive &&
			(binding == nil || binding.input.InputBindingCurrent(binding.lease) == boundCurrent)
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
	s := &a.sourceLanguage
	request, sequence := s.request+1, s.auto.sequence+1
	if !a.retireSourceLanguage(nil) {
		return errSourceLanguageSuperseded
	}
	if s.request != request || s.auto.sequence != sequence {
		return errSourceLanguageSuperseded
	}
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
	// Explicit work owns presentation and the sole slot. Retire automatic
	// publication before admission; a physical invocation still runs to done.
	s := &a.sourceLanguage
	previousOwner := s.owner
	s.auto.explicit, s.auto.explicitPresentation = true, s.presentation
	a.invalidateSourceAutoState(nil)
	cancelAuto := a.prepareCancelSourceAutoTimer()
	current := a.sourceLanguageContinuation()
	cancelAuto()
	if !current() {
		return errSourceLanguageSuperseded
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
	if s.task != nil {
		// Frozen B0 issues an opaque token only together with a source snapshot.
		// Reuse the already admitted read target; never materialize a second
		// source while the physical slot is occupied. Changed source/attachment
		// is explicitly stale and requires a fresh request after retirement.
		currentOwner := s.owner
		if currentOwner == nil {
			currentOwner = previousOwner
		}
		if err := a.sourceLanguageOwnerValid(currentOwner, intent.completion != nil); err != nil {
			return a.failSourceLanguage(sourceLanguageStale, "pending-target-changed", err)
		}
		owner := *currentOwner
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
	if d := s.intent.automatic; d != nil {
		a.completeSourceAuto(d, out)
		return
	}
	if s.auto.desired != nil && s.owner == nil {
		d := s.auto.desired
		if !a.sourceAutoNow().Before(d.due) {
			a.trySourceAuto(d)
		} else if effect := a.prepareSourceAuto(); effect != nil {
			effect()
		}
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
	for _, t := range a.Tabs {
		a.revokeSourceInput(t)
	}
	a.closeSourceLanguagePresentation()
	return task
}

const sourceAutoDelay = 500 * time.Millisecond

// AutoServiceSymbols uses the real Symbols operation: AnalyzeService returns
// symbols for valid source and actual validation diagnostics for invalid source
// in that same invocation. No second diagnostics analysis or synthetic payload.
const AutoServiceSymbols = code.ServiceSymbols

type sourceInputBinding struct {
	tab                                   *Tab
	doc                                   *document.Document
	input                                 *editor.Editor
	buffer                                *text.Buffer
	lease                                 editor.InputBinding
	epoch, event                          uint64
	stamp                                 text.ChangeStamp
	delivery                              editor.InputNotificationID
	manager                               *plugins.Manager
	route                                 uint64
	profile                               code.StructureProfile
	mode, path, name                      string
	encoding                              document.Encoding
	bom, docReadOnly, readOnly, inspector bool
	view                                  editor.View
	adapter                               *editor.Adapter
	viewGeneration                        uint64
}

// Descriptors carry scalar authority only. There is no source, source target,
// selection or result while blocked, waiting for a deadline or a physical slot.
type sourceAutoDescriptor struct {
	binding                                           *sourceInputBinding
	stamp                                             text.ChangeStamp
	intent, activation, session, cancel, presentation uint64
	due                                               time.Time
}
type sourceAutoState struct {
	desired                     *sourceAutoDescriptor
	sequence, cancel            uint64
	timer                       autosaveState
	queued, blocked             bool
	manual                      bool
	explicit                    bool
	explicitPresentation        uint64
	cancelBinding, closeBinding *sourceInputBinding
	cancelFloor, closeFloor     text.ChangeStamp
}
type sourceAutoHooks struct {
	queued    bool // explicit contract: dispatch enqueues, never invokes inline
	now       func() time.Time
	afterFunc func(time.Duration, func()) autosaveTimer
	dispatch  func(func()) // must enqueue; the timer's UI-entry guard rejects inline
}

// Recovery owns the exact live attachment even for inactive, plain or rich
// documents. Language authority adds presentation/route/metadata checks below.
func (a *App) sourceInputAttachmentCurrent(b *sourceInputBinding) bool {
	if b == nil || a.shuttingDown.Load() || a.restoring {
		return false
	}
	t := b.tab
	return t != nil && a.tabIsOpen(t) && t.inputBinding == b && t.inputEpoch == b.epoch &&
		t.Doc == b.doc && t.Editor == b.input && t.Doc.Buffer == b.buffer && t.Editor.Buffer == b.buffer &&
		b.input.InputBindingCurrent(b.lease)
}

func (a *App) sourceInputCurrent(b *sourceInputBinding) bool {
	if !a.sourceInputAttachmentCurrent(b) {
		return false
	}
	t := b.tab
	return a.pluginManager == b.manager && t.sourceLanguageRoute == b.route && t.sourceLanguageProfile == b.profile &&
		t.Mode == b.mode && t.Doc.Path == b.path && t.Name == b.name && t.Doc.Encoding == b.encoding &&
		t.Doc.BOM == b.bom && t.Doc.ReadOnly == b.docReadOnly && t.Editor.ReadOnly == b.readOnly &&
		t.ByteInspector == b.inspector && t.View == b.view && t.Adapter == b.adapter &&
		t.OutlineFactoryGeneration == b.viewGeneration && b.input.InputBindingCurrent(b.lease)
}

func (a *App) sourceInputConsumerCurrent(b *sourceInputBinding) bool {
	if b == nil || b.doc == nil || b.input == nil || b.buffer == nil {
		return false
	}
	t := b.tab
	if t == nil || !a.tabIsOpen(t) || t.inputBinding != b || t.inputEpoch != b.epoch ||
		t.Doc != b.doc || t.Editor != b.input || t.Doc.Buffer != b.buffer || t.Editor.Buffer != b.buffer ||
		!b.input.InputBindingCurrent(b.lease) {
		return false
	}
	return a.pluginManager == b.manager && t.sourceLanguageRoute == b.route && t.sourceLanguageProfile == b.profile &&
		t.Mode == b.mode && t.Doc.Path == b.path && t.Name == b.name && t.Doc.Encoding == b.encoding &&
		t.Doc.BOM == b.bom && t.Doc.ReadOnly == b.docReadOnly && t.Editor.ReadOnly == b.readOnly &&
		t.ByteInspector == b.inspector && t.View == b.view && t.Adapter == b.adapter &&
		t.OutlineFactoryGeneration == b.viewGeneration
}

func (a *App) revokeSourceInput(t *Tab) {
	if t == nil {
		return
	}
	b := t.inputBinding
	t.inputBinding = nil
	t.inputEpoch++
	if b != nil && b.input.InputBindingCurrent(b.lease) {
		b.input.BindInputTerminal(nil)
	}
}

func (a *App) bindSourceInput(t *Tab) {
	if t == nil || t.Doc == nil || t.Editor == nil || t.Doc.Buffer == nil || t.Editor.Buffer != t.Doc.Buffer || !a.tabIsOpen(t) {
		return
	}
	if a.sourceInputCurrent(t.inputBinding) {
		return
	}
	oldBinding := t.inputBinding
	var stopOld func()
	if oldBinding != nil && (a.sourceLanguage.auto.desired != nil && a.sourceLanguage.auto.desired.binding == oldBinding ||
		a.sourceLanguage.intent.automatic != nil && a.sourceLanguage.intent.automatic.binding == oldBinding) {
		a.invalidateSourceAutoState(oldBinding)
		stopOld = a.prepareCancelSourceAutoTimer()
	}
	a.revokeSourceInput(t)
	b := &sourceInputBinding{tab: t, doc: t.Doc, input: t.Editor, buffer: t.Doc.Buffer, epoch: t.inputEpoch,
		stamp: t.Doc.Buffer.ChangeStamp(), manager: a.pluginManager, route: t.sourceLanguageRoute, profile: t.sourceLanguageProfile,
		mode: t.Mode, path: t.Doc.Path, name: t.Name, encoding: t.Doc.Encoding, bom: t.Doc.BOM,
		docReadOnly: t.Doc.ReadOnly, readOnly: t.Editor.ReadOnly, inspector: t.ByteInspector,
		view: t.View, adapter: t.Adapter, viewGeneration: t.OutlineFactoryGeneration}
	t.inputBinding = b
	b.lease = b.input.BindInputTerminal(func(d editor.InputDelivery) { a.consumeSourceInput(b, d) })
	b.input.OnChange = func() {
		d, ok := b.input.LastInputDelivery() // capture before any Host effect
		if ok && d.Binding == b.lease && d.Stamp == b.buffer.ChangeStamp() {
			a.consumeSourceInput(b, d)
		} else {
			// Format Host.Changed owns its already completed Buffer transaction;
			// it has no Editor delivery. Observe the real full stamp directly.
			a.consumeSourceChange(b, nil)
		}
	}
	b.input.OnError = func(err error) {
		d, ok := b.input.LastInputDelivery() // error publication is last
		event := b.event
		if ok {
			if !a.consumeSourceInput(b, d) {
				return
			}
			event = b.event
		}
		if a.sourceInputAttachmentCurrent(b) && b.event == event {
			a.report(err)
		}
	}
	// Last effect: Stop may reenter and install different callbacks.
	if stopOld != nil {
		stopOld()
	}
}

// No callback follows the report in OnError, and no Editor write follows its
// captured OnChange. Recovery applies to every live document, independently of
// active language eligibility. Both timer generations commit before any seam.
func (a *App) consumeSourceInput(b *sourceInputBinding, d editor.InputDelivery) bool {
	return a.consumeSourceChange(b, &d)
}

func (a *App) provedSourceRollback(b *sourceInputBinding, d *editor.InputDelivery, before, live text.ChangeStamp) bool {
	if d == nil || d.Recovery != editor.InputRecoveryRollback ||
		(d.Disposition != editor.InputSame && d.Disposition != editor.InputRealCommit) ||
		!a.sourceInputAttachmentCurrent(b) || d.Binding != b.lease || d.Editor != b.input || d.Buffer != b.buffer ||
		before.BaselineID != live.BaselineID || d.Stamp != live {
		return false
	}
	latest, ok := b.input.LastInputDelivery()
	return ok && latest == *d
}

func (a *App) heldAutosaveBoundary(lastEdit time.Time, persistPending bool) bool {
	s := &a.autosave
	s.mu.Lock()
	armed, due := s.armed, s.due
	s.mu.Unlock()
	return persistPending && armed && due.Equal(lastEdit.Add(autosaveDelay))
}

// nil delivery is only the existing complete format Host.Changed boundary. It
// samples real Buffer identity; it never fabricates an Editor delivery or C.
func (a *App) consumeSourceChange(b *sourceInputBinding, d *editor.InputDelivery) bool {
	if !a.sourceInputAttachmentCurrent(b) || d != nil && (d.Binding != b.lease || d.Editor != b.input || d.Buffer != b.buffer) {
		return false
	}
	// Capture this event's actual Host authority before either timer seam. A
	// stale language seal may still recover its own exact live attachment.
	t := b.tab
	view, adapter, viewGeneration := t.View, t.Adapter, t.OutlineFactoryGeneration
	manager, route, profile, mode := a.pluginManager, t.sourceLanguageRoute, t.sourceLanguageProfile, t.Mode
	path, name, encoding, bom := t.Doc.Path, t.Name, t.Doc.Encoding, t.Doc.BOM
	docReadOnly, readOnly, inspector, active := t.Doc.ReadOnly, t.Editor.ReadOnly, t.ByteInspector, a.Current()
	live := b.buffer.ChangeStamp()
	// A sticky duplicate never overrides an externally changed baseline. It
	// may still be consumed by the final error boundary to retire old work.
	if d != nil && b.delivery == d.Delivery && b.stamp == live && d.Stamp == live {
		return true
	}
	if d == nil && b.stamp == live {
		return true
	}
	before := b.stamp
	if d != nil {
		b.delivery = d.Delivery
	}
	b.stamp = live
	b.event++
	event := b.event
	invalid := before.BaselineID != live.BaselineID || d != nil &&
		(d.Stamp != live || d.Disposition == editor.InputUnprovedRetirement || d.Disposition == editor.InputRebaseInvalidation)
	real := !invalid && before.ConfirmedChangeID != live.ConfirmedChangeID
	var recover bool
	if !invalid {
		provisional := d != nil && d.Recovery == editor.InputRecoveryProvisional
		rollback := a.provedSourceRollback(b, d, before, live)
		heldBoundary := rollback && !real && a.heldAutosaveBoundary(a.lastEdit, a.persistPending)
		renewRollback := rollback && (real || !heldBoundary)
		recover = real || provisional || rollback
		if real || provisional || renewRollback {
			a.lastEdit = a.autosaveNow()
		}
		if recover {
			a.persistPending = true
		}
	}
	s, auto := &a.sourceLanguage, &a.sourceLanguage.auto
	// Presentation generations also cover open-close with no intervening frame.
	if auto.explicit && !s.visible && s.presentation != auto.explicitPresentation {
		auto.explicit = false
	}
	if invalid {
		a.invalidateSourceAutoState(b)
	} else if real {
		if auto.cancelBinding != b || auto.cancelFloor != live {
			auto.manual = false
		}
		a.invalidateSourceAutoState(nil)
		freshAfterClose := auto.closeBinding != b || auto.closeFloor != live
		if !auto.manual && freshAfterClose && !s.visible && !auto.explicit && a.sourceAutoEligible(b) {
			auto.sequence++
			auto.desired = &sourceAutoDescriptor{binding: b, stamp: live, intent: auto.sequence, due: a.sourceAutoNow().Add(sourceAutoDelay),
				activation: s.activation, session: s.session, cancel: auto.cancel, presentation: s.presentation}
		}
	}
	var recoveryEffect func()
	if recover {
		recoveryEffect = a.prepareAutosave()
	}
	autoEffect := a.prepareSourceAuto()
	sequence, cancel, presentation := auto.sequence, auto.cancel, s.presentation
	session, activation, request := s.session, s.activation, s.request
	current := func() bool {
		return a.sourceInputAttachmentCurrent(b) && b.event == event && b.stamp == live && b.buffer.ChangeStamp() == live &&
			t.View == view && t.Adapter == adapter && t.OutlineFactoryGeneration == viewGeneration && a.Current() == active &&
			a.pluginManager == manager && t.sourceLanguageRoute == route && t.sourceLanguageProfile == profile && t.Mode == mode &&
			t.Doc.Path == path && t.Name == name && t.Doc.Encoding == encoding && t.Doc.BOM == bom &&
			t.Doc.ReadOnly == docReadOnly && t.Editor.ReadOnly == readOnly && t.ByteInspector == inspector &&
			auto.sequence == sequence && auto.cancel == cancel && s.presentation == presentation &&
			s.session == session && s.activation == activation && s.request == request
	}
	if recoveryEffect != nil {
		recoveryEffect()
		if !current() {
			return false
		}
	}
	if autoEffect != nil {
		autoEffect()
		if !current() {
			return false
		}
	}
	return current()
}

func (a *App) sourceAutoNow() time.Time {
	if h := a.sourceAutoHooks; h != nil && h.now != nil {
		return h.now()
	}
	return time.Now()
}
func (a *App) sourceAutoEligible(b *sourceInputBinding) bool {
	if !a.sourceInputCurrent(b) || a.Current() != b.tab {
		return false
	}
	t := b.tab
	if !t.textCapable() || t.ByteInspector || (t.Mode != "source" && t.Mode != "split") {
		return false
	}
	profile, ok := a.configuredSourceFoldProfile(t)
	return ok && profile != "" && profile == b.profile
}
func (a *App) sourceAutoCurrent(d *sourceAutoDescriptor) bool {
	s := &a.sourceLanguage
	return d != nil && s.auto.desired == d && !s.auto.manual && !s.auto.explicit && !s.visible && s.presentation == d.presentation &&
		s.auto.cancel == d.cancel && s.session == d.session && s.activation == d.activation &&
		a.sourceAutoEligible(d.binding) && d.binding.buffer.ChangeStamp() == d.stamp
}

// This phase is callback-free; callers reserve timer cancellation before seams.
func (a *App) invalidateSourceAutoState(b *sourceInputBinding) {
	s, auto := &a.sourceLanguage, &a.sourceLanguage.auto
	if b != nil && (auto.desired == nil || auto.desired.binding != b) && (s.intent.automatic == nil || s.intent.automatic.binding != b) {
		return
	}
	auto.desired = nil
	auto.queued, auto.blocked = false, false
	auto.sequence++
	if s.intent.automatic != nil {
		if s.task != nil {
			s.task.abandon()
		}
		s.owner = nil
		s.result, s.failure = code.ServiceResult{}, nil
		s.intent = sourceLanguageIntent{}
		s.analysisVersion = 0
		s.status, s.reason = sourceLanguageStale, "automatic-retired"
		a.invalidateSourceLanguageRows()
	}
}

func (a *App) prepareSourceAuto() func() {
	s := &a.sourceLanguage
	auto, d := &s.auto, s.auto.desired
	if d == nil || !a.sourceAutoCurrent(d) {
		if d != nil {
			a.invalidateSourceAutoState(nil)
		}
		return a.prepareCancelSourceAutoTimer()
	}
	if auto.queued || s.task != nil && s.intent.automatic == d {
		return nil
	}
	// Preedit blocks without capture, Flush, or a periodic retry timer.
	if a.modalDepth > 0 || d.binding.input.InputActionError() != nil {
		auto.blocked = true
		return a.prepareCancelSourceAutoTimer()
	}
	auto.blocked = false
	dispatch := a.sourceAutoDispatcher()
	if dispatch == nil {
		s.status, s.reason = sourceLanguageHostError, "automatic-ui-dispatch-unavailable"
		return a.prepareCancelSourceAutoTimer()
	}
	// A proven terminal event resumes the same due descriptor by posting UI
	// work, rather than starting a zero-delay timer or running inside Editor.
	if !a.sourceAutoNow().Before(d.due) {
		cancel := a.prepareCancelSourceAutoTimer()
		auto.queued = true
		return func() {
			cancel()
			if !a.sourceAutoCurrent(d) || !auto.queued {
				return
			}
			if !sourceAutoDispatcherQueues(dispatch) {
				a.sourceAutoUnavailable(d)
				return
			}
			if !a.sourceAutoCurrent(d) || !auto.queued {
				return
			}
			queueSourceAuto(dispatch, func() {
				if !a.sourceAutoCurrent(d) || !auto.queued {
					return
				}
				auto.queued = false
				a.trySourceAuto(d)
			})
		}
	}
	after := func(delay time.Duration, fn func()) autosaveTimer { return time.AfterFunc(delay, fn) }
	if h := a.sourceAutoHooks; h != nil {
		after = h.afterFunc
	}
	timer := &auto.timer
	timer.mu.Lock()
	if timer.armed && timer.timer != nil && timer.due.Equal(d.due) {
		timer.mu.Unlock()
		return nil
	}
	old := timer.timer
	timer.request++
	request := timer.request
	timer.timer, timer.armed, timer.due = nil, true, d.due
	timer.mu.Unlock()
	delay := max(time.Duration(0), d.due.Sub(a.sourceAutoNow()))
	return func() {
		if old != nil {
			old.Stop()
		}
		if !a.sourceAutoCurrent(d) || !a.sourceAutoTimerCurrent(request) {
			return
		}
		if !sourceAutoDispatcherQueues(dispatch) {
			a.sourceAutoUnavailable(d)
			return
		}
		if !a.sourceAutoCurrent(d) || !a.sourceAutoTimerCurrent(request) {
			return
		}
		handle := after(delay, func() {
			// This dispatcher was admitted on UI. A fast genuine UI queue may
			// run before this timer-thread call returns, and must not be dropped.
			dispatch(func() { a.deliverSourceAuto(request) })
		})
		ownerCurrent := a.sourceAutoCurrent(d)
		timer.mu.Lock()
		current := ownerCurrent && timer.request == request && timer.armed && !a.shuttingDown.Load()
		if current {
			timer.timer = handle
		}
		timer.mu.Unlock()
		if !current && handle != nil {
			handle.Stop()
		}
	}
}

func (a *App) sourceAutoTimerCurrent(request uint64) bool {
	t := &a.sourceLanguage.auto.timer
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.request == request && t.armed
}
func (a *App) prepareCancelSourceAutoTimer() func() {
	t := &a.sourceLanguage.auto.timer
	t.mu.Lock()
	t.request++
	old := t.timer
	t.timer, t.armed, t.due = nil, false, time.Time{}
	t.mu.Unlock()
	return func() {
		if old != nil {
			old.Stop()
		}
	}
}

func (a *App) deliverSourceAuto(request uint64) {
	t := &a.sourceLanguage.auto.timer
	t.mu.Lock()
	if t.request != request || !t.armed {
		t.mu.Unlock()
		return
	}
	t.timer, t.armed, t.due = nil, false, time.Time{}
	t.mu.Unlock()
	d := a.sourceLanguage.auto.desired
	if !a.sourceAutoCurrent(d) {
		a.invalidateSourceAutoState(nil)
		return
	}
	if a.sourceAutoNow().Before(d.due) {
		if effect := a.prepareSourceAuto(); effect != nil {
			effect()
		}
		return
	}
	a.trySourceAuto(d)
}

func (a *App) trySourceAuto(d *sourceAutoDescriptor) {
	s := &a.sourceLanguage
	if !a.sourceAutoCurrent(d) {
		return
	}
	if a.modalDepth > 0 || d.binding.input.InputActionError() != nil {
		s.auto.blocked = true
		return
	}
	if s.task != nil {
		return
	} // only the descriptor survives a busy slot
	dispatch := a.sourceAutoDispatcher()
	if dispatch == nil {
		return
	}
	if !sourceAutoDispatcherQueues(dispatch) {
		a.sourceAutoUnavailable(d)
		return
	}
	if !a.sourceAutoCurrent(d) || d.binding.input.InputActionError() != nil {
		return
	}
	limits := code.DefaultServiceLimits()
	snapshot, err := d.binding.input.CaptureSourceSnapshot(limits.Structure.Bytes)
	if err != nil {
		a.invalidateSourceAutoState(nil)
		return
	}
	if !a.sourceAutoCurrent(d) {
		return
	}
	s.request++
	b := d.binding
	owner := &sourceLanguageOwner{session: s.session, activation: s.activation, request: s.request, route: b.route,
		tab: b.tab, doc: b.doc, input: b.input, buffer: b.buffer, manager: b.manager, target: snapshot.Target,
		profile: b.profile, mode: b.mode, path: b.path, name: b.name, encoding: b.encoding, bom: b.bom,
		docReadOnly: b.docReadOnly, editorReadOnly: b.readOnly, version: snapshot.Version}
	a.launchSourceLanguage(owner, sourceLanguageIntent{operation: AutoServiceSymbols, limits: limits, automatic: d}, snapshot.Source, dispatch)
}

func (a *App) completeSourceAuto(d *sourceAutoDescriptor, out sourceLanguageOutcome) {
	s := &a.sourceLanguage
	if !a.sourceAutoCurrent(d) {
		a.invalidateSourceAutoState(nil)
		return
	}
	if d.binding.input.InputActionError() != nil {
		// Do not cache a provisional-era result. Retain only the original
		// descriptor for terminal readiness and a fresh current target.
		s.auto.blocked = true
		s.owner, s.intent, s.result = nil, sourceLanguageIntent{}, code.ServiceResult{}
		return
	}
	if a.sourceLanguageOwnerValid(s.owner, false) != nil {
		s.owner, s.intent, s.result = nil, sourceLanguageIntent{}, code.ServiceResult{}
		// A proven unchanged stamp permits coordinate-free recapture only.
		a.trySourceAuto(d)
		return
	}
	err := out.err
	if err == nil {
		err = validateSourceLanguageResult(out.result)
	}
	if err != nil {
		s.owner, s.intent = nil, sourceLanguageIntent{}
		s.auto.desired = nil
		s.status, s.reason, s.failure = sourceLanguageHostError, "automatic-analysis-failed", err
		return
	}
	s.result, s.status, s.reason = out.result, string(out.result.Status), out.result.Reason
	if out.result.Status != code.ServiceComplete && out.result.Status != code.ServiceInvalid {
		s.owner = nil
	}
	// Keep a published seal, but no desired retry: readiness is not a fresh
	// edit, and a successful automatic analysis is never periodically rearmed.
	s.auto.desired = nil
	a.invalidateSourceLanguageRows()
}

// Only these dispatchers have an explicitly queued contract. App.uiDispatch
// and the headless App.update fallback are deliberately excluded.
func (a *App) sourceAutoDispatcher() func(func()) {
	if h := a.sourceAutoHooks; h != nil {
		if !h.queued || h.afterFunc == nil || h.dispatch == nil {
			return nil
		}
		return h.dispatch
	}
	if a.Window != nil {
		return a.Window.Update
	}
	return nil
}

// Probe only from UI, after both timer generations have committed. UI work
// cannot concurrently drain its own queue; an inline-only configuration is
// rejected before any real timer or snapshot. Do not run this return-order
// test on timer/worker threads, where a fast genuine UI queue is legitimate.
func sourceAutoDispatcherQueues(dispatch func(func())) bool {
	var posting, inline atomic.Bool
	posting.Store(true)
	dispatch(func() {
		if posting.Load() {
			inline.Store(true)
		}
	})
	posting.Store(false)
	return !inline.Load()
}

func (a *App) sourceAutoUnavailable(d *sourceAutoDescriptor) {
	if !a.sourceAutoCurrent(d) {
		return
	}
	a.sourceLanguage.status, a.sourceLanguage.reason = sourceLanguageHostError, "automatic-ui-dispatch-unavailable"
	a.sourceLanguage.auto.queued = false
	cancel := a.prepareCancelSourceAutoTimer()
	cancel()
}

// Immediate terminal readiness posts from the UI thread. Reject inline resume;
// timer/worker threads instead use their previously admitted dispatcher directly.
func queueSourceAuto(dispatch func(func()), fn func()) {
	var returned atomic.Bool
	dispatch(func() {
		if !returned.Load() {
			return
		}
		fn()
	})
	returned.Store(true)
}
