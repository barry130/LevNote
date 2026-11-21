package graphview

import (
	"crypto/sha256"
	"errors"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
)

func TestDrawioMultiCloneClampCompletesSelectionHistoryAndFeedback(t *testing.T) {
	fixture := `<mxfile><diagram id="p1" name="clamp"><mxGraphModel><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell value="🙂" id="a" vertex="1" parent="1"><mxGeometry x="20" y="20" width="80" height="40" as="geometry"/></mxCell><mxCell id="b" value="Beta" vertex="1" parent="1"><mxGeometry x="160" y="20" width="80" height="40" as="geometry"/></mxCell><mxCell id="edge" value="🙂" edge="1" parent="1" source="a" target="b"><mxGeometry relative="1" as="geometry"/></mxCell></root></mxGraphModel></diagram></mxfile>`
	// Clone insertion starts with <mxCell value="🙂. A previously valid
	// numerical source caret in the closing tags now splits that surrogate.
	caretPosition := text.UTF16Len(fixture[:strings.Index(fixture, "</root>")]) + 16
	caret := text.Range{Anchor: caretPosition, Head: caretPosition}
	v, tester, buffer, changed := newMultiDrawio(t, fixture, 1180, 780)
	selectMultiNodes(t, tester, v, "a", "b")
	if err := buffer.SetSelection(caret.Anchor, caret.Head); err != nil { t.Fatal(err) }
	before, version := buffer.String(), buffer.Version()
	tester.Key(ui.Cmd, ui.KeyD)
	after := buffer.String()
	clones := slices.Clone(v.drawSelection.ids)
	clamped := text.Range{Anchor: caretPosition-1, Head: caretPosition-1}
	if after == before || *changed != 1 || len(v.drawSelection.ids) != 2 { t.Fatal("clone refused before commit", v.Warnings()) }
	if *changed != 1 || buffer.Version() != version+1 || buffer.Selection() != clamped || len(clones) != 2 || v.selectedID != clones[1] { t.Fatal("real clone does not clamp and select result", *changed, buffer.Selection(), clones) }
	if len(v.drawCloneHistory) != 1 { t.Fatal("actual commit must record history", len(v.drawCloneHistory)) }
	if !slices.Contains(v.Warnings(), "已克隆 2 个图形及内部连线到当前页；可撤销。源码光标已移到前一个有效的 UTF-16 边界。") { t.Fatal("clamping disclosure missing", v.Warnings()) }
	if _, err := text.New(after).SliceUTF16(caretPosition, caretPosition); !errors.Is(err, text.ErrSurrogate) { t.Fatal("source caret not mapped to surrogate") }
	tester.Key(ui.Cmd, ui.KeyZ)
	if buffer.String() != before || buffer.Selection() != caret || !slices.Equal(v.drawSelection.ids, []string{"a", "b"}) { t.Fatal("undo did not restore source and its selection", v.drawSelection.ids) }
	tester.Key(ui.Cmd, ui.KeyY)
	if buffer.String() != after || buffer.Selection() != clamped || !slices.Equal(v.drawSelection.ids, clones) { t.Fatal("redo did not restore clone and clamped caret", v.drawSelection.ids) }
}

