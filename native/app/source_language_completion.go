package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/code"
	"levnote.local/native/ui/editor"
)

const sourceCompletionJSONInstruction = "请选择完整JSON字符串再替换为本文件已有键文字"

// This bounded descriptor has no source snapshot or candidate payload. The
// owning B1 seal retains the original opaque target. Generation includes every
// secondary/direction/primary change, even if the same selector later returns.
type sourceCompletionIntent struct {
	selection           text.Range
	selectionGeneration uint64
	focusStamp          sourceLanguageFocusStamp
	start, end          int
	oldText             string
	proven              bool
	proofFailure        string
}

func languageCompletionCommandAvailable(a *App) bool {
	if a == nil {
		return false
	}
	t, _, err := a.sourceLanguageAdmission()
	// Keep the command discoverable for multiple selections, which receive an
	// explicit refusal. Input admission remains the first execution boundary.
	return err == nil && !t.Doc.ReadOnly && !t.Editor.ReadOnly
}

func sourceCompletionSelectionValid(t *Tab, intent *sourceCompletionIntent) error {
	if t == nil || t.Doc == nil || t.Editor == nil || t.Doc.Buffer == nil || t.Editor.Buffer != t.Doc.Buffer || intent == nil {
		return errors.New("补全原始目标已失效；请重新请求")
	}
	if t.Doc.ReadOnly || t.Editor.ReadOnly {
		return editor.ErrSourceReadOnly
	}
	b := t.Doc.Buffer
	set := b.SelectionSet()
	if len(set.Ranges) != 1 || set.Primary != 0 || set.Ranges[0] != intent.selection || b.SelectionGeneration() != intent.selectionGeneration {
		return errors.New("补全原始选择已变化；请重新请求")
	}
	return nil
}

func goCompletionPart(r rune) bool  { return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) }
func goCompletionStart(r rune) bool { return r == '_' || unicode.IsLetter(r) }

// Read at most one scalar (four UTF8 bytes); indexed Buffer lookup avoids a
// whole-source prefix copy. The field loop admits at most16KiB plus one scalar
// to prove the boundary. A surrogate interior is never silently rounded.
func completionRuneBefore(b *text.Buffer, at int) (rune, int, error) {
	if at == 0 {
		return 0, 0, nil
	}
	start := at - 1
	value, err := b.SliceUTF16(start, at)
	if errors.Is(err, text.ErrSurrogate) {
		start--
		value, err = b.SliceUTF16(start, at)
	}
	if err != nil {
		return 0, at, err
	}
	r, size := utf8.DecodeRuneInString(value)
	if size != len(value) {
		return 0, at, text.ErrRange
	}
	return r, start, nil
}

func completionRuneAfter(b *text.Buffer, at int) (rune, int, error) {
	if at == b.LenUTF16() {
		return 0, at, nil
	}
	end := at + 1
	value, err := b.SliceUTF16(at, end)
	if errors.Is(err, text.ErrSurrogate) {
		end++
		value, err = b.SliceUTF16(at, end)
	}
	if err != nil {
		return 0, at, err
	}
	r, size := utf8.DecodeRuneInString(value)
	if size != len(value) {
		return 0, at, text.ErrRange
	}
	return r, end, nil
}

func completionBoundedText(b *text.Buffer, start, end int) (string, error) {
	if start < 0 || end < start || end > b.LenUTF16() {
		return "", text.ErrRange
	}
	var out strings.Builder
	for at := start; at < end; {
		r, next, err := completionRuneAfter(b, at)
		if err != nil || next > end {
			return "", text.ErrSurrogate
		}
		if utf8.RuneLen(r) > sourceLanguageFieldBytes-out.Len() {
			return "", errors.New("补全替换字段超过 16 KiB 上限")
		}
		out.WriteRune(r)
		at = next
	}
	return out.String(), nil
}

