package markdownview

import (
	"errors"
	"fmt"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/markdown"
	"levnote.local/native/ui/editor"
)

func linkAccess(t *testing.T, tester *ui.Tester, label string) ui.AccessInfo {
	t.Helper()
	info, ok := tester.AccessByRole(ui.RoleLink, linkName(label))
	if !ok || info.Bounds.W <= 0 || info.Bounds.H <= 0 {
		t.Fatalf("native link missing %q: %+v texts=%q", label, info, tester.Texts())
	}
	return info
}

func clickLink(t *testing.T, tester *ui.Tester, label string) {
	t.Helper()
	rect, ok := tester.Find(linkName(label))
	if !ok {
		t.Fatalf("link fragment missing %q", label)
	}
	tester.ClickAtWith(ui.Cmd, rect.X+min(4, rect.W/2), rect.Y+rect.H/2)
}

// Pure command tests capture the same source authority as the native renderer,
// independently converting ranges rather than invoking the renderer helper.
func captureLink(t *testing.T, v *View, ordinal int) linkTarget {
	t.Helper()
	v.refresh()
	for _, row := range v.rows {
		block := v.doc.Blocks[row.block]
		inline := block.Inline
		if row.item >= 0 {
			inline = block.Items[row.item].Inline
		}
		for _, in := range inline {
			if in.Kind != markdown.InlineLink {
				continue
			}
			if ordinal > 0 {
				ordinal--
				continue
			}
			return linkTarget{buffer: v.host.Buffer, version: v.version, rowID: row.id, enclosing: row.span,
				syntax: text.Range{Anchor: text.UTF16Len(v.doc.Raw()[:in.Range.Start]), Head: text.UTF16Len(v.doc.Raw()[:in.Range.End])},
				kind:   row.kind, inlineKind: in.Kind, released: v.linkReleased, item: row.item >= 0, href: strings.Clone(in.Href)}
		}
	}
	t.Fatal("fixture has no link")
	return linkTarget{}
}

func TestLinkNativeSelectionDoubleClickPointerKeyboardReadonly(t *testing.T) {
	for _, readonly := range []bool{false, true} {
		t.Run(fmt.Sprint(readonly), func(t *testing.T) {
			raw := "# 标题\r\n\r\nprefix [链接🙂文字](子目录/目标.txt) **粗体** *斜体* `代码` suffix\r\n"
			buffer := text.New(raw)
			selection := text.Range{Anchor: buffer.LenUTF16() - 2, Head: 2}
			_ = buffer.SetSelection(selection.Anchor, selection.Head)
			var opened []string
			var reports []error
			changes := 0
			v := Adapter().Create(editor.Host{Buffer: buffer, ReadOnly: readonly, Changed: func() { changes++ }, OpenReference: func(s string) { opened = append(opened, s) }, Report: func(err error) { reports = append(reports, err) }}).(*View)
			tester := ui.NewTester(v.Draw, 720, 480)
			info := linkAccess(t, tester, "链接🙂文字")
			if info.Disabled || !info.Focusable || !strings.Contains(info.Description, "Tab 后 Enter") {
				t.Fatal("link access metadata", info)
			}
			rect, _ := tester.Find(linkName("链接🙂文字"))
			x, y := rect.X+4, rect.Y+rect.H/2
			tester.Press(x, y)
			tester.Move(rect.X+rect.W-4, y)
			tester.Release(rect.X+rect.W-4, y)
			tester.Command("copy")
			if tester.Clipboard() == "" || len(opened) != 0 || v.draft != nil {
				t.Fatal("plain drag cannot select linked text", tester.Clipboard())
			}
			tester.Advance(time.Second)
			clickLink(t, tester, "链接🙂文字")
			if len(opened) != 1 || opened[0] != "子目录/目标.txt" {
				t.Fatal("full destination not opened once", opened, reports)
			}
			tester.Key(0, ui.KeyEnter)
			tester.Key(0, ui.KeySpace)
			if len(opened) != 2 || buffer.String() != raw || buffer.Selection() != selection || buffer.History().UndoDepth != 0 || changes != 0 {
				t.Fatal("navigation altered source/history or Space activated", opened, buffer.Selection(), changes, reports)
			}
			tester.Advance(time.Second)
			tester.ClickAt(x, y)
			tester.ClickAt(x, y)
			if (v.draft != nil) == readonly || len(opened) != 2 {
				t.Fatal("plain double click behavior", readonly, v.draft != nil, opened)
			}
			if !readonly && v.draft.Buffer.String() != strings.TrimSuffix(strings.Split(raw, "\r\n\r\n")[1], "\r\n") {
				t.Fatal("double-click draft did not keep original Markdown")
			}
		})
	}
}

