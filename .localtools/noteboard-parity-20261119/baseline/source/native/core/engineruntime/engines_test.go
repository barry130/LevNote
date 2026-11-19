//go:build windows

package engineruntime

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"levnote.local/native/core/engineipc"
)

func TestRuntimeMindmapIdentityPayloadAndCrossResponse(t *testing.T) {
	for _, mode := range []string{"valid", "wrong-engine", "wrong-format"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			makeBundleFor(t, dir, engineipc.MindmapSVG)
			r := newRunner(dir)
			defer r.Close()
			starts, waits := 0, 0
			r.hooks = &testHooks{workerArgs: []string{"mindmap-" + mode},
				afterStart: func(p *os.Process) { starts++; t.Logf("owned Mindmap helper pid=%d", p.Pid) },
				afterWait:  func() { waits++ },
			}
			result, err := r.Export(context.Background(), engineipc.MindmapSVG, sourceOf("字形\ufeff<&\r\n✅"))
			if mode != "valid" {
				requireFailure(t, result, err, FailureProtocol, true)
			} else if err != nil || string(result.Payload) != "<svg xmlns=\"http://www.w3.org/2000/svg\"><text>字形\ufeff&lt;&amp;\r\n✅</text></svg>" {
				t.Fatalf("independent Mindmap helper payload: %q %v", result.Payload, err)
			}
			if result.Engine.ID != engineipc.MindmapSVG || starts != 1 || waits != 1 {
				t.Fatal("operation lost identity or did not join", result.Engine, starts, waits)
			}
		})
	}
}

type untouchedSource struct{ calls int }

func (s *untouchedSource) WriteTo(io.Writer) (int64, error) {
	s.calls++
	return 0, nil
}

func TestRuntimeOperationLimitsBeforeSourceOrLaunch(t *testing.T) {
	for _, tc := range []struct {
		id   engineipc.EngineID
		size int64
		kind FailureKind
	}{
		{"levnote.foreign", 1, FailureUnavailable},
		{engineipc.MarkdownHTML, (8 << 20) + 1, FailureSourceLimit},
		{engineipc.MindmapSVG, -1, FailureSourceLimit},
		{engineipc.MindmapSVG, (16 << 20) + 1, FailureSourceLimit},
		// Allowed Mindmap sizes advance to bundle verification, unlike Markdown.
		// The missing fixed bundle refuses before reading any fake source bytes.
		{engineipc.MindmapSVG, (8 << 20) + 1, FailureUnavailable},
		{engineipc.MindmapSVG, 16 << 20, FailureUnavailable},
	} {
		source := &untouchedSource{}
		r := newRunner(t.TempDir())
		starts := 0
		r.hooks = &testHooks{afterStart: func(*os.Process) { starts++ }}
		result, err := r.Export(context.Background(), tc.id, Source{Size: tc.size, Data: source})
		requireFailure(t, result, err, tc.kind, false)
		if source.calls != 0 || starts != 0 {
			t.Fatal("refusal wrote source or launched", tc.id, tc.size, source.calls, starts)
		}
		if err := r.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRuntimeMindmapFixedManifestPairAndPath(t *testing.T) {
	for _, changed := range []string{"engine", "format", "filename", "goos", "path", "coherent-markdown"} {
		t.Run(changed, func(t *testing.T) {
			dir := t.TempDir()
			exe := makeBundleFor(t, dir, engineipc.MindmapSVG)
			path := filepath.Join(filepath.Dir(exe), "manifest.json")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var m bundleManifest
			if err := json.Unmarshal(data, &m); err != nil {
				t.Fatal(err)
			}
			switch changed {
			case "engine":
				m.Engine = engineipc.MarkdownHTML
			case "format":
				m.Format = "html"
			case "filename":
				m.Filename = "levnote-markdown-export.exe"
			case "goos":
				m.GOOS = "linux"
			case "path":
				m.Filename = "../markdown-export/levnote-markdown-export.exe"
			case "coherent-markdown":
				m.Engine, m.Format, m.Filename = engineipc.MarkdownHTML, "html", "levnote-markdown-export.exe"
			}
			data, _ = json.Marshal(m)
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			r := newRunner(dir)
			defer r.Close()
			starts := 0
			r.hooks = &testHooks{afterStart: func(*os.Process) { starts++ }}
			source := &untouchedSource{}
			result, err := r.Export(context.Background(), engineipc.MindmapSVG, Source{Size: 1, Data: source})
			requireFailure(t, result, err, FailureIntegrity, false)
			if starts != 0 || source.calls != 0 {
				t.Fatal("foreign manifest redirected request")
			}
		})
	}
	// A valid Markdown installation is never discovered/adopted as Mindmap.
	dir, _ := makeBundle(t)
	r := newRunner(dir)
	defer r.Close()
	result, err := r.Export(context.Background(), engineipc.MindmapSVG, sourceOf("x"))
	requireFailure(t, result, err, FailureUnavailable, false)
	if strings.Contains(err.Error(), "HTML") {
		t.Fatal("missing Mindmap bundle falsely reported HTML")
	}
}

func TestRuntimeSingleAdmissionAcrossEnginesAndJoinedReuse(t *testing.T) {
	dir, _ := makeBundle(t)
	makeBundleFor(t, dir, engineipc.MindmapSVG)
	r := newRunner(dir)
	defer r.Close()
	entered, release := make(chan struct{}), make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	starts, waits := 0, 0
	r.hooks = &testHooks{afterVerify: func(*verifiedBundle) { close(entered); <-release },
		afterStart: func(*os.Process) { starts++ }, afterWait: func() { waits++ }}
	type outcome struct {
		result Result
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		value, err := r.Export(context.Background(), engineipc.MarkdownHTML, sourceOf("first"))
		done <- outcome{value, err}
	}()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("first verification never reached")
	}
	source := &untouchedSource{}
	result, err := r.Export(context.Background(), engineipc.MindmapSVG, Source{Size: 1, Data: source})
	requireFailure(t, result, err, FailureBusy, false)
	if source.calls != 0 {
		t.Fatal("busy second operation read source")
	}
	close(release)
	select {
	case first := <-done:
		if first.err != nil || string(first.result.Payload) != "<p>first</p>" {
			t.Fatal("first operation failed", first.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("first export did not join")
	}
	r.hooks.afterVerify = nil
	r.hooks.workerArgs = []string{"mindmap-valid"}
	result, err = r.Export(context.Background(), engineipc.MindmapSVG, sourceOf("next"))
	if err != nil || string(result.Payload) != `<svg xmlns="http://www.w3.org/2000/svg"><text>next</text></svg>` || result.Engine.ID != engineipc.MindmapSVG || starts != 2 || waits != 2 {
		t.Fatal("cross-engine reuse did not wait/join both", result.Engine, starts, waits, err)
	}
}
