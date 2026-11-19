package graphview

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/drawio"
	"levnote.local/native/ui/editor"
)

func graphResourceMeasure(t *testing.T, label string, run func()) {
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

func largeMindmapFixture(count int) string {
	var out strings.Builder
	out.WriteString(`{"format":"levnote.mindmap","version":1,"layout":"logicalStructure","root":{"id":"root","topic":"Root","children":[`)
	for i := 1; i < count; i++ {
		if i > 1 {
			out.WriteByte(',')
		}
		fmt.Fprintf(&out, `{"id":"node-%d","topic":"Topic %04d"}`, i, i)
	}
	out.WriteString(`]}}`)
	return out.String()
}

func largeDrawioFixture(count int) string {
	var out strings.Builder
	out.WriteString(`<mxfile><diagram id="p1" name="Page"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/>`)
	for i := 0; i < count; i++ {
		fmt.Fprintf(&out, `<mxCell id="n%d" value="Node %04d" vertex="1" parent="1"><mxGeometry x="%d" y="%d" width="100" height="40" as="geometry"/></mxCell>`, i, i, (i%20)*150, (i/20)*80)
	}
	out.WriteString(`</root></mxGraphModel></diagram></mxfile>`)
	return out.String()
}

func TestGraphFrameResourceProbe(t *testing.T) {
	for _, count := range []int{500, 2000} {
		for _, kind := range []documentKind{kindMindmap, kindDrawio} {
			t.Run(fmt.Sprintf("kind%d-%d", kind, count), func(t *testing.T) {
				source := largeMindmapFixture(count)
				if kind == kindDrawio {
					source = largeDrawioFixture(count)
				}
				v := newView(kind, editor.Host{Buffer: text.New(source)})
				graphResourceMeasure(t, "cold parse/layout", func() { v.ensureSynced(nil) })
				var tester *ui.Tester
				graphResourceMeasure(t, "cold fitted view", func() { tester = ui.NewTester(v.Draw, 960, 640) })
				graphResourceMeasure(t, "five fitted unchanged frames", func() {
					for i := 0; i < 5; i++ {
						tester.Frame()
					}
				})
				t.Logf("fitted nodes=%d shaped=%d hidden=%d cached=%d", v.frameNodes, v.frameLabels, v.frameHiddenLabels, len(v.labels.layouts))
				v.zoom, v.panX, v.panY, v.fitted = 1, 20, 20, true
				tester.Frame()
				graphResourceMeasure(t, "five unchanged frames", func() {
					for i := 0; i < 5; i++ {
						tester.Frame()
					}
				})
				t.Logf("nodes drawn=%d labels shaped=%d pending-frame=%v", v.frameNodes, v.frameLabels, tester.FramePending())
				if v.frameNodes > 100 || v.frameLabels != 0 || len(v.labels.layouts) > maxGraphLabelLayouts || tester.FramePending() {
					t.Fatal("viewport work/cache is not bounded or settled")
				}
				graphResourceMeasure(t, "five pan frames", func() {
					for i := 0; i < 5; i++ {
						v.panY -= 40
						tester.Frame()
					}
				})
			})
		}
	}
}

func TestLabelCacheBoundsAndInvalidation(t *testing.T) {
	v := newView(kindDrawio, editor.Host{Buffer: text.New(testDrawio)})
	tester := ui.NewTester(v.Draw, 900, 640)
	v.zoom, v.panX, v.panY, v.fitted = 1, 20, 20, true
	tester.Frame()
	tester.Frame()
	if v.frameLabels != 0 || len(v.labels.layouts) == 0 || v.nodeIndex == nil {
		t.Fatal("steady cache missing")
	}
	v.panX += 1
	tester.Frame()
	if v.frameLabels != 0 {
		t.Fatal("pan unnecessarily reshaped labels")
	}
	tester.SetDark(true)
	if v.frameLabels != 0 {
		t.Fatal("paint-only theme unnecessarily reshaped geometry")
	}
	v.zoom = 1.1
	tester.Frame()
	foundZoom := false
	for _, entry := range v.labels.entries {
		if entry.key.size == 14*v.zoom && entry.layout != nil {
			foundZoom = true
		}
	}
	if !foundZoom {
		t.Fatal("zoom reused incorrect geometry")
	}
	var previous *ui.TextLayout
	for _, entry := range v.labels.entries {
		if entry.key.id == "a" && entry.layout != nil {
			previous = entry.layout
		}
	}
	replaceSource(t, v.host.Buffer, `value="开始"`, `value="changed"`)
	tester.Frame()
	for _, entry := range v.labels.entries {
		if entry.key.id == "a" && entry.layout == previous {
			t.Fatal("changed content retained stale geometry")
		}
	}
	// Text identity is source-version scoped, while font/width/size are in key.
	v.frameLabelUses = 0
	a := v.labelLayout("test", "sans-serif", 14, 100, func() string { return "test" })
	b := v.labelLayout("test", "monospace", 14, 100, func() string { return "test" })
	c := v.labelLayout("test", "sans-serif", 14, 80, func() string { return "test" })
	if a == b || a == c || b == c {
		t.Fatal("font/width ignored")
	}
	for i := 0; i < 500; i++ {
		v.frameLabelUses = 0
		v.labelLayout(fmt.Sprint(i), "sans-serif", 14, 100, func() string { return "cache" })
	}
	if len(v.labels.layouts) > maxGraphLabelLayouts {
		t.Fatal("unbounded label cache")
	}
	selected, zoom, panX := v.selectedID, v.zoom, v.panX
	v.Release()
	if len(v.labels.layouts) != 0 || v.nodeIndex != nil || v.indexPage != nil || v.edgeScratch != nil || selected != v.selectedID || zoom != v.zoom || panX != v.panX {
		t.Fatal("Release retained heavy cache or lost preferences")
	}
}

func TestDenseVisibleLabelsDoNotThrashBoundedCache(t *testing.T) {
	v := newView(kindDrawio, editor.Host{Buffer: text.New(largeDrawioFixture(500))})
	v.ensureSynced(nil)
	for i := range v.currentPage().Model.Nodes {
		v.currentPage().Model.Nodes[i].X, v.currentPage().Model.Nodes[i].Y = 30, 30
	}
	v.zoom, v.panX, v.panY, v.fitted = 1, 0, 0, true
	tester := ui.NewTester(v.Draw, 900, 640)
	tester.Frame()
	if v.frameLabels != 0 || v.frameHiddenLabels == 0 || len(v.labels.layouts) > maxGraphLabelLayouts {
		t.Fatal("all-visible labels thrashed cache", v.frameLabels, v.frameHiddenLabels, len(v.labels.layouts))
	}
	v.selectedID = "n499"
	tester.Frame()
	tester.Frame()
	if v.frameLabels != 0 {
		t.Fatal("new selected label caused persistent eviction thrash")
	}
}

func TestCullingKeepsCrossingEdgesAndHitTesting(t *testing.T) {
	canvas := ui.Rect{X: 0, Y: 0, W: 100, H: 100}
	start, end, ok := clipSegment([2]float32{-100, 50}, [2]float32{200, 50}, canvas)
	if !ok || start[0] != 0 || end[0] != 100 || !polylineVisible([][2]float32{{-100, 50}, {200, 50}}, canvas) {
		t.Fatal("offscreen endpoints hid crossing edge", start, end, ok)
	}
	if _, _, ok := clipSegment([2]float32{-100, -10}, [2]float32{200, -10}, canvas); ok {
		t.Fatal("fully outside segment remained visible")
	}
	v := newView(kindDrawio, editor.Host{Buffer: text.New(testDrawio)})
	v.ensureSynced(nil)
	v.zoom, v.panX, v.panY = 1, -40, 0
	page := v.currentPage()
	v.ensureNodeIndex(page)
	page.Model.Edges = append(page.Model.Edges, drawio.Edge{SourceID: "a", TargetID: "b"})
	points, ok := v.diagramEdgePoints(page.Model.Edges[0], canvas)
	if !ok || !polylineVisible(points, canvas) {
		t.Fatal("indexed edge failed", points)
	}
	if id, ok := v.drawioNodeAt(0, 30); !ok || id != "a" {
		t.Fatal("partial offscreen node cannot be selected", id, ok)
	}
}

func TestLongLabelsAreBoundedWhileFieldKeepsFullText(t *testing.T) {
	value := strings.Repeat("中🙂", 10000)
	source := strings.Replace(testMindmap, `"topic": "中心"`, `"topic": "`+value+`"`, 1)
	v := newView(kindMindmap, editor.Host{Buffer: text.New(source)})
	tester := ui.NewTester(v.Draw, 900, 640)
	if v.draft == nil || v.draft.input.Buffer.String() != value || !strings.Contains(strings.Join(v.Warnings(), " "), "完整文本仍保留") {
		t.Fatal("label truncation lost field or disclosure")
	}
	if label := boundedLabel(value); len(label) > maxGraphLabelBytes+3 || !strings.HasSuffix(label, "…") {
		t.Fatal("label unbounded")
	}
	if err := tester.Click("下一段"); err != nil {
		t.Fatal("bounded small field lost next segment control", err)
	}
	if _, column, _ := v.draft.input.GetPosition(); column < 512 {
		t.Fatal("next segment did not reach full text", column)
	}
	if err := tester.Click("上一段"); err != nil {
		t.Fatal("bounded small field lost previous segment control", err)
	}
	focusDraft(t, tester)
	tester.Compose("替换🙂", 3)
	if err := v.Flush(); err != nil || parseMind(t, v.host.Buffer).Root.Topic != "替换🙂" {
		t.Fatal("long label field could not flush IME", err)
	}
}

func TestPageAndSameVersionBufferReplacementDropCachedGeometry(t *testing.T) {
	first := strings.Replace(testDrawio, `id="p1"`, `id="first"`, 1)
	second := strings.Replace(testDrawio, `id="p1"`, `id="second"`, 1)
	second = strings.Replace(second, `value="开始"`, `value="Second page"`, 1)
	source := strings.TrimSuffix(first, "</mxfile>") + second[strings.Index(second, "<diagram"):]
	v := newView(kindDrawio, editor.Host{Buffer: text.New(source)})
	tester := ui.NewTester(v.Draw, 900, 640)
	page := v.indexPage
	if !v.changePage(1) {
		t.Fatal("page change failed")
	}
	tester.Frame()
	if v.indexPage == page || v.currentLabel() != "Second page" || v.indexPage != v.currentPage() {
		t.Fatal("page switch retained old index/text")
	}
	oldVersion := v.host.Buffer.Version()
	v.host.Buffer = text.New(strings.Replace(testDrawio, `value="开始"`, `value="Replacement buffer"`, 1))
	if v.host.Buffer.Version() != oldVersion {
		t.Fatal("fixture versions differ")
	}
	tester.Frame()
	if v.currentLabel() != "Replacement buffer" || v.syncedBuffer != v.host.Buffer {
		t.Fatal("same-version new buffer reused old parser/cache")
	}
}
