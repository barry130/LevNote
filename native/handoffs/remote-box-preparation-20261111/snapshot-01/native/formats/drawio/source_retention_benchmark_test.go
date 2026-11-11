package drawio

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
)

const sourceRetentionBenchmarkPages = 8

func sourceRetentionBenchmarkSource() []byte {
	var source strings.Builder
	source.WriteString("\r\n<mxfile host=\"local\" modified=\"2026-11-10T00:00:00.000Z\">\r\n")
	source.WriteString("  <unknown file-level=\"keep-me\"><child value=\"1.00\"/></unknown>\r\n")
	for page := 0; page < sourceRetentionBenchmarkPages-1; page++ {
		fmt.Fprintf(&source, "  <diagram id=\"bench-%d\" name=\"Benchmark %d\">\r\n", page, page)
		fmt.Fprintf(&source, "    <mxGraphModel dx=\"800\" dy=\"600\" page=\"1\" pageWidth=\"850\" pageHeight=\"1100\">\r\n")
		source.WriteString("      <root>\r\n")
		source.WriteString("        <mxCell id=\"0\"/>\r\n")
		source.WriteString("        <mxCell id=\"1\" parent=\"0\"/>\r\n")
		for node := 0; node < 24; node++ {
			fmt.Fprintf(&source,
				"        <mxCell id=\"p%d-n%d\" value=\"Node %d\" style=\"rounded=1;whiteSpace=wrap;html=0;\" vertex=\"1\" parent=\"1\"><mxGeometry x=\"%d\" y=\"%d\" width=\"120\" height=\"60\" as=\"geometry\"/></mxCell>\r\n",
				page, node, node, 20+(node%4)*180, 80+(node/4)*100)
		}
		fmt.Fprintf(&source,
			"        <unknown page=\"%d\" extension=\"123.00\"><nested huge=\"123456789012345678901234567890\"/></unknown>"+
				"\r\n",
			page)
		source.WriteString("      </root>\r\n    </mxGraphModel>\r\n  </diagram>\r\n")
	}

	compressedModel := `<mxGraphModel dx="800" dy="600"><root><mxCell id="0"/><mxCell id="1" parent="0"/><mxCell id="compressed-node" value="Compressed" vertex="1" parent="1"><mxGeometry x="5" y="6" width="100" height="40" as="geometry"/></mxCell><unknown compressed="123.00"/></root></mxGraphModel>`
	payload := compressRawDeflate(compressedModel)
	source.WriteString(`  <diagram id="compressed" name="Compressed">`)
	source.WriteString(payload)
	source.WriteString(`</diagram>`)
	source.WriteString("</mxfile>\r\n")
	return []byte(source.String())
}

func TestDrawioSourceRetentionBenchmarkFixture(t *testing.T) {
	document, _, err := Parse(sourceRetentionBenchmarkSource(), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(document.Pages) != 8 {
		t.Fatalf("pages = %d, want 8", len(document.Pages))
	}
	if len(document.Pages[0].Model.Nodes) != 24 {
		t.Fatalf("first page nodes = %d, want 24", len(document.Pages[0].Model.Nodes))
	}
	compressed := document.Pages[len(document.Pages)-1]
	if !compressed.Compressed || !compressed.ReadOnly || findNode(compressed.Model.Nodes, "compressed-node") == nil {
		t.Fatalf("last page is not the expected compressed read-only page: %#v", compressed)
	}
}

func BenchmarkDrawioSourceRetention(b *testing.B) {
	source := sourceRetentionBenchmarkSource()
	b.Run("ParseRetained", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(source)))
		for range b.N {
			document, _, err := Parse(source, ParseOptions{})
			if err != nil {
				b.Fatal(err)
			}
			runtime.KeepAlive(document)
		}
	})

	b.Run("UpdateSerialize", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(source)))
		for range b.N {
			document, _, err := Parse(source, ParseOptions{})
			if err != nil {
				b.Fatal(err)
			}
			if err := document.UpdateText(0, "p0-n0", "Updated"); err != nil {
				b.Fatal(err)
			}
			output, err := document.Serialize()
			if err != nil {
				b.Fatal(err)
			}
			runtime.KeepAlive(document)
			runtime.KeepAlive(output)
		}
	})
}
