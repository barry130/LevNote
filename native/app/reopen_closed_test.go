package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/session"
)

func reopenFixture(t *testing.T, a *App, path string, content []byte) *Tab {
	t.Helper()
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	a.Open(path)
	tab := a.Current()
	if tab == nil || !strings.EqualFold(tab.path(), canonical(path)) {
		t.Fatalf("open fixture: tabs=%d message=%s", len(a.Tabs), a.Message)
	}
	if !a.CloseTab(a.Active) {
		t.Fatalf("close fixture: %s", a.Message)
	}
	return tab
}

func reopenDispatch(t *testing.T, queued <-chan func()) func() {
	t.Helper()
	select {
	case fn := <-queued:
		return fn
	case <-time.After(3 * time.Second):
		t.Fatal("open did not dispatch its next UI phase")
		return nil
	}
}

func reopenDone(t *testing.T, done <-chan struct{}) {
	t.Helper()
	if done == nil {
		t.Fatal("reopen was not started")
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("open queue did not finish")
	}
}

func reopenPump(t *testing.T, a *App, queued <-chan func(), done <-chan struct{}) {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case fn := <-queued:
			fn()
		case <-done:
			a.reapReopen()
			return
		case <-deadline:
			t.Fatal("open queue did not finish while delivering UI callbacks")
		}
	}
}

func TestReopenClosedReadsCurrentDiskAndPreservesBytes(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		data                   []byte
		text                   string
		inspector, reinterpret bool
	}{
		{"utf8.txt", []byte("replacement\r\n"), "replacement\r\n", false, false},
		{"bom.txt", append([]byte{0xef, 0xbb, 0xbf}, []byte("BOM\r\n")...), "BOM\r\n", false, false},
		{"utf16.txt", []byte{0xff, 0xfe, 'h', 0, 'i', 0, 13, 0, 10, 0}, "hi\r\n", false, false},
		{"bytes.bin", []byte{0, 1, 2, 3}, "", true, false},
		{"legacy.txt", []byte{0xc4, 0xe3, 0xba, 0xc3}, "", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			a := newTestApp(t, filepath.Join(dir, "config"))
			path := filepath.Join(dir, tc.name)
			old := reopenFixture(t, a, path, []byte("previous disk"))
			if err := os.WriteFile(path, tc.data, 0600); err != nil {
				t.Fatal(err)
			}
			reopenDone(t, a.reopenClosedFile(0))
			current := a.Current()
			if current == nil || current == old || len(a.closedFiles) != 0 || current.Doc.Dirty() ||
				current.ByteInspector != tc.inspector || current.CanReinterpret != tc.reinterpret {
				t.Fatalf("not adopted from fresh bytes: current=%p old=%p history=%v message=%s", current, old, a.closedFiles, a.Message)
			}
			if !tc.inspector && current.Doc.Buffer.String() != tc.text {
				t.Fatalf("text=%q want=%q", current.Doc.Buffer.String(), tc.text)
			}
			actual, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(actual, tc.data) || !strings.Contains(a.Message, "按当前磁盘内容重新打开") {
				t.Fatalf("bytes or outcome changed: err=%v message=%s", err, a.Message)
			}
		})
	}
}

func TestReopenClosedDoesNotRememberLegacyEncoding(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "legacy.txt")
	data := []byte{0xc4, 0xe3, 0xba, 0xc3}
	a := newTestApp(t, filepath.Join(dir, "config"))
	reopenFixture(t, a, path, data)
	reopenDone(t, a.reopenClosedFile(0))
	if !a.reopenWithEncoding(a.Current(), document.GBK) || a.Current().Doc.Buffer.String() != "你好" {
		t.Fatal("explicit encoding fixture failed")
	}
	if !a.CloseTab(a.Active) {
		t.Fatal(a.Message)
	}
	reopenDone(t, a.reopenClosedFile(0))
	if !a.Current().ByteInspector || !a.Current().CanReinterpret || !strings.Contains(a.Current().Warning, "明确选择编码") {
		t.Fatal("reopen silently reused old code page")
	}
}