func TestDrawioCloneHistoryEntryAndByteBudgets(t *testing.T) {
	// Two digests, two string headers and the one-byte after ID. Old tests
	// filled the budget with XML; the bounded record now owns only digests and IDs.
	const base = 2*sha256.Size + 4*(strconv.IntSize/8) + 1
	const weight = drawioCloneHistoryMaxBytes / drawioCloneHistoryMaxEntries
	makeRecord := func(size int, id string) drawioCloneHistory {
		return drawioCloneHistory{
			beforeText: drawioCloneTextIdentity([]byte("original")), afterText: drawioCloneTextIdentity([]byte("result")),
			beforeIDs: []string{strings.Repeat(id, size-base)}, afterIDs: []string{"n"},
		}
	}
	history := make([]drawioCloneHistory, drawioCloneHistoryMaxEntries, drawioCloneHistoryMaxEntries+1)
	for i := range history {
		history[i] = makeRecord(weight, "a")
	}
	if drawioCloneHistoryBytes(history) != drawioCloneHistoryMaxBytes {
		t.Fatal("byte-boundary fixture invalid")
	}
	partial, recorded := appendDrawioCloneHistory(history[:len(history)-1], makeRecord(weight, "b"))
	if !recorded || len(partial) != drawioCloneHistoryMaxEntries || drawioCloneHistoryBytes(partial) != drawioCloneHistoryMaxBytes {
		t.Fatal("exact 256/32MiB combination must fit")
	}
	slots := history
	newest := makeRecord(base+len("newest"), "n")
	newest.beforeIDs[0] = "newest"
	history, recorded = appendDrawioCloneHistory(history, newest)
	if !recorded || len(history) != drawioCloneHistoryMaxEntries ||
		drawioCloneHistoryBytes(history) != drawioCloneHistoryMaxBytes-weight+base+len("newest") || history[0].beforeIDs == nil ||
		!slices.Equal(history[len(history)-1].beforeIDs, []string{"newest"}) || slots[0].beforeText != "" || slots[0].afterText != "" ||
		slots[0].beforeIDs != nil || slots[0].afterIDs != nil {
		t.Fatal("entry overflow did not clean physical oldest slot and retain newest", len(history))
	}

	// Test below the entry cap, where incoming would exceed the byte cap by
	// exactly one byte. Only the oldest record should be evicted.
	history = make([]drawioCloneHistory, drawioCloneHistoryMaxEntries-1)
	for i := range history {
		history[i] = makeRecord(weight, "x")
	}
	history[len(history)-1] = makeRecord(2*weight-base, "y")
	slots = history
	history, recorded = appendDrawioCloneHistory(history, makeRecord(base+1, "a"))
	if !recorded || len(history) != drawioCloneHistoryMaxEntries-1 ||
		drawioCloneHistoryBytes(history) != drawioCloneHistoryMaxBytes-weight+1 || history[0].beforeIDs[0] != strings.Repeat("x", weight-base) ||
		history[len(history)-1].beforeIDs[0] != "a" || slots[0].beforeIDs != nil || slots[0].afterIDs != nil {
		t.Fatal("32 MiB plus one byte did not evict exactly oldest")
	}

	history = []drawioCloneHistory{makeRecord(drawioCloneHistoryMaxBytes, "x")}
	history, recorded = appendDrawioCloneHistory(history, makeRecord(base+1, "a"))
	if !recorded || len(history) != 1 || drawioCloneHistoryBytes(history) != base+1 || history[0].beforeIDs[0] != "a" {
		t.Fatal("full-budget history must free all older records")
	}
	if got, recorded := appendDrawioCloneHistory(nil, makeRecord(drawioCloneHistoryMaxBytes+1, "x")); recorded || len(got) != 0 {
		t.Fatal("oversized single record retained")
	}
}

func TestDrawioCloneHistoryWitnessOwnerAndReleaseReferences(t *testing.T) {
	v, _, buffer, _ := newMultiDrawio(t, multiDrawio, 1180, 780)
	v.setDrawioSelection([]string{"a", "b"}, "b")
	original := drawioCloneTextIdentity([]byte("original"))
	intermediate := drawioCloneTextIdentity([]byte("intermediate"))
	target := drawioCloneHistory{buffer: buffer, release: v.releaseEpoch, pageEpoch: v.pageEpoch,
		beforeText: original, afterText: intermediate, beforeIDs: []string{"a", "b"}, afterIDs: []string{"a", "b"}}
	later := drawioCloneHistory{buffer: buffer, release: v.releaseEpoch, pageEpoch: v.pageEpoch,
		beforeText: intermediate, afterText: drawioCloneTextIdentity([]byte("next")), beforeIDs: []string{"a", "b"}, undoEpoch: v.drawSelection.epoch}
	v.drawCloneHistory = []drawioCloneHistory{target, later}
	if !v.knownDrawioCloneBoundary(buffer, v.releaseEpoch, v.pageEpoch, 0, intermediate) {
		t.Fatal("adjacent same-owner witness rejected")
	}
	for _, mutate := range []func(*drawioCloneHistory){
		func(r *drawioCloneHistory) { r.release++ },
		func(r *drawioCloneHistory) { r.pageEpoch++ },
		func(r *drawioCloneHistory) { r.page++ },
		func(r *drawioCloneHistory) { r.buffer = text.New("intermediate") },
		func(r *drawioCloneHistory) { r.undoEpoch++ },
		func(r *drawioCloneHistory) { r.beforeIDs = []string{"other"} },
	} {
		saved := v.drawCloneHistory[1]
		mutate(&v.drawCloneHistory[1])
		if v.knownDrawioCloneBoundary(buffer, v.releaseEpoch, v.pageEpoch, 0, intermediate) {
			t.Fatal("later witness accepted different owner or unknown epoch")
		}
		v.drawCloneHistory[1] = saved
	}
	if v.knownDrawioCloneBoundary(buffer, v.releaseEpoch, v.pageEpoch, 0, original) {
		t.Fatal("lookup accepted original text as later explicit-selection witness")
	}
	slots := v.drawCloneHistory
	v.Release()
	if len(v.drawCloneHistory) != 0 {
		t.Fatal("history remains after release")
	}
	for i, slot := range slots {
		if slot.buffer != nil || slot.beforeText != "" || slot.afterText != "" || slot.beforeIDs != nil || slot.afterIDs != nil {
			t.Fatal("Release retained view slot", i)
		}
	}
	if v.beginDrawioCloneHistory(buffer, target.release, target.pageEpoch, 0, true) != nil {
		t.Fatal("history accepted after release")
	}
}
