package graphview

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/search"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/drawio"
	"levnote.local/native/formats/mindmap"
	"levnote.local/native/ui/editor"
)

const fieldSearchMutationWork = 64 << 20

// Coordinates below belong to one decoded field. They never cross the optional
// host contract, and are never interpreted as JSON/XML source positions.
type graphSearchField struct {
	id, source string
	field      nodeField
	edge       bool
}
type graphFieldHit struct {
	field   int
	match   search.Match
	preview string
}
type fieldSearchOwner struct {
	buffer                             *text.Buffer
	version, release, pageEpoch, epoch uint64
	page                               int
	kind                               documentKind
	readOnly                           bool
}
type graphFieldResults struct {
	result           editor.FieldSearchResult
	owner            fieldSearchOwner
	opts             search.Options
	limits           editor.FieldSearchLimits
	pageID, pageName string
	pageCount        int
	fields           []graphSearchField
	hits             []graphFieldHit
}
type graphFieldSearch struct {
	epoch, session                       uint64
	observed                             fieldSearchOwner
	observedOK                           bool
	results                              *graphFieldResults
	index                                int
	open                                 bool
	query, replacement, edgeInput        *editor.Editor
	queryComposing, replacementComposing bool
	queryFailed, replacementFailed       bool
	opts                                 search.Options
	err                                  error
	notice                               string
	persistentNotice                     bool
	press                                fieldSearchPress
	commitReceipt                        *fieldSearchReceipt
	handles                              [fieldSearchActionCount]ui.Handle
}

var _ editor.FieldSearchProvider = (*view)(nil)

func (v *view) fieldSearchOwner() fieldSearchOwner {
	o := fieldSearchOwner{buffer: v.host.Buffer, release: v.releaseEpoch, pageEpoch: v.pageEpoch, page: v.pageIndex, kind: v.kind, readOnly: v.host.ReadOnly}
	if o.buffer != nil {
		o.version = o.buffer.Version()
	}
	return o
}
func (v *view) invalidateFieldSearch() {
	v.search.epoch++
	v.search.results = nil
	v.search.commitReceipt = nil
	v.search.index = -1
}

// Called before sync's fast return as well as every public operation. Once an
// intermediate owner has been observed, returning to its old bytes cannot revive
// a token or an in-flight native press.
func (v *view) syncFieldSearchOwner() {
	o := v.fieldSearchOwner()
	if !v.search.observedOK || o != v.search.observed {
		v.invalidateFieldSearch()
		if v.search.edgeInput != nil {
			v.search.edgeInput.ReleaseView()
			v.search.edgeInput = nil
		}
		v.search.observed, v.search.observedOK = o, true
	}
}
func (v *view) sameFieldSearchOwner(owner fieldSearchOwner) bool {
	o := v.fieldSearchOwner()
	ownerEpoch := owner.epoch
	owner.epoch = 0
	return o == owner && ownerEpoch == v.search.epoch
}
func validFieldSearchLimits(l editor.FieldSearchLimits) bool {
	max := editor.DefaultFieldSearchLimits()
	values := [...]int{l.SourceBytes, l.ScanBytes, l.FieldBytes, l.Fields, l.Matches, l.PreviewBytes, l.QueryBytes, l.ReplacementBytes, l.OutputBytes}
	caps := [...]int{max.SourceBytes, max.ScanBytes, max.FieldBytes, max.Fields, max.Matches, max.PreviewBytes, max.QueryBytes, max.ReplacementBytes, max.OutputBytes}
	for i, value := range values {
		if value <= 0 || value > caps[i] {
			return false
		}
	}
	return true
}
func (v *view) fieldSearchBusy() bool {
	s := &v.search
	if v.fieldSearchGraphDraftBusy() || s.queryComposing || s.replacementComposing || s.queryFailed || s.replacementFailed {
		return true
	}
	for _, input := range []*editor.Editor{s.query, s.replacement} {
		if input != nil {
			if _, marked := input.MarkedRange(); marked || input.Buffer.InGroup() {
				return true
			}
		}
	}
	return false
}
func (v *view) fieldSearchGraphDraftBusy() bool {
	if v.flushing || v.draftError != nil || v.host.Buffer != nil && v.host.Buffer.InGroup() {
		return true
	}
	if d := v.draft; d != nil {
		if _, marked := d.input.MarkedRange(); marked || d.input.Buffer.InGroup() {
			return true
		}
		// A version-proven clean cache needs no materialization (including a
		// large source field). Unknown/changed large drafts fail before String.
		// Inspection never confirms, flushes or rewrites the draft cache.
		if d.pendingKnown && d.pendingVersion == d.input.Buffer.Version() {
			if d.pending {
				return true
			}
		} else if d.input.Buffer.LenBytes() > editor.DefaultFieldSearchLimits().FieldBytes || d.input.Buffer.LenBytes() != len(d.base) || d.input.Buffer.String() != d.base {
			return true
		}
	}
	return v.pendingTagDraft()
}

