package app

import (
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/formats/drawio"
	"levnote.local/native/formats/markdown"
	"levnote.local/native/formats/mindmap"
)

const (
	oversizeMarkdownSmall = "# 旧预览\n\n仍可切回源码编辑。\n"
	oversizeMindmapSmall  = `{"format":"levnote.mindmap","version":1,"layout":"logicalStructure","root":{"id":"root","topic":"旧中心","children":[]}}`
	oversizeDrawioSmall   = `<mxfile host="levnote"><diagram id="p1" name="旧页面"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="shape" value="旧形状" vertex="1" parent="1"><mxGeometry x="20" y="20" width="120" height="60" as="geometry"/></mxCell></root></mxGraphModel></diagram></mxfile>`
)

type oversizeAdapterCase struct {
	name             string
	title            string
	small            string
	limit            int
	oldVisible       string
	checkRenderedOld bool
}

func oversizedFixture(base string, limit int) string {
	return base + strings.Repeat(" ", limit+1-len(base))
}

func TestOversizeRichViewsSupportSourceToRichModeLifecycle(t *testing.T) {
	cases := []oversizeAdapterCase{
		{
			name: "markdown", title: "oversize.md", small: oversizeMarkdownSmall,
			limit:      markdown.MaxParseBytes,
			oldVisible: "旧预览", checkRenderedOld: true,
		},
		{
			name: "mindmap", title: "oversize.mindmap", small: oversizeMindmapSmall,
			limit:      mindmap.DefaultMaxBytes,
			oldVisible: "旧中心",
		},
		{
			name: "drawio", title: "oversize.drawio", small: oversizeDrawioSmall,
			limit:      drawio.DefaultMaxBytes,
			oldVisible: "旧形状",
		},
	}
	for _, tc := range cases {
		for _, mode := range []string{"preview", "split"} {
			t.Run(tc.name+"/"+mode, func(t *testing.T) {
				a := newTestApp(t, t.TempDir())
				tab := a.add(document.New(tc.small), tc.title, "")
				if tab == nil || tab.Adapter == nil || tab.Mode != "preview" {
					t.Fatalf("small rich document did not open in preview: %#v", tab)
				}
				tester := ui.NewTester(a.View, 760, 480)
				tester.Frame()
				if tc.checkRenderedOld && !tester.HasText(tc.oldVisible) {
					t.Fatalf("small preview did not render %q: %#v", tc.oldVisible, tester.Texts())
				}
				if !a.setMode(tab, "source") {
					t.Fatalf("could not enter source before oversize edit: %s", a.Message)
				}

				source := oversizedFixture(tc.small, tc.limit)
				buffer := tab.Doc.Buffer
				if err := buffer.ReplaceUTF16(0, buffer.LenUTF16(), source); err != nil {
					t.Fatal(err)
				}
				if err := buffer.SetSelection(3, 1); err != nil {
					t.Fatal(err)
				}
				bufferID := buffer
				version, selection, history := buffer.Version(), buffer.Selection(), buffer.History()
				view := tab.View
				if !a.setMode(tab, mode) || tab.Mode != mode {
					t.Fatalf("source -> %s mode transition failed: %s", mode, a.Message)
				}

				tester.Frame()
				tester.Frame()
				if tab.Doc.Buffer != bufferID || buffer.Version() != version ||
					buffer.Selection() != selection || buffer.History() != history ||
					buffer.LenBytes() != len(source) || buffer.String() != source {
					t.Fatalf("oversize refusal changed authoritative source state: id=%v version=%d selection=%v history=%v",
						tab.Doc.Buffer == bufferID, buffer.Version(), buffer.Selection(), buffer.History())
				}
				if tab.View != view {
					t.Fatal("unchanged refusal replaced the mounted view")
				}
				warnings := strings.Join(tab.ViewWarnings, "\n")
				if !strings.Contains(warnings, "预览上限") || !strings.Contains(warnings, "源码") {
					t.Fatalf("oversize refusal lacks source guidance: %q", warnings)
				}
				if tester.HasText(tc.oldVisible) {
					t.Fatalf("stale rich content remained visible after refusal: %#v", tester.Texts())
				}

				a.Message = ""
				a.exportCurrent()
				if !strings.Contains(a.Message, "预览上限") && !(tab.Adapter.ExportEngine != nil && strings.Contains(a.Message, "导出上限")) {
					t.Fatalf("oversize export did not stop on the view error before publication: %q", a.Message)
				}

				if err := buffer.ReplaceUTF16(0, buffer.LenUTF16(), tc.small); err != nil {
					t.Fatal(err)
				}
				tester.Frame()
				if strings.Contains(strings.Join(tab.ViewWarnings, "\n"), "预览上限") ||
					tc.checkRenderedOld && !tester.HasText(tc.oldVisible) {
					t.Fatalf("view did not recover after shrinking: warnings=%q texts=%#v", tab.ViewWarnings, tester.Texts())
				}
				if data, err := tab.View.Export(tab.Adapter.Exports[0]); err != nil || len(data) == 0 {
					t.Fatalf("recovered export = %d bytes, %v", len(data), err)
				}
			})
		}
	}
}

