package markdownview

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"
	"unsafe"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/markdown"
	"levnote.local/native/ui/editor"
)

func outlineItems(t *testing.T, source editor.OutlineSource) editor.OutlineSnapshot {
	t.Helper()
	snapshot, err := source.OutlineSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func TestOutlineIdentityMapsDuplicateInsertionsAndUTF16(t *testing.T) {
	buffer := text.New("🙂前文\r\n\r\n# A\r\n# A\r\n")
	host := editor.Host{Buffer: buffer, Changed: func() { t.Fatal("inspection mutated document") }}
	source := Adapter().Outline(host)
	snapshot := outlineItems(t, source)
	if snapshot.Items[0].Offset != text.UTF16Len("🙂前文\r\n\r\n") || snapshot.Items[1].AnchorID != "a-2" || snapshot.Items[1].Level != 1 {
		t.Fatal(snapshot)
	}
	old := snapshot.Items[1]
	insert := "# A\r\n"
	if err := buffer.ReplaceUTF16(old.Offset, old.Offset, insert); err != nil {
		t.Fatal(err)
	}
	mapped, err := source.ResolveOutlineItem(old.ID)
	if err != nil || mapped.Offset != old.Offset+text.UTF16Len(insert) || mapped.AnchorID != "a-3" || mapped.ID == old.ID {
		t.Fatalf("duplicate silently retargeted: %+v %v", mapped, err)
	}
	view := Adapter().Create(host).(*View)
	if err := view.NavigateOutlineItem(old.ID); err != nil {
		t.Fatal("independent mounted view cannot resolve source identity", err)
	}
	if current, ok := view.CurrentOutlineItem(); !ok || current != mapped.ID {
		t.Fatal("preview current target lost", current, mapped.ID)
	}
	view.Release()
	if err := view.NavigateOutlineItem(old.ID); err != nil {
		t.Fatal("release invalidated same-buffer identity", err)
	}
	recreated := Adapter().Create(host).(*View)
	if err := recreated.NavigateOutlineItem(old.ID); err != nil {
		t.Fatal("recreated view rejected same-buffer identity", err)
	}
	if !buffer.Undo() {
		t.Fatal("insertion undo")
	}
	back, err := source.ResolveOutlineItem(old.ID)
	if err != nil || back.Offset != old.Offset || back.AnchorID != "a-2" {
		t.Fatal("outside edit undo should map original target", back, err)
	}
}

func TestOutlineDuplicateInsertedByFlushDoesNotRetarget(t *testing.T) {
	buffer := text.New("正文\n\n# A\n# A\n")
	source := Adapter().Outline(editor.Host{Buffer: buffer})
	old := outlineItems(t, source).Items[1]
	inserted := false
	v := Adapter().Create(editor.Host{Buffer: buffer, Changed: func() {
		if !inserted {
			inserted = true
			_ = buffer.ReplaceUTF16(0, 0, "# A\n")
		}
	}}).(*View)
	tester := ui.NewTester(v.Draw, 720, 640)
	beginDraft(t, v, tester, "正文")
	tester.Command("selectAll")
	tester.Compose("新🙂正文", 4)
	if err := v.NavigateOutlineItem(old.ID); err != nil {
		t.Fatal(err)
	}
	item, err := source.ResolveOutlineItem(old.ID)
	if err != nil || item.AnchorID != "a-3" || item.Offset != text.UTF16Len("# A\n新🙂正文\n\n# A\n") {
		t.Fatal("flush insertion retargeted duplicate", item, err)
	}
}

func TestOutlineTargetEditedByCompositionFailsAfterSynchronizingDraft(t *testing.T) {
	buffer := text.New("# Same\n# Same\n")
	source := Adapter().Outline(editor.Host{Buffer: buffer})
	old := outlineItems(t, source).Items[0]
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	tester := ui.NewTester(v.Draw, 720, 640)
	beginDraft(t, v, tester, "Same")
	tester.Command("selectAll")
	tester.Compose("changed🙂", 8)
	draft := v.draft
	if err := v.NavigateOutlineItem(old.ID); !errors.Is(err, editor.ErrOutlineStale) {
		t.Fatal("changed heading silently resolved by label/position", err)
	}
	if v.draft != draft || v.pendingDraft() || buffer.String() != "# changed🙂\n# Same\n" {
		t.Fatal("synchronized draft should stay available after stale navigation")
	}
}

func TestOutlineParserPartialKeepsSourceOffsetsButRejectsPreviewNavigation(t *testing.T) {
	buffer := text.New(strings.Repeat("# A\n", 8200))
	source := Adapter().Outline(editor.Host{Buffer: buffer})
	snapshot := outlineItems(t, source)
	if !snapshot.Truncated || snapshot.Items[0].Offset != 0 {
		t.Fatal("partial source outline lost truthful descriptor", snapshot)
	}
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	if err := v.NavigateOutlineItem(snapshot.Items[0].ID); !errors.Is(err, editor.ErrOutlineUnsupported) {
		t.Fatal("partial preview fabricated a visible target", err)
	}
}

func TestOutlineRejectsEditedExpiredRestoredOrReplacedTargets(t *testing.T) {
	for _, action := range []string{"edit", "expiry", "restore", "replace", "malformed"} {
		t.Run(action, func(t *testing.T) {
			buffer := text.New("# Same\n# Same\n")
			source := Adapter().Outline(editor.Host{Buffer: buffer}).(*outlineSource)
			old := outlineItems(t, source).Items[1]
			switch action {
			case "edit":
				_ = buffer.ReplaceUTF16(old.Offset+2, old.Offset+3, "X")
			case "expiry":
				for i := 0; i < 65; i++ {
					_ = buffer.ReplaceUTF16(buffer.LenUTF16(), buffer.LenUTF16(), "x")
				}
			case "restore":
				buffer.Restore(buffer.Snapshot())
			case "replace":
				source.buffer = text.New(buffer.String())
			case "malformed":
				old.ID += ":unexpected"
			}
			if _, err := source.ResolveOutlineItem(old.ID); !errors.Is(err, editor.ErrOutlineStale) {
				t.Fatal("ambiguous target did not fail", err)
			}
		})
	}
}

func TestOutlineBoundedCopiedCacheAndPreMaterializationGuard(t *testing.T) {
	buffer := text.New("# " + strings.Repeat("中🙂", 300) + "\n" + strings.Repeat("# same\n", 2100))
	source := Adapter().Outline(editor.Host{Buffer: buffer}).(*outlineSource)
	snapshot := outlineItems(t, source)
	if !snapshot.Truncated || len(snapshot.Items) != editor.MaxOutlineItems || len(snapshot.Items[0].Text) > editor.MaxOutlineLabelBytes || !utf8.ValidString(snapshot.Items[0].Text) {
		t.Fatal("unbounded outline", len(snapshot.Items), len(snapshot.Items[0].Text))
	}
	snapshot.Items[0].Text = "caller-owned"
	second := outlineItems(t, source)
	if second.Items[0].Text == "caller-owned" {
		t.Fatal("caller mutated cached slice")
	}
	// The descriptor label owns a bounded allocation instead of sharing the
	// huge original string. This also covers the short-label Clone branch.
	raw := "# short\n" + strings.Repeat("padding", 10000)
	label, clipped := editor.OutlineLabel(raw[2:7])
	base, data := uintptr(unsafe.Pointer(unsafe.StringData(raw))), uintptr(unsafe.Pointer(unsafe.StringData(label)))
	if clipped || data >= base && data < base+uintptr(len(raw)) {
		t.Fatal("small label retains full raw input")
	}
	source.Release()
	if source.loaded || source.entries != nil || source.cachedBuffer != nil {
		t.Fatal("release retained outline cache")
	}
	source.buffer = text.New(strings.Repeat("x", markdown.MaxParseBytes+1))
	if _, err := source.OutlineSnapshot(); !errors.Is(err, editor.ErrOutlineUnsupported) || source.entries != nil {
		t.Fatal("over-budget input not rejected", err)
	}
}

func TestOutlineNavigationFlushesNativeCompositionAndRejectsConflict(t *testing.T) {
	for _, conflict := range []bool{false, true} {
		t.Run(fmt.Sprint(conflict), func(t *testing.T) {
			buffer := text.New("# First\n\n正文\n\n# Target\n")
			v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
			source := Adapter().Outline(editor.Host{Buffer: buffer})
			target := outlineItems(t, source).Items[1]
			tester := ui.NewTester(v.Draw, 720, 640)
			beginDraft(t, v, tester, "正文")
			tester.Command("selectAll")
			tester.Compose("中文🙂", 3)
			draft := v.draft
			if conflict {
				_ = buffer.ReplaceUTF16(v.editSpan.Start(), v.editSpan.End(), "外部改动")
			}
			err := v.NavigateOutlineItem(target.ID)
			if conflict {
				if !errors.Is(err, errDraftConflict) || v.draft != draft || !v.pendingDraft() || strings.Contains(buffer.String(), "中文🙂") {
					t.Fatal("conflict lost draft or navigated", err)
				}
				return
			}
			if err != nil || v.draft != nil || !strings.Contains(buffer.String(), "中文🙂") || buffer.History().UndoDepth != 1 {
				t.Fatal("navigation lost composition", err, buffer.String())
			}
			resolved, err := source.ResolveOutlineItem(target.ID)
			if err != nil || resolved.Offset != text.UTF16Len("# First\n\n中文🙂\n\n") {
				t.Fatal("navigation stale after composition", resolved, err)
			}
		})
	}
}

func TestOutlineFarHeadingVirtualScroll(t *testing.T) {
	var source strings.Builder
	for i := 0; i < 2000; i++ {
		fmt.Fprintf(&source, "# Heading %d\n", i)
	}
	buffer := text.New(source.String())
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	items := outlineItems(t, Adapter().Outline(editor.Host{Buffer: buffer})).Items
	tester := ui.NewTester(v.Draw, 532, 420)
	if err := v.NavigateOutlineItem(items[1999].ID); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	first, last := v.list.Visible()
	if last != 1999 || first < 1950 || v.frameRows > 40 || !tester.HasText("Heading 1999") {
		t.Fatalf("far heading not virtualized/visible: %d..%d rows%d", first, last, v.frameRows)
	}
}

func BenchmarkMarkdownOutline(b *testing.B) {
	buffer := text.New(strings.Repeat("# 中文🙂 repeated\n", 2000))
	for _, cold := range []bool{true, false} {
		b.Run(fmt.Sprint("cold=", cold), func(b *testing.B) {
			source := Adapter().Outline(editor.Host{Buffer: buffer})
			source.OutlineSnapshot()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if cold {
					source.Release()
				}
				source.OutlineSnapshot()
			}
		})
	}
}
