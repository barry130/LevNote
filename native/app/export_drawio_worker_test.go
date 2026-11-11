package app

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/enginecatalog"
	"levnote.local/native/core/engineipc"
	"levnote.local/native/core/engineruntime"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

const drawioWorkerBody = `<mxfile><diagram id="p1" name="第一页面"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="same" value="PAGE_ONE" vertex="1" parent="1"><mxGeometry x="20" y="20" width="120" height="60" as="geometry"/></mxCell></root></mxGraphModel></diagram><diagram id="p2" name="第二页面"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="same" value="PAGE_TWO" vertex="1" parent="1"><mxGeometry x="40" y="30" width="140" height="70" as="geometry"/></mxCell></root></mxGraphModel></diagram></mxfile>`
const drawioWorkerSVG = `<svg xmlns="http://www.w3.org/2000/svg"><text>第二页面</text></svg>` + "\n"

func drawioWorkerOutcome() exportOutcome {
	return exportOutcome{result: engineruntime.Result{
		Payload:  []byte(drawioWorkerSVG),
		Warnings: []engineipc.Warning{},
		Engine: engineruntime.EngineInfo{
			ID: engineipc.DrawioSVG, Version: "drawio-test-v1", SHA256: strings.Repeat("c", 64),
		},
	}}
}

func drawioWorkerApp(t *testing.T) (*App, *Tab, *controlledExporter, chan func(), string, string) {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, "diagram.drawio")
	disk := append([]byte{0xef, 0xbb, 0xbf}, []byte(drawioWorkerBody)...)
	if err := os.WriteFile(path, disk, 0600); err != nil {
		t.Fatal(err)
	}
	a := newTestApp(t, filepath.Join(root, "config"))
	a.Open(path)
	tab := a.Current()
	if tab == nil || tab.Adapter == nil || tab.Adapter.ExportEngine == nil ||
		*tab.Adapter.ExportEngine != (editor.ExportEngine{
			ID: "levnote.drawio-svg", Format: "svg", RequiresView: true,
		}) {
		t.Fatal("registered draw.io engine descriptor missing")
	}
	f, queue := newControlledExporter(), make(chan func(), 16)
	a.exporter, a.uiDispatch = f, func(fn func()) { queue <- fn }
	return a, tab, f, queue, root, path
}

func clickDrawioPageButton(t *testing.T, tester *ui.Tester, name string) {
	t.Helper()
	button, ok := tester.AccessByRole(ui.RoleButton, name)
	if !ok || button.Disabled || button.Offscreen {
		t.Fatalf("draw.io page button %q unavailable: ok=%v access=%+v", name, ok, button)
	}
	if err := tester.Click(name); err != nil {
		t.Fatalf("draw.io page button %q was not clickable: %v", name, err)
	}
	tester.Frame()
}

func TestDrawioWorkerSecondPageExactPublicationPreservesState(t *testing.T) {
	a, tab, f, queue, root, path := drawioWorkerApp(t)
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	clickDrawioPageButton(t, tester, "下一页")
	if err := tab.Doc.Buffer.SetSelection(5, 3); err != nil {
		t.Fatal(err)
	}
	before, version := tab.Doc.Buffer.String(), tab.Doc.Buffer.Version()
	selection, history := tab.Doc.Buffer.Selection(), tab.Doc.Buffer.History()
	baseline, dirty := tab.Doc.DiskToken(), tab.Doc.Dirty()
	view := tab.View
	out := filepath.Join(root, "second-page.svg")
	dialogs := 0
	a.exportDialog = func(options mygo.SaveDialogOptions) (string, error) {
		dialogs++
		if options.Title != "导出 SVG" || options.DefaultPath != "diagram.svg" {
			t.Errorf("unexpected SaveDialog options: %+v", options)
		}
		return out, nil
	}

	a.exportCurrent()
	job, call := a.exportJob, awaitExportCall(t, f)
	if job == nil || call.id != engineipc.DrawioSVG || call.source.Page == nil ||
		*call.source.Page != (engineipc.RequestPage{Index: 1, ID: "p2"}) {
		t.Fatalf("draw.io request = job:%v id:%s page:%+v", job != nil, call.id, call.source.Page)
	}
	var snapshot bytes.Buffer
	n, err := call.source.Data.WriteTo(&snapshot)
	if err != nil || n != int64(len(drawioWorkerBody)) || call.source.Size != int64(len(drawioWorkerBody)) ||
		snapshot.String() != drawioWorkerBody || dialogs != 0 {
		t.Fatalf("draw.io source snapshot mismatch: n=%d size=%d err=%v dialogs=%d", n, call.source.Size, err, dialogs)
	}

	call.response <- drawioWorkerOutcome()
	awaitExportDone(t, job)
	drainOneExportCallback(t, queue)
	if dialogs != 1 || a.exportJob != nil || !bytes.Equal(mustReadFile(t, out), []byte(drawioWorkerSVG)) {
		t.Fatalf("draw.io publication mismatch: dialogs=%d job=%v", dialogs, a.exportJob != nil)
	}
	if tab.Doc.Buffer.String() != before || tab.Doc.Buffer.Version() != version ||
		tab.Doc.Buffer.Selection() != selection || tab.Doc.Buffer.History() != history ||
		tab.Doc.Dirty() != dirty || !tab.Doc.SameBaseline(baseline) {
		t.Fatal("successful draw.io export changed buffer/history/selection/dirty state")
	}
	if tab.View != view || tab.Mode != "preview" {
		t.Fatal("successful draw.io export replaced its prepared view or mode")
	}
	if tab.Doc.Encoding != document.UTF8 || !tab.Doc.BOM || !bytes.Equal(mustReadFile(t, path), append([]byte{0xef, 0xbb, 0xbf}, []byte(drawioWorkerBody)...)) {
		t.Fatal("successful draw.io export changed source disk bytes, encoding or BOM")
	}
}

