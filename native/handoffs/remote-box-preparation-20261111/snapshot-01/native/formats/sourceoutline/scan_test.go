package sourceoutline

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"
)

var iniRules = []Rule{{Prefix: "[", Suffix: "]", Level: 1}, {Prefix: "## ", Level: 2}, {Prefix: "#", Level: 3}}

type chunkReader struct {
	data        string
	step        int
	read, calls int
	eofWithData bool
}

func (r *chunkReader) Read(target []byte) (int, error) {
	r.calls++
	if r.data == "" {
		return 0, io.EOF
	}
	n := min(len(target), len(r.data), r.step)
	copy(target, r.data[:n])
	r.data = r.data[n:]
	r.read += n
	if r.data == "" && r.eofWithData {
		return n, io.EOF
	}
	return n, nil
}

func unitLen(value string) int {
	n := 0
	for _, char := range value {
		n++
		if char > 0xffff {
			n++
		}
	}
	return n
}

func TestScanUnicodeOffsetsCRAndFirstRuleAcrossFragmentation(t *testing.T) {
	raw := "🙂前文\r\n  [网络🙂] \t\r\n## title\n# a\rb\r\r\n\t[末尾]\r"
	want := []Entry{
		{Start: unitLen("🙂前文\r\n"), End: unitLen("🙂前文\r\n  [网络🙂] \t\r\n"), Level: 1, Text: "网络🙂"},
		{Start: unitLen("🙂前文\r\n  [网络🙂] \t\r\n"), End: unitLen("🙂前文\r\n  [网络🙂] \t\r\n## title\n"), Level: 2, Text: "title"},
		{Start: unitLen("🙂前文\r\n  [网络🙂] \t\r\n## title\n"), End: unitLen("🙂前文\r\n  [网络🙂] \t\r\n## title\n# a\rb\r\r\n"), Level: 3, Text: "a\rb\r"},
		{Start: unitLen("🙂前文\r\n  [网络🙂] \t\r\n## title\n# a\rb\r\r\n"), End: unitLen(raw), Level: 1, Text: "末尾"},
	}
	for _, step := range []int{1, 2, 3, 7, 4096} {
		for _, eof := range []bool{false, true} {
			reader := &chunkReader{data: raw, step: step, eofWithData: eof}
			got, err := Scan(reader, iniRules)
			if err != nil || got.Truncated || got.Bytes != len(raw) || got.Lines != 5 || !reflect.DeepEqual(got.Entries, want) {
				t.Fatalf("step%d eof%v: %+v err%v", step, eof, got, err)
			}
		}
	}
}

func TestScanEmptyFirstMatchAndExactTerminalCR(t *testing.T) {
	rules := []Rule{{Prefix: "[", Suffix: "]", Level: 1}, {Prefix: "[", Level: 2}, {Prefix: "#", Level: 3}}
	got, err := Scan(strings.NewReader("[]\n[ ]\n[x]\n# a\r \n# b\r\r\n# c\r"), rules)
	if err != nil || len(got.Entries) != 4 || got.Entries[0].Text != "x" || got.Entries[1].Text != "a\r" || got.Entries[2].Text != "b\r" || got.Entries[3].Text != "c" {
		t.Fatalf("first-marker/terminal-CR semantics: %+v %v", got, err)
	}
}

func referenceScan(raw string, rules []Rule) []Entry {
	var result []Entry
	offset := 0
	for len(raw) > 0 {
		n := strings.IndexByte(raw, '\n')
		if n < 0 {
			n = len(raw)
		} else {
			n++
		}
		physical := raw[:n]
		value := strings.TrimSuffix(strings.TrimSuffix(physical, "\n"), "\r")
		value = strings.TrimLeft(value, " \t")
		for _, rule := range rules {
			if !strings.HasPrefix(value, rule.Prefix) {
				continue
			}
			body := strings.TrimRight(value[len(rule.Prefix):], " \t")
			if !strings.HasSuffix(body, rule.Suffix) {
				continue
			}
			label := strings.Trim(body[:len(body)-len(rule.Suffix)], " \t")
			if label != "" {
				result = append(result, Entry{Start: offset, End: offset + unitLen(physical), Text: label, Level: rule.Level})
			}
			break
		}
		offset += unitLen(physical)
		raw = raw[n:]
	}
	return result
}

