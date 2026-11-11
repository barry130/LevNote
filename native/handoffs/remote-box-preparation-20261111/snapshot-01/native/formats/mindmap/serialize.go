package mindmap

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func (d *Document) Serialize(options SerializeOptions) ([]byte, error) {
	if d == nil {
		return nil, ErrInvalidDocument
	}
	if options.Indent == 0 {
		options.Indent = 2
	}
	if options.Indent < 0 || options.Indent > 16 {
		return nil, fmt.Errorf("%w: indent must be between 0 and 16", ErrInvalidDocument)
	}
	if options.LineEnding == "" {
		options.LineEnding = "\n"
	}
	if options.LineEnding != "\n" && options.LineEnding != "\r\n" {
		return nil, fmt.Errorf("%w: line ending must be LF or CRLF", ErrInvalidDocument)
	}
	canonical, err := d.canonicalBytes(options)
	if err != nil {
		return nil, err
	}
	if !options.ForceCanonical {
		currentHash := hashBytes(canonical)
		if d.original != nil && !d.dirty && currentHash == d.baseline {
			return cloneRaw(d.original), nil
		}
	}
	return canonical, nil
}

func (d *Document) canonicalBytes(options SerializeOptions) ([]byte, error) {
	if options.Indent == 0 {
		options.Indent = 2
	}
	if options.LineEnding == "" {
		options.LineEnding = "\n"
	}
	if err := d.validate(); err != nil {
		return nil, err
	}
	top := encodeObject{
		{key: "format", value: Format},
		{key: "version", value: d.Version},
		{key: "layout", value: d.Layout},
	}
	for _, member := range d.Extra.Members {
		if _, known := knownTopKeys[member.Key]; known {
			return nil, fmt.Errorf("%w: top-level extra contains reserved key %q", ErrInvalidDocument, member.Key)
		}
		top = append(top, encodeMember{key: member.Key, value: member.Raw})
	}
	node, err := encodeNode(d.Root)
	if err != nil {
		return nil, err
	}
	top = append(top, encodeMember{key: "root", value: node})

	var buffer bytes.Buffer
	writeJSONObject(&buffer, top, options.Indent, 0, options.LineEnding)
	buffer.WriteString(options.LineEnding)
	return buffer.Bytes(), nil
}

func encodeNode(node *Node) (encodeObject, error) {
	if node == nil {
		return nil, fmt.Errorf("%w: nil node", ErrInvalidDocument)
	}
	object := encodeObject{
		{key: "id", value: node.ID},
		{key: "topic", value: node.Topic},
	}
	used := map[string]struct{}{"id": {}, "topic": {}, "children": {}}
	if node.Note != nil {
		object = append(object, encodeMember{key: "note", value: *node.Note})
		used["note"] = struct{}{}
	}
	if node.Collapsed != nil {
		object = append(object, encodeMember{key: "collapsed", value: *node.Collapsed})
		used["collapsed"] = struct{}{}
	}
	if node.Style != nil {
		if err := validateObjectMembers(*node.Style, "style", nil); err != nil {
			return nil, err
		}
		object = append(object, encodeMember{key: "style", value: *node.Style})
		used["style"] = struct{}{}
	}
	if node.Icons != nil {
		object = append(object, encodeMember{key: "icons", value: *node.Icons})
		used["icons"] = struct{}{}
	}
	if node.Image != nil {
		value, err := encodeImage(*node.Image)
		if err != nil {
			return nil, err
		}
		object = append(object, encodeMember{key: "image", value: value})
		used["image"] = struct{}{}
	}
	if node.HyperLink != nil {
		object = append(object, encodeMember{key: "hyperLink", value: *node.HyperLink})
		used["hyperLink"] = struct{}{}
	}
	if node.Tags != nil {
		value, err := encodeTags(*node.Tags)
		if err != nil {
			return nil, err
		}
		object = append(object, encodeMember{key: "tags", value: value})
		used["tags"] = struct{}{}
	}
	if node.BranchColor != nil {
		object = append(object, encodeMember{key: "branchColor", value: *node.BranchColor})
		used["branchColor"] = struct{}{}
	}
	for _, member := range node.Extra.Members {
		if _, known := knownNodeKeys[member.Key]; known {
			return nil, fmt.Errorf("%w: node %q extra contains reserved key %q", ErrInvalidDocument, node.ID, member.Key)
		}
		if _, duplicate := used[member.Key]; duplicate {
			return nil, fmt.Errorf("%w: node %q duplicates key %q", ErrInvalidDocument, node.ID, member.Key)
		}
		if !json.Valid(member.Raw) {
			return nil, fmt.Errorf("%w: node %q field %q is invalid JSON", ErrInvalidDocument, node.ID, member.Key)
		}
		object = append(object, encodeMember{key: member.Key, value: member.Raw})
		used[member.Key] = struct{}{}
	}
	children := make(encodeArray, len(node.Children))
	for index, child := range node.Children {
		encoded, err := encodeNode(child)
		if err != nil {
			return nil, err
		}
		children[index] = encoded
	}
	object = append(object, encodeMember{key: "children", value: children})
	return object, nil
}

