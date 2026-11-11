package markdownview

import (
	"errors"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

func beginDraft(t *testing.T, view *View, tester *ui.Tester, label string) {
	t.Helper()
	rect, ok := tester.Find(label)
	if !ok {
		t.Fatalf("block not rendered: %q %#v", label, tester.Texts())
	}
	before := view.draft
	tester.ClickAt(rect.X+rect.W/2, rect.Y+rect.H/2)
	tester.Frame()
	tester.ClickAt(rect.X+rect.W/2, rect.Y+rect.H/2)
	tester.Frame()
	if view.draft == nil || view.draft == before {
		t.Fatal("actual double click did not enter new block editor")
	}
}

func TestNativeDoubleClickEntryAndCompositionFlushBarriers(t *testing.T) {
	for _, action := range []string{"flush", "export", "release", "commit", "done", "switch", "add"} {
		t.Run(action, func(t *testing.T) {
			original := "# 标题\r\n\r\n第二块\r\n"
			buffer := text.New(original)
			changes := 0
			v := Adapter().Create(editor.Host{Buffer: buffer, Changed: func() { changes++ }}).(*View)
			tester := ui.NewTester(v.Draw, 720, 640)
			beginDraft(t, v, tester, "标题")
			// Entry focuses a caret; replacement is a separate explicit action.
			if v.draft.Buffer.String() != "标题" {
				t.Fatal("entry altered source text")
			}
			tester.Command("selectAll")
			tester.Compose("中文🙂", 3)
			if !v.pendingDraft() || buffer.InGroup() || buffer.String() != original {
				t.Fatal("preedit must remain local without holding host transaction")
			}
			switch action {
			case "flush":
				if err := editor.FlushView(v); err != nil {
					t.Fatal(err)
				}
			case "export":
				if html, err := v.Export("html"); err != nil || !strings.Contains(string(html), "中文🙂") {
					t.Fatal("export omitted preedit", err)
				}
			case "release":
				v.Release()
			case "commit":
				tester.Type("中文🙂")
			case "done":
				if err := tester.Click("完成编辑"); err != nil {
					t.Fatal(err)
				}
			case "switch":
				beginDraft(t, v, tester, "第二块")
			case "add":
				if err := tester.Click("添加段落"); err != nil {
					t.Fatal(err)
				}
			}
			want := "# 中文🙂\r\n\r\n第二块\r\n"
			if action == "add" {
				want += "\r\n\r\n新段落\r\n"
			}
			if buffer.String() != want || changes == 0 {
				t.Fatalf("IME barrier lost text/syntax: %q", buffer.String())
			}
			if action == "add" {
				buffer.Undo()
			}
			if !buffer.Undo() || buffer.String() != original {
				t.Fatal("one composition undo did not restore original bytes")
			}
		})
	}
}

func TestNativeUnicodeMultiStepEditDeleteExportThenContinue(t *testing.T) {
	buffer := text.New("前文🙂\r\n\r\n# 标题\r\n\r\n后文\r\n")
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	tester := ui.NewTester(v.Draw, 720, 640)
	beginDraft(t, v, tester, "标题")
	tester.Command("selectAll")
	tester.Type("新🙂标题")
	if _, err := v.Export("html"); err != nil {
		t.Fatal(err)
	}
	tester.Type("追加")
	tester.Key(0, ui.KeyBackspace)
	tester.Command("selectAll")
	tester.Type("短\r\n第二行🙂")
	if err := v.Flush(); err != nil {
		t.Fatal(err)
	}
	want := "前文🙂\r\n\r\n# 短\r\n第二行🙂\r\n\r\n后文\r\n"
	if buffer.String() != want || v.editBase != "短\r\n第二行🙂" {
		t.Fatalf("UTF16/byte bookkeeping lost text: %q", buffer.String())
	}
	tester.Command("selectAll")
	tester.Key(0, ui.KeyBackspace)
	if err := v.Flush(); err != nil || buffer.String() != "前文🙂\r\n\r\n# \r\n\r\n后文\r\n" {
		t.Fatal("deletion corrupted range", err, buffer.String())
	}
}

func TestPendingDraftRebasesOutsideSourceEdits(t *testing.T) {
	buffer := text.New("前文🙂\r\n\r\n# 标题\r\n\r\n尾部\r\n")
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	tester := ui.NewTester(v.Draw, 720, 640)
	beginDraft(t, v, tester, "标题")
	tester.Command("selectAll")
	tester.Compose("本地🙂", 3)
	_ = buffer.ReplaceUTF16(0, text.UTF16Len("前文🙂"), "改动的前文🙂🙂")
	_ = buffer.ReplaceUTF16(buffer.LenUTF16(), buffer.LenUTF16(), "追加\r\n")
	tester.Frame()
	if err := v.Flush(); err != nil {
		t.Fatal("unrelated source edits should map", err)
	}
	want := "改动的前文🙂🙂\r\n\r\n# 本地🙂\r\n\r\n尾部\r\n追加\r\n"
	if buffer.String() != want {
		t.Fatalf("unrelated edits lost: %q", buffer.String())
	}
}

func TestChangedPrecedingBlockRemapsNextActualDoubleClick(t *testing.T) {
	buffer := text.New("# 标题\n\n第二块🙂\n\n第三块\n")
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	tester := ui.NewTester(v.Draw, 720, 640)
	beginDraft(t, v, tester, "标题")
	tester.Command("selectAll")
	tester.Type("很长的标题🙂\n额外行\n\n额外段落")
	beginDraft(t, v, tester, "第二块🙂")
	if v.draft.Buffer.String() != "第二块🙂" {
		t.Fatal("stale block index opened wrong source", v.draft.Buffer.String())
	}
	tester.Command("selectAll")
	tester.Type("第二块改动")
	if err := v.Flush(); err != nil {
		t.Fatal(err)
	}
	want := "# 很长的标题🙂\n额外行\n\n额外段落\n\n第二块改动\n\n第三块\n"
	if buffer.String() != want {
		t.Fatalf("wrong target after block length/kind changes: %q", buffer.String())
	}
}

func TestNativeConflictRecoveryReleaseAndBlockedActions(t *testing.T) {
	buffer := text.New("# 标题\n\n第二块\n")
	recoverFails, recovered := true, ""
	v := Adapter().Create(editor.Host{Buffer: buffer, RecoverDraft: func(title, value string) error {
		if !strings.HasSuffix(title, ".txt") {
			t.Fatal(title)
		}
		recovered = value
		if recoverFails {
			return errors.New("new document failed")
		}
		return nil
	}}).(*View)
	tester := ui.NewTester(v.Draw, 720, 640)
	beginDraft(t, v, tester, "标题")
	tester.Command("selectAll")
	tester.Compose("保留草稿🙂", 5)
	_ = buffer.ReplaceUTF16(2, 4, "外部")
	authority := buffer.String()
	tester.Frame()
	draft := v.draft
	if err := v.Flush(); !errors.Is(err, errDraftConflict) {
		t.Fatal("same field edit did not conflict", err)
	}
	if _, err := v.Export("html"); !errors.Is(err, errDraftConflict) {
		t.Fatal("conflicting export succeeded", err)
	}
	v.addParagraph()
	block := v.doc.Blocks[1]
	v.begin(1, block)
	if v.draft != draft || buffer.String() != authority {
		t.Fatal("failed flush changed source or discarded editor")
	}
	v.Release()
	if v.draft != draft || len(v.doc.Blocks) != 0 || v.doc.Raw() != "" {
		t.Fatal("release lost draft or kept parsed cache")
	}
	tester.Frame()
	if !tester.HasText("块草稿尚未写回") || !tester.HasText("打开未同步副本") {
		t.Fatal("released failed draft not visible/recoverable")
	}
	if err := v.RecoverDraft(); err == nil || v.draft != draft || buffer.String() != authority {
		t.Fatal("failed recovery discarded input or changed source")
	}
	recoverFails = false
	if err := v.RecoverDraft(); err != nil || v.draft != nil || recovered != "保留草稿🙂" || buffer.String() != authority {
		t.Fatal("recovery must create separate exact-text copy", err)
	}
}

func TestForeignGroupRetainsDraftAndRetryDoesNotStealTransaction(t *testing.T) {
	buffer := text.New("# 标题\n")
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	tester := ui.NewTester(v.Draw, 720, 480)
	beginDraft(t, v, tester, "标题")
	tester.Command("selectAll")
	_ = buffer.BeginGroup()
	tester.Type("草稿🙂")
	if !errors.Is(v.draftError, text.ErrGroup) || buffer.String() != "# 标题\n" || !buffer.InGroup() {
		t.Fatal("native draft stole foreign group")
	}
	v.Release()
	tester.Frame()
	if !buffer.InGroup() || v.draft == nil || !v.pendingDraft() {
		t.Fatal("release closed foreign group or lost draft")
	}
	buffer.EndGroup()
	if err := v.Flush(); err != nil || buffer.String() != "# 草稿🙂\n" || buffer.History().UndoDepth != 1 {
		t.Fatal("retry failed", err)
	}
}

func TestCleanSourceUndoWinsAndDirtyUndoConflicts(t *testing.T) {
	for _, pending := range []bool{false, true} {
		buffer := text.New("# 标题\n")
		v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
		tester := ui.NewTester(v.Draw, 720, 480)
		beginDraft(t, v, tester, "标题")
		tester.Command("selectAll")
		tester.Type("已同步")
		if pending {
			tester.Compose("未提交🙂", 4)
		}
		if !buffer.Undo() {
			t.Fatal("host undo blocked")
		}
		tester.Frame()
		err := v.Flush()
		if buffer.String() != "# 标题\n" {
			t.Fatal("save silently reapplied undone edit")
		}
		if pending && (!errors.Is(err, errDraftConflict) || v.draft == nil) {
			t.Fatal("dirty undo must retain conflict", err)
		}
		if !pending && (err != nil || v.draft != nil) {
			t.Fatal("clean undo must yield to source", err)
		}
	}
}

func TestDraftIdentityRestoreExpiryAndForeignCancellationStayRecoverable(t *testing.T) {
	for _, change := range []string{"buffer", "restore", "expired", "canceled-target", "deleted-target", "changed-syntax"} {
		t.Run(change, func(t *testing.T) {
			buffer := text.New("# 标题\n\n尾部\n")
			v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
			tester := ui.NewTester(v.Draw, 720, 480)
			beginDraft(t, v, tester, "标题")
			tester.Command("selectAll")
			tester.Compose("待恢复🙂", 4)
			switch change {
			case "buffer":
				v.host.Buffer = text.New(buffer.String())
			case "restore":
				buffer.Restore(buffer.Snapshot())
			case "expired":
				for i := 0; i < 65; i++ {
					_ = buffer.ReplaceUTF16(buffer.LenUTF16(), buffer.LenUTF16(), "x")
				}
			case "canceled-target":
				_ = buffer.BeginGroup()
				_ = buffer.ReplaceUTF16(2, 4, "临时")
				buffer.CancelGroup()
			case "deleted-target":
				_ = buffer.ReplaceUTF16(0, 5, "")
			case "changed-syntax":
				_ = buffer.ReplaceUTF16(0, 2, "````\n")
			}
			authority := v.host.Buffer.String()
			tester.Frame()
			if err := v.Flush(); !errors.Is(err, errDraftConflict) || v.draft == nil || v.draft.Buffer.String() != "待恢复🙂" || v.host.Buffer.String() != authority {
				t.Fatal("uncertain field target accepted or draft lost", err)
			}
			v.DiscardDraft()
			if v.draft != nil || v.host.Buffer.String() != authority {
				t.Fatal("explicit discard changed authoritative source")
			}
		})
	}
}

func TestCleanExternalReplacementWinsWithoutLocalInput(t *testing.T) {
	buffer := text.New("# 标题\n")
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	tester := ui.NewTester(v.Draw, 720, 480)
	beginDraft(t, v, tester, "标题")
	_ = buffer.ReplaceUTF16(2, 4, "外部新标题🙂")
	if err := v.Flush(); err != nil || v.draft != nil || buffer.String() != "# 外部新标题🙂\n" {
		t.Fatal("clean stale editor overwrote external replacement", err)
	}
	tester.Frame()
	if !tester.HasText("外部新标题") {
		t.Fatal("source replacement not redrawn")
	}
}

func TestAddParagraphFollowsCRLFWithoutFinalNewline(t *testing.T) {
	buffer := text.New("# 标题\r\n\r\n最后一行🙂")
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	tester := ui.NewTester(v.Draw, 720, 480)
	if err := tester.Click("添加段落"); err != nil {
		t.Fatal(err)
	}
	if buffer.String() != "# 标题\r\n\r\n最后一行🙂\r\n\r\n新段落\r\n" {
		t.Fatal("paragraph introduced mixed line endings", buffer.String())
	}
}

func TestMarkdownExtensionsAndExportWarningsAreNotDuplicated(t *testing.T) {
	for _, path := range []string{"a.md", "a.MARKDOWN", "a.mdown", "a.mkd", "a.mkdn", "a.mdwn", "a.mdtxt", "a.mdtext"} {
		if !Adapter().Matches(path) {
			t.Fatal("missing supported extension", path)
		}
	}
	var reports []string
	v := Adapter().Create(editor.Host{Buffer: text.New("prefix ![image](asset.png)\n"), Report: func(err error) { reports = append(reports, err.Error()) }}).(*View)
	if _, err := v.Export("html"); err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	for _, warning := range reports {
		if seen[warning] {
			t.Fatal("duplicate export warning", warning)
		}
		seen[warning] = true
	}
	if len(reports) == 0 {
		t.Fatal("unsupported image warning missing")
	}
}

func TestReadOnlyAndSameVersionReleaseAreLossless(t *testing.T) {
	for _, readonly := range []bool{false, true} {
		source := "# 标题\r\n\r\n原始 **文本**\r\n"
		buffer := text.New(source)
		v := Adapter().Create(editor.Host{Buffer: buffer, ReadOnly: readonly}).(*View)
		tester := ui.NewTester(v.Draw, 720, 480)
		if !readonly {
			beginDraft(t, v, tester, "标题")
		}
		v.scroll.Y = 42
		v.Release()
		if v.doc.Raw() != "" || v.draft != nil || v.scroll.Y != 42 {
			t.Fatal("release retained cache or lost scroll")
		}
		v.Warnings()
		tester.Frame()
		if _, err := v.Export("html"); err != nil || buffer.String() != source || buffer.CanUndo() {
			t.Fatal("untouched lifecycle changed bytes", err)
		}
		if readonly {
			v.begin(0, v.doc.Blocks[0])
			v.addParagraph()
			if v.draft != nil || buffer.String() != source {
				t.Fatal("readonly allowed editing")
			}
		}
	}
}

func TestOversizedPreviewShowsGuidanceAndDoesNotExportEmptyHTML(t *testing.T) {
	for _, size := range []int{maxPreviewBlockBytes + 1, 8<<20 + 1} {
		source := strings.Repeat("x", size)
		v := Adapter().Create(editor.Host{Buffer: text.New(source)}).(*View)
		tester := ui.NewTester(v.Draw, 720, 480)
		if tester.HasText("文档为空") || !tester.HasText("源码") {
			t.Fatal("large source shown as empty or missing guidance")
		}
		for _, displayed := range tester.Texts() {
			if len(displayed) > 512 {
				t.Fatal("large raw block was sent to text shaping")
			}
		}
		if size > 8<<20 {
			if _, err := v.Export("html"); err == nil {
				t.Fatal("oversize exported empty content")
			}
		}
		if v.host.Buffer.String() != source {
			t.Fatal("fallback changed source")
		}
	}
}

func TestMarkdownViewScreenshots(t *testing.T) {
	dir := os.Getenv("LEVNOTE_MARKDOWN_SCREENSHOT_DIR")
	if dir == "" {
		t.Skip("set LEVNOTE_MARKDOWN_SCREENSHOT_DIR")
	}
	buffer := text.New("# 标题\n\n段落中文🙂\n")
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	tester := ui.NewTester(v.Draw, 265, 480)
	saveImage := func(name string) {
		t.Helper()
		_ = os.MkdirAll(dir, 0755)
		f, err := os.Create(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, tester.Image()); err != nil {
			t.Fatal(err)
		}
	}
	saveImage("markdown-narrow.png")
	beginDraft(t, v, tester, "标题")
	tester.Command("selectAll")
	tester.Compose("保留草稿🙂", 5)
	_ = buffer.ReplaceUTF16(2, 4, "外部")
	_ = v.Flush()
	tester.Frame()
	saveImage("markdown-conflict-narrow.png")
}
