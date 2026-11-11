package mindmap

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

// DefaultMaxBytes is the input byte limit used by zero-valued ParseOptions.
// Callers can inspect a buffer length before materializing it for Parse.
const DefaultMaxBytes = 16 << 20

const (
	defaultMaxDepth = 256
	defaultMaxNodes = 100_000
	hardMaxBytes    = 64 << 20
	hardMaxNodes    = 500_000
)

var knownTopKeys = map[string]struct{}{
	"format": {}, "version": {}, "layout": {}, "root": {},
}

var knownNodeKeys = map[string]struct{}{
	"id": {}, "topic": {}, "note": {}, "collapsed": {}, "style": {},
	"icons": {}, "image": {}, "hyperLink": {}, "tags": {},
	"branchColor": {}, "children": {},
}

func normalizedParseOptions(options ParseOptions) ParseOptions {
	if options.MaxBytes <= 0 || options.MaxBytes > hardMaxBytes {
		options.MaxBytes = DefaultMaxBytes
	}
	if options.MaxDepth <= 0 {
		options.MaxDepth = defaultMaxDepth
	}
	if options.MaxDepth > 1024 {
		options.MaxDepth = 1024
	}
	if options.MaxNodes <= 0 || options.MaxNodes > hardMaxNodes {
		options.MaxNodes = defaultMaxNodes
	}
	return options
}

// Parse accepts only LevNote v1 .mindmap JSON. ADR 0038 deliberately removed
// the earlier qingmo.mindmap compatibility layer.
func Parse(data []byte, options ParseOptions) (*Document, []Issue, error) {
	options = normalizedParseOptions(options)
	if len(data) > options.MaxBytes {
		return nil, nil, limitError("input is larger than %d bytes", options.MaxBytes)
	}
	if !utf8.Valid(data) {
		return nil, nil, fmt.Errorf("%w: input is not UTF-8", ErrInvalidJSON)
	}
	if err := validateJSON(data, options.MaxDepth); err != nil {
		return nil, nil, err
	}
	top, err := parseOrderedObject(data)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrInvalidJSON, err)
	}
	issues := make([]Issue, 0)

	formatRaw, ok := top.Get("format")
	if !ok {
		return nil, issues, validation("MISSING_FORMAT", "format", "format is required")
	}
	format, err := decodeString(formatRaw)
	if err != nil {
		return nil, issues, validation("BAD_FORMAT", "format", "format must be a string")
	}
	if format != Format {
		if format == "qingmo.mindmap" {
			issues = append(issues, Issue{
				Severity: SeverityError,
				Code:     IssueLegacyFormat,
				Message:  "qingmo.mindmap is a pre-1.0 format and is intentionally unsupported",
			})
		}
		return nil, issues, validation("WRONG_FORMAT", "format", fmt.Sprintf("expected %q, got %q", Format, format))
	}

	versionRaw, ok := top.Get("version")
	if !ok {
		return nil, issues, validation("MISSING_VERSION", "version", "version is required")
	}
	version, err := decodeInteger(versionRaw)
	if err != nil || version < 1 {
		return nil, issues, validation("BAD_VERSION", "version", "version must be a positive integer")
	}
	if version > Version {
		issues = append(issues, Issue{
			Severity: SeverityWarning,
			Code:     IssueFutureVersion,
			Message:  fmt.Sprintf("file version %d is newer than supported version %d; unknown fields are preserved", version, Version),
		})
	}

	layout := LayoutBoth
	if layoutRaw, exists := top.Get("layout"); exists {
		layout, err = decodeString(layoutRaw)
		if err != nil || layout == "" {
			return nil, issues, validation("BAD_LAYOUT", "layout", "layout must be a non-empty string")
		}
		if !KnownLayout(layout) {
			issues = append(issues, Issue{
				Severity: SeverityWarning,
				Code:     IssueUnknownLayout,
				Message:  fmt.Sprintf("unknown layout %q is preserved and renders as %q", layout, LayoutBoth),
			})
		}
	}

	rootRaw, ok := top.Get("root")
	if !ok {
		return nil, issues, validation("MISSING_ROOT", "root", "root is required")
	}
	if firstNonSpace(rootRaw) != '{' {
		return nil, issues, validation("BAD_ROOT", "root", "root must be an object")
	}
	seenIDs := make(map[string]struct{})
	nodeCount := 0
	root, err := parseNode(rootRaw, "root", seenIDs, &nodeCount, options.MaxNodes, options.MaxDepth, &issues, 0)
	if err != nil {
		return nil, issues, err
	}

	extra := Object{}
	unknownCount := 0
	for _, member := range top.Members {
		if _, known := knownTopKeys[member.Key]; known {
			continue
		}
		extra.Members = append(extra.Members, member)
		unknownCount++
	}
	if unknownCount > 0 {
		issues = append(issues, Issue{
			Severity: SeverityWarning,
			Code:     IssueUnknownKey,
			Message:  fmt.Sprintf("preserved %d unknown top-level field(s)", unknownCount),
		})
	}

	document := &Document{
		Root:     root,
		Layout:   layout,
		Version:  version,
		Extra:    extra,
		Issues:   append([]Issue(nil), issues...),
		original: cloneRaw(data),
	}
	canonical, err := document.canonicalBytes(SerializeOptions{ForceCanonical: true})
	if err != nil {
		return nil, issues, fmt.Errorf("mindmap canonicalization failed after a successful parse: %w", err)
	}
	document.baseline = hashBytes(canonical)
	return document, issues, nil
}