func fieldSearchMindOptions(maxBytes int) mindmap.ParseOptions {
	return mindmap.ParseOptions{MaxBytes: maxBytes, MaxDepth: 128, MaxNodes: 16384}
}
func fieldSearchDrawOptions(maxBytes int) drawio.ParseOptions {
	return drawio.ParseOptions{MaxBytes: maxBytes, MaxDecompressedBytes: 4 << 20, MaxXMLDepth: 128, MaxElements: 32768, MaxTextBytes: 4 << 20}
}
func fieldSearchPage(doc *drawio.Document, index int) (*drawio.Page, error) {
	if doc == nil || index < 0 || index >= len(doc.Pages) {
		return nil, editor.ErrFieldSearchStale
	}
	return &doc.Pages[index], nil
}
func fieldSearchPlainNode(node drawio.Node) bool {
	return !node.HTML && !node.Image && !node.Stencil && len(node.Unsupported) == 0 && node.Text == node.PlainText
}
func fieldSearchPlainEdge(edge drawio.Edge) bool {
	return !edge.HTML && len(edge.Unsupported) == 0 && edge.Text == edge.PlainText
}

func (v *view) FindFields(opts search.Options, limits editor.FieldSearchLimits) (editor.FieldSearchResult, error) {
	v.syncFieldSearchOwner()
	v.invalidateFieldSearch() // even an unsuccessful new query retires old results
	return v.findFieldsAfterInvalidation(opts, limits)
}

// A committed reveal already retired its old results. A refresh of that same
// request can leave its receipt intact on failure, for owned error publication.
func (v *view) findFieldsAfterInvalidation(opts search.Options, limits editor.FieldSearchLimits) (editor.FieldSearchResult, error) {
	if !validFieldSearchLimits(limits) || len(opts.Query) > limits.QueryBytes {
		return editor.FieldSearchResult{}, editor.ErrFieldSearchBudget
	}
	if v.fieldSearchBusy() {
		return editor.FieldSearchResult{}, editor.ErrFieldSearchDraft
	}
	if v.host.Buffer == nil {
		return editor.FieldSearchResult{}, editor.ErrFieldSearchStale
	}
	if v.host.Buffer.LenBytes() > limits.SourceBytes {
		return editor.FieldSearchResult{}, editor.ErrFieldSearchBudget
	}
	// Compile/validate once even when every decoded field is empty or excluded.
	if _, err := search.FindStringBounded("", opts, 1); err != nil {
		return editor.FieldSearchResult{}, err
	}
	r := &graphFieldResults{owner: v.fieldSearchOwner(), opts: opts, limits: limits}
	r.owner.epoch = v.search.epoch
	raw := v.host.Buffer.String()
	scanned, fields, previewBytes, omitted := 0, 0, 0, 0
	var scanError error
	collect := func(id string, field nodeField, source string, edge bool) bool {
		if fields == limits.Fields || len(source) > limits.ScanBytes-scanned || len(source) > limits.FieldBytes || len(id) > editor.MaxOutlineIDBytes {
			r.result.Truncated = true
			return false
		}
		fields++
		scanned += len(source)
		if opts.Query == "" {
			return true
		}
		remaining := limits.Matches - len(r.hits)
		if remaining == 0 {
			r.result.Truncated = true
			return false
		}
		hits, err := search.FindStringBounded(source, opts, remaining)
		if err != nil && !errors.Is(err, search.ErrTooManyMatches) {
			scanError = err
			return false
		}
		if len(hits) > 0 {
			index := len(r.fields)
			r.fields = append(r.fields, graphSearchField{strings.Clone(id), strings.Clone(source), field, edge})
			for _, hit := range hits {
				preview := fieldSearchPreview(hit.Value, min(96, limits.PreviewBytes-previewBytes))
				previewBytes += len(preview)
				r.result.PreviewTruncated = r.result.PreviewTruncated || len(preview) < len(hit.Value)
				r.hits = append(r.hits, graphFieldHit{index, hit, preview})
			}
		}
		if err != nil {
			r.result.Truncated = true
			return false
		}
		return true
	}
	switch v.kind {
	case kindMindmap:
		doc, _, err := mindmap.Parse([]byte(raw), fieldSearchMindOptions(limits.SourceBytes))
		if err != nil {
			return editor.FieldSearchResult{}, err
		}
		r.result.Scope = "思维导图：先序节点，每节点纯文本主题后备注；不搜索链接、标签、JSON语法及未知或非文本字段。只读隐藏目标不能展开；引擎只读预览尚不支持字段查找。"
		var walk func(*mindmap.Node) bool
		walk = func(node *mindmap.Node) bool {
			if node == nil {
				return true
			}
			if !collect(node.ID, fieldTopic, node.Topic, false) {
				return false
			}
			if node.Note != nil && !collect(node.ID, fieldNote, *node.Note, false) {
				return false
			}
			for _, child := range node.Children {
				if !walk(child) {
					return false
				}
			}
			return true
		}
		walk(doc.Root)
	case kindDrawio:
		doc, _, err := drawio.Parse([]byte(raw), fieldSearchDrawOptions(limits.SourceBytes))
		if err != nil {
			return editor.FieldSearchResult{}, err
		}
		page, err := fieldSearchPage(doc, v.pageIndex)
		if err != nil {
			return editor.FieldSearchResult{}, err
		}
		r.pageID, r.pageName, r.pageCount = page.ID, page.Name, len(doc.Pages)
		r.result.Scope = "draw.io：仅当前页已解码的纯文本节点及连接线标签；不搜索XML、其它页、富HTML、图片、stencil和不透明字段。"
		if page.ReadOnly {
			r.result.Scope += " 当前页格式只读，仅可查找和定位，拒绝替换。"
		}
		stopped := false
		for _, node := range page.Model.Nodes {
			if !fieldSearchPlainNode(node) {
				omitted++
				continue
			}
			if !collect(node.ID, fieldDrawioLabel, node.PlainText, false) {
				stopped = true
				break
			}
		}
		if !stopped {
			for _, edge := range page.Model.Edges {
				if !fieldSearchPlainEdge(edge) {
					omitted++
					continue
				}
				if !collect(edge.ID, fieldDrawioLabel, edge.PlainText, true) {
					break
				}
			}
		}
		if omitted > 0 {
			r.result.Scope += fmt.Sprintf(" 当前页已排除%d个非纯文本或不支持的标签。", omitted)
		}
	default:
		return editor.FieldSearchResult{}, editor.ErrFieldSearchStale
	}
	if scanError != nil {
		return editor.FieldSearchResult{}, scanError
	}
	if !v.sameFieldSearchOwner(r.owner) {
		return editor.FieldSearchResult{}, editor.ErrFieldSearchStale
	}
	r.result.Token = editor.NewFieldSearchToken()
	r.result.Count = len(r.hits)
	v.search.commitReceipt = nil
	v.search.results = r
	v.search.index = -1
	return r.result, nil
}

