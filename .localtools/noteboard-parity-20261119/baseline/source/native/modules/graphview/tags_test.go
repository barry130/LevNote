package graphview

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/mindmap"
	"levnote.local/native/ui/editor"
)

const tagInputLabel = "新标签（单行，最多512字节）"

func tagsFixture() string {
	return strings.Replace(markerFixture([]string{"custom", "⭐"}, true),
		`"tags":[{"text":"标签","style":{"color":"red","future":1.20},"className":"dense","riskLevel":2.00}]`,
		`"tags":["重复","",{"text":"重复","style":{"color":"red","future":1.20},"className":"dense","riskLevel":2.00},"重复"]`, 1)
}

func openTags(t *testing.T, tester *ui.Tester) {
	t.Helper()
	revealFieldControl(t, tester, "展开标签面板", 760, 480)
	chooseField(t, tester, "展开标签面板")
}

func clickTag(t *testing.T, tester *ui.Tester, label string) {
	t.Helper()
	revealFieldControl(t, tester, label, 760, 480)
	chooseField(t, tester, label)
}

func typeTag(t *testing.T, tester *ui.Tester, value string) {
	t.Helper()
	clickTag(t, tester, tagInputLabel)
	tester.Command("selectAll")
	tester.Type(value)
}

func tagTexts(t *testing.T, buffer *text.Buffer) []string {
	t.Helper()
	tags, present, err := parseMind(t, buffer).Tags("root")
	if err != nil || !present {
		t.Fatal("tag list absent", err)
	}
	values := make([]string, len(tags))
	for i, tag := range tags {
		values[i] = tag.Text
	}
	return values
}

func expectTagTexts(t *testing.T, buffer *text.Buffer, values ...string) {
	t.Helper()
	if got := tagTexts(t, buffer); !reflect.DeepEqual(got, values) && !(len(got) == 0 && len(values) == 0) {
		t.Fatalf("tags=%#v want %#v", got, values)
	}
}

func stagedTagView(source, value string) *view {
	v := newView(kindMindmap, editor.Host{Buffer: text.New(source)})
	v.ensureSynced(nil)
	v.tagOpen = true
	v.refreshTagDraft()
	_ = v.tagDraft.input.Buffer.ReplaceUTF16(0, 0, value)
	return v
}

func currentTagTarget(v *view, action int) tagGesture {
	v.ensureSynced(nil)
	return v.captureTag(action, v.tagPresentation())
}

func TestTagsNativeOrdinalMetadataAndSeparateUndo(t *testing.T) {
	source := tagsFixture()
	buffer := text.New(source)
	_ = buffer.SetSelection(11, 3)
	selection := buffer.Selection()
	v := newView(kindMindmap, editor.Host{Buffer: buffer})
	tester := ui.NewTester(v.Draw, 760, 480)
	if _, ok := tester.AccessByRole(ui.RoleButton, "移除第1个标签"); ok {
		t.Fatal("collapsed list created controls")
	}
	openTags(t, tester)
	revealFieldControl(t, tester, "第2个标签", 760, 480)
	empty, visible := tester.Access("第2个标签")
	if !visible || empty.Offscreen || !strings.Contains(empty.Description, "（空字符串）") || buffer.String() != source || buffer.CanUndo() {
		t.Fatal("viewing changed bytes or hid empty legacy tag", empty)
	}
	clickTag(t, tester, "移除第4个标签")
	expectTagTexts(t, buffer, "重复", "", "重复")
	if buffer.Selection() != selection || buffer.History().UndoDepth != 1 {
		t.Fatal("ordinal remove selection/history", buffer.Selection(), buffer.History())
	}
	doc := parseMind(t, buffer)
	tags := *doc.Root.Tags
	if !tags[2].Object || !tags[2].ClassNameSet || tags[2].ClassName != "dense" || !tags[2].StyleSet || len(tags[2].Style.Members) != 2 || tags[2].Style.Members[1].Key != "future" || string(tags[2].Style.Members[1].Raw) != "1.20" || string(tags[2].Extra.Members[0].Raw) != "2.00" {
		t.Fatal("object survivor metadata altered", tags[2])
	}
	for _, literal := range []string{`1.00`, `0.30`, `1e3`, `"custom"`, `"child.txt"`} {
		if !strings.Contains(buffer.String(), literal) {
			t.Fatal("unrelated metadata lost", literal)
		}
	}
	if !buffer.Undo() || buffer.String() != source || buffer.Selection() != selection || !buffer.Redo() {
		t.Fatal("exact undo/redo failed")
	}
	tester.Frame()
	typeTag(t, tester, "  新,🙂标签  ")
	beforeAdd := buffer.String()
	clickTag(t, tester, "添加标签")
	expectTagTexts(t, buffer, "重复", "", "重复", "  新,🙂标签  ")
	if v.pendingTagDraft() || buffer.History().UndoDepth != 2 {
		t.Fatal("add did not clear only successful input", v.tagError)
	}
	if !buffer.Undo() || buffer.String() != beforeAdd || !buffer.Redo() {
		t.Fatal("add not one exact undo")
	}
	tester.Frame()
	clickTag(t, tester, "清空全部标签")
	expectTagTexts(t, buffer)
	if parseMind(t, buffer).Root.Tags == nil {
		t.Fatal("clear lost explicit []")
	}
}