func TestOversizeSourceEditorInputRespectsWritableAndReadonly(t *testing.T) {
	for _, readonly := range []bool{false, true} {
		name := "writable"
		if readonly {
			name = "readonly"
		}
		t.Run(name, func(t *testing.T) {
			a := newTestApp(t, t.TempDir())
			source := oversizedFixture(oversizeMarkdownSmall, markdown.MaxParseBytes)
			tab := a.add(document.New(source), "large.md", "")
			if !a.setMode(tab, "source") {
				t.Fatalf("could not enter source: %s", a.Message)
			}
			tab.Doc.ReadOnly = readonly
			tab.Editor.ReadOnly = readonly
			if !a.revealTab(tab, revealTarget{Start: 1, End: 2}) {
				t.Fatalf("source reveal failed: %s", a.Message)
			}
			tester := ui.NewTester(a.View, 760, 480)
			tester.Frame()
			if err := tester.Click("文档编辑区"); err != nil {
				t.Fatal(err)
			}
			version, selection, size := tab.Doc.Buffer.Version(), tab.Doc.Buffer.Selection(), tab.Doc.Buffer.LenBytes()
			tester.Type("改")
			tester.Frame()
			changed := tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.LenBytes() != size
			if changed == readonly {
				t.Fatalf("%s source input changed=%v, readonly=%v version=%d selection=%v",
					name, changed, readonly, tab.Doc.Buffer.Version(), tab.Doc.Buffer.Selection())
			}
			if readonly && tab.Doc.Buffer.Selection() != selection {
				t.Fatalf("readonly source input moved selection: %v -> %v", selection, tab.Doc.Buffer.Selection())
			}
		})
	}
}

func TestOversizeMarkdownComposeFlushesRangeBeforeRefusal(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	tab := a.add(document.New("# 标题\n\n第二块\n"), "compose.md", "")
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	beginRealMarkdownComposition(t, tab, tester)

	buffer := tab.Doc.Buffer
	fill := markdown.MaxParseBytes + 1 - buffer.LenBytes()
	if fill <= 0 {
		t.Fatal("test fixture already exceeds the Markdown limit")
	}
	end := buffer.LenUTF16()
	if err := buffer.ReplaceUTF16(end, end, strings.Repeat(" ", fill)); err != nil {
		t.Fatal(err)
	}
	if !a.flushTab(tab, "验收过大 Markdown 草稿") {
		t.Fatalf("safe Markdown range flush was rejected: %s", a.Message)
	}
	if !strings.HasPrefix(buffer.String(), "# 中文🙂\n") || buffer.LenBytes() <= markdown.MaxParseBytes {
		t.Fatalf("Markdown range flush was not applied at oversize source: %q", buffer.String()[:64])
	}
	tester.Frame()
	if !strings.Contains(strings.Join(tab.ViewWarnings, "\n"), "预览上限") {
		t.Fatalf("Markdown did not disclose preview refusal after safe flush: %q", tab.ViewWarnings)
	}
}