func fieldSearchPreview(value string, limit int) string {
	end := min(len(value), max(limit, 0))
	for end > 0 && end < len(value) && !utf8.RuneStart(value[end]) {
		end--
	}
	return strings.Clone(value[:end])
}
func (v *view) fieldSearchResults(token editor.FieldSearchToken, index int, all bool) (*graphFieldResults, error) {
	v.syncFieldSearchOwner()
	r := v.search.results
	if r == nil || r.result.Token != token || !v.sameFieldSearchOwner(r.owner) || !all && (index < 0 || index >= len(r.hits)) {
		return nil, editor.ErrFieldSearchStale
	}
	if v.fieldSearchBusy() {
		return nil, editor.ErrFieldSearchDraft
	}
	if v.host.Buffer.LenBytes() > r.limits.SourceBytes {
		return nil, editor.ErrFieldSearchBudget
	}
	return r, nil
}
func (r *graphFieldResults) samePage(doc *drawio.Document) bool {
	page, err := fieldSearchPage(doc, r.owner.page)
	return err == nil && page.ID == r.pageID && page.Name == r.pageName && len(doc.Pages) == r.pageCount
}
func searchFieldValue(mind *mindmap.Document, diagram *drawio.Document, pageIndex int, field graphSearchField) (string, error) {
	if mind != nil {
		node, ok := mind.FindNode(field.id)
		if !ok {
			return "", editor.ErrFieldSearchStale
		}
		value, present := mindmapField(node, field.field)
		if !present {
			return "", editor.ErrFieldSearchStale
		}
		return value, nil
	}
	page, err := fieldSearchPage(diagram, pageIndex)
	if err != nil {
		return "", err
	}
	if field.edge {
		for _, edge := range page.Model.Edges {
			if edge.ID == field.id && fieldSearchPlainEdge(edge) {
				return edge.PlainText, nil
			}
		}
	} else {
		for _, node := range page.Model.Nodes {
			if node.ID == field.id && fieldSearchPlainNode(node) {
				return node.PlainText, nil
			}
		}
	}
	return "", editor.ErrFieldSearchStale
}

