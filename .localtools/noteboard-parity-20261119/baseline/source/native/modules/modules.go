// Package modules is the sole composition root for native format views.
// The workspace depends on editor.Adapter, not on a parser or rendering engine.
package modules

import (
	"context"
	"errors"
	"io"

	"levnote.local/native/core/plugins"
	"levnote.local/native/formats/raster"
	"levnote.local/native/formats/sourceoutline"
	"levnote.local/native/modules/graphview"
	"levnote.local/native/modules/imageview"
	"levnote.local/native/modules/markdownview"
	"levnote.local/native/modules/sourceview"
	"levnote.local/native/ui/editor"
)

func Builtins() []editor.Adapter {
	return []editor.Adapter{
		markdownview.Adapter(),
		graphview.MindmapAdapter(),
		graphview.DrawioAdapter(),
		imageview.Adapter(),
	}
}

// DecodeImage is the neutral composition seam from the host to the shipped
// raster codecs. The host owns and validates its immutable binary snapshot;
// this function opens no file, starts no goroutine, and retains neither the
// reader nor the source bytes. On success Pixels ownership transfers to the
// caller without another full-size copy.
func DecodeImage(ctx context.Context, r io.ReaderAt, size int64) (editor.ImageResult, error) {
	decoded, err := raster.Decode(ctx, r, size)
	if err != nil {
		return editor.ImageResult{}, err
	}
	format := ""
	switch decoded.Kind {
	case raster.KindPNG:
		format = "PNG"
	case raster.KindJPEG:
		format = "JPEG"
	default:
		decoded.Pixels = nil
		return editor.ImageResult{}, errUnknownRasterKind
	}
	return editor.ImageResult{
		Format:          format,
		Width:           decoded.Width,
		Height:          decoded.Height,
		SourceWidth:     decoded.SourceWidth,
		SourceHeight:    decoded.SourceHeight,
		Orientation:     decoded.Orientation,
		CompressedBytes: size,
		Pixels:          decoded.Pixels,
	}, nil
}

var errUnknownRasterKind = errors.New("raster decode returned an unknown kind")

// NativeRouteAdapter resolves only shipped capabilities. Each call returns a
// fresh adapter, independent of registry enablement and installed manifest IDs.
func NativeRouteAdapter(route plugins.NativeRoute) (editor.Adapter, bool) {
	switch route {
	case plugins.NativeRouteMarkdown:
		return markdownview.Adapter(), true
	case plugins.NativeRouteMindmap:
		return graphview.MindmapAdapter(), true
	case plugins.NativeRouteDrawio:
		return graphview.DrawioAdapter(), true
	default:
		return editor.Adapter{}, false
	}
}

// InstalledSourceOutlineFactory is the sole composition seam between the
// plugin loader DTO and the neutral source-outline provider. The app stores and
// invokes the returned generic factory without importing a concrete format
// engine or source-outline package.
func InstalledSourceOutlineFactory(pluginID string, syntax plugins.SyntaxManifest) func(editor.Host) editor.OutlineSource {
	if syntax.Schema != 2 || len(syntax.Outline) == 0 {
		return nil
	}
	rules := make([]sourceoutline.Rule, 0, len(syntax.Outline))
	for _, rule := range syntax.Outline {
		rules = append(rules, sourceoutline.Rule{
			Prefix: rule.Prefix,
			Suffix: rule.Suffix,
			Level:  rule.Level,
		})
	}
	config := sourceoutline.Config{
		PluginID: pluginID,
		Schema:   syntax.Schema,
		Rules:    rules,
	}
	return func(host editor.Host) editor.OutlineSource {
		return sourceview.NewOutline(host, config)
	}
}
