// Package mindmap implements LevNote's native, self-authored .mindmap format.
//
// The package is deliberately independent of any editor toolkit. It exposes a
// tree model, deterministic layout, structural edits, and lossless parsing and
// serialization for the JSON format described by docs/format-spec.md §7.1.
package mindmap

import (
	"encoding/json"
	"errors"
)

const (
	Format  = "levnote.mindmap"
	Version = 1

	LayoutLeft  = "logicalStructureLeft"
	LayoutRight = "logicalStructureRight"
	LayoutBoth  = "logicalStructure"
	LayoutOrg   = "orgChart"
)

var (
	ErrReadOnly          = errors.New("mindmap document is not writable")
	ErrNodeNotFound      = errors.New("mindmap node not found")
	ErrDuplicateID       = errors.New("mindmap node id is duplicated")
	ErrRootImmutable     = errors.New("mindmap root node cannot be moved or deleted")
	ErrCycle             = errors.New("mindmap move would create a cycle")
	ErrInvalidDocument   = errors.New("invalid mindmap document")
	ErrInvalidJSON       = errors.New("invalid mindmap JSON")
	ErrLimitExceeded     = errors.New("mindmap limit exceeded")
	ErrUnknownLayoutKind = errors.New("unknown editable mindmap layout")
)

// Member is one ordered JSON object member. Raw remains validated JSON and
// preserves number spellings exactly (for example 1.00 and very large ints).
type Member struct {
	Key string
	Raw json.RawMessage
}

// Object is an ordered JSON object used by the forward-compatibility buckets.
type Object struct {
	Members []Member
}

func (o Object) Clone() Object {
	clone := Object{Members: make([]Member, len(o.Members))}
	for i, member := range o.Members {
		clone.Members[i] = Member{Key: member.Key, Raw: cloneRaw(member.Raw)}
	}
	return clone
}

func (o Object) Len() int { return len(o.Members) }

func (o Object) Get(key string) (json.RawMessage, bool) {
	for _, member := range o.Members {
		if member.Key == key {
			return member.Raw, true
		}
	}
	return nil, false
}

// Set replaces an existing member at its original position or appends a new
// member. The raw value must be valid JSON.
func (o *Object) Set(key string, raw json.RawMessage) error {
	if o == nil {
		return ErrInvalidDocument
	}
	if !json.Valid(raw) {
		return ErrInvalidJSON
	}
	for i := range o.Members {
		if o.Members[i].Key == key {
			o.Members[i].Raw = cloneRaw(raw)
			return nil
		}
	}
	o.Members = append(o.Members, Member{Key: key, Raw: cloneRaw(raw)})
	return nil
}

func (o *Object) Delete(key string) bool {
	if o == nil {
		return false
	}
	for i := range o.Members {
		if o.Members[i].Key == key {
			copy(o.Members[i:], o.Members[i+1:])
			o.Members[len(o.Members)-1] = Member{}
			o.Members = o.Members[:len(o.Members)-1]
			return true
		}
	}
	return false
}

// Image is the editable view of the format's image object. Extra retains every
// nested key, and Raw is the original complete object when it came from disk.
type Image struct {
	URL    string
	Width  json.Number
	Height json.Number
	Fit    string
	Extra  Object
	Raw    json.RawMessage
}

// Tag is either a plain string (Object == false) or a tag object.
type Tag struct {
	Text         string
	ClassName    string
	ClassNameSet bool
	Style        Object
	StyleSet     bool
	Extra        Object
	Object       bool
	Raw          json.RawMessage
}

// Node is one mindmap topic. Optional pointers distinguish an absent property
// from a present empty value; an empty icons/tags array is meaningful.
type Node struct {
	ID          string
	Topic       string
	Children    []*Node
	Note        *string
	Collapsed   *bool
	Style       *Object
	Icons       *[]string
	Image       *Image
	HyperLink   *string
	Tags        *[]Tag
	BranchColor *string
	Extra       Object

	raw map[string]json.RawMessage
}