func TestLinkNativeWrappedFragmentsAndTaskCoexist(t *testing.T) {
	label := "wrapped link 中文🙂 across several lines and the last line"
	raw := "prefix [" + label + "](local.txt) suffix\n\n- [ ] task [另一个](other.txt) **bold**\n"
	buffer := text.New(raw)
	var opened []string
	v := Adapter().Create(editor.Host{Buffer: buffer, OpenReference: func(s string) { opened = append(opened, s) }}).(*View)
	tester := ui.NewTester(v.Draw, 265, 480)
	info := linkAccess(t, tester, label)
	first, _ := tester.Find(linkName(label))
	if info.Bounds.H < first.H*2 {
		t.Fatal("link did not wrap", info, first)
	}
	tester.ClickAtWith(ui.Cmd, first.X+4, first.Y+first.H/2)
	tester.ClickAtWith(ui.Cmd, info.Bounds.X+4, info.Bounds.Y+info.Bounds.H-first.H/2)
	tester.ClickAtWith(ui.Cmd, info.Bounds.X+4, first.Y+first.H/2)
	if len(opened) != 2 || v.draft != nil {
		t.Fatal("wrapped fragment/gap/double press", opened, v.draft != nil)
	}
	clickLink(t, tester, "另一个")
	if len(opened) != 3 || opened[2] != "other.txt" || buffer.String() != raw {
		t.Fatal("task label link route", opened)
	}
	box := taskAccess(t, tester, "未完成任务：task [另一个](other.txt) **bold**")
	tester.ClickAt(box.Bounds.X+box.Bounds.W/2, box.Bounds.Y+box.Bounds.H/2)
	if buffer.String() != strings.Replace(raw, "[ ]", "[x]", 1) || len(opened) != 3 || v.draft != nil {
		t.Fatal("checkbox/link routing conflict", buffer.String(), opened)
	}
	clickLink(t, tester, "另一个")
	if len(opened) != 4 || tester.FramePending() {
		t.Fatal("rebuilt task link did not settle", opened)
	}
}

func TestLinkBoundedMetadataEmptyAndUnavailable(t *testing.T) {
	label := strings.Repeat("中🙂", 40)
	href := strings.Repeat("目", 200)
	tooLong := strings.Repeat("x", maxLinkTargetBytes+1)
	raw := "[]()\n\n[" + label + "](" + href + ")\n\n[巨大目标](" + tooLong + ")\n\n[外部](https://example.invalid)\n"
	buffer := text.New(raw)
	var opened []string
	v := Adapter().Create(editor.Host{Buffer: buffer, OpenReference: func(s string) { opened = append(opened, s) }}).(*View)
	tester := ui.NewTester(v.Draw, 720, 640)
	info := linkAccess(t, tester, "")
	if info.Name != "链接：（空链接）" || info.Disabled {
		t.Fatal("empty link not discoverable", info)
	}
	clickLink(t, tester, "")
	info = linkAccess(t, tester, label)
	if !strings.HasSuffix(info.Name, "…") || !utf8.ValidString(info.Name) || len(info.Name) > len("链接：")+131 || len(info.Description) > len("Ctrl/Cmd+点击或 Tab 后 Enter 打开：")+515 || !utf8.ValidString(info.Description) {
		t.Fatal("unbounded redundant metadata", info)
	}
	clickLink(t, tester, label)
	if len(opened) != 2 || opened[0] != "" || opened[1] != href {
		t.Fatal("destination was truncated/normalized", opened)
	}
	info = linkAccess(t, tester, "巨大目标")
	if !info.Disabled || info.Focusable || info.Description != linkTooLong || !tester.HasText("〔链接不可用〕") {
		t.Fatal("oversize target not explicitly unavailable", info)
	}
	clickLink(t, tester, "巨大目标")
	if len(opened) != 2 {
		t.Fatal("oversize target opened")
	}
	if linkAccess(t, tester, "外部").Disabled {
		t.Fatal("view duplicated host classification")
	}
	clickLink(t, tester, "外部")
	if len(opened) != 3 || opened[2] != "https://example.invalid" || buffer.String() != raw {
		t.Fatal("host classification not authoritative")
	}
	v.host.OpenReference = nil
	tester.Frame()
	if info = linkAccess(t, tester, ""); !info.Disabled || info.Description != linkNoHost {
		t.Fatal("missing host not exposed", info)
	}
}

