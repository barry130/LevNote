package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/plugins"
	"levnote.local/native/formats/code"
	"levnote.local/native/modules"
	"levnote.local/native/ui/editor"
)

type syntaxPalette struct {
	keyword, stringValue, number, comment, tag ui.Color
}

func syntaxColors(dark bool) syntaxPalette {
	if dark {
		return syntaxPalette{
			keyword: ui.Hex("#93c5fd"), stringValue: ui.Hex("#86efac"),
			number: ui.Hex("#fbbf24"), comment: ui.Hex("#cbd5e1"), tag: ui.Hex("#d8b4fe"),
		}
	}
	return syntaxPalette{
		keyword: ui.Hex("#1d4ed8"), stringValue: ui.Hex("#166534"),
		number: ui.Hex("#92400e"), comment: ui.Hex("#475569"), tag: ui.Hex("#6d28d9"),
	}
}

func (a *App) adapterByID(id string) (editor.Adapter, bool) {
	for _, adapter := range a.adapters {
		if adapter.ID == id {
			return adapter, true
		}
	}
	return editor.Adapter{}, false
}

func (a *App) configureTab(t *Tab) {
	if t == nil {
		return
	}
	a.invalidateFindCache()
	a.closeGoToLine()
	a.invalidateTabExport(t, "文档格式已重新配置，已取消导出")
	t.Adapter = nil
	t.OutlineFactory = nil
	a.outlineFactorySequence++
	t.OutlineFactoryGeneration = a.outlineFactorySequence
	t.MatchedPlugin = plugins.Manifest{}
	t.FormatWarning = ""
	t.SyntaxNotice = ""
	t.SourceCapabilityBadge = ""
	t.SourceCapabilityDetail = ""
	t.Editor.Highlight = nil
	if t.isBinary() {
		a.configureBinary(t)
		return
	}
	if t.ByteInspector {
		t.Mode = "source"
		return
	}
	path := t.Doc.Path
	if path == "" {
		path = t.Name
	}
	var match plugins.Match
	var err error
	if a.pluginManager != nil {
		match, err = a.pluginManager.Match(path)
	} else {
		err = fmt.Errorf("plugin manager is unavailable")
	}
	if err != nil {
		t.FormatWarning = fmt.Sprintf("没有启用的格式插件声明 %s；%s", extensionLabel(path), sourceFallbackText(t))
	} else {
		t.MatchedPlugin = match.Manifest
		if t.MatchedPlugin.Declarative {
			t.SourceCapabilityBadge = "源码"
			t.SourceCapabilityDetail = "声明式源码格式；不执行插件代码"
		}
		if syntax, ok := a.pluginManager.Syntax(match.ID); ok {
			configuredHighlight := declarativeHasConfiguredHighlight(syntax)
			rule := code.DeclarativeSyntax{
				LineComment: syntax.LineComment, BlockStart: syntax.BlockStart,
				BlockEnd: syntax.BlockEnd, Keywords: append([]string(nil), syntax.Keywords...),
			}
			t.Editor.Highlight = func(line string) []ui.Span {
				return codeSpans(line, code.HighlightDeclarative(line, rule), syntaxColors(a.Dark))
			}
			t.OutlineFactory = modules.InstalledSourceOutlineFactory(match.ID, syntax)
			if t.MatchedPlugin.Declarative {
				t.SourceCapabilityBadge = declarativeCapabilityBadge(syntax)
				t.SourceCapabilityDetail = declarativeCapability(syntax)
			}
			if syntax.Schema == 3 {
				if adapter, available := a.nativeRouteAdapter(syntax.NativeRoute); available {
					t.Adapter = &adapter
				} else {
					t.FormatWarning = "配置指定的原生格式不可用；" + sourceFallbackText(t)
				}
			}
			if syntax.Schema == 2 && len(syntax.Outline) > 0 {
				if configuredHighlight {
					t.SyntaxNotice = "源码大纲与高亮都按单行字面规则处理；规则可能命中多行注释、字符串或块内的同形文本，文档文本不受影响。"
				} else {
					t.SyntaxNotice = "源码大纲按逐行字面规则处理；规则可能命中多行注释、字符串或块内的同形文本，文档文本不受影响。高亮仅含基础字符串/数字识别，未配置额外高亮规则。"
				}
			} else if configuredHighlight {
				t.SyntaxNotice = "高亮按单行独立处理；跨行块注释或字符串可能显示不准确，文档文本不受影响。"
			}
		} else if language := code.LanguageForPath(path); language != "" {
			t.Editor.Highlight = func(line string) []ui.Span {
				return codeSpans(line, code.Highlight(line, language), syntaxColors(a.Dark))
			}
			switch language {
			case code.LanguageGo, code.LanguageTypeScript, code.LanguageJavaScript,
				code.LanguagePython, code.LanguageRust:
				t.SyntaxNotice = "高亮按单行独立处理；跨行块注释或字符串可能显示不准确，文档文本不受影响。"
			}
		}
	}
	if t.Editor.Highlight != nil && t.SourceCapabilityBadge == "" {
		t.SourceCapabilityBadge = "源码编辑"
		t.SourceCapabilityDetail = "源码编辑；高亮按单行规则处理"
	}
	if t.Adapter != nil {
		t.Mode = "preview"
		return
	}
	if match.Manifest.Builtin {
		if match.ID == "image" {
			t.Mode = "source"
			t.FormatWarning = "此标签保留文本草稿；图片扩展名不会把现有文本改成二进制图片。"
			return
		}
		if adapter, ok := a.adapterByID(match.ID); ok {
			t.Adapter = &adapter
			t.Mode = "preview"
			return
		}
	}
	t.Mode = "source"
}

