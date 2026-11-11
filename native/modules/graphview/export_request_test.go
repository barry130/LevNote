package graphview

import (
	"errors"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/drawio"
	"levnote.local/native/ui/editor"
)

func TestDrawioAdapterAdvertisesViewRequiredFourthEngine(t *testing.T) {
	capability := DrawioAdapter().ExportEngine
	if capability == nil {
		t.Fatal("draw.io export capability is missing")
	}
	if *capability != (editor.ExportEngine{
		ID:           "levnote.drawio-svg",
		Format:       "svg",
		RequiresView: true,
	}) {
		t.Fatalf("draw.io export capability = %+v", *capability)
	}
}

func TestDrawioExportRequestTracksPageCacheAndReleaseEpoch(t *testing.T) {
	buffer := text.New(`<mxfile>` + pageXML("p1", "PAGE_ONE") + pageXML("p2", "PAGE_TWO") + `</mxfile>`)
	v := newView(kindDrawio, editor.Host{Buffer: buffer})
	tester := ui.NewTester(v.Draw, 960, 640)

	first, err := v.PrepareExportRequest()
	if err != nil {
		t.Fatal(err)
	}
	if first.Page != (editor.ExportPage{Index: 0, ID: "p1"}) || first.Token != v.ExportRequestToken() {
		t.Fatalf("first request = %+v token=%d", first, v.ExportRequestToken())
	}
	if token := v.ExportRequestToken(); token != first.Token {
		t.Fatalf("preparation token changed after return: %d -> %d", first.Token, token)
	}
	again, err := v.PrepareExportRequest()
	if err != nil || again != first {
		t.Fatalf("cache-hit preparation = %+v, %v", again, err)
	}
	tester.Frame()
	_ = v.Warnings()
	if token := v.ExportRequestToken(); token != first.Token {
		t.Fatalf("unchanged Draw/Warnings advanced token: %d -> %d", first.Token, token)
	}

	if !v.changePage(1) {
		t.Fatal("could not select page B")
	}
	second, err := v.PrepareExportRequest()
	if err != nil {
		t.Fatal(err)
	}
	if second.Page != (editor.ExportPage{Index: 1, ID: "p2"}) || second.Token <= first.Token {
		t.Fatalf("page B request = %+v, previous=%d", second, first.Token)
	}
	if !v.changePage(0) {
		t.Fatal("could not return to page A")
	}
	third, err := v.PrepareExportRequest()
	if err != nil {
		t.Fatal(err)
	}
	if third.Page != (editor.ExportPage{Index: 0, ID: "p1"}) || third.Token <= second.Token {
		t.Fatalf("A -> B -> A request = %+v, page B token=%d", third, second.Token)
	}

	staleSource := strings.Replace(buffer.String(), `value="PAGE_ONE"`, `value="PAGE_ONE_REPARSED"`, 1)
	if err := buffer.ReplaceUTF16(0, buffer.LenUTF16(), staleSource); err != nil {
		t.Fatal(err)
	}
	reparsed, err := v.PrepareExportRequest()
	if err != nil {
		t.Fatal(err)
	}
	if reparsed.Page != (editor.ExportPage{Index: 0, ID: "p1"}) || reparsed.Token <= third.Token {
		t.Fatalf("same-index reparse request = %+v, previous=%d", reparsed, third.Token)
	}

	beforeRelease := reparsed.Token
	v.Release()
	released := v.ExportRequestToken()
	if released <= beforeRelease {
		t.Fatalf("Release did not advance token: %d -> %d", beforeRelease, released)
	}
	reused, err := v.PrepareExportRequest()
	if err != nil {
		t.Fatal(err)
	}
	if reused.Page != (editor.ExportPage{Index: 0, ID: "p1"}) || reused.Token < released || reused.Token != v.ExportRequestToken() {
		t.Fatalf("released view reuse = %+v, release token=%d", reused, released)
	}
}

