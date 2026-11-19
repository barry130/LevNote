// Package drawio implements a native, self-authored reader and safe editor for
// draw.io XML files. It never loads draw.io or any web runtime.
package drawio

import "errors"

const (
	ShapeRectangle = "rectangle"
	ShapeEllipse   = "ellipse"
	ShapeDiamond   = "diamond"
	ShapeRounded   = "rounded"
	ShapeConnector = "connector"
)

var (
	ErrInvalidXML      = errors.New("invalid draw.io XML")
	ErrUnsupported     = errors.New("unsupported draw.io construct")
	ErrReadOnly        = errors.New("draw.io page is read-only")
	ErrPageNotFound    = errors.New("draw.io page not found")
	ErrCellNotFound    = errors.New("draw.io cell not found")
	ErrDuplicateID     = errors.New("draw.io cell id is duplicated")
	ErrGeometryMissing = errors.New("draw.io geometry is missing")
	ErrLimitExceeded   = errors.New("draw.io limit exceeded")
	ErrInvalidGeometry = errors.New("invalid draw.io geometry")
	ErrInvalidShape    = errors.New("unsupported native draw.io shape")
	ErrEmptyDocument   = errors.New("draw.io document contains no model")
	ErrSVGOutputLimit  = errors.New("draw.io SVG output exceeds byte limit")
)

type Severity string

const (
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

type IssueCode string

const (
	IssueHTMLLabel          IssueCode = "HTML_LABEL"
	IssueImage              IssueCode = "IMAGE"
	IssueStencil            IssueCode = "STENCIL"
	IssueUnsupportedElement IssueCode = "UNSUPPORTED_ELEMENT"
	IssueMissingGeometry    IssueCode = "MISSING_GEOMETRY"
	IssueMissingEndpoint    IssueCode = "MISSING_ENDPOINT"
	IssueCompressedReadOnly IssueCode = "COMPRESSED_READ_ONLY"
	IssueEmptyPage          IssueCode = "EMPTY_PAGE"
)

type Issue struct {
	Severity Severity
	Code     IssueCode
	Message  string
	Page     int
	CellID   string
}

type Point struct {
	X float64
	Y float64
}

type Rect struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}

// Node is a rendered mxCell vertex and the stable model used by the host canvas.
type Node struct {
	ID          string
	Text        string
	PlainText   string
	X           float64
	Y           float64
	Width       float64
	Height      float64
	Shape       string
	Parent      string
	Style       string
	HTML        bool
	Image       bool
	Stencil     bool
	Unsupported []string

	startTagStart int
	startTagEnd   int
	endOffset     int
	geometryStart int
	geometryEnd   int
}

// Edge is a native connector model. SourcePoint and TargetPoint are set for
// detached endpoints; node endpoints resolve through SourceID and TargetID.
type Edge struct {
	ID          string
	Text        string
	PlainText   string
	SourceID    string
	TargetID    string
	Parent      string
	Style       string
	Points      []Point
	SourcePoint *Point
	TargetPoint *Point
	HTML        bool
	Unsupported []string

	startTagStart int
	startTagEnd   int
	endOffset     int
	geometryStart int
	geometryEnd   int
}

type Model struct {
	ID             string
	Nodes          []Node
	Edges          []Edge
	Bounds         Rect
	ReadOnlyReason string
}

// Scene aliases Model; callers that expect drawio's "scene" terminology can
// use either name.
type Scene = Model

type Page struct {
	Index          int
	ID             string
	Name           string
	Compressed     bool
	ReadOnly       bool
	ReadOnlyReason string
	Model          Model

	modelStart int
	modelEnd   int
	rootClose  int
}

type Document struct {
	Pages    []Page
	Issues   []Issue
	Source   []byte
	IsMxFile bool

	original []byte
	source   []byte
	options  ParseOptions
	dirty    bool
}

func (d *Document) Dirty() bool { return d != nil && d.dirty }

func (d *Document) OriginalBytes() []byte {
	if d == nil {
		return nil
	}
	return cloneBytes(d.original)
}

type ParseOptions struct {
	MaxBytes             int
	MaxDecompressedBytes int
	MaxXMLDepth          int
	MaxElements          int
	MaxAttributes        int
	MaxTextBytes         int
}

type SVGOptions struct {
	Padding       float64
	Background    string
	DefaultFill   string
	DefaultStroke string
	StrokeWidth   float64
	FontFamily    string
	FontSize      float64
	MaxBytes      int
}

type ShapeSpec struct {
	ID     string
	Text   string
	Shape  string
	Parent string
	X      float64
	Y      float64
	Width  float64
	Height float64
}

type ConnectorSpec struct {
	ID       string
	Text     string
	Parent   string
	SourceID string
	TargetID string
}

type ReadOnlyError struct {
	Page   int
	Reason string
}

func (e *ReadOnlyError) Error() string {
	if e == nil {
		return ""
	}
	return "draw.io page " + itoa(e.Page) + " is read-only: " + e.Reason
}

func (e *ReadOnlyError) Unwrap() error { return ErrReadOnly }

func cloneBytes(value []byte) []byte {
	if value == nil {
		return nil
	}
	return append([]byte(nil), value...)
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	var digits [24]byte
	index := len(digits)
	for value > 0 {
		index--
		digits[index] = byte('0' + value%10)
		value /= 10
	}
	if negative {
		index--
		digits[index] = '-'
	}
	return string(digits[index:])
}
