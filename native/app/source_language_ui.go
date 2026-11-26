package app

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/code"
)

const (
	sourceLanguageDisplayRows = 256
	sourceLanguageLabelBytes  = 512
)

type sourceLanguagePosition struct {
	line, column int
}

type sourceLanguageRow struct {
	id, label, detail string
	kind              string
	start, end        int // UTF-16 offsets in the owning captured source
	position          sourceLanguagePosition
}

type sourceLanguageAction struct {
	kind         string
	rowID        string
	request      uint64
	presentation uint64
	renameDialog *sourceRenameDialog
	renameEpoch  uint64
}

func sourceLanguageByteOffset(source string, offset int, selectedRange bool) (int, error) {
	if offset < 0 {
		return 0, errors.New("语言结果位置无效")
	}
	unit, byteAt := 0, 0
	for byteAt < len(source) {
		if unit == offset {
			break
		}
		r, width := utf8.DecodeRuneInString(source[byteAt:])
		if r == utf8.RuneError && width == 1 {
			return 0, errors.New("源码不是有效 UTF-8，无法解析 UTF-16 位置")
		}
		next := unit + 1
		if r > 0xffff {
			next++
		}
		if offset > unit && offset < next {
			return 0, errors.New("语言结果位置落在 UTF-16 代理对中间")
		}
		unit, byteAt = next, byteAt+width
	}
	if unit != offset {
		if offset == unit && byteAt == len(source) {
			return byteAt, nil
		}
		return 0, errors.New("语言结果位置超出源码范围")
	}
	if selectedRange {
		return byteAt, nil
	}
	if byteAt < len(source) {
		right, _ := utf8.DecodeRuneInString(source[byteAt:])
		if isSourceIdentifierPart(right) {
			return byteAt, nil
		}
	}
	if byteAt == 0 {
		return byteAt, nil
	}
	leftStart := byteAt - 1
	for leftStart > 0 && !utf8.RuneStart(source[leftStart]) {
		leftStart--
	}
	left, _ := utf8.DecodeRuneInString(source[leftStart:byteAt])
	if isSourceIdentifierPart(left) {
		return leftStart, nil
	}
	return byteAt, nil
}

func isSourceIdentifierPart(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '$' || r == '@'
}

func sourceLanguageBoundText(value string) string {
	if len(value) <= sourceLanguageLabelBytes {
		return value
	}
	limit := sourceLanguageLabelBytes - len("…")
	for limit > 0 && !utf8.RuneStart(value[limit]) {
		limit--
	}
	return value[:limit] + "…"
}

func sourceLanguageOperation(operation code.ServiceOperation) string {
	switch operation {
	case code.ServiceDiagnostics:
		return "分析"
	case code.ServiceSymbols:
		return "符号"
	case code.ServiceDefinition:
		return "定义"
	case code.ServiceReferences:
		return "引用"
	case code.ServiceInfo:
		return "信息"
	case code.ServiceCompletion:
		return "源码补全"
	case code.ServiceRename:
		return "本文件符号重命名"
	default:
		return string(operation)
	}
}

func (a *App) openSourceLanguagePresentation() error {
	if err := a.sourceLanguageInputError(); err != nil {
		return err
	}
	t := a.Current()
	if t == nil || !a.tabIsOpen(t) || t.Doc == nil || t.Editor == nil || t.Doc.Buffer == nil {
		return errors.New("当前没有可显示语言结果的文本标签")
	}
	s := &a.sourceLanguage
	if !(s.visible && s.presentationTab == t && s.presentationDoc == t.Doc && s.presentationInput == t.Editor && s.presentationBuffer == t.Doc.Buffer && s.presentationRoute == t.sourceLanguageRoute && s.presentationMode == t.Mode) {
		s.presentation++
		s.visible = true
		s.presentationTab, s.presentationDoc, s.presentationInput = t, t.Doc, t.Editor
		s.presentationBuffer, s.presentationRoute, s.presentationMode = t.Doc.Buffer, t.sourceLanguageRoute, t.Mode
		s.action = nil
		s.message = ""
		s.selected = -1
		a.invalidateSourceLanguageRows()
	}
	// Opening itself owns presentation, including direct/helper callers that
	// do not immediately start a request. Drop auto before any timer Stop seam.
	s.auto.explicit, s.auto.explicitPresentation = true, s.presentation
	physicalOwner, physicalIntent := s.owner, s.intent
	a.invalidateSourceAutoState(nil)
	if s.task != nil && physicalIntent.automatic != nil {
		// Keep only the already occupied slot's original authority so an
		// explicit request can reuse its B0 target without a second snapshot.
		// Its abandoned result stays empty and can never publish as explicit.
		s.owner, s.intent = physicalOwner, physicalIntent
	}
	cancelAuto := a.prepareCancelSourceAutoTimer()
	current := a.sourceLanguageContinuation()
	cancelAuto()
	if !current() || !a.sourceLanguagePresentationCurrent() {
		return errSourceLanguageSuperseded
	}
	return nil
}

