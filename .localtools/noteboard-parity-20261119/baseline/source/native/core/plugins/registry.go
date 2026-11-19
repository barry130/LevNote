// Package plugins defines the ordered, data-only editor format registry.
package plugins

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"levnote.local/native/core/document"
)

const APIVersion = 1

// ErrEnabledNotLoaded means settings were saved and the registry reloaded,
// but the requested plugin could not be made active. Consumers must reconcile
// their views even though SetEnabled returns an error. Persistence failures do
// not wrap this sentinel and leave the live registry unchanged.
var ErrEnabledNotLoaded = errors.New("enabled plugin could not be loaded")

type Capabilities struct {
	Read   bool
	Edit   bool
	Source bool
	Export bool
}

type Manifest struct {
	ID          string
	Name        string
	Extensions  []string
	Version     string
	APIVersion  int
	Caps        Capabilities
	Order       int
	Declarative bool
	Enabled     bool
	Active      bool
	Builtin     bool
	SourcePath  string
}

type Match struct {
	ID       string
	Manifest Manifest
}

type Registry struct {
	manifests []Manifest
}

type Manager struct {
	registry     *Registry
	installed    []Manifest
	syntax       map[string]SyntaxManifest
	disabled     map[string]bool
	loadErrors   []PluginError
	settingsErr  *PluginError
	settingsRoot string
	installRoot  string
}

func NewRegistry() *Registry {
	return &Registry{manifests: builtinManifests()}
}

func NewManager(settingsRoot string, installRoot string) (*Manager, error) {
	if err := ensurePrivateDirectory(installRoot); err != nil {
		return nil, fmt.Errorf("prepare plugin install directory: %w", err)
	}
	manager := &Manager{
		registry: NewRegistry(), disabled: map[string]bool{},
		syntax: map[string]SyntaxManifest{}, settingsRoot: settingsRoot, installRoot: installRoot,
	}
	settings, err := LoadSettings(settingsRoot)
	if err != nil {
		// A broken preference file must not hide otherwise usable formats.
		// The UI consumes LoadErrors and can repair it by changing a setting.
		manager.settingsErr = &PluginError{
			Path: SettingsPath(settingsRoot), Message: "load plugin settings: " + err.Error(),
		}
	} else {
		manager.disabled = settings.Disabled
	}
	manager.reload()
	return manager, nil
}

func builtinManifests() []Manifest {
	return []Manifest{
		{
			ID: "markdown", Name: "Markdown",
			Extensions: []string{".md", ".markdown", ".mdown", ".mkd", ".mkdn", ".mdwn", ".mdtxt", ".mdtext"},
			Version:    "1.0.0", APIVersion: APIVersion, Caps: Capabilities{Read: true, Edit: true, Source: true, Export: true},
			Order: 1, Enabled: true, Active: true, Builtin: true,
		},
		{
			ID: "mindmap", Name: "Mind Map", Extensions: []string{".mindmap"},
			Version: "1.0.0", APIVersion: APIVersion, Caps: Capabilities{Read: true, Edit: true, Source: true},
			Order: 2, Enabled: true, Active: true, Builtin: true,
		},
		{
			ID: "drawio", Name: "draw.io Diagram", Extensions: []string{".drawio", ".dio"},
			Version: "1.0.0", APIVersion: APIVersion, Caps: Capabilities{Read: true, Edit: true, Source: true},
			Order: 3, Enabled: true, Active: true, Builtin: true,
		},
		{
			ID: "image", Name: "Native Image Reader", Extensions: []string{".png", ".jpg", ".jpeg"},
			Version: "1.0.0", APIVersion: APIVersion, Caps: Capabilities{Read: true, Edit: false, Source: true},
			Order: 4, Enabled: true, Active: true, Builtin: true,
		},
		{
			ID: "code", Name: "Code / Text",
			Extensions: []string{
				".md", ".markdown", ".mdown", ".mkd", ".mkdn", ".mdwn", ".mdtxt", ".mdtext",
				".mindmap", ".drawio", ".dio",
				".go", ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs", ".json", ".jsonc", ".py", ".pyw", ".rs", ".xml", ".svg",
				".yml", ".yaml", ".toml", ".ini", ".cfg", ".conf", ".env", ".csv", ".tsv", ".txt", ".text", ".log",
			},
			Version: "1.0.0", APIVersion: APIVersion, Caps: Capabilities{Read: true, Edit: true, Source: true, Export: true},
			Order: 100, Enabled: true, Active: true, Builtin: true,
		},
	}
}

