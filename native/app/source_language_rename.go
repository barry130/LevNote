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
	"levnote.local/native/ui/editor"
)

const (
	sourceRenameNameBytes = 512
	sourceRenameNameUnits = 128
)

// original is the OPEN target, including every directed source selection. It
// never follows a caret or a newer snapshot. Only explicit OPEN replaces it.
type sourceRenameDialog struct {
	original      *sourceLanguageOwner
	utf16Offset   int
	byteOffset    int
	selectedRange bool
	field         *editor.Editor
	buffer        *text.Buffer
	epoch         uint64
	request       *sourceRenameRequest
	patch         []text.PatchEdit
	previewRows   []sourceLanguageRow
	focus         bool
}

// Each explicit preview gets its own small name snapshot and operation seal.
// No source string is retained by either dialog or request.
type sourceRenameRequest struct {
	dialog *sourceRenameDialog
	epoch  uint64
	field  *editor.Editor
	buffer *text.Buffer
	target editor.SourceTarget
	name   string
}

func languageRenameCommandAvailable(a *App) bool {
	if a == nil {
		return false
	}
	t, profile, err := a.sourceLanguageAdmission()
	return err == nil && profile == code.StructureGo && !t.Doc.ReadOnly && !t.Editor.ReadOnly
}

func (a *App) sourceRenameInputError() error {
	if err := a.sourceLanguageInputError(); err != nil {
		return err
	}
	if a.commandPalette.composing || a.openDocuments.composing || a.findCompositionActive() || a.GoToLineComposing || a.sourceCompositionActive(a.Current()) {
		return errors.New("请先完成或取消当前输入法组合")
	}
	return nil
}

func (a *App) runSourceRenameCommand() {
	if err := a.openSourceRename(); err != nil {
		a.report(err)
	}
}

