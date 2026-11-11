package graphview

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/formats/mindmap"
)

func (v *view) drawToolbar(c *ui.Context) {
	theme := c.Theme()
	ui.Row(c).
		Key("graph-toolbar").
		Wrap().
		MinHeight(46).
		Padding(6, 12).
		Gap(6).
		Background(theme.Surface).
		Border(1, theme.Border).
		Children(func() {
			readOnly := v.host.ReadOnly || v.host.Buffer == nil || v.syncFailed
			switch v.kind {
			case kindMindmap:
				v.drawMindmapToolbar(c, readOnly)
			case kindDrawio:
				v.drawDrawioToolbar(c, readOnly)
			}
			ui.Row(c).Shrink(0).Gap(6).Children(func() {
				if ui.Button(c, "−").Label("缩小画布").Width(30).Tooltip("缩小").Clicked() {
					v.zoomAroundCenter(0.85)
				}
				ui.Text(c, fmt.Sprintf("%d%%", int(v.zoom*100+0.5))).FontSize(12).Width(44).NoWrap()
				if ui.Button(c, "+").Label("放大画布").Width(30).Tooltip("放大").Clicked() {
					v.zoomAroundCenter(1.18)
				}
				if ui.Button(c, "适应").Label("适应画布").Height(30).Clicked() {
					v.fitCanvas()
				}
			})
		})
}

func (v *view) drawMindmapToolbar(c *ui.Context, readOnly bool) {
	if v.mind == nil {
		ui.Text(c, "解析失败").FontSize(12).TextColor(c.Theme().Warning)
		return
	}
	node := v.selectedMindNode()
	hasNode := node != nil
	hasParent := hasNode && node.ID != v.mind.RootID()
	if ui.Button(c, "+ 子主题").Shrink(0).Label("添加子节点").Disabled(readOnly || !hasNode).Clicked() {
		v.addMindmapChild()
	}
	if ui.Button(c, "+ 同级").Shrink(0).Label("添加同级主题").Disabled(readOnly || !hasParent).Clicked() {
		v.addMindmapSibling()
	}
	collapseLabel := "折叠"
	if node != nil && node.Collapsed != nil && *node.Collapsed {
		collapseLabel = "展开"
	}
	if ui.Button(c, collapseLabel).Shrink(0).Label(collapseLabel + "选中主题").Disabled(readOnly || !hasNode).Clicked() {
		v.toggleMindmapCollapse()
		v.focusLabel()
	}
	if ui.Button(c, "删除").Shrink(0).Label("删除选中主题").Disabled(readOnly || !hasParent).Clicked() {
		v.deleteMindmapNode()
	}
	reparentLabel := "改父级"
	if v.reparentSource != "" {
		reparentLabel = "选父级…"
	}
	if ui.Button(c, reparentLabel).Shrink(0).Label("更改主题父级").Disabled(readOnly || !hasParent).Clicked() {
		if err := v.Flush(); err != nil {
			return
		}
		if v.reparentSource == "" && hasNode {
			v.reparentSource = node.ID
			v.connectSource = ""
		} else {
			v.reparentSource = ""
		}
	}
	layoutLabel := "布局"
	switch v.mind.Layout {
	case mindmap.LayoutLeft:
		layoutLabel = "向左"
	case mindmap.LayoutRight:
		layoutLabel = "向右"
	case mindmap.LayoutOrg:
		layoutLabel = "组织"
	default:
		layoutLabel = "双向"
	}
	if ui.Button(c, layoutLabel).Shrink(0).Label("切换导图布局").Disabled(readOnly).Clicked() {
		v.cycleMindmapLayout()
	}
}

