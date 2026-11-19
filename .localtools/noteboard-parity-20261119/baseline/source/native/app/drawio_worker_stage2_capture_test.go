//go:build windows

package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/enginecatalog"
)

func TestDrawioWorkerStage2Captures(t *testing.T) {
	output := os.Getenv("LEVNOTE_DRAWIO_STAGE2_SCREENSHOT_DIR")
	if output == "" {
		t.Skip("set LEVNOTE_DRAWIO_STAGE2_SCREENSHOT_DIR for draw.io stage-2 visual acceptance")
	}
	if err := os.MkdirAll(output, 0755); err != nil {
		t.Fatal(err)
	}

	t.Run("source-export-guide", func(t *testing.T) {
		a, tab, _, _, _, _ := drawioWorkerApp(t)
		if !a.setMode(tab, "source") {
			t.Fatal(a.Message)
		}
		tester := ui.NewTester(a.View, 760, 480)
		tester.Frame()
		requireVisibleButton(t, tester, "选择页面后导出", 760, 480)
		button, ok := tester.AccessByRole(ui.RoleButton, "选择页面后导出")
		if !ok || !button.Disabled || button.Description != "图纸导出需要阅读或分屏视图；请切换到阅读或分屏并选择页面" {
			t.Fatalf("source-mode export guidance = ok:%v %+v", ok, button)
		}
		captureAcceptanceTester(t, filepath.Join(output, "drawio-source-export-guide-760x480.png"), tester)
	})

	t.Run("fourth-engine-panel", func(t *testing.T) {
		a := newTestApp(t, t.TempDir())
		a.ShowPlugins, a.showEnginePackages, a.Dark = true, true, true
		tester := ui.NewTester(a.View, 760, 480)
		tester.SetDark(true)
		tester.Frame()
		requireVisibleButton(t, tester, "选择图纸 SVG 导出", 760, 480)
		if err := tester.Click("选择图纸 SVG 导出"); err != nil {
			t.Fatalf("fourth engine selector was not clickable: %v", err)
		}
		tester.Frame()
		if a.packageEngine != enginecatalog.DrawioSVG || !tester.HasText("图纸 SVG 导出") {
			t.Fatalf("fourth engine selector state = %s, texts=%q", a.packageEngine, tester.Texts())
		}
		requireVisibleButton(t, tester, "恢复图纸 SVG 导出的随包版本", 760, 480)
		captureAcceptanceTester(t, filepath.Join(output, "drawio-fourth-engine-panel-dark-760x480.png"), tester)
	})
}
