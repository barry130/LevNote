package markdownview

import (
	"errors"
	"fmt"
	"image/png"
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

func imageReferenceAccess(t *testing.T, tester *ui.Tester, alt string) ui.AccessInfo {
	t.Helper()
	info, ok := tester.AccessByRole(ui.RoleLink, imageReferenceName(alt))
	if !ok || info.Bounds.W <= 0 || info.Bounds.H <= 0 {
		t.Fatalf("image reference missing %q: %+v texts=%q", alt, info, tester.Texts())
	}
	return info
}

func clickImageReference(t *testing.T, tester *ui.Tester, alt string) {
	t.Helper()
	rect, ok := tester.Find(imageReferenceName(alt))
	if !ok {
		t.Fatalf("image reference fragment missing %q", alt)
	}
	tester.ClickAtWith(ui.Cmd, rect.X+min(4, rect.W/2), rect.Y+rect.H/2)
}

func captureImageReference(t *testing.T, v *View, ordinal int) linkTarget {
	t.Helper()
	v.refresh()
	for _, row := range v.rows {
		block := v.doc.Blocks[row.block]
		inline := block.Inline
		if row.item >= 0 {
			inline = block.Items[row.item].Inline
		}
		for _, in := range inline {
			if in.Kind != markdown.InlineImage {
				continue
			}
			if ordinal > 0 {
				ordinal--
				continue
			}
			return linkTarget{buffer: v.host.Buffer, version: v.version, rowID: row.id,
				released: v.linkReleased, enclosing: row.span, kind: row.kind, inlineKind: in.Kind,
				syntax: text.Range{Anchor: text.UTF16Len(v.doc.Raw()[:in.Range.Start]), Head: text.UTF16Len(v.doc.Raw()[:in.Range.End])},
				item:   row.item >= 0, href: strings.Clone(in.Href)}
		}
	}
	t.Fatal("fixture has no image reference")
	return linkTarget{}
}

func TestImageReferenceNativePointerKeyboardSelectionReadonly(t *testing.T) {
	for _, readonly := range []bool{false, true} {
		t.Run(fmt.Sprint(readonly), func(t *testing.T) {
			const syntax = "![图片🙂](子目录/原%20图.png)"
			raw := "# 标题\r\n\r\n前文 " + syntax + " 后文 [普通链接](note.md)\r\n"
			buffer := text.New(raw)
			_ = buffer.SetSelection(buffer.LenUTF16()-2, 2)
			selection, history, version := buffer.Selection(), buffer.History(), buffer.Version()
			var opened []string
			changes := 0
			v := Adapter().Create(editor.Host{Buffer: buffer, ReadOnly: readonly, Changed: func() { changes++ },
				OpenReference: func(s string) { opened = append(opened, s) }}).(*View)
			tester := ui.NewTester(v.Draw, 720, 480)
			info := imageReferenceAccess(t, tester, "图片🙂")
			if info.Disabled || !info.Focusable || !strings.Contains(info.Description, "图片引用（不内嵌显示）") ||
				!strings.HasSuffix(info.Description, "子目录/原%20图.png") || !tester.HasText(syntax) {
				t.Fatal("reference metadata/raw text", info, tester.Texts())
			}
			rect, _ := tester.Find(info.Name)
			x, y := rect.X+4, rect.Y+rect.H/2
			tester.Press(x, y)
			tester.Move(rect.X+rect.W-4, y)
			tester.Release(rect.X+rect.W-4, y)
			tester.Command("copy")
			if tester.Clipboard() == "" || len(opened) != 0 || v.draft != nil {
				t.Fatal("plain pointer lost raw selection", tester.Clipboard(), opened)
			}
			tester.Advance(time.Second)
			clickImageReference(t, tester, "图片🙂")
			if len(opened) != 1 || opened[0] != "子目录/原%20图.png" {
				t.Fatal("pointer changed raw target", opened)
			}
			tester.Key(0, ui.KeyTab) // ordinary link remains in the same native focus route
			if !linkAccess(t, tester, "普通链接").Focused {
				t.Fatal("mixed link focus route changed")
			}
			tester.Key(ui.Shift, ui.KeyTab)
			if !imageReferenceAccess(t, tester, "图片🙂").Focused {
				t.Fatal("ShiftTab did not return to image")
			}
			tester.Key(0, ui.KeySpace)
			tester.Key(0, ui.KeyEnter)
			tester.KeyRepeat(0, ui.KeyEnter)
			tester.KeyRepeat(0, ui.KeyEnter)
			if len(opened) != 2 || buffer.String() != raw || buffer.Version() != version ||
				buffer.Selection() != selection || buffer.History() != history || changes != 0 {
				t.Fatal("held key/Space/navigation changed authority", opened, buffer.History())
			}
			tester.Advance(time.Second)
			tester.ClickAt(x, y)
			tester.ClickAt(x, y)
			if (v.draft != nil) == readonly || len(opened) != 2 {
				t.Fatal("plain doubleclick/readonly behavior", readonly, opened)
			}
			if !readonly && !strings.Contains(v.draft.Buffer.String(), syntax) {
				t.Fatal("block edit lost raw image syntax")
			}
		})
	}
}

func TestImageReferenceNarrowWrappedEmptyAndBoundedMetadata(t *testing.T) {
	alt := "中文🙂图片引用与原始路径换行以后仍然完整可读"
	syntax := "![" + alt + "](子目录/原图.png)"
	raw := "前文 " + syntax + " 后文\n\n![](empty.jpg)\n\n- [ ] task ![列表](list.png)\n"
	var opened []string
	v := Adapter().Create(editor.Host{Buffer: text.New(raw), OpenReference: func(s string) { opened = append(opened, s) }}).(*View)
	tester := ui.NewTester(v.Draw, 265, 480)
	info := imageReferenceAccess(t, tester, alt)
	first, _ := tester.Find(info.Name)
	if info.Bounds.H < first.H*2 || info.Bounds.X < 0 || info.Bounds.X+info.Bounds.W > 265 || !tester.HasText(syntax) {
		t.Fatal("narrow reference wrapping/raw text", info, first, tester.Texts())
	}
	tester.ClickAtWith(ui.Cmd, first.X+4, first.Y+first.H/2)
	tester.Advance(time.Second)
	tester.ClickAtWith(ui.Cmd, info.Bounds.X+4, info.Bounds.Y+info.Bounds.H-first.H/2)
	tester.Advance(time.Second)
	tester.ClickAtWith(ui.Cmd, info.Bounds.X+4, first.Y+first.H/2)
	if len(opened) != 2 || v.draft != nil {
		t.Fatal("wrapped fragments/gap route", opened)
	}
	if empty := imageReferenceAccess(t, tester, ""); empty.Name != "图片引用：（无替代文字）" || empty.Disabled {
		t.Fatal("empty alt not discoverable", empty)
	}
	tester.Advance(time.Second)
	clickImageReference(t, tester, "")
	clickImageReference(t, tester, "列表")
	if len(opened) != 4 || opened[2] != "empty.jpg" || opened[3] != "list.png" {
		t.Fatal("empty/list image target", opened)
	}
	if dir := os.Getenv("LEVNOTE_IMAGE_REFERENCE_CAPTURE_DIR"); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(filepath.Join(dir, "markdown-image-reference-265x480.png"))
		if err != nil {
			t.Fatal(err)
		}
		err = png.Encode(f, tester.Image())
		_ = f.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	longAlt, longHref := strings.Repeat("中🙂", 80), strings.Repeat("目", 200)+".png"
	v = Adapter().Create(editor.Host{Buffer: text.New("![" + longAlt + "](" + longHref + ")"), OpenReference: func(s string) { opened = append(opened, s) }}).(*View)
	tester = ui.NewTester(v.Draw, 720, 640)
	info = imageReferenceAccess(t, tester, longAlt)
	if !strings.HasSuffix(info.Name, "…") || len(info.Name) > len("图片引用：")+131 ||
		!utf8.ValidString(info.Name) || !utf8.ValidString(info.Description) || len(info.Description) > 640 {
		t.Fatal("unbounded image metadata", info)
	}
	clickImageReference(t, tester, longAlt)
	if opened[len(opened)-1] != longHref {
		t.Fatal("target truncated", opened)
	}
	v.host.OpenReference = nil
	tester.Frame()
	if info = imageReferenceAccess(t, tester, longAlt); !info.Disabled || info.Focusable ||
		info.Description != imageReferenceNoHost || !tester.HasText("〔图片引用不可用〕") {
		t.Fatal("missing host not explicit", info)
	}
}

func TestImageReferenceSharedBudgetsAndLiteralFallback(t *testing.T) {
	for _, count := range []int{16, 17} {
		var source, visible strings.Builder
		for i := 0; i < count; i++ {
			if i%2 == 0 {
				fmt.Fprintf(&source, "![图%d](p%d.png) ", i, i)
				fmt.Fprintf(&visible, "![图%d](p%d.png) ", i, i)
			} else {
				fmt.Fprintf(&source, "[文%d](p%d.txt) ", i, i)
				fmt.Fprintf(&visible, "文%d ", i)
			}
		}
		v := Adapter().Create(editor.Host{Buffer: text.New(source.String()), OpenReference: func(string) { t.Fatal("draw opened reference") }}).(*View)
		tester := ui.NewTester(v.Draw, 720, 480)
		_, active := tester.AccessByRole(ui.RoleLink, "")
		if active != (count == 16) || tester.HasText(denseLinkGuidance) != (count == 17) || !tester.HasText(visible.String()) {
			t.Fatal("mixed control budget or raw fallback", count, tester.Texts())
		}
	}
	for _, count := range []int{64, 65} {
		raw := "![图](p.png) " + strings.Repeat("**格式** ", count-1)
		v := Adapter().Create(editor.Host{Buffer: text.New(raw), OpenReference: func(string) {}}).(*View)
		tester := ui.NewTester(v.Draw, 265, 480)
		_, active := tester.AccessByRole(ui.RoleLink, "")
		if active != (count == 64) || tester.HasText(styledLinkGuidance) != (count == 65) || !tester.HasText("![图](p.png)") {
			t.Fatal("shared span budget", count, tester.Texts())
		}
	}
	oversize := "![大图](" + strings.Repeat("x", 4097) + ")"
	v := Adapter().Create(editor.Host{Buffer: text.New(oversize)}).(*View)
	tester := ui.NewTester(v.Draw, 265, 480)
	if _, active := tester.AccessByRole(ui.RoleLink, ""); active || !tester.HasText(oversize) || len(v.Warnings()) == 0 {
		t.Fatal("oversized image must be warned literal, not a disabled partial action")
	}
	if tester.FramePending() {
		t.Fatal("literal fallback did not settle")
	}
}

func TestImageReferenceOwnedSyntaxAndOuterLinkRemainLiteral(t *testing.T) {
	const image = "![图](inside.png)"
	raw := "`" + image + "`\n\n**" + image + "**\n\n*" + image + "*\n\n" +
		"| H |\n|---|\n| " + image + " |\n\n[" + image + "](outer.md)\n"
	var opened []string
	v := Adapter().Create(editor.Host{Buffer: text.New(raw), OpenReference: func(s string) { opened = append(opened, s) }}).(*View)
	tester := ui.NewTester(v.Draw, 720, 640)
	if _, exists := tester.AccessByRole(ui.RoleLink, imageReferenceName("图")); exists {
		t.Fatal("image inside another syntax owner became an independent control")
	}
	if !tester.HasText(image) || v.host.Buffer.String() != raw {
		t.Fatal("nonrecursive owners lost original image syntax", tester.Texts())
	}
	clickLink(t, tester, image)
	if len(opened) != 1 || opened[0] != "outer.md" || v.host.Buffer.String() != raw {
		t.Fatal("outer ordinary link semantics changed", opened)
	}
}

func TestImageReferenceMappedIdentityAndKind(t *testing.T) {
	for _, action := range []string{"unchanged", "prefix", "duplicate-prefix", "href", "image-to-link", "buffer", "release", "open-fence", "restore", "foreign-group"} {
		t.Run(action, func(t *testing.T) {
			raw := "![相同](first.png)\r\n\r\n- [ ] ![相同](second.png)\r\n"
			buffer := text.New(raw)
			var opened []string
			v := Adapter().Create(editor.Host{Buffer: buffer, OpenReference: func(s string) { opened = append(opened, s) }}).(*View)
			target := captureImageReference(t, v, 1)
			success := action == "unchanged" || action == "prefix" || action == "duplicate-prefix"
			switch action {
			case "prefix":
				_ = buffer.ReplaceUTF16(0, 0, "# 新🙂\r\n\r\n")
			case "duplicate-prefix":
				_ = buffer.ReplaceUTF16(0, 0, "![相同](wrong.png)\r\n\r\n")
			case "href":
				_ = buffer.ReplaceUTF16(target.syntax.End()-11, target.syntax.End()-1, "other0.png")
			case "image-to-link":
				_ = buffer.ReplaceUTF16(target.syntax.Start(), target.syntax.Start()+1, "")
			case "buffer":
				v.host.Buffer = text.New(raw)
			case "release":
				v.Release()
			case "open-fence":
				_ = buffer.ReplaceUTF16(0, 0, "```\n")
			case "restore":
				buffer.Restore(buffer.Snapshot())
			case "foreign-group":
				_ = buffer.BeginGroup()
			}
			before, selection, history := v.host.Buffer.String(), v.host.Buffer.Selection(), v.host.Buffer.History()
			err := v.activateLink(target)
			if success && (err != nil || len(opened) != 1 || opened[0] != "second.png") ||
				!success && (err == nil || len(opened) != 0) {
				t.Fatal("captured image identity", err, opened)
			}
			if v.host.Buffer.String() != before || v.host.Buffer.Selection() != selection || v.host.Buffer.History() != history {
				t.Fatal("activation changed authority")
			}
		})
	}
	// Explicitly exercise the kind gate with equal ranges/target; edit mapping
	// alone cannot distinguish a stale descriptor from a different inline kind.
	v := Adapter().Create(editor.Host{Buffer: text.New("![图](same.png)")}).(*View)
	target := captureImageReference(t, v, 0)
	doc := markdown.Parse("![图](same.png)")
	doc.Blocks[0].Inline[0].Kind = markdown.InlineLink
	if findLink(doc, target.enclosing, target.syntax, target) {
		t.Fatal("same-range wrong kind accepted")
	}
}

func TestImageReferenceNativeGestureCancellationStaleAndRepeat(t *testing.T) {
	for _, action := range []string{"cancel", "modifier-loss", "scroll", "owner", "release", "href", "kind", "repeat"} {
		t.Run(action, func(t *testing.T) {
			buffer := text.New("![图](first.png)\n\n" + headingFixture(100))
			var opened []string
			v := Adapter().Create(editor.Host{Buffer: buffer, OpenReference: func(s string) { opened = append(opened, s) }}).(*View)
			var once func()
			tester := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().ObserveInput(func(event ui.InputEvent) {
					if event.Kind == ui.InputPointerDown && once != nil {
						fn := once
						once = nil
						fn()
					}
				}).Children(func() { v.Draw(c) })
			}, 265, 480)
			rect, _ := tester.Find(imageReferenceName("图"))
			x, y := rect.X+4, rect.Y+rect.H/2
			switch action {
			case "cancel":
				once = tester.CancelPress
			case "modifier-loss":
				once = func() { tester.Release(x, y) }
			case "scroll":
				once = func() { tester.Scroll(x, y, 0, 2200) }
			case "owner":
				once = func() { v.host.Buffer = text.New("![图](wrong.png)\n") }
			case "release":
				once = v.Release
			case "href":
				once = func() { _ = buffer.ReplaceUTF16(6, 11, "other") }
			case "kind":
				once = func() { _ = buffer.ReplaceUTF16(0, 1, "") }
			}
			tester.ClickAtWith(ui.Cmd, x, y)
			if action == "repeat" {
				tester.ClickAtWith(ui.Cmd, x, y)
				if len(opened) != 1 || opened[0] != "first.png" {
					t.Fatal("rapid continuation opened twice", opened)
				}
			} else if len(opened) != 0 {
				t.Fatal("canceled/stale image opened", opened)
			}
			if v.linkActivation.buffer != nil || v.draft != nil || tester.FramePending() {
				t.Fatal("gesture/source retained or not settled")
			}
		})
	}
}

