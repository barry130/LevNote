package graphview

import (
	"errors"
	"fmt"
	"slices"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/drawio"
)

type drawioAction struct {
	buffer                                  *text.Buffer
	version                                 uint64
	selection                               text.Range
	releaseEpoch, pageEpoch, selectionEpoch uint64
	page                                    int
	primary                                 string
	ids                                     []string
	field                                   nodeField
	input                                   fieldIdentity
}

func (v *view) captureDrawioAction() drawioAction {
	g := drawioAction{buffer: v.host.Buffer, releaseEpoch: v.releaseEpoch, pageEpoch: v.pageEpoch,
		selectionEpoch: v.drawSelection.epoch, page: v.pageIndex, primary: v.selectedID,
		ids: slices.Clone(v.drawSelection.ids), field: v.field, input: v.fieldIdentity()}
	if g.buffer != nil {
		g.version, g.selection = g.buffer.Version(), g.buffer.Selection()
	}
	return g
}

func (v *view) sameDrawioOwner(g drawioAction) bool {
	return v.kind == kindDrawio && v.host.Buffer == g.buffer && v.releaseEpoch == g.releaseEpoch &&
		v.pageEpoch == g.pageEpoch && v.pageIndex == g.page && v.drawSelection.epoch == g.selectionEpoch &&
		v.selectedID == g.primary && slices.Equal(v.drawSelection.ids, g.ids) && v.field == g.field && v.sameFieldIdentity(g.input)
}

func (v *view) sameDrawioSource(g drawioAction) bool {
	return v.sameDrawioOwner(g) && g.buffer != nil && g.buffer.Version() == g.version && g.buffer.Selection() == g.selection
}

func (v *view) reportDrawioAction(g drawioAction, err error) {
	if err != nil && v.sameDrawioOwner(g) {
		v.addWarning(err.Error())
		v.report(err)
	}
}

// A real ordinary-field receipt is the sole permitted version transition before
// a geometric edit. Its Changed callback must leave source, field and owner intact.
func (v *view) flushDrawioAction(g drawioAction) (drawioAction, error) {
	receipt, err := v.flushOrdinaryField()
	if err != nil {
		return g, err
	}
	expected := g.version
	if receipt.written {
		expected++
	}
	if !v.sameDrawioOwner(g) || receipt.version != expected {
		return g, ErrDraftConflict
	}
	g.version, g.selection = receipt.version, receipt.selection
	if !v.sameDrawioSource(g) || v.flushing || v.CompositionActive() {
		return g, ErrDraftConflict
	}
	return g, nil
}

type drawioPositionPlanner func(*drawio.Document) ([]drawio.PositionUpdate, error)

// The planner is rerun after a label commit. No-op/unsupported geometry exits
// before Flush, so it retains both pending field text and the source redo branch.
func (v *view) applyDrawioPositions(g drawioAction, plan drawioPositionPlanner) {
	fail := func(err error) { v.reportDrawioAction(g, err) }
	if !v.sameDrawioSource(g) {
		fail(ErrDraftConflict)
		return
	}
	if v.flushing {
		fail(ErrDraftConflict)
		return
	}
	if v.host.ReadOnly {
		fail(errors.New("文档当前只读"))
		return
	}
	if g.buffer.InGroup() {
		fail(text.ErrGroup)
		return
	}
	if v.CompositionActive() {
		fail(errors.New("请先确认正在输入的文字，再调整图形"))
		return
	}
	v.ensureSynced(nil)
	if !v.sameDrawioSource(g) {
		fail(ErrDraftConflict)
		return
	}
	if v.syncFailed || v.syncedBuffer != g.buffer || v.lastSynced != g.version {
		err := v.syncError
		if err == nil {
			err = ErrDraftConflict
		}
		fail(err)
		return
	}
	prepare := func() (string, error) {
		document, _, err := drawio.Parse([]byte(g.buffer.String()), drawio.ParseOptions{})
		if err != nil {
			return "", err
		}
		updates, err := plan(document)
		if err != nil {
			return "", err
		}
		if len(updates) == 0 {
			return g.buffer.String(), nil
		}
		if err = document.UpdatePositions(g.page, updates); err != nil {
			return "", err
		}
		out, err := document.Serialize()
		return string(out), err
	}
	value, err := prepare()
	if err != nil {
		fail(err)
		return
	}
	if value == g.buffer.String() {
		return
	}
	oldVersion := g.version
	g, err = v.flushDrawioAction(g)
	if err != nil {
		return
	}
	if !v.sameDrawioSource(g) || v.host.ReadOnly || g.buffer.InGroup() {
		fail(ErrDraftConflict)
		return
	}
	if g.version != oldVersion {
		value, err = prepare()
		if err != nil {
			fail(err)
			return
		}
	}
	if value == g.buffer.String() {
		return
	}
	if err = g.buffer.CheckReplaceUTF16(0, g.buffer.LenUTF16(), value); err != nil {
		fail(err)
		return
	}
	if !v.sameDrawioSource(g) {
		fail(ErrDraftConflict)
		return
	}
	clamped, err := func() (bool, error) {
		v.flushing = true
		defer func() { v.flushing = false }()
		return v.commitBufferSelection(value, true)
	}()
	if err != nil {
		fail(err)
		return
	}
	if !v.sameDrawioOwner(g) || v.host.ReadOnly || g.buffer.InGroup() || v.CompositionActive() ||
		g.buffer.Version() != g.version+1 || g.buffer.String() != value {
		return
	}
	boundary := func(offset int) int {
		offset = max(0, min(offset, g.buffer.LenUTF16()))
		if _, err := g.buffer.SliceUTF16(offset, offset); errors.Is(err, text.ErrSurrogate) {
			offset--
		}
		return offset
	}
	expected := text.Range{Anchor: boundary(g.selection.Anchor), Head: boundary(g.selection.Head)}
	if g.buffer.Selection() != expected {
		return
	}
	g.version++
	g.selection = expected
	v.ensureSynced(nil)
	if !v.sameDrawioSource(g) {
		return
	}
	if clamped {
		v.addWarning("源码选区已夹取到有效的 UTF-16 边界；撤销可恢复原选区。")
	}
	v.canvasHandle.Focus()
}

