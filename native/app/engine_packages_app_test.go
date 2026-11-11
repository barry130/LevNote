package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/enginecatalog"
)

func TestEnginePackagePoisonRefusesTransitionsButLeavesInspectionAvailable(t *testing.T) {
	config := t.TempDir()
	stateRoot := filepath.Join(config, "engine-packages")
	if err := os.MkdirAll(stateRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stateRoot, "selections.json"), []byte(`{"schema":1,"selections":[`), 0o600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, config)
	a.enginePoison = os.ErrClosed

	// A poisoned runner cannot safely perform a transition: it must be a
	// visible refusal rather than a package job that is guaranteed to fail.
	a.startPackageWork(packageRepairAction, enginecatalog.MarkdownHTML, "")
	if a.packageJob != nil || !strings.Contains(a.Message, "无法切换引擎") {
		t.Fatalf("poisoned repair was admitted: job=%#v message=%q", a.packageJob, a.Message)
	}
	a.startPackageWork(packageActivateAction, enginecatalog.MarkdownHTML, "deadbeef")
	if a.packageJob != nil || !strings.Contains(a.Message, "无法切换引擎") {
		t.Fatalf("poisoned activation was admitted: job=%#v message=%q", a.packageJob, a.Message)
	}

	// List remains a diagnostic store operation and therefore stays reachable.
	a.startPackageWork(packageListAction, enginecatalog.MarkdownHTML, "")
	if a.packageJob == nil || a.packageJob.action != packageListAction {
		t.Fatal("poisoned service incorrectly blocked package inspection")
	}
	a.joinPackageWork()

	a.ShowPlugins, a.showEnginePackages = true, true
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	repair, ok := tester.AccessByRole(ui.RoleButton, "将所有引擎恢复为随包选择并保留安装包")
	if !ok || !repair.Disabled {
		t.Fatalf("poisoned repair control remained actionable: ok=%v access=%+v", ok, repair)
	}
}
