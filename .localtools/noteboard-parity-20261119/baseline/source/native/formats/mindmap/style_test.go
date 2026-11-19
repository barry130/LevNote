package mindmap

import (
	"bytes"
	"errors"
	"testing"
)

const colorStyleFixture = `{
  "format":"levnote.mindmap","version":1,"layout":"logicalStructure",
  "futureTop":1.00,"root":{"id":"root","topic":"中心🙂","style":{"future":{"n":1.20},"color":" #1E40AF ","border":1e3},"children":[{"id":"child","topic":"子主题"}]}}`

func parseColorStyle(t *testing.T, source string) *Document {
	t.Helper()
	d, _, err := Parse([]byte(source), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestStylePropertyNoopOriginalBytesAndPresence(t *testing.T) {
	source := colorStyleFixture
	d := parseColorStyle(t, source)
	for _, operation := range []func() error{
		func() error { return d.SetStyleProperty("root", StyleTextColor, "#1e40af") },
		func() error { return d.ClearStyleProperty("root", StyleFillColor) },
	} {
		if err := operation(); err != nil {
			t.Fatal(err)
		}
		got, err := d.Serialize(SerializeOptions{})
		if err != nil || string(got) != source || d.dirty {
			t.Fatal("no-op changed source or dirty flag", err, string(got))
		}
	}
	for _, style := range []string{"", `,"style":{}`} {
		source := `{"format":"levnote.mindmap","version":1,"root":{"id":"r","topic":"root"` + style + `}}`
		d := parseColorStyle(t, source)
		if err := d.ClearStyleProperty("r", StyleTextColor); err != nil {
			t.Fatal(err)
		}
		got, _ := d.Serialize(SerializeOptions{})
		if string(got) != source {
			t.Fatal("clearing absent key changed optional presence")
		}
	}
}

func TestStylePropertyPreservesUnknownRawAndIndependentKeys(t *testing.T) {
	source := `{"format":"levnote.mindmap","version":1,"root":{"id":"r","topic":"root","style":{"future":{"n":1.20},"color":"navy","border":1e3},"children":[]},"top":2.00}`
	d := parseColorStyle(t, source)
	old := d.Root.Style
	if err := d.SetStyleProperty("r", StyleFillColor, "#dbeafe"); err != nil {
		t.Fatal(err)
	}
	if _, exists := old.Get("background-color"); exists {
		t.Fatal("edit mutated previous style object")
	}
	if err := d.ClearStyleProperty("r", StyleTextColor); err != nil {
		t.Fatal(err)
	}
	raw, err := d.Serialize(SerializeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	reparsed := parseColorStyle(t, string(raw))
	if _, exists := reparsed.Root.Style.Get("color"); exists {
		t.Fatal("reset retained removed key")
	}
	for _, want := range []struct{ key, raw string }{{"future", `{"n":1.20}`}, {"border", "1e3"}, {"background-color", `"#dbeafe"`}} {
		got, exists := reparsed.Root.Style.Get(want.key)
		if !exists || string(got) != want.raw {
			t.Fatalf("%s = %s", want.key, got)
		}
	}
	if !bytes.Contains(raw, []byte("2.00")) {
		t.Fatal("top-level number changed")
	}
	single := parseColorStyle(t, `{"format":"levnote.mindmap","version":1,"root":{"id":"r","topic":"root","style":{"border":"red"}}}`)
	if err := single.ClearStyleProperty("r", StyleBorderColor); err != nil || single.Root.Style != nil {
		t.Fatal("final reset retained empty shell", err)
	}
}

func TestStylePropertyValidationRollbackAndUnsupportedValues(t *testing.T) {
	d := parseColorStyle(t, `{"format":"levnote.mindmap","version":1,"root":{"id":"r","topic":"root","style":{"color":"url(custom)","future":1.20},"children":[{"id":"c","topic":"child"}]}}`)
	original, _ := d.Serialize(SerializeOptions{})
	for _, action := range []func() error{
		func() error { return d.SetStyleProperty("r", "font-size", "20") },
		func() error { return d.SetStyleProperty("r", StyleTextColor, "url(remote)") },
		func() error { return d.ClearStyleProperty("r", "font-size") },
	} {
		if err := action(); !errors.Is(err, ErrInvalidDocument) {
			t.Fatal("bad input accepted", err)
		}
	}
	if err := d.SetStyleProperty("missing", StyleTextColor, "red"); !errors.Is(err, ErrNodeNotFound) {
		t.Fatal(err)
	}
	old := d.Root.Style
	d.Root.Children[0].ID = "r"
	if err := d.SetStyleProperty("r", StyleTextColor, "red"); err == nil || d.Root.Style != old || d.dirty {
		t.Fatal("failed edit did not roll back", err)
	}
	if err := d.ClearStyleProperty("r", StyleTextColor); err == nil || d.Root.Style != old || d.dirty {
		t.Fatal("failed reset did not roll back", err)
	}
	d.Root.Children[0].ID = "c"
	got, _ := d.Serialize(SerializeOptions{})
	if !bytes.Equal(got, original) {
		t.Fatal("failed operation changed original bytes")
	}
}

func TestStylePropertyEquivalentHexAndNamedNoops(t *testing.T) {
	for _, pair := range [][2]string{{"#ABC", "#aabbcc"}, {"#aabbccff", "#abc"}, {"red", "#ff0000"}, {"#1234", "#11223344"}, {"transparent", "#0000"}} {
		source := `{"format":"levnote.mindmap","version":1,"root":{"id":"r","topic":"root","style":{"color":"` + pair[0] + `"}}}`
		d := parseColorStyle(t, source)
		if err := d.SetStyleProperty("r", StyleTextColor, pair[1]); err != nil {
			t.Fatal(err)
		}
		data, _ := d.Serialize(SerializeOptions{})
		if string(data) != source || d.dirty {
			t.Fatal("equivalent color rewrote original literal", pair)
		}
	}
}

func TestStyleTextFallbackOpaqueContrastAndExplicitInk(t *testing.T) {
	for _, tc := range []struct {
		style, want string
		ok          bool
	}{
		{`{}`, "", false},
		{`{"background-color":"#fef3c7"}`, "#17211e", true},
		{`{"background-color":"#000"}`, "#f8fafc", true},
		{`{"background-color":"navy"}`, "#f8fafc", true},
		{`{"background-color":"#ffffff80"}`, "", false},
		{`{"background-color":"transparent"}`, "", false},
		{`{"background-color":"var(custom)"}`, "", false},
		{`{"background-color":"#fff","color":"white"}`, "", false},
		{`{"background-color":"#fff","color":17}`, "", false},
	} {
		source := `{"format":"levnote.mindmap","version":1,"root":{"id":"r","topic":"root","style":` + tc.style + `}}`
		d := parseColorStyle(t, source)
		if got, ok := StyleTextFallback(d.Root); got != tc.want || ok != tc.ok {
			t.Fatal(tc.style, got, ok)
		}
		data, _ := d.Serialize(SerializeOptions{})
		if string(data) != source || d.dirty {
			t.Fatal("presentation mutated content")
		}
	}
}