func prepareSourceCompletion(t *Tab, profile code.StructureProfile) (sourceLanguageIntent, error) {
	b := t.Doc.Buffer
	set := b.SelectionSet()
	if len(set.Ranges) != 1 || set.Primary != 0 {
		return sourceLanguageIntent{}, errors.New("源码补全仅支持一个选择；请先保留一个选择后重试")
	}
	range_ := set.Ranges[0]
	edit := &sourceCompletionIntent{selection: range_, selectionGeneration: b.SelectionGeneration(), start: range_.Start(), end: range_.End()}
	intent := sourceLanguageIntent{operation: code.ServiceCompletion, limits: code.DefaultServiceLimits(), completion: edit}
	if profile == code.StructureJSON {
		// Actual token proof is deferred to the single admitted snapshot. Empty
		// and incomplete selections still reach the real whole-syntax service.
		value, err := completionBoundedText(b, edit.start, edit.end)
		if err != nil {
			return sourceLanguageIntent{}, err
		}
		edit.oldText = strings.Clone(value)
		return intent, nil
	}
	if edit.start == edit.end {
		at, bytes := edit.end, 0
		var reversed []rune
		for at > 0 {
			r, previous, err := completionRuneBefore(b, at)
			if err != nil {
				return sourceLanguageIntent{}, err
			}
			if !goCompletionPart(r) {
				break
			}
			if utf8.RuneLen(r) > sourceLanguageFieldBytes-bytes {
				return sourceLanguageIntent{}, errors.New("补全标识符前缀超过 16 KiB 上限；不能只替换其后缀")
			}
			bytes += utf8.RuneLen(r)
			reversed = append(reversed, r)
			at = previous
		}
		for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
			reversed[i], reversed[j] = reversed[j], reversed[i]
		}
		edit.start, edit.oldText = at, string(reversed)
	} else {
		value, err := completionBoundedText(b, edit.start, edit.end)
		if err != nil {
			return sourceLanguageIntent{}, err
		}
		first := true
		for _, r := range value {
			if !goCompletionPart(r) || first && !goCompletionStart(r) {
				return sourceLanguageIntent{}, errors.New("请选择一个完整 Go 标识符，或把光标放到其前缀末尾")
			}
			first = false
		}
		left, _, err := completionRuneBefore(b, edit.start)
		if err != nil {
			return sourceLanguageIntent{}, err
		}
		right, _, err := completionRuneAfter(b, edit.end)
		if err != nil {
			return sourceLanguageIntent{}, err
		}
		if goCompletionPart(left) || goCompletionPart(right) {
			return sourceLanguageIntent{}, errors.New("选择必须覆盖完整 Go 标识符；不会自动扩展或截断")
		}
		edit.oldText = strings.Clone(value)
	}
	intent.prefix = strings.Clone(edit.oldText)
	return intent, nil
}

func (a *App) runSourceCompletionCommand() {
	if err := a.sourceLanguageInputError(); err != nil {
		a.report(err)
		return
	}
	if err := a.requestSourceCompletion(); err != nil {
		if !errors.Is(err, errSourceLanguageSuperseded) {
			a.report(err)
		}
		return
	}
	if err := a.openSourceLanguagePresentation(); err != nil && !errors.Is(err, errSourceLanguageSuperseded) {
		a.report(err)
	}
}

func (a *App) requestSourceCompletion() error {
	if err := a.sourceLanguageInputError(); err != nil {
		return err
	}
	if a.commandPalette.composing || a.openDocuments.composing || a.findCompositionActive() || a.GoToLineComposing || a.Current() != nil && a.Current().Editor != nil && a.sourceCompositionActive(a.Current()) {
		return errors.New("请先完成或取消当前输入法组合，再请求源码补全")
	}
	t, profile, err := a.sourceLanguageAdmission()
	if err != nil {
		return a.failSourceLanguage(sourceLanguageHostError, "completion-admission", err)
	}
	if t.Doc.ReadOnly || t.Editor.ReadOnly {
		return a.failSourceLanguage(sourceLanguageHostError, "completion-readonly", editor.ErrSourceReadOnly)
	}
	intent, err := prepareSourceCompletion(t, profile)
	if err != nil {
		return a.failSourceLanguage(sourceLanguageHostError, "completion-selection", err)
	}
	return a.requestSourceLanguage(intent)
}

