package graphview

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/search"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

const (
	fieldSearchToggle = iota
	fieldSearchFind
	fieldSearchPrevious
	fieldSearchNext
	fieldSearchReplace
	fieldSearchReplaceAll
	fieldSearchRegex
	fieldSearchCase
	fieldSearchWholeWord
	fieldSearchActionCount
)

type fieldSearchPress struct {
	owner                            fieldSearchOwner
	token                            editor.FieldSearchToken
	query, replacement               *text.Buffer
	queryEditor, replacementEditor   *editor.Editor
	queryVersion, replacementVersion uint64
	session                          uint64
	open                             bool
	opts                             search.Options
	index, action                    int
	bounds                           ui.Rect
	active                           bool
	err                              error
}

// Compare only across Changed, before the caller performs its own normal sync.
// A callback may publish a new draft/preview without starting another Find.
// Error text is sufficient here: UI actions clear the previous error on entry.
// Keep the snapshot comparable even for errors with non-comparable values.
type fieldSearchPublication struct {
	errorText        string
	hasError         bool
	notice           string
	persistentNotice bool
	index            int
	edge             *editor.Editor
	draft            fieldIdentity
	selectedID       string
	field            nodeField
}

func (v *view) fieldSearchPublication() fieldSearchPublication {
	s := &v.search
	p := fieldSearchPublication{hasError: s.err != nil, notice: s.notice, persistentNotice: s.persistentNotice, index: s.index, edge: s.edgeInput, draft: v.fieldIdentity(), selectedID: v.selectedID, field: v.field}
	if s.err != nil {
		p.errorText = s.err.Error()
	}
	return p
}

func (v *view) initFieldSearchInputs() {
	s := &v.search
	if s.query != nil {
		return
	}
	create := func(label string, replacement bool) *editor.Editor {
		limit := editor.DefaultFieldSearchLimits().QueryBytes
		if replacement {
			limit = editor.DefaultFieldSearchLimits().ReplacementBytes
		}
		input := editor.New(text.NewWithLimit("", limit))
		input.Label = label
		input.LineNumbers = false
		input.FontSize = 12
		input.MinHeight = 30
		input.ShapeLimit = 512
		input.OnError = func(err error) {
			s.err = err
			if replacement {
				s.replacementFailed = true
			} else {
				s.queryFailed = true
			}
		}
		input.OnChange = func() {
			if !replacement {
				v.invalidateFieldSearch()
			}
			if replacement {
				s.replacementFailed = false
			} else {
				s.queryFailed = false
			}
		}
		return input
	}
	s.query = create("字段查找文字", false)
	s.replacement = create("字段替换文字", true)
}
func (v *view) fieldSearchCompositionActive() bool {
	if v.search.queryComposing || v.search.replacementComposing {
		return true
	}
	for _, input := range []*editor.Editor{v.search.query, v.search.replacement} {
		if input != nil {
			if _, active := input.MarkedRange(); active {
				return true
			}
		}
	}
	return false
}
func (v *view) releaseFieldSearch() {
	v.search.session++
	v.invalidateFieldSearch()
	for _, input := range []*editor.Editor{v.search.query, v.search.replacement, v.search.edgeInput} {
		if input != nil {
			input.ReleaseView()
		}
	}
	v.search.edgeInput = nil
	v.search.queryComposing, v.search.replacementComposing = false, false
}
func (v *view) observeFieldSearchInput(ev ui.InputEvent) {
	if ev.Kind == ui.InputScroll || ev.Kind == ui.InputZoom || ev.Kind == ui.InputPointerUp && ev.Canceled {
		v.search.press = fieldSearchPress{}
	}
}
func (v *view) captureFieldSearchAction(action int) fieldSearchPress {
	v.initFieldSearchInputs()
	s := &v.search
	target := fieldSearchPress{owner: v.fieldSearchOwner(), query: s.query.Buffer, replacement: s.replacement.Buffer, queryEditor: s.query, replacementEditor: s.replacement, queryVersion: s.query.Buffer.Version(), replacementVersion: s.replacement.Buffer.Version(), session: s.session, open: s.open, opts: s.opts, index: s.index, action: action, active: true}
	target.owner.epoch = s.epoch
	if s.results != nil {
		target.token = s.results.result.Token
	}
	return target
}
func (v *view) sameFieldSearchAction(target fieldSearchPress) bool {
	s := &v.search
	return v.sameFieldSearchOwner(target.owner) && s.session == target.session && s.open == target.open && s.query != nil && s.replacement != nil && s.query == target.queryEditor && s.replacement == target.replacementEditor && s.query.Buffer == target.query && s.replacement.Buffer == target.replacement && s.query.Buffer.Version() == target.queryVersion && s.replacement.Buffer.Version() == target.replacementVersion && s.opts == target.opts
}

