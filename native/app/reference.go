package app

import (
	"errors"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"levnote.local/native/ui/editor"
)

type referenceKind uint8

const (
	referenceLocal referenceKind = iota
	referenceAnchor
	referenceRejected
)

type referenceResolution struct {
	kind   referenceKind
	path   string
	anchor string
	reason string
}

func rejectedReference(reason string) referenceResolution {
	return referenceResolution{kind: referenceRejected, reason: reason}
}

// resolveLocalReference is intentionally free of filesystem and UI access.
// It never resolves a relative path through the process working directory.
func resolveLocalReference(sourcePath, reference string) referenceResolution {
	if reference == "" {
		return rejectedReference("引用为空，未打开任何内容。")
	}
	if strings.HasPrefix(reference, "#") {
		anchor, err := url.PathUnescape(strings.TrimPrefix(reference, "#"))
		if err != nil {
			return rejectedReference("锚点编码无效，未执行跳转。")
		}
		if anchor == "" {
			return rejectedReference("锚点为空，未执行跳转。")
		}
		return referenceResolution{kind: referenceAnchor, anchor: anchor}
	}

	if hasDoublePathRoot(reference) {
		return rejectedReference("本地版不会打开网络地址或远程 UNC 路径。")
	}
	if filepath.IsAbs(reference) {
		return referenceResolution{kind: referenceLocal, path: filepath.Clean(reference)}
	}
	if strings.HasPrefix(reference, "/") || strings.HasPrefix(reference, `\`) {
		return rejectedReference("本地版不会使用当前盘符解析根相对路径；请使用完整的 Windows 绝对路径。")
	}

	if isWindowsDriveAbsolute(reference) {
		return referenceResolution{kind: referenceLocal, path: cleanWindowsDriveAbsolute(reference)}
	}
	if hasWindowsDrivePrefix(reference) {
		return rejectedReference("本地版不会打开盘符相对路径；请使用完整的 Windows 绝对路径。")
	}
	if scheme := referenceScheme(reference); scheme != "" {
		if !strings.EqualFold(scheme, "file") {
			return rejectedReference("本地版不会打开网络地址、外部 URL 或系统外壳链接。")
		}
		return resolveFileURL(reference)
	}

	if sourcePath == "" {
		return rejectedReference("当前文档尚未保存，无法解析本地相对引用；请先保存文档。")
	}
	if !filepath.IsAbs(sourcePath) {
		return rejectedReference("当前文档路径不是绝对路径，无法安全解析本地引用。")
	}
	return referenceResolution{
		kind: referenceLocal,
		path: filepath.Clean(filepath.Join(filepath.Dir(sourcePath), filepath.FromSlash(reference))),
	}
}

// Reject UNC, device, and mixed-separator double roots before any cleaning.
func hasDoublePathRoot(value string) bool {
	return len(value) >= 2 && (value[0] == '/' || value[0] == '\\') &&
		(value[1] == '/' || value[1] == '\\')
}

// Windows literals keep Windows lexical semantics on every test host.
// The leading slash in the path package prevents .. from escaping the drive.
func cleanWindowsDriveAbsolute(value string) string {
	tail := path.Clean("/" + strings.ReplaceAll(value[3:], `\`, "/"))
	return value[:2] + strings.ReplaceAll(tail, "/", `\`)
}

func isWindowsDriveAbsolute(value string) bool {
	return len(value) >= 3 &&
		isASCIIAlpha(value[0]) && value[1] == ':' &&
		(value[2] == '/' || value[2] == '\\')
}

func hasWindowsDrivePrefix(value string) bool {
	return len(value) >= 2 && isASCIIAlpha(value[0]) && value[1] == ':'
}

func isASCIIAlpha(value byte) bool {
	return value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z'
}

// referenceScheme recognizes RFC 3986 schemes without treating a Windows drive
// such as C:\ as one. Drive prefixes are handled before this helper is called.
func referenceScheme(value string) string {
	colon := strings.IndexByte(value, ':')
	if colon <= 0 {
		return ""
	}
	for index := 0; index < colon; index++ {
		value := value[index]
		letter := isASCIIAlpha(value)
		if index == 0 && !letter {
			return ""
		}
		if !letter && (value < '0' || value > '9') && value != '+' && value != '-' && value != '.' {
			return ""
		}
	}
	return value[:colon]
}

func resolveFileURL(reference string) referenceResolution {
	parsed, err := url.Parse(reference)
	if err != nil {
		return rejectedReference("文件 URL 编码无效，未执行打开。")
	}
	if parsed.Host != "" || parsed.User != nil {
		return rejectedReference("本地版只支持不带主机名的本机文件 URL；远程主机或共享路径不会打开。")
	}
	if parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || parsed.RawFragment != "" || strings.Contains(reference, "#") {
		return rejectedReference("文件 URL 含有查询或片段，未执行打开。")
	}

	decoded := parsed.Path
	if parsed.Opaque != "" {
		decoded, err = url.PathUnescape(parsed.Opaque)
		if err != nil {
			return rejectedReference("文件 URL 编码无效，未执行打开。")
		}
	}
	if decoded == "" {
		return rejectedReference("文件 URL 没有本机文件路径，未执行打开。")
	}
	if strings.ContainsRune(decoded, '\x00') {
		return rejectedReference("文件 URL 含有空字符，未执行打开。")
	}
	if hasDoublePathRoot(decoded) {
		return rejectedReference("本地版不会打开编码后的远程 UNC 路径。")
	}

	// Only /C:/path contributes an extra URL separator. A genuine Unix root
	// stays intact, and the already-decoded path is never unescaped twice.
	if strings.HasPrefix(decoded, "/") && isWindowsDriveAbsolute(decoded[1:]) {
		decoded = decoded[1:]
	}
	if hasDoublePathRoot(decoded) {
		return rejectedReference("本地版不会打开编码后的远程 UNC 路径。")
	}
	if isWindowsDriveAbsolute(decoded) {
		return referenceResolution{kind: referenceLocal, path: cleanWindowsDriveAbsolute(decoded)}
	}
	if !isNativeAbsoluteFilePath(decoded) {
		return rejectedReference("文件 URL 必须指向本机绝对路径。")
	}
	return referenceResolution{kind: referenceLocal, path: filepath.Clean(decoded)}
}

func (a *App) editorHost(t *Tab) editor.Host {
	if t.isBinary() {
		return editor.Host{ReadOnly: true, Path: t.path(), Report: a.report,
			BeginImage: func(owner editor.View, request editor.ImageRequest, complete func(editor.ImageResult, error)) (func(), error) {
				return a.beginImage(t, owner, request, complete)
			},
			ShowSource: func(owner editor.View) bool {
				return t == a.Current() && a.tabIsOpen(t) && t.View == owner && a.setMode(t, "source")
			},
		}
	}
	host := editor.Host{
		Buffer: t.Doc.Buffer, ReadOnly: t.Doc.ReadOnly, Path: t.Doc.Path,
		Changed: t.Editor.OnChange, Report: a.report,
		OpenReference: func(path string) { a.openReference(t, path) },
		RecoverDraft: func(title, content string) error {
			return a.recoverDraft(title, content)
		},
		BeginScene: func(owner editor.View, request editor.SceneRequest, complete func(editor.SceneResult, error)) (func(), error) {
			return a.beginScene(t, owner, request, complete)
		},
		ShowSource: func(owner editor.View) bool {
			return t == a.Current() && a.tabIsOpen(t) && t.View == owner && a.setMode(t, "source")
		},
	}
	if t.Adapter != nil && t.Adapter.SceneEngine != nil {
		capability := *t.Adapter.SceneEngine
		host.SceneEngine = &capability
	}
	return host
}

func (a *App) tabIsOpen(target *Tab) bool {
	for _, tab := range a.Tabs {
		if tab == target {
			return true
		}
	}
	return false
}

func (a *App) openReference(t *Tab, reference string) {
	if t == nil || !a.tabIsOpen(t) {
		a.report(errors.New("引用所属的文档已关闭，未执行打开。"))
		return
	}
	if t != a.Current() {
		a.report(errors.New("引用所属的标签已不再活动，未执行打开。"))
		return
	}
	resolved := resolveLocalReference(t.path(), reference)
	switch resolved.kind {
	case referenceAnchor:
		a.activateOutlineAnchor(t, resolved.anchor, "打开锚点引用")
	case referenceLocal:
		// Open/OpenMany remains the single barrier for flushing the source,
		// ordered navigation and opening or selecting the destination tab.
		a.Open(resolved.path)
	default:
		a.report(errors.New(resolved.reason))
	}
}