func TestScanMatchesSimpleReferenceAcrossLiteralSuffixAndLongWhitespace(t *testing.T) {
	rules := []Rule{{Prefix: "[", Suffix: " ]", Level: 1}, {Prefix: "[", Suffix: "]", Level: 2}, {Prefix: "前", Suffix: "中🙂", Level: 3}, {Prefix: "#", Level: 4}}
	rng := rand.New(rand.NewSource(81))
	parts := []string{"a", " ", "\t", "🙂", "中", "\r", "-", "]"}
	for i := 0; i < 500; i++ {
		var raw strings.Builder
		raw.WriteString(strings.Repeat(" \t", rng.Intn(40)))
		rule := rules[rng.Intn(len(rules))]
		raw.WriteString(rule.Prefix)
		for j := 0; j < rng.Intn(80); j++ {
			raw.WriteString(parts[rng.Intn(len(parts))])
		}
		raw.WriteString(strings.Repeat(" \t", rng.Intn(80)))
		if rng.Intn(4) != 0 {
			raw.WriteString(rule.Suffix)
		}
		raw.WriteString(strings.Repeat(" \t", rng.Intn(40)))
		if rng.Intn(2) == 0 {
			raw.WriteString("\r\n")
		}
		value := raw.String()
		got, err := Scan(&chunkReader{data: value, step: 1 + rng.Intn(9)}, rules)
		want := referenceScan(value, rules)
		if err != nil || !reflect.DeepEqual(got.Entries, want) {
			t.Fatalf("case%d input%q got%+v want%+v err%v", i, value, got.Entries, want, err)
		}
	}
}

func TestScanBudgetsAndReadAhead(t *testing.T) {
	for _, test := range []struct {
		name, raw    string
		items, lines int
		partial      bool
	}{
		{"exact-byte-line", "#" + strings.Repeat(" ", MaxScanBytes-3) + "x\n", 1, 1, false},
		{"byte-cut-line", "#" + strings.Repeat("x", MaxScanBytes+100), 0, 0, true},
		{"exact-lines", strings.Repeat("\n", MaxScanLines), 0, MaxScanLines, false},
		{"line-cut", strings.Repeat("\n", MaxScanLines) + "# omitted\n", 0, MaxScanLines, true},
		{"exact-items", strings.Repeat("# x\n", MaxItems), MaxItems, MaxItems, false},
		{"item-cut", strings.Repeat("# x\n", MaxItems+1), MaxItems, MaxItems, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := &chunkReader{data: test.raw, step: 4096}
			result, err := Scan(r, []Rule{{Prefix: "#", Level: 1}})
			if err != nil || result.Truncated != test.partial || len(result.Entries) != test.items || result.Lines != test.lines || result.Bytes > MaxScanBytes || r.read > MaxScanBytes+1 {
				t.Fatalf("result%+v read%d err%v", result, r.read, err)
			}
			if r.calls > (MaxScanBytes+4095)/4096+2 {
				t.Fatal("unbounded reads", r.calls)
			}
		})
	}
	// Item limit occurs inside the already-read chunk; there must be no probe.
	r := &chunkReader{data: strings.Repeat("#x\n", MaxItems+50), step: 4096}
	result, err := Scan(r, []Rule{{Prefix: "#", Level: 1}})
	if err != nil || !result.Truncated || r.calls != 2 {
		t.Fatal("unnecessary read after prefetched omission", r.calls, err)
	}
}

func TestScanHugeCompletedLabelIsBoundedAndUTF8(t *testing.T) {
	raw := "[" + strings.Repeat("中🙂", 200000) + strings.Repeat(" \t", 10000) + "]\n"
	got, err := Scan(strings.NewReader(raw), iniRules)
	if err != nil || len(got.Entries) != 1 || !got.Truncated || len(got.Entries[0].Text) > MaxLabelBytes || !utf8.ValidString(got.Entries[0].Text) || !strings.HasSuffix(got.Entries[0].Text, "…") {
		t.Fatal("huge bounded label", len(got.Entries), err)
	}
}

func TestScanExactEOFAndCrossBudgetRune(t *testing.T) {
	rules := []Rule{{Prefix: "#", Level: 1}}
	for _, raw := range []string{"", "# final", "# final\r", "# final\n"} {
		got, err := Scan(strings.NewReader(raw), rules)
		if err != nil || got.Truncated || !reflect.DeepEqual(got.Entries, referenceScan(raw, rules)) {
			t.Fatal("exact EOF semantics", got, err)
		}
	}
	exact := "#" + strings.Repeat(" ", MaxScanBytes-2) + "x"
	got, err := Scan(strings.NewReader(exact), rules)
	if err != nil || got.Truncated || got.Bytes != MaxScanBytes || got.Lines != 1 || len(got.Entries) != 1 || got.Entries[0].Text != "x" || got.Entries[0].End != MaxScanBytes {
		t.Fatal("exact2MiB EOF final line was not accepted", got, err)
	}
	// A rune straddles the processing budget. The prefix is valid but the
	// physical line is incomplete, so truncate without reporting invalid UTF8.
	raw := "#" + strings.Repeat("x", MaxScanBytes-2) + "🙂\n"
	got, err = Scan(strings.NewReader(raw), rules)
	if err != nil || !got.Truncated || len(got.Entries) != 0 || got.Bytes != MaxScanBytes {
		t.Fatal("budget-cut rune/line emitted or misreported", got, err)
	}
	// Empty-label fallback is prohibited even if a later rule could name it.
	got, err = Scan(strings.NewReader("## \n"), []Rule{{Prefix: "## ", Level: 1}, {Prefix: "#", Level: 2}})
	if err != nil || len(got.Entries) != 0 || got.Truncated {
		t.Fatal("empty first full marker match fell through", got, err)
	}
}

