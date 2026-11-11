package main

import (
	"bytes"
	"reflect"
	"testing"

	"levnote.local/native/core/engineipc"
)

func TestWorkerImageReferenceWarningMetadataAndLiteralHTML(t *testing.T) {
	const accepted = "![<&>](dir/a%20b.png)"
	const rejected = `![bad](a.png "title")`
	source := []byte("\xef\xbb\xbf" + accepted + "\r\n\r\n" + rejected + "\r\n")
	unchanged := append([]byte(nil), source...)

	meta, html := callWorker(t, source)
	wantHTML := []byte("<!doctype html>\n<meta charset=\"utf-8\">\n<body>\n" +
		"<p>\xef\xbb\xbf![&lt;&amp;&gt;](dir/a%20b.png)</p>\n" +
		"<p>![bad](a.png &#34;title&#34;)</p>\n" +
		"</body>\n")
	acceptedStart := bytes.Index(source, []byte(accepted))
	rejectedStart := bytes.Index(source, []byte(rejected))
	wantWarnings := []engineipc.Warning{
		{
			Code: "image-reference", Message: "image syntax is preserved as an explicit picture reference; inline images are not rendered",
			Start: uint32(acceptedStart), End: uint32(acceptedStart + len(accepted)),
		},
		{
			Code: "unsupported-image", Message: "inline syntax is not fully supported; source text remains visible",
			Start: uint32(rejectedStart), End: uint32(rejectedStart + len(rejected)),
		},
	}
	if meta.Status != engineipc.StatusOK || meta.Error != nil ||
		!bytes.Equal(html, wantHTML) ||
		!reflect.DeepEqual(meta.Warnings, wantWarnings) {
		t.Fatalf("status=%s error=%+v html=%q warnings=%+v", meta.Status, meta.Error, html, meta.Warnings)
	}
	if bytes.Contains(html, []byte("<img")) || bytes.Contains(html, []byte(" src=")) {
		t.Fatalf("worker emitted an HTML image resource: %q", html)
	}
	if !bytes.Equal(source, unchanged) {
		t.Fatalf("worker changed source bytes:\n got %q\nwant %q", source, unchanged)
	}
}
