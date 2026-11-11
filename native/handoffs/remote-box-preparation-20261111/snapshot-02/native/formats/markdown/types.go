// Package markdown implements a practical CommonMark subset without rewriting source text.
package markdown

// SourceRange is a half-open byte range into the original Markdown source.
type SourceRange struct {
	Start int
	End   int
}

// Warning is non-fatal parse information. The source is always retained.
type Warning struct {
	Code    string
	Message string
	Range   SourceRange
}

type BlockKind string

const (
	BlockHeading   BlockKind = "heading"
	BlockParagraph BlockKind = "paragraph"
	BlockCode      BlockKind = "code"
	BlockList      BlockKind = "list"
	BlockQuote     BlockKind = "quote"
	BlockHTML      BlockKind = "html"
	BlockDivider   BlockKind = "divider"
	BlockDocBlock  BlockKind = "doc-block"
	BlockTable     BlockKind = "table"
)

type InlineKind string

const (
	InlineBold   InlineKind = "bold"
	InlineItalic InlineKind = "italic"
	InlineCode   InlineKind = "code"
	InlineLink   InlineKind = "link"
	InlineImage  InlineKind = "image"
)

type InlineSpan struct {
	Kind  InlineKind
	Range SourceRange
	Text  string
	Href  string
}

// TaskMarker identifies a task checkbox at the start of a list item body.
// Range covers exactly the three original source bytes of [ ], [x], or [X].
type TaskMarker struct {
	Range   SourceRange
	Checked bool
}

type ListItem struct {
	Range     SourceRange
	TextRange SourceRange
	Text      string
	Inline    []InlineSpan
	Task      *TaskMarker
}

type DocBlockAttrs struct {
	ID    string
	Kind  string
	Mode  string
	Src   string
	Title string
	Extra map[string]string
}

type DocBlockRef struct {
	Attrs DocBlockAttrs
	// HeaderRange is the exact byte interval from after the opening fence
	// line through the byte before the separator line. It excludes the
	// separator and payload.
	HeaderRange  SourceRange
	Payload      string
	FenceLength  int
	PayloadRange SourceRange
	// ReferenceProblem is empty when the recognized reference header is
	// structurally usable. A normal inline block without src can also be
	// structurally valid and still must not expose an open-reference action;
	// activation requires empty ReferenceProblem and explicit mode reference.
	// It never promises host support, file existence, navigation permission,
	// destination size, or readonly status. The parser keeps this string
	// bounded and never embeds source header values.
	ReferenceProblem string
}

type TableAlignment uint8

const (
	TableAlignDefault TableAlignment = iota
	TableAlignLeft
	TableAlignCenter
	TableAlignRight
)

type TableCellRef struct {
	Range     SourceRange
	TextRange SourceRange
}

type TableRowRef struct {
	Range SourceRange
	Cells []TableCellRef
}

type TableRef struct {
	Rows           []TableRowRef
	DelimiterRange SourceRange
	Alignments     []TableAlignment
	Problem        string
}

type Block struct {
	Kind      BlockKind
	Range     SourceRange
	TextRange SourceRange
	Text      string
	Inline    []InlineSpan
	Level     int
	Language  string
	Ordered   bool
	Start     int
	Items     []ListItem
	DocBlock  DocBlockRef
	Table     TableRef
	Raw       string
}

type Document struct {
	source   string
	Blocks   []Block
	Warnings []Warning
	// Partial is true when a deterministic parser limit stopped before the
	// entire source was interpreted. Raw always retains the complete source.
	Partial bool
}

func (d Document) Raw() string { return d.source }
