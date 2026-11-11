package app

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/session"
)

const openDocumentsQueryLabel = "搜索已打开文档"

type openDocumentRow struct {
	id                                   int
	title, path, foldedTitle, foldedPath string
	dirty, readOnly, binary, clipped     bool
	titleFoldClipped, pathFoldClipped    bool
}

type openDocumentPicker struct {
	open, composing                bool
	generation                     uint64
	originID, selectedID, selected int
	query, filteredQuery, message  string
	rows, matches                  []openDocumentRow
	list                           ui.ListState
	queryHandle                    ui.Handle
	reveal                         bool
	pendingID                      int
	pendingClose                   bool
}

// Only metadata is copied. Neither a substring of an unbounded label nor a
// document, view or binary snapshot survives in this transient list.
func openDocumentLabel(value, previous string, limit int) (string, bool) {
	var bounded [4096]byte
	n := 0
	clipped := false
	for _, r := range value {
		if unicode.IsControl(r) {
			r = ' '
		}
		size := utf8.RuneLen(r)
		if n+size > limit {
			clipped = true
			break
		}
		n += utf8.EncodeRune(bounded[n:], r)
	}
	if string(bounded[:n]) == previous {
		return previous, clipped
	}
	return string(bounded[:n]), clipped
}

func (a *App) openDocumentsAllowed() error {
	if a.shuttingDown.Load() || a.restoring {
		return errors.New("正在恢复或关闭工作区；未打开文档列表")
	}
	if a.modalDepth > 0 {
		return errors.New("请先结束当前对话框再打开文档列表")
	}
	if !a.ShowPlugins && (a.ShowFind && a.findCompositionActive() || a.ShowGoToLine && a.GoToLineComposing) || a.sourceCompositionActive(a.Current()) {
		return errors.New("输入法正在组合文本；请先完成或取消组合后再打开文档列表")
	}
	return nil
}

func (a *App) openDocumentPicker() bool {
	p := &a.openDocuments
	if p.open {
		p.queryHandle.Focus()
		return true
	}
	if err := a.openDocumentsAllowed(); err != nil {
		a.report(err)
		return false
	}
	origin := a.Current()
	if origin == nil {
		a.Message = "没有已打开的文档"
		return false
	}
	if !a.flushTab(origin, "打开文档列表") {
		return false
	}
	if a.Current() != origin || !a.tabIsOpen(origin) {
		a.Message = "同步输入期间活动文档已变化；请重新打开文档列表"
		return false
	}
	if err := a.openDocumentsAllowed(); err != nil {
		a.report(err)
		return false
	}
	a.closeOpenDocuments()
	p.open, p.originID, p.selectedID, p.selected = true, origin.ID, origin.ID, -1
	p.queryHandle.Focus() // applied after Overlay records its actual opener
	a.refreshOpenDocuments("")
	return true
}

func (a *App) closeOpenDocuments() {
	p := &a.openDocuments
	p.queryHandle.CancelFocus()
	p.list.Handle.CancelFocus()
	queryHandle, listHandle := p.queryHandle, p.list.Handle
	clear(p.rows)
	clear(p.matches)
	// MyGo retains bindings until its window closes. Keep exactly these two
	// opaque identities, with no pending focus or application row callbacks.
	*p = openDocumentPicker{generation: p.generation + 1, selected: -1, queryHandle: queryHandle, list: ui.ListState{Handle: listHandle}}
}

func (a *App) expireOpenDocuments() {
	p := &a.openDocuments
	if p.open && (a.shuttingDown.Load() || a.restoring || a.modalDepth > 0 || a.Current() == nil || a.Current().ID != p.originID) {
		a.closeOpenDocuments()
	}
}

// A list may build more rows during layout. Consume actions on the next pass,
// before constructing any rows or releasing their underlying presentation.
func (a *App) processOpenDocuments() {
	a.expireOpenDocuments()
	p := &a.openDocuments
	if !p.open {
		return
	}
	id, close, generation := p.pendingID, p.pendingClose, p.generation
	p.pendingID, p.pendingClose = 0, false
	if close {
		a.cancelOpenDocuments(generation)
	} else if id != 0 {
		a.activateOpenDocument(generation, id)
	}
}

