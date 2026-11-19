package markdownview

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

const referenceViewBody = "# 标题\r\n\r\n```doc-block\r\nid: shared\r\nkind: note\r\ntitle: 同名\r\nmode: inline\r\n---\r\n第一个内联块\r\n```\r\n\r\n```doc-block\r\nid: shared\r\nkind: text\r\ntitle: 同名\r\nmode: reference\r\nsrc: targets/a.txt\r\n---\r\n引用载荷\r\n```\r\n\r\n尾段\r\n"

func referenceAccess(t *testing.T, tester *ui.Tester, enabled bool) ui.AccessInfo {
	t.Helper()
	info, ok := tester.AccessByRole(ui.RoleButton, "打开引用")
	if !ok {
		t.Fatalf("reference button missing: %#v", tester.Texts())
	}
	if info.Name != "打开引用" || info.Bounds.W <= 0 || info.Bounds.H <= 0 || info.Offscreen {
		t.Fatalf("reference button geometry/name = %#v", info)
	}
	if enabled {
		if info.Disabled || !info.Focusable {
			t.Fatalf("enabled reference button state = %#v", info)
		}
	} else if !info.Disabled || info.Focusable {
		t.Fatalf("disabled reference button state = %#v", info)
	}
	if len(info.Description) > maxReferenceTooltipBytes+len("…") {
		t.Fatalf("reference tooltip exceeds bound: %d", len(info.Description))
	}
	return info
}