func (a *App) closeSourceLanguagePresentation() bool {
	s := &a.sourceLanguage
	a.releaseSourceRename()
	s.presentation++
	s.visible = false
	s.presentationTab, s.presentationDoc, s.presentationInput, s.presentationBuffer = nil, nil, nil, nil
	s.presentationRoute, s.presentationMode = 0, ""
	s.action = nil
	s.completionFocus = false
	s.rows, s.cachedPositions = nil, nil
	s.rowsReady = false
	s.rowRequest, s.rowPresentation = 0, 0
	s.selected = -1
	if a.shuttingDown.Load() {
		s.listLifetime.close(&s.list)
	} else {
		s.listLifetime.reset(&s.list)
	}
	// Close consumes the current confirmed floor, without moving recovery or
	// replaying edits observed while explicit UI owned presentation.
	s.auto.explicit = false
	s.auto.explicitPresentation = s.presentation
	s.auto.closeBinding, s.auto.closeFloor = nil, text.ChangeStamp{}
	if t := a.Current(); t != nil && t.Doc != nil && t.Doc.Buffer != nil {
		s.auto.closeBinding, s.auto.closeFloor = t.inputBinding, t.Doc.Buffer.ChangeStamp()
	}
	a.invalidateSourceAutoState(nil)
	cancelAuto := a.prepareCancelSourceAutoTimer()
	current := a.sourceLanguageContinuation()
	cancelAuto() // final effect: a reentrant Stop owns its callback-exit state
	return current()
}

func (a *App) invalidateSourceLanguageRows() {
	s := &a.sourceLanguage
	s.rows, s.cachedPositions = nil, nil
	s.rowsReady = false
	s.rowRequest, s.rowPresentation = 0, 0
	s.selected = -1
	s.listLifetime.reset(&s.list)
}

// sourceLanguageDisplayMatches protects publication after a validity check or
// checked reveal callback. It only observes the captured UI-owned identities.
func (a *App) sourceLanguageDisplayMatches(owner *sourceLanguageOwner, request, presentation uint64) bool {
	s := &a.sourceLanguage
	return owner != nil && s.owner == owner && s.request == request &&
		s.presentation == presentation && a.sourceLanguagePresentationCurrent()
}

// retireStaleSourceLanguageDisplay checks a completed display only after pure
// input admission succeeds. Native preedit/refusal and foreign groups retain
// their result owner and cached rows; they are not proof of permanent staleness.
func (a *App) retireStaleSourceLanguageDisplay(owner *sourceLanguageOwner, request, presentation uint64) bool {
	if !a.sourceLanguageDisplayMatches(owner, request, presentation) || owner.input.InputActionError() != nil {
		return false
	}
	err := a.sourceLanguageOwnerValid(owner, false)
	if err == nil || !a.sourceLanguageDisplayMatches(owner, request, presentation) || owner.input.InputActionError() != nil {
		return false
	}
	// The captured display is still current. Retirement and the following
	// message publication invoke no host callbacks; no successor can be touched
	// by an old validity check returning across a callback boundary.
	a.retireSourceLanguage(nil)
	s := &a.sourceLanguage
	s.status, s.reason = sourceLanguageStale, "display-target-changed"
	s.message = "语言结果已过期；请重新分析当前源码"
	a.Message = s.message
	return true
}

