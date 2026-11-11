package markdownview

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

func measuredViewWork(t *testing.T, label string, run func()) {
	t.Helper()
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()
	run()
	elapsed := time.Since(start)
	runtime.ReadMemStats(&after)
	t.Logf("%s elapsed=%s allocated=%d B allocations=%d", label, elapsed, after.TotalAlloc-before.TotalAlloc, after.Mallocs-before.Mallocs)
}

func headingFixture(count int) string {
	var out strings.Builder
	for i := 0; i < count; i++ {
		fmt.Fprintf(&out, "# Heading %04d\n", i)
	}
	return out.String()
}

func listFixture(count int) string {
	var out strings.Builder
	for i := 0; i < count; i++ {
		fmt.Fprintf(&out, "- List item %04d\n", i)
	}
	return out.String()
}

func TestMarkdownFrameResourceProbe(t *testing.T) {
	for _, tc := range []struct{ name, source string }{{"2000 headings", headingFixture(2000)}, {"1000 list items", listFixture(1000)}} {
		t.Run(tc.name, func(t *testing.T) {
			v := Adapter().Create(editor.Host{Buffer: text.New(tc.source)}).(*View)
			measuredViewWork(t, "cold parse", v.refresh)
			var tester *ui.Tester
			measuredViewWork(t, "cold view", func() { tester = ui.NewTester(v.Draw, 800, 600) })
			measuredViewWork(t, "five unchanged frames", func() {
				for i := 0; i < 5; i++ {
					tester.Frame()
				}
			})
			t.Logf("built rows=%d pending-frame=%v", v.frameRows, tester.FramePending())
			if v.frameRows > 60 || tester.FramePending() {
				t.Fatal("unchanged view is not bounded/settled", v.frameRows)
			}
		})
	}
}

func TestVirtualPreviewFarRowsAndSourceInsertionRetainIdentity(t *testing.T) {
	buffer := text.New(headingFixture(2000))
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	tester := ui.NewTester(v.Draw, 800, 600)
	id := v.rows[1700].id
	v.list.ScrollTo(1700, ui.Start)
	tester.Frame()
	if _, ok := tester.Find("Heading 1700"); !ok || v.frameRows > 60 {
		t.Fatal("far row unreachable or unbounded", v.frameRows)
	}
	_ = buffer.ReplaceUTF16(0, 0, "# Inserted🙂\n")
	tester.Frame()
	if v.rows[1701].id != id {
		t.Fatal("outside insertion replaced stable row identity")
	}
	if rect, ok := tester.Find("Heading 1700"); !ok || rect.Y < 0 || rect.Y > 180 {
		t.Fatal("source insertion moved the current anchor", rect, ok)
	}
	beginDraft(t, v, tester, "Heading 1700")
	tester.Command("selectAll")
	tester.Compose("远处中文🙂", 5)
	v.list.ScrollTo(100, ui.Start)
	tester.Frame()
	if _, composing := v.draft.MarkedRange(); !composing {
		t.Fatal("virtual scroll unmounted active IME client")
	}
	if err := v.Flush(); err != nil || !strings.Contains(buffer.String(), "# 远处中文🙂\n") {
		t.Fatal("far draft flush lost text", err)
	}
	tester.Key(0, ui.KeyEscape)
	if v.draft != nil {
		t.Fatal("Escape did not finish independent draft")
	}
	if rect, ok := tester.Find("Heading 0099"); !ok || rect.Y < 0 || rect.Y > 180 {
		t.Fatal("finish lost current preview anchor", rect, ok)
	}
}

func TestVirtualListItemsAndNarrowDraftRemainUsable(t *testing.T) {
	buffer := text.New(listFixture(1000))
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	tester := ui.NewTester(v.Draw, 265, 480)
	v.list.ScrollTo(900, ui.Start)
	tester.Frame()
	beginDraft(t, v, tester, "List item 0900")
	tester.Command("selectAll")
	tester.Compose("- 修改🙂\n", 5)
	v.list.ScrollTo(200, ui.Start)
	tester.Frame()
	first, last := v.list.Visible()
	if last <= first || v.frameRows > 60 {
		t.Fatal("narrow draft left no scrollable preview", first, last, v.frameRows)
	}
	button, ok := tester.Find("完成编辑")
	if !ok || button.Y+button.H > 410 || button.X+button.W > 265 {
		t.Fatal("narrow draft controls out of viewport", button)
	}
	if dir := os.Getenv("LEVNOTE_MARKDOWN_SCREENSHOT_DIR"); dir != "" {
		_ = os.MkdirAll(dir, 0755)
		file, err := os.Create(filepath.Join(dir, "markdown-virtual-draft-narrow.png"))
		if err != nil {
			t.Fatal(err)
		}
		if err = png.Encode(file, tester.Image()); err != nil {
			t.Fatal(err)
		}
		_ = file.Close()
	}
	if err := tester.Click("完成编辑"); err != nil || buffer.String() != "- 修改🙂\n" {
		t.Fatal("virtual list field did not flush full replacement", err, buffer.String())
	}
}

func TestConflictDraftFitsActualSplitContentBounds(t *testing.T) {
	buffer := text.New("# 标题\n\n" + headingFixture(200))
	v := Adapter().Create(editor.Host{Buffer: buffer}).(*View)
	tester := ui.NewTester(func(c *ui.Context) {
		ui.Column(c).Width(760).Height(480).Children(func() {
			ui.Box(c).Height(150).Shrink(0)
			ui.Row(c).Height(300).Shrink(0).Children(func() {
				ui.Box(c).Width(495).Shrink(0)
				ui.Box(c).Width(265).Height(300).Children(func() { v.Draw(c) })
			})
			ui.Text(c, "STATUS BAR").Label("status").Height(30).Shrink(0)
		})
	}, 760, 480)
	beginDraft(t, v, tester, "标题")
	tester.Command("selectAll")
	tester.Compose("冲突草稿🙂", 4)
	_ = buffer.ReplaceUTF16(2, 4, "外部")
	_ = v.Flush()
	tester.Frame()
	for _, label := range []string{"完成编辑", "打开未同步副本", "放弃草稿", "添加段落"} {
		rect, ok := tester.Find(label)
		if !ok || rect.Y < 150 || rect.Y+rect.H > 450 || rect.X < 495 || rect.X+rect.W > 760 {
			t.Fatal("split content/control overpainted shell", label, rect)
		}
	}
	v.list.ScrollTo(120, ui.Start)
	tester.Frame()
	if first, last := v.list.Visible(); last < first || first < 100 || v.draft == nil {
		t.Fatal("conflict fixed panel disabled lower preview scrolling", first, last)
	}
	if dir := os.Getenv("LEVNOTE_MARKDOWN_SCREENSHOT_DIR"); dir != "" {
		_ = os.MkdirAll(dir, 0755)
		file, err := os.Create(filepath.Join(dir, "markdown-conflict-bounded-split.png"))
		if err != nil {
			t.Fatal(err)
		}
		if err = png.Encode(file, tester.Image()); err != nil {
			t.Fatal(err)
		}
		_ = file.Close()
	}
}
