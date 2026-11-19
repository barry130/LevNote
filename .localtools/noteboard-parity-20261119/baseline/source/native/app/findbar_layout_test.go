package app

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestFindbarControlsFitMainPaneAtMinimumWindow(t *testing.T) {
	cases := []struct {
		name     string
		dark     bool
		fontSize float32
	}{
		{name: "light-default", fontSize: 16},
		{name: "dark-default", dark: true, fontSize: 16},
		{name: "light-maximum-editor-font", fontSize: 32},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := newTestApp(t, t.TempDir())
			a.Dark = tc.dark
			a.FontSize = tc.fontSize
			a.NewDocument("text")
			a.ShowFind = true
			tester := ui.NewTester(a.View, 760, 480)
			tester.Frame()
			rects := findbarControlRects(t, a, tester)
			for _, label := range findbarControlLabels {
				t.Logf("%s control %q: %#v", tc.name, label, rects[label])
			}
			editor, ok := tester.Find("文档编辑区")
			if !ok || editor.W < 200 || editor.H < 100 ||
				editor.X+editor.W > 760 || editor.Y+editor.H > 480-29 {
				t.Fatalf("%s leaves an unusable editor area at 760x480: %#v", tc.name, editor)
			}
			t.Logf("%s editor area: %#v", tc.name, editor)
		})
	}
}

var findbarControlLabels = []string{
	"查找内容",
	"上一个",
	"下一个",
	"区分大小写",
	"正则",
	"全词",
	"关闭查找",
	"替换内容",
	"替换当前",
	"全部替换",
	"搜索工作区",
}

func findbarControlRects(t *testing.T, a *App, tester *ui.Tester) map[string]ui.Rect {
	t.Helper()
	labels := findbarControlLabels
	rects := make(map[string]ui.Rect, len(labels))
	mainLeft := 1 + a.SidebarWidth
	const statusTop = 480 - 29
	for _, label := range labels {
		info, ok := tester.Access(label)
		if !ok {
			t.Fatalf("Find-bar control %q is missing: %#v", label, tester.Texts())
		}
		rect := info.Bounds
		rects[label] = rect
		if rect.W <= 0 || rect.H <= 0 ||
			rect.X < mainLeft || rect.X+rect.W > 760 ||
			rect.Y < 0 || rect.Y+rect.H > statusTop {
			t.Fatalf("Find-bar control %q is outside the main pane above status at 760x480: %#v", label, rect)
		}
	}
	for i := range labels {
		for j := i + 1; j < len(labels); j++ {
			left, right := rects[labels[i]], rects[labels[j]]
			if left.X < right.X+right.W && right.X < left.X+left.W &&
				left.Y < right.Y+right.H && right.Y < left.Y+left.H {
				t.Fatalf("Find-bar controls %q and %q overlap at 760x480: %#v %#v",
					labels[i], labels[j], left, right)
			}
		}
	}
	return rects
}