func (v *view) drawDrawioToolbar(c *ui.Context, readOnly bool) {
	page := v.currentPage()
	pageReadOnly := page == nil || page.ReadOnly
	ui.Row(c).Shrink(0).Gap(6).Children(func() {
		if ui.Button(c, "‹").Label("上一页").Width(30).Disabled(v.diagram == nil || v.pageIndex <= 0).Clicked() {
			v.changePage(v.pageIndex - 1)
		}
		pageName := "无页面"
		if page != nil && page.Name != "" {
			pageName = page.Name
		}
		ui.Text(c, truncate(pageName, 80)).Label("图纸页面名称").FontSize(12).MaxWidth(140).MaxLines(1).Ellipsis("…")
		if ui.Button(c, "›").Label("下一页").Width(30).Disabled(v.diagram == nil || v.pageIndex+1 >= len(v.diagram.Pages)).Clicked() {
			v.changePage(v.pageIndex + 1)
		}
	})
	for _, shape := range []struct {
		label  string
		access string
		value  string
	}{
		{"矩形", "添加矩形", "rectangle"},
		{"椭圆", "添加椭圆", "ellipse"},
		{"菱形", "添加菱形", "diamond"},
		{"圆角", "添加圆角矩形", "rounded"},
	} {
		if ui.Button(c, shape.label).Shrink(0).Label(shape.access).Disabled(readOnly || pageReadOnly).Clicked() {
			v.addDiagramShape(shape.value)
		}
	}
	connectLabel := "连接"
	if v.connectSource != "" {
		connectLabel = "选终点…"
	}
	if ui.Button(c, connectLabel).Shrink(0).Label("连接两个节点").Disabled(readOnly || pageReadOnly || len(page.Model.Nodes) < 2).Clicked() {
		if err := v.Flush(); err != nil {
			return
		}
		if v.connectSource == "" {
			if v.selectedID != "" {
				v.connectSource = v.selectedID
				v.reparentSource = ""
			}
		} else {
			v.connectSource = ""
		}
	}
}

func (v *view) drawWarnings(c *ui.Context) {
	theme := c.Theme()
	text := v.warnings[0]
	if len(v.warnings) > 1 {
		text = fmt.Sprintf("%s（另有 %d 条兼容性提示）", text, len(v.warnings)-1)
	}
	ui.Text(c, text).FontSize(11).TextColor(theme.Warning).Padding(5, 14).Background(theme.SurfaceHover.Alpha(0.35))
}

func (v *view) drawInspector(c *ui.Context) {
	theme := c.Theme()
	ui.Scroll(c).
		Key("graph-inspector").
		Width(224).
		MinHeight(0).
		Clip().
		Padding(12).
		Gap(9).
		Background(theme.Surface).
		Border(1, theme.Border).
		Children(func() {
			ui.Text(c, "节点属性").Bold().FontSize(13)
			if v.selectedID == "" && !v.pendingGraphDraft() {
				ui.Text(c, "选择一个节点后可编辑文本与结构。").FontSize(12).TextColor(theme.TextMuted)
				return
			}
			if v.host.ReadOnly {
				ui.Text(c, "只读").FontSize(11).TextColor(theme.TextMuted)
			}
			readOnly := v.host.ReadOnly || v.syncFailed
			if page := v.currentPage(); page != nil && page.ReadOnly {
				readOnly = true
			}
			if v.kind == kindMindmap {
				v.drawFieldSelector(c)
				if v.field == fieldNote {
					ui.Text(c, "纯文本备注").FontSize(11).TextColor(theme.TextMuted)
				} else if v.field == fieldHyperLink {
					ui.Text(c, "本地引用；网页和其他协议会被拒绝。清空后移除链接。").FontSize(11).TextColor(theme.TextMuted)
				}
			}
			if v.draft != nil {
				v.draft.input.ReadOnly = readOnly && !v.pendingDraft()
				height := float32(76)
				if v.draft.input.Buffer.LenUTF16() > 512 {
					height = 150
				}
				fieldBox := ui.Column(c).Key("graph-label").Height(height).FillWidth().Border(1, theme.Border)
				if v.kind == kindMindmap {
					if fieldBox.Shortcut(0, ui.KeyF6) {
						v.focusTagControl(c, v.draft.input, &v.tagHandle)
					}
					// In a property form Tab moves between controls. Scope the
					// override here; the source/drawio editor keeps literal indent.
					if fieldBox.Shortcut(0, ui.KeyTab) {
						v.focusTagControl(c, v.draft.input, &v.markerHandle)
					}
					if fieldBox.Shortcut(ui.Shift, ui.KeyTab) {
						v.focusTagControl(c, v.draft.input, &v.fieldHandle)
					}
				}
				fieldBox.Children(func() { v.draft.input.View(c) })
			}
			if v.draftError != nil {
				ui.Text(c, "节点草稿尚未写回："+v.draftError.Error()).MaxLines(2).Ellipsis("…").Tooltip(v.draftError.Error()).FontSize(11).TextColor(theme.Warning)
				ui.Row(c).FillWidth().Shrink(0).Gap(6).Children(func() {
					if ui.Button(c, "重试写回").Grow(1).Height(28).FontSize(11).Clicked() {
						v.setDraftError(v.Flush())
					}
					if v.pendingTagDraft() {
						v.drawTagButton(c, v.tagPresentation(), tagRecover, false, true)
					} else if ui.Button(c, "恢复为新文档").Grow(1).Height(28).FontSize(11).Clicked() {
						v.setDraftError(v.RecoverDraft())
					}
				})
			}
			if v.kind == kindMindmap {
				v.drawMarkers(c)
			}
			ui.Text(c, "ID").FontSize(11).TextColor(theme.TextMuted)
			ui.Text(c, truncate(v.selectedID, 80)).FontSize(11).MaxWidth(200).MaxLines(1).Ellipsis("…")
			switch v.kind {
			case kindMindmap:
				v.drawMindmapInspector(c, readOnly)
			case kindDrawio:
				v.drawDrawioInspector(c, readOnly)
			}
			if v.kind == kindMindmap {
				v.drawTags(c)
			}
			if v.connectSource != "" {
				ui.Text(c, "连接模式：点击目标节点").FontSize(11).TextColor(theme.Accent)
			}
			if v.reparentSource != "" {
				ui.Text(c, "改父级模式：点击新父节点").FontSize(11).TextColor(theme.Accent)
			}
			if v.kind == kindDrawio {
				if node := v.selectedDrawioNode(); node != nil && node.HTML {
					ui.Text(c, "标签含 HTML；编辑文本会替换该标签的原有格式，其他属性保留。").FontSize(11).TextColor(theme.Warning)
				}
			}
		})
}