func TestDrawioWorkerFlushesRealSecondPageDraftBeforePrepare(t *testing.T) {
	a, tab, f, queue, root, path := drawioWorkerApp(t)
	tester := ui.NewTester(a.View, 1180, 780)
	tester.Frame()
	clickDrawioPageButton(t, tester, "下一页")

	baseline, versionBefore := tab.Doc.Buffer.String(), tab.Doc.Buffer.Version()
	historyBefore := tab.Doc.Buffer.History()
	diskBefore := append([]byte(nil), mustReadFile(t, path)...)
	const pending = "第二页待导出🙂"
	if err := tester.Click("节点文本"); err != nil {
		t.Fatal(err)
	}
	tester.Command("selectAll")
	tester.Compose(pending, len([]rune(pending)))
	if tab.Doc.Buffer.String() != baseline || tab.Doc.Buffer.Version() != versionBefore ||
		tab.Doc.Buffer.History() != historyBefore {
		t.Fatal("pending graph composition reached the authoritative buffer before Flush")
	}

	out := filepath.Join(root, "flushed-second-page.svg")
	dialogs := 0
	dialogSource := ""
	var dialogHistory text.HistoryStatus
	a.exportDialog = func(mygo.SaveDialogOptions) (string, error) {
		dialogs++
		dialogSource = tab.Doc.Buffer.String()
		dialogHistory = tab.Doc.Buffer.History()
		if got := mustReadFile(t, path); !bytes.Equal(got, diskBefore) {
			t.Fatal("draw.io export changed the original disk file before Save")
		}
		return out, nil
	}

	a.exportCurrent()
	job, call := a.exportJob, awaitExportCall(t, f)
	sourceAfterFlush := tab.Doc.Buffer.String()
	if job == nil || call.id != engineipc.DrawioSVG || call.source.Page == nil ||
		*call.source.Page != (engineipc.RequestPage{Index: 1, ID: "p2"}) {
		t.Fatalf("flushed draw.io request = job:%v id:%s page:%+v", job != nil, call.id, call.source.Page)
	}
	if sourceAfterFlush == baseline || !strings.Contains(sourceAfterFlush, pending) ||
		tab.Doc.Buffer.Version() != versionBefore+1 ||
		tab.Doc.Buffer.History().UndoDepth != historyBefore.UndoDepth+1 ||
		tab.Doc.Buffer.History().RedoDepth != 0 {
		t.Fatalf("Flush was not one intended field transaction: version=%d->%d history=%+v source=%q",
			versionBefore, tab.Doc.Buffer.Version(), tab.Doc.Buffer.History(), sourceAfterFlush)
	}
	var snapshot bytes.Buffer
	n, err := call.source.Data.WriteTo(&snapshot)
	if err != nil || n != int64(len(sourceAfterFlush)) || call.source.Size != int64(len(sourceAfterFlush)) ||
		snapshot.String() != sourceAfterFlush || dialogs != 0 {
		t.Fatalf("post-Flush request snapshot mismatch: n=%d size=%d err=%v dialogs=%d",
			n, call.source.Size, err, dialogs)
	}

	call.response <- drawioWorkerOutcome()
	awaitExportDone(t, job)
	drainOneExportCallback(t, queue)
	if dialogs != 1 || dialogSource != sourceAfterFlush ||
		dialogHistory.UndoDepth != historyBefore.UndoDepth+1 || dialogHistory.RedoDepth != 0 {
		t.Fatalf("Save did not observe the exact one-transaction Flush result: dialogs=%d history=%+v",
			dialogs, dialogHistory)
	}
	if !bytes.Equal(mustReadFile(t, out), []byte(drawioWorkerSVG)) ||
		!bytes.Equal(mustReadFile(t, path), diskBefore) {
		t.Fatal("synthetic runner publication or original disk preservation mismatch")
	}
	if !tab.Doc.Buffer.Undo() || tab.Doc.Buffer.String() != baseline {
		t.Fatal("the single intended field transaction did not undo to the exact baseline")
	}
}

