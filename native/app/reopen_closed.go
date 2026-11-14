package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

const closedFileLimit = 16
const closedPathLimit = 128 << 10

// History is UI-owned metadata only. It must never retain a closed Tab or any
// document, presentation, draft, undo stack, bitmap, or filesystem handle.
type closedFile struct {
	id   uint64
	path string
}

const (
	reopenPending uint32 = iota
	reopenAdopted
	reopenRefused
)

// Only these small outcome fields cross the open worker/UI boundary. A queue
// finishing is distinct from UI adoption; failure does not consume history.
type reopenReceipt struct {
	entryID  uint64
	outcome  atomic.Uint32
	finished atomic.Bool
}

// Accept only a complete path on the actual host. UNC/device/root-relative
// Windows paths and mixed double-root paths never become local via cleaning.
func isNativeAbsoluteFilePath(path string) bool {
	if !filepath.IsAbs(path) {
		return false
	}
	if hasDoublePathRoot(path) {
		return false
	}
	return filepath.Separator != '\\' || isWindowsDriveAbsolute(path)
}

func closedFilePath(t *Tab) (string, string) {
	if t == nil || !t.isBinary() && (t.Doc == nil || t.Doc.IsNew || t.Doc.Path == "") {
		return "", ""
	}
	path := t.path()
	valid := func(p string) bool {
		return len(p) <= closedPathLimit && utf8.ValidString(p) &&
			isNativeAbsoluteFilePath(p) && !strings.ContainsRune(p, '\x00')
	}
	if !valid(path) {
		return "", "标签已关闭；路径过长或不是完整本地路径，未加入最近关闭的文件"
	}
	path = canonical(path)
	if !valid(path) {
		return "", "标签已关闭；路径解析后不是受支持的本地路径，未加入最近关闭的文件"
	}
	return strings.Clone(path), ""
}

// Callers compare canonical native paths. Windows keeps folded identity;
// Unix must not merge two real files that differ only by letter case.
func sameNativeFilePath(left, right string) bool {
	if filepath.Separator == '\\' {
		return strings.EqualFold(left, right)
	}
	return left == right
}

func (a *App) recordClosedFile(path string) {
	for i, entry := range a.closedFiles {
		if sameNativeFilePath(entry.path, path) {
			a.removeClosedFile(i)
			break
		}
	}
	if len(a.closedFiles) == closedFileLimit {
		a.removeClosedFile(len(a.closedFiles) - 1)
	}
	a.nextClosedID++
	a.closedFiles = append(a.closedFiles, closedFile{})
	copy(a.closedFiles[1:], a.closedFiles[:len(a.closedFiles)-1])
	a.closedFiles[0] = closedFile{id: a.nextClosedID, path: strings.Clone(path)}
}

func (a *App) removeClosedFile(index int) {
	copy(a.closedFiles[index:], a.closedFiles[index+1:])
	a.closedFiles[len(a.closedFiles)-1] = closedFile{}
	a.closedFiles = a.closedFiles[:len(a.closedFiles)-1]
}

func (a *App) closedFileIndex(id uint64) int {
	for i, entry := range a.closedFiles {
		if entry.id == id {
			return i
		}
	}
	return -1
}

func (a *App) reapReopen() {
	if a.reopening != nil && a.reopening.finished.Load() {
		if a.reopening.outcome.Load() == reopenRefused && a.closedFileIndex(a.reopening.entryID) >= 0 {
			a.Message += "；记录仍保留，可从最近关闭菜单重试或选择其他文件"
		}
		a.reopening = nil
	}
}

func (a *App) cancelReopen() {
	if a.reopening != nil {
		a.reopening.outcome.CompareAndSwap(reopenPending, reopenRefused)
	}
}

func (a *App) clearClosedFiles() {
	a.cancelReopen()
	a.reopening = nil
	clear(a.closedFiles[:cap(a.closedFiles)])
	a.closedFiles = nil
}

func (a *App) reopenBarrier() error {
	if a.shuttingDown.Load() || a.restoring {
		return errors.New("正在恢复或关闭工作区；未重新打开文件")
	}
	if a.modalDepth > 0 {
		return errors.New("请先结束当前对话框再重新打开文件")
	}
	if !a.ShowPlugins && (a.ShowFind && a.findCompositionActive() || a.ShowGoToLine && a.GoToLineComposing) {
		return errors.New("输入法正在组合文本；请先完成组合后再重新打开文件")
	}
	return nil
}

func (a *App) checkReopen(r *reopenReceipt) error {
	if r == nil {
		return nil
	}
	if a.reopening != r || r.outcome.Load() != reopenPending || a.closedFileIndex(r.entryID) < 0 {
		r.outcome.CompareAndSwap(reopenPending, reopenRefused)
		return errors.New("最近关闭记录已变化；本次重新打开已取消，请重新选择")
	}
	if err := a.reopenBarrier(); err != nil {
		r.outcome.CompareAndSwap(reopenPending, reopenRefused)
		return err
	}
	return nil
}