func TestTagsNativeKeyboardComposeBarriersAndDraftFlush(t *testing.T) {
	for _, key := range []ui.Key{ui.KeyEnter, ui.KeySpace} {
		t.Run(fmt.Sprint(key), func(t *testing.T) {
			v := newView(kindMindmap, editor.Host{Buffer: text.New(tagsFixture())})
			tester := ui.NewTester(v.Draw, 760, 480)
			// F6 starts from the real node editor, reaches the collapsed section.
			chooseField(t, tester, "节点文本")
			tester.Key(0, ui.KeyF6)
			if !tester.Focused("展开标签面板") {
				t.Fatal("F6 did not reach tag section")
			}
			tester.Key(0, key)
			for i := 0; i < 14 && !tester.Focused(tagInputLabel); i++ {
				tester.Key(0, ui.KeyTab)
			}
			if !tester.Focused(tagInputLabel) {
				t.Fatal("Tab did not reach tag input")
			}
			tester.Compose("组合🙂", 4)
			d := v.tagDraft
			selection, version := d.input.Buffer.Selection(), d.input.Buffer.Version()
			for _, shortcut := range []struct {
				mods ui.Modifiers
				key  ui.Key
			}{{0, ui.KeyF6}, {0, ui.KeyTab}, {ui.Shift, ui.KeyTab}} {
				tester.Key(shortcut.mods, shortcut.key)
				_, composing := d.input.MarkedRange()
				if !tester.Focused(tagInputLabel) || !composing || !d.input.Buffer.InGroup() || d.input.Buffer.Version() != version || d.input.Buffer.Selection() != selection || !tester.HasText("请先确认正在输入") {
					t.Fatalf("composition focus escape changed draft: key=%v focused=%t composing=%t group=%t version=%d/%d selection=%v/%v text=%q", shortcut.key, tester.Focused(tagInputLabel), composing, d.input.Buffer.InGroup(), d.input.Buffer.Version(), version, d.input.Buffer.Selection(), selection, tester.Texts())
				}
			}
			if err := v.Flush(); !errors.Is(err, errTagPending) || v.selectNode("left") || v.selectField(fieldNote) {
				t.Fatal("public barrier bypassed pending composition", err)
			}
			if _, err := v.Export("svg"); !errors.Is(err, errTagPending) {
				t.Fatal("export ignored pending input", err)
			}
			tester.Type("组合🙂") // native confirmation, not an appended duplicate
			tester.Key(0, ui.KeyF6)
			if !tester.Focused("添加标签") {
				t.Fatal("F6 not at Add after confirmation")
			}
			tester.Key(0, key)
			expectTagTexts(t, v.host.Buffer, "重复", "", "重复", "重复", "组合🙂")
			if v.host.Buffer.History().UndoDepth != 1 {
				t.Fatal("keyboard duplicated tag commit", v.host.Buffer.History(), v.tagError)
			}
		})
	}
}

func TestTagsNativeNodeCompositionFocusGuard(t *testing.T) {
	v := newView(kindMindmap, editor.Host{Buffer: text.New(tagsFixture())})
	tester := ui.NewTester(v.Draw, 760, 480)
	focusDraft(t, tester)
	tester.Compose("主题🙂", 4)
	d := v.draft
	selection, version := d.input.Buffer.Selection(), d.input.Buffer.Version()
	for _, shortcut := range []struct {
		mods ui.Modifiers
		key  ui.Key
	}{{0, ui.KeyF6}, {0, ui.KeyTab}, {ui.Shift, ui.KeyTab}} {
		tester.Key(shortcut.mods, shortcut.key)
		_, composing := d.input.MarkedRange()
		if !tester.Focused("节点文本") || !composing || !d.input.Buffer.InGroup() || d.input.Buffer.Version() != version || d.input.Buffer.Selection() != selection || v.host.Buffer.CanUndo() {
			t.Fatal("ordinary composition was unmarked by focus escape")
		}
	}
	tester.Type("主题🙂")
	if v.tagError != nil || v.draftError != nil {
		t.Fatal("confirmed ordinary input retained focus warning", v.tagError, v.draftError)
	}
	tester.Key(0, ui.KeyTab)
	if !tester.Focused("展开标记面板") || parseMind(t, v.host.Buffer).Root.Topic != "主题🙂" {
		t.Fatal("confirmed Tab traversal regressed")
	}
}