func TestLinkExactIdentityAfterJournalAndContextChanges(t *testing.T) {
	for _, change := range []string{"unchanged", "before", "duplicate-before", "closed-fence", "open-fence", "label", "href", "delete", "restore", "expired", "buffer", "foreign-group", "partial", "oversize"} {
		t.Run(change, func(t *testing.T) {
			raw := "[相同](first.txt)\r\n\r\n前文🙂\r\n\r\n- [ ] [相同](second.txt)\r\n\r\n尾部\r\n"
			buffer := text.New(raw)
			var opened []string
			changes := 0
			v := Adapter().Create(editor.Host{Buffer: buffer, Changed: func() { changes++ }, OpenReference: func(s string) { opened = append(opened, s) }}).(*View)
			target := captureLink(t, v, 1)
			success := false
			prefix := ""
			switch change {
			case "unchanged":
				success = true
			case "before":
				prefix, success = "# 新🙂\n\n", true
			case "duplicate-before":
				prefix, success = "[相同](wrong.txt)\n\n", true
			case "closed-fence":
				prefix, success = "```\ncode\n```\n\n", true
			case "open-fence":
				prefix = "```\n"
			case "label":
				_ = buffer.ReplaceUTF16(target.syntax.Start()+1, target.syntax.Start()+3, "别的")
			case "href":
				_ = buffer.ReplaceUTF16(target.syntax.End()-11, target.syntax.End()-1, "wrong0.txt")
			case "delete":
				_ = buffer.ReplaceUTF16(target.enclosing.Start(), target.enclosing.End(), "")
			case "restore":
				buffer.Restore(buffer.Snapshot())
			case "expired":
				for i := 0; i < 65; i++ {
					_ = buffer.ReplaceUTF16(buffer.LenUTF16(), buffer.LenUTF16(), "x")
				}
			case "buffer":
				v.host.Buffer = text.New(raw)
			case "foreign-group":
				_ = buffer.BeginGroup()
			case "partial":
				_ = buffer.ReplaceUTF16(buffer.LenUTF16(), buffer.LenUTF16(), "\n\n"+strings.Repeat("- x\n", 4200))
			case "oversize":
				_ = buffer.ReplaceUTF16(buffer.LenUTF16(), buffer.LenUTF16(), strings.Repeat("x", markdown.MaxParseBytes))
			}
			if prefix != "" {
				_ = buffer.ReplaceUTF16(0, 0, prefix)
			}
			authority, selection, history := v.host.Buffer.String(), v.host.Buffer.Selection(), v.host.Buffer.History()
			err := v.activateLink(target)
			if success {
				if err != nil || len(opened) != 1 || opened[0] != "second.txt" {
					t.Fatal("exact duplicate identity failed", err, opened)
				}
			} else if err == nil || len(opened) != 0 {
				t.Fatal("stale target navigated", err, opened)
			}
			if v.host.Buffer.String() != authority || v.host.Buffer.Selection() != selection || v.host.Buffer.History() != history || changes != 0 {
				t.Fatal("navigation mutated source/history")
			}
			if change == "foreign-group" && !buffer.InGroup() {
				t.Fatal("foreign transaction taken")
			}
		})
	}
}