func (a *App) acknowledgeReopen(r *reopenReceipt) {
	if r != nil && a.reopening == r && r.outcome.CompareAndSwap(reopenPending, reopenAdopted) {
		if index := a.closedFileIndex(r.entryID); index >= 0 {
			a.removeClosedFile(index)
		}
	}
}

func (a *App) reopenAdoptionCheck(r *reopenReceipt) func() error {
	if r == nil {
		return nil
	}
	origin := a.Current()
	return func() error {
		if err := a.checkReopen(r); err != nil {
			return err
		}
		if a.Current() != origin {
			return errors.New("同步输入期间当前标签已变化；未重新打开文件")
		}
		return nil
	}
}

// id==0 is the keyboard's newest item. Menu rows always supply their exact ID.
func (a *App) reopenClosedFile(id uint64) <-chan struct{} {
	a.reapReopen()
	if a.reopening != nil {
		a.Message = "正在重新打开文件，请稍候；不会重复排队"
		return nil
	}
	if err := a.reopenBarrier(); err != nil {
		a.Message = err.Error()
		return nil
	}
	if id == 0 && len(a.closedFiles) > 0 {
		id = a.closedFiles[0].id
	}
	index := a.closedFileIndex(id)
	if index < 0 {
		a.Message = "没有可重新打开的记录；仅记录本次运行中单独关闭的已保存本地文件"
		return nil
	}
	entry := a.closedFiles[index]
	r := &reopenReceipt{entryID: entry.id}
	a.reopening = r
	a.Message = "正在按磁盘内容重新打开 " + boundedClosedLabel(filepath.Base(entry.path), 128)
	done := a.openRequests([]openRequest{{path: entry.path, reopen: r}})
	a.reapReopen()
	return done
}

func (a *App) selectOpenDuplicate(path string, request openRequest) bool {
	for index, tab := range a.Tabs {
		if !sameNativeFilePath(canonical(tab.path()), path) {
			continue
		}
		var beforeSelect func() error
		if request.reopen != nil {
			check := a.reopenAdoptionCheck(request.reopen)
			beforeSelect = func() error {
				if err := check(); err != nil {
					return err
				}
				if index >= len(a.Tabs) || a.Tabs[index] != tab {
					return errors.New("目标标签已变化；未重新打开文件")
				}
				return nil
			}
		}
		if a.selectChecked(index, beforeSelect) {
			a.acknowledgeReopen(request.reopen)
			if request.reveal != nil {
				a.revealTab(tab, *request.reveal)
			}
			if request.reopen != nil {
				a.Message = "文件已在标签中打开：" + tab.Name
			}
		}
		return true
	}
	return false
}

func boundedClosedLabel(value string, limit int) string {
	// Keep controls from introducing menu rows, without changing the path used
	// for opening. Truncation always falls on a UTF-8 boundary.
	value = strings.Map(func(r rune) rune {
		if r < ' ' || r == 127 {
			return ' '
		}
		return r
	}, value)
	if len(value) <= limit {
		return value
	}
	end := limit - len("…")
	for end > 0 && !utf8.RuneStart(value[end]) {
		end--
	}
	return value[:end] + "…"
}

func (a *App) closedFilesMenu(c *ui.Context) {
	a.reapReopen()
	blocked := a.reopenBarrier() != nil
	button := ui.Button(c.Key("closed-files-menu"), "▾").Label("最近关闭的文件").Width(26).
		Tooltip("最近关闭的已保存文件 · Ctrl+Shift+T · 按当前磁盘内容打开，不恢复已放弃的修改").Disabled(blocked)
	if blocked {
		// Disabled prevents opening, but a pointer press otherwise clears the
		// field's focus before the menu callback can run.
		button.KeepFocus()
	}
	button.PopoverMenuButton(func(m *ui.Menu) {
		m.MaxWidth(400)
		m.Item("按当前磁盘内容打开 · 不恢复已放弃的修改").Disabled(true)
		if len(a.closedFiles) == 0 {
			m.Item("暂无记录；未保存的草稿不在此列").Disabled(true)
			return
		}
		busy := a.reopening != nil
		if busy {
			m.Item("正在重新打开，请稍候…").Disabled(true)
		}
		m.Separator()
		for i, entry := range a.closedFiles {
			item := m.Item(fmt.Sprintf("%d. %s", i+1, boundedClosedLabel(filepath.Base(entry.path), 128))).
				ID(fmt.Sprintf("closed-file-%d", entry.id)).
				Subtitle(boundedClosedLabel(entry.path, 240)).Tooltip(boundedClosedLabel(entry.path, 2048)).
				Disabled(blocked || busy)
			if i == 0 {
				item.Shortcut(ui.Cmd|ui.Shift, ui.KeyT)
			}
			if item.Chosen() {
				a.reopenClosedFile(entry.id)
				return // Synchronous headless adoption can mutate the slice.
			}
		}
	})
}