func TestTagsNativeConfirmedFocusWarningOnly(t *testing.T) {
	var reports []error
	v := newView(kindMindmap, editor.Host{Buffer: text.New(tagsFixture()), Report: func(err error) { reports = append(reports, err) }})
	tester := ui.NewTester(v.Draw, 760, 480)
	openTags(t, tester)
	clickTag(t, tester, tagInputLabel)
	tester.Compose("确认🙂", 4)
	tester.Key(0, ui.KeyF6)
	if v.tagError != errTagConfirm || len(reports) != 0 || !strings.Contains(strings.Join(v.Warnings(), " "), errTagConfirm.Error()) {
		t.Fatal("real composition did not produce the focus warning", v.tagError)
	}
	tester.Type("确认🙂")
	if v.tagError != nil || v.tagDraft.input.Buffer.String() != "确认🙂" || !v.pendingTagDraft() || v.host.Buffer.CanUndo() {
		t.Fatal("confirmation retained warning or changed source/input", v.tagError)
	}
	// A genuine validation error is not focus guidance and must survive a
	// later confirmation, including an empty confirmed input.
	typeTag(t, tester, "坏\n标签")
	clickTag(t, tester, "添加标签")
	if v.tagError != errTagInput || len(reports) != 1 || reports[0] != errTagInput {
		t.Fatal("multiline refusal fixture", v.tagError)
	}
	clickTag(t, tester, tagInputLabel)
	tester.Command("selectAll")
	tester.Compose("更正🙂", 4)
	tester.Type("更正🙂")
	if v.tagError != errTagInput || v.tagDraft.input.Buffer.String() != "更正🙂" {
		t.Fatal("confirmation erased unrelated refusal", v.tagError)
	}
	tester.Command("selectAll")
	tester.Type("")
	if v.tagError != errTagInput || v.host.Buffer.CanUndo() {
		t.Fatal("empty confirmation erased unrelated refusal", v.tagError)
	}
	// Confirming the ordinary field while a tag remains pending replaces only
	// the guidance with the real public barrier, retaining both exact drafts.
	typeTag(t, tester, "等待标签")
	revealFieldControl(t, tester, "节点文本", 760, 480)
	chooseField(t, tester, "节点文本")
	tester.Command("selectAll")
	tester.Compose("等待字段", 4)
	tester.Key(0, ui.KeyTab)
	if v.tagError != errTagConfirm || v.draftError != errTagConfirm {
		t.Fatal("ordinary composition did not share the warning")
	}
	for _, err := range reports {
		if err == errTagConfirm {
			t.Fatal("transient focus warning was retained in host failure history")
		}
	}
	tester.Type("等待字段")
	if v.tagError != errTagPending || v.draftError != errTagPending || v.draft.input.Buffer.String() != "等待字段" || v.tagDraft.input.Buffer.String() != "等待标签" || v.host.Buffer.CanUndo() {
		t.Fatal("ordinary confirmation erased pending barrier/drafts", v.tagError, v.draftError)
	}
	typeTag(t, tester, "")
	if v.tagError != nil || v.draftError != nil || v.pendingTagDraft() || !v.pendingDraft() || v.host.Buffer.CanUndo() {
		t.Fatal("cleared input retained resolved pending warning or lost field", v.tagError, v.draftError)
	}
	typeTag(t, tester, "放弃标签")
	if err := v.Flush(); err != errTagPending {
		t.Fatal("pending discard fixture", err)
	}
	clickTag(t, tester, "放弃标签输入")
	if v.tagError != nil || v.draftError != nil || v.pendingTagDraft() || !v.pendingDraft() || v.host.Buffer.CanUndo() {
		t.Fatal("explicit discard retained resolved pending warning or lost field", v.tagError, v.draftError)
	}
}

func TestTagsNativeDualPendingAddAndReentrantBarrier(t *testing.T) {
	v := newView(kindMindmap, editor.Host{Buffer: text.New(tagsFixture())})
	tester := ui.NewTester(v.Draw, 760, 480)
	openTags(t, tester)
	typeTag(t, tester, "待添加🙂")
	revealFieldControl(t, tester, "节点文本", 760, 480)
	chooseField(t, tester, "节点文本")
	tester.Command("selectAll")
	tester.Type("本地字段")
	if !v.pendingDraft() || !v.pendingTagDraft() || v.host.Buffer.CanUndo() {
		t.Fatal("pending tag failed to retain ordinary OnChange draft")
	}
	field := v.draft
	changes := 0
	v.host.Changed = func() {
		changes++
		if err := v.Flush(); !errors.Is(err, errTagPending) {
			t.Error("nested Flush bypassed pending tag", err)
		}
		if v.selectField(fieldNote) || v.selectNode("left") {
			t.Error("nested host action switched")
		}
	}
	clickTag(t, tester, "添加标签")
	expectTagTexts(t, v.host.Buffer, "重复", "", "重复", "重复", "待添加🙂")
	if parseMind(t, v.host.Buffer).Root.Topic != "本地字段" || changes != 2 || v.host.Buffer.History().UndoDepth != 2 || v.draft != field || v.pendingGraphDraft() {
		t.Fatal("own field receipt rejected valid write or merged transactions", changes, v.draftError, v.tagError)
	}
	if !v.host.Buffer.Undo() {
		t.Fatal("missing tag undo")
	}
	expectTagTexts(t, v.host.Buffer, "重复", "", "重复", "重复")
	if parseMind(t, v.host.Buffer).Root.Topic != "本地字段" || !v.host.Buffer.Undo() || v.host.Buffer.String() != tagsFixture() {
		t.Fatal("field/tag undo not independent")
	}
}

