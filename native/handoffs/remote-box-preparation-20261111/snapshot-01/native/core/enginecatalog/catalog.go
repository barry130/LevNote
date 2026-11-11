// Package enginecatalog owns the fixed repository engine identities. It has no
// process, file, registry, or plugin state and accepts no caller descriptors.
package enginecatalog

import "levnote.local/native/core/enginescene"

type EngineID string

const (
	MarkdownHTML   EngineID = "levnote.markdown-export"
	MindmapSVG     EngineID = "levnote.mindmap-svg"
	MindmapPreview EngineID = "levnote.mindmap-preview"
	DrawioSVG      EngineID = "levnote.drawio-svg"
)

const ProtocolVersion = 1

const (
	MarkdownSourceMaxBytes  = 8 << 20
	MarkdownPayloadMaxBytes = 64 << 20
	MindmapSourceMaxBytes   = 16 << 20
	MindmapPayloadMaxBytes  = 64 << 20
	DrawioSourceMaxBytes    = 32 << 20
	DrawioPayloadMaxBytes   = 64 << 20
)

type Descriptor struct {
	Engine          EngineID
	Format          string
	Protocol        int
	RelativePath    string
	Filename        string
	GOOS, GOARCH    string
	SourceMaxBytes  int64
	PayloadMaxBytes int64
}

var descriptors = [...]Descriptor{
	{
		Engine: MarkdownHTML, Format: "html", Protocol: ProtocolVersion,
		RelativePath: "engines/markdown-export", Filename: "levnote-markdown-export.exe",
		GOOS: "windows", GOARCH: "amd64",
		SourceMaxBytes: MarkdownSourceMaxBytes, PayloadMaxBytes: MarkdownPayloadMaxBytes,
	},
	{
		Engine: MindmapSVG, Format: "svg", Protocol: ProtocolVersion,
		RelativePath: "engines/mindmap-svg", Filename: "levnote-mindmap-svg.exe",
		GOOS: "windows", GOARCH: "amd64",
		SourceMaxBytes: MindmapSourceMaxBytes, PayloadMaxBytes: MindmapPayloadMaxBytes,
	},
	{
		Engine: MindmapPreview, Format: "mindmap-scene-v1", Protocol: ProtocolVersion,
		RelativePath: "engines/mindmap-preview", Filename: "levnote-mindmap-preview.exe",
		GOOS: "windows", GOARCH: "amd64",
		SourceMaxBytes: MindmapSourceMaxBytes, PayloadMaxBytes: enginescene.MaxPayloadBytes,
	},
	{
		Engine: DrawioSVG, Format: "svg", Protocol: ProtocolVersion,
		RelativePath: "engines/drawio-svg", Filename: "levnote-drawio-svg.exe",
		GOOS: "windows", GOARCH: "amd64",
		SourceMaxBytes: DrawioSourceMaxBytes, PayloadMaxBytes: DrawioPayloadMaxBytes,
	},
}

func Lookup(engine EngineID) (Descriptor, bool) {
	for _, descriptor := range descriptors {
		if descriptor.Engine == engine {
			return descriptor, true
		}
	}
	return Descriptor{}, false
}

func Match(engine EngineID, format, goos, goarch string) (Descriptor, bool) {
	descriptor, ok := Lookup(engine)
	if !ok || descriptor.Format != format || descriptor.GOOS != goos || descriptor.GOARCH != goarch {
		return Descriptor{}, false
	}
	return descriptor, true
}
