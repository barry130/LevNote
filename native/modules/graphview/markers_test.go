package graphview

import (
	"encoding/json"
	"errors"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/mindmap"
	"levnote.local/native/ui/editor"
)

// Deliberately retain raw JSON number spellings and rich tags/images/unknown
// fields. Expected arrays below are independent literals, never helper output.
func markerFixture(values []string, present bool) string {
	icons := ""
	if present {
		if values == nil {
			values = []string{}
		}
		data, _ := json.Marshal(values)
		icons = `,"icons":` + string(data)
	}
	return `{"format":"levnote.mindmap","version":1,"layout":"logicalStructure",` +
		`"futureTop":{"number":1.00},"root":{"id":"root","topic":"中心","note":"原备注","hyperLink":"child.txt",` +
		`"tags":[{"text":"标签","style":{"color":"red","future":1.20},"className":"dense","riskLevel":2.00}],` +
		`"image":{"url":"images/local.png","width":24,"height":16,"future":0.30},` +
		`"unknownNode":{"number":1e3}` + icons + `,"children":[{"id":"left","topic":"子主题","icons":["❓"],"children":[]}]}}`
}

func readMarkers(t *testing.T, buffer *text.Buffer, id string) ([]string, bool) {
	t.Helper()
	values, present, err := parseMind(t, buffer).Icons(id)
	if err != nil {
		t.Fatal(err)
	}
	return values, present
}

func expectMarkers(t *testing.T, buffer *text.Buffer, expected []string, present bool) {
	t.Helper()
	got, exists := readMarkers(t, buffer, "root")
	if exists != present || !slices.Equal(got, expected) {
		t.Fatalf("icons = %#v present=%t, want %#v present=%t", got, exists, expected, present)
	}
}

func openMarkers(t *testing.T, tester *ui.Tester, width, height float32) {
	t.Helper()
	revealFieldControl(t, tester, "展开标记面板", width, height)
	chooseField(t, tester, "展开标记面板")
}

func clickMarker(t *testing.T, tester *ui.Tester, name string, width, height float32) {
	t.Helper()
	revealFieldControl(t, tester, name, width, height)
	chooseField(t, tester, name)
}

func markerTarget(v *view, action int) markerGesture {
	v.ensureSynced(nil)
	return v.captureMarker(action, v.host.Buffer, v.selectedID, v.host.Buffer.Version())
}

func TestMarkersNativeDiscoverySelectionAndIndependentVocabulary(t *testing.T) {
	source := markerFixture([]string{"custom", "2️⃣", "🟥", "2️⃣", "custom", "⭐", "⭐", "🟦"}, true)
	buffer := text.New(source)
	v := newView(kindMindmap, editor.Host{Buffer: buffer})
	tester := ui.NewTester(v.Draw, 760, 480)
	if _, ok := tester.AccessByRole(ui.RoleButton, "优先级 1"); ok || v.markerOpen {
		t.Fatal("collapsed section built palette controls")
	}
	openMarkers(t, tester, 760, 480)
	for _, label := range []string{"优先级 2", "红色标记", "蓝色标记", "星标标记"} {
		access, ok := tester.AccessByRole(ui.RoleButton, label)
		if !ok || !access.Checked || access.Disabled || !strings.Contains(strings.Join(tester.Texts(), " "), "✓") {
			t.Fatal("preexisting duplicates/same-group marks not represented as selected", label, access)
		}
	}
	if buffer.String() != source || buffer.CanUndo() {
		t.Fatal("viewing marker panel changed original bytes")
	}
	steps := []struct {
		label string
		want  []string
	}{
		{"优先级 2", []string{"custom", "🟥", "custom", "⭐", "⭐", "🟦"}},
		{"优先级 3", []string{"custom", "🟥", "custom", "⭐", "⭐", "🟦", "3️⃣"}},
		{"优先级 4", []string{"custom", "🟥", "custom", "⭐", "⭐", "🟦", "4️⃣"}},
		{"绿色标记", []string{"custom", "custom", "⭐", "⭐", "4️⃣", "🟩"}},
		{"星标标记", []string{"custom", "custom", "4️⃣", "🟩"}},
		{"完成标记", []string{"custom", "custom", "4️⃣", "🟩", "✅"}},
		{"疑问标记", []string{"custom", "custom", "4️⃣", "🟩", "✅", "❓"}},
	}
	for _, step := range steps {
		clickMarker(t, tester, step.label, 760, 480)
		expectMarkers(t, buffer, step.want, true)
	}
	clickMarker(t, tester, "收起标记面板", 760, 480)
	if _, ok := tester.AccessByRole(ui.RoleButton, "优先级 1"); ok {
		t.Fatal("collapse retained palette")
	}
}

