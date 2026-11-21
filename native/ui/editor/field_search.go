package editor

import (
	"errors"

	"levnote.local/native/core/search"
)

var (
	ErrFieldSearchStale   = errors.New("字段查找的来源、页面或条件已变化，请重新查找")
	ErrFieldSearchBudget  = errors.New("字段查找超过处理预算，请缩小范围或使用源码查找")
	ErrFieldSearchDraft   = errors.New("请先显式完成或放弃节点草稿及输入法组合，再进行字段查找操作")
	ErrFieldSearchPartial = errors.New("字段查找结果已截断，不能将部分结果作为全部替换；请缩小查找范围")
)

// FieldSearchLimits are positive byte/count ceilings, not hints. Implementations
// reject invalid/unsupported budgets before expanding the source. Hosts should
// start with DefaultFieldSearchLimits and may lower individual ceilings.
// PreviewBytes bounds all retained preview text; preview clipping is separate
// from an incomplete scan. OutputBytes bounds the serialized candidate.
type FieldSearchLimits struct {
	SourceBytes, ScanBytes, FieldBytes        int
	Fields, Matches, PreviewBytes             int
	QueryBytes, ReplacementBytes, OutputBytes int
}

func DefaultFieldSearchLimits() FieldSearchLimits {
	return FieldSearchLimits{
		SourceBytes: 4 << 20, ScanBytes: 2 << 20, FieldBytes: 64 << 10,
		Fields: 4096, Matches: 2048, PreviewBytes: 16 << 10,
		QueryBytes: 1024, ReplacementBytes: 64 << 10, OutputBytes: 8 << 20,
	}
}

// FieldSearchToken is opaque, belongs to one mounted provider, and must only be
// returned to that same provider. It never contains a document offset or node
// identifier. Re-search, changed options, reconfiguration, release, buffer and
// page transitions invalidate earlier tokens, including A -> B -> A transitions.
type FieldSearchToken struct{ key *fieldSearchKey }
type fieldSearchKey struct{ allocated byte }

// NewFieldSearchToken creates a unique comparable identity for a provider's
// bounded result. Its private allocation prevents cross-provider collisions.
func NewFieldSearchToken() FieldSearchToken {
	return FieldSearchToken{key: &fieldSearchKey{allocated: 1}}
}

type FieldSearchResult struct {
	Token                       FieldSearchToken
	Count                       int
	Truncated, PreviewTruncated bool
	Scope                       string
}

type FieldReplaceResult struct {
	Count             int
	Changed, Undoable bool
	// SourceSelectionClamped discloses numerical UTF-16 endpoints that no
	// longer exist after format serialization; Undo restores the exact before.
	SourceSelectionClamped bool
}

// FieldSearchProvider is optional on a mounted View. Hosts know neither graph
// node types nor field coordinates. FindFields is a read-only inspection: it
// never Flushes/Unmarks a draft, moves selection/canvas, or changes history/dirty.
// Each result is bounded and tied to the exact owner/query/options; partial
// scans may be navigated/replaced singly but never used for replace-all.
// Reveal uses a zero-based result index. Expanding persisted folded ancestors
// is an undoable edit and invalidates the old token; readonly hidden targets
// fail without changing the selection or canvas. No navigation activates links.
// Replace uses index for one match or all=true for the complete result set.
// Preflight failures preserve source, selection, history/redo and drafts. A
// successful write may outlive a synchronous host callback: Changed/Count then
// describe that write even if a returned stale error rejects further UI work.
// Undoable is false when a custom history budget did not retain this operation;
// implementations must keep that limitation visible. Every write invalidates
// the token. Scene/worker readonly views do not yet provide this capability.
type FieldSearchProvider interface {
	FindFields(search.Options, FieldSearchLimits) (FieldSearchResult, error)
	RevealFieldMatch(FieldSearchToken, int) error
	ReplaceFieldMatch(token FieldSearchToken, index int, replacement string, all bool) (FieldReplaceResult, error)
}