func normalizeExtensions(values []string) ([]string, error) {
	seen := map[string]bool{}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		extension := strings.ToLower(strings.TrimSpace(value))
		if extension == "" {
			return nil, fmt.Errorf("empty extension")
		}
		if !strings.HasPrefix(extension, ".") {
			extension = "." + extension
		}
		if len(extension) < 2 || len(extension) > 32 {
			return nil, fmt.Errorf("extension %q is out of bounds", value)
		}
		if strings.ContainsAny(extension, "\\/:*?\"<>|") {
			return nil, fmt.Errorf("extension %q has invalid characters", value)
		}
		if seen[extension] {
			continue
		}
		seen[extension] = true
		normalized = append(normalized, extension)
	}
	return normalized, nil
}

func validateManifest(manifest Manifest) error {
	if err := validatePluginID(manifest.ID); err != nil {
		return err
	}
	if strings.TrimSpace(manifest.Name) == "" || len(manifest.Name) > 128 {
		return fmt.Errorf("plugin name is empty or over 128 characters")
	}
	if manifest.APIVersion != APIVersion {
		return fmt.Errorf("plugin %q has unsupported API version %d", manifest.ID, manifest.APIVersion)
	}
	if strings.TrimSpace(manifest.Version) == "" || len(manifest.Version) > 64 {
		return fmt.Errorf("plugin %q has an invalid version", manifest.ID)
	}
	extensions, err := normalizeExtensions(manifest.Extensions)
	if err != nil {
		return fmt.Errorf("plugin %q: %w", manifest.ID, err)
	}
	if len(extensions) == 0 {
		return fmt.Errorf("plugin %q has no extensions", manifest.ID)
	}
	return nil
}

func validatePluginID(id string) error {
	if id == "" || len(id) > 64 {
		return fmt.Errorf("plugin id is empty or over 64 characters")
	}
	if id[0] == '-' || id[len(id)-1] == '-' || strings.Contains(id, "--") {
		return fmt.Errorf("plugin id %q must use non-empty lowercase words separated by single hyphens", id)
	}
	for _, char := range id {
		isLower := char >= 'a' && char <= 'z'
		isNumber := char >= '0' && char <= '9'
		if !isLower && !isNumber && char != '-' {
			return fmt.Errorf("plugin id %q contains invalid characters", id)
		}
	}
	return nil
}

func (r *Registry) Register(manifest Manifest, declarative bool) error {
	if err := validateManifest(manifest); err != nil {
		return err
	}
	extensions, _ := normalizeExtensions(manifest.Extensions)
	manifest.Extensions = extensions
	manifest = cloneManifest(manifest)
	manifest.Declarative = declarative
	if manifest.Order == 0 {
		manifest.Order = len(r.manifests) + 1
	}
	for _, existing := range r.manifests {
		if existing.ID == manifest.ID {
			return fmt.Errorf("plugin id %q is already registered", manifest.ID)
		}
	}
	r.manifests = append(r.manifests, manifest)
	sort.SliceStable(r.manifests, func(i, j int) bool {
		return r.manifests[i].Order < r.manifests[j].Order
	})
	return nil
}

func (r *Registry) Match(path string) (Match, error) {
	extension := strings.ToLower(filepath.Ext(path))
	if extension == "" {
		return Match{}, fmt.Errorf("path %q has no extension", path)
	}
	for _, manifest := range r.manifests {
		if !manifest.Enabled || !manifest.Active {
			continue
		}
		for _, candidate := range manifest.Extensions {
			if candidate == extension {
				return Match{ID: manifest.ID, Manifest: cloneManifest(manifest)}, nil
			}
		}
	}
	return Match{}, fmt.Errorf("no plugin claims extension %q", extension)
}

