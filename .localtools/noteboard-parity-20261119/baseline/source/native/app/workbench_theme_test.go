package app

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/session"
)

func TestWorkbenchThemeRestoreAndPersist(t *testing.T) {
	for _, tc := range []struct {
		name     string
		stored   string
		wantDark bool
	}{
		{name: "new_config", wantDark: true},
		{name: "no_preferences", stored: `{"version":1,"active":0,"tabs":[]}`, wantDark: true},
		{name: "no_theme", stored: `{"version":1,"active":0,"tabs":[],"preferences":{"fontSize":19}}`, wantDark: true},
		{name: "empty_theme", stored: `{"version":1,"active":0,"tabs":[],"preferences":{"theme":""}}`, wantDark: true},
		{name: "system_default", stored: `{"version":1,"active":0,"tabs":[],"preferences":{"theme":"system"}}`, wantDark: true},
		{name: "unknown_theme", stored: `{"version":1,"active":0,"tabs":[],"preferences":{"theme":"unknown"}}`, wantDark: true},
		{name: "saved_light", stored: `{"version":1,"active":0,"tabs":[],"preferences":{"theme":"light"}}`},
		{name: "saved_dark", stored: `{"version":1,"active":0,"tabs":[],"preferences":{"theme":"dark"}}`, wantDark: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			configDir := t.TempDir()
			path := filepath.Join(configDir, "session.json")
			if tc.stored != "" {
				if err := os.WriteFile(path, []byte(tc.stored), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			a := newTestApp(t, configDir)
			if !a.Dark {
				t.Fatal("new applications must start with the dark workbench")
			}
			a.Restore()
			if a.Dark != tc.wantDark {
				t.Fatalf("Restore dark = %v, want %v", a.Dark, tc.wantDark)
			}
			if tc.stored != "" {
				after, err := os.ReadFile(path)
				if err != nil || string(after) != tc.stored {
					t.Fatalf("Restore rewrote the saved preference: %q, %v", after, err)
				}
			}
			a.Persist()
			if a.lastPersistError != "" {
				t.Fatal(a.lastPersistError)
			}
			saved, issues := session.Load(path)
			want := "light"
			if tc.wantDark {
				want = "dark"
			}
			if len(issues) != 0 || saved.Preferences.Theme != want {
				t.Fatalf("persisted theme = %q, want %q; issues = %v", saved.Preferences.Theme, want, issues)
			}
			if tc.name == "no_theme" && saved.Preferences.FontSize != 19 {
				t.Fatalf("theme default changed the saved editor font size: %v", saved.Preferences.FontSize)
			}
			a.Shutdown()
			reopened := newTestApp(t, configDir)
			reopened.Restore()
			if reopened.Dark != tc.wantDark {
				t.Fatalf("reopened dark = %v, want %v", reopened.Dark, tc.wantDark)
			}
		})
	}
}

func TestWorkbenchThemeSelectionPersists(t *testing.T) {
	configDir := t.TempDir()
	a := newTestApp(t, configDir)
	a.Restore()
	for _, dark := range []bool{false, true} {
		a.Dark = dark
		a.Persist()
		if a.lastPersistError != "" {
			t.Fatal(a.lastPersistError)
		}
		a.Shutdown()
		a = newTestApp(t, configDir)
		a.Restore()
		if a.Dark != dark {
			t.Fatalf("explicit choice dark = %v became %v after restart", dark, a.Dark)
		}
	}
}

func TestWorkbenchThemeReadableColors(t *testing.T) {
	for _, dark := range []bool{false, true} {
		name := "light"
		if dark {
			name = "dark"
		}
		t.Run(name, func(t *testing.T) {
			style := workbenchAppearance(dark)
			theme := style.Theme
			check := func(name string, foreground, background ui.Color) {
				t.Helper()
				if ratio := workbenchContrast(foreground, background); ratio < 4.5 {
					t.Errorf("%s contrast = %.3f, want at least 4.5", name, ratio)
				}
			}
			for name, background := range map[string]ui.Color{
				"editor": theme.Background, "control": theme.Surface,
				"hover": theme.SurfaceHover, "pressed": theme.SurfacePressed,
				"chrome": style.Chrome, "sidebar": style.Sidebar,
				"activity": style.Activity, "tab_bar": style.TabBar, "active_tab": style.TabActive,
			} {
				check(name+"/text", theme.Text, background)
				check(name+"/muted", theme.TextMuted, background)
				check(name+"/subtle", style.Subtle, background)
				check(name+"/focus", theme.Focus, background)
				check(name+"/danger", theme.Danger, background)
				check(name+"/warning", theme.Warning, background)
				check(name+"/success", theme.Success, background)
			}
			for name, background := range map[string]ui.Color{
				"accent": theme.Accent, "accent_hover": theme.AccentHover,
				"accent_pressed": theme.AccentPressed, "inactive_accent": theme.InactiveAccent,
			} {
				check(name, theme.AccentText, background)
			}
			check("status", style.StatusText, style.Status)
			check("selected_text", theme.Text, theme.Selection)
			check("tooltip", theme.InverseText, theme.Inverse)
			// Existing syntax colors must remain readable on the actual editor base.
			colors := syntaxColors(dark)
			for name, foreground := range map[string]ui.Color{
				"keyword": colors.keyword, "string": colors.stringValue,
				"number": colors.number, "comment": colors.comment, "tag": colors.tag,
			} {
				check("syntax/"+name, foreground, theme.Background)
			}
		})
	}
}

func workbenchContrast(foreground, background ui.Color) float64 {
	linear := func(value float64) float64 {
		value /= 255
		if value <= 0.04045 {
			return value / 12.92
		}
		return math.Pow((value+0.055)/1.055, 2.4)
	}
	luminance := func(r, g, b float64) float64 {
		return 0.2126*linear(r) + 0.7152*linear(g) + 0.0722*linear(b)
	}
	alpha := float64(foreground.A) / 255
	blend := func(fg, bg uint8) float64 {
		return float64(fg)*alpha + float64(bg)*(1-alpha)
	}
	fg := luminance(blend(foreground.R, background.R), blend(foreground.G, background.G), blend(foreground.B, background.B))
	bg := luminance(float64(background.R), float64(background.G), float64(background.B))
	return (math.Max(fg, bg) + 0.05) / (math.Min(fg, bg) + 0.05)
}