func sourceRenameSeed(source string, at int) (string, string) {
	if at < 0 || at >= len(source) {
		return "", "请输入本文件符号的新名称"
	}
	part := func(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' }
	left, right := at, at
	for left > 0 {
		r, n := utf8.DecodeLastRuneInString(source[:left])
		if !part(r) {
			break
		}
		left -= n
		if at-left > sourceRenameNameBytes {
			return "", "原名称超过名称字段预算；请输入完整新名称"
		}
	}
	for right < len(source) {
		r, n := utf8.DecodeRuneInString(source[right:])
		if !part(r) {
			break
		}
		right += n
		if right-left > sourceRenameNameBytes {
			return "", "原名称超过名称字段预算；请输入完整新名称"
		}
	}
	seed := source[left:right]
	if len(seed) > sourceRenameNameBytes || text.UTF16Len(seed) > sourceRenameNameUnits {
		return "", "原名称超过 128 UTF-16 单位；请输入完整新名称"
	}
	return strings.Clone(seed), ""
}

// The busy OPEN path has no permission to materialize another whole source.
// Buffer.WriteTo visits immutable pieces through StringWriter; this cursor keeps
// only scalar coordinates and never retains a chunk or a source prefix.
type sourceRenameOffsetCursor struct {
	want, units, bytes, at, leftAt int
	selected, leftPart, found      bool
}

func (p *sourceRenameOffsetCursor) Write(b []byte) (int, error) { return p.WriteString(string(b)) }
func (p *sourceRenameOffsetCursor) WriteString(value string) (int, error) {
	if p.found {
		return len(value), nil
	}
	for i := 0; i < len(value); {
		r, n := utf8.DecodeRuneInString(value[i:])
		if r == utf8.RuneError && n == 1 {
			return i, errors.New("源码不是有效 UTF-8")
		}
		if p.units == p.want {
			p.at = p.bytes
			if !p.selected && !isSourceIdentifierPart(r) && p.leftPart {
				p.at = p.leftAt
			}
			p.found = true
			return len(value), nil
		}
		u := 1
		if r > 0xffff {
			u++
		}
		if p.units+u > p.want {
			return i, errors.New("源码位置位于 UTF-16 代理对中间")
		}
		p.leftAt, p.leftPart = p.bytes, isSourceIdentifierPart(r)
		p.bytes += n
		p.units += u
		i += n
	}
	return len(value), nil
}

func sourceRenameOriginalByte(buffer *text.Buffer, units int, selected bool) (int, error) {
	if units < 0 {
		return 0, errors.New("源码位置无效")
	}
	p := sourceRenameOffsetCursor{want: units, selected: selected}
	if _, err := buffer.WriteTo(&p); err != nil {
		return 0, err
	}
	if !p.found {
		if p.units != units {
			return 0, errors.New("源码位置超出范围")
		}
		p.at = p.bytes
		if !selected && p.leftPart {
			p.at = p.leftAt
		}
	}
	return p.at, nil
}

func (a *App) openSourceRename() error {
	// Pure gates precede retirement, focus or snapshots. In particular this
	// must leave a first refused markless native input owned by its client.
	if err := a.sourceRenameInputError(); err != nil {
		return err
	}
	t, profile, err := a.sourceLanguageAdmission()
	if err != nil {
		return err
	}
	if profile != code.StructureGo {
		return errors.New("本文件符号重命名仅支持 Go 源码")
	}
	if t.Doc.ReadOnly || t.Editor.ReadOnly {
		return errors.New("只读源码不能重命名")
	}
	if a.sourceLanguageDispatcher() == nil {
		return errors.New("当前宿主没有可用的 UI 排队调度器")
	}
	s := &a.sourceLanguage
	var original sourceLanguageOwner
	var seed, message string
	set := t.Doc.Buffer.SelectionSet()
	if len(set.Ranges) == 0 || set.Primary < 0 || set.Primary >= len(set.Ranges) {
		return errors.New("当前源码没有有效主选择")
	}
	primary := set.Ranges[set.Primary]
	byteOffset := 0
	if s.task != nil {
		if err := a.sourceLanguageOwnerValid(s.owner, true); err != nil {
			return fmt.Errorf("分析繁忙且原目标已变化；请结束分析后重新打开重命名：%w", err)
		}
		byteOffset, err = sourceRenameOriginalByte(t.Doc.Buffer, primary.Start(), primary.Start() != primary.End())
		if err != nil {
			return err
		}
		original = *s.owner
		message = "分析繁忙；请输入完整新名称后显式生成预览"
	} else {
		snapshot, err := t.Editor.CaptureSourceSnapshot(code.DefaultServiceLimits().Structure.Bytes)
		if err != nil {
			return err
		}
		at, err := sourceLanguageByteOffset(snapshot.Source, primary.Start(), primary.Start() != primary.End())
		if err != nil {
			return err
		}
		byteOffset = at
		seed, message = sourceRenameSeed(snapshot.Source, at)
		original = sourceLanguageOwner{tab: t, doc: t.Doc, input: t.Editor, buffer: t.Doc.Buffer, manager: a.pluginManager,
			target: snapshot.Target, route: t.sourceLanguageRoute, profile: profile, mode: t.Mode, path: t.Doc.Path,
			name: t.Name, encoding: t.Doc.Encoding, bom: t.Doc.BOM, docReadOnly: t.Doc.ReadOnly, editorReadOnly: t.Editor.ReadOnly, version: snapshot.Version}
	}
	// Constructor accommodates oversize seeds; enforce our independent ceiling
	// BEFORE calling it. No normalization, trimming or identifier rewriting.
	if len(seed) > sourceRenameNameBytes || text.UTF16Len(seed) > sourceRenameNameUnits {
		return errors.New("重命名字段初值超出预算")
	}
	a.retireSourceLanguage(nil)
	a.closeSourceLanguagePresentation()
	if err := a.openSourceLanguagePresentation(); err != nil {
		return err
	}
	original.session, original.activation, original.request = s.session, s.activation, s.request
	b := text.NewWithLimit(seed, sourceRenameNameBytes)
	b.SetHistoryLimits(32, 64<<10)
	field := editor.New(b)
	field.Label, field.MinHeight, field.ShapeLimit = "本文件符号新名称", 28, 128
	field.FontSize, field.LineNumbers, field.SoftWrap = 13, false, false
	d := &sourceRenameDialog{original: &original, utf16Offset: primary.Start(), byteOffset: byteOffset, selectedRange: primary.Start() != primary.End(), field: field, buffer: b, epoch: 1, focus: true}
	s.rename, s.owner = d, &original
	s.intent = sourceLanguageIntent{operation: code.ServiceRename, limits: code.DefaultServiceLimits()}
	s.status, s.reason, s.message = sourceLanguageIdle, "", message
	field.OnChange = func() { a.invalidateSourceRenamePreview(d, "名称已变化；请重新生成预览") }
	field.OnError = func(err error) {
		message := "名称输入未完成：" + sourceLanguageBoundText(err.Error())
		s := &a.sourceLanguage
		// Preserve this request only while its full source/name seals hold.
		// A queued action has an additional generation check; any target ABA fails.
		actionCurrent := s.action == nil
		if s.action != nil {
			actionCurrent = s.action.renameDialog == d && a.sourceLanguageActionCurrent(*s.action)
		}
		if r := d.request; r != nil && a.sourceLanguagePresentationCurrent() && a.sourceRenameRequestValid(r) == nil {
			if !actionCurrent {
				s.action = nil
			}
			s.message = message
			return
		}
		a.invalidateSourceRenamePreview(d, message)
	}
	return nil
}

func (a *App) sourceRenameOriginalValid(d *sourceRenameDialog) error {
	s := &a.sourceLanguage
	if d == nil || s.rename != d || d.original == nil || a.modalDepth > 0 || a.ShowPlugins {
		return errors.New("重命名对话框已失效")
	}
	// Request generations can advance while the original activation and opaque
	// source target stay fixed. Adjust ONLY the Host request counter for this
	// validation; never replace the original source token or metadata.
	owner := *d.original
	owner.request = s.request
	if err := a.sourceLanguageOwnerValid(&owner, true); err != nil {
		return err
	}
	if owner.doc.ReadOnly || owner.input.ReadOnly {
		return errors.New("只读源码不能重命名")
	}
	return nil
}

func (a *App) sourceRenameRequestValid(r *sourceRenameRequest) error {
	if r == nil {
		return errors.New("没有当前重命名预览")
	}
	d := r.dialog
	if a.sourceLanguage.rename != d || d == nil || d.request != r || d.epoch != r.epoch || d.field != r.field || d.buffer != r.buffer || r.field.Buffer != r.buffer {
		return errors.New("名称字段或预览代次已变化")
	}
	if err := a.sourceRenameInputError(); err != nil {
		return err
	}
	if r.field.ReadOnly {
		return errors.New("名称字段为只读")
	}
	if err := r.field.CheckSourceTarget(r.target, false); err != nil {
		return fmt.Errorf("名称输入身份已变化；请重新生成预览：%w", err)
	}
	return a.sourceRenameOriginalValid(d)
}

// Safe inside field OnChange/OnError: never releases/rebinds the calling field,
// focuses anything, settles input, or replaces the original source identity.
func (a *App) invalidateSourceRenamePreview(d *sourceRenameDialog, message string) {
	s := &a.sourceLanguage
	if d == nil || s.rename != d {
		return
	}
	d.epoch++
	d.request, d.patch, d.previewRows = nil, nil, nil
	s.request++
	owner := *d.original
	owner.request = s.request
	s.owner, s.pending, s.action = &owner, nil, nil
	s.intent = sourceLanguageIntent{operation: code.ServiceRename, limits: code.DefaultServiceLimits()}
	s.result, s.failure = code.ServiceResult{}, nil
	s.status, s.reason, s.message = sourceLanguageStale, "rename-name-or-input-changed", message
	if s.task != nil {
		s.task.abandon()
	}
	a.invalidateSourceLanguageRows()
}

func (a *App) rejectSourceRename(r *sourceRenameRequest, err error) {
	if r != nil && a.sourceLanguage.rename == r.dialog && r.dialog.request == r {
		a.invalidateSourceRenamePreview(r.dialog, sourceLanguageBoundText(err.Error()))
	}
}

func (a *App) releaseSourceRename() {
	s := &a.sourceLanguage
	d := s.rename
	if d == nil {
		return
	}
	// Detach before releasing native geometry; field callbacks cannot retire a
	// replacement dialog opened by another callback.
	s.rename = nil
	d.request, d.patch, d.previewRows = nil, nil, nil
	d.original = nil
	if d.field != nil {
		d.field.OnChange, d.field.OnError = nil, nil
		d.field.ReleaseView()
	}
	d.field, d.buffer = nil, nil
}

func (a *App) checkSourceRenameFrame() {
	d := a.sourceLanguage.rename
	if d == nil {
		return
	}
	if d.request != nil {
		if err := a.sourceRenameRequestValid(d.request); err != nil {
			a.rejectSourceRename(d.request, err)
		}
	}
	// During pending input keep its native owner and accessible field. When
	// input settles, an invalid original target requires a new explicit OPEN.
	if a.sourceRenameInputError() != nil {
		return
	}
	if err := a.sourceRenameOriginalValid(d); err != nil {
		a.retireSourceLanguage(nil)
		a.closeSourceLanguagePresentation()
		a.Message = "重命名原目标已变化；请重新打开重命名"
	}
}

func (a *App) requestSourceRename() error {
	d := a.sourceLanguage.rename
	if d == nil {
		return errors.New("请先打开本文件符号重命名")
	}
	if d.request != nil {
		if err := a.sourceRenameRequestValid(d.request); err != nil {
			a.rejectSourceRename(d.request, err)
		}
	}
	if err := a.sourceRenameInputError(); err != nil {
		return err
	}
	if err := a.sourceRenameOriginalValid(d); err != nil {
		return err
	}
	if d.field == nil || d.field.Buffer != d.buffer || d.field.ReadOnly {
		return errors.New("名称字段已替换或只读")
	}
	snapshot, err := d.field.CaptureSourceSnapshot(sourceRenameNameBytes)
	if err != nil {
		return err
	}
	if !utf8.ValidString(snapshot.Source) || text.UTF16Len(snapshot.Source) > sourceRenameNameUnits {
		a.invalidateSourceRenamePreview(d, "新名称超过 128 UTF-16 单位；不会截断或生成部分补丁")
		return errors.New("新名称必须不超过 128 UTF-16 单位")
	}
	// Identifier legality is exclusively ServiceRename's job, including empty,
	// whitespace, newline, keyword, underscore and Unicode spellings.
	a.invalidateSourceRenamePreview(d, "")
	r := &sourceRenameRequest{dialog: d, epoch: d.epoch, field: d.field, buffer: d.buffer, target: snapshot.Target, name: strings.Clone(snapshot.Source)}
	d.request = r
	return a.requestSourceLanguage(sourceLanguageIntent{rename: r})
}

func (a *App) admitSourceRename(r *sourceRenameRequest, launch func(*sourceLanguageOwner, sourceLanguageIntent, string, func(func()))) error {
	if err := a.sourceRenameRequestValid(r); err != nil {
		a.rejectSourceRename(r, err)
		return err
	}
	dispatch := a.sourceLanguageDispatcher()
	if dispatch == nil {
		err := errors.New("当前宿主没有可用的 UI 排队调度器")
		a.rejectSourceRename(r, err)
		return err
	}
	s, d := &a.sourceLanguage, r.dialog
	intent := sourceLanguageIntent{operation: code.ServiceRename, limits: code.DefaultServiceLimits(), newName: r.name, utf16Offset: d.utf16Offset, hasUTF16Offset: true, hasSelectedRange: d.selectedRange, rename: r}
	s.request++
	owner := *d.original
	owner.request = s.request
	if s.task != nil {
		// Original opaque target already passed full selection checks. Keep one
		// bounded successor, with its own name token and captured dispatcher.
		s.owner = &owner
		s.pending = &sourceLanguagePending{owner: &owner, intent: intent, dispatch: dispatch}
		s.intent, s.result, s.failure = intent, code.ServiceResult{}, nil
		s.status, s.reason = sourceLanguageRunning, "latest-pending"
		a.invalidateSourceLanguageRows()
		s.task.abandon()
		return nil
	}
	snapshot, err := owner.input.CaptureSourceSnapshot(intent.limits.Structure.Bytes)
	if err != nil {
		a.rejectSourceRename(r, err)
		return err
	}
	intent.byteOffset, err = sourceLanguageByteOffset(snapshot.Source, intent.utf16Offset, intent.hasSelectedRange)
	if err == nil && intent.byteOffset != d.byteOffset {
		err = errors.New("重命名原始字节位置已变化")
	}
	if err != nil {
		a.rejectSourceRename(r, err)
		return err
	}
	launch(&owner, intent, snapshot.Source, dispatch)
	return nil
}

// One forward cursor proves both byte and UTF-16 endpoints, original OldText,
// physical LF/CRLF lines and UTF-16 columns. No prefix materialization per row.
func proveSourceRenamePlan(source string, result code.ServiceResult, name string) ([]text.PatchEdit, []sourceLanguageRow, error) {
	bad := func() ([]text.PatchEdit, []sourceLanguageRow, error) {
		return nil, nil, errors.New("重命名服务计划的范围、文本或预算无效")
	}
	if result.Status != code.ServiceComplete || len(result.Diagnostics) != 0 || len(result.Symbols) != 0 || len(result.Completion) != 0 || result.Definition != nil || len(result.References) != 0 || result.Info != nil {
		return bad()
	}
	if result.NoOp {
		if len(result.Edits) != 0 {
			return bad()
		}
		return nil, nil, nil
	}
	limits := code.DefaultServiceLimits()
	if len(result.Edits) == 0 || len(result.Edits) > limits.RenameEdits {
		return bad()
	}
	patch := make([]text.PatchEdit, 0, len(result.Edits))
	rows := make([]sourceLanguageRow, 0, len(result.Edits))
	off, units, line, column, remaining := 0, 0, 0, 0, limits.TextBytes
	advance := func(to int) bool {
		if to < off || to > len(source) {
			return false
		}
		for off < to {
			r, n := utf8.DecodeRuneInString(source[off:])
			if r == utf8.RuneError && n == 1 || off+n > to {
				return false
			}
			off += n
			u := 1
			if r > 0xffff {
				u++
			}
			units += u
			column += u
			if r == '\n' {
				line++
				column = 0
			}
		}
		return true
	}
	for i, edit := range result.Edits {
		span := edit.Span
		if span.Bytes.Start < 0 || span.Bytes.End <= span.Bytes.Start || span.Bytes.End > len(source) || i > 0 && span.Bytes.Start <= result.Edits[i-1].Span.Bytes.Start || !advance(span.Bytes.Start) {
			return bad()
		}
		if units != span.UTF16.Start || line != span.Line || edit.OldText == "" || edit.NewText != name || edit.OldText == edit.NewText || !utf8.ValidString(edit.OldText) || !utf8.ValidString(edit.NewText) || len(edit.OldText) > remaining {
			return bad()
		}
		remaining -= len(edit.OldText)
		if len(edit.NewText) > remaining {
			return bad()
		}
		remaining -= len(edit.NewText)
		if source[span.Bytes.Start:span.Bytes.End] != edit.OldText {
			return bad()
		}
		position := sourceLanguagePosition{line: line + 1, column: column + 1}
		if !advance(span.Bytes.End) || units != span.UTF16.End {
			return bad()
		}
		patch = append(patch, text.PatchEdit{Start: span.UTF16.Start, End: span.UTF16.End, OldText: strings.Clone(edit.OldText), NewText: strings.Clone(edit.NewText)})
		label := sourceLanguageBoundText(fmt.Sprintf("%s → %s · %d:%d", edit.OldText, edit.NewText, position.line, position.column))
		rows = append(rows, sourceLanguageRow{id: sourceLanguageRowID("rename", i, span.UTF16.Start, span.UTF16.End), kind: "rename", start: span.UTF16.Start, end: span.UTF16.End, label: label, detail: label, position: position})
	}
	return patch, rows, nil
}

func (a *App) completeSourceRename(id uint64, result code.ServiceResult, failure error) {
	s := &a.sourceLanguage
	r := s.intent.rename
	if r == nil || s.owner == nil || s.owner.request != id {
		return
	}
	if err := a.sourceRenameRequestValid(r); err != nil {
		a.rejectSourceRename(r, err)
		return
	}
	if failure != nil {
		a.rejectSourceRename(r, failure)
		s.status, s.reason, s.failure = sourceLanguageHostError, "analyzer-failed", failure
		return
	}
	if err := validateSourceLanguageResult(result); err != nil {
		a.rejectSourceRename(r, err)
		s.status, s.reason, s.failure = sourceLanguageHostError, "invalid-service-payload", err
		return
	}
	if result.Status != code.ServiceComplete {
		r.dialog.patch, r.dialog.previewRows = nil, nil
		s.result, s.status, s.reason = result, string(result.Status), result.Reason
		a.invalidateSourceLanguageRows()
		return
	}
	// The task released its source before this UI notice. Capture only after
	// original-target validation, solely to prove the genuine complete plan.
	snapshot, err := r.dialog.original.input.CaptureSourceSnapshot(code.DefaultServiceLimits().Structure.Bytes)
	if err != nil {
		a.rejectSourceRename(r, err)
		return
	}
	patch, rows, err := proveSourceRenamePlan(snapshot.Source, result, r.name)
	if err != nil {
		a.rejectSourceRename(r, err)
		s.status, s.reason, s.failure = sourceLanguageHostError, "invalid-rename-plan", err
		return
	}
	r.dialog.patch, r.dialog.previewRows = patch, rows
	s.result, s.status, s.reason = result, string(result.Status), result.Reason
	if result.NoOp {
		s.message = "名称未改变"
	} else {
		s.message = fmt.Sprintf("完整预览：%d 处修改；确认后一次应用", len(patch))
	}
	a.invalidateSourceLanguageRows()
}

func (a *App) sourceRenameRows() []sourceLanguageRow {
	s := &a.sourceLanguage
	d := s.rename
	if d == nil || d.request == nil || s.result.Status != code.ServiceComplete {
		return nil
	}
	if err := a.sourceRenameRequestValid(d.request); err != nil {
		a.rejectSourceRename(d.request, err)
		return nil
	}
	if !s.rowsReady || s.rowRequest != s.request || s.rowPresentation != s.presentation {
		s.rows = d.previewRows
		s.rowsReady, s.rowRequest, s.rowPresentation = true, s.request, s.presentation
	}
	return s.rows
}

func (a *App) queueSourceRenameAction(kind string) {
	s := &a.sourceLanguage
	d := s.rename
	if d == nil || !a.sourceLanguagePresentationCurrent() {
		return
	}
	if d.request != nil {
		if err := a.sourceRenameRequestValid(d.request); err != nil {
			a.rejectSourceRename(d.request, err)
			return
		}
	}
	if err := a.sourceRenameInputError(); err != nil {
		s.message = err.Error()
		return
	}
	if kind == "rename-confirm" && (d.request == nil || s.result.Status != code.ServiceComplete || s.result.NoOp || len(d.patch) == 0) {
		return
	}
	s.action = &sourceLanguageAction{kind: kind, request: s.request, presentation: s.presentation, renameDialog: d, renameEpoch: d.epoch}
}

func (a *App) processSourceRenameAction(action *sourceLanguageAction) {
	s := &a.sourceLanguage
	if !a.sourceLanguageActionCurrent(*action) {
		if s.action == action {
			s.action = nil
		}
		return
	}
	d := s.rename
	if d.request != nil {
		if err := a.sourceRenameRequestValid(d.request); err != nil {
			a.rejectSourceRename(d.request, err)
			return
		}
	}
	if err := a.sourceRenameInputError(); err != nil {
		s.action = nil
		s.message = err.Error()
		return
	}
	s.action = nil
	switch action.kind {
	case "rename-generate":
		if err := a.requestSourceRename(); err != nil && s.rename == d {
			s.message = sourceLanguageBoundText(err.Error())
		}
	case "rename-cancel":
		a.retireSourceLanguage(nil)
		a.closeSourceLanguagePresentation()
	case "rename-confirm":
		a.confirmSourceRename(*action)
	}
}

func (a *App) confirmSourceRename(action sourceLanguageAction) {
	s := &a.sourceLanguage
	if !a.sourceLanguageActionCurrent(action) {
		return
	}
	d := s.rename
	if d == nil || d.request == nil {
		return
	}
	r := d.request
	if err := a.sourceRenameRequestValid(r); err != nil {
		a.rejectSourceRename(r, err)
		return
	}
	if s.result.Status != code.ServiceComplete || s.result.NoOp || len(d.patch) == 0 || len(d.patch) != len(s.result.Edits) || len(d.previewRows) != len(d.patch) {
		return
	}
	// Reprove the ENTIRE plan against the fixed source at confirmation. This
	// snapshot does not replace either source or name opaque target.
	owner := *d.original
	snapshot, err := owner.input.CaptureSourceSnapshot(code.DefaultServiceLimits().Structure.Bytes)
	if err != nil {
		a.rejectSourceRename(r, err)
		return
	}
	patch, _, err := proveSourceRenamePlan(snapshot.Source, s.result, r.name)
	if err != nil {
		a.rejectSourceRename(r, err)
		return
	}
	if err := a.sourceRenameRequestValid(r); err != nil {
		a.rejectSourceRename(r, err)
		return
	}
	// Consume every old payload BEFORE B3's terminal OnChange. Success returns
	// without any Host write, focus operation or expected-stale-token check.
	a.retireSourceLanguage(nil)
	a.closeSourceLanguagePresentation()
	request, presentation, session := s.request, s.presentation, s.session
	_, err = owner.input.ApplySourceEdits(owner.target, patch, editor.PreserveMappedSelections)
	if err != nil && s.request == request && s.presentation == presentation && s.session == session && a.Current() == owner.tab && owner.tab.Doc == owner.doc && owner.tab.Editor == owner.input {
		a.Message = "无法确认重命名：" + sourceLanguageBoundText(err.Error())
	}
}

func (a *App) sourceRenamePanel(c *ui.Context) {
	s := &a.sourceLanguage
	d := s.rename
	if d == nil {
		return
	}
	rows := a.sourceRenameRows()
	if s.rename != d {
		return
	}
	request, presentation, epoch := s.request, s.presentation, d.epoch
	current := func() bool {
		return s.rename == d && s.request == request && s.presentation == presentation && d.epoch == epoch && a.sourceLanguagePresentationCurrent()
	}
	inputError := a.sourceRenameInputError()
	ui.Column(c.Key(fmt.Sprintf("language-rename-%d", presentation))).MaxHeight(248).Shrink(0).Padding(5, 10).Gap(3).Background(c.Theme().Surface).Border(1, c.Theme().Border).Children(func() {
		ui.Text(c, "Go 本文件保守局部绑定重命名；不做类型检查或跨文件分析").FontSize(10).SingleLine()
		ui.Text(c, fmt.Sprintf("%s · 原版本 %d · %d 处修改", sourceLanguageBoundText(d.original.name), d.original.version, len(d.patch))).FontSize(10).SingleLine()
		ui.Text(c, "本文件符号新名称 · 最多 128 UTF-16 单位；字段 UTF-8 字节上限 512").FontSize(10).SingleLine()
		ui.Column(c.Key("rename-name-field")).Height(34).Shrink(0).Children(func() { d.field.View(c) })
		if d.focus && inputError == nil && current() {
			d.focus = false
			d.field.Focus()
		}
		ui.Row(c.Key("rename-actions")).Gap(7).Children(func() {
			if workbenchButton(c, "生成重命名预览").Label("生成重命名预览").KeepFocus().Height(25).Disabled(inputError != nil).Clicked() && current() {
				a.queueSourceRenameAction("rename-generate")
			}
			if workbenchButton(c, "确认重命名").Label("确认重命名").KeepFocus().Height(25).Disabled(inputError != nil || len(d.patch) == 0 || s.result.Status != code.ServiceComplete || s.result.NoOp).Clicked() && current() {
				a.queueSourceRenameAction("rename-confirm")
			}
			if workbenchButton(c, "取消重命名").Label("取消重命名").KeepFocus().Height(25).Disabled(inputError != nil).Clicked() && current() {
				a.queueSourceRenameAction("rename-cancel")
			}
		})
		status := a.sourceLanguageReadinessMessage()
		if s.message != "" {
			status = s.message + " · " + status
		}
		ui.Text(c, sourceLanguageBoundText(status)).Label("重命名状态").FontSize(10).SingleLine().Tooltip(status)
		ui.Text(c, fmt.Sprintf("重命名预览 · 全部 %d 处", len(rows))).Label("重命名预览").FontSize(11).Bold()
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
			if !current() || i < 0 || i >= len(rows) {
				return ""
			}
			return rows[i].label
		}
		// KeepFocus prevents pointer-down focus changes; the live input gate
		// additionally prevents row input's later explicit owner.Focus(). It
		// also covers an already-built list before the next frame retires rows.
		// This handler belongs only to the list, never to the name field, so
		// the installed name client retains its own Escape/final-text route.
		list := ui.List(c.Key("rename-preview-list"), &s.list, len(rows)).Grow(1).MinHeight(24).KeepFocus().HandleInput(func(ev ui.InputEvent) bool {
			switch ev.Kind {
			case ui.InputPointerDown, ui.InputPointerUp, ui.InputPointerMove, ui.InputSelect, ui.InputKeyDown, ui.InputKeyUp, ui.InputText, ui.InputCompose, ui.InputCommand:
				return !current() || a.sourceRenameInputError() != nil
			}
			return false // ordinary scrolling remains available without focus
		})
		// Read the explicit key BEFORE Rows installs MyGo's own Enter binding.
		// OnShortcut would run after bound list input consumed that same key.
		// Submitted/DoubleClicked are intentionally never application actions.
		enter := list.Shortcut(0, ui.KeyEnter)
		repeated := enter && c.ShortcutRepeated()
		list.OnShortcut(0, ui.KeyEscape, func() {
			if current() {
				a.queueSourceRenameAction("rename-cancel")
			}
		})
		list.Rows(func(row ui.ListRow) {
			i := row.Index
			if !current() || i < 0 || i >= len(rows) {
				return
			}
			ui.Text(row.Context, rows[i].label).Label("重命名项：" + rows[i].label).SingleLine().FillWidth().FontSize(11).Tooltip(rows[i].detail)
		})
		if current() && selected >= 0 && selected < len(rows) {
			s.selected = selected
			if enter && !repeated {
				a.queueSourceRenameAction("rename-confirm")
			}
		}
		s.listLifetime.built = true
	})
}