func TestImageReferenceKeyboardPrefixAndSynchronousRelease(t *testing.T) {
	buffer := text.New("![图](original.png)\n\n" + headingFixture(80))
	var v *View
	var opened []string
	v = Adapter().Create(editor.Host{Buffer: buffer, OpenReference: func(s string) { opened = append(opened, s); v.Release() }}).(*View)
	armed := false
	tester := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Fill().ObserveInput(func(event ui.InputEvent) {
			if armed && event.Kind == ui.InputKeyDown && event.Key == ui.KeyEnter {
				armed = false
				_ = buffer.ReplaceUTF16(0, 0, "![图](wrong.png)\n\n# 新🙂\n\n")
			}
		}).Children(func() { v.Draw(c) })
	}, 265, 480)
	for i := 0; i < 10 && !imageReferenceAccess(t, tester, "图").Focused; i++ {
		tester.Key(0, ui.KeyTab)
	}
	if !imageReferenceAccess(t, tester, "图").Focused {
		t.Fatal("image not keyboard reachable")
	}
	oldList := v.list
	armed = true
	tester.Key(0, ui.KeyEnter)
	if len(opened) != 1 || opened[0] != "original.png" || v.list == oldList || tester.FramePending() {
		t.Fatal("keyboard prefix mapping/reentrant release", opened)
	}
}

