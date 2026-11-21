package app

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/code"
)

func TestRenamePaletteNativeNameAndAccessibleControls760KeyboardPointer(t *testing.T) {
	for _, pointer := range []bool{false, true} {
		a, tab, q, path := languageFixture(t, ".go", renameLifecycleSource)
		tester := ui.NewTester(a.View, 760, 480)
		tester.Frame()
		renamePosition(t, tab, renameLifecycleSource, strings.Index(renameLifecycleSource, "x int"))
		before := languageState(tab)
		paletteOpen(t, a, tester)
		tester.Type("语言 重命名本文件符号")
		if a.commandPalette.selectedID != "language.rename" {
			t.Fatal("real palette lacks rename command", a.commandPalette.selectedID)
		}
		tester.Key(0, ui.KeyEnter)
		tester.Frame()
		d := a.sourceLanguage.rename
		if a.commandPalette.open || d == nil || a.sourceLanguage.task != nil {
			t.Fatal("OPEN analyzed or failed to create native name field")
		}
		if d.field.Buffer != d.buffer || d.buffer == tab.Doc.Buffer || d.buffer.MaxBytes() != 512 || d.field.Label != "本文件符号新名称" || d.field.LineNumbers || d.field.ShapeLimit != 128 {
			t.Fatal("name field is not separate bounded project Editor")
		}
		if err := d.buffer.SetSelection(0, d.buffer.LenUTF16()); err != nil {
			t.Fatal(err)
		}
		d.field.Focus()
		tester.Frame()
		tester.Type("next")
		if d.buffer.String() != "next" || a.sourceLanguage.task != nil || q.outstanding.Load() != 0 {
			t.Fatal("native typing analyzed or did not reach field", d.buffer.String())
		}
		languageUnchanged(t, tab, before)
		if err := tester.Click("生成重命名预览"); err != nil {
			t.Fatal(err)
		}
		tester.Frame()
		renameFinish(t, a, q)
		tester.Frame()
		for _, label := range []string{"本文件符号新名称", "生成重命名预览", "重命名预览", "确认重命名", "取消重命名"} {
			access, ok := tester.Access(label)
			if !ok || access.Bounds.W <= 0 || access.Bounds.H <= 0 || access.Bounds.X < 0 || access.Bounds.Y < 0 || access.Bounds.X+access.Bounds.W > 760 || access.Bounds.Y+access.Bounds.H > 480 {
				t.Fatalf("accessible control %q missing/clipped: %+v", label, access)
			}
		}
		handle := a.sourceLanguage.list.Handle
		a.sourceLanguage.list.Handle.Focus()
		tester.Frame()
		tester.Key(0, ui.KeyDown)
		languageUnchanged(t, tab, before)
		if pointer {
			if err := tester.Click("确认重命名"); err != nil {
				t.Fatal(err)
			}
		} else {
			tester.Key(0, ui.KeyEnter)
		}
		tester.Frame()
		if tab.Doc.Buffer.String() != "package p; func f(next int){ _=next }" || a.sourceLanguage.visible || a.sourceLanguage.rename != nil || a.sourceLanguage.list.Handle != handle || string(mustReadFile(t, path)) != renameLifecycleSource {
			t.Fatal("public full confirmation or retained list handle", tab.Doc.Buffer.String())
		}
		if !tab.Doc.Buffer.Undo() || tab.Doc.Buffer.String() != renameLifecycleSource || !reflect.DeepEqual(tab.Doc.Buffer.SelectionSet(), before.selections) || !tab.Doc.Buffer.Redo() || tab.Doc.Buffer.String() != "package p; func f(next int){ _=next }" {
			t.Fatal("public confirmation whole Undo/Redo")
		}
	}
}

func TestRename1024VirtualRowsDoNotNavigateSourceOrLoseListHandle(t *testing.T) {
	source := "package p\nfunc f(){ x:=0\n" + strings.Repeat("_=x\n", 1023) + "}\n"
	a, tab, q, _ := languageFixture(t, ".go", source)
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	renamePosition(t, tab, source, strings.Index(source, "x:=0"))
	renameReady(t, a, q, "next")
	tester.Frame()
	rows := a.sourceLanguageRows()
	if len(rows) != 1024 || len(a.sourceLanguage.rename.patch) != 1024 {
		t.Fatal("full edit data truncated by virtual viewport")
	}
	before := languageState(tab)
	handle := a.sourceLanguage.list.Handle
	a.sourceLanguage.list.Handle.Focus()
	tester.Frame()
	tester.Key(0, ui.KeyDown)
	tester.Key(0, ui.KeyDown)
	a.sourceLanguage.list.ScrollIntoView(1023)
	tester.Frame()
	if _, ok := tester.Access("重命名项：" + rows[1023].label); !ok {
		t.Fatal("last of 1024 rows inaccessible through virtual list")
	}
	languageUnchanged(t, tab, before)
	if a.sourceLanguage.list.Handle != handle || len(a.sourceLanguage.rename.patch) != 1024 {
		t.Fatal("scrolling changed preview owner/complete membership")
	}
	// A field edit resets semantic rows at the next safe boundary while the
	// stable public list Handle survives all late layout callbacks.
	renameName(t, a, "other")
	tester.Frame()
	if a.sourceLanguage.list.Handle != handle || len(a.sourceLanguageRows()) != 0 || len(a.sourceLanguage.rename.patch) != 0 {
		t.Fatal("name edit retained stale list rows or replaced Handle")
	}
	renamePreview(t, a, q, "next")
	tester.Frame()
	if a.sourceLanguage.list.Handle != handle || len(a.sourceLanguageRows()) != 1024 {
		t.Fatal("new preview did not restore full membership")
	}
}