func TestTagsNativePointerCompositionRepeatCancelAndScroll(t *testing.T) {
	v := newView(kindMindmap, editor.Host{Buffer: text.New(tagsFixture())})
	tester := ui.NewTester(v.Draw, 760, 480)
	openTags(t, tester)
	clickTag(t, tester, tagInputLabel)
	tester.Compose("指针🙂", 4)
	clickTag(t, tester, "添加标签")
	expectTagTexts(t, v.host.Buffer, "重复", "", "重复", "重复", "指针🙂")
	for _, mode := range []string{"cancel", "scroll", "outside", "collapse", "release"} {
		t.Run(mode, func(t *testing.T) {
			revealFieldControl(t, tester, "移除第1个标签", 760, 480)
			r, _ := tester.Find("移除第1个标签")
			x, y := r.X+r.W/2, r.Y+r.H/2
			tester.Advance(time.Second)
			tester.Press(x, y)
			if !v.tagPress.active {
				t.Fatal("native pointer did not capture target")
			}
			before, history := v.host.Buffer.String(), v.host.Buffer.History()
			switch mode {
			case "cancel":
				tester.CancelPress()
			case "scroll":
				tester.Wheel(750, 445, 0, 80)
				tester.Advance(time.Second)
				moved, _ := tester.Find("移除第1个标签")
				if moved.Y == r.Y {
					t.Fatal("scroll handler swallowed real inspector movement")
				}
				tester.Release(x, y)
			case "outside":
				tester.Release(10, 10)
			case "collapse":
				v.tagOpen = false
				v.invalidateTagActions()
				tester.Frame()
				tester.Release(x, y)
				v.tagOpen = true
				tester.Frame()
			case "release":
				v.Release()
				tester.Frame()
				tester.Release(x, y)
			}
			if v.host.Buffer.String() != before || v.host.Buffer.History() != history || v.tagPress.active {
				t.Fatal("canceled pointer applied tag", v.tagError)
			}
		})
	}
	revealFieldControl(t, tester, "移除第1个标签", 760, 480)
	// Use a real key-down stream: repeated key down must not be a second edit.
	typeTag(t, tester, "一次")
	tester.Key(0, ui.KeyF6)
	tester.KeyDown(ui.KeyEnter)
	for i := 0; i < 5; i++ {
		tester.KeyRepeat(0, ui.KeyEnter)
	}
	tester.KeyUp(ui.KeyEnter)
	if got := tagTexts(t, v.host.Buffer); len(got) != 6 || got[5] != "一次" {
		t.Fatal("held key repeated add", got, v.tagError)
	}
	tester.Frame()
	tester.Frame()
	tester.Advance(2 * time.Second)
	tester.Frame()
	if v.tagPress.active || tester.FramePending() {
		t.Fatal("settled button retained capture or invalidation loop")
	}
}

func TestTagsReceiptRejectsChangedCallbackMutations(t *testing.T) {
	for _, mode := range []string{"release", "buffer", "source", "source-selection", "field-input", "field-selection", "field-replace", "field-owner", "node", "field", "away-back", "collapse"} {
		t.Run(mode, func(t *testing.T) {
			v := stagedTagView(tagsFixture(), "staged")
			buffer, d := v.host.Buffer, v.draft
			_ = d.input.Buffer.ReplaceUTF16(0, d.input.Buffer.LenUTF16(), "written-field")
			g := currentTagTarget(v, tagAdd)
			v.host.Changed = func() {
				switch mode {
				case "release":
					v.Release()
				case "buffer":
					v.host.Buffer = text.New(tagsFixture())
				case "source":
					_ = buffer.ReplaceUTF16(0, 0, " ")
				case "source-selection":
					_ = buffer.SetSelection(1, 1)
				case "field-input":
					_ = d.input.Buffer.ReplaceUTF16(0, 0, "later")
				case "field-selection":
					_ = d.input.Buffer.SetSelection(0, 0)
				case "field-replace":
					v.draft = &fieldDraft{input: editor.New(text.New("replacement"))}
				case "field-owner":
					d.buffer = text.New(tagsFixture())
				case "node":
					v.selectedID = "left"
				case "field":
					v.field = fieldNote
				case "away-back":
					v.selectedID = "left"
					v.invalidateTagActions()
					v.selectedID = "root"
				case "collapse":
					v.tagOpen = false
					v.invalidateTagActions()
				}
			}
			err := v.activateTag(g)
			if err == nil || !strings.Contains(err.Error(), "节点字段已写回，标签未修改") {
				t.Fatal("post-write refusal not truthful", err)
			}
			expectTagTexts(t, buffer, "重复", "", "重复", "重复")
			if d.base != "中心" || !v.pendingTagDraft() || mode == "release" && v.mind != nil || mode == "node" && v.selectedID != "left" {
				t.Fatal("stale callback state restored", d.base, v.selectedID)
			}
		})
	}
}

func TestTagsPostFieldRawGrowthProspectiveBudget(t *testing.T) {
	value := strings.Repeat("x", 3830)
	source := `{"format":"levnote.mindmap","version":1,"root":{"id":"root","topic":"根","tags":[{"text":"` + value + `"}]}}`
	added := strings.Repeat("a", 512)
	v := stagedTagView(source, added)
	before := *v.mind.Root.Tags
	// Independent raw-envelope arithmetic: compact {"text":"..."} is N+11;
	// decoded text contributes another N, plus1tag+1newtag+512newbytes.
	if len(before[0].Raw) != len(value)+11 || 2*len(value)+12+513 > 8192 {
		t.Fatal("prewrite fixture does not fit")
	}
	if err := tagProspectiveBudget(before, &added); err != nil {
		t.Fatal("prewrite prospective refused", err)
	}
	_ = v.draft.input.Buffer.ReplaceUTF16(0, v.draft.input.Buffer.LenUTF16(), "written")
	err := v.activateTag(currentTagTarget(v, tagAdd))
	post := *parseMind(t, v.host.Buffer).Root.Tags
	postBytes := 1 + len(post[0].Raw) + len(post[0].Text)
	if postBytes > 8192 || postBytes+513 <= 8192 {
		t.Fatalf("fixture must overflow ONLY after canonical Raw growth: before=%d after=%d", 2*len(value)+12, postBytes)
	}
	if !errors.Is(err, errTagBudget) || !strings.Contains(err.Error(), "节点字段已写回，标签未修改") || v.host.Buffer.History().UndoDepth != 1 || !v.pendingTagDraft() || len(post) != 1 || post[0].Text != value {
		t.Fatal("post-Flush prospective budget not enforced", err)
	}
}

