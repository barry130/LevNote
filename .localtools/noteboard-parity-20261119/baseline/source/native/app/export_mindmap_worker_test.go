package app

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/engineipc"
	"levnote.local/native/core/engineruntime"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

const mindmapWorkerBody = `{"format":"levnote.mindmap","version":1,"root":{"id":"root","topic":"主题<&🙂","icons":["⭐"],"tags":["同名"],"unknown":1.00}}` + "\r\n"
const mindmapWorkerSVG = `<svg xmlns="http://www.w3.org/2000/svg"><text>⭐ 主题&lt;&amp;🙂 [同名]</text></svg>` + "\n"

func mindmapWorkerOutcome() exportOutcome {
	return exportOutcome{result: engineruntime.Result{Payload: []byte(mindmapWorkerSVG), Warnings: []engineipc.Warning{},
		Engine: engineruntime.EngineInfo{ID: engineipc.MindmapSVG, Version: "svg-test-v1", SHA256: strings.Repeat("b", 64)}}}
}

func mindmapWorkerApp(t *testing.T) (*App, *Tab, *controlledExporter, chan func(), string) {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "map.mindmap")
	if err := os.WriteFile(path, append([]byte{0xef, 0xbb, 0xbf}, []byte(mindmapWorkerBody)...), 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	if tab == nil || tab.Adapter == nil || tab.Adapter.ExportEngine == nil || *tab.Adapter.ExportEngine != (editor.ExportEngine{ID: "levnote.mindmap-svg", Format: "svg"}) {
		t.Fatal("registered Mindmap engine descriptor missing")
	}
	f, queue := newControlledExporter(), make(chan func(), 16)
	a.exporter, a.uiDispatch = f, func(fn func()) { queue <- fn }
	return a, tab, f, queue, root
}

func TestMindmapWorkerModesReadonlyAndExactPublication(t *testing.T) {
	for _, mode := range []string{"source", "preview", "split"} {
		for _, readonly := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/readonly-%v", mode, readonly), func(t *testing.T) {
				a, tab, f, queue, root := mindmapWorkerApp(t)
				tab.Doc.ReadOnly, tab.Editor.ReadOnly = readonly, readonly
				if !a.setMode(tab, mode) {
					t.Fatal(a.Message)
				}
				if mode != "source" {
					tester := ui.NewTester(a.View, 1180, 780)
					if readonly {
						call, job := awaitExportCall(t, f), a.previewJob
						if call.id != engineipc.MindmapPreview {
							t.Fatal("readonly scene not requested")
						}
						call.response <- previewSuccess()
						awaitPreviewDone(t, job)
					}
					tester.Frame()
				}
				if mode == "source" && tab.View != nil {
					t.Fatal("source export mounted a view")
				}
				_ = tab.Doc.Buffer.SetSelection(8, 2)
				version, history, selection, baseline, dirty := tab.Doc.Buffer.Version(), tab.Doc.Buffer.History(), tab.Doc.Buffer.Selection(), tab.Doc.DiskToken(), tab.Doc.Dirty()
				path := filepath.Join(root, "out.svg")
				dialogs := 0
				a.exportDialog = func(options mygo.SaveDialogOptions) (string, error) {
					dialogs++
					if options.Title != "导出 SVG" || options.DefaultPath != "map.svg" {
						t.Error("wrong export format/name", options)
					}
					return path, nil
				}
				for attempt := 0; attempt < 2; attempt++ {
					a.exportCurrent()
					job, call := a.exportJob, awaitExportCall(t, f)
					var snapshot bytes.Buffer
					n, err := call.source.Data.WriteTo(&snapshot)
					if err != nil || n != call.source.Size || snapshot.String() != mindmapWorkerBody || call.id != engineipc.MindmapSVG {
						t.Fatal("Mindmap snapshot/identity changed", err)
					}
					if dialogs != attempt {
						t.Fatal("dialog opened before joined result")
					}
					if call.source.Page != nil {
						t.Fatalf("legacy Mindmap request carried page %+v", call.source.Page)
					}
					call.response <- mindmapWorkerOutcome()
					awaitExportDone(t, job)
					drainOneExportCallback(t, queue)
					if got := mustReadFile(t, path); string(got) != mindmapWorkerSVG {
						t.Fatalf("independent SVG payload publication %q", got)
					}
					if attempt == 1 && (!strings.Contains(a.Message, "导出失败") || !strings.Contains(a.exportStatusText(*tab.Adapter.ExportEngine), "文件写入失败")) {
						t.Fatal("existing SVG target was not refused", a.Message)
					}
					job.mu.Lock()
					retained := len(job.result.Payload)
					job.mu.Unlock()
					if retained != 0 || a.exportJob != nil {
						t.Fatal("completed SVG payload/job retained")
					}
				}
				if dialogs != 2 || tab.Doc.Buffer.String() != mindmapWorkerBody || tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.History() != history || tab.Doc.Buffer.Selection() != selection || tab.Doc.Dirty() != dirty || !tab.Doc.SameBaseline(baseline) || tab.Doc.ReadOnly != readonly || tab.Doc.Encoding != document.UTF8 || !tab.Doc.BOM || !bytes.Equal(mustReadFile(t, tab.Doc.Path), append([]byte{0xef, 0xbb, 0xbf}, []byte(mindmapWorkerBody)...)) {
					t.Fatal("export changed exact disk/source/BOM/history/baseline")
				}
				if mode == "source" && tab.View != nil {
					t.Fatal("source completion mounted view")
				}
			})
		}
	}
}

