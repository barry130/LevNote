package markdownview

import (
	"errors"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/markdown"
	"levnote.local/native/ui/editor"
)

func TestEveryPartialParseShowsSourceGuidanceAndRefusesHTML(t *testing.T) {
	for _, tc := range []struct{ name, source string }{
		{"bytes", strings.Repeat("x", (8<<20)+1)},
		{"lines", strings.Repeat("\n", 65538)},
		{"blocks", strings.Repeat("# h\n", 8194)},
		{"list", strings.Repeat("- item\n", 4098)},
		{"inline-scan", strings.Repeat("[", 32) + strings.Repeat("x", 65536)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			buffer := text.New(tc.source)
			var reports []error
			v := Adapter().Create(editor.Host{Buffer: buffer, Report: func(err error) { reports = append(reports, err) }}).(*View)
			tester := ui.NewTester(v.Draw, 532, 480)
			if !v.doc.Partial {
				t.Fatal("fixture did not hit parser budget")
			}
			guidance := partialSourceGuidance
			if tc.name == "bytes" {
				guidance = oversizeSourceGuidance
			}
			if _, ok := tester.Find(guidance); !ok {
				t.Fatal("partial preview omitted source guidance", tester.Texts())
			}
			if _, ok := tester.Find("添加段落"); ok {
				t.Fatal("partial preview still offered editing incomplete structure")
			}
			if output, err := v.Export("html"); len(output) != 0 || !errors.Is(err, markdown.ErrPartialDocument) {
				t.Fatal("partial export produced output or hid reason", len(output), err)
			}
			if len(reports) == 0 || !strings.Contains(strings.Join(v.Warnings(), " "), "源码") || buffer.String() != tc.source || buffer.History().UndoDepth != 0 {
				t.Fatal("partial parse altered source or failed to disclose")
			}
		})
	}
}

func TestPartialSourceAfterActiveIMEFlushRetainsDraft(t *testing.T) {
	buffer := text.New("# 标题\n")
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	tester := ui.NewTester(v.Draw, 720, 480)
	beginDraft(t, v, tester, "标题")
	tester.Command("selectAll")
	tester.Compose("本地🙂", 3)
	// An unrelated source insertion triggers a parser budget, but the pending
	// field still maps and must flush before export reports incomplete parsing.
	_ = buffer.ReplaceUTF16(buffer.LenUTF16(), buffer.LenUTF16(), strings.Repeat("\n", 65538))
	if output, err := v.Export("html"); len(output) != 0 || !errors.Is(err, markdown.ErrPartialDocument) {
		t.Fatal(len(output), err)
	}
	if !strings.HasPrefix(buffer.String(), "# 本地🙂\n") || v.draft == nil || v.pendingDraft() {
		t.Fatal("partial rejection discarded or stranded active composition")
	}
	v.Release()
	tester.Frame()
	if !v.doc.Partial || !strings.HasPrefix(buffer.String(), "# 本地🙂\n") {
		t.Fatal("release/rebuild lost committed field")
	}
}
