package app

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/document"
	"levnote.local/native/core/plugins"
	"levnote.local/native/core/text"
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

type configureControlSnapshot struct {
	tab                                     *Tab
	id                                      int
	doc                                     *document.Document
	input                                   *editor.Editor
	docBuffer, inputBuffer                  *text.Buffer
	view                                    editor.View
	adapter                                 *editor.Adapter
	manager                                 *plugins.Manager
	match                                   plugins.Match
	activeIndex                             int
	current                                 *Tab
	configDir                               string
	name, originalPath                      string
	docPath                                 string
	docEncoding                             document.Encoding
	docBOM                                  bool
	docReadOnly, inputReadOnly, inspector   bool
	docIsNew                                bool
	inputBinding                            *sourceInputBinding
	inputEpoch, route                       uint64
	profile                                 code.StructureProfile
	outlineFactory                          func(editor.Host) editor.OutlineSource
	outlineGeneration, hostEpoch            uint64
	docStamp, inputStamp                    text.ChangeStamp
	docVersion, inputVersion                uint64
	docSelectionGen, inputSelectionGen      uint64
	docSelection, inputSelection            text.Range
	docHistory, inputHistory                text.HistoryStatus
	docGrouped, inputGrouped                bool
	terminalSeal                            editorTerminalSeal
	sourceSession, sourceActivation         uint64
	sourceRequest, sourcePresentation       uint64
	sourceAutoSequence, sourceAutoCancel    uint64
	sourceOwner                             *sourceLanguageOwner
	sourcePending                           *sourceLanguagePending
	sourcePresentationTab                   *Tab
	sourceAutoDesired                      *sourceAutoDescriptor
	sourceIntentAutomatic                   *sourceAutoDescriptor
}

func (a *App) configureControlSnapshot(t *Tab) (configureControlSnapshot, bool) {
	if t == nil || t.isBinary() || !t.textCapable() || !a.tabIsOpen(t) ||
		a.shuttingDown.Load() || a.restoring || t.Doc == nil || t.Editor == nil ||
		t.Doc.Buffer == nil || t.Editor.Buffer == nil || t.Doc.Buffer == t.Editor.Buffer ||
		!t.Editor.ReadOnly || t.Editor.InputActionError() != nil || t.Adapter == nil ||
		t.MatchedPlugin.ID == "" || a.pluginManager == nil {
		return configureControlSnapshot{}, false
	}
	path := t.Doc.Path
	if path == "" {
		path = t.Name
	}
	match, err := a.pluginManager.Match(path)
	if err != nil || match.ID != "code" || match.Manifest.ID != "code" || !match.Manifest.Builtin ||
		!match.Manifest.Enabled || !match.Manifest.Active {
		return configureControlSnapshot{}, false
	}
	oldManifest, err := a.pluginManager.Manifest(t.MatchedPlugin.ID)
	if err != nil || oldManifest.ID != t.MatchedPlugin.ID || oldManifest.Enabled || oldManifest.Active {
		return configureControlSnapshot{}, false
	}
	s := &a.sourceLanguage
	if s.task != nil || s.rename != nil || s.owner != nil && s.owner.tab == t ||
		s.pending != nil && s.pending.owner.tab == t || s.presentationTab == t ||
		s.auto.desired != nil && s.auto.desired.binding.tab == t ||
		s.intent.automatic != nil && s.intent.automatic.binding.tab == t {
		return configureControlSnapshot{}, false
	}
	terminalSeal, sealValid := captureEditorTerminalSeal(t.Editor)
	if !sealValid {
		return configureControlSnapshot{}, false
	}
	docBuffer, inputBuffer := t.Doc.Buffer, t.Editor.Buffer
	snapshot := configureControlSnapshot{
		tab: t, id: t.ID, doc: t.Doc, input: t.Editor, docBuffer: docBuffer, inputBuffer: inputBuffer,
		view: t.View, adapter: t.Adapter, manager: a.pluginManager, match: match,
		activeIndex: a.Active, current: a.Current(), configDir: a.configDir,
		name: t.Name, originalPath: t.OriginalPath, docPath: t.Doc.Path, docEncoding: t.Doc.Encoding,
		docBOM: t.Doc.BOM, docReadOnly: t.Doc.ReadOnly, inputReadOnly: t.Editor.ReadOnly,
		inspector: t.ByteInspector, docIsNew: t.Doc.IsNew, inputBinding: t.inputBinding, inputEpoch: t.inputEpoch,
		route: t.sourceLanguageRoute, profile: t.sourceLanguageProfile, outlineFactory: t.OutlineFactory,
		outlineGeneration: t.OutlineFactoryGeneration, hostEpoch: t.formatHostEpoch,
		docStamp: docBuffer.ChangeStamp(), inputStamp: inputBuffer.ChangeStamp(),
		docVersion: docBuffer.Version(), inputVersion: inputBuffer.Version(),
		docSelectionGen: docBuffer.SelectionGeneration(), inputSelectionGen: inputBuffer.SelectionGeneration(),
		docSelection: docBuffer.Selection(), inputSelection: inputBuffer.Selection(),
		docHistory: docBuffer.History(), inputHistory: inputBuffer.History(),
		docGrouped: docBuffer.InGroup(), inputGrouped: inputBuffer.InGroup(),
		terminalSeal: terminalSeal,
		sourceSession: s.session, sourceActivation: s.activation, sourceRequest: s.request,
		sourcePresentation: s.presentation, sourceAutoSequence: s.auto.sequence, sourceAutoCancel: s.auto.cancel,
		sourceOwner: s.owner, sourcePending: s.pending, sourcePresentationTab: s.presentationTab,
		sourceAutoDesired: s.auto.desired, sourceIntentAutomatic: s.intent.automatic,
	}
	return snapshot, true
}

