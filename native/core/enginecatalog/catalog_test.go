package enginecatalog

import (
	"testing"

	"levnote.local/native/core/enginescene"
)

func TestFixedDescriptors(t *testing.T) {
	markdown, ok := Lookup(MarkdownHTML)
	if !ok || markdown != (Descriptor{
		Engine: MarkdownHTML, Format: "html", Protocol: ProtocolVersion,
		RelativePath: "engines/markdown-export", Filename: "levnote-markdown-export.exe",
		GOOS: "windows", GOARCH: "amd64",
		SourceMaxBytes: MarkdownSourceMaxBytes, PayloadMaxBytes: MarkdownPayloadMaxBytes,
	}) || markdown.SourceMaxBytes != 8<<20 || markdown.PayloadMaxBytes != 64<<20 {
		t.Fatalf("markdown descriptor=%+v ok=%v", markdown, ok)
	}
	mindmap, ok := Lookup(MindmapSVG)
	if !ok || mindmap != (Descriptor{
		Engine: MindmapSVG, Format: "svg", Protocol: ProtocolVersion,
		RelativePath: "engines/mindmap-svg", Filename: "levnote-mindmap-svg.exe",
		GOOS: "windows", GOARCH: "amd64",
		SourceMaxBytes: MindmapSourceMaxBytes, PayloadMaxBytes: MindmapPayloadMaxBytes,
	}) || mindmap.SourceMaxBytes != 16<<20 {
		t.Fatalf("mindmap descriptor=%+v ok=%v", mindmap, ok)
	}

	preview, ok := Lookup(MindmapPreview)
	if !ok || preview != (Descriptor{
		Engine: MindmapPreview, Format: "mindmap-scene-v1", Protocol: ProtocolVersion,
		RelativePath: "engines/mindmap-preview", Filename: "levnote-mindmap-preview.exe",
		GOOS: "windows", GOARCH: "amd64",
		SourceMaxBytes: MindmapSourceMaxBytes, PayloadMaxBytes: enginescene.MaxPayloadBytes,
	}) || preview.SourceMaxBytes != 16<<20 || preview.PayloadMaxBytes != 32<<20 {
		t.Fatalf("preview descriptor=%+v ok=%v", preview, ok)
	}

	drawio, ok := Lookup(DrawioSVG)
	if !ok || drawio != (Descriptor{
		Engine: DrawioSVG, Format: "svg", Protocol: ProtocolVersion,
		RelativePath: "engines/drawio-svg", Filename: "levnote-drawio-svg.exe",
		GOOS: "windows", GOARCH: "amd64",
		SourceMaxBytes: DrawioSourceMaxBytes, PayloadMaxBytes: DrawioPayloadMaxBytes,
	}) || drawio.SourceMaxBytes != 32<<20 || drawio.PayloadMaxBytes != 64<<20 {
		t.Fatalf("drawio descriptor=%+v ok=%v", drawio, ok)
	}
}

func TestLookupAndMatchRejectPartialIdentity(t *testing.T) {
	if _, ok := Lookup(EngineID("unknown")); ok {
		t.Fatal("unknown engine accepted")
	}
	if _, ok := Match(MarkdownHTML, "svg", "windows", "amd64"); ok {
		t.Fatal("cross-engine format accepted")
	}
	if _, ok := Match(MindmapSVG, "svg", "linux", "amd64"); ok {
		t.Fatal("unsupported platform accepted")
	}
	if _, ok := Match(MindmapSVG, "svg", "windows", "arm64"); ok {
		t.Fatal("unsupported architecture accepted")
	}
	if _, ok := Match(MindmapPreview, "mindmap-scene-v1", "windows", "amd64"); !ok {
		t.Fatal("preview descriptor rejected")
	}
	if _, ok := Match(DrawioSVG, "svg", "windows", "amd64"); !ok {
		t.Fatal("drawio descriptor rejected")
	}
	for _, test := range []struct {
		name, format, goos, goarch string
	}{
		{"cross-format", "svg", "windows", "amd64"},
		{"cross-engine-format", "html", "windows", "amd64"},
		{"wrong-os", "mindmap-scene-v1", "linux", "amd64"},
		{"wrong-arch", "mindmap-scene-v1", "windows", "arm64"},
	} {
		if _, ok := Match(MindmapPreview, test.format, test.goos, test.goarch); ok {
			t.Fatalf("preview %s accepted", test.name)
		}
	}
}
