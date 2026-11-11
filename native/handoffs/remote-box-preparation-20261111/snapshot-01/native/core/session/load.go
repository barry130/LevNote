package session

import (
	"bytes"
	"encoding/json"
	"errors"
)

type decodedTab struct {
	tab Tab
	err error
}

// boundedTabs decodes the first MaxTabs array entries and counts the rest.
// Decoding []json.RawMessage first allocates a descriptor and byte copy for
// every entry, even if nearly all of them will be omitted. The scanner only
// walks validated JSON and holds no additional data for omitted entries.
type boundedTabs struct {
	entries []decodedTab
	omitted int
}

func (tabs *boundedTabs) UnmarshalJSON(data []byte) error {
	*tabs = boundedTabs{}
	data = bytes.TrimSpace(data)
	if bytes.Equal(data, []byte("null")) {
		return nil
	}
	if len(data) < 2 || data[0] != '[' || !json.Valid(data) {
		return errors.New("session tabs must be a valid JSON array")
	}
	for i := 1; i < len(data); {
		for i < len(data) && jsonSpace(data[i]) {
			i++
		}
		if data[i] == ']' {
			break
		}
		start, depth, quoted := i, 0, false
		for ; i < len(data); i++ {
			b := data[i]
			if quoted {
				if b == '\\' {
					i++ // validated JSON guarantees a complete escape
				} else if b == '"' {
					quoted = false
				}
				continue
			}
			if depth == 0 && (b == ',' || b == ']') {
				break
			}
			switch b {
			case '"':
				quoted = true
			case '{', '[':
				depth++
			case '}', ']':
				depth--
			}
		}
		if len(tabs.entries) < MaxTabs {
			var entry decodedTab
			entry.err = decodeTab(data[start:i], &entry.tab)
			if entry.err == nil {
				entry.err = validateTab(entry.tab)
			}
			tabs.entries = append(tabs.entries, entry)
		} else {
			tabs.omitted++
		}
		if data[i] == ']' {
			break
		}
		i++ // comma; next loop consumes surrounding whitespace
	}
	return nil
}

// decodeTab discriminates by the raw record key, then decodes the appropriate
// small branch. Ordinary text records still have exactly one draft-sized
// unmarshal; binary records reject forbidden keys by presence, even when the
// value is empty, null or false.
func decodeTab(data []byte, tab *Tab) error {
	fields, err := objectFields(data)
	if err != nil {
		return err
	}
	binary, hasRecord := fields[binaryRecordKey]
	if hasRecord {
		var discriminator string
		if err := json.Unmarshal(binary.value, &discriminator); err != nil || discriminator != binaryRecordValue {
			return errors.New("session tab has unknown record discriminator")
		}
		var wire binaryTabWire
		if err := json.Unmarshal(data, &wire); err != nil {
			return err
		}
		for key := range fields {
			switch key {
			case binaryRecordKey, binaryKindKey, binaryPathKey, binaryTitleKey, binaryPluginKey, binaryViewKey:
			case binaryDraftKey, binaryEncodingKey, binaryBOMKey, binaryOriginalPathKey, binarySelectionKey, binaryScrollKey:
				return errors.New("binary tab contains forbidden text field")
			default:
				return errors.New("binary tab contains unknown field")
			}
		}
		if wire.Record != binaryRecordValue || wire.Path == "" {
			return errors.New("binary tab record is invalid")
		}
		if _, exists := fields[binaryKindKey]; !exists {
			return errors.New("binary tab kind is required")
		}
		if string(bytes.TrimSpace(fields[binaryKindKey].value)) == "null" {
			return errors.New("binary tab kind is required")
		}
		binaryTab, err := newBinaryTab(BinaryTab{
			Path: wire.Path, Kind: wire.Kind, Title: wire.Title,
			PluginID: wire.PluginID, View: wire.View,
		})
		if err != nil {
			return err
		}
		*tab = binaryTab
		return nil
	}
	return json.Unmarshal(data, tab)
}

type wireField struct {
	value []byte
}

func objectFields(data []byte) (map[string]wireField, error) {
	data = bytes.TrimSpace(data)
	if len(data) < 2 || data[0] != '{' || data[len(data)-1] != '}' {
		return nil, errors.New("session tab must be a JSON object")
	}
	fields := make(map[string]wireField)
	i := 1
	for i < len(data) {
		for i < len(data) && jsonSpace(data[i]) {
			i++
		}
		if i == len(data)-1 {
			break
		}
		if data[i] != '"' {
			return nil, errors.New("session tab key is not a string")
		}
		keyStart := i
		i++
		for ; i < len(data); i++ {
			if data[i] == '\\' {
				i++
				continue
			}
			if data[i] == '"' {
				break
			}
		}
		if i >= len(data) {
			return nil, errors.New("session tab key is unterminated")
		}
		quotedKey := data[keyStart : i+1]
		i++
		for i < len(data)-1 && jsonSpace(data[i]) {
			i++
		}
		if i >= len(data) || data[i] != ':' {
			return nil, errors.New("session tab field is missing a separator")
		}
		i++
		for i < len(data)-1 && jsonSpace(data[i]) {
			i++
		}
		valueStart := i
		depth, quoted := 0, false
		for ; i < len(data); i++ {
			b := data[i]
			if !quoted && depth == 0 && (b == ',' || b == '}') {
				break
			}
			if quoted {
				if b == '\\' {
					i++
				} else if b == '"' {
					quoted = false
				}
				continue
			}
			switch b {
			case '"':
				quoted = true
			case '{', '[':
				depth++
			case '}', ']':
				depth--
			case ':':
				if depth == 0 {
					return nil, errors.New("session tab field value is invalid")
				}
			}
		}
		if i >= len(data) {
			return nil, errors.New("session tab field value is unterminated")
		}
		var key string
		if err := json.Unmarshal(quotedKey, &key); err != nil {
			return nil, err
		}
		if _, exists := fields[key]; exists {
			return nil, errors.New("session tab contains duplicate field")
		}
		fields[key] = wireField{value: data[valueStart:i]}
		if data[i] == '}' {
			break
		}
		i++ // comma; next iteration consumes whitespace
	}
	for i < len(data) && jsonSpace(data[i]) {
		i++
	}
	if i != len(data)-1 {
		return nil, errors.New("session tab object is not terminated")
	}
	return fields, nil
}

func jsonSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\r' || b == '\n'
}