func (v *view) RevealFieldMatch(token editor.FieldSearchToken, index int) error {
	r, err := v.fieldSearchResults(token, index, false)
	if err != nil {
		return err
	}
	hit := r.hits[index]
	field := r.fields[hit.field]
	raw := v.host.Buffer.String()
	var mind *mindmap.Document
	var diagram *drawio.Document
	if v.kind == kindMindmap {
		mind, _, err = mindmap.Parse([]byte(raw), fieldSearchMindOptions(r.limits.OutputBytes))
	} else {
		diagram, _, err = drawio.Parse([]byte(raw), fieldSearchDrawOptions(r.limits.OutputBytes))
	}
	if err != nil {
		return err
	}
	if diagram != nil && !r.samePage(diagram) {
		return editor.ErrFieldSearchStale
	}
	value, err := searchFieldValue(mind, diagram, v.pageIndex, field)
	if err != nil || value != field.source {
		return editor.ErrFieldSearchStale
	}
	var expansion *fieldSearchReceipt
	if mind != nil {
		ancestors, found := collapsedAncestors(mind.Root, field.id)
		if !found {
			return editor.ErrFieldSearchStale
		}
		if len(ancestors) > 0 && v.host.ReadOnly {
			return editor.ErrOutlineReadOnlyHidden
		}
		if len(ancestors) > 0 {
			for _, id := range ancestors {
				if err = mind.SetCollapsed(id, false); err != nil {
					return err
				}
			}
			data, e := mind.Serialize(mindmap.SerializeOptions{})
			if e != nil {
				return e
			}
			committed, err := v.commitFieldSearchCandidate(r, string(data), "展开")
			if err != nil {
				return err
			}
			if committed.Changed {
				expansion = v.search.commitReceipt // commit already synced and verified the receipt
			}
		}
	}
	// Only after all fallible hidden/readonly checks may navigation touch the
	// parsed view, selected field or canvas. There is deliberately no Flush.
	if expansion == nil {
		v.ensureSynced(nil)
		if !v.sameFieldSearchOwner(r.owner) || v.fieldSearchBusy() {
			return editor.ErrFieldSearchStale
		}
	}
	v.search.edgeInput = nil
	v.invalidateTagActions()
	v.clearDraft()
	if mind != nil {
		v.selectedID, v.field = field.id, field.field
		v.refreshLabelDraft()
		v.refreshTagDraft()
		v.pendingOutlineCenter = field.id
		v.fitted = true
		v.centerOutlineTarget()
	} else {
		v.setDrawioSingle(field.id)
		v.refreshLabelDraft()
		if field.edge {
			input := editor.New(text.New(field.source))
			input.Label = "已定位连接线标签（只读显示）"
			input.ReadOnly = true
			input.LineNumbers = false
			input.MinHeight = 42
			input.FontSize = 12
			input.ShapeLimit = 512
			v.search.edgeInput = input
		} else {
			v.centerFieldSearchNode(field.id)
		}
	}
	input := v.search.edgeInput
	if input == nil && v.draft != nil {
		input = v.draft.input
	}
	if input == nil {
		return editor.ErrFieldSearchStale
	}
	input.ReadOnly = v.host.ReadOnly || field.edge || diagram != nil && diagram.Pages[v.pageIndex].ReadOnly
	input.Reveal(hit.match.Start, hit.match.End)
	if expansion != nil {
		if !v.sameFieldSearchReceipt(expansion) {
			return editor.ErrFieldSearchStale
		}
		// Refresh only this committed reveal, not a newer request. Failure
		// leaves no results or selected index and keeps the receipt available
		// for the initiating toolbar action. Success publishes a fresh token.
		refreshed, err := v.findFieldsAfterInvalidation(r.opts, r.limits)
		if err != nil || index >= refreshed.Count {
			return err
		}
	}
	v.search.index = index
	return nil
}
func (v *view) centerFieldSearchNode(id string) {
	page := v.currentPage()
	if page == nil || v.canvasRect.W <= 0 || v.canvasRect.H <= 0 {
		return
	}
	for _, node := range page.Model.Nodes {
		if node.ID == id {
			v.panX = v.canvasRect.W/2 - float32(node.X+node.Width/2)*v.zoom
			v.panY = v.canvasRect.H/2 - float32(node.Y+node.Height/2)*v.zoom
			v.fitted = true
			return
		}
	}
}