func (a *App) refreshOpenDocuments(query string) {
	p := &a.openDocuments
	oldID, oldIndex := p.selectedID, p.selected
	var next [session.MaxTabs]openDocumentRow
	n := 0
	for _, t := range a.Tabs[:min(len(a.Tabs), session.MaxTabs)] {
		if t == nil {
			continue
		}
		var old openDocumentRow
		if n < len(p.rows) && p.rows[n].id == t.ID {
			old = p.rows[n]
		} else {
			for _, row := range p.rows {
				if row.id == t.ID {
					old = row
					break
				}
			}
		}
		title, tc := openDocumentLabel(t.Name, old.title, 256)
		path, pc := openDocumentLabel(t.path(), old.path, 4096)
		ft, fp := old.foldedTitle, old.foldedPath
		fc, fpc := old.titleFoldClipped, old.pathFoldClipped
		if title != old.title {
			ft, fc = openDocumentLabel(strings.ToLower(title), "", 256)
		}
		if path != old.path {
			fp, fpc = openDocumentLabel(strings.ToLower(path), "", 4096)
		}
		row := openDocumentRow{id: t.ID, title: title, path: path, foldedTitle: ft, foldedPath: fp, binary: t.isBinary(), clipped: tc || pc || fc || fpc, titleFoldClipped: fc, pathFoldClipped: fpc}
		if t.Doc != nil {
			row.readOnly, row.dirty = t.Doc.ReadOnly, !t.Doc.ReadOnly && t.Doc.Dirty()
		}
		next[n], n = row, n+1
	}
	changed := len(p.rows) != n
	if !changed {
		for i, row := range p.rows {
			if row != next[i] {
				changed = true
				break
			}
		}
	}
	if !changed && p.filteredQuery == query {
		return
	}
	p.rows = append(p.rows[:0], next[:n]...)
	clear(p.rows[len(p.rows):cap(p.rows)])
	folded := strings.ToLower(query)
	var previousIDs [session.MaxTabs]int
	previousCount := len(p.matches)
	for i, row := range p.matches {
		previousIDs[i] = row.id
	}
	p.matches = p.matches[:0]
	for _, row := range p.rows {
		if strings.Contains(row.foldedTitle, folded) || strings.Contains(row.foldedPath, folded) {
			p.matches = append(p.matches, row)
		}
	}
	clear(p.matches[len(p.matches):cap(p.matches)])
	membershipChanged := previousCount != len(p.matches)
	if !membershipChanged {
		for i, row := range p.matches {
			if row.id != previousIDs[i] {
				membershipChanged = true
				break
			}
		}
	}
	if membershipChanged {
		// A filter changes row membership, not the public list control identity.
		// Reuse its handle instead of registering another one on every keystroke.
		handle := p.list.Handle
		p.list = ui.ListState{Handle: handle}
		p.reveal = true
	}
	p.selected = -1
	for i, row := range p.matches {
		if row.id == oldID {
			p.selected = i
			break
		}
	}
	if p.selected < 0 && len(p.matches) > 0 {
		p.selected = 0
	}
	p.selectedID = 0
	if p.selected >= 0 {
		p.selectedID = p.matches[p.selected].id
	}
	if p.filteredQuery != query || oldID != p.selectedID || oldIndex != p.selected {
		p.reveal = true
	}
	p.filteredQuery = query
}

func (a *App) openDocumentsCurrent(generation uint64) bool {
	p := &a.openDocuments
	return p.open && p.generation == generation && a.Current() != nil && a.Current().ID == p.originID && !a.shuttingDown.Load() && !a.restoring && a.modalDepth == 0
}

