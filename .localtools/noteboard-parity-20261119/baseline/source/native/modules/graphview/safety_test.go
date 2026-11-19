package graphview

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"errors"
	"image/png"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/drawio"
	"levnote.local/native/formats/mindmap"
	"levnote.local/native/ui/editor"
)

func focusDraft(t *testing.T, tester *ui.Tester) {
	t.Helper()
	if err := tester.Click("节点文本"); err != nil {
		t.Fatal(err)
	}
	tester.Command("selectAll")
}

func parseMind(t *testing.T, buffer *text.Buffer) *mindmap.Document {
	t.Helper()
	doc, _, err := mindmap.Parse([]byte(buffer.String()), mindmap.ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func replaceSource(t *testing.T, buffer *text.Buffer, old, next string) {
	t.Helper()
	if err := buffer.ReplaceUTF16(0, buffer.LenUTF16(), strings.Replace(buffer.String(), old, next, 1)); err != nil {
		t.Fatal(err)
	}
}

func TestNativeCompositionImmediatelyFlushesBeforeSaveSelectionAndExport(t *testing.T) {
	for _, action := range []string{"flush", "selection", "export", "release", "commit"} {
		t.Run(action, func(t *testing.T) {
			buffer := text.New(testMindmap)
			v := newView(kindMindmap, editor.Host{Buffer: buffer})
			tester := ui.NewTester(v.Draw, 960, 640)
			focusDraft(t, tester)
			tester.Compose("中文🙂", 3)
			if buffer.String() != testMindmap || !v.pendingDraft() {
				t.Fatal("preedit must stay pending independently of authoritative source")
			}
			switch action {
			case "flush":
				if err := editor.FlushView(v); err != nil {
					t.Fatal(err)
				}
			case "selection":
				if !v.selectNode("left") {
					t.Fatal("selection blocked after successful flush")
				}
			case "export":
				if svg, err := v.Export("svg"); err != nil || !bytes.Contains(svg, []byte("中文🙂")) {
					t.Fatal("export omitted pending input", err)
				}
			case "release":
				v.Release()
			case "commit":
				tester.Type("中文🙂")
			}
			if parseMind(t, buffer).Root.Topic != "中文🙂" || buffer.History().UndoDepth != 1 {
				t.Fatalf("composition/save lost text or grouping: %q %+v", parseMind(t, buffer).Root.Topic, buffer.History())
			}
			if !buffer.Undo() || buffer.String() != testMindmap {
				t.Fatal("IME undo did not restore original bytes")
			}
		})
	}
}

func TestGroupConflictRetainsNativeDraftAcrossRefreshSelectionUndoAndRelease(t *testing.T) {
	buffer := text.New(testMindmap)
	var reports []error
	v := newView(kindMindmap, editor.Host{Buffer: buffer, Report: func(err error) { reports = append(reports, err) }})
	tester := ui.NewTester(v.Draw, 960, 640)
	focusDraft(t, tester)
	if err := buffer.BeginGroup(); err != nil {
		t.Fatal(err)
	}
	tester.Type("retain 中文🙂")
	draft := v.draft
	if !errors.Is(v.draftError, text.ErrGroup) || !buffer.InGroup() || buffer.String() != testMindmap {
		t.Fatal("foreign transaction stolen or failure hidden")
	}
	if v.selectNode("left") {
		t.Fatal("selection changed with failed draft")
	}
	v.addMindmapChild()
	v.undo()
	if _, err := v.Export("svg"); !errors.Is(err, text.ErrGroup) {
		t.Fatalf("export ignored pending draft: %v", err)
	}
	v.Release()
	if v.mind != nil || len(v.layout.Nodes) != 0 || v.draft != draft || !v.pendingDraft() || v.selectedID != "root" {
		t.Fatal("release discarded draft or retained heavy cache")
	}
	tester.Frame()
	if v.draft != draft || !tester.HasText("节点草稿尚未写回") || len(reports) == 0 {
		t.Fatal("redraw erased failed input")
	}
	buffer.EndGroup()
	if err := v.Flush(); err != nil {
		t.Fatal(err)
	}
	if parseMind(t, buffer).Root.Topic != "retain 中文🙂" || v.pendingDraft() {
		t.Fatal("retry did not commit retained text")
	}
}

func TestFieldConflictPreservesUnrelatedEditsAndRecoveryFailure(t *testing.T) {
	source := strings.Replace(testMindmap, `"topic": "中心"`, `"topic": "中心", "vendor": {"raw":[1,"二"]}`, 1)
	buffer := text.New(source)
	var recovery string
	recoverFails := true
	v := newView(kindMindmap, editor.Host{Buffer: buffer, Path: "sample.mindmap", RecoverDraft: func(title, content string) error {
		if !strings.HasSuffix(title, ".txt") {
			t.Fatal(title)
		}
		recovery = content
		if recoverFails {
			return errors.New("new document failed")
		}
		return nil
	}})
	tester := ui.NewTester(v.Draw, 960, 640)
	focusDraft(t, tester)
	tester.Compose("local🙂", 6)
	replaceSource(t, buffer, `"topic": "左分支"`, `"topic": "source-left"`)
	tester.Frame()
	if err := v.Flush(); err != nil {
		t.Fatal("unrelated field must not conflict", err)
	}
	doc := parseMind(t, buffer)
	left, _ := doc.FindNode("left")
	if doc.Root.Topic != "local🙂" || left.Topic != "source-left" || !strings.Contains(buffer.String(), `"vendor"`) || !strings.Contains(buffer.String(), `[1,"二"]`) {
		t.Fatal("field merge lost unrelated/unknown metadata", buffer.String())
	}
	focusDraft(t, tester)
	tester.Compose("conflicting 草稿", 14)
	replaceSource(t, buffer, `"topic": "local🙂"`, `"topic": "external"`)
	authority := buffer.String()
	tester.Frame()
	if err := v.Flush(); !errors.Is(err, ErrDraftConflict) {
		t.Fatalf("same field conflict %v", err)
	}
	if v.selectNode("right") {
		t.Fatal("conflict switched selected node")
	}
	if err := v.RecoverDraft(); err == nil || !v.pendingDraft() || buffer.String() != authority {
		t.Fatal("failed recovery discarded draft/changed source")
	}
	recoverFails = false
	if err := v.RecoverDraft(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(recovery, "conflicting 草稿") || buffer.String() != authority || v.pendingDraft() || v.mind != nil {
		t.Fatal("recovery must open separate text and release old heavy caches")
	}
}

func TestParseFailureAndDeletedNodeDoNotReplacePendingDraft(t *testing.T) {
	for _, change := range []string{"invalid", "deleted"} {
		t.Run(change, func(t *testing.T) {
			buffer := text.New(testMindmap)
			v := newView(kindMindmap, editor.Host{Buffer: buffer})
			tester := ui.NewTester(v.Draw, 900, 620)
			v.selectNode("left")
			tester.Frame()
			focusDraft(t, tester)
			tester.Compose("keep", 4)
			if change == "invalid" {
				_ = buffer.ReplaceUTF16(0, buffer.LenUTF16(), "{broken")
			} else {
				doc := parseMind(t, buffer)
				_, _ = doc.DeleteNode("left")
				data, _ := doc.Serialize(mindmap.SerializeOptions{})
				_ = buffer.ReplaceUTF16(0, buffer.LenUTF16(), string(data))
			}
			draft := v.draft
			source := buffer.String()
			tester.Frame()
			v.Warnings()
			v.Release()
			tester.Frame()
			if err := v.Flush(); err == nil {
				t.Fatal("invalid authoritative target accepted")
			}
			if v.draft != draft || v.draft.input.Buffer.String() != "keep" || v.selectedID != "left" || buffer.String() != source {
				t.Fatal("parse/deletion refreshed away pending input")
			}
		})
	}
}

func pageXML(id, label string) string {
	return `<diagram id="` + id + `" name="` + id + `"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="same" value="` + label + `" vertex="1" parent="1" vendor="keep"><mxGeometry x="1" y="2" width="100" height="40" as="geometry"/></mxCell></root></mxGraphModel></diagram>`
}

func TestDrawioDraftTracksPageIDAcrossReorderAndBlocksChangedField(t *testing.T) {
	p1, p2 := pageXML("p1", "first"), pageXML("p2", "second")
	buffer := text.New(`<mxfile>` + p1 + p2 + `</mxfile>`)
	v := newView(kindDrawio, editor.Host{Buffer: buffer})
	tester := ui.NewTester(v.Draw, 960, 640)
	focusDraft(t, tester)
	tester.Compose("第一🙂", 3)
	_ = buffer.ReplaceUTF16(0, buffer.LenUTF16(), `<mxfile>`+p2+p1+`</mxfile>`)
	tester.Frame()
	if err := v.Flush(); err != nil {
		t.Fatal(err)
	}
	doc, _, err := drawio.Parse([]byte(buffer.String()), drawio.ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if doc.Pages[0].Model.Nodes[0].PlainText != "second" || doc.Pages[1].Model.Nodes[0].PlainText != "第一🙂" || v.pageIndex != 1 {
		t.Fatal("stale page index edited different node")
	}
	if !strings.Contains(buffer.String(), `vendor="keep"`) {
		t.Fatal("unknown attr lost")
	}
	focusDraft(t, tester)
	tester.Compose("pending", 7)
	replaceSource(t, buffer, `value="第一🙂"`, `value="external"`)
	tester.Frame()
	if v.changePage(0) || v.pageIndex != 1 || !v.pendingDraft() {
		t.Fatal("page change ignored draft conflict")
	}
	if _, err := v.Export("svg"); !errors.Is(err, ErrDraftConflict) {
		t.Fatal("conflicted export succeeded", err)
	}
}

func TestDrawioSourceChangeCancelsStaleDragAndConnection(t *testing.T) {
	p1, p2 := pageXML("p1", "first"), pageXML("p2", "second")
	buffer := text.New(`<mxfile>` + p1 + p2 + `</mxfile>`)
	var reports []error
	v := newView(kindDrawio, editor.Host{Buffer: buffer, Report: func(err error) { reports = append(reports, err) }})
	tester := ui.NewTester(v.Draw, 960, 640)
	x, y := nodeCenterOnWindow(v, v.currentPage().Model.Nodes[0])
	tester.Press(x, y)
	tester.Move(x+30, y+20)
	if v.draggingID != "same" {
		t.Fatal("native drag did not start")
	}
	authority := `<mxfile>` + p2 + p1 + `</mxfile>`
	_ = buffer.ReplaceUTF16(0, buffer.LenUTF16(), authority)
	tester.Release(x+30, y+20)
	if buffer.String() != authority || v.draggingID != "" || len(reports) == 0 {
		t.Fatal("stale drag moved a node in a changed page or cancellation was silent")
	}
	v.connectSource = "same"
	if !v.changePage(1) || v.connectSource != "" {
		t.Fatal("page change retained stale connection source")
	}
}

func TestDrawioMissingDraftPageDoesNotShowOrModifyAnotherPage(t *testing.T) {
	p1, p2 := pageXML("p1", "first"), pageXML("p2", "second")
	buffer := text.New(`<mxfile>` + p1 + p2 + `</mxfile>`)
	v := newView(kindDrawio, editor.Host{Buffer: buffer})
	tester := ui.NewTester(v.Draw, 960, 640)
	focusDraft(t, tester)
	tester.Compose("留在原页", 4)
	authority := `<mxfile>` + p2 + `</mxfile>`
	_ = buffer.ReplaceUTF16(0, buffer.LenUTF16(), authority)
	tester.Frame()
	if v.currentPage() != nil || !errors.Is(v.draftError, ErrDraftConflict) || !v.pendingDraft() {
		t.Fatal("missing draft page displayed as another page")
	}
	if err := v.Flush(); !errors.Is(err, ErrDraftConflict) || buffer.String() != authority {
		t.Fatal("missing draft page wrote wrong target", err)
	}
}

func TestDrawioInvalidTextRetainsDraftAndCanBeCorrected(t *testing.T) {
	buffer := text.New(testDrawio)
	v := newView(kindDrawio, editor.Host{Buffer: buffer})
	tester := ui.NewTester(v.Draw, 960, 640)
	focusDraft(t, tester)
	tester.Type("bad\x00label")
	if v.draftError == nil || !v.pendingDraft() || buffer.String() != testDrawio {
		t.Fatal("invalid XML text was lost or written")
	}
	tester.Frame()
	focusDraft(t, tester)
	tester.Type("valid 中文")
	if v.pendingDraft() || v.draftError != nil || !strings.Contains(buffer.String(), `value="valid 中文"`) {
		t.Fatal("corrected draft did not retry")
	}
}

func compressedPage(t *testing.T) string {
	t.Helper()
	model := `<mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="a" value="compressed" vertex="1" parent="1"><mxGeometry x="1" y="2" width="100" height="40" as="geometry"/></mxCell></root></mxGraphModel>`
	var buffer bytes.Buffer
	writer, _ := flate.NewWriter(&buffer, flate.DefaultCompression)
	_, _ = writer.Write([]byte(url.PathEscape(model)))
	_ = writer.Close()
	return `<mxfile><diagram id="zip">` + base64.StdEncoding.EncodeToString(buffer.Bytes()) + `</diagram></mxfile>`
}

func TestReadOnlyPagesAndDocumentsPreserveBytes(t *testing.T) {
	for _, tc := range []struct {
		source   string
		kind     documentKind
		readOnly bool
	}{{testMindmap, kindMindmap, true}, {testDrawio, kindDrawio, true}, {compressedPage(t), kindDrawio, false}} {
		buffer := text.New(tc.source)
		v := newView(tc.kind, editor.Host{Buffer: buffer, ReadOnly: tc.readOnly})
		tester := ui.NewTester(v.Draw, 900, 640)
		focusDraft(t, tester)
		tester.Type("forbidden")
		v.addMindmapChild()
		v.addDiagramShape("rectangle")
		if err := v.Flush(); err != nil {
			t.Fatal("clean readonly Flush failed", err)
		}
		if _, err := v.Export("svg"); err != nil {
			t.Fatal(err)
		}
		v.Release()
		v.Warnings()
		tester.Frame()
		if buffer.String() != tc.source || buffer.CanUndo() {
			t.Fatal("readonly lifecycle changed original bytes")
		}
	}
}

func TestReleaseRebuildsSameVersionAndRetainsViewState(t *testing.T) {
	for _, tc := range []struct {
		source string
		kind   documentKind
	}{{testMindmap, kindMindmap}, {`<mxfile>` + pageXML("p1", "a") + pageXML("p2", "b") + `</mxfile>`, kindDrawio}} {
		buffer := text.New(tc.source)
		v := newView(tc.kind, editor.Host{Buffer: buffer})
		tester := ui.NewTester(v.Draw, 900, 640)
		if tc.kind == kindDrawio {
			v.changePage(1)
		} else {
			v.selectNode("left")
		}
		v.zoom, v.panX, v.panY, v.fitted = .72, 41, 53, true
		selected, page := v.selectedID, v.pageIndex
		version := buffer.Version()
		v.Release()
		if v.mind != nil || v.diagram != nil || len(v.layout.Nodes) != 0 || v.draft != nil {
			t.Fatal("inactive heavy cache retained")
		}
		v.Warnings()
		if _, err := v.Export("svg"); err != nil {
			t.Fatal(err)
		}
		tester.Frame()
		if v.selectedID != selected || v.pageIndex != page || v.zoom != .72 || v.panX != 41 || v.panY != 53 || buffer.Version() != version || buffer.String() != tc.source {
			t.Fatal("release changed view state or document")
		}
	}
}

func TestUnsupportedGraphDataIsDisclosedAndRetained(t *testing.T) {
	image := strings.Replace(testMindmap, `"topic": "中心"`, `"topic": "中心", "image": {"url":"https://invalid/image.png","width":80,"height":40,"future":true}`, 1)
	buffer := text.New(image)
	v := newView(kindMindmap, editor.Host{Buffer: buffer})
	tester := ui.NewTester(v.Draw, 900, 640)
	if !strings.Contains(strings.Join(v.Warnings(), " "), "图片已保留") {
		t.Fatal("image downgrade not disclosed")
	}
	focusDraft(t, tester)
	tester.Type("changed")
	parsed := parseMind(t, buffer)
	future, ok := parsed.Root.Image.Extra.Get("future")
	if !ok || string(future) != "true" || parsed.Root.Image.URL != "https://invalid/image.png" {
		t.Fatal("unknown image metadata removed")
	}
}

func TestGraphDraftScreenshot(t *testing.T) {
	dir := os.Getenv("LEVNOTE_GRAPH_SCREENSHOT_DIR")
	if dir == "" {
		t.Skip("set LEVNOTE_GRAPH_SCREENSHOT_DIR")
	}
	buffer := text.New(testMindmap)
	v := newView(kindMindmap, editor.Host{Buffer: buffer})
	tester := ui.NewTester(v.Draw, 960, 640)
	focusDraft(t, tester)
	tester.Compose("正在输入🙂", 5)
	replaceSource(t, buffer, `"topic": "中心"`, `"topic": "外部修改"`)
	_ = v.Flush()
	tester.Frame()
	_ = os.MkdirAll(dir, 0755)
	f, err := os.Create(filepath.Join(dir, "graph-draft-conflict.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, tester.Image()); err != nil {
		t.Fatal(err)
	}
}

func TestNarrowGraphToolbarRemainsVisible(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		kind         documentKind
	}{
		{"mindmap", testMindmap, kindMindmap},
		{"drawio", strings.Replace(testDrawio, `name="流程"`, `name="本地流程页面很长也不应该挤成逐字竖列"`, 1), kindDrawio},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := newView(tc.kind, editor.Host{Buffer: text.New(tc.source)})
			tester := ui.NewTester(v.Draw, 532, 380)
			for _, label := range []string{"放大画布", "缩小画布", "适应画布"} {
				rect, ok := tester.Find(label)
				if !ok || rect.X < 0 || rect.Y < 0 || rect.X+rect.W > 532 || rect.Y+rect.H > 380 {
					t.Fatalf("control clipped at narrow width: %s %+v", label, rect)
				}
			}
			if tc.kind == kindDrawio {
				rect, ok := tester.Find("图纸页面名称")
				if !ok || rect.H > 24 {
					t.Fatalf("page title wrapped vertically: %+v", rect)
				}
			}
			before := v.zoom
			if err := tester.Click("放大画布"); err != nil || v.zoom <= before {
				t.Fatal("zoom is not actionable", err)
			}
			if dir := os.Getenv("LEVNOTE_GRAPH_SCREENSHOT_DIR"); dir != "" {
				_ = os.MkdirAll(dir, 0755)
				f, err := os.Create(filepath.Join(dir, tc.name+"-narrow.png"))
				if err != nil {
					t.Fatal(err)
				}
				defer f.Close()
				if err := png.Encode(f, tester.Image()); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
