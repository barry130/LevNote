// Package graphview implements LevNote's native mindmap and draw.io canvas
// adapters. The host buffer is the sole document authority: the parsed format
// is a versioned read cache that is rebuilt whenever the buffer changes.
package graphview

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/drawio"
	"levnote.local/native/formats/mindmap"
	"levnote.local/native/ui/editor"
)

var ErrPNGUnsupported = errors.New("native graph PNG export is not implemented; use SVG")

type documentKind uint8

const (
	kindMindmap documentKind = iota + 1
	kindDrawio
)

const (
	mindmapTemplate = `{
  "format": "levnote.mindmap",
  "version": 1,
  "layout": "logicalStructure",
  "root": {
    "id": "root",
    "topic": "中心主题",
    "children": []
  }
}
`
	drawioTemplate = `<mxfile host="levnote"><diagram id="page-1" name="页面 1"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/></root></mxGraphModel></diagram></mxfile>`
)

func MindmapAdapter() editor.Adapter {
	return editor.Adapter{
		ID:   "mindmap",
		Name: "原生思维导图",
		Matches: func(path string) bool {
			return strings.EqualFold(filepath.Ext(path), ".mindmap")
		},
		Create: func(host editor.Host) editor.View {
			if host.ReadOnly && host.SceneEngine != nil {
				return newSceneView(host)
			}
			return newView(kindMindmap, host)
		},
		Outline:      newMindmapOutlineSource,
		Template:     func() string { return mindmapTemplate },
		Exports:      []string{"svg"},
		ExportEngine: &editor.ExportEngine{ID: "levnote.mindmap-svg", Format: "svg"},
		SceneEngine:  &editor.SceneCapability{ID: "levnote.mindmap-preview", Format: "mindmap-scene-v1"},
	}
}

func DrawioAdapter() editor.Adapter {
	return editor.Adapter{
		ID:   "drawio",
		Name: "原生 draw.io 图纸",
		Matches: func(path string) bool {
			return strings.EqualFold(filepath.Ext(path), ".drawio")
		},
		Create: func(host editor.Host) editor.View {
			return newView(kindDrawio, host)
		},
		Template: func() string { return drawioTemplate },
		Exports:  []string{"svg"},
		ExportEngine: &editor.ExportEngine{
			ID:           "levnote.drawio-svg",
			Format:       "svg",
			RequiresView: true,
		},
	}
}

type view struct {
	kind documentKind
	host editor.Host

	lastSynced             uint64
	syncedBuffer           *text.Buffer
	syncFailed             bool
	syncError              error
	mind                   *mindmap.Document
	diagram                *drawio.Document
	layout                 mindmap.LayoutResult
	warnings               []string
	selectedID             string
	field                  nodeField
	pageIndex              int
	draft                  *fieldDraft
	draftError             error
	flushing               bool
	linkGesture            fieldLinkTarget
	fieldHandle            ui.Handle
	linkHandle             ui.Handle
	linkEnabled            bool
	markerHandle           ui.Handle
	markerOpen             bool
	markerInfo             markerPresentation
	markerPress            markerGesture
	markerError            error
	tagDraft               *tagDraft
	tagOpen                bool
	tagEpoch               uint64
	releaseEpoch           uint64
	pageEpoch              uint64
	exportEpoch            uint64
	tagPress               tagGesture
	tagInfo                tagPresentation
	tagError               error
	tagHandle              ui.Handle
	tagAddHandle           ui.Handle
	tagPointerContinuation bool
	lastFieldReceipt       fieldFlushReceipt

	canvasHandle ui.Handle
	canvasRect   ui.Rect
	zoom         float32
	panX, panY   float32
	fitted       bool

	panning    bool
	panStartX  float32
	panStartY  float32
	panOriginX float32
	panOriginY float32
	draggingID string
	dragStartX float32
	dragStartY float32
	dragWorldX float32
	dragWorldY float32
	dragBaseX  float64
	dragBaseY  float64
	dragMoved  bool

	reparentSource       string
	connectSource        string
	frameNodes           int
	frameLabels          int // new ShapeText calls in the most recent canvas paint
	frameLabelUses       int
	frameHiddenLabels    int
	frameEdges           int
	labels               labelCache
	nodeIndex            map[string]int
	indexPage            *drawio.Page
	edgeScratch          [][2]float32
	outline              *mindmapOutlineSource
	pendingOutlineCenter string
}

func newView(kind documentKind, host editor.Host) *view {
	return &view{
		kind:         kind,
		host:         host,
		lastSynced:   ^uint64(0),
		zoom:         1,
		pageIndex:    0,
		canvasHandle: ui.Handle{},
	}
}

func (v *view) CompositionActive() bool {
	if v.draft != nil {
		if _, composing := v.draft.input.MarkedRange(); composing {
			return true
		}
	}
	if v.tagDraft != nil {
		if _, composing := v.tagDraft.input.MarkedRange(); composing {
			return true
		}
	}
	return false
}

