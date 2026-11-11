package plugins

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"levnote.local/native/core/document"
)

type SyntaxManifest struct {
	Schema      int
	ID          string
	Name        string
	Version     string
	APIVersion  int
	Extensions  []string
	LineComment string
	BlockStart  string
	BlockEnd    string
	Keywords    []string
	Outline     []OutlineRule
	NativeRoute NativeRoute
}

// OutlineRule is the loader-boundary DTO for schema2 source outlines. Rule
// semantics are deliberately literal and line-local; plugins cannot request a
// language parser, regex engine, or multiline lexical guarantee.
type OutlineRule struct {
	Prefix string
	Suffix string
	Level  int
}

type PluginError struct {
	Path    string
	Message string
}

type LoadResult struct {
	Manifests []Manifest
	Files     []LoadedManifest
	Errors    []PluginError
}

type LoadedManifest struct {
	Manifest Manifest
	Syntax   SyntaxManifest
	Path     string
}

const maxManifestBytes = 64 << 10
const maxSettingsBytes = 64 << 10
const maxInstalledPlugins = 256
const maxInstallDirEntries = 1024
const installedPluginOrder = 50

var manifestFieldOrder = []string{
	"schema", "id", "name", "version", "apiVersion", "extensions", "lineComment", "blockStart", "blockEnd", "keywords", "outline",
}

const (
	maxOutlineRules     = 8
	maxOutlineMarker    = 16
	minOutlineLevel     = 1
	maxOutlineLevel     = 6
	outlineSchema1Error = "schema1 does not support outline; use schema2"
)

func readManifestFile(path string) ([]byte, SyntaxManifest, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, SyntaxManifest{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, SyntaxManifest{}, errors.New("symbolic links are not accepted as plugin manifests")
	}
	if !info.Mode().IsRegular() {
		return nil, SyntaxManifest{}, errors.New("plugin manifest is not a regular file")
	}
	if info.Size() > maxManifestBytes {
		return nil, SyntaxManifest{}, fmt.Errorf("manifest is larger than %d bytes", maxManifestBytes)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, SyntaxManifest{}, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxManifestBytes+1))
	if err != nil {
		return nil, SyntaxManifest{}, err
	}
	if len(data) > maxManifestBytes {
		return nil, SyntaxManifest{}, fmt.Errorf("manifest is larger than %d bytes", maxManifestBytes)
	}
	syntax, err := decodeManifest(data)
	if err != nil {
		return nil, SyntaxManifest{}, err
	}
	return data, syntax, nil
}

func loadManifestFile(path string) (SyntaxManifest, error) {
	_, syntax, err := readManifestFile(path)
	return syntax, err
}

