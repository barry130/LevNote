package graphview

import (
	"crypto/sha256"
	"slices"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/drawio"
)

// This is the expected source insertion for a, b and their internal edge in
// multiDrawio, with the 24px offset. It is independent of the clone planner.
const drawioABCloneXML = `<mxCell id="levnote-1" value="开始" vertex="1" parent="1" style="rounded=1;fillColor=#dae8fc;"><mxGeometry x="44.5" y="54.25" width="80.0" height="40.00" as="geometry"/></mxCell><mxCell id="levnote-2" value="处理" vertex="1" parent="1" style="fillColor=#d5e8d4;"><mxGeometry x="184.25" y="124.5" width="120.00" height="60" as="geometry"/></mxCell><mxCell id="levnote-3" edge="1" parent="1" source="levnote-1" target="levnote-2"><mxGeometry relative="1" as="geometry"><Array as="points"><mxPoint x="135.5" y="80.25"/></Array></mxGeometry></mxCell>`

func TestDrawioCloneHistoryLargeDefaultBudgetExactReplay(t *testing.T) {
	prefix, suffix, ok := strings.Cut(multiDrawio, "</root>")
	if !ok {
		t.Fatal("missing fixture insertion anchor")
	}
	filler := "<!--retain this spelling\r\n" + strings.Repeat("x", 17<<20) + "-->\r\n"
	before := prefix + filler + "</root>" + suffix
	after := prefix + filler + drawioABCloneXML + "</root>" + suffix
	if len(before) <= 16<<20 || len(before)+len(after) <= 32<<20 || len(after) > drawio.DefaultMaxBytes {
		t.Fatalf("fixture must exercise default history overflow but fit the format: before=%d after=%d", len(before), len(after))
	}
	v, tester, buffer, changed := newMultiDrawio(t, before, 1180, 780)
	t.Cleanup(v.Release)
	selectMultiNodes(t, tester, v, "a", "b")
	// Make the first ordered ID primary through the canvas.
	clickMultiNode(t, tester, v, "a", 0)
	if !slices.Equal(v.drawSelection.ids, []string{"a", "b"}) || v.selectedID != "a" {
		t.Fatal("fixture must begin with a primary different from the last ID")
	}
	sourceSelection := text.Range{
		Anchor: len(utf16.Encode([]rune(prefix))) + len(filler) + 20,
		Head:   len(utf16.Encode([]rune(`<mxfile custom="keep-🙂`))),
	}
	if err := buffer.SetSelection(sourceSelection.Anchor, sourceSelection.Head); err != nil {
		t.Fatal(err)
	}
	version := buffer.Version()
	originalIDs, clonedIDs := []string{"a", "b"}, []string{"levnote-1", "levnote-2"}
	check := func(phase, wantText, primary string, wantIDs []string, step int) {
		t.Helper()
		exactText := buffer.String() == wantText
		if !exactText || buffer.Selection() != sourceSelection || buffer.InGroup() ||
			buffer.Version() != version+uint64(step) || *changed != step || !slices.Equal(v.drawSelection.ids, wantIDs) ||
			v.selectedID != primary {
			t.Errorf("%s: exact text=%t selection=%+v want=%+v version=%d changed=%d group=%t ids=%v primary=%q",
				phase, exactText, buffer.Selection(), sourceSelection, buffer.Version(), *changed,
				buffer.InGroup(), v.drawSelection.ids, v.selectedID)
		}
	}
	tester.Key(ui.Cmd, ui.KeyD)
	check("clone", after, "levnote-2", clonedIDs, 1)
	if history := buffer.History(); history != (text.HistoryStatus{UndoDepth: 1, RetainedBytes: len(drawioABCloneXML)}) {
		t.Errorf("clone should retain only its insertion in the default history: %+v", history)
	}
	if len(v.drawCloneHistory) != 1 {
		t.Errorf("large clone selection history entries=%d want=1", len(v.drawCloneHistory))
	} else {
		record := v.drawCloneHistory[0]
		beforeSHA, afterSHA := sha256.Sum256([]byte(before)), sha256.Sum256([]byte(after))
		if string(record.beforeText) != string(beforeSHA[:]) || string(record.afterText) != string(afterSHA[:]) ||
			!slices.Equal(record.beforeIDs, originalIDs) || !slices.Equal(record.afterIDs, clonedIDs) {
			t.Error("selection record must preserve both exact source digests and ID lists")
		}
	}
	tester.Key(ui.Cmd, ui.KeyZ)
	check("undo", before, "a", originalIDs, 2)
	if buffer.CanUndo() || !buffer.CanRedo() {
		t.Errorf("clone should undo in one step: %+v", buffer.History())
	}
	tester.Key(ui.Cmd, ui.KeyY)
	check("redo", after, "levnote-2", clonedIDs, 3)
	if !buffer.CanUndo() || buffer.CanRedo() {
		t.Errorf("clone should redo in one step: %+v", buffer.History())
	}
}