func TestScanSixteenByteLiteralSuffixAfterLongWhitespace(t *testing.T) {
	rule := Rule{Prefix: "@", Suffix: " 12345678901234]", Level: 4}
	if len(rule.Suffix) != 16 {
		t.Fatal("fixture suffix length")
	}
	for _, label := range []string{"中文🙂", strings.Repeat("x", 600), ""} {
		raw := " \t@" + label + strings.Repeat(" \t", 2000) + rule.Suffix + " \t\r\n"
		got, err := Scan(&chunkReader{data: raw, step: 3}, []Rule{rule})
		if err != nil {
			t.Fatal(err)
		}
		if label == "" {
			if len(got.Entries) != 0 {
				t.Fatal("empty label emitted")
			}
			continue
		}
		if len(got.Entries) != 1 || got.Entries[0].Level != 4 {
			t.Fatal(got)
		}
		if len(label) <= MaxLabelBytes {
			if want := referenceScan(raw, []Rule{rule}); !reflect.DeepEqual(got.Entries, want) {
				t.Fatal(got.Entries, want)
			}
		} else if !got.Truncated || !strings.HasSuffix(got.Entries[0].Text, "…") {
			t.Fatal("long suffix/label clipping", got)
		}
	}
}

func TestScanZeroProgressResetsAfterDataAndPropagatesLimitError(t *testing.T) {
	remaining, zeroes := "# ok\n", 0
	result, err := Scan(functionReader(func(target []byte) (int, error) {
		if remaining == "" {
			return 0, io.EOF
		}
		if zeroes < 7 {
			zeroes++
			return 0, nil
		}
		zeroes = 0
		target[0], remaining = remaining[0], remaining[1:]
		return 1, nil
	}), []Rule{{Prefix: "#", Level: 1}})
	if err != nil || len(result.Entries) != 1 {
		t.Fatal("short successful progress rejected", err)
	}
	remaining = strings.Repeat("\n", MaxScanLines)
	want := errors.New("late read failure")
	_, err = Scan(functionReader(func(target []byte) (int, error) {
		n := copy(target, remaining)
		remaining = remaining[n:]
		if remaining == "" {
			return n, want
		}
		return n, nil
	}), []Rule{{Prefix: "#", Level: 1}})
	if !errors.Is(err, want) {
		t.Fatal("reader error hidden by limit", err)
	}
}

type functionReader func([]byte) (int, error)

func (r functionReader) Read(p []byte) (int, error) { return r(p) }

func TestScanNoProgressReaderErrorsAndInvalidUTF8(t *testing.T) {
	rules := []Rule{{Prefix: "#", Level: 1}}
	calls := 0
	_, err := Scan(functionReader(func([]byte) (int, error) { calls++; return 0, nil }), rules)
	if !errors.Is(err, io.ErrNoProgress) || calls != 8 {
		t.Fatal("no-progress not bounded", calls, err)
	}
	for _, raw := range []string{"#\xff\n", "#中\xf0", "#\xe4\n"} {
		if _, err := Scan(&chunkReader{data: raw, step: 1}, rules); !errors.Is(err, ErrUTF8) {
			t.Fatal("invalid UTF8 accepted", err)
		}
	}
	want := errors.New("read failure")
	_, err = Scan(functionReader(func(p []byte) (int, error) { return copy(p, "# valid\n# incomplete"), want }), rules)
	if !errors.Is(err, want) {
		t.Fatal("read error hidden", err)
	}
	for _, n := range []int{-1, 5000} {
		if _, err := Scan(functionReader(func([]byte) (int, error) { return n, nil }), rules); err == nil {
			t.Fatal("bad reader count accepted")
		}
	}
	for _, rules := range [][]Rule{
		{{Prefix: "", Level: 1}}, {{Prefix: "\t", Level: 1}}, {{Prefix: "x", Level: 0}}, {{Prefix: "x", Level: 1}, {Prefix: "x", Level: 2}}, {{Prefix: "x", Suffix: "\xff", Level: 1}},
	} {
		if _, err := Scan(strings.NewReader("x"), rules); err == nil {
			t.Fatal("invalid rule accepted", rules)
		}
	}
}

func BenchmarkScanHugeLine(b *testing.B) {
	for _, matched := range []bool{false, true} {
		b.Run(fmt.Sprint("matched=", matched), func(b *testing.B) {
			prefix := "!"
			if matched {
				prefix = "#"
			}
			raw := prefix + strings.Repeat("x", MaxScanBytes-2) + "\n"
			b.ReportAllocs()
			b.SetBytes(int64(len(raw)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				Scan(strings.NewReader(raw), []Rule{{Prefix: "#", Level: 1}})
			}
		})
	}
}