func decodeManifest(data []byte) (SyntaxManifest, error) {
	if len(data) > maxManifestBytes {
		return SyntaxManifest{}, fmt.Errorf("manifest is larger than %d bytes", maxManifestBytes)
	}
	if !utf8.Valid(data) {
		return SyntaxManifest{}, errors.New("manifest is not valid UTF-8")
	}
	fields, strict, err := discoverManifestFields(data)
	if err != nil {
		return SyntaxManifest{}, err
	}
	if strict {
		return decodeNativeRouteManifest(data, fields)
	}
	var schemaEnvelope struct {
		Schema *int `json:"schema"`
	}
	if err := json.Unmarshal(data, &schemaEnvelope); err != nil {
		return SyntaxManifest{}, err
	}
	if schemaEnvelope.Schema == nil {
		return SyntaxManifest{}, errors.New("schema is required")
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(data, &envelope); err != nil {
		return SyntaxManifest{}, err
	}
	for key := range envelope {
		allowed := false
		for _, field := range manifestFieldOrder {
			if key == field {
				allowed = true
				break
			}
		}
		if !allowed {
			return SyntaxManifest{}, fmt.Errorf("unknown field %q", key)
		}
	}
	if schemaEnvelope.Schema != nil && *schemaEnvelope.Schema == 1 {
		if _, exists := envelope["outline"]; exists {
			return SyntaxManifest{}, errors.New(outlineSchema1Error)
		}
	}

	var manifest SyntaxManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return SyntaxManifest{}, err
	}
	if *schemaEnvelope.Schema == 2 {
		if raw, exists := envelope["outline"]; exists {
			if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
				return SyntaxManifest{}, errors.New("outline must be an array; omit it for no capability")
			}
			var rawRules []json.RawMessage
			if err := json.Unmarshal(raw, &rawRules); err != nil {
				return SyntaxManifest{}, fmt.Errorf("outline: %w", err)
			}
			rules := make([]OutlineRule, 0, len(rawRules))
			for i, rawRule := range rawRules {
				if err := validateStrictObject(rawRule, []string{"prefix", "suffix", "level"}); err != nil {
					return SyntaxManifest{}, fmt.Errorf("outline rule %d: %w", i, err)
				}
				var ruleFields map[string]json.RawMessage
				if err := json.Unmarshal(rawRule, &ruleFields); err != nil {
					return SyntaxManifest{}, fmt.Errorf("outline rule %d: %w", i, err)
				}
				rawPrefix, prefixPresent := ruleFields["prefix"]
				if !prefixPresent {
					return SyntaxManifest{}, fmt.Errorf("outline rule %d: prefix is required", i)
				}
				var prefix string
				if err := decodeStrictJSONString(rawPrefix, "prefix", i); err != nil {
					return SyntaxManifest{}, err
				}
				if err := json.Unmarshal(rawPrefix, &prefix); err != nil {
					return SyntaxManifest{}, fmt.Errorf("outline rule %d: prefix: %w", i, err)
				}
				rawLevel, levelPresent := ruleFields["level"]
				if !levelPresent {
					return SyntaxManifest{}, fmt.Errorf("outline rule %d: level is required", i)
				}
				var level int
				if err := json.Unmarshal(rawLevel, &level); err != nil {
					return SyntaxManifest{}, fmt.Errorf("outline rule %d: level: %w", i, err)
				}
				suffix := ""
				if rawSuffix, present := ruleFields["suffix"]; present {
					if err := decodeStrictJSONString(rawSuffix, "suffix", i); err != nil {
						return SyntaxManifest{}, err
					}
					if err := json.Unmarshal(rawSuffix, &suffix); err != nil {
						return SyntaxManifest{}, fmt.Errorf("outline rule %d: suffix: %w", i, err)
					}
				}
				rules = append(rules, OutlineRule{Prefix: prefix, Suffix: suffix, Level: level})
			}
			manifest.Outline = rules
		} else {
			manifest.Outline = nil
		}
	}
	if manifest.APIVersion == 0 {
		manifest.APIVersion = APIVersion
	}
	if manifest.Version == "" {
		manifest.Version = "1.0.0"
	}
	if err := validateSyntax(manifest); err != nil {
		return SyntaxManifest{}, err
	}
	return manifest, nil
}

func validateStrictObject(data []byte, allowedFields []string) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for field := range fields {
		found := false
		for _, allowed := range allowedFields {
			if field == allowed {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unknown field %q", field)
		}
	}
	return nil
}

func decodeStrictJSONString(data []byte, field string, index int) error {
	if len(data) == 0 || strings.HasPrefix(strings.TrimSpace(string(data)), "null") {
		return fmt.Errorf("outline rule %d: %s must not be null", index, field)
	}
	if hasIllFormedSurrogateEscape(data) {
		return fmt.Errorf("outline rule %d: %s contains an ill-formed Unicode surrogate escape", index, field)
	}
	return nil
}

func hasIllFormedSurrogateEscape(data []byte) bool {
	for i := 0; i < len(data); {
		if data[i] != '"' {
			i++
			continue
		}
		i++
		for i < len(data) {
			if data[i] != '\\' {
				if data[i] == '"' {
					i++
					break
				}
				i++
				continue
			}
			if i+1 >= len(data) {
				return true
			}
			switch data[i+1] {
			case '\\', '"', '/':
				i += 2
			case 'b', 'f', 'n', 'r', 't':
				i += 2
			case 'u':
				if i+6 > len(data) {
					return true
				}
				value, err := parseHex4(data[i+2 : i+6])
				if err != nil {
					return true
				}
				i += 6
				if value >= 0xdc00 && value <= 0xdfff {
					return true
				}
				if value >= 0xd800 && value <= 0xdbff {
					if i+6 > len(data) || data[i] != '\\' || data[i+1] != 'u' {
						return true
					}
					low, err := parseHex4(data[i+2 : i+6])
					if err != nil || low < 0xdc00 || low > 0xdfff {
						return true
					}
					i += 6
				}
			default:
				return true
			}
		}
	}
	return false
}

func parseHex4(data []byte) (int, error) {
	value := 0
	for _, char := range data {
		digit := -1
		switch {
		case char >= '0' && char <= '9':
			digit = int(char - '0')
		case char >= 'a' && char <= 'f':
			digit = int(char-'a') + 10
		case char >= 'A' && char <= 'F':
			digit = int(char-'A') + 10
		}
		if digit < 0 {
			return 0, errors.New("invalid hexadecimal digit")
		}
		value = value*16 + digit
	}
	return value, nil
}