func TestLinkNativeComposeFlushConflictAndSameList(t *testing.T) {
	for _, mode := range []string{"other", "same", "conflict"} {
		t.Run(mode, func(t *testing.T) {
			raw := "# 标题\r\n\r\n- [ ] task [目标](target.txt)\r\n- second\r\n\r\n尾部\r\n"
			buffer := text.New(raw)
			var opened []string
			var reported error
			v := Adapter().Create(editor.Host{Buffer: buffer, OpenReference: func(s string) { opened = append(opened, s) }, Report: func(err error) { reported = err }}).(*View)
			tester := ui.NewTester(v.Draw, 720, 640)
			if mode == "same" {
				beginDraft(t, v, tester, "task 目标")
			} else {
				beginDraft(t, v, tester, "标题")
			}
			tester.Command("selectAll")
			composition := "中文🙂"
			if mode == "same" {
				composition = "- [ ] changed [目标](target.txt)\r\n- second\r\n"
			}
			tester.Compose(composition, 3)
			if mode == "conflict" {
				_ = buffer.ReplaceUTF16(2, 4, "外部")
			}
			authority := buffer.String()
			clickLink(t, tester, "目标")
			if mode == "other" {
				if len(opened) != 1 || opened[0] != "target.txt" || reported != nil || !strings.Contains(buffer.String(), "# 中文🙂") || buffer.History().UndoDepth != 1 {
					t.Fatal("other-field Flush/navigation", opened, reported, buffer.String())
				}
				if !buffer.Undo() || buffer.String() != raw {
					t.Fatal("navigation added history")
				}
			} else if mode == "same" {
				if len(opened) != 0 || !errors.Is(reported, errLinkStale) || !strings.Contains(buffer.String(), "changed") {
					t.Fatal("same-list Flush retargeted", opened, reported, buffer.String())
				}
			} else if len(opened) != 0 || !errors.Is(reported, errDraftConflict) || buffer.String() != authority || v.draft == nil || v.draft.Buffer.String() != composition {
				t.Fatal("failed Flush lost draft/navigated", opened, reported)
			}
		})
	}
}

func TestLinkSynchronousReleaseKeepsActiveListBuild(t *testing.T) {
	buffer := text.New("[open](target.txt)\n\n" + headingFixture(100))
	var v *View
	opened := 0
	v = Adapter().Create(editor.Host{Buffer: buffer, OpenReference: func(string) { opened++; v.Release() }}).(*View)
	tester := ui.NewTester(v.Draw, 265, 480)
	oldList := v.list
	clickLink(t, tester, "open")
	if opened != 1 || v.list == oldList || tester.FramePending() {
		t.Fatal("release invalidated active list ownership", opened, tester.FramePending())
	}
	clickLink(t, tester, "open")
	if opened != 2 {
		t.Fatal("view did not recover after release")
	}
}

func TestLinkDenseRowsFarVirtualizationResource(t *testing.T) {
	var dense strings.Builder
	for i := 0; i < 700; i++ {
		fmt.Fprintf(&dense, "[x](t%03d.txt) ", i)
	}
	var far strings.Builder
	for i := 0; i < 1000; i++ {
		fmt.Fprintf(&far, "- [ ] row%04d [目标%04d](t%04d.txt)\n", i, i, i)
	}
	for _, fixture := range []struct{ name, raw string }{{"700 links in one row", dense.String()}, {"1000 link task rows", far.String()}} {
		t.Run(fixture.name, func(t *testing.T) {
			var opened []string
			v := Adapter().Create(editor.Host{Buffer: text.New(fixture.raw), OpenReference: func(s string) { opened = append(opened, s) }}).(*View)
			tester := ui.NewTester(v.Draw, 720, 480)
			if len(v.rows) > 1 {
				v.list.ScrollTo(900, ui.Start)
				tester.Frame()
				clickLink(t, tester, "目标0900")
				if len(opened) != 1 || opened[0] != "t0900.txt" || v.frameRows > 60 {
					t.Fatal("far-row link wrong or not virtual", opened, v.frameRows)
				}
			} else {
				if _, ok := tester.AccessByRole(ui.RoleLink, ""); ok || !tester.HasText(denseLinkGuidance) {
					t.Fatal("dense row must expose readable text/source guidance without partial interaction")
				}
				if v.host.Buffer.String() != fixture.raw || len(opened) != 0 {
					t.Fatal("dense fallback altered source")
				}
			}
			measuredViewWork(t, fixture.name+" five unchanged native frames", func() {
				for i := 0; i < 5; i++ {
					tester.Frame()
				}
			})
			if tester.FramePending() {
				t.Fatal("links never settle")
			}
			t.Logf("built rows=%d pending-frame=%v", v.frameRows, tester.FramePending())
		})
	}
}

