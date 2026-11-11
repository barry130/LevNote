package sourceview

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
	"levnote.local/native/core/text"
	"levnote.local/native/formats/sourceoutline"
	"levnote.local/native/ui/editor"
)

func iniConfig() sourceoutline.Config {
	return sourceoutline.Config{PluginID: "ini-outline", Schema: 2, Rules: []sourceoutline.Rule{{Prefix: "[", Suffix: "]", Level: 1}}}
}

func getSnapshot(t *testing.T, s editor.OutlineSource) editor.OutlineSnapshot {
	t.Helper()
	result, err := s.OutlineSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestSourceOutlineObservationalUnicodeAndDuplicateRangeIdentity(t *testing.T) {
	buffer := text.New("🙂before\r\n  [Same]\r\n[Same]\r\n")
	_ = buffer.SetSelection(5, 2)
	selection := buffer.Selection()
	s := NewOutline(editor.Host{Buffer: buffer, Changed: func() { t.Fatal("observational snapshot changed source") }}, iniConfig()).(*source)
	snapshot := getSnapshot(t, s)
	if len(snapshot.Items) != 2 || snapshot.Items[0].Offset != text.UTF16Len("🙂before\r\n") || snapshot.Items[0].AnchorID != "" || snapshot.Items[0].Level != 1 || snapshot.Items[0].ID == snapshot.Items[1].ID || buffer.Selection() != selection || buffer.CanUndo() {
		t.Fatal("invalid observation", snapshot)
	}
	old := snapshot.Items[1]
	_ = buffer.ReplaceUTF16(old.Offset, old.Offset, "[Same]\r\n")
	mapped, err := s.ResolveOutlineItem(old.ID)
	if err != nil || mapped.Offset != old.Offset+8 || mapped.ID == old.ID {
		t.Fatal("duplicate insertion retargeted", mapped, err)
	}
	s.Release()
	if s.loaded || s.items != nil || s.spans != nil || s.cachedBuffer != nil {
		t.Fatal("release kept cache")
	}
	second := NewOutline(editor.Host{Buffer: buffer}, iniConfig())
	for _, resolver := range []editor.OutlineSource{s, second} {
		if got, err := resolver.ResolveOutlineItem(old.ID); err != nil || got.ID != mapped.ID {
			t.Fatal("release/recreate changed identity", got, err)
		}
	}
	if !buffer.Undo() {
		t.Fatal("undo")
	}
	if got, err := s.ResolveOutlineItem(old.ID); err != nil || got.Offset != old.Offset {
		t.Fatal("outside insertion undo mapping", got, err)
	}
	if _, err := s.ResolveOutlineAnchor("same"); !errors.Is(err, editor.ErrOutlineUnsupported) {
		t.Fatal("invented anchor", err)
	}
}

func TestSourceOutlineRulesetOwnershipAndInvalidation(t *testing.T) {
	buffer := text.New("[A]\n[B]\n")
	config := iniConfig()
	s := NewOutline(editor.Host{Buffer: buffer}, config).(*source)
	old := getSnapshot(t, s).Items[1]
	config.Rules[0].Prefix = "changed"
	if got, err := s.ResolveOutlineItem(old.ID); err != nil || got.Text != "B" {
		t.Fatal("caller modified immutable rules", err)
	}
	for _, action := range []string{"rule", "order", "id", "buffer", "target", "expiry", "restore", "syntax"} {
		t.Run(action, func(t *testing.T) {
			b := text.New("[A]\n[B]\n")
			cfg := iniConfig()
			cfg.Rules = append(cfg.Rules, sourceoutline.Rule{Prefix: "#", Level: 2})
			before := NewOutline(editor.Host{Buffer: b}, cfg)
			old := getSnapshot(t, before).Items[1]
			switch action {
			case "rule":
				cfg.Rules[0].Level = 2
			case "order":
				cfg.Rules[0], cfg.Rules[1] = cfg.Rules[1], cfg.Rules[0]
			case "id":
				cfg.PluginID = "other-ini"
			case "buffer":
				b = text.New(b.String())
			case "target":
				_ = b.ReplaceUTF16(old.Offset+1, old.Offset+2, "X")
			case "expiry":
				for i := 0; i < 65; i++ {
					_ = b.ReplaceUTF16(0, 0, " ")
				}
			case "restore":
				b.Restore(b.Snapshot())
			case "syntax":
				_ = b.ReplaceUTF16(old.Offset, old.Offset, "prefix")
			}
			after := NewOutline(editor.Host{Buffer: b}, cfg)
			if _, err := after.ResolveOutlineItem(old.ID); !errors.Is(err, editor.ErrOutlineStale) {
				t.Fatal("reinterpreted stale identity", err)
			}
		})
	}
	for _, config := range []sourceoutline.Config{{}, {PluginID: "id", Schema: 1}, {PluginID: "id", Schema: 2, Rules: []sourceoutline.Rule{{Level: 1}}}} {
		bad := NewOutline(editor.Host{Buffer: buffer}, config)
		if _, err := bad.OutlineSnapshot(); err == nil {
			t.Fatal("bad config snapshot not rejected")
		}
		if _, err := bad.ResolveOutlineItem(old.ID); err == nil {
			t.Fatal("bad config resolve not rejected")
		}
	}
}

func TestBufferReaderAcrossFragmentedUnicodePiecesAndVersionChange(t *testing.T) {
	buffer := text.New(strings.Repeat("中🙂a", 2000))
	buffer.SetHistoryLimits(0, 0)
	for i := 0; i < 120; i++ {
		offset := i * 5
		if err := buffer.ReplaceUTF16(offset, offset, "\n"); err != nil {
			t.Fatal(err)
		}
	}
	r := &bufferReader{buffer: buffer, version: buffer.Version()}
	var out strings.Builder
	var chunk [7]byte
	for {
		n, err := r.Read(chunk[:])
		out.Write(chunk[:n])
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if out.String() != buffer.String() {
		t.Fatal("fragmented/surrogate source changed")
	}
	r = &bufferReader{buffer: buffer, version: buffer.Version()}
	_, _ = r.Read(chunk[:])
	_ = buffer.ReplaceUTF16(0, 0, "x")
	if _, err := r.Read(chunk[:]); !errors.Is(err, editor.ErrOutlineStale) {
		t.Fatal("reader ignored source change", err)
	}
}

func TestSourceOutlineBoundedLongLineCacheAndCopiedDescriptors(t *testing.T) {
	buffer := text.New("[" + strings.Repeat("中🙂", 300000) + "]\n[after]")
	s := NewOutline(editor.Host{Buffer: buffer}, iniConfig()).(*source)
	snapshot := getSnapshot(t, s)
	if !snapshot.Truncated || len(snapshot.Items) != 0 || s.readChunks > 1600 {
		t.Fatal("cut huge line emitted or unbounded read windows", len(snapshot.Items), s.readChunks)
	}
	t.Logf("2MiB cut huge-line reader windows=%d; returned items=%d partial=%v", s.readChunks, len(snapshot.Items), snapshot.Truncated)
	chunks := s.readChunks
	for i := 0; i < 5; i++ {
		getSnapshot(t, s)
		if s.readChunks != chunks {
			t.Fatal("cached frame reparsed")
		}
	}
	s.buffer = text.New("[" + strings.Repeat("中🙂", 10000) + "]\n[short]\n")
	snapshot = getSnapshot(t, s)
	if !snapshot.Truncated || len(snapshot.Items) != 2 || len(snapshot.Items[0].Text) > 512 || !utf8.ValidString(snapshot.Items[0].Text) {
		t.Fatal("long label not bounded", snapshot)
	}
	snapshot.Items[1].Text = "caller"
	if getSnapshot(t, s).Items[1].Text != "short" {
		t.Fatal("snapshot slice aliases provider")
	}
	s.Release()
	if s.items != nil || s.spans != nil || s.readChunks != 0 || s.cachedBuffer != nil {
		t.Fatal("release retained scan state")
	}
}

func TestSourceOutlineRangeCancelAndTerminalInsertion(t *testing.T) {
	buffer := text.New("[A]\n[B]\n")
	s := NewOutline(editor.Host{Buffer: buffer}, iniConfig())
	old := getSnapshot(t, s).Items[1]
	_ = buffer.BeginGroup()
	_ = buffer.ReplaceUTF16(0, 0, "🙂prefix\n")
	buffer.CancelGroup()
	if item, err := s.ResolveOutlineItem(old.ID); err != nil || item.Offset != old.Offset {
		t.Fatal("canceled outside edit lost target", item, err)
	}
	_ = buffer.ReplaceUTF16(buffer.LenUTF16(), buffer.LenUTF16(), "[C]\n")
	if item, err := s.ResolveOutlineItem(old.ID); err != nil || item.Offset != old.Offset {
		t.Fatal("insertion after terminal newline changed preceding target", item, err)
	}
	endless := text.New("[B]")
	last := NewOutline(editor.Host{Buffer: endless}, iniConfig())
	item := getSnapshot(t, last).Items[0]
	_ = endless.ReplaceUTF16(endless.LenUTF16(), endless.LenUTF16(), "suffix")
	if _, err := last.ResolveOutlineItem(item.ID); !errors.Is(err, editor.ErrOutlineStale) {
		t.Fatal("EOF line extension reused stale range", err)
	}
}

func TestSourceOutlineNativeComposeFlushRevalidation(t *testing.T) {
	buffer := text.New("plain\n[Same]\n[Same]\n")
	s := NewOutline(editor.Host{Buffer: buffer}, iniConfig())
	old := getSnapshot(t, s).Items[1]
	input := editor.New(buffer)
	input.Label = "源码"
	tester := ui.NewTester(input.View, 600, 300)
	input.Reveal(0, 0)
	tester.Frame()
	tester.Compose("[Same]\n", 7)
	if _, composing := input.MarkedRange(); !composing {
		t.Fatal("actual Compose did not establish pending native input")
	}
	if err := input.Flush(); err != nil {
		t.Fatal(err)
	}
	current, err := s.ResolveOutlineItem(old.ID)
	if err != nil || current.Offset != old.Offset+7 || current.Text != "Same" {
		t.Fatal("Compose insertion retargeted", current, err, buffer.String())
	}
	input.Reveal(current.Offset, current.Offset)
	tester.Frame()
	if buffer.Selection().Head != current.Offset {
		t.Fatal("source reveal did not use resolved offset")
	}
}

func sourceFixture(nodes int) string {
	var raw strings.Builder
	for i := 0; i < nodes; i++ {
		fmt.Fprintf(&raw, "[Section 中文🙂 %d]\r\n", i)
	}
	return raw.String()
}

func BenchmarkSourceOutlineProvider(b *testing.B) {
	for _, fragmented := range []bool{false, true} {
		buffer := text.New(sourceFixture(2000))
		buffer.SetHistoryLimits(0, 0)
		if fragmented {
			for i := 1; i <= 200; i++ {
				offset, _ := buffer.OffsetOfLine(i * 5)
				_ = buffer.ReplaceUTF16(offset, offset, " ")
			}
		}
		for _, cold := range []bool{true, false} {
			b.Run(fmt.Sprintf("fragmented=%v/cold=%v", fragmented, cold), func(b *testing.B) {
				s := NewOutline(editor.Host{Buffer: buffer}, iniConfig())
				s.OutlineSnapshot()
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if cold {
						s.Release()
					}
					s.OutlineSnapshot()
				}
			})
		}
	}
}