func TestMindmapWorkerFieldFlushAndPendingTagRecovery(t *testing.T) {
	a, tab, f, queue, _ := mindmapWorkerApp(t)
	tester := ui.NewTester(a.View, 1180, 780)
	a.exportDialog = func(mygo.SaveDialogOptions) (string, error) { return "", nil }
	field := revealTagAppControl(t, tester, 1180, 780, "节点文本", ui.RoleTextField, 48)
	tester.ClickAt(field.Bounds.X+field.Bounds.W/2, field.Bounds.Y+field.Bounds.H/2)
	tester.Command("selectAll")
	tester.Compose("写回🙂", text.UTF16Len("写回🙂"))
	if tab.Doc.Buffer.String() != mindmapWorkerBody {
		t.Fatal("composition fixture already wrote source")
	}
	a.exportCurrent()
	job, call := a.exportJob, awaitExportCall(t, f)
	var snapshot bytes.Buffer
	_, err := call.source.Data.WriteTo(&snapshot)
	if err != nil || parseMindmapBytes(t, snapshot.Bytes()).Root.Topic != "写回🙂" || snapshot.String() != tab.Doc.Buffer.String() || tab.Doc.Buffer.History().UndoDepth != 1 {
		t.Fatal("real field Flush missing from snapshot", err)
	}
	call.response <- mindmapWorkerOutcome()
	awaitExportDone(t, job)
	drainOneExportCallback(t, queue)
	if !tab.Editor.Command("undo") || tab.Doc.Buffer.String() != mindmapWorkerBody {
		t.Fatal("field commit not independently undoable")
	}
	tester.Frame()
	openTagsAppPanel(t, tester, 1180, 780)
	typeTagAppInput(t, tester, 1180, 780, "未添加🙂", false)
	typeTagOrdinaryField(t, tester, 1180, 780, "未写字段")
	before, history, selection := tab.Doc.Buffer.String(), tab.Doc.Buffer.History(), tab.Doc.Buffer.Selection()
	dialogs := 0
	a.exportDialog = func(mygo.SaveDialogOptions) (string, error) { dialogs++; return "", nil }
	a.exportCurrent()
	if a.exportJob != nil || len(f.calls) != 0 || dialogs != 0 || !strings.Contains(a.Message, "标签") || tab.Doc.Buffer.String() != before || tab.Doc.Buffer.History() != history || tab.Doc.Buffer.Selection() != selection {
		t.Fatal("pending tag escaped preflight", a.Message)
	}
	clickTagAppButton(t, tester, 1180, 780, "恢复标签及节点草稿")
	recovered := a.Current()
	want := "map.mindmap\n节点 ID: root\n页面 ID: \n字段: 节点文本\n原字段存在: true\n\n未写字段\n\n--- 未添加的标签输入 ---\n节点 ID: root\n字段: 新标签（尚未添加）\n\n未添加🙂"
	if recovered == tab || !recovered.Doc.IsNew || recovered.Doc.Buffer.String() != want || tab.Doc.Buffer.String() != before || tab.Doc.Buffer.History() != history || a.exportJob != nil {
		t.Fatal("real Host.RecoverDraft lost drafts/retargeted export", recovered.Doc.Buffer.String())
	}
}