func clickReference(t *testing.T, tester *ui.Tester, info ui.AccessInfo) {
	t.Helper()
	tester.ClickAt(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	tester.Frame()
}

func focusReference(t *testing.T, tester *ui.Tester, info ui.AccessInfo) {
	t.Helper()
	tester.Press(info.Bounds.X+info.Bounds.W/2, info.Bounds.Y+info.Bounds.H/2)
	tester.CancelPress()
	tester.Frame()
	if tester.Focused("打开引用") {
		return
	}
	for attempt := 0; attempt < 120; attempt++ {
		tester.Key(0, ui.KeyTab)
		tester.Frame()
		if tester.Focused("打开引用") {
			return
		}
	}
	t.Fatalf("reference button could not receive keyboard focus: %#v", tester.Texts())
}

func replaceReferenceBufferText(t *testing.T, buffer *text.Buffer, old, replacement string) {
	t.Helper()
	source := buffer.String()
	index := strings.Index(source, old)
	if index < 0 {
		t.Fatalf("fixture text %q is absent", old)
	}
	start := text.UTF16Len(source[:index])
	end := start + text.UTF16Len(old)
	if err := buffer.ReplaceUTF16(start, end, replacement); err != nil {
		t.Fatal(err)
	}
}

func TestReferenceNativePointerEnterSpaceCancelAndFlush(t *testing.T) {
	for _, gesture := range []string{"pointer", "enter", "space"} {
		t.Run(gesture, func(t *testing.T) {
			buffer := text.New(referenceViewBody)
			var opened []string
			changes := 0
			v := Adapter().Create(editor.Host{
				Buffer: buffer,
				Changed: func() {
					changes++
				},
				OpenReference: func(src string) {
					opened = append(opened, src)
				},
			}).(*View)
			tester := ui.NewTester(v.Draw, 1180, 780)
			tester.Frame()

			beginDraft(t, v, tester, "标题")
			tester.Command("selectAll")
			tester.Compose("中文🙂", 3)
			tester.Frame()
			before := buffer.String()
			button := referenceAccess(t, tester, true)

			tester.Press(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
			tester.CancelPress()
			tester.Frame()
			if len(opened) != 0 || v.referenceActivation.buffer != nil {
				t.Fatal("canceled press activated or retained a reference gesture")
			}

			switch gesture {
			case "pointer":
				clickReference(t, tester, button)
			case "enter", "space":
				focusReference(t, tester, button)
				if gesture == "enter" {
					tester.Key(0, ui.KeyEnter)
				} else {
					tester.Key(0, ui.KeySpace)
				}
				tester.Frame()
			}
			if len(opened) != 1 || opened[0] != "targets/a.txt" {
				t.Fatalf("%s reference opens = %#v", gesture, opened)
			}
			want := strings.Replace(before, "# 标题", "# 中文🙂", 1)
			if buffer.String() != want {
				t.Fatalf("%s did not Flush pending composition: %q", gesture, buffer.String())
			}
			if changes != 1 || buffer.History().UndoDepth != 1 || buffer.History().RedoDepth != 0 {
				t.Fatalf("%s Flush history = %#v changes=%d", gesture, buffer.History(), changes)
			}
			if v.referenceActivation.buffer != nil {
				t.Fatal("reference gesture survived activation")
			}
		})
	}
}

func TestReferenceNativeStalePressSourceIdentity(t *testing.T) {
	for _, when := range []string{"release-before-frame", "widget-gone"} {
		t.Run(when, func(t *testing.T) {
			buffer := text.New(referenceViewBody)
			var opened []string
			var reports []string
			v := Adapter().Create(editor.Host{
				Buffer:        buffer,
				OpenReference: func(src string) { opened = append(opened, src) },
				Report:        func(err error) { reports = append(reports, err.Error()) },
			}).(*View)
			tester := ui.NewTester(v.Draw, 1180, 780)
			tester.Frame()
			button := referenceAccess(t, tester, true)
			x, y := button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2
			tester.Press(x, y)
			replaceReferenceBufferText(t, buffer, "src: targets/a.txt", "src: targets/b.txt")
			if when == "widget-gone" {
				tester.Frame()
			}
			tester.Release(x, y)
			tester.Frame()
			if len(opened) != 0 {
				t.Fatalf("stale gesture opened %#v", opened)
			}
			if len(reports) != 1 || !strings.Contains(reports[0], "引用") {
				t.Fatalf("stale %s reports = %#v", when, reports)
			}
			if v.referenceActivation.buffer != nil {
				t.Fatal("stale gesture retained source identity")
			}

			button = referenceAccess(t, tester, true)
			clickReference(t, tester, button)
			if len(opened) != 1 || opened[0] != "targets/b.txt" {
				t.Fatalf("fresh gesture opens = %#v", opened)
			}
		})
	}
}

func TestReferenceSecondVisibleTargetKeepsItsGesture(t *testing.T) {
	body := "# 标题\r\n\r\n```doc-block\r\nid: shared\r\nkind: text\r\ntitle: 同名\r\nmode: reference\r\nsrc: targets/a.txt\r\n---\r\n```\r\n\r\n```doc-block\r\nid: shared\r\nkind: document\r\ntitle: 同名\r\nmode: reference\r\nsrc: targets/b.txt\r\n---\r\n```\r\n"
	for _, gesture := range []string{"pointer", "enter"} {
		t.Run(gesture, func(t *testing.T) {
			buffer := text.New(body)
			var opened []string
			v := Adapter().Create(editor.Host{Buffer: buffer, OpenReference: func(src string) {
				opened = append(opened, src)
			}}).(*View)
			tester := ui.NewTester(v.Draw, 1180, 780)
			tester.Frame()

			first := referenceAccess(t, tester, true)
			switch gesture {
			case "pointer":
				firstMeta, ok := tester.Find("text · reference")
				if !ok {
					t.Fatal("first reference metadata missing")
				}
				secondMeta, ok := tester.Find("document · reference")
				if !ok {
					t.Fatal("second reference metadata missing")
				}
				x := first.Bounds.X + first.Bounds.W/2
				y := secondMeta.Y + first.Bounds.Y - firstMeta.Y + first.Bounds.H/2
				tester.Press(x, y)
				if v.referenceActivation.buffer != buffer || v.referenceActivation.rowID != v.rows[2].id {
					t.Fatalf("second pointer captured %#v, want row %d", v.referenceActivation, v.rows[2].id)
				}
				tester.Release(x, y)
			case "enter":
				focusReference(t, tester, first)
				tester.Key(0, ui.KeyTab)
				tester.Frame()
				tester.Key(0, ui.KeyEnter)
			}
			tester.Frame()
			if len(opened) != 1 || opened[0] != "targets/b.txt" {
				t.Fatalf("%s second same-id/title reference opened %#v", gesture, opened)
			}
		})
	}
}

func TestReferenceHeldLifecycleInvalidationClearsCapture(t *testing.T) {
	t.Run("host-unavailable-after-down", func(t *testing.T) {
		buffer := text.New(referenceViewBody)
		v := Adapter().Create(editor.Host{
			Buffer:        buffer,
			OpenReference: func(string) {},
		}).(*View)
		tester := ui.NewTester(v.Draw, 1180, 780)
		tester.Frame()
		button := referenceAccess(t, tester, true)
		x, y := button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2
		tester.Press(x, y)
		v.host.OpenReference = nil
		if err := buffer.ReplaceUTF16(0, 0, "前置\r\n\r\n"); err != nil {
			t.Fatal(err)
		}
		tester.Frame()
		tester.Release(x, y)
		tester.Frame()
		if v.referenceActivation.buffer != nil || v.referencePointerUpBuffer != nil {
			t.Fatal("host loss retained the held gesture")
		}
		if !tester.HasText(referenceNoHost) {
			t.Fatalf("host loss lost its visible reason: %#v", tester.Texts())
		}
	})

	for _, replacement := range []string{"nil", "new-buffer"} {
		t.Run(replacement, func(t *testing.T) {
			buffer := text.New(referenceViewBody)
			var reports []string
			v := Adapter().Create(editor.Host{
				Buffer:        buffer,
				OpenReference: func(string) {},
				Report:        func(err error) { reports = append(reports, err.Error()) },
			}).(*View)
			tester := ui.NewTester(v.Draw, 1180, 780)
			tester.Frame()
			button := referenceAccess(t, tester, true)
			tester.Press(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
			if replacement == "nil" {
				v.host.Buffer = nil
			} else {
				v.host.Buffer = text.New(buffer.String())
			}
			tester.Frame()
			if len(reports) != 1 || !strings.Contains(reports[0], "引用") {
				t.Fatalf("%s reports = %#v", replacement, reports)
			}
			if v.referenceActivation.buffer != nil || v.referencePointerUpBuffer != nil {
				t.Fatalf("%s retained the held gesture", replacement)
			}
		})
	}

	t.Run("oversize-target-after-down", func(t *testing.T) {
		buffer := text.New(referenceViewBody)
		var opened []string
		var reports []string
		v := Adapter().Create(editor.Host{
			Buffer: buffer,
			Report: func(err error) { reports = append(reports, err.Error()) },
			OpenReference: func(src string) {
				opened = append(opened, src)
			},
		}).(*View)
		tester := ui.NewTester(v.Draw, 1180, 780)
		tester.Frame()
		button := referenceAccess(t, tester, true)
		x, y := button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2
		tester.Press(x, y)
		replaceReferenceBufferText(t, buffer, "targets/a.txt", strings.Repeat("a", maxReferenceTargetBytes+1))
		tester.Release(x, y)
		tester.Frame()
		if len(opened) != 0 || len(reports) != 1 || !strings.Contains(reports[0], "引用") {
			t.Fatalf("oversize held target opened/report = %#v %#v", opened, reports)
		}
		if v.referenceActivation.buffer != nil || v.referencePointerUpBuffer != nil {
			t.Fatal("oversize held target retained the gesture")
		}
	})

	t.Run("partial-source-after-down", func(t *testing.T) {
		buffer := text.New(referenceViewBody)
		var opened []string
		var reports []string
		v := Adapter().Create(editor.Host{
			Buffer:        buffer,
			OpenReference: func(src string) { opened = append(opened, src) },
			Report:        func(err error) { reports = append(reports, err.Error()) },
		}).(*View)
		tester := ui.NewTester(v.Draw, 1180, 780)
		tester.Frame()
		button := referenceAccess(t, tester, true)
		x, y := button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2
		tester.Press(x, y)
		if err := buffer.ReplaceUTF16(0, 0, strings.Repeat("段落\r\n\r\n", 8200)); err != nil {
			t.Fatal(err)
		}
		tester.Release(x, y)
		tester.Frame()
		if len(opened) != 0 || len(reports) != 1 || !strings.Contains(reports[0], "引用") {
			t.Fatalf("partial held source opened/report = %#v %#v", opened, reports)
		}
		if v.referenceActivation.buffer != nil || v.referencePointerUpBuffer != nil {
			t.Fatal("partial held source retained the gesture")
		}
	})

	t.Run("release-outside-drops-pending-and-refreshes", func(t *testing.T) {
		buffer := text.New(referenceViewBody)
		var opened []string
		var reports []string
		v := Adapter().Create(editor.Host{
			Buffer:        buffer,
			OpenReference: func(src string) { opened = append(opened, src) },
			Report:        func(err error) { reports = append(reports, err.Error()) },
		}).(*View)
		tester := ui.NewTester(v.Draw, 1180, 780)
		tester.Frame()
		button := referenceAccess(t, tester, true)
		x, y := button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2
		tester.Press(x, y)
		if err := buffer.ReplaceUTF16(0, 0, "前置\r\n\r\n"); err != nil {
			t.Fatal(err)
		}
		tester.Frame()
		tester.Release(button.Bounds.X+button.Bounds.W*3/2, y)
		tester.Frame()
		tester.Frame()
		if len(opened) != 0 || len(reports) != 0 {
			t.Fatalf("release outside opened/reported = %#v %#v", opened, reports)
		}
		if v.referenceActivation.buffer != nil || v.referencePointerUpBuffer != nil ||
			v.version != buffer.Version() {
			t.Fatal("release outside retained a gesture or stale preview")
		}
	})

	t.Run("virtualized-out-cancels", func(t *testing.T) {
		body := referenceViewBody + strings.Repeat("后续段落\r\n\r\n", 80)
		buffer := text.New(body)
		var opened []string
		var reports []string
		v := Adapter().Create(editor.Host{
			Buffer:        buffer,
			OpenReference: func(src string) { opened = append(opened, src) },
			Report:        func(err error) { reports = append(reports, err.Error()) },
		}).(*View)
		tester := ui.NewTester(v.Draw, 760, 480)
		tester.Frame()
		button := referenceAccess(t, tester, true)
		x, y := button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2
		tester.Press(x, y)
		tester.Scroll(x, y, 0, 1000)
		tester.Frame()
		if len(opened) != 0 || len(reports) != 0 {
			t.Fatalf("virtualized-out opened/reported = %#v %#v", opened, reports)
		}
		if v.referenceActivation.buffer != nil || v.referencePointerUpBuffer != nil {
			t.Fatal("virtualized-out retained the canceled gesture")
		}
	})
}

func TestReferenceUnrelatedInsertionRebasesAndSameBlockEditRefuses(t *testing.T) {
	for _, interveningFrame := range []bool{false, true} {
		t.Run(fmt.Sprintf("insertion-before/frame=%t", interveningFrame), func(t *testing.T) {
			buffer := text.New(referenceViewBody)
			var opened []string
			v := Adapter().Create(editor.Host{Buffer: buffer, OpenReference: func(src string) {
				opened = append(opened, src)
			}}).(*View)
			tester := ui.NewTester(v.Draw, 1180, 780)
			tester.Frame()
			button := referenceAccess(t, tester, true)
			x, y := button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2
			tester.Press(x, y)
			if err := buffer.ReplaceUTF16(0, 0, "前置段落\r\n\r\n"); err != nil {
				t.Fatal(err)
			}
			if interveningFrame {
				tester.Frame()
			}
			tester.Release(x, y)
			tester.Frame()
			if len(opened) != 1 || opened[0] != "targets/a.txt" {
				t.Fatalf("unrelated insertion did not rebase exact target: opened=%#v captured=%#v rows=%#v", opened, v.referenceActivation, v.rows)
			}
		})
	}

	t.Run("same-block-payload", func(t *testing.T) {
		buffer := text.New(referenceViewBody)
		var opened []string
		var reports []string
		v := Adapter().Create(editor.Host{
			Buffer:        buffer,
			OpenReference: func(src string) { opened = append(opened, src) },
			Report:        func(err error) { reports = append(reports, err.Error()) },
		}).(*View)
		tester := ui.NewTester(v.Draw, 1180, 780)
		tester.Frame()
		button := referenceAccess(t, tester, true)
		x, y := button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2
		tester.Press(x, y)
		replaceReferenceBufferText(t, buffer, "引用载荷", "被改写")
		if len(reports) != 0 {
			t.Fatalf("release-time refusal did not wait for the physical release: %#v", reports)
		}
		tester.Release(x, y)
		tester.Frame()
		if len(opened) != 0 || len(reports) != 1 || !strings.Contains(reports[0], "引用") {
			t.Fatalf("same block rewrite opened/refused incorrectly: %#v %#v", opened, reports)
		}
	})
}

func TestReferenceTargetBoundDisplayAndDiagnostics(t *testing.T) {
	for _, size := range []int{maxReferenceTargetBytes, maxReferenceTargetBytes + 1} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			target := strings.Repeat("a", size)
			body := "# 标题\n\n```doc-block\nid: long\nkind: text\nmode: reference\nsrc: " + target + "\n---\n```\n"
			buffer := text.New(body)
			var reports []string
			v := Adapter().Create(editor.Host{
				Buffer:        buffer,
				OpenReference: func(string) {},
				Report:        func(err error) { reports = append(reports, err.Error()) },
			}).(*View)
			tester := ui.NewTester(v.Draw, 1180, 780)
			tester.Frame()
			enabled := size == maxReferenceTargetBytes
			info := referenceAccess(t, tester, enabled)
			if strings.Contains(info.Name, target) {
				t.Fatalf("access label retained unbounded target: %#v", info)
			}
			if !enabled {
				clickReference(t, tester, info)
				if !tester.HasText("引用不可用") {
					t.Fatalf("oversize reason missing: %#v", tester.Texts())
				}
				tester.Frame()
				if len(reports) != 0 {
					t.Fatalf("disabled control reported activation: %#v", reports)
				}
			}
			if v.host.Buffer.String() != body {
				t.Fatal("display or disabled path changed complete source")
			}
		})
	}

	t.Run("inline-src", func(t *testing.T) {
		body := "# 标题\n\n```doc-block\nid: inline\nkind: note\nmode: inline\nsrc: hidden.txt\n---\n载荷\n```\n"
		v := Adapter().Create(editor.Host{Buffer: text.New(body), OpenReference: func(string) {}}).(*View)
		tester := ui.NewTester(v.Draw, 760, 480)
		tester.Frame()
		if _, ok := tester.AccessByRole(ui.RoleButton, "打开引用"); ok {
			t.Fatal("inline block with src exposed a reference button")
		}
		if !tester.HasText("引用头不可用") || !tester.HasText("载荷") {
			t.Fatalf("inline-src diagnosis/payload missing: %#v", tester.Texts())
		}
	})

	t.Run("rune-safe-tooltip", func(t *testing.T) {
		value := strings.Repeat("界", maxReferenceTargetBytes/3+20)
		got := referenceTooltip("打开引用：", value)
		if len(got) > maxReferenceTooltipBytes || !strings.HasSuffix(got, "…") || !utf8.ValidString(got) {
			t.Fatalf("tooltip bound/runes = %d %q", len(got), got)
		}
	})
}