type drawioPreparerView struct {
	failingView
	prepares int
	token    editor.ExportToken
	prepare  func() (editor.ViewExportRequest, error)
}

func (v *drawioPreparerView) PrepareExportRequest() (editor.ViewExportRequest, error) {
	v.prepares++
	if v.prepare != nil {
		return v.prepare()
	}
	return editor.ViewExportRequest{Page: editor.ExportPage{Index: 1, ID: "p2"}, Token: v.token}, nil
}

func (v *drawioPreparerView) ExportRequestToken() editor.ExportToken { return v.token }

func TestDrawioWorkerRefusesBeforePreparationWithoutRunner(t *testing.T) {
	for _, change := range []string{"source", "nil-view", "group", "oversize", "flush"} {
		t.Run(change, func(t *testing.T) {
			a, tab, f, _, _, _ := drawioWorkerApp(t)
			view := &drawioPreparerView{}
			tab.View = view
			expected := ""
			switch change {
			case "source":
				if !a.setMode(tab, "source") {
					t.Fatal(a.Message)
				}
				expected = "阅读或分屏"
			case "nil-view":
				tab.View = nil
				expected = "视图尚未准备好"
			case "group":
				if err := tab.Doc.Buffer.BeginGroup(); err != nil {
					t.Fatal(err)
				}
				defer tab.Doc.Buffer.CancelGroup()
				expected = "未完成的编辑事务"
			case "oversize":
				tab.Doc.Buffer = text.New(strings.Repeat("x", int(enginecatalog.DrawioSourceMaxBytes)+1))
				tab.Editor.Buffer = tab.Doc.Buffer
				expected = "图纸源码超过 32 MiB 导出上限"
			case "flush":
				view.flushErr = errors.New("draft remains")
				expected = "draft remains"
			}
			a.exportCurrent()
			if a.exportJob != nil || len(f.calls) != 0 || view.prepares != 0 || !strings.Contains(a.Message, expected) {
				t.Fatalf("refusal %s: job=%v calls=%d prepares=%d message=%q", change, a.exportJob != nil, len(f.calls), view.prepares, a.Message)
			}
		})
	}
}

func TestDrawioWorkerRefusesPreparationReentryWithoutRunner(t *testing.T) {
	for _, change := range []string{"buffer", "version", "view"} {
		t.Run(change, func(t *testing.T) {
			a, tab, f, _, _, _ := drawioWorkerApp(t)
			view := &drawioPreparerView{token: 7}
			view.prepare = func() (editor.ViewExportRequest, error) {
				switch change {
				case "buffer":
					tab.Doc.Buffer = text.New(drawioWorkerBody)
					tab.Editor.Buffer = tab.Doc.Buffer
				case "version":
					if err := tab.Doc.Buffer.ReplaceUTF16(0, 0, " "); err != nil {
						t.Fatal(err)
					}
				case "view":
					tab.View = &drawioPreparerView{token: 7}
				}
				return editor.ViewExportRequest{Page: editor.ExportPage{Index: 1, ID: "p2"}, Token: 7}, nil
			}
			tab.View = view
			a.exportCurrent()
			if a.exportJob != nil || len(f.calls) != 0 || view.prepares != 1 ||
				!strings.Contains(a.Message, "导出准备期间文档或格式已变化") {
				t.Fatalf("reentry %s escaped: job=%v calls=%d prepares=%d message=%q",
					change, a.exportJob != nil, len(f.calls), view.prepares, a.Message)
			}
		})
	}
}