func TestLinkControlsBoundaryPreservesTextAndSelection(t *testing.T) {
	for _, count := range []int{16, 17} {
		var source, visible strings.Builder
		for i := 0; i < count; i++ {
			fmt.Fprintf(&source, "[目标%d](t%d.txt) ", i, i)
			fmt.Fprintf(&visible, "目标%d ", i)
		}
		v := Adapter().Create(editor.Host{Buffer: text.New(source.String()), OpenReference: func(string) {}}).(*View)
		tester := ui.NewTester(v.Draw, 720, 480)
		_, active := tester.AccessByRole(ui.RoleLink, "")
		if active != (count == 16) || tester.HasText(denseLinkGuidance) != (count == 17) || !tester.HasText(visible.String()) {
			t.Fatal("link count boundary lost text/control disclosure", count, tester.Texts())
		}
		if count == 17 {
			rect, ok := tester.Find(visible.String())
			if !ok {
				t.Fatal("fallback text unavailable")
			}
			tester.Press(rect.X+4, rect.Y+8)
			tester.Move(rect.X+80, rect.Y+8)
			tester.Release(rect.X+80, rect.Y+8)
			tester.Command("copy")
			if tester.Clipboard() == "" {
				t.Fatal("dense fallback lost selection")
			}
			tester.Advance(time.Second)
			beginDraft(t, v, tester, visible.String())
			if v.draft.Buffer.String() != source.String() {
				t.Fatal("dense fallback draft lost targets")
			}
		}
	}
}

func TestLinkNativeGestureIdentityCancellationAndModifierLoss(t *testing.T) {
	for _, action := range []string{"prefix", "duplicate-prefix", "cancel", "modifier-loss"} {
		t.Run(action, func(t *testing.T) {
			raw := "[same](original.txt)\n\nend\n"
			buffer := text.New(raw)
			var opened []string
			var reported error
			v := Adapter().Create(editor.Host{Buffer: buffer, OpenReference: func(s string) { opened = append(opened, s) }, Report: func(err error) { reported = err }}).(*View)
			var once func()
			tester := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().ObserveInput(func(event ui.InputEvent) {
					if event.Kind == ui.InputPointerDown && once != nil {
						run := once
						once = nil
						run()
					}
				}).Children(func() { v.Draw(c) })
			}, 720, 480)
			rect, _ := tester.Find(linkName("same"))
			x, y := rect.X+4, rect.Y+rect.H/2
			switch action {
			case "prefix":
				once = func() { _ = buffer.ReplaceUTF16(0, 0, "# 新增🙂\n\n") }
			case "duplicate-prefix":
				once = func() { _ = buffer.ReplaceUTF16(0, 0, "[same](wrong.txt)\n\n") }
			case "cancel":
				once = tester.CancelPress
			case "modifier-loss":
				once = func() { tester.Release(x, y) }
			}
			tester.ClickAtWith(ui.Cmd, x, y)
			if action == "prefix" || action == "duplicate-prefix" {
				// Inserting above moves the actual fragment before release; a
				// geometry-canceled press may safely do nothing. It must never
				// resolve through the new first link's label.
				if len(opened) > 1 || len(opened) == 1 && opened[0] != "original.txt" {
					t.Fatal("native gesture silently retargeted duplicate", opened)
				}
			} else if len(opened) != 0 {
				t.Fatal("canceled/modifier-lost gesture navigated", opened, reported)
			}
			if v.draft != nil || tester.FramePending() {
				t.Fatal("link gesture began draft or failed to settle")
			}
		})
	}
}

