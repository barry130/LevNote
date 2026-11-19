// Command levnote-mindmap-svg serves exactly one bounded stdin/stdout frame.
// The host owns all file access, process lifetime and final file publication.
package main

import (
	"errors"
	"io"
	"os"
	"unicode/utf8"

	"levnote.local/native/core/enginecatalog"
	"levnote.local/native/core/engineipc"
	"levnote.local/native/formats/mindmap"
)

// Packaging supplies -X main.engineVersion=<version>. The standalone default
// remains a valid explicit version, never a manifest read or environment lookup.
var engineVersion = "development"

func main() { os.Exit(serve(os.Stdin, os.Stdout)) }

func serve(input io.Reader, output io.Writer) int {
	return serveWithLimit(input, output, mindmap.MaxSVGBytes)
}

// The private budget seam supports small boundary fixtures, just as it does
// for the Markdown worker. Production enters through serve with the fixed cap.
func serveWithLimit(input io.Reader, output io.Writer, svgLimit int) int {
	meta := engineipc.ResponseMeta{
		Engine:        engineipc.MindmapSVG,
		EngineVersion: engineVersion,
		Format:        "svg",
		Status:        engineipc.StatusOK,
		Warnings:      []engineipc.Warning{},
	}
	if !engineipc.ValidEngineVersion(engineVersion) {
		return 2
	}
	_, source, err := engineipc.ReadRequestFor(engineipc.MindmapSVG, input)
	var svg []byte
	if err != nil {
		code := "invalid-request"
		var failed *engineipc.Failure
		if errors.As(err, &failed) && (failed.Code == "source-limit" || failed.Code == "invalid-utf8") {
			code = failed.Code
		}
		setError(&meta, code, "Mindmap SVG export request was refused")
	} else {
		svg, meta.Warnings, meta.Error = renderWithLimit(source, svgLimit)
		if meta.Error != nil {
			meta.Status = engineipc.StatusError
		}
	}
	if err := engineipc.WriteResponse(output, meta, svg); err != nil {
		return 1
	}
	return 0
}

func setError(meta *engineipc.ResponseMeta, code, message string) {
	meta.Status = engineipc.StatusError
	meta.Warnings = []engineipc.Warning{}
	meta.Error = &engineipc.EngineError{Code: code, Message: message}
}

func render(source []byte) ([]byte, []engineipc.Warning, *engineipc.EngineError) {
	return renderWithLimit(source, mindmap.MaxSVGBytes)
}

func renderWithLimit(source []byte, svgLimit int) ([]byte, []engineipc.Warning, *engineipc.EngineError) {
	warnings := []engineipc.Warning{}
	refuse := func(code, message string) ([]byte, []engineipc.Warning, *engineipc.EngineError) {
		return nil, warnings, &engineipc.EngineError{Code: code, Message: message}
	}
	descriptor, ok := enginecatalog.Lookup(engineipc.MindmapSVG)
	if !ok || descriptor.Format != "svg" {
		return refuse("render-failed", "Mindmap output is not available")
	}
	if len(source) > int(descriptor.SourceMaxBytes) {
		return refuse("source-limit", "Mindmap source exceeds the export limit")
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
	presentation := mindmap.PresentationIssues(document)
	converted, warningErr := wireWarnings(len(source), parsedIssues, layout.Issues, presentation)
	if warningErr != nil {
		return refuse("warning-limit", "Mindmap warning metadata exceeds the export bounds")
	}
	svg, exportErr := mindmap.ExportSVG(document, layout, svgLimit)
	if errors.Is(exportErr, mindmap.ErrSVGOutputLimit) {
		return refuse("output-limit", "SVG output exceeds the export limit")
	}
	if exportErr != nil || svg == nil {
		return refuse("render-failed", "Mindmap could not be exported completely")
	}
	return svg, converted, nil
}

func wireWarnings(sourceBytes int, groups ...[]mindmap.Issue) ([]engineipc.Warning, error) {
	converted := make([]engineipc.Warning, 0)
	for _, issues := range groups {
		for _, issue := range issues {
			if issue.Severity != mindmap.SeverityWarning {
				return nil, errors.New("non-warning issue")
			}
			if len(converted) == engineipc.WarningsMaxCount ||
				len(issue.Code) == 0 || len(issue.Code) > engineipc.WarningCodeMaxBytes ||
				len(issue.Message) > engineipc.WarningMessageMaxBytes ||
				!utf8.ValidString(string(issue.Code)) || !utf8.ValidString(issue.Message) ||
				sourceBytes < 0 || sourceBytes > int(^uint32(0)) {
				return nil, errors.New("warning bounds")
			}
			converted = append(converted, engineipc.Warning{
				Code:    string(issue.Code),
				Message: issue.Message,
				Start:   0,
				End:     uint32(sourceBytes),
			})
		}
	}
	return converted, nil
}