func parseNode(
	raw json.RawMessage,
	path string,
	seenIDs map[string]struct{},
	count *int,
	maxNodes int,
	maxDepth int,
	issues *[]Issue,
	depth int,
) (*Node, error) {
	if depth > maxDepth {
		return nil, limitError("node depth exceeds %d", maxDepth)
	}
	object, err := parseOrderedObject(raw)
	if err != nil {
		return nil, validation("BAD_NODE", path, "node must be an object with unique keys")
	}
	*count++
	if *count > maxNodes {
		return nil, limitError("node count exceeds %d", maxNodes)
	}

	node := &Node{raw: make(map[string]json.RawMessage)}
	idRaw, ok := object.Get("id")
	if !ok {
		return nil, validation("MISSING_NODE_ID", path+".id", "node id is required")
	}
	node.ID, err = decodeString(idRaw)
	if err != nil || node.ID == "" {
		return nil, validation("MISSING_NODE_ID", path+".id", "node id must be a non-empty string")
	}
	if _, exists := seenIDs[node.ID]; exists {
		return nil, &ValidationError{
			Code:    "DUPLICATE_NODE_ID",
			Path:    path + ".id",
			Message: fmt.Sprintf("duplicate node id %q", node.ID),
			Cause:   ErrDuplicateID,
		}
	}
	seenIDs[node.ID] = struct{}{}

	topicRaw, ok := object.Get("topic")
	if !ok {
		return nil, validation("MISSING_TOPIC", path+".topic", "topic is required")
	}
	node.Topic, err = decodeString(topicRaw)
	if err != nil {
		return nil, validation("MISSING_TOPIC", path+".topic", "topic must be a string")
	}
	if node.Topic == "" {
		*issues = append(*issues, Issue{
			Severity: SeverityWarning,
			Code:     IssueEmptyTopic,
			Message:  "empty topic metadata is preserved",
			NodeID:   node.ID,
		})
	}

	unknownCount := 0
	for _, member := range object.Members {
		node.raw[member.Key] = cloneRaw(member.Raw)
		if _, known := knownNodeKeys[member.Key]; known {
			continue
		}
		node.Extra.Members = append(node.Extra.Members, member)
		unknownCount++
	}
	if unknownCount > 0 {
		*issues = append(*issues, Issue{
			Severity: SeverityWarning,
			Code:     IssueUnknownKey,
			Message:  fmt.Sprintf("node %q has %d preserved unknown field(s)", node.ID, unknownCount),
			NodeID:   node.ID,
		})
	}

	if value, exists := object.Get("note"); exists {
		decoded, err := decodeString(value)
		if err != nil {
			return nil, validation("BAD_FIELD", path+".note", "note must be a string")
		}
		node.Note = &decoded
	}
	if value, exists := object.Get("collapsed"); exists {
		decoded, err := decodeBool(value)
		if err != nil {
			return nil, validation("BAD_FIELD", path+".collapsed", "collapsed must be a boolean")
		}
		node.Collapsed = &decoded
	}
	if value, exists := object.Get("style"); exists {
		if firstNonSpace(value) != '{' {
			return nil, validation("BAD_FIELD", path+".style", "style must be an object")
		}
		style, err := parseOrderedObject(value)
		if err != nil {
			return nil, validation("BAD_FIELD", path+".style", "style must be an object with unique keys")
		}
		node.Style = &style
	}
	if value, exists := object.Get("icons"); exists {
		icons, err := decodeStringArray(value)
		if err != nil {
			return nil, validation("BAD_FIELD", path+".icons", "icons must be an array of strings")
		}
		node.Icons = &icons
	}
	if value, exists := object.Get("image"); exists {
		image, err := parseImage(value)
		if err != nil {
			return nil, validation("BAD_FIELD", path+".image", err.Error())
		}
		node.Image = image
	}
	if value, exists := object.Get("hyperLink"); exists {
		decoded, err := decodeString(value)
		if err != nil || decoded == "" {
			return nil, validation("BAD_FIELD", path+".hyperLink", "hyperLink must be a non-empty string")
		}
		node.HyperLink = &decoded
	}
	if value, exists := object.Get("tags"); exists {
		tags, err := parseTags(value)
		if err != nil {
			return nil, validation("BAD_FIELD", path+".tags", err.Error())
		}
		node.Tags = &tags
	}
	if value, exists := object.Get("branchColor"); exists {
		decoded, err := decodeString(value)
		if err != nil || decoded == "" {
			return nil, validation("BAD_FIELD", path+".branchColor", "branchColor must be a non-empty string")
		}
		node.BranchColor = &decoded
	}

	if childrenRaw, exists := object.Get("children"); exists {
		if firstNonSpace(childrenRaw) != '[' {
			return nil, validation("BAD_CHILDREN", path+".children", "children must be an array")
		}
		var children []json.RawMessage
		if err := json.Unmarshal(childrenRaw, &children); err != nil {
			return nil, validation("BAD_CHILDREN", path+".children", "children must be an array")
		}
		node.Children = make([]*Node, 0, len(children))
		for index, childRaw := range children {
			child, err := parseNode(
				childRaw,
				fmt.Sprintf("%s.children[%d]", path, index),
				seenIDs,
				count,
				maxNodes,
				maxDepth,
				issues,
				depth+1,
			)
			if err != nil {
				return nil, err
			}
			node.Children = append(node.Children, child)
		}
	}
	return node, nil
}