func TestTagsNativeHeldDraftControlsCannotRetarget(t *testing.T) {
	for _, action := range []string{"放弃标签输入", "恢复标签及节点草稿"} {
		for _, mode := range []string{"replacement", "edit", "selection", "scroll", "release"} {
			t.Run(action+"/"+mode, func(t *testing.T) {
				v := newView(kindMindmap, editor.Host{Buffer: text.New(tagsFixture())})
				recovered := 0
				v.host.RecoverDraft = func(string, string) error { recovered++; return nil }
				tester := ui.NewTester(v.Draw, 760, 480)
				openTags(t, tester)
				typeTag(t, tester, "staged")
				revealFieldControl(t, tester, action, 760, 480)
				r, _ := tester.Find(action)
				x, y := r.X+r.W/2, r.Y+r.H/2
				tester.Advance(time.Second)
				tester.Press(x, y)
				if !v.tagPress.active {
					t.Fatal("draft action did not capture native press")
				}
				switch mode {
				case "replacement":
					v.tagDraft = &tagDraft{buffer: v.host.Buffer, nodeID: "root", input: editor.New(text.New("replacement"))}
				case "edit":
					_ = v.tagDraft.input.Buffer.ReplaceUTF16(0, 0, "new")
				case "selection":
					_ = v.tagDraft.input.Buffer.SetSelection(0, 0)
				case "scroll":
					tester.Wheel(750, 445, 0, 60)
					tester.Advance(time.Second)
				case "release":
					v.Release()
				}
				wanted := v.tagDraft
				tester.Frame()
				tester.Release(x, y)
				if v.tagDraft != wanted || !v.pendingTagDraft() || recovered != 0 || v.host.Buffer.CanUndo() {
					t.Fatal("held action discarded/recovered replacement", v.tagError)
				}
			})
		}
	}
}

func TestTagsNativeRapidContinuationCannotDeleteNextOrdinal(t *testing.T) {
	v := newView(kindMindmap, editor.Host{Buffer: text.New(tagsFixture())})
	tester := ui.NewTester(v.Draw, 760, 480)
	openTags(t, tester)
	revealFieldControl(t, tester, "移除第1个标签", 760, 480)
	r, _ := tester.Find("移除第1个标签")
	x, y := r.X+r.W/2, r.Y+r.H/2
	tester.Advance(time.Second)
	tester.ClickAt(x, y)
	expectTagTexts(t, v.host.Buffer, "", "重复", "重复")
	next, _ := tester.Find("移除第1个标签")
	if x < next.X || x >= next.X+next.W || y < next.Y || y >= next.Y+next.H {
		t.Fatal("continued click must intersect newly reindexed remove control", r, next)
	}
	tester.ClickAt(x, y) // same rapid gesture, no virtual time separation
	expectTagTexts(t, v.host.Buffer, "", "重复", "重复")
	if v.host.Buffer.History().UndoDepth != 1 {
		t.Fatal("continued multi-click deleted next row")
	}
	tester.Advance(time.Second)
	tester.ClickAt(x, y)
	expectTagTexts(t, v.host.Buffer, "重复", "重复")
	if v.host.Buffer.History().UndoDepth != 2 {
		t.Fatal("new independent single click failed")
	}
}

func TestTagsNativeOpenerContinuationAndExplicitDiscard(t *testing.T) {
	v := newView(kindMindmap, editor.Host{Buffer: text.New(tagsFixture())})
	tester := ui.NewTester(v.Draw, 760, 480)
	revealFieldControl(t, tester, "展开标签面板", 760, 480)
	r, _ := tester.Find("展开标签面板")
	x, y := r.X+r.W/2, r.Y+r.H/2
	tester.Advance(time.Second)
	tester.ClickAt(x, y)
	opened, _ := tester.Find("收起标签面板")
	if !v.tagOpen || opened != r {
		t.Fatal("opening shifted the opener's bounds", r, opened)
	}
	tester.ClickAt(x, y)
	if v.tagOpen || v.host.Buffer.String() != tagsFixture() || v.host.Buffer.CanUndo() {
		t.Fatal("rapid opener continuation reached a document action")
	}
	tester.Advance(time.Second)
	tester.ClickAt(x, y)
	typeTag(t, tester, "discard-only")
	revealFieldControl(t, tester, "节点文本", 760, 480)
	chooseField(t, tester, "节点文本")
	tester.Command("selectAll")
	tester.Type("field-kept")
	field := v.draft
	clickTag(t, tester, "放弃标签输入")
	if v.pendingTagDraft() || v.draft != field || !v.pendingDraft() || field.input.Buffer.String() != "field-kept" || v.host.Buffer.CanUndo() {
		t.Fatal("discard touched source or ordinary draft")
	}
	if err := v.Flush(); err != nil || parseMind(t, v.host.Buffer).Root.Topic != "field-kept" || v.host.Buffer.History().UndoDepth != 1 {
		t.Fatal("discard failed to release public barrier", err)
	}
}

