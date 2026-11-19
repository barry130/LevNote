package markdownview_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/modules/markdownview"
	"levnote.local/native/ui/editor"
)

func openFencedDraft(t *testing.T, tester *ui.Tester) {
	t.Helper()
	rect, ok := tester.Find("go")
	if !ok {
		t.Fatal("code block language label not rendered", tester.Texts())
	}
	tester.ClickAt(rect.X+rect.W/2, rect.Y+rect.H/2)
	tester.Frame()
	tester.ClickAt(rect.X+rect.W/2, rect.Y+rect.H/2)
	tester.Frame()
	if _, ok := tester.Find("完成编辑"); !ok {
		t.Fatal("double click did not open the code block editor", tester.Texts())
	}
}

func TestFencedDraftReplacementPreservesFollowingParagraph(t *testing.T) {
	original := "```go\nold\n```\n\nafter\n"
	buffer := text.New(original)
	view := markdownview.Adapter().Create(editor.Host{Buffer: buffer})
	t.Cleanup(view.Release)
	tester := ui.NewTester(view.Draw, 720, 640)
	openFencedDraft(t, tester)
	tester.Command("selectAll")
	tester.Type("new")
	if err := tester.Click("完成编辑"); err != nil {
		t.Fatal(err)
	}
	want := "```go\nnew\n```\n\nafter\n"
	if got := buffer.String(); got != want {
		t.Fatalf("code edit joined its closing fence or changed following prose:\ngot  %q\nwant %q", got, want)
	}
	html, err := view.Export("html")
	if err != nil || !strings.Contains(string(html), "<p>after</p>") {
		t.Fatalf("following paragraph is no longer a paragraph: %s (%v)", html, err)
	}
	if !buffer.Undo() || buffer.String() != original {
		t.Fatalf("one document undo did not restore exact source: %q", buffer.String())
	}
	if !buffer.Redo() || buffer.String() != want {
		t.Fatalf("redo did not preserve the closing fence: %q", buffer.String())
	}
}

func TestFencedDraftBoundaryVariants(t *testing.T) {
	for _, tc := range []struct {
		name, original, input, want string
	}{
		{"empty-lf", "```go\n```\n\nafter\n", "new", "```go\nnew\n```\n\nafter\n"},
		{"empty-crlf", "```go\r\n```\r\n\r\nafter\r\n", "新🙂", "```go\r\n新🙂\r\n```\r\n\r\nafter\r\n"},
		{"replace-crlf", "前文🙂\r\n\r\n```go\r\nold\r\n```\r\n\r\nafter\r\n", "新🙂", "前文🙂\r\n\r\n```go\r\n新🙂\r\n```\r\n\r\nafter\r\n"},
		{"delete-lf", "```go\nold\n```\n\nafter\n", "", "```go\n```\n\nafter\n"},
		{"delete-crlf", "```go\r\nold\r\n```\r\n\r\nafter\r\n", "", "```go\r\n```\r\n\r\nafter\r\n"},
		{"long-fence-no-final-newline", "````go\r\nold\r\n````   	", "new", "````go\r\nnew\r\n````   	"},
		{"explicit-final-line-ending", "```go\r\nold\r\n```\r\n", "new\r\n", "```go\r\nnew\r\n```\r\n"},
		{"unclosed-eof", "```go\nold", "new", "```go\nnew"},
		{"unclosed-crlf", "```go\r\nold\r\n", "new", "```go\r\nnew"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			buffer := text.New(tc.original)
			view := markdownview.Adapter().Create(editor.Host{Buffer: buffer})
			t.Cleanup(view.Release)
			tester := ui.NewTester(view.Draw, 720, 640)
			openFencedDraft(t, tester)
			tester.Command("selectAll")
			if tc.input == "" {
				tester.Key(0, ui.KeyBackspace)
			} else {
				tester.Type(tc.input)
			}
			if got := buffer.String(); got != tc.want {
				t.Fatalf("source after input:\ngot  %q\nwant %q", got, tc.want)
			}
			version := buffer.Version()
			for i := 0; i < 2; i++ {
				if err := editor.FlushView(view); err != nil || buffer.Version() != version || buffer.String() != tc.want {
					t.Fatalf("repeat Flush changed accepted draft: %q (%v)", buffer.String(), err)
				}
			}
			if !tester.HasText("修改已同步到文档") {
				t.Fatal("accepted draft remains visibly dirty", tester.Texts())
			}
			if err := tester.Click("完成编辑"); err != nil || buffer.Version() != version {
				t.Fatal("completion rewrote the accepted source", err)
			}
			if !buffer.Undo() || buffer.String() != tc.original {
				t.Fatalf("one undo lost original source: %q", buffer.String())
			}
			if !buffer.Redo() || buffer.String() != tc.want {
				t.Fatalf("redo changed fence boundary: %q", buffer.String())
			}
		})
	}
}

