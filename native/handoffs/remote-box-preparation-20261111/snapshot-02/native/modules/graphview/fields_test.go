package graphview

import (
	"encoding/json"
	"errors"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/mindmap"
	"levnote.local/native/ui/editor"
)

func fieldFixture(note, href *string) string {
	root := map[string]any{"id": "root", "topic": "中心", "future": map[string]any{"keep": "原样"}, "children": []any{
		map[string]any{"id": "left", "topic": "子主题", "hyperLink": "child.txt"},
	}}
	if note != nil {
		root["note"] = *note
	}
	if href != nil {
		root["hyperLink"] = *href
	}
	data, _ := json.Marshal(map[string]any{"format": "levnote.mindmap", "version": 1, "layout": "logicalStructure", "root": root, "future": 4})
	return string(data)
}

func stringPtr(value string) *string { return &value }

func chooseField(t *testing.T, tester *ui.Tester, name string) {
	t.Helper()
	if err := tester.Click(name); err != nil {
		t.Fatal(err)
	}
}

func composeField(t *testing.T, tester *ui.Tester, label, value string) {
	t.Helper()
	chooseField(t, tester, label)
	tester.Command("selectAll")
	tester.Compose(value, len([]rune(value)))
}

func revealFieldControl(t *testing.T, tester *ui.Tester, label string, width, height float32) {
	t.Helper()
	tester.Wheel(width-6, height-35, 0, -3000)
	tester.Advance(time.Second)
	for i := 0; i < 20; i++ {
		r, ok := tester.Find(label)
		if ok && r.W > 0 && r.H > 0 && r.X >= 0 && r.Y >= 0 && r.X+r.W <= width+0.1 && r.Y+r.H <= height+0.1 {
			return
		}
		tester.Wheel(width-6, height-35, 0, 30)
		tester.Advance(time.Second)
	}
	t.Fatalf("%s not reachable inside %vx%v", label, width, height)
}

func TestMindmapFieldsNativeOptionalNoopComposeAndUndo(t *testing.T) {
	for _, initial := range []*string{nil, stringPtr("")} {
		source := fieldFixture(initial, stringPtr("same.txt"))
		buffer := text.New(source)
		changed := 0
		v := newView(kindMindmap, editor.Host{Buffer: buffer, Changed: func() { changed++ }})
		tester := ui.NewTester(v.Draw, 760, 480)
		for _, label := range []string{"字段：纯文本备注", "字段：本地链接", "字段：主题文本", "字段：纯文本备注"} {
			chooseField(t, tester, label)
			if err := v.Flush(); err != nil {
				t.Fatal(err)
			}
		}
		if buffer.String() != source || buffer.CanUndo() || changed != 0 || (parseMind(t, buffer).Root.Note == nil) != (initial == nil) {
			t.Fatal("clean field selection changed optional state/source/history")
		}
		value := "备注中文🙂\r\n第二行  "
		composeField(t, tester, "节点备注（纯文本）", value)
		if buffer.String() != source {
			t.Fatal("preedit leaked into authoritative source")
		}
		chooseField(t, tester, "字段：本地链接")
		if got := parseMind(t, buffer).Root.Note; got == nil || *got != value || changed != 1 || buffer.History().UndoDepth != 1 {
			t.Fatal("field-switch Flush lost exact note or duplicated transaction", got, changed, buffer.History())
		}
		if err := v.Flush(); err != nil || changed != 1 {
			t.Fatal("noop barrier added history", err)
		}
		committed := buffer.String()
		if !buffer.Undo() || buffer.String() != source || !buffer.Redo() || buffer.String() != committed {
			t.Fatal("one field transaction did not preserve exact undo/redo")
		}
		chooseField(t, tester, "字段：纯文本备注")
		chooseField(t, tester, "节点备注（纯文本）")
		tester.Command("selectAll")
		tester.Key(0, ui.KeyBackspace)
		if got := parseMind(t, buffer).Root.Note; got == nil || *got != "" {
			t.Fatal("cleared note lost explicit presence")
		}
		chooseField(t, tester, "字段：本地链接")
		chooseField(t, tester, "节点链接（本地）")
		tester.Command("selectAll")
		tester.Key(0, ui.KeyBackspace)
		if parseMind(t, buffer).Root.HyperLink != nil || v.draft.sourcePresent {
			t.Fatal("cleared hyperlink must be absent")
		}
		if !strings.Contains(buffer.String(), `"future"`) || !strings.Contains(buffer.String(), "原样") {
			t.Fatal("field edits removed unknown metadata")
		}
	}
}