func validateSyntax(manifest SyntaxManifest) error {
	if manifest.Schema != 1 && manifest.Schema != 2 && manifest.Schema != 3 {
		return fmt.Errorf("unsupported schema %d", manifest.Schema)
	}
	if manifest.Schema == 3 {
		if manifest.NativeRoute != NativeRouteMarkdown &&
			manifest.NativeRoute != NativeRouteMindmap &&
			manifest.NativeRoute != NativeRouteDrawio {
			return fmt.Errorf("unsupported native route %q", manifest.NativeRoute)
		}
		if len(manifest.Outline) != 0 {
			return errors.New("schema3 does not support outline rules; the native adapter supplies any outline capability")
		}
	} else if manifest.NativeRoute != "" {
		return errors.New("nativeRoute requires schema3")
	}
	base := Manifest{
		ID: manifest.ID, Name: manifest.Name, Extensions: manifest.Extensions,
		Version: manifest.Version, APIVersion: manifest.APIVersion,
	}
	if err := validateManifest(base); err != nil {
		return err
	}
	if len(manifest.LineComment) > 8 {
		return fmt.Errorf("line comment is over 8 bytes")
	}
	if len(manifest.BlockStart) > 8 || len(manifest.BlockEnd) > 8 {
		return fmt.Errorf("block markers are over 8 bytes")
	}
	if (manifest.BlockStart == "") != (manifest.BlockEnd == "") {
		return errors.New("block markers must both be empty or both be set")
	}
	if len(manifest.Keywords) > 512 {
		return fmt.Errorf("keyword list has more than 512 items")
	}
	seen := map[string]bool{}
	for _, keyword := range manifest.Keywords {
		if keyword == "" || len(keyword) > 64 {
			return fmt.Errorf("keyword %q is empty or over 64 bytes", keyword)
		}
		if seen[keyword] {
			continue
		}
		seen[keyword] = true
	}
	return validateOutline(manifest.Schema, manifest.Outline)
}

func validateOutline(schema int, rules []OutlineRule) error {
	if schema == 1 && len(rules) > 0 {
		return errors.New(outlineSchema1Error)
	}
	if len(rules) > maxOutlineRules {
		return fmt.Errorf("outline has more than %d rules", maxOutlineRules)
	}
	seen := make(map[[2]string]struct{}, len(rules))
	for i, rule := range rules {
		if len(rule.Prefix) == 0 || len(rule.Prefix) > maxOutlineMarker {
			return fmt.Errorf("outline rule %d prefix is empty or over %d bytes", i, maxOutlineMarker)
		}
		if len(rule.Suffix) > maxOutlineMarker {
			return fmt.Errorf("outline rule %d suffix is over %d bytes", i, maxOutlineMarker)
		}
		if err := validateUTF8Marker(rule.Prefix, "prefix"); err != nil {
			return fmt.Errorf("outline rule %d: %w", i, err)
		}
		if err := validateUTF8Marker(rule.Suffix, "suffix"); err != nil {
			return fmt.Errorf("outline rule %d: %w", i, err)
		}
		if rule.Level < minOutlineLevel || rule.Level > maxOutlineLevel {
			return fmt.Errorf("outline rule %d level must be between %d and %d", i, minOutlineLevel, maxOutlineLevel)
		}
		signature := [2]string{rule.Prefix, rule.Suffix}
		if _, exists := seen[signature]; exists {
			return fmt.Errorf("outline rule %d duplicates prefix and suffix markers", i)
		}
		seen[signature] = struct{}{}
	}
	return nil
}

func validateUTF8Marker(value, field string) error {
	if !utf8.ValidString(value) {
		return fmt.Errorf("%s is not valid UTF-8", field)
	}
	for _, char := range value {
		if char < 0x20 || char == 0x7f {
			return fmt.Errorf("%s contains a control byte", field)
		}
	}
	return nil
}

func manifestOf(syntax SyntaxManifest, order int) Manifest {
	// Syntax preserves the installed spelling; matching and public metadata use
	// the same normalization. This is called only after validateSyntax succeeds.
	extensions, _ := normalizeExtensions(syntax.Extensions)
	return Manifest{
		ID: syntax.ID, Name: syntax.Name, Extensions: extensions,
		Version: syntax.Version, APIVersion: syntax.APIVersion, Order: order,
		Caps: Capabilities{Read: true, Edit: true, Source: true}, Declarative: true,
	}
}

