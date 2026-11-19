//go:build windows

package engineruntime

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"levnote.local/native/core/engineipc"
)

func TestRuntimeDrawioPageIsCopiedBeforeAdmissionHook(t *testing.T) {
	dir := t.TempDir()
	makeBundleFor(t, dir, engineipc.DrawioSVG)
	r := newRunner(dir)
	r.timeout = 5 * time.Second
	r.waitDelay = 100 * time.Millisecond
	page := &engineipc.RequestPage{Index: 1, ID: "second"}
	r.hooks = &testHooks{
		workerArgs: []string{"drawio-valid"},
		afterAdmission: func() {
			page.Index = 0
			page.ID = "caller-mutated"
		},
	}
	result, err := r.Export(context.Background(), engineipc.DrawioSVG, Source{Size: 13, Data: strings.NewReader("drawio-source"), Page: page})
	if err != nil || string(result.Payload) != `<svg xmlns="http://www.w3.org/2000/svg" data-page="1" data-id="second"><text>drawio-source</text></svg>` ||
		result.Engine.ID != engineipc.DrawioSVG {
		t.Fatalf("admission-owned page result=%+v err=%v", result, err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestRuntimeInvalidDrawioOptionsRefuseBeforeHooksSourceAndPackageLookup(t *testing.T) {
	for name, test := range map[string]struct {
		engine engineipc.EngineID
		page   *engineipc.RequestPage
	}{
		"old-engine-page": {engineipc.MarkdownHTML, &engineipc.RequestPage{}},
		"missing":         {engineipc.DrawioSVG, nil},
		"negative":        {engineipc.DrawioSVG, &engineipc.RequestPage{Index: -1}},
		"over-index":      {engineipc.DrawioSVG, &engineipc.RequestPage{Index: engineipc.PageIndexMax + 1}},
		"over-id":         {engineipc.DrawioSVG, &engineipc.RequestPage{ID: strings.Repeat("x", engineipc.PageIDMaxBytes+1)}},
		"invalid-utf8":    {engineipc.DrawioSVG, &engineipc.RequestPage{ID: string([]byte{0xff})}},
	} {
		t.Run(name, func(t *testing.T) {
			r := newRunner(t.TempDir())
			r.manager = &PackageManager{gate: make(chan struct{}, 1), loadErr: errors.New("poisoned selection must not be consulted")}
			hookCalls, starts := 0, 0
			r.hooks = &testHooks{
				afterAdmission: func() { hookCalls++ },
				afterStart:     func(*os.Process) { starts++ },
			}
			source := &untouchedSource{}
			result, err := r.Export(context.Background(), test.engine, Source{Size: 1, Data: source, Page: test.page})
			requireFailure(t, result, err, FailureProtocol, false)
			if source.calls != 0 || hookCalls != 0 || starts != 0 {
				t.Fatalf("invalid options reached source/hook/launch: source=%d hooks=%d starts=%d", source.calls, hookCalls, starts)
			}
			if closeErr := r.Close(); closeErr != nil {
				t.Fatal(closeErr)
			}
		})
	}
}