func TestMindmapFieldsNativeLinkComposePointerKeyboardAndRelease(t *testing.T) {
	for _, mode := range []string{"pointer", "enter", "space", "release"} {
		t.Run(mode, func(t *testing.T) {
			buffer := text.New(fieldFixture(nil, stringPtr("old.txt")))
			var opened []string
			var reported error
			var v *view
			v = newView(kindMindmap, editor.Host{Buffer: buffer, Report: func(err error) { reported = err }, OpenReference: func(href string) {
				opened = append(opened, href)
				if mode == "release" {
					v.Release()
				}
			}})
			tester := ui.NewTester(v.Draw, 760, 480)
			chooseField(t, tester, "字段：本地链接")
			composeField(t, tester, "节点链接（本地）", "子目录/目标🙂.txt")
			if mode == "enter" || mode == "space" {
				// Property focus escape now preserves active composition. Confirm
				// it through native text input before traversing the controls.
				tester.Type("子目录/目标🙂.txt")
				for i := 0; i < 12 && !tester.Focused("打开链接"); i++ {
					tester.Key(0, ui.KeyTab)
				}
				if !tester.Focused("打开链接") {
					t.Fatal("Tab did not focus explicit write/open")
				}
				key := ui.KeyEnter
				if mode == "space" {
					key = ui.KeySpace
				}
				tester.Key(0, key)
			} else {
				chooseField(t, tester, "写回并打开链接")
			}
			if reported != nil || len(opened) != 1 || opened[0] != "子目录/目标🙂.txt" || buffer.History().UndoDepth != 1 {
				t.Fatal("native action lost draft identity or duplicated open", opened, reported, buffer.History())
			}
			tester.Frame()
			before, history := buffer.String(), buffer.History()
			selection := buffer.Selection()
			chooseField(t, tester, "打开链接")
			if len(opened) != 2 || buffer.String() != before || buffer.History() != history || buffer.Selection() != selection {
				t.Fatal("clean open changed source/history/selection")
			}
			tester.Frame()
		})
	}
}

func TestMindmapFieldsNativeConflictMergeReadonlyAndRecovery(t *testing.T) {
	source := fieldFixture(nil, stringPtr("local.txt"))
	buffer := text.New(source)
	var recovered string
	v := newView(kindMindmap, editor.Host{Buffer: buffer, Path: "fields.mindmap", RecoverDraft: func(_, value string) error { recovered = value; return nil }})
	tester := ui.NewTester(v.Draw, 760, 480)
	chooseField(t, tester, "字段：纯文本备注")
	composeField(t, tester, "节点备注（纯文本）", "保留🙂\r\n")
	replaceSource(t, buffer, `"topic":"中心"`, `"topic":"其他字段已改"`)
	tester.Frame()
	chooseField(t, tester, "字段：本地链接")
	if parseMind(t, buffer).Root.Topic != "其他字段已改" || *parseMind(t, buffer).Root.Note != "保留🙂\r\n" {
		t.Fatal("other-field merge was lost")
	}
	chooseField(t, tester, "字段：纯文本备注")
	composeField(t, tester, "节点备注（纯文本）", "冲突草稿")
	draft := v.draft
	replaceSource(t, buffer, `"note": "保留🙂\r\n"`, `"note": "source"`)
	authority := buffer.String()
	tester.Frame()
	chooseField(t, tester, "字段：本地链接")
	if v.field != fieldNote || v.draft != draft || !errors.Is(v.draftError, ErrDraftConflict) || buffer.String() != authority {
		t.Fatal("field switch lost conflicting input", v.draftError)
	}
	v.Release()
	tester.Frame()
	if v.draft != draft || !v.pendingDraft() {
		t.Fatal("Release lost field draft")
	}
	if err := v.RecoverDraft(); err != nil || !strings.Contains(recovered, "字段: 节点备注（纯文本）") || !strings.HasSuffix(recovered, "冲突草稿") || buffer.String() != authority {
		t.Fatal("recovery lost field identity/text", recovered, err)
	}
	readonly := newView(kindMindmap, editor.Host{Buffer: text.New(source), ReadOnly: true, OpenReference: func(string) {}})
	roTester := ui.NewTester(readonly.Draw, 760, 480)
	chooseField(t, roTester, "字段：纯文本备注")
	composeField(t, roTester, "节点备注（纯文本）", "禁止")
	chooseField(t, roTester, "字段：本地链接")
	chooseField(t, roTester, "打开链接")
	if readonly.host.Buffer.String() != source || readonly.host.Buffer.CanUndo() {
		t.Fatal("readonly fields mutated source")
	}
}