func TestDrawioExportRequestRefusesInvalidState(t *testing.T) {
	t.Run("nil buffer", func(t *testing.T) {
		v := newView(kindDrawio, editor.Host{})
		if _, err := v.PrepareExportRequest(); !errors.Is(err, ErrViewExportNoBuffer) {
			t.Fatal(err)
		}
	})
	t.Run("incompatible view", func(t *testing.T) {
		v := newView(kindMindmap, editor.Host{Buffer: text.New(testMindmap)})
		if _, err := v.PrepareExportRequest(); !errors.Is(err, ErrViewExportKind) {
			t.Fatal(err)
		}
	})
	t.Run("open group", func(t *testing.T) {
		buffer := text.New(testDrawio)
		if err := buffer.BeginGroup(); err != nil {
			t.Fatal(err)
		}
		defer buffer.CancelGroup()
		v := newView(kindDrawio, editor.Host{Buffer: buffer})
		if _, err := v.PrepareExportRequest(); !errors.Is(err, ErrViewExportGroup) {
			t.Fatal(err)
		}
	})
	t.Run("parse failure", func(t *testing.T) {
		v := newView(kindDrawio, editor.Host{Buffer: text.New(`<mxfile>`)})
		if _, err := v.PrepareExportRequest(); !errors.Is(err, drawio.ErrInvalidXML) {
			t.Fatal(err)
		}
	})
	t.Run("source limit", func(t *testing.T) {
		buffer := text.New(strings.Repeat("x", drawio.DefaultMaxBytes+1))
		v := newView(kindDrawio, editor.Host{Buffer: buffer})
		if _, err := v.PrepareExportRequest(); !errors.Is(err, drawio.ErrLimitExceeded) {
			t.Fatal(err)
		}
	})
	t.Run("missing page", func(t *testing.T) {
		buffer := text.New(`<mxfile>` + pageXML("p1", "one") + pageXML("p2", "two") + `</mxfile>`)
		v := newView(kindDrawio, editor.Host{Buffer: buffer})
		v.ensureSynced(nil)
		v.setPageIndex(2)
		if _, err := v.PrepareExportRequest(); !errors.Is(err, drawio.ErrPageNotFound) {
			t.Fatal(err)
		}
	})
	t.Run("invalid index", func(t *testing.T) {
		v := newView(kindDrawio, editor.Host{Buffer: text.New(testDrawio)})
		v.ensureSynced(nil)
		v.setPageIndex(maxViewExportPageIndex + 1)
		if _, err := v.PrepareExportRequest(); !errors.Is(err, ErrViewExportPageIndex) {
			t.Fatal(err)
		}
	})
	t.Run("overlong page ID", func(t *testing.T) {
		id := strings.Repeat("x", maxViewExportPageIDBytes+1)
		buffer := text.New(`<mxfile>` + pageXML(id, "one") + `</mxfile>`)
		v := newView(kindDrawio, editor.Host{Buffer: buffer})
		if _, err := v.PrepareExportRequest(); !errors.Is(err, ErrViewExportPageID) {
			t.Fatal(err)
		}
	})
}

func TestDrawioExportRequestDoesNotFlushPendingDraft(t *testing.T) {
	buffer := text.New(testDrawio)
	v := newView(kindDrawio, editor.Host{Buffer: buffer})
	tester := ui.NewTester(v.Draw, 960, 640)
	focusDraft(t, tester)
	tester.Compose("pending🙂", 3)

	draft := v.draft
	source, version, history := buffer.String(), buffer.Version(), buffer.History()
	pageIndex, selectedID := v.pageIndex, v.selectedID
	request, err := v.PrepareExportRequest()
	if !errors.Is(err, ErrViewExportPending) || request != (editor.ViewExportRequest{}) {
		t.Fatalf("pending draft preparation = %+v, %v", request, err)
	}
	if v.draft != draft || !v.pendingDraft() || buffer.String() != source || buffer.Version() != version ||
		buffer.History() != history || v.pageIndex != pageIndex || v.selectedID != selectedID {
		t.Fatal("preparation discarded or published a pending draft")
	}
}

func TestDrawioExportRequestKeepsCanceledGestureWarningAfterReparse(t *testing.T) {
	buffer := text.New(`<mxfile>` + pageXML("p1", "one") + pageXML("p2", "two") + `</mxfile>`)
	v := newView(kindDrawio, editor.Host{Buffer: buffer})
	v.ensureSynced(nil)
	changedSource := strings.Replace(buffer.String(), `value="one"`, `value="one changed"`, 1)
	if err := buffer.ReplaceUTF16(0, buffer.LenUTF16(), changedSource); err != nil {
		t.Fatal(err)
	}
	v.draggingID = "same"

	request, err := v.PrepareExportRequest()
	if err != nil {
		t.Fatal(err)
	}
	if request.Page != (editor.ExportPage{Index: 0, ID: "p1"}) {
		t.Fatalf("reparsed request = %+v", request)
	}
	if v.draggingID != "" {
		t.Fatal("stale source synchronization did not cancel the graph gesture")
	}
	const warning = "图形源内容已改变，未完成的拖动或连接操作已取消；请重新操作"
	if !strings.Contains(strings.Join(v.Warnings(), "\n"), warning) {
		t.Fatalf("reparse warning list does not preserve canceled gesture: %q", v.Warnings())
	}
}