func TestLinkNativeKeyboardCapturedIdentitySurvivesPrefixEdit(t *testing.T) {
	buffer := text.New("[same](original.txt)\n\nend\n")
	var opened []string
	var reported error
	v := Adapter().Create(editor.Host{Buffer: buffer, OpenReference: func(s string) { opened = append(opened, s) }, Report: func(err error) { reported = err }}).(*View)
	armed := false
	tester := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().ObserveInput(func(event ui.InputEvent) {
			if armed && event.Kind == ui.InputKeyDown && event.Key == ui.KeyEnter {
				armed = false
				_ = buffer.ReplaceUTF16(0, 0, "[same](wrong.txt)\n\n# 新增🙂\n\n")
			}
		}).Children(func() { v.Draw(c) })
	}, 720, 480)
	for i := 0; i < 10; i++ {
		if linkAccess(t, tester, "same").Focused {
			break
		}
		tester.Key(0, ui.KeyTab)
	}
	if !linkAccess(t, tester, "same").Focused {
		t.Fatal("Tab did not focus exact original link")
	}
	armed = true
	tester.Key(0, ui.KeyEnter)
	if len(opened) != 1 || opened[0] != "original.txt" || reported != nil {
		t.Fatal("actual key captured wrong duplicate/current parser", opened, reported)
	}
}

func TestLinkDecoratedWorkBoundaryProbe(t *testing.T) {
	var styled strings.Builder
	for i := 0; i < 512; i++ {
		styled.WriteString("**bold** ")
		if i%32 == 0 {
			styled.WriteString("[x](target.txt) ")
		}
	}
	for _, fixture := range []struct{ name, source string }{{"one 30KiB ASCII link label", "[" + strings.Repeat("long label ", 2800) + "](target.txt)"}, {"16 links among512bold spans", styled.String()}} {
		t.Run(fixture.name, func(t *testing.T) {
			v := Adapter().Create(editor.Host{Buffer: text.New(fixture.source), OpenReference: func(string) {}}).(*View)
			var tester *ui.Tester
			measuredViewWork(t, fixture.name+" cold265px", func() { tester = ui.NewTester(v.Draw, 265, 480) })
			measuredViewWork(t, fixture.name+" five unchanged265px frames", func() {
				for i := 0; i < 5; i++ {
					tester.Frame()
				}
			})
			if tester.FramePending() || v.doc.Partial {
				t.Fatal("boundary fixture did not settle/parse")
			}
		})
	}
}

func TestLinkStyledBudgetBoundaryAndVisibleFallback(t *testing.T) {
	for _, count := range []int{64, 65} {
		for _, list := range []bool{false, true} {
			for _, readonly := range []bool{false, true} {
				var source strings.Builder
				if list {
					source.WriteString("- [ ] ")
				}
				source.WriteString("[目标](target.txt) ")
				for i := 1; i < count; i++ {
					source.WriteString("**格式** ")
				}
				v := Adapter().Create(editor.Host{Buffer: text.New(source.String()), ReadOnly: readonly, OpenReference: func(string) {}}).(*View)
				tester := ui.NewTester(v.Draw, 265, 480)
				_, active := tester.AccessByRole(ui.RoleLink, "")
				if active != (count == 64) || tester.HasText(styledLinkGuidance) != (count == 65) || v.host.Buffer.String() != source.String() {
					t.Fatal("style eligibility/disclosure boundary", count, list, readonly, tester.Texts())
				}
				if list {
					if taskAccess(t, tester, "未完成任务："+boundedLinkText(strings.TrimPrefix(source.String(), "- [ ] "), 128)).Disabled != readonly {
						t.Fatal("fallback disabled independent task control")
					}
				}
				if count == 65 && list && readonly {
					if dir := os.Getenv("LEVNOTE_MARKDOWN_SCREENSHOT_DIR"); dir != "" {
						if err := os.MkdirAll(dir, 0755); err != nil {
							t.Fatal(err)
						}
						f, err := os.Create(filepath.Join(dir, "markdown-links-dense-fallback-265px.png"))
						if err != nil {
							t.Fatal(err)
						}
						err = png.Encode(f, tester.Image())
						_ = f.Close()
						if err != nil {
							t.Fatal(err)
						}
					}
				}
			}
		}
	}
}