func TestMindmapLongNoteUnchangedFieldProbe(t *testing.T) {
	note := strings.Repeat("备注🙂", 20000)
	buffer := text.New(fieldFixture(&note, nil))
	v := newView(kindMindmap, editor.Host{Buffer: buffer, ReadOnly: true})
	tester := ui.NewTester(v.Draw, 532, 380)
	chooseField(t, tester, "字段：纯文本备注")
	tester.Frame()
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()
	for i := 0; i < 5; i++ {
		tester.Frame()
	}
	runtime.ReadMemStats(&after)
	t.Logf("long-note %dB readonly unchanged5frames: elapsed=%s allocated=%d allocs=%d pending=%v", len(note), time.Since(start), after.TotalAlloc-before.TotalAlloc, after.Mallocs-before.Mallocs, tester.FramePending())
	if v.draft.input.Buffer.String() != note || tester.FramePending() {
		t.Fatal("long note lost content or did not settle")
	}
}

func TestMindmapFieldsOptionalIdentityFailureLifecycle(t *testing.T) {
	for _, kind := range []string{"presence", "deleted", "buffer", "group", "readonly", "oversize"} {
		t.Run(kind, func(t *testing.T) {
			source := fieldFixture(nil, stringPtr("original.txt"))
			buffer := text.New(source)
			var recovered string
			v := newView(kindMindmap, editor.Host{Buffer: buffer, RecoverDraft: func(_, value string) error { recovered = value; return nil }})
			tester := ui.NewTester(v.Draw, 760, 480)
			chooseField(t, tester, "字段：纯文本备注")
			composeField(t, tester, "节点备注（纯文本）", "必须保留🙂\r\n ")
			draft := v.draft
			switch kind {
			case "presence":
				_ = buffer.ReplaceUTF16(0, buffer.LenUTF16(), fieldFixture(stringPtr(""), stringPtr("original.txt")))
			case "deleted":
				replaceSource(t, buffer, `"id":"root"`, `"id":"replacement"`)
			case "buffer":
				v.host.Buffer = text.New(source)
			case "group":
				if err := buffer.BeginGroup(); err != nil {
					t.Fatal(err)
				}
			case "readonly":
				v.host.ReadOnly = true
			case "oversize":
				_ = buffer.ReplaceUTF16(0, buffer.LenUTF16(), strings.Repeat(" ", mindmap.DefaultMaxBytes+1))
			}
			authority := v.host.Buffer.String()
			tester.Frame()
			chooseField(t, tester, "字段：本地链接")
			if v.draft != draft || v.field != fieldNote || v.draftError == nil || v.host.Buffer.String() != authority {
				t.Fatal("blocked field switch lost identity/source", v.draftError)
			}
			v.Release()
			tester.Frame()
			if v.draft != draft || !v.pendingDraft() || v.draft.input.Buffer.String() != "必须保留🙂\r\n " {
				t.Fatal("failed draft disappeared on Release")
			}
			if err := v.RecoverDraft(); err != nil || !strings.Contains(recovered, "原字段存在: false") || !strings.HasSuffix(recovered, "必须保留🙂\r\n ") {
				t.Fatal("recovery omitted original presence or exact field", err, recovered)
			}
			if v.host.Buffer.String() != authority {
				t.Fatal("recovery wrote source")
			}
			if kind == "group" && !buffer.InGroup() {
				t.Fatal("foreign group stolen")
			}
		})
	}
}