func TestMarkersNativePointerTabEnterSpaceOneTransactionSelectionAndMetadata(t *testing.T) {
	for _, mode := range []string{"pointer", "enter", "space"} {
		t.Run(mode, func(t *testing.T) {
			source := markerFixture([]string{"custom", "1️⃣", "custom"}, true)
			buffer := text.New(source)
			_ = buffer.SetSelection(11, 3)
			selection := buffer.Selection()
			changed := 0
			v := newView(kindMindmap, editor.Host{Buffer: buffer, Changed: func() { changed++ }})
			tester := ui.NewTester(v.Draw, 760, 480)
			openMarkers(t, tester, 760, 480)
			if mode == "pointer" {
				clickMarker(t, tester, "优先级 2", 760, 480)
			} else {
				for i := 0; i < 35 && !tester.Focused("优先级 2"); i++ {
					tester.Key(0, ui.KeyTab)
				}
				if !tester.Focused("优先级 2") {
					t.Fatal("native Tab did not reach priority button")
				}
				key := ui.KeyEnter
				if mode == "space" {
					key = ui.KeySpace
				}
				tester.KeyDown(key)
				tester.KeyRepeat(0, key)
				tester.KeyUp(key)
			}
			expectMarkers(t, buffer, []string{"custom", "custom", "2️⃣"}, true)
			if changed != 1 || buffer.History().UndoDepth != 1 || buffer.Selection() != selection || v.selectedID != "root" {
				t.Fatal("marker input duplicated commit or lost host selection", changed, buffer.History(), buffer.Selection())
			}
			committed := buffer.String()
			var before, after map[string]any
			_ = json.Unmarshal([]byte(source), &before)
			_ = json.Unmarshal([]byte(committed), &after)
			delete(before["root"].(map[string]any), "icons")
			delete(after["root"].(map[string]any), "icons")
			if !reflect.DeepEqual(before, after) {
				t.Fatal("unrelated rich metadata changed")
			}
			for _, spelling := range []string{"1.00", "1.20", "2.00", "0.30", "1e3", `"className": "dense"`} {
				if !strings.Contains(committed, spelling) {
					t.Fatal("real edit lost unknown/rich number spelling", spelling)
				}
			}
			parsed := parseMind(t, buffer)
			for _, raw := range []struct {
				object mindmap.Object
				key    string
				want   string
			}{
				{parsed.Extra, "futureTop", `{"number":1.00}`},
				{parsed.Root.Extra, "unknownNode", `{"number":1e3}`},
				{(*parsed.Root.Tags)[0].Style, "future", `1.20`},
				{(*parsed.Root.Tags)[0].Extra, "riskLevel", `2.00`},
				{parsed.Root.Image.Extra, "future", `0.30`},
			} {
				got, exists := raw.object.Get(raw.key)
				if !exists || string(got) != raw.want {
					t.Fatalf("raw %s = %s, want %s", raw.key, got, raw.want)
				}
			}
			if !buffer.Undo() || buffer.String() != source || buffer.Selection() != selection || !buffer.Redo() || buffer.String() != committed || buffer.Selection() != selection {
				t.Fatal("one undo/redo did not preserve bytes/source selection")
			}
			tester.Frame()
			if v.markerPress.active || v.markerPress.icons != nil {
				t.Fatal("released native input retained snapshot")
			}
		})
	}
}

