package graphview

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/mindmap"
)

const (
	maxMarkerCount        = 64
	maxMarkerBytes        = 4096
	maxMarkerSummaryBytes = 512
	markerClear           = 19
)

// The vocabulary belongs to this inspector, never to the file format. Unknown
// glyphs remain ordinary content; each action preserves their exact ordering.
var markerChoices = [...]struct {
	glyph, label, face string
	group              int
}{
	{"1️⃣", "优先级 1", "1️⃣", 1}, {"2️⃣", "优先级 2", "2️⃣", 1}, {"3️⃣", "优先级 3", "3️⃣", 1},
	{"4️⃣", "优先级 4", "4️⃣", 1}, {"5️⃣", "优先级 5", "5️⃣", 1}, {"6️⃣", "优先级 6", "6️⃣", 1},
	{"7️⃣", "优先级 7", "7️⃣", 1}, {"8️⃣", "优先级 8", "8️⃣", 1}, {"9️⃣", "优先级 9", "9️⃣", 1},
	{"🟥", "红色标记", "🟥 红", 2}, {"🟧", "橙色标记", "🟧 橙", 2}, {"🟨", "黄色标记", "🟨 黄", 2},
	{"🟩", "绿色标记", "🟩 绿", 2}, {"🟦", "蓝色标记", "🟦 蓝", 2}, {"🟪", "紫色标记", "🟪 紫", 2},
	{"⭐", "星标标记", "⭐ 星标", 0}, {"✅", "完成标记", "✅ 完成", 0},
	{"❗", "重要标记", "❗ 重要", 0}, {"❓", "疑问标记", "❓ 疑问", 0},
}

var errMarkerStale = errors.New("目标缓冲区、节点或标记已改变，请重新操作")
var errMarkerBudget = errors.New("标记超过64个或4096字节，完整内容仍保留；请在源码中查看或修改")
var errMarkerCandidate = errors.New("操作后标记将超过64个或4096字节；请在源码中修改")
var errMarkerReadOnly = errors.New("当前文档只读，不能修改标记")
var errMarkerUnavailable = errors.New("没有可编辑的选中节点；请检查文档或在源码中修改")

// No parsed node or source offset survives a press. Only a bounded defensive
// glyph snapshot and the exact immutable identities cross the Flush boundary.
type markerGesture struct {
	buffer   *text.Buffer
	nodeID   string
	icons    []string
	present  bool
	action   int
	active   bool
	released bool
	err      error
}

// Presentation is cached per source version/selected node. Idle drawing never
// copies the source or icon list, and a collapsed section builds no palette.
type markerPresentation struct {
	buffer   *text.Buffer
	version  uint64
	nodeID   string
	loaded   bool
	present  bool
	count    int
	summary  string
	selected [len(markerChoices)]bool
	err      error
}

func markerValues(node *mindmap.Node) ([]string, bool) {
	if node == nil || node.Icons == nil {
		return nil, false
	}
	return *node.Icons, true
}

// Count is checked first; the byte sum never scans beyond the editing budget.
// This must precede Icons (which defensively copies), snapshots and candidates.
func markerBudget(values []string) error {
	if len(values) > maxMarkerCount {
		return errMarkerBudget
	}
	bytes := 0
	for _, value := range values {
		if len(value) > maxMarkerBytes-bytes {
			return errMarkerBudget
		}
		bytes += len(value)
	}
	return nil
}

func markerText(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	end := max(0, limit-len("…"))
	for end > 0 && !utf8.RuneStart(value[end]) {
		end--
	}
	return value[:end] + "…"
}

func markerSummary(values []string, present bool) string {
	if len(values) == 0 {
		if present {
			return "0 个标记（已清空）；完整列表见源码。"
		}
		return "0 个标记（尚未设置）；完整列表见源码。"
	}
	const suffix = "；完整列表见源码。"
	limit := maxMarkerSummaryBytes - len(suffix)
	var out strings.Builder
	fmt.Fprintf(&out, "%d 个标记：", len(values))
	clipped := false
	for i, value := range values {
		// Bound iterations too, including huge arrays of empty strings.
		if i == maxMarkerCount {
			clipped = true
			break
		}
		if i > 0 {
			if out.Len() == limit {
				clipped = true
				break
			}
			out.WriteByte(' ')
		}
		if value == "" {
			value = "\"\""
		}
		if len(value) > limit-out.Len() {
			remaining := limit - out.Len()
			for remaining > 0 && !utf8.RuneStart(value[remaining]) {
				remaining--
			}
			out.WriteString(value[:remaining])
			clipped = true
			break
		}
		out.WriteString(value)
	}
	result := out.String()
	if clipped {
		result = markerText(result+"…", limit)
	}
	return result + suffix
}