type fieldSearchPatch struct {
	field graphSearchField
	value string
}

// fieldSearchSpend shares the replacement ledger with format preflight. It
// admits each bounded pass before entering format code; zero-output work counts.
func fieldSearchSpend(work *int, costs ...int) error {
	for _, cost := range costs {
		if cost < 0 || *work < cost {
			*work = 0
			return errors.Join(editor.ErrFieldSearchBudget, search.ErrReplacementWork)
		}
		*work -= cost
	}
	return nil
}

// JSON canonicalization adds indentation/default members as well as escaping.
// At depth <=128, 1024 bytes per byte outside JSON strings conservatively admits
// that structural growth; string payload and XML need at most six escaped bytes.
// The scan itself and all later passes debit the same ledger, before allocation.
func fieldSearchSerializationBound(raw string, kind documentKind, work *int) (int, error) {
	if err := fieldSearchSpend(work, 1, len(raw)); err != nil {
		return 0, err
	}
	upper := 0
	quoted, escaped := false, false
	for i := 0; i < len(raw); i++ {
		cost := 6
		if kind == kindMindmap {
			if !quoted && raw[i] != '"' {
				cost += 1024
			}
			if escaped {
				escaped = false
			} else if quoted && raw[i] == '\\' {
				escaped = true
			} else if raw[i] == '"' {
				quoted = !quoted
			}
		}
		if cost > *work-upper {
			*work = 0
			return 0, errors.Join(editor.ErrFieldSearchBudget, search.ErrReplacementWork)
		}
		upper += cost
	}
	return upper, nil
}
func fieldSearchReplacementError(err error) error {
	if errors.Is(err, search.ErrReplacementWork) || errors.Is(err, search.ErrReplacementLimit) {
		return errors.Join(editor.ErrFieldSearchBudget, err)
	}
	return err
}

func (v *view) ReplaceFieldMatch(token editor.FieldSearchToken, index int, replacement string, all bool) (editor.FieldReplaceResult, error) {
	work := fieldSearchMutationWork
	return v.replaceFieldMatchWithWork(token, index, replacement, all, &work)
}