func TestMarkersClearPresenceNoopAndSingleUndo(t *testing.T) {
	for _, present := range []bool{false, true} {
		source := markerFixture(nil, present)
		buffer := text.New(source)
		changed := 0
		v := newView(kindMindmap, editor.Host{Buffer: buffer, Changed: func() { changed++ }})
		tester := ui.NewTester(v.Draw, 760, 480)
		openMarkers(t, tester, 760, 480)
		access, ok := tester.AccessByRole(ui.RoleButton, "清空全部标记")
		if !ok || !access.Disabled || !strings.Contains(access.Description, "自定义") {
			t.Fatal("empty/absent clear was enabled or unexplained", access)
		}
		if err := v.activateMarker(markerTarget(v, markerClear)); err != nil || buffer.String() != source || buffer.CanUndo() || changed != 0 {
			t.Fatal("semantic no-op clear changed bytes/history", err)
		}
		expectMarkers(t, buffer, nil, present)
	}
	source := markerFixture([]string{"custom", "⭐", "custom"}, true)
	buffer := text.New(source)
	v := newView(kindMindmap, editor.Host{Buffer: buffer})
	tester := ui.NewTester(v.Draw, 760, 480)
	openMarkers(t, tester, 760, 480)
	clickMarker(t, tester, "清空全部标记", 760, 480)
	expectMarkers(t, buffer, []string{}, true)
	if !strings.Contains(buffer.String(), `"icons": []`) || buffer.History().UndoDepth != 1 || !buffer.Undo() || buffer.String() != source {
		t.Fatal("clear failed to write present[] or undo all custom markers")
	}
}

func TestMarkersNativeHeldGestureIdentityRefusalAndMerge(t *testing.T) {
	for _, change := range []string{"reorder", "change", "presence", "delete", "node", "buffer", "group", "readonly", "oversize", "invalid", "unrelated", "cancel", "outside", "scroll", "collapse", "release"} {
		t.Run(change, func(t *testing.T) {
			source := markerFixture([]string{"custom", "1️⃣"}, true)
			if change == "presence" {
				source = markerFixture(nil, false)
			}
			buffer := text.New(source)
			changed := 0
			v := newView(kindMindmap, editor.Host{Buffer: buffer, Changed: func() { changed++ }})
			tester := ui.NewTester(v.Draw, 760, 480)
			openMarkers(t, tester, 760, 480)
			revealFieldControl(t, tester, "优先级 2", 760, 480)
			r, _ := tester.Find("优先级 2")
			x, y := r.X+r.W/2, r.Y+r.H/2
			tester.Press(x, y)
			if !v.markerPress.active || v.markerPress.buffer != buffer || v.markerPress.nodeID != "root" || v.markerPress.action != 1 {
				t.Fatal("actual press did not capture bounded action identity")
			}
			switch change {
			case "reorder":
				replaceSource(t, buffer, `["custom","1️⃣"]`, `["1️⃣","custom"]`)
			case "change":
				replaceSource(t, buffer, `["custom","1️⃣"]`, `["custom","3️⃣"]`)
			case "presence":
				replaceSource(t, buffer, `"id":"root"`, `"id":"root","icons":[]`)
			case "delete":
				replaceSource(t, buffer, `"id":"root"`, `"id":"replacement"`)
			case "node":
				v.selectNode("left")
			case "buffer":
				v.host.Buffer = text.New(source)
			case "group":
				_ = buffer.BeginGroup()
			case "readonly":
				v.host.ReadOnly = true
			case "oversize":
				_ = buffer.ReplaceUTF16(buffer.LenUTF16(), buffer.LenUTF16(), strings.Repeat(" ", mindmap.DefaultMaxBytes+1))
			case "invalid":
				_ = buffer.ReplaceUTF16(0, buffer.LenUTF16(), "{broken")
			case "unrelated":
				replaceSource(t, buffer, `"topic":"中心"`, `"topic":"外部主题"`)
			case "cancel":
				tester.CancelPress()
			case "outside":
				x, y = 10, 10
			case "scroll":
				tester.Wheel(750, 445, 0, 30)
				tester.Advance(time.Second)
			case "collapse":
				v.markerOpen = false
			case "release":
				v.Release()
			}
			authority, history := v.host.Buffer.String(), v.host.Buffer.History()
			tester.Frame() // fixed-key controls rebuilt before original release
			tester.Release(x, y)
			if change == "unrelated" {
				expectMarkers(t, buffer, []string{"custom", "2️⃣"}, true)
				if parseMind(t, buffer).Root.Topic != "外部主题" || changed != 1 {
					t.Fatal("unrelated source edit did not merge", v.markerError)
				}
			} else if v.host.Buffer.String() != authority || v.host.Buffer.History() != history || changed != 0 {
				t.Fatal("stale/canceled press changed or retargeted source", change, v.markerError)
			}
			if slices.Contains([]string{"reorder", "change", "presence", "delete", "node", "buffer"}, change) &&
				!errors.Is(v.markerError, errMarkerStale) {
				t.Fatal("identity refusal not visible", change, v.markerError, tester.Texts())
			}
			if v.markerPress.active || v.markerPress.icons != nil {
				t.Fatal("completed/canceled gesture retained snapshot")
			}
			if change == "group" && !buffer.InGroup() {
				t.Fatal("marker action stole foreign group")
			}
		})
	}
}