func TestRenameCancelButtonsAndSavedActionsCannotConsumeReplacement(t *testing.T) {
	a, tab, q, _ := languageFixture(t, ".go", renameLifecycleSource)
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	renamePosition(t, tab, renameLifecycleSource, strings.Index(renameLifecycleSource, "x int"))
	renameReady(t, a, q, "next")
	tester.Frame()
	a.queueSourceLanguageAction("rename-confirm", "")
	saved := *a.sourceLanguage.action
	a.sourceLanguage.action = nil // Keep the copied confirmation inert until the later replay.
	handle := a.sourceLanguage.list.Handle
	before := languageState(tab)
	if err := tester.Click("取消重命名"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if a.sourceLanguage.rename != nil || a.sourceLanguage.visible || a.sourceLanguage.owner != nil || len(a.sourceLanguage.result.Edits) != 0 || a.sourceLanguage.intent.rename != nil || a.sourceLanguage.list.Handle != handle {
		t.Fatal("actual cancel retained dialog/patch/name ownership")
	}
	languageUnchanged(t, tab, before)
	renameReady(t, a, q, "other")
	tester.Frame()
	d, request, presentation := a.sourceLanguage.rename, a.sourceLanguage.request, a.sourceLanguage.presentation
	a.sourceLanguage.action = &saved
	a.processSourceLanguageActions()
	if a.sourceLanguage.rename != d || a.sourceLanguage.request != request || a.sourceLanguage.presentation != presentation || len(d.patch) != 2 {
		t.Fatal("old action consumed replacement dialog")
	}
	a.sourceLanguage.list.Handle.Focus()
	tester.Frame()
	tester.Key(0, ui.KeyEscape)
	tester.Frame()
	if a.sourceLanguage.rename != nil || a.sourceLanguage.visible {
		t.Fatal("preview list Escape did not cancel")
	}
	languageUnchanged(t, tab, before)
}

func TestRenameSourceActualNativeMarklessMultiRefusesWithoutLosingEscape(t *testing.T) {
	for _, multi := range []bool{false, true} {
		a, tab, q, _ := languageFixture(t, ".go", renameLifecycleSource)
		b := text.NewWithLimit(renameLifecycleSource, len(renameLifecycleSource))
		tab.Doc.Buffer = b
		tab.Editor.SetBuffer(b)
		tester := ui.NewTester(a.View, 760, 480)
		tester.Frame()
		u := renameUnits(renameLifecycleSource[:strings.Index(renameLifecycleSource, "x int")])
		set := text.SelectionSet{Ranges: []text.Range{{Anchor: u, Head: u}}, Primary: 0}
		if multi {
			set = text.SelectionSet{Ranges: []text.Range{{Anchor: 0, Head: 0}, {Anchor: u, Head: u}}, Primary: 1}
		}
		if err := b.SetSelectionSet(set); err != nil {
			t.Fatal(err)
		}
		renameReady(t, a, q, "next")
		tester.Frame()
		a.queueSourceLanguageAction("rename-confirm", "")
		saved := *a.sourceLanguage.action
		a.sourceLanguage.action = nil // Hold this already-created action through native input.
		d, field := a.sourceLanguage.rename, a.sourceLanguage.rename.field
		// Programmatic focus keeps the admitted selection intact. Compose uses
		// the installed native client, including first-refused multi admission.
		tab.Editor.Focus()
		tester.Frame()
		if !tester.Focused(tab.Editor.Label) {
			t.Fatal("source native client not focused")
		}
		tester.Compose("🙂", 1)
		if tab.Editor.InputActionError() == nil {
			t.Fatal("fixture lacks retained first-refused native input")
		}
		if _, marked := tab.Editor.MarkedRange(); marked {
			t.Fatal("first-refused source unexpectedly has a mark")
		}
		if multi && !tab.Editor.MultiCompositionPending() {
			t.Fatal("actual native multi route was not exercised")
		}
		before := languageState(tab)
		a.sourceLanguage.action = &saved
		a.processSourceLanguageActions()
		if err := a.requestSourceRename(); err == nil {
			t.Fatal("generation admitted pending native source")
		}
		a.queueSourceLanguageAction("rename-cancel", "")
		a.processSourceLanguageActions()
		if a.sourceLanguage.rename != d || d.field != field || tab.Editor.InputActionError() == nil {
			t.Fatal("refusal retired/settled live native owner")
		}
		languageUnchanged(t, tab, before)
		tester.Key(0, ui.KeyEscape)
		if tab.Editor.InputActionError() != nil || b.String() != renameLifecycleSource || b.InGroup() {
			t.Fatal("source's installed native Escape route lost")
		}
		tester.Frame()
		if a.sourceLanguage.rename != nil || tab.Doc.Buffer.String() != renameLifecycleSource {
			t.Fatal("old source target revived after native cancellation")
		}
		if q.outstanding.Load() != 0 {
			t.Fatal("refusal analyzed")
		}
	}
}

func TestRenameNativeNamePendingPointerAndForeignGroupsPreserveOwner(t *testing.T) {
	for _, kind := range []string{"marked-name", "markless-name", "field-foreign-group", "field-foreign-composition", "source-foreign-group", "source-foreign-composition"} {
		t.Run(kind, func(t *testing.T) {
			a, tab, q, _ := languageFixture(t, ".go", renameLifecycleSource)
			tester := ui.NewTester(a.View, 760, 480)
			tester.Frame()
			renamePosition(t, tab, renameLifecycleSource, strings.Index(renameLifecycleSource, "x int"))
			renameReady(t, a, q, "next")
			tester.Frame()
			d := a.sourceLanguage.rename
			field := d.field
			b := d.buffer
			a.queueSourceLanguageAction("rename-confirm", "")
			saved := *a.sourceLanguage.action
			var token text.CompositionToken
			if strings.HasPrefix(kind, "source-") {
				b = tab.Doc.Buffer
			}
			switch kind {
			case "marked-name":
				field.SetMarkedText(&ui.TextInputRange{Start: 0, End: 0}, "中", ui.TextInputRange{})
			case "markless-name":
				field.SetMarkedText(&ui.TextInputRange{Start: 0, End: 0}, strings.Repeat("🙂", 129), ui.TextInputRange{})
				if _, marked := field.MarkedRange(); marked {
					t.Fatal("markless name fixture")
				}
			case "field-foreign-group", "source-foreign-group":
				if err := b.BeginGroup(); err != nil {
					t.Fatal(err)
				}
			default:
				var err error
				token, err = b.BeginCompositionGroup()
				if err != nil {
					t.Fatal(err)
				}
			}
			if a.sourceRenameInputError() == nil {
				t.Fatal("input refusal fixture")
			}
			before := languageState(tab)
			fieldBefore := d.buffer.String()
			fieldHistory := d.buffer.History()
			fieldSet := d.buffer.SelectionSet()
			a.sourceLanguage.action = &saved
			a.processSourceLanguageActions()
			if err := a.requestSourceRename(); err == nil {
				t.Fatal("native/foreign input admitted generation")
			}
			// Public pointer controls are KeepFocus and disabled while input is
			// pending; their event must not unmark or close the name field.
			if err := tester.Click("取消重命名"); err != nil {
				t.Fatal(err)
			}
			tester.Frame()
			if a.sourceLanguage.rename != d || d.field != field || a.sourceRenameInputError() == nil || d.buffer.String() != fieldBefore || d.buffer.History() != fieldHistory || !reflect.DeepEqual(d.buffer.SelectionSet(), fieldSet) {
				t.Fatal("refusal changed native/foreign owner")
			}
			languageUnchanged(t, tab, before)
			switch kind {
			case "marked-name", "markless-name":
				field.HandleInput(ui.InputEvent{Kind: ui.InputKeyDown, Key: ui.KeyEscape})
				if field.InputActionError() != nil {
					t.Fatal("field owned Escape lost")
				}
			case "field-foreign-group", "source-foreign-group":
				if err := b.ReplaceUTF16(0, 0, " "); err != nil || !b.CancelGroup() {
					t.Fatal("foreign ordinary continuation lost", err)
				}
			default:
				if !b.OwnsComposition(token) || b.ReplaceCompositionUTF16(token, 0, 0, " ") != nil || b.CancelCompositionGroup(token) != nil {
					t.Fatal("foreign composition token stolen")
				}
			}
		})
	}
}

func TestRenameReadonlyAndOtherNativeFieldsGateEvenNoOp(t *testing.T) {
	for _, kind := range []string{"document-open", "editor-open", "document-confirm", "editor-confirm", "name-readonly", "palette", "picker", "goto"} {
		a, tab, q, _ := languageFixture(t, ".go", renameLifecycleSource)
		renamePosition(t, tab, renameLifecycleSource, strings.Index(renameLifecycleSource, "x int"))
		if strings.HasSuffix(kind, "-open") {
			if kind == "document-open" {
				tab.Doc.ReadOnly = true
			} else {
				tab.Editor.SetReadOnly(true)
			}
			before := languageState(tab)
			if languageRenameCommandAvailable(a) || a.openSourceRename() == nil || a.sourceLanguage.task != nil {
				t.Fatal("readonly OPEN admitted")
			}
			languageUnchanged(t, tab, before)
			continue
		}
		renameReady(t, a, q, "next")
		d := a.sourceLanguage.rename
		a.queueSourceLanguageAction("rename-confirm", "")
		saved := *a.sourceLanguage.action
		switch kind {
		case "document-confirm":
			tab.Doc.ReadOnly = true
		case "editor-confirm":
			tab.Editor.SetReadOnly(true)
		case "name-readonly":
			d.field.SetReadOnly(true)
		case "palette":
			a.commandPalette.composing = true
		case "picker":
			a.openDocuments.composing = true
		case "goto":
			a.GoToLineComposing = true
		}
		before := languageState(tab)
		a.sourceLanguage.action = &saved
		a.processSourceLanguageActions()
		languageUnchanged(t, tab, before)
		if a.sourceLanguage.rename != d {
			t.Fatal("refusal discarded name input owner")
		}
	}
	// JSON never acquires a spelling-based rename fallback.
	a, tab, q, _ := languageFixture(t, ".json", "{\"x\":1}")
	before := languageState(tab)
	if languageRenameCommandAvailable(a) || a.openSourceRename() == nil || a.sourceLanguage.task != nil || q.outstanding.Load() != 0 {
		t.Fatal("JSON acquired local rename")
	}
	languageUnchanged(t, tab, before)
}

func TestRenameStandaloneHistoryAndGrowthRefusalsAreAtomic(t *testing.T) {
	for _, kind := range []string{"no-history", "tiny-history", "growth"} {
		a, tab, q, _ := languageFixture(t, ".go", renameLifecycleSource)
		if kind == "growth" {
			b := text.NewWithLimit(renameLifecycleSource, len(renameLifecycleSource))
			tab.Doc.Buffer = b
			tab.Editor.SetBuffer(b)
		}
		renamePosition(t, tab, renameLifecycleSource, strings.Index(renameLifecycleSource, "x int"))
		renameReady(t, a, q, "longerName")
		if kind == "no-history" {
			tab.Doc.Buffer.SetHistoryLimits(0, 0)
		}
		if kind == "tiny-history" {
			tab.Doc.Buffer.SetHistoryLimits(1, 1)
		}
		before := languageState(tab)
		calls := 0
		tab.Editor.OnChange = func() { calls++ }
		renameConfirm(t, a)
		languageUnchanged(t, tab, before)
		if calls != 0 || a.Message == "" || a.sourceLanguage.rename != nil || a.sourceLanguage.visible {
			t.Fatal("atomic B3 refusal hidden or changed host", kind, a.Message, calls)
		}
	}
}

func TestRenameActualServiceTextBudgetRetainsNoPrefix(t *testing.T) {
	old := strings.Repeat("x", 512)
	source := "package p\nfunc f(){ " + old + " := 0\n" + strings.Repeat("_="+old+"\n", 1023) + "}\n"
	a, tab, q, _ := languageFixture(t, ".go", source)
	renamePosition(t, tab, source, strings.Index(source, old))
	before := languageState(tab)
	if err := a.openSourceRename(); err != nil {
		t.Fatal(err)
	}
	if a.sourceLanguage.rename.buffer.String() != "" || a.sourceLanguage.message == "" {
		t.Fatal("oversize original name silently truncated")
	}
	renamePreview(t, a, q, strings.Repeat("n", 128))
	if a.sourceLanguage.result.Status != code.ServiceLimit || a.sourceLanguage.result.Resource != "text-bytes" || a.sourceLanguage.result.Limit != 512<<10 || len(a.sourceLanguage.result.Edits) != 0 || len(a.sourceLanguage.rename.patch) != 0 || len(a.sourceLanguageRows()) != 0 {
		t.Fatal("genuine service text budget hid a partial plan", a.sourceLanguage.result, a.sourceLanguage.failure)
	}
	a.queueSourceLanguageAction("rename-confirm", "")
	a.processSourceLanguageActions()
	languageUnchanged(t, tab, before)
}

func TestRenameSourceCoordinateSealAcrossPieceChunksAndUnicode(t *testing.T) {
	const source = "package p\r\n// 🙂前\nfunc f(变量 int){ _=变量 }"
	b := text.New(source)
	// Make several immutable pieces without changing the final source.
	for i := 0; i < 3; i++ {
		if b.ReplaceUTF16(0, 0, " ") != nil || !b.Undo() {
			t.Fatal("piece fixture")
		}
	}
	for _, tc := range []struct {
		byteAt   int
		selected bool
	}{
		{strings.Index(source, "变量"), false},
		{strings.Index(source, "变量") + len("变量"), false},
		{strings.Index(source, "变量") + len("变量"), true},
		{strings.Index(source, "🙂") + len("🙂"), false},
		{len(source), false},
	} {
		units := renameUnits(source[:tc.byteAt])
		want, err := sourceLanguageByteOffset(source, units, tc.selected)
		if err != nil {
			t.Fatal(err)
		}
		got, err := sourceRenameOriginalByte(b, units, tc.selected)
		if err != nil || got != want {
			t.Fatalf("streamed OPEN byte/UTF-16 seal at %d: %d vs %d, %v", tc.byteAt, got, want, err)
		}
	}
}

func TestRenamePreviewDoubleClickNeverConfirmsAndExplicitActionsApplyOnce(t *testing.T) {
	{
		a, tab, q, _ := languageFixture(t, ".go", renameLifecycleSource)
		tester := ui.NewTester(a.View, 760, 480)
		tester.Frame()
		renamePosition(t, tab, renameLifecycleSource, strings.Index(renameLifecycleSource, "x int"))
		renameReady(t, a, q, "next")
		tester.Frame()
		rows := a.sourceLanguageRows()
		if len(rows) == 0 {
			t.Fatal("missing real preview row")
		}
		access, ok := tester.Access("重命名项：" + rows[0].label)
		if !ok {
			t.Fatal("preview row inaccessible")
		}
		x, y := access.Bounds.X+access.Bounds.W/2, access.Bounds.Y+access.Bounds.H/2
		before := languageState(tab)
		tester.ClickAt(x, y)
		tester.ClickAt(x, y) // Same-row Tester clicks exercise MyGo's double-click submit path.
		tester.Frame()
		if a.sourceLanguage.action != nil || a.sourceLanguage.rename == nil || a.sourceLanguage.result.Status != code.ServiceComplete || len(a.sourceLanguage.rename.patch) != 2 {
			t.Fatal("preview-row double-click became a confirmation")
		}
		languageUnchanged(t, tab, before)
	}
	for _, trigger := range []string{"enter", "button"} {
		t.Run(trigger, func(t *testing.T) {
			a, tab, q, _ := languageFixture(t, ".go", renameLifecycleSource)
			tester := ui.NewTester(a.View, 760, 480)
			tester.Frame()
			renamePosition(t, tab, renameLifecycleSource, strings.Index(renameLifecycleSource, "x int"))
			renameReady(t, a, q, "next")
			tester.Frame()
			changes := 0
			oldChange := tab.Editor.OnChange
			tab.Editor.OnChange = func() {
				changes++
				if oldChange != nil {
					oldChange()
				}
			}
			if trigger == "enter" {
				a.sourceLanguage.list.Handle.Focus()
				tester.Frame()
				tester.Key(0, ui.KeyEnter)
			} else if err := tester.Click("确认重命名"); err != nil {
				t.Fatal(err)
			}
			tester.Frame()
			if tab.Doc.Buffer.String() != "package p; func f(next int){ _=next }" || changes != 1 || a.sourceLanguage.rename != nil || a.sourceLanguage.visible {
				t.Fatal("explicit confirmation did not apply exactly once", trigger, changes, tab.Doc.Buffer.String())
			}
		})
	}
}

func TestRenamePreviewPendingAndForeignInputDoubleClickKeepsFieldOwner(t *testing.T) {
	for _, kind := range []string{"pending-click-escape", "pending-double-final", "markless-double-escape", "foreign-group-double", "foreign-composition-double"} {
		t.Run(kind, func(t *testing.T) {
			a, tab, q, _ := languageFixture(t, ".go", renameLifecycleSource)
			tester := ui.NewTester(a.View, 760, 480)
			tester.Frame()
			renamePosition(t, tab, renameLifecycleSource, strings.Index(renameLifecycleSource, "x int"))
			renameReady(t, a, q, "next")
			tester.Frame()
			d := a.sourceLanguage.rename
			field, b := d.field, d.buffer
			rows := a.sourceLanguageRows()
			if len(rows) == 0 {
				t.Fatal("missing preview before input transition")
			}
			access, ok := tester.Access("重命名项：" + rows[0].label)
			if !ok {
				t.Fatal("preview row inaccessible")
			}
			x, y := access.Bounds.X+access.Bounds.W/2, access.Bounds.Y+access.Bounds.H/2
			isDouble := strings.Contains(kind, "double")
			if isDouble {
				tester.ClickAt(x, y)
				field.Focus()
				tester.Frame()
			}
			if !tester.Focused(field.Label) {
				t.Fatal("name field did not own focus before pending row input")
			}
			a.queueSourceRenameAction("rename-confirm")
			if a.sourceLanguage.action == nil {
				t.Fatal("missing saved pre-input confirmation")
			}
			saved := *a.sourceLanguage.action
			a.sourceLanguage.action = nil
			before := languageState(tab)
			base := b.String()
			selection := b.Selection()
			prefix, err := b.SliceUTF16(0, selection.Start())
			if err != nil {
				t.Fatal(err)
			}
			suffix, err := b.SliceUTF16(selection.End(), b.LenUTF16())
			if err != nil {
				t.Fatal(err)
			}
			var token text.CompositionToken
			switch kind {
			case "pending-click-escape", "pending-double-final":
				field.SetMarkedText(nil, "中", ui.TextInputRange{Start: 1, End: 1})
			case "markless-double-escape":
				field.SetMarkedText(nil, strings.Repeat("🙂", 129), ui.TextInputRange{})
				if _, marked := field.MarkedRange(); marked {
					t.Fatal("first-refused field input acquired a mark")
				}
			case "foreign-group-double":
				if err := b.BeginGroup(); err != nil {
					t.Fatal(err)
				}
			case "foreign-composition-double":
				var err error
				token, err = b.BeginCompositionGroup()
				if err != nil {
					t.Fatal(err)
				}
			}
			ownerErr := field.InputActionError()
			if ownerErr == nil {
				t.Fatal("pending/foreign name input not detected", kind)
			}
			mark, marked := field.MarkedRange()
			if strings.HasPrefix(kind, "pending-") && !marked || kind == "markless-double-escape" && marked || kind == "foreign-composition-double" && !b.OwnsComposition(token) {
				t.Fatal("composition owner state changed before list event", kind)
			}
			pendingValue, pendingVersion, pendingHistory, pendingSelection := b.String(), b.Version(), b.History(), b.SelectionSet()
			tester.ClickAt(x, y) // On a double path this is the second click, against the still-built row.
			if !tester.Focused(field.Label) || a.sourceLanguage.rename != d || d.field != field || d.buffer != b || field.Buffer != b || field.InputActionError() == nil {
				t.Fatal("preview row input stole focus or retired the native field owner", kind)
			}
			afterMark, afterMarked := field.MarkedRange()
			if b.String() != pendingValue || b.Version() != pendingVersion || b.History() != pendingHistory || !reflect.DeepEqual(b.SelectionSet(), pendingSelection) || marked != afterMarked || marked && mark != afterMark || !errors.Is(field.InputActionError(), ownerErr) {
				t.Fatal("row input changed pending field bytes, history, selection or composition owner", kind)
			}
			if strings.HasPrefix(kind, "pending-") && !b.InGroup() || kind == "foreign-group-double" && !b.InGroup() || kind == "foreign-composition-double" && !b.OwnsComposition(token) {
				t.Fatal("row input settled or released a pending/foreign group", kind)
			}
			a.sourceLanguage.action = &saved
			a.processSourceLanguageActions()
			languageUnchanged(t, tab, before)
			switch kind {
			case "pending-click-escape", "markless-double-escape":
				tester.Key(0, ui.KeyEscape)
				if field.InputActionError() != nil || b.String() != base {
					t.Fatal("name field's own Escape route was lost", kind, b.String())
				}
			case "pending-double-final":
				tester.Type("终")
				if field.InputActionError() != nil || b.String() != prefix+"终"+suffix {
					t.Fatal("name field's final committed text route was lost", b.String())
				}
			case "foreign-group-double":
				if err := b.ReplaceUTF16(b.LenUTF16(), b.LenUTF16(), "!"); err != nil || !b.CancelGroup() {
					t.Fatal("ordinary foreign input owner could not continue", err)
				}
				if b.String() != base {
					t.Fatal("ordinary owner cancellation did not restore name", b.String())
				}
			case "foreign-composition-double":
				span := b.Selection()
				if err := b.ReplaceCompositionUTF16(token, span.Start(), span.End(), "!"); err != nil || b.CancelCompositionGroup(token) != nil {
					t.Fatal("foreign composition owner could not continue")
				}
				if b.String() != base {
					t.Fatal("foreign composition cancellation did not restore name", b.String())
				}
			}
			if a.sourceLanguage.rename != d || d.request != nil || len(d.patch) != 0 || len(d.previewRows) != 0 {
				t.Fatal("old preview/action revived after name input transition")
			}
			languageUnchanged(t, tab, before)
		})
	}
}

func TestRenameNamePasteDeleteUndoRedoAreRealGesturesAndNotABA(t *testing.T) {
	a, tab, q, _ := languageFixture(t, ".go", renameLifecycleSource)
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	renamePosition(t, tab, renameLifecycleSource, strings.Index(renameLifecycleSource, "x int"))
	renameReady(t, a, q, "next")
	tester.Frame()
	d := a.sourceLanguage.rename
	field, b := d.field, d.buffer
	before := languageState(tab)
	save := func() (*sourceRenameRequest, sourceLanguageAction) {
		r := d.request
		a.queueSourceRenameAction("rename-confirm")
		if r == nil || a.sourceLanguage.action == nil || len(d.patch) != 2 {
			t.Fatal("missing fresh complete preview/action before gesture")
		}
		action := *a.sourceLanguage.action
		a.sourceLanguage.action = nil
		return r, action
	}
	regenerate := func() {
		if err := tester.Click("生成重命名预览"); err != nil {
			t.Fatal(err)
		}
		renameFinish(t, a, q)
		tester.Frame()
		if a.sourceLanguage.result.Status != code.ServiceComplete {
			t.Fatal("fresh preview failed after gesture", a.sourceLanguage.result)
		}
	}
	retired := func(want string, r *sourceRenameRequest, action sourceLanguageAction) {
		if b.String() != want || d.request != nil || len(d.patch) != 0 || len(d.previewRows) != 0 || a.sourceLanguage.action != nil {
			t.Fatal("gesture did not invalidate its preview/action", b.String(), want)
		}
		if err := a.sourceRenameRequestValid(r); err == nil {
			t.Fatal("gesture retained its old opaque preview identity")
		}
		a.sourceLanguage.action = &action
		a.processSourceLanguageActions()
		languageUnchanged(t, tab, before)
	}
	if err := tester.Click(field.Label); err != nil {
		t.Fatal(err)
	}
	original, originalAction := save()
	tester.Key(ui.Cmd, ui.KeyA)
	tester.SetClipboard("other")
	tester.Key(ui.Cmd, ui.KeyV)
	retired("other", original, originalAction)
	// Generating a preview does not edit the name or its undo/redo history.
	regenerate()
	undoRequest, undoAction := save()
	tester.Key(ui.Cmd, ui.KeyZ)
	retired("next", undoRequest, undoAction)
	if err := a.sourceRenameRequestValid(original); err == nil {
		t.Fatal("Undo revived the original same-spelling name target")
	}
	a.sourceLanguage.action = &originalAction
	a.processSourceLanguageActions()
	languageUnchanged(t, tab, before)
	regenerate()
	redoRequest, redoAction := save()
	tester.Key(ui.Cmd|ui.Shift, ui.KeyZ)
	retired("other", redoRequest, redoAction)
	regenerate()
	deleteRequest, deleteAction := save()
	tester.Key(ui.Cmd, ui.KeyA)
	tester.Key(0, ui.KeyBackspace)
	retired("", deleteRequest, deleteAction)
	renamePreview(t, a, q, "next")
	if a.sourceLanguage.rename != d || d.original == nil || len(d.patch) != 2 {
		t.Fatal("fresh explicit preview did not recover after gestures")
	}
}

func TestRenameName512BytesAcceptedByBufferThenRejectedAt128UTF16Units(t *testing.T) {
	a, tab, q, _ := languageFixture(t, ".go", renameLifecycleSource)
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	renamePosition(t, tab, renameLifecycleSource, strings.Index(renameLifecycleSource, "x int"))
	renameReady(t, a, q, "next")
	tester.Frame()
	d := a.sourceLanguage.rename
	field, b := d.field, d.buffer
	candidate := strings.Repeat("a", 512)
	sourceBefore := languageState(tab)
	if field.ShapeLimit != 128 || b.MaxBytes() != 512 {
		t.Fatal("name byte and shaping controls were not independently configured")
	}
	a.queueSourceRenameAction("rename-confirm")
	saved := *a.sourceLanguage.action
	a.sourceLanguage.action = nil
	var reported error
	oldError := field.OnError
	field.OnError = func(err error) {
		reported = err
		if oldError != nil {
			oldError(err)
		}
	}
	if err := tester.Click(field.Label); err != nil {
		t.Fatal(err)
	}
	tester.Key(ui.Cmd, ui.KeyA)
	tester.Type(candidate)
	if b.String() != candidate || len(b.String()) != 512 || renameUnits(b.String()) != 512 || field.InputActionError() != nil {
		t.Fatal("512 ASCII bytes were not fully admitted", len(b.String()), renameUnits(b.String()))
	}
	if reported == nil || !strings.Contains(reported.Error(), "超长行") {
		t.Fatal("128-unit ShapeLimit did not disclose its segmented-display warning", reported)
	}
	value, version, history, selection, snapshot := b.String(), b.Version(), b.History(), b.SelectionSet(), b.Snapshot()
	err := a.requestSourceRename()
	if err == nil || !strings.Contains(err.Error(), "128 UTF-16") {
		t.Fatal("512-byte/512-unit name bypassed the exact UTF-16 admission", err)
	}
	if !strings.Contains(a.sourceLanguage.message, "128 UTF-16") || !strings.Contains(a.sourceLanguage.message, "不会截断") {
		t.Fatal("128-unit refusal was not made visible without truncation", a.sourceLanguage.message)
	}
	if b.String() != value || b.Version() != version || b.History() != history || !reflect.DeepEqual(b.SelectionSet(), selection) || !b.EqualSnapshot(snapshot) || !tester.Focused(field.Label) || d.field != field || d.buffer != b || field.Buffer != b {
		t.Fatal("128-unit refusal cleared or moved the name field/owner/undo/selection")
	}
	if a.sourceLanguage.task != nil || q.outstanding.Load() != 0 || d.request != nil || len(d.patch) != 0 || len(d.previewRows) != 0 || a.sourceLanguage.action != nil {
		t.Fatal("over-128-unit name produced an Analyzer/preview/action")
	}
	a.sourceLanguage.action = &saved
	a.processSourceLanguageActions()
	languageUnchanged(t, tab, sourceBefore)
}

func TestRenameName513BytePasteIsAtomicAndRetainsSamePreview(t *testing.T) {
	a, tab, q, path := languageFixture(t, ".go", renameLifecycleSource)
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	renamePosition(t, tab, renameLifecycleSource, strings.Index(renameLifecycleSource, "x int"))
	renameReady(t, a, q, "next")
	tester.Frame()
	d := a.sourceLanguage.rename
	field, b := d.field, d.buffer
	oldRequest := d.request
	beforeValue := b.String()
	sourceBefore := languageState(tab)
	var reported error
	oldError := field.OnError
	field.OnError = func(err error) {
		reported = err
		if oldError != nil {
			oldError(err)
		}
	}
	if err := tester.Click(field.Label); err != nil {
		t.Fatal(err)
	}
	if err := b.SetSelection(b.LenUTF16(), 0); err != nil {
		t.Fatal(err)
	} // Preserve a backward selection through the refused replacement.
	version, history, selection, snapshot := b.Version(), b.History(), b.SelectionSet(), b.Snapshot()
	tester.SetClipboard(strings.Repeat("a", 513))
	tester.Key(ui.Cmd, ui.KeyV)
	var limitError *text.SizeLimitError
	if reported == nil || !errors.Is(reported, text.ErrSizeLimit) || !errors.As(reported, &limitError) || limitError.Limit != 512 {
		t.Fatal("513-byte native paste did not report the exact Buffer admission error", reported)
	}
	if !strings.Contains(a.sourceLanguage.message, "名称输入未完成") || !strings.Contains(a.sourceLanguage.message, "512 字节") {
		t.Fatal("rejected paste did not show its actionable error", a.sourceLanguage.message)
	}
	if b.String() != beforeValue || b.Version() != version || b.History() != history || !reflect.DeepEqual(b.SelectionSet(), selection) || !b.EqualSnapshot(snapshot) {
		t.Fatal("513-byte paste partially changed field bytes, version, history or backward selection")
	}
	if !tester.Focused(field.Label) || d.field != field || d.buffer != b || field.Buffer != b || d.request != oldRequest || a.sourceLanguage.result.Status != code.ServiceComplete || len(d.patch) != 2 || len(d.previewRows) == 0 || a.sourceLanguage.action != nil {
		t.Fatal("unchanged name target did not retain the same valid preview/field owner")
	}
	if a.sourceLanguage.task != nil || q.outstanding.Load() != 0 {
		t.Fatal("513-byte refusal launched service work")
	}
	languageUnchanged(t, tab, sourceBefore)
	// This direct callback is a same-generation action-retention control; the
	// paste above and the explicit confirmation below are actual Tester gestures.
	a.queueSourceRenameAction("rename-confirm")
	action := a.sourceLanguage.action
	if action == nil {
		t.Fatal("unchanged preview could not queue its explicit confirmation")
	}
	field.OnError(reported)
	if a.sourceLanguage.action != action || d.request != oldRequest || len(d.patch) != 2 {
		t.Fatal("same valid request/action was retired by an unchanged-input refusal")
	}
	staleAction := *action
	staleAction.renameEpoch++
	a.sourceLanguage.action = &staleAction
	field.OnError(reported)
	if a.sourceLanguage.action != nil || d.request != oldRequest || len(d.patch) != 2 {
		t.Fatal("mismatched action survived refusal or incorrectly retired the still-valid preview")
	}
	changes := 0
	oldChange := tab.Editor.OnChange
	tab.Editor.OnChange = func() {
		changes++
		if oldChange != nil {
			oldChange()
		}
	}
	if err := tester.Click("确认重命名"); err != nil {
		t.Fatal(err)
	}
	tester.Frame()
	if changes != 1 || tab.Doc.Buffer.String() != "package p; func f(next int){ _=next }" || !tab.Doc.Dirty() || string(mustReadFile(t, path)) != renameLifecycleSource || a.sourceLanguage.rename != nil {
		t.Fatal("preserved preview did not confirm exactly once", changes, tab.Doc.Buffer.String())
	}
}

func TestRenameInputErrorCannotPreserveNameTargetABA(t *testing.T) {
	a, tab, q, _ := languageFixture(t, ".go", renameLifecycleSource)
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	renamePosition(t, tab, renameLifecycleSource, strings.Index(renameLifecycleSource, "x int"))
	renameReady(t, a, q, "next")
	tester.Frame()
	d := a.sourceLanguage.rename
	field, b := d.field, d.buffer
	oldRequest := d.request
	before := languageState(tab)
	if err := tester.Click(field.Label); err != nil {
		t.Fatal(err)
	}
	a.queueSourceRenameAction("rename-confirm")
	if a.sourceLanguage.action == nil {
		t.Fatal("missing valid same-generation action")
	}
	saved := *a.sourceLanguage.action
	oldChange := field.OnChange
	field.OnChange = nil
	oldVersion := b.Version()
	field.ReplaceText(&ui.TextInputRange{Start: 0, End: b.LenUTF16()}, "other")
	field.ReplaceText(&ui.TextInputRange{Start: 0, End: b.LenUTF16()}, "next")
	field.OnChange = oldChange
	if b.String() != "next" || b.Version() == oldVersion {
		t.Fatal("A-to-B-to-A fixture did not restore bytes with a newer target version")
	}
	if err := a.sourceRenameRequestValid(oldRequest); err == nil {
		t.Fatal("A-to-B-to-A name retained the old opaque target")
	}
	version, history, selection, snapshot := b.Version(), b.History(), b.SelectionSet(), b.Snapshot()
	field.OnError(&text.SizeLimitError{Limit: 512})
	if !strings.Contains(a.sourceLanguage.message, "名称输入未完成") || d.request != nil || len(d.patch) != 0 || len(d.previewRows) != 0 || a.sourceLanguage.action != nil {
		t.Fatal("stale name target ABA incorrectly preserved preview/action")
	}
	if b.String() != "next" || b.Version() != version || b.History() != history || !reflect.DeepEqual(b.SelectionSet(), selection) || !b.EqualSnapshot(snapshot) || !tester.Focused(field.Label) || d.field != field || d.buffer != b || field.Buffer != b {
		t.Fatal("stale-target refusal cleared or moved its field/owner/history/selection")
	}
	a.sourceLanguage.action = &saved
	a.processSourceLanguageActions()
	languageUnchanged(t, tab, before)
}

func TestRenameInputErrorCannotPreserveSourceSelectionGenerationABA(t *testing.T) {
	a, tab, q, _ := languageFixture(t, ".go", renameLifecycleSource)
	tester := ui.NewTester(a.View, 760, 480)
	tester.Frame()
	renamePosition(t, tab, renameLifecycleSource, strings.Index(renameLifecycleSource, "x int"))
	renameReady(t, a, q, "next")
	tester.Frame()
	d := a.sourceLanguage.rename
	field, nameBuffer := d.field, d.buffer
	oldRequest := d.request
	if err := tester.Click(field.Label); err != nil {
		t.Fatal(err)
	}
	a.queueSourceRenameAction("rename-confirm")
	if a.sourceLanguage.action == nil {
		t.Fatal("missing valid same-generation action")
	}
	saved := *a.sourceLanguage.action
	original := tab.Doc.Buffer.SelectionSet()
	originalGeneration := tab.Doc.Buffer.SelectionGeneration()
	r := original.Ranges[original.Primary]
	alternate := text.SelectionSet{Ranges: []text.Range{{Anchor: r.Anchor + 1, Head: r.Head + 1}}, Primary: original.Primary}
	if err := tab.Doc.Buffer.SetSelectionSet(alternate); err != nil {
		t.Fatal(err)
	}
	if err := tab.Doc.Buffer.SetSelectionSet(original); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(tab.Doc.Buffer.SelectionSet(), original) || tab.Doc.Buffer.SelectionGeneration() == originalGeneration {
		t.Fatal("source selection ABA fixture did not restore values with a new generation")
	}
	if err := a.sourceRenameRequestValid(oldRequest); err == nil {
		t.Fatal("source selection-generation ABA retained the old opaque target")
	}
	stateAfterABA := languageState(tab)
	field.OnError(&text.SizeLimitError{Limit: 512})
	if !strings.Contains(a.sourceLanguage.message, "名称输入未完成") || d.request != nil || len(d.patch) != 0 || len(d.previewRows) != 0 || a.sourceLanguage.action != nil {
		t.Fatal("source selection-generation ABA incorrectly preserved preview/action")
	}
	if d.field != field || d.buffer != nameBuffer || field.Buffer != nameBuffer || !tester.Focused(field.Label) {
		t.Fatal("stale source-target refusal cleared or moved the name field/owner")
	}
	languageUnchanged(t, tab, stateAfterABA)
	a.sourceLanguage.action = &saved
	a.processSourceLanguageActions()
	languageUnchanged(t, tab, stateAfterABA)
	if err := a.sourceRenameRequestValid(oldRequest); err == nil {
		t.Fatal("source selection-generation ABA retained the old opaque request")
	}
}

func renameGoBindingWorkFixture(height, occurrences int) string {
	var source strings.Builder
	source.WriteString("package p\nimport u \"unsafe\"\nfunc f(x/*outer*/ int) {\n{\nx := 1\n_ = x\n")
	for i := 0; i < height; i++ {
		source.WriteString("const (\n_ = u.Sizeof(func() {\n{\n")
	}
	source.WriteString("_ = []int{")
	source.WriteString(strings.Repeat("x,", occurrences))
	source.WriteString("}\n")
	for i := 0; i < height; i++ {
		source.WriteString("}\n})\n_\n)\n")
	}
	source.WriteString("}\n}\n")
	return source.String()
}

func TestRenameActualServiceGoBindingWorkBudgetRetainsNoPrefix(t *testing.T) {
	source := renameGoBindingWorkFixture(32, 1200)
	a, tab, q, _ := languageFixture(t, ".go", source)
	renamePosition(t, tab, source, strings.Index(source, "x/*outer*/"))
	before := languageState(tab)
	if err := a.openSourceRename(); err != nil {
		t.Fatal(err)
	}
	renamePreview(t, a, q, "renamed")
	d := a.sourceLanguage.rename
	if a.sourceLanguage.result.Status != code.ServiceLimit || a.sourceLanguage.result.Resource != "go-binding-work" || a.sourceLanguage.result.Limit != 1<<20 {
		t.Fatal("real default ServiceRename did not exhaust go-binding-work", a.sourceLanguage.result)
	}
	if len(a.sourceLanguage.result.Edits) != 0 || len(d.patch) != 0 || len(d.previewRows) != 0 || len(a.sourceLanguageRows()) != 0 {
		t.Fatal("go-binding-work failure retained a partial preview")
	}
	a.queueSourceRenameAction("rename-confirm")
	if a.sourceLanguage.action != nil {
		t.Fatal("failed real service result queued a confirmation")
	}
	a.processSourceLanguageActions()
	if a.sourceLanguage.task != nil || a.sourceLanguage.pending != nil || q.outstanding.Load() != 0 {
		t.Fatal("go-binding-work refusal left work outstanding")
	}
	languageUnchanged(t, tab, before)
}
