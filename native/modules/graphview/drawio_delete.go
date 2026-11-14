package graphview

import (
	"errors"
	"fmt"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/drawio"
)

// deleteDrawioCell captures its target before synchronization or any host
// callback. A source edit never licenses restoring an obsolete view or draft.
func (v *view) deleteDrawioCell() bool {
	buffer := v.host.Buffer
	released, pageEpoch, epoch := v.releaseEpoch, v.pageEpoch, v.tagEpoch
	page, selected, field := v.pageIndex, v.selectedID, v.field
	input := v.fieldIdentity()
	sameOwner := func() bool {
		return v.host.Buffer == buffer && v.releaseEpoch == released &&
			v.pageEpoch == pageEpoch && v.pageIndex == page && v.tagEpoch == epoch &&
			v.selectedID == selected && v.field == field && v.sameFieldIdentity(input)
	}
	refuse := func(err error) bool {
		if sameOwner() {
			v.addWarning("未删除：" + err.Error() + "；原文仍保留，可切到源码视图继续编辑。")
			v.report(err)
		}
		return true
	}
	if buffer == nil {
		return refuse(errors.New("没有可写回的文档缓冲区"))
	}
	if v.host.ReadOnly {
		return refuse(errors.New("文档当前只读"))
	}
	if buffer.InGroup() {
		return refuse(text.ErrGroup)
	}
	if v.flushing {
		return refuse(ErrDraftConflict)
	}
	if v.CompositionActive() {
		return refuse(errors.New("请先确认正在输入的文字，再删除图形"))
	}
	if selected == "" {
		return refuse(errors.New("请先在画布中选择要删除的图形或连接线"))
	}
	version, selection := buffer.Version(), buffer.Selection()
	sameSource := func() bool {
		return sameOwner() && !v.host.ReadOnly && !buffer.InGroup() &&
			!v.CompositionActive() && buffer.Version() == version && buffer.Selection() == selection
	}
	v.ensureSynced(nil)
	if !sameSource() {
		return refuse(ErrDraftConflict)
	}
	if v.syncFailed || v.syncedBuffer != buffer || v.lastSynced != version {
		err := v.syncError
		if err == nil {
			err = errors.New("无法确认当前图纸和删除目标")
		}
		return refuse(err)
	}
	if err := v.writable(); err != nil {
		return refuse(err)
	}
	// Format refusal precedes ordinary-field Flush. An unsupported deletion
	// must not write an otherwise retained property draft or add its history.
	raw := buffer.String()
	serialized, incident, err := drawioDeletedSource(raw, page, selected)
	if err != nil {
		return refuse(err)
	}
	receipt, err := v.flushOrdinaryField()
	if err != nil {
		// The field helper reports only while its own callback receipt is valid.
		// Do not report again or resurrect state after a release/replacement.
		return true
	}
	expectedVersion := version
	if receipt.written {
		expectedVersion++
	}
	if !sameOwner() || receipt.version != expectedVersion {
		return refuse(ErrDraftConflict)
	}
	version, selection = receipt.version, receipt.selection
	if !sameSource() || v.flushing {
		return refuse(ErrDraftConflict)
	}
	if receipt.written {
		raw = buffer.String()
		serialized, incident, err = drawioDeletedSource(raw, page, selected)
		if err != nil {
			return refuse(err)
		}
	}
	value := string(serialized)
	if value == raw {
		return refuse(errors.New("删除没有产生可确认的源内容变更"))
	}
	if err := buffer.CheckReplaceUTF16(0, buffer.LenUTF16(), value); err != nil {
		return refuse(err)
	}
	if !sameSource() {
		return refuse(ErrDraftConflict)
	}
	// Reentrant canvas mutations cannot enter while Changed is observing this
	// single transaction. The existing helper retains the caret inside history.
	clamped, err := func() (bool, error) {
		v.flushing = true
		defer func() { v.flushing = false }()
		return v.commitBufferSelection(value, true)
	}()
	if err != nil {
		return refuse(err)
	}
	if !sameOwner() || v.host.ReadOnly || buffer.InGroup() || v.CompositionActive() ||
		buffer.Version() != version+1 || buffer.String() != value {
		return true
	}
	boundary := func(offset int) int {
		offset = max(0, min(offset, buffer.LenUTF16()))
		if _, err := buffer.SliceUTF16(offset, offset); errors.Is(err, text.ErrSurrogate) {
			offset--
		}
		return offset
	}
	selection = text.Range{Anchor: boundary(selection.Anchor), Head: boundary(selection.Head)}
	version++
	if buffer.Selection() != selection {
		return true
	}
	// Only after Changed's exact source/version/caret/owner proof may the
	// deleted target's old draft and graph gesture be retired by synchronization.
	v.cancelGraphGesture()
	v.selectedID = ""
	v.lastSynced = ^uint64(0)
	v.ensureSynced(nil)
	if v.host.Buffer != buffer || v.releaseEpoch != released || v.pageEpoch != pageEpoch ||
		v.pageIndex != page || buffer.Version() != version || buffer.Selection() != selection ||
		v.host.ReadOnly || buffer.InGroup() || v.syncedBuffer != buffer || v.lastSynced != version || v.syncFailed {
		return true
	}
	message := fmt.Sprintf("已删除选中图形及 %d 条关联线；可撤销。", incident)
	if clamped {
		message += "源码光标已移到前一个有效的 UTF-16 边界。"
	}
	// Visible success status uses the existing banner, not a fabricated Report
	// failure; genuine compatibility warnings remain available alongside it.
	v.warnings = append([]string{message}, v.warnings...)
	// The status row can change the canvas parent's positional identity. A
	// handle request follows the next binding instead of retaining that ID.
	v.canvasHandle.Focus()
	return true
}