func TestMarkersNativeStaleWithoutDrawAndKeyboardIdentity(t *testing.T) {
	for _, mode := range []string{"pointer", "enter", "space"} {
		t.Run(mode, func(t *testing.T) {
			buffer := text.New(markerFixture([]string{"1️⃣"}, true))
			v := newView(kindMindmap, editor.Host{Buffer: buffer})
			armed := false
			tester := ui.NewTester(func(c *ui.Context) {
				ui.Column(c).Fill().ObserveInput(func(ev ui.InputEvent) {
					if armed && (ev.Kind == ui.InputPointerDown || ev.Kind == ui.InputKeyDown && (ev.Key == ui.KeyEnter || ev.Key == ui.KeySpace)) {
						armed = false
						replaceSource(t, buffer, `["1️⃣"]`, `["3️⃣"]`)
					}
				}).Children(func() { v.Draw(c) })
			}, 760, 480)
			openMarkers(t, tester, 760, 480)
			revealFieldControl(t, tester, "优先级 2", 760, 480)
			if mode == "pointer" {
				armed = true
				chooseField(t, tester, "优先级 2")
			} else {
				for i := 0; i < 35 && !tester.Focused("优先级 2"); i++ {
					tester.Key(0, ui.KeyTab)
				}
				if !tester.Focused("优先级 2") {
					t.Fatal("priority not keyboard reachable")
				}
				armed = true
				key := ui.KeyEnter
				if mode == "space" {
					key = ui.KeySpace
				}
				tester.Key(0, key)
			}
			expectMarkers(t, buffer, []string{"3️⃣"}, true)
			problem, visible := tester.Access("标记操作问题")
			if !errors.Is(v.markerError, errMarkerStale) || buffer.History().UndoDepth != 1 || !visible || !strings.Contains(problem.Description, "未完成") {
				t.Fatal("input rebound to source updated before Draw", v.markerError, buffer.History())
			}
		})
	}
}

