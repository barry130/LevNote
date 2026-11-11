//go:build windows

package app

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/enginecatalog"
)

func TestEnginePackagePanelCaptures(t *testing.T) {
	output := os.Getenv("LEVNOTE_ENGINE_PACKAGE_SCREENSHOT_DIR")
	if output == "" {
		t.Skip("set LEVNOTE_ENGINE_PACKAGE_SCREENSHOT_DIR for engine panel visual acceptance")
	}
	for _, dark := range []bool{false, true} {
		t.Run(boolLabel(dark), func(t *testing.T) {
			release := enginePackageFixture(t, enginecatalog.MarkdownHTML, "app.panel.installed")
			config := t.TempDir()
			a := newTestApp(t, config)
			runEnginePackageJob(t, a, packageImportAction, enginecatalog.MarkdownHTML, release)
			receipt := enginePackageCachedReceipt(t, a, enginecatalog.MarkdownHTML, "app.panel.installed")
			runEnginePackageJob(t, a, packageActivateAction, enginecatalog.MarkdownHTML, receipt.Digest)
			a.packageOutcome = "操作完成"
			a.ShowPlugins, a.showEnginePackages = true, true
			a.Dark = dark
			tester := ui.NewTester(a.View, 760, 480)
			tester.SetDark(dark)
			tester.Frame()

			requireVisibleButton(t, tester, "恢复"+packageEngineLabel(enginecatalog.MarkdownHTML)+"的随包版本", 760, 480)
			requireVisibleButton(t, tester, "启用"+packageEngineLabel(enginecatalog.MarkdownHTML)+" "+receipt.Version+" "+shortPackageDigest(receipt.Digest), 760, 480)
			tester.Scroll(380, 430, 0, 120)
			tester.Frame()
			requireVisibleButton(t, tester, "检查已安装的"+packageEngineLabel(enginecatalog.MarkdownHTML), 760, 480)
			tester.Scroll(380, 60, 0, -120)
			tester.Frame()
			if !tester.HasText("当前选择：已安装 "+receipt.Version) || !tester.HasText("操作完成") {
				t.Fatalf("final selection and status were not visible: %q", tester.Texts())
			}

			if err := os.MkdirAll(output, 0755); err != nil {
				t.Fatal(err)
			}
			captureAcceptanceTester(t, filepath.Join(output, "engine-panel-installed-"+boolLabel(dark)+"-760x480.png"), tester)

			controlled := newControlledExporter()
			entered := make(chan struct{}, 1)
			closeRelease := make(chan struct{})
			var releaseOnce sync.Once
			releaseGate := func() { releaseOnce.Do(func() { close(closeRelease) }) }
			t.Cleanup(releaseGate)
			controlled.onClose = func() {
				entered <- struct{}{}
				<-closeRelease
			}
			a.exporter = controlled
			a.startPackageWork(packageResetAction, enginecatalog.MarkdownHTML, "")
			job := a.packageJob
			if job == nil {
				t.Fatalf("controlled reset did not start: %q", a.Message)
			}
			timer := time.NewTimer(enginePackageTestWait)
			select {
			case <-entered:
				if !timer.Stop() {
					<-timer.C
				}
			case <-timer.C:
				releaseGate()
				t.Fatal("controlled reset did not enter its cleanup gate")
			}

			a.ShowPlugins = false
			tester.Frame()
			requireVisibleButton(t, tester, "取消当前引擎包操作并等待清理", 760, 480)
			requireVisibleButton(t, tester, "查看当前引擎包操作", 760, 480)
			if !tester.HasText("Markdown HTML 导出 · 恢复随包引擎") {
				t.Fatalf("hidden-panel pending reset was not globally visible: %q", tester.Texts())
			}
			captureAcceptanceTester(t, filepath.Join(output, "engine-panel-pending-hidden-"+boolLabel(dark)+"-760x480.png"), tester)

			if err := tester.Click("查看操作"); err != nil {
				t.Fatalf("hidden-panel reopen action was not clickable: %v", err)
			}
			tester.Frame()
			if !a.ShowPlugins || !a.showEnginePackages {
				t.Fatal("pending operation did not reopen the engine panel")
			}
			if !tester.HasText("正在恢复随包引擎…") || !tester.HasText("操作正在后台完成；关闭面板不会停止它。") {
				t.Fatalf("reopened panel did not retain the pending reset: %q", tester.Texts())
			}
			releaseGate()
			awaitPackageDone(t, job)
			a.deliverPackageWork()
		})
	}
}

func requireVisibleButton(t *testing.T, tester *ui.Tester, name string, width, height int) {
	t.Helper()
	button, ok := tester.AccessByRole(ui.RoleButton, name)
	if !ok {
		t.Fatalf("button %q is missing", name)
	}
	rect := button.Bounds
	if button.Offscreen || rect.W <= 0 || rect.H <= 0 ||
		rect.X < 0 || rect.Y < 0 || rect.X+rect.W > float32(width) || rect.Y+rect.H > float32(height) {
		t.Fatalf("button %q is not fully visible in %dx%d: %+v", name, width, height, button)
	}
}

func boolLabel(value bool) string {
	if value {
		return "dark"
	}
	return "light"
}
