package graphview

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

func TestNativeSceneRefreshSameVersionDrawPreservesDocument(t *testing.T) {
	for _, dirty := range []bool{false, true} {
		name := "clean"
		if dirty {
			name = "dirty-with-undo"
		}
		t.Run(name, func(t *testing.T) {
			v, requests, _ := nativeSceneFixture(t)
			doc := document.New("literal source🙂\r\n")
			doc.IsNew, doc.ReadOnly = false, true
			if dirty {
				if err := doc.Buffer.ReplaceUTF16(0, 7, "changed"); err != nil {
					t.Fatal(err)
				}
			}
			if err := doc.Buffer.SetSelection(6, 1); err != nil {
				t.Fatal(err)
			}
			v.host.Buffer = doc.Buffer
			before := doc.Buffer.Snapshot()
			version, selection, history := doc.Buffer.Version(), doc.Buffer.Selection(), doc.Buffer.History()
			if doc.Dirty() != dirty || doc.Buffer.CanUndo() != dirty {
				t.Fatal("document fixture does not exercise intended dirty/history state")
			}
			_ = v.Warnings()
			initial := literalNativeScene()
			initial.Warnings = []editor.SceneWarning{{Code: "OLD", Message: "old engine warning"}}
			(*requests)[0].complete(initial, nil)
			tester := ui.NewTester(v.Draw, 760, 480)
			if err := v.NavigateOutlineItem("fold"); err != nil {
				t.Fatal(err)
			}
			v.zoom, v.panX, v.panY, v.fitted = 1.25, 17, 23, true
			tester.Frame()
			if len(v.labels.layouts) == 0 {
				t.Fatal("expected actual drawn label cache before refresh")
			}
			box := v.canvasRect
			x, y := box.X+box.W-20, box.Y+box.H-20
			// A real secondary-button pan retains the selected node. Refresh
			// must stop this held gesture without resetting the viewport.
			tester.PressButton(1, x, y)
			tester.Move(x-9, y-7)
			if !v.panning || v.selectedID != "fold" {
				t.Fatal("native held-pan fixture did not capture")
			}
			zoom, panX, panY, selected := v.zoom, v.panX, v.panY, v.selectedID
			v.pendingCenter = "root"
			v.RefreshScene()
			if len(*requests) != 1 || v.started || v.loading || !v.stale || v.err != nil || len(v.warnings) != 0 ||
				v.pendingCenter != "" || v.panning || v.labels.layouts != nil || v.frameLabelUses != 0 || v.frameHiddenLabels != 0 {
				t.Fatal("refresh did not defer request/clear transient state", len(*requests), v.status())
			}
			if !reflect.DeepEqual(v.scene, initial.Scene) || v.selectedID != selected || v.zoom != zoom || v.panX != panX || v.panY != panY || !v.fitted {
				t.Fatal("refresh replaced retained scene or viewport")
			}
			if _, ok := v.CurrentOutlineItem(); ok {
				t.Fatal("stale selection presented as current")
			}
			tester.Frame()
			if len(*requests) != 2 || !v.loading || !v.stale || !tester.HasText("正在更新只读预览；当前显示的是旧内容") {
				t.Fatal("same-version draw did not request one disclosed replacement")
			}
			tester.Move(x-60, y-60)
			tester.ReleaseButton(1, x-60, y-60)
			if v.panX != panX || v.panY != panY {
				t.Fatal("held pre-refresh pan resumed after refresh")
			}
			(*requests)[1].complete(literalNativeScene(), nil)
			for i := 0; i < 3; i++ {
				tester.Frame()
				_ = v.Warnings()
			}
			if len(*requests) != 2 || v.loading || v.stale || v.err != nil || tester.FramePending() || len(v.Warnings()) != 0 {
				t.Fatal("successful refresh did not settle without duplicate requests")
			}
			if v.selectedID != selected || v.zoom != zoom || v.panX != panX || v.panY != panY || !v.fitted {
				t.Fatal("replacement lost existing visible selection/viewport")
			}
			if doc.Buffer != v.host.Buffer || !doc.Buffer.EqualSnapshot(before) || doc.Buffer.Version() != version ||
				doc.Buffer.Selection() != selection || doc.Buffer.History() != history || doc.Dirty() != dirty || !doc.ReadOnly || !v.host.ReadOnly {
				t.Fatal("refresh changed document state")
			}
		})
	}
}

func TestNativeSceneRefreshCoalescesAndRejectsOldCallbacks(t *testing.T) {
	v, requests, _ := nativeSceneFixture(t)
	_ = v.Warnings()
	old, cancel := (*requests)[0], v.cancel
	cancels := 0
	v.cancel = func() {
		cancels++
		cancel()
		old.complete(literalNativeScene(), nil) // synchronous cancellation completion
	}
	epoch := v.epoch
	v.RefreshScene()
	v.RefreshScene()
	if cancels != 1 || !old.canceled || len(*requests) != 1 || v.epoch <= epoch || len(v.scene.Nodes) != 0 || v.loading || v.started || v.stale {
		t.Fatal("repeated refresh duplicated cancellation or accepted old callback")
	}
	_ = v.Warnings()
	_ = v.Warnings()
	if len(*requests) != 2 || (*requests)[1].request.Epoch <= old.request.Epoch {
		t.Fatal("refresh did not coalesce into one new generation")
	}
	old.complete(editor.SceneResult{}, errors.New("late old error"))
	old.complete(literalNativeScene(), nil)
	if v.err != nil || !v.loading || len(v.scene.Nodes) != 0 {
		t.Fatal("late old success/error replaced active request")
	}
	second := (*requests)[1]
	v.RefreshScene()
	if !second.canceled || len(*requests) != 2 {
		t.Fatal("active replacement not canceled/deferred")
	}
	_ = v.Warnings()
	third := (*requests)[2]
	v.Release()
	if !third.canceled || v.started || v.loading || len(v.scene.Nodes) != 0 {
		t.Fatal("release did not clear active replacement")
	}
	second.complete(literalNativeScene(), nil)
	third.complete(literalNativeScene(), nil)
	third.complete(editor.SceneResult{}, errors.New("released error"))
	if v.started || v.loading || v.err != nil || len(v.scene.Nodes) != 0 || v.index != nil {
		t.Fatal("late completion resurrected released scene")
	}
	v.RefreshScene()
	if len(*requests) != 3 || v.started || v.stale {
		t.Fatal("refresh after Release eagerly restarted or retained stale content")
	}
	_ = v.Warnings()
	_ = v.Warnings()
	if len(*requests) != 4 || (*requests)[3].request.Epoch <= third.request.Epoch {
		t.Fatal("released view did not lazily start one new request")
	}
	v.Release()
}