func TestMarkersPendingFieldFlushMergeFailureAndRecovery(t *testing.T) {
	for _, failure := range []bool{false, true} {
		buffer := text.New(markerFixture([]string{"1️⃣", "custom"}, true))
		changed, recovered := 0, ""
		v := newView(kindMindmap, editor.Host{Buffer: buffer, Changed: func() { changed++ }, RecoverDraft: func(_, value string) error { recovered = value; return nil }})
		tester := ui.NewTester(v.Draw, 760, 480)
		chooseField(t, tester, "字段：纯文本备注")
		openMarkers(t, tester, 760, 480)
		revealFieldControl(t, tester, "节点备注（纯文本）", 760, 480)
		composeField(t, tester, "节点备注（纯文本）", "本地备注🙂\r\n  ")
		draft := v.draft
		if failure {
			replaceSource(t, buffer, `"note":"原备注"`, `"note":"冲突源备注"`)
		} else {
			replaceSource(t, buffer, `"topic":"中心"`, `"topic":"外部改过主题"`)
		}
		tester.Frame()
		clickMarker(t, tester, "优先级 2", 760, 480)
		if !failure {
			expectMarkers(t, buffer, []string{"custom", "2️⃣"}, true)
			if v.field != fieldNote || v.draft != draft || parseMind(t, buffer).Root.Topic != "外部改过主题" || *parseMind(t, buffer).Root.Note != "本地备注🙂\r\n  " || changed != 2 || buffer.History().UndoDepth != 3 {
				t.Fatal("pending field and marker did not merge as independent transactions", changed, buffer.History(), v.markerError)
			}
			if !buffer.Undo() {
				t.Fatal("missing marker undo")
			}
			expectMarkers(t, buffer, []string{"1️⃣", "custom"}, true)
			if *parseMind(t, buffer).Root.Note != "本地备注🙂\r\n  " {
				t.Fatal("marker undo rolled back pending field's separate commit")
			}
		} else {
			if !errors.Is(v.draftError, ErrDraftConflict) || !errors.Is(v.markerError, ErrDraftConflict) || v.draft != draft || !v.pendingDraft() || changed != 0 {
				t.Fatal("failed field Flush lost draft or applied marker", v.draftError, v.markerError)
			}
			expectMarkers(t, buffer, []string{"1️⃣", "custom"}, true)
			selection, history, value, authority := draft.input.Buffer.Selection(), draft.input.Buffer.History(), draft.input.Buffer.String(), buffer.String()
			v.Release()
			if v.draft != draft || draft.input.Buffer.Selection() != selection || draft.input.Buffer.History() != history || draft.input.Buffer.String() != value {
				t.Fatal("Release changed failed exact draft state")
			}
			if err := v.RecoverDraft(); err != nil || !strings.HasSuffix(recovered, "本地备注🙂\r\n  ") || buffer.String() != authority {
				t.Fatal("marker-blocked draft was not recoverable", err, recovered)
			}
		}
	}
}

