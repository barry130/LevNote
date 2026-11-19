package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/session"
)

func TestAppSaveAdoptedPublicationFailureReconcilesTargetAndRecovery(t *testing.T) {
	root := t.TempDir()
	source, target := filepath.Join(root, "source.txt"), filepath.Join(root, "target.go")
	if err := os.WriteFile(source, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(source)
	tab := a.Current()
	buffer, input := tab.Doc.Buffer, tab.Editor
	if err := buffer.ReplaceUTF16(0, 3, "draft"); err != nil {
		t.Fatal(err)
	}
	tab.Doc.ReadOnly, tab.Editor.ReadOnly = true, true
	version, selection, history := buffer.Version(), buffer.Selection(), buffer.History()
	view := &failingView{}
	tab.View, tab.Mode = view, "preview"
	oldGeneration := tab.OutlineFactoryGeneration
	canceled := false
	a.watches[source] = func() { canceled = true }
	a.saveDialog = func(*Tab) (string, error) { return target, nil }
	a.saveDocument = func(doc *document.Document, path string) error {
		if err := os.WriteFile(path, []byte("other"), 0600); err != nil {
			t.Fatal(err)
		}
		// Model the exact Document SaveAs verify-failure metadata outcome, not
		// success: no saved snapshot/disk receipt is changed by this host seam.
		doc.Path, doc.IsNew, doc.ReadOnly = path, false, false
		return &document.PublicationError{Path: path, MetadataAdopted: true, Cause: document.ErrConflict}
	}
	if a.Save(tab, true) {
		t.Fatal("verify failure claimed success")
	}
	if tab.Doc.Path != target || tab.Name != "target.go" || tab.Editor.ReadOnly || tab.Doc.ReadOnly || tab.Doc.IsNew ||
		!tab.Conflict || tab.Mode != "source" || tab.View != nil || view.releases != 1 || !canceled || a.watches[source] != nil ||
		tab.OutlineFactoryGeneration == oldGeneration || a.editorHost(tab).Path != target || !a.persistPending {
		t.Fatalf("adopted target left stale name/editor/view/watch/recovery: name=%q readonly=%v mode=%q released=%d canceled=%v pending=%v", tab.Name, tab.Editor.ReadOnly, tab.Mode, view.releases, canceled, a.persistPending)
	}
	if !strings.Contains(a.Message, target) || !strings.Contains(a.Message, "元数据已采用目标") || strings.Contains(a.Message, "已保存 ") ||
		!tab.Doc.Dirty() || tab.Doc.Buffer != buffer || tab.Editor != input || buffer.Version() != version ||
		buffer.Selection() != selection || buffer.History() != history || buffer.String() != "draft" {
		t.Fatal("adopted failure hid partial metadata or lost dirty draft/history")
	}
	a.Persist()
	snapshot, issues := session.Load(filepath.Join(root, "config", "session.json"))
	if len(issues) != 0 || len(snapshot.Tabs) != 1 || snapshot.Tabs[0].Path != target || snapshot.Tabs[0].Title != "target.go" ||
		snapshot.Tabs[0].Draft == nil || *snapshot.Tabs[0].Draft != "draft" {
		t.Fatalf("recovery did not retain dirty draft against actual adopted target: %+v %+v", snapshot, issues)
	}
	a.saveDocument = nil
	if a.Save(tab, false) || tab.Doc.Path != target || !tab.Doc.Dirty() {
		t.Fatal("subsequent Save falsely succeeded or returned to source metadata")
	}
}

func TestAppSaveNamesActualPublishedTargetWithoutSuccess(t *testing.T) {
	root := t.TempDir()
	source, target := filepath.Join(root, "source.txt"), filepath.Join(root, "published.txt")
	if err := os.WriteFile(source, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(source)
	tab := a.Current()
	buffer := tab.Doc.Buffer
	if err := buffer.ReplaceUTF16(0, 3, "draft"); err != nil {
		t.Fatal(err)
	}
	tab.Doc.ReadOnly, tab.Editor.ReadOnly = true, true
	version, selection, history := buffer.Version(), buffer.Selection(), buffer.History()
	a.saveDialog = func(*Tab) (string, error) { return target, nil }
	cause := &os.PathError{Op: "sync", Path: root, Err: os.ErrPermission}
	// Document public tests exercise real publication and outcome classification.
	// This host seam supplies that outcome after physically creating its target;
	// it does not fake a successful document baseline or a rendered UI notice.
	a.saveDocument = func(doc *document.Document, path string) error {
		if doc != tab.Doc || path != target {
			t.Fatal("host did not save the intended document/target")
		}
		if err := os.WriteFile(path, []byte("draft"), 0600); err != nil {
			t.Fatal(err)
		}
		return &document.PublicationError{Path: path, Cause: cause}
	}
	if a.Save(tab, true) {
		t.Fatal("partial publication claimed success")
	}
	if !strings.Contains(a.Message, "字节可能已写入目标") || !strings.Contains(a.Message, target) ||
		!strings.Contains(a.Message, "文档元数据保持原状态") || !strings.Contains(a.Message, cause.Error()) ||
		!strings.Contains(a.Message, "重试前请检查目标或另存新副本") || strings.Contains(a.Message, "已保存 ") ||
		len(a.Issues) == 0 || a.Issues[len(a.Issues)-1] != a.Message {
		t.Fatalf("visible notice hid publication/actual target/metadata/cause: %q", a.Message)
	}
	tester := ui.NewTester(a.View, 1180, 780)
	if !tester.HasText(a.Message) {
		t.Fatal("save notice was not rendered in the public app view")
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "draft" || tab.Doc.Path != source || tab.Name != "source.txt" || !tab.Doc.ReadOnly || !tab.Editor.ReadOnly ||
		!tab.Doc.Dirty() || tab.Saving || tab.Doc.Buffer != buffer || buffer.Version() != version || buffer.Selection() != selection || buffer.History() != history {
		t.Fatal("post-publication host failure changed old metadata or discarded dirty draft")
	}
	if !errors.Is(&document.PublicationError{Path: target, Cause: cause}, os.ErrPermission) {
		t.Fatal("publication outcome lost error cause")
	}
}