func TestOversizeGraphDraftFailedModeSwitchRetainsThenRecovers(t *testing.T) {
	cases := []oversizeAdapterCase{
		{
			name: "mindmap", title: "draft.mindmap", small: oversizeMindmapSmall,
			limit: mindmap.DefaultMaxBytes,
		},
		{
			name: "drawio", title: "draft.drawio", small: oversizeDrawioSmall,
			limit: drawio.DefaultMaxBytes,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := newTestApp(t, t.TempDir())
			tab := a.add(document.New(tc.small), tc.title, "")
			tester := ui.NewTester(a.View, 760, 480)
			tester.Frame()
			beginRealGraphComposition(t, tab, tester)
			tester.Frame()

			source := oversizedFixture(tc.small, tc.limit)
			if err := tab.Doc.Buffer.ReplaceUTF16(0, tab.Doc.Buffer.LenUTF16(), source); err != nil {
				t.Fatal(err)
			}
			if err := tab.Doc.Buffer.SetSelection(2, 4); err != nil {
				t.Fatal(err)
			}
			tester.Frame()
			view := tab.View
			version, selection := tab.Doc.Buffer.Version(), tab.Doc.Buffer.Selection()

			if a.setMode(tab, "source") {
				t.Fatal("oversize graph draft unexpectedly allowed mode switch")
			}
			if tab.Mode != "preview" || tab.View != view || tab.Doc.Buffer.Version() != version ||
				tab.Doc.Buffer.Selection() != selection || tab.Doc.Buffer.String() != source {
				t.Fatal("failed graph mode switch changed tab, view, source, version or selection")
			}
			tester.Frame()
			if !tester.HasText("重试写回") || !tester.HasText("恢复为新文档") {
				t.Fatalf("retained graph draft is not recoverable: %#v", tester.Texts())
			}

			if err := tab.Doc.Buffer.ReplaceUTF16(0, tab.Doc.Buffer.LenUTF16(), tc.small); err != nil {
				t.Fatal(err)
			}
			tester.Frame()
			if !a.setMode(tab, "source") {
				t.Fatalf("graph draft did not recover after shrinking: %s", a.Message)
			}
			if !strings.Contains(tab.Doc.Buffer.String(), "中文🙂") {
				t.Fatal("recovery flush did not write the retained graph draft")
			}
		})
	}
}

func TestOversizeGraphDraftCanBeOpenedAsNewDocument(t *testing.T) {
	a := newTestApp(t, t.TempDir())
	tab := a.add(document.New(oversizeMindmapSmall), "recover.mindmap", "")
	tester := ui.NewTester(a.View, 1060, 700)
	tester.Frame()
	beginRealGraphComposition(t, tab, tester)
	tester.Frame()

	source := oversizedFixture(oversizeMindmapSmall, mindmap.DefaultMaxBytes)
	if err := tab.Doc.Buffer.ReplaceUTF16(0, tab.Doc.Buffer.LenUTF16(), source); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if a.setMode(tab, "source") {
		t.Fatal("oversize graph draft unexpectedly allowed mode switch")
	}
	tester.Frame()
	clickTesterText(t, tester, "恢复为新文档")
	tester.Frame()
	if len(a.Tabs) != 2 || a.Current() == tab ||
		!strings.Contains(a.Current().Doc.Buffer.String(), "节点 ID: root") ||
		!strings.Contains(a.Current().Doc.Buffer.String(), "中文🙂") {
		t.Fatalf("graph draft recovery did not create a new document: %#v", a.Current())
	}
	if tab.Doc.Buffer.String() != source {
		t.Fatal("graph draft recovery mutated the oversized authoritative source")
	}
}