// The private ledger seam permits tiny deterministic boundary tests. Public
// operations always start with the same finite ceiling, never unlimited work.
func (v *view) replaceFieldMatchWithWork(token editor.FieldSearchToken, index int, replacement string, all bool, work *int) (editor.FieldReplaceResult, error) {
	if work == nil {
		remaining := fieldSearchMutationWork
		work = &remaining
	}
	r, err := v.fieldSearchResults(token, index, all)
	if err != nil {
		return editor.FieldReplaceResult{}, err
	}
	if all && r.result.Truncated {
		return editor.FieldReplaceResult{}, editor.ErrFieldSearchPartial
	}
	if v.host.ReadOnly {
		return editor.FieldReplaceResult{}, errors.New("当前图视图只读，不能替换字段")
	}
	if len(replacement) > r.limits.ReplacementBytes {
		return editor.FieldReplaceResult{}, editor.ErrFieldSearchBudget
	}
	template, err := search.CompileReplacementTemplate(r.opts, replacement, work)
	if err != nil {
		return editor.FieldReplaceResult{}, fieldSearchReplacementError(err)
	}
	if len(r.hits) == 0 {
		return editor.FieldReplaceResult{Undoable: true}, nil
	}
	// Source materialization, parse input and parse scan; template, binding,
	// per-hit expansion and every subsequent candidate copy use THIS ledger.
	size := v.host.Buffer.LenBytes()
	if err = fieldSearchSpend(work, 1, size, size, size, len(r.fields), len(r.hits)); err != nil {
		return editor.FieldReplaceResult{}, err
	}
	raw := v.host.Buffer.String()
	var mind *mindmap.Document
	var diagram *drawio.Document
	if v.kind == kindMindmap {
		mind, _, err = mindmap.Parse([]byte(raw), fieldSearchMindOptions(r.limits.OutputBytes))
	} else {
		diagram, _, err = drawio.Parse([]byte(raw), fieldSearchDrawOptions(r.limits.OutputBytes))
		if err == nil && !r.samePage(diagram) {
			err = editor.ErrFieldSearchStale
		}
		if err == nil && diagram.Pages[v.pageIndex].ReadOnly {
			err = &drawio.ReadOnlyError{Page: v.pageIndex, Reason: diagram.Pages[v.pageIndex].ReadOnlyReason}
		}
	}
	if err != nil {
		return editor.FieldReplaceResult{}, err
	}
	groups := make([][]search.Match, len(r.fields))
	count := 0
	for i, hit := range r.hits {
		if all || i == index {
			groups[hit.field] = append(groups[hit.field], hit.match)
			count++
		}
	}
	patches := make([]fieldSearchPatch, 0, len(groups))
	remaining := r.limits.OutputBytes
	for i, hits := range groups {
		if len(hits) == 0 {
			continue
		}
		field := r.fields[i]
		if err = fieldSearchSpend(work, 1, len(raw), len(field.source)); err != nil {
			return editor.FieldReplaceResult{}, err
		}
		value, e := searchFieldValue(mind, diagram, v.pageIndex, field)
		if e != nil || value != field.source {
			return editor.FieldReplaceResult{}, editor.ErrFieldSearchStale
		}
		bound, e := template.Bind(value, work)
		if e != nil {
			return editor.FieldReplaceResult{}, fieldSearchReplacementError(e)
		}
		next := value
		for j := len(hits) - 1; j >= 0; j-- {
			expansion, e := bound.Expand(hits[j], min(remaining, r.limits.FieldBytes), work)
			if e != nil {
				return editor.FieldReplaceResult{}, fieldSearchReplacementError(e)
			}
			remaining -= len(expansion.Text)
			next, e = search.StitchReplacement(next, expansion, r.limits.FieldBytes, work)
			if e != nil {
				return editor.FieldReplaceResult{}, fieldSearchReplacementError(e)
			}
		}
		if next != value {
			patches = append(patches, fieldSearchPatch{field, next})
		}
	}
	if len(patches) == 0 {
		return editor.FieldReplaceResult{Count: count, Undoable: true}, nil
	}
	// Charge candidate format updates and serialization before either runs.
	upper, err := fieldSearchSerializationBound(raw, v.kind, work)
	if err != nil {
		return editor.FieldReplaceResult{}, err
	}
	for _, patch := range patches {
		if upper > *work || len(patch.value) > (*work-upper)/6 {
			*work = 0
			return editor.FieldReplaceResult{}, errors.Join(editor.ErrFieldSearchBudget, search.ErrReplacementWork)
		}
		upper += 6 * len(patch.value)
	}
	for _, patch := range patches {
		if err = fieldSearchSpend(work, 1, upper); err != nil {
			return editor.FieldReplaceResult{}, err
		}
		if mind != nil {
			err = applyMindmapField(mind, patch.field.id, patch.field.field, patch.value)
		} else {
			err = diagram.UpdateText(v.pageIndex, patch.field.id, patch.value)
		}
		if err != nil {
			return editor.FieldReplaceResult{}, err
		}
	}
	if err = fieldSearchSpend(work, upper); err != nil {
		return editor.FieldReplaceResult{}, err
	}
	var serialized []byte
	if mind != nil {
		serialized, err = mind.Serialize(mindmap.SerializeOptions{})
	} else {
		serialized, err = diagram.Serialize()
	}
	if err != nil {
		return editor.FieldReplaceResult{}, err
	}
	if err = fieldSearchSpend(work, len(serialized)); err != nil {
		return editor.FieldReplaceResult{}, err
	}
	result, err := v.commitFieldSearchCandidateWithWork(r, string(serialized), "替换", work)
	if result.Changed {
		result.Count = count
	}
	return result, err
}
func fieldSearchByteOffset(value string, offset int) (int, error) {
	units := 0
	for i, r := range value {
		if units == offset {
			return i, nil
		}
		if r > 0xffff {
			units += 2
		} else {
			units++
		}
		if units > offset {
			return 0, text.ErrSurrogate
		}
	}
	if units == offset {
		return len(value), nil
	}
	return 0, text.ErrRange
}

// A receipt freezes the postcommit publication owner before Changed reenters.
// The retired token/query identify the operation; current results must stay nil.
// A new Find (even the same query), input session, page or owner retires it.
type fieldSearchReceipt struct {
	owner, previousOwner             fieldSearchOwner
	selectionGeneration              uint64
	token                            editor.FieldSearchToken
	queryOptions, inputOptions       search.Options
	sourceResults                    *graphFieldResults
	query, replacement               *editor.Editor
	queryBuffer, replacementBuffer   *text.Buffer
	queryVersion, replacementVersion uint64
	pageID, pageName                 string
	pageCount                        int
	open                             bool
	session                          uint64
	publicationChanged               bool
}