func sourceFallbackText(t *Tab) string {
	if t.Doc.ReadOnly {
		return "可以查看源码；另存为可写副本。"
	}
	return "已用纯文本源码打开，内容仍可编辑和保存。"
}

func (a *App) nativeRouteAdapter(route plugins.NativeRoute) (editor.Adapter, bool) {
	adapter, ok := modules.NativeRouteAdapter(route)
	if !ok {
		return editor.Adapter{}, false
	}
	// The closed catalog is independent of registry enablement. An unexpected
	// missing compiled host adapter is still a visible source fallback.
	compiled, exists := a.adapterByID(adapter.ID)
	return adapter, exists && compiled.Create != nil
}

func nativeRouteLabel(route plugins.NativeRoute) string {
	if adapter, ok := modules.NativeRouteAdapter(route); ok {
		return strings.TrimSpace(strings.TrimPrefix(adapter.Name, "原生"))
	}
	return "不可用格式"
}

func declarativeCapability(syntax plugins.SyntaxManifest) string {
	configuredHighlight := declarativeHasConfiguredHighlight(syntax)
	switch {
	case syntax.Schema == 3:
		return "使用 " + nativeRouteLabel(syntax.NativeRoute) + " · 随包原生格式能力；具体阅读、编辑、分屏与导出支持以当前格式为准；源码高亮按单行规则处理"
	case syntax.Schema == 2 && len(syntax.Outline) > 0 && configuredHighlight:
		return fmt.Sprintf("schema2 · 源码大纲（%d 条逐行字面规则）+ 基础字符串/数字高亮与已配置行内规则；不识别多行注释/字符串上下文", len(syntax.Outline))
	case syntax.Schema == 2 && len(syntax.Outline) > 0:
		return fmt.Sprintf("schema2 · 源码大纲（%d 条逐行字面规则）+ 基础字符串/数字高亮（未配置额外高亮规则）；不识别多行注释/字符串上下文", len(syntax.Outline))
	case syntax.Schema == 2 && configuredHighlight:
		return "schema2 · 未声明源码大纲；提供基础字符串/数字高亮与已配置行内规则"
	case syntax.Schema == 2:
		return "schema2 · 未声明源码大纲；提供基础字符串/数字高亮（未配置额外高亮规则）"
	case configuredHighlight:
		return "schema1 · 提供基础字符串/数字高亮与已配置行内规则；不支持源码大纲"
	default:
		return "schema1 · 提供基础字符串/数字高亮（未配置额外高亮规则）；不支持源码大纲"
	}
}

func declarativeCapabilityBadge(syntax plugins.SyntaxManifest) string {
	if syntax.Schema == 3 {
		return "使用 " + nativeRouteLabel(syntax.NativeRoute)
	}
	hasOutline := syntax.Schema == 2 && len(syntax.Outline) > 0
	if hasOutline {
		return "源码大纲 · 高亮"
	}
	return "仅行内高亮"
}