func TestFindbarActionsAndFocusSurviveMinimumReflow(t *testing.T) {
	a, tab := newSearchTextApp(t, "needle needle")
	a.ShowFind = true
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()

	if err := tester.Click("查找内容"); err != nil {
		t.Fatal(err)
	}
	tester.Type("needle")
	tester.Frame()
	if err := tester.Click("替换内容"); err != nil {
		t.Fatal(err)
	}
	tester.Type("match")
	tester.Frame()
	if a.Query != "needle" || a.Replacement != "match" {
		t.Fatalf("typed Find-bar text = %q / %q", a.Query, a.Replacement)
	}

	if err := tester.Click("查找内容"); err != nil {
		t.Fatal(err)
	}
	tester.Compose("猫", 1)
	beforeResize, ok := tester.AccessByRole(ui.RoleTextField, "查找内容")
	if !ok || !beforeResize.Focused {
		t.Fatalf("query composition did not retain focus before reflow: %#v", beforeResize)
	}
	tester.SetSize(760, 480)
	tester.Frame()
	tester.Frame()
	afterResize, ok := tester.AccessByRole(ui.RoleTextField, "查找内容")
	if !ok || !afterResize.Focused || afterResize.Value != beforeResize.Value ||
		a.Query != "needle" || a.Replacement != "match" {
		t.Fatalf("minimum reflow reset query/replacement or focus: before=%#v after=%#v query=%q replacement=%q",
			beforeResize, afterResize, a.Query, a.Replacement)
	}
	findbarControlRects(t, a, tester)

	tester.Type("猫")
	tester.Frame()
	if a.Query != "needle猫" {
		t.Fatalf("composition after reflow = %q, want needle猫", a.Query)
	}
	a.Query = "needle"
	tester.Frame()

	if err := tester.Click("下一个"); err != nil {
		t.Fatal(err)
	}
	assertSearchSelection(t, tab, 0, 6)
	if err := tester.Click("上一个"); err != nil {
		t.Fatal(err)
	}
	assertSearchSelection(t, tab, 7, 13)

	if err := tester.Click("区分大小写"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if !a.CaseSensitive {
		t.Fatal("case-sensitive option did not toggle")
	}
	if err := tester.Click("正则"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if !a.Regex {
		t.Fatal("regex option did not toggle")
	}
	if err := tester.Click("全词"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if !a.WholeWord {
		t.Fatal("whole-word option did not toggle")
	}

	tester.ClickAt(rectCenter(findbarControlRects(t, a, tester)["替换当前"]))
	tester.Frame()
	if tab.Doc.Buffer.String() != "needle match" {
		t.Fatalf("replace-current result = %q", tab.Doc.Buffer.String())
	}

	if err := tester.Click("关闭查找"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if a.ShowFind || tester.HasText("查找内容") {
		t.Fatal("close action did not dismiss the Find bar")
	}
}

func TestFindbarTabFocusAtMinimumWindow(t *testing.T) {
	a, _ := newSearchTextApp(t, "needle")
	a.ShowFind = true
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	if err := tester.Click("查找内容"); err != nil {
		t.Fatal(err)
	}
	tester.Key(0, ui.KeyTab)
	tester.Frame()
	if !tester.Focused("上一个") {
		t.Fatalf("Tab from query did not focus previous: %#v", tester.Texts())
	}
	for _, label := range findbarControlLabels[2:] {
		tester.Key(0, ui.KeyTab)
		tester.Frame()
		if !tester.Focused(label) {
			t.Fatalf("Tab traversal did not reach %q", label)
		}
	}
}

func TestFindbarWholeWordEscapeAtMinimumWindow(t *testing.T) {
	a, _ := newSearchTextApp(t, "needle")
	a.ShowFind = true
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	if err := tester.Click("查找内容"); err != nil {
		t.Fatal(err)
	}
	for range 5 {
		tester.Key(0, ui.KeyTab)
		tester.Frame()
	}
	if !tester.Focused("全词") {
		t.Fatalf("whole-word checkbox did not retain focus: %#v", tester.Texts())
	}
	tester.Key(0, ui.KeyEscape)
	tester.Frame()
	if a.ShowFind || tester.HasText("全词") {
		t.Fatal("Escape from whole-word checkbox did not close Find")
	}
}

func TestFindbarLayoutScreenshots(t *testing.T) {
	out := os.Getenv("LEVNOTE_FINDBAR_LAYOUT_SCREENSHOT_DIR")
	if out == "" {
		t.Skip("set LEVNOTE_FINDBAR_LAYOUT_SCREENSHOT_DIR for Find-bar layout captures")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		dark bool
	}{
		{name: "light"},
		{name: "dark", dark: true},
	} {
		a := newTestApp(t, t.TempDir())
		a.Dark = tc.dark
		a.NewDocument("text")
		tab := a.Current()
		if err := tab.Doc.Buffer.ReplaceUTF16(0, 0, "alpha needle beta needle"); err != nil {
			t.Fatal(err)
		}
		a.ShowFind = true
		a.Query = "needle"
		a.Replacement = "match"
		a.WholeWord = true
		a.find(true)
		tester := ui.NewTester(a.View, 760, 480)
		tester.Frame()
		tester.Frame()
		file, err := os.Create(filepath.Join(out, "findbar-"+tc.name+"-760x480.png"))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(file, tester.Image()); err != nil {
			file.Close()
			t.Fatal(err)
		}
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func rectCenter(rect ui.Rect) (float32, float32) {
	return rect.X + rect.W/2, rect.Y + rect.H/2
}