func (v *view) captureFieldSearchReceipt(r *graphFieldResults) *fieldSearchReceipt {
	s := &v.search
	receipt := &fieldSearchReceipt{owner: v.fieldSearchOwner(), previousOwner: r.owner, token: r.result.Token, queryOptions: r.opts, inputOptions: s.opts, sourceResults: r, query: s.query, replacement: s.replacement, pageID: r.pageID, pageName: r.pageName, pageCount: r.pageCount, open: s.open, session: s.session}
	receipt.owner.epoch = s.epoch
	receipt.selectionGeneration = receipt.owner.buffer.SelectionGeneration()
	if s.query != nil {
		receipt.queryBuffer, receipt.queryVersion = s.query.Buffer, s.query.Buffer.Version()
	}
	if s.replacement != nil {
		receipt.replacementBuffer, receipt.replacementVersion = s.replacement.Buffer, s.replacement.Buffer.Version()
	}
	return receipt
}
func (v *view) sameFieldSearchReceipt(receipt *fieldSearchReceipt) bool {
	s := &v.search
	if receipt == nil || receipt.publicationChanged || s.session != receipt.session || s.commitReceipt != receipt || receipt.sourceResults == nil || receipt.sourceResults.opts != receipt.queryOptions || receipt.sourceResults.result.Token != receipt.token || receipt.sourceResults.owner != receipt.previousOwner || s.results != nil || !v.sameFieldSearchOwner(receipt.owner) || receipt.owner.buffer.SelectionGeneration() != receipt.selectionGeneration || s.opts != receipt.inputOptions || s.open != receipt.open || s.query != receipt.query || s.replacement != receipt.replacement {
		return false
	}
	if s.query != nil && (s.query.Buffer != receipt.queryBuffer || s.query.Buffer.Version() != receipt.queryVersion) || s.replacement != nil && (s.replacement.Buffer != receipt.replacementBuffer || s.replacement.Buffer.Version() != receipt.replacementVersion) {
		return false
	}
	if v.kind == kindDrawio && v.diagram != nil && v.syncedBuffer == receipt.owner.buffer && v.lastSynced == receipt.owner.version {
		page, err := fieldSearchPage(v.diagram, receipt.owner.page)
		if err != nil || page.ID != receipt.pageID || page.Name != receipt.pageName || len(v.diagram.Pages) != receipt.pageCount {
			return false
		}
	}
	// The commit admitted only idle input. A callback can start a group or
	// an empty/same-text preedit without changing any input buffer version.
	// That input now owns publication; observe it without settling it.
	return !v.fieldSearchBusy()
}

