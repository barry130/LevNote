package graphview

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/mindmap"
)

// The palette offers dark ink and pale fills, independent of workbench theme.
// Existing custom colors remain document content, including unsupported values.
var styleInks = [...]struct{ name, color string }{
	{"墨灰", "#243247"}, {"蓝", "#1e40af"}, {"绿", "#166534"},
	{"紫", "#6b21a8"}, {"红", "#991b1b"}, {"棕", "#78350f"},
}
var styleFills = [...]struct{ name, color string }{
	{"雾白", "#f8fafc"}, {"浅蓝", "#dbeafe"}, {"浅绿", "#dcfce7"},
	{"浅紫", "#f3e8ff"}, {"浅红", "#fee2e2"}, {"浅黄", "#fef3c7"},
}
var styleProperties = [...]struct {
	key   mindmap.StyleProperty
	label string
}{
	{mindmap.StyleTextColor, "文字色"},
	{mindmap.StyleFillColor, "底色"},
	{mindmap.StyleBorderColor, "边框色"},
}

var errStyleStale = errors.New("颜色操作的节点或文档已改变，请重新选择颜色")
var errStyleReadOnly = errors.New("当前文档只读，颜色保持不变")

type styleGesture struct {
	buffer                                 *text.Buffer
	nodeID                                 string
	version, epoch, tagEpoch, releaseEpoch uint64
	selection                              text.Range
	field                                  fieldIdentity
	input                                  tagInputIdentity
	property, choice                       int // choice -1 restores this property to its default
	active, released                       bool
	err                                    error
}

type styleValue struct {
	present        bool
	summary, color string
}

type stylePresentation struct {
	buffer  *text.Buffer
	version uint64
	nodeID  string
	loaded  bool
	values  [3]styleValue
}

func styleChoice(property, choice int) (string, string) {
	if choice < 0 {
		return "恢复默认", ""
	}
	if property == 1 {
		return styleFills[choice].name, styleFills[choice].color
	}
	return styleInks[choice].name, styleInks[choice].color
}

func styleActionLabel(property, choice int) string {
	name, _ := styleChoice(property, choice)
	if choice < 0 {
		return styleProperties[property].label + "恢复默认"
	}
	return styleProperties[property].label + "：" + name
}

func (v *view) stylePresentation() stylePresentation {
	b := v.host.Buffer
	if b == nil || v.mind == nil || v.syncFailed {
		return stylePresentation{}
	}
	if info := v.styleInfo; info.loaded && info.buffer == b && info.version == b.Version() && info.nodeID == v.selectedID {
		return info
	}
	node := v.selectedMindNode()
	if node == nil {
		return stylePresentation{}
	}
	info := stylePresentation{buffer: b, version: b.Version(), nodeID: v.selectedID, loaded: true}
	for i, property := range styleProperties {
		value := styleValue{summary: "默认"}
		if node.Style != nil {
			if raw, present := node.Style.Get(string(property.key)); present {
				value.present = true
				// Oversized/custom data stays in the source; never allocate its full
				// decoded value or put unbounded file data into a widget label.
				if len(raw) > 192 {
					value.summary = "自定义值（较长，完整内容见源码）"
				} else {
					var literal string
					if json.Unmarshal(raw, &literal) == nil {
						value.color, _ = mindmap.LiteralColor(literal)
						value.summary = markerText(literal, 96)
						if value.color == "" {
							value.summary += "（原值保留，显示默认）"
						}
					} else {
						value.summary = markerText(string(raw), 96) + "（原值保留，显示默认）"
					}
				}
			}
		}
		info.values[i] = value
	}
	v.styleInfo = info
	return info
}

func (v *view) captureStyle(info stylePresentation, property, choice int) styleGesture {
	g := styleGesture{buffer: info.buffer, nodeID: info.nodeID, version: info.version,
		epoch: v.styleEpoch, tagEpoch: v.tagEpoch, releaseEpoch: v.releaseEpoch,
		field: v.fieldIdentity(), input: v.tagInputIdentity(), property: property, choice: choice, active: true}
	if property < 0 || property >= len(styleProperties) || choice < -1 || choice >= len(styleInks) ||
		!info.loaded || !v.styleOpen || info.buffer == nil || info.buffer != v.host.Buffer ||
		info.version != info.buffer.Version() || info.buffer != v.syncedBuffer || info.version != v.lastSynced ||
		info.nodeID != v.selectedID || v.syncFailed || v.mind == nil {
		g.err = errStyleStale
		return g
	}
	g.selection = info.buffer.Selection()
	if v.CompositionActive() {
		g.err = errTagConfirm
	}
	return g
}