func TestDrawioWorkerRejectsStalePageViewAndSourceBeforePublish(t *testing.T) {
	for _, change := range []string{"B-A-B", "same-index-reparse", "release", "post-dialog-page", "post-dialog-view", "post-dialog-version", "post-dialog-editor", "post-dialog-name"} {
		t.Run(change, func(t *testing.T) {
			a, tab, f, queue, root, _ := drawioWorkerApp(t)
			tester := ui.NewTester(a.View, 1180, 780)
			tester.Frame()
			clickDrawioPageButton(t, tester, "下一页")
			out := filepath.Join(root, "stale.svg")
			dialogs := 0
			a.exportDialog = func(mygo.SaveDialogOptions) (string, error) {
				dialogs++
				if change == "post-dialog-page" {
					beforeToken := tab.View.(editor.ExportRequestTokenProvider).ExportRequestToken()
					clickDrawioPageButton(t, tester, "上一页")
					afterToken := tab.View.(editor.ExportRequestTokenProvider).ExportRequestToken()
					if afterToken <= beforeToken {
						t.Fatalf("post-dialog page transition did not advance token: %d -> %d; texts=%q", beforeToken, afterToken, tester.Texts())
					}
				} else if change == "post-dialog-view" {
					token := tab.View.(editor.ExportRequestTokenProvider).ExportRequestToken()
					tab.View = &drawioPreparerView{token: token}
				} else if change == "post-dialog-version" {
					if err := tab.Doc.Buffer.ReplaceUTF16(0, 0, " "); err != nil {
						t.Fatal(err)
					}
				} else if change == "post-dialog-editor" {
					tab.Editor = editor.New(tab.Doc.Buffer)
				} else if change == "post-dialog-name" {
					tab.Name = "renamed.drawio"
				}
				return out, nil
			}

			a.exportCurrent()
			job, call := a.exportJob, awaitExportCall(t, f)
			if job == nil || call.source.Page == nil || *call.source.Page != (engineipc.RequestPage{Index: 1, ID: "p2"}) {
				t.Fatalf("initial draw.io request mismatch: %+v", call.source.Page)
			}
			// The export-progress row moves the graph toolbar. Refresh the tester's
			// hit targets before exercising a real page transition while the job
			// remains active.
			tester.Frame()
			switch change {
			case "B-A-B":
				beforeToken := tab.View.(editor.ExportRequestTokenProvider).ExportRequestToken()
				clickDrawioPageButton(t, tester, "上一页")
				middleToken := tab.View.(editor.ExportRequestTokenProvider).ExportRequestToken()
				clickDrawioPageButton(t, tester, "下一页")
				afterToken := tab.View.(editor.ExportRequestTokenProvider).ExportRequestToken()
				if middleToken <= beforeToken || afterToken <= middleToken {
					t.Fatalf("B-A-B transitions did not advance tokens: %d -> %d -> %d; texts=%q",
						beforeToken, middleToken, afterToken, tester.Texts())
				}
			case "same-index-reparse":
				reparsed := strings.Replace(tab.Doc.Buffer.String(), `value="PAGE_TWO"`, `value="PAGE_TWO_REPARSED"`, 1)
				if err := tab.Doc.Buffer.ReplaceUTF16(0, tab.Doc.Buffer.LenUTF16(), reparsed); err != nil {
					t.Fatal(err)
				}
			case "release":
				if !a.setMode(tab, "source") {
					t.Fatal(a.Message)
				}
			}
			call.response <- drawioWorkerOutcome()
			awaitExportDone(t, job)
			drainOneExportCallback(t, queue)
			postDialog := strings.HasPrefix(change, "post-dialog-")
			if postDialog && dialogs != 1 {
				t.Fatal("post-dialog mutation did not reach the final recheck")
			}
			job.mu.Lock()
			payloadCleared := job.result.Payload == nil && len(job.result.Warnings) == 0
			job.mu.Unlock()
			if (!postDialog && dialogs != 0) || a.exportJob != nil || !payloadCleared {
				t.Fatalf("stale %s passed a publication gate: dialogs=%d job=%v payloadCleared=%v",
					change, dialogs, a.exportJob != nil, payloadCleared)
			}
			if _, err := os.Stat(out); !os.IsNotExist(err) {
				t.Fatalf("stale %s published a file: %v", change, err)
			}
		})
	}
}
