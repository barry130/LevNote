package app

import (
	"errors"
	"fmt"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/search"
	"levnote.local/native/core/text"
	"levnote.local/native/ui/editor"
)

type findMatchOffset struct {
	Start, End int
}

type findFocusTarget uint8

const (
	findFocusNone findFocusTarget = iota
	findFocusQuery
	findFocusReplacement
)

type findNavigationCache struct {
	tab           *Tab
	doc           *document.Document
	editor        *editor.Editor
	buffer        *text.Buffer
	version       uint64
	options       search.Options
	matches       []findMatchOffset
	err           error
	lastIndex     int
	lastSelection text.Range
}

// searchNavigationFind is a tiny test seam for counting cold scans. Production
// always uses the shared search implementation.
var searchNavigationFind = search.Find

func (a *App) invalidateFindCache() {
	a.findNavigation = findNavigationCache{lastIndex: -1}
	a.FindIndex = -1
}

func (a *App) expireFindCache() {
	cache := &a.findNavigation
	t := a.Current()
	if cache.tab == nil {
		return
	}
	if t == nil || t.Doc == nil || cache.tab != t || cache.doc != t.Doc || cache.editor != t.Editor ||
		cache.buffer != t.Doc.Buffer || cache.buffer == nil || cache.version != cache.buffer.Version() ||
		cache.options != a.findOptions() {
		a.invalidateFindCache()
	}
}

func (a *App) closeFind() {
	a.findGeneration++
	a.ShowFind = false
	a.clearFindTransient()
	a.invalidateFindCache()
}

func (a *App) clearFindTransient() {
	a.FindQueryComposing = false
	a.FindReplacementComposing = false
	a.findQueryHandle.CancelFocus()
	a.findReplacementHandle.CancelFocus()
	a.clearFindFocusRequest()
}

func (a *App) clearFindFocusRequest() {
	a.findFocus = findFocusNone
	a.findFocusGeneration = 0
	a.findFocusTab = nil
	a.findFocusDoc = nil
	a.findFocusEditor = nil
	a.findFocusBuffer = nil
}

func (a *App) expireFindFocus() {
	if a.findFocus == findFocusNone {
		return
	}
	if !a.ShowFind || a.ShowPlugins || a.findGeneration != a.findFocusGeneration ||
		!a.findTargetCurrent(a.findFocusTab, a.findFocusDoc, a.findFocusEditor, a.findFocusBuffer) {
		a.clearFindFocusRequest()
	}
}

func (a *App) reconcileFindFieldComposition(input ui.Element, composing *bool) {
	if *composing && !input.Focused() && !input.Composing() {
		*composing = false
	}
}

func (a *App) toggleFind() {
	if a.ShowFind {
		a.cancelFind()
		return
	}
	a.openFind()
}

func (a *App) findCompositionActive() bool {
	return a.FindQueryComposing || a.FindReplacementComposing
}

func (a *App) openFind() bool {
	if a.findCompositionActive() || a.GoToLineComposing {
		a.Message = "输入法正在组合文本；请先完成组合后再查找"
		return false
	}
	t := a.Current()
	if t == nil || t.Doc == nil || t.Editor == nil || t.Doc.Buffer == nil {
		a.Message = "没有可查找的文档"
		return false
	}
	if a.sourceCompositionActive(t) {
		a.Message = "输入法正在组合文本；请先完成组合后再查找"
		return false
	}
	if !a.ShowFind {
		a.findGeneration++
		a.ShowFind = true
	}
	a.requestFindFocusAt(a.findGeneration, findFocusQuery, t, t.Doc, t.Editor, t.Doc.Buffer)
	return true
}

func (a *App) cancelFind() {
	a.cancelFindAt(a.findGeneration)
}

func (a *App) cancelFindAt(generation uint64) bool {
	if !a.ShowFind || a.findGeneration != generation {
		return false
	}
	if a.findCompositionActive() {
		a.Message = "输入法正在组合文本；请先完成组合后再关闭查找"
		return false
	}
	t := a.Current()
	a.closeFind()
	if t != nil && t == a.Current() && a.tabIsOpen(t) &&
		t.Doc != nil && t.Editor != nil && t.Doc.Buffer == t.Editor.Buffer &&
		t.Mode != "preview" {
		t.Editor.Focus()
	}
	return true
}

func (a *App) findActionAllowedAt(generation uint64) bool {
	if !a.ShowFind || a.findGeneration != generation {
		return false
	}
	if a.findCompositionActive() {
		a.Message = "输入法正在组合文本；请先完成组合后再操作查找"
		return false
	}
	return true
}

func (a *App) findFromQueryKeyAt(generation uint64, key ui.TextKey, next bool) bool {
	if !a.findActionAllowedAt(generation) {
		return false
	}
	t := a.Current()
	if t == nil || t.Doc == nil || t.Editor == nil {
		return false
	}
	doc, input, buffer := t.Doc, t.Editor, t.Doc.Buffer
	a.Query = key.Text()
	a.find(next)
	a.requestFindFocusAt(generation, findFocusQuery, t, doc, input, buffer)
	return true
}

func (a *App) replaceFromReplacementKeyAt(generation uint64, key ui.TextKey) bool {
	if !a.findActionAllowedAt(generation) {
		return false
	}
	t := a.Current()
	if t == nil || t.Doc == nil || t.Editor == nil {
		return false
	}
	doc, input, buffer := t.Doc, t.Editor, t.Doc.Buffer
	a.Replacement = key.Text()
	a.replace(false)
	a.requestFindFocusAt(generation, findFocusReplacement, t, doc, input, buffer)
	return true
}

