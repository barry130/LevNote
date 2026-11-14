package app

import "github.com/egoist/mygo/ui"

// appListLifetime keeps the last build alive through MyGo's late layout. Only
// the public Handle survives a semantic reset; no old ListState is retained.
// App.View starts the next safe boundary before building any list.
type appListLifetime struct {
	built         bool
	resetPending  bool
	revealPending bool
	revealRow     int
}

func resetAppListState(state *ui.ListState) {
	handle := state.Handle
	*state = ui.ListState{Handle: handle}
}

func (l *appListLifetime) begin(state *ui.ListState) {
	l.built = false
	if l.resetPending {
		resetAppListState(state)
		l.resetPending = false
	}
	if l.revealPending {
		state.ScrollIntoView(l.revealRow)
		l.revealPending = false
	}
	// Do not cancel focus here: a deliberate new request made after the
	// invalidation must survive the deferred reset on the same Handle.
}

func (l *appListLifetime) reset(state *ui.ListState) {
	state.Handle.CancelFocus()
	l.revealPending = false
	if l.built {
		l.resetPending = true
		return
	}
	resetAppListState(state)
	l.resetPending = false
}

func (l *appListLifetime) reveal(state *ui.ListState, row int) {
	if l.resetPending {
		l.revealPending, l.revealRow = true, row
		return
	}
	state.ScrollIntoView(row)
}

func (l *appListLifetime) close(state *ui.ListState) {
	// Shutdown is called after the product window closes. There is no live
	// layout to defer, so release all application-owned row callbacks now.
	l.built = false
	l.reset(state)
}

func (a *App) beginAppListFrame() {
	a.treeListLifetime.begin(&a.TreeList)
	a.searchListLifetime.begin(&a.SearchList)
	a.outlineListLifetime.begin(&a.OutlineList)
}

func (a *App) cancelHiddenListFocus() {
	if a.ShowOutline || a.Root == "" {
		a.TreeList.Handle.CancelFocus()
	}
	if !a.ShowWorkspaceSearch || a.ShowPlugins {
		a.SearchList.Handle.CancelFocus()
	}
	if !a.ShowOutline || a.Current() == nil || a.Current().isBinary() {
		a.OutlineList.Handle.CancelFocus()
	}
}

func (a *App) closeAppLists() {
	a.treeListLifetime.close(&a.TreeList)
	a.searchListLifetime.close(&a.SearchList)
	a.outlineListLifetime.close(&a.OutlineList)
	a.TreeSelected, a.SearchSelected, a.OutlineSelected = -1, -1, -1
	a.searchRoot = ""
	a.Entries, a.SearchResults, a.OutlineItems = nil, nil, nil
	a.treeRowBuilt, a.searchRowBuilt, a.outlineRowBuilt = nil, nil, nil
}
