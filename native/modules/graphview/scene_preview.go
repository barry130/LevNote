package graphview

import (
	"errors"
	"fmt"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/enginescene"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

// sceneView never parses source or mounts an editable graph inspector. Its
// only document-derived presentation input is an owned neutral scene result.
type sceneView struct {
	host                                                                   editor.Host
	epoch, version                                                         uint64
	buffer                                                                 *text.Buffer
	started, loading, stale                                                bool
	cancel                                                                 func()
	scene                                                                  enginescene.Scene
	warnings                                                               []string
	err                                                                    error
	selectedID                                                             string
	pendingCenter                                                          string
	index                                                                  map[string]int
	labels                                                                 labelCache
	canvasHandle                                                           ui.Handle
	canvasRect                                                             ui.Rect
	zoom, panX, panY                                                       float32
	fitted, panning                                                        bool
	panStartX, panStartY, panOriginX, panOriginY                           float32
	frameNodes, frameEdges, frameLabels, frameLabelUses, frameHiddenLabels int
}

func newSceneView(host editor.Host) *sceneView {
	return &sceneView{host: host, zoom: 1}
}

func (v *sceneView) RefreshScene() {
	// Revoke the callback before cancel: a provider may finish synchronously
	// during cancellation. No state is written after calling external code.
	v.epoch++
	cancel := v.cancel
	v.cancel = nil
	v.started, v.loading, v.stale = false, false, len(v.scene.Nodes) > 0
	v.err, v.warnings = nil, nil
	v.pendingCenter = ""
	v.panning = false
	v.panStartX, v.panStartY, v.panOriginX, v.panOriginY = 0, 0, 0, 0
	v.labels = labelCache{}
	v.frameNodes, v.frameEdges, v.frameLabels, v.frameLabelUses, v.frameHiddenLabels = 0, 0, 0, 0, 0
	if cancel != nil {
		cancel()
	}
}

func (v *sceneView) ensureScene() {
	buffer := v.host.Buffer
	if buffer == nil {
		if !v.started {
			v.started = true
			v.err = errors.New("没有可用的文档缓冲区")
		}
		return
	}
	version := buffer.Version()
	if v.started && v.buffer == buffer && v.version == version {
		return
	}
	v.epoch++
	if v.cancel != nil {
		v.cancel()
		v.cancel = nil
	}
	v.buffer, v.version, v.started = buffer, version, true
	v.loading, v.stale, v.err = true, len(v.scene.Nodes) > 0, nil
	v.warnings = nil
	if v.host.SceneEngine == nil || v.host.BeginScene == nil {
		v.loading = false
		v.err = errors.New("此只读预览缺少场景引擎服务，请查看源码")
		return
	}
	epoch := v.epoch
	request := editor.SceneRequest{Capability: *v.host.SceneEngine, Epoch: epoch}
	cancel, err := v.host.BeginScene(v, request, func(result editor.SceneResult, err error) {
		if !v.started || v.epoch != epoch || v.buffer != buffer || v.host.Buffer != buffer || buffer.Version() != version {
			return
		}
		v.cancel = nil
		v.loading, v.err = false, err
		if err != nil {
			return
		}
		v.scene = result.Scene
		v.stale = false
		v.labels = labelCache{}
		v.index = make(map[string]int, len(result.Scene.Nodes))
		for i, node := range result.Scene.Nodes {
			v.index[node.ID] = i
		}
		if _, exists := v.index[v.selectedID]; !exists {
			v.selectedID = ""
		}
		v.warnings = make([]string, len(result.Warnings))
		for i, warning := range result.Warnings {
			v.warnings[i] = warning.Message
		}
	})
	if err != nil {
		v.loading, v.err = false, err
		return
	}
	v.cancel = cancel
}

func (v *sceneView) Release() {
	v.epoch++
	if v.cancel != nil {
		v.cancel()
		v.cancel = nil
	}
	v.started, v.loading, v.stale = false, false, false
	v.buffer = nil
	v.scene = enginescene.Scene{}
	v.index, v.warnings, v.err = nil, nil, nil
	v.labels = labelCache{}
	v.panning = false
	v.pendingCenter = ""
	v.canvasRect = ui.Rect{}
	v.canvasHandle.CancelFocus()
}

func (v *sceneView) Warnings() []string {
	v.ensureScene()
	warnings := make([]string, 0, len(v.warnings)+1)
	if status := v.status(); status != "" {
		warnings = append(warnings, status)
	}
	return append(warnings, v.warnings...)
}

func (v *sceneView) status() string {
	if v.err != nil {
		prefix := "只读预览不可用："
		if v.stale {
			prefix = "旧预览未更新："
		}
		return prefix + v.err.Error()
	}
	if v.loading {
		if v.stale {
			return "正在更新只读预览；当前显示的是旧内容"
		}
		return "正在加载只读预览…"
	}
	return ""
}

func (*sceneView) Export(string) ([]byte, error) {
	return nil, errors.New("只读场景不在视图内导出，请使用应用的 SVG 导出")
}

func (v *sceneView) NavigateOutlineItem(id string) error {
	v.ensureScene()
	if v.loading || v.stale || v.err != nil || len(v.scene.Nodes) == 0 {
		return editor.ErrOutlineStale
	}
	index, ok := v.index[id]
	if !ok {
		return editor.ErrOutlineReadOnlyHidden
	}
	v.selectedID = id
	v.pendingCenter = id
	v.fitted = true
	v.centerPendingScene(index)
	return nil
}

func (v *sceneView) centerPendingScene(index int) {
	if v.pendingCenter == "" || v.canvasRect.W <= 0 || v.canvasRect.H <= 0 {
		return
	}
	node := v.scene.Nodes[index]
	v.panX = sceneFinitePan(float64(v.canvasRect.W)/2 - (node.X+node.Width/2)*float64(v.zoom))
	v.panY = sceneFinitePan(float64(v.canvasRect.H)/2 - (node.Y+node.Height/2)*float64(v.zoom))
	v.pendingCenter = ""
}

func (v *sceneView) CurrentOutlineItem() (string, bool) {
	if v.loading || v.stale || v.err != nil {
		return "", false
	}
	_, ok := v.index[v.selectedID]
	return v.selectedID, ok
}

func (v *sceneView) Draw(c *ui.Context) {
	v.ensureScene()
	ui.Column(c).Fill().MinHeight(0).Clip().Children(func() {
		ui.Row(c.Key("scene-toolbar")).Wrap().Padding(6, 10).Gap(6).Background(c.Theme().Surface).Children(func() {
			ui.Text(c, "只读场景").FontSize(12)
			if ui.Button(c, "−").Label("缩小画布").Width(30).Clicked() {
				v.zoomAt(v.canvasRect.W/2, v.canvasRect.H/2, 0.85)
			}
			ui.Text(c, fmt.Sprintf("%d%%", int(v.zoom*100+0.5))).FontSize(12).Width(44)
			if ui.Button(c, "+").Label("放大画布").Width(30).Clicked() {
				v.zoomAt(v.canvasRect.W/2, v.canvasRect.H/2, 1.18)
			}
			if ui.Button(c, "适应").Label("适应画布").Height(28).Clicked() {
				v.fitScene()
			}
			if ui.Button(c, "源码").Label("查看只读文档源码").Height(28).Clicked() && v.host.ShowSource != nil {
				v.host.ShowSource(v)
			}
		})
		ui.Text(c, "此视图只读；节点属性、标签与图片引用请在源码中查看。").FontSize(11).Padding(4, 10).TextColor(c.Theme().TextMuted)
		if status := v.status(); status != "" {
			ui.Text(c.Key("scene-status"), status).FontSize(11).Padding(4, 10).TextColor(c.Theme().Warning)
		}
		showOverview := v.frameHiddenLabels > 0
		ui.Box(c.Key("scene-canvas")).FillWidth().Grow(1).MinHeight(0).Clip().Focusable().
			Label("只读思维导图画布").Role(ui.RoleGroup).Bind(&v.canvasHandle).
			HandleInput(v.sceneInput).Draw(func(p *ui.Painter, rect ui.Rect) {
			v.drawScene(p, rect)
			if showOverview != (v.frameHiddenLabels > 0) {
				c.Invalidate()
			}
		}).Children(func() {
			if showOverview {
				// An absolute semantic label does not shrink the canvas, so
				// visibility cannot oscillate as a footer changes culling.
				ui.Text(c.Key("scene-overview"), "概览中部分标签已隐藏；放大或选中节点查看").
					Absolute().Left(8).Bottom(5).FontSize(11).TextColor(c.Theme().TextMuted)
			}
		})
	})
}