func TestFencedDraftUnchangedAndReadOnlyPreserveBytes(t *testing.T) {
	for _, readonly := range []bool{false, true} {
		for _, original := range []string{"```go\r\n```", "````go\r\nold\r\n````  	", "```go\nold"} {
			buffer := text.New(original)
			changes := 0
			view := markdownview.Adapter().Create(editor.Host{Buffer: buffer, ReadOnly: readonly, Changed: func() { changes++ }})
			tester := ui.NewTester(view.Draw, 720, 640)
			if readonly {
				rect, ok := tester.Find("go")
				if !ok {
					t.Fatal("code language label missing")
				}
				tester.ClickAt(rect.X+rect.W/2, rect.Y+rect.H/2)
				tester.ClickAt(rect.X+rect.W/2, rect.Y+rect.H/2)
				tester.Command("selectAll")
				tester.Type("new")
				if _, ok := tester.Find("完成编辑"); ok {
					t.Fatal("readonly code block opened an editor")
				}
			} else {
				openFencedDraft(t, tester)
				if err := tester.Click("完成编辑"); err != nil {
					t.Fatal(err)
				}
			}
			if err := editor.FlushView(view); err != nil {
				t.Fatal(err)
			}
			view.Release()
			if buffer.String() != original || buffer.CanUndo() || changes != 0 {
				t.Fatalf("unchanged or readonly lifecycle rewrote bytes: %q", buffer.String())
			}
		}
	}
}

func TestFencedDraftBudgetFailurePreservesInputAndRetries(t *testing.T) {
	original := "```go\r\na\r\n```\r\n\r\nafter\r\n"
	buffer := text.NewWithLimit(original, len(original))
	recoveryFailure := errors.New("recovery blocked for this check")
	recovered := ""
	view := markdownview.Adapter().Create(editor.Host{Buffer: buffer, RecoverDraft: func(_, value string) error {
		recovered = value
		return recoveryFailure
	}})
	t.Cleanup(view.Release)
	tester := ui.NewTester(view.Draw, 720, 640)
	openFencedDraft(t, tester)
	tester.Command("selectAll")
	tester.Type("new🙂")
	if err := editor.FlushView(view); !errors.Is(err, text.ErrSizeLimit) || buffer.String() != original || buffer.CanUndo() {
		t.Fatal("budget failure wrote source or dropped the failure", err, buffer.String())
	}
	if err := tester.Click("打开未同步副本"); err != nil || recovered != "new🙂" {
		t.Fatal("recovery did not receive exact input without a synthetic line ending", err, recovered)
	}
	// Removing unrelated trailing prose frees enough budget; the same local
	// input must rebase and retry without needing another keystroke.
	end := buffer.LenUTF16()
	if err := buffer.ReplaceUTF16(end-text.UTF16Len("after\r\n"), end, ""); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if err := editor.FlushView(view); err != nil {
		t.Fatal("retained draft did not retry after budget was freed", err)
	}
	want := "```go\r\nnew🙂\r\n```\r\n\r\n"
	if buffer.String() != want {
		t.Fatalf("retry lost input or fence boundary: %q", buffer.String())
	}
	if err := tester.Click("完成编辑"); err != nil {
		t.Fatal(err)
	}
}

func TestFencedDraftForeignGroupFailureRetries(t *testing.T) {
	original := "```go\nold\n```\n\nafter\n"
	buffer := text.New(original)
	view := markdownview.Adapter().Create(editor.Host{Buffer: buffer})
	t.Cleanup(view.Release)
	tester := ui.NewTester(view.Draw, 720, 640)
	openFencedDraft(t, tester)
	if err := buffer.BeginGroup(); err != nil {
		t.Fatal(err)
	}
	tester.Command("selectAll")
	tester.Type("new")
	if err := editor.FlushView(view); !errors.Is(err, text.ErrGroup) || buffer.String() != original || !buffer.InGroup() {
		t.Fatal("foreign group was changed or draft failure was lost", err)
	}
	buffer.EndGroup()
	if err := editor.FlushView(view); err != nil || buffer.String() != "```go\nnew\n```\n\nafter\n" {
		t.Fatal("retained draft did not retry after group completion", err, buffer.String())
	}
}