func (v *view) markerPresentation() markerPresentation {
	buffer := v.host.Buffer
	if buffer == nil || v.syncFailed || v.mind == nil {
		v.markerInfo = markerPresentation{summary: "完整标记列表可在源码中查看。", err: errMarkerUnavailable}
		return v.markerInfo
	}
	if info := v.markerInfo; info.loaded && info.buffer == buffer && info.version == buffer.Version() && info.nodeID == v.selectedID {
		return info
	}
	info := markerPresentation{buffer: buffer, version: buffer.Version(), nodeID: v.selectedID, loaded: true}
	node := v.selectedMindNode()
	if node == nil {
		info.summary, info.err = "完整标记列表可在源码中查看。", errMarkerUnavailable
	} else {
		values, present := markerValues(node)
		info.present, info.count = present, len(values)
		info.err = markerBudget(values)
		info.summary = markerSummary(values, present)
		if info.err == nil {
			for i, choice := range markerChoices {
				info.selected[i] = slices.Contains(values, choice.glyph)
			}
		}
	}
	v.markerInfo = info
	return info
}

func markerActionLabel(action int) string {
	if action >= 0 && action < len(markerChoices) {
		return markerChoices[action].label
	}
	return "清空全部标记"
}

func (v *view) captureMarker(action int, buffer *text.Buffer, nodeID string, version uint64) markerGesture {
	target := markerGesture{buffer: buffer, nodeID: nodeID, action: action, active: true}
	// Reject a stale native control even when source changed without a Draw.
	// Refreshing here would silently turn an old displayed state into a new
	// action baseline before the user has ever seen that state.
	if buffer == nil || buffer != v.host.Buffer || nodeID != v.selectedID ||
		v.syncedBuffer != buffer || v.lastSynced != version || buffer.Version() != version || v.syncFailed || v.mind == nil {
		target.err = errMarkerStale
		return target
	}
	node, ok := v.mind.FindNode(nodeID)
	if !ok {
		target.err = errMarkerStale
		return target
	}
	values, present := markerValues(node)
	if err := markerBudget(values); err != nil {
		target.err = err
		return target
	}
	target.present = present
	target.icons = make([]string, len(values))
	for i, value := range values {
		target.icons[i] = strings.Clone(value)
	}
	return target
}

func markerInGroup(value string, group int) bool {
	if group == 0 {
		return false
	}
	for _, choice := range markerChoices {
		if choice.group == group && choice.glyph == value {
			return true
		}
	}
	return false
}

func markerCandidate(values []string, action int) ([]string, error) {
	if err := markerBudget(values); err != nil {
		return nil, err
	}
	if action == markerClear {
		return []string{}, nil
	}
	if action < 0 || action >= len(markerChoices) {
		return nil, errMarkerUnavailable
	}
	choice := markerChoices[action]
	remove := slices.Contains(values, choice.glyph)
	keep := func(value string) bool {
		if remove {
			return value != choice.glyph
		}
		return !markerInGroup(value, choice.group)
	}
	count := 0
	if !remove {
		count = 1
	}
	for _, value := range values {
		if keep(value) {
			count++
		}
	}
	if count > maxMarkerCount {
		return nil, errMarkerCandidate
	}
	bytes := 0
	if !remove {
		bytes = len(choice.glyph)
	}
	for _, value := range values {
		if keep(value) {
			if len(value) > maxMarkerBytes-bytes {
				return nil, errMarkerCandidate
			}
			bytes += len(value)
		}
	}
	// Allocate only after the complete prospective count/byte budget passed.
	candidate := make([]string, 0, count)
	for _, value := range values {
		if keep(value) {
			candidate = append(candidate, value)
		}
	}
	if !remove {
		candidate = append(candidate, choice.glyph)
	}
	return candidate, nil
}