func (v *view) runFieldSearchAction(target fieldSearchPress) {
	v.syncFieldSearchOwner()
	s := &v.search
	if target.err != nil {
		s.err = target.err
		return
	}
	if !v.sameFieldSearchAction(target) {
		s.err = editor.ErrFieldSearchStale
		return
	}
	if v.fieldSearchBusy() {
		s.err = editor.ErrFieldSearchDraft
		return
	}
	s.err = nil
	switch target.action {
	case fieldSearchToggle:
		s.session++
		s.open = !s.open
		if s.open {
			s.query.Focus()
		} else {
			s.query.CancelFocus()
			s.replacement.CancelFocus()
		}
	case fieldSearchRegex, fieldSearchCase, fieldSearchWholeWord:
		switch target.action {
		case fieldSearchRegex:
			s.opts.Regex = !s.opts.Regex
		case fieldSearchCase:
			s.opts.CaseSensitive = !s.opts.CaseSensitive
		case fieldSearchWholeWord:
			s.opts.WholeWord = !s.opts.WholeWord
		}
		v.invalidateFieldSearch()
	case fieldSearchFind:
		limits := editor.DefaultFieldSearchLimits()
		if s.query.Buffer.LenBytes() > limits.QueryBytes {
			s.err = editor.ErrFieldSearchBudget
			return
		}
		opts := s.opts
		opts.Query = s.query.Buffer.String()
		_, s.err = v.FindFields(opts, limits)
	case fieldSearchPrevious, fieldSearchNext:
		r := s.results
		if r == nil || r.result.Token != target.token {
			s.err = editor.ErrFieldSearchStale
			return
		}
		if len(r.hits) == 0 {
			s.err = fmt.Errorf("没有匹配的字段")
			return
		}
		index := target.index
		if target.action == fieldSearchNext {
			index = (index + 1) % len(r.hits)
		} else {
			if index < 0 {
				index = 0
			}
			index = (index - 1 + len(r.hits)) % len(r.hits)
		}
		err := v.RevealFieldMatch(target.token, index)
		if err == nil || s.err != nil {
			return // success may have a fresh token; never clear a newer error
		}
		if !v.sameFieldSearchAction(target) {
			// After a committed unfold, only its still-current receipt can
			// publish an error (including a failed automatic refresh).
			receipt := s.commitReceipt
			if !v.sameFieldSearchReceipt(receipt) || receipt.previousOwner != target.owner || receipt.token != target.token {
				return
			}
			s.commitReceipt = nil // the initiating action consumed the error receipt
		}
		s.err = err
	case fieldSearchReplace, fieldSearchReplaceAll:
		if s.replacement.Buffer.LenBytes() > editor.DefaultFieldSearchLimits().ReplacementBytes {
			s.err = editor.ErrFieldSearchBudget
			return
		}
		index := target.index
		if index < 0 {
			index = 0
		}
		result, err := v.ReplaceFieldMatch(target.token, index, s.replacement.Buffer.String(), target.action == fieldSearchReplaceAll)
		if result.Changed {
			receipt := s.commitReceipt
			if !v.sameFieldSearchReceipt(receipt) || receipt.previousOwner != target.owner || receipt.token != target.token {
				return
			}
		} else if !v.sameFieldSearchOwner(target.owner) || s.results == nil || s.results.result.Token != target.token || s.query == nil || s.replacement == nil || s.query.Buffer != target.query || s.replacement.Buffer != target.replacement || s.query.Buffer.Version() != target.queryVersion || s.replacement.Buffer.Version() != target.replacementVersion || s.opts != target.opts {
			return
		}
		s.err = err
		if result.Changed && err == nil && result.Undoable {
			s.notice = fmt.Sprintf("已替换%d处；一次撤销可恢复。", result.Count)
			if result.SourceSelectionClamped {
				s.notice += " 源码选区已夹取到有效字符边界。"
			}
		}
		if !result.Changed && err == nil && !s.persistentNotice {
			s.notice = fmt.Sprintf("匹配%d处，字段内容未改变。", result.Count)
		}
	}
}

