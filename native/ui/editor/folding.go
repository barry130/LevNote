package editor

import (
	"errors"
	"fmt"
	"sort"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

const MaxFoldCandidates = 4096
const MaxFoldBytes = 2 << 20

// FoldSnapshot owns an immutable source string. Providers run synchronously on
// the UI thread, must bound their work, and must never read or mutate a live
// Buffer. The editor checks LenBytes before taking this snapshot.
type FoldSnapshot struct {
	Source    string
	Version   uint64
	LineCount int
}
type FoldProvider func(FoldSnapshot) (FoldResult, error)

// FoldRange uses original, half-open UTF-16 delimiter spans and zero-based
// logical lines. Only [OpenLine+1, CloseLine) is hidden.
type FoldRange struct {
	OpenLine, CloseLine                      int
	OpenStart, OpenEnd, CloseStart, CloseEnd int
}
type FoldResult struct {
	Complete, Unsupported bool
	Diagnostics           int
	Folds                 []FoldRange
}

// FoldTarget is an opaque, single-candidate identity. Retained UI actions must
// use this token rather than treating a logical line as a stable identity.
type FoldTarget struct {
	buffer                                                *text.Buffer
	provider                                              *foldProviderIdentity
	version, generation, candidates, foldView, projection uint64
	span                                                  FoldRange
}

// A nonzero sized identity keeps independent Editors and reinstalled providers
// distinct even when the Buffer and the caller's generation are shared.
type foldProviderIdentity struct{ marker byte }
type FoldCandidate struct {
	Range     FoldRange
	Collapsed bool
	Target    FoldTarget
}

type foldInterval struct {
	start, end                int
	hiddenBefore, extraBefore int
}
type foldControlPress struct {
	identity       FoldTarget
	selection      text.Range
	compositionGen uint64
	key            ui.Key
	action         int
}

type foldState struct {
	provider                           FoldProvider
	identity                           *foldProviderIdentity
	generation, candidates, projection uint64
	maxBytes                           int
	buffer                             *text.Buffer
	version                            uint64
	ready                              bool
	viewGeneration                     uint64
	status                             string
	reported                           map[string]bool
	ranges                             []FoldRange
	collapsed                          map[FoldRange]bool
	intervals                          []foldInterval
	pointer                            *FoldTarget
	// The last rendered identity rejects a press on pixels from an older frame.
	painted FoldTarget
}

var ErrFoldStale = errors.New("折叠目标已变化；请重新分析折叠后重试")

type foldAnalysisError struct{ message string }

func (err *foldAnalysisError) Error() string { return err.message }

// SetFoldProvider explicitly enables folding and analyzes once. Nil disables
// folding and releases all candidates. Every installation has a new internal
// epoch, including an external generation A→B→A cycle. No timer is installed.
func (e *Editor) SetFoldProvider(provider FoldProvider, generation uint64, maxBytes int) error {
	if err := e.pendingMultiIssue(); err != nil {
		return err
	}
	e.folds = &foldState{
		provider: provider, identity: &foldProviderIdentity{marker: 1},
		generation: generation, maxBytes: min(maxBytes, MaxFoldBytes),
	}
	e.folds.buffer, e.folds.version, e.folds.viewGeneration = e.Buffer, e.Buffer.Version(), e.viewGeneration
	e.services.Invalidate()
	if provider == nil {
		return nil
	}
	return e.RefreshFolds()
}

// syncFolds only invalidates; drawing and geometry queries never analyze text.
func (e *Editor) syncFolds() {
	f := e.folds
	if f == nil || f.provider == nil {
		return
	}
	if f.buffer != e.Buffer || f.version != e.Buffer.Version() || f.viewGeneration != e.viewGeneration {
		f.buffer, f.version, f.viewGeneration = e.Buffer, e.Buffer.Version(), e.viewGeneration
		f.candidates++
		f.ready = false
		f.ranges, f.intervals, f.collapsed = nil, nil, nil
		f.reported = nil
		f.status = "文档已变化；暂时显示全文，请重新分析折叠"
	}
}
func (e *Editor) foldIdentity() FoldTarget {
	e.syncFolds()
	if f := e.folds; f != nil {
		return FoldTarget{buffer: e.Buffer, version: e.Buffer.Version(), provider: f.identity, generation: f.generation, candidates: f.candidates, foldView: e.viewGeneration, projection: f.projection}
	}
	return FoldTarget{}
}
func sameFoldSource(a, b FoldTarget) bool {
	return a.buffer != nil && a.buffer == b.buffer && a.version == b.version && a.provider == b.provider && a.generation == b.generation && a.candidates == b.candidates && a.foldView == b.foldView
}
func (e *Editor) bumpFoldProjection() {
	if e.folds != nil {
		e.folds.projection++
	}
}
func (e *Editor) validFoldIdentity(t FoldTarget) bool {
	now := e.foldIdentity()
	return sameFoldSource(t, now) && t.projection == now.projection
}
func (e *Editor) foldFailure(kind, message string) error {
	f := e.folds
	f.ready = false
	f.ranges, f.intervals, f.collapsed = nil, nil, nil
	f.status = message
	err := &foldAnalysisError{message: message}
	if f.reported == nil {
		f.reported = map[string]bool{}
	}
	if !f.reported[kind] {
		f.reported[kind] = true
		e.fail(err)
	}
	e.clampScroll()
	e.services.Invalidate()
	return err
}
func callFoldProvider(provider FoldProvider, snapshot FoldSnapshot) (result FoldResult, err error) {
	defer func() {
		if recover() != nil {
			result = FoldResult{}
			err = errors.New("折叠分析器异常")
		}
	}()
	return provider(snapshot)
}

// RefreshFolds is the only analysis entry point after installation. An error
// always publishes the full source and a visible reason, never partial folds.
func (e *Editor) RefreshFolds() error {
	if err := e.pendingMultiIssue(); err != nil {
		return err
	}
	e.syncFolds()
	f := e.folds
	if f == nil || f.provider == nil {
		return errors.New("此编辑区未启用折叠")
	}
	f.candidates++
	f.ready = false
	f.ranges, f.intervals, f.collapsed = nil, nil, nil
	if f.maxBytes <= 0 || e.Buffer.LenBytes() > f.maxBytes {
		return e.foldFailure("budget", "折叠分析超过字节预算；已显示全文")
	}
	identity := e.foldIdentity()
	snapshot := FoldSnapshot{Source: e.Buffer.Snapshot().String(), Version: e.Buffer.Version(), LineCount: e.Buffer.LineCount()}
	result, err := callFoldProvider(f.provider, snapshot)
	if err := e.pendingMultiIssue(); err != nil {
		return err
	}
	if !e.validFoldIdentity(identity) || e.folds != f {
		return ErrFoldStale
	}
	if err != nil {
		return e.foldFailure("provider", "折叠分析失败；已显示全文："+boundedFoldMessage(err.Error()))
	}
	if result.Unsupported {
		return e.foldFailure("unsupported", "当前文档不支持折叠分析；已显示全文")
	}
	if result.Diagnostics != 0 {
		return e.foldFailure("diagnostics", "折叠分析：文档存在结构诊断；暂时显示全文")
	}
	if !result.Complete {
		return e.foldFailure("incomplete", "折叠分析未完成；已显示全文")
	}
	if len(result.Folds) > MaxFoldCandidates {
		return e.foldFailure("candidates", "折叠候选超过4096个；已显示全文")
	}
	ranges := append([]FoldRange(nil), result.Folds...)
	if err := e.validateFolds(ranges); err != nil {
		return e.foldFailure("invalid", "折叠范围无效；已显示全文")
	}
	f.ranges, f.collapsed, f.ready = ranges, map[FoldRange]bool{}, true
	f.status = fmt.Sprintf("折叠分析完成：%d 个块", len(ranges))
	e.clampScroll()
	e.services.Invalidate()
	return nil
}
func boundedFoldMessage(message string) string {
	if len(message) > 512 {
		return message[:512]
	}
	return message
}
func (e *Editor) validateFolds(ranges []FoldRange) error {
	sort.Slice(ranges, func(i, j int) bool {
		if ranges[i].OpenStart != ranges[j].OpenStart {
			return ranges[i].OpenStart < ranges[j].OpenStart
		}
		return ranges[i].CloseEnd > ranges[j].CloseEnd
	})
	stack := make([]FoldRange, 0)
	seen := map[FoldRange]bool{}
	for _, r := range ranges {
		if r.OpenLine < 0 || r.CloseLine < 0 || r.CloseLine >= e.Buffer.LineCount() || r.CloseLine-r.OpenLine < 2 ||
			r.OpenStart < 0 || r.OpenStart >= r.OpenEnd || r.OpenEnd > r.CloseStart || r.CloseStart >= r.CloseEnd || r.CloseEnd > e.Buffer.LenUTF16() || seen[r] {
			return text.ErrRange
		}
		seen[r] = true
		for _, pair := range [][3]int{{r.OpenStart, r.OpenEnd, r.OpenLine}, {r.CloseStart, r.CloseEnd, r.CloseLine}} {
			if _, err := e.Buffer.SliceUTF16(pair[0], pair[0]); err != nil {
				return err
			}
			if _, err := e.Buffer.SliceUTF16(pair[1], pair[1]); err != nil {
				return err
			}
			line, _, err := e.Buffer.LineColumn(pair[0])
			if err != nil || line != pair[2] {
				return text.ErrRange
			}
			span, err := e.Buffer.LineRange(pair[2])
			if err != nil || pair[0] < span.Start() || pair[1] > span.End() {
				return text.ErrRange
			}
		}
		for len(stack) > 0 && r.OpenStart >= stack[len(stack)-1].CloseEnd {
			stack = stack[:len(stack)-1]
		}
		if len(stack) > 0 {
			parent := stack[len(stack)-1]
			if r.OpenStart < parent.OpenEnd || r.CloseEnd > parent.CloseStart {
				return text.ErrRange
			}
		}
		stack = append(stack, r)
	}
	return nil
}
func (e *Editor) foldCandidateAtLine(line int) (FoldCandidate, bool) {
	if e.folds == nil || !e.folds.ready {
		return FoldCandidate{}, false
	}
	i := sort.Search(len(e.folds.ranges), func(i int) bool { return e.folds.ranges[i].OpenLine >= line })
	if i >= len(e.folds.ranges) || e.folds.ranges[i].OpenLine != line {
		return FoldCandidate{}, false
	}
	r := e.folds.ranges[i]
	target := e.foldIdentity()
	target.span = r
	return FoldCandidate{Range: r, Collapsed: e.folds.collapsed[r], Target: target}, true
}

func (e *Editor) FoldCandidates() []FoldCandidate {
	e.syncFolds()
	f := e.folds
	if f == nil || !f.ready {
		return nil
	}
	result := make([]FoldCandidate, len(f.ranges))
	for i, r := range f.ranges {
		target := e.foldIdentity()
		target.span = r
		result[i] = FoldCandidate{Range: r, Collapsed: f.collapsed[r], Target: target}
	}
	return result
}
func (e *Editor) ensureFolds() error {
	e.syncFolds()
	if e.folds == nil || e.folds.provider == nil {
		return errors.New("此编辑区未启用折叠")
	}
	if !e.folds.ready {
		return e.RefreshFolds()
	}
	return nil
}
func (e *Editor) hiddenOffsets(r FoldRange) (int, int) {
	start, _ := e.Buffer.OffsetOfLine(r.OpenLine + 1)
	end, _ := e.Buffer.OffsetOfLine(r.CloseLine)
	return start, end
}
func intersectsFold(start, end, left, right int) bool {
	if start == end {
		return start >= left && start < right
	}
	return start < right && end > left
}
func (e *Editor) ToggleFold(target FoldTarget) error {
	if err := e.pendingMultiIssue(); err != nil {
		return err
	}
	if !e.validFoldIdentity(target) || e.folds == nil || !e.folds.ready {
		return ErrFoldStale
	}
	f := e.folds
	found := false
	for _, r := range f.ranges {
		if r == target.span {
			found = true
			break
		}
	}
	if !found {
		return ErrFoldStale
	}
	r := target.span
	if !f.collapsed[r] {
		left, right := e.hiddenOffsets(r)
		selection := e.Buffer.Selection()
		if intersectsFold(selection.Start(), selection.End(), left, right) || e.composing && intersectsFold(e.marked.Start, e.marked.End, left, right) {
			err := errors.New("请先将光标、选区或输入法组合移出折叠区域")
			return err
		}
	}
	e.changeFolds(func() {
		if f.collapsed[r] {
			delete(f.collapsed, r)
		} else {
			f.collapsed[r] = true
		}
	})
	return nil
}
func (e *Editor) ToggleFoldAt(line int) error {
	if err := e.pendingMultiIssue(); err != nil {
		return err
	}
	if err := e.ensureFolds(); err != nil {
		return err
	}
	for _, candidate := range e.FoldCandidates() {
		if candidate.Range.OpenLine == line {
			return e.ToggleFold(candidate.Target)
		}
	}
	return errors.New("此行没有可折叠块")
}
func (e *Editor) foldForPosition(offset int, collapsedOnly bool) (FoldTarget, bool) {
	line, _, err := e.Buffer.LineColumn(offset)
	if err != nil {
		return FoldTarget{}, false
	}
	var target FoldTarget
	best, exact := int(^uint(0)>>1), false
	for _, candidate := range e.FoldCandidates() {
		r := candidate.Range
		if (collapsedOnly && !candidate.Collapsed) || line < r.OpenLine || line > r.CloseLine {
			continue
		}
		inside := offset >= r.OpenStart && offset <= r.CloseEnd
		size := r.CloseEnd - r.OpenStart
		if (inside && !exact) || (inside == exact && size < best) {
			target, best, exact = candidate.Target, size, inside
		}
	}
	return target, target.buffer != nil
}
func (e *Editor) FoldAtPosition(offset int) error {
	if err := e.pendingMultiIssue(); err != nil {
		return err
	}
	if err := e.ensureFolds(); err != nil {
		return err
	}
	target, ok := e.foldForPosition(offset, false)
	if !ok {
		return errors.New("当前位置没有可折叠块")
	}
	if e.folds.collapsed[target.span] {
		return nil
	}
	return e.ToggleFold(target)
}
func (e *Editor) UnfoldAtPosition(offset int) error {
	if err := e.pendingMultiIssue(); err != nil {
		return err
	}
	e.syncFolds()
	if e.folds == nil || !e.folds.ready {
		return nil
	}
	target, ok := e.foldForPosition(offset, true)
	if !ok {
		return nil
	}
	return e.ToggleFold(target)
}
func (e *Editor) UnfoldAll() error {
	if err := e.pendingMultiIssue(); err != nil {
		return err
	}
	e.syncFolds()
	if e.folds != nil && len(e.folds.collapsed) > 0 {
		e.changeFolds(func() { e.folds.collapsed = map[FoldRange]bool{} })
	}
	return nil
}

// Preserve the source line and wrapped row at the viewport top when possible.
// Hidden anchors map to their closing line. Wrap counts remain intact.
func (e *Editor) changeFolds(change func()) {
	e.invalidateCacheIfChanged()
	visual := max(0, int((e.ScrollY-12)/e.rowHeight()))
	line, row := e.lineAndRowForVisual(visual)
	fraction := e.ScrollY - float32(visual)*e.rowHeight()
	change()
	e.folds.projection++
	e.rebuildFoldIntervals()
	// Drop shaped layouts for newly hidden lines while retaining their sparse
	// row-count values for Fenwick subtraction. This walks only the 256-entry
	// shape cache, never the potentially much larger rowCounts map.
	for line := range e.cache {
		if e.hiddenLine(line) {
			delete(e.cache, line)
		}
	}
	if visible := e.nextVisible(line, 1); visible != line {
		line, row = visible, 0
	}
	e.ScrollY = float32(e.visualBase(line)+row)*e.rowHeight() + fraction
	e.clampScroll()
	e.services.Invalidate()
}
func (e *Editor) rebuildFoldIntervals() {
	if e.folds == nil {
		return
	}
	e.folds.intervals = sourceFoldIntervals(e.folds.collapsed)
	e.rebuildFoldExtras()
}

// These projection primitives have no Editor, host callback or invalidation.
// Detached navigation and live folding must use the same interval/Fenwick math.
func sourceFoldIntervals(collapsed map[FoldRange]bool) []foldInterval {
	spans := make([]foldInterval, 0, len(collapsed))
	for r := range collapsed {
		spans = append(spans, foldInterval{start: r.OpenLine + 1, end: r.CloseLine})
	}
	sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })
	union := spans[:0]
	for _, s := range spans {
		if len(union) > 0 && s.start <= union[len(union)-1].end {
			union[len(union)-1].end = max(union[len(union)-1].end, s.end)
		} else {
			union = append(union, s)
		}
	}
	return union
}

