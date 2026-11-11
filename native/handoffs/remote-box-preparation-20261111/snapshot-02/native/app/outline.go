package app

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

var errOutlineHasNoSourcePosition = errors.New("此格式没有源码定位坐标，请切到阅读/分屏定位")

func (a *App) setOutlineVisible(visible bool) {
	if visible && a.Current().isBinary() {
		a.Message = "图片不提供文档大纲"
		return
	}
	if a.ShowOutline == visible {
		return
	}
	a.ShowOutline = visible
	if !visible {
		a.releaseOutline()
		return
	}
	a.OutlineSelected = -1
	a.OutlineList = ui.ListState{}
}

func (a *App) releaseOutline() {
	a.outlineGeneration++
	if a.OutlineSource != nil {
		a.OutlineSource.Release()
	}
	a.OutlineSource = nil
	a.OutlineTabID = 0
	a.OutlineBuffer = nil
	a.OutlineFactoryGeneration = 0
	a.OutlineItems = nil
	a.OutlineSnapshotVersion = 0
	a.OutlineCheckedVersion = 0
	a.OutlineTruncated = false
	a.OutlineLoaded = false
	a.OutlineError = ""
	a.OutlineSelected = -1
	a.OutlineList = ui.ListState{}
}

func (a *App) outlineSourceFor(t *Tab) (editor.OutlineSource, error) {
	if t == nil || t.Doc == nil || t.Doc.Buffer == nil {
		return nil, editor.ErrOutlineUnsupported
	}
	var factory func(editor.Host) editor.OutlineSource
	if t.Adapter != nil && t.Adapter.Outline != nil {
		factory = t.Adapter.Outline
	} else {
		factory = t.OutlineFactory
	}
	if factory == nil {
		return nil, editor.ErrOutlineUnsupported
	}
	if a.OutlineSource != nil && a.OutlineTabID == t.ID && a.OutlineBuffer == t.Doc.Buffer &&
		a.OutlineFactoryGeneration == t.OutlineFactoryGeneration {
		return a.OutlineSource, nil
	}
	a.releaseOutline()
	source := factory(a.editorHost(t))
	if source == nil {
		return nil, editor.ErrOutlineUnsupported
	}
	a.OutlineSource = source
	a.OutlineTabID = t.ID
	a.OutlineBuffer = t.Doc.Buffer
	a.OutlineFactoryGeneration = t.OutlineFactoryGeneration
	a.outlineGeneration++
	return source, nil
}

func (a *App) outlineSidebar(c *ui.Context) {
	t := c.Theme()
	if current := a.Current(); current != nil {
		_ = a.ensureOutlineSnapshot(current, "", false)
	}
	ui.Column(c.Key("outline-panel")).Grow(1).MinHeight(0).Padding(8, 10).Gap(5).Children(func() {
		if a.Current() == nil {
			ui.Text(c, "尚未打开文档。").FontSize(12).TextColor(t.TextMuted)
			return
		}
		if a.OutlineError != "" {
			ui.Text(c, a.OutlineError).FontSize(12).TextColor(t.Warning)
			if a.OutlineError != editor.ErrOutlineUnsupported.Error() && ui.Button(c, "重试大纲").Height(25).Clicked() {
				a.releaseOutline()
			}
			return
		}
		if a.OutlineTruncated {
			ui.Text(c, outlineTruncationNotice(a.Current())).
				FontSize(11).TextColor(t.Warning)
		}
		if a.OutlineLoaded && len(a.OutlineItems) == 0 {
			if !a.OutlineTruncated {
				ui.Text(c, "此文档没有可显示的大纲项。").FontSize(12).TextColor(t.TextMuted)
			}
			return
		}
		items := a.OutlineItems
		if len(items) == 0 {
			return
		}
		a.OutlineList.Selected = &a.OutlineSelected
		a.OutlineList.Key = func(i int) any {
			if i < 0 || i >= len(items) {
				return ""
			}
			return items[i].ID
		}
		a.OutlineList.Label = func(i int) string {
			if i < 0 || i >= len(items) {
				return ""
			}
			return items[i].Text
		}
		list := ui.List(c.Key("outline-list"), &a.OutlineList, len(items), func(i int) {
			if i < 0 || i >= len(items) {
				return
			}
			if a.outlineRowBuilt != nil {
				a.outlineRowBuilt(i)
			}
			item := items[i]
			indent := float32(min(max(item.Level-1, 0), 5) * 10)
			ui.Row(c).FillWidth().Height(28).Padding(4, indent).Children(func() {
				ui.Text(c, item.Text).SingleLine().Tooltip(item.Text)
			})
		}).Grow(1)
		_ = list
		if list.Submitted() {
			a.activateSelectedOutlineItem()
		}
	})
}