func TestMindmapWorkerPreflightPairSourceCapAndFlushOwners(t *testing.T) {
	for _, change := range []string{"unknown", "wrong-format", "oversize", "flush-buffer", "flush-adapter", "flush-capability", "flush-recover"} {
		t.Run(change, func(t *testing.T) {
			a, tab, f, _, _ := mindmapWorkerApp(t)
			expected := "导出前文档或格式已变化"
			adapter := *tab.Adapter
			capability := *adapter.ExportEngine
			adapter.ExportEngine = &capability
			tab.Adapter = &adapter
			switch change {
			case "unknown":
				capability.ID = "levnote.foreign"
				expected = "声明不匹配"
			case "wrong-format":
				capability.Format = "html"
				expected = "声明不匹配"
			case "oversize":
				if !a.setMode(tab, "source") {
					t.Fatal(a.Message)
				}
				tab.Doc.Buffer = text.New(strings.Repeat("x", (16<<20)+1))
				tab.Editor.Buffer = tab.Doc.Buffer
				expected = "思维导图源码超过 16 MiB 导出上限"
			default:
				tab.View = &callbackExportFlushView{callback: func() {
					switch change {
					case "flush-buffer":
						tab.Doc.Buffer = text.New("replacement")
						tab.Editor.Buffer = tab.Doc.Buffer
					case "flush-adapter":
						replacement := adapter
						tab.Adapter = &replacement
					case "flush-capability":
						capability.Format = "html"
					case "flush-recover":
						if err := a.recoverDraft("copy.txt", "retained"); err != nil {
							t.Fatal(err)
						}
					}
				}}
			}
			a.exportCurrent()
			if a.exportJob != nil || len(f.calls) != 0 || !strings.Contains(a.Message, expected) {
				t.Fatal("wrong preflight admission/message", change, a.Message)
			}
			if change == "wrong-format" || change == "unknown" {
				if status := a.exportStatusText(capability); !strings.Contains(status, "不可用") || strings.Contains(status, "随包外部 HTML 导出") {
					t.Fatal("wrong pair advertised healthy engine", status)
				}
			}
		})
	}
	// A source above Markdown's cap but under Mindmap's cap reaches only the
	// controlled app seam; this is not a parser or resource measurement.
	a, tab, f, queue, _ := mindmapWorkerApp(t)
	if !a.setMode(tab, "source") {
		t.Fatal(a.Message)
	}
	tab.Doc.Buffer = text.New(strings.Repeat(" ", (8<<20)+1))
	tab.Editor.Buffer = tab.Doc.Buffer
	a.exportDialog = func(mygo.SaveDialogOptions) (string, error) { return "", nil }
	a.exportCurrent()
	job, call := a.exportJob, awaitExportCall(t, f)
	if call.source.Size != (8<<20)+1 || call.id != engineipc.MindmapSVG || tab.View != nil {
		t.Fatal("Mindmap inherited Markdown cap/mounted view")
	}
	call.response <- mindmapWorkerOutcome()
	awaitExportDone(t, job)
	drainOneExportCallback(t, queue)
}

type callbackExportFlushView struct {
	failingView
	callback func()
}

func (v *callbackExportFlushView) Flush() error {
	if v.callback != nil {
		v.callback()
	}
	return nil
}