func parseImage(raw json.RawMessage) (*Image, error) {
	if firstNonSpace(raw) != '{' {
		return nil, errors.New("must be an object")
	}
	object, err := parseOrderedObject(raw)
	if err != nil {
		return nil, errors.New("must be an object with unique keys")
	}
	urlRaw, ok := object.Get("url")
	if !ok {
		return nil, errors.New("url is required")
	}
	url, err := decodeString(urlRaw)
	if err != nil || url == "" {
		return nil, errors.New("url must be a non-empty string")
	}
	widthRaw, ok := object.Get("width")
	if !ok {
		return nil, errors.New("width is required")
	}
	width, err := decodePositiveNumber(widthRaw)
	if err != nil {
		return nil, errors.New("width must be a positive finite number")
	}
	heightRaw, ok := object.Get("height")
	if !ok {
		return nil, errors.New("height is required")
	}
	height, err := decodePositiveNumber(heightRaw)
	if err != nil {
		return nil, errors.New("height must be a positive finite number")
	}
	image := &Image{URL: url, Width: width, Height: height, Raw: cloneRaw(raw)}
	if fitRaw, exists := object.Get("fit"); exists {
		fit, err := decodeString(fitRaw)
		if err != nil || (fit != "fill" && fit != "contain" && fit != "cover") {
			return nil, errors.New("fit must be fill, contain, or cover")
		}
		image.Fit = fit
	}
	for _, member := range object.Members {
		switch member.Key {
		case "url", "width", "height", "fit":
		default:
			image.Extra.Members = append(image.Extra.Members, member)
		}
	}
	return image, nil
}