func TestDrawioExportRequestStopsSyncAfterReentrantOwnerChange(t *testing.T) {
	setup := func(t *testing.T, callback func(*view, *text.Buffer)) (*view, *text.Buffer, func()) {
		t.Helper()
		buffer := text.New(`<mxfile>` + pageXML("p1", "one") + pageXML("p2", "two") + `</mxfile>`)
		reenter := false
		var v *view
		v = newView(kindDrawio, editor.Host{
			Buffer: buffer,
			Report: func(error) {
				if reenter {
					callback(v, buffer)
				}
			},
		})
		v.ensureSynced(nil)
		changedSource := strings.Replace(buffer.String(), `value="one"`, `value="one changed"`, 1)
		if err := buffer.ReplaceUTF16(0, buffer.LenUTF16(), changedSource); err != nil {
			t.Fatal(err)
		}
		v.draggingID = "same"
		return v, buffer, func() { reenter = true }
	}
	requireRefused := func(t *testing.T, v *view) {
		t.Helper()
		request, err := v.PrepareExportRequest()
		if !errors.Is(err, ErrViewExportChanged) || request != (editor.ViewExportRequest{}) {
			t.Fatalf("reentrant preparation = %+v, %v", request, err)
		}
	}

	t.Run("nil buffer", func(t *testing.T) {
		v, _, enter := setup(t, func(v *view, _ *text.Buffer) { v.host.Buffer = nil })
		before := v.diagram
		enter()
		requireRefused(t, v)
		if v.diagram != before {
			t.Fatal("outer synchronization published a cache after the buffer was removed")
		}
	})
	t.Run("replacement buffer", func(t *testing.T) {
		v, _, enter := setup(t, func(v *view, _ *text.Buffer) {
			v.host.Buffer = text.New(`<mxfile>` + pageXML("other", "replacement") + `</mxfile>`)
		})
		before := v.diagram
		enter()
		requireRefused(t, v)
		if v.diagram != before {
			t.Fatal("outer synchronization published a cache for the old buffer")
		}
	})
	t.Run("release", func(t *testing.T) {
		v, _, enter := setup(t, func(v *view, _ *text.Buffer) { v.Release() })
		enter()
		requireRefused(t, v)
		if v.diagram != nil {
			t.Fatal("outer synchronization resurrected a released cache")
		}
	})
	t.Run("source change", func(t *testing.T) {
		v, buffer, enter := setup(t, func(_ *view, buffer *text.Buffer) {
			changed := strings.Replace(buffer.String(), `value="one changed"`, `value="callback source"`, 1)
			_ = buffer.ReplaceUTF16(0, buffer.LenUTF16(), changed)
		})
		before := v.diagram
		enter()
		requireRefused(t, v)
		if v.diagram != before || v.lastSynced == buffer.Version() {
			t.Fatal("outer synchronization consumed source changed by its callback")
		}
	})
	t.Run("page A B A", func(t *testing.T) {
		v, _, enter := setup(t, func(v *view, _ *text.Buffer) {
			v.setPageIndex(1)
			v.setPageIndex(0)
		})
		before := v.diagram
		enter()
		requireRefused(t, v)
		if v.diagram != before || v.pageIndex != 0 {
			t.Fatal("outer synchronization published a callback-retargeted page")
		}
	})
	t.Run("pending callback draft", func(t *testing.T) {
		v, buffer, enter := setup(t, func(v *view, buffer *text.Buffer) {
			input := editor.New(text.New("callback draft"))
			draft := &fieldDraft{
				nodeID:        "same",
				pageID:        "p1",
				pageName:      "p1",
				pageIndex:     0,
				pageCount:     2,
				version:       buffer.Version(),
				buffer:        buffer,
				field:         fieldDrawioLabel,
				sourcePresent: true,
				base:          "one changed",
				sourceBase:    "one changed",
				input:         input,
			}
			v.draft = draft
		})
		source, history := buffer.String(), buffer.History()
		enter()
		request, err := v.PrepareExportRequest()
		if !errors.Is(err, ErrViewExportPending) || request != (editor.ViewExportRequest{}) {
			t.Fatalf("callback draft preparation = %+v, %v", request, err)
		}
		if v.draft == nil || !v.pendingDraft() || buffer.String() != source || buffer.History() != history {
			t.Fatal("synchronization discarded the callback-owned pending draft or source state")
		}
	})
}