func (a *App) beginSourceLanguageFrame() {
	s := &a.sourceLanguage
	s.listLifetime.begin(&s.list)
	a.checkSourceRenameFrame()
	if s.visible && (a.ShowPlugins || !a.sourceLanguagePresentationCurrent()) {
		a.retireSourceLanguage(s.presentationTab)
		return
	}
	if s.visible && s.owner != nil && (s.status == string(code.ServiceComplete) || s.status == string(code.ServiceInvalid)) {
		a.retireStaleSourceLanguageDisplay(s.owner, s.request, s.presentation)
	}
}

func (a *App) cancelHiddenSourceLanguageListFocus() {
	s := &a.sourceLanguage
	if !s.visible || a.ShowPlugins || !a.sourceLanguagePresentationCurrent() {
		s.list.Handle.CancelFocus()
	}
}

func (a *App) sourceLanguagePresentationCurrent() bool {
	s := &a.sourceLanguage
	t := s.presentationTab
	return s.visible && t != nil && a.Current() == t && a.tabIsOpen(t) && t.Doc == s.presentationDoc && t.Editor == s.presentationInput && t.Doc.Buffer == s.presentationBuffer && t.Editor.Buffer == s.presentationBuffer && t.sourceLanguageRoute == s.presentationRoute && t.Mode == s.presentationMode && !a.shuttingDown.Load() && !a.restoring
}

func (a *App) sourceLanguageReadinessMessage() string {
	s := &a.sourceLanguage
	status := code.ServiceStatus(s.status)
	if s.status == sourceLanguageHostError && s.failure != nil {
		return "宿主错误 · " + sourceLanguageBoundText(s.failure.Error())
	}
	switch status {
	case code.ServiceComplete:
		if s.intent.operation == code.ServiceDiagnostics {
			return "已完成语法分析；未发现语法诊断"
		}
		return "已完成 · " + sourceLanguageOperation(s.intent.operation)
	case code.ServiceInvalid:
		return "源码无效 · " + sourceLanguageBoundText(s.reason)
	case code.ServiceUnsupported:
		return "不支持 · " + sourceLanguageBoundText(s.reason)
	case code.ServiceAmbiguous:
		return "目标有歧义 · " + sourceLanguageBoundText(s.reason)
	case code.ServiceLimit:
		if s.reason == "go-validator-nesting" {
			return "Go 语法校验嵌套过深；校验器的数值阈值未提供"
		}
		if s.result.Resource != "" && s.result.Limit > 0 {
			return fmt.Sprintf("已达分析上限 · %s · %d", s.result.Resource, s.result.Limit)
		}
		return "已达分析上限 · " + sourceLanguageBoundText(s.reason)
	case code.ServiceCanceled:
		return "已取消 · " + sourceLanguageBoundText(s.reason)
	}
	switch s.status {
	case sourceLanguageRunning:
		if s.pending != nil {
			return "分析进行中；当前执行结束后运行最新请求"
		}
		return "正在分析"
	case sourceLanguageStale:
		return "结果已过期 · 请显式重新分析"
	case sourceLanguageHostError:
		if s.failure != nil {
			return "宿主错误 · " + sourceLanguageBoundText(s.failure.Error())
		}
		return "宿主错误 · " + sourceLanguageBoundText(s.reason)
	default:
		return "尚未分析"
	}
}

func sourceLanguageRowID(kind string, index, start, end int) string {
	return fmt.Sprintf("%s:%d:%d:%d", kind, index, start, end)
}