func TestImageReferenceComposeFlushAndConflict(t *testing.T) {
	for _, mode := range []string{"other", "same", "conflict"} {
		t.Run(mode, func(t *testing.T) {
			raw := "# 标题\r\n\r\n- [ ] task ![图](target.png)\r\n- second\r\n"
			buffer := text.New(raw)
			var opened []string
			var reported error
			v := Adapter().Create(editor.Host{Buffer: buffer, OpenReference: func(s string) { opened = append(opened, s) }, Report: func(err error) { reported = err }}).(*View)
			tester := ui.NewTester(v.Draw, 720, 640)
			if mode == "same" {
				beginDraft(t, v, tester, "task ![图](target.png)")
			} else {
				beginDraft(t, v, tester, "标题")
			}
			tester.Command("selectAll")
			composition := "中文🙂"
			if mode == "same" {
				composition = "- [ ] changed ![图](target.png)\r\n- second\r\n"
			}
			tester.Compose(composition, 3)
			if mode == "conflict" {
				_ = buffer.ReplaceUTF16(2, 4, "外部")
			}
			before := buffer.String()
			clickImageReference(t, tester, "图")
			if mode == "other" {
				if len(opened) != 1 || opened[0] != "target.png" || reported != nil || !strings.Contains(buffer.String(), "# 中文🙂") || buffer.History().UndoDepth != 1 {
					t.Fatal("other draft flush", opened, reported)
				}
				if !buffer.Undo() || buffer.String() != raw {
					t.Fatal("image action added undo")
				}
			} else if mode == "same" {
				if len(opened) != 0 || !errors.Is(reported, errLinkStale) || !strings.Contains(buffer.String(), "changed") {
					t.Fatal("same field silently retargeted", opened, reported)
				}
			} else if len(opened) != 0 || !errors.Is(reported, errDraftConflict) || buffer.String() != before || v.draft == nil || v.draft.Buffer.String() != composition {
				t.Fatal("conflict lost draft", opened, reported)
			}
		})
	}
}