func TestMindmapFieldLinkNativeCapturedTarget(t *testing.T) {
	for _, change := range []string{"href", "node", "buffer", "cancel", "outside", "pending-input", "pending-source", "pending-field"} {
		t.Run(change, func(t *testing.T) {
			source := fieldFixture(nil, stringPtr("original.txt"))
			buffer := text.New(source)
			var opened []string
			var reported error
			v := newView(kindMindmap, editor.Host{Buffer: buffer, OpenReference: func(s string) { opened = append(opened, s) }, Report: func(err error) { reported = err }})
			tester := ui.NewTester(v.Draw, 760, 480)
			label := "打开链接"
			if strings.HasPrefix(change, "pending-") {
				chooseField(t, tester, "字段：本地链接")
				composeField(t, tester, "节点链接（本地）", "new.txt")
				label = "写回并打开链接"
			}
			r, ok := tester.Find(label)
			if !ok {
				t.Fatal("missing link button")
			}
			x, y := r.X+r.W/2, r.Y+r.H/2
			tester.Press(x, y)
			if v.linkGesture.buffer == nil {
				t.Fatal("native press did not capture identity")
			}
			switch change {
			case "href", "pending-source":
				replaceSource(t, buffer, "original.txt", "changed.txt")
			case "node":
				v.selectNode("left")
			case "buffer":
				v.host.Buffer = text.New(source)
			case "cancel":
				tester.CancelPress()
			case "outside":
				x, y = 10, 10
			case "pending-input":
				_ = v.draft.input.Buffer.ReplaceUTF16(0, v.draft.input.Buffer.LenUTF16(), "unexpected.txt")
			case "pending-field":
				v.selectField(fieldNote)
			}
			tester.Frame()
			tester.Release(x, y)
			if len(opened) != 0 {
				t.Fatal("stale/canceled action silently retargeted", opened, reported)
			}
			if v.linkGesture.draft != nil {
				t.Fatal("completed/canceled action retained draft")
			}
			if change == "pending-source" && (!v.pendingDraft() || !errors.Is(v.draftError, ErrDraftConflict)) {
				t.Fatal("failed Flush lost local pending link", v.draftError)
			}
		})
	}
}

func TestMindmapFieldLinkKeyboardSourceRecheckAndLimits(t *testing.T) {
	for _, key := range []ui.Key{ui.KeyEnter, ui.KeySpace} {
		buffer := text.New(fieldFixture(nil, stringPtr("original.txt")))
		opened := 0
		var reported error
		v := newView(kindMindmap, editor.Host{Buffer: buffer, ReadOnly: true, OpenReference: func(string) { opened++ }, Report: func(err error) { reported = err }})
		armed := false
		tester := ui.NewTester(func(c *ui.Context) {
			ui.Column(c).Fill().ObserveInput(func(ev ui.InputEvent) {
				if armed && ev.Kind == ui.InputKeyDown && ev.Key == key {
					armed = false
					replaceSource(t, buffer, "original.txt", "other.txt")
				}
			}).Children(func() { v.Draw(c) })
		}, 760, 480)
		for i := 0; i < 20 && !tester.Focused("打开链接"); i++ {
			tester.Key(0, ui.KeyTab)
		}
		if !tester.Focused("打开链接") {
			t.Fatal("link not keyboard reachable")
		}
		armed = true
		tester.Key(0, key)
		if opened != 0 || !errors.Is(reported, errFieldLinkStale) {
			t.Fatal("key resolved post-event target", opened, reported)
		}
	}
	for _, count := range []int{4096, 4097, 30000} {
		href := strings.Repeat("x", count)
		buffer := text.New(fieldFixture(nil, &href))
		opened := ""
		v := newView(kindMindmap, editor.Host{Buffer: buffer, OpenReference: func(value string) { opened = value }})
		tester := ui.NewTester(v.Draw, 760, 480)
		access, ok := tester.AccessByRole(ui.RoleButton, "打开链接")
		if !ok || access.Disabled != (count > maxFieldLinkBytes) || len(access.Description) > 512+len("…") {
			t.Fatal("link budget/access metadata incorrect", count, access)
		}
		if count == 4096 {
			chooseField(t, tester, "打开链接")
			if opened != href {
				t.Fatal("complete destination altered")
			}
		} else if !tester.HasText("链接超过4096字节") || opened != "" || v.linkGesture.href != "" {
			t.Fatal("over-budget destination copied/opened silently")
		}
	}
}