func (a *App) sourceLanguageRows() []sourceLanguageRow {
	s := &a.sourceLanguage
	if s.rename != nil {
		return a.sourceRenameRows()
	}
	if s.intent.completion != nil {
		return a.sourceCompletionRows()
	}
	if s.rowsReady && s.rowRequest == s.request && s.rowPresentation == s.presentation {
		return s.rows
	}
	s.rows, s.cachedPositions = nil, make(map[string]sourceLanguagePosition)
	s.rowsReady = true
	s.rowRequest, s.rowPresentation = s.request, s.presentation
	owner := s.owner
	if owner == nil || a.sourceLanguageOwnerValid(owner, false) != nil {
		return nil
	}
	appendSpan := func(kind, name, detail string, index, start, end int) {
		if len(s.rows) >= sourceLanguageDisplayRows {
			return
		}
		line, column, err := owner.buffer.LineColumn(start)
		if err != nil {
			return
		}
		pos := sourceLanguagePosition{line: line + 1, column: column + 1}
		id := sourceLanguageRowID(kind, index, start, end)
		label := fmt.Sprintf("%s · %d:%d", name, pos.line, pos.column)
		if detail != "" {
			label += " · " + detail
		}
		row := sourceLanguageRow{id: id, label: sourceLanguageBoundText(label), detail: sourceLanguageBoundText(detail), kind: kind, start: start, end: end, position: pos}
		s.rows = append(s.rows, row)
		s.cachedPositions[id] = pos
	}
	if len(s.result.Diagnostics) > 0 {
		for i, d := range s.result.Diagnostics {
			appendSpan("diagnostic", d.Code, d.Message, i, d.Span.UTF16.Start, d.Span.UTF16.End)
		}
		return s.rows
	}
	switch s.intent.operation {
	case code.ServiceSymbols:
		for i, symbol := range s.result.Symbols {
			appendSpan("symbol", symbol.Name, symbol.Kind, i, symbol.Span.UTF16.Start, symbol.Span.UTF16.End)
		}
	case code.ServiceDefinition:
		if d := s.result.Definition; d != nil {
			appendSpan("definition", d.Name, d.Kind, 0, d.Span.UTF16.Start, d.Span.UTF16.End)
		}
	case code.ServiceReferences:
		for i, ref := range s.result.References {
			appendSpan("reference", fmt.Sprintf("引用 %d", i+1), "", i, ref.UTF16.Start, ref.UTF16.End)
		}
	case code.ServiceInfo:
		if info := s.result.Info; info != nil {
			detail := strings.TrimSpace(info.Signature)
			if info.Documentation != "" {
				if detail != "" {
					detail += " · "
				}
				detail += info.Documentation
			}
			appendSpan("info", info.Symbol.Name, detail, 0, info.Symbol.Span.UTF16.Start, info.Symbol.Span.UTF16.End)
		}
	}
	return s.rows
}

func (a *App) sourceLanguageActionCurrent(action sourceLanguageAction) bool {
	s := &a.sourceLanguage
	if !a.sourceLanguagePresentationCurrent() || action.presentation != s.presentation || action.request != s.request {
		return false
	}
	if action.renameDialog != nil {
		return s.rename == action.renameDialog && action.renameEpoch == s.rename.epoch
	}
	if action.kind != "navigate" && action.kind != "complete" {
		return true
	}
	// This predicate also runs inside checked Editor callbacks; inspect only
	// the already-published row set and never build or mutate a row cache here.
	if !s.rowsReady || s.rowRequest != s.request || s.rowPresentation != s.presentation {
		return false
	}
	for _, row := range s.rows {
		if row.id == action.rowID {
			return true
		}
	}
	return false
}

func (a *App) queueSourceLanguageAction(kind, rowID string) {
	if strings.HasPrefix(kind, "rename-") {
		a.queueSourceRenameAction(kind)
		return
	}
	if err := a.sourceLanguageInputError(); err != nil {
		if a.sourceLanguagePresentationCurrent() {
			a.sourceLanguage.message = err.Error()
		}
		return
	}
	if !a.sourceLanguagePresentationCurrent() {
		return
	}
	s := &a.sourceLanguage
	if s.intent.completion != nil && (a.commandPalette.composing || a.openDocuments.composing || a.findCompositionActive() || a.GoToLineComposing || a.Current() != nil && a.Current().Editor != nil && a.sourceCompositionActive(a.Current())) {
		s.message = "请先完成或取消当前输入法组合，再操作源码补全"
		return
	}
	s.action = &sourceLanguageAction{kind: kind, rowID: rowID, request: s.request, presentation: s.presentation}
}