func NewNode(id, topic string) *Node {
	return &Node{ID: id, Topic: topic}
}

func (n *Node) Clone() *Node {
	if n == nil {
		return nil
	}
	clone := &Node{
		ID:    n.ID,
		Topic: n.Topic,
		Extra: n.Extra.Clone(),
		raw:   make(map[string]json.RawMessage, len(n.raw)),
	}
	clone.Children = make([]*Node, len(n.Children))
	for i, child := range n.Children {
		clone.Children[i] = child.Clone()
	}
	if n.Note != nil {
		value := *n.Note
		clone.Note = &value
	}
	if n.Collapsed != nil {
		value := *n.Collapsed
		clone.Collapsed = &value
	}
	if n.Style != nil {
		value := n.Style.Clone()
		clone.Style = &value
	}
	if n.Icons != nil {
		value := append([]string(nil), (*n.Icons)...)
		clone.Icons = &value
	}
	if n.Image != nil {
		value := *n.Image
		value.Extra = n.Image.Extra.Clone()
		value.Raw = cloneRaw(n.Image.Raw)
		clone.Image = &value
	}
	if n.HyperLink != nil {
		value := *n.HyperLink
		clone.HyperLink = &value
	}
	if n.Tags != nil {
		value := make([]Tag, len(*n.Tags))
		for i, tag := range *n.Tags {
			tag.Style = tag.Style.Clone()
			tag.Extra = tag.Extra.Clone()
			tag.Raw = cloneRaw(tag.Raw)
			value[i] = tag
		}
		clone.Tags = &value
	}
	if n.BranchColor != nil {
		value := *n.BranchColor
		clone.BranchColor = &value
	}
	for key, value := range n.raw {
		clone.raw[key] = cloneRaw(value)
	}
	return clone
}

// Document is a parsed .mindmap file. Original is retained so an unedited
// document can be serialized byte-for-byte.
type Document struct {
	Root    *Node
	Layout  string
	Version int
	Extra   Object
	Issues  []Issue

	original []byte
	baseline [32]byte
	dirty    bool
}

func (d *Document) RootID() string {
	if d == nil || d.Root == nil {
		return ""
	}
	return d.Root.ID
}

func (d *Document) Dirty() bool {
	if d == nil {
		return false
	}
	if d.dirty {
		return true
	}
	body, err := d.canonicalBytes(SerializeOptions{})
	if err != nil {
		return true
	}
	return hashBytes(body) != d.baseline
}

// OriginalBytes returns a copy of the exact bytes supplied to Parse.
func (d *Document) OriginalBytes() []byte {
	if d == nil {
		return nil
	}
	return cloneRaw(d.original)
}

type Severity string

const (
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

type IssueCode string

const (
	IssueFutureVersion IssueCode = "FUTURE_VERSION"
	IssueUnknownLayout IssueCode = "UNKNOWN_LAYOUT"
	IssueUnknownKey    IssueCode = "UNKNOWN_KEY"
	IssueEmptyTopic    IssueCode = "EMPTY_TOPIC"
	IssueLegacyFormat  IssueCode = "LEGACY_FORMAT"
)

type Issue struct {
	Severity Severity
	Code     IssueCode
	Message  string
	NodeID   string
}

type ValidationError struct {
	Code    IssueCode
	Path    string
	Message string
	Cause   error
}

func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	if e.Path == "" {
		return e.Message
	}
	return e.Path + ": " + e.Message
}

func (e *ValidationError) Unwrap() error {
	if e.Cause != nil {
		return e.Cause
	}
	return ErrInvalidDocument
}

type ParseOptions struct {
	MaxBytes int
	MaxDepth int
	MaxNodes int
}

type SerializeOptions struct {
	Indent         int
	LineEnding     string
	ForceCanonical bool
}

func cloneRaw(value json.RawMessage) json.RawMessage {
	if value == nil {
		return nil
	}
	return append(json.RawMessage(nil), value...)
}
