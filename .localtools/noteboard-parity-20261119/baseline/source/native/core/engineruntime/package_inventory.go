package engineruntime

import (
	"bytes"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	releaseInventoryLimit      = 2 << 20
	releaseInventoryEntryLimit = 256
	inventoryPathMaxBytes      = 1024
	inventoryTextMaxBytes      = 1024
)

type releaseFile struct {
	Path   string
	Size   int64
	SHA256 string
}

type releaseInventory struct {
	Version string
	Files   map[string]releaseFile
}

// parseReleaseInventory is deliberately pure: it opens no file and treats the
// inventory as bounded, local descriptive metadata rather than authentication.
func parseReleaseInventory(data []byte) (releaseInventory, error) {
	if !inventoryDataAllowed(data) {
		return releaseInventory{}, packageInventoryFailure()
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	inventory, err := inventoryDecodeRoot(decoder)
	if err != nil {
		return releaseInventory{}, packageInventoryFailure()
	}
	if _, err := decoder.Token(); err != io.EOF {
		return releaseInventory{}, packageInventoryFailure()
	}
	return inventory, nil
}

func inventoryDataAllowed(data []byte) bool {
	return len(data) != 0 && len(data) <= releaseInventoryLimit &&
		utf8.Valid(data) && validJSONEscapes(data)
}

func packageInventoryFailure() error {
	return failure(FailureIntegrity, "发布清单字段无效或不兼容", nil)
}

func inventoryDecodeRoot(decoder *json.Decoder) (releaseInventory, error) {
	if err := inventoryExpectDelim(decoder, '{'); err != nil {
		return releaseInventory{}, err
	}
	allowed := map[string]bool{
		"schema": true, "version": true, "goos": true, "goarch": true,
		"goamd64": true, "excludes": true, "scope": true,
		"configuration": true, "files": true,
	}
	seen := make(map[string]bool, len(allowed))
	inventory := releaseInventory{Files: make(map[string]releaseFile)}
	for decoder.More() {
		key, err := inventoryDecodeKey(decoder, allowed, seen)
		if err != nil {
			return releaseInventory{}, err
		}
		switch key {
		case "schema":
			value, err := inventoryDecodeInt(decoder)
			if err != nil || value != int64(1) {
				return releaseInventory{}, packageInventoryFailure()
			}
		case "version":
			value, err := inventoryDecodeString(decoder)
			if err != nil || !validVersion(value) {
				return releaseInventory{}, packageInventoryFailure()
			}
			inventory.Version = value
		case "goos":
			value, err := inventoryDecodeString(decoder)
			if err != nil || value != "windows" {
				return releaseInventory{}, packageInventoryFailure()
			}
		case "goarch":
			value, err := inventoryDecodeString(decoder)
			if err != nil || value != "amd64" {
				return releaseInventory{}, packageInventoryFailure()
			}
		case "goamd64":
			value, err := inventoryDecodeString(decoder)
			if err != nil || value != "v1" {
				return releaseInventory{}, packageInventoryFailure()
			}
		case "excludes":
			if err := inventoryDecodeExcludes(decoder); err != nil {
				return releaseInventory{}, err
			}
		case "scope", "configuration":
			if _, err := inventoryDecodeText(decoder); err != nil {
				return releaseInventory{}, err
			}
		case "files":
			files, err := inventoryDecodeFiles(decoder)
			if err != nil {
				return releaseInventory{}, err
			}
			inventory.Files = files
		}
	}
	if err := inventoryExpectDelim(decoder, '}'); err != nil {
		return releaseInventory{}, err
	}
	if len(seen) != len(allowed) {
		return releaseInventory{}, packageInventoryFailure()
	}
	return inventory, nil
}

func inventoryDecodeExcludes(decoder *json.Decoder) error {
	if err := inventoryExpectDelim(decoder, '['); err != nil {
		return err
	}
	if !decoder.More() {
		return packageInventoryFailure()
	}
	value, err := inventoryDecodeString(decoder)
	if err != nil || value != "release-manifest.json" || decoder.More() {
		return packageInventoryFailure()
	}
	return inventoryExpectDelim(decoder, ']')
}

func inventoryDecodeFiles(decoder *json.Decoder) (map[string]releaseFile, error) {
	if err := inventoryExpectDelim(decoder, '['); err != nil {
		return nil, err
	}
	files := make(map[string]releaseFile)
	seenPaths := make(map[string]bool, releaseInventoryEntryLimit)
	count := 0
	for decoder.More() {
		count++
		if count > releaseInventoryEntryLimit {
			return nil, packageInventoryFailure()
		}
		file, err := inventoryDecodeFile(decoder, seenPaths)
		if err != nil {
			return nil, err
		}
		files[file.Path] = file
	}
	if err := inventoryExpectDelim(decoder, ']'); err != nil || count == 0 {
		return nil, packageInventoryFailure()
	}
	return files, nil
}

func inventoryDecodeFile(decoder *json.Decoder, seenPaths map[string]bool) (releaseFile, error) {
	if err := inventoryExpectDelim(decoder, '{'); err != nil {
		return releaseFile{}, err
	}
	allowed := map[string]bool{"path": true, "size": true, "sha256": true}
	seen := make(map[string]bool, len(allowed))
	var file releaseFile
	for decoder.More() {
		key, err := inventoryDecodeKey(decoder, allowed, seen)
		if err != nil {
			return releaseFile{}, err
		}
		switch key {
		case "path":
			path, err := inventoryDecodeText(decoder)
			if err != nil {
				return releaseFile{}, err
			}
			file.Path = path
		case "size":
			value, err := inventoryDecodeInt(decoder)
			if err != nil || value < 0 {
				return releaseFile{}, packageInventoryFailure()
			}
			file.Size = value
		case "sha256":
			value, err := inventoryDecodeString(decoder)
			if err != nil || !inventoryValidSHA256(value) {
				return releaseFile{}, packageInventoryFailure()
			}
			file.SHA256 = strings.ToLower(value)
		}
	}
	if err := inventoryExpectDelim(decoder, '}'); err != nil || len(seen) != len(allowed) {
		return releaseFile{}, packageInventoryFailure()
	}
	if !inventoryValidPath(file.Path) || strings.EqualFold(file.Path, "release-manifest.json") {
		return releaseFile{}, packageInventoryFailure()
	}
	for existing := range seenPaths {
		if strings.EqualFold(existing, file.Path) {
			return releaseFile{}, packageInventoryFailure()
		}
	}
	seenPaths[file.Path] = true
	return file, nil
}

func inventoryDecodeKey(decoder *json.Decoder, allowed map[string]bool, seen map[string]bool) (string, error) {
	token, err := decoder.Token()
	key, ok := token.(string)
	if err != nil || !ok || !allowed[key] || seen[key] {
		return "", packageInventoryFailure()
	}
	seen[key] = true
	return key, nil
}

func inventoryDecodeString(decoder *json.Decoder) (string, error) {
	raw, err := inventoryDecodeRaw(decoder)
	if err != nil {
		return "", err
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", packageInventoryFailure()
	}
	return value, nil
}

func inventoryDecodeText(decoder *json.Decoder) (string, error) {
	value, err := inventoryDecodeString(decoder)
	if err != nil || value == "" || len(value) > inventoryTextMaxBytes {
		return "", packageInventoryFailure()
	}
	return value, nil
}

func inventoryDecodeInt(decoder *json.Decoder) (int64, error) {
	raw, err := inventoryDecodeRaw(decoder)
	if err != nil {
		return 0, err
	}
	value, parseErr := strconv.ParseInt(string(raw), 10, 64)
	if parseErr != nil {
		return 0, packageInventoryFailure()
	}
	return value, nil
}

func inventoryDecodeRaw(decoder *json.Decoder) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := decoder.Decode(&raw); err != nil || len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil, packageInventoryFailure()
	}
	return raw, nil
}

