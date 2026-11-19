package app

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestCloseTabBackingRetention(t *testing.T) {
	t.Run("last tab with history", func(t *testing.T) {
		a := newTestApp(t, t.TempDir())
		a.NewDocument("text")
		closed := a.Current()
		for i := 0; i < 128; i++ {
			closed.Editor.ReplaceText(nil, fmt.Sprintf("history segment %03d\n", i))
		}
		if history := closed.Doc.Buffer.History(); history.UndoDepth < 2 {
			t.Fatalf("fixture did not retain substantial edit history: %+v", history)
		}
		a.closeDialog = func(*Tab) (closeDecision, error) {
			return closeDiscard, nil
		}

		if !a.CloseTab(0) {
			t.Fatal("closing the last tab failed")
		}
		if len(a.Tabs) != 0 || a.Active != -1 {
			t.Fatalf("last tab remained live: len=%d active=%d", len(a.Tabs), a.Active)
		}
		assertTabAbsentFromBacking(t, a, closed)
		assertTabBackingTailCleared(t, a)
	})

	t.Run("middle tab preserves remaining order and selection", func(t *testing.T) {
		a := newTestApp(t, t.TempDir())
		a.NewDocument("text")
		first := a.Current()
		first.Editor.ReplaceText(nil, "first")
		a.NewDocument("text")
		closed := a.Current()
		closed.Editor.ReplaceText(nil, "closed")
		a.NewDocument("text")
		last := a.Current()
		last.Editor.ReplaceText(nil, "last")
		a.closeDialog = func(*Tab) (closeDecision, error) {
			return closeDiscard, nil
		}

		if !a.CloseTab(1) {
			t.Fatal("closing the middle tab failed")
		}
		if len(a.Tabs) != 2 || a.Tabs[0] != first || a.Tabs[1] != last {
			t.Fatalf("middle removal changed remaining order: %#v", a.Tabs)
		}
		if a.Active != 1 || a.Current() != last {
			t.Fatalf("middle removal selected the wrong tab: active=%d current=%p", a.Active, a.Current())
		}
		if first.Doc.Buffer.String() != "first" || last.Doc.Buffer.String() != "last" {
			t.Fatal("middle removal changed remaining document content")
		}
		assertTabAbsentFromBacking(t, a, closed)
		assertTabBackingTailCleared(t, a)
	})

	t.Run("closing all tabs clears every vacated slot", func(t *testing.T) {
		a := newTestApp(t, t.TempDir())
		closed := make([]*Tab, 0, 3)
		for i := 0; i < 3; i++ {
			a.NewDocument("text")
			tab := a.Current()
			tab.Editor.ReplaceText(nil, fmt.Sprintf("document %d", i))
			closed = append(closed, tab)
		}
		a.closeDialog = func(*Tab) (closeDecision, error) {
			return closeDiscard, nil
		}

		for len(a.Tabs) > 0 {
			if !a.CloseTab(len(a.Tabs) - 1) {
				t.Fatalf("close failed with %d tabs remaining", len(a.Tabs))
			}
		}
		if a.Active != -1 {
			t.Fatalf("active tab after closing all = %d", a.Active)
		}
		for _, tab := range closed {
			assertTabAbsentFromBacking(t, a, tab)
		}
		assertTabBackingTailCleared(t, a)
	})

	t.Run("cancel preserves tabs content selection and recovery", func(t *testing.T) {
		dir := t.TempDir()
		a := newTestApp(t, dir)
		for _, file := range []struct {
			name  string
			value string
		}{
			{name: "first.txt", value: "first draft"},
			{name: "second.txt", value: "second draft"},
		} {
			path := filepath.Join(dir, file.name)
			if err := os.WriteFile(path, []byte(file.name), 0600); err != nil {
				t.Fatal(err)
			}
			a.Open(path)
			a.Current().Editor.Command("selectAll")
			a.Current().Editor.ReplaceText(nil, file.value)
		}
		a.Persist()

		first := a.Tabs[0]
		beforeTabs := append([]*Tab(nil), a.Tabs...)
		backing := a.Tabs[:cap(a.Tabs)]
		beforeBacking := append([]*Tab(nil), backing...)
		beforeContent := []string{first.Doc.Buffer.String(), a.Tabs[1].Doc.Buffer.String()}
		beforeActive := a.Active
		beforeCurrent := a.Current()
		sessionPath := filepath.Join(dir, "session.json")
		beforeSession, err := os.ReadFile(sessionPath)
		if err != nil {
			t.Fatal(err)
		}
		cancelled := false
		a.closeDialog = func(tab *Tab) (closeDecision, error) {
			cancelled = tab == first
			return closeCancel, nil
		}

		if a.CloseTab(0) {
			t.Fatal("close succeeded after cancel")
		}
		if !cancelled {
			t.Fatal("cancel decision did not receive the requested tab")
		}
		if len(a.Tabs) != len(beforeTabs) || !sameTabPointers(a.Tabs, beforeTabs) {
			t.Fatalf("cancel changed live tab pointers: before=%#v after=%#v", beforeTabs, a.Tabs)
		}
		if !sameTabPointers(a.Tabs[:cap(a.Tabs)], beforeBacking) {
			t.Fatal("cancel changed pointers in the live backing array")
		}
		if a.Active != beforeActive || a.Current() != beforeCurrent {
			t.Fatalf("cancel changed selection: active=%d current=%p", a.Active, a.Current())
		}
		if first.Doc.Buffer.String() != beforeContent[0] ||
			a.Tabs[1].Doc.Buffer.String() != beforeContent[1] {
			t.Fatal("cancel changed document content")
		}
		afterSession, err := os.ReadFile(sessionPath)
		if err != nil {
			t.Fatal(err)
		}
		if string(afterSession) != string(beforeSession) {
			t.Fatal("cancel changed the persisted recovery snapshot")
		}
	})
}

func assertTabAbsentFromBacking(t *testing.T, a *App, closed *Tab) {
	t.Helper()
	for i, tab := range a.Tabs[:cap(a.Tabs)] {
		if tab == closed {
			t.Fatalf("closed tab %p remains in live backing array at index %d (len=%d cap=%d)",
				closed, i, len(a.Tabs), cap(a.Tabs))
		}
	}
}

func assertTabBackingTailCleared(t *testing.T, a *App) {
	t.Helper()
	for i, tab := range a.Tabs[len(a.Tabs):cap(a.Tabs)] {
		if tab != nil {
			t.Fatalf("stale tab %p remains beyond slice length at backing index %d (len=%d cap=%d)",
				tab, len(a.Tabs)+i, len(a.Tabs), cap(a.Tabs))
		}
	}
}

func sameTabPointers(left, right []*Tab) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