func TestAcceptanceOversizePreviewScreenshots(t *testing.T) {
	out := os.Getenv("LEVNOTE_OVERSIZE_SCREENSHOT_DIR")
	if out == "" {
		t.Skip("set LEVNOTE_OVERSIZE_SCREENSHOT_DIR for oversized-preview visual acceptance")
	}
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}

	markdownApp := newTestApp(t, t.TempDir())
	markdownTab := markdownApp.add(document.New(oversizeMarkdownSmall), "large-notes.md", "")
	markdownTester := ui.NewTester(markdownApp.View, 760, 480)
	markdownTester.Frame()
	if !markdownApp.setMode(markdownTab, "source") {
		t.Fatalf("could not enter Markdown source for acceptance: %s", markdownApp.Message)
	}
	if err := markdownTab.Doc.Buffer.ReplaceUTF16(
		0, markdownTab.Doc.Buffer.LenUTF16(),
		oversizedFixture(oversizeMarkdownSmall, markdown.MaxParseBytes),
	); err != nil {
		t.Fatal(err)
	}
	if !markdownApp.setMode(markdownTab, "preview") {
		t.Fatalf("could not enter oversized Markdown preview: %s", markdownApp.Message)
	}
	markdownTester.Frame()
	for _, label := range []string{"源码", "分屏"} {
		rect, ok := markdownTester.Find(label)
		if !ok {
			t.Fatalf("oversize Markdown lost %q source route: %#v", label, markdownTester.Texts())
		}
		if rect.W <= 0 || rect.H <= 0 || rect.X < 0 || rect.X+rect.W > 760 || rect.Y < 0 || rect.Y+rect.H > 480-29 {
			t.Fatalf("oversize Markdown source route %q is not a reachable control: %#v", label, rect)
		}
		t.Logf("oversize Markdown source route %q: %#v", label, rect)
	}
	captureOversizeScreenshot(t, filepath.Join(out, "oversize-markdown-source-route-760x480.png"), markdownTester)

	graphApp := newTestApp(t, t.TempDir())
	graphTab := graphApp.add(document.New(oversizeMindmapSmall), "large-draft.mindmap", "")
	graphTester := ui.NewTester(graphApp.View, 760, 480)
	graphTester.Frame()
	beginRealGraphComposition(t, graphTab, graphTester)
	graphTester.Frame()
	if err := graphTab.Doc.Buffer.ReplaceUTF16(
		0, graphTab.Doc.Buffer.LenUTF16(),
		oversizedFixture(oversizeMindmapSmall, mindmap.DefaultMaxBytes),
	); err != nil {
		t.Fatal(err)
	}
	graphTester.Frame()
	if graphApp.setMode(graphTab, "source") {
		t.Fatal("oversize screenshot draft unexpectedly allowed mode switch")
	}
	graphTester.Frame()
	for _, label := range []string{"重试写回", "恢复为新文档"} {
		rect, ok := graphTester.Find(label)
		if !ok {
			t.Fatalf("oversize graph draft lost %q: %#v", label, graphTester.Texts())
		}
		if rect.W <= 0 || rect.H <= 0 || rect.X < 0 || rect.X+rect.W > 760 || rect.Y < 0 || rect.Y+rect.H > 480-29 {
			t.Fatalf("oversize graph recovery control %q is not a reachable control above the status row: %#v", label, rect)
		}
		t.Logf("oversize graph recovery control %q: %#v", label, rect)
	}
	captureOversizeScreenshot(t, filepath.Join(out, "oversize-graph-draft-recovery-760x480.png"), graphTester)
}

func captureOversizeScreenshot(t *testing.T, path string, tester *ui.Tester) {
	t.Helper()
	file, err := os.Create(path)
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