// rowCounts may contain arbitrarily many visited lines. Only O(F log N)
// endpoint queries are needed; no fold transition walks rowCounts or hidden lines.
func (e *Editor) rebuildFoldExtras() {
	if e.folds != nil {
		updateFoldRowExtras(e.folds.intervals, e.rowTree)
	}
}

func updateFoldRowExtras(intervals []foldInterval, tree map[int]int) {
	hidden, extra := 0, 0
	for i := range intervals {
		s := &intervals[i]
		s.hiddenBefore, s.extraBefore = hidden, extra
		hidden += s.end - s.start
		extra += rowPrefix(tree, s.end) - rowPrefix(tree, s.start)
	}
}
func (e *Editor) foldDeductions(line int) (int, int) {
	return foldRowDeductions(e.folds, e.rowTree, line)
}

func foldRowDeductions(f *foldState, tree map[int]int, line int) (int, int) {
	if f == nil || len(f.intervals) == 0 {
		return 0, 0
	}
	i := sort.Search(len(f.intervals), func(i int) bool { return f.intervals[i].end >= line })
	if i == len(f.intervals) {
		s := f.intervals[i-1]
		return s.hiddenBefore + s.end - s.start, s.extraBefore + rowPrefix(tree, s.end) - rowPrefix(tree, s.start)
	}
	s := f.intervals[i]
	end := max(s.start, min(line, s.end))
	return s.hiddenBefore + end - s.start, s.extraBefore + rowPrefix(tree, end) - rowPrefix(tree, s.start)
}
func (e *Editor) hiddenLineMustReject(offset int) bool {
	line, _, err := e.Buffer.LineColumn(offset)
	return err == nil && e.hiddenLine(line)
}
func (e *Editor) rangeTouchesHidden(start, end int) bool {
	if e.folds == nil {
		return false
	}
	for _, interval := range e.folds.intervals {
		left, _ := e.Buffer.OffsetOfLine(interval.start)
		right, _ := e.Buffer.OffsetOfLine(interval.end)
		if intersectsFold(start, end, left, right) {
			return true
		}
	}
	return false
}
func (e *Editor) hiddenLine(line int) bool {
	e.syncFolds()
	f := e.folds
	if f == nil {
		return false
	}
	i := sort.Search(len(f.intervals), func(i int) bool { return f.intervals[i].end > line })
	return i < len(f.intervals) && f.intervals[i].start <= line
}
func (e *Editor) nextVisible(line, dir int) int {
	e.syncFolds()
	f := e.folds
	if f == nil {
		return line
	}
	i := sort.Search(len(f.intervals), func(i int) bool { return f.intervals[i].end > line })
	if i < len(f.intervals) && f.intervals[i].start <= line {
		if dir < 0 {
			return f.intervals[i].start - 1
		}
		return f.intervals[i].end
	}
	return line
}