func (a *App) cancelOpenDocuments(generation uint64) {
	if !a.openDocumentsCurrent(generation) {
		return
	}
	if a.openDocuments.composing {
		a.openDocuments.message = "请先完成或取消输入法组合"
		return
	}
	a.closeOpenDocuments()
}

func (a *App) activateOpenDocument(generation uint64, id int) bool {
	if !a.openDocumentsCurrent(generation) {
		return false
	}
	p := &a.openDocuments
	if p.composing {
		p.message = "请先完成或取消输入法组合"
		return false
	}
	index := -1
	for i, t := range a.Tabs {
		if t != nil && t.ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		p.message = "文档已关闭；请选择仍打开的文档"
		return false
	}
	origin, target := a.Current(), a.Tabs[index] // synchronous call only
	check := func() error {
		if !a.openDocumentsCurrent(generation) || a.Current() != origin || index >= len(a.Tabs) || a.Tabs[index] != target || target.ID != id {
			return errors.New("同步输入期间文档列表已变化；未切换，请重新选择")
		}
		return nil
	}
	// Select's current-slot path deliberately skips flushing; the picker must
	// still refuse a stale/pending draft when its selected row is current.
	ok := false
	if index == a.Active {
		if a.flushTab(origin, "选择已打开文档") {
			if err := check(); err != nil {
				a.report(err)
			} else {
				ok = true
			}
		}
	} else {
		ok = a.selectChecked(index, check)
	}
	a.closeOpenDocuments() // on refusal, MyGo restores the still-mounted opener
	return ok
}

func (a *App) openDocumentsKey(generation uint64, k ui.TextKey) bool {
	if !a.openDocumentsCurrent(generation) {
		return false
	}
	if a.openDocuments.composing {
		// OnTextKey only delivers non-text keys. Letting any reach MyGo's
		// editKey path can commit preedit without a matching InputText event.
		// Native Compose/Text alone own commit/cancel while composing.
		return true
	}
	if k.Mods != 0 {
		return false
	}
	p := &a.openDocuments
	switch k.Key {
	case ui.KeyEscape:
		p.pendingClose = true
		return true
	case ui.KeyUp, ui.KeyDown, ui.KeyEnter:
		p.query = k.Text()
		a.refreshOpenDocuments(p.query)
		if k.Key == ui.KeyEnter {
			p.pendingID = p.selectedID
			return true
		}
		if len(p.matches) > 0 {
			d := 1
			if k.Key == ui.KeyUp {
				d = -1
			}
			p.selected = max(0, min(len(p.matches)-1, p.selected+d))
			p.selectedID, p.reveal = p.matches[p.selected].id, true
		}
		return true
	}
	return false
}

func openDocumentRowLabel(row openDocumentRow) string {
	return fmt.Sprintf("选择文档 %d：%s", row.id, row.title)
}