// This is a UTF8/UTF16 coordinate conversion, not a JSON lexer. It consumes the
// one admitted immutable snapshot and validates both original selector ends.
func completionByteBounds(source string, start, end int) (int, int, error) {
	if start < 0 || end < start {
		return 0, 0, text.ErrRange
	}
	left, right, units := -1, -1, 0
	for at, r := range source {
		if units == start {
			left = at
		}
		if units == end {
			right = at
			break
		}
		next := units + 1
		if r > 0xffff {
			next++
		}
		if start > units && start < next || end > units && end < next {
			return 0, 0, text.ErrSurrogate
		}
		units = next
	}
	if units == start && left < 0 {
		left = len(source)
	}
	if units == end && right < 0 {
		right = len(source)
	}
	if left < 0 || right < left {
		return 0, 0, text.ErrRange
	}
	return left, right, nil
}

func proveSourceCompletion(source string, profile code.StructureProfile, edit *sourceCompletionIntent) (bool, string) {
	left, right, err := completionByteBounds(source, edit.start, edit.end)
	if err != nil || right-left > sourceLanguageFieldBytes || source[left:right] != edit.oldText {
		return false, "补全原始范围与已准入快照不一致"
	}
	if profile == code.StructureGo {
		return true, ""
	}
	dec := json.NewDecoder(strings.NewReader(source))
	dec.UseNumber()
	previous := 0
	for {
		token, err := dec.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return false, ""
			}
			return false, "JSON token 边界证明失败：" + sourceLanguageBoundText(err.Error())
		}
		end64 := dec.InputOffset()
		if end64 < int64(previous) || end64 > int64(len(source)) {
			return false, "JSON token 偏移无效"
		}
		end, start := int(end64), previous
		// Token already validated the grammar and elided these separators. No
		// quote search, escape scanner or decoded-string re-encoding occurs.
		for start < end && strings.ContainsRune(" \t\r\n,:", rune(source[start])) {
			start++
		}
		_, isString := token.(string)
		if isString && start < end && source[start] == '"' && source[end-1] == '"' && start == left && end == right {
			return true, ""
		}
		if end >= right {
			return false, ""
		}
		previous = end
	}
}

func validateSourceCompletionResult(owner *sourceLanguageOwner, edit *sourceCompletionIntent, result code.ServiceResult) error {
	if owner == nil || edit == nil || result.Status != code.ServiceComplete {
		return errors.New("补全结果未完成")
	}
	if edit.proofFailure != "" {
		return errors.New(edit.proofFailure)
	}
	if !edit.proven {
		return errors.New(sourceCompletionJSONInstruction)
	}
	limits := code.DefaultServiceLimits()
	if len(result.Completion) > limits.Completion {
		return errors.New("补全载荷超过候选上限")
	}
	if len(result.Diagnostics) != 0 || len(result.Symbols) != 0 || result.Definition != nil || len(result.References) != 0 || result.Info != nil || len(result.Edits) != 0 || result.NoOp {
		return errors.New("补全载荷混入其它操作的结果")
	}
	remaining := limits.TextBytes
	for _, item := range result.Completion {
		if len(item.Text) == 0 || len(item.Text) > remaining || len(item.Text) > sourceLanguageFieldBytes || !utf8.ValidString(item.Text) ||
			owner.profile == code.StructureGo && item.Kind != "keyword" && item.Kind != "file-word" ||
			owner.profile == code.StructureJSON && item.Kind != "json-key" {
			return errors.New("补全候选包含无效文本或种类")
		}
		remaining -= len(item.Text)
	}
	return nil
}

