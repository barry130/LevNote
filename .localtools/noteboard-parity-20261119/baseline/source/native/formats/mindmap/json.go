package mindmap

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
)

func parseOrderedObject(raw json.RawMessage) (Object, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	token, err := decoder.Token()
	if err != nil {
		return Object{}, err
	}
	delim, ok := token.(json.Delim)
	if !ok || delim != '{' {
		return Object{}, ErrInvalidJSON
	}
	object := Object{}
	seen := make(map[string]struct{})
	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return Object{}, err
		}
		key, ok := keyToken.(string)
		if !ok {
			return Object{}, ErrInvalidJSON
		}
		if _, exists := seen[key]; exists {
			return Object{}, fmt.Errorf("%w: duplicate object key %q", ErrInvalidJSON, key)
		}
		seen[key] = struct{}{}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return Object{}, err
		}
		object.Members = append(object.Members, Member{Key: key, Raw: cloneRaw(value)})
	}
	if _, err := decoder.Token(); err != nil {
		return Object{}, err
	}
	if token, err := decoder.Token(); err != io.EOF {
		if err != nil {
			return Object{}, err
		}
		return Object{}, fmt.Errorf("%w: trailing JSON token %v", ErrInvalidJSON, token)
	}
	return object, nil
}

func validateJSON(data []byte, maxDepth int) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	values := 0
	if err := validateJSONValue(decoder, 0, maxDepth, &values); err != nil {
		return err
	}
	if token, err := decoder.Token(); err != io.EOF {
		if err != nil {
			return err
		}
		return fmt.Errorf("%w: trailing JSON token %v", ErrInvalidJSON, token)
	}
	return nil
}

func validateJSONValue(decoder *json.Decoder, depth, maxDepth int, values *int) error {
	if depth > maxDepth {
		return fmt.Errorf("%w: JSON depth exceeds %d", ErrLimitExceeded, maxDepth)
	}
	(*values)++
	if *values > 4_000_000 {
		return fmt.Errorf("%w: too many JSON values", ErrLimitExceeded)
	}
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidJSON, err)
	}
	delim, isDelim := token.(json.Delim)
	if !isDelim {
		return nil
	}
	switch delim {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return fmt.Errorf("%w: %v", ErrInvalidJSON, err)
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("%w: object key is not a string", ErrInvalidJSON)
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("%w: duplicate object key %q", ErrInvalidJSON, key)
			}
			seen[key] = struct{}{}
			if err := validateJSONValue(decoder, depth+1, maxDepth, values); err != nil {
				return err
			}
		}
		if closing, err := decoder.Token(); err != nil || closing != json.Delim('}') {
			return fmt.Errorf("%w: unterminated object", ErrInvalidJSON)
		}
	case '[':
		for decoder.More() {
			if err := validateJSONValue(decoder, depth+1, maxDepth, values); err != nil {
				return err
			}
		}
		if closing, err := decoder.Token(); err != nil || closing != json.Delim(']') {
			return fmt.Errorf("%w: unterminated array", ErrInvalidJSON)
		}
	default:
		return fmt.Errorf("%w: unexpected delimiter %q", ErrInvalidJSON, delim)
	}
	return nil
}

type encodeMember struct {
	key   string
	value any
}

type encodeObject []encodeMember
type encodeArray []any

func writeJSONObject(dst *bytes.Buffer, object encodeObject, indent int, depth int, newline string) {
	if len(object) == 0 {
		dst.WriteString("{}")
		return
	}
	dst.WriteByte('{')
	dst.WriteString(newline)
	for i, member := range object {
		dst.WriteString(makeIndent((depth + 1) * indent))
		writeJSONString(dst, member.key)
		dst.WriteString(": ")
		writeJSONValue(dst, member.value, indent, depth+1, newline)
		if i != len(object)-1 {
			dst.WriteByte(',')
		}
		dst.WriteString(newline)
	}
	dst.WriteString(makeIndent(depth * indent))
	dst.WriteByte('}')
}

func writeJSONArray(dst *bytes.Buffer, values encodeArray, indent int, depth int, newline string) {
	if len(values) == 0 {
		dst.WriteString("[]")
		return
	}
	dst.WriteByte('[')
	dst.WriteString(newline)
	for i, value := range values {
		dst.WriteString(makeIndent((depth + 1) * indent))
		writeJSONValue(dst, value, indent, depth+1, newline)
		if i != len(values)-1 {
			dst.WriteByte(',')
		}
		dst.WriteString(newline)
	}
	dst.WriteString(makeIndent(depth * indent))
	dst.WriteByte(']')
}

func writeJSONValue(dst *bytes.Buffer, value any, indent int, depth int, newline string) {
	switch typed := value.(type) {
	case nil:
		dst.WriteString("null")
	case string:
		writeJSONString(dst, typed)
	case bool:
		if typed {
			dst.WriteString("true")
		} else {
			dst.WriteString("false")
		}
	case int:
		fmt.Fprint(dst, typed)
	case json.Number:
		dst.WriteString(typed.String())
	case json.RawMessage:
		dst.Write(typed)
	case Object:
		writeJSONObject(dst, encodeObjectFromObject(typed), indent, depth, newline)
	case encodeObject:
		writeJSONObject(dst, typed, indent, depth, newline)
	case encodeArray:
		writeJSONArray(dst, typed, indent, depth, newline)
	case []string:
		array := make(encodeArray, len(typed))
		for i, entry := range typed {
			array[i] = entry
		}
		writeJSONArray(dst, array, indent, depth, newline)
	default:
		panic(fmt.Sprintf("mindmap: unsupported canonical JSON value %T", value))
	}
}

func writeJSONString(dst *bytes.Buffer, value string) {
	encoded, _ := json.Marshal(value)
	dst.Write(encoded)
}

func encodeObjectFromObject(object Object) encodeObject {
	result := make(encodeObject, len(object.Members))
	for i, member := range object.Members {
		result[i] = encodeMember{key: member.Key, value: member.Raw}
	}
	return result
}

func makeIndent(width int) string {
	if width <= 0 {
		return ""
	}
	return string(bytes.Repeat([]byte{' '}, width))
}

func hashBytes(data []byte) [32]byte { return sha256.Sum256(data) }
