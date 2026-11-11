// Command levnote-mindmap-preview serves one bounded native-scene response.
// The host owns files, process lifetime and the eventual native presentation.
package main

import (
	"errors"
	"io"
	"os"
	"unicode/utf8"

	"levnote.local/native/core/enginecatalog"
	"levnote.local/native/core/engineipc"
	"levnote.local/native/core/enginescene"
	"levnote.local/native/formats/mindmap"
)

// Packaging supplies -X main.engineVersion=<version>. No file or environment
// lookup participates in the command's identity.
var engineVersion = "development"

func main() { os.Exit(serve(os.Stdin, os.Stdout)) }

func serve(input io.Reader, output io.Writer) int {
	if !engineipc.ValidEngineVersion(engineVersion) {
		return 2
	}
	meta := engineipc.ResponseMeta{
		Engine: engineipc.MindmapPreview, EngineVersion: engineVersion,
		Format: "mindmap-scene-v1", Status: engineipc.StatusOK,
		Warnings: []engineipc.Warning{},
	}
	_, source, err := engineipc.ReadRequestFor(engineipc.MindmapPreview, input)
	var payload []byte
	if err != nil {
		code := "invalid-request"
		var failed *engineipc.Failure
		if errors.As(err, &failed) && (failed.Code == "source-limit" || failed.Code == "invalid-utf8") {
			code = failed.Code
		}
		meta.Error = &engineipc.EngineError{Code: code, Message: "Mindmap preview request was refused"}
	} else {
		payload, meta.Warnings, meta.Error = render(source)
	}
	if meta.Error != nil {
		meta.Status = engineipc.StatusError
	}
	if err := engineipc.WriteResponse(output, meta, payload); err != nil {
		return 1
	}
	return 0
}

func render(source []byte) ([]byte, []engineipc.Warning, *engineipc.EngineError) {
	refuse := func(code, message string) ([]byte, []engineipc.Warning, *engineipc.EngineError) {
		return nil, []engineipc.Warning{}, &engineipc.EngineError{Code: code, Message: message}
	}
	descriptor, ok := enginecatalog.Lookup(engineipc.MindmapPreview)
	if !ok || descriptor.Format != "mindmap-scene-v1" {
		return refuse("render-failed", "Mindmap preview is not available")
	}
	if int64(len(source)) > descriptor.SourceMaxBytes {
		return refuse("source-limit", "Mindmap source exceeds the preview limit")
	}
	if !utf8.Valid(source) {
		return refuse("invalid-utf8", "Mindmap source is not valid UTF8")
	}
	document, parsedIssues, err := mindmap.Parse(source, mindmap.ParseOptions{MaxBytes: int(descriptor.SourceMaxBytes)})
	if err != nil {
		if errors.Is(err, mindmap.ErrLimitExceeded) {
			return refuse("source-limit", "Mindmap source exceeds the parser limits")
		}
		return refuse("render-failed", "Mindmap source or schema is invalid")
	}
	layout := mindmap.Layout(document, mindmap.NativeLayoutOptions())
	for _, issue := range layout.Issues {
		if issue.Severity == mindmap.SeverityError {
			return refuse("render-failed", "Mindmap layout failed")
		}
	}
	warnings, err := wireWarnings(len(source), parsedIssues, layout.Issues, mindmap.PresentationIssues(document))
	if err != nil {
		return refuse("warning-limit", "Mindmap warning metadata exceeds the preview bounds")
	}
	payload, err := enginescene.Encode(previewScene(document, layout))
	if errors.Is(err, enginescene.ErrLimitExceeded) {
		return refuse("output-limit", "Native scene exceeds the preview limit")
	}
	if err != nil || payload == nil {
		return refuse("render-failed", "Mindmap preview could not be produced completely")
	}
	return payload, warnings, nil
}

const previewImageMessage = "节点图片引用已保留；此只读预览不加载或显示图片，完整引用请在源码中查看"

func wireWarnings(sourceBytes int, groups ...[]mindmap.Issue) ([]engineipc.Warning, error) {
	converted := make([]engineipc.Warning, 0)
	for _, issues := range groups {
		for _, issue := range issues {
			if issue.Severity != mindmap.SeverityWarning {
				return nil, errors.New("non-warning issue")
			}
			message := issue.Message
			if issue.Code == mindmap.IssueImagePlaceholder {
				// The neutral scene has no image or badge field. Keep the issue's
				// position/code while accurately describing this producer.
				message = previewImageMessage
			}
			if len(converted) == engineipc.WarningsMaxCount ||
				len(issue.Code) == 0 || len(issue.Code) > engineipc.WarningCodeMaxBytes ||
				len(message) > engineipc.WarningMessageMaxBytes ||
				!utf8.ValidString(string(issue.Code)) || !utf8.ValidString(message) ||
				sourceBytes < 0 || uint64(sourceBytes) > uint64(^uint32(0)) {
				return nil, errors.New("warning bounds")
			}
			converted = append(converted, engineipc.Warning{
				Code: string(issue.Code), Message: message, Start: 0, End: uint32(sourceBytes),
			})
		}
	}
	return converted, nil
}