func parseTags(raw json.RawMessage) ([]Tag, error) {
	if firstNonSpace(raw) != '[' {
		return nil, errors.New("must be an array")
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, errors.New("must be an array")
	}
	tags := make([]Tag, 0, len(entries))
	for _, entry := range entries {
		first := firstNonSpace(entry)
		if first == '"' {
			text, err := decodeString(entry)
			if err != nil {
				return nil, errors.New("tag strings must be valid")
			}
			tags = append(tags, Tag{Text: text, Raw: cloneRaw(entry)})
			continue
		}
		if first != '{' {
			return nil, errors.New("each tag must be a string or object")
		}
		object, err := parseOrderedObject(entry)
		if err != nil {
			return nil, errors.New("tag objects must have unique keys")
		}
		textRaw, ok := object.Get("text")
		if !ok {
			return nil, errors.New("tag object text is required")
		}
		text, err := decodeString(textRaw)
		if err != nil || text == "" {
			return nil, errors.New("tag object text must be a non-empty string")
		}
		tag := Tag{Text: text, Object: true, Raw: cloneRaw(entry)}
		if classRaw, exists := object.Get("className"); exists {
			tag.ClassName, err = decodeString(classRaw)
			if err != nil {
				return nil, errors.New("tag className must be a string")
			}
			tag.ClassNameSet = true
		}
		if styleRaw, exists := object.Get("style"); exists {
			if firstNonSpace(styleRaw) != '{' {
				return nil, errors.New("tag style must be an object")
			}
			tag.Style, err = parseOrderedObject(styleRaw)
			if err != nil {
				return nil, errors.New("tag style must be an object with unique keys")
			}
			tag.StyleSet = true
		}
		for _, member := range object.Members {
			switch member.Key {
			case "text", "className", "style":
			default:
				tag.Extra.Members = append(tag.Extra.Members, member)
			}
		}
		tags = append(tags, tag)
	}
	return tags, nil
}

func decodeString(raw json.RawMessage) (string, error) {
	var value string
	if firstNonSpace(raw) != '"' {
		return "", ErrInvalidJSON
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", err
	}
	return value, nil
}

func decodeBool(raw json.RawMessage) (bool, error) {
	switch strings.TrimSpace(string(raw)) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, ErrInvalidJSON
	}
}

func decodeInteger(raw json.RawMessage) (int, error) {
	value := strings.TrimSpace(string(raw))
	if value == "" || strings.ContainsAny(value, ".eE") {
		return 0, ErrInvalidJSON
	}
	return strconv.Atoi(value)
}

func decodePositiveNumber(raw json.RawMessage) (json.Number, error) {
	value := json.Number(strings.TrimSpace(string(raw)))
	number, err := value.Float64()
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) || number <= 0 {
		return "", ErrInvalidJSON
	}
	return value, nil
}

func decodeStringArray(raw json.RawMessage) ([]string, error) {
	if firstNonSpace(raw) != '[' {
		return nil, ErrInvalidJSON
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, err
	}
	if values == nil {
		return nil, ErrInvalidJSON
	}
	return values, nil
}

func firstNonSpace(raw []byte) byte {
	for _, value := range raw {
		switch value {
		case ' ', '\n', '\r', '\t':
			continue
		default:
			return value
		}
	}
	return 0
}

func validation(code IssueCode, path, message string) error {
	return &ValidationError{Code: code, Path: path, Message: message}
}

func limitError(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrLimitExceeded, fmt.Sprintf(format, args...))
}

func KnownLayout(layout string) bool {
	switch layout {
	case LayoutLeft, LayoutRight, LayoutBoth, LayoutOrg:
		return true
	default:
		return false
	}
}