func (e *Editor) hasCollapsedFolds() bool {
	e.syncFolds()
	return e.folds != nil && len(e.folds.collapsed) != 0
}

func (e *Editor) admitFoldEdit(r ui.TextInputRange, value string) bool {
	if !e.hasCollapsedFolds() {
		return true
	}
	if err := e.Buffer.CheckReplaceUTF16(r.Start, r.End, value); err != nil {
		if e.composing || e.compositionError != nil {
			e.refuseComposition(err)
		} else {
			e.fail(err)
		}
		return false
	}
	e.revealFolds(r.Start, r.End, true)
	return true
}

// Navigation and admitted edits reveal every ancestor intersecting the source
// range. Editing also includes adjacent lines to expose delimiter merges.
func (e *Editor) revealFolds(start, end int, neighbors bool) {
	e.syncFolds()
	f := e.folds
	if f == nil || len(f.collapsed) == 0 {
		return
	}
	start, end = min(start, end), max(start, end)
	if neighbors {
		first, _, err := e.Buffer.LineColumn(start)
		if err != nil {
			return
		}
		last, _, err := e.Buffer.LineColumn(end)
		if err != nil {
			return
		}
		start, _ = e.Buffer.OffsetOfLine(max(0, first-1))
		if last+2 < e.Buffer.LineCount() {
			end, _ = e.Buffer.OffsetOfLine(last + 2)
		} else {
			end = e.Buffer.LenUTF16()
		}
	}
	var remove []FoldRange
	for r := range f.collapsed {
		left, right := e.hiddenOffsets(r)
		if intersectsFold(start, end, left, right) {
			remove = append(remove, r)
		}
	}
	if len(remove) > 0 {
		e.changeFolds(func() {
			for _, r := range remove {
				delete(f.collapsed, r)
			}
		})
	}
}
func (e *Editor) foldAtPoint(x, y float32) (FoldTarget, bool) {
	e.syncFolds()
	f := e.folds
	if f == nil || !f.ready || !sameFoldSource(f.painted, e.foldIdentity()) || x < 2 || x >= 18 {
		return FoldTarget{}, false
	}
	localY := y - 12 + e.ScrollY
	if localY < 0 || y < 0 || y >= e.height {
		return FoldTarget{}, false
	}
	visual := int(localY / e.rowHeight())
	if visual >= e.totalVisualRows() {
		return FoldTarget{}, false
	}
	line, row := e.lineAndRowForVisual(visual)
	if row != 0 {
		return FoldTarget{}, false
	}
	candidate, ok := e.foldCandidateAtLine(line)
	return candidate.Target, ok
}
func (e *Editor) foldPointer(ev ui.InputEvent) bool {
	if e.folds == nil || e.folds.provider == nil {
		return false
	}
	if ev.Kind == ui.InputPointerDown && ev.Button == 0 {
		if target, ok := e.foldAtPoint(ev.X, ev.Y); ok {
			e.folds.pointer = &target
			return true
		}
	}
	if ev.Kind == ui.InputPointerMove && e.folds.pointer != nil {
		return true
	}
	if ev.Kind == ui.InputPointerUp && e.folds.pointer != nil {
		target := *e.folds.pointer
		e.folds.pointer = nil
		if !ev.Canceled && ev.Button == 0 {
			current, ok := e.foldAtPoint(ev.X, ev.Y)
			if ok && current == target {
				e.fail(e.ToggleFold(target))
			}
		}
		return true
	}
	return false
}
func (e *Editor) foldControls(c *ui.Context) {
	e.syncFolds()
	f := e.folds
	if f == nil || f.provider == nil {
		return
	}
	identity, selection := e.foldIdentity(), e.Buffer.Selection()
	ui.Row(c.Key("fold-controls")).Wrap().Padding(4, 8).Gap(6).Children(func() {
		for action, label := range []string{"折叠当前块", "展开当前块", "展开全部", "重新分析折叠"} {
			button := ui.Button(c, label).KeepFocus().Disabled(e.compositionMulti)
			press := foldControlPress{identity: identity, selection: selection, compositionGen: e.compositionGeneration, action: action}
			button.HandleInput(func(ev ui.InputEvent) bool { return e.foldControlInput(press, ev) }).OnClick(func() {
				if e.rejectPendingMulti() {
					return
				}
				captured := press
				if e.foldControl.action == action && e.foldControl.identity.buffer != nil {
					captured = e.foldControl
					e.foldControl = foldControlPress{}
				}
				if e.runFoldAction(captured) && !e.composing && e.compositionIssue() == nil {
					button.Focus()
				}
			})
		}
		ui.Text(c, f.status).FontSize(11)
	})
}