func TestTagsNativeIdentityStaleWithoutDrawAndAwayBack(t *testing.T) {
	for _, mode := range []string{"without-draw", "away-back", "field", "collapse"} {
		t.Run(mode, func(t *testing.T) {
			v := newView(kindMindmap, editor.Host{Buffer: text.New(tagsFixture())})
			armed := false
			tester := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().ObserveInput(func(ev ui.InputEvent) {
					if armed && ev.Kind == ui.InputPointerDown {
						armed = false
						replaceSource(t, v.host.Buffer, `"riskLevel":2.00`, `"riskLevel":3.00`)
					}
				}).Children(func() { v.Draw(c) })
			}, 760, 480)
			openTags(t, tester)
			revealFieldControl(t, tester, "移除第1个标签", 760, 480)
			r, _ := tester.Find("移除第1个标签")
			x, y := r.X+r.W/2, r.Y+r.H/2
			tester.Advance(time.Second)
			armed = mode == "without-draw"
			tester.Press(x, y)
			switch mode {
			case "away-back":
				if !v.selectNode("left") || !v.selectNode("root") {
					t.Fatal("fixture selection failed")
				}
			case "field":
				if !v.selectField(fieldNote) {
					t.Fatal("fixture field failed")
				}
			case "collapse":
				v.tagOpen = false
				v.invalidateTagActions()
				v.tagOpen = true
			}
			before, history := v.host.Buffer.String(), v.host.Buffer.History()
			tester.Frame()
			tester.Release(x, y)
			if v.host.Buffer.String() != before || v.host.Buffer.History() != history {
				t.Fatal("stale identity retargeted tag", mode, v.tagError)
			}
			if mode == "without-draw" && !errors.Is(v.tagError, errTagStale) {
				t.Fatal("stale display was silently recaptured", v.tagError)
			}
		})
	}
}

func TestTagsSuccessfulCommitKeepsReplacementInputsAndReleasedCache(t *testing.T) {
	for _, mode := range []string{"release", "tag-replacement", "tag-edit", "field-replacement", "clean-field-replacement", "buffer"} {
		t.Run(mode, func(t *testing.T) {
			v := stagedTagView(tagsFixture(), "applied")
			buffer := v.host.Buffer
			old := v.tagDraft
			v.host.Changed = func() {
				switch mode {
				case "release":
					v.Release()
				case "tag-replacement":
					v.tagDraft = &tagDraft{buffer: buffer, nodeID: "root", input: editor.New(text.New("replacement"))}
				case "tag-edit":
					_ = old.input.Buffer.ReplaceUTF16(0, 0, "later")
				case "field-replacement":
					v.draft = &fieldDraft{input: editor.New(text.New("new-field")), base: "", buffer: buffer}
				case "clean-field-replacement":
					v.draft = &fieldDraft{input: editor.New(text.New("clean-replacement")), base: "clean-replacement", buffer: buffer}
				case "buffer":
					v.host.Buffer = text.New(testMindmap)
				}
			}
			if err := v.activateTag(currentTagTarget(v, tagAdd)); err != nil {
				t.Fatal("successful edit misreported as refusal", err)
			}
			expectTagTexts(t, buffer, "重复", "", "重复", "重复", "applied")
			if mode == "release" && (v.mind != nil || v.tagInfo.loaded) {
				t.Fatal("released cache resurrected")
			}
			if mode == "tag-replacement" && v.tagDraft.input.Buffer.String() != "replacement" || mode == "tag-edit" && v.tagDraft.input.Buffer.String() != "laterapplied" {
				t.Fatal("replacement/later draft cleared")
			}
			if mode == "field-replacement" && v.draft.input.Buffer.String() != "new-field" {
				t.Fatal("field replacement overwritten")
			}
			if mode == "clean-field-replacement" && v.draft.input.Buffer.String() != "clean-replacement" {
				t.Fatal("clean callback replacement overwritten by refresh")
			}
		})
	}
}

func TestTagsRecoveryExactBothLegacyAndReentrantReplacement(t *testing.T) {
	for _, mode := range []string{"field-only", "both", "tag-only", "failure", "replacement"} {
		t.Run(mode, func(t *testing.T) {
			v := stagedTagView(tagsFixture(), "标签🙂\r\n原样")
			v.host.Path = "sample.mindmap"
			if mode == "field-only" {
				v.clearTagDraft()
			}
			if mode != "tag-only" {
				_ = v.draft.input.Buffer.ReplaceUTF16(0, v.draft.input.Buffer.LenUTF16(), "字段🙂\r\n原样")
			}
			oldField, oldTag := v.draft, v.tagDraft
			var title, content string
			v.host.RecoverDraft = func(name, value string) error {
				title, content = name, value
				if mode == "failure" {
					return errors.New("host refused 4MiB limit")
				}
				if mode == "replacement" {
					v.draft = &fieldDraft{input: editor.New(text.New("replacement-field"))}
					v.tagDraft = &tagDraft{input: editor.New(text.New("replacement-tag"))}
				}
				return nil
			}
			err := v.RecoverDraft()
			if title != "sample.mindmap-节点草稿.txt" || v.host.Buffer.String() != tagsFixture() || v.host.Buffer.CanUndo() {
				t.Fatal("recovery altered source/title")
			}
			legacy := "sample.mindmap\n节点 ID: root\n页面 ID: \n字段: 节点文本\n原字段存在: true\n\n字段🙂\r\n原样"
			if mode == "field-only" && content != legacy {
				t.Fatal("legacy envelope changed", content)
			}
			if mode != "field-only" && (!strings.HasSuffix(content, "标签🙂\r\n原样") || !strings.Contains(content, "字段: 新标签（尚未添加）")) {
				t.Fatal("exact tag missing from recovery", content)
			}
			if mode == "both" && !strings.HasPrefix(content, legacy+"\n\n--- 未添加的标签输入 ---\n") {
				t.Fatal("combined field envelope lost")
			}
			if mode == "failure" {
				if err == nil || v.draft != oldField || v.tagDraft != oldTag || !v.pendingGraphDraft() {
					t.Fatal("failed recovery dropped drafts")
				}
			} else if err != nil {
				t.Fatal(err)
			}
			if mode == "replacement" && (v.draft.input.Buffer.String() != "replacement-field" || v.tagDraft.input.Buffer.String() != "replacement-tag") {
				t.Fatal("callback replacement cleared")
			}
		})
	}
}