func TestReferenceFailedFlushRetainsDraft(t *testing.T) {
	buffer := text.New(referenceViewBody)
	var opened []string
	var reports []string
	v := Adapter().Create(editor.Host{
		Buffer:        buffer,
		OpenReference: func(src string) { opened = append(opened, src) },
		Report:        func(err error) { reports = append(reports, err.Error()) },
	}).(*View)
	tester := ui.NewTester(v.Draw, 1180, 780)
	beginDraft(t, v, tester, "标题")
	tester.Command("selectAll")
	tester.Compose("保留草稿🙂", 5)
	replaceReferenceBufferText(t, buffer, "标题", "外部")
	tester.Frame()
	draft := v.draft
	version, selection, history := buffer.Version(), buffer.Selection(), buffer.History()
	button := referenceAccess(t, tester, true)
	clickReference(t, tester, button)

	if len(opened) != 0 || v.draft != draft || buffer.Version() != version ||
		buffer.Selection() != selection || buffer.History() != history {
		t.Fatal("failed reference Flush opened or changed authoritative state")
	}
	if len(reports) == 0 || !strings.Contains(strings.Join(reports, "\n"), "草稿仍保留") {
		t.Fatalf("failed Flush refusal = %#v", reports)
	}
	if !tester.HasText("完成编辑") || !tester.HasText("打开未同步副本") {
		t.Fatalf("failed Flush lost recovery controls: %#v", tester.Texts())
	}
}