func outlineTruncationNotice(t *Tab) string {
	if t != nil && t.Adapter == nil && t.OutlineFactory != nil {
		return "源码大纲部分内容或标题文字已省略；上限 2 MiB、65,536 行、2048 项、每项 512 字节。完整内容请看源码。"
	}
	return fmt.Sprintf("大纲内容不完整；最多显示 %d 项，较长文字已截短，完整内容仍在文档中。", editor.MaxOutlineItems)
}

func (a *App) ensureOutlineSnapshot(t *Tab, preferredID string, force bool) error {
	if t == nil || !a.ShowOutline {
		return nil
	}
	source, err := a.outlineSourceFor(t)
	if err != nil {
		a.OutlineLoaded = true
		a.OutlineCheckedVersion = t.Doc.Buffer.Version()
		a.OutlineError = err.Error()
		a.OutlineItems = nil
		a.OutlineTruncated = false
		a.OutlineSelected = -1
		a.OutlineList = ui.ListState{}
		return err
	}
	version := t.Doc.Buffer.Version()
	if !force && a.OutlineLoaded && a.OutlineCheckedVersion == version {
		return nil
	}
	previousID := preferredID
	if previousID == "" && a.OutlineSelected >= 0 && a.OutlineSelected < len(a.OutlineItems) {
		previousID = a.OutlineItems[a.OutlineSelected].ID
	}
	snapshot, err := source.OutlineSnapshot()
	a.OutlineLoaded = true
	a.OutlineCheckedVersion = version
	if err != nil {
		a.OutlineError = fmt.Sprintf("读取大纲失败：%v", err)
		a.OutlineItems = nil
		a.OutlineTruncated = false
		a.OutlineSelected = -1
		a.OutlineList = ui.ListState{}
		return err
	}
	if snapshot.Version != version {
		err = fmt.Errorf("%w：大纲版本与文档不一致", editor.ErrOutlineStale)
		a.OutlineError = err.Error()
		a.OutlineItems = nil
		a.OutlineTruncated = false
		a.OutlineSelected = -1
		a.OutlineList = ui.ListState{}
		return err
	}
	items, truncated := boundedOutlineItems(snapshot.Items)
	a.OutlineItems = items
	a.OutlineSnapshotVersion = version
	a.OutlineTruncated = snapshot.Truncated || truncated
	a.OutlineError = ""

	selectionID := preferredID
	if selectionID == "" && previousID != "" {
		if resolved, resolveErr := source.ResolveOutlineItem(previousID); resolveErr == nil {
			selectionID = resolved.ID
		}
	}
	// CurrentOutlineProvider is an optional snapshot-time hint, not a caret
	// subscription. Refresh reconstructs it when no stable selection wins.
	a.OutlineSelected = -1
	if selectionID == "" && t.View != nil {
		if current, ok := t.View.(editor.CurrentOutlineProvider); ok {
			if id, ok := current.CurrentOutlineItem(); ok {
				selectionID = id
			}
		}
	}
	a.selectOutlineItem(selectionID, true)
	return nil
}

func boundedOutlineItems(source []editor.OutlineItem) ([]editor.OutlineItem, bool) {
	limit := min(len(source), editor.MaxOutlineItems)
	items := make([]editor.OutlineItem, 0, limit)
	truncated := len(source) > limit
	for i := 0; i < limit; i++ {
		item := source[i]
		if item.ID == "" || len(item.ID) > editor.MaxOutlineIDBytes {
			truncated = true
			continue
		}
		item.Level = max(item.Level, 1)
		item.Text, truncated = boundedOutlineLabel(item.Text, truncated)
		if item.AnchorID != "" {
			item.AnchorID, truncated = boundedOutlineLabel(item.AnchorID, truncated)
		}
		items = append(items, item)
	}
	return items, truncated
}

