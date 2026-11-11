package engineipc

import (
	"bytes"
	"encoding/json"
	"io"
	"unicode/utf8"
)

// Only the schema's fixed keys can enter this bounded object; arbitrary JSON
// maps, recursive object trees and permissive case folding are not accepted.
func object(data []byte, required []string, optional string) (map[string]json.RawMessage, error) {
	if !utf8.Valid(data) || !validEscapes(data) {
		return nil, invalid("metadata text is invalid")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return nil, invalid("metadata must be an object")
	}
	values := make(map[string]json.RawMessage, len(required)+1)
	for d.More() {
		token, err = d.Token()
		key, ok := token.(string)
		if err != nil || !ok {
			return nil, invalid("invalid metadata key")
		}
		allowed := key == optional && optional != ""
		for _, name := range required {
			allowed = allowed || name == key
		}
		if !allowed || values[key] != nil {
			return nil, invalid("unknown or duplicate metadata key")
		}
		var value json.RawMessage
		if d.Decode(&value) != nil || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return nil, invalid("invalid metadata value")
		}
		values[key] = value
	}
	if token, err = d.Token(); err != nil || token != json.Delim('}') {
		return nil, invalid("unclosed metadata object")
	}
	if _, err = d.Token(); err != io.EOF {
		return nil, invalid("trailing metadata bytes")
	}
	for _, name := range required {
		if values[name] == nil {
			return nil, invalid("missing metadata field")
		}
	}
	return values, nil
}

func decodeRequest(data []byte, expected EngineID, format string) (RequestMeta, error) {
	var m RequestMeta
	required := []string{"engine", "format"}
	if expected == DrawioSVG {
		required = append(required, "page")
	}
	values, err := object(data, required, "")
	if err != nil {
		return m, err
	}
	if json.Unmarshal(data, &m) != nil || m.Engine != expected || m.Format != format {
		return m, invalid("invalid request metadata")
	}
	if expected == DrawioSVG {
		fields, err := object(values["page"], []string{"index", "id"}, "")
		if err != nil {
			return m, err
		}
		page := RequestPage{}
		if json.Unmarshal(fields["index"], &page.Index) != nil || json.Unmarshal(fields["id"], &page.ID) != nil {
			return m, invalid("invalid page field type")
		}
		m.Page = &page
	}
	if !validRequestPage(m.Engine, m.Page) {
		return m, invalid("invalid request page")
	}
	return m, nil
}

func decodeResponse(data []byte, payloadBytes int, sourceBytes uint64) (ResponseMeta, error) {
	var m ResponseMeta
	values, err := object(data, []string{"engine", "engineVersion", "format", "status", "warnings"}, "error")
	if err != nil {
		return m, err
	}
	for _, field := range []struct {
		name   string
		target any
	}{{"engine", &m.Engine}, {"engineVersion", &m.EngineVersion}, {"format", &m.Format}, {"status", &m.Status}} {
		if json.Unmarshal(values[field.name], field.target) != nil {
			return m, invalid("invalid response field type")
		}
	}
	d := json.NewDecoder(bytes.NewReader(values["warnings"]))
	token, err := d.Token()
	if err != nil || token != json.Delim('[') {
		return m, invalid("warnings must be an array")
	}
	m.Warnings = make([]Warning, 0)
	for d.More() {
		if len(m.Warnings) >= WarningsMaxCount {
			return m, invalid("warning count exceeds limit")
		}
		var raw json.RawMessage
		if d.Decode(&raw) != nil {
			return m, invalid("invalid warning entry")
		}
		if _, err := object(raw, []string{"code", "message", "start", "end"}, ""); err != nil {
			return m, err
		}
		var warning Warning
		if json.Unmarshal(raw, &warning) != nil {
			return m, invalid("invalid warning field type")
		}
		m.Warnings = append(m.Warnings, warning)
	}
	if token, err = d.Token(); err != nil || token != json.Delim(']') {
		return m, invalid("unclosed warning array")
	}
	if _, err = d.Token(); err != io.EOF {
		return m, invalid("trailing warning data")
	}
	if raw := values["error"]; raw != nil {
		if _, err := object(raw, []string{"code", "message"}, ""); err != nil {
			return m, err
		}
		m.Error = &EngineError{}
		if json.Unmarshal(raw, m.Error) != nil {
			return m, invalid("invalid engine error field type")
		}
	}
	if err := validateResponse(m, payloadBytes, sourceBytes); err != nil {
		return m, err
	}
	return m, nil
}

// encoding/json repairs invalid surrogate escapes, so verify them before it
// sees the bytes. Escaped backslashes are skipped without interpreting \u.
func validEscapes(data []byte) bool {
	for i := 0; i < len(data); i++ {
		if data[i] != '\\' {
			continue
		}
		i++
		if i >= len(data) {
			return false
		}
		if data[i] != 'u' {
			continue
		}
		if i+4 >= len(data) {
			return false
		}
		v, ok := quad(data[i+1 : i+5])
		if !ok {
			return false
		}
		i += 4
		if v >= 0xdc00 && v <= 0xdfff {
			return false
		}
		if v < 0xd800 || v > 0xdbff {
			continue
		}
		if i+6 >= len(data) || data[i+1] != '\\' || data[i+2] != 'u' {
			return false
		}
		v, ok = quad(data[i+3 : i+7])
		if !ok || v < 0xdc00 || v > 0xdfff {
			return false
		}
		i += 6
	}
	return true
}
func quad(data []byte) (uint16, bool) {
	var v uint16
	for _, c := range data {
		v <<= 4
		switch {
		case c >= '0' && c <= '9':
			v |= uint16(c - '0')
		case c >= 'a' && c <= 'f':
			v |= uint16(c - 'a' + 10)
		case c >= 'A' && c <= 'F':
			v |= uint16(c - 'A' + 10)
		default:
			return 0, false
		}
	}
	return v, true
}
