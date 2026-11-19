// Command levnote-drawio-svg serves exactly one bounded draw.io SVG request.
// The host owns all file access, process lifetime and final publication.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"unicode/utf8"

	"levnote.local/native/core/enginecatalog"
	"levnote.local/native/core/engineipc"
	"levnote.local/native/formats/drawio"
)

// Packaging supplies -X main.engineVersion=<version>. The standalone default
// remains a valid explicit version, never a manifest read or environment lookup.
var engineVersion = "development"

func main() { os.Exit(serve(os.Stdin, os.Stdout)) }

func serve(input io.Reader, output io.Writer) int {
	return serveWithLimit(input, output, drawio.MaxSVGBytes)
}

// The private limit seam supports small boundary fixtures. Production enters
// through serve with the accepted fixed SVG cap.
func serveWithLimit(input io.Reader, output io.Writer, svgLimit int) int {
	meta := engineipc.ResponseMeta{
		Engine:        engineipc.DrawioSVG,
		EngineVersion: engineVersion,
		Format:        "svg",
		Status:        engineipc.StatusOK,
		Warnings:      []engineipc.Warning{},
	}
	if !engineipc.ValidEngineVersion(engineVersion) {
		return 2
	}
	request, source, err := engineipc.ReadRequestFor(engineipc.DrawioSVG, input)
	var svg []byte
	if err != nil {
		code := "invalid-request"
		var failed *engineipc.Failure
		if errors.As(err, &failed) && (failed.Code == "source-limit" || failed.Code == "invalid-utf8") {
			code = failed.Code
		}
		setError(&meta, code, "draw.io SVG export request was refused")
	} else if request.Page == nil {
		setError(&meta, "invalid-request", "draw.io page metadata is required")
	} else {
		svg, meta.Warnings, meta.Error = renderWithLimit(source, *request.Page, svgLimit)
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

func render(source []byte, page engineipc.RequestPage) ([]byte, []engineipc.Warning, *engineipc.EngineError) {
	return renderWithLimit(source, page, drawio.MaxSVGBytes)
}

func renderWithLimit(source []byte, page engineipc.RequestPage, svgLimit int) ([]byte, []engineipc.Warning, *engineipc.EngineError) {
	refuse := func(code, message string) ([]byte, []engineipc.Warning, *engineipc.EngineError) {
		return nil, []engineipc.Warning{}, &engineipc.EngineError{Code: code, Message: message}
	}
	descriptor, ok := enginecatalog.Lookup(engineipc.DrawioSVG)
	if !ok || descriptor.Format != "svg" {
		return refuse("render-failed", "draw.io output is not available")
	}
	if len(source) > int(descriptor.SourceMaxBytes) {
		return refuse("source-limit", "draw.io source exceeds the export limit")
	}
	if !utf8.Valid(source) {
		return refuse("invalid-utf8", "draw.io source is not valid UTF8")
	}
	if page.Index < 0 || page.Index > engineipc.PageIndexMax || len(page.ID) > engineipc.PageIDMaxBytes || !utf8.ValidString(page.ID) {
		return refuse("invalid-page", "draw.io page identity is invalid")
	}

	document, parsedIssues, err := drawio.Parse(source, drawio.ParseOptions{MaxBytes: int(descriptor.SourceMaxBytes)})
	if err != nil {
		if errors.Is(err, drawio.ErrLimitExceeded) {
			return refuse("source-limit", "draw.io source exceeds the parser limits")
		}
		if errors.Is(err, drawio.ErrPageNotFound) {
			return refuse("invalid-page", "draw.io page is unavailable")
		}
		return refuse("render-failed", "draw.io source or page is invalid")
	}
	if page.Index >= len(document.Pages) || document.Pages[page.Index].ID != page.ID {
		return refuse("invalid-page", "draw.io page index and identity do not match")
	}

	warnings, warningErr := wireParseWarnings(
		len(source), page.Index, engineipc.WarningsMaxCount, parsedIssues,
	)
	if warningErr != nil {
		return refuse("warning-limit", "draw.io warning metadata exceeds the export bounds")
	}

	svg, emittedIssues, exportErr := document.ExportSVG(page.Index, drawio.SVGOptions{MaxBytes: svgLimit})
	if errors.Is(exportErr, drawio.ErrSVGOutputLimit) {
		return refuse("output-limit", "draw.io SVG output exceeds the export limit")
	}
	if exportErr != nil || svg == nil {
		return refuse("render-failed", "draw.io page could not be exported completely")
	}
	emitted, warningErr := wireWarnings(
		len(source), engineipc.WarningsMaxCount-len(warnings), emittedIssues...,
	)
	if warningErr != nil {
		return refuse("warning-limit", "draw.io warning metadata exceeds the export bounds")
	}
	warnings = append(warnings, emitted...)
	return svg, warnings, nil
}

func wireParseWarnings(sourceBytes, page, limit int, issues []drawio.Issue) ([]engineipc.Warning, error) {
	if !validWarningCount(sourceBytes, limit) {
		return nil, errors.New("warning source bounds")
	}
	count := 0
	for _, issue := range issues {
		if issue.Page != drawio.DocumentPage && issue.Page != page {
			continue
		}
		count++
		if count > limit {
			return nil, errors.New("warning count bounds")
		}
	}
	converted := make([]engineipc.Warning, 0, count)
	for _, issue := range issues {
		if issue.Page != drawio.DocumentPage && issue.Page != page {
			continue
		}
		warning, err := wireWarning(sourceBytes, issue)
		if err != nil {
			return nil, err
		}
		converted = append(converted, warning)
	}
	return converted, nil
}

func wireWarnings(sourceBytes, limit int, issues ...drawio.Issue) ([]engineipc.Warning, error) {
	if !validWarningCount(sourceBytes, limit) || len(issues) > limit {
		return nil, errors.New("warning count bounds")
	}
	converted := make([]engineipc.Warning, 0, len(issues))
	for _, issue := range issues {
		warning, err := wireWarning(sourceBytes, issue)
		if err != nil {
			return nil, err
		}
		converted = append(converted, warning)
	}
	return converted, nil
}

func validWarningCount(sourceBytes, limit int) bool {
	return sourceBytes >= 0 && uint64(sourceBytes) <= uint64(^uint32(0)) &&
		limit >= 0 && limit <= engineipc.WarningsMaxCount
}

func wireWarning(sourceBytes int, issue drawio.Issue) (engineipc.Warning, error) {
	if issue.Severity != drawio.SeverityWarning {
		return engineipc.Warning{}, errors.New("warning severity")
	}
	code := string(issue.Code)
	if len(code) == 0 || len(code) > engineipc.WarningCodeMaxBytes || !utf8.ValidString(code) ||
		len(issue.Message) > engineipc.WarningMessageMaxBytes || !utf8.ValidString(issue.Message) ||
		len(issue.CellID) > engineipc.WarningMessageMaxBytes || !utf8.ValidString(issue.CellID) {
		return engineipc.Warning{}, errors.New("warning component bounds")
	}
	prefix := warningPrefix(issue)
	if len(prefix) > engineipc.WarningMessageMaxBytes-len(issue.Message) {
		return engineipc.Warning{}, errors.New("warning message bounds")
	}
	return engineipc.Warning{
		Code: code, Message: prefix + issue.Message, Start: 0, End: uint32(sourceBytes),
	}, nil
}

func warningPrefix(issue drawio.Issue) string {
	scope := "document"
	if issue.Page != drawio.DocumentPage {
		scope = fmt.Sprintf("page %d", issue.Page)
	}
	if issue.CellID != "" {
		scope += " cell " + issue.CellID
	}
	return scope + ": "
}