func TestReopenClosedSaveAsDiscardAndNoDraftAuthority(t *testing.T) {
	for _, decision := range []closeDecision{closeSave, closeDiscard} {
		t.Run(fmt.Sprint(decision), func(t *testing.T) {
			dir := t.TempDir()
			a := newTestApp(t, filepath.Join(dir, "config"))
			path := filepath.Join(dir, "saved.txt")
			a.NewDocument("text")
			tab := a.Current()
			tab.Editor.ReplaceText(nil, "new saved content")
			a.closeDialog = func(*Tab) (closeDecision, error) { return decision, nil }
			a.saveDialog = func(*Tab) (string, error) { return path, nil }
			if !a.CloseTab(0) {
				t.Fatal(a.Message)
			}
			if decision == closeSave {
				if len(a.closedFiles) != 1 || a.closedFiles[0].path != canonical(path) {
					t.Fatal("SaveAs path was not captured after dialog")
				}
				reopenDone(t, a.reopenClosedFile(0))
				a.Current().Editor.ReplaceText(nil, "discarded ")
				a.closeDialog = func(*Tab) (closeDecision, error) { return closeDiscard, nil }
				if !a.CloseTab(0) {
					t.Fatal(a.Message)
				}
				reopenDone(t, a.reopenClosedFile(0))
				if a.Current().Doc.Buffer.String() != "new saved content" {
					t.Fatal("discarded changes were resurrected")
				}
			} else if len(a.closedFiles) != 0 {
				t.Fatal("unsaved draft entered history")
			}
			if err := a.recoverDraft("draft.txt", "recovered only"); err != nil {
				t.Fatal(err)
			}
			a.Current().OriginalPath = path
			a.closeDialog = func(*Tab) (closeDecision, error) { return closeDiscard, nil }
			a.Message = "ordinary close"
			if !a.CloseTab(a.Active) || len(a.closedFiles) != 0 || a.Message != "ordinary close" {
				t.Fatal("OriginalPath used as saved authority or unsaved close warned")
			}
		})
	}
}

func TestReopenClosedFailedCloseNeverRecords(t *testing.T) {
	for _, reason := range []string{"cancel", "flush", "save", "persistence", "slot-reentry", "shutdown-reentry"} {
		t.Run(reason, func(t *testing.T) {
			dir := t.TempDir()
			a := newTestApp(t, filepath.Join(dir, "config"))
			path := filepath.Join(dir, "saved.txt")
			if err := os.WriteFile(path, []byte("disk"), 0600); err != nil {
				t.Fatal(err)
			}
			a.Open(path)
			tab := a.Current()
			tab.Editor.ReplaceText(nil, "dirty ")
			a.NewDocument("text")
			other := a.Current()
			a.closeDialog = func(*Tab) (closeDecision, error) { return closeCancel, nil }
			switch reason {
			case "flush":
				tab.View = &failingView{flushErr: errors.New("unsynced draft")}
			case "save":
				tab.Doc.ReadOnly = true
				a.closeDialog = func(*Tab) (closeDecision, error) { return closeSave, nil }
				a.saveDialog = func(*Tab) (string, error) { return "", errors.New("dialog failed") }
			case "persistence":
				a.closeDialog = func(*Tab) (closeDecision, error) { return closeDiscard, nil }
				if err := os.MkdirAll(filepath.Join(a.configDir, "session.json"), 0700); err != nil {
					t.Fatal(err)
				}
			case "slot-reentry":
				a.closeDialog = func(*Tab) (closeDecision, error) {
					a.Tabs[0], a.Tabs[1] = a.Tabs[1], a.Tabs[0]
					a.Active = 0
					return closeDiscard, nil
				}
			case "shutdown-reentry":
				a.closeDialog = func(*Tab) (closeDecision, error) {
					a.Shutdown()
					return closeDiscard, nil
				}
			}
			if a.CloseTab(0) || len(a.closedFiles) != 0 || !a.tabIsOpen(tab) || !a.tabIsOpen(other) || len(a.Tabs) != 2 {
				t.Fatal("unsuccessful close removed target/other or recorded history")
			}
			if tab.Doc.Buffer.String() != "dirty disk" {
				t.Fatal("failed close lost draft")
			}
		})
	}
}