func boundedOutlineLabel(value string, alreadyTruncated bool) (string, bool) {
	if len(value) <= editor.MaxOutlineLabelBytes {
		return value, alreadyTruncated
	}
	const ellipsis = "…"
	limit := editor.MaxOutlineLabelBytes - len(ellipsis)
	for limit > 0 && !utf8.RuneStart(value[limit]) {
		limit--
	}
	return value[:limit] + ellipsis, true
}

func (a *App) outlineRowAt(index int) (editor.OutlineItem, bool) {
	if index < 0 || index >= len(a.OutlineItems) {
		return editor.OutlineItem{}, false
	}
	return a.OutlineItems[index], true
}

func (a *App) selectOutlineItem(id string, scroll bool) {
	a.OutlineSelected = -1
	for i, item := range a.OutlineItems {
		if item.ID == id {
			a.OutlineSelected = i
			if scroll {
				a.OutlineList.ScrollIntoView(i)
			}
			return
		}
	}
}

func (a *App) activateSelectedOutlineItem() bool {
	t := a.Current()
	if t == nil {
		a.report(errors.New("尚未打开文档，无法定位大纲项。"))
		return false
	}
	item, ok := a.outlineRowAt(a.OutlineSelected)
	if !ok {
		return false
	}
	return a.activateOutlineTarget(t, item.ID, false, "大纲定位")
}

func (a *App) activateOutlineAnchor(t *Tab, anchor, action string) bool {
	if t == nil {
		a.report(errors.New("引用所属的文档已关闭，未执行跳转。"))
		return false
	}
	return a.activateOutlineTarget(t, anchor, true, action)
}

func (a *App) activateOutlineTarget(t *Tab, id string, anchor bool, action string) bool {
	if id == "" {
		a.report(editor.ErrOutlineStale)
		return false
	}
	if t != a.Current() {
		a.report(errors.New("大纲所属标签已不再活动，未执行跳转。"))
		return false
	}
	if t.Doc == nil || t.Doc.Buffer == nil {
		a.report(fmt.Errorf("%s：%w：文档缓冲区不可用", action, editor.ErrOutlineUnsupported))
		return false
	}
	capturedTabID := t.ID
	capturedAdapter := t.Adapter
	capturedFactoryGeneration := t.OutlineFactoryGeneration
	capturedPath := t.Doc.Path
	capturedMode := t.Mode
	capturedBuffer := t.Doc.Buffer
	source, sourceErr := a.outlineSourceFor(t)
	if sourceErr != nil {
		a.report(fmt.Errorf("%s：%w", action, sourceErr))
		return false
	}
	capturedController := a.outlineGeneration
	if !a.flushTab(t, action) {
		return false
	}
	if !a.outlineNavigationContextMatches(t, capturedTabID, capturedAdapter, capturedFactoryGeneration, capturedPath, capturedMode, capturedBuffer, capturedController) {
		a.report(fmt.Errorf("%s：%w：活动文档已变化", action, editor.ErrOutlineStale))
		return false
	}
	// Flush is a lifecycle boundary. Use the controller that remains current
	// after it rather than the one observed before the fallible operation.
	source = a.OutlineSource
	if source == nil {
		a.report(fmt.Errorf("%s：%w：大纲控制器已释放", action, editor.ErrOutlineStale))
		return false
	}
	version := capturedBuffer.Version()
	if a.ShowOutline && (!a.OutlineLoaded || a.OutlineSnapshotVersion != version) {
		if err := a.ensureOutlineSnapshot(t, "", true); err != nil {
			a.report(fmt.Errorf("%s：%w", action, err))
			return false
		}
		if !a.outlineNavigationContextMatches(t, capturedTabID, capturedAdapter, capturedFactoryGeneration, capturedPath, capturedMode, capturedBuffer, capturedController) {
			a.report(fmt.Errorf("%s：%w：活动文档已变化", action, editor.ErrOutlineStale))
			return false
		}
		source = a.OutlineSource
	}

	resolveVersion := capturedBuffer.Version()
	var item editor.OutlineItem
	var err error
	if anchor {
		item, err = source.ResolveOutlineAnchor(id)
	} else {
		item, err = source.ResolveOutlineItem(id)
	}
	if err != nil {
		a.report(fmt.Errorf("%s：%w", action, err))
		return false
	}
	if !a.outlineNavigationContextMatches(t, capturedTabID, capturedAdapter, capturedFactoryGeneration, capturedPath, capturedMode, capturedBuffer, capturedController) ||
		capturedBuffer.Version() != resolveVersion {
		a.report(fmt.Errorf("%s：%w：定位期间文档已变化", action, editor.ErrOutlineStale))
		return false
	}

	a.Message = ""
	if err := a.navigateOutlineItem(t, item); err != nil {
		a.report(fmt.Errorf("%s：%w", action, err))
		return false
	}
	if !a.outlineNavigationContextMatches(t, capturedTabID, capturedAdapter, capturedFactoryGeneration, capturedPath, capturedMode, capturedBuffer, capturedController) {
		a.report(fmt.Errorf("%s：%w：定位后活动文档已变化", action, editor.ErrOutlineStale))
		return false
	}
	a.markPersistPending()
	if a.ShowOutline && (capturedBuffer.Version() != version || !a.outlineContainsID(item.ID)) {
		if err := a.ensureOutlineSnapshot(t, item.ID, true); err != nil {
			a.report(fmt.Errorf("%s后刷新大纲：%w", action, err))
			return false
		}
	}
	a.selectOutlineItem(item.ID, true)
	if a.Message == "" {
		a.Message = "已定位：" + item.Text
	}
	return true
}