func (r *Registry) Manifests() []Manifest {
	result := make([]Manifest, len(r.manifests))
	for i, manifest := range r.manifests {
		result[i] = cloneManifest(manifest)
	}
	return result
}

func (r *Registry) Manifest(id string) (Manifest, error) {
	for _, manifest := range r.manifests {
		if manifest.ID == id {
			return cloneManifest(manifest), nil
		}
	}
	return Manifest{}, fmt.Errorf("plugin %q is not registered", id)
}

func cloneManifest(manifest Manifest) Manifest {
	manifest.Extensions = append([]string(nil), manifest.Extensions...)
	return manifest
}

func cloneSyntax(syntax SyntaxManifest) SyntaxManifest {
	syntax.Extensions = append([]string(nil), syntax.Extensions...)
	syntax.Keywords = append([]string(nil), syntax.Keywords...)
	syntax.Outline = append([]OutlineRule(nil), syntax.Outline...)
	return syntax
}

func (m *Manager) Match(path string) (Match, error) {
	return m.registry.Match(path)
}

func (m *Manager) Manifests() []Manifest {
	manifests := builtinManifests()
	for i := range manifests {
		manifests[i].Enabled = !m.disabled[manifests[i].ID]
		manifests[i].Active = manifests[i].Enabled
		manifests[i] = cloneManifest(manifests[i])
	}
	for _, manifest := range m.installed {
		manifests = append(manifests, cloneManifest(manifest))
	}
	return manifests
}

func (m *Manager) Manifest(id string) (Manifest, error) {
	for _, manifest := range m.Manifests() {
		if manifest.ID == id {
			return manifest, nil
		}
	}
	return Manifest{}, fmt.Errorf("plugin %q is not installed", id)
}

// Enabled reports whether a registered builtin or installed plugin may match.
// Unknown IDs are disabled.
func (m *Manager) Enabled(id string) bool {
	for _, manifest := range m.Manifests() {
		if manifest.ID == id {
			return manifest.Enabled && manifest.Active
		}
	}
	return false
}

// Syntax returns the declarative syntax owned by an installed plugin. The
// returned slices are copies so callers cannot mutate manager state.
func (m *Manager) Syntax(id string) (SyntaxManifest, bool) {
	syntax, ok := m.syntax[id]
	if !ok {
		return SyntaxManifest{}, false
	}
	return cloneSyntax(syntax), true
}

func (m *Manager) SetEnabled(id string, enabled bool) error {
	manifest, err := m.Manifest(id)
	if err != nil {
		return fmt.Errorf("plugin %q is not installed", id)
	}
	nextDisabled := make(map[string]bool, len(m.disabled)+1)
	for disabledID, value := range m.disabled {
		if value && disabledID != id {
			nextDisabled[disabledID] = true
		}
	}
	if !enabled {
		nextDisabled[id] = true
	}
	if err := SaveSettings(m.settingsRoot, Settings{Disabled: nextDisabled}); err != nil {
		return fmt.Errorf("save plugin settings: %w", err)
	}
	m.disabled = nextDisabled
	m.settingsErr = nil
	m.reload()
	if enabled {
		if current, lookupErr := m.Manifest(id); lookupErr != nil || !current.Active {
			return fmt.Errorf("%w: plugin %q; inspect plugin errors", ErrEnabledNotLoaded, manifest.ID)
		}
	}
	return nil
}