func TestReferenceCallbackReleaseAndUnavailableStates(t *testing.T) {
	t.Run("release-in-open", func(t *testing.T) {
		buffer := text.New(referenceViewBody)
		var v *View
		opened := 0
		var releasedDocCleared, releasedGestureCleared bool
		v = Adapter().Create(editor.Host{
			Buffer: buffer,
			OpenReference: func(src string) {
				opened++
				if src != "targets/a.txt" {
					t.Fatalf("callback target = %q", src)
				}
				v.Release()
				releasedDocCleared = v.docBuffer == nil
				releasedGestureCleared = v.referenceActivation.buffer == nil
			},
		}).(*View)
		tester := ui.NewTester(v.Draw, 1180, 780)
		tester.Frame()
		button := referenceAccess(t, tester, true)
		clickReference(t, tester, button)
		tester.Frame()
		if opened != 1 || !releasedDocCleared || !releasedGestureCleared ||
			v.referenceActivation.buffer != nil {
			t.Fatal("synchronous Release retained callback state")
		}
	})

	t.Run("host-unavailable", func(t *testing.T) {
		v := Adapter().Create(editor.Host{Buffer: text.New(referenceViewBody)}).(*View)
		tester := ui.NewTester(v.Draw, 760, 480)
		tester.Frame()
		referenceAccess(t, tester, false)
		if !tester.HasText(referenceNoHost) {
			t.Fatalf("host unavailable reason missing: %#v", tester.Texts())
		}
	})
}

