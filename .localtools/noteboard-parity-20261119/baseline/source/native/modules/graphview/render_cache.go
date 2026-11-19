package graphview

import (
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/formats/drawio"
	"levnote.local/native/formats/mindmap"
)

const maxGraphLabelBytes = mindmap.MaxDisplayLabelBytes
const maxGraphLabelLayouts = 256

type labelKey struct {
	id, font    string
	size, width float32
}

type labelCache struct {
	layouts map[labelKey]int
	entries [maxGraphLabelLayouts]labelEntry
	clock   uint64
}

type labelEntry struct {
	key    labelKey
	layout *ui.TextLayout
	used   uint64
}

func (cache *labelCache) get(key labelKey) *ui.TextLayout {
	if index, ok := cache.layouts[key]; ok {
		cache.clock++
		cache.entries[index].used = cache.clock
		return cache.entries[index].layout
	}
	return nil
}

func (cache *labelCache) put(key labelKey, layout *ui.TextLayout) {
	if cache.layouts == nil {
		cache.layouts = make(map[labelKey]int)
	}
	oldest := 0
	for i := 1; i < len(cache.entries); i++ {
		if cache.entries[i].used < cache.entries[oldest].used {
			oldest = i
		}
	}
	delete(cache.layouts, cache.entries[oldest].key)
	cache.clock++
	cache.entries[oldest] = labelEntry{key, layout, cache.clock}
	cache.layouts[key] = oldest
}

func (v *view) clearRenderCache() {
	v.labels = labelCache{}
	v.nodeIndex = nil
	v.indexPage = nil
	v.edgeScratch = nil
}

func intersects(a, b ui.Rect) bool {
	return a.X <= b.X+b.W && a.X+a.W >= b.X && a.Y <= b.Y+b.H && a.Y+a.H >= b.Y
}

// Bounding boxes intentionally overdraw diagonal paths. They preserve crossing
// edges whose endpoint nodes are both outside the viewport.
func polylineVisible(points [][2]float32, rect ui.Rect) bool {
	if len(points) < 2 {
		return false
	}
	minX, maxX, minY, maxY := points[0][0], points[0][0], points[0][1], points[0][1]
	for _, point := range points[1:] {
		minX, maxX = min32(minX, point[0]), max32(maxX, point[0])
		minY, maxY = min32(minY, point[1]), max32(maxY, point[1])
	}
	return intersects(ui.Rect{X: minX - 8, Y: minY - 8, W: maxX - minX + 16, H: maxY - minY + 16}, rect)
}

func boundedLabel(value string) string {
	if len(value) <= maxGraphLabelBytes {
		return value
	}
	end := maxGraphLabelBytes
	for end > 0 && !utf8.RuneStart(value[end]) {
		end--
	}
	return value[:end] + "…"
}

func boundedMindmapLabel(node *mindmap.Node, fallback string) string {
	return mindmap.DisplayLabel(node, fallback)
}

func (v *view) labelLayout(id, font string, size, width float32, value func() string) *ui.TextLayout {
	// At overview scale the small boxes cannot show readable labels. A stable
	// budget also prevents all-visible diagrams from churning a bounded cache.
	if id != v.selectedID && (v.zoom < 0.45 || v.frameLabelUses >= maxGraphLabelLayouts-1) {
		v.frameHiddenLabels++
		return nil
	}
	v.frameLabelUses++
	key := labelKey{id, font, size, min32(width, 1024)}
	if layout := v.labels.get(key); layout != nil {
		return layout
	}
	layout := ui.ShapeText(value(), ui.Font{Family: font, Size: size}, key.width)
	v.frameLabels++
	v.labels.put(key, layout)
	return layout
}

// Clip a straight connector before stroking it, so a far-offscreen endpoint
// does not feed thousands of invisible pixels to path tessellation/raster work.
func clipSegment(a, b [2]float32, rect ui.Rect) ([2]float32, [2]float32, bool) {
	start, end := float32(0), float32(1)
	dx, dy := b[0]-a[0], b[1]-a[1]
	for _, bound := range [4][2]float32{{-dx, a[0] - rect.X}, {dx, rect.X + rect.W - a[0]}, {-dy, a[1] - rect.Y}, {dy, rect.Y + rect.H - a[1]}} {
		p, q := bound[0], bound[1]
		if p == 0 {
			if q < 0 {
				return a, b, false
			}
			continue
		}
		u := q / p
		if p < 0 {
			start = max32(start, u)
		} else {
			end = min32(end, u)
		}
		if start > end {
			return a, b, false
		}
	}
	return [2]float32{a[0] + dx*start, a[1] + dy*start}, [2]float32{a[0] + dx*end, a[1] + dy*end}, true
}

func (v *view) ensureNodeIndex(page *drawio.Page) {
	if v.indexPage == page {
		return
	}
	v.indexPage = page
	v.nodeIndex = make(map[string]int, len(page.Model.Nodes))
	for i, node := range page.Model.Nodes {
		v.nodeIndex[node.ID] = i
	}
}

func (v *view) diagramNodeBox(node drawio.Node, canvas ui.Rect) ui.Rect {
	x, y := v.displayNodePosition(node)
	sx, sy := v.worldToScreen(float32(x), float32(y))
	return ui.Rect{X: canvas.X + sx, Y: canvas.Y + sy, W: max32(float32(node.Width)*v.zoom, 8), H: max32(float32(node.Height)*v.zoom, 8)}
}