func (seal editorTerminalSeal) currentExceptRegistration(input *editor.Editor) bool {
	if input == nil || input != seal.editor {
		return false
	}
	current, ok := editorTerminalSealFromValue(input, reflect.ValueOf(*input))
	return ok && seal.equalExceptRegistration(current)
}

func (snapshot configureControlSnapshot) currentAfterCleanup(a *App) bool {
	t, s := snapshot.tab, &a.sourceLanguage
	return t == snapshot.tab && a.tabIsOpen(t) && !a.shuttingDown.Load() && !a.restoring &&
		t.ID == snapshot.id && t.Doc == snapshot.doc && t.Editor == snapshot.input && t.View == snapshot.view &&
		t.Doc.Buffer == snapshot.docBuffer && t.Editor.Buffer == snapshot.inputBuffer &&
		snapshot.docBuffer.ChangeStamp() == snapshot.docStamp && snapshot.inputBuffer.ChangeStamp() == snapshot.inputStamp &&
		snapshot.docBuffer.Version() == snapshot.docVersion && snapshot.inputBuffer.Version() == snapshot.inputVersion &&
		snapshot.docBuffer.SelectionGeneration() == snapshot.docSelectionGen &&
		snapshot.inputBuffer.SelectionGeneration() == snapshot.inputSelectionGen &&
		snapshot.docBuffer.Selection() == snapshot.docSelection && snapshot.inputBuffer.Selection() == snapshot.inputSelection &&
		snapshot.docBuffer.History() == snapshot.docHistory && snapshot.inputBuffer.History() == snapshot.inputHistory &&
		snapshot.docBuffer.InGroup() == snapshot.docGrouped && snapshot.inputBuffer.InGroup() == snapshot.inputGrouped &&
		t.Name == snapshot.name && t.OriginalPath == snapshot.originalPath && t.Doc.Path == snapshot.docPath &&
		t.Doc.Encoding == snapshot.docEncoding && t.Doc.BOM == snapshot.docBOM && t.Doc.ReadOnly == snapshot.docReadOnly &&
		t.Doc.IsNew == snapshot.docIsNew && t.Editor.ReadOnly == snapshot.inputReadOnly && t.ByteInspector == snapshot.inspector &&
		t.Adapter == nil && t.OutlineFactory == nil && t.Mode == "source" && t.MatchedPlugin.ID == snapshot.match.ID &&
		t.MatchedPlugin.Builtin == snapshot.match.Manifest.Builtin && t.inputBinding == nil &&
		t.inputEpoch == snapshot.inputEpoch+1 && t.sourceLanguageRoute == a.sourceLanguage.routeSequence &&
		t.sourceLanguageProfile == "" && t.OutlineFactoryGeneration == a.outlineFactorySequence &&
		t.formatHostEpoch == snapshot.hostEpoch+1 && !t.formatHostActive && t.Editor.Highlight == nil &&
		a.Active == snapshot.activeIndex && a.Current() == snapshot.current && a.configDir == snapshot.configDir &&
		a.pluginManager == snapshot.manager && snapshot.terminalSeal.currentExceptRegistration(snapshot.input) &&
		s.session == snapshot.sourceSession && s.activation == snapshot.sourceActivation &&
		s.request == snapshot.sourceRequest && s.presentation == snapshot.sourcePresentation &&
		s.auto.sequence == snapshot.sourceAutoSequence && s.auto.cancel == snapshot.sourceAutoCancel &&
		s.task == nil && s.rename == nil && s.owner == snapshot.sourceOwner && s.pending == snapshot.sourcePending &&
		s.presentationTab == snapshot.sourcePresentationTab && s.auto.desired == snapshot.sourceAutoDesired &&
		s.intent.automatic == snapshot.sourceIntentAutomatic
}