func TestDrawioCloneHistoryRejectsSameLengthSourceDrift(t *testing.T) {
	before, after := multiDrawio, strings.Replace(multiDrawio, "</root>", drawioABCloneXML+"</root>", 1)
	for _, tc := range []struct {
		name, prior, next string
	}{
		{"before-replay", strings.Replace(before, "z='007'", "z='008'", 1), after},
		{"after-replay", before, strings.Replace(after, "z='007'", "z='008'", 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, tester, buffer, changed := newMultiDrawio(t, before, 1180, 780)
			t.Cleanup(v.Release)
			selectMultiNodes(t, tester, v, "a", "b")
			tester.Key(ui.Cmd, ui.KeyD)
			tester.Key(ui.Cmd, ui.KeyZ)
			ids := []string{"a", "b"}
			if !slices.Equal(v.drawSelection.ids, ids) || buffer.String() != before || *changed != 2 {
				t.Fatal("fixture requires one undo with a witnessed selection epoch")
			}
			epoch := v.drawSelection.epoch
			// Source editing can build a different redo with one shared XML
			// boundary and identical IDs. The other boundary differs by one byte.
			if tc.prior != before {
				if err := buffer.ReplaceUTF16(0, buffer.LenUTF16(), tc.prior); err != nil {
					t.Fatal(err)
				}
			}
			if err := buffer.ReplaceUTF16(0, buffer.LenUTF16(), tc.next); err != nil {
				t.Fatal(err)
			}
			if len(tc.prior) != len(before) || len(tc.next) != len(after) || !buffer.Undo() || buffer.String() != tc.prior {
				t.Fatal("fixture did not arm the different same-length redo")
			}
			tester.Frame()
			tester.Key(ui.Cmd, ui.KeyY)
			if buffer.String() != tc.next || *changed != 3 || buffer.CanRedo() || !slices.Equal(v.drawSelection.ids, ids) ||
				v.selectedID != "b" || v.drawSelection.epoch != epoch {
				t.Fatalf("different XML must replay as text without reviving clone selection: changed=%d ids=%v epoch=%d want=%d",
					*changed, v.drawSelection.ids, v.drawSelection.epoch, epoch)
			}
		})
	}
}

func TestDrawioCloneHistorySeparatesSuccessfulFieldFlush(t *testing.T) {
	v, buffer, changed, label := prepareDrawioCloneBudgetCase(t, multiDrawio, 4)
	t.Cleanup(v.Release)
	original := text.Range{Anchor: 25, Head: 19}
	if err := buffer.SetSelection(original.Anchor, original.Head); err != nil {
		t.Fatal(err)
	}
	fieldSource := strings.Replace(multiDrawio, `value="开始"`, `value="`+label+`"`, 1)
	copiedXML := strings.Replace(drawioABCloneXML, `value="开始"`, `value="`+label+`"`, 1)
	cloned := strings.Replace(fieldSource, "</root>", copiedXML+"</root>", 1)
	end := len(utf16.Encode([]rune(fieldSource)))
	fieldCaret := text.Range{Anchor: end, Head: end}
	beforeIDs, afterIDs := []string{"a", "b"}, []string{"levnote-1", "levnote-2"}
	check := func(want string, caret text.Range, ids []string, step, undos, redos int) {
		t.Helper()
		history := buffer.History()
		if buffer.String() != want || buffer.Selection() != caret || *changed != step || history.UndoDepth != undos ||
			history.RedoDepth != redos || buffer.InGroup() || !slices.Equal(v.drawSelection.ids, ids) {
			t.Fatalf("field/clone step %d changed=%d caret=%+v history=%+v ids=%v", step, *changed, buffer.Selection(), history, v.drawSelection.ids)
		}
	}
	v.cloneDrawioCells()
	check(cloned, fieldCaret, afterIDs, 2, 2, 0)
	v.undo()
	check(fieldSource, fieldCaret, beforeIDs, 3, 1, 1)
	v.undo()
	check(multiDrawio, original, beforeIDs, 4, 0, 2)
	v.redo()
	check(fieldSource, fieldCaret, beforeIDs, 5, 1, 1)
	v.redo()
	check(cloned, fieldCaret, afterIDs, 6, 2, 0)
}

func TestDrawioCloneHistoryUnavailableDisclosesCommittedClone(t *testing.T) {
	for _, tc := range []struct {
		name          string
		entries, size int
	}{
		{"disabled", 0, 0},
		{"insertion-exceeds-budget", 256, len(drawioABCloneXML) - 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const prefix = "已克隆 2 个图形及内部连线到当前页"
			v, tester, buffer, changed := newMultiDrawio(t, multiDrawio, 1180, 780)
			t.Cleanup(v.Release)
			buffer.SetHistoryLimits(tc.entries, tc.size)
			selectMultiNodes(t, tester, v, "a", "b")
			caret, version := buffer.Selection(), buffer.Version()
			want := strings.Replace(multiDrawio, "</root>", drawioABCloneXML+"</root>", 1)
			tester.Key(ui.Cmd, ui.KeyD)
			if buffer.String() != want || buffer.Selection() != caret || buffer.Version() != version+1 || *changed != 1 ||
				!slices.Equal(v.drawSelection.ids, []string{"levnote-1", "levnote-2"}) || v.selectedID != "levnote-2" {
				t.Fatal("a small compatible clone should still commit and select its copies", v.Warnings())
			}
			if buffer.CanUndo() || buffer.CanRedo() || !buffer.History().Truncated || len(v.drawCloneHistory) != 0 {
				t.Errorf("clone has no replayable text history: %+v visual=%d", buffer.History(), len(v.drawCloneHistory))
			}
			warnings := strings.Join(v.Warnings(), "\n")
			if !strings.Contains(warnings, prefix) || !strings.Contains(warnings, "不可撤销") ||
				!tester.HasText("克隆不可撤销") || strings.Contains(warnings, "；可撤销。") {
				t.Errorf("feedback must disclose committed clone without undo: %q", warnings)
			}
			tester.Key(ui.Cmd, ui.KeyZ)
			if buffer.String() != want || buffer.Selection() != caret || buffer.Version() != version+1 || *changed != 1 ||
				!slices.Equal(v.drawSelection.ids, []string{"levnote-1", "levnote-2"}) {
				t.Fatal("unavailable undo changed the committed source or selection")
			}
		})
	}
}
