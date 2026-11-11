package graphview

import (
	"errors"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/drawio"
	"levnote.local/native/formats/mindmap"
)

// syncOwnership covers every value a Report callback could replace while a
// stale parse is being abandoned or completed. lastSynced/syncedBuffer also
// detect a nested synchronization performed by the callback.
type syncOwnership struct {
	buffer                  *text.Buffer
	version                 uint64
	releaseEpoch, pageEpoch uint64
	pageIndex               int
	lastSynced              uint64
	syncedBuffer            *text.Buffer
}

func (v *view) captureSyncOwnership() syncOwnership {
	ownership := syncOwnership{
		buffer:       v.host.Buffer,
		releaseEpoch: v.releaseEpoch,
		pageEpoch:    v.pageEpoch,
		pageIndex:    v.pageIndex,
		lastSynced:   v.lastSynced,
		syncedBuffer: v.syncedBuffer,
	}
	if ownership.buffer != nil {
		ownership.version = ownership.buffer.Version()
	}
	return ownership
}

func (v *view) sameSyncOwnership(ownership syncOwnership) bool {
	if v.host.Buffer != ownership.buffer || v.releaseEpoch != ownership.releaseEpoch ||
		v.pageEpoch != ownership.pageEpoch || v.pageIndex != ownership.pageIndex ||
		v.lastSynced != ownership.lastSynced || v.syncedBuffer != ownership.syncedBuffer {
		return false
	}
	return ownership.buffer == nil || ownership.buffer.Version() == ownership.version
}

func (v *view) ensureSynced(c *ui.Context) {
	if v.host.Buffer == nil {
		v.clearParsedCache()
		v.syncFailed = true
		v.warnings = []string{"没有可用的原生文本缓冲区"}
		return
	}
	current := v.host.Buffer.Version()
	if current == v.lastSynced && v.syncedBuffer == v.host.Buffer {
		return
	}
	ownership := v.captureSyncOwnership()
	gestureWarning := ""
	if v.draggingID != "" || v.connectSource != "" || v.reparentSource != "" {
		v.cancelGraphGesture()
		err := errors.New("图形源内容已改变，未完成的拖动或连接操作已取消；请重新操作")
		v.report(err)
		if !v.sameSyncOwnership(ownership) {
			return
		}
		gestureWarning = err.Error()
	}
	v.lastSynced, v.syncedBuffer = current, v.host.Buffer
	v.clearParsedCache()
	v.syncError = nil
	selected := v.selectedID
	pending := v.pendingDraft()
	graphPending := pending || v.pendingTagDraft()
	if err := v.previewSizeError(); err != nil {
		v.syncFailed, v.syncError = true, err
		v.warnings = []string{err.Error()}
		v.addWarning(gestureWarning)
		if graphPending {
			v.setDraftError(err)
		} else {
			v.clearDraft()
			v.report(err)
		}
		return
	}
	raw := v.host.Buffer.String()
	switch v.kind {
	case kindMindmap:
		document, issues, err := mindmap.Parse([]byte(raw), mindmap.ParseOptions{})
		if err != nil {
			v.mind = nil
			v.layout = mindmap.LayoutResult{}
			v.warnings = []string{"思维导图无法解析：" + err.Error()}
			v.syncFailed = true
			v.syncError = err
			v.addWarning(gestureWarning)
			v.report(err)
			return
		}
		v.mind = document
		v.warnings = mindmapWarnings(issues)
		v.addWarning(gestureWarning)
		v.syncFailed = false
		if selected == "" && !graphPending {
			selected = document.RootID()
		}
		if _, ok := document.FindNode(selected); !ok && !graphPending {
			selected = document.RootID()
		}
		v.selectedID = selected
		v.layout = mindmap.Layout(document, mindmap.NativeLayoutOptions())
		for _, issue := range v.layout.Issues {
			v.addWarning(issue.Message)
		}
		for _, issue := range mindmap.PresentationIssues(document) {
			v.addWarning(issue.Message)
		}
		v.refreshLabelDraft()
		v.refreshTagDraft()
	case kindDrawio:
		document, issues, err := drawio.Parse([]byte(raw), drawio.ParseOptions{})
		if err != nil {
			v.diagram = nil
			v.warnings = []string{"draw.io 图纸无法解析：" + err.Error()}
			v.syncFailed = true
			v.syncError = err
			v.addWarning(gestureWarning)
			v.report(err)
			return
		}
		v.diagram = document
		v.warnings = drawioWarnings(issues)
		v.addWarning(gestureWarning)
		if v.host.ReadOnly {
			v.addWarning("当前视图只读")
		}
		if pending {
			if page, err := v.draftPage(document, v.draft); err == nil {
				v.setPageIndex(page)
			} else {
				v.setPageIndex(-1)
				reported := v.captureSyncOwnership()
				v.setDraftError(err)
				if !v.sameSyncOwnership(reported) {
					return
				}
			}
		}
		if (v.pageIndex < 0 || v.pageIndex >= len(document.Pages)) && !pending {
			v.setPageIndex(0)
		}
		page := v.currentPage()
		if page == nil {
			v.syncFailed = true
			v.addWarning("正在编辑的图纸页面已不存在或无法确认；节点草稿仍被保留")
			return
		}
		if page.ReadOnly {
			v.addWarning(page.ReadOnlyReason)
		}
		v.syncFailed = false
		if !v.pageHasCell(selected) && !pending {
			selected = ""
			if len(page.Model.Nodes) > 0 {
				selected = page.Model.Nodes[0].ID
			}
		}
		v.selectedID = selected
		for _, node := range page.Model.Nodes {
			if len(node.PlainText) > maxGraphLabelBytes {
				v.addWarning("较长节点标签仅显示前 512 字节；完整文本仍保留，可在节点文本中编辑")
				break
			}
		}
		v.refreshLabelDraft()
	}
	if c != nil {
		c.Invalidate()
	}
}