func (v *view) activateMarker(target markerGesture) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("未完成“%s”：%w", markerActionLabel(target.action), err)
		}
	}()
	if err = v.Flush(); err != nil {
		return err
	}
	if !target.active || target.err != nil {
		if target.err != nil {
			return target.err
		}
		return errMarkerStale
	}
	if v.kind != kindMindmap || target.buffer == nil || target.buffer != v.host.Buffer || target.nodeID != v.selectedID {
		return errMarkerStale
	}
	if v.host.ReadOnly {
		return errMarkerReadOnly
	}
	if v.host.Buffer.InGroup() {
		return text.ErrGroup
	}
	if err = v.previewSizeError(); err != nil {
		return err
	}
	v.ensureSynced(nil)
	if v.syncError != nil {
		return v.syncError
	}
	if v.mind == nil || v.syncFailed || v.selectedID != target.nodeID {
		return errMarkerStale
	}
	node, ok := v.mind.FindNode(target.nodeID)
	if !ok {
		return errMarkerStale
	}
	values, present := markerValues(node)
	if err = markerBudget(values); err != nil {
		return err
	}
	if present != target.present || !slices.Equal(values, target.icons) {
		return errMarkerStale
	}
	if target.action == markerClear && len(values) == 0 {
		return nil // already true: do not create [] on an absent field
	}
	icons, err := markerCandidate(values, target.action)
	if err != nil {
		return err
	}
	if present && slices.Equal(values, icons) {
		return nil
	}
	// Mutate a fresh candidate, never the visible cache. This preserves all
	// unrelated source changes that kept this exact ordered list/presence.
	candidate, _, err := mindmap.Parse([]byte(v.host.Buffer.String()), mindmap.ParseOptions{})
	if err != nil {
		return err
	}
	if err = candidate.SetIcons(target.nodeID, icons); err != nil {
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
	clamped, err := v.commitBufferSelection(string(serialized), true)
	if err != nil {
		return err
	}
	v.finishCommitted(target.nodeID)
	if clamped {
		v.addWarning("修改标记后源码已重新序列化；原选区已夹取到有效字符边界，撤销可恢复原选区")
	}
	return nil
}

func (v *view) setMarkerError(err error) {
	previous := v.markerError
	v.markerError = err
	if err != nil && (previous == nil || previous.Error() != err.Error()) {
		v.report(err)
	}
}

func (v *view) observeMarkerInput(ev ui.InputEvent) {
	if ev.Kind == ui.InputPointerDown || ev.Kind == ui.InputPointerUp && ev.Canceled ||
		ev.Kind == ui.InputKeyDown && (ev.Mods != 0 || ev.Key != ui.KeyEnter && ev.Key != ui.KeySpace) {
		v.markerPress = markerGesture{}
	}
}

func (v *view) markerScrollInput(ev ui.InputEvent) bool {
	if ev.Kind == ui.InputScroll || ev.Kind == ui.InputZoom {
		v.markerPress = markerGesture{}
	}
	return false
}

