package graphview

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

func TestDrawioMultiCloneClampCompletesSelectionHistoryAndFeedback(t *testing.T) {
	fixture := `<mxfile><diagram id="p1" name="clamp"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="a" value="Alpha" vertex="1" parent="1"><mxGeometry x="20" y="20" width="80" height="40" as="geometry"/></mxCell><mxCell id="b" value="Beta" vertex="1" parent="1"><mxGeometry x="160" y="20" width="80" height="40" as="geometry"/></mxCell><mxCell id="edge" value="🙂" edge="1" parent="1" source="a" target="b"><mxGeometry relative="1" as="geometry"/></mxCell></root></mxGraphModel></diagram></mxfile>`
	prefix := `value="🙂`
	caretPosition := text.UTF16Len(fixture[:strings.Index(fixture, prefix)]) + text.UTF16Len(prefix)
	caret := text.Range{Anchor: caretPosition, Head: caretPosition}
	v, tester, buffer, changed := newMultiDrawio(t, fixture, 1180, 780)
	selectMultiNodes(t, tester, v, "a", "b")
	if err := buffer.SetSelection(caret.Anchor, caret.Head); err != nil { t.Fatal(err) }
	before, version := buffer.String(), buffer.Version()
	tester.Key(ui.Cmd, ui.KeyD)
	after, displacement := buffer.String(), text.UTF16Len(buffer.String())-text.UTF16Len(before)
	clones := slices.Clone(v.drawSelection.ids)
	clamped := text.Range{Anchor: caretPosition+displacement-1, Head: caretPosition+displacement-1}
	if after == before || *changed != 1 || len(v.drawSelection.ids) != 2 { t.Fatal("clone refused before commit", v.Warnings()) }
	if *changed != 1 || buffer.Version() != version+1 || buffer.Selection() != clamped || len(clones) != 2 || v.selectedID != clones[1] { t.Fatal("real clone does not clamp and select result", *changed, buffer.Selection(), clones) }
	if len(v.drawCloneHistory) != 1 { t.Fatal("actual commit must record history", len(v.drawCloneHistory)) }
	if !slices.Contains(v.Warnings(), "已克隆 2 个图形及内部连线到当前页；可撤销。源码光标已移到前一个有效的 UTF-16 边界。") { t.Fatal("clamping disclosure missing", v.Warnings()) }
	if _, err := text.New(after).SliceUTF16(caretPosition+displacement, 0); !errors.Is(err, text.ErrSurrogate) { t.Fatal("source caret not mapped to surrogate") }
	tester.Key(ui.Cmd, ui.KeyZ)
	if buffer.String() != before || buffer.Selection() != caret || !slices.Equal(v.drawSelection.ids, []string{"a", "b"}) { t.Fatal("undo did not restore source and its selection", v.drawSelection.ids) }
	tester.Key(ui.Cmd, ui.KeyY)
	if buffer.String() != after || buffer.Selection() != clamped || !slices.Equal(v.drawSelection.ids, clones) { t.Fatal("redo did not restore clone and clamped caret", v.drawSelection.ids) }
}