func TestMindmapFieldTopicFocusAndDrawioIsolation(t *testing.T) {
	v := newView(kindMindmap, editor.Host{Buffer: text.New(fieldFixture(stringPtr("note"), nil))})
	tester := ui.NewTester(v.Draw, 760, 480)
	chooseField(t, tester, "字段：纯文本备注")
	chooseField(t, tester, "折叠选中主题")
	if v.field != fieldNote || v.draft.field != fieldNote {
		t.Fatal("collapse forced topic")
	}
	chooseField(t, tester, "添加子节点")
	if v.field != fieldTopic || v.draft.field != fieldTopic || !tester.Focused("节点文本") {
		t.Fatal("new topic focused previous note")
	}
	chooseField(t, tester, "字段：纯文本备注")
	// Native double click on a visible node must restore topic editing.
	found := false
	for _, box := range v.layout.Nodes {
		if box.ID == "root" {
			found = true
			x, y := v.worldToScreen(float32(box.X+box.Width/2), float32(box.Y+box.Height/2))
			tester.Advance(time.Second)
			tester.ClickAt(v.canvasRect.X+x, v.canvasRect.Y+y)
			tester.ClickAt(v.canvasRect.X+x, v.canvasRect.Y+y)
			break
		}
	}
	if !found || v.field != fieldTopic {
		t.Fatal("double click retained note editor")
	}
	draw := newView(kindDrawio, editor.Host{Buffer: text.New(testDrawio)})
	dt := ui.NewTester(draw.Draw, 760, 480)
	if dt.HasText("字段：纯文本备注") || draw.draft.field != fieldDrawioLabel {
		t.Fatal("mindmap fields leaked into drawio")
	}
	focusDraft(t, dt)
	dt.Compose("标签🙂", 3)
	if err := draw.Flush(); err != nil || !strings.Contains(draw.host.Buffer.String(), "标签🙂") {
		t.Fatal("drawio draft regressed", err)
	}
}

func TestMindmapFieldLinkOtherDraftFlushEmptyAndMissingHost(t *testing.T) {
	buffer := text.New(fieldFixture(nil, stringPtr("a.txt")))
	var opened []string
	var reported error
	v := newView(kindMindmap, editor.Host{Buffer: buffer, OpenReference: func(value string) { opened = append(opened, value) }, Report: func(err error) { reported = err }})
	tester := ui.NewTester(v.Draw, 760, 480)
	chooseField(t, tester, "字段：纯文本备注")
	composeField(t, tester, "节点备注（纯文本）", "其他字段🙂")
	chooseField(t, tester, "打开链接")
	if len(opened) != 1 || opened[0] != "a.txt" || parseMind(t, buffer).Root.Note == nil || *parseMind(t, buffer).Root.Note != "其他字段🙂" {
		t.Fatal("other field Compose was not flushed before exact open", opened, reported)
	}
	chooseField(t, tester, "字段：本地链接")
	composeField(t, tester, "节点链接（本地）", "")
	chooseField(t, tester, "写回并打开链接")
	if !errors.Is(reported, errFieldLinkMissing) || parseMind(t, buffer).Root.HyperLink != nil || len(opened) != 1 {
		t.Fatal("empty committed target opened stale link", opened, reported)
	}
	// Clear action is one real mutation; a clean refresh/Undo restores the
	// old field instead of reapplying the independent empty editor baseline.
	if !buffer.Undo() {
		t.Fatal("missing link clear undo")
	}
	tester.Frame()
	if v.draft.input.Buffer.String() != "a.txt" || !v.draft.sourcePresent {
		t.Fatal("clean field did not follow source Undo")
	}
	before := buffer.String()
	if err := v.Flush(); err != nil || buffer.String() != before {
		t.Fatal("clean Undo field rewrote source", err)
	}
	v.host.OpenReference = nil
	tester.Frame()
	info, ok := tester.AccessByRole(ui.RoleButton, "打开链接")
	if !ok || !info.Disabled || !tester.HasText("当前宿主不能打开本地引用") {
		t.Fatal("missing handler silently available")
	}
}