func TestTagsBudgetInputPresenceAndIdle(t *testing.T) {
	for _, values := range [][]mindmap.Tag{
		make([]mindmap.Tag, 65),
		{{Text: strings.Repeat("x", 4096), Raw: []byte(strings.Repeat("r", 4096))}},
		{{Text: "x", ClassName: strings.Repeat("c", 8191)}},
		{{Object: true, Style: mindmap.Object{Members: make([]mindmap.Member, 8192)}}},
		{{Object: true, Extra: mindmap.Object{Members: []mindmap.Member{{Key: strings.Repeat("k", 8192)}}}}},
	} {
		if !errors.Is(tagBudget(values), errTagBudget) {
			t.Fatal("budget failed Raw/decoded/member accounting")
		}
	}
	if err := tagBudget([]mindmap.Tag{{Text: strings.Repeat("x", 8191)}}); err != nil {
		t.Fatal("exact budget", err)
	}
	for _, value := range []string{strings.Repeat("中", 171), "a\nb", "a\rb", strings.Repeat("x", 1024*1024)} {
		v := stagedTagView(tagsFixture(), value)
		if err := v.activateTag(currentTagTarget(v, tagAdd)); !errors.Is(err, errTagInput) || v.host.Buffer.CanUndo() || v.tagDraft.input.Buffer.String() != value {
			t.Fatal("rejected paste truncated/committed", len(value), err)
		}
	}
	v := stagedTagView(tagsFixture(), strings.Repeat("x", 512))
	if err := v.activateTag(currentTagTarget(v, tagAdd)); err != nil {
		t.Fatal("exact512byte input refused", err)
	}
	for _, tagPart := range []string{"", `,"tags":[]`} {
		source := `{"format":"levnote.mindmap","version":1,"root":{"id":"root","topic":"根"` + tagPart + `}}`
		v := stagedTagView(source, "")
		if err := v.activateTag(currentTagTarget(v, tagClear)); err != nil || v.host.Buffer.String() != source || v.host.Buffer.CanUndo() {
			t.Fatal("no-op clear changed absent/empty", err)
		}
	}
	v = stagedTagView(tagsFixture(), "")
	v.host.ReadOnly = true
	if err := v.activateTag(currentTagTarget(v, 0)); err == nil || v.host.Buffer.CanUndo() {
		t.Fatal("readonly mutation")
	}
	v.host.ReadOnly = false
	_ = v.host.Buffer.BeginGroup()
	if err := v.activateTag(currentTagTarget(v, 0)); !errors.Is(err, text.ErrGroup) || !v.host.Buffer.InGroup() {
		t.Fatal("foreign group stolen", err)
	}
	v.host.Buffer.EndGroup()
	tester := ui.NewTester(v.Draw, 760, 480)
	doc, draft, tagDraft, summary := v.mind, v.draft, v.tagDraft, v.tagInfo.summary
	for i := 0; i < 5; i++ {
		tester.Frame()
	}
	if v.mind != doc || v.draft != draft || v.tagDraft != tagDraft || v.tagInfo.summary != summary || len(summary) > 512 || !utf8.ValidString(summary) || tester.FramePending() {
		t.Fatal("idle rebuilt source/drafts or self-invalidated")
	}
}

func TestTagsSharedDrawioReceiptAndFieldOnlyRecovery(t *testing.T) {
	v := newView(kindDrawio, editor.Host{Buffer: text.New(testDrawio), Path: "sample.drawio"})
	tester := ui.NewTester(v.Draw, 760, 480)
	if _, ok := tester.AccessByRole(ui.RoleButton, "展开标签面板"); ok {
		t.Fatal("tag UI leaked into drawio")
	}
	focusDraft(t, tester)
	tester.Compose("图纸🙂", 4)
	if err := v.Flush(); err != nil || v.host.Buffer.History().UndoDepth != 1 || !strings.Contains(v.host.Buffer.String(), "图纸🙂") {
		t.Fatal("drawio helper behavior changed", err)
	}
	if !v.host.Buffer.Undo() || v.host.Buffer.String() != testDrawio {
		t.Fatal("drawio exact undo changed")
	}
	tester.Frame()
	focusDraft(t, tester)
	tester.Compose("保留图纸", 4)
	var recovered string
	v.host.RecoverDraft = func(_, content string) error { recovered = content; return nil }
	if err := v.RecoverDraft(); err != nil || recovered != "sample.drawio\n节点 ID: a\n页面 ID: p1\n字段: 节点文本\n原字段存在: true\n\n保留图纸" {
		t.Fatal("drawio legacy recovery", err, recovered)
	}
}

func TestTagsNativeCollapsedDualDraftRecovery(t *testing.T) {
	v := newView(kindMindmap, editor.Host{Buffer: text.New(tagsFixture())})
	tester := ui.NewTester(v.Draw, 760, 480)
	openTags(t, tester)
	typeTag(t, tester, "标签保持")
	revealFieldControl(t, tester, "节点文本", 760, 480)
	chooseField(t, tester, "节点文本")
	tester.Command("selectAll")
	tester.Type("字段保持")
	clickTag(t, tester, "收起标签面板")
	if v.tagOpen || !v.pendingDraft() || !v.pendingTagDraft() {
		t.Fatal("fixture must retain both pending after collapse")
	}
	var content string
	v.host.RecoverDraft = func(_, value string) error { content = value; return nil }
	clickTag(t, tester, "恢复节点及标签草稿")
	if !strings.Contains(content, "字段保持") || !strings.HasSuffix(content, "标签保持") || v.pendingGraphDraft() || v.host.Buffer.CanUndo() {
		t.Fatal("visible collapsed recovery refused/lost input", v.tagError, content)
	}
}