func (a *App) applyConfigureControlCleanup(snapshot configureControlSnapshot) bool {
	t := snapshot.tab
	a.revokeSourceLanguageShortcut(t)
	a.revokeSourceInput(t)
	t.formatHostEpoch++
	t.formatHostActive = false
	a.sourceLanguage.routeSequence++
	t.sourceLanguageRoute = a.sourceLanguage.routeSequence
	t.sourceLanguageProfile = ""
	t.Adapter = nil
	t.OutlineFactory = nil
	a.outlineFactorySequence++
	t.OutlineFactoryGeneration = a.outlineFactorySequence
	t.Editor.Highlight = nil
	t.MatchedPlugin = snapshot.match.Manifest
	t.Mode = "source"
	t.SyntaxNotice = ""
	warning := "原格式插件已停用；文档缓冲区与输入缓冲区不一致。已撤下格式视图并保留两份内容，仅只读查看；请先保留草稿，再重新打开文档。"
	t.FormatWarning = warning
	t.SourceCapabilityBadge = "只读"
	t.SourceCapabilityDetail = warning
	return snapshot.currentAfterCleanup(a)
}

func (a *App) configureTab(t *Tab) bool {
	if t == nil {
		return true
	}
	if !t.isBinary() && t.textCapable() && t.Doc != nil && t.Editor != nil &&
		t.Doc.Buffer != nil && t.Editor.Buffer != nil && t.Doc.Buffer != t.Editor.Buffer {
		sourceRequest, sourceActivation := a.sourceLanguage.request, a.sourceLanguage.activation
		sourcePresentation := a.sourceLanguage.presentation
		snapshot, ok := a.configureControlSnapshot(t)
		if !ok {
			if a.tabIsOpen(t) && !a.shuttingDown.Load() && !a.restoring &&
				a.sourceLanguage.request == sourceRequest && a.sourceLanguage.activation == sourceActivation &&
				a.sourceLanguage.presentation == sourcePresentation {
				a.report(errors.New("文档与输入附件不一致；未重配置，请先保留草稿并重新打开"))
			}
			return false
		}
		return a.applyConfigureControlCleanup(snapshot)
	}
	if !a.retireSourceLanguage(t) {
		return false
	}
	t.formatHostEpoch++
	t.formatHostActive = false
	a.sourceLanguage.routeSequence++
	t.sourceLanguageRoute = a.sourceLanguage.routeSequence
	t.sourceLanguageProfile = ""
	a.disableSourceFolding(t)
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
		return true
	}
	if t.ByteInspector {
		t.Mode = "source"
		return true
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
		return true
	}
	if match.Manifest.Builtin {
		if match.ID == "image" {
			t.Mode = "source"
			t.FormatWarning = "此标签保留文本草稿；图片扩展名不会把现有文本改成二进制图片。"
			return true
		}
		if adapter, ok := a.adapterByID(match.ID); ok {
			t.Adapter = &adapter
			t.Mode = "preview"
			return true
		}
	}
	t.Mode = "source"
	a.installConfiguredSourceFolding(t)
	a.configureSourceLanguage(t)
	return true
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
	a.finishSetPluginEnabled(id, enabled, a.pluginManager.SetEnabled(id, enabled))
}

// finishSetPluginEnabled reconciles the persistence/registry outcome on the UI
// thread. Keeping this boundary separate also permits instance-local fault tests.
func (a *App) finishSetPluginEnabled(id string, enabled bool, err error) {
	if err != nil {
		var publication *document.StatePublicationError
		published := errors.As(err, &publication) && publication.Committed
		if published || errors.Is(err, plugins.ErrEnabledNotLoaded) {
			if !a.reconfigureTabs() {
				return
			}
		}
		a.report(fmt.Errorf("格式插件 %s：%w", id, err))
		return
	}
	if !a.reconfigureTabs() {
		return
	}
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
	if !a.reconfigureTabs() {
		return
	}
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

func (a *App) reconfigureTabs() bool {
	for _, t := range a.Tabs {
		if t.isBinary() && t == a.Current() {
			if !a.releaseTabPresentation(t, true) {
				return false
			}
			t.View = nil
		} else {
			if !a.disposePresentation(t) {
				return false
			}
		}
		oldMode := t.Mode
		t.Mode = "source"
		if !a.configureTab(t) {
			return false
		}
		if t.Adapter != nil && oldMode != "" {
			t.Mode = oldMode
		}
		a.bindSourceInput(t)
	}
	current := a.sourceLanguageContinuation()
	a.markPersistPending()
	if !current() {
		return false
	}
	if current := a.Current(); current != nil {
		current.Editor.Focus()
	}
	return true
}

func (a *App) flushAllViews(action string) bool {
	for _, t := range a.Tabs {
		if !a.flushTab(t, action) {
			return false
		}
	}
	return true
}