func (v *view) sameStyleGesture(g styleGesture) bool {
	return g.active && v.styleOpen && v.styleEpoch == g.epoch && v.tagEpoch == g.tagEpoch &&
		v.releaseEpoch == g.releaseEpoch && v.kind == kindMindmap && g.buffer != nil &&
		g.buffer == v.host.Buffer && g.nodeID == v.selectedID &&
		v.sameFieldIdentity(g.field) && v.sameTagInput(g.input)
}

func (v *view) activateStyle(g styleGesture) (err error) {
	fieldWritten := false
	defer func() {
		if err != nil && fieldWritten {
			err = fmt.Errorf("节点文字已写回，颜色未修改：%w", err)
		}
	}()
	if g.err != nil {
		return g.err
	}
	if !v.sameStyleGesture(g) || g.buffer.Version() != g.version || g.buffer.Selection() != g.selection {
		return errStyleStale
	}
	if v.host.ReadOnly {
		return errStyleReadOnly
	}
	if v.CompositionActive() {
		return errTagConfirm
	}
	if v.flushing {
		return ErrDraftConflict
	}
	if g.buffer.InGroup() {
		return text.ErrGroup
	}
	if err = v.previewSizeError(); err != nil {
		return err
	}
	if v.pendingTagDraft() {
		return v.Flush()
	}
	err = v.Flush()
	receipt := v.lastFieldReceipt
	fieldWritten = receipt.written
	if err != nil {
		return err
	}
	// A field flush may advance exactly its own transaction. Its receipt proves
	// that Changed did not replace the source, selection, inputs or lifecycle.
	if !v.sameStyleGesture(g) || v.host.ReadOnly || g.buffer.InGroup() ||
		g.buffer.Version() != receipt.version || g.buffer.Selection() != receipt.selection {
		return errStyleStale
	}
	source := g.buffer.String()
	candidate, _, err := mindmap.Parse([]byte(source), mindmap.ParseOptions{})
	if err != nil {
		return err
	}
	property := styleProperties[g.property].key
	if g.choice < 0 {
		err = candidate.ClearStyleProperty(g.nodeID, property)
	} else {
		_, color := styleChoice(g.property, g.choice)
		err = candidate.SetStyleProperty(g.nodeID, property, color)
	}
	if err != nil {
		return err
	}
	serialized, err := candidate.Serialize(mindmap.SerializeOptions{})
	if err != nil {
		return err
	}
	if len(serialized) > mindmap.DefaultMaxBytes {
		return errMindmapPreviewTooLarge
	}
	if _, _, err = mindmap.Parse(serialized, mindmap.ParseOptions{}); err != nil {
		return err
	}
	if string(serialized) == source {
		return nil
	}
	clamped, err := v.commitBufferSelection(string(serialized), true)
	if err != nil {
		return err
	}
	// Changed can release the view or install another document/draft. Never
	// restore a captured selection or refresh a replacement after that callback.
	if v.sameStyleGesture(g) && !v.host.ReadOnly && !g.buffer.InGroup() &&
		g.buffer.Version() == receipt.version+1 && g.buffer.String() == string(serialized) {
		v.styleInfo = stylePresentation{}
		v.ensureSynced(nil)
		if clamped {
			v.addWarning("修改颜色后源码已重新序列化；原选区已夹取到有效字符边界，撤销可恢复原选区")
		}
	}
	return nil
}

func (v *view) setStyleError(err error) {
	previous := v.styleError
	v.styleError = err
	if err != nil && (previous == nil || previous.Error() != err.Error()) {
		v.report(err)
	}
}

func (v *view) observeStyleInput(ev ui.InputEvent) {
	if ev.Kind == ui.InputPointerDown || ev.Kind == ui.InputPointerUp && ev.Canceled ||
		ev.Kind == ui.InputScroll || ev.Kind == ui.InputZoom ||
		ev.Kind == ui.InputKeyDown && (ev.Mods != 0 || ev.Key != ui.KeyEnter && ev.Key != ui.KeySpace) {
		v.stylePress = styleGesture{}
	}
}