func TestLinkInlineCostCurve(t *testing.T) {
	for _, count := range []int{8, 16, 32, 64} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			var source strings.Builder
			for i := 0; i < count; i++ {
				fmt.Fprintf(&source, "[x](t%d.txt) ", i)
			}
			v := Adapter().Create(editor.Host{Buffer: text.New(source.String()), OpenReference: func(string) {}}).(*View)
			tester := ui.NewTester(v.Draw, 720, 480)
			measuredViewWork(t, fmt.Sprintf("%d native links five unchanged frames", count), func() {
				for i := 0; i < 5; i++ {
					tester.Frame()
				}
			})
		})
	}
}

func TestLinkReadableThemeContrastAndNarrowGeometry(t *testing.T) {
	luminance := func(c ui.Color) float64 {
		channel := func(value uint8) float64 {
			x := float64(value) / 255
			if x <= .04045 {
				return x / 12.92
			}
			return math.Pow((x+.055)/1.055, 2.4)
		}
		return .2126*channel(c.R) + .7152*channel(c.G) + .0722*channel(c.B)
	}
	contrast := func(a, b ui.Color) float64 {
		x, y := luminance(a), luminance(b)
		return (max(x, y) + .05) / (min(x, y) + .05)
	}
	for _, dark := range []bool{false, true} {
		for _, readonly := range []bool{false, true} {
			theme := ui.LightTheme()
			if dark {
				theme = ui.DarkTheme()
			}
			theme.Accent = ui.Hex("#33766c") // actual shell control fill
			foreground := linkForeground(theme)
			for _, bg := range []ui.Color{theme.Background, theme.Surface} {
				if ratio := contrast(foreground, bg); ratio < 4.5 {
					t.Fatal("link contrast below4.5", dark, ratio)
				}
			}
			t.Logf("dark=%v readonly=%v former-accent/background=%.3f new/background=%.3f new/surface=%.3f foreground=%+v", dark, readonly, contrast(theme.Accent, theme.Background), contrast(foreground, theme.Background), contrast(foreground, theme.Surface), foreground)
			raw := "# 本地阅读\n\n前文 [换行链接中文🙂与完整标签，继续在下一行阅读](target.txt) **粗体** `代码`\n\n- [ ] task [另一个](other.txt)\n\n> [引用中的链接](quote.txt)\n"
			v := Adapter().Create(editor.Host{Buffer: text.New(raw), ReadOnly: readonly, OpenReference: func(string) {}}).(*View)
			tester := ui.NewTester(func(c *ui.Context) {
				c.SetTheme(theme)
				ui.Column(c).Width(265).Height(480).Background(theme.Background).Children(func() {
					ui.Box(c).Height(450).Shrink(0).Children(func() { v.Draw(c) })
					ui.Text(c, "STATUS").Height(30).Shrink(0)
				})
			}, 265, 480)
			for _, label := range []string{"换行链接中文🙂与完整标签，继续在下一行阅读", "另一个", "引用中的链接"} {
				info := linkAccess(t, tester, label)
				if info.Disabled || info.Bounds.X < 0 || info.Bounds.X+info.Bounds.W > 265 || info.Bounds.Y+info.Bounds.H > 450 {
					t.Fatal("link outside bounded content", info)
				}
			}
			if dir := os.Getenv("LEVNOTE_MARKDOWN_SCREENSHOT_DIR"); dir != "" {
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				f, err := os.Create(filepath.Join(dir, fmt.Sprintf("markdown-links-narrow-dark-%v-readonly-%v.png", dark, readonly)))
				if err != nil {
					t.Fatal(err)
				}
				err = png.Encode(f, tester.Image())
				_ = f.Close()
				if err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}