func TestReferenceCapturedHeaderRangeAndUncertainJournal(t *testing.T) {
	buffer := text.New(referenceViewBody)
	v := Adapter().Create(editor.Host{Buffer: buffer, OpenReference: func(string) {}}).(*View)
	tester := ui.NewTester(v.Draw, 1180, 780)
	tester.Frame()
	block := v.doc.Blocks[2]
	row := v.rows[2]
	target, ok := referenceTargetOf(previewSnapshot{v.doc, v.rows, v.docBuffer, v.version}, row, block)
	if !ok {
		t.Fatal("reference target was not capturable")
	}
	header, _ := buffer.SliceUTF16(target.header.Start(), target.header.End())
	if header != "id: shared\r\nkind: text\r\ntitle: 同名\r\nmode: reference\r\nsrc: targets/a.txt\r\n" {
		t.Fatalf("header UTF16 range = %q", header)
	}

	original := buffer.String()
	for _, change := range []string{"restore", "expired", "buffer", "group"} {
		t.Run(change, func(t *testing.T) {
			current := text.New(original)
			v.host.Buffer = current
			captured := target
			captured.buffer = current
			captured.version = current.Version()
			switch change {
			case "restore":
				snapshot := current.Snapshot()
				_ = current.ReplaceUTF16(target.enclosing.Start(), target.enclosing.End(), "changed")
				current.Restore(snapshot)
			case "expired":
				for i := 0; i < 65; i++ {
					_ = current.ReplaceUTF16(current.LenUTF16(), current.LenUTF16(), "x")
				}
			case "buffer":
				v.host.Buffer = text.New(current.String())
			case "group":
				_ = current.BeginGroup()
			}
			before := current.String()
			version, selection, history := current.Version(), current.Selection(), current.History()
			err := v.activateReference(captured)
			if change == "group" {
				if !errors.Is(err, text.ErrGroup) {
					t.Fatalf("%s reference error = %v", change, err)
				}
			} else if !errors.Is(err, errReferenceStale) {
				t.Fatalf("%s reference error = %v", change, err)
			}
			if current.String() != before || current.Version() != version ||
				current.Selection() != selection || current.History() != history {
				t.Fatalf("%s refusal changed authoritative state", change)
			}
			if change == "group" {
				current.EndGroup()
			}
			v.host.Buffer = current
		})
	}

	if err := v.activateReference(referenceTarget{src: strings.Repeat("x", maxReferenceTargetBytes+1)}); !errors.Is(err, errReferenceStale) && err == nil {
		t.Fatal("over-limit captured target was accepted")
	}
}