func TestReopenClosedWindowCloseCancelsRunningImage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "image.png")
	imageFixture(t, path, "PNG")
	a := newTestApp(t, filepath.Join(dir, "config"))
	a.Open(path)
	if !a.CloseTab(0) {
		t.Fatal(a.Message)
	}
	id := a.closedFiles[0].id
	a.binaryLane.close()
	started := make(chan struct{})
	a.binaryLane = newBinaryLane(func(ctx context.Context, r binaryRequest) binaryOutput {
		close(started)
		<-ctx.Done()
		return binaryOutput{err: ctx.Err()}
	})
	queued := make(chan func(), 16)
	a.uiDispatch = func(fn func()) { queued <- fn }
	done := a.reopenClosedFile(id)
	r := a.reopening
	reopenDispatch(t, queued)()
	reopenDispatch(t, queued)()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("binary read did not start")
	}
	if a.CanClose() || !strings.Contains(a.Message, "正在取消图片任务") {
		t.Fatal("window close did not cancel the live task and keep outcome visible")
	}
	reopenPump(t, a, queued, done)
	if a.Current() != nil || a.binaryJob != nil || len(a.closedFiles) != 1 ||
		a.closedFiles[0].id != id || r.outcome.Load() != reopenRefused || !strings.Contains(a.Message, "记录仍保留") {
		t.Fatalf("canceled binary lost path or opened: %s", a.Message)
	}
	if !a.CanClose() || len(a.closedFiles) != 1 {
		t.Fatal("confirmed window close populated or removed recent-close history")
	}
}

func TestReopenClosedBoundDedupAndShutdownClearBacking(t *testing.T) {
	dir := t.TempDir()
	a := newTestApp(t, filepath.Join(dir, "config"))
	for i := 0; i < 19; i++ {
		reopenFixture(t, a, filepath.Join(dir, fmt.Sprintf("saved-%02d.txt", i)), []byte("disk"))
	}
	if len(a.closedFiles) != 16 || filepath.Base(a.closedFiles[15].path) != "saved-03.txt" {
		t.Fatalf("wrong bounded order: %v", a.closedFiles)
	}
	prior := a.closedFiles[8]
	a.Open(strings.ToUpper(prior.path))
	if !a.CloseTab(a.Active) || len(a.closedFiles) != 16 || a.closedFiles[0].id == prior.id ||
		!strings.EqualFold(a.closedFiles[0].path, prior.path) {
		t.Fatal("reclose did not dedup with a new ID")
	}
	if a.reopenClosedFile(prior.id) != nil || a.Current() != nil {
		t.Fatal("stale ID reopened replacement")
	}
	backing := a.closedFiles[:cap(a.closedFiles)]
	reopenDone(t, a.reopenClosedFile(a.closedFiles[7].id))
	for _, entry := range backing[len(a.closedFiles):] {
		if entry != (closedFile{}) {
			t.Fatal("removed path retained in backing tail")
		}
	}
	if !a.CanClose() || len(a.closedFiles) != 15 {
		t.Fatal("whole-window close recorded entries")
	}
	a.Shutdown()
	if a.closedFiles != nil || a.reopening != nil {
		t.Fatal("shutdown retained history")
	}
	for _, entry := range backing {
		if entry != (closedFile{}) {
			t.Fatal("shutdown retained backing path")
		}
	}
}