func (e *Editor) foldControlInput(press foldControlPress, ev ui.InputEvent) bool {
	if e.rejectPendingMulti() {
		return true
	}
	switch ev.Kind {
	case ui.InputPointerDown:
		if ev.Button == 0 {
			e.foldControl = press
		}
	case ui.InputPointerUp:
		if ev.Canceled {
			e.foldControl = foldControlPress{}
		}
	case ui.InputKeyDown:
		if ev.Mods == 0 && (ev.Key == ui.KeyEnter || ev.Key == ui.KeySpace) {
			if !ev.Repeat {
				press.key = ev.Key
				e.foldControl = press
			}
			return true
		}
	case ui.InputKeyUp:
		if ev.Key == ui.KeyEnter || ev.Key == ui.KeySpace {
			captured := e.foldControl
			e.foldControl = foldControlPress{}
			if captured.action == press.action && captured.key == ev.Key {
				e.runFoldAction(captured)
			}
			return true
		}
	}
	return false
}

func (e *Editor) runFoldAction(press foldControlPress) bool {
	if e.rejectPendingMulti() {
		return false
	}
	if press.compositionGen != e.compositionGeneration || !sameFoldSource(press.identity, e.foldIdentity()) || e.Buffer.Selection() != press.selection {
		e.fail(ErrFoldStale)
		return false
	}
	var err error
	switch press.action {
	case 0:
		err = e.FoldAtPosition(press.selection.Head)
	case 1:
		err = e.UnfoldAtPosition(press.selection.Head)
	case 2:
		err = e.UnfoldAll()
	case 3:
		err = e.RefreshFolds()
	}
	var analysis *foldAnalysisError
	if !errors.As(err, &analysis) {
		e.fail(err)
	}
	return err == nil
}