func TestMindmapWorkerStaleSnapshotNoFallbackAndStatus(t *testing.T) {
	for _, change := range []string{"source", "adapter", "capability", "missing", "corrupt"} {
		t.Run(change, func(t *testing.T) {
			a, tab, f, queue, _ := mindmapWorkerApp(t)
			forbidden := &exportForbiddenView{}
			tab.View = forbidden
			adapter := *tab.Adapter
			capability := *adapter.ExportEngine
			adapter.ExportEngine = &capability
			tab.Adapter = &adapter
			if status := a.exportStatusText(capability); !strings.Contains(status, "本地引擎 SVG 导出 · 未检查") || len(f.calls) != 0 {
				t.Fatal("status inspected or called SVG HTML", status)
			}
			dialogs := 0
			a.exportDialog = func(mygo.SaveDialogOptions) (string, error) { dialogs++; return "", nil }
			a.exportCurrent()
			job, call := a.exportJob, awaitExportCall(t, f)
			switch change {
			case "source":
				_ = tab.Doc.Buffer.ReplaceUTF16(0, 0, " ")
			case "adapter":
				replacement := adapter
				tab.Adapter = &replacement
			case "capability":
				capability.Format = "html"
			}
			var snapshot bytes.Buffer
			if _, err := call.source.Data.WriteTo(&snapshot); err != nil || snapshot.String() != mindmapWorkerBody {
				t.Fatal("snapshot changed after live edit", err)
			}
			outcome := mindmapWorkerOutcome()
			if change == "missing" || change == "corrupt" {
				kind := engineruntime.FailureUnavailable
				if change == "corrupt" {
					kind = engineruntime.FailureIntegrity
				}
				a.exportStatuses = map[string]exportEngineStatus{capability.ID: {checked: true, lastVerified: outcome.result.Engine}}
				outcome = exportOutcome{err: &engineruntime.Failure{Kind: kind, Detail: "明确拒绝"}}
			}
			call.response <- outcome
			awaitExportDone(t, job)
			drainOneExportCallback(t, queue)
			if dialogs != 0 || forbidden.exports != 0 || a.exportJob != nil {
				t.Fatal("invalid/failed SVG fell back or published")
			}
			if change == "missing" || change == "corrupt" {
				status := a.exportStatusText(capability)
				if !strings.Contains(status, "本地引擎 SVG 导出") || !strings.Contains(status, "历史已验证") || !strings.Contains(status, "明确拒绝") {
					t.Fatal("SVG historical failure missing", status)
				}
				if markdown := a.exportStatusText(editor.ExportEngine{ID: "levnote.markdown-export", Format: "html"}); !strings.Contains(markdown, "未检查") {
					t.Fatal("Mindmap status overwrote Markdown", markdown)
				}
			}
		})
	}
}

func TestMindmapWorkerAndMarkdownShareAppAdmission(t *testing.T) {
	a, markdown, f, queue, _ := workerApp(t)
	a.NewDocument("mindmap")
	mindmap := a.Current()
	if !a.Select(0) {
		t.Fatal(a.Message)
	}
	f.ignoreCancellation = true // Model a result that is still cleaning up.
	a.exportDialog = func(mygo.SaveDialogOptions) (string, error) { return "", nil }
	a.exportCurrent()
	first, call := a.exportJob, awaitExportCall(t, f)
	if call.id != engineipc.MarkdownHTML || a.Current() != markdown {
		t.Fatal("first identity")
	}
	if !a.Select(1) || a.Current() != mindmap {
		t.Fatal(a.Message)
	}
	a.exportCurrent()
	if a.exportJob != first || len(f.calls) != 0 || !strings.Contains(a.Message, "已有导出") {
		t.Fatal("second engine replaced active cleanup")
	}
	call.response <- exportSuccess()
	awaitExportDone(t, first)
	drainOneExportCallback(t, queue)
	a.exportCurrent()
	second, call := a.exportJob, awaitExportCall(t, f)
	if second == first || call.id != engineipc.MindmapSVG {
		t.Fatal("joined runner could not switch engines")
	}
	call.response <- mindmapWorkerOutcome()
	awaitExportDone(t, second)
	drainOneExportCallback(t, queue)
	if a.exportJob != nil {
		t.Fatal("second job retained")
	}
}

func TestMindmapWorkerDisableRetainsSourceFallback(t *testing.T) {
	a, tab, f, queue, _ := mindmapWorkerApp(t)
	if !a.setMode(tab, "source") {
		t.Fatal(a.Message)
	}
	a.exportDialog = func(mygo.SaveDialogOptions) (string, error) { t.Fatal("disabled worker opened dialog"); return "", nil }
	a.exportCurrent()
	job, call := a.exportJob, awaitExportCall(t, f)
	before, version, history := tab.Doc.Buffer.String(), tab.Doc.Buffer.Version(), tab.Doc.Buffer.History()
	a.setPluginEnabled("mindmap", false)
	if tab.Adapter != nil || tab.View != nil || tab.Mode != "source" || tab.MatchedPlugin.ID != "code" {
		t.Fatal("disabled Mindmap retained export/preview capability")
	}
	call.response <- mindmapWorkerOutcome()
	awaitExportDone(t, job)
	drainOneExportCallback(t, queue)
	a.exportCurrent()
	if a.exportJob != nil || len(f.calls) != 0 || tab.Doc.Buffer.String() != before || tab.Doc.Buffer.Version() != version || tab.Doc.Buffer.History() != history {
		t.Fatal("disabled source fallback exported or changed source")
	}
	a.setPluginEnabled("mindmap", true)
	if tab.Adapter == nil || tab.Adapter.ExportEngine == nil || tab.Adapter.ExportEngine.ID != "levnote.mindmap-svg" {
		t.Fatal("re-enable did not restore fixed capability")
	}
}