func encodeImage(image Image) (encodeObject, error) {
	if image.URL == "" {
		return nil, fmt.Errorf("%w: image URL is empty", ErrInvalidDocument)
	}
	width, err := decodePositiveNumber(json.RawMessage(image.Width.String()))
	if err != nil {
		return nil, fmt.Errorf("%w: image width is invalid", ErrInvalidDocument)
	}
	height, err := decodePositiveNumber(json.RawMessage(image.Height.String()))
	if err != nil {
		return nil, fmt.Errorf("%w: image height is invalid", ErrInvalidDocument)
	}
	object := encodeObject{
		{key: "url", value: image.URL},
		{key: "width", value: width},
		{key: "height", value: height},
	}
	if image.Fit != "" {
		if image.Fit != "fill" && image.Fit != "contain" && image.Fit != "cover" {
			return nil, fmt.Errorf("%w: image fit is invalid", ErrInvalidDocument)
		}
		object = append(object, encodeMember{key: "fit", value: image.Fit})
	}
	reserved := map[string]struct{}{"url": {}, "width": {}, "height": {}, "fit": {}}
	if err := validateObjectMembers(image.Extra, "image extra", reserved); err != nil {
		return nil, err
	}
	for _, member := range image.Extra.Members {
		object = append(object, encodeMember{key: member.Key, value: member.Raw})
	}
	return object, nil
}

func encodeTags(tags []Tag) (encodeArray, error) {
	array := make(encodeArray, len(tags))
	for i, tag := range tags {
		if !tag.Object {
			array[i] = tag.Text
			continue
		}
		if tag.Text == "" {
			return nil, fmt.Errorf("%w: tag text is empty", ErrInvalidDocument)
		}
		object := encodeObject{{key: "text", value: tag.Text}}
		reserved := map[string]struct{}{"text": {}, "className": {}, "style": {}}
		if tag.ClassNameSet {
			object = append(object, encodeMember{key: "className", value: tag.ClassName})
		}
		if tag.StyleSet {
			if err := validateObjectMembers(tag.Style, "tag style", nil); err != nil {
				return nil, err
			}
			object = append(object, encodeMember{key: "style", value: tag.Style})
		}
		if err := validateObjectMembers(tag.Extra, "tag extra", reserved); err != nil {
			return nil, err
		}
		for _, member := range tag.Extra.Members {
			object = append(object, encodeMember{key: member.Key, value: member.Raw})
		}
		array[i] = object
	}
	return array, nil
}

func validateObjectMembers(object Object, label string, reserved map[string]struct{}) error {
	seen := make(map[string]struct{}, len(object.Members))
	for _, member := range object.Members {
		if _, exists := seen[member.Key]; exists {
			return fmt.Errorf("%w: %s duplicates key %q", ErrInvalidDocument, label, member.Key)
		}
		seen[member.Key] = struct{}{}
		if reserved != nil {
			if _, forbidden := reserved[member.Key]; forbidden {
				return fmt.Errorf("%w: %s contains reserved key %q", ErrInvalidDocument, label, member.Key)
			}
		}
		if !json.Valid(member.Raw) {
			return fmt.Errorf("%w: %s field %q is invalid JSON", ErrInvalidDocument, label, member.Key)
		}
	}
	return nil
}

func (d *Document) validate() error {
	if d == nil || d.Root == nil {
		return fmt.Errorf("%w: root is required", ErrInvalidDocument)
	}
	if d.Version < 1 {
		return fmt.Errorf("%w: version must be positive", ErrInvalidDocument)
	}
	if d.Layout == "" {
		return fmt.Errorf("%w: layout must not be empty", ErrInvalidDocument)
	}
	if err := validateObjectMembers(d.Extra, "top-level extra", knownTopKeys); err != nil {
		return err
	}
	seenIDs := make(map[string]struct{})
	seenPointers := make(map[*Node]struct{})
	count := 0
	var walk func(*Node, int) error
	walk = func(node *Node, depth int) error {
		if node == nil {
			return fmt.Errorf("%w: nil child node", ErrInvalidDocument)
		}
		if depth > defaultMaxDepth {
			return limitError("node depth exceeds %d", defaultMaxDepth)
		}
		if _, seen := seenPointers[node]; seen {
			return fmt.Errorf("%w: node pointer cycle", ErrCycle)
		}
		seenPointers[node] = struct{}{}
		count++
		if count > defaultMaxNodes {
			return limitError("node count exceeds %d", defaultMaxNodes)
		}
		if node.ID == "" {
			return fmt.Errorf("%w: node id is empty", ErrInvalidDocument)
		}
		if _, exists := seenIDs[node.ID]; exists {
			return fmt.Errorf("%w: %q", ErrDuplicateID, node.ID)
		}
		seenIDs[node.ID] = struct{}{}
		if _, err := encodeNodeMetadata(node); err != nil {
			return err
		}
		for _, child := range node.Children {
			if err := walk(child, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(d.Root, 0)
}

func encodeNodeMetadata(node *Node) (struct{}, error) {
	if node.Style != nil {
		if err := validateObjectMembers(*node.Style, "style", nil); err != nil {
			return struct{}{}, err
		}
	}
	if node.Image != nil {
		if _, err := encodeImage(*node.Image); err != nil {
			return struct{}{}, err
		}
	}
	if node.HyperLink != nil && *node.HyperLink == "" {
		return struct{}{}, fmt.Errorf("%w: hyperLink is empty", ErrInvalidDocument)
	}
	if node.BranchColor != nil && *node.BranchColor == "" {
		return struct{}{}, fmt.Errorf("%w: branchColor is empty", ErrInvalidDocument)
	}
	if node.Tags != nil {
		if _, err := encodeTags(*node.Tags); err != nil {
			return struct{}{}, err
		}
	}
	return struct{}{}, nil
}