func completionKind(kind string) string {
	switch kind {
	case "keyword":
		return "Go 关键字"
	case "file-word":
		return "本文件词"
	case "json-key":
		return "JSON 原始键"
	default:
		return "未知候选"
	}
}

const sourceLanguageCompletionRows = 512

func (a *App) sourceCompletionFocusCurrent(edit *sourceCompletionIntent) bool {
	if edit == nil {
		return false
	}
	stamp := edit.focusStamp
	if a.sourceLanguageFocusStamp() != stamp {
		return false
	}
	// Find focus requests are one-shot, but the request is still live while the
	// field is being bound. Do not let this frame's completion list win that race.
	if a.findFocus != findFocusNone && a.findFocusGeneration == a.findGeneration {
		return false
	}
	return true
}

func (a *App) sourceCompletionRows() []sourceLanguageRow {
	s := &a.sourceLanguage
	if s.intent.completion == nil || s.result.Status != code.ServiceComplete || s.owner == nil {
		return nil
	}
	if a.sourceLanguageInputError() != nil {
		return nil
	}
	if err := a.sourceLanguageOwnerValid(s.owner, true); err != nil {
		a.failSourceLanguage(sourceLanguageStale, "completion-selection-changed", err)
		return nil
	}
	if s.rowsReady && s.rowRequest == s.request && s.rowPresentation == s.presentation {
		return s.rows
	}
	s.rows = nil
	s.rowsReady, s.rowRequest, s.rowPresentation = true, s.request, s.presentation
	for i, item := range s.result.Completion {
		if i >= sourceLanguageCompletionRows {
			break
		}
		id := sourceLanguageRowID("completion", i, s.intent.completion.start, s.intent.completion.end)
		label := sourceLanguageBoundText(item.Text + " · " + completionKind(item.Kind))
		s.rows = append(s.rows, sourceLanguageRow{id: id, label: label, detail: sourceLanguageBoundText(completionKind(item.Kind)), kind: "completion", start: s.intent.completion.start, end: s.intent.completion.end})
	}
	return s.rows
}

// Called only after the queued action has been consumed at the safe frame
// boundary. Release all old presentation text BEFORE B3's terminal OnChange;
// successful return performs no Host write or old-target validation at all.
func (a *App) confirmSourceCompletion(action sourceLanguageAction) {
	s := &a.sourceLanguage
	if !a.sourceLanguageActionCurrent(action) || s.intent.completion == nil || s.owner == nil {
		return
	}
	if err := a.sourceLanguageInputError(); err != nil {
		s.message = err.Error()
		return
	}
	if a.commandPalette.composing || a.openDocuments.composing || a.findCompositionActive() || a.GoToLineComposing || a.Current() != nil && a.Current().Editor != nil && a.sourceCompositionActive(a.Current()) {
		s.message = "请先完成或取消当前输入法组合，再确认补全"
		return
	}
	owner, edit := s.owner, s.intent.completion
	if err := sourceCompletionSelectionValid(owner.tab, edit); err != nil {
		a.failSourceLanguage(sourceLanguageHostError, "completion-selection-changed", err)
		return
	}
	if err := a.sourceLanguageOwnerValid(owner, true); err != nil {
		a.failSourceLanguage(sourceLanguageStale, "completion-confirm-stale", err)
		return
	}
	index := -1
	for i, row := range s.rows {
		if row.kind == "completion" && row.id == action.rowID {
			index = i
			break
		}
	}
	if index < 0 || index >= len(s.result.Completion) || s.result.Status != code.ServiceComplete {
		return
	}
	item := s.result.Completion[index]
	if err := validateSourceCompletionResult(owner, edit, s.result); err != nil {
		s.message = err.Error()
		return
	}
	current, err := owner.buffer.SliceUTF16(edit.start, edit.end)
	if err != nil || current != edit.oldText {
		a.failSourceLanguage(sourceLanguageStale, "completion-old-text-changed", errors.New("补全原文已变化；请重新请求"))
		return
	}
	patch := text.PatchEdit{Start: edit.start, End: edit.end, OldText: strings.Clone(edit.oldText), NewText: strings.Clone(item.Text)}
	a.retireSourceLanguage(nil)
	a.closeSourceLanguagePresentation()
	request, presentation, session := s.request, s.presentation, s.session
	if patch.OldText == patch.NewText {
		return
	} // full admission above is mandatory
	_, err = owner.input.ApplySourceEdits(owner.target, []text.PatchEdit{patch}, editor.SingleInsertionCaret)
	if err != nil && s.request == request && s.presentation == presentation && s.session == session && a.Current() == owner.tab && owner.tab.Doc == owner.doc && owner.tab.Editor == owner.input {
		a.Message = "无法确认补全：" + sourceLanguageBoundText(err.Error())
	}
}