func mindmapWarnings(issues []mindmap.Issue) []string {
	messages := make([]string, 0, len(issues))
	for _, issue := range issues {
		messages = append(messages, issue.Message)
	}
	return messages
}

func drawioWarnings(issues []drawio.Issue) []string {
	messages := make([]string, 0, len(issues))
	for _, issue := range issues {
		messages = append(messages, issue.Message)
	}
	return messages
}

func (v *view) refreshLabelDraft() {
	v.refreshDraft()
}

func (v *view) selectedMindNode() *mindmap.Node {
	if v.mind == nil || v.selectedID == "" {
		return nil
	}
	node, _ := v.mind.FindNode(v.selectedID)
	return node
}

func (v *view) selectedDrawioNode() *drawio.Node {
	if v.diagram == nil || v.pageIndex < 0 || v.pageIndex >= len(v.diagram.Pages) {
		return nil
	}
	for index := range v.diagram.Pages[v.pageIndex].Model.Nodes {
		node := &v.diagram.Pages[v.pageIndex].Model.Nodes[index]
		if node.ID == v.selectedID {
			return node
		}
	}
	return nil
}

func (v *view) currentPage() *drawio.Page {
	if v.diagram == nil || v.pageIndex < 0 || v.pageIndex >= len(v.diagram.Pages) {
		return nil
	}
	return &v.diagram.Pages[v.pageIndex]
}

func (v *view) pageHasCell(id string) bool {
	if id == "" {
		return false
	}
	page := v.currentPage()
	if page == nil {
		return false
	}
	for _, node := range page.Model.Nodes {
		if node.ID == id {
			return true
		}
	}
	for _, edge := range page.Model.Edges {
		if edge.ID == id {
			return true
		}
	}
	return false
}

func (v *view) writable() error {
	if v.host.Buffer == nil {
		return errors.New("no host buffer")
	}
	if v.host.ReadOnly {
		return errors.New("document is read-only")
	}
	if err := v.previewSizeError(); err != nil {
		return err
	}
	if v.kind == kindDrawio {
		page := v.currentPage()
		if page != nil && page.ReadOnly {
			return &drawio.ReadOnlyError{Page: v.pageIndex, Reason: page.ReadOnlyReason}
		}
	}
	return nil
}