func (v *view) drawStyleButton(c *ui.Context, info stylePresentation, property, choice int, disabled bool) {
	name, color := styleChoice(property, choice)
	label := styleActionLabel(property, choice)
	current := info.values[property]
	selected := !current.present && choice < 0 || choice >= 0 && current.color == color
	face := name
	if selected {
		face = "✓ " + name
	}
	button := ui.Button(c.Key(choice), face).Label(label).Grow(1).MinWidth(0).Height(28).
		Padding(3, 3).FontSize(11).Disabled(disabled || choice < 0 && !current.present).Checked(selected)
	if choice >= 0 {
		if property == 1 {
			button.Background(ui.Hex(color)).TextColor(ui.Hex("#243247"))
		} else {
			button.Background(ui.Hex("#f8fafc")).TextColor(ui.Hex(color))
		}
		button.Border(1, ui.Hex(color)).Tooltip(label + " " + color)
	} else {
		button.Tooltip("只移除此项颜色，其他样式保持原样。")
	}
	if selected {
		button.Border(2, c.Theme().Focus)
	}
	button.HandleInput(func(ev ui.InputEvent) bool {
		if ev.Kind == ui.InputScroll || ev.Kind == ui.InputZoom {
			v.stylePress = styleGesture{}
		}
		if ev.Kind == ui.InputKeyDown && ev.Mods == 0 && (ev.Key == ui.KeyEnter || ev.Key == ui.KeySpace) {
			if ev.Repeat {
				return true
			}
			v.stylePress = v.captureStyle(info, property, choice)
			v.stylePress.released = true
		} else if ev.Kind == ui.InputPointerDown && ev.Button == 0 {
			v.stylePress = v.captureStyle(info, property, choice)
		} else if ev.Kind == ui.InputPointerUp {
			if ev.Canceled {
				v.stylePress = styleGesture{}
			} else {
				v.stylePress.released = true
			}
		}
		return false
	})
	if button.Clicked() {
		g := v.stylePress
		v.stylePress = styleGesture{}
		if !g.active {
			return
		}
		err := errStyleStale
		if g.property == property && g.choice == choice {
			err = v.activateStyle(g)
		}
		// Errors must be visible, but never attach an old operation's status to
		// a view that Changed released or rebound to a different document.
		if v.host.Buffer == g.buffer && v.releaseEpoch == g.releaseEpoch && v.selectedID == g.nodeID {
			v.setStyleError(err)
		} else if err != nil {
			v.report(err)
		}
		c.Invalidate()
	}
}

func (v *view) drawStyles(c *ui.Context) {
	info := v.stylePresentation()
	theme := c.Theme()
	ui.Column(c).Key("mindmap-styles").FillWidth().Shrink(0).Gap(6).Children(func() {
		label, face := "展开颜色面板", "▸ 颜色"
		if v.styleOpen {
			label, face = "收起颜色面板", "▾ 颜色"
		}
		if ui.Button(c.Key("style-toggle"), face).Label(label).FillWidth().Height(30).
			Bind(&v.styleHandle).Expanded(v.styleOpen).Tooltip("调整选中节点的文字、底色与边框颜色。").Clicked() {
			v.styleOpen = !v.styleOpen
			v.styleEpoch++
			v.stylePress = styleGesture{}
			c.Invalidate()
		}
		if !v.styleOpen {
			return
		}
		reason := error(nil)
		switch {
		case !info.loaded:
			reason = errStyleStale
		case v.host.ReadOnly:
			reason = errStyleReadOnly
		case v.CompositionActive():
			reason = errTagConfirm
		case v.host.Buffer.InGroup():
			reason = text.ErrGroup
		}
		if reason != nil {
			ui.Text(c, reason.Error()).FontSize(11).TextColor(theme.Warning)
		}
		if v.styleError != nil {
			message := markerText(v.styleError.Error(), 512)
			ui.Text(c, message).Label("颜色操作问题").MaxLines(3).Ellipsis("…").Tooltip(message).FontSize(11).TextColor(theme.Warning)
		}
		for property, spec := range styleProperties {
			ui.Column(c.Key(property)).FillWidth().Shrink(0).Gap(4).Children(func() {
				summary := spec.label + "：" + info.values[property].summary
				ui.Text(c, summary).Label("当前" + summary).MaxLines(2).Ellipsis("…").Tooltip(summary).FontSize(11).TextColor(theme.TextMuted)
				for start := 0; start < len(styleInks); start += 3 {
					ui.Row(c).FillWidth().Shrink(0).Gap(4).Children(func() {
						for choice := start; choice < start+3; choice++ {
							v.drawStyleButton(c, info, property, choice, reason != nil)
						}
					})
				}
				ui.Row(c).FillWidth().Shrink(0).Children(func() { v.drawStyleButton(c, info, property, -1, reason != nil) })
			})
		}
		ui.Text(c, "颜色随文档保存，SVG 同步生效").FontSize(11).TextColor(theme.TextMuted)
	})
}