func (a *App) sourceCompletionPanel(c *ui.Context, rows []sourceLanguageRow, inputError error) {
	s := &a.sourceLanguage
	request, presentation := s.request, s.presentation
	ui.Text(c, "源码补全候选").Label("源码补全候选").Bold().FontSize(11)
	ui.Text(c, "仅提供 Go 关键字 / 本文件词，或 JSON 原始键；不做类型检查").FontSize(10).SingleLine()
	if s.result.Status == code.ServiceComplete && len(rows) == 0 {
		ui.Text(c, "当前请求没有可用的本文件词汇候选").FontSize(11).SingleLine()
	}
	if len(s.result.Completion) > len(rows) && len(rows) == sourceLanguageCompletionRows {
		ui.Text(c, fmt.Sprintf("显示前 %d / %d 个候选", len(rows), len(s.result.Completion))).FontSize(10)
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
		if s.request != request || s.presentation != presentation || i < 0 || i >= len(rows) {
			return ""
		}
		return rows[i].label
	}
	// Virtual rows can change the List's measured height after its first layout.
	// Put the action row before it so that measuring rows cannot move a button
	// between pointer down and up. Apply clicks after the List reads selection.
	var confirmClicked, cancelClicked bool
	ui.Row(c.Key("language-completion-actions")).Gap(8).Children(func() {
		confirmClicked = workbenchButton(c, "确认补全").Label("确认补全").KeepFocus().Height(26).Disabled(inputError != nil || len(rows) == 0).Clicked()
		cancelClicked = workbenchButton(c, "取消补全").Label("取消补全").KeepFocus().Height(26).Disabled(inputError != nil).Clicked()
	})
	list := ui.List(c.Key("language-completion-list"), &s.list, len(rows), func(i int) {
		if s.request != request || s.presentation != presentation || i < 0 || i >= len(rows) {
			return
		}
		ui.Text(c, rows[i].label).Label("补全候选：" + rows[i].label).SingleLine().FillWidth().FontSize(11)
	}).Grow(1).MinHeight(24).KeepFocus()
	list.OnShortcut(0, ui.KeyEscape, func() {
		if s.request == request && s.presentation == presentation {
			a.queueSourceLanguageAction("completion-cancel", "")
		}
	})
	if s.completionFocus {
		s.completionFocus = false
		if inputError == nil && s.request == request && s.presentation == presentation && s.owner != nil && a.sourceLanguageOwnerValid(s.owner, true) == nil && a.sourceCompletionFocusCurrent(s.intent.completion) {
			// A first-ever list has no usable Handle at worker publication.
			// Focus only this freshly built public Handle, after pure admission.
			s.list.Handle.Focus()
		}
	}
	if selected >= 0 && selected < len(rows) {
		s.selected = selected
		if list.Submitted() {
			a.queueSourceLanguageAction("complete", rows[selected].id)
		}
	}
	if confirmClicked && s.request == request && s.presentation == presentation && selected >= 0 && selected < len(rows) {
		a.queueSourceLanguageAction("complete", rows[selected].id)
	}
	if cancelClicked && s.request == request && s.presentation == presentation {
		a.queueSourceLanguageAction("completion-cancel", "")
	}
	s.listLifetime.built = true
}