func drawioDeletedSource(raw string, page int, selected string) ([]byte, int, error) {
	candidate, _, err := drawio.Parse([]byte(raw), drawio.ParseOptions{})
	if err != nil {
		return nil, 0, err
	}
	incident, err := candidate.DeleteCell(page, selected)
	if err != nil {
		return nil, 0, err
	}
	// DeleteCell stages its complete source spans and reparses before publishing.
	serialized, err := candidate.Serialize()
	return serialized, incident, err
}

// drawioEdgeAt follows the actual painted polyline in local canvas DIPs.
// pointerDown checks vertices first, since vertices are painted above edges.
func (v *view) drawioEdgeAt(localX, localY float32) (string, bool) {
	page := v.currentPage()
	canvas := ui.Rect{W: v.canvasRect.W, H: v.canvasRect.H}
	if page == nil || v.syncFailed || canvas.W <= 0 || canvas.H <= 0 ||
		!(localX >= 0 && localX <= canvas.W && localY >= 0 && localY <= canvas.H) {
		return "", false
	}
	// Budget the whole search before selecting anything. A partial search could
	// choose a lower edge while an unexamined later-painted edge covers it.
	remaining := 250_000
	for _, edge := range page.Model.Edges {
		if len(edge.Points) >= remaining {
			return "", false
		}
		remaining -= len(edge.Points) + 1
	}
	v.ensureNodeIndex(page)
	for index := len(page.Model.Edges) - 1; index >= 0; index-- {
		edge := page.Model.Edges[index]
		points, ok := v.diagramEdgePoints(edge, canvas)
		if !ok || !polylineVisible(points, canvas) {
			continue
		}
		for segment := 1; segment < len(points); segment++ {
			from, to, visible := clipSegment(points[segment-1], points[segment], canvas)
			if visible && drawioSegmentHit(localX, localY, from, to) {
				return edge.ID, true
			}
		}
	}
	return "", false
}

func drawioSegmentHit(x, y float32, from, to [2]float32) bool {
	ax, ay := float64(from[0]), float64(from[1])
	dx, dy := float64(to[0])-ax, float64(to[1])-ay
	px, py := float64(x)-ax, float64(y)-ay
	position := float64(0)
	if lengthSquared := dx*dx + dy*dy; lengthSquared > 0 {
		position = max(0, min(1, (px*dx+py*dy)/lengthSquared))
	}
	px, py = px-position*dx, py-position*dy
	return px*px+py*py <= 6*6
}