func (v *view) drawMarkerButton(c *ui.Context, info markerPresentation, action int, disabled bool) {
	label, face, tip := markerActionLabel(action), "清空全部标记", "移除全部标记，包括自定义字形，并写入空列表 []。"
	selected := false
	if action != markerClear {
		choice := markerChoices[action]
		selected = info.selected[action]
		face = "□ " + choice.face
		tip = label + "：添加此标记；已有标记再点一次会移除该字形的全部重复项。"
		if choice.group != 0 {
			tip += "添加时替换同组标记。"
		}
		if selected {
			face = "✓ " + choice.face
		}
	}
	button := ui.Button(c.Key(action), face).Label(label).Grow(1).MinWidth(0).
		Height(30).Padding(4, 3).FontSize(11).Tooltip(tip).Disabled(disabled)
	if action != markerClear && info.err == nil {
		button.Checked(selected)
	}
	if selected {
		button.Border(2, c.Theme().TextMuted).Background(c.Theme().SurfaceHover)
	}
	// Bind only tiny identities from this displayed frame, not node pointers
	// or icon slices. The actual native gesture constructs the bounded copy.
	buffer, id, version := info.buffer, info.nodeID, info.version
	button.HandleInput(func(ev ui.InputEvent) bool {
		if ev.Kind == ui.InputScroll || ev.Kind == ui.InputZoom {
			v.markerPress = markerGesture{}
		}
		if ev.Kind == ui.InputKeyDown && ev.Mods == 0 && (ev.Key == ui.KeyEnter || ev.Key == ui.KeySpace) {
			if ev.Repeat {
				return true
			}
			v.markerPress = v.captureMarker(action, buffer, id, version)
			v.markerPress.released = true
		} else if ev.Kind == ui.InputPointerDown && ev.Button == 0 {
			v.markerPress = v.captureMarker(action, buffer, id, version)
		} else if ev.Kind == ui.InputPointerUp {
			if ev.Canceled {
				v.markerPress = markerGesture{}
			} else {
				v.markerPress.released = true
			}
		}
		return false
	})
	if button.Clicked() {
		target := v.markerPress
		v.markerPress = markerGesture{}
		if !target.active {
			return // canceled/scroll-cleared gesture has no action
		}
		if target.action != action {
			v.setMarkerError(fmt.Errorf("未完成“%s”：%w", label, errMarkerStale))
		} else {
			v.setMarkerError(v.activateMarker(target))
		}
		// Summary/error rows were built before this action. One repaint shows
		// their final state; unchanged frames never invalidate themselves.
		c.Invalidate()
	}
}

func (v *view) drawMarkers(c *ui.Context) {
	info := v.markerPresentation()
	theme := c.Theme()
	ui.Column(c).Key("mindmap-markers").FillWidth().Shrink(0).Gap(6).Children(func() {
		label, face := "展开标记面板", "▸ 标记"
		if v.markerOpen {
			label, face = "收起标记面板", "▾ 标记"
		}
		if ui.Button(c.Key("marker-toggle"), face).Label(label).FillWidth().Height(30).
			Bind(&v.markerHandle).Expanded(v.markerOpen).Tooltip("查看或编辑选中节点的标记；完整列表可在源码中查看。").Clicked() {
			v.markerOpen = !v.markerOpen
			v.markerPress = markerGesture{}
			c.Invalidate()
		}
		ui.Text(c, info.summary).Label("当前节点标记摘要").MaxLines(2).Ellipsis("…").
			Tooltip(info.summary).FontSize(11).TextColor(theme.TextMuted)
		reason := info.err
		if reason == nil && v.host.ReadOnly {
			reason = errMarkerReadOnly
		}
		if reason == nil && v.host.Buffer != nil && v.host.Buffer.InGroup() {
			reason = errors.New("其他文本事务正在进行，暂时不能修改标记")
		}
		if reason != nil {
			ui.Text(c, reason.Error()).FontSize(11).TextColor(theme.Warning)
		}
		if v.markerError != nil {
			message := markerText(v.markerError.Error(), maxMarkerSummaryBytes)
			ui.Text(c, message).Label("标记操作问题").MaxLines(3).Ellipsis("…").Tooltip(message).FontSize(11).TextColor(theme.Warning)
		}
		if !v.markerOpen {
			return
		}
		for _, group := range [...]struct {
			label               string
			start, end, columns int
		}{
			{"优先级（同组互斥）", 0, 9, 3},
			{"色标（同组互斥）", 9, 15, 3},
			{"符号（可多选）", 15, 19, 2},
		} {
			ui.Text(c, group.label).FontSize(11).TextColor(theme.TextMuted)
			for start := group.start; start < group.end; start += group.columns {
				ui.Row(c).FillWidth().Shrink(0).Gap(4).Children(func() {
					for i := start; i < min(start+group.columns, group.end); i++ {
						v.drawMarkerButton(c, info, i, reason != nil)
					}
				})
			}
		}
		ui.Row(c).FillWidth().Shrink(0).Children(func() {
			v.drawMarkerButton(c, info, markerClear, reason != nil || info.count == 0)
		})
	})
}