func TestNativeSceneRefreshFailureRetainsStaleThenDropsAbsentSelection(t *testing.T) {
	v, requests, _ := nativeSceneFixture(t)
	_ = v.Warnings()
	initial := literalNativeScene()
	(*requests)[0].complete(initial, nil)
	if err := v.NavigateOutlineItem("fold"); err != nil {
		t.Fatal(err)
	}
	v.zoom, v.panX, v.panY = 1.5, 19, -31
	before, version, history, selection := v.host.Buffer.String(), v.host.Buffer.Version(), v.host.Buffer.History(), v.host.Buffer.Selection()
	v.RefreshScene()
	if err := v.NavigateOutlineItem("root"); !errors.Is(err, editor.ErrOutlineStale) || v.selectedID != "fold" || v.pendingCenter != "" {
		t.Fatal("stale navigation changed target/viewport", err)
	}
	if len(*requests) != 2 {
		t.Fatal("navigation must use the same single lazy request")
	}
	(*requests)[1].complete(editor.SceneResult{}, errors.New("selected package unavailable"))
	for i := 0; i < 3; i++ {
		warnings := v.Warnings()
		if len(warnings) != 1 || !strings.Contains(warnings[0], "旧预览未更新") || !strings.Contains(warnings[0], "selected package unavailable") {
			t.Fatal("failed refresh hid retained stale content", warnings)
		}
	}
	if len(*requests) != 2 || !v.stale || v.loading || !reflect.DeepEqual(v.scene, initial.Scene) || v.selectedID != "fold" {
		t.Fatal("failure retried or discarded retained content")
	}
	if err := v.NavigateOutlineItem("root"); !errors.Is(err, editor.ErrOutlineStale) {
		t.Fatal("failed refresh allowed navigation", err)
	}
	v.RefreshScene()
	_ = v.Warnings()
	replacement := literalNativeScene()
	replacement.Scene.Nodes = replacement.Scene.Nodes[:1]
	replacement.Scene.Edges = nil
	(*requests)[2].complete(replacement, nil)
	if v.selectedID != "" || v.stale || v.err != nil || len(v.scene.Nodes) != 1 || len(v.index) != 1 || len(v.Warnings()) != 0 {
		t.Fatal("successful replacement retained missing selection/stale failure")
	}
	if v.zoom != 1.5 || v.panX != 19 || v.panY != -31 || v.host.Buffer.String() != before || v.host.Buffer.Version() != version ||
		v.host.Buffer.History() != history || v.host.Buffer.Selection() != selection {
		t.Fatal("failed/retried refresh changed source or viewport")
	}
}

func TestNativeSceneRefreshCancelMayRelease(t *testing.T) {
	v, requests, _ := nativeSceneFixture(t)
	_ = v.Warnings()
	(*requests)[0].complete(literalNativeScene(), nil)
	v.RefreshScene()
	_ = v.Warnings()
	old, cancel := (*requests)[1], v.cancel
	v.cancel = func() {
		cancel()
		v.Release()
		old.complete(literalNativeScene(), nil)
	}
	v.RefreshScene()
	if !old.canceled || v.buffer != nil || v.started || v.loading || v.stale || v.cancel != nil || v.err != nil || len(v.scene.Nodes) != 0 || v.index != nil {
		t.Fatal("refresh wrote old state after cancellation released its owner")
	}
	if len(*requests) != 2 {
		t.Fatal("cancellation/Release started an extra request")
	}
}

func TestNativeSceneRefreshOptionalReadonlyContract(t *testing.T) {
	capability := *MindmapAdapter().SceneEngine
	readonly := MindmapAdapter().Create(editor.Host{ReadOnly: true, Buffer: text.New(testMindmap), SceneEngine: &capability})
	if _, ok := readonly.(editor.SceneRefresher); !ok {
		t.Fatal("readonly scene lacks optional refresh")
	}
	for _, view := range []editor.View{
		MindmapAdapter().Create(editor.Host{Buffer: text.New(testMindmap), SceneEngine: &capability}),
		MindmapAdapter().Create(editor.Host{ReadOnly: true, Buffer: text.New(testMindmap)}),
		DrawioAdapter().Create(editor.Host{ReadOnly: true, Buffer: text.New(testDrawio), SceneEngine: &capability}),
	} {
		if _, ok := view.(editor.SceneRefresher); ok {
			t.Fatalf("refresh leaked into non-scene view %T", view)
		}
	}
}