func (a *App) openDocumentsDialog(c *ui.Context) {
	a.expireOpenDocuments()
	p := &a.openDocuments
	if !p.open {
		return
	}
	a.refreshOpenDocuments(p.query)
	generation := p.generation
	shown, clickedID, cancel := true, 0, false
	var input ui.Element
	width, height := c.Size()
	panel := ui.DialogBase(c.Key(fmt.Sprintf("open-documents-%d", generation)), &shown, func(backdrop, panel ui.Element) {
		backdrop.KeepFocus().Background(ui.RGBA(0, 0, 0, 0.25))
		panel.Width(min(float32(560), max(240, width-32))).Height(min(float32(360), max(160, height-32))).
			Background(c.Theme().Surface).Border(1, c.Theme().Border).Radius(6).
			Label("已打开文档").ObserveInput(func(ev ui.InputEvent) {
			if !a.openDocumentsCurrent(generation) {
				return
			}
			switch ev.Kind {
			case ui.InputCompose:
				p.composing = ev.Text != ""
			case ui.InputText:
				p.composing = false
			}
		})
		if panel.Shortcut(ui.Cmd, ui.KeyP) {
			p.queryHandle.Focus()
		}
		ui.Column(c).Fill().Padding(12).Gap(7).Children(func() {
			ui.Row(c).Gap(8).Children(func() {
				ui.Text(c, "已打开文档").Bold().Grow(1)
				ui.Text(c, fmt.Sprintf("%d / %d", len(p.matches), len(p.rows))).TextColor(c.Theme().TextMuted)
				if ui.Button(c, "取消").Label("关闭已打开文档").KeepFocus().Height(28).Clicked() {
					cancel = true
				}
			})
			input = ui.TextInput(c.Key("query"), &p.query).Label(openDocumentsQueryLabel).
				Placeholder("按标题或路径搜索").FillWidth().Height(32).MaxLength(128).Bind(&p.queryHandle).
				OnTextKey(func(k ui.TextKey) bool { return a.openDocumentsKey(generation, k) })
			// The native input applies changed text during this build.
			if p.query != p.filteredQuery {
				a.refreshOpenDocuments(p.query)
			}
			rows := p.matches
			p.list.Selected = &p.selected
			p.list.Key = func(i int) any { return rows[i].id }
			p.list.Label = func(i int) string { return fmt.Sprintf("文档 %d：%s", rows[i].id, rows[i].title) }
			if p.reveal && p.selected >= 0 {
				p.list.ScrollIntoView(p.selected)
				p.reveal = false
			}
			list := ui.List(c.Key("open-document-list"), &p.list, len(rows), func(i int) {
				row := rows[i]
				secondary := c.Theme().TextMuted
				if i == p.selected {
					secondary = c.Theme().AccentText
				}
				button := ui.ButtonBase(c).Role(ui.RoleButton).FillWidth().Height(48).Padding(5, 8).KeepFocus().Label(openDocumentRowLabel(row))
				button.Children(func() {
					ui.Column(c).FillWidth().Gap(2).Children(func() {
						ui.Row(c).Gap(6).Children(func() {
							ui.Text(c, row.title).SingleLine().Grow(1)
							badge := fmt.Sprintf("#%d", row.id)
							if row.id == p.originID {
								badge += " · 当前"
							}
							if row.dirty {
								badge += " · 未保存"
							}
							if row.readOnly {
								badge += " · 只读"
							}
							if row.binary {
								badge += " · 二进制"
							}
							ui.Text(c, badge).FontSize(11).TextColor(secondary)
						})
						path := row.path
						if path == "" {
							path = "尚未保存到文件"
						}
						ui.Text(c, path).FontSize(11).TextColor(secondary).SingleLine()
					})
				})
				if button.Clicked() && a.openDocumentsCurrent(generation) {
					if p.composing {
						p.message = "请先完成或取消输入法组合"
					} else {
						p.pendingID = row.id
					}
					c.Invalidate()
				}
			}).Grow(1).MinHeight(0).KeepFocus()
			if list.Submitted() && p.selected >= 0 && p.selected < len(rows) {
				clickedID = rows[p.selected].id
			}
			if p.selected >= 0 && p.selected < len(rows) {
				p.selectedID = rows[p.selected].id
			}
			if len(rows) == 0 {
				ui.Text(c, "没有匹配的已打开文档").FontSize(12)
			}
			hint := "标题 / 路径子串 · ↑↓选择 · Enter 切换 · Esc 返回"
			for _, row := range p.rows {
				if row.clipped {
					hint = "长标题或路径已截短；仅搜索显示的部分"
					break
				}
			}
			if p.message != "" {
				hint = p.message
			}
			ui.Text(c, hint).FontSize(11).TextColor(c.Theme().TextMuted).MaxLines(2)
		})
	})
	if p.composing && !input.Focused() && !input.Composing() {
		p.composing = false
	}
	if panel.Dismissed() || cancel || clickedID != 0 {
		if p.composing {
			p.message = "请先完成或取消输入法组合"
		} else if panel.Dismissed() || cancel {
			p.pendingClose = true
		} else {
			p.pendingID = clickedID
		}
		c.Invalidate()
	}
}
