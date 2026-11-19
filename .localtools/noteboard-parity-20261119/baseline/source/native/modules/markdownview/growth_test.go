package markdownview

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

func TestGrowthMarkdownDraftRefusalRetainsRecoveryAndRetry(t *testing.T) {
	for _, finish := range []string{"recover", "shorten-retry"} {
		t.Run(finish, func(t *testing.T) {
			const source = "# 标题\r\n\r\n正文\r\n"
			buffer := text.NewWithLimit(source, len(source))
			var reported error
			var recovered string
			changes := 0
			recoveryErr := errors.New("destination unavailable")
			v := Adapter().Create(editor.Host{Buffer: buffer, Path: "growth.md", Report: func(err error) { reported = err }, Changed: func() { changes++ }, RecoverDraft: func(_, value string) error {
				if recoveryErr != nil {
					return recoveryErr
				}
				recovered = value
				return nil
			}}).(*View)
			tester := ui.NewTester(v.Draw, 760, 480)
			beginDraft(t, v, tester, "标题")
			tester.Command("selectAll")
			tester.Compose("中文🙂", 3)
			draft := v.draft
			version, selection, history, snapshot := buffer.Version(), buffer.Selection(), buffer.History(), buffer.Snapshot()
			if err := v.Flush(); !errors.Is(err, text.ErrSizeLimit) || !errors.Is(reported, text.ErrSizeLimit) {
				t.Fatal("real writeback failed without size identity", err, reported)
			}
			tester.Frame()
			if v.draft != draft || draft.Buffer.String() != "中文🙂" || !v.pendingDraft() || changes != 0 ||
				buffer.Version() != version || buffer.Selection() != selection || buffer.History() != history || !buffer.EqualSnapshot(snapshot) {
				t.Fatal("refused rich flush lost independent draft or changed host")
			}
			if _, err := v.Export("html"); !errors.Is(err, text.ErrSizeLimit) {
				t.Fatal("export bypassed draft barrier", err)
			}
			v.Release()
			tester.Frame()
			if v.draft != draft || draft.Buffer.String() != "中文🙂" || !v.pendingDraft() || buffer.String() != source {
				t.Fatal("release lost refused draft")
			}
			switch finish {
			case "recover":
				if err := v.RecoverDraft(); err != recoveryErr || v.draft != draft || !v.pendingDraft() {
					t.Fatal("failed recovery cleared draft", err)
				}
				recoveryErr = nil
				if err := v.RecoverDraft(); err != nil || recovered != "中文🙂" || v.draft != nil || buffer.String() != source || buffer.Version() != version {
					t.Fatal("recovery did not preserve original and full draft", err)
				}
			case "shorten-retry":
				draft.ReplaceText(&ui.TextInputRange{Start: 0, End: draft.Buffer.LenUTF16()}, "短")
				if err := v.Flush(); err != nil || buffer.String() != "# 短\r\n\r\n正文\r\n" || v.pendingDraft() || changes != 1 {
					t.Fatal("shortened draft did not retry", err)
				}
				if !buffer.Undo() || buffer.String() != source || !buffer.Redo() || buffer.String() != "# 短\r\n\r\n正文\r\n" {
					t.Fatal("accepted retry history changed")
				}
			}
		})
	}
}