func (v *view) drawMindmapInspector(c *ui.Context, readOnly bool) {
	node := v.selectedMindNode()
	if node == nil {
		return
	}
	theme := c.Theme()
	ui.Text(c, fmt.Sprintf("子节点 %d 个", len(node.Children))).FontSize(11).TextColor(theme.TextMuted)
	if v.drawFieldLink(c, node) {
		// Host.OpenReference may synchronously Release the whole parsed view.
		// Nothing below this point may dereference the released graph cache.
		return
	}
	if node.ID != v.mind.RootID() {
		if ui.Button(c, "提升到祖父级").Label("提升选中主题").FillWidth().Disabled(readOnly).Clicked() {
			v.promoteMindmapNode()
		}
	}
}

func (v *view) drawDrawioInspector(c *ui.Context, readOnly bool) {
	node := v.selectedDrawioNode()
	if node == nil {
		return
	}
	theme := c.Theme()
	ui.Text(c, fmt.Sprintf("%s · %.0f, %.0f", node.Shape, node.X, node.Y)).FontSize(11).TextColor(theme.TextMuted)
	if len(node.Unsupported) > 0 {
		ui.Text(c, "保留但未原生解释："+strings.Join(node.Unsupported, "、")).FontSize(11).TextColor(theme.Warning)
	}
	page := v.currentPage()
	if page != nil && page.ReadOnly {
		ui.Text(c, page.ReadOnlyReason).FontSize(11).TextColor(theme.Warning)
	}
	ui.Text(c, "拖动画布节点可直接改坐标；每次释放是一步撤销。").FontSize(11).TextColor(theme.TextMuted)
	if ui.Button(c, "撤销").FillWidth().Disabled(readOnly || v.host.Buffer == nil || !v.host.Buffer.CanUndo()).Clicked() {
		v.undo()
	}
}

func (v *view) currentLabel() string {
	if v.kind == kindMindmap {
		if node := v.selectedMindNode(); node != nil {
			return node.Topic
		}
	}
	if v.kind == kindDrawio {
		if node := v.selectedDrawioNode(); node != nil {
			return node.PlainText
		}
	}
	return ""
}

func truncate(value string, limit int) string {
	end := 0
	for count := 0; end < len(value); count++ {
		if count == limit {
			return value[:end] + "…"
		}
		_, width := utf8.DecodeRuneInString(value[end:])
		end += width
	}
	return value
}