func (a *App) outlineNavigationContextMatches(
	t *Tab,
	tabID int,
	adapter *editor.Adapter,
	factoryGeneration uint64,
	path string,
	mode string,
	buffer *text.Buffer,
	controllerGeneration uint64,
) bool {
	return t != nil && t == a.Current() && t.ID == tabID && t.Adapter == adapter &&
		t.OutlineFactoryGeneration == factoryGeneration &&
		t.Doc != nil && t.Doc.Path == path && t.Doc.Buffer == buffer &&
		t.Mode == mode && a.OutlineSource != nil &&
		a.OutlineTabID == tabID && a.OutlineBuffer == buffer && a.OutlineFactoryGeneration == factoryGeneration &&
		a.outlineGeneration == controllerGeneration
}

func (a *App) outlineContainsID(id string) bool {
	for _, item := range a.OutlineItems {
		if item.ID == id {
			return true
		}
	}
	return false
}

func (a *App) navigateOutlineItem(t *Tab, item editor.OutlineItem) error {
	switch t.Mode {
	case "source":
		if item.Offset < 0 {
			return errOutlineHasNoSourcePosition
		}
		return a.revealOutlineOffset(t, item.Offset)
	case "preview", "split":
		if t.Adapter == nil || t.Adapter.Create == nil {
			return editor.ErrOutlineUnsupported
		}
		if t.View == nil {
			t.View = t.Adapter.Create(a.editorHost(t))
		}
		navigator, ok := t.View.(editor.OutlineNavigator)
		if !ok {
			return editor.ErrOutlineUnsupported
		}
		if err := navigator.NavigateOutlineItem(item.ID); err != nil {
			return err
		}
		if t.Mode == "preview" {
			return nil
		}
		if item.Offset < 0 {
			a.Message = "已在图形中定位；此格式没有源码定位坐标"
			return nil
		}
		return a.revealOutlineOffset(t, item.Offset)
	default:
		return fmt.Errorf("%w：未知视图模式 %q", editor.ErrOutlineUnsupported, t.Mode)
	}
}

func (a *App) revealOutlineOffset(t *Tab, offset int) error {
	if offset < 0 || offset > t.Doc.Buffer.LenUTF16() {
		return fmt.Errorf("%w：源码位置越界", editor.ErrOutlineStale)
	}
	t.Editor.Reveal(offset, offset)
	return nil
}

func outlineErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return editor.ErrOutlineUnsupported.Error()
	}
	return message
}