func TestMarkersBudgetCurrentCandidateAndSummary(t *testing.T) {
	// This boundary previously stopped after a fitting first glyph without
	// disclosing that the next glyph was omitted.
	for _, count := range []int{458, 459, 460, 461, 462, 463, 464, 465, 466} {
		value := markerSummary([]string{strings.Repeat("x", count), strings.Repeat("中🙂", 20)}, true)
		if len(value) > 512 || !utf8.ValidString(value) || !strings.Contains(value, "…；完整列表见源码。") {
			t.Fatal("truncated summary omitted ellipsis or crossed byte/rune bound", count, len(value), value)
		}
	}
	for _, tc := range []struct {
		name    string
		values  []string
		blocked bool
	}{
		{"64", append(slices.Repeat([]string{"custom"}, 63), "⭐"), false},
		{"65", slices.Repeat([]string{"custom"}, 65), true},
		{"4096", []string{strings.Repeat("x", 4093), "⭐"}, false},
		{"4097", []string{strings.Repeat("x", 4094), "⭐"}, true},
		{"multirune", []string{strings.Repeat("中🙂", 900)}, true},
		{"empty-huge", slices.Repeat([]string{""}, 1000), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := markerFixture(tc.values, true)
			buffer := text.New(source)
			v := newView(kindMindmap, editor.Host{Buffer: buffer})
			tester := ui.NewTester(v.Draw, 760, 480)
			openMarkers(t, tester, 760, 480)
			info, ok := tester.Access("当前节点标记摘要")
			if !ok || len(info.Description) > 512 || !utf8.ValidString(info.Description) || !strings.Contains(info.Description, "源码") {
				t.Fatal("summary not bounded/rune-safe or omitted source guidance", len(info.Description), info)
			}
			for _, label := range []string{"星标标记", "清空全部标记"} {
				access, ok := tester.AccessByRole(ui.RoleButton, label)
				if !ok || access.Disabled != tc.blocked {
					t.Fatal("current budget failed to disable native action", label, access)
				}
			}
			if tc.blocked {
				if err := v.activateMarker(markerTarget(v, markerClear)); !errors.Is(err, errMarkerBudget) || buffer.String() != source || buffer.CanUndo() {
					t.Fatal("oversize clear bypassed action budget", err)
				}
			} else {
				clickMarker(t, tester, "星标标记", 760, 480)
				want := slices.Clone(tc.values[:len(tc.values)-1])
				expectMarkers(t, buffer, want, true)
			}
		})
	}
	for _, values := range [][]string{slices.Repeat([]string{"custom"}, 64), {strings.Repeat("x", 4096)}} {
		buffer := text.New(markerFixture(values, true))
		v := newView(kindMindmap, editor.Host{Buffer: buffer})
		tester := ui.NewTester(v.Draw, 760, 480)
		openMarkers(t, tester, 760, 480)
		before := buffer.String()
		clickMarker(t, tester, "星标标记", 760, 480)
		problem, visible := tester.Access("标记操作问题")
		if !errors.Is(v.markerError, errMarkerCandidate) || buffer.String() != before || buffer.CanUndo() || !visible || !strings.Contains(problem.Description, "操作后标记将超过") {
			t.Fatal("prospective overflow mutated source or hid refusal", v.markerError)
		}
	}
}

func TestMarkersReadonlyReleaseAndIdleCache(t *testing.T) {
	source := markerFixture([]string{"⭐", "custom"}, true)
	buffer := text.New(source)
	v := newView(kindMindmap, editor.Host{Buffer: buffer, ReadOnly: true})
	tester := ui.NewTester(v.Draw, 760, 480)
	openMarkers(t, tester, 760, 480)
	access, ok := tester.AccessByRole(ui.RoleButton, "星标标记")
	if !ok || !access.Disabled || !access.Checked || !tester.HasText("当前文档只读") {
		t.Fatal("readonly section cannot reveal/read selected marks", access)
	}
	if err := v.activateMarker(markerTarget(v, 15)); !errors.Is(err, errMarkerReadOnly) || buffer.String() != source || buffer.CanUndo() {
		t.Fatal("readonly mutation was not refused", err)
	}
	doc, draft, summary := v.mind, v.draft, v.markerInfo.summary
	for i := 0; i < 5; i++ {
		tester.Frame()
	}
	if v.mind != doc || v.draft != draft || v.markerInfo.summary != summary || tester.FramePending() {
		t.Fatal("idle marker frames rebuilt graph/draft or self-invalidated")
	}
	v.markerPress = markerTarget(v, 15)
	v.Release()
	if v.markerPress.active || v.markerPress.icons != nil || v.markerInfo.loaded || v.markerInfo.buffer != nil || v.markerOpen || v.mind != nil || v.labels.layouts != nil {
		t.Fatal("Release retained marker snapshots/presentation/render cache")
	}
	draw := newView(kindDrawio, editor.Host{Buffer: text.New(testDrawio)})
	drawTester := ui.NewTester(draw.Draw, 760, 480)
	if _, ok := drawTester.AccessByRole(ui.RoleButton, "展开标记面板"); ok {
		t.Fatal("marker inspector leaked into drawio")
	}
}