func TestMindmapLongFieldBoundedInspector(t *testing.T) {
	for _, readonly := range []bool{false, true} {
		buffer := text.New(fieldFixture(stringPtr(strings.Repeat("备注🙂", 1000)), stringPtr("target.txt")))
		v := newView(kindMindmap, editor.Host{Buffer: buffer, ReadOnly: readonly, OpenReference: func(string) {}, RecoverDraft: func(string, string) error { return nil }})
		tester := ui.NewTester(v.Draw, 532, 285)
		chooseField(t, tester, "字段：纯文本备注")
		if !readonly {
			revealFieldControl(t, tester, "节点备注（纯文本）", 532, 285)
			composeField(t, tester, "节点备注（纯文本）", strings.Repeat("草稿🙂", 800))
			doc := parseMind(t, buffer)
			if err := doc.SetNote("root", "external"); err != nil {
				t.Fatal(err)
			}
			data, _ := doc.Serialize(mindmap.SerializeOptions{})
			_ = buffer.ReplaceUTF16(0, buffer.LenUTF16(), string(data))
			if err := v.Flush(); !errors.Is(err, ErrDraftConflict) {
				t.Fatal(err)
			}
			tester.Frame()
		}
		for _, label := range []string{"节点备注（纯文本）", "下一段", "打开链接", "重试写回", "恢复为新文档"} {
			if readonly && (label == "重试写回" || label == "恢复为新文档") {
				continue
			}
			// Start each search at top; inspector scroll has its own bounded
			// clip, even when the long-field navigation wraps into two rows.
			revealFieldControl(t, tester, label, 532, 285)
		}
	}
}

func TestMindmapFieldInspectorImages(t *testing.T) {
	dir := os.Getenv("LEVNOTE_GRAPH_FIELDS_SCREENSHOT_DIR")
	if dir == "" {
		t.Skip("set LEVNOTE_GRAPH_FIELDS_SCREENSHOT_DIR")
	}
	for _, mode := range []string{"light", "dark", "readonly", "conflict"} {
		buffer := text.New(fieldFixture(stringPtr("纯文本备注🙂\r\n第二行保留空格  "), stringPtr("子目录/目标.txt")))
		v := newView(kindMindmap, editor.Host{Buffer: buffer, ReadOnly: mode == "readonly", OpenReference: func(string) {}, RecoverDraft: func(string, string) error { return nil }})
		tester := ui.NewTester(v.Draw, 760, 480)
		if mode != "light" {
			tester.SetDark(true)
		}
		chooseField(t, tester, "字段：纯文本备注")
		if mode == "conflict" {
			composeField(t, tester, "节点备注（纯文本）", "草稿仍可恢复🙂")
			replaceSource(t, buffer, `"note":"纯文本备注🙂\r\n第二行保留空格  "`, `"note":"外部改动"`)
			_ = v.Flush()
			tester.Frame()
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(filepath.Join(dir, "mindmap-fields-"+mode+"-760x480.png"))
		if err != nil {
			t.Fatal(err)
		}
		if err := png.Encode(f, tester.Image()); err != nil {
			t.Fatal(err)
		}
		_ = f.Close()
	}
}