func declarativeHasConfiguredHighlight(syntax plugins.SyntaxManifest) bool {
	return syntax.LineComment != "" || syntax.BlockStart != "" || len(syntax.Keywords) > 0
}

func extensionLabel(path string) string {
	extension := strings.TrimSpace(filepath.Ext(path))
	if extension == "" {
		return "此无扩展名文件"
	}
	return extension + " 文件"
}

func codeSpans(source string, result code.Result, colors syntaxPalette) []ui.Span {
	if source == "" || len(result.Spans) == 0 || len(result.Warnings) > 0 {
		return nil
	}
	spans := make([]ui.Span, 0, len(result.Spans))
	cursor := 0
	for _, token := range result.Spans {
		if token.Start < cursor || token.End < token.Start || token.End > len(source) {
			return nil
		}
		span := ui.Span{Text: source[token.Start:token.End]}
		switch token.Token {
		case code.TokenKeyword:
			span.Color = colors.keyword
			span.Weight = 600
		case code.TokenString:
			span.Color = colors.stringValue
		case code.TokenNumber:
			span.Color = colors.number
		case code.TokenComment:
			span.Color = colors.comment
			span.Italic = true
		case code.TokenTag:
			span.Color = colors.tag
		}
		spans = append(spans, span)
		cursor = token.End
	}
	if cursor != len(source) {
		return nil
	}
	return spans
}

func (a *App) setPluginEnabled(id string, enabled bool) {
	if a.pluginManager == nil {
		a.report(fmt.Errorf("格式插件管理器不可用"))
		return
	}
	if !a.flushAllViews("更改格式插件") {
		return
	}
	if err := a.pluginManager.SetEnabled(id, enabled); err != nil {
		if errors.Is(err, plugins.ErrEnabledNotLoaded) {
			a.reconfigureTabs()
		}
		a.report(fmt.Errorf("格式插件 %s：%w", id, err))
		return
	}
	a.reconfigureTabs()
	state := "已停用"
	if enabled {
		state = "已启用"
	}
	a.Message = fmt.Sprintf("插件 %s %s", id, state)
}

func (a *App) installPlugin() {
	defer a.deferExportDelivery()()
	if a.pluginManager == nil {
		a.report(fmt.Errorf("格式插件管理器不可用"))
		return
	}
	if !a.flushAllViews("安装格式插件") {
		return
	}
	paths, err := a.chooseOpenPaths(mygo.OpenDialogOptions{
		Parent: a.Window, Title: "安装声明式格式插件",
		Message: "选择本地 JSON 格式配置：高亮、逐行源码大纲，或使用内置 Markdown / 思维导图 / draw.io。",
		Filters: []mygo.FileFilter{{Name: "JSON 清单", Extensions: []string{"json"}}},
	})
	if err != nil {
		a.report(err)
		return
	}
	if len(paths) == 0 {
		return
	}
	// Native dialogs can pump nested input; a new pending draft must be
	// synchronized before installing and replacing any presentation.
	if !a.flushAllViews("安装格式插件") {
		return
	}
	if err := a.pluginManager.InstallFile(paths[0]); err != nil {
		a.report(fmt.Errorf("安装格式插件：%w", err))
		return
	}
	a.reconfigureTabs()
	a.Message = "已安装格式插件 " + filepath.Base(paths[0])
}

func (a *App) refreshPlugins() {
	if a.pluginManager == nil {
		a.report(fmt.Errorf("格式插件管理器不可用"))
		return
	}
	if !a.flushAllViews("重新扫描格式插件") {
		return
	}
	if err := a.pluginManager.Refresh(); err != nil {
		a.report(err)
	} else {
		a.Message = "格式插件列表已刷新"
	}
	a.reconfigureTabs()
}

func (a *App) reconfigureTabs() {
	for _, t := range a.Tabs {
		if t.isBinary() && t == a.Current() {
			a.releaseTabPresentation(t, true)
			t.View = nil
		} else {
			a.disposePresentation(t)
		}
		oldMode := t.Mode
		t.Mode = "source"
		a.configureTab(t)
		if t.Adapter != nil && oldMode != "" {
			t.Mode = oldMode
		}
	}
	a.markPersistPending()
	if current := a.Current(); current != nil {
		current.Editor.Focus()
	}
}

func (a *App) flushAllViews(action string) bool {
	for _, t := range a.Tabs {
		if !a.flushTab(t, action) {
			return false
		}
	}
	return true
}
