package editor

import (
	"errors"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

// View is the only interface the workspace uses to host a format's native
// presentation. It never inspects another editor's widget tree or IDs.
type View interface {
	Draw(*ui.Context)
	Release()
	Warnings() []string
	Export(format string) ([]byte, error)
}

// Flusher lets a view synchronize pending edits before the host saves or
// replaces it. A failed flush must leave the view, document and selection alive.
type Flusher interface{ Flush() error }

// Composer lets the host defer actions that would move focus while native
// input is composing text in a rich view.
type Composer interface{ CompositionActive() bool }

// HostFindFocusGuard is an optional, UI-thread-only View admission check for
// host Find focus. It must be a pure, repeatable observation: it must not flush
// or settle input, move/cancel focus, mutate drafts, or establish source target
// or route identity. Implementations must refuse before host Find changes focus
// or panels when an independent draft is confirmed but uncommitted, or its
// readiness cannot be proved. Preserve the actual owned input/draft error when
// one exists; use ErrHostFindFocusDraft only for an independent or uncertain
// draft, not as a source-conflict label. Clean readonly idle input may pass.
type HostFindFocusGuard interface{ HostFindFocusError() error }

// ErrHostFindFocusDraft reports an unsettled or not-provably-clean independent
// draft. It deliberately does not describe a source conflict or input owner.
var ErrHostFindFocusDraft = errors.New("请先完成或放弃当前未写回草稿，再打开查找")

func FlushView(view View) error {
	if flusher, ok := view.(Flusher); ok {
		return flusher.Flush()
	}
	return nil
}

// OutlineItem separates navigation identity from the current link anchor.
// Level is one-based. Offset is the UTF-16 heading syntax start, or -1 when
// the format cannot expose a truthful source position. Text is bounded.
type OutlineItem struct {
	ID, AnchorID, Text string
	Level, Offset      int
}

const (
	MaxOutlineItems      = 2048
	MaxOutlineLabelBytes = 512
	MaxOutlineIDBytes    = 512
)

var (
	ErrOutlineStale          = errors.New("大纲目标已变化，请重新选择")
	ErrOutlineUnsupported    = errors.New("此视图不支持大纲定位")
	ErrOutlineReadOnlyHidden = errors.New("只读文档中的目标已折叠，无法展开定位")
)

type OutlineSnapshot struct {
	Version   uint64
	Items     []OutlineItem
	Truncated bool
}

// OutlineSource inspects a buffer without mounting a view, flushing a draft,
// changing selection or constructing graph layout. Snapshots own their bounded
// item slice; implementations cache by buffer identity and version. Resolve
// maps a captured identity through edits and returns the current identity, or
// fails visibly. A current link slug is resolved separately from item identity.
type OutlineSource interface {
	OutlineSnapshot() (OutlineSnapshot, error)
	ResolveOutlineItem(id string) (OutlineItem, error)
	ResolveOutlineAnchor(anchor string) (OutlineItem, error)
	Release()
}

// OutlineNavigator is optional on a mounted View. It defensively flushes and
// resolves before changing viewport, selection or undoable folded metadata.
type OutlineNavigator interface{ NavigateOutlineItem(id string) error }
type CurrentOutlineProvider interface{ CurrentOutlineItem() (id string, ok bool) }

// OutlineProvider is the legacy mounted-view contract. New hosts should use
// Adapter.Outline and OutlineNavigator for fallible, source-mode-safe access.
type OutlineProvider interface {
	Outline() []OutlineItem
	GoToOutlineItem(id string) bool
}

type Host struct {
	Buffer        *text.Buffer
	ReadOnly      bool
	Path          string
	Changed       func()
	Report        func(error)
	OpenReference func(string)
	// RecoverDraft opens unsynchronized text as a new unsaved document. The
	// originating view may clear its draft only after this succeeds.
	RecoverDraft func(title, content string) error
	// DraftRecoveryChanged asks the host to schedule session capture for the
	// current live owner. Call on the UI thread when a confirmed independent
	// draft changes or is cleared. The callback does not commit the draft.
	DraftRecoveryChanged func(owner View)
	// SceneEngine is copied from the enabled adapter. BeginScene and its
	// idempotent cancel handle are UI-only and nonblocking. Completion transfers
	// result ownership asynchronously, at most once, unless canceled first.
	SceneEngine *SceneCapability
	BeginScene  func(owner View, request SceneRequest, complete func(SceneResult, error)) (cancel func(), err error)
	// BeginImage decodes the active view's binary document without giving the
	// view a path or source bytes. It is UI-only and nonblocking. The host
	// captures the current binary authority, token, active owner, and
	// generation. Completion runs on the UI thread, at most once unless
	// canceled, and transfers successful Pixels ownership to the view. Cancel
	// is idempotent and UI-only. A stale completion drops only its pixels.
	BeginImage func(owner View, request ImageRequest, complete func(ImageResult, error)) (cancel func(), err error)
	ShowSource func(owner View) bool
}
type Adapter struct {
	ID, Name string
	Matches  func(string) bool
	Create   func(Host) View
	Outline  func(Host) OutlineSource
	Template func() string
	Exports  []string
	// ExportEngine advertises a separately packaged export capability. The host
	// owns execution and publication; editing and preview remain native Views.
	ExportEngine *ExportEngine
	SceneEngine  *SceneCapability
}

type ExportEngine struct {
	ID, Format string
	// RequiresView means the host must prepare a mounted format view before
	// exporting. Existing file-only engines leave this false.
	RequiresView bool
}