func (a *App) processSourceLanguageActions() {
	s := &a.sourceLanguage
	action := s.action
	if action == nil {
		return
	}
	if action.renameDialog != nil {
		a.processSourceRenameAction(action)
		return
	}
	if !a.sourceLanguageActionCurrent(*action) {
		if s.action == action {
			s.action = nil
		}
		return
	}
	if err := a.sourceLanguageInputError(); err != nil {
		if a.sourceLanguageActionCurrent(*action) {
			s.action = nil
			s.message = err.Error()
		}
		return
	}
	if !a.sourceLanguageActionCurrent(*action) {
		return
	}
	if s.intent.completion != nil && (a.commandPalette.composing || a.openDocuments.composing || a.findCompositionActive() || a.GoToLineComposing || a.Current() != nil && a.Current().Editor != nil && a.sourceCompositionActive(a.Current())) {
		s.action = nil
		s.message = "请先完成或取消当前输入法组合，再操作源码补全"
		return
	}
	s.action = nil
	switch action.kind {
	case "close":
		if !a.retireSourceLanguage(nil) {
			return
		}
		a.closeSourceLanguagePresentation()
	case "cancel":
		a.runSourceLanguageCancel()
	case "reanalyze":
		intent := s.intent
		if intent.completion != nil {
			// A read-style reanalysis must not silently rebind an editing
			// descriptor to a new caret or selection after ABA.
			if err := a.sourceLanguageOwnerValid(s.owner, true); err != nil {
				a.failSourceLanguage(sourceLanguageStale, "completion-reanalyze-stale", err)
				return
			}
		}
		if intent.operation == "" {
			s.message = "没有可重新分析的语言请求"
			return
		}
		err := a.requestSourceLanguage(intent)
		if err != nil && a.sourceLanguagePresentationCurrent() && action.request == s.request {
			s.message = err.Error()
		}
	case "navigate":
		a.navigateSourceLanguageRow(*action)
	case "complete":
		a.confirmSourceCompletion(*action)
	case "completion-cancel":
		if !a.cancelSourceLanguage() {
			return
		}
		a.closeSourceLanguagePresentation()
	}
}

func (a *App) navigateSourceLanguageRow(action sourceLanguageAction) {
	s := &a.sourceLanguage
	if !a.sourceLanguageActionCurrent(action) {
		return
	}
	owner := s.owner
	if !a.sourceLanguageDisplayMatches(owner, action.request, action.presentation) {
		return
	}
	if err := owner.input.InputActionError(); err != nil {
		if a.sourceLanguageDisplayMatches(owner, action.request, action.presentation) {
			s.message = err.Error()
		}
		return
	}
	if a.retireStaleSourceLanguageDisplay(owner, action.request, action.presentation) ||
		!a.sourceLanguageDisplayMatches(owner, action.request, action.presentation) || !a.sourceLanguageActionCurrent(action) {
		return
	}
	var row sourceLanguageRow
	found := false
	for _, item := range s.rows {
		if item.id == action.rowID {
			row, found = item, true
			break
		}
	}
	if !found {
		return
	}
	current := func() bool {
		return a.sourceLanguageDisplayMatches(owner, action.request, action.presentation) &&
			a.sourceLanguageActionCurrent(action) && a.sourceLanguageOwnerValid(owner, false) == nil
	}
	err := owner.input.RevealSourceTargetChecked(owner.target, row.start, row.end, current)
	if !current() {
		// A source change during a callback needs disclosure only while the
		// original display still owns publication. A replacement request, cancel,
		// or presentation must retain its exact callback-exit state.
		a.retireStaleSourceLanguageDisplay(owner, action.request, action.presentation)
		return
	}
	if err != nil {
		s.message = "无法转到语言结果：" + sourceLanguageBoundText(err.Error())
		return
	}
	s.message = fmt.Sprintf("已定位到第 %d 行第 %d UTF-16 列", row.position.line, row.position.column)
}

