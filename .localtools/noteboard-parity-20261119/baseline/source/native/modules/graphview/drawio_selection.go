package graphview

import (
	"slices"

	"levnote.local/native/core/text"
)

// Selection is view state. IDs retain selection order; membership never includes
// edges. selectedID mirrors primary for the existing single-cell inspector.
type drawioSelection struct {
	ids              []string
	members          map[string]bool
	primary          string
	epoch            uint64
	initialized      bool
	buffer           *text.Buffer
	page, pageCount  int
	pageEpoch        uint64
	pageID, pageName string
}

func (v *view) drawioMultiSelected() bool {
	return v.kind == kindDrawio && len(v.drawSelection.ids) > 1
}

func (v *view) drawioSelected(id string) bool {
	return v.drawSelection.members[id] || len(v.drawSelection.ids) == 0 && id == v.selectedID
}

// All draw.io selection publication passes here, including sync and deletion.
// Callers settle a retained field before changing its owner; sync never flushes.
func (v *view) setDrawioSelection(ids []string, primary string) {
	s := &v.drawSelection
	members := make(map[string]bool, len(ids))
	unique := make([]string, 0, len(ids))
	for _, id := range ids {
		if id != "" && !members[id] {
			members[id] = true
			unique = append(unique, id)
		}
	}
	if len(unique) > 0 && !members[primary] {
		primary = unique[len(unique)-1]
	}
	if !slices.Equal(s.ids, unique) || s.primary != primary || v.selectedID != primary {
		s.epoch++
		v.alignPress = drawioAlignmentPress{}
		v.alignInfo = drawioAlignmentInfo{}
		v.invalidateTagActions()
	}
	s.ids, s.members, s.primary = unique, members, primary
	v.selectedID = primary
	s.initialized, s.buffer, s.page, s.pageEpoch = true, v.host.Buffer, v.pageIndex, v.pageEpoch
	s.pageID, s.pageName, s.pageCount = "", "", 0
	if page := v.currentPage(); page != nil {
		s.pageID, s.pageName, s.pageCount = page.ID, page.Name, len(v.diagram.Pages)
	}
	if len(unique) > 1 {
		v.connectSource = ""
	}
}

func (v *view) setDrawioSingle(id string) {
	var ids []string
	if page := v.currentPage(); page != nil {
		for _, node := range page.Model.Nodes {
			if node.ID == id {
				ids = []string{id}
				break
			}
		}
	}
	v.setDrawioSelection(ids, id)
}

func (v *view) resetDrawioSelection() {
	if v.kind != kindDrawio {
		return
	}
	v.setDrawioSelection(nil, "")
	v.drawSelection.epoch++
	v.alignPress, v.alignInfo = drawioAlignmentPress{}, drawioAlignmentInfo{}
}

func (v *view) syncDrawioSelection() {
	page := v.currentPage()
	if page == nil {
		return
	}
	s := v.drawSelection
	first := !s.initialized
	samePage := s.buffer == v.host.Buffer && s.page == v.pageIndex && s.pageEpoch == v.pageEpoch &&
		s.pageID == page.ID && s.pageName == page.Name && s.pageCount == len(v.diagram.Pages)
	if first {
		id := v.selectedID
		if !v.pageHasCell(id) && len(page.Model.Nodes) > 0 {
			id = page.Model.Nodes[0].ID
		}
		v.setDrawioSingle(id)
		return
	}
	if !samePage {
		if s.pageID == page.ID && s.pageName == page.Name && s.pageCount == len(v.diagram.Pages) {
			// 同一页的缓冲区替换或往返回到本页：多选状态不跨缓冲保留。
			v.resetDrawioSelection()
			return
		}
		// 真实换页：与初次加载一致，选中新页第一个图形。
		id := ""
		if len(page.Model.Nodes) > 0 {
			id = page.Model.Nodes[0].ID
		}
		v.setDrawioSingle(id)
		return
	}
	present := make(map[string]bool, len(page.Model.Nodes))
	for _, node := range page.Model.Nodes {
		present[node.ID] = true
	}
	ids := make([]string, 0, len(s.ids))
	for _, id := range s.ids {
		if present[id] {
			ids = append(ids, id)
		}
	}
	if len(s.ids) == 0 && present[s.primary] {
		ids = append(ids, s.primary)
	}
	primary := s.primary
	if len(ids) == 0 && !v.pageHasCell(primary) {
		primary = ""
	}
	v.setDrawioSelection(ids, primary)
}

func (v *view) toggleDrawioSelection(id string) {
	ids := slices.Clone(v.drawSelection.ids)
	primary := v.selectedID
	if v.drawSelection.members[id] {
		ids = slices.DeleteFunc(ids, func(item string) bool { return item == id })
		if primary == id {
			primary = ""
			if len(ids) > 0 {
				primary = ids[len(ids)-1]
			}
		}
	} else {
		ids = append(ids, id)
		primary = id
	}
	v.setDrawioSelection(ids, primary)
}