func TestMarkersCandidateDocumentLimitAndSelectionClamp(t *testing.T) {
	// Compact source fits the parser, but its canonical representation plus a
	// real marker would exceed the existing document byte limit.
	source := markerFixture(nil, false)
	source = strings.Replace(source, `"原备注"`, `"`+strings.Repeat("x", mindmap.DefaultMaxBytes-len(source)-4)+`"`, 1)
	buffer := text.New(source)
	v := newView(kindMindmap, editor.Host{Buffer: buffer})
	target := markerTarget(v, 15)
	if len(source) >= mindmap.DefaultMaxBytes {
		t.Fatal("fixture must fit before canonical serialization")
	}
	if err := v.activateMarker(target); !errors.Is(err, mindmap.ErrLimitExceeded) || buffer.String() != source || buffer.CanUndo() {
		t.Fatal("candidate document limit applied after commit", err)
	}
	buffer = text.New(markerFixture([]string{strings.Repeat("x", 4096)}, true))
	_ = buffer.SetSelection(buffer.LenUTF16(), buffer.LenUTF16()-2)
	selection := buffer.Selection()
	v = newView(kindMindmap, editor.Host{Buffer: buffer})
	if err := v.activateMarker(markerTarget(v, markerClear)); err != nil || !strings.Contains(strings.Join(v.Warnings(), " "), "修改标记后源码") || strings.Contains(strings.Join(v.Warnings(), " "), "展开节点后") {
		t.Fatal("clamping omitted marker-specific wording", err, v.Warnings())
	}
	if buffer.Selection() == selection || !buffer.Undo() || buffer.Selection() != selection {
		t.Fatal("clamped selection/undo not preserved")
	}
}

func TestMarkersInspectorImagesAndRecoveryReachable(t *testing.T) {
	dir := os.Getenv("LEVNOTE_GRAPH_MARKERS_SCREENSHOT_DIR")
	if dir == "" {
		t.Skip("set LEVNOTE_GRAPH_MARKERS_SCREENSHOT_DIR")
	}
	for _, mode := range []string{"light", "dark"} {
		buffer := text.New(markerFixture([]string{"2️⃣", "🟦", "⭐", "custom"}, true))
		v := newView(kindMindmap, editor.Host{Buffer: buffer, OpenReference: func(string) {}, RecoverDraft: func(string, string) error { return nil }})
		tester := ui.NewTester(v.Draw, 760, 480)
		tester.SetDark(mode == "dark")
		chooseField(t, tester, "字段：纯文本备注")
		openMarkers(t, tester, 760, 480)
		clickMarker(t, tester, "疑问标记", 760, 480)
		for _, name := range []string{"节点备注（纯文本）", "收起标记面板", "优先级 1", "清空全部标记", "打开链接"} {
			revealFieldControl(t, tester, name, 760, 480)
		}
		revealFieldControl(t, tester, "收起标记面板", 760, 480)
		tester.Wheel(750, 445, 0, 70)
		tester.Advance(time.Second)
		saveMarkerImage(t, dir, "mindmap-markers-"+mode+"-760x480.png", tester)
		revealFieldControl(t, tester, "节点备注（纯文本）", 760, 480)
		composeField(t, tester, "节点备注（纯文本）", "冲突草稿仍可恢复🙂")
		replaceSource(t, buffer, `"note": "原备注"`, `"note": "外部备注"`)
		tester.Frame()
		clickMarker(t, tester, "优先级 3", 760, 480)
		for _, name := range []string{"重试写回", "恢复为新文档", "清空全部标记", "节点备注（纯文本）"} {
			revealFieldControl(t, tester, name, 760, 480)
		}
		revealFieldControl(t, tester, "恢复为新文档", 760, 480)
		saveMarkerImage(t, dir, "mindmap-markers-recovery-"+mode+"-760x480.png", tester)
	}
}

func saveMarkerImage(t *testing.T, dir, name string, tester *ui.Tester) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, tester.Image()); err != nil {
		t.Fatal(err)
	}
}
