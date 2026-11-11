package app

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"levnote.local/native/ui/editor"
)

func TestResolveLocalReferenceClassification(t *testing.T) {
	base := filepath.Join(t.TempDir(), "来源 文件.md")
	parent := filepath.Dir(base)
	fileURL := (&url.URL{
		Scheme: "file",
		Path:   "/" + filepath.ToSlash(filepath.Join(parent, "子 目录", "目标 %20.txt")),
	}).String()

	tests := []struct {
		name      string
		source    string
		reference string
		kind      referenceKind
		path      string
		anchor    string
		wantError bool
	}{
		{name: "relative Unicode and spaces", source: base, reference: "子 目录/目标 文件.txt", kind: referenceLocal, path: filepath.Join(parent, "子 目录", "目标 文件.txt")},
		{name: "parent path", source: base, reference: "../共享/附件.txt", kind: referenceLocal, path: filepath.Clean(filepath.Join(parent, "..", "共享", "附件.txt"))},
		{name: "absolute drive slash", source: base, reference: `C:/资料/目标.txt`, kind: referenceLocal, path: `C:\资料\目标.txt`},
		{name: "absolute drive backslash", source: base, reference: `C:\资料\目标.txt`, kind: referenceLocal, path: `C:\资料\目标.txt`},
		{name: "single rooted local path", source: base, reference: `\资料\目标.txt`, kind: referenceRejected, wantError: true},
		{name: "local file URL decoded once", source: base, reference: fileURL, kind: referenceLocal, path: filepath.Join(parent, "子 目录", "目标 %20.txt")},
		{name: "anchor", reference: "#%E6%A0%87%E9%A2%98", kind: referenceAnchor, anchor: "标题"},
		{name: "empty anchor", reference: "#", kind: referenceRejected, wantError: true},
		{name: "unsaved relative", reference: "目标.txt", kind: referenceRejected, wantError: true},
		{name: "network URL", source: base, reference: "https://example.com/a.txt", kind: referenceRejected, wantError: true},
		{name: "protocol relative", source: base, reference: "//example.com/a.txt", kind: referenceRejected, wantError: true},
		{name: "UNC path", source: base, reference: `\\server\share\a.txt`, kind: referenceRejected, wantError: true},
		{name: "remote file host", source: base, reference: "file://server/share/a.txt", kind: referenceRejected, wantError: true},
		{name: "encoded UNC file URL", source: base, reference: "file:///%5C%5Cserver/share/a.txt", kind: referenceRejected, wantError: true},
		{name: "double encoded UNC remains a literal path", source: base, reference: "file:///C:/%255C%255Cserver/a.txt", kind: referenceLocal, path: `C:\%5C%5Cserver\a.txt`},
		{name: "mailto", source: base, reference: "mailto:user@example.com", kind: referenceRejected, wantError: true},
		{name: "javascript", source: base, reference: "javascript:alert(1)", kind: referenceRejected, wantError: true},
		{name: "data", source: base, reference: "data:text/plain,hello", kind: referenceRejected, wantError: true},
		{name: "drive relative", source: base, reference: `C:目标.txt`, kind: referenceRejected, wantError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolved := resolveLocalReference(test.source, test.reference)
			if resolved.kind != test.kind {
				t.Fatalf("kind = %v, want %v; resolution=%#v", resolved.kind, test.kind, resolved)
			}
			if test.path != "" && resolved.path != test.path {
				t.Fatalf("path = %q, want %q", resolved.path, test.path)
			}
			if resolved.anchor != test.anchor {
				t.Fatalf("anchor = %q, want %q", resolved.anchor, test.anchor)
			}
			if test.wantError != (resolved.reason != "") {
				t.Fatalf("reason = %q, wantError=%v", resolved.reason, test.wantError)
			}
		})
	}
}

func TestEditorHostOpenReferenceUsesSavedSourceAndExistingBarriers(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "子 目录")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	sourcePath := filepath.Join(root, "来源.md")
	targetPath := filepath.Join(subdir, "目标 文件.txt")
	if err := os.WriteFile(sourcePath, []byte("# 来源\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetPath, []byte("目标内容"), 0600); err != nil {
		t.Fatal(err)
	}

	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(sourcePath)
	source := a.Current()
	if source == nil || source.Doc.Path != sourcePath {
		t.Fatalf("source tab = %#v", source)
	}
	a.editorHost(source).OpenReference("子 目录/目标 文件.txt")
	if len(a.Tabs) != 2 || a.Current() == nil || a.Current().Doc.Path != targetPath {
		t.Fatalf("local reference did not open through ordered barrier: tabs=%d current=%#v", len(a.Tabs), a.Current())
	}
}

func TestEditorHostOpenReferenceRejectsUnsavedExternalAndRemoteFile(t *testing.T) {
	root := t.TempDir()
	a := newTestApp(t, filepath.Join(root, "config"))
	a.NewDocument("text")
	unsaved := a.Current()
	a.editorHost(unsaved).OpenReference("relative.txt")
	if len(a.Tabs) != 1 || !strings.Contains(a.Message, "尚未保存") {
		t.Fatalf("unsaved reference = tabs %d message %q", len(a.Tabs), a.Message)
	}

	sourcePath := filepath.Join(root, "source.txt")
	if err := os.WriteFile(sourcePath, []byte("source"), 0600); err != nil {
		t.Fatal(err)
	}
	targetPath := filepath.Join(root, "must-not-open.txt")
	if err := os.WriteFile(targetPath, []byte("target"), 0600); err != nil {
		t.Fatal(err)
	}
	a.Open(sourcePath)
	before := len(a.Tabs)
	source := a.Current()
	a.editorHost(source).OpenReference("https://example.com/must-not-open.txt")
	if len(a.Tabs) != before || !strings.Contains(a.Message, "不会打开网络地址") {
		t.Fatalf("external URL = tabs %d message %q", len(a.Tabs), a.Message)
	}
	a.editorHost(source).OpenReference("file://server/share/must-not-open.txt")
	if len(a.Tabs) != before || !strings.Contains(a.Message, "主机名") {
		t.Fatalf("remote file URL = tabs %d message %q", len(a.Tabs), a.Message)
	}
}

func TestEditorHostAnchorUsesOutlineSourceResolverAfterFlush(t *testing.T) {
	root := t.TempDir()
	a := newTestApp(t, filepath.Join(root, "config"))
	a.NewDocument("markdown")
	tab := a.Current()
	tab.Mode = "source"
	source := &outlineSourceStub{}
	source.resolveAnchorFn = func(anchor string) (editor.OutlineItem, error) {
		if anchor == "目标" {
			return editor.OutlineItem{ID: "exact-heading-id", Text: "目标", Level: 1, Offset: 0, AnchorID: anchor}, nil
		}
		return editor.OutlineItem{}, errors.New("current anchor missing")
	}
	installOutlineStub(a, tab, source)
	view := &failingView{}
	tab.View = view
	a.editorHost(tab).OpenReference("#目标")
	if view.flushes != 1 || len(source.anchorInputs) != 1 || source.anchorInputs[0] != "目标" {
		t.Fatalf("anchor route = flushes %d resolver inputs %#v message %q", view.flushes, source.anchorInputs, a.Message)
	}

	view.flushErr = os.ErrPermission
	a.editorHost(tab).OpenReference("#另一个")
	if view.flushes != 2 || len(source.anchorInputs) != 1 || !strings.Contains(a.Message, "未能同步格式编辑") {
		t.Fatalf("failed anchor flush = flushes %d resolver inputs %#v message %q", view.flushes, source.anchorInputs, a.Message)
	}
}