func TestReopenClosedUnusualPathsAndBoundedLabels(t *testing.T) {
	for _, path := range []string{`\\server\share\a.txt`, "relative.txt", "https://example.invalid/file", `C:\` + strings.Repeat("界", closedPathLimit/3+1)} {
		a := newTestApp(t, t.TempDir())
		d := document.New("disk")
		d.Path, d.IsNew = path, false
		tab := a.add(d, "unusual", "")
		if !a.CloseTab(0) || len(a.closedFiles) != 0 || !strings.Contains(a.Message, "未加入最近关闭") {
			t.Fatalf("unusual path not truthfully omitted: %s", a.Message)
		}
		assertTabAbsentFromBacking(t, a, tab)
	}
	value := strings.Repeat("界🙂", 1000) + "\n\t"
	for _, limit := range []int{128, 240, 2048} {
		label := boundedClosedLabel(value, limit)
		if len(label) > limit || !utf8.ValidString(label) || strings.ContainsAny(label, "\n\t") {
			t.Fatal("menu label not bounded")
		}
	}
}

func TestReopenClosedMissingTopRetainedMenuOpensOlder(t *testing.T) {
	dir := t.TempDir()
	a := newTestApp(t, filepath.Join(dir, "config"))
	older := filepath.Join(dir, "older.txt")
	missing := filepath.Join(dir, "missing.txt")
	reopenFixture(t, a, older, []byte("older disk"))
	reopenFixture(t, a, missing, []byte("missing disk"))
	missingID := a.closedFiles[0].id
	if err := os.Remove(missing); err != nil {
		t.Fatal(err)
	}
	tester := ui.NewTester(a.View, 760, 480)
	tester.Key(ui.Cmd|ui.Shift, ui.KeyT)
	if a.Current() != nil || len(a.closedFiles) != 2 || a.closedFiles[0].id != missingID || !strings.Contains(a.Message, "记录仍保留") {
		t.Fatalf("missing top popped/skipped or lacked retry: %s", a.Message)
	}
	if err := tester.Click("最近关闭的文件"); err != nil {
		t.Fatal(err)
	}
	if err := tester.ChooseMenuItem("2. older.txt"); err != nil {
		t.Fatal(err)
	}
	if a.Current() == nil || a.Current().path() != canonical(older) || len(a.closedFiles) != 1 || a.closedFiles[0].id != missingID {
		t.Fatal("menu failed to adopt exact older entry")
	}
	tester.KeyRepeat(ui.Cmd|ui.Shift, ui.KeyT)
	if len(a.Tabs) != 1 || len(a.closedFiles) != 1 {
		t.Fatal("repeated key consumed another history entry")
	}
}

func TestReopenClosedSyntheticSourceAndRichComposition(t *testing.T) {
	for _, rich := range []bool{false, true} {
		t.Run(fmt.Sprint(rich), func(t *testing.T) {
			dir := t.TempDir()
			a := newTestApp(t, filepath.Join(dir, "config"))
			reopenFixture(t, a, filepath.Join(dir, "target.txt"), []byte("target"))
			if rich {
				path := filepath.Join(dir, "source.md")
				if err := os.WriteFile(path, []byte("# 标题\n\nbody\n"), 0600); err != nil {
					t.Fatal(err)
				}
				a.Open(path)
			} else {
				a.NewDocument("text")
			}
			origin := a.Current()
			tester := ui.NewTester(a.View, 760, 480)
			if rich {
				beginRealMarkdownComposition(t, origin, tester)
			} else {
				if err := tester.Click("文档编辑区"); err != nil {
					t.Fatal(err)
				}
				tester.Compose("中文🙂", 3)
			}
			tester.Key(ui.Cmd|ui.Shift, ui.KeyT)
			if a.Current() == origin || len(a.closedFiles) != 0 || !strings.Contains(origin.Doc.Buffer.String(), "中文🙂") {
				t.Fatal("reopen bypassed source/rich Flush")
			}
		})
	}
}

func TestReopenClosedUtilityPreeditBlocksKeyAndMenuFocus(t *testing.T) {
	for _, panel := range []string{"query", "replacement", "goto"} {
		t.Run(panel, func(t *testing.T) {
			dir := t.TempDir()
			a := newTestApp(t, filepath.Join(dir, "config"))
			reopenFixture(t, a, filepath.Join(dir, "target.txt"), []byte("target"))
			a.NewDocument("text")
			origin := a.Current()
			tester := ui.NewTester(a.View, 760, 480)
			label := "查找内容"
			if panel == "goto" {
				tester.Key(ui.Cmd, ui.KeyG)
				label = "转到行号"
				tester.Compose("12", 2)
			} else {
				tester.Key(ui.Cmd, ui.KeyF)
				if panel == "replacement" {
					label = "替换内容"
					if err := tester.Click(label); err != nil {
						t.Fatal(err)
					}
				}
				tester.Compose("组合", 2)
			}
			caret, ok := tester.TextCaret()
			if !ok || !tester.Focused(label) {
				t.Fatal("missing preedit fixture")
			}
			tester.Key(ui.Cmd|ui.Shift, ui.KeyT)
			rect, ok := tester.Find("最近关闭的文件")
			if !ok {
				t.Fatal("missing menu")
			}
			tester.ClickAt(rect.X+rect.W/2, rect.Y+rect.H/2)
			after, active := tester.TextCaret()
			if a.Current() != origin || len(a.closedFiles) != 1 || a.reopening != nil || len(tester.Menu()) != 0 ||
				!tester.Focused(label) || !active || after != caret || !strings.Contains(a.Message, "请先完成组合") {
				t.Fatalf("menu/key lost utility preedit: focused=%v caret=%v/%v message=%s", tester.Focused(label), after, caret, a.Message)
			}
			if err := tester.Click("文档编辑区"); err != nil {
				t.Fatal(err)
			}
			tester.Frame()
			if a.findCompositionActive() || a.GoToLineComposing || a.Current() != origin {
				t.Fatal("pointer commit kept stale composition or queued reopen")
			}
			tester.Key(ui.Cmd|ui.Shift, ui.KeyT)
			if a.Current() == origin || len(a.closedFiles) != 0 {
				t.Fatal("explicit retry did not reopen")
			}
		})
	}
}

func TestReopenClosedAsyncIdentityNewCloseRecloseAndEviction(t *testing.T) {
	for _, change := range []string{"newer", "reclose", "evict"} {
		t.Run(change, func(t *testing.T) {
			dir := t.TempDir()
			a := newTestApp(t, filepath.Join(dir, "config"))
			path := filepath.Join(dir, "pending.txt")
			reopenFixture(t, a, path, []byte("pending disk"))
			id := a.closedFiles[0].id
			queued := make(chan func(), 32)
			a.uiDispatch = func(fn func()) { queued <- fn }
			done := a.reopenClosedFile(id)
			r := a.reopening
			if a.reopenClosedFile(0) != nil || a.reopening != r {
				t.Fatal("busy call queued a second request")
			}
			reopenDispatch(t, queued)() // lookup acknowledged; disk load may now finish.
			apply := reopenDispatch(t, queued)
			count := 1
			if change == "evict" {
				count = closedFileLimit
			}
			for i := 0; i < count; i++ {
				otherPath := filepath.Join(dir, fmt.Sprintf("newer-%d.txt", i))
				if change == "reclose" {
					otherPath = path
				}
				d, err := document.Open(path)
				if err != nil {
					t.Fatal(err)
				}
				d.Path = otherPath
				a.add(d, filepath.Base(otherPath), "")
				if !a.CloseTab(a.Active) {
					t.Fatal(a.Message)
				}
			}
			newTop := a.closedFiles[0].id
			apply()
			reopenDone(t, done)
			a.reapReopen()
			if a.closedFiles[0].id != newTop {
				t.Fatal("old completion consumed newer record")
			}
			if change == "newer" {
				if a.Current() == nil || a.Current().path() != path || a.closedFileIndex(id) >= 0 || r.outcome.Load() != reopenAdopted {
					t.Fatal("valid old ID was not adopted")
				}
			} else if a.Current() != nil || r.outcome.Load() != reopenRefused {
				t.Fatal("stale pending request opened a replacement record")
			}
		})
	}
}

func TestReopenClosedExistingAndPublicationDuplicateFlushRefusal(t *testing.T) {
	for _, late := range []bool{false, true} {
		t.Run(fmt.Sprint(late), func(t *testing.T) {
			dir := t.TempDir()
			a := newTestApp(t, filepath.Join(dir, "config"))
			path := filepath.Join(dir, "target.txt")
			reopenFixture(t, a, path, []byte("disk"))
			a.NewDocument("text")
			origin := a.Current()
			origin.Editor.ReplaceText(nil, "draft")
			failure := &failingView{flushErr: errors.New("unsynchronized rich draft")}
			origin.View = failure
			addExisting := func() *Tab {
				d, err := document.Open(path)
				if err != nil {
					t.Fatal(err)
				}
				return a.appendTab(d, "target.txt", "", false, false)
			}
			var existing *Tab
			if !late {
				existing = addExisting()
			}
			queued := make(chan func(), 16)
			a.uiDispatch = func(fn func()) { queued <- fn }
			done := a.reopenClosedFile(0)
			if late {
				reopenDispatch(t, queued)()
				apply := reopenDispatch(t, queued)
				existing = addExisting()
				apply()
			}
			reopenPump(t, a, queued, done)
			if len(a.closedFiles) != 1 || len(a.Tabs) != 2 || a.Current() != origin || failure.releases != 0 || origin.Doc.Buffer.String() != "draft" {
				t.Fatal("failed Select counted as adoption")
			}
			failure.flushErr = nil
			reopenPump(t, a, queued, a.reopenClosedFile(0))
			if a.Current() != existing || len(a.Tabs) != 2 || len(a.closedFiles) != 0 || !strings.Contains(a.Message, "已在标签中打开") {
				t.Fatal("existing Select retry failed")
			}
		})
	}
}

type reopenRecoveryView struct {
	failingView
	onFlush func() error
}

func (v *reopenRecoveryView) Flush() error { return v.onFlush() }

func TestReopenClosedSynchronousRecoveryDuringFlush(t *testing.T) {
	for _, route := range []string{"text", "existing", "png"} {
		t.Run(route, func(t *testing.T) {
			dir := t.TempDir()
			a := newTestApp(t, filepath.Join(dir, "config"))
			path := filepath.Join(dir, "target.txt")
			if route == "png" {
				path = filepath.Join(dir, "target.png")
				imageFixture(t, path, "PNG")
				a.Open(path)
				if !a.CloseTab(a.Active) {
					t.Fatal(a.Message)
				}
			} else {
				reopenFixture(t, a, path, []byte("disk"))
			}
			if route == "existing" {
				a.Open(path)
			}
			a.NewDocument("text")
			origin := a.Current()
			origin.Editor.ReplaceText(nil, "authority")
			v := &reopenRecoveryView{onFlush: func() error { return a.recoverDraft("conflict.txt", "kept recovery") }}
			origin.View = v
			count := len(a.Tabs)
			reopenDone(t, a.reopenClosedFile(0))
			if len(a.Tabs) != count+1 || a.Current() == origin || a.Current().Doc.Buffer.String() != "kept recovery" ||
				len(a.closedFiles) != 1 || v.releases != 0 || origin.Doc.Buffer.String() != "authority" {
				t.Fatal("adoption displaced synchronous recovery or lost history/draft")
			}
			if err := a.persistTabs(a.Tabs, a.Active, "recovery assertion"); err != nil {
				t.Fatal(err)
			}
			snapshot, issues := session.Load(filepath.Join(a.configDir, "session.json"))
			if len(issues) != 0 || snapshot.Tabs[len(snapshot.Tabs)-1].Draft == nil || *snapshot.Tabs[len(snapshot.Tabs)-1].Draft != "kept recovery" {
				t.Fatal("recovered draft not preserved")
			}
		})
	}
}

func TestReopenClosedModalRestoringAndAdoptionPreedit(t *testing.T) {
	for _, block := range []string{"modal", "restoring", "query", "replacement", "goto"} {
		t.Run(block, func(t *testing.T) {
			dir := t.TempDir()
			a := newTestApp(t, filepath.Join(dir, "config"))
			reopenFixture(t, a, filepath.Join(dir, "target.txt"), []byte("disk"))
			a.NewDocument("text")
			origin := a.Current()
			queued := make(chan func(), 16)
			a.uiDispatch = func(fn func()) { queued <- fn }
			done := a.reopenClosedFile(0)
			reopenDispatch(t, queued)()
			apply := reopenDispatch(t, queued)
			switch block {
			case "modal":
				a.modalDepth = 1
			case "restoring":
				a.restoring = true
			case "query":
				a.ShowFind, a.FindQueryComposing = true, true
			case "replacement":
				a.ShowFind, a.FindReplacementComposing = true, true
			case "goto":
				a.ShowGoToLine, a.GoToLineComposing = true, true
			}
			apply()
			reopenDone(t, done)
			a.reapReopen()
			if a.Current() != origin || len(a.closedFiles) != 1 || len(a.Tabs) != 1 {
				t.Fatal("adoption bypassed final guard")
			}
			if a.reopenClosedFile(0) != nil {
				t.Fatal("guard failed at command entry")
			}
			a.modalDepth, a.restoring = 0, false
			a.FindQueryComposing, a.FindReplacementComposing, a.GoToLineComposing = false, false, false
			reopenPump(t, a, queued, a.reopenClosedFile(0))
			if a.Current() == origin || len(a.closedFiles) != 0 {
				t.Fatal("explicit retry after guard failed")
			}
		})
	}
}

func TestReopenClosedTimeoutShutdownAndLateCallbacks(t *testing.T) {
	for _, phase := range []string{"lookup-timeout", "adoption-timeout", "shutdown", "window-close"} {
		t.Run(phase, func(t *testing.T) {
			dir := t.TempDir()
			a := newTestApp(t, filepath.Join(dir, "config"))
			reopenFixture(t, a, filepath.Join(dir, "target.txt"), []byte("disk"))
			backing := a.closedFiles[:cap(a.closedFiles)]
			a.updateWaitTimeout = 50 * time.Millisecond
			queued := make(chan func(), 16)
			a.uiDispatch = func(fn func()) { queued <- fn }
			done := a.reopenClosedFile(0)
			r := a.reopening
			late := reopenDispatch(t, queued)
			if phase != "lookup-timeout" {
				late()
				late = reopenDispatch(t, queued)
			}
			if phase == "shutdown" {
				a.Shutdown()
			}
			if phase == "window-close" {
				if !a.CanClose() {
					t.Fatal(a.Message)
				}
				late()
			}
			reopenDone(t, done)
			if phase != "window-close" {
				late()
			}
			for len(queued) > 0 {
				(<-queued)()
			}
			a.reapReopen()
			if len(a.Tabs) != 0 || r.outcome.Load() != reopenRefused || a.reopening != nil {
				t.Fatal("late open mutated workspace")
			}
			if phase == "shutdown" {
				if len(a.closedFiles) != 0 || backing[0] != (closedFile{}) {
					t.Fatal("shutdown retained path")
				}
			} else if len(a.closedFiles) != 1 {
				t.Fatal("unsuccessful request consumed record")
			}
		})
	}
}

func TestReopenClosedPNGAdoptionRefusalCancellationAndTimeout(t *testing.T) {
	for _, mode := range []string{"success", "flush", "duplicate", "cancel", "timeout", "shutdown", "modal-preedit"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "image.png")
			imageFixture(t, path, "PNG")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			a := newTestApp(t, filepath.Join(dir, "config"))
			a.Open(path)
			old := a.Current()
			if !a.CloseTab(0) {
				t.Fatal(a.Message)
			}
			if a.binarySnapshot != nil {
				t.Fatal("closed image retained snapshot")
			}
			a.NewDocument("text")
			origin := a.Current()
			queued := make(chan func(), 16)
			a.uiDispatch = func(fn func()) { queued <- fn }
			a.updateWaitTimeout = 50 * time.Millisecond
			done := a.reopenClosedFile(0)
			r := a.reopening
			reopenDispatch(t, queued)() // duplicate lookup
			reopenDispatch(t, queued)() // binary lane start
			j := a.binaryJob
			if j == nil || j.reopen != r {
				t.Fatal("binary job lost receipt")
			}
			apply := reopenDispatch(t, queued) // worker has finished, UI has not adopted.
			var existing *Tab
			switch mode {
			case "flush", "duplicate":
				origin.View = &failingView{flushErr: errors.New("pending input")}
				if mode == "duplicate" {
					file, snapshot, err := document.OpenBinary(context.Background(), path, document.BinaryKindPNG)
					if err != nil {
						t.Fatal(err)
					}
					existing = a.appendBinary(binaryOutput{file: file, snapshot: snapshot}, "image.png", "")
				}
			case "cancel":
				a.cancelReopen()
			case "timeout":
				reopenDone(t, done)
			case "shutdown":
				a.Shutdown()
			case "modal-preedit":
				a.modalDepth = 1
				apply()
				if a.Current() != origin || a.binaryJob != j || r.outcome.Load() != reopenPending || len(a.closedFiles) != 1 {
					t.Fatal("modal adopted ready binary")
				}
				a.ShowFind, a.FindQueryComposing = true, true
				a.modalDepth = 0
				a.deliverBinary()
			}
			if mode != "modal-preedit" {
				apply()
			}
			reopenDone(t, done)
			if mode == "timeout" {
				a.deliverBinary()
			}
			a.reapReopen()
			if mode == "success" {
				if a.Current() == old || a.Current() == origin || a.Current().Binary == nil || a.heldBinary(a.Current()) == nil || len(a.closedFiles) != 0 || r.outcome.Load() != reopenAdopted {
					t.Fatal("PNG never adopted")
				}
			} else {
				if a.Current() != origin || r.outcome.Load() != reopenRefused || a.binarySnapshot != nil {
					t.Fatal("refused PNG was adopted")
				}
				if mode != "shutdown" && len(a.closedFiles) != 1 {
					t.Fatal("refused PNG consumed history")
				}
				if mode == "duplicate" {
					origin.View.(*failingView).flushErr = nil
					reopenPump(t, a, queued, a.reopenClosedFile(0))
					if a.Current() != existing || len(a.Tabs) != 2 || len(a.closedFiles) != 0 {
						t.Fatal("binary duplicate retry failed")
					}
				}
			}
			if a.binaryJob != nil {
				t.Fatal("binary slot not released")
			}
			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(after, data) {
				t.Fatal("PNG bytes changed")
			}
		})
	}
}

func TestReopenClosedMenuGeometryAndEmptyState(t *testing.T) {
	dir := t.TempDir()
	a := newTestApp(t, filepath.Join(dir, "config"))
	tester := ui.NewTester(a.View, 760, 480)
	if err := tester.Click("最近关闭的文件"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(tester.Menu(), " "), "暂无记录") {
		t.Fatal("missing empty menu state")
	}
	tester.CloseMenu()
	for _, name := range []string{"older.md", "最新关闭.txt"} {
		reopenFixture(t, a, filepath.Join(dir, name), []byte("disk"))
	}
	tester.Frame()
	for _, label := range []string{"新建", "打开", "最近关闭的文件", "文件夹", "保存", "插件", "设置"} {
		rect, ok := tester.Find(label)
		if !ok || rect.X < 0 || rect.X+rect.W > 760 || rect.Y < 0 || rect.Y+rect.H > 46 {
			t.Fatalf("toolbar %s outside 760x480: %+v", label, rect)
		}
	}
	if _, ok := tester.Find("留一点空间给想法。"); !ok {
		t.Fatal("zero-tab welcome lost")
	}
	if err := tester.Click("最近关闭的文件"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(tester.Menu(), " "), "不恢复已放弃的修改") {
		t.Fatal("path-only limitation missing")
	}
	if path := os.Getenv("LEVNOTE_REOPEN_SCREENSHOT"); path != "" {
		captureAcceptanceTester(t, path, tester)
	}
}

func TestReopenClosedSuccessfulShortcutRepeatAndStaleMenuID(t *testing.T) {
	dir := t.TempDir()
	a := newTestApp(t, filepath.Join(dir, "config"))
	older, newer := filepath.Join(dir, "older.txt"), filepath.Join(dir, "newer.txt")
	reopenFixture(t, a, older, []byte("older"))
	reopenFixture(t, a, newer, []byte("newer"))
	tester := ui.NewTester(a.View, 760, 480)
	if err := tester.Click("最近关闭的文件"); err != nil {
		t.Fatal(err)
	}
	oldID := a.closedFiles[0].id
	// Keep the menu's old ID snapshot while that path is reclosed with a fresh
	// ID, as can happen during a native menu's nested event loop.
	a.Open(newer)
	if !a.CloseTab(a.Active) || a.closedFiles[0].id == oldID {
		t.Fatal("reclose fixture did not replace ID")
	}
	if err := tester.ChooseMenuItem("1. newer.txt"); err != nil {
		t.Fatal(err)
	}
	if a.Current() != nil || len(a.closedFiles) != 2 {
		t.Fatal("stale menu row activated the replacement at its old ordinal")
	}
	tester.Key(ui.Cmd|ui.Shift, ui.KeyT)
	if a.Current() == nil || a.Current().path() != newer || len(a.closedFiles) != 1 {
		t.Fatal("fresh shortcut did not adopt newest file")
	}
	for _, mods := range []ui.Modifiers{ui.Cmd, ui.Shift, ui.Cmd | ui.Shift | ui.Alt} {
		tester.Key(mods, ui.KeyT)
	}
	tester.KeyRepeat(ui.Cmd|ui.Shift, ui.KeyT)
	if a.Current().path() != newer || len(a.Tabs) != 1 || len(a.closedFiles) != 1 {
		t.Fatal("held/nonmatching key reopened an older entry after success")
	}
	tester.Key(ui.Cmd|ui.Shift, ui.KeyT)
	if a.Current().path() != older || len(a.Tabs) != 2 || len(a.closedFiles) != 0 {
		t.Fatal("second deliberate shortcut did not reopen older entry")
	}
}
