package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"levnote.local/native/core/document"
	"levnote.local/native/core/engineipc"
	"levnote.local/native/core/engineruntime"
	"levnote.local/native/formats/mindmap"
)

// The later delivery gate places this tester in a disposable complete bundle.
// Only SaveDialog is injected; the registered source-mode adapter and NewBundled
// use real transport/worker/publication. Default runs never launch that worker.
func TestMindmapWorkerExportBundledHostPathAcceptance(t *testing.T) {
	if os.Getenv("LEVNOTE_WORKER_BUNDLE_ACCEPTANCE") != "1" {
		t.Skip("explicit packaged bundle integration gate")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	bundleDir := filepath.Dir(exe)
	for _, relative := range []string{"levnote-native.exe", "engines/mindmap-svg/levnote-mindmap-svg.exe", "engines/mindmap-svg/manifest.json"} {
		if info, err := os.Stat(filepath.Join(bundleDir, relative)); err != nil || !info.Mode().IsRegular() {
			t.Fatalf("tester needs the complete fixed bundle: %s %v", relative, err)
		}
	}
	var manifest struct {
		Engine, EngineVersion, Format, SHA256 string
	}
	manifestBytes := mustReadFile(t, filepath.Join(bundleDir, "engines/mindmap-svg/manifest.json"))
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil || manifest.Engine != "levnote.mindmap-svg" || manifest.Format != "svg" || manifest.EngineVersion == "" {
		t.Fatal("wrong Mindmap manifest", err)
	}
	root := t.TempDir()
	// Exact spelling, BOM and CRLF are intentional. The external image is only
	// a placeholder; unknown values retain their original numeric spellings.
	source := "{\r\n" +
		`"format":"levnote.mindmap","version":1,"unknown":1.00,` + "\r\n" +
		`"root":{"id":"r","topic":"根<&🙂","icons":["⭐","custom<&"],"tags":["甲&",{"text":"乙","extra":{"n":2.50}}],"style":{"background-color":"red","color":"rebeccapurple","border":"#123"},"image":{"url":"https://invalid.example/image.png","width":90,"height":40,"future":3.00},"future":4.00}` + "\r\n}\r\n"
	originalBytes := append([]byte{0xef, 0xbb, 0xbf}, []byte(source)...)
	path := filepath.Join(root, "bundled.mindmap")
	if err := os.WriteFile(path, originalBytes, 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	if tab == nil || tab.Adapter == nil || tab.Adapter.ExportEngine == nil ||
		tab.Adapter.ExportEngine.ID != "levnote.mindmap-svg" || tab.Adapter.ExportEngine.Format != "svg" ||
		!a.setMode(tab, "source") || tab.View != nil {
		t.Fatal("registered source-mode Mindmap capability missing")
	}
	if err := tab.Doc.Buffer.SetSelection(8, 2); err != nil {
		t.Fatal(err)
	}
	version, selection, history := tab.Doc.Buffer.Version(), tab.Doc.Buffer.Selection(), tab.Doc.Buffer.History()
	dirty, baseline := tab.Doc.Dirty(), tab.Doc.DiskToken()
	runner, err := engineruntime.NewBundled()
	if err != nil {
		t.Fatal(err)
	}
	a.exporter = runner
	queue := make(chan func(), 4)
	a.uiDispatch = func(fn func()) { queue <- fn }
	destination := filepath.Join(root, "bundled.svg")
	dialogs := 0
	a.exportDialog = func(options mygo.SaveDialogOptions) (string, error) {
		dialogs++
		if options.Title != "导出 SVG" || options.DefaultPath != "bundled.svg" {
			t.Error("wrong SVG dialog metadata", options)
		}
		return destination, nil
	}
	wantWarnings := []engineipc.Warning{
		{Code: "UNKNOWN_KEY", Message: `node "r" has 1 preserved unknown field(s)`, End: uint32(len(source))},
		{Code: "UNKNOWN_KEY", Message: "preserved 1 unknown top-level field(s)", End: uint32(len(source))},
		{Code: "IMAGE_PLACEHOLDER", Message: "1 个节点图片已保留；原生画布不访问文件或网络，显示为“图”占位", End: uint32(len(source))},
	}
	parsed, parseIssues, err := mindmap.Parse([]byte(source), mindmap.ParseOptions{})
	if err != nil || len(parseIssues) != 2 {
		t.Fatal("independent fixture parse warnings", err, parseIssues)
	}
	parity, err := mindmap.ExportSVG(parsed, mindmap.Layout(parsed, mindmap.NativeLayoutOptions()), 0)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	var published []byte
	var engine engineruntime.EngineInfo
	for attempt := 0; attempt < 2; attempt++ {
		a.exportCurrent()
		job := a.exportJob
		if job == nil {
			t.Fatal("real export did not start", a.Message)
		}
		select {
		case <-job.done:
		case <-time.After(40 * time.Second):
			t.Fatal("production Mindmap worker did not join within runner bound")
		}
		job.mu.Lock()
		runErr, info := job.err, job.result.Engine
		payload := append([]byte(nil), job.result.Payload...)
		warnings := append([]engineipc.Warning(nil), job.result.Warnings...)
		job.mu.Unlock()
		if runErr != nil || info.ID != engineipc.MindmapSVG || info.Version != manifest.EngineVersion ||
			!strings.EqualFold(info.SHA256, manifest.SHA256) || len(info.SHA256) != 64 {
			t.Fatal("real Mindmap identity/outcome", info, runErr)
		}
		if attempt > 0 && info != engine {
			t.Fatal("fixed engine changed between requests")
		}
		engine = info
		if !bytes.Equal(payload, parity) || !reflect.DeepEqual(warnings, wantWarnings) {
			t.Fatalf("real SVG/parity or exact whole-source warning ranges differ: %v", warnings)
		}
		assertBundledMindmapSVG(t, payload)
		if dialogs != attempt {
			t.Fatal("dialog ran before joined delivery")
		}
		drainOneExportCallback(t, queue)
		got := mustReadFile(t, destination)
		if attempt == 0 {
			published = got
			if !bytes.Equal(got, payload) || !strings.Contains(a.Message, "3 项提示") {
				t.Fatal("SVG publication/warning disclosure changed")
			}
		} else if !bytes.Equal(got, published) || !strings.Contains(a.Message, "导出失败") ||
			!strings.Contains(a.exportStatusText(*tab.Adapter.ExportEngine), "文件写入失败") {
			t.Fatal("exclusive SVG target was overwritten or refusal hidden")
		}
		job.mu.Lock()
		retained := len(job.result.Payload)
		job.mu.Unlock()
		if retained != 0 || a.exportJob != nil || dialogs != attempt+1 || tab.View != nil ||
			tab.Doc.Buffer.String() != source || tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.Selection() != selection ||
			tab.Doc.Buffer.History() != history || tab.Doc.Dirty() != dirty || !tab.Doc.SameBaseline(baseline) ||
			tab.Doc.Encoding != document.UTF8 || !tab.Doc.BOM || !bytes.Equal(mustReadFile(t, path), originalBytes) {
			t.Fatal("real export mutated disk/source/BOM/state or retained payload/view")
		}
	}
	if err := a.closeEngineExport(); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(published)
	sourceDigest := sha256.Sum256(originalBytes)
	record := struct {
		Scope, Tester, EngineVersion, EngineSHA256, SVGSHA256, DiskSourceSHA256 string
		SVGBytes, Warnings, Dialogs, WorkerRequests                             int
		ElapsedSeconds                                                          float64
		RuntimeJoined                                                           bool
	}{"registered source-mode app host-path tester; real bundled SVG worker, injected SaveDialog only; joined-return contract, no PID sampling",
		exe, engine.Version, engine.SHA256, hex.EncodeToString(digest[:]), hex.EncodeToString(sourceDigest[:]),
		len(published), len(wantWarnings), dialogs, 2, time.Since(start).Seconds(), true}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(data))
}

func assertBundledMindmapSVG(t *testing.T, payload []byte) {
	t.Helper()
	var svg struct {
		XMLName xml.Name
		ViewBox string `xml:"viewBox,attr"`
		Width   string `xml:"width,attr"`
		Height  string `xml:"height,attr"`
		Rects   []struct {
			X      string `xml:"x,attr"`
			Y      string `xml:"y,attr"`
			Width  string `xml:"width,attr"`
			Height string `xml:"height,attr"`
			Fill   string `xml:"fill,attr"`
			Stroke string `xml:"stroke,attr"`
		} `xml:"rect"`
		Texts []struct {
			X      string   `xml:"x,attr"`
			Y      string   `xml:"y,attr"`
			Fill   string   `xml:"fill,attr"`
			Text   string   `xml:",chardata"`
			TSpans []string `xml:"tspan"`
		} `xml:"text"`
	}
	if err := xml.Unmarshal(payload, &svg); err != nil || svg.XMLName != (xml.Name{Space: "http://www.w3.org/2000/svg", Local: "svg"}) ||
		svg.ViewBox != "-24 -24 196 144" || svg.Width != "196" || svg.Height != "144" || len(svg.Rects) != 2 || len(svg.Texts) != 2 {
		t.Fatal("independent SVG XML/outer geometry", svg, err)
	}
	node, label, image := svg.Rects[1], svg.Texts[0], svg.Texts[1]
	if node.X != "28" || node.Y != "28" || node.Width != "92" || node.Height != "40" || node.Fill != "#ff0000" || node.Stroke != "#123" ||
		label.X != "74" || label.Fill != "#663399" || len(label.TSpans) != 1 || label.TSpans[0] != "⭐ custom<& 根<&🙂 [甲& · 乙]" ||
		image.X != "112" || image.Y != "42" || image.Text != "图" {
		t.Fatal("independent node/label/placeholder geometry", node, label, image)
	}
	if !bytes.Contains(payload, []byte("custom&lt;&amp; 根&lt;&amp;🙂 [甲&amp; · 乙]")) ||
		bytes.Contains(payload, []byte("<image")) || bytes.Contains(payload, []byte("https://")) || bytes.Contains(payload, []byte("<script")) {
		t.Fatal("literal escaping or image placeholder boundary changed")
	}
}