func inventoryExpectDelim(decoder *json.Decoder, expected json.Delim) error {
	token, err := decoder.Token()
	if err != nil || token != expected {
		return packageInventoryFailure()
	}
	return nil
}

func inventoryValidSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for i := range value {
		switch {
		case value[i] >= '0' && value[i] <= '9':
		case value[i] >= 'a' && value[i] <= 'f':
		case value[i] >= 'A' && value[i] <= 'F':
		default:
			return false
		}
	}
	return true
}

func inventoryValidPath(path string) bool {
	if path == "" || len(path) > inventoryPathMaxBytes || !utf8.ValidString(path) {
		return false
	}
	if strings.HasPrefix(path, "/") || strings.ContainsAny(path, "\\\x00:") {
		return false
	}
	for _, segment := range strings.Split(path, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
		if strings.HasSuffix(segment, ".") || strings.HasSuffix(segment, " ") {
			return false
		}
		for _, r := range segment {
			if r < 0x20 || r == 0x7f || strings.ContainsRune(`<>:"|?*`, r) {
				return false
			}
		}
		if inventoryReservedDeviceName(segment) {
			return false
		}
	}
	return true
}

func inventoryReservedDeviceName(segment string) bool {
	name := strings.ToUpper(strings.SplitN(segment, ".", 2)[0])
	name = strings.TrimRight(name, " ")
	switch name {
	case "CON", "PRN", "AUX", "NUL", "CONIN$", "CONOUT$":
		return true
	}
	if strings.HasPrefix(name, "COM") {
		suffix := name[len("COM"):]
		return suffix != "" && inventoryDeviceNumber(suffix)
	}
	if strings.HasPrefix(name, "LPT") {
		suffix := name[len("LPT"):]
		return suffix != "" && inventoryDeviceNumber(suffix)
	}
	return false
}

func inventoryDeviceNumber(value string) bool {
	switch value {
	case "1", "2", "3", "4", "5", "6", "7", "8", "9",
		"¹", "²", "³":
		return true
	default:
		return false
	}
}