// InstallFile validates and copies one declarative manifest into the install
// directory. Existing IDs and destination paths are never overwritten.
func (m *Manager) InstallFile(path string) error {
	data, syntax, err := readManifestFile(path)
	if err != nil {
		return fmt.Errorf("validate plugin manifest: %w", err)
	}
	if _, err := m.Manifest(syntax.ID); err == nil {
		return fmt.Errorf("plugin id %q is already installed", syntax.ID)
	}
	name := filepath.Base(path)
	if !strings.EqualFold(filepath.Ext(name), ".json") ||
		strings.EqualFold(name, filepath.Base(SettingsPath(""))) {
		return fmt.Errorf("plugin manifest must use a .json file name")
	}
	destination := filepath.Join(m.installRoot, name)
	if _, err := os.Lstat(destination); err == nil {
		return fmt.Errorf("plugin file %q already exists", destination)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("inspect plugin destination: %w", err)
	}
	if err := document.WriteNewFile(destination, data); err != nil {
		if errors.Is(err, document.ErrExists) {
			return fmt.Errorf("plugin file %q already exists", destination)
		}
		return fmt.Errorf("publish plugin file atomically: %w", err)
	}

	m.reload()
	loaded, loadIssue := m.installedAt(destination)
	if loadIssue != nil {
		return m.rollbackInstall(destination, fmt.Errorf("installed plugin failed to load: %s", loadIssue.Message))
	}
	if loaded.ID != syntax.ID {
		return m.rollbackInstall(destination, fmt.Errorf("installed plugin loaded as %q instead of %q", loaded.ID, syntax.ID))
	}
	if _, ok := m.Syntax(loaded.ID); !ok {
		return m.rollbackInstall(destination, fmt.Errorf("installed plugin %q has no preserved syntax definition", loaded.ID))
	}
	return nil
}

func (m *Manager) rollbackInstall(destination string, cause error) error {
	removeErr := os.Remove(destination)
	m.reload()
	if removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
		return fmt.Errorf("%w; rollback also failed to remove %q: %v", cause, destination, removeErr)
	}
	return cause
}

func (m *Manager) installedAt(path string) (Manifest, *PluginError) {
	cleanPath := filepath.Clean(path)
	for _, manifest := range m.installed {
		if filepath.Clean(manifest.SourcePath) == cleanPath {
			return cloneManifest(manifest), nil
		}
	}
	for i := range m.loadErrors {
		if filepath.Clean(m.loadErrors[i].Path) == cleanPath {
			issue := m.loadErrors[i]
			return Manifest{}, &issue
		}
	}
	return Manifest{}, &PluginError{Path: path, Message: "installed manifest was not loaded"}
}

func (m *Manager) Refresh() error {
	settings, err := LoadSettings(m.settingsRoot)
	if err != nil {
		m.settingsErr = &PluginError{
			Path: SettingsPath(m.settingsRoot), Message: "load plugin settings: " + err.Error(),
		}
	} else {
		m.disabled = settings.Disabled
		m.settingsErr = nil
	}
	m.reload()
	if len(m.loadErrors) > 0 {
		first := m.loadErrors[0]
		return fmt.Errorf("load plugins: %s: %s", first.Path, first.Message)
	}
	return nil
}

func (m *Manager) reload() {
	m.registry = NewRegistry()
	m.installed = nil
	m.syntax = map[string]SyntaxManifest{}
	installed := LoadInstallDir(m.installRoot)
	m.loadErrors = append([]PluginError(nil), installed.Errors...)
	if m.settingsErr != nil {
		m.loadErrors = append([]PluginError{*m.settingsErr}, m.loadErrors...)
	}
	for i := range m.registry.manifests {
		manifest := &m.registry.manifests[i]
		manifest.Enabled = !m.disabled[manifest.ID]
		manifest.Active = manifest.Enabled
	}
	for _, loaded := range installed.Files {
		manifest := loaded.Manifest
		manifest.SourcePath = loaded.Path
		manifest.Enabled = !m.disabled[manifest.ID]
		manifest.Active = manifest.Enabled
		if err := m.registry.Register(manifest, true); err != nil {
			manifest.Active = false
			m.loadErrors = append(m.loadErrors, PluginError{
				Path: loaded.Path, Message: err.Error(),
			})
			m.installed = append(m.installed, manifest)
			continue
		}
		m.installed = append(m.installed, manifest)
		m.syntax[manifest.ID] = cloneSyntax(loaded.Syntax)
	}
}

func (m *Manager) LoadErrors() []PluginError {
	result := make([]PluginError, len(m.loadErrors))
	copy(result, m.loadErrors)
	return result
}
