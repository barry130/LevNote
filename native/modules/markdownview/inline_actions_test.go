package markdownview

import (
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

const inlineActionSource = "# 标题🙂\r\n\r\n<!-- unknown: keep exactly -->\r\n"

func inlineActionView(t *testing.T, buffer *text.Buffer) (*View, *ui.Tester, *int) {
	t.Helper()
	changes := 0
	v := Adapter().Create(editor.Host{Buffer: buffer, Changed: func() { changes++ }}).(*View)
	tester := ui.NewTester(v.Draw, 760, 560)
	beginDraft(t, v, tester, "标题🙂")
	tester.Command("selectAll")
	return v, tester, &changes
}

func clickInlineButton(t *testing.T, tester *ui.Tester, label string) {
	t.Helper()
	tester.Advance(time.Second)
	info, ok := tester.AccessByRole(ui.RoleButton, label)
	if !ok || info.Disabled || info.Offscreen {
		t.Fatalf("format button missing: %s %#v", label, info)
	}
	if err := tester.Click(label); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
}

func TestMarkdownInlineButtonsKeysUndoRedoAndFinish(t *testing.T) {
	for _, tc := range []struct {
		label, marker string
		key           ui.Key
	}{
		{"粗体", "**", ui.KeyB}, {"斜体", "*", ui.KeyI}, {"行内代码", "`", ui.KeyE},
	} {
		t.Run(tc.label, func(t *testing.T) {
			buffer := text.New(inlineActionSource)
			v, tester, changes := inlineActionView(t, buffer)
			clickInlineButton(t, tester, tc.label)
			want := strings.Replace(inlineActionSource, "标题🙂", tc.marker+"标题🙂"+tc.marker, 1)
			if buffer.String() != want || *changes != 1 || buffer.History().UndoDepth != 1 || v.draft.Buffer.History().UndoDepth != 1 {
				t.Fatalf("pointer action did not sync exactly once: %q changes=%d error=%v", buffer.String(), *changes, v.inlineActionError)
			}
			if v.draft.Buffer.Selection() != (text.Range{Anchor: len(tc.marker), Head: len(tc.marker) + 4}) {
				t.Fatal("selection did not stay inside markers", v.draft.Buffer.Selection())
			}
			tester.Key(ui.Cmd, ui.KeyZ)
			if buffer.String() != inlineActionSource {
				t.Fatal("focused draft undo escaped", buffer.String())
			}
			tester.Key(ui.Cmd, ui.KeyY)
			if buffer.String() != want {
				t.Fatal("focused draft redo failed", buffer.String())
			}
			tester.Key(ui.Cmd, tc.key)
			if buffer.String() != inlineActionSource {
				t.Fatalf("local shortcut did not toggle: %q / %v", buffer.String(), v.inlineActionError)
			}
			tester.Key(ui.Cmd, tc.key)
			if buffer.String() != want {
				t.Fatal("shortcut did not wrap again", buffer.String())
			}
			if err := tester.Click("完成编辑"); err != nil {
				t.Fatal(err)
			}
			tester.Frame()
			if v.draft != nil || buffer.String() != want {
				t.Fatal("finish discarded formatted source")
			}
			tester.Key(ui.Cmd, tc.key)
			if buffer.String() != want {
				t.Fatal("format shortcut escaped draft scope")
			}
		})
	}
}

func TestMarkdownInlineReversedUTF16AndEmptyCaret(t *testing.T) {
	buffer := text.New(inlineActionSource)
	v, tester, _ := inlineActionView(t, buffer)
	v.draft.Reveal(4, 0)
	tester.Frame()
	tester.Key(ui.Cmd, ui.KeyB)
	if v.draft.Buffer.Selection() != (text.Range{Anchor: 6, Head: 2}) {
		t.Fatal("reverse UTF16 selection lost", v.draft.Buffer.Selection())
	}
	tester.Key(ui.Cmd, ui.KeyZ)
	if v.draft.Buffer.Selection() != (text.Range{Anchor: 4, Head: 0}) {
		t.Fatal("undo selection lost")
	}
	tester.Key(ui.Cmd, ui.KeyY)
	if v.draft.Buffer.Selection() != (text.Range{Anchor: 6, Head: 2}) {
		t.Fatal("redo selection lost")
	}
	tester.Key(ui.Cmd, ui.KeyB)
	v.draft.Reveal(4, 4)
	tester.Frame()
	tester.Key(ui.Cmd, ui.KeyE)
	if v.draft.Buffer.String() != "标题🙂``" || v.draft.Buffer.Selection() != (text.Range{Anchor: 5, Head: 5}) {
		t.Fatal("empty code caret", v.draft.Buffer.String(), v.draft.Buffer.Selection())
	}
	tester.Type("中文")
	if !strings.Contains(buffer.String(), "标题🙂`中文`\r\n") {
		t.Fatal("typing did not land inside code", buffer.String())
	}
}

func TestMarkdownInlineCompositionRetainedUntilExplicitCommit(t *testing.T) {
	buffer := text.New(inlineActionSource)
	v, tester, changes := inlineActionView(t, buffer)
	tester.Compose("新🙂", 3)
	local := v.draft.Buffer.String()
	for _, key := range []ui.Key{ui.KeyB, ui.KeyI, ui.KeyE} {
		tester.Key(ui.Cmd, key)
		if _, marked := v.draft.MarkedRange(); !marked || v.draft.Buffer.String() != local || buffer.String() != inlineActionSource || *changes != 0 {
			t.Fatal("shortcut committed composition")
		}
	}
	clickInlineButton(t, tester, "粗体")
	if _, marked := v.draft.MarkedRange(); !marked || buffer.String() != inlineActionSource || *changes != 0 || v.inlineActionError == nil {
		t.Fatal("button committed composition or hid feedback")
	}
	tester.Type("新🙂")
	tester.Command("selectAll")
	tester.Key(ui.Cmd, ui.KeyB)
	if !strings.Contains(buffer.String(), "**新🙂**") || buffer.History().UndoDepth != 2 {
		t.Fatal("composition and formatting not separate", buffer.String(), buffer.History())
	}
}

func TestMarkdownInlineAdmissionReadonlyConflictAndHeldSelection(t *testing.T) {
	for _, reason := range []string{"readonly", "host-limit", "local-limit", "foreign-group", "pending", "held-selection", "replacement", "release"} {
		t.Run(reason, func(t *testing.T) {
			buffer := text.New(inlineActionSource)
			if reason == "host-limit" {
				buffer = text.NewWithLimit(inlineActionSource, len(inlineActionSource))
			}
			v, tester, changes := inlineActionView(t, buffer)
			if reason == "local-limit" {
				v.draft.Buffer = text.NewWithLimit("标题🙂", len("标题🙂"))
				v.draft.Reveal(0, 4)
				tester.Frame()
			}
			tester.Advance(time.Second)
			button, _ := tester.AccessByRole(ui.RoleButton, "粗体")
			x, y := button.Bounds.X+4, button.Bounds.Y+4
			tester.Press(x, y)
			switch reason {
			case "readonly":
				v.host.ReadOnly = true
			case "foreign-group":
				if err := buffer.BeginGroup(); err != nil {
					t.Fatal(err)
				}
			case "pending":
				if err := v.draft.Buffer.ReplaceUTF16(0, 1, "待"); err != nil {
					t.Fatal(err)
				}
				if err := buffer.ReplaceUTF16(2, 3, "外"); err != nil {
					t.Fatal(err)
				}
			case "held-selection":
				if err := v.draft.Buffer.SetSelection(0, 0); err != nil {
					t.Fatal(err)
				}
			case "replacement":
				v.host.Buffer = text.New(inlineActionSource)
			case "release":
				v.Release()
			}
			before, version, history := buffer.String(), buffer.Version(), buffer.History()
			tester.Release(x, y)
			tester.Frame()
			if buffer.String() != before || buffer.Version() != version || buffer.History() != history || *changes != 0 {
				t.Fatalf("%s changed source/history", reason)
			}
			if reason == "pending" && (v.draft == nil || !strings.Contains(v.draft.Buffer.String(), "待")) {
				t.Fatal("conflict dropped draft")
			}
			if reason == "foreign-group" && (!buffer.InGroup() || !buffer.CancelGroup()) {
				t.Fatal("format closed foreign group")
			}
		})
	}
}

func TestMarkdownInlineMultilineAndNestedFeedback(t *testing.T) {
	for _, value := range []string{"两行\r\n文字", "**已有粗体**", "`边界", " 两侧空白 "} {
		buffer := text.New(inlineActionSource)
		v, tester, changes := inlineActionView(t, buffer)
		tester.Type(value)
		tester.Command("selectAll")
		before, version, count := buffer.String(), buffer.Version(), *changes
		tester.Key(ui.Cmd, ui.KeyE)
		if v.inlineActionError == nil || buffer.String() != before || buffer.Version() != version || *changes != count {
			t.Fatal("unsupported boundary mutated or lacked feedback", value, v.inlineActionError)
		}
	}
}