type drawioAlignmentPress struct {
	action           drawioAction
	alignment        drawio.Alignment
	active, released bool
}

type drawioAlignmentInfo struct {
	buffer                             *text.Buffer
	version, pageEpoch, selectionEpoch uint64
	page                               int
	ready                              bool
	reason                             string
}

func (v *view) drawioAlignmentReason() string {
	if v.host.ReadOnly {
		return "当前文档只读"
	}
	if v.syncFailed || v.diagram == nil || v.host.Buffer == nil {
		return "当前图纸尚未就绪"
	}
	if len(v.drawSelection.ids) < 2 {
		return "按 Ctrl 点击图形可多选"
	}
	if v.CompositionActive() {
		return "请先确认正在输入的文字"
	}
	if v.host.Buffer.InGroup() {
		return "请先完成正在编辑的事务"
	}
	info := v.alignInfo
	if !info.ready || info.buffer != v.host.Buffer || info.version != v.host.Buffer.Version() ||
		info.page != v.pageIndex || info.pageEpoch != v.pageEpoch || info.selectionEpoch != v.drawSelection.epoch {
		info = drawioAlignmentInfo{buffer: v.host.Buffer, version: v.host.Buffer.Version(), page: v.pageIndex,
			pageEpoch: v.pageEpoch, selectionEpoch: v.drawSelection.epoch, ready: true}
		if err := v.diagram.ValidatePositionTargets(v.pageIndex, v.drawSelection.ids); err != nil {
			info.reason = err.Error()
		}
		v.alignInfo = info
	}
	return info.reason
}

var drawioAlignmentButtons = []struct {
	key, label string
	alignment  drawio.Alignment
}{
	{"drawio.align.left", "左对齐", drawio.AlignLeft},
	{"drawio.align.center-x", "水平居中", drawio.AlignCenterX},
	{"drawio.align.right", "右对齐", drawio.AlignRight},
	{"drawio.align.top", "顶对齐", drawio.AlignTop},
	{"drawio.align.center-y", "垂直居中", drawio.AlignCenterY},
	{"drawio.align.bottom", "底对齐", drawio.AlignBottom},
}

func (v *view) observeDrawioInput(ev ui.InputEvent) {
	if v.kind != kindDrawio {
		return
	}
	if ev.Kind == ui.InputPointerDown && v.drawGesture.kind != drawioNoGesture {
		v.drawGesture = drawioPointerGesture{}
	}
	if ev.Kind == ui.InputPointerDown || ev.Kind == ui.InputPointerUp && ev.Canceled ||
		ev.Kind == ui.InputScroll || ev.Kind == ui.InputZoom || ev.Kind == ui.InputKeyDown &&
		(ev.Mods != 0 || ev.Key != ui.KeyEnter && ev.Key != ui.KeySpace) {
		v.alignPress = drawioAlignmentPress{}
	}
}

func (v *view) drawDrawioAlignment(c *ui.Context) {
	reason := v.drawioAlignmentReason()
	ui.Text(c, fmt.Sprintf("已选择 %d 个图形", len(v.drawSelection.ids))).Bold().FontSize(13)
	for row := 0; row < 2; row++ {
		ui.Row(c.Key(row)).FillWidth().Shrink(0).Gap(4).Children(func() {
			for _, spec := range drawioAlignmentButtons[row*3 : row*3+3] {
				button := ui.Button(c.Key(spec.key), spec.label).Label(spec.label).Key(spec.key).KeepFocus().Grow(1).MinWidth(0).
					Height(30).FontSize(11).Padding(3, 2).Disabled(reason != "").Tooltip(spec.label)
				button.HandleInput(func(ev ui.InputEvent) bool {
					if ev.Kind == ui.InputPointerDown && ev.Button == 0 {
						v.alignPress = drawioAlignmentPress{action: v.captureDrawioAction(), alignment: spec.alignment, active: true}
					} else if ev.Kind == ui.InputPointerUp {
						if ev.Canceled {
							v.alignPress = drawioAlignmentPress{}
						} else {
							v.alignPress.released = true
						}
					} else if ev.Kind == ui.InputKeyDown && ev.Mods == 0 && (ev.Key == ui.KeyEnter || ev.Key == ui.KeySpace) {
						if ev.Repeat {
							return true
						}
						v.alignPress = drawioAlignmentPress{action: v.captureDrawioAction(), alignment: spec.alignment, active: true, released: true}
					}
					return false
				})
				if button.Clicked() {
					press := v.alignPress
					v.alignPress = drawioAlignmentPress{}
					if press.active && press.released && press.alignment == spec.alignment {
						v.applyDrawioPositions(press.action, func(d *drawio.Document) ([]drawio.PositionUpdate, error) {
							return d.PlanAlignment(press.action.page, press.action.ids, spec.alignment)
						})
					}
				}
			}
		})
	}
	if reason != "" {
		ui.Text(c, reason).FontSize(11).TextColor(c.Theme().Warning).MaxLines(4).Ellipsis("…").Tooltip(reason)
	}
	ui.Text(c, "Ctrl 点击切换选择；Shift 拖动空白框选。拖动已选图形可整组移动。").FontSize(11).TextColor(c.Theme().TextMuted)
}