func (a *App) findTargetCurrent(t *Tab, doc *document.Document, input *editor.Editor, buffer *text.Buffer) bool {
	return t != nil && a.Current() == t && a.tabIsOpen(t) &&
		t.Doc == doc && doc != nil && doc.Buffer == buffer &&
		t.Editor == input && input != nil && input.Buffer == buffer
}

func (a *App) requestFindFocusAt(generation uint64, target findFocusTarget, t *Tab, doc *document.Document, input *editor.Editor, buffer *text.Buffer) {
	if !a.ShowFind || a.findGeneration != generation || target == findFocusNone ||
		!a.findTargetCurrent(t, doc, input, buffer) {
		return
	}
	input.CancelFocus()
	a.findFocus = target
	a.findFocusGeneration = generation
	a.findFocusTab = t
	a.findFocusDoc = doc
	a.findFocusEditor = input
	a.findFocusBuffer = buffer
}

// resolveFindFocus applies a key/open request from the prior event after the
// Find controls have been bound. requestFindFocusAt already cancels the source
// reveal's pending focus, so this one-shot request owns the next focused target.
func (a *App) resolveFindFocus(generation uint64) {
	target := a.findFocus
	if target == findFocusNone {
		return
	}
	t, doc, input, buffer := a.findFocusTab, a.findFocusDoc, a.findFocusEditor, a.findFocusBuffer
	a.findFocus = findFocusNone
	a.findFocusTab = nil
	a.findFocusDoc = nil
	a.findFocusEditor = nil
	a.findFocusBuffer = nil
	if !a.ShowFind || generation != a.findGeneration ||
		generation != a.findFocusGeneration ||
		!a.findTargetCurrent(t, doc, input, buffer) {
		return
	}
	switch target {
	case findFocusQuery:
		a.findQueryHandle.Focus()
	case findFocusReplacement:
		a.findReplacementHandle.Focus()
	}
}

func (a *App) searchMatchesCurrent(t *Tab, options search.Options) (*findNavigationCache, error) {
	cache := &a.findNavigation
	buffer := t.Doc.Buffer
	if cache.tab == t && cache.doc == t.Doc && cache.editor == t.Editor && cache.buffer == buffer &&
		cache.version == buffer.Version() && cache.options == options {
		return cache, cache.err
	}

	a.invalidateFindCache()
	version := buffer.Version()
	snapshot := buffer.Snapshot()
	matches, searchErr := searchNavigationFind(snapshot, options)
	if buffer.Version() != version {
		a.invalidateFindCache()
		return nil, errors.New("文档在查找期间已变化；未跳转")
	}
	var offsets []findMatchOffset
	if searchErr == nil {
		offsets = make([]findMatchOffset, 0, len(matches))
		for _, match := range matches {
			offsets = append(offsets, findMatchOffset{Start: match.Start, End: match.End})
		}
	}
	a.findNavigation = findNavigationCache{
		tab: t, doc: t.Doc, editor: t.Editor, buffer: buffer,
		version: version, options: options, matches: offsets, err: searchErr,
		lastIndex: -1,
	}
	return &a.findNavigation, searchErr
}

func chooseFindIndex(matches []findMatchOffset, selection text.Range, lastIndex int, lastSelection text.Range, next bool) int {
	if len(matches) == 0 {
		return -1
	}
	if lastIndex >= 0 && lastIndex < len(matches) && selection == lastSelection {
		if next {
			return (lastIndex + 1) % len(matches)
		}
		return (lastIndex - 1 + len(matches)) % len(matches)
	}

	start, end := selection.Start(), selection.End()
	if start != end {
		for i, match := range matches {
			if match.Start == start && match.End == end {
				if next {
					return (i + 1) % len(matches)
				}
				return (i - 1 + len(matches)) % len(matches)
			}
		}
	}
	if next {
		for i, match := range matches {
			if match.Start >= end {
				return i
			}
		}
		return 0
	}
	for i := len(matches) - 1; i >= 0; i-- {
		if matches[i].End <= start {
			return i
		}
	}
	return len(matches) - 1
}

func (a *App) find(next bool) {
	t := a.Current()
	if !t.textCapable() {
		return
	}
	doc, input, buffer := t.Doc, t.Editor, t.Doc.Buffer
	if !a.flushTab(t, "查找") {
		return
	}
	if !a.findTargetCurrent(t, doc, input, buffer) {
		a.invalidateFindCache()
		a.Message = "查找时文档已变化；未跳转"
		return
	}

	options := a.findOptions()
	if options.Query == "" {
		a.invalidateFindCache()
		a.Message = "没有匹配结果"
		return
	}
	cache, err := a.searchMatchesCurrent(t, options)
	if err != nil {
		a.FindIndex = -1
		a.report(err)
		return
	}
	if !a.findTargetCurrent(t, doc, input, buffer) {
		a.invalidateFindCache()
		a.Message = "查找时文档已变化；未跳转"
		return
	}
	if len(cache.matches) == 0 {
		a.FindIndex = -1
		a.Message = "没有匹配结果"
		return
	}

	index := chooseFindIndex(cache.matches, buffer.Selection(), cache.lastIndex, cache.lastSelection, next)
	if index < 0 || index >= len(cache.matches) {
		a.invalidateFindCache()
		a.Message = "搜索结果位置已失效；未跳转"
		return
	}
	target := cache.matches[index]
	if !a.findTargetCurrent(t, doc, input, buffer) ||
		!a.revealSource(t, revealTarget{Start: target.Start, End: target.End}) {
		a.invalidateFindCache()
		return
	}
	cache.lastIndex = index
	cache.lastSelection = buffer.Selection()
	a.FindIndex = index
	a.Message = fmt.Sprintf("匹配 %d / %d", index+1, len(cache.matches))
}