func (v *view) Draw(c *ui.Context) {
	defer func() {
		if v.linkGesture.released {
			v.linkGesture = fieldLinkTarget{}
		}
		if v.markerPress.released {
			v.markerPress = markerGesture{}
		}
		if v.tagPress.released {
			v.tagPress = tagGesture{}
		}
	}()
	v.ensureSynced(c)
	if v.kind == kindMindmap && v.selectedMindNode() == nil {
		v.linkGesture = fieldLinkTarget{}
		v.linkEnabled = false
	}
	theme := c.Theme()
	ui.Column(c).Fill().MinHeight(0).Clip().Background(theme.Background).
		ObserveInput(v.observeGraphInspectorInput).HandleInput(v.graphInspectorScrollInput).Children(func() {
		v.drawToolbar(c)
		if len(v.warnings) > 0 {
			v.drawWarnings(c)
		}
		ui.Row(c).Grow(1).MinHeight(0).Clip().AlignItems(ui.Stretch).Children(func() {
			ui.Box(c).
				Key("graph-canvas").
				Grow(1).
				MinWidth(180).
				MinHeight(120).
				Clip().
				Focusable().
				Label(v.canvasLabel()).
				Role(ui.RoleGroup).
				Bind(&v.canvasHandle).
				HandleInput(v.handleCanvasInput).
				Draw(v.drawCanvas)
			v.drawInspector(c)
		})
	})
}

func (v *view) Release() {
	v.releaseEpoch++
	v.bumpExportEpoch()
	v.invalidateTagActions()
	v.tagInfo = tagPresentation{}
	v.tagPointerContinuation = false
	v.tagHandle.CancelFocus()
	v.tagAddHandle.CancelFocus()
	v.linkGesture = fieldLinkTarget{}
	v.markerPress = markerGesture{}
	v.markerInfo = markerPresentation{}
	v.markerOpen, v.markerError = false, nil
	v.markerHandle.CancelFocus()
	v.fieldHandle.CancelFocus()
	v.linkHandle.CancelFocus()
	// Hosts must call Flush first. Defensively preserve a failed draft even
	// when a caller releases directly: only geometry is disposable.
	if err := v.Flush(); err != nil {
		v.setDraftError(err)
	}
	if v.draft != nil {
		v.draft.input.ReleaseView()
		if !v.pendingGraphDraft() {
			v.draft = nil
		}
	}
	if v.tagDraft != nil {
		v.tagDraft.input.ReleaseView()
		if !v.pendingTagDraft() {
			v.tagDraft = nil
		}
	}
	v.panning = false
	v.cancelGraphGesture()
	v.canvasRect = ui.Rect{}
	v.mind = nil
	v.diagram = nil
	v.layout = mindmap.LayoutResult{}
	v.clearRenderCache()
	v.warnings = nil
	v.canvasHandle.CancelFocus()
	v.lastSynced = ^uint64(0)
	v.syncedBuffer = nil
	v.syncError = nil
	if v.outline != nil {
		v.outline.Release()
		v.outline = nil
	}
}

func (v *view) Warnings() []string {
	v.ensureSynced(nil)
	warnings := append([]string(nil), v.warnings...)
	if v.draftError != nil {
		warnings = append(warnings, "节点草稿尚未写回："+v.draftError.Error())
	}
	if v.tagError != nil {
		warnings = append(warnings, "标签操作："+v.tagError.Error())
	}
	return warnings
}

func (v *view) Export(format string) ([]byte, error) {
	if err := v.Flush(); err != nil {
		return nil, err
	}
	v.ensureSynced(nil)
	if v.syncError != nil {
		return nil, v.syncError
	}
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "svg", "image/svg+xml":
		switch v.kind {
		case kindMindmap:
			if v.mind == nil {
				return nil, errors.New("mindmap is not readable")
			}
			return exportMindmapSVG(v.mind, v.layout)
		case kindDrawio:
			if v.diagram == nil {
				return nil, errors.New("draw.io document is not readable")
			}
			data, issues, err := v.diagram.ExportSVG(v.pageIndex, drawio.SVGOptions{})
			if err != nil {
				return nil, err
			}
			for _, issue := range issues {
				v.addWarning(issue.Message)
			}
			return data, nil
		}
	case "png", "image/png":
		return nil, ErrPNGUnsupported
	}
	return nil, fmt.Errorf("unsupported export format %q", format)
}

func (v *view) canvasLabel() string {
	if v.kind == kindMindmap {
		return "思维导图画布"
	}
	if v.diagram != nil && v.pageIndex >= 0 && v.pageIndex < len(v.diagram.Pages) &&
		v.diagram.Pages[v.pageIndex].Name != "" {
		return "draw.io 画布：" + v.diagram.Pages[v.pageIndex].Name
	}
	return "draw.io 画布"
}

func (v *view) addWarning(message string) {
	if message == "" {
		return
	}
	for _, existing := range v.warnings {
		if existing == message {
			return
		}
	}
	v.warnings = append(v.warnings, message)
}

func (v *view) report(err error) {
	if err != nil && v.host.Report != nil {
		v.host.Report(err)
	}
}