// KeepFocus prevents default pointer focus changes before HandleInput runs.
// HandleInput captures a native down against the displayed owner. Rebuilding a
// button between down/up never replaces this captured owner with a fresh one.
func (v *view) drawFieldSearchButton(c *ui.Context, action int, face, label string, checked *bool) {
	handle := &v.search.handles[action]
	target := v.captureFieldSearchAction(action)
	button := ui.Button(c.Key(action), face).Label(label).Shrink(0).Height(28).Padding(4, 7).FontSize(11).KeepFocus().Bind(handle)
	if checked != nil {
		button.Checked(*checked)
		if *checked {
			button.Border(1, c.Theme().TextMuted)
		}
	}
	button.HandleInput(func(ev ui.InputEvent) bool {
		switch ev.Kind {
		case ui.InputPointerDown:
			if ev.Button != 0 {
				return false
			}
			press := target
			press.bounds = handle.Bounds(c)
			if v.fieldSearchBusy() {
				press.err = editor.ErrFieldSearchDraft
			} else {
				handle.Focus()
			}
			v.search.press = press
			return true
		case ui.InputPointerUp:
			press := v.search.press
			v.search.press = fieldSearchPress{}
			bounds := handle.Bounds(c)
			if !ev.Canceled && press.active && press.action == action && bounds == press.bounds && bounds.Contains(bounds.X+ev.X, bounds.Y+ev.Y) {
				v.runFieldSearchAction(press)
				c.Invalidate()
			}
			return true
		case ui.InputKeyDown:
			if ev.Mods == 0 && (ev.Key == ui.KeyEnter || ev.Key == ui.KeySpace) {
				if !ev.Repeat {
					v.runFieldSearchAction(target)
					c.Invalidate()
				}
				return true
			}
		case ui.InputScroll, ui.InputZoom:
			v.search.press = fieldSearchPress{}
		}
		return false
	})
}
func (v *view) drawFieldSearchToggle(c *ui.Context) {
	face := "字段查找"
	if v.search.open {
		face = "收起查找"
	}
	v.drawFieldSearchButton(c, fieldSearchToggle, face, "打开或收起字段查找替换", nil)
}
func (v *view) drawFieldSearchInput(c *ui.Context, input *editor.Editor, replacement bool) {
	blocked := v.fieldSearchGraphDraftBusy()
	other := v.search.replacement
	if replacement {
		other = v.search.query
		blocked = blocked || v.search.queryComposing || v.search.queryFailed
	} else {
		blocked = blocked || v.search.replacementComposing || v.search.replacementFailed
	}
	if other != nil {
		_, marked := other.MarkedRange()
		blocked = blocked || marked || other.Buffer.InGroup()
	}
	if blocked {
		// A different editor owns pending input. A KeepFocus guard occupies
		// the same geometry and prevents pointer focus from unmarking it.
		ui.Box(c).Width(210).Height(34).Border(1, c.Theme().Border).KeepFocus().
			Label(input.Label).Role(ui.RoleTextField).HandleInput(func(ev ui.InputEvent) bool {
			if ev.Kind == ui.InputPointerDown || ev.Kind == ui.InputPointerUp {
				v.search.err = editor.ErrFieldSearchDraft
				c.Invalidate()
				return true
			}
			return false
		}).Children(func() { ui.Text(c, "请先完成其它输入").FontSize(11).Padding(7) })
		return
	}
	box := ui.Column(c).Width(210).Height(34).Border(1, c.Theme().Border).ObserveInput(func(ev ui.InputEvent) {
		if ev.Kind == ui.InputCompose {
			if replacement {
				v.search.replacementComposing = ev.Text != ""
			} else {
				v.search.queryComposing = ev.Text != ""
			}
			if !replacement {
				v.invalidateFieldSearch()
			}
		} else if ev.Kind == ui.InputKeyDown && ev.Key == ui.KeyEscape {
			if replacement {
				v.search.replacementFailed, v.search.replacementComposing = false, false
			} else {
				v.search.queryFailed, v.search.queryComposing = false, false
			}
		} else if ev.Kind == ui.InputText {
			if replacement {
				v.search.replacementComposing = false
			} else {
				v.search.queryComposing = false
			}
		}
	})
	if box.Shortcut(0, ui.KeyTab) {
		if v.fieldSearchCompositionActive() {
			v.search.err = editor.ErrFieldSearchDraft
		} else if replacement {
			v.search.handles[fieldSearchReplace].Focus()
		} else {
			v.search.handles[fieldSearchFind].Focus()
		}
	}
	if box.Shortcut(ui.Shift, ui.KeyTab) {
		if v.fieldSearchCompositionActive() {
			v.search.err = editor.ErrFieldSearchDraft
		} else if replacement {
			v.search.handles[fieldSearchWholeWord].Focus()
		} else {
			v.search.handles[fieldSearchToggle].Focus()
		}
	}
	if box.Shortcut(0, ui.KeyEnter) {
		if v.fieldSearchCompositionActive() {
			v.search.err = editor.ErrFieldSearchDraft
		} else {
			action := fieldSearchFind
			if replacement {
				action = fieldSearchReplace
			}
			v.runFieldSearchAction(v.captureFieldSearchAction(action))
		}
	}
	box.Children(func() { input.View(c) })
}
func (v *view) drawFieldSearch(c *ui.Context) {
	s := &v.search
	if !s.open {
		if s.err != nil {
			ui.Text(c, s.err.Error()).Label("字段查找操作问题："+s.err.Error()).FontSize(11).MaxLines(2).Ellipsis("…").Tooltip(s.err.Error()).TextColor(c.Theme().Warning).Padding(4, 12)
		}
		return
	}
	v.initFieldSearchInputs()
	theme := c.Theme()
	ui.Column(c).Key("graph-field-search").FillWidth().Shrink(0).Gap(4).Padding(5, 12).Background(theme.Surface).Border(1, theme.Border).Children(func() {
		ui.Row(c).Wrap().Gap(5).Children(func() {
			v.drawFieldSearchInput(c, s.query, false)
			v.drawFieldSearchButton(c, fieldSearchFind, "查找", "查找图字段", nil)
			v.drawFieldSearchButton(c, fieldSearchPrevious, "上一处", "上一个字段匹配", nil)
			v.drawFieldSearchButton(c, fieldSearchNext, "下一处", "下一个字段匹配", nil)
			v.drawFieldSearchButton(c, fieldSearchRegex, "正则", "字段查找正则表达式", &s.opts.Regex)
			v.drawFieldSearchButton(c, fieldSearchCase, "区分大小写", "字段查找区分大小写", &s.opts.CaseSensitive)
			v.drawFieldSearchButton(c, fieldSearchWholeWord, "整词", "字段查找整词匹配", &s.opts.WholeWord)
		})
		ui.Row(c).Wrap().Gap(5).Children(func() {
			v.drawFieldSearchInput(c, s.replacement, true)
			v.drawFieldSearchButton(c, fieldSearchReplace, "替换当前", "替换当前字段匹配", nil)
			v.drawFieldSearchButton(c, fieldSearchReplaceAll, "全部替换", "替换全部字段匹配", nil)
			status := "尚未查找（Enter查找；Tab到操作按钮）"
			if r := s.results; r != nil {
				status = fmt.Sprintf("%d/%d处", s.index+1, r.result.Count)
				if r.result.Truncated {
					status += "；结果已截断，不能全部替换"
				}
				if r.result.PreviewTruncated {
					status += "；预览已截断"
				}
			}
			ui.Text(c, status).Label("字段查找结果状态：" + status).FontSize(11).TextColor(theme.TextMuted)
		})
		scope := "思维导图：先序纯文本主题及备注；不搜JSON、链接、标签和非文本字段。引擎只读预览尚不支持字段查找；只读隐藏目标不能展开。"
		if v.kind == kindDrawio {
			scope = "draw.io：仅当前页已解码的纯文本节点及连接线标签；不搜XML、其它页、富HTML、图片、stencil和不透明字段；格式只读页拒绝替换。"
		}
		if s.results != nil {
			scope = s.results.result.Scope
		}
		ui.Column(c).Label("字段查找范围说明").Role(ui.RoleGroup).FillWidth().Children(func() {
			ui.Text(c, scope).FontSize(11).MaxLines(3).Ellipsis("…").Tooltip(scope).TextColor(theme.TextMuted)
		})
		if r := s.results; r != nil && s.index >= 0 && s.index < len(r.hits) {
			hit := r.hits[s.index]
			field := r.fields[hit.field]
			label := field.field.inputLabel()
			if field.edge {
				label = "连接线标签"
			}
			ui.Text(c, fmt.Sprintf("%s · %s · UTF-16 %d–%d · %s", field.id, label, hit.match.Start, hit.match.End, hit.preview)).Label("当前字段匹配").FontSize(11).MaxLines(1).Ellipsis("…")
		}
		if s.edgeInput != nil {
			ui.Text(c, "已定位连接线标签；只读展示，可用上方替换操作修改。").FontSize(11)
			ui.Column(c).Height(45).FillWidth().Border(1, theme.Border).Children(func() { s.edgeInput.View(c) })
		}
		if s.err != nil {
			ui.Text(c, s.err.Error()).Label("字段查找操作问题：" + s.err.Error()).FontSize(11).MaxLines(3).Ellipsis("…").Tooltip(s.err.Error()).TextColor(theme.Warning)
		}
		if s.notice != "" {
			ui.Text(c, s.notice).Label("字段替换结果说明：" + s.notice).FontSize(11).MaxLines(2).Ellipsis("…").Tooltip(s.notice).TextColor(theme.Warning)
		}
	})
}
