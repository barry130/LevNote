// Command levnote-markdown-export serves exactly one bounded stdin/stdout frame.
// The host owns all file access, process lifetime and final file publication.
package main

import (
	"errors"
	"io"
	"os"
	"unicode/utf8"

	"levnote.local/native/core/engineipc"
	"levnote.local/native/formats/markdown"
)

// Packaging supplies -X main.engineVersion=<version>. The standalone default
// remains a valid explicit version, never a manifest read or environment lookup.
var engineVersion = "development"

func main() { os.Exit(serve(os.Stdin, os.Stdout)) }

func serve(input io.Reader, output io.Writer) int {
	return serveWithLimit(input, output, engineipc.HTMLMaxBytes)
}

// The private budget seam supports small boundary fixtures. Production always
// enters through serve with the fixed protocol cap; no flag or environment knob.
func serveWithLimit(input io.Reader, output io.Writer, htmlLimit int) int {
	meta := engineipc.ResponseMeta{Engine: engineipc.MarkdownHTML, EngineVersion: engineVersion, Format: "html", Status: engineipc.StatusOK, Warnings: []engineipc.Warning{}}
	if !engineipc.ValidEngineVersion(engineVersion) {
		return 2
	}
	_, source, err := engineipc.ReadRequest(input)
	var html []byte
	if err != nil {
		code := "invalid-request"
		var failed *engineipc.Failure
		if errors.As(err, &failed) && (failed.Code == "source-limit" || failed.Code == "invalid-utf8") {
			code = failed.Code
		}
		setError(&meta, code, "Markdown export request was refused")
	} else {
		html, meta.Warnings, meta.Error = renderWithLimit(source, htmlLimit)
		if meta.Error != nil {
			meta.Status = engineipc.StatusError
		}
	}
	if err := engineipc.WriteResponse(output, meta, html); err != nil {
		return 1
	}
	return 0
}

func setError(meta *engineipc.ResponseMeta, code, message string) {
	meta.Status = engineipc.StatusError
	meta.Error = &engineipc.EngineError{Code: code, Message: message}
}

func render(source []byte) ([]byte, []engineipc.Warning, *engineipc.EngineError) {
	return renderWithLimit(source, engineipc.HTMLMaxBytes)
}

func renderWithLimit(source []byte, htmlLimit int) ([]byte, []engineipc.Warning, *engineipc.EngineError) {
	empty := []engineipc.Warning{}
	refuse := func(code, message string) ([]byte, []engineipc.Warning, *engineipc.EngineError) {
		return nil, empty, &engineipc.EngineError{Code: code, Message: message}
	}
	if len(source) > engineipc.SourceMaxBytes {
		return refuse("source-limit", "Markdown source exceeds the export limit")
	}
	if !utf8.Valid(source) {
		return refuse("invalid-utf8", "Markdown source is not valid UTF8")
	}
	doc := markdown.Parse(string(source))
	html, warnings, renderErr := markdown.ExportHTMLBytes(doc, htmlLimit)
	if errors.Is(renderErr, markdown.ErrOutputLimit) {
		return refuse("output-limit", "HTML output exceeds the export limit")
	}
	converted, err := wireWarnings(warnings, len(source))
	if err != nil {
		return refuse("warning-limit", "Markdown warning metadata exceeds the export bounds")
	}
	if renderErr != nil {
		code := "render-failed"
		if errors.Is(renderErr, markdown.ErrPartialDocument) {
			code = "partial-document"
		}
		return nil, converted, &engineipc.EngineError{Code: code, Message: "Markdown could not be exported completely"}
	}
	return html, converted, nil
}

func wireWarnings(warnings []markdown.Warning, sourceBytes int) ([]engineipc.Warning, error) {
	if len(warnings) > engineipc.WarningsMaxCount {
		return nil, errors.New("warning count")
	}
	converted := make([]engineipc.Warning, 0, len(warnings))
	for _, warning := range warnings {
		if warning.Range.Start < 0 || warning.Range.End < warning.Range.Start || warning.Range.End > sourceBytes ||
			len(warning.Code) == 0 || len(warning.Code) > engineipc.WarningCodeMaxBytes || len(warning.Message) > engineipc.WarningMessageMaxBytes ||
			!utf8.ValidString(warning.Code) || !utf8.ValidString(warning.Message) {
			return nil, errors.New("warning bounds")
		}
		converted = append(converted, engineipc.Warning{Code: warning.Code, Message: warning.Message, Start: uint32(warning.Range.Start), End: uint32(warning.Range.End)})
	}
	return converted, nil
}