func (a *App) runSourceLanguageCommand(operation string) {
	if err := a.sourceLanguageInputError(); err != nil {
		a.report(err)
		return
	}
	op := code.ServiceOperation(operation)
	switch operation {
	case "analyze", "diagnostics":
		op = code.ServiceDiagnostics
	case "symbols":
		op = code.ServiceSymbols
	case "definition":
		op = code.ServiceDefinition
	case "references":
		op = code.ServiceReferences
	case "info":
		op = code.ServiceInfo
	default:
		err := fmt.Errorf("未知语言服务命令 %q", operation)
		a.report(err)
		return
	}
	if err := a.openSourceLanguagePresentation(); err != nil {
		if !errors.Is(err, errSourceLanguageSuperseded) {
			a.report(err)
		}
		return
	}
	t := a.Current()
	intent := sourceLanguageIntent{operation: op, limits: code.DefaultServiceLimits()}
	if op == code.ServiceDefinition || op == code.ServiceReferences || op == code.ServiceInfo {
		selection := t.Doc.Buffer.SelectionSet()
		if len(selection.Ranges) == 0 || selection.Primary < 0 || selection.Primary >= len(selection.Ranges) {
			a.sourceLanguage.message = "当前选择没有有效源码位置"
			return
		}
		primary := selection.Ranges[selection.Primary]
		intent.utf16Offset = primary.Start()
		intent.selectedEndUTF16 = primary.End()
		intent.hasUTF16Offset = true
		intent.hasSelectedRange = primary.Start() != primary.End()
	}
	if err := a.requestSourceLanguage(intent); err != nil {
		if errors.Is(err, errSourceLanguageSuperseded) {
			return
		}
		// Route, byte-budget and captured-source failures are visible language
		// outcomes; the B1 failure boundary may retire an older presentation.
		if !a.sourceLanguagePresentationCurrent() {
			if showErr := a.openSourceLanguagePresentation(); showErr != nil {
				if !errors.Is(showErr, errSourceLanguageSuperseded) {
					a.report(showErr)
				}
				return
			}
		}
		if a.sourceLanguagePresentationCurrent() {
			a.sourceLanguage.message = err.Error()
		}
		return
	}
	a.sourceLanguage.message = ""
}

func (a *App) runSourceLanguageCancel() {
	if err := a.sourceLanguageInputError(); err != nil {
		a.report(err)
		return
	}
	if !languageCancelCommandAvailable(a) {
		a.report(errors.New("当前没有可取消的语言分析"))
		return
	}
	if !a.cancelSourceLanguage() {
		return
	}
	if err := a.openSourceLanguagePresentation(); err != nil && !errors.Is(err, errSourceLanguageSuperseded) {
		a.report(err)
	}
}

