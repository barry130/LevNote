package main

import (
	"bytes"
	"fmt"
	"testing"

	"levnote.local/native/core/engineipc"
	"levnote.local/native/core/enginescene"
)

func TestPreviewWorkerStyleTextFallbackAndExplicitInk(t *testing.T) {
	for _, tc := range []struct {
		name, style, fill, stroke, ink string
	}{
		{"unstyled", "", "", "", ""},
		{"empty-style", ",\"style\":{}", "", "", ""},
		{"pale-fill", ",\"style\":{\"background-color\":\"#fef3c7\",\"future\":1.20}", "#fef3c7ff", "", "#17211eff"},
		{"dark-fill", ",\"style\":{\"background-color\":\"black\"}", "#000000ff", "", "#f8fafcff"},
		{"opaque-alpha", ",\"style\":{\"background-color\":\"#ffff\"}", "#ffffffff", "", "#17211eff"},
		{"explicit-ink", ",\"style\":{\"background-color\":\"#fff\",\"color\":\"#1e40af\",\"border\":\"#166534\"}", "#ffffffff", "#166534ff", "#1e40afff"},
		{"transparent-ink", ",\"style\":{\"background-color\":\"#fff\",\"color\":\"transparent\"}", "#ffffffff", "", "#00000000"},
		{"invalid-ink", ",\"style\":{\"background-color\":\"#fff\",\"color\":7}", "#ffffffff", "", ""},
		{"custom-ink", ",\"style\":{\"background-color\":\"#fff\",\"color\":\"var(custom)\"}", "#ffffffff", "", ""},
		{"transparent-fill", ",\"style\":{\"background-color\":\"none\"}", "#00000000", "", ""},
		{"translucent-fill", ",\"style\":{\"background-color\":\"#ffffff80\"}", "#ffffff80", "", ""},
		{"custom-fill", ",\"style\":{\"background-color\":\"var(custom)\"}", "", "", ""},
		{"border-only", ",\"style\":{\"border\":\"#166534\"}", "", "#166534ff", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte(fmt.Sprintf(`{"format":"levnote.mindmap","version":1,"root":{"id":"r","topic":"Root"%s}}`, tc.style))
			before := bytes.Clone(source)
			meta, payload := callPreviewWorker(t, source)
			if meta.Status != engineipc.StatusOK || meta.Error != nil {
				t.Fatalf("worker status=%+v", meta)
			}
			scene, err := enginescene.Decode(payload)
			if err != nil || len(scene.Nodes) != 1 {
				t.Fatalf("scene=%+v err=%v", scene, err)
			}
			node := scene.Nodes[0]
			if node.Fill != tc.fill || node.Stroke != tc.stroke || node.Text != tc.ink {
				t.Fatalf("paint=(%s,%s,%s) want=(%s,%s,%s)", node.Fill, node.Stroke, node.Text, tc.fill, tc.stroke, tc.ink)
			}
			if !bytes.Equal(source, before) {
				t.Fatal("readonly paint projection changed original source bytes")
			}
		})
	}
}
