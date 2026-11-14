package app

import (
	"bytes"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

// Exercise the user's reopen command, not the history slice or receipt helpers.
func TestAutonomousReliabilityReopenNativeFileReadsCurrentBytes(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "本机文件.txt")
	if err := os.WriteFile(path, []byte("previous disk\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	old := a.Current()
	if old == nil || old.Doc == nil || old.Doc.Path != path {
		t.Fatalf("native file did not open: %s", a.Message)
	}
	if !a.CloseTab(a.Active) || a.Current() != nil {
		t.Fatalf("native file did not close: %s", a.Message)
	}

	currentBytes := append([]byte{0xef, 0xbb, 0xbf}, []byte("最新磁盘内容\r\n")...)
	if err := os.WriteFile(path, currentBytes, 0600); err != nil {
		t.Fatal(err)
	}
	tester := ui.NewTester(a.View, 760, 480)
	tester.Key(ui.Cmd|ui.Shift, ui.KeyT)
	current := a.Current()
	if current == nil || current == old || current.Doc == nil || current.Doc.Path != path ||
		current.Doc.Dirty() || current.Doc.Buffer.String() != "最新磁盘内容\r\n" {
		t.Fatalf("reopen did not adopt current native file bytes: current=%p old=%p message=%s", current, old, a.Message)
	}
	actual, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(actual, currentBytes) {
		t.Fatalf("reopen changed original bytes: err=%v actual=%x", err, actual)
	}
}

// Real differently-cased Unix files stay distinct across open/close/reopen.
// On Windows the same public flow instead verifies folded native identity.
func TestAutonomousReliabilityNativePathIdentity(t *testing.T) {
	root := t.TempDir()
	firstPath := filepath.Join(root, "case.txt")
	secondPath := filepath.Join(root, "CASE.txt")
	if err := os.WriteFile(firstPath, []byte("first file\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	folded := filepath.Separator == '\\'
	if !folded {
		if err := os.WriteFile(secondPath, []byte("second file\r\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(firstPath)
	first := a.Current()
	if first == nil || first.Doc == nil || first.Doc.Path != firstPath {
		t.Fatalf("first native file did not open: %s", a.Message)
	}
	a.Open(secondPath)
	if folded {
		if len(a.Tabs) != 1 || a.Current() != first {
			t.Fatalf("Windows case variant was not a duplicate: tabs=%d message=%s", len(a.Tabs), a.Message)
		}
	} else {
		second := a.Current()
		if len(a.Tabs) != 2 || second == first || second == nil || second.Doc == nil ||
			second.Doc.Path != secondPath || second.Doc.Buffer.String() != "second file\r\n" {
			t.Fatalf("distinct native file was falsely deduplicated: tabs=%d current=%p message=%s", len(a.Tabs), second, a.Message)
		}
		if !a.CloseTab(0) {
			t.Fatalf("first native file did not close: %s", a.Message)
		}
	}
	if !a.CloseTab(a.Active) || a.Current() != nil {
		t.Fatalf("last native file did not close: %s", a.Message)
	}
	tester := ui.NewTester(a.View, 760, 480)
	tester.Key(ui.Cmd|ui.Shift, ui.KeyT)
	wantPath, wantText := firstPath, "first file\r\n"
	if !folded {
		wantPath, wantText = secondPath, "second file\r\n"
	}
	if current := a.Current(); current == nil || current.Doc == nil || current.Doc.Path != wantPath ||
		current.Doc.Buffer.String() != wantText || current.Doc.Dirty() {
		t.Fatalf("newest native identity did not reopen independently: %s", a.Message)
	}
	if !folded {
		tester.Key(ui.Cmd|ui.Shift, ui.KeyT)
		if current := a.Current(); len(a.Tabs) != 2 || current == nil || current.Doc == nil ||
			current.Doc.Path != firstPath || current.Doc.Buffer.String() != "first file\r\n" || current.Doc.Dirty() {
			t.Fatalf("second reopen consumed or selected a differently-cased file: tabs=%d message=%s", len(a.Tabs), a.Message)
		}
	}
	actual, err := os.ReadFile(firstPath)
	if err != nil || !bytes.Equal(actual, []byte("first file\r\n")) {
		t.Fatalf("native identity flow changed first disk bytes: err=%v actual=%x", err, actual)
	}
	if !folded {
		actual, err = os.ReadFile(secondPath)
		if err != nil || !bytes.Equal(actual, []byte("second file\r\n")) {
			t.Fatalf("native identity flow changed second disk bytes: err=%v actual=%x", err, actual)
		}
	}
}

func TestAutonomousReliabilityEditorHostNativeReferences(t *testing.T) {
	for _, form := range []string{"native absolute", "native file URL"} {
		t.Run(form, func(t *testing.T) {
			root := t.TempDir()
			sourcePath := filepath.Join(root, "来源.md")
			targetPath := filepath.Join(root, "目标 %20.txt")
			sourceBytes := []byte("# 来源\r\n")
			targetBytes := append([]byte{0xef, 0xbb, 0xbf}, []byte("原样目标\r\n")...)
			if err := os.WriteFile(sourcePath, sourceBytes, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(targetPath, targetBytes, 0600); err != nil {
				t.Fatal(err)
			}
			a := newTestApp(t, filepath.Join(root, "config"))
			a.Open(sourcePath)
			source := a.Current()
			if source == nil || source.Doc == nil || source.Doc.Path != sourcePath {
				t.Fatalf("saved source did not open: %s", a.Message)
			}
			reference := targetPath
			if form == "native file URL" {
				urlPath := filepath.ToSlash(targetPath)
				if filepath.VolumeName(targetPath) != "" {
					urlPath = "/" + urlPath
				}
				reference = (&url.URL{Scheme: "file", Path: urlPath}).String()
			}
			a.editorHost(source).OpenReference(reference)
			current := a.Current()
			if len(a.Tabs) != 2 || current == source || current == nil || current.Doc == nil ||
				current.Doc.Path != targetPath || current.Doc.Buffer.String() != "原样目标\r\n" || current.Doc.Dirty() {
				t.Fatalf("native reference did not open exact current bytes: tabs=%d reference=%q message=%s", len(a.Tabs), reference, a.Message)
			}
			actual, err := os.ReadFile(targetPath)
			if err != nil || !bytes.Equal(actual, targetBytes) {
				t.Fatalf("reference changed target bytes: err=%v actual=%x", err, actual)
			}
			actual, err = os.ReadFile(sourcePath)
			if err != nil || !bytes.Equal(actual, sourceBytes) {
				t.Fatalf("reference changed source bytes: err=%v actual=%x", err, actual)
			}
		})
	}
}

func TestAutonomousReliabilityFileURLTokens(t *testing.T) {
	for _, token := range []string{"#", "#target", "?", "?download=1", "%00", ""} {
		t.Run(token, func(t *testing.T) {
			root := t.TempDir()
			sourcePath := filepath.Join(root, "source.md")
			targetPath := filepath.Join(root, "附件#.txt")
			if err := os.WriteFile(sourcePath, []byte("# source\n"), 0600); err != nil {
				t.Fatal(err)
			}
			targetBytes := []byte("exact local target\r\n")
			if err := os.WriteFile(targetPath, targetBytes, 0600); err != nil {
				t.Fatal(err)
			}
			a := newTestApp(t, filepath.Join(root, "config"))
			a.Open(sourcePath)
			source := a.Current()
			if source == nil || source.Doc == nil {
				t.Fatalf("source did not open: %s", a.Message)
			}
			urlPath := filepath.ToSlash(targetPath)
			if filepath.VolumeName(targetPath) != "" {
				urlPath = "/" + urlPath
			}
			reference := (&url.URL{Scheme: "file", Path: urlPath}).String() + token
			beforeMessage := a.Message
			a.editorHost(source).OpenReference(reference)
			if token == "" {
				current := a.Current()
				if len(a.Tabs) != 2 || current == nil || current == source || current.Doc == nil ||
					current.Doc.Path != targetPath || current.Doc.Buffer.String() != string(targetBytes) {
					t.Fatalf("encoded # filename was mistaken for a fragment: %s", a.Message)
				}
			} else if len(a.Tabs) != 1 || a.Current() != source || a.Message == beforeMessage {
				t.Fatalf("file URL token opened a target or failed silently: token=%q tabs=%d message=%s", token, len(a.Tabs), a.Message)
			}
			actual, err := os.ReadFile(targetPath)
			if err != nil || !bytes.Equal(actual, targetBytes) {
				t.Fatalf("file URL token changed target bytes: err=%v actual=%x", err, actual)
			}
		})
	}
}

func TestAutonomousReliabilityFailedReopenRetainsHistory(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "retained.txt")
	aside := filepath.Join(root, "retained-away.txt")
	if err := os.WriteFile(path, []byte("previous disk\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	old := a.Current()
	if old == nil || old.Doc == nil || !a.CloseTab(a.Active) {
		t.Fatalf("native file did not open and close: %s", a.Message)
	}
	if err := os.Rename(path, aside); err != nil {
		t.Fatal(err)
	}
	tester := ui.NewTester(a.View, 760, 480)
	tester.Key(ui.Cmd|ui.Shift, ui.KeyT)
	if a.Current() != nil || len(a.Tabs) != 0 || a.Message == "" {
		t.Fatalf("failed reopen adopted a missing file or failed silently: %s", a.Message)
	}
	if err := os.Rename(aside, path); err != nil {
		t.Fatal(err)
	}
	currentBytes := append([]byte{0xef, 0xbb, 0xbf}, []byte("restored current disk\r\n")...)
	if err := os.WriteFile(path, currentBytes, 0600); err != nil {
		t.Fatal(err)
	}
	tester.Key(ui.Cmd|ui.Shift, ui.KeyT)
	current := a.Current()
	if current == nil || current == old || current.Doc == nil || current.Doc.Path != path ||
		current.Doc.Buffer.String() != "restored current disk\r\n" || current.Doc.Dirty() {
		t.Fatalf("failed reopen consumed history instead of retrying current disk bytes: %s", a.Message)
	}
	actual, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(actual, currentBytes) {
		t.Fatalf("retry changed current disk bytes: err=%v actual=%x", err, actual)
	}
}
