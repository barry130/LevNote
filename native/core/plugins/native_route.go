package plugins

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

// NativeRoute is a closed data-only reference to a shipped native adapter.
// It never names a path, executable, or independently installed engine.
type NativeRoute string

const (
	NativeRouteMarkdown NativeRoute = "builtin.markdown"
	NativeRouteMindmap  NativeRoute = "builtin.mindmap"
	NativeRouteDrawio   NativeRoute = "builtin.drawio"
)

type manifestField struct {
	name  string
	value json.RawMessage
}

// discoverManifestFields runs only after the byte/UTF-8 admission checks. Keep
// every top-level key until dispatch: a final schema:2 must not hide an earlier
// schema:3, and decoded names expose escaped duplicates. Legacy duplicates
// retain their existing last-value behavior unless schema3 is involved.
func discoverManifestFields(data []byte) ([]manifestField, bool, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	opening, err := decoder.Token()
	if err != nil {
		return nil, false, err
	}
	if opening != json.Delim('{') {
		return nil, false, errors.New("manifest must be an object")
	}
	var fields []manifestField
	strict := false
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return nil, false, err
		}
		name, ok := key.(string)
		if !ok {
			return nil, false, errors.New("manifest field name must be a string")
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return nil, false, err
		}
		fields = append(fields, manifestField{name: name, value: value})
		if strings.EqualFold(name, "nativeRoute") {
			strict = true
		}
		if strings.EqualFold(name, "schema") {
			var schema int
			if json.Unmarshal(value, &schema) == nil && schema == 3 {
				strict = true
			}
		}
	}
	if _, err := decoder.Token(); err != nil {
		return nil, false, err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return nil, false, errors.New("manifest has trailing data")
	}
	return fields, strict, nil
}

func decodeNativeRouteManifest(data []byte, fields []manifestField) (SyntaxManifest, error) {
	if hasIllFormedSurrogateEscape(data) {
		return SyntaxManifest{}, errors.New("schema3 contains an ill-formed Unicode surrogate escape")
	}
	present := make(map[string]bool, len(fields))
	for _, field := range fields {
		if present[field.name] {
			return SyntaxManifest{}, fmt.Errorf("duplicate field %q", field.name)
		}
		present[field.name] = true
		switch field.name {
		case "schema", "id", "name", "version", "apiVersion", "extensions", "nativeRoute",
			"lineComment", "blockStart", "blockEnd", "keywords":
		default:
			return SyntaxManifest{}, fmt.Errorf("unknown schema3 field %q", field.name)
		}
		if bytes.Equal(bytes.TrimSpace(field.value), []byte("null")) {
			return SyntaxManifest{}, fmt.Errorf("field %q must not be null", field.name)
		}
		if field.name == "extensions" || field.name == "keywords" {
			var values []json.RawMessage
			if err := json.Unmarshal(field.value, &values); err != nil {
				return SyntaxManifest{}, fmt.Errorf("field %q: %w", field.name, err)
			}
			for _, value := range values {
				if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
					return SyntaxManifest{}, fmt.Errorf("field %q must not contain null", field.name)
				}
			}
		}
	}
	for _, required := range []string{"schema", "id", "name", "extensions", "nativeRoute"} {
		if !present[required] {
			return SyntaxManifest{}, fmt.Errorf("field %q is required", required)
		}
	}
	var manifest SyntaxManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return SyntaxManifest{}, err
	}
	if manifest.Schema != 3 {
		return SyntaxManifest{}, errors.New("nativeRoute requires schema3")
	}
	if !present["version"] {
		manifest.Version = "1.0.0"
	}
	if !present["apiVersion"] {
		manifest.APIVersion = APIVersion
	}
	if err := validateSyntax(manifest); err != nil {
		return SyntaxManifest{}, err
	}
	return manifest, nil
}