func TestFencedDraftExternalConflictPreservesInputForRecovery(t *testing.T) {
	original := "```go\nold\n```\n\nafter\n"
	buffer := text.New(original)
	recovered := ""
	view := markdownview.Adapter().Create(editor.Host{Buffer: buffer, RecoverDraft: func(_, value string) error {
		recovered = value
		return nil
	}})
	t.Cleanup(view.Release)
	tester := ui.NewTester(view.Draw, 720, 640)
	openFencedDraft(t, tester)
	tester.Command("selectAll")
	tester.Compose("new🙂", 5)
	if err := buffer.ReplaceUTF16(6, 9, "source"); err != nil {
		t.Fatal(err)
	}
	want := "```go\nsource\n```\n\nafter\n"
	tester.Frame()
	if err := editor.FlushView(view); err == nil || buffer.String() != want {
		t.Fatal("conflicting local input overwrote source", err, buffer.String())
	}
	if err := tester.Click("打开未同步副本"); err != nil || recovered != "new🙂" || buffer.String() != want {
		t.Fatal("conflicting input was not recovered verbatim", err, recovered, buffer.String())
	}
}

func TestFencedDraftContinuedInputAndCleanHostUndo(t *testing.T) {
	original := "```go\r\nold\r\n```\r\n\r\nafter\r\n"
	buffer := text.New(original)
	view := markdownview.Adapter().Create(editor.Host{Buffer: buffer})
	t.Cleanup(view.Release)
	tester := ui.NewTester(view.Draw, 720, 640)
	openFencedDraft(t, tester)
	tester.Command("selectAll")
	tester.Type("new")
	tester.Type("🙂")
	want := "```go\r\nnew🙂\r\n```\r\n\r\nafter\r\n"
	if err := editor.FlushView(view); err != nil || buffer.String() != want {
		t.Fatal("continued input used the wrong source span", err, buffer.String())
	}
	if !buffer.Undo() {
		t.Fatal("host undo did not restore the previous code text")
	}
	undone := buffer.String()
	tester.Frame()
	if err := editor.FlushView(view); err != nil || buffer.String() != undone || undone == want {
		t.Fatal("clean local editor reapplied text after host undo", err, buffer.String())
	}
}

// Enter on a replaced single-line local draft defaults to LF. That input must
// not change the original structural line ending before a closing fence.
func TestFencedDraftEnterPreservesClosingLineEnding(t *testing.T) {
	for _, body := range []string{"old\r\n", ""} {
		t.Run(map[bool]string{true: "empty", false: "replace"}[body == ""], func(t *testing.T) {
			original := "```go\r\n" + body + "```\r\n\r\nafter\r\n"
			buffer := text.New(original)
			view := markdownview.Adapter().Create(editor.Host{Buffer: buffer})
			t.Cleanup(view.Release)
			tester := ui.NewTester(view.Draw, 720, 640)
			openFencedDraft(t, tester)
			tester.Command("selectAll")
			tester.Type("new")
			tester.Key(0, ui.KeyEnter)
			want := "```go\r\nnew\r\n```\r\n\r\nafter\r\n"
			if err := editor.FlushView(view); err != nil || buffer.String() != want {
				t.Fatalf("Enter changed the closing fence separator: got %q want %q (%v)", buffer.String(), want, err)
			}
			// Internal newlines still belong to the exact input; only the last
			// separator is structural. Continued typing must keep both baselines.
			tester.Type("next")
			want = "```go\r\nnew\nnext\r\n```\r\n\r\nafter\r\n"
			if err := editor.FlushView(view); err != nil || buffer.String() != want {
				t.Fatalf("continued input lost its source span: got %q want %q (%v)", buffer.String(), want, err)
			}
			version := buffer.Version()
			if err := tester.Click("完成编辑"); err != nil || buffer.Version() != version || buffer.String() != want {
				t.Fatal("completion rewrote the accepted keyboard draft", err)
			}
		})
	}
}