// Candidate serialization, growth, coordinates and source ownership are all
// checked before BeginGroup. Only one minimal UTF-8-safe delta reaches authority.
func (v *view) commitFieldSearchCandidate(r *graphFieldResults, after, action string) (editor.FieldReplaceResult, error) {
	work := fieldSearchMutationWork
	return v.commitFieldSearchCandidateWithWork(r, after, action, &work)
}
func (v *view) commitFieldSearchCandidateWithWork(r *graphFieldResults, after, action string, work *int) (editor.FieldReplaceResult, error) {
	if len(after) > r.limits.OutputBytes {
		return editor.FieldReplaceResult{}, editor.ErrFieldSearchBudget
	}
	if !v.sameFieldSearchOwner(r.owner) {
		return editor.FieldReplaceResult{}, editor.ErrFieldSearchStale
	}
	if v.host.ReadOnly || v.fieldSearchBusy() {
		return editor.FieldReplaceResult{}, editor.ErrFieldSearchDraft
	}
	buffer := r.owner.buffer
	if err := fieldSearchSpend(work, 1, buffer.LenBytes(), len(after)); err != nil {
		return editor.FieldReplaceResult{}, err
	}
	before := buffer.String()
	if before == after {
		return editor.FieldReplaceResult{Undoable: true}, nil
	}
	// Candidate parse/copy, prefix+suffix comparisons, UTF-16 scans, buffer
	// validation/write and probe construction all consume the original ledger.
	for i := 0; i < 8; i++ {
		if err := fieldSearchSpend(work, len(before), len(after)); err != nil {
			return editor.FieldReplaceResult{}, err
		}
	}
	if v.kind == kindMindmap {
		if _, _, err := mindmap.Parse([]byte(after), fieldSearchMindOptions(r.limits.OutputBytes)); err != nil {
			return editor.FieldReplaceResult{}, err
		}
	} else {
		if _, _, err := drawio.Parse([]byte(after), fieldSearchDrawOptions(r.limits.OutputBytes)); err != nil {
			return editor.FieldReplaceResult{}, err
		}
	}
	prefix := 0
	for prefix < len(before) && prefix < len(after) && before[prefix] == after[prefix] {
		prefix++
	}
	for prefix > 0 && (prefix < len(before) && !utf8.RuneStart(before[prefix]) || prefix < len(after) && !utf8.RuneStart(after[prefix])) {
		prefix--
	}
	suffix := 0
	for suffix < len(before)-prefix && suffix < len(after)-prefix && before[len(before)-suffix-1] == after[len(after)-suffix-1] {
		suffix++
	}
	for suffix > 0 && (len(before)-suffix < len(before) && !utf8.RuneStart(before[len(before)-suffix]) || len(after)-suffix < len(after) && !utf8.RuneStart(after[len(after)-suffix])) {
		suffix--
	}
	start, end := text.UTF16Len(before[:prefix]), text.UTF16Len(before[:len(before)-suffix])
	next := after[prefix : len(after)-suffix]
	if err := buffer.CheckReplaceUTF16(start, end, next); err != nil {
		return editor.FieldReplaceResult{}, err
	}
	selection := buffer.SelectionSet()
	for range selection.Ranges {
		// Each endpoint's SliceUTF16 scans start and end (4 passes),
		// probe normalization scans both endpoints (2), and the authoritative
		// SetSelectionSet normalizes both again (2). Charge every original
		// range before any boundary clamp or merge can reduce its count.
		if err := fieldSearchSpend(work, 1, 8*len(after)); err != nil {
			return editor.FieldReplaceResult{}, err
		}
	}
	probe := text.New(after)
	selectionClamped := false
	boundary := func(offset int) int {
		offset = max(0, min(offset, probe.LenUTF16()))
		if _, e := probe.SliceUTF16(offset, offset); errors.Is(e, text.ErrSurrogate) {
			offset--
		}
		return offset
	}
	for i, span := range selection.Ranges {
		selection.Ranges[i] = text.Range{Anchor: boundary(span.Anchor), Head: boundary(span.Head)}
		selectionClamped = selectionClamped || selection.Ranges[i] != span
	}
	normalized, err := probe.NormalizeSelectionSet(selection)
	if err != nil {
		return editor.FieldReplaceResult{}, err
	}
	selectionClamped = selectionClamped || normalized.Primary != selection.Primary || len(normalized.Ranges) != len(selection.Ranges)
	if !v.sameFieldSearchOwner(r.owner) || v.fieldSearchBusy() {
		return editor.FieldReplaceResult{}, editor.ErrFieldSearchStale
	}
	if err = buffer.BeginGroup(); err != nil {
		return editor.FieldReplaceResult{}, err
	}
	if err = buffer.ReplaceUTF16(start, end, next); err != nil {
		buffer.CancelGroup()
		return editor.FieldReplaceResult{}, err
	}
	if err = buffer.SetSelectionSet(normalized); err != nil {
		buffer.CancelGroup()
		return editor.FieldReplaceResult{}, err
	}
	if !buffer.EndGroup() {
		return editor.FieldReplaceResult{}, errors.New("字段操作没有产生事务")
	}
	result := editor.FieldReplaceResult{Changed: true, Undoable: buffer.CanUndo(), SourceSelectionClamped: selectionClamped}
	v.invalidateFieldSearch()
	v.search.observed, v.search.observedOK = v.fieldSearchOwner(), true
	v.lastSynced = ^uint64(0)
	// Retire only the old preview, before any callback can install a new one.
	if v.search.edgeInput != nil {
		v.search.edgeInput.ReleaseView()
		v.search.edgeInput = nil
	}
	receipt := v.captureFieldSearchReceipt(r)
	v.search.commitReceipt = receipt
	v.search.persistentNotice = !result.Undoable
	if !result.Undoable {
		v.search.notice = "此次" + action + "不可撤销：当前历史预算未保留本次事务。"
	} else {
		v.search.notice = ""
	}
	publication := v.fieldSearchPublication()
	if v.host.Changed != nil {
		v.host.Changed()
	}
	receipt.publicationChanged = v.fieldSearchPublication() != publication
	// A stale receipt reports the committed fact without publishing into the
	// new query, clearing its preview, or changing the retained transaction.
	if receipt.owner.version != r.owner.version+1 || !v.sameFieldSearchReceipt(receipt) {
		return result, editor.ErrFieldSearchStale
	}
	v.ensureSynced(nil)
	if !v.sameFieldSearchReceipt(receipt) {
		return result, editor.ErrFieldSearchStale
	}
	result.Undoable = buffer.CanUndo()
	v.search.persistentNotice = !result.Undoable
	if !result.Undoable {
		v.search.notice = "此次" + action + "不可撤销：当前历史预算未保留本次事务。"
	}
	if result.SourceSelectionClamped {
		v.search.notice += " 源码选区已夹取到有效字符边界。"
	}
	return result, nil
}