func (a *App) sourceLanguagePanel(c *ui.Context) {
	s := &a.sourceLanguage
	if !s.visible || !a.sourceLanguagePresentationCurrent() || a.ShowPlugins {
		return
	}
	if s.rename != nil {
		a.sourceRenamePanel(c)
		return
	}
	rows := a.sourceLanguageRows()
	inputError := a.sourceLanguageInputError()
	if s.intent.completion != nil {
		if inputError == nil && (a.commandPalette.composing || a.openDocuments.composing || a.findCompositionActive() || a.GoToLineComposing || a.Current() != nil && a.Current().Editor != nil && a.sourceCompositionActive(a.Current())) {
			inputError = errors.New("请先完成或取消当前输入法组合")
		}
		// Completion has its own compact controls in the same owned panel;
		// generic navigation/reanalysis never consumes an editing candidate.
		ui.Column(c.Key(fmt.Sprintf("language-completion-%d", s.presentation))).MaxHeight(206).Shrink(0).
			Padding(6, 10).Gap(4).Background(c.Theme().Surface).Border(1, c.Theme().Border).Children(func() {
			ui.Text(c, "语言服务结果").Bold().FontSize(12)
			status := a.sourceLanguageReadinessMessage()
			ui.Text(c, status).Label("语言服务状态").FontSize(11).SingleLine().Tooltip(status)
			if s.message != "" {
				ui.Text(c, sourceLanguageBoundText(s.message)).FontSize(11).MaxLines(2)
			}
			a.sourceCompletionPanel(c, rows, inputError)
		})
		return
	}
	profile := s.presentationTab.sourceLanguageProfile
	capability := "严格 JSON · RFC 8259；不是 JSONC"
	if profile == code.StructureGo {
		capability = "Go 词法 / 语法与本文件保守绑定；不做类型检查"
	}
	ui.Column(c.Key(fmt.Sprintf("language-results-%d", s.presentation))).MaxHeight(206).Shrink(0).
		Padding(6, 10).Gap(4).Background(c.Theme().Surface).Border(1, c.Theme().Border).Children(func() {
		ui.Row(c.Key("language-results-heading")).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "语言服务结果").Bold().Grow(1).MinWidth(0)
			if workbenchButton(c, "重新分析语言结果").Label("重新分析语言结果").KeepFocus().Height(25).Disabled(inputError != nil || s.intent.operation == "").Clicked() {
				a.queueSourceLanguageAction("reanalyze", "")
			}
			if workbenchButton(c, "取消语言分析").Label("取消语言分析").KeepFocus().Height(25).Disabled(inputError != nil || (s.task == nil && s.pending == nil && s.owner == nil)).Clicked() {
				a.queueSourceLanguageAction("cancel", "")
			}
			if workbenchButton(c, "关闭语言结果").Label("关闭语言结果").KeepFocus().Height(25).Disabled(inputError != nil).Clicked() {
				a.queueSourceLanguageAction("close", "")
			}
		})
		ui.Text(c, "能力："+capability).FontSize(11).SingleLine().Tooltip(capability)
		status := a.sourceLanguageReadinessMessage()
		ui.Text(c, "语言服务状态").Label("语言服务状态").FontSize(11).SingleLine().Tooltip(status)
		ui.Text(c, status).FontSize(11).SingleLine().Tooltip(status)
		if s.intent.operation != "" {
			ui.Text(c, fmt.Sprintf("分析版本 %d · 操作 %s", s.analysisVersion, sourceLanguageOperation(s.intent.operation))).FontSize(10).TextColor(c.Theme().TextMuted).SingleLine()
		}
		if s.message != "" {
			ui.Text(c, sourceLanguageBoundText(s.message)).FontSize(11).TextColor(c.Theme().Warning).MaxLines(2).Tooltip(s.message)
		}
		listTitle := sourceLanguageOperation(s.intent.operation) + "结果"
		if s.intent.operation == code.ServiceDiagnostics {
			listTitle = "语法诊断"
		} else if s.result.Status == code.ServiceInvalid && len(s.result.Diagnostics) > 0 {
			listTitle = "语法诊断"
		} else if s.intent.operation == "" {
			listTitle = "分析结果"
		}
		ui.Text(c, listTitle).FontSize(11).Bold()
		if len(rows) == 0 {
			if s.status == string(code.ServiceComplete) && s.intent.operation == code.ServiceDiagnostics {
				ui.Text(c, "语法检查已完成；无诊断项").FontSize(11).TextColor(c.Theme().TextMuted)
			} else {
				ui.Text(c, "当前没有可显示的语言结果").FontSize(11).TextColor(c.Theme().TextMuted)
			}
			return
		}
		selected := s.selected
		if selected < 0 || selected >= len(rows) {
			selected = 0
		}
		s.list.Selected = &selected
		s.list.Key = func(i int) any {
			if i < 0 || i >= len(rows) {
				return i
			}
			return rows[i].id
		}
		s.list.Label = func(i int) string {
			if !a.sourceLanguagePresentationCurrent() || i < 0 || i >= len(rows) {
				return ""
			}
			return rows[i].label
		}
		list := ui.List(c.Key("language-result-list"), &s.list, len(rows), func(i int) {
			if !a.sourceLanguagePresentationCurrent() || i < 0 || i >= len(rows) {
				return
			}
			row := rows[i]
			ui.Text(c, row.label).Label("语言结果：" + row.label).FontSize(11).SingleLine().FillWidth().Tooltip(row.detail)
		}).Grow(1).MinHeight(24).KeepFocus()
		if selected >= 0 && selected < len(rows) {
			s.selected = selected
			if list.Submitted() {
				a.queueSourceLanguageAction("navigate", rows[selected].id)
			}
		}
		if workbenchButton(c.Key("language-result-go"), "转到语言结果").Label("转到语言结果").KeepFocus().Height(26).Disabled(inputError != nil || len(rows) == 0).Clicked() {
			if s.selected >= 0 && s.selected < len(rows) {
				a.queueSourceLanguageAction("navigate", rows[s.selected].id)
			} else if len(rows) > 0 {
				s.selected = 0
				a.queueSourceLanguageAction("navigate", rows[0].id)
			}
		}
		s.listLifetime.built = true
	})
}