func LoadInstallDir(root string) LoadResult {
	var result LoadResult
	info, statErr := os.Lstat(root)
	if statErr != nil {
		if errors.Is(statErr, fs.ErrNotExist) {
			return result
		}
		result.Errors = append(result.Errors, PluginError{Path: root, Message: statErr.Error()})
		return result
	}
	if info.Mode()&os.ModeSymlink != 0 {
		result.Errors = append(result.Errors, PluginError{Path: root, Message: "plugin directory symbolic links are not accepted"})
		return result
	}
	if !info.IsDir() {
		result.Errors = append(result.Errors, PluginError{Path: root, Message: "plugin directory is not a directory"})
		return result
	}
	dir, err := os.Open(root)
	if err != nil {
		result.Errors = append(result.Errors, PluginError{Path: root, Message: err.Error()})
		return result
	}
	defer dir.Close()
	entries, readErr := dir.ReadDir(maxInstallDirEntries + 1)
	if len(entries) > maxInstallDirEntries {
		entries = entries[:maxInstallDirEntries]
		result.Errors = append(result.Errors, PluginError{
			Path: root, Message: fmt.Sprintf(
				"directory listing truncated at %d entries; additional entries were not inspected",
				maxInstallDirEntries,
			),
		})
	}
	if readErr != nil && !errors.Is(readErr, io.EOF) {
		result.Errors = append(result.Errors, PluginError{Path: root, Message: readErr.Error()})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})
	seenManifests := 0
	for _, entry := range entries {
		if !strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			continue
		}
		if strings.EqualFold(entry.Name(), filepath.Base(SettingsPath(""))) {
			continue
		}
		if seenManifests >= maxInstalledPlugins {
			result.Errors = append(result.Errors, PluginError{
				Path: root, Message: fmt.Sprintf("more than %d plugin manifests", maxInstalledPlugins),
			})
			break
		}
		seenManifests++
		path := filepath.Join(root, entry.Name())
		info, infoErr := os.Lstat(path)
		if infoErr != nil {
			result.Errors = append(result.Errors, PluginError{Path: path, Message: infoErr.Error()})
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			result.Errors = append(result.Errors, PluginError{Path: path, Message: "symbolic links are not accepted as plugin manifests"})
			continue
		}
		if !info.Mode().IsRegular() {
			result.Errors = append(result.Errors, PluginError{Path: path, Message: "plugin manifest is not a regular file"})
			continue
		}
		syntax, err := loadManifestFile(path)
		if err != nil {
			result.Errors = append(result.Errors, PluginError{Path: path, Message: err.Error()})
			continue
		}
		manifest := manifestOf(syntax, installedPluginOrder)
		result.Manifests = append(result.Manifests, manifest)
		result.Files = append(result.Files, LoadedManifest{Manifest: manifest, Syntax: syntax, Path: path})
	}
	return result
}

type Settings struct {
	Disabled map[string]bool
}

func SettingsPath(root string) string { return filepath.Join(root, "plugins.json") }

func LoadSettings(root string) (Settings, error) {
	path := SettingsPath(root)
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Settings{Disabled: map[string]bool{}}, nil
		}
		return Settings{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return Settings{}, errors.New("plugin settings symbolic links are not accepted")
	}
	if !info.Mode().IsRegular() {
		return Settings{}, errors.New("plugin settings path is not a regular file")
	}
	if info.Size() > maxSettingsBytes {
		return Settings{}, fmt.Errorf("plugin settings are larger than %d bytes", maxSettingsBytes)
	}
	file, err := os.Open(path)
	if err != nil {
		return Settings{}, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxSettingsBytes+1))
	if err != nil {
		return Settings{}, err
	}
	if len(data) > maxSettingsBytes {
		return Settings{}, fmt.Errorf("plugin settings are larger than %d bytes", maxSettingsBytes)
	}
	if !utf8.Valid(data) {
		return Settings{}, errors.New("plugin settings are not valid UTF-8")
	}
	var envelope struct {
		Disabled []string `json:"disabled"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return Settings{}, err
	}
	disabled := map[string]bool{}
	for _, id := range envelope.Disabled {
		if id == "" {
			continue
		}
		if err := validatePluginID(id); err != nil {
			return Settings{}, err
		}
		disabled[id] = true
	}
	return Settings{Disabled: disabled}, nil
}

func SaveSettings(root string, settings Settings) error {
	if err := ensurePrivateDirectory(root); err != nil {
		return err
	}
	disabled := make([]string, 0, len(settings.Disabled))
	for id, value := range settings.Disabled {
		if value {
			disabled = append(disabled, id)
		}
	}
	sortStrings(disabled)
	data, err := json.MarshalIndent(map[string][]string{"disabled": disabled}, "", "  ")
	if err != nil {
		return err
	}
	return document.WritePrivateState(SettingsPath(root), data)
}

func ensurePrivateDirectory(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		if err := os.MkdirAll(path, 0o700); err != nil {
			return err
		}
		info, err = os.Lstat(path)
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("configuration directory %q must not be a symbolic link", path)
	}
	if !info.IsDir() {
		return fmt.Errorf("configuration path %q is not a directory", path)
	}
	return nil
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}