func TestTagsActualLimitsCandidateAndRecoveryRemainExact(t *testing.T) {
	for _, count := range []int{64, 65, 10000} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			tagJSON, _ := json.Marshal(make([]string, count))
			source := `{"format":"levnote.mindmap","version":1,"root":{"id":"root","topic":"根","tags":` + string(tagJSON) + `}}`
			v := stagedTagView(source, "new")
			info := v.tagPresentation()
			if info.count != count || len(info.summary) > 512 || !utf8.ValidString(info.summary) || !strings.Contains(info.summary, fmt.Sprint(count)+" 个标签") || len(info.rows) > 64 {
				t.Fatal("bounded count/summary", info.count, len(info.summary), len(info.rows))
			}
			if err := v.activateTag(currentTagTarget(v, tagAdd)); !errors.Is(err, errTagBudget) || v.host.Buffer.CanUndo() || v.tagDraft.input.Buffer.String() != "new" {
				t.Fatal("actual count/candidate guard", err)
			}
			if count > 64 {
				v.clearTagDraft()
				if err := v.activateTag(currentTagTarget(v, tagClear)); !errors.Is(err, errTagBudget) || v.host.Buffer.String() != source {
					t.Fatal("oversize existing clear bypassed budget", err)
				}
			}
		})
	}
	// Current source/candidate prospective descriptors fit, but the new
	// serialized string also acquires Raw bytes. Reject before source commit.
	source := `{"format":"levnote.mindmap","version":1,"root":{"id":"root","topic":"根","tags":["` + strings.Repeat("x", 3838) + `"]}}`
	v := stagedTagView(source, strings.Repeat("y", 512))
	if err := v.activateTag(currentTagTarget(v, tagAdd)); !errors.Is(err, errTagBudget) || v.host.Buffer.String() != source || v.host.Buffer.CanUndo() || !v.pendingTagDraft() {
		t.Fatal("post-serialization actual budget guard", err)
	}
	// Reject a whole-document canonical candidate exceeding existing16MiB.
	source = tagsFixture()
	source = strings.Replace(source, `"原备注"`, `"`+strings.Repeat("x", mindmap.DefaultMaxBytes-len(source)-4)+`"`, 1)
	v = stagedTagView(source, "new")
	if err := v.activateTag(currentTagTarget(v, tagAdd)); !errors.Is(err, mindmap.ErrLimitExceeded) || v.host.Buffer.String() != source || v.host.Buffer.CanUndo() {
		t.Fatal("document candidate committed beyond preview limit", err)
	}
	// The host owns recovery's4MiB limit. Its rejection retains both strings.
	large := strings.Repeat("z", 4*1024*1024+1)
	v = stagedTagView(tagsFixture(), large)
	_ = v.draft.input.Buffer.ReplaceUTF16(0, v.draft.input.Buffer.LenUTF16(), "field")
	v.host.RecoverDraft = func(_, content string) error {
		if len(content) <= 4*1024*1024 || !strings.HasSuffix(content, large) {
			t.Fatal("recovery truncated oversized input")
		}
		return errors.New("草稿副本超过4MiB")
	}
	if err := v.RecoverDraft(); err == nil || v.draft.input.Buffer.String() != "field" || v.tagDraft.input.Buffer.String() != large || v.host.Buffer.CanUndo() {
		t.Fatal("failed bounded host recovery dropped staged data", err)
	}
}

func TestTagsPendingReleaseRefreshAndSelectionClamp(t *testing.T) {
	v := stagedTagView(tagsFixture(), "tag")
	_ = v.draft.input.Buffer.ReplaceUTF16(0, v.draft.input.Buffer.LenUTF16(), "field")
	d, tag := v.draft, v.tagDraft
	replaceSource(t, v.host.Buffer, `"id":"root"`, `"id":"replacement"`)
	authority := v.host.Buffer.String()
	v.ensureSynced(nil)
	v.Release()
	if v.draft != d || v.tagDraft != tag || v.selectedID != "root" || v.mind != nil || v.tagInfo.loaded || v.tagPress.active {
		t.Fatal("pending owner/cache not retained/released")
	}
	v.host.Buffer = text.New(testMindmap)
	v.ensureSynced(nil)
	if err := v.activateTag(currentTagTarget(v, tagAdd)); !errors.Is(err, errTagInput) || v.draft != d || v.tagDraft != tag || v.host.Buffer.String() != testMindmap {
		t.Fatal("pending input reassigned to replacement buffer", err, authority)
	}
	encoded, _ := json.Marshal([]string{strings.Repeat("x", 3000)})
	source := `{"format":"levnote.mindmap","version":1,"root":{"id":"root","topic":"根","tags":` + string(encoded) + `}}`
	v = stagedTagView(source, "")
	_ = v.host.Buffer.SetSelection(v.host.Buffer.LenUTF16(), v.host.Buffer.LenUTF16()-3)
	selection := v.host.Buffer.Selection()
	if err := v.activateTag(currentTagTarget(v, tagClear)); err != nil || !strings.Contains(strings.Join(v.Warnings(), " "), "修改标签后源码") {
		t.Fatal("selection clamp disclosure", err, v.Warnings())
	}
	if v.host.Buffer.Selection() == selection || !v.host.Buffer.Undo() || v.host.Buffer.String() != source || v.host.Buffer.Selection() != selection {
		t.Fatal("clamp/undo did not preserve original exact source selection")
	}
}