func TestDrawioCloneHistoryEntryAndByteBudgets(t *testing.T) {
	weight := drawioCloneHistoryMaxBytes/(2*drawioCloneHistoryMaxEntries)
	history := make([]drawioCloneHistory, drawioCloneHistoryMaxEntries, drawioCloneHistoryMaxEntries+1)
	for i := range history { history[i] = drawioCloneHistory{beforeText: strings.Repeat("a", weight), afterText: strings.Repeat("b", weight), beforeIDs: []string{"old"}, afterIDs: []string{"new"}} }
	if drawioCloneHistoryBytes(history) != drawioCloneHistoryMaxBytes { t.Fatal("byte-boundary fixture invalid") }
	partial := appendDrawioCloneHistory(history[:len(history)-1], drawioCloneHistory{beforeText: strings.Repeat("a", weight), afterText: strings.Repeat("b", weight)})
	if len(partial) != drawioCloneHistoryMaxEntries || drawioCloneHistoryBytes(partial) != drawioCloneHistoryMaxBytes { t.Fatal("exact 256/32MiB combination must fit") }
	slots := history
	history = appendDrawioCloneHistory(history, drawioCloneHistory{beforeIDs: []string{"newest"}})
	if len(history) != drawioCloneHistoryMaxEntries || drawioCloneHistoryBytes(history) != drawioCloneHistoryMaxBytes-2*weight || history[0].beforeIDs == nil || !slices.Equal(history[len(history)-1].beforeIDs, []string{"newest"}) || slots[0].beforeText != "" || slots[0].afterText != "" || slots[0].beforeIDs != nil || slots[0].afterIDs != nil { t.Fatal("entry overflow did not clean physical oldest slot and retain newest", len(history)) }
	history = make([]drawioCloneHistory, drawioCloneHistoryMaxEntries)
	for i := range history { history[i] = drawioCloneHistory{beforeText: strings.Repeat("x", weight), afterText: strings.Repeat("y", weight)} }
	history = appendDrawioCloneHistory(history, drawioCloneHistory{beforeText:"a", afterText:"b"})
	if len(history) != drawioCloneHistoryMaxEntries || drawioCloneHistoryBytes(history) != drawioCloneHistoryMaxBytes-2*weight+2 || history[0].beforeText != strings.Repeat("x", weight) || history[len(history)-1].beforeText != "a" { t.Fatal("32 MiB plus one byte did not evict exactly oldest") }
	history = []drawioCloneHistory{{beforeText: strings.Repeat("x", drawioCloneHistoryMaxBytes/2), afterText: strings.Repeat("y", drawioCloneHistoryMaxBytes/2)}}
	history = appendDrawioCloneHistory(history, drawioCloneHistory{beforeText: "a"})
	if len(history) != 1 || drawioCloneHistoryBytes(history) != 1 || history[0].beforeText != "a" { t.Fatal("oversized history must free all older records") }
	if got := appendDrawioCloneHistory(nil, drawioCloneHistory{beforeText: strings.Repeat("x", drawioCloneHistoryMaxBytes+1)}); len(got) != 0 { t.Fatal("oversized single record retained") }
}

func TestDrawioCloneHistoryWitnessOwnerAndReleaseReferences(t *testing.T) {
	v, _, buffer, _ := newMultiDrawio(t, multiDrawio, 1180, 780)
	v.setDrawioSelection([]string{"a", "b"}, "b")
	target := drawioCloneHistory{buffer:buffer, release:v.releaseEpoch, pageEpoch:v.pageEpoch, beforeText:"original", afterText:"intermediate", beforeIDs:[]string{"a", "b"}, afterIDs:[]string{"a", "b"}}
	later := drawioCloneHistory{buffer:buffer, release:v.releaseEpoch, pageEpoch:v.pageEpoch, beforeText:"intermediate", afterText:"next", beforeIDs:[]string{"a", "b"}, undoEpoch:v.drawSelection.epoch}
	v.drawCloneHistory = []drawioCloneHistory{target, later}
	if !v.knownDrawioCloneBoundary(buffer, v.releaseEpoch, v.pageEpoch, 0, "intermediate") { t.Fatal("adjacent same-owner witness rejected") }
	for _, mutate := range []func(*drawioCloneHistory){func(r *drawioCloneHistory){r.release++},func(r *drawioCloneHistory){r.pageEpoch++},func(r *drawioCloneHistory){r.page++},func(r *drawioCloneHistory){r.buffer=text.New("intermediate")},func(r *drawioCloneHistory){r.undoEpoch++},func(r *drawioCloneHistory){r.beforeIDs=[]string{"other"}}} {
		saved := v.drawCloneHistory[1]
		mutate(&v.drawCloneHistory[1])
		if v.knownDrawioCloneBoundary(buffer, v.releaseEpoch, v.pageEpoch, 0, "intermediate") { t.Fatal("later witness accepted different owner or unknown epoch") }
		v.drawCloneHistory[1] = saved
	}
	if v.knownDrawioCloneBoundary(buffer, v.releaseEpoch, v.pageEpoch, 0, "original") { t.Fatal("lookup accepted original text as later explicit-selection witness") }
	slots := v.drawCloneHistory
	v.Release()
	if len(v.drawCloneHistory) != 0 { t.Fatal("history remains after release") }
	for i, slot := range slots { if slot.buffer != nil || slot.beforeText != "" || slot.afterText != "" || slot.beforeIDs != nil || slot.afterIDs != nil { t.Fatal("Release retained view slot", i) } }
	if v.beginDrawioCloneHistory(buffer, target.release, target.pageEpoch, 0, true) != nil { t.Fatal("history accepted after release") }
}
